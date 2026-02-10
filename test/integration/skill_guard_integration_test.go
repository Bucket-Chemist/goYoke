package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	skillGuardBinary string
	buildOnce        sync.Once
	buildErr         error
)

// buildSkillGuard builds the gogent-skill-guard binary once per test run
func buildSkillGuard(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		// Get project root relative to test file
		_, filename, _, ok := runtime.Caller(0)
		if !ok {
			buildErr = fmt.Errorf("failed to get runtime caller info")
			return
		}
		projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")

		// Use os.TempDir() not t.TempDir() - binary needs to persist across tests
		tmpDir := os.TempDir()
		skillGuardBinary = filepath.Join(tmpDir, fmt.Sprintf("gogent-skill-guard-test-%d", time.Now().UnixNano()))
		cmd := exec.Command("go", "build", "-o", skillGuardBinary, "./cmd/gogent-skill-guard")
		cmd.Dir = projectRoot
		output, err := cmd.CombinedOutput()
		if err != nil {
			buildErr = fmt.Errorf("build failed: %v\nOutput: %s", err, output)
		}
	})
	if buildErr != nil {
		t.Fatalf("Failed to build skill-guard binary: %v", buildErr)
	}
	return skillGuardBinary
}

// setupConfigDir creates a temporary config directory with agents-index.json fixture
func setupConfigDir(t *testing.T) string {
	t.Helper()
	configDir := t.TempDir()
	agentsDir := filepath.Join(configDir, "agents")
	require.NoError(t, os.MkdirAll(agentsDir, 0755))

	indexJSON := `{
  "skill_guards": {
    "braintrust": {
      "router_allowed_tools": ["Task", "Bash", "Read", "AskUserQuestion", "Skill"],
      "team_dir_suffix": "braintrust"
    },
    "review": {
      "router_allowed_tools": ["Task", "Bash", "Read", "Glob", "AskUserQuestion", "Skill"],
      "team_dir_suffix": "code-review"
    },
    "implement": {
      "router_allowed_tools": ["Task", "Bash", "Read", "AskUserQuestion", "Skill"],
      "team_dir_suffix": "implementation"
    }
  }
}`
	require.NoError(t, os.WriteFile(
		filepath.Join(agentsDir, "agents-index.json"),
		[]byte(indexJSON), 0644))

	return configDir
}

