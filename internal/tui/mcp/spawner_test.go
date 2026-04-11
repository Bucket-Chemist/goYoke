package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// acFileExists
// ---------------------------------------------------------------------------

func TestAcFileExists_FileInDirectory(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "implementation-plan.json")
	require.NoError(t, os.WriteFile(file, []byte("{}"), 0o644))

	criterion := "implementation-plan.json written to " + dir + "/"
	assert.True(t, acFileExists(criterion), "should detect file at dir/filename")
}

func TestAcFileExists_FullFilePath(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "plan.json")
	require.NoError(t, os.WriteFile(file, []byte("{}"), 0o644))

	criterion := "plan.json written to " + file
	assert.True(t, acFileExists(criterion), "should detect file when target is the full path")
}

func TestAcFileExists_DirectoryWithFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "output.md"), []byte("hi"), 0o644))

	criterion := "written to " + dir + "/"
	assert.True(t, acFileExists(criterion), "should detect non-empty directory")
}

func TestAcFileExists_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	criterion := "written to " + dir + "/"
	assert.False(t, acFileExists(criterion), "empty directory should not satisfy criterion")
}

func TestAcFileExists_MissingFile(t *testing.T) {
	dir := t.TempDir()
	criterion := "missing-file.json written to " + dir + "/"
	assert.False(t, acFileExists(criterion), "absent file should not satisfy criterion")
}

func TestAcFileExists_NoPatternsInText(t *testing.T) {
	assert.False(t, acFileExists("implement the routing module"), "text without path pattern should return false")
	assert.False(t, acFileExists(""), "empty criterion should return false")
}

// ---------------------------------------------------------------------------
// verifyACDeliverables
// ---------------------------------------------------------------------------

func TestVerifyACDeliverables_MarksCompletedWhenFileExists(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "plan.json")
	require.NoError(t, os.WriteFile(file, []byte("{}"), 0o644))

	acState := []state.AcceptanceCriterion{
		{Text: "plan.json written to " + dir + "/", Completed: false},
		{Text: "unrelated criterion", Completed: false},
	}

	result := verifyACDeliverables("agent-1", acState, nil)
	require.Len(t, result, 2)
	assert.True(t, result[0].Completed, "file-backed criterion should be marked completed")
	assert.False(t, result[1].Completed, "unrelated criterion should remain incomplete")
}

func TestVerifyACDeliverables_SkipsAlreadyCompleted(t *testing.T) {
	acState := []state.AcceptanceCriterion{
		{Text: "written to /nonexistent/dir/", Completed: true},
	}
	result := verifyACDeliverables("agent-1", acState, nil)
	assert.True(t, result[0].Completed, "already-completed criterion must not be touched")
}

func TestVerifyACDeliverables_EmptyACState(t *testing.T) {
	result := verifyACDeliverables("agent-1", nil, nil)
	assert.Nil(t, result)
}

func TestVerifyACDeliverables_OriginalNotMutated(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "out.json"), []byte("{}"), 0o644))

	original := []state.AcceptanceCriterion{
		{Text: "out.json written to " + dir + "/", Completed: false},
	}
	_ = verifyACDeliverables("agent-1", original, nil)
	assert.False(t, original[0].Completed, "original slice must not be mutated")
}

func TestCLIOutputCollector_PreservesResultAfterTruncation(t *testing.T) {
	collector := newCLIOutputCollector(1024)

	filler := map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"id":    "m1",
			"type":  "message",
			"role":  "assistant",
			"model": "m",
			"content": []map[string]any{
				{"type": "text", "text": strings.Repeat("x", 700)},
			},
			"stop_reason": nil,
			"usage": map[string]any{
				"input_tokens":  1,
				"output_tokens": 1,
			},
		},
		"session_id": "sess-fill",
		"uuid":       "u1",
	}
	fillerLine, err := json.Marshal(filler)
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		collector.appendLine(fillerLine)
	}
	require.True(t, collector.truncated, "collector should truncate once the raw buffer cap is hit")

	resultLine := []byte(`{"type":"result","subtype":"success","is_error":false,"duration_ms":1,"duration_api_ms":1,"num_turns":4,"result":"done","stop_reason":"end_turn","session_id":"sess-123","total_cost_usd":1.25,"usage":{"input_tokens":0,"output_tokens":0},"uuid":"u2"}`)
	collector.appendLine(resultLine)

	_, parseErr := parseCLIOutput(collector.bytes())
	require.Error(t, parseErr, "truncated raw output should no longer be parseable")

	result := collector.fallbackResult()
	require.NotNil(t, result)
	assert.Equal(t, "done", result.Result)
	assert.Equal(t, "sess-123", result.SessionID)
	assert.Equal(t, 4, result.NumTurns)
	assert.Equal(t, 1.25, result.TotalCostUSD)
}
