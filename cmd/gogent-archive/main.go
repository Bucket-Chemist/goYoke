package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/session"
)

const DEFAULT_TIMEOUT = 5 * time.Second

func main() {
	// Subcommand routing
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "list":
			listSessions()
			return
		case "show":
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "[gogent-archive] Usage: gogent-archive show <session-id>")
				fmt.Fprintln(os.Stderr, "  Missing required argument: session-id")
				fmt.Fprintln(os.Stderr, "  Example: gogent-archive show abc123def456")
				os.Exit(1)
			}
			showSession(os.Args[2])
			return
		case "stats":
			showStats()
			return
		case "--help", "-h":
			printHelp()
			return
		case "--version", "-v":
			fmt.Printf("gogent-archive version %s\n", getVersion())
			return
		}
	}

	// Default: SessionEnd hook mode (existing behavior)
	if err := run(); err != nil {
		outputError(err.Error())
		os.Exit(1)
	}
}

func run() error {
	// Determine project directory from env or cwd
	projectDir := os.Getenv("GOGENT_PROJECT_DIR")
	if projectDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("[gogent-archive] Failed to get working directory: %w. Set GOGENT_PROJECT_DIR environment variable or run from project root.", err)
		}
		projectDir = cwd
	}

	// Parse SessionEnd event from STDIN with timeout
	event, err := session.ParseSessionEvent(os.Stdin, DEFAULT_TIMEOUT)
	if err != nil {
		return fmt.Errorf("[gogent-archive] Failed to parse SessionEnd event: %w. Ensure hook provides valid JSON on STDIN.", err)
	}

	// Collect session metrics
	metrics, err := session.CollectSessionMetrics(event.SessionID)
	if err != nil {
		return fmt.Errorf("[gogent-archive] Failed to collect metrics for session %s: %w. Check temp files exist and are readable.", event.SessionID, err)
	}

	// Generate JSONL handoff
	handoffCfg := session.DefaultHandoffConfig(projectDir)
	handoff, hMetrics, err := session.GenerateHandoff(handoffCfg, metrics)
	if err != nil {
		return fmt.Errorf("[gogent-archive] Failed to generate handoff: %w", err)
	}

	if handoff == nil {
		return fmt.Errorf("[gogent-archive] No handoff data generated. This may be normal for first session. Cannot generate markdown for empty handoff.")
	}

	// Log generation metrics at debug level (internal use only)
	_ = hMetrics // Metrics available for future use (e.g., performance monitoring)

	// Render markdown for human consumption
	mdPath := filepath.Join(projectDir, ".claude", "memory", "last-handoff.md")
	if err := os.MkdirAll(filepath.Dir(mdPath), 0755); err != nil {
		return fmt.Errorf("[gogent-archive] Failed to create directory for %s: %w", mdPath, err)
	}
	markdown := session.RenderHandoffMarkdown(handoff)
	if err := os.WriteFile(mdPath, []byte(markdown), 0644); err != nil {
		return fmt.Errorf("[gogent-archive] Failed to write markdown to %s: %w", mdPath, err)
	}

	// Archive artifacts AFTER handoff generation
	if err := session.ArchiveArtifacts(*handoffCfg, event.SessionID); err != nil {
		return fmt.Errorf("[gogent-archive] Failed to archive artifacts: %w", err)
	}

	// Output confirmation JSON matching bash hook format
	confirmation := map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName":     "SessionEnd",
			"additionalContext": fmt.Sprintf("📦 SESSION ARCHIVED: Handoff saved to %s. JSONL history at %s.", mdPath, handoffCfg.HandoffPath),
			"handoff_jsonl":     handoffCfg.HandoffPath,
			"handoff_md":        mdPath,
			"session_id":        event.SessionID,
			"metrics": map[string]int{
				"tool_calls": metrics.ToolCalls,
				"errors":     metrics.ErrorsLogged,
				"violations": metrics.RoutingViolations,
			},
		},
	}

	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(confirmation); err != nil {
		return fmt.Errorf("[gogent-archive] Failed to encode confirmation JSON: %w. Check stdout is writable.", err)
	}

	return nil
}

// outputError writes error message in hook-compatible JSON format
func outputError(message string) {
	output := map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName":     "SessionEnd",
			"additionalContext": "🔴 " + message,
		},
	}

	encoder := json.NewEncoder(os.Stdout)
	_ = encoder.Encode(output) // Best effort - if this fails, nothing we can do
}

// getProjectDir determines project directory from env or cwd
// Exits with error if detection fails (matching run() behavior)
func getProjectDir() string {
	projectDir := os.Getenv("GOGENT_PROJECT_DIR")
	if projectDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[gogent-archive] Failed to get working directory: %v\n", err)
			fmt.Fprintln(os.Stderr, "  Set GOGENT_PROJECT_DIR environment variable or run from project root.")
			os.Exit(1)
		}
		projectDir = cwd
	}
	return projectDir
}

