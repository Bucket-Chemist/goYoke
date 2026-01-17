package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetCurrentTier(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := t.TempDir()
	testTierFile := filepath.Join(tmpDir, "current-tier")

	tests := []struct {
		name        string
		fileContent string
		createFile  bool
		wantTier    string
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid tier: haiku",
			fileContent: "haiku",
			createFile:  true,
			wantTier:    "haiku",
			wantErr:     false,
		},
		{
			name:        "valid tier: haiku_thinking",
			fileContent: "haiku_thinking",
			createFile:  true,
			wantTier:    "haiku_thinking",
			wantErr:     false,
		},
		{
			name:        "valid tier: sonnet",
			fileContent: "sonnet",
			createFile:  true,
			wantTier:    "sonnet",
			wantErr:     false,
		},
		{
			name:        "valid tier: opus",
			fileContent: "opus",
			createFile:  true,
			wantTier:    "opus",
			wantErr:     false,
		},
		{
			name:        "valid tier: external",
			fileContent: "external",
			createFile:  true,
			wantTier:    "external",
			wantErr:     false,
		},
		{
			name:        "missing file returns external",
			fileContent: "",
			createFile:  false,
			wantTier:    "external",
			wantErr:     false,
		},
		{
			name:        "empty file returns external",
			fileContent: "",
			createFile:  true,
			wantTier:    "external",
			wantErr:     false,
		},
		{
			name:        "whitespace only returns external",
			fileContent: "   \n\t  ",
			createFile:  true,
			wantTier:    "external",
			wantErr:     false,
		},
		{
			name:        "leading whitespace trimmed",
			fileContent: "  sonnet",
			createFile:  true,
			wantTier:    "sonnet",
			wantErr:     false,
		},
		{
			name:        "trailing whitespace trimmed",
			fileContent: "haiku\n",
			createFile:  true,
			wantTier:    "haiku",
			wantErr:     false,
		},
		{
			name:        "leading and trailing whitespace trimmed",
			fileContent: "\n  opus  \n",
			createFile:  true,
			wantTier:    "opus",
			wantErr:     false,
		},
		{
			name:        "invalid tier value",
			fileContent: "invalid_tier",
			createFile:  true,
			wantTier:    "",
			wantErr:     true,
			errContains: "Invalid tier value",
		},
		{
			name:        "uppercase tier (invalid)",
			fileContent: "SONNET",
			createFile:  true,
			wantTier:    "",
			wantErr:     true,
			errContains: "Invalid tier value",
		},
		{
			name:        "mixed case tier (invalid)",
			fileContent: "Haiku",
			createFile:  true,
			wantTier:    "",
			wantErr:     true,
			errContains: "Invalid tier value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up test file before each test
			os.Remove(testTierFile)

			// Create file if test requires it
			if tt.createFile {
				err := os.WriteFile(testTierFile, []byte(tt.fileContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			}

			// Execute GetCurrentTierFromPath (testing internal function)
			gotTier, err := GetCurrentTierFromPath(testTierFile)

			// Check error expectation
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetCurrentTier() expected error, got nil")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("GetCurrentTier() error = %v, should contain %q", err, tt.errContains)
				}
				return
			}

			// Check no error when not expected
			if err != nil {
				t.Errorf("GetCurrentTier() unexpected error: %v", err)
				return
			}

			// Check tier value
			if gotTier != tt.wantTier {
				t.Errorf("GetCurrentTier() = %v, want %v", gotTier, tt.wantTier)
			}
		})
	}
}

func TestIsValidTier(t *testing.T) {
	tests := []struct {
		name  string
		tier  string
		want  bool
	}{
		{name: "haiku is valid", tier: "haiku", want: true},
		{name: "haiku_thinking is valid", tier: "haiku_thinking", want: true},
		{name: "sonnet is valid", tier: "sonnet", want: true},
		{name: "opus is valid", tier: "opus", want: true},
		{name: "external is valid", tier: "external", want: true},
		{name: "invalid tier", tier: "invalid", want: false},
		{name: "empty tier", tier: "", want: false},
		{name: "uppercase tier", tier: "SONNET", want: false},
		{name: "mixed case tier", tier: "Haiku", want: false},
		{name: "whitespace tier", tier: "  sonnet  ", want: false}, // Should be trimmed before validation
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidTier(tt.tier); got != tt.want {
				t.Errorf("isValidTier(%q) = %v, want %v", tt.tier, got, tt.want)
			}
		})
	}
}
