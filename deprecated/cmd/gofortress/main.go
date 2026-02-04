package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/deprecated/internal/callback"
	"github.com/Bucket-Chemist/GOgent-Fortress/deprecated/internal/cli"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/lifecycle"
	"github.com/Bucket-Chemist/GOgent-Fortress/deprecated/internal/mcp"
	"github.com/Bucket-Chemist/GOgent-Fortress/deprecated/internal/tui/agents"
	"github.com/Bucket-Chemist/GOgent-Fortress/deprecated/internal/tui/claude"
	"github.com/Bucket-Chemist/GOgent-Fortress/deprecated/internal/tui/layout"
)

const (
	version = "0.1.0"
)

var (
	sessionID  = flag.String("session", "", "Session ID to resume (default: generate new)")
	sessionShort = flag.String("s", "", "Session ID to resume (shorthand)")
	listSessions = flag.Bool("list", false, "List recent sessions and exit")
	listShort    = flag.Bool("l", false, "List recent sessions and exit (shorthand)")
	workingDir   = flag.String("working-dir", "", "Working directory for claude process (default: current directory)")
	workingDirShort = flag.String("w", "", "Working directory (shorthand)")
	verbose      = flag.Bool("verbose", false, "Enable verbose output from claude process")
	showVersion  = flag.Bool("version", false, "Show version and exit")
	showVersionShort = flag.Bool("v", false, "Show version (shorthand)")
)

