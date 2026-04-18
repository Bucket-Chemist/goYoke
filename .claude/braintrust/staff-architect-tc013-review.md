# Critical Review: TC-013a/b/c Team-Run Migration Tickets

**Reviewed:** 2026-02-08T16:45:00Z
**Reviewer:** Staff Architect Critical Review
**Input:** `tickets/team-coordination/tickets/TC-013a.md`, `TC-013b.md`, `TC-013c.md`
**Supporting:** `ReviewOrch-team-bridge.md`, `ImplMgr-team-bridge.md`, `TC-020.md`
**Verified Against:** Go source (`cmd/goyoke-team-run/*.go`), JSON schemas (`schemas/stdin/*.json`), team templates (`schemas/teams/*.json`), contract files (`schemas/teams/stdin-stdout/*.json`), review skill (`skills/review/SKILL.md`), team-result skill (`skills/team-result/SKILL.md`)

---

## Executive Assessment

**Overall Verdict:** CONCERNS

**Confidence Level:** HIGH

- Rationale: Every claim was verified against Go source code, JSON schemas, and contract files. The codebase is well-structured and the verification was exhaustive.

**Issue Counts:**
- Critical: 3 (must fix)
- Major: 7 (should fix)
- Minor: 6 (consider fixing)

**Commendations:** 5

**Summary:** The tickets are well-decomposed, properly sequenced, and show good awareness of existing implementation details. However, three critical issues will cause immediate implementation failures: the braintrust budget arithmetic makes Wave 2 unreachable, the wave failure propagation behavior contradicts test expectations without scoping the required code change, and the schema/contract duality is unresolved with a concrete validation-breaking conflict. Seven major issues would cause significant rework if not addressed before implementation begins.

**Go/No-Go Recommendation:**
Do NOT start implementation until C-1 (budget decision), C-2 (wave failure propagation scope), and C-3 (schema authority resolution) are resolved. These are 30-minute decisions, not days of work. Once decided, proceed with TC-013a confidently.

---

## Issue Register

### Critical Issues (Must Fix Before Proceeding)

| ID | Layer | Location | Issue | Impact | Recommendation |
|----|-------|----------|-------|--------|----------------|
| C-1 | Failure Modes | TC-013b Section 5 | Braintrust budget arithmetic blocks Wave 2 | Beethoven never runs; braintrust produces no synthesis | Raise budget to $16.00 or lower opus estimate |
| C-2 | Assumptions | TC-013c Test 4 | Wave failure propagation requires unscoped code change | Test expects behavior that `wave.go` does not implement | Add explicit deliverable for `runWaves` modification |
| C-3 | Dependencies | All tickets | Schema vs contract duality unresolved | Implementer does not know which to follow; architect contract breaks schema validation | Declare authority and fix architect contract |

---

**Detail for C-1: Braintrust budget arithmetic blocks Wave 2**

TC-013b Section 5 (lines 109-115) states:

> Default braintrust budget is $5.00 (template). With `estimateCost` returning $5.00 per Opus agent, the budget is insufficient for 2 Opus agents in Wave 1.
>
> **Options:**
> - Raise default budget to $12.00
> - Lower Opus estimate in `estimateCost` to $2.50
> - Both

This is presented as "options" without a decision. The arithmetic is worse than stated.

**Actual budget flow (verified from `wave.go` lines 35-40 and `config.go` lines 373-379):**

1. Wave 1 starts. Budget = $5.00.
2. Einstein spawns. `tryReserveBudget($5.00)` succeeds. Remaining = $0.00.
3. Staff-Architect attempts spawn. `tryReserveBudget($5.00)` fails. Budget gate blocks.
4. Wave 1 completes with only Einstein. Staff-Architect status = `pending` (never started).
5. Inter-wave script runs `goyoke-team-prepare-synthesis`. It reads `stdout_einstein.json` (exists) and `stdout_staff-architect.json` (DOES NOT EXIST). The prepare-synthesis binary has graceful degradation but produces a one-sided pre-synthesis.
6. Wave 2: Beethoven attempts spawn. Budget = (reconciled after Einstein, probably ~$0-2 remaining). `tryReserveBudget($5.00)` almost certainly fails.
7. Result: Beethoven never runs. No synthesis produced. Braintrust is broken.

