# validate-routing.sh Hook Documentation

## Overview

The `validate-routing.sh` hook is the enforcement engine that prevents subagent_type mismatches before execution. It runs automatically on every Task() call.

**Location**: `~/.claude/hooks/validate-routing.sh`

**Purpose**: Catch routing errors immediately instead of wasting tokens on doomed executions.

**Status**: Active (enforced on all Task() calls)

## Core Principle: Task() is Always Allowed (GAP-006)

**CRITICAL**: Task() is DELEGATION, not a direct tool. Session tier restrictions do NOT apply to Task() calls.

### The Rule

```
Task() is always allowed regardless of session tier.
Tier restrictions apply to SPAWNED agents, not to the ACT of spawning.
```

### Why This Matters

Before GAP-006 fix, the hook treated Task() like any other tool:

```
Scout recommends "external" tier (use Gemini, Bash only)
  ↓
Hook blocks Task() because "Task not in external.tools"
  ↓
/explore workflow BROKEN - can't spawn architect
```

After GAP-006 fix:

```
Scout recommends "external" tier (use Gemini, Bash only for DIRECT work)
  ↓
Hook ALLOWS Task() (delegation is unrestricted)
  ↓
Architect spawns at SONNET tier (its own tier)
  ↓
/explore workflow SUCCEEDS
```

### Session Tier vs Delegation

| Session Tier | What It Controls | Task() Allowed? |
|--------------|------------------|-----------------|
| haiku | Direct tools: Read, Glob, Grep, Bash | ✅ YES |
| haiku_thinking | Direct tools: + Write, Edit, WebFetch | ✅ YES |
| sonnet | Direct tools: All standard tools | ✅ YES |
| external | Direct tools: Bash ONLY | ✅ YES |

**The tier indicates what MODEL to use for direct analysis work. It does NOT restrict which agents you can spawn.**

### Analogy

Think of tiers like job roles:
- **Session tier** = "What level of work can YOU do directly?"
- **Task delegation** = "Who can you ask to help?"

A junior dev (haiku tier) can't write complex code directly, but CAN ask a senior dev (sonnet tier) to do it. The old hook blocked the junior from even asking.

### Implementation

```bash
# validate-routing.sh:117-157 (GAP-006 fix)
if [[ "$tool_name" == "Task" ]]; then
    : # Fall through - Task is delegation, always allowed
else
    # Apply tier restrictions for direct tools (Read, Write, etc)
    allowed_tools=$(echo "$schema" | jq -r '.tiers[$tier].tools')
    # ... block if tool not in allowed_tools
fi
```

## How It Works

### Execution Flow

Every Task() call goes through this validation:

```
1. User writes Task() call
     ↓
2. Hook intercepts before execution
     ↓
3. Hook extracts: agent_name, subagent_type_provided
     ↓
4. Hook queries: routing-schema.json for subagent_type_required
     ↓
5. Hook compares: provided vs required
     ↓
6. Decision:
     ├─ MATCH → Execute Task()
     ├─ MISMATCH → Block and log violation
```

### Example: Success Case

```javascript
Task({
  description: "Update documentation",
  subagent_type: "general-purpose",  // ← Correct
  model: "haiku",
  prompt: "AGENT: tech-docs-writer\n..."
})
```

**Validation result**: ✅ ALLOW

- Hook extracts: agent = "tech-docs-writer", provided = "general-purpose"
- Hook looks up: required = "general-purpose"
- Hook compares: "general-purpose" == "general-purpose"
- Result: Match found, Task() executes

### Example: Failure Case

```javascript
Task({
  description: "Find files",
  subagent_type: "general-purpose",  // ← WRONG (should be "Explore")
  model: "haiku",
  prompt: "AGENT: codebase-search\n..."
})
```

**Validation result**: ❌ BLOCK

- Hook extracts: agent = "codebase-search", provided = "general-purpose"
- Hook looks up: required = "Explore"
- Hook compares: "general-purpose" != "Explore"
- Result: Mismatch found, Task() fails with error

**Error message**:
```
ERROR [validate-routing] INVALID subagent_type for 'codebase-search'
  Provided:  general-purpose
  Required:  Explore
  Reason:    Agent is read-only (file discovery). Use Explore.
  Fix:       Change subagent_type to 'Explore'

  Logged to: /tmp/claude-routing-violations.jsonl
  See docs:  ~/.claude/docs/agent-reference-table.md
```

## Error Messages

The hook produces clear, actionable error messages. Here are common examples:

### Error: Explore for write-requiring agent

```
ERROR [validate-routing] INVALID subagent_type for 'tech-docs-writer'
  Provided:  Explore
  Required:  general-purpose
  Reason:    Agent needs write permissions to create/modify documentation
  Fix:       Use subagent_type: 'general-purpose'
```

