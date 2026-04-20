package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// runHoldLock is the --hold-lock daemon mode.
// Invoked as: goyoke-skill-guard --hold-lock <lock-path> <cc-pid> <ready-fd>
//
// It acquires an exclusive flock on the lock file, signals readiness via the
// ready pipe, then polls the CC process until it dies or a signal is received,
// at which point it cleans up and exits.
func runHoldLock() {
	if len(os.Args) < 5 {
		fmt.Fprintln(os.Stderr, "[skill-guard:lock-holder] ERROR: usage: goyoke-skill-guard --hold-lock <lock-path> <cc-pid> <ready-fd>")
		os.Exit(1)
	}

	lockPath := os.Args[2]

	ccPID, err := strconv.Atoi(os.Args[3])
	if err != nil {
		fmt.Fprintf(os.Stderr, "[skill-guard:lock-holder] ERROR: invalid cc-pid %q: %v\n", os.Args[3], err)
		os.Exit(1)
	}

	readyFD, err := strconv.Atoi(os.Args[4])
	if err != nil {
		fmt.Fprintf(os.Stderr, "[skill-guard:lock-holder] ERROR: invalid ready-fd %q: %v\n", os.Args[4], err)
		os.Exit(1)
	}

	// Open lock file (create if not exists)
	fd, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[skill-guard:lock-holder] ERROR: open lock file %q: %v\n", lockPath, err)
		os.Exit(1)
	}

	// Acquire exclusive lock
	if err := syscall.Flock(int(fd.Fd()), syscall.LOCK_EX); err != nil {
		fmt.Fprintf(os.Stderr, "[skill-guard:lock-holder] ERROR: acquire LOCK_EX on %q: %v\n", lockPath, err)
		fd.Close()
		os.Exit(1)
	}

	// cleanup releases the lock, removes guard files, and closes fd.
	cleanup := func() {
		guardPath := strings.TrimSuffix(lockPath, ".lock") + ".json"
		os.Remove(guardPath) //nolint:errcheck
		os.Remove(lockPath)  //nolint:errcheck
		fd.Close()
		fmt.Fprintln(os.Stderr, "[skill-guard:lock-holder] cleanup complete")
	}

	// Signal readiness via pipe
	readyPipe := os.NewFile(uintptr(readyFD), "ready-pipe")
	if _, err := readyPipe.Write([]byte("R")); err != nil {
		fmt.Fprintf(os.Stderr, "[skill-guard:lock-holder] WARNING: write to ready pipe: %v\n", err)
	}
	readyPipe.Close()

	// Watch for SIGTERM/SIGINT
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check if CC process is still alive (signal 0 = existence check)
			if err := syscall.Kill(ccPID, 0); err != nil {
				// ESRCH = no such process
				cleanup()
				os.Exit(0)
			}
		case <-sigCh:
			cleanup()
			os.Exit(0)
		}
	}
}

// isGuardStale checks whether the lock-holder daemon is still alive by
// attempting a non-blocking shared flock on the lock file.
//
// Returns true (stale) when:
//   - Lock file does not exist
//   - Any error opening the file (fail-open)
//   - Shared lock can be acquired (no exclusive lock held)
//   - Any unexpected flock error (fail-open)
//
// Returns false (active) when:
//   - EWOULDBLOCK: an exclusive lock is currently held by the daemon
func isGuardStale(lockPath string) bool {
	fd, err := os.Open(lockPath)
	if err != nil {
		// Missing file → stale; other open errors → fail-open (treat as stale)
		return true
	}
	defer fd.Close()

	err = syscall.Flock(int(fd.Fd()), syscall.LOCK_SH|syscall.LOCK_NB)
	if err == nil {
		// Acquired shared lock — no exclusive lock is held → stale
		syscall.Flock(int(fd.Fd()), syscall.LOCK_UN) //nolint:errcheck
		return true
	}

	if err == syscall.EWOULDBLOCK {
		// Exclusive lock is held by daemon → active
		return false
	}

	// Unexpected error → fail-open (treat as stale)
	return true
}
