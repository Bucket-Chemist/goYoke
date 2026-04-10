package workflow

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/session"
)

// GetEndstateLogPath returns XDG-compliant path for endstate logs (global)
func GetEndstateLogPath() string {
	return filepath.Join(config.GetGOgentDir(), "agent-endstates.jsonl")
}

// GetProjectEndstateLogPath returns project-scoped path for endstate logs
func GetProjectEndstateLogPath(projectDir string) string {
	return filepath.Join(config.ProjectMemoryDir(projectDir), "agent-endstates.jsonl")
}

// LogEndstate writes endstate decision to JSONL file using XDG-compliant path
func LogEndstate(event *routing.SubagentStopEvent, metadata *routing.ParsedAgentMetadata, response *EndstateResponse) error {
	logPath := GetEndstateLogPath()

	// Ensure directory exists
	dir := filepath.Dir(logPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("[agent-endstate] Failed to create directory %s: %w", dir, err)
	}

	log := session.EndstateLog{
		Timestamp:       time.Now().UTC(),
		AgentID:         metadata.AgentID,
		AgentClass:      string(routing.GetAgentClass(metadata.AgentID)),
		Tier:            metadata.Tier,
		ExitCode:        metadata.ExitCode,
		Duration:        metadata.DurationMs,
		OutputTokens:    metadata.OutputTokens,
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

// ReadEndstateLogs reads all endstate logs from file.
// Returns empty slice if file doesn't exist (not an error).
// Uses bufio.Scanner for robust JSONL parsing (matches codebase pattern).
func ReadEndstateLogs() ([]session.EndstateLog, error) {
	logPath := GetEndstateLogPath()

	file, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []session.EndstateLog{}, nil
		}
		return nil, fmt.Errorf("[agent-endstate] Failed to open log file: %w", err)
	}
	defer file.Close()

	var logs []session.EndstateLog
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var log session.EndstateLog
		if err := json.Unmarshal([]byte(line), &log); err != nil {
			// Skip malformed lines (graceful degradation)
			continue
		}

		logs = append(logs, log)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("[agent-endstate] Error reading logs: %w", err)
	}

	return logs, nil
}

// GetAgentStats returns statistics for a specific agent.
// Returns (successCount, failureCount, successRate, error).
// Returns (0, 0, 0, nil) for agents with no runs.
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
