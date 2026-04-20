package utils

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Bucket-Chemist/goYoke/pkg/routing"
	"gopkg.in/yaml.v3"
)

// RunValidateSchemas implements the goyoke-validate-schemas utility.
func RunValidateSchemas(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer) error {
	index, err := routing.LoadAgentIndex()
	if err != nil {
		return fmt.Errorf("validate-schemas: error loading agents-index.json: %w", err)
	}

	agentsDir, err := vsGetAgentsDir()
	if err != nil {
		return fmt.Errorf("validate-schemas: %w", err)
	}

	fmt.Fprintf(stdout, "Validating %d agents...\n", len(index.Agents))

	var results []vsValidationResult
	consistentCount := 0
	inconsistentCount := 0

	for _, agent := range index.Agents {
		agentResults := vsValidateAgent(&agent, agentsDir)

		allValid := true
		for _, r := range agentResults {
			if !r.IsValid {
				allValid = false
				break
			}
		}

		if allValid {
			fmt.Fprintf(stdout, "✓ %s: consistent\n", agent.ID)
			consistentCount++
		} else {
			fmt.Fprintf(stdout, "✗ %s: inconsistencies found\n", agent.ID)
			inconsistentCount++
			results = append(results, agentResults...)
		}
	}

	if len(results) > 0 {
		fmt.Fprintln(stdout, "\nInconsistencies:")
		for _, r := range results {
			if !r.IsValid {
				fmt.Fprintf(stdout, "  %s.%s: mismatch\n", r.AgentID, r.Field)
				fmt.Fprintf(stdout, "    index: %q\n", r.IndexValue)
				fmt.Fprintf(stdout, "    frontmatter: %q\n", r.FrontValue)
				fmt.Fprintf(stdout, "    → %s:%d\n", r.FilePath, r.LineNumber)
			}
		}
	}

	fmt.Fprintf(stdout, "\nSummary: %d consistent, %d inconsistent\n", consistentCount, inconsistentCount)

	if inconsistentCount > 0 {
		return fmt.Errorf("validate-schemas: %d agent(s) have inconsistencies", inconsistentCount)
	}
	return nil
}

type vsFrontmatterData struct {
	ID          string      `yaml:"id"`
	Name        string      `yaml:"name"`
	Description string      `yaml:"description"`
	Model       string      `yaml:"model"`
	Tools       []string    `yaml:"tools"`
	Thinking    any `yaml:"thinking"`
	Delegation  struct {
		CanSpawn []string `yaml:"can_spawn"`
	} `yaml:"delegation"`
	SpawnedBy []string `yaml:"spawned_by"`
}

type vsValidationResult struct {
	AgentID    string
	Field      string
	IndexValue string
	FrontValue string
	FilePath   string
	LineNumber int
	IsValid    bool
}

