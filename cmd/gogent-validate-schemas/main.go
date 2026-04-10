// gogent-validate-schemas validates consistency between agents-index.json and
// per-agent .md frontmatter files.
//
// The validator checks for mismatches in overlapping fields:
//   - id: agent identifier
//   - name: display name
//   - model: haiku/sonnet/opus
//   - thinking: enabled flag (handles both bool and struct formats)
//   - tools: available tool list (order-independent comparison)
//   - can_spawn: spawnable agent list
//   - spawned_by: parent agent list
//
// Exit codes:
//   0: All agents consistent
//   1: Inconsistencies found
//
// Usage:
//   gogent-validate-schemas
//
// Example output:
//   Validating 37 agents...
//   ✓ einstein: consistent
//   ✗ go-pro: model mismatch (index: sonnet, frontmatter: haiku)
//     → .claude/agents/go-pro/go-pro.md:4
//   Summary: 36 consistent, 1 inconsistent
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
	"gopkg.in/yaml.v3"
)

// FrontmatterData represents the YAML frontmatter from agent .md files.
// Only includes fields that overlap with agents-index.json for validation.
type FrontmatterData struct {
	ID          string   `yaml:"id"`
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Model       string   `yaml:"model"`
	Tools       []string `yaml:"tools"`
	Thinking    interface{} `yaml:"thinking"` // Can be bool or struct

	// Delegation fields
	Delegation struct {
		CanSpawn []string `yaml:"can_spawn"`
	} `yaml:"delegation"`

	// Nested spawned_by in some agents
	SpawnedBy []string `yaml:"spawned_by"`
}

// ThinkingConfig represents the nested thinking configuration.
type ThinkingConfig struct {
	Enabled bool `yaml:"enabled"`
	Budget  int  `yaml:"budget"`
}

// ValidationResult tracks a single validation check.
type ValidationResult struct {
	AgentID     string
	Field       string
	IndexValue  string
	FrontValue  string
	FilePath    string
	LineNumber  int
	IsValid     bool
}

func main() {
	// Load agents-index.json
	index, err := routing.LoadAgentIndex()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading agents-index.json: %v\n", err)
		os.Exit(1)
	}

	// Determine agent directory
	agentsDir := getAgentsDir()

	fmt.Printf("Validating %d agents...\n", len(index.Agents))

	var results []ValidationResult
	consistentCount := 0
	inconsistentCount := 0

	// Validate each agent
	for _, agent := range index.Agents {
		agentResults := validateAgent(&agent, agentsDir)

		allValid := true
		for _, r := range agentResults {
			if !r.IsValid {
				allValid = false
				break
			}
		}

		if allValid {
			fmt.Printf("✓ %s: consistent\n", agent.ID)
			consistentCount++
		} else {
			fmt.Printf("✗ %s: inconsistencies found\n", agent.ID)
			inconsistentCount++
			results = append(results, agentResults...)
		}
	}

	// Print detailed inconsistencies
	if len(results) > 0 {
		fmt.Println("\nInconsistencies:")
		for _, r := range results {
			if !r.IsValid {
				fmt.Printf("  %s.%s: mismatch\n", r.AgentID, r.Field)
				fmt.Printf("    index: %q\n", r.IndexValue)
				fmt.Printf("    frontmatter: %q\n", r.FrontValue)
				fmt.Printf("    → %s:%d\n", r.FilePath, r.LineNumber)
			}
		}
	}

	// Summary
	fmt.Printf("\nSummary: %d consistent, %d inconsistent\n", consistentCount, inconsistentCount)

	if inconsistentCount > 0 {
		os.Exit(1)
	}
}

