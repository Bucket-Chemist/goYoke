// Package lifecycle manages the ordered shutdown of TUI subsystems with
// timing budgets. It has no dependency on bubbletea or any TUI-specific
// packages; subsystems are represented by narrow interfaces so that the
// package is independently testable.
package lifecycle

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

// ErrShutdownTimeout is returned by Shutdown when the total timing budget is
// exceeded before all phases complete.
var ErrShutdownTimeout = fmt.Errorf("shutdown: total budget exceeded")

// ---------------------------------------------------------------------------
// Interfaces
// ---------------------------------------------------------------------------

// Shutdownable represents a CLI driver that can be interrupted and shut down.
// Interrupt sends SIGINT for a graceful cancellation; Shutdown sends SIGTERM
// followed by a SIGKILL escalation after the CLI budget.
type Shutdownable interface {
	Interrupt() error
	Shutdown() error
}

// BridgeShutdownable represents an IPC bridge that can be shut down.
// Shutdown closes the UDS listener, removes the socket file, and drains any
// pending modal channels.
type BridgeShutdownable interface {
	Shutdown()
}

// ---------------------------------------------------------------------------
// ShutdownOpts
// ---------------------------------------------------------------------------

// ShutdownOpts carries the configuration for a ShutdownManager.
// All duration fields default to their documented values when set to zero.
type ShutdownOpts struct {
	// Driver is the CLI subprocess manager. May be nil; CLI phases are skipped.
	Driver Shutdownable

	// Bridge is the UDS IPC server. May be nil; bridge phase is skipped.
	Bridge BridgeShutdownable

	// SessionSaver is called first during shutdown to persist session state.
	// May be nil; save phase is skipped.
	SessionSaver func()

	// OnStatus is an optional callback invoked with a human-readable status
	// string at each phase transition. May be nil.
	OnStatus func(string)

	// TotalBudget is the maximum wall-clock time allowed for the full shutdown
	// sequence. Zero uses the default of 10 seconds.
	TotalBudget time.Duration

	// CLIBudget is the grace period granted to the CLI subprocess between
	// SIGINT and SIGTERM. Zero uses the default of 2 seconds.
	CLIBudget time.Duration

	// HookBudget is the sleep at the end of shutdown to allow Go runtime hooks
	// to complete. Zero uses the default of 500 milliseconds.
	HookBudget time.Duration
}

// ---------------------------------------------------------------------------
// ShutdownManager
// ---------------------------------------------------------------------------

// ShutdownManager executes the TUI shutdown sequence in a defined order with
// configurable timing budgets. The zero value is not usable; use
// NewShutdownManager instead.
//
// Shutdown sequence (10 s total budget by default):
//  1. Save session  (fast, atomic write)
//  2. Interrupt CLI (SIGINT) then wait CLIBudget
//  3. Shutdown CLI  (SIGTERM → SIGKILL escalation) then wait briefly
//  4. Shutdown bridge
//  5. Wait HookBudget for Go runtime hooks
type ShutdownManager struct {
	driver       Shutdownable
	bridge       BridgeShutdownable
	sessionSaver func()
	onStatus     func(string)
	totalBudget  time.Duration
	cliBudget    time.Duration
	hookBudget   time.Duration
	mu           sync.Mutex
	done         bool
}

// NewShutdownManager creates a ShutdownManager from the supplied options.
// Zero-value durations are replaced with their defaults (10 s / 2 s / 500 ms).
func NewShutdownManager(opts ShutdownOpts) *ShutdownManager {
	total := opts.TotalBudget
	if total == 0 {
		total = 10 * time.Second
	}

	cli := opts.CLIBudget
	if cli == 0 {
		cli = 2 * time.Second
	}

	hook := opts.HookBudget
	if hook == 0 {
		hook = 500 * time.Millisecond
	}

	return &ShutdownManager{
		driver:       opts.Driver,
		bridge:       opts.Bridge,
		sessionSaver: opts.SessionSaver,
		onStatus:     opts.OnStatus,
		totalBudget:  total,
		cliBudget:    cli,
		hookBudget:   hook,
	}
}

// IsDone reports whether Shutdown has been called at least once.
func (sm *ShutdownManager) IsDone() bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.done
}

// Shutdown executes the graceful shutdown sequence with timing budget.
// It is safe to call multiple times; subsequent calls are no-ops and return nil.
// Returns nil on success, or ErrShutdownTimeout if the total budget was exceeded.
func (sm *ShutdownManager) Shutdown() error {
	sm.mu.Lock()
	if sm.done {
		sm.mu.Unlock()
		return nil
	}
	sm.done = true
	sm.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), sm.totalBudget)
	defer cancel()

	// ---------------------------------------------------------------------------
	// Phase 1: Save session
	// ---------------------------------------------------------------------------

	sm.status("Saving session...")
	if sm.sessionSaver != nil {
		sm.sessionSaver()
	}

	// ---------------------------------------------------------------------------
	// Phase 2: Interrupt CLI (SIGINT) then wait cliBudget
	// ---------------------------------------------------------------------------

	sm.status("Stopping CLI...")
	if sm.driver != nil {
		// Errors here are expected when the process is not running; proceed.
		_ = sm.driver.Interrupt()

		// Wait for the CLI grace period or total budget expiry — whichever is
		// sooner. This gives the subprocess time to flush its state cleanly
		// before we escalate to SIGTERM.
		cliTimer := time.NewTimer(sm.cliBudget)
		select {
		case <-cliTimer.C:
			// CLI budget elapsed; proceed to SIGTERM.
		case <-ctx.Done():
			cliTimer.Stop()
			return ErrShutdownTimeout
		}

		// ---------------------------------------------------------------------------
		// Phase 3: Shutdown CLI (SIGTERM → SIGKILL escalation) then brief wait
		// ---------------------------------------------------------------------------

		_ = sm.driver.Shutdown()

		// Allow a brief window (100 ms) for the SIGTERM to take effect before
		// we proceed to bridge shutdown. The driver's internal SIGKILL escalation
		// goroutine continues independently; we do not block on it here.
		briefTimer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-briefTimer.C:
			// Brief wait elapsed; proceed.
		case <-ctx.Done():
			briefTimer.Stop()
			return ErrShutdownTimeout
		}
	}

	// ---------------------------------------------------------------------------
	// Phase 4: Shutdown bridge (AFTER driver)
	// ---------------------------------------------------------------------------

	sm.status("Closing bridge...")
	if sm.bridge != nil {
		sm.bridge.Shutdown()
	}

	// ---------------------------------------------------------------------------
	// Phase 5: Wait for hooks
	// ---------------------------------------------------------------------------

	sm.status("Waiting for hooks...")
	hookTimer := time.NewTimer(sm.hookBudget)
	select {
	case <-hookTimer.C:
		// Hook budget elapsed; proceed.
	case <-ctx.Done():
		hookTimer.Stop()
		return ErrShutdownTimeout
	}

	// ---------------------------------------------------------------------------
	// Done
	// ---------------------------------------------------------------------------

	sm.status("Shutdown complete")

	// Final context check: if we used the entire budget, report timeout.
	select {
	case <-ctx.Done():
		return ErrShutdownTimeout
	default:
		return nil
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// status calls onStatus with msg if a callback is registered.
func (sm *ShutdownManager) status(msg string) {
	if sm.onStatus != nil {
		sm.onStatus(msg)
	}
}
