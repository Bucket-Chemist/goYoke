package archive

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/config"
	"github.com/Bucket-Chemist/goYoke/pkg/session"
)

// listSessions displays session history as a table with optional filtering.
func listSessions() {
	listFlags := flag.NewFlagSet("list", flag.ExitOnError)
	sinceFlag := listFlags.String("since", "", "Filter sessions since duration (e.g., 7d) or date (YYYY-MM-DD)")
	betweenFlag := listFlags.String("between", "", "Filter sessions between dates (YYYY-MM-DD,YYYY-MM-DD)")
	hasSharpEdges := listFlags.Bool("has-sharp-edges", false, "Show only sessions with sharp edges")
	hasViolations := listFlags.Bool("has-violations", false, "Show only sessions with routing violations")
	clean := listFlags.Bool("clean", false, "Show only sessions with no sharp edges or violations")
	listFlags.Parse(os.Args[2:])

	projectDir := getProjectDir()
	handoffPath := filepath.Join(config.ProjectMemoryDir(projectDir), "handoffs.jsonl")

	handoffs, err := session.LoadAllHandoffs(handoffPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-archive] Failed to load handoffs: %v\n", err)
		fmt.Fprintln(os.Stderr, "  Verify .goyoke/memory/handoffs.jsonl exists and is readable.")
		os.Exit(1)
	}

	if len(handoffs) == 0 {
		fmt.Println("No sessions recorded. Run Claude Code in this project to generate session history.")
		return
	}

	if *sinceFlag != "" {
		handoffs = filterSince(handoffs, *sinceFlag)
	}
	if *betweenFlag != "" {
		handoffs = filterBetween(handoffs, *betweenFlag)
	}

	if *hasSharpEdges || *hasViolations || *clean {
		handoffs = filterByArtifacts(handoffs, *hasSharpEdges, *hasViolations, *clean)
	}

	if len(handoffs) == 0 {
		fmt.Println("No sessions match the specified filters.")
		return
	}

	fmt.Println("Session ID                    | Timestamp  | Tool Calls | Errors | Violations")
	fmt.Println("------------------------------|------------|------------|--------|------------")

	for _, h := range handoffs {
		timestamp := time.Unix(h.Timestamp, 0).Format("2006-01-02")
		fmt.Printf("%-30s | %-10s | %10d | %6d | %10d\n",
			h.SessionID, timestamp, h.Context.Metrics.ToolCalls,
			h.Context.Metrics.ErrorsLogged, h.Context.Metrics.RoutingViolations)
	}
}

// showSession renders a specific session handoff as markdown to stdout.
func showSession(sessionID string) {
	projectDir := getProjectDir()
	handoffPath := filepath.Join(config.ProjectMemoryDir(projectDir), "handoffs.jsonl")

	handoffs, err := session.LoadAllHandoffs(handoffPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-archive] Failed to load handoffs: %v\n", err)
		fmt.Fprintln(os.Stderr, "  Verify .goyoke/memory/handoffs.jsonl exists and is readable.")
		os.Exit(1)
	}

	for _, h := range handoffs {
		if h.SessionID == sessionID {
			markdown := session.RenderHandoffMarkdown(&h)
			fmt.Print(markdown)
			return
		}
	}

	fmt.Fprintf(os.Stderr, "[goyoke-archive] Session %s not found in handoff history.\n", sessionID)
	fmt.Fprintln(os.Stderr, "  Run 'goyoke-archive list' to see available sessions.")
	os.Exit(1)
}

