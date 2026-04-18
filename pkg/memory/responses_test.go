package memory

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/Bucket-Chemist/goYoke/pkg/routing"
)

// TestGenerateBlockingResponse_WithMatches verifies enhanced response includes pattern matches
func TestGenerateBlockingResponse_WithMatches(t *testing.T) {
	// Build test index with sample templates
	index := &SharpEdgeIndex{
		ByErrorType: map[string][]SharpEdgeTemplate{
			"TypeError": {
				{
					ID:          "go-pro-042",
					ErrorType:   "TypeError",
					FilePattern: "pkg/routing/*.go",
					Keywords:    []string{"type assertion", "bool"},
					Description: "Type assertion on already-typed field in routing validation",
					Solution:    "Field AgentSubagentMapping is map[string]string, not interface{}. Use direct map access: value := AgentSubagentMapping[key]. Remove type assertion.",
					Source:      "/home/user/.claude/agents/go-pro/sharp-edges.yaml",
				},
			},
		},
		ByKeyword: map[string][]SharpEdgeTemplate{
			"type assertion": {
				{
					ID:          "go-pro-042",
					ErrorType:   "TypeError",
					FilePattern: "pkg/routing/*.go",
					Keywords:    []string{"type assertion", "bool"},
					Description: "Type assertion on already-typed field in routing validation",
					Solution:    "Field AgentSubagentMapping is map[string]string, not interface{}. Use direct map access: value := AgentSubagentMapping[key]. Remove type assertion.",
					Source:      "/home/user/.claude/agents/go-pro/sharp-edges.yaml",
				},
			},
		},
		All: []SharpEdgeTemplate{
			{
				ID:          "go-pro-042",
				ErrorType:   "TypeError",
				FilePattern: "pkg/routing/*.go",
				Keywords:    []string{"type assertion", "bool"},
				Description: "Type assertion on already-typed field in routing validation",
				Solution:    "Field AgentSubagentMapping is map[string]string, not interface{}. Use direct map access: value := AgentSubagentMapping[key]. Remove type assertion.",
				Source:      "/home/user/.claude/agents/go-pro/sharp-edges.yaml",
			},
		},
	}

	// Build test edge that should match
	edge := &SharpEdge{
		File:         "pkg/routing/task_validation.go",
		ErrorType:    "TypeError",
		ErrorMessage: "invalid type assertion: field is bool, not interface{}",
	}

	// Generate response
	resp := GenerateBlockingResponse(edge, index, 3)

	// Verify response structure
	if resp == nil {
		t.Fatal("Expected non-nil response")
	}

	if resp.Decision != routing.DecisionBlock {
		t.Errorf("Expected decision='block', got %q", resp.Decision)
	}

	expectedReason := "⚠️ SHARP EDGE DETECTED: 3 consecutive failures on 'pkg/routing/task_validation.go' (TypeError)"
	if resp.Reason != expectedReason {
		t.Errorf("Expected reason %q, got %q", expectedReason, resp.Reason)
	}

	// Verify hookSpecificOutput
	if resp.HookSpecificOutput == nil {
		t.Fatal("Expected non-nil hookSpecificOutput")
	}

	hookEventName, ok := resp.HookSpecificOutput["hookEventName"].(string)
	if !ok || hookEventName != "PostToolUse" {
		t.Errorf("Expected hookEventName='PostToolUse', got %v", resp.HookSpecificOutput["hookEventName"])
	}

	additionalContext, ok := resp.HookSpecificOutput["additionalContext"].(string)
	if !ok {
		t.Fatal("Expected additionalContext to be string")
	}

	// Verify base message content
	if !strings.Contains(additionalContext, "DEBUGGING LOOP DETECTED") {
		t.Error("Expected base debugging loop message")
	}

	if !strings.Contains(additionalContext, "3 failures on pkg/routing/task_validation.go") {
		t.Error("Expected failure count and file in message")
	}

	// Verify pattern matches are included
	if !strings.Contains(additionalContext, "SIMILAR SHARP EDGES FOUND") {
		t.Error("Expected pattern matches section")
	}

	if !strings.Contains(additionalContext, "go-pro-042") {
		t.Error("Expected matched pattern ID in response")
	}

	if !strings.Contains(additionalContext, "Type assertion on already-typed field") {
		t.Error("Expected pattern description in response")
	}

	if !strings.Contains(additionalContext, "Use direct map access") {
		t.Error("Expected solution text in response")
	}

	if !strings.Contains(additionalContext, "/home/user/.claude/agents/go-pro/sharp-edges.yaml") {
		t.Error("Expected source reference in response")
	}

	if !strings.Contains(additionalContext, "Try the suggested solution from the highest-scored match") {
		t.Error("Expected guidance to use highest match")
	}

	// Verify JSON serialization works
	if err := resp.Validate(); err != nil {
		t.Errorf("Response validation failed: %v", err)
	}
}

