// Package model — reduce_motion setting tests (UX-020).
package model

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/goYoke/internal/tui/components/settingstree"
)

// ---------------------------------------------------------------------------
// trackingTabBar records whether HandleMsg was called.
// ---------------------------------------------------------------------------

type trackingTabBar struct {
	handleMsgCalled bool
	lastMsg         tea.Msg
}

func (tb *trackingTabBar) View() string                { return "" }
func (tb *trackingTabBar) SetWidth(_ int)              {}
func (tb *trackingTabBar) ActiveTab() TabID            { return TabChat }
func (tb *trackingTabBar) HandleMsg(msg tea.Msg) tea.Cmd {
	tb.handleMsgCalled = true
	tb.lastMsg = msg
	return nil
}

// ---------------------------------------------------------------------------
// TestSettingChangedMsg_ReduceMotionOn
// ---------------------------------------------------------------------------

func TestSettingChangedMsg_ReduceMotionOn(t *testing.T) {
	m := NewAppModel()
	panel := &mockClaudePanel{}
	m.SetClaudePanel(panel)

	updated, _ := m.Update(settingstree.SettingChangedMsg{Key: "reduce_motion", Value: "on"})
	result := updated.(AppModel)

	if !result.shared.reduceMotion {
		t.Error("shared.reduceMotion should be true after 'on' message")
	}
	if !result.statusLine.ReduceMotion {
		t.Error("statusLine.ReduceMotion should be true after 'on' message")
	}
}

// ---------------------------------------------------------------------------
// TestSettingChangedMsg_ReduceMotionOff
// ---------------------------------------------------------------------------

func TestSettingChangedMsg_ReduceMotionOff(t *testing.T) {
	m := NewAppModel()
	panel := &mockClaudePanel{}
	m.SetClaudePanel(panel)

	// Turn on first.
	updated, _ := m.Update(settingstree.SettingChangedMsg{Key: "reduce_motion", Value: "on"})
	m = updated.(AppModel)

	// Now turn off.
	updated, _ = m.Update(settingstree.SettingChangedMsg{Key: "reduce_motion", Value: "off"})
	result := updated.(AppModel)

	if result.shared.reduceMotion {
		t.Error("shared.reduceMotion should be false after 'off' message")
	}
	if result.statusLine.ReduceMotion {
		t.Error("statusLine.ReduceMotion should be false after 'off' message")
	}
}

// ---------------------------------------------------------------------------
// TestTabFlash_ReduceMotion_Skipped
// ---------------------------------------------------------------------------

func TestTabFlash_ReduceMotion_Skipped(t *testing.T) {
	m := NewAppModel()
	tb := &trackingTabBar{}
	m.SetTabBar(tb)
	// Activate reduce_motion so the flash should be skipped.
	m.shared.reduceMotion = true

	_, _ = m.Update(TabFlashMsg{TabIndex: 0})

	if tb.handleMsgCalled {
		t.Error("tabBar.HandleMsg should NOT be called when reduceMotion is true")
	}
}

// ---------------------------------------------------------------------------
// TestTabFlash_ReduceMotion_Off_AllowsFlash
// ---------------------------------------------------------------------------

func TestTabFlash_ReduceMotion_Off_AllowsFlash(t *testing.T) {
	m := NewAppModel()
	tb := &trackingTabBar{}
	m.SetTabBar(tb)
	// reduceMotion defaults to false.

	_, _ = m.Update(TabFlashMsg{TabIndex: 0})

	if !tb.handleMsgCalled {
		t.Error("tabBar.HandleMsg SHOULD be called when reduceMotion is false")
	}
}
