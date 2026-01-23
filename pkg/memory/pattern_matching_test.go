package memory

import (
	"os"
	"path/filepath"
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
