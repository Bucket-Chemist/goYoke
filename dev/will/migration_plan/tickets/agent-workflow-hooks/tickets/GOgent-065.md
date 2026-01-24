---
id: GOgent-065
title: Endstate Logging & Decision Storage
description: **Task**:
status: pending
time_estimate: 1.5h
dependencies: ["GOgent-064"]
priority: high
week: 4
tags: ["agent-endstate", "week-4"]
tests_required: true
acceptance_criteria_count: 13
---

### GOgent-065: Endstate Logging & Decision Storage

**Time**: 1.5 hours
**Dependencies**: GOgent-064

**Task**:
Store endstate decisions in JSONL format for analysis and audit trail. Use XDG-compliant paths and integrate with HandoffArtifacts schema.

**File**: `pkg/workflow/logging.go`

**Imports**:
```go
package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
)
```

**Implementation**:
```go
// EndstateLog represents a single logged endstate decision
type EndstateLog struct {
	Timestamp      time.Time `json:"timestamp"`
	AgentID        string    `json:"agent_id"`
	AgentClass     string    `json:"agent_class"`
	Tier           string    `json:"tier"`
	ExitCode       int       `json:"exit_code"`
	Duration       int       `json:"duration_ms"`
	OutputTokens   int       `json:"output_tokens"`
	Decision       string    `json:"decision"` // "prompt" or "silent"
	Recommendations []string  `json:"recommendations"`
}

// GetEndstateLogPath returns XDG-compliant path for endstate logs (global)
func GetEndstateLogPath() string {
	return filepath.Join(config.GetGOgentDir(), "agent-endstates.jsonl")
}

// GetProjectEndstateLogPath returns project-scoped path for endstate logs
func GetProjectEndstateLogPath(projectDir string) string {
	return filepath.Join(projectDir, ".claude", "memory", "agent-endstates.jsonl")
}

// LogEndstate writes endstate decision to JSONL file using XDG-compliant path
func LogEndstate(event *SubagentStopEvent, metadata *ParsedAgentMetadata, response *EndstateResponse) error {
	logPath := GetEndstateLogPath()

	log := EndstateLog{
		Timestamp:       time.Now().UTC(),
		AgentID:         metadata.AgentID,
		AgentClass:      string(event.GetAgentClass()),
		Tier:            metadata.Tier,
		ExitCode:        event.ExitCode,
		Duration:        event.Duration,
		OutputTokens:    event.OutputTokens,
		Decision:        response.Decision,
		Recommendations: response.Recommendations,
	}

	data, err := json.Marshal(log)
	if err != nil {
		return fmt.Errorf("[agent-endstate] Failed to marshal log: %w", err)
	}

	// Append to file
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("[agent-endstate] Failed to open log file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(string(data) + "\n"); err != nil {
		return fmt.Errorf("[agent-endstate] Failed to write log: %w", err)
	}

	return nil
}

// ReadEndstateLogs reads all endstate logs from file
func ReadEndstateLogs() ([]EndstateLog, error) {
	logPath := GetEndstateLogPath()

	data, err := os.ReadFile(logPath)
	if os.IsNotExist(err) {
		return []EndstateLog{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("[agent-endstate] Failed to read logs: %w", err)
	}

	lines := 0
	var logs []EndstateLog
	for _, line := range string(data) {
		if line == '\n' {
			lines++
		}
	}

	// Parse JSONL
	offset := 0
	content := string(data)
	for {
		newlineIdx := -1
		for i := offset; i < len(content); i++ {
			if content[i] == '\n' {
				newlineIdx = i
				break
			}
		}

		if newlineIdx == -1 {
			if offset < len(content) {
				// Last line without newline
				newlineIdx = len(content)
			} else {
				break
			}
		}

		line := content[offset:newlineIdx]
		if line == "" {
			offset = newlineIdx + 1
			continue
		}

		var log EndstateLog
		if err := json.Unmarshal([]byte(line), &log); err != nil {
			// Skip malformed lines
			offset = newlineIdx + 1
			continue
		}

		logs = append(logs, log)
		offset = newlineIdx + 1
	}

	return logs, nil
}

// GetAgentStats returns statistics for a specific agent
func GetAgentStats(agentID string) (int, int, float64, error) {
	logs, err := ReadEndstateLogs()
	if err != nil {
		return 0, 0, 0, err
	}

	var successCount, failureCount int
	var totalDuration int

	for _, log := range logs {
		if log.AgentID != agentID {
			continue
		}

		if log.ExitCode == 0 {
			successCount++
		} else {
			failureCount++
		}
		totalDuration += log.Duration
	}

	totalRuns := successCount + failureCount
	if totalRuns == 0 {
		return 0, 0, 0, nil
	}

	successRate := float64(successCount) / float64(totalRuns) * 100.0

	return successCount, failureCount, successRate, nil
}
```