// TestGenerateBlockingResponse_NoMatches verifies response when no patterns match
func TestGenerateBlockingResponse_NoMatches(t *testing.T) {
	// Empty index
	index := &SharpEdgeIndex{
		ByErrorType: make(map[string][]SharpEdgeTemplate),
		ByKeyword:   make(map[string][]SharpEdgeTemplate),
		All:         []SharpEdgeTemplate{},
	}

	edge := &SharpEdge{
		File:         "pkg/unknown/file.go",
		ErrorType:    "UnknownError",
		ErrorMessage: "something went wrong",
	}

	resp := GenerateBlockingResponse(edge, index, 5)

	// Verify response structure
	if resp == nil {
		t.Fatal("Expected non-nil response")
	}

	if resp.Decision != routing.DecisionBlock {
		t.Errorf("Expected decision='block', got %q", resp.Decision)
	}

	additionalContext, ok := resp.HookSpecificOutput["additionalContext"].(string)
	if !ok {
		t.Fatal("Expected additionalContext to be string")
	}

	// Verify base message present
	if !strings.Contains(additionalContext, "DEBUGGING LOOP DETECTED") {
		t.Error("Expected base debugging loop message")
	}

	// Verify no matches message
	if !strings.Contains(additionalContext, "No similar patterns found in sharp-edges.yaml") {
		t.Error("Expected no matches message")
	}

	if strings.Contains(additionalContext, "SIMILAR SHARP EDGES FOUND") {
		t.Error("Should not show matches section when no matches")
	}

	// Verify JSON serialization works
	if err := resp.Validate(); err != nil {
		t.Errorf("Response validation failed: %v", err)
	}
}

// TestGenerateBlockingResponse_MultipleMatches verifies multiple pattern matches are shown
func TestGenerateBlockingResponse_MultipleMatches(t *testing.T) {
	// Build index with multiple matching templates
	index := &SharpEdgeIndex{
		ByErrorType: map[string][]SharpEdgeTemplate{
			"TypeError": {
				{
					ID:          "match-1",
					ErrorType:   "TypeError",
					FilePattern: "pkg/*.go",
					Keywords:    []string{"assertion"},
					Description: "First match description",
					Solution:    "First solution",
					Source:      "/path/to/first.yaml",
				},
				{
					ID:          "match-2",
					ErrorType:   "TypeError",
					FilePattern: "pkg/routing/*.go",
					Keywords:    []string{"validation"},
					Description: "Second match description",
					Solution:    "Second solution",
					Source:      "/path/to/second.yaml",
				},
			},
		},
		ByKeyword:   make(map[string][]SharpEdgeTemplate),
		All:         []SharpEdgeTemplate{},
	}

	edge := &SharpEdge{
		File:         "pkg/routing/validation.go",
		ErrorType:    "TypeError",
		ErrorMessage: "validation assertion failed",
	}

	resp := GenerateBlockingResponse(edge, index, 2)

	additionalContext, ok := resp.HookSpecificOutput["additionalContext"].(string)
	if !ok {
		t.Fatal("Expected additionalContext to be string")
	}

	// Verify both matches are shown
	if !strings.Contains(additionalContext, "match-1") {
		t.Error("Expected first match ID")
	}

	if !strings.Contains(additionalContext, "match-2") {
		t.Error("Expected second match ID")
	}

	if !strings.Contains(additionalContext, "Match 1") {
		t.Error("Expected numbered match labels")
	}

	if !strings.Contains(additionalContext, "score:") {
		t.Error("Expected match scores")
	}

	if !strings.Contains(additionalContext, "matched on:") {
		t.Error("Expected matched signals")
	}
}

