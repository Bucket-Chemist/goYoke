package session

import (
	"testing"
)

// TestClassifyIntent_Routing verifies routing category classification
func TestClassifyIntent_Routing(t *testing.T) {
	cases := []struct {
		question string
		response string
		expected IntentCategory
	}{
		{"Which model?", "Use sonnet for this", CategoryRouting},
		{"How to proceed?", "Delegate to haiku", CategoryRouting},
		{"", "use opus", CategoryRouting},
		{"What tier?", "This needs agent coordination", CategoryRouting},
		{"", "tier 1.5", CategoryRouting},
		{"Model selection?", "Sonnet is best here", CategoryRouting},
	}

	for _, tc := range cases {
		result := ClassifyIntent(tc.question, tc.response)
		if result != tc.expected {
			t.Errorf("ClassifyIntent(%q, %q) = %s, want %s",
				tc.question, tc.response, result, tc.expected)
		}
	}
}

// TestClassifyIntent_Tooling verifies tooling category classification
func TestClassifyIntent_Tooling(t *testing.T) {
	cases := []struct {
		question string
		response string
		expected IntentCategory
	}{
		{"Which tool?", "Use Edit not sed", CategoryTooling},
		{"How to search?", "Prefer grep over find", CategoryTooling},
		{"", "Don't use bash for this", CategoryTooling},
		{"Tool selection?", "Use Read first", CategoryTooling},
		{"", "tool Glob", CategoryTooling},
	}

	for _, tc := range cases {
		result := ClassifyIntent(tc.question, tc.response)
		if result != tc.expected {
			t.Errorf("ClassifyIntent(%q, %q) = %s, want %s",
				tc.question, tc.response, result, tc.expected)
		}
	}
}

// TestClassifyIntent_Style verifies style category classification
func TestClassifyIntent_Style(t *testing.T) {
	cases := []struct {
		question string
		response string
		expected IntentCategory
	}{
		{"How should I format?", "Be more concise", CategoryStyle},
		{"Output preference?", "Make it verbose", CategoryStyle},
		{"", "shorter responses", CategoryStyle},
		{"Response style?", "More detailed output", CategoryStyle},
		{"", "format as brief", CategoryStyle},
	}

	for _, tc := range cases {
		result := ClassifyIntent(tc.question, tc.response)
		if result != tc.expected {
			t.Errorf("ClassifyIntent(%q, %q) = %s, want %s",
				tc.question, tc.response, result, tc.expected)
		}
	}
}

// TestClassifyIntent_Workflow verifies workflow category classification
func TestClassifyIntent_Workflow(t *testing.T) {
	cases := []struct {
		question string
		response string
		expected IntentCategory
	}{
		{"What sequence?", "Always check first", CategoryWorkflow},
		{"Order of operations?", "Never skip validation", CategoryWorkflow},
		{"", "after commit do push", CategoryWorkflow},
		{"Workflow?", "before deploy do test", CategoryWorkflow},
		{"", "first build then test", CategoryWorkflow},
	}

	for _, tc := range cases {
		result := ClassifyIntent(tc.question, tc.response)
		if result != tc.expected {
			t.Errorf("ClassifyIntent(%q, %q) = %s, want %s",
				tc.question, tc.response, result, tc.expected)
		}
	}
}

// TestClassifyIntent_Domain verifies domain category classification
func TestClassifyIntent_Domain(t *testing.T) {
	cases := []struct {
		question string
		response string
		expected IntentCategory
	}{
		{"What framework?", "We use React", CategoryDomain},
		{"Project conventions?", "Our project uses Go", CategoryDomain},
		{"", "this project convention", CategoryDomain},
		{"Library choice?", "The codebase uses Cobra", CategoryDomain},
		{"", "our framework is FastAPI", CategoryDomain},
	}

	for _, tc := range cases {
		result := ClassifyIntent(tc.question, tc.response)
		if result != tc.expected {
			t.Errorf("ClassifyIntent(%q, %q) = %s, want %s",
				tc.question, tc.response, result, tc.expected)
		}
	}
}

// TestClassifyIntent_Correction verifies correction priority (checked before other categories)
func TestClassifyIntent_Correction(t *testing.T) {
	cases := []struct {
		question string
		response string
		expected IntentCategory
	}{
		{"Which file?", "No, I meant the other one", CategoryCorrection},
		{"Confirm?", "Actually, not that", CategoryCorrection},
		{"", "wrong, use this instead", CategoryCorrection},
		{"", "incorrect, I meant X", CategoryCorrection},
		{"", "No you meant the file", CategoryCorrection},
	}

	for _, tc := range cases {
		result := ClassifyIntent(tc.question, tc.response)
		if result != tc.expected {
			t.Errorf("ClassifyIntent(%q, %q) = %s, want %s",
				tc.question, tc.response, result, tc.expected)
		}
	}
}

