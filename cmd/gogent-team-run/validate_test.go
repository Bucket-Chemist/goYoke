package main

import (
	"os"
	"path/filepath"
	"strings"
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

// strPtr returns a pointer to the given string
func strPtr(s string) *string { return &s }

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T) *TeamRunner
		wantErr   bool
		errSubstr string
	}{
		// on_complete_script tests
		{
			name: "null script passes",
			setup: func(t *testing.T) *TeamRunner {
				config := &TeamConfig{
					Waves: []Wave{
						{
							WaveNumber:       1,
							OnCompleteScript: nil,
							Members:          []Member{},
						},
					},
				}
				tr, _ := setupTestRunner(t, config)
				return tr
			},
			wantErr: false,
		},
		{
			name: "empty string script passes",
			setup: func(t *testing.T) *TeamRunner {
				config := &TeamConfig{
					Waves: []Wave{
						{
							WaveNumber:       1,
							OnCompleteScript: strPtr(""),
							Members:          []Member{},
						},
					},
				}
				tr, _ := setupTestRunner(t, config)
				return tr
			},
			wantErr: false,
		},
		{
			name: "bare name on PATH passes",
			setup: func(t *testing.T) *TeamRunner {
				config := &TeamConfig{
					Waves: []Wave{
						{
							WaveNumber:       1,
							OnCompleteScript: strPtr("echo"), // echo is always on PATH
							Members:          []Member{},
						},
					},
				}
				tr, _ := setupTestRunner(t, config)
				return tr
			},
			wantErr: false,
		},
		{
			name: "bare name not on PATH fails",
			setup: func(t *testing.T) *TeamRunner {
				config := &TeamConfig{
					Waves: []Wave{
						{
							WaveNumber:       1,
							OnCompleteScript: strPtr("nonexistent-xyz-binary-99"),
							Members:          []Member{},
						},
					},
				}
				tr, _ := setupTestRunner(t, config)
				return tr
			},
			wantErr:   true,
			errSubstr: "not found on PATH",
		},
		{
			name: "absolute path exists passes",
			setup: func(t *testing.T) *TeamRunner {
				// Create a temp script file
				tempDir := t.TempDir()
				scriptPath := filepath.Join(tempDir, "test-script.sh")
				err := os.WriteFile(scriptPath, []byte("#!/bin/bash\nexit 0\n"), 0755)
				if err != nil {
					t.Fatalf("failed to create test script: %v", err)
				}

				config := &TeamConfig{
					Waves: []Wave{
						{
							WaveNumber:       1,
							OnCompleteScript: &scriptPath,
							Members:          []Member{},
						},
					},
				}
				tr, _ := setupTestRunner(t, config)
				return tr
			},
			wantErr: false,
		},
		{
			name: "absolute path missing fails",
			setup: func(t *testing.T) *TeamRunner {
				config := &TeamConfig{
					Waves: []Wave{
						{
							WaveNumber:       1,
							OnCompleteScript: strPtr("/nonexistent/path/binary"),
							Members:          []Member{},
						},
					},
				}
				tr, _ := setupTestRunner(t, config)
				return tr
			},
			wantErr:   true,
			errSubstr: "no such file",
		},

		// stdin_file tests
		{
			name: "stdin_file exists passes",
			setup: func(t *testing.T) *TeamRunner {
				teamDir := t.TempDir()

				// Create stdin file in teamDir
				stdinFile := "agent-input.json"
				stdinPath := filepath.Join(teamDir, stdinFile)
				err := os.WriteFile(stdinPath, []byte(`{"test": "data"}`), 0644)
				if err != nil {
					t.Fatalf("failed to create stdin file: %v", err)
				}

				config := &TeamConfig{
					Waves: []Wave{
						{
							WaveNumber: 1,
							Members: []Member{
								{
									Name:      "agent-1",
									StdinFile: stdinFile,
								},
							},
						},
					},
				}

				// Manually create runner with specific teamDir
				tr := &TeamRunner{
					teamDir:   teamDir,
					config:    config,
					spawner:   &claudeSpawner{},
					childPIDs: make(map[int]struct{}),
				}
				return tr
			},
			wantErr: false,
		},
		{
			name: "stdin_file missing fails",
			setup: func(t *testing.T) *TeamRunner {
				config := &TeamConfig{
					Waves: []Wave{
						{
							WaveNumber: 1,
							Members: []Member{
								{
									Name:      "agent-1",
									StdinFile: "nonexistent.json",
								},
							},
						},
					},
				}
				tr, _ := setupTestRunner(t, config)
				return tr
			},
			wantErr:   true,
			errSubstr: "stdin_file",
		},
		{
			name: "stdin_file empty string passes",
			setup: func(t *testing.T) *TeamRunner {
				config := &TeamConfig{
					Waves: []Wave{
						{
							WaveNumber: 1,
							Members: []Member{
								{
									Name:      "agent-1",
									StdinFile: "",
								},
							},
						},
					},
				}
				tr, _ := setupTestRunner(t, config)
				return tr
			},
			wantErr: false,
		},

		// project_root tests
		{
			name: "project_root exists passes",
			setup: func(t *testing.T) *TeamRunner {
				projectRoot := t.TempDir()

				config := &TeamConfig{
					ProjectRoot: projectRoot,
					Waves:       []Wave{},
				}
				tr, _ := setupTestRunner(t, config)
				return tr
			},
			wantErr: false,
		},
		{
			name: "project_root missing fails",
			setup: func(t *testing.T) *TeamRunner {
				config := &TeamConfig{
					ProjectRoot: "/nonexistent-dir-xyz-999",
					Waves:       []Wave{},
				}
				tr, _ := setupTestRunner(t, config)
				return tr
			},
			wantErr:   true,
			errSubstr: "project_root",
		},

		// Multi-wave tests
		{
			name: "wave 1 valid wave 2 bad script",
			setup: func(t *testing.T) *TeamRunner {
				config := &TeamConfig{
					Waves: []Wave{
						{
							WaveNumber:       1,
							OnCompleteScript: nil, // valid
							Members:          []Member{},
						},
						{
							WaveNumber:       2,
							OnCompleteScript: strPtr("nonexistent-binary-xyz-123"), // bad
							Members:          []Member{},
						},
					},
				}
				tr, _ := setupTestRunner(t, config)
				return tr
			},
			wantErr:   true,
			errSubstr: "wave 2",
		},
		{
			name: "all waves valid passes",
			setup: func(t *testing.T) *TeamRunner {
				tempDir := t.TempDir()
				script1 := filepath.Join(tempDir, "script1.sh")
				script2 := filepath.Join(tempDir, "script2.sh")
				_ = os.WriteFile(script1, []byte("#!/bin/bash\nexit 0\n"), 0755)
				_ = os.WriteFile(script2, []byte("#!/bin/bash\nexit 0\n"), 0755)

				config := &TeamConfig{
					Waves: []Wave{
						{
							WaveNumber:       1,
							OnCompleteScript: &script1,
							Members:          []Member{},
						},
						{
							WaveNumber:       2,
							OnCompleteScript: &script2,
							Members:          []Member{},
						},
					},
				}
				tr, _ := setupTestRunner(t, config)
				return tr
			},
			wantErr: false,
		},

		// Edge cases
		{
			name: "nil config passes",
			setup: func(t *testing.T) *TeamRunner {
				teamDir := t.TempDir()
				tr := &TeamRunner{
					teamDir:   teamDir,
					config:    nil, // nil config
					spawner:   &claudeSpawner{},
					childPIDs: make(map[int]struct{}),
				}
				return tr
			},
			wantErr: false,
		},
		{
			name: "empty waves passes",
			setup: func(t *testing.T) *TeamRunner {
				config := &TeamConfig{
					Waves: []Wave{}, // empty waves
				}
				tr, _ := setupTestRunner(t, config)
				return tr
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := tt.setup(t)
			err := tr.ValidateConfig()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errSubstr)
				}
				if !strings.Contains(err.Error(), tt.errSubstr) {
					t.Fatalf("expected error containing %q, got: %v", tt.errSubstr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}
