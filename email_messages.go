package loops

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// EmailMessage is the persisted form of an email's content and metadata,
// including its LMX body. Warnings is populated by the LMX linter when the
// message is fetched or updated.
type EmailMessage struct {
	EmailMessageID    string       `json:"emailMessageId"`
	CampaignID        *string      `json:"campaignId"`
	Subject           string       `json:"subject"`
	PreviewText       string       `json:"previewText"`
	FromName          string       `json:"fromName"`
	FromEmail         string       `json:"fromEmail"`
	ReplyToEmail      string       `json:"replyToEmail"`
	LMX               string       `json:"lmx"`
	ContentRevisionID *string      `json:"contentRevisionId"`
	UpdatedAt         string       `json:"updatedAt"`
	Warnings          []LmxWarning `json:"warnings,omitempty"`
}

// UpdateEmailMessageRequest is the request body for [Client.UpdateEmailMessage].
//
// Set selects which fields from the embedded [EmailMessageFields] are
// applied — only fields whose key is true in Set are sent. This lets the
// caller distinguish "leave alone" from "set to empty string".
//
// ExpectedRevisionID is optional; if set, the update fails if the message's
// current ContentRevisionID does not match (optimistic concurrency control).
type UpdateEmailMessageRequest struct {
	EmailMessageFields
	Set                map[string]bool
	ExpectedRevisionID string
}

// GetEmailMessage returns the email message identified by id.
func (c *Client) GetEmailMessage(id string) (*EmailMessage, error) {
	req, err := c.newRequest(http.MethodGet, "/email-messages/"+id, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errorFromResponse(resp)
	}

	var result EmailMessage
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// UpdateEmailMessage updates the email message identified by id with the
// fields selected in req.Set and returns its new state. See
// [UpdateEmailMessageRequest] for the selection semantics.
func (c *Client) UpdateEmailMessage(id string, req UpdateEmailMessageRequest) (*EmailMessage, error) {
	body := map[string]any{}
	if req.Set["subject"] {
		body["subject"] = req.Subject
	}
	if req.Set["previewText"] {
		body["previewText"] = req.PreviewText
	}
	if req.Set["fromName"] {
		body["fromName"] = req.FromName
	}
	if req.Set["fromEmail"] {
		body["fromEmail"] = req.FromEmail
	}
	if req.Set["replyToEmail"] {
		body["replyToEmail"] = req.ReplyToEmail
	}
	if req.Set["lmx"] {
		body["lmx"] = req.LMX
	}
	if req.ExpectedRevisionID != "" {
		body["expectedRevisionId"] = req.ExpectedRevisionID
	}

	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	httpReq, err := c.newRequest(http.MethodPost, "/email-messages/"+id, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	resp, err := c.do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errorFromResponse(resp)
	}

	var result EmailMessage
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
