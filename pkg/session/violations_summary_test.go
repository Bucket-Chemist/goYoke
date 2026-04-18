package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/routing"
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

// TestClusterViolationsByType tests the ClusterViolationsByType function
func TestClusterViolationsByType(t *testing.T) {
	t.Parallel()

	t.Run("empty_violations_list", func(t *testing.T) {
		t.Parallel()

		result := ClusterViolationsByType(nil)
		if result == nil {
			t.Error("Expected non-nil map for nil input")
		}
		if len(result) != 0 {
			t.Errorf("Expected empty map, got %d entries", len(result))
		}

		// Also test with empty slice
		result = ClusterViolationsByType([]*routing.Violation{})
		if result == nil {
			t.Error("Expected non-nil map for empty slice")
		}
		if len(result) != 0 {
			t.Errorf("Expected empty map, got %d entries", len(result))
		}
	})

	t.Run("single_violation_type", func(t *testing.T) {
		t.Parallel()

		violations := []*routing.Violation{
			{ViolationType: "tool_permission", Tool: "Bash", Reason: "reason1"},
			{ViolationType: "tool_permission", Tool: "Write", Reason: "reason2"},
		}

		result := ClusterViolationsByType(violations)

		if len(result) != 1 {
			t.Fatalf("Expected 1 cluster, got %d", len(result))
		}

		cluster, exists := result["tool_permission"]
		if !exists {
			t.Fatal("Expected cluster for 'tool_permission'")
		}

		if cluster.Type != "tool_permission" {
			t.Errorf("Expected Type 'tool_permission', got %q", cluster.Type)
		}

		if cluster.Count != 2 {
			t.Errorf("Expected Count 2, got %d", cluster.Count)
		}

		if len(cluster.Samples) != 2 {
			t.Errorf("Expected 2 samples, got %d", len(cluster.Samples))
		}

		// Verify samples are the original violations
		if cluster.Samples[0].Tool != "Bash" {
			t.Errorf("First sample should have Tool 'Bash', got %q", cluster.Samples[0].Tool)
		}
		if cluster.Samples[1].Tool != "Write" {
			t.Errorf("Second sample should have Tool 'Write', got %q", cluster.Samples[1].Tool)
		}
	})

	t.Run("multiple_violation_types", func(t *testing.T) {
		t.Parallel()

		violations := []*routing.Violation{
			{ViolationType: "tool_permission", Tool: "Bash", Reason: "r1"},
			{ViolationType: "blocked_task_opus", Agent: "orchestrator", Reason: "r2"},
			{ViolationType: "subagent_type_mismatch", Agent: "tech-docs-writer", Reason: "r3"},
			{ViolationType: "tool_permission", Tool: "Write", Reason: "r4"},
		}

		result := ClusterViolationsByType(violations)

		if len(result) != 3 {
			t.Fatalf("Expected 3 clusters, got %d", len(result))
		}

		// Check tool_permission cluster
		toolCluster := result["tool_permission"]
		if toolCluster == nil {
			t.Fatal("Missing tool_permission cluster")
		}
		if toolCluster.Count != 2 {
			t.Errorf("tool_permission Count: expected 2, got %d", toolCluster.Count)
		}
		if len(toolCluster.Samples) != 2 {
			t.Errorf("tool_permission Samples: expected 2, got %d", len(toolCluster.Samples))
		}

		// Check blocked_task_opus cluster
		opusCluster := result["blocked_task_opus"]
		if opusCluster == nil {
			t.Fatal("Missing blocked_task_opus cluster")
		}
		if opusCluster.Count != 1 {
			t.Errorf("blocked_task_opus Count: expected 1, got %d", opusCluster.Count)
		}
		if len(opusCluster.Samples) != 1 {
			t.Errorf("blocked_task_opus Samples: expected 1, got %d", len(opusCluster.Samples))
		}

		// Check subagent_type_mismatch cluster
		subagentCluster := result["subagent_type_mismatch"]
		if subagentCluster == nil {
			t.Fatal("Missing subagent_type_mismatch cluster")
		}
		if subagentCluster.Count != 1 {
			t.Errorf("subagent_type_mismatch Count: expected 1, got %d", subagentCluster.Count)
		}
	})

	t.Run("more_than_3_violations_same_type_sample_limit", func(t *testing.T) {
		t.Parallel()

		violations := []*routing.Violation{
			{ViolationType: "tool_permission", Tool: "Tool1", Reason: "r1"},
			{ViolationType: "tool_permission", Tool: "Tool2", Reason: "r2"},
			{ViolationType: "tool_permission", Tool: "Tool3", Reason: "r3"},
			{ViolationType: "tool_permission", Tool: "Tool4", Reason: "r4"},
			{ViolationType: "tool_permission", Tool: "Tool5", Reason: "r5"},
		}

		result := ClusterViolationsByType(violations)

		cluster := result["tool_permission"]
		if cluster == nil {
			t.Fatal("Missing tool_permission cluster")
		}

		// Count should be all 5
		if cluster.Count != 5 {
			t.Errorf("Expected Count 5, got %d", cluster.Count)
		}

		// Samples should be limited to 3
		if len(cluster.Samples) != 3 {
			t.Errorf("Expected 3 samples (limit), got %d", len(cluster.Samples))
		}

		// Verify samples are the FIRST 3 violations
		expectedTools := []string{"Tool1", "Tool2", "Tool3"}
		for i, expected := range expectedTools {
			if cluster.Samples[i].Tool != expected {
				t.Errorf("Sample[%d] should have Tool %q, got %q", i, expected, cluster.Samples[i].Tool)
			}
		}
	})

	t.Run("count_accuracy_large_set", func(t *testing.T) {
		t.Parallel()

		// Create 100 violations of type A, 50 of type B, 25 of type C
		var violations []*routing.Violation

		for i := 0; i < 100; i++ {
			violations = append(violations, &routing.Violation{
				ViolationType: "type_a",
				Reason:        "reason_a",
			})
		}
		for i := 0; i < 50; i++ {
			violations = append(violations, &routing.Violation{
				ViolationType: "type_b",
				Reason:        "reason_b",
			})
		}
		for i := 0; i < 25; i++ {
			violations = append(violations, &routing.Violation{
				ViolationType: "type_c",
				Reason:        "reason_c",
			})
		}

		result := ClusterViolationsByType(violations)

		if len(result) != 3 {
			t.Fatalf("Expected 3 clusters, got %d", len(result))
		}

		// Verify counts
		if result["type_a"].Count != 100 {
			t.Errorf("type_a Count: expected 100, got %d", result["type_a"].Count)
		}
		if result["type_b"].Count != 50 {
			t.Errorf("type_b Count: expected 50, got %d", result["type_b"].Count)
		}
		if result["type_c"].Count != 25 {
			t.Errorf("type_c Count: expected 25, got %d", result["type_c"].Count)
		}

		// Verify sample limits (all should be 3)
		for typeName, cluster := range result {
			if len(cluster.Samples) != 3 {
				t.Errorf("%s Samples: expected 3, got %d", typeName, len(cluster.Samples))
			}
		}
	})

	t.Run("nil_violations_in_slice_skipped", func(t *testing.T) {
		t.Parallel()

		violations := []*routing.Violation{
			{ViolationType: "type_a", Reason: "r1"},
			nil,
			{ViolationType: "type_a", Reason: "r2"},
			nil,
			nil,
			{ViolationType: "type_b", Reason: "r3"},
		}

		result := ClusterViolationsByType(violations)

		if len(result) != 2 {
			t.Fatalf("Expected 2 clusters, got %d", len(result))
		}

		if result["type_a"].Count != 2 {
			t.Errorf("type_a Count: expected 2, got %d", result["type_a"].Count)
		}
		if result["type_b"].Count != 1 {
			t.Errorf("type_b Count: expected 1, got %d", result["type_b"].Count)
		}
	})

	t.Run("empty_violation_type_string", func(t *testing.T) {
		t.Parallel()

		violations := []*routing.Violation{
			{ViolationType: "", Reason: "empty type 1"},
			{ViolationType: "", Reason: "empty type 2"},
			{ViolationType: "valid_type", Reason: "valid"},
		}

		result := ClusterViolationsByType(violations)

		if len(result) != 2 {
			t.Fatalf("Expected 2 clusters, got %d", len(result))
		}

		// Empty string is a valid key
		emptyCluster := result[""]
		if emptyCluster == nil {
			t.Fatal("Missing cluster for empty string type")
		}
		if emptyCluster.Count != 2 {
			t.Errorf("Empty type Count: expected 2, got %d", emptyCluster.Count)
		}

		validCluster := result["valid_type"]
		if validCluster == nil {
			t.Fatal("Missing cluster for valid_type")
		}
		if validCluster.Count != 1 {
			t.Errorf("valid_type Count: expected 1, got %d", validCluster.Count)
		}
	})

	t.Run("preserves_violation_references", func(t *testing.T) {
		t.Parallel()

		v1 := &routing.Violation{ViolationType: "test", Tool: "Tool1", Reason: "r1"}
		v2 := &routing.Violation{ViolationType: "test", Tool: "Tool2", Reason: "r2"}

		violations := []*routing.Violation{v1, v2}

		result := ClusterViolationsByType(violations)
		cluster := result["test"]

		// Samples should be the same pointers, not copies
		if cluster.Samples[0] != v1 {
			t.Error("Sample[0] should be same pointer as v1")
		}
		if cluster.Samples[1] != v2 {
			t.Error("Sample[1] should be same pointer as v2")
		}
	})
}

