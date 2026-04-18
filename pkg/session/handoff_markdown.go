package session

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// RenderHandoffMarkdown converts a Handoff struct to human-readable markdown
func RenderHandoffMarkdown(h *Handoff) string {
	var sb strings.Builder

	// Header
	timestamp := time.Unix(h.Timestamp, 0).Format("2006-01-02 15:04:05")
	sb.WriteString(fmt.Sprintf("# Session Handoff - %s\n\n", timestamp))

	// Session Context
	sb.WriteString("## Session Context\n\n")
	sb.WriteString(fmt.Sprintf("- **Session ID**: %s\n", h.SessionID))
	sb.WriteString(fmt.Sprintf("- **Project**: %s\n", h.Context.ProjectDir))
	if h.Context.ActiveTicket != "" {
		sb.WriteString(fmt.Sprintf("- **Active Ticket**: %s\n", h.Context.ActiveTicket))
	}
	if h.Context.Phase != "" {
		sb.WriteString(fmt.Sprintf("- **Phase**: %s\n", h.Context.Phase))
	}
	sb.WriteString("\n")

	// Session Metrics
	sb.WriteString("## Session Metrics\n\n")
	sb.WriteString(fmt.Sprintf("- **Tool Calls**: %d\n", h.Context.Metrics.ToolCalls))
	sb.WriteString(fmt.Sprintf("- **Errors Logged**: %d\n", h.Context.Metrics.ErrorsLogged))
	sb.WriteString(fmt.Sprintf("- **Routing Violations**: %d\n", h.Context.Metrics.RoutingViolations))
	sb.WriteString("\n")

	// Git Info (if present)
	if h.Context.GitInfo.Branch != "" {
		sb.WriteString("## Git State\n\n")
		sb.WriteString(fmt.Sprintf("- **Branch**: %s\n", h.Context.GitInfo.Branch))
		if h.Context.GitInfo.IsDirty {
			sb.WriteString("- **Status**: Uncommitted changes present\n")
			if len(h.Context.GitInfo.Uncommitted) > 0 {
				sb.WriteString("- **Uncommitted Files**:\n")
				for _, file := range h.Context.GitInfo.Uncommitted {
					sb.WriteString(fmt.Sprintf("  - %s\n", file))
				}
			}
		} else {
			sb.WriteString("- **Status**: Clean\n")
		}
		sb.WriteString("\n")
	}

	// Sharp Edges
	if len(h.Artifacts.SharpEdges) > 0 {
		sb.WriteString("## Sharp Edges\n\n")
		for _, edge := range h.Artifacts.SharpEdges {
			sb.WriteString(fmt.Sprintf("- **%s**: %s (%d consecutive failures)\n",
				edge.File, edge.ErrorType, edge.ConsecutiveFailures))
			if edge.Context != "" {
				sb.WriteString(fmt.Sprintf("  - Context: %s\n", edge.Context))
			}
		}
		sb.WriteString("\n")
	}

	// Routing Violations
	if len(h.Artifacts.RoutingViolations) > 0 {
		sb.WriteString("## Routing Violations\n\n")
		for _, v := range h.Artifacts.RoutingViolations {
			sb.WriteString(fmt.Sprintf("- **%s**: %s", v.Agent, v.ViolationType))
			if v.ExpectedTier != "" && v.ActualTier != "" {
				sb.WriteString(fmt.Sprintf(" (expected: %s, actual: %s)", v.ExpectedTier, v.ActualTier))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// Error Patterns
	if len(h.Artifacts.ErrorPatterns) > 0 {
		sb.WriteString("## Error Patterns\n\n")
		for _, p := range h.Artifacts.ErrorPatterns {
			sb.WriteString(fmt.Sprintf("- **%s**: %d occurrences\n", p.ErrorType, p.Count))
			if p.Context != "" {
				sb.WriteString(fmt.Sprintf("  - Context: %s\n", p.Context))
			}
		}
		sb.WriteString("\n")
	}

	// Actions
	if len(h.Actions) > 0 {
		sb.WriteString("## Immediate Actions\n\n")
		for _, action := range h.Actions {
			sb.WriteString(fmt.Sprintf("%d. %s\n", action.Priority, action.Description))
			if action.Context != "" {
				sb.WriteString(fmt.Sprintf("   - %s\n", action.Context))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// FormatSharpEdge formats a single sharp edge as a markdown line
// Includes ErrorMessage (truncated to 100 chars) and Severity badge when present
func FormatSharpEdge(edge SharpEdge) string {
	var sb strings.Builder

	// Add severity badge if present
	if edge.Severity != "" {
		badge := map[string]string{
			"high":   "🔴",
			"medium": "🟡",
			"low":    "🟢",
		}[edge.Severity]
		if badge == "" {
			badge = "⚪"
		}
		sb.WriteString(badge)
		sb.WriteString(" ")
	}

	// Base format: - **file**: error_type (N failures)
	sb.WriteString(fmt.Sprintf("- **%s**: %s (%d failures)",
		edge.File,
		edge.ErrorType,
		edge.ConsecutiveFailures,
	))

	// Add error message if present (truncated to 100 chars)
	if edge.ErrorMessage != "" {
		truncated := edge.ErrorMessage
		if len(truncated) > 100 {
			truncated = truncated[:100] + "..."
		}
		sb.WriteString(fmt.Sprintf("\n  Error: `%s`", truncated))
	}

	// Add resolution if present
	if edge.Resolution != "" {
		sb.WriteString(fmt.Sprintf("\n  ✅ Resolved: %s", edge.Resolution))
	}

	return sb.String()
}

// FormatSharpEdges formats multiple sharp edges as a markdown section
func FormatSharpEdges(edges []SharpEdge) string {
	if len(edges) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Sharp Edges\n\n")

	for _, edge := range edges {
		sb.WriteString(FormatSharpEdge(edge))
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// RenderHandoffSummary creates a brief one-line summary of a handoff
func RenderHandoffSummary(h *Handoff) string {
	timestamp := time.Unix(h.Timestamp, 0).Format("2006-01-02 15:04")
	summary := fmt.Sprintf("[%s] Session %s: %d tool calls",
		timestamp, h.SessionID, h.Context.Metrics.ToolCalls)

	if len(h.Artifacts.SharpEdges) > 0 {
		summary += fmt.Sprintf(", %d sharp edge(s)", len(h.Artifacts.SharpEdges))
	}
	if len(h.Artifacts.RoutingViolations) > 0 {
		summary += fmt.Sprintf(", %d violation(s)", len(h.Artifacts.RoutingViolations))
	}

	return summary
}

// truncateString truncates a string to maxLen with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// FormatUserIntent formats a single user intent as a markdown line
// Uses confidence badges: explicit=💬, inferred=🤔, default=⚪
func FormatUserIntent(intent UserIntent) string {
	var sb strings.Builder

	// Confidence badge
	badge := map[string]string{
		"explicit": "💬",
		"inferred": "🤔",
		"default":  "⚪",
	}[intent.Confidence]
	if badge == "" {
		badge = "❓"
	}

	// Base format with question and answer
	sb.WriteString(fmt.Sprintf("%s **Q:** %s\n   **A:** %s",
		badge,
		truncateString(intent.Question, 80),
		truncateString(intent.Response, 100),
	))

	// Add action if present
	if intent.ActionTaken != "" {
		sb.WriteString(fmt.Sprintf("\n   ➡️ Action: %s", truncateString(intent.ActionTaken, 80)))
	}

	return sb.String()
}

// FormatUserIntents formats multiple user intents as a markdown section
func FormatUserIntents(intents []UserIntent) string {
	if len(intents) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## User Intents\n\n")

	for _, intent := range intents {
		sb.WriteString(FormatUserIntent(intent))
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// RenderAllHandoffs creates a markdown document showing all handoffs
func RenderAllHandoffs(handoffs []Handoff) string {
	var sb strings.Builder

	sb.WriteString("# Session History\n\n")
	sb.WriteString(fmt.Sprintf("Total sessions: %d\n\n", len(handoffs)))

	if len(handoffs) == 0 {
		sb.WriteString("No sessions recorded.\n")
		return sb.String()
	}

	// Show summary of each session
	sb.WriteString("## Session Summary\n\n")
	for i, h := range handoffs {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, RenderHandoffSummary(&h)))
	}
	sb.WriteString("\n")

	// Show full detail of most recent session
	sb.WriteString("## Most Recent Session\n\n")
	sb.WriteString(RenderHandoffMarkdown(&handoffs[len(handoffs)-1]))

	return sb.String()
}

// FormatWeeklyIntentSummary renders the intent section for weekly report
func FormatWeeklyIntentSummary(summary WeeklyIntentSummary) string {
	var sb strings.Builder

	sb.WriteString("## User Intents This Week\n\n")
	sb.WriteString(fmt.Sprintf("**Total Captured:** %d intents across %d sessions\n\n",
		summary.TotalIntents, summary.SessionCount))

	// goYoke-041c: Honor rate section
	if summary.TotalAnalyzed > 0 {
		sb.WriteString("**Honor Rate:**\n")
		sb.WriteString(fmt.Sprintf("- Overall: %.0f%% (%d/%d)\n",
			summary.HonorRatePercent, summary.TotalHonored, summary.TotalAnalyzed))

		// Per-category honor rates (sorted by rate descending)
		if len(summary.HonorRateByCategory) > 0 {
			type catRate struct {
				cat  string
				rate float64
			}
			var sorted []catRate
			for cat, rate := range summary.HonorRateByCategory {
				sorted = append(sorted, catRate{cat, rate})
			}
			sort.Slice(sorted, func(i, j int) bool {
				return sorted[i].rate > sorted[j].rate
			})

			for _, cr := range sorted {
				icon := ""
				if cr.rate < 60 {
					icon = " ⚠️" // Low honor rate alert
				}
				sb.WriteString(fmt.Sprintf("- %s: %.0f%%%s\n", cr.cat, cr.rate, icon))
			}
		}

		// Alert for low overall honor rate
		if summary.HonorRatePercent < 70 {
			sb.WriteString("\n**Low Honor Rate Alert:** Overall rate below 70% - review preferences\n")
		}

		sb.WriteString("\n")
	}

	// Category distribution
	if len(summary.CategoryDistribution) > 0 {
		sb.WriteString("**By Category:**\n")
		// Sort by percentage descending
		type catPct struct {
			cat string
			pct float64
		}
		var sorted []catPct
		for cat, pct := range summary.CategoryPercentages {
			sorted = append(sorted, catPct{cat, pct})
		}
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].pct > sorted[j].pct
		})

		for _, cp := range sorted {
			sb.WriteString(fmt.Sprintf("- %s: %d (%.0f%%)\n",
				cp.cat, summary.CategoryDistribution[cp.cat], cp.pct))
		}
		sb.WriteString("\n")
	}

	// Recurring preferences
	if len(summary.RecurringPreferences) > 0 {
		sb.WriteString("**Recurring Preferences:**\n")
		// Limit to top 5 for display
		limit := min(5, len(summary.RecurringPreferences))
		for i := 0; i < limit; i++ {
			pref := summary.RecurringPreferences[i]
			sb.WriteString(fmt.Sprintf("- %q (%d sessions, %s)\n",
				truncate(pref.Pattern, 40), len(pref.SessionIDs), pref.Category))
		}
		sb.WriteString("\n")
	}

	// Drift alerts
	if len(summary.DriftAlerts) > 0 {
		sb.WriteString("**Preference Changes:**\n")
		for _, alert := range summary.DriftAlerts {
			icon := "🆕"
			if alert.Type == "dropped" {
				icon = "❌"
			} else if alert.Type == "changed" {
				icon = "🔄"
			}
			sb.WriteString(fmt.Sprintf("- %s %s\n", icon, alert.Message))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
