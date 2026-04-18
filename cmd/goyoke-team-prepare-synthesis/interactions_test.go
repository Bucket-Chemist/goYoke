package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- helpers ---

// makeReviewerJSON builds a minimal reviewer stdout JSON for testing ExtractFindings.
func makeReviewerJSON(t *testing.T, findings []reviewerFinding) []byte {
	t.Helper()
	out := reviewerStdout{Status: "completed", Findings: findings}
	b, err := json.Marshal(out)
	require.NoError(t, err)
	return b
}

// makeRule builds a minimal InteractionRule for testing.
func makeRule(id, algebra, condType string, matchers []FindingMatcher) InteractionRule {
	return InteractionRule{
		ID:              id,
		Name:            id,
		Algebra:         algebra,
		Condition:       RuleCondition{Type: condType, Matchers: matchers},
		MessageTemplate: "{finding_a.reviewer}/{finding_a.sharp_edge_id}",
		Layer:           1,
	}
}

// makeFinding builds a minimal ExtractedFinding for testing.
func makeFinding(reviewerID, sharpEdgeID, severity, category, message string) ExtractedFinding {
	return ExtractedFinding{
		ReviewerID:  reviewerID,
		FindingID:   "F-1",
		SharpEdgeID: sharpEdgeID,
		Severity:    severity,
		Category:    category,
		Message:     message,
	}
}

// --- Pattern matching tests ---

func TestGlobMatch_ExactID(t *testing.T) {
	assert.True(t, globMatch("proteomics-fdr-global-only", "proteomics-fdr-global-only"))
}

func TestGlobMatch_Wildcard(t *testing.T) {
	assert.True(t, globMatch("proteomics-fdr-*", "proteomics-fdr-global-only"))
	assert.True(t, globMatch("proteomics-fdr-*", "proteomics-fdr-multistage-dependent"))
}

func TestGlobMatch_DoubleWildcard(t *testing.T) {
	assert.True(t, globMatch("*-version-*", "proteogenomics-version-vep-pyensembl"))
	assert.True(t, globMatch("*-version-*", "genomics-version-build38"))
}

func TestGlobMatch_NoMatch(t *testing.T) {
	assert.False(t, globMatch("proteomics-fdr-*", "proteomics-quant-tmt-no-compression"))
	assert.False(t, globMatch("proteomics-fdr-*", "genomics-fdr-global"))
}

func TestGlobMatch_ReviewerWildcard(t *testing.T) {
	assert.True(t, globMatch("*", "proteomics-reviewer"))
	assert.True(t, globMatch("*", "genomics-reviewer"))
	assert.True(t, globMatch("*", "bioinformatician-reviewer"))
}

func TestGlobMatch_EmptyID(t *testing.T) {
	assert.False(t, globMatch("proteomics-*", ""))
}

// --- Severity comparison tests ---

func TestSeverity_ExactMatch(t *testing.T) {
	assert.True(t, severityAtLeast("warning", "warning"))
	assert.True(t, severityAtLeast("critical", "critical"))
	assert.True(t, severityAtLeast("info", "info"))
}

func TestSeverity_HigherPasses(t *testing.T) {
	assert.True(t, severityAtLeast("critical", "warning"))
	assert.True(t, severityAtLeast("critical", "info"))
	assert.True(t, severityAtLeast("warning", "info"))
}

func TestSeverity_LowerFails(t *testing.T) {
	assert.False(t, severityAtLeast("info", "warning"))
	assert.False(t, severityAtLeast("info", "critical"))
	assert.False(t, severityAtLeast("warning", "critical"))
}

func TestSeverity_NoMinimum(t *testing.T) {
	assert.True(t, severityAtLeast("info", ""))
	assert.True(t, severityAtLeast("warning", ""))
	assert.True(t, severityAtLeast("critical", ""))
}

// --- Severity mapping tests ---

