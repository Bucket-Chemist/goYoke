package agents

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/cli"
)

// Helper to create test agents
func testAgents() []cli.SubagentConfig {
	return []cli.SubagentConfig{
		{
			Name:        "security-reviewer",
			Description: "Analyzes code for security vulnerabilities",
			Tier:        "sonnet",
		},
		{
			Name:        "code-reviewer",
			Description: "Reviews code quality and patterns",
			Tier:        "sonnet",
		},
		{
			Name:        "go-pro",
			Description: "Go implementation specialist",
			Tier:        "sonnet",
		},
		{
			Name:        "go-tui",
			Description: "Bubbletea TUI specialist",
			Tier:        "sonnet",
		},
		{
			Name:        "haiku-scout",
			Description: "Fast codebase exploration",
			Tier:        "haiku",
		},
	}
}

func TestNewPickerModel(t *testing.T) {
	agents := testAgents()
	m := NewPickerModel(agents)

	if len(m.agents) != len(agents) {
		t.Errorf("expected %d agents, got %d", len(agents), len(m.agents))
	}

	if len(m.filtered) != len(agents) {
		t.Errorf("expected %d filtered agents, got %d", len(agents), len(m.filtered))
	}

	if m.cursor != 0 {
		t.Errorf("expected cursor at 0, got %d", m.cursor)
	}

	if m.filterMode {
		t.Error("expected filterMode to be false")
	}

	if m.filter != "" {
		t.Errorf("expected empty filter, got %q", m.filter)
	}
}

func TestPickerModel_Navigation(t *testing.T) {
	tests := []struct {
		name           string
		initialCursor  int
		key            string
		expectedCursor int
	}{
		{
			name:           "down from start",
			initialCursor:  0,
			key:            "down",
			expectedCursor: 1,
		},
		{
			name:           "up from start stays at 0",
			initialCursor:  0,
			key:            "up",
			expectedCursor: 0,
		},
		{
			name:           "down from middle",
			initialCursor:  2,
			key:            "down",
			expectedCursor: 3,
		},
		{
			name:           "up from middle",
			initialCursor:  2,
			key:            "up",
			expectedCursor: 1,
		},
		{
			name:           "down at end stays at end",
			initialCursor:  4,
			key:            "down",
			expectedCursor: 4,
		},
		{
			name:           "j moves down",
			initialCursor:  0,
			key:            "j",
			expectedCursor: 1,
		},
		{
			name:           "k moves up",
			initialCursor:  2,
			key:            "k",
			expectedCursor: 1,
		},
		{
			name:           "home goes to start",
			initialCursor:  3,
			key:            "home",
			expectedCursor: 0,
		},
		{
			name:           "g goes to start",
			initialCursor:  3,
			key:            "g",
			expectedCursor: 0,
		},
		{
			name:           "end goes to last",
			initialCursor:  0,
			key:            "end",
			expectedCursor: 4,
		},
		{
			name:           "G goes to last",
			initialCursor:  0,
			key:            "G",
			expectedCursor: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewPickerModel(testAgents())
			m.cursor = tt.initialCursor

			newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})
			m = newModel.(PickerModel)

			if m.cursor != tt.expectedCursor {
				t.Errorf("expected cursor at %d, got %d", tt.expectedCursor, m.cursor)
			}
		})
	}
}

func TestPickerModel_CursorBounds(t *testing.T) {
	m := NewPickerModel(testAgents())

	// Try to move up from position 0
	for i := 0; i < 10; i++ {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
		m = newModel.(PickerModel)
	}

	if m.cursor != 0 {
		t.Errorf("cursor should not go below 0, got %d", m.cursor)
	}

	// Try to move down beyond list
	m.cursor = len(m.filtered) - 1
	for i := 0; i < 10; i++ {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = newModel.(PickerModel)
	}

	maxCursor := len(m.filtered) - 1
	if m.cursor != maxCursor {
		t.Errorf("cursor should not exceed %d, got %d", maxCursor, m.cursor)
	}
}