// runSkillGuard executes the skill-guard binary with the given stdin and environment
// Returns (stdout, stderr, error)
func runSkillGuard(t *testing.T, binary string, stdin string, env []string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(binary)
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Env = env

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// TestBinary_SkillSetupCreatesGuard tests that the Skill tool creates a guard file and team directory
func TestBinary_SkillSetupCreatesGuard(t *testing.T) {
	binary := buildSkillGuard(t)
	sessionDir := t.TempDir()
	configDir := setupConfigDir(t)

	stdin := `{"tool_name":"Skill","tool_input":{"skill":"review"}}`

	env := append(os.Environ(),
		"GOGENT_SESSION_DIR="+sessionDir,
		"CLAUDE_CONFIG_DIR="+configDir,
	)

	stdout, _, err := runSkillGuard(t, binary, stdin, env)
	require.NoError(t, err, "skill-guard should execute successfully")
	assert.Equal(t, "{}\n", stdout, "setup mode should output empty JSON")

	// Verify guard file created
	guardPath := filepath.Join(sessionDir, "active-skill.json")
	guardData, err := os.ReadFile(guardPath)
	require.NoError(t, err, "guard file should exist")

	var guard struct {
		Skill              string   `json:"skill"`
		TeamDir            string   `json:"team_dir"`
		RouterAllowedTools []string `json:"router_allowed_tools"`
		CreatedAt          string   `json:"created_at"`
	}
	require.NoError(t, json.Unmarshal(guardData, &guard), "guard file should contain valid JSON")

	// Verify guard file contents
	assert.Equal(t, "review", guard.Skill, "guard should record correct skill name")
	assert.Contains(t, guard.TeamDir, ".code-review", "team dir should use correct suffix")
	assert.Contains(t, guard.RouterAllowedTools, "Glob", "guard should record allowed tools from config")
	assert.Contains(t, guard.RouterAllowedTools, "Task", "guard should include Task in allowed tools")
	assert.NotEmpty(t, guard.CreatedAt, "guard should have creation timestamp")

	// Verify CreatedAt is valid RFC3339
	_, err = time.Parse(time.RFC3339, guard.CreatedAt)
	assert.NoError(t, err, "CreatedAt should be valid RFC3339 timestamp")

	// Verify team dir exists
	_, err = os.Stat(guard.TeamDir)
	assert.NoError(t, err, "team directory should be created")
}

// TestBinary_GuardBlocksTool tests that blocked tools receive a block response
func TestBinary_GuardBlocksTool(t *testing.T) {
	binary := buildSkillGuard(t)
	sessionDir := t.TempDir()
	configDir := setupConfigDir(t)

	env := append(os.Environ(),
		"GOGENT_SESSION_DIR="+sessionDir,
		"CLAUDE_CONFIG_DIR="+configDir,
	)

	// Setup: create guard via Skill invocation
	setupStdin := `{"tool_name":"Skill","tool_input":{"skill":"braintrust"}}`
	_, _, err := runSkillGuard(t, binary, setupStdin, env)
	require.NoError(t, err, "setup should succeed")

	// Guard: test blocked tool (Glob is not in braintrust's allowed list)
	guardStdin := `{"tool_name":"Glob","tool_input":{"pattern":"*.go"}}`
	stdout, _, err := runSkillGuard(t, binary, guardStdin, env)
	require.NoError(t, err, "guard mode should execute successfully")

	// Parse response
	var response map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &response), "output should be valid JSON")

	// Verify block decision
	assert.Equal(t, "block", response["decision"], "should block disallowed tool")
	reason, ok := response["reason"].(string)
	require.True(t, ok, "reason should be a string")
	assert.Contains(t, reason, "braintrust", "reason should mention active skill")
	assert.Contains(t, reason, "Glob", "reason should mention blocked tool")
}

// TestBinary_GuardAllowsTool tests that allowed tools pass through with empty JSON
func TestBinary_GuardAllowsTool(t *testing.T) {
	binary := buildSkillGuard(t)
	sessionDir := t.TempDir()
	configDir := setupConfigDir(t)

	env := append(os.Environ(),
		"GOGENT_SESSION_DIR="+sessionDir,
		"CLAUDE_CONFIG_DIR="+configDir,
	)

	// Setup: create guard via Skill invocation
	setupStdin := `{"tool_name":"Skill","tool_input":{"skill":"review"}}`
	_, _, err := runSkillGuard(t, binary, setupStdin, env)
	require.NoError(t, err, "setup should succeed")

	// Guard: test allowed tool (Glob IS in review's allowed list)
	guardStdin := `{"tool_name":"Glob","tool_input":{"pattern":"*.go"}}`
	stdout, _, err := runSkillGuard(t, binary, guardStdin, env)
	require.NoError(t, err, "guard mode should execute successfully")

	assert.Equal(t, "{}\n", stdout, "allowed tool should pass through with empty JSON")
}

// TestBinary_NoGuardPassesAll tests that without an active guard, all tools pass through
func TestBinary_NoGuardPassesAll(t *testing.T) {
	binary := buildSkillGuard(t)
	sessionDir := t.TempDir()
	configDir := setupConfigDir(t)

	env := append(os.Environ(),
		"GOGENT_SESSION_DIR="+sessionDir,
		"CLAUDE_CONFIG_DIR="+configDir,
	)

	// No setup - no guard file exists

	// Test various tools
	tools := []string{
		`{"tool_name":"Glob","tool_input":{"pattern":"*.go"}}`,
		`{"tool_name":"Read","tool_input":{"file_path":"/tmp/test.go"}}`,
		`{"tool_name":"Write","tool_input":{"file_path":"/tmp/test.go","content":"test"}}`,
		`{"tool_name":"Task","tool_input":{"model":"haiku","prompt":"test"}}`,
	}

	for _, toolStdin := range tools {
		stdout, _, err := runSkillGuard(t, binary, toolStdin, env)
		require.NoError(t, err, "guard mode should execute successfully")
		assert.Equal(t, "{}\n", stdout, "without active guard, all tools should pass through")
	}
}

