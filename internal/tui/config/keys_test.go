package config

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// keysOf returns the slice of key strings registered on a binding.
// This mirrors the internal representation so tests can assert exact values
// without relying on private fields.
func keysOf(b key.Binding) []string {
	// key.Binding exposes Keys() via the public method added in bubbles v0.14+.
	return b.Keys()
}

// helpKey returns the short key string from the binding's Help() text.
func helpKey(b key.Binding) string {
	return b.Help().Key
}

// helpDesc returns the description string from the binding's Help() text.
func helpDesc(b key.Binding) string {
	return b.Help().Desc
}

// ---------------------------------------------------------------------------
// DefaultKeyMap — structural smoke test
// ---------------------------------------------------------------------------

func TestDefaultKeyMap_ReturnsNonZeroMap(t *testing.T) {
	km := DefaultKeyMap()

	// Verify every top-level group is populated by spot-checking one field.
	assert.NotEmpty(t, keysOf(km.Global.ForceQuit), "Global.ForceQuit must have keys")
	assert.NotEmpty(t, keysOf(km.Tab.TabChat), "Tab.TabChat must have keys")
	assert.NotEmpty(t, keysOf(km.Claude.Submit), "Claude.Submit must have keys")
	assert.NotEmpty(t, keysOf(km.Agent.AgentExpand), "Agent.AgentExpand must have keys")
	assert.NotEmpty(t, keysOf(km.Modal.ModalCancel), "Modal.ModalCancel must have keys")
}

// ---------------------------------------------------------------------------
// Global keybindings
// ---------------------------------------------------------------------------

func TestGlobalKeys_KeyStrings(t *testing.T) {
	km := DefaultKeyMap()

	tests := []struct {
		name     string
		binding  key.Binding
		wantKeys []string
	}{
		{"ToggleFocus", km.Global.ToggleFocus, []string{"tab"}},
		{"CycleProvider", km.Global.CycleProvider, []string{"shift+tab"}},
		{"CycleRightPanel", km.Global.CycleRightPanel, []string{"alt+r"}},
		{"CyclePermMode", km.Global.CyclePermMode, []string{"alt+p"}},
		{"Interrupt", km.Global.Interrupt, []string{"esc"}},
		{"ForceQuit", km.Global.ForceQuit, []string{"ctrl+c"}},
		{"ClearScreen", km.Global.ClearScreen, []string{"ctrl+l"}},
		{"ToggleTaskBoard", km.Global.ToggleTaskBoard, []string{"alt+b"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantKeys, keysOf(tc.binding),
				"%s key strings must match spec", tc.name)
		})
	}
}

func TestGlobalKeys_HelpTextNonEmpty(t *testing.T) {
	km := DefaultKeyMap()

	bindings := []struct {
		name    string
		binding key.Binding
	}{
		{"ToggleFocus", km.Global.ToggleFocus},
		{"CycleProvider", km.Global.CycleProvider},
		{"CycleRightPanel", km.Global.CycleRightPanel},
		{"CyclePermMode", km.Global.CyclePermMode},
		{"Interrupt", km.Global.Interrupt},
		{"ForceQuit", km.Global.ForceQuit},
		{"ClearScreen", km.Global.ClearScreen},
		{"ToggleTaskBoard", km.Global.ToggleTaskBoard},
	}

	for _, tc := range bindings {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotEmpty(t, helpKey(tc.binding),
				"%s Help().Key must not be empty", tc.name)
			assert.NotEmpty(t, helpDesc(tc.binding),
				"%s Help().Desc must not be empty", tc.name)
		})
	}
}

// ---------------------------------------------------------------------------
// Tab keybindings
// ---------------------------------------------------------------------------

func TestTabKeys_KeyStrings(t *testing.T) {
	km := DefaultKeyMap()

	tests := []struct {
		name     string
		binding  key.Binding
		wantKeys []string
	}{
		{"TabChat", km.Tab.TabChat, []string{"alt+c"}},
		{"TabAgentConfig", km.Tab.TabAgentConfig, []string{"alt+a"}},
		{"TabTeamConfig", km.Tab.TabTeamConfig, []string{"alt+t"}},
		{"TabTelemetry", km.Tab.TabTelemetry, []string{"alt+y"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantKeys, keysOf(tc.binding),
				"%s key strings must match spec", tc.name)
		})
	}
}

