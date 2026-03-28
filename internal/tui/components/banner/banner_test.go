package banner_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/banner"
)

func TestNewBannerModel(t *testing.T) {
	m := banner.NewBannerModel(80)
	if m == (banner.BannerModel{}) {
		t.Fatal("NewBannerModel returned zero value")
	}
}

func TestBannerInit(t *testing.T) {
	m := banner.NewBannerModel(80)
	cmd := m.Init()
	if cmd != nil {
		t.Error("Init should return nil command")
	}
}

func TestBannerViewContainsTitle(t *testing.T) {
	m := banner.NewBannerModel(80)
	view := m.View()
	if !strings.Contains(view, "GOgent-Fortress") {
		t.Errorf("View() does not contain 'GOgent-Fortress'; got:\n%s", view)
	}
}

func TestBannerViewThreeRows(t *testing.T) {
	m := banner.NewBannerModel(80)
	view := m.View()
	// A rounded border box always has exactly 3 lines:
	// top-border, content, bottom-border.
	// Strip any trailing newline before counting.
	view = strings.TrimRight(view, "\n")
	lines := strings.Split(view, "\n")
	if len(lines) != 3 {
		t.Errorf("View() should produce 3 rows; got %d:\n%s", len(lines), view)
	}
}

func TestBannerUpdateWindowSizeMsg(t *testing.T) {
	m := banner.NewBannerModel(80)
	newModel, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if cmd != nil {
		t.Error("Update with WindowSizeMsg should return nil command")
	}
	// After update the new model should render without panicking.
	view := newModel.(banner.BannerModel).View()
	if !strings.Contains(view, "GOgent-Fortress") {
		t.Errorf("View() after resize does not contain title; got:\n%s", view)
	}
}

func TestBannerSetWidth(t *testing.T) {
	m := banner.NewBannerModel(80)
	m.SetWidth(100)
	view := m.View()
	if !strings.Contains(view, "GOgent-Fortress") {
		t.Errorf("View() after SetWidth does not contain title; got:\n%s", view)
	}
}

func TestBannerUpdateUnknownMsg(t *testing.T) {
	m := banner.NewBannerModel(80)
	// Unknown message types should be passed through without error.
	newModel, cmd := m.Update("some-string-message")
	if cmd != nil {
		t.Error("Update with unknown message should return nil command")
	}
	if newModel == nil {
		t.Error("Update must always return a non-nil model")
	}
}

func TestBannerNarrowWidth(t *testing.T) {
	// Ensure the banner does not panic on very narrow widths.
	m := banner.NewBannerModel(1)
	_ = m.View()
}
