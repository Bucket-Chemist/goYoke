package harness

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// ReplayEvent represents a complete tool use cycle for session replay testing.
// This is DISTINCT from SessionEvent (SessionEnd hook input) and ToolEvent (PreToolUse input).
//
// Design rationale:
// - SessionEvent: Represents the input for SessionEnd hook (session termination)
// - ToolEvent: Represents PreToolUse hook input (tool invocation validation)
// - ReplayEvent: Represents a complete tool cycle INCLUDING response, for multi-turn testing
//
// ReplayEvent combines both hook phases (Pre and Post) in a single structure to enable
// testing sequences like: "fail 3 times on same file -> expect blocking".
type ReplayEvent struct {
	// Timestamp for ordering events chronologically
	Timestamp int64 `json:"ts"`

	// HookType determines which CLI processes this event.
	// Valid values: "PreToolUse" (gogent-validate), "PostToolUse" (gogent-sharp-edge)
	HookType string `json:"hook_type"`

	// Tool identification (mirrors ToolEvent for consistency)
	ToolName  string                 `json:"tool_name"`
	ToolInput map[string]interface{} `json:"tool_input"`

	// Response data (PostToolUse only - represents tool execution result)
	ToolResponse map[string]interface{} `json:"tool_response,omitempty"`
	Success      bool                   `json:"success"`

	// ExpectedDecision validates per-event hook output.
	// Empty string means no per-event validation (only session-level expectations apply).
	// Values: "block", "allow", "" (any)
	ExpectedDecision string `json:"expected_decision,omitempty"`
}

// ReplaySession represents a complete recorded session for replay testing.
// Each session is an ordered sequence of events that should execute together
// with state persisting between events (but resetting between sessions).
type ReplaySession struct {
	ID          string              `json:"session_id"`
	Description string              `json:"description"`
	Events      []ReplayEvent       `json:"events"`
	Expected    ReplayExpectations  `json:"expected"`
}

// ReplayExpectations defines what should result from replaying the session.
// These are checked AFTER all events in the session have been executed.
type ReplayExpectations struct {
	// Artifact counts validate that the correct number of items were created
	SharpEdgesCreated int  `json:"sharp_edges_created"`
	BlockingResponses int  `json:"blocking_responses"`
	HandoffCreated    bool `json:"handoff_created"`

	// FileContains maps relative paths to substrings that must appear in the file
	FileContains map[string][]string `json:"file_contains,omitempty"`

	// FilesCreated lists relative paths that must exist after replay
	FilesCreated []string `json:"files_created,omitempty"`

	// FilesNotCreated lists relative paths that must NOT exist after replay
	FilesNotCreated []string `json:"files_not_created,omitempty"`
}

// ReplayResult captures the outcome of replaying a single session.
type ReplayResult struct {
	SessionID         string        `json:"session_id"`
	Passed            bool          `json:"passed"`
	Duration          time.Duration `json:"duration"`
	SharpEdgesCreated int           `json:"sharp_edges_created"`
	BlockingResponses int           `json:"blocking_responses"`
	EventErrors       []string      `json:"event_errors,omitempty"`
	ValidationErrors  []string      `json:"validation_errors,omitempty"`
	Error             error         `json:"-"`
	ErrorMsg          string        `json:"error,omitempty"`
}

// SessionReplayer executes recorded sessions against the hook system.
// Key design decision: State persists WITHIN a session (events accumulate),
// but resets BETWEEN sessions (each session gets a fresh temp directory).
type SessionReplayer struct {
	validatePath  string
	archivePath   string
	sharpEdgePath string
	fixturesDir   string
	schemaPath    string
	agentsPath    string
	verbose       bool
}

// NewSessionReplayer creates a replayer with paths to CLI binaries.
// The sharpEdgePath is required for PostToolUse event execution.
func NewSessionReplayer(validatePath, archivePath, sharpEdgePath, fixturesDir string) *SessionReplayer {
	return &SessionReplayer{
		validatePath:  validatePath,
		archivePath:   archivePath,
		sharpEdgePath: sharpEdgePath,
		fixturesDir:   fixturesDir,
	}
}

