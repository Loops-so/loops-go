package loops

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Webhook event names, used as the eventName discriminator on webhook payloads
// and returned by [WebhookEvent.Type].
const (
	WebhookEventContactCreated                 = "contact.created"
	WebhookEventContactDeleted                 = "contact.deleted"
	WebhookEventContactUnsubscribed            = "contact.unsubscribed"
	WebhookEventContactMailingListSubscribed   = "contact.mailingList.subscribed"
	WebhookEventContactMailingListUnsubscribed = "contact.mailingList.unsubscribed"
	WebhookEventTransactionalEmailSent         = "transactional.email.sent"
	WebhookEventCampaignEmailSent              = "campaign.email.sent"
	WebhookEventLoopEmailSent                  = "loop.email.sent"
	WebhookEventEmailDelivered                 = "email.delivered"
	WebhookEventEmailSoftBounced               = "email.softBounced"
	WebhookEventEmailHardBounced               = "email.hardBounced"
	WebhookEventEmailSpamReported              = "email.spamReported"
	WebhookEventEmailClicked                   = "email.clicked"
	WebhookEventEmailUnsubscribed              = "email.unsubscribed"
	WebhookEventEmailResubscribed              = "email.resubscribed"
	WebhookEventEmailOpened                    = "email.opened"
	WebhookEventTestingTestEvent               = "testing.testEvent"
)

// Webhook email sourceType values, used on the metric payloads. Workflow
// emails use "loop".
const (
	WebhookSourceTypeLoop          = "loop"
	WebhookSourceTypeCampaign      = "campaign"
	WebhookSourceTypeTransactional = "transactional"
)

// WebhookContactIdentity identifies the contact a webhook event relates to.
type WebhookContactIdentity struct {
	ID     string  `json:"id"`
	Email  string  `json:"email"`
	UserID *string `json:"userId"`
}

// WebhookEmail describes the email a webhook event relates to.
type WebhookEmail struct {
	ID             string `json:"id"`
	EmailMessageID string `json:"emailMessageId"`
	Subject        string `json:"subject"`
}

// WebhookMailingList describes a mailing list referenced by a webhook event.
type WebhookMailingList struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	IsPublic    bool    `json:"isPublic"`
}

// WebhookContact is the full contact object delivered with a
// [WebhookContactCreatedPayload]. Custom collects any fields not part of the
// fixed schema (for example, custom contact properties defined on your
// account), mirroring [Contact].
type WebhookContact struct {
	ID           string          `json:"id"`
	Email        string          `json:"email"`
	FirstName    *string         `json:"firstName"`
	LastName     *string         `json:"lastName"`
	Source       string          `json:"source"`
	Subscribed   bool            `json:"subscribed"`
	UserGroup    string          `json:"userGroup"`
	UserID       *string         `json:"userId"`
	Notes        *string         `json:"notes"`
	MailingLists map[string]bool `json:"mailingLists"`
	OptInStatus  *string         `json:"optInStatus"`
	Custom       map[string]any  `json:"-"`
}

var knownWebhookContactFields = map[string]bool{
	"id": true, "email": true, "firstName": true, "lastName": true,
	"source": true, "subscribed": true, "userGroup": true, "userId": true,
	"notes": true, "mailingLists": true, "optInStatus": true,
}

// UnmarshalJSON implements [json.Unmarshaler] and collects unknown fields into
// [WebhookContact.Custom].
func (c *WebhookContact) UnmarshalJSON(data []byte) error {
	type Alias WebhookContact
	if err := json.Unmarshal(data, (*Alias)(c)); err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for k, v := range raw {
		if knownWebhookContactFields[k] {
			continue
		}
		if c.Custom == nil {
			c.Custom = make(map[string]any)
		}
		var val any
		if err := json.Unmarshal(v, &val); err != nil {
			c.Custom[k] = string(v)
		} else {
			c.Custom[k] = val
		}
	}
	return nil
}

