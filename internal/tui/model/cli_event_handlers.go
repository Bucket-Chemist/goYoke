// Package model defines shared state types for the GOgent-Fortress TUI.
// This file contains all CLI event handlers for AppModel's Update method.
// Extracted from app.go as part of TUI-043.
package model

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/cli"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
)

// planStepPattern matches common plan step markers in assistant text.
// Supported forms:
//   - "Step N of M" / "Step N/M"
//   - "Phase N of M" / "Phase N/M"
//   - "step N of M" (case-insensitive via regexp flag (?i))
var planStepPattern = regexp.MustCompile(`(?i)\b(?:step|phase)\s+(\d+)\s*(?:of|/)\s*(\d+)\b`)

// parsePlanStep attempts to extract a (step, total) pair from text.
// Returns (0, 0) when no step marker is found.
func parsePlanStep(text string) (step, total int) {
	m := planStepPattern.FindStringSubmatch(text)
	if m == nil {
		return 0, 0
	}
	s, err1 := strconv.Atoi(m[1])
	t, err2 := strconv.Atoi(m[2])
	if err1 != nil || err2 != nil {
		return 0, 0
	}
	return s, t
}

// resolveContextWindow derives context window capacity from a model ID string.
// Used as a fallback before the first ResultEvent reports the actual capacity.
// Mirrors packages/tui/src/utils/resolveContextWindow.ts.
func resolveContextWindow(modelID string) int {
	if strings.Contains(modelID, "[1m]") {
		return 1_000_000
	}
	return 200_000
}

// handleCLIStarted handles cli.CLIStartedMsg: the subprocess started — begin
// listening for NDJSON events.
func (m AppModel) handleCLIStarted() (tea.Model, tea.Cmd) {
	return m, m.waitForCLIEvent()
}

// handleSystemInit handles cli.SystemInitEvent: the CLI session is ready;
// records session metadata and registers the root router agent.
func (m AppModel) handleSystemInit(msg cli.SystemInitEvent) (tea.Model, tea.Cmd) {
	m.cliReady = true
	m.sessionID = msg.SessionID
	m.activeModel = msg.Model
	// Latch the 1M context flag on the first session init so it survives
	// model switches. Once set, it is never cleared — the user's account
	// capability does not change mid-session.
	if strings.Contains(msg.Model, "[1m]") {
		m.context1M = true
	}
	// Persist session ID to active provider for resume support (TUI-031).
	if m.shared != nil && m.shared.providerState != nil && msg.SessionID != "" {
		m.shared.providerState.SetSessionID(msg.SessionID)
	}
	// Sync status line with session metadata.
	m.statusLine.ActiveModel = msg.Model
	m.statusLine.PermissionMode = msg.PermissionMode
	if m.shared != nil && m.shared.providerState != nil {
		m.statusLine.Provider = string(m.shared.providerState.GetActiveProvider())
	}
	if m.statusLine.SessionStart.IsZero() {
		m.statusLine.SessionStart = time.Now()
	}

	// Register the root "Router" agent so the agent tree shows the
	// session immediately (matching Node.js TUI behaviour).
	if m.shared != nil && m.shared.agentRegistry != nil {
		tier := "sonnet"
		modelLower := strings.ToLower(msg.Model)
		if strings.Contains(modelLower, "haiku") {
			tier = "haiku"
		} else if strings.Contains(modelLower, "opus") {
			tier = "opus"
		}
		_ = m.shared.agentRegistry.Register(state.Agent{
			ID:          "router-root",
			AgentType:   "router",
			Description: "Router",
			Model:       msg.Model,
			Tier:        tier,
			Status:      state.StatusRunning,
			StartedAt:   time.Now(),
		})
		m.shared.agentRegistry.InvalidateTreeCache()
		m.agentTree.SetNodes(m.shared.agentRegistry.Tree())
		// Auto-select the root agent in the detail panel so the right panel
		// shows live data immediately instead of "Select an agent".
		if id := m.agentTree.SelectedID(); id != "" {
			if agent := m.shared.agentRegistry.Get(id); agent != nil {
				m.agentDetail.SetAgent(agent)
			}
		}
	}

	return m, m.waitForCLIEvent()
}

