package telemetry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
)

// EscalationEvent captures when an agent escalates to a higher tier.
// Tracks Einstein/Gemini invocations and their outcomes.
type EscalationEvent struct {
	// Core identification
	Timestamp    string `json:"timestamp"` // RFC3339 format
	SessionID    string `json:"session_id"`
	EscalationID string `json:"escalation_id"` // UUID for tracking

	// Escalation context
	FromTier  string `json:"from_tier"`  // e.g., "sonnet"
	ToTier    string `json:"to_tier"`    // e.g., "opus"
	FromAgent string `json:"from_agent"` // e.g., "orchestrator"
	ToAgent   string `json:"to_agent"`   // e.g., "einstein"

	// Trigger information
	Reason      string `json:"reason"`       // e.g., "3 consecutive failures"
	TriggerType string `json:"trigger_type"` // "failure_cascade", "user_request", "complexity"

	// Associated artifacts
	GAPDocPath string `json:"gap_doc_path,omitempty"` // Path to GAP document if generated

	// Outcome tracking (updated after resolution)
	Outcome           string `json:"outcome"` // "pending", "resolved", "still_blocked"
	ResolutionTimeMs  int64  `json:"resolution_time_ms,omitempty"`
	ResolutionSummary string `json:"resolution_summary,omitempty"`
	TokensUsed        int    `json:"tokens_used,omitempty"`

	// Context
	ProjectDir string `json:"project_dir,omitempty"`
}

// Valid trigger types for escalation events
var ValidTriggerTypes = map[string]bool{
	"failure_cascade": true, // 3+ consecutive failures
	"user_request":    true, // User explicitly asked for Einstein
	"complexity":      true, // Scout/orchestrator determined higher tier needed
	"timeout":         true, // Lower tier couldn't complete in time
	"cost_ceiling":    true, // Lower tier exceeded budget without result
}

// Valid outcome states for escalation events
var ValidOutcomes = map[string]bool{
	"pending":       true, // Escalation in progress
	"resolved":      true, // Successfully resolved
	"still_blocked": true, // Einstein couldn't solve it either
	"cancelled":     true, // User cancelled before completion
}

// GetEscalationsLogPath returns the global escalations log path.
func GetEscalationsLogPath() string {
	baseDir := config.GetGOgentDir()
	return filepath.Join(baseDir, "escalations.jsonl")
}

// GetProjectEscalationsLogPath returns project-scoped escalations log path.
func GetProjectEscalationsLogPath(projectDir string) string {
	return filepath.Join(config.ProjectMemoryDir(projectDir), "escalations.jsonl")
}

// LogEscalation appends escalation event to BOTH global and project logs.
// Timestamp is auto-populated. Returns the escalation ID for tracking.
func LogEscalation(esc *EscalationEvent, projectDir string) (string, error) {
	// Auto-populate timestamp
	esc.Timestamp = time.Now().Format(time.RFC3339)

	// Populate project directory if provided
	if projectDir != "" {
		esc.ProjectDir = projectDir
	}

	// Set initial outcome if not specified
	if esc.Outcome == "" {
		esc.Outcome = "pending"
	}

	// Validate trigger type
	if esc.TriggerType != "" && !ValidTriggerTypes[esc.TriggerType] {
		return "", fmt.Errorf("[escalations] Invalid trigger type '%s'. Valid types: failure_cascade, user_request, complexity, timeout, cost_ceiling", esc.TriggerType)
	}

	// Marshal once, write twice
	data, err := json.Marshal(esc)
	if err != nil {
		return "", fmt.Errorf("[escalations] Failed to marshal escalation: %w", err)
	}
	data = append(data, '\n')

	// WRITE 1: Global XDG cache
	globalPath := GetEscalationsLogPath()
	if err := appendToFile(globalPath, data); err != nil {
		return "", fmt.Errorf("[escalations] Failed to write global log: %w", err)
	}

	// WRITE 2: Project memory (optional)
	if projectDir != "" {
		projectPath := GetProjectEscalationsLogPath(projectDir)
		if err := appendToFile(projectPath, data); err != nil {
			fmt.Fprintf(os.Stderr, "[escalations] Warning: Failed project log: %v\n", err)
		}
	}

	return esc.EscalationID, nil
}