func vsValidateAgent(agent *routing.Agent, agentsDir string) []vsValidationResult {
	var results []vsValidationResult
	mdPath := filepath.Join(agentsDir, agent.ID, agent.ID+".md")

	frontmatter, lineMap, err := vsExtractFrontmatter(mdPath)
	if err != nil {
		return results
	}

	check := func(field, indexVal, frontVal string) {
		if frontVal != "" && frontVal != indexVal {
			results = append(results, vsValidationResult{
				AgentID: agent.ID, Field: field,
				IndexValue: indexVal, FrontValue: frontVal,
				FilePath: mdPath, LineNumber: lineMap[field], IsValid: false,
			})
		}
	}

	check("id", agent.ID, frontmatter.ID)
	check("name", agent.Name, frontmatter.Name)

	if frontmatter.Model != "" && vsNormalizeModel(frontmatter.Model) != agent.Model {
		results = append(results, vsValidationResult{
			AgentID: agent.ID, Field: "model",
			IndexValue: agent.Model, FrontValue: frontmatter.Model,
			FilePath: mdPath, LineNumber: lineMap["model"], IsValid: false,
		})
	}

	frontThinking := vsExtractThinkingValue(frontmatter.Thinking)
	if frontThinking != nil && *frontThinking != agent.Thinking {
		results = append(results, vsValidationResult{
			AgentID: agent.ID, Field: "thinking",
			IndexValue: fmt.Sprintf("%v", agent.Thinking),
			FrontValue: fmt.Sprintf("%v", *frontThinking),
			FilePath: mdPath, LineNumber: lineMap["thinking"], IsValid: false,
		})
	}

	if len(frontmatter.Tools) > 0 && !vsStringSlicesEqual(frontmatter.Tools, agent.Tools) {
		results = append(results, vsValidationResult{
			AgentID: agent.ID, Field: "tools",
			IndexValue: strings.Join(agent.Tools, ", "),
			FrontValue: strings.Join(frontmatter.Tools, ", "),
			FilePath: mdPath, LineNumber: lineMap["tools"], IsValid: false,
		})
	}

	if len(frontmatter.Delegation.CanSpawn) > 0 && len(agent.CanSpawn) > 0 {
		if !vsStringSlicesEqual(frontmatter.Delegation.CanSpawn, agent.CanSpawn) {
			results = append(results, vsValidationResult{
				AgentID: agent.ID, Field: "can_spawn",
				IndexValue: strings.Join(agent.CanSpawn, ", "),
				FrontValue: strings.Join(frontmatter.Delegation.CanSpawn, ", "),
				FilePath: mdPath, LineNumber: lineMap["delegation.can_spawn"], IsValid: false,
			})
		}
	}

	if len(frontmatter.SpawnedBy) > 0 && len(agent.SpawnedBy) > 0 {
		if !vsStringSlicesEqual(frontmatter.SpawnedBy, agent.SpawnedBy) {
			results = append(results, vsValidationResult{
				AgentID: agent.ID, Field: "spawned_by",
				IndexValue: strings.Join(agent.SpawnedBy, ", "),
				FrontValue: strings.Join(frontmatter.SpawnedBy, ", "),
				FilePath: mdPath, LineNumber: lineMap["spawned_by"], IsValid: false,
			})
		}
	}

	return results
}

func vsExtractFrontmatter(path string) (*vsFrontmatterData, map[string]int, error) {
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

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if line == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			}
			break
		}
		if inFrontmatter {
			frontmatterLines = append(frontmatterLines, line)
			trimmed := strings.TrimSpace(line)
			if strings.Contains(trimmed, ":") {
				parts := strings.SplitN(trimmed, ":", 2)
				lineMap[strings.TrimSpace(parts[0])] = lineNum
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("error reading file: %w", err)
	}

	var data vsFrontmatterData
	yamlContent := []byte(strings.Join(frontmatterLines, "\n"))
	if err := yaml.Unmarshal(yamlContent, &data); err != nil {
		return nil, nil, fmt.Errorf("error parsing frontmatter: %w", err)
	}

	if bytes.Contains(yamlContent, []byte("can_spawn:")) {
		lineMap["delegation.can_spawn"] = lineMap["can_spawn"]
	}

	return &data, lineMap, nil
}

func vsExtractThinkingValue(thinking any) *bool {
	if thinking == nil {
		return nil
	}
	boolPtr := func(v bool) *bool { return &v }
	switch v := thinking.(type) {
	case bool:
		return &v
	case map[string]any:
		if enabled, ok := v["enabled"].(bool); ok {
			return boolPtr(enabled)
		}
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

func vsStringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aMap := make(map[string]int)
	bMap := make(map[string]int)
	for _, s := range a {
		aMap[s]++
	}
	for _, s := range b {
		bMap[s]++
	}
	for k, v := range aMap {
		if bMap[k] != v {
			return false
		}
	}
	return true
}

func vsNormalizeModel(model string) string {
	aliases := map[string]string{
		"claude-opus-4-6":           "opus",
		"claude-sonnet-4-6":         "sonnet",
		"claude-haiku-4-5-20251001": "haiku",
	}
	if tier, ok := aliases[model]; ok {
		return tier
	}
	return model
}

func vsGetAgentsDir() (string, error) {
	if projectDir := os.Getenv("GOYOKE_PROJECT_DIR"); projectDir != "" {
		return filepath.Join(projectDir, ".claude", "agents"), nil
	}

	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home := os.Getenv("HOME")
		if home == "" {
			return "", fmt.Errorf("HOME environment variable not set")
		}
		configHome = filepath.Join(home, ".config")
	}

	return filepath.Join(configHome, "..", ".claude", "agents"), nil
}