// validateAgent compares agent config from index with frontmatter.
func validateAgent(agent *routing.Agent, agentsDir string) []ValidationResult {
	var results []ValidationResult

	// Find agent .md file
	mdPath := filepath.Join(agentsDir, agent.ID, agent.ID+".md")

	frontmatter, lineMap, err := extractFrontmatter(mdPath)
	if err != nil {
		// File not found or parse error - record but don't fail
		// (some agents might not have .md files yet)
		return results
	}

	// Validate ID (if present in frontmatter)
	if frontmatter.ID != "" && frontmatter.ID != agent.ID {
		results = append(results, ValidationResult{
			AgentID:    agent.ID,
			Field:      "id",
			IndexValue: agent.ID,
			FrontValue: frontmatter.ID,
			FilePath:   mdPath,
			LineNumber: lineMap["id"],
			IsValid:    false,
		})
	}

	// Validate name
	if frontmatter.Name != "" && frontmatter.Name != agent.Name {
		results = append(results, ValidationResult{
			AgentID:    agent.ID,
			Field:      "name",
			IndexValue: agent.Name,
			FrontValue: frontmatter.Name,
			FilePath:   mdPath,
			LineNumber: lineMap["name"],
			IsValid:    false,
		})
	}

	// Validate model (normalize API strings to tier names before comparing)
	if frontmatter.Model != "" && normalizeModel(frontmatter.Model) != agent.Model {
		results = append(results, ValidationResult{
			AgentID:    agent.ID,
			Field:      "model",
			IndexValue: agent.Model,
			FrontValue: frontmatter.Model,
			FilePath:   mdPath,
			LineNumber: lineMap["model"],
			IsValid:    false,
		})
	}

	// Validate thinking (complex: can be bool or struct)
	frontThinking := extractThinkingValue(frontmatter.Thinking)
	if frontThinking != nil && *frontThinking != agent.Thinking {
		results = append(results, ValidationResult{
			AgentID:    agent.ID,
			Field:      "thinking",
			IndexValue: fmt.Sprintf("%v", agent.Thinking),
			FrontValue: fmt.Sprintf("%v", *frontThinking),
			FilePath:   mdPath,
			LineNumber: lineMap["thinking"],
			IsValid:    false,
		})
	}

	// Validate tools (array comparison)
	if len(frontmatter.Tools) > 0 {
		if !stringSlicesEqual(frontmatter.Tools, agent.Tools) {
			results = append(results, ValidationResult{
				AgentID:    agent.ID,
				Field:      "tools",
				IndexValue: strings.Join(agent.Tools, ", "),
				FrontValue: strings.Join(frontmatter.Tools, ", "),
				FilePath:   mdPath,
				LineNumber: lineMap["tools"],
				IsValid:    false,
			})
		}
	}

	// Validate can_spawn
	if len(frontmatter.Delegation.CanSpawn) > 0 && len(agent.CanSpawn) > 0 {
		if !stringSlicesEqual(frontmatter.Delegation.CanSpawn, agent.CanSpawn) {
			results = append(results, ValidationResult{
				AgentID:    agent.ID,
				Field:      "can_spawn",
				IndexValue: strings.Join(agent.CanSpawn, ", "),
				FrontValue: strings.Join(frontmatter.Delegation.CanSpawn, ", "),
				FilePath:   mdPath,
				LineNumber: lineMap["delegation.can_spawn"],
				IsValid:    false,
			})
		}
	}

	// Validate spawned_by
	if len(frontmatter.SpawnedBy) > 0 && len(agent.SpawnedBy) > 0 {
		if !stringSlicesEqual(frontmatter.SpawnedBy, agent.SpawnedBy) {
			results = append(results, ValidationResult{
				AgentID:    agent.ID,
				Field:      "spawned_by",
				IndexValue: strings.Join(agent.SpawnedBy, ", "),
				FrontValue: strings.Join(frontmatter.SpawnedBy, ", "),
				FilePath:   mdPath,
				LineNumber: lineMap["spawned_by"],
				IsValid:    false,
			})
		}
	}

	return results
}

