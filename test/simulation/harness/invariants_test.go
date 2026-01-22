package harness

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInvariant_NeverCrash(t *testing.T) {
	inv := PreToolUseInvariants[0] // P1: never_crash

	passed, _ := inv.Check(nil, "", 0, "")
	if !passed {
		t.Error("Expected pass for exit code 0")
	}

	passed, msg := inv.Check(nil, "", 1, "")
	if passed {
		t.Error("Expected fail for exit code 1")
	}
	if msg == "" {
		t.Error("Expected error message")
	}
}

func TestInvariant_ValidJSONOutput(t *testing.T) {
	inv := PreToolUseInvariants[1] // P2: valid_json_output

	tests := []struct {
		output string
		valid  bool
	}{
		{"", true},
		{"{}", true},
		{`{"decision": "allow"}`, true},
		{"not json", false},
		{"{invalid}", false},
	}

	for _, tt := range tests {
		passed, _ := inv.Check(nil, tt.output, 0, "")
		if passed != tt.valid {
			t.Errorf("output %q: got valid=%v, want %v", tt.output, passed, tt.valid)
		}
	}
}

func TestInvariant_NonTaskPassthrough(t *testing.T) {
	inv := PreToolUseInvariants[2] // P3: non_task_passthrough

	tests := []struct {
		name   string
		input  interface{}
		output string
		valid  bool
	}{
		{
			name:   "Task tool - any output is OK",
			input:  map[string]interface{}{"tool_name": "Task"},
			output: `{"decision": "allow"}`,
			valid:  true,
		},
		{
			name:   "Non-Task tool - empty output",
			input:  map[string]interface{}{"tool_name": "Bash"},
			output: "",
			valid:  true,
		},
		{
			name:   "Non-Task tool - minimal output",
			input:  map[string]interface{}{"tool_name": "Read"},
			output: "{}",
			valid:  true,
		},
		{
			name:   "Non-Task tool - non-empty output",
			input:  map[string]interface{}{"tool_name": "Bash"},
			output: `{"decision": "allow"}`,
			valid:  false,
		},
		{
			name:   "Invalid input type",
			input:  "string",
			output: "anything",
			valid:  true, // Can't determine, assume pass
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, _ := inv.Check(tt.input, tt.output, 0, "")
			if passed != tt.valid {
				t.Errorf("got valid=%v, want %v", passed, tt.valid)
			}
		})
	}
}

func TestInvariant_OpusAlwaysBlocked(t *testing.T) {
	inv := PreToolUseInvariants[3] // P4: opus_always_blocked

	tests := []struct {
		name   string
		input  interface{}
		output string
		valid  bool
	}{
		{
			name: "Opus allowed - should fail",
			input: map[string]interface{}{
				"tool_name": "Task",
				"tool_input": map[string]interface{}{
					"model": "opus",
				},
			},
			output: `{"decision": "allow"}`,
			valid:  false,
		},
		{
			name: "Opus blocked - should pass",
			input: map[string]interface{}{
				"tool_name": "Task",
				"tool_input": map[string]interface{}{
					"model": "opus",
				},
			},
			output: `{"decision": "block"}`,
			valid:  true,
		},
		{
			name: "Sonnet - not checked",
			input: map[string]interface{}{
				"tool_name": "Task",
				"tool_input": map[string]interface{}{
					"model": "sonnet",
				},
			},
			output: `{"decision": "allow"}`,
			valid:  true,
		},
		{
			name:   "Non-Task tool",
			input:  map[string]interface{}{"tool_name": "Bash"},
			output: `{}`,
			valid:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, _ := inv.Check(tt.input, tt.output, 0, "")
			if passed != tt.valid {
				t.Errorf("got valid=%v, want %v", passed, tt.valid)
			}
		})
	}
}

func TestInvariant_DecisionIsAllowOrBlock(t *testing.T) {
	inv := PreToolUseInvariants[4] // P5: decision_is_allow_or_block

	tests := []struct {
		name   string
		input  interface{}
		output string
		valid  bool
	}{
		{
			name:   "Decision is allow",
			input:  map[string]interface{}{"tool_name": "Task"},
			output: `{"decision": "allow"}`,
			valid:  true,
		},
		{
			name:   "Decision is block",
			input:  map[string]interface{}{"tool_name": "Task"},
			output: `{"decision": "block"}`,
			valid:  true,
		},
		{
			name:   "Invalid decision value",
			input:  map[string]interface{}{"tool_name": "Task"},
			output: `{"decision": "maybe"}`,
			valid:  false,
		},
		{
			name:   "No decision field - valid",
			input:  map[string]interface{}{"tool_name": "Task"},
			output: `{}`,
			valid:  true,
		},
		{
			name:   "Empty output - valid",
			input:  map[string]interface{}{"tool_name": "Task"},
			output: "",
			valid:  true,
		},
		{
			name:   "Non-Task tool - not checked",
			input:  map[string]interface{}{"tool_name": "Bash"},
			output: `{"decision": "invalid"}`,
			valid:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, _ := inv.Check(tt.input, tt.output, 0, "")
			if passed != tt.valid {
				t.Errorf("got valid=%v, want %v", passed, tt.valid)
			}
		})
	}
}

