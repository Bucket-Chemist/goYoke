package teampreparesynth

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// ExtractedFinding is a normalized finding from a wave 0 reviewer stdout.
type ExtractedFinding struct {
	ReviewerID  string // e.g., "proteomics-reviewer"
	FindingID   string // e.g., "PROT-13"
	SharpEdgeID string // e.g., "proteomics-fdr-global-only" (may be empty)
	Severity    string // normalized: "critical", "warning", "info"
	Category    string // e.g., "fdr-control"
	File        string
	Line        int
	Title       string
	Message     string
}

// InteractionRule is a declarative cross-domain interaction pattern.
type InteractionRule struct {
	ID               string        `json:"id"`
	Name             string        `json:"name"`
	Algebra          string        `json:"algebra"` // additive, multiplicative, negating, gating
	SeverityOverride string        `json:"severity_override,omitempty"`
	Description      string        `json:"description"`
	Condition        RuleCondition `json:"condition"`
	MessageTemplate  string        `json:"message_template"`
	Layer            int           `json:"layer"`
	Tags             []string      `json:"tags"`
}

// RuleCondition specifies how matchers are combined.
type RuleCondition struct {
	Type     string          `json:"type"` // requires_all, requires_any
	Matchers []FindingMatcher `json:"matchers"`
}

// FindingMatcher is a single matcher within a rule condition.
type FindingMatcher struct {
	ReviewerPattern  string `json:"reviewer_pattern,omitempty"`
	SharpEdgePattern string `json:"sharp_edge_pattern,omitempty"`
	SeverityMinimum  string `json:"severity_minimum,omitempty"`
	FindingCategory  string `json:"finding_category,omitempty"`
	FindingContains  string `json:"finding_contains,omitempty"`
	FindingPresent   bool   `json:"finding_present,omitempty"`
}

// DetectedInteraction is a matched rule with the findings that triggered it.
type DetectedInteraction struct {
	RuleID          string
	RuleName        string
	Algebra         string
	Severity        string // resolved: override > algebra > max(findings)
	MatchedFindings []ExtractedFinding
	Message         string // template with substitutions
	Layer           int
	Tags            []string
}

// InteractionRulesConfig is the top-level structure of interaction-rules.json.
type InteractionRulesConfig struct {
	Version string            `json:"version"`
	Rules   []InteractionRule `json:"rules"`
}

// DetectionResult contains all detected interactions and unmatched findings.
type DetectionResult struct {
	DetectedInteractions []DetectedInteraction
	UnmatchedFindings    []ExtractedFinding
	RulesEvaluated       int
	FindingsTotal        int
}

// reviewerStdout is the parsed structure of a wave 0 reviewer's stdout JSON.
// Only fields needed for interaction detection are parsed.
type reviewerStdout struct {
	Status   string            `json:"status"`
	Findings []reviewerFinding `json:"findings"`
}

