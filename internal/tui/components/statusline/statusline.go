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

// uncommittedCountMsg carries the result of `git status --porcelain | wc -l`.
type uncommittedCountMsg int

// gitBranchTickMsg is fired by the periodic git-branch refresh timer.
type gitBranchTickMsg time.Time

// authStatusTickMsg is fired by the periodic auth-status refresh timer.
type authStatusTickMsg time.Time

// sessionTimerTickMsg is fired by the 1-second session elapsed timer.
type sessionTimerTickMsg time.Time

// spinnerTickMsg is fired during streaming to animate the thinking indicator.
type spinnerTickMsg time.Time

// spinnerFrames are the Braille spinner animation frames.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

// StatusLineModel is the Bubbletea model for the bottom status bar.
// It holds data fields rendered into two rows:
//
//	Row 1: SessionCost  TokenCount  ContextPercent  PermissionMode  Elapsed  [thinking...]
//	Row 2: ActiveModel  Provider    GitBranch       AuthStatus  AgentCount  [UncommittedCount]
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

	// SessionStart is the time the session was initialized. Zero until the
	// first SystemInitEvent is received.
	SessionStart time.Time

	// Streaming is true while the assistant is generating a response.
	Streaming bool

	// UncommittedCount is the number of git uncommitted files in the working tree.
	UncommittedCount int

	// AgentCount is the number of currently active agents.
	AgentCount int

	// PlanActive is true while the assistant is operating in plan mode.
	PlanActive bool

	// PlanStep is the current step number within the plan (0 = unknown).
	PlanStep int

	// PlanTotalSteps is the total number of steps in the plan (0 = unknown).
	PlanTotalSteps int

	// theme holds the active theme for semantic coloring.
	theme config.Theme

	// spinnerIdx is the current frame index for the thinking spinner animation.
	spinnerIdx int

	// width is updated via tea.WindowSizeMsg or SetWidth.
	width int
}

// NewStatusLineModel returns a StatusLineModel with the given terminal width
// and sensible empty defaults for all data fields.
func NewStatusLineModel(width int) StatusLineModel {
	return StatusLineModel{
		width: width,
		theme: config.DefaultTheme(),
	}
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
//   - tea.WindowSizeMsg      — keeps width in sync
//   - gitBranchMsg           — updates GitBranch field
//   - authStatusMsg          — updates AuthStatus field
//   - uncommittedCountMsg    — updates UncommittedCount field
//   - gitBranchTickMsg       — fires the next background git fetch
//   - authStatusTickMsg      — fires the next background auth fetch
//   - sessionTimerTickMsg    — fires the next 1s session elapsed tick
//   - spinnerTickMsg         — advances the thinking spinner animation
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

	case uncommittedCountMsg:
		m.UncommittedCount = int(msg)

	case gitBranchTickMsg:
		return m, gitBranchCmd()

	case authStatusTickMsg:
		return m, authStatusCmd()

	case sessionTimerTickMsg:
		// Always reschedule the session timer — it runs for the lifetime of the session.
		return m, scheduleSessionTimerTick()

	case spinnerTickMsg:
		m.spinnerIdx = (m.spinnerIdx + 1) % len(spinnerFrames)
		if m.Streaming {
			// Continue animating as long as we are still streaming.
			return m, scheduleSpinnerTick()
		}
		// Streaming stopped between ticks — let the animation trail off.
	}

	return m, nil
}

// contextBarWidth is the number of fill characters inside the progress bar brackets.
const contextBarWidth = 10

// renderContextBar renders a visual progress bar for context window utilization.
//
// Wide terminal (>= 80): ctx:[=====     ] 52%
// Narrow terminal (< 80): ctx:52%
//
// The filled portion is colored using the same semantic thresholds as
// contextStyle (green/yellow/red). Empty space is rendered with StyleMuted.
func (m StatusLineModel) renderContextBar() string {
	pct := m.ContextPercent
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}

	// Narrow terminal fallback: text only, no brackets.
	if m.width < 80 {
		style := m.contextStyle(pct)
		return style.Render(fmt.Sprintf("ctx:%.0f%%", pct))
	}

	// Calculate number of filled characters.
	fillCount := int(pct / 100 * float64(contextBarWidth))
	if fillCount > contextBarWidth {
		fillCount = contextBarWidth
	}

	filled := strings.Repeat("=", fillCount)
	empty := strings.Repeat(" ", contextBarWidth-fillCount)

	// Color the filled portion based on threshold; mute the empty portion.
	style := m.contextStyle(pct)
	coloredFill := style.Render(filled)
	mutedEmpty := config.StyleMuted.Render(empty)

	return fmt.Sprintf("ctx:[%s%s] %.0f%%", coloredFill, mutedEmpty, pct)
}

