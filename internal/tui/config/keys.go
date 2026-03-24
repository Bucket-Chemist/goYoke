// Package config provides the foundational theme system for the GOgent-Fortress TUI.
// This file defines the keybinding registry using charmbracelet/bubbles/key.
package config

import "github.com/charmbracelet/bubbles/key"

// ---------------------------------------------------------------------------
// KeyMap
//
// KeyMap holds every named keybinding for the TUI.  Bindings are grouped into
// four logical sections so it is easy to reason about which bindings are active
// in each context:
//
//   - GlobalKeys   — always active when no modal is open
//   - TabKeys      — quick-jump shortcuts to named tabs
//   - ClaudeKeys   — bindings for the Claude input/output panel
//   - AgentKeys    — bindings for the agent-list panel
//   - ModalKeys    — bindings active only while a modal overlay is shown
//
// The zero value is not usable; use DefaultKeyMap instead.
// ---------------------------------------------------------------------------

// GlobalKeys groups the keybindings that are always active when no modal is
// open, regardless of which panel or tab has focus.
type GlobalKeys struct {
	// ToggleFocus moves focus between the major UI panes (forward direction).
	ToggleFocus key.Binding

	// ReverseToggleFocus moves focus between the major UI panes in reverse.
	// TUI-052: Shift+Tab rebind from CycleProvider to ReverseToggleFocus (standard keyboard convention)
	ReverseToggleFocus key.Binding

	// CycleProvider rotates through the available LLM providers.
	CycleProvider key.Binding

	// CycleRightPanel rotates the content shown in the right-hand panel.
	CycleRightPanel key.Binding

	// CyclePermMode rotates through permission-escalation modes.
	CyclePermMode key.Binding

	// Interrupt sends an interrupt signal to the active agent.
	Interrupt key.Binding

	// ForceQuit exits the application immediately.
	ForceQuit key.Binding

	// ClearScreen redraws the terminal from scratch.
	ClearScreen key.Binding

	// ToggleTaskBoard shows or hides the task-board overlay.
	ToggleTaskBoard key.Binding

	// ViewPlan opens the full-screen Glamour plan viewer when the right
	// panel is in RPMPlanPreview mode.
	ViewPlan key.Binding

	// Search opens the unified cross-panel fuzzy search overlay (TUI-059).
	Search key.Binding
}

// TabKeys groups the alt-key shortcuts that jump directly to a named tab.
type TabKeys struct {
	// TabChat jumps to the Chat tab.
	TabChat key.Binding

	// TabAgentConfig jumps to the Agent Config tab.
	TabAgentConfig key.Binding

	// TabTeamConfig jumps to the Team Config tab.
	TabTeamConfig key.Binding

	// TabTelemetry jumps to the Telemetry tab.
	TabTelemetry key.Binding
}

// ClaudeKeys groups the keybindings that are active when the Claude
// input/output panel has focus.
type ClaudeKeys struct {
	// Submit sends the current input to the active agent.
	Submit key.Binding

	// HistoryPrev recalls the previous item from the input history.
	HistoryPrev key.Binding

	// HistoryNext recalls the next item from the input history.
	HistoryNext key.Binding

	// ToggleToolExpansion collapses or expands the most-recent tool call.
	ToggleToolExpansion key.Binding

	// CycleExpansion cycles through the expansion levels of tool calls.
	CycleExpansion key.Binding

	// Search activates the in-panel search overlay (TUI-035).
	Search key.Binding

	// SearchNext moves to the next search result (TUI-035).
	SearchNext key.Binding

	// SearchPrev moves to the previous search result (TUI-035).
	SearchPrev key.Binding

	// CopyLastResponse copies the last assistant message to the clipboard (TUI-035).
	CopyLastResponse key.Binding
}

// AgentKeys groups the keybindings that are active when the agent-list panel
// has focus.
type AgentKeys struct {
	// AgentUp moves the selection cursor up one row.
	AgentUp key.Binding

	// AgentDown moves the selection cursor down one row.
	AgentDown key.Binding

	// AgentExpand expands the selected agent entry to show detail.
	AgentExpand key.Binding
}

// ModalKeys groups the keybindings that are active while a modal overlay is
// displayed.  These shadow the global bindings for the duration of the modal.
type ModalKeys struct {
	// ModalUp moves the modal selection cursor up one row.
	ModalUp key.Binding

	// ModalDown moves the modal selection cursor down one row.
	ModalDown key.Binding

	// ModalSelect confirms the current modal selection.
	ModalSelect key.Binding

	// ModalCancel dismisses the modal without making a selection.
	ModalCancel key.Binding
}

// KeyMap is the top-level registry of all keybindings for the TUI.
// Embed or pass by value to components that need keybinding access.
type KeyMap struct {
	Global GlobalKeys
	Tab    TabKeys
	Claude ClaudeKeys
	Agent  AgentKeys
	Modal  ModalKeys

	// VimEnabled controls whether the vim keybinding overlay is active.
	// Off by default; toggled via the settings panel (TUI-062).
	VimEnabled bool

	// VimMode is the current mode when VimEnabled is true.
	// The zero value VimNormal is the initial resting state.
	VimMode VimMode

	// Vim holds the vim key bindings used when VimEnabled is true.
	Vim VimKeys
}

