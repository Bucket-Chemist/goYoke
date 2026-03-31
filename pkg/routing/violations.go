package routing

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
)

// Violation represents a routing rule violation.
// Logged to both XDG cache (global) and .gogent/memory/ (project-scoped).
type Violation struct {
	// Existing fields from GOgent-011
	Timestamp     string `json:"timestamp"`
	SessionID     string `json:"session_id"`
	ViolationType string `json:"violation_type"`
	Agent         string `json:"agent,omitempty"`
	Model         string `json:"model,omitempty"`
	Tool          string `json:"tool,omitempty"`
	Reason        string `json:"reason"`
	Allowed       string `json:"allowed,omitempty"`
	Override      string `json:"override,omitempty"`

	// NEW: File context (critical for correlation with sharp edges)
	File string `json:"file,omitempty"`

	// NEW: Tier context (critical for pattern analysis)
	CurrentTier  string `json:"current_tier,omitempty"`
	RequiredTier string `json:"required_tier,omitempty"`

	// NEW: Task context (critical for understanding user intent)
	TaskDescription string `json:"task_description,omitempty"` // First 200 chars of prompt

	// NEW: Enforcement outcome (critical for effectiveness analysis)
	HookDecision string `json:"hook_decision,omitempty"` // "allow", "warn", "block"

	// NEW: Project context (enables cross-project pattern detection)
	ProjectDir string `json:"project_dir,omitempty"`
}

// LogViolation appends violation to BOTH:
// 1. Global XDG cache: ~/.cache/gogent/routing-violations.jsonl (survives project deletion)
// 2. Project memory: <project>/.gogent/memory/routing-violations.jsonl (session integration)
//
// Timestamp is auto-populated in RFC3339 format.
// Project log failure does NOT fail the entire operation (graceful degradation).
func LogViolation(v *Violation, projectDir string) error {
	// Auto-populate timestamp
	v.Timestamp = time.Now().Format(time.RFC3339)

	// Populate project directory if provided
	if projectDir != "" {
		v.ProjectDir = projectDir
	}

	// Marshal once, write twice
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("[violations] Failed to marshal violation: %w", err)
	}
	data = append(data, '\n') // JSONL format

	// WRITE 1: Global XDG cache (primary, required)
	globalPath := config.GetViolationsLogPath()
	if err := appendToFile(globalPath, data); err != nil {
		return fmt.Errorf("[violations] Failed to write global log: %w", err)
	}

	// WRITE 2: Project memory (secondary, optional)
	if projectDir != "" {
		projectPath := config.GetProjectViolationsLogPath(projectDir)
		if err := appendToFile(projectPath, data); err != nil {
			// Log warning but don't fail - global write succeeded
			fmt.Fprintf(os.Stderr, "[violations] Warning: Failed project log: %v\n", err)
		}
	}

	return nil
}

// appendToFile appends data to file (creates if not exists).
// Helper for dual-write pattern.
func appendToFile(path string, data []byte) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	// Open/create file in append mode
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}
