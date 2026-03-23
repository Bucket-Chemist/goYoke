// Package banner implements the fixed-height top banner component for the
// GOgent-Fortress TUI. It renders the application name centred inside a
// rounded lipgloss border and responds to window-resize messages so it
// always fills the full terminal width.
package banner

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
)

// BannerModel is the Bubbletea model for the top banner strip.
// It renders "GOgent-Fortress" centred inside a rounded border box.
//
// The zero value is not usable; use NewBannerModel instead.
type BannerModel struct {
	width int
}

// NewBannerModel returns a BannerModel with the given terminal width.
func NewBannerModel(width int) BannerModel {
	return BannerModel{width: width}
}

// Init implements tea.Model. The banner requires no startup commands.
func (m BannerModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model. It handles tea.WindowSizeMsg to keep the
// banner width in sync with the terminal size.
func (m BannerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
	}
	return m, nil
}

// View implements tea.Model. It renders the banner as a 3-row rounded border
// box that spans the full terminal width. The application name is styled
// with config.StyleTitle and centred horizontally inside the box.
//
// The rendered output is always exactly 3 rows tall:
//   - row 1: top border
//   - row 2: title text
//   - row 3: bottom border
func (m BannerModel) View() string {
	title := config.StyleTitle.Render("GOgent-Fortress")

	// The border consumes 2 columns on each side (border + space handled by
	// lipgloss padding). We use lipgloss.Place to centre the title inside the
	// inner width of the box.
	innerWidth := m.width - 2 // 1 border char on each side
	if innerWidth < 1 {
		innerWidth = 1
	}

	centred := lipgloss.Place(
		innerWidth, 1,
		lipgloss.Center, lipgloss.Center,
		title,
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(config.ColorPrimary).
		Width(m.width - 2). // lipgloss Width sets inner width; total = width
		Render(centred)

	return box
}

// SetWidth updates the banner width for responsive resizing. It mirrors the
// state change that Update applies on tea.WindowSizeMsg so callers can resize
// the component directly when composing layouts.
func (m *BannerModel) SetWidth(w int) {
	m.width = w
}
