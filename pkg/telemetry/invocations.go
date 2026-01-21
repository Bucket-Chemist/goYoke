package telemetry

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
)

// AgentInvocation captures a single agent execution (success or failure).
// Logged to both global XDG cache and project memory for comprehensive telemetry.
type AgentInvocation struct {
	// Core identification
	Timestamp    string `json:"timestamp"`      // RFC3339 format, auto-populated
	SessionID    string `json:"session_id"`     // Session identifier
	InvocationID string `json:"invocation_id"`  // UUID for this specific invocation

	// Agent context
	Agent string `json:"agent"` // e.g., "python-pro", "orchestrator"
	Model string `json:"model"` // e.g., "haiku", "sonnet", "opus"
	Tier  string `json:"tier"`  // e.g., "haiku_thinking", "sonnet"

	// Performance metrics
	DurationMs     int64 `json:"duration_ms"`
	InputTokens    int   `json:"input_tokens"`
	OutputTokens   int   `json:"output_tokens"`
	ThinkingTokens int   `json:"thinking_tokens,omitempty"`

	// Outcome
	Success   bool   `json:"success"`
	ErrorType string `json:"error_type,omitempty"` // If !Success

	// Task context
	TaskDescription string   `json:"task_description"`           // First 200 chars
	ParentTaskID    string   `json:"parent_task_id,omitempty"`   // For delegation chains
	ToolsUsed       []string `json:"tools_used"`

	// Project context
	ProjectDir string `json:"project_dir,omitempty"`
}

// GetInvocationsLogPath returns the global invocations log path.
// Uses XDG Base Directory compliance: XDG_RUNTIME_DIR > XDG_CACHE_HOME > ~/.cache/gogent
func GetInvocationsLogPath() string {
	baseDir := config.GetGOgentDir()
	return filepath.Join(baseDir, "agent-invocations.jsonl")
}

// GetProjectInvocationsLogPath returns the project-scoped invocations log path.
func GetProjectInvocationsLogPath(projectDir string) string {
	return filepath.Join(projectDir, ".claude", "memory", "agent-invocations.jsonl")
}

// LogInvocation appends invocation to BOTH:
// 1. Global XDG cache: ~/.cache/gogent/agent-invocations.jsonl (survives project deletion)
// 2. Project memory: <project>/.claude/memory/agent-invocations.jsonl (session integration)
//
// Timestamp is auto-populated in RFC3339 format.
// Project log failure does NOT fail the entire operation (graceful degradation).
func LogInvocation(inv *AgentInvocation, projectDir string) error {
	// Auto-populate timestamp
	inv.Timestamp = time.Now().Format(time.RFC3339)

	// Populate project directory if provided
	if projectDir != "" {
		inv.ProjectDir = projectDir
	}

	// Marshal once, write twice
	data, err := json.Marshal(inv)
	if err != nil {
		return fmt.Errorf("[invocations] Failed to marshal invocation: %w", err)
	}
	data = append(data, '\n') // JSONL format

	// WRITE 1: Global XDG cache (primary, required)
	globalPath := GetInvocationsLogPath()
	if err := appendToFile(globalPath, data); err != nil {
		return fmt.Errorf("[invocations] Failed to write global log: %w", err)
	}

	// WRITE 2: Project memory (secondary, optional)
	if projectDir != "" {
		projectPath := GetProjectInvocationsLogPath(projectDir)
		if err := appendToFile(projectPath, data); err != nil {
			// Log warning but don't fail - global write succeeded
			fmt.Fprintf(os.Stderr, "[invocations] Warning: Failed project log: %v\n", err)
		}
	}

	return nil
}

// LoadInvocations reads all invocations from a JSONL file.
// Returns empty slice for missing file (normal case).
func LoadInvocations(path string) ([]AgentInvocation, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []AgentInvocation{}, nil
		}
		return nil, fmt.Errorf("[invocations] Failed to open %s: %w", path, err)
	}
	defer file.Close()

	var invocations []AgentInvocation
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var inv AgentInvocation
		if err := json.Unmarshal([]byte(line), &inv); err != nil {
			// Skip malformed lines but continue
			continue
		}
		invocations = append(invocations, inv)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("[invocations] Error reading %s: %w", path, err)
	}

	return invocations, nil
}

// appendToFile appends data to file (creates if not exists).
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
