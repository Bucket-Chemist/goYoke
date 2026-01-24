---
id: GOgent-072
title: Merge Attention-Gate into gogent-sharp-edge
description: **Task**:
status: pending
time_estimate: 2h
dependencies: ["GOgent-069", "GOgent-071"]
priority: high
week: 4
tags: ["attention-gate", "week-4"]
tests_required: true
acceptance_criteria_count: 15
---

### GOgent-072: Merge Attention-Gate into gogent-sharp-edge

**Time**: 2 hours
**Dependencies**: GOgent-069 (flush logic), GOgent-071 (integration tests)

**Task**:
Extend existing `gogent-sharp-edge` CLI with counter/reminder/flush logic. DO NOT create new CLI - merge into existing PostToolUse handler to avoid hook conflicts.

**CRITICAL**: Claude Code typically supports ONE hook per event type. Having both `gogent-sharp-edge` and `gogent-attention-gate` for PostToolUse creates configuration conflicts. Solution: Merge all PostToolUse logic into single handler.

**File**: `cmd/gogent-sharp-edge/main.go` (EXTEND existing CLI)

**Current gogent-sharp-edge Functionality** (keep all of this):
- Parse PostToolUse events via `routing.ParsePostToolEvent()`
- Detect failures with `routing.DetectFailure()`
- Track repeated failures for sharp edge detection
- Generate blocking responses on critical failures

**New Functionality to ADD**:
- Increment tool counter using `config.GetToolCountAndIncrement()`
- Check if reminder needed with `config.ShouldRemind(count)`
- Check if flush needed with `config.ShouldFlush(count)`
- Generate reminder message via `session.GenerateRoutingReminder()`
- Execute flush via `session.ArchivePendingLearnings()`
- Combine sharp-edge + reminder + flush into single response

**Implementation** (extend `cmd/gogent-sharp-edge/main.go`):

```go
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/session"
)

const (
	DEFAULT_TIMEOUT = 5 * time.Second
)

func main() {
	// Parse PostToolUse event (existing functionality)
	event, err := routing.ParsePostToolEvent(os.Stdin, DEFAULT_TIMEOUT)
	if err != nil {
		outputError(fmt.Sprintf("Failed to parse event: %v", err))
		os.Exit(1)
	}

	// === EXISTING SHARP-EDGE LOGIC ===
	// Detect failure patterns
	failure := routing.DetectFailure(event)

	// === NEW: ATTENTION-GATE LOGIC ===
	// Increment tool counter (non-blocking on error)
	count, counterErr := config.GetToolCountAndIncrement()
	if counterErr != nil {
		fmt.Fprintf(os.Stderr, "[sharp-edge] Warning: counter error: %v\n", counterErr)
		count = 0 // Continue with count=0 on error
	}

	// Check reminder threshold
	var reminderMsg string
	if config.ShouldRemind(count) {
		summary := "See routing-schema.json for complete tier mappings"
		reminderMsg = session.GenerateRoutingReminder(count, summary)
	}

	// Check flush threshold
	var flushMsg string
	if config.ShouldFlush(count) {
		projectDir := getProjectDir()
		shouldFlush, _, flushErr := session.ShouldFlushLearnings(projectDir)
		if flushErr != nil {
			fmt.Fprintf(os.Stderr, "[sharp-edge] Warning: flush check error: %v\n", flushErr)
		} else if shouldFlush {
			ctx, archiveErr := session.ArchivePendingLearnings(projectDir)
			if archiveErr != nil {
				fmt.Fprintf(os.Stderr, "[sharp-edge] Warning: archive error: %v\n", archiveErr)
			} else {
				flushMsg = session.GenerateFlushNotification(ctx)
			}
		}
	}

	// === COMBINE ALL RESPONSES ===
	response := buildCombinedResponse(failure, reminderMsg, flushMsg)

	// Output final response
	if err := response.Marshal(os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "[sharp-edge] Failed to marshal response: %v\n", err)
		fmt.Println("{}") // Fallback empty response
	}
}

// getProjectDir returns project directory with environment variable priority.
// Priority: GOGENT_PROJECT_DIR > CLAUDE_PROJECT_DIR > CWD
func getProjectDir() string {
	// 1. GOgent-specific override (highest priority)
	if dir := os.Getenv("GOGENT_PROJECT_DIR"); dir != "" {
		return dir
	}
	// 2. Claude Code standard
	if dir := os.Getenv("CLAUDE_PROJECT_DIR"); dir != "" {
		return dir
	}
	// 3. Current working directory (fallback)
	dir, _ := os.Getwd()
	return dir
}

// buildCombinedResponse merges sharp-edge, reminder, and flush responses.
func buildCombinedResponse(failure *routing.FailureInfo, reminderMsg, flushMsg string) *routing.HookResponse {
	response := routing.NewHookResponse("PostToolUse")

	// Add sharp-edge failure context (if any)
	if failure != nil && failure.ShouldBlock {
		response.SetDecision("block")
		response.AddField("failure_reason", failure.Reason)
		response.AddField("failure_count", failure.Count)
		response.AddField("affected_file", failure.FilePath)
	} else {
		response.SetDecision("allow")
	}

	// Note: Required accessor methods for HookResponse:
	// - SetDecision(string) - sets decision field
	// - GetDecision() string - retrieves decision field
	// These must be added to pkg/routing/response.go

	// Add attention-gate context (if any)
	var contextParts []string
	if failure != nil && failure.Context != "" {
		contextParts = append(contextParts, failure.Context)
	}
	if reminderMsg != "" {
		contextParts = append(contextParts, reminderMsg)
	}
	if flushMsg != "" {
		contextParts = append(contextParts, flushMsg)
	}

	if len(contextParts) > 0 {
		combinedContext := ""
		for i, part := range contextParts {
			if i > 0 {
				combinedContext += "\n\n"
			}
			combinedContext += part
		}
		response.AddContext(combinedContext)
	}

	return response
}

func outputError(message string) {
	response := routing.NewHookResponse("PostToolUse")
	response.SetDecision("allow") // Don't block on parse errors
	response.AddContext(fmt.Sprintf("🔴 %s", message))
	response.Marshal(os.Stdout)
}
```

