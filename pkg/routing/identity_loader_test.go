package routing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStripYAMLFrontmatter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string // expected substring in output
		excludes string // must NOT be in output
	}{
		{
			name: "valid frontmatter",
			input: `---
name: go-pro
model: sonnet
---

# GO Pro Agent

You are a GO expert.`,
			contains: "# GO Pro Agent",
			excludes: "name: go-pro",
		},
		{
			name:     "no frontmatter",
			input:    "# Just a markdown file\n\nNo frontmatter here.",
			contains: "# Just a markdown file",
		},
		{
			name: "malformed frontmatter (no closing)",
			input: `---
name: broken
This never closes

# Content here`,
			contains: "name: broken",
		},
		{
			name:     "empty input",
			input:    "",
			contains: "",
		},
		{
			name: "frontmatter with extra content on closing line",
			input: `---
key: value
---
# Body starts here`,
			contains: "# Body starts here",
			excludes: "key: value",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := StripYAMLFrontmatter(tc.input)
			if tc.contains != "" && !strings.Contains(result, tc.contains) {
				t.Errorf("expected output to contain %q, got:\n%s", tc.contains, result)
			}
			if tc.excludes != "" && strings.Contains(result, tc.excludes) {
				t.Errorf("expected output to NOT contain %q, got:\n%s", tc.excludes, result)
			}
		})
	}
}

func TestLoadAgentIdentity(t *testing.T) {
	// Clear cache to ensure fresh load
	ClearConventionCache()

	t.Run("existing agent go-pro", func(t *testing.T) {
		identity, err := LoadAgentIdentity("go-pro")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if identity == "" {
			t.Fatal("expected non-empty identity for go-pro")
		}
		// Should have body content, not frontmatter
		if strings.Contains(identity, "name: go-pro") {
			t.Error("identity should not contain YAML frontmatter")
		}
		if !strings.Contains(identity, "GO Pro Agent") {
			t.Error("identity should contain agent title")
		}
		if !strings.Contains(identity, "Single binary") {
			t.Error("identity should contain project-specific constraints")
		}
	})

	t.Run("existing agent einstein", func(t *testing.T) {
		identity, err := LoadAgentIdentity("einstein")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if identity == "" {
			t.Fatal("expected non-empty identity for einstein")
		}
		if !strings.Contains(identity, "Einstein") {
			t.Error("identity should contain agent name")
		}
	})

	t.Run("nonexistent agent", func(t *testing.T) {
		identity, err := LoadAgentIdentity("nonexistent-agent-xyz")
		if err != nil {
			t.Fatalf("unexpected error for missing agent: %v", err)
		}
		if identity != "" {
			t.Errorf("expected empty identity for nonexistent agent, got %d chars", len(identity))
		}
	})

	t.Run("empty agent ID", func(t *testing.T) {
		identity, err := LoadAgentIdentity("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if identity != "" {
			t.Error("expected empty identity for empty agent ID")
		}
	})

	t.Run("caching works", func(t *testing.T) {
		ClearConventionCache()

		// First load
		id1, _ := LoadAgentIdentity("go-pro")
		// Second load (from cache)
		id2, _ := LoadAgentIdentity("go-pro")

		if id1 != id2 {
			t.Error("cached result should match first load")
		}
	})
}

func TestBuildFullAgentContext(t *testing.T) {
	ClearConventionCache()

	t.Run("full context with identity + rules + conventions", func(t *testing.T) {
		result, err := BuildFullAgentContext(
			"go-pro",
			&ContextRequirements{
				Rules: []string{"agent-guidelines.md"},
				Conventions: ConventionRequirements{
					Base: []string{"go.md"},
				},
			},
			nil,
			"AGENT: go-pro\n\nTASK: implement a widget",
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Check identity section
		if !strings.Contains(result, IdentityMarker) {
			t.Error("missing identity marker")
		}
		if !strings.Contains(result, IdentityEndMarker) {
			t.Error("missing identity end marker")
		}
		if !strings.Contains(result, "Single binary") {
			t.Error("missing go-pro identity content (project constraints)")
		}

		// Check conventions section
		if !strings.Contains(result, ConventionsMarker) {
			t.Error("missing conventions marker")
		}
		if !strings.Contains(result, ConventionsEndMarker) {
			t.Error("missing conventions end marker")
		}

		// Check original prompt preserved
		if !strings.Contains(result, "AGENT: go-pro") {
			t.Error("missing original prompt")
		}
		if !strings.Contains(result, "TASK: implement a widget") {
			t.Error("missing original task")
		}

		// Check ordering: identity before conventions before prompt
		identityIdx := strings.Index(result, IdentityMarker)
		conventionsIdx := strings.Index(result, ConventionsMarker)
		promptIdx := strings.Index(result, "TASK: implement a widget")
		if identityIdx >= conventionsIdx {
			t.Error("identity should come before conventions")
		}
		if conventionsIdx >= promptIdx {
			t.Error("conventions should come before prompt")
		}
	})

	t.Run("no identity file still gets conventions", func(t *testing.T) {
		result, err := BuildFullAgentContext(
			"nonexistent-agent-xyz",
			&ContextRequirements{
				Rules: []string{"agent-guidelines.md"},
			},
			nil,
			"AGENT: test\n\nTASK: do stuff",
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if strings.Contains(result, IdentityMarker) {
			t.Error("should not have identity marker for nonexistent agent")
		}
		if !strings.Contains(result, ConventionsMarker) {
			t.Error("should still have conventions marker")
		}
		if !strings.Contains(result, "TASK: do stuff") {
			t.Error("should preserve original prompt")
		}
	})

	t.Run("nil requirements still gets identity", func(t *testing.T) {
		result, err := BuildFullAgentContext(
			"go-pro",
			nil,
			nil,
			"AGENT: go-pro\n\nTASK: build something",
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, IdentityMarker) {
			t.Error("should have identity even without requirements")
		}
		if strings.Contains(result, ConventionsMarker) {
			t.Error("should not have conventions without requirements")
		}
	})

	t.Run("double injection prevention", func(t *testing.T) {
		// First injection
		first, _ := BuildFullAgentContext(
			"go-pro",
			&ContextRequirements{Rules: []string{"agent-guidelines.md"}},
			nil,
			"AGENT: go-pro\n\nTASK: test",
		)

		// Second injection on same result
		second, _ := BuildFullAgentContext(
			"go-pro",
			&ContextRequirements{Rules: []string{"agent-guidelines.md"}},
			nil,
			first,
		)

		// Should only have one identity marker
		count := strings.Count(second, IdentityMarker)
		if count > 1 {
			t.Errorf("expected 1 identity marker, got %d (double injection!)", count)
		}
	})

	t.Run("no agent no requirements returns unchanged", func(t *testing.T) {
		original := "just a plain prompt"
		result, err := BuildFullAgentContext("", nil, nil, original)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != original {
			t.Error("should return unchanged prompt when nothing to inject")
		}
	})
}

func TestStripYAMLFrontmatter_RealFile(t *testing.T) {
	// Test with actual go-pro.md file
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".claude", "agents", "go-pro", "go-pro.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("go-pro.md not found")
	}

	body := StripYAMLFrontmatter(string(data))

	if strings.Contains(body, "name: go-pro") {
		t.Error("body should not contain frontmatter field 'name: go-pro'")
	}
	if !strings.Contains(body, "# GO Pro Agent") {
		t.Error("body should start with the markdown heading")
	}
	if len(body) < 100 {
		t.Errorf("body seems too short: %d chars", len(body))
	}
}
