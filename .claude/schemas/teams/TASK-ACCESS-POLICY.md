# Task() Access Policy for Team-Spawned Agents

## Overview

Agents spawned by `gogent-team-run` via `claude -p` have **partial Task() access**:
- ✅ Allowed: `Task(model: "haiku")`, `Task(model: "sonnet")`
- ❌ Blocked: `Task(model: "opus")`

This differs from MCP-spawned agents (via `spawn_agent` tool), which have **no Task() access**.

## Enforcement Mechanism

### Nesting Level

The Go binary sets `GOGENT_NESTING_LEVEL=2` when spawning:

```go
cmd := exec.Command("claude", "-p", ...)
cmd.Env = append(os.Environ(),
    "GOGENT_NESTING_LEVEL=2",
    // ...
)
```

### Validation Hook

`gogent-validate` (PreToolUse hook) reads `GOGENT_NESTING_LEVEL` and applies rules.

**Current behavior** (from `cmd/gogent-validate/main.go`):

```go
const MAX_TASK_NESTING_LEVEL = 0 // Only Router (Level 0) can use Task()

nestingLevel := routing.GetNestingLevel()

if nestingLevel > MAX_TASK_NESTING_LEVEL {
    // Block Task() entirely
    response := routing.BlockResponseForNesting(nestingLevel)
    outputJSON(response)
    return
}
```

Level 2 > Level 0 → **ALL Task() calls currently blocked.**

**Required change** (separate ticket): Modify validation to allow Task(haiku/sonnet) at Level 2, while continuing to block Task(opus).

```go
const MAX_OPUS_NESTING_LEVEL = 0 // Only Router can spawn Opus

nestingLevel := routing.GetNestingLevel()

if nestingLevel > MAX_TASK_NESTING_LEVEL {
    if taskInput, err := routing.ParseTaskInput(event.ToolInput); err == nil {
        if taskInput.Model == "opus" && nestingLevel > MAX_OPUS_NESTING_LEVEL {
            response := routing.BlockResponse(
                fmt.Sprintf(
                    "Task(opus) blocked at nesting level %d. Use Task(haiku) or Task(sonnet) instead.",
                    nestingLevel,
                ),
            )
            logNestingBlock(event, nestingLevel, isExplicit)
            outputJSON(response)
            return
        }
    }
}

// Allow Task(haiku/sonnet) at all levels
```

## Spawn Path Comparison

| Spawn Method | Nesting Level | Task() Access | Use Case |
|--------------|---------------|---------------|----------|
| **Router (Level 0)** | 0 | Task(haiku/sonnet/opus) | Initial orchestrator spawn |
| **Team-spawned (Level 2)** | 2 | Task(haiku/sonnet) only | Einstein, Staff-Architect |
| **MCP-spawned (Level 1+)** | 1+ | None (no Task tool) | Mozart's children via spawn_agent |

### Why Different Levels?

- **Router**: Level 0 (root, full access)
- **Team-spawned**: Level 2 (skip Level 1 to distinguish from MCP-spawned)
- **MCP-spawned**: Level 1+ (increments per spawn)

**Alternative considered**: Use Level 1 for team-spawned. **Rejected** because it conflates team-spawned with first-generation MCP agents.

## Rationale for Partial Access

### Why Allow Task(haiku/sonnet)?

1. **Codebase exploration**: Einstein needs `Task(haiku)` to spawn a scout for file discovery
2. **Cost efficiency**: Opus delegating to Haiku saves 90% on mechanical work
3. **Pattern consistency**: MCP agents can use `spawn_agent`; team agents need equivalent

### Why Block Task(opus)?

1. **Budget control**: Opus spawning Opus creates exponential cost risk
2. **Nesting complexity**: Opus → Opus → Opus creates 3-level chains (hard to debug)
3. **Architectural intent**: Opus agents should be terminal nodes (analysis, not coordination)

## Prompt Envelope Template

When building prompt envelopes for team members, include this capability notice:

```markdown
## Your Capabilities

You are spawned via `gogent-team-run` at nesting level 2.

**Available delegation**:
- ✅ `Task(model: "haiku")` - For mechanical tasks (file search, pattern extraction)
- ✅ `Task(model: "sonnet")` - For implementation or focused analysis
- ❌ `Task(model: "opus")` - Blocked by gogent-validate

If you need Opus-level analysis, return a recommendation in your stdout instead.

**MCP Tools**:
You do NOT have access to `spawn_agent` (that's for MCP-spawned agents only).

**Important**: Always specify `model` explicitly in Task() calls. If omitted, the CLI
defaults to the session model, which may be Opus — causing an unintended block.
```

