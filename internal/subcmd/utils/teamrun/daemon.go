package teamrun

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/process"
)

// Sentinel errors for programmatic error handling (enables errors.Is())
var (
	ErrTeamAlreadyRunning = errors.New("team already running")
	ErrInvalidTeamDir     = errors.New("invalid team directory")
)

// Daemon timing constants
const (
	PIDFileName    = "goyoke-team-run.pid"
	ConfigFileName = "config.json"
	RunnerLogFile  = "runner.log"

	// sigTermGracePeriod is time to wait after SIGTERM before escalating to SIGKILL.
	// Matches typical systemd KillSignal timeout.
	sigTermGracePeriod = 5 * time.Second

	// shutdownTimeout is total time allowed for daemon shutdown, encompassing
	// the SIGTERM grace period and SIGKILL cleanup.
	shutdownTimeout = 10 * time.Second
)

// PIDFile represents a process ID file for daemon lifecycle management
type PIDFile struct {
	path string
	pid  int
}

// acquirePIDFile writes the current PID to a file and checks for double-start
// Returns error if another instance is already running for this team
func acquirePIDFile(teamDir string) (*PIDFile, error) {
	pidPath := filepath.Join(teamDir, PIDFileName)

	// Check for existing PID file
	if data, err := os.ReadFile(pidPath); err == nil {
		existingPID, parseErr := strconv.Atoi(strings.TrimSpace(string(data)))
		if parseErr == nil && existingPID > 0 && processExists(existingPID) {
			return nil, fmt.Errorf("%w (PID %d)", ErrTeamAlreadyRunning, existingPID)
		}
		// Stale PID file - clean up
		if err := os.Remove(pidPath); err != nil && !os.IsNotExist(err) {
			log.Printf("Warning: failed to remove stale PID file %s: %v", pidPath, err)
		}
		log.Printf("Cleaned up stale PID file (process %d not running)", existingPID)
	}

	// Write current PID
	pid := os.Getpid()
	if err := os.WriteFile(pidPath, []byte(fmt.Sprintf("%d\n", pid)), 0644); err != nil {
		return nil, fmt.Errorf("write PID file: %w", err)
	}

	log.Printf("Acquired PID file: %s (PID %d)", pidPath, pid)
	return &PIDFile{path: pidPath, pid: pid}, nil
}

// Release removes the PID file
// Safe to call multiple times (via defer)
func (pf *PIDFile) Release() error {
	if err := os.Remove(pf.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove PID file: %w", err)
	}
	log.Printf("Released PID file: %s", pf.path)
	return nil
}

// processExists checks if a process with given PID is running
// Uses signal 0 to check existence without killing
func processExists(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Send signal 0 to check existence without side effects
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// redirectOutput redirects stdout and stderr to team_dir/runner.log
// Returns the log file handle (caller must close)
func redirectOutput(teamDir string) (*os.File, error) {
	logPath := filepath.Join(teamDir, RunnerLogFile)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open runner.log: %w", err)
	}

	logFd := int(logFile.Fd())

	if err := dup2(logFd, int(os.Stdout.Fd())); err != nil {
		logFile.Close()
		return nil, fmt.Errorf("dup2 stdout: %w", err)
	}
	if err := dup2(logFd, int(os.Stderr.Fd())); err != nil {
		// stdout already redirected — do NOT close logFile (would orphan stdout)
		// Return logFile so caller can still close it cleanly
		return logFile, fmt.Errorf("dup2 stderr (stdout already redirected): %w", err)
	}

	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	return logFile, nil
}

// daemonizeStdin replaces stdin (fd 0) with /dev/null
// Prevents "bad file descriptor" errors when daemon tries to read stdin
func daemonizeStdin() error {
	devNull, err := os.Open("/dev/null")
	if err != nil {
		return fmt.Errorf("open /dev/null: %w", err)
	}
	defer devNull.Close()

	// dup2 atomically replaces fd 0 with devNull, closing old stdin
	if err := dup2(int(devNull.Fd()), 0); err != nil {
		return fmt.Errorf("dup2 /dev/null to stdin: %w", err)
	}

	return nil
}

// TeamRunner manages team execution and child process lifecycle.
//
// Lock Ordering Contract:
// When acquiring multiple locks, ALWAYS acquire writeMu BEFORE configMu.
// This prevents deadlocks between concurrent config mutations and saves.
//
//	Correct:   writeMu.Lock() → configMu.Lock()
//	WRONG:     configMu.Lock() → writeMu.Lock()  // WILL DEADLOCK
//
// Purpose of each lock:
//   - writeMu: Serializes all disk writes (prevents concurrent file corruption)
//   - configMu: Protects the config struct in memory (RWMutex for read concurrency)
//
// Common patterns:
//   - Read-only access: configMu.RLock() only (no writeMu needed)
//   - Mutate + save: writeMu.Lock() → configMu.Lock() → mutate → unlock configMu → save → unlock writeMu
type TeamRunner struct {
	teamDir string

	config     *TeamConfig   // Team configuration (loaded from config.json)
	configPath string        // Path to config.json
	configMu   sync.RWMutex  // Protects config reads/writes (acquire AFTER writeMu when both needed)
	writeMu    sync.Mutex    // Serializes config writes (acquire FIRST when both locks needed)

	spawner        Spawner              // Injected spawn implementation
	uds            *TeamRunUDSClient    // UDS client for TUI notifications (noop when nil)
	budgetWarnSent atomic.Bool          // True after budget warning toast has been sent (fire-once)
	childPIDs      map[int]struct{}     // Track spawned child PIDs
	childrenMu     sync.Mutex           // Protect childPIDs map
}

