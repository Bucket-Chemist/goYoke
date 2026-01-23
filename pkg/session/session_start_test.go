package session

import (
	"io"
	"strings"
	"testing"
	"time"
)

func TestParseSessionStartEvent_Startup(t *testing.T) {
	jsonInput := `{
		"type": "startup",
		"session_id": "session-abc-123",
		"timestamp": 1234567890,
		"schema_version": "1.0"
	}`

	reader := strings.NewReader(jsonInput)
	event, err := ParseSessionStartEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if event.Type != "startup" {
		t.Errorf("Expected type 'startup', got: %s", event.Type)
	}

	if event.SessionID != "session-abc-123" {
		t.Errorf("Expected session_id 'session-abc-123', got: %s", event.SessionID)
	}

	if event.SchemaVersion != "1.0" {
		t.Errorf("Expected schema_version '1.0', got: %s", event.SchemaVersion)
	}

	if !event.IsStartup() {
		t.Error("Expected IsStartup() to return true")
	}

	if event.IsResume() {
		t.Error("Expected IsResume() to return false")
	}
}

func TestParseSessionStartEvent_Resume(t *testing.T) {
	jsonInput := `{
		"type": "resume",
		"session_id": "session-xyz-789",
		"timestamp": 9876543210,
		"schema_version": "1.0"
	}`

	reader := strings.NewReader(jsonInput)
	event, err := ParseSessionStartEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if event.Type != "resume" {
		t.Errorf("Expected type 'resume', got: %s", event.Type)
	}

	if event.SessionID != "session-xyz-789" {
		t.Errorf("Expected session_id 'session-xyz-789', got: %s", event.SessionID)
	}

	if event.IsResume() == false {
		t.Error("Expected IsResume() to return true")
	}

	if event.IsStartup() {
		t.Error("Expected IsStartup() to return false")
	}
}

func TestParseSessionStartEvent_DefaultType(t *testing.T) {
	jsonInput := `{
		"session_id": "session-default-456"
	}`

	reader := strings.NewReader(jsonInput)
	event, err := ParseSessionStartEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if event.Type != "startup" {
		t.Errorf("Expected default type 'startup', got: %s", event.Type)
	}

	if event.SchemaVersion != "1.0" {
		t.Errorf("Expected default schema_version '1.0', got: %s", event.SchemaVersion)
	}

	if !event.IsStartup() {
		t.Error("Expected IsStartup() to return true for default type")
	}
}

func TestParseSessionStartEvent_InvalidType(t *testing.T) {
	jsonInput := `{
		"type": "invalid_type",
		"session_id": "session-bad-999"
	}`

	reader := strings.NewReader(jsonInput)
	_, err := ParseSessionStartEvent(reader, 5*time.Second)

	if err == nil {
		t.Error("Expected error for invalid type")
	}

	if !strings.Contains(err.Error(), "Invalid type") {
		t.Errorf("Expected 'Invalid type' in error, got: %v", err)
	}

	if !strings.Contains(err.Error(), "invalid_type") {
		t.Errorf("Expected 'invalid_type' in error message, got: %v", err)
	}
}

func TestParseSessionStartEvent_Timeout(t *testing.T) {
	reader := &blockingReader{delay: 10 * time.Second}
	_, err := ParseSessionStartEvent(reader, 100*time.Millisecond)

	if err == nil {
		t.Error("Expected timeout error")
	}

	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("Expected 'timeout' in error, got: %v", err)
	}
}

func TestParseSessionStartEvent_EmptyInput(t *testing.T) {
	jsonInput := ``

	reader := strings.NewReader(jsonInput)
	_, err := ParseSessionStartEvent(reader, 5*time.Second)

	if err == nil {
		t.Error("Expected error for empty input")
	}

	if !strings.Contains(err.Error(), "Failed to parse JSON") {
		t.Errorf("Expected JSON parse error, got: %v", err)
	}
}

func TestParseSessionStartEvent_InvalidJSON(t *testing.T) {
	jsonInput := `{invalid json structure`

	reader := strings.NewReader(jsonInput)
	_, err := ParseSessionStartEvent(reader, 5*time.Second)

	if err == nil {
		t.Error("Expected error for invalid JSON")
	}

	if !strings.Contains(err.Error(), "Failed to parse JSON") {
		t.Errorf("Expected JSON parse error, got: %v", err)
	}
}

func TestParseSessionStartEvent_MissingSessionID(t *testing.T) {
	jsonInput := `{
		"type": "startup"
	}`

	reader := strings.NewReader(jsonInput)
	event, err := ParseSessionStartEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if event.SessionID != "unknown" {
		t.Errorf("Expected default session_id 'unknown', got: %s", event.SessionID)
	}
}

func TestParseSessionStartEvent_FullFields(t *testing.T) {
	jsonInput := `{
		"type": "resume",
		"session_id": "session-full-test",
		"timestamp": 1234567890,
		"schema_version": "2.0"
	}`

	reader := strings.NewReader(jsonInput)
	event, err := ParseSessionStartEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if event.Type != "resume" {
		t.Errorf("Expected type 'resume', got: %s", event.Type)
	}

	if event.SessionID != "session-full-test" {
		t.Errorf("Expected session_id 'session-full-test', got: %s", event.SessionID)
	}

	if event.Timestamp != 1234567890 {
		t.Errorf("Expected timestamp 1234567890, got: %d", event.Timestamp)
	}

	if event.SchemaVersion != "2.0" {
		t.Errorf("Expected schema_version '2.0', got: %s", event.SchemaVersion)
	}
}

// blockingReader simulates slow/blocking input for timeout testing
type blockingReader struct {
	delay time.Duration
}

func (r *blockingReader) Read(p []byte) (n int, err error) {
	time.Sleep(r.delay)
	return 0, io.EOF
}
