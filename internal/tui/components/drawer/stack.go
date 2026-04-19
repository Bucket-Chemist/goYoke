package drawer

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DrawerStack manages four DrawerModels (options, plan, teams, figures) as a
// vertical stack rendered at the bottom of the right panel. The zero value is
// not usable; use NewDrawerStack instead.
type DrawerStack struct {
	options DrawerModel
	plan    DrawerModel
	teams   DrawerModel
	figures DrawerModel
	width   int
	height  int
}

// NewDrawerStack creates a DrawerStack with options, plan, teams, and figures
// drawers, all starting in the minimised state.
func NewDrawerStack() DrawerStack {
	return DrawerStack{
		options: NewDrawerModel(DrawerOptions, "Options", "⚙"),
		plan:    NewDrawerModel(DrawerPlan, "Plan", "📋"),
		teams:   NewDrawerModel(DrawerTeams, "Teams", "📊"),
		figures: NewDrawerModel(DrawerFigures, "Figures", "📈"),
	}
}

// MinimizedRowHeight is the rendered height of a minimized drawer (border
// top + label + border bottom).
const MinimizedRowHeight = 3

// SetSize distributes the available width and height among the drawers.
// Each minimized drawer takes 3 rows (bordered label). Expanded drawers
// split the remaining height evenly. All drawers get the full width.
func (s *DrawerStack) SetSize(w, h int) {
	s.width = w
	s.height = h

	drawers := []*DrawerModel{&s.options, &s.plan, &s.teams, &s.figures}

	var expandedCount int
	minimizedRows := 0
	for _, d := range drawers {
		if d.State() == DrawerExpanded {
			expandedCount++
		} else {
			minimizedRows += MinimizedRowHeight
		}
	}

	if expandedCount == 0 {
		for _, d := range drawers {
			d.SetSize(w, MinimizedRowHeight)
		}
		return
	}

	expandedH := h - minimizedRows
	if expandedH < expandedCount {
		expandedH = expandedCount
	}
	perExpanded := expandedH / expandedCount
	remainder := expandedH - perExpanded*expandedCount

	expandedIdx := 0
	for _, d := range drawers {
		if d.State() != DrawerExpanded {
			d.SetSize(w, MinimizedRowHeight)
		} else {
			dh := perExpanded
			if expandedIdx == 0 && remainder > 0 {
				dh += remainder
			}
			d.SetSize(w, dh)
			expandedIdx++
		}
	}
}

// Options returns a pointer to the options DrawerModel.
func (s *DrawerStack) Options() *DrawerModel { return &s.options }

// Plan returns a pointer to the plan DrawerModel.
func (s *DrawerStack) Plan() *DrawerModel { return &s.plan }

// Teams returns a pointer to the teams DrawerModel.
func (s *DrawerStack) Teams() *DrawerModel { return &s.teams }

// Figures returns a pointer to the figures DrawerModel.
func (s *DrawerStack) Figures() *DrawerModel { return &s.figures }

// TeamsIsMinimized returns true when the teams drawer is in the minimized state.
func (s *DrawerStack) TeamsIsMinimized() bool {
	return s.teams.State() == DrawerMinimized
}

// FiguresIsMinimized returns true when the figures drawer is in the minimized state.
func (s *DrawerStack) FiguresIsMinimized() bool {
	return s.figures.State() == DrawerMinimized
}

// ExpandedDrawers returns the DrawerID strings for all currently expanded drawers.
func (s DrawerStack) ExpandedDrawers() []string {
	var ids []string
	if s.options.State() == DrawerExpanded {
		ids = append(ids, string(DrawerOptions))
	}
	if s.plan.State() == DrawerExpanded {
		ids = append(ids, string(DrawerPlan))
	}
	if s.teams.State() == DrawerExpanded {
		ids = append(ids, string(DrawerTeams))
	}
	if s.figures.State() == DrawerExpanded {
		ids = append(ids, string(DrawerFigures))
	}
	return ids
}

