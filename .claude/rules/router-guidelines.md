---
alwaysApply: true
---

# Router Guidelines

Router-essential guidance for the GOgent-Fortress system. This file contains ONLY the subset of rules that apply to Claude's routing decisions and system awareness.

---

## 1. Routing Discipline

### 1.1 Always Check Before Acting

Before using ANY tool, verify:
1. Does this task match a Key Trigger in CLAUDE.md?
2. Is the current tier appropriate for this work?
3. Should I scout first to assess scope?

**Reference:** `~/.claude/routing-schema.json` is the source of truth for tier thresholds.

### 1.2 Tier Selection

**Haiku tier**: Mechanical tasks (count, find, grep) regardless of context size → use directly or delegate to codebase-search.

**Haiku+Thinking**: Structured output (docs, scaffold) under 1000 lines → delegate to specialist agent.

**Sonnet tier**: Tasks requiring reasoning under 5000 tokens → delegate to implementation agent.

**Sonnet (orchestrator)**: Multi-source synthesis → coordinate multiple agents.

**Opus tier (einstein)**: Novel, complex, or security-critical tasks → **generate GAP document** per Section 3.

**External (Gemini)**: Over 10 files or 50K tokens → pipe to gemini-slave first.

#### 1.2.1 Go Implementation Agents (Sonnet Tier)

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

## 2. Hook Awareness

### 2.1 Active Hooks

The following Go binaries run as hooks and inject context automatically:

| Binary | Event | What You'll See |
|--------|-------|-----------------|
| `gogent-load-context` | SessionStart | Routing schema, previous handoff, git context |
| `gogent-validate` | PreToolUse (Task) | Block/allow decision, subagent_type enforcement |
| `gogent-sharp-edge` | PostToolUse | Tool counter, routing reminders (every 10), failure tracking |
| `gogent-agent-endstate` | SubagentStop | Decision outcomes, tier-specific follow-up prompts |
| `gogent-orchestrator-guard` | SubagentStop | Background task collection enforcement |
| `gogent-archive` | SessionEnd | Handoff generation, metrics capture |

### 2.2 Responding to Hook Injections

When you see `additionalContext` in a hook response:
- READ the injected guidance
- FOLLOW the recommendations
- Do NOT ignore or dismiss

---

## 3. Escalation Protocol

### 3.1 Escalate to Einstein Protocol

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

**Reference:** See `orchestrator/orchestrator.md` for complete GAP generation code example.

---

## 4. Multi-Model Strategy

### 4.1 CRITICAL: Tiered Model Routing

Use `Task(model: "haiku")` or `Task(model: "sonnet")` to delegate work to cheaper models. Only keep quality-critical tasks in Opus.

### 4.2 When to Use Different Models

| Task Type | Model | Rationale |
|-----------|-------|-----------|
| **OPUS (Quality Critical)** | | |
| Interview/requirements gathering | Opus | Quality of questions determines outcome |
| Planning/architecture | Opus | Complex tradeoffs need depth |
| Cross-domain synthesis | Opus | Connecting 5+ sources needs reasoning |
| Conflict judgment | Opus | Requires nuanced assessment |
| MCP API calls | Opus | Direct API, delegation overhead exceeds savings |
| **SONNET (Reasoning, Familiar)** | | |
| Go implementation (go-pro, go-tui, go-cli) | Sonnet | Needs reasoning, follows Go idioms |
| Code understanding | Sonnet | Needs reasoning but standard patterns |
| Core implementation | Sonnet | Following established patterns |
| Single-domain analysis | Sonnet | Focused analysis, not cross-cutting |
| Documentation generation | Sonnet | Structured output with reasoning |
| Concurrency design (go-concurrent) | Sonnet | Channel patterns, error propagation |
| **HAIKU (Mechanical Work)** | | |
| File discovery (glob, find, ls) | Haiku | Pure file operations |
| Pattern extraction (grep, regex) | Haiku | Mechanical matching |
| Keyword extraction | Haiku | Text parsing |
| Result formatting | Haiku | Structured output, no reasoning |
| Skill/index loading | Haiku | File reading |
| Boilerplate generation | Haiku | Template following |
| Sharp edge detection | Haiku | Pattern matching against known list |
| Code review (style only) | Haiku | Convention checking, no design judgment |

### 4.3 Routing Enforcement

When using `/explore` or similar workflows:

1. **ALWAYS announce routing** with `[ROUTING] → Model (reason)`
2. **Use Task tool** with explicit `model: "haiku"` or `model: "sonnet"`
3. **Never use Glob/Grep/Read directly** for exploration - spawn Haiku scouts
4. **Stay in Opus** only for interview, planning, synthesis, and judgment

### 4.4 Cost Impact

