package agents

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/telemetry"
)

func TestNewDetailModel(t *testing.T) {
	m := NewDetailModel()

	if m.agent != nil {
		t.Errorf("expected nil agent, got %v", m.agent)
	}
	if m.width != 0 {
		t.Errorf("expected width 0, got %d", m.width)
	}
	if m.height != 0 {
		t.Errorf("expected height 0, got %d", m.height)
	}
}

func TestDetailModel_SetAgent(t *testing.T) {
	m := NewDetailModel()
	agent := &AgentNode{
		AgentID: "test-agent",
		Tier:    "sonnet",
		Status:  StatusRunning,
	}

	m.SetAgent(agent)

	if m.agent != agent {
		t.Errorf("expected agent %v, got %v", agent, m.agent)
	}
}

func TestDetailModel_SetSize(t *testing.T) {
	m := NewDetailModel()

	m.SetSize(80, 24)

	if m.width != 80 {
		t.Errorf("expected width 80, got %d", m.width)
	}
	if m.height != 24 {
		t.Errorf("expected height 24, got %d", m.height)
	}
}

func TestDetailModel_View_NilAgent(t *testing.T) {
	m := NewDetailModel()
	m.SetSize(40, 20)

	view := m.View()

	if !strings.Contains(view, "No agent selected") {
		t.Errorf("expected 'No agent selected' in view, got: %s", view)
	}
}

func TestDetailModel_View_WithAgent(t *testing.T) {
	m := NewDetailModel()
	m.SetSize(60, 20)

	spawnTime := time.Now().Add(-5 * time.Second)
	agent := &AgentNode{
		AgentID:     "go-tui",
		Tier:        "sonnet",
		Status:      StatusRunning,
		Description: "Implement TUI panel with agent delegation tree view",
		SpawnTime:   spawnTime,
	}

	m.SetAgent(agent)
	view := m.View()

	// Check that all expected sections are present
	expectedSections := []string{
		"Selected: go-tui",
		"Tier: sonnet",
		"Status: running",
		"Duration:",
		"Task Description:",
		"Implement TUI panel",
		"[Space] Expand/Collapse",
		"[q] Query agent",
		"[x] Stop agent",
		"[s] Spawn new agent",
	}

	for _, section := range expectedSections {
		if !strings.Contains(view, section) {
			t.Errorf("expected view to contain '%s', got:\n%s", section, view)
		}
	}
}

func TestDetailModel_View_AllStatuses(t *testing.T) {
	testCases := []struct {
		status           AgentStatus
		expectedIndicator string
	}{
		{StatusSpawning, "⏳"},
		{StatusRunning, "⟳"},
		{StatusCompleted, "✓"},
		{StatusError, "✗"},
	}

	for _, tc := range testCases {
		t.Run(string(tc.status), func(t *testing.T) {
			m := NewDetailModel()
			m.SetSize(60, 20)

			spawnTime := time.Now().Add(-2 * time.Second)
			agent := &AgentNode{
				AgentID:     "test-agent",
				Tier:        "haiku",
				Status:      tc.status,
				Description: "Test description",
				SpawnTime:   spawnTime,
			}

			// For completed/error status, set completion time
			if tc.status == StatusCompleted || tc.status == StatusError {
				completeTime := spawnTime.Add(1500 * time.Millisecond)
				agent.CompleteTime = &completeTime
				duration := 1500 * time.Millisecond
				agent.Duration = &duration
			}

			m.SetAgent(agent)
			view := m.View()

			if !strings.Contains(view, tc.expectedIndicator) {
				t.Errorf("expected indicator '%s' for status %s, got:\n%s",
					tc.expectedIndicator, tc.status, view)
			}
		})
	}
}

func TestDetailModel_View_EmptyDescription(t *testing.T) {
	m := NewDetailModel()
	m.SetSize(60, 20)

	agent := &AgentNode{
		AgentID:     "test-agent",
		Tier:        "haiku",
		Status:      StatusRunning,
		Description: "",
		SpawnTime:   time.Now(),
	}

	m.SetAgent(agent)
	view := m.View()

	if !strings.Contains(view, "(no description)") {
		t.Errorf("expected '(no description)' for empty description, got:\n%s", view)
	}
}

func TestDetailModel_View_LongDescription(t *testing.T) {
	m := NewDetailModel()
	m.SetSize(40, 30)

	longDesc := "This is a very long task description that should be wrapped across multiple lines to fit within the available width of the detail panel. It contains many words and should demonstrate the word wrapping functionality."

	agent := &AgentNode{
		AgentID:     "test-agent",
		Tier:        "sonnet",
		Status:      StatusRunning,
		Description: longDesc,
		SpawnTime:   time.Now(),
	}

	m.SetAgent(agent)
	view := m.View()

	// Check that description is present and appears to be wrapped
	lines := strings.Split(view, "\n")
	var descriptionLines []string
	inDescription := false

	for _, line := range lines {
		if strings.Contains(line, "Task Description:") {
			inDescription = true
			continue
		}
		if inDescription && strings.HasPrefix(line, "---") {
			break
		}
		if inDescription && line != "" {
			descriptionLines = append(descriptionLines, line)
		}
	}

	// Should have multiple lines for the wrapped description
	if len(descriptionLines) < 2 {
		t.Errorf("expected description to wrap to multiple lines, got %d lines:\n%v",
			len(descriptionLines), descriptionLines)
	}

	// Each line should be within width constraints (allowing for some padding)
	for _, line := range descriptionLines {
		// Remove ANSI codes for length check
		plainLine := stripAnsi(line)
		if len(plainLine) > m.width-2 {
			t.Errorf("line exceeds width: %d > %d, line: %s",
				len(plainLine), m.width-2, plainLine)
		}
	}
}

