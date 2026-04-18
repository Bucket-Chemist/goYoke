package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// parseStreamEvent
// ---------------------------------------------------------------------------

func TestParseStreamEvent_AssistantType(t *testing.T) {
	line := []byte(`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"toolu_01","name":"Read","input":{"file_path":"/some/file.go"}}]}}`)
	ev := parseStreamEvent(line)
	require.NotNil(t, ev)
	assert.Equal(t, "assistant", ev.Type)
	assert.Equal(t, "", ev.Subtype)
}

func TestParseStreamEvent_NonAssistantIgnored(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"system init", `{"type":"system","subtype":"init","cwd":"/home/user"}`},
		{"result", `{"type":"result","subtype":"success","is_error":false}`},
		{"rate_limit_event", `{"type":"rate_limit_event","rate_limit_info":{}}`},
		{"user event", `{"type":"user","message":{"role":"user","content":[]}}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ev := parseStreamEvent([]byte(tc.line))
			// parseStreamEvent returns the discriminator for ALL valid JSON lines —
			// filtering by type="assistant" is the caller's responsibility.
			require.NotNil(t, ev)
			assert.NotEqual(t, "assistant", ev.Type)
		})
	}
}

func TestParseStreamEvent_MalformedJSON(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"empty line", ""},
		{"whitespace only", "   \t\n"},
		{"invalid JSON", "{not json}"},
		{"truncated JSON", `{"type":"assi`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ev := parseStreamEvent([]byte(tc.line))
			assert.Nil(t, ev, "expected nil for input %q", tc.line)
		})
	}
}

// ---------------------------------------------------------------------------
// parseAssistantEvent
// ---------------------------------------------------------------------------

func TestParseAssistantEvent_ToolUseExtraction(t *testing.T) {
	// Real-world shaped claude CLI NDJSON assistant event with a single tool_use.
	line := []byte(`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"toolu_01XYZ","name":"Read","input":{"file_path":"/home/user/main.go"}}]}}`)

	ev := parseAssistantEvent(line)
	require.NotNil(t, ev)
	require.Len(t, ev.Message.Content, 1)

	block := ev.Message.Content[0]
	assert.Equal(t, "tool_use", block.Type)
	assert.Equal(t, "toolu_01XYZ", block.ID)
	assert.Equal(t, "Read", block.Name)

	var input struct {
		FilePath string `json:"file_path"`
	}
	require.NoError(t, json.Unmarshal(block.Input, &input))
	assert.Equal(t, "/home/user/main.go", input.FilePath)
}

