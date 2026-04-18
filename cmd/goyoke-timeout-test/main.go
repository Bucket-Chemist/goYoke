// goyoke-timeout-test: Test hook to determine Claude Code's timeout behavior
//
// This hook sleeps for a configurable duration before responding to test
// whether Claude Code has a timeout on hook responses.
//
// Usage:
//   1. Build: go build -o bin/goyoke-timeout-test ./cmd/goyoke-timeout-test
//   2. Add to hooks in settings.json (PreToolUse event)
//   3. Run Claude and trigger a tool call
//   4. Observe behavior - does Claude wait or timeout?

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

const (
	// SLEEP_DURATION is how long to wait before responding
	// Change this to test different durations
	SLEEP_DURATION = 30 * time.Second

	// LOG_FILE records timestamps for analysis
	LOG_FILE = "/tmp/goyoke-timeout-test.log"
)

type HookResponse struct {
	Allow             bool   `json:"allow"`
	AdditionalContext string `json:"additionalContext,omitempty"`
}

func log(msg string) {
	f, err := os.OpenFile(LOG_FILE, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "[%s] %s\n", time.Now().Format(time.RFC3339Nano), msg)
}

func main() {
	log("=== Hook started ===")

	// Read stdin (the hook event)
	log("Reading stdin...")
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		log(fmt.Sprintf("ERROR reading stdin: %v", err))
		// Return allow anyway so we don't break things
		json.NewEncoder(os.Stdout).Encode(HookResponse{Allow: true})
		return
	}
	log(fmt.Sprintf("Received %d bytes from stdin", len(input)))

	// Log what tool was called (if we can parse it)
	var event map[string]interface{}
	if err := json.Unmarshal(input, &event); err == nil {
		if toolName, ok := event["tool_name"].(string); ok {
			log(fmt.Sprintf("Tool: %s", toolName))
		}
	}

	// THE TEST: Sleep for SLEEP_DURATION
	log(fmt.Sprintf("Sleeping for %v...", SLEEP_DURATION))
	startSleep := time.Now()
	time.Sleep(SLEEP_DURATION)
	actualSleep := time.Since(startSleep)
	log(fmt.Sprintf("Woke up after %v", actualSleep))

	// Send response
	response := HookResponse{
		Allow:             true,
		AdditionalContext: fmt.Sprintf("[TIMEOUT TEST] Hook slept for %v before responding. If you see this, Claude waited!", actualSleep),
	}

	log("Sending response...")
	if err := json.NewEncoder(os.Stdout).Encode(response); err != nil {
		log(fmt.Sprintf("ERROR writing response: %v", err))
	}

	log("=== Hook completed ===")
}
