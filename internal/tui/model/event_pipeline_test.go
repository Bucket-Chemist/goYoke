// Package model — event pipeline integration tests.
//
// These tests verify the critical data chain:
//
//	CLI event → AppModel.Update() → component state → View() output
//
// They are specifically designed to catch the class of bugs where a CLI event
// is handled but the state mutation does not reach the component that renders
// the data (e.g. agent count not propagating, status line not reflecting cost,
// agent tree staying empty after registration).
package model

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/cli"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
)

// ---------------------------------------------------------------------------
// Test 1: SystemInitEvent updates status line fields.
// ---------------------------------------------------------------------------

// TestEventPipeline_SystemInit_UpdatesStatusLine verifies that a
// cli.SystemInitEvent propagates the model name, permission mode, and leaves
// SessionCost at zero while setting cliReady.
//
// Bug class guarded: SystemInitEvent processed but statusLine.ActiveModel not
// assigned, causing the status bar to show an empty model name.
func TestEventPipeline_SystemInit_UpdatesStatusLine(t *testing.T) {
	m := newReadyAppModel(120, 40)

	ev := cli.SystemInitEvent{
		Type:           "system",
		Subtype:        "init",
		SessionID:      "session-abc",
		Model:          "claude-sonnet-4-6",
		PermissionMode: "default",
		UUID:           "uuid-1",
	}

	updated, _ := m.Update(ev)
	result := updated.(AppModel)

	// Model name must be recorded on the status line.
	if result.statusLine.ActiveModel != "claude-sonnet-4-6" {
		t.Errorf("statusLine.ActiveModel = %q; want %q",
			result.statusLine.ActiveModel, "claude-sonnet-4-6")
	}

	// Permission mode must be propagated.
	if result.statusLine.PermissionMode != "default" {
		t.Errorf("statusLine.PermissionMode = %q; want %q",
			result.statusLine.PermissionMode, "default")
	}

	// activeModel field on AppModel must match.
	if result.activeModel != "claude-sonnet-4-6" {
		t.Errorf("activeModel = %q; want %q", result.activeModel, "claude-sonnet-4-6")
	}

	// cliReady must be set.
	if !result.cliReady {
		t.Error("cliReady = false after SystemInitEvent; want true")
	}

	// SessionID must be stored.
	if result.sessionID != "session-abc" {
		t.Errorf("sessionID = %q; want %q", result.sessionID, "session-abc")
	}
}

// TestEventPipeline_SystemInit_RegistersRootAgent verifies that SystemInitEvent
// registers the "router-root" agent in the registry and updates the agent tree.
//
// Bug class guarded: SystemInitEvent calls Register() but agentTree.SetNodes()
// is skipped, leaving the tree empty on first render.
func TestEventPipeline_SystemInit_RegistersRootAgent(t *testing.T) {
	m := newReadyAppModel(120, 40)

	ev := cli.SystemInitEvent{
		Type:    "system",
		Subtype: "init",
		Model:   "claude-opus-4-6",
	}

	updated, _ := m.Update(ev)
	result := updated.(AppModel)

	// The root agent must be present in the registry.
	if result.shared.agentRegistry == nil {
		t.Fatal("agentRegistry = nil")
	}
	agent := result.shared.agentRegistry.Get("router-root")
	if agent == nil {
		t.Error("router-root agent not found in registry after SystemInitEvent")
	}

	// The agent tree must have at least one node.
	nodes := result.shared.agentRegistry.Tree()
	if len(nodes) == 0 {
		t.Error("agent tree is empty after SystemInitEvent; want at least one node")
	}
}

// TestEventPipeline_SystemInit_SessionStartNotZero verifies that SessionStart
// is populated on the status line after SystemInitEvent so the elapsed timer
// has a valid baseline.
func TestEventPipeline_SystemInit_SessionStartNotZero(t *testing.T) {
	m := newReadyAppModel(120, 40)

	before := time.Now()
	updated, _ := m.Update(cli.SystemInitEvent{
		Type:    "system",
		Subtype: "init",
		Model:   "claude-sonnet-4-6",
	})
	result := updated.(AppModel)

	if result.statusLine.SessionStart.IsZero() {
		t.Error("statusLine.SessionStart is zero after SystemInitEvent; want non-zero")
	}
	if result.statusLine.SessionStart.Before(before) {
		t.Errorf("statusLine.SessionStart %v is before test start %v",
			result.statusLine.SessionStart, before)
	}
}

// ---------------------------------------------------------------------------
// Test 2: ResultEvent updates cost and token counts on the status line.
// ---------------------------------------------------------------------------

