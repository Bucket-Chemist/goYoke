package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLoadUserIntents_ValidJSONL tests loading valid JSONL file
func TestLoadUserIntents_ValidJSONL(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	intentsPath := filepath.Join(claudeDir, "user-intents.jsonl")
	content := `{"timestamp":1705000000,"question":"Q1?","response":"A1","confidence":"explicit","source":"ask_user"}
{"timestamp":1705000001,"question":"Q2?","response":"A2","confidence":"inferred","source":"hook_prompt","action_taken":"Did something"}`

	if err := os.WriteFile(intentsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	intents, err := loadUserIntents(intentsPath)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(intents) != 2 {
		t.Errorf("Expected 2 intents, got: %d", len(intents))
	}

	// Verify first intent
	if intents[0].Question != "Q1?" {
		t.Errorf("Expected Q1?, got: %s", intents[0].Question)
	}
	if intents[0].Response != "A1" {
		t.Errorf("Expected A1, got: %s", intents[0].Response)
	}
	if intents[0].Confidence != "explicit" {
		t.Errorf("Expected explicit, got: %s", intents[0].Confidence)
	}
	if intents[0].Source != "ask_user" {
		t.Errorf("Expected ask_user, got: %s", intents[0].Source)
	}

	// Verify second intent
	if intents[1].ActionTaken != "Did something" {
		t.Errorf("Expected 'Did something', got: %s", intents[1].ActionTaken)
	}
	if intents[1].Confidence != "inferred" {
		t.Errorf("Expected inferred, got: %s", intents[1].Confidence)
	}
}

// TestLoadUserIntents_MissingFile tests that missing file returns empty slice
func TestLoadUserIntents_MissingFile(t *testing.T) {
	intents, err := loadUserIntents("/nonexistent/path.jsonl")
	if err != nil {
		t.Errorf("Expected no error for missing file, got: %v", err)
	}
	if len(intents) != 0 {
		t.Errorf("Expected empty slice, got: %d intents", len(intents))
	}
}

// TestLoadUserIntents_EmptyFile tests loading an empty file
func TestLoadUserIntents_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	intentsPath := filepath.Join(tmpDir, "intents.jsonl")
	if err := os.WriteFile(intentsPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	intents, err := loadUserIntents(intentsPath)
	if err != nil {
		t.Errorf("Expected no error for empty file, got: %v", err)
	}
	if len(intents) != 0 {
		t.Errorf("Expected empty slice for empty file, got: %d intents", len(intents))
	}
}

// TestLoadUserIntents_MalformedLines tests that malformed lines are skipped
func TestLoadUserIntents_MalformedLines(t *testing.T) {
	tmpDir := t.TempDir()
	intentsPath := filepath.Join(tmpDir, "intents.jsonl")

	content := `{"timestamp":1,"question":"Valid","response":"Yes","confidence":"explicit","source":"ask_user"}
not json
{"timestamp":2,"question":"Also valid","response":"No","confidence":"explicit","source":"ask_user"}
{broken json`

	if err := os.WriteFile(intentsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	intents, err := loadUserIntents(intentsPath)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(intents) != 2 {
		t.Errorf("Expected 2 valid intents (skipping malformed), got: %d", len(intents))
	}
}

// TestLoadUserIntents_AllFields tests that all fields are parsed correctly
func TestLoadUserIntents_AllFields(t *testing.T) {
	tmpDir := t.TempDir()
	intentsPath := filepath.Join(tmpDir, "intents.jsonl")

	content := `{"timestamp":1705000000,"question":"Which auth method?","response":"JWT with refresh tokens","confidence":"explicit","context":"Setting up auth","source":"ask_user","action_taken":"Implemented JWT auth"}`

	if err := os.WriteFile(intentsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	intents, err := loadUserIntents(intentsPath)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(intents) != 1 {
		t.Fatalf("Expected 1 intent, got: %d", len(intents))
	}

	intent := intents[0]
	if intent.Timestamp != 1705000000 {
		t.Errorf("Expected timestamp 1705000000, got: %d", intent.Timestamp)
	}
	if intent.Question != "Which auth method?" {
		t.Errorf("Expected 'Which auth method?', got: %s", intent.Question)
	}
	if intent.Response != "JWT with refresh tokens" {
		t.Errorf("Expected 'JWT with refresh tokens', got: %s", intent.Response)
	}
	if intent.Context != "Setting up auth" {
		t.Errorf("Expected 'Setting up auth', got: %s", intent.Context)
	}
	if intent.ActionTaken != "Implemented JWT auth" {
		t.Errorf("Expected 'Implemented JWT auth', got: %s", intent.ActionTaken)
	}
}

// TestQueryUserIntents_NoFilters tests query without any filters
func TestQueryUserIntents_NoFilters(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	intentsPath := filepath.Join(claudeDir, "user-intents.jsonl")
	content := `{"timestamp":1,"question":"Q1","response":"A1","confidence":"explicit","source":"ask_user"}
{"timestamp":2,"question":"Q2","response":"A2","confidence":"inferred","source":"hook_prompt"}`

	if err := os.WriteFile(intentsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	q := NewQuery(tmpDir)
	intents, err := q.QueryUserIntents(UserIntentFilters{})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(intents) != 2 {
		t.Errorf("Expected 2 intents, got: %d", len(intents))
	}
}

// TestQueryUserIntents_SourceFilter tests filtering by source
func TestQueryUserIntents_SourceFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	intentsPath := filepath.Join(claudeDir, "user-intents.jsonl")
	content := `{"timestamp":1,"question":"Q1","response":"A1","confidence":"explicit","source":"ask_user"}
{"timestamp":2,"question":"Q2","response":"A2","confidence":"explicit","source":"hook_prompt"}
{"timestamp":3,"question":"Q3","response":"A3","confidence":"explicit","source":"ask_user"}
{"timestamp":4,"question":"Q4","response":"A4","confidence":"explicit","source":"manual"}`

	if err := os.WriteFile(intentsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	q := NewQuery(tmpDir)

	// Filter by ask_user
	source := "ask_user"
	intents, err := q.QueryUserIntents(UserIntentFilters{Source: &source})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(intents) != 2 {
		t.Errorf("Expected 2 ask_user intents, got: %d", len(intents))
	}

	// Filter by hook_prompt
	source = "hook_prompt"
	intents, err = q.QueryUserIntents(UserIntentFilters{Source: &source})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(intents) != 1 {
		t.Errorf("Expected 1 hook_prompt intent, got: %d", len(intents))
	}

	// Filter by manual
	source = "manual"
	intents, err = q.QueryUserIntents(UserIntentFilters{Source: &source})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(intents) != 1 {
		t.Errorf("Expected 1 manual intent, got: %d", len(intents))
	}
}

// TestQueryUserIntents_ConfidenceFilter tests filtering by confidence level
func TestQueryUserIntents_ConfidenceFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	intentsPath := filepath.Join(claudeDir, "user-intents.jsonl")
	content := `{"timestamp":1,"question":"Q1","response":"A1","confidence":"explicit","source":"ask_user"}
{"timestamp":2,"question":"Q2","response":"","confidence":"default","source":"ask_user"}
{"timestamp":3,"question":"Q3","response":"A3","confidence":"inferred","source":"ask_user"}
{"timestamp":4,"question":"Q4","response":"A4","confidence":"explicit","source":"ask_user"}`

	if err := os.WriteFile(intentsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	q := NewQuery(tmpDir)

	// Filter explicit
	conf := "explicit"
	intents, err := q.QueryUserIntents(UserIntentFilters{Confidence: &conf})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(intents) != 2 {
		t.Errorf("Expected 2 explicit intents, got: %d", len(intents))
	}

	// Filter default
	conf = "default"
	intents, err = q.QueryUserIntents(UserIntentFilters{Confidence: &conf})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(intents) != 1 {
		t.Errorf("Expected 1 default intent, got: %d", len(intents))
	}

	// Filter inferred
	conf = "inferred"
	intents, err = q.QueryUserIntents(UserIntentFilters{Confidence: &conf})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(intents) != 1 {
		t.Errorf("Expected 1 inferred intent, got: %d", len(intents))
	}
}

// TestQueryUserIntents_HasActionFilter tests filtering by action presence
func TestQueryUserIntents_HasActionFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	intentsPath := filepath.Join(claudeDir, "user-intents.jsonl")
	content := `{"timestamp":1,"question":"Q1","response":"A1","confidence":"explicit","source":"ask_user","action_taken":"Did X"}
{"timestamp":2,"question":"Q2","response":"A2","confidence":"explicit","source":"ask_user"}
{"timestamp":3,"question":"Q3","response":"A3","confidence":"explicit","source":"ask_user","action_taken":"Did Y"}
{"timestamp":4,"question":"Q4","response":"A4","confidence":"explicit","source":"ask_user","action_taken":""}`

	if err := os.WriteFile(intentsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	q := NewQuery(tmpDir)
	intents, err := q.QueryUserIntents(UserIntentFilters{HasAction: true})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(intents) != 2 {
		t.Errorf("Expected 2 intents with actions, got: %d", len(intents))
	}
}

// TestQueryUserIntents_SinceFilter tests filtering by timestamp
func TestQueryUserIntents_SinceFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	now := time.Now().Unix()
	oldTimestamp := now - (30 * 24 * 60 * 60)   // 30 days ago
	recentTimestamp := now - (5 * 24 * 60 * 60) // 5 days ago

	intentsPath := filepath.Join(claudeDir, "user-intents.jsonl")
	content := `{"timestamp":` + itoa(oldTimestamp) + `,"question":"Old Q","response":"A","confidence":"explicit","source":"ask_user"}
{"timestamp":` + itoa(recentTimestamp) + `,"question":"Recent Q","response":"A","confidence":"explicit","source":"ask_user"}`

	if err := os.WriteFile(intentsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	q := NewQuery(tmpDir)
	since := now - (7 * 24 * 60 * 60) // 7 days ago
	intents, err := q.QueryUserIntents(UserIntentFilters{Since: &since})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(intents) != 1 {
		t.Errorf("Expected 1 recent intent, got: %d", len(intents))
	}
	if len(intents) > 0 && intents[0].Question != "Recent Q" {
		t.Errorf("Expected 'Recent Q', got: %s", intents[0].Question)
	}
}

// TestQueryUserIntents_LimitFilter tests limiting results
func TestQueryUserIntents_LimitFilter(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	intentsPath := filepath.Join(claudeDir, "user-intents.jsonl")
	content := `{"timestamp":1,"question":"Q1","response":"A1","confidence":"explicit","source":"ask_user"}
{"timestamp":2,"question":"Q2","response":"A2","confidence":"explicit","source":"ask_user"}
{"timestamp":3,"question":"Q3","response":"A3","confidence":"explicit","source":"ask_user"}`

	if err := os.WriteFile(intentsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	q := NewQuery(tmpDir)
	intents, err := q.QueryUserIntents(UserIntentFilters{Limit: 2})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(intents) != 2 {
		t.Errorf("Expected 2 intents (limit), got: %d", len(intents))
	}
}

// TestQueryUserIntents_CombinedFilters tests multiple filters together
func TestQueryUserIntents_CombinedFilters(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	intentsPath := filepath.Join(claudeDir, "user-intents.jsonl")
	content := `{"timestamp":1,"question":"Q1","response":"A1","confidence":"explicit","source":"ask_user","action_taken":"Did it"}
{"timestamp":2,"question":"Q2","response":"A2","confidence":"explicit","source":"ask_user"}
{"timestamp":3,"question":"Q3","response":"A3","confidence":"inferred","source":"ask_user","action_taken":"Did that"}
{"timestamp":4,"question":"Q4","response":"A4","confidence":"explicit","source":"hook_prompt","action_taken":"Action"}`

	if err := os.WriteFile(intentsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	q := NewQuery(tmpDir)
	source := "ask_user"
	conf := "explicit"
	intents, err := q.QueryUserIntents(UserIntentFilters{
		Source:     &source,
		Confidence: &conf,
		HasAction:  true,
	})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Only Q1 matches: ask_user + explicit + has action
	if len(intents) != 1 {
		t.Errorf("Expected 1 intent matching all filters, got: %d", len(intents))
	}
	if len(intents) > 0 && intents[0].Question != "Q1" {
		t.Errorf("Expected Q1, got: %s", intents[0].Question)
	}
}

// TestQueryUserIntents_MissingFile tests query on missing file
func TestQueryUserIntents_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()

	q := NewQuery(tmpDir)
	intents, err := q.QueryUserIntents(UserIntentFilters{})
	if err != nil {
		t.Errorf("Expected no error for missing file, got: %v", err)
	}

	if len(intents) != 0 {
		t.Errorf("Expected empty slice for missing file, got: %d intents", len(intents))
	}
}

// TestQueryUserIntents_MalformedLines tests that malformed lines are skipped in query
func TestQueryUserIntents_MalformedLines(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	intentsPath := filepath.Join(claudeDir, "user-intents.jsonl")
	content := `not valid json
{"timestamp":1,"question":"Valid","response":"A","confidence":"explicit","source":"ask_user"}
{broken json
{"timestamp":2,"question":"Also valid","response":"B","confidence":"explicit","source":"ask_user"}`

	if err := os.WriteFile(intentsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	q := NewQuery(tmpDir)
	intents, err := q.QueryUserIntents(UserIntentFilters{})
	if err != nil {
		t.Fatalf("Expected no error (skip malformed), got: %v", err)
	}

	if len(intents) != 2 {
		t.Errorf("Expected 2 valid intents (skipped malformed), got: %d", len(intents))
	}
}

// TestFormatUserIntent_WithAction tests formatting with action
func TestFormatUserIntent_WithAction(t *testing.T) {
	intent := UserIntent{
		Timestamp:   1705000000,
		Question:    "Should we use Redis for caching?",
		Response:    "Yes, with 1 hour TTL",
		Confidence:  "explicit",
		Source:      "ask_user",
		ActionTaken: "Added Redis cache layer",
	}

	formatted := FormatUserIntent(intent)

	if !strings.Contains(formatted, "💬") {
		t.Error("Expected explicit badge (💬)")
	}
	if !strings.Contains(formatted, "**Q:**") {
		t.Error("Expected question marker")
	}
	if !strings.Contains(formatted, "**A:**") {
		t.Error("Expected answer marker")
	}
	if !strings.Contains(formatted, "➡️ Action:") {
		t.Error("Expected action marker")
	}
	if !strings.Contains(formatted, "Redis") {
		t.Error("Expected content to be present")
	}
}

// TestFormatUserIntent_WithoutAction tests formatting without action
func TestFormatUserIntent_WithoutAction(t *testing.T) {
	intent := UserIntent{
		Timestamp:  1705000000,
		Question:   "Any preference?",
		Response:   "No",
		Confidence: "inferred",
		Source:     "ask_user",
	}

	formatted := FormatUserIntent(intent)

	if !strings.Contains(formatted, "🤔") {
		t.Error("Expected inferred badge (🤔)")
	}
	if strings.Contains(formatted, "➡️") {
		t.Error("Expected no action marker for empty action")
	}
}

// TestFormatUserIntent_DefaultConfidence tests formatting with default confidence
func TestFormatUserIntent_DefaultConfidence(t *testing.T) {
	intent := UserIntent{
		Timestamp:  1705000000,
		Question:   "Preferred log format?",
		Response:   "",
		Confidence: "default",
		Source:     "ask_user",
	}

	formatted := FormatUserIntent(intent)

	if !strings.Contains(formatted, "⚪") {
		t.Error("Expected default badge (⚪)")
	}
}

// TestFormatUserIntent_UnknownConfidence tests formatting with unknown confidence
func TestFormatUserIntent_UnknownConfidence(t *testing.T) {
	intent := UserIntent{
		Timestamp:  1705000000,
		Question:   "Test?",
		Response:   "Test",
		Confidence: "unknown_value",
		Source:     "ask_user",
	}

	formatted := FormatUserIntent(intent)

	if !strings.Contains(formatted, "❓") {
		t.Error("Expected unknown badge (❓) for unrecognized confidence")
	}
}

// TestFormatUserIntent_Truncation tests that long strings are truncated
func TestFormatUserIntent_Truncation(t *testing.T) {
	longQuestion := strings.Repeat("Q", 100)
	longResponse := strings.Repeat("R", 150)
	longAction := strings.Repeat("A", 100)

	intent := UserIntent{
		Timestamp:   1705000000,
		Question:    longQuestion,
		Response:    longResponse,
		Confidence:  "explicit",
		Source:      "ask_user",
		ActionTaken: longAction,
	}

	formatted := FormatUserIntent(intent)

	// Should be truncated
	if strings.Contains(formatted, longQuestion) {
		t.Error("Question should be truncated")
	}
	if strings.Contains(formatted, longResponse) {
		t.Error("Response should be truncated")
	}
	if strings.Contains(formatted, longAction) {
		t.Error("Action should be truncated")
	}
	// Should have ellipsis
	if !strings.Contains(formatted, "...") {
		t.Error("Expected truncation with ellipsis")
	}
}

// TestFormatUserIntents_Empty tests formatting empty slice
func TestFormatUserIntents_Empty(t *testing.T) {
	formatted := FormatUserIntents([]UserIntent{})
	if formatted != "" {
		t.Errorf("Expected empty string for empty intents, got: %s", formatted)
	}
}

// TestFormatUserIntents_Multiple tests formatting multiple intents
func TestFormatUserIntents_Multiple(t *testing.T) {
	intents := []UserIntent{
		{
			Timestamp:   1,
			Question:    "Q1?",
			Response:    "A1",
			Confidence:  "explicit",
			Source:      "ask_user",
			ActionTaken: "Action 1",
		},
		{
			Timestamp:  2,
			Question:   "Q2?",
			Response:   "A2",
			Confidence: "inferred",
			Source:     "hook_prompt",
		},
	}

	formatted := FormatUserIntents(intents)

	// Should have header
	if !strings.Contains(formatted, "## User Intents") {
		t.Error("Expected User Intents header")
	}

	// Should have both intents
	if !strings.Contains(formatted, "Q1?") {
		t.Error("Expected first intent question")
	}
	if !strings.Contains(formatted, "Q2?") {
		t.Error("Expected second intent question")
	}

	// Should have appropriate badges
	if !strings.Contains(formatted, "💬") {
		t.Error("Expected explicit badge")
	}
	if !strings.Contains(formatted, "🤔") {
		t.Error("Expected inferred badge")
	}
}

// TestLoadArtifacts_IncludesUserIntents tests that LoadArtifacts includes user intents
func TestLoadArtifacts_IncludesUserIntents(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create user intents file
	intentsPath := filepath.Join(claudeDir, "user-intents.jsonl")
	content := `{"timestamp":1,"question":"Q1","response":"A1","confidence":"explicit","source":"ask_user"}
{"timestamp":2,"question":"Q2","response":"A2","confidence":"inferred","source":"hook_prompt"}`
	if err := os.WriteFile(intentsPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create empty files for other artifacts
	if err := os.WriteFile(filepath.Join(claudeDir, "pending-learnings.jsonl"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &HandoffConfig{
		ProjectDir:        tmpDir,
		PendingPath:       filepath.Join(claudeDir, "pending-learnings.jsonl"),
		ViolationsPath:    filepath.Join(claudeDir, "routing-violations.jsonl"),
		ErrorPatternsPath: filepath.Join(claudeDir, "error-patterns.jsonl"),
		UserIntentsPath:   intentsPath,
	}

	artifacts, err := LoadArtifacts(cfg)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(artifacts.UserIntents) != 2 {
		t.Errorf("Expected 2 user intents, got: %d", len(artifacts.UserIntents))
	}
}

// TestLoadArtifacts_MissingUserIntentsFile tests that missing user intents file is OK
func TestLoadArtifacts_MissingUserIntentsFile(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "memory")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create empty files for other artifacts (but NOT user-intents.jsonl)
	if err := os.WriteFile(filepath.Join(claudeDir, "pending-learnings.jsonl"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &HandoffConfig{
		ProjectDir:        tmpDir,
		PendingPath:       filepath.Join(claudeDir, "pending-learnings.jsonl"),
		ViolationsPath:    filepath.Join(claudeDir, "routing-violations.jsonl"),
		ErrorPatternsPath: filepath.Join(claudeDir, "error-patterns.jsonl"),
		UserIntentsPath:   filepath.Join(claudeDir, "user-intents.jsonl"),
	}

	artifacts, err := LoadArtifacts(cfg)
	if err != nil {
		t.Fatalf("Expected no error for missing user intents file, got: %v", err)
	}

	if len(artifacts.UserIntents) != 0 {
		t.Errorf("Expected 0 user intents for missing file, got: %d", len(artifacts.UserIntents))
	}
}

// TestDefaultHandoffConfig_IncludesUserIntentsPath tests that default config includes user intents path
func TestDefaultHandoffConfig_IncludesUserIntentsPath(t *testing.T) {
	cfg := DefaultHandoffConfig("/test/project")

	expectedPath := "/test/project/.claude/memory/user-intents.jsonl"
	if cfg.UserIntentsPath != expectedPath {
		t.Errorf("Expected UserIntentsPath '%s', got: '%s'", expectedPath, cfg.UserIntentsPath)
	}
}

// TestValidConfidenceLevels tests the valid confidence level map
func TestValidConfidenceLevels(t *testing.T) {
	validLevels := []string{"explicit", "inferred", "default"}
	for _, level := range validLevels {
		if !ValidConfidenceLevels[level] {
			t.Errorf("Expected '%s' to be a valid confidence level", level)
		}
	}

	invalidLevels := []string{"unknown", "maybe", ""}
	for _, level := range invalidLevels {
		if ValidConfidenceLevels[level] {
			t.Errorf("Expected '%s' to be an invalid confidence level", level)
		}
	}
}

// TestValidIntentSources tests the valid intent sources map
func TestValidIntentSources(t *testing.T) {
	validSources := []string{"ask_user", "hook_prompt", "manual"}
	for _, source := range validSources {
		if !ValidIntentSources[source] {
			t.Errorf("Expected '%s' to be a valid intent source", source)
		}
	}

	invalidSources := []string{"unknown", "auto", ""}
	for _, source := range invalidSources {
		if ValidIntentSources[source] {
			t.Errorf("Expected '%s' to be an invalid intent source", source)
		}
	}
}

// TestTruncateString tests the truncate helper function
func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is longer", 10, "this is..."},
		{"", 10, ""},
		{"abc", 3, "abc"},
		{"abcd", 3, "..."},
	}

	for _, tc := range tests {
		result := truncateString(tc.input, tc.maxLen)
		if result != tc.expected {
			t.Errorf("truncateString(%q, %d) = %q, want %q", tc.input, tc.maxLen, result, tc.expected)
		}
	}
}
