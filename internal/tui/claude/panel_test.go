package claude

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/cli"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockClaudeProcess is a mock implementation of ClaudeProcess for testing.
type MockClaudeProcess struct {
	sessionID     string
	events        chan cli.Event
	restartEvents chan cli.RestartEvent
	sendMessages  []string
	sendError     error
	running       bool
}

// NewMockClaudeProcess creates a new mock Claude process.
func NewMockClaudeProcess(sessionID string) *MockClaudeProcess {
	return &MockClaudeProcess{
		sessionID:     sessionID,
		events:        make(chan cli.Event, 100),
		restartEvents: make(chan cli.RestartEvent, 10),
		sendMessages:  make([]string, 0),
		running:       true,
	}
}

func (m *MockClaudeProcess) Send(message string) error {
	if m.sendError != nil {
		return m.sendError
	}
	m.sendMessages = append(m.sendMessages, message)
	return nil
}

func (m *MockClaudeProcess) Events() <-chan cli.Event {
	return m.events
}

func (m *MockClaudeProcess) RestartEvents() <-chan cli.RestartEvent {
	return m.restartEvents
}

func (m *MockClaudeProcess) SessionID() string {
	return m.sessionID
}

func (m *MockClaudeProcess) IsRunning() bool {
	return m.running
}

func (m *MockClaudeProcess) SendEvent(event cli.Event) {
	m.events <- event
}

func (m *MockClaudeProcess) SendRestartEvent(event cli.RestartEvent) {
	m.restartEvents <- event
}

func (m *MockClaudeProcess) Close() {
	close(m.events)
	close(m.restartEvents)
}

// TestPanelModel_NewPanelModel verifies initial panel state
func TestPanelModel_NewPanelModel(t *testing.T) {
	process := NewMockClaudeProcess("test-session-123")
	defer process.Close()

	panel := NewPanelModel(process)

	assert.Equal(t, "test-session-123", panel.sessionID)
	assert.Equal(t, 0, len(panel.messages))
	assert.Equal(t, 0, len(panel.hooks))
	assert.Equal(t, 0.0, panel.cost)
	assert.True(t, panel.focused)
	assert.False(t, panel.streaming)
	assert.NotNil(t, panel.viewport)
	assert.NotNil(t, panel.textarea)
}

// TestPanelModel_HandleInput_Enter verifies message sending on Enter
func TestPanelModel_HandleInput_Enter(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)
	panel.textarea.SetValue("Hello Claude")

	t.Run("sends message on Enter", func(t *testing.T) {
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedPanel, _ := panel.handleInput(msg)

		// Check message was added to history
		assert.Len(t, updatedPanel.messages, 1)
		assert.Equal(t, "user", updatedPanel.messages[0].Role)
		assert.Equal(t, "Hello Claude", updatedPanel.messages[0].Content)

		// Check streaming flag is set
		assert.True(t, updatedPanel.streaming)

		// Check textarea is cleared
		assert.Equal(t, "", updatedPanel.textarea.Value())
	})

	t.Run("ignores Enter when streaming", func(t *testing.T) {
		streamingPanel := panel
		streamingPanel.streaming = true
		streamingPanel.textarea.SetValue("Another message")

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedPanel, _ := streamingPanel.handleInput(msg)

		// Message should not be added
		assert.Len(t, updatedPanel.messages, 0)
	})

	t.Run("ignores Enter with empty textarea", func(t *testing.T) {
		emptyPanel := panel
		emptyPanel.textarea.SetValue("")

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedPanel, _ := emptyPanel.handleInput(msg)

		// Message should not be added
		assert.Len(t, updatedPanel.messages, 0)
	})
}

// TestPanelModel_HandleInput_Esc verifies Esc clears input
func TestPanelModel_HandleInput_Esc(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)
	panel.textarea.SetValue("Some text")

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedPanel, _ := panel.handleInput(msg)

	// Textarea should be cleared
	assert.Equal(t, "", updatedPanel.textarea.Value())
}

// TestPanelModel_HandleInput_CtrlL verifies Ctrl+L clears conversation
func TestPanelModel_HandleInput_CtrlL(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)
	panel.messages = []Message{
		{Role: "user", Content: "Message 1"},
		{Role: "assistant", Content: "Response 1"},
		{Role: "user", Content: "Message 2"},
	}

	msg := tea.KeyMsg{Type: tea.KeyCtrlL}
	updatedPanel, _ := panel.handleInput(msg)

	// Messages should be cleared
	assert.Len(t, updatedPanel.messages, 0)
}

