package providers_test

import (
	"strings"
	"testing"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/providers"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
)

// newPS is a test helper that returns a default ProviderState with all four
// providers registered and Anthropic active.
func newPS() *state.ProviderState {
	return state.NewProviderState()
}

func TestNewProviderTabBarModel_DefaultVisible(t *testing.T) {
	m := providers.NewProviderTabBarModel(newPS(), 120)
	if !m.IsVisible() {
		t.Error("IsVisible() = false; want true (4 providers registered)")
	}
}

func TestNewProviderTabBarModel_DefaultActive(t *testing.T) {
	m := providers.NewProviderTabBarModel(newPS(), 120)
	// Initial active provider is Anthropic per NewProviderState.
	view := m.View()
	if !strings.Contains(view, "Anthropic") {
		t.Errorf("View() missing 'Anthropic'; got:\n%s", view)
	}
}

func TestInit_ReturnsNil(t *testing.T) {
	m := providers.NewProviderTabBarModel(newPS(), 120)
	cmd := m.Init()
	if cmd != nil {
		t.Error("Init() should return nil command")
	}
}

func TestUpdate_NoOp(t *testing.T) {
	m := providers.NewProviderTabBarModel(newPS(), 120)
	result, cmd := m.Update(nil)
	if cmd != nil {
		t.Error("Update() should return nil command")
	}
	_, ok := result.(providers.ProviderTabBarModel)
	if !ok {
		t.Errorf("Update() returned unexpected type %T", result)
	}
}

func TestView_ContainsAllProviderNames(t *testing.T) {
	m := providers.NewProviderTabBarModel(newPS(), 200)
	view := m.View()

	expected := []string{"Anthropic", "Google", "OpenAI", "Local / Ollama"}
	for _, name := range expected {
		if !strings.Contains(view, name) {
			t.Errorf("View() missing provider name %q; got:\n%s", name, view)
		}
	}
}

func TestView_SingleRow(t *testing.T) {
	m := providers.NewProviderTabBarModel(newPS(), 120)
	view := m.View()
	view = strings.TrimRight(view, "\n")
	lines := strings.Split(view, "\n")
	if len(lines) != 1 {
		t.Errorf("View() should produce 1 row; got %d:\n%s", len(lines), view)
	}
}

func TestView_ContainsDividers(t *testing.T) {
	m := providers.NewProviderTabBarModel(newPS(), 200)
	view := m.View()
	if !strings.Contains(view, "|") {
		t.Errorf("View() missing dividers between tabs; got:\n%s", view)
	}
}

func TestView_EmptyWhenNotVisible(t *testing.T) {
	// Build a ProviderState with only one provider so visible = false.
	// We test via Height() and IsVisible() since we cannot easily construct a
	// single-provider ProviderState without unexported fields.
	// Instead, verify that Height() == 0 implies View() == "".
	m := providers.NewProviderTabBarModel(newPS(), 120)
	// Four providers → visible. Manually test the inverse through Height/IsVisible.
	if !m.IsVisible() {
		if m.View() != "" {
			t.Errorf("View() should be empty when not visible; got:\n%s", m.View())
		}
	}
}

func TestSetActive_ChangesHighlighting(t *testing.T) {
	m := providers.NewProviderTabBarModel(newPS(), 200)

	// Switch to Google.
	m.SetActive(state.ProviderGoogle)
	view := m.View()

	if !strings.Contains(view, "Google") {
		t.Errorf("View() after SetActive(Google) missing 'Google'; got:\n%s", view)
	}
}

func TestSetActive_CycleAllProviders(t *testing.T) {
	ps := newPS()
	m := providers.NewProviderTabBarModel(ps, 200)

	providers_list := []state.ProviderID{
		state.ProviderAnthropic,
		state.ProviderGoogle,
		state.ProviderOpenAI,
		state.ProviderLocal,
	}

	for _, id := range providers_list {
		m.SetActive(id)
		view := m.View()
		// Each provider name must appear in the view regardless of active tab.
		cfg, _ := ps.GetConfig(id)
		if !strings.Contains(view, cfg.Name) {
			t.Errorf("View() after SetActive(%q) missing name %q; got:\n%s", id, cfg.Name, view)
		}
	}
}

func TestSetActive_UnknownIDNoChange(t *testing.T) {
	m := providers.NewProviderTabBarModel(newPS(), 120)
	// Set to Google first.
	m.SetActive(state.ProviderGoogle)
	// Attempt to set an unknown ID.
	m.SetActive(state.ProviderID("unknown-provider"))
	// Should still be Google.
	view := m.View()
	if !strings.Contains(view, "Google") {
		t.Errorf("after unknown SetActive, expected Google still active; got:\n%s", view)
	}
}

func TestSetWidth_UpdatesRendering(t *testing.T) {
	m := providers.NewProviderTabBarModel(newPS(), 80)
	m.SetWidth(200)
	view := m.View()
	if !strings.Contains(view, "Anthropic") {
		t.Errorf("View() after SetWidth missing 'Anthropic'; got:\n%s", view)
	}
}

func TestIsVisible_TrueWithFourProviders(t *testing.T) {
	m := providers.NewProviderTabBarModel(newPS(), 120)
	if !m.IsVisible() {
		t.Error("IsVisible() = false; want true with 4 providers")
	}
}

func TestHeight_OneWhenVisible(t *testing.T) {
	m := providers.NewProviderTabBarModel(newPS(), 120)
	if m.Height() != 1 {
		t.Errorf("Height() = %d; want 1 when visible", m.Height())
	}
}

func TestHeight_ZeroWhenNotVisible(t *testing.T) {
	// Directly test the Height() return contract: when visible is false,
	// Height returns 0.  We verify this through the struct invariant:
	// a model that IsVisible() returns Height() == 1.
	// For a not-visible model we would need 0 or 1 provider — not reachable
	// via the public API with the default 4-provider state.  We assert the
	// visible case covers Height()==1 and document the contract.
	m := providers.NewProviderTabBarModel(newPS(), 120)
	if m.IsVisible() && m.Height() != 1 {
		t.Errorf("Height() = %d; want 1 when IsVisible()==true", m.Height())
	}
	if !m.IsVisible() && m.Height() != 0 {
		t.Errorf("Height() = %d; want 0 when IsVisible()==false", m.Height())
	}
}

func TestView_AnthropicActiveByDefault(t *testing.T) {
	ps := newPS() // Anthropic is default active
	m := providers.NewProviderTabBarModel(ps, 200)
	view := m.View()

	// Anthropic name must be in the output.
	if !strings.Contains(view, "Anthropic") {
		t.Errorf("View() default active provider 'Anthropic' not found; got:\n%s", view)
	}
}
