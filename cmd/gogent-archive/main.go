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
		case "sharp-edges":
			listSharpEdges()
			return
		case "user-intents":
			listUserIntents()
			return
		case "decisions":
			listDecisions()
			return
		case "preferences":
			listPreferences()
			return
		case "performance":
			showPerformance()
			return
		case "weekly":
			generateWeeklySummary()
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
	fmt.Println("")
	fmt.Println("Session Commands:")
	fmt.Println("  gogent-archive list                  List all sessions")
	fmt.Println("  gogent-archive list --since 7d       List sessions from last 7 days")
	fmt.Println("  gogent-archive list --between <dates> List sessions between dates (YYYY-MM-DD,YYYY-MM-DD)")
	fmt.Println("  gogent-archive list --has-sharp-edges Show only sessions with sharp edges")
	fmt.Println("  gogent-archive list --has-violations Show only sessions with routing violations")
	fmt.Println("  gogent-archive list --clean          Show only clean sessions (no errors/violations)")
	fmt.Println("  gogent-archive show <id>             Show specific session handoff")
	fmt.Println("  gogent-archive stats                 Show aggregate statistics with breakdowns")
	fmt.Println("")
	fmt.Println("Weekly Analysis Commands:")
	fmt.Println("  gogent-archive weekly                Generate weekly intent summary (last 7 days)")
	fmt.Println("  gogent-archive weekly --since <date> Generate summary from specific start date (YYYY-MM-DD)")
	fmt.Println("  gogent-archive weekly --intents-only Show only intent section")
	fmt.Println("  gogent-archive weekly --drift        Show preference changes (drift alerts)")
	fmt.Println("")
	fmt.Println("Sharp Edge Commands:")
	fmt.Println("  gogent-archive sharp-edges           List all sharp edges")
	fmt.Println("  gogent-archive sharp-edges --severity high  Filter by severity (high, medium, low)")
	fmt.Println("  gogent-archive sharp-edges --file 'pkg/*'   Filter by file pattern (glob)")
	fmt.Println("  gogent-archive sharp-edges --error-type <type> Filter by error type")
	fmt.Println("  gogent-archive sharp-edges --unresolved     Show only unresolved edges")
	fmt.Println("  gogent-archive sharp-edges --since 7d       Filter by time")
	fmt.Println("")
	fmt.Println("User Intent Commands:")
	fmt.Println("  gogent-archive user-intents          List all user intents")
	fmt.Println("  gogent-archive user-intents --source ask_user  Filter by source (ask_user, hook_prompt, manual)")
	fmt.Println("  gogent-archive user-intents --confidence explicit  Filter by confidence (explicit, inferred, default)")
	fmt.Println("  gogent-archive user-intents --category routing  Filter by category (routing, tooling, style, etc.)")
	fmt.Println("  gogent-archive user-intents --keyword pytest    Filter by keyword")
	fmt.Println("  gogent-archive user-intents --has-action    Show only intents with actions taken")
	fmt.Println("  gogent-archive user-intents --since 7d      Filter by time")
	fmt.Println("")
	fmt.Println("Decision Commands:")
	fmt.Println("  gogent-archive decisions              List all decisions")
	fmt.Println("  gogent-archive decisions --category architecture  Filter by category (architecture, tooling, pattern)")
	fmt.Println("  gogent-archive decisions --impact high      Filter by impact level (high, medium, low)")
	fmt.Println("  gogent-archive decisions --since 7d         Filter by time")
	fmt.Println("")
	fmt.Println("Preference Commands:")
	fmt.Println("  gogent-archive preferences            List all preference overrides")
	fmt.Println("  gogent-archive preferences --category routing  Filter by category (routing, tooling, formatting)")
	fmt.Println("  gogent-archive preferences --scope project   Filter by scope (session, project, global)")
	fmt.Println("  gogent-archive preferences --since 7d        Filter by time")
	fmt.Println("")
	fmt.Println("Performance Commands:")
	fmt.Println("  gogent-archive performance            List all performance metrics")
	fmt.Println("  gogent-archive performance --by-operation    Group metrics by operation (summary view)")
	fmt.Println("  gogent-archive performance --slow-only       Show only slow operations (>1000ms)")
	fmt.Println("  gogent-archive performance --since 7d        Filter by time")
	fmt.Println("")
	fmt.Println("Other Commands:")
	fmt.Println("  gogent-archive --help                Show this help")
	fmt.Println("  gogent-archive --version             Show version information")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  gogent-archive sharp-edges --severity high --unresolved")
	fmt.Println("  gogent-archive user-intents --source ask_user --has-action")
	fmt.Println("  gogent-archive decisions --category architecture --impact high")
	fmt.Println("  gogent-archive preferences --scope project")
	fmt.Println("  gogent-archive performance --by-operation --slow-only")
	fmt.Println("  gogent-archive list --since 2026-01-15 --clean")
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

