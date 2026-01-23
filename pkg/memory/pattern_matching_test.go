package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestYAML creates a temporary YAML file with the given content
func createTestYAML(t *testing.T, dir, content string) string {
	t.Helper()
	yamlPath := filepath.Join(dir, "sharp-edges.yaml")
	err := os.WriteFile(yamlPath, []byte(content), 0644)
	require.NoError(t, err, "Failed to create test YAML file")
	return yamlPath
}

// TestLoadSharpEdgesIndex_SingleDirectory tests loading from a single agent directory
func TestLoadSharpEdgesIndex_SingleDirectory(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "sharp-edges-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create valid YAML content
	yamlContent := `- id: "test-001"
  error_type: "TypeError"
  file_pattern: "*.py"
  keywords: ["type assertion", "bool"]
  description: "Type assertion on already-typed field"
  solution: "Use direct field access instead of type assertion"

- id: "test-002"
  error_type: "nil_pointer"
  file_pattern: "*.go"
  keywords: ["map access", "nil"]
  description: "Accessing map without checking if key exists"
  solution: "Use two-value form: value, ok := map[key]"
`
	createTestYAML(t, tmpDir, yamlContent)

	// Load index
	index, err := LoadSharpEdgesIndex([]string{tmpDir})
	require.NoError(t, err)
	assert.NotNil(t, index)

	// Verify All contains both templates
	assert.Len(t, index.All, 2, "Expected 2 templates in All")

	// Verify ByErrorType index
	assert.Len(t, index.ByErrorType["TypeError"], 1)
	assert.Equal(t, "test-001", index.ByErrorType["TypeError"][0].ID)

	assert.Len(t, index.ByErrorType["nil_pointer"], 1)
	assert.Equal(t, "test-002", index.ByErrorType["nil_pointer"][0].ID)

	// Verify ByKeyword index
	assert.Len(t, index.ByKeyword["type assertion"], 1)
	assert.Equal(t, "test-001", index.ByKeyword["type assertion"][0].ID)

	assert.Len(t, index.ByKeyword["map access"], 1)
	assert.Equal(t, "test-002", index.ByKeyword["map access"][0].ID)

	// Verify Source field is populated
	for _, tmpl := range index.All {
		assert.Contains(t, tmpl.Source, "sharp-edges.yaml")
	}
}