// TestGenerateBlockingResponse_NilEdge verifies graceful handling of nil edge
func TestGenerateBlockingResponse_NilEdge(t *testing.T) {
	index := &SharpEdgeIndex{
		ByErrorType: make(map[string][]SharpEdgeTemplate),
		ByKeyword:   make(map[string][]SharpEdgeTemplate),
		All:         []SharpEdgeTemplate{},
	}

	resp := GenerateBlockingResponse(nil, index, 3)

	if resp == nil {
		t.Fatal("Expected non-nil response even with nil edge")
	}

	if resp.Decision != routing.DecisionBlock {
		t.Errorf("Expected decision='block', got %q", resp.Decision)
	}

	expectedReason := "⚠️ SHARP EDGE DETECTED: 3 consecutive failures on 'unknown' (unknown)"
	if resp.Reason != expectedReason {
		t.Errorf("Expected reason with 'unknown' fields, got %q", resp.Reason)
	}

	additionalContext, ok := resp.HookSpecificOutput["additionalContext"].(string)
	if !ok {
		t.Fatal("Expected additionalContext to be string")
	}

	// Verify base message still present
	if !strings.Contains(additionalContext, "DEBUGGING LOOP DETECTED") {
		t.Error("Expected base message even with nil edge")
	}

	if !strings.Contains(additionalContext, "3 failures on unknown") {
		t.Error("Expected 'unknown' in message")
	}
}

// TestGenerateBlockingResponse_NilIndex verifies graceful handling of nil index
func TestGenerateBlockingResponse_NilIndex(t *testing.T) {
	edge := &SharpEdge{
		File:         "pkg/test.go",
		ErrorType:    "TestError",
		ErrorMessage: "test error message",
	}

	resp := GenerateBlockingResponse(edge, nil, 4)

	if resp == nil {
		t.Fatal("Expected non-nil response even with nil index")
	}

	additionalContext, ok := resp.HookSpecificOutput["additionalContext"].(string)
	if !ok {
		t.Fatal("Expected additionalContext to be string")
	}

	// Should show no matches message
	if !strings.Contains(additionalContext, "No similar patterns found") {
		t.Error("Expected no matches message with nil index")
	}

	if strings.Contains(additionalContext, "SIMILAR SHARP EDGES FOUND") {
		t.Error("Should not show matches section with nil index")
	}
}

// TestGenerateBlockingResponse_JSONSerialization verifies response can be marshaled to JSON
func TestGenerateBlockingResponse_JSONSerialization(t *testing.T) {
	index := &SharpEdgeIndex{
		ByErrorType: make(map[string][]SharpEdgeTemplate),
		ByKeyword:   make(map[string][]SharpEdgeTemplate),
		All:         []SharpEdgeTemplate{},
	}

	edge := &SharpEdge{
		File:         "test.go",
		ErrorType:    "TestError",
		ErrorMessage: "test message",
	}

	resp := GenerateBlockingResponse(edge, index, 3)

	// Marshal to JSON
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	// Unmarshal back
	var unmarshaled map[string]interface{}
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify structure
	if decision, ok := unmarshaled["decision"].(string); !ok || decision != "block" {
		t.Errorf("Expected decision='block' in JSON, got %v", unmarshaled["decision"])
	}

	if reason, ok := unmarshaled["reason"].(string); !ok || reason == "" {
		t.Error("Expected non-empty reason in JSON")
	}

	hookOutput, ok := unmarshaled["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected hookSpecificOutput in JSON")
	}

	if eventName, ok := hookOutput["hookEventName"].(string); !ok || eventName != "PostToolUse" {
		t.Error("Expected hookEventName='PostToolUse' in JSON")
	}

	if additionalContext, ok := hookOutput["additionalContext"].(string); !ok || additionalContext == "" {
		t.Error("Expected non-empty additionalContext in JSON")
	}
}

