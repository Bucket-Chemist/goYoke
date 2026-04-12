// Package settingstree implements an interactive settings panel with
// tree-style navigation for the GOgent-Fortress TUI.
//
// The component supports three node types:
//   - SettingToggle:  boolean on/off that flips on Enter/Space
//   - SettingSelect:  a fixed list of options that cycles on Enter/Space
//   - SettingDisplay: read-only value updated via SetValue
//
// SettingChangedMsg is emitted as a tea.Cmd whenever a Toggle or Select
// setting is mutated, allowing the parent model to react without a direct
// dependency on this package.
//
// The package has no dependency on the model package; it imports only config
// to keep the import graph acyclic.
package settingstree

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
)

// ---------------------------------------------------------------------------
// Public types
// ---------------------------------------------------------------------------

// SettingType describes how a SettingNode is rendered and interacted with.
type SettingType int

const (
	// SettingToggle is a boolean on/off setting. Enter/Space flips the value
	// between "on" and "off" and emits SettingChangedMsg.
	SettingToggle SettingType = iota

	// SettingSelect presents a fixed list of options. Enter/Space advances
	// to the next option (wrapping) and emits SettingChangedMsg.
	SettingSelect

	// SettingDisplay is read-only. The value can be updated externally via
	// SetValue; Enter/Space are no-ops and no message is emitted.
	SettingDisplay
)

// SettingNode represents a single configurable item within a SettingsSection.
type SettingNode struct {
	// Key is the unique identifier used in SettingChangedMsg and SetValue.
	Key string

	// Label is the human-readable name displayed in the panel.
	Label string

	// Type determines how the node behaves on activation.
	Type SettingType

	// Value is the current display value ("on"/"off" for Toggle,
	// one of Options for Select, any string for Display).
	Value string

	// Options is the ordered list of valid values for SettingSelect nodes.
	// It is unused for Toggle and Display nodes.
	Options []string

	// Description is shown at the bottom of the panel when this node is
	// selected. May be empty.
	Description string
}

// SettingsSection groups related SettingNodes under a collapsible header.
type SettingsSection struct {
	// Title is the header text rendered for the section.
	Title string

	// Nodes is the ordered list of settings in this section.
	Nodes []SettingNode

	// Expanded controls whether the nodes are visible.
	Expanded bool
}

// SettingChangedMsg is emitted by SettingsTreeModel when the user activates a
// Toggle or Select node. The parent model should type-switch on this message.
type SettingChangedMsg struct {
	// Key matches SettingNode.Key.
	Key string
	// Value is the new value after the change.
	Value string
}

// ---------------------------------------------------------------------------
// SettingsTreeModel
// ---------------------------------------------------------------------------

// SettingsTreeModel is the Bubbletea sub-model for the interactive settings
// panel. It maintains a list of collapsible sections, each containing
// SettingNodes, and handles keyboard navigation and activation.
//
// The zero value is not usable; use NewSettingsTreeModel instead.
type SettingsTreeModel struct {
	sections        []SettingsSection
	selectedSection int
	// selectedItem is the index of the selected node within the current
	// section. -1 means the section header itself is selected.
	selectedItem int
	width        int
	height       int
	focused      bool
}

// NewSettingsTreeModel returns a SettingsTreeModel pre-populated with the
// three canonical sections (Display, Session, Status) and sensible defaults.
// All sections start expanded.
func NewSettingsTreeModel() SettingsTreeModel {
	return SettingsTreeModel{
		sections: defaultSections(),
		// Start with the header of the first section selected.
		selectedSection: 0,
		selectedItem:    -1,
	}
}