func TestExtractFindings_SeverityMapping(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"CRITICAL", "critical"},
		{"HIGH", "warning"},
		{"MEDIUM", "warning"},
		{"LOW", "info"},
		{"INFO", "info"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			raw := makeReviewerJSON(t, []reviewerFinding{
				{ID: "F-1", Severity: tc.input, Message: "test"},
			})
			findings := ExtractFindings("test-reviewer", raw)
			require.Len(t, findings, 1)
			assert.Equal(t, tc.expected, findings[0].Severity)
		})
	}
}

// --- Rule matching tests ---

func TestRequiresAll_BothPresent(t *testing.T) {
	rule := makeRule("r1", "additive", "requires_all", []FindingMatcher{
		{ReviewerPattern: "genomics-reviewer", SharpEdgePattern: "genomics-ref-*"},
		{ReviewerPattern: "proteomics-reviewer", SharpEdgePattern: "proteomics-fdr-*"},
	})
	findings := []ExtractedFinding{
		makeFinding("genomics-reviewer", "genomics-ref-wrong-build", "warning", "ref", "msg"),
		makeFinding("proteomics-reviewer", "proteomics-fdr-global-only", "warning", "fdr", "msg"),
	}

	result := DetectInteractions(InteractionRulesConfig{Rules: []InteractionRule{rule}}, findings)
	require.Len(t, result.DetectedInteractions, 1)
	assert.Equal(t, "r1", result.DetectedInteractions[0].RuleID)
	assert.Len(t, result.DetectedInteractions[0].MatchedFindings, 2)
}

func TestRequiresAll_OneMissing(t *testing.T) {
	rule := makeRule("r1", "additive", "requires_all", []FindingMatcher{
		{ReviewerPattern: "genomics-reviewer", SharpEdgePattern: "genomics-ref-*"},
		{ReviewerPattern: "proteomics-reviewer", SharpEdgePattern: "proteomics-fdr-*"},
	})
	// Only genomics finding present
	findings := []ExtractedFinding{
		makeFinding("genomics-reviewer", "genomics-ref-wrong-build", "warning", "ref", "msg"),
	}

	result := DetectInteractions(InteractionRulesConfig{Rules: []InteractionRule{rule}}, findings)
	assert.Empty(t, result.DetectedInteractions)
}

func TestRequiresAll_SeverityBelowMin(t *testing.T) {
	rule := makeRule("r1", "additive", "requires_all", []FindingMatcher{
		{ReviewerPattern: "proteomics-reviewer", SharpEdgePattern: "proteomics-fdr-*", SeverityMinimum: "warning"},
	})
	// Finding exists but severity is below minimum
	findings := []ExtractedFinding{
		makeFinding("proteomics-reviewer", "proteomics-fdr-global-only", "info", "fdr", "msg"),
	}

	result := DetectInteractions(InteractionRulesConfig{Rules: []InteractionRule{rule}}, findings)
	assert.Empty(t, result.DetectedInteractions)
}

func TestRequiresAny_OnePresent(t *testing.T) {
	rule := makeRule("r1", "gating", "requires_any", []FindingMatcher{
		{ReviewerPattern: "genomics-reviewer", SharpEdgePattern: "genomics-ref-*"},
		{ReviewerPattern: "proteogenomics-reviewer", SharpEdgePattern: "proteogenomics-version-*"},
	})
	// Only one of the two is present
	findings := []ExtractedFinding{
		makeFinding("genomics-reviewer", "genomics-ref-wrong-build", "critical", "ref", "msg"),
	}

	result := DetectInteractions(InteractionRulesConfig{Rules: []InteractionRule{rule}}, findings)
	require.Len(t, result.DetectedInteractions, 1)
	assert.Equal(t, "r1", result.DetectedInteractions[0].RuleID)
}

func TestRequiresAny_NonePresent(t *testing.T) {
	rule := makeRule("r1", "gating", "requires_any", []FindingMatcher{
		{ReviewerPattern: "genomics-reviewer", SharpEdgePattern: "genomics-ref-*"},
		{ReviewerPattern: "proteogenomics-reviewer", SharpEdgePattern: "proteogenomics-version-*"},
	})
	// Neither finding is present
	findings := []ExtractedFinding{
		makeFinding("proteomics-reviewer", "proteomics-fdr-global-only", "warning", "fdr", "msg"),
	}

	result := DetectInteractions(InteractionRulesConfig{Rules: []InteractionRule{rule}}, findings)
	assert.Empty(t, result.DetectedInteractions)
}