// TestPanelModel_AppendStreamingText verifies streaming text behavior
func TestPanelModel_AppendStreamingText(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)

	t.Run("creates new assistant message when empty", func(t *testing.T) {
		panel.appendStreamingText("Hello")

		require.Len(t, panel.messages, 1)
		assert.Equal(t, "assistant", panel.messages[0].Role)
		assert.Equal(t, "Hello", panel.messages[0].Content)
	})

	t.Run("appends to existing assistant message", func(t *testing.T) {
		panel.appendStreamingText(" World")

		require.Len(t, panel.messages, 1)
		assert.Equal(t, "Hello World", panel.messages[0].Content)
	})

	t.Run("creates new message after user message", func(t *testing.T) {
		panel.messages = append(panel.messages, Message{
			Role:    "user",
			Content: "Question",
		})

		panel.appendStreamingText("Answer")

		require.Len(t, panel.messages, 3)
		assert.Equal(t, "assistant", panel.messages[2].Role)
		assert.Equal(t, "Answer", panel.messages[2].Content)
	})
}

// TestPanelModel_HandleEvent_Assistant verifies assistant event handling
func TestPanelModel_HandleEvent_Assistant(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)

	t.Run("processes assistant event with text content", func(t *testing.T) {
		// Create assistant event JSON
		eventJSON := `{
			"type": "assistant",
			"message": {
				"content": [
					{"type": "text", "text": "Hello from Claude"}
				]
			},
			"partial": true
		}`

		event, err := cli.ParseEvent([]byte(eventJSON))
		require.NoError(t, err)

		updatedPanel := panel.handleEvent(event)

		require.Len(t, updatedPanel.messages, 1)
		assert.Equal(t, "assistant", updatedPanel.messages[0].Role)
		assert.Equal(t, "Hello from Claude", updatedPanel.messages[0].Content)
	})

	t.Run("processes multiple content blocks", func(t *testing.T) {
		eventJSON := `{
			"type": "assistant",
			"message": {
				"content": [
					{"type": "text", "text": "Part 1 "},
					{"type": "text", "text": "Part 2"}
				]
			}
		}`

		event, err := cli.ParseEvent([]byte(eventJSON))
		require.NoError(t, err)

		panel2 := NewPanelModel(process)
		updatedPanel := panel2.handleEvent(event)

		require.Len(t, updatedPanel.messages, 1)
		assert.Equal(t, "Part 1 Part 2", updatedPanel.messages[0].Content)
	})

	t.Run("ignores non-text content blocks", func(t *testing.T) {
		eventJSON := `{
			"type": "assistant",
			"message": {
				"content": [
					{"type": "thinking", "text": "Thinking..."},
					{"type": "text", "text": "Response"}
				]
			}
		}`

		event, err := cli.ParseEvent([]byte(eventJSON))
		require.NoError(t, err)

		panel3 := NewPanelModel(process)
		updatedPanel := panel3.handleEvent(event)

		require.Len(t, updatedPanel.messages, 1)
		assert.Equal(t, "Response", updatedPanel.messages[0].Content)
	})
}

// TestPanelModel_HandleEvent_Result verifies result event handling
func TestPanelModel_HandleEvent_Result(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)
	panel.streaming = true

	eventJSON := `{
		"type": "result",
		"total_cost_usd": 1.23,
		"is_error": false
	}`

	event, err := cli.ParseEvent([]byte(eventJSON))
	require.NoError(t, err)

	updatedPanel := panel.handleEvent(event)

	// Cost should be updated
	assert.Equal(t, 1.23, updatedPanel.cost)

	// Streaming should be stopped
	assert.False(t, updatedPanel.streaming)
}

// TestPanelModel_HandleEvent_System verifies system event handling
func TestPanelModel_HandleEvent_System(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)

	t.Run("processes hook_response with success", func(t *testing.T) {
		eventJSON := `{
			"type": "system",
			"subtype": "hook_response",
			"hook_name": "gogent-validate",
			"exit_code": 0
		}`

		event, err := cli.ParseEvent([]byte(eventJSON))
		require.NoError(t, err)

		updatedPanel := panel.handleEvent(event)

		require.Len(t, updatedPanel.hooks, 1)
		assert.Equal(t, "gogent-validate", updatedPanel.hooks[0].Name)
		assert.True(t, updatedPanel.hooks[0].Success)
	})

	t.Run("processes hook_response with failure", func(t *testing.T) {
		eventJSON := `{
			"type": "system",
			"subtype": "hook_response",
			"hook_name": "gogent-sharp-edge",
			"exit_code": 1
		}`

		event, err := cli.ParseEvent([]byte(eventJSON))
		require.NoError(t, err)

		panel2 := NewPanelModel(process)
		updatedPanel := panel2.handleEvent(event)

		require.Len(t, updatedPanel.hooks, 1)
		assert.Equal(t, "gogent-sharp-edge", updatedPanel.hooks[0].Name)
		assert.False(t, updatedPanel.hooks[0].Success)
	})
}

