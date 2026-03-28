package teams

// TODO: multi-team view — currently shows most recent running team only

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/model"
)

// TeamsHealthModel is a display-only health dashboard for a running team.
// It renders the team header, budget bar, wave/member breakdown, and footer.
//
// The zero value is not usable; use NewTeamsHealthModel instead.
type TeamsHealthModel struct {
	registry *TeamRegistry
	width    int
	height   int
	tier     model.LayoutTier
}

// NewTeamsHealthModel returns a TeamsHealthModel backed by the given registry.
func NewTeamsHealthModel(reg *TeamRegistry) *TeamsHealthModel {
	return &TeamsHealthModel{registry: reg}
}

// SetSize stores the available rendering dimensions.
func (m *TeamsHealthModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetTier stores the responsive layout tier.
func (m *TeamsHealthModel) SetTier(tier model.LayoutTier) {
	m.tier = tier
}

// View renders the health dashboard as a pure string. It is safe to call
// from a Bubbletea View() method: data is obtained via MostRecentRunning()
// which holds only an RLock internally.
func (m *TeamsHealthModel) View() string {
	ts := m.registry.MostRecentRunning()
	if ts == nil {
		empty := config.StyleMuted.Render("No active teams")
		if m.width > 0 && m.height > 0 {
			return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, empty)
		}
		return empty
	}

	var sb strings.Builder

	sb.WriteString(m.renderHeader(ts))
	sb.WriteByte('\n')
	sb.WriteString(m.renderBudget(ts))

	for _, wave := range ts.Config.Waves {
		sb.WriteString("\n\n")
		sb.WriteString(m.renderWaveHeader(wave, ts))
		sb.WriteByte('\n')
		for _, member := range wave.Members {
			sb.WriteString(m.renderMember(member, wave.WaveNumber, ts.StreamSizes))
			sb.WriteByte('\n')
		}
	}

	sb.WriteString("\n")
	sb.WriteString(m.renderFooter(ts))

	out := strings.TrimRight(sb.String(), "\n")

	if m.height > 0 {
		out = m.scrollToCurrentWave(out, ts)
	}

	return out
}

// renderHeader renders team name, status, first-running member PID, and uptime.
func (m *TeamsHealthModel) renderHeader(ts *TeamState) string {
	name := ts.Config.TeamName
	if name == "" {
		name = "unnamed"
	}

	statusStr := statusStyleFor(ts.Config.Status).Render(ts.Config.Status)

	// Use the first running member's PID as a proxy for the team PID.
	pidStr := ""
outer:
	for _, w := range ts.Config.Waves {
		for _, mem := range w.Members {
			if mem.ProcessPID != nil && mem.Status == "running" {
				pidStr = fmt.Sprintf("  PID %d", *mem.ProcessPID)
				break outer
			}
		}
	}

	uptime := ""
	if ts.Config.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, ts.Config.CreatedAt); err == nil {
			uptime = "  " + formatDuration(time.Since(t))
		}
	}

	return config.StyleTitle.Render(fmt.Sprintf("Team: %s", name)) +
		"  " + statusStr + pidStr + uptime
}

// renderBudget renders the budget bar with Unicode block characters.
func (m *TeamsHealthModel) renderBudget(ts *TeamState) string {
	maxUSD := ts.Config.BudgetMaxUSD
	remaining := ts.Config.BudgetRemainingUSD
	if maxUSD <= 0 {
		return config.StyleMuted.Render("Budget: —")
	}

	used := maxUSD - remaining
	if used < 0 {
		used = 0
	}
	usedPct := used / maxUSD
	if usedPct > 1.0 {
		usedPct = 1.0
	}

	barWidth := 20
	if m.width > 50 {
		barWidth = m.width - 30
		if barWidth > 40 {
			barWidth = 40
		}
	}

	filled := int(float64(barWidth) * usedPct)
	empty := barWidth - filled

	bar := budgetColor(usedPct).Render(strings.Repeat("█", filled)) +
		config.StyleMuted.Render(strings.Repeat("░", empty))

	label := fmt.Sprintf(" %.0f%% ($%.2f / $%.2f)", usedPct*100, used, maxUSD)
	return "Budget: " + bar + config.StyleMuted.Render(label)
}

// renderWaveHeader renders the wave section divider. Marks the current wave.
func (m *TeamsHealthModel) renderWaveHeader(wave Wave, ts *TeamState) string {
	cur := ts.CurrentWaveNumber()
	label := fmt.Sprintf("── Wave %d", wave.WaveNumber)
	if wave.Description != "" {
		label += " · " + wave.Description
	}
	if wave.WaveNumber == cur {
		label += " (current)"
	}
	return config.StyleSubtle.Render(label)
}

// renderMember renders a single member row. waveNum is the 1-based wave number
// the member belongs to, used to compute the "waiting for Wave N" message.
func (m *TeamsHealthModel) renderMember(member Member, waveNum int, streamSizes map[string]int64) string {
	icon := healthIcon(member)
	statusCol := config.StyleMuted.Render(fmt.Sprintf("%-12s", member.Status))

	switch member.Status {
	case "failed":
		msg := member.ErrorMessage
		if member.KillReason != "" {
			msg = member.KillReason
		}
		return fmt.Sprintf("  %s %-14s %s  %s",
			icon, member.Name, statusCol, config.StyleError.Render(msg))

	case "pending":
		waitMsg := "waiting to start"
		if waveNum > 1 {
			waitMsg = fmt.Sprintf("waiting for Wave %d", waveNum-1)
		}
		return fmt.Sprintf("  %s %-14s %s  %s",
			icon, member.Name, statusCol, config.StyleMuted.Render(waitMsg))

	default:
		pidStr := fmt.Sprintf("%-10s", "")
		if member.ProcessPID != nil {
			pidStr = fmt.Sprintf("PID %-6d", *member.ProcessPID)
		}

		activity := ""
		if member.LastActivityTime != nil {
			activity = formatRelativeTime(*member.LastActivityTime)
		}

		streamSize := int64(0)
		if streamSizes != nil {
			streamSize = streamSizes[member.Name]
		}

		return fmt.Sprintf("  %s %-14s %s  %s  %d stalls  %-12s  %s",
			icon, member.Name, statusCol,
			pidStr,
			member.StallCount,
			config.StyleMuted.Render(activity),
			config.StyleMuted.Render(formatBytes(streamSize)),
		)
	}
}

