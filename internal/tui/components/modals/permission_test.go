package modals

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newTestHandler returns a PermissionHandler backed by a fresh ModalQueue.
func newTestHandler() (*PermissionHandler, *ModalQueue) {
	km := defaultKM()
	mq := NewModalQueue(km)
	return NewPermissionHandler(&mq), &mq
}

// simulateResponse builds a ModalResponseMsg as if the user selected value on
// the modal with the given requestID.
func simulateResponse(requestID, value string) ModalResponseMsg {
	return ModalResponseMsg{
		RequestID: requestID,
		Response: ModalResponse{
			Value:     value,
			Cancelled: false,
		},
	}
}

// simulateCancelled builds a cancelled ModalResponseMsg.
func simulateCancelled(requestID string) ModalResponseMsg {
	return ModalResponseMsg{
		RequestID: requestID,
		Response: ModalResponse{
			Cancelled: true,
		},
	}
}

// ---------------------------------------------------------------------------
// NewPermissionHandler
// ---------------------------------------------------------------------------

func TestNewPermissionHandler_NotNil(t *testing.T) {
	h, _ := newTestHandler()
	require.NotNil(t, h)
}

func TestNewPermissionHandler_NoPendingFlows(t *testing.T) {
	h, _ := newTestHandler()
	assert.False(t, h.IsPending("nonexistent"))
}

// ---------------------------------------------------------------------------
// isExitPlanOptions
// ---------------------------------------------------------------------------

func TestIsExitPlanOptions_Matches(t *testing.T) {
	opts := []string{"Approve Plan", "Request Changes", "Reject Plan"}
	assert.True(t, isExitPlanOptions(opts))
}

func TestIsExitPlanOptions_WrongCount(t *testing.T) {
	assert.False(t, isExitPlanOptions([]string{"Approve Plan", "Reject Plan"}))
}

func TestIsExitPlanOptions_WrongValues(t *testing.T) {
	assert.False(t, isExitPlanOptions([]string{"Approve", "Changes", "Reject"}))
}

func TestIsExitPlanOptions_Empty(t *testing.T) {
	assert.False(t, isExitPlanOptions(nil))
	assert.False(t, isExitPlanOptions([]string{}))
}

// ---------------------------------------------------------------------------
// rootRequestID
// ---------------------------------------------------------------------------

func TestRootRequestID_NoSuffix(t *testing.T) {
	assert.Equal(t, "req-001", rootRequestID("req-001"))
}

func TestRootRequestID_WithStepSuffix(t *testing.T) {
	assert.Equal(t, "req-001", rootRequestID("req-001:step1"))
}

func TestRootRequestID_WithStep0Suffix(t *testing.T) {
	assert.Equal(t, "req-001", rootRequestID("req-001:step0"))
}

func TestRootRequestID_NotStepSuffix(t *testing.T) {
	// ":foo" is not a ":step<digits>" suffix — must return original.
	assert.Equal(t, "req-001:foo", rootRequestID("req-001:foo"))
}

func TestRootRequestID_MultipleColons(t *testing.T) {
	// Only the last ":step<N>" is stripped.
	assert.Equal(t, "a:b:c", rootRequestID("a:b:c:step5"))
}

// ---------------------------------------------------------------------------
// buildExitPlanResult
// ---------------------------------------------------------------------------

func TestBuildExitPlanResult_Approve(t *testing.T) {
	result, err := buildExitPlanResult("Approve Plan", "")
	require.NoError(t, err)

	var v exitPlanResult
	require.NoError(t, json.Unmarshal([]byte(result), &v))
	assert.Equal(t, "approve", v.Decision)
	assert.Empty(t, v.Feedback)
}

func TestBuildExitPlanResult_Reject(t *testing.T) {
	result, err := buildExitPlanResult("Reject Plan", "")
	require.NoError(t, err)

	var v exitPlanResult
	require.NoError(t, json.Unmarshal([]byte(result), &v))
	assert.Equal(t, "reject", v.Decision)
}

func TestBuildExitPlanResult_Changes(t *testing.T) {
	result, err := buildExitPlanResult("Request Changes", "please fix the tests")
	require.NoError(t, err)

	var v exitPlanResult
	require.NoError(t, json.Unmarshal([]byte(result), &v))
	assert.Equal(t, "changes", v.Decision)
	assert.Equal(t, "please fix the tests", v.Feedback)
}

func TestBuildExitPlanResult_UnknownDecision(t *testing.T) {
	// Unknown decision values are passed through as-is.
	result, err := buildExitPlanResult("whatever", "")
	require.NoError(t, err)

	var v exitPlanResult
	require.NoError(t, json.Unmarshal([]byte(result), &v))
	assert.Equal(t, "whatever", v.Decision)
}

