// Package slashcmd implements a slash command dropdown autocomplete component
// for the goYoke TUI. It renders a filterable, scrollable list of
// available slash commands and emits a SlashCmdSelectedMsg when the user
// confirms a selection.
//
// The component follows Bubbletea's Elm Architecture: no I/O is performed in
// View. State mutations are confined to Update and the explicit mutator methods
// (Show, Hide, Filter, SetWidth).
package slashcmd

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/goYoke/internal/tui/config"
)

// ---------------------------------------------------------------------------
// Data types
// ---------------------------------------------------------------------------

// SlashCommand describes a single slash command available in the TUI.
type SlashCommand struct {
	// Name is the command identifier without the leading "/", e.g. "explore".
	Name string
	// Description is a short human-readable summary of what the command does.
	Description string
}

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

// SlashCmdSelectedMsg is emitted when the user selects a command from the
// dropdown. Command always includes the leading "/" prefix.
type SlashCmdSelectedMsg struct {
	// Command is the full slash command including the "/" prefix,
	// e.g. "/explore".
	Command string
}

// ---------------------------------------------------------------------------
// Styles (package-level, constructed once)
// ---------------------------------------------------------------------------

var (
	// dropdownBorderStyle renders the outer box around the dropdown.
	dropdownBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(config.ColorPrimary).
				Padding(0, 1)

	// itemStyle renders a non-selected dropdown item.
	itemStyle = lipgloss.NewStyle().
			Foreground(config.ColorMuted)

	// selectedItemStyle renders the currently highlighted dropdown item.
	selectedItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(config.ColorAccent)

	// cmdNameStyle renders the "/name" portion of each item.
	cmdNameStyle = lipgloss.NewStyle().
			Foreground(config.ColorPrimary)

	// selectedCmdNameStyle renders "/name" when the item is selected.
	selectedCmdNameStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(config.ColorAccent)

	// descStyle renders the description portion of each item.
	descStyle = lipgloss.NewStyle().
			Foreground(config.ColorMuted)

	// selectedDescStyle renders the description when the item is selected.
	selectedDescStyle = lipgloss.NewStyle().
				Foreground(config.ColorPrimary)
)

// ---------------------------------------------------------------------------
// Layout constants
// ---------------------------------------------------------------------------

const (
	// defaultMaxVisible is the maximum number of dropdown items visible at once
	// before scrolling kicks in.
	defaultMaxVisible = 8
)

// ---------------------------------------------------------------------------
// DefaultCommands
// ---------------------------------------------------------------------------

// DefaultCommands returns the canonical list of slash commands available in
// goYoke. This list mirrors the Slash Commands table in CLAUDE.md.
func DefaultCommands() []SlashCommand {
	return []SlashCommand{
		{"explore", "Structured codebase exploration"},
		{"braintrust", "Multi-perspective deep analysis"},
		{"review", "Multi-domain code review"},
		{"review-plan", "Critical 7-layer review of plans"},
		{"sandbox", "Manage sandbox permissions and write protected files"},
		{"cwd", "Change working directory for CC sessions"},
		{"ticket", "Ticket-driven implementation"},
		{"implement", "Plan + implement a feature"},
		{"init-auto", "Initialize project config"},
		{"benchmark", "Run gold standard prompts"},
		{"benchmark-meta", "Analyze benchmark trends"},
		{"benchmark-agent", "Evaluate agents via Harbor"},
		{"memory-improvement", "Audit system memory"},
		{"explore-add", "Add custom skill"},
		{"dummies-guide", "Explain config system"},
		{"team-status", "Show team progress"},
		{"team-result", "Display team output"},
		{"team-cancel", "Stop running team"},
		{"plan-tickets", "Comprehensive planning workflow"},
		{"teams", "List all teams"},
		{"model", "Switch model (e.g. /model haiku)"},
	}
}

// HelpText returns a formatted string listing all available slash commands,
// suitable for display as a system message in the Claude panel. Extra commands
// (e.g. dynamically loaded skill commands) are appended after the defaults.
func HelpText(extra ...SlashCommand) string {
	var sb strings.Builder
	sb.WriteString("Available slash commands:\n")
	all := append(DefaultCommands(), extra...)
	for _, cmd := range all {
		sb.WriteString("  /" + cmd.Name + " — " + cmd.Description + "\n")
	}
	sb.WriteString("\nLocal commands: /clear (clear conversation), /model [name] (switch model), /help (this message)")
	return sb.String()
}

// ---------------------------------------------------------------------------
// Skill command loader
// ---------------------------------------------------------------------------

// localCommandNames is the set of built-in TUI command names that must never
// be overridden by skill discovery.
var localCommandNames = map[string]struct{}{
	"clear":  {},
	"exit":   {},
	"quit":   {},
	"help":   {},
	"cwd":    {},
	"model":  {},
	"effort": {},
}

