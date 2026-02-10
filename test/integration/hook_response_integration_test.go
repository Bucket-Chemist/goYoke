package integration

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

// TestHookWorkflow_FullPipeline demonstrates the complete workflow:
// STDIN → Parse Event → Create Response → Marshal JSON → STDOUT
func TestHookWorkflow_FullPipeline(t *testing.T) {
	tests := []struct {
		name           string
		inputJSON      string
		createResponse func(*routing.ToolEvent) *routing.HookResponse
		wantDecision   string
		wantReason     string
		wantFields     map[string]interface{}
	}{
		{
			name: "block_task_opus",
			inputJSON: `{
				"tool_name": "Task",
				"tool_input": {
					"model": "opus",
					"prompt": "AGENT: einstein\n\nAnalyze this",
					"subagent_type": "general-purpose"
				},
				"session_id": "test-123",
				"hook_event_name": "PreToolUse",
				"cwd": "/tmp/test"
			}`,
			createResponse: func(event *routing.ToolEvent) *routing.HookResponse {
				resp := routing.NewBlockResponse(event.HookEventName, "Task(opus) blocked - use /einstein instead")
				resp.AddField("permissionDecision", "deny")
				resp.AddField("toolName", event.ToolName)
				return resp
			},
			wantDecision: "block",
			wantReason:   "Task(opus) blocked - use /einstein instead",
			wantFields: map[string]interface{}{
				"hookEventName":      "PreToolUse",
				"permissionDecision": "deny",
				"toolName":           "Task",
			},
		},
		{
			name: "warn_subagent_mismatch",
			inputJSON: `{
				"tool_name": "Task",
				"tool_input": {
					"model": "haiku",
					"prompt": "AGENT: tech-docs-writer\n\nUpdate README",
					"subagent_type": "Explore"
				},
				"session_id": "test-456",
				"hook_event_name": "PreToolUse",
				"cwd": "/tmp/test"
			}`,
			createResponse: func(event *routing.ToolEvent) *routing.HookResponse {
				resp := routing.NewWarnResponse(event.HookEventName, "Subagent_type mismatch detected")
				resp.AddField("agent", "tech-docs-writer")
				resp.AddField("requested", "Explore")
				resp.AddField("expected", "general-purpose")
				return resp
			},
			wantDecision: "approve",
			wantReason:   "Subagent_type mismatch detected",
			wantFields: map[string]interface{}{
				"hookEventName": "PreToolUse",
				"agent":         "tech-docs-writer",
				"requested":     "Explore",
				"expected":      "general-purpose",
			},
		},
		{
			name: "pass_with_context",
			inputJSON: `{
				"tool_name": "Read",
				"tool_input": {
					"file_path": "/home/user/test.go"
				},
				"session_id": "test-789",
				"hook_event_name": "PreToolUse",
				"cwd": "/tmp/test"
			}`,
			createResponse: func(event *routing.ToolEvent) *routing.HookResponse {
				resp := routing.NewPassResponse(event.HookEventName)
				resp.AddField("additionalContext", "Reading test.go - consider running tests after")
				return resp
			},
			wantDecision: "", // Pass responses have empty decision field
			wantReason:   "",
			wantFields: map[string]interface{}{
				"hookEventName":      "PreToolUse",
				"additionalContext":  "Reading test.go - consider running tests after",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Step 1: Simulate STDIN with input JSON
			stdinReader := strings.NewReader(tt.inputJSON)

			// Step 2: Read and parse event (simulating ReadStdin + ParseToolEvent)
			data, err := routing.ReadStdin(stdinReader, 5*time.Second)
			if err != nil {
				t.Fatalf("Failed to read STDIN: %v", err)
			}

			var event routing.ToolEvent
			if err := json.Unmarshal(data, &event); err != nil {
				t.Fatalf("Failed to parse event: %v", err)
			}

			// Step 3: Create response based on event
			response := tt.createResponse(&event)

			// Step 4: Validate response
			if err := response.Validate(); err != nil {
				t.Fatalf("Response validation failed: %v", err)
			}

			// Step 5: Marshal to output (simulating STDOUT)
			var output bytes.Buffer
			if err := response.Marshal(&output); err != nil {
				t.Fatalf("Failed to marshal response: %v", err)
			}

			// Step 6: Verify output is valid JSON
			var outputJSON map[string]interface{}
			if err := json.Unmarshal(output.Bytes(), &outputJSON); err != nil {
				t.Fatalf("Output is not valid JSON: %v\nOutput: %s", err, output.String())
			}

			// Step 7: Verify response fields
			if decision, ok := outputJSON["decision"].(string); ok {
				if decision != tt.wantDecision {
					t.Errorf("Expected decision %q, got %q", tt.wantDecision, decision)
				}
			}

			if reason, ok := outputJSON["reason"].(string); ok {
				if reason != tt.wantReason {
					t.Errorf("Expected reason %q, got %q", tt.wantReason, reason)
				}
			}

			// Step 8: Verify hookSpecificOutput fields
			hookOutput, ok := outputJSON["hookSpecificOutput"].(map[string]interface{})
			if !ok {
				t.Fatalf("Expected hookSpecificOutput to be object, got %T", outputJSON["hookSpecificOutput"])
			}

			for key, wantValue := range tt.wantFields {
				gotValue, ok := hookOutput[key]
				if !ok {
					t.Errorf("Expected hookSpecificOutput[%q] to exist", key)
					continue
				}
				if gotValue != wantValue {
					t.Errorf("Expected hookSpecificOutput[%q] = %v, got %v", key, wantValue, gotValue)
				}
			}
		})
	}
}

