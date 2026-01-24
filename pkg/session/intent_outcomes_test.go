package session

import (
	"strings"
	"testing"
)

func TestAnalyzeRoutingIntent_Sonnet(t *testing.T) {
	intent := UserIntent{
		Category: "routing",
		Response: "Use Sonnet for this task",
	}

	// Test: Sonnet was used
	actions := SessionActions{
		ModelsUsed: []string{"sonnet", "haiku"},
	}
	outcome := analyzeRoutingIntent(intent, actions)
	if !outcome.Honored {
		t.Errorf("Expected honored=true when sonnet was used, got false")
	}
	if outcome.Confidence != "high" {
		t.Errorf("Expected confidence=high, got %s", outcome.Confidence)
	}

	// Test: Sonnet was NOT used
	actions = SessionActions{
		ModelsUsed: []string{"haiku"},
	}
	outcome = analyzeRoutingIntent(intent, actions)
	if outcome.Honored {
		t.Errorf("Expected honored=false when sonnet was not used, got true")
	}
	if outcome.Confidence != "high" {
		t.Errorf("Expected confidence=high, got %s", outcome.Confidence)
	}
}

func TestAnalyzeRoutingIntent_NoModelMentioned(t *testing.T) {
	intent := UserIntent{
		Category: "routing",
		Response: "Use the orchestrator agent",
	}

	actions := SessionActions{
		ModelsUsed: []string{"sonnet"},
	}

	outcome := analyzeRoutingIntent(intent, actions)
	if !outcome.Honored {
		t.Errorf("Expected honored=true for non-specific routing, got false")
	}
	if outcome.Confidence != "low" {
		t.Errorf("Expected confidence=low, got %s", outcome.Confidence)
	}
}

func TestAnalyzeRoutingIntent_Opus(t *testing.T) {
	intent := UserIntent{
		Category: "routing",
		Response: "Use Opus for this complex task",
	}

	// Test: Opus was used
	actions := SessionActions{
		ModelsUsed: []string{"opus", "sonnet"},
	}
	outcome := analyzeRoutingIntent(intent, actions)
	if !outcome.Honored {
		t.Errorf("Expected honored=true when opus was used, got false")
	}
	if outcome.Confidence != "high" {
		t.Errorf("Expected confidence=high, got %s", outcome.Confidence)
	}

	// Test: Opus was NOT used
	actions = SessionActions{
		ModelsUsed: []string{"sonnet", "haiku"},
	}
	outcome = analyzeRoutingIntent(intent, actions)
	if outcome.Honored {
		t.Errorf("Expected honored=false when opus was not used, got true")
	}
	if outcome.Confidence != "high" {
		t.Errorf("Expected confidence=high, got %s", outcome.Confidence)
	}
}

func TestAnalyzeRoutingIntent_Haiku(t *testing.T) {
	intent := UserIntent{
		Category: "routing",
		Response: "Use haiku for simple search",
	}

	// Test: Haiku was used
	actions := SessionActions{
		ModelsUsed: []string{"haiku"},
	}
	outcome := analyzeRoutingIntent(intent, actions)
	if !outcome.Honored {
		t.Errorf("Expected honored=true when haiku was used, got false")
	}

	// Test: Haiku was NOT used
	actions = SessionActions{
		ModelsUsed: []string{"sonnet"},
	}
	outcome = analyzeRoutingIntent(intent, actions)
	if outcome.Honored {
		t.Errorf("Expected honored=false when haiku was not used, got true")
	}
}

func TestAnalyzeToolingIntent_UseEdit(t *testing.T) {
	intent := UserIntent{
		Category: "tooling",
		Response: "Prefer Edit tool over Write",
	}

	// Test: Edit was used
	actions := SessionActions{
		ToolsUsed: []string{"Edit", "Read"},
	}
	outcome := analyzeToolingIntent(intent, actions)
	if !outcome.Honored {
		t.Errorf("Expected honored=true when Edit was used, got false")
	}
	if outcome.Confidence != "high" {
		t.Errorf("Expected confidence=high, got %s", outcome.Confidence)
	}

	// Test: Edit was NOT used
	actions = SessionActions{
		ToolsUsed: []string{"Write", "Read"},
	}
	outcome = analyzeToolingIntent(intent, actions)
	if outcome.Honored {
		t.Errorf("Expected honored=false when Edit was not used, got true")
	}
}

