package mcp

import (
	"os"
	"path/filepath"
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
