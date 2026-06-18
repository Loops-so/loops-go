package loops

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// Component is a reusable LMX snippet that can be included from other email
// messages. It is identified by ID and has an LMX body.
type Component struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	LMX  string `json:"lmx"`
}

// GetComponent returns the component identified by id.
func (c *Client) GetComponent(id string) (*Component, error) {
	req, err := c.newRequest(http.MethodGet, "/components/"+id, nil)
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

	var result Component
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ListComponents returns a single page of components along with pagination
// information. To iterate every page, use [Paginate].
func (c *Client) ListComponents(params PaginationParams) ([]Component, *Pagination, error) {
	q := url.Values{}
	if params.PerPage != "" {
		q.Set("perPage", params.PerPage)
	}
	if params.Cursor != "" {
		q.Set("cursor", params.Cursor)
	}

	path := "/components"
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
		Pagination Pagination  `json:"pagination"`
		Data       []Component `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Data, &result.Pagination, nil
}
