// Package statusline implements the adaptive status-bar component for the
// GOgent-Fortress TUI. It surfaces session cost, token usage, context
// percentage, permission mode, active model, provider, git branch, and
// authentication status. At Standard width (80-119 cols) it renders one row;
// at Wide+ (120+) it renders two rows.
package statusline

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/util"
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

// CostFlashExpiredMsg is fired 500ms after a cost flash starts, signalling the
// bright-white highlight should be cleared.
type CostFlashExpiredMsg struct{}

// spinnerFrames are the Braille spinner animation frames.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

// StatusLineModel is the Bubbletea model for the bottom status bar.
// At Wide+ (width >= 120) it renders two rows:
//
//	Row 1: [model] [perm] 📁 project | 🌿 branch     ████░░ 45% 234K/1M | agents:N · auth
//	Row 2: $0.45                                       ⏱ 5m 12s | ↻ streaming
//
// At Standard/Compact (width < 120) it renders one row with critical fields only:
//
//	Row 1: $0.45 [model]        agents:N/T ██░░ 45% | ⏱ 5m12s
//
// The zero value is not usable; use NewStatusLineModel instead.
type StatusLineModel struct {
	// SessionCost is the cumulative cost of the current session in US dollars.
	SessionCost float64

	// TokenCount is the total number of tokens consumed in the session.
	TokenCount int

	// ContextPercent is the percentage of the context window currently used.
	ContextPercent float64

	// ContextUsedTokens is the number of tokens currently consumed in the context window.
	ContextUsedTokens int

	// ContextCapacity is the total context window size in tokens for the active model.
	ContextCapacity int

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

	// AgentStats holds per-status agent counts for the sparkline display.
	AgentStats state.AgentStats

	// PlanActive is true while the assistant is operating in plan mode.
	PlanActive bool

	// PlanStep is the current step number within the plan (0 = unknown).
	PlanStep int

	// PlanTotalSteps is the total number of steps in the plan (0 = unknown).
	PlanTotalSteps int

	// CWD is the current working directory, set via the CWD selector modal.
	// Empty string means the default (process CWD). Displayed in Row 1 with
	// safety coloring: green (project), yellow (home), red (root).
	CWD string

	// VimEnabled is true when the vim keybinding overlay is active (TUI-062).
	// When true, VimMode is rendered in the status bar.
	VimEnabled bool

	// VimMode is the current vim input mode ("NORMAL" or "INSERT").
	// Only displayed when VimEnabled is true.
	VimMode string

	// MouseEnabled is true when mouse capture is active (scroll wheel works).
	// When false, native terminal text selection is available. Always rendered.
	MouseEnabled bool

	// TeamActive is true when a team is currently running.
	TeamActive bool

	// TeamName is the name of the running team (truncated for display).
	TeamName string

	// TeamMemberStatuses is a slice of member status strings for dot rendering.
	// Each entry is one of: "running", "complete", "pending", "failed", "error", "skipped", "killed".
	TeamMemberStatuses []string

	// TeamCurrentWave is the 1-based current wave number.
	TeamCurrentWave int

	// TeamTotalWaves is the total number of waves.
	TeamTotalWaves int

	// TeamCost is the cumulative cost of the team in USD.
	TeamCost float64

	// ReduceMotion disables the spinner animation when true (WCAG 2.3.1).
	// When true, streaming is shown as a static "⠿ streaming" indicator instead
	// of the animated Braille spinner. Set via the Settings → Display panel.
	ReduceMotion bool

	// CostFlashEnabled enables the cost flash-on-increase animation (opt-in).
	// When false (default), the cost badge never flashes regardless of other settings.
	CostFlashEnabled bool

	// costFlashUntil is the expiry time for the current cost flash. Zero means no
	// flash is active. Set by CheckCostFlash, cleared by CostFlashExpiredMsg.
	costFlashUntil time.Time

	// prevCost is the SessionCost value from the previous CheckCostFlash call,
	// used to detect increases.
	prevCost float64

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
		if m.ReduceMotion {
			// Reduce motion: do not advance the frame index and do not
			// reschedule — the static indicator in View() needs no ticking.
			return m, nil
		}
		m.spinnerIdx = (m.spinnerIdx + 1) % len(spinnerFrames)
		if m.Streaming {
			// Continue animating as long as we are still streaming.
			return m, scheduleSpinnerTick()
		}
		// Streaming stopped between ticks — let the animation trail off.

	case CostFlashExpiredMsg:
		m.costFlashUntil = time.Time{}
	}

	return m, nil
}

