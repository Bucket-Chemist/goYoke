package session

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// SessionEvent represents SessionEnd hook event
type SessionEvent struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	HookEventName  string `json:"hook_event_name"`
	Timestamp      int64  `json:"timestamp,omitempty"`
}

// SessionMetrics represents collected session statistics
type SessionMetrics struct {
	ToolCalls         int    `json:"tool_calls"`
	ErrorsLogged      int    `json:"errors_logged"`
	RoutingViolations int    `json:"routing_violations"`
	SessionID         string `json:"session_id"`
	Duration          int64  `json:"duration_seconds,omitempty"`
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
