package session

import (
	"testing"
)

// contains checks if a string is in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// TestExtractKeywords_Tools verifies tool keyword extraction
func TestExtractKeywords_Tools(t *testing.T) {
	keywords := ExtractKeywords("Use Edit not sed, and run bash after")
	if !contains(keywords, "edit") {
		t.Errorf("Expected 'edit' in keywords: %v", keywords)
	}
	if !contains(keywords, "bash") {
		t.Errorf("Expected 'bash' in keywords: %v", keywords)
	}
}

// TestExtractKeywords_Models verifies model keyword extraction
func TestExtractKeywords_Models(t *testing.T) {
	keywords := ExtractKeywords("Use sonnet not haiku, opus is too expensive")
	if !contains(keywords, "sonnet") {
		t.Errorf("Expected 'sonnet' in keywords: %v", keywords)
	}
	if !contains(keywords, "haiku") {
		t.Errorf("Expected 'haiku' in keywords: %v", keywords)
	}
	if !contains(keywords, "opus") {
		t.Errorf("Expected 'opus' in keywords: %v", keywords)
	}
}

// TestExtractKeywords_TestFrameworks verifies test framework extraction
func TestExtractKeywords_TestFrameworks(t *testing.T) {
	keywords := ExtractKeywords("Run pytest and go test, skip jest")
	if !contains(keywords, "pytest") {
		t.Errorf("Expected 'pytest' in keywords: %v", keywords)
	}
	if !contains(keywords, "go test") {
		t.Errorf("Expected 'go test' in keywords: %v", keywords)
	}
	if !contains(keywords, "jest") {
		t.Errorf("Expected 'jest' in keywords: %v", keywords)
	}
}

// TestExtractKeywords_BuildTools verifies build tool extraction
func TestExtractKeywords_BuildTools(t *testing.T) {
	keywords := ExtractKeywords("Use make then npm install, not yarn")
	if !contains(keywords, "make") {
		t.Errorf("Expected 'make' in keywords: %v", keywords)
	}
	if !contains(keywords, "npm") {
		t.Errorf("Expected 'npm' in keywords: %v", keywords)
	}
	if !contains(keywords, "yarn") {
		t.Errorf("Expected 'yarn' in keywords: %v", keywords)
	}
}

// TestExtractKeywords_VCS verifies version control keyword extraction
func TestExtractKeywords_VCS(t *testing.T) {
	keywords := ExtractKeywords("After git commit, do git push")
	if !contains(keywords, "git") {
		t.Errorf("Expected 'git' in keywords: %v", keywords)
	}
	if !contains(keywords, "commit") {
		t.Errorf("Expected 'commit' in keywords: %v", keywords)
	}
	if !contains(keywords, "push") {
		t.Errorf("Expected 'push' in keywords: %v", keywords)
	}
}

// TestExtractKeywords_FilePaths verifies file path extraction
func TestExtractKeywords_FilePaths(t *testing.T) {
	keywords := ExtractKeywords("Check pkg/session/query.go and main.go")
	if !contains(keywords, "query.go") {
		t.Errorf("Expected 'query.go' in keywords: %v", keywords)
	}
	if !contains(keywords, "main.go") {
		t.Errorf("Expected 'main.go' in keywords: %v", keywords)
	}
}

// TestExtractKeywords_FilePathsVariousExtensions verifies extraction of different file types
func TestExtractKeywords_FilePathsVariousExtensions(t *testing.T) {
	cases := []struct {
		input    string
		expected []string
	}{
		{
			"Edit src/main.py and test.py",
			[]string{"main.py", "test.py"},
		},
		{
			"Check config.json and settings.yaml",
			[]string{"config.json", "settings.yaml"},
		},
		{
			"Read README.md and setup.sh",
			[]string{"readme.md", "setup.sh"},
		},
		{
			"Update handler.go and handler_test.go",
			[]string{"handler.go", "handler_test.go"},
		},
		{
			"Fix component.tsx and index.ts",
			[]string{"component.tsx", "index.ts"},
		},
	}

	for _, tc := range cases {
		keywords := ExtractKeywords(tc.input)
		for _, exp := range tc.expected {
			if !contains(keywords, exp) {
				t.Errorf("Input %q: expected %q in keywords %v", tc.input, exp, keywords)
			}
		}
	}
}

// TestExtractKeywords_MaxLimit verifies keyword limit enforcement
func TestExtractKeywords_MaxLimit(t *testing.T) {
	// Response with many keywords (more than 10)
	response := "edit bash read write glob grep task sonnet haiku opus pytest jest mocha unittest go test npm yarn"
	keywords := ExtractKeywords(response)
	if len(keywords) > 10 {
		t.Errorf("Expected max 10 keywords, got %d: %v", len(keywords), keywords)
	}
}

