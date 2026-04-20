// Package archive implements the goyoke-archive SessionEnd hook.
package archive

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/config"
	"github.com/Bucket-Chemist/goYoke/pkg/session"
)

const defaultTimeout = 5 * time.Second

// Main is the entrypoint for the goyoke-archive hook.
func Main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "list":
			listSessions()
			return
		case "show":
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "[goyoke-archive] Usage: goyoke-archive show <session-id>")
				fmt.Fprintln(os.Stderr, "  Missing required argument: session-id")
				fmt.Fprintln(os.Stderr, "  Example: goyoke-archive show abc123def456")
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
			fmt.Printf("goyoke-archive version %s\n", getVersion())
			return
		}
	}

	if err := run(); err != nil {
		outputError(err.Error())
		os.Exit(1)
	}
}

func run() error {
	projectDir := os.Getenv("GOYOKE_PROJECT_DIR")
	if projectDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("[goyoke-archive] Failed to get working directory: %w. Set GOYOKE_PROJECT_DIR environment variable or run from project root.", err)
		}
		projectDir = cwd
	}

	event, err := session.ParseSessionEvent(os.Stdin, defaultTimeout)
	if err != nil {
		return fmt.Errorf("[goyoke-archive] Failed to parse SessionEnd event: %w. Ensure hook provides valid JSON on STDIN.", err)
	}

	metrics, err := session.CollectSessionMetrics(event.SessionID)
	if err != nil {
		return fmt.Errorf("[goyoke-archive] Failed to collect metrics for session %s: %w. Check temp files exist and are readable.", event.SessionID, err)
	}

	handoffCfg := session.DefaultHandoffConfig(projectDir)
	handoff, hMetrics, err := session.GenerateHandoff(handoffCfg, metrics)
	if err != nil {
		return fmt.Errorf("[goyoke-archive] Failed to generate handoff: %w", err)
	}

	if handoff == nil {
		return fmt.Errorf("[goyoke-archive] No handoff data generated. This may be normal for first session. Cannot generate markdown for empty handoff.")
	}

	_ = hMetrics

	mdPath := filepath.Join(config.ProjectMemoryDir(projectDir), "last-handoff.md")
	if err := os.MkdirAll(filepath.Dir(mdPath), 0755); err != nil {
		return fmt.Errorf("[goyoke-archive] Failed to create directory for %s: %w", mdPath, err)
	}
	markdown := session.RenderHandoffMarkdown(handoff)
	if err := os.WriteFile(mdPath, []byte(markdown), 0644); err != nil {
		return fmt.Errorf("[goyoke-archive] Failed to write markdown to %s: %w", mdPath, err)
	}

	if err := session.ArchiveArtifacts(*handoffCfg, event.SessionID); err != nil {
		return fmt.Errorf("[goyoke-archive] Failed to archive artifacts: %w", err)
	}

	cleanupPermCache(event.SessionID)
	cleanupSkillGuard(event.SessionID)

	if err := analyzeAndUpdateIntentOutcomes(projectDir, event.SessionID); err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke-archive] Warning: Failed to analyze intent outcomes: %v\n", err)
	}

	fmt.Fprintf(os.Stderr, "[goyoke-archive] 📦 SESSION ARCHIVED: %s (tools=%d, errors=%d, violations=%d)\n",
		event.SessionID, metrics.ToolCalls, metrics.ErrorsLogged, metrics.RoutingViolations)

	fmt.Println("{}")

	return nil
}

// outputError writes error message to stderr and outputs empty JSON.
func outputError(message string) {
	fmt.Fprintf(os.Stderr, "[goyoke-archive] 🔴 %s\n", message)
	fmt.Println("{}")
}

func getVersion() string {
	version := "dev"
	return version
}

