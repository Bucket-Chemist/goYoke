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
	Timestamp        string `json:"timestamp"`         // RFC3339 format
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
	baseDir := config.GetGOgentDir()
	return filepath.Join(baseDir, "scout-recommendations.jsonl")
}

// GetProjectScoutLogPath returns project-scoped scout log path.
func GetProjectScoutLogPath(projectDir string) string {
	return filepath.Join(projectDir, ".claude", "memory", "scout-recommendations.jsonl")
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
	scanner := bufio.NewScanner(file)

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
