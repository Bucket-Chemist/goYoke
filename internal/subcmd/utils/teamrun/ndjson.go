package teamrun

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// ---------------------------------------------------------------------------
// Stream file scanning for diff summary (UX-028)
// ---------------------------------------------------------------------------

// countNL returns the number of newline-terminated lines in s.
// Returns 0 for empty strings, otherwise counts newlines and adds 1 if the
// string does not end with a newline.
func countNL(s string) int {
	if s == "" {
		return 0
	}
	n := strings.Count(s, "\n")
	if !strings.HasSuffix(s, "\n") {
		n++
	}
	return n
}

// scanStreamNDJSON reads a stream_*.ndjson file and extracts the set of file
// paths touched by Write/Edit tool_use events, plus line deltas.
// Best-effort: returns empty set and zeros on any read or parse error.
func scanStreamNDJSON(path string) (files map[string]struct{}, linesAdded, linesRemoved int) {
	files = make(map[string]struct{})

	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	for _, rawLine := range bytes.Split(data, []byte("\n")) {
		rawLine = bytes.TrimSpace(rawLine)
		if len(rawLine) == 0 {
			continue
		}

		ev := parseStreamEvent(rawLine)
		if ev == nil || ev.Type != "assistant" {
			continue
		}

		ae := parseAssistantEvent(rawLine)
		if ae == nil {
			continue
		}

		for _, block := range ae.Message.Content {
			if block.Type != "tool_use" {
				continue
			}
			switch block.Name {
			case "Write":
				var input struct {
					FilePath string `json:"file_path"`
					Content  string `json:"content"`
				}
				if err := json.Unmarshal(block.Input, &input); err != nil || input.FilePath == "" {
					continue
				}
				files[input.FilePath] = struct{}{}
				linesAdded += countNL(input.Content)

			case "Edit":
				var input struct {
					FilePath  string `json:"file_path"`
					OldString string `json:"old_string"`
					NewString string `json:"new_string"`
				}
				if err := json.Unmarshal(block.Input, &input); err != nil || input.FilePath == "" {
					continue
				}
				files[input.FilePath] = struct{}{}
				linesRemoved += countNL(input.OldString)
				linesAdded += countNL(input.NewString)
			}
		}
	}
	return
}

// countChangedFilesInDir scans the stream_*.ndjson files for each agentID in
// the team directory and returns the count of unique file paths modified.
// Best-effort: returns 0 on any error.
func countChangedFilesInDir(teamDir string, agentIDs []string) int {
	fileSet := make(map[string]struct{})
	for _, agentID := range agentIDs {
		path := filepath.Join(teamDir, fmt.Sprintf("stream_%s.ndjson", agentID))
		files, _, _ := scanStreamNDJSON(path)
		for f := range files {
			fileSet[f] = struct{}{}
		}
	}
	return len(fileSet)
}