func TestAnalyzeToolingIntent_DontUseSed(t *testing.T) {
	intent := UserIntent{
		Category: "tooling",
		Response: "Don't use sed, prefer Edit",
	}

	// Test: sed was avoided
	actions := SessionActions{
		CommandsRun: []string{"go test", "git status"},
	}
	outcome := analyzeToolingIntent(intent, actions)
	if !outcome.Honored {
		t.Errorf("Expected honored=true when sed was avoided, got false")
	}

	// Test: sed was used despite preference
	actions = SessionActions{
		CommandsRun: []string{"sed -i 's/foo/bar/' file.txt"},
	}
	outcome = analyzeToolingIntent(intent, actions)
	if outcome.Honored {
		t.Errorf("Expected honored=false when sed was used, got true")
	}
}

func TestAnalyzeWorkflowIntent_RunTests(t *testing.T) {
	intent := UserIntent{
		Category: "workflow",
		Response: "Run tests after making changes",
	}

	// Test: Tests were run
	actions := SessionActions{
		CommandsRun: []string{"go build", "go test ./...", "git add ."},
	}
	outcome := analyzeWorkflowIntent(intent, actions)
	if !outcome.Honored {
		t.Errorf("Expected honored=true when tests were run, got false")
	}

	// Test: Tests were NOT run
	actions = SessionActions{
		CommandsRun: []string{"go build", "git add ."},
	}
	outcome = analyzeWorkflowIntent(intent, actions)
	if outcome.Honored {
		t.Errorf("Expected honored=false when tests were not run, got true")
	}
}

func TestAnalyzeApprovalIntent(t *testing.T) {
	intent := UserIntent{
		Category: "approval",
		Response: "Yes, proceed with the changes",
	}

	// Test: Action proceeded
	actions := SessionActions{
		ToolsUsed:   []string{"Edit"},
		FilesEdited: []string{"main.go"},
	}
	outcome := analyzeApprovalIntent(intent, actions)
	if !outcome.Honored {
		t.Errorf("Expected honored=true when action proceeded, got false")
	}

	// Test: No action taken after approval
	actions = SessionActions{
		ToolsUsed:   []string{},
		FilesEdited: []string{},
		CommandsRun: []string{},
	}
	outcome = analyzeApprovalIntent(intent, actions)
	if outcome.Honored {
		t.Errorf("Expected honored=false when no action taken, got true")
	}
}

func TestAnalyzeRejectionIntent(t *testing.T) {
	intent := UserIntent{
		Category: "rejection",
		Response: "No, don't do that",
	}

	// Test: Action was stopped
	actions := SessionActions{
		ActionsStopped: true,
	}
	outcome := analyzeRejectionIntent(intent, actions)
	if !outcome.Honored {
		t.Errorf("Expected honored=true when action was stopped, got false")
	}
	if outcome.Confidence != "high" {
		t.Errorf("Expected confidence=high, got %s", outcome.Confidence)
	}

	// Test: Limited actions (still honored)
	actions = SessionActions{
		ActionsStopped: false,
	}
	outcome = analyzeRejectionIntent(intent, actions)
	if !outcome.Honored {
		t.Errorf("Expected honored=true for rejection with limited actions, got false")
	}
	if outcome.Confidence != "medium" {
		t.Errorf("Expected confidence=medium, got %s", outcome.Confidence)
	}
}

func TestAnalyzeCorrectionIntent(t *testing.T) {
	intent := UserIntent{
		Category: "correction",
		Response: "Actually, use this pattern instead",
	}

	actions := SessionActions{}

	outcome := analyzeCorrectionIntent(intent, actions)
	if !outcome.Honored {
		t.Errorf("Expected honored=true for correction (default), got false")
	}
	if outcome.Confidence != "low" {
		t.Errorf("Expected confidence=low, got %s", outcome.Confidence)
	}
}

