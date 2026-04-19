package drawer

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// ---- DrawerModel tests ----

func TestNewDrawerModelStartsMinimized(t *testing.T) {
	m := NewDrawerModel(DrawerOptions, "Options", "⚙")
	if m.State() != DrawerMinimized {
		t.Errorf("State()=%v, want DrawerMinimized", m.State())
	}
	if m.HasContent() {
		t.Error("new drawer should not have content")
	}
	if m.ID() != DrawerOptions {
		t.Errorf("ID()=%v, want DrawerOptions", m.ID())
	}
}

func TestDrawerModelSetContent(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		wantExpanded   bool
		wantHasContent bool
	}{
		{"non-empty content auto-expands", "hello", true, true},
		{"empty content stays minimized", "", false, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := NewDrawerModel(DrawerOptions, "Options", "⚙")
			m.SetContent(tc.content)
			if (m.State() == DrawerExpanded) != tc.wantExpanded {
				t.Errorf("expanded=%v, want %v", m.State() == DrawerExpanded, tc.wantExpanded)
			}
			if m.HasContent() != tc.wantHasContent {
				t.Errorf("HasContent()=%v, want %v", m.HasContent(), tc.wantHasContent)
			}
		})
	}
}

func TestDrawerModelClearContent(t *testing.T) {
	m := NewDrawerModel(DrawerOptions, "Options", "⚙")
	m.SetContent("some content") // auto-expands
	m.ClearContent()
	if m.State() != DrawerMinimized {
		t.Error("ClearContent should minimize the drawer")
	}
	if m.HasContent() {
		t.Error("ClearContent should remove content")
	}
}

func TestDrawerModelToggle(t *testing.T) {
	m := NewDrawerModel(DrawerOptions, "Options", "⚙")
	// starts minimized → toggle → expanded
	m.Toggle()
	if m.State() != DrawerExpanded {
		t.Error("Toggle from minimized should expand")
	}
	// expanded → toggle → minimized
	m.Toggle()
	if m.State() != DrawerMinimized {
		t.Error("Toggle from expanded should minimize")
	}
}

func TestDrawerModelExpandMinimize(t *testing.T) {
	m := NewDrawerModel(DrawerOptions, "Options", "⚙")
	m.Expand()
	if m.State() != DrawerExpanded {
		t.Error("Expand should set state to DrawerExpanded")
	}
	m.Minimize()
	if m.State() != DrawerMinimized {
		t.Error("Minimize should set state to DrawerMinimized")
	}
}

func TestDrawerModelFocus(t *testing.T) {
	m := NewDrawerModel(DrawerOptions, "Options", "⚙")
	if m.IsFocused() {
		t.Error("new drawer should not be focused")
	}
	m.SetFocused(true)
	if !m.IsFocused() {
		t.Error("SetFocused(true) should focus the drawer")
	}
	m.SetFocused(false)
	if m.IsFocused() {
		t.Error("SetFocused(false) should unfocus the drawer")
	}
}

func TestViewMinimizedNonEmpty(t *testing.T) {
	m := NewDrawerModel(DrawerOptions, "Options", "⚙")
	m.SetSize(40, 10)
	v := m.ViewMinimized()
	if v == "" {
		t.Error("ViewMinimized should return non-empty string")
	}
	if !strings.Contains(v, "Options") {
		t.Error("ViewMinimized should contain the drawer label")
	}
}

func TestViewExpandedNonEmpty(t *testing.T) {
	m := NewDrawerModel(DrawerOptions, "Options", "⚙")
	m.SetSize(80, 20)
	m.SetContent("test content")
	v := m.ViewExpanded()
	if v == "" {
		t.Error("ViewExpanded should return non-empty string")
	}
	if !strings.Contains(v, "Options") {
		t.Error("ViewExpanded should contain the drawer label")
	}
}

func TestViewExpandedFocusedStyle(t *testing.T) {
	m := NewDrawerModel(DrawerOptions, "Options", "⚙")
	m.SetSize(80, 20)
	m.SetContent("content")

	unfocused := m.ViewExpanded()
	m.SetFocused(true)
	focused := m.ViewExpanded()

	if unfocused == focused {
		t.Error("focused and unfocused ViewExpanded should produce different output")
	}
}