func TestRequiresAll_MultipleMatchesFirstWins(t *testing.T) {
	// Matcher 1 is satisfied by three findings; matcher 2 by one.
	// The FIRST finding in slice order should win for matcher 1.
	rule := makeRule("r1", "additive", "requires_all", []FindingMatcher{
		{ReviewerPattern: "proteomics-reviewer", SharpEdgePattern: "proteomics-fdr-*"},
		{ReviewerPattern: "genomics-reviewer", SharpEdgePattern: "genomics-ref-*"},
	})
	findings := []ExtractedFinding{
		makeFinding("proteomics-reviewer", "proteomics-fdr-a", "info", "fdr", "first"),
		makeFinding("proteomics-reviewer", "proteomics-fdr-b", "warning", "fdr", "second"),
		makeFinding("proteomics-reviewer", "proteomics-fdr-c", "critical", "fdr", "third"),
		makeFinding("genomics-reviewer", "genomics-ref-build38", "warning", "ref", "only"),
	}

	result := DetectInteractions(InteractionRulesConfig{Rules: []InteractionRule{rule}}, findings)
	require.Len(t, result.DetectedInteractions, 1)
	// First matched finding for matcher 1 should be "proteomics-fdr-a" (index 0)
	assert.Equal(t, "proteomics-fdr-a", result.DetectedInteractions[0].MatchedFindings[0].SharpEdgeID)
}

func TestFindingPresent_OnlyChecksReviewer(t *testing.T) {
	// When finding_present: true, only reviewer_pattern is checked.
	// Even if sharp_edge_id, severity, category don't match, the finding is accepted.
	rule := makeRule("r1", "gating", "requires_all", []FindingMatcher{
		{ReviewerPattern: "mass-spec-reviewer", SharpEdgePattern: "mass-spec-critical-*", SeverityMinimum: "critical"},
		{ReviewerPattern: "proteomics-reviewer", FindingPresent: true},
	})

	// Proteomics finding has a mismatched sharp_edge_id and low severity,
	// but FindingPresent:true means it should still match on reviewer alone.
	findings := []ExtractedFinding{
		makeFinding("mass-spec-reviewer", "mass-spec-critical-snr", "critical", "acquisition", "msg"),
		makeFinding("proteomics-reviewer", "some-unrelated-edge", "info", "stats", "msg"),
	}

	result := DetectInteractions(InteractionRulesConfig{Rules: []InteractionRule{rule}}, findings)
	require.Len(t, result.DetectedInteractions, 1)

	// Verify the proteomics finding was matched despite mismatched sharp_edge_id and low severity
	var proteomicsMatched bool
	for _, f := range result.DetectedInteractions[0].MatchedFindings {
		if f.ReviewerID == "proteomics-reviewer" {
			proteomicsMatched = true
		}
	}
	assert.True(t, proteomicsMatched, "proteomics finding should be matched via finding_present:true")

	// A finding_present matcher with no reviewer match should NOT match
	ruleNonMatch := makeRule("r2", "gating", "requires_all", []FindingMatcher{
		{ReviewerPattern: "mass-spec-reviewer", SharpEdgePattern: "mass-spec-critical-*", SeverityMinimum: "critical"},
		{ReviewerPattern: "bioinformatician-reviewer", FindingPresent: true},
	})
	result2 := DetectInteractions(InteractionRulesConfig{Rules: []InteractionRule{ruleNonMatch}}, findings)
	assert.Empty(t, result2.DetectedInteractions, "bioinformatician finding absent so rule should not fire")
}

// --- Edge case tests ---

