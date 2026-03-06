package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstructionsAudit_AppendsToLog(t *testing.T) {
	// Use a temp directory for the audit log.
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "instructions-audit.jsonl")

	input := `{
		"session_id": "test-session-001",
		"hook_event_name": "InstructionsLoaded",
		"file_path": "/home/user/.claude/CLAUDE.md",
		"memory_type": "Project",
		"load_reason": "session_start",
		"agent_id": "go-pro",
		"agent_type": "GO Pro",
		"cwd": "/home/user/project"
	}`

	event, err := readEvent(strings.NewReader(input), 3*time.Second)
	require.NoError(t, err)

	record := auditRecord{
		Timestamp:  time.Now().Unix(),
		SessionID:  event.SessionID,
		FilePath:   event.FilePath,
		MemoryType: event.MemoryType,
		LoadReason: event.LoadReason,
		AgentID:    event.AgentID,
		AgentType:  event.AgentType,
	}

	err = appendAuditRecord(logPath, record)
	require.NoError(t, err)

	// Verify the file was created and contains valid JSONL.
	f, err := os.Open(logPath)
	require.NoError(t, err)
	defer f.Close()

	scanner := bufio.NewScanner(f)
	require.True(t, scanner.Scan(), "expected at least one line in audit log")

	var got auditRecord
	err = json.Unmarshal(scanner.Bytes(), &got)
	require.NoError(t, err)

	assert.Equal(t, "test-session-001", got.SessionID)
	assert.Equal(t, "/home/user/.claude/CLAUDE.md", got.FilePath)
	assert.Equal(t, "Project", got.MemoryType)
	assert.Equal(t, "session_start", got.LoadReason)
	assert.Equal(t, "go-pro", got.AgentID)
	assert.Equal(t, "GO Pro", got.AgentType)
	assert.Greater(t, got.Timestamp, int64(0))
}

func TestInstructionsAudit_AppendsMultipleRecords(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "instructions-audit.jsonl")

	for i := 0; i < 3; i++ {
		record := auditRecord{
			Timestamp: time.Now().Unix(),
			SessionID: "session-multi",
			FilePath:  "/some/CLAUDE.md",
		}
		require.NoError(t, appendAuditRecord(logPath, record))
	}

	f, err := os.Open(logPath)
	require.NoError(t, err)
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineCount := 0
	for scanner.Scan() {
		lineCount++
	}
	assert.Equal(t, 3, lineCount, "expected 3 JSONL lines")
}

func TestInstructionsAudit_HandlesEmptyInput(t *testing.T) {
	// Empty stdin should return an error but not crash.
	_, err := readEvent(strings.NewReader(""), 3*time.Second)
	assert.Error(t, err, "expected error on empty input")
}

func TestInstructionsAudit_HandlesInvalidJSON(t *testing.T) {
	_, err := readEvent(strings.NewReader("not json"), 3*time.Second)
	assert.Error(t, err)
}

func TestInstructionsAudit_ReadEvent_ValidJSON(t *testing.T) {
	input := `{
		"session_id": "abc123",
		"file_path": "/home/user/.claude/CLAUDE.md",
		"memory_type": "Global",
		"load_reason": "session_start",
		"agent_id": "python-pro",
		"agent_type": "Python Pro"
	}`

	event, err := readEvent(strings.NewReader(input), 3*time.Second)

	require.NoError(t, err)
	assert.Equal(t, "abc123", event.SessionID)
	assert.Equal(t, "Global", event.MemoryType)
	assert.Equal(t, "python-pro", event.AgentID)
}

func TestInstructionsAudit_CreatesDirectoryIfMissing(t *testing.T) {
	tmpDir := t.TempDir()
	// Use a nested path that doesn't exist yet.
	logPath := filepath.Join(tmpDir, "nested", "deep", "instructions-audit.jsonl")

	record := auditRecord{
		Timestamp: time.Now().Unix(),
		SessionID: "dir-test",
	}

	err := appendAuditRecord(logPath, record)
	require.NoError(t, err)

	_, err = os.Stat(logPath)
	assert.NoError(t, err, "log file should exist after append")
}
