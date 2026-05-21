package loops

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const createUploadResponse = `{
	"success": true,
	"emailAssetId": "asset_abc123",
	"presignedUrl": "https://storage.example.com/upload/abc?sig=xyz",
	"expiresAt": "2026-05-21T10:15:00Z"
}`

func TestCreateUpload(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantAPIErr *APIError
		wantErrMsg string
	}{
		{
			name:       "success",
			statusCode: http.StatusOK,
			body:       createUploadResponse,
		},
		{
			name:       "unsupported content type",
			statusCode: http.StatusBadRequest,
			body:       `{"success":false,"message":"Unsupported content type","supportedContentTypes":["image/jpeg","image/png","image/gif","image/webp"]}`,
			wantAPIErr: &APIError{StatusCode: http.StatusBadRequest, Message: "Unsupported content type"},
		},
		{
			name:       "file too large",
			statusCode: http.StatusRequestEntityTooLarge,
			body:       `{"success":false,"message":"File too large","maxBytes":4000000}`,
			wantAPIErr: &APIError{StatusCode: http.StatusRequestEntityTooLarge, Message: "File too large"},
		},
		{
			name:       "email message not found",
			statusCode: http.StatusNotFound,
			body:       `{"success":false,"message":"Email message not found"}`,
			wantAPIErr: &APIError{StatusCode: http.StatusNotFound, Message: "Email message not found"},
		},
		{
			name:       "invalid json",
			statusCode: http.StatusOK,
			body:       `not json`,
			wantErrMsg: "failed to decode response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath, gotMethod string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				gotMethod = r.Method
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := NewClient("test-key", WithBaseURL(server.URL))
			result, err := client.CreateUpload(CreateUploadRequest{
				EmailMessageID: "em_abc123",
				ContentType:    "image/png",
				ContentLength:  1024,
			})

			if tt.wantAPIErr != nil {
				var apiErr *APIError
				if !errors.As(err, &apiErr) {
					t.Fatalf("expected *APIError, got %T: %v", err, err)
				}
				if apiErr.StatusCode != tt.wantAPIErr.StatusCode {
					t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, tt.wantAPIErr.StatusCode)
				}
				if apiErr.Message != tt.wantAPIErr.Message {
					t.Errorf("Message = %q, want %q", apiErr.Message, tt.wantAPIErr.Message)
				}
				return
			}

			if tt.wantErrMsg != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErrMsg)
				}
				if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("error = %q, want it to contain %q", err.Error(), tt.wantErrMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotMethod != http.MethodPost {
				t.Errorf("method = %q, want POST", gotMethod)
			}
			if gotPath != "/uploads" {
				t.Errorf("path = %q, want /uploads", gotPath)
			}
			if result.EmailAssetID != "asset_abc123" {
				t.Errorf("EmailAssetID = %q, want asset_abc123", result.EmailAssetID)
			}
			if result.PresignedURL != "https://storage.example.com/upload/abc?sig=xyz" {
				t.Errorf("PresignedURL = %q", result.PresignedURL)
			}
			if result.ExpiresAt != "2026-05-21T10:15:00Z" {
				t.Errorf("ExpiresAt = %q, want 2026-05-21T10:15:00Z", result.ExpiresAt)
			}
		})
	}
}

func TestCreateUpload_RequestBody(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(createUploadResponse))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.CreateUpload(CreateUploadRequest{
		EmailMessageID: "em_abc123",
		ContentType:    "image/png",
		ContentLength:  2048,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if body["emailMessageId"] != "em_abc123" {
		t.Errorf("emailMessageId = %v, want em_abc123", body["emailMessageId"])
	}
	if body["contentType"] != "image/png" {
		t.Errorf("contentType = %v, want image/png", body["contentType"])
	}
	if body["contentLength"] != float64(2048) {
		t.Errorf("contentLength = %v, want 2048", body["contentLength"])
	}
}

const completeUploadResponse = `{
	"success": true,
	"emailAssetId": "asset_abc123",
	"finalUrl": "https://assets.example.com/asset_abc123.png"
}`

func TestCompleteUpload(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		statusCode int
		body       string
		wantAPIErr *APIError
		wantErrMsg string
	}{
		{
			name:       "success",
			id:         "asset_abc123",
			statusCode: http.StatusOK,
			body:       completeUploadResponse,
		},
		{
			name:       "not found",
			id:         "asset_missing",
			statusCode: http.StatusNotFound,
			body:       `{"success":false,"message":"Asset not found"}`,
			wantAPIErr: &APIError{StatusCode: http.StatusNotFound, Message: "Asset not found"},
		},
		{
			name:       "too early",
			id:         "asset_abc123",
			statusCode: http.StatusTooEarly,
			body:       `{"success":false,"message":"Upload not yet complete"}`,
			wantAPIErr: &APIError{StatusCode: http.StatusTooEarly, Message: "Upload not yet complete"},
		},
		{
			name:       "invalid json",
			id:         "asset_abc123",
			statusCode: http.StatusOK,
			body:       `not json`,
			wantErrMsg: "failed to decode response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath, gotMethod string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				gotMethod = r.Method
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := NewClient("test-key", WithBaseURL(server.URL))
			result, err := client.CompleteUpload(tt.id)

			if tt.wantAPIErr != nil {
				var apiErr *APIError
				if !errors.As(err, &apiErr) {
					t.Fatalf("expected *APIError, got %T: %v", err, err)
				}
				if apiErr.StatusCode != tt.wantAPIErr.StatusCode {
					t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, tt.wantAPIErr.StatusCode)
				}
				if apiErr.Message != tt.wantAPIErr.Message {
					t.Errorf("Message = %q, want %q", apiErr.Message, tt.wantAPIErr.Message)
				}
				return
			}

			if tt.wantErrMsg != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErrMsg)
				}
				if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("error = %q, want it to contain %q", err.Error(), tt.wantErrMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotMethod != http.MethodPost {
				t.Errorf("method = %q, want POST", gotMethod)
			}
			if want := "/uploads/" + tt.id + "/complete"; gotPath != want {
				t.Errorf("path = %q, want %q", gotPath, want)
			}
			if result.EmailAssetID != "asset_abc123" {
				t.Errorf("EmailAssetID = %q, want asset_abc123", result.EmailAssetID)
			}
			if result.FinalURL != "https://assets.example.com/asset_abc123.png" {
				t.Errorf("FinalURL = %q", result.FinalURL)
			}
		})
	}
}
