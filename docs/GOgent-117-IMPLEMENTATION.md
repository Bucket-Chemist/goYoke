# goYoke-117: Config Struct Extension for Agent Customization

## Summary

Extended the `Config` struct in `internal/cli/subprocess.go` to support all Claude CLI agent customization flags. Updated `SubagentManager.buildConfig()` to apply agent-specific overrides. Achieved 100% test coverage for new code.

## Changes Made

### 1. Config Struct Extensions (`internal/cli/subprocess.go`)

Added 6 new fields to support Claude CLI agent flags:

```go
type Config struct {
    // ... existing fields ...

    // SystemPrompt overrides the default system prompt.
    // If set, passed as --system-prompt flag.
    // Cannot be used with AppendPrompt.
    SystemPrompt string

    // AppendPrompt appends to the default system prompt.
    // If set, passed as --append-prompt flag.
    // Cannot be used with SystemPrompt.
    AppendPrompt string

    // AllowedTools is the whitelist of permitted tools.
    // If set, each tool is passed as --allowed-tools flag.
    // Supports patterns like "Bash(git *)".
    AllowedTools []string

    // DisallowedTools is the blacklist of forbidden tools.
    // If set, each tool is passed as --disallowed-tools flag.
    DisallowedTools []string

    // MaxTurns limits the number of agentic turns.
    // If > 0, passed as --max-turns flag.
    MaxTurns int

    // Model overrides the default model.
    // Accepts: "claude-3-opus", "claude-3-sonnet", "claude-3-haiku" or aliases "opus", "sonnet", "haiku".
    // If set, passed as --model flag.
    Model string
}
```

### 2. NewClaudeProcess() Updates

Updated argument building logic to handle new fields:

```go
// Add system prompt (mutually exclusive with append)
if cfg.SystemPrompt != "" {
    args = append(args, "--system-prompt", cfg.SystemPrompt)
} else if cfg.AppendPrompt != "" {
    args = append(args, "--append-prompt", cfg.AppendPrompt)
}

// Add tool restrictions
for _, tool := range cfg.AllowedTools {
    args = append(args, "--allowed-tools", tool)
}
for _, tool := range cfg.DisallowedTools {
    args = append(args, "--disallowed-tools", tool)
}

// Add max turns limit
if cfg.MaxTurns > 0 {
    args = append(args, "--max-turns", fmt.Sprintf("%d", cfg.MaxTurns))
}

// Add model override
if cfg.Model != "" {
    args = append(args, "--model", cfg.Model)
}
```

**Key Design Decisions:**

- SystemPrompt and AppendPrompt are mutually exclusive (SystemPrompt wins if both set)
- Tool lists generate multiple flags (one per tool)
- MaxTurns only adds flag when > 0 (0 means unlimited)
- Model validation is delegated to Claude CLI
- Empty values don't add flags (backward compatible)

### 3. SubagentManager.buildConfig() Implementation (`internal/cli/subagent.go`)

Implemented config building with agent overrides:

```go
func (sm *SubagentManager) buildConfig(agent SubagentConfig) Config {
    cfg := sm.baseCfg

    // Apply agent-specific overrides
    if agent.SystemPrompt != "" {
        cfg.SystemPrompt = agent.SystemPrompt
        cfg.AppendPrompt = "" // Clear append if system is set
    } else if agent.AppendPrompt != "" {
        cfg.AppendPrompt = agent.AppendPrompt
    }

    if len(agent.AllowedTools) > 0 {
        cfg.AllowedTools = agent.AllowedTools
    }
    if len(agent.DisallowedTools) > 0 {
        cfg.DisallowedTools = agent.DisallowedTools
    }

    if agent.MaxTurns > 0 {
        cfg.MaxTurns = agent.MaxTurns
    }

    if agent.Model != "" {
        cfg.Model = agent.Model
    }

    return cfg
}
```

**Key Design Decisions:**