// TestBinary_MultipleSkillsLastWins tests that invoking multiple skills overwrites the guard
func TestBinary_MultipleSkillsLastWins(t *testing.T) {
	binary := buildSkillGuard(t)
	sessionDir := t.TempDir()
	configDir := setupConfigDir(t)

	env := append(os.Environ(),
		"GOGENT_SESSION_DIR="+sessionDir,
		"CLAUDE_CONFIG_DIR="+configDir,
	)

	// Setup: create first guard (braintrust - doesn't allow Glob)
	setupStdin1 := `{"tool_name":"Skill","tool_input":{"skill":"braintrust"}}`
	_, _, err := runSkillGuard(t, binary, setupStdin1, env)
	require.NoError(t, err, "first setup should succeed")

	// Verify Glob is blocked
	globStdin := `{"tool_name":"Glob","tool_input":{"pattern":"*.go"}}`
	stdout, _, err := runSkillGuard(t, binary, globStdin, env)
	require.NoError(t, err)
	assert.Contains(t, stdout, "block", "Glob should be blocked with braintrust guard")

	// Setup: create second guard (review - allows Glob)
	setupStdin2 := `{"tool_name":"Skill","tool_input":{"skill":"review"}}`
	_, _, err = runSkillGuard(t, binary, setupStdin2, env)
	require.NoError(t, err, "second setup should succeed")

	// Verify Glob is now allowed
	stdout, _, err = runSkillGuard(t, binary, globStdin, env)
	require.NoError(t, err)
	assert.Equal(t, "{}\n", stdout, "Glob should now be allowed with review guard")

	// Verify guard file reflects latest skill
	guardPath := filepath.Join(sessionDir, "active-skill.json")
	guardData, err := os.ReadFile(guardPath)
	require.NoError(t, err)

	var guard struct {
		Skill string `json:"skill"`
	}
	require.NoError(t, json.Unmarshal(guardData, &guard))
	assert.Equal(t, "review", guard.Skill, "guard should reflect latest skill invocation")
}

// TestBinary_UnknownSkillPassesThrough tests that invoking an unknown skill passes through with no guard
func TestBinary_UnknownSkillPassesThrough(t *testing.T) {
	binary := buildSkillGuard(t)
	sessionDir := t.TempDir()
	configDir := setupConfigDir(t)

	env := append(os.Environ(),
		"GOGENT_SESSION_DIR="+sessionDir,
		"CLAUDE_CONFIG_DIR="+configDir,
	)

	// Setup with unknown skill - binary passes through with {}
	setupStdin := `{"tool_name":"Skill","tool_input":{"skill":"nonexistent"}}`
	stdout, _, err := runSkillGuard(t, binary, setupStdin, env)
	require.NoError(t, err, "binary should execute successfully")
	assert.Equal(t, "{}\n", stdout, "unknown skill should pass through with empty JSON")

	// Verify no guard file created
	guardPath := filepath.Join(sessionDir, "active-skill.json")
	_, err = os.Stat(guardPath)
	assert.True(t, os.IsNotExist(err), "no guard file should be created for unknown skill")
}

// TestBinary_TeamDirNaming tests that team directories use correct naming pattern
func TestBinary_TeamDirNaming(t *testing.T) {
	binary := buildSkillGuard(t)
	sessionDir := t.TempDir()
	configDir := setupConfigDir(t)

	env := append(os.Environ(),
		"GOGENT_SESSION_DIR="+sessionDir,
		"CLAUDE_CONFIG_DIR="+configDir,
	)

	tests := []struct {
		skill          string
		expectedSuffix string
	}{
		{"braintrust", "braintrust"},
		{"review", "code-review"},
		{"implement", "implementation"},
	}

	for _, tc := range tests {
		t.Run(tc.skill, func(t *testing.T) {
			stdin := fmt.Sprintf(`{"tool_name":"Skill","tool_input":{"skill":"%s"}}`, tc.skill)
			_, _, err := runSkillGuard(t, binary, stdin, env)
			require.NoError(t, err)

			// Read guard file
			guardPath := filepath.Join(sessionDir, "active-skill.json")
			guardData, err := os.ReadFile(guardPath)
			require.NoError(t, err)

			var guard struct {
				TeamDir string `json:"team_dir"`
			}
			require.NoError(t, json.Unmarshal(guardData, &guard))

			// Verify naming pattern: {sessionDir}/teams/{timestamp}.{suffix}
			assert.Contains(t, guard.TeamDir, sessionDir, "team dir should be under session dir")
			assert.Contains(t, guard.TeamDir, "/teams/", "team dir should be under teams/ subdirectory")
			assert.Contains(t, guard.TeamDir, "."+tc.expectedSuffix, "team dir should use correct suffix")

			// Verify timestamp format (rough check - should have digits before dot)
			teamDirBase := filepath.Base(guard.TeamDir)
			parts := strings.Split(teamDirBase, ".")
			require.Len(t, parts, 2, "team dir name should be {timestamp}.{suffix}")
			assert.NotEmpty(t, parts[0], "timestamp part should not be empty")
		})
	}
}

