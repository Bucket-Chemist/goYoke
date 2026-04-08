package model

// BenchmarkStartup and BenchmarkViewRendering live here because they exercise
// the model package directly. Internal tests (package model) can construct
// AppModel via NewAppModel and access unexported rendering helpers.
//
// Target thresholds (TUI-040):
//   - BenchmarkStartup:       < 200 ms
//   - BenchmarkViewRendering: < 16 ms per frame (60 fps)
//
// Run:
//
//	go test -bench=Benchmark -benchmem -count=5 ./internal/tui/model/

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
)

// ---------------------------------------------------------------------------
// BenchmarkStartup
//
// Measures the cost of creating an AppModel plus the minimal wiring that
// occurs before the Bubbletea program starts: NewAppModel + injecting mocks
// for every widget slot + processing the first WindowSizeMsg (which triggers
// ready=true and propagates dimensions to every child component).
//
// A real application also calls tea.NewProgram and connects a CLI subprocess,
// but those operations require a TTY and an external binary. The benchmark
// covers the in-process portion of startup, which is the portion attributable
// to Go binary startup overhead.
// ---------------------------------------------------------------------------

// BenchmarkStartup measures NewAppModel() plus initial WindowSizeMsg dispatch.
func BenchmarkStartup(b *testing.B) {
	b.ReportAllocs()

	b.ResetTimer()
	for b.Loop() {
		m := NewAppModel()

		// Inject lightweight mocks so that WindowSizeMsg propagation
		// exercises all optional if-branches in Update.
		m.SetClaudePanel(&benchClaudePanel{})
		m.SetToasts(&benchToast{})
		m.SetTeamList(&benchTeamList{})
		m.SetTabBar(&benchTabBar{})

		// Fire the first WindowSizeMsg — sets ready=true and distributes
		// dimensions to all child components.
		updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
		_ = updated.(AppModel).View()
	}
}

// ---------------------------------------------------------------------------
// BenchmarkViewRendering
//
// Measures the cost of one View() call after the model has been initialised
// with a realistic terminal size and child component mocks that return
// non-trivial rendered strings (simulating 100 message lines visible in the
// Claude panel, the agent tree empty, status line + banner chrome).
//
// Target: < 16 ms per call (60 fps budget).
// ---------------------------------------------------------------------------

// BenchmarkViewRendering measures View() at 120x40 with realistic mock data.
func BenchmarkViewRendering(b *testing.B) {
	b.ReportAllocs()

	m := benchNewModel(120, 40, benchLargeClaudeContent())

	b.ResetTimer()
	for b.Loop() {
		_ = m.View()
	}
}

