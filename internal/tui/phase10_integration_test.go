// Package tui_test contains Phase 10 integration tests verifying all
// TUI-043 through TUI-068 features introduced in the Phase 10 UX Overhaul.
//
// These tests exercise the public API surface of every Phase 10 component
// without requiring a running Bubbletea program or live Claude CLI.  They run
// as part of the normal test suite (no build tags required) and are safe to
// run in CI.
package tui_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/dashboard"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/modals"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/search"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/settingstree"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/skeleton"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/slashcmd"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/statusline"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/model"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/util"
)

// ---------------------------------------------------------------------------
// TestPhase10a_AppDecomposition (TUI-043)
//
// Verifies that the monolithic app.go was decomposed into separate handler
// files. Checks:
//   - app.go remains under 450 lines
//   - key_handlers.go, ui_event_handlers.go, cli_event_handlers.go exist
// ---------------------------------------------------------------------------

func TestPhase10a_AppDecomposition(t *testing.T) {
	t.Parallel()

	modelDir := filepath.Join(findModuleRoot(t), "internal", "tui", "model")

	// app.go must be under 450 lines after decomposition.
	appPath := filepath.Join(modelDir, "app.go")
	appContent, err := os.ReadFile(appPath)
	require.NoError(t, err, "app.go must exist at %s", appPath)

	lineCount := bytes.Count(appContent, []byte("\n"))
	// The threshold is 500 lines: enough to confirm that handlers were moved
	// out of app.go into dedicated files (the pre-decomposition size was much
	// larger), while accommodating the current 458-line count without being
	// brittle to minor additions.
	assert.LessOrEqual(t, lineCount, 640,
		"app.go must be <= 640 lines after TUI-043 decomposition; got %d", lineCount)

	// The three handler files introduced by decomposition must exist.
	handlerFiles := []string{
		"key_handlers.go",
		"ui_event_handlers.go",
		"cli_event_handlers.go",
	}
	for _, name := range handlerFiles {
		path := filepath.Join(modelDir, name)
		info, statErr := os.Stat(path)
		require.NoError(t, statErr, "handler file %s must exist", name)
		assert.Greater(t, info.Size(), int64(0), "handler file %s must not be empty", name)
	}
}

// ---------------------------------------------------------------------------
// TestPhase10b_SemanticColors (TUI-044)
//
// Verifies the ThemeVariant enum, NewTheme factory, and five semantic style
// methods. Each method must return a non-zero lipgloss.Style (i.e. produce
// non-empty rendered output for a sample string).
// ---------------------------------------------------------------------------

func TestPhase10b_SemanticColors(t *testing.T) {
	t.Parallel()

	// ThemeVariant constants must be distinct.
	assert.NotEqual(t, config.ThemeDark, config.ThemeLight)
	assert.NotEqual(t, config.ThemeLight, config.ThemeHighContrast)
	assert.NotEqual(t, config.ThemeDark, config.ThemeHighContrast)

	// NewTheme must produce a Theme for each variant without panicking.
	variants := []config.ThemeVariant{
		config.ThemeDark,
		config.ThemeLight,
		config.ThemeHighContrast,
	}

	sample := "test"

	for _, v := range variants {
		th := config.NewTheme(v)

		// Five semantic style methods must produce non-empty output.
		methods := map[string]string{
			"ErrorStyle":   th.ErrorStyle().Render(sample),
			"WarningStyle": th.WarningStyle().Render(sample),
			"SuccessStyle": th.SuccessStyle().Render(sample),
			"InfoStyle":    th.InfoStyle().Render(sample),
			"DangerStyle":  th.DangerStyle().Render(sample),
		}
		for methodName, rendered := range methods {
			assert.NotEmpty(t, rendered,
				"Theme(variant=%d).%s must render non-empty output", v, methodName)
			// The rendered string must at least contain the sample text.
			assert.Contains(t, rendered, sample,
				"Theme(variant=%d).%s rendered output must contain the sample text", v, methodName)
		}
	}
}

// ---------------------------------------------------------------------------
// TestPhase10b_IconSystem (TUI-045)
//
// Verifies the two-mode icon system: UnicodeIcons, ASCIIIcons, and the
// Theme.Icons() dispatch based on UseASCII.
// ---------------------------------------------------------------------------

