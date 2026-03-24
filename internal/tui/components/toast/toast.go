// Package toast implements a transient notification overlay for the
// GOgent-Fortress TUI. Toasts are small bordered boxes that appear at the
// top-right of the terminal and expire automatically after a configurable
// duration.
//
// The component follows Bubbletea's Elm Architecture: all I/O (the tick that
// drives expiry) is performed via commands returned from Update, never from
// View.
package toast

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/model"
)

// ---------------------------------------------------------------------------
// Local tick message
//
// tickMsg is intentionally a package-private type so it does not conflict
// with model.TickMsg or any other package's tick.
// ---------------------------------------------------------------------------

type tickMsg time.Time

// tickCmd returns a tea.Cmd that fires tickMsg after one second.
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// ---------------------------------------------------------------------------
// Layout constants
// ---------------------------------------------------------------------------

const (
	// toastWidth is the fixed inner content width for each toast box.
	toastWidth = 36

	// defaultDuration is the time a toast is visible before it expires.
	defaultDuration = 5 * time.Second

	// defaultMaxItems is the maximum number of toasts shown simultaneously.
	defaultMaxItems = 3
)

// ---------------------------------------------------------------------------
// ToastItem
// ---------------------------------------------------------------------------

// ToastItem represents a single transient notification.
type ToastItem struct {
	// Message is the human-readable notification text.
	Message string
	// Level determines the visual styling. Valid values: "info", "success",
	// "warning", "error".
	Level model.ToastLevel
	// CreatedAt is the wall-clock time when the toast was created.
	CreatedAt time.Time
	// Duration is how long the toast remains visible. Defaults to 5s.
	Duration time.Duration
}

// expired returns true if the toast has lived past its Duration.
func (t ToastItem) expired() bool {
	d := t.Duration
	if d <= 0 {
		d = defaultDuration
	}
	return time.Since(t.CreatedAt) >= d
}

// ---------------------------------------------------------------------------
// Lipgloss styles (package-level, constructed once)
// ---------------------------------------------------------------------------

var (
	// infoBoxStyle is used for level "info".
	infoBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(config.ColorPrimary).
			Padding(0, 1).
			Width(toastWidth)

	// successBoxStyle is used for level "success".
	successBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(config.ColorSuccess).
			Padding(0, 1).
			Width(toastWidth)

	// warningBoxStyle is used for level "warning" or "warn".
	warningBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(config.ColorWarning).
			Padding(0, 1).
			Width(toastWidth)

	// errorBoxStyle is used for level "error".
	errorBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(config.ColorError).
			Padding(0, 1).
			Width(toastWidth)

	// iconInfoStyle colors the info icon.
	iconInfoStyle = lipgloss.NewStyle().Foreground(config.ColorPrimary).Bold(true)

	// iconSuccessStyle colors the success icon.
	iconSuccessStyle = lipgloss.NewStyle().Foreground(config.ColorSuccess).Bold(true)

	// iconWarningStyle colors the warning icon.
	iconWarningStyle = lipgloss.NewStyle().Foreground(config.ColorWarning).Bold(true)

	// iconErrorStyle colors the error icon.
	iconErrorStyle = lipgloss.NewStyle().Foreground(config.ColorError).Bold(true)
)

// ---------------------------------------------------------------------------
// ToastModel
// ---------------------------------------------------------------------------

// ToastModel is the Bubbletea sub-model that manages the toast notification
// queue. It maintains an ordered list of active ToastItems, drives their
// expiry via a local tick command, and renders them as a stacked block.
//
// The zero value is not usable; use NewToastModel instead.
type ToastModel struct {
	// items is the ordered list of active toasts (oldest first).
	items []ToastItem
	// width is the current terminal width, used for position calculations.
	width int
	// height is the current terminal height, used for position calculations.
	height int
	// maxItems is the maximum number of toasts visible at once.
	maxItems int
}

// NewToastModel returns a ToastModel with maxItems=3 and sensible defaults.
func NewToastModel() ToastModel {
	return ToastModel{
		maxItems: defaultMaxItems,
	}
}

