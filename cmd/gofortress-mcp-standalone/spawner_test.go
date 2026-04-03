package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	routing "github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// makeAgent creates a minimal *routing.Agent for use in tests.
func makeAgent(model string, allowedTools []string) *routing.Agent {
	ag := &routing.Agent{
		ID:    "test-agent",
		Name:  "Test Agent",
		Model: model,
		Tier:  2,
	}
	if len(allowedTools) > 0 {
		ag.CliFlags = &routing.AgentCliFlags{
			AllowedTools: allowedTools,
		}
	}
	return ag
}

// argsContainsSeq returns true if args contains the subsequence [a, b] with b
// immediately following a.
func argsContainsSeq(args []string, a, b string) bool {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == a && args[i+1] == b {
			return true
		}
	}
	return false
}

// argAfter returns the value immediately following the flag in args, or "".
func argAfter(args []string, flag string) string {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// TestBuildSpawnArgs
// ---------------------------------------------------------------------------

func TestBuildSpawnArgs_RequiredFlags(t *testing.T) {
	agent := makeAgent("sonnet", nil)
	args := buildSpawnArgs(agent, SpawnAgentInput{})

	assert.Contains(t, args, "-p", "must include -p flag")
	assert.True(t, argsContainsSeq(args, "--output-format", "json"), "must include --output-format json (not stream-json)")
	assert.NotContains(t, args, "--no-cache", "must NOT include --no-cache (not a valid CLI flag)")
	assert.NotContains(t, args, "--verbose", "must NOT include --verbose (not needed for json format)")
	// --permission-mode bypassPermissions is required for non-interactive -p mode.
	assert.True(t, argsContainsSeq(args, "--permission-mode", "bypassPermissions"), "must include --permission-mode bypassPermissions")
}

func TestBuildSpawnArgs_DefaultModel(t *testing.T) {
	agent := makeAgent("haiku", nil)
	args := buildSpawnArgs(agent, SpawnAgentInput{})

	model := argAfter(args, "--model")
	assert.Equal(t, "haiku", model, "should use agent.Model when input.Model is empty")
}

func TestBuildSpawnArgs_ModelOverride(t *testing.T) {
	agent := makeAgent("haiku", nil)
	args := buildSpawnArgs(agent, SpawnAgentInput{Model: "opus"})

	model := argAfter(args, "--model")
	assert.Equal(t, "opus", model, "should use input.Model when set")
}

func TestBuildSpawnArgs_AllowedToolsFromAgent(t *testing.T) {
	agent := makeAgent("sonnet", []string{"Read", "Write", "Bash"})
	args := buildSpawnArgs(agent, SpawnAgentInput{})

	toolsVal := argAfter(args, "--allowedTools")
	assert.Equal(t, "Read,Write,Bash", toolsVal, "should use agent.GetAllowedTools() when input.AllowedTools is empty")
}

func TestBuildSpawnArgs_AllowedToolsOverride(t *testing.T) {
	agent := makeAgent("sonnet", []string{"Read"})
	args := buildSpawnArgs(agent, SpawnAgentInput{AllowedTools: []string{"Glob", "Grep"}})

	toolsVal := argAfter(args, "--allowedTools")
	assert.Equal(t, "Glob,Grep", toolsVal, "should use input.AllowedTools when set")
}

func TestBuildSpawnArgs_AllowedToolsFallback(t *testing.T) {
	// Agent with no cli_flags configured falls back to ["Read","Glob","Grep"].
	agent := makeAgent("sonnet", nil)
	args := buildSpawnArgs(agent, SpawnAgentInput{})

	toolsVal := argAfter(args, "--allowedTools")
	assert.Equal(t, "Read,Glob,Grep", toolsVal, "should fall back to default tools when both agent and input are empty")
}

func TestBuildSpawnArgs_MCPConfig_Interactive(t *testing.T) {
	t.Setenv("GOFORTRESS_MCP_CONFIG", "/tmp/mcp-config.json")
	agent := makeAgent("sonnet", []string{"Read", "Write"})
	agent.Interactive = true

	args := buildSpawnArgs(agent, SpawnAgentInput{})

	assert.True(t, argsContainsSeq(args, "--mcp-config", "/tmp/mcp-config.json"),
		"--mcp-config must be present for interactive agents when env var is set")

	toolsVal := argAfter(args, "--allowedTools")
	assert.Contains(t, toolsVal, "mcp__gofortress-interactive__*",
		"interactive MCP tool glob must be merged into allowedTools")
	assert.Contains(t, toolsVal, "Read",
		"original agent tools must be preserved")
}

func TestBuildSpawnArgs_MCPConfig_NonInteractive(t *testing.T) {
	t.Setenv("GOFORTRESS_MCP_CONFIG", "/tmp/mcp-config.json")
	agent := makeAgent("sonnet", []string{"Read", "Write"})
	agent.Interactive = false

	args := buildSpawnArgs(agent, SpawnAgentInput{})

	assert.NotContains(t, args, "--mcp-config",
		"--mcp-config must be absent for non-interactive agents")

	toolsVal := argAfter(args, "--allowedTools")
	assert.NotContains(t, toolsVal, "mcp__gofortress-interactive__*",
		"MCP tool glob must not be added for non-interactive agents")
}

func TestBuildSpawnArgs_MCPConfig_EnvVarAbsent(t *testing.T) {
	t.Setenv("GOFORTRESS_MCP_CONFIG", "")
	agent := makeAgent("sonnet", []string{"Read"})
	agent.Interactive = true

	args := buildSpawnArgs(agent, SpawnAgentInput{})

	assert.NotContains(t, args, "--mcp-config",
		"--mcp-config must be absent when GOFORTRESS_MCP_CONFIG is not set")

	toolsVal := argAfter(args, "--allowedTools")
	assert.NotContains(t, toolsVal, "mcp__gofortress-interactive__*",
		"MCP tool glob must not be added when env var is absent")
}

func TestBuildSpawnArgs_MCPConfig_InteractiveWithAllowedToolsOverride(t *testing.T) {
	// MCP glob must be appended even when caller provides an explicit AllowedTools override.
	t.Setenv("GOFORTRESS_MCP_CONFIG", "/tmp/test-mcp.json")
	agent := makeAgent("sonnet", nil)
	agent.Interactive = true

	args := buildSpawnArgs(agent, SpawnAgentInput{AllowedTools: []string{"Read", "Bash"}})

	assert.True(t, argsContainsSeq(args, "--mcp-config", "/tmp/test-mcp.json"),
		"--mcp-config must be present for interactive agents when env var is set")

	toolsVal := argAfter(args, "--allowedTools")
	assert.Contains(t, toolsVal, "mcp__gofortress-interactive__*",
		"MCP glob must be appended to explicit AllowedTools override")
	assert.Contains(t, toolsVal, "Read",
		"explicit tools must be preserved in override case")
	assert.Contains(t, toolsVal, "Bash",
		"explicit tools must be preserved in override case")
}

func TestBuildSpawnArgs_MCPConfig_NonInteractiveNoEnv(t *testing.T) {
	// Baseline: non-interactive agent with no env var — no --mcp-config, no MCP tools.
	t.Setenv("GOFORTRESS_MCP_CONFIG", "")
	agent := makeAgent("sonnet", []string{"Read", "Write"})
	agent.Interactive = false

	args := buildSpawnArgs(agent, SpawnAgentInput{})

	assert.NotContains(t, args, "--mcp-config",
		"--mcp-config must be absent for non-interactive agents with no env var")

	toolsVal := argAfter(args, "--allowedTools")
	assert.NotContains(t, toolsVal, "mcp__gofortress-interactive__*",
		"MCP tool glob must not be present for non-interactive agents with no env var")
}

func TestBuildSpawnArgs_NoTimeoutFlag(t *testing.T) {
	// --timeout is NOT a valid claude CLI flag. Timeout is managed by
	// runSubprocess() via time.AfterFunc.
	agent := makeAgent("sonnet", nil)
	args := buildSpawnArgs(agent, SpawnAgentInput{Timeout: 60000})
	assert.NotContains(t, args, "--timeout", "must NOT include --timeout (not a valid CLI flag)")
}

func TestBuildSpawnArgs_MaxBudgetPresent(t *testing.T) {
	agent := makeAgent("sonnet", nil)
	args := buildSpawnArgs(agent, SpawnAgentInput{MaxBudget: 0.5})

	budgetVal := argAfter(args, "--max-budget-usd")
	assert.NotEmpty(t, budgetVal, "--max-budget-usd must be present when input.MaxBudget > 0")
	assert.Equal(t, "0.5000", budgetVal)
}

func TestBuildSpawnArgs_MaxBudgetAbsent(t *testing.T) {
	agent := makeAgent("sonnet", nil)
	args := buildSpawnArgs(agent, SpawnAgentInput{MaxBudget: 0})

	assert.NotContains(t, args, "--max-budget-usd", "--max-budget-usd must be absent when input.MaxBudget is 0")
}

// ---------------------------------------------------------------------------
// TestParseCLIOutput
// ---------------------------------------------------------------------------

func TestParseCLIOutput_StreamJSON(t *testing.T) {
	input := `{"type":"system","subtype":"init","session_id":"abc123"}
{"type":"assistant","message":"thinking..."}
{"type":"result","result":"Hello world","total_cost_usd":0.042,"num_turns":3,"is_error":false,"session_id":"abc123"}
`
	res, err := parseCLIOutput([]byte(input))

	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, "Hello world", res.Result)
	assert.InDelta(t, 0.042, res.TotalCostUSD, 0.0001)
	assert.Equal(t, 3, res.NumTurns)
	assert.False(t, res.IsError)
	assert.Equal(t, "abc123", res.SessionID)
}

