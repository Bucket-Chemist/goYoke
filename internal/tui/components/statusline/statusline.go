// Package statusline implements the two-row status-bar component for the
// GOgent-Fortress TUI. It surfaces session cost, token usage, context
// percentage, permission mode, active model, provider, git branch, and
// authentication status across two compact rows at the bottom of the screen.
package statusline

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
)

// ---------------------------------------------------------------------------
// Message types
// ---------------------------------------------------------------------------

// gitBranchMsg carries the result of `git rev-parse --abbrev-ref HEAD`.
type gitBranchMsg struct {
	Branch string
	Err    error
}

// authStatusMsg carries the result of `claude auth status`.
type authStatusMsg struct {
	Status string
	Err    error
}

// gitBranchTickMsg is fired by the periodic git-branch refresh timer.
type gitBranchTickMsg time.Time

// authStatusTickMsg is fired by the periodic auth-status refresh timer.
type authStatusTickMsg time.Time

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// tea.Model interface
// ---------------------------------------------------------------------------

// Init implements tea.Model. The status line requires no startup commands.
func (m StatusLineModel) Init() tea.Cmd {
	return nil
}

// Update handles status-line messages. It returns a typed StatusLineModel
// (not tea.Model) so the parent can use it directly without a type assertion.
//
// Handled messages:
//   - tea.WindowSizeMsg   — keeps width in sync
//   - gitBranchMsg        — updates GitBranch field
//   - authStatusMsg       — updates AuthStatus field
//   - gitBranchTickMsg    — fires the next background git fetch
//   - authStatusTickMsg   — fires the next background auth fetch
func (m StatusLineModel) Update(msg tea.Msg) (StatusLineModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case gitBranchMsg:
		if msg.Err == nil {
			m.GitBranch = msg.Branch
		} else {
			m.GitBranch = "N/A"
		}

	case authStatusMsg:
		// msg always pre-fills Status with "N/A" on error.
		m.AuthStatus = msg.Status

	case gitBranchTickMsg:
		return m, gitBranchCmd()

	case authStatusTickMsg:
		return m, authStatusCmd()
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
		value(state.FormatCost(m.SessionCost)),
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

// ---------------------------------------------------------------------------
// Public helpers
// ---------------------------------------------------------------------------

// SetWidth updates the status line width for responsive resizing.
func (m *StatusLineModel) SetWidth(w int) {
	m.width = w
}

// StartTicks returns the initial commands to fetch git branch and auth status
// immediately, plus the periodic tick schedulers. Call from AppModel.Init()
// or after the CLI driver is ready.
func (m StatusLineModel) StartTicks() tea.Cmd {
	return tea.Batch(
		gitBranchCmd(),
		authStatusCmd(),
		scheduleGitBranchTick(),
		scheduleAuthStatusTick(),
	)
}

// ---------------------------------------------------------------------------
// Background commands (never block Update or View)
// ---------------------------------------------------------------------------

// binaryExists reports whether the named executable is available on PATH.
func binaryExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// gitBranchCmd runs `git rev-parse --abbrev-ref HEAD` in a goroutine and
// returns the result as a gitBranchMsg. If the git binary is not found or the
// command fails, the Err field is set and GitBranch will be "N/A".
func gitBranchCmd() tea.Cmd {
	return func() tea.Msg {
		if !binaryExists("git") {
			return gitBranchMsg{Err: fmt.Errorf("git not found")}
		}
		out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
		if err != nil {
			return gitBranchMsg{Err: err}
		}
		return gitBranchMsg{Branch: strings.TrimSpace(string(out))}
	}
}

// authStatusCmd runs `claude auth status` in a goroutine and extracts a short
// status string from the output. If the claude binary is not found or the
// command fails, Status is set to "N/A".
func authStatusCmd() tea.Cmd {
	return func() tea.Msg {
		if !binaryExists("claude") {
			return authStatusMsg{Status: "N/A", Err: fmt.Errorf("claude not found")}
		}
		out, err := exec.Command("claude", "auth", "status").Output()
		if err != nil {
			return authStatusMsg{Status: "N/A", Err: err}
		}
		status := strings.TrimSpace(string(out))
		if len(status) > 30 {
			status = status[:30] + "..."
		}
		return authStatusMsg{Status: status}
	}
}

// scheduleGitBranchTick returns a command that fires after 30 seconds,
// triggering the next background git-branch fetch.
func scheduleGitBranchTick() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return gitBranchTickMsg(t)
	})
}

// scheduleAuthStatusTick returns a command that fires after 60 seconds,
// triggering the next background auth-status fetch.
func scheduleAuthStatusTick() tea.Cmd {
	return tea.Tick(60*time.Second, func(t time.Time) tea.Msg {
		return authStatusTickMsg(t)
	})
}