func TestPickerModel_Enter(t *testing.T) {
	m := NewPickerModel(testAgents())
	m.cursor = 2 // go-pro

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Fatal("expected command, got nil")
	}

	msg := cmd()
	spawnMsg, ok := msg.(SpawnAgentMsg)
	if !ok {
		t.Fatalf("expected SpawnAgentMsg, got %T", msg)
	}

	expectedName := "go-pro"
	if spawnMsg.AgentName != expectedName {
		t.Errorf("expected agent name %q, got %q", expectedName, spawnMsg.AgentName)
	}
}

func TestPickerModel_EnterEmptyList(t *testing.T) {
	m := NewPickerModel([]cli.SubagentConfig{})

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd != nil {
		t.Error("expected no command for empty list")
	}
}

func TestPickerModel_Escape(t *testing.T) {
	m := NewPickerModel(testAgents())

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if cmd == nil {
		t.Fatal("expected command, got nil")
	}

	msg := cmd()
	_, ok := msg.(PickerCancelMsg)
	if !ok {
		t.Fatalf("expected PickerCancelMsg, got %T", msg)
	}
}

func TestPickerModel_FilterMode(t *testing.T) {
	m := NewPickerModel(testAgents())

	// Enter filter mode with '/'
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = newModel.(PickerModel)

	if !m.filterMode {
		t.Error("expected filterMode to be true")
	}

	if m.filter != "" {
		t.Errorf("expected empty filter, got %q", m.filter)
	}
}

func TestPickerModel_FilterInput(t *testing.T) {
	m := NewPickerModel(testAgents())
	m.filterMode = true

	// Type "go"
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	m = newModel.(PickerModel)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})
	m = newModel.(PickerModel)

	if m.filter != "go" {
		t.Errorf("expected filter 'go', got %q", m.filter)
	}

	// Should filter to only go-pro and go-tui
	if len(m.filtered) != 2 {
		t.Errorf("expected 2 filtered agents, got %d", len(m.filtered))
	}

	// Check that filtered agents are correct
	names := make(map[string]bool)
	for _, agent := range m.filtered {
		names[agent.Name] = true
	}

	if !names["go-pro"] || !names["go-tui"] {
		t.Error("expected go-pro and go-tui in filtered list")
	}
}

func TestPickerModel_FilterBackspace(t *testing.T) {
	m := NewPickerModel(testAgents())
	m.filterMode = true
	m.filter = "go"
	m.applyFilter()

	if len(m.filtered) != 2 {
		t.Fatalf("expected 2 filtered agents, got %d", len(m.filtered))
	}

	// Backspace once
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = newModel.(PickerModel)

	if m.filter != "g" {
		t.Errorf("expected filter 'g', got %q", m.filter)
	}

	// Should now match more agents (go-pro, go-tui, security-reviewer, code-reviewer)
	if len(m.filtered) < 2 {
		t.Errorf("expected more filtered agents after backspace, got %d", len(m.filtered))
	}

	// Backspace again
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = newModel.(PickerModel)

	if m.filter != "" {
		t.Errorf("expected empty filter, got %q", m.filter)
	}

	// Should show all agents
	if len(m.filtered) != len(testAgents()) {
		t.Errorf("expected all %d agents, got %d", len(testAgents()), len(m.filtered))
	}

	// Backspace on empty filter should do nothing
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = newModel.(PickerModel)

	if m.filter != "" {
		t.Error("backspace on empty filter should do nothing")
	}
}

func TestPickerModel_FilterExit(t *testing.T) {
	m := NewPickerModel(testAgents())
	m.filterMode = true
	m.filter = "go"

	tests := []struct {
		name string
		key  tea.KeyType
	}{
		{"enter", tea.KeyEnter},
		{"esc", tea.KeyEsc},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testM := m
			newModel, _ := testM.Update(tea.KeyMsg{Type: tt.key})
			testM = newModel.(PickerModel)

			if testM.filterMode {
				t.Error("expected filterMode to be false")
			}

			if testM.filter != "go" {
				t.Errorf("expected filter to remain 'go', got %q", testM.filter)
			}
		})
	}
}

