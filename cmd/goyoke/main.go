// Package main is the entry point for the goyoke TUI binary.
//
// Usage:
//
//	goyoke [flags]
//
// Flags:
//
//	--session-id      Resume a specific session by ID
//	--verbose         Enable verbose logging to stderr
//	--debug           Write all tea.Msg values to a debug log file
//	--model           Initial model override (e.g. "claude-opus-4-6")
//	--permission-mode Initial permission mode: default, acceptEdits, plan
//	--version         Print version and exit
//	--mcp-server      Run MCP server mode (redirects to goyoke-mcp)
//	--config-dir      Override Claude config directory (e.g. ~/.claude-em)
//	--resume          Resume the most recent session
//
// Version is injected at build time:
//
//	go build -ldflags "-X main.version=v1.0.0" ./cmd/goyoke/...
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/goYoke/defaults"
	"github.com/Bucket-Chemist/goYoke/internal/hooks"
	"github.com/Bucket-Chemist/goYoke/internal/lifecycle"
	"github.com/Bucket-Chemist/goYoke/internal/subcmd"
	mcpcmd "github.com/Bucket-Chemist/goYoke/internal/subcmd/mcp"
	"github.com/Bucket-Chemist/goYoke/internal/subcmd/utils"
	tuiLifecycle "github.com/Bucket-Chemist/goYoke/internal/tui/lifecycle"
	"github.com/Bucket-Chemist/goYoke/internal/tui/bridge"
	"github.com/Bucket-Chemist/goYoke/internal/tui/cli"
	claudepkg "github.com/Bucket-Chemist/goYoke/internal/tui/components/claude"
	"github.com/Bucket-Chemist/goYoke/internal/tui/components/cwdselector"
	"github.com/Bucket-Chemist/goYoke/internal/tui/components/dashboard"
	"github.com/Bucket-Chemist/goYoke/internal/tui/components/drawer"
	"github.com/Bucket-Chemist/goYoke/internal/tui/components/planpreview"
	"github.com/Bucket-Chemist/goYoke/internal/tui/components/providers"
	"github.com/Bucket-Chemist/goYoke/internal/tui/components/settings"
	"github.com/Bucket-Chemist/goYoke/internal/tui/components/tabbar"
	"github.com/Bucket-Chemist/goYoke/internal/tui/components/taskboard"
	"github.com/Bucket-Chemist/goYoke/internal/tui/components/teams"
	"github.com/Bucket-Chemist/goYoke/internal/tui/components/telemetry"
	"github.com/Bucket-Chemist/goYoke/internal/tui/components/toast"
	"github.com/Bucket-Chemist/goYoke/internal/tui/config"
	"github.com/Bucket-Chemist/goYoke/internal/tui/model"
	"github.com/Bucket-Chemist/goYoke/internal/tui/session"
	"github.com/Bucket-Chemist/goYoke/pkg/resolve"
)

// version is injected at build time via:
//
//	-ldflags "-X main.version=v1.0.0"
var version = "dev"

