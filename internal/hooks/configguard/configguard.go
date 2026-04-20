// Package configguard implements the goyoke-config-guard hook.
// It blocks mid-session changes to settings.json files.
package configguard

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// DefaultTimeout is the read timeout for stdin events.
const DefaultTimeout = 5 * time.Second

// ConfigChangeEvent represents the Claude Code ConfigChange hook event.
type ConfigChangeEvent struct {
	SessionID      string `json:"session_id"`
	HookEventName  string `json:"hook_event_name"`
	Source         string `json:"source"`
	FilePath       string `json:"file_path"`
	CWD            string `json:"cwd"`
	PermissionMode string `json:"permission_mode"`
}

// ReadEvent reads and parses the ConfigChange event from r with a timeout.
func ReadEvent(r io.Reader, timeout time.Duration) (*ConfigChangeEvent, error) {
	type result struct {
		event *ConfigChangeEvent
		err   error
	}
	ch := make(chan result, 1)
	go func() {
		data, err := io.ReadAll(r)
		if err != nil {
			ch <- result{nil, fmt.Errorf("read stdin: %w", err)}
			return
		}
		var event ConfigChangeEvent
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

// Decision evaluates the event and returns whether to block.
// Returns (block bool, reason string).
func Decision(event *ConfigChangeEvent) (block bool, reason string) {
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

// Main is the entrypoint for the goyoke-config-guard hook.
func Main() {
	event, err := ReadEvent(os.Stdin, DefaultTimeout)
	if err != nil {
		// Parse failure: pass through silently so we never block on bad input.
		fmt.Fprintf(os.Stderr, "[goyoke-config-guard] Warning: %v\n", err)
		os.Exit(0)
	}

	block, reason := Decision(event)
	if block {
		fmt.Fprintf(os.Stderr, "%s\n", reason)
		os.Exit(2)
	}

	os.Exit(0)
}