// TestPanelModel_HandleEvent_Error verifies error event handling
func TestPanelModel_HandleEvent_Error(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)
	panel.streaming = true

	eventJSON := `{
		"type": "error",
		"error": "Connection failed"
	}`

	event, err := cli.ParseEvent([]byte(eventJSON))
	require.NoError(t, err)

	updatedPanel := panel.handleEvent(event)

	// Streaming should be stopped
	assert.False(t, updatedPanel.streaming)

	// Error should be added to messages
	require.Len(t, updatedPanel.messages, 1)
	assert.Contains(t, updatedPanel.messages[0].Content, "Error")
	assert.Contains(t, updatedPanel.messages[0].Content, "Connection failed")
}

// TestPanelModel_RenderHookSidebar verifies hook sidebar rendering
func TestPanelModel_RenderHookSidebar(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)

	t.Run("renders empty state", func(t *testing.T) {
		output := panel.renderHookSidebar(20)
		assert.Contains(t, output, "Recent Hooks")
		assert.Contains(t, output, "No hooks yet")
	})

	t.Run("renders hooks with indicators", func(t *testing.T) {
		panel.hooks = []HookEvent{
			{Name: "hook-1", Success: true},
			{Name: "hook-2", Success: false},
		}

		output := panel.renderHookSidebar(20)
		assert.Contains(t, output, "hook-1")
		assert.Contains(t, output, "hook-2")
		assert.Contains(t, output, "✓") // Success indicator
		assert.Contains(t, output, "✗") // Failure indicator
	})

	t.Run("shows only last 5 hooks", func(t *testing.T) {
		panel2 := NewPanelModel(process)
		for i := 0; i < 10; i++ {
			panel2.hooks = append(panel2.hooks, HookEvent{
				Name:    "hook",
				Success: true,
			})
		}

		output := panel2.renderHookSidebar(20)
		// Count occurrences of checkmark (should be 5)
		count := 0
		for _, r := range output {
			if r == '✓' {
				count++
			}
		}
		assert.Equal(t, 5, count)
	})
}

// TestPanelModel_UpdateViewport verifies viewport content generation
func TestPanelModel_UpdateViewport(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)

	t.Run("renders user and assistant messages", func(t *testing.T) {
		panel.messages = []Message{
			{Role: "user", Content: "Question"},
			{Role: "assistant", Content: "Answer"},
		}

		panel.updateViewport()
		content := panel.viewport.View()

		assert.Contains(t, content, "Question")
		assert.Contains(t, content, "Answer")
	})

	t.Run("shows streaming indicator when streaming", func(t *testing.T) {
		panel2 := NewPanelModel(process)
		panel2.streaming = true
		panel2.messages = []Message{
			{Role: "assistant", Content: "Partial"},
		}

		panel2.updateViewport()
		content := panel2.viewport.View()

		assert.Contains(t, content, "streaming")
	})
}

// TestPanelModel_SetSize verifies size updates
func TestPanelModel_SetSize(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)
	panel.SetSize(100, 50)

	assert.Equal(t, 100, panel.width)
	assert.Equal(t, 50, panel.height)
	assert.Equal(t, 80, panel.viewport.Width)  // 100 - 20 for sidebar (20% of width)
	assert.Equal(t, 45, panel.viewport.Height) // 50 - 5 for header/input
	// Textarea width may be adjusted internally by bubbles, just verify it was set
	assert.Greater(t, panel.textarea.Width(), 90)
}

// TestPanelModel_FocusBlur verifies focus state management
func TestPanelModel_FocusBlur(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)

	// Initially focused
	assert.True(t, panel.focused)

	panel.Blur()
	assert.False(t, panel.focused)

	panel.Focus()
	assert.True(t, panel.focused)
}

// TestPanelModel_ClearConversation verifies conversation clearing
func TestPanelModel_ClearConversation(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)
	panel.messages = []Message{
		{Role: "user", Content: "Test"},
		{Role: "assistant", Content: "Response"},
	}

	panel.ClearConversation()

	assert.Len(t, panel.messages, 0)
}

// TestPanelModel_StreamingWorkflow verifies end-to-end streaming
func TestPanelModel_StreamingWorkflow(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)

	// User sends message
	panel.textarea.SetValue("Tell me a story")
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	panel, _ = panel.handleInput(msg)

	require.Len(t, panel.messages, 1)
	assert.Equal(t, "user", panel.messages[0].Role)
	assert.True(t, panel.streaming)

	// Receive streaming chunks
	chunks := []string{"Once ", "upon ", "a ", "time..."}
	for _, chunk := range chunks {
		eventJSON := map[string]interface{}{
			"type": "assistant",
			"message": map[string]interface{}{
				"content": []map[string]string{
					{"type": "text", "text": chunk},
				},
			},
			"partial": true,
		}
		data, _ := json.Marshal(eventJSON)
		event, _ := cli.ParseEvent(data)
		panel = panel.handleEvent(event)
	}

	require.Len(t, panel.messages, 2)
	assert.Equal(t, "Once upon a time...", panel.messages[1].Content)
	assert.True(t, panel.streaming)

	// Receive result event
	resultJSON := `{
		"type": "result",
		"total_cost_usd": 0.05,
		"is_error": false
	}`
	event, _ := cli.ParseEvent([]byte(resultJSON))
	panel = panel.handleEvent(event)

	assert.False(t, panel.streaming)
	assert.Equal(t, 0.05, panel.cost)
}

