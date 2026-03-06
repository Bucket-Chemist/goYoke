package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

const defaultTimeout = 5 * time.Second

// configChangeEvent represents the Claude Code ConfigChange hook event.
type configChangeEvent struct {
	SessionID      string `json:"session_id"`
	HookEventName  string `json:"hook_event_name"`
	Source         string `json:"source"`
	FilePath       string `json:"file_path"`
	CWD            string `json:"cwd"`
	PermissionMode string `json:"permission_mode"`
}

// readEvent reads and parses the ConfigChange event from r with a timeout.
func readEvent(r io.Reader, timeout time.Duration) (*configChangeEvent, error) {
	type result struct {
		event *configChangeEvent
		err   error
	}
	ch := make(chan result, 1)
	go func() {
		data, err := io.ReadAll(r)
		if err != nil {
			ch <- result{nil, fmt.Errorf("read stdin: %w", err)}
			return
		}
		var event configChangeEvent
		if err := json.Unmarshal(data, &event); err != nil {
			ch <- result{nil, fmt.Errorf("parse JSON: %w", err)}
			return
		}
		ch <- result{&event, nil}
	}()

	select {
	case res := <-ch:
		return res.event, res.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("stdin read timeout after %v", timeout)
	}
}

// decision evaluates the event and returns whether to block.
// Returns (block bool, reason string).
func decision(event *configChangeEvent) (block bool, reason string) {
	// Always allow policy_settings and skills sources.
	if event.Source == "policy_settings" || event.Source == "skills" {
		return false, ""
	}

	// Block mid-session changes to settings.json files.
	if strings.HasSuffix(event.FilePath, "settings.json") {
		return true, "Mid-session settings changes blocked. Use /hooks menu to review."
	}

	return false, ""
}

func main() {
	event, err := readEvent(os.Stdin, defaultTimeout)
	if err != nil {
		// Parse failure: pass through silently so we never block on bad input.
		fmt.Fprintf(os.Stderr, "[gogent-config-guard] Warning: %v\n", err)
		os.Exit(0)
	}

	block, reason := decision(event)
	if block {
		fmt.Fprintf(os.Stderr, "%s\n", reason)
		os.Exit(2)
	}

	os.Exit(0)
}
