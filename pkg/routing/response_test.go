package routing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

// Test NewBlockResponse constructor
func TestNewBlockResponse(t *testing.T) {
	resp := NewBlockResponse("PreToolUse", "Tool not allowed in this context")

	if resp.Decision != DecisionBlock {
		t.Errorf("expected decision %q, got %q", DecisionBlock, resp.Decision)
	}

	if resp.Reason != "Tool not allowed in this context" {
		t.Errorf("expected reason %q, got %q", "Tool not allowed in this context", resp.Reason)
	}

	if resp.HookSpecificOutput == nil {
		t.Fatal("expected hookSpecificOutput to be non-nil")
	}

	hookEventName, ok := resp.HookSpecificOutput["hookEventName"]
	if !ok {
		t.Error("expected hookEventName in hookSpecificOutput")
	}

	if hookEventName != "PreToolUse" {
		t.Errorf("expected hookEventName %q, got %q", "PreToolUse", hookEventName)
	}
}

// Test NewWarnResponse constructor
func TestNewWarnResponse(t *testing.T) {
	resp := NewWarnResponse("PostToolUse", "Tool usage exceeds rate limit")

	if resp.Decision != DecisionWarn {
		t.Errorf("expected decision %q, got %q", DecisionWarn, resp.Decision)
	}

	if resp.Reason != "Tool usage exceeds rate limit" {
		t.Errorf("expected reason %q, got %q", "Tool usage exceeds rate limit", resp.Reason)
	}

	if resp.HookSpecificOutput == nil {
		t.Fatal("expected hookSpecificOutput to be non-nil")
	}

	hookEventName, ok := resp.HookSpecificOutput["hookEventName"]
	if !ok {
		t.Error("expected hookEventName in hookSpecificOutput")
	}

	if hookEventName != "PostToolUse" {
		t.Errorf("expected hookEventName %q, got %q", "PostToolUse", hookEventName)
	}
}

// Test NewPassResponse constructor
func TestNewPassResponse(t *testing.T) {
	resp := NewPassResponse("PreToolUse")

	if resp.Decision != "" {
		t.Errorf("expected empty decision for pass response, got %q", resp.Decision)
	}

	if resp.Reason != "" {
		t.Errorf("expected empty reason for pass response, got %q", resp.Reason)
	}

	if resp.HookSpecificOutput == nil {
		t.Fatal("expected hookSpecificOutput to be non-nil")
	}

	hookEventName, ok := resp.HookSpecificOutput["hookEventName"]
	if !ok {
		t.Error("expected hookEventName in hookSpecificOutput")
	}

	if hookEventName != "PreToolUse" {
		t.Errorf("expected hookEventName %q, got %q", "PreToolUse", hookEventName)
	}
}

// Test AddField method
func TestAddField(t *testing.T) {
	tests := []struct {
		name  string
		resp  *HookResponse
		key   string
		value interface{}
	}{
		{
			name:  "add string field",
			resp:  NewBlockResponse("PreToolUse", "blocked"),
			key:   "additionalContext",
			value: "Some context",
		},
		{
			name:  "add int field",
			resp:  NewWarnResponse("PostToolUse", "warning"),
			key:   "attemptCount",
			value: 3,
		},
		{
			name:  "add map field",
			resp:  NewPassResponse("PreToolUse"),
			key:   "metadata",
			value: map[string]string{"foo": "bar"},
		},
		{
			name:  "add field to nil hookSpecificOutput",
			resp:  &HookResponse{Decision: DecisionBlock, Reason: "test"},
			key:   "testField",
			value: "testValue",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.resp.AddField(tc.key, tc.value)

			if tc.resp.HookSpecificOutput == nil {
				t.Fatal("expected hookSpecificOutput to be non-nil after AddField")
			}

			val, ok := tc.resp.HookSpecificOutput[tc.key]
			if !ok {
				t.Errorf("expected field %q to be present", tc.key)
			}

			// Compare values (type-specific comparison)
			switch expected := tc.value.(type) {
			case string:
				if val != expected {
					t.Errorf("expected value %q, got %v", expected, val)
				}
			case int:
				if val != expected {
					t.Errorf("expected value %d, got %v", expected, val)
				}
			case map[string]string:
				valMap, ok := val.(map[string]string)
				if !ok {
					t.Errorf("expected map[string]string, got %T", val)
				}
				if valMap["foo"] != expected["foo"] {
					t.Errorf("expected map value %q, got %q", expected["foo"], valMap["foo"])
				}
			}
		})
	}
}

