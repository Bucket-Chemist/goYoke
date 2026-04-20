package routing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Bucket-Chemist/goYoke/internal/codemap"
)

func makeTestGraph(t *testing.T, nodes []codemap.ModuleNode, edges []codemap.ModuleDependencyEdge, generatedAt string) string {
	t.Helper()
	tmpDir := t.TempDir()
	mapDir := filepath.Join(tmpDir, ".claude", "codebase-map")
	if err := os.MkdirAll(mapDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if generatedAt == "" {
		generatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	g := map[string]any{
		"version":      "1.0",
		"generated_at": generatedAt,
		"layers": map[string]any{
			"module_dependencies": map[string]any{
				"nodes": nodes,
				"edges": edges,
			},
		},
	}
	data, err := json.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mapDir, "graph.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	return tmpDir
}

func TestIdentifyModulesFromPrompt(t *testing.T) {
	moduleIDs := []string{"internal/routing", "cmd/goyoke-validate", "pkg/config", "internal/tui/mcp", "pkg/enforcement"}

	tests := []struct {
		name   string
		prompt string
		want   []string
	}{
		{
			name:   "exact module ID",
			prompt: "fix bug in internal/routing",
			want:   []string{"internal/routing"},
		},
		{
			name:   "file path prefix",
			prompt: "fix bug in internal/routing/validator.go",
			want:   []string{"internal/routing"},
		},
		{
			name:   "directory reference",
			prompt: "refactor cmd/goyoke-validate",
			want:   []string{"cmd/goyoke-validate"},
		},
		{
			name:   "no match returns nil",
			prompt: "implement a new feature from scratch",
			want:   nil,
		},
		{
			name:   "multiple modules",
			prompt: "fix internal/routing and pkg/config",
			want:   []string{"internal/routing", "pkg/config"},
		},
		{
			name:   "max 3 modules capped",
			prompt: "fix internal/routing, cmd/goyoke-validate, pkg/config, and internal/tui/mcp",
			want:   []string{"internal/routing", "cmd/goyoke-validate", "pkg/config"},
		},
		{
			name:   "fuzzy directory name match",
			prompt: "the routing package needs a refactor",
			want:   []string{"internal/routing"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := identifyModulesFromPrompt(tc.prompt, moduleIDs)
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i, id := range tc.want {
				if got[i] != id {
					t.Errorf("position %d: got %q, want %q", i, got[i], id)
				}
			}
		})
	}
}

func TestFormatModuleContext(t *testing.T) {
	desc := "Core routing logic"
	nodes := []codemap.ModuleNode{
		{
			ID:           "internal/routing",
			Category:     "internal",
			Language:     "go",
			SymbolCount:  240,
			FileCount:    23,
			Description:  &desc,
			KeyTypes:     []string{"ContextRequirements", "Agent"},
			KeyFunctions: []string{"BuildFullAgentContext", "LoadAgentIdentity"},
		},
	}
	edges := []codemap.ModuleDependencyEdge{
		{From: "internal/routing", To: "pkg/config"},
		{From: "cmd/goyoke-validate", To: "internal/routing"},
	}

	result := formatModuleContext(nodes, edges, "")

	if !strings.Contains(result, "## Module Context (from codebase-map)") {
		t.Error("missing header")
	}
	if !strings.Contains(result, "**Module**: internal/routing") {
		t.Error("missing module ID")
	}
	if !strings.Contains(result, "**Category**: internal") {
		t.Error("missing category")
	}
	if !strings.Contains(result, "Core routing logic") {
		t.Error("missing description")
	}
	if !strings.Contains(result, "ContextRequirements") {
		t.Error("missing key types")
	}
	if !strings.Contains(result, "BuildFullAgentContext") {
		t.Error("missing key functions")
	}
	if !strings.Contains(result, "**Dependencies**: pkg/config") {
		t.Error("missing dependencies")
	}
	if !strings.Contains(result, "**Depended on by**: cmd/goyoke-validate") {
		t.Error("missing depended on by")
	}
	if !strings.Contains(result, "**Symbols**: 240 (23 files)") {
		t.Error("missing symbols line")
	}
}

func TestFormatModuleContext_WithStaleWarning(t *testing.T) {
	nodes := []codemap.ModuleNode{{ID: "pkg/config", Category: "pkg", SymbolCount: 10, FileCount: 1}}
	result := formatModuleContext(nodes, nil, "(Note: Module map is 10 days old. Run /codebase-map --incremental to update.)")
	if !strings.Contains(result, "10 days old") {
		t.Error("missing stale warning")
	}
}

func TestFormatModuleContext_Empty(t *testing.T) {
	result := formatModuleContext(nil, nil, "")
	if result != "" {
		t.Errorf("expected empty, got %q", result)
	}
}

func TestInjectCodebaseMapContext_FeatureFlagOff(t *testing.T) {
	t.Setenv("GOYOKE_CODEBASE_MAP_INJECT", "")
	tmpDir := makeTestGraph(t,
		[]codemap.ModuleNode{{ID: "internal/routing", Category: "internal", SymbolCount: 1, FileCount: 1}},
		nil, "")

	result := InjectCodebaseMapContext("go-pro", "fix internal/routing bug", tmpDir)
	if result != "" {
		t.Errorf("expected empty when flag off, got %q", result)
	}
}

func TestInjectCodebaseMapContext_NoGraphFile(t *testing.T) {
	t.Setenv("GOYOKE_CODEBASE_MAP_INJECT", "1")
	tmpDir := t.TempDir() // no graph.json

	result := InjectCodebaseMapContext("go-pro", "fix internal/routing bug", tmpDir)
	if result != "" {
		t.Errorf("expected empty when no graph file, got %q", result)
	}
}

func TestInjectCodebaseMapContext_AgentNotEligible(t *testing.T) {
	t.Setenv("GOYOKE_CODEBASE_MAP_INJECT", "1")
	tmpDir := makeTestGraph(t,
		[]codemap.ModuleNode{{ID: "internal/routing", Category: "internal", SymbolCount: 1, FileCount: 1}},
		nil, "")

	result := InjectCodebaseMapContext("some-other-agent", "fix internal/routing bug", tmpDir)
	if result != "" {
		t.Errorf("expected empty for non-eligible agent, got %q", result)
	}
}

func TestInjectCodebaseMapContext_NoModulesIdentified(t *testing.T) {
	t.Setenv("GOYOKE_CODEBASE_MAP_INJECT", "1")
	tmpDir := makeTestGraph(t,
		[]codemap.ModuleNode{{ID: "internal/routing", Category: "internal", SymbolCount: 1, FileCount: 1}},
		nil, "")

	result := InjectCodebaseMapContext("go-pro", "implement a brand new feature", tmpDir)
	if result != "" {
		t.Errorf("expected empty when no modules matched, got %q", result)
	}
}

func TestInjectCodebaseMapContext_Injects(t *testing.T) {
	t.Setenv("GOYOKE_CODEBASE_MAP_INJECT", "1")
	desc := "Core routing logic"
	tmpDir := makeTestGraph(t,
		[]codemap.ModuleNode{
			{
				ID: "internal/routing", Category: "internal", Language: "go",
				SymbolCount: 240, FileCount: 23, Description: &desc,
				KeyTypes:     []string{"Agent"},
				KeyFunctions: []string{"BuildFullAgentContext"},
			},
		},
		[]codemap.ModuleDependencyEdge{{From: "internal/routing", To: "pkg/config"}},
		"")

	result := InjectCodebaseMapContext("go-pro", "fix bug in internal/routing/validator.go", tmpDir)

	if result == "" {
		t.Fatal("expected non-empty context block")
	}
	if !strings.Contains(result, "internal/routing") {
		t.Error("missing module ID in context")
	}
	if !strings.Contains(result, "Core routing logic") {
		t.Error("missing description in context")
	}
	if !strings.Contains(result, "pkg/config") {
		t.Error("missing dependency in context")
	}
}

func TestStalenessWarning(t *testing.T) {
	t.Run("fresh map returns empty", func(t *testing.T) {
		recent := time.Now().UTC().Format(time.RFC3339)
		if w := mapStalenessWarning(recent); w != "" {
			t.Errorf("expected empty for fresh map, got %q", w)
		}
	})

	t.Run("stale map returns warning", func(t *testing.T) {
		old := time.Now().UTC().Add(-10 * 24 * time.Hour).Format(time.RFC3339)
		w := mapStalenessWarning(old)
		if !strings.Contains(w, "days old") {
			t.Errorf("expected staleness warning, got %q", w)
		}
		if !strings.Contains(w, "codebase-map") {
			t.Errorf("expected update hint in warning, got %q", w)
		}
	})

	t.Run("empty timestamp returns empty", func(t *testing.T) {
		if w := mapStalenessWarning(""); w != "" {
			t.Errorf("expected empty for missing timestamp, got %q", w)
		}
	})

	t.Run("unparseable timestamp returns empty", func(t *testing.T) {
		if w := mapStalenessWarning("not-a-date"); w != "" {
			t.Errorf("expected empty for bad timestamp, got %q", w)
		}
	})
}

func TestInjectCodebaseMapContext_StaleWarningIncluded(t *testing.T) {
	t.Setenv("GOYOKE_CODEBASE_MAP_INJECT", "1")
	oldTime := time.Now().UTC().Add(-10 * 24 * time.Hour).Format(time.RFC3339)
	tmpDir := makeTestGraph(t,
		[]codemap.ModuleNode{{ID: "internal/routing", Category: "internal", SymbolCount: 1, FileCount: 1}},
		nil,
		oldTime)

	result := InjectCodebaseMapContext("go-pro", "fix internal/routing bug", tmpDir)
	if !strings.Contains(result, "days old") {
		t.Errorf("expected stale warning in output, got %q", result)
	}
}

func TestExtractPathTokens(t *testing.T) {
	tokens := extractPathTokens("fix bug in internal/routing/validator.go and pkg/config/config.go")
	found := map[string]bool{}
	for _, tok := range tokens {
		found[tok] = true
	}
	if !found["internal/routing/validator"] {
		t.Errorf("expected internal/routing/validator in tokens, got %v", tokens)
	}
	if !found["pkg/config/config"] {
		t.Errorf("expected pkg/config/config in tokens, got %v", tokens)
	}
}
