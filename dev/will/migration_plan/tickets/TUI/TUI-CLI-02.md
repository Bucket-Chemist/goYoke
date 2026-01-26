# TUI-CLI-02: Event Type Definitions

> **Estimated Hours:** 1.5
> **Priority:** P0 - Foundation
> **Dependencies:** TUI-CLI-01
> **Phase:** 2 - Event System

---

## Description

Define Go structs for all Claude CLI stream-json event types received from the subprocess.

---

## Event Types

| Type | Subtype | Purpose |
|------|---------|---------|
| `system` | `init` | Session initialization |
| `system` | `hook_started` | Hook execution begins |
| `system` | `hook_response` | Hook output |
| `assistant` | - | Model response (with content blocks) |
| `result` | `success`/`error` | Final result |

---

## Tasks

### 1. Create Event Type Definitions

**File:** `internal/cli/events.go`

```go
package cli

import "encoding/json"

// Event is the base type for all Claude CLI events
type Event struct {
    Type    string          `json:"type"`
    Subtype string          `json:"subtype,omitempty"`
    Raw     json.RawMessage `json:"-"` // Original JSON for re-parsing
}

// SystemEvent represents system-level events (init, hooks)
type SystemEvent struct {
    Event
    HookID     string   `json:"hook_id,omitempty"`
    HookName   string   `json:"hook_name,omitempty"`
    CWD        string   `json:"cwd,omitempty"`
    SessionID  string   `json:"session_id"`
    Tools      []string `json:"tools,omitempty"`
    Model      string   `json:"model,omitempty"`
    ExitCode   int      `json:"exit_code,omitempty"`
    Stdout     string   `json:"stdout,omitempty"`
}

// AssistantEvent represents model responses
type AssistantEvent struct {
    Event
    Message   AssistantMessage `json:"message"`
    SessionID string           `json:"session_id"`
    Partial   bool             `json:"partial,omitempty"` // True for streaming chunks
}

type AssistantMessage struct {
    Content []ContentBlock `json:"content"`
    Model   string         `json:"model"`
    Usage   Usage          `json:"usage,omitempty"`
}

type ContentBlock struct {
    Type string `json:"type"` // "text", "tool_use", etc.
    Text string `json:"text,omitempty"`
    ID   string `json:"id,omitempty"`
    Name string `json:"name,omitempty"`
}

type Usage struct {
    InputTokens  int `json:"input_tokens"`
    OutputTokens int `json:"output_tokens"`
}

// ResultEvent represents final session result
type ResultEvent struct {
    Event
    IsError      bool    `json:"is_error"`
    DurationMs   int64   `json:"duration_ms"`
    Result       string  `json:"result"`
    SessionID    string  `json:"session_id"`
    TotalCostUSD float64 `json:"total_cost_usd"`
}

// UserMessage is the format for sending messages to Claude
type UserMessage struct {
    Type    string `json:"type"` // Always "user"
    Content string `json:"content"`
}
```

### 2. Create Event Parsing Functions

```go
// ParseEvent parses raw JSON into appropriate event type
func ParseEvent(data []byte) (Event, error) {
    var base Event
    if err := json.Unmarshal(data, &base); err != nil {
        return Event{}, err
    }
    base.Raw = data

    // Could add type-specific parsing here
    return base, nil
}

// AsSystem attempts to parse as SystemEvent
func (e Event) AsSystem() (*SystemEvent, error) {
    var se SystemEvent
    if err := json.Unmarshal(e.Raw, &se); err != nil {
        return nil, err
    }
    return &se, nil
}

// AsAssistant attempts to parse as AssistantEvent
func (e Event) AsAssistant() (*AssistantEvent, error) {
    var ae AssistantEvent
    if err := json.Unmarshal(e.Raw, &ae); err != nil {
        return nil, err
    }
    return &ae, nil
}

// AsResult attempts to parse as ResultEvent
func (e Event) AsResult() (*ResultEvent, error) {
    var re ResultEvent
    if err := json.Unmarshal(e.Raw, &re); err != nil {
        return nil, err
    }
    return &re, nil
}
```

---

## Files to Create

| File | Purpose |
|------|---------|
| `internal/cli/events.go` | Event type definitions and parsing |
| `internal/cli/events_test.go` | Unit tests with real Claude CLI output samples |

---

## Acceptance Criteria

- [ ] All event types unmarshal correctly from Claude CLI output
- [ ] Unknown event types don't panic (graceful handling)
- [ ] Partial messages identified via `Partial` field
- [ ] Content blocks properly extracted
- [ ] `UserMessage` correctly formats for sending
- [ ] Unit tests with real Claude CLI output samples

---

## Test Strategy

### Unit Tests
- Parse each event type from sample JSON
- Handle unknown event types gracefully
- Validate content block extraction
- Test UserMessage serialization

### Sample Events
Collect real events from Claude CLI runs:
```bash
claude --output-format stream-json --print "Hello" 2>/dev/null | head -20
```
