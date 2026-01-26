package claude

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/cli"
)

// ProcessState represents the current state of the Claude process
type ProcessState int

const (
	StateConnecting ProcessState = iota // Initial state, waiting for first event
	StateReady                          // Process running, ready for input
	StateStreaming                      // Currently processing a request
	StateRestarting                     // Process crashed, restarting
	StateStopped                        // Process explicitly stopped
	StateError                          // Fatal error occurred
)

// String returns the display name of the state
func (s ProcessState) String() string {
	switch s {
	case StateConnecting:
		return "Connecting"
	case StateReady:
		return "Ready"
	case StateStreaming:
		return "Streaming"
	case StateRestarting:
		return "Restarting"
	case StateStopped:
		return "Stopped"
	case StateError:
		return "Error"
	default:
		return "Unknown"
	}
}

// Icon returns the status icon for the state
func (s ProcessState) Icon() string {
	switch s {
	case StateConnecting:
		return "🔄"
	case StateReady:
		return "🟢"
	case StateStreaming:
		return "💭"
	case StateRestarting:
		return "♻️"
	case StateStopped:
		return "⬛"
	case StateError:
		return "🔴"
	default:
		return "❓"
	}
}

// ClaudeProcessInterface defines the interface for Claude process interaction.
// This allows for easy mocking in tests.
type ClaudeProcessInterface interface {
	Send(message string) error
	Events() <-chan cli.Event
	RestartEvents() <-chan cli.RestartEvent
	SessionID() string
	IsRunning() bool
}

// PanelModel is a Bubble Tea component for Claude CLI conversation.
// It displays streaming output from Claude, handles user input,
// and shows hook events in a sidebar.
type PanelModel struct {
	process        ClaudeProcessInterface
	viewport       viewport.Model
	textarea       textarea.Model
	messages       []Message
	hooks          []HookEvent
	cost           float64
	sessionID      string
	width          int
	height         int
	focused        bool
	streaming      bool
	restartInfo    *cli.RestartEvent // Current restart state (nil if not restarting)
	restartMessage string            // Message to display during restart
	state          ProcessState      // Current process state
	currentModel   string            // Current model name (opus/sonnet/haiku)
	config         cli.Config        // Original config for restart
}

// Message represents a single message in the conversation history.
type Message struct {
	Role    string // "user" or "assistant"
	Content string
}

// HookEvent represents a hook execution event.
type HookEvent struct {
	Name    string
	Success bool
}

const (
	minSidebarWidth = 15
	maxSidebarWidth = 30
)

// NewPanelModel creates a new PanelModel with the given Claude process and config.
// The model is initialized with default dimensions and empty state.
// The config is stored for potential restart operations.
func NewPanelModel(process ClaudeProcessInterface, config cli.Config) PanelModel {
	ta := textarea.New()
	ta.Placeholder = "Type your message here..."
	ta.Focus()
	ta.CharLimit = 4096
	ta.SetWidth(80)
	ta.SetHeight(1)
	ta.ShowLineNumbers = false

	vp := viewport.New(80, 20)

	// Extract current model from config, default to "sonnet" if not set
	currentModel := config.Model
	if currentModel == "" {
		currentModel = "sonnet"
	}

	return PanelModel{
		process:      process,
		viewport:     vp,
		textarea:     ta,
		messages:     make([]Message, 0),
		hooks:        make([]HookEvent, 0),
		sessionID:    process.SessionID(),
		width:        80,
		height:       25,
		focused:      true,
		streaming:    false,
		state:        StateConnecting, // Start in connecting state
		currentModel: currentModel,
		config:       config,
	}
}

// processStoppedMsg is sent when the Claude process event channel closes.
type processStoppedMsg struct{}

// modelChangedMsg indicates model was successfully changed
type modelChangedMsg struct {
	model   string
	process *cli.ClaudeProcess
}

// processErrorMsg wraps errors from process operations
type processErrorMsg struct {
	err error
}

// Init implements tea.Model.Init.
func (m PanelModel) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		waitForEvent(m.process.Events()),
		waitForRestartEvent(m.process.RestartEvents()),
	)
}