// LoadEscalations reads all escalation events from a JSONL file.
// Returns empty slice for missing file.
func LoadEscalations(path string) ([]EscalationEvent, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []EscalationEvent{}, nil
		}
		return nil, fmt.Errorf("[escalations] Failed to open %s: %w", path, err)
	}
	defer file.Close()

	var escalations []EscalationEvent
	scanner := newTelemetryScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var esc EscalationEvent
		if err := json.Unmarshal([]byte(line), &esc); err != nil {
			continue // Skip malformed lines
		}
		escalations = append(escalations, esc)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("[escalations] Error reading %s: %w", path, err)
	}

	return escalations, nil
}

// UpdateEscalationOutcome marks an escalation as resolved or still blocked.
// This rewrites the entire file to update the specific escalation.
// Note: For high-frequency updates, consider a more efficient approach.
func UpdateEscalationOutcome(path string, escalationID string, outcome string, resolutionMs int64, summary string) error {
	if !ValidOutcomes[outcome] {
		return fmt.Errorf("[escalations] Invalid outcome '%s'. Valid: pending, resolved, still_blocked, cancelled", outcome)
	}

	escalations, err := LoadEscalations(path)
	if err != nil {
		return err
	}

	found := false
	for i := range escalations {
		if escalations[i].EscalationID == escalationID {
			escalations[i].Outcome = outcome
			escalations[i].ResolutionTimeMs = resolutionMs
			escalations[i].ResolutionSummary = summary
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("[escalations] Escalation ID '%s' not found in %s", escalationID, path)
	}

	// Rewrite file
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("[escalations] Failed to rewrite %s: %w", path, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, esc := range escalations {
		if err := encoder.Encode(esc); err != nil {
			return fmt.Errorf("[escalations] Failed to write escalation: %w", err)
		}
	}

	return nil
}

// EscalationFilters defines filter criteria for escalation queries.
type EscalationFilters struct {
	TriggerType *string // Filter by trigger type
	Outcome     *string // Filter by outcome
	ToTier      *string // Filter by destination tier
	FromAgent   *string // Filter by source agent
	Since       *int64  // Filter by timestamp (Unix seconds)
	Limit       int     // Maximum results
}

// FilterEscalations applies filters to a list of escalations.
func FilterEscalations(escalations []EscalationEvent, filters EscalationFilters) []EscalationEvent {
	var filtered []EscalationEvent

	for _, esc := range escalations {
		// Apply filters
		if filters.TriggerType != nil && esc.TriggerType != *filters.TriggerType {
			continue
		}
		if filters.Outcome != nil && esc.Outcome != *filters.Outcome {
			continue
		}
		if filters.ToTier != nil && esc.ToTier != *filters.ToTier {
			continue
		}
		if filters.FromAgent != nil && esc.FromAgent != *filters.FromAgent {
			continue
		}
		if filters.Since != nil {
			escTime, _ := time.Parse(time.RFC3339, esc.Timestamp)
			if escTime.Unix() < *filters.Since {
				continue
			}
		}

		filtered = append(filtered, esc)

		if filters.Limit > 0 && len(filtered) >= filters.Limit {
			break
		}
	}

	return filtered
}

// ===== PATTERN ANALYSIS FUNCTIONS =====

// EscalationStats aggregates metrics for escalation analysis.
type EscalationStats struct {
	TotalCount        int     `json:"total_count"`
	ResolvedCount     int     `json:"resolved_count"`
	StillBlockedCount int     `json:"still_blocked_count"`
	PendingCount      int     `json:"pending_count"`
	CancelledCount    int     `json:"cancelled_count"`
	ResolutionRate    float64 `json:"resolution_rate"` // ResolvedCount / (ResolvedCount + StillBlockedCount)
	AvgResolutionMs   int64   `json:"avg_resolution_ms"`
	TotalTokensUsed   int     `json:"total_tokens_used"`
}

// AgentEscalationStats aggregates escalations for a single source agent.
type AgentEscalationStats struct {
	FromAgent        string         `json:"from_agent"`
	TotalCount       int            `json:"total_count"`
	ResolvedCount    int            `json:"resolved_count"`
	ResolutionRate   float64        `json:"resolution_rate"`
	TriggerBreakdown map[string]int `json:"trigger_breakdown"` // TriggerType -> count
}

// TriggerEscalationStats aggregates escalations by trigger type.
type TriggerEscalationStats struct {
	TriggerType        string         `json:"trigger_type"`
	TotalCount         int            `json:"total_count"`
	ResolvedCount      int            `json:"resolved_count"`
	ResolutionRate     float64        `json:"resolution_rate"`
	FromAgentBreakdown map[string]int `json:"from_agent_breakdown"`
}

// EscalationROI represents return on investment for escalations.
type EscalationROI struct {
	TotalEscalations       int     `json:"total_escalations"`
	CompletedCount         int     `json:"completed_count"`
	ResolvedCount          int     `json:"resolved_count"`
	StillBlockedCount      int     `json:"still_blocked_count"`
	ResolutionRate         float64 `json:"resolution_rate"` // Resolved / Completed
	TotalTokensUsed        int     `json:"total_tokens_used"`
	AvgTokensPerEscalation int     `json:"avg_tokens_per_escalation"`
	EstimatedCostSaved     float64 `json:"estimated_cost_saved"` // If hadn't escalated (rough estimate)
}

// LatencyStats provides timing analysis for escalation resolution.
type LatencyStats struct {
	SampleCount int   `json:"sample_count"` // Escalations with resolution time
	MinMs       int64 `json:"min_ms"`
	MaxMs       int64 `json:"max_ms"`
	AvgMs       int64 `json:"avg_ms"`
	MedianMs    int64 `json:"median_ms"`
	P90Ms       int64 `json:"p90_ms"` // 90th percentile
}

// EscalationTrend shows whether escalations are increasing or decreasing.
type EscalationTrend struct {
	EarlyCount    int     `json:"early_count"`
	LateCount     int     `json:"late_count"`
	Trend         string  `json:"trend"`          // "improving", "stable", "worsening"
	ChangePercent float64 `json:"change_percent"` // Percentage change
	Message       string  `json:"message"`
}

// ClusterEscalationsByFromAgent groups escalations by source agent.
// Identifies which agents escalate most frequently.
func ClusterEscalationsByFromAgent(escalations []EscalationEvent) map[string]*AgentEscalationStats {
	clusters := make(map[string]*AgentEscalationStats)

	for _, esc := range escalations {
		agent := esc.FromAgent
		if agent == "" {
			agent = "unknown"
		}

		stats, exists := clusters[agent]
		if !exists {
			stats = &AgentEscalationStats{
				FromAgent:        agent,
				TriggerBreakdown: make(map[string]int),
			}
			clusters[agent] = stats
		}

		stats.TotalCount++
		if esc.Outcome == "resolved" {
			stats.ResolvedCount++
		}
		stats.TriggerBreakdown[esc.TriggerType]++
	}

	// Calculate resolution rates
	for _, stats := range clusters {
		completedCount := 0
		for _, esc := range escalations {
			if esc.FromAgent == stats.FromAgent && (esc.Outcome == "resolved" || esc.Outcome == "still_blocked") {
				completedCount++
			}
		}
		if completedCount > 0 {
			stats.ResolutionRate = float64(stats.ResolvedCount) / float64(completedCount)
		}
	}

	return clusters
}

// ClusterEscalationsByTrigger groups escalations by trigger type.
// Identifies common escalation patterns.
func ClusterEscalationsByTrigger(escalations []EscalationEvent) map[string]*TriggerEscalationStats {
	clusters := make(map[string]*TriggerEscalationStats)

	for _, esc := range escalations {
		trigger := esc.TriggerType
		if trigger == "" {
			trigger = "unknown"
		}

		stats, exists := clusters[trigger]
		if !exists {
			stats = &TriggerEscalationStats{
				TriggerType:        trigger,
				FromAgentBreakdown: make(map[string]int),
			}
			clusters[trigger] = stats
		}

		stats.TotalCount++
		if esc.Outcome == "resolved" {
			stats.ResolvedCount++
		}
		stats.FromAgentBreakdown[esc.FromAgent]++
	}

	// Calculate resolution rates
	for trigger, stats := range clusters {
		completedCount := 0
		for _, esc := range escalations {
			if esc.TriggerType == trigger && (esc.Outcome == "resolved" || esc.Outcome == "still_blocked") {
				completedCount++
			}
		}
		if completedCount > 0 {
			stats.ResolutionRate = float64(stats.ResolvedCount) / float64(completedCount)
		}
	}

	return clusters
}

// CalculateEscalationROI measures the effectiveness of escalations.
func CalculateEscalationROI(escalations []EscalationEvent) EscalationROI {
	roi := EscalationROI{
		TotalEscalations: len(escalations),
	}

	for _, esc := range escalations {
		switch esc.Outcome {
		case "resolved":
			roi.CompletedCount++
			roi.ResolvedCount++
		case "still_blocked":
			roi.CompletedCount++
			roi.StillBlockedCount++
		}
		roi.TotalTokensUsed += esc.TokensUsed
	}

	if roi.CompletedCount > 0 {
		roi.ResolutionRate = float64(roi.ResolvedCount) / float64(roi.CompletedCount)
	}

	if roi.TotalEscalations > 0 {
		roi.AvgTokensPerEscalation = roi.TotalTokensUsed / roi.TotalEscalations
	}

	// Rough estimate: if resolved, saved ~5 more Sonnet attempts
	// 5 attempts * ~2000 tokens * $0.009/1K = $0.09 per resolution
	roi.EstimatedCostSaved = float64(roi.ResolvedCount) * 0.09

	return roi
}

// AnalyzeEscalationLatency calculates timing statistics for resolved escalations.
func AnalyzeEscalationLatency(escalations []EscalationEvent) LatencyStats {
	var latencies []int64

	for _, esc := range escalations {
		if esc.ResolutionTimeMs > 0 {
			latencies = append(latencies, esc.ResolutionTimeMs)
		}
	}

	stats := LatencyStats{
		SampleCount: len(latencies),
	}

	if len(latencies) == 0 {
		return stats
	}

	// Sort for percentile calculations
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	stats.MinMs = latencies[0]
	stats.MaxMs = latencies[len(latencies)-1]

	// Calculate average
	var total int64
	for _, l := range latencies {
		total += l
	}
	stats.AvgMs = total / int64(len(latencies))

	// Calculate median
	mid := len(latencies) / 2
	if len(latencies)%2 == 0 {
		stats.MedianMs = (latencies[mid-1] + latencies[mid]) / 2
	} else {
		stats.MedianMs = latencies[mid]
	}

	// Calculate P90
	p90Index := int(float64(len(latencies)) * 0.9)
	if p90Index >= len(latencies) {
		p90Index = len(latencies) - 1
	}
	stats.P90Ms = latencies[p90Index]

	return stats
}

// GetEscalationTrend analyzes whether escalation frequency is changing.
// Compares first half vs second half of escalations by timestamp.
func GetEscalationTrend(escalations []EscalationEvent) EscalationTrend {
	if len(escalations) < 2 {
		return EscalationTrend{
			Trend:   "insufficient_data",
			Message: "Not enough escalations to analyze trend",
		}
	}

	// Sort by timestamp
	sorted := make([]EscalationEvent, len(escalations))
	copy(sorted, escalations)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp < sorted[j].Timestamp
	})

	// Parse timestamps and find midpoint
	firstTime, _ := time.Parse(time.RFC3339, sorted[0].Timestamp)
	lastTime, _ := time.Parse(time.RFC3339, sorted[len(sorted)-1].Timestamp)
	midpoint := firstTime.Add(lastTime.Sub(firstTime) / 2)

	// Count early vs late
	earlyCount := 0
	lateCount := 0
	for _, esc := range sorted {
		escTime, _ := time.Parse(time.RFC3339, esc.Timestamp)
		if escTime.Before(midpoint) {
			earlyCount++
		} else {
			lateCount++
		}
	}

	trend := EscalationTrend{
		EarlyCount: earlyCount,
		LateCount:  lateCount,
	}

	if earlyCount == 0 {
		trend.Trend = "insufficient_data"
		trend.Message = "No early escalations for comparison"
		return trend
	}

	trend.ChangePercent = float64(lateCount-earlyCount) / float64(earlyCount) * 100

	if lateCount < earlyCount {
		trend.Trend = "improving"
		reduction := -trend.ChangePercent
		trend.Message = fmt.Sprintf("Improving: %.0f%% reduction in escalations", reduction)
	} else if lateCount > earlyCount {
		trend.Trend = "worsening"
		trend.Message = fmt.Sprintf("Worsening: %.0f%% increase in escalations", trend.ChangePercent)
	} else {
		trend.Trend = "stable"
		trend.Message = "Stable: consistent escalation rate"
	}

	return trend
}
