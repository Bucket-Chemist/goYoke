package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// StdoutEnvelope represents the agent stdout JSON structure.
// Per TC-009 stdin-stdout schema requirements.
type StdoutEnvelope struct {
	Schema  string                 `json:"$schema"`
	Status  string                 `json:"status"`
	Content map[string]interface{} `json:"content"`
}

// validateStdout reads and validates the agent's stdout JSON file.
// Checks:
// - Path safety: stdout path must be within teamDir (W2 path traversal protection)
// - File exists and is readable
// - Valid JSON structure
// - Required fields: $schema and status
//
// Returns error if any validation fails.
func validateStdout(stdoutPath string, teamDir string) error {
	// W2: Path traversal protection
	if err := validatePathWithinDir(stdoutPath, teamDir); err != nil {
		return fmt.Errorf("stdout path security: %w", err)
	}

	data, err := os.ReadFile(stdoutPath)
	if err != nil {
		return fmt.Errorf("read stdout file: %w", err)
	}

	if len(data) == 0 {
		return fmt.Errorf("stdout file is empty")
	}

	var stdout StdoutEnvelope
	if err := json.Unmarshal(data, &stdout); err != nil {
		return fmt.Errorf("parse stdout JSON: %w", err)
	}

	if stdout.Schema == "" {
		return fmt.Errorf("missing $schema field in stdout")
	}
	if stdout.Status == "" {
		return fmt.Errorf("missing status field in stdout")
	}

	return nil
}

// validateOutputPath is an alias for validatePathWithinDir (defined in envelope.go).
// Kept for backward compatibility with validate_test.go tests.
var validateOutputPath = validatePathWithinDir
