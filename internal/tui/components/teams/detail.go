package teams

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
)

// ---------------------------------------------------------------------------
// DiffSummary (UX-028)
// ---------------------------------------------------------------------------

// DiffSummary holds the aggregate change metrics for a completed team run,
// derived from scanning stream NDJSON files for Write/Edit tool_use events.
type DiffSummary struct {
	FilesChanged int
	LinesAdded   int
	LinesRemoved int
	TotalCost    float64
}

// isTeamComplete returns true for terminal success statuses.
func isTeamComplete(status string) bool {
	return status == "complete" || status == "completed"
}

// countLines returns the number of lines in s: 0 for empty, otherwise the
// count of newline-separated segments, treating a trailing newline as part of
// the last line rather than introducing an extra empty line.
func countLines(s string) int {
	if s == "" {
		return 0
	}
	n := strings.Count(s, "\n")
	if !strings.HasSuffix(s, "\n") {
		n++
	}
	return n
}

// scanNDJSONForChanges reads a stream NDJSON file and extracts file paths,
// lines added, and lines removed from Write/Edit tool_use content blocks.
// Best-effort: returns an empty set and zero counts on any read or parse error.
func scanNDJSONForChanges(path string) (files map[string]struct{}, linesAdded, linesRemoved int) {
	files = make(map[string]struct{})

	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	for _, rawLine := range bytes.Split(data, []byte("\n")) {
		rawLine = bytes.TrimSpace(rawLine)
		if len(rawLine) == 0 {
			continue
		}

		// Quick first-pass: skip non-assistant events.
		var ev struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(rawLine, &ev); err != nil || ev.Type != "assistant" {
			continue
		}

		// Second-pass: extract content blocks.
		var msg struct {
			Message struct {
				Content []struct {
					Type  string          `json:"type"`
					Name  string          `json:"name"`
					Input json.RawMessage `json:"input"`
				} `json:"content"`
			} `json:"message"`
		}
		if err := json.Unmarshal(rawLine, &msg); err != nil {
			continue
		}

		for _, block := range msg.Message.Content {
			if block.Type != "tool_use" {
				continue
			}
			switch block.Name {
			case "Write":
				var input struct {
					FilePath string `json:"file_path"`
					Content  string `json:"content"`
				}
				if err := json.Unmarshal(block.Input, &input); err != nil || input.FilePath == "" {
					continue
				}
				files[input.FilePath] = struct{}{}
				linesAdded += countLines(input.Content)

			case "Edit":
				var input struct {
					FilePath  string `json:"file_path"`
					OldString string `json:"old_string"`
					NewString string `json:"new_string"`
				}
				if err := json.Unmarshal(block.Input, &input); err != nil || input.FilePath == "" {
					continue
				}
				files[input.FilePath] = struct{}{}
				linesRemoved += countLines(input.OldString)
				linesAdded += countLines(input.NewString)
			}
		}
	}
	return
}

// computeDiffSummary scans stream_*.ndjson files in the team directory for
// Write/Edit tool_use events and returns aggregated change metrics.
// Best-effort: file/line counts are zero when stream files are missing or
// malformed.
func computeDiffSummary(ts *TeamState) DiffSummary {
	if ts == nil {
		return DiffSummary{}
	}
	summary := DiffSummary{TotalCost: ts.TotalCostUSD()}
	fileSet := make(map[string]struct{})

	for _, wave := range ts.Config.Waves {
		for _, member := range wave.Members {
			agentID := member.Agent
			if agentID == "" {
				agentID = member.Name
			}
			streamPath := filepath.Join(ts.Dir, fmt.Sprintf("stream_%s.ndjson", agentID))
			files, added, removed := scanNDJSONForChanges(streamPath)

			// Fallback: if primary path found nothing and Name differs from Agent.
			if len(files) == 0 && member.Name != "" && member.Name != agentID {
				altPath := filepath.Join(ts.Dir, fmt.Sprintf("stream_%s.ndjson", member.Name))
				files, added, removed = scanNDJSONForChanges(altPath)
			}

			for f := range files {
				fileSet[f] = struct{}{}
			}
			summary.LinesAdded += added
			summary.LinesRemoved += removed
		}
	}
	summary.FilesChanged = len(fileSet)
	return summary
}

