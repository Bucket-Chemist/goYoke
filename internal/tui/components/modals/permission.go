package modals

import (
	"encoding/json"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// ---------------------------------------------------------------------------
// PermissionFlowType
// ---------------------------------------------------------------------------

// PermissionFlowType enumerates the distinct permission-flow patterns.
type PermissionFlowType int

const (
	// FlowEnterPlan presents a single Confirm modal ("Enter plan mode?").
	FlowEnterPlan PermissionFlowType = iota

	// FlowExitPlan presents a Select modal with Approve/Request Changes/Reject.
	// If the user selects "Request Changes" a second Input modal is shown for
	// feedback text.
	FlowExitPlan

	// FlowAskUser presents a single Ask modal with the supplied options.
	FlowAskUser

	// FlowConfirm presents a single Confirm modal.
	FlowConfirm

	// FlowInput presents a single Input modal.
	FlowInput

	// FlowSelect presents a single Select modal with the supplied options.
	FlowSelect

	// FlowToolPermission presents a three-option Select modal: Allow / Deny /
	// Allow for Session. Used by the permission gate hook to request user
	// approval for tool invocations.
	FlowToolPermission
)

// ---------------------------------------------------------------------------
// PermissionFlow
// ---------------------------------------------------------------------------

// PermissionFlow tracks the state of a multi-step permission interaction that
// is linked to a single MCP request ID.
type PermissionFlow struct {
	// RequestID is the IPC request identifier used to route the final response.
	RequestID string

	// FlowType determines how many steps the flow has and how responses are
	// combined into the final PermissionResult value.
	FlowType PermissionFlowType

	// Steps is the total number of modal steps in this flow.
	Steps int

	// Current is the 0-based index of the step currently being shown.
	Current int

	// Responses accumulates the ModalResponse from each completed step.
	Responses []ModalResponse

	// message is the user-visible prompt text passed in from the bridge.
	message string

	// options is the list of selectable options (nil for free-text flows).
	options []string
}

// ---------------------------------------------------------------------------
// PermissionResult
// ---------------------------------------------------------------------------

// PermissionResult is the final answer to return to the MCP server via bridge.
// Value is the serialised response string — its exact format depends on the
// FlowType:
//
//   - FlowEnterPlan / FlowConfirm / FlowAskUser / FlowInput / FlowSelect:
//     Value is the plain string value from the modal response.
//   - FlowExitPlan:
//     Value is a JSON object: {"decision":"approve|changes|reject","feedback":"…"}
//
// Cancelled is true when the user dismissed without completing the flow.
type PermissionResult struct {
	// RequestID links this result back to the originating IPC request.
	RequestID string
	// Value is the serialised answer string.
	Value string
	// Cancelled is true when the user dismissed the modal without answering.
	Cancelled bool
}

// exitPlanResult is the JSON shape for FlowExitPlan responses.
type exitPlanResult struct {
	Decision string `json:"decision"`
	Feedback string `json:"feedback,omitempty"`
}

// ---------------------------------------------------------------------------
// PermissionHandler
// ---------------------------------------------------------------------------

// PermissionHandler orchestrates multi-step permission flows on top of the
// ModalQueue.  Each MCP request maps to exactly one PermissionFlow; the
// handler tracks pending flows and routes ModalResponseMsg values back to the
// correct flow.
//
// The zero value is not usable; use NewPermissionHandler instead.
type PermissionHandler struct {
	queue              *ModalQueue
	pending            map[string]*PermissionFlow
	completedPermGates map[string]struct{}
}

// NewPermissionHandler creates a PermissionHandler backed by the given queue.
// queue must be non-nil; its lifetime must exceed that of the handler.
func NewPermissionHandler(queue *ModalQueue) *PermissionHandler {
	return &PermissionHandler{
		queue:              queue,
		pending:            make(map[string]*PermissionFlow),
		completedPermGates: make(map[string]struct{}),
	}
}

// IsPending reports whether a flow for the given requestID is still in
// progress (i.e. at least one modal step has been enqueued but not yet
// resolved).
func (h *PermissionHandler) IsPending(requestID string) bool {
	_, ok := h.pending[requestID]
	return ok
}

// ---------------------------------------------------------------------------
// HandleBridgeRequest
// ---------------------------------------------------------------------------

// HandleBridgeRequest translates a BridgeModalRequest into one or more queued
// ModalRequests and starts the first step.  It returns a tea.Cmd that
// activates the queue if no modal is currently shown.
//
// Callers should call queue.IsActive() after HandleBridgeRequest returns to
// decide whether to show the modal overlay; the returned tea.Cmd takes care of
// queue activation.
func (h *PermissionHandler) HandleBridgeRequest(requestID, message string, options []string) tea.Cmd {
	flow := h.buildFlow(requestID, message, options)
	h.pending[requestID] = flow
	return h.enqueueStep(flow)
}

// HandlePermGateRequest starts a tool-permission flow. The modal presents
// three choices: Allow, Deny, Allow for Session. The flow resolves in a
// single step.
func (h *PermissionHandler) HandlePermGateRequest(requestID, message string, options []string, timeoutMS int) tea.Cmd {
	flow := &PermissionFlow{
		RequestID: requestID,
		FlowType:  FlowToolPermission,
		Steps:     1,
		message:   message,
		options:   options,
	}
	h.pending[requestID] = flow
	return h.enqueueStep(flow)
}

// WasPermGateFlow reports whether the given requestID was completed as a
// FlowToolPermission flow. The check is one-time: the record is deleted after
// the first successful lookup.
func (h *PermissionHandler) WasPermGateFlow(requestID string) bool {
	_, ok := h.completedPermGates[requestID]
	if ok {
		delete(h.completedPermGates, requestID)
	}
	return ok
}

// buildFlow selects the correct FlowType from the bridge message content.
// Heuristics:
//   - options == ["Approve Plan", "Request Changes", "Reject Plan"] → FlowExitPlan
//   - options == [] / nil and message contains "Enter plan mode" → FlowEnterPlan
//   - options == [] / nil → FlowInput (free-text)
//   - len(options) == 2 and options[0]=="Yes" / options[0]=="No" → FlowConfirm
//   - len(options) > 0 otherwise → FlowAskUser
func (h *PermissionHandler) buildFlow(requestID, message string, options []string) *PermissionFlow {
	flow := &PermissionFlow{
		RequestID: requestID,
		message:   message,
		options:   options,
	}

	switch {
	case isExitPlanOptions(options):
		flow.FlowType = FlowExitPlan
		flow.Steps = 1 // may grow to 2 if "Request Changes" is chosen

	case len(options) == 0:
		// No options → free-text input.
		flow.FlowType = FlowInput
		flow.Steps = 1

	case len(options) == 2 && options[0] == "Yes" && options[1] == "No":
		flow.FlowType = FlowConfirm
		flow.Steps = 1

	default:
		flow.FlowType = FlowAskUser
		flow.Steps = 1
	}

	return flow
}

// isExitPlanOptions returns true when opts matches the canonical ExitPlanMode
// option set.
func isExitPlanOptions(opts []string) bool {
	if len(opts) != 3 {
		return false
	}
	return opts[0] == "Approve Plan" &&
		opts[1] == "Request Changes" &&
		opts[2] == "Reject Plan"
}

// enqueueStep pushes the first (or next) modal for the given flow into the
// queue and returns a tea.Cmd that activates the queue when it was idle.
func (h *PermissionHandler) enqueueStep(flow *PermissionFlow) tea.Cmd {
	req := h.buildRequest(flow)
	h.queue.Push(req)

	// Activate the queue only when it was idle before this Push.
	if !h.queue.IsActive() {
		h.queue.Activate()
	}

	// Return a no-op Cmd; the queue view will be rendered by AppModel.View.
	return nil
}

// buildRequest creates the ModalRequest for the current step of a flow.
func (h *PermissionHandler) buildRequest(flow *PermissionFlow) ModalRequest {
	switch flow.FlowType {
	case FlowEnterPlan:
		return ModalRequest{
			ID:      flow.RequestID,
			Type:    Confirm,
			Header:  "Enter Plan Mode",
			Message: flow.message,
		}

	case FlowExitPlan:
		if flow.Current == 0 {
			return ModalRequest{
				ID:      flow.RequestID,
				Type:    Select,
				Header:  "Plan Review",
				Message: flow.message,
				Options: flow.options,
			}
		}
		// Step 1: feedback input after "Request Changes"
		return ModalRequest{
			ID:      stepID(flow.RequestID, 1),
			Type:    Input,
			Header:  "Request Changes",
			Message: "Describe the changes you'd like:",
		}

	case FlowAskUser:
		return ModalRequest{
			ID:      flow.RequestID,
			Type:    Ask,
			Header:  "Question",
			Message: flow.message,
			Options: flow.options,
		}

	case FlowConfirm:
		return ModalRequest{
			ID:      flow.RequestID,
			Type:    Confirm,
			Header:  "Confirm",
			Message: flow.message,
		}

	case FlowInput:
		return ModalRequest{
			ID:      flow.RequestID,
			Type:    Input,
			Header:  "Input Required",
			Message: flow.message,
		}

	case FlowSelect:
		return ModalRequest{
			ID:      flow.RequestID,
			Type:    Select,
			Header:  "Select",
			Message: flow.message,
			Options: flow.options,
		}

	case FlowToolPermission:
		return ModalRequest{
			ID:      flow.RequestID,
			Type:    Select,
			Header:  "Tool Permission Required",
			Message: flow.message,
			Options: flow.options,
		}

	default:
		return ModalRequest{
			ID:      flow.RequestID,
			Type:    Input,
			Header:  "Input Required",
			Message: flow.message,
		}
	}
}

// stepID returns a derived request ID for a subsequent step within a flow.
// Only one level of nesting is currently needed.
func stepID(base string, step int) string {
	return fmt.Sprintf("%s:step%d", base, step)
}

// ---------------------------------------------------------------------------
// HandleResponse
// ---------------------------------------------------------------------------

// HandleResponse processes a ModalResponseMsg produced by the queue.  It
// advances the flow associated with the response:
//
//   - For single-step flows it returns a *PermissionResult immediately.
//   - For FlowExitPlan with "Request Changes" it enqueues a second Input step
//     and returns (nil, cmd) so the caller can present the follow-up modal.
//   - When the second step of FlowExitPlan completes it returns the combined
//     result.
//
// The caller is responsible for calling queue.Resolve() / advancing the queue
// before passing the ModalResponseMsg here; HandleResponse only inspects the
// response and manages flow state, it does not touch the queue directly.
func (h *PermissionHandler) HandleResponse(msg ModalResponseMsg) (*PermissionResult, tea.Cmd) {
	requestID := rootRequestID(msg.RequestID)
	flow, ok := h.pending[requestID]
	if !ok {
		// Unknown flow — ignore.
		return nil, nil
	}

	// Accumulate the response for this step.
	flow.Responses = append(flow.Responses, msg.Response)

	// Cancelled at any step terminates the whole flow.
	if msg.Response.Cancelled {
		delete(h.pending, requestID)
		return &PermissionResult{
			RequestID: requestID,
			Cancelled: true,
		}, nil
	}

	switch flow.FlowType {
	case FlowExitPlan:
		return h.handleExitPlanResponse(flow, requestID)
	case FlowToolPermission:
		// Single-step flow — record as a completed perm gate and return.
		delete(h.pending, requestID)
		h.completedPermGates[requestID] = struct{}{}
		return &PermissionResult{
			RequestID: requestID,
			Value:     msg.Response.Value,
		}, nil
	default:
		// All other single-step flows are complete.
		delete(h.pending, requestID)
		return &PermissionResult{
			RequestID: requestID,
			Value:     msg.Response.Value,
		}, nil
	}
}

// handleExitPlanResponse manages the one-or-two-step ExitPlan flow.
func (h *PermissionHandler) handleExitPlanResponse(flow *PermissionFlow, requestID string) (*PermissionResult, tea.Cmd) {
	switch flow.Current {
	case 0:
		// Step 0: selection from "Approve Plan / Request Changes / Reject Plan"
		decision := flow.Responses[0].Value
		if decision == "Request Changes" {
			// Transition to step 1 — feedback input.
			flow.Current = 1
			cmd := h.enqueueStep(flow)
			return nil, cmd
		}
		// Approve or Reject → flow is complete, no feedback needed.
		result, err := buildExitPlanResult(decision, "")
		if err != nil {
			// Fallback: return the raw decision value on JSON serialisation failure.
			delete(h.pending, requestID)
			return &PermissionResult{RequestID: requestID, Value: decision}, nil
		}
		delete(h.pending, requestID)
		return &PermissionResult{RequestID: requestID, Value: result}, nil

	case 1:
		// Step 1: feedback text input
		decision := flow.Responses[0].Value // from step 0
		feedback := flow.Responses[1].Value // from step 1 (this response)
		result, err := buildExitPlanResult(decision, feedback)
		if err != nil {
			delete(h.pending, requestID)
			return &PermissionResult{RequestID: requestID, Value: decision}, nil
		}
		delete(h.pending, requestID)
		return &PermissionResult{RequestID: requestID, Value: result}, nil

	default:
		// Unexpected step — treat as error and cancel.
		delete(h.pending, requestID)
		return &PermissionResult{RequestID: requestID, Cancelled: true}, nil
	}
}

// buildExitPlanResult serialises the exit-plan decision + optional feedback
// into the canonical JSON format returned to the MCP server.
func buildExitPlanResult(decision, feedback string) (string, error) {
	// Normalise the decision to lowercase for the MCP consumer.
	var d string
	switch decision {
	case "Approve Plan":
		d = "approve"
	case "Request Changes":
		d = "changes"
	case "Reject Plan":
		d = "reject"
	default:
		d = decision
	}

	payload := exitPlanResult{Decision: d, Feedback: feedback}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// rootRequestID strips the ":step<N>" suffix added by stepID so that step
// responses can be matched back to their parent flow.
func rootRequestID(id string) string {
	for i := len(id) - 1; i >= 0; i-- {
		if id[i] == ':' {
			// Check that the suffix is ":step<digits>".
			suffix := id[i:]
			if len(suffix) >= 6 && suffix[:5] == ":step" {
				allDigits := true
				for _, c := range suffix[5:] {
					if c < '0' || c > '9' {
						allDigits = false
						break
					}
				}
				if allDigits {
					return id[:i]
				}
			}
			break
		}
	}
	return id
}