// TestPanelModel_HookTracking verifies hook event tracking
func TestPanelModel_HookTracking(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)

	hooks := []struct {
		name    string
		success bool
	}{
		{"gogent-load-context", true},
		{"gogent-validate", true},
		{"gogent-sharp-edge", false},
		{"gogent-agent-endstate", true},
	}

	for _, h := range hooks {
		eventJSON := map[string]interface{}{
			"type":      "system",
			"subtype":   "hook_response",
			"hook_name": h.name,
			"exit_code": func() int {
				if h.success {
					return 0
				}
				return 1
			}(),
		}
		data, _ := json.Marshal(eventJSON)
		event, _ := cli.ParseEvent(data)
		panel = panel.handleEvent(event)
	}

	require.Len(t, panel.hooks, 4)
	assert.Equal(t, "gogent-load-context", panel.hooks[0].Name)
	assert.True(t, panel.hooks[0].Success)
	assert.Equal(t, "gogent-sharp-edge", panel.hooks[2].Name)
	assert.False(t, panel.hooks[2].Success)
}

// TestPanelModel_Update verifies Update method integration
func TestPanelModel_Update(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)

	t.Run("handles WindowSizeMsg", func(t *testing.T) {
		msg := tea.WindowSizeMsg{Width: 120, Height: 40}
		updatedModel, _ := panel.Update(msg)
		updatedPanel := updatedModel.(PanelModel)

		assert.Equal(t, 120, updatedPanel.width)
		assert.Equal(t, 40, updatedPanel.height)
	})

	t.Run("handles cli.Event", func(t *testing.T) {
		eventJSON := `{
			"type": "result",
			"total_cost_usd": 0.15
		}`
		event, _ := cli.ParseEvent([]byte(eventJSON))

		updatedModel, _ := panel.Update(event)
		updatedPanel := updatedModel.(PanelModel)

		assert.Equal(t, 0.15, updatedPanel.cost)
	})

	t.Run("handles errMsg", func(t *testing.T) {
		panel2 := panel
		panel2.streaming = true

		updatedModel, _ := panel2.Update(errMsg{})
		updatedPanel := updatedModel.(PanelModel)

		assert.False(t, updatedPanel.streaming)
	})
}

// TestPanelModel_View verifies View rendering
func TestPanelModel_View(t *testing.T) {
	process := NewMockClaudeProcess("test-session-abc")
	defer process.Close()

	panel := NewPanelModel(process)
	panel.cost = 1.23
	panel.hooks = []HookEvent{
		{Name: "hook-1", Success: true},
	}

	view := panel.View()

	// Check header contains session ID and cost
	assert.Contains(t, view, "test-ses") // Truncated session ID
	assert.Contains(t, view, "$1.23")

	// Check hook sidebar
	assert.Contains(t, view, "Recent Hooks")
	assert.Contains(t, view, "hook-1")
}

// TestTruncate verifies string truncation
func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"shorter than max", "hello", 10, "hello"},
		{"equal to max", "hello", 5, "hello"},
		{"longer than max", "hello world", 5, "hello"},
		{"empty string", "", 5, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := truncate(tc.input, tc.maxLen)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// BenchmarkAppendStreamingText benchmarks streaming text performance
func BenchmarkAppendStreamingText(b *testing.B) {
	process := NewMockClaudeProcess("bench-session")
	defer process.Close()

	panel := NewPanelModel(process)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		panel.appendStreamingText("test ")
	}
}

// BenchmarkHandleEvent benchmarks event processing
func BenchmarkHandleEvent(b *testing.B) {
	process := NewMockClaudeProcess("bench-session")
	defer process.Close()

	panel := NewPanelModel(process)

	eventJSON := `{
		"type": "assistant",
		"message": {
			"content": [{"type": "text", "text": "Response"}]
		}
	}`
	event, _ := cli.ParseEvent([]byte(eventJSON))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		panel = panel.handleEvent(event)
	}
}

// TestPanelModel_Init_SubscribesToRestartEvents verifies Init includes restart event subscription
func TestPanelModel_Init_SubscribesToRestartEvents(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)
	cmd := panel.Init()

	// Init should return a batch command
	require.NotNil(t, cmd)

	// Execute the batch to verify it includes restart subscription
	// This is indirect verification - we can't easily test channel subscription
	// but we verify the command executes without panic
	msg := cmd()
	assert.NotNil(t, msg)
}

