package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func main() {
	// Validate arguments
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: gogent-team-run <team-directory>")
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

	// Become session leader if not already
	// This enables immunity to Ctrl+C in parent terminal
	// Errors are expected in re-exec scenarios or systemd launches
	_, _ = syscall.Setsid()

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
		// Waves completed normally
		if err != nil {
			log.Printf("Wave execution failed: %v", err)
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
