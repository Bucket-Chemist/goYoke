# SYNTHESIS REPORT: Team Coordination Review Findings

> **Synthesized by**: standards-reviewer (from backend, frontend, architecture domain reviews)
> **Date**: 2026-02-06
> **Input**: 39 total findings (14 backend + 8 frontend + 17 architecture)

---

## 1. Cross-Reviewer Agreement (Findings Raised by 2+ Reviewers)

| Issue | Reviewers | Severity | Tickets | Summary |
|-------|-----------|----------|---------|---------|
| **TC-009 Bottleneck** | Arch, Backend | CRITICAL | TC-008, TC-009 | TC-009 (2-3d) blocks TC-008 (5-7d), creating Phase 1→2 bottleneck |
| **PID File Race Condition** | Backend, Arch | CRITICAL | TC-004, TC-016 | TOCTOU race: two processes check, find no file, both write |
| **CLI Output Format Unverified** | Backend, Arch | CRITICAL | TC-005, TC-008 | Entire cost tracking is load-bearing assumption with zero validation |
| **Budget Check Race Condition** | Backend, Arch | CRITICAL | TC-002, TC-008 | Budget check outside critical section allows overspend |
| **Task Access Policy Not Implemented** | Frontend, Arch | CRITICAL | TC-007 | TC-007 documents intent but no ticket implements gogent-validate enforcement |
| **Heartbeat Granularity** | Backend, Arch | HIGH | TC-004, TC-008, TC-012 | 30s/60s thresholds = 40s detection lag for hung agents |
| **Config.json Consistency** | Backend, Arch | HIGH | TC-002, TC-008, TC-011 | Concurrent updates risk lost updates; direct budget bypass |
| **Agent Timeout Not Enforced** | Backend, Arch | HIGH | TC-008, TC-009 | cmd.Wait() blocks indefinitely; timeout_ms field defined but unused |
| **Wave Failure Semantics Undefined** | Backend, Arch | HIGH | TC-003, TC-008, TC-010 | Does failed member block wave? Inter-wave script expects all outputs |
| **Schema Field Inconsistencies** | Backend, Arch | HIGH | TC-008, TC-009, TC-013 | Budget nested vs flat; stdin file naming agent ID vs member name |
| **Child Process Cleanup Gap** | Backend, Arch | HIGH | TC-003, TC-004, TC-008 | PID registered after Start(); cancellation window misses processes |
| **Project Root Resolution Brittle** | Backend, Arch | MEDIUM | TC-006, TC-013 | Env vars not set by TUI; pwd fallback wrong in subdirectories |

---

## 2. Unique Findings Per Domain

### Backend Only (7)

| Severity | Finding | Ticket(s) |
|----------|---------|-----------|
| CRITICAL | WaitGroup recursion footgun (caller must call wg.Add(1) once) | TC-003, TC-008 |
| HIGH | Cost extraction error levels inadequate (distinguish no-field vs malformed) | TC-005, TC-008 |
| MEDIUM | Missing agent validation on startup (typos get defaults) | TC-008, TC-014 |
| MEDIUM | CLI argument ordering inconsistency | TC-001, TC-008, TC-014 |
| MEDIUM | Missing crash safety test (SIGKILL mid-write) | TC-002, TC-008, TC-011 |
| MEDIUM | Config atomic write recovery unspecified (abandoned .tmp files) | TC-008, TC-004 |
| LOW | Inconsistent error message formatting | TC-004, TC-012 |

### Frontend Only (5)

| Severity | Finding | Ticket(s) |
|----------|---------|-----------|
| CRITICAL | SDK concurrency investigation not in TC-015 acceptance criteria | TC-015 |
| HIGH | TC-015 line number references approximate (update note) | TC-015 |
| HIGH | Event flow sequence diagram missing | TC-015 |
| HIGH | Session directory discovery error handling missing | TC-012 |
| MEDIUM | Unicode status indicators no ASCII fallback | TC-012 |

### Architecture Only (5)

| Severity | Finding | Ticket(s) |
|----------|---------|-----------|
| CRITICAL | Mozart interview phase not specified | TC-013 |
| HIGH | No build/deployment story (Makefile) | All |
| HIGH | TC-011 missing TC-003 dependency | TC-011 |
| HIGH | TC-013 orchestrator rewrites need separate design docs | TC-013 |
| HIGH | Wave computation algorithm not specified for cycles | TC-013 |

---

## 3. Contradictions Between Reviewers

| Topic | Reviewers | Resolution |
|-------|-----------|-----------|
| Budget field structure | Backend (consistency issue) vs Arch (explicit contradiction) | **Arch is correct**: TC-009 nested `"budget": {...}` vs TC-008 flat `BudgetMaxUSD`. Standardize to one schema. |
| Stdin file naming | Backend (consistency) vs Arch (contradiction) | **Arch is correct**: TC-009 `stdin_einstein.json` vs TC-008 `stdin_{member.Name}.json`. Define naming per workflow type. |
| Heartbeat thresholds | Backend (10s/30s) vs Arch (60s/120s/180s) | **No contradiction**: Backend suggests tighter intervals, Arch defines remediation policy. Combine both. |

**No severe contradictions.** All reviewers align on critical issues.

---

## 4. Action Items (Sorted by Priority)

### CRITICAL — Blocking Implementation Start

| # | Action | Ticket(s) | Effort |
|---|--------|-----------|--------|
| 1 | Create TC-017: gogent-validate Level 2 enforcement (allow haiku/sonnet, block opus) | NEW | 1d |
| 2 | Run TC-005 verification: test `claude -p --output-format json`, document actual format | TC-005 | 0.5d |
| 3 | Split TC-009 into TC-009a (minimal templates, 1d) + TC-009b (full schemas, 1-2d) | TC-009 | — |
| 4 | Specify Mozart interview protocol: questions, decision points, config field mapping | TC-013 | 1d |
| 5 | Fix PID file race: use O_CREAT\|O_EXCL atomic create-or-fail | TC-004, TC-016 | 0.5d |

### CRITICAL — Non-Blocking

| # | Action | Ticket(s) | Effort |
|---|--------|-----------|--------|
| 6 | Move budget check inside updateMember() critical section | TC-002, TC-008 | 0.5d |
| 7 | Add SDK concurrency investigation to TC-015 acceptance criteria as Phase 1 gate | TC-015 | 0d |
| 8 | Add WaitGroup safety comment + consider extracting retry loop | TC-003, TC-008 | 0.5d |

### HIGH

| # | Action | Ticket(s) | Effort |
|---|--------|-----------|--------|
| 9 | Standardize budget schema: choose nested OR flat; add unmarshal validation test | TC-008, TC-009, TC-013 | 0.5d |
| 10 | Define stdin file naming: braintrust=agent ID, implementation=task ID | TC-008, TC-009, TC-013 | 0.5d |
| 11 | Implement agent timeout via context.WithTimeout() + exec.CommandContext() | TC-008, TC-009 | 1d |
| 12 | Reduce heartbeat to 10s; define policy: 60s warn, 120s strong warn, 180s auto-cancel | TC-008, TC-012 | 0.5d |
| 13 | Move budget deduction inside updateMember(); audit all direct config access | TC-002, TC-008 | 0.5d |
| 14 | Make process registration atomic with cmd.Start(); add finalizer goroutine | TC-003, TC-004, TC-008 | 1d |
| 15 | Add TC-003 to TC-011 blocked_by | TC-011 | 0d |
| 16 | Create orchestrator rewrite design docs (Mozart, ReviewOrch, ImplMgr) | TC-013 | 2-3d |
| 17 | Implement proper project root detection: env → git root → ask user | TC-006, TC-013 | 1d |
| 18 | Specify topological sort + cycle detection for wave computation | TC-013, TC-011 | 1d |
| 19 | Add session directory discovery error handling + TUI integration verification | TC-012 | 0.5d |
| 20 | Distinguish cost extraction errors: ErrNoCostField vs ErrMalformedJSON | TC-005, TC-008 | 1d |
| 21 | Add event flow sequence diagram to TC-015 | TC-015 | 0.5d |
| 22 | Add Makefile/build story to TC-008 acceptance criteria | TC-008 | 0.5d |

### MEDIUM

| # | Action | Ticket(s) | Effort |
|---|--------|-----------|--------|
| 23 | Verify CLI flag order with `claude --help`; standardize | TC-001, TC-008, TC-014 | 0.5d |
| 24 | Add agent validation: fail if agent not found in agents-index.json | TC-008, TC-014 | 0.5d |
| 25 | Add ASCII fallback status indicators | TC-012 | 0.5d |
| 26 | Add review conflict resolution logic to /team-result | TC-012 | 0.5d |
| 27 | Add crash safety integration test (SIGKILL mid-write → restart) | TC-008, TC-011 | 1d |
| 28 | Document atomic write recovery: check for .tmp files on startup | TC-008, TC-004 | 0.5d |
| 29 | Standardize fallback defaults across TUI and Go binary paths | TC-014 | 0.5d |
| 30 | Mark stdin schema optional fields explicitly | TC-009 | 0.5d |
| 31 | Document PID check limitation (same-machine only); add timestamp | TC-016 | 0.5d |

