package routing

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
)

// Violation represents a routing rule violation.
// Logged to JSONL file at config.GetViolationsLogPath() for audit trail and debugging.
type Violation struct {
	Timestamp     string `json:"timestamp"`
	SessionID     string `json:"session_id"`
	ViolationType string `json:"violation_type"`
	Agent         string `json:"agent,omitempty"`
	Model         string `json:"model,omitempty"`
	Tool          string `json:"tool,omitempty"`
	Reason        string `json:"reason"`
	Allowed       string `json:"allowed,omitempty"`
	Override      string `json:"override,omitempty"`
}

// LogViolation appends violation to JSONL log file.
// Creates log file if it doesn't exist.
// Timestamp is auto-populated in RFC3339 format.
func LogViolation(v *Violation) error {
	// Auto-populate timestamp
	v.Timestamp = time.Now().Format(time.RFC3339)

	// Open log file (append mode, create if not exists)
	logPath := config.GetViolationsLogPath()
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("[violations] Failed to open log: %w", err)
	}
	defer f.Close()

	// Marshal violation to JSON
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("[violations] Failed to marshal violation: %w", err)
	}

	// Write JSONL entry (append newline)
	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("[violations] Failed to write log: %w", err)
	}

	return nil
}
