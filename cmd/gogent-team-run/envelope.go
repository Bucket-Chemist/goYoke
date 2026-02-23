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
	Agent       string             `json:"agent"`
	Context     map[string]any     `json:"context"`
	Task        json.RawMessage    `json:"task,omitempty"`        // string (review) or object (implementation)
	Description string             `json:"description,omitempty"`

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

// stdoutSchemasBaseDir can be overridden in tests to use a temp directory.
// When empty, defaults to $HOME/.claude/schemas/teams/stdin-stdout/
var stdoutSchemasBaseDir string

// resolveStdoutSchema finds the stdout contract schema for the given workflow and agent.
// Returns the pretty-printed stdout JSON schema and the schema filename (without .json),
// or empty strings if no matching schema is found.
// This is a best-effort lookup — missing schemas are not errors.
func resolveStdoutSchema(workflowType, agentID string) (stdoutJSON string, schemaName string) {
	dir := stdoutSchemasBaseDir
	if dir == "" {
		dir = filepath.Join(os.Getenv("HOME"), ".claude", "schemas", "teams", "stdin-stdout")
	}

	candidates := buildSchemaCandidates(workflowType, agentID)

	for _, candidate := range candidates {
		path := filepath.Join(dir, candidate)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		// Parse as map to extract "stdout" key
		var contract map[string]json.RawMessage
		if err := json.Unmarshal(data, &contract); err != nil {
			continue
		}

		stdoutRaw, ok := contract["stdout"]
		if !ok {
			continue
		}

		// Pretty-print the stdout schema
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, stdoutRaw, "", "  "); err != nil {
			continue
		}

		name := strings.TrimSuffix(candidate, ".json")
		return pretty.String(), name
	}

	return "", ""
}

// formalSchemasBaseDir can be overridden in tests to use a temp directory.
// When empty, defaults to $HOME/.claude/schemas/stdout/
var formalSchemasBaseDir string

// resolveFormalSchema loads a formal JSON Schema from schemas/stdout/,
// strips meta-fields and descriptions, and returns minified JSON suitable
// for the --json-schema CLI flag.
//
// Resolution candidates (reuses suffix-stripping pattern):
//   - exact: einstein → schemas/stdout/einstein.json
//   - suffix-stripped: staff-architect-critical-review → staff-architect.json
//   - role fallback: backend-reviewer → reviewer.json
//   - generic: go-pro → worker.json
func resolveFormalSchema(agentID string) (string, bool) {
	dir := formalSchemasBaseDir
	if dir == "" {
		dir = filepath.Join(os.Getenv("HOME"), ".claude", "schemas", "stdout")
	}

	candidates := buildFormalSchemaCandidates(agentID)

	for _, candidate := range candidates {
		path := filepath.Join(dir, candidate)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		// Parse schema
		var schema map[string]json.RawMessage
		if err := json.Unmarshal(data, &schema); err != nil {
			continue
		}

		// Strip top-level meta-fields (not valid in --json-schema input)
		for _, key := range []string{"$schema", "$id", "title", "description", "version"} {
			delete(schema, key)
		}

		// Strip description fields from nested properties
		if props, ok := schema["properties"]; ok {
			schema["properties"] = stripNestedDescriptions(props)
		}

		// Minify
		cleaned, err := json.Marshal(schema)
		if err != nil {
			continue
		}
		var compact bytes.Buffer
		if err := json.Compact(&compact, cleaned); err != nil {
			continue
		}

		return compact.String(), true
	}

	return "", false
}

// buildFormalSchemaCandidates generates candidate filenames for formal schema lookup.
// Unlike buildSchemaCandidates, these don't have a workflow-type prefix.
func buildFormalSchemaCandidates(agentID string) []string {
	var candidates []string

	// 1. Exact: einstein.json, go-pro.json
	candidates = append(candidates, agentID+".json")

	// 2. Strip common agent suffixes, with role fallback
	suffixes := []string{"-reviewer", "-critical-review", "-pro", "-writer", "-archivist"}
	for _, suffix := range suffixes {
		if stripped, ok := strings.CutSuffix(agentID, suffix); ok {
			// Try the stripped name (e.g., "backend" from "backend-reviewer")
			candidates = append(candidates, stripped+".json")
			// Role fallback: try the suffix as a schema name
			// (e.g., "reviewer.json" from "-reviewer" suffix)
			role := strings.TrimPrefix(suffix, "-")
			candidates = append(candidates, role+".json")
		}
	}

	// 3. Generic worker fallback
	candidates = append(candidates, "worker.json")

	return candidates
}

