package enforcement

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Bucket-Chemist/goYoke/pkg/routing"
)

// TestGenerateGuardResponse_AllowCompletion verifies allow decision when no uncollected tasks.
func TestGenerateGuardResponse_AllowCompletion(t *testing.T) {
	// Create test transcript with fully collected tasks
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")

	content := `{"tool_name": "Bash", "tool_input": {"run_in_background": true, "task_id": "bg-1"}, "captured_at": 1000}
{"tool_name": "TaskOutput", "tool_input": {"task_id": "bg-1"}, "captured_at": 2000}
`
	if err := os.WriteFile(transcriptPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test transcript: %v", err)
	}

	// Create analyzer and parse transcript
	analyzer := routing.NewTranscriptAnalyzer(transcriptPath)
	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Create metadata
	metadata := &routing.ParsedAgentMetadata{
		AgentID:    "orchestrator",
		AgentModel: "sonnet",
	}

	// Generate response
	resp := GenerateGuardResponse(analyzer, metadata)

	// Verify decision
	if resp.Decision != "allow" {
		t.Errorf("Expected decision 'allow', got '%s'", resp.Decision)
	}

	// Verify reason
	expectedReason := "All background tasks collected"
	if resp.Reason != expectedReason {
		t.Errorf("Expected reason '%s', got '%s'", expectedReason, resp.Reason)
	}

	// Verify AdditionalContext contains allow message
	if !strings.Contains(resp.AdditionalContext, "✅ ORCHESTRATOR COMPLETION ALLOWED") {
		t.Errorf("AdditionalContext missing allow banner")
	}

	if !strings.Contains(resp.AdditionalContext, "orchestrator") {
		t.Errorf("AdditionalContext missing agent ID")
	}

	if !strings.Contains(resp.AdditionalContext, "sonnet") {
		t.Errorf("AdditionalContext missing model")
	}

	// Verify empty remediation steps
	if len(resp.RemediationSteps) != 0 {
		t.Errorf("Expected empty RemediationSteps for allow, got %d items", len(resp.RemediationSteps))
	}

	// Verify hookEventName
	if resp.HookEventName != "SubagentStop" {
		t.Errorf("Expected hookEventName 'SubagentStop', got '%s'", resp.HookEventName)
	}
}

// TestGenerateGuardResponse_BlockCompletion verifies block decision with uncollected tasks.
func TestGenerateGuardResponse_BlockCompletion(t *testing.T) {
	// Create test transcript with uncollected tasks
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")

	content := `{"tool_name": "Bash", "tool_input": {"run_in_background": true, "task_id": "bg-1"}, "captured_at": 1000}
{"tool_name": "Bash", "tool_input": {"run_in_background": true, "task_id": "bg-2"}, "captured_at": 2000}
{"tool_name": "TaskOutput", "tool_input": {"task_id": "bg-1"}, "captured_at": 3000}
`
	if err := os.WriteFile(transcriptPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test transcript: %v", err)
	}

	// Create analyzer and parse transcript
	analyzer := routing.NewTranscriptAnalyzer(transcriptPath)
	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Create metadata
	metadata := &routing.ParsedAgentMetadata{
		AgentID:    "architect",
		AgentModel: "sonnet",
	}

	// Generate response
	resp := GenerateGuardResponse(analyzer, metadata)

	// Verify decision
	if resp.Decision != "block" {
		t.Errorf("Expected decision 'block', got '%s'", resp.Decision)
	}

	// Verify reason
	expectedReason := "Orchestrator completed with uncollected background tasks"
	if resp.Reason != expectedReason {
		t.Errorf("Expected reason '%s', got '%s'", expectedReason, resp.Reason)
	}

	// Verify AdditionalContext contains block message
	if !strings.Contains(resp.AdditionalContext, "🛑 ORCHESTRATOR COMPLETION BLOCKED") {
		t.Errorf("AdditionalContext missing block banner")
	}

	if !strings.Contains(resp.AdditionalContext, "VIOLATION: Fan-out without fan-in") {
		t.Errorf("AdditionalContext missing violation notice")
	}

	if !strings.Contains(resp.AdditionalContext, "bg-2") {
		t.Errorf("AdditionalContext missing uncollected task ID")
	}

	// Verify remediation steps
	expectedSteps := []string{
		"identify_uncollected_task_ids",
		"call_TaskOutput_for_each",
		"wait_for_all_collections",
		"verify_results_in_transcript",
	}

	if len(resp.RemediationSteps) != len(expectedSteps) {
		t.Errorf("Expected %d remediation steps, got %d", len(expectedSteps), len(resp.RemediationSteps))
	}

	for i, expected := range expectedSteps {
		if i >= len(resp.RemediationSteps) {
			t.Errorf("Missing remediation step %d: %s", i, expected)
			continue
		}
		if resp.RemediationSteps[i] != expected {
			t.Errorf("Remediation step %d: expected '%s', got '%s'", i, expected, resp.RemediationSteps[i])
		}
	}
}