func TestEmptyFindings(t *testing.T) {
	rules := InteractionRulesConfig{Rules: []InteractionRule{
		makeRule("r1", "additive", "requires_all", []FindingMatcher{
			{ReviewerPattern: "proteomics-reviewer", SharpEdgePattern: "proteomics-fdr-*"},
		}),
	}}

	result := DetectInteractions(rules, []ExtractedFinding{})
	assert.Empty(t, result.DetectedInteractions)
	assert.Empty(t, result.UnmatchedFindings)
	assert.Equal(t, 1, result.RulesEvaluated)
	assert.Equal(t, 0, result.FindingsTotal)
}

func TestEmptyRules(t *testing.T) {
	findings := []ExtractedFinding{
		makeFinding("proteomics-reviewer", "proteomics-fdr-global-only", "warning", "fdr", "msg"),
		makeFinding("genomics-reviewer", "genomics-ref-build38", "warning", "ref", "msg"),
	}

	result := DetectInteractions(InteractionRulesConfig{}, findings)
	assert.Empty(t, result.DetectedInteractions)
	// All findings should be unmatched
	assert.Len(t, result.UnmatchedFindings, 2)
	assert.Equal(t, 0, result.RulesEvaluated)
	assert.Equal(t, 2, result.FindingsTotal)
}

func TestFindingWithoutSharpEdge(t *testing.T) {
	// A finding with empty SharpEdgeID cannot match sharp_edge_pattern rules.
	sharpEdgeRule := makeRule("r-sharp", "additive", "requires_all", []FindingMatcher{
		{ReviewerPattern: "proteomics-reviewer", SharpEdgePattern: "proteomics-*"},
	})
	// But it CAN match finding_category or finding_contains rules.
	categoryRule := makeRule("r-category", "additive", "requires_any", []FindingMatcher{
		{ReviewerPattern: "proteomics-reviewer", FindingCategory: "fdr-control"},
	})
	containsRule := makeRule("r-contains", "additive", "requires_any", []FindingMatcher{
		{ReviewerPattern: "proteomics-reviewer", FindingContains: "FDR rate"},
	})

	findingNoSharpEdge := ExtractedFinding{
		ReviewerID:  "proteomics-reviewer",
		FindingID:   "PROT-1",
		SharpEdgeID: "", // empty — no sharp edge
		Severity:    "warning",
		Category:    "fdr-control",
		Message:     "FDR rate may be inflated",
	}

	rules := InteractionRulesConfig{Rules: []InteractionRule{sharpEdgeRule, categoryRule, containsRule}}
	result := DetectInteractions(rules, []ExtractedFinding{findingNoSharpEdge})

	// sharp_edge_pattern rule should NOT fire (no sharp edge ID)
	ruleIDs := make(map[string]bool)
	for _, d := range result.DetectedInteractions {
		ruleIDs[d.RuleID] = true
	}
	assert.False(t, ruleIDs["r-sharp"], "sharp_edge_pattern rule must not match finding without sharp_edge_id")
	assert.True(t, ruleIDs["r-category"], "finding_category rule should match")
	assert.True(t, ruleIDs["r-contains"], "finding_contains rule should match")
}

func TestSameReviewerBothSides(t *testing.T) {
	// A rule where both matchers target the same reviewer (e.g., proteomics MBR + proteomics stats).
	rule := makeRule("mbr-stats", "additive", "requires_all", []FindingMatcher{
		{ReviewerPattern: "proteomics-reviewer", SharpEdgePattern: "proteomics-mbr-*"},
		{ReviewerPattern: "proteomics-reviewer", SharpEdgePattern: "proteomics-stats-*"},
	})
	findings := []ExtractedFinding{
		makeFinding("proteomics-reviewer", "proteomics-mbr-enabled", "warning", "identification", "MBR enabled"),
		makeFinding("proteomics-reviewer", "proteomics-stats-no-correction", "warning", "statistics", "No correction"),
	}

	result := DetectInteractions(InteractionRulesConfig{Rules: []InteractionRule{rule}}, findings)
	require.Len(t, result.DetectedInteractions, 1)
	assert.Equal(t, "mbr-stats", result.DetectedInteractions[0].RuleID)
	assert.Len(t, result.DetectedInteractions[0].MatchedFindings, 2)
}

