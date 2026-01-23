---
id: GOgent-061
title: Session Context Response Generator
description: **Task**:
status: pending
time_estimate: 1.5h
dependencies: [\n  - GOgent-056
  - to
  - 060]
priority: HIGH
week: 4
tags:
  - session-start
  - week-4
tests_required: true
acceptance_criteria_count: 14
---

## GOgent-061: Session Context Response Generator

**Time**: 1.5 hours
**Dependencies**: GOgent-056 to 060
**Priority**: HIGH

**Task**:
Combine all context sources and generate SessionStart hook response JSON.

**File**: `pkg/session/context_response.go` (new file)

**Implementation**:
```go
package session

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ContextComponents holds all context pieces for session initialization
type ContextComponents struct {
	SessionType      string                  // "startup" or "resume"
	RoutingSummary   string                  // From schema.FormatTierSummary()
	HandoffSummary   string                  // From LoadHandoffSummary() - resume only
	PendingLearnings string                  // From CheckPendingLearnings()
	GitInfo          string                  // From FormatGitInfo()
	ProjectInfo      *ProjectDetectionResult // From DetectProjectType()
}

// SessionStartResponse is the hook output format for SessionStart
type SessionStartResponse struct {
	HookSpecificOutput HookSpecificOutput `json:"hookSpecificOutput"`
}

// HookSpecificOutput contains the context injection payload
type HookSpecificOutput struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
}

// GenerateSessionStartResponse creates the complete context injection response.
// Output follows Claude Code hook response format.
func GenerateSessionStartResponse(ctx *ContextComponents) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("[context-response] ContextComponents nil. Cannot generate response without context.")
	}

	var contextParts []string

	// Session header
	header := fmt.Sprintf("🚀 SESSION INITIALIZED (%s)", ctx.SessionType)
	contextParts = append(contextParts, header)

	// Routing summary (always include)
	if ctx.RoutingSummary != "" {
		contextParts = append(contextParts, ctx.RoutingSummary)
	}

	// Handoff for resume sessions only
	if ctx.SessionType == "resume" && ctx.HandoffSummary != "" {
		contextParts = append(contextParts, "PREVIOUS SESSION HANDOFF:\n"+ctx.HandoffSummary)
	}

	// Pending learnings warning (if any)
	if ctx.PendingLearnings != "" {
		contextParts = append(contextParts, ctx.PendingLearnings)
	}

	// Git info (if in git repo)
	if ctx.GitInfo != "" {
		contextParts = append(contextParts, ctx.GitInfo)
	}

	// Project type detection
	if ctx.ProjectInfo != nil {
		contextParts = append(contextParts, FormatProjectType(ctx.ProjectInfo))
	}

	// Hook status footer
	contextParts = append(contextParts, "Routing hooks are ACTIVE. Tool usage validated against routing-schema.json.")

	// Combine all parts
	fullContext := strings.Join(contextParts, "\n\n")

	// Build response
	response := SessionStartResponse{
		HookSpecificOutput: HookSpecificOutput{
			HookEventName:     "SessionStart",
			AdditionalContext: fullContext,
		},
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", fmt.Errorf("[context-response] Failed to marshal response: %w", err)
	}

	return string(data), nil
}

// GenerateErrorResponse creates an error response in hook format.
// Errors are displayed but don't block session start.
func GenerateErrorResponse(message string) string {
	response := SessionStartResponse{
		HookSpecificOutput: HookSpecificOutput{
			HookEventName:     "SessionStart",
			AdditionalContext: fmt.Sprintf("🔴 SESSION START ERROR: %s\n\nSession continues but context injection failed.", message),
		},
	}

	data, _ := json.MarshalIndent(response, "", "  ")
	return string(data)
}
```

**Tests**: `pkg/session/context_response_test.go` (new file)

