package codemap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFullPipeline runs the complete pipeline on the goYoke repo itself.
func TestFullPipeline(t *testing.T) {
	dir := t.TempDir()
	extractDir := filepath.Join(dir, "extract")
	graphPath := filepath.Join(dir, "graph.json")
	manifestPath := filepath.Join(dir, "manifest.json")

	// Stage 1: discover files.
	modules, err := DiscoverFiles(repoRoot, DiscoveryOpts{})
	require.NoError(t, err)
	require.NotEmpty(t, modules)

	projectModule, err := ResolveProjectModule(repoRoot)
	require.NoError(t, err)

	// Stage 2: extract all Go modules (integration test stays Go-only).
	var extractions []*ModuleExtraction
	for key, filePaths := range modules {
		if key.Language != "go" {
			continue
		}
		e, err := ExtractModule(key.Path, filePaths, repoRoot, projectModule, false)
		require.NoError(t, err, "extract module %s", key.Path)
		extractions = append(extractions, e)
	}
	require.NotEmpty(t, extractions)

	// Stage 3: write per-module JSON.
	for _, e := range extractions {
		require.NoError(t, WriteModuleExtraction(extractDir, e))
	}

	entries, err := os.ReadDir(extractDir)
	require.NoError(t, err)
	assert.Len(t, entries, len(extractions), "one JSON file per module")

	// Verify each output file is valid JSON with expected fields.
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(extractDir, entry.Name()))
		require.NoError(t, err)
		var me ModuleExtraction
		require.NoError(t, json.Unmarshal(data, &me), "file %s must be valid JSON", entry.Name())
		assert.NotEmpty(t, me.Module)
		assert.NotEmpty(t, me.Language)
		assert.NotEmpty(t, me.ExtractedAt)
		// Pretty-printed: indented output expected.
		assert.Contains(t, string(data), "\n  ", "JSON must be indented")
	}

	// Stage 4: aggregate graph.
	graph := AggregateGraph(extractions, "goYoke", "test-commit")
	require.NoError(t, WriteGraph(graphPath, graph))

	// Verify graph.json is valid.
	graphData, err := os.ReadFile(graphPath)
	require.NoError(t, err)
	var g Graph
	require.NoError(t, json.Unmarshal(graphData, &g), "graph.json must be valid JSON")
	assert.Equal(t, "1.0", g.Version)
	assert.NotEmpty(t, g.Layers.ModuleDependencies.Nodes)
	assert.NotEmpty(t, g.Layers.ModuleDependencies.Edges, "goYoke modules have internal imports")
	assert.Equal(t, len(extractions), g.Stats.TotalModules)

	// Stage 5: build and write manifest.
	m, err := BuildManifest(repoRoot, extractions, "test")
	require.NoError(t, err)
	require.NoError(t, WriteManifest(manifestPath, m))

	manifestData, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	var mf Manifest
	require.NoError(t, json.Unmarshal(manifestData, &mf), "manifest.json must be valid JSON")

	assert.Equal(t, "1.0", mf.Version)
	assert.NotEmpty(t, mf.RepoName)
	assert.NotEmpty(t, mf.ProjectRoot)
	assert.NotEmpty(t, mf.Languages)
	assert.NotNil(t, mf.LastFullMap)
	assert.Equal(t, len(extractions), mf.Stats.TotalModules)
	assert.Greater(t, mf.Stats.TotalFiles, 0)
	assert.Greater(t, mf.Stats.TotalSymbols, 0)
	assert.NotEmpty(t, mf.Modules)

	// Each module entry must have files with timestamps.
	for modName, mod := range mf.Modules {
		assert.NotEmpty(t, mod.Files, "module %s should have file entries", modName)
		assert.NotEmpty(t, mod.LastExtracted, "module %s should have last_extracted", modName)
		assert.Equal(t, "extracted", mod.Status, "module %s status", modName)
		for filePath, fe := range mod.Files {
			assert.NotEmpty(t, fe.ExtractedAt, "file %s should have extracted_at", filePath)
		}
	}
}

