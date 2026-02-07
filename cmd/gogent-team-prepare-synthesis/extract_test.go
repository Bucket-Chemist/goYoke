package main

import (
	"io"
	"log"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractEinstein(t *testing.T) {
	tests := []struct {
		name                     string
		fixtureFile              string
		wantExecutiveSummary     string
		wantRootCausesLen        int
		wantFrameworksLen        int
		wantFirstPrinciplesLen   int
		wantNovelApproachLen     int
		wantTradeoffsLen         int
		wantAssumptionsSurfLen   int
		wantOpenQuestionsLen     int
		wantHandoffNotes         string
		wantFallback             bool
	}{
		{
			name:                   "valid_complete",
			fixtureFile:            "valid_einstein.json",
			wantExecutiveSummary:   "Test executive summary from Einstein",
			wantRootCausesLen:      1,
			wantFrameworksLen:      1,
			wantFirstPrinciplesLen: 2, // 1 assumption challenged + 1 constraint
			wantNovelApproachLen:   1,
			wantTradeoffsLen:       1,
			wantAssumptionsSurfLen: 1,
			wantOpenQuestionsLen:   1,
			wantHandoffNotes:       "Test handoff notes",
			wantFallback:           false,
		},
		{
			name:                   "file_missing",
			fixtureFile:            "nonexistent.json",
			wantExecutiveSummary:   "(unavailable: Einstein analysis file unavailable)",
			wantRootCausesLen:      1,
			wantFallback:           true,
		},
		{
			name:                   "malformed_json",
			fixtureFile:            "malformed.json",
			wantExecutiveSummary:   "(unavailable: Einstein analysis could not parse JSON)",
			wantRootCausesLen:      1,
			wantFallback:           true,
		},
		{
			name:                   "empty_json",
			fixtureFile:            "empty.json",
			wantExecutiveSummary:   "(unavailable: Einstein analysis empty content)",
			wantRootCausesLen:      1,
			wantFallback:           true,
		},
		{
			name:                   "status_failed",
			fixtureFile:            "failed_einstein.json",
			wantExecutiveSummary:   "(unavailable: Einstein analysis agent reported failure)",
			wantRootCausesLen:      1,
			wantFallback:           true,
		},
		{
			name:                   "unknown_fields",
			fixtureFile:            "extra_fields_einstein.json",
			wantExecutiveSummary:   "Test executive summary from Einstein",
			wantRootCausesLen:      1,
			wantFrameworksLen:      1,
			wantFirstPrinciplesLen: 2,
			wantNovelApproachLen:   1,
			wantTradeoffsLen:       1,
			wantAssumptionsSurfLen: 1,
			wantOpenQuestionsLen:   1,
			wantHandoffNotes:       "Test handoff notes",
			wantFallback:           false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fixturePath := filepath.Join("testdata", tc.fixtureFile)
			result := extractEinstein(fixturePath)

			assert.Equal(t, tc.wantExecutiveSummary, result.ExecutiveSummary)
			assert.Len(t, result.RootCauses, tc.wantRootCausesLen)

			if tc.wantFallback {
				assert.Contains(t, result.RootCauses[0], "(unavailable:")
			}

			if !tc.wantFallback {
				assert.Len(t, result.Frameworks, tc.wantFrameworksLen)
				assert.Len(t, result.FirstPrinciples, tc.wantFirstPrinciplesLen)
				assert.Len(t, result.NovelApproaches, tc.wantNovelApproachLen)
				assert.Len(t, result.TheoreticalTradeoffs, tc.wantTradeoffsLen)
				assert.Len(t, result.AssumptionsSurfaced, tc.wantAssumptionsSurfLen)
				assert.Len(t, result.OpenQuestions, tc.wantOpenQuestionsLen)
				assert.Equal(t, tc.wantHandoffNotes, result.HandoffNotes)
			}
		})
	}
}

func TestExtractStaffArch(t *testing.T) {
	tests := []struct {
		name                  string
		fixtureFile           string
		wantExecutiveVerdict  string
		wantCriticalIssuesLen int
		wantMajorIssuesLen    int
		wantMinorIssuesLen    int
		wantCommendationsLen  int
		wantFailureModesLen   int
		wantRecommendLen      int
		wantSignOffLen        int
		wantHandoffNotes      string
		wantFallback          bool
	}{
		{
			name:                  "valid_complete",
			fixtureFile:           "valid_staff_arch.json",
			wantExecutiveVerdict:  "**Verdict:** APPROVE_WITH_CONDITIONS",
			wantCriticalIssuesLen: 1,
			wantMajorIssuesLen:    1,
			wantMinorIssuesLen:    1,
			wantCommendationsLen:  2,
			wantFailureModesLen:   1,
			wantRecommendLen:      3, // HIGH + MEDIUM + LOW
			wantSignOffLen:        1,
			wantHandoffNotes:      "Staff handoff notes",
			wantFallback:          false,
		},
		{
			name:                  "file_missing",
			fixtureFile:           "nonexistent.json",
			wantExecutiveVerdict:  "(unavailable: Staff-Architect review file unavailable)",
			wantCriticalIssuesLen: 1,
			wantFallback:          true,
		},
		{
			name:                  "malformed_json",
			fixtureFile:           "malformed.json",
			wantExecutiveVerdict:  "(unavailable: Staff-Architect review could not parse JSON)",
			wantCriticalIssuesLen: 1,
			wantFallback:          true,
		},
		{
			name:                  "status_failed",
			fixtureFile:           "failed_staff_arch.json",
			wantExecutiveVerdict:  "(unavailable:",
			wantCriticalIssuesLen: 1,
			wantFallback:          true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fixturePath := filepath.Join("testdata", tc.fixtureFile)
			result := extractStaffArch(fixturePath)

			assert.Contains(t, result.ExecutiveVerdict, tc.wantExecutiveVerdict)
			assert.Len(t, result.CriticalIssues, tc.wantCriticalIssuesLen)

			if tc.wantFallback {
				assert.Contains(t, result.CriticalIssues[0], "(unavailable:")
			}

			if !tc.wantFallback {
				assert.Len(t, result.MajorIssues, tc.wantMajorIssuesLen)
				assert.Len(t, result.MinorIssues, tc.wantMinorIssuesLen)
				assert.Len(t, result.Commendations, tc.wantCommendationsLen)
				assert.Len(t, result.FailureModes, tc.wantFailureModesLen)
				assert.Len(t, result.Recommendations, tc.wantRecommendLen)
				assert.Len(t, result.SignOffConditions, tc.wantSignOffLen)
				assert.Equal(t, tc.wantHandoffNotes, result.HandoffNotes)
			}
		})
	}
}

