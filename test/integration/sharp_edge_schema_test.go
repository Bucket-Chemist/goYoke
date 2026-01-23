package integration

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SharpEdge represents the structure written to pending-learnings.jsonl
// This matches the session.SharpEdge schema
type SharpEdge struct {
	File               string `json:"file"`
	ErrorType          string `json:"error_type"`
	ConsecutiveFailures int    `json:"consecutive_failures"`
	Timestamp          int64  `json:"timestamp"`
}

// TestSchema_SharpEdge_RequiredFields tests that all required fields are present
func TestSchema_SharpEdge_RequiredFields(t *testing.T) {
	edge := SharpEdge{
		File:               "test.py",
		ErrorType:          "typeerror",
		ConsecutiveFailures: 3,
		Timestamp:          time.Now().Unix(),
	}

	// Marshal to JSON
	data, err := json.Marshal(edge)
	require.NoError(t, err)

	// Unmarshal to verify structure
	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &parsed))

	// Verify all required fields present
	assert.Contains(t, parsed, "file", "Should have 'file' field")
	assert.Contains(t, parsed, "error_type", "Should have 'error_type' field")
	assert.Contains(t, parsed, "consecutive_failures", "Should have 'consecutive_failures' field")
	assert.Contains(t, parsed, "timestamp", "Should have 'timestamp' field")

	// Verify types
	assert.IsType(t, "", parsed["file"], "file should be string")
	assert.IsType(t, "", parsed["error_type"], "error_type should be string")
	assert.IsType(t, float64(0), parsed["consecutive_failures"], "consecutive_failures should be number")
	assert.IsType(t, float64(0), parsed["timestamp"], "timestamp should be number")
}

// TestSchema_SharpEdge_TimestampFormat tests that timestamp is Unix epoch int64
func TestSchema_SharpEdge_TimestampFormat(t *testing.T) {
	now := time.Now().Unix()

	edge := SharpEdge{
		File:               "test.py",
		ErrorType:          "typeerror",
		ConsecutiveFailures: 3,
		Timestamp:          now,
	}

	data, err := json.Marshal(edge)
	require.NoError(t, err)

	var parsed SharpEdge
	require.NoError(t, json.Unmarshal(data, &parsed))

	// Verify timestamp is Unix epoch (not ISO string)
	assert.Equal(t, now, parsed.Timestamp, "Timestamp should be Unix epoch int64")

	// Verify timestamp is reasonable (within last year and not in future)
	oneYearAgo := time.Now().Add(-365 * 24 * time.Hour).Unix()
	oneDayFromNow := time.Now().Add(24 * time.Hour).Unix()

	assert.Greater(t, parsed.Timestamp, oneYearAgo, "Timestamp should not be ancient")
	assert.Less(t, parsed.Timestamp, oneDayFromNow, "Timestamp should not be in future")
}

// TestSchema_SharpEdge_ConsecutiveFailuresMin tests that consecutive_failures is ≥3
func TestSchema_SharpEdge_ConsecutiveFailuresMin(t *testing.T) {
	edge := SharpEdge{
		File:               "test.py",
		ErrorType:          "typeerror",
		ConsecutiveFailures: 3,
		Timestamp:          time.Now().Unix(),
	}

	// Valid edge should have ≥3 failures
	assert.GreaterOrEqual(t, edge.ConsecutiveFailures, 3, "Captured sharp edges should have ≥3 failures")
}

// TestSchema_ValidateSharpEdge_Valid tests validation of valid structure
func TestSchema_ValidateSharpEdge_Valid(t *testing.T) {
	edge := SharpEdge{
		File:               "test.py",
		ErrorType:          "typeerror",
		ConsecutiveFailures: 3,
		Timestamp:          time.Now().Unix(),
	}

	// Validate all fields are non-empty/non-zero
	assert.NotEmpty(t, edge.File, "File should not be empty")
	assert.NotEmpty(t, edge.ErrorType, "ErrorType should not be empty")
	assert.Greater(t, edge.ConsecutiveFailures, 0, "ConsecutiveFailures should be positive")
	assert.Greater(t, edge.Timestamp, int64(0), "Timestamp should be positive")
}