// stripNestedDescriptions removes "description" keys from property definitions
// within a JSON Schema "properties" object. This reduces token overhead in
// constrained decoding where descriptions serve no purpose.
func stripNestedDescriptions(propertiesRaw json.RawMessage) json.RawMessage {
	var properties map[string]json.RawMessage
	if err := json.Unmarshal(propertiesRaw, &properties); err != nil {
		return propertiesRaw
	}

	for key, propRaw := range properties {
		var prop map[string]json.RawMessage
		if err := json.Unmarshal(propRaw, &prop); err != nil {
			continue
		}

		// Remove description from this property
		delete(prop, "description")

		// Recurse into nested properties (objects with their own "properties" key)
		if nestedProps, ok := prop["properties"]; ok {
			prop["properties"] = stripNestedDescriptions(nestedProps)
		}

		// Recurse into items (arrays with item schemas)
		if items, ok := prop["items"]; ok {
			var itemSchema map[string]json.RawMessage
			if err := json.Unmarshal(items, &itemSchema); err == nil {
				delete(itemSchema, "description")
				if nestedProps, ok := itemSchema["properties"]; ok {
					itemSchema["properties"] = stripNestedDescriptions(nestedProps)
				}
				if updated, err := json.Marshal(itemSchema); err == nil {
					prop["items"] = updated
				}
			}
		}

		if updated, err := json.Marshal(prop); err == nil {
			properties[key] = updated
		}
	}

	result, err := json.Marshal(properties)
	if err != nil {
		return propertiesRaw
	}
	return result
}

// buildSchemaCandidates generates candidate filenames for contract schema lookup.
// Tries in order: exact match, suffix-stripped variants, generic worker fallback.
func buildSchemaCandidates(workflowType, agentID string) []string {
	var candidates []string

	// 1. Exact: e.g., "review-backend-reviewer.json" or "braintrust-einstein.json"
	candidates = append(candidates, fmt.Sprintf("%s-%s.json", workflowType, agentID))

	// 2. Strip common agent suffixes
	suffixes := []string{"-reviewer", "-critical-review", "-pro", "-writer", "-archivist"}
	for _, suffix := range suffixes {
		if stripped, ok := strings.CutSuffix(agentID, suffix); ok {
			candidates = append(candidates, fmt.Sprintf("%s-%s.json", workflowType, stripped))
		}
	}

	// 3. Generic worker fallback: e.g., "implementation-worker.json"
	candidates = append(candidates, fmt.Sprintf("%s-worker.json", workflowType))

	return candidates
}

// buildPromptEnvelope reads stdin JSON from member.StdinFile and builds a formatted
// prompt string for the Claude CLI. Validates required fields and prevents path traversal.
//
// The envelope includes:
// - AGENT: header with agent name
// - Task/context description
// - Capabilities notice (Task delegation policy at nesting level 2)
// - All workflow-specific fields from stdin
// - Stdout contract schema with output format instructions (when available)
//
// Returns error if:
// - stdin path escapes teamDir (path traversal)
// - stdin file doesn't exist or is unreadable
// - stdin is not valid JSON
// - required fields (task/description and context) are empty
func buildPromptEnvelope(teamDir string, member *Member, workflowType string, constrainedDecoding bool) (string, error) {
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
	// Task can be a string (review workflow) or object (implementation workflow)
	taskPresent := len(stdin.Task) > 0 && string(stdin.Task) != "null" && string(stdin.Task) != `""`
	if !taskPresent && stdin.Description == "" {
		return "", fmt.Errorf("stdin: task field is empty (checked both 'task' and 'description')")
	}

	contextField := ""
	if len(stdin.Context) > 0 {
		contextField = "present"
	}
	if contextField == "" {
		return "", fmt.Errorf("stdin: context field is empty or missing")
	}

	// Build envelope with capabilities notice
	var builder strings.Builder

	fmt.Fprintf(&builder, "AGENT: %s\n\n", member.Agent)

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

	// Resolve and embed stdout contract schema for output format compliance
	if workflowType != "" {
		stdoutSchema, schemaName := resolveStdoutSchema(workflowType, member.Agent)
		if stdoutSchema != "" {
			builder.WriteString("## Expected Output Format\n\n")
			if workflowType == "implementation" {
				builder.WriteString("Complete your implementation task using the tools available to you (Read, Write, Edit, Bash, Glob, Grep). Create files, run builds, run tests — do the actual work.\n\n")
				builder.WriteString("After ALL implementation work is done, your FINAL response must be a single JSON code block summarizing what you accomplished.\n\n")
			} else {
				builder.WriteString("Your response MUST be a single JSON code block. Do NOT include any text outside the code block — no markdown, no explanations, no preamble or postamble.\n\n")
			}
			builder.WriteString("The JSON must conform to this schema:\n\n")
			builder.WriteString("```json\n")
			builder.WriteString(stdoutSchema)
			builder.WriteString("\n```\n\n")
			if constrainedDecoding {
				fmt.Fprintf(&builder, "Include \"schema_id\": \"%s\" in your JSON output.\n", schemaName)
			} else {
				fmt.Fprintf(&builder, "Include \"$schema\": \"%s\" in your JSON output.\n", schemaName)
			}
			builder.WriteString("Use exact field names, types, and enum values as shown in the schema.\n\n")
		}
	}

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
