package claude

import (
	"strings"

	"github.com/acarl005/stripansi"
	"github.com/charmbracelet/lipgloss"
)

// sanitizePrompt removes ANSI escape sequences from untrusted input
// CRITICAL: Must be called on all MCP server message content
func sanitizePrompt(s string) string {
	return stripansi.Strip(s)
}

var (
	modalStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(50)

	modalTitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("170"))

	modalHelpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	destructiveStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)
)

// RenderModal renders the current modal as a string
func (m *ModalState) RenderModal() string {
	if !m.Active {
		return ""
	}

	var content strings.Builder

	// Title/message - CRITICAL: Sanitize untrusted MCP input
	message := sanitizePrompt(m.Prompt.Message)
	if strings.HasPrefix(message, "⚠️") {
		content.WriteString(destructiveStyle.Render(message))
	} else {
		content.WriteString(modalTitleStyle.Render(message))
	}
	content.WriteString("\n\n")

	// Content based on type
	switch m.Type {
	case ConfirmModal:
		content.WriteString("[Y]es  [N]o  [Esc] Cancel")

	case TextInputModal:
		content.WriteString(m.TextInput.View())
		content.WriteString("\n\n")
		content.WriteString(modalHelpStyle.Render("[Enter] Submit  [Esc] Cancel"))

	case SelectionModal:
		content.WriteString(m.SelectList.View())
		content.WriteString("\n")
		content.WriteString(modalHelpStyle.Render("[Enter] Select  [↑/↓] Navigate  [Esc] Cancel"))
	}

	return modalStyle.Render(content.String())
}

// OverlayModal composites the modal over a background
func OverlayModal(background, modal string, width, height int) string {
	if modal == "" {
		return background
	}

	bgLines := strings.Split(background, "\n")
	modalLines := strings.Split(modal, "\n")

	modalWidth := lipgloss.Width(modal)
	modalHeight := len(modalLines)

	// Center the modal
	startX := max((width-modalWidth)/2, 0)
	startY := max((height-modalHeight)/2, 0)

	// Ensure background has enough lines
	for len(bgLines) < height {
		bgLines = append(bgLines, strings.Repeat(" ", width))
	}

	// Composite modal onto background
	for i, modalLine := range modalLines {
		bgY := startY + i
		if bgY < len(bgLines) {
			bgLines[bgY] = overlayLine(bgLines[bgY], modalLine, startX, width)
		}
	}

	return strings.Join(bgLines, "\n")
}

func overlayLine(background, overlay string, startX, maxWidth int) string {
	// Pad background to maxWidth
	for len(background) < maxWidth {
		background += " "
	}

	bgRunes := []rune(background)
	overlayRunes := []rune(overlay)

	for i, r := range overlayRunes {
		pos := startX + i
		if pos < len(bgRunes) {
			bgRunes[pos] = r
		}
	}

	return string(bgRunes)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