func TestDetailModel_statusIndicator(t *testing.T) {
	testCases := []struct {
		status   AgentStatus
		expected string
	}{
		{StatusSpawning, "⏳"},
		{StatusRunning, "⟳"},
		{StatusCompleted, "✓"},
		{StatusError, "✗"},
		{AgentStatus("unknown"), ""},
	}

	for _, tc := range testCases {
		t.Run(string(tc.status), func(t *testing.T) {
			m := DetailModel{
				agent: &AgentNode{Status: tc.status},
			}

			indicator := m.statusIndicator()

			if indicator != tc.expected {
				t.Errorf("expected indicator '%s', got '%s'", tc.expected, indicator)
			}
		})
	}
}

func TestDetailModel_durationString(t *testing.T) {
	t.Run("completed agent", func(t *testing.T) {
		spawnTime := time.Now().Add(-2 * time.Second)
		completeTime := spawnTime.Add(1500 * time.Millisecond)
		duration := 1500 * time.Millisecond

		m := DetailModel{
			agent: &AgentNode{
				SpawnTime:    spawnTime,
				CompleteTime: &completeTime,
				Duration:     &duration,
			},
		}

		durationStr := m.durationString()

		// Should show final duration without ellipsis
		if strings.HasSuffix(durationStr, "...") {
			t.Errorf("completed agent should not have ellipsis: %s", durationStr)
		}
		if !strings.Contains(durationStr, "1.5s") {
			t.Errorf("expected duration ~1.5s, got: %s", durationStr)
		}
	})

	t.Run("active agent", func(t *testing.T) {
		spawnTime := time.Now().Add(-3 * time.Second)

		m := DetailModel{
			agent: &AgentNode{
				SpawnTime:    spawnTime,
				CompleteTime: nil,
			},
		}

		durationStr := m.durationString()

		// Should show elapsed time with ellipsis
		if !strings.HasSuffix(durationStr, "...") {
			t.Errorf("active agent should have ellipsis: %s", durationStr)
		}
		// Should be approximately 3 seconds (allowing some tolerance)
		if !strings.Contains(durationStr, "s") {
			t.Errorf("expected duration in seconds, got: %s", durationStr)
		}
	})
}

func TestWordWrap(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		width    int
		validate func(t *testing.T, result string)
	}{
		{
			name:  "short text, no wrap needed",
			input: "Short text",
			width: 20,
			validate: func(t *testing.T, result string) {
				if strings.Contains(result, "\n") {
					t.Error("short text should not be wrapped")
				}
				if result != "Short text" {
					t.Errorf("expected 'Short text', got '%s'", result)
				}
			},
		},
		{
			name:  "long text, wrap at width",
			input: "This is a long sentence that needs to be wrapped",
			width: 20,
			validate: func(t *testing.T, result string) {
				lines := strings.Split(result, "\n")
				if len(lines) < 2 {
					t.Error("long text should be wrapped to multiple lines")
				}
				for i, line := range lines {
					if len(line) > 20 {
						t.Errorf("line %d exceeds width: %d > 20, line: '%s'",
							i, len(line), line)
					}
				}
			},
		},
		{
			name:  "single long word",
			input: "Supercalifragilisticexpialidocious",
			width: 20,
			validate: func(t *testing.T, result string) {
				// Single word longer than width should not be broken
				if result != "Supercalifragilisticexpialidocious" {
					t.Errorf("single long word should not be modified: '%s'", result)
				}
			},
		},
		{
			name:  "zero width",
			input: "Some text",
			width: 0,
			validate: func(t *testing.T, result string) {
				if result != "Some text" {
					t.Error("zero width should return original text")
				}
			},
		},
		{
			name:  "negative width",
			input: "Some text",
			width: -1,
			validate: func(t *testing.T, result string) {
				if result != "Some text" {
					t.Error("negative width should return original text")
				}
			},
		},
		{
			name:  "text exactly at width",
			input: "Exactly twenty chars",
			width: 20,
			validate: func(t *testing.T, result string) {
				if strings.Contains(result, "\n") {
					t.Error("text exactly at width should not wrap")
				}
			},
		},
		{
			name:  "multiple spaces preserved as single",
			input: "Multiple    spaces    here",
			width: 20,
			validate: func(t *testing.T, result string) {
				// strings.Fields collapses multiple spaces
				if strings.Contains(result, "  ") {
					t.Error("multiple spaces should be collapsed")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := wordWrap(tc.input, tc.width)
			tc.validate(t, result)
		})
	}
}