**Root cause**: Trying to read-only an agent that needs to write files

**How to fix**: Change to `"general-purpose"`

### Error: general-purpose for read-only agent

```
ERROR [validate-routing] INVALID subagent_type for 'codebase-search'
  Provided:  general-purpose
  Required:  Explore
  Reason:    Agent is read-only. Reconnaissance should never modify files.
  Fix:       Use subagent_type: 'Explore'
```

**Root cause**: Giving write permissions to a discovery agent

**How to fix**: Change to `"Explore"`

### Error: Plan for implementation agent

```
ERROR [validate-routing] INVALID subagent_type for 'python-pro'
  Provided:  Plan
  Required:  general-purpose
  Reason:    Implementation agents need full tool access, not planning mode
  Fix:       Use subagent_type: 'general-purpose'
```

**Root cause**: Confusing planning and implementation tiers

**How to fix**: Change to `"general-purpose"`

### Error: Bash for non-Gemini agent

```
ERROR [validate-routing] INVALID subagent_type for 'orchestrator'
  Provided:  Bash
  Required:  Plan
  Reason:    Only gemini-slave uses Bash subagent_type (shell piping)
  Fix:       Use subagent_type: 'Plan'
```

**Root cause**: Bash is only for Gemini, not other agents

**How to fix**: Use correct subagent_type for your agent

## Violation Log

All validation failures are logged to: `/tmp/claude-routing-violations.jsonl`

### Log Entry Format

Each violation is a JSON object with these fields:

```json
{
  "timestamp": "2026-01-13T14:22:45.123Z",
  "session_id": "sess_abc123xyz",
  "session_branch": "main",
  "agent": "tech-docs-writer",
  "subagent_type_provided": "Explore",
  "subagent_type_required": "general-purpose",
  "reason": "Agent requires write permissions for documentation task",
  "suggestion": "Change subagent_type to 'general-purpose' in Task() call",
  "context": {
    "model": "haiku",
    "task_description": "Update documentation"
  },
  "resolution": "Fixed in iteration 2",
  "violation_id": "viol_xyz789"
}
```

### Fields

| Field | Meaning | Example |
|-------|---------|---------|
| `timestamp` | When violation occurred | "2026-01-13T14:22:45.123Z" |
| `session_id` | Session identifier | "sess_abc123xyz" |
| `agent` | Agent that was called | "tech-docs-writer" |
| `subagent_type_provided` | What user wrote | "Explore" |
| `subagent_type_required` | What schema requires | "general-purpose" |
| `reason` | Why it's wrong | "Agent needs write permissions" |
| `suggestion` | How to fix it | "Use subagent_type: 'general-purpose'" |
| `context` | Additional info | model, description, etc. |
| `resolution` | How it was fixed | "Fixed in iteration 2", "Ignored" |
| `violation_id` | Unique ID for tracking | "viol_xyz789" |

## Reading the Violation Log

### View recent violations

```bash
tail -20 /tmp/claude-routing-violations.jsonl
```

### View violations for specific agent

```bash
grep '"agent": "tech-docs-writer"' /tmp/claude-routing-violations.jsonl
```

### View violations from today

```bash
jq 'select(.timestamp | startswith("2026-01-13"))' /tmp/claude-routing-violations.jsonl
```

### Count violations by agent

```bash
jq -r '.agent' /tmp/claude-routing-violations.jsonl | sort | uniq -c
```

### Find unresolved violations

```bash
jq 'select(.resolution == null)' /tmp/claude-routing-violations.jsonl
```

### Pretty-print a violation

```bash
tail -1 /tmp/claude-routing-violations.jsonl | jq '.'
```

## Debugging Guide

### Step 1: Understand the Error

When you see a validation error:

1. **Read the error message** carefully
2. **Find the agent name** in the error
3. **Note the required subagent_type**

### Step 2: Verify in Schema

Double-check the schema:

```bash
jq '.agent_subagent_mapping["your-agent-name"]' ~/.claude/routing-schema.json
```

Replace `your-agent-name` with the agent from the error.

### Step 3: Fix the Task() Call

Update your Task() call with the correct subagent_type:

```javascript
// BEFORE (wrong)
Task({
  subagent_type: "Explore",
  prompt: "AGENT: tech-docs-writer\n..."
})

// AFTER (correct)
Task({
  subagent_type: "general-purpose",
  prompt: "AGENT: tech-docs-writer\n..."
})
```

### Step 4: Test

Re-run the Task() call. If validation passes, execution proceeds.

## Common Mistakes and Fixes

