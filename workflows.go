package loops

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// WorkflowSummary is an entry in the [Client.ListWorkflows] response.
type WorkflowSummary struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// SimplifiedWorkflow is returned by [Client.GetWorkflow]. Nodes is keyed by
// node ID; entry shapes are discriminated by their TypeName.
type SimplifiedWorkflow struct {
	ID            string                            `json:"id"`
	Name          string                            `json:"name,omitempty"`
	Description   string                            `json:"description,omitempty"`
	Emoji         string                            `json:"emoji,omitempty"`
	MailingListID *string                           `json:"mailingListId"`
	RootNodeID    *string                           `json:"rootNodeId"`
	Nodes         map[string]SimplifiedWorkflowNode `json:"nodes"`
}

// Workflow node typeName discriminator values, used in [WorkflowNode] and
// [SimplifiedWorkflowNode].
const (
	WorkflowNodeTypeSignupTrigger          = "SignupTrigger"
	WorkflowNodeTypeEventTrigger           = "EventTrigger"
	WorkflowNodeTypeContactPropertyTrigger = "ContactPropertyTrigger"
	WorkflowNodeTypeAddToListTrigger       = "AddToListTrigger"
	WorkflowNodeTypeBlankTrigger           = "BlankTrigger"
	WorkflowNodeTypeAudienceFilter         = "AudienceFilter"
	WorkflowNodeTypeTimerAction            = "TimerAction"
	WorkflowNodeTypeSendEmailAction        = "SendEmailAction"
	WorkflowNodeTypeExitAction             = "ExitAction"
	WorkflowNodeTypeBranchNode             = "BranchNode"
	WorkflowNodeTypeExperimentBranchNode   = "ExperimentBranchNode"
	WorkflowNodeTypeVariantNode            = "VariantNode"
)

// WorkflowTimerUnit is the unit of a [TimerActionWorkflowNode] delay.
type WorkflowTimerUnit string

const (
	WorkflowTimerUnitSeconds WorkflowTimerUnit = "s"
	WorkflowTimerUnitMinutes WorkflowTimerUnit = "m"
	WorkflowTimerUnitHours   WorkflowTimerUnit = "h"
	WorkflowTimerUnitDays    WorkflowTimerUnit = "d"
)

// WorkflowExperimentType is the kind of experiment driving an
// [ExperimentBranchWorkflowNode].
type WorkflowExperimentType string

const (
	WorkflowExperimentTypeWebhook   WorkflowExperimentType = "webhook"
	WorkflowExperimentTypeAutosplit WorkflowExperimentType = "autosplit"
)

// WorkflowEventProperty describes one event property surfaced by an
// [EventTriggerWorkflowNode]. Type is one of "string", "number", "boolean",
// or "date".
type WorkflowEventProperty struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// WorkflowContactPropertyQuery is the contact-property predicate of a
// [ContactPropertyTriggerWorkflowNode] or
// [SimplifiedContactPropertyTriggerWorkflowNode].
type WorkflowContactPropertyQuery struct {
	Key string                            `json:"key"`
	Is  WorkflowContactPropertyComparison `json:"is"`
	Was WorkflowContactPropertyComparison `json:"was"`
}

// WorkflowContactPropertyComparison is a single comparison in a
// [WorkflowContactPropertyQuery].
type WorkflowContactPropertyComparison struct {
	Value    WorkflowContactPropertyValue `json:"value"`
	Operator string                       `json:"operator"`
}

// WorkflowContactPropertyValue holds the right-hand side of a comparison;
// exactly one of String, Number, or Bool is set depending on the operator.
// Value-less operators may leave all fields zero.
type WorkflowContactPropertyValue struct {
	String *string
	Number *float64
	Bool   *bool
}

// MarshalJSON serializes the value as a string, number, or boolean.
func (v WorkflowContactPropertyValue) MarshalJSON() ([]byte, error) {
	switch {
	case v.String != nil:
		return json.Marshal(*v.String)
	case v.Number != nil:
		return json.Marshal(*v.Number)
	case v.Bool != nil:
		return json.Marshal(*v.Bool)
	}
	return []byte("null"), nil
}

