package codemap

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeExtraction(module, lang string, imports []string, symbols []Symbol) *ModuleExtraction {
	return &ModuleExtraction{
		Module:           module,
		Language:         lang,
		Files:            []FileExtract{{Path: module + "/main.go", LineCount: 10, Symbols: symbols}},
		Imports:          ImportGraph{Internal: imports},
		ExtractedAt:      time.Now().UTC().Format(time.RFC3339),
		ExtractorVersion: "test",
	}
}

func TestAggregateGraph_BasicStructure(t *testing.T) {
	extractions := []*ModuleExtraction{
		makeExtraction("cmd/goyoke-validate", "go", []string{"github.com/example/goYoke/internal/routing"}, nil),
		makeExtraction("internal/routing", "go", nil, nil),
	}

	g := AggregateGraph(extractions, "goYoke", "abc123")

	require.NotNil(t, g)
	assert.Equal(t, "1.0", g.Version)
	assert.Equal(t, "goYoke", g.Repo)
	assert.Equal(t, "abc123", g.GitCommit)
	assert.NotEmpty(t, g.GeneratedAt)
}

func TestAggregateGraph_ModuleNodes(t *testing.T) {
	extractions := []*ModuleExtraction{
		makeExtraction("cmd/foo", "go", nil, nil),
		makeExtraction("internal/bar", "go", nil, nil),
		makeExtraction("pkg/baz", "go", nil, nil),
	}

	g := AggregateGraph(extractions, "repo", "")

	assert.Len(t, g.Layers.ModuleDependencies.Nodes, 3)
	assert.Equal(t, 3, g.Stats.TotalModules)
}

func TestCategoryDetection(t *testing.T) {
	tests := []struct {
		module   string
		expected string
	}{
		{"cmd/goyoke-validate", "command"},
		{"cmd/foo/bar", "command"},
		{"internal/routing", "internal"},
		{"internal/codemap", "internal"},
		{"pkg/routing", "pkg"},
		{"test/integration", "test"},
		{"", "internal"},
		{"scripts", "internal"},
	}
	for _, tc := range tests {
		t.Run(tc.module, func(t *testing.T) {
			assert.Equal(t, tc.expected, moduleCategory(tc.module))
		})
	}
}

func TestAggregateGraph_DependencyEdges(t *testing.T) {
	extractions := []*ModuleExtraction{
		makeExtraction("cmd/app", "go", []string{"github.com/x/repo/internal/core"}, nil),
		makeExtraction("internal/core", "go", nil, nil),
	}

	g := AggregateGraph(extractions, "repo", "")

	require.Len(t, g.Layers.ModuleDependencies.Edges, 1)
	edge := g.Layers.ModuleDependencies.Edges[0]
	assert.Equal(t, "cmd/app", edge.From)
	assert.Equal(t, "internal/core", edge.To)
	assert.Equal(t, "imports", edge.Type)
	assert.Equal(t, 1, g.Stats.TotalDependencyEdges)
}

func TestAggregateGraph_NoDuplicateEdges(t *testing.T) {
	// Two files in same module both importing the same target.
	e := &ModuleExtraction{
		Module:   "cmd/app",
		Language: "go",
		Files: []FileExtract{
			{Path: "cmd/app/a.go"},
			{Path: "cmd/app/b.go"},
		},
		Imports:          ImportGraph{Internal: []string{"repo/internal/core", "repo/internal/core"}},
		ExtractedAt:      time.Now().UTC().Format(time.RFC3339),
		ExtractorVersion: "test",
	}
	target := makeExtraction("internal/core", "go", nil, nil)

	g := AggregateGraph([]*ModuleExtraction{e, target}, "repo", "")
	assert.Len(t, g.Layers.ModuleDependencies.Edges, 1, "duplicate import paths must produce only one edge")
}

func TestAggregateGraph_TypeNodes(t *testing.T) {
	symbols := []Symbol{
		{Name: "Config", Kind: "type", Exported: true},
		{Name: "Handler", Kind: "interface", Exported: true},
		{Name: "processItem", Kind: "function", Exported: false},
	}
	extractions := []*ModuleExtraction{
		makeExtraction("internal/server", "go", nil, symbols),
	}

	g := AggregateGraph(extractions, "repo", "")

	nodes := g.Layers.TypeRelationships.Nodes
	require.Len(t, nodes, 2, "only type and interface symbols become type nodes")

	ids := make([]string, len(nodes))
	for i, n := range nodes {
		ids[i] = n.ID
	}
	assert.Contains(t, ids, "internal/server.Config")
	assert.Contains(t, ids, "internal/server.Handler")
}

