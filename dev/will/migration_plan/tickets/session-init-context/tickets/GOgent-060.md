---
id: GOgent-060
title: Project Type Detection
description: **Task**:
status: pending
time_estimate: 1.5h
dependencies: []
priority: MEDIUM
week: 4
tags:
  - session-start
  - week-4
tests_required: true
acceptance_criteria_count: 14
---

## GOgent-060: Project Type Detection

**Time**: 1.5 hours
**Dependencies**: None
**Priority**: MEDIUM

**Task**:
Auto-detect project type (Python, R, R+Shiny, JavaScript, Go) for convention loading in CLAUDE.md Gate 1.

**File**: `pkg/session/project_detection.go` (new file)

**Implementation**:
```go
package session

import (
	"os"
	"path/filepath"
	"strings"
)

// ProjectType represents detected project language/framework
type ProjectType string

const (
	ProjectGeneric    ProjectType = "generic"
	ProjectPython     ProjectType = "python"
	ProjectR          ProjectType = "r"
	ProjectRShiny     ProjectType = "r-shiny"
	ProjectRGolem     ProjectType = "r-golem"
	ProjectJavaScript ProjectType = "javascript"
	ProjectTypeScript ProjectType = "typescript"
	ProjectGo         ProjectType = "go"
	ProjectRust       ProjectType = "rust"
)

// ProjectDetectionResult contains detection output with metadata
type ProjectDetectionResult struct {
	Type        ProjectType `json:"type"`
	Indicators  []string    `json:"indicators"` // Files that triggered detection
	Conventions []string    `json:"conventions"` // Convention files to load
}

// DetectProjectType auto-detects project type from indicator files.
// Detection priority: Go > Python > R (with Shiny/Golem) > JavaScript/TypeScript > Rust > Generic
func DetectProjectType(projectDir string) *ProjectDetectionResult {
	result := &ProjectDetectionResult{
		Type:        ProjectGeneric,
		Indicators:  []string{},
		Conventions: []string{},
	}

	// Go detection (highest priority - this is GOgent-Fortress)
	if fileExists(filepath.Join(projectDir, "go.mod")) {
		result.Type = ProjectGo
		result.Indicators = append(result.Indicators, "go.mod")
		result.Conventions = []string{"go.md"}
		return result
	}

	// Python detection
	pythonIndicators := []string{"pyproject.toml", "setup.py", "requirements.txt", "uv.lock", "Pipfile"}
	for _, indicator := range pythonIndicators {
		if fileExists(filepath.Join(projectDir, indicator)) {
			result.Type = ProjectPython
			result.Indicators = append(result.Indicators, indicator)
			result.Conventions = []string{"python.md"}
			return result
		}
	}

	// R detection (with Shiny/Golem variants)
	rIndicators := []string{"DESCRIPTION", "NAMESPACE", "renv.lock"}
	for _, indicator := range rIndicators {
		if fileExists(filepath.Join(projectDir, indicator)) {
			result.Type = ProjectR
			result.Indicators = append(result.Indicators, indicator)
			result.Conventions = []string{"R.md"}

			// Check for Golem (superset of Shiny)
			if isGolemProject(projectDir) {
				result.Type = ProjectRGolem
				result.Conventions = []string{"R.md", "R-shiny.md", "R-golem.md"}
				result.Indicators = append(result.Indicators, "inst/golem-config.yml or golem dependency")
				return result
			}

			// Check for Shiny
			if isShinyProject(projectDir) {
				result.Type = ProjectRShiny
				result.Conventions = []string{"R.md", "R-shiny.md"}
				result.Indicators = append(result.Indicators, "shiny dependency or app.R/ui.R")
			}

			return result
		}
	}

	// Also check for standalone R files without DESCRIPTION
	if hasRFiles(projectDir) {
		// Check for Shiny indicators even without DESCRIPTION
		if fileExists(filepath.Join(projectDir, "app.R")) || fileExists(filepath.Join(projectDir, "ui.R")) {
			result.Type = ProjectRShiny
			result.Indicators = append(result.Indicators, "app.R or ui.R (standalone Shiny)")
			result.Conventions = []string{"R.md", "R-shiny.md"}
			return result
		}
	}

	// TypeScript detection (before JavaScript - more specific)
	if fileExists(filepath.Join(projectDir, "tsconfig.json")) {
		result.Type = ProjectTypeScript
		result.Indicators = append(result.Indicators, "tsconfig.json")
		result.Conventions = []string{"typescript.md"}
		return result
	}

	// JavaScript detection
	if fileExists(filepath.Join(projectDir, "package.json")) {
		result.Type = ProjectJavaScript
		result.Indicators = append(result.Indicators, "package.json")
		result.Conventions = []string{"javascript.md"}
		return result
	}

	// Rust detection
	if fileExists(filepath.Join(projectDir, "Cargo.toml")) {
		result.Type = ProjectRust
		result.Indicators = append(result.Indicators, "Cargo.toml")
		result.Conventions = []string{"rust.md"}
		return result
	}

	return result
}

// fileExists checks if file exists (not directory)
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// hasRFiles checks for any .R files in project root
func hasRFiles(projectDir string) bool {
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".r") {
			return true
		}
	}
	return false
}

// isShinyProject checks for Shiny indicators
func isShinyProject(projectDir string) bool {
	// Check for app.R or ui.R
	if fileExists(filepath.Join(projectDir, "app.R")) ||
		fileExists(filepath.Join(projectDir, "ui.R")) {
		return true
	}

	// Check DESCRIPTION for shiny dependency
	descPath := filepath.Join(projectDir, "DESCRIPTION")
	if content, err := os.ReadFile(descPath); err == nil {
		contentLower := strings.ToLower(string(content))
		if strings.Contains(contentLower, "shiny") {
			return true
		}
	}

	return false
}

// isGolemProject checks for Golem framework indicators
func isGolemProject(projectDir string) bool {
	// Check for golem-config.yml
	if fileExists(filepath.Join(projectDir, "inst", "golem-config.yml")) {
		return true
	}

	// Check DESCRIPTION for golem dependency
	descPath := filepath.Join(projectDir, "DESCRIPTION")
	if content, err := os.ReadFile(descPath); err == nil {
		contentLower := strings.ToLower(string(content))
		if strings.Contains(contentLower, "golem") {
			return true
		}
	}

	return false
}

// FormatProjectType returns human-readable project type string for context injection
func FormatProjectType(result *ProjectDetectionResult) string {
	if result.Type == ProjectGeneric {
		return "PROJECT TYPE: Generic (no language-specific conventions)"
	}

	conventions := strings.Join(result.Conventions, ", ")
	indicators := strings.Join(result.Indicators, ", ")

	return fmt.Sprintf("PROJECT TYPE: %s\n  Detected via: %s\n  Conventions: %s",
		string(result.Type),
		indicators,
		conventions,
	)
}
```