Even raising to $12.00 is marginal: $5.00 + $5.00 reserved for Wave 1 = $2.00 remaining. After reconciliation (actual < estimated), maybe $6-8 remaining. Beethoven needs $5.00 reservation. Tight.

**Recommendation:** Raise `braintrust.json` budget to $16.00 (covers 3 opus agents at $5.00 estimate with $1.00 margin). This is not user-facing cost -- it's a reservation ceiling. Actual spend is reconciled to real cost. Document this decision in TC-013b as a concrete action, not an open question.

---

**Detail for C-2: Wave failure propagation requires unscoped code change**

TC-013c Test Case 4 (line 198-201) states:

> **Test 4: Task failure mid-wave**
> - Wave 1 has 2 tasks, one fails after retries
> - Wave 2 tasks have status implications

The ImplMgr-team-bridge.md (line 253) explicitly documents:

> **Failure propagation:** If a member in wave N fails (after retries), the current implementation continues to wave N+1. TC-013 test case 8 expects that wave N+1 does NOT execute when a dependency fails. This behavior is NOT currently implemented in `wave.go` -- `runWaves` iterates all waves unconditionally. If dependency-failure-stops-downstream is required, `runWaves` needs modification.

Verified in `wave.go` lines 14-59: `runWaves` iterates `for waveIdx, wave := range tr.config.Waves` unconditionally. There is no check for failed members in previous waves. The only stops are budget exhaustion (line 19) and context cancellation (line 27-31).

TC-013c presents this test case without scoping the `runWaves` code change as a deliverable. The Files to Create/Modify table (lines 155-165) lists only the new `goyoke-plan-impl` binary and the SKILL.md. It does NOT list `cmd/goyoke-team-run/wave.go`.

**Recommendation:** Add `cmd/goyoke-team-run/wave.go` to TC-013c's Files to Create/Modify table. Scope the change explicitly: after each wave completes, check if any member has `status == "failed"`. If yes, skip subsequent waves and set their members to `status: "skipped"`. Add a new test to `wave_test.go`. Estimate: 0.5 day additional effort.

---

**Detail for C-3: Schema vs contract duality unresolved**

Two parallel sets of stdin specifications exist:

1. **Formal JSON Schemas** in `schemas/stdin/*.json` (e.g., `reviewer.json`) -- used for CI validation via `ajv validate`
2. **Contract/template instances** in `schemas/teams/stdin-stdout/*.json` (e.g., `review-architect.json`) -- used as implementation examples

These are NOT consistent. Concrete example:

`schemas/stdin/reviewer.json` line 38 requires:
```json
"required": ["files", "total_files", "languages_detected"]
```

`schemas/teams/stdin-stdout/review-architect.json` lines 11-28 uses:
```json
"review_scope": {
  "changed_files": [...],
  "direct_imports": [...],
  "total_files": 0,
  "languages_detected": ["go"]
}
```

The `files` array is MISSING from the architect contract. It uses `changed_files` instead. This passes `buildPromptEnvelope` validation (which only checks `agent`, `context`, and `task`/`description`) but FAILS formal schema validation.

Furthermore, `focus_areas` in the contract files has rich nested structure (`{enabled, priorities}` per domain) while the schema defines `focus_areas` as a generic `type: object` with no internal structure. This passes validation but means the schema provides no value for `focus_areas` validation.

None of the three tickets address this. TC-013a line 42 says "must comply with `schemas/stdin/reviewer.json`" but the architect contract cannot comply.

**Recommendation:** Make one decision and document it:

**Option A (Schema is authoritative):** Fix `review-architect.json` contract to include `files[]` as required. Keep `changed_files` and `direct_imports` as additional optional fields (which the schema already permits at lines 66-79).

