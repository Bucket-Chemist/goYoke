package session

import (
	"strings"
	"testing"
)

func TestRenderHandoffMarkdown_Minimal(t *testing.T) {
	handoff := &Handoff{
		SchemaVersion: "1.0",
		Timestamp:     1234567890,
		SessionID:     "test-minimal",
		Context: SessionContext{
			ProjectDir: "/test/project",
			Metrics: SessionMetrics{
				ToolCalls:         10,
				ErrorsLogged:      0,
				RoutingViolations: 0,
				SessionID:         "test-minimal",
			},
		},
		Artifacts: HandoffArtifacts{},
		Actions:   []Action{},
	}

	markdown := RenderHandoffMarkdown(handoff)

	// Check required sections
	requiredSections := []string{
		"# Session Handoff",
		"## Session Context",
		"Session ID**: test-minimal",
		"Project**: /test/project",
		"## Session Metrics",
		"Tool Calls**: 10",
		"Errors Logged**: 0",
		"Routing Violations**: 0",
	}

	for _, section := range requiredSections {
		if !strings.Contains(markdown, section) {
			t.Errorf("Missing required section: %s", section)
		}
	}
}

func TestRenderHandoffMarkdown_WithArtifacts(t *testing.T) {
	handoff := &Handoff{
		SchemaVersion: "1.0",
		Timestamp:     1234567890,
		SessionID:     "test-artifacts",
		Context: SessionContext{
			ProjectDir: "/test/project",
			Metrics: SessionMetrics{
				ToolCalls:         50,
				ErrorsLogged:      5,
				RoutingViolations: 2,
				SessionID:         "test-artifacts",
			},
		},
		Artifacts: HandoffArtifacts{
			SharpEdges: []SharpEdge{
				{File: "test.go", ErrorType: "nil_pointer", ConsecutiveFailures: 3, Context: "test context"},
				{File: "main.go", ErrorType: "type_mismatch", ConsecutiveFailures: 2},
			},
			RoutingViolations: []RoutingViolation{
				{Agent: "test-agent", ViolationType: "wrong_tier", ExpectedTier: "haiku", ActualTier: "sonnet"},
			},
			ErrorPatterns: []ErrorPattern{
				{ErrorType: "import_error", Count: 5, Context: "missing module"},
			},
		},
		Actions: []Action{
			{Priority: 1, Description: "Review sharp edges", Context: "Fix before continuing"},
			{Priority: 2, Description: "Check violations", Context: "Pattern issue"},
		},
	}

	markdown := RenderHandoffMarkdown(handoff)

	// Check artifact sections
	artifactSections := []string{
		"## Sharp Edges",
		"test.go**: nil_pointer (3 consecutive failures)",
		"Context: test context",
		"main.go**: type_mismatch (2 consecutive failures)",
		"## Routing Violations",
		"test-agent**: wrong_tier (expected: haiku, actual: sonnet)",
		"## Error Patterns",
		"import_error**: 5 occurrences",
		"Context: missing module",
		"## Immediate Actions",
		"1. Review sharp edges",
		"Fix before continuing",
		"2. Check violations",
		"Pattern issue",
	}

	for _, section := range artifactSections {
		if !strings.Contains(markdown, section) {
			t.Errorf("Missing artifact section: %s", section)
		}
	}
}

func TestRenderHandoffMarkdown_WithGitInfo(t *testing.T) {
	handoff := &Handoff{
		SchemaVersion: "1.0",
		Timestamp:     1234567890,
		SessionID:     "test-git",
		Context: SessionContext{
			ProjectDir: "/test/project",
			Metrics: SessionMetrics{
				ToolCalls:         20,
				ErrorsLogged:      0,
				RoutingViolations: 0,
				SessionID:         "test-git",
			},
			GitInfo: GitInfo{
				Branch:      "feature/test",
				IsDirty:     true,
				Uncommitted: []string{"test.go", "main.go"},
			},
		},
		Artifacts: HandoffArtifacts{},
		Actions:   []Action{},
	}

	markdown := RenderHandoffMarkdown(handoff)

	gitSections := []string{
		"## Git State",
		"Branch**: feature/test",
		"Status**: Uncommitted changes present",
		"Uncommitted Files**:",
		"test.go",
		"main.go",
	}

	for _, section := range gitSections {
		if !strings.Contains(markdown, section) {
			t.Errorf("Missing git section: %s", section)
		}
	}
}

