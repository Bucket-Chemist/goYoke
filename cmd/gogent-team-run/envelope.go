package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StdinEnvelope represents the minimal validation structure for stdin JSON files.
// We only validate required fields (agent, context, task/description) but preserve
// the full JSON for the agent to consume.
type StdinEnvelope struct {
	// Fields used for validation only
	Agent       string                 `json:"agent"`
	Context     map[string]interface{} `json:"context"`
	Task        string                 `json:"task,omitempty"`
	Description string                 `json:"description,omitempty"`

	// Raw JSON for full preservation
	raw json.RawMessage
}

// UnmarshalJSON custom unmarshaler to preserve full JSON while extracting validation fields
func (s *StdinEnvelope) UnmarshalJSON(data []byte) error {
	// Store raw JSON for later serialization
	s.raw = make(json.RawMessage, len(data))
	copy(s.raw, data)

	// Extract validation fields
	type Alias StdinEnvelope
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(s),
	}

	return json.Unmarshal(data, aux)
}

// buildPromptEnvelope reads stdin JSON from member.StdinFile and builds a formatted
// prompt string for the Claude CLI. Validates required fields and prevents path traversal.
//
// The envelope includes:
// - AGENT: header with agent name
// - Task/context description
// - Capabilities notice (Task delegation policy at nesting level 2)
// - All workflow-specific fields from stdin
//
// Returns error if:
// - stdin path escapes teamDir (path traversal)
// - stdin file doesn't exist or is unreadable
// - stdin is not valid JSON
// - required fields (task/description and context) are empty
func buildPromptEnvelope(teamDir string, member *Member) (string, error) {
	// W2: Validate stdin path is within teamDir (path traversal protection)
	stdinPath := filepath.Join(teamDir, member.StdinFile)
	if err := validatePathWithinDir(stdinPath, teamDir); err != nil {
		return "", fmt.Errorf("stdin path security: %w", err)
	}

	// Read and parse stdin
	stdinData, err := os.ReadFile(stdinPath)
	if err != nil {
		return "", fmt.Errorf("read stdin file %s: %w", member.StdinFile, err)
	}

	var stdin StdinEnvelope
	if err := json.Unmarshal(stdinData, &stdin); err != nil {
		return "", fmt.Errorf("parse stdin JSON: %w", err)
	}

	// W3: Validate required fields
	taskField := stdin.Task
	if taskField == "" {
		taskField = stdin.Description
	}
	if taskField == "" {
		return "", fmt.Errorf("stdin: task field is empty (checked both 'task' and 'description')")
	}

	contextField := ""
	if stdin.Context != nil && len(stdin.Context) > 0 {
		contextField = "present"
	}
	if contextField == "" {
		return "", fmt.Errorf("stdin: context field is empty or missing")
	}

	// Build envelope with capabilities notice
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("AGENT: %s\n\n", member.Agent))

	// Serialize full stdin for agent consumption (use raw JSON to preserve all fields)
	builder.WriteString("# Stdin Envelope\n\n")

	// Pretty-print the raw JSON
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, stdin.raw, "", "  "); err != nil {
		return "", fmt.Errorf("format stdin JSON: %w", err)
	}

	builder.WriteString("```json\n")
	builder.WriteString(prettyJSON.String())
	builder.WriteString("\n```\n\n")

	// Add capabilities notice per TC-007 nesting level policy
	builder.WriteString(`## Your Capabilities

You are spawned via gogent-team-run at nesting level 2.

**Available delegation:**
- Task(model: "haiku") — For mechanical tasks (file search, pattern extraction)
- Task(model: "sonnet") — For focused analysis or implementation
- Task(model: "opus") — BLOCKED by gogent-validate

Always specify model explicitly in Task() calls. If omitted, the CLI defaults to the
session model, which may be Opus — causing an unintended block.

**MCP Tools:**
You do NOT have access to spawn_agent (that's for MCP-spawned agents only).

`)

	return builder.String(), nil
}

// validatePathWithinDir ensures targetPath does not escape baseDir.
// Used for both stdin and stdout path validation (W2 path traversal protection).
//
// Returns error if:
// - Path resolution fails
// - targetPath is outside baseDir (including symlink attacks)
//
// Accepts:
// - Relative paths that resolve within baseDir
// - Absolute paths within baseDir
// - targetPath == baseDir (exact match)
func validatePathWithinDir(targetPath, baseDir string) error {
	// Resolve to absolute paths (follows symlinks)
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("resolve target path: %w", err)
	}

	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return fmt.Errorf("resolve base dir: %w", err)
	}

	// Clean paths to normalize (remove .., ., etc.)
	absTarget = filepath.Clean(absTarget)
	absBase = filepath.Clean(absBase)

	// Check if target is within base or equal to base
	if absTarget == absBase {
		return nil
	}

	// Ensure target starts with base + separator (prevents /base and /base-evil confusion)
	if !strings.HasPrefix(absTarget, absBase+string(filepath.Separator)) {
		return fmt.Errorf("path %s escapes base directory %s", targetPath, baseDir)
	}

	return nil
}