// renderCompletionSummary formats a DiffSummary into a compact one-line
// completion indicator shown below the wave list when a team finishes.
func renderCompletionSummary(s DiffSummary, _ int) string {
	var detail string
	switch {
	case s.FilesChanged == 0:
		detail = fmt.Sprintf("no file changes — $%.2f", s.TotalCost)
	case s.LinesAdded > 0 || s.LinesRemoved > 0:
		detail = fmt.Sprintf("%d file(s) modified, +%d -%d lines — $%.2f",
			s.FilesChanged, s.LinesAdded, s.LinesRemoved, s.TotalCost)
	default:
		detail = fmt.Sprintf("%d file(s) modified — $%.2f", s.FilesChanged, s.TotalCost)
	}
	return config.StyleSuccess.Render("✓ done") + " — " + config.StyleMuted.Render(detail)
}

// ---------------------------------------------------------------------------
// TeamDetailModel
// ---------------------------------------------------------------------------

// TeamDetailModel renders the full details of a single team's execution,
// grouped by wave. It displays a tab bar at the top for cycling between all
// registered teams. Key navigation (left/h, right/l, d) is handled via
// HandleMsg.
//
// The zero value is usable and renders the empty-state placeholder.
type TeamDetailModel struct {
	team         *TeamState
	teamRegistry *TeamRegistry
	agents       *state.AgentRegistry // may be nil
	width        int
	height       int

	// Tab navigation state (UX-027).
	allTeams  []*TeamState    // non-dismissed teams from registry, newest-first
	activeIdx int             // index into allTeams for the active tab
	dismissed map[string]bool // dirs marked as dismissed from tab bar
	tabOffset int             // first visible tab index for overflow scrolling

	// Diff summary cache (UX-028): computed once when a team first reaches a
	// terminal-success status. Reset when the active team directory changes.
	diffSummary *DiffSummary
	diffForDir  string
}

// NewTeamDetailModel returns a TeamDetailModel backed by the given registries.
// Both teamReg and agentReg may be nil; when nil the detail renders without
// live activity data.
func NewTeamDetailModel(teamReg *TeamRegistry, agentReg *state.AgentRegistry) TeamDetailModel {
	return TeamDetailModel{
		teamRegistry: teamReg,
		agents:       agentReg,
		dismissed:    make(map[string]bool),
	}
}

// SetTeam updates the displayed team. Passing nil clears the detail pane and
// shows the empty-state placeholder. This must be called from the parent
// model's Update method — never from View.
func (m *TeamDetailModel) SetTeam(team *TeamState) {
	m.team = team
	if team != nil {
		for i, ts := range m.allTeams {
			if ts.Dir == team.Dir {
				m.activeIdx = i
				m.syncTabOffset()
				break
			}
		}
	}
}

// SetTeamByDir looks up the team by directory path in the local registry and
// updates the displayed team. A no-op when the registry is nil or the dir is
// not found.
func (m *TeamDetailModel) SetTeamByDir(dir string) {
	if m.teamRegistry == nil || dir == "" {
		return
	}
	m.team = m.teamRegistry.Get(dir)
	for i, ts := range m.allTeams {
		if ts.Dir == dir {
			m.activeIdx = i
			m.syncTabOffset()
			break
		}
	}
}

