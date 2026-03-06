package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const defaultTimeout = 3 * time.Second

// instructionsLoadedEvent represents the Claude Code InstructionsLoaded hook event.
type instructionsLoadedEvent struct {
	SessionID  string `json:"session_id"`
	FilePath   string `json:"file_path"`
	MemoryType string `json:"memory_type"`
	LoadReason string `json:"load_reason"`
	AgentID    string `json:"agent_id"`
	AgentType  string `json:"agent_type"`
	CWD        string `json:"cwd"`
}

// auditRecord is the JSONL record written for each InstructionsLoaded event.
type auditRecord struct {
	Timestamp  int64  `json:"timestamp"`
	SessionID  string `json:"session_id"`
	FilePath   string `json:"file_path"`
	MemoryType string `json:"memory_type"`
	LoadReason string `json:"load_reason"`
	AgentID    string `json:"agent_id"`
	AgentType  string `json:"agent_type"`
}

// readEvent reads and parses the InstructionsLoaded event from r with a timeout.
func readEvent(r io.Reader, timeout time.Duration) (*instructionsLoadedEvent, error) {
	type result struct {
		event *instructionsLoadedEvent
		err   error
	}
	ch := make(chan result, 1)
	go func() {
		data, err := io.ReadAll(r)
		if err != nil {
			ch <- result{nil, fmt.Errorf("read stdin: %w", err)}
			return
		}
		if len(data) == 0 {
			ch <- result{nil, fmt.Errorf("empty stdin")}
			return
		}
		var event instructionsLoadedEvent
		if err := json.Unmarshal(data, &event); err != nil {
			ch <- result{nil, fmt.Errorf("parse JSON: %w", err)}
			return
		}
		ch <- result{&event, nil}
	}()

	select {
	case res := <-ch:
		return res.event, res.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("stdin read timeout after %v", timeout)
	}
}

// auditLogPath resolves the path to the instructions-audit.jsonl file.
// Uses $XDG_DATA_HOME/gogent/ or falls back to ~/.local/share/gogent/.
func auditLogPath() string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = os.Getenv("HOME")
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "gogent", "instructions-audit.jsonl")
}

// appendAuditRecord writes an audit record as a JSONL line to path.
func appendAuditRecord(path string, record auditRecord) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal record: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write record: %w", err)
	}

	return nil
}

func main() {
	event, err := readEvent(os.Stdin, defaultTimeout)
	if err != nil {
		// Parse failure is non-fatal: this hook must never block.
		fmt.Fprintf(os.Stderr, "[gogent-instructions-audit] Warning: %v\n", err)
		os.Exit(0)
	}

	record := auditRecord{
		Timestamp:  time.Now().Unix(),
		SessionID:  event.SessionID,
		FilePath:   event.FilePath,
		MemoryType: event.MemoryType,
		LoadReason: event.LoadReason,
		AgentID:    event.AgentID,
		AgentType:  event.AgentType,
	}

	if err := appendAuditRecord(auditLogPath(), record); err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-instructions-audit] Warning: append failed: %v\n", err)
	}

	os.Exit(0)
}
