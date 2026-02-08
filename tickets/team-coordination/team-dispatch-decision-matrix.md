# Team Dispatch Decision Matrix

**Purpose:** Document when to use each execution path. Answers: "Given a workflow, should it use MCP spawn, Task(), or gogent-team-run?"

---

## 1. Execution Path Comparison

| Criterion | MCP `spawn_agent` | `Task()` (Foreground) | `gogent-team-run` (Background) |
|-----------|-------------------|----------------------|-------------------------------|
| **TUI blocks?** | Yes (parent waits) | Yes (parent waits) | No (returns in <15s) |
| **Cost visibility** | Real-time in parent context | Real-time in parent context | Post-hoc from config.json |
| **Error handling** | Parent catches, can react | Parent catches, can react | Errors in runner.log + config.json |
| **User interaction** | Possible (AskUserQuestion) | Possible (AskUserQuestion) | None after launch |
| **Max concurrency** | Limited by MCP server | Limited by Task nesting rules | Limited by budget + CPU |
| **Available at Level 0** | Yes | Yes | Yes (via Bash) |
| **Available at Level 1+** | Yes | No (blocked by hooks) | Yes (via Bash) |
| **Process isolation** | Separate CLI process | Separate CLI process | Separate CLI process + session leader |
| **Progress tracking** | Manual (parent polls) | Manual (parent polls) | `/team-status` reads config.json |
| **Cancellation** | Kill PID manually | Context cancellation | `/team-cancel` (SIGTERM cascade) |
| **Budget management** | Parent tracks manually | Parent tracks manually | Go binary tracks atomically |
| **Retry logic** | Parent must implement | Parent must implement | Go binary handles (max_retries) |
| **Wave sequencing** | Parent must implement | Parent must implement | Go binary handles automatically |

---

## 2. Per-Workflow Recommendation

| Workflow | Recommended Path | Rationale |
|----------|-----------------|-----------|
| **Braintrust** (full: 3 Opus agents) | **Background** (`gogent-team-run`) | 5-7 min execution; TUI must not freeze; wave sequencing needed |
| **Braintrust** (Einstein only) | **Foreground** (MCP `spawn_agent`) | Single agent, <2 min; user may want to interact with output immediately |
| **Code Review** (2-4 reviewers) | **Background** (`gogent-team-run`) | 2-3 min; 4 parallel agents; user can work meanwhile |
| **Code Review** (quick, 1 reviewer) | **Foreground** (`Task()`) | Single agent, <30s; team-run overhead not worth it |
| **Implementation** (multi-wave) | **Background** (`gogent-team-run`) | Variable duration; parallel waves; budget tracking needed |
| **Implementation** (single task) | **Foreground** (`Task()`) | Direct delegation; no wave/budget overhead needed |

---

## 3. Decision Flowchart

```
How many agents will run?
  |
  +-- 1 agent
  |     |
  |     +-- Needs user interaction? --> Foreground (Task or MCP spawn)
  |     +-- No interaction needed? --> Foreground (simpler, lower overhead)
  |
  +-- 2-3 agents, total <2 min
  |     |
  |     +-- User preference: "run in background" --> Background (gogent-team-run)
  |     +-- User preference: "I'll wait" --> Foreground (MCP spawn in parallel)
  |     +-- Default --> Background (better UX)
  |
  +-- 3+ agents OR estimated >2 min
        |
        +-- Needs user interaction DURING execution?
        |     |
        |     +-- Yes --> Foreground interview FIRST, then background dispatch
        |     |           (Mozart pattern: Task(opus) for interview -> team-run)
        |     |
        |     +-- No --> Direct background dispatch
        |               (Review pattern: router generates config -> team-run)
        |
        +-- Has wave dependencies?
              |
              +-- Yes --> Background (gogent-team-run handles wave sequencing)
              +-- No --> Background (parallel execution, budget tracking)
```

---

## 4. Per-Workflow Dispatch Patterns

### Braintrust (Full)

```
Router
  |
  +-- Task(opus) --> Mozart (foreground, ~30s)
        |
        +-- Interview (Q1-Q4)
        +-- Scout (Task(haiku), ~10s)
        +-- Generate config.json + 3 stdin files
        +-- gogent-team-run "$team_dir"  (background)
        +-- Return: "Team dispatched. Use /team-status"
```

### Code Review

```
Router (no LLM orchestrator needed)
  |
  +-- git diff --staged (capture files)
  +-- Classify files -> select reviewers
  +-- Generate config.json + N stdin files
  +-- gogent-team-run "$team_dir"  (background)
  +-- Return: "Review dispatched. Use /team-status"
```

### Implementation

```
Router (no LLM orchestrator needed)
  |
  +-- Read specs.md
  +-- gogent-plan-impl (Go binary: parse + wave DAG + generate config)
  +-- gogent-team-run "$team_dir"  (background)
  +-- Return: "Implementation dispatched. Use /team-status"
```

---

## 5. Fallback Strategy

If `gogent-team-run` binary is missing or fails to launch:

1. **Detection:** Check binary exists on PATH before attempting launch
2. **Fallback:** Revert to foreground MCP spawn pattern
3. **Feature flag:** `settings.json -> "use_team_pattern": true/false`
4. **Both paths produce same output format:** Same files, same locations, same schema

```json
// settings.json
{
  "use_team_pattern": true,
  "team_pattern_fallback": "mcp_spawn"
}
```

**Fallback routing:**
- `use_team_pattern: true` + binary exists -> background (gogent-team-run)
- `use_team_pattern: true` + binary missing -> fallback to foreground + warn user
- `use_team_pattern: false` -> foreground (MCP spawn / Task)

---

## 6. Cost Comparison

| Path | Marginal cost | Why |
|------|--------------|-----|
| **Background (gogent-team-run)** | ~$0 | Go binary, no LLM tokens for orchestration |
| **Foreground (MCP spawn)** | ~$0.01-0.05/spawn | Envelope overhead per spawn call |
| **Foreground (Task)** | ~$0.01-0.10/spawn | Context inheritance overhead |

**For 4+ agents, background is strictly cheaper:**
- No orchestrator LLM maintaining state between spawns
- No context window growth from collecting agent outputs
- Budget tracking in Go is free (vs. LLM reasoning about budget)

**Break-even point:** ~2 agents. Below 2 agents, the overhead of generating config files exceeds the savings from eliminating orchestrator LLM tokens.

---

## 7. Coexistence Rules

Both paths MUST coexist because:
1. **TC-019 confirmed** SDK supports concurrent queries (MCP path remains viable)
2. **Single-agent workflows** don't benefit from team-run overhead
3. **Interactive workflows** (Mozart interview) need foreground for the interactive phase
4. **Rollback safety** requires the old path to remain functional

**Coexistence boundary:**
- Config generation is the same regardless of execution path
- The decision point is: "who executes the config?" (Go binary vs LLM orchestrator)
- Both paths read the same schemas, produce the same output format
- `/team-status` and `/team-result` work for background path only

---

## 8. Migration Timeline

| Phase | What Changes | Both Paths Active? |
|-------|-------------|-------------------|
| **Phase 1** (TC-013a: Review) | Review gains background option | Yes — feature flag |
| **Phase 2** (TC-013b: Braintrust) | Mozart gains background dispatch after interview | Yes — feature flag |
| **Phase 3** (TC-013c: Implementation) | Impl-manager gains background option | Yes — feature flag |
| **Phase 4** (Post-validation) | Default flips to background | Yes — old path available |
| **Phase 5** (Deprecation) | Remove foreground orchestrator agents | No — background only |

Phase 5 is optional and should only happen after sustained stability (weeks, not days).
