package telemetry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLogScoutRecommendation_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Unsetenv("XDG_CACHE_HOME")

	rec := &ScoutRecommendation{
		SessionID:        "test-session",
		RecommendationID: "rec-001",
		ScoutType:        "haiku-scout",
		TaskDescription:  "Refactor authentication module",
		RecommendedTier:  "sonnet",
		ActualTier:       "sonnet",
		Confidence:       0.85,
		ScopeMetrics: ScopeMetrics{
			TotalFiles:      5,
			TotalLines:      1200,
			EstimatedTokens: 15000,
		},
	}

	id, err := LogScoutRecommendation(rec, "")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if id != "rec-001" {
		t.Errorf("Expected ID 'rec-001', got: %s", id)
	}

	// Verify followed was auto-determined
	if !rec.Followed {
		t.Error("Expected Followed to be true (tiers match)")
	}

	// Verify timestamp was populated
	if rec.Timestamp == "" {
		t.Error("Expected timestamp to be populated")
	}

	// Verify file created
	path := GetScoutLogPath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	if !strings.Contains(string(data), "haiku-scout") {
		t.Error("Expected log to contain scout type")
	}
	if !strings.Contains(string(data), "rec-001") {
		t.Error("Expected log to contain recommendation ID")
	}
}

func TestLogScoutRecommendation_NotFollowed(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Unsetenv("XDG_CACHE_HOME")

	rec := &ScoutRecommendation{
		RecommendationID: "rec-002",
		ScoutType:        "haiku-scout",
		RecommendedTier:  "haiku",
		ActualTier:       "sonnet", // Different!
		FollowedReason:   "User override for complex task",
		Confidence:       0.60,
	}

	_, err := LogScoutRecommendation(rec, "")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if rec.Followed {
		t.Error("Expected Followed to be false (tiers differ)")
	}
}

func TestLogScoutRecommendation_InvalidConfidence(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Unsetenv("XDG_CACHE_HOME")

	testCases := []struct {
		name       string
		confidence float64
	}{
		{"confidence too high", 1.5},
		{"confidence negative", -0.1},
		{"confidence way too high", 10.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rec := &ScoutRecommendation{
				RecommendationID: "rec-bad",
				Confidence:       tc.confidence,
			}

			_, err := LogScoutRecommendation(rec, "")
			if err == nil {
				t.Errorf("Expected error for confidence %f", tc.confidence)
			}
			if !strings.Contains(err.Error(), "Invalid confidence") {
				t.Errorf("Expected 'Invalid confidence' error, got: %v", err)
			}
		})
	}
}

func TestLogScoutRecommendation_ValidConfidenceBoundaries(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Unsetenv("XDG_CACHE_HOME")

	testCases := []struct {
		name       string
		confidence float64
	}{
		{"min confidence", 0.0},
		{"max confidence", 1.0},
		{"mid confidence", 0.5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rec := &ScoutRecommendation{
				RecommendationID: "rec-valid-" + tc.name,
				Confidence:       tc.confidence,
			}

			_, err := LogScoutRecommendation(rec, "")
			if err != nil {
				t.Errorf("Expected no error for valid confidence %f, got: %v", tc.confidence, err)
			}
		})
	}
}

func TestLogScoutRecommendation_InvalidScoutType(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Unsetenv("XDG_CACHE_HOME")

	rec := &ScoutRecommendation{
		RecommendationID: "rec-bad",
		ScoutType:        "invalid-scout",
		Confidence:       0.5,
	}

	_, err := LogScoutRecommendation(rec, "")
	if err == nil {
		t.Error("Expected error for invalid scout type")
	}
	if !strings.Contains(err.Error(), "Invalid scout type") {
		t.Errorf("Expected 'Invalid scout type' error, got: %v", err)
	}
}

func TestLogScoutRecommendation_ValidScoutTypes(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Unsetenv("XDG_CACHE_HOME")

	validTypes := []string{"haiku-scout", "gemini-scout"}

	for _, scoutType := range validTypes {
		t.Run(scoutType, func(t *testing.T) {
			rec := &ScoutRecommendation{
				RecommendationID: "rec-" + scoutType,
				ScoutType:        scoutType,
				Confidence:       0.5,
			}

			_, err := LogScoutRecommendation(rec, "")
			if err != nil {
				t.Errorf("Expected no error for valid scout type %s, got: %v", scoutType, err)
			}
		})
	}
}

