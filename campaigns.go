package loops

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// Campaign describes a campaign, as returned by [Client.GetCampaign] and
// [Client.UpdateCampaign]. A campaign holds a single [EmailMessage] linked
// via EmailMessageID.
type Campaign struct {
	CampaignID     string  `json:"campaignId"`
	EmailMessageID *string `json:"emailMessageId"`
	Name           string  `json:"name"`
	Status         string  `json:"status"`
	CreatedAt      string  `json:"createdAt"`
	UpdatedAt      string  `json:"updatedAt"`
}

// CampaignListItem is the entry shape returned by [Client.ListCampaigns]. It
// includes the email's Subject (in addition to the fields on [Campaign]) so
// list views don't need an extra fetch.
type CampaignListItem struct {
	CampaignID     string  `json:"campaignId"`
	EmailMessageID *string `json:"emailMessageId"`
	Name           string  `json:"name"`
	Subject        string  `json:"subject"`
	Status         string  `json:"status"`
	CreatedAt      string  `json:"createdAt"`
	UpdatedAt      string  `json:"updatedAt"`
}

// LmxWarning is a non-fatal issue reported by the Loops Markup (LMX) linter
// when validating email content. Severity is typically "warning" or "info".
type LmxWarning struct {
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Path     string `json:"path,omitempty"`
}

// EmailMessageFields holds the editable fields of an email message. It is
// embedded in [UpdateEmailMessageRequest]; the request's Set map determines
// which fields are actually written.
type EmailMessageFields struct {
	Subject      string `json:"subject,omitempty"`
	PreviewText  string `json:"previewText,omitempty"`
	FromName     string `json:"fromName,omitempty"`
	FromEmail    string `json:"fromEmail,omitempty"`
	ReplyToEmail string `json:"replyToEmail,omitempty"`
	LMX          string `json:"lmx,omitempty"`
}

// CreateCampaignRequest is the request body for [Client.CreateCampaign].
type CreateCampaignRequest struct {
	Name string `json:"name"`
}

// UpdateCampaignRequest is the request body for [Client.UpdateCampaign].
type UpdateCampaignRequest struct {
	Name string `json:"name"`
}

// CampaignCreateResponse is returned by [Client.CreateCampaign]. It embeds
// the new [Campaign] and adds the initial content-revision ID for the
// campaign's email message.
type CampaignCreateResponse struct {
	Campaign
	EmailMessageContentRevisionID *string `json:"emailMessageContentRevisionId"`
}

// CreateCampaign creates a new campaign and an empty email message attached
// to it.
func (c *Client) CreateCampaign(req CreateCampaignRequest) (*CampaignCreateResponse, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	httpReq, err := c.newRequest(http.MethodPost, "/campaigns", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	resp, err := c.do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, errorFromResponse(resp)
	}

	var result CampaignCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// UpdateCampaign updates the campaign identified by id and returns its new
// state.
func (c *Client) UpdateCampaign(id string, req UpdateCampaignRequest) (*Campaign, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	httpReq, err := c.newRequest(http.MethodPost, "/campaigns/"+id, bytes.NewReader(b))
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

	var result Campaign
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetCampaign returns the campaign identified by id.
func (c *Client) GetCampaign(id string) (*Campaign, error) {
	req, err := c.newRequest(http.MethodGet, "/campaigns/"+id, nil)
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

	var result Campaign
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ListCampaigns returns a single page of campaigns along with pagination
// information. To iterate every page, use [Paginate].
func (c *Client) ListCampaigns(params PaginationParams) ([]CampaignListItem, *Pagination, error) {
	q := url.Values{}
	if params.PerPage != "" {
		q.Set("perPage", params.PerPage)
	}
	if params.Cursor != "" {
		q.Set("cursor", params.Cursor)
	}

	path := "/campaigns"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}

	req, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, errorFromResponse(resp)
	}

	var result struct {
		Pagination Pagination         `json:"pagination"`
		Data       []CampaignListItem `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Data, &result.Pagination, nil
}
