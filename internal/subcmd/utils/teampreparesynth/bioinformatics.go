package teampreparesynth

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// wave0Output represents a single reviewer's result for the synthesizer's stdin.
type wave0Output struct {
	ReviewerID     string  `json:"reviewer_id"`
	StdoutFilePath string  `json:"stdout_file_path"`
	Status         string  `json:"status"`
	CostUSD        float64 `json:"cost_usd"`
}

// prepareBioinformaticsReview updates the synthesizer (staff-bioinformatician or pasteur)
// stdin JSON with wave_0_outputs, writes pre-synthesis.md with programmatic interaction
// detection, writes detected-interactions.json sidecar, and adds detected_interactions_path
// to the stdin. Graceful degradation: missing rules or stdout files are logged as warnings
// and do not cause failure.
//
// M-2: checks for "staff-bioinformatician" first, falls back to "pasteur" for backward compat.
// M-1: rules path is resolved from GOYOKE_CONFIG_DIR or ~/.claude — not hardcoded.
func prepareBioinformaticsReview(teamDir string, config *TeamConfig, completedWaveIdx int) error {
	if completedWaveIdx+1 >= len(config.Waves) {
		log.Printf("[WARN] prepareBioinformaticsReview: completedWaveIdx %d has no next wave", completedWaveIdx)
		return nil
	}

	// Find synthesizer: staff-bioinformatician preferred, pasteur for backward compat.
	synthesizerMember := findSynthesizerMember(config, completedWaveIdx+1)
	if synthesizerMember == nil {
		log.Printf("[WARN] prepareBioinformaticsReview: no synthesizer (staff-bioinformatician or pasteur) in wave %d, skipping", completedWaveIdx+1)
		return nil
	}

	stdinPath := filepath.Join(teamDir, synthesizerMember.StdinFile)
	data, err := os.ReadFile(stdinPath)
	if err != nil {
		log.Printf("[WARN] prepareBioinformaticsReview: cannot read %s: %v, skipping", stdinPath, err)
		return nil
	}

	var stdinData map[string]interface{}
	if err := json.Unmarshal(data, &stdinData); err != nil {
		log.Printf("[WARN] prepareBioinformaticsReview: cannot parse %s: %v, skipping", stdinPath, err)
		return nil
	}

	// Extract findings from completed wave 0 reviewers (graceful: missing files are skipped).
	allFindings := extractAllFindings(config, teamDir, completedWaveIdx)
	nReviewers := countUniqueReviewers(allFindings)

	// Load interaction rules (M-1: parameterized path, graceful on missing/malformed file).
	rulesPath := resolveRulesPath()
	rules, err := LoadRules(rulesPath)
	if err != nil {
		log.Printf("[interactions] Warning: could not load rules from %s: %v (proceeding without interaction detection)", rulesPath, err)
		rules = InteractionRulesConfig{}
	}

	// Detect interactions deterministically.
	result := DetectInteractions(rules, allFindings)
	log.Printf("[interactions] Detected %d interactions from %d findings (%d rules evaluated)",
		len(result.DetectedInteractions), result.FindingsTotal, result.RulesEvaluated)

	// Write pre-synthesis.md (non-fatal on error).
	preSynthesisPath := filepath.Join(teamDir, "pre-synthesis.md")
	preSynthesisContent := generateBioinformaticsPreSynthesis(result, rules, nReviewers)
	if err := os.WriteFile(preSynthesisPath, []byte(preSynthesisContent), 0644); err != nil {
		log.Printf("[WARN] prepareBioinformaticsReview: could not write pre-synthesis.md: %v", err)
		preSynthesisPath = ""
	}

	// Write detected-interactions.json sidecar (non-fatal on error).
	interactionsJSONPath, err := writeDetectedInteractionsJSON(teamDir, result, rules)
	if err != nil {
		log.Printf("[WARN] prepareBioinformaticsReview: could not write detected-interactions.json: %v", err)
		interactionsJSONPath = ""
	}

	// Build wave_0_outputs from completed wave members.
	outputs := make([]wave0Output, 0, len(config.Waves[completedWaveIdx].Members))
	for _, m := range config.Waves[completedWaveIdx].Members {
		outputs = append(outputs, wave0Output{
			ReviewerID:     m.Agent,
			StdoutFilePath: filepath.Join(teamDir, m.StdoutFile),
			Status:         memberStatusToOutputStatus(m.Status),
			CostUSD:        m.CostUSD,
		})
	}

	stdinData["wave_0_outputs"] = outputs
	if preSynthesisPath != "" {
		stdinData["wave0_findings_path"] = preSynthesisPath
	}
	if interactionsJSONPath != "" {
		stdinData["detected_interactions_path"] = interactionsJSONPath
	}

	updated, err := json.MarshalIndent(stdinData, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal updated stdin: %w", err)
	}

	// Atomic write: write to .tmp then rename.
	tmpPath := stdinPath + ".tmp"
	if err := os.WriteFile(tmpPath, updated, 0644); err != nil {
		return fmt.Errorf("write tmp stdin file: %w", err)
	}
	if err := os.Rename(tmpPath, stdinPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename tmp to stdin file: %w", err)
	}

	log.Printf("[INFO] prepareBioinformaticsReview: updated %s with %d wave_0_outputs, %d interactions",
		stdinPath, len(outputs), len(result.DetectedInteractions))
	return nil
}

