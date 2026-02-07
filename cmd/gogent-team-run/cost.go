package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// CostStatus represents the outcome of cost extraction from CLI output.
type CostStatus string

const (
	CostOK       CostStatus = "ok"       // Cost extracted successfully
	CostFallback CostStatus = "fallback" // No cost field found; caller should use estimated cost
	CostError    CostStatus = "error"    // JSON parse failed
)

// CostResult contains the result of cost extraction.
type CostResult struct {
	Cost   float64
	Status CostStatus
	Err    error
}

// extractCostFromCLIOutput parses Claude CLI JSON output and extracts cost.
// Tries fields in order: cost_usd, total_cost_usd, usage.cost_usd.
// Returns CostFallback (not zero) when no field found — caller must use estimated cost.
//
// Usage:
//
//	result := extractCostFromCLIOutput(stdoutBytes)
//	if result.Status == CostFallback {
//	    actualCost = estimatedCost  // Use conservative fallback
//	} else if result.Status == CostError {
//	    return fmt.Errorf("cost extraction: %w", result.Err)
//	} else {
//	    actualCost = result.Cost
//	}
func extractCostFromCLIOutput(output []byte) CostResult {
	var data map[string]interface{}
	if err := json.Unmarshal(output, &data); err != nil {
		return CostResult{
			Cost:   0,
			Status: CostError,
			Err:    fmt.Errorf("CLI output not valid JSON: %w", err),
		}
	}

	// Try fields in priority order
	fields := []string{"cost_usd", "total_cost_usd", "usage.cost_usd"}
	for _, field := range fields {
		if val, ok := getNestedFloat(data, field); ok {
			return CostResult{
				Cost:   val,
				Status: CostOK,
			}
		}
	}

	// C3 FIX: Missing field = CostFallback status, caller uses estimated cost
	return CostResult{
		Cost:   0,
		Status: CostFallback,
		Err:    fmt.Errorf("no cost field found in CLI output (checked: %s)", strings.Join(fields, ", ")),
	}
}

// getNestedFloat traverses a map by dot-separated path (e.g., "usage.cost_usd").
// Returns (value, true) if found, (0, false) otherwise.
//
// Examples:
//
//	getNestedFloat({"cost": 1.5}, "cost") → (1.5, true)
//	getNestedFloat({"usage": {"cost": 2.0}}, "usage.cost") → (2.0, true)
//	getNestedFloat({"usage": {}}, "usage.missing") → (0, false)
func getNestedFloat(m map[string]interface{}, path string) (float64, bool) {
	parts := strings.Split(path, ".")
	current := m

	// Traverse all but last part
	for i := 0; i < len(parts)-1; i++ {
		val, ok := current[parts[i]]
		if !ok {
			return 0, false
		}
		// Next level must be a map
		nextMap, ok := val.(map[string]interface{})
		if !ok {
			return 0, false
		}
		current = nextMap
	}

	// Get final value
	val, ok := current[parts[len(parts)-1]]
	if !ok {
		return 0, false
	}

	// Handle both float64 and int representations
	switch v := val.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}