// UnmarshalJSON decodes a string, number, or boolean into the value.
func (v *WorkflowContactPropertyValue) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	switch data[0] {
	case '"':
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		v.String = &s
		return nil
	case 't', 'f':
		var b bool
		if err := json.Unmarshal(data, &b); err != nil {
			return err
		}
		v.Bool = &b
		return nil
	default:
		var n float64
		if err := json.Unmarshal(data, &n); err != nil {
			return err
		}
		v.Number = &n
		return nil
	}
}

// SimplifiedWorkflowNode is one entry in [SimplifiedWorkflow.Nodes]. Exactly
// one variant pointer is set; TypeName is the discriminator and matches the
// active variant (see the WorkflowNodeType* constants).
type SimplifiedWorkflowNode struct {
	TypeName               string
	SignupTrigger          *SimplifiedSignupTriggerWorkflowNode
	EventTrigger           *SimplifiedEventTriggerWorkflowNode
	ContactPropertyTrigger *SimplifiedContactPropertyTriggerWorkflowNode
	AddToListTrigger       *SimplifiedAddToListTriggerWorkflowNode
	BlankTrigger           *SimplifiedBlankTriggerWorkflowNode
	AudienceFilter         *SimplifiedAudienceFilterWorkflowNode
	TimerAction            *SimplifiedTimerActionWorkflowNode
	SendEmailAction        *SimplifiedSendEmailActionWorkflowNode
	ExitAction             *SimplifiedExitActionWorkflowNode
	BranchNode             *SimplifiedBranchWorkflowNode
	ExperimentBranchNode   *SimplifiedExperimentBranchWorkflowNode
	VariantNode            *SimplifiedVariantWorkflowNode
}

// SimplifiedSignupTriggerWorkflowNode is the SignupTrigger variant of
// [SimplifiedWorkflowNode].
type SimplifiedSignupTriggerWorkflowNode struct {
	NextNodeIDs []string `json:"nextNodeIds"`
}

// SimplifiedEventTriggerWorkflowNode is the EventTrigger variant of
// [SimplifiedWorkflowNode].
type SimplifiedEventTriggerWorkflowNode struct {
	NextNodeIDs []string `json:"nextNodeIds"`
	EventName   string   `json:"eventName,omitempty"`
	ReEligible  *bool    `json:"reEligible,omitempty"`
}

// SimplifiedContactPropertyTriggerWorkflowNode is the ContactPropertyTrigger
// variant of [SimplifiedWorkflowNode].
type SimplifiedContactPropertyTriggerWorkflowNode struct {
	NextNodeIDs          []string                      `json:"nextNodeIds"`
	ContactPropertyQuery *WorkflowContactPropertyQuery `json:"contactPropertyQuery,omitempty"`
	ReEligible           *bool                         `json:"reEligible,omitempty"`
}

// SimplifiedAddToListTriggerWorkflowNode is the AddToListTrigger variant of
// [SimplifiedWorkflowNode].
type SimplifiedAddToListTriggerWorkflowNode struct {
	NextNodeIDs []string `json:"nextNodeIds"`
	MailingList string   `json:"mailingList,omitempty"`
	ReEligible  *bool    `json:"reEligible,omitempty"`
}

// SimplifiedBlankTriggerWorkflowNode is the BlankTrigger variant of
// [SimplifiedWorkflowNode].
type SimplifiedBlankTriggerWorkflowNode struct {
	NextNodeIDs []string `json:"nextNodeIds"`
}

// SimplifiedAudienceFilterWorkflowNode is the AudienceFilter variant of
// [SimplifiedWorkflowNode].
type SimplifiedAudienceFilterWorkflowNode struct {
	NextNodeIDs []string `json:"nextNodeIds"`
}

// SimplifiedTimerActionWorkflowNode is the TimerAction variant of
// [SimplifiedWorkflowNode].
type SimplifiedTimerActionWorkflowNode struct {
	NextNodeIDs []string          `json:"nextNodeIds"`
	Amount      *float64          `json:"amount,omitempty"`
	Unit        WorkflowTimerUnit `json:"unit,omitempty"`
}