// TestClassifyIntent_Approval verifies approval category
func TestClassifyIntent_Approval(t *testing.T) {
	cases := []string{
		"yes",
		"Yes",
		"proceed",
		"Go ahead",
		"looks good",
		"do it",
		"confirmed",
		"approve",
		"ok",
		"okay",
		"sure",
		"yep",
		"yeah",
	}

	for _, response := range cases {
		result := ClassifyIntent("Ready?", response)
		if result != CategoryApproval {
			t.Errorf("ClassifyIntent(_, %q) = %s, want approval", response, result)
		}
	}
}

// TestClassifyIntent_Rejection verifies rejection category
func TestClassifyIntent_Rejection(t *testing.T) {
	cases := []string{
		"stop",
		"cancel",
		"abort",
		"wrong approach",
		"undo",
		"don't",
		"halt",
		"no don't",
		"wait",
	}

	for _, response := range cases {
		result := ClassifyIntent("Proceed?", response)
		if result != CategoryRejection {
			t.Errorf("ClassifyIntent(_, %q) = %s, want rejection", response, result)
		}
	}
}

// TestClassifyIntent_PriorityOrder verifies correction takes priority over other categories
func TestClassifyIntent_PriorityOrder(t *testing.T) {
	// Response contains both "tool" (tooling) and "wrong" (correction)
	// Correction should win due to priority order
	result := ClassifyIntent("Which tool?", "Wrong, use the other tool")
	if result != CategoryCorrection {
		t.Errorf("Expected correction priority, got %s", result)
	}

	// Response contains both "agent" (routing) and "no I meant" (correction)
	// Correction should win
	result = ClassifyIntent("Which agent?", "No I meant the routing agent")
	if result != CategoryCorrection {
		t.Errorf("Expected correction priority, got %s", result)
	}
}

// TestClassifyIntent_General verifies fallback to general category
func TestClassifyIntent_General(t *testing.T) {
	cases := []struct {
		question string
		response string
	}{
		{"How are you?", "I'm fine"},
		{"What time?", "3pm"},
		{"", "random text here"},
		{"Question?", "Answer without keywords"},
	}

	for _, tc := range cases {
		result := ClassifyIntent(tc.question, tc.response)
		if result != CategoryGeneral {
			t.Errorf("ClassifyIntent(%q, %q) = %s, want general",
				tc.question, tc.response, result)
		}
	}
}

// TestClassifyIntent_CaseInsensitive verifies case-insensitive matching
func TestClassifyIntent_CaseInsensitive(t *testing.T) {
	cases := []struct {
		response string
		expected IntentCategory
	}{
		{"USE SONNET", CategoryRouting},
		{"prefer EDIT", CategoryTooling},
		{"ALWAYS check", CategoryWorkflow},
		{"YES", CategoryApproval},
		{"STOP", CategoryRejection},
	}

	for _, tc := range cases {
		result := ClassifyIntent("", tc.response)
		if result != tc.expected {
			t.Errorf("ClassifyIntent(_, %q) = %s, want %s",
				tc.response, result, tc.expected)
		}
	}
}

// TestClassifyIntent_Deterministic verifies same input produces same output
func TestClassifyIntent_Deterministic(t *testing.T) {
	question := "Which model should I use?"
	response := "Use sonnet for this task"

	// Run classification 10 times
	first := ClassifyIntent(question, response)
	for i := 0; i < 9; i++ {
		result := ClassifyIntent(question, response)
		if result != first {
			t.Errorf("Classification not deterministic: iteration %d got %s, expected %s",
				i, result, first)
		}
	}
}

// TestClassifyIntent_EmptyInputs verifies handling of empty inputs
func TestClassifyIntent_EmptyInputs(t *testing.T) {
	// Both empty
	result := ClassifyIntent("", "")
	if result != CategoryGeneral {
		t.Errorf("Empty inputs should return general, got %s", result)
	}

	// Question only
	result = ClassifyIntent("use sonnet", "")
	if result != CategoryRouting {
		t.Errorf("Expected routing from question only, got %s", result)
	}

	// Response only
	result = ClassifyIntent("", "prefer edit")
	if result != CategoryTooling {
		t.Errorf("Expected tooling from response only, got %s", result)
	}
}