// TestLoadSharpEdgesIndex_MultipleDirectories tests loading from multiple agent directories
func TestLoadSharpEdgesIndex_MultipleDirectories(t *testing.T) {
	// Create two temporary directories
	tmpDir1, err := os.MkdirTemp("", "sharp-edges-test-1-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir1)

	tmpDir2, err := os.MkdirTemp("", "sharp-edges-test-2-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir2)

	// Create YAML in first directory
	yamlContent1 := `- id: "python-001"
  error_type: "TypeError"
  file_pattern: "*.py"
  keywords: ["python", "type"]
  description: "Python type error"
  solution: "Check types"
`
	createTestYAML(t, tmpDir1, yamlContent1)

	// Create YAML in second directory
	yamlContent2 := `- id: "go-001"
  error_type: "nil_pointer"
  file_pattern: "*.go"
  keywords: ["go", "nil"]
  description: "Go nil pointer"
  solution: "Check for nil"

- id: "go-002"
  error_type: "race_condition"
  file_pattern: "*.go"
  keywords: ["go", "concurrent"]
  description: "Race condition detected"
  solution: "Use mutex or channels"
`
	createTestYAML(t, tmpDir2, yamlContent2)

	// Load index from both directories
	index, err := LoadSharpEdgesIndex([]string{tmpDir1, tmpDir2})
	require.NoError(t, err)
	assert.NotNil(t, index)

	// Verify All contains templates from both directories
	assert.Len(t, index.All, 3, "Expected 3 templates total")

	// Verify templates from first directory
	assert.Len(t, index.ByErrorType["TypeError"], 1)
	assert.Equal(t, "python-001", index.ByErrorType["TypeError"][0].ID)

	// Verify templates from second directory
	assert.Len(t, index.ByErrorType["nil_pointer"], 1)
	assert.Equal(t, "go-001", index.ByErrorType["nil_pointer"][0].ID)

	assert.Len(t, index.ByErrorType["race_condition"], 1)
	assert.Equal(t, "go-002", index.ByErrorType["race_condition"][0].ID)

	// Verify keyword index includes entries from both directories
	assert.Len(t, index.ByKeyword["python"], 1)
	assert.Len(t, index.ByKeyword["go"], 2)
}

// TestLoadSharpEdgesIndex_MissingFile tests graceful handling of missing sharp-edges.yaml
func TestLoadSharpEdgesIndex_MissingFile(t *testing.T) {
	// Create temporary directory without sharp-edges.yaml
	tmpDir, err := os.MkdirTemp("", "sharp-edges-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Load index - should succeed with empty index
	index, err := LoadSharpEdgesIndex([]string{tmpDir})
	require.NoError(t, err)
	assert.NotNil(t, index)

	// Verify empty indexes
	assert.Empty(t, index.All)
	assert.Empty(t, index.ByErrorType)
	assert.Empty(t, index.ByKeyword)
}

// TestLoadSharpEdgesIndex_MalformedYAML tests graceful handling of malformed YAML
func TestLoadSharpEdgesIndex_MalformedYAML(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "sharp-edges-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create malformed YAML (invalid indentation)
	malformedYAML := `- id: "test-001"
  error_type: "TypeError"
 keywords: ["invalid indent"]  # Wrong indentation
  description: "This is malformed"
`
	createTestYAML(t, tmpDir, malformedYAML)

	// Capture stderr to verify warning is logged
	// (In production, this would write to stderr)
	index, err := LoadSharpEdgesIndex([]string{tmpDir})
	require.NoError(t, err)
	assert.NotNil(t, index)

	// Verify empty indexes (malformed file skipped)
	assert.Empty(t, index.All)
	assert.Empty(t, index.ByErrorType)
	assert.Empty(t, index.ByKeyword)
}

// TestLoadSharpEdgesIndex_MixedValidInvalid tests handling of mixed valid/invalid directories
func TestLoadSharpEdgesIndex_MixedValidInvalid(t *testing.T) {
	// Create three directories
	validDir, err := os.MkdirTemp("", "sharp-edges-valid-*")
	require.NoError(t, err)
	defer os.RemoveAll(validDir)

	missingDir, err := os.MkdirTemp("", "sharp-edges-missing-*")
	require.NoError(t, err)
	defer os.RemoveAll(missingDir)

	malformedDir, err := os.MkdirTemp("", "sharp-edges-malformed-*")
	require.NoError(t, err)
	defer os.RemoveAll(malformedDir)

	// Create valid YAML
	validYAML := `- id: "valid-001"
  error_type: "ValidError"
  file_pattern: "*.test"
  keywords: ["valid"]
  description: "Valid template"
  solution: "This should be loaded"
`
	createTestYAML(t, validDir, validYAML)

	// Create malformed YAML
	malformedYAML := `{invalid yaml content: [unclosed`
	createTestYAML(t, malformedDir, malformedYAML)

	// Load from all three directories
	index, err := LoadSharpEdgesIndex([]string{validDir, missingDir, malformedDir})
	require.NoError(t, err)
	assert.NotNil(t, index)

	// Verify only valid template is loaded
	assert.Len(t, index.All, 1)
	assert.Equal(t, "valid-001", index.All[0].ID)
	assert.Len(t, index.ByErrorType["ValidError"], 1)
	assert.Len(t, index.ByKeyword["valid"], 1)
}

// TestLoadSharpEdgesIndex_EmptyDirectoryList tests handling of empty directory list
func TestLoadSharpEdgesIndex_EmptyDirectoryList(t *testing.T) {
	index, err := LoadSharpEdgesIndex([]string{})
	require.NoError(t, err)
	assert.NotNil(t, index)

	// Verify empty indexes
	assert.Empty(t, index.All)
	assert.Empty(t, index.ByErrorType)
	assert.Empty(t, index.ByKeyword)
}

// TestLoadSharpEdgesIndex_MultipleKeywords tests indexing with multiple keywords
func TestLoadSharpEdgesIndex_MultipleKeywords(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sharp-edges-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create template with multiple keywords
	yamlContent := `- id: "multi-keyword-001"
  error_type: "ComplexError"
  file_pattern: "*.test"
  keywords: ["keyword1", "keyword2", "keyword3"]
  description: "Template with multiple keywords"
  solution: "Test keyword indexing"
`
	createTestYAML(t, tmpDir, yamlContent)

	index, err := LoadSharpEdgesIndex([]string{tmpDir})
	require.NoError(t, err)

	// Verify each keyword is indexed
	for _, keyword := range []string{"keyword1", "keyword2", "keyword3"} {
		assert.Len(t, index.ByKeyword[keyword], 1, "Keyword %s should have 1 template", keyword)
		assert.Equal(t, "multi-keyword-001", index.ByKeyword[keyword][0].ID)
	}
}

// TestLoadSharpEdgesIndex_DuplicateErrorTypes tests handling of multiple templates with same error type
func TestLoadSharpEdgesIndex_DuplicateErrorTypes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sharp-edges-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create multiple templates with same error type
	yamlContent := `- id: "dup-001"
  error_type: "TypeError"
  file_pattern: "*.py"
  keywords: ["type1"]
  description: "First type error"
  solution: "Solution 1"

- id: "dup-002"
  error_type: "TypeError"
  file_pattern: "*.go"
  keywords: ["type2"]
  description: "Second type error"
  solution: "Solution 2"
`
	createTestYAML(t, tmpDir, yamlContent)

	index, err := LoadSharpEdgesIndex([]string{tmpDir})
	require.NoError(t, err)

	// Verify both templates are indexed under same error type
	assert.Len(t, index.ByErrorType["TypeError"], 2)
	ids := []string{
		index.ByErrorType["TypeError"][0].ID,
		index.ByErrorType["TypeError"][1].ID,
	}
	assert.Contains(t, ids, "dup-001")
	assert.Contains(t, ids, "dup-002")
}

// TestLoadSharpEdgesIndex_EmptyYAML tests handling of empty YAML file
func TestLoadSharpEdgesIndex_EmptyYAML(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sharp-edges-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create empty YAML file
	createTestYAML(t, tmpDir, "")

	index, err := LoadSharpEdgesIndex([]string{tmpDir})
	require.NoError(t, err)
	assert.NotNil(t, index)

	// Verify empty indexes (empty YAML is valid, just contains no templates)
	assert.Empty(t, index.All)
	assert.Empty(t, index.ByErrorType)
	assert.Empty(t, index.ByKeyword)
}

// TestSharpEdgeTemplate_AllFields tests that all fields are correctly parsed
func TestSharpEdgeTemplate_AllFields(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sharp-edges-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create template with all fields populated
	yamlContent := `- id: "full-001"
  error_type: "FullError"
  file_pattern: "*.full"
  keywords: ["full", "complete"]
  description: "Complete description with all fields"
  solution: "Comprehensive solution explanation"
`
	yamlPath := createTestYAML(t, tmpDir, yamlContent)

	index, err := LoadSharpEdgesIndex([]string{tmpDir})
	require.NoError(t, err)
	require.Len(t, index.All, 1)

	tmpl := index.All[0]
	assert.Equal(t, "full-001", tmpl.ID)
	assert.Equal(t, "FullError", tmpl.ErrorType)
	assert.Equal(t, "*.full", tmpl.FilePattern)
	assert.Equal(t, []string{"full", "complete"}, tmpl.Keywords)
	assert.Equal(t, "Complete description with all fields", tmpl.Description)
	assert.Equal(t, "Comprehensive solution explanation", tmpl.Solution)
	assert.Equal(t, yamlPath, tmpl.Source)
}

// TestLoadSharpEdgesIndex_Coverage tests edge cases for coverage
func TestLoadSharpEdgesIndex_Coverage(t *testing.T) {
	t.Run("nonexistent directory", func(t *testing.T) {
		// Directory doesn't exist at all
		index, err := LoadSharpEdgesIndex([]string{"/nonexistent/path/to/nowhere"})
		require.NoError(t, err)
		assert.Empty(t, index.All)
	})

	t.Run("empty keywords list", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "sharp-edges-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Template with empty keywords array
		yamlContent := `- id: "no-keywords"
  error_type: "TestError"
  file_pattern: "*.test"
  keywords: []
  description: "No keywords"
  solution: "Test solution"
`
		createTestYAML(t, tmpDir, yamlContent)

		index, err := LoadSharpEdgesIndex([]string{tmpDir})
		require.NoError(t, err)
		assert.Len(t, index.All, 1)
		assert.Empty(t, index.All[0].Keywords)
		// ByKeyword should not have any entries for this template
		assert.Empty(t, index.ByKeyword)
	})

	t.Run("unreadable file permissions", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping permission test when running as root")
		}

		tmpDir, err := os.MkdirTemp("", "sharp-edges-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		yamlPath := createTestYAML(t, tmpDir, "test")
		// Make file unreadable
		err = os.Chmod(yamlPath, 0000)
		require.NoError(t, err)

		// Should handle gracefully (warning to stderr, continue)
		index, err := LoadSharpEdgesIndex([]string{tmpDir})
		require.NoError(t, err)
		assert.Empty(t, index.All) // File couldn't be read
	})
}