// TestBinary_InvalidJSON tests that invalid JSON input passes through gracefully
func TestBinary_InvalidJSON(t *testing.T) {
	binary := buildSkillGuard(t)
	sessionDir := t.TempDir()
	configDir := setupConfigDir(t)

	env := append(os.Environ(),
		"GOGENT_SESSION_DIR="+sessionDir,
		"CLAUDE_CONFIG_DIR="+configDir,
	)

	invalidInputs := []string{
		`{invalid json}`,
		`{"tool_name":"Skill"}`, // Missing tool_input
		``,                       // Empty input
		`not json at all`,
	}

	// Binary design: invalid JSON passes through with {} (lenient, don't block on errors)
	for _, stdin := range invalidInputs {
		stdout, _, err := runSkillGuard(t, binary, stdin, env)
		require.NoError(t, err, "binary should execute successfully even with invalid JSON: %s", stdin)
		assert.Equal(t, "{}\n", stdout, "invalid JSON should pass through with empty JSON: %s", stdin)
	}
}

// TestBinary_MissingSessionDir tests behavior when GOGENT_SESSION_DIR is not set
func TestBinary_MissingSessionDir(t *testing.T) {
	binary := buildSkillGuard(t)
	configDir := setupConfigDir(t)

	// No GOGENT_SESSION_DIR in environment - binary falls back to resolution chain
	env := append([]string{}, "CLAUDE_CONFIG_DIR="+configDir)

	stdin := `{"tool_name":"Glob","tool_input":{"pattern":"*.go"}}`
	stdout, _, err := runSkillGuard(t, binary, stdin, env)

	// Binary falls back to .claude/sessions/unknown - executes successfully
	require.NoError(t, err, "binary should fall back to default session dir")
	assert.Equal(t, "{}\n", stdout, "should pass through with empty JSON (no guard)")
}

// TestBinary_SkillToolAllowedDuringGuard tests that Skill tool itself is always allowed
func TestBinary_SkillToolAllowedDuringGuard(t *testing.T) {
	binary := buildSkillGuard(t)
	sessionDir := t.TempDir()
	configDir := setupConfigDir(t)

	env := append(os.Environ(),
		"GOGENT_SESSION_DIR="+sessionDir,
		"CLAUDE_CONFIG_DIR="+configDir,
	)

	// Setup: create guard
	setupStdin := `{"tool_name":"Skill","tool_input":{"skill":"braintrust"}}`
	_, _, err := runSkillGuard(t, binary, setupStdin, env)
	require.NoError(t, err)

	// Test: Invoke another skill (should be allowed - Skill tool is always in allowed list)
	skillStdin := `{"tool_name":"Skill","tool_input":{"skill":"review"}}`
	stdout, _, err := runSkillGuard(t, binary, skillStdin, env)
	require.NoError(t, err)
	assert.Equal(t, "{}\n", stdout, "Skill tool should always be allowed")

	// Verify guard was updated to new skill
	guardPath := filepath.Join(sessionDir, "active-skill.json")
	guardData, err := os.ReadFile(guardPath)
	require.NoError(t, err)

	var guard struct {
		Skill string `json:"skill"`
	}
	require.NoError(t, json.Unmarshal(guardData, &guard))
	assert.Equal(t, "review", guard.Skill, "guard should be updated to new skill")
}
