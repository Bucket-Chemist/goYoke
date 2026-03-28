package planpreview

import (
	"strings"
	"testing"
)

func TestNewPlanPreviewModel(t *testing.T) {
	m := NewPlanPreviewModel()
	if m.hasContent {
		t.Error("expected hasContent=false on new model")
	}
	if m.content != "" {
		t.Errorf("expected empty content, got %q", m.content)
	}
}

func TestSetSize(t *testing.T) {
	m := NewPlanPreviewModel()
	m.SetSize(80, 24)
	if m.width != 80 {
		t.Errorf("expected width 80, got %d", m.width)
	}
	if m.height != 24 {
		t.Errorf("expected height 24, got %d", m.height)
	}
	if m.viewport.Width != 80 {
		t.Errorf("expected viewport width 80, got %d", m.viewport.Width)
	}
}

func TestSetContent(t *testing.T) {
	m := NewPlanPreviewModel()
	m.SetSize(80, 24)
	m.SetContent("# My Plan\n\nStep 1")
	if !m.hasContent {
		t.Error("expected hasContent=true after SetContent")
	}
	if m.content != "# My Plan\n\nStep 1" {
		t.Errorf("unexpected content: %q", m.content)
	}
	if m.rendered == "" {
		t.Error("expected rendered to be non-empty after SetContent")
	}
}

func TestSetContent_Empty(t *testing.T) {
	m := NewPlanPreviewModel()
	m.SetContent("")
	if m.hasContent {
		t.Error("expected hasContent=false for empty content")
	}
}

func TestClearContent(t *testing.T) {
	m := NewPlanPreviewModel()
	m.SetSize(80, 24)
	m.SetContent("# Plan\n\nsome content")
	if !m.hasContent {
		t.Fatal("precondition: hasContent should be true")
	}
	m.ClearContent()
	if m.hasContent {
		t.Error("expected hasContent=false after ClearContent")
	}
	if m.content != "" {
		t.Errorf("expected empty content after ClearContent, got %q", m.content)
	}
	if m.rendered != "" {
		t.Errorf("expected empty rendered after ClearContent, got %q", m.rendered)
	}
}

func TestView_NoContent(t *testing.T) {
	m := NewPlanPreviewModel()
	view := m.View()
	if !strings.Contains(view, "No plan loaded") {
		t.Errorf("expected 'No plan loaded' when empty, got:\n%s", view)
	}
}

func TestView_ContainsHeader(t *testing.T) {
	m := NewPlanPreviewModel()
	view := m.View()
	if !strings.Contains(view, "Plan Preview") {
		t.Errorf("expected 'Plan Preview' header, got:\n%s", view)
	}
}

func TestView_WithContent(t *testing.T) {
	m := NewPlanPreviewModel()
	m.SetSize(80, 30)
	m.SetContent("# Implementation Plan\n\n- Step 1\n- Step 2\n")
	view := m.View()
	// Should contain the header and not the "No plan loaded" placeholder.
	if strings.Contains(view, "No plan loaded") {
		t.Errorf("should not show 'No plan loaded' when content is set, got:\n%s", view)
	}
	if !strings.Contains(view, "Plan Preview") {
		t.Errorf("expected 'Plan Preview' header, got:\n%s", view)
	}
}

func TestSetSize_ReRendersContent(t *testing.T) {
	m := NewPlanPreviewModel()
	m.SetSize(80, 24)
	m.SetContent("# Hello\n\nsome text")
	rendered80 := m.rendered

	// Resize to a different width — should trigger re-render.
	m.SetSize(60, 24)
	if m.rendered == "" {
		t.Error("expected re-render after SetSize when content is present")
	}
	// Rendered output may differ at different widths.
	_ = rendered80 // just ensure no panic
}

func TestDivider(t *testing.T) {
	tests := []struct {
		width    int
		expected int
	}{
		{0, 20},
		{30, 30},
		{50, 40}, // capped
	}
	for _, tc := range tests {
		d := divider(tc.width)
		got := len([]rune(d))
		if got != tc.expected {
			t.Errorf("divider(%d): expected %d runes, got %d", tc.width, tc.expected, got)
		}
	}
}
