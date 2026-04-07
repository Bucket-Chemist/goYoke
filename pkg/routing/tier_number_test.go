package routing

import (
	"testing"
)

func TestTierNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected float64
	}{
		{"float64 tier 2", float64(2), 2.0},
		{"float64 tier 1", float64(1), 1.0},
		{"float64 tier 1.5", float64(1.5), 1.5},
		{"float64 tier 3", float64(3), 3.0},
		{"int tier 1", int(1), 1.0},
		{"int tier 2", int(2), 2.0},
		{"string external", "external", 0.0},
		{"string other", "haiku", 0.0},
		{"nil", nil, 0.0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := TierNumber(tc.input)
			if got != tc.expected {
				t.Errorf("TierNumber(%v) = %v, want %v", tc.input, got, tc.expected)
			}
		})
	}
}