### LOW

| # | Action | Ticket(s) | Effort |
|---|--------|-----------|--------|
| 32 | Establish error message convention (prefix, severity, next steps) | TC-004, TC-012 | 0.5d |
| 33 | Add pre-synthesis.md validation test against beethoven schema | TC-010, TC-009 | 0.5d |
| 34 | Add StatusLine background team status integration | TC-012, TC-015 | 0.5d |

---

## 5. Missing Tickets

| New Ticket | Purpose | Severity | Effort | Blocks |
|-----------|---------|----------|--------|--------|
| **TC-017** | gogent-validate Level 2 enforcement (allow Task haiku/sonnet, block opus) | CRITICAL | 1d | TC-008 testing |
| **TC-009a** | Minimal team templates (braintrust.json + review.json MVP) | CRITICAL | 1d | TC-008 start |
| **TC-018** | Mozart interview protocol specification | CRITICAL | 1d | TC-013 |
| **TC-019** | SDK concurrency investigation (concurrent query() support) | CRITICAL | 1-2d | TC-015 |
| **TC-020** | Orchestrator rewrite design docs (3 separate docs) | MEDIUM | 2-3d | TC-013 clarity |

---

## 6. Verdict

### Can implementation proceed?

**CONDITIONAL** — 5 critical blockers must resolve first (~4 days of work).

---

## 7. Critical Blocker Specifications

### Blocker 1: TC-005 — Verify CLI JSON Output Format (0.5d)

**Why blocking**: The entire cost tracking system (TC-008, TC-012, TC-013) depends on parsing a `cost_usd` field from `claude -p --output-format json`. This format is undocumented. If the field doesn't exist or has a different name, cost tracking and budget enforcement are broken.

**Exact steps**:

```bash
# Step 1: Capture raw output from multiple scenarios
echo "What is 2+2?" | claude -p --output-format json > /tmp/claude-short.json 2>&1
echo "Explain quantum computing in detail" | claude -p --output-format json > /tmp/claude-long.json 2>&1
echo "hello" | claude -p --output-format json --model invalid-model > /tmp/claude-error.json 2>&1
echo "hello" | claude -p --output-format json --max-budget-usd 0.01 > /tmp/claude-budget.json 2>&1
claude --version > /tmp/claude-version.txt

# Step 2: Extract cost-related fields
for f in /tmp/claude-short.json /tmp/claude-long.json /tmp/claude-error.json /tmp/claude-budget.json; do
  echo "=== $f ==="
  cat "$f" | python3 -c "import json,sys; d=json.load(sys.stdin); print(json.dumps(d, indent=2))" 2>/dev/null || echo "(not valid JSON)"
done

# Step 3: Search for cost field names
grep -i "cost\|usage\|billing\|price\|token" /tmp/claude-short.json /tmp/claude-long.json
```

**What to document** (write to `cmd/gogent-team-run/docs/claude-cli-output-format.md`):
- Full JSON structure with all fields
- Exact cost field name(s): `cost_usd`? `total_cost_usd`? nested under `usage`?
- Field type: float64 or string?
- Error representation format
- What `--max-budget-usd` returns when exceeded

**Decision gate**:
- If cost field EXISTS → document field name, update TC-008 `extractCostFromCLIOutput()` spec to match
- If cost field MISSING → escalate: redesign cost tracking to use `--max-budget-usd` CLI flag as primary enforcement instead of runtime parsing. Update TC-008, TC-012, TC-013 accordingly.

**Existing reference**: `packages/tui/src/mcp/spawnAgent.ts` has `parseCliOutput()` that tries `cost_usd || total_cost_usd`. Find it and verify which field name works.

---

### Blocker 2: TC-017 — gogent-validate Level 2 Enforcement (1d)

**Why blocking**: TC-007 documents that team-spawned agents (Level 2) should use Task(haiku/sonnet) but not Task(opus). However, the current `gogent-validate` code at `cmd/gogent-validate/main.go:101` blocks ALL Task() calls when `nestingLevel > 0`. No ticket implements the selective enforcement. Without this, team-spawned Einstein/Staff-Architect cannot delegate to haiku scouts.

**Current code** (`cmd/gogent-validate/main.go:17-19, 97-109`):

```go
const (
    MAX_TASK_NESTING_LEVEL = 0 // Strict: Only Router (Level 0) can use Task()
)

// Line 101:
if nestingLevel > MAX_TASK_NESTING_LEVEL {
    logNestingBlock(event, nestingLevel, isExplicit)
    response := routing.BlockResponseForNesting(nestingLevel)
    outputJSON(response)
    return
}
```

**Required change** — replace the block at lines 97-109 with:

```go
if nestingLevel > MAX_TASK_NESTING_LEVEL {
    // Level 1+: Allow haiku/sonnet, block opus
    if taskInput, err := routing.ParseTaskInput(event.ToolInput); err == nil {
        if taskInput.Model == "opus" {
            logNestingBlock(event, nestingLevel, isExplicit)
            response := routing.BlockResponse(
                fmt.Sprintf(
                    "Task(opus) blocked at nesting level %d. Use Task(haiku) or Task(sonnet) instead.",
                    nestingLevel,
                ),
            )
            outputJSON(response)
            return
        }
        // Task(haiku) and Task(sonnet) allowed at Level 1+
    } else {
        // Cannot parse model — block defensively
        logNestingBlock(event, nestingLevel, isExplicit)
        response := routing.BlockResponseForNesting(nestingLevel)
        outputJSON(response)
        return
    }
}
```

**Also update** `pkg/routing/task_validation.go:246` — the `BlockResponseForNesting` message should say "Task(opus)" not "Task()" since haiku/sonnet are now allowed.

**Files to modify**:

| File | Change |
|------|--------|
| `cmd/gogent-validate/main.go` | Replace lines 97-109 with model-aware check |
| `pkg/routing/task_validation.go` | Update `BlockResponseForNesting` message |
| `cmd/gogent-validate/main_test.go` | Add tests: Level 2 + haiku → allow, Level 2 + opus → block, Level 2 + no model → block |

**Test cases**:

| Test | Nesting Level | Model | Expected |
|------|---------------|-------|----------|
| Router spawns opus | 0 | opus | ALLOW |
| Team agent spawns haiku | 2 | haiku | ALLOW |
| Team agent spawns sonnet | 2 | sonnet | ALLOW |
| Team agent spawns opus | 2 | opus | BLOCK |
| Team agent no model specified | 2 | (empty) | BLOCK (defensive) |
| MCP agent spawns haiku | 1 | haiku | ALLOW |

---

### Blocker 3: TC-009a — Minimal Team Templates (1d)

**Why blocking**: TC-008 needs schemas to parse config.json. The full TC-009 (2-3d, 15+ schema files) is a bottleneck. Ship a minimal subset first so TC-008 can start.

**TC-009a scope** — create these 4 files ONLY:

**File 1**: `.claude/schemas/teams/braintrust.json` (minimal):

```json
{
  "team_name": "braintrust-{timestamp}",
  "workflow_type": "braintrust",
  "project_root": "/absolute/path",
  "session_id": "{uuid}",
  "created_at": "{ISO-8601}",
  "background_pid": null,
  "budget_max_usd": 5.0,
  "budget_remaining_usd": 5.0,
  "waves": [
    {
      "wave_number": 1,
      "members": [
        {
          "name": "einstein",
          "agent_id": "einstein",
          "model": "opus",
          "stdin_file": "stdin_einstein.json",
          "stdout_file": "stdout_einstein.json",
          "status": "pending",
          "pid": null,
          "cost_usd": 0.0,
          "retry_count": 0,
          "max_retries": 1,
          "timeout_ms": 600000
        },
        {
          "name": "staff-architect",
          "agent_id": "staff-architect-critical-review",
          "model": "opus",
          "stdin_file": "stdin_staff-architect.json",
          "stdout_file": "stdout_staff-architect.json",
          "status": "pending",
          "pid": null,
          "cost_usd": 0.0,
          "retry_count": 0,
          "max_retries": 1,
          "timeout_ms": 600000
        }
      ],
      "on_complete_script": "gogent-team-prepare-synthesis"
    },
    {
      "wave_number": 2,
      "members": [
        {
          "name": "beethoven",
          "agent_id": "beethoven",
          "model": "opus",
          "stdin_file": "stdin_beethoven.json",
          "stdout_file": "stdout_beethoven.json",
          "status": "pending",
          "pid": null,
          "cost_usd": 0.0,
          "retry_count": 0,
          "max_retries": 1,
          "timeout_ms": 600000
        }
      ],
      "on_complete_script": null
    }
  ]
}
```

