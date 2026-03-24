package skeleton

import (
	"strings"
	"testing"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/util"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// stripANSI removes ANSI escape sequences from s so test assertions can
// compare plain structure without worrying about terminal color codes.
func stripANSI(s string) string {
	var out strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && s[j] != 'm' {
				j++
			}
			i = j + 1
			continue
		}
		out.WriteByte(s[i])
		i++
	}
	return out.String()
}

// newSized returns a SkeletonModel with the given variant and dimensions.
func newSized(v SkeletonVariant, w, h int) SkeletonModel {
	return New(v).SetSize(w, h)
}

// advanceFrames sends n AnimateTickMsg messages to m, advancing the shimmer
// by n frames. It returns the final model.
func advanceFrames(m SkeletonModel, n int) SkeletonModel {
	for range n {
		m, _ = m.Update(util.AnimateTickMsg{})
	}
	return m
}

// lineCount returns the number of newline-separated lines in s.
func lineCount(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

// ---------------------------------------------------------------------------
// New / constructor
// ---------------------------------------------------------------------------

func TestNew_DefaultsActive(t *testing.T) {
	m := New(SkeletonConversation)
	if !m.Active() {
		t.Error("new SkeletonModel should be active by default")
	}
}

func TestNew_ZeroSizeDimensions(t *testing.T) {
	m := New(SkeletonConversation)
	if m.width != 0 || m.height != 0 {
		t.Errorf("new model should have zero dimensions, got w=%d h=%d", m.width, m.height)
	}
}

// ---------------------------------------------------------------------------
// SetSize
// ---------------------------------------------------------------------------

func TestSetSize_UpdatesDimensions(t *testing.T) {
	m := New(SkeletonAgentTree).SetSize(80, 24)
	if m.width != 80 {
		t.Errorf("expected width=80, got %d", m.width)
	}
	if m.height != 24 {
		t.Errorf("expected height=24, got %d", m.height)
	}
}

func TestSetSize_ImmutableOriginal(t *testing.T) {
	original := New(SkeletonSettings)
	updated := original.SetSize(120, 40)

	if original.width != 0 {
		t.Error("SetSize should return a copy; original width should remain 0")
	}
	if updated.width != 120 {
		t.Errorf("updated width should be 120, got %d", updated.width)
	}
}

func TestSetSize_CanBeCalledMultipleTimes(t *testing.T) {
	m := New(SkeletonDashboard).SetSize(40, 10).SetSize(80, 20)
	if m.width != 80 || m.height != 20 {
		t.Errorf("expected 80x20, got %dx%d", m.width, m.height)
	}
}

// ---------------------------------------------------------------------------
// ShouldShow — 500 ms threshold guard
// ---------------------------------------------------------------------------

func TestShouldShow_BelowThresholdFalse(t *testing.T) {
	m := New(SkeletonConversation)
	if m.ShouldShow(400 * time.Millisecond) {
		t.Error("ShouldShow should return false at 400ms (below 500ms threshold)")
	}
}

func TestShouldShow_AtThresholdTrue(t *testing.T) {
	m := New(SkeletonConversation)
	if !m.ShouldShow(500 * time.Millisecond) {
		t.Error("ShouldShow should return true at exactly 500ms")
	}
}

func TestShouldShow_AboveThresholdTrue(t *testing.T) {
	m := New(SkeletonConversation)
	if !m.ShouldShow(600 * time.Millisecond) {
		t.Error("ShouldShow should return true at 600ms (above 500ms threshold)")
	}
}

func TestShouldShow_ZeroDurationFalse(t *testing.T) {
	m := New(SkeletonConversation)
	if m.ShouldShow(0) {
		t.Error("ShouldShow should return false at 0 duration")
	}
}

func TestShouldShow_LargeValueTrue(t *testing.T) {
	m := New(SkeletonConversation)
	if !m.ShouldShow(10 * time.Second) {
		t.Error("ShouldShow should return true at 10 seconds")
	}
}

// ---------------------------------------------------------------------------
// Active
// ---------------------------------------------------------------------------

func TestActive_TrueAfterNew(t *testing.T) {
	m := New(SkeletonSettings)
	if !m.Active() {
		t.Error("model should be active after New")
	}
}

// ---------------------------------------------------------------------------
// Update — frame advancement via AnimateTickMsg
// ---------------------------------------------------------------------------

func TestUpdate_AnimateTickMsg_AdvancesFrame(t *testing.T) {
	m := New(SkeletonConversation)
	initial := m.frame

	m, _ = m.Update(util.AnimateTickMsg{})

	if m.frame == initial {
		t.Errorf("frame should advance on AnimateTickMsg: initial=%d, after=%d", initial, m.frame)
	}
}

func TestUpdate_AnimateTickMsg_ReturnsCmd(t *testing.T) {
	m := New(SkeletonConversation)
	_, cmd := m.Update(util.AnimateTickMsg{})
	if cmd == nil {
		t.Error("Update should return a non-nil cmd for AnimateTickMsg when active")
	}
}

func TestUpdate_UnknownMsg_NoFrameChange(t *testing.T) {
	m := New(SkeletonConversation)
	before := m.frame

	type otherMsg struct{}
	m, _ = m.Update(otherMsg{})

	if m.frame != before {
		t.Errorf("unknown message should not change frame: before=%d after=%d", before, m.frame)
	}
}

func TestUpdate_UnknownMsg_NilCmd(t *testing.T) {
	m := New(SkeletonConversation)

	type otherMsg struct{}
	_, cmd := m.Update(otherMsg{})
	if cmd != nil {
		t.Error("unknown message should return nil cmd")
	}
}

func TestUpdate_MultipleFrames_CountCorrect(t *testing.T) {
	m := New(SkeletonConversation)
	m = advanceFrames(m, 5)
	if m.frame != 5 {
		t.Errorf("expected frame=5 after 5 advances, got %d", m.frame)
	}
}

// ---------------------------------------------------------------------------
// Shimmer wraps at totalFrames
// ---------------------------------------------------------------------------

func TestShimmer_WrapsAtTotalFrames(t *testing.T) {
	m := New(SkeletonConversation)
	// Advance exactly totalFrames times: frame should wrap back to 0.
	m = advanceFrames(m, totalFrames)
	if m.frame != 0 {
		t.Errorf("frame should wrap to 0 after %d advances, got %d", totalFrames, m.frame)
	}
}

func TestShimmer_WrapsCyclically(t *testing.T) {
	m := New(SkeletonConversation)
	// Advance 2 full cycles + 7 extra: should land at 7.
	m = advanceFrames(m, 2*totalFrames+7)
	if m.frame != 7 {
		t.Errorf("expected frame=7 after 2 full cycles + 7, got %d", m.frame)
	}
}

// ---------------------------------------------------------------------------
// View — zero dimensions
// ---------------------------------------------------------------------------

func TestView_ZeroWidth_ReturnsEmpty(t *testing.T) {
	m := New(SkeletonConversation).SetSize(0, 24)
	if got := m.View(); got != "" {
		t.Errorf("expected empty string for zero width, got %q", got)
	}
}

func TestView_ZeroHeight_ReturnsEmpty(t *testing.T) {
	m := New(SkeletonConversation).SetSize(80, 0)
	if got := m.View(); got != "" {
		t.Errorf("expected empty string for zero height, got %q", got)
	}
}

func TestView_BothZero_ReturnsEmpty(t *testing.T) {
	m := New(SkeletonConversation)
	if got := m.View(); got != "" {
		t.Errorf("expected empty string when both dimensions are zero, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// View — each variant renders non-empty at multiple widths
// ---------------------------------------------------------------------------

func TestView_Conversation_NonEmptyAtWidths(t *testing.T) {
	widths := []int{10, 40, 80, 120}
	for _, w := range widths {
		m := newSized(SkeletonConversation, w, 10)
		got := m.View()
		if got == "" {
			t.Errorf("SkeletonConversation width=%d: expected non-empty view", w)
		}
	}
}

func TestView_AgentTree_NonEmptyAtWidths(t *testing.T) {
	widths := []int{10, 40, 80, 120}
	for _, w := range widths {
		m := newSized(SkeletonAgentTree, w, 10)
		got := m.View()
		if got == "" {
			t.Errorf("SkeletonAgentTree width=%d: expected non-empty view", w)
		}
	}
}

func TestView_Settings_NonEmptyAtWidths(t *testing.T) {
	widths := []int{10, 40, 80, 120}
	for _, w := range widths {
		m := newSized(SkeletonSettings, w, 10)
		got := m.View()
		if got == "" {
			t.Errorf("SkeletonSettings width=%d: expected non-empty view", w)
		}
	}
}

func TestView_Dashboard_NonEmptyAtWidths(t *testing.T) {
	widths := []int{10, 40, 80, 120}
	for _, w := range widths {
		m := newSized(SkeletonDashboard, w, 10)
		got := m.View()
		if got == "" {
			t.Errorf("SkeletonDashboard width=%d: expected non-empty view", w)
		}
	}
}

// ---------------------------------------------------------------------------
// View — line count matches height
// ---------------------------------------------------------------------------

func TestView_LineCountMatchesHeight(t *testing.T) {
	tests := []struct {
		variant SkeletonVariant
		name    string
	}{
		{SkeletonConversation, "Conversation"},
		{SkeletonAgentTree, "AgentTree"},
		{SkeletonSettings, "Settings"},
		{SkeletonDashboard, "Dashboard"},
	}
	for _, tt := range tests {
		height := 8
		m := newSized(tt.variant, 80, height)
		got := m.View()
		lines := lineCount(got)
		if lines != height {
			t.Errorf("%s: expected %d lines, got %d", tt.name, height, lines)
		}
	}
}

// ---------------------------------------------------------------------------
// Shimmer frame state changes with each tick
// ---------------------------------------------------------------------------

// TestShimmer_FrameAdvancesOnTick verifies that the internal frame counter
// changes correctly, confirming the shimmer position state is tracked
// independently of rendered output (which depends on TTY color support).
func TestShimmer_FrameAdvancesOnTick(t *testing.T) {
	m0 := newSized(SkeletonConversation, 80, 5)
	m1 := advanceFrames(m0, totalFrames/4)

	if m0.frame == m1.frame {
		t.Errorf("frame should differ after %d advances: both are %d", totalFrames/4, m0.frame)
	}
	if m1.frame != totalFrames/4 {
		t.Errorf("expected frame=%d after %d advances, got %d", totalFrames/4, totalFrames/4, m1.frame)
	}
}

// ---------------------------------------------------------------------------
// View — table-driven variant × width × height
// ---------------------------------------------------------------------------

func TestView_AllVariants_TableDriven(t *testing.T) {
	type tc struct {
		variant SkeletonVariant
		name    string
		width   int
		height  int
	}
	tests := []tc{
		{SkeletonConversation, "Conversation 80x20", 80, 20},
		{SkeletonConversation, "Conversation 40x5", 40, 5},
		{SkeletonConversation, "Conversation 10x3", 10, 3},
		{SkeletonAgentTree, "AgentTree 80x20", 80, 20},
		{SkeletonAgentTree, "AgentTree 120x10", 120, 10},
		{SkeletonSettings, "Settings 80x12", 80, 12},
		{SkeletonSettings, "Settings 40x6", 40, 6},
		{SkeletonDashboard, "Dashboard 80x8", 80, 8},
		{SkeletonDashboard, "Dashboard 120x4", 120, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newSized(tt.variant, tt.width, tt.height)
			got := m.View()
			if got == "" {
				t.Errorf("%s: expected non-empty view", tt.name)
			}
			lines := lineCount(got)
			if lines != tt.height {
				t.Errorf("%s: expected %d lines, got %d", tt.name, tt.height, lines)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// renderLine edge cases — very narrow widths
// ---------------------------------------------------------------------------

func TestView_VeryNarrowWidth_NoPanic(t *testing.T) {
	// Width narrower than any indent should not panic.
	for _, w := range []int{1, 2, 3, 4, 5} {
		m := newSized(SkeletonAgentTree, w, 4)
		_ = m.View() // just verify no panic
	}
}

// ---------------------------------------------------------------------------
// renderSettingsRow — key+value columns
// ---------------------------------------------------------------------------

func TestView_Settings_RendersAtHeight1(t *testing.T) {
	m := newSized(SkeletonSettings, 80, 1)
	got := m.View()
	if got == "" {
		t.Error("Settings variant with height=1 should render a non-empty line")
	}
	if strings.Contains(got, "\n") {
		t.Error("height=1 should produce exactly one line (no newlines)")
	}
}

// ---------------------------------------------------------------------------
// Active flag preserved through Update
// ---------------------------------------------------------------------------

func TestUpdate_ActivePreservedAfterTick(t *testing.T) {
	m := New(SkeletonConversation)
	m, _ = m.Update(util.AnimateTickMsg{})
	if !m.Active() {
		t.Error("model should remain active after one AnimateTickMsg")
	}
}

// ---------------------------------------------------------------------------
// Shimmer position boundary — frame 0 vs frame near totalFrames
// ---------------------------------------------------------------------------

func TestShimmer_FrameNearEnd_NoPanic(t *testing.T) {
	m := newSized(SkeletonConversation, 80, 6)
	m = advanceFrames(m, totalFrames-1)
	// Must not panic at the last frame before wrap.
	_ = m.View()
}

func TestShimmer_FrameAtWrap_NoPanic(t *testing.T) {
	m := newSized(SkeletonConversation, 80, 6)
	m = advanceFrames(m, totalFrames)
	_ = m.View()
}

// ---------------------------------------------------------------------------
// Dashboard — grid-style rendering at height 1 and 4
// ---------------------------------------------------------------------------

func TestView_Dashboard_Height1(t *testing.T) {
	m := newSized(SkeletonDashboard, 80, 1)
	got := m.View()
	if got == "" {
		t.Error("Dashboard height=1 should be non-empty")
	}
}

func TestView_Dashboard_Height4_FourLines(t *testing.T) {
	m := newSized(SkeletonDashboard, 80, 4)
	got := m.View()
	if lineCount(got) != 4 {
		t.Errorf("Dashboard height=4: expected 4 lines, got %d", lineCount(got))
	}
}
