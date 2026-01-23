package integration

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestSessionStartIntegration tests full workflow with real binary
func TestSessionStartIntegration_StartupWithGoProject(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Build binary
	buildBinary(t)
	defer cleanupBinary()

	// Create Go project structure
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)

	// Initialize git repo
	initGitRepo(t, tmpDir)

	// Run binary
	input := `{"type":"startup","session_id":"int-test-001","hook_event_name":"SessionStart"}`
	output := runBinary(t, input, tmpDir)

	// Validate response
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(output), &resp); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	hookOutput := resp["hookSpecificOutput"].(map[string]interface{})
	context := hookOutput["additionalContext"].(string)

	// Verify startup indicators
	assertContains(t, context, "startup", "Should indicate startup session")

	// Verify project detection
	assertContains(t, context, "go", "Should detect Go project")

	// Verify git info
	assertContains(t, context, "GIT:", "Should include git info")
}

func TestSessionStartIntegration_ResumeWithHandoff(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	buildBinary(t)
	defer cleanupBinary()

	// Create project with handoff
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	handoffContent := `# Session Handoff

## Last Session
Implemented feature XYZ.

## Next Steps
- Complete testing
- Update documentation
`
	os.WriteFile(filepath.Join(memoryDir, "last-handoff.md"), []byte(handoffContent), 0644)

	// Run as resume session
	input := `{"type":"resume","session_id":"int-test-002","hook_event_name":"SessionStart"}`
	output := runBinary(t, input, tmpDir)

	var resp map[string]interface{}
	json.Unmarshal([]byte(output), &resp)

	hookOutput := resp["hookSpecificOutput"].(map[string]interface{})
	context := hookOutput["additionalContext"].(string)

	// Verify resume with handoff
	assertContains(t, context, "resume", "Should indicate resume session")
	assertContains(t, context, "PREVIOUS SESSION HANDOFF", "Should include handoff header")
	assertContains(t, context, "feature XYZ", "Should include handoff content")
}

func TestSessionStartIntegration_WithPendingLearnings(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	buildBinary(t)
	defer cleanupBinary()

	// Create project with pending learnings
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	pendingContent := `{"ts":1234567890,"file":"test.go","error_type":"type_mismatch"}
{"ts":1234567891,"file":"main.go","error_type":"nil_pointer"}
`
	os.WriteFile(filepath.Join(memoryDir, "pending-learnings.jsonl"), []byte(pendingContent), 0644)

	input := `{"type":"startup","session_id":"int-test-003","hook_event_name":"SessionStart"}`
	output := runBinary(t, input, tmpDir)

	var resp map[string]interface{}
	json.Unmarshal([]byte(output), &resp)

	hookOutput := resp["hookSpecificOutput"].(map[string]interface{})
	context := hookOutput["additionalContext"].(string)

	// Verify pending learnings warning
	assertContains(t, context, "PENDING LEARNINGS", "Should warn about pending learnings")
	assertContains(t, context, "2 sharp edge", "Should count sharp edges correctly")
}

func TestSessionStartIntegration_MultiLanguageProject(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	buildBinary(t)
	defer cleanupBinary()

	// Create project with multiple language indicators (Go should win)
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), []byte("[project]"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte("{}"), 0644)

	input := `{"type":"startup","session_id":"int-test-004","hook_event_name":"SessionStart"}`
	output := runBinary(t, input, tmpDir)

	var resp map[string]interface{}
	json.Unmarshal([]byte(output), &resp)

	hookOutput := resp["hookSpecificOutput"].(map[string]interface{})
	context := hookOutput["additionalContext"].(string)

	// Go should be detected (highest priority)
	assertContains(t, context, "PROJECT TYPE: go", "Go should have priority")
}

// Helper functions

func buildBinary(t *testing.T) {
	t.Helper()
	cmd := exec.Command("go", "build", "-o", "test-load-context", "../../cmd/gogent-load-context")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
}

func cleanupBinary() {
	os.Remove("test-load-context")
}

func runBinary(t *testing.T, input string, projectDir string) string {
	t.Helper()

	cmd := exec.Command("./test-load-context")
	cmd.Stdin = bytes.NewBufferString(input)
	cmd.Env = append(os.Environ(), "GOGENT_PROJECT_DIR="+projectDir)

	output, err := cmd.Output()
	if err != nil {
		// Check if it's a non-zero exit but still has output
		if len(output) > 0 {
			return string(output)
		}
		t.Fatalf("Binary failed: %v", err)
	}
	return string(output)
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "add", "."},
		{"git", "commit", "-m", "initial"},
	}

	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Logf("Git command %v failed (may be expected): %v", args, err)
		}
	}
}

func assertContains(t *testing.T, haystack, needle, message string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("%s: expected to find %q in output", message, needle)
	}
}