// TestFindSimilar_ExactErrorTypeMatch tests exact error type matching
func TestFindSimilar_ExactErrorTypeMatch(t *testing.T) {
	// Create index with one template
	index := &SharpEdgeIndex{
		ByErrorType: map[string][]SharpEdgeTemplate{
			"TypeError": {
				{
					ID:          "test-001",
					ErrorType:   "TypeError",
					FilePattern: "*.py",
					Keywords:    []string{"type", "assertion"},
					Description: "Type assertion error",
					Solution:    "Check types before assertion",
				},
			},
		},
		ByKeyword: map[string][]SharpEdgeTemplate{
			"type": {
				{
					ID:          "test-001",
					ErrorType:   "TypeError",
					FilePattern: "*.py",
					Keywords:    []string{"type", "assertion"},
					Description: "Type assertion error",
					Solution:    "Check types before assertion",
				},
			},
			"assertion": {
				{
					ID:          "test-001",
					ErrorType:   "TypeError",
					FilePattern: "*.py",
					Keywords:    []string{"type", "assertion"},
					Description: "Type assertion error",
					Solution:    "Check types before assertion",
				},
			},
		},
	}

	edge := &SharpEdge{
		ErrorType:    "TypeError",
		File:         "test.py",
		ErrorMessage: "invalid type assertion",
	}

	matches := FindSimilar(edge, index)

	// Should match with error_type (5) + file_pattern (3) + 2 keywords (4) = 12 points
	require.Len(t, matches, 1)
	assert.Equal(t, "test-001", matches[0].Template.ID)
	assert.Equal(t, 12, matches[0].Score) // error_type + file_pattern + both keywords
	assert.Contains(t, matches[0].MatchedOn, "error_type")
	assert.Contains(t, matches[0].MatchedOn, "file_pattern")
	assert.Contains(t, matches[0].MatchedOn, "keyword:type")
	assert.Contains(t, matches[0].MatchedOn, "keyword:assertion")
}