**Option B (Contracts are authoritative):** Update `reviewer.json` schema to make `files` optional when `changed_files` is present (use JSON Schema `oneOf` or `anyOf`).

Option A is simpler and recommended. The contract file just needs a `files` array added alongside `changed_files`.

---

### Major Issues (Should Fix, Can Proceed with Caution)

| ID | Layer | Location | Issue | Impact | Recommendation |
|----|-------|----------|-------|--------|----------------|
| M-1 | Contractor Readiness | TC-013a Section 1 | No code for config generation | Implementer must design LLM-to-JSON generation from scratch | Add pseudocode or reference the bridge doc's example more explicitly |
| M-2 | Assumptions | TC-013a Section 5 | Feature flag not defined anywhere | `use_team_pattern` referenced but settings.json does not exist | Define the schema and create settings.json |
| M-3 | Dependencies | TC-013a Section 6 | `/team-result` stdout expectations undocumented | Review SKILL.md and team-result SKILL.md describe different output schemas | Align stdout shapes between contracts and team-result expectations |
| M-4 | Testing | TC-013a-c | No unit tests specified for config generators | Config generation is the core deliverable but only e2e tests described | Add unit test requirements |
| M-5 | Architecture Smells | TC-013a | Review SKILL.md has no team-run path | Current skill describes only foreground orchestrator dispatch | Scope the SKILL.md rewrite |
| M-6 | Contractor Readiness | TC-013b Section 1 | Mozart stdin template location unspecified | "Phase 2.5 (lines 370-429)" but line numbers will change after edit | Use section headers, not line numbers |
| M-7 | Failure Modes | TC-013b Section 3 | Inter-wave script path resolution unverified | `on_complete_script: "goyoke-team-prepare-synthesis"` -- binary must be on PATH | Add PATH verification step |

---

**Detail for M-1: No code for config generation**

TC-013a says "router generates config.json + stdin files directly" (line 21) and describes WHAT to generate (Section 2-3) but not HOW. The router is an LLM session. It would need to:

1. Run `git diff --staged --name-only` via Bash
2. Classify files by extension (trivial)
3. Generate JSON config and stdin files (non-trivial)
4. Write files to disk (via Write tool)
5. Launch binary (via Bash)

The bridge document provides a complete stdin example (lines 66-111) and launch sequence (lines 133-162). But the ticket itself does not reference these sections or say "follow the bridge doc steps."

**Recommendation:** Add to TC-013a Section 1: "Follow the config generation procedure in `ReviewOrch-team-bridge.md` Sections 3-4 exactly. The bridge doc's stdin example is the authoritative template." Additionally, consider whether this should be a Go binary (like `goyoke-plan-impl` for TC-013c) rather than LLM-generated JSON. LLM JSON generation is fragile; a small Go helper that takes git diff output and emits config.json + stdin files would be more reliable.

---

**Detail for M-2: Feature flag not defined**

All three tickets reference `settings.json -> "use_team_pattern": true/false`. A grep for `use_team_pattern` shows it only appears in the tickets themselves and the decision matrix. There is no `settings.json` file in the `.claude/` directory (only `routing-schema.json` exists). There is no schema for settings, no code that reads this flag, and no default value.

**Recommendation:** Define the settings schema and create the file. At minimum:
```json
{
  "use_team_pattern": false,
  "braintrust_budget_usd": 16.0
}
```
Add to TC-013a as a prerequisite deliverable: "Create `.claude/settings.json` with `use_team_pattern: false` default. Router checks this flag before choosing dispatch path."

---

**Detail for M-3: `/team-result` stdout expectations mismatch**

The `/team-result` SKILL.md (lines 257-340) describes the REVIEW WORKFLOW output format. It expects each reviewer stdout to have:
```json
{
  "reviewer": "string",
  "status": "string",
  "overall_assessment": "string",
  "findings": [{"severity": "CRITICAL|HIGH|MEDIUM|LOW|INFO", ...}]
}
```