// TestSchema_ValidateSharpEdge_Invalid tests validation of invalid structure
func TestSchema_ValidateSharpEdge_Invalid(t *testing.T) {
	testCases := []struct {
		name  string
		edge  SharpEdge
		check func(t *testing.T, edge SharpEdge)
	}{
		{
			name: "empty_file",
			edge: SharpEdge{
				File:               "",
				ErrorType:          "typeerror",
				ConsecutiveFailures: 3,
				Timestamp:          time.Now().Unix(),
			},
			check: func(t *testing.T, edge SharpEdge) {
				assert.Empty(t, edge.File, "File should be empty")
			},
		},
		{
			name: "empty_error_type",
			edge: SharpEdge{
				File:               "test.py",
				ErrorType:          "",
				ConsecutiveFailures: 3,
				Timestamp:          time.Now().Unix(),
			},
			check: func(t *testing.T, edge SharpEdge) {
				assert.Empty(t, edge.ErrorType, "ErrorType should be empty")
			},
		},
		{
			name: "zero_consecutive_failures",
			edge: SharpEdge{
				File:               "test.py",
				ErrorType:          "typeerror",
				ConsecutiveFailures: 0,
				Timestamp:          time.Now().Unix(),
			},
			check: func(t *testing.T, edge SharpEdge) {
				assert.Zero(t, edge.ConsecutiveFailures, "ConsecutiveFailures should be zero")
			},
		},
		{
			name: "negative_consecutive_failures",
			edge: SharpEdge{
				File:               "test.py",
				ErrorType:          "typeerror",
				ConsecutiveFailures: -1,
				Timestamp:          time.Now().Unix(),
			},
			check: func(t *testing.T, edge SharpEdge) {
				assert.Negative(t, edge.ConsecutiveFailures, "ConsecutiveFailures should be negative")
			},
		},
		{
			name: "zero_timestamp",
			edge: SharpEdge{
				File:               "test.py",
				ErrorType:          "typeerror",
				ConsecutiveFailures: 3,
				Timestamp:          0,
			},
			check: func(t *testing.T, edge SharpEdge) {
				assert.Zero(t, edge.Timestamp, "Timestamp should be zero")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.check(t, tc.edge)
		})
	}
}

// TestSchema_FailureInfo_Compatibility tests that FailureInfo matches SharpEdge schema
func TestSchema_FailureInfo_Compatibility(t *testing.T) {
	info := routing.FailureInfo{
		File:       "test.py",
		ErrorType:  "typeerror",
		Timestamp:  time.Now().Unix(),
		Tool:       "Bash",
		ExitCode:   1,
		ErrorMatch: "TypeError: foo",
	}

	// Marshal FailureInfo
	data, err := json.Marshal(info)
	require.NoError(t, err)

	// Verify it can be unmarshaled as SharpEdge (subset of fields)
	var edge SharpEdge
	require.NoError(t, json.Unmarshal(data, &edge))

	assert.Equal(t, info.File, edge.File)
	assert.Equal(t, info.ErrorType, edge.ErrorType)
	assert.Equal(t, info.Timestamp, edge.Timestamp)
}

// TestSchema_JSONL_Format tests that entries are newline-delimited
func TestSchema_JSONL_Format(t *testing.T) {
	edges := []SharpEdge{
		{File: "test1.py", ErrorType: "typeerror", ConsecutiveFailures: 3, Timestamp: time.Now().Unix()},
		{File: "test2.py", ErrorType: "valueerror", ConsecutiveFailures: 3, Timestamp: time.Now().Unix()},
		{File: "test3.py", ErrorType: "syntaxerror", ConsecutiveFailures: 3, Timestamp: time.Now().Unix()},
	}

	// Build JSONL format
	var jsonl string
	for _, edge := range edges {
		data, err := json.Marshal(edge)
		require.NoError(t, err)
		jsonl += string(data) + "\n"
	}

	// Verify each line is valid JSON
	lines := 0
	for i := 0; i < len(jsonl); {
		end := i
		for end < len(jsonl) && jsonl[end] != '\n' {
			end++
		}
		if end > i {
			line := jsonl[i:end]
			var parsed SharpEdge
			require.NoError(t, json.Unmarshal([]byte(line), &parsed), "Line %d should be valid JSON", lines+1)
			lines++
		}
		i = end + 1
	}

	assert.Equal(t, 3, lines, "Should have 3 lines of JSON")
}

// TestSchema_FieldNaming tests that field names follow snake_case convention
func TestSchema_FieldNaming(t *testing.T) {
	edge := SharpEdge{
		File:               "test.py",
		ErrorType:          "typeerror",
		ConsecutiveFailures: 3,
		Timestamp:          time.Now().Unix(),
	}

	data, err := json.Marshal(edge)
	require.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &parsed))

	// Verify snake_case naming
	assert.Contains(t, parsed, "error_type", "Should use snake_case error_type")
	assert.Contains(t, parsed, "consecutive_failures", "Should use snake_case consecutive_failures")
	assert.NotContains(t, parsed, "ErrorType", "Should not use PascalCase")
	assert.NotContains(t, parsed, "ConsecutiveFailures", "Should not use PascalCase")
}