// defaultSections returns the three canonical settings sections.
func defaultSections() []SettingsSection {
	return []SettingsSection{
		{
			Title:    "Display",
			Expanded: true,
			Nodes: []SettingNode{
				{
					Key:         "theme",
					Label:       "Theme",
					Type:        SettingSelect,
					Value:       "Dark",
					Options:     []string{"Dark", "Light", "High Contrast"},
					Description: "Select the color palette for the TUI.",
				},
				{
					Key:         "ascii_icons",
					Label:       "ASCII Icons",
					Type:        SettingToggle,
					Value:       "off",
					Description: "Use ASCII-only icons instead of Unicode symbols.",
				},
				{
					Key:         "vim_keys",
					Label:       "Vim Keys",
					Type:        SettingToggle,
					Value:       "off",
					Description: "Enable j/k navigation in all panels.",
				},
				{
					Key:         "reduce_motion",
					Label:       "Reduce Motion",
					Type:        SettingToggle,
					Value:       "off",
					Description: "Disable animations for accessibility (WCAG 2.3.1).",
				},
				{
					Key:         "timestamps",
					Label:       "Timestamps",
					Type:        SettingToggle,
					Value:       "off",
					Description: "Show relative timestamps at turn boundaries.",
				},
				{
					Key:         "cost_flash",
					Label:       "Cost Flash",
					Type:        SettingToggle,
					Value:       "off",
					Description: "Flash cost badge on increase (disabled by Reduce Motion).",
				},
			},
		},
		{
			Title:    "Session",
			Expanded: true,
			Nodes: []SettingNode{
				{
					Key:         "model",
					Label:       "Model",
					Type:        SettingDisplay,
					Value:       "",
					Description: "Active Claude model for this session.",
				},
				{
					Key:         "provider",
					Label:       "Provider",
					Type:        SettingDisplay,
					Value:       "",
					Description: "Active provider (Anthropic, AWS Bedrock, etc.).",
				},
				{
					Key:         "permission_mode",
					Label:       "Permission Mode",
					Type:        SettingDisplay,
					Value:       "",
					Description: "Tool permission mode set at session start.",
				},
				{
					Key:         "session_dir",
					Label:       "Session Dir",
					Type:        SettingDisplay,
					Value:       "",
					Description: "Directory where session state and logs are stored.",
				},
			},
		},
		{
			Title:    "Status",
			Expanded: true,
			Nodes: []SettingNode{
				{
					Key:         "mcp_servers",
					Label:       "MCP Servers",
					Type:        SettingDisplay,
					Value:       "",
					Description: "Number and names of connected MCP servers.",
				},
				{
					Key:         "agent_count",
					Label:       "Agent Count",
					Type:        SettingDisplay,
					Value:       "",
					Description: "Total number of agents registered in this session.",
				},
				{
					Key:         "git_branch",
					Label:       "Git Branch",
					Type:        SettingDisplay,
					Value:       "",
					Description: "Current git branch of the project directory.",
				},
			},
		},
	}
}

// ---------------------------------------------------------------------------
// tea.Model interface
// ---------------------------------------------------------------------------

// Init implements tea.Model. No startup commands are needed.
func (m SettingsTreeModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model. When focused it handles Up/Down navigation and
// Enter/Space activation. Unknown messages and unfocused state are ignored.
func (m SettingsTreeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "up", "k":
		return m.moveUp()
	case "down", "j":
		return m.moveDown()
	case "enter", " ":
		return m.activate()
	}

	return m, nil
}

// View implements tea.Model. It renders all sections and their nodes,
// highlighting the currently selected item. The description of the selected
// item is shown at the bottom when non-empty. View is a pure function — no
// I/O is performed here.
func (m SettingsTreeModel) View() string {
	var sb strings.Builder

	for si, sec := range m.sections {
		// Section header.
		sectionSelected := si == m.selectedSection && m.selectedItem == -1
		sb.WriteString(m.renderSectionHeader(sec, sectionSelected))
		sb.WriteByte('\n')

		// Nodes (only when expanded).
		if sec.Expanded {
			for ni, node := range sec.Nodes {
				nodeSelected := si == m.selectedSection && ni == m.selectedItem
				sb.WriteString(m.renderNode(node, nodeSelected))
				sb.WriteByte('\n')
			}
		}
	}

	// Description footer for the currently selected item.
	desc := m.selectedDescription()
	if desc != "" {
		sb.WriteByte('\n')
		sb.WriteString(config.StyleMuted.Render(desc))
	}

	return strings.TrimRight(sb.String(), "\n")
}

// ---------------------------------------------------------------------------
// Public setters
// ---------------------------------------------------------------------------

