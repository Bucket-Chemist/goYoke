package integration

import (
	"strings"
	"testing"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/routing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEdgeCase_MalformedJSON_Input tests handling of garbage input
func TestEdgeCase_MalformedJSON_Input(t *testing.T) {
	// DetectFailure expects PostToolEvent, not raw JSON
	// This tests nil/empty event handling
	info := routing.DetectFailure(nil)
	assert.Nil(t, info, "Should handle nil event gracefully")
}

// TestEdgeCase_EmptyInput tests handling of empty event
func TestEdgeCase_EmptyInput(t *testing.T) {
	event := &routing.PostToolEvent{}
	info := routing.DetectFailure(event)
	// Empty event has nil response, should return nil
	assert.Nil(t, info, "Should handle empty event gracefully")
}

// TestEdgeCase_PartialJSON tests handling of partially filled event
func TestEdgeCase_PartialJSON(t *testing.T) {
	// Event with tool name but no response
	event := &routing.PostToolEvent{
		ToolName:   "Bash",
		ToolInput:  map[string]interface{}{"command": "test"},
		ToolResponse: nil,
		CapturedAt: time.Now().Unix(),
	}

	info := routing.DetectFailure(event)
	assert.Nil(t, info, "Should handle nil response gracefully")
}

// TestEdgeCase_ExtremelyLongErrorMessage tests handling of large error messages
func TestEdgeCase_ExtremelyLongErrorMessage(t *testing.T) {
	// Create 100KB error message
	largeError := strings.Repeat("ERROR: Something went wrong. ", 3500) // ~100KB

	event := &routing.PostToolEvent{
		ToolName:   "Bash",
		ToolInput:  map[string]interface{}{"command": "test"},
		ToolResponse: map[string]interface{}{
			"output": largeError,
		},
		CapturedAt: time.Now().Unix(),
	}

	info := routing.DetectFailure(event)
	require.NotNil(t, info, "Should detect failure in large output")
	assert.Equal(t, "generic_error", info.ErrorType, "Should detect error keyword")
	assert.Equal(t, "ERROR", info.ErrorMatch, "Should match error keyword")
}

// TestEdgeCase_UnicodeInFilePath tests handling of unicode paths
func TestEdgeCase_UnicodeInFilePath(t *testing.T) {
	unicodePath := "/home/user/файл.py" // Russian characters

	event := &routing.PostToolEvent{
		ToolName:   "Edit",
		ToolInput:  map[string]interface{}{"file_path": unicodePath},
		ToolResponse: map[string]interface{}{
			"success": false,
		},
		CapturedAt: time.Now().Unix(),
	}

	info := routing.DetectFailure(event)
	require.NotNil(t, info)
	assert.Equal(t, unicodePath, info.File, "Should preserve unicode in file path")
}

// TestEdgeCase_SpecialCharsInError tests handling of special characters
func TestEdgeCase_SpecialCharsInError(t *testing.T) {
	errorWithSpecialChars := `TypeError: "foo" cannot be used with 'bar'
Line breaks and "quotes" everywhere`

	event := &routing.PostToolEvent{
		ToolName:   "Bash",
		ToolInput:  map[string]interface{}{"command": "python test.py"},
		ToolResponse: map[string]interface{}{
			"output": errorWithSpecialChars,
		},
		CapturedAt: time.Now().Unix(),
	}

	info := routing.DetectFailure(event)
	require.NotNil(t, info)
	assert.Equal(t, "typeerror", info.ErrorType)
	assert.Contains(t, errorWithSpecialChars, info.ErrorMatch)
}

// TestEdgeCase_VeryOldTimestamp tests handling of timestamp from 1970
func TestEdgeCase_VeryOldTimestamp(t *testing.T) {
	oldTimestamp := int64(100) // Very early Unix epoch

	event := &routing.PostToolEvent{
		ToolName:   "Bash",
		ToolInput:  map[string]interface{}{"command": "test"},
		ToolResponse: map[string]interface{}{
			"exit_code": 1,
		},
		CapturedAt: oldTimestamp,
	}

	info := routing.DetectFailure(event)
	require.NotNil(t, info)
	assert.Equal(t, oldTimestamp, info.Timestamp, "Should preserve old timestamp")
}

// TestEdgeCase_FutureTimestamp tests handling of timestamp in future
func TestEdgeCase_FutureTimestamp(t *testing.T) {
	// Timestamp 1 day in the future
	futureTimestamp := time.Now().Add(24 * time.Hour).Unix()

	event := &routing.PostToolEvent{
		ToolName:   "Bash",
		ToolInput:  map[string]interface{}{"command": "test"},
		ToolResponse: map[string]interface{}{
			"exit_code": 1,
		},
		CapturedAt: futureTimestamp,
	}

	info := routing.DetectFailure(event)
	require.NotNil(t, info)
	assert.Equal(t, futureTimestamp, info.Timestamp, "Should accept future timestamp")
}

// TestEdgeCase_ZeroExitCode tests that exit code 0 is not treated as failure
func TestEdgeCase_ZeroExitCode(t *testing.T) {
	event := &routing.PostToolEvent{
		ToolName:   "Bash",
		ToolInput:  map[string]interface{}{"command": "echo success"},
		ToolResponse: map[string]interface{}{
			"exit_code": 0,
			"output":    "success",
		},
		CapturedAt: time.Now().Unix(),
	}

	info := routing.DetectFailure(event)
	assert.Nil(t, info, "Exit code 0 should not be treated as failure")
}

// TestEdgeCase_NegativeExitCode tests handling of negative exit codes
func TestEdgeCase_NegativeExitCode(t *testing.T) {
	event := &routing.PostToolEvent{
		ToolName:   "Bash",
		ToolInput:  map[string]interface{}{"command": "test"},
		ToolResponse: map[string]interface{}{
			"exit_code": -1,
		},
		CapturedAt: time.Now().Unix(),
	}

	info := routing.DetectFailure(event)
	require.NotNil(t, info, "Should detect negative exit code as failure")
	assert.Equal(t, "exit_code_-1", info.ErrorType)
	assert.Equal(t, -1, info.ExitCode)
}

// TestEdgeCase_MultipleErrorTypesInOutput tests priority when multiple errors present
func TestEdgeCase_MultipleErrorTypesInOutput(t *testing.T) {
	// Output with multiple Python error types
	multiError := `
Traceback (most recent call last):
  File "test.py", line 10, in main
    result = process()
  File "test.py", line 5, in process
    raise ValueError("invalid")
ValueError: invalid
During handling of above exception, another exception occurred:
TypeError: cannot concatenate
`

	event := &routing.PostToolEvent{
		ToolName:   "Bash",
		ToolInput:  map[string]interface{}{"command": "python test.py"},
		ToolResponse: map[string]interface{}{
			"output": multiError,
		},
		CapturedAt: time.Now().Unix(),
	}

	info := routing.DetectFailure(event)
	require.NotNil(t, info)
	// Should match first Python error found (ValueError appears first)
	assert.Equal(t, "valueerror", info.ErrorType)
}

// TestEdgeCase_EmptyFilePath tests extraction when no file path available
func TestEdgeCase_EmptyFilePath(t *testing.T) {
	event := &routing.PostToolEvent{
		ToolName:   "Bash",
		ToolInput:  map[string]interface{}{}, // No command or file_path
		ToolResponse: map[string]interface{}{
			"exit_code": 1,
		},
		CapturedAt: time.Now().Unix(),
	}

	info := routing.DetectFailure(event)
	require.NotNil(t, info)
	assert.Equal(t, "unknown", info.File, "Should use 'unknown' when path unavailable")
}

// TestEdgeCase_ComplexCommandParsing tests file path extraction from complex commands
func TestEdgeCase_ComplexCommandParsing(t *testing.T) {
	testCases := []struct {
		name     string
		command  string
		expected string
	}{
		{
			name:     "python_with_args",
			command:  "python -m pytest tests/test_main.py -v --cov",
			expected: "tests/test_main.py", // First path-like (has /)
		},
		{
			name:     "go_test_with_flags",
			command:  "go test ./pkg/routing -v -race",
			expected: "./pkg/routing",
		},
		{
			name:     "pipe_command",
			command:  "cat file.txt | grep error",
			expected: "file.txt",
		},
		{
			name:     "quoted_path",
			command:  `python "path with spaces/file.py"`,
			expected: `spaces/file.py"`, // Naive whitespace splitting: "path with spaces/file.py" becomes ["\"path", "with", "spaces/file.py\""]
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			event := &routing.PostToolEvent{
				ToolName:   "Bash",
				ToolInput:  map[string]interface{}{"command": tc.command},
				ToolResponse: map[string]interface{}{
					"exit_code": 1,
				},
				CapturedAt: time.Now().Unix(),
			}

			info := routing.DetectFailure(event)
			require.NotNil(t, info)
			assert.Equal(t, tc.expected, info.File)
		})
	}
}

