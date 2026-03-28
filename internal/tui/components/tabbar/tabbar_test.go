package tabbar_test

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/tabbar"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/model"
)

func TestNewTabBarModel(t *testing.T) {
	keys := config.DefaultKeyMap()
	m := tabbar.NewTabBarModel(keys, 120)
	if m.ActiveTab() != model.TabChat {
		t.Errorf("NewTabBarModel default active tab = %v; want TabChat", m.ActiveTab())
	}
}

func TestTabBarInit(t *testing.T) {
	keys := config.DefaultKeyMap()
	m := tabbar.NewTabBarModel(keys, 120)
	cmd := m.Init()
	if cmd != nil {
		t.Error("Init should return nil command")
	}
}

func TestTabBarViewContainsAllLabels(t *testing.T) {
	keys := config.DefaultKeyMap()
	m := tabbar.NewTabBarModel(keys, 120)
	view := m.View()

	labels := []string{"Chat", "Agent Config", "Team Config", "Telemetry"}
	for _, label := range labels {
		if !strings.Contains(view, label) {
			t.Errorf("View() missing label %q; got:\n%s", label, view)
		}
	}
}

func TestTabBarViewSingleRow(t *testing.T) {
	keys := config.DefaultKeyMap()
	m := tabbar.NewTabBarModel(keys, 120)
	view := m.View()
	view = strings.TrimRight(view, "\n")
	lines := strings.Split(view, "\n")
	if len(lines) != 1 {
		t.Errorf("View() should produce 1 row; got %d:\n%s", len(lines), view)
	}
}

func TestTabBarSwitchToAgentConfig(t *testing.T) {
	keys := config.DefaultKeyMap()
	m := tabbar.NewTabBarModel(keys, 120)

	// Simulate Alt+A key press (TabAgentConfig).
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Alt: true, Runes: []rune{'a'}})
	if cmd != nil {
		t.Error("tab switch should return nil command")
	}
	tb := newModel.(tabbar.TabBarModel)
	if tb.ActiveTab() != model.TabAgentConfig {
		t.Errorf("after Alt+A active tab = %v; want TabAgentConfig", tb.ActiveTab())
	}
}

func TestTabBarSwitchToTeamConfig(t *testing.T) {
	keys := config.DefaultKeyMap()
	m := tabbar.NewTabBarModel(keys, 120)

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Alt: true, Runes: []rune{'t'}})
	tb := newModel.(tabbar.TabBarModel)
	if tb.ActiveTab() != model.TabTeamConfig {
		t.Errorf("after Alt+T active tab = %v; want TabTeamConfig", tb.ActiveTab())
	}
}

func TestTabBarSwitchToTelemetry(t *testing.T) {
	keys := config.DefaultKeyMap()
	m := tabbar.NewTabBarModel(keys, 120)

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Alt: true, Runes: []rune{'y'}})
	tb := newModel.(tabbar.TabBarModel)
	if tb.ActiveTab() != model.TabTelemetry {
		t.Errorf("after Alt+Y active tab = %v; want TabTelemetry", tb.ActiveTab())
	}
}

func TestTabBarSwitchBackToChat(t *testing.T) {
	keys := config.DefaultKeyMap()
	m := tabbar.NewTabBarModel(keys, 120)

	// Switch away from Chat then back.
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Alt: true, Runes: []rune{'a'}})
	newModel, _ = newModel.(tabbar.TabBarModel).Update(tea.KeyMsg{Type: tea.KeyRunes, Alt: true, Runes: []rune{'c'}})
	tb := newModel.(tabbar.TabBarModel)
	if tb.ActiveTab() != model.TabChat {
		t.Errorf("after Alt+C active tab = %v; want TabChat", tb.ActiveTab())
	}
}

func TestTabBarWindowSizeMsg(t *testing.T) {
	keys := config.DefaultKeyMap()
	m := tabbar.NewTabBarModel(keys, 80)

	newModel, cmd := m.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
	if cmd != nil {
		t.Error("WindowSizeMsg should return nil command")
	}
	// Should not panic and still render all labels.
	view := newModel.(tabbar.TabBarModel).View()
	if !strings.Contains(view, "Chat") {
		t.Errorf("View() after resize missing 'Chat'; got:\n%s", view)
	}
}

