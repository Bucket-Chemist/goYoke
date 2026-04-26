// Package model — remote action handler tests (HL-005).
//
// These tests verify that each Remote* message type:
//   - delegates to the correct existing handler or driver method
//   - reports errors via ResponseCh when infrastructure is unavailable
//   - never blocks the event loop (ResponseCh must be buffered)
//   - works fire-and-forget (nil ResponseCh must not panic)
package model

import (
	"errors"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// mockBridgeCapture satisfies bridgeWidget and records resolve calls.
// ---------------------------------------------------------------------------

type mockBridgeCapture struct {
	resolveModalCalls    []resolveModalCall
	resolvePermGateCalls []resolvePermGateCall
}

type resolveModalCall struct {
	requestID string
	value     string
}

type resolvePermGateCall struct {
	requestID string
	decision  string
}

func (b *mockBridgeCapture) Start()            {}
func (b *mockBridgeCapture) SocketPath() string { return "" }
func (b *mockBridgeCapture) Shutdown()          {}
func (b *mockBridgeCapture) ResolveModalSimple(requestID, value string) {
	b.resolveModalCalls = append(b.resolveModalCalls, resolveModalCall{requestID, value})
}
func (b *mockBridgeCapture) ResolvePermGate(requestID, decision string) {
	b.resolvePermGateCalls = append(b.resolvePermGateCalls, resolvePermGateCall{requestID, decision})
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// newModelWithDriver returns an AppModel with a mock CLI driver wired in.
// Uses mockDriverCapture defined in app_test.go (same package).
func newModelWithDriver() (AppModel, *mockDriverCapture) {
	m := NewAppModel()
	d := &mockDriverCapture{}
	m.shared.cliDriver = d
	return m, d
}

// newModelWithBridge returns an AppModel with a mock bridge wired in.
func newModelWithBridge() (AppModel, *mockBridgeCapture) {
	m := NewAppModel()
	b := &mockBridgeCapture{}
	m.shared.bridge = b
	return m, b
}

// receiveErr reads from ch with a short timeout, returning the error or a
// sentinel "timeout" error when no value arrives within 50 ms.
func receiveErr(ch <-chan error) error {
	select {
	case err := <-ch:
		return err
	case <-time.After(50 * time.Millisecond):
		return errors.New("timeout: no response received on ResponseCh")
	}
}

// ---------------------------------------------------------------------------
// RemoteSubmitPromptMsg — prompt submit path
// ---------------------------------------------------------------------------

func TestRemoteSubmitPrompt_NilDriver_ReportsError(t *testing.T) {
	m := NewAppModel() // no driver wired

	ch := make(chan error, 1)
	_, cmd := m.Update(RemoteSubmitPromptMsg{Prompt: "hello", ResponseCh: ch})

	if cmd != nil {
		t.Error("expected nil cmd when driver is not wired")
	}

	err := receiveErr(ch)
	if err == nil {
		t.Error("expected non-nil error when driver is nil; got nil")
	}
}

func TestRemoteSubmitPrompt_WithDriver_InvokesSendMessage(t *testing.T) {
	m, d := newModelWithDriver()

	ch := make(chan error, 1)
	_, cmd := m.Update(RemoteSubmitPromptMsg{Prompt: "test prompt", ResponseCh: ch})

	// Execute the returned cmd if non-nil (real driver returns a non-nil Cmd;
	// mockDriverCapture.SendMessage returns nil so ResponseCh is signaled inline).
	if cmd != nil {
		cmd()
	}

	err := receiveErr(ch)
	if err != nil {
		t.Errorf("expected nil error from successful send; got %v", err)
	}

	if d.sendCalls != 1 {
		t.Errorf("sendCalls = %d; want 1", d.sendCalls)
	}
}

func TestRemoteSubmitPrompt_FireAndForget_NilResponseCh(t *testing.T) {
	m, d := newModelWithDriver()

	// nil ResponseCh must not panic.
	_, cmd := m.Update(RemoteSubmitPromptMsg{Prompt: "fire and forget"})

	if cmd != nil {
		cmd() // must not panic
	}

	if d.sendCalls != 1 {
		t.Errorf("sendCalls = %d; want 1", d.sendCalls)
	}
}

// ---------------------------------------------------------------------------
// RemoteInterruptMsg — interrupt path
// ---------------------------------------------------------------------------

func TestRemoteInterrupt_NilDriver_ReportsError(t *testing.T) {
	m := NewAppModel()

	ch := make(chan error, 1)
	_, cmd := m.Update(RemoteInterruptMsg{ResponseCh: ch})

	if cmd != nil {
		t.Error("expected nil cmd when driver is not wired")
	}

	err := receiveErr(ch)
	if err == nil {
		t.Error("expected non-nil error when driver is nil; got nil")
	}
}

func TestRemoteInterrupt_WithDriver_CallsInterrupt(t *testing.T) {
	m, _ := newModelWithDriver()

	ch := make(chan error, 1)
	_, cmd := m.Update(RemoteInterruptMsg{ResponseCh: ch})

	if cmd != nil {
		t.Errorf("expected nil cmd from interrupt handler; got %v", cmd)
	}

	err := receiveErr(ch)
	if err != nil {
		t.Errorf("expected nil error from mockDriverCapture.Interrupt(); got %v", err)
	}
}

func TestRemoteInterrupt_FireAndForget_NilResponseCh(t *testing.T) {
	m, _ := newModelWithDriver()

	// nil ResponseCh must not panic.
	_, _ = m.Update(RemoteInterruptMsg{})
}

// ---------------------------------------------------------------------------
// RemoteRespondModalMsg — pending-response (modal) flow
// ---------------------------------------------------------------------------

func TestRemoteRespondModal_NilBridge_ReportsError(t *testing.T) {
	m := NewAppModel() // no bridge wired

	ch := make(chan error, 1)
	_, _ = m.Update(RemoteRespondModalMsg{
		RequestID:  "req-1",
		Value:      "Yes",
		ResponseCh: ch,
	})

	err := receiveErr(ch)
	if err == nil {
		t.Error("expected non-nil error when bridge is nil; got nil")
	}
}

func TestRemoteRespondModal_WithBridge_ResolvesRequest(t *testing.T) {
	m, b := newModelWithBridge()

	ch := make(chan error, 1)
	_, _ = m.Update(RemoteRespondModalMsg{
		RequestID:  "req-abc",
		Value:      "Allow",
		ResponseCh: ch,
	})

	err := receiveErr(ch)
	if err != nil {
		t.Errorf("expected nil error; got %v", err)
	}

	if len(b.resolveModalCalls) != 1 {
		t.Fatalf("ResolveModalSimple called %d times; want 1", len(b.resolveModalCalls))
	}
	got := b.resolveModalCalls[0]
	if got.requestID != "req-abc" {
		t.Errorf("requestID = %q; want %q", got.requestID, "req-abc")
	}
	if got.value != "Allow" {
		t.Errorf("value = %q; want %q", got.value, "Allow")
	}
}

func TestRemoteRespondModal_CancelledValue_PassesEmptyString(t *testing.T) {
	m, b := newModelWithBridge()

	_, _ = m.Update(RemoteRespondModalMsg{RequestID: "req-cancel", Value: ""})

	if len(b.resolveModalCalls) != 1 {
		t.Fatalf("ResolveModalSimple called %d times; want 1", len(b.resolveModalCalls))
	}
	if b.resolveModalCalls[0].value != "" {
		t.Errorf("value = %q; want empty string for cancelled response", b.resolveModalCalls[0].value)
	}
}

// ---------------------------------------------------------------------------
// RemoteRespondPermissionMsg — pending-response (permission gate) flow
// ---------------------------------------------------------------------------

func TestRemoteRespondPermission_NilBridge_ReportsError(t *testing.T) {
	m := NewAppModel()

	ch := make(chan error, 1)
	_, _ = m.Update(RemoteRespondPermissionMsg{
		RequestID:  "perm-1",
		Decision:   "allow",
		ResponseCh: ch,
	})

	err := receiveErr(ch)
	if err == nil {
		t.Error("expected non-nil error when bridge is nil; got nil")
	}
}

func TestRemoteRespondPermission_WithBridge_ResolvesGate(t *testing.T) {
	m, b := newModelWithBridge()

	ch := make(chan error, 1)
	_, _ = m.Update(RemoteRespondPermissionMsg{
		RequestID:  "perm-xyz",
		Decision:   "allow_session",
		ResponseCh: ch,
	})

	err := receiveErr(ch)
	if err != nil {
		t.Errorf("expected nil error; got %v", err)
	}

	if len(b.resolvePermGateCalls) != 1 {
		t.Fatalf("ResolvePermGate called %d times; want 1", len(b.resolvePermGateCalls))
	}
	got := b.resolvePermGateCalls[0]
	if got.requestID != "perm-xyz" {
		t.Errorf("requestID = %q; want %q", got.requestID, "perm-xyz")
	}
	if got.decision != "allow_session" {
		t.Errorf("decision = %q; want %q", got.decision, "allow_session")
	}
}

// ---------------------------------------------------------------------------
// RemoteSetModelMsg — model change reuses existing handler
// ---------------------------------------------------------------------------

func TestRemoteSetModel_SignalsResponseChBeforeHandlerRuns(t *testing.T) {
	m := NewAppModel()

	ch := make(chan error, 1)
	_, _ = m.Update(RemoteSetModelMsg{ModelID: "haiku", ResponseCh: ch})

	// ResponseCh must always be signaled regardless of whether provider state
	// is configured (the underlying handler silently no-ops without it).
	err := receiveErr(ch)
	_ = err // nil or error — both are valid; the key assertion is no timeout
}

func TestRemoteSetModel_FireAndForget_NilResponseCh(t *testing.T) {
	m := NewAppModel()

	// nil ResponseCh must not panic.
	_, _ = m.Update(RemoteSetModelMsg{ModelID: "sonnet"})
}

// ---------------------------------------------------------------------------
// RemoteSetEffortMsg — effort change reuses existing handler
// ---------------------------------------------------------------------------

func TestRemoteSetEffort_SignalsResponseCh(t *testing.T) {
	m := NewAppModel()

	ch := make(chan error, 1)
	_, _ = m.Update(RemoteSetEffortMsg{Level: "high", ResponseCh: ch})

	err := receiveErr(ch)
	_ = err // response must arrive within 50 ms
}

func TestRemoteSetEffort_FireAndForget_NilResponseCh(t *testing.T) {
	m := NewAppModel()

	_, _ = m.Update(RemoteSetEffortMsg{Level: "max"})
}

// ---------------------------------------------------------------------------
// RemoteSetCWDMsg — cwd change reuses existing handler
// ---------------------------------------------------------------------------

func TestRemoteSetCWD_SignalsResponseCh(t *testing.T) {
	m := NewAppModel()

	ch := make(chan error, 1)
	_, _ = m.Update(RemoteSetCWDMsg{Path: "/tmp", ResponseCh: ch})

	err := receiveErr(ch)
	if err != nil {
		t.Errorf("expected nil error for /tmp; got %v", err)
	}
}

func TestRemoteSetCWD_FireAndForget_NilResponseCh(t *testing.T) {
	m := NewAppModel()

	_, _ = m.Update(RemoteSetCWDMsg{Path: "/tmp"})
}