But the contract files (`review-backend.json` stdout section) use:
```json
{
  "reviewer": "backend-reviewer",
  "status": "complete|partial|failed",
  "overall_assessment": "APPROVE|WARNING|BLOCK",
  "findings": [{"severity": "critical|warning|info", ...}]
}
```

Issues:
- Severity enum mismatch: SKILL expects `CRITICAL|HIGH|MEDIUM|LOW|INFO` (5 levels); contracts use `critical|warning|info` (3 levels, lowercase)
- The SKILL deduplicates by `file + line + category`; the contracts include `sharp_edge_id` which would be better for deduplication
- Case mismatch: SKILL expects uppercase severity, contracts use lowercase

**Recommendation:** Align severity enums. Either update `/team-result` SKILL.md to handle both 3-level and 5-level severities (with case normalization), or update contracts to use the 5-level scheme. The 3-level scheme from the contracts is simpler and matches the review SKILL's approval criteria better. Pick one and document it.

---

**Detail for M-4: No unit tests for config generators**

TC-013a has 4 test cases, TC-013b has 4, TC-013c has 5. All are end-to-end integration tests ("stage changes, run review, check output"). None specify unit tests for:
- JSON schema compliance of generated stdin files
- Config.json field population correctness
- Reviewer selection logic (file extension mapping)
- Dynamic member exclusion (TC-013a: only selected reviewers in config)

TC-013c does specify unit tests for `goyoke-plan-impl` (line 163: `main_test.go`), but TC-013a and TC-013b have no unit test files listed.

**Recommendation:** For TC-013a, add a unit test that generates a reviewer stdin file and validates it against `schemas/stdin/reviewer.json` using `ajv validate` or equivalent. For TC-013b, add a unit test that generates Einstein and Staff-Architect stdin files and validates against respective schemas.

---

**Detail for M-5: Review SKILL.md has no team-run path**

The current `skills/review/SKILL.md` (read above) describes only the foreground path: Phase 3 dispatches to `review-orchestrator` via `Task(sonnet)`. There is no mention of `goyoke-team-run`, team directories, config.json generation, or the background dispatch path.

TC-013a's Files to Create/Modify (line 97) says "Modify -- Add team-run dispatch path alongside existing foreground path." This is correct but undersells the scope. The SKILL.md needs:
- A new Phase 3 alternative (team-run path)
- Feature flag check logic
- Team directory creation
- Config + stdin generation
- Launch command
- Verification step
- Updated Phase 4 (use `/team-result` instead of inline report)

This is essentially a rewrite of the core workflow section.

**Recommendation:** Acknowledge this in the effort estimate. The SKILL.md modification is a significant deliverable, not a minor edit. Consider scoping it as a separate sub-task within TC-013a.

---

**Detail for M-6: Mozart stdin template location unspecified**

TC-013b Section 1 references "Phase 2.5 (lines 370-429)" in `mozart.md`. Line numbers are fragile -- they change with any edit. The ticket should reference section headers instead.

**Recommendation:** Change "lines 370-429" to "Phase 2.5: Config Generation Templates" (or whatever the section header is). Verify the section exists and is the correct one.

---

**Detail for M-7: Inter-wave script path resolution**

`braintrust.json` template sets `on_complete_script: "goyoke-team-prepare-synthesis"` (a bare binary name, not an absolute path). `runInterWaveScript` in `wave.go` line 101 executes:
```go
cmd := exec.CommandContext(ctx, scriptPath, teamDir)
cmd.Dir = teamDir
```

This relies on the binary being on `$PATH`. If the user has not run `go install ./cmd/goyoke-team-prepare-synthesis/`, the script will fail with "executable file not found in $PATH."

TC-013b Section 3 (line 100) says "Verify: `goyoke-team-prepare-synthesis` binary is on PATH" but this is a note, not a deliverable or test case.

