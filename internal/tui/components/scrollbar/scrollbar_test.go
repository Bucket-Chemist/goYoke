package scrollbar_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/goYoke/internal/tui/components/scrollbar"
)

// lineCount counts the number of lines in s by counting "\n" separators.
// Works correctly on ANSI-styled output because lipgloss does not inject "\n"
// into rendered cells; our join uses plain "\n" as the only separator.
func lineCount(s string) int {
	return strings.Count(s, "\n") + 1
}

func TestRender(t *testing.T) {
	t.Run("no scroll needed – content equals viewport", func(t *testing.T) {
		if got := scrollbar.Render(10, 10, 0); got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})

	t.Run("no scroll needed – content smaller than viewport", func(t *testing.T) {
		if got := scrollbar.Render(10, 5, 0); got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})

	t.Run("small content – line count matches viewportHeight", func(t *testing.T) {
		result := scrollbar.Render(10, 20, 0)
		if result == "" {
			t.Fatal("expected non-empty string")
		}
		if got := lineCount(result); got != 10 {
			t.Errorf("expected 10 lines, got %d", got)
		}
	})

	t.Run("large content – line count matches viewportHeight", func(t *testing.T) {
		result := scrollbar.Render(20, 1000, 0)
		if result == "" {
			t.Fatal("expected non-empty string")
		}
		if got := lineCount(result); got != 20 {
			t.Errorf("expected 20 lines, got %d", got)
		}
	})

	t.Run("thumb minimum – very large content forces thumb size to 1", func(t *testing.T) {
		// viewportHeight=5, contentHeight=10000 → thumbSize = 5*5/10000 = 0 → clamped to 1
		result := scrollbar.Render(5, 10000, 0)
		if result == "" {
			t.Fatal("expected non-empty string")
		}
		if got := lineCount(result); got != 5 {
			t.Errorf("expected 5 lines, got %d", got)
		}
	})

	t.Run("top position – scrollOffset zero", func(t *testing.T) {
		result := scrollbar.Render(10, 40, 0)
		if result == "" {
			t.Fatal("expected non-empty string")
		}
		if got := lineCount(result); got != 10 {
			t.Errorf("expected 10 lines, got %d", got)
		}
	})

	t.Run("bottom position – scrollOffset at maximum", func(t *testing.T) {
		viewportH := 10
		contentH := 40
		maxOffset := contentH - viewportH // 30
		result := scrollbar.Render(viewportH, contentH, maxOffset)
		if result == "" {
			t.Fatal("expected non-empty string")
		}
		if got := lineCount(result); got != viewportH {
			t.Errorf("expected %d lines, got %d", viewportH, got)
		}
	})

	t.Run("middle position – line count unchanged", func(t *testing.T) {
		result := scrollbar.Render(15, 60, 20)
		if result == "" {
			t.Fatal("expected non-empty string")
		}
		if got := lineCount(result); got != 15 {
			t.Errorf("expected 15 lines, got %d", got)
		}
	})

	t.Run("scrollOffset clamped below zero", func(t *testing.T) {
		result := scrollbar.Render(10, 30, -5)
		if result == "" {
			t.Fatal("expected non-empty string")
		}
		if got := lineCount(result); got != 10 {
			t.Errorf("expected 10 lines, got %d", got)
		}
	})

	t.Run("scrollOffset clamped above maxOffset", func(t *testing.T) {
		result := scrollbar.Render(10, 30, 9999)
		if result == "" {
			t.Fatal("expected non-empty string")
		}
		if got := lineCount(result); got != 10 {
			t.Errorf("expected 10 lines, got %d", got)
		}
	})
}

func TestRenderStyled(t *testing.T) {
	trackColor := lipgloss.AdaptiveColor{Light: "8", Dark: "8"}
	thumbColor := lipgloss.AdaptiveColor{Light: "6", Dark: "6"}

	t.Run("no scroll needed", func(t *testing.T) {
		if got := scrollbar.RenderStyled(10, 10, 0, trackColor, thumbColor); got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})

	t.Run("returns correct line count", func(t *testing.T) {
		result := scrollbar.RenderStyled(10, 40, 15, trackColor, thumbColor)
		if result == "" {
			t.Fatal("expected non-empty string")
		}
		if got := lineCount(result); got != 10 {
			t.Errorf("expected 10 lines, got %d", got)
		}
	})
}