// TestFormatResponseJSON_Valid verifies JSON parsing succeeds.
func TestFormatResponseJSON_Valid(t *testing.T) {
	// Create test response
	resp := &GuardResponse{
		HookEventName:     "SubagentStop",
		Decision:          "block",
		Reason:            "Test reason with \"quotes\" and \nnewlines",
		AdditionalContext: "Context with special chars: \t\r\n",
		RemediationSteps:  []string{"step1", "step2"},
	}

	// Format to JSON
	jsonStr := resp.FormatJSON()

	// Verify it's valid JSON by unmarshaling
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("FormatJSON produced invalid JSON: %v\nJSON: %s", err, jsonStr)
	}

	// Verify structure
	if hookEvent, ok := parsed["hookEventName"].(string); !ok || hookEvent != "SubagentStop" {
		t.Errorf("hookEventName not correctly parsed")
	}

	if decision, ok := parsed["decision"].(string); !ok || decision != "block" {
		t.Errorf("decision not correctly parsed")
	}

	if _, ok := parsed["reason"].(string); !ok {
		t.Errorf("reason not correctly parsed")
	}

	// Verify hookSpecificOutput structure
	if hookOutput, ok := parsed["hookSpecificOutput"].(map[string]interface{}); !ok {
		t.Errorf("hookSpecificOutput not present or wrong type")
	} else {
		if _, ok := hookOutput["additionalContext"].(string); !ok {
			t.Errorf("additionalContext not present in hookSpecificOutput")
		}

		if steps, ok := hookOutput["remediationSteps"].([]interface{}); !ok {
			t.Errorf("remediationSteps not present or wrong type")
		} else if len(steps) != 2 {
			t.Errorf("Expected 2 remediation steps, got %d", len(steps))
		}
	}
}