// showStats displays aggregate session statistics with breakdowns.
func showStats() {
	projectDir := getProjectDir()
	handoffPath := filepath.Join(config.ProjectMemoryDir(projectDir), "handoffs.jsonl")

	handoffs, err := session.LoadAllHandoffs(handoffPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-archive] Failed to load handoffs: %v\n", err)
		fmt.Fprintln(os.Stderr, "  Verify .goyoke/memory/handoffs.jsonl exists and is readable.")
		os.Exit(1)
	}

	if len(handoffs) == 0 {
		fmt.Println("No sessions recorded. Run Claude Code in this project to generate session history.")
		return
	}

	totalSessions := len(handoffs)
	totalToolCalls := 0
	totalErrors := 0
	totalViolations := 0
	errorTypes := make(map[string]int)
	violationTypes := make(map[string]int)

	for _, h := range handoffs {
		totalToolCalls += h.Context.Metrics.ToolCalls
		totalErrors += h.Context.Metrics.ErrorsLogged
		totalViolations += h.Context.Metrics.RoutingViolations

		for _, edge := range h.Artifacts.SharpEdges {
			errorTypes[edge.ErrorType]++
		}

		for _, violation := range h.Artifacts.RoutingViolations {
			violationTypes[violation.ViolationType]++
		}
	}

	avgToolCalls := 0
	if totalSessions > 0 {
		avgToolCalls = totalToolCalls / totalSessions
	}

	fmt.Printf("Total Sessions: %d\n", totalSessions)
	fmt.Printf("Avg Tool Calls per Session: %d\n", avgToolCalls)
	fmt.Printf("Total Errors: %d\n", totalErrors)
	fmt.Printf("Total Violations: %d\n", totalViolations)

	if len(errorTypes) > 0 {
		fmt.Println("\nErrors Breakdown:")
		for errType, count := range errorTypes {
			fmt.Printf("  - %s: %d sessions\n", errType, count)
		}
	}

	if len(violationTypes) > 0 {
		fmt.Println("\nViolations Breakdown:")
		for violationType, count := range violationTypes {
			fmt.Printf("  - %s: %d sessions\n", violationType, count)
		}
	}
}

// generateWeeklySummary displays weekly intent summary with optional filters.
func generateWeeklySummary() {
	weeklyFlags := flag.NewFlagSet("weekly", flag.ExitOnError)
	intentsOnlyFlag := weeklyFlags.Bool("intents-only", false, "Show only intent summary section")
	driftFlag := weeklyFlags.Bool("drift", false, "Highlight preference changes (drift alerts)")
	sinceFlag := weeklyFlags.String("since", "", "Start date for weekly range (YYYY-MM-DD, defaults to 7 days ago)")
	weeklyFlags.Parse(os.Args[2:])

	projectDir := getProjectDir()

	var weekStart, weekEnd time.Time
	if *sinceFlag != "" {
		parsedDate, err := time.Parse("2006-01-02", *sinceFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[goyoke-archive] Invalid --since date format '%s'\n", *sinceFlag)
			fmt.Fprintln(os.Stderr, "  Expected YYYY-MM-DD format (e.g., '2026-01-15')")
			os.Exit(1)
		}
		weekStart = parsedDate
		weekEnd = weekStart.AddDate(0, 0, 7)
	} else {
		now := time.Now()
		weekEnd = now
		weekStart = now.AddDate(0, 0, -7)
	}

	intentsPath := filepath.Join(config.ProjectMemoryDir(projectDir), "user-intents.jsonl")
	intents, err := session.LoadAllUserIntents(intentsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-archive] Failed to load user intents: %v\n", err)
		os.Exit(1)
	}

	if len(intents) == 0 {
		fmt.Println("No user intents recorded for the specified period.")
		return
	}

	currentSummary := session.AggregateWeeklyIntents(intents, weekStart, weekEnd)

	if *driftFlag {
		prevWeekStart := weekStart.AddDate(0, 0, -7)
		prevWeekEnd := weekStart
		previousSummary := session.AggregateWeeklyIntents(intents, prevWeekStart, prevWeekEnd)
		currentSummary.DriftAlerts = session.DetectPreferenceDrift(currentSummary, previousSummary)
	}

	if *intentsOnlyFlag {
		fmt.Print(session.FormatWeeklyIntentSummary(currentSummary))
	} else {
		fmt.Printf("# Weekly Summary - %s to %s\n\n",
			weekStart.Format("2006-01-02"),
			weekEnd.Format("2006-01-02"))
		fmt.Print(session.FormatWeeklyIntentSummary(currentSummary))
	}
}