// SetSchemaPath sets the routing schema path for test isolation.
func (r *SessionReplayer) SetSchemaPath(path string) {
	r.schemaPath = path
}

// SetAgentsPath sets the agents index path for test isolation.
func (r *SessionReplayer) SetAgentsPath(path string) {
	r.agentsPath = path
}

// SetVerbose enables verbose output during replay.
func (r *SessionReplayer) SetVerbose(v bool) {
	r.verbose = v
}

// ReplayAll loads and replays all sessions from the fixtures directory.
// Returns results for each session in order.
func (r *SessionReplayer) ReplayAll() ([]ReplayResult, error) {
	sessions, err := r.loadSessions()
	if err != nil {
		return nil, fmt.Errorf("load sessions: %w", err)
	}

	if len(sessions) == 0 {
		return nil, fmt.Errorf("no sessions found in %s", filepath.Join(r.fixturesDir, "sessions"))
	}

	var results []ReplayResult
	for _, session := range sessions {
		result := r.ReplaySession(session)
		results = append(results, result)

		if r.verbose {
			status := "PASS"
			if !result.Passed {
				status = "FAIL"
			}
			fmt.Printf("[%s] Session %s (%v)\n", status, session.ID, result.Duration)
		}
	}

	return results, nil
}

// ReplaySession executes a single session and validates expectations.
// State management: Creates a fresh temp directory for the session,
// executes all events in order (state accumulates), then validates.
func (r *SessionReplayer) ReplaySession(session ReplaySession) ReplayResult {
	start := time.Now()
	result := ReplayResult{
		SessionID: session.ID,
	}

	// Create fresh temp directory for THIS session.
	// State accumulates within the session, but each session starts clean.
	// This isolation prevents cross-session contamination in test results.
	sessionTempDir, err := os.MkdirTemp("", "replay-"+session.ID+"-")
	if err != nil {
		result.Error = fmt.Errorf("create temp dir: %w", err)
		result.ErrorMsg = result.Error.Error()
		result.Duration = time.Since(start)
		return result
	}
	defer os.RemoveAll(sessionTempDir) // Cleanup after session completes

	// Initialize required directories for CLIs to write to
	requiredDirs := []string{
		filepath.Join(sessionTempDir, ".claude", "memory"),
		filepath.Join(sessionTempDir, ".gogent"),
	}
	for _, dir := range requiredDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			result.Error = fmt.Errorf("create dir %s: %w", dir, err)
			result.ErrorMsg = result.Error.Error()
			result.Duration = time.Since(start)
			return result
		}
	}

	// Track metrics across events (state accumulates)
	var blockingCount int
	var sharpEdgesCreated int

	// Execute events IN ORDER - state accumulates within session
	for i, event := range session.Events {
		output, err := r.executeEvent(event, sessionTempDir, session.ID)
		if err != nil {
			result.Error = fmt.Errorf("event %d (%s): %w", i, event.HookType, err)
			result.ErrorMsg = result.Error.Error()
			result.Duration = time.Since(start)
			return result
		}

		// Parse output for metrics
		var hookResp map[string]interface{}
		if json.Unmarshal([]byte(output), &hookResp) == nil {
			if decision, _ := hookResp["decision"].(string); decision == "block" {
				blockingCount++
			}
		}

		// Check for sharp edge creation after PostToolUse events
		if event.HookType == "PostToolUse" {
			pendingPath := filepath.Join(sessionTempDir, ".claude", "memory", "pending-learnings.jsonl")
			if content, err := os.ReadFile(pendingPath); err == nil {
				sharpEdgesCreated = countJSONLLines(content)
			}
		}

		// Validate per-event expectation if specified
		if event.ExpectedDecision != "" {
			actualDecision, _ := hookResp["decision"].(string)
			// Handle empty response (pass-through) - treat as "allow"
			if output == "" || output == "{}" {
				actualDecision = "allow"
			}
			if actualDecision != event.ExpectedDecision {
				result.EventErrors = append(result.EventErrors,
					fmt.Sprintf("event %d: expected decision %q, got %q (output: %s)",
						i, event.ExpectedDecision, actualDecision, truncateOutput(output, 200)))
			}
		}
	}

	// Record final counts
	result.SharpEdgesCreated = sharpEdgesCreated
	result.BlockingResponses = blockingCount

	// Validate session-level expectations
	result.ValidationErrors = r.validateExpectations(session.Expected, sessionTempDir, sharpEdgesCreated, blockingCount)
	result.Passed = len(result.EventErrors) == 0 && len(result.ValidationErrors) == 0
	result.Duration = time.Since(start)

	return result
}

