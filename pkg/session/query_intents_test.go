package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestQueryUserIntents_CategoryFilter verifies category filtering
func TestQueryUserIntents_CategoryFilter(t *testing.T) {
	// Setup test data
	tmpDir := t.TempDir()
	intentsPath := filepath.Join(tmpDir, ".gogent", "memory", "user-intents.jsonl")
	if err := os.MkdirAll(filepath.Dir(intentsPath), 0755); err != nil {
		t.Fatal(err)
	}

	// Create test intents with different categories
	testIntents := []UserIntent{
		{
			Timestamp: 1000,
			Question:  "Which model?",
			Response:  "Use sonnet",
			Category:  "routing",
			Source:    "ask_user",
		},
		{
			Timestamp: 2000,
			Question:  "Which tool?",
			Response:  "Use edit",
			Category:  "tooling",
			Source:    "ask_user",
		},
		{
			Timestamp: 3000,
			Question:  "Format?",
			Response:  "Be concise",
			Category:  "style",
			Source:    "ask_user",
		},
		{
			Timestamp: 4000,
			Question:  "Another model?",
			Response:  "Use haiku",
			Category:  "routing",
			Source:    "ask_user",
		},
	}

	// Write test data
	f, err := os.Create(intentsPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, intent := range testIntents {
		data, _ := json.Marshal(intent)
		f.WriteString(string(data) + "\n")
	}
	f.Close()

	// Query with category filter
	q := NewQuery(tmpDir)
	category := "routing"
	intents, err := q.QueryUserIntents(UserIntentFilters{
		Category: &category,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Verify results
	if len(intents) != 2 {
		t.Errorf("Expected 2 routing intents, got %d", len(intents))
	}
	for _, intent := range intents {
		if intent.Category != "routing" {
			t.Errorf("Expected routing category, got %q", intent.Category)
		}
	}
}

// TestQueryUserIntents_KeywordFilter verifies keyword filtering
func TestQueryUserIntents_KeywordFilter(t *testing.T) {
	tmpDir := t.TempDir()
	intentsPath := filepath.Join(tmpDir, ".gogent", "memory", "user-intents.jsonl")
	if err := os.MkdirAll(filepath.Dir(intentsPath), 0755); err != nil {
		t.Fatal(err)
	}

	// Create test intents with different keywords
	testIntents := []UserIntent{
		{
			Timestamp: 1000,
			Question:  "Tool choice?",
			Response:  "Use edit",
			Keywords:  []string{"edit", "bash"},
			Source:    "ask_user",
		},
		{
			Timestamp: 2000,
			Question:  "Model?",
			Response:  "Sonnet",
			Keywords:  []string{"sonnet"},
			Source:    "ask_user",
		},
		{
			Timestamp: 3000,
			Question:  "Files?",
			Response:  "main.go",
			Keywords:  []string{"main.go", "test.go"},
			Source:    "ask_user",
		},
		{
			Timestamp: 4000,
			Question:  "Another tool?",
			Response:  "Edit again",
			Keywords:  []string{"edit", "query.go"},
			Source:    "ask_user",
		},
	}

	f, err := os.Create(intentsPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, intent := range testIntents {
		data, _ := json.Marshal(intent)
		f.WriteString(string(data) + "\n")
	}
	f.Close()

	// Query with keyword filter
	q := NewQuery(tmpDir)
	intents, err := q.QueryUserIntents(UserIntentFilters{
		Keywords: []string{"edit"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Should match intents 1 and 4 (both have "edit" keyword)
	if len(intents) != 2 {
		t.Errorf("Expected 2 intents with 'edit' keyword, got %d", len(intents))
	}
	for _, intent := range intents {
		hasEdit := false
		for _, kw := range intent.Keywords {
			if kw == "edit" {
				hasEdit = true
				break
			}
		}
		if !hasEdit {
			t.Errorf("Expected 'edit' keyword in intent: %v", intent.Keywords)
		}
	}
}

// TestQueryUserIntents_KeywordFilter_CaseInsensitive verifies case-insensitive keyword matching
func TestQueryUserIntents_KeywordFilter_CaseInsensitive(t *testing.T) {
	tmpDir := t.TempDir()
	intentsPath := filepath.Join(tmpDir, ".gogent", "memory", "user-intents.jsonl")
	if err := os.MkdirAll(filepath.Dir(intentsPath), 0755); err != nil {
		t.Fatal(err)
	}

	testIntents := []UserIntent{
		{
			Timestamp: 1000,
			Response:  "Use EDIT",
			Keywords:  []string{"edit"},
			Source:    "ask_user",
		},
	}

	f, err := os.Create(intentsPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, intent := range testIntents {
		data, _ := json.Marshal(intent)
		f.WriteString(string(data) + "\n")
	}
	f.Close()

	q := NewQuery(tmpDir)

	// Query with uppercase keyword
	intents, err := q.QueryUserIntents(UserIntentFilters{
		Keywords: []string{"EDIT"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(intents) != 1 {
		t.Errorf("Expected 1 intent (case-insensitive), got %d", len(intents))
	}
}

// TestQueryUserIntents_MultipleKeywords verifies ANY keyword matching
func TestQueryUserIntents_MultipleKeywords(t *testing.T) {
	tmpDir := t.TempDir()
	intentsPath := filepath.Join(tmpDir, ".gogent", "memory", "user-intents.jsonl")
	if err := os.MkdirAll(filepath.Dir(intentsPath), 0755); err != nil {
		t.Fatal(err)
	}

	testIntents := []UserIntent{
		{
			Timestamp: 1000,
			Response:  "Use edit",
			Keywords:  []string{"edit"},
			Source:    "ask_user",
		},
		{
			Timestamp: 2000,
			Response:  "Use bash",
			Keywords:  []string{"bash"},
			Source:    "ask_user",
		},
		{
			Timestamp: 3000,
			Response:  "Use grep",
			Keywords:  []string{"grep"},
			Source:    "ask_user",
		},
	}

	f, err := os.Create(intentsPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, intent := range testIntents {
		data, _ := json.Marshal(intent)
		f.WriteString(string(data) + "\n")
	}
	f.Close()

	q := NewQuery(tmpDir)

	// Query with multiple keywords (ANY match)
	intents, err := q.QueryUserIntents(UserIntentFilters{
		Keywords: []string{"edit", "bash"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Should match first 2 intents (edit OR bash)
	if len(intents) != 2 {
		t.Errorf("Expected 2 intents (edit OR bash), got %d", len(intents))
	}
}

// TestQueryUserIntents_CategoryAndKeyword verifies combined filters
func TestQueryUserIntents_CategoryAndKeyword(t *testing.T) {
	tmpDir := t.TempDir()
	intentsPath := filepath.Join(tmpDir, ".gogent", "memory", "user-intents.jsonl")
	if err := os.MkdirAll(filepath.Dir(intentsPath), 0755); err != nil {
		t.Fatal(err)
	}

	testIntents := []UserIntent{
		{
			Timestamp: 1000,
			Response:  "Use sonnet",
			Category:  "routing",
			Keywords:  []string{"sonnet"},
			Source:    "ask_user",
		},
		{
			Timestamp: 2000,
			Response:  "Use edit tool",
			Category:  "tooling",
			Keywords:  []string{"edit"},
			Source:    "ask_user",
		},
		{
			Timestamp: 3000,
			Response:  "Delegate to haiku",
			Category:  "routing",
			Keywords:  []string{"haiku"},
			Source:    "ask_user",
		},
	}

	f, err := os.Create(intentsPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, intent := range testIntents {
		data, _ := json.Marshal(intent)
		f.WriteString(string(data) + "\n")
	}
	f.Close()

	q := NewQuery(tmpDir)

	// Query for routing category with "sonnet" keyword
	category := "routing"
	intents, err := q.QueryUserIntents(UserIntentFilters{
		Category: &category,
		Keywords: []string{"sonnet"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Should match only first intent (routing AND sonnet)
	if len(intents) != 1 {
		t.Errorf("Expected 1 intent (routing + sonnet), got %d", len(intents))
	}
	if intents[0].Category != "routing" || !contains(intents[0].Keywords, "sonnet") {
		t.Errorf("Expected routing category with sonnet keyword, got %+v", intents[0])
	}
}

// TestQueryUserIntents_BackwardCompatibility verifies empty category/keywords don't break queries
func TestQueryUserIntents_BackwardCompatibility(t *testing.T) {
	tmpDir := t.TempDir()
	intentsPath := filepath.Join(tmpDir, ".gogent", "memory", "user-intents.jsonl")
	if err := os.MkdirAll(filepath.Dir(intentsPath), 0755); err != nil {
		t.Fatal(err)
	}

	// Old intents without category/keywords fields
	testIntents := []UserIntent{
		{
			Timestamp: 1000,
			Question:  "Old intent",
			Response:  "No category",
			Source:    "ask_user",
			// Category and Keywords omitted (legacy format)
		},
	}

	f, err := os.Create(intentsPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, intent := range testIntents {
		data, _ := json.Marshal(intent)
		f.WriteString(string(data) + "\n")
	}
	f.Close()

	q := NewQuery(tmpDir)

	// Query all intents (no filter)
	intents, err := q.QueryUserIntents(UserIntentFilters{})
	if err != nil {
		t.Fatal(err)
	}

	if len(intents) != 1 {
		t.Errorf("Expected 1 intent, got %d", len(intents))
	}

	// Verify empty category/keywords parsed correctly
	if intents[0].Category != "" {
		t.Errorf("Expected empty category, got %q", intents[0].Category)
	}
	if len(intents[0].Keywords) != 0 {
		t.Errorf("Expected empty keywords, got %v", intents[0].Keywords)
	}
}

// TestQueryUserIntents_EmptyKeywordsNoMatch verifies intents with no keywords don't match keyword filter
func TestQueryUserIntents_EmptyKeywordsNoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	intentsPath := filepath.Join(tmpDir, ".gogent", "memory", "user-intents.jsonl")
	if err := os.MkdirAll(filepath.Dir(intentsPath), 0755); err != nil {
		t.Fatal(err)
	}

	testIntents := []UserIntent{
		{
			Timestamp: 1000,
			Response:  "No keywords",
			Keywords:  []string{}, // Empty keywords
			Source:    "ask_user",
		},
	}

	f, err := os.Create(intentsPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, intent := range testIntents {
		data, _ := json.Marshal(intent)
		f.WriteString(string(data) + "\n")
	}
	f.Close()

	q := NewQuery(tmpDir)

	// Query with keyword filter
	intents, err := q.QueryUserIntents(UserIntentFilters{
		Keywords: []string{"edit"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Should match nothing (intent has no keywords)
	if len(intents) != 0 {
		t.Errorf("Expected 0 intents, got %d", len(intents))
	}
}
