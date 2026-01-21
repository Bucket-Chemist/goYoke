package session

import (
	"fmt"
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
