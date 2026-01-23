package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/session"
)

func TestMain_Startup(t *testing.T) {
	// Build the binary first
	buildCmd := exec.Command("go", "build", "-o", "test-gogent-load-context", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("test-gogent-load-context")

	// Prepare input
	input := `{"type":"startup","session_id":"test-123","hook_event_name":"SessionStart"}`

	// Run binary
	cmd := exec.Command("./test-gogent-load-context")
	cmd.Stdin = bytes.NewBufferString(input)

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Binary execution failed: %v", err)
	}

	// Verify valid JSON output
	var response session.SessionStartResponse
	if err := json.Unmarshal(output, &response); err != nil {
		t.Fatalf("Invalid JSON output: %v. Output: %s", err, string(output))
	}

	// Verify content
	if response.HookSpecificOutput.HookEventName != "SessionStart" {
		t.Errorf("Expected hookEventName 'SessionStart', got: %s", response.HookSpecificOutput.HookEventName)
	}

	if !strings.Contains(response.HookSpecificOutput.AdditionalContext, "startup") {
		t.Error("Response should indicate startup session")
	}
}

func TestMain_Resume(t *testing.T) {
	// Build binary
	buildCmd := exec.Command("go", "build", "-o", "test-gogent-load-context", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("test-gogent-load-context")

	// Create temp project with handoff
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	handoffContent := "# Test Handoff\n\nPrevious session info."
	os.WriteFile(filepath.Join(memoryDir, "last-handoff.md"), []byte(handoffContent), 0644)

	// Prepare input
	input := `{"type":"resume","session_id":"test-456","hook_event_name":"SessionStart"}`

	// Run binary with project directory
	cmd := exec.Command("./test-gogent-load-context")
	cmd.Stdin = bytes.NewBufferString(input)
	cmd.Env = append(os.Environ(), "GOGENT_PROJECT_DIR="+tmpDir)

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Binary execution failed: %v", err)
	}

	// Verify response includes handoff
	var response session.SessionStartResponse
	json.Unmarshal(output, &response)

	if !strings.Contains(response.HookSpecificOutput.AdditionalContext, "resume") {
		t.Error("Response should indicate resume session")
	}

	if !strings.Contains(response.HookSpecificOutput.AdditionalContext, "PREVIOUS SESSION HANDOFF") {
		t.Error("Resume response should include handoff")
	}
}

func TestMain_InvalidInput(t *testing.T) {
	// Build binary
	buildCmd := exec.Command("go", "build", "-o", "test-gogent-load-context", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("test-gogent-load-context")

	// Invalid JSON input
	input := "not valid json"

	cmd := exec.Command("./test-gogent-load-context")
	cmd.Stdin = bytes.NewBufferString(input)

	output, _ := cmd.Output() // Exit code 1 expected

	// Should still output valid JSON error
	var response session.SessionStartResponse
	if err := json.Unmarshal(output, &response); err != nil {
		t.Fatalf("Error response should be valid JSON: %v", err)
	}

	if !strings.Contains(response.HookSpecificOutput.AdditionalContext, "ERROR") {
		t.Error("Should indicate error")
	}
}

func TestMain_ToolCounterInitialized(t *testing.T) {
	// Build binary
	buildCmd := exec.Command("go", "build", "-o", "test-gogent-load-context", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("test-gogent-load-context")

	// Use temp XDG directory
	tmpDir := t.TempDir()

	input := `{"type":"startup","session_id":"test-789","hook_event_name":"SessionStart"}`

	cmd := exec.Command("./test-gogent-load-context")
	cmd.Stdin = bytes.NewBufferString(input)
	// Clear existing XDG vars and set only XDG_CACHE_HOME to ensure predictable behavior
	cmd.Env = []string{
		"XDG_CACHE_HOME=" + tmpDir,
		"PATH=" + os.Getenv("PATH"),
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Binary execution failed: %v\nOutput: %s", err, string(output))
	}

	// Verify tool counter was created
	counterPath := filepath.Join(tmpDir, "gogent", "tool-counter")
	if _, err := os.Stat(counterPath); os.IsNotExist(err) {
		t.Errorf("Tool counter file should be created at %s", counterPath)
		// Debug: check what was created
		if entries, err := os.ReadDir(tmpDir); err == nil {
			t.Logf("Contents of tmpDir: %v", entries)
			if len(entries) > 0 {
				for _, entry := range entries {
					t.Logf("  - %s (dir=%v)", entry.Name(), entry.IsDir())
					if entry.IsDir() {
						subEntries, _ := os.ReadDir(filepath.Join(tmpDir, entry.Name()))
						for _, sub := range subEntries {
							t.Logf("    - %s", sub.Name())
						}
					}
				}
			}
		}
	}
}
