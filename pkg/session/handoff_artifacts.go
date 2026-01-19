package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ValidateSharpEdge validates SharpEdge JSON against schema
func ValidateSharpEdge(data []byte) error {
	// Parse into a map to validate against schema (not current struct)
	var edge map[string]interface{}
	if err := json.Unmarshal(data, &edge); err != nil {
		return fmt.Errorf("[sharp-edge-validation] Invalid JSON: %w. Ensure JSONL is well-formed.", err)
	}

	// Required field: file
	file, fileExists := edge["file"]
	if !fileExists {
		return fmt.Errorf("[sharp-edge-validation] Missing required field 'file'. Sharp edge must specify file path.")
	}
	if fileStr, ok := file.(string); !ok || fileStr == "" {
		return fmt.Errorf("[sharp-edge-validation] Missing required field 'file'. Sharp edge must specify file path.")
	}

	// Required field: error_type
	errorType, errorTypeExists := edge["error_type"]
	if !errorTypeExists {
		return fmt.Errorf("[sharp-edge-validation] Missing required field 'error_type'. Sharp edge must classify error.")
	}
	if errorTypeStr, ok := errorType.(string); !ok || errorTypeStr == "" {
		return fmt.Errorf("[sharp-edge-validation] Missing required field 'error_type'. Sharp edge must classify error.")
	}

	// Required field: failure_count (minimum 3)
	failureCount, failureCountExists := edge["failure_count"]
	if !failureCountExists {
		return fmt.Errorf("[sharp-edge-validation] Invalid failure_count 0. Must be ≥3 for sharp edge threshold.")
	}
	// JSON numbers are float64 by default
	var count int
	switch v := failureCount.(type) {
	case float64:
		count = int(v)
	case int:
		count = v
	default:
		return fmt.Errorf("[sharp-edge-validation] Invalid failure_count type. Must be integer.")
	}
	if count < 3 {
		return fmt.Errorf("[sharp-edge-validation] Invalid failure_count %d. Must be ≥3 for sharp edge threshold.", count)
	}

	// Required field: last_occurrence
	lastOccurrence, lastOccurrenceExists := edge["last_occurrence"]
	if !lastOccurrenceExists {
		return fmt.Errorf("[sharp-edge-validation] Missing required field 'last_occurrence'. Sharp edge must have timestamp.")
	}
	// Validate it's a number (int or float)
	switch v := lastOccurrence.(type) {
	case float64:
		if int64(v) == 0 {
			return fmt.Errorf("[sharp-edge-validation] Missing required field 'last_occurrence'. Sharp edge must have timestamp.")
		}
	case int64:
		if v == 0 {
			return fmt.Errorf("[sharp-edge-validation] Missing required field 'last_occurrence'. Sharp edge must have timestamp.")
		}
	case int:
		if v == 0 {
			return fmt.Errorf("[sharp-edge-validation] Missing required field 'last_occurrence'. Sharp edge must have timestamp.")
		}
	default:
		return fmt.Errorf("[sharp-edge-validation] Invalid last_occurrence type. Must be integer.")
	}

	return nil
}

// LoadArtifacts loads all session artifacts (sharp edges, violations, error patterns)
func LoadArtifacts(cfg *HandoffConfig) (HandoffArtifacts, error) {
	artifacts := HandoffArtifacts{
		SharpEdges:        []SharpEdge{},
		RoutingViolations: []RoutingViolation{},
		ErrorPatterns:     []ErrorPattern{},
	}

	// Load pending learnings (sharp edges)
	edges, err := loadPendingLearnings(cfg.PendingPath)
	if err != nil {
		return artifacts, fmt.Errorf("[handoff] Failed to load pending learnings: %w", err)
	}
	artifacts.SharpEdges = edges

	// Load routing violations
	violations, err := loadViolations(cfg.ViolationsPath)
	if err != nil {
		return artifacts, fmt.Errorf("[handoff] Failed to load routing violations: %w", err)
	}
	artifacts.RoutingViolations = violations

	// Load error patterns
	patterns, err := loadErrorPatterns(cfg.ErrorPatternsPath)
	if err != nil {
		return artifacts, fmt.Errorf("[handoff] Failed to load error patterns: %w", err)
	}
	artifacts.ErrorPatterns = patterns

	return artifacts, nil
}

// loadPendingLearnings reads sharp edges from pending-learnings.jsonl
func loadPendingLearnings(path string) ([]SharpEdge, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []SharpEdge{}, nil // Missing file is normal
		}
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer file.Close()

	var edges []SharpEdge
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var edge SharpEdge
		if err := json.Unmarshal([]byte(line), &edge); err != nil {
			// Skip malformed lines but continue
			continue
		}
		edges = append(edges, edge)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", path, err)
	}

	return edges, nil
}

// loadViolations reads routing violations from routing-violations.jsonl
func loadViolations(path string) ([]RoutingViolation, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []RoutingViolation{}, nil // Missing file is normal
		}
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer file.Close()

	var violations []RoutingViolation
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var v RoutingViolation
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			// Skip malformed lines but continue
			continue
		}
		violations = append(violations, v)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", path, err)
	}

	return violations, nil
}

// loadErrorPatterns reads error patterns from error-patterns.jsonl
func loadErrorPatterns(path string) ([]ErrorPattern, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []ErrorPattern{}, nil // Missing file is normal
		}
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer file.Close()

	var patterns []ErrorPattern
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var p ErrorPattern
		if err := json.Unmarshal([]byte(line), &p); err != nil {
			// Skip malformed lines but continue
			continue
		}
		patterns = append(patterns, p)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", path, err)
	}

	return patterns, nil
}
