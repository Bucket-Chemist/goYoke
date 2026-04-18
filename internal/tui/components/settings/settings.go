// Package settings implements the settings panel for the goYoke TUI.
// It displays static configuration values in a compact key-value layout.
//
// The component is display-only: Update is a no-op and state is set
// exclusively via SetConfig. No I/O is performed in View.
package settings

import (
	"fmt"
	"strings"

	"github.com/Bucket-Chemist/goYoke/internal/tui/config"
	"github.com/Bucket-Chemist/goYoke/internal/tui/model"
)

// ---------------------------------------------------------------------------
// SettingsModel
// ---------------------------------------------------------------------------

// SettingsModel is the display-only model for the settings panel. It renders
// the active session configuration: model, provider, permission mode, session
// directory, and MCP server count.
//
// The zero value is usable and renders an empty configuration.
type SettingsModel struct {
	width  int
	height int

	// Static config — set once via SetConfig.
	model          string
	provider       string
	permissionMode string
	sessionDir     string
	mcpServers     []string
}

// NewSettingsModel returns a SettingsModel with sensible zero defaults.
func NewSettingsModel() SettingsModel {
	return SettingsModel{}
}

// SetSize updates the rendering dimensions. Call this on every
// tea.WindowSizeMsg so the component is aware of available space.
func (m *SettingsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetTier satisfies the settingsWidget interface.  Tier-specific rendering
// adaptations are reserved for a future ticket; this is a no-op placeholder.
func (m *SettingsModel) SetTier(_ model.LayoutTier) {}

// SetConfig sets all configuration fields in a single call. mcpServers may be
// nil or empty when no MCP servers are configured.
func (m *SettingsModel) SetConfig(model, provider, permMode, sessionDir string, mcpServers []string) {
	m.model = model
	m.provider = provider
	m.permissionMode = permMode
	m.sessionDir = sessionDir
	// Defensive copy so callers cannot mutate the internal slice.
	if len(mcpServers) > 0 {
		cp := make([]string, len(mcpServers))
		copy(cp, mcpServers)
		m.mcpServers = cp
	} else {
		m.mcpServers = nil
	}
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

// View renders the settings as a headed key-value list. It is a pure
// function of the model state — no I/O is performed here.
func (m SettingsModel) View() string {
	var sb strings.Builder

	// Header.
	sb.WriteString(config.StyleTitle.Render("Settings"))
	sb.WriteByte('\n')
	sb.WriteString(config.StyleSubtle.Render(divider(m.width)))
	sb.WriteByte('\n')

	// Row helper: fixed-width label on the left, value on the right.
	row := func(label, value string) {
		labelStr := config.StyleHighlight.Render(fmt.Sprintf("%-14s", label))
		valueStr := config.StyleSubtle.Render(value)
		sb.WriteString(labelStr)
		sb.WriteString(valueStr)
		sb.WriteByte('\n')
	}

	row("Model:", orDash(m.model))
	row("Provider:", orDash(m.provider))
	row("Permission:", orDash(m.permissionMode))
	row("Session Dir:", truncatePath(m.sessionDir, m.pathWidth()))
	row("MCP Servers:", m.formatMCPServers())

	return strings.TrimRight(sb.String(), "\n")
}

// formatMCPServers returns a human-readable summary of the configured MCP
// servers.
func (m SettingsModel) formatMCPServers() string {
	n := len(m.mcpServers)
	switch n {
	case 0:
		return "none"
	case 1:
		return fmt.Sprintf("%s (1 server)", m.mcpServers[0])
	default:
		return fmt.Sprintf("%d servers", n)
	}
}

// pathWidth returns the usable width for the session dir path.
func (m SettingsModel) pathWidth() int {
	// Reserve space for the label and spacing overhead (~16 chars).
	w := m.width - 16
	if w < 20 {
		return 20
	}
	return w
}

// orDash returns s when non-empty, otherwise an em-dash placeholder.
func orDash(s string) string {
	if s == "" {
		return "\u2014" // em dash
	}
	return s
}

// truncatePath shortens a path to at most maxLen runes from the right,
// prepending "…" if it was cut. This preserves the end of the path (the most
// informative part) rather than the beginning.
func truncatePath(path string, maxLen int) string {
	if path == "" {
		return "\u2014" // em dash
	}
	runes := []rune(path)
	if len(runes) <= maxLen {
		return path
	}
	if maxLen <= 1 {
		return "\u2026" // horizontal ellipsis
	}
	return "\u2026" + string(runes[len(runes)-(maxLen-1):])
}

// divider returns a horizontal rule string fitting the given width.
func divider(width int) string {
	if width <= 0 {
		width = 20
	}
	if width > 40 {
		width = 40
	}
	return strings.Repeat("\u2500", width)
}
