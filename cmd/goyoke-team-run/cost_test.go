package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractCostFromCLIOutput(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantCost   float64
		wantStatus CostStatus
		wantErr    bool
	}{
		{
			name:       "TopLevel",
			input:      `{"cost_usd": 2.45}`,
			wantCost:   2.45,
			wantStatus: CostOK,
			wantErr:    false,
		},
		{
			name:       "TotalField",
			input:      `{"total_cost_usd": 1.80}`,
			wantCost:   1.80,
			wantStatus: CostOK,
			wantErr:    false,
		},
		{
			name:       "Nested",
			input:      `{"usage": {"cost_usd": 0.50}}`,
			wantCost:   0.50,
			wantStatus: CostOK,
			wantErr:    false,
		},
		{
			name:       "MissingField",
			input:      `{"status": "ok"}`,
			wantCost:   0,
			wantStatus: CostFallback,
			wantErr:    true,
		},
		{
			name:       "InvalidJSON",
			input:      `not json`,
			wantCost:   0,
			wantStatus: CostError,
			wantErr:    true,
		},
		{
			name:       "EmptyObject",
			input:      `{}`,
			wantCost:   0,
			wantStatus: CostFallback,
			wantErr:    true,
		},
		{
			name:       "NegativeValue",
			input:      `{"cost_usd": -5.0}`,
			wantCost:   -5.0,
			wantStatus: CostOK,
			wantErr:    false,
		},
		{
			name:       "LargeValue",
			input:      `{"cost_usd": 999.99}`,
			wantCost:   999.99,
			wantStatus: CostOK,
			wantErr:    false,
		},
		{
			name:       "ZeroCost",
			input:      `{"cost_usd": 0}`,
			wantCost:   0,
			wantStatus: CostOK,
			wantErr:    false,
		},
		{
			name:       "FloatPrecision",
			input:      `{"cost_usd": 0.000001}`,
			wantCost:   0.000001,
			wantStatus: CostOK,
			wantErr:    false,
		},
		{
			name:       "PriorityOrder_TopLevelWins",
			input:      `{"cost_usd": 3.0, "total_cost_usd": 5.0}`,
			wantCost:   3.0,
			wantStatus: CostOK,
			wantErr:    false,
		},
		{
			name:       "PriorityOrder_NestedFallback",
			input:      `{"total_cost_usd": 2.5, "usage": {"cost_usd": 1.0}}`,
			wantCost:   2.5,
			wantStatus: CostOK,
			wantErr:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractCostFromCLIOutput([]byte(tc.input))

			assert.Equal(t, tc.wantCost, result.Cost, "cost mismatch")
			assert.Equal(t, tc.wantStatus, result.Status, "status mismatch")

			if tc.wantErr {
				assert.Error(t, result.Err, "expected error")
			} else {
				assert.NoError(t, result.Err, "unexpected error")
			}
		})
	}
}

func TestGetNestedFloat(t *testing.T) {
	tests := []struct {
		name      string
		data      map[string]interface{}
		path      string
		wantValue float64
		wantOK    bool
	}{
		{
			name:      "TopLevelFloat",
			data:      map[string]interface{}{"cost": 1.5},
			path:      "cost",
			wantValue: 1.5,
			wantOK:    true,
		},
		{
			name:      "TopLevelInt",
			data:      map[string]interface{}{"count": 42},
			path:      "count",
			wantValue: 42.0,
			wantOK:    true,
		},
		{
			name: "DeepPath",
			data: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": map[string]interface{}{
						"level3": 3.14,
					},
				},
			},
			path:      "level1.level2.level3",
			wantValue: 3.14,
			wantOK:    true,
		},
		{
			name: "MissingIntermediate",
			data: map[string]interface{}{
				"level1": map[string]interface{}{},
			},
			path:      "level1.missing.field",
			wantValue: 0,
			wantOK:    false,
		},
		{
			name:      "MissingTopLevel",
			data:      map[string]interface{}{"other": 1.0},
			path:      "missing",
			wantValue: 0,
			wantOK:    false,
		},
		{
			name: "IntermediateNotMap",
			data: map[string]interface{}{
				"value": "string",
			},
			path:      "value.nested",
			wantValue: 0,
			wantOK:    false,
		},
		{
			name: "FinalNotNumber",
			data: map[string]interface{}{
				"field": "not a number",
			},
			path:      "field",
			wantValue: 0,
			wantOK:    false,
		},
		{
			name: "NestedInt",
			data: map[string]interface{}{
				"usage": map[string]interface{}{
					"tokens": 1000,
				},
			},
			path:      "usage.tokens",
			wantValue: 1000.0,
			wantOK:    true,
		},
		{
			name: "ZeroValue",
			data: map[string]interface{}{
				"cost": 0.0,
			},
			path:      "cost",
			wantValue: 0,
			wantOK:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotValue, gotOK := getNestedFloat(tc.data, tc.path)

			assert.Equal(t, tc.wantOK, gotOK, "ok flag mismatch")
			if tc.wantOK {
				assert.InDelta(t, tc.wantValue, gotValue, 0.0001, "value mismatch")
			}
		})
	}
}

// TestCostFallbackSemantics verifies that CostFallback signals the caller to use estimated cost
func TestCostFallbackSemantics(t *testing.T) {
	// Simulate missing cost field
	result := extractCostFromCLIOutput([]byte(`{"status": "success", "output": "done"}`))

	require.Equal(t, CostFallback, result.Status, "expected CostFallback for missing field")
	require.Error(t, result.Err, "CostFallback should have non-nil error")
	require.Equal(t, float64(0), result.Cost, "Cost should be 0 when fallback")

	// Verify error message is informative
	assert.Contains(t, result.Err.Error(), "no cost field found", "error should explain missing field")
}

// TestCostErrorSemantics verifies that CostError signals a parsing failure
func TestCostErrorSemantics(t *testing.T) {
	result := extractCostFromCLIOutput([]byte(`{invalid json`))

	require.Equal(t, CostError, result.Status, "expected CostError for invalid JSON")
	require.Error(t, result.Err, "CostError should have non-nil error")
	require.Equal(t, float64(0), result.Cost, "Cost should be 0 on error")

	// Verify error message mentions JSON parsing
	assert.Contains(t, result.Err.Error(), "not valid JSON", "error should mention JSON parsing")
}
