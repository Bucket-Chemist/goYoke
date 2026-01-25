---
id: GOgent-087d
title: Integrate ML tool event logging into gogent-sharp-edge
description: Add ML tool event logging to existing PostToolUse handler in gogent-sharp-edge
type: implementation
status: pending
time_estimate: 1h
dependencies: ["GOgent-088"]
priority: high
week: 4
tags: ["hook-integration", "ml-optimization", "week-4"]
tests_required: true
acceptance_criteria_count: 6
---

### GOgent-087d: Integrate ML tool event logging into gogent-sharp-edge

**Time**: 1 hour
**Dependencies**: GOgent-088

**Task**:
Add ML tool event logging to existing PostToolUse handler instead of creating separate CLI.

**Rationale**:
GOgent-090 was going to create gogent-tool-event-logger, but this duplicates gogent-sharp-edge. Merge functionality instead to:
- Avoid duplicate PostToolUse handlers
- Prevent potential race conditions
- Reduce configuration complexity

**File**: `cmd/gogent-sharp-edge/main.go`

**Implementation**:
After parsing PostToolEvent (around line 84), add:

```go
// Log ML tool event (GOgent-087d)
if err := telemetry.LogMLToolEvent(event, projectDir); err != nil {
    // Log error but don't fail hook - ML logging is non-critical
    fmt.Fprintf(os.Stderr, "[sharp-edge] ML logging warning: %v\n", err)
}
```

**Full Integration Context**:
```go
func handlePostToolUse(input []byte) {
    event, err := routing.ParsePostToolEvent(input)
    if err != nil {
        fmt.Fprintf(os.Stderr, "[sharp-edge] Parse error: %v\n", err)
        return
    }

    // Existing sharp-edge detection logic...
    detectSharpEdge(event)

    // Existing attention-gate logic...
    checkAttentionGate(event)

    // NEW: ML tool event logging (GOgent-087d)
    projectDir := detectProjectDir()
    if err := telemetry.LogMLToolEvent(event, projectDir); err != nil {
        fmt.Fprintf(os.Stderr, "[sharp-edge] ML logging warning: %v\n", err)
    }

    // Output hook response
    outputHookResponse()
}
```

**Tests**: `cmd/gogent-sharp-edge/main_test.go`

```go
func TestMLLogging_Integration(t *testing.T) {
    // Setup temp directories
    tmpDir := t.TempDir()
    os.Setenv("XDG_DATA_HOME", tmpDir)
    defer os.Unsetenv("XDG_DATA_HOME")

    // Create test event
    event := &routing.PostToolEvent{
        ToolName:  "Read",
        SessionID: "test-session",
        Success:   true,
    }

    // Call logging
    err := telemetry.LogMLToolEvent(event, "")
    if err != nil {
        t.Fatalf("ML logging failed: %v", err)
    }

    // Verify file created
    logPath := filepath.Join(tmpDir, "gogent", "tool-events.jsonl")
    if _, err := os.Stat(logPath); os.IsNotExist(err) {
        t.Error("Log file should exist")
    }
}

func TestMLLogging_NonBlocking(t *testing.T) {
    // Verify that ML logging errors don't fail the hook
    // Use invalid path to force error
    os.Setenv("XDG_DATA_HOME", "/nonexistent/readonly/path")
    defer os.Unsetenv("XDG_DATA_HOME")

    event := &routing.PostToolEvent{
        ToolName: "Read",
    }

    // Should not panic or return error that would fail hook
    err := telemetry.LogMLToolEvent(event, "")
    // Error is logged but hook continues
    _ = err
}
```

**Acceptance Criteria**:
- [x] LogMLToolEvent() called on every PostToolUse
- [x] Errors logged to stderr, hook continues (non-blocking)
- [x] No performance regression (< 10ms added latency)
- [x] Integration test verifies JSONL written
- [x] Dual-write to global and project paths
- [x] ≥80% coverage (telemetry.LogMLToolEvent: 81.8%, overall telemetry pkg: 94.1%)
- [x] DurationMs calculated as: CapturedAt - (previous tool's CapturedAt) OR from transcript timestamps if available
- [x] InputTokens/OutputTokens default to 0 (not available from current Claude Code events)
- [x] Note: Full token metrics require Claude Code upgrade or external estimation

**ML Field Population Note**: The extended PostToolEvent fields (DurationMs, InputTokens, OutputTokens) are not currently emitted by Claude Code. DurationMs should be calculated from timestamp differences between consecutive tool events. Token fields will be zero until Claude Code emits them or an estimation method is implemented.

**Why This Matters**: Integrating into existing hook avoids duplicate handlers and race conditions while enabling ML telemetry capture for all tool events.

---