// TestEscapeJSON verifies all escape sequences work correctly.
func TestEscapeJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`plain text`, `plain text`},
		{`text with "quotes"`, `text with \"quotes\"`},
		{"text with \nnewline", `text with \nnewline`},
		{"text with \ttab", `text with \ttab`},
		{"text with \rcarriage", `text with \rcarriage`},
		{`text with \backslash`, `text with \\backslash`},
		{`complex: "test" \n \t \r`, `complex: \"test\" \\n \\t \\r`},
	}

	for _, test := range tests {
		result := escapeJSON(test.input)
		if result != test.expected {
			t.Errorf("escapeJSON(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

// TestFormatStringArray verifies empty, single, multiple items.
func TestFormatStringArray(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{
			name:     "empty array",
			input:    []string{},
			expected: `[]`,
		},
		{
			name:     "nil array",
			input:    nil,
			expected: `[]`,
		},
		{
			name:     "single item",
			input:    []string{"item1"},
			expected: `["item1"]`,
		},
		{
			name:     "multiple items",
			input:    []string{"item1", "item2", "item3"},
			expected: `["item1", "item2", "item3"]`,
		},
		{
			name:     "items with special chars",
			input:    []string{`item with "quotes"`, "item with \nnewline"},
			expected: `["item with \"quotes\"", "item with \nnewline"]`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := formatStringArray(test.input)
			if result != test.expected {
				t.Errorf("formatStringArray() = %q, expected %q", result, test.expected)
			}

			// Verify it's valid JSON
			var parsed []string
			if err := json.Unmarshal([]byte(result), &parsed); err != nil {
				t.Errorf("formatStringArray produced invalid JSON: %v", err)
			}
		})
	}
}

// TestGuardResponse_ReferencesGuidelines verifies LLM-guidelines.md mentioned in block response.
func TestGuardResponse_ReferencesGuidelines(t *testing.T) {
	// Create test transcript with uncollected task
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")

	content := `{"tool_name": "Bash", "tool_input": {"run_in_background": true, "task_id": "bg-1"}, "captured_at": 1000}`
	if err := os.WriteFile(transcriptPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test transcript: %v", err)
	}

	analyzer := routing.NewTranscriptAnalyzer(transcriptPath)
	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	metadata := &routing.ParsedAgentMetadata{
		AgentID:    "orchestrator",
		AgentModel: "sonnet",
	}

	resp := GenerateGuardResponse(analyzer, metadata)

	// Verify LLM-guidelines reference in block response
	if !strings.Contains(resp.AdditionalContext, "LLM-guidelines.md") {
		t.Errorf("Block response should reference LLM-guidelines.md")
	}

	if !strings.Contains(resp.AdditionalContext, "§ 2.2") {
		t.Errorf("Block response should reference specific section (§ 2.2)")
	}
}

// TestGuardResponse_RemediationSteps verifies remediation steps present when blocking.
func TestGuardResponse_RemediationSteps(t *testing.T) {
	// Create test transcript with uncollected task
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")

	content := `{"tool_name": "Bash", "tool_input": {"run_in_background": true, "task_id": "bg-1"}, "captured_at": 1000}`
	if err := os.WriteFile(transcriptPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test transcript: %v", err)
	}

	analyzer := routing.NewTranscriptAnalyzer(transcriptPath)
	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	metadata := &routing.ParsedAgentMetadata{
		AgentID:    "orchestrator",
		AgentModel: "sonnet",
	}

	resp := GenerateGuardResponse(analyzer, metadata)

	// Verify remediation steps are present
	if len(resp.RemediationSteps) == 0 {
		t.Errorf("Block response must have RemediationSteps")
	}

	// Verify specific required steps
	requiredKeywords := []string{
		"TaskOutput",
		"uncollected",
		"verify",
	}

	stepsStr := strings.Join(resp.RemediationSteps, " ")
	for _, keyword := range requiredKeywords {
		found := false
		for _, step := range resp.RemediationSteps {
			if strings.Contains(strings.ToLower(step), strings.ToLower(keyword)) {
				found = true
				break
			}
		}
		if !found && !strings.Contains(strings.ToLower(stepsStr), strings.ToLower(keyword)) {
			t.Errorf("RemediationSteps should mention '%s'", keyword)
		}
	}

	// Verify AdditionalContext provides clear required actions
	if !strings.Contains(resp.AdditionalContext, "REQUIRED ACTIONS") {
		t.Errorf("Block response should include REQUIRED ACTIONS section")
	}

	if !strings.Contains(resp.AdditionalContext, "TaskOutput") {
		t.Errorf("Block response should explain TaskOutput requirement")
	}
}

// TestGenerateGuardResponse_NoTasks verifies allow decision when no tasks at all.
func TestGenerateGuardResponse_NoTasks(t *testing.T) {
	// Create test transcript with no background tasks
	tmpDir := t.TempDir()
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")

	content := `{"tool_name": "Read", "tool_input": {"file_path": "/test/file.go"}, "captured_at": 1000}
{"tool_name": "Edit", "tool_input": {"file_path": "/test/file.go"}, "captured_at": 2000}
`
	if err := os.WriteFile(transcriptPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test transcript: %v", err)
	}

	analyzer := routing.NewTranscriptAnalyzer(transcriptPath)
	if err := analyzer.Analyze(); err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	metadata := &routing.ParsedAgentMetadata{
		AgentID:    "orchestrator",
		AgentModel: "sonnet",
	}

	resp := GenerateGuardResponse(analyzer, metadata)

	// Verify allow decision (no tasks = no uncollected tasks)
	if resp.Decision != "allow" {
		t.Errorf("Expected decision 'allow' when no tasks, got '%s'", resp.Decision)
	}

	if len(resp.RemediationSteps) != 0 {
		t.Errorf("Expected empty RemediationSteps when allowing, got %d items", len(resp.RemediationSteps))
	}
}

// TestFormatJSON_ComplexContext verifies JSON formatting with multiline context.
func TestFormatJSON_ComplexContext(t *testing.T) {
	resp := &GuardResponse{
		HookEventName: "SubagentStop",
		Decision:      "block",
		Reason:        "Multi-line\nreason\nwith tabs\tand quotes \"test\"",
		AdditionalContext: `🛑 BLOCK HEADER
Agent: orchestrator (model: sonnet)
Uncollected: bg-1, bg-2

VIOLATION: Pattern
Reference: ~/.claude/file.md § 2.2

REQUIRED ACTIONS:
1. Action one
2. Action two`,
		RemediationSteps: []string{"step_one", "step_two"},
	}

	jsonStr := resp.FormatJSON()

	// Verify valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("FormatJSON failed on complex context: %v\nJSON: %s", err, jsonStr)
	}

	// Verify AdditionalContext preserved newlines as \n
	if hookOutput, ok := parsed["hookSpecificOutput"].(map[string]interface{}); ok {
		if context, ok := hookOutput["additionalContext"].(string); ok {
			if !strings.Contains(context, "BLOCK HEADER") {
				t.Errorf("AdditionalContext missing expected content")
			}
			if !strings.Contains(context, "orchestrator") {
				t.Errorf("AdditionalContext missing agent ID")
			}
		} else {
			t.Errorf("additionalContext not a string")
		}
	} else {
		t.Errorf("hookSpecificOutput not found or wrong type")
	}
}