This goes in the common envelope section built by `buildPromptEnvelope()`.

## Verification Tests (Phase 2)

During TC-008 implementation, verify with these test cases:

### Test 1: Einstein can spawn haiku scout

```bash
# Prompt: "Use Task(haiku) to count files in src/"
# Expected: Haiku agent spawns, returns count, Einstein continues
# Assertion: No "blocked by gogent-validate" error
```

### Test 2: Einstein cannot spawn opus

```bash
# Prompt: "Use Task(opus) to analyze architecture"
# Expected: gogent-validate blocks with clear message
# Assertion: Error contains "Task(opus) blocked at nesting level 2"
```

### Test 3: Einstein can spawn sonnet

```bash
# Prompt: "Use Task(sonnet) to refactor this function"
# Expected: Sonnet agent spawns, performs refactor, Einstein receives result
# Assertion: Tool result contains refactored code
```

### Full Integration Test

```bash
# Setup: Create minimal team config with Einstein
cat > /tmp/test-team/config.json <<EOF
{
  "project_root": "/home/user/Documents/GOgent-Fortress",
  "waves": [
    {
      "members": [
        {
          "name": "einstein",
          "agent_type": "einstein",
          "stdin_file": "stdin_einstein.json"
        }
      ]
    }
  ],
  "budget_remaining_usd": 10.0
}
EOF

# Create stdin with Task() test prompt
cat > /tmp/test-team/stdin_einstein.json <<EOF
{
  "task": "Try to spawn both haiku and opus",
  "prompt": "First call Task(model: 'haiku', prompt: 'count to 3'). Then call Task(model: 'opus', prompt: 'count to 5')."
}
EOF

# Run team
gogent-team-run /tmp/test-team

# Expected results:
# 1. Task(haiku) succeeds
# 2. Task(opus) blocked with error message
# 3. config.json shows einstein status = "completed" (not "failed")
```

**Acceptance**: Einstein completes despite Opus block (proves haiku works, opus blocked).

## Edge Cases

### Case 1: Agent Requests Opus via spawn_agent

Team-spawned agents do NOT have `spawn_agent` tool (it's an MCP tool, not CLI tool).

If agent tries:
```javascript
mcp__gofortress__spawn_agent({agent: "beethoven", model: "opus", ...})
```

**Result**: `tool_use` error: "Unknown tool: mcp__gofortress__spawn_agent"

This is correct behavior. Documented in the prompt envelope template above.

### Case 2: Agent Uses Task() with No Model

```javascript
Task({prompt: "Do something"}) // No model specified
```

**Current behavior**: CLI defaults to session model. If Einstein is Opus, this spawns Opus.

**Risk**: Accidental Opus spawn bypasses the budget control intent.

**Mitigation**: Prompt envelope MUST instruct: "Always specify model explicitly in Task() calls." The validation hook should also treat no-model Task() calls at Level 2+ as blocked (future enhancement).

### Case 3: Budget Exhaustion During Task()

Einstein spawns haiku scout → scout exhausts remaining budget → scout errors.

**Handling**: Einstein receives error in tool_result, can proceed with partial information. No special handling needed (graceful degradation).

## Implementation References

| File | Lines | What |
|------|-------|------|
| `cmd/gogent-validate/main.go` | 98-109 | Nesting level validation |
| `cmd/gogent-validate/main.go` | 156-167 | Task input parsing |
| `cmd/gogent-validate/main.go` | 406-427 | Nesting block logging |
| `pkg/routing/validator.go` | — | TaskInput parsing (used in validation) |
| `packages/tui/src/lib/spawnAgent.ts` | ~170 | MCP spawn pattern (for comparison) |

## Feeds Into

- **TC-008** (Go binary): Uses `GOGENT_NESTING_LEVEL=2` when spawning agents
- **TC-013** (orchestrator rewrites): Prompt envelopes reference this policy

## Future Work

A separate ticket is needed to modify `gogent-validate` to allow Task(haiku/sonnet) at Level 2. This ticket documents the **intent**; implementation requires changes to `routing.BlockResponseForNesting()`.

## Cross-References

- Einstein's agent prompt: Should reference this policy
- Braintrust skill prompt: Should mention Einstein can use Task(haiku)
- `agents-index.json`: Agent definitions (source of truth for models)
