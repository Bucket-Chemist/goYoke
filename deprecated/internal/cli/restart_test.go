package cli

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultRestartPolicy(t *testing.T) {
	policy := DefaultRestartPolicy()

	assert.True(t, policy.Enabled)
	assert.Equal(t, 3, policy.MaxRestarts)
	assert.Equal(t, 1*time.Second, policy.RestartDelay)
	assert.Equal(t, 30*time.Second, policy.MaxDelay)
	assert.Equal(t, 2.0, policy.BackoffFactor)
	assert.Equal(t, 60*time.Second, policy.ResetAfter)
	assert.True(t, policy.PreserveSession)
}

func TestRestartState_ShouldRestart(t *testing.T) {
	tests := []struct {
		name           string
		policy         RestartPolicy
		state          RestartState
		expectedResult bool
	}{
		{
			name:           "policy disabled",
			policy:         RestartPolicy{Enabled: false, MaxRestarts: 3},
			state:          RestartState{Attempts: 0},
			expectedResult: false,
		},
		{
			name:           "first attempt",
			policy:         RestartPolicy{Enabled: true, MaxRestarts: 3},
			state:          RestartState{Attempts: 0},
			expectedResult: true,
		},
		{
			name:           "within max attempts",
			policy:         RestartPolicy{Enabled: true, MaxRestarts: 3},
			state:          RestartState{Attempts: 2},
			expectedResult: true,
		},
		{
			name:           "at max attempts",
			policy:         RestartPolicy{Enabled: true, MaxRestarts: 3},
			state:          RestartState{Attempts: 3},
			expectedResult: false,
		},
		{
			name:           "exceeded max attempts",
			policy:         RestartPolicy{Enabled: true, MaxRestarts: 3},
			state:          RestartState{Attempts: 5},
			expectedResult: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.state.ShouldRestart(tc.policy)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestRestartState_NextDelay(t *testing.T) {
	policy := RestartPolicy{
		RestartDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
	}

	tests := []struct {
		attempts      int
		expectedDelay time.Duration
	}{
		{0, 1 * time.Second},     // 1 * 2^0 = 1s
		{1, 2 * time.Second},     // 1 * 2^1 = 2s
		{2, 4 * time.Second},     // 1 * 2^2 = 4s
		{3, 8 * time.Second},     // 1 * 2^3 = 8s
		{4, 16 * time.Second},    // 1 * 2^4 = 16s
		{5, 30 * time.Second},    // 1 * 2^5 = 32s, capped at 30s
		{6, 30 * time.Second},    // 1 * 2^6 = 64s, capped at 30s
		{10, 30 * time.Second},   // Way over, still capped
	}

	for _, tc := range tests {
		t.Run(tc.expectedDelay.String(), func(t *testing.T) {
			state := RestartState{Attempts: tc.attempts}
			delay := state.NextDelay(policy)
			assert.Equal(t, tc.expectedDelay, delay)
			assert.Equal(t, tc.expectedDelay, state.CurrentDelay)
		})
	}
}

func TestRestartState_NextDelay_DifferentFactors(t *testing.T) {
	tests := []struct {
		name          string
		factor        float64
		baseDelay     time.Duration
		attempts      int
		expectedDelay time.Duration
	}{
		{
			name:          "factor 1.5",
			factor:        1.5,
			baseDelay:     1 * time.Second,
			attempts:      3,
			expectedDelay: time.Duration(1.5 * 1.5 * 1.5 * float64(time.Second)),
		},
		{
			name:          "factor 3.0",
			factor:        3.0,
			baseDelay:     500 * time.Millisecond,
			attempts:      2,
			expectedDelay: time.Duration(3.0 * 3.0 * 500 * float64(time.Millisecond)),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			policy := RestartPolicy{
				RestartDelay:  tc.baseDelay,
				MaxDelay:      1 * time.Hour,
				BackoffFactor: tc.factor,
			}
			state := RestartState{Attempts: tc.attempts}
			delay := state.NextDelay(policy)
			assert.Equal(t, tc.expectedDelay, delay)
		})
	}
}

func TestRestartState_Reset(t *testing.T) {
	state := RestartState{
		Attempts:     5,
		LastAttempt:  time.Now().Add(-10 * time.Second),
		CurrentDelay: 30 * time.Second,
	}

	beforeReset := time.Now()
	state.Reset()
	afterReset := time.Now()

	assert.Equal(t, 0, state.Attempts)
	assert.Equal(t, time.Duration(0), state.CurrentDelay)
	assert.True(t, state.LastSuccess.After(beforeReset) || state.LastSuccess.Equal(beforeReset))
	assert.True(t, state.LastSuccess.Before(afterReset) || state.LastSuccess.Equal(afterReset))
}

func TestRestartState_RecordAttempt(t *testing.T) {
	state := RestartState{Attempts: 2}

	beforeRecord := time.Now()
	state.RecordAttempt()
	afterRecord := time.Now()

	assert.Equal(t, 3, state.Attempts)
	assert.True(t, state.LastAttempt.After(beforeRecord) || state.LastAttempt.Equal(beforeRecord))
	assert.True(t, state.LastAttempt.Before(afterRecord) || state.LastAttempt.Equal(afterRecord))
}

func TestRestartState_RecordSuccess(t *testing.T) {
	state := RestartState{}

	beforeRecord := time.Now()
	state.RecordSuccess()
	afterRecord := time.Now()

	assert.True(t, state.LastSuccess.After(beforeRecord) || state.LastSuccess.Equal(beforeRecord))
	assert.True(t, state.LastSuccess.Before(afterRecord) || state.LastSuccess.Equal(afterRecord))
}

func TestRestartState_ResetAfterSuccessfulRun(t *testing.T) {
	policy := RestartPolicy{
		Enabled:     true,
		MaxRestarts: 3,
		ResetAfter:  100 * time.Millisecond,
	}

	state := RestartState{
		Attempts:    2,
		LastSuccess: time.Now().Add(-200 * time.Millisecond), // Success was 200ms ago
	}

	// ShouldRestart should reset the counter if enough time has passed
	assert.True(t, state.ShouldRestart(policy))
	assert.Equal(t, 0, state.Attempts, "Attempts should be reset after successful run period")
}

func TestRestartState_NoResetBeforeSuccessfulRun(t *testing.T) {
	policy := RestartPolicy{
		Enabled:     true,
		MaxRestarts: 3,
		ResetAfter:  1 * time.Second,
	}

	state := RestartState{
		Attempts:    2,
		LastSuccess: time.Now().Add(-50 * time.Millisecond), // Success was only 50ms ago
	}

	// Should not reset yet - not enough time has passed
	assert.True(t, state.ShouldRestart(policy))
	assert.Equal(t, 2, state.Attempts, "Attempts should not reset before ResetAfter duration")
}

func TestRestartState_ExponentialBackoffSequence(t *testing.T) {
	policy := RestartPolicy{
		RestartDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		MaxRestarts:   6,
		Enabled:       true,
	}

	state := RestartState{}

	expectedDelays := []time.Duration{
		1 * time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
		16 * time.Second,
		30 * time.Second, // Capped
	}

	for i, expectedDelay := range expectedDelays {
		require.True(t, state.ShouldRestart(policy), "attempt %d should be allowed", i)
		delay := state.NextDelay(policy)
		assert.Equal(t, expectedDelay, delay, "attempt %d delay mismatch", i)
		state.RecordAttempt()
	}

	// After max attempts, should not restart
	assert.False(t, state.ShouldRestart(policy), "should not restart after max attempts")
}

func TestRestartEvent_Structure(t *testing.T) {
	// Just verify the struct can be instantiated with all fields
	event := RestartEvent{
		Reason:     "crash",
		AttemptNum: 1,
		SessionID:  "test-session",
		WillResume: true,
		NextDelay:  5 * time.Second,
		Timestamp:  time.Now(),
		ExitCode:   1,
	}

	assert.Equal(t, "crash", event.Reason)
	assert.Equal(t, 1, event.AttemptNum)
	assert.Equal(t, "test-session", event.SessionID)
	assert.True(t, event.WillResume)
	assert.Equal(t, 5*time.Second, event.NextDelay)
	assert.Equal(t, 1, event.ExitCode)
	assert.False(t, event.Timestamp.IsZero())
}

func TestRestartPolicy_CustomValues(t *testing.T) {
	policy := RestartPolicy{
		Enabled:         false,
		MaxRestarts:     10,
		RestartDelay:    500 * time.Millisecond,
		MaxDelay:        1 * time.Minute,
		BackoffFactor:   1.5,
		ResetAfter:      2 * time.Minute,
		PreserveSession: false,
	}

	assert.False(t, policy.Enabled)
	assert.Equal(t, 10, policy.MaxRestarts)
	assert.Equal(t, 500*time.Millisecond, policy.RestartDelay)
	assert.Equal(t, 1*time.Minute, policy.MaxDelay)
	assert.Equal(t, 1.5, policy.BackoffFactor)
	assert.Equal(t, 2*time.Minute, policy.ResetAfter)
	assert.False(t, policy.PreserveSession)
}
