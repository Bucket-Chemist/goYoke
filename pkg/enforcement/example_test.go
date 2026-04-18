package enforcement_test

import (
	"fmt"

	"github.com/Bucket-Chemist/goYoke/pkg/enforcement"
	"github.com/Bucket-Chemist/goYoke/pkg/routing"
)

// ExampleAnalyzeToolEventForDocTheater demonstrates the complete workflow
// of detecting documentation theater in CLAUDE.md edits
func ExampleAnalyzeToolEventForDocTheater() {
	// Simulate a Write operation to CLAUDE.md with theater patterns
	event := &routing.ToolEvent{
		ToolName: "Write",
		ToolInput: map[string]interface{}{
			"file_path": "/home/user/.claude/CLAUDE.md",
			"content": `## Enforcement Rules

You MUST NOT use Task(opus) directly.
This is BLOCKED (by policy).
NEVER use Einstein without permission.
`,
		},
	}

	// Analyze the event for documentation theater
	results := enforcement.AnalyzeToolEventForDocTheater(event)

	if results != nil && len(results) > 0 {
		fmt.Printf("Detected %d theater pattern(s)\n", len(results))

		// Generate warning message
		_ = enforcement.GenerateWarning(results, "CLAUDE.md")
		fmt.Println("Warning generated successfully")

		// Check for critical patterns
		detector := enforcement.NewPatternDetector()
		hasCritical := detector.HasDocumentationTheater(event.ExtractWriteContent())
		fmt.Printf("Has critical patterns: %v\n", hasCritical)
	}

	// Output:
	// Detected 3 theater pattern(s)
	// Warning generated successfully
	// Has critical patterns: true
}

// ExampleAnalyzeToolEventForDocTheater_cleanContent demonstrates
// that legitimate enforcement references are not flagged
func ExampleAnalyzeToolEventForDocTheater_cleanContent() {
	// Legitimate enforcement reference (describes, doesn't command)
	event := &routing.ToolEvent{
		ToolName: "Edit",
		ToolInput: map[string]interface{}{
			"file_path": "/home/user/.claude/CLAUDE.md",
			"new_string": `## Einstein Escalation

Einstein invocation is prevented by the validate-routing.sh hook.
See routing-schema.json for the rule definition.

Follow the escalate_to_einstein protocol when needed.
`,
		},
	}

	results := enforcement.AnalyzeToolEventForDocTheater(event)

	if results == nil || len(results) == 0 {
		fmt.Println("Content is clean - no theater patterns detected")
	}

	// Output:
	// Content is clean - no theater patterns detected
}

// ExampleAnalyzeToolEventForDocTheater_filtering demonstrates
// that non-write operations and non-CLAUDE.md files are filtered
func ExampleAnalyzeToolEventForDocTheater_filtering() {
	// Example 1: Read operation (filtered out)
	readEvent := &routing.ToolEvent{
		ToolName: "Read",
		ToolInput: map[string]interface{}{
			"file_path": "/home/user/.claude/CLAUDE.md",
		},
	}

	results1 := enforcement.AnalyzeToolEventForDocTheater(readEvent)
	fmt.Printf("Read operation: %v\n", results1 == nil)

	// Example 2: Non-CLAUDE.md file (filtered out)
	otherFileEvent := &routing.ToolEvent{
		ToolName: "Write",
		ToolInput: map[string]interface{}{
			"file_path": "/home/user/project/README.md",
			"content":   "You MUST NOT do this",
		},
	}

	results2 := enforcement.AnalyzeToolEventForDocTheater(otherFileEvent)
	fmt.Printf("Non-CLAUDE.md file: %v\n", results2 == nil)

	// Output:
	// Read operation: true
	// Non-CLAUDE.md file: true
}