func TestPickerModel_FilterCaseInsensitive(t *testing.T) {
	m := NewPickerModel(testAgents())
	m.filter = "GO"
	m.applyFilter()

	if len(m.filtered) != 2 {
		t.Errorf("expected 2 filtered agents (case insensitive), got %d", len(m.filtered))
	}
}

func TestPickerModel_FilterByDescription(t *testing.T) {
	m := NewPickerModel(testAgents())
	m.filter = "security"
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Fatalf("expected 1 filtered agent, got %d", len(m.filtered))
	}

	if m.filtered[0].Name != "security-reviewer" {
		t.Errorf("expected security-reviewer, got %s", m.filtered[0].Name)
	}
}

func TestPickerModel_FilterCursorBounds(t *testing.T) {
	m := NewPickerModel(testAgents())
	m.cursor = 4 // Last agent

	// Filter to reduce list
	m.filter = "go"
	m.applyFilter()

	// Cursor should be adjusted to within new bounds
	if m.cursor >= len(m.filtered) {
		t.Errorf("cursor %d should be within filtered list of length %d", m.cursor, len(m.filtered))
	}
}

func TestPickerModel_EmptyAgentList(t *testing.T) {
	m := NewPickerModel([]cli.SubagentConfig{})

	view := m.View()
	if !strings.Contains(view, "No agents match filter") {
		t.Error("expected 'No agents match filter' message in view")
	}
}

func TestPickerModel_View(t *testing.T) {
	m := NewPickerModel(testAgents())
	m.width = 80
	m.height = 20

	view := m.View()

	// Should contain title
	if !strings.Contains(view, "Select Agent to Spawn") {
		t.Error("view should contain title")
	}

	// Should contain help text
	if !strings.Contains(view, "navigate") {
		t.Error("view should contain help text")
	}

	// Should contain agent names
	if !strings.Contains(view, "security-reviewer") {
		t.Error("view should contain agent names")
	}
}

func TestPickerModel_ViewWithFilter(t *testing.T) {
	m := NewPickerModel(testAgents())
	m.width = 80
	m.height = 20
	m.filter = "go"
	m.applyFilter()

	view := m.View()

	// Should show filter
	if !strings.Contains(view, "Filter: go") {
		t.Error("view should show filter")
	}

	// Should only show filtered agents
	if !strings.Contains(view, "go-pro") {
		t.Error("view should contain go-pro")
	}

	if !strings.Contains(view, "go-tui") {
		t.Error("view should contain go-tui")
	}

	// Should NOT show filtered out agents
	if strings.Contains(view, "security-reviewer") {
		t.Error("view should not contain filtered out agents")
	}
}

func TestPickerModel_ViewFilterMode(t *testing.T) {
	m := NewPickerModel(testAgents())
	m.width = 80
	m.height = 20
	m.filterMode = true
	m.filter = "go"

	view := m.View()

	// Should show filter with cursor
	if !strings.Contains(view, "Filter: go█") {
		t.Error("view should show filter with cursor in filter mode")
	}
}

func TestPickerModel_SetSize(t *testing.T) {
	m := NewPickerModel(testAgents())
	m.SetSize(100, 50)

	if m.width != 100 {
		t.Errorf("expected width 100, got %d", m.width)
	}

	if m.height != 50 {
		t.Errorf("expected height 50, got %d", m.height)
	}
}

func TestPickerModel_FocusBlur(t *testing.T) {
	m := NewPickerModel(testAgents())

	if m.focused {
		t.Error("expected model to start unfocused")
	}

	m.Focus()
	if !m.focused {
		t.Error("expected model to be focused")
	}

	m.Blur()
	if m.focused {
		t.Error("expected model to be blurred")
	}
}

