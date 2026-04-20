// Package cwdselector implements a modal overlay for changing the working
// directory used by Claude Code sessions. By changing the CWD, users control
// the scope of CC's hardcoded write restrictions (CC can write to CWD and
// subdirectories). Setting CWD to "/" grants unrestricted write access.
//
// The Model satisfies the cwdSelectorWidget interface defined in model/interfaces.go.
// It uses the pointer-receiver mutation pattern: HandleMsg mutates the model
// in place and returns only the tea.Cmd, avoiding the self-returning interface
// problem.
package cwdselector

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/goYoke/internal/tui/config"
	"github.com/Bucket-Chemist/goYoke/internal/tui/model"
)

// preset is a directory entry in the selector list.
type preset struct {
	Path  string // Absolute path
	Label string // Display label (e.g., "home", "root", "proj")
}

// Model is the CWD selector modal. Use pointer receivers throughout so it
// can be stored in sharedState and mutated via the cwdSelectorWidget interface.
type Model struct {
	active    bool
	presets   []preset
	cursor    int
	custom    bool // true when the text input is focused
	textInput textinput.Model
	width     int
	height    int
}

// Package-level styles.
var (
	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(config.ColorPrimary).
			Padding(1, 2)

	titleStyle    = config.StyleTitle
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(config.ColorPrimary)
	normalStyle   = config.StyleMuted
	hintStyle     = config.StyleSubtle
	projLabelSty  = lipgloss.NewStyle().Foreground(config.ColorSuccess)
	homeLabelSty  = lipgloss.NewStyle().Foreground(config.ColorWarning)
	rootLabelSty  = lipgloss.NewStyle().Bold(true).Foreground(config.ColorError)
)

// New returns a CWD selector with auto-discovered project presets.
func New() *Model {
	ti := textinput.New()
	ti.Placeholder = "/path/to/directory"
	ti.CharLimit = 256

	return &Model{
		presets:   discoverPresets(),
		textInput: ti,
	}
}

// IsActive returns true when the modal is visible.
func (m *Model) IsActive() bool { return m.active }

// Show makes the modal visible and resets state.
func (m *Model) Show() {
	m.active = true
	m.cursor = 0
	m.custom = false
	m.textInput.Reset()
	m.presets = discoverPresets() // refresh on each open
}

// SetSize updates the terminal dimensions for centering.
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// HandleMsg processes a tea.Msg, mutates the model in place, and returns
// any Cmd to run. This follows the pointer-receiver mutation pattern used
// by searchOverlayWidget and other sharedState widgets.
func (m *Model) HandleMsg(msg tea.Msg) tea.Cmd {
	if !m.active {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return nil
}

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	// Custom path input mode.
	if m.custom {
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			m.custom = false
			m.textInput.Blur()
			return nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			path := strings.TrimSpace(m.textInput.Value())
			if path == "" {
				return nil
			}
			return m.selectPath(path)
		default:
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return cmd
		}
	}

	// Preset list navigation.
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
		m.active = false
		return nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		if m.cursor > 0 {
			m.cursor--
		}
		return nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		if m.cursor < len(m.presets)-1 {
			m.cursor++
		}
		return nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		if m.cursor >= 0 && m.cursor < len(m.presets) {
			return m.selectPath(m.presets[m.cursor].Path)
		}
		return nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("/"))):
		m.custom = true
		m.textInput.Focus()
		return textinput.Blink
	}

	return nil
}

// selectPath validates the path and emits CWDChangedMsg.
func (m *Model) selectPath(path string) tea.Cmd {
	// Expand ~ to home directory.
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[1:])
		}
	}

	// Resolve to absolute.
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil
	}

	// Validate directory exists.
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return nil
	}

	m.active = false
	return func() tea.Msg {
		return model.CWDChangedMsg{Path: abs}
	}
}

// View renders the modal overlay. Returns "" when not active.
func (m *Model) View() string {
	if !m.active {
		return ""
	}

	cwd, _ := os.Getwd()

	var b strings.Builder

	b.WriteString(titleStyle.Render("Change Working Directory"))
	b.WriteString("\n\n")
	b.WriteString(hintStyle.Render("Current: " + shortenPath(cwd)))
	b.WriteString("\n\n")

	// Preset list.
	for i, p := range m.presets {
		cursor := "  "
		style := normalStyle
		if i == m.cursor && !m.custom {
			cursor = "▸ "
			style = selectedStyle
		}

		label := renderLabel(p.Label)
		b.WriteString(fmt.Sprintf("%s%s  %s\n",
			cursor,
			style.Render(shortenPath(p.Path)),
			label,
		))
	}

	b.WriteString("\n")
	b.WriteString(hintStyle.Render("────────────────────────────────"))
	b.WriteString("\n")

	// Custom path input.
	if m.custom {
		b.WriteString("  Custom: ")
		b.WriteString(m.textInput.View())
	} else {
		b.WriteString(hintStyle.Render("  Press / for custom path"))
	}

	b.WriteString("\n\n")
	b.WriteString(hintStyle.Render("↑↓ navigate  enter select  / custom  esc cancel"))

	box := borderStyle.Width(44).Render(b.String())

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

// renderLabel returns a styled label string for a preset type.
func renderLabel(label string) string {
	switch label {
	case "proj":
		return projLabelSty.Render("(" + label + ")")
	case "home":
		return homeLabelSty.Render("(" + label + ")")
	case "root":
		return rootLabelSty.Render("(" + label + ")")
	default:
		return normalStyle.Render("(" + label + ")")
	}
}

// shortenPath replaces the home directory prefix with ~.
func shortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

// discoverPresets scans for Claude-configured projects and builds the preset list.
func discoverPresets() []preset {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/"
	}

	projectRoot := os.Getenv("GOYOKE_PROJECT_ROOT")
	if projectRoot == "" {
		projectRoot = filepath.Join(home, "Documents", "goYoke")
	}

	presets := []preset{
		{Path: projectRoot, Label: "proj"},
		{Path: home, Label: "home"},
		{Path: "/", Label: "root"},
	}

	// Scan ~/Documents for other Claude-configured projects.
	docsDir := filepath.Join(home, "Documents")
	entries, err := os.ReadDir(docsDir)
	if err != nil {
		return presets
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(docsDir, e.Name())

		// Skip the main project (already listed).
		if dir == projectRoot {
			continue
		}

		// Check for .claude/CLAUDE.md marker.
		claudeMD := filepath.Join(dir, ".claude", "CLAUDE.md")
		if _, err := os.Stat(claudeMD); err == nil {
			presets = append(presets, preset{Path: dir, Label: e.Name()})
		}
	}

	return presets
}