// SetSize updates the viewport dimensions.
func (m *SettingsTreeModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetFocused sets whether the panel has keyboard focus.
func (m *SettingsTreeModel) SetFocused(focused bool) {
	m.focused = focused
}

// SetValue updates the Value field of the node identified by key. It is
// intended for Display nodes whose values originate from outside (model name,
// git branch, etc.), but it will update any node type.
// A no-op when key is not found.
func (m *SettingsTreeModel) SetValue(key, value string) {
	for si := range m.sections {
		for ni := range m.sections[si].Nodes {
			if m.sections[si].Nodes[ni].Key == key {
				m.sections[si].Nodes[ni].Value = value
				return
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Navigation helpers
// ---------------------------------------------------------------------------

// moveUp moves the selection one step upward. Within a section it moves from
// node to node; at the top node of a section it moves to the section header;
// at a section header it moves to the last visible item of the previous
// section (or wraps to the end).
func (m SettingsTreeModel) moveUp() (tea.Model, tea.Cmd) {
	if m.selectedItem == -1 {
		// On a section header — move to previous section.
		if m.selectedSection == 0 {
			// Already at the very top; do nothing.
			return m, nil
		}
		m.selectedSection--
		// Land on last node of previous section if expanded, else on header.
		prev := m.sections[m.selectedSection]
		if prev.Expanded && len(prev.Nodes) > 0 {
			m.selectedItem = len(prev.Nodes) - 1
		} else {
			m.selectedItem = -1
		}
		return m, nil
	}

	// On a node — move to the previous node or the section header.
	if m.selectedItem == 0 {
		m.selectedItem = -1
		return m, nil
	}
	m.selectedItem--
	return m, nil
}

// moveDown moves the selection one step downward. Within an expanded section
// it descends through nodes; after the last node (or header of a collapsed
// section) it advances to the next section header.
func (m SettingsTreeModel) moveDown() (tea.Model, tea.Cmd) {
	sec := m.sections[m.selectedSection]

	if m.selectedItem == -1 {
		// On a section header.
		if sec.Expanded && len(sec.Nodes) > 0 {
			m.selectedItem = 0
			return m, nil
		}
		// Section collapsed or empty — advance to next section.
		return m.advanceToNextSection()
	}

	// On a node.
	if m.selectedItem < len(sec.Nodes)-1 {
		m.selectedItem++
		return m, nil
	}

	// At the last node of the section — advance to next section.
	return m.advanceToNextSection()
}

// advanceToNextSection moves the selection to the header of the section after
// the current one. Does nothing when already at the last section.
func (m SettingsTreeModel) advanceToNextSection() (tea.Model, tea.Cmd) {
	if m.selectedSection >= len(m.sections)-1 {
		// Already at last section — do nothing.
		return m, nil
	}
	m.selectedSection++
	m.selectedItem = -1
	return m, nil
}

// ---------------------------------------------------------------------------
// Activation helper
// ---------------------------------------------------------------------------

// activate handles Enter/Space on the currently selected item.
func (m SettingsTreeModel) activate() (tea.Model, tea.Cmd) {
	if m.selectedItem == -1 {
		// Toggle section collapse/expand.
		m.sections[m.selectedSection].Expanded = !m.sections[m.selectedSection].Expanded
		return m, nil
	}

	node := &m.sections[m.selectedSection].Nodes[m.selectedItem]

	switch node.Type {
	case SettingToggle:
		if node.Value == "on" {
			node.Value = "off"
		} else {
			node.Value = "on"
		}
		return m, emitChanged(node.Key, node.Value)

	case SettingSelect:
		if len(node.Options) == 0 {
			return m, nil
		}
		idx := optionIndex(node.Options, node.Value)
		idx = (idx + 1) % len(node.Options)
		node.Value = node.Options[idx]
		return m, emitChanged(node.Key, node.Value)

	case SettingDisplay:
		// Read-only — no-op.
		return m, nil
	}

	return m, nil
}

// emitChanged returns a Cmd that emits SettingChangedMsg.
func emitChanged(key, value string) tea.Cmd {
	return func() tea.Msg {
		return SettingChangedMsg{Key: key, Value: value}
	}
}

// optionIndex returns the index of value in options, or 0 if not found.
func optionIndex(options []string, value string) int {
	for i, o := range options {
		if o == value {
			return i
		}
	}
	return 0
}

// ---------------------------------------------------------------------------
// View helpers
// ---------------------------------------------------------------------------

// renderSectionHeader renders a section header row.
func (m SettingsTreeModel) renderSectionHeader(sec SettingsSection, selected bool) string {
	indicator := "▼"
	if !sec.Expanded {
		indicator = "▸"
	}

	text := indicator + " " + sec.Title
	style := config.StyleTitle
	if selected && m.focused {
		style = lipgloss.NewStyle().
			Bold(true).
			Foreground(config.ColorAccent).
			Reverse(true)
	}
	return style.Render(text)
}

// renderNode renders a single setting node row.
func (m SettingsTreeModel) renderNode(node SettingNode, selected bool) string {
	valueStr := node.Value
	if valueStr == "" {
		valueStr = "\u2014" // em dash for empty display nodes
	}

	var typeHint string
	switch node.Type {
	case SettingToggle:
		if node.Value == "on" {
			typeHint = config.StyleSuccess.Render("[on]")
		} else {
			typeHint = config.StyleMuted.Render("[off]")
		}
		valueStr = typeHint
	case SettingSelect:
		valueStr = config.StyleHighlight.Render(valueStr)
	case SettingDisplay:
		valueStr = config.StyleMuted.Render(valueStr)
	}

	label := "  " + node.Label + ": "

	if selected && m.focused {
		labelStyle := lipgloss.NewStyle().Bold(true).Foreground(config.ColorAccent)
		return labelStyle.Render(label) + valueStr
	}

	labelStyle := lipgloss.NewStyle().Foreground(config.ColorPrimary)
	return labelStyle.Render(label) + valueStr
}

// selectedDescription returns the description of the currently highlighted
// item, or an empty string when no description is available.
func (m SettingsTreeModel) selectedDescription() string {
	if m.selectedItem == -1 {
		return ""
	}
	sec := m.sections[m.selectedSection]
	if m.selectedItem < 0 || m.selectedItem >= len(sec.Nodes) {
		return ""
	}
	return sec.Nodes[m.selectedItem].Description
}