| Mistake | Error | Fix |
|---------|-------|-----|
| Using "Explore" for documentation | "tech-docs-writer requires general-purpose" | Change to `"general-purpose"` |
| Using "general-purpose" for search | "codebase-search requires Explore" | Change to `"Explore"` |
| Using "Plan" for implementation | "python-pro requires general-purpose" | Change to `"general-purpose"` |
| Using "Bash" for non-Gemini | "orchestrator requires Plan" | Use correct subagent_type for agent |
| Typo in subagent_type | "Unknown subagent_type: Expore" | Check spelling: "Explore" (capital E) |
| Mixing up agent and model | "No such agent: haiku" | Agent and model are different. Check agent name |

## Hook Comparison

How `validate-routing.sh` compares to other hooks:

| Hook | When | What | Action |
|------|------|------|--------|
| `load-routing-context` | Session start | Load routing schema | Injects context |
| `validate-routing` | Pre-Task execution | Check subagent_type | BLOCK if invalid |
| `sharp-edge-detector` | Post-Bash/Edit/Write | Detect failures | Logs sharp edges |
| `attention-gate` | Every 10 tool calls | Routing compliance | Reminds user |
| `agent-endstate` | Agent completion | Tier follow-ups | Suggests next steps |
| `session-archive` | Session end | Save learnings | Archives to memory |

## Configuration

The hook reads configuration from:

1. **routing-schema.json**: Source of truth for subagent_type mappings
2. **agent.yaml** files: Per-agent tool requirements
3. **Override flag**: `--force-tier=<tier>` for escape hatches (audited)

### Customize Validation Rules

To add new agents or change mappings:

1. Edit `/home/doktersmol/.claude/routing-schema.json`
2. Update `agent_subagent_mapping` section
3. Restart session (hook reloads schema)
4. New validation rules take effect immediately

## Troubleshooting the Hook

### Hook not blocking invalid calls

**Cause**: Hook may not be active

**Check**:
```bash
ls -la ~/.claude/hooks/validate-routing.sh
# Should exist and be executable
```

**Fix**: Reinstall hooks or check session configuration

### Violation log not being written

**Cause**: Permissions or path issue

**Check**:
```bash
ls -la /tmp/
# /tmp should be writable
```

**Fix**: Check disk space, file permissions

### Hook is too strict

**Cause**: Schema may be too strict for your use case

**Workaround**: Use `--force-tier=` flag (will be logged):
```javascript
Task({
  subagent_type: "Explore",
  force_override: "--force-tier=general-purpose",
  // ... rest of Task()
})
```

**Note**: Overrides are audited. Use sparingly.

## Best Practices

### 1. Use the Reference Table

Always check `/home/doktersmol/.claude/docs/agent-reference-table.md` before writing a Task() call.

### 2. Verify Before Running

Run this before executing Task():

```bash
jq '.agent_subagent_mapping["your-agent"]' ~/.claude/routing-schema.json
```

### 3. Read Error Messages

The error messages are detailed and actionable. Follow the "Fix:" suggestion.

### 4. Monitor the Log

Periodically check for patterns:

```bash
jq -r '.agent' /tmp/claude-routing-violations.jsonl | sort | uniq -c | sort -rn
```

If you see repeated violations for the same agent, it may signal:
- A documentation gap
- An unclear agent purpose
- A schema that needs updating

### 5. Document Decisions

If you override the hook (rare), document why:

```javascript
Task({
  description: "Emergency fix - override validate-routing",
  subagent_type: "Explore",  // Override required for this task
  force_override: "--force-tier=general-purpose",
  // ... explain in comments why override was needed
})
```

## Related Documentation

- **Routing Enforcement Architecture**: `/home/doktersmol/.claude/docs/architecture/routing-enforcement.md`
- **Agent Reference Table**: `/home/doktersmol/.claude/docs/agent-reference-table.md`
- **Main Configuration**: `/home/doktersmol/.claude/CLAUDE.md`
- **Routing Schema**: `/home/doktersmol/.claude/routing-schema.json`

## FAQ

**Q: Why does the hook exist if documentation says the same thing?**
A: Documentation can be misread. Code cannot. The hook enforces rules automatically, preventing silent failures.

**Q: Can I disable the hook?**
A: Not recommended. It prevents expensive mistakes. If truly needed, contact system admin.

**Q: What happens if I ignore a validation error?**
A: Task() call won't execute. You must fix it. This prevents wasted tokens.

**Q: Are violations permanent?**
A: Violations are logged but not permanent. You can mark them as `"resolution": "fixed"` in the log after fixing the issue.

**Q: How often should I clean up the violation log?**
A: The log is auto-rotated. Typically kept for 30 days. No manual cleanup needed.

**Q: Can I add custom validation rules?**
A: Yes, by editing `routing-schema.json` and restarting your session.

**Q: What if the schema is wrong?**
A: File an issue with:
- The agent name
- The subagent_type that should be required
- Why the current mapping is wrong
- Evidence (error logs, documentation references)