// TestClusterViolationsByAgent tests the ClusterViolationsByAgent function
func TestClusterViolationsByAgent(t *testing.T) {
	t.Parallel()

	t.Run("empty_violations_list", func(t *testing.T) {
		t.Parallel()

		result := ClusterViolationsByAgent(nil)
		if result == nil {
			t.Error("Expected non-nil map for nil input")
		}
		if len(result) != 0 {
			t.Errorf("Expected empty map, got %d entries", len(result))
		}

		// Also test with empty slice
		result = ClusterViolationsByAgent([]*routing.Violation{})
		if result == nil {
			t.Error("Expected non-nil map for empty slice")
		}
		if len(result) != 0 {
			t.Errorf("Expected empty map, got %d entries", len(result))
		}
	})

	t.Run("single_agent_with_violations", func(t *testing.T) {
		t.Parallel()

		violations := []*routing.Violation{
			{ViolationType: "tool_permission", Agent: "python-pro", Reason: "r1"},
			{ViolationType: "subagent_type_mismatch", Agent: "python-pro", Reason: "r2"},
		}

		result := ClusterViolationsByAgent(violations)

		if len(result) != 1 {
			t.Fatalf("Expected 1 cluster, got %d", len(result))
		}

		cluster, exists := result["python-pro"]
		if !exists {
			t.Fatal("Expected cluster for 'python-pro'")
		}

		if cluster.Agent != "python-pro" {
			t.Errorf("Expected Agent 'python-pro', got %q", cluster.Agent)
		}

		if cluster.TotalCount != 2 {
			t.Errorf("Expected TotalCount 2, got %d", cluster.TotalCount)
		}

		if len(cluster.Samples) != 2 {
			t.Errorf("Expected 2 samples, got %d", len(cluster.Samples))
		}

		// Check ByType breakdown
		if cluster.ByType["tool_permission"] != 1 {
			t.Errorf("Expected 1 tool_permission, got %d", cluster.ByType["tool_permission"])
		}
		if cluster.ByType["subagent_type_mismatch"] != 1 {
			t.Errorf("Expected 1 subagent_type_mismatch, got %d", cluster.ByType["subagent_type_mismatch"])
		}
	})

	t.Run("multiple_agents", func(t *testing.T) {
		t.Parallel()

		violations := []*routing.Violation{
			{ViolationType: "tool_permission", Agent: "python-pro", Reason: "r1"},
			{ViolationType: "subagent_type_mismatch", Agent: "tech-docs-writer", Reason: "r2"},
			{ViolationType: "tool_permission", Agent: "scaffolder", Reason: "r3"},
			{ViolationType: "blocked_task_opus", Agent: "python-pro", Reason: "r4"},
		}

		result := ClusterViolationsByAgent(violations)

		if len(result) != 3 {
			t.Fatalf("Expected 3 clusters, got %d", len(result))
		}

		// Check python-pro cluster
		pythonCluster := result["python-pro"]
		if pythonCluster == nil {
			t.Fatal("Missing python-pro cluster")
		}
		if pythonCluster.TotalCount != 2 {
			t.Errorf("python-pro TotalCount: expected 2, got %d", pythonCluster.TotalCount)
		}
		if pythonCluster.ByType["tool_permission"] != 1 {
			t.Errorf("python-pro tool_permission: expected 1, got %d", pythonCluster.ByType["tool_permission"])
		}
		if pythonCluster.ByType["blocked_task_opus"] != 1 {
			t.Errorf("python-pro blocked_task_opus: expected 1, got %d", pythonCluster.ByType["blocked_task_opus"])
		}

		// Check tech-docs-writer cluster
		docsCluster := result["tech-docs-writer"]
		if docsCluster == nil {
			t.Fatal("Missing tech-docs-writer cluster")
		}
		if docsCluster.TotalCount != 1 {
			t.Errorf("tech-docs-writer TotalCount: expected 1, got %d", docsCluster.TotalCount)
		}

		// Check scaffolder cluster
		scaffolderCluster := result["scaffolder"]
		if scaffolderCluster == nil {
			t.Fatal("Missing scaffolder cluster")
		}
		if scaffolderCluster.TotalCount != 1 {
			t.Errorf("scaffolder TotalCount: expected 1, got %d", scaffolderCluster.TotalCount)
		}
	})

	t.Run("multiple_violation_types_per_agent", func(t *testing.T) {
		t.Parallel()

		violations := []*routing.Violation{
			{ViolationType: "subagent_type_mismatch", Agent: "python-pro", Reason: "r1"},
			{ViolationType: "subagent_type_mismatch", Agent: "python-pro", Reason: "r2"},
			{ViolationType: "subagent_type_mismatch", Agent: "python-pro", Reason: "r3"},
			{ViolationType: "delegation_ceiling", Agent: "python-pro", Reason: "r4"},
			{ViolationType: "delegation_ceiling", Agent: "python-pro", Reason: "r5"},
		}

		result := ClusterViolationsByAgent(violations)

		cluster := result["python-pro"]
		if cluster == nil {
			t.Fatal("Missing python-pro cluster")
		}

		if cluster.TotalCount != 5 {
			t.Errorf("Expected TotalCount 5, got %d", cluster.TotalCount)
		}

		if cluster.ByType["subagent_type_mismatch"] != 3 {
			t.Errorf("Expected 3 subagent_type_mismatch, got %d", cluster.ByType["subagent_type_mismatch"])
		}
		if cluster.ByType["delegation_ceiling"] != 2 {
			t.Errorf("Expected 2 delegation_ceiling, got %d", cluster.ByType["delegation_ceiling"])
		}
	})

	t.Run("empty_agent_grouped_as_unknown", func(t *testing.T) {
		t.Parallel()

		violations := []*routing.Violation{
			{ViolationType: "tool_permission", Agent: "", Reason: "no agent 1"},
			{ViolationType: "tier_exceeded", Agent: "", Reason: "no agent 2"},
			{ViolationType: "tool_permission", Agent: "go-pro", Reason: "has agent"},
		}

		result := ClusterViolationsByAgent(violations)

		if len(result) != 2 {
			t.Fatalf("Expected 2 clusters, got %d", len(result))
		}

		// Check unknown cluster
		unknownCluster := result["unknown"]
		if unknownCluster == nil {
			t.Fatal("Missing 'unknown' cluster for empty agent")
		}
		if unknownCluster.Agent != "unknown" {
			t.Errorf("Expected Agent 'unknown', got %q", unknownCluster.Agent)
		}
		if unknownCluster.TotalCount != 2 {
			t.Errorf("unknown TotalCount: expected 2, got %d", unknownCluster.TotalCount)
		}

		// Check go-pro cluster
		goProCluster := result["go-pro"]
		if goProCluster == nil {
			t.Fatal("Missing go-pro cluster")
		}
		if goProCluster.TotalCount != 1 {
			t.Errorf("go-pro TotalCount: expected 1, got %d", goProCluster.TotalCount)
		}
	})

	t.Run("more_than_3_violations_sample_limit", func(t *testing.T) {
		t.Parallel()

		violations := []*routing.Violation{
			{ViolationType: "type1", Agent: "test-agent", Reason: "r1"},
			{ViolationType: "type2", Agent: "test-agent", Reason: "r2"},
			{ViolationType: "type3", Agent: "test-agent", Reason: "r3"},
			{ViolationType: "type4", Agent: "test-agent", Reason: "r4"},
			{ViolationType: "type5", Agent: "test-agent", Reason: "r5"},
		}

		result := ClusterViolationsByAgent(violations)

		cluster := result["test-agent"]
		if cluster == nil {
			t.Fatal("Missing test-agent cluster")
		}

		// TotalCount should be all 5
		if cluster.TotalCount != 5 {
			t.Errorf("Expected TotalCount 5, got %d", cluster.TotalCount)
		}

		// Samples should be limited to 3
		if len(cluster.Samples) != 3 {
			t.Errorf("Expected 3 samples (limit), got %d", len(cluster.Samples))
		}

		// Verify samples are the FIRST 3 violations
		expectedReasons := []string{"r1", "r2", "r3"}
		for i, expected := range expectedReasons {
			if cluster.Samples[i].Reason != expected {
				t.Errorf("Sample[%d] should have Reason %q, got %q", i, expected, cluster.Samples[i].Reason)
			}
		}
	})

	t.Run("nil_violations_in_slice_skipped", func(t *testing.T) {
		t.Parallel()

		violations := []*routing.Violation{
			{ViolationType: "type_a", Agent: "agent1", Reason: "r1"},
			nil,
			{ViolationType: "type_b", Agent: "agent1", Reason: "r2"},
			nil,
			nil,
			{ViolationType: "type_c", Agent: "agent2", Reason: "r3"},
		}

		result := ClusterViolationsByAgent(violations)

		if len(result) != 2 {
			t.Fatalf("Expected 2 clusters, got %d", len(result))
		}

		if result["agent1"].TotalCount != 2 {
			t.Errorf("agent1 TotalCount: expected 2, got %d", result["agent1"].TotalCount)
		}
		if result["agent2"].TotalCount != 1 {
			t.Errorf("agent2 TotalCount: expected 1, got %d", result["agent2"].TotalCount)
		}
	})

	t.Run("cross_reference_counts_match_type_clustering", func(t *testing.T) {
		t.Parallel()

		// Create violations that can be verified both ways
		violations := []*routing.Violation{
			{ViolationType: "tool_permission", Agent: "python-pro", Reason: "r1"},
			{ViolationType: "tool_permission", Agent: "go-pro", Reason: "r2"},
			{ViolationType: "subagent_type_mismatch", Agent: "python-pro", Reason: "r3"},
			{ViolationType: "subagent_type_mismatch", Agent: "python-pro", Reason: "r4"},
			{ViolationType: "subagent_type_mismatch", Agent: "scaffolder", Reason: "r5"},
		}

		// Get both clustering results
		byAgent := ClusterViolationsByAgent(violations)
		byType := ClusterViolationsByType(violations)

		// Verify total counts match
		totalByAgent := 0
		for _, cluster := range byAgent {
			totalByAgent += cluster.TotalCount
		}
		totalByType := 0
		for _, cluster := range byType {
			totalByType += cluster.Count
		}

		if totalByAgent != totalByType {
			t.Errorf("Total mismatch: byAgent=%d, byType=%d", totalByAgent, totalByType)
		}

		if totalByAgent != 5 {
			t.Errorf("Expected total 5, got %d", totalByAgent)
		}

		// Verify specific cross-reference
		// tool_permission: 2 total (python-pro:1, go-pro:1)
		if byType["tool_permission"].Count != 2 {
			t.Errorf("Expected 2 tool_permission by type, got %d", byType["tool_permission"].Count)
		}
		if byAgent["python-pro"].ByType["tool_permission"] != 1 {
			t.Errorf("Expected 1 tool_permission for python-pro, got %d", byAgent["python-pro"].ByType["tool_permission"])
		}
		if byAgent["go-pro"].ByType["tool_permission"] != 1 {
			t.Errorf("Expected 1 tool_permission for go-pro, got %d", byAgent["go-pro"].ByType["tool_permission"])
		}

		// subagent_type_mismatch: 3 total (python-pro:2, scaffolder:1)
		if byType["subagent_type_mismatch"].Count != 3 {
			t.Errorf("Expected 3 subagent_type_mismatch by type, got %d", byType["subagent_type_mismatch"].Count)
		}
		if byAgent["python-pro"].ByType["subagent_type_mismatch"] != 2 {
			t.Errorf("Expected 2 subagent_type_mismatch for python-pro, got %d", byAgent["python-pro"].ByType["subagent_type_mismatch"])
		}
		if byAgent["scaffolder"].ByType["subagent_type_mismatch"] != 1 {
			t.Errorf("Expected 1 subagent_type_mismatch for scaffolder, got %d", byAgent["scaffolder"].ByType["subagent_type_mismatch"])
		}
	})

	t.Run("preserves_violation_references", func(t *testing.T) {
		t.Parallel()

		v1 := &routing.Violation{ViolationType: "test", Agent: "my-agent", Reason: "r1"}
		v2 := &routing.Violation{ViolationType: "test", Agent: "my-agent", Reason: "r2"}

		violations := []*routing.Violation{v1, v2}

		result := ClusterViolationsByAgent(violations)
		cluster := result["my-agent"]

		// Samples should be the same pointers, not copies
		if cluster.Samples[0] != v1 {
			t.Error("Sample[0] should be same pointer as v1")
		}
		if cluster.Samples[1] != v2 {
			t.Error("Sample[1] should be same pointer as v2")
		}
	})

	t.Run("count_accuracy_large_set", func(t *testing.T) {
		t.Parallel()

		// Create 100 violations for agent_a, 50 for agent_b, 25 for agent_c
		var violations []*routing.Violation

		for i := 0; i < 100; i++ {
			violations = append(violations, &routing.Violation{
				ViolationType: "type_x",
				Agent:         "agent_a",
				Reason:        "reason_a",
			})
		}
		for i := 0; i < 50; i++ {
			violations = append(violations, &routing.Violation{
				ViolationType: "type_y",
				Agent:         "agent_b",
				Reason:        "reason_b",
			})
		}
		for i := 0; i < 25; i++ {
			violations = append(violations, &routing.Violation{
				ViolationType: "type_z",
				Agent:         "agent_c",
				Reason:        "reason_c",
			})
		}

		result := ClusterViolationsByAgent(violations)

		if len(result) != 3 {
			t.Fatalf("Expected 3 clusters, got %d", len(result))
		}

		// Verify counts
		if result["agent_a"].TotalCount != 100 {
			t.Errorf("agent_a TotalCount: expected 100, got %d", result["agent_a"].TotalCount)
		}
		if result["agent_b"].TotalCount != 50 {
			t.Errorf("agent_b TotalCount: expected 50, got %d", result["agent_b"].TotalCount)
		}
		if result["agent_c"].TotalCount != 25 {
			t.Errorf("agent_c TotalCount: expected 25, got %d", result["agent_c"].TotalCount)
		}

		// Verify sample limits (all should be 3)
		for agentName, cluster := range result {
			if len(cluster.Samples) != 3 {
				t.Errorf("%s Samples: expected 3, got %d", agentName, len(cluster.Samples))
			}
		}
	})
}