// View implements tea.Model. It renders the status bar as two rows:
//
//	Row 1: $cost  tokens  ctx:[bar] %  perm-mode  elapsed  [thinking...]
//	Row 2: model  provider  branch  auth  agents:N  [uncommitted:N]
//
// Cost, context, permission and auth fields use semantic colors from the
// active theme (green/yellow/red based on thresholds). The two rows are
// joined vertically.
func (m StatusLineModel) View() string {
	label := config.StyleMuted.Render
	value := config.StyleStatusBar.Render

	// Row 1: financial / quota fields
	costField := lipgloss.JoinHorizontal(lipgloss.Top,
		label("cost:"),
		m.costStyle(m.SessionCost).Render(state.FormatCost(m.SessionCost)),
	)
	tokenField := lipgloss.JoinHorizontal(lipgloss.Top,
		label(" tokens:"),
		value(formatTokens(m.TokenCount)),
	)
	// Context field uses a visual progress bar (falls back to text on narrow terminals).
	ctxField := lipgloss.JoinHorizontal(lipgloss.Top,
		label(" "),
		m.renderContextBar(),
	)
	permField := lipgloss.JoinHorizontal(lipgloss.Top,
		label(" perm:"),
		m.permStyle(m.PermissionMode).Render(m.PermissionMode),
	)

	// Elapsed time since session started.
	elapsedField := ""
	if !m.SessionStart.IsZero() {
		elapsed := time.Since(m.SessionStart)
		mins := int(elapsed.Minutes())
		secs := int(elapsed.Seconds()) % 60
		elapsedField = lipgloss.JoinHorizontal(lipgloss.Top,
			label(" ⏱"),
			value(fmt.Sprintf("%dm%ds", mins, secs)),
		)
	}

	// Streaming / thinking indicator uses InfoStyle (cyan).
	thinkingField := ""
	if m.Streaming {
		frame := spinnerFrames[m.spinnerIdx%len(spinnerFrames)]
		thinkingField = lipgloss.JoinHorizontal(lipgloss.Top,
			label(" "),
			m.theme.InfoStyle().Render(frame+" thinking..."),
		)
	}

	// Plan mode indicator: shown before cost when plan mode is active.
	planField := ""
	if m.PlanActive {
		var planText string
		if m.PlanTotalSteps > 0 {
			planText = fmt.Sprintf("[PLAN MODE: step %d/%d]", m.PlanStep, m.PlanTotalSteps)
		} else {
			planText = "[PLAN MODE]"
		}
		planField = m.theme.WarningStyle().Render(planText)
	}

	row1Parts := []string{costField, tokenField, ctxField, permField}
	if planField != "" {
		row1Parts = append([]string{planField, " "}, row1Parts...)
	}
	if elapsedField != "" {
		row1Parts = append(row1Parts, elapsedField)
	}
	if thinkingField != "" {
		row1Parts = append(row1Parts, thinkingField)
	}

	row1 := lipgloss.NewStyle().
		Width(m.width).
		Render(lipgloss.JoinHorizontal(lipgloss.Top, row1Parts...))

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

	// Auth field: green for authenticated, red for "N/A".
	var authValue string
	if m.AuthStatus == "N/A" || m.AuthStatus == "" {
		authValue = m.theme.ErrorStyle().Render(m.AuthStatus)
	} else {
		authValue = m.theme.SuccessStyle().Render(m.AuthStatus)
	}
	authField := lipgloss.JoinHorizontal(lipgloss.Top,
		label(" auth:"),
		authValue,
	)

	// Agent count field.
	agentField := lipgloss.JoinHorizontal(lipgloss.Top,
		label(" agents:"),
		value(fmt.Sprintf("%d", m.AgentCount)),
	)

	row2Parts := []string{modelField, providerField, branchField, authField, agentField}

	// Uncommitted count field: only shown when > 0, styled with warning color.
	if m.UncommittedCount > 0 {
		uncommittedField := lipgloss.JoinHorizontal(lipgloss.Top,
			label(" uncommitted:"),
			m.theme.WarningStyle().Render(fmt.Sprintf("%d", m.UncommittedCount)),
		)
		row2Parts = append(row2Parts, uncommittedField)
	}

	row2 := lipgloss.NewStyle().
		Width(m.width).
		Render(lipgloss.JoinHorizontal(lipgloss.Top, row2Parts...))

	return lipgloss.JoinVertical(lipgloss.Left, row1, row2)
}

