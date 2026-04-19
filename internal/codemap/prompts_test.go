package codemap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildNarrativePrompt_ContainsRepoAndStats(t *testing.T) {
	graph := &Graph{
		Repo: "myrepo",
		Stats: GraphStats{
			TotalModules:         3,
			TotalSymbols:         42,
			TotalDependencyEdges: 5,
		},
		Layers: GraphLayers{
			ModuleDependencies: ModuleDepLayer{
				Nodes: []ModuleNode{
					{ID: "cmd/app", Category: "command", SymbolCount: 10},
					{ID: "internal/core", Category: "internal", SymbolCount: 20},
				},
				Edges: []ModuleDependencyEdge{
					{From: "cmd/app", To: "internal/core", Type: "imports"},
				},
			},
		},
	}

	enriched := map[string]*EnrichedModule{
		"internal/core": {
			Module:            "internal/core",
			ModuleDescription: "Core business logic.",
			ModuleCategory:    "internal",
			KeyTypes:          []string{"Config", "Client"},
			KeyFunctions:      []string{"NewClient", "Process"},
		},
	}

	prompt := buildNarrativePrompt(graph, enriched)

	assert.Contains(t, prompt, "myrepo")
	assert.Contains(t, prompt, "3")   // module count
	assert.Contains(t, prompt, "42")  // symbol count
	assert.Contains(t, prompt, "cmd/app")
	assert.Contains(t, prompt, "internal/core")
	assert.Contains(t, prompt, "Core business logic.")
	assert.Contains(t, prompt, "Config")
	assert.Contains(t, prompt, "NewClient")
	assert.Contains(t, prompt, "ARCHITECTURE.md")
	// Dependency edge
	assert.Contains(t, prompt, "cmd/app")
}

func TestBuildNarrativePrompt_NoEnrichment(t *testing.T) {
	graph := &Graph{
		Repo: "repo",
		Layers: GraphLayers{
			ModuleDependencies: ModuleDepLayer{
				Nodes: []ModuleNode{
					{ID: "pkg/util", Category: "pkg", SymbolCount: 5},
				},
			},
		},
	}

	prompt := buildNarrativePrompt(graph, map[string]*EnrichedModule{})

	assert.Contains(t, prompt, "repo")
	assert.Contains(t, prompt, "pkg/util")
	assert.Contains(t, prompt, "(no description)")
}

func TestBuildNarrativePrompt_NoEdges(t *testing.T) {
	graph := &Graph{
		Repo: "solo",
		Layers: GraphLayers{
			ModuleDependencies: ModuleDepLayer{
				Nodes: []ModuleNode{{ID: "cmd/x", Category: "command"}},
			},
		},
	}

	prompt := buildNarrativePrompt(graph, nil)
	// Should not contain dependency section header when no edges
	assert.NotContains(t, prompt, "## Dependency Edges")
}