// TestAnalyzeViolationTrend tests the AnalyzeViolationTrend function
func TestAnalyzeViolationTrend(t *testing.T) {
	t.Parallel()

	t.Run("nil_input_returns_insufficient_data", func(t *testing.T) {
		t.Parallel()

		result := AnalyzeViolationTrend(nil)

		if result == nil {
			t.Fatal("Expected non-nil result for nil input")
		}
		if result.Trend != "insufficient_data" {
			t.Errorf("Expected Trend 'insufficient_data', got %q", result.Trend)
		}
		if result.EarlyCount != 0 {
			t.Errorf("Expected EarlyCount 0, got %d", result.EarlyCount)
		}
		if result.LateCount != 0 {
			t.Errorf("Expected LateCount 0, got %d", result.LateCount)
		}
	})

	t.Run("empty_slice_returns_insufficient_data", func(t *testing.T) {
		t.Parallel()

		result := AnalyzeViolationTrend([]*routing.Violation{})

		if result.Trend != "insufficient_data" {
			t.Errorf("Expected Trend 'insufficient_data', got %q", result.Trend)
		}
	})

	t.Run("single_violation_returns_insufficient_data", func(t *testing.T) {
		t.Parallel()

		violations := []*routing.Violation{
			{Timestamp: "2024-01-15T10:00:00Z", ViolationType: "test"},
		}

		result := AnalyzeViolationTrend(violations)

		if result.Trend != "insufficient_data" {
			t.Errorf("Expected Trend 'insufficient_data', got %q", result.Trend)
		}
		if !containsString(result.Message, "1 violation") {
			t.Errorf("Expected message to mention single violation, got %q", result.Message)
		}
	})

	t.Run("improving_trend_more_early_fewer_late", func(t *testing.T) {
		t.Parallel()

		// 5 violations in first half, 2 in second half
		violations := []*routing.Violation{
			{Timestamp: "2024-01-15T10:00:00Z", ViolationType: "a"},
			{Timestamp: "2024-01-15T10:10:00Z", ViolationType: "b"},
			{Timestamp: "2024-01-15T10:20:00Z", ViolationType: "c"},
			{Timestamp: "2024-01-15T10:25:00Z", ViolationType: "d"},
			{Timestamp: "2024-01-15T10:30:00Z", ViolationType: "e"}, // midpoint is ~10:30
			{Timestamp: "2024-01-15T10:50:00Z", ViolationType: "f"},
			{Timestamp: "2024-01-15T11:00:00Z", ViolationType: "g"},
		}

		result := AnalyzeViolationTrend(violations)

		if result.Trend != "improving" {
			t.Errorf("Expected Trend 'improving', got %q", result.Trend)
		}
		if result.EarlyCount <= result.LateCount {
			t.Errorf("EarlyCount (%d) should be > LateCount (%d)", result.EarlyCount, result.LateCount)
		}
	})

	t.Run("worsening_trend_fewer_early_more_late", func(t *testing.T) {
		t.Parallel()

		// 2 violations in first half, 5 in second half
		violations := []*routing.Violation{
			{Timestamp: "2024-01-15T10:00:00Z", ViolationType: "a"},
			{Timestamp: "2024-01-15T10:10:00Z", ViolationType: "b"},
			{Timestamp: "2024-01-15T10:35:00Z", ViolationType: "c"}, // midpoint is ~10:30
			{Timestamp: "2024-01-15T10:40:00Z", ViolationType: "d"},
			{Timestamp: "2024-01-15T10:50:00Z", ViolationType: "e"},
			{Timestamp: "2024-01-15T10:55:00Z", ViolationType: "f"},
			{Timestamp: "2024-01-15T11:00:00Z", ViolationType: "g"},
		}

		result := AnalyzeViolationTrend(violations)

		if result.Trend != "worsening" {
			t.Errorf("Expected Trend 'worsening', got %q", result.Trend)
		}
		if result.LateCount <= result.EarlyCount {
			t.Errorf("LateCount (%d) should be > EarlyCount (%d)", result.LateCount, result.EarlyCount)
		}
	})

	t.Run("stable_trend_equal_distribution", func(t *testing.T) {
		t.Parallel()

		// 3 violations in first half, 3 in second half
		violations := []*routing.Violation{
			{Timestamp: "2024-01-15T10:00:00Z", ViolationType: "a"},
			{Timestamp: "2024-01-15T10:10:00Z", ViolationType: "b"},
			{Timestamp: "2024-01-15T10:20:00Z", ViolationType: "c"},
			{Timestamp: "2024-01-15T10:40:00Z", ViolationType: "d"},
			{Timestamp: "2024-01-15T10:50:00Z", ViolationType: "e"},
			{Timestamp: "2024-01-15T11:00:00Z", ViolationType: "f"},
		}

		result := AnalyzeViolationTrend(violations)

		if result.Trend != "stable" {
			t.Errorf("Expected Trend 'stable', got %q", result.Trend)
		}
		if result.EarlyCount != result.LateCount {
			t.Errorf("EarlyCount (%d) should == LateCount (%d)", result.EarlyCount, result.LateCount)
		}
	})

	t.Run("all_violations_in_first_half", func(t *testing.T) {
		t.Parallel()

		// All clustered at the start
		violations := []*routing.Violation{
			{Timestamp: "2024-01-15T10:00:00Z", ViolationType: "a"},
			{Timestamp: "2024-01-15T10:01:00Z", ViolationType: "b"},
			{Timestamp: "2024-01-15T10:02:00Z", ViolationType: "c"},
			{Timestamp: "2024-01-15T10:03:00Z", ViolationType: "d"},
			{Timestamp: "2024-01-15T11:00:00Z", ViolationType: "e"}, // Last one (far future)
		}

		result := AnalyzeViolationTrend(violations)

		if result.Trend != "improving" {
			t.Errorf("Expected Trend 'improving' when most violations are early, got %q", result.Trend)
		}
		// 4 in first half (before ~10:30), 1 in second half
		if result.EarlyCount < 3 {
			t.Errorf("Expected at least 3 early violations, got %d", result.EarlyCount)
		}
	})

	t.Run("all_violations_in_second_half", func(t *testing.T) {
		t.Parallel()

		// First one at start, rest clustered at end
		violations := []*routing.Violation{
			{Timestamp: "2024-01-15T10:00:00Z", ViolationType: "a"},
			{Timestamp: "2024-01-15T10:57:00Z", ViolationType: "b"},
			{Timestamp: "2024-01-15T10:58:00Z", ViolationType: "c"},
			{Timestamp: "2024-01-15T10:59:00Z", ViolationType: "d"},
			{Timestamp: "2024-01-15T11:00:00Z", ViolationType: "e"},
		}

		result := AnalyzeViolationTrend(violations)

		if result.Trend != "worsening" {
			t.Errorf("Expected Trend 'worsening' when most violations are late, got %q", result.Trend)
		}
		// 1 in first half (before ~10:30), 4 in second half
		if result.LateCount < 3 {
			t.Errorf("Expected at least 3 late violations, got %d", result.LateCount)
		}
	})

	t.Run("same_timestamp_returns_stable", func(t *testing.T) {
		t.Parallel()

		sameTime := "2024-01-15T10:30:00Z"
		violations := []*routing.Violation{
			{Timestamp: sameTime, ViolationType: "a"},
			{Timestamp: sameTime, ViolationType: "b"},
			{Timestamp: sameTime, ViolationType: "c"},
			{Timestamp: sameTime, ViolationType: "d"},
		}

		result := AnalyzeViolationTrend(violations)

		if result.Trend != "stable" {
			t.Errorf("Expected Trend 'stable' for same timestamps, got %q", result.Trend)
		}
		if result.EarlyCount != 4 {
			t.Errorf("Expected EarlyCount 4 (all assigned to early), got %d", result.EarlyCount)
		}
		if result.LateCount != 0 {
			t.Errorf("Expected LateCount 0, got %d", result.LateCount)
		}
		if !containsString(result.Message, "same timestamp") {
			t.Errorf("Expected message to mention same timestamp, got %q", result.Message)
		}
	})

	t.Run("nil_violations_in_slice_skipped", func(t *testing.T) {
		t.Parallel()

		violations := []*routing.Violation{
			{Timestamp: "2024-01-15T10:00:00Z", ViolationType: "a"},
			nil,
			{Timestamp: "2024-01-15T10:30:00Z", ViolationType: "b"},
			nil,
			{Timestamp: "2024-01-15T11:00:00Z", ViolationType: "c"},
		}

		result := AnalyzeViolationTrend(violations)

		// Should process 3 valid violations
		totalProcessed := result.EarlyCount + result.LateCount
		if totalProcessed != 3 {
			t.Errorf("Expected 3 violations processed, got %d", totalProcessed)
		}
	})

	t.Run("invalid_timestamps_skipped", func(t *testing.T) {
		t.Parallel()

		violations := []*routing.Violation{
			{Timestamp: "2024-01-15T10:00:00Z", ViolationType: "valid1"},
			{Timestamp: "not-a-timestamp", ViolationType: "invalid1"},
			{Timestamp: "2024-01-15T11:00:00Z", ViolationType: "valid2"},
			{Timestamp: "2024/01/15", ViolationType: "invalid2"},
			{Timestamp: "", ViolationType: "empty"},
		}

		result := AnalyzeViolationTrend(violations)

		// Should process only 2 valid violations
		totalProcessed := result.EarlyCount + result.LateCount
		if totalProcessed != 2 {
			t.Errorf("Expected 2 violations processed, got %d", totalProcessed)
		}
		// With only 2 violations, we should get stable (1 early, 1 late)
		if result.Trend != "stable" {
			t.Errorf("Expected Trend 'stable' for 2 violations at start/end, got %q", result.Trend)
		}
	})

	t.Run("unsorted_violations_sorted_correctly", func(t *testing.T) {
		t.Parallel()

		// Violations not in chronological order
		violations := []*routing.Violation{
			{Timestamp: "2024-01-15T10:50:00Z", ViolationType: "e"},
			{Timestamp: "2024-01-15T10:00:00Z", ViolationType: "a"},
			{Timestamp: "2024-01-15T11:00:00Z", ViolationType: "f"},
			{Timestamp: "2024-01-15T10:10:00Z", ViolationType: "b"},
			{Timestamp: "2024-01-15T10:20:00Z", ViolationType: "c"},
			{Timestamp: "2024-01-15T10:30:00Z", ViolationType: "d"},
		}

		result := AnalyzeViolationTrend(violations)

		// Should still compute correctly
		// Midpoint is between 10:00 and 11:00 = 10:30
		// Early: a, b, c, d (10:00, 10:10, 10:20, 10:30)
		// Late: e, f (10:50, 11:00)
		if result.EarlyCount != 4 {
			t.Errorf("Expected EarlyCount 4, got %d", result.EarlyCount)
		}
		if result.LateCount != 2 {
			t.Errorf("Expected LateCount 2, got %d", result.LateCount)
		}
		if result.Trend != "improving" {
			t.Errorf("Expected Trend 'improving', got %q", result.Trend)
		}
	})

	t.Run("two_violations_exact_split", func(t *testing.T) {
		t.Parallel()

		violations := []*routing.Violation{
			{Timestamp: "2024-01-15T10:00:00Z", ViolationType: "a"},
			{Timestamp: "2024-01-15T11:00:00Z", ViolationType: "b"},
		}

		result := AnalyzeViolationTrend(violations)

		// Midpoint is 10:30
		// a (10:00) is before/equal midpoint -> early
		// b (11:00) is after midpoint -> late
		if result.EarlyCount != 1 {
			t.Errorf("Expected EarlyCount 1, got %d", result.EarlyCount)
		}
		if result.LateCount != 1 {
			t.Errorf("Expected LateCount 1, got %d", result.LateCount)
		}
		if result.Trend != "stable" {
			t.Errorf("Expected Trend 'stable', got %q", result.Trend)
		}
	})

	t.Run("violation_at_exact_midpoint_counted_as_early", func(t *testing.T) {
		t.Parallel()

		// Midpoint = 10:30
		violations := []*routing.Violation{
			{Timestamp: "2024-01-15T10:00:00Z", ViolationType: "a"},
			{Timestamp: "2024-01-15T10:30:00Z", ViolationType: "b"}, // exactly at midpoint
			{Timestamp: "2024-01-15T11:00:00Z", ViolationType: "c"},
		}

		result := AnalyzeViolationTrend(violations)

		// Midpoint violation should be counted as early
		if result.EarlyCount != 2 {
			t.Errorf("Expected EarlyCount 2 (including midpoint), got %d", result.EarlyCount)
		}
		if result.LateCount != 1 {
			t.Errorf("Expected LateCount 1, got %d", result.LateCount)
		}
	})

	t.Run("message_format_improving", func(t *testing.T) {
		t.Parallel()

		violations := []*routing.Violation{
			{Timestamp: "2024-01-15T10:00:00Z", ViolationType: "a"},
			{Timestamp: "2024-01-15T10:10:00Z", ViolationType: "b"},
			{Timestamp: "2024-01-15T10:20:00Z", ViolationType: "c"},
			{Timestamp: "2024-01-15T11:00:00Z", ViolationType: "d"},
		}

		result := AnalyzeViolationTrend(violations)

		if !containsString(result.Message, "decreased") {
			t.Errorf("Expected improving message to contain 'decreased', got %q", result.Message)
		}
		if !containsString(result.Message, "early") {
			t.Errorf("Expected message to contain 'early', got %q", result.Message)
		}
		if !containsString(result.Message, "late") {
			t.Errorf("Expected message to contain 'late', got %q", result.Message)
		}
	})

	t.Run("message_format_worsening", func(t *testing.T) {
		t.Parallel()

		violations := []*routing.Violation{
			{Timestamp: "2024-01-15T10:00:00Z", ViolationType: "a"},
			{Timestamp: "2024-01-15T10:40:00Z", ViolationType: "b"},
			{Timestamp: "2024-01-15T10:50:00Z", ViolationType: "c"},
			{Timestamp: "2024-01-15T11:00:00Z", ViolationType: "d"},
		}

		result := AnalyzeViolationTrend(violations)

		if !containsString(result.Message, "increased") {
			t.Errorf("Expected worsening message to contain 'increased', got %q", result.Message)
		}
	})

	t.Run("large_set_count_accuracy", func(t *testing.T) {
		t.Parallel()

		// Create 100 violations: 70 in first half, 30 in second half
		var violations []*routing.Violation
		baseTime := "2024-01-15T10:00:00Z"
		bt, _ := time.Parse(time.RFC3339, baseTime)

		// First 70: within first 30 minutes (midpoint will be at 30 min)
		for i := 0; i < 70; i++ {
			t := bt.Add(time.Duration(i) * 20 * time.Second) // 0-23 minutes
			violations = append(violations, &routing.Violation{
				Timestamp:     t.Format(time.RFC3339),
				ViolationType: "early",
			})
		}
		// Last 30: after midpoint
		for i := 0; i < 30; i++ {
			t := bt.Add(35*time.Minute + time.Duration(i)*30*time.Second) // 35-50 minutes
			violations = append(violations, &routing.Violation{
				Timestamp:     t.Format(time.RFC3339),
				ViolationType: "late",
			})
		}
		// Add final anchor point at 60 minutes
		violations = append(violations, &routing.Violation{
			Timestamp:     bt.Add(60 * time.Minute).Format(time.RFC3339),
			ViolationType: "anchor",
		})

		result := AnalyzeViolationTrend(violations)

		if result.Trend != "improving" {
			t.Errorf("Expected Trend 'improving', got %q", result.Trend)
		}
		totalProcessed := result.EarlyCount + result.LateCount
		if totalProcessed != 101 {
			t.Errorf("Expected 101 violations processed, got %d", totalProcessed)
		}
	})

	t.Run("message_format_stable", func(t *testing.T) {
		t.Parallel()

		violations := []*routing.Violation{
			{Timestamp: "2024-01-15T10:00:00Z", ViolationType: "a"},
			{Timestamp: "2024-01-15T10:10:00Z", ViolationType: "b"},
			{Timestamp: "2024-01-15T10:20:00Z", ViolationType: "c"},
			{Timestamp: "2024-01-15T10:40:00Z", ViolationType: "d"},
			{Timestamp: "2024-01-15T10:50:00Z", ViolationType: "e"},
			{Timestamp: "2024-01-15T11:00:00Z", ViolationType: "f"},
		}

		result := AnalyzeViolationTrend(violations)

		if !containsString(result.Message, "unchanged") {
			t.Errorf("Expected stable message to contain 'unchanged', got %q", result.Message)
		}
	})

	t.Run("only_valid_timestamps_after_filtering", func(t *testing.T) {
		t.Parallel()

		// All but one are invalid, leaving only one valid
		violations := []*routing.Violation{
			{Timestamp: "invalid1", ViolationType: "a"},
			{Timestamp: "2024-01-15T10:00:00Z", ViolationType: "b"},
			{Timestamp: "invalid2", ViolationType: "c"},
		}

		result := AnalyzeViolationTrend(violations)

		// Should be insufficient data after filtering
		if result.Trend != "insufficient_data" {
			t.Errorf("Expected Trend 'insufficient_data' (only 1 valid), got %q", result.Trend)
		}
	})
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
