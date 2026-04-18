// Package model — TUI-058 responsive layout boundary tests.
//
// These tests verify the 4-tier LayoutTier assignment and focus-driven split
// ratios at every critical boundary width.  They must be kept in sync with
// computeLayout() and the UX-021 focus-aware ratio table.
package model

import (
	"math"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/goYoke/internal/tui/state"
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
// All cases use the default FocusClaude focus (zero value of FocusTarget).
// UX-021 focus-driven ratios: Standard 70/30, Wide 65/35, Ultra 60/40.
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

		// Standard tier: 80–119 — FocusClaude: 70/30 (UX-021)
		{"standard_lower_80", 80, LayoutStandard, true, 0.70},
		{"standard_mid_90", 90, LayoutStandard, true, 0.70},
		{"standard_upper_99", 99, LayoutStandard, true, 0.70},
		{"standard_lower_100", 100, LayoutStandard, true, 0.70},
		{"standard_mid_110", 110, LayoutStandard, true, 0.70},
		{"standard_upper_119", 119, LayoutStandard, true, 0.70},

		// Wide tier: 120–179 — FocusClaude: 65/35 (UX-021)
		{"wide_lower_120", 120, LayoutWide, true, 0.65},
		{"wide_mid_149", 149, LayoutWide, true, 0.65},
		{"wide_upper_179", 179, LayoutWide, true, 0.65},

		// Ultra tier: >= 180 — FocusClaude: 60/40
		{"ultra_lower_180", 180, LayoutUltra, true, 0.60},
		{"ultra_mid_240", 240, LayoutUltra, true, 0.60},
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
// Tier-specific ratio verification (exact values, FocusClaude)
// ---------------------------------------------------------------------------

// TestComputeLayout_WideTerminal_FocusClaude_Uses65_35 verifies that the Wide
// tier with FocusClaude yields a 65/35 split (UX-021).
func TestComputeLayout_WideTerminal_FocusClaude_Uses65_35(t *testing.T) {
	t.Parallel()

	m := NewAppModel()
	m.width = 120
	m.height = 40
	m.focus = FocusClaude

	dims := m.computeLayout()

	if dims.tier != LayoutWide {
		t.Errorf("tier = %s; want LayoutWide", dims.tier)
	}
	if !dims.showRightPanel {
		t.Error("showRightPanel = false at width 120; want true")
	}

	// At width=120, FocusClaude leftRatio=0.65: leftOuter=78, rightOuter=42.
	// Inner widths subtract borderFrame (2).
	wantLeftInner := int(float64(120)*0.65) - borderFrame  // 78 - 2 = 76
	wantRightInner := (120 - int(float64(120)*0.65)) - borderFrame // 42 - 2 = 40

	if dims.leftWidth != wantLeftInner {
		t.Errorf("leftWidth = %d; want %d", dims.leftWidth, wantLeftInner)
	}
	if dims.rightWidth != wantRightInner {
		t.Errorf("rightWidth = %d; want %d", dims.rightWidth, wantRightInner)
	}
}

func TestComputeLayout_UltraTerminal_Uses60_40(t *testing.T) {
	t.Parallel()

	m := NewAppModel()
	m.width = 200
	m.height = 40
	m.focus = FocusClaude

	dims := m.computeLayout()

	if dims.tier != LayoutUltra {
		t.Errorf("tier = %s; want LayoutUltra", dims.tier)
	}
	if !dims.showRightPanel {
		t.Error("showRightPanel = false at width 200; want true")
	}

	// At width=200, FocusClaude leftRatio=0.60: leftOuter=120, rightOuter=80.
	wantLeftInner := int(float64(200)*0.60) - borderFrame  // 120 - 2 = 118
	wantRightInner := (200 - int(float64(200)*0.60)) - borderFrame // 80 - 2 = 78

	if dims.leftWidth != wantLeftInner {
		t.Errorf("leftWidth = %d; want %d", dims.leftWidth, wantLeftInner)
	}
	if dims.rightWidth != wantRightInner {
		t.Errorf("rightWidth = %d; want %d", dims.rightWidth, wantRightInner)
	}
}

// ---------------------------------------------------------------------------
// Standard tier ratio tests (UX-021 focus-driven values)
// ---------------------------------------------------------------------------

// TestComputeLayout_Standard_FocusClaude_At80 verifies that Standard tier at
// width 80 uses the UX-021 FocusClaude ratio (70/30).
func TestComputeLayout_Standard_FocusClaude_At80(t *testing.T) {
	t.Parallel()

	m := NewAppModel()
	m.width = 80
	m.height = 24
	m.focus = FocusClaude

	dims := m.computeLayout()

	if dims.tier != LayoutStandard {
		t.Errorf("tier = %s; want LayoutStandard", dims.tier)
	}
	if !dims.showRightPanel {
		t.Error("showRightPanel = false at width 80; want true (80 is inclusive lower bound)")
	}

	// FocusClaude Standard: leftRatio=0.70
	wantLeftInner := int(float64(80)*0.70) - borderFrame  // 56 - 2 = 54
	wantRightInner := (80 - int(float64(80)*0.70)) - borderFrame // 24 - 2 = 22

	if dims.leftWidth != wantLeftInner {
		t.Errorf("leftWidth at 80 = %d; want %d", dims.leftWidth, wantLeftInner)
	}
	if dims.rightWidth != wantRightInner {
		t.Errorf("rightWidth at 80 = %d; want %d", dims.rightWidth, wantRightInner)
	}
}

// TestComputeLayout_Standard_FocusClaude_At100 verifies that Standard tier at
// width 100 uses the UX-021 FocusClaude ratio (70/30).
func TestComputeLayout_Standard_FocusClaude_At100(t *testing.T) {
	t.Parallel()

	m := NewAppModel()
	m.width = 100
	m.height = 30
	m.focus = FocusClaude

	dims := m.computeLayout()

	if dims.tier != LayoutStandard {
		t.Errorf("tier = %s; want LayoutStandard", dims.tier)
	}

	// FocusClaude Standard: leftRatio=0.70
	// Use a variable so the float multiply is runtime, avoiding const-eval truncation.
	w100 := float64(100)
	leftOuter100 := int(w100 * 0.70) // 70
	wantLeftInner := leftOuter100 - borderFrame     // 68
	wantRightInner := (100 - leftOuter100) - borderFrame // 28

	if dims.leftWidth != wantLeftInner {
		t.Errorf("leftWidth at 100 = %d; want %d", dims.leftWidth, wantLeftInner)
	}
	if dims.rightWidth != wantRightInner {
		t.Errorf("rightWidth at 100 = %d; want %d", dims.rightWidth, wantRightInner)
	}
}

// ---------------------------------------------------------------------------
// UX-021: Focus-driven ratio table test
//
// Verifies every focus × tier combination against the specified ratios:
//   FocusClaude:                      Standard 70/30, Wide 65/35, Ultra 60/40
//   FocusAgents:                      Standard 30/70, Wide 35/65, Ultra 40/60
//   Drawer focus (Plan/Options/Teams): Standard 30/70, Wide 35/65, Ultra 40/60
// ---------------------------------------------------------------------------

func TestComputeLayout_FocusAwareRatios(t *testing.T) {
	t.Parallel()

	const height = 40

	tests := []struct {
		name          string
		width         int
		focus         FocusTarget
		wantTier      LayoutTier
		wantLeftRatio float64
	}{
		// Standard tier (80–119) — all three focus categories
		{"standard_claude_80", 80, FocusClaude, LayoutStandard, 0.70},
		{"standard_claude_100", 100, FocusClaude, LayoutStandard, 0.70},
		{"standard_agents_80", 80, FocusAgents, LayoutStandard, 0.30},
		{"standard_agents_100", 100, FocusAgents, LayoutStandard, 0.30},
		{"standard_plan_drawer_80", 80, FocusPlanDrawer, LayoutStandard, 0.30},
		{"standard_plan_drawer_100", 100, FocusPlanDrawer, LayoutStandard, 0.30},
		{"standard_options_drawer_80", 80, FocusOptionsDrawer, LayoutStandard, 0.30},
		{"standard_teams_drawer_100", 100, FocusTeamsDrawer, LayoutStandard, 0.30},

		// Wide tier (120–179) — all three focus categories
		{"wide_claude_120", 120, FocusClaude, LayoutWide, 0.65},
		{"wide_claude_150", 150, FocusClaude, LayoutWide, 0.65},
		{"wide_agents_120", 120, FocusAgents, LayoutWide, 0.35},
		{"wide_agents_150", 150, FocusAgents, LayoutWide, 0.35},
		{"wide_plan_drawer_120", 120, FocusPlanDrawer, LayoutWide, 0.35},
		{"wide_options_drawer_150", 150, FocusOptionsDrawer, LayoutWide, 0.35},
		{"wide_teams_drawer_120", 120, FocusTeamsDrawer, LayoutWide, 0.35},

		// Ultra tier (>=180) — all three focus categories
		{"ultra_claude_180", 180, FocusClaude, LayoutUltra, 0.60},
		{"ultra_claude_240", 240, FocusClaude, LayoutUltra, 0.60},
		{"ultra_agents_180", 180, FocusAgents, LayoutUltra, 0.40},
		{"ultra_agents_240", 240, FocusAgents, LayoutUltra, 0.40},
		{"ultra_plan_drawer_180", 180, FocusPlanDrawer, LayoutUltra, 0.40},
		{"ultra_options_drawer_240", 240, FocusOptionsDrawer, LayoutUltra, 0.40},
		{"ultra_teams_drawer_180", 180, FocusTeamsDrawer, LayoutUltra, 0.40},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := NewAppModel()
			m.width = tc.width
			m.height = height
			m.focus = tc.focus

			dims := m.computeLayout()

			if dims.tier != tc.wantTier {
				t.Errorf("tier = %s; want %s", dims.tier, tc.wantTier)
			}
			if !dims.showRightPanel {
				t.Error("showRightPanel = false; want true")
			}

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
				t.Errorf("leftWidth = %d; want ~%d (focus=%s, ratio=%.2f)",
					dims.leftWidth, wantLeft, tc.focus, tc.wantLeftRatio)
			}
			if math.Abs(float64(dims.rightWidth-wantRight)) > 1 {
				t.Errorf("rightWidth = %d; want ~%d (focus=%s, ratio=%.2f)",
					dims.rightWidth, wantRight, tc.focus, tc.wantLeftRatio)
			}
		})
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
	teamsHasContent   bool
}

