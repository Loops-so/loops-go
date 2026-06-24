package loops

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const listWorkflowsResponse = `{
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
			"id": "wf_1",
			"name": "Onboarding",
			"createdAt": "2026-05-01T10:00:00Z",
			"updatedAt": "2026-05-02T10:00:00Z"
		},
		{
			"id": "wf_2",
			"name": "Win-back",
			"createdAt": "2026-04-01T10:00:00Z",
			"updatedAt": "2026-04-05T10:00:00Z"
		}
	]
}`

const getSimplifiedWorkflowResponse = `{
	"id": "wf_1",
	"name": "Onboarding",
	"description": "Welcome new signups",
	"emoji": "👋",
	"mailingListId": null,
	"rootNodeId": "node_root",
	"nodes": {
		"node_root": {
			"typeName": "SignupTrigger",
			"nextNodeIds": ["node_send"]
		},
		"node_send": {
			"typeName": "SendEmailAction",
			"nextNodeIds": ["node_branch"],
			"emailMessageId": "em_welcome",
			"subject": "Welcome to Acme"
		},
		"node_branch": {
			"typeName": "BranchNode",
			"nextNodeIds": ["node_exit"]
		},
		"node_exit": {
			"typeName": "ExitAction",
			"nextNodeIds": []
		}
	}
}`

func TestListWorkflows(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(listWorkflowsResponse))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	workflows, pagination, err := client.ListWorkflows(PaginationParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(workflows) != 2 {
		t.Fatalf("len(workflows) = %d, want 2", len(workflows))
	}
	if pagination == nil {
		t.Fatal("pagination is nil")
	}
	if workflows[0].ID != "wf_1" || workflows[0].Name != "Onboarding" {
		t.Errorf("workflows[0] = %+v", workflows[0])
	}
}

func TestListWorkflows_QueryParams(t *testing.T) {
	var gotPath, gotPerPage, gotCursor string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotPerPage = r.URL.Query().Get("perPage")
		gotCursor = r.URL.Query().Get("cursor")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"pagination":{},"data":[]}`))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	if _, _, err := client.ListWorkflows(PaginationParams{PerPage: "25", Cursor: "abc"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/workflows" {
		t.Errorf("path = %q, want /workflows", gotPath)
	}
	if gotPerPage != "25" {
		t.Errorf("perPage = %q, want 25", gotPerPage)
	}
	if gotCursor != "abc" {
		t.Errorf("cursor = %q, want abc", gotCursor)
	}
}

func TestGetWorkflow(t *testing.T) {
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
			id:         "wf_1",
			statusCode: http.StatusOK,
			body:       getSimplifiedWorkflowResponse,
		},
		{
			name:       "not found",
			id:         "wf_missing",
			statusCode: http.StatusNotFound,
			body:       `{"message":"Workflow not found"}`,
			wantAPIErr: &APIError{StatusCode: http.StatusNotFound, Message: "Workflow not found"},
		},
		{
			name:       "invalid json",
			id:         "wf_1",
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
			result, err := client.GetWorkflow(tt.id)

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
			if want := "/workflows/" + tt.id; gotPath != want {
				t.Errorf("path = %q, want %q", gotPath, want)
			}
			if result.ID != "wf_1" {
				t.Errorf("ID = %q, want wf_1", result.ID)
			}
			if result.RootNodeID == nil || *result.RootNodeID != "node_root" {
				t.Errorf("RootNodeID = %v, want node_root", result.RootNodeID)
			}
			if result.MailingListID != nil {
				t.Errorf("MailingListID = %v, want nil", result.MailingListID)
			}
			if len(result.Nodes) != 4 {
				t.Fatalf("len(Nodes) = %d, want 4", len(result.Nodes))
			}

			root := result.Nodes["node_root"]
			if root.TypeName != WorkflowNodeTypeSignupTrigger {
				t.Errorf("root.TypeName = %q, want SignupTrigger", root.TypeName)
			}
			if root.SignupTrigger == nil {
				t.Fatal("root.SignupTrigger is nil")
			}
			if len(root.SignupTrigger.NextNodeIDs) != 1 || root.SignupTrigger.NextNodeIDs[0] != "node_send" {
				t.Errorf("root.SignupTrigger.NextNodeIDs = %v", root.SignupTrigger.NextNodeIDs)
			}

			send := result.Nodes["node_send"]
			if send.TypeName != WorkflowNodeTypeSendEmailAction {
				t.Errorf("send.TypeName = %q, want SendEmailAction", send.TypeName)
			}
			if send.SendEmailAction == nil {
				t.Fatal("send.SendEmailAction is nil")
			}
			if send.SendEmailAction.EmailMessageID != "em_welcome" {
				t.Errorf("send.SendEmailAction.EmailMessageID = %q", send.SendEmailAction.EmailMessageID)
			}
			if send.SendEmailAction.Subject != "Welcome to Acme" {
				t.Errorf("send.SendEmailAction.Subject = %q", send.SendEmailAction.Subject)
			}

			branch := result.Nodes["node_branch"]
			if branch.TypeName != WorkflowNodeTypeBranchNode || branch.BranchNode == nil {
				t.Errorf("branch = %+v", branch)
			}

			exit := result.Nodes["node_exit"]
			if exit.TypeName != WorkflowNodeTypeExitAction || exit.ExitAction == nil {
				t.Errorf("exit = %+v", exit)
			}
		})
	}
}

func TestGetWorkflowNode_EventTrigger(t *testing.T) {
	body := `{
		"id": "node_evt",
		"workflowId": "wf_1",
		"typeName": "EventTrigger",
		"nextNodeIds": ["node_next"],
		"eventName": "signed_up",
		"eventProperties": [
			{"name": "plan", "type": "string"},
			{"name": "seats", "type": "number"}
		],
		"reEligible": true
	}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if want := "/workflows/wf_1/nodes/node_evt"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	node, err := client.GetWorkflowNode("wf_1", "node_evt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node.TypeName != WorkflowNodeTypeEventTrigger {
		t.Fatalf("TypeName = %q, want EventTrigger", node.TypeName)
	}
	if node.EventTrigger == nil {
		t.Fatal("EventTrigger is nil")
	}
	if node.EventTrigger.ID != "node_evt" || node.EventTrigger.WorkflowID != "wf_1" {
		t.Errorf("EventTrigger ids = %+v", node.EventTrigger)
	}
	if node.EventTrigger.EventName == nil || *node.EventTrigger.EventName != "signed_up" {
		t.Errorf("EventName = %v", node.EventTrigger.EventName)
	}
	if len(node.EventTrigger.EventProperties) != 2 {
		t.Fatalf("len(EventProperties) = %d", len(node.EventTrigger.EventProperties))
	}
	if node.EventTrigger.EventProperties[1].Name != "seats" || node.EventTrigger.EventProperties[1].Type != "number" {
		t.Errorf("EventProperties[1] = %+v", node.EventTrigger.EventProperties[1])
	}
	if !node.EventTrigger.ReEligible {
		t.Error("ReEligible = false, want true")
	}
}

