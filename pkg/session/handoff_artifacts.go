package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
)

// UserIntent captures user decision inputs and expressed preferences
// This enables tracking of explicit user choices for cross-session memory
type UserIntent struct {
	Timestamp   int64    `json:"timestamp"`              // When captured
	Question    string   `json:"question"`               // What was asked
	Response    string   `json:"response"`               // User's answer
	Confidence  string   `json:"confidence"`             // "explicit", "inferred", "default"
	Context     string   `json:"context,omitempty"`      // Why this was asked
	Source      string   `json:"source"`                 // "ask_user", "hook_prompt", "manual"
	ActionTaken string   `json:"action_taken,omitempty"` // What we did with the response
	SessionID   string   `json:"session_id,omitempty"`   // Session that captured this intent (GOgent-037d)
	ToolContext string   `json:"tool_context,omitempty"` // Tool invocation context (GOgent-037d)
	Category    string   `json:"category,omitempty"`     // Intent category from ClassifyIntent (GOgent-041)
	Keywords    []string `json:"keywords,omitempty"`     // Extracted keywords from ExtractKeywords (GOgent-041)
	// GOgent-041c: Outcome Tracking
	Honored     *bool  `json:"honored,omitempty"`      // Whether intent was followed (nil = not yet analyzed)
	OutcomeNote string `json:"outcome_note,omitempty"` // Explanation of outcome
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

// Decision captures architectural decisions made during session
type Decision struct {
	Timestamp    int64  `json:"timestamp"`
	Category     string `json:"category"`     // "architecture", "tooling", "pattern"
	Decision     string `json:"decision"`     // What was decided
	Rationale    string `json:"rationale"`    // Why
	Alternatives string `json:"alternatives"` // What was considered
	Impact       string `json:"impact"`       // "high", "medium", "low"
}

// PreferenceOverride captures user-specific workflow preferences
type PreferenceOverride struct {
	Timestamp int64  `json:"timestamp"`
	Category  string `json:"category"` // "routing", "tooling", "formatting"
	Key       string `json:"key"`      // Preference identifier
	Value     string `json:"value"`    // Preferred value
	Reason    string `json:"reason"`   // Why user prefers this
	Scope     string `json:"scope"`    // "session", "project", "global"
}

// PerformanceMetric captures execution performance patterns
type PerformanceMetric struct {
	Timestamp   int64  `json:"timestamp"`
	Operation   string `json:"operation"`    // "handoff_generation", "validation", etc.
	DurationMs  int64  `json:"duration_ms"`  // Execution time
	MemoryBytes int64  `json:"memory_bytes"` // Peak memory (if measurable)
	Success     bool   `json:"success"`      // Operation outcome
	Context     string `json:"context"`      // Additional context
}

// EndstateLog represents a single logged endstate decision (v1.3 - GOgent-065)
type EndstateLog struct {
	Timestamp       time.Time `json:"timestamp"`
	AgentID         string    `json:"agent_id"`
	AgentClass      string    `json:"agent_class"`
	Tier            string    `json:"tier"`
	ExitCode        int       `json:"exit_code"`
	Duration        int       `json:"duration_ms"`
	OutputTokens    int       `json:"output_tokens"`
	Decision        string    `json:"decision"` // "prompt" or "silent"
	Recommendations []string  `json:"recommendations"`
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

// LoadArtifacts loads all session artifacts (sharp edges, violations, error patterns, user intents,
// decisions, preferences, performance metrics, agent endstates)
func LoadArtifacts(cfg *HandoffConfig) (HandoffArtifacts, error) {
	artifacts := HandoffArtifacts{
		SharpEdges:          []SharpEdge{},
		RoutingViolations:   []RoutingViolation{},
		ErrorPatterns:       []ErrorPattern{},
		UserIntents:         []UserIntent{},
		Decisions:           []Decision{},
		PreferenceOverrides: []PreferenceOverride{},
		PerformanceMetrics:  []PerformanceMetric{},
		AgentEndstates:      []EndstateLog{},
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

	// Load decisions
	decisions, err := loadDecisions(cfg.DecisionsPath)
	if err != nil {
		return artifacts, fmt.Errorf("[handoff] Failed to load decisions: %w", err)
	}
	artifacts.Decisions = decisions

	// Load preference overrides
	preferences, err := loadPreferences(cfg.PreferencesPath)
	if err != nil {
		return artifacts, fmt.Errorf("[handoff] Failed to load preferences: %w", err)
	}
	artifacts.PreferenceOverrides = preferences

	// Load performance metrics
	perfMetrics, err := loadPerformance(cfg.PerformancePath)
	if err != nil {
		return artifacts, fmt.Errorf("[handoff] Failed to load performance metrics: %w", err)
	}
	artifacts.PerformanceMetrics = perfMetrics

	// Load agent endstates (v1.3)
	endstates, err := loadEndstates(cfg.ProjectDir)
	if err != nil {
		return artifacts, fmt.Errorf("[handoff] Failed to load agent endstates: %w", err)
	}
	artifacts.AgentEndstates = endstates

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
	scanner := newSessionScanner(file)

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
	scanner := newSessionScanner(file)

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
	scanner := newSessionScanner(file)

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
	scanner := newSessionScanner(file)

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

// LoadAllUserIntents loads all user intents from JSONL file (exported for CLI usage)
func LoadAllUserIntents(path string) ([]UserIntent, error) {
	return loadUserIntents(path)
}

// loadDecisions reads decisions from decisions.jsonl
func loadDecisions(path string) ([]Decision, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Decision{}, nil // Missing file is normal
		}
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer file.Close()

	var decisions []Decision
	scanner := newSessionScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var d Decision
		if err := json.Unmarshal([]byte(line), &d); err != nil {
			// Skip malformed lines but continue
			continue
		}
		decisions = append(decisions, d)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", path, err)
	}

	return decisions, nil
}

// loadPreferences reads preference overrides from preferences.jsonl
func loadPreferences(path string) ([]PreferenceOverride, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []PreferenceOverride{}, nil // Missing file is normal
		}
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer file.Close()

	var preferences []PreferenceOverride
	scanner := newSessionScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var p PreferenceOverride
		if err := json.Unmarshal([]byte(line), &p); err != nil {
			// Skip malformed lines but continue
			continue
		}
		preferences = append(preferences, p)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", path, err)
	}

	return preferences, nil
}

// loadPerformance reads performance metrics from performance.jsonl
func loadPerformance(path string) ([]PerformanceMetric, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []PerformanceMetric{}, nil // Missing file is normal
		}
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer file.Close()

	var metrics []PerformanceMetric
	scanner := newSessionScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var m PerformanceMetric
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			// Skip malformed lines but continue
			continue
		}
		metrics = append(metrics, m)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", path, err)
	}

	return metrics, nil
}

// loadEndstates reads agent endstate logs from project-scoped agent-endstates.jsonl (v1.3)
func loadEndstates(projectDir string) ([]EndstateLog, error) {
	path := filepath.Join(config.ProjectMemoryDir(projectDir), "agent-endstates.jsonl")

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []EndstateLog{}, nil // Missing file is normal
		}
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer file.Close()

	var endstates []EndstateLog
	scanner := newSessionScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var e EndstateLog
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			// Skip malformed lines but continue
			continue
		}
		endstates = append(endstates, e)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", path, err)
	}

	return endstates, nil
}