// TestPanelModel_HandleRestartEvent_Crash verifies handling of crash restart event
func TestPanelModel_HandleRestartEvent_Crash(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)

	restartEvent := cli.RestartEvent{
		Reason:     "crash",
		AttemptNum: 1,
		SessionID:  "test-session",
		WillResume: true,
		NextDelay:  5 * time.Second,
		Timestamp:  time.Now(),
		ExitCode:   1,
	}

	updatedModel, cmd := panel.Update(restartEvent)
	updatedPanel := updatedModel.(PanelModel)

	// Check restart info is set
	require.NotNil(t, updatedPanel.restartInfo)
	assert.Equal(t, "crash", updatedPanel.restartInfo.Reason)
	assert.Equal(t, 1, updatedPanel.restartInfo.AttemptNum)

	// Check restart message is formatted
	assert.Contains(t, updatedPanel.restartMessage, "Restarting in")
	assert.Contains(t, updatedPanel.restartMessage, "attempt 1")

	// Check re-subscription command is returned
	assert.NotNil(t, cmd)
}

// TestPanelModel_HandleRestartEvent_MaxExceeded verifies max restarts exceeded handling
func TestPanelModel_HandleRestartEvent_MaxExceeded(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)
	panel.streaming = true

	restartEvent := cli.RestartEvent{
		Reason:     "max_restarts_exceeded",
		AttemptNum: 5,
		SessionID:  "test-session",
		WillResume: false,
		NextDelay:  0,
		Timestamp:  time.Now(),
		ExitCode:   139,
	}

	updatedModel, cmd := panel.Update(restartEvent)
	updatedPanel := updatedModel.(PanelModel)

	// Check restart info is set
	require.NotNil(t, updatedPanel.restartInfo)
	assert.Equal(t, "max_restarts_exceeded", updatedPanel.restartInfo.Reason)

	// Check error message is formatted
	assert.Contains(t, updatedPanel.restartMessage, "Max restarts exceeded")
	assert.Contains(t, updatedPanel.restartMessage, "exit 139")

	// Check streaming is stopped
	assert.False(t, updatedPanel.streaming)

	// Check re-subscription command is returned
	assert.NotNil(t, cmd)
}

// TestPanelModel_ClearRestartInfo_OnRecovery verifies restart info is cleared on next event
func TestPanelModel_ClearRestartInfo_OnRecovery(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)

	// Set restart info
	restartEvent := cli.RestartEvent{
		Reason:     "crash",
		AttemptNum: 1,
		SessionID:  "test-session",
		WillResume: true,
		NextDelay:  5 * time.Second,
		Timestamp:  time.Now(),
		ExitCode:   1,
	}
	updatedModel, _ := panel.Update(restartEvent)
	panel = updatedModel.(PanelModel)

	require.NotNil(t, panel.restartInfo)

	// Send a regular event (process recovered)
	eventJSON := `{
		"type": "assistant",
		"message": {
			"content": [{"type": "text", "text": "I'm back!"}]
		}
	}`
	event, _ := cli.ParseEvent([]byte(eventJSON))

	updatedModel, _ = panel.Update(event)
	updatedPanel := updatedModel.(PanelModel)

	// Check restart info is cleared
	assert.Nil(t, updatedPanel.restartInfo)
	assert.Equal(t, "", updatedPanel.restartMessage)
}

// TestPanelModel_View_ShowsRestartIndicator verifies View displays restart status
func TestPanelModel_View_ShowsRestartIndicator(t *testing.T) {
	process := NewMockClaudeProcess("test-session-xyz")
	defer process.Close()

	panel := NewPanelModel(process)

	t.Run("shows restarting indicator", func(t *testing.T) {
		restartEvent := cli.RestartEvent{
			Reason:     "crash",
			AttemptNum: 2,
			SessionID:  "test-session-xyz",
			WillResume: true,
			NextDelay:  10 * time.Second,
			Timestamp:  time.Now(),
			ExitCode:   1,
		}
		updatedModel, _ := panel.Update(restartEvent)
		updatedPanel := updatedModel.(PanelModel)

		view := updatedPanel.View()

		// Check header contains restart message (format: [Restarting in Xs... (attempt N)])
		assert.Contains(t, view, "[Restarting in")
		assert.Contains(t, view, "attempt 2")
	})

	t.Run("shows error indicator for max restarts", func(t *testing.T) {
		panel2 := NewPanelModel(process)
		restartEvent := cli.RestartEvent{
			Reason:     "max_restarts_exceeded",
			AttemptNum: 5,
			SessionID:  "test-session-xyz",
			WillResume: false,
			NextDelay:  0,
			Timestamp:  time.Now(),
			ExitCode:   139,
		}
		updatedModel, _ := panel2.Update(restartEvent)
		updatedPanel := updatedModel.(PanelModel)

		view := updatedPanel.View()

		// Check header contains ERROR indicator
		assert.Contains(t, view, "[ERROR:")
		assert.Contains(t, view, "Max restarts exceeded")
	})

	t.Run("clears indicator after recovery", func(t *testing.T) {
		panel3 := NewPanelModel(process)

		// Set restart state
		restartEvent := cli.RestartEvent{
			Reason:     "crash",
			AttemptNum: 1,
			SessionID:  "test-session-xyz",
			WillResume: true,
			NextDelay:  5 * time.Second,
			Timestamp:  time.Now(),
			ExitCode:   1,
		}
		updatedModel, _ := panel3.Update(restartEvent)
		panel3 = updatedModel.(PanelModel)

		// Send recovery event
		eventJSON := `{
			"type": "result",
			"total_cost_usd": 0.1
		}`
		event, _ := cli.ParseEvent([]byte(eventJSON))
		updatedModel, _ = panel3.Update(event)
		updatedPanel := updatedModel.(PanelModel)

		view := updatedPanel.View()

		// Check no restart indicator
		assert.NotContains(t, view, "[RESTARTING:")
		assert.NotContains(t, view, "[ERROR:")
	})
}