// NewTeamRunner creates a TeamRunner for the given team directory.
// It is safe for concurrent use by multiple goroutines.
//
// Loads config.json if present. If config.json doesn't exist (e.g., in tests),
// returns success with nil config to maintain compatibility with existing tests.
func NewTeamRunner(teamDir string) (*TeamRunner, error) {
	tr := &TeamRunner{
		teamDir:    teamDir,
		configPath: filepath.Join(teamDir, ConfigFileName),
		spawner:    &claudeSpawner{},
		childPIDs:  make(map[int]struct{}),
	}

	// Attempt to load config (non-fatal if missing)
	if err := tr.LoadConfig(); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	return tr, nil
}

// registerChild adds a child PID to tracking
// Thread-safe for concurrent registration
func (tr *TeamRunner) registerChild(pid int) {
	tr.childrenMu.Lock()
	defer tr.childrenMu.Unlock()
	tr.childPIDs[pid] = struct{}{}
	log.Printf("Registered child process: PID %d", pid)
}

// unregisterChild removes a child PID from tracking
// Thread-safe for concurrent unregistration
func (tr *TeamRunner) unregisterChild(pid int) {
	tr.childrenMu.Lock()
	defer tr.childrenMu.Unlock()
	delete(tr.childPIDs, pid)
	log.Printf("Unregistered child process: PID %d", pid)
}

// childCount returns the number of tracked child processes.
// Safe for concurrent access.
func (tr *TeamRunner) childCount() int {
	tr.childrenMu.Lock()
	defer tr.childrenMu.Unlock()
	return len(tr.childPIDs)
}

// killAllChildren sends SIGTERM to all children, waits sigTermGracePeriod, then SIGKILL.
// Returns a slice of errors encountered during signal cascade (empty slice = success).
// Signal cascade: SIGTERM (graceful) → grace period → SIGKILL (force)
// Sends signals to entire process group (negative PID) to kill session leaders and their children.
func (tr *TeamRunner) killAllChildren() []error {
	tr.childrenMu.Lock()
	childCount := len(tr.childPIDs)
	childPIDs := make([]int, 0, childCount)
	for pid := range tr.childPIDs {
		childPIDs = append(childPIDs, pid)
	}
	tr.childrenMu.Unlock()

	if childCount == 0 {
		log.Printf("No child processes to terminate")
		return nil
	}

	log.Printf("Terminating %d child process(es)", childCount)

	var errs []error

	// Phase 1: Send SIGTERM to all children and their process groups
	for _, pid := range childPIDs {
		// Try to kill the entire process group first (for session leaders)
		if err := process.KillGroup(pid, syscall.SIGTERM); err != nil {
			// If process group kill fails, try individual process
			if err := process.Kill(pid, syscall.SIGTERM); err != nil {
				log.Printf("Failed to send SIGTERM to %d: %v", pid, err)
				errs = append(errs, fmt.Errorf("SIGTERM to %d: %w", pid, err))
			} else {
				log.Printf("Sent SIGTERM to child process: PID %d", pid)
			}
		} else {
			log.Printf("Sent SIGTERM to child process group: PGID %d", pid)
		}
	}

	// Grace period for graceful shutdown
	log.Printf("Waiting %v grace period for graceful shutdown...", sigTermGracePeriod)
	time.Sleep(sigTermGracePeriod)

	// Phase 2: SIGKILL stragglers
	killCount := 0
	for _, pid := range childPIDs {
		if !processExists(pid) {
			log.Printf("Child process %d exited gracefully", pid)
			continue
		}

		// Try process group kill first, then individual
		if err := process.KillGroup(pid, syscall.SIGKILL); err != nil {
			if err := process.Kill(pid, syscall.SIGKILL); err != nil {
				log.Printf("Failed to send SIGKILL to %d: %v", pid, err)
				errs = append(errs, fmt.Errorf("SIGKILL to %d: %w", pid, err))
			} else {
				log.Printf("Sent SIGKILL to stubborn child process: PID %d", pid)
				killCount++
			}
		} else {
			log.Printf("Sent SIGKILL to stubborn child process group: PGID %d", pid)
			killCount++
		}
	}

	if killCount > 0 {
		log.Printf("Force-killed %d stubborn child process(es)", killCount)
	}

	// Clear stale PIDs from tracking
	tr.childrenMu.Lock()
	for _, pid := range childPIDs {
		delete(tr.childPIDs, pid)
	}
	tr.childrenMu.Unlock()

	return errs
}