// LoadSkillCommands scans configDir/skills/ for subdirectories and returns a
// SlashCommand for each discovered skill. If the subdirectory contains a
// SKILL.md file with YAML frontmatter that includes a description: field, that
// value is used; otherwise a default description "Skill: <name>" is used.
//
// Commands whose names collide with localCommandNames are silently filtered.
//
// The function is fault-tolerant: unreadable directories or files produce
// slog warnings but do not stop discovery of other skills.
func LoadSkillCommands(configDir string) []SlashCommand {
	skillsDir := filepath.Join(configDir, "skills")

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		slog.Warn("slashcmd: cannot read skills directory", "dir", skillsDir, "err", err)
		return nil
	}

	var cmds []SlashCommand
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()

		if _, reserved := localCommandNames[name]; reserved {
			continue
		}

		desc := extractSkillDescription(filepath.Join(skillsDir, name, "SKILL.md"), name)
		cmds = append(cmds, SlashCommand{Name: name, Description: desc})
	}

	return cmds
}

// extractSkillDescription reads skillFile and tries to extract a description:
// field from YAML frontmatter. Returns "Skill: <name>" when the file is
// missing, unreadable, lacks frontmatter, or has no description field.
func extractSkillDescription(skillFile, name string) string {
	defaultDesc := "Skill: " + name

	data, err := os.ReadFile(skillFile)
	if err != nil {
		return defaultDesc
	}

	desc := parseFrontmatterDescription(string(data))
	if desc == "" {
		return defaultDesc
	}
	return desc
}

// parseFrontmatterDescription extracts the value of the "description:" key
// from YAML frontmatter at the top of content. Frontmatter must be delimited
// by "---" lines at the start of the file. Returns an empty string if not
// found.
func parseFrontmatterDescription(content string) string {
	if !strings.HasPrefix(content, "---") {
		return ""
	}

	// Advance past the opening "---" and its trailing newline.
	rest := content[3:]
	if strings.HasPrefix(rest, "\r\n") {
		rest = rest[2:]
	} else if strings.HasPrefix(rest, "\n") {
		rest = rest[1:]
	}

	// Locate the closing "---".
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return ""
	}

	frontmatter := rest[:end]

	for _, line := range strings.Split(frontmatter, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "description:") {
			continue
		}
		val := strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		// Strip optional surrounding quotes.
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') ||
				(val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		return val
	}

	return ""
}

// ---------------------------------------------------------------------------
// SlashCmdModel
// ---------------------------------------------------------------------------

// SlashCmdModel is the Bubbletea sub-model for the slash command dropdown.
// When not visible, View returns an empty string.
//
// The zero value is not usable; use NewSlashCmdModel instead.
type SlashCmdModel struct {
	// commands is the full unfiltered list of slash commands.
	commands []SlashCommand
	// filtered is the subset of commands that match the current query.
	filtered []SlashCommand
	// selected is the index into filtered for the highlighted item.
	selected int
	// query is the current filter text (without the leading "/").
	query string
	// visible controls whether the dropdown is shown.
	visible bool
	// width is the terminal width used to constrain rendering.
	width int
	// maxVisible is the maximum number of items shown before scrolling.
	maxVisible int
	// scrollOffset is the index of the first visible item in filtered.
	scrollOffset int
}

// NewSlashCmdModel returns a SlashCmdModel initialised with DefaultCommands
// and a maxVisible of 8. The dropdown starts hidden. Any extra commands
// (e.g. dynamically loaded skill commands) are appended after the defaults.
func NewSlashCmdModel(extra ...SlashCommand) SlashCmdModel {
	cmds := append(DefaultCommands(), extra...)
	return SlashCmdModel{
		commands:   cmds,
		filtered:   append([]SlashCommand(nil), cmds...),
		maxVisible: defaultMaxVisible,
	}
}

// ---------------------------------------------------------------------------
// Public mutators
// ---------------------------------------------------------------------------

// Show makes the dropdown visible and applies the given query as an initial
// filter. Resets selection to the first item. If query produces no matches
// the dropdown is still shown with an empty filtered list; callers can check
// IsVisible after Filter to detect this edge case if needed.
func (m *SlashCmdModel) Show(query string) {
	m.visible = true
	m.selected = 0
	m.scrollOffset = 0
	m.applyFilter(query)
	// If filter produced zero results, hide immediately.
	if len(m.filtered) == 0 {
		m.visible = false
	}
}

// Hide closes the dropdown without emitting any message.
func (m *SlashCmdModel) Hide() {
	m.visible = false
}

// IsVisible returns true when the dropdown is currently shown.
func (m SlashCmdModel) IsVisible() bool {
	return m.visible
}

