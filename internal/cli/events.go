package cli

import (
	"encoding/json"
	"fmt"
	"time"
)

// Event is the base type for all Claude CLI events.
// It uses a two-stage parsing approach: first parse the base Event
// to determine type, then use AsSystem/AsAssistant/AsResult to get
// the fully typed event with all fields.
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
	SessionID string           `json:"session_id,omitempty"`
	Partial   bool             `json:"partial,omitempty"` // True for streaming chunks
}

// AssistantMessage contains the actual content from the assistant
type AssistantMessage struct {
	Content []ContentBlock `json:"content"`
	Model   string         `json:"model,omitempty"`
	Usage   Usage          `json:"usage,omitempty"`
}

// ContentBlock represents a single content element in a message
type ContentBlock struct {
	Type string `json:"type"` // "text", "tool_use", "thinking", etc.
	Text string `json:"text,omitempty"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// Usage tracks token consumption for a message
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

// ErrorEvent represents an error from Claude CLI
type ErrorEvent struct {
	Event
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// UserMessage is the format for sending messages to Claude
type UserMessage struct {
	Content string `json:"content"`
}

// ParseEvent parses raw JSON into the base Event type.
// Use As* methods on the returned Event to get the fully typed event.
//
// Example:
//
//	event, err := ParseEvent(data)
//	if err != nil { return err }
//	if event.Type == "system" {
//	    sysEvent, err := event.AsSystem()
//	    // ... handle system event
//	}
func ParseEvent(data []byte) (Event, error) {
	var base Event
	if err := json.Unmarshal(data, &base); err != nil {
		return Event{}, fmt.Errorf("unmarshal event: %w", err)
	}
	base.Raw = data
	return base, nil
}

// AsSystem attempts to parse the event as SystemEvent.
// Returns error if the event is not a system event or is malformed.
func (e Event) AsSystem() (*SystemEvent, error) {
	if e.Type != "system" {
		return nil, fmt.Errorf("event type is %q, not system", e.Type)
	}
	var se SystemEvent
	if err := json.Unmarshal(e.Raw, &se); err != nil {
		return nil, fmt.Errorf("unmarshal system event: %w", err)
	}
	return &se, nil
}

// AsAssistant attempts to parse the event as AssistantEvent.
// Returns error if the event is not an assistant event or is malformed.
func (e Event) AsAssistant() (*AssistantEvent, error) {
	if e.Type != "assistant" {
		return nil, fmt.Errorf("event type is %q, not assistant", e.Type)
	}
	var ae AssistantEvent
	if err := json.Unmarshal(e.Raw, &ae); err != nil {
		return nil, fmt.Errorf("unmarshal assistant event: %w", err)
	}
	return &ae, nil
}

// AsResult attempts to parse the event as ResultEvent.
// Returns error if the event is not a result event or is malformed.
func (e Event) AsResult() (*ResultEvent, error) {
	if e.Type != "result" {
		return nil, fmt.Errorf("event type is %q, not result", e.Type)
	}
	var re ResultEvent
	if err := json.Unmarshal(e.Raw, &re); err != nil {
		return nil, fmt.Errorf("unmarshal result event: %w", err)
	}
	return &re, nil
}

// AsError attempts to parse the event as ErrorEvent.
// Returns error if parsing fails.
func (e Event) AsError() (*ErrorEvent, error) {
	if e.Type != "error" {
		return nil, fmt.Errorf("event type is %q, not error", e.Type)
	}
	var ee ErrorEvent
	if err := json.Unmarshal(e.Raw, &ee); err != nil {
		return nil, fmt.Errorf("unmarshal error event: %w", err)
	}
	return &ee, nil
}

// IsSystem returns true if this is a system event
func (e Event) IsSystem() bool {
	return e.Type == "system"
}

// IsAssistant returns true if this is an assistant event
func (e Event) IsAssistant() bool {
	return e.Type == "assistant"
}

// IsResult returns true if this is a result event
func (e Event) IsResult() bool {
	return e.Type == "result"
}

// IsError returns true if this is an error event
func (e Event) IsError() bool {
	return e.Type == "error"
}

// IsPartial returns true if this is a partial (streaming) assistant event.
// Returns false for non-assistant events.
func (e Event) IsPartial() bool {
	if !e.IsAssistant() {
		return false
	}
	ae, err := e.AsAssistant()
	if err != nil {
		return false
	}
	return ae.Partial
}

// GetSessionID extracts the session ID from any event type.
// Returns empty string if the event type doesn't have a session ID field.
func (e Event) GetSessionID() string {
	switch e.Type {
	case "system":
		se, err := e.AsSystem()
		if err != nil {
			return ""
		}
		return se.SessionID
	case "assistant":
		ae, err := e.AsAssistant()
		if err != nil {
			return ""
		}
		return ae.SessionID
	case "result":
		re, err := e.AsResult()
		if err != nil {
			return ""
		}
		return re.SessionID
	default:
		return ""
	}
}

// TimestampedEvent is a wrapper for events that include a timestamp.
// Some events from Claude CLI include timestamps, this struct handles those.
type TimestampedEvent struct {
	Event
	Timestamp time.Time `json:"timestamp"`
}

// ParseTimestampedEvent parses an event that includes a timestamp field.
// Falls back to ParseEvent if no timestamp is present.
func ParseTimestampedEvent(data []byte) (TimestampedEvent, error) {
	var te TimestampedEvent
	if err := json.Unmarshal(data, &te); err != nil {
		// Try parsing as regular event
		event, err := ParseEvent(data)
		if err != nil {
			return TimestampedEvent{}, err
		}
		return TimestampedEvent{Event: event}, nil
	}
	te.Event.Raw = data
	return te, nil
}