// memberStatusToOutputStatus maps member.Status to the output status values
// expected by the synthesizer's stdin schema.
func memberStatusToOutputStatus(status string) string {
	switch status {
	case "completed":
		return "completed"
	case "failed":
		return "failed"
	default:
		return "timeout"
	}
}

// findSynthesizerMember returns the MemberInfo for the synthesizer in the given wave.
// Checks for "staff-bioinformatician" first, falls back to "pasteur" for backward compat.
// Returns nil if neither is found.
func findSynthesizerMember(config *TeamConfig, waveIdx int) *MemberInfo {
	if waveIdx >= len(config.Waves) {
		return nil
	}
	var pasteurMember *MemberInfo
	for i := range config.Waves[waveIdx].Members {
		m := &config.Waves[waveIdx].Members[i]
		if m.Agent == "staff-bioinformatician" {
			return m
		}
		if m.Agent == "pasteur" {
			pasteurMember = m
		}
	}
	return pasteurMember
}

// resolveRulesPath returns the path to interaction-rules.json.
// Uses GOYOKE_CONFIG_DIR environment variable if set, otherwise falls back to ~/.claude.
func resolveRulesPath() string {
	configDir := os.Getenv("GOYOKE_CONFIG_DIR")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Printf("[interactions] Warning: cannot resolve home dir: %v", err)
			return ""
		}
		configDir = filepath.Join(home, ".claude")
	}
	return filepath.Join(configDir, "schemas", "teams", "interaction-rules.json")
}

// extractAllFindings reads stdout files from all completed wave members and extracts findings.
// Files that cannot be read are skipped with a log warning (graceful degradation).
func extractAllFindings(config *TeamConfig, teamDir string, waveIdx int) []ExtractedFinding {
	var allFindings []ExtractedFinding
	for _, m := range config.Waves[waveIdx].Members {
		if m.Status != "completed" || m.StdoutFile == "" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(teamDir, m.StdoutFile))
		if err != nil {
			log.Printf("[interactions] Warning: could not read %s: %v", m.StdoutFile, err)
			continue
		}
		allFindings = append(allFindings, ExtractFindings(m.Agent, data)...)
	}
	return allFindings
}

// countUniqueReviewers counts the distinct reviewer IDs across findings.
func countUniqueReviewers(findings []ExtractedFinding) int {
	seen := make(map[string]bool)
	for _, f := range findings {
		seen[f.ReviewerID] = true
	}
	return len(seen)
}