func TestViewDispatch(t *testing.T) {
	m := NewDrawerModel(DrawerOptions, "Options", "⚙")
	m.SetSize(80, 20)

	// minimized → delegates to ViewMinimized
	if m.View() != m.ViewMinimized() {
		t.Error("View() should equal ViewMinimized() when minimized")
	}

	// expanded → delegates to ViewExpanded
	m.SetContent("something")
	if m.View() != m.ViewExpanded() {
		t.Error("View() should equal ViewExpanded() when expanded")
	}
}

func TestHandleKeyEsc(t *testing.T) {
	m := NewDrawerModel(DrawerOptions, "Options", "⚙")
	m.SetContent("content") // auto-expands
	m.HandleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if m.State() != DrawerMinimized {
		t.Error("esc should minimize the drawer")
	}
}

func TestHandleKeyScrollDoesNotPanic(t *testing.T) {
	m := NewDrawerModel(DrawerOptions, "Options", "⚙")
	m.SetSize(80, 20)
	m.SetContent(strings.Repeat("line\n", 50))

	keys := []tea.KeyMsg{
		{Type: tea.KeyUp},
		{Type: tea.KeyDown},
		{Type: tea.KeyPgUp},
		{Type: tea.KeyPgDown},
		{Type: tea.KeyRunes, Runes: []rune{'k'}},
		{Type: tea.KeyRunes, Runes: []rune{'j'}},
	}
	for _, k := range keys {
		m.HandleKey(k) // must not panic
	}
}

func TestHandleKeyUnknown(t *testing.T) {
	m := NewDrawerModel(DrawerOptions, "Options", "⚙")
	cmd := m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if cmd != nil {
		t.Error("unknown key should return nil cmd")
	}
}

// ---- DrawerStack tests ----

func TestNewDrawerStackCreatesTwoDrawers(t *testing.T) {
	s := NewDrawerStack()
	if s.Options().ID() != DrawerOptions {
		t.Errorf("Options drawer ID=%v, want DrawerOptions", s.Options().ID())
	}
	if s.Plan().ID() != DrawerPlan {
		t.Errorf("Plan drawer ID=%v, want DrawerPlan", s.Plan().ID())
	}
	if s.Options().State() != DrawerMinimized {
		t.Error("Options drawer should start minimized")
	}
	if s.Plan().State() != DrawerMinimized {
		t.Error("Plan drawer should start minimized")
	}
}

func TestExpandedDrawers(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*DrawerStack)
		wantLen int
		wantIDs []string
	}{
		{
			name:    "both minimized returns empty",
			setup:   func(_ *DrawerStack) {},
			wantLen: 0,
			wantIDs: nil,
		},
		{
			name:    "options expanded only",
			setup:   func(s *DrawerStack) { s.Options().Expand() },
			wantLen: 1,
			wantIDs: []string{string(DrawerOptions)},
		},
		{
			name:    "plan expanded only",
			setup:   func(s *DrawerStack) { s.Plan().Expand() },
			wantLen: 1,
			wantIDs: []string{string(DrawerPlan)},
		},
		{
			name: "both expanded",
			setup: func(s *DrawerStack) {
				s.Options().Expand()
				s.Plan().Expand()
			},
			wantLen: 2,
			wantIDs: []string{string(DrawerOptions), string(DrawerPlan)},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := NewDrawerStack()
			tc.setup(&s)
			ids := s.ExpandedDrawers()
			if len(ids) != tc.wantLen {
				t.Errorf("len(ExpandedDrawers())=%d, want %d (got %v)", len(ids), tc.wantLen, ids)
				return
			}
			for i, want := range tc.wantIDs {
				if ids[i] != want {
					t.Errorf("ids[%d]=%v, want %v", i, ids[i], want)
				}
			}
		})
	}
}

