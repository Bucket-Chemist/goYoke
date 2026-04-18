package modals

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/goYoke/internal/tui/config"
	"github.com/Bucket-Chemist/goYoke/internal/tui/state"
)

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

// ModelModalClosedMsg is sent when the user dismisses the model modal.
type ModelModalClosedMsg struct{}

// ModelSelectedMsg is sent when the user selects a model from the modal.
type ModelSelectedMsg struct {
	ModelID string
}

// ---------------------------------------------------------------------------
// Layout constants
// ---------------------------------------------------------------------------

const (
	modelBorderFrame = 2
	modelHeaderRows  = 3
	modelFooterRows  = 3
	modelMaxWidth    = 72
	modelMaxHeight   = 35
)

// ---------------------------------------------------------------------------
// Styles
// ---------------------------------------------------------------------------

var (
	modelBorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(config.ColorPrimary)

	modelTitleStyle = config.StyleTitle.Copy()

	modelDividerStyle = config.StyleSubtle.Copy()

	modelFooterStyle = config.StyleSubtle.Copy()

	modelSectionStyle = lipgloss.NewStyle().Bold(true).
		Foreground(config.ColorPrimary)

	modelIDStyle = lipgloss.NewStyle().
		Foreground(config.ColorPrimary)

	modelIDActiveStyle = lipgloss.NewStyle().
		Foreground(config.ColorPrimary).Bold(true)

	modelDisplayStyle = lipgloss.NewStyle()

	modelDisplayActiveStyle = lipgloss.NewStyle().Bold(true)

	modelDescStyle = config.StyleSubtle.Copy()

	modelStrengthStyle = lipgloss.NewStyle().
		Foreground(config.ColorMuted)

	modelMetaStyle = config.StyleSubtle.Copy()

	modelCursorStyle = lipgloss.NewStyle().
		Foreground(config.ColorAccent).Bold(true)

	modelTipStyle = lipgloss.NewStyle().
		Foreground(config.ColorMuted).Italic(true)
)

// ---------------------------------------------------------------------------
// ModelModal
// ---------------------------------------------------------------------------

// ModelModal is an interactive model selector overlay. It groups models by
// tier, shows capabilities, and allows selection via keyboard navigation.
type ModelModal struct {
	models      []state.ModelConfig
	activeModel string
	selectedIdx int
	// selectableIdxs maps cursor positions to indices in the models slice,
	// skipping section headers during navigation.
	selectableIdxs []int
	viewport       viewport.Model
	width          int
	height         int
	active         bool
}

// NewModelModal returns a ModelModal in its initial (inactive) state.
func NewModelModal() ModelModal {
	return ModelModal{
		viewport: viewport.New(0, 0),
	}
}

// ---------------------------------------------------------------------------
// Mutators
// ---------------------------------------------------------------------------

// Show activates the modal with the given model list and current model ID.
func (m *ModelModal) Show(models []state.ModelConfig, activeModel string) {
	m.models = models
	m.activeModel = activeModel
	m.selectedIdx = 0
	m.buildSelectableIndexes()

	// Pre-select the active model.
	for i, idx := range m.selectableIdxs {
		if idx < len(m.models) && m.models[idx].ID == activeModel {
			m.selectedIdx = i
			break
		}
	}

	m.rebuildViewport(m.width, m.height)
	m.active = true
}

// Hide deactivates the modal.
func (m *ModelModal) Hide() { m.active = false }

// IsActive reports whether the modal is currently shown.
func (m ModelModal) IsActive() bool { return m.active }

// SetSize updates the terminal dimensions.
func (m *ModelModal) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.rebuildViewport(w, h)
}

func (m *ModelModal) buildSelectableIndexes() {
	m.selectableIdxs = nil
	for i := range m.models {
		m.selectableIdxs = append(m.selectableIdxs, i)
	}
}

func (m *ModelModal) rebuildViewport(w, h int) {
	innerW := w - modelBorderFrame
	if innerW > modelMaxWidth {
		innerW = modelMaxWidth
	}
	if innerW < 1 {
		innerW = 1
	}
	innerH := h - modelBorderFrame - modelHeaderRows - modelFooterRows
	if innerH > modelMaxHeight {
		innerH = modelMaxHeight
	}
	if innerH < 1 {
		innerH = 1
	}

	m.viewport.Width = innerW
	m.viewport.Height = innerH
	m.viewport.SetContent(m.buildContent(innerW))
}

// ---------------------------------------------------------------------------
// Content builder
// ---------------------------------------------------------------------------

func (m ModelModal) buildContent(width int) string {
	var sb strings.Builder

	tierOrder := []string{"flagship", "balanced", "fast", ""}
	tierLabels := map[string]string{
		"flagship": "FLAGSHIP",
		"balanced": "BALANCED",
		"fast":     "FAST",
		"":         "OTHER",
	}

	grouped := make(map[string][]int)
	for i, mc := range m.models {
		tier := mc.Tier
		grouped[tier] = append(grouped[tier], i)
	}

	first := true
	for _, tier := range tierOrder {
		idxs, ok := grouped[tier]
		if !ok || len(idxs) == 0 {
			continue
		}
		if !first {
			sb.WriteString("\n")
		}
		first = false

		label := tierLabels[tier]
		sb.WriteString(modelSectionStyle.Render(label))
		sb.WriteString("\n")
		divW := width
		if divW > 68 {
			divW = 68
		}
		sb.WriteString(modelDividerStyle.Render(strings.Repeat("\u2500", divW)))
		sb.WriteString("\n")

		for _, idx := range idxs {
			mc := m.models[idx]
			isActive := mc.ID == m.activeModel
			isCursor := m.cursorModelIdx() == idx

			sb.WriteString(m.renderModelEntry(mc, isActive, isCursor, width))
		}
	}

	sb.WriteString("\n")
	sb.WriteString(modelDividerStyle.Render(strings.Repeat("\u2500", min(width, 68))))
	sb.WriteString("\n")
	sb.WriteString(modelTipStyle.Render("Tip: /model <any-id> accepts any Claude CLI model ID"))
	sb.WriteString("\n")

	return sb.String()
}