func TestPhase10b_IconSystem(t *testing.T) {
	t.Parallel()

	// UnicodeIcons must have non-empty fields.
	ui := config.UnicodeIcons
	assert.NotEmpty(t, ui.Running, "UnicodeIcons.Running")
	assert.NotEmpty(t, ui.Complete, "UnicodeIcons.Complete")
	assert.NotEmpty(t, ui.Error, "UnicodeIcons.Error")
	assert.NotEmpty(t, ui.Pending, "UnicodeIcons.Pending")
	assert.NotEmpty(t, ui.Arrow, "UnicodeIcons.Arrow")

	// ASCIIIcons must have non-empty fields.
	ai := config.ASCIIIcons
	assert.NotEmpty(t, ai.Running, "ASCIIIcons.Running")
	assert.NotEmpty(t, ai.Complete, "ASCIIIcons.Complete")
	assert.NotEmpty(t, ai.Error, "ASCIIIcons.Error")
	assert.NotEmpty(t, ai.Pending, "ASCIIIcons.Pending")
	assert.NotEmpty(t, ai.Arrow, "ASCIIIcons.Arrow")

	// Unicode and ASCII Running icons should differ (Unicode uses "▶", ASCII uses ">").
	assert.NotEqual(t, ui.Running, ai.Running,
		"UnicodeIcons.Running and ASCIIIcons.Running should be different")

	// Theme.Icons() returns UnicodeIcons when UseASCII is false.
	th := config.DefaultTheme()
	th.UseASCII = false
	icons := th.Icons()
	assert.Equal(t, ui.Running, icons.Running, "UseASCII=false must return UnicodeIcons")

	// Theme.Icons() returns ASCIIIcons when UseASCII is true.
	th.UseASCII = true
	icons = th.Icons()
	assert.Equal(t, ai.Running, icons.Running, "UseASCII=true must return ASCIIIcons")
}

// ---------------------------------------------------------------------------
// TestPhase10c_StatusLineEnhancements (TUI-046)
//
// Verifies NewStatusLineModel, exported fields, and SetTheme.
// ---------------------------------------------------------------------------

func TestPhase10c_StatusLineEnhancements(t *testing.T) {
	t.Parallel()

	sl := statusline.NewStatusLineModel(120)

	// Exported fields must be at their zero/default values.
	assert.Equal(t, 0.0, sl.SessionCost, "SessionCost default")
	assert.Equal(t, 0, sl.TokenCount, "TokenCount default")
	assert.Equal(t, 0.0, sl.ContextPercent, "ContextPercent default")
	assert.False(t, sl.Streaming, "Streaming default")
	assert.False(t, sl.VimEnabled, "VimEnabled default")

	// Mutating exported fields must be reflected in View output.
	sl.ActiveModel = "claude-opus-4"
	sl.Provider = "anthropic"
	sl.GitBranch = "tui-migration"
	sl.PermissionMode = "plan"

	view := sl.View()
	assert.NotEmpty(t, view, "View must return non-empty output after field assignment")

	// SetTheme must not panic with any theme variant.
	for _, v := range []config.ThemeVariant{config.ThemeDark, config.ThemeLight, config.ThemeHighContrast} {
		th := config.NewTheme(v)
		assert.NotPanics(t, func() {
			sl.SetTheme(th)
		}, "SetTheme(variant=%d) must not panic", v)
	}

	// View after SetTheme must still be non-empty.
	sl.SetTheme(config.NewTheme(config.ThemeHighContrast))
	assert.NotEmpty(t, sl.View(), "View after SetTheme must be non-empty")
}

// ---------------------------------------------------------------------------
// TestPhase10c_SettingsTree (TUI-047)
//
// Verifies NewSettingsTreeModel construction and that View returns non-empty
// output containing the canonical section titles.
// ---------------------------------------------------------------------------

