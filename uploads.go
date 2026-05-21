package loops

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type CreateUploadRequest struct {
	EmailMessageID string `json:"emailMessageId"`
	ContentType    string `json:"contentType"`
	ContentLength  int64  `json:"contentLength"`
}

type CreateUploadResponse struct {
	EmailAssetID string `json:"emailAssetId"`
	PresignedURL string `json:"presignedUrl"`
	ExpiresAt    string `json:"expiresAt"`
}

type CompleteUploadResponse struct {
	EmailAssetID string `json:"emailAssetId"`
	FinalURL     string `json:"finalUrl"`
}

type UploadRequest struct {
	EmailMessageID string
	ContentType    string
	ContentLength  int64
	Body           io.Reader
}

func (c *Client) CreateUpload(req CreateUploadRequest) (*CreateUploadResponse, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	httpReq, err := c.newRequest(http.MethodPost, "/uploads", bytes.NewReader(b))
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

	var result CreateUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (c *Client) Upload(req UploadRequest) (*CompleteUploadResponse, error) {
	created, err := c.CreateUpload(CreateUploadRequest{
		EmailMessageID: req.EmailMessageID,
		ContentType:    req.ContentType,
		ContentLength:  req.ContentLength,
	})
	if err != nil {
		return nil, err
	}

	putReq, err := http.NewRequest(http.MethodPut, created.PresignedURL, req.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to build upload request: %w", err)
	}
	putReq.Header.Set("Content-Type", req.ContentType)
	putReq.ContentLength = req.ContentLength

	putResp, err := c.httpClient.Do(putReq)
	if err != nil {
		return nil, fmt.Errorf("upload to presigned URL failed: %w", err)
	}
	defer putResp.Body.Close()

	if putResp.StatusCode < 200 || putResp.StatusCode >= 300 {
		return nil, fmt.Errorf("upload to presigned URL failed: status %d", putResp.StatusCode)
	}

	return c.CompleteUpload(created.EmailAssetID)
}

func (c *Client) CompleteUpload(id string) (*CompleteUploadResponse, error) {
	req, err := c.newRequest(http.MethodPost, "/uploads/"+id+"/complete", nil)
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

	var result CompleteUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