// executeEvent runs the appropriate CLI for a single event within the session.
func (r *SessionReplayer) executeEvent(event ReplayEvent, tempDir, sessionID string) (string, error) {
	var cmdPath string
	var input interface{}

	switch event.HookType {
	case "PreToolUse":
		if r.validatePath == "" {
			return "", fmt.Errorf("validatePath not set for PreToolUse event")
		}
		cmdPath = r.validatePath
		// Construct ToolEvent-compatible input
		input = map[string]interface{}{
			"tool_name":       event.ToolName,
			"tool_input":      event.ToolInput,
			"session_id":      sessionID,
			"hook_event_name": "PreToolUse",
			"captured_at":     event.Timestamp,
		}

	case "PostToolUse":
		if r.sharpEdgePath == "" {
			return "", fmt.Errorf("sharpEdgePath not set for PostToolUse event")
		}
		cmdPath = r.sharpEdgePath
		// Construct PostToolUse-compatible input
		input = map[string]interface{}{
			"tool_name":       event.ToolName,
			"tool_input":      event.ToolInput,
			"tool_response":   event.ToolResponse,
			"session_id":      sessionID,
			"hook_event_name": "PostToolUse",
			"captured_at":     event.Timestamp,
		}

	default:
		return "", fmt.Errorf("unknown hook type: %s (expected PreToolUse or PostToolUse)", event.HookType)
	}

	return r.runCLI(cmdPath, input, tempDir)
}

// runCLI executes a CLI with JSON input and returns output.
// Uses the same environment setup pattern as DefaultRunner for consistency.
func (r *SessionReplayer) runCLI(cmdPath string, input interface{}, tempDir string) (string, error) {
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("marshal input: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdPath)
	cmd.Stdin = bytes.NewReader(inputBytes)
	cmd.Env = r.buildEnv(tempDir)

	// Process group setup for clean timeout handling (mirrors DefaultRunner)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	cmd.WaitDelay = 100 * time.Millisecond

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	// Check context timeout first
	if ctx.Err() != nil {
		return "", fmt.Errorf("command timeout: %w", ctx.Err())
	}

	// Non-zero exit is acceptable for sharp-edge (may block)
	if exitErr, ok := err.(*exec.ExitError); ok {
		// Log exit code for debugging but don't fail
		if r.verbose {
			fmt.Printf("[DEBUG] CLI exit code: %d\n", exitErr.ExitCode())
		}
	} else if err != nil {
		return "", fmt.Errorf("execute command: %w (stderr: %s)", err, stderr.String())
	}

	output := stdout.String()
	return output, nil
}

// buildEnv creates a minimal, controlled environment for CLI execution.
func (r *SessionReplayer) buildEnv(tempDir string) []string {
	env := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"USER=" + os.Getenv("USER"),
		"LANG=C.UTF-8",
		"LC_ALL=C.UTF-8",
		"NO_COLOR=1",
		"TERM=dumb",
	}

	// Test isolation paths
	if r.schemaPath != "" {
		env = append(env, "GOGENT_ROUTING_SCHEMA="+r.schemaPath)
	}
	if r.agentsPath != "" {
		env = append(env, "GOGENT_AGENTS_INDEX="+r.agentsPath)
	}
	if tempDir != "" {
		env = append(env, "GOGENT_PROJECT_DIR="+tempDir)
		env = append(env, "GOGENT_STORAGE_PATH="+filepath.Join(tempDir, ".gogent", "failure-tracker.jsonl"))
	}

	// Set max failures for deterministic threshold testing
	env = append(env, "GOGENT_MAX_FAILURES=3")
	// Very long window so time-based expiry doesn't affect tests
	env = append(env, "GOGENT_FAILURE_WINDOW=999999999")

	return env
}

