package cli_test

import (
	"fmt"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/deprecated/internal/cli"
)

// ExampleClaudeProcess demonstrates basic usage of the Claude subprocess manager.
// This example uses the mock claude binary for demonstration.
func ExampleClaudeProcess() {
	// Create configuration
	cfg := cli.Config{
		ClaudePath: "./testdata/mock-claude",
		Verbose:    true,
	}

	// Create process
	proc, err := cli.NewClaudeProcess(cfg)
	if err != nil {
		fmt.Printf("Failed to create process: %v\n", err)
		return
	}

	// Start the subprocess
	if err := proc.Start(); err != nil {
		fmt.Printf("Failed to start process: %v\n", err)
		return
	}
	defer proc.Stop()

	// Wait for init event
	select {
	case event := <-proc.Events():
		fmt.Printf("Received event type: %s\n", event.Type)
	case err := <-proc.Errors():
		fmt.Printf("Error: %v\n", err)
	case <-time.After(1 * time.Second):
		fmt.Println("Timeout waiting for init event")
	}

	// Send a message
	if err := proc.Send("Hello Claude!"); err != nil {
		fmt.Printf("Failed to send message: %v\n", err)
		return
	}

	// Receive response
	select {
	case event := <-proc.Events():
		fmt.Printf("Received event type: %s\n", event.Type)
	case err := <-proc.Errors():
		fmt.Printf("Error: %v\n", err)
	case <-time.After(1 * time.Second):
		fmt.Println("Timeout waiting for response")
	}

	// Output:
	// Received event type: system
	// Received event type: assistant
}

// ExampleClaudeProcess_structured demonstrates sending structured messages.
func ExampleClaudeProcess_structured() {
	cfg := cli.Config{
		ClaudePath:     "./testdata/mock-claude",
		SessionID:      "example-session",
		IncludePartial: true,
	}

	proc, err := cli.NewClaudeProcess(cfg)
	if err != nil {
		fmt.Printf("Failed to create process: %v\n", err)
		return
	}

	if err := proc.Start(); err != nil {
		fmt.Printf("Failed to start process: %v\n", err)
		return
	}
	defer proc.Stop()

	fmt.Printf("Session ID: %s\n", proc.SessionID())
	fmt.Printf("Running: %v\n", proc.IsRunning())

	// Output:
	// Session ID: example-session
	// Running: true
}

// ExampleNDJSONWriter demonstrates NDJSON writing.
func ExampleNDJSONWriter() {
	// This would typically write to a file or network connection
	// For demo purposes, we're just showing the pattern
	fmt.Println("NDJSON Writer example - see streams_test.go for full usage")

	// Output:
	// NDJSON Writer example - see streams_test.go for full usage
}
