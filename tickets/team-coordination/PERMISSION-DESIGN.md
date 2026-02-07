# Permission Handling for Team-Spawned Agents

## Core Decision

**Primary mechanism**: `--allowedTools` flag (comma-separated list)
**Fallback**: `--permission-mode delegate` (belt-and-suspenders, do not rely on it)

## Evidence

From `docs/PERMISSION_HANDLING.md`:
- `--permission-mode delegate` does **NOT** work in pipe mode (`-p` flag)
- Permission events in pipe mode are **DENIALS**, not **REQUESTS**
- No response protocol exists to approve denied tools
- `--allowedTools` is the intended mechanism for automation

From `spawnAgent.ts:buildCliArgs()` (lines 323-346):
- Line 335: `--permission-mode delegate` included
- Lines 337-339: `--allowedTools` is the working implementation
- Line 351-371: `parseCliOutput()` handles cost extraction

## Per-Agent Tool Lists

### Team Pattern (CLI Mode via Go Binary)

**Critical:** When agents are spawned by `gogent-team-run` Go binary in CLI mode (`claude -p --output-format json`):
- **Input**: JSON via stdin
- **Output**: JSON via stdout
- **File I/O**: Go binary handles all file writes via native `os.WriteFile()` based on agent JSON output
- **No Write tool needed**: Agents output to stdout, not files

| Agent Type | Allowed Tools | Rationale |
|------------|---------------|-----------|
| **Analysis agents** (einstein, staff-architect-critical-review, beethoven) | `Read`, `Glob`, `Grep` | Analysis only, JSON output to stdout |
| **Review agents** (backend-reviewer, frontend-reviewer, standards-reviewer, architect-reviewer) | `Read`, `Glob`, `Grep` | Code review, JSON output to stdout |
| **Review orchestrator** (review-orchestrator) | `Read`, `Glob`, `Grep` | File classification, JSON output to stdout |
| **Implementation agents** (worker agents in impl workflow) | `Read`, `Glob`, `Grep`, `Bash` | Implementation with JSON output (Go binary writes files) |

### MCP Pattern (Current, Foreground Spawning)

**For agents spawned via MCP `spawn_agent` tool** (not through Go binary):
- stdout is captured by the MCP tool and returned to the caller
- Same principle: agents output to stdout, caller handles file I/O
- Only implementation agents need Write/Edit (they modify code directly in foreground mode)

| Agent Type | Allowed Tools | Rationale |
|------------|---------------|-----------|
| **Orchestrators** (mozart, review-orchestrator) | `Read`, `Glob`, `Grep` | Coordination, output captured via stdout by MCP spawn |
| **Implementation agents** (go-pro, python-pro, etc.) | `Read`, `Write`, `Edit`, `Glob`, `Grep`, `Bash` | Full implementation cycle (foreground = direct file access) |

## Backward Compatibility

The `allowed_tools` field in agents-index.json is **metadata only** — it does not break existing spawn paths.

| Spawn Path | Reads `allowed_tools` from config? | Who controls tools? | Impact |
|---|---|---|---|
| **Task()** (Claude Code CLI) | No | Parent session grants full access | **No change** — agents keep current behavior |
| **MCP spawn_agent** (TUI) | No (only reads `effortLevel`) | Caller passes `allowedTools` param optionally | **No change** — field is ignored |
| **gogent-team-run** (Go binary, TC-008) | **Yes** | Config-driven, with conservative fallback | **New consumer** — only path that reads it |

**Future enhancement (not in scope for TC-001/TC-008):**
MCP `spawn_agent` could optionally read `allowed_tools` from config as a default when the caller doesn't pass `allowedTools` explicitly. This would bring parity between spawn paths but is a separate ticket.

## CLI Invocation Pattern

```bash
# Correct pattern (copied from spawnAgent.ts)
claude -p --output-format json \
  --model opus \
  --permission-mode delegate \
  --allowedTools Read,Write,Glob,Grep,Bash,Edit \
  --max-budget-usd 5.0
```