// reviewerFinding is a single finding from a reviewer's stdout JSON.
type reviewerFinding struct {
	ID          string `json:"id"`
	SharpEdgeID string `json:"sharp_edge_id"`
	Severity    string `json:"severity"` // CRITICAL, HIGH, MEDIUM, LOW, INFO (uppercase)
	Category    string `json:"category"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	Title       string `json:"title"`
	Message     string `json:"message"`
}

// LoadRules reads and parses interaction-rules.json from the given path.
// Returns empty config (not error) if file doesn't exist — graceful degradation.
func LoadRules(path string) (InteractionRulesConfig, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return InteractionRulesConfig{}, nil
	}
	if err != nil {
		return InteractionRulesConfig{}, fmt.Errorf("read rules file: %w", err)
	}
	var cfg InteractionRulesConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return InteractionRulesConfig{}, fmt.Errorf("parse rules file: %w", err)
	}
	return cfg, nil
}

// ExtractFindings parses a reviewer's stdout JSON and returns normalized findings.
// Handles the severity mapping: CRITICAL→critical, HIGH→warning, MEDIUM→warning, LOW→info, INFO→info.
// Returns empty slice (not error) for malformed JSON — graceful degradation with log warning.
func ExtractFindings(reviewerID string, stdoutJSON []byte) []ExtractedFinding {
	var out reviewerStdout
	if err := json.Unmarshal(stdoutJSON, &out); err != nil {
		log.Printf("[WARN] ExtractFindings: malformed JSON for reviewer %s: %v", reviewerID, err)
		return []ExtractedFinding{}
	}

	findings := make([]ExtractedFinding, 0, len(out.Findings))
	for _, f := range out.Findings {
		findings = append(findings, ExtractedFinding{
			ReviewerID:  reviewerID,
			FindingID:   f.ID,
			SharpEdgeID: f.SharpEdgeID,
			Severity:    mapSeverity(f.Severity),
			Category:    f.Category,
			File:        f.File,
			Line:        f.Line,
			Title:       f.Title,
			Message:     f.Message,
		})
	}
	return findings
}

// mapSeverity maps a reviewer's uppercase severity to the normalized 3-level system.
func mapSeverity(s string) string {
	switch strings.ToUpper(s) {
	case "CRITICAL":
		return "critical"
	case "HIGH", "WARNING":
		return "warning"
	case "MEDIUM":
		return "warning"
	case "LOW":
		return "info"
	case "INFO":
		return "info"
	default:
		return "info"
	}
}

// DetectInteractions evaluates all rules against all findings.
// Findings are ordered by (reviewer config order, finding array index) for deterministic matching.
// A finding may participate in multiple interactions (not consumed on first match).
func DetectInteractions(rules InteractionRulesConfig, findings []ExtractedFinding) DetectionResult {
	detected := make([]DetectedInteraction, 0)
	// Track which finding indices appear in at least one detected interaction.
	matchedIndices := make(map[int]bool)

	for _, rule := range rules.Rules {
		indices, ok := matchRuleByIndices(rule, findings)
		if !ok {
			continue
		}

		matchedFindings := make([]ExtractedFinding, len(indices))
		for j, idx := range indices {
			matchedFindings[j] = findings[idx]
			matchedIndices[idx] = true
		}

		interaction := DetectedInteraction{
			RuleID:          rule.ID,
			RuleName:        rule.Name,
			Algebra:         rule.Algebra,
			Severity:        resolveSeverity(rule, matchedFindings),
			MatchedFindings: matchedFindings,
			Message:         renderMessage(rule.MessageTemplate, matchedFindings),
			Layer:           rule.Layer,
			Tags:            rule.Tags,
		}
		detected = append(detected, interaction)
	}

	// Unmatched: findings that appeared in zero detected interactions.
	unmatched := make([]ExtractedFinding, 0)
	for i, f := range findings {
		if !matchedIndices[i] {
			unmatched = append(unmatched, f)
		}
	}

	return DetectionResult{
		DetectedInteractions: detected,
		UnmatchedFindings:    unmatched,
		RulesEvaluated:       len(rules.Rules),
		FindingsTotal:        len(findings),
	}
}

// matchRuleByIndices attempts to match a rule against findings.
// Returns (matchedIndices, true) if the rule fires, (nil, false) otherwise.
// Each returned index points to the finding in the findings slice that satisfied the corresponding matcher.
func matchRuleByIndices(rule InteractionRule, findings []ExtractedFinding) ([]int, bool) {
	switch rule.Condition.Type {
	case "requires_all":
		return matchRequiresAllByIndices(rule.Condition.Matchers, findings)
	case "requires_any":
		return matchRequiresAnyByIndices(rule.Condition.Matchers, findings)
	default:
		return nil, false
	}
}

// matchRequiresAllByIndices returns matched indices only if every matcher finds at least one matching finding.
// The first matching finding (by slice order) is used for each matcher.
func matchRequiresAllByIndices(matchers []FindingMatcher, findings []ExtractedFinding) ([]int, bool) {
	indices := make([]int, 0, len(matchers))
	for _, m := range matchers {
		found := -1
		for i, f := range findings {
			if matcherMatches(m, f) {
				found = i
				break
			}
		}
		if found == -1 {
			return nil, false
		}
		indices = append(indices, found)
	}
	return indices, true
}

// matchRequiresAnyByIndices returns matched indices if at least one matcher finds a matching finding.
// Returns indices for all matchers that found a match.
func matchRequiresAnyByIndices(matchers []FindingMatcher, findings []ExtractedFinding) ([]int, bool) {
	var indices []int
	for _, m := range matchers {
		for i, f := range findings {
			if matcherMatches(m, f) {
				indices = append(indices, i)
				break
			}
		}
	}
	if len(indices) == 0 {
		return nil, false
	}
	return indices, true
}

// globMatch matches a pattern against a value using filepath.Match semantics.
// Returns false for empty pattern. Returns false if pattern is invalid.
func globMatch(pattern, value string) bool {
	if pattern == "" {
		return false
	}
	matched, err := filepath.Match(pattern, value)
	if err != nil {
		return false
	}
	return matched
}

// severityAtLeast returns true if finding severity >= minimum.
// Ordering: info < warning < critical.
// Returns true if minimum is empty (no constraint).
func severityAtLeast(findingSeverity, minimum string) bool {
	if minimum == "" {
		return true
	}
	levels := map[string]int{
		"info":     0,
		"warning":  1,
		"critical": 2,
	}
	findingLevel, ok1 := levels[findingSeverity]
	minLevel, ok2 := levels[minimum]
	if !ok1 || !ok2 {
		return false
	}
	return findingLevel >= minLevel
}

// matcherMatches returns true if a single finding satisfies a matcher.
// When finding_present is true, ONLY reviewer_pattern is checked — all other fields are ignored.
func matcherMatches(m FindingMatcher, f ExtractedFinding) bool {
	if m.FindingPresent {
		// finding_present: true — only check reviewer_pattern, ignore all other fields.
		if m.ReviewerPattern == "" {
			return true
		}
		return globMatch(m.ReviewerPattern, f.ReviewerID)
	}

	if m.ReviewerPattern != "" && !globMatch(m.ReviewerPattern, f.ReviewerID) {
		return false
	}

	if m.SharpEdgePattern != "" && !globMatch(m.SharpEdgePattern, f.SharpEdgeID) {
		return false
	}

	if m.SeverityMinimum != "" && !severityAtLeast(f.Severity, m.SeverityMinimum) {
		return false
	}

	// finding_category: case-sensitive substring match (spec §4.2: "finding.Category contains")
	if m.FindingCategory != "" && !strings.Contains(f.Category, m.FindingCategory) {
		return false
	}

	// finding_contains: case-insensitive substring match (M-4: reduce false positives)
	if m.FindingContains != "" && !strings.Contains(strings.ToLower(f.Message), strings.ToLower(m.FindingContains)) {
		return false
	}

	return true
}

// severityLevels maps severity names to ordinal values for comparison.
var severityLevels = map[string]int{
	"info":     0,
	"warning":  1,
	"critical": 2,
}

// severityNames maps ordinal values back to severity names.
var severityNames = []string{"info", "warning", "critical"}

// resolveSeverity determines the final severity for a detected interaction.
// Priority: severity_override > algebra-based > max(matched finding severities).
func resolveSeverity(rule InteractionRule, matched []ExtractedFinding) string {
	if rule.SeverityOverride != "" {
		return rule.SeverityOverride
	}

	if len(matched) == 0 {
		return "info"
	}

	maxLevel := 0
	minLevel := 2
	for _, f := range matched {
		if l, ok := severityLevels[f.Severity]; ok {
			if l > maxLevel {
				maxLevel = l
			}
			if l < minLevel {
				minLevel = l
			}
		}
	}

	switch rule.Algebra {
	case "additive":
		// Escalate max severity by one level, capped at critical.
		escalated := maxLevel + 1
		if escalated > 2 {
			escalated = 2
		}
		return severityNames[escalated]
	case "multiplicative":
		// Upstream finding multiplies downstream impact — use max severity.
		return severityNames[maxLevel]
	case "gating":
		// Upstream finding invalidates downstream — use max severity.
		return severityNames[maxLevel]
	case "negating":
		// One finding mitigates another — use min severity.
		return severityNames[minLevel]
	default:
		return severityNames[maxLevel]
	}
}

// renderMessage substitutes finding references into the message template.
// Supports: {finding_a.*}, {finding_b.*}, {matched_findings}.
func renderMessage(template string, matched []ExtractedFinding) string {
	result := template

	if len(matched) > 0 {
		result = strings.ReplaceAll(result, "{finding_a.sharp_edge_id}", matched[0].SharpEdgeID)
		result = strings.ReplaceAll(result, "{finding_a.reviewer}", matched[0].ReviewerID)
		result = strings.ReplaceAll(result, "{finding_a.title}", matched[0].Title)
		result = strings.ReplaceAll(result, "{finding_a.message}", matched[0].Message)
		result = strings.ReplaceAll(result, "{finding_a.severity}", matched[0].Severity)
	}
	if len(matched) > 1 {
		result = strings.ReplaceAll(result, "{finding_b.sharp_edge_id}", matched[1].SharpEdgeID)
		result = strings.ReplaceAll(result, "{finding_b.reviewer}", matched[1].ReviewerID)
		result = strings.ReplaceAll(result, "{finding_b.title}", matched[1].Title)
		result = strings.ReplaceAll(result, "{finding_b.message}", matched[1].Message)
		result = strings.ReplaceAll(result, "{finding_b.severity}", matched[1].Severity)
	}

	if strings.Contains(result, "{matched_findings}") {
		parts := make([]string, 0, len(matched))
		for _, f := range matched {
			parts = append(parts, fmt.Sprintf("%s/%s(%s)", f.ReviewerID, f.SharpEdgeID, f.Severity))
		}
		result = strings.ReplaceAll(result, "{matched_findings}", strings.Join(parts, ", "))
	}

	return result
}