// listSharpEdges displays sharp edges with optional filtering
func listSharpEdges() {
	edgesFlags := flag.NewFlagSet("sharp-edges", flag.ExitOnError)
	severityFlag := edgesFlags.String("severity", "", "Filter by severity (high, medium, low)")
	fileFlag := edgesFlags.String("file", "", "Filter by file pattern (glob)")
	errorTypeFlag := edgesFlags.String("error-type", "", "Filter by error type")
	unresolvedFlag := edgesFlags.Bool("unresolved", false, "Show only unresolved edges")
	sinceFlag := edgesFlags.String("since", "", "Filter since duration (7d) or date (YYYY-MM-DD)")
	edgesFlags.Parse(os.Args[2:])

	projectDir := getProjectDir()
	q := session.NewQuery(projectDir)

	// Build filters
	filters := session.SharpEdgeFilters{}
	if *severityFlag != "" {
		filters.Severity = severityFlag
	}
	if *fileFlag != "" {
		filters.File = fileFlag
	}
	if *errorTypeFlag != "" {
		filters.ErrorType = errorTypeFlag
	}
	if *unresolvedFlag {
		filters.Unresolved = true
	}
	if *sinceFlag != "" {
		since := parseSinceFilter(*sinceFlag)
		timestamp := since.Unix()
		filters.Since = &timestamp
	}

	edges, err := q.QuerySharpEdges(filters)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-archive] Failed to query sharp edges: %v\n", err)
		os.Exit(1)
	}

	if len(edges) == 0 {
		fmt.Println("No sharp edges recorded.")
		return
	}

	// Print table header
	fmt.Println("File                           | Error Type          | Failures | Severity | Status")
	fmt.Println("-------------------------------|---------------------|----------|----------|--------")

	for _, edge := range edges {
		file := truncateForTable(edge.File, 30)
		errType := truncateForTable(edge.ErrorType, 19)
		severity := edge.Severity
		if severity == "" {
			severity = "-"
		}
		status := "Open"
		if edge.ResolvedAt != 0 {
			status = "Resolved"
		}
		fmt.Printf("%-30s | %-19s | %8d | %-8s | %s\n",
			file, errType, edge.ConsecutiveFailures, severity, status)
	}

	fmt.Printf("\nTotal: %d sharp edge(s)\n", len(edges))
}

// listUserIntents displays user intents with optional filtering
func listUserIntents() {
	intentsFlags := flag.NewFlagSet("user-intents", flag.ExitOnError)
	sourceFlag := intentsFlags.String("source", "", "Filter by source (ask_user, hook_prompt, manual)")
	confidenceFlag := intentsFlags.String("confidence", "", "Filter by confidence (explicit, inferred, default)")
	hasActionFlag := intentsFlags.Bool("has-action", false, "Show only intents with actions taken")
	sinceFlag := intentsFlags.String("since", "", "Filter since duration (7d) or date (YYYY-MM-DD)")
	categoryFlag := intentsFlags.String("category", "", "Filter by category (routing, tooling, style, etc.)")
	keywordFlag := intentsFlags.String("keyword", "", "Filter by keyword")
	intentsFlags.Parse(os.Args[2:])

	projectDir := getProjectDir()
	q := session.NewQuery(projectDir)

	// Build filters
	filters := session.UserIntentFilters{}
	if *sourceFlag != "" {
		filters.Source = sourceFlag
	}
	if *confidenceFlag != "" {
		filters.Confidence = confidenceFlag
	}
	if *hasActionFlag {
		filters.HasAction = true
	}
	if *sinceFlag != "" {
		since := parseSinceFilter(*sinceFlag)
		timestamp := since.Unix()
		filters.Since = &timestamp
	}
	// GOgent-041: Category and keyword filters
	if *categoryFlag != "" {
		filters.Category = categoryFlag
	}
	if *keywordFlag != "" {
		filters.Keywords = []string{*keywordFlag}
	}

	intents, err := q.QueryUserIntents(filters)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-archive] Failed to query user intents: %v\n", err)
		os.Exit(1)
	}

	if len(intents) == 0 {
		fmt.Println("No user intents recorded.")
		return
	}

	// Print table header (GOgent-041: Added Category column)
	fmt.Println("Timestamp  | Category   | Source      | Question                     | Response")
	fmt.Println("-----------|------------|-------------|------------------------------|---------------------------")

	for _, intent := range intents {
		timestamp := time.Unix(intent.Timestamp, 0).Format("2006-01-02")
		category := truncateForTable(intent.Category, 10)
		if category == "" {
			category = "-"
		}
		source := truncateForTable(intent.Source, 11)
		question := truncateForTable(intent.Question, 28)
		response := truncateForTable(intent.Response, 25)
		fmt.Printf("%s | %-10s | %-11s | %-28s | %s\n",
			timestamp, category, source, question, response)
	}

	fmt.Printf("\nTotal: %d user intent(s)\n", len(intents))
}

