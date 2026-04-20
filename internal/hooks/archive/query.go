package archive

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/session"
)

// listSharpEdges displays sharp edges with optional filtering.
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
		fmt.Fprintf(os.Stderr, "[goyoke-archive] Failed to query sharp edges: %v\n", err)
		os.Exit(1)
	}

	if len(edges) == 0 {
		fmt.Println("No sharp edges recorded.")
		return
	}

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

// listUserIntents displays user intents with optional filtering.
func listUserIntents() {
	intentsFlags := flag.NewFlagSet("user-intents", flag.ExitOnError)
	sourceFlag := intentsFlags.String("source", "", "Filter by source (ask_user, hook_prompt, manual)")
	confidenceFlag := intentsFlags.String("confidence", "", "Filter by confidence (explicit, inferred, default)")
	hasActionFlag := intentsFlags.Bool("has-action", false, "Show only intents with actions taken")
	sinceFlag := intentsFlags.String("since", "", "Filter since duration (7d) or date (YYYY-MM-DD)")
	categoryFlag := intentsFlags.String("category", "", "Filter by category (routing, tooling, style, etc.)")
	keywordFlag := intentsFlags.String("keyword", "", "Filter by keyword")
	honoredFlag := intentsFlags.String("honored", "", "Filter by honored status (true/false)")
	intentsFlags.Parse(os.Args[2:])

	projectDir := getProjectDir()
	q := session.NewQuery(projectDir)

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
	if *categoryFlag != "" {
		filters.Category = categoryFlag
	}
	if *keywordFlag != "" {
		filters.Keywords = []string{*keywordFlag}
	}
	if *honoredFlag != "" {
		honored := *honoredFlag == "true"
		filters.Honored = &honored
	}

	intents, err := q.QueryUserIntents(filters)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-archive] Failed to query user intents: %v\n", err)
		os.Exit(1)
	}

	if len(intents) == 0 {
		fmt.Println("No user intents recorded.")
		return
	}

	fmt.Println("Timestamp  | Category   | Honored | Source      | Question                     | Response")
	fmt.Println("-----------|------------|---------|-------------|------------------------------|---------------------------")

	for _, intent := range intents {
		timestamp := time.Unix(intent.Timestamp, 0).Format("2006-01-02")
		category := truncateForTable(intent.Category, 10)
		if category == "" {
			category = "-"
		}
		honored := "?"
		if intent.Honored != nil {
			if *intent.Honored {
				honored = "Yes"
			} else {
				honored = "No"
			}
		}
		source := truncateForTable(intent.Source, 11)
		question := truncateForTable(intent.Question, 28)
		response := truncateForTable(intent.Response, 25)
		fmt.Printf("%s | %-10s | %-7s | %-11s | %-28s | %s\n",
			timestamp, category, honored, source, question, response)
	}

	fmt.Printf("\nTotal: %d user intent(s)\n", len(intents))
}

// listDecisions displays decisions with optional filtering.
func listDecisions() {
	decisionsFlags := flag.NewFlagSet("decisions", flag.ExitOnError)
	categoryFlag := decisionsFlags.String("category", "", "Filter by category (architecture, tooling, pattern)")
	impactFlag := decisionsFlags.String("impact", "", "Filter by impact level (high, medium, low)")
	sinceFlag := decisionsFlags.String("since", "", "Filter since duration (7d) or date (YYYY-MM-DD)")
	decisionsFlags.Parse(os.Args[2:])

	projectDir := getProjectDir()
	q := session.NewQuery(projectDir)

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
		fmt.Fprintf(os.Stderr, "[goyoke-archive] Failed to query decisions: %v\n", err)
		os.Exit(1)
	}

	if len(decisions) == 0 {
		fmt.Println("No decisions recorded.")
		return
	}

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

// listPreferences displays preference overrides with optional filtering.
func listPreferences() {
	prefsFlags := flag.NewFlagSet("preferences", flag.ExitOnError)
	categoryFlag := prefsFlags.String("category", "", "Filter by category (routing, tooling, formatting)")
	scopeFlag := prefsFlags.String("scope", "", "Filter by scope (session, project, global)")
	sinceFlag := prefsFlags.String("since", "", "Filter since duration (7d) or date (YYYY-MM-DD)")
	prefsFlags.Parse(os.Args[2:])

	projectDir := getProjectDir()
	q := session.NewQuery(projectDir)

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
		fmt.Fprintf(os.Stderr, "[goyoke-archive] Failed to query preferences: %v\n", err)
		os.Exit(1)
	}

	if len(preferences) == 0 {
		fmt.Println("No preferences recorded.")
		return
	}

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