Aggressive tiered routing saves ~70% on exploration workflows:
- Haiku: ~$0.0005/1k tokens (50x cheaper than Opus)
- Sonnet: ~$0.009/1k tokens (5x cheaper than Opus)
- Opus: ~$0.045/1k tokens (baseline)

### 4.5 Parallel Agent Patterns

For complex research tasks, launch multiple Haiku scouts in parallel:
```
- Haiku Scout 1: File discovery (glob patterns)
- Haiku Scout 2: Pattern extraction (grep)
- Haiku Scout 3: Code snippet extraction
→ Sonnet Analyst: Synthesize findings
→ Opus Main: Make architectural decisions
```

For Go implementation tasks, consider:
```
- Haiku Scout: Find existing patterns (grep for similar interfaces)
- go-pro (Sonnet): Implement core logic
- code-reviewer (Haiku): Verify conventions
```

---

## 5. Context Window Optimization

### 5.1 What to Include

- Full class/type definitions for code being modified
- Representative sample data (structure, not volume)
- Error messages with complete tracebacks
- Related functions that must integrate
- Relevant configuration/constants

### 5.2 What to Summarize

- Large datasets → representative samples + schema
- Long files → relevant sections + structure overview
- History → key decisions and constraints

### 5.3 What to Reference

- Rule files by name: "per go.md conventions"
- Previous conversation context: "as discussed above"
- External docs: use WebFetch tool or MCP-provided fetch tools

---

## 6. Enforcement Architecture

### 6.1 The Anti-Pattern: Documentation Theater

**Definition:** Adding imperative enforcement language ("MUST NOT", "NEVER", "BLOCKED") to CLAUDE.md or other documentation files, creating the illusion of enforcement without any actual mechanism.

**Why it fails:**
- Text instructions are probabilistic suggestions, not deterministic rules
- Attention to early instructions degrades over long conversations
- No mechanism exists to actually BLOCK a tool call via text
- Creates false confidence that behavioral problems are "solved"
- CLAUDE.md becomes bloated with unenforceable imperatives

### 6.2 The Correct Pattern: Declarative → Programmatic → Reference

**Three components, in order:**

1. **Declarative Rule** (`routing-schema.json`)
   - Single source of truth for what's allowed/blocked
   - Parsed by hooks at runtime
   - Example: `"task_invocation_blocked": true`

2. **Programmatic Enforcement** (Go hook binary, e.g., `gogent-validate`)
   - Actually runs before/after tool use
   - Can block, warn, or modify behavior
   - Example: Check schema rule, return `routing.BlockResponse()` with reason

3. **Reference Documentation** (`CLAUDE.md`)
   - Points to enforcement, doesn't replace it
   - Example: "Blocked by gogent-validate (PreToolUse hook)"
   - Provides context for WHY, not enforcement of WHAT

### 6.3 Decision Tree: Where Does This Go?

```
Is this enforcement of a behavior?
│
├─ YES: Can it be detected programmatically?
│   │
│   ├─ YES: What kind of enforcement?
│   │   │
│   │   ├─ Block action → routing-schema.json rule
│   │   │                 + gogent-validate check (Go binary)
│   │   │                 + CLAUDE.md reference
│   │   │
│   │   ├─ Require action → Hook injects reminder at trigger
│   │   │                   + CLAUDE.md documents workflow
│   │   │
│   │   └─ Warn on pattern → PreToolUse hook with warning
│   │                        + CLAUDE.md notes the check
│   │
│   └─ NO: Is it methodology guidance?
│       │
│       ├─ YES → LLM-guidelines.md (this file)
│       │
│       └─ NO → agent-behavior.md or conventions/*.md
│
└─ NO: Is this describing existing system behavior?
    │
    ├─ YES → CLAUDE.md (gates, workflows, triggers)
    │
    └─ NO → Probably doesn't need to be written
```

### 6.4 What Goes Where: Quick Reference

| Need | ❌ Wrong | ✅ Right |
|------|----------|----------|
| Block a tool pattern | "You MUST NOT use X" in CLAUDE.md | `routing-schema.json` rule + `gogent-validate` enforcement + CLAUDE.md reference |
| Require pre-check | "ALWAYS check Y first" in CLAUDE.md | Hook injects reminder at trigger point |
| Prevent anti-pattern | "NEVER do Z" in CLAUDE.md | This section in LLM-guidelines.md + warning hook |
| Document workflow | Gates 1-5 in CLAUDE.md | ✅ Appropriate (this IS documentation) |
| Agent-specific rule | In CLAUDE.md | `agents/*/sharp-edges.yaml` or `agents/*/{agent-name}.md` (unified frontmatter) |

### 6.5 Pre-Commit Checklist for CLAUDE.md Edits

Before adding enforcement-style language to CLAUDE.md:

- [ ] Is this DESCRIPTION of existing behavior, or ENFORCEMENT of new behavior?
- [ ] If enforcement: Is it implemented in a hook FIRST?
- [ ] Does CLAUDE.md text REFERENCE the hook (file + line), not REPLACE it?
- [ ] Are there any new "MUST", "NEVER", "BLOCKED" without corresponding code?
- [ ] Would this still work if the LLM ignores this paragraph?

If any answer is wrong, implement programmatic enforcement first.

### 6.6 What CLAUDE.md IS For

✅ **Appropriate content:**
- Gates (workflow checkpoints with structure)
- Trigger tables (pattern → agent mapping)
- System constraints (Arch Linux, Python paths)
- References ("See hook X for enforcement")
- Context loading (conventions, skills)

❌ **Inappropriate content:**
- Behavioral blocking ("MUST NOT use X")
- Imperative requirements without enforcement
- Rules that depend on LLM "remembering"
- Anything that fails silently when ignored

### 6.7 Example: Correct vs Incorrect

**Scenario:** Need to prevent Task(opus) invocations

❌ **Incorrect (documentation theater):**
```markdown
## Gate 6: Einstein Protection

**You MUST NOT invoke Einstein via Task tool.**
**This is BLOCKED. Use /einstein slash command instead.**
```

✅ **Correct (layered enforcement):**

1. `routing-schema.json`:
```json
"opus": {
  "task_invocation_blocked": true,
  "blocked_reason": "60K+ token inheritance overhead"
}
```

2. `cmd/gogent-validate/main.go`:
```go
if event.Task != nil && event.Task.Model == "opus" {
    return routing.BlockResponse(
        "Task(opus) blocked by gogent-validate. Use /einstein instead.",
    )
}
```

3. `CLAUDE.md`:
```markdown
## Gate 6: Einstein Escalation

Einstein invocation via Task tool is blocked by `gogent-validate` (PreToolUse hook).
See `routing-schema.json` → `opus.task_invocation_blocked`.

When Einstein triggers fire, use `escalate_to_einstein` protocol instead.
Reference: `~/.claude/skills/einstein/SKILL.md`
```

The CLAUDE.md version describes and references; it doesn't pretend to enforce.

---

## 7. Cost Optimization

### 7.1 Token Budget Awareness

| Tier | Thinking Budget | Cost/1K tokens |
|------|-----------------|----------------|
| Haiku | 0 or 2-4K | $0.0005 |
| Haiku+Thinking | 4-6K | $0.001 |
| Sonnet | 10-16K | $0.009 |
| Opus | 16-32K | $0.045 |

### 7.2 Cost-Saving Patterns

1. **Scout before expensive work**: $0.02 scout can prevent $0.50 mis-routing
2. **Haiku for mechanical**: Never use Sonnet for grep/find/count
3. **Gemini for large context**: Cheaper than Sonnet for >50K tokens
4. **Batch similar operations**: One agent call with multiple files > multiple calls

### 7.3 Delegation Overhead Threshold

If task is <$0.01 of work, do it directly rather than delegating. Delegation itself costs tokens.

---

## 8. System-Level Anti-Patterns

These anti-patterns apply to Claude's internal behavior when executing tasks within the GOgent-Fortress system.

### 8.1 FORBIDDEN Behaviors

| Anti-Pattern | Why Bad | Correct Approach |
|--------------|---------|------------------|
| Retrying identically after failure | Wastes tokens, won't help | Modify approach (different tool, smaller scope, more context) |
| Using Sonnet for file search | 50x cost waste | Use Haiku/codebase-search |
| Spawning background tasks without collecting | Orphaned work, blocked by gogent-orchestrator-guard | Always call TaskOutput before final synthesis |
| Ignoring hook injections | Misses guidance/enforcement | Read and follow additionalContext from hooks |
| Skipping scout on unknown scope | Potential mis-routing to expensive tier | Scout first with haiku-scout or gogent-scout |
| Large context without Gemini | Context overflow, high cost | Pipe to gemini-slave for >50K tokens |

### 8.2 WARNING Behaviors

| Behavior | Risk | Mitigation |
|----------|------|------------|
| >3 agents in one task | Coordination complexity, hard to debug | Consider orchestrator agent for multi-agent coordination |
| Opus for routine work | High cost ($0.045/1K tokens vs $0.0005 Haiku) | Verify Opus triggers present, consider downgrade to Sonnet/Haiku |
| Direct file editing without reading | Context gaps, incorrect assumptions | Always Read file first to understand current state |

---

## 9. Checklist: Before Routing

- [ ] Does this match a slash command trigger?
- [ ] Does this match an agent trigger in CLAUDE.md?
- [ ] Is scope unknown? (scout first)
- [ ] Is this trivial? (handle directly)
- [ ] Am I choosing the correct tier?
- [ ] Have I announced routing decision with `[ROUTING] → agent (reason)`?

---

**Remember:** Your routing effectiveness is bounded by proper tier selection and cost awareness. Wrong tier = wasted effort + suboptimal output.