// listSessions displays session history as a table with optional filtering
func listSessions() {
	// Parse flags
	listFlags := flag.NewFlagSet("list", flag.ExitOnError)
	sinceFlag := listFlags.String("since", "", "Filter sessions since duration (e.g., 7d) or date (YYYY-MM-DD)")
	betweenFlag := listFlags.String("between", "", "Filter sessions between dates (YYYY-MM-DD,YYYY-MM-DD)")
	hasSharpEdges := listFlags.Bool("has-sharp-edges", false, "Show only sessions with sharp edges")
	hasViolations := listFlags.Bool("has-violations", false, "Show only sessions with routing violations")
	clean := listFlags.Bool("clean", false, "Show only sessions with no sharp edges or violations")
	listFlags.Parse(os.Args[2:])

	projectDir := getProjectDir()
	handoffPath := filepath.Join(projectDir, ".claude", "memory", "handoffs.jsonl")

	handoffs, err := session.LoadAllHandoffs(handoffPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-archive] Failed to load handoffs: %v\n", err)
		fmt.Fprintln(os.Stderr, "  Verify .claude/memory/handoffs.jsonl exists and is readable.")
		os.Exit(1)
	}

	// CRITICAL: Handle zero sessions gracefully (Acceptance Criteria requirement)
	if len(handoffs) == 0 {
		fmt.Println("No sessions recorded. Run Claude Code in this project to generate session history.")
		return
	}

	// Apply date filtering
	if *sinceFlag != "" {
		handoffs = filterSince(handoffs, *sinceFlag)
	}
	if *betweenFlag != "" {
		handoffs = filterBetween(handoffs, *betweenFlag)
	}

	// Apply artifact presence filters
	if *hasSharpEdges || *hasViolations || *clean {
		handoffs = filterByArtifacts(handoffs, *hasSharpEdges, *hasViolations, *clean)
	}

	// Check again after filtering
	if len(handoffs) == 0 {
		fmt.Println("No sessions match the specified filters.")
		return
	}

	// Print table header
	fmt.Println("Session ID                    | Timestamp  | Tool Calls | Errors | Violations")
	fmt.Println("------------------------------|------------|------------|--------|------------")

	for _, h := range handoffs {
		timestamp := time.Unix(h.Timestamp, 0).Format("2006-01-02")
		fmt.Printf("%-30s | %-10s | %10d | %6d | %10d\n",
			h.SessionID, timestamp, h.Context.Metrics.ToolCalls,
			h.Context.Metrics.ErrorsLogged, h.Context.Metrics.RoutingViolations)
	}
}

// showSession renders a specific session handoff as markdown to stdout
func showSession(sessionID string) {
	projectDir := getProjectDir()
	handoffPath := filepath.Join(projectDir, ".claude", "memory", "handoffs.jsonl")

	handoffs, err := session.LoadAllHandoffs(handoffPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-archive] Failed to load handoffs: %v\n", err)
		fmt.Fprintln(os.Stderr, "  Verify .claude/memory/handoffs.jsonl exists and is readable.")
		os.Exit(1)
	}

	for _, h := range handoffs {
		if h.SessionID == sessionID {
			// CRITICAL FIX: Use session.RenderHandoffMarkdown (existing function)
			// NOT generateMarkdownContent (doesn't exist!)
			markdown := session.RenderHandoffMarkdown(&h)
			fmt.Print(markdown)
			return
		}
	}

	// Session not found
	fmt.Fprintf(os.Stderr, "[gogent-archive] Session %s not found in handoff history.\n", sessionID)
	fmt.Fprintln(os.Stderr, "  Run 'gogent-archive list' to see available sessions.")
	os.Exit(1)
}

// showStats displays aggregate session statistics with breakdowns
func showStats() {
	projectDir := getProjectDir()
	handoffPath := filepath.Join(projectDir, ".claude", "memory", "handoffs.jsonl")

	handoffs, err := session.LoadAllHandoffs(handoffPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-archive] Failed to load handoffs: %v\n", err)
		fmt.Fprintln(os.Stderr, "  Verify .claude/memory/handoffs.jsonl exists and is readable.")
		os.Exit(1)
	}

	if len(handoffs) == 0 {
		fmt.Println("No sessions recorded. Run Claude Code in this project to generate session history.")
		return
	}

	// Aggregate metrics
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

		// Aggregate error types
		for _, edge := range h.Artifacts.SharpEdges {
			errorTypes[edge.ErrorType]++
		}

		// Aggregate violation types
		for _, violation := range h.Artifacts.RoutingViolations {
			violationTypes[violation.ViolationType]++
		}
	}

	avgToolCalls := 0
	if totalSessions > 0 {
		avgToolCalls = totalToolCalls / totalSessions
	}

	// Print core stats
	fmt.Printf("Total Sessions: %d\n", totalSessions)
	fmt.Printf("Avg Tool Calls per Session: %d\n", avgToolCalls)
	fmt.Printf("Total Errors: %d\n", totalErrors)
	fmt.Printf("Total Violations: %d\n", totalViolations)

	// Print error breakdown if errors exist
	if len(errorTypes) > 0 {
		fmt.Println("\nErrors Breakdown:")
		for errType, count := range errorTypes {
			fmt.Printf("  - %s: %d sessions\n", errType, count)
		}
	}

	// Print violation breakdown if violations exist
	if len(violationTypes) > 0 {
		fmt.Println("\nViolations Breakdown:")
		for violationType, count := range violationTypes {
			fmt.Printf("  - %s: %d sessions\n", violationType, count)
		}
	}
}