// ---------------------------------------------------------------------------
// HandleBridgeRequest — flow type detection
// ---------------------------------------------------------------------------

func TestHandleBridgeRequest_FlowInput_NoOptions(t *testing.T) {
	h, mq := newTestHandler()

	_ = h.HandleBridgeRequest("req-input", "Enter value:", nil)

	assert.True(t, h.IsPending("req-input"), "flow must be pending")
	require.True(t, mq.IsActive(), "queue must be active")
	assert.Equal(t, Input, mq.ActiveModel().request.Type)
}

func TestHandleBridgeRequest_FlowConfirm_YesNo(t *testing.T) {
	h, mq := newTestHandler()

	_ = h.HandleBridgeRequest("req-confirm", "Proceed?", []string{"Yes", "No"})

	require.True(t, mq.IsActive())
	assert.Equal(t, Confirm, mq.ActiveModel().request.Type)
}

func TestHandleBridgeRequest_FlowAskUser_MultipleOptions(t *testing.T) {
	h, mq := newTestHandler()

	_ = h.HandleBridgeRequest("req-ask", "Which option?", []string{"A", "B", "C"})

	require.True(t, mq.IsActive())
	assert.Equal(t, Ask, mq.ActiveModel().request.Type)
}

func TestHandleBridgeRequest_FlowExitPlan_CanonicalOptions(t *testing.T) {
	h, mq := newTestHandler()

	_ = h.HandleBridgeRequest("req-exit",
		"Review the plan:",
		[]string{"Approve Plan", "Request Changes", "Reject Plan"},
	)

	require.True(t, mq.IsActive())
	assert.Equal(t, Select, mq.ActiveModel().request.Type)
}

func TestHandleBridgeRequest_RegistersPending(t *testing.T) {
	h, _ := newTestHandler()
	_ = h.HandleBridgeRequest("req-001", "hello", nil)
	assert.True(t, h.IsPending("req-001"))
}

func TestHandleBridgeRequest_ModalIDMatchesRequest(t *testing.T) {
	h, mq := newTestHandler()
	_ = h.HandleBridgeRequest("req-xyz", "test", nil)

	require.True(t, mq.IsActive())
	// For single-step flows the modal ID equals the request ID.
	assert.Equal(t, "req-xyz", mq.ActiveModel().request.ID)
}

// ---------------------------------------------------------------------------
// HandleResponse — single-step flows
// ---------------------------------------------------------------------------

func TestHandleResponse_Input_Completes(t *testing.T) {
	h, mq := newTestHandler()
	_ = h.HandleBridgeRequest("req-in", "Enter value:", nil)

	// Advance queue state to match what AppModel.Update would do.
	mq.Resolve(ModalResponse{Type: Input, Value: "typed text"})

	result, cmd := h.HandleResponse(simulateResponse("req-in", "typed text"))

	require.NotNil(t, result, "result must be non-nil for completed single-step flow")
	assert.Equal(t, "req-in", result.RequestID)
	assert.Equal(t, "typed text", result.Value)
	assert.False(t, result.Cancelled)
	assert.Nil(t, cmd)
	assert.False(t, h.IsPending("req-in"), "flow must be removed from pending after completion")
}

func TestHandleResponse_Confirm_Yes(t *testing.T) {
	h, mq := newTestHandler()
	_ = h.HandleBridgeRequest("req-cn", "Continue?", []string{"Yes", "No"})
	mq.Resolve(ModalResponse{Type: Confirm, Value: "Yes"})

	result, _ := h.HandleResponse(simulateResponse("req-cn", "Yes"))

	require.NotNil(t, result)
	assert.Equal(t, "Yes", result.Value)
}

func TestHandleResponse_Confirm_No(t *testing.T) {
	h, mq := newTestHandler()
	_ = h.HandleBridgeRequest("req-cn2", "Continue?", []string{"Yes", "No"})
	mq.Resolve(ModalResponse{Type: Confirm, Value: "No"})

	result, _ := h.HandleResponse(simulateResponse("req-cn2", "No"))

	require.NotNil(t, result)
	assert.Equal(t, "No", result.Value)
}

func TestHandleResponse_Ask_NamedOption(t *testing.T) {
	h, mq := newTestHandler()
	_ = h.HandleBridgeRequest("req-ask2", "Which?", []string{"Alpha", "Beta"})
	mq.Resolve(ModalResponse{Type: Ask, Value: "Alpha"})

	result, _ := h.HandleResponse(simulateResponse("req-ask2", "Alpha"))

	require.NotNil(t, result)
	assert.Equal(t, "Alpha", result.Value)
}