func TestDuplicateSharpEdge(t *testing.T) {
	// Two findings with the same sharp_edge_id from different reviewers.
	// A wildcard rule for any reviewer: the FIRST finding in slice order is used.
	rule := makeRule("r1", "additive", "requires_all", []FindingMatcher{
		{ReviewerPattern: "*", SharpEdgePattern: "fdr-global-*"},
	})
	findings := []ExtractedFinding{
		makeFinding("proteomics-reviewer", "fdr-global-only", "warning", "fdr", "first"),
		makeFinding("genomics-reviewer", "fdr-global-only", "critical", "fdr", "second"),
	}

	result := DetectInteractions(InteractionRulesConfig{Rules: []InteractionRule{rule}}, findings)
	require.Len(t, result.DetectedInteractions, 1)
	// First finding should be used
	assert.Equal(t, "proteomics-reviewer", result.DetectedInteractions[0].MatchedFindings[0].ReviewerID)
}

func TestMalformedJSON_GracefulDegradation(t *testing.T) {
	// ExtractFindings with malformed JSON must return empty slice, no panic.
	result := ExtractFindings("proteomics-reviewer", []byte("{invalid json}"))
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestMissingRulesFile_GracefulDegradation(t *testing.T) {
	// LoadRules with non-existent path must return empty config and nil error.
	cfg, err := LoadRules("/nonexistent/path/to/interaction-rules.json")
	require.NoError(t, err)
	assert.Empty(t, cfg.Rules)
}

func TestFindingContains_CaseInsensitive(t *testing.T) {
	rule := makeRule("r1", "additive", "requires_any", []FindingMatcher{
		{ReviewerPattern: "proteomics-reviewer", FindingContains: "fdr rate"},
	})

	tests := []struct {
		name    string
		message string
		fires   bool
	}{
		{"lowercase", "the fdr rate is inflated", true},
		{"uppercase", "The FDR RATE is inflated", true},
		{"mixed case", "The FDR Rate may be 3x nominal", true},
		{"no match", "quantification issue detected", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			findings := []ExtractedFinding{
				makeFinding("proteomics-reviewer", "proteomics-fdr-x", "warning", "fdr", tc.message),
			}
			result := DetectInteractions(InteractionRulesConfig{Rules: []InteractionRule{rule}}, findings)
			if tc.fires {
				assert.Len(t, result.DetectedInteractions, 1, "rule should fire for message: %q", tc.message)
			} else {
				assert.Empty(t, result.DetectedInteractions, "rule should not fire for message: %q", tc.message)
			}
		})
	}
}

// --- Algebra tests ---

func TestAlgebra_Additive(t *testing.T) {
	// Two warning findings, additive algebra → escalates to critical.
	rule := makeRule("r1", "additive", "requires_all", []FindingMatcher{
		{ReviewerPattern: "genomics-reviewer"},
		{ReviewerPattern: "proteomics-reviewer"},
	})
	findings := []ExtractedFinding{
		makeFinding("genomics-reviewer", "g1", "warning", "cat", "msg"),
		makeFinding("proteomics-reviewer", "p1", "warning", "cat", "msg"),
	}

	result := DetectInteractions(InteractionRulesConfig{Rules: []InteractionRule{rule}}, findings)
	require.Len(t, result.DetectedInteractions, 1)
	assert.Equal(t, "critical", result.DetectedInteractions[0].Severity,
		"additive: two warnings should escalate to critical")
}

func TestAlgebra_Multiplicative(t *testing.T) {
	// warning + info → multiplicative → max = warning.
	rule := makeRule("r1", "multiplicative", "requires_all", []FindingMatcher{
		{ReviewerPattern: "genomics-reviewer"},
		{ReviewerPattern: "proteomics-reviewer"},
	})
	findings := []ExtractedFinding{
		makeFinding("genomics-reviewer", "g1", "warning", "cat", "msg"),
		makeFinding("proteomics-reviewer", "p1", "info", "cat", "msg"),
	}

	result := DetectInteractions(InteractionRulesConfig{Rules: []InteractionRule{rule}}, findings)
	require.Len(t, result.DetectedInteractions, 1)
	assert.Equal(t, "warning", result.DetectedInteractions[0].Severity,
		"multiplicative: max(warning, info) should be warning")
}

