# Routing Enforcement Architecture (GAP-002)

## Overview

Claude Code uses a three-layer routing enforcement system to guarantee correct agent execution and prevent silent failures. This document explains the architecture, its evolution, and why it matters.

## The Problem: Pre-GAP-002

Before January 2026, agent routing relied on documentation:

```
Developer reads CLAUDE.md
     ↓
Developer writes Task() call with subagent_type
     ↓
If subagent_type is wrong → Silent failure (wrong tool permissions)
     ↓
Hours of debugging "why doesn't my Task() work?"
```

### Cost of Silent Failures

- **Token waste**: Doomed execution attempts cost $0.01-$0.20 each
- **Time waste**: Debugging mysterious failures
- **Inconsistency**: Same mistake in different sessions
- **No audit trail**: Impossible to detect patterns

## The Solution: Three-Layer Enforcement

GAP-002 implemented programmatic enforcement across three layers:

### Layer 1: Data Schema (`routing-schema.json`)

The source of truth for all routing decisions.

**File**: `/home/doktersmol/.claude/routing-schema.json`

**Key section**: `agent_subagent_mapping`

```json
{
  "agent_subagent_mapping": {
    "codebase-search": "Explore",
    "tech-docs-writer": "general-purpose",
    "orchestrator": "Plan",
    ...
  }
}
```

**What it defines**:
- Each agent's required `subagent_type`
- Tool permissions for each subagent_type (Explore, general-purpose, Plan, Bash)
- Which agents can write, read-only, run bash, etc.

### Layer 2: Agent Configuration (`agent.yaml`)

Each agent has a manifest defining its:
- Model (Haiku, Sonnet, Opus)
- Tools it needs
- Tier classification
- Sharp edges and failure patterns

**Location**: `~/.claude/agents/[agent-name]/agent.yaml`

**What it validates**:
- Agent exists
- Agent's declared tools match schema
- Model tier aligns with task complexity

### Layer 3: Pre-Execution Hook (`validate-routing.sh`)

The enforcement engine that runs before every Task() call.

**Location**: `~/.claude/hooks/validate-routing.sh`

**What it does**:

1. **Intercepts** every Task() call
2. **Extracts** agent name and subagent_type
3. **Looks up** required subagent_type in schema
4. **Compares** provided vs required
5. **Either**:
   - ✅ **ALLOW** - Calls match, execution proceeds
   - ❌ **BLOCK** - Calls mismatch, Task() fails immediately with clear error
6. **Logs** all violations to `/tmp/claude-routing-violations.jsonl`

## Architecture Diagram

```
Task() Call
    ↓
[validate-routing.sh Hook]
    ↓
    ├─ Extract: agent_name, subagent_type_provided
    ├─ Query: routing-schema.json for subagent_type_required
    ├─ Lookup: agent.yaml for constraints
    ↓
[Decision Point]
    ├─ Match? YES → [ALLOW]
    │                ↓
    │           Execute Task()
    │                ↓
    │           Success/Failure
    │
    └─ Match? NO → [BLOCK]
                    ↓
              Log violation to /tmp/claude-routing-violations.jsonl
                    ↓
              Return error with suggestion
                    ↓
              Task() fails with actionable message
```

## Subagent Types and Tool Permissions

### Explore (Read-Only)
**For**: codebase-search, haiku-scout, code-reviewer, librarian
**Tools**: Read, Glob, Grep, Bash (output only)
**Rationale**: Reconnaissance work should never modify files
**Example**: Finding files with problematic patterns

### general-purpose (Full Write)
**For**: scaffolder, tech-docs-writer, python-pro, r-pro, memory-archivist
**Tools**: All (respects agent.yaml)
**Rationale**: Implementation and documentation agents need write permissions
**Example**: Creating new files, editing existing code

### Plan (Coordination)
**For**: orchestrator, architect
**Tools**: Read, Glob, Grep, Write, Task, AskUserQuestion
**Rationale**: Planning agents need to write plans and spawn other agents
**Example**: Creating implementation roadmaps, interviewing users

### Bash (External Processes)
**Tools**: Bash, Read
**Rationale**: External context engines use shell piping, not file modification
**Example**: Analyzing large codebases via piped commands

## Before and After: Cost Impact

### Before GAP-002 (Documentation-Based)

```
Developer writes wrong subagent_type
    ↓
Task() call succeeds (no validation)
    ↓
Agent receives wrong tool permissions
    ↓
Execution fails mysteriously
    ↓
Developer debugs for 30 minutes
    ↓
Cost: $0.02 (failed execution) + human time
```

### After GAP-002 (Programmatic)

```
Developer writes wrong subagent_type
    ↓
validate-routing.sh catches error immediately
    ↓
Task() fails with clear error message:
   "tech-docs-writer requires subagent_type 'general-purpose',
    not 'Explore'"
    ↓
Developer fixes in 30 seconds
    ↓
Cost: $0.00 (prevented execution) + 30 seconds
```