**Key details**:
- Flag order: `--permission-mode delegate` **BEFORE** `--allowedTools` (matches spawnAgent.ts)
- Tool names: Exact case-sensitive match: `Read`, `Write`, `Edit`, `Bash`, `Glob`, `Grep`
- Comma-separated: **No spaces**: `Read,Write,Glob` (not `Read, Write, Glob`)
- Pipe mode: `-p` flag is mandatory for automation

## Integration with agents-index.json

TC-014 will add `allowed_tools` field to agents-index.json:

```json
{
  "agents": [
    {
      "id": "einstein",
      "allowed_tools": ["Read", "Glob", "Grep"]
    },
    {
      "id": "go-pro",
      "allowed_tools": ["Read", "Write", "Edit", "Glob", "Grep", "Bash"]
    }
  ]
}
```

The Go binary will read this field and construct `--allowedTools` accordingly.

## Conservative Fallback

If `allowed_tools` field is missing from agents-index.json, use conservative default:

```json
["Read", "Glob", "Grep"]
```

**Rationale**: Read-only tools are safe for all agent types. Better to require explicit opt-in for Write/Bash than to accidentally grant dangerous permissions.

## Manual Verification Test

Before Phase 2 implementation, verify the flag pattern works:

### Test 1: Verify Write Tool Works in Pipe Mode

```bash
echo "Create /tmp/test-gogent.txt with content 'hello'" | \
  claude -p --output-format json \
  --permission-mode delegate \
  --allowedTools Write
```

**Expected**: File created successfully, no permission denial
**Actual result**: PASS. File created at `/tmp/test-gogent.txt` with content `hello`. `permission_denials: []`.

### Test 2: Verify Read Tool Works in Pipe Mode

```bash
echo "Read /etc/hostname" | \
  claude -p --output-format json \
  --permission-mode delegate \
  --allowedTools Read
```

**Expected**: Hostname returned in JSON output
**Actual result**: PASS. Returned `doktersmol-framework`. `permission_denials: []`.

### Observation: Tool Visibility

The init event's `tools` array lists ALL available tools regardless of `--allowedTools` value.
`--allowedTools` pre-approves listed tools but does **not** hide others from the agent's awareness.
Non-approved tools would trigger permission denials if the agent attempts to use them.

**Quality Gate for Phase 2**: Both tests pass. Pattern is confirmed working.

## Error Handling

If `allowed_tools` field is missing from agents-index.json:
1. Log warning: "Agent {id} missing allowed_tools field, using conservative default: [Read, Glob, Grep]"
2. Use conservative default
3. Continue execution (non-fatal)

If `allowed_tools` field is empty array `[]`:
1. Omit `--allowedTools` flag entirely (no tools allowed)
2. This is a valid edge case: deliberate lockdown for pure reasoning agents

## Per-Workflow Tool Lists (Team Pattern)

All agents output JSON to stdout. Go binary handles file writes.

### Braintrust Workflow

| Agent | Allowed Tools | Rationale |
|-------|---------------|-----------|
| mozart (orchestrator) | `Read`, `Glob`, `Grep` | Coordination, JSON output to stdout |
| einstein (theoretical analysis) | `Read`, `Glob`, `Grep` | Analysis, JSON output to stdout |
| staff-architect-critical-review | `Read`, `Glob`, `Grep` | Critical review, JSON output to stdout |
| beethoven (synthesis) | `Read`, `Glob`, `Grep` | Synthesis, JSON output to stdout |

### Review Workflow

| Agent | Allowed Tools | Rationale |
|-------|---------------|-----------|
| review-orchestrator | `Read`, `Glob`, `Grep` | File classification, JSON output to stdout |
| backend-reviewer | `Read`, `Glob`, `Grep` | Backend review, JSON output to stdout |
| frontend-reviewer | `Read`, `Glob`, `Grep` | Frontend review, JSON output to stdout |
| standards-reviewer | `Read`, `Glob`, `Grep` | Standards review, JSON output to stdout |
| architect-reviewer | `Read`, `Glob`, `Grep` | Architecture review, JSON output to stdout |