func TestInvariant_SessionEnd_NeverCrash(t *testing.T) {
	inv := SessionEndInvariants[0] // S1: never_crash

	passed, _ := inv.Check(nil, "", 0, "")
	if !passed {
		t.Error("Expected pass for exit code 0")
	}

	passed, msg := inv.Check(nil, "", 1, "")
	if passed {
		t.Error("Expected fail for exit code 1")
	}
	if msg == "" {
		t.Error("Expected error message")
	}
}

func TestInvariant_HandoffCreated(t *testing.T) {
	inv := SessionEndInvariants[1] // S2: handoff_created

	tempDir := t.TempDir()

	// Should fail when file doesn't exist
	passed, msg := inv.Check(nil, "", 0, tempDir)
	if passed {
		t.Error("Expected fail when handoffs.jsonl doesn't exist")
	}
	if msg == "" {
		t.Error("Expected error message")
	}

	// Create the file
	memoryDir := filepath.Join(tempDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)
	os.WriteFile(filepath.Join(memoryDir, "handoffs.jsonl"), []byte("{}"), 0644)

	passed, _ = inv.Check(nil, "", 0, tempDir)
	if !passed {
		t.Error("Expected pass when handoffs.jsonl exists")
	}
}

func TestInvariant_SchemaVersionCurrent(t *testing.T) {
	inv := SessionEndInvariants[2] // S3: schema_version_current

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	tests := []struct {
		name    string
		content string
		valid   bool
	}{
		{
			name:    "Wrong version",
			content: `{"schema_version": "1.0"}`,
			valid:   false,
		},
		{
			name:    "Correct version",
			content: `{"schema_version": "1.1"}`,
			valid:   true,
		},
		{
			name:    "Multiple lines - last version correct",
			content: "{\"schema_version\": \"1.0\"}\n{\"schema_version\": \"1.1\"}",
			valid:   true,
		},
		{
			name:    "Multiple lines - last version wrong",
			content: "{\"schema_version\": \"1.1\"}\n{\"schema_version\": \"1.0\"}",
			valid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.WriteFile(filepath.Join(memoryDir, "handoffs.jsonl"), []byte(tt.content), 0644)
			passed, _ := inv.Check(nil, "", 0, tempDir)
			if passed != tt.valid {
				t.Errorf("got valid=%v, want %v", passed, tt.valid)
			}
		})
	}
}

func TestInvariant_SchemaVersionCurrent_EdgeCases(t *testing.T) {
	inv := SessionEndInvariants[2] // S3: schema_version_current

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Empty file
	os.WriteFile(filepath.Join(memoryDir, "handoffs.jsonl"), []byte(""), 0644)
	passed, msg := inv.Check(nil, "", 0, tempDir)
	if passed {
		t.Error("Expected fail for empty file")
	}
	if msg == "" {
		t.Error("Expected error message for empty file")
	}

	// Invalid JSON
	os.WriteFile(filepath.Join(memoryDir, "handoffs.jsonl"), []byte("not json"), 0644)
	passed, msg = inv.Check(nil, "", 0, tempDir)
	if passed {
		t.Error("Expected fail for invalid JSON")
	}
	if msg == "" {
		t.Error("Expected error message for invalid JSON")
	}

	// Missing file
	os.Remove(filepath.Join(memoryDir, "handoffs.jsonl"))
	passed, msg = inv.Check(nil, "", 0, tempDir)
	if passed {
		t.Error("Expected fail for missing file")
	}
	if msg == "" {
		t.Error("Expected error message for missing file")
	}
}

func TestInvariant_MarkdownCreated(t *testing.T) {
	inv := SessionEndInvariants[3] // S4: markdown_created

	tempDir := t.TempDir()

	// Should fail when file doesn't exist
	passed, msg := inv.Check(nil, "", 0, tempDir)
	if passed {
		t.Error("Expected fail when last-handoff.md doesn't exist")
	}
	if msg == "" {
		t.Error("Expected error message")
	}

	// Create the file
	memoryDir := filepath.Join(tempDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)
	os.WriteFile(filepath.Join(memoryDir, "last-handoff.md"), []byte("# Handoff"), 0644)

	passed, _ = inv.Check(nil, "", 0, tempDir)
	if !passed {
		t.Error("Expected pass when last-handoff.md exists")
	}
}

