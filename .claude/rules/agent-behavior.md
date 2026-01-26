---
paths:
  - "**/*"
alwaysApply: true
---

# Agent Behavioral Guidelines

**Audience:** This document instructs Claude (the agent) on optimal behavior patterns.
**Scope:** All sessions, all project types.

---

## 1. Routing Discipline

### 1.1 Always Check Before Acting

Before using ANY tool, verify:
1. Does this task match a Key Trigger in CLAUDE.md?
2. Is the current tier appropriate for this work?
3. Should I scout first to assess scope?

**Reference:** `~/.claude/routing-schema.json` is the source of truth for tier thresholds.

### 1.2 Tier Selection Matrix

| Task Complexity | Context Size | Tier | Action |
|-----------------|--------------|------|--------|
| Mechanical (count, find, grep) | Any | Haiku | Direct or codebase-search |
| Structured output (docs, scaffold) | <1000 lines | Haiku+Thinking | Delegate to specialist |
| Reasoning required | <5000 tokens | Sonnet | Delegate to implementation agent |
| Multi-source synthesis | Any | Sonnet (orchestrator) | Coordinate multiple agents |
| Novel/complex/security | Any | Opus (einstein) | **Generate GAP doc** (see 3.4) |
| >10 files or >50K tokens | Large | External (Gemini) | Pipe to gemini-slave first |

### 1.2.1 Go Implementation Agents (Sonnet Tier)

| Trigger Patterns | Agent | Use For |
|------------------|-------|---------|
| implement, struct, interface, go build | `go-pro` | Core Go implementation |
| Cobra, CLI, subcommand, flags | `go-cli` | CLI applications |
| Bubbletea, TUI, lipgloss, tea.Model | `go-tui` | Terminal interfaces |
| HTTP client, API, rate limit, retry | `go-api` | HTTP clients/servers |
| goroutine, errgroup, channel, mutex | `go-concurrent` | Concurrent patterns |

These agents understand Go idioms: explicit error handling, small interfaces, composition over inheritance, table-driven tests.

### 1.3 Scout Before Commit

**When scope is unknown:**
```
[SCOUTING] Unknown scope detected. Spawning haiku-scout...
```

Then wait for scout results before selecting tier. This prevents $0.50 Opus calls on $0.02 Haiku work.

---

## 2. Parallel Agent Management

### 2.1 Background vs Foreground

| Pattern | Use When | Mechanism |
|---------|----------|-----------|
| **Foreground (default)** | Next step depends on this output | `Task({...})` |
| **Background** | Independent work, will collect later | `Bash({..., run_in_background: true})` |
| **Parallel foreground** | Multiple independent, need all before continuing | Multiple `Task()` in same message |

### 2.2 MANDATORY: Background Task Collection

**Enforcement:** `gogent-orchestrator-guard` (SubagentStop hook) blocks orchestrator completion when background tasks remain uncollected.

**If you spawn background tasks, you MUST:**

1. Track every task_id returned
2. Before ANY final output or synthesis:
   ```javascript
   TaskOutput({task_id: "bg-task-1", block: true})
   TaskOutput({task_id: "bg-task-2", block: true})
   ```
3. NEVER conclude orchestration with uncollected background tasks

**Violation Pattern (BLOCKED by hook):**
```javascript
Bash({..., run_in_background: true})  // Spawned
Bash({..., run_in_background: true})  // Spawned
// ... do other work ...
// Output synthesis WITHOUT calling TaskOutput → BLOCKED by gogent-orchestrator-guard
```

### 2.3 Fan-Out, Fan-In Pattern

For parallel information gathering:

```javascript
// 1. FAN-OUT: Spawn all tasks
const task1 = Task({...})  // Returns task_id
const task2 = Task({...})  // Returns task_id  
const task3 = Task({...})  // Returns task_id

// 2. FAN-IN: Collect all results (MANDATORY)
const result1 = TaskOutput({task_id: task1, block: true})
const result2 = TaskOutput({task_id: task2, block: true})
const result3 = TaskOutput({task_id: task3, block: true})

// 3. SYNTHESIZE: Only after all collected
// Now proceed with synthesis
```

