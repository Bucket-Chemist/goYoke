// Package permissiongate implements the gogent-permission-gate hook.
// It gates tool invocations through the GOgent-Fortress TUI permission modal.
package permissiongate

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

const readTimeout = 5 * time.Second

// ToolEvent represents the subset of the PreToolUse JSON payload this hook needs.
type ToolEvent struct {
	ToolName  string                 `json:"tool_name"`
	ToolInput map[string]interface{} `json:"tool_input"`
	SessionID string                 `json:"session_id"`
}

// Main is the entrypoint for the gogent-permission-gate hook.
func Main() {
	event, rawInput, err := ParseStdin(os.Stdin)
	if err != nil {
		DenyWithReason("Failed to parse tool event")
		return
	}

	classification := DefaultPolicy.Classify(event.ToolName)

	switch classification {
	case ClassAutoAllow, ClassSkip:
		Allow()
		return
	}

	if event.SessionID != "" {
		if cached, ok := CheckCache(event.SessionID, event.ToolName); ok && cached == "allow_session" {
			Allow()
			return
		}
	}

	decision, err := RequestPermission(event.ToolName, rawInput, event.SessionID)
	if err != nil {
		DenyWithReason(err.Error())
		return
	}

	switch decision {
	case "allow":
		Allow()
	case "allow_session":
		if event.SessionID != "" {
			WriteCache(event.SessionID, event.ToolName, "allow_session")
		}
		Allow()
	default:
		command := ExtractCommand(event.ToolInput)
		if command != "" {
			DenyWithReason(fmt.Sprintf("User denied: Bash command: %s", command))
		} else {
			DenyWithReason(fmt.Sprintf("User denied: %s", event.ToolName))
		}
	}
}

// ParseStdin reads and parses the PreToolUse JSON from r with a timeout.
// Returns the parsed event and the raw tool_input bytes for forwarding to the TUI.
func ParseStdin(r io.Reader) (*ToolEvent, []byte, error) {
	type result struct {
		event    *ToolEvent
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

		var ev ToolEvent
		if err := json.Unmarshal(data, &ev); err != nil {
			ch <- result{nil, nil, fmt.Errorf("invalid JSON: %w", err)}
			return
		}

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

// ExtractCommand returns the "command" field from tool_input when present.
func ExtractCommand(toolInput map[string]interface{}) string {
	if cmd, ok := toolInput["command"].(string); ok {
		return cmd
	}
	return ""
}

// Allow writes the hook pass-through response ({}) to stdout.
func Allow() {
	fmt.Println("{}")
}

// DenyWithReason writes a block response with a human-readable reason.
func DenyWithReason(reason string) {
	output := map[string]interface{}{
		"decision": "block",
		"reason":   reason,
	}
	data, _ := json.Marshal(output)
	fmt.Println(string(data))
}