// Refresh re-reads the currently displayed team from the registry and rebuilds
// the sorted tab list. Called after every poll tick to keep the detail view
// up-to-date.
func (m *TeamDetailModel) Refresh() {
	if m.teamRegistry == nil {
		return
	}

	// Remember the active dir before rebuilding so we can restore the selection.
	activeDir := ""
	if m.team != nil {
		activeDir = m.team.Dir
	} else if m.activeIdx >= 0 && m.activeIdx < len(m.allTeams) {
		activeDir = m.allTeams[m.activeIdx].Dir
	}

	// Rebuild allTeams from registry, filtering dismissed dirs.
	all := m.teamRegistry.All() // sorted newest-first
	m.allTeams = make([]*TeamState, 0, len(all))
	for _, ts := range all {
		if !m.dismissed[ts.Dir] {
			m.allTeams = append(m.allTeams, ts)
		}
	}

	// Restore selection by dir match.
	found := false
	for i, ts := range m.allTeams {
		if ts.Dir == activeDir {
			m.activeIdx = i
			m.team = ts
			found = true
			break
		}
	}

	if !found {
		if len(m.allTeams) == 0 {
			m.activeIdx = 0
			m.team = nil
		} else {
			// Clamp index and select the team at that position.
			if m.activeIdx >= len(m.allTeams) {
				m.activeIdx = len(m.allTeams) - 1
			}
			m.team = m.allTeams[m.activeIdx]
		}
	}

	m.syncTabOffset()

	// Compute diff summary once when the active team first reaches a terminal
	// success state. Re-compute when the team directory changes.
	if m.team != nil && isTeamComplete(m.team.Config.Status) && m.diffForDir != m.team.Dir {
		s := computeDiffSummary(m.team)
		m.diffSummary = &s
		m.diffForDir = m.team.Dir
	}
}

// HandleMsg implements model.TeamDetailWidget. It handles TeamSelectedMsg by
// looking up the team in the registry and tea.KeyMsg for tab navigation
// (left/h → previous tab, right/l → next tab, d → dismiss active tab).
// All other messages are silently ignored. Never emits commands.
func (m *TeamDetailModel) HandleMsg(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case TeamSelectedMsg:
		m.SetTeamByDir(msg.Dir)
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			m.prevTab()
		case "right", "l":
			m.nextTab()
		case "d":
			m.dismissActive()
		}
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

// View implements tea.Model. It renders the tab bar followed by the full
// detail view for the currently active team, or a placeholder when no team
// is selected. The view is a pure function of the model state — no I/O.
func (m TeamDetailModel) View() string {
	if m.team == nil && len(m.allTeams) == 0 {
		return config.StyleMuted.Render("Select a team")
	}

	tabBar := m.renderTabBar(m.contentWidth())

	if m.team == nil {
		return tabBar + "\n" + config.StyleMuted.Render("Select a team")
	}

	return tabBar + "\n" + m.renderDetail()
}

// ---------------------------------------------------------------------------
// Tab navigation helpers
// ---------------------------------------------------------------------------

// nextTab advances the active tab index forward with wrap-around.
func (m *TeamDetailModel) nextTab() {
	n := len(m.allTeams)
	if n == 0 {
		return
	}
	m.activeIdx = (m.activeIdx + 1) % n
	if m.activeIdx == 0 {
		m.tabOffset = 0 // wrapped to beginning
	}
	m.syncTabOffset()
	m.team = m.allTeams[m.activeIdx]
}

// prevTab moves the active tab index backward with wrap-around.
func (m *TeamDetailModel) prevTab() {
	n := len(m.allTeams)
	if n == 0 {
		return
	}
	m.activeIdx = (m.activeIdx - 1 + n) % n
	if m.activeIdx < m.tabOffset {
		m.tabOffset = m.activeIdx
	}
	m.syncTabOffset()
	m.team = m.allTeams[m.activeIdx]
}

// dismissActive marks the currently active team as dismissed (removed from the
// tab bar). The team remains in the registry; it is only hidden locally.
func (m *TeamDetailModel) dismissActive() {
	if len(m.allTeams) == 0 || m.dismissed == nil {
		return
	}
	dir := m.allTeams[m.activeIdx].Dir
	m.dismissed[dir] = true
	m.Refresh()
}

// syncTabOffset adjusts tabOffset so that activeIdx is within the visible
// window. Uses a conservative average tab width estimate since actual label
// widths are only known during rendering.
func (m *TeamDetailModel) syncTabOffset() {
	n := len(m.allTeams)
	if n == 0 {
		m.tabOffset = 0
		return
	}
	// Clamp activeIdx.
	if m.activeIdx >= n {
		m.activeIdx = n - 1
	}
	if m.activeIdx < 0 {
		m.activeIdx = 0
	}
	// Scroll left: active went before visible range.
	if m.activeIdx < m.tabOffset {
		m.tabOffset = m.activeIdx
		return
	}
	// Scroll right: estimate visible window from available width.
	if m.width <= 0 {
		return
	}
	const avgTabWidth = 14 // " label " where label ≤ 12 chars
	const overhead = 10    // badge + scroll indicators
	availWidth := m.width - overhead
	if availWidth < avgTabWidth {
		m.tabOffset = m.activeIdx
		return
	}
	maxVisible := availWidth / avgTabWidth
	if maxVisible < 1 {
		maxVisible = 1
	}
	for m.activeIdx >= m.tabOffset+maxVisible {
		m.tabOffset++
	}
	// Clamp tabOffset.
	if m.tabOffset >= n {
		m.tabOffset = n - 1
	}
	if m.tabOffset < 0 {
		m.tabOffset = 0
	}
}