func TestHandleResponse_Select_SingleStep(t *testing.T) {
	h, mq := newTestHandler()
	_ = h.HandleBridgeRequest("req-sel", "Pick one:", []string{"X", "Y", "Z"})
	mq.Resolve(ModalResponse{Type: Ask, Value: "Y"})

	result, _ := h.HandleResponse(simulateResponse("req-sel", "Y"))

	require.NotNil(t, result)
	assert.Equal(t, "Y", result.Value)
}

// ---------------------------------------------------------------------------
// HandleResponse — cancellation
// ---------------------------------------------------------------------------

func TestHandleResponse_Cancelled_SingleStep(t *testing.T) {
	h, mq := newTestHandler()
	_ = h.HandleBridgeRequest("req-cancel", "Do it?", nil)
	mq.Resolve(ModalResponse{Type: Input, Cancelled: true})

	result, cmd := h.HandleResponse(simulateCancelled("req-cancel"))

	require.NotNil(t, result)
	assert.True(t, result.Cancelled)
	assert.Empty(t, result.Value)
	assert.Nil(t, cmd)
	assert.False(t, h.IsPending("req-cancel"))
}

func TestHandleResponse_Cancelled_ExitPlanStep0(t *testing.T) {
	h, mq := newTestHandler()
	_ = h.HandleBridgeRequest("req-ep-cancel",
		"Review:",
		[]string{"Approve Plan", "Request Changes", "Reject Plan"},
	)
	mq.Resolve(ModalResponse{Type: Select, Cancelled: true})

	result, cmd := h.HandleResponse(simulateCancelled("req-ep-cancel"))

	require.NotNil(t, result)
	assert.True(t, result.Cancelled)
	assert.Nil(t, cmd)
}

// ---------------------------------------------------------------------------
// HandleResponse — ExitPlan multi-step
// ---------------------------------------------------------------------------

func TestHandleResponse_ExitPlan_Approve(t *testing.T) {
	h, mq := newTestHandler()
	_ = h.HandleBridgeRequest("req-ep1",
		"Review the plan:",
		[]string{"Approve Plan", "Request Changes", "Reject Plan"},
	)
	mq.Resolve(ModalResponse{Type: Select, Value: "Approve Plan"})

	result, cmd := h.HandleResponse(simulateResponse("req-ep1", "Approve Plan"))

	require.NotNil(t, result, "Approve Plan must complete in one step")
	assert.Equal(t, "req-ep1", result.RequestID)
	assert.False(t, result.Cancelled)
	assert.Nil(t, cmd)

	// Value must be JSON with decision=approve.
	var v exitPlanResult
	require.NoError(t, json.Unmarshal([]byte(result.Value), &v))
	assert.Equal(t, "approve", v.Decision)
	assert.Empty(t, v.Feedback)
}

func TestHandleResponse_ExitPlan_Reject(t *testing.T) {
	h, mq := newTestHandler()
	_ = h.HandleBridgeRequest("req-ep2",
		"Review:",
		[]string{"Approve Plan", "Request Changes", "Reject Plan"},
	)
	mq.Resolve(ModalResponse{Type: Select, Value: "Reject Plan"})

	result, _ := h.HandleResponse(simulateResponse("req-ep2", "Reject Plan"))

	require.NotNil(t, result)
	var v exitPlanResult
	require.NoError(t, json.Unmarshal([]byte(result.Value), &v))
	assert.Equal(t, "reject", v.Decision)
}

func TestHandleResponse_ExitPlan_RequestChanges_Step1(t *testing.T) {
	h, mq := newTestHandler()
	_ = h.HandleBridgeRequest("req-ep3",
		"Review:",
		[]string{"Approve Plan", "Request Changes", "Reject Plan"},
	)

	// Step 0: user selects "Request Changes".
	mq.Resolve(ModalResponse{Type: Select, Value: "Request Changes"})
	result0, cmd0 := h.HandleResponse(simulateResponse("req-ep3", "Request Changes"))

	// Must NOT complete yet — a second modal step is queued.
	assert.Nil(t, result0, "result must be nil after step 0 of Request Changes flow")
	assert.Nil(t, cmd0, "cmd from enqueueStep returns nil (queue was already active)")
	assert.True(t, h.IsPending("req-ep3"), "flow still pending after step 0")

	// The queue should now be active with the feedback Input modal.
	require.True(t, mq.IsActive(), "queue must be active for step 1")
	assert.Equal(t, Input, mq.ActiveModel().request.Type)

	// Step 1: user submits feedback.
	mq.Resolve(ModalResponse{Type: Input, Value: "add error handling"})
	result1, cmd1 := h.HandleResponse(ModalResponseMsg{
		RequestID: stepID("req-ep3", 1),
		Response:  ModalResponse{Value: "add error handling"},
	})

	require.NotNil(t, result1, "result must be non-nil after step 1 completes")
	assert.Equal(t, "req-ep3", result1.RequestID)
	assert.Nil(t, cmd1)
	assert.False(t, h.IsPending("req-ep3"))

	var v exitPlanResult
	require.NoError(t, json.Unmarshal([]byte(result1.Value), &v))
	assert.Equal(t, "changes", v.Decision)
	assert.Equal(t, "add error handling", v.Feedback)
}

