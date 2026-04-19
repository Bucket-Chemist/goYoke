// Golden file tests for interaction detection output.
//
// To regenerate all golden files after an intentional format or rule change:
//
//	go test ./cmd/goyoke-team-prepare-synthesis/... -run TestGolden -update
//
// Commit the updated golden files alongside the code change so reviewers can
// see exactly what output changed.
package main

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "regenerate golden files instead of comparing them")

// goldenFixtures maps a short name to the fixture directory under testdata/mock-wave0/.
var goldenFixtures = []struct {
	name    string
	fixture string
}{
	{"vcf-vep", "fixture-vcf-vep"},
	{"dia-clean", "fixture-dia-clean"},
	{"critical", "fixture-critical"},
	{"partial", "fixture-partial"},
}

// runDetectionAndGetMarkdown drives the full detection pipeline for a fixture
// and returns the pre-synthesis.md content string.
func runDetectionAndGetMarkdown(t *testing.T, fixture string) string {
	t.Helper()
	rules := loadRulesForTest(t)
	findings := loadFixtureFindings(t, fixture)
	result := DetectInteractions(rules, findings)
	nReviewers := countUniqueReviewers(findings)
	return generateBioinformaticsPreSynthesis(result, rules, nReviewers)
}

// runDetectionAndGetJSON drives the full detection pipeline for a fixture and
// returns the unmarshalled detectedInteractionsJSON struct. GeneratedAt is NOT
// zeroed here — callers that need stable comparison must zero it themselves.
func runDetectionAndGetJSON(t *testing.T, fixture string) detectedInteractionsJSON {
	t.Helper()
	rules := loadRulesForTest(t)
	findings := loadFixtureFindings(t, fixture)
	result := DetectInteractions(rules, findings)

	tmpDir := t.TempDir()
	path, err := writeDetectedInteractionsJSON(tmpDir, result, rules)
	require.NoError(t, err, "writeDetectedInteractionsJSON for fixture %s", fixture)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var out detectedInteractionsJSON
	require.NoError(t, json.Unmarshal(data, &out))
	return out
}

// readGolden reads a golden file and returns its content as a string.
func readGolden(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err, "golden file missing — run with -update to generate: %s", path)
	return string(data)
}

// readGoldenJSON reads a golden file and unmarshals it as detectedInteractionsJSON.
func readGoldenJSON(t *testing.T, path string) detectedInteractionsJSON {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err, "golden file missing — run with -update to generate: %s", path)
	var out detectedInteractionsJSON
	require.NoError(t, json.Unmarshal(data, &out))
	return out
}

// TestGolden_Markdown verifies that pre-synthesis.md output matches the golden files.
func TestGolden_Markdown(t *testing.T) {
	for _, tc := range goldenFixtures {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := runDetectionAndGetMarkdown(t, tc.fixture)
			goldenPath := filepath.Join("testdata", "golden", tc.name+"-pre-synthesis.md")

			if *update {
				require.NoError(t, os.MkdirAll(filepath.Dir(goldenPath), 0755))
				require.NoError(t, os.WriteFile(goldenPath, []byte(got), 0644))
				t.Logf("updated golden file: %s", goldenPath)
				return
			}

			want := readGolden(t, goldenPath)
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("pre-synthesis.md mismatch for fixture %s (-want +got):\n%s", tc.name, diff)
			}
		})
	}
}

// TestGolden_JSON verifies that detected-interactions.json output matches golden files.
// generated_at is zeroed before comparison because it contains a wall-clock timestamp.
func TestGolden_JSON(t *testing.T) {
	for _, tc := range goldenFixtures {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := runDetectionAndGetJSON(t, tc.fixture)
			got.GeneratedAt = "" // normalize timestamp
			goldenPath := filepath.Join("testdata", "golden", tc.name+"-detected-interactions.json")

			if *update {
				require.NoError(t, os.MkdirAll(filepath.Dir(goldenPath), 0755))
				data, err := json.MarshalIndent(got, "", "  ")
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(goldenPath, data, 0644))
				t.Logf("updated golden file: %s", goldenPath)
				return
			}

			want := readGoldenJSON(t, goldenPath)
			want.GeneratedAt = "" // normalize timestamp in golden too

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("detected-interactions.json mismatch for fixture %s (-want +got):\n%s", tc.name, diff)
			}
		})
	}
}

// TestGolden_RulesDrift verifies that testdata/interaction-rules.json matches the
// deployed rules at ~/.claude/schemas/teams/interaction-rules.json.
// Skipped automatically in CI environments where ~/.claude is absent.
func TestGolden_RulesDrift(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("cannot resolve home dir (%v), skipping drift test", err)
	}
	deployedPath := filepath.Join(home, ".claude", "schemas", "teams", "interaction-rules.json")
	if _, err := os.Stat(deployedPath); os.IsNotExist(err) {
		t.Skip("deployed interaction-rules.json not found (~/.claude absent), skipping drift test")
	}

	testdataRules, err := LoadRules(filepath.Join("testdata", "interaction-rules.json"))
	require.NoError(t, err)

	deployedRules, err := LoadRules(deployedPath)
	require.NoError(t, err)

	if diff := cmp.Diff(testdataRules, deployedRules); diff != "" {
		t.Errorf("testdata/interaction-rules.json drifted from deployed rules (-testdata +deployed):\n%s\n\nRun: cp ~/.claude/schemas/teams/interaction-rules.json testdata/interaction-rules.json", diff)
	}
}
