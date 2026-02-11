package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMainIntegration(t *testing.T) {
	// Create temporary team directory
	tmpDir := t.TempDir()

	// Copy fixtures to temp directory
	einsteinSrc := filepath.Join("testdata", "valid_einstein.json")
	staffArchSrc := filepath.Join("testdata", "valid_staff_arch.json")

	einsteinData, err := os.ReadFile(einsteinSrc)
	require.NoError(t, err)
	staffArchData, err := os.ReadFile(staffArchSrc)
	require.NoError(t, err)

	einsteinDst := filepath.Join(tmpDir, einsteinStdoutFile)
	staffArchDst := filepath.Join(tmpDir, staffArchStdoutFile)

	err = os.WriteFile(einsteinDst, einsteinData, 0644)
	require.NoError(t, err)
	err = os.WriteFile(staffArchDst, staffArchData, 0644)
	require.NoError(t, err)

	// Run extraction
	einstein := extractEinstein(einsteinDst)
	staffArch := extractStaffArch(staffArchDst)

	// Generate markdown
	markdown := generateMarkdown(einstein, staffArch)

	// Write output
	outputPath := filepath.Join(tmpDir, outputFile)
	err = os.WriteFile(outputPath, []byte(markdown), 0644)
	require.NoError(t, err)

	// Verify output file exists and contains expected content
	outputData, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	output := string(outputData)
	assert.Contains(t, output, "# Pre-Synthesis Input for Beethoven")
	assert.Contains(t, output, "## Einstein: Theoretical Analysis")
	assert.Contains(t, output, "## Staff-Architect: Critical Review")
	assert.Contains(t, output, "Test executive summary from Einstein")
	assert.Contains(t, output, "APPROVE_WITH_CONDITIONS")
}

func TestMainIntegrationMissingFiles(t *testing.T) {
	// Create temporary team directory with no files
	tmpDir := t.TempDir()

	einsteinPath := filepath.Join(tmpDir, einsteinStdoutFile)
	staffArchPath := filepath.Join(tmpDir, staffArchStdoutFile)

	// Run extraction on missing files
	einstein := extractEinstein(einsteinPath)
	staffArch := extractStaffArch(staffArchPath)

	// Should get fallback sections (renamed from "fallback:" to "unavailable:" per fix #8)
	assert.Contains(t, einstein.ExecutiveSummary, "unavailable:")
	assert.Contains(t, staffArch.ExecutiveVerdict, "unavailable:")

	// Generate markdown should still succeed
	markdown := generateMarkdown(einstein, staffArch)
	assert.Contains(t, markdown, "unavailable:")

	// Write output
	outputPath := filepath.Join(tmpDir, outputFile)
	err := os.WriteFile(outputPath, []byte(markdown), 0644)
	require.NoError(t, err)

	// Verify output exists
	_, err = os.Stat(outputPath)
	assert.NoError(t, err)
}

func TestMainIntegrationMalformedJSON(t *testing.T) {
	// Create temporary team directory
	tmpDir := t.TempDir()

	// Write malformed JSON
	malformedPath := filepath.Join(tmpDir, einsteinStdoutFile)
	err := os.WriteFile(malformedPath, []byte("{invalid json"), 0644)
	require.NoError(t, err)

	staffArchPath := filepath.Join(tmpDir, staffArchStdoutFile)
	err = os.WriteFile(staffArchPath, []byte("{\"status\":\"complete\"}"), 0644)
	require.NoError(t, err)

	// Run extraction
	einstein := extractEinstein(malformedPath)
	staffArch := extractStaffArch(staffArchPath)

	// Einstein should fallback, staff-arch should extract what it can
	assert.Contains(t, einstein.ExecutiveSummary, "unavailable:")
	assert.Contains(t, einstein.ExecutiveSummary, "could not parse JSON")

	// Generate and write markdown
	markdown := generateMarkdown(einstein, staffArch)
	outputPath := filepath.Join(tmpDir, outputFile)
	err = os.WriteFile(outputPath, []byte(markdown), 0644)
	require.NoError(t, err)

	// Verify output
	outputData, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Contains(t, string(outputData), "unavailable:")
}

func TestValidateTeamDir(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T) string
		wantErr   bool
		errSubstr string
	}{
		{
			name: "valid_directory",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr: false,
		},
		{
			name: "nonexistent_directory",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent")
			},
			wantErr:   true,
			errSubstr: "not found",
		},
		{
			name: "not_a_directory",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				filePath := filepath.Join(dir, "regular_file")
				err := os.WriteFile(filePath, []byte("test"), 0644)
				require.NoError(t, err)
				return filePath
			},
			wantErr:   true,
			errSubstr: "not a directory",
		},
		{
			name: "relative_path_resolved",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				subDir := filepath.Join(dir, "sub")
				err := os.Mkdir(subDir, 0755)
				require.NoError(t, err)
				return filepath.Join(dir, "sub", "..", "sub")
			},
			wantErr: false,
		},
		{
			name: "dot_dot_path_cleaned",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				subDir := filepath.Join(dir, "a", "b")
				err := os.MkdirAll(subDir, 0755)
				require.NoError(t, err)
				// Path with .. that should resolve
				return filepath.Join(dir, "a", "b", "..", "b")
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			inputPath := tc.setup(t)
			result, err := validateTeamDir(inputPath)

			if tc.wantErr {
				assert.Error(t, err)
				if tc.errSubstr != "" {
					assert.Contains(t, err.Error(), tc.errSubstr)
				}
			} else {
				assert.NoError(t, err)
				assert.True(t, filepath.IsAbs(result), "expected absolute path, got: %s", result)
				assert.Equal(t, filepath.Clean(result), result)
			}
		})
	}
}