// listDecisions displays decisions with optional filtering
func listDecisions() {
	decisionsFlags := flag.NewFlagSet("decisions", flag.ExitOnError)
	categoryFlag := decisionsFlags.String("category", "", "Filter by category (architecture, tooling, pattern)")
	impactFlag := decisionsFlags.String("impact", "", "Filter by impact level (high, medium, low)")
	sinceFlag := decisionsFlags.String("since", "", "Filter since duration (7d) or date (YYYY-MM-DD)")
	decisionsFlags.Parse(os.Args[2:])

	projectDir := getProjectDir()
	q := session.NewQuery(projectDir)

	// Build filters
	filters := session.DecisionFilters{}
	if *categoryFlag != "" {
		filters.Category = categoryFlag
	}
	if *impactFlag != "" {
		filters.Impact = impactFlag
	}
	if *sinceFlag != "" {
		since := parseSinceFilter(*sinceFlag)
		timestamp := since.Unix()
		filters.Since = &timestamp
	}

	decisions, err := q.QueryDecisions(filters)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-archive] Failed to query decisions: %v\n", err)
		os.Exit(1)
	}

	if len(decisions) == 0 {
		fmt.Println("No decisions recorded.")
		return
	}

	// Print table header
	fmt.Println("Timestamp  | Category     | Impact | Decision                       | Rationale")
	fmt.Println("-----------|--------------|--------|--------------------------------|--------------------------")

	for _, d := range decisions {
		timestamp := time.Unix(d.Timestamp, 0).Format("2006-01-02")
		category := truncateForTable(d.Category, 12)
		impact := truncateForTable(d.Impact, 6)
		decision := truncateForTable(d.Decision, 30)
		rationale := truncateForTable(d.Rationale, 24)
		fmt.Printf("%s | %-12s | %-6s | %-30s | %s\n",
			timestamp, category, impact, decision, rationale)
	}

	fmt.Printf("\nTotal: %d decision(s)\n", len(decisions))
}

