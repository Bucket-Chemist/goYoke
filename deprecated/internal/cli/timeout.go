package cli

import "time"

// TimeoutConfig configures timeout behavior for Claude process communication
type TimeoutConfig struct {
	// ResultTimeout is max time to wait for result after last activity
	ResultTimeout time.Duration

	// InactivityTimeout is max time with no events at all
	InactivityTimeout time.Duration
}

// DefaultTimeoutConfig returns sensible defaults
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		ResultTimeout:     5 * time.Minute, // Tool execution can be slow
		InactivityTimeout: 2 * time.Minute, // But silence is suspicious
	}
}