```go
package session

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerateSessionStartResponse_Startup(t *testing.T) {
	ctx := &ContextComponents{
		SessionType:    "startup",
		RoutingSummary: "ROUTING TIERS ACTIVE:\n  • haiku: find files...",
		GitInfo:        "GIT: Branch: main | Uncommitted: 2 file(s)",
		ProjectInfo: &ProjectDetectionResult{
			Type:        ProjectGo,
			Indicators:  []string{"go.mod"},
			Conventions: []string{"go.md"},
		},
	}

	response, err := GenerateSessionStartResponse(ctx)

	if err != nil {
		t.Fatalf("GenerateSessionStartResponse failed: %v", err)
	}

	// Verify valid JSON
	var parsed SessionStartResponse
	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		t.Fatalf("Invalid JSON response: %v", err)
	}

	// Verify hook event name
	if parsed.HookSpecificOutput.HookEventName != "SessionStart" {
		t.Errorf("Expected hookEventName 'SessionStart', got: %s", parsed.HookSpecificOutput.HookEventName)
	}

	context := parsed.HookSpecificOutput.AdditionalContext

	// Verify content
	if !strings.Contains(context, "SESSION INITIALIZED (startup)") {
		t.Error("Should indicate startup session")
	}

	if !strings.Contains(context, "ROUTING TIERS") {
		t.Error("Should include routing summary")
	}

	if !strings.Contains(context, "GIT:") {
		t.Error("Should include git info")
	}

	if !strings.Contains(context, "go") {
		t.Error("Should include project type")
	}

	if !strings.Contains(context, "hooks are ACTIVE") {
		t.Error("Should include hook status footer")
	}
}

func TestGenerateSessionStartResponse_Resume(t *testing.T) {
	ctx := &ContextComponents{
		SessionType:      "resume",
		RoutingSummary:   "ROUTING TIERS ACTIVE:\n  • haiku: find files...",
		HandoffSummary:   "# Session Handoff\n\nLast session completed feature X.",
		PendingLearnings: "⚠️ PENDING LEARNINGS: 3 sharp edge(s)",
		ProjectInfo: &ProjectDetectionResult{
			Type: ProjectGeneric,
		},
	}

	response, err := GenerateSessionStartResponse(ctx)

	if err != nil {
		t.Fatalf("GenerateSessionStartResponse failed: %v", err)
	}

	var parsed SessionStartResponse
	json.Unmarshal([]byte(response), &parsed)
	context := parsed.HookSpecificOutput.AdditionalContext

	// Verify resume-specific content
	if !strings.Contains(context, "SESSION INITIALIZED (resume)") {
		t.Error("Should indicate resume session")
	}

	if !strings.Contains(context, "PREVIOUS SESSION HANDOFF") {
		t.Error("Should include handoff header for resume")
	}

	if !strings.Contains(context, "feature X") {
		t.Error("Should include handoff content")
	}

	if !strings.Contains(context, "PENDING LEARNINGS") {
		t.Error("Should include pending learnings warning")
	}
}

func TestGenerateSessionStartResponse_StartupNoHandoff(t *testing.T) {
	ctx := &ContextComponents{
		SessionType:    "startup",
		HandoffSummary: "# Some handoff content", // Should be ignored for startup
		ProjectInfo:    &ProjectDetectionResult{Type: ProjectGeneric},
	}

	response, err := GenerateSessionStartResponse(ctx)

	if err != nil {
		t.Fatalf("GenerateSessionStartResponse failed: %v", err)
	}

	var parsed SessionStartResponse
	json.Unmarshal([]byte(response), &parsed)
	context := parsed.HookSpecificOutput.AdditionalContext

	// Handoff should NOT be included for startup
	if strings.Contains(context, "PREVIOUS SESSION HANDOFF") {
		t.Error("Startup session should not include handoff")
	}
}

func TestGenerateSessionStartResponse_Nil(t *testing.T) {
	_, err := GenerateSessionStartResponse(nil)

	if err == nil {
		t.Error("Expected error for nil context, got nil")
	}
}

func TestGenerateErrorResponse(t *testing.T) {
	response := GenerateErrorResponse("Test error message")

	var parsed SessionStartResponse
	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if !strings.Contains(parsed.HookSpecificOutput.AdditionalContext, "SESSION START ERROR") {
		t.Error("Should contain error indicator")
	}

	if !strings.Contains(parsed.HookSpecificOutput.AdditionalContext, "Test error message") {
		t.Error("Should contain error message")
	}
}
```

**Acceptance Criteria**:
- [ ] `ContextComponents` struct aggregates all context sources
- [ ] `GenerateSessionStartResponse()` combines components into valid JSON
- [ ] Includes routing summary, git info, project type for ALL sessions
- [ ] Includes handoff only for RESUME sessions (not startup)
- [ ] Includes pending learnings warning if present
- [ ] Output matches Claude Code hook response format
- [ ] `GenerateErrorResponse()` handles error cases gracefully
- [ ] Tests verify startup vs resume, JSON validity, content inclusion
- [ ] `go test ./pkg/session/...` passes

**Test Deliverables**:
- [ ] Test file created: `pkg/session/context_response_test.go`
- [ ] Test file size: ~140 lines
- [ ] Number of test functions: 5
- [ ] Tests passing: ✅
- [ ] **ECOSYSTEM TEST PASS REQUIRED**: `make test-ecosystem`

**Why This Matters**: Response generation is the final step in context injection. Must produce valid JSON that Claude Code can parse and inject into conversation.

---
