package main

import (
	"fmt"
	"os"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/deprecated/internal/cli"
)

// Demo program showing basic usage of the Claude subprocess manager.
// Uses the mock claude binary for demonstration.
//
// Usage:
//   cd internal/cli/testdata
//   go build -o mock-claude mock-claude.go  # Build mock first
//   go run demo.go

func main() {
	fmt.Println("=== Claude Subprocess Manager Demo ===\n")

	// Create configuration
	cfg := cli.Config{
		ClaudePath:     "./mock-claude",
		Verbose:        true,
		IncludePartial: true,
	}

	// Create process
	proc, err := cli.NewClaudeProcess(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create process: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Session ID: %s\n", proc.SessionID())
	fmt.Printf("Process created. Starting...\n\n")

	// Start subprocess
	if err := proc.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start process: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		fmt.Println("\nStopping process...")
		proc.Stop()
		fmt.Println("Process stopped.")
	}()

	fmt.Printf("Process running: %v\n\n", proc.IsRunning())

	// Wait for init event
	fmt.Println("Waiting for init event...")
	select {
	case event := <-proc.Events():
		fmt.Printf("✓ Received init event: type=%s\n\n", event.Type)
	case err := <-proc.Errors():
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	case <-time.After(2 * time.Second):
		fmt.Fprintf(os.Stderr, "Timeout waiting for init event\n")
		return
	}

	// Send test messages
	messages := []string{
		"Hello Claude!",
		"What is 2+2?",
		"Explain quantum computing",
	}

	for i, msg := range messages {
		fmt.Printf("Sending message %d: %q\n", i+1, msg)
		if err := proc.Send(msg); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to send: %v\n", err)
			return
		}

		// Wait for response
		select {
		case event := <-proc.Events():
			fmt.Printf("✓ Received response: type=%s\n", event.Type)
			if content, ok := event.RawData["message"].(map[string]interface{}); ok {
				fmt.Printf("  Content: %v\n", content)
			}
		case err := <-proc.Errors():
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		case <-time.After(2 * time.Second):
			fmt.Fprintf(os.Stderr, "Timeout waiting for response\n")
			return
		}
		fmt.Println()
	}

	fmt.Println("Demo complete!")
}