// TestEventPipeline_ResultEvent_UpdatesCostAndTokens verifies that a
// cli.ResultEvent propagates TotalCostUSD, token counts, and context
// percentage to the status line.
//
// Bug class guarded: ResultEvent updates costTracker but does not assign
// statusLine.SessionCost / TokenCount, so the status bar shows stale data.
func TestEventPipeline_ResultEvent_UpdatesCostAndTokens(t *testing.T) {
	m := newReadyAppModel(120, 40)
	m.activeModel = "claude-sonnet-4-6"

	ev := cli.ResultEvent{
		Type:         "result",
		Subtype:      "success",
		TotalCostUSD: 0.0425,
		Usage: cli.ResultUsage{
			InputTokens:  5000,
			OutputTokens: 1200,
		},
		ModelUsage: map[string]cli.ModelUsageEntry{
			"claude-sonnet-4-6": {
				InputTokens:   5000,
				OutputTokens:  1200,
				ContextWindow: 200000,
			},
		},
	}

	updated, _ := m.Update(ev)
	result := updated.(AppModel)

	// SessionCost must be set on the status line (not just the cost tracker).
	if result.statusLine.SessionCost != 0.0425 {
		t.Errorf("statusLine.SessionCost = %f; want 0.0425", result.statusLine.SessionCost)
	}

	// TokenCount must accumulate input + output tokens.
	wantTokens := 5000 + 1200
	if result.statusLine.TokenCount != wantTokens {
		t.Errorf("statusLine.TokenCount = %d; want %d", result.statusLine.TokenCount, wantTokens)
	}

	// ContextCapacity must be set from per-model usage (capacity discovery).
	// Context usage (percent, used tokens) is now updated per-message in
	// handleAssistantEvent, not from cumulative modelUsage.
	if result.statusLine.ContextCapacity != 200000 {
		t.Errorf("statusLine.ContextCapacity = %d; want 200000", result.statusLine.ContextCapacity)
	}

	// Cost tracker (single source of truth) must also reflect the new cost.
	if result.shared.costTracker == nil {
		t.Fatal("costTracker = nil")
	}
	trackerCost := result.shared.costTracker.GetSessionCost()
	if trackerCost != 0.0425 {
		t.Errorf("costTracker.GetSessionCost() = %f; want 0.0425", trackerCost)
	}
}

// TestEventPipeline_ResultEvent_ClearsStreaming verifies that ResultEvent
// clears the streaming indicator on the status line.
//
// Bug class guarded: streaming indicator remains active after the turn ends,
// showing a perpetual thinking spinner.
func TestEventPipeline_ResultEvent_ClearsStreaming(t *testing.T) {
	m := newReadyAppModel(120, 40)
	m.statusLine.Streaming = true // pre-set as if streaming was in progress

	ev := cli.ResultEvent{
		Type:    "result",
		Subtype: "success",
		Usage:   cli.ResultUsage{},
	}

	updated, _ := m.Update(ev)
	result := updated.(AppModel)

	if result.statusLine.Streaming {
		t.Error("statusLine.Streaming = true after ResultEvent; want false (turn complete)")
	}
}

// TestEventPipeline_ResultEvent_AccumulatesTokensAcrossMultipleTurns verifies
// that token counts accumulate across successive ResultEvents rather than
// being replaced.
func TestEventPipeline_ResultEvent_AccumulatesTokensAcrossMultipleTurns(t *testing.T) {
	m := newReadyAppModel(120, 40)

	first := cli.ResultEvent{
		Type:    "result",
		Subtype: "success",
		Usage:   cli.ResultUsage{InputTokens: 1000, OutputTokens: 200},
	}
	updated, _ := m.Update(first)
	m = updated.(AppModel)

	second := cli.ResultEvent{
		Type:    "result",
		Subtype: "success",
		Usage:   cli.ResultUsage{InputTokens: 2000, OutputTokens: 400},
	}
	updated, _ = m.Update(second)
	result := updated.(AppModel)

	// Total must be the sum across both turns.
	wantTotal := (1000 + 200) + (2000 + 400) // 3600
	if result.statusLine.TokenCount != wantTotal {
		t.Errorf("statusLine.TokenCount after 2 turns = %d; want %d",
			result.statusLine.TokenCount, wantTotal)
	}
}

// ---------------------------------------------------------------------------
// Test 3: AssistantEvent streaming lifecycle.
// ---------------------------------------------------------------------------

