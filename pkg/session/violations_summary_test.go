package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

func TestFormatViolationsSummary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		fileContent    string      // JSONL content to write (empty string = don't create file)
		createFile     bool        // Whether to create the file
		maxLines       int         // maxLines parameter
		wantLines      int         // Expected number of output lines
		wantNil        bool        // Expect nil slice (vs empty slice)
		wantErr        bool        // Expect error
		checkContains  []string    // Strings that must appear in output
		checkNotContain []string   // Strings that must NOT appear in output
	}{
		{
			name:       "missing_file_returns_nil",
			createFile: false,
			maxLines:   10,
			wantLines:  0,
			wantNil:    true,
			wantErr:    false,
		},
		{
			name:        "empty_file_returns_empty_slice",
			fileContent: "",
			createFile:  true,
			maxLines:    10,
			wantLines:   0,
			wantNil:     false,
			wantErr:     false,
		},
		{
			name:        "whitespace_only_file_returns_empty_slice",
			fileContent: "   \n\n   \n",
			createFile:  true,
			maxLines:    10,
			wantLines:   0,
			wantNil:     false,
			wantErr:     false,
		},
		{
			name: "tool_permission_violation_formatted",
			fileContent: `{"violation_type":"tool_permission","tool":"Bash","allowed":"Read,Glob","reason":"Tier restriction"}`,
			createFile:  true,
			maxLines:    10,
			wantLines:   1,
			wantNil:     false,
			wantErr:     false,
			checkContains: []string{
				"Tool permission:",
				"**Bash**",
				"(allowed: Read,Glob)",
			},
		},
		{
			name: "blocked_task_opus_violation_formatted",
			fileContent: `{"violation_type":"blocked_task_opus","agent":"orchestrator","reason":"Use /einstein instead"}`,
			createFile:  true,
			maxLines:    10,
			wantLines:   1,
			wantNil:     false,
			wantErr:     false,
			checkContains: []string{
				"Einstein blocking:",
				"Task(model: opus)",
				"**orchestrator**",
			},
		},
		{
			name: "subagent_type_mismatch_violation_formatted",
			fileContent: `{"violation_type":"subagent_type_mismatch","agent":"tech-docs-writer","reason":"Expected general-purpose, got Explore"}`,
			createFile:  true,
			maxLines:    10,
			wantLines:   1,
			wantNil:     false,
			wantErr:     false,
			checkContains: []string{
				"Subagent type:",
				"**tech-docs-writer**",
				"Expected general-purpose, got Explore",
			},
		},
		{
			name: "unknown_violation_type_uses_default_format",
			fileContent: `{"violation_type":"custom_violation","reason":"Something went wrong"}`,
			createFile:  true,
			maxLines:    10,
			wantLines:   1,
			wantNil:     false,
			wantErr:     false,
			checkContains: []string{
				"custom_violation:",
				"Something went wrong",
			},
		},
		{
			name: "multiple_violations_all_types",
			fileContent: `{"violation_type":"tool_permission","tool":"Write","allowed":"Read","reason":"test1"}
{"violation_type":"blocked_task_opus","agent":"einstein","reason":"test2"}
{"violation_type":"subagent_type_mismatch","agent":"scaffolder","reason":"wrong type"}
{"violation_type":"tier_exceeded","reason":"exceeded sonnet limit"}`,
			createFile: true,
			maxLines:   10,
			wantLines:  4,
			wantNil:    false,
			wantErr:    false,
			checkContains: []string{
				"Tool permission:",
				"Einstein blocking:",
				"Subagent type:",
				"tier_exceeded:",
			},
		},
		{
			name: "respects_maxLines_limit",
			fileContent: `{"violation_type":"type1","reason":"first"}
{"violation_type":"type2","reason":"second"}
{"violation_type":"type3","reason":"third"}
{"violation_type":"type4","reason":"fourth"}
{"violation_type":"type5","reason":"fifth"}`,
			createFile: true,
			maxLines:   3,
			wantLines:  3,
			wantNil:    false,
			wantErr:    false,
			// Should contain the LAST 3 (most recent)
			checkContains: []string{
				"type3:", // third
				"type4:", // fourth
				"type5:", // fifth
			},
			checkNotContain: []string{
				"type1:", // first (should be excluded)
				"type2:", // second (should be excluded)
			},
		},
		{
			name: "maxLines_greater_than_violations",
			fileContent: `{"violation_type":"only1","reason":"test1"}
{"violation_type":"only2","reason":"test2"}`,
			createFile:    true,
			maxLines:      100,
			wantLines:     2,
			wantNil:       false,
			wantErr:       false,
			checkContains: []string{"only1:", "only2:"},
		},
		{
			name: "maxLines_zero_returns_all",
			fileContent: `{"violation_type":"a","reason":"r1"}
{"violation_type":"b","reason":"r2"}
{"violation_type":"c","reason":"r3"}`,
			createFile:    true,
			maxLines:      0, // 0 means no limit
			wantLines:     3,
			wantNil:       false,
			wantErr:       false,
			checkContains: []string{"a:", "b:", "c:"},
		},
		{
			name: "malformed_jsonl_lines_skipped",
			fileContent: `not json at all
{"violation_type":"valid1","reason":"this is valid"}
{broken json}
{"violation_type":"valid2","reason":"also valid"}
another invalid line`,
			createFile:    true,
			maxLines:      10,
			wantLines:     2, // Only valid lines
			wantNil:       false,
			wantErr:       false,
			checkContains: []string{"valid1:", "valid2:"},
		},
		{
			name: "most_recent_first_ordering",
			fileContent: `{"violation_type":"oldest","timestamp":"2024-01-01T00:00:00Z","reason":"first written"}
{"violation_type":"middle","timestamp":"2024-01-02T00:00:00Z","reason":"second written"}
{"violation_type":"newest","timestamp":"2024-01-03T00:00:00Z","reason":"last written"}`,
			createFile: true,
			maxLines:   10,
			wantLines:  3,
			wantNil:    false,
			wantErr:    false,
			// Result[0] should be newest (last in file)
			// We can't check ordering directly but we verify all present
			checkContains: []string{"oldest:", "middle:", "newest:"},
		},
		{
			name: "violations_with_all_fields_populated",
			fileContent: `{"timestamp":"2024-01-15T10:30:00Z","session_id":"sess-123","violation_type":"tool_permission","agent":"go-pro","model":"sonnet","tool":"Edit","reason":"Not allowed","allowed":"Read,Glob","override":"none","file":"/path/to/file.go","current_tier":"haiku","required_tier":"sonnet","task_description":"Fix the bug","hook_decision":"block","project_dir":"/home/user/project"}`,
			createFile: true,
			maxLines:   10,
			wantLines:  1,
			wantNil:    false,
			wantErr:    false,
			checkContains: []string{
				"Tool permission:",
				"**Edit**",
				"(allowed: Read,Glob)",
			},
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			path := filepath.Join(tmpDir, "routing-violations.jsonl")

			if tt.createFile {
				if err := os.WriteFile(path, []byte(tt.fileContent), 0644); err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			}

			result, err := FormatViolationsSummary(path, tt.maxLines)

			// Check error expectation
			if tt.wantErr && err == nil {
				t.Errorf("Expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// Check nil vs empty slice
			if tt.wantNil && result != nil {
				t.Errorf("Expected nil slice, got: %v", result)
			}
			if !tt.wantNil && !tt.wantErr && result == nil {
				t.Errorf("Expected non-nil slice, got nil")
			}

			// Check result length
			if len(result) != tt.wantLines {
				t.Errorf("Expected %d lines, got %d: %v", tt.wantLines, len(result), result)
			}

			// Check required contents
			for _, needle := range tt.checkContains {
				found := false
				for _, line := range result {
					if containsString(line, needle) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected output to contain %q, but it wasn't found in: %v", needle, result)
				}
			}

			// Check forbidden contents
			for _, needle := range tt.checkNotContain {
				for _, line := range result {
					if containsString(line, needle) {
						t.Errorf("Expected output NOT to contain %q, but found in: %s", needle, line)
					}
				}
			}
		})
	}
}