// BenchmarkViewRenderingWithAgents measures View() when the agent tree is
// populated with 10 nodes at various depths — the common production state.
func BenchmarkViewRenderingWithAgents(b *testing.B) {
	b.ReportAllocs()

	m := benchNewModel(120, 40, benchLargeClaudeContent())

	// Populate the agent tree with 10 synthetic nodes.
	m.agentTree.SetNodes(benchMakeAgentNodes(10))

	b.ResetTimer()
	for b.Loop() {
		_ = m.View()
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// benchNewModel returns an AppModel initialised at the given terminal size
// with all widget slots filled by lightweight mocks.
func benchNewModel(width, height int, claudeContent string) AppModel {
	m := NewAppModel()

	m.SetClaudePanel(&benchClaudePanel{content: claudeContent})
	m.SetToasts(&benchToast{})
	m.SetTeamList(&benchTeamList{})
	m.SetTabBar(&benchTabBar{})

	// Apply WindowSizeMsg to set ready=true and propagate dimensions.
	updated, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return updated.(AppModel)
}

// benchLargeClaudeContent returns a multi-line string that simulates 100
// rendered conversation lines in the Claude panel.
func benchLargeClaudeContent() string {
	const line = "You: What is the best approach to implementing a concurrent HTTP client in Go?\nClaude: Use errgroup for coordination and a semaphore for rate-limiting concurrent requests.\n"
	var content string
	for range 50 {
		content += line
	}
	return content
}

// benchMakeAgentNodes returns n synthetic AgentTreeNode pointers for
// populating AgentTreeModel.SetNodes in benchmarks.
func benchMakeAgentNodes(n int) []*state.AgentTreeNode {
	nodes := make([]*state.AgentTreeNode, n)
	for i := range n {
		depth := i % 3 // depths 0,1,2 cycling
		agent := &state.Agent{
			ID:          benchID(i),
			AgentType:   "go-pro",
			Description: "implement feature",
			Status:      state.StatusRunning,
		}
		nodes[i] = &state.AgentTreeNode{
			Agent:  agent,
			Depth:  depth,
			IsLast: i == n-1,
		}
	}
	return nodes
}

// benchID returns a short deterministic string ID for index i.
func benchID(i int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	id := make([]byte, 4)
	v := i
	for j := range id {
		id[j] = alphabet[v%len(alphabet)]
		v /= len(alphabet)
	}
	return string(id)
}

// ---------------------------------------------------------------------------
// Minimal mock types — only interface methods required by AppModel.
// ---------------------------------------------------------------------------

// benchClaudePanel satisfies claudePanelWidget.
type benchClaudePanel struct {
	content string
}

func (p *benchClaudePanel) HandleMsg(_ tea.Msg) tea.Cmd              { return nil }
func (p *benchClaudePanel) View() string                              { return p.content }
func (p *benchClaudePanel) SetSize(_, _ int)                         {}
func (p *benchClaudePanel) SetFocused(_ bool)                        {}
func (p *benchClaudePanel) IsStreaming() bool                        { return false }
func (p *benchClaudePanel) SaveMessages() []state.DisplayMessage     { return nil }
func (p *benchClaudePanel) RestoreMessages(_ []state.DisplayMessage) {}
func (p *benchClaudePanel) SetSender(_ MessageSender)                {}
func (p *benchClaudePanel) AppendSystemMessage(_ string)             {}
func (p *benchClaudePanel) SetTier(_ LayoutTier)                     {}
func (p *benchClaudePanel) ViewConversation() string                 { return p.content }
func (p *benchClaudePanel) ViewInput() string                        { return "" }
func (p *benchClaudePanel) ApplyOverlay(composed string) string      { return composed }

// benchToast satisfies toastWidget.
type benchToast struct{}

func (t *benchToast) HandleMsg(_ tea.Msg) tea.Cmd { return nil }
func (t *benchToast) View() string                 { return "" }
func (t *benchToast) SetSize(_, _ int)             {}
func (t *benchToast) IsEmpty() bool                { return true }
func (t *benchToast) Height() int                  { return 0 }
func (t *benchToast) SetTier(_ LayoutTier)         {}

// benchTeamList satisfies teamListWidget.
type benchTeamList struct{}

func (tl *benchTeamList) HandleMsg(_ tea.Msg) tea.Cmd                                        { return nil }
func (tl *benchTeamList) View() string                                                        { return "" }
func (tl *benchTeamList) SetSize(_, _ int)                                                    {}
func (tl *benchTeamList) StartPolling(_ string) tea.Cmd                                      { return nil }
func (tl *benchTeamList) PollNow() tea.Cmd                                                   { return nil }
func (tl *benchTeamList) SelectedTeam() string                                               { return "" }
func (tl *benchTeamList) CreateDetailModel(_ *state.AgentRegistry) TeamDetailWidget          { return nil }

// benchTabBar satisfies tabBarWidget.
type benchTabBar struct{}

func (tb *benchTabBar) View() string                { return "Chat | Agent Config | Team Config | Telemetry" }
func (tb *benchTabBar) SetWidth(_ int)              {}
func (tb *benchTabBar) HandleMsg(_ tea.Msg) tea.Cmd { return nil }
func (tb *benchTabBar) ActiveTab() TabID            { return TabChat }
