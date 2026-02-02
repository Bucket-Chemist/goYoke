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
		Types:       []ProjectType{ProjectGo},
		Primary:     ProjectGo,
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

func TestDetectProjectType_GoTypescript(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "tsconfig.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"dependencies": {"typescript": "^5.0.0"}}`), 0644)

	result := DetectProjectType(tmpDir)

	// Check multiple types detected
	if len(result.Types) != 2 {
		t.Errorf("Expected 2 types, got %d: %v", len(result.Types), result.Types)
	}

	// Check Go is first (Primary)
	if result.Primary != ProjectGo {
		t.Errorf("Expected Primary=Go, got: %s", result.Primary)
	}

	// Check backward compatibility
	if result.Type != ProjectGo {
		t.Errorf("Expected Type=Go (backward compat), got: %s", result.Type)
	}

	// Check both types present
	hasGo := false
	hasTS := false
	for _, typ := range result.Types {
		if typ == ProjectGo {
			hasGo = true
		}
		if typ == ProjectTypeScript {
			hasTS = true
		}
	}
	if !hasGo || !hasTS {
		t.Errorf("Expected both Go and TypeScript types, got: %v", result.Types)
	}

	// Check conventions include both
	hasGoMd := false
	hasTSMd := false
	for _, conv := range result.Conventions {
		if conv == "go.md" {
			hasGoMd = true
		}
		if conv == "typescript.md" {
			hasTSMd = true
		}
	}
	if !hasGoMd || !hasTSMd {
		t.Errorf("Expected go.md and typescript.md conventions, got: %v", result.Conventions)
	}
}

func TestDetectProjectType_GoTypescriptReact(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "tsconfig.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{
		"dependencies": {
			"react": "^18.0.0",
			"typescript": "^5.0.0"
		}
	}`), 0644)

	result := DetectProjectType(tmpDir)

	// Check 3 types detected
	if len(result.Types) != 3 {
		t.Errorf("Expected 3 types, got %d: %v", len(result.Types), result.Types)
	}

	// Check Primary is still Go
	if result.Primary != ProjectGo {
		t.Errorf("Expected Primary=Go, got: %s", result.Primary)
	}

	// Check all three types present
	hasGo := false
	hasTS := false
	hasReact := false
	for _, typ := range result.Types {
		if typ == ProjectGo {
			hasGo = true
		}
		if typ == ProjectTypeScript {
			hasTS = true
		}
		if typ == ProjectReact {
			hasReact = true
		}
	}
	if !hasGo || !hasTS || !hasReact {
		t.Errorf("Expected Go, TypeScript, and React types, got: %v", result.Types)
	}

	// Check conventions include all three
	hasGoMd := false
	hasTSMd := false
	hasReactMd := false
	for _, conv := range result.Conventions {
		if conv == "go.md" {
			hasGoMd = true
		}
		if conv == "typescript.md" {
			hasTSMd = true
		}
		if conv == "react.md" {
			hasReactMd = true
		}
	}
	if !hasGoMd || !hasTSMd || !hasReactMd {
		t.Errorf("Expected go.md, typescript.md, and react.md conventions, got: %v", result.Conventions)
	}

	// Check indicators mention React dependency
	indicatorsStr := strings.Join(result.Indicators, ", ")
	if !strings.Contains(indicatorsStr, "react dependency") {
		t.Errorf("Expected indicators to mention react dependency, got: %s", indicatorsStr)
	}
}

func TestDetectProjectType_TypescriptReactOnly(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "tsconfig.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{
		"dependencies": {
			"react": "^18.0.0"
		}
	}`), 0644)

	result := DetectProjectType(tmpDir)

	// Check 2 types: TypeScript + React
	if len(result.Types) != 2 {
		t.Errorf("Expected 2 types, got %d: %v", len(result.Types), result.Types)
	}

	// Check Primary is TypeScript (first detected)
	if result.Primary != ProjectTypeScript {
		t.Errorf("Expected Primary=TypeScript, got: %s", result.Primary)
	}

	// Check both types present
	hasTS := false
	hasReact := false
	for _, typ := range result.Types {
		if typ == ProjectTypeScript {
			hasTS = true
		}
		if typ == ProjectReact {
			hasReact = true
		}
	}
	if !hasTS || !hasReact {
		t.Errorf("Expected TypeScript and React types, got: %v", result.Types)
	}
}

func TestFormatProjectType_Multiple(t *testing.T) {
	result := &ProjectDetectionResult{
		Types:       []ProjectType{ProjectGo, ProjectTypeScript, ProjectReact},
		Primary:     ProjectGo,
		Type:        ProjectGo,
		Indicators:  []string{"go.mod", "tsconfig.json", "package.json (react dependency)"},
		Conventions: []string{"go.md", "typescript.md", "react.md"},
	}

	formatted := FormatProjectType(result)

	// Check format contains " + " separator
	if !strings.Contains(formatted, "go + typescript + react") {
		t.Errorf("Expected 'go + typescript + react' format, got: %s", formatted)
	}

	// Check all conventions listed
	if !strings.Contains(formatted, "go.md") || !strings.Contains(formatted, "typescript.md") || !strings.Contains(formatted, "react.md") {
		t.Errorf("Expected all conventions listed, got: %s", formatted)
	}
}
