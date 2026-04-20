package telemetry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/config"
)

// ScopeMetrics captures the scope assessment from a scout run.
type ScopeMetrics struct {
	TotalFiles      int `json:"total_files"`
	TotalLines      int `json:"total_lines"`
	EstimatedTokens int `json:"estimated_tokens"`
}

// ScoutRecommendation captures a routing recommendation from a scout.
// Tracks whether the recommendation was followed and the outcome.
type ScoutRecommendation struct {
	// Core identification
	Timestamp        string `json:"timestamp"` // RFC3339 format
	SessionID        string `json:"session_id"`
	RecommendationID string `json:"recommendation_id"` // UUID

	// Scout context
	ScoutType       string `json:"scout_type"`       // "haiku-scout", "gemini-scout"
	TaskDescription string `json:"task_description"` // First 200 chars of task

	// Recommendation
	RecommendedTier string       `json:"recommended_tier"`
	Confidence      float64      `json:"confidence"` // 0.0-1.0
	ScopeMetrics    ScopeMetrics `json:"scope_metrics"`

	// Compliance tracking
	ActualTier     string `json:"actual_tier"`
	Followed       bool   `json:"followed"`
	FollowedReason string `json:"followed_reason,omitempty"` // Why deviated if !Followed

	// Outcome tracking (added after task completion)
	TaskOutcome  string `json:"task_outcome,omitempty"` // "success", "failure", "escalated"
	OutcomeNotes string `json:"outcome_notes,omitempty"`

	// Context
	ProjectDir string `json:"project_dir,omitempty"`
}

// Valid scout types
var ValidScoutTypes = map[string]bool{
	"haiku-scout":  true,
	"gemini-scout": true,
}

// Valid task outcomes
var ValidTaskOutcomes = map[string]bool{
	"success":   true,
	"failure":   true,
	"escalated": true,
	"":          true, // Outcome not yet recorded
}

// GetScoutLogPath returns the global scout recommendations log path.
func GetScoutLogPath() string {
	baseDir := config.GetgoYokeDir()
	return filepath.Join(baseDir, "scout-recommendations.jsonl")
}

// GetProjectScoutLogPath returns project-scoped scout log path.
func GetProjectScoutLogPath(projectDir string) string {
	return filepath.Join(config.ProjectMemoryDir(projectDir), "scout-recommendations.jsonl")
}

// LogScoutRecommendation appends scout recommendation to BOTH global and project logs.
// Automatically determines if recommendation was followed based on tier match.
func LogScoutRecommendation(rec *ScoutRecommendation, projectDir string) (string, error) {
	// Auto-populate timestamp
	rec.Timestamp = time.Now().Format(time.RFC3339)

	// Populate project directory
	if projectDir != "" {
		rec.ProjectDir = projectDir
	}

	// Validate scout type
	if rec.ScoutType != "" && !ValidScoutTypes[rec.ScoutType] {
		return "", fmt.Errorf("[scout] Invalid scout type '%s'. Valid types: haiku-scout, gemini-scout", rec.ScoutType)
	}

	// Validate confidence range
	if rec.Confidence < 0.0 || rec.Confidence > 1.0 {
		return "", fmt.Errorf("[scout] Invalid confidence %f. Must be between 0.0 and 1.0", rec.Confidence)
	}

	// Auto-determine if followed (if both tiers are set)
	if rec.RecommendedTier != "" && rec.ActualTier != "" {
		rec.Followed = (rec.RecommendedTier == rec.ActualTier)
	}

	// Marshal once, write twice
	data, err := json.Marshal(rec)
	if err != nil {
		return "", fmt.Errorf("[scout] Failed to marshal recommendation: %w", err)
	}
	data = append(data, '\n')

	// WRITE 1: Global XDG cache
	globalPath := GetScoutLogPath()
	if err := appendToFile(globalPath, data); err != nil {
		return "", fmt.Errorf("[scout] Failed to write global log: %w", err)
	}

	// WRITE 2: Project memory (optional)
	if projectDir != "" {
		projectPath := GetProjectScoutLogPath(projectDir)
		if err := appendToFile(projectPath, data); err != nil {
			fmt.Fprintf(os.Stderr, "[scout] Warning: Failed project log: %v\n", err)
		}
	}

	return rec.RecommendationID, nil
}

