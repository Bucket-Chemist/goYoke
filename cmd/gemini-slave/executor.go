package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// GeminiExecutor handles execution of the Gemini CLI with proper configuration.
type GeminiExecutor struct {
	Model        string        // Model to use (e.g., gemini-3-flash-preview)
	OutputFormat string        // Output format: "", "json", "stream-json"
	Timeout      time.Duration // Execution timeout
	HomeOverride string        // Override HOME to ~/.gemini-slave for config isolation
}

// NewGeminiExecutor creates a new executor with defaults.
func NewGeminiExecutor(model string, timeout time.Duration) *GeminiExecutor {
	home := os.Getenv("HOME")
	homeOverride := home + "/.gemini-slave"

	return &GeminiExecutor{
		Model:        model,
		Timeout:      timeout,
		HomeOverride: homeOverride,
	}
}

// Execute runs the Gemini CLI with the provided prompt.
// Returns the output or an error with context.
func (e *GeminiExecutor) Execute(prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), e.Timeout)
	defer cancel()

	// Build Gemini CLI arguments
	args := []string{"-m", e.Model, "--yolo", "-e", "none"}
	if e.OutputFormat != "" {
		args = append(args, "-o", e.OutputFormat)
	}

	cmd := exec.CommandContext(ctx, "gemini", args...)
	cmd.Stdin = strings.NewReader(prompt)

	// Override HOME so gemini CLI reads config from ~/.gemini-slave/.gemini/
	// This isolates gemini-slave's Gemini config from the user's main config
	env := os.Environ()
	env = append(env, "HOME="+e.HomeOverride)
	cmd.Env = env

	// Execute and capture output
	output, err := cmd.CombinedOutput()

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("gemini execution timed out after %v", e.Timeout)
	}

	// Check for execution errors
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("gemini failed (exit %d): %s", exitErr.ExitCode(), string(output))
		}
		return "", fmt.Errorf("gemini execution failed: %w", err)
	}

	return string(output), nil
}