// Test Validate with valid responses
func TestValidate_ValidCases(t *testing.T) {
	tests := []struct {
		name string
		resp *HookResponse
	}{
		{
			name: "valid block response",
			resp: NewBlockResponse("PreToolUse", "Tool blocked"),
		},
		{
			name: "valid warn response",
			resp: NewWarnResponse("PostToolUse", "Warning message"),
		},
		{
			name: "valid pass response",
			resp: NewPassResponse("PreToolUse"),
		},
		{
			name: "block with additional fields",
			resp: func() *HookResponse {
				r := NewBlockResponse("PreToolUse", "Blocked")
				r.AddField("additionalContext", "More info")
				r.AddField("permissionDecision", "deny")
				return r
			}(),
		},
		{
			name: "warn with complex hookSpecificOutput",
			resp: func() *HookResponse {
				r := NewWarnResponse("PostToolUse", "Warning")
				r.AddField("metadata", map[string]interface{}{
					"attempts": 3,
					"lastError": "timeout",
				})
				return r
			}(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.resp.Validate()
			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

// Test Validate with invalid responses
func TestValidate_InvalidCases(t *testing.T) {
	tests := []struct {
		name          string
		resp          *HookResponse
		expectedError string
	}{
		{
			name: "invalid decision value",
			resp: &HookResponse{
				Decision: "invalid",
				Reason:   "Some reason",
				HookSpecificOutput: map[string]interface{}{
					"hookEventName": "PreToolUse",
				},
			},
			expectedError: "Invalid decision value",
		},
		{
			name: "block without reason",
			resp: &HookResponse{
				Decision: DecisionBlock,
				Reason:   "",
				HookSpecificOutput: map[string]interface{}{
					"hookEventName": "PreToolUse",
				},
			},
			expectedError: "requires non-empty reason field",
		},
		{
			name: "warn without reason",
			resp: &HookResponse{
				Decision: DecisionWarn,
				Reason:   "",
				HookSpecificOutput: map[string]interface{}{
					"hookEventName": "PostToolUse",
				},
			},
			expectedError: "requires non-empty reason field",
		},
		{
			name: "missing hookSpecificOutput",
			resp: &HookResponse{
				Decision:           DecisionBlock,
				Reason:             "Blocked",
				HookSpecificOutput: nil,
			},
			expectedError: "Missing hookSpecificOutput",
		},
		{
			name: "missing hookEventName",
			resp: &HookResponse{
				Decision: DecisionBlock,
				Reason:   "Blocked",
				HookSpecificOutput: map[string]interface{}{
					"otherField": "value",
				},
			},
			expectedError: "Missing hookEventName in hookSpecificOutput",
		},
		{
			name: "empty hookEventName string",
			resp: &HookResponse{
				Decision: DecisionBlock,
				Reason:   "Blocked",
				HookSpecificOutput: map[string]interface{}{
					"hookEventName": "",
				},
			},
			expectedError: "hookEventName must be a non-empty string",
		},
		{
			name: "hookEventName wrong type",
			resp: &HookResponse{
				Decision: DecisionBlock,
				Reason:   "Blocked",
				HookSpecificOutput: map[string]interface{}{
					"hookEventName": 123,
				},
			},
			expectedError: "hookEventName must be a non-empty string",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.resp.Validate()
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !strings.Contains(err.Error(), tc.expectedError) {
				t.Errorf("expected error containing %q, got: %v", tc.expectedError, err)
			}

			if !strings.Contains(err.Error(), "[hook-response]") {
				t.Errorf("error should have [hook-response] prefix, got: %v", err)
			}
		})
	}
}

// Test Marshal output format
func TestMarshal_OutputFormat(t *testing.T) {
	tests := []struct {
		name             string
		resp             *HookResponse
		expectedDecision string
		expectedReason   string
	}{
		{
			name:             "block response",
			resp:             NewBlockResponse("PreToolUse", "Tool blocked"),
			expectedDecision: "block",
			expectedReason:   "Tool blocked",
		},
		{
			name:             "warn response (maps to approve per Claude Code schema)",
			resp:             NewWarnResponse("PostToolUse", "Warning message"),
			expectedDecision: "approve", // Claude Code only supports approve|block
			expectedReason:   "Warning message",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tc.resp.Marshal(&buf)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}

			// Parse JSON output
			var parsed map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
				t.Fatalf("failed to parse marshaled JSON: %v", err)
			}

			// Verify decision
			decision, ok := parsed["decision"]
			if !ok {
				t.Error("expected 'decision' field in output")
			}
			if decision != tc.expectedDecision {
				t.Errorf("expected decision %q, got %q", tc.expectedDecision, decision)
			}

			// Verify reason
			reason, ok := parsed["reason"]
			if !ok {
				t.Error("expected 'reason' field in output")
			}
			if reason != tc.expectedReason {
				t.Errorf("expected reason %q, got %q", tc.expectedReason, reason)
			}

			// Verify hookSpecificOutput
			hookSpecific, ok := parsed["hookSpecificOutput"]
			if !ok {
				t.Error("expected 'hookSpecificOutput' field in output")
			}

			hookSpecificMap, ok := hookSpecific.(map[string]interface{})
			if !ok {
				t.Fatalf("expected hookSpecificOutput to be a map, got %T", hookSpecific)
			}

			hookEventName, ok := hookSpecificMap["hookEventName"]
			if !ok {
				t.Error("expected 'hookEventName' in hookSpecificOutput")
			}

			if hookEventName == "" {
				t.Error("expected non-empty hookEventName")
			}
		})
	}
}

