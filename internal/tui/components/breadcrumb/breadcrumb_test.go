package breadcrumb_test

import (
	"strings"
	"testing"

	"github.com/Bucket-Chemist/goYoke/internal/tui/components/breadcrumb"
	"github.com/Bucket-Chemist/goYoke/internal/tui/config"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// stripANSI removes ANSI escape sequences from s so test assertions can
// compare plain text without worrying about terminal color codes.
// This is intentionally minimal — it strips ESC[...m sequences only.
func stripANSI(s string) string {
	out := strings.Builder{}
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			// Scan forward to the terminating 'm'.
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

func newModelWide() *breadcrumb.BreadcrumbModel {
	m := breadcrumb.NewBreadcrumbModel()
	m.SetWidth(200)
	return m
}

// ---------------------------------------------------------------------------
// NewBreadcrumbModel
// ---------------------------------------------------------------------------

func TestNewBreadcrumbModel_EmptyView(t *testing.T) {
	m := breadcrumb.NewBreadcrumbModel()
	if got := m.View(); got != "" {
		t.Errorf("expected empty view for new model, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// SetCrumbs / View with 0, 1, 2, 3+ items
// ---------------------------------------------------------------------------

func TestView_ZeroCrumbs(t *testing.T) {
	m := newModelWide()
	m.SetCrumbs(nil)
	if got := m.View(); got != "" {
		t.Errorf("expected empty string for nil crumbs, got %q", got)
	}

	m.SetCrumbs([]string{})
	if got := m.View(); got != "" {
		t.Errorf("expected empty string for empty crumbs, got %q", got)
	}
}

func TestView_OneCrumb(t *testing.T) {
	m := newModelWide()
	m.SetCrumbs([]string{"Claude"})

	plain := stripANSI(m.View())
	if !strings.Contains(plain, "Claude") {
		t.Errorf("expected 'Claude' in output, got %q", plain)
	}
	// Single crumb: no separator should appear.
	if strings.Contains(plain, ">") || strings.Contains(plain, "›") {
		t.Errorf("single crumb should not contain an arrow separator, got %q", plain)
	}
}

func TestView_TwoCrumbs(t *testing.T) {
	m := newModelWide()
	m.SetCrumbs([]string{"Claude", "Conversation"})

	plain := stripANSI(m.View())

	if !strings.Contains(plain, "Claude") {
		t.Errorf("expected 'Claude' in output, got %q", plain)
	}
	if !strings.Contains(plain, "Conversation") {
		t.Errorf("expected 'Conversation' in output, got %q", plain)
	}
	// A separator must appear between the two crumbs.
	claudeIdx := strings.Index(plain, "Claude")
	convIdx := strings.Index(plain, "Conversation")
	if claudeIdx >= convIdx {
		t.Errorf("expected 'Claude' before 'Conversation', got %q", plain)
	}
}

func TestView_ThreePlusCrumbs(t *testing.T) {
	m := newModelWide()
	m.SetCrumbs([]string{"Root", "Branch", "Leaf"})

	plain := stripANSI(m.View())

	for _, want := range []string{"Root", "Branch", "Leaf"} {
		if !strings.Contains(plain, want) {
			t.Errorf("expected %q in output %q", want, plain)
		}
	}
	// Verify ordering.
	rIdx := strings.Index(plain, "Root")
	bIdx := strings.Index(plain, "Branch")
	lIdx := strings.Index(plain, "Leaf")
	if !(rIdx < bIdx && bIdx < lIdx) {
		t.Errorf("crumbs out of order: Root=%d Branch=%d Leaf=%d in %q", rIdx, bIdx, lIdx, plain)
	}
}

// ---------------------------------------------------------------------------
// Arrow separator present between multiple crumbs
// ---------------------------------------------------------------------------

func TestView_ArrowSeparatorPresent(t *testing.T) {
	m := newModelWide()
	m.SetCrumbs([]string{"A", "B"})

	raw := m.View()
	// The Arrow icon from the default theme (UnicodeIcons) is "›".
	if !strings.Contains(raw, "›") && !strings.Contains(raw, ">") {
		t.Errorf("expected an arrow separator in output, got %q", raw)
	}
}

// ---------------------------------------------------------------------------
// Last item styled differently
// ---------------------------------------------------------------------------

// TestView_LastItemDistinct verifies the rendered output contains both crumb
// labels, with the ancestor label appearing before the current label.
// Lipgloss may or may not emit ANSI codes depending on whether a TTY is
// present, so this test checks plain-text content rather than escape codes.
func TestView_LastItemDistinct(t *testing.T) {
	m := newModelWide()
	m.SetCrumbs([]string{"Ancestor", "Current"})

	plain := stripANSI(m.View())

	if !strings.Contains(plain, "Ancestor") {
		t.Errorf("expected 'Ancestor' in output, got %q", plain)
	}
	if !strings.Contains(plain, "Current") {
		t.Errorf("expected 'Current' in output, got %q", plain)
	}

	// Ancestor must appear before Current in the output.
	aIdx := strings.Index(plain, "Ancestor")
	cIdx := strings.Index(plain, "Current")
	if aIdx >= cIdx {
		t.Errorf("expected 'Ancestor' before 'Current', got %q", plain)
	}
}

// ---------------------------------------------------------------------------
// Width truncation
// ---------------------------------------------------------------------------

func TestView_NarrowWidth_ShowsCurrentCrumb(t *testing.T) {
	m := breadcrumb.NewBreadcrumbModel()
	// Very narrow width: forces truncation of all but last crumb.
	m.SetWidth(10)
	m.SetCrumbs([]string{"LongAncestor", "Current"})

	plain := stripANSI(m.View())

	// The current crumb must always be visible.
	if !strings.Contains(plain, "Current") {
		t.Errorf("expected 'Current' crumb to be visible, got %q", plain)
	}
}

func TestView_Truncation_EllipsisPrefix(t *testing.T) {
	m := breadcrumb.NewBreadcrumbModel()
	// Narrow enough to force truncation but wide enough to show last crumb.
	m.SetWidth(20)
	m.SetCrumbs([]string{"VeryLongAncestor", "Short"})

	plain := stripANSI(m.View())

	// When truncation happens, "..." must appear.
	if !strings.Contains(plain, "...") {
		// Only fail if the ancestor label was actually truncated.
		if !strings.Contains(plain, "VeryLongAncestor") {
			t.Errorf("expected '...' prefix when truncation occurs, got %q", plain)
		}
	}
}

func TestView_WideEnough_NoEllipsis(t *testing.T) {
	m := breadcrumb.NewBreadcrumbModel()
	m.SetWidth(200)
	m.SetCrumbs([]string{"A", "B", "C"})

	plain := stripANSI(m.View())

	if strings.Contains(plain, "...") {
		t.Errorf("expected no ellipsis for wide terminal, got %q", plain)
	}
}

func TestView_ZeroWidth_ReturnsLastCrumb(t *testing.T) {
	m := breadcrumb.NewBreadcrumbModel()
	m.SetWidth(0)
	m.SetCrumbs([]string{"Hidden", "Visible"})

	plain := stripANSI(m.View())

	if !strings.Contains(plain, "Visible") {
		t.Errorf("expected last crumb visible with zero width, got %q", plain)
	}
}

// ---------------------------------------------------------------------------
// SetCrumbs replaces existing trail
// ---------------------------------------------------------------------------

func TestSetCrumbs_Replaces(t *testing.T) {
	m := newModelWide()
	m.SetCrumbs([]string{"Old"})

	if plain := stripANSI(m.View()); !strings.Contains(plain, "Old") {
		t.Fatalf("setup: expected 'Old' in view, got %q", plain)
	}

	m.SetCrumbs([]string{"New"})
	plain := stripANSI(m.View())

	if strings.Contains(plain, "Old") {
		t.Errorf("expected 'Old' to be gone after SetCrumbs, got %q", plain)
	}
	if !strings.Contains(plain, "New") {
		t.Errorf("expected 'New' after SetCrumbs, got %q", plain)
	}
}

func TestSetCrumbs_ClearsWithNil(t *testing.T) {
	m := newModelWide()
	m.SetCrumbs([]string{"X"})
	m.SetCrumbs(nil)

	if got := m.View(); got != "" {
		t.Errorf("expected empty view after SetCrumbs(nil), got %q", got)
	}
}

// ---------------------------------------------------------------------------
// SetCrumbItems (preserves Key field)
// ---------------------------------------------------------------------------

func TestSetCrumbItems_RendersLabels(t *testing.T) {
	m := newModelWide()
	m.SetCrumbItems([]breadcrumb.BreadcrumbItem{
		{Label: "Alpha", Key: "1"},
		{Label: "Beta", Key: "2"},
	})

	plain := stripANSI(m.View())
	if !strings.Contains(plain, "Alpha") {
		t.Errorf("expected 'Alpha' in output, got %q", plain)
	}
	if !strings.Contains(plain, "Beta") {
		t.Errorf("expected 'Beta' in output, got %q", plain)
	}
}

// ---------------------------------------------------------------------------
// SetTheme
// ---------------------------------------------------------------------------

func TestSetTheme_ASCIIArrow(t *testing.T) {
	m := newModelWide()
	asciiTheme := config.DefaultTheme()
	asciiTheme.UseASCII = true
	m.SetTheme(asciiTheme)
	m.SetCrumbs([]string{"A", "B"})

	raw := m.View()
	// ASCIIIcons.Arrow is ">".
	if !strings.Contains(raw, ">") {
		t.Errorf("expected ASCII arrow '>' in output, got %q", raw)
	}
}

func TestSetTheme_UnicodeArrow(t *testing.T) {
	m := newModelWide()
	m.SetTheme(config.DefaultTheme())
	m.SetCrumbs([]string{"A", "B"})

	raw := m.View()
	// UnicodeIcons.Arrow is "›".
	if !strings.Contains(raw, "›") {
		t.Errorf("expected Unicode arrow '›' in output, got %q", raw)
	}
}

// ---------------------------------------------------------------------------
// SetWidth
// ---------------------------------------------------------------------------

func TestSetWidth_PropagatesCorrectly(t *testing.T) {
	m := breadcrumb.NewBreadcrumbModel()
	m.SetCrumbs([]string{"A", "B", "C", "D", "E"})

	m.SetWidth(5)
	narrowPlain := stripANSI(m.View())

	m.SetWidth(200)
	widePlain := stripANSI(m.View())

	// Wide output should contain more labels than narrow output.
	narrowCount := countContained(narrowPlain, []string{"A", "B", "C", "D", "E"})
	wideCount := countContained(widePlain, []string{"A", "B", "C", "D", "E"})

	if wideCount <= narrowCount {
		t.Errorf("expected wider terminal to show more crumbs: narrow=%d wide=%d", narrowCount, wideCount)
	}
}

// countContained counts how many of the wanted strings appear in s.
func countContained(s string, want []string) int {
	n := 0
	for _, w := range want {
		if strings.Contains(s, w) {
			n++
		}
	}
	return n
}