**File 2**: `.claude/schemas/teams/review.json` (minimal):

```json
{
  "team_name": "review-{timestamp}",
  "workflow_type": "review",
  "project_root": "/absolute/path",
  "session_id": "{uuid}",
  "created_at": "{ISO-8601}",
  "background_pid": null,
  "budget_max_usd": 2.0,
  "budget_remaining_usd": 2.0,
  "waves": [
    {
      "wave_number": 1,
      "members": [
        {
          "name": "backend-reviewer",
          "agent_id": "backend-reviewer",
          "model": "haiku",
          "stdin_file": "stdin_backend-reviewer.json",
          "stdout_file": "stdout_backend-reviewer.json",
          "status": "pending",
          "pid": null,
          "cost_usd": 0.0,
          "retry_count": 0,
          "max_retries": 1,
          "timeout_ms": 120000
        },
        {
          "name": "frontend-reviewer",
          "agent_id": "frontend-reviewer",
          "model": "haiku",
          "stdin_file": "stdin_frontend-reviewer.json",
          "stdout_file": "stdout_frontend-reviewer.json",
          "status": "pending",
          "pid": null,
          "cost_usd": 0.0,
          "retry_count": 0,
          "max_retries": 1,
          "timeout_ms": 120000
        },
        {
          "name": "standards-reviewer",
          "agent_id": "standards-reviewer",
          "model": "haiku",
          "stdin_file": "stdin_standards-reviewer.json",
          "stdout_file": "stdout_standards-reviewer.json",
          "status": "pending",
          "pid": null,
          "cost_usd": 0.0,
          "retry_count": 0,
          "max_retries": 1,
          "timeout_ms": 120000
        },
        {
          "name": "architect-reviewer",
          "agent_id": "architect-reviewer",
          "model": "haiku",
          "stdin_file": "stdin_architect-reviewer.json",
          "stdout_file": "stdout_architect-reviewer.json",
          "status": "pending",
          "pid": null,
          "cost_usd": 0.0,
          "retry_count": 0,
          "max_retries": 1,
          "timeout_ms": 120000
        }
      ],
      "on_complete_script": null
    }
  ]
}
```

**File 3**: `.claude/schemas/teams/common-types.md` — document the field contract:

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `team_name` | string | yes | `{workflow}-{unix_timestamp}` |
| `workflow_type` | enum | yes | `braintrust`, `review`, `implementation` |
| `project_root` | string | yes | Absolute path, must exist |
| `budget_max_usd` | float64 | yes | Top-level, NOT nested (resolves schema contradiction) |
| `budget_remaining_usd` | float64 | yes | Top-level, NOT nested |
| `background_pid` | int/null | yes | Written by gogent-team-run after start |
| `waves[].members[].name` | string | yes | Unique within team; used for stdin/stdout filenames |
| `waves[].members[].agent_id` | string | yes | Must exist in agents-index.json |
| `waves[].members[].stdin_file` | string | yes | `stdin_{name}.json` (name, not agent_id) |

**File 4**: `.claude/schemas/teams/README.md` — one-paragraph explanation of how templates work.

**Critical design decision resolved**: Budget fields are **top-level flat** (`budget_max_usd`, `budget_remaining_usd`), NOT nested under a `budget` object. This resolves the TC-009/TC-008/TC-013 contradiction flagged by all 3 reviewers. Update TC-008's `TeamConfig` Go struct to match.

**TC-009b** (remaining work, non-blocking): Full stdin/stdout schemas for all agent types, implementation.json template, JSON Schema formal definitions.

---

### Blocker 4: Mozart Interview Protocol (1d)

**Why blocking**: TC-013 says "Mozart conducts interview" but never specifies what questions to ask, how answers map to team config, or what decision points exist. Without this, Mozart generates garbage configs.

**Interview protocol** (add to TC-013 as new section "Phase 1: Interview Protocol"):

**Question 1 — Problem Statement** (always asked):
```
"What problem or question do you want the Braintrust to analyze?"
```
→ Maps to: `stdin_einstein.json:task.problem_statement` and `stdin_staff-architect.json:task.problem_statement`

**Question 2 — Scope** (always asked):
```
"Which files or areas of the codebase are relevant? (Or should I scout first?)"
```
→ Decision point:
  - User provides files → Mozart reads them, includes in stdin `context.relevant_files[]`
  - User says "scout" → Mozart spawns haiku scout, waits ~10s, includes `scout_metrics.json` path in stdin

**Question 3 — Team Composition** (optional, only if ambiguous):
```
"Should I include both Einstein (theoretical) and Staff-Architect (practical review), or just one?"
```
→ Default: both (full braintrust). If user says "just Einstein" → single-member Wave 1, skip Staff-Architect, skip inter-wave synthesis.

**Question 4 — Budget** (only if user has budget concerns):
```
"Default budget is $5.00 for the team. Want to adjust?"
```
→ Maps to: `config.json:budget_max_usd` and `budget_remaining_usd`

**Decision flow**:

```
Q1 (problem) → always
Q2 (scope)   → if "scout" → spawn haiku scout → wait → continue
Q3 (team)    → only if problem is narrowly scoped (skip for broad analysis)
Q4 (budget)  → only if user mentions cost concerns
─────────────────────────────
Output: Problem Brief (confirm with user)
─────────────────────────────
User confirms → generate config.json + stdin files → launch gogent-team-run
User modifies → update Brief → re-confirm
```

**Config field mapping from interview**:

| Interview Output | Config Field | Stdin Field |
|-----------------|-------------|-------------|
| Problem statement | — | `task.problem_statement` (all agents) |
| Relevant files | — | `context.relevant_files[]` (all agents) |
| Scout metrics path | — | `reads_from.scout_metrics` (einstein only) |
| Team composition | `waves[0].members[]` | — |
| Budget | `budget_max_usd` | — |
| Project root | `project_root` | `paths.project_root` (all agents) |
| Session ID | `session_id` | — |

---

### Blocker 5: PID File Race Fix (0.5d)

**Why blocking**: Current `acquirePIDFile()` in TC-004 (lines 112-132) has a TOCTOU race: two processes can both check "file exists?", both get "no", both write. One overwrites the other silently.

**Fix**: Replace check-then-write with atomic exclusive create.

**Replace TC-004's `acquirePIDFile()` with**:

```go
func acquirePIDFile(teamDir string) (*PIDFile, error) {
    pidPath := filepath.Join(teamDir, PIDFileName)

    // Attempt atomic exclusive create (fails if file already exists)
    f, err := os.OpenFile(pidPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
    if err != nil {
        if os.IsExist(err) {
            // File exists — check if process is still alive
            data, readErr := os.ReadFile(pidPath)
            if readErr != nil {
                return nil, fmt.Errorf("read existing PID file: %w", readErr)
            }
            existingPID, _ := strconv.Atoi(strings.TrimSpace(string(data)))
            if existingPID > 0 && processExists(existingPID) {
                return nil, fmt.Errorf("team already running (PID %d)", existingPID)
            }
            // Stale PID — remove and retry ONCE
            log.Printf("[WARN] Stale PID file (PID %d dead), reclaiming", existingPID)
            os.Remove(pidPath)
            f, err = os.OpenFile(pidPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
            if err != nil {
                return nil, fmt.Errorf("reclaim PID file: %w (another process beat us)", err)
            }
        } else {
            return nil, fmt.Errorf("open PID file: %w", err)
        }
    }

    // Write our PID
    pid := os.Getpid()
    fmt.Fprintf(f, "%d\n", pid)
    f.Close()

    return &PIDFile{path: pidPath, pid: pid}, nil
}
```

**Key change**: `os.O_CREATE|os.O_EXCL` is atomic at the kernel level — two processes cannot both succeed. The loser gets `os.ErrExist`.

**Update in TC-004.md**: Replace the `acquirePIDFile()` code block at lines 112-132.
**Update in TC-016.md**: Reference TC-004's atomic pattern, remove the check-then-write version.

**Test cases** (add to TC-011):

| Test | Scenario | Expected |
|------|----------|----------|
| `TestPIDFile_AtomicCreate` | No PID file exists | Created successfully |
| `TestPIDFile_AliveProcess` | PID file with alive PID | Error "already running" |
| `TestPIDFile_StaleReclaim` | PID file with dead PID | Warning logged, file reclaimed |
| `TestPIDFile_RaceReclaim` | Two goroutines call acquirePIDFile simultaneously | Exactly one succeeds, other gets error |

---

## 8. Missing Ticket Specifications

### TC-017: gogent-validate Level 2 Enforcement