// Test Marshal with pass response (no decision/reason)
func TestMarshal_PassResponse(t *testing.T) {
	resp := NewPassResponse("PreToolUse")
	resp.AddField("additionalContext", "Some context")

	var buf bytes.Buffer
	err := resp.Marshal(&buf)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Parse JSON output
	var parsed map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to parse marshaled JSON: %v", err)
	}

	// decision and reason should be empty strings (default JSON marshaling)
	decision, ok := parsed["decision"]
	if ok && decision != "" {
		t.Errorf("expected empty or missing decision, got %q", decision)
	}

	reason, ok := parsed["reason"]
	if ok && reason != "" {
		t.Errorf("expected empty or missing reason, got %q", reason)
	}

	// hookSpecificOutput should be present
	hookSpecific, ok := parsed["hookSpecificOutput"]
	if !ok {
		t.Fatal("expected hookSpecificOutput field in output")
	}

	hookSpecificMap := hookSpecific.(map[string]interface{})
	if hookSpecificMap["additionalContext"] != "Some context" {
		t.Errorf("expected additionalContext %q, got %v", "Some context", hookSpecificMap["additionalContext"])
	}
}

// Test Marshal with invalid response
func TestMarshal_InvalidResponse(t *testing.T) {
	resp := &HookResponse{
		Decision: "invalid",
		Reason:   "test",
	}

	var buf bytes.Buffer
	err := resp.Marshal(&buf)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	if !strings.Contains(err.Error(), "Invalid decision value") {
		t.Errorf("expected validation error, got: %v", err)
	}
}

// Test Marshal with flexible hookSpecificOutput
func TestMarshal_FlexibleHookSpecificOutput(t *testing.T) {
	tests := []struct {
		name   string
		resp   *HookResponse
		fields map[string]interface{}
	}{
		{
			name: "permission decision fields",
			resp: NewBlockResponse("PreToolUse", "Permission denied"),
			fields: map[string]interface{}{
				"permissionDecision":       "deny",
				"permissionDecisionReason": "Tool not allowed",
			},
		},
		{
			name: "additional context",
			resp: NewWarnResponse("PostToolUse", "Rate limit"),
			fields: map[string]interface{}{
				"additionalContext": "You have made 100 requests",
			},
		},
		{
			name: "complex nested structure",
			resp: NewBlockResponse("PreToolUse", "Validation failed"),
			fields: map[string]interface{}{
				"validationErrors": map[string]interface{}{
					"field1": "error1",
					"field2": "error2",
				},
				"timestamp": time.Now().Unix(),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Add fields
			for key, value := range tc.fields {
				tc.resp.AddField(key, value)
			}

			var buf bytes.Buffer
			err := tc.resp.Marshal(&buf)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}

			// Parse JSON output
			var parsed map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
				t.Fatalf("failed to parse marshaled JSON: %v", err)
			}

			hookSpecific := parsed["hookSpecificOutput"].(map[string]interface{})

			// Verify all fields present
			for key := range tc.fields {
				if _, ok := hookSpecific[key]; !ok {
					t.Errorf("expected field %q in hookSpecificOutput", key)
				}
			}
		})
	}
}

