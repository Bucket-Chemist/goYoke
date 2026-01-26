# TUI-CLI-01a: Auto-Restart Behavior

> **Estimated Hours:** 1.0
> **Priority:** P1 - Robustness
> **Dependencies:** TUI-CLI-01
> **Phase:** 1 - Foundation

---

## Description

Implement automatic restart behavior for the Claude subprocess when it crashes or exits unexpectedly. The TUI should seamlessly resume the session without user intervention.

**Behavior:**
- Auto-restart on unexpected exit (non-zero exit code)
- Resume same session ID for continuity
- Exponential backoff on repeated failures
- Configurable restart limits
- Notify TUI of restart events

---

## Tasks

### 1. Define Restart Policy

**File:** `internal/cli/restart.go`

```go
package cli

import (
    "time"
)

type RestartPolicy struct {
    Enabled         bool          // Enable auto-restart (default: true)
    MaxRestarts     int           // Max restarts before giving up (default: 3)
    RestartDelay    time.Duration // Initial delay between restarts (default: 1s)
    MaxDelay        time.Duration // Max delay with backoff (default: 30s)
    BackoffFactor   float64       // Delay multiplier per attempt (default: 2.0)
    ResetAfter      time.Duration // Reset restart count after success (default: 60s)
    PreserveSession bool          // Resume same session ID (default: true)
}

func DefaultRestartPolicy() RestartPolicy {
    return RestartPolicy{
        Enabled:         true,
        MaxRestarts:     3,
        RestartDelay:    1 * time.Second,
        MaxDelay:        30 * time.Second,
        BackoffFactor:   2.0,
        ResetAfter:      60 * time.Second,
        PreserveSession: true,
    }
}

type RestartState struct {
    Attempts      int
    LastAttempt   time.Time
    LastSuccess   time.Time
    CurrentDelay  time.Duration
}

func (rs *RestartState) ShouldRestart(policy RestartPolicy) bool

func (rs *RestartState) NextDelay(policy RestartPolicy) time.Duration

func (rs *RestartState) Reset()
```

### 2. Add Monitor Goroutine to ClaudeProcess

**File:** `internal/cli/subprocess.go` (modify)

```go
type ClaudeProcess struct {
    // ... existing fields ...

    policy        RestartPolicy
    restartState  RestartState
    restartEvents chan RestartEvent
    exitChan      chan error
}

type RestartEvent struct {
    Reason      string    // "crash", "exit", "error"
    AttemptNum  int
    SessionID   string
    WillResume  bool
    NextDelay   time.Duration
    Timestamp   time.Time
}

func (cp *ClaudeProcess) RestartEvents() <-chan RestartEvent {
    return cp.restartEvents
}

func (cp *ClaudeProcess) monitor() {
    for {
        select {
        case <-cp.done:
            return
        case err := <-cp.exitChan:
            if cp.shouldRestart(err) {
                delay := cp.restartState.NextDelay(cp.policy)

                // Notify TUI
                cp.restartEvents <- RestartEvent{
                    Reason:     classifyExitReason(err),
                    AttemptNum: cp.restartState.Attempts + 1,
                    SessionID:  cp.sessionID,
                    WillResume: cp.policy.PreserveSession,
                    NextDelay:  delay,
                    Timestamp:  time.Now(),
                }

                time.Sleep(delay)
                cp.restart()
            } else {
                // Max restarts exceeded - notify fatal
                cp.restartEvents <- RestartEvent{
                    Reason:     "max_restarts_exceeded",
                    AttemptNum: cp.restartState.Attempts,
                    SessionID:  cp.sessionID,
                    WillResume: false,
                    Timestamp:  time.Now(),
                }
            }
        }
    }
}

func (cp *ClaudeProcess) restart() error {
    cp.mu.Lock()
    defer cp.mu.Unlock()

    cp.restartState.Attempts++
    cp.restartState.LastAttempt = time.Now()

    // Preserve session ID if configured
    sessionID := cp.sessionID
    if !cp.policy.PreserveSession {
        sessionID = generateSessionID()
    }

    // Create new command with same config
    cfg := cp.config
    cfg.SessionID = sessionID

    newProcess, err := NewClaudeProcess(cfg)
    if err != nil {
        return err
    }

    // Transfer state
    newProcess.policy = cp.policy
    newProcess.restartState = cp.restartState
    newProcess.restartEvents = cp.restartEvents

    // Start new process
    if err := newProcess.Start(); err != nil {
        return err
    }

    // Update self
    cp.cmd = newProcess.cmd
    cp.stdin = newProcess.stdin
    cp.stdout = newProcess.stdout
    cp.stderr = newProcess.stderr
    cp.done = newProcess.done
    cp.running = true

    return nil
}

func classifyExitReason(err error) string {
    if err == nil {
        return "normal_exit"
    }
    if exitErr, ok := err.(*exec.ExitError); ok {
        if exitErr.ExitCode() == -1 {
            return "signal"
        }
        return "crash"
    }
    return "error"
}
```

### 3. TUI Message Types for Restart

**File:** `internal/tui/messages.go`

```go
package tui

// ProcessRestartingMsg sent when subprocess is restarting
type ProcessRestartingMsg struct {
    Reason      string
    AttemptNum  int
    SessionID   string
    WillResume  bool
    NextDelay   time.Duration
}

// ProcessFatalMsg sent when restart limit exceeded
type ProcessFatalMsg struct {
    Reason     string
    LastError  error
    Attempts   int
}
```

---

## Files to Create

| File | Purpose |
|------|---------|
| `internal/cli/restart.go` | Restart policy and state |
| `internal/cli/restart_test.go` | Unit tests |

## Files to Modify

| File | Change |
|------|--------|
| `internal/cli/subprocess.go` | Add monitor goroutine, restart logic |

---

## Acceptance Criteria

- [ ] `RestartPolicy` configurable with sensible defaults
- [ ] Process restarts automatically on crash
- [ ] Exponential backoff between restart attempts
- [ ] Restart count resets after successful run period
- [ ] Session ID preserved across restarts (configurable)
- [ ] `RestartEvents()` channel notifies TUI of restart attempts
- [ ] Fatal error emitted when max restarts exceeded
- [ ] Unit tests for backoff calculation
- [ ] Integration test with process that crashes

---

## TUI Integration

The TUI should handle restart events:

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case ProcessRestartingMsg:
        m.status = fmt.Sprintf("Restarting (attempt %d/%d)...",
            msg.AttemptNum, m.maxRestarts)
        m.showRestartBanner = true
        return m, nil

    case ProcessFatalMsg:
        m.status = "Claude process failed"
        m.showErrorModal = true
        m.lastError = msg.LastError
        return m, nil
    }
    // ...
}
```

---

## Notes

- Don't restart on intentional quit (ctrl+c, /exit)
- Log restart attempts to stderr for debugging
- Consider saving conversation buffer for resume display
- Restart delay should be visible in TUI status bar