func TestDetailModel_View_WidthAdaptive(t *testing.T) {
	agent := &AgentNode{
		AgentID:     "test-agent",
		Tier:        "haiku",
		Status:      StatusRunning,
		Description: "Test description for width adaptation",
		SpawnTime:   time.Now(),
	}

	widths := []int{40, 60, 80, 100}

	for _, width := range widths {
		t.Run(fmt.Sprintf("width_%d", width), func(t *testing.T) {
			m := NewDetailModel()
			m.SetSize(width, 20)
			m.SetAgent(agent)

			view := m.View()

			// Check that separators match width
			lines := strings.Split(view, "\n")
			for _, line := range lines {
				plainLine := stripAnsi(line)
				if strings.HasPrefix(plainLine, "---") || strings.HasPrefix(plainLine, "───") {
					expectedLen := width - 2
					if len(plainLine) != expectedLen {
						t.Errorf("separator length mismatch: expected %d, got %d",
							expectedLen, len(plainLine))
					}
				}
			}
		})
	}
}

func TestDetailModel_View_RealTimeUpdate(t *testing.T) {
	// Test that duration updates for active agents
	spawnTime := time.Now().Add(-1 * time.Second)
	agent := &AgentNode{
		AgentID:     "test-agent",
		Tier:        "haiku",
		Status:      StatusRunning,
		Description: "Test",
		SpawnTime:   spawnTime,
	}

	m := NewDetailModel()
	m.SetSize(60, 20)
	m.SetAgent(agent)

	// First view
	view1 := m.View()

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Second view
	view2 := m.View()

	// Both should contain duration, but actual values may differ slightly
	// due to real-time calculation
	if !strings.Contains(view1, "Duration:") || !strings.Contains(view2, "Duration:") {
		t.Error("views should contain duration information")
	}

	// Both should have ellipsis for active agent
	if !strings.Contains(view1, "...") || !strings.Contains(view2, "...") {
		t.Error("active agent duration should have ellipsis")
	}
}

// stripAnsi removes ANSI escape codes from a string for testing
func stripAnsi(s string) string {
	// Simple ANSI stripping - just remove common escape sequences
	// This is a basic implementation for testing purposes
	result := strings.Builder{}
	inEscape := false

	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z') {
				inEscape = false
			}
			continue
		}
		result.WriteByte(s[i])
	}

	return result.String()
}

// TestDetailModel_Integration tests the full integration with AgentNode
func TestDetailModel_Integration(t *testing.T) {
	// Create a realistic agent node with telemetry events
	spawnEvent := &telemetry.AgentLifecycleEvent{
		EventType:       "agent_spawn",
		AgentID:         "go-tui-agent",
		SessionID:       "test-session",
		ParentAgent:     "orchestrator",
		Tier:            "sonnet",
		TaskDescription: "Implement Bubbletea TUI component with agent tree visualization",
		Timestamp:       time.Now().Add(-10 * time.Second).Unix(),
	}

	spawnTime := time.Unix(spawnEvent.Timestamp, 0)
	agent := &AgentNode{
		AgentID:     spawnEvent.AgentID,
		ParentID:    spawnEvent.ParentAgent,
		SessionID:   spawnEvent.SessionID,
		Tier:        spawnEvent.Tier,
		Description: spawnEvent.TaskDescription,
		SpawnEvent:  spawnEvent,
		Status:      StatusRunning,
		SpawnTime:   spawnTime,
		Children:    make([]*AgentNode, 0),
	}

	m := NewDetailModel()
	m.SetSize(70, 25)
	m.SetAgent(agent)

	view := m.View()

	// Validate all components are present
	expectedComponents := []string{
		"Selected: go-tui-agent",
		"Tier: sonnet",
		"Status: running ⟳",
		"Duration:",
		"Task Description:",
		"Implement Bubbletea TUI",
		"[Space] Expand/Collapse",
		"[q] Query agent",
		"[x] Stop agent",
		"[s] Spawn new agent",
	}

	for _, component := range expectedComponents {
		if !strings.Contains(view, component) {
			t.Errorf("expected component '%s' not found in view:\n%s",
				component, view)
		}
	}

	// Test with completion
	completeTime := spawnTime.Add(8500 * time.Millisecond)
	duration := 8500 * time.Millisecond
	agent.CompleteTime = &completeTime
	agent.Duration = &duration
	agent.Status = StatusCompleted

	m.SetAgent(agent)
	view = m.View()

	if !strings.Contains(view, "✓") {
		t.Error("completed agent should show check mark")
	}
	if !strings.Contains(view, "8.5s") {
		t.Error("completed agent should show final duration")
	}
	if strings.Contains(view, "...") {
		t.Error("completed agent should not have ellipsis")
	}
}