**Priority**: CRITICAL | **Effort**: 1d | **Blocks**: TC-008 testing

See Blocker 2 above for exact code changes.

**Acceptance criteria**:
- [ ] `go test ./cmd/gogent-validate/...` passes with new test cases (6 cases from table above)
- [ ] `go test -race ./cmd/gogent-validate/...` passes
- [ ] Manual test: `GOGENT_NESTING_LEVEL=2 claude -p "Use Task(haiku) to count to 3"` succeeds
- [ ] Manual test: `GOGENT_NESTING_LEVEL=2 claude -p "Use Task(opus) to count to 3"` blocked with clear message

---

### TC-009a: Minimal Team Templates

**Priority**: CRITICAL | **Effort**: 1d | **Blocks**: TC-008

See Blocker 3 above for exact file contents.

**Acceptance criteria**:
- [ ] `braintrust.json` and `review.json` templates exist in `.claude/schemas/teams/`
- [ ] `common-types.md` documents all field contracts
- [ ] Budget fields are flat (NOT nested) — resolves cross-ticket contradiction
- [ ] Templates can be hand-filled for a real team run (human-readable test)
- [ ] Go struct `TeamConfig` in TC-008 unmarshals both templates without error

---

### TC-018: Mozart Interview Protocol

**Priority**: CRITICAL | **Effort**: 1d | **Blocks**: TC-013

See Blocker 4 above for complete protocol.

**Acceptance criteria**:
- [ ] Interview protocol added to TC-013 as "Phase 1: Interview Protocol" section
- [ ] All 4 questions documented with decision flow
- [ ] Config field mapping table complete (interview output → config.json field → stdin field)
- [ ] Decision flow handles scout-first path
- [ ] Decision flow handles single-agent (Einstein-only) path

---

### TC-019: SDK Concurrency Investigation

**Priority**: CRITICAL | **Effort**: 1-2d | **Blocks**: TC-015

**Exact investigation steps**:

```typescript
// Test 1: Can we call query() twice concurrently?
import { query } from '@anthropic-ai/claude-agent-sdk';

const stream1 = query({ prompt: "Count to 10 slowly" });
const stream2 = query({ prompt: "Count to 5 slowly" });

// If both complete without error → concurrent queries supported
// If one errors with "session busy" → single-session semantics
// If one queues behind the other → sequential execution only
const [result1, result2] = await Promise.all([
  collectStream(stream1),
  collectStream(stream2),
]);
```

```typescript
// Test 2: Does a second query() cancel the first?
const stream1 = query({ prompt: "Count to 100" });
setTimeout(async () => {
  const stream2 = query({ prompt: "Say hello" });
  // Check: did stream1 get cancelled? Did stream2 work?
}, 1000);
```

**Decision gate**:
- SDK supports concurrent queries → TC-015 Phase 2 proceeds as designed
- SDK does NOT support concurrent queries → TC-015 needs alternative: separate Node.js child processes per query, or SDK feature request to Anthropic

**Acceptance criteria**:
- [ ] Test results documented with exact SDK version
- [ ] Behavior for concurrent `query()` calls documented (works / errors / queues)
- [ ] If unsupported: alternative approach documented in TC-015

---

### TC-020: Orchestrator Rewrite Design Docs

**Priority**: MEDIUM | **Effort**: 2-3d | **Blocks**: TC-013 clarity

Create 3 files:
- `tickets/team-coordination/Mozart-rewrite.md` — full prompt template for Mozart Phase 5 (team dispatch)
- `tickets/team-coordination/ReviewOrch-rewrite.md` — git diff capture → config.json → stdin generation → launch
- `tickets/team-coordination/ImplMgr-rewrite.md` — specs.md parsing → DAG → wave computation → config.json → launch

Each doc must include:
1. Before/after prompt comparison
2. Config.json generation logic (field by field)
3. Stdin file generation (absolute path resolution)
4. Launch command with exact arguments
5. At least 2 test cases (happy path + error path)

---

## 9. Recommended Schedule

```
Week 1: CRITICAL BLOCKERS (4 days, all parallel)
  Day 1-2: TC-005 verify + TC-017 (gogent-validate) + TC-009a (minimal schemas)
  Day 3-4: TC-018 (Mozart interview) + Blocker 5 (PID race fix)
  → TC-008 unblocked end of Week 1

Week 2-3: PHASE 2 (TC-008, 5-7 days)
  Parallel: TC-003, TC-004, TC-006, TC-014, TC-019 (SDK investigation)
  Parallel: TC-020 (orchestrator design docs)

Week 3-4: PHASE 2 COMPLETION
  TC-010 (inter-wave binary) + TC-011 (unit tests)
  TC-009b (full schemas, non-blocking)
  TC-015 begins (if TC-019 confirms SDK support)

Week 4-5: PHASE 3-4
  TC-012 (slash commands) → TC-013 (orchestrator rewrites)
```

**Total revised timeline: ~5-6 weeks** (was 16-23 days, now includes critical blocker resolution + new tickets).

---

**All risks are resolvable. No architectural redesign needed. The ticket suite is sound — it just needs the gaps filled before execution.**

---

## 10. Action Item Specifications (Items 6-34)

Items 1-5 are specified in Section 7 (Critical Blocker Specifications). This section specifies the remaining 29 action items.

---

### 10.1 CRITICAL — Non-Blocking

#### Item 6: Move Budget Check Inside Critical Section

**Why**: TC-008 `wave.go` lines 173-179 check budget OUTSIDE the mutex, then release it before spawning. Two goroutines can both read "$1.50 remaining", both spawn, and overspend.

**Current code** (TC-008 `wave.go` lines 171-183):
```go
for memberIdx := range wave.Members {
    tr.configMu.Lock()
    if tr.config.BudgetRemainingUSD <= 0 {  // CHECK here
        tr.configMu.Unlock()
        break
    }
    tr.configMu.Unlock()  // RELEASE here — race window opens
    wg.Add(1)
    go tr.spawnAndWait(waveIdx, memberIdx, &wg)  // SPAWN here
}
```

**Required change**: Replace the pre-spawn budget check with a `tryReserveBudget()` method that atomically checks AND reserves a per-agent budget estimate:

```go
// In config.go — add to TeamRunner:
func (tr *TeamRunner) tryReserveBudget(estimatedCost float64) bool {
    tr.configMu.Lock()
    defer tr.configMu.Unlock()
    if tr.config.BudgetRemainingUSD < estimatedCost {
        return false
    }
    tr.config.BudgetRemainingUSD -= estimatedCost  // Reserve
    tr.writeConfigAtomic()
    return true
}

// After agent completes, reconcile actual vs estimated:
func (tr *TeamRunner) reconcileCost(estimatedCost, actualCost float64) {
    tr.configMu.Lock()
    defer tr.configMu.Unlock()
    tr.config.BudgetRemainingUSD += estimatedCost  // Return reservation
    tr.config.BudgetRemainingUSD -= actualCost     // Deduct actual
    tr.writeConfigAtomic()
}
```

**In wave.go** — replace lines 173-179:
```go
for memberIdx := range wave.Members {
    estimated := estimateCost(wave.Members[memberIdx].Agent)
    if !tr.tryReserveBudget(estimated) {
        log.Printf("[Budget Gate] Cannot reserve $%.2f for %s", estimated, wave.Members[memberIdx].Name)
        break
    }
    wg.Add(1)
    go tr.spawnAndWait(waveIdx, memberIdx, &wg, estimated)
}
```

**Estimated cost per agent** (add to `config.go`):
```go
func estimateCost(agentID string) float64 {
    defaults := map[string]float64{
        "opus": 1.50, "sonnet": 0.30, "haiku": 0.05,
    }
    // Look up model from agents-index.json, return default for tier
    return defaults["opus"]  // Conservative fallback
}
```

**Files to modify**: `cmd/gogent-team-run/config.go`, `cmd/gogent-team-run/wave.go`
**Test cases** (add to TC-011):

| Test | Scenario | Expected |
|------|----------|----------|
| `TestBudgetReservation_Atomic` | 4 goroutines reserve $1.50 each from $5.00 budget | Exactly 3 succeed, 4th rejected |
| `TestBudgetReconciliation` | Reserved $1.50, actual cost $0.80 | Budget increases by $0.70 |
| `TestBudgetExhaustion_MidWave` | 2 members, budget for 1 | First spawns, second rejected |

---

#### Item 7: Add SDK Concurrency Investigation to TC-015 Acceptance Criteria

**Why**: TC-015 Phase 1 investigation is in "Open Questions" but NOT in the acceptance checklist. Implementers may skip it.

**Change**: Add to TC-015.md acceptance criteria section (after existing items):
```markdown
- [ ] **PHASE 1 GATE** (must complete before Phase 2):
  - [ ] SDK concurrency semantics documented with exact SDK version tested
  - [ ] Confirmed: concurrent `query()` calls work, OR
  - [ ] Documented: SDK limitation + alternative approach (separate processes / SDK feature request)
  - [ ] Test results committed to `packages/tui/docs/sdk-concurrency-investigation.md`
```