// TestFullPipeline_MermaidGeneration verifies mermaid output files are created.
func TestFullPipeline_MermaidGeneration(t *testing.T) {
	modules, err := DiscoverFiles(repoRoot, DiscoveryOpts{})
	require.NoError(t, err)

	projectModule, _ := ResolveProjectModule(repoRoot)

	var extractions []*ModuleExtraction
	for key, filePaths := range modules {
		if key.Language != "go" {
			continue
		}
		e, err := ExtractModule(key.Path, filePaths, repoRoot, projectModule, false)
		require.NoError(t, err)
		extractions = append(extractions, e)
	}

	graph := AggregateGraph(extractions, "goYoke", "")

	mermaidDir := filepath.Join(t.TempDir(), "mermaid")
	require.NoError(t, GenerateMermaid(graph, mermaidDir))

	// Module dependency diagram must exist.
	assert.FileExists(t, filepath.Join(mermaidDir, "module-dependencies.mmd"))

	// Verify content looks like a Mermaid flowchart.
	data, err := os.ReadFile(filepath.Join(mermaidDir, "module-dependencies.mmd"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "graph TD")
}

// TestMultiLanguagePipeline verifies that DiscoverFiles groups mixed-language files
// into separate ModuleKeys and that ctags extraction works for each language.
func TestMultiLanguagePipeline(t *testing.T) {
	if !CtagsAvailable() {
		t.Skip("ctags not installed")
	}

	mixedDir := filepath.Join("testdata", "mixed")
	modules, err := DiscoverFiles(mixedDir, DiscoveryOpts{})
	require.NoError(t, err)

	langs := make(map[string]bool)
	for key := range modules {
		langs[key.Language] = true
	}
	assert.True(t, langs["go"], "expected Go module in mixed directory")
	assert.True(t, langs["python"], "expected Python module in mixed directory")
	assert.True(t, langs["typescript"], "expected TypeScript module in mixed directory")
	assert.True(t, langs["rust"], "expected Rust module in mixed directory")

	// Extract each language module and verify basic output.
	for key, files := range modules {
		t.Run(key.Language, func(t *testing.T) {
			var e *ModuleExtraction
			var extractErr error

			if key.Language == "go" {
				e, extractErr = ExtractModule(key.Path, files, mixedDir, "testmixed", false)
			} else {
				e, extractErr = ExtractModuleCtags(key.Path, files, key.Language, mixedDir, "testmixed", false)
			}
			require.NoError(t, extractErr)
			require.NotNil(t, e)
			assert.Equal(t, key.Language, e.Language)
			assert.NotEmpty(t, e.Files, "expected at least one file in %s module", key.Language)
		})
	}

	// Aggregate graph should handle multi-language modules without panic.
	var extractions []*ModuleExtraction
	for key, files := range modules {
		var e *ModuleExtraction
		if key.Language == "go" {
			e, _ = ExtractModule(key.Path, files, mixedDir, "testmixed", false)
		} else {
			e, _ = ExtractModuleCtags(key.Path, files, key.Language, mixedDir, "testmixed", false)
		}
		if e != nil {
			extractions = append(extractions, e)
		}
	}
	graph := AggregateGraph(extractions, "testmixed", "test")
	assert.Greater(t, graph.Stats.TotalModules, 0)
}

// TestGoYokeMultiLanguage smoke-tests the pipeline against the goYoke repo itself.
func TestGoYokeMultiLanguage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if !CtagsAvailable() {
		t.Skip("ctags not installed")
	}

	modules, err := DiscoverFiles(repoRoot, DiscoveryOpts{})
	require.NoError(t, err)
	require.NotEmpty(t, modules)

	goCount := 0
	for key := range modules {
		if key.Language == "go" {
			goCount++
		}
	}
	assert.Greater(t, goCount, 0, "expected Go modules in goYoke repo")

	projectModule, err := ResolveProjectModule(repoRoot)
	require.NoError(t, err)

	for key, files := range modules {
		if key.Language != "go" {
			continue
		}
		e, err := ExtractModule(key.Path, files, repoRoot, projectModule, false)
		assert.NoError(t, err, "extract Go module %s should not error", key.Path)
		if e != nil {
			assert.NotEmpty(t, e.Files, "module %s should have files", key.Path)
		}
	}
}

// TestPipeline_MiniRepo runs the full extraction pipeline on the minimal fixture repo
// at testdata/integration/mini-repo/ and verifies cross-module dependency edges.
func TestPipeline_MiniRepo(t *testing.T) {
	miniRepoPath := filepath.Join("testdata", "integration", "mini-repo")

	// Stage 1: discover files — should find 2 Go modules: "" (root) and "pkg".
	modules, err := DiscoverFiles(miniRepoPath, DiscoveryOpts{})
	require.NoError(t, err)

	goModules := make(map[string][]string)
	for key, files := range modules {
		if key.Language == "go" {
			goModules[key.Path] = files
		}
	}

	assert.Len(t, goModules, 2, "mini-repo has two Go modules: root and pkg")
	assert.Contains(t, goModules, "", "root module (main.go) must be discovered")
	assert.Contains(t, goModules, "pkg", "pkg module (lib.go) must be discovered")

	// Stage 2: extract each Go module using the mini-repo's module name.
	const projectModule = "testproject"
	var extractions []*ModuleExtraction
	for path, filePaths := range goModules {
		e, err := ExtractModule(path, filePaths, miniRepoPath, projectModule, false)
		require.NoError(t, err, "extract module %q", path)
		extractions = append(extractions, e)
	}
	assert.Len(t, extractions, 2)

	// Stage 3: write per-module JSON and verify output.
	extractDir := filepath.Join(t.TempDir(), "extract")
	for _, e := range extractions {
		require.NoError(t, WriteModuleExtraction(extractDir, e))
	}
	entries, err := os.ReadDir(extractDir)
	require.NoError(t, err)
	assert.Len(t, entries, 2, "one JSON file per module")

	// Stage 4: aggregate graph — must have a dependency edge from root → pkg.
	graph := AggregateGraph(extractions, "testproject", "")
	assert.Equal(t, 2, graph.Stats.TotalModules)

	var foundEdge bool
	for _, edge := range graph.Layers.ModuleDependencies.Edges {
		if edge.From == "" && edge.To == "pkg" {
			foundEdge = true
			break
		}
	}
	assert.True(t, foundEdge, "expected dependency edge from root module to pkg")

	// Stage 5: build manifest and verify stats.
	m, err := BuildManifest(miniRepoPath, extractions, "test")
	require.NoError(t, err)
	assert.Equal(t, 2, m.Stats.TotalModules)
	assert.Greater(t, m.Stats.TotalFiles, 0)
	assert.Greater(t, m.Stats.TotalSymbols, 0)
}
