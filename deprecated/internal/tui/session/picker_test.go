package session

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Bucket-Chemist/GOgent-Fortress/deprecated/internal/cli"
)

func TestNewPickerModel(t *testing.T) {
	sessions := []cli.Session{
		{ID: "session-1", LastUsed: time.Now()},
		{ID: "session-2", LastUsed: time.Now()},
	}

	m := NewPickerModel(sessions)

	assert.Len(t, m.sessions, 2)
	assert.Equal(t, 0, m.cursor)
	assert.Nil(t, m.selected)
}

func TestPickerModel_NavigationDown(t *testing.T) {
	sessions := []cli.Session{
		{ID: "session-1", LastUsed: time.Now()},
		{ID: "session-2", LastUsed: time.Now()},
		{ID: "session-3", LastUsed: time.Now()},
	}

	m := NewPickerModel(sessions)

	// Navigate down
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = model.(PickerModel)
	assert.Equal(t, 1, m.cursor)

	// Navigate down again
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = model.(PickerModel)
	assert.Equal(t, 2, m.cursor)

	// Try to go past end (should stay at 2)
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = model.(PickerModel)
	assert.Equal(t, 2, m.cursor)
}

func TestPickerModel_NavigationUp(t *testing.T) {
	sessions := []cli.Session{
		{ID: "session-1", LastUsed: time.Now()},
		{ID: "session-2", LastUsed: time.Now()},
		{ID: "session-3", LastUsed: time.Now()},
	}

	m := NewPickerModel(sessions)
	m.cursor = 2 // Start at bottom

	// Navigate up
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = model.(PickerModel)
	assert.Equal(t, 1, m.cursor)

	// Navigate up again
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = model.(PickerModel)
	assert.Equal(t, 0, m.cursor)

	// Try to go past start (should stay at 0)
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = model.(PickerModel)
	assert.Equal(t, 0, m.cursor)
}

func TestPickerModel_NavigationVim(t *testing.T) {
	sessions := []cli.Session{
		{ID: "session-1", LastUsed: time.Now()},
		{ID: "session-2", LastUsed: time.Now()},
	}

	m := NewPickerModel(sessions)

	// j = down
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	model, _ := m.Update(keyMsg)
	m = model.(PickerModel)
	assert.Equal(t, 1, m.cursor)

	// k = up
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	model, _ = m.Update(keyMsg)
	m = model.(PickerModel)
	assert.Equal(t, 0, m.cursor)
}

func TestPickerModel_SelectSession(t *testing.T) {
	sessions := []cli.Session{
		{ID: "session-1", Name: "First", LastUsed: time.Now()},
		{ID: "session-2", Name: "Second", LastUsed: time.Now()},
	}

	m := NewPickerModel(sessions)
	m.cursor = 1 // Select second session

	// Press Enter
	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(PickerModel)

	assert.NotNil(t, m.selected)
	assert.Equal(t, "session-2", m.selected.ID)
	assert.Equal(t, "Second", m.selected.Name)
	assert.NotNil(t, cmd) // Should return tea.Quit
}

func TestPickerModel_Cancel(t *testing.T) {
	sessions := []cli.Session{
		{ID: "session-1", LastUsed: time.Now()},
	}

	m := NewPickerModel(sessions)

	// Press Esc
	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = model.(PickerModel)

	assert.Nil(t, m.selected)
	assert.NotNil(t, cmd) // Should return tea.Quit

	// Reset and try 'q'
	m = NewPickerModel(sessions)
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	model, cmd = m.Update(keyMsg)
	m = model.(PickerModel)

	assert.Nil(t, m.selected)
	assert.NotNil(t, cmd)
}

func TestPickerModel_DeleteSession(t *testing.T) {
	sessions := []cli.Session{
		{ID: "session-1", Name: "First", LastUsed: time.Now()},
		{ID: "session-2", Name: "Second", LastUsed: time.Now()},
		{ID: "session-3", Name: "Third", LastUsed: time.Now()},
	}

	m := NewPickerModel(sessions)
	m.cursor = 1 // Select second session

	// Press 'd' to delete
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	model, cmd := m.Update(keyMsg)
	m = model.(PickerModel)

	assert.NotNil(t, cmd)

	// Execute the command to get the message
	msg := cmd()
	deleteMsg, ok := msg.(sessionDeletedMsg)
	require.True(t, ok)
	assert.Equal(t, "session-2", deleteMsg.id)

	// Process the delete message
	model, _ = m.Update(deleteMsg)
	m = model.(PickerModel)

	// Verify session was removed
	assert.Len(t, m.sessions, 2)
	assert.Equal(t, "session-1", m.sessions[0].ID)
	assert.Equal(t, "session-3", m.sessions[1].ID)
	// Cursor should stay at 1
	assert.Equal(t, 1, m.cursor)
}

func TestPickerModel_DeleteLastSession(t *testing.T) {
	sessions := []cli.Session{
		{ID: "session-1", LastUsed: time.Now()},
		{ID: "session-2", LastUsed: time.Now()},
	}

	m := NewPickerModel(sessions)
	m.cursor = 1 // Last session

	// Delete last session
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	model, cmd := m.Update(keyMsg)
	m = model.(PickerModel)
	msg := cmd()
	model, _ = m.Update(msg)
	m = model.(PickerModel)

	// Cursor should move back
	assert.Len(t, m.sessions, 1)
	assert.Equal(t, 0, m.cursor)
}

