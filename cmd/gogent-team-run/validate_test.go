package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateStdout(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, teamDir string) string // Returns stdout path
		wantErr   bool
		errSubstr string
	}{
		{
			name: "Valid",
			setup: func(t *testing.T, teamDir string) string {
				stdoutPath := filepath.Join(teamDir, "output.json")
				content := `{
					"$schema": "https://example.com/schema.json",
					"status": "completed",
					"content": {"result": "success"}
				}`
				require.NoError(t, os.WriteFile(stdoutPath, []byte(content), 0644))
				return stdoutPath
			},
			wantErr: false,
		},
		{
			name: "MissingSchema",
			setup: func(t *testing.T, teamDir string) string {
				stdoutPath := filepath.Join(teamDir, "output.json")
				content := `{
					"status": "completed",
					"content": {"result": "success"}
				}`
				require.NoError(t, os.WriteFile(stdoutPath, []byte(content), 0644))
				return stdoutPath
			},
			wantErr:   true,
			errSubstr: "missing $schema field",
		},
		{
			name: "MissingStatus",
			setup: func(t *testing.T, teamDir string) string {
				stdoutPath := filepath.Join(teamDir, "output.json")
				content := `{
					"$schema": "https://example.com/schema.json",
					"content": {"result": "success"}
				}`
				require.NoError(t, os.WriteFile(stdoutPath, []byte(content), 0644))
				return stdoutPath
			},
			wantErr:   true,
			errSubstr: "missing status field",
		},
		{
			name: "InvalidJSON",
			setup: func(t *testing.T, teamDir string) string {
				stdoutPath := filepath.Join(teamDir, "output.json")
				content := `{not valid json`
				require.NoError(t, os.WriteFile(stdoutPath, []byte(content), 0644))
				return stdoutPath
			},
			wantErr:   true,
			errSubstr: "parse stdout JSON",
		},
		{
			name: "EmptyFile",
			setup: func(t *testing.T, teamDir string) string {
				stdoutPath := filepath.Join(teamDir, "output.json")
				require.NoError(t, os.WriteFile(stdoutPath, []byte(""), 0644))
				return stdoutPath
			},
			wantErr:   true,
			errSubstr: "stdout file is empty",
		},
		{
			name: "FileNotFound",
			setup: func(t *testing.T, teamDir string) string {
				return filepath.Join(teamDir, "nonexistent.json")
			},
			wantErr:   true,
			errSubstr: "read stdout file",
		},
		{
			name: "PathTraversal",
			setup: func(t *testing.T, teamDir string) string {
				// Attempt to escape teamDir
				return filepath.Join(teamDir, "../../../etc/passwd")
			},
			wantErr:   true,
			errSubstr: "stdout path security",
		},
		{
			name: "ExtraFields",
			setup: func(t *testing.T, teamDir string) string {
				stdoutPath := filepath.Join(teamDir, "output.json")
				content := `{
					"$schema": "https://example.com/schema.json",
					"status": "completed",
					"content": {"result": "success"},
					"extra_field": "allowed",
					"another_extra": 42
				}`
				require.NoError(t, os.WriteFile(stdoutPath, []byte(content), 0644))
				return stdoutPath
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			teamDir := t.TempDir()
			stdoutPath := tc.setup(t, teamDir)

			err := validateStdout(stdoutPath, teamDir)

			if tc.wantErr {
				require.Error(t, err)
				if tc.errSubstr != "" {
					assert.Contains(t, err.Error(), tc.errSubstr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateOutputPath(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T) (targetPath, baseDir string)
		wantErr   bool
		errSubstr string
	}{
		{
			name: "ValidRelativePath",
			setup: func(t *testing.T) (string, string) {
				baseDir := t.TempDir()
				targetPath := filepath.Join(baseDir, "subdir", "file.txt")
				return targetPath, baseDir
			},
			wantErr: false,
		},
		{
			name: "ValidAbsolutePath",
			setup: func(t *testing.T) (string, string) {
				baseDir := t.TempDir()
				subdir := filepath.Join(baseDir, "subdir")
				require.NoError(t, os.MkdirAll(subdir, 0755))
				targetPath := filepath.Join(subdir, "file.txt")
				return targetPath, baseDir
			},
			wantErr: false,
		},
		{
			name: "TraversalAttempt",
			setup: func(t *testing.T) (string, string) {
				baseDir := t.TempDir()
				targetPath := filepath.Join(baseDir, "../../../etc/passwd")
				return targetPath, baseDir
			},
			wantErr:   true,
			errSubstr: "escapes base directory",
		},
		{
			name: "TraversalWithDots",
			setup: func(t *testing.T) (string, string) {
				baseDir := t.TempDir()
				targetPath := filepath.Join(baseDir, "subdir", "..", "..", "outside.txt")
				return targetPath, baseDir
			},
			wantErr:   true,
			errSubstr: "escapes base directory",
		},
		{
			name: "ExactMatch",
			setup: func(t *testing.T) (string, string) {
				baseDir := t.TempDir()
				return baseDir, baseDir
			},
			wantErr: false,
		},
		{
			name: "DirectChild",
			setup: func(t *testing.T) (string, string) {
				baseDir := t.TempDir()
				targetPath := filepath.Join(baseDir, "file.txt")
				return targetPath, baseDir
			},
			wantErr: false,
		},
		{
			name: "DeepNesting",
			setup: func(t *testing.T) (string, string) {
				baseDir := t.TempDir()
				targetPath := filepath.Join(baseDir, "a", "b", "c", "d", "file.txt")
				return targetPath, baseDir
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			targetPath, baseDir := tc.setup(t)

			err := validateOutputPath(targetPath, baseDir)

			if tc.wantErr {
				require.Error(t, err)
				if tc.errSubstr != "" {
					assert.Contains(t, err.Error(), tc.errSubstr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