// printHelp displays usage information for all subcommands.
func printHelp() {
	fmt.Println("goyoke-archive - Session handoff archival and querying")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  goyoke-archive                       Read SessionEnd JSON from STDIN (hook mode)")
	fmt.Println("")
	fmt.Println("Session Commands:")
	fmt.Println("  goyoke-archive list                  List all sessions")
	fmt.Println("  goyoke-archive list --since 7d       List sessions from last 7 days")
	fmt.Println("  goyoke-archive list --between <dates> List sessions between dates (YYYY-MM-DD,YYYY-MM-DD)")
	fmt.Println("  goyoke-archive list --has-sharp-edges Show only sessions with sharp edges")
	fmt.Println("  goyoke-archive list --has-violations Show only sessions with routing violations")
	fmt.Println("  goyoke-archive list --clean          Show only clean sessions (no errors/violations)")
	fmt.Println("  goyoke-archive show <id>             Show specific session handoff")
	fmt.Println("  goyoke-archive stats                 Show aggregate statistics with breakdowns")
	fmt.Println("")
	fmt.Println("Weekly Analysis Commands:")
	fmt.Println("  goyoke-archive weekly                Generate weekly intent summary (last 7 days)")
	fmt.Println("  goyoke-archive weekly --since <date> Generate summary from specific start date (YYYY-MM-DD)")
	fmt.Println("  goyoke-archive weekly --intents-only Show only intent section")
	fmt.Println("  goyoke-archive weekly --drift        Show preference changes (drift alerts)")
	fmt.Println("")
	fmt.Println("Sharp Edge Commands:")
	fmt.Println("  goyoke-archive sharp-edges           List all sharp edges")
	fmt.Println("  goyoke-archive sharp-edges --severity high  Filter by severity (high, medium, low)")
	fmt.Println("  goyoke-archive sharp-edges --file 'pkg/*'   Filter by file pattern (glob)")
	fmt.Println("  goyoke-archive sharp-edges --error-type <type> Filter by error type")
	fmt.Println("  goyoke-archive sharp-edges --unresolved     Show only unresolved edges")
	fmt.Println("  goyoke-archive sharp-edges --since 7d       Filter by time")
	fmt.Println("")
	fmt.Println("User Intent Commands:")
	fmt.Println("  goyoke-archive user-intents          List all user intents")
	fmt.Println("  goyoke-archive user-intents --source ask_user  Filter by source (ask_user, hook_prompt, manual)")
	fmt.Println("  goyoke-archive user-intents --confidence explicit  Filter by confidence (explicit, inferred, default)")
	fmt.Println("  goyoke-archive user-intents --category routing  Filter by category (routing, tooling, style, etc.)")
	fmt.Println("  goyoke-archive user-intents --keyword pytest    Filter by keyword")
	fmt.Println("  goyoke-archive user-intents --has-action    Show only intents with actions taken")
	fmt.Println("  goyoke-archive user-intents --honored true  Filter by honored status (true/false)")
	fmt.Println("  goyoke-archive user-intents --since 7d      Filter by time")
	fmt.Println("")
	fmt.Println("Decision Commands:")
	fmt.Println("  goyoke-archive decisions              List all decisions")
	fmt.Println("  goyoke-archive decisions --category architecture  Filter by category (architecture, tooling, pattern)")
	fmt.Println("  goyoke-archive decisions --impact high      Filter by impact level (high, medium, low)")
	fmt.Println("  goyoke-archive decisions --since 7d         Filter by time")
	fmt.Println("")
	fmt.Println("Preference Commands:")
	fmt.Println("  goyoke-archive preferences            List all preference overrides")
	fmt.Println("  goyoke-archive preferences --category routing  Filter by category (routing, tooling, formatting)")
	fmt.Println("  goyoke-archive preferences --scope project   Filter by scope (session, project, global)")
	fmt.Println("  goyoke-archive preferences --since 7d        Filter by time")
	fmt.Println("")
	fmt.Println("Performance Commands:")
	fmt.Println("  goyoke-archive performance            List all performance metrics")
	fmt.Println("  goyoke-archive performance --by-operation    Group metrics by operation (summary view)")
	fmt.Println("  goyoke-archive performance --slow-only       Show only slow operations (>1000ms)")
	fmt.Println("  goyoke-archive performance --since 7d        Filter by time")
	fmt.Println("")
	fmt.Println("Other Commands:")
	fmt.Println("  goyoke-archive --help                Show this help")
	fmt.Println("  goyoke-archive --version             Show version information")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  goyoke-archive sharp-edges --severity high --unresolved")
	fmt.Println("  goyoke-archive user-intents --source ask_user --has-action")
	fmt.Println("  goyoke-archive decisions --category architecture --impact high")
	fmt.Println("  goyoke-archive preferences --scope project")
	fmt.Println("  goyoke-archive performance --by-operation --slow-only")
	fmt.Println("  goyoke-archive list --since 2026-01-15 --clean")
	fmt.Println("  goyoke-archive show abc123def456")
	fmt.Println("")
	fmt.Println("For subcommand-specific help, use: goyoke-archive <subcommand> --help")
}
