package modals

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Activation / deactivation
// ---------------------------------------------------------------------------

func TestNewPlanViewModal_StartsInactive(t *testing.T) {
	m := NewPlanViewModal()
	assert.False(t, m.IsActive(), "new modal must start inactive")
}

func TestShow_Activates(t *testing.T) {
	m := NewPlanViewModal()
	m.Show()
	assert.True(t, m.IsActive(), "Show() must set active = true")
}

func TestHide_Deactivates(t *testing.T) {
	m := NewPlanViewModal()
	m.Show()
	require.True(t, m.IsActive())
	m.Hide()
	assert.False(t, m.IsActive(), "Hide() must set active = false")
}

// ---------------------------------------------------------------------------
// SetContent
// ---------------------------------------------------------------------------

func TestSetContent_RendersMarkdown(t *testing.T) {
	m := NewPlanViewModal()
	m.SetSize(120, 40)
	m.SetContent("# Hello Plan\n\nThis is a plan.", 120)
	// Pre-rendered content must be non-empty.
	assert.NotEmpty(t, m.rendered, "SetContent must pre-render markdown into rendered field")
}

func TestSetContent_EmptyMarkdown_NoRenderError(t *testing.T) {
	m := NewPlanViewModal()
	m.SetContent("", 80)
	// Empty content should not panic; rendered may be empty or the raw string.
	// The key contract is that View() returns "" when inactive.
	assert.False(t, m.IsActive())
}

func TestSetContent_PopulatesViewport(t *testing.T) {
	m := NewPlanViewModal()
	m.SetSize(100, 40)
	m.SetContent("## Section\n\nBody text here.", 100)
	// The viewport's total line count should increase after SetContent.
	assert.Greater(t, m.viewport.TotalLineCount(), 0, "viewport must have content after SetContent")
}

// ---------------------------------------------------------------------------
// Update — Esc/q closes
// ---------------------------------------------------------------------------

func TestUpdate_EscCloses(t *testing.T) {
	m := NewPlanViewModal()
	m.Show()
	require.True(t, m.IsActive())

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	assert.False(t, updated.IsActive(), "Esc must deactivate the modal")
	require.NotNil(t, cmd, "Esc must return a non-nil command")
	msg := cmd()
	_, ok := msg.(PlanViewClosedMsg)
	assert.True(t, ok, "command must emit PlanViewClosedMsg, got %T", msg)
}

func TestUpdate_QCloses(t *testing.T) {
	m := NewPlanViewModal()
	m.Show()

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})

	assert.False(t, updated.IsActive(), "'q' must deactivate the modal")
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(PlanViewClosedMsg)
	assert.True(t, ok, "command must emit PlanViewClosedMsg, got %T", msg)
}

func TestUpdate_InactiveIgnoresKeys(t *testing.T) {
	m := NewPlanViewModal()
	// Modal is inactive; pressing Esc must not emit PlanViewClosedMsg.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	assert.False(t, updated.IsActive())
	assert.Nil(t, cmd, "inactive modal must not produce a command")
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func TestView_EmptyWhenInactive(t *testing.T) {
	m := NewPlanViewModal()
	assert.Equal(t, "", m.View(), "View() must return \"\" when inactive")
}

func TestView_ContainsBorder_WhenActive(t *testing.T) {
	m := NewPlanViewModal()
	m.SetSize(120, 40)
	m.SetContent("# Plan\n\nDetails.", 120)
	m.Show()

	view := m.View()
	assert.NotEmpty(t, view, "View() must be non-empty when active")

	// Rounded border characters are multi-byte runes; check for the top-left
	// corner or the horizontal bar which are common to the rounded border.
	hasBorderChar := strings.ContainsRune(view, '─') ||
		strings.ContainsRune(view, '│') ||
		strings.ContainsRune(view, '╭') ||
		strings.ContainsRune(view, '╰')
	assert.True(t, hasBorderChar, "View() must contain border characters when active")
}

func TestView_ContainsTitleWhenActive(t *testing.T) {
	m := NewPlanViewModal()
	m.SetSize(100, 30)
	m.SetContent("## Hello", 100)
	m.Show()

	view := m.View()
	assert.Contains(t, view, "Plan Preview", "View() must contain the title when active")
}

func TestView_ContainsHintLine_WhenActive(t *testing.T) {
	m := NewPlanViewModal()
	m.SetSize(100, 30)
	m.SetContent("content", 100)
	m.Show()

	view := m.View()
	// The footer hint line should contain "close".
	assert.Contains(t, view, "close", "View() must contain the close hint in the footer")
}

// ---------------------------------------------------------------------------
// SetSize
// ---------------------------------------------------------------------------

func TestSetSize_UpdatesViewport(t *testing.T) {
	m := NewPlanViewModal()
	m.SetSize(80, 24)

	// After SetSize the viewport should reflect the computed dimensions.
	// Inner width = outer - border (2); inner height = outer - border (2) -
	// header rows (3) - footer rows (2).
	expectedW := 80 - planBorderFrame
	expectedH := 24 - planBorderFrame - planHeaderRows - planFooterRows
	if expectedW < 1 {
		expectedW = 1
	}
	if expectedH < 1 {
		expectedH = 1
	}

	assert.Equal(t, expectedW, m.viewport.Width, "viewport width after SetSize")
	assert.Equal(t, expectedH, m.viewport.Height, "viewport height after SetSize")
}

func TestSetSize_SmallTerminal_ClampsToMinimum(t *testing.T) {
	m := NewPlanViewModal()
	// Very small terminal: all dimensions should clamp to at least 1.
	m.SetSize(2, 2)
	assert.GreaterOrEqual(t, m.viewport.Width, 1)
	assert.GreaterOrEqual(t, m.viewport.Height, 1)
}

// ---------------------------------------------------------------------------
// Viewport scroll keys
// ---------------------------------------------------------------------------

func TestViewport_ScrollKeys_DoNotPanic(t *testing.T) {
	m := NewPlanViewModal()
	m.SetSize(120, 40)
	// Provide enough content to be scrollable.
	content := strings.Repeat("Line of content\n", 100)
	m.SetContent(content, 120)
	m.Show()

	scrollKeys := []struct {
		name string
		msg  tea.Msg
	}{
		{"down", tea.KeyMsg{Type: tea.KeyDown}},
		{"up", tea.KeyMsg{Type: tea.KeyUp}},
		{"pgdown", tea.KeyMsg{Type: tea.KeyPgDown}},
		{"pgup", tea.KeyMsg{Type: tea.KeyPgUp}},
	}

	for _, tc := range scrollKeys {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				m, _ = m.Update(tc.msg)
			}, "scroll key %q must not panic", tc.name)
			assert.True(t, m.IsActive(), "scroll key must not close the modal")
		})
	}
}

func TestViewport_ScrollDown_AdvancesPosition(t *testing.T) {
	m := NewPlanViewModal()
	m.SetSize(120, 20)
	content := strings.Repeat("Line of content\n", 200)
	m.SetContent(content, 120)
	m.Show()

	initialOffset := m.viewport.YOffset

	// Press down enough times to scroll at least one line.
	for i := 0; i < 5; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	assert.Greater(t, m.viewport.YOffset, initialOffset,
		"pressing down must scroll the viewport")
}

// ---------------------------------------------------------------------------
// ModalType string
// ---------------------------------------------------------------------------

func TestPlanViewModalType_String(t *testing.T) {
	assert.Equal(t, "PlanView", PlanView.String())
}