**Savings**: 99% of debugging time, zero token waste

## Reliability Improvements

### Silent Failure Prevention

Before: Wrong subagent_type → mysterious errors
After: Wrong subagent_type → immediate, actionable error

### Audit Trail

All violations logged to `/tmp/claude-routing-violations.jsonl`:

```json
{
  "timestamp": "2026-01-13T10:22:45Z",
  "session_id": "sess_abc123xyz",
  "agent": "tech-docs-writer",
  "subagent_type_provided": "Explore",
  "subagent_type_required": "general-purpose",
  "reason": "Agent requires write permissions for documentation task",
  "suggestion": "Change subagent_type to 'general-purpose' in Task() call",
  "resolution": "Fixed in iteration 2"
}
```

Use this to identify patterns and improve guidance.

### Consistency Across Sessions

Because enforcement is programmatic (not documentation-based), every session enforces the same rules. No drift, no "but it worked last time" surprises.

## Implementation Details

### How validate-routing.sh Works

```bash
#!/bin/bash
# Simplified pseudocode

TASK_CALL="$1"  # Task({ agent: "tech-docs-writer", subagent_type: "Explore", ... })

# Extract from Task call
AGENT=$(echo "$TASK_CALL" | grep -oP 'agent:\s*"\K[^"]+')
PROVIDED=$(echo "$TASK_CALL" | grep -oP 'subagent_type:\s*"\K[^"]+')

# Look up required subagent_type
REQUIRED=$(jq ".agent_subagent_mapping[\"$AGENT\"]" ~/.claude/routing-schema.json)

# Compare
if [ "$PROVIDED" == "$REQUIRED" ]; then
  # ALLOW - execute
  exit 0
else
  # BLOCK - log and fail
  log_violation "$AGENT" "$PROVIDED" "$REQUIRED"
  exit 1
fi
```

### Error Message Format

When validation fails, users see:

```
ERROR [validate-routing] INVALID subagent_type for 'tech-docs-writer'
  Provided:  Explore
  Required:  general-purpose
  Reason:    Agent requires write permissions for documentation tasks
  Fix:       Use subagent_type: 'general-purpose' in Task() call

  See: ~/.claude/docs/agent-reference-table.md for all agent mappings
       ~/.claude/docs/hooks/validate-routing.md for debugging guide
```

## Verification and Debugging

### Quick Lookup

Check an agent's required subagent_type:

```bash
jq '.agent_subagent_mapping["tech-docs-writer"]' ~/.claude/routing-schema.json
# Output: "general-purpose"
```

### View All Mappings

```bash
jq '.agent_subagent_mapping' ~/.claude/routing-schema.json
```

### Check Violation Log

```bash
tail -f /tmp/claude-routing-violations.jsonl
```

### Query Specific Agent

```bash
grep '"agent": "orchestrator"' /tmp/claude-routing-violations.jsonl
```

## Migration Guide (For Legacy Code)

If you have old Task() calls with wrong subagent_types:

1. **Identify**: Run them, note the agent name
2. **Look up**: Check `routing-schema.json` for correct subagent_type
3. **Fix**: Update the subagent_type in the Task() call
4. **Test**: Re-run, verify no validation errors

Example migration:

```javascript
// BEFORE (will be blocked)
Task({
  subagent_type: "Explore",
  prompt: "AGENT: tech-docs-writer\n..."
})

// AFTER (will be allowed)
Task({
  subagent_type: "general-purpose",
  prompt: "AGENT: tech-docs-writer\n..."
})
```

## Related Documentation

- **Agent Reference Table**: `/home/doktersmol/.claude/docs/agent-reference-table.md`
- **Hook Documentation**: `/home/doktersmol/.claude/docs/hooks/validate-routing.md`
- **Routing Schema**: `/home/doktersmol/.claude/routing-schema.json`
- **Main Guide**: `/home/doktersmol/.claude/CLAUDE.md` (Task() Invocation Pattern section)

## FAQ

**Q: Why is subagent_type different from agent name?**
A: Different agents have different tool needs. Some need read-only access (codebase-search), others need write (tech-docs-writer). The subagent_type tells the system what permissions to grant.

**Q: What happens if I use the wrong subagent_type?**
A: Task() fails immediately with a clear error message. No wasted tokens, no mysterious failures.

**Q: Can I override the validation?**
A: Not recommended. The validation exists to catch mistakes before expensive execution. If you truly need to override, use the `--force-tier=` flag (audited at `/tmp/claude-routing-violations.jsonl`).

**Q: How often is the schema updated?**
A: When new agents are added or tool permissions change. Check the `updated` field in `routing-schema.json`.

**Q: Where do I report a routing bug?**
A: Check `/tmp/claude-routing-violations.jsonl` for violations, then file an issue with:
- The Task() call you attempted
- The error message
- What you expected to happen