// renderFooter renders the summary statistics line.
func (m *TeamsHealthModel) renderFooter(ts *TeamState) string {
	totalWaves := len(ts.Config.Waves)
	totalMembers := 0
	for _, w := range ts.Config.Waves {
		totalMembers += len(w.Members)
	}
	return config.StyleMuted.Render(
		fmt.Sprintf("%d waves · %d members · $%.2f spent",
			totalWaves, totalMembers, ts.TotalCostUSD()),
	)
}

// scrollToCurrentWave clips the rendered output to m.height lines, scrolled
// so the current wave header is near the top. Falls back to the end of content
// when no running wave can be identified.
func (m *TeamsHealthModel) scrollToCurrentWave(out string, ts *TeamState) string {
	lines := strings.Split(out, "\n")
	if len(lines) <= m.height {
		return out
	}

	cur := ts.CurrentWaveNumber()
	target := fmt.Sprintf("── Wave %d", cur)

	waveLineIdx := -1
	for i, line := range lines {
		if strings.Contains(line, target) {
			waveLineIdx = i
			break
		}
	}

	start := 0
	if waveLineIdx < 0 {
		start = len(lines) - m.height
	} else {
		start = waveLineIdx - 2
	}
	if start < 0 {
		start = 0
	}

	end := start + m.height
	if end > len(lines) {
		end = len(lines)
		start = end - m.height
		if start < 0 {
			start = 0
		}
	}

	return strings.Join(lines[start:end], "\n")
}

// formatDuration formats a positive duration as a human-readable string
// ("3m 42s", "1h 5m", "47s"). Negative durations render as "0s".
func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	h := int(d.Hours())
	mins := int(d.Minutes()) % 60
	secs := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, mins)
	}
	if mins > 0 {
		return fmt.Sprintf("%dm %ds", mins, secs)
	}
	return fmt.Sprintf("%ds", secs)
}

// ---------------------------------------------------------------------------
// Exported helper functions (tested in health_test.go)
// ---------------------------------------------------------------------------

// formatRelativeTime converts an ISO 8601 (RFC3339) timestamp to a
// human-readable relative time string:
//
//   - empty input  → ""
//   - parse error  → ""
//   - < 1s         → "just now"
//   - < 60s        → "Xs ago"
//   - < 60m        → "Xm Ys ago"
//   - >= 1h        → "Xh Ym ago"
func formatRelativeTime(iso8601 string) string {
	if iso8601 == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, iso8601)
	if err != nil {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Second:
		return "just now"
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds ago", mins, secs)
	default:
		hours := int(d.Hours())
		mins := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh %dm ago", hours, mins)
	}
}

// formatBytes converts a byte count to a human-readable string:
//
//   - 0        → "0B"
//   - < 1 KiB  → "NB"
//   - < 1 MiB  → "NKB"
//   - >= 1 MiB → "N.XMB"
func formatBytes(b int64) string {
	switch {
	case b == 0:
		return "0B"
	case b < 1024:
		return fmt.Sprintf("%dB", b)
	case b < 1024*1024:
		return fmt.Sprintf("%dKB", b/1024)
	default:
		return fmt.Sprintf("%.1fMB", float64(b)/(1024*1024))
	}
}

// healthIcon returns a styled icon string derived from a member's Status and
// HealthStatus fields. Status takes priority; HealthStatus is consulted only
// for members that are neither pending, failed, nor completed.
//
// Icon reference:
//
//	● U+25CF  filled circle   (healthy / completed / stalled)
//	▲ U+25B2  up triangle     (stall_warning)
//	◻ U+25FB  white square    (pending)
//	✕ U+2715  multiplication  (failed)
func healthIcon(member Member) string {
	switch member.Status {
	case "failed":
		return config.StyleError.Render("✕")
	case "pending":
		return config.StyleMuted.Render("◻")
	case "completed":
		return config.StyleSuccess.Render("●")
	}
	// Active (running or unrecognised): reflect health monitoring state.
	switch member.HealthStatus {
	case "stall_warning":
		return config.StyleWarning.Render("▲")
	case "stalled":
		return config.StyleError.Render("●")
	default: // "healthy", "" or any unrecognised value
		return config.StyleSuccess.Render("●")
	}
}

// budgetColor returns the lipgloss style appropriate for a given budget usage fraction:
//
//   - < 70%  → config.StyleSuccess (green)
//   - 70–90% → config.StyleWarning (yellow)
//   - > 90%  → config.StyleError   (red)
func budgetColor(usedPct float64) lipgloss.Style {
	switch {
	case usedPct > 0.90:
		return config.StyleError
	case usedPct >= 0.70:
		return config.StyleWarning
	default:
		return config.StyleSuccess
	}
}