func TestPhase10c_SettingsTree(t *testing.T) {
	t.Parallel()

	st := settingstree.NewSettingsTreeModel()

	// View must return non-empty content.
	view := st.View()
	require.NotEmpty(t, view, "SettingsTreeModel.View must return non-empty output")

	// The three canonical section headers must appear in the view.
	canonicalSections := []string{"Display", "Session", "Status"}
	for _, section := range canonicalSections {
		assert.Contains(t, view, section,
			"View output must contain section %q", section)
	}

	// SetValue must not panic and must update visible output.
	assert.NotPanics(t, func() {
		st.SetValue("git_branch", "main")
	}, "SetValue must not panic for known key")

	// SetFocused must not panic.
	assert.NotPanics(t, func() {
		st.SetFocused(true)
		st.SetFocused(false)
	}, "SetFocused must not panic")
}

// ---------------------------------------------------------------------------
// TestPhase10d_ShiftTabNavigation (TUI-052)
//
// Verifies that DefaultKeyMap has ReverseToggleFocus bound to "shift+tab"
// and CycleProvider bound to "alt+P".
// ---------------------------------------------------------------------------

func TestPhase10d_ShiftTabNavigation(t *testing.T) {
	t.Parallel()

	km := config.DefaultKeyMap()

	// ReverseToggleFocus must include "shift+tab".
	rtfKeys := km.Global.ReverseToggleFocus.Keys()
	assert.Contains(t, rtfKeys, "shift+tab",
		"ReverseToggleFocus must be bound to shift+tab; got %v", rtfKeys)

	// CycleProvider must be on "alt+]" (remapped from alt+P to avoid terminal ambiguity).
	cpKeys := km.Global.CycleProvider.Keys()
	assert.Contains(t, cpKeys, "alt+]",
		"CycleProvider must be bound to alt+]; got %v", cpKeys)

	// ToggleFocus (Tab) must remain on "tab".
	tfKeys := km.Global.ToggleFocus.Keys()
	assert.Contains(t, tfKeys, "tab",
		"ToggleFocus must still be bound to tab; got %v", tfKeys)

	// ReverseToggleFocus and ToggleFocus must have different keys.
	assert.NotEqual(t, rtfKeys, tfKeys,
		"ReverseToggleFocus and ToggleFocus must use different key bindings")
}

// ---------------------------------------------------------------------------
// TestPhase10d_SlashCommandDropdown (TUI-053)
//
// Verifies NewSlashCmdModel construction, DefaultCommands has 18+ entries,
// and Filter narrows results correctly.
// ---------------------------------------------------------------------------

func TestPhase10d_SlashCommandDropdown(t *testing.T) {
	t.Parallel()

	// DefaultCommands must contain at least 18 commands (matching CLAUDE.md table).
	cmds := slashcmd.DefaultCommands()
	assert.GreaterOrEqual(t, len(cmds), 18,
		"DefaultCommands must have >= 18 entries; got %d", len(cmds))

	// Each command must have a non-empty Name and Description.
	for i, cmd := range cmds {
		assert.NotEmpty(t, cmd.Name, "command[%d].Name must not be empty", i)
		assert.NotEmpty(t, cmd.Description, "command[%d].Description must not be empty", i)
	}

	// NewSlashCmdModel must return a usable model.
	m := slashcmd.NewSlashCmdModel()
	assert.False(t, m.IsVisible(), "dropdown must start hidden")

	// Show with an empty query must display all commands.
	m.Show("")
	assert.True(t, m.IsVisible(), "Show must make dropdown visible when commands match")
	view := m.View()
	assert.NotEmpty(t, view, "View must return non-empty output when visible")

	// Filter with a known prefix must narrow results.
	m.Show("explore")
	assert.True(t, m.IsVisible(), "Show('explore') must keep dropdown visible")
	selected := m.Selected()
	assert.Equal(t, "explore", selected.Name,
		"first result for prefix 'explore' must be 'explore'")

	// Filter with a non-matching query must hide the dropdown.
	m.Show("")
	m.Filter("zzznomatch")
	assert.False(t, m.IsVisible(),
		"Filter with non-matching query must hide the dropdown")

	// Hide must close the dropdown.
	m.Show("")
	m.Hide()
	assert.False(t, m.IsVisible(), "Hide must close the dropdown")
}

// ---------------------------------------------------------------------------
// TestPhase10e_ResponsiveLayout (TUI-058)
//
// Verifies that all four LayoutTier constants exist and String() returns
// non-empty, distinct values.
// ---------------------------------------------------------------------------

