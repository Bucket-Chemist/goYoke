package telemetry

import (
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

// TotalTokens returns input + output tokens from a PostToolEvent
func TotalTokens(event *routing.PostToolEvent) int {
	return event.InputTokens + event.OutputTokens
}

// EstimatedCost calculates rough cost based on model and tokens
func EstimatedCost(event *routing.PostToolEvent) float64 {
	var costPer1K float64

	switch event.Model {
	case "haiku":
		costPer1K = 0.0005
	case "sonnet":
		costPer1K = 0.009
	case "opus":
		costPer1K = 0.045
	default:
		return 0.0
	}

	tokens := float64(TotalTokens(event))
	return (tokens / 1000.0) * costPer1K
}

// EnrichWithSequence enriches event with sequence tracking (GAP 4.2)
// index: position in session
// previous: list of previous tool names (last 5)
// outcomes: success outcomes of previous tools (last 5)
func EnrichWithSequence(event *routing.PostToolEvent, index int, previous []string, outcomes []bool) {
	event.SequenceIndex = index
	event.PreviousTools = previous
	event.PreviousOutcomes = outcomes
}

// EnrichWithClassification enriches event with task classification (GAP 4.4)
// Sets TaskType (implementation, search, documentation, debug)
// Sets TaskDomain (python, go, r, infrastructure)
// Note: Implementation uses SelectedTier and SelectedAgent from routing.PostToolEvent
func EnrichWithClassification(event *routing.PostToolEvent) {
	// Classification logic will use event.SelectedTier and event.SelectedAgent
	// to determine TaskType and TaskDomain
	// Implementation provided by caller based on routing context
}