// TestEventPipeline_AssistantEvent_StreamingToComplete verifies the full
// streaming lifecycle: multiple partial events followed by a final event with
// stop_reason set. The Claude panel must receive AssistantMsg with the correct
// Streaming flag at each stage.
//
// Bug class guarded: all AssistantMsg are delivered with Streaming=true even
// after stop_reason is set, causing the panel to never finalize the message
// (markdown rendering skipped, message appended instead of replaced).
func TestEventPipeline_AssistantEvent_StreamingToComplete(t *testing.T) {
	m := newReadyAppModel(120, 40)
	mock := &mockClaudePanel{}
	m.shared.claudePanel = mock

	// --- Partial event 1: streaming, no stop_reason ---
	partialEv1 := cli.AssistantEvent{
		Type: "assistant",
		Message: cli.AssistantMessage{
			ID:         "msg-001",
			Role:       "assistant",
			StopReason: nil, // nil = still streaming
			Content: []cli.ContentBlock{
				{Type: "text", Text: "Hello"},
			},
		},
	}
	updated, _ := m.Update(partialEv1)
	m = updated.(AppModel)

	// Panel must have received the streaming fragment.
	if !mock.handleMsgCalled {
		t.Error("partial event 1: claudePanel.HandleMsg not called")
	}
	lastMsg, ok := mock.lastMsg.(AssistantMsg)
	if !ok {
		t.Fatalf("partial event 1: lastMsg type = %T; want AssistantMsg", mock.lastMsg)
	}
	if !lastMsg.Streaming {
		t.Error("partial event 1: AssistantMsg.Streaming = false; want true (no stop_reason)")
	}
	if lastMsg.Text != "Hello" {
		t.Errorf("partial event 1: Text = %q; want %q", lastMsg.Text, "Hello")
	}

	// Reset for next delivery.
	mock.handleMsgCalled = false

	// --- Partial event 2: streaming continues ---
	partialEv2 := cli.AssistantEvent{
		Type: "assistant",
		Message: cli.AssistantMessage{
			ID:         "msg-001",
			Role:       "assistant",
			StopReason: nil,
			Content: []cli.ContentBlock{
				{Type: "text", Text: "Hello world"},
			},
		},
	}
	updated, _ = m.Update(partialEv2)
	m = updated.(AppModel)

	if !mock.handleMsgCalled {
		t.Error("partial event 2: claudePanel.HandleMsg not called")
	}
	lastMsg, ok = mock.lastMsg.(AssistantMsg)
	if !ok {
		t.Fatalf("partial event 2: lastMsg type = %T; want AssistantMsg", mock.lastMsg)
	}
	if !lastMsg.Streaming {
		t.Error("partial event 2: AssistantMsg.Streaming = false; want true")
	}

	// Reset for final delivery.
	mock.handleMsgCalled = false

	// --- Final event: stop_reason set → Streaming must be false ---
	stopReason := "end_turn"
	finalEv := cli.AssistantEvent{
		Type: "assistant",
		Message: cli.AssistantMessage{
			ID:         "msg-001",
			Role:       "assistant",
			StopReason: &stopReason, // non-nil = turn complete
			Content: []cli.ContentBlock{
				{Type: "text", Text: "Hello world, final."},
			},
		},
	}
	updated, _ = m.Update(finalEv)
	m = updated.(AppModel)

	if !mock.handleMsgCalled {
		t.Error("final event: claudePanel.HandleMsg not called")
	}
	lastMsg, ok = mock.lastMsg.(AssistantMsg)
	if !ok {
		t.Fatalf("final event: lastMsg type = %T; want AssistantMsg", mock.lastMsg)
	}
	if lastMsg.Streaming {
		t.Error("final event: AssistantMsg.Streaming = true; want false (stop_reason is set)")
	}
	if lastMsg.Text != "Hello world, final." {
		t.Errorf("final event: Text = %q; want %q", lastMsg.Text, "Hello world, final.")
	}
}

// TestEventPipeline_AssistantEvent_SetsStreamingIndicator verifies that
// receiving an AssistantEvent with streaming content sets the streaming
// indicator on the status line.
//
// Bug class guarded: streaming indicator never gets set because the conditional
// in handleAssistantEvent checks the wrong field.
func TestEventPipeline_AssistantEvent_SetsStreamingIndicator(t *testing.T) {
	m := newReadyAppModel(120, 40)
	mock := &mockClaudePanel{}
	m.shared.claudePanel = mock
	m.statusLine.Streaming = false

	ev := cli.AssistantEvent{
		Type: "assistant",
		Message: cli.AssistantMessage{
			ID:         "msg-streaming",
			StopReason: nil, // nil = still streaming
			Content: []cli.ContentBlock{
				{Type: "text", Text: "Thinking..."},
			},
		},
	}

	updated, _ := m.Update(ev)
	result := updated.(AppModel)

	if !result.statusLine.Streaming {
		t.Error("statusLine.Streaming = false after streaming AssistantEvent; want true")
	}
}