// contextBarWidth is the number of fill characters inside the progress bar brackets.
const contextBarWidth = 10

// renderContextBar renders a visual progress bar for context window utilization.
//
// Wide terminal (>= 100): █████░░░░░ 45% 234K/1M
// Medium terminal (80-99): █████░░░░░ 45% 234K/1M  (shorter bar)
// Narrow terminal (< 80):  45% 234K/1M
//
// The filled portion is colored using semantic thresholds (green/yellow/red).
// Empty space is rendered with StyleMuted. Exact token counts always shown.
func (m StatusLineModel) renderContextBar() string {
	pct := m.ContextPercent
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}

	// Format token counts: "234K/1M" or "—" if no data yet.
	var tokenLabel string
	if m.ContextCapacity > 0 {
		tokenLabel = formatTokens(m.ContextUsedTokens) + "/" + formatTokens(m.ContextCapacity)
	}

	style := m.contextStyle(pct)

	// Narrow terminal fallback: text only, no bar.
	if m.width < 80 {
		if tokenLabel != "" {
			return style.Render(fmt.Sprintf("%.0f%%", pct)) + config.StyleMuted.Render(" "+tokenLabel)
		}
		return config.StyleMuted.Render("ctx:—")
	}

	// No data yet — show empty bar with dash.
	if m.ContextCapacity == 0 {
		emptyBar := config.StyleMuted.Render(strings.Repeat("░", contextBarWidth))
		return emptyBar + config.StyleMuted.Render(" —")
	}

	// Calculate fill from percentage.
	fillCount := int(pct / 100 * float64(contextBarWidth))
	if fillCount > contextBarWidth {
		fillCount = contextBarWidth
	}

	filled := strings.Repeat("█", fillCount)
	empty := strings.Repeat("░", contextBarWidth-fillCount)

	coloredFill := style.Render(filled)
	mutedEmpty := config.StyleMuted.Render(empty)
	pctStr := style.Render(fmt.Sprintf("%.0f%%", pct))

	return coloredFill + mutedEmpty + " " + pctStr + config.StyleMuted.Render(" "+tokenLabel)
}

// renderTeamIndicator renders the compact team status indicator for Row 1.
// Returns an empty string when no team is active (TeamActive == false).
//
// Format: "⚡{name} {dots} {wave}/{total} {cost}"
//   - name: team name truncated to 8 runes
//   - dots: one dot per member colored by status
//   - wave/total: e.g. "2/4"
//   - cost: formatted as "$X.XX"
//
// Dot coloring per status:
//
//	"running"         → SuccessStyle (green) ●
//	"complete"        → SuccessStyle + Bold (bright green) ●
//	"pending"         → StyleMuted (grey) ○
//	"failed"/"error"  → ErrorStyle (red) ●
//	"skipped"/"killed"→ WarningStyle (yellow) ●
func (m StatusLineModel) renderTeamIndicator() string {
	if !m.TeamActive {
		return ""
	}

	// Prefix with team name (truncated to 8 runes).
	name := util.Truncate(m.TeamName, 8)
	prefix := m.theme.InfoStyle().Render("⚡") + config.StyleStatusBar.Render(name)

	// One dot per member, colored by status.
	var dotsBuilder strings.Builder
	for _, status := range m.TeamMemberStatuses {
		var dot string
		switch status {
		case "running":
			dot = m.theme.SuccessStyle().Render("●")
		case "complete":
			dot = m.theme.SuccessStyle().Bold(true).Render("●")
		case "pending":
			dot = config.StyleMuted.Render("○")
		case "failed", "error":
			dot = m.theme.ErrorStyle().Render("●")
		case "skipped", "killed":
			dot = m.theme.WarningStyle().Render("●")
		default:
			dot = config.StyleMuted.Render("○")
		}
		dotsBuilder.WriteString(dot)
	}
	dots := dotsBuilder.String()

	// Wave progress: "2/4"
	waveStr := config.StyleStatusBar.Render(fmt.Sprintf("%d/%d", m.TeamCurrentWave, m.TeamTotalWaves))

	// Team cost using the same threshold coloring as session cost.
	costStr := m.costStyle(m.TeamCost).Render(state.FormatCost(m.TeamCost))

	return prefix + " " + dots + " " + waveStr + " " + costStr
}