// TestGenerateBlockingResponse_FailureCountInMessage verifies failure count appears in message
func TestGenerateBlockingResponse_FailureCountInMessage(t *testing.T) {
	index := &SharpEdgeIndex{
		ByErrorType: make(map[string][]SharpEdgeTemplate),
		ByKeyword:   make(map[string][]SharpEdgeTemplate),
		All:         []SharpEdgeTemplate{},
	}

	tests := []struct {
		name         string
		failureCount int
	}{
		{"threshold", 3},
		{"above threshold", 5},
		{"high count", 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edge := &SharpEdge{
				File:         "test.go",
				ErrorType:    "TestError",
				ErrorMessage: "test",
			}

			resp := GenerateBlockingResponse(edge, index, tt.failureCount)

			additionalContext := resp.HookSpecificOutput["additionalContext"].(string)

			// Verify count appears in base message
			expectedMsg := fmt.Sprintf("%d failures on test.go", tt.failureCount)
			if !strings.Contains(additionalContext, expectedMsg) {
				t.Errorf("Expected '%s' in message, but message was: %s", expectedMsg, additionalContext)
			}

			// Verify count in reason
			expectedReason := fmt.Sprintf("%d consecutive failures on 'test.go'", tt.failureCount)
			if !strings.Contains(resp.Reason, expectedReason) {
				t.Errorf("Expected '%s' in reason, but reason was: %s", expectedReason, resp.Reason)
			}
		})
	}
}

// TestGenerateBlockingResponse_MatchScoreAndSignals verifies match details are shown
func TestGenerateBlockingResponse_MatchScoreAndSignals(t *testing.T) {
	index := &SharpEdgeIndex{
		ByErrorType: map[string][]SharpEdgeTemplate{
			"TypeError": {
				{
					ID:          "test-match",
					ErrorType:   "TypeError",
					FilePattern: "*.go",
					Keywords:    []string{"assertion", "failed"},
					Description: "Test match",
					Solution:    "Test solution",
					Source:      "/test.yaml",
				},
			},
		},
		ByKeyword:   make(map[string][]SharpEdgeTemplate),
		All:         []SharpEdgeTemplate{},
	}

	edge := &SharpEdge{
		File:         "test.go",
		ErrorType:    "TypeError",
		ErrorMessage: "assertion failed",
	}

	resp := GenerateBlockingResponse(edge, index, 3)

	additionalContext := resp.HookSpecificOutput["additionalContext"].(string)

	// Verify match details format
	if !strings.Contains(additionalContext, "Match 1") {
		t.Error("Expected match number label")
	}

	if !strings.Contains(additionalContext, "score:") {
		t.Error("Expected score field")
	}

	if !strings.Contains(additionalContext, "matched on:") {
		t.Error("Expected matched signals field")
	}

	if !strings.Contains(additionalContext, "**ID**: test-match") {
		t.Error("Expected ID field")
	}

	if !strings.Contains(additionalContext, "**Description**: Test match") {
		t.Error("Expected description field")
	}

	if !strings.Contains(additionalContext, "**Suggested Solution**: Test solution") {
		t.Error("Expected solution field")
	}

	if !strings.Contains(additionalContext, "**Source**: /test.yaml") {
		t.Error("Expected source field")
	}
}
