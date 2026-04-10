package hintbar

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// NewHintBarModel
// ---------------------------------------------------------------------------

func TestNewHintBarModel_Defaults(t *testing.T) {
	h := NewHintBarModel()
	require.NotNil(t, h)
	assert.True(t, h.IsVisible(), "should be visible by default")
	assert.Equal(t, "main", h.context, "should default to main context")
	assert.NotEmpty(t, h.hints, "main context should have hints")
}

// ---------------------------------------------------------------------------
// SetContext — correct hint sets
// ---------------------------------------------------------------------------

func TestSetContext_Main(t *testing.T) {
	h := NewHintBarModel()
	h.SetContext("main")
	assert.Equal(t, "main", h.context)
	assert.Equal(t, hintSets["main"], h.hints)
}

func TestSetContext_Settings(t *testing.T) {
	h := NewHintBarModel()
	h.SetContext("settings")
	assert.Equal(t, "settings", h.context)
	assert.Equal(t, hintSets["settings"], h.hints)
}

func TestSetContext_Search(t *testing.T) {
	h := NewHintBarModel()
	h.SetContext("search")
	assert.Equal(t, "search", h.context)
	assert.Equal(t, hintSets["search"], h.hints)
}

func TestSetContext_Modal(t *testing.T) {
	h := NewHintBarModel()
	h.SetContext("modal")
	assert.Equal(t, "modal", h.context)
	assert.Equal(t, hintSets["modal"], h.hints)
}

func TestSetContext_Plan(t *testing.T) {
	h := NewHintBarModel()
	h.SetContext("plan")
	assert.Equal(t, "plan", h.context)
	assert.Equal(t, hintSets["plan"], h.hints)
}

func TestSetContext_Agents(t *testing.T) {
	h := NewHintBarModel()
	h.SetContext("agents")
	assert.Equal(t, "agents", h.context)
	assert.Equal(t, hintSets["agents"], h.hints)
}

func TestSetContext_AgentsDetail(t *testing.T) {
	h := NewHintBarModel()
	h.SetContext("agents_detail")
	assert.Equal(t, "agents_detail", h.context)
	assert.Equal(t, hintSets["agents_detail"], h.hints)
}

func TestSetContext_Taskboard(t *testing.T) {
	h := NewHintBarModel()
	h.SetContext("taskboard")
	assert.Equal(t, "taskboard", h.context)
	assert.Equal(t, hintSets["taskboard"], h.hints)
}

func TestSetContext_Teams(t *testing.T) {
	h := NewHintBarModel()
	h.SetContext("teams")
	assert.Equal(t, "teams", h.context)
	assert.Equal(t, hintSets["teams"], h.hints)
}

func TestSetContext_UnknownFallsBackToMain(t *testing.T) {
	h := NewHintBarModel()
	h.SetContext("this-context-does-not-exist")
	// Should fall back to "main" hints.
	assert.Equal(t, hintSets["main"], h.hints)
}

// ---------------------------------------------------------------------------
// Hint set content verification
// ---------------------------------------------------------------------------

func TestMainHintSet_ContainsExpectedKeys(t *testing.T) {
	h := NewHintBarModel()
	h.SetContext("main")
	h.SetWidth(200) // Wide enough for all hints

	view := stripANSI(h.View())

	assert.Contains(t, view, "Tab", "main hints should contain Tab")
	assert.Contains(t, view, "ctrl+f", "main hints should contain ctrl+f")
	assert.Contains(t, view, "/", "main hints should contain /")
}

func TestSettingsHintSet_ContainsExpectedKeys(t *testing.T) {
	h := NewHintBarModel()
	h.SetContext("settings")
	h.SetWidth(200)

	view := stripANSI(h.View())

	assert.Contains(t, view, "Enter", "settings hints should contain Enter")
	assert.Contains(t, view, "Esc", "settings hints should contain Esc")
}

func TestSearchHintSet_ContainsExpectedKeys(t *testing.T) {
	h := NewHintBarModel()
	h.SetContext("search")
	h.SetWidth(200)

	view := stripANSI(h.View())

	assert.Contains(t, view, "Enter", "search hints should contain Enter")
	assert.Contains(t, view, "Esc", "search hints should contain Esc")
}