**Also**: Cross-reference TC-019 (new ticket from Section 8) as the implementation vehicle for this investigation.

**Files to modify**: `tickets/team-coordination/tickets/TC-015.md` — acceptance criteria section

---

#### Item 8: Add WaitGroup Safety Comment + Extract Retry Loop

**Why**: Backend review found that `spawnAndWait()` has a subtle footgun — the caller must call `wg.Add(1)` exactly once. If anyone refactors the wave scheduler to call `spawnAndWait()` differently, WaitGroup can go negative.

**Change 1** — Add comment to `spawn.go` `spawnAndWait()`:
```go
// spawnAndWait spawns a Claude CLI process and waits for completion with retries.
//
// CONTRACT: The caller (runWaves) must call wg.Add(1) exactly once before
// spawning this goroutine. This function calls wg.Done() exactly once via
// defer, regardless of retry count. Do NOT call this function without
// wg.Add(1) — the WaitGroup will go negative and panic.
func (tr *TeamRunner) spawnAndWait(waveIdx, memberIdx int, wg *sync.WaitGroup, estimatedCost float64) {
    defer wg.Done()
    // ...
```

**Change 2** — Extract retry loop to internal function for clarity:
```go
func (tr *TeamRunner) spawnAndWait(waveIdx, memberIdx int, wg *sync.WaitGroup, estimatedCost float64) {
    defer wg.Done()
    member := &tr.config.Waves[waveIdx].Members[memberIdx]

    success := tr.executeWithRetry(waveIdx, memberIdx, member)

    // Reconcile budget
    tr.reconcileCost(estimatedCost, member.CostUSD)

    if !success {
        log.Printf("[FAILED] %s: all retries exhausted", member.Name)
    }
}

// executeWithRetry handles the retry loop. Returns true if agent completed successfully.
func (tr *TeamRunner) executeWithRetry(waveIdx, memberIdx int, member *TeamMember) bool {
    for attempt := 0; attempt <= member.MaxRetries; attempt++ {
        // ... existing spawn logic ...
        if member.Status == "completed" {
            return true
        }
    }
    tr.updateMember(waveIdx, memberIdx, func(m *TeamMember) {
        m.Status = "failed"
    })
    return false
}
```

**Files to modify**: `cmd/gogent-team-run/spawn.go`
**Test case**: `TestSpawnAndWait_SingleDone` — verify `wg.Done()` called exactly once regardless of retry count (use a WaitGroup wrapper that counts Done calls).

---

### 10.2 HIGH — Concurrency Fixes

#### Item 13: Move Budget Deduction Inside updateMember()

**Why**: TC-008 lines 283-285 deduct budget via direct `tr.configMu.Lock()` + modify + unlock — bypassing `updateMember()`. This is a consistency hole: someone can deduct budget without updating the member's `cost_usd` field atomically.

**Current code** (TC-008 `spawn.go` lines 282-285):
```go
tr.configMu.Lock()
tr.config.BudgetRemainingUSD -= cost
tr.configMu.Unlock()
```

**Required change**: Remove direct budget manipulation. Move cost deduction into the `updateMember` call that marks completion:

```go
// In spawn.go — replace lines 282-285 + lines 296-301 with single atomic update:
tr.updateMember(waveIdx, memberIdx, func(m *TeamMember) {
    m.Status = "completed"
    m.CompletedAt = time.Now().Unix()
    m.CostUSD = cost
})
// Budget deduction happens in reconcileCost() (see Item 6)
```

**Audit requirement**: Grep TC-008 for all `tr.configMu.Lock()` calls outside `updateMember()`. The ONLY direct mutex usage should be in `tryReserveBudget()` and `reconcileCost()` (Item 6). All member-field mutations go through `updateMember()`.

**Files to modify**: `cmd/gogent-team-run/spawn.go`, `cmd/gogent-team-run/config.go`

---

#### Item 14: Make Process Registration Atomic with cmd.Start()

**Why**: TC-008 lines 258-262 register the PID AFTER `cmd.Start()`. If context cancellation fires between `Start()` and `registerChild()`, the process won't be in the kill list.

**Current code** (TC-008 `spawn.go` lines 251-262):
```go
if err := cmd.Start(); err != nil { ... }

// Gap: context cancellation here = orphan
tr.registerChild(cmd.Process.Pid)
tr.updateMember(waveIdx, memberIdx, func(m *TeamMember) {
    m.ProcessPID = cmd.Process.Pid
})
```

**Required change**: Register before start using a pre-allocated slot, or register immediately after start with no yield point:

```go
// Option A (preferred): Register immediately, no interleaving possible
if err := cmd.Start(); err != nil { ... }
pid := cmd.Process.Pid
tr.registerChild(pid)  // Immediately after Start(), same goroutine, no yield
defer tr.unregisterChild(pid)

tr.updateMember(waveIdx, memberIdx, func(m *TeamMember) {
    m.ProcessPID = pid
})
```

**Additionally**: Add a finalizer goroutine in `main.go` that runs after context cancellation:
```go
// In main.go, after runWaves() returns or context cancelled:
func (tr *TeamRunner) cleanupOrphanProcesses() {
    tr.childrenMu.Lock()
    defer tr.childrenMu.Unlock()
    for pid := range tr.childPIDs {
        if processExists(pid) {
            log.Printf("[CLEANUP] Killing orphaned process %d", pid)
            syscall.Kill(pid, syscall.SIGKILL)
        }
    }
}
```

**Files to modify**: `cmd/gogent-team-run/spawn.go`, `cmd/gogent-team-run/main.go`
**Test case** (TC-011): `TestChildCleanup_ContextCancel` — start 2 agents, cancel context mid-execution, verify both PIDs are killed within 5 seconds.

---

### 10.3 HIGH — Schema Standardization

#### Item 9: Standardize Budget Schema to Flat Fields

**Why**: Three different budget representations across TC-008 (flat), TC-009 (nested), TC-013 (different field names). Unmarshaling will fail.

**Decision** (already resolved in Blocker 3, TC-009a): **Flat fields**. `budget_max_usd` and `budget_remaining_usd` at top level.

**Files to update**:

| File | Current | Change To |
|------|---------|-----------|
| TC-009.md | `"budget": {"max_total_usd": ..., "budget_remaining_usd": ...}` | `"budget_max_usd": 5.0, "budget_remaining_usd": 5.0` |
| TC-008.md | `BudgetMaxUSD float64` (correct) | No change needed — already flat |
| TC-013.md | `budget_total_usd` | Rename to `budget_max_usd` for consistency with TC-008 |

**Validation test** (add to TC-011):
```go
func TestConfigUnmarshal_BudgetFields(t *testing.T) {
    raw := `{"budget_max_usd": 5.0, "budget_remaining_usd": 5.0, ...}`
    var config TeamConfig
    err := json.Unmarshal([]byte(raw), &config)
    require.NoError(t, err)
    assert.Equal(t, 5.0, config.BudgetMaxUSD)
    assert.Equal(t, 5.0, config.BudgetRemainingUSD)
}
```

**Files to modify**: `tickets/team-coordination/tickets/TC-009.md`, `tickets/team-coordination/tickets/TC-013.md`

---

#### Item 10: Define Stdin File Naming Convention

**Why**: TC-009 uses agent ID (`stdin_einstein.json`), TC-008 uses member name (`stdin_{name}.json`), TC-013 uses task ID (`stdin_TC-001.json`). Ambiguous when same agent runs in multiple members.

**Convention** (decided in Blocker 3, TC-009a):

| Workflow | Naming Scheme | Example | Rationale |
|----------|--------------|---------|-----------|
| Braintrust | `stdin_{member.name}.json` | `stdin_einstein.json` | member.name == agent ID (unique per team) |
| Review | `stdin_{member.name}.json` | `stdin_backend-reviewer.json` | member.name == agent ID |
| Implementation | `stdin_{member.name}.json` | `stdin_TC-001.json` | member.name == task ID (unique) |

**Rule**: `member.name` is the key. It is always unique within a team. For braintrust/review, name matches agent_id. For implementation, name is the ticket ID.

**Update TC-009.md**: Add a "File Naming Conventions" section:
```markdown
### File Naming Conventions

- Stdin: `stdin_{member.name}.json` where `member.name` is unique within the team
- Stdout: `stdout_{member.name}.json`
- For braintrust: `member.name` = agent role (e.g., "einstein", "staff-architect", "beethoven")
- For review: `member.name` = reviewer domain (e.g., "backend-reviewer", "frontend-reviewer")
- For implementation: `member.name` = ticket ID (e.g., "TC-001", "TC-002")
```

