package main

import (
	"encoding/json"
	"strings"
)

// ---------------------------------------------------------------------------
// Minimal NDJSON event parser for team-run UDS notifications.
//
// This is a simplified version of internal/tui/cli/events.go + agent_sync.go,
// scoped to what team-run needs: extracting tool_use events from claude CLI
// stream-json output to drive UDS progress notifications.
// ---------------------------------------------------------------------------

// streamEvent is the first-pass discriminator used to determine the event type
// without fully parsing the payload.
type streamEvent struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype,omitempty"`
}

// contentBlock is a polymorphic content block inside an assistant message.
// Only the fields relevant to tool_use are populated; all others are zero-valued.
type contentBlock struct {
	Type  string          `json:"type"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

// assistantMessage is the nested message object inside an assistant event.
type assistantMessage struct {
	Content []contentBlock `json:"content"`
}

// assistantEvent is the full parsed form of a type="assistant" NDJSON line.
type assistantEvent struct {
	Message assistantMessage `json:"message"`
}

// toolActivity summarises a single tool_use block for UDS notifications.
type toolActivity struct {
	Tool    string
	Target  string
	Preview string
}

// todoItem represents a single item from a TodoWrite tool_use input.
// Shared with uds.go for agent_todo_update IPC notifications.
type todoItem struct {
	Content string `json:"content"`
	Status  string `json:"status"`
}

// ---------------------------------------------------------------------------
// parseStreamEvent
// ---------------------------------------------------------------------------

// parseStreamEvent performs a minimal first-pass parse to extract the type
// and optional subtype discriminator from a single NDJSON line.
//
// Returns nil for empty, whitespace-only, or malformed JSON lines (skip,
// no error). The caller should check for nil before using the result.
func parseStreamEvent(line []byte) *streamEvent {
	if len(strings.TrimSpace(string(line))) == 0 {
		return nil
	}
	var ev streamEvent
	if err := json.Unmarshal(line, &ev); err != nil {
		return nil
	}
	if ev.Type == "" {
		return nil
	}
	return &ev
}

// ---------------------------------------------------------------------------
// parseAssistantEvent
// ---------------------------------------------------------------------------

// parseAssistantEvent fully parses a type="assistant" NDJSON line and returns
// the structured event. Returns nil on empty input or JSON parse failure —
// callers must check for nil.
func parseAssistantEvent(line []byte) *assistantEvent {
	if len(line) == 0 {
		return nil
	}
	var ev assistantEvent
	if err := json.Unmarshal(line, &ev); err != nil {
		return nil
	}
	return &ev
}

// ---------------------------------------------------------------------------
// extractToolActivity
// ---------------------------------------------------------------------------

// extractToolActivity builds a toolActivity summary from a tool_use content
// block. The Target is extracted from the first recognisable string field in
// block.Input; Preview is formatted as "Tool: target".
func extractToolActivity(block contentBlock) toolActivity {
	target := extractToolTarget(block.Name, block.Input)
	preview := block.Name
	if target != "" {
		preview = block.Name + ": " + target
	}
	return toolActivity{
		Tool:    block.Name,
		Target:  target,
		Preview: preview,
	}
}

// extractToolTarget returns a human-readable target string (file path,
// command, pattern, URL, etc.) appropriate for the named tool.
// Falls back to the tool name itself for unrecognised tools. Long command
// strings are truncated to 80 characters.
func extractToolTarget(toolName string, input json.RawMessage) string {
	if len(input) == 0 {
		return toolName
	}

	var fields struct {
		FilePath string `json:"file_path"`
		Command  string `json:"command"`
		Pattern  string `json:"pattern"`
		URL      string `json:"url"`
		Query    string `json:"query"`
	}
	if err := json.Unmarshal(input, &fields); err != nil {
		return toolName
	}

	switch toolName {
	case "Read", "Write", "Edit":
		return fields.FilePath
	case "Bash":
		return truncate(fields.Command, 80)
	case "Grep":
		return fields.Pattern
	case "Glob":
		return fields.Pattern
	case "WebFetch":
		return fields.URL
	case "WebSearch":
		return fields.Query
	default:
		return toolName
	}
}

// truncate shortens s to at most maxLen runes, appending "…" when truncated.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}

// ---------------------------------------------------------------------------
// parseTodoItems
// ---------------------------------------------------------------------------

// parseTodoItems parses a TodoWrite tool_use Input JSON blob and returns the
// list of todo items. Returns nil for empty or malformed input without
// panicking.
func parseTodoItems(input json.RawMessage) []todoItem {
	if len(input) == 0 {
		return nil
	}
	var payload struct {
		Todos []struct {
			Content string `json:"content"`
			Status  string `json:"status"`
		} `json:"todos"`
	}
	if err := json.Unmarshal(input, &payload); err != nil {
		return nil
	}
	if len(payload.Todos) == 0 {
		return nil
	}
	items := make([]todoItem, len(payload.Todos))
	for i, t := range payload.Todos {
		items[i] = todoItem{Content: t.Content, Status: t.Status}
	}
	return items
}
