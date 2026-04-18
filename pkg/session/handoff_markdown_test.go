package session

import (
	"strings"
	"testing"
	"time"
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
			ActiveTicket: "goYoke-028",
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

	if !strings.Contains(markdown, "Active Ticket**: goYoke-028") {
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

func TestFormatWeeklyIntentSummary(t *testing.T) {
	summary := WeeklyIntentSummary{
		WeekStart:    time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		WeekEnd:      time.Date(2026, 1, 22, 0, 0, 0, 0, time.UTC),
		TotalIntents: 23,
		SessionCount: 8,
		CategoryDistribution: map[string]int{
			"approval": 8,
			"domain":   6,
			"routing":  4,
			"workflow": 3,
			"tooling":  2,
		},
		CategoryPercentages: map[string]float64{
			"approval": 34.78,
			"domain":   26.09,
			"routing":  17.39,
			"workflow": 13.04,
			"tooling":  8.70,
		},
		RecurringPreferences: []RecurringPreference{
			{Category: "domain", Pattern: "Use pytest not unittest", Frequency: 3, SessionIDs: []string{"s1", "s2", "s3"}},
			{Category: "workflow", Pattern: "Run tests after edit", Frequency: 2, SessionIDs: []string{"s1", "s2"}},
			{Category: "tooling", Pattern: "Prefer Edit over sed", Frequency: 2, SessionIDs: []string{"s2", "s4"}},
		},
		DriftAlerts: []PreferenceDriftAlert{
			{Type: "new", Category: "workflow", NewPattern: "Always run make test-ecosystem", Message: "New workflow preference"},
			{Type: "dropped", Category: "routing", OldPattern: "use haiku for search", Message: "Dropped routing preference"},
		},
	}

	markdown := FormatWeeklyIntentSummary(summary)

	// Check required sections
	requiredSections := []string{
		"## User Intents This Week",
		"**Total Captured:** 23 intents across 8 sessions",
		"**By Category:**",
		"approval: 8 (35%)", // Rounded
		"domain: 6 (26%)",
		"**Recurring Preferences:**",
		"\"Use pytest not unittest\"",
		"**Preference Changes:**",
		"🆕 New workflow preference",
		"❌ Dropped routing preference",
	}

	for _, section := range requiredSections {
		if !strings.Contains(markdown, section) {
			t.Errorf("Missing required section: %s\nFull output:\n%s", section, markdown)
		}
	}
}

func TestFormatWeeklyIntentSummary_Empty(t *testing.T) {
	summary := WeeklyIntentSummary{
		WeekStart:            time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		WeekEnd:              time.Date(2026, 1, 22, 0, 0, 0, 0, time.UTC),
		TotalIntents:         0,
		SessionCount:         0,
		CategoryDistribution: map[string]int{},
		CategoryPercentages:  map[string]float64{},
	}

	markdown := FormatWeeklyIntentSummary(summary)

	// Should still have header
	if !strings.Contains(markdown, "## User Intents This Week") {
		t.Error("Missing header section")
	}

	// Should show zero counts
	if !strings.Contains(markdown, "0 intents across 0 sessions") {
		t.Error("Missing zero counts message")
	}

	// Should NOT have categories or preferences sections
	if strings.Contains(markdown, "**By Category:**") {
		t.Error("Should not show category section when empty")
	}

	if strings.Contains(markdown, "**Recurring Preferences:**") {
		t.Error("Should not show preferences section when empty")
	}
}

func TestFormatWeeklyIntentSummary_NoDrift(t *testing.T) {
	summary := WeeklyIntentSummary{
		WeekStart:    time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		WeekEnd:      time.Date(2026, 1, 22, 0, 0, 0, 0, time.UTC),
		TotalIntents: 5,
		SessionCount: 2,
		CategoryDistribution: map[string]int{
			"test": 5,
		},
		CategoryPercentages: map[string]float64{
			"test": 100.0,
		},
		RecurringPreferences: []RecurringPreference{
			{Category: "test", Pattern: "Test pattern", Frequency: 2, SessionIDs: []string{"s1", "s2"}},
		},
		DriftAlerts: []PreferenceDriftAlert{}, // No drift
	}

	markdown := FormatWeeklyIntentSummary(summary)

	// Should NOT have drift section
	if strings.Contains(markdown, "**Preference Changes:**") {
		t.Error("Should not show drift section when no alerts")
	}
}

func TestFormatWeeklyIntentSummary_LimitTop5Preferences(t *testing.T) {
	// Create 10 recurring preferences
	preferences := make([]RecurringPreference, 10)
	for i := 0; i < 10; i++ {
		preferences[i] = RecurringPreference{
			Category:   "test",
			Pattern:    "Pattern " + string(rune('A'+i)),
			Frequency:  10 - i, // Descending frequency
			SessionIDs: []string{"s1"},
		}
	}

	summary := WeeklyIntentSummary{
		WeekStart:            time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		WeekEnd:              time.Date(2026, 1, 22, 0, 0, 0, 0, time.UTC),
		TotalIntents:         50,
		SessionCount:         5,
		CategoryDistribution: map[string]int{"test": 50},
		CategoryPercentages:  map[string]float64{"test": 100.0},
		RecurringPreferences: preferences,
	}

	markdown := FormatWeeklyIntentSummary(summary)

	// Count how many patterns appear (should be max 5)
	patternCount := 0
	for i := 0; i < 10; i++ {
		pattern := "Pattern " + string(rune('A'+i))
		if strings.Contains(markdown, pattern) {
			patternCount++
		}
	}

	if patternCount > 5 {
		t.Errorf("Expected max 5 patterns shown, found %d", patternCount)
	}

	// First 5 should be present
	for i := 0; i < 5; i++ {
		pattern := "Pattern " + string(rune('A'+i))
		if !strings.Contains(markdown, pattern) {
			t.Errorf("Expected pattern '%s' to be in top 5", pattern)
		}
	}
}

func TestFormatWeeklyIntentSummary_HonorRateDisplay(t *testing.T) {
	summary := WeeklyIntentSummary{
		WeekStart:    time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		WeekEnd:      time.Date(2026, 1, 22, 0, 0, 0, 0, time.UTC),
		TotalIntents: 10,
		SessionCount: 3,
		CategoryDistribution: map[string]int{
			"routing": 5,
			"tooling": 3,
			"workflow": 2,
		},
		CategoryPercentages: map[string]float64{
			"routing": 50.0,
			"tooling": 30.0,
			"workflow": 20.0,
		},
		TotalAnalyzed: 10,
		TotalHonored:  8,
		HonorRatePercent: 80.0,
		HonorRateByCategory: map[string]float64{
			"routing": 100.0,
			"tooling": 66.7,
			"workflow": 50.0,
		},
	}

	markdown := FormatWeeklyIntentSummary(summary)

	// Check honor rate section
	if !strings.Contains(markdown, "**Honor Rate:**") {
		t.Error("Missing honor rate section")
	}

	if !strings.Contains(markdown, "Overall: 80% (8/10)") {
		t.Error("Missing overall honor rate")
	}

	// Check per-category honor rates
	if !strings.Contains(markdown, "routing: 100%") {
		t.Error("Missing routing honor rate")
	}
	if !strings.Contains(markdown, "tooling: 67%") {
		t.Error("Missing tooling honor rate")
	}
	if !strings.Contains(markdown, "workflow: 50% ⚠️") {
		t.Error("Missing workflow honor rate with warning")
	}
}

func TestFormatWeeklyIntentSummary_LowHonorRateAlert(t *testing.T) {
	summary := WeeklyIntentSummary{
		WeekStart:        time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		WeekEnd:          time.Date(2026, 1, 22, 0, 0, 0, 0, time.UTC),
		TotalIntents:     10,
		SessionCount:     2,
		TotalAnalyzed:    10,
		TotalHonored:     5,
		HonorRatePercent: 50.0,
		CategoryDistribution: map[string]int{"test": 10},
		CategoryPercentages:  map[string]float64{"test": 100.0},
	}

	markdown := FormatWeeklyIntentSummary(summary)

	// Check for low honor rate alert
	if !strings.Contains(markdown, "**Low Honor Rate Alert:**") {
		t.Error("Missing low honor rate alert for 50% rate")
	}
	if !strings.Contains(markdown, "below 70%") {
		t.Error("Missing threshold mention in alert")
	}
}

func TestFormatWeeklyIntentSummary_NoHonorRateWhenZeroAnalyzed(t *testing.T) {
	summary := WeeklyIntentSummary{
		WeekStart:            time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		WeekEnd:              time.Date(2026, 1, 22, 0, 0, 0, 0, time.UTC),
		TotalIntents:         10,
		SessionCount:         2,
		TotalAnalyzed:        0, // No analysis done
		CategoryDistribution: map[string]int{"general": 10},
		CategoryPercentages:  map[string]float64{"general": 100.0},
	}

	markdown := FormatWeeklyIntentSummary(summary)

	// Should NOT show honor rate section
	if strings.Contains(markdown, "**Honor Rate:**") {
		t.Error("Should not show honor rate section when TotalAnalyzed is 0")
	}
}
