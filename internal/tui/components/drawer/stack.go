package drawer

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DrawerStack manages two DrawerModels (options and plan) as a vertical stack.
// The zero value is not usable; use NewDrawerStack instead.
type DrawerStack struct {
	options DrawerModel
	plan    DrawerModel
	width   int
	height  int
}

// NewDrawerStack creates a DrawerStack with an options drawer and a plan drawer,
// both starting in the minimised state.
func NewDrawerStack() DrawerStack {
	return DrawerStack{
		options: NewDrawerModel(DrawerOptions, "Options", "⚙"),
		plan:    NewDrawerModel(DrawerPlan, "Plan", "📋"),
	}
}

// SetSize distributes the available width and height between the two drawers
// based on their current expansion states:
//   - Both expanded: 50/50 height split
//   - One expanded: that drawer gets height-1, the other gets 1 row (tab)
//   - Neither expanded: each gets 1 row
func (s *DrawerStack) SetSize(w, h int) {
	s.width = w
	s.height = h

	optExpanded := s.options.State() == DrawerExpanded
	planExpanded := s.plan.State() == DrawerExpanded

	switch {
	case optExpanded && planExpanded:
		half := h / 2
		s.options.SetSize(w, half)
		s.plan.SetSize(w, h-half)
	case optExpanded:
		s.options.SetSize(w, h-1)
		s.plan.SetSize(w, 1)
	case planExpanded:
		s.options.SetSize(w, 1)
		s.plan.SetSize(w, h-1)
	default:
		s.options.SetSize(w, 1)
		s.plan.SetSize(w, 1)
	}
}

// Options returns a pointer to the options DrawerModel.
func (s *DrawerStack) Options() *DrawerModel { return &s.options }

// Plan returns a pointer to the plan DrawerModel.
func (s *DrawerStack) Plan() *DrawerModel { return &s.plan }

// ExpandedDrawers returns the DrawerID strings for all currently expanded drawers.
func (s DrawerStack) ExpandedDrawers() []string {
	var ids []string
	if s.options.State() == DrawerExpanded {
		ids = append(ids, string(DrawerOptions))
	}
	if s.plan.State() == DrawerExpanded {
		ids = append(ids, string(DrawerPlan))
	}
	return ids
}

// View renders the stack. Expanded drawers are stacked vertically; minimised
// drawers appear as horizontal tabs at the bottom.
func (s DrawerStack) View() string {
	var parts []string

	if s.options.State() == DrawerExpanded {
		parts = append(parts, s.options.ViewExpanded())
	}
	if s.plan.State() == DrawerExpanded {
		parts = append(parts, s.plan.ViewExpanded())
	}

	var tabs []string
	if s.options.State() == DrawerMinimized {
		tabs = append(tabs, s.options.ViewMinimized())
	}
	if s.plan.State() == DrawerMinimized {
		tabs = append(tabs, s.plan.ViewMinimized())
	}

	if len(tabs) > 0 {
		parts = append(parts, lipgloss.JoinHorizontal(lipgloss.Top, tabs...))
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