// Filter updates the filter query and re-filters the command list. If the
// new query produces no matches the dropdown is hidden automatically.
func (m *SlashCmdModel) Filter(query string) {
	m.applyFilter(query)
	if len(m.filtered) == 0 {
		m.visible = false
		return
	}
	// Clamp selection after re-filtering.
	if m.selected >= len(m.filtered) {
		m.selected = len(m.filtered) - 1
	}
	m.clampScroll()
}

// Selected returns the SlashCommand currently highlighted in the dropdown.
// Returns the zero value when filtered is empty.
func (m SlashCmdModel) Selected() SlashCommand {
	if len(m.filtered) == 0 || m.selected < 0 || m.selected >= len(m.filtered) {
		return SlashCommand{}
	}
	return m.filtered[m.selected]
}

// SetWidth updates the terminal width used during rendering.
func (m *SlashCmdModel) SetWidth(w int) {
	m.width = w
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

// Update handles keyboard events for the dropdown. Only "up", "k", "down",
// "j", "enter", and "escape" are consumed when the dropdown is visible; all
// other messages are ignored and returned unchanged so the parent can handle
// them.
func (m SlashCmdModel) Update(msg tea.Msg) (SlashCmdModel, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "up", "k":
		if m.selected > 0 {
			m.selected--
			m.clampScroll()
		}

	case "down", "j":
		if m.selected < len(m.filtered)-1 {
			m.selected++
			m.clampScroll()
		}

	case "enter":
		if len(m.filtered) == 0 {
			return m, nil
		}
		chosen := m.filtered[m.selected]
		m.visible = false
		return m, func() tea.Msg {
			return SlashCmdSelectedMsg{Command: "/" + chosen.Name}
		}

	case "escape", "esc":
		m.visible = false
	}

	return m, nil
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

// View renders the dropdown box. Returns an empty string when the dropdown is
// not visible. No I/O is performed here.
func (m SlashCmdModel) View() string {
	if !m.visible || len(m.filtered) == 0 {
		return ""
	}

	// Determine the visible window into m.filtered.
	start := m.scrollOffset
	end := start + m.maxVisible
	if end > len(m.filtered) {
		end = len(m.filtered)
	}
	window := m.filtered[start:end]

	var sb strings.Builder

	for i, cmd := range window {
		absIdx := start + i
		isSelected := absIdx == m.selected

		var line string
		if isSelected {
			name := selectedCmdNameStyle.Render("/" + cmd.Name)
			desc := selectedDescStyle.Render("  " + cmd.Description)
			line = selectedItemStyle.Render(name + desc)
		} else {
			name := cmdNameStyle.Render("/" + cmd.Name)
			desc := descStyle.Render("  " + cmd.Description)
			line = itemStyle.Render(name + desc)
		}

		sb.WriteString(line)
		if i < len(window)-1 {
			sb.WriteByte('\n')
		}
	}

	// Show a scroll indicator when there are items above or below the window.
	if len(m.filtered) > m.maxVisible {
		sb.WriteByte('\n')
		indicator := descStyle.Render(m.scrollIndicator(start, end))
		sb.WriteString(indicator)
	}

	// Wrap the content in a rounded border.
	content := sb.String()

	style := dropdownBorderStyle
	if m.width > 4 {
		// Inner content width = terminal width minus border (2) and padding (2).
		innerW := m.width - 4
		if innerW > 0 {
			style = style.Width(innerW)
		}
	}

	return style.Render(content)
}

// ---------------------------------------------------------------------------
// Private helpers
// ---------------------------------------------------------------------------

// applyFilter re-filters m.commands using a case-insensitive prefix match on
// Name. An empty query shows all commands.
func (m *SlashCmdModel) applyFilter(query string) {
	m.query = query
	q := strings.ToLower(strings.TrimPrefix(query, "/"))

	if q == "" {
		m.filtered = append([]SlashCommand(nil), m.commands...)
		return
	}

	filtered := m.filtered[:0]
	for _, cmd := range m.commands {
		if strings.HasPrefix(strings.ToLower(cmd.Name), q) {
			filtered = append(filtered, cmd)
		}
	}
	m.filtered = filtered
}

// clampScroll adjusts scrollOffset so that the selected item is always
// within the visible window.
func (m *SlashCmdModel) clampScroll() {
	if m.selected < m.scrollOffset {
		m.scrollOffset = m.selected
	}
	if m.selected >= m.scrollOffset+m.maxVisible {
		m.scrollOffset = m.selected - m.maxVisible + 1
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

// scrollIndicator builds a compact "↑ N above · M below ↓" hint string.
func (m SlashCmdModel) scrollIndicator(start, end int) string {
	above := start
	below := len(m.filtered) - end
	var parts []string
	if above > 0 {
		parts = append(parts, "↑")
	}
	if below > 0 {
		parts = append(parts, "↓")
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " ")
}