func TestGenerateMarkdown(t *testing.T) {
	tests := []struct {
		name         string
		einstein     EinsteinSections
		staffArch    StaffArchSections
		wantContains []string
	}{
		{
			name: "both_valid",
			einstein: EinsteinSections{
				Status:               "complete",
				ExecutiveSummary:     "Test summary",
				RootCauses:           []string{"Cause 1"},
				Frameworks:           []string{"Framework 1"},
				FirstPrinciples:      []string{"FP 1"},
				NovelApproaches:      []string{"Approach 1"},
				TheoreticalTradeoffs: []string{"Tradeoff 1"},
				AssumptionsSurfaced:  []string{"Assumption 1"},
				OpenQuestions:        []string{"Question 1"},
				HandoffNotes:         "Einstein notes",
			},
			staffArch: StaffArchSections{
				Status:            "complete",
				ExecutiveVerdict:  "APPROVE",
				CriticalIssues:    []string{"Critical 1"},
				MajorIssues:       []string{"Major 1"},
				MinorIssues:       []string{"Minor 1"},
				Commendations:     []string{"Good work"},
				FailureModes:      []string{"FM 1"},
				Recommendations:   []string{"Rec 1"},
				SignOffConditions: []string{"Condition 1"},
				HandoffNotes:      "Staff notes",
			},
			wantContains: []string{
				"# Pre-Synthesis Input for Beethoven",
				"## Einstein: Theoretical Analysis",
				"Test summary",
				"Cause 1",
				"Framework 1",
				"FP 1",
				"Approach 1",
				"Tradeoff 1",
				"Assumption 1",
				"Question 1",
				"Einstein notes",
				"## Staff-Architect: Critical Review",
				"APPROVE",
				"Critical 1",
				"Major 1",
				"Minor 1",
				"Good work",
				"FM 1",
				"Rec 1",
				"Condition 1",
				"Staff notes",
			},
		},
		{
			name: "both_fallback",
			einstein: EinsteinSections{
				Status:               "unavailable",
				ExecutiveSummary:     "(unavailable: Einstein analysis file unavailable)",
				RootCauses:           []string{"(unavailable: file unavailable)"},
				Frameworks:           []string{"(unavailable: file unavailable)"},
				FirstPrinciples:      []string{"(unavailable: file unavailable)"},
				NovelApproaches:      []string{"(unavailable: file unavailable)"},
				TheoreticalTradeoffs: []string{"(unavailable: file unavailable)"},
				AssumptionsSurfaced:  []string{"(unavailable: file unavailable)"},
				OpenQuestions:        []string{"(unavailable: file unavailable)"},
				HandoffNotes:         "(unavailable: file unavailable)",
			},
			staffArch: StaffArchSections{
				Status:            "unavailable",
				ExecutiveVerdict:  "(unavailable: Staff-Architect review file unavailable)",
				CriticalIssues:    []string{"(unavailable: file unavailable)"},
				MajorIssues:       []string{"(unavailable: file unavailable)"},
				MinorIssues:       []string{"(unavailable: file unavailable)"},
				Commendations:     []string{"(unavailable: file unavailable)"},
				FailureModes:      []string{"(unavailable: file unavailable)"},
				Recommendations:   []string{"(unavailable: file unavailable)"},
				SignOffConditions: []string{"(unavailable: file unavailable)"},
				HandoffNotes:      "(unavailable: file unavailable)",
			},
			wantContains: []string{
				"WARNING: One or more analyses reported non-complete status",
				"(unavailable: Einstein analysis file unavailable)",
				"(unavailable: Staff-Architect review file unavailable)",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := generateMarkdown(tc.einstein, tc.staffArch)

			for _, want := range tc.wantContains {
				assert.Contains(t, result, want)
			}
		})
	}
}

func TestFallbackSections(t *testing.T) {
	t.Run("fallback_einstein", func(t *testing.T) {
		result := fallbackEinsteinSections("test reason")
		assert.Equal(t, "unavailable", result.Status)
		assert.Contains(t, result.ExecutiveSummary, "unavailable:")
		assert.Contains(t, result.ExecutiveSummary, "test reason")
		assert.Len(t, result.RootCauses, 1)
		assert.Contains(t, result.RootCauses[0], "unavailable:")
	})

	t.Run("fallback_staff_arch", func(t *testing.T) {
		result := fallbackStaffArchSections("test reason")
		assert.Equal(t, "unavailable", result.Status)
		assert.Contains(t, result.ExecutiveVerdict, "unavailable:")
		assert.Contains(t, result.ExecutiveVerdict, "test reason")
		assert.Len(t, result.CriticalIssues, 1)
		assert.Contains(t, result.CriticalIssues[0], "unavailable:")
	})
}

// FIX #12: Correct test init() to suppress logs
func init() {
	log.SetOutput(io.Discard)
}