// TestFindSimilar_ErrorTypeWithFilePattern tests error type + file pattern match
func TestFindSimilar_ErrorTypeWithFilePattern(t *testing.T) {
	index := &SharpEdgeIndex{
		ByErrorType: map[string][]SharpEdgeTemplate{
			"nil_pointer": {
				{
					ID:          "go-001",
					ErrorType:   "nil_pointer",
					FilePattern: "pkg/routing/*.go",
					Keywords:    []string{"nil", "map"},
					Description: "Nil pointer dereference",
					Solution:    "Check for nil before access",
				},
			},
		},
		ByKeyword: map[string][]SharpEdgeTemplate{
			"nil": {
				{
					ID:          "go-001",
					ErrorType:   "nil_pointer",
					FilePattern: "pkg/routing/*.go",
					Keywords:    []string{"nil", "map"},
					Description: "Nil pointer dereference",
					Solution:    "Check for nil before access",
				},
			},
		},
	}

	edge := &SharpEdge{
		ErrorType:    "nil_pointer",
		File:         "pkg/routing/task_validation.go",
		ErrorMessage: "nil pointer dereference",
	}

	matches := FindSimilar(edge, index)

	// Should match with error_type (5) + file_pattern (3) + keyword:nil (2) = 10 points
	require.Len(t, matches, 1)
	assert.Equal(t, "go-001", matches[0].Template.ID)
	assert.Equal(t, 10, matches[0].Score) // error_type + file_pattern + 1 keyword
	assert.Contains(t, matches[0].MatchedOn, "error_type")
	assert.Contains(t, matches[0].MatchedOn, "file_pattern")
	assert.Contains(t, matches[0].MatchedOn, "keyword:nil")
}