**Files to modify**: `tickets/team-coordination/tickets/TC-009.md`, `tickets/team-coordination/tickets/TC-008.md` (update `member.StdinFile` reference), `tickets/team-coordination/tickets/TC-013.md`

---

#### Item 30: Mark Stdin Schema Optional Fields Explicitly

**Why**: TC-009 stdin schemas use `"required"` array inconsistently. Fields like `reads_from.scout_metrics` may not exist if scouts didn't run. Agents receiving incomplete stdin may error.

**Change**: Update each stdin schema in TC-009 to explicitly list required vs optional:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "required": ["agent", "task", "context", "paths"],
  "properties": {
    "reads_from": {
      "description": "Optional file references. Agents should proceed with available context if fields are missing.",
      "properties": {
        "scout_metrics": { "type": "string", "description": "Path to scout_metrics.json. OPTIONAL: only present if scout was run." },
        "wave1_outputs": { "type": "array", "description": "OPTIONAL: paths to Wave 1 stdout files. Only present for Wave 2+ agents." }
      }
    }
  }
}
```

**Also add to prompt envelope boilerplate** (in TC-008 `envelope.go`):
```
If optional fields in your stdin are missing, proceed with whatever context is available.
Do NOT error or ask for missing optional fields.
```

**Files to modify**: `tickets/team-coordination/tickets/TC-009.md` (all stdin schemas)

---

### 10.4 HIGH — Process Management

#### Item 11: Implement Agent Timeout via context.WithTimeout()

**Why**: TC-009 defines `timeout_ms` per member but TC-008's `spawnAndWait()` uses `exec.CommandContext(tr.ctx)` — the parent context only. A single agent can block the entire wave indefinitely.

**Required change** in `spawn.go`:
```go
func (tr *TeamRunner) executeWithRetry(waveIdx, memberIdx int, member *TeamMember) bool {
    for attempt := 0; attempt <= member.MaxRetries; attempt++ {
        // Per-agent timeout from config
        timeout := time.Duration(member.TimeoutMS) * time.Millisecond
        if timeout == 0 {
            timeout = 10 * time.Minute  // Default
        }
        agentCtx, agentCancel := context.WithTimeout(tr.ctx, timeout)

        cmd := exec.CommandContext(agentCtx, "claude", args...)
        // ... spawn logic ...

        err := cmd.Wait()
        agentCancel()  // Always cancel to free resources

        if agentCtx.Err() == context.DeadlineExceeded {
            tr.updateMember(waveIdx, memberIdx, func(m *TeamMember) {
                m.ErrorMessage = fmt.Sprintf("timeout after %v", timeout)
            })
            continue  // Retry
        }
        // ... rest of logic ...
    }
}
```

**Type change**: Add `TimeoutMS` to `TeamMember` struct:
```go
type TeamMember struct {
    // ... existing fields ...
    TimeoutMS  int64  `json:"timeout_ms,omitempty"`  // 0 = use default (600000)
}
```

**Files to modify**: `cmd/gogent-team-run/spawn.go`, `cmd/gogent-team-run/types.go`
**Test case** (TC-011): `TestAgentTimeout_DeadlineExceeded` — spawn agent with `timeout_ms: 1000`, agent sleeps for 5s, verify killed + marked failed with "timeout" message.

---

#### Item 12: Reduce Heartbeat Interval + Define Remediation Policy

**Why**: 30s heartbeat + 60s stale threshold = up to 40s detection lag. Backend review recommends 10s. Architecture review wants remediation thresholds.

**Changes to `heartbeat.go`**:
```go
const (
    HeartbeatInterval = 10 * time.Second   // Was 30s
)
```

**Remediation policy** (add to TC-012 `/team-status` output):

| Heartbeat Age | Status Display | User Action |
|--------------|---------------|-------------|
| < 30s | `heartbeat: Xs ago` (normal) | None |
| 30-60s | `⚠️ heartbeat: Xs ago (delayed)` | Check system load |
| 60-120s | `⚠️ heartbeat: Xs ago (stale — process may be hung)` | Consider `/team-cancel` |
| > 120s | `⛔ heartbeat: Xs ago (likely dead)` | `/team-cancel` + check `runner.log` |

**Important clarification** (from backend review): "Heartbeat freshness indicates the main `gogent-team-run` process is alive, NOT that individual agents are making progress." Add this note to TC-012's heartbeat section.

**Files to modify**: `cmd/gogent-team-run/heartbeat.go` (interval), `tickets/team-coordination/tickets/TC-012.md` (remediation table + clarification)

---

### 10.5 HIGH — Design Specifications

#### Item 17: Implement Proper Project Root Detection

**Why**: TC-006 documents env var → fallback to `pwd`. But TUI doesn't set the env var. `pwd` is wrong if user is in a subdirectory.

**Detection algorithm** (add to TC-006 or TC-008 `config.go`):
```go
func resolveProjectRoot() (string, error) {
    // Priority 1: Explicit env var (set by TUI or user)
    if dir := os.Getenv("GOGENT_PROJECT_DIR"); dir != "" {
        if _, err := os.Stat(dir); err == nil {
            return dir, nil
        }
        return "", fmt.Errorf("GOGENT_PROJECT_DIR=%s does not exist", dir)
    }

    // Priority 2: Git root (works from any subdirectory)
    out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
    if err == nil {
        root := strings.TrimSpace(string(out))
        if root != "" {
            return root, nil
        }
    }

    // Priority 3: Walk up from cwd looking for indicator files
    cwd, _ := os.Getwd()
    for dir := cwd; dir != "/"; dir = filepath.Dir(dir) {
        for _, indicator := range []string{"go.mod", "package.json", "pyproject.toml", "DESCRIPTION"} {
            if _, err := os.Stat(filepath.Join(dir, indicator)); err == nil {
                return dir, nil
            }
        }
    }

    return "", fmt.Errorf("cannot detect project root: set GOGENT_PROJECT_DIR or run from a git repository")
}
```

**Validation** (after detection):
```go
// Verify detected root is plausible
root, err := resolveProjectRoot()
if err != nil { return err }
if _, err := os.Stat(filepath.Join(root, ".claude")); os.IsNotExist(err) {
    log.Printf("[WARN] No .claude/ directory in detected project root %s", root)
}
```

**TUI change** (add to TC-013): Mozart must set `project_root` in config.json by calling this function at planning time. The Go binary reads it from config.json, not from env/cwd.

**Files to modify**: `cmd/gogent-team-run/config.go` (detection function), `tickets/team-coordination/tickets/TC-006.md` (update resolution spec), `tickets/team-coordination/tickets/TC-013.md` (Mozart must call this)

---

#### Item 18: Specify Topological Sort + Cycle Detection for Wave Computation

**Why**: TC-013 `gogent-compute-waves` mentions "detect cycles" but specifies no algorithm. Missing deps or circular deps cause silent failure.

**Algorithm** (Kahn's algorithm — simple, no recursion):
```go
// In cmd/gogent-compute-waves/main.go:

type Task struct {
    ID        string
    BlockedBy []string
}

func computeWaves(tasks []Task) ([][]string, error) {
    // Build adjacency + in-degree
    inDegree := make(map[string]int)
    dependents := make(map[string][]string)  // dependency -> tasks that depend on it

    for _, t := range tasks {
        if _, ok := inDegree[t.ID]; !ok {
            inDegree[t.ID] = 0
        }
        for _, dep := range t.BlockedBy {
            dependents[dep] = append(dependents[dep], t.ID)
            inDegree[t.ID]++
        }
    }

    // Kahn's algorithm
    var waves [][]string
    for len(inDegree) > 0 {
        // Find all tasks with no remaining dependencies
        var wave []string
        for id, deg := range inDegree {
            if deg == 0 {
                wave = append(wave, id)
            }
        }

        if len(wave) == 0 {
            // Remaining tasks all have deps → cycle
            var remaining []string
            for id := range inDegree {
                remaining = append(remaining, id)
            }
            return nil, fmt.Errorf("circular dependency detected among: %v", remaining)
        }

        sort.Strings(wave)  // Deterministic ordering
        waves = append(waves, wave)

        // Remove completed tasks, decrement dependents
        for _, id := range wave {
            delete(inDegree, id)
            for _, dependent := range dependents[id] {
                if _, ok := inDegree[dependent]; ok {
                    inDegree[dependent]--
                }
            }
        }
    }

    return waves, nil
}
```

**Edge cases** (add to TC-011 tests):

| Test | Input | Expected |
|------|-------|----------|
| `TestWaves_Linear` | A→B→C | [[A], [B], [C]] |
| `TestWaves_Parallel` | A, B (no deps) | [[A, B]] |
| `TestWaves_Diamond` | A→C, B→C, C→D | [[A, B], [C], [D]] |
| `TestWaves_Cycle` | A→B, B→A | Error: "circular dependency" |
| `TestWaves_MissingDep` | A→X (X not in list) | Error or skip X (design decision needed) |
| `TestWaves_Empty` | No tasks | [] (empty) |

**Missing dep behavior** (decision needed — add to TC-013):
- Option A: Error if dep not in task list (strict) — **recommended**
- Option B: Ignore missing deps (lenient)

**Files to modify**: `tickets/team-coordination/tickets/TC-013.md` (add algorithm section), `cmd/gogent-compute-waves/main.go` (implementation)

---

#### Item 21: Add Event Flow Sequence Diagram to TC-015

**Why**: Frontend review found the freeze mechanism needs visual clarity for implementers.

**Diagram** (add to TC-015 after Root Cause Analysis section):

```
Sequence: TUI Freeze During MCP Agent Spawn