func TestAlgebra_Gating(t *testing.T) {
	// critical + warning → gating → upstream gates downstream → critical.
	rule := makeRule("r1", "gating", "requires_all", []FindingMatcher{
		{ReviewerPattern: "genomics-reviewer"},
		{ReviewerPattern: "proteomics-reviewer"},
	})
	findings := []ExtractedFinding{
		makeFinding("genomics-reviewer", "g1", "critical", "cat", "msg"),
		makeFinding("proteomics-reviewer", "p1", "warning", "cat", "msg"),
	}

	result := DetectInteractions(InteractionRulesConfig{Rules: []InteractionRule{rule}}, findings)
	require.Len(t, result.DetectedInteractions, 1)
	assert.Equal(t, "critical", result.DetectedInteractions[0].Severity,
		"gating: upstream critical gates downstream warning → critical")
}

func TestAlgebra_Negating(t *testing.T) {
	// warning mitigated by info → negating → min = info.
	rule := makeRule("r1", "negating", "requires_all", []FindingMatcher{
		{ReviewerPattern: "proteomics-reviewer"},
		{ReviewerPattern: "bioinformatician-reviewer"},
	})
	findings := []ExtractedFinding{
		makeFinding("proteomics-reviewer", "p1", "warning", "cat", "MBR risk"),
		makeFinding("bioinformatician-reviewer", "b1", "info", "cat", "MBR excluded"),
	}

	result := DetectInteractions(InteractionRulesConfig{Rules: []InteractionRule{rule}}, findings)
	require.Len(t, result.DetectedInteractions, 1)
	assert.Equal(t, "info", result.DetectedInteractions[0].Severity,
		"negating: warning mitigated by info → info")
}

func TestAlgebra_SeverityOverride(t *testing.T) {
	// info + info inputs with severity_override=critical → override wins.
	rule := InteractionRule{
		ID:               "r1",
		Name:             "r1",
		Algebra:          "additive",
		SeverityOverride: "critical",
		Condition: RuleCondition{
			Type: "requires_all",
			Matchers: []FindingMatcher{
				{ReviewerPattern: "genomics-reviewer"},
				{ReviewerPattern: "proteomics-reviewer"},
			},
		},
		MessageTemplate: "msg",
		Layer:           1,
	}
	findings := []ExtractedFinding{
		makeFinding("genomics-reviewer", "g1", "info", "cat", "msg"),
		makeFinding("proteomics-reviewer", "p1", "info", "cat", "msg"),
	}

	result := DetectInteractions(InteractionRulesConfig{Rules: []InteractionRule{rule}}, findings)
	require.Len(t, result.DetectedInteractions, 1)
	assert.Equal(t, "critical", result.DetectedInteractions[0].Severity,
		"severity_override=critical wins over additive(info, info)=warning")
}

// --- Multi-interaction tests ---