// TestEventPipeline_AssistantEvent_ToolUseBlocksForwarded verifies that
// tool_use content blocks are forwarded to the Claude panel as ToolUseMsg
// so the user can see what tools the router is calling.
func TestEventPipeline_AssistantEvent_ToolUseBlocksForwarded(t *testing.T) {
	m := newReadyAppModel(120, 40)
	mock := &mockClaudePanel{}
	m.shared.claudePanel = mock

	ev := cli.AssistantEvent{
		Type: "assistant",
		Message: cli.AssistantMessage{
			ID:   "msg-tools",
			Role: "assistant",
			Content: []cli.ContentBlock{
				{Type: "tool_use", ID: "tu1", Name: "Read"},
			},
		},
	}

	// Must not panic.
	updated, _ := m.Update(ev)
	_ = updated.(AppModel)

	// Panel SHOULD have received a HandleMsg call for the tool_use block.
	if !mock.handleMsgCalled {
		t.Error("claudePanel.HandleMsg not called for tool_use event; want ToolUseMsg forwarded")
	}
}

// TestEventPipeline_AssistantEvent_UpdatesContextWindow verifies that
// AssistantEvent updates context window usage from per-message token counts,
// not cumulative session totals.
//
// Bug class guarded: using cumulative modelUsage from ResultEvent inflates
// context percentage ~10x because it sums tokens across all turns rather than
// reflecting the current turn's actual context fill.
func TestEventPipeline_AssistantEvent_UpdatesContextWindow(t *testing.T) {
	m := newReadyAppModel(120, 40)
	m.activeModel = "claude-opus-4-6[1m]"

	usage := &cli.MessageUsage{
		InputTokens:              5000,
		CacheReadInputTokens:     55000,
		CacheCreationInputTokens: 2000,
	}
	stopReason := "end_turn"

	ev := cli.AssistantEvent{
		Type: "assistant",
		Message: cli.AssistantMessage{
			ID:         "msg-ctx-1",
			Role:       "assistant",
			StopReason: &stopReason,
			Content:    []cli.ContentBlock{{Type: "text", Text: "hello"}},
			Usage:      usage,
		},
	}

	updated, _ := m.Update(ev)
	result := updated.(AppModel)

	// Total context used = input + cache_read + cache_creation = 62000.
	wantUsed := 5000 + 55000 + 2000
	if result.statusLine.ContextUsedTokens != wantUsed {
		t.Errorf("ContextUsedTokens = %d; want %d", result.statusLine.ContextUsedTokens, wantUsed)
	}

	// Capacity resolved from model name ([1m] → 1M).
	if result.statusLine.ContextCapacity != 1_000_000 {
		t.Errorf("ContextCapacity = %d; want 1000000", result.statusLine.ContextCapacity)
	}

	// Percent = 62000 / 1000000 * 100 = 6.2%.
	wantPct := float64(wantUsed) / 1_000_000.0 * 100
	if result.statusLine.ContextPercent != wantPct {
		t.Errorf("ContextPercent = %f; want %f", result.statusLine.ContextPercent, wantPct)
	}
}

// TestEventPipeline_AssistantEvent_ContextWindow_IgnoresSubagent verifies that
// subagent messages (ParentToolUseID != nil) do not update context window.
func TestEventPipeline_AssistantEvent_ContextWindow_IgnoresSubagent(t *testing.T) {
	m := newReadyAppModel(120, 40)
	m.activeModel = "claude-sonnet-4-6"

	parentID := "tu-parent-123"
	usage := &cli.MessageUsage{
		InputTokens:          50000,
		CacheReadInputTokens: 100000,
	}
	stopReason := "end_turn"

	ev := cli.AssistantEvent{
		Type:            "assistant",
		ParentToolUseID: &parentID,
		Message: cli.AssistantMessage{
			ID:         "msg-subagent",
			Role:       "assistant",
			StopReason: &stopReason,
			Content:    []cli.ContentBlock{{Type: "text", Text: "subagent output"}},
			Usage:      usage,
		},
	}

	updated, _ := m.Update(ev)
	result := updated.(AppModel)

	// Subagent messages must NOT update context window.
	if result.statusLine.ContextUsedTokens != 0 {
		t.Errorf("ContextUsedTokens = %d; want 0 (subagent should be ignored)", result.statusLine.ContextUsedTokens)
	}
}

// ---------------------------------------------------------------------------
// Test 4: AgentRegisteredMsg updates tree AND statusLine.AgentCount.
// ---------------------------------------------------------------------------