// SetSize updates the terminal dimensions used when the AppModel overlays
// this component via lipgloss.Place. Call this on every tea.WindowSizeMsg.
func (m *ToastModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Count returns the number of currently active (non-expired) toasts stored.
func (m ToastModel) Count() int {
	return len(m.items)
}

// IsEmpty returns true when there are no active toasts.
func (m ToastModel) IsEmpty() bool {
	return len(m.items) == 0
}

// SetTier satisfies the toastWidget interface.  Tier-specific rendering
// adaptations are reserved for a future ticket; this is a no-op placeholder.
func (m *ToastModel) SetTier(_ model.LayoutTier) {}

// ---------------------------------------------------------------------------
// tea.Model interface
// ---------------------------------------------------------------------------

// Init implements tea.Model. The toast component requires no startup commands.
func (m ToastModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
//
//   - model.ToastMsg: appends a new ToastItem, enforces maxItems by evicting
//     the oldest, and schedules the first tick if the queue was previously empty.
//   - tickMsg: removes any expired items and schedules the next tick if items remain.
func (m ToastModel) Update(msg tea.Msg) (ToastModel, tea.Cmd) {
	switch msg := msg.(type) {
	case model.ToastMsg:
		return m.handleToastMsg(msg)

	case tickMsg:
		return m.handleTick()
	}

	return m, nil
}

// handleToastMsg adds a new toast and returns a tick command.
func (m ToastModel) handleToastMsg(msg model.ToastMsg) (ToastModel, tea.Cmd) {
	wasEmpty := len(m.items) == 0

	item := ToastItem{
		Message:   msg.Text,
		Level:     msg.Level,
		CreatedAt: time.Now(),
		Duration:  defaultDuration,
	}

	m.items = append(m.items, item)

	// Evict oldest items to respect maxItems.
	for len(m.items) > m.maxItems {
		m.items = m.items[1:]
	}

	// Only start ticking if we weren't already ticking.
	if wasEmpty {
		return m, tickCmd()
	}
	return m, nil
}

// handleTick removes expired items and schedules the next tick when needed.
func (m ToastModel) handleTick() (ToastModel, tea.Cmd) {
	kept := m.items[:0]
	for _, item := range m.items {
		if !item.expired() {
			kept = append(kept, item)
		}
	}
	// Allocate a fresh backing array when items were removed to avoid
	// holding onto the original slice memory.
	if len(kept) < len(m.items) {
		fresh := make([]ToastItem, len(kept))
		copy(fresh, kept)
		m.items = fresh
	}

	if len(m.items) > 0 {
		return m, tickCmd()
	}
	return m, nil
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

// View implements tea.Model. It renders all active toasts as a vertically
// stacked block. The block is designed to be placed by the caller (AppModel)
// at the top-right using lipgloss.Place; this method only returns the content
// block itself.
//
// Returns an empty string when there are no active toasts.
func (m ToastModel) View() string {
	if len(m.items) == 0 {
		return ""
	}

	rendered := make([]string, 0, len(m.items))
	for _, item := range m.items {
		rendered = append(rendered, m.renderItem(item))
	}

	return strings.Join(rendered, "\n")
}

// renderItem renders a single ToastItem as a bordered box.
func (m ToastModel) renderItem(item ToastItem) string {
	icon, iconSt, boxSt := levelStyle(item.Level)

	content := fmt.Sprintf("%s %s", iconSt.Render(icon), item.Message)

	return boxSt.Render(content)
}

// HandleMsg is the pointer-receiver equivalent of Update. It mutates the
// model in place and returns only the tea.Cmd. This satisfies the
// toastWidget interface defined in the model package.
func (m *ToastModel) HandleMsg(msg tea.Msg) tea.Cmd {
	updated, cmd := m.Update(msg)
	*m = updated
	return cmd
}

// levelStyle returns the icon string, icon lipgloss.Style, and box
// lipgloss.Style for the given toast level.
func levelStyle(level model.ToastLevel) (string, lipgloss.Style, lipgloss.Style) {
	switch level {
	case "error":
		return string(config.IconError), iconErrorStyle, errorBoxStyle
	case "success":
		return string(config.IconComplete), iconSuccessStyle, successBoxStyle
	case "warning", "warn":
		return string(config.IconPaused), iconWarningStyle, warningBoxStyle
	default: // "info" and anything unrecognised
		return string(config.IconRunning), iconInfoStyle, infoBoxStyle
	}
}
