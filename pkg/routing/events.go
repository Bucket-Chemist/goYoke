package routing

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ToolEvent represents PreToolUse events from Claude Code hooks.
// These events are emitted before a tool is invoked.
type ToolEvent struct {
	ToolName      string                 `json:"tool_name"`
	ToolInput     map[string]interface{} `json:"tool_input"`
	SessionID     string                 `json:"session_id"`
	HookEventName string                 `json:"hook_event_name"`
	CapturedAt    int64                  `json:"captured_at"`

	// v2.1.69 common fields
	CWD            string `json:"cwd,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	AgentID        string `json:"agent_id,omitempty"`
	AgentType      string `json:"agent_type,omitempty"`
	TranscriptPath string `json:"transcript_path,omitempty"`
}

// PostToolEvent represents PostToolUse events with execution results.
// These events include both the input and the tool's response.
type PostToolEvent struct {
	// Core fields (DO NOT MODIFY - backward compatibility)
	ToolName      string                 `json:"tool_name"`
	ToolInput     map[string]interface{} `json:"tool_input"`
	ToolResponse  map[string]interface{} `json:"tool_response"`
	SessionID     string                 `json:"session_id"`
	HookEventName string                 `json:"hook_event_name"`
	CapturedAt    int64                  `json:"captured_at"`

	// v2.1.69 common fields
	CWD            string `json:"cwd,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	AgentID        string `json:"agent_id,omitempty"`
	AgentType      string `json:"agent_type,omitempty"`

	// === ML Telemetry Fields (GOgent-086b) ===
	// All omitempty for backward compatibility

	// Performance metrics
	DurationMs   int64 `json:"duration_ms,omitempty"`
	InputTokens  int   `json:"input_tokens,omitempty"`
	OutputTokens int   `json:"output_tokens,omitempty"`

	// Model context
	Model string `json:"model,omitempty"`
	Tier  string `json:"tier,omitempty"`

	// Outcome
	Success bool `json:"success,omitempty"`

	// Sequence tracking (GAP 4.2)
	SequenceIndex    int      `json:"sequence_index,omitempty"`
	PreviousTools    []string `json:"previous_tools,omitempty"`
	PreviousOutcomes []bool   `json:"previous_outcomes,omitempty"`

	// Task classification (GAP 4.4)
	TaskType   string `json:"task_type,omitempty"`
	TaskDomain string `json:"task_domain,omitempty"`

	// Routing info (for Task() events)
	SelectedTier  string `json:"selected_tier,omitempty"`
	SelectedAgent string `json:"selected_agent,omitempty"`

	// Correlation
	EventID string `json:"event_id,omitempty"`

	// Understanding context (Addendum A.4)
	TargetSize       int64   `json:"target_size,omitempty"`
	CoverageAchieved float64 `json:"coverage_achieved,omitempty"`
	EntitiesFound    int     `json:"entities_found,omitempty"`
}

// VALIDATION NOTES (GOgent-006, updated GOgent-v2169):
//
// Struct validated against 100+ real production events from event-corpus.json.
// All events conform to this structure. Key validation findings:
//
// 1. CWD Field: Added in v2.1.69 as a common field on all hook events.
// 2. Timestamp: All events use "captured_at" (Unix epoch int64). No alternatives found.
// 3. ToolInput: Always a JSON object (map[string]interface{}). Never null or string.
// 4. Field Visibility: Core 5 fields required and present in 100% of events.
// 5. PostToolUse: Adds tool_response field (map[string]interface{}) to base structure.
// 6. v2.1.69: Added common fields: cwd, permission_mode, agent_id, agent_type.
//    agent_id/agent_type present only when running inside a subagent or with --agent.
//
// Corpus coverage: Task, Read, Write, Edit, Bash, Glob, Grep tools.
// Event types: PreToolUse, PostToolUse.
// Original validation date: 2026-01-16
// Schema update: 2026-03-06 (v2.1.69 compatibility)

// ToolEvent Helper Methods (GOgent-080)

// ExtractFilePath gets file_path from tool_input.
// Returns empty string if file_path is not present or not a string.
func (e *ToolEvent) ExtractFilePath() string {
	if path, ok := e.ToolInput["file_path"].(string); ok {
		return path
	}
	return ""
}

// ExtractWriteContent gets content for Write tool or new_string for Edit tool.
// Returns empty string if neither field is present or not a string.
func (e *ToolEvent) ExtractWriteContent() string {
	// Write tool uses "content" field
	if content, ok := e.ToolInput["content"].(string); ok {
		return content
	}
	// Edit tool uses "new_string" field
	if newStr, ok := e.ToolInput["new_string"].(string); ok {
		return newStr
	}
	return ""
}

