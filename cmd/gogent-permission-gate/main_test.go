package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// parseStdin tests
// =============================================================================

func TestParseStdin_ValidBashEvent(t *testing.T) {
	input := `{
		"tool_name": "Bash",
		"session_id": "test-session-123",
		"tool_input": {"command": "ls -la /tmp", "description": "List tmp"}
	}`

	event, rawInput, err := parseStdin(strings.NewReader(input))

	require.NoError(t, err)
	assert.Equal(t, "Bash", event.ToolName)
	assert.Equal(t, "test-session-123", event.SessionID)
	assert.NotEmpty(t, rawInput)

	// Verify rawInput is valid JSON containing the command.
	var decoded map[string]interface{}
	require.NoError(t, json.Unmarshal(rawInput, &decoded))
	assert.Equal(t, "ls -la /tmp", decoded["command"])
}

func TestParseStdin_ValidNonBashEvent(t *testing.T) {
	input := `{
		"tool_name": "Write",
		"session_id": "session-abc",
		"tool_input": {"file_path": "/tmp/test.go", "content": "package main"}
	}`

	event, _, err := parseStdin(strings.NewReader(input))

	require.NoError(t, err)
	assert.Equal(t, "Write", event.ToolName)
	assert.Equal(t, "session-abc", event.SessionID)
}

func TestParseStdin_EmptyStdin(t *testing.T) {
	_, _, err := parseStdin(bytes.NewReader([]byte{}))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty stdin")
}

func TestParseStdin_MalformedJSON(t *testing.T) {
	_, _, err := parseStdin(strings.NewReader(`{not valid json`))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON")
}

func TestParseStdin_MissingToolInput(t *testing.T) {
	// tool_input absent — should still parse; ToolInput will be nil map.
	input := `{"tool_name": "Bash", "session_id": "s1"}`
	event, rawInput, err := parseStdin(strings.NewReader(input))

	require.NoError(t, err)
	assert.Equal(t, "Bash", event.ToolName)
	// rawInput should still be valid JSON.
	var decoded interface{}
	require.NoError(t, json.Unmarshal(rawInput, &decoded))
}

// =============================================================================
// Policy classification tests
// =============================================================================

func TestPolicy_BashNeedsApproval(t *testing.T) {
	assert.Equal(t, classNeedsApproval, defaultPolicy.Classify("Bash"))
}

func TestPolicy_WriteAutoAllow(t *testing.T) {
	assert.Equal(t, classAutoAllow, defaultPolicy.Classify("Write"))
}

func TestPolicy_ReadAutoAllow(t *testing.T) {
	assert.Equal(t, classAutoAllow, defaultPolicy.Classify("Read"))
}

func TestPolicy_TaskSkip(t *testing.T) {
	assert.Equal(t, classSkip, defaultPolicy.Classify("Task"))
}

func TestPolicy_AgentSkip(t *testing.T) {
	assert.Equal(t, classSkip, defaultPolicy.Classify("Agent"))
}

func TestPolicy_UnknownToolDefaultsAutoAllow(t *testing.T) {
	assert.Equal(t, classAutoAllow, defaultPolicy.Classify("UnknownTool"))
}

func TestPolicy_AllAutoAllowTools(t *testing.T) {
	autoAllowTools := []string{
		"Read", "Glob", "Grep", "TodoWrite", "EnterPlanMode",
		"ExitPlanMode", "WebSearch", "WebFetch", "ToolSearch",
		"AskUserQuestion", "Skill", "Write", "Edit", "NotebookEdit",
	}
	for _, tool := range autoAllowTools {
		t.Run(tool, func(t *testing.T) {
			assert.Equal(t, classAutoAllow, defaultPolicy.Classify(tool))
		})
	}
}

// =============================================================================
// Session cache tests
// =============================================================================

func TestCache_WriteAndCheckAllowSession(t *testing.T) {
	// Redirect cache to a temp dir so tests are isolated.
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)

	sessionID := "cache-test-session"
	toolName := "Bash"

	// Initially absent.
	decision, ok := CheckCache(sessionID, toolName)
	assert.False(t, ok)
	assert.Empty(t, decision)

	// Write allow_session.
	WriteCache(sessionID, toolName, "allow_session")

	// Should now be present.
	decision, ok = CheckCache(sessionID, toolName)
	assert.True(t, ok)
	assert.Equal(t, "allow_session", decision)
}

func TestCache_DifferentSessionsAreIsolated(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)

	WriteCache("session-A", "Bash", "allow_session")

	// Session B should have an empty cache.
	_, ok := CheckCache("session-B", "Bash")
	assert.False(t, ok)
}

func TestCache_CorruptCacheReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)

	sessionID := "corrupt-session"

	// Write garbage bytes to the cache file.
	path := cachePath(sessionID)
	require.NoError(t, os.WriteFile(path, []byte("NOT JSON"), 0600))

	// Should not panic or error — returns empty.
	decision, ok := CheckCache(sessionID, "Bash")
	assert.False(t, ok)
	assert.Empty(t, decision)
}

func TestCache_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)

	sessionID := "perm-test-session"
	WriteCache(sessionID, "Bash", "allow_session")

	info, err := os.Stat(cachePath(sessionID))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

// =============================================================================
// Auto-allow passthrough integration test
// =============================================================================

func TestMain_AutoAllowWritesTool(t *testing.T) {
	// Capture stdout by redirecting os.Stdout during the call.
	// We test the allow() helper directly — no UDS needed.
	buf := captureStdout(t, func() {
		allow()
	})
	assert.Equal(t, "{}\n", buf)
}

func TestMain_DenyWithReason(t *testing.T) {
	buf := captureStdout(t, func() {
		denyWithReason("User denied: Bash")
	})
	var out map[string]string
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(buf)), &out))
	assert.Equal(t, "block", out["decision"])
	assert.Equal(t, "User denied: Bash", out["reason"])
}

// =============================================================================
// extractCommand tests
// =============================================================================

func TestExtractCommand_Present(t *testing.T) {
	input := map[string]interface{}{"command": "ls -la", "description": "list"}
	assert.Equal(t, "ls -la", extractCommand(input))
}

func TestExtractCommand_Absent(t *testing.T) {
	input := map[string]interface{}{"file_path": "/tmp/x"}
	assert.Empty(t, extractCommand(input))
}

// =============================================================================
// helpers
// =============================================================================

// captureStdout redirects os.Stdout to a pipe, runs fn, and returns the
// captured output as a string.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = origStdout

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	r.Close()

	return fmt.Sprintf("%s", buf[:n])
}