func TestFindingInMultipleInteractions(t *testing.T) {
	// The same finding (proteomics FDR) participates in two different rules.
	// It should appear in both DetectedInteractions and NOT be in UnmatchedFindings.
	fdrRule1 := makeRule("fdr-chain", "multiplicative", "requires_all", []FindingMatcher{
		{ReviewerPattern: "proteogenomics-reviewer", SharpEdgePattern: "proteogenomics-db-*"},
		{ReviewerPattern: "proteomics-reviewer", SharpEdgePattern: "proteomics-fdr-*"},
	})
	fdrRule2 := makeRule("multistage-fdr", "multiplicative", "requires_all", []FindingMatcher{
		{ReviewerPattern: "proteomics-reviewer", SharpEdgePattern: "proteomics-fdr-multistage-*"},
		{ReviewerPattern: "proteogenomics-reviewer", SharpEdgePattern: "proteogenomics-db-*"},
	})

	fdrFinding := makeFinding("proteomics-reviewer", "proteomics-fdr-multistage-dependent", "warning", "fdr", "FDR issue")
	dbFinding := makeFinding("proteogenomics-reviewer", "proteogenomics-db-inflation", "warning", "database", "DB inflated")
	otherFinding := makeFinding("genomics-reviewer", "genomics-snp-filter", "info", "variant", "Other")

	findings := []ExtractedFinding{fdrFinding, dbFinding, otherFinding}
	rules := InteractionRulesConfig{Rules: []InteractionRule{fdrRule1, fdrRule2}}

	result := DetectInteractions(rules, findings)

	// Both rules should fire (fdr-chain requires proteomics-fdr-*, multistage-fdr requires proteomics-fdr-multistage-*)
	require.Len(t, result.DetectedInteractions, 2, "both rules should fire")

	// The fdrFinding participated in at least one interaction, so it should NOT be unmatched.
	for _, uf := range result.UnmatchedFindings {
		assert.NotEqual(t, fdrFinding.SharpEdgeID, uf.SharpEdgeID,
			"fdrFinding participated in interactions and must not be unmatched")
	}

	// The otherFinding did not participate — it should be unmatched.
	var otherUnmatched bool
	for _, uf := range result.UnmatchedFindings {
		if uf.SharpEdgeID == otherFinding.SharpEdgeID {
			otherUnmatched = true
		}
	}
	assert.True(t, otherUnmatched, "otherFinding should be in UnmatchedFindings")
}

// --- LoadRules tests ---

func TestLoadRules_ValidFile(t *testing.T) {
	dir := t.TempDir()
	rulesPath := filepath.Join(dir, "interaction-rules.json")

	content := `{
		"version": "1.0.0",
		"rules": [
			{
				"id": "test-rule",
				"name": "Test Rule",
				"algebra": "additive",
				"description": "A test rule",
				"condition": {
					"type": "requires_all",
					"matchers": [
						{"reviewer_pattern": "proteomics-reviewer", "sharp_edge_pattern": "proteomics-fdr-*"}
					]
				},
				"message_template": "test",
				"layer": 1,
				"tags": ["test"]
			}
		]
	}`
	err := os.WriteFile(rulesPath, []byte(content), 0644)
	require.NoError(t, err)

	cfg, err := LoadRules(rulesPath)
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", cfg.Version)
	require.Len(t, cfg.Rules, 1)
	assert.Equal(t, "test-rule", cfg.Rules[0].ID)
	assert.Equal(t, "additive", cfg.Rules[0].Algebra)
	require.Len(t, cfg.Rules[0].Condition.Matchers, 1)
	assert.Equal(t, "proteomics-fdr-*", cfg.Rules[0].Condition.Matchers[0].SharpEdgePattern)
}

func TestLoadRules_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	rulesPath := filepath.Join(dir, "interaction-rules.json")
	err := os.WriteFile(rulesPath, []byte("{bad json"), 0644)
	require.NoError(t, err)

	_, err = LoadRules(rulesPath)
	assert.Error(t, err, "malformed JSON should return an error")
}

// --- ExtractFindings tests ---

func TestExtractFindings_ValidJSON(t *testing.T) {
	raw := makeReviewerJSON(t, []reviewerFinding{
		{ID: "PROT-1", SharpEdgeID: "proteomics-fdr-global-only", Severity: "HIGH",
			Category: "fdr-control", File: "pipeline.nf", Line: 42,
			Title: "Global FDR", Message: "Only PSM-level FDR applied"},
	})

	findings := ExtractFindings("proteomics-reviewer", raw)
	require.Len(t, findings, 1)

	f := findings[0]
	assert.Equal(t, "proteomics-reviewer", f.ReviewerID)
	assert.Equal(t, "PROT-1", f.FindingID)
	assert.Equal(t, "proteomics-fdr-global-only", f.SharpEdgeID)
	assert.Equal(t, "warning", f.Severity) // HIGH → warning
	assert.Equal(t, "fdr-control", f.Category)
	assert.Equal(t, "pipeline.nf", f.File)
	assert.Equal(t, 42, f.Line)
	assert.Equal(t, "Global FDR", f.Title)
	assert.Equal(t, "Only PSM-level FDR applied", f.Message)
}