func main() {
	resolve.SetDefault(defaults.FS)

	// Multicall dispatch: check argv[0] for busybox-style symlink invocation,
	// then check for explicit subcommands. Both must run before flag.Parse()
	// so subcommand names are not misinterpreted as unknown flags.
	reg := buildRegistry()
	if fn, remainingArgs, ok := subcmd.DispatchByArgv0(os.Args[0], reg); ok {
		if err := fn(context.Background(), remainingArgs, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) > 1 {
		err := reg.Dispatch(context.Background(), os.Args[1:], os.Stdin, os.Stdout)
		if err == nil {
			return
		}
		if !errors.Is(err, subcmd.ErrUnknownCommand) && !errors.Is(err, subcmd.ErrNoCommand) {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		// ErrUnknownCommand / ErrNoCommand → fall through to flag parsing + TUI
	}

	sessionID := flag.String("session-id", "", "resume a specific session by ID")
	verbose := flag.Bool("verbose", false, "enable verbose logging to stderr")
	debug := flag.Bool("debug", false, "write all tea.Msg values to a debug log file")
	modelOverride := flag.String("model", "", "initial model override (e.g. claude-opus-4-6)")
	permMode := flag.String("permission-mode", "acceptEdits", "initial permission mode: default, acceptEdits, plan")
	printVersion := flag.Bool("version", false, "print version and exit")
	mcpServer := flag.Bool("mcp-server", false, "run MCP server mode instead of TUI (use goyoke-mcp binary)")
	configDir := flag.String("config-dir", "", "override Claude config directory (e.g. ~/.claude-em)")
	resume := flag.Bool("resume", false, "resume most recent session")
	mcpBinaryFlag := flag.String("mcp-binary", "", "explicit path to goyoke-mcp binary (overrides auto-discovery)")

	flag.Parse()

	if *printVersion {
		fmt.Printf("goyoke version %s\n", version)
		os.Exit(0)
	}

	if *mcpServer {
		if err := mcpcmd.Run(context.Background(), nil, os.Stdin, os.Stdout); err != nil {
			log.Fatal(err)
		}
		return
	}

	// If --config-dir is set, propagate to environment so the claude subprocess
	// and session store both use the correct config directory.
	if *configDir != "" {
		if err := os.Setenv("CLAUDE_CONFIG_DIR", *configDir); err != nil {
			log.Printf("[goyoke] warning: could not set CLAUDE_CONFIG_DIR: %v", err)
		}
	}

	// Clean up any socket files left by crashed sessions.
	if err := lifecycle.CleanupStaleSockets(); err != nil {
		if *verbose {
			log.Printf("[goyoke] cleanup stale sockets: %v", err)
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

	// Load cross-session prompt history. The config dir respects --config-dir,
	// GOYOKE_CONFIG_DIR, and CLAUDE_CONFIG_DIR for multi-config setups.
	histDir := os.Getenv("GOYOKE_CONFIG_DIR")
	if histDir == "" {
		histDir = os.Getenv("CLAUDE_CONFIG_DIR")
	}
	if histDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			histDir = filepath.Join(home, ".goyoke")
		}
	}
	if histDir != "" {
		history := claudepkg.LoadInputHistory(histDir)
		cp.SetHistory(history)
	}

	toastModel := toast.NewToastModel()
	app.SetToasts(&toastModel)

	teamReg := teams.NewTeamRegistry()
	teamList := teams.NewTeamListModel(teamReg)
	app.SetTeamList(&teamList)
	teamsHealth := teams.NewTeamsHealthModel(teamReg)
	app.SetTeamsHealth(teamsHealth)

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
	// (including .goyoke/current-session and .goyoke/tmp symlink), and
	// create initial SessionData.
	// -----------------------------------------------------------------------

	sessionStore := session.NewStore("")

	// If --resume is set and no explicit --session-id was given, resolve the
	// most recent session from the store and treat it as the target session ID.
	if *resume && *sessionID == "" {
		sessions, err := sessionStore.ListSessions()
		if err != nil {
			log.Printf("[goyoke] warning: could not list sessions for --resume: %v", err)
		} else if len(sessions) > 0 {
			*sessionID = sessions[0].ID
			if *verbose {
				log.Printf("[goyoke] --resume: selected session %s (last used %s)",
					sessions[0].ID, sessions[0].LastUsed.Format(time.RFC3339))
			}
		} else if *verbose {
			log.Printf("[goyoke] --resume: no existing sessions found, starting new session")
		}
	}

	var sessionData *session.SessionData
	if *sessionID != "" {
		sd, err := sessionStore.LoadSession(*sessionID)
		if err != nil {
			log.Printf("[goyoke] warning: could not load session %q: %v", *sessionID, err)
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
				log.Printf("[goyoke] resumed session %s (cost=$%.4f, providers=%d)",
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
			log.Printf("[goyoke] warning: could not setup session dir: %v", err)
		}
		if *verbose {
			log.Printf("[goyoke] new session %s", newID)
		}
	}

	app.SetSessionStore(sessionStore)
	app.SetSessionData(sessionData)

	// Set up team list polling so the teams health dashboard receives data.
	// The teams directory lives at {sessionDir}/teams/, created by
	// goyoke-skill-guard when a team is dispatched via goyoke-team-run.
	// StartPolling sets the dir and marks the model ready; the returned Cmd
	// is discarded here because Init() calls PollNow() to kick the first
	// tick inside the Bubbletea event loop.
	teamsDir := filepath.Join(sessionStore.SessionDir(sessionData.ID), "teams")
	_ = teamList.StartPolling(teamsDir)

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

	drawerStack := drawer.NewDrawerStack()
	app.SetDrawerStack(&drawerStack)

	cwdSel := cwdselector.New()
	app.SetCWDSelector(cwdSel)

	// -----------------------------------------------------------------------
	// Phase 1b: Locate goyoke-mcp binary and generate MCP config.
	//
	// The goyoke-mcp binary bridges Claude CLI's MCP tool calls to the
	// TUI's IPC bridge via Unix domain socket.  We generate a temporary
	// mcp-config.json and pass it via --mcp-config so the Claude subprocess
	// spawns the MCP server automatically.
	// -----------------------------------------------------------------------

	mcpConfigPath := ""
	mcpBinary, mcpFound := findMCPBinary(*mcpBinaryFlag)
	if mcpFound {
		path, err := writeMCPConfig(mcpBinary)
		if err != nil {
			// Always warn — a broken MCP config means spawn_agent silently fails.
			fmt.Fprintf(os.Stderr, "[goyoke] warning: could not write MCP config: %v\n", err)
		} else {
			mcpConfigPath = path
			if *verbose {
				log.Printf("[goyoke] MCP config: %s (binary: %s)", path, mcpBinary)
			}
		}
	} else {
		// Always emit this warning regardless of --verbose; silent degradation is
		// the root cause of spawn_agent appearing broken.
		fmt.Fprintf(os.Stderr, "[goyoke] warning: goyoke-mcp binary not found; MCP tools (spawn_agent, ask_user, etc.) will be unavailable\n")
		fmt.Fprintf(os.Stderr, "[goyoke] hint: run 'make build-go-mcp' to build it, or pass --mcp-binary=/path/to/goyoke-mcp\n")
	}

	// Resolve the initial model: explicit --model flag takes precedence;
	// otherwise use the provider config's default (first Anthropic model).
	initialModel := *modelOverride
	if initialModel == "" && ps != nil {
		initialModel = ps.GetActiveCLIModel()
	}

	// Write embedded settings-template.json to temp file for --settings injection.
	var settingsPath string
	if tmplData, tmplErr := resolve.Default().ReadFile("settings-template.json"); tmplErr == nil {
		runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
		if runtimeDir == "" {
			runtimeDir = os.TempDir()
		}
		tmpPath := filepath.Join(runtimeDir, fmt.Sprintf("goyoke-hooks-%d.json", os.Getpid()))
		if writeErr := os.WriteFile(tmpPath, tmplData, 0600); writeErr == nil {
			settingsPath = tmpPath
			defer os.Remove(tmpPath)
		} else {
			log.Printf("[goyoke] Warning: failed to write settings temp file: %v", writeErr)
		}
	} else {
		log.Printf("[goyoke] Warning: settings-template.json not found in embedded FS, hooks not injected")
	}

	// Build the CLI driver options from flags.
	cliOpts := cli.CLIDriverOpts{
		SessionID:      *sessionID,
		Model:          initialModel,
		ProjectDir:     ".", // current working directory
		PermissionMode: *permMode,
		Verbose:        *verbose,
		Debug:          *debug,
		ConfigDir:      *configDir,
		MCPConfigPath:  mcpConfigPath,
		SettingsPath:   settingsPath,
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
		[]string{"goyoke"},
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
			fmt.Fprintf(os.Stderr, "[goyoke] warning: could not open debug log: %v\n", err)
		} else {
			defer f.Close()
			opts = append(opts, tea.WithInput(os.Stdin))
			log.SetOutput(f)
			log.SetFlags(log.Ltime | log.Lmicroseconds)
			if *verbose {
				log.Printf("[goyoke] debug log opened: %s", f.Name())
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
		fmt.Fprintf(os.Stderr, "[goyoke] bridge error: %v\n", err)
		os.Exit(1)
	}
	app.SetBridge(ipcBridge)
	ipcBridge.Start()

	// Expose the UDS path so the MCP server subprocess can connect.
	if err := os.Setenv("GOYOKE_SOCKET", ipcBridge.SocketPath()); err != nil {
		if *verbose {
			log.Printf("[goyoke] warning: could not set GOYOKE_SOCKET: %v", err)
		}
	}

	// Expose the MCP config path so spawned Claude subprocesses can load MCP tools.
	if mcpConfigPath != "" {
		if err := os.Setenv("GOYOKE_MCP_CONFIG", mcpConfigPath); err != nil {
			if *verbose {
				log.Printf("[goyoke] warning: could not set GOYOKE_MCP_CONFIG: %v", err)
			}
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
				log.Printf("[goyoke] %s", msg)
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
			log.Printf("[goyoke] OS signal received, initiating shutdown")
		}
		_ = sm.Shutdown()
	})

	// -----------------------------------------------------------------------
	// Run — blocks until the user quits.
	// -----------------------------------------------------------------------

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "[goyoke] error: %v\n", err)
		os.Exit(1)
	}

	// Post-run: execute sequenced shutdown if not already done (e.g. user
	// quit via menu rather than Ctrl+C). The double-call guard in
	// ShutdownManager ensures this is a no-op if already shut down.
	if err := sm.Shutdown(); err != nil {
		if *verbose {
			log.Printf("[goyoke] shutdown warning: %v", err)
		}
	}
}

// openDebugLog creates a timestamped debug log file in the system temp
// directory and returns the open file.  The caller is responsible for
// closing it.
func openDebugLog() (*os.File, error) {
	name := fmt.Sprintf("goyoke-debug-%s.log", time.Now().Format("20060102-150405"))
	path := filepath.Join(os.TempDir(), name)
	return os.Create(path)
}

// findMCPBinary searches for the goyoke-mcp binary.
//
// Resolution order:
//  1. explicit override (non-empty explicitPath is used as-is)
//  2. filesystem candidates relative to the running binary
//  3. exec.LookPath (searches $PATH)
//
// Returns the absolute path and true if found, or ("", false) otherwise.
func findMCPBinary(explicitPath string) (string, bool) {
	// 1. Honour explicit --mcp-binary flag.
	if explicitPath != "" {
		abs, err := filepath.Abs(explicitPath)
		if err != nil {
			return "", false
		}
		if info, err := os.Stat(abs); err == nil && !info.IsDir() {
			return abs, true
		}
		// Explicit path provided but not found — report the problem clearly.
		fmt.Fprintf(os.Stderr, "[goyoke] warning: --mcp-binary path not found: %s\n", abs)
		return "", false
	}

	exe, _ := os.Executable()
	exeDir := filepath.Dir(exe)

	// 2. Filesystem candidates (ordered by likelihood in dev and installed layouts).
	candidates := []string{
		filepath.Join(exeDir, "goyoke-mcp"),              // same dir as TUI binary
		filepath.Join(exeDir, "bin", "goyoke-mcp"),       // bin/ subdir
		filepath.Join(exeDir, "..", "bin", "goyoke-mcp"), // parent/bin (dev layout)
	}

	for _, path := range candidates {
		abs, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		if info, err := os.Stat(abs); err == nil && !info.IsDir() {
			return abs, true
		}
	}

	// 3. Fall back to $PATH search.
	if path, err := exec.LookPath("goyoke-mcp"); err == nil {
		return path, true
	}

	return "", false
}

// mcpConfig is the JSON structure expected by claude --mcp-config.
type mcpConfig struct {
	MCPServers map[string]mcpServerEntry `json:"mcpServers"`
}

type mcpServerEntry struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// buildRegistry wires all multicall subcommands into a single dispatch registry.
func buildRegistry() *subcmd.Registry {
	reg := subcmd.NewRegistry()
	hooks.RegisterAll(reg)
	utils.RegisterAll(reg)
	reg.Register("mcp", mcpcmd.Run)
	return reg
}

// writeMCPConfig writes a temporary MCP configuration JSON file pointing at
// the given goyoke-mcp binary.  Returns the path to the created file.
// The file is created in os.TempDir and will be cleaned up by the OS.
func writeMCPConfig(mcpBinaryPath string) (string, error) {
	cfg := mcpConfig{
		MCPServers: map[string]mcpServerEntry{
			"goyoke-interactive": {
				Command: mcpBinaryPath,
				Env:     map[string]string{},
			},
		},
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal MCP config: %w", err)
	}

	f, err := os.CreateTemp("", "goyoke-mcp-*.json")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		os.Remove(f.Name())
		return "", fmt.Errorf("write MCP config: %w", err)
	}

	return f.Name(), nil
}