// renderAgentSparkline renders the agent count with per-status sparkline dots.
// Format: "agents: {running}/{total} {dots}"
// Dot colors match the agent tree status colors:
//
//	running  → SuccessStyle (green) ●
//	complete → SuccessStyle + Bold (bright green) ●
//	pending  → StyleMuted (grey) ○
//	error    → ErrorStyle (red) ●
//	killed   → WarningStyle (yellow) ●
//
// Returns an empty string when no agents exist (Total == 0).
func (m StatusLineModel) renderAgentSparkline() string {
	stats := m.AgentStats
	if stats.Total == 0 {
		return ""
	}

	prefix := config.StyleMuted.Render(fmt.Sprintf("agents: %d/%d ", stats.Running, stats.Total))

	var dots strings.Builder
	for range stats.Running {
		dots.WriteString(m.theme.SuccessStyle().Render("●"))
	}
	for range stats.Pending {
		dots.WriteString(config.StyleMuted.Render("○"))
	}
	for range stats.Complete {
		dots.WriteString(m.theme.SuccessStyle().Bold(true).Render("●"))
	}
	for range stats.Error {
		dots.WriteString(m.theme.ErrorStyle().Render("●"))
	}
	for range stats.Killed {
		dots.WriteString(m.theme.WarningStyle().Render("●"))
	}

	return prefix + dots.String()
}

// Height returns the number of rows the status line occupies at the current
// width. Wide+ (>= 120 cols) uses 2 rows; Standard/Compact (< 120) uses 1 row.
// This must agree with lipgloss.Height(View()) at all widths.
func (m StatusLineModel) Height() int {
	if m.width >= 120 {
		return 2
	}
	return 1
}

// View implements tea.Model. It dispatches to viewFull (2 rows, width >= 120)
// or viewCompact (1 row, width < 120) based on the current terminal width.
func (m StatusLineModel) View() string {
	if m.width >= 120 {
		return m.viewFull()
	}
	return m.viewCompact()
}