---

## 3. Failure Handling

### 3.1 Automatic Escalation Triggers

| Condition | Action |
|-----------|--------|
| 2 failures on same file | Warning injected (via hook) |
| 3 failures on same file | Sharp edge captured, escalation prompted |
| Agent returns error | Retry with modified approach ONCE |
| Retry also fails | Escalate to next tier |

### 3.2 Retry with Modification

When an approach fails, do NOT retry identically. Modify:
- Different tool selection
- Smaller scope
- More context provided
- Different agent

**Bad:**
```
[Attempt 1] Edit file X → Error
[Attempt 2] Edit file X → Same error  // WRONG: identical retry
```

**Good:**
```
[Attempt 1] Edit file X → Error
[Analysis] Error suggests type mismatch
[Attempt 2] Read file X first, then Edit with correct types
```

### 3.3 Sharp Edge Protocol

When a debugging loop is detected:
1. STOP current approach
2. Document the pattern (auto-logged by hook)
3. Consider: What assumption was wrong?
4. Either:
   - Fix assumption and retry differently, OR
   - Escalate to higher tier with context

### 3.4 Escalate to Einstein Protocol

**Enforcement:** `gogent-validate` (Go binary, PreToolUse hook) **blocks** `Task(model: "opus")` calls. Must use `/einstein` slash command.

#### Trigger Conditions

Escalate to Einstein when:

| Condition | Detection |
|-----------|-----------|
| **3+ consecutive failures** | Same file/function, same error class |
| **Architectural decision required** | Solution requires cross-module tradeoffs |
| **Complexity exceeds Sonnet tier** | Scout returns `recommended_tier: opus` |
| **User explicitly requests** | "call einstein", "deep analysis needed" |
| **Novel problem** | No pattern in sharp-edges.yaml applies |

#### Escalation Procedure

1. **STOP** current execution
2. **Generate GAP document** using template at `~/.claude/schemas/einstein-gap.md`
3. **Write** to `.claude/tmp/einstein-gap-{timestamp}.md`
4. **Output notification**:
   ```
   [ESCALATED] GAP document ready: .claude/tmp/einstein-gap-{timestamp}.md

   🚨 Run `/einstein` to process this escalation.

   Summary:
   - Problem: {brief_problem}
   - Attempts: {attempt_count}
   - Blocker: {primary_blocker}
   ```
5. **WAIT** for user to run `/einstein`

#### What NOT to Do