// ---------------------------------------------------------------------------
// DefaultKeyMap
// ---------------------------------------------------------------------------

// DefaultKeyMap returns a KeyMap populated with the standard GOgent-Fortress
// keybindings.  Each binding includes Help text so the bubbles/help component
// can render a contextual cheat-sheet.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Global: GlobalKeys{
			ToggleFocus: key.NewBinding(
				key.WithKeys("tab"),
				key.WithHelp("tab", "toggle focus"),
			),
			// TUI-052: Shift+Tab rebind from CycleProvider to ReverseToggleFocus (standard keyboard convention)
			ReverseToggleFocus: key.NewBinding(
				key.WithKeys("shift+tab"),
				key.WithHelp("shift+tab", "reverse focus"),
			),
			CycleProvider: key.NewBinding(
				key.WithKeys("alt+P"),
				key.WithHelp("alt+shift+p", "cycle provider"),
			),
			CycleRightPanel: key.NewBinding(
				key.WithKeys("alt+r"),
				key.WithHelp("alt+r", "cycle right panel"),
			),
			CyclePermMode: key.NewBinding(
				key.WithKeys("alt+p"),
				key.WithHelp("alt+p", "cycle perm mode"),
			),
			Interrupt: key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "interrupt"),
			),
			ForceQuit: key.NewBinding(
				key.WithKeys("ctrl+c"),
				key.WithHelp("ctrl+c", "quit"),
			),
			ClearScreen: key.NewBinding(
				key.WithKeys("ctrl+l"),
				key.WithHelp("ctrl+l", "clear screen"),
			),
			ToggleTaskBoard: key.NewBinding(
				key.WithKeys("alt+b"),
				key.WithHelp("alt+b", "toggle task board"),
			),
			ViewPlan: key.NewBinding(
				key.WithKeys("alt+v"),
				key.WithHelp("alt+v", "view plan"),
			),
			Search: key.NewBinding(
				key.WithKeys("ctrl+f"),
				key.WithHelp("ctrl+f", "search"),
			),
		},

		Tab: TabKeys{
			TabChat: key.NewBinding(
				key.WithKeys("alt+c"),
				key.WithHelp("alt+c", "chat tab"),
			),
			TabAgentConfig: key.NewBinding(
				key.WithKeys("alt+a"),
				key.WithHelp("alt+a", "agent config tab"),
			),
			TabTeamConfig: key.NewBinding(
				key.WithKeys("alt+t"),
				key.WithHelp("alt+t", "team config tab"),
			),
			TabTelemetry: key.NewBinding(
				key.WithKeys("alt+y"),
				key.WithHelp("alt+y", "telemetry tab"),
			),
		},

		Claude: ClaudeKeys{
			Submit: key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "submit"),
			),
			HistoryPrev: key.NewBinding(
				key.WithKeys("up"),
				key.WithHelp("↑", "history prev"),
			),
			HistoryNext: key.NewBinding(
				key.WithKeys("down"),
				key.WithHelp("↓", "history next"),
			),
			ToggleToolExpansion: key.NewBinding(
				key.WithKeys("alt+e"),
				key.WithHelp("alt+e", "toggle tool expansion"),
			),
			CycleExpansion: key.NewBinding(
				key.WithKeys("alt+E"),
				key.WithHelp("alt+shift+e", "cycle expansion"),
			),
			Search: key.NewBinding(
				key.WithKeys("/"),
				key.WithHelp("/", "search"),
			),
			SearchNext: key.NewBinding(
				key.WithKeys("ctrl+n"),
				key.WithHelp("ctrl+n", "next result"),
			),
			SearchPrev: key.NewBinding(
				key.WithKeys("ctrl+p"),
				key.WithHelp("ctrl+p", "prev result"),
			),
			CopyLastResponse: key.NewBinding(
				key.WithKeys("ctrl+y"),
				key.WithHelp("ctrl+y", "copy last response"),
			),
		},

		Agent: AgentKeys{
			AgentUp: key.NewBinding(
				key.WithKeys("up"),
				key.WithHelp("↑", "agent up"),
			),
			AgentDown: key.NewBinding(
				key.WithKeys("down"),
				key.WithHelp("↓", "agent down"),
			),
			AgentExpand: key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "expand agent"),
			),
		},

		Modal: ModalKeys{
			ModalUp: key.NewBinding(
				key.WithKeys("up"),
				key.WithHelp("↑", "up"),
			),
			ModalDown: key.NewBinding(
				key.WithKeys("down"),
				key.WithHelp("↓", "down"),
			),
			ModalSelect: key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "select"),
			),
			ModalCancel: key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "cancel"),
			),
		},

		// VimEnabled is false by default; the Vim bindings are pre-populated so
		// they are ready to use the moment the user enables vim mode.
		VimEnabled: false,
		VimMode:    VimNormal,
		Vim:        DefaultVimKeys(),
	}
}