// IsClaudeMDFile checks if target is a CLAUDE.md file (or variant like CLAUDE.en.md).
// Returns false if file_path cannot be extracted.
func (e *ToolEvent) IsClaudeMDFile() bool {
	path := e.ExtractFilePath()
	if path == "" {
		return false
	}
	filename := filepath.Base(path)
	return filename == "CLAUDE.md" ||
		(strings.HasPrefix(filename, "CLAUDE.") && strings.HasSuffix(filename, ".md"))
}

// IsWriteOperation checks if this is a Write or Edit operation.
func (e *ToolEvent) IsWriteOperation() bool {
	return e.ToolName == "Write" || e.ToolName == "Edit"
}

// TaskInput represents Task tool_input structure.
// This is the specific structure for Task tool invocations.
type TaskInput struct {
	Model           string `json:"model"`
	Prompt          string `json:"prompt"`
	SubagentType    string `json:"subagent_type"`
	Description     string `json:"description"`
	MaxTurns        int    `json:"max_turns,omitempty"`
	RunInBackground bool   `json:"run_in_background,omitempty"`
	Resume          string `json:"resume,omitempty"`
}

// ParseToolEvent reads JSON from io.Reader and parses into ToolEvent.
// Returns an error if JSON parsing fails or required fields are missing.
// Uses ReadStdin for timeout protection.
func ParseToolEvent(r io.Reader, timeout time.Duration) (*ToolEvent, error) {
	data, err := ReadStdin(r, timeout)
	if err != nil {
		return nil, err
	}

	var event ToolEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf(
			"[event-parser] Failed to parse JSON: %w. Input: %s. Ensure hook receives valid JSON from Claude Code.",
			err,
			truncate(data, 200),
		)
	}

	// Validate required fields
	if event.ToolName == "" {
		return nil, fmt.Errorf(
			"[event-parser] Missing tool_name. Event incomplete: %s. Ensure hook emits complete ToolEvent structure.",
			truncate(data, 200),
		)
	}

	if event.HookEventName == "" {
		return nil, fmt.Errorf(
			"[event-parser] Missing hook_event_name. Event incomplete: %s. Ensure hook emits complete ToolEvent structure.",
			truncate(data, 200),
		)
	}

	return &event, nil
}

// ParsePostToolEvent reads and parses PostToolUse events.
// Returns an error if JSON parsing fails or required fields are missing.
// Uses ReadStdin for timeout protection.
func ParsePostToolEvent(r io.Reader, timeout time.Duration) (*PostToolEvent, error) {
	data, err := ReadStdin(r, timeout)
	if err != nil {
		return nil, err
	}

	var event PostToolEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf(
			"[event-parser] Failed to parse JSON: %w. Input: %s. Ensure hook receives valid JSON from Claude Code.",
			err,
			truncate(data, 200),
		)
	}

	// Validate required fields
	if event.ToolName == "" {
		return nil, fmt.Errorf(
			"[event-parser] Missing tool_name. Event incomplete: %s. Ensure hook emits complete ToolEvent structure.",
			truncate(data, 200),
		)
	}

	if event.HookEventName == "" {
		return nil, fmt.Errorf(
			"[event-parser] Missing hook_event_name. Event incomplete: %s. Ensure hook emits complete ToolEvent structure.",
			truncate(data, 200),
		)
	}

	if event.ToolResponse == nil {
		return nil, fmt.Errorf(
			"[event-parser] Missing tool_response. PostToolUse event incomplete: %s. Ensure hook emits complete PostToolEvent structure.",
			truncate(data, 200),
		)
	}

	return &event, nil
}

// ParseTaskInput extracts Task parameters from tool_input map.
// Returns an error if the prompt field is missing (required).
// Other fields are optional and may be empty strings.
func ParseTaskInput(toolInput map[string]interface{}) (*TaskInput, error) {
	// Marshal to JSON bytes
	data, err := json.Marshal(toolInput)
	if err != nil {
		return nil, fmt.Errorf(
			"[event-parser] Failed to marshal tool_input: %w. Input may contain non-serializable types.",
			err,
		)
	}

	// Unmarshal to TaskInput struct
	var taskInput TaskInput
	if err := json.Unmarshal(data, &taskInput); err != nil {
		return nil, fmt.Errorf(
			"[event-parser] Failed to parse Task tool_input: %w. Input: %s. Ensure Task tool_input follows expected schema.",
			err,
			truncate(data, 200),
		)
	}

	// Validate required field
	if taskInput.Prompt == "" {
		return nil, fmt.Errorf(
			"[event-parser] Task input missing required field 'prompt'. Input: %s. Task invocations must include prompt.",
			truncate(data, 200),
		)
	}

	return &taskInput, nil
}

