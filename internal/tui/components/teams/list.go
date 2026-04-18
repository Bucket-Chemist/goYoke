package teams

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Bucket-Chemist/goYoke/internal/tui/config"
	"github.com/Bucket-Chemist/goYoke/internal/tui/model"
	"github.com/Bucket-Chemist/goYoke/internal/tui/state"
)

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

// TeamSelectedMsg is emitted when the user moves the cursor to a team or
// explicitly selects one (Enter). The parent model should update
// TeamDetailModel in response.
type TeamSelectedMsg struct {
	// Dir is the filesystem path for the selected team directory.
	Dir string
}

// pollTickMsg is a package-private message that drives the 2-second polling
// cycle. seq is compared against TeamListModel.pollSeq to discard ticks from
// superseded chains (created by a prior StartPolling or PollNow call).
type pollTickMsg struct {
	time time.Time
	seq  int
}

// ---------------------------------------------------------------------------
// TeamListModel
// ---------------------------------------------------------------------------

// TeamListModel is the Bubbletea sub-model for the scrollable team list. It
// polls the filesystem every 2 seconds and renders each known team as a
// single summary row.
//
// The zero value is not usable; use NewTeamListModel instead.
type TeamListModel struct {
	registry *TeamRegistry
	selected int          // cursor index into teams snapshot
	teams    []*TeamState // snapshot refreshed on every poll/update
	width    int
	height   int
	teamsDir string // directory containing team subdirectories
	polling  bool   // true once polling has been started
	pollSeq  int    // incremented on each StartPolling/PollNow; stale ticks are dropped
}

// NewTeamListModel returns a TeamListModel backed by the given registry.
func NewTeamListModel(registry *TeamRegistry) TeamListModel {
	return TeamListModel{
		registry: registry,
	}
}

// Init implements tea.Model. Init returns nil; polling starts via
// StartPolling so the caller can supply teamsDir before ticks begin.
func (m TeamListModel) Init() tea.Cmd {
	return nil
}

// pollCmd returns a Cmd that fires after 2 seconds with the current pollSeq.
// Ticks carrying a seq that no longer matches m.pollSeq are discarded in
// Update, which makes the chain idempotent across PollNow / StartPolling.
func (m TeamListModel) pollCmd() tea.Cmd {
	seq := m.pollSeq
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return pollTickMsg{time: t, seq: seq}
	})
}

// StartPolling sets the directory to poll and returns a tea.Cmd that
// immediately fires a pollTickMsg to kick off the 2-second polling cycle.
// It increments pollSeq so any tick from a previous chain is discarded.
func (m *TeamListModel) StartPolling(teamsDir string) tea.Cmd {
	m.teamsDir = teamsDir
	m.polling = true
	m.pollSeq++
	seq := m.pollSeq
	return func() tea.Msg {
		return pollTickMsg{time: time.Now(), seq: seq}
	}
}

// PollNow returns a Cmd that immediately fires a pollTickMsg with the new
// pollSeq, making any pending tick from the previous chain stale. It is
// idempotent: calling it repeatedly only advances the seq and kills the old
// chain; the new chain re-establishes the 2-second cadence on the next tick.
// Returns nil when polling has not been started.
func (m *TeamListModel) PollNow() tea.Cmd {
	if !m.polling {
		return nil
	}
	m.pollSeq++
	seq := m.pollSeq
	return func() tea.Msg {
		return pollTickMsg{time: time.Now(), seq: seq}
	}
}

// ScanNow performs an immediate filesystem scan of the teams directory and
// refreshes the local snapshot. Unlike PollNow it does NOT schedule a follow-up
// tick, so it cannot create duplicate poll chains. Use this when a TeamUpdateMsg
// arrives and you need the registry populated before reading drawer content.
func (m *TeamListModel) ScanNow() {
	if m.teamsDir != "" {
		scanTeamsDir(m.teamsDir, m.registry)
	}
	m.teams = m.registry.All()
	m.clampSelected()
}