// ---------------------------------------------------------------------------
// Public helpers
// ---------------------------------------------------------------------------

// SetWidth updates the status line width for responsive resizing.
func (m *StatusLineModel) SetWidth(w int) {
	m.width = w
}

// SetTheme updates the active theme used for semantic coloring.
func (m *StatusLineModel) SetTheme(t config.Theme) {
	m.theme = t
}

// SetStreaming sets the Streaming flag and, when transitioning to true,
// returns a command to start the spinner animation. When transitioning to
// false, returns nil — the spinner halts naturally on the next spinnerTickMsg.
func (m *StatusLineModel) SetStreaming(v bool) tea.Cmd {
	m.Streaming = v
	if v {
		m.spinnerIdx = 0
		return scheduleSpinnerTick()
	}
	return nil
}

// StartTicks returns the initial commands to fetch git branch and auth status
// immediately, plus the periodic tick schedulers. Call from AppModel.Init()
// or after the CLI driver is ready.
func (m StatusLineModel) StartTicks() tea.Cmd {
	return tea.Batch(
		gitBranchCmd(),
		authStatusCmd(),
		uncommittedCountCmd(),
		scheduleGitBranchTick(),
		scheduleAuthStatusTick(),
		scheduleSessionTimerTick(),
	)
}

// ---------------------------------------------------------------------------
// Semantic color helpers
// ---------------------------------------------------------------------------

// costStyle returns a lipgloss.Style based on session cost thresholds:
//   - >= $1.00  → ErrorStyle  (red)
//   - >= $0.10  → WarningStyle (yellow)
//   - < $0.10   → SuccessStyle (green)
func (m StatusLineModel) costStyle(cost float64) lipgloss.Style {
	switch {
	case cost >= 1.00:
		return m.theme.ErrorStyle()
	case cost >= 0.10:
		return m.theme.WarningStyle()
	default:
		return m.theme.SuccessStyle()
	}
}

// contextStyle returns a lipgloss.Style based on context window usage thresholds:
//   - >= 80%  → ErrorStyle   (red)
//   - >= 50%  → WarningStyle (yellow)
//   - < 50%   → SuccessStyle (green)
func (m StatusLineModel) contextStyle(pct float64) lipgloss.Style {
	switch {
	case pct >= 80:
		return m.theme.ErrorStyle()
	case pct >= 50:
		return m.theme.WarningStyle()
	default:
		return m.theme.SuccessStyle()
	}
}

// permStyle returns a lipgloss.Style based on the permission mode:
//   - "allow-all" → ErrorStyle   (red)
//   - "plan"      → WarningStyle (yellow)
//   - other       → SuccessStyle (green)
func (m StatusLineModel) permStyle(mode string) lipgloss.Style {
	switch mode {
	case "allow-all":
		return m.theme.ErrorStyle()
	case "plan":
		return m.theme.WarningStyle()
	default:
		return m.theme.SuccessStyle()
	}
}

// ---------------------------------------------------------------------------
// Token formatting
// ---------------------------------------------------------------------------

