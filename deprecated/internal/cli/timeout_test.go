package cli

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultTimeoutConfig(t *testing.T) {
	cfg := DefaultTimeoutConfig()

	// Should have non-zero values
	assert.Greater(t, cfg.ResultTimeout, time.Duration(0))
	assert.Greater(t, cfg.InactivityTimeout, time.Duration(0))

	// Verify expected defaults
	assert.Equal(t, 5*time.Minute, cfg.ResultTimeout)
	assert.Equal(t, 2*time.Minute, cfg.InactivityTimeout)
}

func TestTimeoutConfig_CustomValues(t *testing.T) {
	cfg := TimeoutConfig{
		ResultTimeout:     10 * time.Second,
		InactivityTimeout: 5 * time.Second,
	}

	assert.Equal(t, 10*time.Second, cfg.ResultTimeout)
	assert.Equal(t, 5*time.Second, cfg.InactivityTimeout)
}

func TestTimeoutConfig_ZeroValues(t *testing.T) {
	cfg := TimeoutConfig{}

	// Zero values are valid (means no timeout)
	assert.Equal(t, time.Duration(0), cfg.ResultTimeout)
	assert.Equal(t, time.Duration(0), cfg.InactivityTimeout)
}

func TestTimeoutConfig_Sensible(t *testing.T) {
	cfg := DefaultTimeoutConfig()

	// Inactivity timeout should be shorter than result timeout
	assert.Less(t, cfg.InactivityTimeout, cfg.ResultTimeout,
		"InactivityTimeout should be shorter than ResultTimeout")

	// Both should be in minute range for tool execution
	assert.GreaterOrEqual(t, cfg.ResultTimeout, 1*time.Minute,
		"ResultTimeout should allow for slow tool execution")
	assert.GreaterOrEqual(t, cfg.InactivityTimeout, 30*time.Second,
		"InactivityTimeout should not be too aggressive")
}

func TestTimeoutConfig_Comparison(t *testing.T) {
	tests := []struct {
		name                 string
		cfg                  TimeoutConfig
		description          string
		expectValid          bool
	}{
		{
			name: "aggressive timeouts",
			cfg: TimeoutConfig{
				ResultTimeout:     1 * time.Second,
				InactivityTimeout: 500 * time.Millisecond,
			},
			description: "very short timeouts for testing",
			expectValid: true,
		},
		{
			name: "conservative timeouts",
			cfg: TimeoutConfig{
				ResultTimeout:     10 * time.Minute,
				InactivityTimeout: 5 * time.Minute,
			},
			description: "long timeouts for slow operations",
			expectValid: true,
		},
		{
			name: "disabled timeouts",
			cfg: TimeoutConfig{
				ResultTimeout:     0,
				InactivityTimeout: 0,
			},
			description: "timeouts disabled",
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify we can create these configs
			assert.NotNil(t, tt.cfg)
		})
	}
}