func TestGetWorkflowNode_ContactPropertyTrigger_NullQuery(t *testing.T) {
	body := `{
		"id": "node_cp",
		"workflowId": "wf_1",
		"typeName": "ContactPropertyTrigger",
		"nextNodeIds": [],
		"contactPropertyQuery": null,
		"reEligible": false
	}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	node, err := client.GetWorkflowNode("wf_1", "node_cp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node.ContactPropertyTrigger == nil {
		t.Fatal("ContactPropertyTrigger is nil")
	}
	if node.ContactPropertyTrigger.ContactPropertyQuery != nil {
		t.Errorf("ContactPropertyQuery = %+v, want nil", node.ContactPropertyTrigger.ContactPropertyQuery)
	}
}

func TestGetWorkflowNode_ContactPropertyTrigger_WithQuery(t *testing.T) {
	body := `{
		"id": "node_cp",
		"workflowId": "wf_1",
		"typeName": "ContactPropertyTrigger",
		"nextNodeIds": ["node_next"],
		"contactPropertyQuery": {
			"key": "plan",
			"is": {"value": "pro", "operator": "equal"},
			"was": {"value": "free", "operator": "equal"}
		},
		"reEligible": true
	}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	node, err := client.GetWorkflowNode("wf_1", "node_cp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	q := node.ContactPropertyTrigger.ContactPropertyQuery
	if q == nil {
		t.Fatal("ContactPropertyQuery is nil")
	}
	if q.Key != "plan" {
		t.Errorf("Key = %q, want plan", q.Key)
	}
	if q.Is.Operator != "equal" {
		t.Errorf("Is.Operator = %q, want equal", q.Is.Operator)
	}
	if q.Is.Value.String == nil || *q.Is.Value.String != "pro" {
		t.Errorf("Is.Value.String = %v", q.Is.Value.String)
	}
	if q.Was.Value.String == nil || *q.Was.Value.String != "free" {
		t.Errorf("Was.Value.String = %v", q.Was.Value.String)
	}
}

func TestGetWorkflowNode_TimerAction(t *testing.T) {
	body := `{
		"id": "node_t",
		"workflowId": "wf_1",
		"typeName": "TimerAction",
		"nextNodeIds": ["node_next"],
		"amount": 3,
		"unit": "d"
	}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	node, err := client.GetWorkflowNode("wf_1", "node_t")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node.TimerAction == nil {
		t.Fatal("TimerAction is nil")
	}
	if node.TimerAction.Amount != 3 {
		t.Errorf("Amount = %v, want 3", node.TimerAction.Amount)
	}
	if node.TimerAction.Unit != WorkflowTimerUnitDays {
		t.Errorf("Unit = %q, want d", node.TimerAction.Unit)
	}
}

func TestGetWorkflowNode_AudienceFilter(t *testing.T) {
	body := `{
		"id": "node_af",
		"workflowId": "wf_1",
		"typeName": "AudienceFilter",
		"nextNodeIds": ["node_next"],
		"audienceFilter": {
			"match": "all",
			"conditions": [
				{"type": "property", "key": "plan", "operator": "equals", "value": "pro"}
			]
		},
		"audienceSegmentId": "seg_abc"
	}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	node, err := client.GetWorkflowNode("wf_1", "node_af")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node.AudienceFilter == nil {
		t.Fatal("AudienceFilter (node) is nil")
	}
	if node.AudienceFilter.AudienceSegmentID != "seg_abc" {
		t.Errorf("AudienceSegmentID = %q, want seg_abc", node.AudienceFilter.AudienceSegmentID)
	}
	af := node.AudienceFilter.AudienceFilter
	if af == nil {
		t.Fatal("AudienceFilter.AudienceFilter is nil")
	}
	if af.Match != "all" || len(af.Conditions) != 1 {
		t.Errorf("audienceFilter = %+v", af)
	}
	if af.Conditions[0].Property == nil || af.Conditions[0].Property.Key != "plan" {
		t.Errorf("audienceFilter.Conditions[0] = %+v", af.Conditions[0])
	}
}

func TestWorkflowNode_MarshalRoundTrip(t *testing.T) {
	original := WorkflowNode{
		TypeName: WorkflowNodeTypeTimerAction,
		TimerAction: &TimerActionWorkflowNode{
			ID:          "n1",
			WorkflowID:  "wf_1",
			NextNodeIDs: []string{"n2"},
			Amount:      15,
			Unit:        WorkflowTimerUnitMinutes,
		},
	}
	raw, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"typeName":"TimerAction"`) {
		t.Errorf("Marshal missing typeName: %s", raw)
	}

	var round WorkflowNode
	if err := json.Unmarshal(raw, &round); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if round.TypeName != original.TypeName {
		t.Errorf("round-trip TypeName = %q, want %q", round.TypeName, original.TypeName)
	}
	if round.TimerAction == nil || round.TimerAction.Unit != WorkflowTimerUnitMinutes {
		t.Errorf("round-trip TimerAction = %+v", round.TimerAction)
	}
}

func TestSimplifiedWorkflowNode_MarshalRoundTrip(t *testing.T) {
	original := SimplifiedWorkflowNode{
		TypeName: WorkflowNodeTypeEventTrigger,
		EventTrigger: &SimplifiedEventTriggerWorkflowNode{
			NextNodeIDs: []string{"n2"},
			EventName:   "signed_up",
			ReEligible:  ptr(true),
		},
	}
	raw, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"typeName":"EventTrigger"`) {
		t.Errorf("Marshal missing typeName: %s", raw)
	}

	var round SimplifiedWorkflowNode
	if err := json.Unmarshal(raw, &round); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if round.EventTrigger == nil || round.EventTrigger.EventName != "signed_up" {
		t.Errorf("round-trip EventTrigger = %+v", round.EventTrigger)
	}
}

func TestWorkflowContactPropertyValue_RoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		value WorkflowContactPropertyValue
		want  string
	}{
		{"string", WorkflowContactPropertyValue{String: ptr("pro")}, `"pro"`},
		{"number", WorkflowContactPropertyValue{Number: ptr(5.5)}, `5.5`},
		{"bool true", WorkflowContactPropertyValue{Bool: ptr(true)}, `true`},
		{"bool false", WorkflowContactPropertyValue{Bool: ptr(false)}, `false`},
		{"null", WorkflowContactPropertyValue{}, `null`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if string(raw) != tt.want {
				t.Errorf("Marshal = %s, want %s", raw, tt.want)
			}
			var round WorkflowContactPropertyValue
			if err := json.Unmarshal(raw, &round); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
		})
	}
}