// TestFindSimilar_ErrorTypeWithFilePatternAndKeywords tests full match (highest score)
func TestFindSimilar_ErrorTypeWithFilePatternAndKeywords(t *testing.T) {
	index := &SharpEdgeIndex{
		ByErrorType: map[string][]SharpEdgeTemplate{
			"TypeError": {
				{
					ID:          "full-match",
					ErrorType:   "TypeError",
					FilePattern: "pkg/routing/*.go",
					Keywords:    []string{"type", "assertion", "bool", "interface"},
					Description: "Type assertion on typed field",
					Solution:    "Use direct field access",
				},
			},
		},
		ByKeyword: map[string][]SharpEdgeTemplate{
			"type": {
				{
					ID:          "full-match",
					ErrorType:   "TypeError",
					FilePattern: "pkg/routing/*.go",
					Keywords:    []string{"type", "assertion", "bool", "interface"},
					Description: "Type assertion on typed field",
					Solution:    "Use direct field access",
				},
			},
		},
	}

	edge := &SharpEdge{
		ErrorType:    "TypeError",
		File:         "pkg/routing/task_validation.go",
		ErrorMessage: "invalid type assertion: field is bool, not interface{}",
	}

	matches := FindSimilar(edge, index)

	// Should match: error_type (5) + file_pattern (3) + 4 keywords × 2 = 16 points
	require.Len(t, matches, 1)
	assert.Equal(t, "full-match", matches[0].Template.ID)
	assert.Equal(t, 16, matches[0].Score)
	assert.Contains(t, matches[0].MatchedOn, "error_type")
	assert.Contains(t, matches[0].MatchedOn, "file_pattern")
	assert.Contains(t, matches[0].MatchedOn, "keyword:type")
	assert.Contains(t, matches[0].MatchedOn, "keyword:assertion")
	assert.Contains(t, matches[0].MatchedOn, "keyword:bool")
	assert.Contains(t, matches[0].MatchedOn, "keyword:interface")
}

// TestFindSimilar_KeywordMatchDifferentErrorType tests keyword match with different error type
func TestFindSimilar_KeywordMatchDifferentErrorType(t *testing.T) {
	index := &SharpEdgeIndex{
		ByErrorType: map[string][]SharpEdgeTemplate{
			"ValueError": {
				{
					ID:          "value-001",
					ErrorType:   "ValueError",
					FilePattern: "*.py",
					Keywords:    []string{"invalid", "format"},
					Description: "Invalid value format",
					Solution:    "Check value format",
				},
			},
		},
		ByKeyword: map[string][]SharpEdgeTemplate{
			"invalid": {
				{
					ID:          "value-001",
					ErrorType:   "ValueError",
					FilePattern: "*.py",
					Keywords:    []string{"invalid", "format"},
					Description: "Invalid value format",
					Solution:    "Check value format",
				},
			},
			"format": {
				{
					ID:          "value-001",
					ErrorType:   "ValueError",
					FilePattern: "*.py",
					Keywords:    []string{"invalid", "format"},
					Description: "Invalid value format",
					Solution:    "Check value format",
				},
			},
		},
	}

	edge := &SharpEdge{
		ErrorType:    "TypeError", // Different error type
		File:         "test.py",
		ErrorMessage: "invalid input format",
	}

	matches := FindSimilar(edge, index)

	// Should match via keywords: keyword (2) + keyword (2) + file_pattern (3) = 7 points
	require.Len(t, matches, 1)
	assert.Equal(t, "value-001", matches[0].Template.ID)
	assert.GreaterOrEqual(t, matches[0].Score, 5) // Must meet threshold
	assert.Contains(t, matches[0].MatchedOn, "file_pattern")
	// Should have at least one keyword match
	hasKeywordMatch := false
	for _, m := range matches[0].MatchedOn {
		if strings.HasPrefix(m, "keyword:") {
			hasKeywordMatch = true
			break
		}
	}
	assert.True(t, hasKeywordMatch, "Expected at least one keyword match")
}