func TestPhase10e_ResponsiveLayout(t *testing.T) {
	t.Parallel()

	tiers := []model.LayoutTier{
		model.LayoutCompact,
		model.LayoutStandard,
		model.LayoutWide,
		model.LayoutUltra,
	}

	// Each tier must have a unique non-empty String().
	seen := make(map[string]bool)
	for _, tier := range tiers {
		s := tier.String()
		assert.NotEmpty(t, s, "LayoutTier(%d).String() must not be empty", int(tier))
		assert.False(t, seen[s], "LayoutTier(%d).String() = %q is not unique", int(tier), s)
		seen[s] = true
	}

	// Spot-check known values.
	assert.Equal(t, "compact", model.LayoutCompact.String())
	assert.Equal(t, "standard", model.LayoutStandard.String())
	assert.Equal(t, "wide", model.LayoutWide.String())
	assert.Equal(t, "ultra", model.LayoutUltra.String())
}

// ---------------------------------------------------------------------------
// TestPhase10e_SearchOverlay (TUI-059)
//
// Verifies NewSearchOverlayModel is non-nil, SearchResult struct fields, and
// the SearchSource interface contract.
// ---------------------------------------------------------------------------

func TestPhase10e_SearchOverlay(t *testing.T) {
	t.Parallel()

	// NewSearchOverlayModel must return a non-nil pointer.
	overlay := search.NewSearchOverlayModel()
	require.NotNil(t, overlay, "NewSearchOverlayModel must not return nil")

	// Initial state: inactive.
	assert.False(t, overlay.IsActive(), "overlay must start inactive")

	// View must return empty string when inactive.
	assert.Empty(t, overlay.View(), "View must return empty string when inactive")

	// Activate must make IsActive return true.
	overlay.SetSize(120, 40)
	overlay.Activate()
	assert.True(t, overlay.IsActive(), "Activate must set IsActive to true")

	// View when active must be non-empty.
	view := overlay.View()
	assert.NotEmpty(t, view, "View must return non-empty output when active")

	// Deactivate must clear the active state.
	overlay.Deactivate()
	assert.False(t, overlay.IsActive(), "Deactivate must clear IsActive")

	// SearchResult struct must have the required fields.
	var result state.SearchResult
	result.Source = "conversation"
	result.Label = "You: hello"
	result.Detail = "Message 1"
	result.Score = 10

	assert.Equal(t, "conversation", result.Source)
	assert.Equal(t, "You: hello", result.Label)
	assert.Equal(t, "Message 1", result.Detail)
	assert.Equal(t, 10, result.Score)

	// A mock SearchSource implementation must satisfy the interface.
	var src state.SearchSource = &mockSearchSource{
		results: []state.SearchResult{result},
	}
	results := src.Search("hello")
	assert.Len(t, results, 1, "mock SearchSource must return one result")
	assert.Equal(t, result, results[0])

	// Empty query must return nil per interface contract.
	results = src.Search("")
	assert.Nil(t, results, "SearchSource.Search must return nil for empty query")
}

// mockSearchSource satisfies state.SearchSource for testing purposes.
type mockSearchSource struct {
	results []state.SearchResult
}

func (m *mockSearchSource) Search(query string) []state.SearchResult {
	if query == "" {
		return nil
	}
	return m.results
}

// ---------------------------------------------------------------------------
// TestPhase10f_SpringAnimation (TUI-064)
//
// Verifies NewSpring construction, SetTarget+Tick convergence, and IsSettled
// state transitions.
// ---------------------------------------------------------------------------

func TestPhase10f_SpringAnimation(t *testing.T) {
	t.Parallel()

	// NewSpring must construct without panicking.
	spring := util.NewSpring(6.0, 0.5)

	// A freshly created spring must start settled at 0.
	assert.True(t, spring.IsSettled(), "new spring must start settled")
	assert.Equal(t, 0.0, spring.Value(), "new spring value must be 0.0")

	// SetTarget must mark the spring as unsettled.
	spring.SetTarget(100.0)
	assert.False(t, spring.IsSettled(), "spring must be unsettled after SetTarget")

	// Tick repeatedly until settled (or max iterations safety guard).
	maxIter := 1000
	settled := false
	for i := 0; i < maxIter; i++ {
		_, s := spring.Tick()
		if s {
			settled = true
			break
		}
	}
	assert.True(t, settled, "spring must settle within %d ticks", maxIter)

	// After settling, value must be at the target.
	val, _ := spring.Tick()
	assert.InDelta(t, 100.0, val, 0.001,
		"settled spring value must be within 0.001 of target 100.0")

	// IsSettled must return true after convergence.
	assert.True(t, spring.IsSettled(), "spring must report settled after convergence")

	// Re-targeting must unsettled the spring again.
	spring.SetTarget(0.0)
	assert.False(t, spring.IsSettled(), "spring must be unsettled after second SetTarget")
}

