package main

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildBioConfig constructs a TeamConfig with a wave of reviewers + a pasteur wave.
func buildBioConfig(_ string, reviewers []MemberInfo, pasteurStdinFile string) *TeamConfig {
	return &TeamConfig{
		WorkflowType: "review-bioinformatics",
		Waves: []WaveInfo{
			{Members: reviewers},
			{Members: []MemberInfo{
				{Agent: "pasteur", StdinFile: pasteurStdinFile},
			}},
		},
	}
}

// writePasteurStdin writes a minimal pasteur stdin JSON to the given path.
func writePasteurStdin(t *testing.T, path string, extra map[string]any) {
	t.Helper()
	data := map[string]any{
		"problem_brief": "test brief",
	}
	maps.Copy(data, extra)
	b, err := json.Marshal(data)
	require.NoError(t, err)
	err = os.WriteFile(path, b, 0644)
	require.NoError(t, err)
}

func readPasteurStdin(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var result map[string]any
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)
	return result
}

func TestPrepareBioinformaticsReview_HappyPath(t *testing.T) {
	dir := t.TempDir()
	stdinPath := filepath.Join(dir, "stdin_pasteur.json")
	writePasteurStdin(t, stdinPath, nil)

	reviewers := []MemberInfo{
		{Agent: "genomics-reviewer", StdoutFile: "stdout_genomics.json", Status: "completed", CostUSD: 0.10},
		{Agent: "proteomics-reviewer", StdoutFile: "stdout_proteomics.json", Status: "completed", CostUSD: 0.12},
		{Agent: "bioinformatician-reviewer", StdoutFile: "stdout_bioinformatician.json", Status: "completed", CostUSD: 0.08},
	}
	cfg := buildBioConfig(dir, reviewers, "stdin_pasteur.json")

	err := prepareBioinformaticsReview(dir, cfg, 0)
	require.NoError(t, err)

	result := readPasteurStdin(t, stdinPath)
	outputs, ok := result["wave_0_outputs"].([]any)
	require.True(t, ok, "wave_0_outputs should be a slice")
	assert.Len(t, outputs, 3)

	// Verify each entry
	for _, entry := range outputs {
		m, ok := entry.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "completed", m["status"])
		assert.NotEmpty(t, m["reviewer_id"])
		assert.NotEmpty(t, m["stdout_file_path"])
	}

	// Verify runtime-injected path fields (SYNTH-002 contract)
	assert.NotEmpty(t, result["wave0_findings_path"], "wave0_findings_path should be injected into synthesizer stdin")
	assert.NotEmpty(t, result["detected_interactions_path"], "detected_interactions_path should be injected into synthesizer stdin")
}

func TestPrepareBioinformaticsReview_MixedResults(t *testing.T) {
	dir := t.TempDir()
	stdinPath := filepath.Join(dir, "stdin_pasteur.json")
	writePasteurStdin(t, stdinPath, nil)

	reviewers := []MemberInfo{
		{Agent: "genomics-reviewer", StdoutFile: "stdout_genomics.json", Status: "completed", CostUSD: 0.10},
		{Agent: "proteomics-reviewer", StdoutFile: "stdout_proteomics.json", Status: "failed", CostUSD: 0.0},
		{Agent: "bioinformatician-reviewer", StdoutFile: "stdout_bioinformatician.json", Status: "completed", CostUSD: 0.08},
	}
	cfg := buildBioConfig(dir, reviewers, "stdin_pasteur.json")

	err := prepareBioinformaticsReview(dir, cfg, 0)
	require.NoError(t, err)

	result := readPasteurStdin(t, stdinPath)
	outputs, ok := result["wave_0_outputs"].([]any)
	require.True(t, ok)
	require.Len(t, outputs, 3)

	statusByID := make(map[string]string)
	for _, entry := range outputs {
		m := entry.(map[string]any)
		statusByID[m["reviewer_id"].(string)] = m["status"].(string)
	}
	assert.Equal(t, "completed", statusByID["genomics-reviewer"])
	assert.Equal(t, "failed", statusByID["proteomics-reviewer"])
	assert.Equal(t, "completed", statusByID["bioinformatician-reviewer"])
}

