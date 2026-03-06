package session

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// SessionEvent represents SessionEnd hook event
type SessionEvent struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	HookEventName  string `json:"hook_event_name"`
	Timestamp      int64  `json:"timestamp,omitempty"`

	// v2.1.69 common fields
	CWD            string `json:"cwd,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
}

// SessionMetrics represents collected session statistics
type SessionMetrics struct {
	ToolCalls         int    `json:"tool_calls"`
	ErrorsLogged      int    `json:"errors_logged"`
	RoutingViolations int    `json:"routing_violations"`
	SessionID         string `json:"session_id"`
	Duration          int64  `json:"duration_seconds,omitempty"`
}

// SessionStartEvent represents SessionStart hook event
type SessionStartEvent struct {
	Type          string `json:"type"`           // "startup", "resume", "clear", or "compact"
	SessionID     string `json:"session_id"`
	Timestamp     int64  `json:"timestamp,omitempty"`
	SchemaVersion string `json:"schema_version"` // Default "1.0"

	// v2.1.69 common + SessionStart-specific fields
	Source         string `json:"source,omitempty"`          // "startup"|"resume"|"clear"|"compact"
	Model          string `json:"model,omitempty"`           // Model identifier
	AgentType      string `json:"agent_type,omitempty"`      // Present with --agent
	CWD            string `json:"cwd,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	TranscriptPath string `json:"transcript_path,omitempty"`
}

// IsResume returns true if this is a resume session
func (e *SessionStartEvent) IsResume() bool {
	return e.Type == "resume"
}

// IsStartup returns true if this is a startup session
func (e *SessionStartEvent) IsStartup() bool {
	return e.Type == "startup"
}

// IsClear returns true if this is a clear session (v2.1.69+)
func (e *SessionStartEvent) IsClear() bool {
	return e.Type == "clear"
}

// IsCompact returns true if this is a compact session (v2.1.69+)
func (e *SessionStartEvent) IsCompact() bool {
	return e.Type == "compact"
}

// ParseSessionEvent reads SessionEnd event from STDIN
func ParseSessionEvent(r io.Reader, timeout time.Duration) (*SessionEvent, error) {
	type result struct {
		event *SessionEvent
		err   error
	}

	ch := make(chan result, 1)

	go func() {
		data, err := io.ReadAll(r)
		if err != nil {
			ch <- result{nil, fmt.Errorf("[session-parser] Failed to read STDIN: %w", err)}
			return
		}

		var event SessionEvent
		if err := json.Unmarshal(data, &event); err != nil {
			ch <- result{nil, fmt.Errorf("[session-parser] Failed to parse JSON: %w. Input: %s", err, string(data[:min(100, len(data))]))}
			return
		}

		// Validate required fields
		if event.SessionID == "" {
			event.SessionID = "unknown"
		}

		ch <- result{&event, nil}
	}()

	select {
	case res := <-ch:
		return res.event, res.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("[session-parser] STDIN read timeout after %v", timeout)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ParseSessionStartEvent reads SessionStart event from STDIN
func ParseSessionStartEvent(r io.Reader, timeout time.Duration) (*SessionStartEvent, error) {
	type result struct {
		event *SessionStartEvent
		err   error
	}

	ch := make(chan result, 1)

	go func() {
		data, err := io.ReadAll(r)
		if err != nil {
			ch <- result{nil, fmt.Errorf("[session-parser] Failed to read STDIN: %w", err)}
			return
		}

		var event SessionStartEvent
		if err := json.Unmarshal(data, &event); err != nil {
			ch <- result{nil, fmt.Errorf("[session-parser] Failed to parse JSON: %w. Input: %s", err, string(data[:min(100, len(data))]))}
			return
		}

		// Default schema_version to "1.0"
		if event.SchemaVersion == "" {
			event.SchemaVersion = "1.0"
		}

		// Default type to "startup"
		if event.Type == "" {
			event.Type = "startup"
		}

		// Validate type field (v2.1.69 adds "clear" and "compact")
		validTypes := map[string]bool{
			"startup": true, "resume": true,
			"clear": true, "compact": true,
		}
		if event.Type != "" && !validTypes[event.Type] {
			// Log unknown types but allow through (future-proofing)
			fmt.Fprintf(os.Stderr, "[session-parser] Warning: Unknown session type: %s (allowing)\n", event.Type)
		}

		// Validate required fields
		if event.SessionID == "" {
			event.SessionID = "unknown"
		}

		ch <- result{&event, nil}
	}()

	select {
	case res := <-ch:
		return res.event, res.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("[session-parser] STDIN read timeout after %v", timeout)
	}
}