**Add import to file**:
```go
import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)
```

**Tests**: `pkg/session/project_detection_test.go` (new file)

```go
package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectProjectType_Go(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644)

	result := DetectProjectType(tmpDir)

	if result.Type != ProjectGo {
		t.Errorf("Expected Go, got: %s", result.Type)
	}

	if len(result.Indicators) == 0 || result.Indicators[0] != "go.mod" {
		t.Error("Should have go.mod as indicator")
	}

	if len(result.Conventions) == 0 || result.Conventions[0] != "go.md" {
		t.Error("Should have go.md convention")
	}
}

func TestDetectProjectType_Python(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), []byte("[project]"), 0644)

	result := DetectProjectType(tmpDir)

	if result.Type != ProjectPython {
		t.Errorf("Expected Python, got: %s", result.Type)
	}
}

func TestDetectProjectType_R(t *testing.T) {
	tmpDir := t.TempDir()
	descContent := `Package: mypackage
Title: Test Package
Version: 1.0.0
`
	os.WriteFile(filepath.Join(tmpDir, "DESCRIPTION"), []byte(descContent), 0644)

	result := DetectProjectType(tmpDir)

	if result.Type != ProjectR {
		t.Errorf("Expected R, got: %s", result.Type)
	}
}

func TestDetectProjectType_RShiny_Description(t *testing.T) {
	tmpDir := t.TempDir()
	descContent := `Package: myapp
