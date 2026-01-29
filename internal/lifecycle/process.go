package lifecycle

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// ProcessManager handles process lifecycle and cleanup
type ProcessManager struct {
	childProcess *os.Process
	socketPath   string
	sigChan      chan os.Signal
	done         chan struct{}
}

// NewProcessManager creates a new process lifecycle manager
func NewProcessManager(socketPath string) *ProcessManager {
	return &ProcessManager{
		socketPath: socketPath,
		sigChan:    make(chan os.Signal, 1),
		done:       make(chan struct{}),
	}
}

// SetChildProcess registers the Claude process for cleanup
func (pm *ProcessManager) SetChildProcess(p *os.Process) {
	pm.childProcess = p
}

// StartSignalHandler begins listening for termination signals
// CRITICAL: Must be called early in main() before spawning Claude
func (pm *ProcessManager) StartSignalHandler(ctx context.Context, onShutdown func()) {
	signal.Notify(pm.sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		select {
		case sig := <-pm.sigChan:
			// Propagate signal to child process
			if pm.childProcess != nil {
				pm.childProcess.Signal(sig)
			}

			// Run shutdown callback (e.g., callbackServer.Shutdown)
			if onShutdown != nil {
				onShutdown()
			}

			// Clean up socket
			os.Remove(pm.socketPath)

			close(pm.done)

		case <-ctx.Done():
			close(pm.done)
		}
	}()
}

// Wait blocks until shutdown complete
func (pm *ProcessManager) Wait() {
	<-pm.done
}

// CleanupStaleSockets removes orphaned socket files from crashed sessions
// CRITICAL: Must be called at startup before creating new socket
func CleanupStaleSockets() error {
	// Check XDG_RUNTIME_DIR first
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = os.TempDir()
	}

	pattern := filepath.Join(runtimeDir, "gofortress-*.sock")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("[lifecycle] Failed to glob socket pattern: %w", err)
	}

	cleaned := 0
	for _, path := range matches {
		pid := extractPIDFromPath(path)
		if pid > 0 && !processExists(pid) {
			if err := os.Remove(path); err != nil {
				// Log but don't fail - socket might be in use
				fmt.Fprintf(os.Stderr, "[lifecycle] Warning: could not remove stale socket %s: %v\n", path, err)
			} else {
				cleaned++
			}
		}
	}

	if cleaned > 0 {
		fmt.Fprintf(os.Stderr, "[lifecycle] Cleaned %d stale socket(s)\n", cleaned)
	}

	return nil
}

// extractPIDFromPath extracts PID from socket filename like "gofortress-12345.sock"
func extractPIDFromPath(path string) int {
	base := filepath.Base(path)
	base = strings.TrimPrefix(base, "gofortress-")
	base = strings.TrimSuffix(base, ".sock")

	pid, err := strconv.Atoi(base)
	if err != nil {
		return 0
	}
	return pid
}

// processExists checks if a process with the given PID is running
func processExists(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, FindProcess always succeeds, so we need to send signal 0
	err = process.Signal(syscall.Signal(0))
	return err == nil
}
