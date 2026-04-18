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
	// Isolate from CC session env vars that would inject session markers
	t.Setenv("GOYOKE_SESSION_DIR", "")

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

func TestGetSessionDir(t *testing.T) {
	tests := []struct {
		name        string
		setupEnv    func(*testing.T) string // returns expected session dir path
		expectEmpty bool
	}{
		{
			name: "GOYOKE_SESSION_DIR set directly (team-run path)",
			setupEnv: func(t *testing.T) string {
				sessionDir := "/tmp/fake-session-dir"
				t.Setenv("GOYOKE_SESSION_DIR", sessionDir)
				t.Setenv("GOYOKE_PROJECT_ROOT", "")
				t.Setenv("GOYOKE_PROJECT_DIR", "")
				t.Setenv("CLAUDE_PROJECT_DIR", "")
				return sessionDir
			},
		},
		{
			name: "GOYOKE_SESSION_DIR takes priority over file-based",
			setupEnv: func(t *testing.T) string {
				// Set up both: direct env var AND file-based marker
				tmpDir := t.TempDir()
				claudeDir := filepath.Join(tmpDir, ".goyoke")
				os.MkdirAll(claudeDir, 0755)
				os.WriteFile(filepath.Join(claudeDir, "current-session"), []byte("/file-based-path"), 0644)
				t.Setenv("GOYOKE_PROJECT_ROOT", tmpDir)
				// Direct env var should win
				t.Setenv("GOYOKE_SESSION_DIR", "/direct-env-path")
				return "/direct-env-path"
			},
		},
		{
			name: "no env vars set",
			setupEnv: func(t *testing.T) string {
				t.Setenv("GOYOKE_SESSION_DIR", "")
				t.Setenv("GOYOKE_PROJECT_ROOT", "")
				t.Setenv("GOYOKE_PROJECT_DIR", "")
				t.Setenv("CLAUDE_PROJECT_DIR", "")
				return ""
			},
			expectEmpty: true,
		},
		{
			name: "GOYOKE_PROJECT_ROOT set with current-session file",
			setupEnv: func(t *testing.T) string {
				tmpDir := t.TempDir()
				sessionDir := filepath.Join(tmpDir, ".goyoke", "sessions", "test-session")
				claudeDir := filepath.Join(tmpDir, ".goyoke")
				os.MkdirAll(claudeDir, 0755)
				os.WriteFile(filepath.Join(claudeDir, "current-session"), []byte(sessionDir), 0644)
				t.Setenv("GOYOKE_SESSION_DIR", "")
				t.Setenv("GOYOKE_PROJECT_ROOT", tmpDir)
				t.Setenv("GOYOKE_PROJECT_DIR", "")
				t.Setenv("CLAUDE_PROJECT_DIR", "")
				return sessionDir
			},
		},
		{
			name: "GOYOKE_PROJECT_DIR set (no ROOT)",
			setupEnv: func(t *testing.T) string {
				tmpDir := t.TempDir()
				sessionDir := filepath.Join(tmpDir, ".goyoke", "sessions", "another-session")
				claudeDir := filepath.Join(tmpDir, ".goyoke")
				os.MkdirAll(claudeDir, 0755)
				os.WriteFile(filepath.Join(claudeDir, "current-session"), []byte(sessionDir), 0644)
				t.Setenv("GOYOKE_SESSION_DIR", "")
				t.Setenv("GOYOKE_PROJECT_ROOT", "")
				t.Setenv("GOYOKE_PROJECT_DIR", tmpDir)
				t.Setenv("CLAUDE_PROJECT_DIR", "")
				return sessionDir
			},
		},
		{
			name: "CLAUDE_PROJECT_DIR set (no ROOT, no DIR)",
			setupEnv: func(t *testing.T) string {
				tmpDir := t.TempDir()
				sessionDir := filepath.Join(tmpDir, ".goyoke", "sessions", "legacy-session")
				claudeDir := filepath.Join(tmpDir, ".goyoke")
				os.MkdirAll(claudeDir, 0755)
				os.WriteFile(filepath.Join(claudeDir, "current-session"), []byte(sessionDir), 0644)
				t.Setenv("GOYOKE_SESSION_DIR", "")
				t.Setenv("GOYOKE_PROJECT_ROOT", "")
				t.Setenv("GOYOKE_PROJECT_DIR", "")
				t.Setenv("CLAUDE_PROJECT_DIR", tmpDir)
				return sessionDir
			},
		},
		{
			name: "env var set but current-session file missing",
			setupEnv: func(t *testing.T) string {
				tmpDir := t.TempDir()
				os.MkdirAll(filepath.Join(tmpDir, ".goyoke"), 0755)
				// Don't write current-session file
				t.Setenv("GOYOKE_SESSION_DIR", "")
				t.Setenv("GOYOKE_PROJECT_ROOT", tmpDir)
				t.Setenv("GOYOKE_PROJECT_DIR", "")
				t.Setenv("CLAUDE_PROJECT_DIR", "")
				return ""
			},
			expectEmpty: true,
		},
		{
			name: "current-session file has trailing whitespace",
			setupEnv: func(t *testing.T) string {
				tmpDir := t.TempDir()
				sessionDir := filepath.Join(tmpDir, ".goyoke", "sessions", "whitespace-test")
				claudeDir := filepath.Join(tmpDir, ".goyoke")
				os.MkdirAll(claudeDir, 0755)
				// Write with trailing whitespace and newlines
				os.WriteFile(filepath.Join(claudeDir, "current-session"), []byte(sessionDir+"  \n\n"), 0644)
				t.Setenv("GOYOKE_SESSION_DIR", "")
				t.Setenv("GOYOKE_PROJECT_ROOT", tmpDir)
				t.Setenv("GOYOKE_PROJECT_DIR", "")
				t.Setenv("CLAUDE_PROJECT_DIR", "")
				return sessionDir // Expected result should be trimmed
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expected := tc.setupEnv(t)
			result := GetSessionDir()

			if tc.expectEmpty {
				if result != "" {
					t.Errorf("expected empty result, got: %q", result)
				}
			} else {
				if result != expected {
					t.Errorf("expected %q, got %q", expected, result)
				}
			}
		})
	}
}