func TestLogScoutRecommendation_EmptyScoutType(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Unsetenv("XDG_CACHE_HOME")

	rec := &ScoutRecommendation{
		RecommendationID: "rec-empty",
		ScoutType:        "", // Empty should be allowed
		Confidence:       0.5,
	}

	_, err := LogScoutRecommendation(rec, "")
	if err != nil {
		t.Errorf("Expected no error for empty scout type, got: %v", err)
	}
}

func TestLogScoutRecommendation_ProjectDir(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Unsetenv("XDG_CACHE_HOME")

	projectDir := filepath.Join(tmpDir, "test-project")

	rec := &ScoutRecommendation{
		RecommendationID: "rec-003",
		ScoutType:        "haiku-scout",
		Confidence:       0.7,
	}

	_, err := LogScoutRecommendation(rec, projectDir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify project directory was populated
	if rec.ProjectDir != projectDir {
		t.Errorf("Expected ProjectDir to be %s, got: %s", projectDir, rec.ProjectDir)
	}

	// Verify both global and project logs were created
	globalPath := GetScoutLogPath()
	if _, err := os.Stat(globalPath); os.IsNotExist(err) {
		t.Error("Expected global log to exist")
	}

	projectPath := GetProjectScoutLogPath(projectDir)
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Error("Expected project log to exist")
	}
}

func TestLoadScoutRecommendations_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "scout.jsonl")

	content := `{"timestamp":"2026-01-22T10:00:00Z","recommendation_id":"r1","scout_type":"haiku-scout","recommended_tier":"sonnet","followed":true,"confidence":0.85}
{"timestamp":"2026-01-22T11:00:00Z","recommendation_id":"r2","scout_type":"gemini-scout","recommended_tier":"haiku","followed":false,"confidence":0.6}`

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	recs, err := LoadScoutRecommendations(path)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(recs) != 2 {
		t.Errorf("Expected 2 recommendations, got: %d", len(recs))
	}

	if recs[0].RecommendationID != "r1" {
		t.Errorf("Expected first rec ID 'r1', got: %s", recs[0].RecommendationID)
	}
	if recs[1].RecommendationID != "r2" {
		t.Errorf("Expected second rec ID 'r2', got: %s", recs[1].RecommendationID)
	}
}

func TestLoadScoutRecommendations_MissingFile(t *testing.T) {
	recs, err := LoadScoutRecommendations("/nonexistent/path/scout.jsonl")
	if err != nil {
		t.Errorf("Expected no error for missing file, got: %v", err)
	}
	if len(recs) != 0 {
		t.Errorf("Expected empty slice, got: %d recommendations", len(recs))
	}
}

func TestLoadScoutRecommendations_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "empty.jsonl")

	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	recs, err := LoadScoutRecommendations(path)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(recs) != 0 {
		t.Errorf("Expected empty slice for empty file, got: %d", len(recs))
	}
}

func TestLoadScoutRecommendations_MalformedLines(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "malformed.jsonl")

	content := `{"recommendation_id":"r1","confidence":0.5}
invalid json line
{"recommendation_id":"r2","confidence":0.7}
{ incomplete
{"recommendation_id":"r3","confidence":0.9}`

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	recs, err := LoadScoutRecommendations(path)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should skip malformed lines and return only valid ones
	if len(recs) != 3 {
		t.Errorf("Expected 3 valid recommendations, got: %d", len(recs))
	}
}

func TestLoadScoutRecommendations_BlankLines(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "blanks.jsonl")

	content := `{"recommendation_id":"r1","confidence":0.5}

{"recommendation_id":"r2","confidence":0.7}

{"recommendation_id":"r3","confidence":0.9}`

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	recs, err := LoadScoutRecommendations(path)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(recs) != 3 {
		t.Errorf("Expected 3 recommendations (blank lines skipped), got: %d", len(recs))
	}
}

