package skillguard

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/filelock"
	"github.com/Bucket-Chemist/goYoke/pkg/process"
)

// runHoldLock is the --hold-lock daemon mode.
// Invoked as: goyoke-skill-guard --hold-lock <lock-path> <cc-pid> <ready-fd>
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

	fd, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[skill-guard:lock-holder] ERROR: open lock file %q: %v\n", lockPath, err)
		os.Exit(1)
	}

	if err := filelock.Lock(int(fd.Fd())); err != nil {
		fmt.Fprintf(os.Stderr, "[skill-guard:lock-holder] ERROR: acquire LOCK_EX on %q: %v\n", lockPath, err)
		fd.Close()
		os.Exit(1)
	}

	cleanup := func() {
		guardPath := strings.TrimSuffix(lockPath, ".lock") + ".json"
		os.Remove(guardPath) //nolint:errcheck
		os.Remove(lockPath)  //nolint:errcheck
		fd.Close()
		fmt.Fprintln(os.Stderr, "[skill-guard:lock-holder] cleanup complete")
	}

	readyPipe := os.NewFile(uintptr(readyFD), "ready-pipe")
	if _, err := readyPipe.Write([]byte("R")); err != nil {
		fmt.Fprintf(os.Stderr, "[skill-guard:lock-holder] WARNING: write to ready pipe: %v\n", err)
	}
	readyPipe.Close()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !process.Exists(ccPID) {
				cleanup()
				os.Exit(0)
			}
		case <-sigCh:
			cleanup()
			os.Exit(0)
		}
	}
}

// isGuardStale checks whether the lock-holder daemon is still alive.
func isGuardStale(lockPath string) bool {
	fd, err := os.Open(lockPath)
	if err != nil {
		return true
	}
	defer fd.Close()

	err = filelock.LockSharedNonBlock(int(fd.Fd()))
	if err == nil {
		filelock.Unlock(int(fd.Fd())) //nolint:errcheck
		return true
	}

	if errors.Is(err, filelock.ErrWouldBlock) {
		return false
	}

	return true
}