// SubagentStopEvent represents the Claude Code SubagentStop hook event.
// As of v2.1.69, agent metadata is available directly in event fields.
// For pre-v2.1.69 compatibility, metadata can also be extracted from transcript file.
// Schema validated via GOgent-063a research, updated GOgent-v2169.
type SubagentStopEvent struct {
	HookEventName  string `json:"hook_event_name"` // Always "SubagentStop"
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"` // Session transcript path
	StopHookActive bool   `json:"stop_hook_active"`

	// v2.1.69 common + SubagentStop-specific fields
	AgentID              string `json:"agent_id,omitempty"`               // Unique subagent identifier
	AgentType            string `json:"agent_type,omitempty"`             // Agent name (e.g., "Explore", "GO Pro")
	AgentTranscriptPath  string `json:"agent_transcript_path,omitempty"`  // Subagent's own transcript (separate from session)
	LastAssistantMessage string `json:"last_assistant_message,omitempty"` // Final response text
	CWD                  string `json:"cwd,omitempty"`
	PermissionMode       string `json:"permission_mode,omitempty"`
}

// ParsedAgentMetadata contains agent information extracted from transcript file.
// All fields are optional as transcript parsing may fail.
type ParsedAgentMetadata struct {
	AgentID      string `json:"agent_id,omitempty"`      // e.g., "orchestrator", "python-pro"
	AgentModel   string `json:"agent_model,omitempty"`   // "haiku", "sonnet", "opus"
	Tier         string `json:"tier,omitempty"`          // Derived from model
	DurationMs   int    `json:"duration_ms,omitempty"`   // Calculated from transcript timestamps
	OutputTokens int    `json:"output_tokens,omitempty"` // From transcript if available
	ExitCode     int    `json:"exit_code,omitempty"`     // 0=success, derived from completion status
}

// AgentClass represents agent classification
type AgentClass string

const (
	ClassOrchestrator   AgentClass = "orchestrator"
	ClassImplementation AgentClass = "implementation"
	ClassSpecialist     AgentClass = "specialist"
	ClassCoordination   AgentClass = "coordination"
	ClassAnalysis       AgentClass = "analysis"
	ClassReview         AgentClass = "review"
	ClassUnknown        AgentClass = "unknown"
)

// GetAgentClass returns the class of agent based on agent_id.
// Orchestrator-class agents may spawn background tasks and need collection validation.
func GetAgentClass(agentID string) AgentClass {
	switch agentID {
	case "orchestrator", "architect", "einstein", "planner", "mozart",
		"review-orchestrator", "impl-manager", "python-architect":
		return ClassOrchestrator
	case "python-pro", "python-ux", "go-pro", "go-cli", "go-tui", "go-api",
		"go-concurrent", "r-pro", "r-shiny-pro", "typescript-pro", "react-pro":
		return ClassImplementation
	case "code-reviewer", "librarian", "tech-docs-writer", "scaffolder",
		"backend-reviewer", "frontend-reviewer", "standards-reviewer", "memory-archivist":
		return ClassSpecialist
	case "codebase-search", "haiku-scout":
		return ClassCoordination
	case "beethoven", "staff-architect-critical-review":
		return ClassAnalysis
	default:
		return ClassUnknown
	}
}

// ParseSubagentStopEvent reads SubagentStop event from STDIN using ACTUAL schema
func ParseSubagentStopEvent(r io.Reader, timeout time.Duration) (*SubagentStopEvent, error) {
	data, err := ReadStdin(r, timeout)
	if err != nil {
		return nil, err
	}

	var event SubagentStopEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("[agent-endstate] Failed to parse JSON: %w. Input: %s", err, truncate(data, 100))
	}

	// Validate required fields (ACTUAL schema)
	if event.SessionID == "" {
		return nil, fmt.Errorf("[agent-endstate] Missing required field: session_id")
	}
	if event.TranscriptPath == "" && event.AgentTranscriptPath == "" {
		return nil, fmt.Errorf("[agent-endstate] Missing required field: transcript_path or agent_transcript_path")
	}
	if event.HookEventName != "SubagentStop" {
		return nil, fmt.Errorf("[agent-endstate] Invalid hook_event_name: %s (expected SubagentStop)", event.HookEventName)
	}

	return &event, nil
}