func TestPrepareBioinformaticsReview_PasteurNotFound(t *testing.T) {
	dir := t.TempDir()

	// Next wave has no pasteur member
	cfg := &TeamConfig{
		Waves: []WaveInfo{
			{Members: []MemberInfo{
				{Agent: "genomics-reviewer", Status: "completed"},
			}},
			{Members: []MemberInfo{
				{Agent: "some-other-agent", StdinFile: "stdin_other.json"},
			}},
		},
	}

	// Should gracefully exit with no error
	err := prepareBioinformaticsReview(dir, cfg, 0)
	require.NoError(t, err)
}

func TestPrepareBioinformaticsReview_MissingStdinFile(t *testing.T) {
	dir := t.TempDir()
	// stdin_pasteur.json does NOT exist

	reviewers := []MemberInfo{
		{Agent: "genomics-reviewer", StdoutFile: "stdout_genomics.json", Status: "completed"},
	}
	cfg := buildBioConfig(dir, reviewers, "stdin_pasteur.json")

	// Should gracefully exit with no error
	err := prepareBioinformaticsReview(dir, cfg, 0)
	require.NoError(t, err)
}

func TestPrepareBioinformaticsReview_MalformedStdinJSON(t *testing.T) {
	dir := t.TempDir()
	stdinPath := filepath.Join(dir, "stdin_pasteur.json")
	err := os.WriteFile(stdinPath, []byte("{invalid json"), 0644)
	require.NoError(t, err)

	reviewers := []MemberInfo{
		{Agent: "genomics-reviewer", StdoutFile: "stdout_genomics.json", Status: "completed"},
	}
	cfg := buildBioConfig(dir, reviewers, "stdin_pasteur.json")

	// Should gracefully exit with no error
	err = prepareBioinformaticsReview(dir, cfg, 0)
	require.NoError(t, err)
}

func TestPrepareBioinformaticsReview_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	stdinPath := filepath.Join(dir, "stdin_pasteur.json")
	writePasteurStdin(t, stdinPath, nil)

	reviewers := []MemberInfo{
		{Agent: "genomics-reviewer", StdoutFile: "stdout_genomics.json", Status: "completed", CostUSD: 0.05},
	}
	cfg := buildBioConfig(dir, reviewers, "stdin_pasteur.json")

	err := prepareBioinformaticsReview(dir, cfg, 0)
	require.NoError(t, err)

	// Verify no .tmp file remains
	tmpPath := stdinPath + ".tmp"
	_, err = os.Stat(tmpPath)
	assert.True(t, os.IsNotExist(err), ".tmp file should not exist after successful write")

	// Verify the main file was updated
	_, err = os.Stat(stdinPath)
	assert.NoError(t, err, "stdin_pasteur.json should exist")
}

func TestPrepareBioinformaticsReview_StdoutFilePathIsAbsolute(t *testing.T) {
	dir := t.TempDir()
	stdinPath := filepath.Join(dir, "stdin_pasteur.json")
	writePasteurStdin(t, stdinPath, nil)

	reviewers := []MemberInfo{
		{Agent: "genomics-reviewer", StdoutFile: "stdout_genomics.json", Status: "completed", CostUSD: 0.05},
	}
	cfg := buildBioConfig(dir, reviewers, "stdin_pasteur.json")

	err := prepareBioinformaticsReview(dir, cfg, 0)
	require.NoError(t, err)

	result := readPasteurStdin(t, stdinPath)
	outputs := result["wave_0_outputs"].([]any)
	require.Len(t, outputs, 1)

	m := outputs[0].(map[string]any)
	stdoutPath := m["stdout_file_path"].(string)
	assert.True(t, filepath.IsAbs(stdoutPath), "stdout_file_path should be absolute, got: %s", stdoutPath)
	assert.Equal(t, filepath.Join(dir, "stdout_genomics.json"), stdoutPath)
}

func TestPrepareBioinformaticsReview_StatusMapping(t *testing.T) {
	tests := []struct {
		memberStatus string
		wantStatus   string
	}{
		{"completed", "completed"},
		{"failed", "failed"},
		{"running", "timeout"},
		{"skipped", "timeout"},
		{"pending", "timeout"},
		{"", "timeout"},
	}

	for _, tc := range tests {
		t.Run(tc.memberStatus, func(t *testing.T) {
			result := memberStatusToOutputStatus(tc.memberStatus)
			assert.Equal(t, tc.wantStatus, result)
		})
	}
}