// TestHookWorkflow_ErrorHandling verifies error paths in the pipeline
func TestHookWorkflow_ErrorHandling(t *testing.T) {
	t.Run("invalid_json_input", func(t *testing.T) {
		stdinReader := strings.NewReader(`{"invalid": json}`)
		_, err := routing.ReadStdin(stdinReader, 5*time.Second)
		if err != nil {
			// ReadStdin succeeds, parsing will fail
			t.Logf("Expected parsing to fail later, got read error: %v", err)
		}
	})

	t.Run("validation_failure", func(t *testing.T) {
		// Create response with invalid decision
		response := &routing.HookResponse{
			Decision: "invalid-decision",
			Reason:   "Test",
			HookSpecificOutput: map[string]interface{}{
				"hookEventName": "PreToolUse",
			},
		}

		// Should fail validation
		if err := response.Validate(); err == nil {
			t.Error("Expected validation to fail for invalid decision")
		}
	})

	t.Run("missing_hook_event_name", func(t *testing.T) {
		response := routing.NewBlockResponse("PreToolUse", "Test")
		// Remove hookEventName
		delete(response.HookSpecificOutput, "hookEventName")

		// Should fail validation
		if err := response.Validate(); err == nil {
			t.Error("Expected validation to fail for missing hookEventName")
		}
	})
}

// TestHookWorkflow_RealWorldScenario simulates a realistic validate-routing hook
func TestHookWorkflow_RealWorldScenario(t *testing.T) {
	// Simulate real Claude Code event
	realEventJSON := `{
		"tool_name": "Task",
		"tool_input": {
			"description": "Search for auth files",
			"prompt": "AGENT: codebase-search\n\nFind authentication implementation",
			"subagent_type": "general-purpose",
			"model": "haiku"
		},
		"session_id": "claude-prod-abc123",
		"hook_event_name": "PreToolUse",
		"cwd": "/home/user/project",
		"timestamp": 1705356789
	}`

	// Step 1: Read STDIN
	data, err := routing.ReadStdin(strings.NewReader(realEventJSON), 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to read STDIN: %v", err)
	}

	// Step 2: Parse event
	var event routing.ToolEvent
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("Failed to parse event: %v", err)
	}

	// Step 3: Simulate validation logic
	// (In real hook: check if subagent_type matches agent requirements)
	resp := routing.NewBlockResponse(event.HookEventName,
		"Subagent_type violation: codebase-search requires 'Explore', got 'general-purpose'")
	resp.AddField("permissionDecision", "deny")
	resp.AddField("agent", "codebase-search")
	resp.AddField("requested", "general-purpose")
	resp.AddField("correct", "Explore")

	// Step 4: Validate
	if err := resp.Validate(); err != nil {
		t.Fatalf("Response validation failed: %v", err)
	}

	// Step 5: Marshal to STDOUT (simulated)
	var output bytes.Buffer
	if err := resp.Marshal(&output); err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Step 6: Verify Claude Code will accept this response
	var claudeResponse map[string]interface{}
	if err := json.Unmarshal(output.Bytes(), &claudeResponse); err != nil {
		t.Fatalf("Claude Code would reject this response: invalid JSON: %v", err)
	}

	// Verify required fields for Claude Code
	if _, ok := claudeResponse["decision"]; !ok {
		t.Error("Claude Code expects 'decision' field for blocking responses")
	}
	if _, ok := claudeResponse["reason"]; !ok {
		t.Error("Claude Code expects 'reason' field for blocking responses")
	}
	if hookOutput, ok := claudeResponse["hookSpecificOutput"].(map[string]interface{}); ok {
		if _, ok := hookOutput["hookEventName"]; !ok {
			t.Error("Claude Code expects 'hookEventName' in hookSpecificOutput")
		}
	} else {
		t.Error("Claude Code expects 'hookSpecificOutput' object")
	}

	t.Logf("✓ Response would be accepted by Claude Code:\n%s", output.String())
}

// BenchmarkHookResponsePipeline measures full pipeline performance
func BenchmarkHookResponsePipeline(b *testing.B) {
	inputJSON := `{
		"tool_name": "Task",
		"tool_input": {"model": "haiku", "prompt": "test"},
		"session_id": "bench-123",
		"hook_event_name": "PreToolUse",
		"cwd": "/tmp"
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Read
		data, _ := routing.ReadStdin(strings.NewReader(inputJSON), 5*time.Second)

		// Parse
		var event routing.ToolEvent
		json.Unmarshal(data, &event)

		// Create response
		resp := routing.NewBlockResponse(event.HookEventName, "Test block")
		resp.AddField("test", "value")

		// Validate
		resp.Validate()

		// Marshal
		var output bytes.Buffer
		resp.Marshal(&output)
	}
}

func TestMain(m *testing.M) {
	// Setup: Ensure we're running from project root or can find files
	// (No setup needed for this package)

	code := m.Run()

	// Teardown: Clean up any test artifacts
	// Clean up skill-guard test binary if it was created
	if skillGuardBinary != "" {
		os.Remove(skillGuardBinary)
	}

	os.Exit(code)
}