// ParseTranscriptForMetadata reads transcript file and extracts agent metadata.
// Returns partial metadata on parsing errors rather than failing completely (graceful degradation).
func ParseTranscriptForMetadata(transcriptPath string) (*ParsedAgentMetadata, error) {
	metadata := &ParsedAgentMetadata{
		ExitCode: 0, // Default to success
	}

	// Check if file exists
	if _, err := os.Stat(transcriptPath); os.IsNotExist(err) {
		return metadata, fmt.Errorf("[agent-endstate] Transcript file not found: %s", transcriptPath)
	}

	file, err := os.Open(transcriptPath)
	if err != nil {
		return metadata, fmt.Errorf("[agent-endstate] Failed to open transcript: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var firstTimestamp, lastTimestamp int64

	for scanner.Scan() {
		line := scanner.Text()

		// Parse JSONL line
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // Skip malformed lines
		}

		// Extract agent_id from AGENT: prefix or task delegation
		if content, ok := entry["content"].(string); ok {
			if strings.HasPrefix(content, "AGENT: ") {
				metadata.AgentID = strings.TrimSpace(strings.TrimPrefix(content, "AGENT: "))
			}
		}

		// Extract model from task delegation
		if model, ok := entry["model"].(string); ok {
			metadata.AgentModel = model
			metadata.Tier = deriveTierFromModel(model)
		}

		// Track timestamps for duration calculation
		if ts, ok := entry["timestamp"].(float64); ok {
			if firstTimestamp == 0 {
				firstTimestamp = int64(ts)
			}
			lastTimestamp = int64(ts)
		}

		// Check for errors or failures
		if role, ok := entry["role"].(string); ok {
			if role == "error" {
				metadata.ExitCode = 1
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return metadata, fmt.Errorf("[agent-endstate] Error reading transcript: %w", err)
	}

	// Calculate duration
	if firstTimestamp > 0 && lastTimestamp > firstTimestamp {
		metadata.DurationMs = int(lastTimestamp - firstTimestamp)
	}

	return metadata, nil
}

// deriveTierFromModel maps model names to tiers
func deriveTierFromModel(model string) string {
	model = strings.ToLower(model)
	if strings.Contains(model, "haiku") {
		return "haiku"
	}
	if strings.Contains(model, "sonnet") {
		return "sonnet"
	}
	if strings.Contains(model, "opus") {
		return "opus"
	}
	return "unknown"
}

// IsSuccess returns true if agent completed successfully (derived from metadata)
func (m *ParsedAgentMetadata) IsSuccess() bool {
	return m.ExitCode == 0
}

// EnrichMetadataFromEvent populates agent metadata from direct event fields (v2.1.69+),
// falling back to transcript parsing when event fields are absent.
// Always parses transcript for duration, tokens, and exit code (not in event fields).
func EnrichMetadataFromEvent(event *SubagentStopEvent) (*ParsedAgentMetadata, error) {
	metadata := &ParsedAgentMetadata{ExitCode: 0}
	hasDirectFields := false

	// Priority 1: Direct event fields (zero I/O)
	if event.AgentID != "" {
		metadata.AgentID = event.AgentID
		hasDirectFields = true
	}
	if event.AgentType != "" && metadata.AgentID == "" {
		// Normalize human-readable type to kebab-case ID
		metadata.AgentID = normalizeAgentType(event.AgentType)
		hasDirectFields = true
	}

	// Priority 2: Transcript parsing for duration, exit code, and fallback ID
	// Use agent_transcript_path if available (v2.1.69+), else session transcript_path
	transcriptPath := event.TranscriptPath
	if event.AgentTranscriptPath != "" {
		transcriptPath = event.AgentTranscriptPath
	}

	fileMetadata, fileErr := ParseTranscriptForMetadata(transcriptPath)
	if fileErr != nil {
		if !hasDirectFields {
			return metadata, fileErr
		}
		// Event fields available but transcript missing — return partial metadata
		return metadata, nil
	}

	// Merge: event fields take precedence for identity
	if metadata.AgentID == "" {
		metadata.AgentID = fileMetadata.AgentID
	}
	if metadata.AgentModel == "" {
		metadata.AgentModel = fileMetadata.AgentModel
	}
	if metadata.Tier == "" {
		metadata.Tier = fileMetadata.Tier
	}

	// Always use transcript-derived metrics (not available in event fields)
	metadata.DurationMs = fileMetadata.DurationMs
	metadata.OutputTokens = fileMetadata.OutputTokens
	metadata.ExitCode = fileMetadata.ExitCode

	return metadata, nil
}

// normalizeAgentType converts human-readable agent type to kebab-case ID.
// Strips parenthetical suffixes before normalization.
// Examples: "GO Pro" → "go-pro", "GO CLI (Cobra)" → "go-cli", "Python UX (PySide6)" → "python-ux"
func normalizeAgentType(agentType string) string {
	// Strip parenthetical suffix: "GO CLI (Cobra)" → "GO CLI"
	if idx := strings.Index(agentType, "("); idx > 0 {
		agentType = strings.TrimSpace(agentType[:idx])
	}
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(agentType), " ", "-"))
}

// truncate limits data to maxLen for error messages.
// Appends "... (truncated)" if data exceeds maxLen.
func truncate(data []byte, maxLen int) string {
	if len(data) <= maxLen {
		return string(data)
	}
	return string(data[:maxLen]) + "... (truncated)"
}