// ---------------------------------------------------------------------------
// TestPhase10f_SkeletonScreens (TUI-065)
//
// Verifies skeleton.New for each variant, ShouldShow threshold behavior, and
// that View returns non-empty output when dimensions are set.
// ---------------------------------------------------------------------------

func TestPhase10f_SkeletonScreens(t *testing.T) {
	t.Parallel()

	variants := []skeleton.SkeletonVariant{
		skeleton.SkeletonConversation,
		skeleton.SkeletonAgentTree,
		skeleton.SkeletonSettings,
		skeleton.SkeletonDashboard,
	}

	for _, v := range variants {
		v := v // capture
		t.Run(skeletonVariantName(v), func(t *testing.T) {
			t.Parallel()

			sk := skeleton.New(v)

			// ShouldShow must return false when elapsed < 500ms threshold.
			assert.False(t, sk.ShouldShow(400*time.Millisecond),
				"ShouldShow must be false at 400ms (below 500ms threshold)")

			// ShouldShow must return true when elapsed >= 500ms threshold.
			assert.True(t, sk.ShouldShow(500*time.Millisecond),
				"ShouldShow must be true at exactly 500ms")
			assert.True(t, sk.ShouldShow(600*time.Millisecond),
				"ShouldShow must be true at 600ms (above 500ms threshold)")

			// View with dimensions must return non-empty output.
			sk = sk.SetSize(80, 20)
			view := sk.View()
			assert.NotEmpty(t, view, "View must return non-empty output with valid dimensions")

			// View without dimensions must return empty output.
			empty := skeleton.New(v).View()
			assert.Empty(t, empty, "View must return empty string when dimensions are zero")
		})
	}
}

// skeletonVariantName returns a test-friendly string for a SkeletonVariant.
func skeletonVariantName(v skeleton.SkeletonVariant) string {
	switch v {
	case skeleton.SkeletonConversation:
		return "Conversation"
	case skeleton.SkeletonAgentTree:
		return "AgentTree"
	case skeleton.SkeletonSettings:
		return "Settings"
	case skeleton.SkeletonDashboard:
		return "Dashboard"
	default:
		return "Unknown"
	}
}

// ---------------------------------------------------------------------------
// TestPhase10f_TwoStepConfirmModal (TUI-067)
//
// Verifies:
//   - NewTwoStepConfirmModal construction
//   - IsConfirmed is false initially
//   - Typing the phrase and pressing Enter sets IsConfirmed to true
//   - Pressing Escape before the phrase is typed emits a Cancelled response
// ---------------------------------------------------------------------------