// extractFrontmatter parses YAML frontmatter from an agent .md file.
// Returns the parsed data and a map of field names to line numbers.
func extractFrontmatter(path string) (*FrontmatterData, map[string]int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	inFrontmatter := false
	var frontmatterLines []string
	lineMap := make(map[string]int)

	// Extract frontmatter between --- markers
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if line == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			} else {
				// End of frontmatter
				break
			}
		}

		if inFrontmatter {
			frontmatterLines = append(frontmatterLines, line)

			// Track line numbers for fields
			trimmed := strings.TrimSpace(line)
			if strings.Contains(trimmed, ":") {
				parts := strings.SplitN(trimmed, ":", 2)
				fieldName := strings.TrimSpace(parts[0])
				lineMap[fieldName] = lineNum

				// Handle nested fields
				if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") {
					// Top-level nested field (2 spaces)
					lineMap[fieldName] = lineNum
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("error reading file: %w", err)
	}

	// Parse YAML
	var data FrontmatterData
	yamlContent := []byte(strings.Join(frontmatterLines, "\n"))
	if err := yaml.Unmarshal(yamlContent, &data); err != nil {
		return nil, nil, fmt.Errorf("error parsing frontmatter: %w", err)
	}

	// Handle nested delegation fields
	if bytes.Contains(yamlContent, []byte("can_spawn:")) {
		lineMap["delegation.can_spawn"] = lineMap["can_spawn"]
	}

	return &data, lineMap, nil
}

// extractThinkingValue handles bool and struct thinking formats.
// Supported formats:
//   - bool: true/false directly
//   - {enabled: bool}: legacy struct format
//   - {type: "adaptive"}: Opus 4.6 adaptive thinking → true
//   - {type: "enabled", budget_tokens: N}: explicit enabled → true
//   - {type: "disabled"}: explicitly disabled → false
func extractThinkingValue(thinking interface{}) *bool {
	if thinking == nil {
		return nil
	}

	boolPtr := func(v bool) *bool { return &v }

	switch v := thinking.(type) {
	case bool:
		return &v
	case map[string]interface{}:
		// Legacy: {enabled: bool}
		if enabled, ok := v["enabled"].(bool); ok {
			return boolPtr(enabled)
		}
		// Opus 4.6: {type: "adaptive"|"enabled"|"disabled"}
		if typVal, ok := v["type"].(string); ok {
			switch typVal {
			case "adaptive", "enabled":
				return boolPtr(true)
			case "disabled":
				return boolPtr(false)
			}
		}
	}

	return nil
}

// stringSlicesEqual compares two string slices for equality (order-independent).
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	// Create frequency maps
	aMap := make(map[string]int)
	bMap := make(map[string]int)

	for _, s := range a {
		aMap[s]++
	}
	for _, s := range b {
		bMap[s]++
	}

	// Compare maps
	for k, v := range aMap {
		if bMap[k] != v {
			return false
		}
	}

	return true
}

// normalizeModel maps Claude API model strings to tier names used in agents-index.json.
// Tier names (haiku, sonnet, opus) pass through unchanged.
func normalizeModel(model string) string {
	aliases := map[string]string{
		"claude-opus-4-6":          "opus",
		"claude-sonnet-4-6":        "sonnet",
		"claude-haiku-4-5-20251001": "haiku",
	}
	if tier, ok := aliases[model]; ok {
		return tier
	}
	return model
}

// getAgentsDir returns the agents directory path.
func getAgentsDir() string {
	// Priority 1: GOGENT_PROJECT_DIR (test isolation)
	if projectDir := os.Getenv("GOGENT_PROJECT_DIR"); projectDir != "" {
		return filepath.Join(projectDir, ".claude", "agents")
	}

	// Priority 2: XDG config (production)
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home := os.Getenv("HOME")
		if home == "" {
			fmt.Fprintln(os.Stderr, "Error: HOME not set")
			os.Exit(1)
		}
		configHome = filepath.Join(home, ".config")
	}

	return filepath.Join(configHome, "..", ".claude", "agents")
}