// Integration test: Parse event → Create response → Marshal → Verify JSON
func TestIntegration_EventToResponse(t *testing.T) {
	// 1. Parse a real event (simulating PreToolUse hook)
	eventJSON := `{
		"tool_name": "Task",
		"tool_input": {
			"model": "opus",
			"prompt": "AGENT: einstein\n\nDeep analysis",
			"subagent_type": "general-purpose",
			"description": "Opus task"
		},
		"session_id": "test-session",
		"hook_event_name": "PreToolUse",
		"captured_at": 1768465022
	}`

	reader := strings.NewReader(eventJSON)
	event, err := ParseToolEvent(reader, 1*time.Second)
	if err != nil {
		t.Fatalf("failed to parse event: %v", err)
	}

	// 2. Create a block response based on the event
	resp := NewBlockResponse(event.HookEventName, "Task(opus) invocation blocked. Use /einstein instead.")
	resp.AddField("toolName", event.ToolName)
	resp.AddField("sessionID", event.SessionID)
	resp.AddField("permissionDecision", "deny")
	resp.AddField("permissionDecisionReason", "Einstein must use GAP document protocol")

	// 3. Marshal to JSON
	var buf bytes.Buffer
	err = resp.Marshal(&buf)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	// 4. Verify the marshaled JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to parse marshaled JSON: %v", err)
	}

	// Verify decision
	if parsed["decision"] != DecisionBlock {
		t.Errorf("expected decision %q, got %v", DecisionBlock, parsed["decision"])
	}

	// Verify reason
	expectedReason := "Task(opus) invocation blocked. Use /einstein instead."
	if parsed["reason"] != expectedReason {
		t.Errorf("expected reason %q, got %v", expectedReason, parsed["reason"])
	}

	// Verify hookSpecificOutput
	hookSpecific := parsed["hookSpecificOutput"].(map[string]interface{})

	if hookSpecific["hookEventName"] != "PreToolUse" {
		t.Errorf("expected hookEventName %q, got %v", "PreToolUse", hookSpecific["hookEventName"])
	}

	if hookSpecific["toolName"] != "Task" {
		t.Errorf("expected toolName %q, got %v", "Task", hookSpecific["toolName"])
	}

	if hookSpecific["permissionDecision"] != "deny" {
		t.Errorf("expected permissionDecision %q, got %v", "deny", hookSpecific["permissionDecision"])
	}

	if hookSpecific["sessionID"] != "test-session" {
		t.Errorf("expected sessionID %q, got %v", "test-session", hookSpecific["sessionID"])
	}

	t.Logf("Integration test complete. Marshaled JSON:\n%s", buf.String())
}

// Test decision constants
// Note: DecisionWarn and DecisionPass are legacy aliases that map to "approve"
// because Claude Code schema only supports "approve" | "block"
func TestDecisionConstants(t *testing.T) {
	if DecisionBlock != "block" {
		t.Errorf("expected DecisionBlock constant to be %q, got %q", "block", DecisionBlock)
	}

	if DecisionApprove != "approve" {
		t.Errorf("expected DecisionApprove constant to be %q, got %q", "approve", DecisionApprove)
	}

	// Legacy aliases - both map to "approve" per Claude Code schema
	if DecisionWarn != "approve" {
		t.Errorf("expected DecisionWarn (legacy) to map to %q, got %q", "approve", DecisionWarn)
	}

	if DecisionPass != "approve" {
		t.Errorf("expected DecisionPass (legacy) to map to %q, got %q", "approve", DecisionPass)
	}
}

// Test error message format compliance for responses
func TestResponseErrorMessageFormat(t *testing.T) {
	tests := []struct {
		name          string
		resp          *HookResponse
		expectedParts []string
	}{
		{
			name: "invalid decision",
			resp: &HookResponse{
				Decision: "invalid",
				HookSpecificOutput: map[string]interface{}{
					"hookEventName": "PreToolUse",
				},
			},
			expectedParts: []string{"[hook-response]", "Invalid decision value", "DecisionApprove or DecisionBlock"},
		},
		{
			name: "missing reason for block",
			resp: &HookResponse{
				Decision: DecisionBlock,
				HookSpecificOutput: map[string]interface{}{
					"hookEventName": "PreToolUse",
				},
			},
			expectedParts: []string{"[hook-response]", "requires non-empty reason field", "Provide context"},
		},
		{
			name: "missing hookEventName",
			resp: &HookResponse{
				Decision: DecisionBlock,
				Reason:   "test",
				HookSpecificOutput: map[string]interface{}{
					"otherField": "value",
				},
			},
			expectedParts: []string{"[hook-response]", "Missing hookEventName", "must identify the triggering event"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.resp.Validate()
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			errMsg := err.Error()
			for _, part := range tc.expectedParts {
				if !strings.Contains(errMsg, part) {
					t.Errorf("error message should contain %q, got: %v", part, errMsg)
				}
			}
		})
	}
}