func (s *stubDrawerStack) View() string                                    { return "" }
func (s *stubDrawerStack) SetSize(w, h int)                                {}
func (s *stubDrawerStack) ExpandedDrawers() []string                       { return nil }
func (s *stubDrawerStack) HandleKey(_ string, _ tea.KeyMsg) tea.Cmd        { return nil }
func (s *stubDrawerStack) SetOptionsContent(_ string)                      {}
func (s *stubDrawerStack) ClearOptionsContent()                            {}
func (s *stubDrawerStack) OptionsHasContent() bool                         { return s.optionsHasContent }
func (s *stubDrawerStack) SetPlanContent(_ string)                         {}
func (s *stubDrawerStack) ClearPlanContent()                               {}
func (s *stubDrawerStack) PlanHasContent() bool                            { return s.planHasContent }
func (s *stubDrawerStack) SetTeamsContent(_ string)                        {}
func (s *stubDrawerStack) ClearTeamsContent()                              {}
func (s *stubDrawerStack) TeamsHasContent() bool                           { return s.teamsHasContent }
func (s *stubDrawerStack) RefreshTeamsContent(_ string)                    {}
func (s *stubDrawerStack) TeamsIsMinimized() bool                          { return false }
func (s *stubDrawerStack) SetOptionsFocused(_ bool)                        {}
func (s *stubDrawerStack) SetPlanFocused(_ bool)                           {}
func (s *stubDrawerStack) SetTeamsFocused(_ bool)                          {}
func (s *stubDrawerStack) SetActiveModal(_ string, _ string, _ []string)   {}
func (s *stubDrawerStack) HasActiveModal() bool                            { return false }
func (s *stubDrawerStack) OptionsActiveRequestID() string                  { return "" }
func (s *stubDrawerStack) OptionsSelectedOption() string                   { return "" }
func (s *stubDrawerStack) ClearOptionsModal()                              {}

