package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Bucket-Chemist/GOgent-Fortress/test/simulation/harness"
)

// TestFindHarnessDir tests harness directory detection.
func TestFindHarnessDir(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() (cleanup func())
		want    string
		wantErr bool
	}{
		{
			name: "current directory has types.go",
			setup: func() func() {
				// Save original dir
				origDir, _ := os.Getwd()
				// Change to harness directory (two levels up from cmd/harness)
				os.Chdir("../..")
				return func() { os.Chdir(origDir) }
			},
			want:    ".",
			wantErr: false,
		},
		{
			name: "no harness directory found",
			setup: func() func() {
				// Create a temp dir without types.go
				tempDir := t.TempDir()
				origDir, _ := os.Getwd()
				os.Chdir(tempDir)
				return func() { os.Chdir(origDir) }
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setup()
			defer cleanup()

			got, err := findHarnessDir()
			if (err != nil) != tt.wantErr {
				t.Errorf("findHarnessDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("findHarnessDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestFindBinary tests binary discovery logic.
func TestFindBinary(t *testing.T) {
	tests := []struct {
		name       string
		binaryName string
		setup      func() (cleanup func())
		wantErr    bool
	}{
		{
			name:       "binary in ~/.local/bin",
			binaryName: "test-binary-local",
			setup: func() func() {
				homeDir, _ := os.UserHomeDir()
				binDir := filepath.Join(homeDir, ".local", "bin")
				os.MkdirAll(binDir, 0755)
				binPath := filepath.Join(binDir, "test-binary-local")
				os.WriteFile(binPath, []byte("#!/bin/bash\necho test"), 0755)
				return func() { os.Remove(binPath) }
			},
			wantErr: false,
		},
		{
			name:       "binary in PATH",
			binaryName: "sh", // sh is universally available
			setup:      func() func() { return func() {} },
			wantErr:    false,
		},
		{
			name:       "binary not found",
			binaryName: "nonexistent-binary-12345",
			setup:      func() func() { return func() {} },
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setup()
			defer cleanup()

			got, err := findBinary(tt.binaryName)
			if (err != nil) != tt.wantErr {
				t.Errorf("findBinary() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == "" {
				t.Errorf("findBinary() returned empty path")
			}
		})
	}
}

// TestParseFilter tests filter string parsing.
func TestParseFilter(t *testing.T) {
	tests := []struct {
		name   string
		filter string
		want   []string
	}{
		{
			name:   "empty filter",
			filter: "",
			want:   nil,
		},
		{
			name:   "single value",
			filter: "V00",
			want:   []string{"V00"},
		},
		{
			name:   "multiple values",
			filter: "V00,V01,V02",
			want:   []string{"V00", "V01", "V02"},
		},
		{
			name:   "values with spaces",
			filter: "V00, V01 , V02",
			want:   []string{"V00", "V01", "V02"},
		},
		{
			name:   "trailing comma",
			filter: "V00,V01,",
			want:   []string{"V00", "V01"},
		},
		{
			name:   "empty values between commas",
			filter: "V00,,V01",
			want:   []string{"V00", "V01"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseFilter(tt.filter)
			if len(got) != len(tt.want) {
				t.Errorf("parseFilter() length = %d, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseFilter()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestValidateConfig tests configuration validation.
func TestValidateConfig(t *testing.T) {
	baseConfig := harness.SimulationConfig{
		Mode:           "deterministic",
		FuzzIterations: 1000,
		FuzzTimeout:    300000000000, // 5 minutes in nanoseconds
		ReportFormat:   "json",
	}

	tests := []struct {
		name    string
		modify  func(*harness.SimulationConfig)
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			modify:  func(c *harness.SimulationConfig) {},
			wantErr: false,
		},
		{
			name: "invalid mode",
			modify: func(c *harness.SimulationConfig) {
				c.Mode = "invalid"
			},
			wantErr: true,
			errMsg:  "invalid mode",
		},
		{
			name: "invalid report format",
			modify: func(c *harness.SimulationConfig) {
				c.ReportFormat = "xml"
			},
			wantErr: true,
			errMsg:  "invalid report format",
		},
		{
			name: "zero iterations",
			modify: func(c *harness.SimulationConfig) {
				c.FuzzIterations = 0
			},
			wantErr: true,
			errMsg:  "iterations must be positive",
		},
		{
			name: "negative iterations",
			modify: func(c *harness.SimulationConfig) {
				c.FuzzIterations = -1
			},
			wantErr: true,
			errMsg:  "iterations must be positive",
		},
		{
			name: "zero timeout",
			modify: func(c *harness.SimulationConfig) {
				c.FuzzTimeout = 0
			},
			wantErr: true,
			errMsg:  "timeout must be positive",
		},
		{
			name: "negative timeout",
			modify: func(c *harness.SimulationConfig) {
				c.FuzzTimeout = -1
			},
			wantErr: true,
			errMsg:  "timeout must be positive",
		},
		{
			name: "valid fuzz mode",
			modify: func(c *harness.SimulationConfig) {
				c.Mode = "fuzz"
			},
			wantErr: false,
		},
		{
			name: "valid mixed mode",
			modify: func(c *harness.SimulationConfig) {
				c.Mode = "mixed"
			},
			wantErr: false,
		},
		{
			name: "valid markdown format",
			modify: func(c *harness.SimulationConfig) {
				c.ReportFormat = "markdown"
			},
			wantErr: false,
		},
		{
			name: "valid tap format",
			modify: func(c *harness.SimulationConfig) {
				c.ReportFormat = "tap"
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := baseConfig
			tt.modify(&cfg)

			err := validateConfig(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("validateConfig() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

// TestReportExtension tests report file extension mapping.
func TestReportExtension(t *testing.T) {
	tests := []struct {
		name   string
		format string
		want   string
	}{
		{
			name:   "json format",
			format: "json",
			want:   "json",
		},
		{
			name:   "markdown format",
			format: "markdown",
			want:   "md",
		},
		{
			name:   "tap format",
			format: "tap",
			want:   "tap",
		},
		{
			name:   "unknown format defaults to json",
			format: "unknown",
			want:   "json",
		},
		{
			name:   "empty format defaults to json",
			format: "",
			want:   "json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reportExtension(tt.format)
			if got != tt.want {
				t.Errorf("reportExtension() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestModeSelection tests that mode selection logic routes correctly.
func TestModeSelection(t *testing.T) {
	// This tests the switch statement in main() indirectly through helper functions
	cfg := harness.DefaultConfig()
	cfg.TempDir = t.TempDir()

	// Create mock generator and runner
	gen := harness.NewGenerator(cfg.TempDir)
	runner := harness.NewRunner(cfg, "/bin/true", "/bin/true", gen)

	tests := []struct {
		name    string
		mode    string
		wantErr bool
	}{
		{
			name:    "deterministic mode",
			mode:    "deterministic",
			wantErr: false,
		},
		// Note: fuzz and mixed modes require functional binaries, tested in integration
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg.Mode = tt.mode

			_, err := runDeterministic(cfg, runner)
			if (err != nil) != tt.wantErr {
				t.Errorf("runDeterministic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestConfigDefaults tests that default configuration values are set correctly.
func TestConfigDefaults(t *testing.T) {
	// Simulate main() flag defaults by checking against harness.DefaultConfig
	defaults := harness.DefaultConfig()

	if defaults.Mode != "deterministic" {
		t.Errorf("default mode = %v, want deterministic", defaults.Mode)
	}
	if defaults.FuzzIterations != 1000 {
		t.Errorf("default iterations = %v, want 1000", defaults.FuzzIterations)
	}
	if defaults.ReportFormat != "json" {
		t.Errorf("default report format = %v, want json", defaults.ReportFormat)
	}
}

// TestFilterParsing tests comprehensive filter parsing edge cases.
func TestFilterParsing(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		verify func(t *testing.T, result []string)
	}{
		{
			name:  "nil for empty string",
			input: "",
			verify: func(t *testing.T, result []string) {
				if result != nil {
					t.Errorf("expected nil for empty string, got %v", result)
				}
			},
		},
		{
			name:  "single filter preserved",
			input: "FUZZ-P",
			verify: func(t *testing.T, result []string) {
				if len(result) != 1 || result[0] != "FUZZ-P" {
					t.Errorf("expected [FUZZ-P], got %v", result)
				}
			},
		},
		{
			name:  "whitespace trimmed",
			input: "  V00  ,  V01  ",
			verify: func(t *testing.T, result []string) {
				if len(result) != 2 || result[0] != "V00" || result[1] != "V01" {
					t.Errorf("expected [V00 V01], got %v", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFilter(tt.input)
			tt.verify(t, result)
		})
	}
}

// TestValidationBoundaries tests validation at boundary conditions.
func TestValidationBoundaries(t *testing.T) {
	tests := []struct {
		name       string
		iterations int
		wantErr    bool
	}{
		{
			name:       "iterations = 1 (minimum valid)",
			iterations: 1,
			wantErr:    false,
		},
		{
			name:       "iterations = 0 (boundary invalid)",
			iterations: 0,
			wantErr:    true,
		},
		{
			name:       "iterations = -1 (negative invalid)",
			iterations: -1,
			wantErr:    true,
		},
		{
			name:       "iterations = 1000000 (very large valid)",
			iterations: 1000000,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := harness.DefaultConfig()
			cfg.FuzzIterations = tt.iterations

			err := validateConfig(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() with iterations=%d error = %v, wantErr %v",
					tt.iterations, err, tt.wantErr)
			}
		})
	}
}

// TestBinaryDiscoveryPriority tests that ~/.local/bin is checked before PATH.
func TestBinaryDiscoveryPriority(t *testing.T) {
	// Create a binary in ~/.local/bin
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot get home directory")
	}

	localBinDir := filepath.Join(homeDir, ".local", "bin")
	os.MkdirAll(localBinDir, 0755)

	testBinary := "test-priority-binary"
	localPath := filepath.Join(localBinDir, testBinary)
	os.WriteFile(localPath, []byte("#!/bin/bash\necho local"), 0755)
	defer os.Remove(localPath)

	// Find the binary
	found, err := findBinary(testBinary)
	if err != nil {
		t.Fatalf("findBinary() error = %v", err)
	}

	// Verify it found the ~/.local/bin version
	if !strings.Contains(found, ".local/bin") {
		t.Errorf("findBinary() = %v, expected path containing .local/bin", found)
	}
}

// TestReportExtensionExhaustive tests all report format mappings.
func TestReportExtensionExhaustive(t *testing.T) {
	formats := []struct {
		input string
		want  string
	}{
		{"json", "json"},
		{"markdown", "md"},
		{"tap", "tap"},
		{"JSON", "json"},       // case sensitivity
		{"MARKDOWN", "json"},   // case sensitivity
		{"invalid", "json"},    // unknown
		{"", "json"},           // empty
		{"md", "json"},         // abbreviation not recognized
		{"txt", "json"},        // other format
	}

	for _, f := range formats {
		t.Run(f.input, func(t *testing.T) {
			got := reportExtension(f.input)
			if got != f.want {
				t.Errorf("reportExtension(%q) = %q, want %q", f.input, got, f.want)
			}
		})
	}
}

// TestFindBinary_LoadContext tests binary discovery for gogent-load-context.
// This test verifies findBinary can locate gogent-load-context when it exists
// in expected locations (bin/ or PATH).
func TestFindBinary_LoadContext(t *testing.T) {
	// Create temp bin directory
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "bin", "gogent-load-context")
	os.MkdirAll(filepath.Dir(binPath), 0755)

	// Create mock binary
	os.WriteFile(binPath, []byte("#!/bin/bash\necho 'mock'"), 0755)

	// Save and restore working directory
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	path, err := findBinary("gogent-load-context")
	if err != nil {
		t.Fatalf("findBinary failed: %v", err)
	}

	if !strings.Contains(path, "gogent-load-context") {
		t.Errorf("Expected path to contain binary name, got: %s", path)
	}
}