func TestTabKeys_HelpTextNonEmpty(t *testing.T) {
	km := DefaultKeyMap()

	bindings := []struct {
		name    string
		binding key.Binding
	}{
		{"TabChat", km.Tab.TabChat},
		{"TabAgentConfig", km.Tab.TabAgentConfig},
		{"TabTeamConfig", km.Tab.TabTeamConfig},
		{"TabTelemetry", km.Tab.TabTelemetry},
	}

	for _, tc := range bindings {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotEmpty(t, helpKey(tc.binding),
				"%s Help().Key must not be empty", tc.name)
			assert.NotEmpty(t, helpDesc(tc.binding),
				"%s Help().Desc must not be empty", tc.name)
		})
	}
}

// ---------------------------------------------------------------------------
// Claude panel keybindings
// ---------------------------------------------------------------------------

func TestClaudeKeys_KeyStrings(t *testing.T) {
	km := DefaultKeyMap()

	tests := []struct {
		name     string
		binding  key.Binding
		wantKeys []string
	}{
		{"Submit", km.Claude.Submit, []string{"enter"}},
		{"HistoryPrev", km.Claude.HistoryPrev, []string{"up"}},
		{"HistoryNext", km.Claude.HistoryNext, []string{"down"}},
		{"ToggleToolExpansion", km.Claude.ToggleToolExpansion, []string{"alt+e"}},
		{"CycleExpansion", km.Claude.CycleExpansion, []string{"alt+E"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantKeys, keysOf(tc.binding),
				"%s key strings must match spec", tc.name)
		})
	}
}

func TestClaudeKeys_HelpTextNonEmpty(t *testing.T) {
	km := DefaultKeyMap()

	bindings := []struct {
		name    string
		binding key.Binding
	}{
		{"Submit", km.Claude.Submit},
		{"HistoryPrev", km.Claude.HistoryPrev},
		{"HistoryNext", km.Claude.HistoryNext},
		{"ToggleToolExpansion", km.Claude.ToggleToolExpansion},
		{"CycleExpansion", km.Claude.CycleExpansion},
	}

	for _, tc := range bindings {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotEmpty(t, helpKey(tc.binding),
				"%s Help().Key must not be empty", tc.name)
			assert.NotEmpty(t, helpDesc(tc.binding),
				"%s Help().Desc must not be empty", tc.name)
		})
	}
}

// ---------------------------------------------------------------------------
// Agent panel keybindings
// ---------------------------------------------------------------------------

func TestAgentKeys_KeyStrings(t *testing.T) {
	km := DefaultKeyMap()

	tests := []struct {
		name     string
		binding  key.Binding
		wantKeys []string
	}{
		{"AgentUp", km.Agent.AgentUp, []string{"up"}},
		{"AgentDown", km.Agent.AgentDown, []string{"down"}},
		{"AgentExpand", km.Agent.AgentExpand, []string{"enter"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantKeys, keysOf(tc.binding),
				"%s key strings must match spec", tc.name)
		})
	}
}

func TestAgentKeys_HelpTextNonEmpty(t *testing.T) {
	km := DefaultKeyMap()

	bindings := []struct {
		name    string
		binding key.Binding
	}{
		{"AgentUp", km.Agent.AgentUp},
		{"AgentDown", km.Agent.AgentDown},
		{"AgentExpand", km.Agent.AgentExpand},
	}

	for _, tc := range bindings {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotEmpty(t, helpKey(tc.binding),
				"%s Help().Key must not be empty", tc.name)
			assert.NotEmpty(t, helpDesc(tc.binding),
				"%s Help().Desc must not be empty", tc.name)
		})
	}
}

// ---------------------------------------------------------------------------
// Modal keybindings
// ---------------------------------------------------------------------------

func TestModalKeys_KeyStrings(t *testing.T) {
	km := DefaultKeyMap()

	tests := []struct {
		name     string
		binding  key.Binding
		wantKeys []string
	}{
		{"ModalUp", km.Modal.ModalUp, []string{"up"}},
		{"ModalDown", km.Modal.ModalDown, []string{"down"}},
		{"ModalSelect", km.Modal.ModalSelect, []string{"enter"}},
		{"ModalCancel", km.Modal.ModalCancel, []string{"esc"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantKeys, keysOf(tc.binding),
				"%s key strings must match spec", tc.name)
		})
	}
}