// TestMarshal_ValidOutput verifies Marshal method produces correct JSON structure.
func TestMarshal_ValidOutput(t *testing.T) {
	resp := &GuardResponse{
		HookEventName:     "SubagentStop",
		Decision:          "block",
		Reason:            "Test reason",
		AdditionalContext: "Test context",
		RemediationSteps:  []string{"step1", "step2"},
	}

	var buf strings.Builder
	if err := resp.Marshal(&buf); err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	output := buf.String()

	// Verify valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("Marshal produced invalid JSON: %v\nOutput: %s", err, output)
	}

	// Verify structure
	hookOutput, ok := parsed["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		t.Fatal("Missing hookSpecificOutput wrapper")
	}

	// Verify fields
	if hookEvent := hookOutput["hookEventName"].(string); hookEvent != "SubagentStop" {
		t.Errorf("Expected hookEventName 'SubagentStop', got '%s'", hookEvent)
	}

	if decision := hookOutput["decision"].(string); decision != "block" {
		t.Errorf("Expected decision 'block', got '%s'", decision)
	}

	if reason := hookOutput["reason"].(string); reason != "Test reason" {
		t.Errorf("Expected reason 'Test reason', got '%s'", reason)
	}

	if context := hookOutput["additionalContext"].(string); context != "Test context" {
		t.Errorf("Expected additionalContext 'Test context', got '%s'", context)
	}

	steps, ok := hookOutput["remediationSteps"].([]interface{})
	if !ok || len(steps) != 2 {
		t.Errorf("Expected 2 remediation steps, got %v", hookOutput["remediationSteps"])
	}
}

// TestMarshal_AllowResponse verifies Marshal works correctly for allow decisions.
func TestMarshal_AllowResponse(t *testing.T) {
	resp := &GuardResponse{
		HookEventName:     "SubagentStop",
		Decision:          "allow",
		Reason:            "All tasks collected",
		AdditionalContext: "✅ ALLOWED",
		RemediationSteps:  []string{}, // Empty for allow
	}

	var buf strings.Builder
	if err := resp.Marshal(&buf); err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	output := buf.String()

	// Verify valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("Marshal produced invalid JSON: %v\nOutput: %s", err, output)
	}

	hookOutput := parsed["hookSpecificOutput"].(map[string]interface{})

	if decision := hookOutput["decision"].(string); decision != "allow" {
		t.Errorf("Expected decision 'allow', got '%s'", decision)
	}

	steps := hookOutput["remediationSteps"].([]interface{})
	if len(steps) != 0 {
		t.Errorf("Expected empty remediation steps for allow, got %d items", len(steps))
	}
}

// TestMarshal_ComplexMultilineContext verifies Marshal handles multiline text correctly.
func TestMarshal_ComplexMultilineContext(t *testing.T) {
	resp := &GuardResponse{
		HookEventName: "SubagentStop",
		Decision:      "block",
		Reason:        "Complex\nreason",
		AdditionalContext: `🛑 ORCHESTRATOR COMPLETION BLOCKED

Agent: orchestrator (model: sonnet)
Background Tasks: 2 spawned, 1 collected
⚠️  Uncollected Tasks: 1
  - bg-task-2

VIOLATION: Fan-out without fan-in
Reference: ~/.claude/rules/LLM-guidelines.md § 2.2 (Background Task Collection)

REQUIRED ACTIONS:
1. Call TaskOutput({task_id: "bg-task-2", block: true})
2. Wait for collection to complete
3. Verify results appear in transcript
4. THEN synthesize/conclude orchestration`,
		RemediationSteps: []string{
			"identify_uncollected_task_ids",
			"call_TaskOutput_for_each",
			"wait_for_all_collections",
			"verify_results_in_transcript",
		},
	}

	var buf strings.Builder
	if err := resp.Marshal(&buf); err != nil {
		t.Fatalf("Marshal failed on complex context: %v", err)
	}

	output := buf.String()

	// Verify valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("Marshal produced invalid JSON: %v\nOutput: %s", err, output)
	}

	hookOutput := parsed["hookSpecificOutput"].(map[string]interface{})

	// Verify multiline context preserved
	context := hookOutput["additionalContext"].(string)
	if !strings.Contains(context, "ORCHESTRATOR COMPLETION BLOCKED") {
		t.Errorf("AdditionalContext missing header")
	}
	if !strings.Contains(context, "bg-task-2") {
		t.Errorf("AdditionalContext missing task ID")
	}
	if !strings.Contains(context, "LLM-guidelines.md") {
		t.Errorf("AdditionalContext missing reference")
	}

	// Verify remediation steps
	steps := hookOutput["remediationSteps"].([]interface{})
	if len(steps) != 4 {
		t.Errorf("Expected 4 remediation steps, got %d", len(steps))
	}
}