// viewFull renders the two-row status bar used at Wide+ (width >= 120):
//
//	Row 1: [model] [perm] 📁 project | 🌿 branch     ████░░ 41% 234K/1M | auth · email
//	Row 2: $0.45                                       ⏱ 5m 12s | ↻ streaming
//
// Cost, context, permission and auth fields use semantic colors.
func (m StatusLineModel) viewFull() string {
	muted := config.StyleMuted.Render

	// ===== ROW 1: identity line =====

	// Model badge: [claude-opus-4-6[1m]]
	modelBadge := m.theme.InfoStyle().Render("[" + m.ActiveModel + "]")

	// Permission badge: [acceptEdits]
	permLabel := m.PermissionMode
	if permLabel == "" {
		permLabel = "default"
	}
	permBadge := m.permStyle(permLabel).Render("[" + permLabel + "]")

	// Vim mode badge (optional)
	vimBadge := ""
	if m.VimEnabled {
		mode := m.VimMode
		if mode == "" {
			mode = "NORMAL"
		}
		var vimStyle lipgloss.Style
		if mode == "INSERT" {
			vimStyle = m.theme.InfoStyle()
		} else {
			vimStyle = config.StyleMuted
		}
		vimBadge = vimStyle.Render("["+mode+"]") + " "
	}

	// Mouse mode badge: muted [M] when enabled (default), warning [T] when disabled (text select).
	var mouseBadge string
	if m.MouseEnabled {
		mouseBadge = config.StyleMuted.Render("[M]") + " "
	} else {
		mouseBadge = m.theme.WarningStyle().Render("[T]") + " "
	}

	// Plan mode badge (optional)
	planBadge := ""
	if m.PlanActive {
		var planText string
		if m.PlanTotalSteps > 0 {
			planText = fmt.Sprintf("[PLAN %d/%d]", m.PlanStep, m.PlanTotalSteps)
		} else {
			planText = "[PLAN]"
		}
		planBadge = m.theme.WarningStyle().Render(planText) + " "
	}

	// Project name: extract from working directory or use provider name
	projectName := m.Provider
	if projectName == "" {
		projectName = "—"
	}

	// CWD scope indicator (safety coloring)
	cwdField := ""
	if m.CWD != "" {
		cwdLabel := shortenCWD(m.CWD)
		var cwdStyle lipgloss.Style
		switch m.CWD {
		case "/":
			cwdStyle = m.theme.ErrorStyle()
		default:
			home, _ := os.UserHomeDir()
			if m.CWD == home {
				cwdStyle = m.theme.WarningStyle()
			} else {
				cwdStyle = m.theme.SuccessStyle()
			}
		}
		cwdField = muted(" 📂 ") + cwdStyle.Render(cwdLabel)
	}

	// Git branch
	branchField := ""
	if m.GitBranch != "" && m.GitBranch != "N/A" {
		branchField = muted(" | 🌿 ") + config.StyleStatusBar.Render(m.GitBranch)
		if m.UncommittedCount > 0 {
			branchField += m.theme.WarningStyle().Render(fmt.Sprintf(" ~%d", m.UncommittedCount))
		}
	}

	costBadge := m.activeCostStyle().Render(state.FormatCost(m.SessionCost))
	row1Left := costBadge + " " + vimBadge + planBadge + mouseBadge + modelBadge + " " + permBadge +
		muted(" 📁 ") + config.StyleStatusBar.Render(projectName) + cwdField + branchField

	// Team indicator (optional — only when a team is running)
	teamInd := m.renderTeamIndicator()
	if teamInd != "" {
		row1Left = row1Left + muted(" | ") + teamInd
	}

	// Auth: right-aligned
	var authValue string
	if m.AuthStatus == "N/A" || m.AuthStatus == "" {
		authValue = m.theme.ErrorStyle().Render("N/A")
	} else {
		authValue = m.theme.SuccessStyle().Render(m.AuthStatus)
	}
	// Context bar: positioned prominently on the right side of Row 1.
	ctxBar := m.renderContextBar()
	row1Right := ctxBar + muted(" | ") + authValue

	// Agent sparkline (only if agents exist)
	if agentSpark := m.renderAgentSparkline(); agentSpark != "" {
		row1Right = agentSpark + muted(" · ") + row1Right
	}

	row1 := m.joinLeftRight(row1Left, row1Right)

	// ===== ROW 2: metrics line =====

	row2Left := ""

	// Elapsed + streaming status: right-aligned
	var row2RightParts []string

	if !m.SessionStart.IsZero() {
		elapsed := time.Since(m.SessionStart)
		mins := int(elapsed.Minutes())
		secs := int(elapsed.Seconds()) % 60
		row2RightParts = append(row2RightParts,
			muted("⏱ ")+config.StyleStatusBar.Render(fmt.Sprintf("%dm %ds", mins, secs)))
	}

	if m.Streaming {
		if m.ReduceMotion {
			row2RightParts = append(row2RightParts,
				m.theme.InfoStyle().Render("⠿ streaming"))
		} else {
			frame := spinnerFrames[m.spinnerIdx%len(spinnerFrames)]
			row2RightParts = append(row2RightParts,
				m.theme.InfoStyle().Render(frame+" streaming"))
		}
	}

	row2Right := strings.Join(row2RightParts, muted(" | "))
	row2 := m.joinLeftRight(row2Left, row2Right)

	return lipgloss.JoinVertical(lipgloss.Left, row1, row2)
}

