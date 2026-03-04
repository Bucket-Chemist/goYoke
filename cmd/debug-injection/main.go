// debug-injection: Dumps BuildFullAgentContext() output for a given agent
// Usage: go run ./cmd/debug-injection/ <agent-id> <output-file>
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <agent-id> <output-file>\n", os.Args[0])
		os.Exit(1)
	}

	agentID := os.Args[1]
	outputFile := os.Args[2]

	fmt.Fprintf(os.Stderr, "[debug] Agent: %s\n", agentID)
	fmt.Fprintf(os.Stderr, "[debug] Output: %s\n", outputFile)

	// Load agent config from agents-index.json
	configDir, err := routing.GetClaudeConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting config dir: %v\n", err)
		os.Exit(1)
	}

	indexPath := filepath.Join(configDir, "agents", "agents-index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading agents-index.json: %v\n", err)
		os.Exit(1)
	}

	var index struct {
		Agents []struct {
			ID                  string                       `json:"id"`
			Name                string                       `json:"name"`
			ContextRequirements *routing.ContextRequirements `json:"context_requirements"`
		} `json:"agents"`
	}
	if err := json.Unmarshal(data, &index); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing agents-index.json: %v\n", err)
		os.Exit(1)
	}

	var requirements *routing.ContextRequirements
	var agentName string
	for _, agent := range index.Agents {
		if agent.ID == agentID {
			requirements = agent.ContextRequirements
			agentName = agent.Name
			break
		}
	}

	if agentName == "" {
		fmt.Fprintf(os.Stderr, "Agent %q not found in agents-index.json\n", agentID)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "[debug] Found agent: %s (%s)\n", agentID, agentName)
	fmt.Fprintf(os.Stderr, "[debug] Context requirements: %+v\n", requirements)

	// Simulate a Task prompt
	originalPrompt := fmt.Sprintf("AGENT: %s\n\nTASK: Test injection verification\nEXPECTED OUTPUT: Review document\n", agentID)

	fmt.Fprintf(os.Stderr, "[debug] Original prompt length: %d bytes\n", len(originalPrompt))

	// Call BuildFullAgentContext
	augmented, err := routing.BuildFullAgentContext(agentID, requirements, nil, originalPrompt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building context: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "[debug] Augmented prompt length: %d bytes\n", len(augmented))
	fmt.Fprintf(os.Stderr, "[debug] Injection added: %d bytes\n", len(augmented)-len(originalPrompt))

	if augmented == originalPrompt {
		fmt.Fprintf(os.Stderr, "[debug] WARNING: No injection occurred! Prompt unchanged.\n")
	} else {
		fmt.Fprintf(os.Stderr, "[debug] SUCCESS: Context was injected.\n")
	}

	// Write to file
	if err := os.WriteFile(outputFile, []byte(augmented), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "[debug] Written to %s\n", outputFile)
}
