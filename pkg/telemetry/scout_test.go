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

func TestLoadScoutRecommendations_LargeLine(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "scout.jsonl")
	largeTask := strings.Repeat("z", 70*1024)
	content := `{"timestamp":"2026-01-22T10:00:00Z","recommendation_id":"r1","scout_type":"haiku-scout","task_description":"` + largeTask + `","recommended_tier":"sonnet","followed":true,"confidence":0.85}`

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	recs, err := LoadScoutRecommendations(path)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("Expected 1 recommendation, got: %d", len(recs))
	}
	if recs[0].TaskDescription != largeTask {
		t.Fatalf("Expected large task description to round-trip")
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
	// Unset XDG_RUNTIME_DIR first (it takes priority over XDG_CACHE_HOME in GetgoYokeDir)
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

	expected := filepath.Join(projectDir, ".goyoke", "memory", "scout-recommendations.jsonl")
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

// Tests for scout accuracy calculation functions

func TestCalculateScoutAccuracy_Basic(t *testing.T) {
	recs := []ScoutRecommendation{
		{Followed: true, TaskOutcome: "success", ScoutType: "haiku-scout", Confidence: 0.9},
		{Followed: true, TaskOutcome: "success", ScoutType: "haiku-scout", Confidence: 0.8},
		{Followed: true, TaskOutcome: "failure", ScoutType: "haiku-scout", Confidence: 0.7},
		{Followed: false, TaskOutcome: "success", ScoutType: "gemini-scout", Confidence: 0.6},
	}

	stats := CalculateScoutAccuracy(recs)

	if stats.TotalRecommendations != 4 {
		t.Errorf("Expected 4 total, got: %d", stats.TotalRecommendations)
	}
	if stats.FollowedCount != 3 {
		t.Errorf("Expected 3 followed, got: %d", stats.FollowedCount)
	}
	// Followed accuracy: 2 success / 3 followed = 0.667
	if stats.FollowedAccuracy < 0.66 || stats.FollowedAccuracy > 0.68 {
		t.Errorf("Expected followed accuracy ~0.67, got: %f", stats.FollowedAccuracy)
	}
}

func TestCalculateScoutAccuracy_NoOutcomes(t *testing.T) {
	recs := []ScoutRecommendation{
		{Followed: true, TaskOutcome: "", Confidence: 0.5},  // No outcome
		{Followed: false, TaskOutcome: "", Confidence: 0.5}, // No outcome
	}

	stats := CalculateScoutAccuracy(recs)

	if stats.FollowedCount != 0 {
		t.Errorf("Expected 0 followed (no outcomes), got: %d", stats.FollowedCount)
	}
}

func TestCalculateScoutAccuracy_EmptyInput(t *testing.T) {
	recs := []ScoutRecommendation{}

	stats := CalculateScoutAccuracy(recs)

	if stats.TotalRecommendations != 0 {
		t.Errorf("Expected 0 total recommendations, got: %d", stats.TotalRecommendations)
	}
	if stats.FollowedAccuracy != 0.0 {
		t.Errorf("Expected 0.0 followed accuracy, got: %f", stats.FollowedAccuracy)
	}
}

func TestCalculateScoutAccuracy_ByScoutType(t *testing.T) {
	recs := []ScoutRecommendation{
		{Followed: true, TaskOutcome: "success", ScoutType: "haiku-scout", Confidence: 0.9},
		{Followed: true, TaskOutcome: "success", ScoutType: "haiku-scout", Confidence: 0.8},
		{Followed: true, TaskOutcome: "failure", ScoutType: "gemini-scout", Confidence: 0.7},
		{Followed: true, TaskOutcome: "success", ScoutType: "gemini-scout", Confidence: 0.6},
	}

	stats := CalculateScoutAccuracy(recs)

	if len(stats.ByScoutType) != 2 {
		t.Errorf("Expected 2 scout types, got: %d", len(stats.ByScoutType))
	}

	haikuStats := stats.ByScoutType["haiku-scout"]
	if haikuStats == nil {
		t.Fatal("Expected haiku-scout stats")
	}
	if haikuStats.TotalCount != 2 {
		t.Errorf("Expected haiku-scout total 2, got: %d", haikuStats.TotalCount)
	}
	if haikuStats.Accuracy != 1.0 {
		t.Errorf("Expected haiku-scout accuracy 1.0, got: %f", haikuStats.Accuracy)
	}

	geminiStats := stats.ByScoutType["gemini-scout"]
	if geminiStats == nil {
		t.Fatal("Expected gemini-scout stats")
	}
	if geminiStats.TotalCount != 2 {
		t.Errorf("Expected gemini-scout total 2, got: %d", geminiStats.TotalCount)
	}
	if geminiStats.Accuracy != 0.5 {
		t.Errorf("Expected gemini-scout accuracy 0.5, got: %f", geminiStats.Accuracy)
	}
}

func TestCalculateScoutAccuracy_UnknownScoutType(t *testing.T) {
	recs := []ScoutRecommendation{
		{Followed: true, TaskOutcome: "success", ScoutType: "", Confidence: 0.9},
		{Followed: true, TaskOutcome: "failure", ScoutType: "", Confidence: 0.8},
	}

	stats := CalculateScoutAccuracy(recs)

	unknownStats := stats.ByScoutType["unknown"]
	if unknownStats == nil {
		t.Fatal("Expected 'unknown' scout type for empty ScoutType")
	}
	if unknownStats.TotalCount != 2 {
		t.Errorf("Expected 2 unknown scouts, got: %d", unknownStats.TotalCount)
	}
}

func TestCalculateScoutAccuracy_IgnoredAccuracy(t *testing.T) {
	recs := []ScoutRecommendation{
		{Followed: false, TaskOutcome: "success", ScoutType: "haiku-scout", Confidence: 0.9},
		{Followed: false, TaskOutcome: "success", ScoutType: "haiku-scout", Confidence: 0.8},
		{Followed: false, TaskOutcome: "failure", ScoutType: "haiku-scout", Confidence: 0.7},
	}

	stats := CalculateScoutAccuracy(recs)

	if stats.IgnoredCount != 3 {
		t.Errorf("Expected 3 ignored, got: %d", stats.IgnoredCount)
	}
	// Ignored accuracy: 2 success / 3 ignored = 0.667
	if stats.IgnoredAccuracy < 0.66 || stats.IgnoredAccuracy > 0.68 {
		t.Errorf("Expected ignored accuracy ~0.67, got: %f", stats.IgnoredAccuracy)
	}
}

func TestCalculateScoutAccuracy_ComplianceRate(t *testing.T) {
	recs := []ScoutRecommendation{
		{Followed: true, TaskOutcome: "success", Confidence: 0.9},
		{Followed: true, TaskOutcome: "failure", Confidence: 0.8},
		{Followed: false, TaskOutcome: "success", Confidence: 0.7},
		{Followed: false, TaskOutcome: "failure", Confidence: 0.6},
	}

	stats := CalculateScoutAccuracy(recs)

	// Compliance rate: 2 followed / 4 total = 0.5
	if stats.ComplianceRate != 0.5 {
		t.Errorf("Expected compliance rate 0.5, got: %f", stats.ComplianceRate)
	}
}

func TestCalculateScoutAccuracy_DivisionByZero(t *testing.T) {
	// Test zero followed recommendations
	recs := []ScoutRecommendation{
		{Followed: false, TaskOutcome: "success", Confidence: 0.9},
	}

	stats := CalculateScoutAccuracy(recs)

	if stats.FollowedAccuracy != 0.0 {
		t.Errorf("Expected 0.0 followed accuracy for zero followed, got: %f", stats.FollowedAccuracy)
	}
}

func TestAnalyzeConfidenceCorrelation_Basic(t *testing.T) {
	recs := []ScoutRecommendation{
		{Confidence: 0.3, TaskOutcome: "failure", Followed: false}, // Low confidence, failure
		{Confidence: 0.5, TaskOutcome: "success", Followed: true},  // Medium
		{Confidence: 0.7, TaskOutcome: "success", Followed: true},  // High
		{Confidence: 0.9, TaskOutcome: "success", Followed: true},  // Very high
	}

	buckets := AnalyzeConfidenceCorrelation(recs)

	// Should have 4 buckets
	if len(buckets) != 4 {
		t.Errorf("Expected 4 buckets, got: %d", len(buckets))
	}

	// First bucket (0.0-0.4) should have 1 failure
	if buckets[0].TotalCount != 1 || buckets[0].SuccessCount != 0 {
		t.Errorf("Expected bucket 0 to have 1 failure, got: total=%d success=%d", buckets[0].TotalCount, buckets[0].SuccessCount)
	}

	// Last bucket (0.8-1.0) should have 1 success
	if buckets[3].TotalCount != 1 || buckets[3].SuccessRate != 1.0 {
		t.Errorf("Expected bucket 3 to have 100%% success rate, got: %f", buckets[3].SuccessRate)
	}
}

func TestAnalyzeConfidenceCorrelation_ConfidenceExactlyOne(t *testing.T) {
	recs := []ScoutRecommendation{
		{Confidence: 1.0, TaskOutcome: "success", Followed: true},
	}

	buckets := AnalyzeConfidenceCorrelation(recs)

	// Confidence 1.0 should go in last bucket (0.8-1.0)
	if buckets[3].TotalCount != 1 {
		t.Errorf("Expected confidence 1.0 in last bucket, got count: %d", buckets[3].TotalCount)
	}
}

func TestAnalyzeConfidenceCorrelation_EmptyInput(t *testing.T) {
	recs := []ScoutRecommendation{}

	buckets := AnalyzeConfidenceCorrelation(recs)

	if len(buckets) != 4 {
		t.Errorf("Expected 4 buckets even for empty input, got: %d", len(buckets))
	}

	for i, bucket := range buckets {
		if bucket.TotalCount != 0 {
			t.Errorf("Expected bucket %d to be empty, got count: %d", i, bucket.TotalCount)
		}
	}
}

func TestAnalyzeConfidenceCorrelation_NoOutcomes(t *testing.T) {
	recs := []ScoutRecommendation{
		{Confidence: 0.5, TaskOutcome: "", Followed: true},
		{Confidence: 0.7, TaskOutcome: "", Followed: true},
	}

	buckets := AnalyzeConfidenceCorrelation(recs)

	// All buckets should be empty since outcomes are missing
	for i, bucket := range buckets {
		if bucket.TotalCount != 0 {
			t.Errorf("Expected bucket %d to be empty (no outcomes), got count: %d", i, bucket.TotalCount)
		}
	}
}

func TestAnalyzeConfidenceCorrelation_FollowedRate(t *testing.T) {
	recs := []ScoutRecommendation{
		{Confidence: 0.5, TaskOutcome: "success", Followed: true},
		{Confidence: 0.55, TaskOutcome: "success", Followed: true},
		{Confidence: 0.52, TaskOutcome: "failure", Followed: false},
		{Confidence: 0.58, TaskOutcome: "success", Followed: false},
	}

	buckets := AnalyzeConfidenceCorrelation(recs)

	// All should be in bucket 1 (0.4-0.6)
	if buckets[1].TotalCount != 4 {
		t.Errorf("Expected 4 in bucket 1, got: %d", buckets[1].TotalCount)
	}
	// Followed rate: 2/4 = 0.5
	if buckets[1].FollowedRate != 0.5 {
		t.Errorf("Expected followed rate 0.5, got: %f", buckets[1].FollowedRate)
	}
}

func TestGetComplianceImpact_FollowBetter(t *testing.T) {
	recs := []ScoutRecommendation{
		{Followed: true, TaskOutcome: "success", Confidence: 0.5},
		{Followed: true, TaskOutcome: "success", Confidence: 0.5},
		{Followed: true, TaskOutcome: "success", Confidence: 0.5},
		{Followed: false, TaskOutcome: "failure", Confidence: 0.5},
		{Followed: false, TaskOutcome: "failure", Confidence: 0.5},
	}

	impact := GetComplianceImpact(recs)

	if impact.FollowedSuccessRate != 1.0 {
		t.Errorf("Expected followed success rate 1.0, got: %f", impact.FollowedSuccessRate)
	}
	if impact.IgnoredSuccessRate != 0.0 {
		t.Errorf("Expected ignored success rate 0.0, got: %f", impact.IgnoredSuccessRate)
	}
	if impact.Recommendation != "follow" {
		t.Errorf("Expected recommendation 'follow', got: %s", impact.Recommendation)
	}
}

func TestGetComplianceImpact_IgnoreBetter(t *testing.T) {
	recs := []ScoutRecommendation{
		{Followed: true, TaskOutcome: "failure", Confidence: 0.5},
		{Followed: true, TaskOutcome: "failure", Confidence: 0.5},
		{Followed: false, TaskOutcome: "success", Confidence: 0.5},
		{Followed: false, TaskOutcome: "success", Confidence: 0.5},
		{Followed: false, TaskOutcome: "success", Confidence: 0.5},
	}

	impact := GetComplianceImpact(recs)

	if impact.FollowedSuccessRate != 0.0 {
		t.Errorf("Expected followed success rate 0.0, got: %f", impact.FollowedSuccessRate)
	}
	if impact.IgnoredSuccessRate != 1.0 {
		t.Errorf("Expected ignored success rate 1.0, got: %f", impact.IgnoredSuccessRate)
	}
	if impact.Recommendation != "ignore" {
		t.Errorf("Expected recommendation 'ignore', got: %s", impact.Recommendation)
	}
}

func TestGetComplianceImpact_Neutral(t *testing.T) {
	recs := []ScoutRecommendation{
		{Followed: true, TaskOutcome: "success", Confidence: 0.5},
		{Followed: true, TaskOutcome: "failure", Confidence: 0.5},
		{Followed: false, TaskOutcome: "success", Confidence: 0.5},
		{Followed: false, TaskOutcome: "failure", Confidence: 0.5},
	}

	impact := GetComplianceImpact(recs)

	// Both should be 0.5
	if impact.FollowedSuccessRate != 0.5 {
		t.Errorf("Expected followed success rate 0.5, got: %f", impact.FollowedSuccessRate)
	}
	if impact.IgnoredSuccessRate != 0.5 {
		t.Errorf("Expected ignored success rate 0.5, got: %f", impact.IgnoredSuccessRate)
	}
	if impact.ImpactDelta != 0.0 {
		t.Errorf("Expected impact delta 0.0, got: %f", impact.ImpactDelta)
	}
	if impact.Recommendation != "neutral" {
		t.Errorf("Expected recommendation 'neutral', got: %s", impact.Recommendation)
	}
}

func TestGetComplianceImpact_InsufficientData(t *testing.T) {
	recs := []ScoutRecommendation{
		{Followed: true, TaskOutcome: "success", Confidence: 0.5},
	}

	impact := GetComplianceImpact(recs)

	if impact.StatisticalSignificance != "insufficient_data" {
		t.Errorf("Expected 'insufficient_data', got: %s", impact.StatisticalSignificance)
	}
}

func TestGetComplianceImpact_LowSignificance(t *testing.T) {
	recs := make([]ScoutRecommendation, 15)
	for i := range recs {
		recs[i] = ScoutRecommendation{
			Followed:    i%2 == 0,
			TaskOutcome: "success",
			Confidence:  0.5,
		}
	}

	impact := GetComplianceImpact(recs)

	if impact.StatisticalSignificance != "low" {
		t.Errorf("Expected 'low' significance for 15 samples, got: %s", impact.StatisticalSignificance)
	}
}

func TestGetComplianceImpact_MediumSignificance(t *testing.T) {
	recs := make([]ScoutRecommendation, 50)
	for i := range recs {
		recs[i] = ScoutRecommendation{
			Followed:    i%2 == 0,
			TaskOutcome: "success",
			Confidence:  0.5,
		}
	}

	impact := GetComplianceImpact(recs)

	if impact.StatisticalSignificance != "medium" {
		t.Errorf("Expected 'medium' significance for 50 samples, got: %s", impact.StatisticalSignificance)
	}
}

func TestGetComplianceImpact_HighSignificance(t *testing.T) {
	recs := make([]ScoutRecommendation, 150)
	for i := range recs {
		recs[i] = ScoutRecommendation{
			Followed:    i%2 == 0,
			TaskOutcome: "success",
			Confidence:  0.5,
		}
	}

	impact := GetComplianceImpact(recs)

	if impact.StatisticalSignificance != "high" {
		t.Errorf("Expected 'high' significance for 150 samples, got: %s", impact.StatisticalSignificance)
	}
}

func TestGetComplianceImpact_EmptyInput(t *testing.T) {
	recs := []ScoutRecommendation{}

	impact := GetComplianceImpact(recs)

	if impact.FollowedSuccessRate != 0.0 {
		t.Errorf("Expected 0.0 followed success rate, got: %f", impact.FollowedSuccessRate)
	}
	if impact.StatisticalSignificance != "insufficient_data" {
		t.Errorf("Expected 'insufficient_data', got: %s", impact.StatisticalSignificance)
	}
}

func TestClusterRecommendationsByTier_Basic(t *testing.T) {
	recs := []ScoutRecommendation{
		{RecommendedTier: "haiku", Followed: true, Confidence: 0.8},
		{RecommendedTier: "sonnet", Followed: true, Confidence: 0.9},
		{RecommendedTier: "sonnet", Followed: false, Confidence: 0.7},
	}

	clusters := ClusterRecommendationsByTier(recs)

	if len(clusters) != 2 {
		t.Errorf("Expected 2 tiers, got: %d", len(clusters))
	}

	sonnetStats := clusters["sonnet"]
	if sonnetStats.RecommendedCount != 2 {
		t.Errorf("Expected sonnet recommended 2 times, got: %d", sonnetStats.RecommendedCount)
	}
	if sonnetStats.ComplianceRate != 0.5 {
		t.Errorf("Expected sonnet compliance 0.5, got: %f", sonnetStats.ComplianceRate)
	}
}

func TestClusterRecommendationsByTier_EmptyTier(t *testing.T) {
	recs := []ScoutRecommendation{
		{RecommendedTier: "", Followed: true, Confidence: 0.8},
		{RecommendedTier: "", Followed: false, Confidence: 0.7},
	}

	clusters := ClusterRecommendationsByTier(recs)

	unknownStats := clusters["unknown"]
	if unknownStats == nil {
		t.Fatal("Expected 'unknown' tier for empty RecommendedTier")
	}
	if unknownStats.RecommendedCount != 2 {
		t.Errorf("Expected 2 unknown tier recommendations, got: %d", unknownStats.RecommendedCount)
	}
}

func TestClusterRecommendationsByTier_AvgConfidence(t *testing.T) {
	recs := []ScoutRecommendation{
		{RecommendedTier: "sonnet", Followed: true, Confidence: 0.8},
		{RecommendedTier: "sonnet", Followed: true, Confidence: 0.6},
	}

	clusters := ClusterRecommendationsByTier(recs)

	sonnetStats := clusters["sonnet"]
	// Average: (0.8 + 0.6) / 2 = 0.7
	if sonnetStats.AvgConfidence != 0.7 {
		t.Errorf("Expected avg confidence 0.7, got: %f", sonnetStats.AvgConfidence)
	}
}

func TestClusterRecommendationsByTier_EmptyInput(t *testing.T) {
	recs := []ScoutRecommendation{}

	clusters := ClusterRecommendationsByTier(recs)

	if len(clusters) != 0 {
		t.Errorf("Expected empty clusters for empty input, got: %d", len(clusters))
	}
}

func TestClusterRecommendationsByTier_ZeroComplianceRate(t *testing.T) {
	recs := []ScoutRecommendation{
		{RecommendedTier: "haiku", Followed: false, Confidence: 0.8},
		{RecommendedTier: "haiku", Followed: false, Confidence: 0.7},
	}

	clusters := ClusterRecommendationsByTier(recs)

	haikuStats := clusters["haiku"]
	if haikuStats.ComplianceRate != 0.0 {
		t.Errorf("Expected compliance rate 0.0, got: %f", haikuStats.ComplianceRate)
	}
}

func TestGetScoutPerformanceSummary_HighAccuracy(t *testing.T) {
	recs := []ScoutRecommendation{
		{Followed: true, TaskOutcome: "success", ScoutType: "haiku-scout", RecommendedTier: "sonnet", Confidence: 0.9},
		{Followed: true, TaskOutcome: "success", ScoutType: "haiku-scout", RecommendedTier: "sonnet", Confidence: 0.85},
		{Followed: true, TaskOutcome: "success", ScoutType: "haiku-scout", RecommendedTier: "sonnet", Confidence: 0.8},
	}

	summary := GetScoutPerformanceSummary(recs)

	if summary.AccuracyStats.FollowedAccuracy != 1.0 {
		t.Errorf("Expected 100%% accuracy, got: %f", summary.AccuracyStats.FollowedAccuracy)
	}

	if summary.OverallVerdict == "" {
		t.Error("Expected verdict to be generated")
	}

	// With high accuracy and small sample, should still be insufficient data
	if summary.ComplianceImpact.StatisticalSignificance != "insufficient_data" {
		t.Logf("Note: Got significance %s for 3 samples", summary.ComplianceImpact.StatisticalSignificance)
	}
}

func TestGetScoutPerformanceSummary_LowAccuracy(t *testing.T) {
	recs := []ScoutRecommendation{
		{Followed: true, TaskOutcome: "failure", ScoutType: "haiku-scout", RecommendedTier: "sonnet", Confidence: 0.9},
		{Followed: true, TaskOutcome: "failure", ScoutType: "haiku-scout", RecommendedTier: "sonnet", Confidence: 0.8},
		{Followed: true, TaskOutcome: "failure", ScoutType: "haiku-scout", RecommendedTier: "sonnet", Confidence: 0.7},
		{Followed: true, TaskOutcome: "failure", ScoutType: "haiku-scout", RecommendedTier: "sonnet", Confidence: 0.6},
		{Followed: true, TaskOutcome: "failure", ScoutType: "haiku-scout", RecommendedTier: "sonnet", Confidence: 0.5},
		{Followed: true, TaskOutcome: "failure", ScoutType: "haiku-scout", RecommendedTier: "sonnet", Confidence: 0.4},
		{Followed: true, TaskOutcome: "failure", ScoutType: "haiku-scout", RecommendedTier: "sonnet", Confidence: 0.3},
		{Followed: true, TaskOutcome: "failure", ScoutType: "haiku-scout", RecommendedTier: "sonnet", Confidence: 0.2},
		{Followed: true, TaskOutcome: "failure", ScoutType: "haiku-scout", RecommendedTier: "sonnet", Confidence: 0.1},
		{Followed: true, TaskOutcome: "success", ScoutType: "haiku-scout", RecommendedTier: "sonnet", Confidence: 0.95},
	}

	summary := GetScoutPerformanceSummary(recs)

	// 1 success / 10 total = 0.1 accuracy
	if summary.AccuracyStats.FollowedAccuracy != 0.1 {
		t.Errorf("Expected 0.1 accuracy, got: %f", summary.AccuracyStats.FollowedAccuracy)
	}

	if summary.OverallVerdict == "" {
		t.Error("Expected verdict to be generated")
	}
}

func TestGetScoutPerformanceSummary_InsufficientData(t *testing.T) {
	recs := []ScoutRecommendation{
		{Followed: true, TaskOutcome: "success", ScoutType: "haiku-scout", RecommendedTier: "sonnet", Confidence: 0.9},
	}

	summary := GetScoutPerformanceSummary(recs)

	if !strings.Contains(summary.OverallVerdict, "Insufficient data") {
		t.Errorf("Expected 'Insufficient data' verdict, got: %s", summary.OverallVerdict)
	}
}

func TestGetScoutPerformanceSummary_ModeratePerformance(t *testing.T) {
	// Create 50 recommendations with ~60% accuracy
	recs := make([]ScoutRecommendation, 50)
	for i := range recs {
		outcome := "success"
		if i >= 30 { // 30 success, 20 failure = 0.6 accuracy
			outcome = "failure"
		}
		recs[i] = ScoutRecommendation{
			Followed:        true,
			TaskOutcome:     outcome,
			ScoutType:       "haiku-scout",
			RecommendedTier: "sonnet",
			Confidence:      0.7,
		}
	}

	summary := GetScoutPerformanceSummary(recs)

	if summary.AccuracyStats.FollowedAccuracy != 0.6 {
		t.Errorf("Expected 0.6 accuracy, got: %f", summary.AccuracyStats.FollowedAccuracy)
	}

	if !strings.Contains(summary.OverallVerdict, "moderate") {
		t.Logf("Got verdict: %s", summary.OverallVerdict)
	}
}

func TestGetScoutPerformanceSummary_AllComponentsPresent(t *testing.T) {
	recs := []ScoutRecommendation{
		{Followed: true, TaskOutcome: "success", ScoutType: "haiku-scout", RecommendedTier: "sonnet", Confidence: 0.9},
		{Followed: true, TaskOutcome: "success", ScoutType: "haiku-scout", RecommendedTier: "haiku", Confidence: 0.8},
		{Followed: false, TaskOutcome: "failure", ScoutType: "gemini-scout", RecommendedTier: "opus", Confidence: 0.7},
	}

	summary := GetScoutPerformanceSummary(recs)

	// Verify all components are populated
	if summary.AccuracyStats.TotalRecommendations != 3 {
		t.Error("AccuracyStats not populated")
	}
	if len(summary.ConfidenceBuckets) != 4 {
		t.Error("ConfidenceBuckets not populated")
	}
	if summary.ComplianceImpact.StatisticalSignificance == "" {
		t.Error("ComplianceImpact not populated")
	}
	if len(summary.TierDistribution) == 0 {
		t.Error("TierDistribution not populated")
	}
	if summary.OverallVerdict == "" {
		t.Error("OverallVerdict not populated")
	}
}
