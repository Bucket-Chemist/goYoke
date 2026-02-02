package session

import (
	"fmt"
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
	ProjectReact      ProjectType = "react"
	ProjectGo         ProjectType = "go"
	ProjectRust       ProjectType = "rust"
)

// ProjectDetectionResult contains detection output with metadata
type ProjectDetectionResult struct {
	Types       []ProjectType `json:"types"`       // Multiple types (additive detection)
	Primary     ProjectType   `json:"primary"`     // First detected (for routing priority)
	Type        ProjectType   `json:"type"`        // DEPRECATED: alias for Primary (backward compat)
	Indicators  []string      `json:"indicators"`  // Files that triggered detection
	Conventions []string      `json:"conventions"` // Convention files to load
}

// DetectProjectType auto-detects project type from indicator files.
// Uses additive detection - can detect multiple languages in polyglot projects.
// Detection order (determines Primary): Go > Python > R > Rust > TypeScript/React > JavaScript
// Backend languages prioritized over frontend for routing purposes.
func DetectProjectType(projectDir string) *ProjectDetectionResult {
	result := &ProjectDetectionResult{
		Types:       []ProjectType{},
		Primary:     ProjectGeneric,
		Type:        ProjectGeneric, // Backward compatibility
		Indicators:  []string{},
		Conventions: []string{},
	}

	conventionsMap := make(map[string]bool) // For deduplication

	// === BACKEND LANGUAGES (priority for routing) ===

	// Go detection (highest priority - this is GOgent-Fortress)
	if fileExists(filepath.Join(projectDir, "go.mod")) {
		result.Types = append(result.Types, ProjectGo)
		result.Indicators = append(result.Indicators, "go.mod")
		conventionsMap["go.md"] = true
		if result.Primary == ProjectGeneric {
			result.Primary = ProjectGo
		}
	}

	// Python detection
	pythonIndicators := []string{"pyproject.toml", "setup.py", "requirements.txt", "uv.lock", "Pipfile"}
	for _, indicator := range pythonIndicators {
		if fileExists(filepath.Join(projectDir, indicator)) {
			result.Types = append(result.Types, ProjectPython)
			result.Indicators = append(result.Indicators, indicator)
			conventionsMap["python.md"] = true
			if result.Primary == ProjectGeneric {
				result.Primary = ProjectPython
			}
			break // Only append once for Python
		}
	}

	// R detection (with Shiny/Golem variants)
	rIndicators := []string{"DESCRIPTION", "NAMESPACE", "renv.lock"}
	for _, indicator := range rIndicators {
		if fileExists(filepath.Join(projectDir, indicator)) {
			// Check for Golem (superset of Shiny)
			if isGolemProject(projectDir) {
				result.Types = append(result.Types, ProjectRGolem)
				result.Indicators = append(result.Indicators, indicator, "inst/golem-config.yml or golem dependency")
				conventionsMap["R.md"] = true
				conventionsMap["R-shiny.md"] = true
				conventionsMap["R-golem.md"] = true
				if result.Primary == ProjectGeneric {
					result.Primary = ProjectRGolem
				}
				break
			}

			// Check for Shiny
			if isShinyProject(projectDir) {
				result.Types = append(result.Types, ProjectRShiny)
				result.Indicators = append(result.Indicators, indicator, "shiny dependency or app.R/ui.R")
				conventionsMap["R.md"] = true
				conventionsMap["R-shiny.md"] = true
				if result.Primary == ProjectGeneric {
					result.Primary = ProjectRShiny
				}
				break
			}

			// Plain R
			result.Types = append(result.Types, ProjectR)
			result.Indicators = append(result.Indicators, indicator)
			conventionsMap["R.md"] = true
			if result.Primary == ProjectGeneric {
				result.Primary = ProjectR
			}
			break
		}
	}

	// Also check for standalone R files without DESCRIPTION
	if len(result.Types) == 0 || !containsRType(result.Types) {
		if hasRFiles(projectDir) {
			// Check for Shiny indicators even without DESCRIPTION
			if fileExists(filepath.Join(projectDir, "app.R")) || fileExists(filepath.Join(projectDir, "ui.R")) {
				result.Types = append(result.Types, ProjectRShiny)
				result.Indicators = append(result.Indicators, "app.R or ui.R (standalone Shiny)")
				conventionsMap["R.md"] = true
				conventionsMap["R-shiny.md"] = true
				if result.Primary == ProjectGeneric {
					result.Primary = ProjectRShiny
				}
			}
		}
	}

	// Rust detection
	if fileExists(filepath.Join(projectDir, "Cargo.toml")) {
		result.Types = append(result.Types, ProjectRust)
		result.Indicators = append(result.Indicators, "Cargo.toml")
		conventionsMap["rust.md"] = true
		if result.Primary == ProjectGeneric {
			result.Primary = ProjectRust
		}
	}

	// === FRONTEND LANGUAGES (lower priority - detected but don't take Primary over backend) ===

	// TypeScript detection
	if fileExists(filepath.Join(projectDir, "tsconfig.json")) {
		result.Types = append(result.Types, ProjectTypeScript)
		result.Indicators = append(result.Indicators, "tsconfig.json")
		conventionsMap["typescript.md"] = true
		if result.Primary == ProjectGeneric {
			result.Primary = ProjectTypeScript
		}

		// Check for React dependency in TypeScript projects
		if hasReactDependency(projectDir) {
			result.Types = append(result.Types, ProjectReact)
			result.Indicators = append(result.Indicators, "package.json (react dependency)")
			conventionsMap["react.md"] = true
		}
	}

	// JavaScript detection (after TypeScript to avoid double-detection)
	if !containsProjectType(result.Types, ProjectTypeScript) && fileExists(filepath.Join(projectDir, "package.json")) {
		result.Types = append(result.Types, ProjectJavaScript)
		result.Indicators = append(result.Indicators, "package.json")
		conventionsMap["javascript.md"] = true
		if result.Primary == ProjectGeneric {
			result.Primary = ProjectJavaScript
		}

		// Check for React in JavaScript projects too
		if hasReactDependency(projectDir) {
			result.Types = append(result.Types, ProjectReact)
			result.Indicators = append(result.Indicators, "package.json (react dependency)")
			conventionsMap["react.md"] = true
		}
	}

	// Convert conventions map to slice
	for convention := range conventionsMap {
		result.Conventions = append(result.Conventions, convention)
	}

	// Set Type for backward compatibility
	result.Type = result.Primary

	return result
}

