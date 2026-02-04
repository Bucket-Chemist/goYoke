package cli_test

import (
	"fmt"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/deprecated/internal/cli"
)

// ExampleParseError demonstrates basic error parsing and classification
func ExampleParseError() {
	// Simulate stderr output from Claude CLI
	stderr := "Error: 429 Too Many Requests. Retry-After: 60"
	exitCode := 1

	// Parse the error
	err := cli.ParseError(stderr, exitCode)

	// Check error type and decide on action
	fmt.Printf("Error type: %s\n", err.Type)
	fmt.Printf("Is retryable: %v\n", err.IsRetryable())
	fmt.Printf("Retry delay: %v\n", err.RetryDelay())

	// Output:
	// Error type: rate_limit
	// Is retryable: true
	// Retry delay: 1m0s
}

// ExampleClaudeError_IsRetryable demonstrates intelligent retry logic
func ExampleClaudeError_IsRetryable() {
	scenarios := []string{
		"Error: 429 Rate limit exceeded",
		"Error: Network connection refused",
		"Error: Invalid API key",
		"Error: Operation timed out",
	}

	for _, stderr := range scenarios {
		err := cli.ParseError(stderr, 1)
		action := "give up"
		if err.IsRetryable() {
			delay := err.RetryDelay()
			action = fmt.Sprintf("retry after %v", delay)
		}
		fmt.Printf("%s -> %s\n", err.Type, action)
	}

	// Output:
	// rate_limit -> retry after 1m0s
	// network -> retry after 5s
	// authentication -> give up
	// timeout -> retry after 5s
}

// ExampleClaudeError_RetryDelay demonstrates dynamic retry delay extraction
func ExampleClaudeError_RetryDelay() {
	// Rate limit with explicit retry-after header
	err1 := cli.ParseError("Error: 429. Retry-After: 30", 1)
	fmt.Printf("Rate limit with header: %v\n", err1.RetryDelay())

	// Rate limit without header (uses default)
	err2 := cli.ParseError("Error: Rate limit exceeded", 1)
	fmt.Printf("Rate limit default: %v\n", err2.RetryDelay())

	// Network error (fixed delay)
	err3 := cli.ParseError("Error: Connection refused", 1)
	fmt.Printf("Network error: %v\n", err3.RetryDelay())

	// Non-retryable error (zero delay)
	err4 := cli.ParseError("Error: Invalid API key", 1)
	fmt.Printf("Auth error: %v\n", err4.RetryDelay())

	// Output:
	// Rate limit with header: 30s
	// Rate limit default: 1m0s
	// Network error: 5s
	// Auth error: 0s
}

// ExampleClaudeError integration demonstrates how to use in subprocess restart logic
func ExampleClaudeError_integration() {
	// Simulate subprocess exit with stderr
	stderr := "Error: Operation timed out"
	exitCode := 1

	// Parse the error
	claudeErr := cli.ParseError(stderr, exitCode)

	// Make restart decision based on error type
	fmt.Printf("Process failed with: %s\n", claudeErr.Error())

	if claudeErr.IsRetryable() {
		delay := claudeErr.RetryDelay()
		fmt.Printf("Will restart after %v\n", delay)
		// In real code: time.Sleep(delay) then restart
	} else {
		fmt.Printf("Non-retryable error, giving up\n")
		// In real code: log error and notify user
	}

	// Output:
	// Process failed with: claude error (type=timeout, code=1): Operation timed out
	// Will restart after 5s
}

// ExampleClaudeError_mcpConnectionCheck demonstrates MCP-specific retry logic
func ExampleClaudeError_mcpConnectionCheck() {
	// MCP connection error - retryable (but matches network pattern first)
	err1 := cli.ParseError("Error: MCP server connection failed", 1)
	fmt.Printf("MCP connection error retryable: %v (delay: %v)\n",
		err1.IsRetryable(), err1.RetryDelay())

	// MCP config error - not retryable
	err2 := cli.ParseError("Error: MCP invalid configuration", 1)
	fmt.Printf("MCP config error retryable: %v\n", err2.IsRetryable())

	// Output:
	// MCP connection error retryable: true (delay: 5s)
	// MCP config error retryable: false
}

// ExampleClaudeError_errorChaining demonstrates error wrapping support
func ExampleClaudeError_errorChaining() {
	// Create a ClaudeError with an original error
	original := fmt.Errorf("connection refused")
	stderr := "Error: Network error occurred"

	claudeErr := cli.ParseError(stderr, 1)
	claudeErr.Original = original

	// Use error chain unwrapping
	fmt.Printf("Error: %s\n", claudeErr.Error())
	if unwrapped := claudeErr.Unwrap(); unwrapped != nil {
		fmt.Printf("Original: %s\n", unwrapped.Error())
	}

	// Output:
	// Error: claude error (type=network, code=1): Network error
	// Original: connection refused
}

// Example integration point for RestartPolicy
type exampleRestartDecision struct {
	shouldRestart bool
	delay         time.Duration
	reason        string
}

func decideRestart(stderr string, exitCode int, attemptCount int) exampleRestartDecision {
	err := cli.ParseError(stderr, exitCode)

	// Don't retry non-retryable errors
	if !err.IsRetryable() {
		return exampleRestartDecision{
			shouldRestart: false,
			reason:        fmt.Sprintf("non-retryable: %s", err.Type),
		}
	}

	// Don't retry beyond max attempts
	maxAttempts := 3
	if attemptCount >= maxAttempts {
		return exampleRestartDecision{
			shouldRestart: false,
			reason:        fmt.Sprintf("max attempts reached (%d)", maxAttempts),
		}
	}

	// Retry with appropriate delay
	return exampleRestartDecision{
		shouldRestart: true,
		delay:         err.RetryDelay(),
		reason:        fmt.Sprintf("retrying %s (attempt %d/%d)", err.Type, attemptCount+1, maxAttempts),
	}
}

func Example_restartDecision() {
	// Test various scenarios
	scenarios := []struct {
		stderr  string
		attempt int
	}{
		{"Error: 429 Rate limit", 1},
		{"Error: Connection refused", 2},
		{"Error: Invalid API key", 1},
		{"Error: Timeout", 3}, // Max attempts
	}

	for _, s := range scenarios {
		decision := decideRestart(s.stderr, 1, s.attempt)
		if decision.shouldRestart {
			fmt.Printf("Restart: yes, delay=%v, %s\n",
				decision.delay, decision.reason)
		} else {
			fmt.Printf("Restart: no, %s\n", decision.reason)
		}
	}

	// Output:
	// Restart: yes, delay=1m0s, retrying rate_limit (attempt 2/3)
	// Restart: yes, delay=5s, retrying network (attempt 3/3)
	// Restart: no, non-retryable: authentication
	// Restart: no, max attempts reached (3)
}