**Testing** (`cmd/gogent-sharp-edge/main_test.go`):

```go
package main

import (
	"os"
	"testing"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

func TestGetProjectDir_Priority(t *testing.T) {
	// Clear env vars
	os.Unsetenv("GOGENT_PROJECT_DIR")
	os.Unsetenv("CLAUDE_PROJECT_DIR")

	// Test 1: CWD fallback
	dir := getProjectDir()
	if dir == "" {
		t.Error("Should return non-empty CWD")
	}

	// Test 2: CLAUDE_PROJECT_DIR
	os.Setenv("CLAUDE_PROJECT_DIR", "/test/claude")
	defer os.Unsetenv("CLAUDE_PROJECT_DIR")
	dir = getProjectDir()
	if dir != "/test/claude" {
		t.Errorf("Expected /test/claude, got: %s", dir)
	}

	// Test 3: GOGENT_PROJECT_DIR overrides CLAUDE_PROJECT_DIR
	os.Setenv("GOGENT_PROJECT_DIR", "/test/gogent")
	defer os.Unsetenv("GOGENT_PROJECT_DIR")
	dir = getProjectDir()
	if dir != "/test/gogent" {
		t.Errorf("Expected /test/gogent, got: %s", dir)
	}
}

func TestBuildCombinedResponse_NoFailure(t *testing.T) {
	response := buildCombinedResponse(nil, "", "")

	if response.GetDecision() != "allow" {
		t.Error("Should allow when no failure")
	}
}

func TestBuildCombinedResponse_WithFailure(t *testing.T) {
	failure := &routing.FailureInfo{
		ShouldBlock: true,
		Reason:      "3 consecutive failures",
		Count:       3,
		FilePath:    "test.go",
		Context:     "Critical failure",
	}

	response := buildCombinedResponse(failure, "", "")

	if response.GetDecision() != "block" {
		t.Error("Should block on critical failure")
	}
}

func TestBuildCombinedResponse_WithReminderAndFlush(t *testing.T) {
	reminderMsg := "🔔 ROUTING CHECKPOINT"
	flushMsg := "📦 LEARNING AUTO-FLUSH"

	response := buildCombinedResponse(nil, reminderMsg, flushMsg)

	if response.GetDecision() != "allow" {
		t.Error("Should allow when only reminder/flush")
	}

	// Context should contain both messages
	// (Exact assertion depends on HookResponse.AddContext() implementation)
}
```

**Acceptance Criteria**:
- [ ] Logic merged into cmd/gogent-sharp-edge/main.go (NO new CLI created)
- [ ] Counter uses config.GetToolCountAndIncrement()
- [ ] Event parsing uses existing routing.ParsePostToolEvent() (NOT new parser)
- [ ] Environment variable priority: GOGENT_PROJECT_DIR > CLAUDE_PROJECT_DIR > CWD
- [ ] Single PostToolUse handler (no hook conflict)
- [ ] Sharp-edge logic preserved (failure detection, blocking)
- [ ] Reminder injected every 10 tools (non-blocking)
- [ ] Flush executed every 20 tools when learnings >= threshold (non-blocking)
- [ ] Combined response merges sharp-edge + reminder + flush
- [ ] Counter errors are non-fatal (warning to stderr, continue)
- [ ] Flush errors are non-fatal (warning to stderr, continue)
- [ ] getProjectDir() implements correct env var priority
- [ ] HookResponse has SetDecision() method
- [ ] HookResponse has GetDecision() method
- [ ] Tests verify env var priority
- [ ] Tests verify combined response building
- [ ] `go build ./cmd/gogent-sharp-edge` succeeds

**Why This Matters**: Merging into single PostToolUse handler avoids hook configuration conflicts and maintains architectural simplicity. All PostToolUse behavior is now in one place.

**Migration Note**: After implementing this, update `.claude/hooks.toml`:
```toml
[hooks.PostToolUse]
command = "gogent-sharp-edge"  # Single handler for all PostToolUse logic
# DO NOT add separate gogent-attention-gate entry
```

**References**:
- Existing sharp-edge: cmd/gogent-sharp-edge/main.go
- Counter functions: pkg/config/paths.go (GOgent-068)
- Session functions: pkg/session/attention_gate.go (GOgent-069)
- GAP Analysis: REFACTORING-MAP.md Section 2, GOgent-072
- PostToolUse conflict: GAP-ANALYSIS Appendix E

---