func TestParseAssistantEvent_MultipleBlocks(t *testing.T) {
	line := []byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"Let me read the file."},{"type":"tool_use","id":"toolu_02","name":"Bash","input":{"command":"go build ./..."}}]}}`)

	ev := parseAssistantEvent(line)
	require.NotNil(t, ev)
	require.Len(t, ev.Message.Content, 2)
	assert.Equal(t, "text", ev.Message.Content[0].Type)
	assert.Equal(t, "tool_use", ev.Message.Content[1].Type)
}

func TestParseAssistantEvent_NilOnMalformed(t *testing.T) {
	ev := parseAssistantEvent([]byte(`{bad json`))
	assert.Nil(t, ev)
}

func TestParseAssistantEvent_NilOnEmpty(t *testing.T) {
	ev := parseAssistantEvent([]byte{})
	assert.Nil(t, ev)
}

// ---------------------------------------------------------------------------
// extractToolActivity
// ---------------------------------------------------------------------------

func TestExtractToolActivity_ReadTool(t *testing.T) {
	block := contentBlock{
		Type:  "tool_use",
		ID:    "toolu_read01",
		Name:  "Read",
		Input: json.RawMessage(`{"file_path":"/internal/tui/model/app.go"}`),
	}

	act := extractToolActivity(block)
	assert.Equal(t, "Read", act.Tool)
	assert.Equal(t, "/internal/tui/model/app.go", act.Target)
	assert.Equal(t, "Read: /internal/tui/model/app.go", act.Preview)
}

func TestExtractToolActivity_BashTool(t *testing.T) {
	block := contentBlock{
		Type:  "tool_use",
		ID:    "toolu_bash01",
		Name:  "Bash",
		Input: json.RawMessage(`{"command":"go test -race ./cmd/goyoke-team-run/..."}`),
	}

	act := extractToolActivity(block)
	assert.Equal(t, "Bash", act.Tool)
	assert.Equal(t, "go test -race ./cmd/goyoke-team-run/...", act.Target)
	assert.Equal(t, "Bash: go test -race ./cmd/goyoke-team-run/...", act.Preview)
}

func TestExtractToolActivity_GrepTool(t *testing.T) {
	block := contentBlock{
		Type:  "tool_use",
		ID:    "toolu_grep01",
		Name:  "Grep",
		Input: json.RawMessage(`{"pattern":"parseTodoItems","path":"."}`),
	}

	act := extractToolActivity(block)
	assert.Equal(t, "Grep", act.Tool)
	assert.Equal(t, "parseTodoItems", act.Target)
	assert.Equal(t, "Grep: parseTodoItems", act.Preview)
}

func TestExtractToolActivity_UnknownTool(t *testing.T) {
	block := contentBlock{
		Type:  "tool_use",
		ID:    "toolu_unknown",
		Name:  "SomeFutureTool",
		Input: json.RawMessage(`{"some_field":"value"}`),
	}

	act := extractToolActivity(block)
	assert.Equal(t, "SomeFutureTool", act.Tool)
	// Falls back to tool name as target
	assert.Equal(t, "SomeFutureTool", act.Target)
	assert.Equal(t, "SomeFutureTool: SomeFutureTool", act.Preview)
}

func TestExtractToolActivity_EmptyInput(t *testing.T) {
	block := contentBlock{
		Type:  "tool_use",
		ID:    "toolu_noinput",
		Name:  "Read",
		Input: nil,
	}

	act := extractToolActivity(block)
	assert.Equal(t, "Read", act.Tool)
	// Empty input falls back to tool name
	assert.Equal(t, "Read", act.Target)
	assert.Equal(t, "Read: Read", act.Preview)
}

func TestExtractToolActivity_LongBashTruncated(t *testing.T) {
	longCmd := "echo " + string(make([]byte, 100))
	block := contentBlock{
		Type:  "tool_use",
		ID:    "toolu_long",
		Name:  "Bash",
		Input: json.RawMessage(`{"command":"` + longCmd + `"}`),
	}

	act := extractToolActivity(block)
	// Target should be truncated to ≤80 runes
	assert.LessOrEqual(t, len([]rune(act.Target)), 80)
}

// ---------------------------------------------------------------------------
// parseTodoItems
// ---------------------------------------------------------------------------

func TestParseTodoItems_ValidInput(t *testing.T) {
	input := json.RawMessage(`{
		"todos": [
			{"content": "Implement parser", "status": "completed"},
			{"content": "Write tests", "status": "in_progress"},
			{"content": "Run go vet", "status": "pending"}
		]
	}`)

	items := parseTodoItems(input)
	require.Len(t, items, 3)
	assert.Equal(t, "Implement parser", items[0].Content)
	assert.Equal(t, "completed", items[0].Status)
	assert.Equal(t, "Write tests", items[1].Content)
	assert.Equal(t, "in_progress", items[1].Status)
	assert.Equal(t, "Run go vet", items[2].Content)
	assert.Equal(t, "pending", items[2].Status)
}

func TestParseTodoItems_EmptyInput(t *testing.T) {
	tests := []struct {
		name  string
		input json.RawMessage
	}{
		{"nil input", nil},
		{"empty bytes", json.RawMessage{}},
		{"empty todos array", json.RawMessage(`{"todos":[]}`)},
		{"null todos", json.RawMessage(`{"todos":null}`)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			items := parseTodoItems(tc.input)
			assert.Nil(t, items, "expected nil for empty/no-op input")
		})
	}
}

func TestParseTodoItems_MalformedInput(t *testing.T) {
	tests := []struct {
		name  string
		input json.RawMessage
	}{
		{"invalid JSON", json.RawMessage(`{not json}`)},
		{"wrong shape", json.RawMessage(`{"items":[{"text":"foo"}]}`)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Must not panic
			items := parseTodoItems(tc.input)
			// Either nil or empty — no panic is the key assertion
			assert.True(t, items == nil || len(items) == 0)
		})
	}
}
