package teamrun

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// StdoutEnvelope represents the agent stdout JSON structure.
// Per TC-009 stdin-stdout schema requirements.
type StdoutEnvelope struct {
	Schema   string                 `json:"$schema"`
	SchemaID string                 `json:"schema_id"`
	Status   string                 `json:"status"`
	Content  map[string]interface{} `json:"content"`
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

	if stdout.Schema == "" && stdout.SchemaID == "" {
		return fmt.Errorf("missing $schema or schema_id field in stdout")
	}
	if stdout.Status == "" {
		return fmt.Errorf("missing status field in stdout")
	}

	return nil
}

// validateOutputPath is an alias for validatePathWithinDir (defined in envelope.go).
// Kept for backward compatibility with validate_test.go tests.
var validateOutputPath = validatePathWithinDir

// ValidateConfig performs pre-flight validation of team configuration.
// Catches missing scripts, files, and paths before any waves execute,
// preventing expensive agent work from being wasted on invalid configs.
//
// Validates:
//   - on_complete_script: resolvable (LookPath for bare names, os.Stat for paths)
//   - stdin_file: exists in teamDir for each member
//   - project_root: directory exists
//
// Returns nil if config is valid, or descriptive error on first failure.
func (tr *TeamRunner) ValidateConfig() error {
	if tr.config == nil {
		return nil
	}

	// Validate project_root exists
	if tr.config.ProjectRoot != "" {
		if info, err := os.Stat(tr.config.ProjectRoot); err != nil {
			return fmt.Errorf("project_root %q: %w", tr.config.ProjectRoot, err)
		} else if !info.IsDir() {
			return fmt.Errorf("project_root %q: not a directory", tr.config.ProjectRoot)
		}
	}

	for _, wave := range tr.config.Waves {
		// Validate on_complete_script
		if wave.OnCompleteScript != nil && *wave.OnCompleteScript != "" {
			script := *wave.OnCompleteScript
			if strings.ContainsRune(script, filepath.Separator) {
				// Absolute or relative path — verify file exists
				if _, err := os.Stat(script); err != nil {
					return fmt.Errorf("wave %d on_complete_script %q: %w",
						wave.WaveNumber, script, err)
				}
			} else {
				// Bare name — verify it's on PATH
				if _, err := exec.LookPath(script); err != nil {
					return fmt.Errorf("wave %d on_complete_script %q not found on PATH: %w",
						wave.WaveNumber, script, err)
				}
			}
		}

		// Validate stdin_file for each member
		for _, member := range wave.Members {
			if member.StdinFile != "" {
				stdinPath := filepath.Join(tr.teamDir, member.StdinFile)
				if _, err := os.Stat(stdinPath); err != nil {
					return fmt.Errorf("wave %d member %s: stdin_file %q: %w",
						wave.WaveNumber, member.Name, member.StdinFile, err)
				}
			}
		}
	}

	return nil
}