func TestParseCLIOutput_NoResult(t *testing.T) {
	input := `{"type":"system","subtype":"init","session_id":"abc123"}
{"type":"assistant","message":"thinking..."}
`
	_, err := parseCLIOutput([]byte(input))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no result entry found")
}

func TestParseCLIOutput_MalformedLine(t *testing.T) {
	// Invalid JSON lines are skipped; the valid result line must still be found.
	input := `{"type":"system","subtype":"init"}
NOT VALID JSON AT ALL {{{
{"type":"result","result":"output","total_cost_usd":0.01,"num_turns":1}
`
	res, err := parseCLIOutput([]byte(input))

	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, "output", res.Result)
	assert.InDelta(t, 0.01, res.TotalCostUSD, 0.0001)
}

func TestParseCLIOutput_LegacyJSONArray(t *testing.T) {
	input := `[{"type":"system","subtype":"init"},{"type":"result","result":"legacy output","total_cost_usd":0.005,"num_turns":2}]`

	res, err := parseCLIOutput([]byte(input))

	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, "legacy output", res.Result)
	assert.InDelta(t, 0.005, res.TotalCostUSD, 0.0001)
	assert.Equal(t, 2, res.NumTurns)
}

func TestParseCLIOutput_EmptyOutput(t *testing.T) {
	_, err := parseCLIOutput([]byte(""))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestParseCLIOutput_WhitespaceOnly(t *testing.T) {
	_, err := parseCLIOutput([]byte("   \n  "))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

// ---------------------------------------------------------------------------
// TestValidateNestingDepth
// ---------------------------------------------------------------------------

func TestValidateNestingDepth_OK(t *testing.T) {
	tests := []struct {
		name  string
		level string
	}{
		{"level 0", "0"},
		{"level 5", "5"},
		{"level 9 (max-1)", "9"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GOGENT_NESTING_LEVEL", tc.level)
			assert.NoError(t, validateNestingDepth())
		})
	}
}

func TestValidateNestingDepth_Exceeded(t *testing.T) {
	tests := []struct {
		name  string
		level string
	}{
		{"level 10 (max)", "10"},
		{"level 11 (above max)", "11"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GOGENT_NESTING_LEVEL", tc.level)
			err := validateNestingDepth()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "exceeded")
		})
	}
}

