package teams

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
)

// ---------------------------------------------------------------------------
// TeamDetailModel
// ---------------------------------------------------------------------------

// TeamDetailModel renders the full details of a single team's execution,
// grouped by wave. It is display-only: Update is a no-op and the model never
// emits commands.
//
// The zero value is usable and renders the empty-state placeholder.
type TeamDetailModel struct {
	team   *TeamState
	width  int
	height int
}

// NewTeamDetailModel returns a TeamDetailModel in the empty (no-team) state.
func NewTeamDetailModel() TeamDetailModel {
	return TeamDetailModel{}
}

// SetTeam updates the displayed team. Passing nil clears the detail pane and
// shows the empty-state placeholder. This must be called from the parent
// model's Update method — never from View.
func (m *TeamDetailModel) SetTeam(team *TeamState) {
	m.team = team
}

// SetSize updates the viewport dimensions for responsive rendering.
func (m *TeamDetailModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Init implements tea.Model. The detail view requires no startup commands.
func (m TeamDetailModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model. The detail pane is display-only; all messages
// are ignored and no commands are emitted.
func (m TeamDetailModel) Update(_ tea.Msg) (TeamDetailModel, tea.Cmd) {
	return m, nil
}

// View implements tea.Model. It renders the full detail view for the currently
// set team, or a placeholder when no team is selected. The view is a pure
// function of the model state — no I/O is performed here.
func (m TeamDetailModel) View() string {
	if m.team == nil {
		return config.StyleMuted.Render("Select a team")
	}

	var sb strings.Builder

	// -------------------------------------------------------------------------
	// Header row
	// -------------------------------------------------------------------------
	header := fmt.Sprintf("Team: %s (%s)",
		m.team.Config.TeamName, m.team.Config.WorkflowType)
	sb.WriteString(config.StyleTitle.Render(header))
	sb.WriteByte('\n')

	// -------------------------------------------------------------------------
	// Status + cost row
	// -------------------------------------------------------------------------
	totalCost := m.team.TotalCostUSD()
	statusStr := m.team.Config.Status
	if statusStr == "" {
		statusStr = "unknown"
	}
	sb.WriteString(config.StyleMuted.Render("Status: "))
	sb.WriteString(statusStyleFor(statusStr).Render(statusStr))
	sb.WriteString(config.StyleMuted.Render(fmt.Sprintf("  Cost: $%.3f", totalCost)))
	sb.WriteByte('\n')

	// Divider
	dividerWidth := m.contentWidth()
	if dividerWidth < 1 {
		dividerWidth = 40
	}
	sb.WriteString(strings.Repeat("─", dividerWidth))
	sb.WriteByte('\n')

	// -------------------------------------------------------------------------
	// Waves
	// -------------------------------------------------------------------------
	if len(m.team.Config.Waves) == 0 {
		sb.WriteString(config.StyleMuted.Render("No waves"))
		return strings.TrimRight(sb.String(), "\n")
	}

	for _, wave := range m.team.Config.Waves {
		// Wave header
		waveHeader := fmt.Sprintf("Wave %d: %s", wave.WaveNumber, wave.Description)
		sb.WriteString(config.StyleTitle.Render(waveHeader))
		sb.WriteByte('\n')

		// Members
		if len(wave.Members) == 0 {
			sb.WriteString(config.StyleMuted.Render("  (no members)"))
			sb.WriteByte('\n')
			continue
		}

		for _, member := range wave.Members {
			sb.WriteString(m.renderMember(member))
			sb.WriteByte('\n')
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}

// renderMember renders a single wave member row.
//
// Format:   [icon] name   status   $cost   elapsed
func (m TeamDetailModel) renderMember(member Member) string {
	icon := statusIcon(member.Status)
	iconStr := statusStyleFor(member.Status).Render(string(icon))

	name := member.Name
	if name == "" {
		name = member.Agent
	}

	statusStr := member.Status
	if statusStr == "" {
		statusStr = "pending"
	}

	costStr := "—"
	if member.CostUSD > 0 {
		costStr = fmt.Sprintf("$%.2f", member.CostUSD)
	}

	elapsed := m.memberElapsed(member)

	return fmt.Sprintf("  %s %-20s  %-10s  %6s  %s",
		iconStr, name, statusStr, costStr, elapsed)
}

// memberElapsed computes a human-readable elapsed time for a member.
// Returns "—" when no timing information is available.
func (m TeamDetailModel) memberElapsed(member Member) string {
	if member.StartedAt == nil {
		return "—"
	}

	start, err := time.Parse(time.RFC3339, *member.StartedAt)
	if err != nil {
		return "—"
	}

	var end time.Time
	if member.CompletedAt != nil {
		t, err := time.Parse(time.RFC3339, *member.CompletedAt)
		if err == nil {
			end = t
		}
	}

	if end.IsZero() {
		// Still running: use current time.
		end = time.Now()
	}

	d := end.Sub(start).Round(time.Second)
	return formatMemberDuration(d)
}

// formatMemberDuration formats a duration as "Xm Ys" or "Xs".
func formatMemberDuration(d time.Duration) string {
	if d < 0 {
		return "—"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	mins := int(d.Minutes())
	secs := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%ds", mins, secs)
}

// contentWidth returns the usable text width for the detail pane.
func (m TeamDetailModel) contentWidth() int {
	if m.width < 20 {
		return 20
	}
	return m.width
}