// validateExpectations checks session-level expectations against actual results.
func (r *SessionReplayer) validateExpectations(expected ReplayExpectations, tempDir string, sharpEdges, blocking int) []string {
	var errors []string

	// Check sharp edge count
	if expected.SharpEdgesCreated != sharpEdges {
		errors = append(errors, fmt.Sprintf("sharp_edges_created: expected %d, got %d",
			expected.SharpEdgesCreated, sharpEdges))
	}

	// Check blocking count
	if expected.BlockingResponses != blocking {
		errors = append(errors, fmt.Sprintf("blocking_responses: expected %d, got %d",
			expected.BlockingResponses, blocking))
	}

	// Check handoff creation
	if expected.HandoffCreated {
		handoffPath := filepath.Join(tempDir, ".claude", "memory", "handoffs.jsonl")
		if _, err := os.Stat(handoffPath); os.IsNotExist(err) {
			errors = append(errors, "handoff_created: expected handoffs.jsonl to exist")
		}
	}

	// Check files that must exist
	for _, relPath := range expected.FilesCreated {
		fullPath := filepath.Join(tempDir, relPath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			errors = append(errors, fmt.Sprintf("files_created: %s not found", relPath))
		}
	}

	// Check files that must NOT exist
	for _, relPath := range expected.FilesNotCreated {
		fullPath := filepath.Join(tempDir, relPath)
		if _, err := os.Stat(fullPath); err == nil {
			errors = append(errors, fmt.Sprintf("files_not_created: %s exists (should not)", relPath))
		}
	}

	// Check file contents
	for relPath, substrings := range expected.FileContains {
		fullPath := filepath.Join(tempDir, relPath)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("file_contains: cannot read %s: %v", relPath, err))
			continue
		}
		contentStr := string(content)
		for _, substr := range substrings {
			if !strings.Contains(contentStr, substr) {
				errors = append(errors, fmt.Sprintf("file_contains: %q not found in %s", substr, relPath))
			}
		}
	}

	return errors
}

// loadSessions loads all session fixtures from the sessions directory.
func (r *SessionReplayer) loadSessions() ([]ReplaySession, error) {
	sessionsDir := filepath.Join(r.fixturesDir, "sessions")

	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No sessions directory yet
		}
		return nil, fmt.Errorf("read sessions dir: %w", err)
	}

	var sessions []ReplaySession

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}

		path := filepath.Join(sessionsDir, entry.Name())
		session, err := r.loadSession(path)
		if err != nil {
			return nil, fmt.Errorf("load session %s: %w", entry.Name(), err)
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// loadSession parses a single session file.
// Session files are JSONL where:
// - First line is the session metadata (id, description, expected)
// - Subsequent lines are events
func (r *SessionReplayer) loadSession(path string) (ReplaySession, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return ReplaySession{}, fmt.Errorf("read file: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) == 0 {
		return ReplaySession{}, fmt.Errorf("empty session file")
	}

	// First line is metadata
	var session ReplaySession
	if err := json.Unmarshal([]byte(lines[0]), &session); err != nil {
		return ReplaySession{}, fmt.Errorf("parse metadata: %w", err)
	}

	// Remaining lines are events
	session.Events = nil // Clear any events from metadata line
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}
		var event ReplayEvent
		if err := json.Unmarshal([]byte(lines[i]), &event); err != nil {
			return ReplaySession{}, fmt.Errorf("parse event %d: %w", i, err)
		}
		session.Events = append(session.Events, event)
	}

	// Derive session ID from filename if not set
	if session.ID == "" {
		base := filepath.Base(path)
		session.ID = strings.TrimSuffix(base, ".jsonl")
	}

	return session, nil
}

// countJSONLLines counts non-empty lines in JSONL content.
func countJSONLLines(content []byte) int {
	if len(content) == 0 {
		return 0
	}
	lines := 0
	for _, b := range content {
		if b == '\n' {
			lines++
		}
	}
	// If content doesn't end with newline, count the last line
	if content[len(content)-1] != '\n' {
		lines++
	}
	return lines
}

// truncateOutput limits output length for error messages.
func truncateOutput(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
