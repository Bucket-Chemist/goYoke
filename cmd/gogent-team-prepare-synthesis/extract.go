package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

// Size limits for extraction loops (FIX #7)
const (
	maxRootCauses          = 50
	maxNovelApproaches     = 20
	maxOpenQuestions       = 30
	maxIssues              = 100
	maxRecommendations     = 50
	maxFailureModes        = 50
	maxFirstPrinciples     = 30
	maxTradeoffs           = 20
	maxAssumptionsSurfaced = 30
)

// Schema version tracking (FIX #17)
// expectedEinsteinFields documents the schema fields this binary extracts.
// If the upstream schema adds new top-level fields, the DisallowUnknownFields
// check (fix #6) will log a warning, alerting maintainers to update.
//
// Schema source: .claude/schemas/teams/stdin-stdout/braintrust-einstein.json
// Last synced: 2026-02-07
const einsteinSchemaNote = "synced with braintrust-einstein.json 2026-02-07"

// Schema source: .claude/schemas/teams/stdin-stdout/braintrust-staff-architect.json
// Last synced: 2026-02-07
const staffArchSchemaNote = "synced with braintrust-staff-architect.json 2026-02-07"

// extractEinstein reads and parses Einstein's stdout JSON file.
// Returns extracted sections with graceful fallback for missing/malformed data.
func extractEinstein(filePath string) EinsteinSections {
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("WARNING: Failed to read Einstein output (%s): %v", filePath, err)
		return fallbackEinsteinSections("file unavailable")
	}

	var output EinsteinOutput
	if err := json.Unmarshal(data, &output); err != nil {
		log.Printf("WARNING: Failed to parse Einstein JSON (%s): %v", filePath, err)
		return fallbackEinsteinSections("could not parse JSON")
	}

	// Check for unknown fields (schema drift detection)
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var check EinsteinOutput
	if err := dec.Decode(&check); err != nil {
		log.Printf("WARNING: Einstein JSON contains unknown fields (%s): %v", filePath, err)
	}

	// Check for failed status
	if output.Status == "failed" {
		log.Printf("WARNING: Einstein reported status 'failed' (%s)", filePath)
		return fallbackEinsteinSections("agent reported failure")
	}

	sections := EinsteinSections{
		Status:           output.Status,
		ExecutiveSummary: output.ExecutiveSummary,
		HandoffNotes:     output.HandoffNotes,
	}

	// Extract root causes
	if output.RootCauseAnalysis != nil {
		for i, cause := range output.RootCauseAnalysis.IdentifiedCauses {
			if i >= maxRootCauses {
				log.Printf("WARNING: Truncating root causes at %d (total: %d)", maxRootCauses, len(output.RootCauseAnalysis.IdentifiedCauses))
				break
			}
			sections.RootCauses = append(sections.RootCauses,
				fmt.Sprintf("**%s** (confidence: %s)\n  - Evidence: %s\n  - Scope: %s",
					cause.Cause, cause.Confidence, cause.Evidence, strings.Join(cause.AffectedScope, ", ")))
		}
	}

	// Extract frameworks
	if output.ConceptualFramework != nil {
		frameworks := fmt.Sprintf("**%s**\n%s\nKey insights:\n",
			output.ConceptualFramework.FrameworkName,
			output.ConceptualFramework.Description)
		for _, insight := range output.ConceptualFramework.KeyInsights {
			frameworks += fmt.Sprintf("  - %s\n", insight)
		}
		sections.Frameworks = append(sections.Frameworks, frameworks)
	}

	// Extract first principles analysis (FIX #2)
	if output.FirstPrinciplesAnalysis != nil {
		for i, ac := range output.FirstPrinciplesAnalysis.AssumptionsChallenged {
			if i >= maxFirstPrinciples {
				log.Printf("WARNING: Truncating assumptions challenged at %d (total: %d)", maxFirstPrinciples, len(output.FirstPrinciplesAnalysis.AssumptionsChallenged))
				break
			}
			sections.FirstPrinciples = append(sections.FirstPrinciples,
				fmt.Sprintf("**%s** (validity: %s)\n  - Evidence: %s\n  - If wrong: %s",
					ac.Assumption, ac.Validity, ac.Evidence, ac.ImplicationIfWrong))
		}
		for i, constraint := range output.FirstPrinciplesAnalysis.FundamentalConstraints {
			if i >= maxFirstPrinciples {
				log.Printf("WARNING: Truncating fundamental constraints at %d (total: %d)", maxFirstPrinciples, len(output.FirstPrinciplesAnalysis.FundamentalConstraints))
				break
			}
			sections.FirstPrinciples = append(sections.FirstPrinciples,
				fmt.Sprintf("**[Constraint]** %s", constraint))
		}
	}

	// Extract novel approaches
	for i, approach := range output.NovelApproaches {
		if i >= maxNovelApproaches {
			log.Printf("WARNING: Truncating novel approaches at %d (total: %d)", maxNovelApproaches, len(output.NovelApproaches))
			break
		}
		approachText := fmt.Sprintf("**%s**\n%s\nFeasibility: %s\nPros: %s\nCons: %s\nRisks: %s",
			approach.Approach,
			approach.Rationale,
			approach.Feasibility,
			strings.Join(approach.Tradeoffs.Pros, ", "),
			strings.Join(approach.Tradeoffs.Cons, ", "),
			strings.Join(approach.Tradeoffs.Risks, ", "))
		sections.NovelApproaches = append(sections.NovelApproaches, approachText)
	}

	// Extract theoretical tradeoffs (FIX #2)
	for i, tradeoff := range output.TheoreticalTradeoffs {
		if i >= maxTradeoffs {
			log.Printf("WARNING: Truncating theoretical tradeoffs at %d (total: %d)", maxTradeoffs, len(output.TheoreticalTradeoffs))
			break
		}
		sections.TheoreticalTradeoffs = append(sections.TheoreticalTradeoffs,
			fmt.Sprintf("**%s**\n  - Option A: %s\n  - Option B: %s\n  - Recommendation: %s",
				tradeoff.Dimension, tradeoff.OptionA, tradeoff.OptionB, tradeoff.Recommendation))
	}

	// Extract assumptions surfaced (FIX #2)
	for i, assumption := range output.AssumptionsSurfaced {
		if i >= maxAssumptionsSurfaced {
			log.Printf("WARNING: Truncating assumptions surfaced at %d (total: %d)", maxAssumptionsSurfaced, len(output.AssumptionsSurfaced))
			break
		}
		sections.AssumptionsSurfaced = append(sections.AssumptionsSurfaced,
			fmt.Sprintf("**%s** (source: %s)\n  - Risk if false: %s\n  - Validation: %s",
				assumption.Assumption, assumption.Source, assumption.RiskIfFalse, assumption.ValidationMethod))
	}

	// Extract open questions
	for i, question := range output.OpenQuestions {
		if i >= maxOpenQuestions {
			log.Printf("WARNING: Truncating open questions at %d (total: %d)", maxOpenQuestions, len(output.OpenQuestions))
			break
		}
		sections.OpenQuestions = append(sections.OpenQuestions,
			fmt.Sprintf("**%s** (importance: %s)\n  - Investigation: %s",
				question.Question, question.Importance, question.SuggestedInvestigation))
	}

	// Check for empty content
	if sections.ExecutiveSummary == "" &&
		len(sections.RootCauses) == 0 &&
		len(sections.Frameworks) == 0 &&
		len(sections.NovelApproaches) == 0 &&
		len(sections.OpenQuestions) == 0 {
		log.Printf("WARNING: Einstein output is empty (%s)", filePath)
		return fallbackEinsteinSections("empty content")
	}

	return sections
}