// listPreferences displays preference overrides with optional filtering
func listPreferences() {
	prefsFlags := flag.NewFlagSet("preferences", flag.ExitOnError)
	categoryFlag := prefsFlags.String("category", "", "Filter by category (routing, tooling, formatting)")
	scopeFlag := prefsFlags.String("scope", "", "Filter by scope (session, project, global)")
	sinceFlag := prefsFlags.String("since", "", "Filter since duration (7d) or date (YYYY-MM-DD)")
	prefsFlags.Parse(os.Args[2:])

	projectDir := getProjectDir()
	q := session.NewQuery(projectDir)

	// Build filters
	filters := session.PreferenceFilters{}
	if *categoryFlag != "" {
		filters.Category = categoryFlag
	}
	if *scopeFlag != "" {
		filters.Scope = scopeFlag
	}
	if *sinceFlag != "" {
		since := parseSinceFilter(*sinceFlag)
		timestamp := since.Unix()
		filters.Since = &timestamp
	}

	preferences, err := q.QueryPreferences(filters)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-archive] Failed to query preferences: %v\n", err)
		os.Exit(1)
	}

	if len(preferences) == 0 {
		fmt.Println("No preferences recorded.")
		return
	}

	// Print table header
	fmt.Println("Timestamp  | Category    | Scope   | Key                  | Value                | Reason")
	fmt.Println("-----------|-------------|---------|----------------------|----------------------|-------------------")

	for _, p := range preferences {
		timestamp := time.Unix(p.Timestamp, 0).Format("2006-01-02")
		category := truncateForTable(p.Category, 11)
		scope := truncateForTable(p.Scope, 7)
		key := truncateForTable(p.Key, 20)
		value := truncateForTable(p.Value, 20)
		reason := truncateForTable(p.Reason, 17)
		fmt.Printf("%s | %-11s | %-7s | %-20s | %-20s | %s\n",
			timestamp, category, scope, key, value, reason)
	}

	fmt.Printf("\nTotal: %d preference(s)\n", len(preferences))
}

// showPerformance displays performance metrics with optional filtering
func showPerformance() {
	perfFlags := flag.NewFlagSet("performance", flag.ExitOnError)
	byOperationFlag := perfFlags.Bool("by-operation", false, "Group metrics by operation type (summary view)")
	slowOnlyFlag := perfFlags.Bool("slow-only", false, "Show only slow metrics (>1000ms)")
	sinceFlag := perfFlags.String("since", "", "Filter since duration (7d) or date (YYYY-MM-DD)")
	perfFlags.Parse(os.Args[2:])

	projectDir := getProjectDir()
	q := session.NewQuery(projectDir)

	// Build filters
	filters := session.PerformanceFilters{}
	if *slowOnlyFlag {
		filters.SlowOnly = true
	}
	if *sinceFlag != "" {
		since := parseSinceFilter(*sinceFlag)
		timestamp := since.Unix()
		filters.Since = &timestamp
	}

	if *byOperationFlag {
		// Show summary grouped by operation
		summaries, err := q.QueryPerformanceSummary(filters)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[gogent-archive] Failed to query performance summary: %v\n", err)
			os.Exit(1)
		}

		if len(summaries) == 0 {
			fmt.Println("No performance metrics recorded.")
			return
		}

		// Print summary header
		fmt.Println("Operation                      | Count | Success | Failed | Avg (ms) | Min (ms) | Max (ms)")
		fmt.Println("-------------------------------|-------|---------|--------|----------|----------|----------")

		var totalOps int
		var totalSuccess int
		var totalFailed int

		for _, s := range summaries {
			operation := truncateForTable(s.Operation, 30)
			fmt.Printf("%-30s | %5d | %7d | %6d | %8.1f | %8d | %8d\n",
				operation, s.Count, s.SuccessCount, s.FailCount, s.AvgMs, s.MinMs, s.MaxMs)
			totalOps += s.Count
			totalSuccess += s.SuccessCount
			totalFailed += s.FailCount
		}

		fmt.Printf("\nTotal: %d operation(s) (%d success, %d failed)\n", totalOps, totalSuccess, totalFailed)
		return
	}

	// Show raw metrics table
	metrics, err := q.QueryPerformance(filters)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-archive] Failed to query performance metrics: %v\n", err)
		os.Exit(1)
	}

	if len(metrics) == 0 {
		fmt.Println("No performance metrics recorded.")
		return
	}

	// Print table header
	fmt.Println("Timestamp  | Operation                      | Duration | Memory      | Success | Context")
	fmt.Println("-----------|--------------------------------|----------|-------------|---------|--------------------")

	for _, m := range metrics {
		timestamp := time.Unix(m.Timestamp, 0).Format("2006-01-02")
		operation := truncateForTable(m.Operation, 30)
		duration := fmt.Sprintf("%dms", m.DurationMs)
		memory := formatBytes(m.MemoryBytes)
		success := "Yes"
		if !m.Success {
			success = "No"
		}
		context := truncateForTable(m.Context, 18)
		fmt.Printf("%s | %-30s | %8s | %11s | %-7s | %s\n",
			timestamp, operation, duration, memory, success, context)
	}

	fmt.Printf("\nTotal: %d metric(s)\n", len(metrics))

	// Show quick stats
	var totalMs int64
	var successCount int
	for _, m := range metrics {
		totalMs += m.DurationMs
		if m.Success {
			successCount++
		}
	}
	avgMs := float64(totalMs) / float64(len(metrics))
	successRate := float64(successCount) / float64(len(metrics)) * 100
	fmt.Printf("Average duration: %.1fms | Success rate: %.1f%%\n", avgMs, successRate)
}