func TestHandleResponse_ExitPlan_RequestChanges_CancelledOnStep1(t *testing.T) {
	h, mq := newTestHandler()
	_ = h.HandleBridgeRequest("req-ep4",
		"Review:",
		[]string{"Approve Plan", "Request Changes", "Reject Plan"},
	)

	// Step 0: select "Request Changes".
	mq.Resolve(ModalResponse{Type: Select, Value: "Request Changes"})
	_, _ = h.HandleResponse(simulateResponse("req-ep4", "Request Changes"))

	// Step 1: cancel the feedback modal.
	mq.Resolve(ModalResponse{Type: Input, Cancelled: true})
	result, _ := h.HandleResponse(ModalResponseMsg{
		RequestID: stepID("req-ep4", 1),
		Response:  ModalResponse{Cancelled: true},
	})

	require.NotNil(t, result)
	assert.True(t, result.Cancelled)
	assert.False(t, h.IsPending("req-ep4"))
}

// ---------------------------------------------------------------------------
// HandleResponse — unknown request ID (no-op)
// ---------------------------------------------------------------------------

func TestHandleResponse_UnknownRequestID_IsNoop(t *testing.T) {
	h, _ := newTestHandler()

	result, cmd := h.HandleResponse(simulateResponse("nonexistent", "value"))

	assert.Nil(t, result)
	assert.Nil(t, cmd)
}

// ---------------------------------------------------------------------------
// Concurrent safety — multiple sequential requests
// ---------------------------------------------------------------------------

func TestHandleBridgeRequest_MultipleSequential_IndependentFlows(t *testing.T) {
	h, mq := newTestHandler()

	// Enqueue two requests sequentially (they queue up).
	_ = h.HandleBridgeRequest("flow-1", "First question:", nil)
	_ = h.HandleBridgeRequest("flow-2", "Second question:", nil)

	// Both must be pending.
	assert.True(t, h.IsPending("flow-1"))
	assert.True(t, h.IsPending("flow-2"))

	// Only flow-1 should be shown first.
	require.True(t, mq.IsActive())
	assert.Equal(t, "flow-1", mq.ActiveModel().request.ID)
	assert.Equal(t, 1, mq.Len(), "flow-2 must still be queued")

	// Resolve flow-1.
	mq.Resolve(ModalResponse{Type: Input, Value: "answer1"})
	r1, _ := h.HandleResponse(simulateResponse("flow-1", "answer1"))
	require.NotNil(t, r1)
	assert.Equal(t, "flow-1", r1.RequestID)

	// flow-2 should now be active.
	require.True(t, mq.IsActive())
	assert.Equal(t, "flow-2", mq.ActiveModel().request.ID)

	// Resolve flow-2.
	mq.Resolve(ModalResponse{Type: Input, Value: "answer2"})
	r2, _ := h.HandleResponse(simulateResponse("flow-2", "answer2"))
	require.NotNil(t, r2)
	assert.Equal(t, "flow-2", r2.RequestID)

	assert.False(t, mq.IsActive())
}

// ---------------------------------------------------------------------------
// Round-trip timing — must complete well under 100ms in-process
// ---------------------------------------------------------------------------

func TestHandleBridgeRequest_RoundTripTiming(t *testing.T) {
	h, mq := newTestHandler()

	start := time.Now()

	_ = h.HandleBridgeRequest("timing-req", "Quick question:", nil)
	require.True(t, mq.IsActive())

	mq.Resolve(ModalResponse{Type: Input, Value: "quick answer"})
	result, _ := h.HandleResponse(simulateResponse("timing-req", "quick answer"))

	elapsed := time.Since(start)

	require.NotNil(t, result)
	assert.Less(t, elapsed.Milliseconds(), int64(100),
		"round-trip must complete in under 100ms; took %v", elapsed)
}
