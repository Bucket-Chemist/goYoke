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
			name:     "Agents advances to PlanDrawer",
			current:  FocusAgents,
			expected: FocusPlanDrawer,
		},
		{
			name:     "PlanDrawer advances to OptionsDrawer",
			current:  FocusPlanDrawer,
			expected: FocusOptionsDrawer,
		},
		{
			name:     "OptionsDrawer advances to TeamsDrawer",
			current:  FocusOptionsDrawer,
			expected: FocusTeamsDrawer,
		},
		{
			name:     "TeamsDrawer advances to FiguresDrawer",
			current:  FocusTeamsDrawer,
			expected: FocusFiguresDrawer,
		},
		{
			name:     "FiguresDrawer wraps around to Claude",
			current:  FocusFiguresDrawer,
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
			name:     "OptionsDrawer steps back to PlanDrawer",
			current:  FocusOptionsDrawer,
			expected: FocusPlanDrawer,
		},
		{
			name:     "PlanDrawer steps back to Agents",
			current:  FocusPlanDrawer,
			expected: FocusAgents,
		},
		{
			name:     "Agents steps back to Claude",
			current:  FocusAgents,
			expected: FocusClaude,
		},
		{
			name:     "TeamsDrawer steps back to OptionsDrawer",
			current:  FocusTeamsDrawer,
			expected: FocusOptionsDrawer,
		},
		{
			name:     "Claude wraps around to FiguresDrawer",
			current:  FocusClaude,
			expected: FocusFiguresDrawer,
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
// TDS-008: FocusRing, FocusNextInRing, FocusPrevInRing tests
// ---------------------------------------------------------------------------

func TestFocusTargetStringDrawers(t *testing.T) {
	tests := []struct {
		name     string
		target   FocusTarget
		expected string
	}{
		{
			name:     "FocusPlanDrawer",
			target:   FocusPlanDrawer,
			expected: "Plan Drawer",
		},
		{
			name:     "FocusOptionsDrawer",
			target:   FocusOptionsDrawer,
			expected: "Options Drawer",
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

func TestFocusRing(t *testing.T) {
	tests := []struct {
		name     string
		expanded []string
		expected []FocusTarget
	}{
		{
			name:     "no expanded drawers",
			expanded: []string{},
			expected: []FocusTarget{FocusClaude, FocusAgents},
		},
		{
			name:     "nil expanded drawers",
			expanded: nil,
			expected: []FocusTarget{FocusClaude, FocusAgents},
		},
		{
			name:     "plan drawer expanded",
			expanded: []string{"plan"},
			expected: []FocusTarget{FocusClaude, FocusAgents, FocusPlanDrawer},
		},
		{
			name:     "options drawer expanded",
			expanded: []string{"options"},
			expected: []FocusTarget{FocusClaude, FocusAgents, FocusOptionsDrawer},
		},
		{
			name:     "both drawers expanded",
			expanded: []string{"plan", "options"},
			expected: []FocusTarget{FocusClaude, FocusAgents, FocusPlanDrawer, FocusOptionsDrawer},
		},
		{
			name:     "unknown drawer IDs ignored",
			expanded: []string{"unknown", "plan"},
			expected: []FocusTarget{FocusClaude, FocusAgents, FocusPlanDrawer},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FocusRing(tc.expanded)
			if len(got) != len(tc.expected) {
				t.Fatalf("FocusRing(%v) len = %d; want %d", tc.expanded, len(got), len(tc.expected))
			}
			for i, g := range got {
				if g != tc.expected[i] {
					t.Errorf("FocusRing(%v)[%d] = %v; want %v", tc.expanded, i, g, tc.expected[i])
				}
			}
		})
	}
}

func TestFocusNextInRing(t *testing.T) {
	baseRing := []FocusTarget{FocusClaude, FocusAgents}
	fullRing := []FocusTarget{FocusClaude, FocusAgents, FocusPlanDrawer, FocusOptionsDrawer}

	tests := []struct {
		name     string
		current  FocusTarget
		ring     []FocusTarget
		expected FocusTarget
	}{
		{
			name:     "Claude → Agents (base ring)",
			current:  FocusClaude,
			ring:     baseRing,
			expected: FocusAgents,
		},
		{
			name:     "Agents wraps to Claude (base ring)",
			current:  FocusAgents,
			ring:     baseRing,
			expected: FocusClaude,
		},
		{
			name:     "Claude → Agents (full ring)",
			current:  FocusClaude,
			ring:     fullRing,
			expected: FocusAgents,
		},
		{
			name:     "Agents → PlanDrawer (full ring)",
			current:  FocusAgents,
			ring:     fullRing,
			expected: FocusPlanDrawer,
		},
		{
			name:     "PlanDrawer → OptionsDrawer (full ring)",
			current:  FocusPlanDrawer,
			ring:     fullRing,
			expected: FocusOptionsDrawer,
		},
		{
			name:     "OptionsDrawer wraps to Claude (full ring)",
			current:  FocusOptionsDrawer,
			ring:     fullRing,
			expected: FocusClaude,
		},
		{
			name:     "current not in ring snaps to ring[0]",
			current:  FocusPlanDrawer,
			ring:     baseRing,
			expected: FocusClaude,
		},
		{
			name:     "empty ring returns FocusClaude",
			current:  FocusAgents,
			ring:     []FocusTarget{},
			expected: FocusClaude,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FocusNextInRing(tc.current, tc.ring)
			if got != tc.expected {
				t.Errorf("FocusNextInRing(%v, %v) = %v; want %v", tc.current, tc.ring, got, tc.expected)
			}
		})
	}
}

func TestFocusPrevInRing(t *testing.T) {
	baseRing := []FocusTarget{FocusClaude, FocusAgents}
	fullRing := []FocusTarget{FocusClaude, FocusAgents, FocusPlanDrawer, FocusOptionsDrawer}

	tests := []struct {
		name     string
		current  FocusTarget
		ring     []FocusTarget
		expected FocusTarget
	}{
		{
			name:     "Agents → Claude (base ring)",
			current:  FocusAgents,
			ring:     baseRing,
			expected: FocusClaude,
		},
		{
			name:     "Claude wraps to Agents (base ring)",
			current:  FocusClaude,
			ring:     baseRing,
			expected: FocusAgents,
		},
		{
			name:     "OptionsDrawer → PlanDrawer (full ring)",
			current:  FocusOptionsDrawer,
			ring:     fullRing,
			expected: FocusPlanDrawer,
		},
		{
			name:     "PlanDrawer → Agents (full ring)",
			current:  FocusPlanDrawer,
			ring:     fullRing,
			expected: FocusAgents,
		},
		{
			name:     "Claude wraps to OptionsDrawer (full ring)",
			current:  FocusClaude,
			ring:     fullRing,
			expected: FocusOptionsDrawer,
		},
		{
			name:     "current not in ring snaps to ring[0]",
			current:  FocusOptionsDrawer,
			ring:     baseRing,
			expected: FocusClaude,
		},
		{
			name:     "empty ring returns FocusClaude",
			current:  FocusAgents,
			ring:     []FocusTarget{},
			expected: FocusClaude,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FocusPrevInRing(tc.current, tc.ring)
			if got != tc.expected {
				t.Errorf("FocusPrevInRing(%v, %v) = %v; want %v", tc.current, tc.ring, got, tc.expected)
			}
		})
	}
}

// TestFocusRingNextPrevInverse verifies that PrevInRing(NextInRing(x, ring), ring) == x
// for all elements in a ring.
func TestFocusRingNextPrevInverse(t *testing.T) {
	rings := []struct {
		name string
		ring []FocusTarget
	}{
		{"base ring", []FocusTarget{FocusClaude, FocusAgents}},
		{"plan ring", []FocusTarget{FocusClaude, FocusAgents, FocusPlanDrawer}},
		{"full ring", []FocusTarget{FocusClaude, FocusAgents, FocusPlanDrawer, FocusOptionsDrawer}},
	}

	for _, r := range rings {
		r := r
		t.Run(r.name, func(t *testing.T) {
			for _, target := range r.ring {
				if got := FocusPrevInRing(FocusNextInRing(target, r.ring), r.ring); got != target {
					t.Errorf("PrevInRing(NextInRing(%v)) = %v; want %v", target, got, target)
				}
				if got := FocusNextInRing(FocusPrevInRing(target, r.ring), r.ring); got != target {
					t.Errorf("NextInRing(PrevInRing(%v)) = %v; want %v", target, got, target)
				}
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
			name:     "PlanPreview advances to Teams",
			current:  RPMPlanPreview,
			expected: RPMTeams,
		},
		{
			name:     "Teams wraps around to Agents",
			current:  RPMTeams,
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
			name:     "Teams steps back to PlanPreview",
			current:  RPMTeams,
			expected: RPMPlanPreview,
		},
		{
			name:     "Agents wraps around to Teams",
			current:  RPMAgents,
			expected: RPMTeams,
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
