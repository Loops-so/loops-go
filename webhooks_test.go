package loops

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestParseWebhook(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		check func(t *testing.T, evt WebhookEvent)
	}{
		{
			name: "contact.created with custom properties",
			body: `{"eventName":"contact.created","eventTime":1734425918,"webhookSchemaVersion":"1.0.0",
				"contactIdentity":{"id":"c1","email":"a@b.com","userId":null},
				"contact":{"id":"c1","email":"a@b.com","firstName":"A","lastName":null,"source":"api",
					"subscribed":true,"userGroup":"","userId":null,"mailingLists":{"ml_1":true},
					"optInStatus":"accepted","plan":"pro"}}`,
			check: func(t *testing.T, evt WebhookEvent) {
				e, ok := evt.(*WebhookContactCreatedPayload)
				if !ok {
					t.Fatalf("type = %T, want *WebhookContactCreatedPayload", evt)
				}
				if e.Type() != WebhookEventContactCreated {
					t.Errorf("Type() = %q", e.Type())
				}
				if e.EventTime != 1734425918 {
					t.Errorf("EventTime = %d", e.EventTime)
				}
				if e.ContactIdentity.Email != "a@b.com" {
					t.Errorf("ContactIdentity = %+v", e.ContactIdentity)
				}
				if e.Contact.Email != "a@b.com" || e.Contact.MailingLists["ml_1"] != true {
					t.Errorf("Contact = %+v", e.Contact)
				}
				if e.Contact.Custom["plan"] != "pro" {
					t.Errorf("Contact.Custom = %+v, want plan=pro", e.Contact.Custom)
				}
			},
		},
		{
			name: "email.delivered metric",
			body: `{"eventName":"email.delivered","eventTime":2,"webhookSchemaVersion":"1.0.0",
				"contactIdentity":{"id":"c1","email":"a@b.com","userId":null},
				"email":{"id":"e1","emailMessageId":"em1","subject":"Hi"},
				"sourceType":"campaign","campaignId":"cmp1"}`,
			check: func(t *testing.T, evt WebhookEvent) {
				e, ok := evt.(*WebhookEmailDeliveredPayload)
				if !ok {
					t.Fatalf("type = %T, want *WebhookEmailDeliveredPayload", evt)
				}
				if e.Type() != WebhookEventEmailDelivered {
					t.Errorf("Type() = %q", e.Type())
				}
				if e.SourceType != WebhookSourceTypeCampaign || e.CampaignID != "cmp1" {
					t.Errorf("source = %q campaign = %q", e.SourceType, e.CampaignID)
				}
				if e.Email.Subject != "Hi" {
					t.Errorf("Email = %+v", e.Email)
				}
			},
		},
		{
			name: "campaign.email.sent",
			body: `{"eventName":"campaign.email.sent","eventTime":3,"webhookSchemaVersion":"1.0.0",
				"contactIdentity":{"id":"c1","email":"a@b.com","userId":null},
				"campaignId":"cmp1","campaignName":"Launch",
				"email":{"id":"e1","emailMessageId":"em1","subject":"Hi"},
				"mailingLists":[{"id":"ml1","name":"News","description":null,"isPublic":true}]}`,
			check: func(t *testing.T, evt WebhookEvent) {
				e, ok := evt.(*WebhookCampaignEmailSentPayload)
				if !ok {
					t.Fatalf("type = %T, want *WebhookCampaignEmailSentPayload", evt)
				}
				if e.CampaignName != "Launch" {
					t.Errorf("CampaignName = %q", e.CampaignName)
				}
				if len(e.MailingLists) != 1 || e.MailingLists[0].Name != "News" {
					t.Errorf("MailingLists = %+v", e.MailingLists)
				}
			},
		},
		{
			name: "testing.testEvent",
			body: `{"eventName":"testing.testEvent","eventTime":4,"message":"test","webhookSchemaVersion":"1.0.0"}`,
			check: func(t *testing.T, evt WebhookEvent) {
				e, ok := evt.(*WebhookTestingTestEventPayload)
				if !ok {
					t.Fatalf("type = %T, want *WebhookTestingTestEventPayload", evt)
				}
				if e.Type() != WebhookEventTestingTestEvent || e.Message != "test" {
					t.Errorf("evt = %+v", e)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt, err := ParseWebhook([]byte(tt.body))
			if err != nil {
				t.Fatalf("ParseWebhook: %v", err)
			}
			tt.check(t, evt)
		})
	}
}

func TestParseWebhook_Errors(t *testing.T) {
	if _, err := ParseWebhook([]byte(`not json`)); err == nil {
		t.Error("expected error for invalid JSON")
	}
	if _, err := ParseWebhook([]byte(`{"eventName":"nope.unknown"}`)); err == nil {
		t.Error("expected error for unknown eventName")
	}
}

