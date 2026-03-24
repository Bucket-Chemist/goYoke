// Package main is the entry point for the gofortress TUI binary.
//
// Usage:
//
//	gofortress [flags]
//
// Flags:
//
//	--session-id      Resume a specific session by ID
//	--verbose         Enable verbose logging to stderr
//	--debug           Write all tea.Msg values to a debug log file
//	--model           Initial model override (e.g. "claude-opus-4-6")
//	--permission-mode Initial permission mode: default, acceptEdits, plan
//	--version         Print version and exit
//	--mcp-server      Run MCP server mode (redirects to gofortress-mcp)
//	--config-dir      Override Claude config directory (e.g. ~/.claude-em)
//	--resume          Resume the most recent session
//
// Version is injected at build time:
//
//	go build -ldflags "-X main.version=v1.0.0" ./cmd/gofortress/...
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/lifecycle"
	tuiLifecycle "github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/lifecycle"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/bridge"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/cli"
	claudepkg "github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/claude"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/dashboard"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/planpreview"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/providers"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/settings"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/tabbar"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/taskboard"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/teams"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/telemetry"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/toast"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/model"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/session"
)

// version is injected at build time via:
//
//	-ldflags "-X main.version=v1.0.0"
var version = "dev"

func main() {
	sessionID := flag.String("session-id", "", "resume a specific session by ID")
	verbose := flag.Bool("verbose", false, "enable verbose logging to stderr")
	debug := flag.Bool("debug", false, "write all tea.Msg values to a debug log file")
	modelOverride := flag.String("model", "", "initial model override (e.g. claude-opus-4-6)")
	permMode := flag.String("permission-mode", "default", "initial permission mode: default, acceptEdits, plan")
	printVersion := flag.Bool("version", false, "print version and exit")
	mcpServer := flag.Bool("mcp-server", false, "run MCP server mode instead of TUI (use gofortress-mcp binary)")
	configDir := flag.String("config-dir", "", "override Claude config directory (e.g. ~/.claude-em)")
	resume := flag.Bool("resume", false, "resume most recent session")

	flag.Parse()

	if *printVersion {
		fmt.Printf("gofortress version %s\n", version)
		os.Exit(0)
	}

	if *mcpServer {
		fmt.Fprintln(os.Stderr, "MCP server mode not yet implemented. Use gofortress-mcp binary.")
		os.Exit(1)
	}

	// If --config-dir is set, propagate to environment so the claude subprocess
	// and session store both use the correct config directory.
	if *configDir != "" {
		if err := os.Setenv("CLAUDE_CONFIG_DIR", *configDir); err != nil {
			log.Printf("[gofortress] warning: could not set CLAUDE_CONFIG_DIR: %v", err)
		}
	}

	// Clean up any socket files left by crashed sessions.
	if err := lifecycle.CleanupStaleSockets(); err != nil {
		if *verbose {
			log.Printf("[gofortress] cleanup stale sockets: %v", err)
		}
	}

	// -----------------------------------------------------------------------
	// Phase 1: Build root model, wire tab bar, wire CLI driver.
	//
	// The CLI driver is wired before tea.NewProgram so that Init() can
	// schedule the Start() command immediately.  Both the local app variable
	// and the model copy inside the program share the same sharedState pointer,
	// so SetCLIDriver / SetBridge written here are visible to Update().
	// -----------------------------------------------------------------------

	app := model.NewAppModel()
	keys := config.DefaultKeyMap()
	tb := tabbar.NewTabBarModel(keys, 0)
	app.SetTabBar(&tb)

	// Wire Phase 6 child components into shared state.
	cp := claudepkg.NewClaudePanelModel(keys)
	app.SetClaudePanel(&cp)

	toastModel := toast.NewToastModel()
	app.SetToasts(&toastModel)

	teamReg := teams.NewTeamRegistry()
	teamList := teams.NewTeamListModel(teamReg)
	app.SetTeamList(&teamList)

	// Phase 7: Provider tab bar.
	// ProviderState is created inside NewAppModel() and stored in shared state.
	// Retrieve it via the getter so both the tab bar and the model reference
	// the same pointer; this avoids duplicating the four-provider configuration.
	ps := app.ProviderState()
	ptb := providers.NewProviderTabBarModel(ps, 0)
	app.SetProviderTabBar(&ptb)

	// -----------------------------------------------------------------------
	// Phase 7b: Session persistence (TUI-033).
	//
	// If --session-id is provided, load the existing session and restore
	// provider state (session IDs, model selections, active provider).
	// Otherwise, generate a fresh session ID, set up the session directory
	// (including .claude/current-session and .claude/tmp symlink), and
	// create initial SessionData.
	// -----------------------------------------------------------------------

	sessionStore := session.NewStore("")

	// If --resume is set and no explicit --session-id was given, resolve the
	// most recent session from the store and treat it as the target session ID.
	if *resume && *sessionID == "" {
		sessions, err := sessionStore.ListSessions()
		if err != nil {
			log.Printf("[gofortress] warning: could not list sessions for --resume: %v", err)
		} else if len(sessions) > 0 {
			*sessionID = sessions[0].ID
			if *verbose {
				log.Printf("[gofortress] --resume: selected session %s (last used %s)",
					sessions[0].ID, sessions[0].LastUsed.Format(time.RFC3339))
			}
		} else if *verbose {
			log.Printf("[gofortress] --resume: no existing sessions found, starting new session")
		}
	}

	var sessionData *session.SessionData
	if *sessionID != "" {
		sd, err := sessionStore.LoadSession(*sessionID)
		if err != nil {
			log.Printf("[gofortress] warning: could not load session %q: %v", *sessionID, err)
		}
		if sd != nil {
			sessionData = sd
			// Restore provider state from persisted data.
			if ps != nil {
				ps.ImportSessionIDs(sd.ProviderSessionIDs)
				ps.ImportModels(sd.ProviderModels)
				if sd.ActiveProvider != "" {
					_ = ps.SwitchProvider(sd.ActiveProvider)
				}
			}
			if *verbose {
				log.Printf("[gofortress] resumed session %s (cost=$%.4f, providers=%d)",
					sd.ID, sd.Cost, len(sd.ProviderSessionIDs))
			}
		}
	}

	if sessionData == nil {
		newID := session.NewSessionID()
		sessionData = &session.SessionData{
			ID:             newID,
			CreatedAt:      time.Now(),
			LastUsed:       time.Now(),
			ActiveProvider: "anthropic",
		}
		if _, err := sessionStore.SetupSessionDir(newID); err != nil {
			log.Printf("[gofortress] warning: could not setup session dir: %v", err)
		}
		if *verbose {
			log.Printf("[gofortress] new session %s", newID)
		}
	}

	app.SetSessionStore(sessionStore)
	app.SetSessionData(sessionData)

	// Phase 7: Right-panel components.
	dashModel := dashboard.NewDashboardModel()
	app.SetDashboard(&dashModel)

	settingsModel := settings.NewSettingsModel()
	app.SetSettings(&settingsModel)

	telModel := telemetry.NewTelemetryModel()
	app.SetTelemetry(&telModel)

	ppModel := planpreview.NewPlanPreviewModel()
	app.SetPlanPreview(&ppModel)

	tbModel := taskboard.NewTaskBoardModel()
	app.SetTaskBoard(&tbModel)

	// Build the CLI driver options from flags.
	cliOpts := cli.CLIDriverOpts{
		SessionID:      *sessionID,
		Model:          *modelOverride,
		ProjectDir:     ".", // current working directory
		PermissionMode: *permMode,
		Verbose:        *verbose,
		Debug:          *debug,
		ConfigDir:      *configDir,
	}
	driver := cli.NewCLIDriver(cliOpts)
	app.SetCLIDriver(driver)
	app.SetBaseCLIOpts(cliOpts) // C-2: preserve flags for provider-switch reconstructions
	cp.SetSender(driver)        // driver satisfies model.MessageSender

	// Wire initial settings data now that cliOpts is populated.
	settingsModel.SetConfig(
		cliOpts.Model,
		"Anthropic",
		cliOpts.PermissionMode,
		cliOpts.ProjectDir,
		[]string{"gofortress"},
	)

	// -----------------------------------------------------------------------
	// Phase 2: Create tea.Program (copies app by value, but shared pointer
	// remains shared — subsequent SetBridge call is visible to both).
	// -----------------------------------------------------------------------

	opts := []tea.ProgramOption{
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	}

	if *debug {
		f, err := openDebugLog()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[gofortress] warning: could not open debug log: %v\n", err)
		} else {
			defer f.Close()
			opts = append(opts, tea.WithInput(os.Stdin))
			log.SetOutput(f)
			log.SetFlags(log.Ltime | log.Lmicroseconds)
			if *verbose {
				log.Printf("[gofortress] debug log opened: %s", f.Name())
			}
		}
	}

	p := tea.NewProgram(app, opts...)

	// -----------------------------------------------------------------------
	// Phase 3: Create IPC bridge with the program as sender, wire into model,
	// expose socket path to child processes via environment variable.
	// -----------------------------------------------------------------------

	ipcBridge, err := bridge.NewIPCBridge(p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[gofortress] bridge error: %v\n", err)
		os.Exit(1)
	}
	app.SetBridge(ipcBridge)
	ipcBridge.Start()

	// Expose the UDS path so the MCP server subprocess can connect.
	if err := os.Setenv("GOFORTRESS_SOCKET", ipcBridge.SocketPath()); err != nil {
		if *verbose {
			log.Printf("[gofortress] warning: could not set GOFORTRESS_SOCKET: %v", err)
		}
	}

	// -----------------------------------------------------------------------
	// Phase 4: Create ShutdownManager for sequenced shutdown (TUI-034).
	//
	// Replaces the previous defer-based LIFO ordering which was wrong:
	// bridge.Shutdown ran BEFORE driver.Shutdown. Correct order is driver
	// first, then bridge (so IPC messages flow during CLI wind-down).
	//
	// The session saver captures the AppModel's saveSession via closure.
	// -----------------------------------------------------------------------

	sm := tuiLifecycle.NewShutdownManager(tuiLifecycle.ShutdownOpts{
		Driver:       driver,
		Bridge:       ipcBridge,
		SessionSaver: func() { app.SaveSessionPublic() },
		OnStatus: func(msg string) {
			if *verbose {
				log.Printf("[gofortress] %s", msg)
			}
		},
	})

	// Wire ShutdownManager into the model so double-Ctrl+C and tea.Quit
	// trigger the sequenced shutdown instead of the raw defer path.
	app.SetShutdownManager(sm)

	// Wire OS-level signal handler (SIGINT/SIGTERM) via ProcessManager.
	// This ensures signals caught at the process level also trigger
	// graceful shutdown, not just Bubbletea key events.
	procMgr := lifecycle.NewProcessManager(ipcBridge.SocketPath())
	procMgr.StartSignalHandler(context.Background(), func() {
		if *verbose {
			log.Printf("[gofortress] OS signal received, initiating shutdown")
		}
		_ = sm.Shutdown()
	})

	// -----------------------------------------------------------------------
	// Run — blocks until the user quits.
	// -----------------------------------------------------------------------

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "[gofortress] error: %v\n", err)
		os.Exit(1)
	}

	// Post-run: execute sequenced shutdown if not already done (e.g. user
	// quit via menu rather than Ctrl+C). The double-call guard in
	// ShutdownManager ensures this is a no-op if already shut down.
	if err := sm.Shutdown(); err != nil {
		if *verbose {
			log.Printf("[gofortress] shutdown warning: %v", err)
		}
	}
}

// openDebugLog creates a timestamped debug log file in the system temp
// directory and returns the open file.  The caller is responsible for
// closing it.
func openDebugLog() (*os.File, error) {
	name := fmt.Sprintf("gofortress-debug-%s.log", time.Now().Format("20060102-150405"))
	path := filepath.Join(os.TempDir(), name)
	return os.Create(path)
}
