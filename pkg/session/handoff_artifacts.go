package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

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