// generateWeeklySummary displays weekly intent summary with optional filters
func generateWeeklySummary() {
	weeklyFlags := flag.NewFlagSet("weekly", flag.ExitOnError)
	intentsOnlyFlag := weeklyFlags.Bool("intents-only", false, "Show only intent summary section")
	driftFlag := weeklyFlags.Bool("drift", false, "Highlight preference changes (drift alerts)")
	sinceFlag := weeklyFlags.String("since", "", "Start date for weekly range (YYYY-MM-DD, defaults to 7 days ago)")
	weeklyFlags.Parse(os.Args[2:])

	projectDir := getProjectDir()

	// Determine week range
	var weekStart, weekEnd time.Time
	if *sinceFlag != "" {
		parsedDate, err := time.Parse("2006-01-02", *sinceFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[gogent-archive] Invalid --since date format '%s'\n", *sinceFlag)
			fmt.Fprintln(os.Stderr, "  Expected YYYY-MM-DD format (e.g., '2026-01-15')")
			os.Exit(1)
		}
		weekStart = parsedDate
		weekEnd = weekStart.AddDate(0, 0, 7)
	} else {
		// Default: last 7 days
		now := time.Now()
		weekEnd = now
		weekStart = now.AddDate(0, 0, -7)
	}

	// Load all user intents
	intentsPath := filepath.Join(projectDir, ".claude", "memory", "user-intents.jsonl")
	intents, err := session.LoadAllUserIntents(intentsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-archive] Failed to load user intents: %v\n", err)
		os.Exit(1)
	}

	if len(intents) == 0 {
		fmt.Println("No user intents recorded for the specified period.")
		return
	}

	// Aggregate intents for current week
	currentSummary := session.AggregateWeeklyIntents(intents, weekStart, weekEnd)

	// If drift flag enabled, load previous week and detect drift
	if *driftFlag {
		prevWeekStart := weekStart.AddDate(0, 0, -7)
		prevWeekEnd := weekStart
		previousSummary := session.AggregateWeeklyIntents(intents, prevWeekStart, prevWeekEnd)
		currentSummary.DriftAlerts = session.DetectPreferenceDrift(currentSummary, previousSummary)
	}

	// Render output
	if *intentsOnlyFlag {
		// Only show intent section
		fmt.Print(session.FormatWeeklyIntentSummary(currentSummary))
	} else {
		// Show full weekly report with header
		fmt.Printf("# Weekly Summary - %s to %s\n\n",
			weekStart.Format("2006-01-02"),
			weekEnd.Format("2006-01-02"))
		fmt.Print(session.FormatWeeklyIntentSummary(currentSummary))
	}
}

// formatBytes formats byte count for human-readable display
func formatBytes(bytes int64) string {
	if bytes == 0 {
		return "-"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// truncateForTable truncates string for table display
func truncateForTable(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// parseSinceFilter parses duration (e.g., "7d") or date (YYYY-MM-DD) into time.Time
func parseSinceFilter(since string) time.Time {
	now := time.Now()

	// Try parsing as duration first (e.g., "7d", "30d")
	if strings.HasSuffix(since, "d") {
		daysStr := strings.TrimSuffix(since, "d")
		days, err := strconv.Atoi(daysStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[gogent-archive] Invalid --since format '%s'\n", since)
			fmt.Fprintln(os.Stderr, "  Use duration format (e.g., '7d', '30d') or date format (YYYY-MM-DD)")
			os.Exit(1)
		}
		return now.AddDate(0, 0, -days)
	}

	// Try parsing as date (YYYY-MM-DD)
	parsedDate, err := time.Parse("2006-01-02", since)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-archive] Invalid --since date format '%s'\n", since)
		fmt.Fprintln(os.Stderr, "  Expected YYYY-MM-DD format (e.g., '2026-01-15')")
		os.Exit(1)
	}
	return parsedDate
}
