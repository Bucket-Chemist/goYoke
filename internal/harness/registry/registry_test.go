package registry_test

import (
	"sort"
	"testing"

	"github.com/Bucket-Chemist/goYoke/internal/harness/registry"
)

func TestNew(t *testing.T) {
	r := registry.New()
	if r == nil {
		t.Fatal("New returned nil")
	}
	if got := r.List(); len(got) != 0 {
		t.Errorf("new registry should be empty; got %d entries", len(got))
	}
}

func TestRegisterAndGet(t *testing.T) {
	r := registry.New()
	p := registry.Provider{
		Name:         "hermes",
		SupportLevel: registry.SupportLevelSupported,
		Capabilities: []string{"submit_prompt", "get_snapshot"},
	}
	r.Register(p)

	got, ok := r.Get("hermes")
	if !ok {
		t.Fatal("expected provider to be found after Register")
	}
	if got.Name != "hermes" {
		t.Errorf("Name: got %q, want %q", got.Name, "hermes")
	}
	if got.SupportLevel != registry.SupportLevelSupported {
		t.Errorf("SupportLevel: got %q, want %q", got.SupportLevel, registry.SupportLevelSupported)
	}
	if len(got.Capabilities) != 2 {
		t.Errorf("Capabilities: got %d, want 2", len(got.Capabilities))
	}
}

func TestGetMissingProvider(t *testing.T) {
	r := registry.New()
	_, ok := r.Get("nonexistent")
	if ok {
		t.Error("Get of unregistered provider should return false")
	}
}

func TestRegisterReplaces(t *testing.T) {
	r := registry.New()
	r.Register(registry.Provider{Name: "p1", SupportLevel: registry.SupportLevelExperimental})
	r.Register(registry.Provider{
		Name:         "p1",
		SupportLevel: registry.SupportLevelSupported,
		Capabilities: []string{"new_cap"},
	})

	got, ok := r.Get("p1")
	if !ok {
		t.Fatal("p1 not found after re-register")
	}
	if got.SupportLevel != registry.SupportLevelSupported {
		t.Errorf("SupportLevel after replace: got %q, want supported", got.SupportLevel)
	}
	if len(got.Capabilities) != 1 || got.Capabilities[0] != "new_cap" {
		t.Errorf("Capabilities after replace: got %v", got.Capabilities)
	}
}

func TestList(t *testing.T) {
	r := registry.New()
	r.Register(registry.Provider{Name: "a", SupportLevel: registry.SupportLevelSupported})
	r.Register(registry.Provider{Name: "b", SupportLevel: registry.SupportLevelManual})
	r.Register(registry.Provider{Name: "c", SupportLevel: registry.SupportLevelExperimental})

	list := r.List()
	if len(list) != 3 {
		t.Fatalf("List: got %d, want 3", len(list))
	}

	names := make([]string, len(list))
	for i, p := range list {
		names[i] = p.Name
	}
	sort.Strings(names)
	expected := []string{"a", "b", "c"}
	for i, n := range expected {
		if names[i] != n {
			t.Errorf("List[%d]: got %q, want %q", i, names[i], n)
		}
	}
}

func TestListEmpty(t *testing.T) {
	r := registry.New()
	list := r.List()
	if list == nil {
		t.Error("List on empty registry should return non-nil slice")
	}
	if len(list) != 0 {
		t.Errorf("List on empty registry: got %d, want 0", len(list))
	}
}

func TestSupportedProviders(t *testing.T) {
	r := registry.New()
	r.Register(registry.Provider{Name: "s1", SupportLevel: registry.SupportLevelSupported})
	r.Register(registry.Provider{Name: "e1", SupportLevel: registry.SupportLevelExperimental})
	r.Register(registry.Provider{Name: "m1", SupportLevel: registry.SupportLevelManual})
	r.Register(registry.Provider{Name: "s2", SupportLevel: registry.SupportLevelSupported})

	supported := r.SupportedProviders()
	if len(supported) != 2 {
		t.Fatalf("SupportedProviders: got %d, want 2", len(supported))
	}
	for _, p := range supported {
		if p.SupportLevel != registry.SupportLevelSupported {
			t.Errorf("SupportedProviders returned non-supported: %+v", p)
		}
	}
}

func TestSupportedProvidersNoneMatch(t *testing.T) {
	r := registry.New()
	r.Register(registry.Provider{Name: "m1", SupportLevel: registry.SupportLevelManual})
	r.Register(registry.Provider{Name: "e1", SupportLevel: registry.SupportLevelExperimental})

	supported := r.SupportedProviders()
	if len(supported) != 0 {
		t.Errorf("SupportedProviders with no supported entries: got %d, want 0", len(supported))
	}
}

func TestSupportLevelConstants(t *testing.T) {
	if registry.SupportLevelSupported == "" {
		t.Error("SupportLevelSupported must not be empty")
	}
	if registry.SupportLevelExperimental == "" {
		t.Error("SupportLevelExperimental must not be empty")
	}
	if registry.SupportLevelManual == "" {
		t.Error("SupportLevelManual must not be empty")
	}
	// All three constants must be distinct.
	if registry.SupportLevelSupported == registry.SupportLevelExperimental ||
		registry.SupportLevelSupported == registry.SupportLevelManual ||
		registry.SupportLevelExperimental == registry.SupportLevelManual {
		t.Error("SupportLevel constants must all be distinct")
	}
}

func TestCapabilitiesStoredCorrectly(t *testing.T) {
	r := registry.New()
	caps := []string{"submit_prompt", "get_snapshot", "interrupt"}
	r.Register(registry.Provider{
		Name:         "full",
		SupportLevel: registry.SupportLevelSupported,
		Capabilities: caps,
	})

	got, ok := r.Get("full")
	if !ok {
		t.Fatal("provider not found")
	}
	if len(got.Capabilities) != len(caps) {
		t.Fatalf("Capabilities: got %d, want %d", len(got.Capabilities), len(caps))
	}
	for i, c := range caps {
		if got.Capabilities[i] != c {
			t.Errorf("Capabilities[%d]: got %q, want %q", i, got.Capabilities[i], c)
		}
	}
}