User          TUI/useClaudeQuery      SDK query()      MCP spawn_agent    Einstein CLI
  │                │                      │                  │                 │
  │─ /braintrust ─►│                      │                  │                 │
  │                │─── query(mozart) ───►│                  │                 │
  │                │  setIsStreaming(true) │                  │                 │
  │                │  streamingRef = true  │                  │                 │
  │                │                      │                  │                 │
  │                │◄─── events ──────────│                  │                 │
  │                │  (for await loop)    │                  │                 │
  │                │                      │── tool_use ─────►│                 │
  │                │                      │  spawn_agent()   │── claude -p ──►│
  │                │                      │                  │                 │
  │── "hello" ───►│                      │                  │   30+ seconds   │
  │                │  if(streamingRef)    │                  │   of thinking   │
  │                │    return; ◄─── BLOCKED                 │                 │
  │                │                      │                  │                 │
  │                │                      │                  │◄── result ──────│
  │                │                      │◄─ tool_result ───│                 │
  │                │◄─── result event ────│                  │                 │
  │                │  streamingRef = false │                  │                 │
  │                │  setIsStreaming(false)│                  │                 │
  │                │                      │                  │                 │
  │── NOW "hello" works again ──────────►│                  │                 │

SINGLE POINT OF FAILURE: Line 600 guard
  if (streamingRef.current) { return; }  ← blocks ALL user input for entire query duration
```

**Files to modify**: `tickets/team-coordination/tickets/TC-015.md` (insert after Root Cause Analysis)

---

### 10.6 HIGH — Error Handling & Build

#### Item 19: Add Session Directory Discovery Error Handling

**Why**: TC-012 slash commands depend on `GOGENT_SESSION_DIR` env var, but no error handling for: env var unset, path invalid, multiple sessions, dir deleted mid-operation.

**Error handling spec** (add to TC-012 acceptance criteria):

```go
// In each slash command handler:
func discoverTeamDirs() ([]string, error) {
    sessionDir := os.Getenv("GOGENT_SESSION_DIR")

    // Case 1: Env var set and valid
    if sessionDir != "" {
        if info, err := os.Stat(sessionDir); err != nil || !info.IsDir() {
            return nil, fmt.Errorf("GOGENT_SESSION_DIR=%s is not a valid directory", sessionDir)
        }
        return findTeamDirs(sessionDir)
    }

    // Case 2: Env var not set — scan for recent sessions
    homeDir, _ := os.UserHomeDir()
    sessionsRoot := filepath.Join(homeDir, "Documents", "GOgent-Fortress", ".claude", "sessions")
    entries, err := os.ReadDir(sessionsRoot)
    if err != nil {
        return nil, fmt.Errorf("no session directory found: set GOGENT_SESSION_DIR or check %s", sessionsRoot)
    }

    // Find most recent session with teams/
    // ... sort by mtime, find first with teams/ subdir ...
}
```

**User-facing error messages**:

| Error | Message |
|-------|---------|
| Env var unset, no sessions found | `No active session found. Start a session first or set GOGENT_SESSION_DIR.` |
| Env var points to invalid path | `Session directory not found: {path}. It may have been moved or deleted.` |
| Session dir exists but no teams/ | `No teams found in current session. Use /braintrust, /review, or /ticket to start one.` |
| Team dir deleted mid-read | `Team {name} directory is no longer accessible. It may have been cleaned up.` |

**TUI integration requirement** (add to TC-013): The TUI MUST set `process.env.GOGENT_SESSION_DIR` during session initialization, pointing to the current session's `.claude/sessions/{id}/` directory.

**Files to modify**: `tickets/team-coordination/tickets/TC-012.md` (error handling section + acceptance criteria), `tickets/team-coordination/tickets/TC-013.md` (TUI env var requirement)

---

#### Item 20: Distinguish Cost Extraction Error Levels

**Why**: TC-008 treats all cost extraction failures identically (`cost = 0, continue`). But "field not present in JSON" and "output is not valid JSON at all" are very different errors with different implications.

**Error type definition** (add to `cmd/gogent-team-run/cost.go`):
```go
type CostError struct {
    Level   string  // "warn" | "error"
    Message string
    RawOutput string  // For debugging
}

func extractCostFromCLIOutput(output []byte) (float64, *CostError) {
    // Attempt 1: Parse as JSON
    var result map[string]interface{}
    if err := json.Unmarshal(output, &result); err != nil {
        return 0, &CostError{
            Level:     "error",
            Message:   fmt.Sprintf("CLI output is not valid JSON: %v", err),
            RawOutput: string(output[:min(len(output), 500)]),
        }
    }

    // Attempt 2: Try known field names
    for _, field := range []string{"cost_usd", "total_cost_usd", "usage.cost_usd"} {
        if val, ok := getNestedFloat(result, field); ok {
            return val, nil
        }
    }

    // No cost field found
    return 0, &CostError{
        Level:     "warn",
        Message:   "no cost field found in CLI JSON output",
        RawOutput: string(output[:min(len(output), 500)]),
    }
}
```

**Logging behavior by level**:

| Error Level | Log Action | Budget Impact |
|-------------|-----------|--------------|
| `nil` (success) | Log cost normally | Deduct actual cost |
| `"warn"` (no field) | `log.Printf("[WARN] ...")` | Continue with `cost = 0`, flag member as `cost_unknown: true` |
| `"error"` (malformed) | `log.Printf("[ERROR] ...")` + write raw output to `runner.log` | Continue with `cost = 0`, flag member as `cost_error: true` |

**Add field to TeamMember struct**:
```go
type TeamMember struct {
    // ... existing fields ...
    CostStatus string `json:"cost_status,omitempty"`  // "" | "ok" | "unknown" | "error"
}
```

**Files to modify**: `cmd/gogent-team-run/cost.go`, `cmd/gogent-team-run/types.go`, `cmd/gogent-team-run/spawn.go`

---

#### Item 22: Add Makefile/Build Story to TC-008

**Why**: No ticket specifies how new Go binaries get built. The project has existing binaries in `cmd/` but no Makefile update.

**Add to TC-008 acceptance criteria**:
```markdown
- [ ] Makefile updated with new targets:
  - `make gogent-team-run` — builds cmd/gogent-team-run
  - `make gogent-team-prepare-synthesis` — builds cmd/gogent-team-prepare-synthesis
  - `make gogent-compute-waves` — builds cmd/gogent-compute-waves (if TC-013 needs it)
  - `make team-tools` — builds all team-related binaries
  - `make install-team-tools` — installs to $GOPATH/bin or ~/.local/bin
- [ ] `make build` (existing target) includes new binaries
- [ ] CI/CD updated if applicable
```

**Makefile snippet** (example):
```makefile
TEAM_BINARIES = gogent-team-run gogent-team-prepare-synthesis

.PHONY: team-tools
team-tools: $(TEAM_BINARIES)

gogent-team-run:
	go build -o bin/$@ ./cmd/$@

gogent-team-prepare-synthesis:
	go build -o bin/$@ ./cmd/$@

install-team-tools: team-tools
	install -m 755 bin/gogent-team-run ~/.local/bin/
	install -m 755 bin/gogent-team-prepare-synthesis ~/.local/bin/