### Implementation Workflow

| Agent | Allowed Tools | Rationale |
|-------|---------------|-----------|
| architect | `Read`, `Glob`, `Grep` | Planning, JSON output to stdout |
| go-pro, python-pro, r-pro, typescript-pro, react-pro | `Read`, `Glob`, `Grep`, `Bash` | Need Bash for build/test verification; file writes via JSON output |
| code-reviewer | `Read`, `Glob`, `Grep` | Post-implementation review, JSON output to stdout |

**Key insight**: `Write`/`Edit` are never needed in team pattern. The Go binary parses JSON stdout and calls `os.WriteFile()` itself. Only `Bash` is granted beyond read-only, and only to implementation agents that need to run builds/tests.

## Reference Implementation

**File**: `packages/tui/src/mcp/tools/spawnAgent.ts`
**Function**: `buildCliArgs()` (lines 323-346)

```typescript
export function buildCliArgs(args: {
  model?: string;
  allowedTools?: string[];
  maxBudget?: number;
}): string[] {
  const cliArgs = ["-p", "--output-format", "json"];

  if (args.model) {
    cliArgs.push("--model", args.model);
  }

  // Use delegate mode instead of dangerously-skip-permissions
  cliArgs.push("--permission-mode", "delegate");

  if (args.allowedTools && args.allowedTools.length > 0) {
    cliArgs.push("--allowedTools", args.allowedTools.join(","));
  }

  if (args.maxBudget) {
    cliArgs.push("--max-budget-usd", String(args.maxBudget));
  }

  return cliArgs;
}
```

**Key observations**:
- `--permission-mode delegate` is always included (line 335)
- `--allowedTools` is only added if array is non-empty (line 337)
- Tools are joined with comma, no spaces (line 338)
- Order: permission-mode before allowedTools

## Go Binary Implementation Pattern

For TC-008, the Go binary should follow this pattern:

```go
// Read agent config from agents-index.json
agent := getAgentConfig(agentID)

// Get allowed tools (with fallback)
allowedTools := agent.AllowedTools
if len(allowedTools) == 0 {
    log.Warnf("Agent %s missing allowed_tools, using conservative default", agentID)
    allowedTools = []string{"Read", "Glob", "Grep"}
}

// Build CLI args
args := []string{"-p", "--output-format", "json"}

if agent.Model != "" {
    args = append(args, "--model", agent.Model)
}

// Always include permission-mode (belt-and-suspenders)
args = append(args, "--permission-mode", "delegate")

// Add allowedTools if non-empty
if len(allowedTools) > 0 {
    toolsStr := strings.Join(allowedTools, ",")
    args = append(args, "--allowedTools", toolsStr)
}

if maxBudget > 0 {
    args = append(args, "--max-budget-usd", fmt.Sprintf("%.2f", maxBudget))
}
```

## Quality Gates for TC-008

Before TC-008 implementation is accepted:

1. **Manual Test 1 passes**: Write tool works in pipe mode
2. **Manual Test 2 passes**: Read tool works in pipe mode
3. **Unit test exists**: `buildCLIArgs()` function tested with allowed_tools from config
4. **Unit test exists**: Conservative fallback used when allowed_tools missing
5. **Integration test exists**: Spawned agent can successfully use Write tool
6. **Integration test exists**: Spawned agent denied Bash when not in allowed_tools

## Security Considerations

**Tools requiring caution**:
- `Bash`: Arbitrary command execution, potential for system damage
- `Write`: File creation/overwrite, potential for data loss
- `Edit`: File modification, potential for code corruption

**Recommended approach**:
1. Analysis agents: Read-only (`Read`, `Glob`, `Grep`)
2. Review agents: Read-only
3. Implementation agents: Full access, but only in controlled workflows
4. User confirmation: Consider prompting before granting Bash to new agent types

**Audit trail**: TC-008 should log every agent spawn with:
- Agent ID
- Allowed tools granted
- Timestamp
- Workflow context

This enables post-hoc security review of tool usage.
