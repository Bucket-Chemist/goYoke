package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// UserIntent captures user decision inputs and expressed preferences
// This enables tracking of explicit user choices for cross-session memory
type UserIntent struct {
	Timestamp   int64  `json:"timestamp"`              // When captured
	Question    string `json:"question"`               // What was asked
	Response    string `json:"response"`               // User's answer
	Confidence  string `json:"confidence"`             // "explicit", "inferred", "default"
	Context     string `json:"context,omitempty"`      // Why this was asked
	Source      string `json:"source"`                 // "ask_user", "hook_prompt", "manual"
	ActionTaken string `json:"action_taken,omitempty"` // What we did with the response
}

// ValidConfidenceLevels defines valid confidence values for UserIntent
var ValidConfidenceLevels = map[string]bool{
	"explicit": true, // User directly answered
	"inferred": true, // Derived from user behavior
	"default":  true, // Used default when user skipped
}

// ValidIntentSources defines valid source values for UserIntent capture
var ValidIntentSources = map[string]bool{
	"ask_user":    true, // AskUserQuestion tool
	"hook_prompt": true, // Hook-injected prompt
	"manual":      true, // Manually recorded
}

// ValidateSharpEdge validates SharpEdge JSON against schema
// FIXED: Field names now match struct (consecutive_failures, timestamp)
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

	// Required field: consecutive_failures (minimum 3)
	// FIXED: Was incorrectly checking "failure_count"
	consecutiveFailures, consecutiveFailuresExists := edge["consecutive_failures"]
	if !consecutiveFailuresExists {
		return fmt.Errorf("[sharp-edge-validation] Invalid consecutive_failures 0. Must be >=3 for sharp edge threshold.")
	}
	// JSON numbers are float64 by default
	var count int
	switch v := consecutiveFailures.(type) {
	case float64:
		count = int(v)
	case int:
		count = v
	default:
		return fmt.Errorf("[sharp-edge-validation] Invalid consecutive_failures type. Must be integer.")
	}
	if count < 3 {
		return fmt.Errorf("[sharp-edge-validation] Invalid consecutive_failures %d. Must be >=3 for sharp edge threshold.", count)
	}

	// Required field: timestamp
	// FIXED: Was incorrectly checking "last_occurrence"
	timestamp, timestampExists := edge["timestamp"]
	if !timestampExists {
		return fmt.Errorf("[sharp-edge-validation] Missing required field 'timestamp'. Sharp edge must have timestamp.")
	}
	// Validate it's a number (int or float)
	switch v := timestamp.(type) {
	case float64:
		if int64(v) == 0 {
			return fmt.Errorf("[sharp-edge-validation] Missing required field 'timestamp'. Sharp edge must have timestamp.")
		}
	case int64:
		if v == 0 {
			return fmt.Errorf("[sharp-edge-validation] Missing required field 'timestamp'. Sharp edge must have timestamp.")
		}
	case int:
		if v == 0 {
			return fmt.Errorf("[sharp-edge-validation] Missing required field 'timestamp'. Sharp edge must have timestamp.")
		}
	default:
		return fmt.Errorf("[sharp-edge-validation] Invalid timestamp type. Must be integer.")
	}

	// Optional field: severity (validate if present)
	if severity, exists := edge["severity"]; exists {
		if s, ok := severity.(string); ok && s != "" {
			validSeverities := map[string]bool{"high": true, "medium": true, "low": true}
			if !validSeverities[s] {
				return fmt.Errorf("[sharp-edge-validation] Invalid severity '%s'. Must be 'high', 'medium', or 'low'.", s)
			}
		}
	}

	return nil
}

// LoadArtifacts loads all session artifacts (sharp edges, violations, error patterns, user intents)
func LoadArtifacts(cfg *HandoffConfig) (HandoffArtifacts, error) {
	artifacts := HandoffArtifacts{
		SharpEdges:        []SharpEdge{},
		RoutingViolations: []RoutingViolation{},
		ErrorPatterns:     []ErrorPattern{},
		UserIntents:       []UserIntent{},
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

	// Load user intents
	intents, err := loadUserIntents(cfg.UserIntentsPath)
	if err != nil {
		return artifacts, fmt.Errorf("[handoff] Failed to load user intents: %w", err)
	}
	artifacts.UserIntents = intents

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

// loadUserIntents reads user intents from user-intents.jsonl
func loadUserIntents(path string) ([]UserIntent, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []UserIntent{}, nil // Missing file is normal
		}
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer file.Close()

	var intents []UserIntent
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var intent UserIntent
		if err := json.Unmarshal([]byte(line), &intent); err != nil {
			// Skip malformed lines but continue
			continue
		}
		intents = append(intents, intent)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", path, err)
	}

	return intents, nil
}
