// Package model defines shared state types for the GOgent-Fortress TUI.
// It contains pure data types with no I/O dependencies; all keyboard
// handling and border switching live in the root AppModel (TUI-008).
package model

// FocusTarget identifies which top-level panel currently holds keyboard focus.
// The two panels correspond to the 70/30 layout: the Claude conversation pane
// on the left and the agents/right-panel stack on the right.
type FocusTarget int

const (
	// FocusClaude indicates that the Claude conversation panel holds focus.
	FocusClaude FocusTarget = iota

	// FocusAgents indicates that the right-panel stack holds focus.
	FocusAgents

	// FocusPlanDrawer indicates that the plan drawer holds focus (TDS-008).
	FocusPlanDrawer

	// FocusOptionsDrawer indicates that the options drawer holds focus (TDS-008).
	FocusOptionsDrawer

	// FocusTeamsDrawer indicates that the teams health drawer holds focus.
	FocusTeamsDrawer
)

// focusTargetCount is the total number of FocusTarget values.
// Update this constant whenever a new FocusTarget is added.
const focusTargetCount = int(FocusTeamsDrawer) + 1

// String returns a human-readable name for the FocusTarget.
func (f FocusTarget) String() string {
	switch f {
	case FocusClaude:
		return "Claude"
	case FocusAgents:
		return "Agents"
	case FocusPlanDrawer:
		return "Plan Drawer"
	case FocusOptionsDrawer:
		return "Options Drawer"
	case FocusTeamsDrawer:
		return "Teams Drawer"
	default:
		return "Unknown"
	}
}

// FocusNext returns the next FocusTarget in cycling order, wrapping around
// from the last target back to the first.
func FocusNext(current FocusTarget) FocusTarget {
	return FocusTarget((int(current) + 1) % focusTargetCount)
}

// FocusPrev returns the previous FocusTarget in cycling order, wrapping
// around from the first target back to the last.
func FocusPrev(current FocusTarget) FocusTarget {
	return FocusTarget((int(current) - 1 + focusTargetCount) % focusTargetCount)
}

// FocusRing builds the current focus ring based on which drawers are expanded.
// The base ring is always [FocusClaude, FocusAgents]. Expanded drawers are
// appended: plan first, then options (TDS-008).
func FocusRing(expandedDrawers []string) []FocusTarget {
	ring := []FocusTarget{FocusClaude, FocusAgents}
	for _, id := range expandedDrawers {
		switch id {
		case "plan":
			ring = append(ring, FocusPlanDrawer)
		case "options":
			ring = append(ring, FocusOptionsDrawer)
		case "teams":
			ring = append(ring, FocusTeamsDrawer)
		}
	}
	return ring
}

// FocusNextInRing returns the next FocusTarget in the given ring.
// If current is not in the ring, returns ring[0] (snap behavior).
func FocusNextInRing(current FocusTarget, ring []FocusTarget) FocusTarget {
	if len(ring) == 0 {
		return FocusClaude
	}
	for i, t := range ring {
		if t == current {
			return ring[(i+1)%len(ring)]
		}
	}
	return ring[0] // snap to first
}

// FocusPrevInRing returns the previous FocusTarget in the given ring.
// If current is not in the ring, returns ring[0] (snap behavior).
func FocusPrevInRing(current FocusTarget, ring []FocusTarget) FocusTarget {
	if len(ring) == 0 {
		return FocusClaude
	}
	for i, t := range ring {
		if t == current {
			return ring[(i-1+len(ring))%len(ring)]
		}
	}
	return ring[0]
}

// ---------------------------------------------------------------------------
// RightPanelMode
// ---------------------------------------------------------------------------

// RightPanelMode identifies which view is active in the right panel.
type RightPanelMode int

const (
	// RPMAgents shows the live agent list.
	RPMAgents RightPanelMode = iota

	// RPMDashboard shows the session metrics dashboard.
	RPMDashboard

	// RPMSettings shows the settings panel.
	RPMSettings

	// RPMTelemetry shows the telemetry / routing-decisions view.
	RPMTelemetry

	// RPMPlanPreview shows the implementation plan preview.
	RPMPlanPreview
)

// rightPanelModeCount is the total number of RightPanelMode values.
// Update this constant whenever a new RightPanelMode is added.
const rightPanelModeCount = int(RPMPlanPreview) + 1

// String returns a human-readable name for the RightPanelMode.
func (r RightPanelMode) String() string {
	switch r {
	case RPMAgents:
		return "Agents"
	case RPMDashboard:
		return "Dashboard"
	case RPMSettings:
		return "Settings"
	case RPMTelemetry:
		return "Telemetry"
	case RPMPlanPreview:
		return "Plan Preview"
	default:
		return "Unknown"
	}
}

// NextRightPanelMode returns the next RightPanelMode in cycling order,
// wrapping around from the last mode back to the first.
func NextRightPanelMode(current RightPanelMode) RightPanelMode {
	return RightPanelMode((int(current) + 1) % rightPanelModeCount)
}

// PrevRightPanelMode returns the previous RightPanelMode in cycling order,
// wrapping around from the first mode back to the last.
func PrevRightPanelMode(current RightPanelMode) RightPanelMode {
	return RightPanelMode((int(current) - 1 + rightPanelModeCount) % rightPanelModeCount)
}
