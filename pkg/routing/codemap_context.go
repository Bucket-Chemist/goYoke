package routing

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bucket-Chemist/goYoke/internal/codemap"
)

// injectableAgents is the allowlist of agent IDs that receive codebase map context injection.
var injectableAgents = map[string]bool{
	"go-pro": true, "go-cli": true, "go-tui": true, "go-api": true, "go-concurrent": true,
	"python-pro": true, "typescript-pro": true, "react-pro": true, "r-pro": true, "rust-pro": true,
	"code-reviewer": true, "architect": true, "tech-docs-writer": true,
	"backend-reviewer": true, "frontend-reviewer": true, "standards-reviewer": true,
}

const maxInjectModules = 3
const mapStalenessDays = 7


// InjectCodebaseMapContext returns a context block for modules relevant to the prompt,
// or empty string when injection should be skipped.
//
// Returns empty string when:
//   - GOYOKE_CODEBASE_MAP_INJECT != "1"
//   - agentID not in the injectable allowlist
//   - graph.json missing or unreadable
//   - no relevant modules identified from the prompt
func InjectCodebaseMapContext(agentID, prompt, projectRoot string) string {
	if os.Getenv("GOYOKE_CODEBASE_MAP_INJECT") != "1" {
		return ""
	}
	if !injectableAgents[agentID] {
		return ""
	}

	graphPath := filepath.Join(projectRoot, ".claude", "codebase-map", "graph.json")
	data, err := os.ReadFile(graphPath)
	if err != nil {
		return ""
	}

	var g codemap.Graph
	if err := json.Unmarshal(data, &g); err != nil {
		return ""
	}

	nodes := g.Layers.ModuleDependencies.Nodes
	edges := g.Layers.ModuleDependencies.Edges

	moduleIDs := make([]string, len(nodes))
	for i, n := range nodes {
		moduleIDs[i] = n.ID
	}

	matched := identifyModulesFromPrompt(prompt, moduleIDs)
	if len(matched) == 0 {
		return ""
	}

	nodeMap := make(map[string]codemap.ModuleNode, len(nodes))
	for _, n := range nodes {
		nodeMap[n.ID] = n
	}

	var matchedNodes []codemap.ModuleNode
	for _, id := range matched {
		if n, ok := nodeMap[id]; ok {
			matchedNodes = append(matchedNodes, n)
		}
	}

	staleNote := mapStalenessWarning(g.GeneratedAt)
	return formatModuleContext(matchedNodes, edges, staleNote)
}

// identifyModulesFromPrompt returns up to maxInjectModules module IDs relevant to the prompt.
//
// Matching is done in three passes (decreasing specificity):
//  1. Exact module ID substring (e.g. "internal/routing" present in prompt)
//  2. File path prefix (e.g. "internal/routing/validator.go" → "internal/routing")
//  3. Directory name fuzzy match (last path segment, ≥4 chars, case-insensitive)
func identifyModulesFromPrompt(prompt string, moduleIDs []string) []string {
	seen := make(map[string]bool)
	var result []string

	add := func(id string) {
		if !seen[id] && len(result) < maxInjectModules {
			seen[id] = true
			result = append(result, id)
		}
	}

	// Pass 1: exact module ID substring
	for _, id := range moduleIDs {
		if strings.Contains(prompt, id) {
			add(id)
		}
	}
	if len(result) >= maxInjectModules {
		return result
	}

	// Pass 2: file path prefix match
	tokens := extractPathTokens(prompt)
	for _, token := range tokens {
		for _, id := range moduleIDs {
			if strings.HasPrefix(token, id+"/") || token == id {
				add(id)
			}
		}
		if len(result) >= maxInjectModules {
			return result
		}
	}

	// Pass 3: directory name fuzzy match (last segment, min 4 chars)
	for _, id := range moduleIDs {
		parts := strings.Split(id, "/")
		lastPart := parts[len(parts)-1]
		if len(lastPart) >= 4 && strings.Contains(strings.ToLower(prompt), strings.ToLower(lastPart)) {
			add(id)
		}
		if len(result) >= maxInjectModules {
			return result
		}
	}

	return result
}

// extractPathTokens pulls path-like tokens (containing "/") from text, stripping .go suffix.
func extractPathTokens(text string) []string {
	var tokens []string
	words := strings.FieldsFunc(text, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '"' || r == '\'' || r == '(' || r == ')'
	})
	for _, w := range words {
		if strings.Contains(w, "/") {
			tokens = append(tokens, strings.TrimSuffix(w, ".go"))
		}
	}
	return tokens
}

// formatModuleContext builds the injection markdown block for the identified modules.
// Cap: maxInjectModules modules.
func formatModuleContext(nodes []codemap.ModuleNode, allEdges []codemap.ModuleDependencyEdge, staleNote string) string {
	if len(nodes) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Module Context (from codebase-map)\n")

	for i, n := range nodes {
		if i > 0 {
			sb.WriteString("\n")
		}
		fmt.Fprintf(&sb, "\n**Module**: %s\n", n.ID)
		fmt.Fprintf(&sb, "**Category**: %s\n", n.Category)

		if n.Description != nil && *n.Description != "" {
			fmt.Fprintf(&sb, "**Description**: %s\n", *n.Description)
		}

		if len(n.KeyTypes) > 0 {
			fmt.Fprintf(&sb, "**Key Types**: %s\n", strings.Join(n.KeyTypes, ", "))
		}

		if len(n.KeyFunctions) > 0 {
			fmt.Fprintf(&sb, "**Key Functions**: %s\n", strings.Join(n.KeyFunctions, ", "))
		}

		var deps, dependents []string
		for _, e := range allEdges {
			switch {
			case e.From == n.ID:
				deps = append(deps, e.To)
			case e.To == n.ID:
				dependents = append(dependents, e.From)
			}
		}
		if len(deps) > 0 {
			fmt.Fprintf(&sb, "**Dependencies**: %s\n", strings.Join(deps, ", "))
		}
		if len(dependents) > 0 {
			fmt.Fprintf(&sb, "**Depended on by**: %s\n", strings.Join(dependents, ", "))
		}

		fmt.Fprintf(&sb, "**Symbols**: %d (%d files)\n", n.SymbolCount, n.FileCount)
	}

	if staleNote != "" {
		fmt.Fprintf(&sb, "\n%s\n", staleNote)
	}

	return sb.String()
}

// mapStalenessWarning returns a human-readable note if the map was generated more than
// mapStalenessDays ago. Returns empty string if the timestamp is missing, unparseable,
// or the map is fresh.
func mapStalenessWarning(generatedAt string) string {
	if generatedAt == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, generatedAt)
	if err != nil {
		return ""
	}
	days := int(time.Since(t).Hours() / 24)
	if days < mapStalenessDays {
		return ""
	}
	return fmt.Sprintf("(Note: Module map is %d days old. Run /codebase-map --incremental to update.)", days)
}

// findProjectRoot locates the project root using git, then go.mod walk, then cwd.
func findProjectRoot() string {
	if out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output(); err == nil {
		if root := strings.TrimSpace(string(out)); root != "" {
			return root
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	for dir := cwd; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		if parent := filepath.Dir(dir); parent == dir {
			break
		}
	}
	return cwd
}