func TestExtractFindings_EmptyFindings(t *testing.T) {
	raw := makeReviewerJSON(t, []reviewerFinding{})
	findings := ExtractFindings("proteomics-reviewer", raw)
	assert.Empty(t, findings)
}

func TestExtractFindings_ReviewerIDPropagated(t *testing.T) {
	raw := makeReviewerJSON(t, []reviewerFinding{
		{ID: "F-1", Severity: "INFO", Message: "msg"},
		{ID: "F-2", Severity: "CRITICAL", Message: "msg2"},
	})

	findings := ExtractFindings("mass-spec-reviewer", raw)
	require.Len(t, findings, 2)
	for _, f := range findings {
		assert.Equal(t, "mass-spec-reviewer", f.ReviewerID)
	}
}

// --- DetectionResult summary fields tests ---

func TestDetectionResult_SummaryFields(t *testing.T) {
	rule1 := makeRule("r1", "additive", "requires_all", []FindingMatcher{
		{ReviewerPattern: "genomics-reviewer"},
		{ReviewerPattern: "proteomics-reviewer"},
	})
	rule2 := makeRule("r2", "gating", "requires_any", []FindingMatcher{
		{ReviewerPattern: "mass-spec-reviewer"},
	})

	findings := []ExtractedFinding{
		makeFinding("genomics-reviewer", "g1", "warning", "cat", "msg"),
		makeFinding("proteomics-reviewer", "p1", "warning", "cat", "msg"),
		makeFinding("bioinformatician-reviewer", "b1", "info", "cat", "msg"), // unmatched
	}

	result := DetectInteractions(InteractionRulesConfig{Rules: []InteractionRule{rule1, rule2}}, findings)

	assert.Equal(t, 2, result.RulesEvaluated)
	assert.Equal(t, 3, result.FindingsTotal)
	assert.Len(t, result.DetectedInteractions, 1, "only rule1 should fire (mass-spec absent)")
	// bioinformatician finding is unmatched
	assert.Len(t, result.UnmatchedFindings, 1)
	assert.Equal(t, "bioinformatician-reviewer", result.UnmatchedFindings[0].ReviewerID)
}

// --- renderMessage tests ---

func TestRenderMessage_Substitutions(t *testing.T) {
	matched := []ExtractedFinding{
		{ReviewerID: "proteogenomics-reviewer", SharpEdgeID: "proteogenomics-db-inflation"},
		{ReviewerID: "proteomics-reviewer", SharpEdgeID: "proteomics-fdr-global-only"},
	}

	tmpl := "DB {finding_a.sharp_edge_id} from {finding_a.reviewer} combined with {finding_b.sharp_edge_id} from {finding_b.reviewer}"
	result := renderMessage(tmpl, matched)
	assert.Contains(t, result, "proteogenomics-db-inflation")
	assert.Contains(t, result, "proteogenomics-reviewer")
	assert.Contains(t, result, "proteomics-fdr-global-only")
	assert.Contains(t, result, "proteomics-reviewer")
}

func TestRenderMessage_MatchedFindings(t *testing.T) {
	matched := []ExtractedFinding{
		{ReviewerID: "genomics-reviewer", SharpEdgeID: "genomics-ref-build38", Severity: "warning"},
		{ReviewerID: "proteomics-reviewer", SharpEdgeID: "proteomics-fdr-x", Severity: "critical"},
	}

	result := renderMessage("Findings: {matched_findings}", matched)
	assert.Contains(t, result, "genomics-reviewer")
	assert.Contains(t, result, "proteomics-reviewer")
}

func TestRenderMessage_EmptyMatched(t *testing.T) {
	// Empty matched slice should not panic.
	result := renderMessage("Template with {finding_a.sharp_edge_id}", []ExtractedFinding{})
	// Template placeholder remains since no finding to substitute.
	assert.Contains(t, result, "{finding_a.sharp_edge_id}")
}
