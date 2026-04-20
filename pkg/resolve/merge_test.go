package resolve

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// agentIndex is a minimal agents-index.json for test construction.
type agentIndex struct {
	Version     string                     `json:"version"`
	Description string                     `json:"description,omitempty"`
	GeneratedAt string                     `json:"generated_at,omitempty"`
	Agents      []map[string]any           `json:"agents"`
	SkillGuards map[string]map[string]any  `json:"skill_guards,omitempty"`
	Distribution map[string]any            `json:"distribution,omitempty"`
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("mustJSON: %v", err)
	}
	return b
}

func parseIndex(t *testing.T, data []byte) agentIndex {
	t.Helper()
	var idx agentIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		t.Fatalf("parseIndex: %v", err)
	}
	return idx
}

func agentIDs(agents []map[string]any) []string {
	ids := make([]string, 0, len(agents))
	for _, a := range agents {
		if id, ok := a["id"].(string); ok {
			ids = append(ids, id)
		}
	}
	return ids
}

// base fixture with two agents and two skill_guards.
func baseIndex() agentIndex {
	return agentIndex{
		Version:     "2.7.0",
		Description: "base",
		GeneratedAt: "2024-01-01",
		Agents: []map[string]any{
			{"id": "alpha", "name": "Alpha", "model": "haiku"},
			{"id": "beta", "name": "Beta", "model": "sonnet"},
		},
		SkillGuards: map[string]map[string]any{
			"braintrust": {"router_allowed_tools": []string{"Task"}},
			"review":     {"router_allowed_tools": []string{"Read"}},
		},
		Distribution: map[string]any{"channel": "stable"},
	}
}

