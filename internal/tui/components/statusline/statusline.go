// Package statusline implements the two-row status-bar component for the
// GOgent-Fortress TUI. It surfaces session cost, token usage, context
// percentage, permission mode, active model, provider, git branch, and
// authentication status across two compact rows at the bottom of the screen.
package statusline

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
)

// StatusLineModel is the Bubbletea model for the bottom status bar.
// It holds eight data fields that are rendered into two rows:
//
//	Row 1: SessionCost  TokenCount  ContextPercent  PermissionMode
//	Row 2: ActiveModel  Provider    GitBranch       AuthStatus
//
// The zero value is not usable; use NewStatusLineModel instead.
type StatusLineModel struct {
	// SessionCost is the cumulative cost of the current session in US dollars.
	SessionCost float64

	// TokenCount is the total number of tokens consumed in the session.
	TokenCount int

	// ContextPercent is the percentage of the context window currently used.
	ContextPercent float64

	// PermissionMode is the current permission escalation mode label.
	PermissionMode string

	// ActiveModel is the name of the LLM model currently in use.
	ActiveModel string

	// Provider is the name of the LLM provider currently in use.
	Provider string

	// GitBranch is the name of the active git branch, if available.
	GitBranch string

	// AuthStatus is a short human-readable authentication status string.
	AuthStatus string

	// width is updated via tea.WindowSizeMsg or SetWidth.
	width int
}

// NewStatusLineModel returns a StatusLineModel with the given terminal width
// and sensible empty defaults for all data fields.
func NewStatusLineModel(width int) StatusLineModel {
	return StatusLineModel{width: width}
}

// Init implements tea.Model. The status line requires no startup commands.
func (m StatusLineModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model. It handles tea.WindowSizeMsg to keep the
// status line width in sync with the terminal size.
func (m StatusLineModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
	}
	return m, nil
}

// View implements tea.Model. It renders the status bar as two rows:
//
//	Row 1: $cost  tokens  ctx%  perm-mode
//	Row 2: model  provider  branch  auth
//
// Each field is labelled and styled with config.StyleMuted for labels and
// config.StyleStatusBar for values. The two rows are joined vertically.
func (m StatusLineModel) View() string {
	label := config.StyleMuted.Render
	value := config.StyleStatusBar.Render

	// Row 1: financial / quota fields
	costField := lipgloss.JoinHorizontal(lipgloss.Top,
		label("cost:"),
		value(fmt.Sprintf("$%.4f", m.SessionCost)),
	)
	tokenField := lipgloss.JoinHorizontal(lipgloss.Top,
		label(" tokens:"),
		value(fmt.Sprintf("%d", m.TokenCount)),
	)
	ctxField := lipgloss.JoinHorizontal(lipgloss.Top,
		label(" ctx:"),
		value(fmt.Sprintf("%.1f%%", m.ContextPercent)),
	)
	permField := lipgloss.JoinHorizontal(lipgloss.Top,
		label(" perm:"),
		value(m.PermissionMode),
	)

	row1 := lipgloss.NewStyle().
		Width(m.width).
		Render(lipgloss.JoinHorizontal(lipgloss.Top,
			costField, tokenField, ctxField, permField,
		))

	// Row 2: identity / environment fields
	modelField := lipgloss.JoinHorizontal(lipgloss.Top,
		label("model:"),
		value(m.ActiveModel),
	)
	providerField := lipgloss.JoinHorizontal(lipgloss.Top,
		label(" provider:"),
		value(m.Provider),
	)
	branchField := lipgloss.JoinHorizontal(lipgloss.Top,
		label(" branch:"),
		value(m.GitBranch),
	)
	authField := lipgloss.JoinHorizontal(lipgloss.Top,
		label(" auth:"),
		value(m.AuthStatus),
	)

	row2 := lipgloss.NewStyle().
		Width(m.width).
		Render(lipgloss.JoinHorizontal(lipgloss.Top,
			modelField, providerField, branchField, authField,
		))

	return lipgloss.JoinVertical(lipgloss.Left, row1, row2)
}

// SetWidth updates the status line width for responsive resizing.
func (m *StatusLineModel) SetWidth(w int) {
	m.width = w
}