// TestPanelModel_RestartEvent_ReSubscription verifies restart events are re-subscribed
func TestPanelModel_RestartEvent_ReSubscription(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)

	// Send first restart event
	restartEvent1 := cli.RestartEvent{
		Reason:     "crash",
		AttemptNum: 1,
		SessionID:  "test-session",
		WillResume: true,
		NextDelay:  5 * time.Second,
		Timestamp:  time.Now(),
		ExitCode:   1,
	}
	updatedModel, cmd := panel.Update(restartEvent1)
	panel = updatedModel.(PanelModel)

	require.NotNil(t, cmd)
	require.NotNil(t, panel.restartInfo)

	// Send second restart event
	restartEvent2 := cli.RestartEvent{
		Reason:     "crash",
		AttemptNum: 2,
		SessionID:  "test-session",
		WillResume: true,
		NextDelay:  10 * time.Second,
		Timestamp:  time.Now(),
		ExitCode:   1,
	}
	updatedModel, cmd = panel.Update(restartEvent2)
	updatedPanel := updatedModel.(PanelModel)

	// Check second event updated the state
	require.NotNil(t, updatedPanel.restartInfo)
	assert.Equal(t, 2, updatedPanel.restartInfo.AttemptNum)
	assert.Contains(t, updatedPanel.restartMessage, "attempt 2")

	// Check re-subscription command is still returned
	assert.NotNil(t, cmd)
}

// TestPanelModel_RestartEvent_Integration verifies full restart workflow
func TestPanelModel_RestartEvent_Integration(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)

	// 1. Process is running normally
	assert.Nil(t, panel.restartInfo)

	// 2. Process crashes - restart event received
	restartEvent := cli.RestartEvent{
		Reason:     "crash",
		AttemptNum: 1,
		SessionID:  "test-session",
		WillResume: true,
		NextDelay:  5 * time.Second,
		Timestamp:  time.Now(),
		ExitCode:   1,
	}
	updatedModel, _ := panel.Update(restartEvent)
	panel = updatedModel.(PanelModel)

	require.NotNil(t, panel.restartInfo)
	assert.Contains(t, panel.restartMessage, "Restarting")

	// 3. Process recovers - regular event received
	eventJSON := `{
		"type": "assistant",
		"message": {
			"content": [{"type": "text", "text": "I'm back!"}]
		}
	}`
	event, _ := cli.ParseEvent([]byte(eventJSON))
	updatedModel, _ = panel.Update(event)
	panel = updatedModel.(PanelModel)

	// 4. Restart info is cleared
	assert.Nil(t, panel.restartInfo)
	assert.Equal(t, "", panel.restartMessage)

	// 5. Message is added to conversation
	require.Len(t, panel.messages, 1)
	assert.Equal(t, "I'm back!", panel.messages[0].Content)
}

// TestPanelModel_Getters verifies getter methods
func TestPanelModel_Getters(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)

	t.Run("IsStreaming", func(t *testing.T) {
		assert.False(t, panel.IsStreaming())
		panel.streaming = true
		assert.True(t, panel.IsStreaming())
	})

	t.Run("GetMessages", func(t *testing.T) {
		assert.Len(t, panel.GetMessages(), 0)
		panel.messages = []Message{
			{Role: "user", Content: "Test"},
		}
		assert.Len(t, panel.GetMessages(), 1)
		assert.Equal(t, "Test", panel.GetMessages()[0].Content)
	})

	t.Run("GetHooks", func(t *testing.T) {
		assert.Len(t, panel.GetHooks(), 0)
		panel.hooks = []HookEvent{
			{Name: "test-hook", Success: true},
		}
		assert.Len(t, panel.GetHooks(), 1)
		assert.Equal(t, "test-hook", panel.GetHooks()[0].Name)
	})

	t.Run("GetCost", func(t *testing.T) {
		assert.Equal(t, 0.0, panel.GetCost())
		panel.cost = 1.23
		assert.Equal(t, 1.23, panel.GetCost())
	})
}