// formatTokens returns a compact human-readable representation of a token count:
//   - 0       → "0"
//   - 1500    → "1.5K"
//   - 150000  → "150K"
//   - 1500000 → "1.5M"
func formatTokens(n int) string {
	switch {
	case n == 0:
		return "0"
	case n < 1000:
		return fmt.Sprintf("%d", n)
	case n < 1_000_000:
		k := float64(n) / 1000.0
		if k == float64(int(k)) {
			return fmt.Sprintf("%dK", int(k))
		}
		return fmt.Sprintf("%.1fK", k)
	default:
		m := float64(n) / 1_000_000.0
		if m == float64(int(m)) {
			return fmt.Sprintf("%dM", int(m))
		}
		return fmt.Sprintf("%.1fM", m)
	}
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

// uncommittedCountCmd runs `git status --porcelain | wc -l` in a goroutine
// and returns the line count as an uncommittedCountMsg. If git is not found
// or the command fails, 0 is returned.
func uncommittedCountCmd() tea.Cmd {
	return func() tea.Msg {
		if !binaryExists("git") {
			return uncommittedCountMsg(0)
		}
		// Run git status --porcelain and count non-empty lines.
		out, err := exec.Command("git", "status", "--porcelain").Output()
		if err != nil {
			return uncommittedCountMsg(0)
		}
		count := 0
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if strings.TrimSpace(line) != "" {
				count++
			}
		}
		return uncommittedCountMsg(count)
	}
}

// authStatusCmd runs `claude auth status` in a goroutine and extracts a short
// status string from the output. It parses for email and login method.
// If the claude binary is not found or the command fails, Status is "N/A".
func authStatusCmd() tea.Cmd {
	return func() tea.Msg {
		if !binaryExists("claude") {
			return authStatusMsg{Status: "N/A", Err: fmt.Errorf("claude not found")}
		}
		out, err := exec.Command("claude", "auth", "status").Output()
		if err != nil {
			return authStatusMsg{Status: "N/A", Err: err}
		}
		status := parseAuthStatus(strings.TrimSpace(string(out)))
		return authStatusMsg{Status: status}
	}
}

// parseAuthStatus extracts a compact auth description from the raw output of
// `claude auth status`. It looks for an email address (contains "@") and a
// login method ("Logged in via ..."). The result is formatted as
// "method • email" or just the email/method if only one is found. Falls back
// to the first non-empty line truncated to 50 chars.
func parseAuthStatus(raw string) string {
	if raw == "" {
		return "N/A"
	}

	var email, method string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Extract email: any line with "@" that doesn't look like a URL.
		if email == "" && strings.Contains(line, "@") && !strings.HasPrefix(line, "http") {
			// Remove label prefixes like "Account: " or "Email: ".
			if idx := strings.LastIndex(line, " "); idx >= 0 {
				candidate := line[idx+1:]
				if strings.Contains(candidate, "@") {
					email = candidate
					continue
				}
			}
			email = line
		}
		// Extract login method.
		lower := strings.ToLower(line)
		if method == "" && (strings.Contains(lower, "logged in") || strings.Contains(lower, "login method")) {
			// Try to extract the method from "Logged in via X" or "Login method: X".
			for _, sep := range []string{" via ", ": "} {
				if idx := strings.Index(lower, sep); idx >= 0 {
					method = strings.TrimSpace(line[idx+len(sep):])
					break
				}
			}
			if method == "" {
				method = "claude.ai"
			}
		}
	}

	switch {
	case email != "" && method != "":
		result := method + " • " + email
		if len(result) > 50 {
			result = result[:50] + "..."
		}
		return result
	case email != "":
		if len(email) > 50 {
			email = email[:50] + "..."
		}
		return email
	case method != "":
		if len(method) > 50 {
			method = method[:50] + "..."
		}
		return method
	default:
		// Fall back to first non-empty line.
		for _, line := range strings.Split(raw, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				if len(line) > 50 {
					return line[:50] + "..."
				}
				return line
			}
		}
		return "N/A"
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

// scheduleSessionTimerTick returns a command that fires after 1 second to
// refresh the elapsed session time display.
func scheduleSessionTimerTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return sessionTimerTickMsg(t)
	})
}

// scheduleSpinnerTick returns a command that fires after 100ms to advance the
// thinking spinner animation.
func scheduleSpinnerTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return spinnerTickMsg(t)
	})
}