func TestAnalyzeOneIntent_UnknownCategory(t *testing.T) {
	intent := UserIntent{
		Category: "general",
		Response: "This is a general preference",
	}

	actions := SessionActions{}

	outcome := analyzeOneIntent(intent, actions)
	if !outcome.Honored {
		t.Errorf("Expected honored=true for unknown category, got false")
	}
	if outcome.Confidence != "low" {
		t.Errorf("Expected confidence=low, got %s", outcome.Confidence)
	}
	if outcome.Note != "Category not trackable for outcome" {
		t.Errorf("Expected specific note for unknown category, got: %s", outcome.Note)
	}
}

func TestAnalyzeIntentOutcomes_MultipleIntents(t *testing.T) {
	intents := []UserIntent{
		{Category: "routing", Response: "Use Sonnet"},
		{Category: "tooling", Response: "Use Edit tool"},
		{Category: "approval", Response: "Yes, proceed"},
	}

	actions := SessionActions{
		ModelsUsed: []string{"sonnet"},
		ToolsUsed:  []string{"Edit", "Write"},
		FilesEdited: []string{"test.go"},
	}

	outcomes := AnalyzeIntentOutcomes(intents, actions)

	if len(outcomes) != len(intents) {
		t.Errorf("Expected %d outcomes, got %d", len(intents), len(outcomes))
	}

	// Verify all were analyzed
	for i, outcome := range outcomes {
		if outcome.IntentIndex != i {
			t.Errorf("Expected outcome index %d, got %d", i, outcome.IntentIndex)
		}
	}

	// Verify all were honored
	for i, outcome := range outcomes {
		if !outcome.Honored {
			t.Errorf("Intent %d should be honored but wasn't: %s", i, outcome.Note)
		}
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		slice  []string
		target string
		want   bool
	}{
		{[]string{"Sonnet", "Haiku"}, "sonnet", true},
		{[]string{"Sonnet", "Haiku"}, "HAIKU", true},
		{[]string{"Sonnet", "Haiku"}, "opus", false},
		{[]string{}, "sonnet", false},
		{[]string{"Edit", "Write"}, "edit", true},
	}

	for _, tt := range tests {
		got := containsIgnoreCase(tt.slice, tt.target)
		if got != tt.want {
			t.Errorf("containsIgnoreCase(%v, %q) = %v, want %v",
				tt.slice, tt.target, got, tt.want)
		}
	}
}

func TestAnalyzeIntentOutcomes_EmptyActions(t *testing.T) {
	intents := []UserIntent{
		{Category: "routing", Response: "Use Sonnet"},
		{Category: "tooling", Response: "Use Edit"},
	}

	// Empty actions - all intents should fail or return low confidence
	actions := SessionActions{}

	outcomes := AnalyzeIntentOutcomes(intents, actions)

	if len(outcomes) != 2 {
		t.Errorf("Expected 2 outcomes, got %d", len(outcomes))
	}

	// Routing intent without model use should fail
	if outcomes[0].Honored {
		t.Errorf("Expected routing intent to be not honored with no models used")
	}

	// Tooling intent without tool use should fail
	if outcomes[1].Honored {
		t.Errorf("Expected tooling intent to be not honored with no tools used")
	}
}