// TestFormatViolation tests the formatViolation helper directly
func TestFormatViolation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		v        *routing.Violation
		expected string
	}{
		{
			name: "tool_permission",
			v: &routing.Violation{
				ViolationType: "tool_permission",
				Tool:          "Bash",
				Allowed:       "Read,Glob,Grep",
			},
			expected: "- Tool permission: Tier attempted **Bash** (allowed: Read,Glob,Grep)",
		},
		{
			name: "blocked_task_opus",
			v: &routing.Violation{
				ViolationType: "blocked_task_opus",
				Agent:         "orchestrator",
				Reason:        "Use /einstein slash command",
			},
			expected: "- Einstein blocking: Attempted Task(model: opus) with agent **orchestrator**",
		},
		{
			name: "subagent_type_mismatch",
			v: &routing.Violation{
				ViolationType: "subagent_type_mismatch",
				Agent:         "tech-docs-writer",
				Reason:        "Expected general-purpose, got Explore",
			},
			expected: "- Subagent type: Agent **tech-docs-writer** - Expected general-purpose, got Explore",
		},
		{
			name: "unknown_type_default_format",
			v: &routing.Violation{
				ViolationType: "custom_type",
				Reason:        "Custom reason here",
			},
			expected: "- custom_type: Custom reason here",
		},
		{
			name: "empty_violation_type",
			v: &routing.Violation{
				ViolationType: "",
				Reason:        "No type specified",
			},
			expected: "- : No type specified",
		},
		{
			name: "tool_permission_empty_allowed",
			v: &routing.Violation{
				ViolationType: "tool_permission",
				Tool:          "Write",
				Allowed:       "",
			},
			expected: "- Tool permission: Tier attempted **Write** (allowed: )",
		},
		{
			name: "blocked_task_opus_empty_agent",
			v: &routing.Violation{
				ViolationType: "blocked_task_opus",
				Agent:         "",
				Reason:        "test",
			},
			expected: "- Einstein blocking: Attempted Task(model: opus) with agent ****",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := formatViolation(tt.v)
			if result != tt.expected {
				t.Errorf("formatViolation() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestFormatViolationsSummary_MostRecentFirst verifies ordering
func TestFormatViolationsSummary_MostRecentFirst(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "violations.jsonl")

	// Write violations in chronological order (oldest first)
	content := `{"violation_type":"first","reason":"r1"}
{"violation_type":"second","reason":"r2"}
{"violation_type":"third","reason":"r3"}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	result, err := FormatViolationsSummary(path, 10)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(result))
	}

	// First result should be the LAST violation (most recent)
	if !containsString(result[0], "third:") {
		t.Errorf("First result should contain 'third:', got: %s", result[0])
	}

	// Last result should be the FIRST violation (oldest)
	if !containsString(result[2], "first:") {
		t.Errorf("Last result should contain 'first:', got: %s", result[2])
	}
}

// TestFormatViolationsSummary_MaxLinesEdgeCases tests boundary conditions
func TestFormatViolationsSummary_MaxLinesEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		numViolations int
		maxLines  int
		wantCount int
	}{
		{"maxLines_1", 5, 1, 1},
		{"maxLines_equals_count", 3, 3, 3},
		{"maxLines_exceeds_count", 2, 10, 2},
		{"single_violation_maxLines_1", 1, 1, 1},
		{"negative_maxLines_treated_as_all", 5, -1, 5}, // negative should return all
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			path := filepath.Join(tmpDir, "violations.jsonl")

			// Generate test violations
			var content string
			for i := 0; i < tt.numViolations; i++ {
				content += `{"violation_type":"type` + string(rune('a'+i)) + `","reason":"r"}` + "\n"
			}

			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			result, err := FormatViolationsSummary(path, tt.maxLines)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(result) != tt.wantCount {
				t.Errorf("Expected %d results, got %d", tt.wantCount, len(result))
			}
		})
	}
}

// containsString is a helper to check if haystack contains needle
func containsString(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle || len(needle) == 0 ||
		(len(haystack) > 0 && containsSubstring(haystack, needle)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
