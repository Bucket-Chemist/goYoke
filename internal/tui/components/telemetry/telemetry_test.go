package telemetry

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewTelemetryModel(t *testing.T) {
	m := NewTelemetryModel()
	if m.loaded {
		t.Error("expected loaded=false on new model")
	}
	if m.loadErr != "" {
		t.Errorf("expected empty loadErr, got %q", m.loadErr)
	}
}

func TestSetSize(t *testing.T) {
	m := NewTelemetryModel()
	m.SetSize(80, 24)
	if m.width != 80 {
		t.Errorf("expected width 80, got %d", m.width)
	}
	if m.height != 24 {
		t.Errorf("expected height 24, got %d", m.height)
	}
	if m.viewport.Width != 80 {
		t.Errorf("expected viewport width 80, got %d", m.viewport.Width)
	}
}

func TestView_Loading(t *testing.T) {
	m := NewTelemetryModel()
	view := m.View()
	if !strings.Contains(view, "Loading...") {
		t.Errorf("expected 'Loading...' when not yet loaded, got:\n%s", view)
	}
}

func TestView_Error(t *testing.T) {
	m := NewTelemetryModel()
	m.HandleMsg(TelemetryLoadedMsg{Err: errors.New("file not found")})
	view := m.View()
	if !strings.Contains(view, "Error:") {
		t.Errorf("expected 'Error:' in view, got:\n%s", view)
	}
	if !strings.Contains(view, "file not found") {
		t.Errorf("expected error message in view, got:\n%s", view)
	}
}

func TestView_Empty(t *testing.T) {
	m := NewTelemetryModel()
	m.HandleMsg(TelemetryLoadedMsg{Entries: nil})
	view := m.View()
	if !strings.Contains(view, "No routing decisions") {
		t.Errorf("expected 'No routing decisions' in empty view, got:\n%s", view)
	}
}

func TestView_WithEntries(t *testing.T) {
	m := NewTelemetryModel()
	m.SetSize(80, 20)
	entries := []RoutingEntry{
		{Timestamp: "2026-03-23T10:00:00Z", Agent: "go-pro", Tier: "sonnet", Decision: "implement"},
		{Timestamp: "2026-03-23T10:01:00Z", Agent: "haiku-scout", Tier: "haiku", Decision: "search"},
	}
	m.HandleMsg(TelemetryLoadedMsg{Entries: entries})
	view := m.View()
	if !strings.Contains(view, "Routing Telemetry") {
		t.Errorf("expected 'Routing Telemetry' in view, got:\n%s", view)
	}
	// Viewport content should contain agent names.
	if !strings.Contains(view, "go-pro") {
		t.Errorf("expected 'go-pro' in view, got:\n%s", view)
	}
}

func TestHandleMsg_LoadedMsg(t *testing.T) {
	m := NewTelemetryModel()
	entries := []RoutingEntry{{Agent: "test-agent", Tier: "haiku"}}
	m.HandleMsg(TelemetryLoadedMsg{Entries: entries})
	if !m.loaded {
		t.Error("expected loaded=true after TelemetryLoadedMsg")
	}
	if len(m.entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(m.entries))
	}
}

func TestHandleMsg_ErrorMsg(t *testing.T) {
	m := NewTelemetryModel()
	m.HandleMsg(TelemetryLoadedMsg{Err: errors.New("test error")})
	if !m.loaded {
		t.Error("expected loaded=true even on error")
	}
	if m.loadErr == "" {
		t.Error("expected loadErr to be set on error")
	}
}