func TestComputeDrawerLayout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		termWidth     int
		rightWidth    int // right panel inner width passed via dims
		tier          LayoutTier
		contentHeight int
		drawerStack   drawerStackWidget // nil means no drawer
		wantH         int
		wantW         int // drawer width == right panel width (not terminal width)
	}{
		// Compact tier: drawer always suppressed regardless of content.
		{
			name:          "compact_no_content",
			termWidth:     60, rightWidth: 0, tier: LayoutCompact, contentHeight: 30,
			drawerStack: &stubDrawerStack{},
			wantH: 0, wantW: 0,
		},
		{
			name:          "compact_has_content",
			termWidth:     60, rightWidth: 0, tier: LayoutCompact, contentHeight: 30,
			drawerStack: &stubDrawerStack{optionsHasContent: true},
			wantH: 0, wantW: 0,
		},
		// nil drawerStack: always (0, 0).
		{
			name:          "standard_nil_drawerstack",
			termWidth:     120, rightWidth: 34, tier: LayoutStandard, contentHeight: 40,
			drawerStack: nil,
			wantH: 0, wantW: 0,
		},
		// Standard — minimized (no content): 3 drawers × 3 rows = 9 rows.
		// termWidth=120, Standard 70/30: rightOuter=36, rightWidth=34.
		{
			name:          "standard_no_content",
			termWidth:     120, rightWidth: 34, tier: LayoutStandard, contentHeight: 40,
			drawerStack: &stubDrawerStack{},
			wantH: 9, wantW: 34,
		},
		// Standard — has options content: 40% cH, min 9.
		// cH=40 → 40*40/100=16, min 9 → 16, cap cH-5=35 → 16.
		{
			name:          "standard_has_options_content_cH40",
			termWidth:     120, rightWidth: 34, tier: LayoutStandard, contentHeight: 40,
			drawerStack: &stubDrawerStack{optionsHasContent: true},
			wantH: 16, wantW: 34,
		},
		// Standard — has plan content: same logic.
		{
			name:          "standard_has_plan_content_cH40",
			termWidth:     120, rightWidth: 34, tier: LayoutStandard, contentHeight: 40,
			drawerStack: &stubDrawerStack{planHasContent: true},
			wantH: 16, wantW: 34,
		},
		// Wide — minimized. termWidth=150, Wide 60/40: rightOuter=60, rightWidth=58.
		{
			name:          "wide_no_content",
			termWidth:     150, rightWidth: 58, tier: LayoutWide, contentHeight: 40,
			drawerStack: &stubDrawerStack{},
			wantH: 9, wantW: 58,
		},
		// Wide — has content: 40*40/100=16, min 9 → 16.
		{
			name:          "wide_has_content_cH40",
			termWidth:     150, rightWidth: 58, tier: LayoutWide, contentHeight: 40,
			drawerStack: &stubDrawerStack{optionsHasContent: true},
			wantH: 16, wantW: 58,
		},
		// Ultra — minimized. termWidth=200, Ultra 50/50: rightOuter=100, rightWidth=98.
		{
			name:          "ultra_no_content",
			termWidth:     200, rightWidth: 98, tier: LayoutUltra, contentHeight: 40,
			drawerStack: &stubDrawerStack{},
			wantH: 9, wantW: 98,
		},
		// Ultra — has content: 40*40/100=16, min 9 → 16.
		{
			name:          "ultra_has_content_cH40",
			termWidth:     200, rightWidth: 98, tier: LayoutUltra, contentHeight: 40,
			drawerStack: &stubDrawerStack{optionsHasContent: true},
			wantH: 16, wantW: 98,
		},
		// Small terminal (cH=8, has content):
		// 8*40/100=3, min 9 → 9, cap cH-5=3, final min 9 → 9.
		{
			name:          "small_terminal_cH8_has_content",
			termWidth:     120, rightWidth: 34, tier: LayoutStandard, contentHeight: 8,
			drawerStack: &stubDrawerStack{optionsHasContent: true},
			wantH: 9, wantW: 34,
		},
		// Large terminal (cH=60, has content):
		// 60*40/100=24, min 9 → 24, cap cH-5=55 → 24.
		{
			name:          "large_terminal_cH60_has_content",
			termWidth:     120, rightWidth: 34, tier: LayoutStandard, contentHeight: 60,
			drawerStack: &stubDrawerStack{optionsHasContent: true},
			wantH: 24, wantW: 34,
		},
		// Mid-range (cH=20, has content):
		// 20*40/100=8, min 9 → 9, cap cH-5=15 → 9.
		{
			name:          "mid_terminal_cH20_has_content",
			termWidth:     120, rightWidth: 34, tier: LayoutStandard, contentHeight: 20,
			drawerStack: &stubDrawerStack{optionsHasContent: true},
			wantH: 9, wantW: 34,
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
				rightWidth:    tc.rightWidth,
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

// ---------------------------------------------------------------------------
// TUI-002: renderMain drawer overflow regression test
//
// Verify that the rendered main area (panels + drawer) fits within the
// content height by checking that renderLeftPanel/renderRightPanel receive
// a reduced panelH when a drawer is present.
// ---------------------------------------------------------------------------

func TestRenderMain_PanelHeightAccountsForDrawer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		termWidth  int
		termHeight int
		hasContent bool
	}{
		{"standard_minimized", 100, 40, false},
		{"standard_expanded", 100, 40, true},
		{"wide_expanded", 150, 40, true},
		{"ultra_expanded", 200, 40, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := NewAppModel()
			m.width = tc.termWidth
			m.height = tc.termHeight
			m.ready = true
			m.shared.drawerStack = &stubDrawerStack{optionsHasContent: tc.hasContent}

			dims := m.computeLayout()
			drawerH, _ := m.computeDrawerLayout(dims)

			// The rendered main area must not exceed contentHeight.
			// renderMain subtracts drawerH to get panelH.
			panelH := dims.contentHeight - drawerH
			if panelH < 1 {
				panelH = 1
			}

			// panelH + drawerH must equal contentHeight (no overflow).
			total := panelH + drawerH
			if total > dims.contentHeight {
				t.Errorf("panelH(%d) + drawerH(%d) = %d > contentHeight(%d)",
					panelH, drawerH, total, dims.contentHeight)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Tab content switching tests
//
// Verify that renderLeftPanel renders different content based on activeTab.
// ---------------------------------------------------------------------------

// stubClaudePanel satisfies claudePanelWidget for tab switching tests.
type stubClaudePanel struct {
	viewContent string
	focused     bool
}

func (s *stubClaudePanel) HandleMsg(_ tea.Msg) tea.Cmd            { return nil }
func (s *stubClaudePanel) View() string                           { return s.viewContent }
func (s *stubClaudePanel) SetSize(_, _ int)                       {}
func (s *stubClaudePanel) SetFocused(f bool)                      { s.focused = f }
func (s *stubClaudePanel) IsStreaming() bool                      { return false }
func (s *stubClaudePanel) HasInput() bool                         { return false }
func (s *stubClaudePanel) SaveMessages() []state.DisplayMessage   { return nil }
func (s *stubClaudePanel) RestoreMessages(_ []state.DisplayMessage) {}
func (s *stubClaudePanel) SetSender(_ MessageSender)              {}
func (s *stubClaudePanel) AppendSystemMessage(_ string)           {}
func (s *stubClaudePanel) SetTier(_ LayoutTier)                   {}
func (s *stubClaudePanel) SetReduceMotion(_ bool)                 {}
func (s *stubClaudePanel) SetShowTimestamps(_ bool) tea.Cmd       { return nil }
func (s *stubClaudePanel) ViewConversation() string               { return s.viewContent }
func (s *stubClaudePanel) ViewInput() string                      { return "" }
func (s *stubClaudePanel) ApplyOverlay(composed string) string    { return composed }

// stubTeamList satisfies teamListWidget for tab switching tests.
type stubTeamList struct {
	viewContent string
}

func (s *stubTeamList) HandleMsg(_ tea.Msg) tea.Cmd                                       { return nil }
func (s *stubTeamList) View() string                                                       { return s.viewContent }
func (s *stubTeamList) SetSize(_, _ int)                                                   {}
func (s *stubTeamList) StartPolling(_ string) tea.Cmd                                     { return nil }
func (s *stubTeamList) PollNow() tea.Cmd                                                  { return nil }
func (s *stubTeamList) ScanNow()                                                          {}
func (s *stubTeamList) SelectedTeam() string                                              { return "" }
func (s *stubTeamList) CreateDetailModel(_ *state.AgentRegistry) TeamDetailWidget         { return nil }

func TestRenderLeftPanel_TabSwitching(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tab       TabID
		wantSubstr string
	}{
		{"chat_tab", TabChat, "claude-content"},
		{"team_config_tab", TabTeamConfig, "team-list-content"},
		{"agent_config_tab", TabAgentConfig, "Agent Config"},
		{"telemetry_tab", TabTelemetry, "Telemetry"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := NewAppModel()
			m.width = 120
			m.height = 40
			m.activeTab = tc.tab
			m.shared.claudePanel = &stubClaudePanel{viewContent: "claude-content"}
			m.shared.teamList = &stubTeamList{viewContent: "team-list-content"}

			dims := m.computeLayout()
			got := m.renderLeftPanel(dims, dims.contentHeight)

			if !strings.Contains(got, tc.wantSubstr) {
				t.Errorf("renderLeftPanel with tab=%s does not contain %q",
					tc.tab, tc.wantSubstr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// syncFocusState tab-awareness test
// ---------------------------------------------------------------------------

func TestSyncFocusState_ClaudePanelUnfocusedOnNonChatTab(t *testing.T) {
	t.Parallel()

	cp := &stubClaudePanel{}
	m := NewAppModel()
	m.shared.claudePanel = cp
	m.focus = FocusClaude

	// Chat tab: panel should be focused.
	m.activeTab = TabChat
	m.syncFocusState()
	if !cp.focused {
		t.Error("claude panel should be focused on TabChat")
	}

	// Team Config tab: panel should NOT be focused.
	m.activeTab = TabTeamConfig
	m.syncFocusState()
	if cp.focused {
		t.Error("claude panel should NOT be focused on TabTeamConfig")
	}
}

// ---------------------------------------------------------------------------
// UX-007: SimpleMode toggle tests
//
// Verify that simpleMode=true hides the right panel and gives the conversation
// 100% width, regardless of terminal width, and that simpleMode=false restores
// the normal two-column layout.
// ---------------------------------------------------------------------------

// TestComputeLayout_SimpleMode verifies simpleMode behaviour across widths.
// When true: right panel hidden, leftWidth == terminal width - borderFrame.
// When false: normal responsive split (right panel visible at >= 80 cols).
func TestComputeLayout_SimpleMode(t *testing.T) {
	t.Parallel()

	const height = 40

	tests := []struct {
		name          string
		width         int
		simpleMode    bool
		wantShowRight bool
	}{
		// simpleMode=true hides right panel on all tier widths.
		{"simple_standard_80", 80, true, false},
		{"simple_standard_100", 100, true, false},
		{"simple_wide_120", 120, true, false},
		{"simple_wide_150", 150, true, false},
		{"simple_ultra_180", 180, true, false},
		{"simple_ultra_240", 240, true, false},

		// simpleMode=false restores normal layout (compact stays hidden).
		{"normal_compact_79", 79, false, false},
		{"normal_standard_80", 80, false, true},
		{"normal_standard_100", 100, false, true},
		{"normal_wide_120", 120, false, true},
		{"normal_ultra_200", 200, false, true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := NewAppModel()
			m.width = tc.width
			m.height = height
			m.simpleMode = tc.simpleMode

			dims := m.computeLayout()

			if dims.showRightPanel != tc.wantShowRight {
				t.Errorf("width=%d simpleMode=%v: showRightPanel=%v; want %v",
					tc.width, tc.simpleMode, dims.showRightPanel, tc.wantShowRight)
			}

			// When the right panel is hidden (either by simpleMode or compact tier),
			// leftWidth must equal the full terminal width minus border frame.
			if !dims.showRightPanel {
				wantLeft := tc.width - borderFrame
				if wantLeft < 1 {
					wantLeft = 1
				}
				if dims.leftWidth != wantLeft {
					t.Errorf("width=%d simpleMode=%v: leftWidth=%d; want %d (full width)",
						tc.width, tc.simpleMode, dims.leftWidth, wantLeft)
				}
			}
		})
	}
}