// SimplifiedSendEmailActionWorkflowNode is the SendEmailAction variant of
// [SimplifiedWorkflowNode].
type SimplifiedSendEmailActionWorkflowNode struct {
	NextNodeIDs    []string `json:"nextNodeIds"`
	EmailMessageID string   `json:"emailMessageId,omitempty"`
	Subject        string   `json:"subject,omitempty"`
}

// SimplifiedExitActionWorkflowNode is the ExitAction variant of
// [SimplifiedWorkflowNode].
type SimplifiedExitActionWorkflowNode struct {
	NextNodeIDs []string `json:"nextNodeIds"`
}

// SimplifiedBranchWorkflowNode is the BranchNode variant of
// [SimplifiedWorkflowNode].
type SimplifiedBranchWorkflowNode struct {
	NextNodeIDs []string `json:"nextNodeIds"`
}

// SimplifiedExperimentBranchWorkflowNode is the ExperimentBranchNode variant
// of [SimplifiedWorkflowNode].
type SimplifiedExperimentBranchWorkflowNode struct {
	NextNodeIDs    []string               `json:"nextNodeIds"`
	SamplingRate   *float64               `json:"samplingRate,omitempty"`
	URL            string                 `json:"url,omitempty"`
	ExperimentID   string                 `json:"experimentId,omitempty"`
	ExperimentType WorkflowExperimentType `json:"experimentType,omitempty"`
}

// SimplifiedVariantWorkflowNode is the VariantNode variant of
// [SimplifiedWorkflowNode].
type SimplifiedVariantWorkflowNode struct {
	NextNodeIDs []string `json:"nextNodeIds"`
	VariantID   string   `json:"variantId,omitempty"`
	IsControl   *bool    `json:"isControl,omitempty"`
}

// MarshalJSON encodes the active variant inline with a "typeName"
// discriminator.
func (n SimplifiedWorkflowNode) MarshalJSON() ([]byte, error) {
	inner, err := pickSimplifiedWorkflowNode(n)
	if err != nil {
		return nil, err
	}
	return marshalDiscriminated(n.TypeName, inner)
}