func TestValidateNestingDepth_Absent(t *testing.T) {
	t.Setenv("GOGENT_NESTING_LEVEL", "")
	assert.NoError(t, validateNestingDepth())
}

func TestValidateNestingDepth_Invalid(t *testing.T) {
	// Non-numeric is treated as level 0 — no error.
	t.Setenv("GOGENT_NESTING_LEVEL", "not-a-number")
	assert.NoError(t, validateNestingDepth())
}

// ---------------------------------------------------------------------------
// TestGetCurrentNestingLevel
// ---------------------------------------------------------------------------

func TestGetCurrentNestingLevel(t *testing.T) {
	tests := []struct {
		name     string
		envVal   string
		expected int
	}{
		{"absent (empty string)", "", 0},
		{"valid integer", "5", 5},
		{"non-numeric", "abc", 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GOGENT_NESTING_LEVEL", tc.envVal)
			assert.Equal(t, tc.expected, getCurrentNestingLevel())
		})
	}
}

// ---------------------------------------------------------------------------
// TestBuildSpawnEnv
// ---------------------------------------------------------------------------

func TestBuildSpawnEnv_NestingIncrement(t *testing.T) {
	env := buildSpawnEnv(3, "agent-uuid-123")

	nestingVal := envValue(env, "GOGENT_NESTING_LEVEL")
	require.NotEmpty(t, nestingVal, "GOGENT_NESTING_LEVEL must be set")
	assert.Equal(t, "4", nestingVal, "GOGENT_NESTING_LEVEL must be nestingLevel+1")
}