**Recommendation:** Add to TC-013b Test Case 3 (inter-wave script failure): verify that the error message in config.json is actionable ("goyoke-team-prepare-synthesis not found in PATH. Run: go install ./cmd/goyoke-team-prepare-synthesis/"). Also consider using absolute path in the template (`${GOPATH}/bin/goyoke-team-prepare-synthesis`) instead of relying on PATH.

---

### Minor Issues (Consider Addressing)

| ID | Layer | Location | Issue | Impact | Recommendation |
|----|-------|----------|-------|--------|----------------|
| m-1 | Assumptions | TC-013a line 52 | Session ID source unclear | `$GOYOKE_SESSION_ID` or generate -- inconsistent with TUI | Align with TUI's `GOYOKE_SESSION_DIR` env var |
| m-2 | Contractor Readiness | TC-013c | specs.md format not formally specified | Parser tolerance unclear | Add 2-3 example specs.md files as test fixtures |
| m-3 | Cost-Benefit | TC-013c | Kahn's algorithm in Go is ~50 LoC | Ticket spends significant space describing it | Reference a known implementation or provide skeleton |
| m-4 | Architecture Smells | TC-013a-c | Feature flag checked by LLM, not programmatically | LLM may forget or misread settings.json | Consider hook-based enforcement |
| m-5 | Testing | TC-013b | Test 2 (Einstein-only) requires Mozart to detect single-agent request | No specification of how Mozart determines "just Einstein" | Document the detection logic |
| m-6 | Contractor Readiness | TC-013c line 215 | Agent validation recommended but not required | Unclear if `goyoke-plan-impl` should fail on unknown agent | Make it a MUST with clear error message |

---

**Detail for m-1: Session directory provenance**

TC-013a line 52 says: `session_id: from $GOYOKE_SESSION_ID or generate`. The TUI actually sets `GOYOKE_SESSION_DIR` (verified in `packages/tui/src/App.tsx` line 18):
```typescript
process.env["GOYOKE_SESSION_DIR"] = join(home, ".claude", "sessions", sessionId);
```

There is no `GOYOKE_SESSION_ID` env var -- the TUI sets `GOYOKE_SESSION_DIR` (the full path, not just the ID). The bridge doc (line 135) uses a different pattern: `session_dir=".claude/sessions/${GOYOKE_SESSION_ID:-...}"`.

**Recommendation:** Use `GOYOKE_SESSION_DIR` (which the TUI actually sets) and extract the session ID from the path if needed. Document this in all three tickets.

---

**Detail for m-4: Feature flag checked by LLM**

The `use_team_pattern` flag is read by the LLM from `settings.json`. This is "documentation theater" per the project's own enforcement architecture principles in `router-guidelines.md` Section 6. The LLM might:
- Forget to check the flag
- Misread the JSON
- Check a cached/stale value

**Recommendation:** For the initial implementation, LLM-based flag checking is acceptable (pragmatic). But log a follow-up ticket to move this to a hook (e.g., `goyoke-validate` could inject the flag state into tool context, or the `/review` skill could check it programmatically via Bash before deciding the dispatch path).

---

## Assumption Register