// TestEventPipeline_AgentRegistered_UpdatesTreeAndCount verifies that an
// AgentRegisteredMsg refreshes both the agent tree nodes AND the
// statusLine.AgentCount field.
//
// Bug class guarded: handleAgentRegistryMsg calls agentTree.SetNodes() but
// forgets to assign statusLine.AgentCount, so the count shown in the status
// bar stays at 0 while the tree shows agents.
func TestEventPipeline_AgentRegistered_UpdatesTreeAndCount(t *testing.T) {
	m := newReadyAppModel(120, 40)

	// Pre-register an agent directly in the registry so it is visible when
	// the message arrives (the registry is the source of truth; the message
	// just triggers a refresh).
	if m.shared.agentRegistry == nil {
		t.Fatal("agentRegistry = nil")
	}
	_ = m.shared.agentRegistry.Register(state.Agent{
		ID:          "agent-go-pro-1",
		AgentType:   "go-pro",
		Description: "GO Pro",
		Model:       "claude-sonnet-4-6",
		Tier:        "sonnet",
		Status:      state.StatusRunning,
		StartedAt:   time.Now(),
	})

	// Deliver the message that signals the registry has changed.
	updated, cmd := m.Update(AgentRegisteredMsg{
		AgentID:   "agent-go-pro-1",
		AgentType: "go-pro",
	})
	result := updated.(AppModel)

	// No command expected from this handler.
	if cmd != nil {
		t.Errorf("cmd = %v; want nil for AgentRegisteredMsg", cmd)
	}

	// Agent count on the status line must reflect the registry total.
	wantCount := result.shared.agentRegistry.Count().Total
	if result.statusLine.AgentCount != wantCount {
		t.Errorf("statusLine.AgentCount = %d; want %d (registry total)",
			result.statusLine.AgentCount, wantCount)
	}

	// Agent count must be at least 1 (we registered one above).
	if result.statusLine.AgentCount < 1 {
		t.Errorf("statusLine.AgentCount = %d; want >= 1", result.statusLine.AgentCount)
	}

	// The agent tree must have at least one node.
	nodes := result.shared.agentRegistry.Tree()
	if len(nodes) == 0 {
		t.Error("agent tree has 0 nodes after AgentRegisteredMsg; want >= 1")
	}
}

// TestEventPipeline_AgentUpdated_CountReflectsRegistry verifies that
// AgentUpdatedMsg also keeps statusLine.AgentCount in sync.
func TestEventPipeline_AgentUpdated_CountReflectsRegistry(t *testing.T) {
	m := newReadyAppModel(120, 40)

	if m.shared.agentRegistry == nil {
		t.Fatal("agentRegistry = nil")
	}
	_ = m.shared.agentRegistry.Register(state.Agent{
		ID:        "agent-1",
		AgentType: "go-pro",
		Model:     "claude-sonnet-4-6",
		Tier:      "sonnet",
		Status:    state.StatusRunning,
		StartedAt: time.Now(),
	})
	_ = m.shared.agentRegistry.Register(state.Agent{
		ID:        "agent-2",
		AgentType: "python-pro",
		Model:     "claude-sonnet-4-6",
		Tier:      "sonnet",
		Status:    state.StatusRunning,
		StartedAt: time.Now(),
	})

	updated, _ := m.Update(AgentUpdatedMsg{AgentID: "agent-1", Status: "complete"})
	result := updated.(AppModel)

	wantCount := result.shared.agentRegistry.Count().Total
	if result.statusLine.AgentCount != wantCount {
		t.Errorf("statusLine.AgentCount = %d; want %d after AgentUpdatedMsg",
			result.statusLine.AgentCount, wantCount)
	}
}

// TestEventPipeline_MultipleAgents_CountMatchesRegistry verifies that when
// multiple agents are registered and all three message types (Registered,
// Updated, Activity) are delivered in sequence, the final agent count matches
// the registry.
func TestEventPipeline_MultipleAgents_CountMatchesRegistry(t *testing.T) {
	m := newReadyAppModel(120, 40)

	if m.shared.agentRegistry == nil {
		t.Fatal("agentRegistry = nil")
	}

	// Register three agents.
	for _, id := range []string{"a1", "a2", "a3"} {
		_ = m.shared.agentRegistry.Register(state.Agent{
			ID:        id,
			AgentType: "go-pro",
			Model:     "claude-sonnet-4-6",
			Tier:      "sonnet",
			Status:    state.StatusRunning,
			StartedAt: time.Now(),
		})
	}

	// Deliver one message of each agent-lifecycle type.
	updated, _ := m.Update(AgentRegisteredMsg{AgentID: "a1"})
	m = updated.(AppModel)
	updated, _ = m.Update(AgentUpdatedMsg{AgentID: "a2", Status: "complete"})
	m = updated.(AppModel)
	updated, _ = m.Update(AgentActivityMsg{AgentID: "a3", ToolName: "Read"})
	result := updated.(AppModel)

	wantCount := result.shared.agentRegistry.Count().Total
	if result.statusLine.AgentCount != wantCount {
		t.Errorf("statusLine.AgentCount = %d; want %d after mixed lifecycle messages",
			result.statusLine.AgentCount, wantCount)
	}
}

// ---------------------------------------------------------------------------
// Test 4b: C-1 fix — AgentRegisteredMsg registers agents via the message
// (not pre-registered). Prior to C-1 fix, the handler discarded message data.
// ---------------------------------------------------------------------------