func (m ModelModal) cursorModelIdx() int {
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.selectableIdxs) {
		return -1
	}
	return m.selectableIdxs[m.selectedIdx]
}

func (m ModelModal) renderModelEntry(mc state.ModelConfig, isActive, isCursor bool, width int) string {
	var sb strings.Builder

	// Cursor / indent
	cursor := "    "
	if isCursor {
		cursor = modelCursorStyle.Render("  \u25b8 ")
	}

	// Model ID + display name
	idStr := mc.ID
	dispStr := mc.DisplayName
	if isActive {
		idStr = modelIDActiveStyle.Render(idStr)
		dispStr = modelDisplayActiveStyle.Render(dispStr)
	} else {
		idStr = modelIDStyle.Render(idStr)
		dispStr = modelDisplayStyle.Render(dispStr)
	}

	// Right-aligned meta: cost + speed
	meta := ""
	if mc.CostTier != "" || mc.Speed != "" {
		parts := []string{}
		if mc.CostTier != "" {
			parts = append(parts, mc.CostTier)
		}
		if mc.Speed != "" {
			parts = append(parts, mc.Speed)
		}
		meta = modelMetaStyle.Render(strings.Join(parts, " "))
	}

	// Line 1: cursor + ID + display name + meta
	line1Left := fmt.Sprintf("%s%-14s%s", cursor, mc.ID, "  "+dispStr)
	_ = idStr // styled version used in line1Left via mc.ID; re-style below
	// Re-render with styled components
	paddedID := fmt.Sprintf("%-14s", mc.ID)
	if isActive {
		paddedID = modelIDActiveStyle.Render(paddedID)
	} else {
		paddedID = modelIDStyle.Render(paddedID)
	}
	line1Left = cursor + paddedID + "  " + dispStr

	if meta != "" {
		gap := width - lipgloss.Width(line1Left) - lipgloss.Width(meta)
		if gap < 2 {
			gap = 2
		}
		sb.WriteString(line1Left + strings.Repeat(" ", gap) + meta)
	} else {
		sb.WriteString(line1Left)
	}
	sb.WriteString("\n")

	// Line 2: description
	if mc.Description != "" {
		sb.WriteString("    " + modelDescStyle.Render(mc.Description))
		sb.WriteString("\n")
	}

	// Line 3: strengths
	if len(mc.Strengths) > 0 {
		sb.WriteString("    " + modelStrengthStyle.Render(strings.Join(mc.Strengths, " \u00b7 ")))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	return sb.String()
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (m ModelModal) Update(msg tea.Msg) (ModelModal, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m ModelModal) handleKey(msg tea.KeyMsg) (ModelModal, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.active = false
		return m, func() tea.Msg { return ModelModalClosedMsg{} }

	case "up", "k":
		if m.selectedIdx > 0 {
			m.selectedIdx--
			m.viewport.SetContent(m.buildContent(m.viewport.Width))
			m.scrollToSelected()
		}
		return m, nil

	case "down", "j":
		if m.selectedIdx < len(m.selectableIdxs)-1 {
			m.selectedIdx++
			m.viewport.SetContent(m.buildContent(m.viewport.Width))
			m.scrollToSelected()
		}
		return m, nil

	case "enter":
		idx := m.cursorModelIdx()
		if idx >= 0 && idx < len(m.models) {
			modelID := m.models[idx].ID
			m.active = false
			return m, func() tea.Msg { return ModelSelectedMsg{ModelID: modelID} }
		}
		return m, nil
	}

	// Forward to viewport for pgup/pgdn scrolling.
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *ModelModal) scrollToSelected() {
	if len(m.selectableIdxs) == 0 {
		return
	}
	// Each model entry is ~4 lines, plus section headers.
	// Approximate the target line.
	linesPerEntry := 4
	targetLine := m.selectedIdx * linesPerEntry
	if targetLine < m.viewport.YOffset {
		m.viewport.YOffset = targetLine
	} else if targetLine >= m.viewport.YOffset+m.viewport.Height {
		m.viewport.YOffset = targetLine - m.viewport.Height + 1
	}
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (m ModelModal) View() string {
	if !m.active {
		return ""
	}

	innerW := m.viewport.Width
	if innerW < 1 {
		innerW = m.width - modelBorderFrame
		if innerW < 1 {
			innerW = 40
		}
	}

	title := modelTitleStyle.Render("Model Selector")
	divider := modelDividerStyle.Render(strings.Repeat("\u2500", innerW))

	scrollPct := 0
	if m.viewport.TotalLineCount() > 0 {
		scrollPct = int(m.viewport.ScrollPercent() * 100)
	}
	vpView := m.viewport.View()

	footer := modelFooterStyle.Render(
		fmt.Sprintf("\u2191/\u2193: navigate  Enter: select  Esc: close  %3d%%", scrollPct),
	)

	inner := strings.Join([]string{title, divider, vpView, footer}, "\n")

	return modelBorderStyle.
		Width(innerW).
		Render(inner)
}