```

**Files to modify**: `tickets/team-coordination/tickets/TC-008.md` (acceptance criteria), `Makefile` (if exists, or create)

---

### 10.7 MEDIUM Priority Items

#### Item 23: Verify CLI Flag Order

**Action**: Run `claude --help` and verify whether `--permission-mode delegate` must come before `--allowedTools`. Document finding in TC-001.
**Effort**: 0.5d
**Files**: TC-001.md — add verification result

---

#### Item 24: Add Agent Validation on Startup

**Action**: In TC-008 `config.go:loadConfig()`, after parsing config.json, validate that every `member.agent_id` exists in `agents-index.json`. Fail with explicit error listing unrecognized agents rather than falling back to defaults silently.

```go
func validateAgents(config *TeamConfig) error {
    index, err := loadAgentsIndex()
    if err != nil { return fmt.Errorf("load agents-index.json: %w", err) }

    for _, wave := range config.Waves {
        for _, member := range wave.Members {
            if _, ok := index[member.Agent]; !ok {
                return fmt.Errorf("unknown agent %q for member %q — check agents-index.json", member.Agent, member.Name)
            }
        }
    }
    return nil
}
```
**Files**: `cmd/gogent-team-run/config.go`

---

#### Item 25: Add ASCII Fallback Status Indicators

**Action**: Add a `--ascii` flag or auto-detect non-UTF8 terminal. Map Unicode indicators to ASCII:

| Unicode | ASCII | Meaning |
|---------|-------|---------|
| ✓ | `[OK]` | Completed |
| ⏳ | `[..]` | Running |
| ⏸ | `[--]` | Pending |
| ✗ | `[XX]` | Failed |
| 🔄 | `[>>]` | Retrying |

**Detection**: `if os.Getenv("TERM") == "dumb" || os.Getenv("GOGENT_ASCII") == "1" { useASCII = true }`
**Files**: TC-012 slash command implementation

---

#### Item 26: Add Review Conflict Resolution Logic

**Action**: When `/team-result` aggregates review findings and two reviewers report the same issue at different severities:
1. Use the **highest** severity reported
2. Show attribution: `[CRITICAL] (reported by: backend, architecture)`
3. If severities differ by 2+ levels, add note: `severity disputed — see individual reviews`

**Files**: TC-012 `/team-result` implementation for review workflow

---

#### Item 27: Add Crash Safety Integration Test

**Action**: Add to TC-011:
```go
func TestCrashSafety_SIGKILLMidWrite(t *testing.T) {
    // 1. Start gogent-team-run with a test config
    // 2. Wait for first config.json write (agent status = "running")
    // 3. Send SIGKILL to gogent-team-run
    // 4. Verify config.json is valid JSON (not corrupted by partial write)
    // 5. Verify no .tmp files left behind (or that startup cleans them)
    // 6. Restart gogent-team-run with same config — verify it resumes or fails cleanly
}
```
**Key assertion**: `config.json` must ALWAYS be valid JSON after SIGKILL — the atomic write pattern (write .tmp then rename) guarantees this because rename is atomic on Linux.
**Files**: `cmd/gogent-team-run/main_test.go`

---

#### Item 28: Document Atomic Write Recovery (.tmp files)

**Action**: Add to TC-008's `loadConfig()`:
```go
func loadConfig(teamDir string) (*TeamConfig, error) {
    configPath := filepath.Join(teamDir, "config.json")
    tmpPath := configPath + ".tmp"

    // Check for abandoned .tmp file (from previous crash)
    if _, err := os.Stat(tmpPath); err == nil {
        log.Printf("[WARN] Found abandoned %s — previous process may have crashed during write. Removing.", tmpPath)
        os.Remove(tmpPath)
    }

    // ... normal load logic ...
}
```
**Document in TC-008**: ".tmp files are safe to remove — they represent incomplete writes. The rename-from-.tmp pattern guarantees config.json is always complete."
**Files**: `cmd/gogent-team-run/config.go`, TC-008.md

---

#### Item 29: Standardize Fallback Defaults Across Spawn Paths

**Action**: TC-014 documents that missing `cli_flags` gets different defaults in TUI vs Go binary. Standardize:
- If `allowed_tools` is missing in agents-index.json → **fail with error**, not silent default
- This aligns with Item 24 (agent validation on startup)

**Alternatively** (if backward-compat needed): Use the SAME default across all paths: `["Read", "Glob", "Grep"]` (read-only). Write/Edit/Bash require explicit opt-in.

**Files**: TC-014.md, `packages/tui/src/mcp/spawnAgent.ts`, `cmd/gogent-team-run/spawn.go`

---

#### Item 31: Document PID Check Limitation + Add Timestamp

**Action**: Add to TC-016.md:
```markdown
### Known Limitations

1. **Same-machine only**: PID liveness check (`kill -0 $PID`) only works on the
   same machine. If the PID file is on a shared filesystem (NFS), a different
   machine's PID may collide. This is acceptable for GOgent-Fortress which is
   single-machine.

2. **PID reuse**: On long-running systems, the OS may reuse a PID. Mitigate by
   adding a timestamp to the PID file:
```

**PID file format change**:
```
12345
1738856422
```
Line 1: PID. Line 2: Unix timestamp of write. On liveness check: if timestamp > 24 hours old AND process exists, treat as coincidental PID reuse and reclaim.

**Files**: TC-016.md, `cmd/gogent-team-run/daemon.go` (PID file format)

---

### 10.8 LOW Priority Items

#### Item 15: Add TC-003 to TC-011's blocked_by

**Action**: Edit TC-011.md header:
```
Blocked By: TC-002 (mutex), TC-003 (retry fix)
```
TC-011 tests retry behavior that TC-003 defines. Without TC-003, the retry tests would be testing the wrong (recursive) pattern.
**Files**: `tickets/team-coordination/tickets/TC-011.md`

---

#### Item 16: Create Orchestrator Rewrite Design Docs

**Action**: Covered by TC-020 specification in Section 8. No additional specification needed here. TC-020 creates Mozart-rewrite.md, ReviewOrch-rewrite.md, ImplMgr-rewrite.md.

---

#### Item 32: Establish Error Message Convention

**Action**: Define a standard error format for all team-related Go binaries:

```
[{LEVEL}] {component}: {message}
  → {actionable next step}
```

Examples:
```
[ERROR] spawn: failed to start einstein (attempt 1/2): exec: "claude": executable file not found in $PATH
  → Verify claude CLI is installed and in PATH

[WARN] cost: no cost field in CLI output for einstein
  → Budget tracking may be inaccurate. Check runner.log for raw output.

[INFO] wave: Wave 1 completed (2/2 agents, $2.86 total)
```

**Files**: Add as a convention section to TC-008.md. All team binaries follow this pattern.

---

#### Item 33: Add Pre-synthesis.md Validation Test

**Action**: Add to TC-011:
```go
func TestPreSynthesis_MatchesBeethovenExpectation(t *testing.T) {
    // Generate pre-synthesis.md from test Wave 1 outputs
    preSynthesis := generatePreSynthesis(testTeamDir)

    // Verify expected sections exist (what Beethoven's stdin expects)
    assert.Contains(t, preSynthesis, "## Einstein Analysis")
    assert.Contains(t, preSynthesis, "## Staff-Architect Review")
    assert.Contains(t, preSynthesis, "### Root Cause")
    assert.Contains(t, preSynthesis, "### Architecture Smells")
    // ... etc
}
```
**Files**: `cmd/gogent-team-prepare-synthesis/main_test.go`

---

#### Item 34: Add StatusLine Background Team Status Integration

**Action**: After TC-012 and TC-015 are implemented, StatusLine.tsx should read background team status from config.json (polled, not push). Add to TC-012 or TC-015 acceptance criteria:
```markdown
- [ ] StatusLine shows background team count: "2 teams running" when teams are active
- [ ] StatusLine refreshes team count on /team-status invocation or on 30s interval
```

**Implementation hint**: StatusLine.tsx currently computes `agentCounts` from Zustand store only (lines 159-171). Add a `backgroundTeamCount` field to the UI store, populated by polling session directory for `config.json` files with `background_pid != null`.

**Files**: TC-012.md or TC-015.md (acceptance criteria), `packages/tui/src/components/StatusLine.tsx`, `packages/tui/src/store/slices/ui.ts`

---

## 11. Summary of Ticket File Updates Required

| Ticket | Sections to Add/Modify |
|--------|----------------------|
| **TC-008** | Budget reservation pattern (Item 6), process registration (Item 14), budget audit (Item 13), agent validation (Item 24), .tmp cleanup (Item 28), Makefile targets (Item 22), error message convention (Item 32) |
| **TC-009** | Budget fields → flat (Item 9), file naming conventions (Item 10), optional field markers (Item 30) |
| **TC-011** | Add TC-003 dependency (Item 15), budget reservation tests (Item 6), process cleanup test (Item 14), crash safety test (Item 27), wave computation tests (Item 18), pre-synthesis validation (Item 33) |
| **TC-012** | Heartbeat remediation table (Item 12), session discovery error handling (Item 19), ASCII fallback (Item 25), review conflict resolution (Item 26), StatusLine integration (Item 34) |
| **TC-013** | Project root detection (Item 17), TUI env var requirement (Item 19), wave computation algorithm (Item 18) |
| **TC-015** | Phase 1 gate in acceptance criteria (Item 7), sequence diagram (Item 21) |
| **TC-016** | PID timestamp + same-machine limitation (Item 31) |
| **TC-014** | Fallback defaults standardization (Item 29) |
