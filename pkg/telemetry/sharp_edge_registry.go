package telemetry

import (
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

var (
	sharpEdgeIDs     map[string]bool
	sharpEdgeIDsOnce sync.Once
)

// SharpEdgesYAML represents the structure of sharp-edges.yaml
type SharpEdgesYAML struct {
	SharpEdges []struct {
		ID string `yaml:"id"`
	} `yaml:"sharp_edges"`
}

// LoadSharpEdgeIDs scans all agent sharp-edges.yaml files and builds registry
func LoadSharpEdgeIDs() error {
	var loadErr error
	sharpEdgeIDsOnce.Do(func() {
		sharpEdgeIDs = make(map[string]bool)

		// Find .claude/agents directory
		agentsDir := filepath.Join(os.Getenv("HOME"), ".claude", "agents")
		if projectDir := os.Getenv("GOYOKE_PROJECT_DIR"); projectDir != "" {
			agentsDir = filepath.Join(projectDir, ".claude", "agents")
		}

		// Walk agent directories
		entries, err := os.ReadDir(agentsDir)
		if err != nil {
			loadErr = err
			return
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			yamlPath := filepath.Join(agentsDir, entry.Name(), "sharp-edges.yaml")
			data, err := os.ReadFile(yamlPath)
			if err != nil {
				continue // Agent may not have sharp edges
			}

			var se SharpEdgesYAML
			if err := yaml.Unmarshal(data, &se); err != nil {
				continue
			}

			for _, edge := range se.SharpEdges {
				if edge.ID != "" {
					sharpEdgeIDs[edge.ID] = true
				}
			}
		}
	})
	return loadErr
}

// IsValidSharpEdgeID checks if an ID exists in the registry
func IsValidSharpEdgeID(id string) bool {
	if sharpEdgeIDs == nil {
		LoadSharpEdgeIDs()
	}
	return sharpEdgeIDs[id]
}

// GetAllSharpEdgeIDs returns all registered IDs (for documentation)
func GetAllSharpEdgeIDs() []string {
	if sharpEdgeIDs == nil {
		LoadSharpEdgeIDs()
	}
	ids := make([]string, 0, len(sharpEdgeIDs))
	for id := range sharpEdgeIDs {
		ids = append(ids, id)
	}
	return ids
}

// ResetRegistry clears the registry (for testing)
func ResetRegistry() {
	sharpEdgeIDs = nil
	sharpEdgeIDsOnce = sync.Once{}
}