// MarshalJSON implements [json.Marshaler] and merges [WebhookContact.Custom]
// into the top-level JSON object.
func (c WebhookContact) MarshalJSON() ([]byte, error) {
	type Alias WebhookContact
	b, err := json.Marshal(Alias(c))
	if err != nil {
		return nil, err
	}
	if len(c.Custom) == 0 {
		return b, nil
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	for k, v := range c.Custom {
		raw, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		m[k] = raw
	}
	return json.Marshal(m)
}

// WebhookBasePayload holds the fields common to every webhook payload except
// the test event. Concrete payload types embed it (directly or via a metric
// payload) and inherit [WebhookBasePayload.Type].
type WebhookBasePayload struct {
	EventName            string                 `json:"eventName"`
	EventTime            int64                  `json:"eventTime"`
	WebhookSchemaVersion string                 `json:"webhookSchemaVersion"`
	ContactIdentity      WebhookContactIdentity `json:"contactIdentity"`
}

// Type returns the event's eventName (one of the WebhookEvent* constants).
func (p WebhookBasePayload) Type() string { return p.EventName }

// WebhookEmailMetricPayload is the shared shape of the email delivery-metric
// events (delivered, bounced, spam reported). Exactly one of CampaignID,
// LoopID, or TransactionalID is set, matching SourceType.
type WebhookEmailMetricPayload struct {
	WebhookBasePayload
	Email           WebhookEmail `json:"email"`
	SourceType      string       `json:"sourceType"`
	CampaignID      string       `json:"campaignId,omitempty"`
	LoopID          string       `json:"loopId,omitempty"`
	TransactionalID string       `json:"transactionalId,omitempty"`
}

// WebhookMarketingEmailMetricPayload is the shared shape of the marketing
// engagement events (clicked, opened, unsubscribed, resubscribed). Unlike
// [WebhookEmailMetricPayload] it is never transactional, so SourceType is
// "loop" or "campaign".
type WebhookMarketingEmailMetricPayload struct {
	WebhookBasePayload
	Email      WebhookEmail `json:"email"`
	SourceType string       `json:"sourceType"`
	CampaignID string       `json:"campaignId,omitempty"`
	LoopID     string       `json:"loopId,omitempty"`
}

// WebhookContactCreatedPayload is the "contact.created" event.
type WebhookContactCreatedPayload struct {
	WebhookBasePayload
	Contact WebhookContact `json:"contact"`
}

// WebhookContactDeletedPayload is the "contact.deleted" event.
type WebhookContactDeletedPayload struct {
	WebhookBasePayload
}

// WebhookContactUnsubscribedPayload is the "contact.unsubscribed" event.
type WebhookContactUnsubscribedPayload struct {
	WebhookBasePayload
}

// WebhookContactMailingListSubscribedPayload is the
// "contact.mailingList.subscribed" event.
type WebhookContactMailingListSubscribedPayload struct {
	WebhookBasePayload
	MailingList WebhookMailingList `json:"mailingList"`
}

// WebhookContactMailingListUnsubscribedPayload is the
// "contact.mailingList.unsubscribed" event.
type WebhookContactMailingListUnsubscribedPayload struct {
	WebhookBasePayload
	MailingList WebhookMailingList `json:"mailingList"`
}

// WebhookTransactionalEmailSentPayload is the "transactional.email.sent" event.
type WebhookTransactionalEmailSentPayload struct {
	WebhookBasePayload
	TransactionalID   string       `json:"transactionalId"`
	TransactionalName string       `json:"transactionalName"`
	Email             WebhookEmail `json:"email"`
}

// WebhookCampaignEmailSentPayload is the "campaign.email.sent" event.
type WebhookCampaignEmailSentPayload struct {
	WebhookBasePayload
	CampaignID   string               `json:"campaignId"`
	CampaignName string               `json:"campaignName"`
	Email        WebhookEmail         `json:"email"`
	MailingLists []WebhookMailingList `json:"mailingLists"`
}

// WebhookLoopEmailSentPayload is the "loop.email.sent" event. Loop is Loops'
// term for a workflow.
type WebhookLoopEmailSentPayload struct {
	WebhookBasePayload
	LoopID       string               `json:"loopId"`
	LoopName     string               `json:"loopName"`
	Email        WebhookEmail         `json:"email"`
	MailingLists []WebhookMailingList `json:"mailingLists"`
}

// WebhookEmailDeliveredPayload is the "email.delivered" event.
type WebhookEmailDeliveredPayload struct {
	WebhookEmailMetricPayload
}

// WebhookEmailSoftBouncedPayload is the "email.softBounced" event.
type WebhookEmailSoftBouncedPayload struct {
	WebhookEmailMetricPayload
}

// WebhookEmailHardBouncedPayload is the "email.hardBounced" event.
type WebhookEmailHardBouncedPayload struct {
	WebhookEmailMetricPayload
}

// WebhookEmailSpamReportedPayload is the "email.spamReported" event.
type WebhookEmailSpamReportedPayload struct {
	WebhookEmailMetricPayload
}

// WebhookEmailClickedPayload is the "email.clicked" event.
type WebhookEmailClickedPayload struct {
	WebhookMarketingEmailMetricPayload
}

// WebhookEmailUnsubscribedPayload is the "email.unsubscribed" event.
type WebhookEmailUnsubscribedPayload struct {
	WebhookMarketingEmailMetricPayload
}

// WebhookEmailResubscribedPayload is the "email.resubscribed" event.
type WebhookEmailResubscribedPayload struct {
	WebhookMarketingEmailMetricPayload
}

// WebhookEmailOpenedPayload is the "email.opened" event.
type WebhookEmailOpenedPayload struct {
	WebhookMarketingEmailMetricPayload
}

// WebhookTestingTestEventPayload is the "testing.testEvent" event, triggered
// from the Webhooks settings page. Unlike other events it carries no contact
// identity.
type WebhookTestingTestEventPayload struct {
	EventName            string `json:"eventName"`
	EventTime            int64  `json:"eventTime"`
	Message              string `json:"message"`
	WebhookSchemaVersion string `json:"webhookSchemaVersion"`
}

// Type returns the event's eventName ([WebhookEventTestingTestEvent]).
func (p WebhookTestingTestEventPayload) Type() string { return p.EventName }

// WebhookEvent is implemented by every webhook payload type returned by
// [ParseWebhook]. Type reports the eventName (one of the WebhookEvent*
// constants); type-switch on the concrete type to access event-specific
// fields.
type WebhookEvent interface {
	Type() string
}

// ParseWebhook decodes a verified webhook request body into the concrete event
// type identified by its eventName. Verify the signature with [VerifyWebhook]
// first. It returns an error if the body is not valid JSON or the eventName is
// unrecognized.
func ParseWebhook(body []byte) (WebhookEvent, error) {
	var probe struct {
		EventName string `json:"eventName"`
	}
	if err := json.Unmarshal(body, &probe); err != nil {
		return nil, fmt.Errorf("loops: failed to decode webhook eventName: %w", err)
	}

	var evt WebhookEvent
	switch probe.EventName {
	case WebhookEventContactCreated:
		evt = &WebhookContactCreatedPayload{}
	case WebhookEventContactDeleted:
		evt = &WebhookContactDeletedPayload{}
	case WebhookEventContactUnsubscribed:
		evt = &WebhookContactUnsubscribedPayload{}
	case WebhookEventContactMailingListSubscribed:
		evt = &WebhookContactMailingListSubscribedPayload{}
	case WebhookEventContactMailingListUnsubscribed:
		evt = &WebhookContactMailingListUnsubscribedPayload{}
	case WebhookEventTransactionalEmailSent:
		evt = &WebhookTransactionalEmailSentPayload{}
	case WebhookEventCampaignEmailSent:
		evt = &WebhookCampaignEmailSentPayload{}
	case WebhookEventLoopEmailSent:
		evt = &WebhookLoopEmailSentPayload{}
	case WebhookEventEmailDelivered:
		evt = &WebhookEmailDeliveredPayload{}
	case WebhookEventEmailSoftBounced:
		evt = &WebhookEmailSoftBouncedPayload{}
	case WebhookEventEmailHardBounced:
		evt = &WebhookEmailHardBouncedPayload{}
	case WebhookEventEmailSpamReported:
		evt = &WebhookEmailSpamReportedPayload{}
	case WebhookEventEmailClicked:
		evt = &WebhookEmailClickedPayload{}
	case WebhookEventEmailUnsubscribed:
		evt = &WebhookEmailUnsubscribedPayload{}
	case WebhookEventEmailResubscribed:
		evt = &WebhookEmailResubscribedPayload{}
	case WebhookEventEmailOpened:
		evt = &WebhookEmailOpenedPayload{}
	case WebhookEventTestingTestEvent:
		evt = &WebhookTestingTestEventPayload{}
	default:
		return nil, fmt.Errorf("loops: unknown webhook eventName %q", probe.EventName)
	}

	if err := json.Unmarshal(body, evt); err != nil {
		return nil, fmt.Errorf("loops: failed to decode %q webhook: %w", probe.EventName, err)
	}
	return evt, nil
}

// Errors returned by [VerifyWebhook] and [VerifyWebhookSignature].
var (
	// ErrWebhookMissingHeaders is returned when any of the webhook-id,
	// webhook-timestamp, or webhook-signature headers is absent.
	ErrWebhookMissingHeaders = errors.New("loops: missing webhook signature headers")
	// ErrWebhookInvalidSignature is returned when no provided signature matches
	// the one computed from the signing secret.
	ErrWebhookInvalidSignature = errors.New("loops: webhook signature verification failed")
	// ErrWebhookTimestampOutOfTolerance is returned when timestamp tolerance is
	// enabled via [WithWebhookTimestampTolerance] and the webhook-timestamp is
	// too far from now.
	ErrWebhookTimestampOutOfTolerance = errors.New("loops: webhook timestamp outside tolerance")
)

type webhookVerifyConfig struct {
	tolerance time.Duration
	now       func() time.Time
}

// WebhookVerifyOption configures [VerifyWebhook] and [VerifyWebhookSignature].
type WebhookVerifyOption func(*webhookVerifyConfig)

// WithWebhookTimestampTolerance enables replay protection: the webhook-timestamp
// must be within d of the current time or verification fails with
// [ErrWebhookTimestampOutOfTolerance]. Disabled by default, matching Loops'
// reference implementation, which verifies only the signature.
func WithWebhookTimestampTolerance(d time.Duration) WebhookVerifyOption {
	return func(c *webhookVerifyConfig) { c.tolerance = d }
}

// withWebhookNow overrides the clock used for timestamp tolerance (tests only).
func withWebhookNow(now func() time.Time) WebhookVerifyOption {
	return func(c *webhookVerifyConfig) { c.now = now }
}

// VerifyWebhook verifies the signature of an incoming Loops webhook request
// using the endpoint signing secret from the Loops dashboard (the value shown
// with a "whsec_" prefix). It reads the webhook-id, webhook-timestamp, and
// webhook-signature headers from header and returns nil when the signature is
// valid. body must be the exact raw request bytes, read before any JSON
// decoding.
func VerifyWebhook(secret string, header http.Header, body []byte, opts ...WebhookVerifyOption) error {
	return VerifyWebhookSignature(
		secret,
		header.Get("webhook-id"),
		header.Get("webhook-timestamp"),
		header.Get("webhook-signature"),
		body,
		opts...,
	)
}

// VerifyWebhookSignature is the header-agnostic form of [VerifyWebhook] for
// callers that read the signature headers themselves.
func VerifyWebhookSignature(secret, webhookID, webhookTimestamp, webhookSignature string, body []byte, opts ...WebhookVerifyOption) error {
	cfg := webhookVerifyConfig{now: time.Now}
	for _, opt := range opts {
		opt(&cfg)
	}

	if webhookID == "" || webhookTimestamp == "" || webhookSignature == "" {
		return ErrWebhookMissingHeaders
	}

	key, err := webhookSigningKey(secret)
	if err != nil {
		return err
	}

	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(webhookID + "." + webhookTimestamp + "." + string(body)))
	expected := mac.Sum(nil)

	if !webhookSignatureMatches(webhookSignature, expected) {
		return ErrWebhookInvalidSignature
	}

	if cfg.tolerance > 0 {
		if err := checkWebhookTimestamp(webhookTimestamp, cfg.now(), cfg.tolerance); err != nil {
			return err
		}
	}
	return nil
}

