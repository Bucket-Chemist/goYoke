// Package model — TUI-058 responsive layout boundary tests.
//
// These tests verify the 4-tier LayoutTier assignment and split ratios at every
// critical boundary width.  They must be kept in sync with computeLayout().
package model

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// LayoutTier.String
// ---------------------------------------------------------------------------

func TestLayoutTierString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tier LayoutTier
		want string
	}{
		{LayoutCompact, "compact"},
		{LayoutStandard, "standard"},
		{LayoutWide, "wide"},
		{LayoutUltra, "ultra"},
		{LayoutTier(99), "unknown"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()
			if got := tc.tier.String(); got != tc.want {
				t.Errorf("LayoutTier(%d).String() = %q; want %q", int(tc.tier), got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// computeLayout — 4-tier boundary tests
//
// The table covers every critical boundary value: the last width in a tier,
// the first width in the next tier, and a representative mid-tier value.
// ---------------------------------------------------------------------------

func TestComputeLayout_TierBoundaries(t *testing.T) {
	t.Parallel()

	const height = 40

	tests := []struct {
		name          string
		width         int
		wantTier      LayoutTier
		wantShowRight bool
		wantLeftRatio float64 // approximate left/total; 0 means full-width (compact)
	}{
		// Compact tier: < 80
		{"compact_boundary_79", 79, LayoutCompact, false, 0},
		{"compact_mid_60", 60, LayoutCompact, false, 0},
		{"compact_min_1", 1, LayoutCompact, false, 0},

		// Standard tier: 80–119
		// Sub-breakpoint 80–99: 75/25
		{"standard_lower_80", 80, LayoutStandard, true, 0.75},
		{"standard_mid_90", 90, LayoutStandard, true, 0.75},
		{"standard_upper_99", 99, LayoutStandard, true, 0.75},
		// Sub-breakpoint 100–119: 70/30
		{"standard_lower_100", 100, LayoutStandard, true, 0.70},
		{"standard_mid_110", 110, LayoutStandard, true, 0.70},
		{"standard_upper_119", 119, LayoutStandard, true, 0.70},

		// Wide tier: 120–179 — 60/40
		{"wide_lower_120", 120, LayoutWide, true, 0.60},
		{"wide_mid_149", 149, LayoutWide, true, 0.60},
		{"wide_upper_179", 179, LayoutWide, true, 0.60},

		// Ultra tier: >= 180 — 50/50
		{"ultra_lower_180", 180, LayoutUltra, true, 0.50},
		{"ultra_mid_240", 240, LayoutUltra, true, 0.50},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := NewAppModel()
			m.width = tc.width
			m.height = height

			dims := m.computeLayout()

			if dims.tier != tc.wantTier {
				t.Errorf("tier = %s (%d); want %s (%d)",
					dims.tier, dims.tier, tc.wantTier, tc.wantTier)
			}
			if dims.showRightPanel != tc.wantShowRight {
				t.Errorf("showRightPanel = %v; want %v", dims.showRightPanel, tc.wantShowRight)
			}

			if !tc.wantShowRight {
				// Compact: full-width single column.
				wantLeft := tc.width - borderFrame
				if wantLeft < 1 {
					wantLeft = 1
				}
				if dims.leftWidth != wantLeft {
					t.Errorf("leftWidth = %d; want %d (compact full-width)", dims.leftWidth, wantLeft)
				}
				return
			}

			// Two-column: verify approximate ratio within 1 column of tolerance.
			leftOuter := int(float64(tc.width) * tc.wantLeftRatio)
			rightOuter := tc.width - leftOuter
			wantLeft := leftOuter - borderFrame
			wantRight := rightOuter - borderFrame
			if wantLeft < 1 {
				wantLeft = 1
			}
			if wantRight < 1 {
				wantRight = 1
			}

			if math.Abs(float64(dims.leftWidth-wantLeft)) > 1 {
				t.Errorf("leftWidth = %d; want ~%d (ratio %.2f)", dims.leftWidth, wantLeft, tc.wantLeftRatio)
			}
			if math.Abs(float64(dims.rightWidth-wantRight)) > 1 {
				t.Errorf("rightWidth = %d; want ~%d (ratio %.2f)", dims.rightWidth, wantRight, tc.wantLeftRatio)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Tier-specific ratio verification (exact values)
// ---------------------------------------------------------------------------

func TestComputeLayout_WideTerminal_Uses60_40(t *testing.T) {
	t.Parallel()

	m := NewAppModel()
	m.width = 120
	m.height = 40

	dims := m.computeLayout()

	if dims.tier != LayoutWide {
		t.Errorf("tier = %s; want LayoutWide", dims.tier)
	}
	if !dims.showRightPanel {
		t.Error("showRightPanel = false at width 120; want true")
	}

	// At width=120, leftRatio=0.60: leftOuter=72, rightOuter=48.
	// Inner widths subtract borderFrame (2).
	wantLeftInner := int(float64(120)*0.60) - borderFrame  // 72 - 2 = 70
	wantRightInner := (120 - int(float64(120)*0.60)) - borderFrame // 48 - 2 = 46

	if dims.leftWidth != wantLeftInner {
		t.Errorf("leftWidth = %d; want %d", dims.leftWidth, wantLeftInner)
	}
	if dims.rightWidth != wantRightInner {
		t.Errorf("rightWidth = %d; want %d", dims.rightWidth, wantRightInner)
	}
}

func TestComputeLayout_UltraTerminal_Uses50_50(t *testing.T) {
	t.Parallel()

	m := NewAppModel()
	m.width = 200
	m.height = 40

	dims := m.computeLayout()

	if dims.tier != LayoutUltra {
		t.Errorf("tier = %s; want LayoutUltra", dims.tier)
	}
	if !dims.showRightPanel {
		t.Error("showRightPanel = false at width 200; want true")
	}

	// At width=200, leftRatio=0.50: leftOuter=100, rightOuter=100.
	wantLeftInner := int(float64(200)*0.50) - borderFrame  // 100 - 2 = 98
	wantRightInner := (200 - int(float64(200)*0.50)) - borderFrame // 100 - 2 = 98

	if dims.leftWidth != wantLeftInner {
		t.Errorf("leftWidth = %d; want %d", dims.leftWidth, wantLeftInner)
	}
	if dims.rightWidth != wantRightInner {
		t.Errorf("rightWidth = %d; want %d", dims.rightWidth, wantRightInner)
	}
}

// ---------------------------------------------------------------------------
// Standard tier backward-compatibility (preserves pre-TUI-058 behaviour)
// ---------------------------------------------------------------------------

// TestComputeLayout_Standard_75_25_At80 mirrors the pre-TUI-058
// TestComputeLayout_ExactBreakpointAt80_ShowsRightPanel.
func TestComputeLayout_Standard_75_25_At80(t *testing.T) {
	t.Parallel()

	m := NewAppModel()
	m.width = 80
	m.height = 24

	dims := m.computeLayout()

	if dims.tier != LayoutStandard {
		t.Errorf("tier = %s; want LayoutStandard", dims.tier)
	}
	if !dims.showRightPanel {
		t.Error("showRightPanel = false at width 80; want true (80 is inclusive lower bound)")
	}

	wantLeftInner := int(float64(80)*0.75) - borderFrame
	wantRightInner := (80 - int(float64(80)*0.75)) - borderFrame

	if dims.leftWidth != wantLeftInner {
		t.Errorf("leftWidth at 80 = %d; want %d", dims.leftWidth, wantLeftInner)
	}
	if dims.rightWidth != wantRightInner {
		t.Errorf("rightWidth at 80 = %d; want %d", dims.rightWidth, wantRightInner)
	}
}

// TestComputeLayout_Standard_70_30_At100 mirrors the pre-TUI-058
// TestComputeLayout_ExactBreakpointAt100_Uses70_30.
func TestComputeLayout_Standard_70_30_At100(t *testing.T) {
	t.Parallel()

	m := NewAppModel()
	m.width = 100
	m.height = 30

	dims := m.computeLayout()

	if dims.tier != LayoutStandard {
		t.Errorf("tier = %s; want LayoutStandard", dims.tier)
	}

	wantLeftInner := int(float64(100)*0.70) - borderFrame
	wantRightInner := (100 - int(float64(100)*0.70)) - borderFrame

	if dims.leftWidth != wantLeftInner {
		t.Errorf("leftWidth at 100 = %d; want %d", dims.leftWidth, wantLeftInner)
	}
	if dims.rightWidth != wantRightInner {
		t.Errorf("rightWidth at 100 = %d; want %d", dims.rightWidth, wantRightInner)
	}
}
