---
title: Agent Configuration Quick Reference
type: reference
tags: [agents, configuration]
related: [concepts/agent-spawning]
created: 2026-04-18
---
# Agent Configuration Quick Reference

## Config Struct Fields

```go
type Config struct {
    // Agent Customization Fields (goYoke-117)
    SystemPrompt    string   // Override default system prompt
    AppendPrompt    string   // Append to default system prompt
    AllowedTools    []string // Whitelist of permitted tools
    DisallowedTools []string // Blacklist of forbidden tools
    MaxTurns        int      // Limit on agentic turns (0 = unlimited)
    Model           string   // Model override (haiku|sonnet|opus or full name)
}
```

## Usage Patterns

### 1. Custom System Prompt

```go
cfg := Config{
    SystemPrompt: "You are a security expert specializing in Go",
}
```

**Generates:** `--system-prompt "You are a security expert specializing in Go"`

### 2. Append to Default Prompt

```go
cfg := Config{
    AppendPrompt: "Focus on performance optimization",
}
```

**Generates:** `--append-prompt "Focus on performance optimization"`

**Note:** SystemPrompt and AppendPrompt are mutually exclusive. If both set, SystemPrompt wins.

### 3. Tool Restrictions

```go
cfg := Config{
    AllowedTools:    []string{"Read", "Write", "Edit"},
    DisallowedTools: []string{"Bash", "WebFetch"},
}
```

**Generates:**
```
--allowed-tools Read
--allowed-tools Write
--allowed-tools Edit
--disallowed-tools Bash
--disallowed-tools WebFetch
```

**Tool patterns supported:**
- `"Bash(git *)"` - Allow only git commands in Bash
- `"Read"` - Exact tool name

### 4. Turn Limiting

```go
cfg := Config{
    MaxTurns: 5,  // Limit to 5 agentic turns
}
```

**Generates:** `--max-turns 5`

**Special cases:**
- `MaxTurns: 0` - Omits flag (unlimited)
- `MaxTurns: -1` - Reserved for future use

### 5. Model Override

```go
cfg := Config{
    Model: "sonnet",  // Use alias
}
```

**Generates:** `--model sonnet`

**Supported values:**
- Aliases: `"haiku"`, `"sonnet"`, `"opus"`
- Full names: `"claude-3-sonnet-20240229"`, etc.

## SubagentManager Integration

### Creating an Agent with Overrides

```go
// Define agent
agentCfg := SubagentConfig{
    Name:         "security-reviewer",
    SystemPrompt: "You are a security expert",
    AllowedTools: []string{"Read", "Grep"},
    Model:        "sonnet",
    MaxTurns:     10,
}

// Register and spawn
mgr := NewSubagentManager(baseConfig)
mgr.Register(agentCfg)
proc, _ := mgr.Spawn(ctx, "security-reviewer")
```

### Inheritance Behavior

```go
baseCfg := Config{
    Model:    "haiku",
    MaxTurns: 20,
}

agentCfg := SubagentConfig{
    Model: "sonnet",  // Override base
    // MaxTurns not set, inherits 20 from base
}

procCfg := mgr.buildConfig(agentCfg)
// procCfg.Model = "sonnet"  (overridden)
// procCfg.MaxTurns = 20     (inherited)
```

## Complete Example

```go
// Base configuration for all agents
baseCfg := Config{
    ClaudePath: "claude",
    Verbose:    true,
}

// Specialized security agent
securityAgent := SubagentConfig{
    Name:            "security-agent",
    Description:     "Security code reviewer",
    SystemPrompt:    "You are a security expert. Focus on vulnerabilities.",
    AllowedTools:    []string{"Read", "Grep", "Edit"},
    DisallowedTools: []string{"Bash", "WebFetch", "WebSearch"},
    Model:           "sonnet",
    MaxTurns:        10,
    Tier:            "sonnet",
}

// Create manager and register
mgr := NewSubagentManager(baseCfg)
mgr.Register(securityAgent)

// Spawn agent (buildConfig is called internally)
ctx := context.Background()
proc, err := mgr.Spawn(ctx, "security-agent")
if err != nil {
    log.Fatal(err)
}

// Send query
proc.Send("Review this function for SQL injection")

// Process events
for event := range proc.Events() {
    // Handle events
}
```

## Field Behavior Summary

| Field | Zero Value Behavior | Override Behavior |
|-------|---------------------|-------------------|
| `SystemPrompt` | Empty = inherit/default | Replaces prompt, clears AppendPrompt |
| `AppendPrompt` | Empty = inherit/default | Appends to prompt |
| `AllowedTools` | Empty slice = inherit | Completely replaces base list |
| `DisallowedTools` | Empty slice = inherit | Completely replaces base list |
| `MaxTurns` | 0 = inherit/unlimited | Replaces base value |
| `Model` | Empty = inherit/default | Replaces base model |

## Testing

```bash
# Test new Config fields
go test ./internal/cli -run "TestNewClaudeProcess_(SystemPrompt|Model|Tools)"

# Test buildConfig behavior
go test ./internal/cli -run "TestSubagentManager_BuildConfig"

# Full test suite
go test ./internal/cli -v
```

## References

- Implementation: `docs/goYoke-117-IMPLEMENTATION.md`
- Source: `internal/cli/subprocess.go`, `internal/cli/subagent.go`
- Tests: `internal/cli/subprocess_test.go`, `internal/cli/subagent_test.go`


---

## See Also

- [[concepts/agent-spawning]] — MCP spawn_agent architecture
- [[INSTALL-NEW-AGENT-GUIDE]] — Step-by-step agent creation
- [[ARCHITECTURE#17.2 Agent Definitions]] — Extension point