Title: Shiny App
Version: 1.0.0
Imports: shiny
`
	os.WriteFile(filepath.Join(tmpDir, "DESCRIPTION"), []byte(descContent), 0644)

	result := DetectProjectType(tmpDir)

	if result.Type != ProjectRShiny {
		t.Errorf("Expected R+Shiny, got: %s", result.Type)
	}

	// Check conventions include both R.md and R-shiny.md
	hasR := false
	hasShiny := false
	for _, c := range result.Conventions {
		if c == "R.md" {
			hasR = true
		}
		if c == "R-shiny.md" {
			hasShiny = true
		}
	}
	if !hasR || !hasShiny {
		t.Errorf("Expected R.md and R-shiny.md conventions, got: %v", result.Conventions)
	}
}

func TestDetectProjectType_RShiny_AppFile(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "DESCRIPTION"), []byte("Package: test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "app.R"), []byte("# Shiny app"), 0644)

	result := DetectProjectType(tmpDir)

	if result.Type != ProjectRShiny {
		t.Errorf("Expected R+Shiny from app.R, got: %s", result.Type)
	}
}

func TestDetectProjectType_RGolem(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "DESCRIPTION"), []byte("Package: test\nImports: golem"), 0644)

	result := DetectProjectType(tmpDir)

	if result.Type != ProjectRGolem {
		t.Errorf("Expected R+Golem, got: %s", result.Type)
	}

	// Check conventions include all three
	if len(result.Conventions) != 3 {
		t.Errorf("Expected 3 conventions for Golem, got: %v", result.Conventions)
	}
}

func TestDetectProjectType_TypeScript(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "tsconfig.json"), []byte("{}"), 0644)

	result := DetectProjectType(tmpDir)

	if result.Type != ProjectTypeScript {
		t.Errorf("Expected TypeScript, got: %s", result.Type)
	}
}

func TestDetectProjectType_JavaScript(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte("{}"), 0644)

	result := DetectProjectType(tmpDir)

	if result.Type != ProjectJavaScript {
		t.Errorf("Expected JavaScript, got: %s", result.Type)
	}
}

func TestDetectProjectType_Rust(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte("[package]"), 0644)

	result := DetectProjectType(tmpDir)

	if result.Type != ProjectRust {
		t.Errorf("Expected Rust, got: %s", result.Type)
	}
}

func TestDetectProjectType_Generic(t *testing.T) {
	tmpDir := t.TempDir()
	// No language indicators

	result := DetectProjectType(tmpDir)

	if result.Type != ProjectGeneric {
		t.Errorf("Expected Generic, got: %s", result.Type)
	}
}

func TestDetectProjectType_Priority_GoOverPython(t *testing.T) {
	tmpDir := t.TempDir()
	// Both Go and Python indicators
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), []byte("[project]"), 0644)

	result := DetectProjectType(tmpDir)

	// Go should take priority
	if result.Type != ProjectGo {
		t.Errorf("Go should have priority over Python, got: %s", result.Type)
	}
}

func TestFormatProjectType(t *testing.T) {
	result := &ProjectDetectionResult{
		Type:        ProjectGo,
		Indicators:  []string{"go.mod"},
		Conventions: []string{"go.md"},
	}

	formatted := FormatProjectType(result)

	if formatted == "" {
		t.Error("Should return non-empty formatted string")
	}

	if !strings.Contains(formatted, "go") {
		t.Error("Should contain project type")
	}

	if !strings.Contains(formatted, "go.mod") {
		t.Error("Should contain indicator")
	}
}
```

**Acceptance Criteria**:
- [ ] `DetectProjectType()` detects: Go, Python, R, R+Shiny, R+Golem, JavaScript, TypeScript, Rust
- [ ] Returns generic for unrecognized projects
- [ ] Go has highest priority (this is GOgent-Fortress)
- [ ] Shiny detection checks DESCRIPTION content AND app.R/ui.R presence
- [ ] Golem detection checks inst/golem-config.yml AND DESCRIPTION
- [ ] `ProjectDetectionResult` includes indicators and convention files
- [ ] Tests verify all project types and priority
- [ ] `go test ./pkg/session/...` passes

**Test Deliverables**:
- [ ] Test file created: `pkg/session/project_detection_test.go`
- [ ] Test file size: ~180 lines
- [ ] Number of test functions: 13
- [ ] Coverage achieved: >90%
- [ ] Tests passing: ✅
- [ ] **ECOSYSTEM TEST PASS REQUIRED**: `make test-ecosystem`

**Why This Matters**: Project type detection drives convention loading (python.md, R.md, go.md) in CLAUDE.md Gate 1. Accurate detection ensures correct coding standards.

---