// filterSince filters handoffs by duration (e.g., "7d") or date (YYYY-MM-DD).
func filterSince(handoffs []session.Handoff, since string) []session.Handoff {
	now := time.Now()
	var cutoff time.Time

	if daysStr, ok := strings.CutSuffix(since, "d"); ok {
		days, err := strconv.Atoi(daysStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[goyoke-archive] Invalid --since format '%s'\n", since)
			fmt.Fprintln(os.Stderr, "  Use duration format (e.g., '7d', '30d') or date format (YYYY-MM-DD)")
			fmt.Fprintln(os.Stderr, "  Example: --since 7d OR --since 2026-01-15")
			os.Exit(1)
		}
		cutoff = now.AddDate(0, 0, -days)
	} else {
		parsedDate, err := time.Parse("2006-01-02", since)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[goyoke-archive] Invalid --since date format '%s'\n", since)
			fmt.Fprintln(os.Stderr, "  Expected YYYY-MM-DD format (e.g., '2026-01-15')")
			os.Exit(1)
		}
		cutoff = parsedDate
	}

	var filtered []session.Handoff
	for _, h := range handoffs {
		sessionTime := time.Unix(h.Timestamp, 0)
		if sessionTime.After(cutoff) || sessionTime.Equal(cutoff) {
			filtered = append(filtered, h)
		}
	}
	return filtered
}

// filterBetween filters handoffs between two dates (YYYY-MM-DD,YYYY-MM-DD).
func filterBetween(handoffs []session.Handoff, between string) []session.Handoff {
	parts := strings.Split(between, ",")
	if len(parts) != 2 {
		fmt.Fprintf(os.Stderr, "[goyoke-archive] Invalid --between format '%s'\n", between)
		fmt.Fprintln(os.Stderr, "  Expected format: YYYY-MM-DD,YYYY-MM-DD")
		fmt.Fprintln(os.Stderr, "  Example: --between 2026-01-01,2026-01-15")
		os.Exit(1)
	}

	startDate, err := time.Parse("2006-01-02", parts[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-archive] Invalid start date in --between: %v\n", err)
		fmt.Fprintln(os.Stderr, "  Expected YYYY-MM-DD format for start date")
		os.Exit(1)
	}

	endDate, err := time.Parse("2006-01-02", parts[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-archive] Invalid end date in --between: %v\n", err)
		fmt.Fprintln(os.Stderr, "  Expected YYYY-MM-DD format for end date")
		os.Exit(1)
	}

	var filtered []session.Handoff
	for _, h := range handoffs {
		sessionTime := time.Unix(h.Timestamp, 0)
		if (sessionTime.After(startDate) || sessionTime.Equal(startDate)) &&
			(sessionTime.Before(endDate) || sessionTime.Equal(endDate)) {
			filtered = append(filtered, h)
		}
	}
	return filtered
}

// filterByArtifacts filters handoffs by presence of sharp edges, violations, or clean sessions.
func filterByArtifacts(handoffs []session.Handoff, hasSharpEdges, hasViolations, clean bool) []session.Handoff {
	var filtered []session.Handoff
	for _, h := range handoffs {
		sharpEdgeCount := len(h.Artifacts.SharpEdges)
		violationCount := len(h.Artifacts.RoutingViolations)

		if clean && (sharpEdgeCount > 0 || violationCount > 0) {
			continue
		}

		if hasSharpEdges && sharpEdgeCount == 0 {
			continue
		}

		if hasViolations && violationCount == 0 {
			continue
		}

		filtered = append(filtered, h)
	}
	return filtered
}