func TestAnalyzeIntentOutcomes_IntegrationScenario(t *testing.T) {
	// Realistic scenario: User wants Sonnet, Edit tool, and tests run
	intents := []UserIntent{
		{
			Timestamp: 1234567890,
			Category:  "routing",
			Response:  "Use Sonnet for implementation",
		},
		{
			Timestamp: 1234567891,
			Category:  "tooling",
			Response:  "Use Edit tool, not sed",
		},
		{
			Timestamp: 1234567892,
			Category:  "workflow",
			Response:  "Run tests after changes",
		},
		{
			Timestamp: 1234567893,
			Category:  "approval",
			Response:  "Yes, go ahead",
		},
	}

	// Session honored all requests
	actions := SessionActions{
		ModelsUsed:  []string{"sonnet"},
		ToolsUsed:   []string{"Edit", "Read"},
		CommandsRun: []string{"go test ./...", "git status"},
		FilesEdited: []string{"pkg/session/intent_outcomes.go"},
	}

	outcomes := AnalyzeIntentOutcomes(intents, actions)

	if len(outcomes) != 4 {
		t.Fatalf("Expected 4 outcomes, got %d", len(outcomes))
	}

	// All should be honored
	for i, outcome := range outcomes {
		if !outcome.Honored {
			t.Errorf("Outcome %d should be honored but wasn't: %s", i, outcome.Note)
		}
	}

	// Verify specific outcomes
	if outcomes[0].Confidence != "high" {
		t.Errorf("Routing outcome should have high confidence, got %s", outcomes[0].Confidence)
	}
	if outcomes[1].Confidence != "high" {
		t.Errorf("Tooling outcome should have high confidence, got %s", outcomes[1].Confidence)
	}
	if outcomes[2].Confidence != "medium" {
		t.Errorf("Workflow outcome should have medium confidence, got %s", outcomes[2].Confidence)
	}
	if outcomes[3].Confidence != "high" {
		t.Errorf("Approval outcome should have high confidence, got %s", outcomes[3].Confidence)
	}
}

func TestAnalyzeOneIntent_DomainCategory(t *testing.T) {
	intent := UserIntent{
		Category: "domain",
		Response: "Use pytest for testing",
	}

	actions := SessionActions{}

	outcome := analyzeOneIntent(intent, actions)
	if !outcome.Honored {
		t.Errorf("Expected honored=true for domain category, got false")
	}
	if outcome.Confidence != "low" {
		t.Errorf("Expected confidence=low for domain category, got %s", outcome.Confidence)
	}
	if outcome.Note != "Category not trackable for outcome" {
		t.Errorf("Expected specific note for domain category, got: %s", outcome.Note)
	}
}

func TestAnalyzeOneIntent_StyleCategory(t *testing.T) {
	intent := UserIntent{
		Category: "style",
		Response: "Use camelCase",
	}

	actions := SessionActions{}

	outcome := analyzeOneIntent(intent, actions)
	if !outcome.Honored {
		t.Errorf("Expected honored=true for style category, got false")
	}
	if outcome.Confidence != "low" {
		t.Errorf("Expected confidence=low for style category, got %s", outcome.Confidence)
	}
}

func TestAnalyzeToolingIntent_NoMatchingPattern(t *testing.T) {
	intent := UserIntent{
		Category: "tooling",
		Response: "Some unrecognized tooling preference",
	}

	actions := SessionActions{
		ToolsUsed: []string{"Read"},
	}

	outcome := analyzeToolingIntent(intent, actions)
	if !outcome.Honored {
		t.Errorf("Expected honored=true for unrecognized pattern, got false")
	}
	if outcome.Confidence != "low" {
		t.Errorf("Expected confidence=low for unrecognized pattern, got %s", outcome.Confidence)
	}
	if !strings.Contains(outcome.Note, "pattern not matched") {
		t.Errorf("Expected 'pattern not matched' in note, got: %s", outcome.Note)
	}
}

func TestAnalyzeWorkflowIntent_NoMatchingPattern(t *testing.T) {
	intent := UserIntent{
		Category: "workflow",
		Response: "Some unrecognized workflow preference",
	}

	actions := SessionActions{
		CommandsRun: []string{"git status"},
	}

	outcome := analyzeWorkflowIntent(intent, actions)
	if !outcome.Honored {
		t.Errorf("Expected honored=true for unrecognized pattern, got false")
	}
	if outcome.Confidence != "low" {
		t.Errorf("Expected confidence=low for unrecognized pattern, got %s", outcome.Confidence)
	}
	if !strings.Contains(outcome.Note, "pattern not matched") {
		t.Errorf("Expected 'pattern not matched' in note, got: %s", outcome.Note)
	}
}