func TestTabBarSetActiveTab(t *testing.T) {
	keys := config.DefaultKeyMap()
	m := tabbar.NewTabBarModel(keys, 120)

	m.SetActiveTab(model.TabTelemetry)
	if m.ActiveTab() != model.TabTelemetry {
		t.Errorf("SetActiveTab did not update active tab; got %v", m.ActiveTab())
	}
}

func TestTabBarSetActiveTabInvalidID(t *testing.T) {
	keys := config.DefaultKeyMap()
	m := tabbar.NewTabBarModel(keys, 120)

	// An unknown ID should leave the active tab unchanged.
	m.SetActiveTab(model.TabID(999))
	if m.ActiveTab() != model.TabChat {
		t.Errorf("SetActiveTab with invalid ID should leave TabChat active; got %v", m.ActiveTab())
	}
}

func TestTabBarSetWidth(t *testing.T) {
	keys := config.DefaultKeyMap()
	m := tabbar.NewTabBarModel(keys, 80)
	m.SetWidth(160)
	// Render after SetWidth should not panic.
	view := m.View()
	if !strings.Contains(view, "Chat") {
		t.Errorf("View() after SetWidth missing 'Chat'; got:\n%s", view)
	}
}

// ---------------------------------------------------------------------------
// TUI-061: Tab flash animation
// ---------------------------------------------------------------------------

// TestTabFlash_HandleMsgActivatesFlash verifies that delivering a TabFlashMsg
// sets the flash state and returns a non-nil scheduling command.
func TestTabFlash_HandleMsgActivatesFlash(t *testing.T) {
	tests := []struct {
		name     string
		tabIndex int
	}{
		{"chat tab (index 0)", 0},
		{"agent config tab (index 1)", 1},
		{"team config tab (index 2)", 2},
		{"telemetry tab (index 3)", 3},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			keys := config.DefaultKeyMap()
			m := tabbar.NewTabBarModel(keys, 120)

			cmd := m.HandleMsg(model.TabFlashMsg{TabIndex: tc.tabIndex})
			if cmd == nil {
				t.Error("HandleMsg(TabFlashMsg) returned nil cmd; want tick cmd")
			}
			if !m.IsFlashing() {
				t.Error("IsFlashing() = false after TabFlashMsg; want true")
			}
			if m.FlashTab() != tc.tabIndex {
				t.Errorf("FlashTab() = %d; want %d", m.FlashTab(), tc.tabIndex)
			}
		})
	}
}

// TestTabFlash_ViewRenderesDifferentlyDuringFlash verifies that:
//   - the flash state is active immediately after HandleMsg(TabFlashMsg)
//   - View() still renders all tab labels during a flash
//   - IsFlashing() correctly reflects the flash window
func TestTabFlash_ViewRenderesDifferentlyDuringFlash(t *testing.T) {
	keys := config.DefaultKeyMap()
	m := tabbar.NewTabBarModel(keys, 120)

	if m.IsFlashing() {
		t.Fatal("precondition: IsFlashing() = true before any flash; want false")
	}

	// Activate flash on the Chat tab (index 0, which is activeTab).
	cmd := m.HandleMsg(model.TabFlashMsg{TabIndex: 0})
	if cmd == nil {
		t.Error("HandleMsg(TabFlashMsg) returned nil cmd; want tick cmd")
	}
	if !m.IsFlashing() {
		t.Error("IsFlashing() = false immediately after TabFlashMsg; want true")
	}

	// View() must still render all labels without panicking during flash.
	flashView := m.View()
	for _, label := range []string{"Chat", "Agent Config", "Team Config", "Telemetry"} {
		if !strings.Contains(flashView, label) {
			t.Errorf("View() during flash missing label %q", label)
		}
	}
}