func TestSetSizeHeightDistribution(t *testing.T) {
	// With 4 drawers stacked vertically, each minimized drawer = 3 rows
	// (border top + label + border bottom). Expanded drawers split the
	// remaining height; remainder goes to the first expanded.
	tests := []struct {
		name      string
		setup     func(*DrawerStack)
		h         int
		wantOptH  int
		wantPlanH int
	}{
		{
			name:      "all minimized each gets 3 rows",
			setup:     func(_ *DrawerStack) {},
			h:         20,
			wantOptH:  3,
			wantPlanH: 3,
		},
		{
			// h=20, 3 minimized (plan+teams+figures) = 9 rows, options = 20-9 = 11.
			name:      "options expanded plan minimized",
			setup:     func(s *DrawerStack) { s.Options().Expand() },
			h:         20,
			wantOptH:  11,
			wantPlanH: 3,
		},
		{
			// h=20, 3 minimized (options+teams+figures) = 9 rows, plan = 20-9 = 11.
			name:      "plan expanded options minimized",
			setup:     func(s *DrawerStack) { s.Plan().Expand() },
			h:         20,
			wantOptH:  3,
			wantPlanH: 11,
		},
		{
			// h=20, 2 minimized (teams+figures) = 6 rows, expanded = 14, 14/2=7 each.
			name: "both expanded teams minimized",
			setup: func(s *DrawerStack) {
				s.Options().Expand()
				s.Plan().Expand()
			},
			h:         20,
			wantOptH:  7,
			wantPlanH: 7,
		},
		{
			// h=21, 2 minimized (teams+figures) = 6 rows, expanded = 15, 15/2=7 rem 1 → opt=8, plan=7.
			name: "both expanded even split",
			setup: func(s *DrawerStack) {
				s.Options().Expand()
				s.Plan().Expand()
			},
			h:         21,
			wantOptH:  8,
			wantPlanH: 7,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := NewDrawerStack()
			tc.setup(&s)
			s.SetSize(80, tc.h)
			if s.options.height != tc.wantOptH {
				t.Errorf("options height=%d, want %d", s.options.height, tc.wantOptH)
			}
			if s.plan.height != tc.wantPlanH {
				t.Errorf("plan height=%d, want %d", s.plan.height, tc.wantPlanH)
			}
		})
	}
}

func TestProxyMethodsOptions(t *testing.T) {
	s := NewDrawerStack()

	s.SetOptionsContent("options content")
	if !s.OptionsHasContent() {
		t.Error("OptionsHasContent should be true after SetOptionsContent")
	}
	if s.Options().State() != DrawerExpanded {
		t.Error("SetOptionsContent should auto-expand options drawer")
	}

	s.ClearOptionsContent()
	if s.OptionsHasContent() {
		t.Error("OptionsHasContent should be false after ClearOptionsContent")
	}
	if s.Options().State() != DrawerMinimized {
		t.Error("ClearOptionsContent should auto-minimize options drawer")
	}
}

func TestProxyMethodsPlan(t *testing.T) {
	s := NewDrawerStack()

	s.SetPlanContent("plan content")
	if !s.PlanHasContent() {
		t.Error("PlanHasContent should be true after SetPlanContent")
	}
	if s.Plan().State() != DrawerExpanded {
		t.Error("SetPlanContent should auto-expand plan drawer")
	}

	s.ClearPlanContent()
	if s.PlanHasContent() {
		t.Error("PlanHasContent should be false after ClearPlanContent")
	}
	if s.Plan().State() != DrawerMinimized {
		t.Error("ClearPlanContent should auto-minimize plan drawer")
	}
}

func TestSetFocusedProxy(t *testing.T) {
	s := NewDrawerStack()

	s.SetOptionsFocused(true)
	if !s.Options().IsFocused() {
		t.Error("SetOptionsFocused(true) should focus options drawer")
	}

	s.SetPlanFocused(true)
	if !s.Plan().IsFocused() {
		t.Error("SetPlanFocused(true) should focus plan drawer")
	}
}

func TestDrawerStackViewNonEmpty(t *testing.T) {
	s := NewDrawerStack()
	s.SetSize(80, 40)

	// Both minimized: tabs rendered
	if s.View() == "" {
		t.Error("View() should be non-empty when both minimized")
	}

	// One expanded: expanded pane + one tab
	s.SetOptionsContent("some options")
	if s.View() == "" {
		t.Error("View() should be non-empty when options expanded")
	}

	// Both expanded
	s.SetPlanContent("some plan")
	if s.View() == "" {
		t.Error("View() should be non-empty when both expanded")
	}
}

// ---- Text-input modal mode tests ----

