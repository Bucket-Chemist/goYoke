// Package model — TUI-058 responsive layout boundary tests.
//
// These tests verify the 4-tier LayoutTier assignment and split ratios at every
// critical boundary width.  They must be kept in sync with computeLayout().
package model

import (
	"math"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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

// ---------------------------------------------------------------------------
// TestRPMAgents_SplitWidths — TUI-006 50/50 right-panel sub-split tests
//
// These tests verify the width arithmetic used by renderRightPanel when
// RPMAgents mode is active.  Rather than parsing rendered strings, they
// compute layoutDims and check that the 50/50 sub-column math holds at each
// tier boundary.
// ---------------------------------------------------------------------------

func TestRPMAgents_SplitWidths(t *testing.T) {
	t.Parallel()

	const height = 40

	tests := []struct {
		name          string
		width         int
		wantTier      LayoutTier
		wantShowRight bool
		wantSplit     bool // Wide/Ultra: 50/50 sub-split applies
	}{
		// Compact: right panel hidden — no split.
		{"compact_60", 60, LayoutCompact, false, false},
		// Standard: both panels visible, but no sub-split.
		{"standard_90", 90, LayoutStandard, true, false},
		{"standard_110", 110, LayoutStandard, true, false},
		// Wide: 50/50 sub-split active.
		{"wide_120", 120, LayoutWide, true, true},
		{"wide_150", 150, LayoutWide, true, true},
		// Ultra: 50/50 sub-split active.
		{"ultra_200", 200, LayoutUltra, true, true},
		{"ultra_240", 240, LayoutUltra, true, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := NewAppModel()
			m.width = tc.width
			m.height = height

			dims := m.computeLayout()

			if dims.tier != tc.wantTier {
				t.Errorf("tier = %s; want %s", dims.tier, tc.wantTier)
			}
			if dims.showRightPanel != tc.wantShowRight {
				t.Errorf("showRightPanel = %v; want %v", dims.showRightPanel, tc.wantShowRight)
			}

			if !tc.wantSplit {
				// Compact or Standard: no sub-split, nothing more to check.
				return
			}

			// Wide/Ultra: verify the 50/50 sub-column arithmetic.
			leftSubW := dims.rightWidth / 2
			rightSubW := dims.rightWidth - leftSubW

			if leftSubW+rightSubW != dims.rightWidth {
				t.Errorf("sub-column widths %d + %d != rightWidth %d",
					leftSubW, rightSubW, dims.rightWidth)
			}
			if leftSubW < 1 {
				t.Errorf("leftSubW = %d; want >= 1", leftSubW)
			}
			if rightSubW < 1 {
				t.Errorf("rightSubW = %d; want >= 1", rightSubW)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// computeDrawerLayout — TUI-002 full-width drawer layout tests
// ---------------------------------------------------------------------------

// stubDrawerStack is a minimal drawerStackWidget implementation for testing.
type stubDrawerStack struct {
	optionsHasContent bool
	planHasContent    bool
}

func (s *stubDrawerStack) View() string                    { return "" }
func (s *stubDrawerStack) SetSize(w, h int)                {}
func (s *stubDrawerStack) ExpandedDrawers() []string       { return nil }
func (s *stubDrawerStack) HandleKey(_ string, _ tea.KeyMsg) tea.Cmd { return nil }
func (s *stubDrawerStack) SetOptionsContent(_ string)      {}
func (s *stubDrawerStack) ClearOptionsContent()            {}
func (s *stubDrawerStack) OptionsHasContent() bool         { return s.optionsHasContent }
func (s *stubDrawerStack) SetPlanContent(_ string)         {}
func (s *stubDrawerStack) ClearPlanContent()               {}
func (s *stubDrawerStack) PlanHasContent() bool            { return s.planHasContent }
func (s *stubDrawerStack) SetOptionsFocused(_ bool)        {}
func (s *stubDrawerStack) SetPlanFocused(_ bool)           {}
func (s *stubDrawerStack) SetActiveModal(_ string, _ string, _ []string) {}
func (s *stubDrawerStack) HasActiveModal() bool            { return false }
func (s *stubDrawerStack) OptionsActiveRequestID() string  { return "" }
func (s *stubDrawerStack) OptionsSelectedOption() string   { return "" }

func TestComputeDrawerLayout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		termWidth     int
		tier          LayoutTier
		contentHeight int
		drawerStack   drawerStackWidget // nil means no drawer
		wantH         int
		wantW         int
	}{
		// Compact tier: drawer always suppressed regardless of content.
		{
			name:          "compact_no_content",
			termWidth:     60, tier: LayoutCompact, contentHeight: 30,
			drawerStack: &stubDrawerStack{},
			wantH: 0, wantW: 0,
		},
		{
			name:          "compact_has_content",
			termWidth:     60, tier: LayoutCompact, contentHeight: 30,
			drawerStack: &stubDrawerStack{optionsHasContent: true},
			wantH: 0, wantW: 0,
		},
		// nil drawerStack: always (0, 0).
		{
			name:          "standard_nil_drawerstack",
			termWidth:     120, tier: LayoutStandard, contentHeight: 40,
			drawerStack: nil,
			wantH: 0, wantW: 0,
		},
		// Standard — minimized (no content): 2-row tab strip.
		{
			name:          "standard_no_content",
			termWidth:     120, tier: LayoutStandard, contentHeight: 40,
			drawerStack: &stubDrawerStack{},
			wantH: 2, wantW: 120,
		},
		// Standard — has options content: 30% cH, capped at 12.
		// cH=40 → 40*30/100=12 → exactly at cap → 12.
		{
			name:          "standard_has_options_content_cH40",
			termWidth:     120, tier: LayoutStandard, contentHeight: 40,
			drawerStack: &stubDrawerStack{optionsHasContent: true},
			wantH: 12, wantW: 120,
		},
		// Standard — has plan content: same cap logic.
		{
			name:          "standard_has_plan_content_cH40",
			termWidth:     120, tier: LayoutStandard, contentHeight: 40,
			drawerStack: &stubDrawerStack{planHasContent: true},
			wantH: 12, wantW: 120,
		},
		// Wide — minimized.
		{
			name:          "wide_no_content",
			termWidth:     150, tier: LayoutWide, contentHeight: 40,
			drawerStack: &stubDrawerStack{},
			wantH: 2, wantW: 150,
		},
		// Wide — has content.
		{
			name:          "wide_has_content_cH40",
			termWidth:     150, tier: LayoutWide, contentHeight: 40,
			drawerStack: &stubDrawerStack{optionsHasContent: true},
			wantH: 12, wantW: 150,
		},
		// Ultra — minimized.
		{
			name:          "ultra_no_content",
			termWidth:     200, tier: LayoutUltra, contentHeight: 40,
			drawerStack: &stubDrawerStack{},
			wantH: 2, wantW: 200,
		},
		// Ultra — has content.
		{
			name:          "ultra_has_content_cH40",
			termWidth:     200, tier: LayoutUltra, contentHeight: 40,
			drawerStack: &stubDrawerStack{optionsHasContent: true},
			wantH: 12, wantW: 200,
		},
		// Small terminal (cH=8, has content):
		// 8*30/100=2 → floor to 5 → cap to contentHeight-5=3 → 3.
		{
			name:          "small_terminal_cH8_has_content",
			termWidth:     120, tier: LayoutStandard, contentHeight: 8,
			drawerStack: &stubDrawerStack{optionsHasContent: true},
			wantH: 3, wantW: 120,
		},
		// Large terminal (cH=60, has content):
		// 60*30/100=18 → cap to 12.
		{
			name:          "large_terminal_cH60_has_content",
			termWidth:     120, tier: LayoutStandard, contentHeight: 60,
			drawerStack: &stubDrawerStack{optionsHasContent: true},
			wantH: 12, wantW: 120,
		},
		// Mid-range (cH=20, has content):
		// 20*30/100=6 → 6 >= 5 → 6 <= 15 → 6.
		{
			name:          "mid_terminal_cH20_has_content",
			termWidth:     120, tier: LayoutStandard, contentHeight: 20,
			drawerStack: &stubDrawerStack{optionsHasContent: true},
			wantH: 6, wantW: 120,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := NewAppModel()
			m.width = tc.termWidth
			m.shared.drawerStack = tc.drawerStack

			dims := layoutDims{
				tier:          tc.tier,
				contentHeight: tc.contentHeight,
			}

			gotH, gotW := m.computeDrawerLayout(dims)

			if gotH != tc.wantH {
				t.Errorf("height = %d; want %d", gotH, tc.wantH)
			}
			if gotW != tc.wantW {
				t.Errorf("width = %d; want %d", gotW, tc.wantW)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TUI-008: Drawer-aware panel height regression tests
//
// Verify that panel heights = contentHeight - drawerHeight across tiers.
// ---------------------------------------------------------------------------

func TestDrawerAwarePanelHeights(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		termWidth     int
		termHeight    int
		hasContent    bool
		wantDrawerH   int // 0 means no drawer
		wantMainHGt0  bool
	}{
		{
			name:         "compact_no_drawer",
			termWidth:    60, termHeight: 40,
			hasContent:   true,
			wantDrawerH:  0,
			wantMainHGt0: true,
		},
		{
			name:         "standard_minimized_drawer",
			termWidth:    100, termHeight: 40,
			hasContent:   false,
			wantDrawerH:  2,
			wantMainHGt0: true,
		},
		{
			name:         "wide_expanded_drawer",
			termWidth:    150, termHeight: 40,
			hasContent:   true,
			wantDrawerH:  12, // 30% of ~30 cH ≈ 9-12, capped at 12
			wantMainHGt0: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := NewAppModel()
			m.width = tc.termWidth
			m.height = tc.termHeight
			m.shared.drawerStack = &stubDrawerStack{optionsHasContent: tc.hasContent}

			dims := m.computeLayout()
			drawerH, _ := m.computeDrawerLayout(dims)

			if tc.wantDrawerH > 0 && drawerH == 0 {
				t.Errorf("drawerH = 0; want > 0")
			}
			if tc.wantDrawerH == 0 && drawerH != 0 {
				t.Errorf("drawerH = %d; want 0", drawerH)
			}

			mainH := dims.contentHeight - drawerH
			if mainH < 1 {
				mainH = 1
			}

			if tc.wantMainHGt0 && mainH < 1 {
				t.Errorf("mainH = %d; want >= 1", mainH)
			}
			if mainH+drawerH > dims.contentHeight {
				t.Errorf("mainH(%d) + drawerH(%d) = %d > contentHeight(%d)",
					mainH, drawerH, mainH+drawerH, dims.contentHeight)
			}
		})
	}
}