// TestEdgeCase_MixedCaseErrorKeywords tests case-insensitive error detection
func TestEdgeCase_MixedCaseErrorKeywords(t *testing.T) {
	testCases := []struct {
		output   string
		expected string
	}{
		{"TYPEERROR: something", "typeerror"},
		{"ValueError: bad", "valueerror"},
		{"syntax ERROR detected", "generic_error"},
		{"Build FAILED", "generic_failed"},
		{"ImportError: missing", "importerror"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			event := &routing.PostToolEvent{
				ToolName:   "Bash",
				ToolInput:  map[string]interface{}{"command": "test"},
				ToolResponse: map[string]interface{}{
					"output": tc.output,
				},
				CapturedAt: time.Now().Unix(),
			}

			info := routing.DetectFailure(event)
			require.NotNil(t, info, "Should detect error in: %s", tc.output)
			assert.Equal(t, tc.expected, info.ErrorType)
		})
	}
}

// TestEdgeCase_ResponseFieldVariations tests different response field names
func TestEdgeCase_ResponseFieldVariations(t *testing.T) {
	testCases := []struct {
		name     string
		response map[string]interface{}
		hasError bool
	}{
		{
			name:     "output_field",
			response: map[string]interface{}{"output": "error occurred"},
			hasError: true,
		},
		{
			name:     "error_field",
			response: map[string]interface{}{"error": "something failed"},
			hasError: true,
		},
		{
			name:     "both_fields",
			response: map[string]interface{}{"output": "success", "error": ""},
			hasError: false, // Output takes priority and has no error keywords
		},
		{
			name:     "neither_field",
			response: map[string]interface{}{"result": "done"},
			hasError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			event := &routing.PostToolEvent{
				ToolName:     "Bash",
				ToolInput:    map[string]interface{}{"command": "test"},
				ToolResponse: tc.response,
				CapturedAt:   time.Now().Unix(),
			}

			info := routing.DetectFailure(event)
			if tc.hasError {
				assert.NotNil(t, info, "Should detect error")
			} else {
				assert.Nil(t, info, "Should not detect error")
			}
		})
	}
}