func TestBuildSpawnEnv_ParentAgent(t *testing.T) {
	env := buildSpawnEnv(0, "my-agent-id")

	assert.Equal(t, "my-agent-id", envValue(env, "GOGENT_PARENT_AGENT"))
}

func TestBuildSpawnEnv_SpawnMethod(t *testing.T) {
	env := buildSpawnEnv(0, "agent-id")

	assert.Equal(t, "mcp-cli", envValue(env, "GOGENT_SPAWN_METHOD"))
}

func TestBuildSpawnEnv_FiltersClaudeCode(t *testing.T) {
	t.Setenv("CLAUDECODE", "1")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "/usr/local/bin/claude")

	env := buildSpawnEnv(0, "agent-id")

	for _, e := range env {
		assert.False(t, strings.HasPrefix(e, "CLAUDECODE="),
			"CLAUDECODE must be filtered from subprocess env")
		assert.False(t, strings.HasPrefix(e, "CLAUDE_CODE_ENTRYPOINT="),
			"CLAUDE_CODE_ENTRYPOINT must be filtered from subprocess env")
	}
}

// ---------------------------------------------------------------------------
// TestFilterEnv
// ---------------------------------------------------------------------------

func TestFilterEnv_RemovesMatchingKeys(t *testing.T) {
	environ := []string{
		"KEEP_ME=value",
		"REMOVE_ME=secret",
		"ALSO_KEEP=123",
		"REMOVE_ME_TOO=gone",
	}

	result := filterEnv(environ, "REMOVE_ME", "REMOVE_ME_TOO")

	assert.True(t, slices.Contains(result, "KEEP_ME=value"), "KEEP_ME must be preserved")
	assert.True(t, slices.Contains(result, "ALSO_KEEP=123"), "ALSO_KEEP must be preserved")
	assert.False(t, slices.Contains(result, "REMOVE_ME=secret"), "REMOVE_ME must be filtered")
	assert.False(t, slices.Contains(result, "REMOVE_ME_TOO=gone"), "REMOVE_ME_TOO must be filtered")
}

func TestFilterEnv_PreservesNonMatchingKeys(t *testing.T) {
	environ := []string{
		"A=1",
		"B=2",
		"C=3",
	}

	result := filterEnv(environ, "D", "E")

	assert.Len(t, result, 3, "no entries should be removed when no keys match")
}

func TestFilterEnv_PrefixMatchOnly(t *testing.T) {
	// "FOO_BAR" must not be removed when filtering "FOO".
	environ := []string{
		"FOO=remove-this",
		"FOO_BAR=keep-this",
	}

	result := filterEnv(environ, "FOO")

	assert.False(t, slices.Contains(result, "FOO=remove-this"), "FOO= must be removed")
	assert.True(t, slices.Contains(result, "FOO_BAR=keep-this"), "FOO_BAR= must be preserved")
}

// ---------------------------------------------------------------------------
// Helper: envValue
// ---------------------------------------------------------------------------

// envValue returns the value of a key in an env slice (KEY=VALUE format), or "".
func envValue(env []string, key string) string {
	prefix := key + "="
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			return e[len(prefix):]
		}
	}
	return ""
}