// webhookSigningKey derives the HMAC key from the signing secret. Loops secrets
// are formatted "<prefix>_<base64>"; the base64 portion decodes to the raw key.
// A secret with no underscore is treated as raw base64.
func webhookSigningKey(secret string) ([]byte, error) {
	if secret == "" {
		return nil, errors.New("loops: empty webhook signing secret")
	}
	b64 := secret
	if _, after, found := strings.Cut(secret, "_"); found {
		b64 = after
	}
	key, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("loops: invalid webhook signing secret encoding: %w", err)
	}
	return key, nil
}

// webhookSignatureMatches reports whether any space-separated "version,sig"
// entry in the header carries the expected signature. The comparison is
// constant-time.
func webhookSignatureMatches(header string, expected []byte) bool {
	for _, part := range strings.Fields(header) {
		_, sig, ok := strings.Cut(part, ",")
		if !ok {
			continue
		}
		got, err := base64.StdEncoding.DecodeString(sig)
		if err != nil {
			continue
		}
		if hmac.Equal(got, expected) {
			return true
		}
	}
	return false
}

func checkWebhookTimestamp(ts string, now time.Time, tolerance time.Duration) error {
	secs, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return fmt.Errorf("loops: invalid webhook timestamp %q: %w", ts, err)
	}
	delta := now.Sub(time.Unix(secs, 0))
	if delta < 0 {
		delta = -delta
	}
	if delta > tolerance {
		return fmt.Errorf("%w (%s old)", ErrWebhookTimestampOutOfTolerance, delta)
	}
	return nil
}