| # | Assumption | Source | Verified? | Risk if False | Mitigation |
|---|-----------|--------|-----------|---------------|------------|
| A-1 | `goyoke-team-run` daemon writes `background_pid` before parent reads it | TC-013a Section 4, bridge doc line 156 | Verified (main.go lines 76-87 writes PID synchronously before `runWaves`) | Race condition: parent reads null PID | `sleep 2` + retry in launch script (bridge doc already has this) |
| A-2 | `buildPromptEnvelope` passes through all stdin JSON to agents | TC-013a Section 3 | Verified (envelope.go lines 96-108: uses `stdin.raw` for full JSON preservation) | Agents receive partial data | None needed -- verified |
| A-3 | Review workflow needs no inter-wave script | TC-013a | Verified (single wave, `on_complete_script: null` in template) | N/A | None needed |
| A-4 | `estimateCost` returns $5.00 for opus agents | TC-013b Section 5 | Verified (config.go line 379: `return 5.00`) | Budget calculations wrong | None needed -- verified |
| A-5 | `goyoke-team-prepare-synthesis` handles missing stdout files gracefully | TC-013b Section 3 | Partially verified (TC-010 states "graceful degradation" but not verified in source) | Crashed binary blocks Wave 2 | Add to TC-013b Test Case 3 |
| A-6 | Agents spawned at nesting level 2 can use Task(haiku/sonnet) | Envelope.go line 117 | Verified (envelope says so; `goyoke-validate` blocks opus only) | Agents are inert (cannot delegate) | None needed |
| A-7 | `settings.json` exists and is readable by the LLM | All tickets | NOT verified -- file does not exist | Feature flag check fails, unknown behavior | Create the file (see M-2) |
| A-8 | Wave members within a wave are independent (no intra-wave deps) | TC-013c Kahn's algorithm | Verified by design (Kahn's ensures wave N contains only tasks whose deps are in waves 0..N-1) | Parallel execution of dependent tasks | None needed |
| A-9 | The `claude` CLI binary is available on PATH inside spawned processes | All tickets | Assumed (spawn.go line 127: `exec.Command("claude", ...)`) | All agent spawns fail | Add to prerequisites |
| A-10 | Reviewer agents produce JSON stdout matching contract schema | TC-013a Section 6 | NOT verified -- agents are LLMs and may produce arbitrary output | `/team-result` parsing fails | Use `validateStdout` (already exists in spawn.go) and handle malformed output gracefully |

---

## Dependency Map

```
TC-013a (Review)
  Depends on:
    [x] TC-020 (bridge docs -- completed)
    [x] TC-008 (goyoke-team-run binary -- completed)
    [x] TC-009 (schemas -- completed)
    [x] TC-012 (/team-status, /team-result, /team-cancel -- completed)
    [ ] settings.json creation (NOT scoped anywhere -- see M-2)
    [ ] Schema/contract alignment (NOT scoped -- see C-3)
  Blocks: TC-013b

TC-013b (Braintrust)
  Depends on:
    [x] TC-013a (validates pattern)
    [x] TC-010 (goyoke-team-prepare-synthesis -- completed)
    [ ] Budget decision (NOT made -- see C-1)
    [ ] Mozart stdin template reconciliation (scoped in ticket)
  Blocks: TC-013c

TC-013c (Implementation)
  Depends on:
    [ ] TC-013b (validates multi-wave)
    [ ] wave.go failure propagation change (NOT scoped -- see C-2)
    [ ] specs.md format stabilization (partially scoped -- see m-2)
  Blocks: None
```

**Hidden dependency:** All three tickets depend on the `claude` CLI being installed and on PATH in the execution environment. This is never stated.

**Circular dependency check:** None found. The sequencing TC-013a -> TC-013b -> TC-013c is clean.

---

## Commendations

1. **Excellent decomposition.** The original TC-013 was a monolithic 3-workflow ticket. Splitting into TC-013a/b/c with correct sequencing (review validates pattern, braintrust validates multi-wave, implementation builds on both) is textbook work breakdown.

2. **Bridge documents are high quality.** The ReviewOrch and ImplMgr bridge docs correctly identify every field name mismatch, schema path correction, and launch command error in the original TC-013. The corrections checklist format with line numbers is contractor-ready.

3. **Strong source-of-truth hierarchy.** TC-020's explicit hierarchy (Go source > schemas > bridge docs > TC-013 inline) prevents ambiguity about which document wins on conflict.

4. **Feature flag for gradual rollout.** All three tickets include `use_team_pattern` flag for fallback to existing foreground paths. This is correct risk mitigation for a significant architectural change.

5. **Budget handling is well-designed.** The reserve-then-reconcile pattern in `wave.go` with floor-at-zero enforcement is sound. The budget gate prevents runaway spending. The only issue is the specific numbers (C-1), not the mechanism.

---

## Recommendations

### High Priority (Must Address Before Implementation)

1. **C-1: Raise braintrust budget to $16.00** in `schemas/teams/braintrust.json`. This is a one-line change. Without it, braintrust produces no synthesis. Decision required, not implementation work.