// viewCompact renders the single-row status bar used at Standard/Compact
// (width < 120). Only critical fields are shown to fit in limited space:
// cost, plan badge, model, agent sparkline, context bar, elapsed time,
// and streaming indicator. Permission mode, auth, git branch, CWD, vim/mouse
// badges are omitted.
func (m StatusLineModel) viewCompact() string {
	muted := config.StyleMuted.Render

	// Cost badge.
	costBadge := m.activeCostStyle().Render(state.FormatCost(m.SessionCost))

	// Plan badge (only when active).
	planBadge := ""
	if m.PlanActive {
		if m.PlanTotalSteps > 0 {
			planBadge = m.theme.WarningStyle().Render(fmt.Sprintf("[P%d/%d]", m.PlanStep, m.PlanTotalSteps)) + " "
		} else {
			planBadge = m.theme.WarningStyle().Render("[PLAN]") + " "
		}
	}

	// Model badge (short).
	modelBadge := m.theme.InfoStyle().Render("[" + m.ActiveModel + "]")

	left := costBadge + " " + planBadge + modelBadge

	// Right: agent sparkline · context bar · elapsed · streaming frame.
	var rightParts []string

	if agentSpark := m.renderAgentSparkline(); agentSpark != "" {
		rightParts = append(rightParts, agentSpark)
	}

	rightParts = append(rightParts, m.renderContextBar())

	if !m.SessionStart.IsZero() {
		elapsed := time.Since(m.SessionStart)
		mins := int(elapsed.Minutes())
		secs := int(elapsed.Seconds()) % 60
		rightParts = append(rightParts,
			muted("⏱ ")+config.StyleStatusBar.Render(fmt.Sprintf("%dm%ds", mins, secs)))
	}

	if m.Streaming {
		if m.ReduceMotion {
			rightParts = append(rightParts, m.theme.InfoStyle().Render("⠿ streaming"))
		} else {
			frame := spinnerFrames[m.spinnerIdx%len(spinnerFrames)]
			rightParts = append(rightParts, m.theme.InfoStyle().Render(frame+" streaming"))
		}
	}

	right := strings.Join(rightParts, muted(" · "))
	return m.joinLeftRight(left, right)
}

// joinLeftRight renders a left-aligned and right-aligned string on one line,
// filling the gap with spaces to span the full terminal width.
func (m StatusLineModel) joinLeftRight(left, right string) string {
	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	gap := m.width - leftW - rightW
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
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

// CheckCostFlash compares SessionCost against the previously observed value and
// starts a 500ms bright-white flash when all of the following hold:
//   - SessionCost has increased since the last call
//   - CostFlashEnabled is true
//   - ReduceMotion is false
//
// It must be called by the parent model immediately after updating SessionCost.
// Returns a tea.Cmd that fires CostFlashExpiredMsg after 500ms, or nil.
func (m *StatusLineModel) CheckCostFlash() tea.Cmd {
	increased := m.SessionCost > m.prevCost
	m.prevCost = m.SessionCost
	if increased && m.CostFlashEnabled && !m.ReduceMotion {
		m.costFlashUntil = time.Now().Add(500 * time.Millisecond)
		return scheduleFlashExpiry()
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

// activeCostStyle returns the lipgloss.Style to use for the cost badge. During
// an active flash it returns bright-white bold; otherwise it delegates to
// costStyle using the current SessionCost.
func (m StatusLineModel) activeCostStyle() lipgloss.Style {
	if m.CostFlashEnabled && !m.ReduceMotion && !m.costFlashUntil.IsZero() && time.Now().Before(m.costFlashUntil) {
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF"))
	}
	return m.costStyle(m.SessionCost)
}

// costStyle returns a bold lipgloss.Style based on session cost thresholds:
//   - >= $5.00  → ErrorStyle   (red, bold)
//   - >= $1.00  → WarningStyle (yellow, bold)
//   - < $1.00   → SuccessStyle (green, bold)
func (m StatusLineModel) costStyle(cost float64) lipgloss.Style {
	switch {
	case cost >= 5.00:
		return m.theme.ErrorStyle().Bold(true)
	case cost >= 1.00:
		return m.theme.WarningStyle().Bold(true)
	default:
		return m.theme.SuccessStyle().Bold(true)
	}
}

// contextStyle returns a lipgloss.Style based on context window usage thresholds:
//   - >= 90%  → ErrorStyle   (red)
//   - >= 70%  → WarningStyle (yellow)
//   - < 70%   → SuccessStyle (green)
func (m StatusLineModel) contextStyle(pct float64) lipgloss.Style {
	switch {
	case pct >= 90:
		return m.theme.ErrorStyle()
	case pct >= 70:
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

// scheduleFlashExpiry returns a command that fires CostFlashExpiredMsg after
// 500ms, ending the cost badge flash highlight.
func scheduleFlashExpiry() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return CostFlashExpiredMsg{}
	})
}

// shortenCWD shortens a CWD path for status bar display.
func shortenCWD(path string) string {
	if path == "/" {
		return "/ (root)"
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == home {
		return "~ (home)"
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}