// handleAssistantEvent handles cli.AssistantEvent: forwards text content to
// the Claude panel and syncs the agent registry from Task tool_use blocks.
func (m AppModel) handleAssistantEvent(msg cli.AssistantEvent) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var rootActivityAppended bool

	// Forward text and tool_use content to Claude panel.
	if m.shared.claudePanel != nil {
		for _, block := range msg.Message.Content {
			switch {
			case block.Type == "text" && block.Text != "":
				streaming := msg.Message.StopReason == nil
				cmd := m.shared.claudePanel.HandleMsg(AssistantMsg{
					Text:      block.Text,
					Streaming: streaming,
				})
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			case block.Type == "tool_use" && block.Name != "":
				activity := cli.ExtractToolActivity(block)
				if msg.ParentToolUseID == nil {
					// Root-level tool_use: route to agent registry activity panel.
					if m.shared.agentRegistry != nil {
						m.shared.agentRegistry.AppendActivity("router-root", activity)
						rootActivityAppended = true
					}
				} else {
					// Subagent tool_use: show inline in conversation panel.
					cmd := m.shared.claudePanel.HandleMsg(ToolUseMsg{
						ToolName: block.Name,
						ToolID:   block.ID,
						Input:    activity.Target,
					})
					if cmd != nil {
						cmds = append(cmds, cmd)
					}
				}
			}
		}
	}

	// Wire TodoWrite tool_use blocks to the TaskBoard so the todo overlay
	// reflects the current task list. The last TodoWrite wins (matches the
	// TS TUI's useTaskBoardData hook pattern).
	if m.shared.taskBoard != nil {
		for _, block := range msg.Message.Content {
			if block.Type == "tool_use" && block.Name == "TodoWrite" && len(block.Input) > 0 {
				m.syncTaskBoard(block.Input)
			}
		}
	}

	// Wire plan mode tool calls to the plan preview panel and drawer.
	// EnterPlanMode → activate plan mode indicator on status line.
	// ExitPlanMode  → read the plan file from disk, populate the plan
	//                 preview widget + drawer, switch right panel to show it.
	// Mirrors TS TUI SessionManager.ts:1117-1230.
	for _, block := range msg.Message.Content {
		if block.Type != "tool_use" {
			continue
		}
		switch block.Name {
		case "EnterPlanMode":
			m.statusLine.PlanActive = true

		case "ExitPlanMode":
			m.syncPlanPreview()
			m.statusLine.PlanActive = false
		}
	}

	// Sync agent registry from Task tool_use blocks.
	if m.shared.agentRegistry != nil {
		result := cli.SyncAssistantEvent(msg, m.shared.agentRegistry)
		if len(result.Registered) > 0 || len(result.Activity) > 0 || rootActivityAppended {
			// C-3: invalidate before reading Tree() so the view reflects
			// the mutations that SyncAssistantEvent just applied.
			m.shared.agentRegistry.InvalidateTreeCache()
			m.agentTree.SetNodes(m.shared.agentRegistry.Tree())
			// Sync detail panel with the currently highlighted tree node so
			// that activity updates are reflected without requiring navigation.
			if id := m.agentTree.SelectedID(); id != "" {
				if agent := m.shared.agentRegistry.Get(id); agent != nil {
					m.agentDetail.SetAgent(agent)
				}
			}
		}
	}

	// Update streaming indicator: if content is present and stop_reason is
	// nil, the assistant is still generating (streaming=true).
	if len(msg.Message.Content) > 0 {
		streaming := msg.Message.StopReason == nil
		if streaming && !m.statusLine.Streaming {
			cmd := m.statusLine.SetStreaming(true)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	// Detect plan step markers in streaming text and emit PlanStepMsg when
	// a "Step N of M" / "Phase N/M" pattern is found (TUI-057).
	for _, block := range msg.Message.Content {
		if block.Type == "text" && block.Text != "" {
			if step, total := parsePlanStep(block.Text); total > 0 {
				cmds = append(cmds, func() tea.Msg {
					return PlanStepMsg{Active: true, Step: step, Total: total}
				})
			}
		}
	}

	// Update context window usage from per-message token counts.
	// Only root-level messages — subagent usage reflects a separate CLI context.
	// Mirrors TS TUI SessionManager.ts:934-944.
	if msg.Message.Usage != nil && msg.ParentToolUseID == nil {
		usage := msg.Message.Usage
		used := usage.InputTokens + usage.CacheReadInputTokens + usage.CacheCreationInputTokens
		capacity := m.statusLine.ContextCapacity
		if capacity == 0 {
			capacity = resolveContextWindow(m.activeModel)
		}
		if capacity > 0 {
			pct := float64(used) / float64(capacity) * 100
			if pct > 100 {
				pct = 100
			}
			m.statusLine.ContextPercent = pct
			m.statusLine.ContextUsedTokens = used
			m.statusLine.ContextCapacity = capacity
		}
	}

	cmds = append(cmds, m.waitForCLIEvent())
	return m, tea.Batch(cmds...)
}

// handleUserEvent handles cli.UserEvent: extracts post-hoc diffs, syncs the
// agent registry from tool_result blocks, and shows the streaming indicator.
func (m AppModel) handleUserEvent(msg cli.UserEvent) (tea.Model, tea.Cmd) {
	// Extract post-hoc diffs.
	m = m.extractDiffs(msg)

	// Dispatch tool_result blocks: root-level → agent registry; subagent → Claude panel.
	var rootResultUpdated bool
	if m.shared != nil {
		for _, block := range msg.Message.Content {
			if block.Type != "tool_result" || block.ToolUseID == "" {
				continue
			}
			if msg.ParentToolUseID == nil {
				// Root-level tool_result: update activity entry in agent registry.
				if m.shared.agentRegistry != nil {
					m.shared.agentRegistry.UpdateActivityResult("router-root", block.ToolUseID, !block.IsError)
					rootResultUpdated = true
				}
			} else if m.shared.claudePanel != nil {
				// Subagent tool_result: update conversation panel block.
				m.shared.claudePanel.HandleMsg(ToolResultMsg{
					ToolID:  block.ToolUseID,
					Success: !block.IsError,
				})
			}
		}
	}

	// Sync agent registry from tool_result blocks.
	if m.shared.agentRegistry != nil {
		result := cli.SyncUserEvent(msg, m.shared.agentRegistry)
		if len(result.Updated) > 0 || len(result.Activity) > 0 || rootResultUpdated {
			// C-3: invalidate before reading Tree() so the view reflects
			// the mutations that SyncUserEvent just applied.
			m.shared.agentRegistry.InvalidateTreeCache()
			m.agentTree.SetNodes(m.shared.agentRegistry.Tree())
			// Sync detail panel with the currently highlighted tree node so
			// that status changes (complete / error) are reflected immediately.
			if id := m.agentTree.SelectedID(); id != "" {
				if agent := m.shared.agentRegistry.Get(id); agent != nil {
					m.agentDetail.SetAgent(agent)
				}
			}
		}
	}

	// The CLI has echoed back a user message — the assistant is about to
	// respond. Show the thinking indicator if not already streaming.
	if !m.statusLine.Streaming {
		cmd := m.statusLine.SetStreaming(true)
		if cmd != nil {
			return m, tea.Batch(m.waitForCLIEvent(), cmd)
		}
	}

	return m, m.waitForCLIEvent()
}

// handleResultEvent handles cli.ResultEvent: updates cost, token counts,
// context window percentage, clears streaming, finalises the Claude panel turn,
// and schedules a debounced session auto-save.
func (m AppModel) handleResultEvent(msg cli.ResultEvent) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Update cost tracker (single source of truth).
	if m.shared.costTracker != nil {
		m.shared.costTracker.UpdateSessionCost(msg.TotalCostUSD)
	}
	m.statusLine.SessionCost = msg.TotalCostUSD

	// Accumulate session token counts from aggregate usage.
	m.statusLine.TokenCount += msg.Usage.InputTokens + msg.Usage.OutputTokens

	// Update context window capacity from per-model usage if available.
	// The actual context usage (used tokens, percent) is updated per-message in
	// handleAssistantEvent — here we only learn/refine the capacity.
	if entry, ok := msg.ModelUsage[m.activeModel]; ok && entry.ContextWindow > 0 {
		capacity := entry.ContextWindow
		if strings.Contains(m.activeModel, "[1m]") {
			capacity = 1_000_000
		}
		m.statusLine.ContextCapacity = capacity
	}

	// Clear streaming indicator — the turn is complete.
	m.statusLine.Streaming = false

	// Forward to Claude panel to finalize streaming.
	if m.shared.claudePanel != nil {
		cmd := m.shared.claudePanel.HandleMsg(ResultMsg{
			SessionID:  msg.SessionID,
			CostUSD:    msg.TotalCostUSD,
			DurationMS: msg.DurationMS,
		})
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Schedule debounced session auto-save (5 s cooldown, TUI-033).
	m.autoSaveSeq++
	seq := m.autoSaveSeq
	cmds = append(cmds, tea.Tick(5*time.Second, func(_ time.Time) tea.Msg {
		return SessionAutoSaveMsg{Seq: seq}
	}))

	cmds = append(cmds, m.waitForCLIEvent())
	return m, tea.Batch(cmds...)
}

// handleCLIDisconnected handles cli.CLIDisconnectedMsg: attempts reconnection
// up to maxReconnectAttempts times, then remains disconnected.
func (m AppModel) handleCLIDisconnected(msg cli.CLIDisconnectedMsg) (tea.Model, tea.Cmd) {
	// Subprocess exited or pipe broken — attempt reconnection.
	if msg.Err != nil && m.reconnectCount < maxReconnectAttempts {
		m.reconnectCount++
		return m, reconnectAfterDelay(m.reconnectCount, m.reconnectSeq)
	}
	// Exceeded retries or clean exit — remain disconnected.
	return m, nil
}

// handleCLIReconnect handles CLIReconnectMsg: discards stale timers created
// before the last provider switch, then restarts the CLI driver.
func (m AppModel) handleCLIReconnect(msg CLIReconnectMsg) (tea.Model, tea.Cmd) {
	// Discard stale timers created before the last provider switch.
	if msg.Seq != m.reconnectSeq {
		return m, nil
	}
	return m, m.startCLI()
}

// extractDiffs inspects a UserEvent for tool_use_result blocks that carry a
// structuredPatch field and appends any found patches to m.diffs.
// This implements the post-hoc diff display path for Write/Edit/Bash tools
// (Path 1 of Option D hybrid permission flow).
func (m AppModel) extractDiffs(ev cli.UserEvent) AppModel {
	if len(ev.ToolUseResult) == 0 {
		return m
	}

	// tool_use_result can be a single object or an array of objects.
	// Try single object first.
	var single toolUseResultWithPatch
	if err := json.Unmarshal(ev.ToolUseResult, &single); err == nil && single.FilePath != "" {
		if len(single.StructuredPatch) > 0 {
			m.diffs = append(m.diffs, DiffEntry{
				FilePath: single.FilePath,
				Patch:    single.StructuredPatch,
			})
		}
		return m
	}

	// Try array variant.
	var many []toolUseResultWithPatch
	if err := json.Unmarshal(ev.ToolUseResult, &many); err == nil {
		for _, r := range many {
			if r.FilePath != "" && len(r.StructuredPatch) > 0 {
				m.diffs = append(m.diffs, DiffEntry{
					FilePath: r.FilePath,
					Patch:    r.StructuredPatch,
				})
			}
		}
	}
	return m
}

// toolUseResultWithPatch is a partial unmarshal target for the ToolUseResult
// JSON field on cli.UserEvent.  Only the fields relevant to diff extraction
// are decoded; all other fields are ignored.
type toolUseResultWithPatch struct {
	FilePath        string          `json:"filePath"`
	StructuredPatch json.RawMessage `json:"structuredPatch,omitempty"`
}

// syncTaskBoard parses the TodoWrite input JSON and feeds the resulting
// task entries to the TaskBoard widget. The input is expected to be:
//
//	{"todos": [{"content":"...","status":"...","activeForm":"..."}]}
//
// Malformed or empty input is silently ignored.
func (m *AppModel) syncTaskBoard(input json.RawMessage) {
	var payload struct {
		Todos []struct {
			Content    string `json:"content"`
			Status     string `json:"status"`
			ActiveForm string `json:"activeForm"`
		} `json:"todos"`
	}
	if err := json.Unmarshal(input, &payload); err != nil {
		return
	}
	if len(payload.Todos) == 0 {
		return
	}
	entries := make([]state.TaskEntry, len(payload.Todos))
	for i, t := range payload.Todos {
		entries[i] = state.TaskEntry{
			ID:         strconv.Itoa(i + 1),
			Content:    t.Content,
			Status:     t.Status,
			ActiveForm: t.ActiveForm,
		}
	}
	m.shared.taskBoard.SetTasks(entries)
	// Taskboard height changes when tasks arrive (auto-show) or are cleared.
	// Re-propagate layout so the Claude panel and other content components
	// shrink to accommodate the new taskboard height.
	m.propagateContentSizes()
}

// syncPlanPreview reads the most recently modified .claude/plans/*.md file
// and populates the plan drawer. The right panel stays on agents view —
// the drawer is the correct place for plan content. Alt+V opens the
// full-screen plan modal for detailed viewing.
// Called when an ExitPlanMode tool_use block is detected in the assistant
// event stream.
func (m *AppModel) syncPlanPreview() {
	planPath := findLatestPlanFile()
	if planPath == "" {
		return
	}
	data, err := os.ReadFile(planPath)
	if err != nil || len(data) == 0 {
		return
	}
	content := string(data)
	// Populate the plan preview for Alt+V full-screen modal.
	if m.shared.planPreview != nil {
		m.shared.planPreview.SetContent(content)
	}
	// Populate the plan drawer — auto-expands to show the plan.
	// The right panel stays on agents (RPMAgents), not switched.
	if m.shared.drawerStack != nil {
		m.shared.drawerStack.SetPlanContent(content)
	}
}

// findLatestPlanFile scans ~/.claude/plans/ for the most recently modified
// .md file and returns its absolute path. Returns "" if no plan files exist.
func findLatestPlanFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	dir := filepath.Join(home, ".claude", "plans")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var latest string
	var latestTime time.Time
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
			latest = filepath.Join(dir, e.Name())
		}
	}
	return latest
}

// waitForCLIEvent returns the WaitForEvent command from the CLI driver, or
// nil when no driver is wired.  It is called after every handled CLI event to
// maintain the re-subscription chain.
func (m AppModel) waitForCLIEvent() tea.Cmd {
	if m.shared == nil || m.shared.cliDriver == nil {
		return nil
	}
	return m.shared.cliDriver.WaitForEvent()
}