// TestFindSimilar_NoMatches tests scenario with no good matches (score below threshold)
func TestFindSimilar_NoMatches(t *testing.T) {
	index := &SharpEdgeIndex{
		ByErrorType: map[string][]SharpEdgeTemplate{
			"ValueError": {
				{
					ID:          "value-001",
					ErrorType:   "ValueError",
					FilePattern: "*.py",
					Keywords:    []string{"value", "format"},
					Description: "Value error",
					Solution:    "Fix value",
				},
			},
		},
		ByKeyword: map[string][]SharpEdgeTemplate{
			"value": {
				{
					ID:          "value-001",
					ErrorType:   "ValueError",
					FilePattern: "*.py",
					Keywords:    []string{"value", "format"},
					Description: "Value error",
					Solution:    "Fix value",
				},
			},
		},
	}

	edge := &SharpEdge{
		ErrorType:    "SyntaxError", // Different error type
		File:         "test.go",      // Different file pattern
		ErrorMessage: "unexpected token", // No matching keywords
	}

	matches := FindSimilar(edge, index)

	// No matches should be returned (all scores below threshold)
	assert.Empty(t, matches)
}

// TestFindSimilar_MultipleMatchesTop3 tests that only top 3 matches are returned
func TestFindSimilar_MultipleMatchesTop3(t *testing.T) {
	index := &SharpEdgeIndex{
		ByErrorType: map[string][]SharpEdgeTemplate{
			"TypeError": {
				{
					ID:          "type-001",
					ErrorType:   "TypeError",
					FilePattern: "*.go",
					Keywords:    []string{"type", "assertion", "bool"},
					Description: "Type error 1",
					Solution:    "Solution 1",
				},
				{
					ID:          "type-002",
					ErrorType:   "TypeError",
					FilePattern: "*.go",
					Keywords:    []string{"type", "assertion"},
					Description: "Type error 2",
					Solution:    "Solution 2",
				},
				{
					ID:          "type-003",
					ErrorType:   "TypeError",
					FilePattern: "*.py",
					Keywords:    []string{"type"},
					Description: "Type error 3",
					Solution:    "Solution 3",
				},
				{
					ID:          "type-004",
					ErrorType:   "TypeError",
					FilePattern: "*.go",
					Keywords:    []string{},
					Description: "Type error 4",
					Solution:    "Solution 4",
				},
				{
					ID:          "type-005",
					ErrorType:   "TypeError",
					FilePattern: "*.js",
					Keywords:    []string{},
					Description: "Type error 5",
					Solution:    "Solution 5",
				},
			},
		},
		ByKeyword: make(map[string][]SharpEdgeTemplate),
	}

	edge := &SharpEdge{
		ErrorType:    "TypeError",
		File:         "test.go",
		ErrorMessage: "type assertion on bool field",
	}

	matches := FindSimilar(edge, index)

	// Should return at most 3 matches
	assert.LessOrEqual(t, len(matches), 3)
	assert.GreaterOrEqual(t, len(matches), 1) // Should have at least one match
}

