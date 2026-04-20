package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/Bucket-Chemist/goYoke/defaults"
	"github.com/Bucket-Chemist/goYoke/pkg/resolve"
)

func main() {
	resolve.SetDefault(defaults.FS)
	// Crash diagnostics: log any panic or unexpected exit before output is redirected.
	// SIGKILL cannot be caught, but panics and os.Exit(1) paths will appear here.
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[FATAL] runner panic: %v\n%s", r, debug.Stack())
		}
		log.Printf("[INFO] runner exiting normally")
	}()

	// Ignore SIGPIPE: prevents runner death if a pipe write fails after
	// stdout/stderr are redirected (e.g., if any inherited FD is still a pipe).
	signal.Ignore(syscall.SIGPIPE)

	// Validate arguments
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: goyoke-team-run <team-directory>")
		os.Exit(1)
	}

	teamDir, _ := filepath.Abs(os.Args[1])

	// Validate team directory exists
	if stat, err := os.Stat(teamDir); err != nil || !stat.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: Team directory does not exist or is not a directory: %s\n", teamDir)
		os.Exit(1)
	}

	// Pre-flight config validation (before PID lock)
	if err := preflight(teamDir); err != nil {
		fmt.Fprintf(os.Stderr, "Pre-flight validation failed: %v\n", err)
		os.Exit(1)
	}

	// Become session leader if not already.
	// This enables immunity to Ctrl+C in parent terminal.
	// EPERM is expected when already a session leader (e.g. launched with Setsid:true by the MCP tool).
	if _, err := syscall.Setsid(); err != nil && err != syscall.EPERM {
		fmt.Fprintf(os.Stderr, "Warning: setsid failed unexpectedly: %v (runner may be vulnerable to parent cleanup)\n", err)
	}

	// Acquire PID file (prevents double-start)
	pidFile, err := acquirePIDFile(teamDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "PID file error: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := pidFile.Release(); err != nil {
			log.Printf("Warning: failed to release PID file: %v", err)
		}
	}()

	// Redirect stdout/stderr to runner.log
	logFile, err := redirectOutput(teamDir)
	if err != nil {
		if logFile != nil {
			// Partial redirect — stderr failed but stdout succeeded
			// Log to stdout (which IS redirected) and continue
			log.Printf("Warning: partial output redirect: %v", err)
		} else {
			fmt.Fprintf(os.Stderr, "Failed to redirect output: %v\n", err)
			os.Exit(1)
		}
	}
	defer func() {
		if err := logFile.Close(); err != nil {
			log.Printf("Warning: failed to close log file: %v", err)
		}
	}()

	// Close stdin and reopen to /dev/null
	if err := daemonizeStdin(); err != nil {
		log.Printf("Warning: Failed to daemonize stdin: %v", err)
		// Non-fatal - continue execution
	}

	// Initialize TeamRunner
	runner, err := NewTeamRunner(teamDir)
	if err != nil {
		log.Fatalf("Failed to initialize TeamRunner: %v", err)
	}

	// Initialize UDS client for TUI notifications (noop when GOYOKE_SOCKET is unset)
	udsClient := NewTeamRunUDSClient(os.Getenv("GOYOKE_SOCKET"))
	defer udsClient.Close()
	runner.uds = udsClient

	// Write startup state to config.json for external monitoring
	pid := os.Getpid()
	now := time.Now().UTC().Format(time.RFC3339)
	runner.configMu.Lock()
	if runner.config != nil {
		runner.config.BackgroundPID = &pid
		runner.config.StartedAt = &now
		runner.config.Status = "running"
	}
	runner.configMu.Unlock()
	if err := runner.SaveConfig(); err != nil {
		log.Fatalf("Failed to write startup state to config.json: %v", err)
	}
	log.Printf("[INFO] main: wrote startup state (PID=%d, status=running) to config.json", pid)

	// Register synthetic team parent agent with TUI
	if !udsClient.isNoop() {
		teamName := filepath.Base(teamDir)
		workflowType := ""
		runner.configMu.RLock()
		if runner.config != nil {
			workflowType = runner.config.WorkflowType
		}
		runner.configMu.RUnlock()
		udsClient.notify(typeAgentRegister, agentRegisterPayload{
			AgentID:     "team:" + teamName,
			AgentType:   "team-run",
			Description: workflowType + " team",
		})
		udsClient.notify(typeAgentUpdate, agentUpdatePayload{
			AgentID: "team:" + teamName,
			Status:  "running",
		})
	}

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start heartbeat (background goroutine)
	startHeartbeat(ctx, teamDir)

	// Setup signal handler for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	// Spawn wave execution in goroutine
	doneCh := make(chan error, 1)
	go func() {
		doneCh <- runWaves(ctx, runner)
	}()

	// Wait for completion or signal
	select {
	case err := <-doneCh:
		// Send team completion status to TUI before updating config
		if !udsClient.isNoop() {
			teamName := filepath.Base(teamDir)
			status := "complete"
			if err != nil {
				status = "error"
			}
			udsClient.notify(typeAgentUpdate, agentUpdatePayload{
				AgentID: "team:" + teamName,
				Status:  status,
			})
			// UX-019: notify TUI so Teams tab flashes and auto-switch can trigger.
			udsClient.notify(typeTeamUpdate, teamUpdatePayload{
				TeamDir: teamDir,
				Status:  status,
			})
		}
		// Waves completed normally
		if err != nil {
			log.Printf("Wave execution failed: %v", err)
			if !udsClient.isNoop() {
				udsClient.notify(typeToast, toastPayload{
					Message: fmt.Sprintf("team failed — /team-status to inspect: %v", err),
					Level:   "error",
				})
			}
			// Update config with failure status
			now := time.Now().UTC().Format(time.RFC3339)
			runner.configMu.Lock()
			if runner.config != nil {
				runner.config.Status = "failed"
				runner.config.CompletedAt = &now
			}
			runner.configMu.Unlock()
			runner.SaveConfig()
			os.Exit(1)
		}
		log.Printf("Wave execution completed successfully")
		if !udsClient.isNoop() {
			totalCost := 0.0
			var agentIDs []string
			runner.configMu.RLock()
			if runner.config != nil {
				for _, wave := range runner.config.Waves {
					for _, member := range wave.Members {
						totalCost += member.CostUSD
						id := member.Agent
						if id == "" {
							id = member.Name
						}
						if id != "" {
							agentIDs = append(agentIDs, id)
						}
					}
				}
			}
			runner.configMu.RUnlock()

			// Scan stream files for changed-file count (best-effort, post-completion).
			filesChanged := countChangedFilesInDir(teamDir, agentIDs)

			var toastMsg string
			if filesChanged > 0 {
				toastMsg = fmt.Sprintf("team complete — %d file(s), $%.2f — /team-result to view", filesChanged, totalCost)
			} else {
				toastMsg = fmt.Sprintf("team complete ($%.2f) — /team-result to view findings", totalCost)
			}
			udsClient.notify(typeToast, toastPayload{
				Message: toastMsg,
				Level:   "info",
			})
		}
		// Update config with completion status
		now := time.Now().UTC().Format(time.RFC3339)
		runner.configMu.Lock()
		if runner.config != nil {
			runner.config.Status = "completed"
			runner.config.CompletedAt = &now
		}
		runner.configMu.Unlock()
		runner.SaveConfig()
	case sig := <-sigCh:
		log.Printf("Received signal %s, shutting down gracefully", sig)
		cancel()

		// Kill children with sync point
		killDone := make(chan []error, 1)
		go func() {
			killDone <- runner.killAllChildren()
		}()

		// Wait for EITHER runWaves completion OR total shutdown timeout
		select {
		case err := <-doneCh:
			if err != nil {
				log.Printf("Wave execution terminated with error: %v", err)
			} else {
				log.Printf("Graceful shutdown completed")
			}
			// Also wait for kill cascade to finish (should be done or nearly done)
			select {
			case killErrs := <-killDone:
				for _, e := range killErrs {
					log.Printf("Child cleanup error: %v", e)
				}
			case <-time.After(shutdownTimeout):
				log.Printf("Kill cascade still running at shutdown timeout")
			}
		case <-time.After(shutdownTimeout):
			log.Printf("Shutdown timeout exceeded, forcing exit")
		}
	}

	// PID file cleanup via defer
}

// preflight performs pre-flight config validation before acquiring PID lock.
// This prevents stale PID files when config is invalid.
// Returns error if config loading or validation fails.
func preflight(teamDir string) error {
	tr := &TeamRunner{
		teamDir:    teamDir,
		configPath: filepath.Join(teamDir, ConfigFileName),
		childPIDs:  make(map[int]struct{}),
		spawner:    &claudeSpawner{},
	}
	if err := tr.LoadConfig(); err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return tr.ValidateConfig()
}

// runWaves is implemented in wave.go (TC-008)
