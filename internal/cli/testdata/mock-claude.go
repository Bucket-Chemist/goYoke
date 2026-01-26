package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Mock claude binary that simulates Claude CLI behavior for testing.
// Reads NDJSON from stdin, emits events on stdout.
//
// Usage: go build -o mock-claude mock-claude.go
//
// Behavior:
// 1. On startup, emit init event with session ID
// 2. Read stdin line by line
// 3. Parse as JSON message
// 4. Echo content as assistant event
// 5. Exit on EOF or SIGTERM

func main() {
	// Emit init event with session ID from args
	sessionID := "test-123"
	for i, arg := range os.Args {
		if arg == "--session-id" && i+1 < len(os.Args) {
			sessionID = os.Args[i+1]
			break
		}
	}

	initEvent := map[string]interface{}{
		"type":       "system",
		"subtype":    "init",
		"session_id": sessionID,
	}
	emitEvent(initEvent)

	// Read stdin and echo messages
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse input message
		var msg map[string]interface{}
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			// Emit error event on parse failure
			errorEvent := map[string]interface{}{
				"type":  "error",
				"error": fmt.Sprintf("parse error: %v", err),
			}
			emitEvent(errorEvent)
			continue
		}

		// Extract content from message
		content, ok := msg["content"].(string)
		if !ok {
			content = fmt.Sprintf("%v", msg)
		}

		// Echo as assistant event
		response := map[string]interface{}{
			"type": "assistant",
			"message": map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": fmt.Sprintf("Echo: %s", content),
					},
				},
			},
		}
		emitEvent(response)
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		errorEvent := map[string]interface{}{
			"type":  "error",
			"error": fmt.Sprintf("scanner error: %v", err),
		}
		emitEvent(errorEvent)
		os.Exit(1)
	}
}

// emitEvent writes a JSON event to stdout with trailing newline
func emitEvent(event map[string]interface{}) {
	data, err := json.Marshal(event)
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal error: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

// parseArgs extracts session ID from command line args
func parseArgs(args []string) string {
	for i, arg := range args {
		if strings.HasPrefix(arg, "--session-id") {
			if strings.Contains(arg, "=") {
				parts := strings.SplitN(arg, "=", 2)
				return parts[1]
			}
			if i+1 < len(args) {
				return args[i+1]
			}
		}
	}
	return "test-123"
}