// TestFindSimilar_ScoreRanking tests that matches are correctly ranked by score
func TestFindSimilar_ScoreRanking(t *testing.T) {
	index := &SharpEdgeIndex{
		ByErrorType: map[string][]SharpEdgeTemplate{
			"TypeError": {
				{
					ID:          "low-score",
					ErrorType:   "TypeError",
					FilePattern: "*.py", // Won't match
					Keywords:    []string{},
					Description: "Low score",
					Solution:    "Solution",
				},
				{
					ID:          "medium-score",
					ErrorType:   "TypeError",
					FilePattern: "*.go", // Matches
					Keywords:    []string{},
					Description: "Medium score",
					Solution:    "Solution",
				},
				{
					ID:          "high-score",
					ErrorType:   "TypeError",
					FilePattern: "*.go", // Matches
					Keywords:    []string{"type", "assertion"}, // Both match
					Description: "High score",
					Solution:    "Solution",
				},
			},
		},
		ByKeyword: map[string][]SharpEdgeTemplate{
			"type": {
				{
					ID:          "high-score",
					ErrorType:   "TypeError",
					FilePattern: "*.go",
					Keywords:    []string{"type", "assertion"},
					Description: "High score",
					Solution:    "Solution",
				},
			},
			"assertion": {
				{
					ID:          "high-score",
					ErrorType:   "TypeError",
					FilePattern: "*.go",
					Keywords:    []string{"type", "assertion"},
					Description: "High score",
					Solution:    "Solution",
				},
			},
		},
	}

	edge := &SharpEdge{
		ErrorType:    "TypeError",
		File:         "test.go",
		ErrorMessage: "type assertion failed",
	}

	matches := FindSimilar(edge, index)

	// Verify scores are in descending order
	require.GreaterOrEqual(t, len(matches), 2)
	for i := 0; i < len(matches)-1; i++ {
		assert.GreaterOrEqual(t, matches[i].Score, matches[i+1].Score,
			"Match %d (score=%d) should have higher or equal score than match %d (score=%d)",
			i, matches[i].Score, i+1, matches[i+1].Score)
	}

	// High score should be first
	assert.Equal(t, "high-score", matches[0].Template.ID)
	assert.Equal(t, 12, matches[0].Score) // error_type(5) + file_pattern(3) + 2 keywords(4) = 12
}

// TestFindSimilar_EmptyIndex tests handling of empty index
func TestFindSimilar_EmptyIndex(t *testing.T) {
	index := &SharpEdgeIndex{
		ByErrorType: make(map[string][]SharpEdgeTemplate),
		ByKeyword:   make(map[string][]SharpEdgeTemplate),
	}

	edge := &SharpEdge{
		ErrorType:    "TypeError",
		File:         "test.go",
		ErrorMessage: "some error",
	}

	matches := FindSimilar(edge, index)

	// Should return empty slice, not nil
	assert.NotNil(t, matches)
	assert.Empty(t, matches)
}

// TestFindSimilar_CaseInsensitiveKeywords tests case-insensitive keyword matching
func TestFindSimilar_CaseInsensitiveKeywords(t *testing.T) {
	index := &SharpEdgeIndex{
		ByErrorType: map[string][]SharpEdgeTemplate{
			"TypeError": {
				{
					ID:          "case-test",
					ErrorType:   "TypeError",
					FilePattern: "*.go",
					Keywords:    []string{"Type", "Assertion"}, // Mixed case
					Description: "Case test",
					Solution:    "Solution",
				},
			},
		},
		ByKeyword: map[string][]SharpEdgeTemplate{
			"Type": {
				{
					ID:          "case-test",
					ErrorType:   "TypeError",
					FilePattern: "*.go",
					Keywords:    []string{"Type", "Assertion"},
					Description: "Case test",
					Solution:    "Solution",
				},
			},
		},
	}

	edge := &SharpEdge{
		ErrorType:    "TypeError",
		File:         "test.go",
		ErrorMessage: "type assertion failed", // Lowercase
	}

	matches := FindSimilar(edge, index)

	// Should match despite case differences
	require.Len(t, matches, 1)
	assert.Contains(t, matches[0].MatchedOn, "keyword:Type")
	assert.Contains(t, matches[0].MatchedOn, "keyword:Assertion")
}