// extractStaffArch reads and parses Staff-Architect's stdout JSON file.
// Returns extracted sections with graceful fallback for missing/malformed data.
func extractStaffArch(filePath string) StaffArchSections {
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("WARNING: Failed to read Staff-Architect output (%s): %v", filePath, err)
		return fallbackStaffArchSections("file unavailable")
	}

	var output StaffArchOutput
	if err := json.Unmarshal(data, &output); err != nil {
		log.Printf("WARNING: Failed to parse Staff-Architect JSON (%s): %v", filePath, err)
		return fallbackStaffArchSections("could not parse JSON")
	}

	// Check for unknown fields (schema drift detection) (FIX #6)
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var check StaffArchOutput
	if err := dec.Decode(&check); err != nil {
		log.Printf("WARNING: Staff-Architect JSON contains unknown fields (%s): %v", filePath, err)
	}

	// Check status (FIX #4)
	if output.Status == "failed" {
		log.Printf("WARNING: Staff-Architect reported status 'failed' (%s)", filePath)
		return fallbackStaffArchSections("agent reported failure")
	}

	sections := StaffArchSections{
		Status:       output.Status,
		HandoffNotes: output.HandoffNotes,
	}

	// Extract executive verdict
	if output.ExecutiveAssessment != nil {
		sections.ExecutiveVerdict = fmt.Sprintf("**Verdict:** %s (confidence: %s)\n**Summary:** %s\n**Issue Counts:** Critical=%d, Major=%d, Minor=%d",
			output.ExecutiveAssessment.Verdict,
			output.ExecutiveAssessment.Confidence,
			output.ExecutiveAssessment.Summary,
			output.ExecutiveAssessment.IssueCounts.Critical,
			output.ExecutiveAssessment.IssueCounts.Major,
			output.ExecutiveAssessment.IssueCounts.Minor)
	}

	// Extract issues by severity (FIX #7)
	issueCount := 0
	for _, issue := range output.IssueRegister {
		if issueCount >= maxIssues {
			log.Printf("WARNING: Truncating issues at %d (total: %d)", maxIssues, len(output.IssueRegister))
			break
		}
		issueText := fmt.Sprintf("**%s: %s** (layer: %s)\n%s\n  - Evidence: %s\n  - Impact: %s\n  - Recommendation: %s",
			issue.ID, issue.Title, issue.Layer, issue.Description, issue.Evidence, issue.Impact, issue.Recommendation)

		switch issue.Severity {
		case "critical":
			sections.CriticalIssues = append(sections.CriticalIssues, issueText)
		case "major":
			sections.MajorIssues = append(sections.MajorIssues, issueText)
		case "minor":
			sections.MinorIssues = append(sections.MinorIssues, issueText)
		}
		issueCount++
	}

	// Extract commendations (FIX #5)
	sections.Commendations = output.Commendations

	// Extract failure modes (FIX #5)
	for i, fm := range output.FailureModeAnalysis {
		if i >= maxFailureModes {
			log.Printf("WARNING: Truncating failure modes at %d (total: %d)", maxFailureModes, len(output.FailureModeAnalysis))
			break
		}
		sections.FailureModes = append(sections.FailureModes,
			fmt.Sprintf("**%s** (probability: %s, impact: %s)\n  - Detection: %s\n  - Mitigation: %s",
				fm.Scenario, fm.Probability, fm.Impact, fm.Detection, fm.Mitigation))
	}

	// Extract recommendations (FIX #1 + #7)
	recCount := 0
	if output.Recommendations != nil {
		for _, rec := range output.Recommendations.HighPriority {
			if recCount >= maxRecommendations {
				log.Printf("WARNING: Truncating recommendations at %d", maxRecommendations)
				break
			}
			sections.Recommendations = append(sections.Recommendations,
				fmt.Sprintf("**[HIGH]** %s\n  - Rationale: %s\n  - Effort: %s", rec.Action, rec.Rationale, rec.Effort))
			recCount++
		}
		for _, rec := range output.Recommendations.MediumPriority {
			if recCount >= maxRecommendations {
				log.Printf("WARNING: Truncating recommendations at %d", maxRecommendations)
				break
			}
			sections.Recommendations = append(sections.Recommendations,
				fmt.Sprintf("**[MEDIUM]** %s\n  - Rationale: %s\n  - Effort: %s", rec.Action, rec.Rationale, rec.Effort))
			recCount++
		}
		// FIX #1: Extract low-priority recommendations
		for _, rec := range output.Recommendations.LowPriority {
			if recCount >= maxRecommendations {
				log.Printf("WARNING: Truncating recommendations at %d", maxRecommendations)
				break
			}
			sections.Recommendations = append(sections.Recommendations,
				fmt.Sprintf("**[LOW]** %s\n  - Rationale: %s\n  - Effort: %s", rec.Action, rec.Rationale, rec.Effort))
			recCount++
		}
	}

	// Extract sign-off conditions
	if output.SignOff != nil {
		sections.SignOffConditions = output.SignOff.Conditions
	}

	return sections
}