func TestRenderHandoffMarkdown_GitClean(t *testing.T) {
	handoff := &Handoff{
		SchemaVersion: "1.0",
		Timestamp:     1234567890,
		SessionID:     "test-git-clean",
		Context: SessionContext{
			ProjectDir: "/test/project",
			Metrics: SessionMetrics{
				ToolCalls:         15,
				ErrorsLogged:      0,
				RoutingViolations: 0,
				SessionID:         "test-git-clean",
			},
			GitInfo: GitInfo{
				Branch:  "main",
				IsDirty: false,
			},
		},
		Artifacts: HandoffArtifacts{},
		Actions:   []Action{},
	}

	markdown := RenderHandoffMarkdown(handoff)

	if !strings.Contains(markdown, "Branch**: main") {
		t.Error("Missing branch info")
	}

	if !strings.Contains(markdown, "Status**: Clean") {
		t.Error("Expected clean status")
	}

	if strings.Contains(markdown, "Uncommitted Files") {
		t.Error("Should not show uncommitted files for clean repo")
	}
}

func TestRenderHandoffMarkdown_WithActiveTicket(t *testing.T) {
	handoff := &Handoff{
		SchemaVersion: "1.0",
		Timestamp:     1234567890,
		SessionID:     "test-ticket",
		Context: SessionContext{
			ProjectDir:   "/test/project",
			ActiveTicket: "GOgent-028",
			Metrics: SessionMetrics{
				ToolCalls:         30,
				ErrorsLogged:      1,
				RoutingViolations: 0,
				SessionID:         "test-ticket",
			},
		},
		Artifacts: HandoffArtifacts{},
		Actions:   []Action{},
	}

	markdown := RenderHandoffMarkdown(handoff)

	if !strings.Contains(markdown, "Active Ticket**: GOgent-028") {
		t.Error("Missing active ticket")
	}
}

func TestRenderHandoffMarkdown_WithPhase(t *testing.T) {
	handoff := &Handoff{
		SchemaVersion: "1.0",
		Timestamp:     1234567890,
		SessionID:     "test-phase",
		Context: SessionContext{
			ProjectDir: "/test/project",
			Phase:      "implementation",
			Metrics: SessionMetrics{
				ToolCalls:         40,
				ErrorsLogged:      2,
				RoutingViolations: 0,
				SessionID:         "test-phase",
			},
		},
		Artifacts: HandoffArtifacts{},
		Actions:   []Action{},
	}

	markdown := RenderHandoffMarkdown(handoff)

	if !strings.Contains(markdown, "Phase**: implementation") {
		t.Error("Missing phase")
	}
}

func TestRenderHandoffSummary_Minimal(t *testing.T) {
	handoff := &Handoff{
		Timestamp: 1234567890,
		SessionID: "test-summary",
		Context: SessionContext{
			Metrics: SessionMetrics{
				ToolCalls: 25,
			},
		},
		Artifacts: HandoffArtifacts{},
	}

	summary := RenderHandoffSummary(handoff)

	if !strings.Contains(summary, "Session test-summary") {
		t.Error("Missing session ID in summary")
	}

	if !strings.Contains(summary, "25 tool calls") {
		t.Error("Missing tool calls count in summary")
	}

	// Should not have artifact counts
	if strings.Contains(summary, "sharp edge") {
		t.Error("Should not mention sharp edges when none present")
	}
}

func TestRenderHandoffSummary_WithArtifacts(t *testing.T) {
	handoff := &Handoff{
		Timestamp: 1234567890,
		SessionID: "test-summary-full",
		Context: SessionContext{
			Metrics: SessionMetrics{
				ToolCalls: 50,
			},
		},
		Artifacts: HandoffArtifacts{
			SharpEdges: []SharpEdge{
				{File: "test1.go"},
				{File: "test2.go"},
			},
			RoutingViolations: []RoutingViolation{
				{Agent: "agent1"},
			},
		},
	}

	summary := RenderHandoffSummary(handoff)

	if !strings.Contains(summary, "50 tool calls") {
		t.Error("Missing tool calls")
	}

	if !strings.Contains(summary, "2 sharp edge(s)") {
		t.Error("Missing sharp edges count")
	}

	if !strings.Contains(summary, "1 violation(s)") {
		t.Error("Missing violations count")
	}
}

