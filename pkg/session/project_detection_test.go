package session

import (
	"os"
	"path/filepath"
	"strings"
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

func TestDetectProjectType_RShiny_Standalone(t *testing.T) {
	tmpDir := t.TempDir()
	// Create .R file to trigger R detection
	os.WriteFile(filepath.Join(tmpDir, "utils.R"), []byte("# Helper functions"), 0644)
	// Create app.R without DESCRIPTION
	os.WriteFile(filepath.Join(tmpDir, "app.R"), []byte("# Standalone Shiny app"), 0644)

	result := DetectProjectType(tmpDir)

	if result.Type != ProjectRShiny {
		t.Errorf("Expected R+Shiny from standalone app.R, got: %s", result.Type)
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
		t.Errorf("Expected R.md and R-shiny.md conventions for standalone Shiny, got: %v", result.Conventions)
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