// UnmarshalJSON dispatches on "typeName" and decodes the matching variant.
func (n *SimplifiedWorkflowNode) UnmarshalJSON(data []byte) error {
	var head struct {
		TypeName string `json:"typeName"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return err
	}
	n.TypeName = head.TypeName
	switch head.TypeName {
	case WorkflowNodeTypeSignupTrigger:
		var v SimplifiedSignupTriggerWorkflowNode
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		n.SignupTrigger = &v
	case WorkflowNodeTypeEventTrigger:
		var v SimplifiedEventTriggerWorkflowNode
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		n.EventTrigger = &v
	case WorkflowNodeTypeContactPropertyTrigger:
		var v SimplifiedContactPropertyTriggerWorkflowNode
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		n.ContactPropertyTrigger = &v
	case WorkflowNodeTypeAddToListTrigger:
		var v SimplifiedAddToListTriggerWorkflowNode
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		n.AddToListTrigger = &v
	case WorkflowNodeTypeBlankTrigger:
		var v SimplifiedBlankTriggerWorkflowNode
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		n.BlankTrigger = &v
	case WorkflowNodeTypeAudienceFilter:
		var v SimplifiedAudienceFilterWorkflowNode
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		n.AudienceFilter = &v
	case WorkflowNodeTypeTimerAction:
		var v SimplifiedTimerActionWorkflowNode
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		n.TimerAction = &v
	case WorkflowNodeTypeSendEmailAction:
		var v SimplifiedSendEmailActionWorkflowNode
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		n.SendEmailAction = &v
	case WorkflowNodeTypeExitAction:
		var v SimplifiedExitActionWorkflowNode
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		n.ExitAction = &v
	case WorkflowNodeTypeBranchNode:
		var v SimplifiedBranchWorkflowNode
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		n.BranchNode = &v
	case WorkflowNodeTypeExperimentBranchNode:
		var v SimplifiedExperimentBranchWorkflowNode
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		n.ExperimentBranchNode = &v
	case WorkflowNodeTypeVariantNode:
		var v SimplifiedVariantWorkflowNode
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		n.VariantNode = &v
	default:
		return fmt.Errorf("workflow node: unknown typeName %q", head.TypeName)
	}
	return nil
}

func pickSimplifiedWorkflowNode(n SimplifiedWorkflowNode) (any, error) {
	switch n.TypeName {
	case WorkflowNodeTypeSignupTrigger:
		return n.SignupTrigger, nil
	case WorkflowNodeTypeEventTrigger:
		return n.EventTrigger, nil
	case WorkflowNodeTypeContactPropertyTrigger:
		return n.ContactPropertyTrigger, nil
	case WorkflowNodeTypeAddToListTrigger:
		return n.AddToListTrigger, nil
	case WorkflowNodeTypeBlankTrigger:
		return n.BlankTrigger, nil
	case WorkflowNodeTypeAudienceFilter:
		return n.AudienceFilter, nil
	case WorkflowNodeTypeTimerAction:
		return n.TimerAction, nil
	case WorkflowNodeTypeSendEmailAction:
		return n.SendEmailAction, nil
	case WorkflowNodeTypeExitAction:
		return n.ExitAction, nil
	case WorkflowNodeTypeBranchNode:
		return n.BranchNode, nil
	case WorkflowNodeTypeExperimentBranchNode:
		return n.ExperimentBranchNode, nil
	case WorkflowNodeTypeVariantNode:
		return n.VariantNode, nil
	}
	return nil, fmt.Errorf("workflow node: unknown typeName %q", n.TypeName)
}

// WorkflowNode is returned by [Client.GetWorkflowNode]. Exactly one variant
// pointer is set; TypeName is the discriminator and matches the active
// variant (see the WorkflowNodeType* constants).
type WorkflowNode struct {
	TypeName               string
	SignupTrigger          *SignupTriggerWorkflowNode
	EventTrigger           *EventTriggerWorkflowNode
	ContactPropertyTrigger *ContactPropertyTriggerWorkflowNode
	AddToListTrigger       *AddToListTriggerWorkflowNode
	BlankTrigger           *BlankTriggerWorkflowNode
	AudienceFilter         *AudienceFilterWorkflowNode
	TimerAction            *TimerActionWorkflowNode
	SendEmailAction        *SendEmailActionWorkflowNode
	ExitAction             *ExitActionWorkflowNode
	BranchNode             *BranchWorkflowNode
	ExperimentBranchNode   *ExperimentBranchWorkflowNode
	VariantNode            *VariantWorkflowNode
}

// SignupTriggerWorkflowNode is the SignupTrigger variant of [WorkflowNode].
type SignupTriggerWorkflowNode struct {
	ID          string   `json:"id"`
	WorkflowID  string   `json:"workflowId"`
	NextNodeIDs []string `json:"nextNodeIds"`
}

// EventTriggerWorkflowNode is the EventTrigger variant of [WorkflowNode].
// EventName is nullable; EventProperties may be empty.
type EventTriggerWorkflowNode struct {
	ID              string                  `json:"id"`
	WorkflowID      string                  `json:"workflowId"`
	NextNodeIDs     []string                `json:"nextNodeIds"`
	EventName       *string                 `json:"eventName"`
	EventProperties []WorkflowEventProperty `json:"eventProperties,omitempty"`
	ReEligible      bool                    `json:"reEligible"`
}

// ContactPropertyTriggerWorkflowNode is the ContactPropertyTrigger variant
// of [WorkflowNode]. ContactPropertyQuery is required by the spec but may be
// null.
type ContactPropertyTriggerWorkflowNode struct {
	ID                   string                        `json:"id"`
	WorkflowID           string                        `json:"workflowId"`
	NextNodeIDs          []string                      `json:"nextNodeIds"`
	ContactPropertyQuery *WorkflowContactPropertyQuery `json:"contactPropertyQuery"`
	ReEligible           bool                          `json:"reEligible"`
}

// AddToListTriggerWorkflowNode is the AddToListTrigger variant of
// [WorkflowNode].
type AddToListTriggerWorkflowNode struct {
	ID          string   `json:"id"`
	WorkflowID  string   `json:"workflowId"`
	NextNodeIDs []string `json:"nextNodeIds"`
	ReEligible  bool     `json:"reEligible"`
}

// BlankTriggerWorkflowNode is the BlankTrigger variant of [WorkflowNode].
type BlankTriggerWorkflowNode struct {
	ID          string   `json:"id"`
	WorkflowID  string   `json:"workflowId"`
	NextNodeIDs []string `json:"nextNodeIds"`
}

// AudienceFilterWorkflowNode is the AudienceFilter variant of [WorkflowNode].
// AudienceFilter reuses the package-level [AudienceFilter] type.
type AudienceFilterWorkflowNode struct {
	ID                string          `json:"id"`
	WorkflowID        string          `json:"workflowId"`
	NextNodeIDs       []string        `json:"nextNodeIds"`
	AudienceFilter    *AudienceFilter `json:"audienceFilter,omitempty"`
	AudienceSegmentID string          `json:"audienceSegmentId,omitempty"`
}

// TimerActionWorkflowNode is the TimerAction variant of [WorkflowNode].
type TimerActionWorkflowNode struct {
	ID          string            `json:"id"`
	WorkflowID  string            `json:"workflowId"`
	NextNodeIDs []string          `json:"nextNodeIds"`
	Amount      float64           `json:"amount"`
	Unit        WorkflowTimerUnit `json:"unit"`
}

// SendEmailActionWorkflowNode is the SendEmailAction variant of
// [WorkflowNode]. The full variant does not include emailMessageId (only the
// simplified variant does).
type SendEmailActionWorkflowNode struct {
	ID          string   `json:"id"`
	WorkflowID  string   `json:"workflowId"`
	NextNodeIDs []string `json:"nextNodeIds"`
	Subject     string   `json:"subject,omitempty"`
}

// ExitActionWorkflowNode is the ExitAction variant of [WorkflowNode].
type ExitActionWorkflowNode struct {
	ID          string   `json:"id"`
	WorkflowID  string   `json:"workflowId"`
	NextNodeIDs []string `json:"nextNodeIds"`
}

// BranchWorkflowNode is the BranchNode variant of [WorkflowNode].
type BranchWorkflowNode struct {
	ID           string   `json:"id"`
	WorkflowID   string   `json:"workflowId"`
	NextNodeIDs  []string `json:"nextNodeIds"`
	EvalStrategy string   `json:"evalStrategy,omitempty"`
}

// ExperimentBranchWorkflowNode is the ExperimentBranchNode variant of
// [WorkflowNode].
type ExperimentBranchWorkflowNode struct {
	ID             string                 `json:"id"`
	WorkflowID     string                 `json:"workflowId"`
	NextNodeIDs    []string               `json:"nextNodeIds"`
	SamplingRate   float64                `json:"samplingRate"`
	URL            string                 `json:"url,omitempty"`
	ExperimentID   string                 `json:"experimentId,omitempty"`
	ExperimentType WorkflowExperimentType `json:"experimentType"`
}

// VariantWorkflowNode is the VariantNode variant of [WorkflowNode].
type VariantWorkflowNode struct {
	ID          string   `json:"id"`
	WorkflowID  string   `json:"workflowId"`
	NextNodeIDs []string `json:"nextNodeIds"`
	VariantID   string   `json:"variantId,omitempty"`
	IsControl   *bool    `json:"isControl,omitempty"`
}

// MarshalJSON encodes the active variant inline with a "typeName"
// discriminator.
func (n WorkflowNode) MarshalJSON() ([]byte, error) {
	inner, err := pickWorkflowNode(n)
	if err != nil {
		return nil, err
	}
	return marshalDiscriminated(n.TypeName, inner)
}

// UnmarshalJSON dispatches on "typeName" and decodes the matching variant.
func (n *WorkflowNode) UnmarshalJSON(data []byte) error {
	var head struct {
		TypeName string `json:"typeName"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return err
	}
	n.TypeName = head.TypeName
	switch head.TypeName {
	case WorkflowNodeTypeSignupTrigger:
		var v SignupTriggerWorkflowNode
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		n.SignupTrigger = &v
	case WorkflowNodeTypeEventTrigger:
		var v EventTriggerWorkflowNode
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		n.EventTrigger = &v
	case WorkflowNodeTypeContactPropertyTrigger:
		var v ContactPropertyTriggerWorkflowNode
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		n.ContactPropertyTrigger = &v
	case WorkflowNodeTypeAddToListTrigger:
		var v AddToListTriggerWorkflowNode
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		n.AddToListTrigger = &v
	case WorkflowNodeTypeBlankTrigger:
		var v BlankTriggerWorkflowNode
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		n.BlankTrigger = &v
	case WorkflowNodeTypeAudienceFilter:
		var v AudienceFilterWorkflowNode
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		n.AudienceFilter = &v
	case WorkflowNodeTypeTimerAction:
		var v TimerActionWorkflowNode
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		n.TimerAction = &v
	case WorkflowNodeTypeSendEmailAction:
		var v SendEmailActionWorkflowNode
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		n.SendEmailAction = &v
	case WorkflowNodeTypeExitAction:
		var v ExitActionWorkflowNode
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		n.ExitAction = &v
	case WorkflowNodeTypeBranchNode:
		var v BranchWorkflowNode
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		n.BranchNode = &v
	case WorkflowNodeTypeExperimentBranchNode:
		var v ExperimentBranchWorkflowNode
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		n.ExperimentBranchNode = &v
	case WorkflowNodeTypeVariantNode:
		var v VariantWorkflowNode
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		n.VariantNode = &v
	default:
		return fmt.Errorf("workflow node: unknown typeName %q", head.TypeName)
	}
	return nil
}