func TestUpdateScoutOutcome_Success(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "scout.jsonl")

	content := `{"recommendation_id":"r1","followed":true,"task_outcome":"","confidence":0.5}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	err := UpdateScoutOutcome(path, "r1", "success", "Completed without issues")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Reload and verify
	recs, err := LoadScoutRecommendations(path)
	if err != nil {
		t.Fatalf("Failed to reload: %v", err)
	}

	if len(recs) != 1 {
		t.Fatalf("Expected 1 recommendation, got: %d", len(recs))
	}

	if recs[0].TaskOutcome != "success" {
		t.Errorf("Expected outcome 'success', got: %s", recs[0].TaskOutcome)
	}
	if recs[0].OutcomeNotes != "Completed without issues" {
		t.Errorf("Expected notes 'Completed without issues', got: %s", recs[0].OutcomeNotes)
	}
}

func TestUpdateScoutOutcome_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "scout.jsonl")

	content := `{"recommendation_id":"r1","followed":true,"confidence":0.5}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	err := UpdateScoutOutcome(path, "nonexistent", "success", "")
	if err == nil {
		t.Error("Expected error for nonexistent recommendation ID")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestUpdateScoutOutcome_InvalidOutcome(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "scout.jsonl")

	content := `{"recommendation_id":"r1","followed":true,"confidence":0.5}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	err := UpdateScoutOutcome(path, "r1", "invalid-outcome", "")
	if err == nil {
		t.Error("Expected error for invalid outcome")
	}
	if !strings.Contains(err.Error(), "Invalid outcome") {
		t.Errorf("Expected 'Invalid outcome' error, got: %v", err)
	}
}

func TestUpdateScoutOutcome_ValidOutcomes(t *testing.T) {
	validOutcomes := []string{"success", "failure", "escalated", ""}

	for _, outcome := range validOutcomes {
		t.Run("outcome_"+outcome, func(t *testing.T) {
			tmpDir := t.TempDir()
			path := filepath.Join(tmpDir, "scout.jsonl")

			content := `{"recommendation_id":"r1","followed":true,"confidence":0.5}`
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			err := UpdateScoutOutcome(path, "r1", outcome, "Test notes")
			if err != nil {
				t.Errorf("Expected no error for valid outcome '%s', got: %v", outcome, err)
			}
		})
	}
}

func TestUpdateScoutOutcome_MultipleRecommendations(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "scout.jsonl")

	content := `{"recommendation_id":"r1","followed":true,"confidence":0.5}
{"recommendation_id":"r2","followed":false,"confidence":0.7}
{"recommendation_id":"r3","followed":true,"confidence":0.9}`

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Update middle recommendation
	err := UpdateScoutOutcome(path, "r2", "failure", "Failed due to missing dependency")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Reload and verify only r2 was updated
	recs, err := LoadScoutRecommendations(path)
	if err != nil {
		t.Fatalf("Failed to reload: %v", err)
	}

	if len(recs) != 3 {
		t.Fatalf("Expected 3 recommendations, got: %d", len(recs))
	}

	// r1 unchanged
	if recs[0].TaskOutcome != "" {
		t.Errorf("Expected r1 outcome to be empty, got: %s", recs[0].TaskOutcome)
	}

	// r2 updated
	if recs[1].TaskOutcome != "failure" {
		t.Errorf("Expected r2 outcome 'failure', got: %s", recs[1].TaskOutcome)
	}
	if recs[1].OutcomeNotes != "Failed due to missing dependency" {
		t.Errorf("Expected r2 notes to be updated, got: %s", recs[1].OutcomeNotes)
	}

	// r3 unchanged
	if recs[2].TaskOutcome != "" {
		t.Errorf("Expected r3 outcome to be empty, got: %s", recs[2].TaskOutcome)
	}
}

func TestFilterScoutRecommendations_ByFollowed(t *testing.T) {
	recs := []ScoutRecommendation{
		{RecommendationID: "r1", Followed: true, Confidence: 0.5},
		{RecommendationID: "r2", Followed: false, Confidence: 0.5},
		{RecommendationID: "r3", Followed: true, Confidence: 0.5},
	}

	followed := true
	filtered := FilterScoutRecommendations(recs, ScoutFilters{Followed: &followed})

	if len(filtered) != 2 {
		t.Errorf("Expected 2 followed recommendations, got: %d", len(filtered))
	}

	for _, rec := range filtered {
		if !rec.Followed {
			t.Errorf("Filtered result contains unfollowed recommendation: %s", rec.RecommendationID)
		}
	}
}

func TestFilterScoutRecommendations_ByMinConfidence(t *testing.T) {
	recs := []ScoutRecommendation{
		{RecommendationID: "r1", Confidence: 0.3},
		{RecommendationID: "r2", Confidence: 0.7},
		{RecommendationID: "r3", Confidence: 0.9},
	}

	minConf := 0.6
	filtered := FilterScoutRecommendations(recs, ScoutFilters{MinConfidence: &minConf})

	if len(filtered) != 2 {
		t.Errorf("Expected 2 high-confidence recommendations, got: %d", len(filtered))
	}

	for _, rec := range filtered {
		if rec.Confidence < minConf {
			t.Errorf("Filtered result contains low-confidence recommendation: %s (conf: %f)", rec.RecommendationID, rec.Confidence)
		}
	}
}

func TestFilterScoutRecommendations_ByScoutType(t *testing.T) {
	recs := []ScoutRecommendation{
		{RecommendationID: "r1", ScoutType: "haiku-scout", Confidence: 0.5},
		{RecommendationID: "r2", ScoutType: "gemini-scout", Confidence: 0.5},
		{RecommendationID: "r3", ScoutType: "haiku-scout", Confidence: 0.5},
	}

	scoutType := "haiku-scout"
	filtered := FilterScoutRecommendations(recs, ScoutFilters{ScoutType: &scoutType})

	if len(filtered) != 2 {
		t.Errorf("Expected 2 haiku-scout recommendations, got: %d", len(filtered))
	}

	for _, rec := range filtered {
		if rec.ScoutType != scoutType {
			t.Errorf("Filtered result contains wrong scout type: %s", rec.ScoutType)
		}
	}
}

func TestFilterScoutRecommendations_ByRecommendedTier(t *testing.T) {
	recs := []ScoutRecommendation{
		{RecommendationID: "r1", RecommendedTier: "haiku", Confidence: 0.5},
		{RecommendationID: "r2", RecommendedTier: "sonnet", Confidence: 0.5},
		{RecommendationID: "r3", RecommendedTier: "sonnet", Confidence: 0.5},
	}

	tier := "sonnet"
	filtered := FilterScoutRecommendations(recs, ScoutFilters{RecommendedTier: &tier})

	if len(filtered) != 2 {
		t.Errorf("Expected 2 sonnet recommendations, got: %d", len(filtered))
	}

	for _, rec := range filtered {
		if rec.RecommendedTier != tier {
			t.Errorf("Filtered result contains wrong tier: %s", rec.RecommendedTier)
		}
	}
}

func TestFilterScoutRecommendations_ByTaskOutcome(t *testing.T) {
	recs := []ScoutRecommendation{
		{RecommendationID: "r1", TaskOutcome: "success", Confidence: 0.5},
		{RecommendationID: "r2", TaskOutcome: "failure", Confidence: 0.5},
		{RecommendationID: "r3", TaskOutcome: "success", Confidence: 0.5},
	}

	outcome := "success"
	filtered := FilterScoutRecommendations(recs, ScoutFilters{TaskOutcome: &outcome})

	if len(filtered) != 2 {
		t.Errorf("Expected 2 success outcomes, got: %d", len(filtered))
	}

	for _, rec := range filtered {
		if rec.TaskOutcome != outcome {
			t.Errorf("Filtered result contains wrong outcome: %s", rec.TaskOutcome)
		}
	}
}

func TestFilterScoutRecommendations_BySince(t *testing.T) {
	// Create recommendations with different timestamps
	baseTime := time.Now()
	old := baseTime.Add(-2 * time.Hour).Format(time.RFC3339)
	recent := baseTime.Add(-30 * time.Minute).Format(time.RFC3339)

	recs := []ScoutRecommendation{
		{RecommendationID: "r1", Timestamp: old, Confidence: 0.5},
		{RecommendationID: "r2", Timestamp: recent, Confidence: 0.5},
		{RecommendationID: "r3", Timestamp: recent, Confidence: 0.5},
	}

	// Filter for last hour
	sinceUnix := baseTime.Add(-1 * time.Hour).Unix()
	filtered := FilterScoutRecommendations(recs, ScoutFilters{Since: &sinceUnix})

	if len(filtered) != 2 {
		t.Errorf("Expected 2 recent recommendations, got: %d", len(filtered))
	}
}

func TestFilterScoutRecommendations_WithLimit(t *testing.T) {
	recs := []ScoutRecommendation{
		{RecommendationID: "r1", Confidence: 0.5},
		{RecommendationID: "r2", Confidence: 0.5},
		{RecommendationID: "r3", Confidence: 0.5},
		{RecommendationID: "r4", Confidence: 0.5},
		{RecommendationID: "r5", Confidence: 0.5},
	}

	filtered := FilterScoutRecommendations(recs, ScoutFilters{Limit: 3})

	if len(filtered) != 3 {
		t.Errorf("Expected limit of 3, got: %d", len(filtered))
	}
}

func TestFilterScoutRecommendations_MultipleFilters(t *testing.T) {
	recs := []ScoutRecommendation{
		{RecommendationID: "r1", ScoutType: "haiku-scout", Followed: true, Confidence: 0.8},
		{RecommendationID: "r2", ScoutType: "haiku-scout", Followed: false, Confidence: 0.9},
		{RecommendationID: "r3", ScoutType: "gemini-scout", Followed: true, Confidence: 0.85},
		{RecommendationID: "r4", ScoutType: "haiku-scout", Followed: true, Confidence: 0.5},
	}

	scoutType := "haiku-scout"
	followed := true
	minConf := 0.7

	filtered := FilterScoutRecommendations(recs, ScoutFilters{
		ScoutType:     &scoutType,
		Followed:      &followed,
		MinConfidence: &minConf,
	})

	// Only r1 matches all filters
	if len(filtered) != 1 {
		t.Errorf("Expected 1 recommendation matching all filters, got: %d", len(filtered))
	}
	if len(filtered) > 0 && filtered[0].RecommendationID != "r1" {
		t.Errorf("Expected r1, got: %s", filtered[0].RecommendationID)
	}
}

func TestFilterScoutRecommendations_NoFilters(t *testing.T) {
	recs := []ScoutRecommendation{
		{RecommendationID: "r1", Confidence: 0.5},
		{RecommendationID: "r2", Confidence: 0.7},
		{RecommendationID: "r3", Confidence: 0.9},
	}

	filtered := FilterScoutRecommendations(recs, ScoutFilters{})

	if len(filtered) != len(recs) {
		t.Errorf("Expected all recommendations with no filters, got: %d", len(filtered))
	}
}

func TestFilterScoutRecommendations_EmptyInput(t *testing.T) {
	recs := []ScoutRecommendation{}

	minConf := 0.5
	filtered := FilterScoutRecommendations(recs, ScoutFilters{MinConfidence: &minConf})

	if len(filtered) != 0 {
		t.Errorf("Expected empty result for empty input, got: %d", len(filtered))
	}
}

func TestGetScoutLogPath(t *testing.T) {
	tmpDir := t.TempDir()
	// Unset XDG_RUNTIME_DIR first (it takes priority over XDG_CACHE_HOME in GetGOgentDir)
	oldRuntime := os.Getenv("XDG_RUNTIME_DIR")
	os.Unsetenv("XDG_RUNTIME_DIR")
	defer func() {
		if oldRuntime != "" {
			os.Setenv("XDG_RUNTIME_DIR", oldRuntime)
		}
	}()

	os.Setenv("XDG_CACHE_HOME", tmpDir)
	defer os.Unsetenv("XDG_CACHE_HOME")

	path := GetScoutLogPath()

	if !strings.Contains(path, "scout-recommendations.jsonl") {
		t.Errorf("Expected path to contain 'scout-recommendations.jsonl', got: %s", path)
	}

	if !strings.HasPrefix(path, tmpDir) {
		t.Errorf("Expected path to start with XDG_CACHE_HOME, got: %s", path)
	}
}

func TestGetProjectScoutLogPath(t *testing.T) {
	projectDir := "/home/user/my-project"
	path := GetProjectScoutLogPath(projectDir)

	expected := filepath.Join(projectDir, ".claude", "memory", "scout-recommendations.jsonl")
	if path != expected {
		t.Errorf("Expected path %s, got: %s", expected, path)
	}
}

func TestScoutRecommendation_FollowedAutoDetection(t *testing.T) {
	testCases := []struct {
		name            string
		recommendedTier string
		actualTier      string
		expectedFollow  bool
	}{
		{"matching tiers", "sonnet", "sonnet", true},
		{"different tiers", "haiku", "sonnet", false},
		{"recommended empty", "", "sonnet", false},
		{"actual empty", "sonnet", "", false},
		{"both empty", "", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			os.Setenv("XDG_CACHE_HOME", tmpDir)
			defer os.Unsetenv("XDG_CACHE_HOME")

			rec := &ScoutRecommendation{
				RecommendationID: "test-" + tc.name,
				RecommendedTier:  tc.recommendedTier,
				ActualTier:       tc.actualTier,
				Confidence:       0.5,
			}

			_, err := LogScoutRecommendation(rec, "")
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if rec.Followed != tc.expectedFollow {
				t.Errorf("Expected Followed=%v, got: %v", tc.expectedFollow, rec.Followed)
			}
		})
	}
}