// SetSize updates the width and height used for rendering.
func (m *TeamListModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetTeamsDir sets the directory that will be scanned during poll ticks.
// This is a low-level setter used by tests; production code uses StartPolling.
func (m *TeamListModel) SetTeamsDir(dir string) {
	m.teamsDir = dir
}

// SelectedTeam returns the directory path of the currently selected team, or
// an empty string when the list is empty.
func (m TeamListModel) SelectedTeam() string {
	if len(m.teams) == 0 || m.selected < 0 || m.selected >= len(m.teams) {
		return ""
	}
	return m.teams[m.selected].Dir
}

// Update implements tea.Model. It handles:
//   - model.TeamUpdateMsg  — refreshes the snapshot from the registry.
//   - pollTickMsg           — scans teamsDir, updates registry, schedules next tick.
//   - tea.KeyMsg            — j/k/up/down navigate the list; enter emits TeamSelectedMsg.
func (m TeamListModel) Update(msg tea.Msg) (TeamListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case model.TeamUpdateMsg:
		m.teams = m.registry.All()
		m.clampSelected()
		return m, nil

	case pollTickMsg:
		// Discard ticks from superseded chains (stale seq).
		if msg.seq != m.pollSeq {
			return m, nil
		}
		// Scan the directory and update the registry for each team found.
		if m.teamsDir != "" {
			scanTeamsDir(m.teamsDir, m.registry)
		}
		// Refresh the local snapshot.
		m.teams = m.registry.All()
		m.clampSelected()
		// Schedule the next poll with the same seq.
		return m, m.pollCmd()

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

// handleKey processes keyboard input for list navigation.
func (m TeamListModel) handleKey(msg tea.KeyMsg) (TeamListModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selected > 0 {
			m.selected--
			return m, emitTeamSelected(m.SelectedTeam())
		}

	case "down", "j":
		if m.selected < len(m.teams)-1 {
			m.selected++
			return m, emitTeamSelected(m.SelectedTeam())
		}

	case "enter":
		dir := m.SelectedTeam()
		if dir != "" {
			return m, emitTeamSelected(dir)
		}
	}

	return m, nil
}

// emitTeamSelected returns a Cmd that emits TeamSelectedMsg for the given dir.
func emitTeamSelected(dir string) tea.Cmd {
	return func() tea.Msg {
		return TeamSelectedMsg{Dir: dir}
	}
}

// clampSelected ensures selected stays within bounds after the teams slice
// changes size.
func (m *TeamListModel) clampSelected() {
	if len(m.teams) == 0 {
		m.selected = 0
		return
	}
	if m.selected >= len(m.teams) {
		m.selected = len(m.teams) - 1
	}
	if m.selected < 0 {
		m.selected = 0
	}
}

// View implements tea.Model. It renders the team list as a scrollable set of
// summary rows. The view is a pure function of the model state — no I/O.
func (m TeamListModel) View() string {
	if len(m.teams) == 0 {
		return config.StyleMuted.Render("No teams")
	}

	var sb strings.Builder
	for i, ts := range m.teams {
		row := m.renderRow(ts, i == m.selected)
		sb.WriteString(row)
		sb.WriteByte('\n')
	}

	return strings.TrimRight(sb.String(), "\n")
}

// renderRow renders a single team summary row.
//
// Format:  [icon] team_name  workflow_type  $cost  Wn/total
func (m TeamListModel) renderRow(ts *TeamState, selected bool) string {
	icon := statusIcon(ts.Config.Status)
	iconStr := statusStyleFor(ts.Config.Status).Render(string(icon))

	name := ts.Config.TeamName
	if name == "" {
		name = filepath.Base(ts.Dir)
	}

	workflow := ts.Config.WorkflowType
	if workflow == "" {
		workflow = "—"
	}

	cost := fmt.Sprintf("$%.2f", ts.TotalCostUSD())

	totalWaves := len(ts.Config.Waves)
	waveProg := "—"
	if totalWaves > 0 {
		cur := ts.CurrentWaveNumber()
		waveProg = fmt.Sprintf("W%d/%d", cur, totalWaves)
	}

	line := fmt.Sprintf("%s %-24s  %-16s  %6s  %s",
		iconStr, name, workflow, cost, waveProg)

	if selected {
		// Highlight selected row.
		plainLine := fmt.Sprintf("%s %-24s  %-16s  %6s  %s",
			string(icon), name, workflow, cost, waveProg)
		return config.StyleHighlight.Render(plainLine)
	}

	return line
}

// ---------------------------------------------------------------------------
// Status icon + style helpers (team-level, based on string status)
// ---------------------------------------------------------------------------

// statusIcon returns the ASCII icon for a string status value from config.json.
func statusIcon(status string) rune {
	switch status {
	case "running":
		return config.IconRunning
	case "completed":
		return config.IconComplete
	case "failed":
		return config.IconError
	default: // "pending" and anything unknown
		return config.IconPending
	}
}

// statusStyleFor returns the lipgloss style for the given string status.
func statusStyleFor(status string) lipgloss.Style {
	switch status {
	case "running":
		return config.StyleWarning
	case "completed":
		return config.StyleSuccess
	case "failed":
		return config.StyleError
	default:
		return config.StyleMuted
	}
}

// ---------------------------------------------------------------------------
// scanTeamsDir
// ---------------------------------------------------------------------------

// HandleMsg is the pointer-receiver equivalent of Update. It mutates the
// model in place and returns only the tea.Cmd. This satisfies the
// teamListWidget interface defined in the model package.
func (m *TeamListModel) HandleMsg(msg tea.Msg) tea.Cmd {
	updated, cmd := m.Update(msg)
	*m = updated
	return cmd
}

// CreateDetailModel returns a preconfigured TeamDetailModel backed by the
// same registry as this list. agentReg may be nil. The concrete type
// satisfies model.TeamDetailWidget without importing the model package.
func (m *TeamListModel) CreateDetailModel(agentReg *state.AgentRegistry) model.TeamDetailWidget {
	td := NewTeamDetailModel(m.registry, agentReg)
	return &td
}

// scanTeamsDir reads every subdirectory of dir, attempts to parse
// config.json, and calls registry.Update for any successfully parsed team.
// Errors are silently ignored so a single corrupt config.json does not block
// the rest of the poll.
func scanTeamsDir(dir string, reg *TeamRegistry) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		// Accept both real directories and symlinks that resolve to directories
		// (ensureTeamVisible creates symlinks for cross-session team visibility).
		if !entry.IsDir() {
			if entry.Type()&os.ModeSymlink == 0 {
				continue
			}
			target := filepath.Join(dir, entry.Name())
			info, err := os.Stat(target) // Stat follows symlinks
			if err != nil || !info.IsDir() {
				continue
			}
		}
		teamDir := filepath.Join(dir, entry.Name())
		cfgPath := filepath.Join(teamDir, "config.json")

		data, err := os.ReadFile(cfgPath)
		if err != nil {
			continue
		}

		var cfg TeamConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			continue
		}

		// Stat stream files for activity tracking.
		var streamSizes map[string]int64
		streamEntries, _ := filepath.Glob(filepath.Join(teamDir, "stream_*.ndjson"))
		if len(streamEntries) > 0 {
			streamSizes = make(map[string]int64, len(streamEntries))
			for _, path := range streamEntries {
				base := filepath.Base(path)
				name := strings.TrimPrefix(base, "stream_")
				name = strings.TrimSuffix(name, ".ndjson")
				if info, err := os.Stat(path); err == nil {
					streamSizes[name] = info.Size()
				}
			}
		}

		reg.Update(teamDir, cfg, streamSizes)
	}
}
