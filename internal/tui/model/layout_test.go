// Package model — TUI-058 responsive layout boundary tests.
//
// These tests verify the 4-tier LayoutTier assignment and split ratios at every
// critical boundary width.  They must be kept in sync with computeLayout().
package model

import (
	"math"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
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
func (s *stubDrawerStack) SetOptionsFocused(_ bool)                        {}
func (s *stubDrawerStack) SetPlanFocused(_ bool)                           {}
func (s *stubDrawerStack) SetTeamsFocused(_ bool)                          {}
func (s *stubDrawerStack) SetActiveModal(_ string, _ string, _ []string)   {}
func (s *stubDrawerStack) HasActiveModal() bool                            { return false }
func (s *stubDrawerStack) OptionsActiveRequestID() string                  { return "" }
func (s *stubDrawerStack) OptionsSelectedOption() string                   { return "" }

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
func (s *stubClaudePanel) SaveMessages() []state.DisplayMessage   { return nil }
func (s *stubClaudePanel) RestoreMessages(_ []state.DisplayMessage) {}
func (s *stubClaudePanel) SetSender(_ MessageSender)              {}
func (s *stubClaudePanel) AppendSystemMessage(_ string)           {}
func (s *stubClaudePanel) SetTier(_ LayoutTier)                   {}
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