func pickWorkflowNode(n WorkflowNode) (any, error) {
	switch n.TypeName {
	case WorkflowNodeTypeSignupTrigger:
		return n.SignupTrigger, nil
	case WorkflowNodeTypeEventTrigger:
		return n.EventTrigger, nil
	case WorkflowNodeTypeContactPropertyTrigger:
		return n.ContactPropertyTrigger, nil
	case WorkflowNodeTypeAddToListTrigger:
		return n.AddToListTrigger, nil
	case WorkflowNodeTypeBlankTrigger:
		return n.BlankTrigger, nil
	case WorkflowNodeTypeAudienceFilter:
		return n.AudienceFilter, nil
	case WorkflowNodeTypeTimerAction:
		return n.TimerAction, nil
	case WorkflowNodeTypeSendEmailAction:
		return n.SendEmailAction, nil
	case WorkflowNodeTypeExitAction:
		return n.ExitAction, nil
	case WorkflowNodeTypeBranchNode:
		return n.BranchNode, nil
	case WorkflowNodeTypeExperimentBranchNode:
		return n.ExperimentBranchNode, nil
	case WorkflowNodeTypeVariantNode:
		return n.VariantNode, nil
	}
	return nil, fmt.Errorf("workflow node: unknown typeName %q", n.TypeName)
}