func TestCheckInvariants(t *testing.T) {
	invariants := []Invariant{
		PreToolUseInvariants[0], // never_crash
		PreToolUseInvariants[1], // valid_json_output
	}

	results := CheckInvariants(invariants, nil, `{"test": true}`, 0, "")

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got: %d", len(results))
	}

	if !AllPassed(results) {
		t.Error("Expected all invariants to pass")
	}

	// Verify structure
	for _, r := range results {
		if r.InvariantID == "" {
			t.Error("InvariantID should not be empty")
		}
		if !r.Passed && r.Message == "" {
			t.Error("Failed invariant should have message")
		}
	}
}

func TestCheckInvariants_WithInput(t *testing.T) {
	invariants := []Invariant{
		PreToolUseInvariants[0],
	}

	input := map[string]interface{}{
		"tool_name": "Task",
		"model":     "haiku",
	}

	results := CheckInvariants(invariants, input, `{}`, 0, "")

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got: %d", len(results))
	}

	if results[0].Input == "" {
		t.Error("Expected input to be serialized")
	}
}

func TestAllPassed(t *testing.T) {
	tests := []struct {
		name    string
		results []InvariantResult
		want    bool
	}{
		{
			name:    "Empty results",
			results: []InvariantResult{},
			want:    true,
		},
		{
			name: "All passed",
			results: []InvariantResult{
				{InvariantID: "P1", Passed: true},
				{InvariantID: "P2", Passed: true},
			},
			want: true,
		},
		{
			name: "One failed",
			results: []InvariantResult{
				{InvariantID: "P1", Passed: true},
				{InvariantID: "P2", Passed: false},
			},
			want: false,
		},
		{
			name: "All failed",
			results: []InvariantResult{
				{InvariantID: "P1", Passed: false},
				{InvariantID: "P2", Passed: false},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AllPassed(tt.results)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFailedInvariants(t *testing.T) {
	results := []InvariantResult{
		{InvariantID: "P1", Passed: true},
		{InvariantID: "P2", Passed: false, Message: "error 1"},
		{InvariantID: "P3", Passed: true},
		{InvariantID: "P4", Passed: false, Message: "error 2"},
	}

	failed := FailedInvariants(results)

	if len(failed) != 2 {
		t.Errorf("Expected 2 failures, got: %d", len(failed))
	}

	if failed[0].InvariantID != "P2" {
		t.Errorf("Expected first failure to be P2, got: %s", failed[0].InvariantID)
	}
	if failed[1].InvariantID != "P4" {
		t.Errorf("Expected second failure to be P4, got: %s", failed[1].InvariantID)
	}
}

func TestFailedInvariants_AllPassed(t *testing.T) {
	results := []InvariantResult{
		{InvariantID: "P1", Passed: true},
		{InvariantID: "P2", Passed: true},
	}

	failed := FailedInvariants(results)

	if len(failed) != 0 {
		t.Errorf("Expected 0 failures, got: %d", len(failed))
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "Empty string",
			input: "",
			want:  []string{},
		},
		{
			name:  "Single line no newline",
			input: "line1",
			want:  []string{"line1"},
		},
		{
			name:  "Single line with newline",
			input: "line1\n",
			want:  []string{"line1"},
		},
		{
			name:  "Multiple lines",
			input: "line1\nline2\nline3",
			want:  []string{"line1", "line2", "line3"},
		},
		{
			name:  "Multiple lines with trailing newline",
			input: "line1\nline2\n",
			want:  []string{"line1", "line2"},
		},
		{
			name:  "Empty lines in middle",
			input: "line1\n\nline3",
			want:  []string{"line1", "line3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitLines(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("got %d lines, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("line %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestPreToolUseInvariants_Count(t *testing.T) {
	if len(PreToolUseInvariants) != 5 {
		t.Errorf("Expected 5 PreToolUse invariants, got: %d", len(PreToolUseInvariants))
	}

	expectedIDs := []string{"P1", "P2", "P3", "P4", "P5"}
	for i, inv := range PreToolUseInvariants {
		if inv.ID != expectedIDs[i] {
			t.Errorf("Invariant %d: expected ID %s, got %s", i, expectedIDs[i], inv.ID)
		}
		if inv.Name == "" {
			t.Errorf("Invariant %d: Name is empty", i)
		}
		if inv.Check == nil {
			t.Errorf("Invariant %d: Check function is nil", i)
		}
	}
}

func TestSessionEndInvariants_Count(t *testing.T) {
	if len(SessionEndInvariants) != 4 {
		t.Errorf("Expected 4 SessionEnd invariants, got: %d", len(SessionEndInvariants))
	}

	expectedIDs := []string{"S1", "S2", "S3", "S4"}
	for i, inv := range SessionEndInvariants {
		if inv.ID != expectedIDs[i] {
			t.Errorf("Invariant %d: expected ID %s, got %s", i, expectedIDs[i], inv.ID)
		}
		if inv.Name == "" {
			t.Errorf("Invariant %d: Name is empty", i)
		}
		if inv.Check == nil {
			t.Errorf("Invariant %d: Check function is nil", i)
		}
	}
}
