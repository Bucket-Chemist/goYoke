# Braintrust Analysis: Subagent Nesting Limitation

**Generated**: 2026-02-04
**Session**: Braintrust investigation of review-orchestrator failure
**Status**: CRITICAL FINDING - Architecture limitation discovered

---

## Executive Summary

During investigation of why `review-orchestrator` fails to spawn subagents, we discovered a **fundamental Claude Code architecture limitation**: subagents cannot spawn sub-subagents. This invalidates the entire orchestrator-pattern design where coordination agents delegate to specialist agents.

**Key Finding**: The Task tool is only available at nesting level 0 (router). Agents spawned via Task() do not have access to Task() themselves.

---

## Timeline of Discovery

### Phase 1: Original Problem

**Symptom**: `review-orchestrator` invoked to coordinate multi-domain code review.

**Expected behavior**:
1. Orchestrator spawns 3 specialist reviewers via Task()
2. Each reviewer examines files in their domain
3. Orchestrator collects and synthesizes findings

**Actual behavior**:
- Attempt 1: Orchestrator simulated all review work directly (zero Task calls)
- Attempt 2: Orchestrator created TaskCreate TODO items instead of Task subprocess invocations

**Initial diagnosis**: Task/TaskCreate tool confusion due to semantic similarity.

### Phase 2: Braintrust Investigation

Invoked `/braintrust` to analyze the problem with multi-perspective deep analysis.

**Mozart (Braintrust orchestrator) was spawned** to:
1. Conduct clarification interview ✅
2. Spawn scouts for reconnaissance ❌
3. Spawn Einstein (theoretical analysis) ❌
4. Spawn Staff-Architect (practical review) ❌
5. Spawn Beethoven (synthesis) ❌

**Mozart's response**: "As a planning agent in read-only mode, I do not have access to the Task tool to spawn subagents."

Mozart delivered comprehensive analysis directly instead of delegating - **the exact same failure pattern as review-orchestrator**.

### Phase 3: Verification Test

Tested whether Task() works at router level:

```javascript
Task({
  description: "Test: Simple subagent spawn",
  subagent_type: "Explore",
  model: "haiku",
  prompt: "AGENT: codebase-search\nTASK: Count .go files in cmd/"
})
```

**Result**: Success. Subagent spawned, executed, returned "46 .go files".

### Phase 4: Revised Diagnosis

| Nesting Level | Agent | Task Tool Access |
|---------------|-------|------------------|
| 0 | Router (main Claude instance) | ✅ Available |
| 1 | Any subagent (Mozart, review-orchestrator, etc.) | ❌ Not available |
| 2 | Sub-subagent (would be spawned by level 1) | N/A (cannot exist) |

**Root cause is NOT Task/TaskCreate confusion. Root cause is architectural: subagents cannot spawn sub-subagents.**

---

## Evidence Summary

### Evidence 1: Mozart's Explicit Statement

```
"As a planning agent in read-only mode, I do not have access to the Task tool to spawn subagents."
```

Mozart is defined with:
- `subagent_type: "Plan"`
- `tools: [Read, Glob, Grep, Task, TaskList, TaskGet, TaskCreate, TaskUpdate, Write, AskUserQuestion]`

The agent definition INCLUDES Task, but the runtime did not provide it.

### Evidence 2: Router-Level Task Success

```
Task({description: "Test: Simple subagent spawn", ...})
Result: Success - agent executed and returned result
```

Task() works when called from nesting level 0.

### Evidence 3: review-orchestrator Definition

From `~/.claude/agents/review-orchestrator/review-orchestrator.md`:

```yaml
tools:
  - Read
  - Glob
  - Grep
  - Task      # Listed but apparently not available at runtime
  - Write
  - TaskList
  - TaskGet
```

The agent definition lists Task as available, but runtime behavior contradicts this.

### Evidence 4: Consistent Pattern Across Orchestrators

Both `review-orchestrator` and `mozart` exhibited identical failure:
- Claimed/intended to spawn subagents
- Did not actually invoke Task tool
- Delivered work directly instead

This consistency suggests a systemic limitation, not agent-specific bug.

---

## Implications

### Implication 1: Orchestrator Pattern is Fundamentally Broken

The GOgent architecture assumes orchestrator agents can delegate:

```
Router → Orchestrator → Specialist Agents
              ↓
         Coordinates multiple specialists
```

This pattern cannot work if orchestrators lack Task tool access.

### Implication 2: All Multi-Level Delegation Fails

Affected agents (any agent that's supposed to spawn other agents):
- `review-orchestrator` - Cannot spawn reviewers
- `mozart` - Cannot spawn Einstein, Staff-Architect, Beethoven
- `impl-manager` - Cannot spawn implementation agents
- `orchestrator` - Cannot spawn any specialists
- Any future coordination agent

### Implication 3: Braintrust Workflow Cannot Function

The Braintrust skill depends on:
```
Mozart (level 1)
  ├── Einstein (level 2) ← Cannot spawn
  ├── Staff-Architect (level 2) ← Cannot spawn
  └── Beethoven (level 2) ← Cannot spawn
```

Without sub-subagent spawning, Braintrust reduces to single-agent analysis.

### Implication 4: Memory Concern May Be Unrelated

Original hypothesis: Memory creep (36GB) related to subprocess lifecycle.

New consideration: If subagents cannot spawn sub-subagents, the deep nesting that would cause memory accumulation may not be occurring. The 36GB may have different causes:
- Context window growth in the router session
- Transcript file accumulation
- Unrelated system issues

---

## Research Questions

### Question 1: Is This By Design?

Does Claude Code intentionally restrict Task tool access at nesting level 1+?

**Research vectors**:
- Claude Code documentation on subagent capabilities
- Anthropic SDK documentation on tool availability in spawned agents
- Claude Code source code (if accessible) for tool filtering logic

### Question 2: Is This Configurable?

Can subagent tool access be configured to include Task?

**Research vectors**:
- `settings.json` or `settings.local.json` options
- Subagent spawn parameters that control tool availability
- Environment variables affecting subagent capabilities

### Question 3: Is This a Recent Change?

Did subagent spawning ever work? Was this a regression?

**Research vectors**:
- Historical session transcripts showing successful sub-subagent spawning
- Claude Code changelog/release notes
- GitHub issues mentioning subagent nesting

### Question 4: What Do Agent Definitions Actually Control?

The `tools` field in agent definitions lists available tools, but runtime contradicts this.

**Research vectors**:
- How does Claude Code interpret agent definition `tools` field?
- Is there a separate runtime tool restriction mechanism?
- Does `subagent_type` affect tool availability?

### Question 5: Are There Workarounds?

**Potential approaches**:
- Flat orchestration (router does all coordination)
- Sequential spawning (router spawns each agent in series)
- Hook-based coordination (hooks communicate between agents)
- MCP-based agent spawning (bypass Claude Code's Task tool)

---

## Flat Orchestration Pattern (Proposed Alternative)

If subagents cannot spawn sub-subagents, redesign workflows to use flat structure:

### Current (Broken) Pattern

```
Router
  └── review-orchestrator (level 1)
        ├── frontend-reviewer (level 2) ← FAILS
        ├── backend-reviewer (level 2) ← FAILS
        └── standards-reviewer (level 2) ← FAILS
```

### Proposed Flat Pattern

```
Router (level 0)
  ├── frontend-reviewer (level 1) ← Works
  ├── backend-reviewer (level 1) ← Works
  ├── standards-reviewer (level 1) ← Works
  └── (Router synthesizes results directly)
```

**Trade-offs**:
- ✅ Actually works
- ✅ Simpler architecture
- ❌ Router becomes bloated with coordination logic
- ❌ No reusable orchestration patterns
- ❌ All CLAUDE.md and skill definitions need rewriting

### Proposed Flat Braintrust Pattern

```
Router (level 0)
  ├── Einstein (level 1) ← Spawned directly
  ├── Staff-Architect (level 1) ← Spawned in parallel
  └── (Router synthesizes, no Beethoven needed)
```

---

## Technical Deep Dive: What We Know About Claude Code Subagents

### Subagent Spawning Mechanism

From Claude Code documentation (inferred from behavior):

```javascript
Task({
  description: "...",
  subagent_type: "Explore" | "Plan" | "general-purpose",
  model: "haiku" | "sonnet" | "opus",
  prompt: "..."
})
```

- Creates a new Claude instance
- Passes the prompt as initial context
- Returns agent output when complete
- Returns `agentId` for potential resumption

### What Subagents Receive

Based on Mozart's behavior, subagents receive:
- Read, Write, Edit tools (file operations)
- Glob, Grep tools (search)
- TaskCreate, TaskUpdate, TaskList, TaskGet tools (TODO tracking)
- AskUserQuestion tool (user interaction)
- Bash tool (command execution) - unclear if always available
- **NOT Task tool** (cannot spawn further subagents)

### Subagent Type Implications

| subagent_type | Intended Use | Observed Tool Access |
|---------------|--------------|---------------------|
| Explore | Read-only exploration | Read, Glob, Grep, (no write?) |
| Plan | Planning with file access | Read, Write, Glob, Grep, TaskCreate |
| general-purpose | Full implementation | Read, Write, Edit, Bash, Glob, Grep |

**Note**: None of these appear to include Task tool at runtime, despite agent definitions.

---

## Transcript Evidence

### Mozart Session Transcript (Key Excerpt)

Mozart was invoked with:
```javascript
Task({
  description: "Mozart: Braintrust problem decomposition",
  subagent_type: "Plan",
  model: "opus",
  prompt: "AGENT: mozart\n\nBRAINTRUST INVOCATION\n..."
})
```

Mozart's tool usage in transcript:
- ✅ Read (accessed files)
- ✅ Glob (searched for files)
- ✅ Grep (searched content)
- ❌ Task (never invoked - "not available")

Mozart explicitly stated the limitation rather than attempting and failing.

### Review-Orchestrator Session Transcript (Key Excerpt)

Review-orchestrator was invoked similarly but exhibited different failure mode:
- Did not state Task was unavailable
- Instead used TaskCreate (TODO items)
- Or simulated work directly

**Hypothesis**: Different models/contexts may have different "awareness" of their tool limitations. Mozart (Opus) explicitly recognized the limitation. Review-orchestrator (Sonnet) may have been less aware and attempted workarounds.

---

## Action Items for Research

### Immediate (Before Implementation)

1. **Search Claude Code documentation** for "subagent", "nesting", "Task tool", "tool availability"
2. **Check Anthropic forums/Discord** for similar reports
3. **Review Claude Code GitHub issues** for subagent-related bugs
4. **Test with explicit tool list** - Does passing `tools: ["Task"]` in spawn enable it?

### If Limitation is Confirmed

1. **Redesign all orchestrator patterns** to flat structure
2. **Update CLAUDE.md** to document the limitation
3. **Rewrite Braintrust skill** for flat execution
4. **Rewrite /review skill** for direct spawning from router
5. **Add sharp edge** documenting the limitation

### If Workaround Exists

1. **Document the workaround** in conventions
2. **Update agent definitions** with correct configuration
3. **Test extensively** before relying on nested spawning

---

## Files Referenced in This Analysis

| File | Purpose |
|------|---------|
| `GAP-review-orchestrator-failure-2026-02-03.md` | Original problem report |
| `~/.claude/agents/review-orchestrator/review-orchestrator.md` | Agent definition |
| `~/.claude/agents/mozart/mozart.md` | Braintrust orchestrator definition |
| `~/.claude/skills/braintrust/SKILL.md` | Braintrust workflow specification |
| `~/.claude/CLAUDE.md` | Routing configuration |
| `~/.claude/routing-schema.json` | Tier and tool definitions |

---

## Conclusion

**The orchestrator pattern in GOgent-Fortress is architecturally incompatible with Claude Code's subagent model.**

Claude Code appears to restrict the Task tool to nesting level 0 (router only). Subagents cannot spawn sub-subagents. This limitation is not documented in agent definitions and contradicts the expected behavior.

**Recommended immediate action**: Research Claude Code documentation and community resources to confirm whether this is:
1. Intentional design limitation
2. Configurable restriction
3. Bug/regression

**Recommended fallback**: Redesign all multi-agent workflows to use flat structure with router-level coordination.

---

## Metadata

```yaml
analysis_id: braintrust-subagent-nesting-2026-02-04
session_id: current
agents_involved:
  - router (level 0, functional)
  - mozart (level 1, Task-restricted)
  - codebase-search (level 1, test - functional)
key_finding: "Subagents cannot spawn sub-subagents"
confidence: HIGH
impact: CRITICAL
requires_research: true
```

---

**END ANALYSIS**
