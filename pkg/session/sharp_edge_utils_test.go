package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractCodeSnippet(t *testing.T) {
	// Create temp directory for test files
	tmpDir := t.TempDir()

	// Test file with known content
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main

import "fmt"

func main() {
	fmt.Println("line 6")
	fmt.Println("line 7")
	fmt.Println("line 8")
	fmt.Println("line 9")
	fmt.Println("line 10")
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name       string
		filePath   string
		lineNumber int
		window     int
		wantLines  int    // expected number of lines in result
		wantEmpty  bool   // expect empty result
		contains   string // substring that should be in result
	}{
		{
			name:       "valid file middle line",
			filePath:   testFile,
			lineNumber: 7,
			window:     2,
			wantLines:  5, // lines 5-9
			contains:   "line 7",
		},
		{
			name:       "line at beginning",
			filePath:   testFile,
			lineNumber: 1,
			window:     2,
			wantLines:  3, // lines 1-3 (can't go before line 1)
			contains:   "package main",
		},
		{
			name:       "line at end",
			filePath:   testFile,
			lineNumber: 11,
			window:     2,
			wantLines:  3, // lines 9-11 (can't go past EOF)
			contains:   "line 10",
		},
		{
			name:       "line past EOF",
			filePath:   testFile,
			lineNumber: 100,
			window:     2,
			wantLines:  3, // last few lines (window adjusted to EOF)
		},
		{
			name:       "file doesn't exist",
			filePath:   filepath.Join(tmpDir, "nonexistent.go"),
			lineNumber: 5,
			window:     2,
			wantEmpty:  true,
		},
		{
			name:       "window size 0",
			filePath:   testFile,
			lineNumber: 7,
			window:     0,
			wantLines:  1, // just the target line
			contains:   "line 7",
		},
		{
			name:       "large window",
			filePath:   testFile,
			lineNumber: 6,
			window:     100,
			wantLines:  11, // entire file
			contains:   "package main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractCodeSnippet(tt.filePath, tt.lineNumber, tt.window)
			if err != nil {
				t.Errorf("ExtractCodeSnippet() error = %v", err)
				return
			}

			if tt.wantEmpty {
				if got != "" {
					t.Errorf("ExtractCodeSnippet() = %q, want empty string", got)
				}
				return
			}

			lines := strings.Split(got, "\n")
			if len(lines) != tt.wantLines {
				t.Errorf("ExtractCodeSnippet() returned %d lines, want %d", len(lines), tt.wantLines)
			}

			if tt.contains != "" && !strings.Contains(got, tt.contains) {
				t.Errorf("ExtractCodeSnippet() result doesn't contain %q", tt.contains)
			}
		})
	}
}

func TestExtractCodeSnippet_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	emptyFile := filepath.Join(tmpDir, "empty.go")

	if err := os.WriteFile(emptyFile, []byte{}, 0644); err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	got, err := ExtractCodeSnippet(emptyFile, 1, 2)
	if err != nil {
		t.Errorf("ExtractCodeSnippet() error = %v", err)
	}
	if got != "" {
		t.Errorf("ExtractCodeSnippet() = %q, want empty string for empty file", got)
	}
}

func TestExtractCodeSnippet_BinaryFile(t *testing.T) {
	tmpDir := t.TempDir()
	binaryFile := filepath.Join(tmpDir, "binary.bin")

	// Create file with null bytes (binary indicator)
	content := []byte("line 1\nline 2\x00binary\nline 3\n")
	if err := os.WriteFile(binaryFile, content, 0644); err != nil {
		t.Fatalf("Failed to create binary file: %v", err)
	}

	got, err := ExtractCodeSnippet(binaryFile, 2, 2)
	if err != nil {
		t.Errorf("ExtractCodeSnippet() error = %v", err)
	}
	if got != "" {
		t.Errorf("ExtractCodeSnippet() = %q, want empty string for binary file", got)
	}
}

func TestExtractCodeSnippet_UnreadableFile(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tmpDir := t.TempDir()
	unreadableFile := filepath.Join(tmpDir, "unreadable.go")

	if err := os.WriteFile(unreadableFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Remove read permission
	if err := os.Chmod(unreadableFile, 0000); err != nil {
		t.Fatalf("Failed to chmod: %v", err)
	}
	defer os.Chmod(unreadableFile, 0644) // Cleanup

	got, err := ExtractCodeSnippet(unreadableFile, 1, 2)
	if err != nil {
		t.Errorf("ExtractCodeSnippet() error = %v, want nil", err)
	}
	if got != "" {
		t.Errorf("ExtractCodeSnippet() = %q, want empty string for unreadable file", got)
	}
}
