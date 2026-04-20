package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loadFixtureFindings reads all stdout_*.json files from a mock-wave0 fixture directory
// and returns all extracted findings. Reviewer ID is derived from filename:
// stdout_genomics-reviewer.json → genomics-reviewer.
//
// This matches the real integration flow: all files are read, and ExtractFindings
// returns empty findings for reviewers with no findings (e.g., failed reviewers).
func loadFixtureFindings(t *testing.T, fixture string) []ExtractedFinding {
	t.Helper()
	fixturePath := filepath.Join("testdata", "mock-wave0", fixture)
	entries, err := os.ReadDir(fixturePath)
	require.NoError(t, err)

	var allFindings []ExtractedFinding
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "stdout_") || !strings.HasSuffix(name, ".json") {
			continue
		}
		// stdout_genomics-reviewer.json → genomics-reviewer
		reviewerID := strings.TrimSuffix(strings.TrimPrefix(name, "stdout_"), ".json")

		data, err := os.ReadFile(filepath.Join(fixturePath, name))
		require.NoError(t, err)

		allFindings = append(allFindings, ExtractFindings(reviewerID, data)...)
	}
	return allFindings
}

// loadRulesForTest loads interaction-rules.json from testdata/.
func loadRulesForTest(t *testing.T) InteractionRulesConfig {
	t.Helper()
	rules, err := LoadRules(filepath.Join("testdata", "interaction-rules.json"))
	require.NoError(t, err)
	require.NotEmpty(t, rules.Rules, "testdata/interaction-rules.json must contain at least one rule")
	return rules
}

// ruleIDSet extracts rule IDs from detected interactions into a set for easy assertion.
func ruleIDSet(result DetectionResult) map[string]bool {
	ids := make(map[string]bool)
	for _, d := range result.DetectedInteractions {
		ids[d.RuleID] = true
	}
	return ids
}

// TestIntegration_VCFVEPPipeline verifies that the vcf-vep fixture produces the
// expected three cross-domain interactions:
//   - version-coherence-break (genomics ref build mismatch + VEP/PyEnsembl version mismatch)
//   - fdr-chain-inflation (proteogenomics DB inflation + proteomics PSM-only FDR)
//   - header-format-protein-inference (FASTA header format + search engine header parsing)
func TestIntegration_VCFVEPPipeline(t *testing.T) {
	rules := loadRulesForTest(t)
	findings := loadFixtureFindings(t, "fixture-vcf-vep")

	result := DetectInteractions(rules, findings)

	ids := ruleIDSet(result)
	assert.True(t, ids["version-coherence-break"],
		"version-coherence-break should fire: genomics-ref-wrong-build + proteogenomics-version-vep-pyensembl")
	assert.True(t, ids["fdr-chain-inflation"],
		"fdr-chain-inflation should fire: proteogenomics-db-search-inflation + proteomics-fdr-global-only")
	assert.True(t, ids["header-format-protein-inference"],
		"header-format-protein-inference should fire: proteogenomics-fasta-header-incomplete + proteomics-search-header-incompatible")
	assert.Equal(t, 3, len(result.DetectedInteractions),
		"exactly 3 interactions expected for vcf-vep fixture")
}

// TestIntegration_CleanPipeline verifies that a well-configured DIA pipeline produces
// zero interactions. All findings should be in the unmatched pool.
func TestIntegration_CleanPipeline(t *testing.T) {
	rules := loadRulesForTest(t)
	findings := loadFixtureFindings(t, "fixture-dia-clean")

	result := DetectInteractions(rules, findings)

	assert.Empty(t, result.DetectedInteractions,
		"no interactions expected for a well-configured DIA pipeline")
	assert.Equal(t, result.FindingsTotal, len(result.UnmatchedFindings),
		"all findings should be unmatched when no interactions detected")
}

// TestIntegration_CriticalFailure verifies that a pipeline with cascading failures
// produces at least 4 interactions. Specifically checks:
//   - fdr-chain-inflation (proteogenomics DB inflation + FDR control failure)
//   - spectral-quality-gates-identification (mass-spec critical quality + proteomics any)
//   - variant-normalization-cascade (genomics-vc-no-normalization + proteogenomics dedup absent)
//   - mbr-no-spectral-in-diffex (MBR without FDR + uncorrected differential expression)
func TestIntegration_CriticalFailure(t *testing.T) {
	rules := loadRulesForTest(t)
	findings := loadFixtureFindings(t, "fixture-critical")

	result := DetectInteractions(rules, findings)

	ids := ruleIDSet(result)
	assert.GreaterOrEqual(t, len(result.DetectedInteractions), 4,
		"at least 4 interactions expected for critical fixture")
	assert.True(t, ids["fdr-chain-inflation"],
		"fdr-chain-inflation should fire")
	assert.True(t, ids["spectral-quality-gates-identification"],
		"spectral-quality-gates-identification should fire: mass-spec-spectral-quality (critical) + proteomics any")
	assert.True(t, ids["variant-normalization-cascade"],
		"variant-normalization-cascade should fire: genomics-vc-no-normalization + proteogenomics-fasta-dedup-missing")
	assert.True(t, ids["mbr-no-spectral-in-diffex"],
		"mbr-no-spectral-in-diffex should fire: proteomics-mbr-no-fdr + proteomics-stats-no-correction")
}

// TestIntegration_PartialFailure verifies graceful handling when one reviewer has failed.
// fixture-partial contains: genomics-reviewer (3 findings) + proteomics-reviewer (failed, 0 findings).
// The engine should not panic, and should detect no cross-domain interactions since only
// genomics findings are available.
func TestIntegration_PartialFailure(t *testing.T) {
	rules := loadRulesForTest(t)
	// proteomics-reviewer in fixture-partial has status "failed" and findings:[].
	// ExtractFindings returns empty slice for it — this is the graceful degradation path.
	findings := loadFixtureFindings(t, "fixture-partial")

	result := DetectInteractions(rules, findings)

	// Must not panic; result fields must be initialised (not nil).
	require.NotNil(t, result.DetectedInteractions)
	require.NotNil(t, result.UnmatchedFindings)

	// No cross-domain interactions possible with only genomics findings present.
	assert.Empty(t, result.DetectedInteractions,
		"no cross-domain interactions expected when only genomics reviewer succeeded")

	// Verify only genomics findings contributed (proteomics reviewer failed → 0 findings).
	for _, f := range findings {
		assert.Equal(t, "genomics-reviewer", f.ReviewerID,
			"only genomics-reviewer findings should be present; proteomics reviewer failed")
	}
}

// TestIntegration_NoRulesFile verifies that a missing rules file causes graceful degradation:
// LoadRules returns an empty config with nil error, and DetectInteractions produces no interactions
// with all findings unmatched.
func TestIntegration_NoRulesFile(t *testing.T) {
	rules, err := LoadRules(filepath.Join("testdata", "nonexistent-rules.json"))
	require.NoError(t, err, "missing rules file must return nil error (graceful degradation)")
	assert.Empty(t, rules.Rules, "missing rules file returns empty config")

	// With empty rules, zero interactions detected, all findings unmatched.
	findings := loadFixtureFindings(t, "fixture-vcf-vep")
	result := DetectInteractions(rules, findings)

	assert.Empty(t, result.DetectedInteractions,
		"no interactions when rules file is missing")
	assert.Equal(t, result.FindingsTotal, len(result.UnmatchedFindings),
		"all findings should be unmatched with empty rules")
	assert.Equal(t, 0, result.RulesEvaluated,
		"zero rules evaluated when rules file is missing")
}