// generateBioinformaticsPreSynthesis creates the pre-synthesis.md content for the
// review-bioinformatics workflow. Includes the programmatic interaction detection section.
func generateBioinformaticsPreSynthesis(result DetectionResult, rules InteractionRulesConfig, nReviewers int) string {
	var sb strings.Builder
	sb.WriteString("# Bioinformatics Review Pre-Synthesis\n\n")
	sb.WriteString("Generated by `goyoke-team-prepare-synthesis`. " +
		"This document provides programmatic interaction detection results for the staff-bioinformatician.\n\n")
	appendInteractionSection(&sb, result, rules, nReviewers)
	return sb.String()
}

// appendInteractionSection writes the interaction detection markdown section to sb.
func appendInteractionSection(sb *strings.Builder, result DetectionResult, rules InteractionRulesConfig, nReviewers int) {
	matchedCount := result.FindingsTotal - len(result.UnmatchedFindings)

	sb.WriteString("---\n\n")
	sb.WriteString("## Programmatic Cross-Domain Interaction Detection\n\n")
	fmt.Fprintf(sb,
		"> Generated by goyoke-team-prepare-synthesis using interaction-rules.json v%s.\n"+
			"> %d rules evaluated against %d findings from %d reviewers.\n"+
			"> %d interactions detected, %d findings unmatched.\n\n",
		rules.Version,
		result.RulesEvaluated,
		result.FindingsTotal,
		nReviewers,
		len(result.DetectedInteractions),
		len(result.UnmatchedFindings),
	)

	if len(result.DetectedInteractions) > 0 {
		sb.WriteString("### Detected Interactions\n\n")
		for _, d := range result.DetectedInteractions {
			fmt.Fprintf(sb, "#### %s: %s (%s)\n\n", strings.ToUpper(d.Severity), d.RuleID, d.Algebra)
			fmt.Fprintf(sb, "**Rule:** %s\n\n", d.RuleName)

			sb.WriteString("| Source | Reviewer | Sharp Edge | Severity | Finding |\n")
			sb.WriteString("|--------|----------|-----------|----------|---------|\n")
			for i, f := range d.MatchedFindings {
				src := string(rune('A' + i))
				fmt.Fprintf(sb, "| %s | %s | %s | %s | %s |\n",
					src, f.ReviewerID, f.SharpEdgeID, f.Severity, f.Title)
			}
			sb.WriteString("\n")

			fmt.Fprintf(sb, "**Interaction:** %s\n", d.Message)
			fmt.Fprintf(sb, "**Algebra:** %s — %s\n", d.Algebra, algebraDescription(d.Algebra))
			sb.WriteString("**Staff-bioinformatician action:** VERIFY this interaction.\n\n")
			sb.WriteString("---\n\n")
		}
	} else {
		sb.WriteString("No interactions detected.\n\n")
	}

	sb.WriteString("### Unmatched Findings\n\n")
	if len(result.UnmatchedFindings) > 0 {
		sb.WriteString("| Reviewer | Sharp Edge | Severity | Finding |\n")
		sb.WriteString("|----------|-----------|----------|---------|\n")
		for _, f := range result.UnmatchedFindings {
			fmt.Fprintf(sb, "| %s | %s | %s | %s |\n",
				f.ReviewerID, f.SharpEdgeID, f.Severity, f.Title)
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("All findings participated in at least one detected interaction.\n\n")
	}

	sb.WriteString("### Interaction Detection Summary\n\n")
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("|--------|-------|\n")
	fmt.Fprintf(sb, "| Rules evaluated | %d |\n", result.RulesEvaluated)
	fmt.Fprintf(sb, "| Interactions detected | %d |\n", len(result.DetectedInteractions))
	fmt.Fprintf(sb, "| Findings matched | %d of %d |\n", matchedCount, result.FindingsTotal)
	fmt.Fprintf(sb, "| Findings unmatched | %d |\n", len(result.UnmatchedFindings))
	sb.WriteString("\n")
}

// algebraDescription returns a brief explanation of the algebra type for markdown.
func algebraDescription(algebra string) string {
	switch algebra {
	case "additive":
		return "findings compound — combined severity exceeds either alone"
	case "multiplicative":
		return "upstream finding multiplies downstream impact"
	case "gating":
		return "upstream finding invalidates downstream findings"
	case "negating":
		return "findings mitigate each other — combined severity is less than the worst alone"
	default:
		return "combined effect"
	}
}

// detectedInteractionsJSON is the top-level structure of detected-interactions.json.
type detectedInteractionsJSON struct {
	SchemaVersion string              `json:"schema_version"`
	RulesVersion  string              `json:"rules_version"`
	GeneratedAt   string              `json:"generated_at"`
	Summary       detectionSummary    `json:"summary"`
	Interactions  []interactionRecord `json:"interactions"`
	Unmatched     []findingRecord     `json:"unmatched_findings"`
}

type detectionSummary struct {
	RulesEvaluated       int `json:"rules_evaluated"`
	FindingsTotal        int `json:"findings_total"`
	InteractionsDetected int `json:"interactions_detected"`
	FindingsMatched      int `json:"findings_matched"`
	FindingsUnmatched    int `json:"findings_unmatched"`
}

type interactionRecord struct {
	RuleID          string          `json:"rule_id"`
	RuleName        string          `json:"rule_name"`
	Algebra         string          `json:"algebra"`
	Severity        string          `json:"severity"`
	Layer           int             `json:"layer"`
	Tags            []string        `json:"tags"`
	Message         string          `json:"message"`
	MatchedFindings []findingRecord `json:"matched_findings"`
}

type findingRecord struct {
	ReviewerID  string `json:"reviewer_id"`
	FindingID   string `json:"finding_id"`
	SharpEdgeID string `json:"sharp_edge_id,omitempty"`
	Severity    string `json:"severity"`
	Category    string `json:"category,omitempty"`
	Title       string `json:"title,omitempty"`
}

// writeDetectedInteractionsJSON writes detected-interactions.json to teamDir.
// Returns the absolute path written, or ("", error) on failure.
func writeDetectedInteractionsJSON(teamDir string, result DetectionResult, rules InteractionRulesConfig) (string, error) {
	matchedCount := result.FindingsTotal - len(result.UnmatchedFindings)

	interactions := make([]interactionRecord, 0, len(result.DetectedInteractions))
	for _, d := range result.DetectedInteractions {
		matched := make([]findingRecord, 0, len(d.MatchedFindings))
		for _, f := range d.MatchedFindings {
			matched = append(matched, findingRecord{
				ReviewerID:  f.ReviewerID,
				FindingID:   f.FindingID,
				SharpEdgeID: f.SharpEdgeID,
				Severity:    f.Severity,
				Category:    f.Category,
				Title:       f.Title,
			})
		}
		tags := d.Tags
		if tags == nil {
			tags = []string{}
		}
		interactions = append(interactions, interactionRecord{
			RuleID:          d.RuleID,
			RuleName:        d.RuleName,
			Algebra:         d.Algebra,
			Severity:        d.Severity,
			Layer:           d.Layer,
			Tags:            tags,
			Message:         d.Message,
			MatchedFindings: matched,
		})
	}

	unmatched := make([]findingRecord, 0, len(result.UnmatchedFindings))
	for _, f := range result.UnmatchedFindings {
		unmatched = append(unmatched, findingRecord{
			ReviewerID:  f.ReviewerID,
			FindingID:   f.FindingID,
			SharpEdgeID: f.SharpEdgeID,
			Severity:    f.Severity,
			Category:    f.Category,
			Title:       f.Title,
		})
	}

	out := detectedInteractionsJSON{
		SchemaVersion: "1.0.0",
		RulesVersion:  rules.Version,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Summary: detectionSummary{
			RulesEvaluated:       result.RulesEvaluated,
			FindingsTotal:        result.FindingsTotal,
			InteractionsDetected: len(result.DetectedInteractions),
			FindingsMatched:      matchedCount,
			FindingsUnmatched:    len(result.UnmatchedFindings),
		},
		Interactions: interactions,
		Unmatched:    unmatched,
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal detected-interactions: %w", err)
	}

	outPath := filepath.Join(teamDir, "detected-interactions.json")
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		return "", fmt.Errorf("write detected-interactions.json: %w", err)
	}
	return outPath, nil
}