// Edge case: Empty hookSpecificOutput map (still valid if hookEventName present)
func TestValidate_EmptyHookSpecificOutputExceptHookEventName(t *testing.T) {
	resp := &HookResponse{
		Decision: DecisionBlock,
		Reason:   "Blocked",
		HookSpecificOutput: map[string]interface{}{
			"hookEventName": "PreToolUse",
		},
	}

	err := resp.Validate()
	if err != nil {
		t.Errorf("expected no error for minimal valid response, got: %v", err)
	}
}

// Edge case: Multiple AddField calls
func TestAddField_MultipleCalls(t *testing.T) {
	resp := NewBlockResponse("PreToolUse", "Blocked")

	resp.AddField("field1", "value1")
	resp.AddField("field2", 42)
	resp.AddField("field3", true)

	if len(resp.HookSpecificOutput) != 4 { // hookEventName + 3 fields
		t.Errorf("expected 4 fields in hookSpecificOutput, got %d", len(resp.HookSpecificOutput))
	}

	if resp.HookSpecificOutput["field1"] != "value1" {
		t.Errorf("expected field1 %q, got %v", "value1", resp.HookSpecificOutput["field1"])
	}

	if resp.HookSpecificOutput["field2"] != 42 {
		t.Errorf("expected field2 %d, got %v", 42, resp.HookSpecificOutput["field2"])
	}

	if resp.HookSpecificOutput["field3"] != true {
		t.Errorf("expected field3 %v, got %v", true, resp.HookSpecificOutput["field3"])
	}
}

// Edge case: Overwriting existing field with AddField
func TestAddField_Overwrite(t *testing.T) {
	resp := NewBlockResponse("PreToolUse", "Blocked")
	resp.AddField("testField", "original")

	if resp.HookSpecificOutput["testField"] != "original" {
		t.Errorf("expected testField %q, got %v", "original", resp.HookSpecificOutput["testField"])
	}

	resp.AddField("testField", "updated")

	if resp.HookSpecificOutput["testField"] != "updated" {
		t.Errorf("expected testField %q after overwrite, got %v", "updated", resp.HookSpecificOutput["testField"])
	}
}

// Edge case: Marshal with write error
func TestMarshal_WriteError(t *testing.T) {
	resp := NewBlockResponse("PreToolUse", "Blocked")

	// Use a writer that always fails
	writer := &failingWriter{}
	err := resp.Marshal(writer)

	if err == nil {
		t.Fatal("expected write error, got nil")
	}

	if !strings.Contains(err.Error(), "[hook-response]") {
		t.Errorf("error should have [hook-response] prefix, got: %v", err)
	}

	if !strings.Contains(err.Error(), "Failed to write JSON output") {
		t.Errorf("expected write error message, got: %v", err)
	}
}

// Test Marshal with non-serializable value
func TestMarshal_NonSerializableValue(t *testing.T) {
	resp := NewBlockResponse("PreToolUse", "Blocked")

	// Add a channel (non-serializable in JSON)
	ch := make(chan int)
	resp.AddField("channel", ch)

	var buf bytes.Buffer
	err := resp.Marshal(&buf)

	if err == nil {
		t.Fatal("expected JSON marshal error for channel, got nil")
	}

	if !strings.Contains(err.Error(), "[hook-response]") {
		t.Errorf("error should have [hook-response] prefix, got: %v", err)
	}

	if !strings.Contains(err.Error(), "Failed to marshal JSON") {
		t.Errorf("expected marshal error message, got: %v", err)
	}
}

// failingWriter always returns an error on Write
type failingWriter struct{}

func (fw *failingWriter) Write(p []byte) (n int, err error) {
	return 0, fmt.Errorf("write failed")
}