func TestSetActiveModalTextInputMode(t *testing.T) {
	m := NewDrawerModel(DrawerOptions, "Options", "⚙")
	m.SetActiveModal("req-1", "What is your name?", nil)

	if !m.HasActiveModal() {
		t.Error("HasActiveModal should be true after SetActiveModal with no options")
	}
	if !m.IsTextInputMode() {
		t.Error("IsTextInputMode should be true when options is nil")
	}
	if m.State() != DrawerExpanded {
		t.Error("SetActiveModal should expand the drawer")
	}
	if !m.HasContent() {
		t.Error("SetActiveModal should set hasContent")
	}
	if !strings.Contains(m.Content(), "What is your name?") {
		t.Error("drawer content should contain the modal message")
	}
	if !strings.Contains(m.Content(), "[Enter] Submit") {
		t.Error("drawer content should contain the submit hint")
	}
}

func TestSetActiveModalEmptySliceIsTextInputMode(t *testing.T) {
	m := NewDrawerModel(DrawerOptions, "Options", "⚙")
	m.SetActiveModal("req-2", "Confirm?", []string{})

	if !m.IsTextInputMode() {
		t.Error("IsTextInputMode should be true when options is empty slice")
	}
}

func TestIsTextInputModeFalseWhenOptionsPresent(t *testing.T) {
	m := NewDrawerModel(DrawerOptions, "Options", "⚙")
	m.SetActiveModal("req-3", "Choose:", []string{"A", "B"})

	if m.IsTextInputMode() {
		t.Error("IsTextInputMode should be false when options are present")
	}
}

func TestTextInputModeKeyTypingUpdatesContent(t *testing.T) {
	m := NewDrawerModel(DrawerOptions, "Options", "⚙")
	m.SetSize(80, 20)
	m.SetActiveModal("req-4", "Enter value:", nil)

	// Type 'h', 'i'
	m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})

	if !strings.Contains(m.Content(), "hi") {
		t.Errorf("drawer content should contain typed text 'hi', got: %q", m.Content())
	}
}

func TestTextInputModeEnterSubmitsValue(t *testing.T) {
	m := NewDrawerModel(DrawerOptions, "Options", "⚙")
	m.SetSize(80, 20)
	m.SetActiveModal("req-5", "What?", nil)

	m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})

	cmd := m.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter in text-input mode should return a non-nil cmd")
	}

	msg := cmd()
	resp, ok := msg.(ModalResponseMsg)
	if !ok {
		t.Fatalf("cmd() should return ModalResponseMsg, got %T", msg)
	}
	if resp.RequestID != "req-5" {
		t.Errorf("RequestID=%q, want %q", resp.RequestID, "req-5")
	}
	if resp.Value != "ok" {
		t.Errorf("Value=%q, want %q", resp.Value, "ok")
	}
	if resp.Cancelled {
		t.Error("Cancelled should be false on Enter submit")
	}

	// Modal should be cleared after submit
	if m.HasActiveModal() {
		t.Error("HasActiveModal should be false after Enter submit")
	}
}

func TestTextInputModeEscCancels(t *testing.T) {
	m := NewDrawerModel(DrawerOptions, "Options", "⚙")
	m.SetSize(80, 20)
	m.SetActiveModal("req-6", "Prompt?", nil)

	cmd := m.HandleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("Esc in text-input mode should return a non-nil cmd")
	}

	msg := cmd()
	resp, ok := msg.(ModalResponseMsg)
	if !ok {
		t.Fatalf("cmd() should return ModalResponseMsg, got %T", msg)
	}
	if resp.RequestID != "req-6" {
		t.Errorf("RequestID=%q, want %q", resp.RequestID, "req-6")
	}
	if !resp.Cancelled {
		t.Error("Cancelled should be true on Esc")
	}
	if resp.Value != "" {
		t.Errorf("Value should be empty on cancel, got %q", resp.Value)
	}
}

func TestDrawerStackHandleKey(t *testing.T) {
	s := NewDrawerStack()
	s.SetSize(80, 20)

	// Route esc to options: should minimize it
	s.SetOptionsContent("content")
	s.HandleKey(string(DrawerOptions), tea.KeyMsg{Type: tea.KeyEsc})
	if s.Options().State() != DrawerMinimized {
		t.Error("esc routed to options should minimize options drawer")
	}

	// Route esc to plan: should minimize it
	s.SetPlanContent("content")
	s.HandleKey(string(DrawerPlan), tea.KeyMsg{Type: tea.KeyEsc})
	if s.Plan().State() != DrawerMinimized {
		t.Error("esc routed to plan should minimize plan drawer")
	}

	// Unknown drawer: returns nil, no panic
	cmd := s.HandleKey("unknown", tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		t.Error("HandleKey with unknown drawer should return nil")
	}
}
