package tabbar_test

import (
	"strings"
	"testing"

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