// LoadScoutRecommendations reads all scout recommendations from a JSONL file.
// Returns empty slice for missing file.
func LoadScoutRecommendations(path string) ([]ScoutRecommendation, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []ScoutRecommendation{}, nil
		}
		return nil, fmt.Errorf("[scout] Failed to open %s: %w", path, err)
	}
	defer file.Close()

	var recommendations []ScoutRecommendation
	scanner := newTelemetryScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var rec ScoutRecommendation
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			continue // Skip malformed lines
		}
		recommendations = append(recommendations, rec)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("[scout] Error reading %s: %w", path, err)
	}

	return recommendations, nil
}

// UpdateScoutOutcome records the outcome of a task after scout recommendation.
// Rewrites file to update the specific recommendation.
func UpdateScoutOutcome(path string, recID string, outcome string, notes string) error {
	if outcome != "" && !ValidTaskOutcomes[outcome] {
		return fmt.Errorf("[scout] Invalid outcome '%s'. Valid: success, failure, escalated", outcome)
	}

	recommendations, err := LoadScoutRecommendations(path)
	if err != nil {
		return err
	}

	found := false
	for i := range recommendations {
		if recommendations[i].RecommendationID == recID {
			recommendations[i].TaskOutcome = outcome
			recommendations[i].OutcomeNotes = notes
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("[scout] Recommendation ID '%s' not found in %s", recID, path)
	}

	// Rewrite file
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("[scout] Failed to rewrite %s: %w", path, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, rec := range recommendations {
		if err := encoder.Encode(rec); err != nil {
			return fmt.Errorf("[scout] Failed to write recommendation: %w", err)
		}
	}

	return nil
}

// ScoutFilters defines filter criteria for scout recommendation queries.
type ScoutFilters struct {
	ScoutType       *string  // Filter by scout type
	Followed        *bool    // Filter by compliance
	RecommendedTier *string  // Filter by recommended tier
	TaskOutcome     *string  // Filter by outcome
	MinConfidence   *float64 // Filter by minimum confidence
	Since           *int64   // Filter by timestamp (Unix seconds)
	Limit           int      // Maximum results
}

// FilterScoutRecommendations applies filters to a list of recommendations.
func FilterScoutRecommendations(recommendations []ScoutRecommendation, filters ScoutFilters) []ScoutRecommendation {
	var filtered []ScoutRecommendation

	for _, rec := range recommendations {
		// Apply filters
		if filters.ScoutType != nil && rec.ScoutType != *filters.ScoutType {
			continue
		}
		if filters.Followed != nil && rec.Followed != *filters.Followed {
			continue
		}
		if filters.RecommendedTier != nil && rec.RecommendedTier != *filters.RecommendedTier {
			continue
		}
		if filters.TaskOutcome != nil && rec.TaskOutcome != *filters.TaskOutcome {
			continue
		}
		if filters.MinConfidence != nil && rec.Confidence < *filters.MinConfidence {
			continue
		}
		if filters.Since != nil {
			recTime, _ := time.Parse(time.RFC3339, rec.Timestamp)
			if recTime.Unix() < *filters.Since {
				continue
			}
		}

		filtered = append(filtered, rec)

		if filters.Limit > 0 && len(filtered) >= filters.Limit {
			break
		}
	}

	return filtered
}

// ScoutAccuracyStats aggregates accuracy metrics for scout recommendations.
type ScoutAccuracyStats struct {
	TotalRecommendations int     `json:"total_recommendations"`
	FollowedCount        int     `json:"followed_count"`
	IgnoredCount         int     `json:"ignored_count"`
	ComplianceRate       float64 `json:"compliance_rate"` // Followed / Total

	// Accuracy when followed
	FollowedSuccessCount int     `json:"followed_success_count"`
	FollowedFailureCount int     `json:"followed_failure_count"`
	FollowedAccuracy     float64 `json:"followed_accuracy"` // Success / (Success + Failure) when followed

	// Accuracy when ignored
	IgnoredSuccessCount int     `json:"ignored_success_count"`
	IgnoredFailureCount int     `json:"ignored_failure_count"`
	IgnoredAccuracy     float64 `json:"ignored_accuracy"` // Success when NOT following

	// Per-scout-type breakdown
	ByScoutType map[string]*ScoutTypeAccuracy `json:"by_scout_type"`
}

// ScoutTypeAccuracy tracks accuracy for a specific scout type.
type ScoutTypeAccuracy struct {
	ScoutType         string  `json:"scout_type"`
	TotalCount        int     `json:"total_count"`
	FollowedCount     int     `json:"followed_count"`
	FollowedSuccesses int     `json:"followed_successes"`
	Accuracy          float64 `json:"accuracy"`
	AvgConfidence     float64 `json:"avg_confidence"`
}

// CalculateScoutAccuracy computes overall and per-scout accuracy metrics.
// Only considers recommendations with recorded outcomes.
func CalculateScoutAccuracy(recommendations []ScoutRecommendation) ScoutAccuracyStats {
	stats := ScoutAccuracyStats{
		TotalRecommendations: len(recommendations),
		ByScoutType:          make(map[string]*ScoutTypeAccuracy),
	}

	for _, rec := range recommendations {
		// Skip if no outcome recorded
		if rec.TaskOutcome == "" {
			continue
		}

		// Track by scout type
		scoutType := rec.ScoutType
		if scoutType == "" {
			scoutType = "unknown"
		}
		typeStats, exists := stats.ByScoutType[scoutType]
		if !exists {
			typeStats = &ScoutTypeAccuracy{ScoutType: scoutType}
			stats.ByScoutType[scoutType] = typeStats
		}
		typeStats.TotalCount++
		typeStats.AvgConfidence = (typeStats.AvgConfidence*float64(typeStats.TotalCount-1) + rec.Confidence) / float64(typeStats.TotalCount)

		isSuccess := rec.TaskOutcome == "success"

		if rec.Followed {
			stats.FollowedCount++
			typeStats.FollowedCount++

			if isSuccess {
				stats.FollowedSuccessCount++
				typeStats.FollowedSuccesses++
			} else {
				stats.FollowedFailureCount++
			}
		} else {
			stats.IgnoredCount++

			if isSuccess {
				stats.IgnoredSuccessCount++
			} else {
				stats.IgnoredFailureCount++
			}
		}
	}

	// Calculate rates
	if stats.TotalRecommendations > 0 {
		stats.ComplianceRate = float64(stats.FollowedCount) / float64(stats.TotalRecommendations)
	}

	followedWithOutcome := stats.FollowedSuccessCount + stats.FollowedFailureCount
	if followedWithOutcome > 0 {
		stats.FollowedAccuracy = float64(stats.FollowedSuccessCount) / float64(followedWithOutcome)
	}

	ignoredWithOutcome := stats.IgnoredSuccessCount + stats.IgnoredFailureCount
	if ignoredWithOutcome > 0 {
		stats.IgnoredAccuracy = float64(stats.IgnoredSuccessCount) / float64(ignoredWithOutcome)
	}

	// Calculate per-scout accuracy
	for _, typeStats := range stats.ByScoutType {
		if typeStats.FollowedCount > 0 {
			typeStats.Accuracy = float64(typeStats.FollowedSuccesses) / float64(typeStats.FollowedCount)
		}
	}

	return stats
}

// ConfidenceBucket groups recommendations by confidence level.
type ConfidenceBucket struct {
	RangeLow     float64 `json:"range_low"`
	RangeHigh    float64 `json:"range_high"`
	TotalCount   int     `json:"total_count"`
	SuccessCount int     `json:"success_count"`
	SuccessRate  float64 `json:"success_rate"`
	FollowedRate float64 `json:"followed_rate"`
}

// AnalyzeConfidenceCorrelation buckets recommendations by confidence to show
// whether higher confidence correlates with better outcomes.
func AnalyzeConfidenceCorrelation(recommendations []ScoutRecommendation) []ConfidenceBucket {
	// Define confidence buckets: 0.0-0.4, 0.4-0.6, 0.6-0.8, 0.8-1.0
	buckets := []ConfidenceBucket{
		{RangeLow: 0.0, RangeHigh: 0.4},
		{RangeLow: 0.4, RangeHigh: 0.6},
		{RangeLow: 0.6, RangeHigh: 0.8},
		{RangeLow: 0.8, RangeHigh: 1.0},
	}

	followedCounts := make([]int, len(buckets))

	for _, rec := range recommendations {
		if rec.TaskOutcome == "" {
			continue // Skip without outcome
		}

		// Find appropriate bucket
		bucketIndex := -1
		for i := range buckets {
			if rec.Confidence >= buckets[i].RangeLow && rec.Confidence < buckets[i].RangeHigh {
				bucketIndex = i
				break
			}
		}
		// Handle confidence == 1.0 (goes in last bucket)
		if rec.Confidence == 1.0 {
			bucketIndex = len(buckets) - 1
		}

		if bucketIndex < 0 {
			continue
		}

		buckets[bucketIndex].TotalCount++
		if rec.TaskOutcome == "success" {
			buckets[bucketIndex].SuccessCount++
		}
		if rec.Followed {
			followedCounts[bucketIndex]++
		}
	}

	// Calculate rates
	for i := range buckets {
		if buckets[i].TotalCount > 0 {
			buckets[i].SuccessRate = float64(buckets[i].SuccessCount) / float64(buckets[i].TotalCount)
			buckets[i].FollowedRate = float64(followedCounts[i]) / float64(buckets[i].TotalCount)
		}
	}

	return buckets
}

// ComplianceImpact shows the difference in outcomes when following vs ignoring scouts.
type ComplianceImpact struct {
	FollowedSuccessRate     float64 `json:"followed_success_rate"`
	IgnoredSuccessRate      float64 `json:"ignored_success_rate"`
	ImpactDelta             float64 `json:"impact_delta"`             // Followed - Ignored
	Recommendation          string  `json:"recommendation"`           // "follow", "ignore", "neutral"
	StatisticalSignificance string  `json:"statistical_significance"` // "high", "medium", "low", "insufficient_data"
}

// GetComplianceImpact analyzes whether following scout recommendations leads to better outcomes.
func GetComplianceImpact(recommendations []ScoutRecommendation) ComplianceImpact {
	var followedSuccess, followedTotal int
	var ignoredSuccess, ignoredTotal int

	for _, rec := range recommendations {
		if rec.TaskOutcome == "" {
			continue
		}

		isSuccess := rec.TaskOutcome == "success"

		if rec.Followed {
			followedTotal++
			if isSuccess {
				followedSuccess++
			}
		} else {
			ignoredTotal++
			if isSuccess {
				ignoredSuccess++
			}
		}
	}

	impact := ComplianceImpact{}

	// Calculate success rates
	if followedTotal > 0 {
		impact.FollowedSuccessRate = float64(followedSuccess) / float64(followedTotal)
	}
	if ignoredTotal > 0 {
		impact.IgnoredSuccessRate = float64(ignoredSuccess) / float64(ignoredTotal)
	}

	impact.ImpactDelta = impact.FollowedSuccessRate - impact.IgnoredSuccessRate

	// Determine recommendation
	if impact.ImpactDelta > 0.1 {
		impact.Recommendation = "follow"
	} else if impact.ImpactDelta < -0.1 {
		impact.Recommendation = "ignore"
	} else {
		impact.Recommendation = "neutral"
	}

	// Determine statistical significance (simplified)
	totalSamples := followedTotal + ignoredTotal
	if totalSamples < 10 {
		impact.StatisticalSignificance = "insufficient_data"
	} else if totalSamples < 30 {
		impact.StatisticalSignificance = "low"
	} else if totalSamples < 100 {
		impact.StatisticalSignificance = "medium"
	} else {
		impact.StatisticalSignificance = "high"
	}

	return impact
}

// TierRecommendationStats shows how often each tier is recommended.
type TierRecommendationStats struct {
	Tier             string  `json:"tier"`
	RecommendedCount int     `json:"recommended_count"`
	ActualCount      int     `json:"actual_count"`
	ComplianceRate   float64 `json:"compliance_rate"` // When recommended, how often used
	AvgConfidence    float64 `json:"avg_confidence"`
}

// ClusterRecommendationsByTier groups recommendations by the tier that was recommended.
func ClusterRecommendationsByTier(recommendations []ScoutRecommendation) map[string]*TierRecommendationStats {
	clusters := make(map[string]*TierRecommendationStats)
	actualCounts := make(map[string]int)

	var confidenceSums = make(map[string]float64)

	for _, rec := range recommendations {
		tier := rec.RecommendedTier
		if tier == "" {
			tier = "unknown"
		}

		stats, exists := clusters[tier]
		if !exists {
			stats = &TierRecommendationStats{Tier: tier}
			clusters[tier] = stats
		}

		stats.RecommendedCount++
		confidenceSums[tier] += rec.Confidence

		if rec.Followed {
			actualCounts[tier]++
		}
	}

	// Calculate compliance rates and averages
	for tier, stats := range clusters {
		stats.ActualCount = actualCounts[tier]
		if stats.RecommendedCount > 0 {
			stats.ComplianceRate = float64(stats.ActualCount) / float64(stats.RecommendedCount)
			stats.AvgConfidence = confidenceSums[tier] / float64(stats.RecommendedCount)
		}
	}

	return clusters
}

// ScoutPerformanceSummary provides a complete overview of scout effectiveness.
type ScoutPerformanceSummary struct {
	AccuracyStats     ScoutAccuracyStats                  `json:"accuracy_stats"`
	ConfidenceBuckets []ConfidenceBucket                  `json:"confidence_buckets"`
	ComplianceImpact  ComplianceImpact                    `json:"compliance_impact"`
	TierDistribution  map[string]*TierRecommendationStats `json:"tier_distribution"`
	OverallVerdict    string                              `json:"overall_verdict"`
}

// GetScoutPerformanceSummary generates a complete scout effectiveness report.
func GetScoutPerformanceSummary(recommendations []ScoutRecommendation) ScoutPerformanceSummary {
	summary := ScoutPerformanceSummary{
		AccuracyStats:     CalculateScoutAccuracy(recommendations),
		ConfidenceBuckets: AnalyzeConfidenceCorrelation(recommendations),
		ComplianceImpact:  GetComplianceImpact(recommendations),
		TierDistribution:  ClusterRecommendationsByTier(recommendations),
	}

	// Generate overall verdict
	if summary.ComplianceImpact.StatisticalSignificance == "insufficient_data" {
		summary.OverallVerdict = "Insufficient data to evaluate scout performance"
	} else if summary.AccuracyStats.FollowedAccuracy > 0.8 && summary.ComplianceImpact.Recommendation == "follow" {
		summary.OverallVerdict = "Scouts performing well - follow recommendations"
	} else if summary.AccuracyStats.FollowedAccuracy < 0.5 {
		summary.OverallVerdict = "Scout accuracy low - review routing configuration"
	} else {
		summary.OverallVerdict = "Scout performance moderate - consider case-by-case evaluation"
	}

	return summary
}
