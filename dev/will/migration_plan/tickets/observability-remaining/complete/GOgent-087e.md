---
id: GOgent-087e
title: Integrate routing decision logging into gogent-validate
description: Log routing decisions when Task() tool is invoked via PreToolUse
type: implementation
status: pending
time_estimate: 1h
dependencies: ["GOgent-087b"]
priority: high
week: 4
tags: ["hook-integration", "ml-optimization", "week-4"]
tests_required: true
acceptance_criteria_count: 7
---

### GOgent-087e: Integrate routing decision logging into gogent-validate

**Time**: 1 hour
**Dependencies**: GOgent-087b

**Task**:
Log routing decisions when Task() tool is invoked via PreToolUse.

**Rationale**:
Routing decisions capture the "why" of agent selection - critical training data for ML optimization.

**File**: `cmd/gogent-validate/main.go`

**Implementation**:
On PreToolUse for Task tool:

```go
// Log routing decision for Task() calls (GOgent-087e)
if event.ToolName == "Task" {
    taskInput, err := routing.ParseTaskInput(event.ToolInput)
    if err == nil {
        decision := telemetry.NewRoutingDecision(
            event.SessionID,
            taskInput.Prompt,
            taskInput.Model,                              // SelectedTier
            extractAgentFromPrompt(taskInput.Prompt),     // SelectedAgent
        )
        if err := telemetry.LogRoutingDecision(decision); err != nil {
            fmt.Fprintf(os.Stderr, "[validate] Routing decision logging warning: %v\n", err)
        }
    }
}

// extractAgentFromPrompt extracts agent ID from "AGENT: agent-name" prefix
func extractAgentFromPrompt(prompt string) string {
    lines := strings.Split(prompt, "\n")
    for _, line := range lines {
        if strings.HasPrefix(line, "AGENT:") {
            return strings.TrimSpace(strings.TrimPrefix(line, "AGENT:"))
        }
    }
    return "unknown"
}
```

**Tests**: `cmd/gogent-validate/main_test.go`

```go
func TestRoutingDecisionLogging(t *testing.T) {
    tmpDir := t.TempDir()
    os.Setenv("XDG_DATA_HOME", tmpDir)
    defer os.Unsetenv("XDG_DATA_HOME")

    decision := telemetry.NewRoutingDecision(
        "sess-123",
        "AGENT: codebase-search\n\nFind all Go files",
        "haiku",
        "codebase-search",
    )

    err := telemetry.LogRoutingDecision(decision)
    if err != nil {
        t.Fatalf("Failed to log: %v", err)
    }

    // Verify file created
    logPath := filepath.Join(tmpDir, "gogent", "routing-decisions.jsonl")
    if _, err := os.Stat(logPath); os.IsNotExist(err) {
        t.Error("Log file should exist")
    }
}

func TestExtractAgentFromPrompt(t *testing.T) {
    tests := []struct {
        prompt   string
        expected string
    }{
        {"AGENT: codebase-search\n\nFind files", "codebase-search"},
        {"AGENT: python-pro\n\n1. TASK: Implement", "python-pro"},
        {"No agent prefix here", "unknown"},
        {"", "unknown"},
    }

    for _, tc := range tests {
        result := extractAgentFromPrompt(tc.prompt)
        if result != tc.expected {
            t.Errorf("Expected %s, got %s for prompt: %s", tc.expected, result, tc.prompt[:min(20, len(tc.prompt))])
        }
    }
}

func TestRoutingDecision_GeneratesUUID(t *testing.T) {
    decision := telemetry.NewRoutingDecision("sess", "prompt", "tier", "agent")

    if decision.DecisionID == "" {
        t.Error("DecisionID should be generated")
    }

    // Verify UUID format (8-4-4-4-12)
    if len(decision.DecisionID) != 36 {
        t.Errorf("DecisionID should be UUID format, got length %d", len(decision.DecisionID))
    }
}
```

**Acceptance Criteria**:
- [x] LogRoutingDecision() called on every Task() PreToolUse
- [x] DecisionID generated (UUID)
- [x] TaskDescription extracted from tool_input.prompt
- [x] SelectedTier extracted from tool_input.model
- [x] SelectedAgent extracted from AGENT: prefix in prompt
- [x] Non-blocking (errors logged, hook continues)
- [x] ≥80% coverage (100% of testable functions)

**Why This Matters**: Routing decisions are the primary training signal for ML optimization - knowing which agent was selected for which task description enables supervised learning.

---