// hasReactDependency checks if package.json contains React in dependencies or devDependencies
func hasReactDependency(projectDir string) bool {
	packageJSONPath := filepath.Join(projectDir, "package.json")
	content, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return false
	}

	// Simple string search - avoid full JSON parsing for performance
	contentStr := string(content)
	return strings.Contains(contentStr, `"react"`)
}

// containsProjectType checks if a ProjectType slice contains a specific type
func containsProjectType(types []ProjectType, target ProjectType) bool {
	for _, t := range types {
		if t == target {
			return true
		}
	}
	return false
}

// containsRType checks if any R variant is in the types slice
func containsRType(types []ProjectType) bool {
	for _, t := range types {
		if t == ProjectR || t == ProjectRShiny || t == ProjectRGolem {
			return true
		}
	}
	return false
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
	if result.Primary == ProjectGeneric {
		return "PROJECT TYPE: Generic (no language-specific conventions)"
	}

	// Join multiple types with " + " for polyglot projects
	var typeStrings []string
	for _, t := range result.Types {
		typeStrings = append(typeStrings, string(t))
	}
	projectTypes := strings.Join(typeStrings, " + ")

	conventions := strings.Join(result.Conventions, ", ")
	indicators := strings.Join(result.Indicators, ", ")

	return fmt.Sprintf("PROJECT TYPE: %s\n  Detected via: %s\n  Conventions: %s",
		projectTypes,
		indicators,
		conventions,
	)
}
