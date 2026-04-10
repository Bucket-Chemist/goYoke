package teams

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
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
	team         *TeamState
	teamRegistry *TeamRegistry
	agents       *state.AgentRegistry // may be nil
	width        int
	height       int
}

// NewTeamDetailModel returns a TeamDetailModel backed by the given registries.
// Both teamReg and agentReg may be nil; when nil the detail renders without
// live activity data.
func NewTeamDetailModel(teamReg *TeamRegistry, agentReg *state.AgentRegistry) TeamDetailModel {
	return TeamDetailModel{
		teamRegistry: teamReg,
		agents:       agentReg,
	}
}

// SetTeam updates the displayed team. Passing nil clears the detail pane and
// shows the empty-state placeholder. This must be called from the parent
// model's Update method — never from View.
func (m *TeamDetailModel) SetTeam(team *TeamState) {
	m.team = team
}

// SetTeamByDir looks up the team by directory path in the local registry and
// updates the displayed team. A no-op when the registry is nil or the dir is
// not found.
func (m *TeamDetailModel) SetTeamByDir(dir string) {
	if m.teamRegistry == nil || dir == "" {
		return
	}
	m.team = m.teamRegistry.Get(dir)
}

// Refresh re-reads the currently displayed team from the registry. Called
// after every poll tick to keep the detail view up-to-date.
func (m *TeamDetailModel) Refresh() {
	if m.team == nil || m.teamRegistry == nil {
		return
	}
	m.team = m.teamRegistry.Get(m.team.Dir)
}

// HandleMsg implements model.TeamDetailWidget. It handles TeamSelectedMsg by
// looking up the team in the registry. All other messages are silently
// ignored. Never emits commands.
func (m *TeamDetailModel) HandleMsg(msg tea.Msg) tea.Cmd {
	if sel, ok := msg.(TeamSelectedMsg); ok {
		m.SetTeamByDir(sel.Dir)
	}
	return nil
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
			// M-2: Resolve agent once per member, pass to both render functions.
			var agent *state.Agent
			if m.agents != nil && member.Status == "running" {
				agent = m.agents.Get(m.teamAgentID(member))
			}
			sb.WriteString(m.renderMember(member, agent))
			sb.WriteByte('\n')
			if agent != nil && len(agent.RecentActivity) > 0 {
				feed := m.renderMemberFeed(agent)
				if feed != "" {
					sb.WriteString(feed)
					sb.WriteByte('\n')
				}
			}
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}

// teamAgentID reconstructs the agent registry key for a team member.
// Format matches spawn.go: "team:{teamDirBasename}:{memberName}"
func (m TeamDetailModel) teamAgentID(member Member) string {
	if m.team == nil {
		return ""
	}
	return "team:" + filepath.Base(m.team.Dir) + ":" + member.Name
}

// renderMember renders a single wave member row.
//
// Format:   [icon] name   status   $cost   elapsed   [tool activity]
func (m TeamDetailModel) renderMember(member Member, agent *state.Agent) string {
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

	line := fmt.Sprintf("  %s %-20s  %-10s  %6s  %s",
		iconStr, name, statusStr, costStr, elapsed)

	// M-1: Append current tool activity for running members with truncation.
	if agent != nil && agent.Activity != nil {
		toolInfo := formatToolActivity(agent.Activity)
		// Measure approximate available width using raw text length as proxy.
		rawLineLen := 2 + 1 + 1 + 20 + 2 + 10 + 2 + 6 + 2 + len(elapsed)
		maxToolWidth := m.contentWidth() - rawLineLen - 2
		if maxToolWidth > 10 {
			if len(toolInfo) > maxToolWidth {
				toolInfo = toolInfo[:maxToolWidth-1] + "…"
			}
			line += "  " + config.StyleMuted.Render(toolInfo)
		}
	}

	return line
}

// renderMemberFeed renders up to 3 most-recent activity entries below a
// running member row.
func (m TeamDetailModel) renderMemberFeed(agent *state.Agent) string {
	recent := agent.RecentActivity
	if len(recent) > 3 {
		recent = recent[len(recent)-3:]
	}
	var sb strings.Builder
	for _, act := range recent {
		icon := activityIcon(act.Success)
		tool := act.ToolName
		if tool == "" {
			tool = act.Type
		}
		target := act.Target
		maxW := m.contentWidth() - 12 // indent + icon + tool + padding
		if maxW > 0 && len(target) > maxW {
			target = target[:maxW-1] + "…"
		}
		if target != "" {
			sb.WriteString(fmt.Sprintf("      %s %s %s", icon, tool, config.StyleMuted.Render(target)))
		} else {
			sb.WriteString(fmt.Sprintf("      %s %s", icon, tool))
		}
		sb.WriteByte('\n')
	}
	return strings.TrimRight(sb.String(), "\n")
}

// formatToolActivity formats a single AgentActivity into a compact one-liner.
func formatToolActivity(act *state.AgentActivity) string {
	if act.Target != "" {
		return fmt.Sprintf("%s %s", act.ToolName, act.Target)
	}
	return act.ToolName
}

// activityIcon returns a status icon for a completed/pending/failed activity.
func activityIcon(success *bool) string {
	if success == nil {
		return "⏳"
	}
	if *success {
		return "✓"
	}
	return "✗"
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
