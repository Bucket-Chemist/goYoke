package model

import (
	"testing"
)

// ---------------------------------------------------------------------------
// FocusTarget tests
// ---------------------------------------------------------------------------

func TestFocusTargetString(t *testing.T) {
	tests := []struct {
		name     string
		target   FocusTarget
		expected string
	}{
		{
			name:     "FocusClaude",
			target:   FocusClaude,
			expected: "Claude",
		},
		{
			name:     "FocusAgents",
			target:   FocusAgents,
			expected: "Agents",
		},
		{
			name:     "unknown value",
			target:   FocusTarget(99),
			expected: "Unknown",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.target.String()
			if got != tc.expected {
				t.Errorf("FocusTarget(%d).String() = %q; want %q", int(tc.target), got, tc.expected)
			}
		})
	}
}

func TestFocusNext(t *testing.T) {
	tests := []struct {
		name     string
		current  FocusTarget
		expected FocusTarget
	}{
		{
			name:     "Claude advances to Agents",
			current:  FocusClaude,
			expected: FocusAgents,
		},
		{
			name:     "Agents wraps around to Claude",
			current:  FocusAgents,
			expected: FocusClaude,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FocusNext(tc.current)
			if got != tc.expected {
				t.Errorf("FocusNext(%v) = %v; want %v", tc.current, got, tc.expected)
			}
		})
	}
}

func TestFocusPrev(t *testing.T) {
	tests := []struct {
		name     string
		current  FocusTarget
		expected FocusTarget
	}{
		{
			name:     "Agents steps back to Claude",
			current:  FocusAgents,
			expected: FocusClaude,
		},
		{
			name:     "Claude wraps around to Agents",
			current:  FocusClaude,
			expected: FocusAgents,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FocusPrev(tc.current)
			if got != tc.expected {
				t.Errorf("FocusPrev(%v) = %v; want %v", tc.current, got, tc.expected)
			}
		})
	}
}

// TestFocusCycleFullLoop verifies that cycling Next through all targets and
// back to the start visits every target exactly once.
func TestFocusCycleFullLoop(t *testing.T) {
	start := FocusClaude
	current := start

	for i := range focusTargetCount {
		next := FocusNext(current)
		if i == focusTargetCount-1 {
			// Last step must wrap back to start.
			if next != start {
				t.Errorf("FocusNext full loop: final step returned %v; want %v (start)", next, start)
			}
		}
		current = next
	}
}

// TestFocusPrevNextInverse verifies that Prev(Next(x)) == x for all targets.
func TestFocusPrevNextInverse(t *testing.T) {
	targets := []FocusTarget{FocusClaude, FocusAgents}

	for _, target := range targets {
		t.Run(target.String(), func(t *testing.T) {
			if got := FocusPrev(FocusNext(target)); got != target {
				t.Errorf("FocusPrev(FocusNext(%v)) = %v; want %v", target, got, target)
			}
			if got := FocusNext(FocusPrev(target)); got != target {
				t.Errorf("FocusNext(FocusPrev(%v)) = %v; want %v", target, got, target)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// RightPanelMode tests
// ---------------------------------------------------------------------------

func TestRightPanelModeString(t *testing.T) {
	tests := []struct {
		name     string
		mode     RightPanelMode
		expected string
	}{
		{
			name:     "RPMAgents",
			mode:     RPMAgents,
			expected: "Agents",
		},
		{
			name:     "RPMDashboard",
			mode:     RPMDashboard,
			expected: "Dashboard",
		},
		{
			name:     "RPMSettings",
			mode:     RPMSettings,
			expected: "Settings",
		},
		{
			name:     "RPMTelemetry",
			mode:     RPMTelemetry,
			expected: "Telemetry",
		},
		{
			name:     "RPMPlanPreview",
			mode:     RPMPlanPreview,
			expected: "Plan Preview",
		},
		{
			name:     "unknown value",
			mode:     RightPanelMode(99),
			expected: "Unknown",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.mode.String()
			if got != tc.expected {
				t.Errorf("RightPanelMode(%d).String() = %q; want %q", int(tc.mode), got, tc.expected)
			}
		})
	}
}

func TestNextRightPanelMode(t *testing.T) {
	tests := []struct {
		name     string
		current  RightPanelMode
		expected RightPanelMode
	}{
		{
			name:     "Agents advances to Dashboard",
			current:  RPMAgents,
			expected: RPMDashboard,
		},
		{
			name:     "Dashboard advances to Settings",
			current:  RPMDashboard,
			expected: RPMSettings,
		},
		{
			name:     "Settings advances to Telemetry",
			current:  RPMSettings,
			expected: RPMTelemetry,
		},
		{
			name:     "Telemetry advances to PlanPreview",
			current:  RPMTelemetry,
			expected: RPMPlanPreview,
		},
		{
			name:     "PlanPreview wraps around to Agents",
			current:  RPMPlanPreview,
			expected: RPMAgents,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NextRightPanelMode(tc.current)
			if got != tc.expected {
				t.Errorf("NextRightPanelMode(%v) = %v; want %v", tc.current, got, tc.expected)
			}
		})
	}
}

func TestPrevRightPanelMode(t *testing.T) {
	tests := []struct {
		name     string
		current  RightPanelMode
		expected RightPanelMode
	}{
		{
			name:     "Dashboard steps back to Agents",
			current:  RPMDashboard,
			expected: RPMAgents,
		},
		{
			name:     "Settings steps back to Dashboard",
			current:  RPMSettings,
			expected: RPMDashboard,
		},
		{
			name:     "Telemetry steps back to Settings",
			current:  RPMTelemetry,
			expected: RPMSettings,
		},
		{
			name:     "PlanPreview steps back to Telemetry",
			current:  RPMPlanPreview,
			expected: RPMTelemetry,
		},
		{
			name:     "Agents wraps around to PlanPreview",
			current:  RPMAgents,
			expected: RPMPlanPreview,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := PrevRightPanelMode(tc.current)
			if got != tc.expected {
				t.Errorf("PrevRightPanelMode(%v) = %v; want %v", tc.current, got, tc.expected)
			}
		})
	}
}

// TestRightPanelModeCycleFullLoop verifies that cycling Next through all modes
// and back to the start visits every mode exactly once.
func TestRightPanelModeCycleFullLoop(t *testing.T) {
	start := RPMAgents
	current := start

	for i := range rightPanelModeCount {
		next := NextRightPanelMode(current)
		if i == rightPanelModeCount-1 {
			// Last step must wrap back to start.
			if next != start {
				t.Errorf("NextRightPanelMode full loop: final step returned %v; want %v (start)", next, start)
			}
		}
		current = next
	}
}

// TestRightPanelModePrevNextInverse verifies that Prev(Next(x)) == x for all modes.
func TestRightPanelModePrevNextInverse(t *testing.T) {
	modes := []RightPanelMode{RPMAgents, RPMDashboard, RPMSettings, RPMTelemetry, RPMPlanPreview}

	for _, mode := range modes {
		t.Run(mode.String(), func(t *testing.T) {
			if got := PrevRightPanelMode(NextRightPanelMode(mode)); got != mode {
				t.Errorf("PrevRightPanelMode(NextRightPanelMode(%v)) = %v; want %v", mode, got, mode)
			}
			if got := NextRightPanelMode(PrevRightPanelMode(mode)); got != mode {
				t.Errorf("NextRightPanelMode(PrevRightPanelMode(%v)) = %v; want %v", mode, got, mode)
			}
		})
	}
}
