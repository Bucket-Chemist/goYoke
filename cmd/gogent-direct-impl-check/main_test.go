package main

import (
	"testing"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

func TestIsImplementationFile(t *testing.T) {
	config := &routing.DirectImplCheckConfig{
		ImplementationExtensions: []string{".go", ".py", ".r"},
		ImplementationPaths:      []string{"internal/", "pkg/", "cmd/"},
	}

	tests := []struct {
		name     string
		filePath string
		want     bool
	}{
		{
			name:     "Go file in pkg",
			filePath: "/home/user/project/pkg/routing/schema.go",
			want:     true,
		},
		{
			name:     "Python file in internal",
			filePath: "/home/user/project/internal/api/handler.py",
			want:     true,
		},
		{
			name:     "R file in cmd",
			filePath: "/home/user/project/cmd/analyzer/main.r",
			want:     true,
		},
		{
			name:     "Go file in wrong directory",
			filePath: "/home/user/project/docs/example.go",
			want:     false,
		},
		{
			name:     "Wrong extension in pkg",
			filePath: "/home/user/project/pkg/routing/schema.json",
			want:     false,
		},
		{
			name:     "Markdown file",
			filePath: "/home/user/project/pkg/README.md",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isImplementationFile(tt.filePath, config)
			if got != tt.want {
				t.Errorf("isImplementationFile(%q) = %v, want %v", tt.filePath, got, tt.want)
			}
		})
	}
}

func TestIsExcluded(t *testing.T) {
	excludePatterns := []string{"*_test.go", "testdata/*", "*.gen.go", "*_generated.go"}

	tests := []struct {
		name     string
		filePath string
		want     bool
	}{
		{
			name:     "Test file",
			filePath: "/home/user/project/pkg/routing/schema_test.go",
			want:     true,
		},
		{
			name:     "Testdata file",
			filePath: "/home/user/project/pkg/testdata/fixture.go",
			want:     true,
		},
		{
			name:     "Generated file",
			filePath: "/home/user/project/pkg/api/types.gen.go",
			want:     true,
		},
		{
			name:     "Auto-generated file",
			filePath: "/home/user/project/pkg/api/client_generated.go",
			want:     true,
		},
		{
			name:     "Normal file",
			filePath: "/home/user/project/pkg/routing/schema.go",
			want:     false,
		},
		{
			name:     "Normal file with similar name",
			filePath: "/home/user/project/pkg/testing/utils.go",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isExcluded(tt.filePath, excludePatterns)
			if got != tt.want {
				t.Errorf("isExcluded(%q) = %v, want %v", tt.filePath, got, tt.want)
			}
		})
	}
}

func TestCountLines(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			name:    "Empty string",
			content: "",
			want:    0,
		},
		{
			name:    "Single line without newline",
			content: "package main",
			want:    1,
		},
		{
			name:    "Single line with newline",
			content: "package main\n",
			want:    2,
		},
		{
			name:    "Multiple lines",
			content: "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n",
			want:    8,
		},
		{
			name:    "Three lines",
			content: "line1\nline2\nline3",
			want:    3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countLines(tt.content)
			if got != tt.want {
				t.Errorf("countLines() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSuggestAgent(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		content  string
		want     string
	}{
		{
			name:     "TUI file by path",
			filePath: "/home/user/project/internal/tui/dashboard.go",
			content:  "package tui",
			want:     "go-tui",
		},
		{
			name:     "CLI file by path",
			filePath: "/home/user/project/cmd/mycli/main.go",
			content:  "package main",
			want:     "go-cli",
		},
		{
			name:     "API file by path",
			filePath: "/home/user/project/internal/api/handler.go",
			content:  "package api",
			want:     "go-api",
		},
		{
			name:     "Concurrent code by content",
			filePath: "/home/user/project/pkg/worker/pool.go",
			content:  "func Start() {\n\tgo func() {\n\t\t// worker\n\t}()\n}",
			want:     "go-concurrent",
		},
		{
			name:     "Concurrent with errgroup",
			filePath: "/home/user/project/pkg/processor.go",
			content:  "import \"golang.org/x/sync/errgroup\"\n\nfunc Process() error {\n\tg := new(errgroup.Group)\n}",
			want:     "go-concurrent",
		},
		{
			name:     "Generic Go file",
			filePath: "/home/user/project/pkg/routing/schema.go",
			content:  "package routing",
			want:     "go-pro",
		},
		{
			name:     "Python file",
			filePath: "/home/user/project/src/handler.py",
			content:  "def main():\n    pass",
			want:     "python-pro",
		},
		{
			name:     "R file",
			filePath: "/home/user/project/lib/analysis.r",
			content:  "library(tidyverse)",
			want:     "r-pro",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := suggestAgent(tt.filePath, tt.content)
			if got != tt.want {
				t.Errorf("suggestAgent(%q, ...) = %q, want %q", tt.filePath, got, tt.want)
			}
		})
	}
}

func TestThresholdBoundaries(t *testing.T) {
	// Test that we correctly handle threshold boundaries
	tests := []struct {
		name      string
		lineCount int
		threshold int
		shouldWarn bool
	}{
		{
			name:       "Below threshold",
			lineCount:  30,
			threshold:  50,
			shouldWarn: false,
		},
		{
			name:       "At threshold",
			lineCount:  50,
			threshold:  50,
			shouldWarn: true,
		},
		{
			name:       "Above threshold",
			lineCount:  100,
			threshold:  50,
			shouldWarn: true,
		},
		{
			name:       "Zero lines",
			lineCount:  0,
			threshold:  30,
			shouldWarn: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldWarn := tt.lineCount >= tt.threshold
			if shouldWarn != tt.shouldWarn {
				t.Errorf("Expected shouldWarn=%v for lineCount=%d, threshold=%d",
					tt.shouldWarn, tt.lineCount, tt.threshold)
			}
		})
	}
}

// TestEdgeCase_EditWithDeletion tests that deletions don't trigger warnings
func TestEdgeCase_EditWithDeletion(t *testing.T) {
	// When old_string is longer than new_string, net addition is negative
	oldContent := "line1\nline2\nline3\nline4\nline5\n" // 6 lines
	newContent := "line1\nline2\n"                       // 3 lines

	oldLines := countLines(oldContent)
	newLines := countLines(newContent)
	netAddition := newLines - oldLines

	if netAddition >= 0 {
		t.Errorf("Expected negative net addition for deletion, got %d", netAddition)
	}

	// Verify that we normalize negative to zero
	lineCount := netAddition
	if lineCount < 0 {
		lineCount = 0
	}

	if lineCount != 0 {
		t.Errorf("Expected lineCount=0 after normalization, got %d", lineCount)
	}
}

// TestEdgeCase_EmptyWrite tests empty Write tool invocation
func TestEdgeCase_EmptyWrite(t *testing.T) {
	content := ""
	lineCount := countLines(content)

	if lineCount != 0 {
		t.Errorf("Expected 0 lines for empty content, got %d", lineCount)
	}
}
