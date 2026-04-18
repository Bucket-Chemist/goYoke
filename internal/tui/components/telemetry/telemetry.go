// Package telemetry implements the routing-decisions telemetry panel for the
// goYoke TUI. It reads the routing-decisions JSONL file and renders
// the last 50 entries in a scrollable viewport.
//
// Loading is performed via a tea.Cmd (LoadEntriesCmd) so no I/O occurs in
// View. The component follows the Bubbletea widget pattern used by other
// goYoke panels.
package telemetry

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/goYoke/internal/tui/config"
	"github.com/Bucket-Chemist/goYoke/internal/tui/model"
)

// ---------------------------------------------------------------------------
// Message types
// ---------------------------------------------------------------------------

// TelemetryLoadedMsg is produced by LoadEntriesCmd when the JSONL file has
// been read. Err is non-nil when the file could not be read or parsed.
type TelemetryLoadedMsg struct {
	Entries []RoutingEntry
	Err     error
}

// ---------------------------------------------------------------------------
// RoutingEntry
// ---------------------------------------------------------------------------

// RoutingEntry represents a single row from the routing-decisions JSONL file.
// Fields are intentionally loose strings to avoid version coupling.
type RoutingEntry struct {
	Timestamp string `json:"timestamp"`
	Agent     string `json:"agent"`
	Tier      string `json:"tier"`
	Decision  string `json:"decision"`
}

// maxEntries is the maximum number of entries rendered to avoid viewport overflow.
const maxEntries = 50
const maxTelemetryLineBytes = 10 * 1024 * 1024

// ---------------------------------------------------------------------------
// TelemetryModel
// ---------------------------------------------------------------------------

// TelemetryModel is the display model for the telemetry panel.
//
// The zero value is not usable; use NewTelemetryModel instead.
type TelemetryModel struct {
	width    int
	height   int
	entries  []RoutingEntry
	viewport viewport.Model
	loaded   bool
	loadErr  string
}

// NewTelemetryModel returns a TelemetryModel with an initialised viewport.
func NewTelemetryModel() TelemetryModel {
	vp := viewport.New(0, 0)
	return TelemetryModel{
		viewport: vp,
	}
}

// SetSize updates the rendering dimensions and resizes the viewport. Call
// this on every tea.WindowSizeMsg.
func (m *TelemetryModel) SetSize(w, h int) {
	m.width = w
	// Reserve 3 rows for the header (title + divider + blank).
	contentH := h - 3
	if contentH < 1 {
		contentH = 1
	}
	m.height = h
	m.viewport.Width = w
	m.viewport.Height = contentH
	// Re-render content at new size if entries are already loaded.
	if m.loaded {
		m.viewport.SetContent(m.renderEntries())
	}
}

// SetTier satisfies the telemetryWidget interface.  Tier-specific rendering
// adaptations are reserved for a future ticket; this is a no-op placeholder.
func (m *TelemetryModel) SetTier(_ model.LayoutTier) {}

// HandleMsg handles TelemetryLoadedMsg and viewport scroll messages. It
// satisfies the telemetryWidget interface used by AppModel.
func (m *TelemetryModel) HandleMsg(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case TelemetryLoadedMsg:
		if msg.Err != nil {
			m.loadErr = msg.Err.Error()
			m.loaded = true
			return nil
		}
		m.entries = msg.Entries
		m.loadErr = ""
		m.loaded = true
		m.viewport.SetContent(m.renderEntries())
		return nil

	default:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return cmd
	}
}

// ---------------------------------------------------------------------------
// LoadEntriesCmd
// ---------------------------------------------------------------------------

// LoadEntriesCmd returns a tea.Cmd that reads up to maxEntries routing
// decision entries from the JSONL file at path. The result is delivered as a
// TelemetryLoadedMsg.
func LoadEntriesCmd(path string) tea.Cmd {
	return func() tea.Msg {
		entries, err := loadJSONL(path)
		return TelemetryLoadedMsg{Entries: entries, Err: err}
	}
}

// loadJSONL reads the JSONL file at path and returns the last maxEntries
// RoutingEntry values.
func loadJSONL(path string) ([]RoutingEntry, error) {
	f, err := os.Open(path) //nolint:gosec // path is controlled by caller
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close() //nolint:errcheck

	var all []RoutingEntry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), maxTelemetryLineBytes)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry RoutingEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Skip malformed lines — partial writes are common in JSONL files.
			continue
		}
		all = append(all, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}

	// Return only the last maxEntries entries to avoid viewport overflow.
	if len(all) > maxEntries {
		all = all[len(all)-maxEntries:]
	}
	return all, nil
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

// View renders the telemetry panel. It is a pure function of the model
// state — no I/O is performed here.
func (m TelemetryModel) View() string {
	var sb strings.Builder

	// Header.
	sb.WriteString(config.StyleTitle.Render("Routing Telemetry"))
	sb.WriteByte('\n')
	sb.WriteString(config.StyleSubtle.Render(divider(m.width)))
	sb.WriteByte('\n')

	if !m.loaded {
		sb.WriteString(config.StyleSubtle.Render("Loading..."))
		return sb.String()
	}

	if m.loadErr != "" {
		sb.WriteString(config.StyleError.Render("Error: " + m.loadErr))
		return sb.String()
	}

	if len(m.entries) == 0 {
		sb.WriteString(config.StyleSubtle.Render("No routing decisions recorded yet."))
		return sb.String()
	}

	sb.WriteString(m.viewport.View())
	return sb.String()
}

// renderEntries builds the viewport content string from the current entries.
func (m TelemetryModel) renderEntries() string {
	var sb strings.Builder
	for _, e := range m.entries {
		line := m.renderEntry(e)
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	return strings.TrimRight(sb.String(), "\n")
}

// renderEntry renders a single RoutingEntry as a compact one-line row.
func (m TelemetryModel) renderEntry(e RoutingEntry) string {
	ts := config.StyleMuted.Render(shortTimestamp(e.Timestamp))
	tier := config.StyleHighlight.Render(fmt.Sprintf("%-8s", orDash(e.Tier)))
	agent := config.StyleSubtle.Render(fmt.Sprintf("%-20s", orDash(e.Agent)))
	decision := orDash(e.Decision)
	return fmt.Sprintf("%s %s %s %s", ts, tier, agent, decision)
}

// shortTimestamp truncates an ISO timestamp to just the time part (HH:MM:SS).
func shortTimestamp(ts string) string {
	if len(ts) >= 19 {
		// ISO 8601: "2006-01-02T15:04:05..."
		return ts[11:19]
	}
	if ts == "" {
		return "--:--:--"
	}
	return ts
}

// orDash returns s when non-empty, otherwise an em-dash placeholder.
func orDash(s string) string {
	if s == "" {
		return "\u2014"
	}
	return s
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