func TestBuildFullAgentContext_SessionMarker(t *testing.T) {
	ClearConventionCache()

	tests := []struct {
		name         string
		setupEnv     func(*testing.T) string // returns session dir path
		agentID      string
		requirements *ContextRequirements
		prompt       string
		checkFn      func(*testing.T, string, string) // check function receives result and session path
	}{
		{
			name: "session dir present, session markers injected",
			setupEnv: func(t *testing.T) string {
				tmpDir := t.TempDir()
				sessionDir := filepath.Join(tmpDir, ".goyoke", "sessions", "test-123")
				claudeDir := filepath.Join(tmpDir, ".goyoke")
				os.MkdirAll(claudeDir, 0755)
				os.WriteFile(filepath.Join(claudeDir, "current-session"), []byte(sessionDir), 0644)
				t.Setenv("GOYOKE_SESSION_DIR", "")
				t.Setenv("GOYOKE_PROJECT_ROOT", tmpDir)
				t.Setenv("GOYOKE_PROJECT_DIR", "")
				t.Setenv("CLAUDE_PROJECT_DIR", "")
				return sessionDir
			},
			agentID: "go-pro",
			requirements: &ContextRequirements{
				Rules: []string{"agent-guidelines.md"},
			},
			prompt: "TASK: do something",
			checkFn: func(t *testing.T, result, sessionPath string) {
				if !strings.Contains(result, SessionMarker) {
					t.Error("missing session marker")
				}
				if !strings.Contains(result, SessionEndMarker) {
					t.Error("missing session end marker")
				}
				if !strings.Contains(result, "SESSION_DIR: "+sessionPath) {
					t.Errorf("missing SESSION_DIR line with path %q", sessionPath)
				}
				if !strings.Contains(result, "Write output artifacts") {
					t.Error("missing session context instructions")
				}
			},
		},
		{
			name: "session context appears before identity marker",
			setupEnv: func(t *testing.T) string {
				tmpDir := t.TempDir()
				sessionDir := filepath.Join(tmpDir, ".goyoke", "sessions", "ordering-test")
				claudeDir := filepath.Join(tmpDir, ".goyoke")
				os.MkdirAll(claudeDir, 0755)
				os.WriteFile(filepath.Join(claudeDir, "current-session"), []byte(sessionDir), 0644)
				t.Setenv("GOYOKE_PROJECT_ROOT", tmpDir)
				return sessionDir
			},
			agentID: "go-pro",
			requirements: &ContextRequirements{
				Rules: []string{"agent-guidelines.md"},
			},
			prompt: "TASK: order check",
			checkFn: func(t *testing.T, result, sessionPath string) {
				sessionIdx := strings.Index(result, SessionMarker)
				identityIdx := strings.Index(result, IdentityMarker)
				if sessionIdx == -1 {
					t.Fatal("session marker not found")
				}
				if identityIdx == -1 {
					t.Fatal("identity marker not found")
				}
				if sessionIdx >= identityIdx {
					t.Error("session context should appear before identity marker")
				}
			},
		},
		{
			name: "prompt already contains session marker, no double injection",
			setupEnv: func(t *testing.T) string {
				tmpDir := t.TempDir()
				sessionDir := filepath.Join(tmpDir, ".goyoke", "sessions", "double-test")
				claudeDir := filepath.Join(tmpDir, ".goyoke")
				os.MkdirAll(claudeDir, 0755)
				os.WriteFile(filepath.Join(claudeDir, "current-session"), []byte(sessionDir), 0644)
				t.Setenv("GOYOKE_PROJECT_ROOT", tmpDir)
				return sessionDir
			},
			agentID:      "go-pro",
			requirements: nil,
			prompt: SessionMarker + "\nSESSION_DIR: /existing/path\n" + SessionEndMarker + "\n\nTASK: test",
			checkFn: func(t *testing.T, result, sessionPath string) {
				count := strings.Count(result, SessionMarker)
				if count > 1 {
					t.Errorf("expected 1 session marker, got %d (double injection!)", count)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sessionPath := tc.setupEnv(t)
			result, err := BuildFullAgentContext(tc.agentID, tc.requirements, nil, tc.prompt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tc.checkFn(t, result, sessionPath)
		})
	}
}

func TestBuildFullAgentContext_NoSessionDir(t *testing.T) {
	ClearConventionCache()

	// Clear all env vars (including GOYOKE_SESSION_DIR which leaks from CC sessions)
	t.Setenv("GOYOKE_SESSION_DIR", "")
	t.Setenv("GOYOKE_PROJECT_ROOT", "")
	t.Setenv("GOYOKE_PROJECT_DIR", "")
	t.Setenv("CLAUDE_PROJECT_DIR", "")

	result, err := BuildFullAgentContext(
		"go-pro",
		&ContextRequirements{
			Rules: []string{"agent-guidelines.md"},
		},
		nil,
		"TASK: no session context",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should NOT contain session markers when no env vars set
	if strings.Contains(result, SessionMarker) {
		t.Error("should not have session marker when no env vars set")
	}
	if strings.Contains(result, SessionEndMarker) {
		t.Error("should not have session end marker when no env vars set")
	}

	// Should still have identity and conventions
	if !strings.Contains(result, IdentityMarker) {
		t.Error("should still have identity marker")
	}
	if !strings.Contains(result, ConventionsMarker) {
		t.Error("should still have conventions marker")
	}
}
