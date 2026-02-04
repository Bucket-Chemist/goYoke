package cli

import (
	"math"
	"time"
)

// RestartPolicy defines the auto-restart behavior for the Claude subprocess.
// When the subprocess exits unexpectedly, it will be automatically restarted
// according to this policy.
type RestartPolicy struct {
	// Enabled controls whether auto-restart is active.
	// Default: true
	Enabled bool

	// MaxRestarts is the maximum number of consecutive restart attempts
	// before giving up. Default: 3
	MaxRestarts int

	// RestartDelay is the initial delay between restarts.
	// Default: 1 second
	RestartDelay time.Duration

	// MaxDelay is the maximum delay with exponential backoff.
	// Default: 30 seconds
	MaxDelay time.Duration

	// BackoffFactor is the multiplier applied to delay per attempt.
	// Default: 2.0 (doubles each time)
	BackoffFactor float64

	// ResetAfter is the duration of successful operation after which
	// the restart counter resets to zero. Default: 60 seconds
	ResetAfter time.Duration

	// PreserveSession controls whether the same session ID is used
	// across restarts. Default: true
	PreserveSession bool
}

// DefaultRestartPolicy returns a RestartPolicy with sensible defaults.
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

// RestartState tracks the current state of restart attempts.
// It maintains attempt counts, timing, and backoff calculation.
type RestartState struct {
	// Attempts is the current number of consecutive restart attempts
	Attempts int

	// LastAttempt is when the last restart was attempted
	LastAttempt time.Time

	// LastSuccess is when the subprocess last started successfully
	// and ran without crashing
	LastSuccess time.Time

	// CurrentDelay is the calculated backoff delay for next restart
	CurrentDelay time.Duration
}

// ShouldRestart determines whether the subprocess should be restarted
// based on the current state and policy.
func (rs *RestartState) ShouldRestart(policy RestartPolicy) bool {
	if !policy.Enabled {
		return false
	}

	// If we've exceeded max restarts, don't restart
	if rs.Attempts >= policy.MaxRestarts {
		return false
	}

	// Check if we should reset the counter based on time since last success
	if !rs.LastSuccess.IsZero() && time.Since(rs.LastSuccess) >= policy.ResetAfter {
		rs.Reset()
	}

	return true
}

// NextDelay calculates the next backoff delay using exponential backoff.
// The formula is: min(RestartDelay * (BackoffFactor ^ Attempts), MaxDelay)
//
// Example with defaults (1s base, 2.0 factor, 30s max):
//   Attempt 0: 1s
//   Attempt 1: 2s
//   Attempt 2: 4s
//   Attempt 3: 8s
//   Attempt 4: 16s
//   Attempt 5: 30s (capped)
func (rs *RestartState) NextDelay(policy RestartPolicy) time.Duration {
	// Calculate exponential backoff
	delay := float64(policy.RestartDelay) * math.Pow(policy.BackoffFactor, float64(rs.Attempts))

	// Cap at maximum delay
	if time.Duration(delay) > policy.MaxDelay {
		rs.CurrentDelay = policy.MaxDelay
	} else {
		rs.CurrentDelay = time.Duration(delay)
	}

	return rs.CurrentDelay
}

// Reset clears the restart state, resetting attempt counter and timing.
// This should be called after a successful run period.
func (rs *RestartState) Reset() {
	rs.Attempts = 0
	rs.CurrentDelay = 0
	rs.LastSuccess = time.Now()
}

// RecordAttempt increments the attempt counter and records timing.
func (rs *RestartState) RecordAttempt() {
	rs.Attempts++
	rs.LastAttempt = time.Now()
}

// RecordSuccess marks a successful subprocess start.
// If the subprocess has been running for ResetAfter duration,
// the restart counter will be reset on next check.
func (rs *RestartState) RecordSuccess() {
	rs.LastSuccess = time.Now()
}

// RestartEvent is sent through the RestartEvents channel when
// the subprocess is being restarted or restart attempts have failed.
type RestartEvent struct {
	// Reason describes why restart was triggered
	// Values: "crash", "exit", "error", "signal", "max_restarts_exceeded"
	Reason string

	// AttemptNum is the current restart attempt number (1-indexed)
	AttemptNum int

	// SessionID is the session ID that will be used
	SessionID string

	// WillResume indicates whether the session will be preserved
	WillResume bool

	// NextDelay is how long until the next restart attempt
	NextDelay time.Duration

	// Timestamp is when this event occurred
	Timestamp time.Time

	// ExitCode is the process exit code, if available (-1 for signals)
	ExitCode int
}
