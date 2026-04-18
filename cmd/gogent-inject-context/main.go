// gogent-inject-context: Reads a prompt from stdin, prepends agent identity +
// conventions via BuildFullAgentContext, and writes the augmented prompt to stdout.
//
// Usage: echo "AGENT: go-tui\n\nTASK: ..." | gogent-inject-context go-tui
//
// This is the workaround for Claude Code's Agent tool not firing PreToolUse hooks.
// The router calls this before every Agent dispatch to ensure subagents receive
// the same context injection that gogent-validate provides for Task() calls.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Bucket-Chemist/goYoke/pkg/routing"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <agent-id>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Reads prompt from stdin, writes augmented prompt to stdout.\n")
		os.Exit(1)
	}

	agentID := os.Args[1]

	// Read original prompt from stdin
	promptBytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-inject-context] Error reading stdin: %v\n", err)
		os.Exit(1)
	}
	originalPrompt := string(promptBytes)

	// Load agent config from agents-index.json
	configDir, err := routing.GetClaudeConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-inject-context] Error getting config dir: %v\n", err)
		// Fall through with original prompt
		fmt.Print(originalPrompt)
		return
	}

	indexPath := filepath.Join(configDir, "agents", "agents-index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-inject-context] Warning: %v\n", err)
		fmt.Print(originalPrompt)
		return
	}

	var index struct {
		Agents []struct {
			ID                  string                       `json:"id"`
			ContextRequirements *routing.ContextRequirements `json:"context_requirements"`
		} `json:"agents"`
	}
	if err := json.Unmarshal(data, &index); err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-inject-context] Warning: %v\n", err)
		fmt.Print(originalPrompt)
		return
	}

	var requirements *routing.ContextRequirements
	for _, agent := range index.Agents {
		if agent.ID == agentID {
			requirements = agent.ContextRequirements
			break
		}
	}

	// Build augmented prompt
	augmented, err := routing.BuildFullAgentContext(agentID, requirements, nil, originalPrompt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-inject-context] Warning: %v\n", err)
		fmt.Print(originalPrompt)
		return
	}

	fmt.Print(augmented)
}