2. **C-2: Scope `wave.go` modification in TC-013c.** Add `cmd/goyoke-team-run/wave.go` to Files to Create/Modify. Write the acceptance criteria: "After wave N completes, if any member has `status: failed`, subsequent waves are skipped and their members set to `status: skipped`." Add to test cases.

3. **C-3: Resolve schema/contract authority.** Recommended: add `files[]` array to `review-architect.json` contract (keep `changed_files` as supplementary). Update any other contracts that omit required schema fields.

### Medium Priority (Address During Implementation)

4. **M-2: Create `settings.json`.** Define minimal schema, create file with `use_team_pattern: false` default.

5. **M-3: Align severity enums** between `/team-result` SKILL.md and reviewer contracts. Pick 3-level or 5-level, normalize case.

6. **M-1: Add pseudocode or Go helper** for TC-013a config generation. Consider whether a lightweight Go binary (like `goyoke-plan-review`) is more reliable than LLM-generated JSON.

7. **M-4: Add schema validation unit tests** for generated stdin files in TC-013a and TC-013b.

### Low Priority (Post-Implementation Improvements)

8. **m-1: Use `GOYOKE_SESSION_DIR`** (set by TUI) instead of `GOYOKE_SESSION_ID` across all tickets.

9. **m-4: Log follow-up ticket** to move `use_team_pattern` checking from LLM responsibility to hook enforcement.

10. **m-2: Add test fixture specs.md files** for TC-013c parser testing.

---

## Risk Heat Map

| Phase | Risk | Likelihood | Impact | Mitigated? |
|-------|------|------------|--------|------------|
| TC-013a | LLM generates invalid JSON | Medium | High (launch fails) | Partially (envelope validates minimally) |
| TC-013a | Reviewer agents produce unexpected stdout | Medium | Medium (team-result parsing fails) | Partially (malformed JSON handling in SKILL.md) |
| TC-013b | Budget blocks Beethoven | **Certain** | **Critical** (no synthesis) | **No** -- see C-1 |
| TC-013b | `goyoke-team-prepare-synthesis` not on PATH | Low | High (Wave 2 never starts) | Partially (TC-013b notes it) |
| TC-013c | specs.md format variations break parser | Medium | High (no config generated) | Partially (parser should be tolerant) |
| TC-013c | Wave 2 runs despite Wave 1 failure | **Certain** | Medium (wasted LLM spend) | **No** -- see C-2 |
| All | `settings.json` does not exist | **Certain** | Low (LLM uses default) | **No** -- see M-2 |

---

## Final Sign-Off

**Reviewed By:** Staff Architect Critical Review
**Review Date:** 2026-02-08
**Files Examined:** 25+ (tickets, schemas, contracts, Go source, skill definitions)

**Conditions for Approval:**

- [ ] C-1: Budget decision made and braintrust.json updated
- [ ] C-2: wave.go modification scoped as TC-013c deliverable
- [ ] C-3: Schema/contract authority declared and architect contract fixed

**Recommended Implementation Order:**

1. Resolve C-1, C-2, C-3 decisions (30 minutes)
2. Create `settings.json` (M-2, 15 minutes)
3. Fix `review-architect.json` contract (C-3, 15 minutes)
4. Proceed with TC-013a implementation
5. After TC-013a validates: proceed with TC-013b (with updated budget)
6. After TC-013b validates: proceed with TC-013c (with wave.go modification)

**Post-Approval Monitoring:**

- Watch TC-013a for LLM JSON generation reliability. If >30% of generated configs fail schema validation, escalate to building a Go helper binary.
- Watch TC-013b for actual opus costs vs $5.00 estimate. If actual costs are consistently <$2.00, lower the estimate to free budget headroom.
- Watch TC-013c specs.md parser for edge cases. The format is not formally specified; real-world specs.md files may deviate.
- After all three tickets complete: run `/team-result` against each workflow type to verify stdout parsing works end-to-end.
