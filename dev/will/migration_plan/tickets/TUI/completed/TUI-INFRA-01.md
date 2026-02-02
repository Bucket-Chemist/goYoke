# TUI-INFRA-01: Agent Lifecycle Telemetry

> **Estimated Hours:** 2.0
> **Priority:** P0 - BLOCKING
> **Dependencies:** None
> **Phase:** 0 - Infrastructure

---

## Description

Create a new telemetry subsystem that tracks agent spawn and completion events in real-time. This enables the TUI to display live agent delegation status.

**Problem:** Currently, `gogent-validate` logs routing decisions and `gogent-agent-endstate` logs collaborations, but there's no unified lifecycle view that tracks agents from spawn → running → complete.

**Solution:** New `agent-lifecycle.jsonl` file with events emitted from both hooks, linked by correlation IDs.

---

## Tasks

### 1. Create Agent Lifecycle Types

**File:** `pkg/telemetry/agent_lifecycle.go`

```go
package telemetry

type AgentLifecycleEvent struct {
    EventID        string  `json:"event_id"`        // UUID
    SessionID      string  `json:"session_id"`
    Timestamp      int64   `json:"timestamp"`
    EventType      string  `json:"event_type"`      // "spawn" | "complete" | "error"

    // Agent identity
    AgentID        string  `json:"agent_id"`        // "python-pro", etc.
    ParentAgent    string  `json:"parent_agent"`    // "terminal" or parent agent
    Tier           string  `json:"tier"`

    // Task context
    TaskDescription string `json:"task_description"`
    DecisionID     string  `json:"decision_id"`     // Links to routing-decisions.jsonl

    // Completion data (only for "complete"/"error")
    Success        *bool   `json:"success,omitempty"`
    DurationMs     *int64  `json:"duration_ms,omitempty"`
    ErrorMessage   *string `json:"error_message,omitempty"`
}

func NewAgentLifecycleEvent(sessionID, eventType, agentID, parentAgent, tier, taskDesc, decisionID string) *AgentLifecycleEvent

func LogAgentLifecycle(event *AgentLifecycleEvent) error

func ReadAgentLifecycleLogs(sessionID string) ([]AgentLifecycleEvent, error)
```

### 2. Add Path Configuration

**File:** `pkg/config/paths.go`

Add:
```go
func GetAgentLifecyclePathWithProjectDir() string {
    return filepath.Join(getMLDataDir(), "agent-lifecycle.jsonl")
}
```

### 3. Modify gogent-validate to Emit Spawn Events

**File:** `cmd/gogent-validate/main.go`

After routing decision logging (around line 53-67), add:

```go
// Emit lifecycle spawn event for TUI real-time tracking
lifecycle := telemetry.NewAgentLifecycleEvent(
    event.SessionID,
    "spawn",
    extractAgentFromPrompt(taskInput.Prompt),
    "terminal", // Parent is terminal for direct spawns
    taskInput.Model,
    truncateDescription(taskInput.Prompt, 100),
    decision.DecisionID,
)
if err := telemetry.LogAgentLifecycle(lifecycle); err != nil {
    fmt.Fprintf(os.Stderr, "[gogent-validate] Warning: Failed to log lifecycle spawn: %v\n", err)
}
```

### 4. Modify gogent-agent-endstate to Emit Complete Events

**File:** `cmd/gogent-agent-endstate/main.go`

After collaboration logging (around line 72-75), add:

```go
// Emit lifecycle complete event for TUI real-time tracking
success := metadata.IsSuccess()
duration := int64(metadata.DurationMs)
lifecycle := telemetry.NewAgentLifecycleEvent(
    event.SessionID,
    "complete",
    metadata.AgentID,
    "terminal",
    metadata.Tier,
    "", // No description on completion
    "", // TODO: Correlate with spawn DecisionID
)
lifecycle.Success = &success
lifecycle.DurationMs = &duration
if err := telemetry.LogAgentLifecycle(lifecycle); err != nil {
    fmt.Fprintf(os.Stderr, "[gogent-agent-endstate] Warning: Failed to log lifecycle complete: %v\n", err)
}
```

---

## Files to Create

| File | Purpose |
|------|---------|
| `pkg/telemetry/agent_lifecycle.go` | Type definitions and logging |
| `pkg/telemetry/agent_lifecycle_test.go` | Unit tests |

## Files to Modify

| File | Change |
|------|--------|
| `pkg/config/paths.go` | Add `GetAgentLifecyclePathWithProjectDir()` |
| `cmd/gogent-validate/main.go` | Emit spawn events |
| `cmd/gogent-agent-endstate/main.go` | Emit complete events |

---

## Acceptance Criteria

- [ ] `AgentLifecycleEvent` struct defined with all fields
- [ ] `LogAgentLifecycle()` writes to JSONL file (append-only)
- [ ] `ReadAgentLifecycleLogs()` can filter by session ID
- [ ] `gogent-validate` emits spawn event on Task() validation pass
- [ ] `gogent-agent-endstate` emits complete event on SubagentStop
- [ ] Events written to `$XDG_DATA_HOME/gogent/agent-lifecycle.jsonl`
- [ ] Unit tests pass with 80%+ coverage
- [ ] Integration test: run a Task, verify both spawn and complete events logged

---

## Verification

```bash
# After running a claudeGO session with Task() calls:
cat ~/.local/share/gogent/agent-lifecycle.jsonl | jq -s 'group_by(.agent_id) | map({agent: .[0].agent_id, events: map(.event_type)})'

# Expected output:
# [
#   {"agent": "python-pro", "events": ["spawn", "complete"]},
#   {"agent": "orchestrator", "events": ["spawn", "complete"]}
# ]
```

---

## Notes

- Use append-only write pattern (O_APPEND) for thread safety
- Keep task description truncated to 100 chars to avoid bloat
- DecisionID correlation between spawn and complete is nice-to-have for MVP
- Error events should include error message when available