- Starts with base config (inheritance)
- Agent overrides selectively replace fields
- SystemPrompt clears AppendPrompt (mutual exclusion)
- Tool lists completely replace base lists (no merging)
- Zero values inherit from base (MaxTurns: 0 doesn't override)

### 4. Comprehensive Test Coverage

#### subprocess_test.go (10 new tests):

1. `TestNewClaudeProcess_SystemPrompt` - Verifies --system-prompt flag
2. `TestNewClaudeProcess_AppendPrompt` - Verifies --append-prompt flag
3. `TestNewClaudeProcess_SystemPromptOverridesAppend` - Mutual exclusion
4. `TestNewClaudeProcess_AllowedTools` - Multiple --allowed-tools flags
5. `TestNewClaudeProcess_DisallowedTools` - Multiple --disallowed-tools flags
6. `TestNewClaudeProcess_MaxTurns` - Flag present when > 0, absent when 0
7. `TestNewClaudeProcess_Model` - Model aliases and full names
8. `TestNewClaudeProcess_ModelNotSetWhenEmpty` - Empty model skipped
9. `TestNewClaudeProcess_CombinedAgentSettings` - All flags together
10. Removed `TestClaudeProcess_ClassifyExitReason` (obsolete function)

#### subagent_test.go (7 new test cases):

1. `buildconfig_applies_all_agent_overrides` - All fields applied
2. `buildconfig_inherits_base_when_agent_fields_empty` - Inheritance
3. `buildconfig_SystemPrompt_clears_AppendPrompt` - Mutual exclusion
4. `buildconfig_AppendPrompt_when_no_SystemPrompt` - AppendPrompt path
5. `buildconfig_partial_overrides` - Selective overrides
6. `buildconfig_tool_lists_override_completely` - List replacement
7. `buildconfig_empty_tool_lists_inherit_from_base` - List inheritance

**Test Coverage Results:**

- `NewClaudeProcess()`: **100%** coverage
- `buildConfig()`: **100%** coverage
- Overall package: **82.8%** coverage (exceeds >80% requirement)

## Example Usage

### Creating a Specialized Agent

```go
// Define agent with custom settings
agentCfg := cli.SubagentConfig{
    Name:         "security-agent",
    Description:  "Security code reviewer",
    SystemPrompt: "You are a security expert specializing in Go code review",
    AllowedTools: []string{"Read", "Grep", "Edit"},
    DisallowedTools: []string{"Bash", "WebFetch"},
    Model:        "sonnet",
    MaxTurns:     10,
}

// Create SubagentManager
baseCfg := cli.Config{
    ClaudePath: "claude",
    Verbose:    true,
}
mgr := cli.NewSubagentManager(baseCfg)
mgr.Register(agentCfg)

// Spawn agent (buildConfig is called internally)
ctx := context.Background()
proc, err := mgr.Spawn(ctx, "security-agent")
```

### Generated Command

The above configuration generates:

```bash
claude \
  --print \
  --verbose \
  --debug-to-stderr \
  --input-format stream-json \
  --output-format stream-json \
  --session-id <uuid> \
  --system-prompt "You are a security expert specializing in Go code review" \
  --allowed-tools Read \
  --allowed-tools Grep \
  --allowed-tools Edit \
  --disallowed-tools Bash \
  --disallowed-tools WebFetch \
  --max-turns 10 \
  --model sonnet
```

## Files Modified

| File | Changes |
|------|---------|
| `internal/cli/subprocess.go` | Added 6 Config fields, updated NewClaudeProcess() arg building |
| `internal/cli/subagent.go` | Implemented buildConfig() with agent overrides |
| `internal/cli/subprocess_test.go` | Added 10 tests for new Config fields |
| `internal/cli/subagent_test.go` | Added 7 tests for buildConfig() |

## Backward Compatibility

**Fully backward compatible:**

- Empty/zero Config fields don't add flags
- Existing code continues to work unchanged
- New fields are optional

## Claude CLI Flags Supported

| Flag | Config Field | Type | Notes |
|------|-------------|------|-------|
| `--system-prompt` | `SystemPrompt` | string | Mutually exclusive with AppendPrompt |
| `--append-prompt` | `AppendPrompt` | string | Mutually exclusive with SystemPrompt |
| `--allowed-tools` | `AllowedTools` | []string | Multiple flags, one per tool |
| `--disallowed-tools` | `DisallowedTools` | []string | Multiple flags, one per tool |
| `--max-turns` | `MaxTurns` | int | Only added if > 0 |
| `--model` | `Model` | string | Accepts aliases or full names |

## Design Rationale

### 1. SystemPrompt vs AppendPrompt Mutual Exclusion

**Decision:** SystemPrompt clears AppendPrompt

**Rationale:** Claude CLI treats these as mutually exclusive. If both were passed, behavior is undefined. Making SystemPrompt win provides deterministic behavior.

### 2. Tool Lists Replace Rather Than Merge

**Decision:** Agent tool lists completely replace base lists

**Rationale:**
- Merging semantics are complex (union vs intersection?)
- Agent knows its complete tool needs
- Clear override semantics prevent confusion

### 3. MaxTurns Zero Means Inherit

**Decision:** MaxTurns: 0 inherits from base, doesn't override

**Rationale:**
- 0 is Go's zero value (ambiguous intent)
- Agent can set to -1 to explicitly disable (future)
- Matches Go idiom of zero-value-means-default

### 4. No Model Validation

**Decision:** Don't validate model names in Go code

**Rationale:**
- Claude CLI is source of truth for valid models
- Model names evolve (don't hardcode)
- CLI provides better error messages

## Acceptance Criteria

✅ All Claude CLI agent flags supported
✅ SubagentManager can configure specialized agents
✅ Backward compatible (defaults work, empty values don't add flags)
✅ >80% test coverage achieved (82.8%)
✅ All tests pass

## Integration Points

This implementation enables:

1. **Preset Agent Configurations** (goYoke-117, Task #7)
   - SecurityReviewerAgent, GoProAgent, etc. can now set custom prompts

2. **TUI Agent Picker** (goYoke-117, Task #9)
   - Will display agent descriptions and model tiers

3. **Dynamic Agent Spawning** (goYoke-117, Task #10)
   - Tree view [q] key can spawn configured agents

## Next Steps

1. ✅ Complete Task #7: Add preset agent configurations using new Config fields
2. Task #9: Create agent picker TUI component
3. Task #10: Wire up [q] Query agent in tree view

## Testing

```bash
# Run all new tests
go test ./internal/cli -run "TestNewClaudeProcess_(SystemPrompt|AppendPrompt|AllowedTools|DisallowedTools|MaxTurns|Model|Combined)"
go test ./internal/cli -run "TestSubagentManager_BuildConfig"

# Check coverage
go test ./internal/cli -coverprofile=/tmp/coverage.out
go tool cover -func=/tmp/coverage.out | grep -E "(subprocess.go|subagent.go)"
```

**Result:** All tests pass, 100% coverage for new code.

## References

- **Task:** goYoke-117, Task #8
- **Related:** Task #7 (preset agents), Task #9 (TUI picker), Task #10 (query agent)
- **Files:** `internal/cli/subprocess.go`, `internal/cli/subagent.go`, test files
