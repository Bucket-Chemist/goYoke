// Package model defines shared state types for the GOgent-Fortress TUI.
// This file contains the provider-switching flow (TUI-029).
package model

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/cli"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
)

// handleProviderSwitch implements the provider-cycling flow (TUI-029):
//
//  1. Save the current conversation history to the active provider slot.
//  2. Save the current session ID to the active provider slot.
//  3. Cycle to the next provider in the canonical order.
//  4. Restore the new provider's conversation history into the panel.
//  5. Restart the CLI driver for the new provider via restartCLIDriver.
//
// The method is a no-op when no ProviderState is wired (e.g. in tests that
// do not inject one).
func (m AppModel) handleProviderSwitch() (tea.Model, tea.Cmd) {
	if m.shared == nil || m.shared.providerState == nil {
		return m, nil
	}

	ps := m.shared.providerState

	// 1. Persist current conversation to the active provider slot.
	// Capture the old provider and its messages BEFORE cycling for handoff generation.
	oldProvider := ps.GetActiveProvider()
	var oldMsgs []state.DisplayMessage
	if m.shared.claudePanel != nil {
		oldMsgs = m.shared.claudePanel.SaveMessages()
		ps.SetActiveMessages(oldMsgs)
	}

	// 2. Persist current session ID to the active provider slot.
	if m.sessionID != "" {
		ps.SetSessionID(m.sessionID)
	}

	// 3. Cycle to the next provider in the canonical ordered list.
	providers := ps.AllProviders()
	nextIdx := 0
	for i, p := range providers {
		if p == oldProvider {
			nextIdx = (i + 1) % len(providers)
			break
		}
	}
	if err := ps.SwitchProvider(providers[nextIdx]); err != nil {
		// Unknown provider — this should never happen with the hardcoded list.
		return m, nil
	}

	// Update provider tab bar highlight to reflect the new active provider.
	if m.shared.providerTabBar != nil {
		m.shared.providerTabBar.SetActive(ps.GetActiveProvider())
	}

	// Sync status line provider name.
	m.statusLine.Provider = string(ps.GetActiveProvider())

	// 4. Restore the new provider's conversation history.
	if m.shared.claudePanel != nil {
		newMsgs := ps.GetActiveMessages()
		m.shared.claudePanel.RestoreMessages(newMsgs)
	}

	// 4.5. Inject handoff context so the new provider knows what was being discussed.
	handoff := buildHandoffSummary(oldMsgs, oldProvider, ps.GetActiveProvider())
	if handoff != "" {
		ps.AppendMessage(state.DisplayMessage{
			Role:      "system",
			Content:   handoff,
			Timestamp: time.Now(),
		})
		// Re-restore so the injected handoff message is visible in the panel.
		if m.shared.claudePanel != nil {
			m.shared.claudePanel.RestoreMessages(ps.GetActiveMessages())
		}
	}

	// 5. Restart the CLI driver for the new provider.
	return m.restartCLIDriver()
}

// restartCLIDriver shuts down the current CLI driver and starts a new one
// configured for the active provider and model. It reads the active model,
// session ID, adapter path, project dir, and env vars from ProviderState.
//
// This helper is shared by provider switching (TUI-029) and model switching.
//
// Callers must ensure ProviderState is non-nil and the desired model/provider
// is already set before calling this method.
func (m AppModel) restartCLIDriver() (tea.Model, tea.Cmd) {
	ps := m.shared.providerState

	// Shutdown the old CLI driver.
	if m.shared.cliDriver != nil {
		_ = m.shared.cliDriver.Shutdown()
	}

	// Build CLI driver options for the active provider/model.
	// Start from the baseline options captured at startup so that flags
	// such as --verbose, --debug, and --permission-mode are not silently lost.
	cfg := ps.GetActiveConfig()
	activeModel := ps.GetActiveModel()

	// Preserve the [1m] context window suffix when the session has 1M access.
	// The context1M flag is latched on the first SystemInitEvent and survives
	// across model switches.  Only opus supports [1m] on Max subscriptions;
	// sonnet[1m] requires extra usage that is not available to all plans.
	if m.context1M &&
		!strings.Contains(activeModel, "[1m]") &&
		strings.Contains(strings.ToLower(activeModel), "opus") {
		activeModel = activeModel + "[1m]"
	}

	opts := m.shared.baseCLIOpts // value copy preserves Verbose, Debug, PermissionMode, MCPConfigPath, etc.
	opts.Model = activeModel
	opts.SessionID = ps.GetActiveSessionID() // Resume if provider was used before (TUI-031)
	opts.AdapterPath = cfg.AdapterPath
	opts.ProjectDir = ps.GetActiveProjectDir()
	// Materialise env-var keys for the new provider.  The values are
	// intentionally left empty here: the actual credentials must be present
	// in the process environment already (set by the user before launch).
	// We only pass the map so the driver knows which vars are relevant.
	if len(cfg.EnvVars) > 0 {
		envCopy := make(map[string]string, len(cfg.EnvVars))
		for k := range cfg.EnvVars {
			envCopy[k] = "" // empty — real value comes from os.Environ()
		}
		opts.EnvVars = envCopy
	} else {
		opts.EnvVars = nil
	}

	// Reset per-session state so the new driver starts fresh.
	m.cliReady = false
	m.sessionID = ""
	m.activeModel = activeModel
	m.reconnectCount = 0
	m.reconnectSeq++ // invalidate any pending reconnect timers

	// Create, wire, and start the new CLI driver.
	newDriver := cli.NewCLIDriver(opts)
	m.shared.cliDriver = newDriver

	// Update the Claude panel's sender so user messages go to the new
	// driver, not the now-shutdown old one.
	if m.shared.claudePanel != nil {
		m.shared.claudePanel.SetSender(newDriver)
	}

	return m, newDriver.Start()
}