// TestExtractKeywords_Deduplication verifies no duplicate keywords
func TestExtractKeywords_Deduplication(t *testing.T) {
	// Repeated keywords
	response := "Use edit and edit again, also edit one more time"
	keywords := ExtractKeywords(response)

	// Check for duplicates
	seen := make(map[string]bool)
	for _, kw := range keywords {
		if seen[kw] {
			t.Errorf("Duplicate keyword found: %q in %v", kw, keywords)
		}
		seen[kw] = true
	}

	// Should have exactly 1 "edit"
	count := 0
	for _, kw := range keywords {
		if kw == "edit" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("Expected 1 'edit' keyword, got %d", count)
	}
}

// TestExtractKeywords_CaseInsensitive verifies case-insensitive matching
func TestExtractKeywords_CaseInsensitive(t *testing.T) {
	keywords := ExtractKeywords("Use EDIT and Bash, also SONNET")
	if !contains(keywords, "edit") {
		t.Errorf("Expected 'edit' (lowercase) in keywords: %v", keywords)
	}
	if !contains(keywords, "bash") {
		t.Errorf("Expected 'bash' (lowercase) in keywords: %v", keywords)
	}
	if !contains(keywords, "sonnet") {
		t.Errorf("Expected 'sonnet' (lowercase) in keywords: %v", keywords)
	}
}

// TestExtractKeywords_EmptyInput verifies handling of empty input
func TestExtractKeywords_EmptyInput(t *testing.T) {
	keywords := ExtractKeywords("")
	if len(keywords) != 0 {
		t.Errorf("Expected empty keywords for empty input, got %v", keywords)
	}
}

// TestExtractKeywords_NoKeywords verifies handling of input with no keywords
func TestExtractKeywords_NoKeywords(t *testing.T) {
	keywords := ExtractKeywords("This text has no matching keywords at all")
	if len(keywords) != 0 {
		t.Errorf("Expected empty keywords, got %v", keywords)
	}
}

// TestExtractKeywords_MixedContent verifies extraction from complex response
func TestExtractKeywords_MixedContent(t *testing.T) {
	response := `Check the file pkg/session/query.go and run go test.
	Use edit tool, not bash for this. Deploy with make build.`

	keywords := ExtractKeywords(response)

	expected := []string{"query.go", "go test", "edit", "bash", "make"}
	for _, exp := range expected {
		if !contains(keywords, exp) {
			t.Errorf("Expected %q in keywords: %v", exp, keywords)
		}
	}
}

// TestExtractKeywords_FilePathsOnly verifies extraction when only file paths present
func TestExtractKeywords_FilePathsOnly(t *testing.T) {
	keywords := ExtractKeywords("Update these files: main.go, test.py, config.json")
	expected := []string{"main.go", "test.py", "config.json"}

	for _, exp := range expected {
		if !contains(keywords, exp) {
			t.Errorf("Expected %q in keywords: %v", exp, keywords)
		}
	}

	// Should have exactly 3 keywords
	if len(keywords) != 3 {
		t.Errorf("Expected 3 keywords, got %d: %v", len(keywords), keywords)
	}
}

// TestExtractKeywords_ToolsOnly verifies extraction when only tools present
func TestExtractKeywords_ToolsOnly(t *testing.T) {
	keywords := ExtractKeywords("Use these tools: edit, bash, grep")
	expected := []string{"edit", "bash", "grep"}

	for _, exp := range expected {
		if !contains(keywords, exp) {
			t.Errorf("Expected %q in keywords: %v", exp, keywords)
		}
	}
}

// TestExtractKeywords_PathsAndTools verifies combined extraction
func TestExtractKeywords_PathsAndTools(t *testing.T) {
	keywords := ExtractKeywords("Edit main.go using the edit tool, then run bash")

	expected := []string{"edit", "bash", "main.go"}
	for _, exp := range expected {
		if !contains(keywords, exp) {
			t.Errorf("Expected %q in keywords: %v", exp, keywords)
		}
	}
}

// TestExtractKeywords_Deterministic verifies same input produces same output
func TestExtractKeywords_Deterministic(t *testing.T) {
	response := "Use edit and bash to modify query.go"

	first := ExtractKeywords(response)
	for i := 0; i < 9; i++ {
		result := ExtractKeywords(response)
		if len(result) != len(first) {
			t.Errorf("Extraction not deterministic: iteration %d got %v, expected %v",
				i, result, first)
			continue
		}

		// Check all elements match (order may vary)
		for _, kw := range first {
			if !contains(result, kw) {
				t.Errorf("Extraction not deterministic: missing %q in iteration %d", kw, i)
			}
		}
	}
}
