package session

import (
	"strings"
)

// IntentOutcome represents the analysis result for a single intent
type IntentOutcome struct {
	IntentIndex int    // Position in intents list
	Honored     bool   // Whether the intent was followed
	Note        string // Explanation
	Confidence  string // "high", "medium", "low"
}

// SessionActions represents what happened in the session after intents
type SessionActions struct {
	ToolsUsed      []string // Tool names that were called
	ModelsUsed     []string // Model tiers used (from Task calls)
	FilesEdited    []string // Files that were modified
	CommandsRun    []string // Bash commands executed
	ActionsStopped bool     // Whether an action was halted/cancelled
}

// AnalyzeIntentOutcomes correlates intents with session actions
func AnalyzeIntentOutcomes(intents []UserIntent, actions SessionActions) []IntentOutcome {
	var outcomes []IntentOutcome

	for i, intent := range intents {
		outcome := analyzeOneIntent(intent, actions)
		outcome.IntentIndex = i
		outcomes = append(outcomes, outcome)
	}

	return outcomes
}

func analyzeOneIntent(intent UserIntent, actions SessionActions) IntentOutcome {
	switch intent.Category {
	case "routing":
		return analyzeRoutingIntent(intent, actions)
	case "tooling":
		return analyzeToolingIntent(intent, actions)
	case "workflow":
		return analyzeWorkflowIntent(intent, actions)
	case "approval":
		return analyzeApprovalIntent(intent, actions)
	case "rejection":
		return analyzeRejectionIntent(intent, actions)
	case "correction":
		return analyzeCorrectionIntent(intent, actions)
	default:
		// General/domain/style - can't reliably determine
		return IntentOutcome{
			Honored:    true, // Assume honored if can't determine
			Note:       "Category not trackable for outcome",
			Confidence: "low",
		}
	}
}

func analyzeRoutingIntent(intent UserIntent, actions SessionActions) IntentOutcome {
	response := strings.ToLower(intent.Response)

	// Check for model preferences
	if strings.Contains(response, "sonnet") {
		if containsIgnoreCase(actions.ModelsUsed, "sonnet") {
			return IntentOutcome{Honored: true, Note: "Sonnet was used", Confidence: "high"}
		}
		return IntentOutcome{Honored: false, Note: "Sonnet was requested but not used", Confidence: "high"}
	}
	if strings.Contains(response, "haiku") {
		if containsIgnoreCase(actions.ModelsUsed, "haiku") {
			return IntentOutcome{Honored: true, Note: "Haiku was used", Confidence: "high"}
		}
		return IntentOutcome{Honored: false, Note: "Haiku was requested but not used", Confidence: "high"}
	}
	if strings.Contains(response, "opus") {
		if containsIgnoreCase(actions.ModelsUsed, "opus") {
			return IntentOutcome{Honored: true, Note: "Opus was used", Confidence: "high"}
		}
		return IntentOutcome{Honored: false, Note: "Opus was requested but not used", Confidence: "high"}
	}

	return IntentOutcome{Honored: true, Note: "Routing intent, no specific model mentioned", Confidence: "low"}
}

func analyzeToolingIntent(intent UserIntent, actions SessionActions) IntentOutcome {
	response := strings.ToLower(intent.Response)

	// Check for "don't use X" patterns FIRST (before "use X" to avoid "use" substring match)
	if strings.Contains(response, "don't use sed") || strings.Contains(response, "not sed") {
		// Check if sed appears in any command
		for _, cmd := range actions.CommandsRun {
			if strings.Contains(strings.ToLower(cmd), "sed") {
				return IntentOutcome{Honored: false, Note: "sed was used despite preference against", Confidence: "high"}
			}
		}
		return IntentOutcome{Honored: true, Note: "sed was avoided as requested", Confidence: "high"}
	}

	// Check for "use X" patterns
	if strings.Contains(response, "use edit") || strings.Contains(response, "prefer edit") {
		if containsIgnoreCase(actions.ToolsUsed, "Edit") {
			return IntentOutcome{Honored: true, Note: "Edit tool was used", Confidence: "high"}
		}
		return IntentOutcome{Honored: false, Note: "Edit was requested but not used", Confidence: "medium"}
	}

	return IntentOutcome{Honored: true, Note: "Tooling intent, pattern not matched", Confidence: "low"}
}

func analyzeWorkflowIntent(intent UserIntent, actions SessionActions) IntentOutcome {
	response := strings.ToLower(intent.Response)

	// Check for "run tests after" pattern
	if strings.Contains(response, "run tests") || strings.Contains(response, "test after") {
		for _, cmd := range actions.CommandsRun {
			if strings.Contains(cmd, "test") {
				return IntentOutcome{Honored: true, Note: "Tests were run", Confidence: "medium"}
			}
		}
		return IntentOutcome{Honored: false, Note: "Tests were requested but not run", Confidence: "medium"}
	}

	return IntentOutcome{Honored: true, Note: "Workflow intent, pattern not matched", Confidence: "low"}
}

func analyzeApprovalIntent(intent UserIntent, actions SessionActions) IntentOutcome {
	// If user approved, check that something happened after
	if len(actions.ToolsUsed) > 0 || len(actions.CommandsRun) > 0 || len(actions.FilesEdited) > 0 {
		return IntentOutcome{Honored: true, Note: "Action proceeded after approval", Confidence: "high"}
	}
	return IntentOutcome{Honored: false, Note: "Approved but no action taken", Confidence: "medium"}
}

func analyzeRejectionIntent(intent UserIntent, actions SessionActions) IntentOutcome {
	if actions.ActionsStopped {
		return IntentOutcome{Honored: true, Note: "Action was stopped as requested", Confidence: "high"}
	}
	// If very few actions after rejection, probably honored
	return IntentOutcome{Honored: true, Note: "Rejection noted, limited subsequent actions", Confidence: "medium"}
}

func analyzeCorrectionIntent(intent UserIntent, actions SessionActions) IntentOutcome {
	// Corrections are hard to verify without deep analysis
	return IntentOutcome{Honored: true, Note: "Correction acknowledged", Confidence: "low"}
}

func containsIgnoreCase(slice []string, target string) bool {
	target = strings.ToLower(target)
	for _, s := range slice {
		if strings.ToLower(s) == target {
			return true
		}
	}
	return false
}
