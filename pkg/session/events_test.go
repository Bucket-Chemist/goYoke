package session

import (
	"io"
	"strings"
	"testing"
	"time"
)

func TestParseSessionEvent_Valid(t *testing.T) {
	jsonInput := `{
		"session_id": "abc-123",
		"transcript_path": "/tmp/session-abc-123.jsonl",
		"hook_event_name": "SessionEnd"
	}`

	reader := strings.NewReader(jsonInput)
	event, err := ParseSessionEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if event.SessionID != "abc-123" {
		t.Errorf("Expected session_id abc-123, got: %s", event.SessionID)
	}

	if event.TranscriptPath != "/tmp/session-abc-123.jsonl" {
		t.Errorf("Expected transcript path, got: %s", event.TranscriptPath)
	}
}

func TestParseSessionEvent_MissingSessionID(t *testing.T) {
	jsonInput := `{
		"transcript_path": "/tmp/session.jsonl",
		"hook_event_name": "SessionEnd"
	}`

	reader := strings.NewReader(jsonInput)
	event, err := ParseSessionEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if event.SessionID != "unknown" {
		t.Errorf("Expected default 'unknown', got: %s", event.SessionID)
	}
}

func TestParseSessionEvent_InvalidJSON(t *testing.T) {
	jsonInput := `{invalid json}`

	reader := strings.NewReader(jsonInput)
	_, err := ParseSessionEvent(reader, 5*time.Second)

	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestParseSessionEvent_Timeout(t *testing.T) {
	reader := &slowReader{delay: 10 * time.Second}
	_, err := ParseSessionEvent(reader, 100*time.Millisecond)

	if err == nil {
		t.Error("Expected timeout error")
	}

	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("Expected timeout in error, got: %v", err)
	}
}

// slowReader for timeout testing
type slowReader struct {
	delay time.Duration
}

func (r *slowReader) Read(p []byte) (n int, err error) {
	time.Sleep(r.delay)
	return 0, io.EOF
}