// TestPanelModel_GetSidebarWidth verifies sidebar width calculation
func TestPanelModel_GetSidebarWidth(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)

	tests := []struct {
		name          string
		width         int
		expectedWidth int
	}{
		{"very narrow", 50, 15},  // 50/5=10, clamped to minSidebarWidth
		{"narrow", 100, 20},      // 100/5=20, within range
		{"wide", 150, 30},        // 150/5=30, at maxSidebarWidth
		{"very wide", 200, 30},   // 200/5=40, clamped to maxSidebarWidth
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			panel.width = tc.width
			assert.Equal(t, tc.expectedWidth, panel.getSidebarWidth())
		})
	}
}

// TestProcessState_String verifies ProcessState string representation
func TestProcessState_String(t *testing.T) {
	tests := []struct {
		state    ProcessState
		expected string
	}{
		{StateConnecting, "Connecting"},
		{StateReady, "Ready"},
		{StateStreaming, "Streaming"},
		{StateRestarting, "Restarting"},
		{StateStopped, "Stopped"},
		{StateError, "Error"},
		{ProcessState(99), "Unknown"}, // Invalid state
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.state.String())
		})
	}
}

// TestProcessState_Icon verifies ProcessState icon representation
func TestProcessState_Icon(t *testing.T) {
	tests := []struct {
		state    ProcessState
		expected string
	}{
		{StateConnecting, "🔄"},
		{StateReady, "🟢"},
		{StateStreaming, "💭"},
		{StateRestarting, "♻️"},
		{StateStopped, "⬛"},
		{StateError, "🔴"},
		{ProcessState(99), "❓"}, // Invalid state
	}

	for _, tc := range tests {
		t.Run(tc.state.String(), func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.state.Icon())
		})
	}
}

// TestPanelModel_InitialState verifies initial state is StateConnecting
func TestPanelModel_InitialState(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)

	assert.Equal(t, StateConnecting, panel.GetState())
}

// TestPanelModel_StateTransition_FirstEvent verifies state transitions to StateReady on first event
func TestPanelModel_StateTransition_FirstEvent(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)
	assert.Equal(t, StateConnecting, panel.GetState())

	// Send first event
	eventJSON := `{
		"type": "system",
		"subtype": "session_start"
	}`
	event, err := cli.ParseEvent([]byte(eventJSON))
	require.NoError(t, err)

	updatedModel, _ := panel.Update(event)
	updatedPanel := updatedModel.(PanelModel)

	// Should transition to Ready
	assert.Equal(t, StateReady, updatedPanel.GetState())
}

// TestPanelModel_StateTransition_MessageSend verifies state transitions to StateStreaming on message send
func TestPanelModel_StateTransition_MessageSend(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)
	panel.state = StateReady // Set to ready first
	panel.textarea.SetValue("Hello")

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedPanel, _ := panel.handleInput(msg)

	// Should transition to Streaming
	assert.Equal(t, StateStreaming, updatedPanel.GetState())
	assert.True(t, updatedPanel.streaming)
}

// TestPanelModel_StateTransition_Result verifies state transitions back to StateReady on result event
func TestPanelModel_StateTransition_Result(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)
	panel.state = StateStreaming
	panel.streaming = true

	eventJSON := `{
		"type": "result",
		"total_cost_usd": 0.15,
		"is_error": false
	}`
	event, err := cli.ParseEvent([]byte(eventJSON))
	require.NoError(t, err)

	updatedPanel := panel.handleEvent(event)

	// Should transition back to Ready
	assert.Equal(t, StateReady, updatedPanel.GetState())
	assert.False(t, updatedPanel.streaming)
}

// TestPanelModel_StateTransition_Restart verifies state transitions to StateRestarting on restart event
func TestPanelModel_StateTransition_Restart(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)
	panel.state = StateReady

	restartEvent := cli.RestartEvent{
		Reason:     "crash",
		AttemptNum: 1,
		SessionID:  "test-session",
		WillResume: true,
		NextDelay:  5 * time.Second,
		Timestamp:  time.Now(),
		ExitCode:   1,
	}

	updatedModel, _ := panel.Update(restartEvent)
	updatedPanel := updatedModel.(PanelModel)

	// Should transition to Restarting
	assert.Equal(t, StateRestarting, updatedPanel.GetState())
}

// TestPanelModel_StateTransition_MaxRestartsExceeded verifies state transitions to StateError on max restarts
func TestPanelModel_StateTransition_MaxRestartsExceeded(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)
	panel.state = StateRestarting

	restartEvent := cli.RestartEvent{
		Reason:     "max_restarts_exceeded",
		AttemptNum: 5,
		SessionID:  "test-session",
		WillResume: false,
		NextDelay:  0,
		Timestamp:  time.Now(),
		ExitCode:   139,
	}

	updatedModel, _ := panel.Update(restartEvent)
	updatedPanel := updatedModel.(PanelModel)

	// Should transition to Error
	assert.Equal(t, StateError, updatedPanel.GetState())
	assert.False(t, updatedPanel.streaming)
}