// TestSharpEdgeTemplate_GetID tests the GetID method
func TestSharpEdgeTemplate_GetID(t *testing.T) {
	tests := []struct {
		name     string
		template SharpEdgeTemplate
		want     string
	}{
		{
			name: "ID field present",
			template: SharpEdgeTemplate{
				ID:   "test-001",
				Name: "Test Name",
			},
			want: "test-001",
		},
		{
			name: "Only Name field present (python-pro style)",
			template: SharpEdgeTemplate{
				Name: "Python Error",
			},
			want: "Python Error",
		},
		{
			name: "Both fields present - ID takes precedence",
			template: SharpEdgeTemplate{
				ID:   "go-001",
				Name: "Go Error",
			},
			want: "go-001",
		},
		{
			name: "Neither field present",
			template: SharpEdgeTemplate{
				ErrorType: "TestError",
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.template.GetID()
			if got != tt.want {
				t.Errorf("GetID() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestSharpEdgeTemplate_GetSolution tests the GetSolution method
func TestSharpEdgeTemplate_GetSolution(t *testing.T) {
	tests := []struct {
		name     string
		template SharpEdgeTemplate
		want     string
	}{
		{
			name: "Solution field present",
			template: SharpEdgeTemplate{
				Solution:   "Use direct field access",
				Mitigation: "Alternative mitigation",
			},
			want: "Use direct field access",
		},
		{
			name: "Only Mitigation field present (python-pro style)",
			template: SharpEdgeTemplate{
				Mitigation: "Check for nil pointer",
			},
			want: "Check for nil pointer",
		},
		{
			name: "Both fields present - Solution takes precedence",
			template: SharpEdgeTemplate{
				Solution:   "Primary solution",
				Mitigation: "Secondary mitigation",
			},
			want: "Primary solution",
		},
		{
			name: "Neither field present",
			template: SharpEdgeTemplate{
				ErrorType: "TestError",
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.template.GetSolution()
			if got != tt.want {
				t.Errorf("GetSolution() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestFindSimilar_FilePatternMatching tests glob pattern matching behavior
func TestFindSimilar_FilePatternMatching(t *testing.T) {
	tests := []struct {
		name            string
		filePattern     string
		edgeFile        string
		shouldMatchFile bool
	}{
		{
			name:            "exact extension match",
			filePattern:     "*.go",
			edgeFile:        "test.go",
			shouldMatchFile: true,
		},
		{
			name:            "directory glob match",
			filePattern:     "pkg/routing/*.go",
			edgeFile:        "pkg/routing/validator.go",
			shouldMatchFile: true,
		},
		{
			name:            "no match different extension",
			filePattern:     "*.py",
			edgeFile:        "test.go",
			shouldMatchFile: false,
		},
		{
			name:            "no match different directory",
			filePattern:     "pkg/routing/*.go",
			edgeFile:        "pkg/memory/validator.go",
			shouldMatchFile: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			index := &SharpEdgeIndex{
				ByErrorType: map[string][]SharpEdgeTemplate{
					"TestError": {
						{
							ID:          "pattern-test",
							ErrorType:   "TestError",
							FilePattern: tt.filePattern,
							Keywords:    []string{},
							Description: "Pattern test",
							Solution:    "Solution",
						},
					},
				},
				ByKeyword: make(map[string][]SharpEdgeTemplate),
			}

			edge := &SharpEdge{
				ErrorType:    "TestError",
				File:         tt.edgeFile,
				ErrorMessage: "test error",
			}

			matches := FindSimilar(edge, index)

			require.Len(t, matches, 1)
			if tt.shouldMatchFile {
				assert.Contains(t, matches[0].MatchedOn, "file_pattern",
					"Expected file_pattern match for %s with pattern %s", tt.edgeFile, tt.filePattern)
				assert.Equal(t, 8, matches[0].Score) // error_type(5) + file_pattern(3)
			} else {
				assert.NotContains(t, matches[0].MatchedOn, "file_pattern",
					"Expected no file_pattern match for %s with pattern %s", tt.edgeFile, tt.filePattern)
				assert.Equal(t, 5, matches[0].Score) // error_type(5) only
			}
		})
	}
}