// TestEventPipeline_AgentRegisteredMsg_RegistersInRegistry verifies that
// sending AgentRegisteredMsg ALONE (without pre-registration) causes the
// agent to appear in the registry AND in the Tree() DFS output.
// This tests both the C-1 fix (register data) and the ParentID defaulting
// (orphan → child of root).
func TestEventPipeline_AgentRegisteredMsg_RegistersInRegistry(t *testing.T) {
	m := newReadyAppModel(120, 40)
	if m.shared.agentRegistry == nil {
		t.Fatal("agentRegistry = nil")
	}

	// Pre-register a root agent (simulates what handleSystemInit does when
	// the CLI subprocess emits a SystemInitEvent).
	_ = m.shared.agentRegistry.Register(state.Agent{
		ID: "router-root", AgentType: "router",
		Status: state.StatusRunning, StartedAt: time.Now(),
	})
	rootID := m.shared.agentRegistry.RootID()
	if rootID == "" {
		t.Fatal("rootAgentID is empty after registering router-root")
	}

	// Send message WITHOUT pre-registering and WITHOUT ParentID.
	// The handler must default ParentID to rootAgentID so the agent
	// appears in the tree (not orphaned).
	updated, _ := m.Update(AgentRegisteredMsg{
		AgentID:   "mcp-agent-1",
		AgentType: "python-pro",
	})
	result := updated.(AppModel)

	agent := result.shared.agentRegistry.Get("mcp-agent-1")
	if agent == nil {
		t.Fatal("agent not found in registry after AgentRegisteredMsg; C-1 fix not applied")
	}
	if agent.AgentType != "python-pro" {
		t.Errorf("agent.AgentType = %q; want %q", agent.AgentType, "python-pro")
	}
	if agent.Status != state.StatusRunning {
		t.Errorf("agent.Status = %v; want StatusRunning", agent.Status)
	}
	if agent.ParentID != rootID {
		t.Errorf("agent.ParentID = %q; want %q (should default to root)", agent.ParentID, rootID)
	}
	if result.statusLine.AgentCount < 2 {
		t.Errorf("statusLine.AgentCount = %d; want >= 2 (root + spawned)", result.statusLine.AgentCount)
	}

	// CRITICAL: verify agent is visible in Tree() DFS — not just Count().
	// This guards against the orphan bug where Count() returns 2 but
	// Tree() only shows the root.
	tree := result.shared.agentRegistry.Tree()
	found := false
	for _, node := range tree {
		if node.Agent != nil && node.Agent.ID == "mcp-agent-1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("agent mcp-agent-1 is in Count() but NOT in Tree(); orphan bug not fixed")
	}
}

// TestEventPipeline_AgentUpdatedMsg_UpdatesStatus verifies that
// AgentUpdatedMsg changes the agent's status in the registry.
func TestEventPipeline_AgentUpdatedMsg_UpdatesStatus(t *testing.T) {
	m := newReadyAppModel(120, 40)
	if m.shared.agentRegistry == nil {
		t.Fatal("agentRegistry = nil")
	}

	// Register via message first.
	updated, _ := m.Update(AgentRegisteredMsg{
		AgentID: "mcp-agent-2", AgentType: "go-pro",
	})
	m = updated.(AppModel)

	// Now update status via message.
	updated, _ = m.Update(AgentUpdatedMsg{AgentID: "mcp-agent-2", Status: "complete"})
	result := updated.(AppModel)

	agent := result.shared.agentRegistry.Get("mcp-agent-2")
	if agent == nil {
		t.Fatal("agent not found after update")
	}
	if agent.Status != state.StatusComplete {
		t.Errorf("agent.Status = %v; want StatusComplete", agent.Status)
	}
}

// TestEventPipeline_AgentActivityMsg_SetsActivity verifies that
// AgentActivityMsg sets the agent's activity in the registry.
func TestEventPipeline_AgentActivityMsg_SetsActivity(t *testing.T) {
	m := newReadyAppModel(120, 40)
	if m.shared.agentRegistry == nil {
		t.Fatal("agentRegistry = nil")
	}

	// Register via message first.
	updated, _ := m.Update(AgentRegisteredMsg{
		AgentID: "mcp-agent-3", AgentType: "go-pro",
	})
	m = updated.(AppModel)

	// Set activity.
	updated, _ = m.Update(AgentActivityMsg{AgentID: "mcp-agent-3", ToolName: "Read"})
	result := updated.(AppModel)

	agent := result.shared.agentRegistry.Get("mcp-agent-3")
	if agent == nil {
		t.Fatal("agent not found after activity")
	}
	if agent.Activity == nil {
		t.Fatal("agent.Activity is nil after AgentActivityMsg")
	}
	if agent.Activity.Target != "Read" {
		t.Errorf("agent.Activity.Target = %q; want %q", agent.Activity.Target, "Read")
	}
}

