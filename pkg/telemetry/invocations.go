package telemetry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/config"
)

// AgentInvocation captures a single agent execution (success or failure).
// Logged to both global XDG cache and project memory for comprehensive telemetry.
type AgentInvocation struct {
	// Core identification
	Timestamp    string `json:"timestamp"`     // RFC3339 format, auto-populated
	SessionID    string `json:"session_id"`    // Session identifier
	InvocationID string `json:"invocation_id"` // UUID for this specific invocation

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
	TaskDescription string   `json:"task_description"`         // First 200 chars
	ParentTaskID    string   `json:"parent_task_id,omitempty"` // For delegation chains
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
	return filepath.Join(config.ProjectMemoryDir(projectDir), "agent-invocations.jsonl")
}

// LogInvocation appends invocation to BOTH:
// 1. Global XDG cache: ~/.cache/gogent/agent-invocations.jsonl (survives project deletion)
// 2. Project memory: <project>/.gogent/memory/agent-invocations.jsonl (session integration)
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
	scanner := newTelemetryScanner(file)

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

// AgentInvocationStats aggregates metrics for a single agent
type AgentInvocationStats struct {
	Agent               string            `json:"agent"`
	TotalCount          int               `json:"total_count"`
	SuccessCount        int               `json:"success_count"`
	FailureCount        int               `json:"failure_count"`
	SuccessRate         float64           `json:"success_rate"`
	AvgDurationMs       int64             `json:"avg_duration_ms"`
	TotalDurationMs     int64             `json:"total_duration_ms"`
	TotalInputTokens    int               `json:"total_input_tokens"`
	TotalOutputTokens   int               `json:"total_output_tokens"`
	TotalThinkingTokens int               `json:"total_thinking_tokens"`
	Samples             []AgentInvocation `json:"samples,omitempty"`
}

// TierInvocationStats aggregates metrics for a model tier
type TierInvocationStats struct {
	Tier              string         `json:"tier"`
	TotalCount        int            `json:"total_count"`
	SuccessCount      int            `json:"success_count"`
	SuccessRate       float64        `json:"success_rate"`
	TotalInputTokens  int            `json:"total_input_tokens"`
	TotalOutputTokens int            `json:"total_output_tokens"`
	AgentBreakdown    map[string]int `json:"agent_breakdown"`
}

// AgentRanking represents an agent with a sortable metric
type AgentRanking struct {
	Agent  string  `json:"agent"`
	Metric float64 `json:"metric"`
	Count  int     `json:"count"`
}

// ClusterInvocationsByAgent groups invocations by agent and computes aggregate statistics.
// Empty agent names are normalized to "unknown".
func ClusterInvocationsByAgent(invocations []AgentInvocation) []AgentInvocationStats {
	agentMap := make(map[string]*AgentInvocationStats)

	for _, inv := range invocations {
		agent := inv.Agent
		if agent == "" {
			agent = "unknown"
		}

		if _, exists := agentMap[agent]; !exists {
			agentMap[agent] = &AgentInvocationStats{
				Agent:   agent,
				Samples: []AgentInvocation{},
			}
		}

		stats := agentMap[agent]
		stats.TotalCount++
		if inv.Success {
			stats.SuccessCount++
		} else {
			stats.FailureCount++
		}
		stats.TotalDurationMs += inv.DurationMs
		stats.TotalInputTokens += inv.InputTokens
		stats.TotalOutputTokens += inv.OutputTokens
		stats.TotalThinkingTokens += inv.ThinkingTokens

		// Keep first 3 invocations as samples
		if len(stats.Samples) < 3 {
			stats.Samples = append(stats.Samples, inv)
		}
	}

	// Calculate derived metrics
	result := make([]AgentInvocationStats, 0, len(agentMap))
	for _, stats := range agentMap {
		if stats.TotalCount > 0 {
			stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalCount)
			stats.AvgDurationMs = stats.TotalDurationMs / int64(stats.TotalCount)
		}
		result = append(result, *stats)
	}

	// Sort by agent name for deterministic output
	sort.Slice(result, func(i, j int) bool {
		return result[i].Agent < result[j].Agent
	})

	return result
}

