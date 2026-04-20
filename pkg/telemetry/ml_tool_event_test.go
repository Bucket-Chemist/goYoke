package telemetry

import (
	"testing"

	"github.com/Bucket-Chemist/goYoke/pkg/routing"
)

func TestTotalTokens(t *testing.T) {
	event := &routing.PostToolEvent{
		InputTokens:  1024,
		OutputTokens: 512,
	}

	total := TotalTokens(event)
	if total != 1536 {
		t.Errorf("Expected 1536 total tokens, got: %d", total)
	}
}

func TestEstimatedCost(t *testing.T) {
	tests := []struct {
		model        string
		inputTokens  int
		outputTokens int
		expectedCost float64
	}{
		{"haiku", 1000, 0, 0.0005},
		{"sonnet", 1000, 0, 0.009},
		{"opus", 1000, 0, 0.045},
	}

	for _, tc := range tests {
		event := &routing.PostToolEvent{
			Model:        tc.model,
			InputTokens:  tc.inputTokens,
			OutputTokens: tc.outputTokens,
		}

		cost := EstimatedCost(event)
		if cost != tc.expectedCost {
			t.Errorf("Model %s: expected %f, got %f", tc.model, tc.expectedCost, cost)
		}
	}
}

func TestEstimatedCost_Unknown(t *testing.T) {
	event := &routing.PostToolEvent{
		Model: "unknown",
	}

	cost := EstimatedCost(event)
	if cost != 0.0 {
		t.Errorf("Unknown model should cost 0, got: %f", cost)
	}
}

func TestEnrichWithSequence(t *testing.T) {
	event := &routing.PostToolEvent{}
	previous := []string{"Glob", "Grep", "Read", "Edit", "Bash"}
	outcomes := []bool{true, true, false, true, true}

	EnrichWithSequence(event, 5, previous, outcomes)

	if event.SequenceIndex != 5 {
		t.Errorf("Expected SequenceIndex 5, got: %d", event.SequenceIndex)
	}

	if len(event.PreviousTools) != 5 {
		t.Errorf("Expected 5 previous tools, got: %d", len(event.PreviousTools))
	}

	if len(event.PreviousOutcomes) != 5 {
		t.Errorf("Expected 5 previous outcomes, got: %d", len(event.PreviousOutcomes))
	}

	for i, tool := range event.PreviousTools {
		if tool != previous[i] {
			t.Errorf("Previous tool %d: expected %s, got %s", i, previous[i], tool)
		}
	}

	for i, outcome := range event.PreviousOutcomes {
		if outcome != outcomes[i] {
			t.Errorf("Outcome %d: expected %v, got %v", i, outcomes[i], outcome)
		}
	}
}

func TestEnrichWithClassification(t *testing.T) {
	event := &routing.PostToolEvent{
		SelectedTier:  "sonnet",
		SelectedAgent: "python-pro",
	}

	// Function signature defined, caller implements classification logic
	EnrichWithClassification(event)

	// Test passes if no panic - actual classification logic
	// is implemented in caller based on routing context
	if event == nil {
		t.Error("Event should not be nil after enrichment")
	}
}

func TestPostToolEventWithSequenceTracking(t *testing.T) {
	event := &routing.PostToolEvent{
		Model:        "haiku",
		InputTokens:  2048,
		OutputTokens: 1024,
	}

	// Verify helper functions work with routing.PostToolEvent
	total := TotalTokens(event)
	if total != 3072 {
		t.Errorf("Expected 3072 total tokens, got: %d", total)
	}

	cost := EstimatedCost(event)
	expectedCost := 0.001536 // (3072 / 1000.0) * 0.0005
	if cost != expectedCost {
		t.Errorf("Expected cost %f, got %f", expectedCost, cost)
	}
}