// TestPanelModel_StateTransition_ProcessStopped verifies state transitions to StateStopped on channel close
func TestPanelModel_StateTransition_ProcessStopped(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)
	panel.state = StateReady

	updatedModel, _ := panel.Update(processStoppedMsg{})
	updatedPanel := updatedModel.(PanelModel)

	// Should transition to Stopped
	assert.Equal(t, StateStopped, updatedPanel.GetState())
	assert.False(t, updatedPanel.streaming)
}

// TestPanelModel_StateTransition_Error verifies state transitions to StateError on error message
func TestPanelModel_StateTransition_Error(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)
	panel.state = StateStreaming
	panel.streaming = true

	updatedModel, _ := panel.Update(errMsg{})
	updatedPanel := updatedModel.(PanelModel)

	// Should transition to Error
	assert.Equal(t, StateError, updatedPanel.GetState())
	assert.False(t, updatedPanel.streaming)
}

// TestPanelModel_View_IncludesStateIcon verifies View includes correct icon for each state
func TestPanelModel_View_IncludesStateIcon(t *testing.T) {
	process := NewMockClaudeProcess("test-session-xyz")
	defer process.Close()

	tests := []struct {
		name         string
		state        ProcessState
		expectedIcon string
	}{
		{"Connecting", StateConnecting, "🔄"},
		{"Ready", StateReady, "🟢"},
		{"Streaming", StateStreaming, "💭"},
		{"Restarting", StateRestarting, "♻️"},
		{"Stopped", StateStopped, "⬛"},
		{"Error", StateError, "🔴"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			panel := NewPanelModel(process)
			panel.state = tc.state

			view := panel.View()

			// Check that the icon appears in the view
			assert.Contains(t, view, tc.expectedIcon)
		})
	}
}

// TestPanelModel_StateTransition_RecoveryFromRestart verifies state clears restart info on recovery
func TestPanelModel_StateTransition_RecoveryFromRestart(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)

	// Set restart state
	restartEvent := cli.RestartEvent{
		Reason:     "crash",
		AttemptNum: 1,
		SessionID:  "test-session",
		WillResume: true,
		NextDelay:  5 * time.Second,
		Timestamp:  time.Now(),
		ExitCode:   1,
	}
	updatedModel, _ := panel.Update(restartEvent)
	panel = updatedModel.(PanelModel)

	require.Equal(t, StateRestarting, panel.GetState())
	require.NotNil(t, panel.restartInfo)

	// Send recovery event
	eventJSON := `{
		"type": "assistant",
		"message": {
			"content": [{"type": "text", "text": "Recovered"}]
		}
	}`
	event, _ := cli.ParseEvent([]byte(eventJSON))
	updatedModel, _ = panel.Update(event)
	updatedPanel := updatedModel.(PanelModel)

	// Should clear restart info and transition to Ready
	assert.Equal(t, StateReady, updatedPanel.GetState())
	assert.Nil(t, updatedPanel.restartInfo)
	assert.Equal(t, "", updatedPanel.restartMessage)
}

// TestPanelModel_FullStateWorkflow verifies complete state transition workflow
func TestPanelModel_FullStateWorkflow(t *testing.T) {
	process := NewMockClaudeProcess("test-session")
	defer process.Close()

	panel := NewPanelModel(process)

	// 1. Initial: Connecting
	assert.Equal(t, StateConnecting, panel.GetState())

	// 2. First event: Connecting → Ready
	eventJSON := `{"type": "system", "subtype": "session_start"}`
	event, _ := cli.ParseEvent([]byte(eventJSON))
	updatedModel, _ := panel.Update(event)
	panel = updatedModel.(PanelModel)
	assert.Equal(t, StateReady, panel.GetState())

	// 3. Send message: Ready → Streaming
	panel.textarea.SetValue("Hello")
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	panel, _ = panel.handleInput(msg)
	assert.Equal(t, StateStreaming, panel.GetState())

	// 4. Receive result: Streaming → Ready
	resultJSON := `{"type": "result", "total_cost_usd": 0.05}`
	resultEvent, _ := cli.ParseEvent([]byte(resultJSON))
	panel = panel.handleEvent(resultEvent)
	assert.Equal(t, StateReady, panel.GetState())

	// 5. Process crash: Ready → Restarting
	restartEvent := cli.RestartEvent{
		Reason:     "crash",
		AttemptNum: 1,
		SessionID:  "test-session",
		WillResume: true,
		NextDelay:  5 * time.Second,
		Timestamp:  time.Now(),
		ExitCode:   1,
	}
	updatedModel, _ = panel.Update(restartEvent)
	panel = updatedModel.(PanelModel)
	assert.Equal(t, StateRestarting, panel.GetState())

	// 6. Recovery: Restarting → Ready
	recoveryJSON := `{"type": "result", "total_cost_usd": 0.05}`
	recoveryEvent, _ := cli.ParseEvent([]byte(recoveryJSON))
	updatedModel, _ = panel.Update(recoveryEvent)
	panel = updatedModel.(PanelModel)
	assert.Equal(t, StateReady, panel.GetState())
}