// ClusterInvocationsByTier groups invocations by model tier and computes aggregate statistics.
// Empty tier names are normalized to "unknown".
func ClusterInvocationsByTier(invocations []AgentInvocation) []TierInvocationStats {
	tierMap := make(map[string]*TierInvocationStats)

	for _, inv := range invocations {
		tier := inv.Tier
		if tier == "" {
			tier = "unknown"
		}

		if _, exists := tierMap[tier]; !exists {
			tierMap[tier] = &TierInvocationStats{
				Tier:           tier,
				AgentBreakdown: make(map[string]int),
			}
		}

		stats := tierMap[tier]
		stats.TotalCount++
		if inv.Success {
			stats.SuccessCount++
		}
		stats.TotalInputTokens += inv.InputTokens
		stats.TotalOutputTokens += inv.OutputTokens

		// Track agent breakdown
		agent := inv.Agent
		if agent == "" {
			agent = "unknown"
		}
		stats.AgentBreakdown[agent]++
	}

	// Calculate derived metrics
	result := make([]TierInvocationStats, 0, len(tierMap))
	for _, stats := range tierMap {
		if stats.TotalCount > 0 {
			stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalCount)
		}
		result = append(result, *stats)
	}

	// Sort by tier name for deterministic output
	sort.Slice(result, func(i, j int) bool {
		return result[i].Tier < result[j].Tier
	})

	return result
}

// GetTopAgentsByUsage returns agents ranked by total invocation count.
// Results are sorted in descending order (highest usage first).
func GetTopAgentsByUsage(stats []AgentInvocationStats, limit int) []AgentRanking {
	rankings := make([]AgentRanking, 0, len(stats))
	for _, s := range stats {
		rankings = append(rankings, AgentRanking{
			Agent:  s.Agent,
			Metric: float64(s.TotalCount),
			Count:  s.TotalCount,
		})
	}

	// Sort by metric descending
	sort.Slice(rankings, func(i, j int) bool {
		return rankings[i].Metric > rankings[j].Metric
	})

	// Apply limit
	if limit > 0 && limit < len(rankings) {
		rankings = rankings[:limit]
	}

	return rankings
}

// GetTopAgentsByErrorRate returns agents ranked by error rate (1 - success_rate).
// Only includes agents with at least minInvocations to avoid skew from small samples.
// Results are sorted in descending order (highest error rate first).
func GetTopAgentsByErrorRate(stats []AgentInvocationStats, minInvocations int, limit int) []AgentRanking {
	rankings := make([]AgentRanking, 0, len(stats))
	for _, s := range stats {
		if s.TotalCount < minInvocations {
			continue
		}
		errorRate := 1.0 - s.SuccessRate
		rankings = append(rankings, AgentRanking{
			Agent:  s.Agent,
			Metric: errorRate,
			Count:  s.TotalCount,
		})
	}

	// Sort by metric descending
	sort.Slice(rankings, func(i, j int) bool {
		return rankings[i].Metric > rankings[j].Metric
	})

	// Apply limit
	if limit > 0 && limit < len(rankings) {
		rankings = rankings[:limit]
	}

	return rankings
}

// GetTopAgentsByLatency returns agents ranked by average duration.
// Results are sorted in descending order (highest latency first).
func GetTopAgentsByLatency(stats []AgentInvocationStats, limit int) []AgentRanking {
	rankings := make([]AgentRanking, 0, len(stats))
	for _, s := range stats {
		rankings = append(rankings, AgentRanking{
			Agent:  s.Agent,
			Metric: float64(s.AvgDurationMs),
			Count:  s.TotalCount,
		})
	}

	// Sort by metric descending
	sort.Slice(rankings, func(i, j int) bool {
		return rankings[i].Metric > rankings[j].Metric
	})

	// Apply limit
	if limit > 0 && limit < len(rankings) {
		rankings = rankings[:limit]
	}

	return rankings
}
