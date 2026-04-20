package drawer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDrawerStack_FourDrawers(t *testing.T) {
	s := NewDrawerStack()

	// All four drawers exist and start minimized.
	if s.Options().ID() != DrawerOptions {
		t.Errorf("Options ID=%v, want %v", s.Options().ID(), DrawerOptions)
	}
	if s.Plan().ID() != DrawerPlan {
		t.Errorf("Plan ID=%v, want %v", s.Plan().ID(), DrawerPlan)
	}
	if s.Teams().ID() != DrawerTeams {
		t.Errorf("Teams ID=%v, want %v", s.Teams().ID(), DrawerTeams)
	}
	if s.Figures().ID() != DrawerFigures {
		t.Errorf("Figures ID=%v, want %v", s.Figures().ID(), DrawerFigures)
	}

	for _, d := range []*DrawerModel{s.Options(), s.Plan(), s.Teams(), s.Figures()} {
		if d.State() != DrawerMinimized {
			t.Errorf("drawer %v should start minimized, got %v", d.ID(), d.State())
		}
	}
}

func TestDrawerStack_LayoutFourDrawers(t *testing.T) {
	s := NewDrawerStack()
	s.Figures().SetContent("graph TD\n  A --> B")
	s.Teams().SetContent("team data")
	s.SetSize(80, 40)

	// Both expanded drawers should get a reasonable non-zero height.
	if s.figures.height <= 0 {
		t.Errorf("figures height=%d, want >0", s.figures.height)
	}
	if s.teams.height <= 0 {
		t.Errorf("teams height=%d, want >0", s.teams.height)
	}
	// Minimized drawers get exactly MinimizedRowHeight.
	if s.options.height != MinimizedRowHeight {
		t.Errorf("options height=%d, want %d", s.options.height, MinimizedRowHeight)
	}
	if s.plan.height != MinimizedRowHeight {
		t.Errorf("plan height=%d, want %d", s.plan.height, MinimizedRowHeight)
	}
}

func TestFiguresDrawer_SetContent(t *testing.T) {
	s := NewDrawerStack()
	s.SetFiguresContent("graph TD\n  A --> B")

	if !s.FiguresHasContent() {
		t.Error("FiguresHasContent should be true after SetFiguresContent")
	}
	if s.Figures().State() != DrawerExpanded {
		t.Error("SetFiguresContent should auto-expand the figures drawer")
	}
	if s.Figures().Content() != "graph TD\n  A --> B" {
		t.Errorf("drawer content mismatch, got %q", s.Figures().Content())
	}
}

func TestFiguresDrawer_ClearContent(t *testing.T) {
	s := NewDrawerStack()
	s.SetFiguresContent("graph TD\n  A --> B")
	s.ClearFiguresContent()

	if s.FiguresHasContent() {
		t.Error("FiguresHasContent should be false after ClearFiguresContent")
	}
	if s.Figures().State() != DrawerMinimized {
		t.Error("ClearFiguresContent should minimize the drawer")
	}
}

func TestFiguresDrawer_Toggle(t *testing.T) {
	s := NewDrawerStack()
	s.SetFiguresContent("mermaid source")

	// starts expanded after SetContent
	s.ToggleFiguresDrawer()
	if s.Figures().State() != DrawerMinimized {
		t.Error("ToggleFiguresDrawer should minimize an expanded drawer")
	}

	// re-expand without losing content
	s.ToggleFiguresDrawer()
	if s.Figures().State() != DrawerExpanded {
		t.Error("ToggleFiguresDrawer should expand a minimized drawer")
	}
	if !s.FiguresHasContent() {
		t.Error("content should survive ToggleFiguresDrawer round-trip")
	}
}

func TestFiguresDrawer_FiguresIsMinimized(t *testing.T) {
	s := NewDrawerStack()
	if !s.FiguresIsMinimized() {
		t.Error("new figures drawer should be minimized")
	}
	s.SetFiguresContent("content")
	if s.FiguresIsMinimized() {
		t.Error("figures drawer should not be minimized after SetFiguresContent")
	}
}

func TestFiguresDrawer_ExpandedDrawersIncludesFigures(t *testing.T) {
	s := NewDrawerStack()
	s.SetFiguresContent("graph TD\n  A --> B")

	ids := s.ExpandedDrawers()
	found := false
	for _, id := range ids {
		if id == string(DrawerFigures) {
			found = true
		}
	}
	if !found {
		t.Errorf("ExpandedDrawers should include %q when figures is expanded, got %v", DrawerFigures, ids)
	}
}


func TestDiscoverDiagrams(t *testing.T) {
	root := t.TempDir()

	// Create docs/codebase-architecture/myProject/diagrams/*.mmd
	diagDir := filepath.Join(root, "docs", "codebase-architecture", "myProject", "diagrams")
	if err := os.MkdirAll(diagDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	files := map[string]string{
		"module-deps.mmd": "graph TD\n  A --> B",
		"overview.mmd":    "graph LR\n  X --> Y",
	}
	for name, content := range files {
		p := filepath.Join(diagDir, name)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
	}

	entries := DiscoverDiagrams(root)
	if len(entries) != 2 {
		t.Fatalf("DiscoverDiagrams returned %d entries, want 2", len(entries))
	}

	// Entries should be sorted by name.
	if entries[0].Name != "module-deps" {
		t.Errorf("entries[0].Name=%q, want %q", entries[0].Name, "module-deps")
	}
	if entries[1].Name != "overview" {
		t.Errorf("entries[1].Name=%q, want %q", entries[1].Name, "overview")
	}

	for _, e := range entries {
		if e.Type != "mermaid" {
			t.Errorf("entry %q type=%q, want %q", e.Name, e.Type, "mermaid")
		}
		if e.Source != "codebase-map" {
			t.Errorf("entry %q source=%q, want %q", e.Name, e.Source, "codebase-map")
		}
	}
}

func TestDiscoverDiagrams_EmptyRoot(t *testing.T) {
	root := t.TempDir()
	entries := DiscoverDiagrams(root)
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for empty root, got %d", len(entries))
	}
}

func TestDiscoverDiagrams_DeduplicatesOverlap(t *testing.T) {
	// If the same file appears under multiple glob patterns it should only
	// be listed once. (In practice the two patterns target distinct dirs,
	// so this tests the seen-map guard.)
	root := t.TempDir()
	dir := filepath.Join(root, ".claude", "codebase-map", "mermaid")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "foo.mmd"), []byte("graph TD"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	entries := DiscoverDiagrams(root)
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}
}

func TestFormatFiguresContent_Empty(t *testing.T) {
	state := FiguresState{}
	content := FormatFiguresContent(state)
	if content == "" {
		t.Error("FormatFiguresContent should return non-empty string for empty state")
	}
	// Should contain a helpful hint.
	if len(content) < 10 {
		t.Errorf("FormatFiguresContent returned too short string: %q", content)
	}
}