func TestAggregateGraph_Stats(t *testing.T) {
	symbols := []Symbol{
		{Name: "Foo", Kind: "function"},
		{Name: "Bar", Kind: "method"},
		{Name: "MyType", Kind: "type"},
	}
	extractions := []*ModuleExtraction{
		makeExtraction("cmd/app", "go", []string{"repo/internal/lib"}, symbols),
		makeExtraction("internal/lib", "go", nil, nil),
	}

	g := AggregateGraph(extractions, "repo", "")

	assert.Equal(t, 2, g.Stats.TotalModules)
	assert.Equal(t, 3, g.Stats.TotalSymbols)
	assert.Equal(t, 1, g.Stats.TotalDependencyEdges)
	assert.Equal(t, 0, g.Stats.TotalCallEdges)
	assert.Equal(t, 0, g.Stats.TotalTypeEdges)
}

func TestAggregateGraph_CallGraphEmpty(t *testing.T) {
	g := AggregateGraph([]*ModuleExtraction{makeExtraction("cmd/x", "go", nil, nil)}, "r", "")
	assert.Empty(t, g.Layers.CallGraph.Nodes)
	assert.Empty(t, g.Layers.CallGraph.Edges)
	assert.Equal(t, 0, g.Stats.TotalCallEdges)
}

func TestWriteGraph(t *testing.T) {
	dir := t.TempDir()
	g := AggregateGraph([]*ModuleExtraction{makeExtraction("cmd/app", "go", nil, nil)}, "repo", "abc")

	path := dir + "/graph.json"
	require.NoError(t, WriteGraph(path, g))

	_, err := os.Stat(path)
	require.NoError(t, err, "graph.json must exist")
}

func TestBuildCallGraphLayer_Empty(t *testing.T) {
	nodes, edges := buildCallGraphLayer([]*ModuleExtraction{})
	assert.Empty(t, nodes)
	assert.Empty(t, edges)
}

func TestBuildCallGraphLayer_FunctionWithCalls(t *testing.T) {
	extractions := []*ModuleExtraction{
		{
			Module:   "cmd/app",
			Language: "go",
			Files: []FileExtract{
				{
					Path: "cmd/app/main.go",
					Symbols: []Symbol{
						{
							Name:  "run",
							Kind:  "function",
							Calls: []string{"internal/core.Process"},
						},
					},
				},
			},
		},
		{
			Module:   "internal/core",
			Language: "go",
			Files: []FileExtract{
				{
					Path: "internal/core/core.go",
					Symbols: []Symbol{
						{
							Name:     "Process",
							Kind:     "function",
							CalledBy: []string{"cmd/app.run"},
						},
					},
				},
			},
		},
	}

	nodes, edges := buildCallGraphLayer(extractions)

	assert.NotEmpty(t, nodes)
	assert.NotEmpty(t, edges)

	// Verify edge properties
	require.Len(t, edges, 1)
	assert.Equal(t, "cmd/app.run", edges[0].From)
	assert.Equal(t, "internal/core.Process", edges[0].To)
	assert.True(t, edges[0].CrossModule)
}

func TestBuildCallGraphLayer_SkipsNoCallSymbols(t *testing.T) {
	extractions := []*ModuleExtraction{
		{
			Module:   "pkg/util",
			Language: "go",
			Files: []FileExtract{
				{
					Path: "pkg/util/util.go",
					Symbols: []Symbol{
						{Name: "Helper", Kind: "function", Calls: nil, CalledBy: nil},
					},
				},
			},
		},
	}

	nodes, edges := buildCallGraphLayer(extractions)
	assert.Empty(t, nodes)
	assert.Empty(t, edges)
}

func TestBuildCallGraphLayer_SkipsNonFunctions(t *testing.T) {
	extractions := []*ModuleExtraction{
		{
			Module:   "pkg/types",
			Language: "go",
			Files: []FileExtract{
				{
					Path: "pkg/types/types.go",
					Symbols: []Symbol{
						{Name: "Config", Kind: "type", Calls: []string{"something.Else"}},
					},
				},
			},
		},
	}

	nodes, edges := buildCallGraphLayer(extractions)
	assert.Empty(t, nodes)
	assert.Empty(t, edges)
}

func TestBuildCallGraphLayer_NoDuplicateEdges(t *testing.T) {
	// Same callee referenced twice from same function
	extractions := []*ModuleExtraction{
		{
			Module:   "cmd/x",
			Language: "go",
			Files: []FileExtract{
				{
					Path: "cmd/x/main.go",
					Symbols: []Symbol{
						{
							Name:  "run",
							Kind:  "function",
							Calls: []string{"pkg/a.Foo", "pkg/a.Foo"},
						},
					},
				},
			},
		},
	}

	_, edges := buildCallGraphLayer(extractions)
	assert.Len(t, edges, 1, "duplicate call edges should be deduplicated")
}

func TestResolveModulePath(t *testing.T) {
	moduleSet := map[string]bool{
		"internal/codemap": true,
		"cmd/goyoke-validate": true,
		"pkg/routing": true,
	}

	assert.Equal(t, "internal/codemap",
		resolveModulePath("github.com/x/y/internal/codemap", moduleSet))
	assert.Equal(t, "cmd/goyoke-validate",
		resolveModulePath("github.com/x/y/cmd/goyoke-validate", moduleSet))
	assert.Equal(t, "",
		resolveModulePath("github.com/x/y/unknown/mod", moduleSet))
}