// waitForEvent creates a tea.Cmd that blocks on the events channel.
// When an event arrives, it's returned as a tea.Msg for the Update loop.
// When the channel closes, returns processStoppedMsg instead of nil.
// This is the idiomatic Bubbletea pattern for channel subscriptions.
func waitForEvent(events <-chan cli.Event) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-events
		if !ok {
			// Channel closed - process stopped
			return processStoppedMsg{}
		}
		return event
	}
}

// waitForRestartEvent creates a tea.Cmd that blocks on the restart events channel.
// When a restart event arrives, it's returned as a tea.Msg for the Update loop.
// When the channel closes, returns nil.
func waitForRestartEvent(events <-chan cli.RestartEvent) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-events
		if !ok {
			// Channel closed
			return nil
		}
		return event
	}
}

// Update implements tea.Model.Update.
func (m PanelModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.focused {
			var cmd tea.Cmd
			m, cmd = m.handleInput(msg)
			cmds = append(cmds, cmd)
		}

	case cli.Event:
		// First event transitions from Connecting to Ready
		if m.state == StateConnecting {
			m.state = StateReady
		}

		// Clear restart info and state on recovery
		if m.restartInfo != nil && msg.Type != "" {
			m.restartInfo = nil
			m.restartMessage = ""
			m.state = StateReady
		}
		m = m.handleEvent(msg)
		// CRITICAL: Re-subscribe to get the next event
		cmds = append(cmds, waitForEvent(m.process.Events()))

	case cli.RestartEvent:
		m.restartInfo = &msg

		if msg.Reason == "max_restarts_exceeded" {
			m.state = StateError
			m.restartMessage = fmt.Sprintf("Process crashed (exit %d). Max restarts exceeded.", msg.ExitCode)
			m.streaming = false
		} else {
			m.state = StateRestarting
			m.restartMessage = fmt.Sprintf("Restarting in %v... (attempt %d)",
				msg.NextDelay.Truncate(time.Second), msg.AttemptNum)
		}

		// Re-subscribe to restart events
		cmds = append(cmds, waitForRestartEvent(m.process.RestartEvents()))

	case processStoppedMsg:
		// Process stopped - channel closed, don't re-subscribe
		m.state = StateStopped
		m.streaming = false

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - m.getSidebarWidth()
		m.viewport.Height = msg.Height - 5
		m.textarea.SetWidth(msg.Width - 2)

	case tea.MouseMsg:
		if m.focused {
			// Forward mouse events to textarea for click-to-focus
			var taCmd tea.Cmd
			m.textarea, taCmd = m.textarea.Update(msg)
			cmds = append(cmds, taCmd)

			// Also forward to viewport for mouse scrolling
			var vpCmd tea.Cmd
			m.viewport, vpCmd = m.viewport.Update(msg)
			cmds = append(cmds, vpCmd)
		}

	case errMsg:
		// Handle error - stop streaming
		m.state = StateError
		m.streaming = false

	case modelChangedMsg:
		// Update model name
		m.currentModel = msg.model
		// Update process reference
		m.process = msg.process
		// Update config
		m.config.Model = msg.model
		// Add system message
		m.messages = append(m.messages, Message{
			Role:    "system",
			Content: fmt.Sprintf("Model changed to %s (session resumed)", msg.model),
		})
		m.updateViewport()
		// Re-subscribe to new process events
		return m, tea.Batch(
			waitForEvent(m.process.Events()),
			waitForRestartEvent(m.process.RestartEvents()),
		)

	case processErrorMsg:
		// Handle process error
		m.messages = append(m.messages, Message{
			Role:    "system",
			Content: fmt.Sprintf("Error: %v", msg.err),
		})
		m.updateViewport()
		m.state = StateError
		m.streaming = false
	}

	// Update viewport
	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