func main() {
	flag.Parse()

	// Handle version flag
	if *showVersion || *showVersionShort {
		fmt.Printf("gofortress %s\n", version)
		os.Exit(0)
	}

	// Resolve shorthand flags
	sessionToResume := *sessionID
	if sessionToResume == "" && *sessionShort != "" {
		sessionToResume = *sessionShort
	}

	workDir := *workingDir
	if workDir == "" && *workingDirShort != "" {
		workDir = *workingDirShort
	}

	listMode := *listSessions || *listShort

	// Handle list mode
	if listMode {
		if err := listRecentSessions(); err != nil {
			fmt.Fprintf(os.Stderr, "Error listing sessions: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// CRITICAL: Clean up stale sockets from previous crashed sessions
	// Must run BEFORE creating new socket to prevent "address in use" errors
	if err := lifecycle.CleanupStaleSockets(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: stale socket cleanup failed: %v\n", err)
	}

	// Start callback server for MCP integration
	pid := os.Getpid()
	callbackServer := callback.NewServer(pid)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// CRITICAL: Set up process lifecycle manager for signal handling
	processManager := lifecycle.NewProcessManager(callbackServer.SocketPath())
	processManager.StartSignalHandler(ctx, func() {
		cancel() // Cancel context to unblock listeners
		callbackServer.Shutdown(context.Background())
	})

	var mcpConfigPath string
	var mcpEnabled bool

	if err := callbackServer.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: MCP callback server failed: %v\n", err)
		fmt.Fprintf(os.Stderr, "         Interactive prompts will be disabled.\n")
	} else {
		mcpEnabled = true
		defer callbackServer.Cleanup()
		defer callbackServer.Shutdown(ctx)

		// Find MCP server binary
		serverBinary, err := mcp.FindServerBinary()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			fmt.Fprintf(os.Stderr, "         Interactive prompts will be disabled.\n")
			mcpEnabled = false
		} else {
			// Generate MCP config
			mcpConfigPath, err = mcp.GenerateConfig(pid, callbackServer.SocketPath(), serverBinary)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
				mcpEnabled = false
			} else {
				defer mcp.Cleanup(mcpConfigPath)
			}
		}
	}

	// Create session manager
	sessionMgr, err := cli.NewSessionManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating session manager: %v\n", err)
		os.Exit(1)
	}

	// Create or resume Claude process
	var process *cli.ClaudeProcess
	var cfg cli.Config

	// Build base config with MCP if enabled
	baseAllowedTools := []string{"Bash", "Read", "Write", "Edit", "Glob", "Grep", "Task", "TaskOutput", "EnterPlanMode", "ExitPlanMode"}
	if mcpEnabled {
		baseAllowedTools = append(baseAllowedTools,
			"mcp__gofortress__ask_user",
			"mcp__gofortress__confirm_action",
			"mcp__gofortress__request_input",
			"mcp__gofortress__select_option",
		)
	}

	if sessionToResume != "" {
		// Resume existing session
		fmt.Printf("Resuming session: %s\n", sessionToResume)
		process, err = sessionMgr.ResumeSession(sessionToResume)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resuming session: %v\n", err)
			os.Exit(1)
		}
		// Create a config for resumed session (we don't have the original config)
		// This is needed for potential model changes
		cfg = cli.Config{
			ClaudePath:    "claude",
			SessionID:     sessionToResume,
			WorkingDir:    workDir,
			Verbose:       *verbose,
			AllowedTools:  baseAllowedTools,
			MCPConfigPath: mcpConfigPath,
		}
	} else {
		// Create new session
		cfg = cli.Config{
			ClaudePath:   "claude",
			SessionID:    "", // Will be generated
			WorkingDir:   workDir,
			Verbose:      *verbose,
			// Pre-approve common tools to avoid permission dialogs.
			// Based on testing, permission-mode flags don't enable interactive permissions in stream-json mode.
			// The "delegate" mode still sends error events, not request events.
			// Solution: Pre-approve tools via AllowedTools.
			AllowedTools:  baseAllowedTools,
			MCPConfigPath: mcpConfigPath,
		}

		process, err = cli.NewClaudeProcess(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Claude process: %v\n", err)
			os.Exit(1)
		}

		if err := process.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting Claude process: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Started new session: %s\n", process.SessionID())
	}

	// Ensure process is stopped on exit
	defer func() {
		if err := process.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "Error stopping process: %v\n", err)
		}
	}()

	// Create agent tree for the session
	tree := agents.NewAgentTree(process.SessionID())

	// Create TUI components with callback server if enabled
	var claudePanel claude.PanelModel
	if mcpEnabled {
		claudePanel = claude.NewPanelModelWithCallback(ctx, process, cfg, callbackServer)
	} else {
		claudePanel = claude.NewPanelModel(process, cfg)
	}

	// CRITICAL: Register Claude process with lifecycle manager for signal propagation
	// This ensures SIGTERM is forwarded to Claude if gofortress is killed
	if claudeProcess := process.GetProcess(); claudeProcess != nil {
		processManager.SetChildProcess(claudeProcess)
	}

	agentTreeView := agents.New(tree)
	layoutModel := layout.NewModel(claudePanel, agentTreeView, process.SessionID())

	// Run TUI
	// Note: Alt screen removed to allow terminal text selection (copy/paste)
	// Without alt screen, terminal scrollback and selection work normally
	p := tea.NewProgram(
		layoutModel,
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

// listRecentSessions displays recent sessions in a formatted table
func listRecentSessions() error {
	sessionMgr, err := cli.NewSessionManager()
	if err != nil {
		return fmt.Errorf("create session manager: %w", err)
	}

	sessions, err := sessionMgr.ListSessions()
	if err != nil {
		return fmt.Errorf("list sessions: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}

	// Create tabwriter for aligned columns
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SESSION ID\tNAME\tLAST USED\tCOST\tTOOL CALLS")
	fmt.Fprintln(w, "----------\t----\t---------\t----\t----------")

	for _, session := range sessions {
		name := session.Name
		if name == "" {
			name = "-"
		}

		// Format last used time
		lastUsed := formatTimeSince(session.LastUsed)

		fmt.Fprintf(w, "%s\t%s\t%s\t$%.2f\t%d\n",
			truncate(session.ID, 12),
			name,
			lastUsed,
			session.Cost,
			session.ToolCalls,
		)
	}

	return w.Flush()
}

// formatTimeSince formats a time as "2h ago", "3d ago", etc.
func formatTimeSince(t time.Time) string {
	duration := time.Since(t)

	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		mins := int(duration.Minutes())
		return fmt.Sprintf("%dm ago", mins)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		return fmt.Sprintf("%dh ago", hours)
	case duration < 7*24*time.Hour:
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	default:
		weeks := int(duration.Hours() / 24 / 7)
		return fmt.Sprintf("%dw ago", weeks)
	}
}

// truncate truncates a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