// printHelp displays usage information for all subcommands
func printHelp() {
	fmt.Println("gogent-archive - Session handoff archival and querying")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  gogent-archive                       Read SessionEnd JSON from STDIN (hook mode)")
	fmt.Println("  gogent-archive list                  List all sessions")
	fmt.Println("  gogent-archive list --since 7d       List sessions from last 7 days")
	fmt.Println("  gogent-archive list --between <dates> List sessions between dates (YYYY-MM-DD,YYYY-MM-DD)")
	fmt.Println("  gogent-archive list --has-sharp-edges Show only sessions with sharp edges")
	fmt.Println("  gogent-archive list --has-violations Show only sessions with routing violations")
	fmt.Println("  gogent-archive list --clean          Show only clean sessions (no errors/violations)")
	fmt.Println("  gogent-archive show <id>             Show specific session handoff")
	fmt.Println("  gogent-archive stats                 Show aggregate statistics with breakdowns")
	fmt.Println("  gogent-archive --help                Show this help")
	fmt.Println("  gogent-archive --version             Show version information")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  gogent-archive list --since 2026-01-15")
	fmt.Println("  gogent-archive list --between 2026-01-01,2026-01-15 --clean")
	fmt.Println("  gogent-archive show abc123def456")
	fmt.Println("")
	fmt.Println("For subcommand-specific help, use: gogent-archive <subcommand> --help")
}

// getVersion returns version from build ldflags or "dev"
func getVersion() string {
	// This will be set by -ldflags "-X main.version=..." during build
	version := "dev"
	return version
}

// filterSince filters handoffs by duration (e.g., "7d") or date (YYYY-MM-DD)
func filterSince(handoffs []session.Handoff, since string) []session.Handoff {
	now := time.Now()
	var cutoff time.Time

	// Try parsing as duration first (e.g., "7d", "30d")
	if strings.HasSuffix(since, "d") {
		daysStr := strings.TrimSuffix(since, "d")
		days, err := strconv.Atoi(daysStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[gogent-archive] Invalid --since format '%s'\n", since)
			fmt.Fprintln(os.Stderr, "  Use duration format (e.g., '7d', '30d') or date format (YYYY-MM-DD)")
			fmt.Fprintln(os.Stderr, "  Example: --since 7d OR --since 2026-01-15")
			os.Exit(1)
		}
		cutoff = now.AddDate(0, 0, -days)
	} else {
		// Try parsing as date (YYYY-MM-DD)
		parsedDate, err := time.Parse("2006-01-02", since)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[gogent-archive] Invalid --since date format '%s'\n", since)
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

// filterBetween filters handoffs between two dates (YYYY-MM-DD,YYYY-MM-DD)
func filterBetween(handoffs []session.Handoff, between string) []session.Handoff {
	parts := strings.Split(between, ",")
	if len(parts) != 2 {
		fmt.Fprintf(os.Stderr, "[gogent-archive] Invalid --between format '%s'\n", between)
		fmt.Fprintln(os.Stderr, "  Expected format: YYYY-MM-DD,YYYY-MM-DD")
		fmt.Fprintln(os.Stderr, "  Example: --between 2026-01-01,2026-01-15")
		os.Exit(1)
	}

	startDate, err := time.Parse("2006-01-02", parts[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-archive] Invalid start date in --between: %v\n", err)
		fmt.Fprintln(os.Stderr, "  Expected YYYY-MM-DD format for start date")
		os.Exit(1)
	}

	endDate, err := time.Parse("2006-01-02", parts[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-archive] Invalid end date in --between: %v\n", err)
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

// filterByArtifacts filters handoffs by presence of sharp edges, violations, or clean sessions
func filterByArtifacts(handoffs []session.Handoff, hasSharpEdges, hasViolations, clean bool) []session.Handoff {
	var filtered []session.Handoff
	for _, h := range handoffs {
		sharpEdgeCount := len(h.Artifacts.SharpEdges)
		violationCount := len(h.Artifacts.RoutingViolations)

		// Clean filter: EXCLUDE sessions with any artifacts
		if clean && (sharpEdgeCount > 0 || violationCount > 0) {
			continue
		}

		// Sharp edges filter: EXCLUDE sessions without sharp edges
		if hasSharpEdges && sharpEdgeCount == 0 {
			continue
		}

		// Violations filter: EXCLUDE sessions without violations
		if hasViolations && violationCount == 0 {
			continue
		}

		filtered = append(filtered, h)
	}
	return filtered
}