// fallbackEinsteinSections returns a fallback structure when Einstein data is unavailable.
// FIX #8: Changed prefix from "(fallback:" to "(unavailable:" for consistency
func fallbackEinsteinSections(reason string) EinsteinSections {
	return EinsteinSections{
		Status:               "unavailable",
		ExecutiveSummary:     fmt.Sprintf("(unavailable: Einstein analysis %s)", reason),
		RootCauses:           []string{fmt.Sprintf("(unavailable: %s)", reason)},
		Frameworks:           []string{fmt.Sprintf("(unavailable: %s)", reason)},
		FirstPrinciples:      []string{fmt.Sprintf("(unavailable: %s)", reason)},
		NovelApproaches:      []string{fmt.Sprintf("(unavailable: %s)", reason)},
		TheoreticalTradeoffs: []string{fmt.Sprintf("(unavailable: %s)", reason)},
		AssumptionsSurfaced:  []string{fmt.Sprintf("(unavailable: %s)", reason)},
		OpenQuestions:        []string{fmt.Sprintf("(unavailable: %s)", reason)},
		HandoffNotes:         fmt.Sprintf("(unavailable: %s)", reason),
	}
}

// fallbackStaffArchSections returns a fallback structure when Staff-Architect data is unavailable.
// FIX #8: Changed prefix from "(fallback:" to "(unavailable:" for consistency
func fallbackStaffArchSections(reason string) StaffArchSections {
	return StaffArchSections{
		Status:            "unavailable",
		ExecutiveVerdict:  fmt.Sprintf("(unavailable: Staff-Architect review %s)", reason),
		CriticalIssues:    []string{fmt.Sprintf("(unavailable: %s)", reason)},
		MajorIssues:       []string{fmt.Sprintf("(unavailable: %s)", reason)},
		MinorIssues:       []string{fmt.Sprintf("(unavailable: %s)", reason)},
		Commendations:     []string{fmt.Sprintf("(unavailable: %s)", reason)},
		FailureModes:      []string{fmt.Sprintf("(unavailable: %s)", reason)},
		Recommendations:   []string{fmt.Sprintf("(unavailable: %s)", reason)},
		SignOffConditions: []string{fmt.Sprintf("(unavailable: %s)", reason)},
		HandoffNotes:      fmt.Sprintf("(unavailable: %s)", reason),
	}
}