// ---------------------------------------------------------------------------
// Tab bar rendering
// ---------------------------------------------------------------------------

const maxTabLabelLen = 12 // maximum visible chars per tab label before truncation

// renderTabBar renders a single-line tab bar with:
//   - scroll indicators (< and >) when tabs overflow the available width
//   - active tab highlighted in accent color (bold + underline + primary foreground)
//   - inactive tabs in muted style
//   - a count badge "(N)" at the right edge
//
// Returns a "No teams (0)" placeholder when the registry contains no teams.
func (m TeamDetailModel) renderTabBar(width int) string {
	n := len(m.allTeams)
	badge := fmt.Sprintf(" (%d)", n)

	if n == 0 {
		return config.StyleMuted.Render("No teams" + badge)
	}

	// Available width for tab labels (reserve badge and up to two indicators).
	badgeWidth := len(badge) // visual width (no ANSI codes in badge text)
	availWidth := width - badgeWidth
	if availWidth < 6 {
		availWidth = 6
	}

	// Build per-tab label entries.
	type tabEntry struct {
		label  string // visible label (already truncated)
		vWidth int    // visual width: len(" " + label + " ")
	}
	entries := make([]tabEntry, n)
	for i, ts := range m.allTeams {
		label := filepath.Base(ts.Dir)
		if len(label) > maxTabLabelLen {
			label = label[:maxTabLabelLen-1] + "…"
		}
		entries[i] = tabEntry{label: label, vWidth: len(label) + 2}
	}

	// Determine the visible range starting from tabOffset.
	offset := m.tabOffset
	if offset >= n {
		offset = n - 1
	}
	if offset < 0 {
		offset = 0
	}

	needLeft := offset > 0
	usedWidth := 0
	if needLeft {
		usedWidth += 2 // "< "
	}

	lastVisible := offset - 1
	for i := offset; i < n; i++ {
		if usedWidth+entries[i].vWidth > availWidth {
			break
		}
		usedWidth += entries[i].vWidth
		lastVisible = i
	}

	needRight := lastVisible < n-1

	// If a right indicator is needed but doesn't fit, drop the last visible tab.
	if needRight && usedWidth+2 > availWidth && lastVisible > offset {
		usedWidth -= entries[lastVisible].vWidth
		lastVisible--
	}

	// Render the tab bar.
	var sb strings.Builder

	if needLeft {
		sb.WriteString(config.StyleMuted.Render("< "))
	}

	activeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(config.ColorPrimary).
		Underline(true)

	for i := offset; i <= lastVisible; i++ {
		label := " " + entries[i].label + " "
		if i == m.activeIdx {
			sb.WriteString(activeStyle.Render(label))
		} else {
			sb.WriteString(config.StyleMuted.Render(label))
		}
	}

	if needRight {
		sb.WriteString(config.StyleMuted.Render(" >"))
	}

	sb.WriteString(config.StyleMuted.Render(badge))

	return sb.String()
}

// ---------------------------------------------------------------------------
// Detail rendering (extracted from the original View method)
// ---------------------------------------------------------------------------

// renderDetail renders the full team detail body (header, status, waves).
// It assumes m.team is non-nil.
func (m TeamDetailModel) renderDetail() string {
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

	// Completion summary (UX-028): shown once all waves are rendered.
	if isTeamComplete(m.team.Config.Status) && m.diffSummary != nil {
		sb.WriteByte('\n')
		sb.WriteString(renderCompletionSummary(*m.diffSummary, m.contentWidth()))
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
