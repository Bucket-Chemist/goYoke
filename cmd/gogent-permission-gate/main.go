// Command gogent-permission-gate is a PreToolUse hook binary that gates Bash
// (and other configurable) tool invocations through the GOgent-Fortress TUI
// permission modal.
//
// The binary reads a single JSON object from stdin, classifies the tool, and
// either auto-allows or contacts the TUI via Unix domain socket for a user
// decision.
//
// Exit codes follow the Claude Code hook contract:
//
//	0  — hook completed; stdout contains the response JSON
//	1  — internal error; Claude Code falls back to default behaviour
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

const readTimeout = 5 * time.Second

// toolEvent represents the subset of the PreToolUse JSON payload that this
// hook needs.  The full schema contains additional fields that are intentionally
// ignored here.
type toolEvent struct {
	ToolName  string                 `json:"tool_name"`
	ToolInput map[string]interface{} `json:"tool_input"`
	SessionID string                 `json:"session_id"`
}

func main() {
	event, rawInput, err := parseStdin(os.Stdin)
	if err != nil {
		denyWithReason("Failed to parse tool event")
		return
	}

	classification := defaultPolicy.Classify(event.ToolName)

	switch classification {
	case classAutoAllow, classSkip:
		allow()
		return
	}

	// classNeedsApproval path — check session cache first.
	if event.SessionID != "" {
		if cached, ok := CheckCache(event.SessionID, event.ToolName); ok && cached == "allow_session" {
			allow()
			return
		}
	}

	// Contact the TUI bridge for a live decision.
	decision, err := RequestPermission(event.ToolName, rawInput, event.SessionID)
	if err != nil {
		denyWithReason(err.Error())
		return
	}

	switch decision {
	case "allow":
		allow()
	case "allow_session":
		if event.SessionID != "" {
			WriteCache(event.SessionID, event.ToolName, "allow_session")
		}
		allow()
	default:
		// "deny" or any unknown decision is treated as a block.
		command := extractCommand(event.ToolInput)
		if command != "" {
			denyWithReason(fmt.Sprintf("User denied: Bash command: %s", command))
		} else {
			denyWithReason(fmt.Sprintf("User denied: %s", event.ToolName))
		}
	}
}

// parseStdin reads and parses the PreToolUse JSON from r with a timeout.
// It returns the parsed event and the raw tool_input bytes for forwarding to
// the TUI.
//
// The read runs in a goroutine so we can apply a timeout (stdin has no
// deadline support).  If the timeout fires the goroutine is orphaned, but
// this is acceptable: the binary exits immediately after parseStdin returns
// an error, so the goroutine is cleaned up by process exit.
func parseStdin(r io.Reader) (*toolEvent, []byte, error) {
	type result struct {
		event    *toolEvent
		rawInput []byte
		err      error
	}

	ch := make(chan result, 1)

	go func() {
		data, err := io.ReadAll(r)
		if err != nil {
			ch <- result{nil, nil, err}
			return
		}
		if len(data) == 0 {
			ch <- result{nil, nil, fmt.Errorf("empty stdin")}
			return
		}

		var ev toolEvent
		if err := json.Unmarshal(data, &ev); err != nil {
			ch <- result{nil, nil, fmt.Errorf("invalid JSON: %w", err)}
			return
		}

		// Re-marshal tool_input for forwarding to the TUI.  We cannot use
		// the original bytes directly because the outer JSON may contain more
		// fields; we only want the tool_input value.
		rawInput, err := json.Marshal(ev.ToolInput)
		if err != nil {
			rawInput = []byte("{}")
		}

		ch <- result{&ev, rawInput, nil}
	}()

	select {
	case res := <-ch:
		return res.event, res.rawInput, res.err
	case <-time.After(readTimeout):
		return nil, nil, fmt.Errorf("stdin read timeout")
	}
}

// extractCommand returns the "command" field from tool_input when present.
// Used to build a more descriptive deny message for Bash invocations.
func extractCommand(toolInput map[string]interface{}) string {
	if cmd, ok := toolInput["command"].(string); ok {
		return cmd
	}
	return ""
}

// allow writes the hook pass-through response ({}) to stdout.
func allow() {
	fmt.Println("{}")
}

// denyWithReason writes a block response with a human-readable reason.
func denyWithReason(reason string) {
	output := map[string]interface{}{
		"decision": "block",
		"reason":   reason,
	}
	data, _ := json.Marshal(output)
	fmt.Println(string(data))
}