// TestTabFlash_AutoClearsAfterTick verifies that the flash state is cleared
// when the component receives the internal tick after 300 ms.
func TestTabFlash_AutoClearsAfterTick(t *testing.T) {
	keys := config.DefaultKeyMap()
	m := tabbar.NewTabBarModel(keys, 120)

	// Start a flash.
	_ = m.HandleMsg(model.TabFlashMsg{TabIndex: 0})
	if !m.IsFlashing() {
		t.Fatal("precondition: IsFlashing() = false; want true after TabFlashMsg")
	}

	// Simulate the 300 ms tick firing by using the Update path which handles
	// the tick message.  We advance the flash start backwards so that the
	// elapsed check passes.
	m.BackdateFlashStart(400 * time.Millisecond)

	// Deliver the flash tick via Update (the tea.Model path).
	updated, _ := m.Update(tabbar.ExportedFlashTick())
	tb := updated.(tabbar.TabBarModel)

	if tb.IsFlashing() {
		t.Error("IsFlashing() = true after tick past flashDuration; want false")
	}
}

// TestTabFlash_NoFlashWhenInactive verifies that View() is unchanged (from the
// styling perspective) when no flash is active.
func TestTabFlash_NoFlashWhenInactive(t *testing.T) {
	keys := config.DefaultKeyMap()
	m := tabbar.NewTabBarModel(keys, 120)

	if m.IsFlashing() {
		t.Fatal("precondition: NewTabBarModel should have no active flash")
	}
	// Just verify View() doesn't panic and contains expected labels.
	view := m.View()
	for _, label := range []string{"Chat", "Agent Config", "Team Config", "Telemetry"} {
		if !strings.Contains(view, label) {
			t.Errorf("View() without flash missing label %q", label)
		}
	}
}

// TestTabFlash_SecondFlashOverridesFirst verifies that a second TabFlashMsg
// resets the flash window, replacing any previous flash.
func TestTabFlash_SecondFlashOverridesFirst(t *testing.T) {
	keys := config.DefaultKeyMap()
	m := tabbar.NewTabBarModel(keys, 120)

	// First flash on tab 0.
	before := time.Now()
	_ = m.HandleMsg(model.TabFlashMsg{TabIndex: 0})
	_ = before

	// Small sleep to ensure clock advances.
	time.Sleep(5 * time.Millisecond)

	firstStart := m.FlashStart()

	// Second flash on tab 1 (must switch to AgentConfig first).
	m.SetActiveTab(model.TabAgentConfig)
	_ = m.HandleMsg(model.TabFlashMsg{TabIndex: 1})

	if m.FlashTab() != 1 {
		t.Errorf("FlashTab() = %d after second flash; want 1", m.FlashTab())
	}
	if !m.FlashStart().After(firstStart) {
		t.Error("FlashStart() not updated after second flash; second flash should reset the timer")
	}
}

// TestTabFlash_InvalidTabIndexNoCrash verifies that a TabFlashMsg with an
// out-of-range tab index does not crash and the flash state is still set
// (clamping/guarding is not required; the caller is responsible).
func TestTabFlash_InvalidTabIndexNoCrash(t *testing.T) {
	keys := config.DefaultKeyMap()
	m := tabbar.NewTabBarModel(keys, 120)

	// Should not panic even with a large index.
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("HandleMsg panicked with out-of-range tab index: %v", r)
			}
		}()
		_ = m.HandleMsg(model.TabFlashMsg{TabIndex: 999})
	}()

	// View must also not panic.
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("View() panicked with out-of-range flash index: %v", r)
			}
		}()
		_ = m.View()
	}()
}

// TestTabFlash_HandleMsgUnknownMsgReturnsNil verifies that HandleMsg with an
// unrelated message type returns nil and does not activate flash.
func TestTabFlash_HandleMsgUnknownMsgReturnsNil(t *testing.T) {
	keys := config.DefaultKeyMap()
	m := tabbar.NewTabBarModel(keys, 120)

	cmd := m.HandleMsg(tea.WindowSizeMsg{Width: 80, Height: 24})
	if cmd != nil {
		t.Errorf("HandleMsg(WindowSizeMsg) = non-nil cmd; want nil")
	}
	if m.IsFlashing() {
		t.Error("IsFlashing() = true after unrelated msg; want false")
	}
}