// signWebhook builds a Webhook-Signature header value using the raw HMAC key
// directly, independent of the SDK's secret parsing.
func signWebhook(rawKey []byte, id, ts string, body []byte) string {
	mac := hmac.New(sha256.New, rawKey)
	mac.Write([]byte(id + "." + ts + "." + string(body)))
	return "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestVerifyWebhookSignature(t *testing.T) {
	rawKey := []byte("super-secret-key-bytes")
	secret := "whsec_" + base64.StdEncoding.EncodeToString(rawKey)
	id := "msg_1"
	ts := "1734425918"
	body := []byte(`{"eventName":"testing.testEvent"}`)
	sig := signWebhook(rawKey, id, ts, body)

	t.Run("valid", func(t *testing.T) {
		if err := VerifyWebhookSignature(secret, id, ts, sig, body); err != nil {
			t.Fatalf("valid signature rejected: %v", err)
		}
	})

	t.Run("multiple signatures, one valid", func(t *testing.T) {
		multi := "v1,AAAA " + sig
		if err := VerifyWebhookSignature(secret, id, ts, multi, body); err != nil {
			t.Fatalf("rejected header with a valid signature among others: %v", err)
		}
	})

	t.Run("tampered body", func(t *testing.T) {
		err := VerifyWebhookSignature(secret, id, ts, sig, []byte(`{"eventName":"tampered"}`))
		if !errors.Is(err, ErrWebhookInvalidSignature) {
			t.Fatalf("err = %v, want ErrWebhookInvalidSignature", err)
		}
	})

	t.Run("wrong secret", func(t *testing.T) {
		other := "whsec_" + base64.StdEncoding.EncodeToString([]byte("different-key"))
		err := VerifyWebhookSignature(other, id, ts, sig, body)
		if !errors.Is(err, ErrWebhookInvalidSignature) {
			t.Fatalf("err = %v, want ErrWebhookInvalidSignature", err)
		}
	})

	t.Run("missing headers", func(t *testing.T) {
		err := VerifyWebhookSignature(secret, "", ts, sig, body)
		if !errors.Is(err, ErrWebhookMissingHeaders) {
			t.Fatalf("err = %v, want ErrWebhookMissingHeaders", err)
		}
	})

	t.Run("secret without prefix", func(t *testing.T) {
		raw := base64.StdEncoding.EncodeToString(rawKey)
		if err := VerifyWebhookSignature(raw, id, ts, sig, body); err != nil {
			t.Fatalf("unprefixed secret rejected: %v", err)
		}
	})
}

func TestVerifyWebhook_Header(t *testing.T) {
	rawKey := []byte("key-bytes")
	secret := "whsec_" + base64.StdEncoding.EncodeToString(rawKey)
	id := "msg_2"
	ts := "1734425918"
	body := []byte(`{"eventName":"email.opened"}`)

	h := http.Header{}
	h.Set("webhook-id", id)
	h.Set("webhook-timestamp", ts)
	h.Set("webhook-signature", signWebhook(rawKey, id, ts, body))

	if err := VerifyWebhook(secret, h, body); err != nil {
		t.Fatalf("VerifyWebhook: %v", err)
	}
}

func TestVerifyWebhook_TimestampTolerance(t *testing.T) {
	rawKey := []byte("key-bytes")
	secret := "whsec_" + base64.StdEncoding.EncodeToString(rawKey)
	id := "msg_3"
	now := time.Unix(2_000_000_000, 0)
	fixedNow := func() time.Time { return now }

	fresh := "2000000000"
	stale := "1999999000" // 1000s earlier
	body := []byte(`{"eventName":"email.opened"}`)

	t.Run("within tolerance", func(t *testing.T) {
		sig := signWebhook(rawKey, id, fresh, body)
		err := VerifyWebhookSignature(secret, id, fresh, sig, body,
			WithWebhookTimestampTolerance(5*time.Minute), withWebhookNow(fixedNow))
		if err != nil {
			t.Fatalf("fresh timestamp rejected: %v", err)
		}
	})

	t.Run("outside tolerance", func(t *testing.T) {
		sig := signWebhook(rawKey, id, stale, body)
		err := VerifyWebhookSignature(secret, id, stale, sig, body,
			WithWebhookTimestampTolerance(5*time.Minute), withWebhookNow(fixedNow))
		if !errors.Is(err, ErrWebhookTimestampOutOfTolerance) {
			t.Fatalf("err = %v, want ErrWebhookTimestampOutOfTolerance", err)
		}
	})

	t.Run("stale allowed when tolerance disabled", func(t *testing.T) {
		sig := signWebhook(rawKey, id, stale, body)
		if err := VerifyWebhookSignature(secret, id, stale, sig, body); err != nil {
			t.Fatalf("stale timestamp rejected with tolerance disabled: %v", err)
		}
	})
}