**HandoffArtifacts Integration**:

Add to `pkg/session/handoff.go`:
```go
type HandoffArtifacts struct {
	// ... existing fields ...

	// v1.3 additions (omitempty for backward compatibility)
	AgentEndstates []EndstateLog `json:"agent_endstates,omitempty"`
}
```

Add to `pkg/session/handoff_artifacts.go`:
```go
// loadEndstates reads agent-endstates.jsonl into artifacts
func loadEndstates(artifacts *HandoffArtifacts, projectDir string) error {
	path := GetEndstateLogPath()
	// ... JSONL parsing following existing pattern from loadSharpEdges()
}
```

**Tests**: `pkg/workflow/logging_test.go`

```go
package workflow

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLogEndstate(t *testing.T) {
	// Use t.TempDir() for test isolation (no global state)
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	event := &SubagentStopEvent{
		AgentID:      "orchestrator",
		AgentModel:   "sonnet",
		Tier:         "sonnet",
		ExitCode:     0,
		Duration:     5000,
		OutputTokens: 2048,
	}

	metadata := &ParsedAgentMetadata{
		AgentID: "orchestrator",
		Tier:    "sonnet",
	}

	response := GenerateEndstateResponse(event)

	// Verify logging works
	if err := LogEndstate(event, metadata, response); err != nil {
		t.Fatalf("LogEndstate failed: %v", err)
	}

	logPath := GetEndstateLogPath()

	// Verify file was created
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}
}

func TestReadEndstateLogs_Empty(t *testing.T) {
	// Use t.TempDir() for test isolation
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// Should return empty list, not error
	logs, err := ReadEndstateLogs()
	if err != nil {
		t.Errorf("ReadEndstateLogs should not error on missing file: %v", err)
	}

	if len(logs) != 0 {
		t.Errorf("Expected empty logs, got: %d entries", len(logs))
	}
}

func TestGetAgentStats_Structure(t *testing.T) {
	// Test that function exists and has correct signature
	_, failCount, successRate, err := GetAgentStats("test-agent")

	if err != nil && !os.IsNotExist(err) {
		// Either nil (no logs) or file not found is OK
	}

	if failCount < 0 {
		t.Error("Failure count should be non-negative")
	}

	if successRate < 0 || successRate > 100 {
		t.Errorf("Success rate should be 0-100, got: %f", successRate)
	}
}
```

**Acceptance Criteria**:
- [ ] Uses XDG-compliant path via `config.GetGOgentDir()` (NOT hardcoded `/tmp/`)
- [ ] `GetEndstateLogPath()` returns global path: `~/.cache/gogent/agent-endstates.jsonl`
- [ ] `GetProjectEndstateLogPath()` returns project path: `.claude/memory/agent-endstates.jsonl`
- [ ] `LogEndstate()` signature accepts metadata parameter
- [ ] `LogEndstate()` writes to XDG-compliant path
- [ ] Appends JSONL format (one JSON per line)
- [ ] Creates file if missing
- [ ] `ReadEndstateLogs()` parses all logs correctly
- [ ] Handles missing file gracefully (returns empty list)
- [ ] `GetAgentStats()` calculates success rate correctly
- [ ] HandoffArtifacts extended with `AgentEndstates []EndstateLog` field (omitempty)
- [ ] Tests use `t.Setenv()` for isolation
- [ ] Tests verify logging, reading, statistics with mocked paths
- [ ] `go test ./pkg/workflow` passes

**Why This Matters**: Logging enables post-session analysis of agent performance and patterns for memory compounding.

---
