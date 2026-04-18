// Package providers implements the provider tab-bar component for the
// GOgent-Fortress TUI. It renders a single-row strip of provider names and
// highlights the currently active provider.
//
// This component is display-only: key handling (Alt+] cycling) is wired
// in model/app.go and communicated back via SetActive.
package providers

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/goYoke/internal/tui/config"
	"github.com/Bucket-Chemist/goYoke/internal/tui/state"
)

// ProviderTabBarModel is the Bubbletea model for the horizontal provider tab
// bar. It renders a single-row strip of provider labels. The active provider
// is highlighted; all others are styled as subtle.
//
// The zero value is not usable; use NewProviderTabBarModel instead.
type ProviderTabBarModel struct {
	providers []state.ProviderID        // ordered list; matches state.AllProviders()
	names     map[state.ProviderID]string // display names from ProviderConfig
	active    state.ProviderID           // currently highlighted provider
	width     int                        // terminal width for full-row background fill
	visible   bool                       // hidden when ≤1 provider registered
}

// NewProviderTabBarModel returns a ProviderTabBarModel initialised from ps.
// It reads the ordered provider list from ps.AllProviders() and the display
// names from ps.GetConfig(id).Name for each provider.
//
// The tab bar is visible only when more than one provider is available
// (with the default four-provider ProviderState this is always true).
func NewProviderTabBarModel(ps *state.ProviderState, width int) ProviderTabBarModel {
	ids := ps.AllProviders()

	names := make(map[state.ProviderID]string, len(ids))
	for _, id := range ids {
		if cfg, ok := ps.GetConfig(id); ok {
			names[id] = cfg.Name
		}
	}

	return ProviderTabBarModel{
		providers: ids,
		names:     names,
		active:    ps.GetActiveProvider(),
		width:     width,
		visible:   len(ids) > 1,
	}
}

// Init implements tea.Model. The provider tab bar requires no startup commands.
func (m ProviderTabBarModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model. The provider tab bar handles no messages
// directly; all key events are handled in AppModel.Update and applied through
// SetActive. Window resizing is applied through SetWidth.
func (m ProviderTabBarModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

// View implements tea.Model. It renders a single-row horizontal strip of
// provider labels. The active provider is styled with config.StyleHighlight;
// inactive providers use config.StyleSubtle. Providers are separated by a
// vertical bar divider.
//
// Returns an empty string when the tab bar is not visible (≤1 provider).
func (m ProviderTabBarModel) View() string {
	if !m.visible {
		return ""
	}

	var parts []string
	for _, id := range m.providers {
		name, ok := m.names[id]
		if !ok {
			name = string(id)
		}

		var label string
		if id == m.active {
			label = config.StyleHighlight.Padding(0, 1).Render(name)
		} else {
			label = config.StyleSubtle.Padding(0, 1).Render(name)
		}
		parts = append(parts, label)
	}

	divider := config.StyleSubtle.Render("|")
	row := strings.Join(parts, divider)

	// Pad the row to the full terminal width so the background fills evenly.
	return lipgloss.NewStyle().
		Width(m.width).
		Render(row)
}

// SetActive updates the highlighted provider. If id does not match any
// registered provider the active provider is unchanged.
func (m *ProviderTabBarModel) SetActive(id state.ProviderID) {
	for _, p := range m.providers {
		if p == id {
			m.active = id
			return
		}
	}
}

// SetWidth updates the tab bar width for responsive resizing.
func (m *ProviderTabBarModel) SetWidth(w int) {
	m.width = w
}

// IsVisible reports whether the provider tab bar should be rendered. It is
// false when only one (or zero) providers are registered.
func (m ProviderTabBarModel) IsVisible() bool {
	return m.visible
}

// Height returns the number of terminal rows consumed by the tab bar: 1 when
// visible, 0 when hidden. Used by AppModel.computeLayout to allocate space.
func (m ProviderTabBarModel) Height() int {
	if m.visible {
		return 1
	}
	return 0
}