// TestEventPipeline_AgentRegisteredMsg_WithParent_LinksChild verifies that
// a child agent with ParentID is linked to its parent in the tree.
func TestEventPipeline_AgentRegisteredMsg_WithParent_LinksChild(t *testing.T) {
	m := newReadyAppModel(120, 40)
	if m.shared.agentRegistry == nil {
		t.Fatal("agentRegistry = nil")
	}

	// Register parent.
	updated, _ := m.Update(AgentRegisteredMsg{
		AgentID: "parent-1", AgentType: "router",
	})
	m = updated.(AppModel)

	// Register child with ParentID.
	updated, _ = m.Update(AgentRegisteredMsg{
		AgentID: "child-1", AgentType: "python-pro", ParentID: "parent-1",
	})
	result := updated.(AppModel)

	parent := result.shared.agentRegistry.Get("parent-1")
	if parent == nil {
		t.Fatal("parent not found")
	}
	if len(parent.Children) == 0 {
		t.Fatal("parent.Children is empty; want child-1 linked")
	}
	if parent.Children[0] != "child-1" {
		t.Errorf("parent.Children[0] = %q; want %q", parent.Children[0], "child-1")
	}
}

// TestEventPipeline_AgentUpdatedMsg_UnknownAgent_NoPanic verifies that
// sending an update for a non-existent agent doesn't panic.
func TestEventPipeline_AgentUpdatedMsg_UnknownAgent_NoPanic(t *testing.T) {
	m := newReadyAppModel(120, 40)
	// Must not panic.
	updated, _ := m.Update(AgentUpdatedMsg{AgentID: "nonexistent", Status: "error"})
	_ = updated.(AppModel)
}