// View renders the stack. All drawers are stacked vertically, each with its
// own border — expanded drawers show content, minimized drawers show a
// compact label.
func (s DrawerStack) View() string {
	var parts []string
	for _, d := range []DrawerModel{s.options, s.plan, s.teams, s.figures} {
		parts = append(parts, d.View())
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// HandleKey routes keyboard input to the currently focused drawer.
func (s *DrawerStack) HandleKey(focusedDrawer string, msg tea.KeyMsg) tea.Cmd {
	switch DrawerID(focusedDrawer) {
	case DrawerOptions:
		return s.options.HandleKey(msg)
	case DrawerPlan:
		return s.plan.HandleKey(msg)
	case DrawerTeams:
		return s.teams.HandleKey(msg)
	case DrawerFigures:
		return s.figures.HandleKey(msg)
	}
	return nil
}

// SetOptionsContent sets content in the options drawer (auto-expands it).
func (s *DrawerStack) SetOptionsContent(content string) { s.options.SetContent(content) }

// ClearOptionsContent clears options drawer content (auto-minimises it).
func (s *DrawerStack) ClearOptionsContent() { s.options.ClearContent() }

// OptionsHasContent reports whether the options drawer holds any content.
func (s *DrawerStack) OptionsHasContent() bool { return s.options.HasContent() }

// SetPlanContent sets content in the plan drawer (auto-expands it).
func (s *DrawerStack) SetPlanContent(content string) { s.plan.SetContent(content) }

// ClearPlanContent clears plan drawer content (auto-minimises it).
func (s *DrawerStack) ClearPlanContent() { s.plan.ClearContent() }

// PlanHasContent reports whether the plan drawer holds any content.
func (s *DrawerStack) PlanHasContent() bool { return s.plan.HasContent() }

// SetOptionsFocused marks the options drawer as focused or unfocused.
func (s *DrawerStack) SetOptionsFocused(focused bool) { s.options.SetFocused(focused) }

// SetPlanFocused marks the plan drawer as focused or unfocused.
func (s *DrawerStack) SetPlanFocused(focused bool) { s.plan.SetFocused(focused) }

// SetActiveModal stores a modal request in the options drawer (TDS-006).
func (s *DrawerStack) SetActiveModal(requestID string, message string, options []string) {
	s.options.SetActiveModal(requestID, message, options)
}

// HasActiveModal reports whether the options drawer is displaying a modal.
func (s *DrawerStack) HasActiveModal() bool { return s.options.HasActiveModal() }

// OptionsActiveRequestID returns the current modal request ID from the options drawer.
func (s *DrawerStack) OptionsActiveRequestID() string { return s.options.ActiveRequestID() }

// OptionsSelectedOption returns the currently highlighted option label from the options drawer.
func (s *DrawerStack) OptionsSelectedOption() string { return s.options.SelectedOption() }

// ClearOptionsModal clears the active modal state and content from the options drawer.
func (s *DrawerStack) ClearOptionsModal() {
	s.options.ClearActiveModal()
	s.options.ClearContent()
}

// SetTeamsContent sets content in the teams drawer (auto-expands it).
func (s *DrawerStack) SetTeamsContent(content string) { s.teams.SetContent(content) }

// ClearTeamsContent clears teams drawer content (auto-minimises it).
func (s *DrawerStack) ClearTeamsContent() { s.teams.ClearContent() }

// TeamsHasContent reports whether the teams drawer holds any content.
func (s *DrawerStack) TeamsHasContent() bool { return s.teams.HasContent() }

// RefreshTeamsContent updates the teams drawer content without changing
// expansion state when content already exists.
func (s *DrawerStack) RefreshTeamsContent(content string) { s.teams.RefreshContent(content) }

// SetTeamsFocused marks the teams drawer as focused or unfocused.
func (s *DrawerStack) SetTeamsFocused(focused bool) { s.teams.SetFocused(focused) }

// ---------------------------------------------------------------------------
// Figures drawer accessors
// ---------------------------------------------------------------------------

// SetFiguresContent sets content in the figures drawer (auto-expands it).
func (s *DrawerStack) SetFiguresContent(content string) { s.figures.SetContent(content) }

// ClearFiguresContent clears figures drawer content (auto-minimises it).
func (s *DrawerStack) ClearFiguresContent() {
	s.figures.ClearContent()
}

// FiguresHasContent reports whether the figures drawer holds any content.
func (s *DrawerStack) FiguresHasContent() bool { return s.figures.HasContent() }

// RefreshFiguresContent updates the figures drawer content without changing
// expansion state when content already exists.
func (s *DrawerStack) RefreshFiguresContent(content string) { s.figures.RefreshContent(content) }

// SetFiguresFocused marks the figures drawer as focused or unfocused.
func (s *DrawerStack) SetFiguresFocused(focused bool) { s.figures.SetFocused(focused) }

// ToggleFiguresDrawer expands the figures drawer if minimized, or minimizes it
// if expanded, without clearing the loaded diagram content.
func (s *DrawerStack) ToggleFiguresDrawer() { s.figures.Toggle() }