func TestModalKeys_HelpTextNonEmpty(t *testing.T) {
	km := DefaultKeyMap()

	bindings := []struct {
		name    string
		binding key.Binding
	}{
		{"ModalUp", km.Modal.ModalUp},
		{"ModalDown", km.Modal.ModalDown},
		{"ModalSelect", km.Modal.ModalSelect},
		{"ModalCancel", km.Modal.ModalCancel},
	}

	for _, tc := range bindings {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotEmpty(t, helpKey(tc.binding),
				"%s Help().Key must not be empty", tc.name)
			assert.NotEmpty(t, helpDesc(tc.binding),
				"%s Help().Desc must not be empty", tc.name)
		})
	}
}

// ---------------------------------------------------------------------------
// DefaultKeyMap is idempotent — two calls return equivalent bindings
// ---------------------------------------------------------------------------

func TestDefaultKeyMap_IsIdempotent(t *testing.T) {
	km1 := DefaultKeyMap()
	km2 := DefaultKeyMap()

	// Spot-check a selection of bindings across all groups.
	pairs := []struct {
		name string
		a, b key.Binding
	}{
		{"Global.ForceQuit", km1.Global.ForceQuit, km2.Global.ForceQuit},
		{"Global.ToggleFocus", km1.Global.ToggleFocus, km2.Global.ToggleFocus},
		{"Tab.TabChat", km1.Tab.TabChat, km2.Tab.TabChat},
		{"Claude.Submit", km1.Claude.Submit, km2.Claude.Submit},
		{"Agent.AgentUp", km1.Agent.AgentUp, km2.Agent.AgentUp},
		{"Modal.ModalCancel", km1.Modal.ModalCancel, km2.Modal.ModalCancel},
	}

	for _, tc := range pairs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, keysOf(tc.a), keysOf(tc.b),
				"%s key strings must be identical across calls", tc.name)
			assert.Equal(t, helpKey(tc.a), helpKey(tc.b),
				"%s Help().Key must be identical across calls", tc.name)
			assert.Equal(t, helpDesc(tc.a), helpDesc(tc.b),
				"%s Help().Desc must be identical across calls", tc.name)
		})
	}
}

// ---------------------------------------------------------------------------
// Binding count — regression guard
// ---------------------------------------------------------------------------

func TestKeyMap_TotalBindingCount(t *testing.T) {
	// Counts are derived from the spec:
	//   Global: 8, Tab: 4, Claude: 5, Agent: 3, Modal: 4 → total 24
	km := DefaultKeyMap()

	globalCount := 8
	tabCount := 4
	claudeCount := 5
	agentCount := 3
	modalCount := 4

	globalBindings := []key.Binding{
		km.Global.ToggleFocus,
		km.Global.CycleProvider,
		km.Global.CycleRightPanel,
		km.Global.CyclePermMode,
		km.Global.Interrupt,
		km.Global.ForceQuit,
		km.Global.ClearScreen,
		km.Global.ToggleTaskBoard,
	}
	tabBindings := []key.Binding{
		km.Tab.TabChat,
		km.Tab.TabAgentConfig,
		km.Tab.TabTeamConfig,
		km.Tab.TabTelemetry,
	}
	claudeBindings := []key.Binding{
		km.Claude.Submit,
		km.Claude.HistoryPrev,
		km.Claude.HistoryNext,
		km.Claude.ToggleToolExpansion,
		km.Claude.CycleExpansion,
	}
	agentBindings := []key.Binding{
		km.Agent.AgentUp,
		km.Agent.AgentDown,
		km.Agent.AgentExpand,
	}
	modalBindings := []key.Binding{
		km.Modal.ModalUp,
		km.Modal.ModalDown,
		km.Modal.ModalSelect,
		km.Modal.ModalCancel,
	}

	assert.Len(t, globalBindings, globalCount, "global binding count")
	assert.Len(t, tabBindings, tabCount, "tab binding count")
	assert.Len(t, claudeBindings, claudeCount, "claude binding count")
	assert.Len(t, agentBindings, agentCount, "agent binding count")
	assert.Len(t, modalBindings, modalCount, "modal binding count")
}