// View implements tea.Model.View.
func (m PanelModel) View() string {
	// Build status string
	statusIcon := m.state.Icon()

	// Add restart info if applicable
	var statusSuffix string
	if m.restartInfo != nil {
		if m.restartInfo.Reason == "max_restarts_exceeded" {
			statusSuffix = " [ERROR: " + m.restartMessage + "]"
		} else {
			statusSuffix = " [" + m.restartMessage + "]"
		}
	}

	// Header with state icon
	header := headerStyle.Render(fmt.Sprintf(
		"%s Claude Code - Session: %s%s  Cost: $%.2f",
		statusIcon,
		truncate(m.sessionID, 8),
		statusSuffix,
		m.cost,
	))

	// Main content area with viewport and hook sidebar
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.viewport.View(),
		m.renderHookSidebar(m.getSidebarWidth()),
	)

	// Input area with textarea
	input := m.textarea.View()

	// Combine all sections
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		input,
	)
}

// SetSize updates the panel dimensions.
func (m *PanelModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width - m.getSidebarWidth()
	m.viewport.Height = height - 5
	m.textarea.SetWidth(width - 2)
}

// Focus sets the focus state of the panel.
func (m *PanelModel) Focus() {
	m.focused = true
	m.textarea.Focus()
}

// Blur removes focus from the panel.
func (m *PanelModel) Blur() {
	m.focused = false
	m.textarea.Blur()
}

// IsStreaming returns whether the panel is currently streaming output.
func (m PanelModel) IsStreaming() bool {
	return m.streaming
}

// GetMessages returns the current conversation history.
func (m PanelModel) GetMessages() []Message {
	return m.messages
}

// GetHooks returns the current hook event history.
func (m PanelModel) GetHooks() []HookEvent {
	return m.hooks
}

// GetCost returns the current total cost in USD.
func (m PanelModel) GetCost() float64 {
	return m.cost
}

// GetState returns the current process state.
func (m PanelModel) GetState() ProcessState {
	return m.state
}

// ClearConversation clears the conversation history (visual only).
func (m *PanelModel) ClearConversation() {
	m.messages = make([]Message, 0)
	m.updateViewport()
}

// getSidebarWidth calculates the sidebar width based on terminal width.
// Allocates 20% to sidebar, clamped between minSidebarWidth and maxSidebarWidth.
func (m *PanelModel) getSidebarWidth() int {
	sidebarWidth := m.width / 5
	if sidebarWidth < minSidebarWidth {
		sidebarWidth = minSidebarWidth
	}
	if sidebarWidth > maxSidebarWidth {
		sidebarWidth = maxSidebarWidth
	}
	return sidebarWidth
}

// errMsg wraps an error for use in Bubble Tea messages.
type errMsg struct{ error }

// truncate truncates a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// CurrentModel returns the current model name.
func (m PanelModel) CurrentModel() string {
	return m.currentModel
}

// requestModelChange returns a command that restarts the process with a new model.
// This method stops the current process, updates the config with the new model,
// creates a new process, and starts it. If successful, returns modelChangedMsg
// with the new process. On error, returns processErrorMsg.
func (m PanelModel) requestModelChange(model string) tea.Cmd {
	return func() tea.Msg {
		// Cast to concrete type to access Stop method
		proc, ok := m.process.(*cli.ClaudeProcess)
		if !ok {
			return processErrorMsg{err: fmt.Errorf("cannot restart: process is not a ClaudeProcess")}
		}

		// Stop current process
		if err := proc.Stop(); err != nil {
			return processErrorMsg{err: fmt.Errorf("stop process: %w", err)}
		}

		// Update config with new model
		newConfig := m.config
		newConfig.Model = model
		// Preserve session ID for continuity
		newConfig.SessionID = m.sessionID

		// Create new process
		newProc, err := cli.NewClaudeProcess(newConfig)
		if err != nil {
			return processErrorMsg{err: fmt.Errorf("create new process: %w", err)}
		}

		// Start new process
		if err := newProc.Start(); err != nil {
			return processErrorMsg{err: fmt.Errorf("start new process: %w", err)}
		}

		// Return success with new process
		return modelChangedMsg{
			model:   model,
			process: newProc,
		}
	}
}

// Styles for rendering
var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			Padding(0, 1)
)