func marshalDiscriminated(typeName string, inner any) ([]byte, error) {
	if typeName == "" {
		return nil, fmt.Errorf("workflow node: typeName is empty")
	}
	if inner == nil {
		return nil, fmt.Errorf("workflow node: %s variant is nil", typeName)
	}
	raw, err := json.Marshal(inner)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	tn, _ := json.Marshal(typeName)
	fields["typeName"] = tn
	return json.Marshal(fields)
}

// ListWorkflows returns a single page of workflow summaries along with
// pagination information. To iterate every page, use [Paginate].
func (c *Client) ListWorkflows(params PaginationParams) ([]WorkflowSummary, *Pagination, error) {
	q := url.Values{}
	if params.PerPage != "" {
		q.Set("perPage", params.PerPage)
	}
	if params.Cursor != "" {
		q.Set("cursor", params.Cursor)
	}

	path := "/workflows"
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
		Pagination Pagination        `json:"pagination"`
		Data       []WorkflowSummary `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Data, &result.Pagination, nil
}

// GetWorkflow returns the simplified workflow graph identified by id.
func (c *Client) GetWorkflow(id string) (*SimplifiedWorkflow, error) {
	req, err := c.newRequest(http.MethodGet, "/workflows/"+id, nil)
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

	var result SimplifiedWorkflow
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetWorkflowNode returns the detailed data for a single workflow node.
func (c *Client) GetWorkflowNode(workflowID, nodeID string) (*WorkflowNode, error) {
	req, err := c.newRequest(http.MethodGet, "/workflows/"+workflowID+"/nodes/"+nodeID, nil)
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

	var result WorkflowNode
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