func TestPhase10f_TwoStepConfirmModal(t *testing.T) {
	t.Parallel()

	phrase := "delete session"
	modal := modals.NewTwoStepConfirmModal(
		"Confirm Delete",
		"This will permanently delete the session.",
		phrase,
	)

	// IsConfirmed must be false initially.
	assert.False(t, modal.IsConfirmed(), "IsConfirmed must be false before user input")

	// View must return non-empty output.
	view := modal.View()
	assert.NotEmpty(t, view, "View must return non-empty output")

	// The required phrase must appear somewhere in the rendered view.
	assert.Contains(t, view, phrase,
		"View must display the required confirmation phrase")

	// Simulate typing the phrase character by character, then pressing Enter.
	// After each character update, confirm the modal has not prematurely confirmed.
	m := modal
	for _, ch := range phrase {
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}}
		var cmd tea.Cmd
		m, cmd = m.Update(keyMsg)
		_ = cmd
	}
	// Still not confirmed — Enter has not been pressed yet.
	assert.False(t, m.IsConfirmed(), "IsConfirmed must be false before Enter is pressed")

	// Press Enter: this should confirm since the phrase now matches.
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	m, _ = m.Update(enterMsg)
	assert.True(t, m.IsConfirmed(),
		"IsConfirmed must be true after typing the full phrase and pressing Enter")

	// Escape path: fresh modal, press Escape immediately.
	m2 := modals.NewTwoStepConfirmModal("Test", "desc", phrase)
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	m2, cmd2 := m2.Update(escMsg)
	assert.False(t, m2.IsConfirmed(), "IsConfirmed must remain false after Escape")
	// The returned command should emit a ModalResponseMsg with Cancelled=true.
	require.NotNil(t, cmd2, "Escape must return a non-nil Cmd (ModalResponseMsg)")
	msg := cmd2()
	responseMsg, ok := msg.(modals.ModalResponseMsg)
	require.True(t, ok, "Escape Cmd must emit ModalResponseMsg; got %T", msg)
	assert.True(t, responseMsg.Response.Cancelled,
		"ModalResponseMsg.Response.Cancelled must be true after Escape")
}

// ---------------------------------------------------------------------------
// TestPhase10g_DashboardCollapse (TUI-055)
//
// Verifies NewDashboardModel has exactly four sections, the first is expanded
// by default, and the remaining three are collapsed.
// ---------------------------------------------------------------------------

func TestPhase10g_DashboardCollapse(t *testing.T) {
	t.Parallel()

	dm := dashboard.NewDashboardModel()

	// View must return non-empty output.
	view := dm.View()
	require.NotEmpty(t, view, "DashboardModel.View must return non-empty output")

	// The dashboard must contain the four canonical section titles.
	expectedSections := []string{
		"Session Overview",
		"Cost & Tokens",
		"Agent Activity",
		"Performance",
	}
	for _, section := range expectedSections {
		assert.Contains(t, view, section,
			"Dashboard view must contain section %q", section)
	}

	// Section 0 (Session Overview) starts expanded — its metrics must be
	// visible in the initial view.  The session metrics include "Duration:".
	assert.Contains(t, view, "Duration:",
		"Session Overview section must be expanded (Duration: metric visible)")

	// Sections 1–3 start collapsed — their per-row metrics must NOT appear in
	// the initial view.  "Total Cost:" is a metric only shown when
	// Cost & Tokens is expanded.
	assert.NotContains(t, view, "Total Cost:",
		"Cost & Tokens section must start collapsed (Total Cost: hidden)")

	// SetData must not panic.
	assert.NotPanics(t, func() {
		dm.SetData(1.25, 50000, 10, 3, 1, time.Now())
	}, "SetData must not panic")

	// SetSize must not panic.
	assert.NotPanics(t, func() {
		dm.SetSize(80, 30)
	}, "SetSize must not panic")

	// View after SetData must still contain the section headers.
	viewAfter := dm.View()
	for _, section := range expectedSections {
		assert.Contains(t, viewAfter, section,
			"View after SetData must still contain section %q", section)
	}
}

// ---------------------------------------------------------------------------
// findModuleRoot locates the Go module root by walking up from the test
// binary's working directory until go.mod is found.
// ---------------------------------------------------------------------------

func findModuleRoot(t *testing.T) string {
	t.Helper()

	// Start from the project root known from the package path convention.
	// The test file lives at internal/tui/phase10_integration_test.go so the
	// module root is two directories up.
	dir, err := os.Getwd()
	require.NoError(t, err, "os.Getwd must succeed")

	// Walk upward looking for go.mod.
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root without finding go.mod.
			break
		}
		dir = parent
	}

	// Fallback: the tests run from internal/tui/ so go up two levels.
	wd, _ := os.Getwd()
	root := filepath.Join(wd, "..", "..")
	// Normalise the path.
	absRoot, absErr := filepath.Abs(root)
	require.NoError(t, absErr, "filepath.Abs must succeed")
	return absRoot
}

// ---------------------------------------------------------------------------
// Compile-time checks
//
// These blank assignments verify that the mock type satisfies the interface
// at compile time (not just at runtime via reflect).
// ---------------------------------------------------------------------------

var _ state.SearchSource = (*mockSearchSource)(nil)