- ❌ **DO NOT** invoke Einstein via Task tool (hook will block it)
- ❌ **DO NOT** continue attempting after 3 failures (you're looping)
- ❌ **DO NOT** generate incomplete GAP documents (garbage in = garbage out)
- ❌ **DO NOT** escalate trivial problems (use code-reviewer for sanity checks first)

#### GAP Document Quality Checklist

Before writing the GAP document, verify:

- [ ] Problem statement is specific (not "it doesn't work")
- [ ] All attempts are logged with actual error messages
- [ ] Relevant file excerpts are included (not just paths)
- [ ] Constraints are explicit (not assumed)
- [ ] Question is answerable from provided context
- [ ] Anti-scope prevents scope creep

**Reference:** See `orchestrator/agent.md` for complete GAP generation code example.

---

## 4. Hook Awareness

### 4.1 Active Hooks

The following Go binaries run as hooks and inject context automatically:

| Binary | Event | What You'll See |
|--------|-------|-----------------|
| `gogent-load-context` | SessionStart | Routing schema, previous handoff, git context |
| `gogent-validate` | PreToolUse (Task) | Block/allow decision, subagent_type enforcement |
| `gogent-sharp-edge` | PostToolUse | Tool counter, routing reminders (every 10), failure tracking |
| `gogent-agent-endstate` | SubagentStop | Decision outcomes, tier-specific follow-up prompts |
| `gogent-orchestrator-guard` | SubagentStop | Background task collection enforcement |
| `gogent-archive` | SessionEnd | Handoff generation, metrics capture |

### 4.2 Responding to Hook Injections

When you see `additionalContext` in a hook response:
- READ the injected guidance
- FOLLOW the recommendations
- Do NOT ignore or dismiss

---

## 5. Memory & Learning

### 5.1 Knowledge Compounding

When you discover something worth remembering:

1. **Sharp Edges** (errors, gotchas): Auto-captured by hook → review at session end
2. **Decisions** (architectural choices): Document in `memory/decisions/`
3. **Patterns** (successful approaches): Propose addition to conventions

### 5.2 Session Handoff

At session end:
- Pending learnings are archived automatically
- Handoff document generated at `memory/last-handoff.md`
- Next session receives this context via `gogent-load-context` hook

### 5.3 Evolution Cycle

```
Work → Detect patterns → Capture to memory → 
Weekly audit (Gemini) → Propose config updates → 
Benchmark test → If improved: commit
```

---

## 6. Cost Optimization

### 6.1 Token Budget Awareness

| Tier | Thinking Budget | Cost/1K tokens |
|------|-----------------|----------------|
| Haiku | 0 or 2-4K | $0.0005 |
| Haiku+Thinking | 4-6K | $0.001 |
| Sonnet | 10-16K | $0.009 |
| Opus | 16-32K | $0.045 |

### 6.2 Cost-Saving Patterns

1. **Scout before expensive work**: $0.02 scout can prevent $0.50 mis-routing
2. **Haiku for mechanical**: Never use Sonnet for grep/find/count
3. **Gemini for large context**: Cheaper than Sonnet for >50K tokens
4. **Batch similar operations**: One agent call with multiple files > multiple calls

### 6.3 Delegation Overhead Threshold

If task is <$0.01 of work, do it directly rather than delegating. Delegation itself costs tokens.

---

## 7. Output Quality

### 7.1 Self-Verification

Before returning output to user:
1. Does it answer the actual question?
2. Does it follow relevant conventions?
3. Are there obvious errors?
4. Would a quick code-reviewer pass help?

### 7.2 Critic Pattern (Optional)

For important outputs, invoke quick review:
```javascript
Task({
  model: "haiku",
  prompt: "Review this output for obvious errors: [output]"
})
```

Cost: ~$0.005. Worth it for user-facing deliverables.

---

## 8. Anti-Patterns

### 8.1 FORBIDDEN Behaviors

| Anti-Pattern | Why Bad | Correct Approach |
|--------------|---------|------------------|
| Retrying identically after failure | Wastes tokens, won't help | Modify approach |
| Using Sonnet for file search | 50x cost waste | Use Haiku/codebase-search |
| Spawning background tasks without collecting | Orphaned work | Always call TaskOutput |
| Ignoring hook injections | Misses guidance | Read and follow |
| Skipping scout on unknown scope | Potential mis-routing | Scout first |
| Large context without Gemini | Context overflow | Pipe to gemini-slave |

### 8.2 WARNING Behaviors

| Behavior | Risk | Mitigation |
|----------|------|------------|
| >3 agents in one task | Coordination complexity | Consider orchestrator |
| Opus for routine work | Cost | Verify Opus triggers present |
| Direct file editing without reading | Context gaps | Read first |

---

## 9. Checklist: Before Completing Task

- [ ] All background tasks collected?
- [ ] Routing tier was appropriate?
- [ ] No obvious errors in output?
- [ ] Sharp edges documented if any?
- [ ] Conventions followed?
- [ ] User's actual question answered?

---

**Remember:** Your effectiveness is bounded by routing discipline. Wrong tier = wasted tokens + suboptimal output.