func TestPickerModel_GetSelected(t *testing.T) {
	m := NewPickerModel(testAgents())

	tests := []struct {
		name           string
		cursor         int
		expectedAgent  string
	}{
		{"first agent", 0, "security-reviewer"},
		{"second agent", 1, "code-reviewer"},
		{"third agent", 2, "go-pro"},
		{"fourth agent", 3, "go-tui"},
		{"fifth agent", 4, "haiku-scout"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.cursor = tt.cursor
			selected := m.GetSelected()

			if selected != tt.expectedAgent {
				t.Errorf("expected %q, got %q", tt.expectedAgent, selected)
			}
		})
	}
}

func TestPickerModel_GetSelectedEmpty(t *testing.T) {
	m := NewPickerModel([]cli.SubagentConfig{})

	selected := m.GetSelected()
	if selected != "" {
		t.Errorf("expected empty string for empty list, got %q", selected)
	}
}

func TestPickerModel_GetSelectedOutOfBounds(t *testing.T) {
	m := NewPickerModel(testAgents())
	m.cursor = 999 // Way out of bounds

	selected := m.GetSelected()
	if selected != "" {
		t.Errorf("expected empty string for out of bounds cursor, got %q", selected)
	}
}

func TestPickerModel_WindowSizeMsg(t *testing.T) {
	m := NewPickerModel(testAgents())

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	m = newModel.(PickerModel)

	if m.width != 120 {
		t.Errorf("expected width 120, got %d", m.width)
	}

	if m.height != 30 {
		t.Errorf("expected height 30, got %d", m.height)
	}
}

func TestPickerModel_Init(t *testing.T) {
	m := NewPickerModel(testAgents())

	cmd := m.Init()
	if cmd != nil {
		t.Error("expected Init to return nil")
	}
}

func TestPickerModel_SelectAgentBounds(t *testing.T) {
	m := NewPickerModel(testAgents())
	m.cursor = 999 // Out of bounds

	cmd := m.selectAgent()
	if cmd != nil {
		t.Error("expected no command when cursor out of bounds")
	}
}

func TestPickerModel_FilterWithEmptyResult(t *testing.T) {
	m := NewPickerModel(testAgents())
	m.filter = "nonexistent"
	m.applyFilter()

	if len(m.filtered) != 0 {
		t.Errorf("expected 0 filtered agents, got %d", len(m.filtered))
	}

	view := m.View()
	if !strings.Contains(view, "No agents match filter") {
		t.Error("expected 'No agents match filter' message")
	}
}

func TestPickerModel_NavigationKeys(t *testing.T) {
	tests := []struct {
		name      string
		keyType   tea.KeyType
		keyString string
		keyRunes  []rune
	}{
		{"up arrow", tea.KeyUp, "", nil},
		{"down arrow", tea.KeyDown, "", nil},
		{"enter", tea.KeyEnter, "", nil},
		{"escape", tea.KeyEsc, "", nil},
		{"backspace", tea.KeyBackspace, "", nil},
		{"k key", tea.KeyRunes, "k", []rune("k")},
		{"j key", tea.KeyRunes, "j", []rune("j")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewPickerModel(testAgents())

			msg := tea.KeyMsg{Type: tt.keyType}
			if tt.keyRunes != nil {
				msg.Runes = tt.keyRunes
			}

			// Should not panic
			_, _ = m.Update(msg)
		})
	}
}

func TestPickerModel_ScrollSupport(t *testing.T) {
	m := NewPickerModel(testAgents())
	m.width = 80
	m.height = 10 // Small height to trigger scrolling

	// Move cursor to bottom
	for i := 0; i < len(m.filtered)-1; i++ {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = newModel.(PickerModel)
	}

	view := m.View()

	// View should still render (testing no panic on scroll calculation)
	if view == "" {
		t.Error("view should not be empty")
	}
}
