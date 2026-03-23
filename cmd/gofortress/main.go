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
//
// Version is injected at build time:
//
//	go build -ldflags "-X main.version=v1.0.0" ./cmd/gofortress/...
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/lifecycle"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/bridge"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/cli"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/tabbar"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/model"
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

	flag.Parse()

	if *printVersion {
		fmt.Printf("gofortress version %s\n", version)
		os.Exit(0)
	}

	if *mcpServer {
		fmt.Fprintln(os.Stderr, "MCP server mode not yet implemented. Use gofortress-mcp binary.")
		os.Exit(1)
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

	// Build the CLI driver options from flags.
	cliOpts := cli.CLIDriverOpts{
		SessionID:      *sessionID,
		Model:          *modelOverride,
		ProjectDir:     ".", // current working directory
		PermissionMode: *permMode,
		Verbose:        *verbose,
		Debug:          *debug,
	}
	driver := cli.NewCLIDriver(cliOpts)
	app.SetCLIDriver(driver)
	defer driver.Shutdown() //nolint:errcheck

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
	defer ipcBridge.Shutdown()

	// Expose the UDS path so the MCP server subprocess can connect.
	if err := os.Setenv("GOFORTRESS_SOCKET", ipcBridge.SocketPath()); err != nil {
		if *verbose {
			log.Printf("[gofortress] warning: could not set GOFORTRESS_SOCKET: %v", err)
		}
	}

	// -----------------------------------------------------------------------
	// Run — blocks until the user quits.
	// -----------------------------------------------------------------------

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "[gofortress] error: %v\n", err)
		os.Exit(1)
	}
}

// openDebugLog creates a timestamped debug log file in the system temp
// directory and returns the open file.  The caller is responsible for
// closing it.
func openDebugLog() (*os.File, error) {
	name := fmt.Sprintf("gofortress-debug-%s.log", time.Now().Format("20060102-150405"))
	path := fmt.Sprintf("%s/%s", os.TempDir(), name)
	return os.Create(path)
}