func TestPickerModel_View(t *testing.T) {
	sessions := []cli.Session{
		{
			ID:        "abc123def456",
			Name:      "My Session",
			LastUsed:  time.Now().Add(-2 * time.Hour),
			Cost:      1.25,
			ToolCalls: 15,
		},
		{
			ID:        "xyz789",
			LastUsed:  time.Now().Add(-30 * time.Minute),
			Cost:      0.50,
			ToolCalls: 8,
		},
	}

	m := NewPickerModel(sessions)
	m.width = 80
	m.height = 20

	view := m.View()

	// Check for expected content
	assert.Contains(t, view, "Select Session")
	assert.Contains(t, view, "My Session") // Named session
	assert.Contains(t, view, "xyz789")     // ID for unnamed session
	assert.Contains(t, view, "$1.25")
	assert.Contains(t, view, "$0.50")
	assert.Contains(t, view, "15 tools")
	assert.Contains(t, view, "8 tools")
	assert.Contains(t, view, "Navigate")
	assert.Contains(t, view, "Resume")
	assert.Contains(t, view, "Delete")
}

func TestPickerModel_ViewEmpty(t *testing.T) {
	m := NewPickerModel([]cli.Session{})
	m.width = 80
	m.height = 20

	view := m.View()

	assert.Contains(t, view, "No Sessions Found")
	assert.Contains(t, view, "Close")
}

func TestPickerModel_RenderSession(t *testing.T) {
	now := time.Now()
	session := cli.Session{
		ID:        "test123456789",
		Name:      "Test Session",
		LastUsed:  now.Add(-3 * time.Hour),
		Cost:      2.50,
		ToolCalls: 25,
	}

	m := NewPickerModel([]cli.Session{session})

	// Not selected
	line := m.renderSession(session, false)
	assert.Contains(t, line, "Test Session")
	assert.Contains(t, line, "$2.50")
	assert.Contains(t, line, "25 tools")
	assert.Contains(t, line, "3h ago")
	assert.True(t, strings.HasPrefix(line, "  ")) // No selection indicator

	// Selected
	line = m.renderSession(session, true)
	assert.Contains(t, line, "> ") // Selection indicator
}

func TestPickerModel_RenderSessionTruncatesID(t *testing.T) {
	session := cli.Session{
		ID:       "very-long-session-id-123456789",
		LastUsed: time.Now(),
	}

	m := NewPickerModel([]cli.Session{session})
	line := m.renderSession(session, false)

	// Should truncate to 8 chars when no name
	assert.Contains(t, line, "very-lon")
	assert.NotContains(t, line, "very-long-session-id")
}

func TestPickerModel_SetSize(t *testing.T) {
	m := NewPickerModel([]cli.Session{})

	m.SetSize(100, 50)

	assert.Equal(t, 100, m.width)
	assert.Equal(t, 50, m.height)
}

func TestFormatAge(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{
			name:     "less than 1 minute",
			time:     now.Add(-30 * time.Second),
			expected: "<1m ago",
		},
		{
			name:     "5 minutes",
			time:     now.Add(-5 * time.Minute),
			expected: "5m ago",
		},
		{
			name:     "30 minutes",
			time:     now.Add(-30 * time.Minute),
			expected: "30m ago",
		},
		{
			name:     "59 minutes",
			time:     now.Add(-59 * time.Minute),
			expected: "59m ago",
		},
		{
			name:     "1 hour",
			time:     now.Add(-1 * time.Hour),
			expected: "1h ago",
		},
		{
			name:     "5 hours",
			time:     now.Add(-5 * time.Hour),
			expected: "5h ago",
		},
		{
			name:     "23 hours",
			time:     now.Add(-23 * time.Hour),
			expected: "23h ago",
		},
		{
			name:     "1 day",
			time:     now.Add(-24 * time.Hour),
			expected: "1d ago",
		},
		{
			name:     "3 days",
			time:     now.Add(-3 * 24 * time.Hour),
			expected: "3d ago",
		},
		{
			name:     "30 days",
			time:     now.Add(-30 * 24 * time.Hour),
			expected: "30d ago",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := formatAge(tc.time)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestPickerModel_Selected(t *testing.T) {
	sessions := []cli.Session{
		{ID: "session-1", LastUsed: time.Now()},
	}

	m := NewPickerModel(sessions)

	// Initially nil
	assert.Nil(t, m.Selected())

	// After selection
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(PickerModel)
	selected := m.Selected()
	require.NotNil(t, selected)
	assert.Equal(t, "session-1", selected.ID)
}

func TestPickerModel_WindowSizeMsg(t *testing.T) {
	m := NewPickerModel([]cli.Session{})

	sizeMsg := tea.WindowSizeMsg{Width: 120, Height: 60}
	model, _ := m.Update(sizeMsg)
	m = model.(PickerModel)

	assert.Equal(t, 120, m.width)
	assert.Equal(t, 60, m.height)
}