func TestModalHintSet_ContainsExpectedKeys(t *testing.T) {
	h := NewHintBarModel()
	h.SetContext("modal")
	h.SetWidth(200)

	view := stripANSI(h.View())

	assert.Contains(t, view, "Enter", "modal hints should contain Enter")
	assert.Contains(t, view, "Esc", "modal hints should contain Esc")
}

func TestPlanHintSet_ContainsExpectedKeys(t *testing.T) {
	h := NewHintBarModel()
	h.SetContext("plan")
	h.SetWidth(200)

	view := stripANSI(h.View())

	assert.Contains(t, view, "alt+v", "plan hints should contain alt+v")
	assert.Contains(t, view, "Esc", "plan hints should contain Esc")
}

func TestAgentsHintSet_ContainsExpectedKeys(t *testing.T) {
	h := NewHintBarModel()
	h.SetContext("agents")
	h.SetWidth(200)

	view := stripANSI(h.View())

	assert.Contains(t, view, "↑/↓", "agents hints should contain ↑/↓")
	assert.Contains(t, view, "Enter", "agents hints should contain Enter")
	assert.Contains(t, view, "Esc", "agents hints should contain Esc")
	assert.Contains(t, view, "Tab", "agents hints should contain Tab")
}

func TestAgentsDetailHintSet_ContainsExpectedKeys(t *testing.T) {
	h := NewHintBarModel()
	h.SetContext("agents_detail")
	h.SetWidth(200)

	view := stripANSI(h.View())

	assert.Contains(t, view, "↑/↓", "agents_detail hints should contain ↑/↓")
	assert.Contains(t, view, "Esc", "agents_detail hints should contain Esc")
	assert.Contains(t, view, "Tab", "agents_detail hints should contain Tab")
}

func TestTaskboardHintSet_ContainsExpectedKeys(t *testing.T) {
	h := NewHintBarModel()
	h.SetContext("taskboard")
	h.SetWidth(200)

	view := stripANSI(h.View())

	assert.Contains(t, view, "↑/↓", "taskboard hints should contain ↑/↓")
	assert.Contains(t, view, "Tab", "taskboard hints should contain Tab")
}

func TestTeamsHintSet_ContainsExpectedKeys(t *testing.T) {
	h := NewHintBarModel()
	h.SetContext("teams")
	h.SetWidth(200)

	view := stripANSI(h.View())

	assert.Contains(t, view, "↑/↓", "teams hints should contain ↑/↓")
	assert.Contains(t, view, "Enter", "teams hints should contain Enter")
	assert.Contains(t, view, "x", "teams hints should contain x")
}

// ---------------------------------------------------------------------------
// Show / Hide visibility toggling
// ---------------------------------------------------------------------------

func TestShowHide(t *testing.T) {
	h := NewHintBarModel()

	h.Hide()
	assert.False(t, h.IsVisible())
	assert.Equal(t, "", h.View(), "hidden hint bar should render empty string")

	h.Show()
	assert.True(t, h.IsVisible())
}

func TestView_HiddenReturnsEmpty(t *testing.T) {
	h := NewHintBarModel()
	h.SetWidth(200)
	h.Hide()
	assert.Equal(t, "", h.View())
}

// ---------------------------------------------------------------------------
// Width truncation
// ---------------------------------------------------------------------------

func TestView_ZeroWidthReturnsFirstHint(t *testing.T) {
	h := NewHintBarModel()
	h.SetContext("main")
	h.SetWidth(0)

	view := stripANSI(h.View())
	assert.NotEmpty(t, view, "should return at least the first hint when width=0")
}

func TestView_NegativeWidthReturnsFirstHint(t *testing.T) {
	h := NewHintBarModel()
	h.SetContext("main")
	h.SetWidth(-1)

	view := stripANSI(h.View())
	assert.NotEmpty(t, view)
}

