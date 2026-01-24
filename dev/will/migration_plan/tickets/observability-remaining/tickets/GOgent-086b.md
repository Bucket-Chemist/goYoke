---
id: GOgent-086b
title: Extend routing.PostToolEvent with ML telemetry fields
description: Add ML telemetry fields to existing PostToolEvent struct using omitempty for backward compatibility
type: implementation
status: pending
time_estimate: 45m
dependencies: []
priority: high
week: 4
tags: ["routing", "ml-optimization", "week-4"]
tests_required: true
acceptance_criteria_count: 7
---

### GOgent-086b: Extend routing.PostToolEvent with ML telemetry fields

**Time**: 45 minutes
**Dependencies**: None

**Task**:
Add ML telemetry fields to existing PostToolEvent struct using omitempty for backward compatibility.

**Rationale**:
Instead of creating new ToolEvent in pkg/observability (which causes name collision), extend existing PostToolEvent. This:
- Avoids import cycles
- Leverages existing parsing
- Maintains backward compatibility via omitempty

**File**: `pkg/routing/events.go`

**Implementation**:
```go
type PostToolEvent struct {
    // Existing fields (DO NOT MODIFY)
    ToolName      string                 `json:"tool_name"`
    ToolInput     map[string]interface{} `json:"tool_input"`
    ToolResponse  map[string]interface{} `json:"tool_response"`
    SessionID     string                 `json:"session_id"`
    HookEventName string                 `json:"hook_event_name"`
    CapturedAt    int64                  `json:"captured_at"`

    // === ML Telemetry Fields (GOgent-086b) ===
    // All omitempty for backward compatibility

    // Performance metrics
    DurationMs   int64 `json:"duration_ms,omitempty"`
    InputTokens  int   `json:"input_tokens,omitempty"`
    OutputTokens int   `json:"output_tokens,omitempty"`

    // Model context
    Model string `json:"model,omitempty"`
    Tier  string `json:"tier,omitempty"`

    // Outcome
    Success bool `json:"success,omitempty"`

    // Sequence tracking (GAP 4.2)
    SequenceIndex    int      `json:"sequence_index,omitempty"`
    PreviousTools    []string `json:"previous_tools,omitempty"`
    PreviousOutcomes []bool   `json:"previous_outcomes,omitempty"`

    // Task classification (GAP 4.4)
    TaskType   string `json:"task_type,omitempty"`
    TaskDomain string `json:"task_domain,omitempty"`

    // Routing info (for Task() events)
    SelectedTier  string `json:"selected_tier,omitempty"`
    SelectedAgent string `json:"selected_agent,omitempty"`

    // Correlation
    EventID string `json:"event_id,omitempty"`

    // Understanding context (Addendum A.4)
    TargetSize       int64   `json:"target_size,omitempty"`
    CoverageAchieved float64 `json:"coverage_achieved,omitempty"`
    EntitiesFound    int     `json:"entities_found,omitempty"`
}
```

**Tests**: `pkg/routing/events_test.go`

```go
func TestPostToolEvent_MLFields(t *testing.T) {
    event := PostToolEvent{
        ToolName:      "Read",
        SessionID:     "sess-123",
        // ML fields
        DurationMs:    150,
        InputTokens:   1024,
        OutputTokens:  512,
        Tier:          "haiku",
        Success:       true,
        SequenceIndex: 5,
        TaskType:      "search",
        TaskDomain:    "python",
    }

    data, err := json.Marshal(event)
    if err != nil {
        t.Fatalf("Failed to marshal: %v", err)
    }

    var parsed PostToolEvent
    if err := json.Unmarshal(data, &parsed); err != nil {
        t.Fatalf("Failed to unmarshal: %v", err)
    }

    if parsed.SequenceIndex != 5 {
        t.Errorf("Expected SequenceIndex 5, got %d", parsed.SequenceIndex)
    }

    if parsed.TaskType != "search" {
        t.Errorf("Expected TaskType search, got %s", parsed.TaskType)
    }
}

func TestPostToolEvent_BackwardCompatibility(t *testing.T) {
    // Old JSON without ML fields should still parse
    oldJSON := `{"tool_name":"Read","session_id":"sess-123","captured_at":1234567890}`

    var event PostToolEvent
    if err := json.Unmarshal([]byte(oldJSON), &event); err != nil {
        t.Fatalf("Failed to parse old format: %v", err)
    }

    if event.ToolName != "Read" {
        t.Errorf("Expected Read, got %s", event.ToolName)
    }

    // ML fields should be zero values
    if event.SequenceIndex != 0 {
        t.Errorf("ML fields should be zero when not provided")
    }
}

func TestPostToolEvent_OmitEmpty(t *testing.T) {
    event := PostToolEvent{
        ToolName:  "Read",
        SessionID: "sess-123",
        // No ML fields set
    }

    data, _ := json.Marshal(event)
    jsonStr := string(data)

    // ML fields should not appear in JSON when empty
    if strings.Contains(jsonStr, "sequence_index") {
        t.Error("Empty ML fields should be omitted from JSON")
    }
}
```

**Acceptance Criteria**:
- [x] ML fields added with omitempty tags
- [x] Existing ParsePostToolEvent() unchanged (backward compat)
- [x] All new fields are optional
- [x] Existing tests still pass
- [x] New unit tests for ML field access
- [x] Documentation updated
- [x] ≥90% coverage on new code

**Why This Matters**: Extending existing struct avoids creating duplicate types and import cycles while enabling ML telemetry capture.

---
