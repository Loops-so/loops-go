package loops

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const getComponentResponse = `{
	"success": true,
	"componentId": "cmpt_abc123",
	"name": "Header",
	"lmx": "<H1>Hello</H1>"
}`

const listComponentsResponse = `{
	"success": true,
	"pagination": {
		"totalResults": 2,
		"returnedResults": 2,
		"perPage": 20,
		"totalPages": 1,
		"nextCursor": "",
		"nextPage": ""
	},
	"data": [
		{
			"componentId": "cmpt_1",
			"name": "Header",
			"lmx": "<H1>Hello</H1>"
		},
		{
			"componentId": "cmpt_2",
			"name": "Footer",
			"lmx": "<Paragraph>Bye</Paragraph>"
		}
	]
}`

func TestGetComponent(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		statusCode int
		body       string
		wantAPIErr *APIError
		wantErrMsg string
		wantID     string
	}{
		{
			name:       "success",
			id:         "cmpt_abc123",
			statusCode: http.StatusOK,
			body:       getComponentResponse,
			wantID:     "cmpt_abc123",
		},
		{
			name:       "not found",
			id:         "cmpt_missing",
			statusCode: http.StatusNotFound,
			body:       `{"success":false,"message":"Component not found"}`,
			wantAPIErr: &APIError{StatusCode: http.StatusNotFound, Message: "Component not found"},
		},
		{
			name:       "invalid id",
			id:         "bad",
			statusCode: http.StatusBadRequest,
			body:       `{"success":false,"message":"Invalid componentId"}`,
			wantAPIErr: &APIError{StatusCode: http.StatusBadRequest, Message: "Invalid componentId"},
		},
		{
			name:       "invalid json",
			id:         "cmpt_abc123",
			statusCode: http.StatusOK,
			body:       `not json`,
			wantErrMsg: "failed to decode response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := NewClient("test-key", WithBaseURL(server.URL))
			result, err := client.GetComponent(tt.id)

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
			if want := "/components/" + tt.id; gotPath != want {
				t.Errorf("path = %q, want %q", gotPath, want)
			}
			if result.ComponentID != tt.wantID {
				t.Errorf("ComponentID = %q, want %q", result.ComponentID, tt.wantID)
			}
			if result.Name != "Header" {
				t.Errorf("Name = %q, want Header", result.Name)
			}
			if result.LMX != "<H1>Hello</H1>" {
				t.Errorf("LMX = %q, want <H1>Hello</H1>", result.LMX)
			}
		})
	}
}

func TestListComponents(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantAPIErr *APIError
		wantErrMsg string
		wantCount  int
	}{
		{
			name:       "success",
			statusCode: http.StatusOK,
			body:       listComponentsResponse,
			wantCount:  2,
		},
		{
			name:       "empty list",
			statusCode: http.StatusOK,
			body:       `{"success":true,"pagination":{"totalResults":0},"data":[]}`,
			wantCount:  0,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"success":false,"error":"Invalid API key"}`,
			wantAPIErr: &APIError{StatusCode: http.StatusUnauthorized, Message: "Invalid API key"},
		},
		{
			name:       "invalid perPage",
			statusCode: http.StatusBadRequest,
			body:       `{"success":false,"message":"perPage must be between 10 and 50"}`,
			wantAPIErr: &APIError{StatusCode: http.StatusBadRequest, Message: "perPage must be between 10 and 50"},
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
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := NewClient("test-key", WithBaseURL(server.URL))
			components, pagination, err := client.ListComponents(PaginationParams{})

			if tt.wantAPIErr != nil {
				var apiErr *APIError
				if !errors.As(err, &apiErr) {
					t.Fatalf("expected *APIError, got %T: %v", err, err)
				}
				if apiErr.StatusCode != tt.wantAPIErr.StatusCode {
					t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, tt.wantAPIErr.StatusCode)
				}
				if tt.wantAPIErr.Message != "" && apiErr.Message != tt.wantAPIErr.Message {
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
			if len(components) != tt.wantCount {
				t.Errorf("len(components) = %d, want %d", len(components), tt.wantCount)
			}
			if pagination == nil {
				t.Fatal("expected pagination, got nil")
			}
		})
	}
}

func TestListComponents_ResponseData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(listComponentsResponse))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	components, _, err := client.ListComponents(PaginationParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if components[0].ComponentID != "cmpt_1" {
		t.Errorf("ComponentID = %q, want cmpt_1", components[0].ComponentID)
	}
	if components[0].Name != "Header" {
		t.Errorf("Name = %q, want Header", components[0].Name)
	}
	if components[1].LMX != "<Paragraph>Bye</Paragraph>" {
		t.Errorf("LMX = %q, want <Paragraph>Bye</Paragraph>", components[1].LMX)
	}
}

func TestListComponents_QueryParams(t *testing.T) {
	tests := []struct {
		name        string
		params      PaginationParams
		wantPerPage string
		wantCursor  string
	}{
		{
			name:   "no params",
			params: PaginationParams{},
		},
		{
			name:        "both params",
			params:      PaginationParams{PerPage: "10", Cursor: "xyz"},
			wantPerPage: "10",
			wantCursor:  "xyz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPerPage, gotCursor string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPerPage = r.URL.Query().Get("perPage")
				gotCursor = r.URL.Query().Get("cursor")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"pagination":{},"data":[]}`))
			}))
			defer server.Close()

			client := NewClient("test-key", WithBaseURL(server.URL))
			client.ListComponents(tt.params)

			if gotPerPage != tt.wantPerPage {
				t.Errorf("perPage = %q, want %q", gotPerPage, tt.wantPerPage)
			}
			if gotCursor != tt.wantCursor {
				t.Errorf("cursor = %q, want %q", gotCursor, tt.wantCursor)
			}
		})
	}
}