func TestRenderAllHandoffs_Empty(t *testing.T) {
	markdown := RenderAllHandoffs([]Handoff{})

	if !strings.Contains(markdown, "Total sessions: 0") {
		t.Error("Missing total sessions count")
	}

	if !strings.Contains(markdown, "No sessions recorded") {
		t.Error("Missing empty message")
	}
}

func TestRenderAllHandoffs_Multiple(t *testing.T) {
	handoffs := []Handoff{
		{
			Timestamp: 1234567890,
			SessionID: "session-1",
			Context: SessionContext{
				ProjectDir: "/test",
				Metrics: SessionMetrics{
					ToolCalls: 10,
				},
			},
			Artifacts: HandoffArtifacts{},
			Actions:   []Action{},
		},
		{
			Timestamp: 1234567900,
			SessionID: "session-2",
			Context: SessionContext{
				ProjectDir: "/test",
				Metrics: SessionMetrics{
					ToolCalls: 20,
				},
			},
			Artifacts: HandoffArtifacts{},
			Actions:   []Action{},
		},
		{
			Timestamp: 1234567910,
			SessionID: "session-3",
			Context: SessionContext{
				ProjectDir: "/test",
				Metrics: SessionMetrics{
					ToolCalls: 30,
				},
			},
			Artifacts: HandoffArtifacts{},
			Actions:   []Action{},
		},
	}

	markdown := RenderAllHandoffs(handoffs)

	if !strings.Contains(markdown, "Total sessions: 3") {
		t.Error("Missing total sessions count")
	}

	if !strings.Contains(markdown, "## Session Summary") {
		t.Error("Missing session summary section")
	}

	if !strings.Contains(markdown, "## Most Recent Session") {
		t.Error("Missing most recent session section")
	}

	// Check all sessions are listed in summary
	if !strings.Contains(markdown, "session-1") {
		t.Error("Missing session-1 in summary")
	}

	if !strings.Contains(markdown, "session-2") {
		t.Error("Missing session-2 in summary")
	}

	if !strings.Contains(markdown, "session-3") {
		t.Error("Missing session-3 in summary")
	}

	// Most recent should be session-3
	lines := strings.Split(markdown, "\n")
	foundMostRecent := false
	for i, line := range lines {
		if strings.Contains(line, "## Most Recent Session") {
			// Check subsequent lines for session-3
			for j := i; j < len(lines) && j < i+10; j++ {
				if strings.Contains(lines[j], "session-3") {
					foundMostRecent = true
					break
				}
			}
			break
		}
	}

	if !foundMostRecent {
		t.Error("Most recent session should be session-3")
	}
}

func TestRenderHandoffMarkdown_NoArtifacts(t *testing.T) {
	handoff := &Handoff{
		SchemaVersion: "1.0",
		Timestamp:     1234567890,
		SessionID:     "test-no-artifacts",
		Context: SessionContext{
			ProjectDir: "/test/project",
			Metrics: SessionMetrics{
				ToolCalls:         8,
				ErrorsLogged:      0,
				RoutingViolations: 0,
				SessionID:         "test-no-artifacts",
			},
		},
		Artifacts: HandoffArtifacts{
			SharpEdges:        []SharpEdge{},
			RoutingViolations: []RoutingViolation{},
			ErrorPatterns:     []ErrorPattern{},
		},
		Actions: []Action{},
	}

	markdown := RenderHandoffMarkdown(handoff)

	// Should not have artifact sections
	artifactSections := []string{
		"## Sharp Edges",
		"## Routing Violations",
		"## Error Patterns",
		"## Immediate Actions",
	}

	for _, section := range artifactSections {
		if strings.Contains(markdown, section) {
			t.Errorf("Should not have section: %s", section)
		}
	}

	// But should still have core sections
	coreSections := []string{
		"## Session Context",
		"## Session Metrics",
	}

	for _, section := range coreSections {
		if !strings.Contains(markdown, section) {
			t.Errorf("Missing core section: %s", section)
		}
	}
}