func TestLoadEntriesCmd(t *testing.T) {
	// Write a temporary JSONL file.
	dir := t.TempDir()
	path := filepath.Join(dir, "decisions.jsonl")
	content := `{"timestamp":"2026-01-01T00:00:00Z","agent":"go-pro","tier":"sonnet","decision":"implement"}
{"timestamp":"2026-01-01T00:01:00Z","agent":"haiku-scout","tier":"haiku","decision":"search"}
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	cmd := LoadEntriesCmd(path)
	msg := cmd()
	loaded, ok := msg.(TelemetryLoadedMsg)
	if !ok {
		t.Fatalf("expected TelemetryLoadedMsg, got %T", msg)
	}
	if loaded.Err != nil {
		t.Fatalf("unexpected error: %v", loaded.Err)
	}
	if len(loaded.Entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(loaded.Entries))
	}
	if loaded.Entries[0].Agent != "go-pro" {
		t.Errorf("expected agent 'go-pro', got %q", loaded.Entries[0].Agent)
	}
}

func TestLoadEntriesCmd_FileNotFound(t *testing.T) {
	cmd := LoadEntriesCmd("/nonexistent/path.jsonl")
	msg := cmd()
	loaded, ok := msg.(TelemetryLoadedMsg)
	if !ok {
		t.Fatalf("expected TelemetryLoadedMsg, got %T", msg)
	}
	if loaded.Err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadEntriesCmd_MaxEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "big.jsonl")

	// Write 60 entries (more than maxEntries=50).
	var sb strings.Builder
	for i := range 60 {
		sb.WriteString(`{"agent":"a` + string(rune('0'+i%10)) + `","tier":"haiku"}` + "\n")
	}
	if err := os.WriteFile(path, []byte(sb.String()), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	cmd := LoadEntriesCmd(path)
	msg := cmd()
	loaded := msg.(TelemetryLoadedMsg)
	if len(loaded.Entries) != maxEntries {
		t.Errorf("expected %d entries (cap), got %d", maxEntries, len(loaded.Entries))
	}
}

func TestLoadEntriesCmd_SkipsMalformedLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mixed.jsonl")
	content := `{"agent":"good"}
not valid json
{"agent":"also-good"}
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	cmd := LoadEntriesCmd(path)
	msg := cmd()
	loaded := msg.(TelemetryLoadedMsg)
	if loaded.Err != nil {
		t.Fatalf("unexpected error: %v", loaded.Err)
	}
	if len(loaded.Entries) != 2 {
		t.Errorf("expected 2 valid entries, got %d", len(loaded.Entries))
	}
}

func TestLoadEntriesCmd_LargeLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.jsonl")
	largeDecision := strings.Repeat("d", 70*1024)
	content := `{"timestamp":"2026-01-01T00:00:00Z","agent":"go-pro","tier":"sonnet","decision":"` + largeDecision + `"}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	cmd := LoadEntriesCmd(path)
	msg := cmd()
	loaded := msg.(TelemetryLoadedMsg)
	if loaded.Err != nil {
		t.Fatalf("unexpected error: %v", loaded.Err)
	}
	if len(loaded.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(loaded.Entries))
	}
	if loaded.Entries[0].Decision != largeDecision {
		t.Fatalf("expected large decision to round-trip")
	}
}

func TestShortTimestamp(t *testing.T) {
	tests := []struct {
		ts   string
		want string
	}{
		{"2026-03-23T10:30:45Z", "10:30:45"},
		{"", "--:--:--"},
		{"short", "short"},
	}
	for _, tc := range tests {
		got := shortTimestamp(tc.ts)
		if got != tc.want {
			t.Errorf("shortTimestamp(%q): want %q, got %q", tc.ts, tc.want, got)
		}
	}
}

func TestOrDash(t *testing.T) {
	if orDash("") != "\u2014" {
		t.Error("expected em-dash for empty string")
	}
	if orDash("hello") != "hello" {
		t.Error("expected 'hello' unchanged")
	}
}

func TestRenderEntry(t *testing.T) {
	m := NewTelemetryModel()
	m.SetSize(80, 24)
	e := RoutingEntry{
		Timestamp: "2026-03-23T15:04:05Z",
		Agent:     "go-pro",
		Tier:      "sonnet",
		Decision:  "implement feature X",
	}
	rendered := m.renderEntry(e)
	if !strings.Contains(rendered, "15:04:05") {
		t.Errorf("expected timestamp in entry, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "go-pro") {
		t.Errorf("expected agent in entry, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "sonnet") {
		t.Errorf("expected tier in entry, got:\n%s", rendered)
	}
}