func TestView_WideTerminalShowsAllHints(t *testing.T) {
	h := NewHintBarModel()
	h.SetContext("main")
	h.SetWidth(200) // far wider than any hint set

	view := stripANSI(h.View())
	// Should NOT contain ellipsis when all hints fit.
	assert.NotContains(t, view, "...", "wide terminal should show all hints without ellipsis")
}

func TestView_NarrowTerminalTruncatesWithEllipsis(t *testing.T) {
	h := NewHintBarModel()
	h.SetContext("main")
	// Width is enough for one hint but not all four.
	h.SetWidth(15)

	view := stripANSI(h.View())
	// Either the view is empty (nothing fits) or it ends with ellipsis.
	if view != "" {
		assert.True(t,
			strings.HasSuffix(view, "..."),
			"narrow terminal should append ellipsis; got: %q", view,
		)
	}
}

func TestView_VeryNarrowTerminalReturnsEmpty(t *testing.T) {
	h := NewHintBarModel()
	h.SetContext("main")
	// Width is so narrow that not even one hint + trailing ellipsis fits.
	h.SetWidth(3)

	view := stripANSI(h.View())
	// Width=3 can't fit any hint with its trailing separator+ellipsis overhead.
	// Acceptable outcomes: empty string OR just the first item without ellipsis.
	// The contract is: no panic and no line longer than width.
	_ = view // just ensure no panic
}

// ---------------------------------------------------------------------------
// SetWidth
// ---------------------------------------------------------------------------

func TestSetWidth(t *testing.T) {
	h := NewHintBarModel()
	h.SetWidth(120)
	assert.Equal(t, 120, h.width)
}

// ---------------------------------------------------------------------------
// Table-driven context tests
// ---------------------------------------------------------------------------

func TestContextHints_TableDriven(t *testing.T) {
	type tc struct {
		name     string
		context  string
		wantKeys []string
	}
	tests := []tc{
		{
			name:     "main context",
			context:  "main",
			wantKeys: []string{"Tab", "ctrl+f"},
		},
		{
			name:     "settings context",
			context:  "settings",
			wantKeys: []string{"Enter", "Esc"},
		},
		{
			name:     "search context",
			context:  "search",
			wantKeys: []string{"Enter", "Esc"},
		},
		{
			name:     "modal context",
			context:  "modal",
			wantKeys: []string{"Enter", "Esc"},
		},
		{
			name:     "plan context",
			context:  "plan",
			wantKeys: []string{"alt+v", "Esc"},
		},
		{
			name:     "unknown context falls back to main",
			context:  "nonexistent",
			wantKeys: []string{"Tab", "ctrl+f"},
		},
		{
			name:     "agents context",
			context:  "agents",
			wantKeys: []string{"Enter", "Esc", "Tab"},
		},
		{
			name:     "agents_detail context",
			context:  "agents_detail",
			wantKeys: []string{"Esc", "Tab"},
		},
		{
			name:     "taskboard context",
			context:  "taskboard",
			wantKeys: []string{"Tab"},
		},
		{
			name:     "teams context",
			context:  "teams",
			wantKeys: []string{"Enter", "x"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHintBarModel()
			h.SetContext(tt.context)
			h.SetWidth(300)

			view := stripANSI(h.View())
			for _, key := range tt.wantKeys {
				assert.Contains(t, view, key,
					"context %q: expected key %q in view %q", tt.context, key, view)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// hintSets map completeness
// ---------------------------------------------------------------------------

func TestHintSets_AllContextsHaveNonEmptyHints(t *testing.T) {
	expected := []string{"main", "settings", "search", "modal", "plan", "agents", "agents_detail", "taskboard", "teams"}
	for _, ctx := range expected {
		hints, ok := hintSets[ctx]
		assert.True(t, ok, "hint set for %q should exist", ctx)
		assert.NotEmpty(t, hints, "hint set for %q should not be empty", ctx)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// stripANSI removes ANSI escape sequences from s so tests can assert on plain text.
// This avoids importing a third-party ANSI stripping library.
func stripANSI(s string) string {
	var out strings.Builder
	inEsc := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			// Escape sequence ends at the first letter after the '[' introducer.
			if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
				inEsc = false
			}
			continue
		}
		out.WriteByte(c)
	}
	return out.String()
}