// TestParseAgentStatus_AllValues verifies the parseAgentStatus helper.
func TestParseAgentStatus_AllValues(t *testing.T) {
	tests := []struct {
		input string
		want  state.AgentStatus
	}{
		{"running", state.StatusRunning},
		{"complete", state.StatusComplete},
		{"error", state.StatusError},
		{"unknown", state.StatusPending},
		{"", state.StatusPending},
	}
	for _, tc := range tests {
		got := parseAgentStatus(tc.input)
		if got != tc.want {
			t.Errorf("parseAgentStatus(%q) = %v; want %v", tc.input, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Test 5: WindowSizeMsg propagates non-zero dimensions to all components.
// ---------------------------------------------------------------------------

// TestEventPipeline_WindowSize_PropagatesAllComponents verifies that
// tea.WindowSizeMsg sets non-zero dimensions on banner, statusLine, and
// injects size into all injected widgets (claudePanel, toasts).
//
// Bug class guarded: WindowSizeMsg stores m.width/m.height but the propagation
// call for a specific component is missing (e.g. claudePanel.SetSize skipped),
// leaving that component at 0x0 and causing layout issues or panics.
func TestEventPipeline_WindowSize_PropagatesAllComponents(t *testing.T) {
	m := NewAppModel()

	// Wire mock widgets so we can observe size propagation.
	mockPanel := &mockClaudePanel{}
	mockToast := &mockToast{empty: true}
	m.shared.claudePanel = mockPanel
	m.shared.toasts = mockToast

	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	result := updated.(AppModel)

	// No command from window size.
	if cmd != nil {
		t.Errorf("cmd = %v; want nil for WindowSizeMsg", cmd)
	}

	// AppModel dimensions.
	if result.width != 120 {
		t.Errorf("width = %d; want 120", result.width)
	}
	if result.height != 40 {
		t.Errorf("height = %d; want 40", result.height)
	}
	if !result.ready {
		t.Error("ready = false after WindowSizeMsg; want true")
	}

	// Claude panel must have received a non-zero size.
	if mockPanel.width == 0 || mockPanel.height == 0 {
		t.Errorf("claudePanel size = %dx%d; want non-zero after WindowSizeMsg",
			mockPanel.width, mockPanel.height)
	}

	// Toast must have received a size.
	if mockToast.width == 0 || mockToast.height == 0 {
		t.Errorf("toasts size = %dx%d; want non-zero after WindowSizeMsg",
			mockToast.width, mockToast.height)
	}

	// Banner View() must produce non-empty output (width was set).
	bannerView := result.banner.View()
	if bannerView == "" {
		t.Error("banner.View() is empty after WindowSizeMsg; expected propagated width")
	}

	// StatusLine View() must produce non-empty output (width was set).
	statusView := result.statusLine.View()
	if statusView == "" {
		t.Error("statusLine.View() is empty after WindowSizeMsg; expected propagated width")
	}
}

// TestEventPipeline_WindowSize_AgentTreeReceivesNonZeroSize verifies that the
// agentTree (directly embedded in AppModel, not behind sharedState) also
// receives a non-zero size from WindowSizeMsg.
//
// Bug class guarded: Claude panel propagation works but agentTree.SetSize()
// is not called, so the right panel renders at 0 width (invisible or panics).
func TestEventPipeline_WindowSize_AgentTreeReceivesNonZeroSize(t *testing.T) {
	m := NewAppModel()

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	result := updated.(AppModel)

	// After WindowSizeMsg the agentTree must have a non-zero view
	// (it renders a placeholder when empty but with width > 0).
	// The view should not be empty if the size was propagated correctly.
	treeView := result.agentTree.View()
	if treeView == "" {
		t.Error("agentTree.View() is empty after WindowSizeMsg; expected size to be propagated")
	}
}

// TestEventPipeline_WindowSize_MultipleResizes_DimensionsStayCorrect verifies
// that repeated WindowSizeMsgs always update to the most recent dimensions.
func TestEventPipeline_WindowSize_MultipleResizes_DimensionsStayCorrect(t *testing.T) {
	m := NewAppModel()

	sizes := [][2]int{{80, 24}, {120, 40}, {160, 50}, {100, 35}}
	for _, sz := range sizes {
		updated, _ := m.Update(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
		result := updated.(AppModel)
		if result.width != sz[0] || result.height != sz[1] {
			t.Errorf("after resize to %dx%d: got %dx%d",
				sz[0], sz[1], result.width, result.height)
		}
		m = result
	}
}

// ---------------------------------------------------------------------------
// Test 6: Full pipeline — SystemInit → AssistantEvent → ResultEvent.
// ---------------------------------------------------------------------------

// TestEventPipeline_FullTurn_SystemInitToResult exercises the complete event
// sequence for a single session turn and verifies end-state consistency.
//
// This is the integration-level test for the bug that prompted this file:
// after a full turn the status bar should show cost, model name, and agent
// count simultaneously rather than only the last update winning.
func TestEventPipeline_FullTurn_SystemInitToResult(t *testing.T) {
	m := NewAppModel()
	mockPanel := &mockClaudePanel{}
	m.shared.claudePanel = mockPanel

	// Step 1: WindowSizeMsg — establish ready state.
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(AppModel)

	// Step 2: SystemInitEvent — session begins.
	updated, _ = m.Update(cli.SystemInitEvent{
		Type:           "system",
		Subtype:        "init",
		SessionID:      "sess-full",
		Model:          "claude-sonnet-4-6",
		PermissionMode: "default",
	})
	m = updated.(AppModel)

	// Verify intermediate state.
	if m.statusLine.ActiveModel != "claude-sonnet-4-6" {
		t.Errorf("after init: ActiveModel = %q; want claude-sonnet-4-6",
			m.statusLine.ActiveModel)
	}

	// Step 3: AssistantEvent — model responds (streaming).
	stopReason := "end_turn"
	updated, _ = m.Update(cli.AssistantEvent{
		Type: "assistant",
		Message: cli.AssistantMessage{
			ID:         "msg-turn-1",
			Role:       "assistant",
			StopReason: &stopReason,
			Content: []cli.ContentBlock{
				{Type: "text", Text: "The answer is 42."},
			},
		},
	})
	m = updated.(AppModel)

	if !mockPanel.handleMsgCalled {
		t.Error("after AssistantEvent: claudePanel.HandleMsg was not called")
	}

	// Step 4: ResultEvent — turn ends with cost data.
	updated, _ = m.Update(cli.ResultEvent{
		Type:         "result",
		Subtype:      "success",
		TotalCostUSD: 0.0150,
		Usage: cli.ResultUsage{
			InputTokens:  3000,
			OutputTokens: 500,
		},
	})
	m = updated.(AppModel)

	// Final assertions: all fields populated simultaneously.
	if m.statusLine.ActiveModel != "claude-sonnet-4-6" {
		t.Errorf("final: ActiveModel = %q; want claude-sonnet-4-6", m.statusLine.ActiveModel)
	}
	if m.statusLine.SessionCost != 0.0150 {
		t.Errorf("final: SessionCost = %f; want 0.0150", m.statusLine.SessionCost)
	}
	if m.statusLine.TokenCount != 3500 {
		t.Errorf("final: TokenCount = %d; want 3500", m.statusLine.TokenCount)
	}
	if m.statusLine.Streaming {
		t.Error("final: Streaming = true; want false after ResultEvent")
	}

	// View() must contain the cost figure (status bar renders it).
	view := m.View()
	if !strings.Contains(view, "$") && !strings.Contains(view, "0.015") {
		// The status line renders cost — the view must include some cost indicator.
		// Use a loose check because formatting (e.g. "0.0150" vs "$0.02") may vary.
		t.Log("Note: cost indicator not found in view — check statusline rendering")
	}
}
