package model

import (
	"strings"
	"testing"

	"github.com/Bucket-Chemist/goYoke/internal/tui/components/modals"
	"github.com/Bucket-Chemist/goYoke/internal/tui/config"
	"github.com/Bucket-Chemist/goYoke/internal/tui/state"
	"github.com/Bucket-Chemist/goYoke/pkg/harnessproto"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// snapshotModel returns a minimal AppModel suitable for snapshot tests.
// The shared registry and modal queue are initialised; no GUI components
// are wired so most widget paths are guarded by nil checks.
func snapshotModel() AppModel {
	return NewAppModel()
}

// ---------------------------------------------------------------------------
// Idle state
// ---------------------------------------------------------------------------

func TestBuildHarnessSnapshot_Idle(t *testing.T) {
	m := snapshotModel()
	m.sessionID = "sess-idle"
	m.activeModel = "claude-sonnet-4-6"

	snap := m.BuildHarnessSnapshot()

	if snap.Protocol != harnessproto.ProtocolName {
		t.Errorf("Protocol = %q; want %q", snap.Protocol, harnessproto.ProtocolName)
	}
	if snap.ProtocolVersion != harnessproto.ProtocolVersion {
		t.Errorf("ProtocolVersion = %q; want %q", snap.ProtocolVersion, harnessproto.ProtocolVersion)
	}
	if snap.SessionID != "sess-idle" {
		t.Errorf("SessionID = %q; want %q", snap.SessionID, "sess-idle")
	}
	if snap.Model != "claude-sonnet-4-6" {
		t.Errorf("Model = %q; want claude-sonnet-4-6", snap.Model)
	}
	if snap.Status != "idle" {
		t.Errorf("Status = %q; want idle", snap.Status)
	}
	if snap.Streaming {
		t.Error("Streaming should be false for idle snapshot")
	}
	if snap.Pending != nil {
		t.Errorf("Pending should be nil for idle snapshot, got %+v", snap.Pending)
	}
	if snap.Agents == nil {
		t.Error("Agents must be non-nil (may be empty)")
	}
	if snap.StateHash == "" {
		t.Error("StateHash must be non-empty")
	}
	if snap.PublishHash == "" {
		t.Error("PublishHash must be non-empty")
	}
	if snap.Timestamp.IsZero() {
		t.Error("Timestamp must be non-zero")
	}
}

// ---------------------------------------------------------------------------
// Streaming state
// ---------------------------------------------------------------------------

func TestBuildHarnessSnapshot_Streaming(t *testing.T) {
	m := snapshotModel()
	m.sessionID = "sess-stream"
	m.statusLine.Streaming = true

	snap := m.BuildHarnessSnapshot()

	if snap.Status != "streaming" {
		t.Errorf("Status = %q; want streaming", snap.Status)
	}
	if !snap.Streaming {
		t.Error("Streaming should be true")
	}
}

// TestBuildHarnessSnapshot_StreamingViaPanelWidget verifies that streaming
// detected on the claudePanel widget overrides a false statusLine.Streaming.
func TestBuildHarnessSnapshot_StreamingViaPanelWidget(t *testing.T) {
	m := snapshotModel()
	panel := &mockClaudePanel{streaming: true}
	m.shared.claudePanel = panel

	snap := m.BuildHarnessSnapshot()

	if snap.Status != "streaming" {
		t.Errorf("Status = %q; want streaming", snap.Status)
	}
	if !snap.Streaming {
		t.Error("Streaming should be true when panel.IsStreaming() is true")
	}
}

// ---------------------------------------------------------------------------
// Pending user-input state (modal overlay)
// ---------------------------------------------------------------------------

func TestBuildHarnessSnapshot_PendingModal(t *testing.T) {
	m := snapshotModel()

	// Push a bridge modal request through the PermissionHandler so the queue
	// holds an active ask modal. This mirrors the real code path in
	// handleBridgeModalRequest (compact tier fallback).
	km := config.DefaultKeyMap()
	mq := modals.NewModalQueue(km)
	ph := modals.NewPermissionHandler(&mq)
	m.shared.modalQueue = &mq
	m.shared.permHandler = ph

	_ = ph.HandleBridgeRequest("req-ask", "Which option do you prefer?", []string{"A", "B"})

	snap := m.BuildHarnessSnapshot()

	if snap.Status != "waiting_modal" {
		t.Errorf("Status = %q; want waiting_modal", snap.Status)
	}
	if snap.Pending == nil {
		t.Fatal("Pending must be non-nil when a modal is active")
	}
	if snap.Pending.Kind != "modal" {
		t.Errorf("Pending.Kind = %q; want modal", snap.Pending.Kind)
	}
	if !strings.Contains(snap.Pending.Message, "Which option") {
		t.Errorf("Pending.Message = %q; want to contain the modal message", snap.Pending.Message)
	}
}

// TestBuildHarnessSnapshot_PendingPermission verifies that tool-permission
// gate requests surface as Kind: "permission" with Priority over plain modals.
func TestBuildHarnessSnapshot_PendingPermission(t *testing.T) {
	m := snapshotModel()

	km := config.DefaultKeyMap()
	mq := modals.NewModalQueue(km)
	ph := modals.NewPermissionHandler(&mq)
	m.shared.modalQueue = &mq
	m.shared.permHandler = ph

	_ = ph.HandlePermGateRequest("req-perm", "Allow Bash command?", []string{"Allow", "Deny", "Allow for Session"}, 30000)

	snap := m.BuildHarnessSnapshot()

	if snap.Status != "waiting_permission" {
		t.Errorf("Status = %q; want waiting_permission", snap.Status)
	}
	if snap.Pending == nil {
		t.Fatal("Pending must be non-nil when a permission gate is active")
	}
	if snap.Pending.Kind != "permission" {
		t.Errorf("Pending.Kind = %q; want permission", snap.Pending.Kind)
	}
}

// ---------------------------------------------------------------------------
// Error state (reconnecting + error in conversation history)
// ---------------------------------------------------------------------------

func TestBuildHarnessSnapshot_Reconnecting(t *testing.T) {
	m := snapshotModel()
	m.reconnectCount = 2

	snap := m.BuildHarnessSnapshot()

	if !snap.Reconnecting {
		t.Error("Reconnecting should be true when reconnectCount > 0")
	}
}

func TestBuildHarnessSnapshot_LastMessagesFromHistory(t *testing.T) {
	m := snapshotModel()

	panel := &mockClaudePanel{
		savedMessages: []state.DisplayMessage{
			{Role: "user", Content: "What is the capital of France?"},
			{Role: "assistant", Content: "The capital of France is Paris."},
			{Role: "system", Content: "Error: connection failed"},
		},
	}
	m.shared.claudePanel = panel

	snap := m.BuildHarnessSnapshot()

	if snap.LastUser != "What is the capital of France?" {
		t.Errorf("LastUser = %q; want user question", snap.LastUser)
	}
	if snap.LastAssistant != "The capital of France is Paris." {
		t.Errorf("LastAssistant = %q; want assistant answer", snap.LastAssistant)
	}
	if !strings.Contains(snap.LastError, "Error") {
		t.Errorf("LastError = %q; want error system message", snap.LastError)
	}
}

// TestBuildHarnessSnapshot_ShuttingDown verifies the shutting_down status
// takes highest priority.
func TestBuildHarnessSnapshot_ShuttingDown(t *testing.T) {
	m := snapshotModel()
	m.shutdownInProgress = true
	m.statusLine.Streaming = true // streaming should be overridden by shutdown

	snap := m.BuildHarnessSnapshot()

	if snap.Status != "shutting_down" {
		t.Errorf("Status = %q; want shutting_down", snap.Status)
	}
	if !snap.ShuttingDown {
		t.Error("ShuttingDown should be true")
	}
}

// ---------------------------------------------------------------------------
// Agent summaries
// ---------------------------------------------------------------------------

func TestBuildHarnessSnapshot_AgentSummaries(t *testing.T) {
	m := snapshotModel()

	_ = m.shared.agentRegistry.Register(state.Agent{
		ID:          "agent-1",
		AgentType:   "go-pro",
		Description: "Implement feature X",
		Model:       "sonnet",
		Status:      state.StatusRunning,
	})
	m.shared.agentRegistry.InvalidateTreeCache()

	snap := m.BuildHarnessSnapshot()

	if len(snap.Agents) == 0 {
		t.Fatal("Agents slice should contain at least one summary")
	}

	var found bool
	for _, a := range snap.Agents {
		if a.ID == "agent-1" {
			found = true
			if a.Status != "running" {
				t.Errorf("agent-1 Status = %q; want running", a.Status)
			}
			if a.Model != "sonnet" {
				t.Errorf("agent-1 Model = %q; want sonnet", a.Model)
			}
		}
	}
	if !found {
		t.Error("agent-1 not found in Agents slice")
	}
}

// ---------------------------------------------------------------------------
// State and publish hashes change with state
// ---------------------------------------------------------------------------

func TestBuildHarnessSnapshot_HashChangesWithState(t *testing.T) {
	m1 := snapshotModel()
	snap1 := m1.BuildHarnessSnapshot()

	m2 := snapshotModel()
	m2.statusLine.Streaming = true
	snap2 := m2.BuildHarnessSnapshot()

	if snap1.StateHash == snap2.StateHash {
		t.Error("StateHash should differ between idle and streaming snapshots")
	}
	if snap1.PublishHash == snap2.PublishHash {
		t.Error("PublishHash should differ between idle and streaming snapshots")
	}
}

// TestBuildHarnessSnapshot_TruncatesLongMessages verifies that conversation
// messages exceeding 200 runes are truncated.
func TestBuildHarnessSnapshot_TruncatesLongMessages(t *testing.T) {
	m := snapshotModel()

	long := strings.Repeat("x", 300)
	panel := &mockClaudePanel{
		savedMessages: []state.DisplayMessage{
			{Role: "user", Content: long},
		},
	}
	m.shared.claudePanel = panel

	snap := m.BuildHarnessSnapshot()

	runes := []rune(snap.LastUser)
	if len(runes) > 201 { // 200 content + 1 ellipsis
		t.Errorf("LastUser length = %d runes; expected ≤ 201", len(runes))
	}
	if !strings.HasSuffix(snap.LastUser, "…") {
		t.Error("Truncated LastUser should end with '…'")
	}
}