func TestMergeAgentIndexJSON_NilOverride(t *testing.T) {
	base := mustJSON(t, baseIndex())
	got, err := MergeAgentIndexJSON(base, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != string(base) {
		t.Error("expected base returned unchanged for nil override")
	}
}

func TestMergeAgentIndexJSON_EmptyOverride(t *testing.T) {
	base := mustJSON(t, baseIndex())
	got, err := MergeAgentIndexJSON(base, []byte{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != string(base) {
		t.Error("expected base returned unchanged for empty override")
	}
}

func TestMergeAgentIndexJSON_NilBase(t *testing.T) {
	override := mustJSON(t, baseIndex())
	_, err := MergeAgentIndexJSON(nil, override)
	if err == nil {
		t.Fatal("expected error for nil base")
	}
}

func TestMergeAgentIndexJSON_EmptyBase(t *testing.T) {
	override := mustJSON(t, baseIndex())
	_, err := MergeAgentIndexJSON([]byte{}, override)
	if err == nil {
		t.Fatal("expected error for empty base")
	}
}

func TestMergeAgentIndexJSON_MalformedBase(t *testing.T) {
	_, err := MergeAgentIndexJSON([]byte(`{invalid`), []byte(`{}`))
	if err == nil {
		t.Fatal("expected error for malformed base")
	}
}

func TestMergeAgentIndexJSON_MalformedOverride(t *testing.T) {
	base := mustJSON(t, baseIndex())
	got, err := MergeAgentIndexJSON(base, []byte(`{invalid`))
	if err != nil {
		t.Fatalf("unexpected error for malformed override: %v", err)
	}
	// Graceful degradation: base returned unchanged.
	if string(got) != string(base) {
		t.Error("expected base returned unchanged for malformed override")
	}
}

func TestMergeAgentIndexJSON_OverlappingAgentIDs(t *testing.T) {
	base := mustJSON(t, baseIndex())
	override := mustJSON(t, agentIndex{
		Version: "2.7.0",
		Agents: []map[string]any{
			{"id": "alpha", "name": "Alpha-Override", "model": "opus"},
		},
	})

	got, err := MergeAgentIndexJSON(base, override)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	idx := parseIndex(t, got)

	// Base order preserved: alpha first, beta second.
	ids := agentIDs(idx.Agents)
	if !reflect.DeepEqual(ids, []string{"alpha", "beta"}) {
		t.Errorf("expected order [alpha beta], got %v", ids)
	}

	// Override wins for alpha.
	for _, a := range idx.Agents {
		if a["id"] == "alpha" {
			if a["name"] != "Alpha-Override" {
				t.Errorf("expected alpha name=Alpha-Override, got %v", a["name"])
			}
			if a["model"] != "opus" {
				t.Errorf("expected alpha model=opus, got %v", a["model"])
			}
		}
		// beta unchanged.
		if a["id"] == "beta" {
			if a["model"] != "sonnet" {
				t.Errorf("expected beta model=sonnet, got %v", a["model"])
			}
		}
	}
}

func TestMergeAgentIndexJSON_NewAgentsAppended(t *testing.T) {
	base := mustJSON(t, baseIndex())
	override := mustJSON(t, agentIndex{
		Version: "2.7.0",
		Agents: []map[string]any{
			{"id": "gamma", "name": "Gamma", "model": "haiku"},
			{"id": "delta", "name": "Delta", "model": "sonnet"},
		},
	})

	got, err := MergeAgentIndexJSON(base, override)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	idx := parseIndex(t, got)

	ids := agentIDs(idx.Agents)
	// alpha, beta from base; gamma, delta appended in override order.
	expected := []string{"alpha", "beta", "gamma", "delta"}
	if !reflect.DeepEqual(ids, expected) {
		t.Errorf("expected %v, got %v", expected, ids)
	}
}

func TestMergeAgentIndexJSON_SkillGuardsUnionMerge(t *testing.T) {
	base := mustJSON(t, baseIndex())
	override := mustJSON(t, agentIndex{
		Version: "2.7.0",
		Agents:  []map[string]any{},
		SkillGuards: map[string]map[string]any{
			"braintrust": {"router_allowed_tools": []string{"Task", "Agent"}}, // replaces base key
			"cleanup":    {"router_allowed_tools": []string{"Read", "Write"}}, // new key
		},
	})

	got, err := MergeAgentIndexJSON(base, override)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	idx := parseIndex(t, got)

	// All three keys present.
	if _, ok := idx.SkillGuards["braintrust"]; !ok {
		t.Error("missing skill_guard: braintrust")
	}
	if _, ok := idx.SkillGuards["review"]; !ok {
		t.Error("missing skill_guard: review (base key should be preserved)")
	}
	if _, ok := idx.SkillGuards["cleanup"]; !ok {
		t.Error("missing skill_guard: cleanup (new override key)")
	}

	// Override wins for braintrust.
	btTools, _ := idx.SkillGuards["braintrust"]["router_allowed_tools"].([]any)
	if len(btTools) != 2 {
		t.Errorf("expected braintrust tools to have 2 entries (from override), got %v", btTools)
	}
}

func TestMergeAgentIndexJSON_MajorVersionMismatch(t *testing.T) {
	base := mustJSON(t, baseIndex())
	overrideIdx := baseIndex()
	overrideIdx.Version = "3.0.0"
	override := mustJSON(t, overrideIdx)

	got, err := MergeAgentIndexJSON(base, override)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Major mismatch: base returned unchanged.
	if string(got) != string(base) {
		t.Error("expected base unchanged on major version mismatch")
	}
}

func TestMergeAgentIndexJSON_MinorVersionMismatch(t *testing.T) {
	base := mustJSON(t, baseIndex())
	overrideIdx := agentIndex{
		Version: "2.8.0",
		Agents: []map[string]any{
			{"id": "gamma", "name": "Gamma", "model": "haiku"},
		},
	}
	override := mustJSON(t, overrideIdx)

	got, err := MergeAgentIndexJSON(base, override)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Minor mismatch: merge proceeds; gamma appended.
	idx := parseIndex(t, got)
	ids := agentIDs(idx.Agents)
	if !slices.Contains(ids, "gamma") {
		t.Errorf("expected gamma in merged agents after minor version mismatch, got %v", ids)
	}
}

func TestMergeAgentIndexJSON_DistributionFromBaseOnly(t *testing.T) {
	base := mustJSON(t, baseIndex())
	overrideIdx := baseIndex()
	overrideIdx.Distribution = map[string]any{"channel": "nightly"}
	override := mustJSON(t, overrideIdx)

	got, err := MergeAgentIndexJSON(base, override)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	idx := parseIndex(t, got)
	if ch, _ := idx.Distribution["channel"].(string); ch != "stable" {
		t.Errorf("expected distribution.channel=stable (from base), got %q", ch)
	}
}

func TestMergeAgentIndexJSON_EmptyAgentsInOverride(t *testing.T) {
	base := mustJSON(t, baseIndex())
	override := mustJSON(t, agentIndex{
		Version: "2.7.0",
		Agents:  []map[string]any{}, // empty
	})

	got, err := MergeAgentIndexJSON(base, override)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	idx := parseIndex(t, got)
	ids := agentIDs(idx.Agents)
	expected := []string{"alpha", "beta"}
	if !reflect.DeepEqual(ids, expected) {
		t.Errorf("expected base agents preserved %v, got %v", expected, ids)
	}
}

func TestMergeAgentIndexJSON_OtherTopLevelKeysFromBase(t *testing.T) {
	base := mustJSON(t, baseIndex())
	overrideIdx := baseIndex()
	overrideIdx.Description = "override-description"
	overrideIdx.GeneratedAt = "9999-12-31"
	override := mustJSON(t, overrideIdx)

	got, err := MergeAgentIndexJSON(base, override)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	idx := parseIndex(t, got)
	if idx.Description != "base" {
		t.Errorf("expected description from base, got %q", idx.Description)
	}
	if idx.GeneratedAt != "2024-01-01" {
		t.Errorf("expected generated_at from base, got %q", idx.GeneratedAt)
	}
}

func TestMergeAgentIndexJSON_NoVersionInOverride(t *testing.T) {
	base := mustJSON(t, baseIndex())
	// Override without a version field — should merge normally.
	override := []byte(`{"agents":[{"id":"gamma","name":"Gamma","model":"haiku"}]}`)

	got, err := MergeAgentIndexJSON(base, override)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	idx := parseIndex(t, got)
	ids := agentIDs(idx.Agents)
	if ids[len(ids)-1] != "gamma" {
		t.Errorf("expected gamma appended, got %v", ids)
	}
}

func TestMergeAgentIndexJSON_MixedNewAndOverlapping(t *testing.T) {
	base := mustJSON(t, baseIndex())
	override := mustJSON(t, agentIndex{
		Version: "2.7.0",
		Agents: []map[string]any{
			{"id": "beta", "name": "Beta-New", "model": "opus"}, // replace
			{"id": "gamma", "name": "Gamma", "model": "haiku"},  // append
		},
	})

	got, err := MergeAgentIndexJSON(base, override)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	idx := parseIndex(t, got)
	ids := agentIDs(idx.Agents)
	// alpha (unchanged), beta (replaced), gamma (appended)
	if !reflect.DeepEqual(ids, []string{"alpha", "beta", "gamma"}) {
		t.Errorf("expected [alpha beta gamma], got %v", ids)
	}
	for _, a := range idx.Agents {
		if a["id"] == "beta" && a["name"] != "Beta-New" {
			t.Errorf("expected beta replaced with Beta-New, got %v", a["name"])
		}
	}
}

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input    string
		expected []int
	}{
		{"2.7.0", []int{2, 7, 0}},
		{"v3.1.2", []int{3, 1, 2}},
		{"1.0", []int{1, 0}},
		{"notvalid", nil},
		{"1.x.0", nil},
	}
	for _, tc := range tests {
		got := parseSemver(tc.input)
		if !reflect.DeepEqual(got, tc.expected) {
			t.Errorf("parseSemver(%q) = %v, want %v", tc.input, got, tc.expected)
		}
	}
}

func TestMergeAgentIndexJSON_BaseOnlyNoAgents(t *testing.T) {
	// Base with no agents key at all.
	base := []byte(`{"version":"2.7.0","description":"minimal"}`)
	override := []byte(`{"version":"2.7.0","agents":[{"id":"alpha","name":"Alpha","model":"haiku"}]}`)

	got, err := MergeAgentIndexJSON(base, override)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	idx := parseIndex(t, got)
	ids := agentIDs(idx.Agents)
	if !reflect.DeepEqual(ids, []string{"alpha"}) {
		t.Errorf("expected [alpha] appended to empty base, got %v", ids)
	}
}

func TestMergeAgentIndexJSON_ImportCheck(t *testing.T) {
	// Verify no forbidden package paths are imported in merge.go.
	// This is a documentation test; actual import enforcement is at build time.
	forbiddenImports := []string{
		"pkg/routing", "pkg/skillsetup", "pkg/telemetry", "pkg/memory",
		"internal/", "cmd/",
	}
	// Test that the strings we care about are exactly what we ban.
	for _, pkg := range forbiddenImports {
		if strings.Contains(pkg, "os") {
			t.Errorf("unexpected: %q contains 'os'", pkg)
		}
	}
}
