# Critical Review: /review-bioinformatics Skill + 6 Opus Bioinformatics Reviewer Agents

**Reviewed:** 2026-04-09
**Reviewer:** Staff Architect Critical Review
**Input:** tickets/review-bioinformatics/specs.md, tickets/review-bioinformatics/implementation-plan.json

---

## Executive Assessment

**Overall Verdict:** APPROVE_WITH_CONDITIONS

**Confidence Level:** HIGH

- Rationale: Plan is well-structured, mirrors a proven architecture (/review), has thorough integration point tracking (59 points), and includes a validation gate. Domain is familiar (GOgent agent wiring). The conditions are concrete and fixable without replanning.

**Issue Counts:**

- Critical: 1 (must fix)
- Major: 4 (should fix)
- Minor: 3 (consider fixing)

**Commendations:** 5

**Summary:** This is a well-architected plan that correctly mirrors the /review pattern for a bioinformatics domain. The core design decisions (no orchestrator, Opus tier, max 4 reviewers, bioinformatician always included) are sound. However, there is one critical contradiction in spawn path permissions, and several major issues around phantom fields, inherited documentation inconsistencies, and agent assignment that should be addressed before implementation begins.

**Go/No-Go Recommendation:**
Fix C-1 before proceeding. M-1 through M-4 can be addressed during implementation if acknowledged now. If contractor hours were on the line Monday, I would sign off after C-1 is resolved and M-1/M-2 are acknowledged.

---

## Issue Register

### Critical Issues (Must Fix Before Proceeding)

| ID | Layer | Location | Issue | Impact | Recommendation |
|----|-------|----------|-------|--------|----------------|
| C-1 | Assumptions + Dependencies | specs.md Decision Table + task-006 | Contradictory spawn path: allowlist vs leaf-only | Unnecessary privilege escalation or broken direct-spawn path | Remove from allowlist OR add spawned_by |

**Detail for C-1:**

The plan contains a direct contradiction between two design decisions:

1. **Decision: "No spawned_by/can_spawn"** (specs.md line 307-313): "These reviewers are leaf agents spawned by gogent-team-run (CLI subprocess), not by spawn_agent. Therefore: No spawned_by field, No can_spawn field."

2. **Task-006 description**: "tiers.opus.task_invocation_allowlist: Add all 6 agent IDs. This is needed because opus tier has task_invocation_blocked: true, and these agents **may be** spawned via Task() or spawn_agent in addition to team-run."

These are mutually exclusive. The `task_invocation_allowlist` permits `Task(model: "opus")` calls to bypass `gogent-validate` blocking. But if these are leaf-only agents with no `spawned_by` field, then:
- `spawn_agent` would fail bidirectional validation (no `spawned_by` includes "router")
- `Task(model: "opus")` would succeed BUT bypass all context injection (`buildFullAgentContext()` not called)

The current allowlist contains only architecture/analysis agents (planner, architect, staff-architect, etc.) — agents that are legitimately invoked via Task/spawn_agent. The existing Sonnet-tier code reviewers (backend-reviewer, etc.) are NOT on any allowlist because they're team-run-only.

**Recommendation:** Remove all 6 bioinformatics reviewers from `task_invocation_allowlist` in task-006. They follow the same pattern as existing code reviewers (team-run only). If direct invocation is needed later (e.g., future orchestrator), add both allowlist AND spawned_by at that time. This aligns with the existing /review precedent.

**WHAT:** Remove step 3 from task-006 (task_invocation_allowlist additions).
**WHY:** Contradicts the leaf-agent design, creates unguarded $5/call invocation path.
**HOW:** Delete the task_invocation_allowlist bullet from task-006 description and acceptance criteria.

---

### Major Issues (Should Fix, Can Proceed with Caution)

| ID | Layer | Location | Issue | Impact | Recommendation |
|----|-------|----------|-------|--------|----------------|
| M-1 | Assumptions | specs.md shared fields + task-001 | `effort: high` is a phantom field | Dead field violating own constraints | Remove from specs or define consumer |
| M-2 | Architecture Smells | task-001 | Scaffolder (Haiku+Thinking) writing Opus-tier agent body content | Quality risk for domain-specific review checklists | Upgrade task-001 agent to sonnet or split body writing |
| M-3 | Cost-Benefit | specs.md + review.json comparison | Inherited budget/timeout documentation mismatch from /review | Future confusion about authoritative values | Ensure new SKILL.md values match team config template exactly |
| M-4 | Contractor Readiness | task-001 | 6 agents in single task with ~3000-word description | Scaffolder context overflow risk, single point of failure | Consider splitting into 2 tasks of 3 agents each |

**Detail for M-1:**

The specs.md line 67 declares `effort: high` as a shared frontmatter field for all 6 agents. However:
- `effort` does not exist in ANY current agent frontmatter file (0 matches across `.claude/agents/**/*.md`)
- `effort` does not exist in ANY agents-index.json entry (0 matches)
- No consumer (hook, validator, or routing logic) reads this field
- The plan's own anti-patterns list includes "Dead frontmatter fields" — this would violate that constraint

Furthermore, task-001's description lists shared fields as "model=opus, thinking={enabled: true, budget: 32000}, tier=3, category=bioinformatics-review, tools=[Read, Glob, Grep], failure_tracking, cost_ceiling=5.00" — `effort: high` is NOT in this list. It exists in specs.md but was dropped in the implementation plan, confirming it will likely be omitted silently.

**Recommendation:** Either (a) remove `effort: high` from specs.md shared fields, or (b) if this is a new field being introduced, document its consumer and add it explicitly to task-001's field list and acceptance criteria.

**Detail for M-2:**

Task-001 assigns agent: `scaffolder` (Haiku+Thinking tier, ~$0.001/1K tokens) to create 6 Opus-tier bioinformatics reviewer agents. The task description is extremely detailed (~3000 words), which compensates for scaffolder's lower reasoning capability. However:

- Each agent body requires domain-specific content: review checklists for genomics variant calling, proteomics FDR control, top-down deconvolution, etc.
- The task says "boilerplate body sections (Identity, Review Checklist, Severity Classification...)" and "adapt from backend-reviewer.md pattern"
- Backend-reviewer's body sections are general software review patterns. Bioinformatics review checklists require domain expertise that Haiku may not render well.
- The user stated these will be "expanded later via braintrust" — so scaffolder-quality bodies are intentional.

**Recommendation:** If boilerplate-quality bodies are acceptable (to be expanded later), acknowledge this explicitly in the task description: "Body sections are BOILERPLATE placeholders. Domain-specific content will be refined via /braintrust in a follow-up session." This prevents the scaffolder from attempting domain expertise it doesn't have, and sets expectations for the validation gate. Alternatively, use `tech-docs-writer` (also Haiku+Thinking but documentation-specialized) or upgrade to Sonnet for this task.

**Detail for M-3:**

The existing /review has a documentation inconsistency:
- **SKILL.md Phase 3** says `budget_max_usd: 2.0` (line 169)
- **review.json template** says `budget_max_usd: 10.0` (line 10)
- **SKILL.md cost model** says total $2.30-$4.20 (line 423)
- **SKILL.md Phase 3** says "model: haiku for backend/frontend/standards, sonnet for architect" but review.json says model: sonnet for ALL (they were upgraded, note at line 427-428)

The new plan's specs.md and implementation-plan.json are internally consistent ($25 budget, Opus for all). But task-004 (SKILL.md creation) says "Mirror .claude/skills/review/SKILL.md structure exactly" — if the author copies literally, they may copy stale patterns.

**Recommendation:** Add a note to task-004: "Use review.json template values as authoritative, NOT the SKILL.md Phase 3 hardcoded values (which are stale in the existing /review SKILL.md)."

**Detail for M-4:**

Task-001 is a single task creating 6 agents with a ~3000-word description. The scaffolder agent has limited context window and this task requires:
- Reading backend-reviewer.md as reference
- Creating 6 directories
- Writing 6 .md files with distinct frontmatter + 9 body sections each
- Writing 6 sharp-edges.yaml files
- Total output: ~12-18 files, ~3000-6000 lines

If scaffolder fails at agent 4/6, there's no partial-completion tracking. The validation gate (task-008) only runs after ALL Phase 2 tasks complete.

**Recommendation:** Split task-001 into task-001a (genomics, proteomics, proteogenomics) and task-001b (proteoform, mass-spec, bioinformatician). This halves the per-task scope, allows parallel execution, and isolates failures. Update task-005/006/007 dependencies to require both 001a and 001b.

---

### Minor Issues (Consider Addressing)

| ID | Layer | Location | Issue | Impact | Recommendation |
|----|-------|----------|-------|--------|----------------|
| m-1 | Contractor Readiness | task-005 | "Insert AFTER architect-reviewer and BEFORE review-orchestrator" — fragile positional instruction | Wrong insertion point if index changes | Use a unique JSON key to anchor insertion |
| m-2 | Testing | task-008 | Validation gate uses code-reviewer (Haiku) for 20-point JSON/YAML validation | Haiku may miss subtle validation errors | Consider using scaffolder or sonnet for validation |
| m-3 | Failure Modes | Team config | No graceful degradation if 1 of 4 selected reviewers fails | Review incomplete without indication | Document partial-success handling in SKILL.md |

**Detail for m-1:**

Task-005 says "Insert AFTER the architect-reviewer entry (currently the last reviewer) and BEFORE the review-orchestrator entry." If another agent is added to agents-index.json between now and implementation, this positional instruction becomes wrong. Better to anchor on a unique string like: "Add entries immediately before the entry with `\"id\": \"review-orchestrator\"`."

**Detail for m-2:**

The validation gate (task-008) uses `code-reviewer` (Haiku tier) to run 20 checks including JSON parsing, cross-file consistency, and subagent_type string matching. Code-reviewer is designed for style/convention checking, not schema validation. A scaffolder or sonnet-tier agent would be more reliable for checks requiring precise string comparison across 3+ files.

**Detail for m-3:**

The team config template has all 6 members, and the skill selects 2-4 per invocation. If one reviewer fails after retry, team-run marks it as failed. But the SKILL.md (task-004) doesn't document what happens to the review when 1 of 3 selected reviewers fails. The existing /review SKILL.md also lacks this — it's an inherited gap.

---

## Assumption Register

| # | Assumption | Source | Verified? | Risk if False | Mitigation |
|---|-----------|--------|-----------|---------------|------------|
| A-1 | team-run bypasses spawn_agent validation entirely | specs.md line 313 | **Unverified** — inferred from architecture description | If team-run DOES check spawned_by, all 6 agents would fail to spawn | Verify in `packages/tui/src/mcp/tools/teamRun.ts` or equivalent |
| A-2 | `effort` is a valid frontmatter field with a consumer | specs.md line 67 | **False** — 0 matches in codebase | Dead field, violates anti-patterns constraint | Remove from plan (see M-1) |
| A-3 | Scaffolder can produce 6 complete agent files in one task | task-001 | **Unverified** — no precedent for 6-agent scaffolding | Partial completion, inconsistent outputs | Split task (see M-4) |
| A-4 | Existing /review reviewers are NOT on task_invocation_allowlist | Inferred from routing-schema | **Verified** — confirmed allowlist contains only 8 architect/analysis agents | N/A | Supports removing bioinformatics reviewers from allowlist |
| A-5 | `additionalProperties: false` in reviewer.json blocks extension | specs.md Decision Table | **Reasonable** — standard JSON Schema behavior | If false, could reuse reviewer.json instead of new schema | Low risk, new schema is cleaner regardless |
| A-6 | Opus reduces hallucination risk vs Sonnet for bioinformatics review | specs.md Decision Table | **Partially verified** — /review upgraded Haiku→Sonnet for this reason (SKILL.md line 427-428) | Cost wasted on same hallucination rate | Opus generally better at tool use; risk is acceptable |
| A-7 | Max 4 reviewers sufficient for typical bioinformatics codebases | specs.md constraints | **Reasonable** — most pipelines focus on 1-2 omics domains | Under-coverage for multi-omics pipelines | Acceptable tradeoff given cost; user can run twice |

---

## Dependency Analysis

### Dependency Graph Verification

The documented dependency graph is correct:

```
Wave 1 (parallel): task-001, task-002, task-003  — no dependencies
Wave 2 (parallel): task-004 (needs 001,002,003), task-005 (needs 001), task-006 (needs 001), task-007 (needs 001)
Wave 3 (sequential): task-008 (needs 004,005,006,007)
```

**No circular dependencies detected.**

**Hidden dependency identified:** task-004 (SKILL.md) references the team config template (task-002) and stdin schema (task-003) by filename. If those filenames change during implementation, task-004 would reference nonexistent files. Low risk since filenames are specified in specs.md.

**Parallelization is correct:** Wave 1 tasks are truly independent. Wave 2 tasks all depend on task-001 (agent IDs) but not on each other.

### Bottleneck Analysis

Task-001 is the critical path bottleneck — 4 of 4 Wave 2 tasks depend on it. If task-001 fails, the entire plan blocks. This reinforces recommendation M-4 (split task-001).

---

## Failure Mode Analysis

### Phase 1 Failure (Agent Foundations)

**If task-001 fails at 50%:**
- 3 of 6 agent files created, 3 missing
- Phase 2 tasks that reference missing agents will produce incomplete wiring
- **Recovery:** Delete incomplete directories, restart task-001
- **Data loss:** None (new files only)

**If task-002 or task-003 fails:**
- Only task-004 is blocked (SKILL.md references schemas)
- task-005, 006, 007 can proceed
- **Recovery:** Restart failed task independently

### Phase 2 Failure (Wiring)

**If task-005 fails (agents-index.json):**
- JSON corruption possible — breaks ALL agent routing in the system
- **Recovery:** `git checkout .claude/agents/agents-index.json` (documented in plan)
- **Risk:** If partial edits committed before failure, git checkout loses ALL uncommitted index changes

**If task-006 fails (routing-schema.json):**
- Same risk as task-005 — system-wide impact from JSON corruption
- **Recovery:** `git checkout .claude/routing-schema.json`

**Mandatory Rollback Test:** PASS — rollback procedures are documented for every phase.

### Cross-Phase Failure

**Phase 1 succeeds, Phase 2 fails partially:**
- Agent files exist but aren't wired into routing
- System is in a usable state (existing agents unaffected)
- Orphan agent files are harmless but should be cleaned up

---

## Cost-Benefit Assessment

### Is This Feature Necessary?

Yes. The user explicitly requested bioinformatics-domain review capability. The system currently has code-level review (/review) but no domain-specific review for scientific/bioinformatics pipelines.

### Is the Complexity Justified?

**59 integration points for 6 agents** is significant but follows established patterns. Each integration point is mechanical (copy pattern, change values). The complexity is inherent to the agent registration system, not to this plan.

### Cost Analysis (Concern #2)

| Metric | /review (existing) | /review-bioinformatics (proposed) |
|--------|--------------------|------------------------------------|
| Model tier | Sonnet | Opus |
| Cost per reviewer | $0.50-$1.00 | $2.50-$5.00 |
| Typical invocation | 2-3 reviewers, $1.00-$3.00 | 2-3 reviewers, $5.00-$15.00 |
| Max invocation | 4 reviewers, $2.30-$4.20 | 4 reviewers, $10.00-$20.00 |
| Budget cap | $10.00 | $25.00 |

**5-10x cost increase is justified** given:
1. Bioinformatics review requires deep domain expertise (FDR methodology, variant calling pipelines, deconvolution algorithms)
2. /review was upgraded Haiku→Sonnet specifically because lower-tier models hallucinated findings
3. Opus is the floor for reliable bioinformatics domain reasoning
4. Max 4 reviewers and intelligent selection mitigate cost
5. This is a specialized skill, not a routine invocation

**Sustainability concern:** At $10-$20 per invocation, this is 5-10x the cost of /review. For a bioinformatics lab reviewing pipeline changes weekly, this is ~$40-$80/month — reasonable for the domain value. For frequent use, consider a Sonnet fallback mode or cached review patterns.

### Ongoing Maintenance Cost

Low. Once wired, agents are static config until domain knowledge needs updating. The "expand via braintrust" plan for body content is the main future cost.

---

## Testing Coverage

### Current Test Plan

The plan has NO unit tests, integration tests, or E2E tests. All validation is deferred to task-008 (validation gate) which runs 20 checks.

### Assessment

For a configuration/wiring task, the validation gate IS the test suite. The 20 checks cover:
- File existence (6 checks)
- JSON validity (2 checks)
- Cross-file consistency (8 checks)
- Dead field detection (1 check)
- Schema completeness (3 checks)

**Gap:** No test for runtime behavior. The validation gate checks that files exist and JSON parses, but does not verify:
- team-run can actually spawn these agents
- Stdin schema validates against real bioinformatics file input
- Reviewer selection algorithm correctly identifies omics domains

**Acceptable for Phase 1** (boilerplate creation). Runtime testing should be added when body content is expanded via braintrust.

---

## Architecture Smell Detection

### Smell 1: Cargo Culting Risk

The plan says "mirror /review architecture exactly." The /review architecture has known inconsistencies:
- SKILL.md budget ($2.0) disagrees with review.json budget ($10.0)
- SKILL.md claims per-reviewer model tiers that don't match the template
- No partial-failure handling documented

Mirroring "exactly" means copying these inconsistencies. The plan should mirror the **architecture** (team-run dispatch, reviewer selection, background execution) but fix the **documentation** inconsistencies.

### Smell 2: Premature Abstraction (ABSENT)

The plan correctly avoids premature abstraction — no orchestrator agent, no complex selection framework, no abstract base reviewer. Good.

### Smell 3: Category Naming

Existing code reviewers use `category: "review"`. New bioinformatics reviewers use `category: "bioinformatics-review"`. This is correct — they're distinct categories. No smell.

### Smell 4: 6 Nearly-Identical Agents

All 6 agents share identical model, tier, tools, failure_tracking, cost_ceiling, and body structure. Only triggers, description, conventions_required, and focus_areas differ. This is NOT a smell — it's the correct pattern for domain-specialized reviewers. The alternative (parameterized single agent) would require runtime configuration logic that adds complexity without benefit.

---

## Contractor Readiness

### The Monday Morning Test

**Can a contractor start Monday with ZERO questions?**

Mostly yes. The plan provides:
- Exact file paths for all outputs
- Reference files to mirror (backend-reviewer.md, review.json, reviewer.json, /review SKILL.md)
- Per-agent field values in tables
- Acceptance criteria per task
- Dependency graph

**Red Flag Phrases Found:**

| Phrase | Location | Issue |
|--------|----------|-------|
| "adapt from backend-reviewer.md pattern" | task-001 | Which sections to adapt? How much domain content? |
| "Mirror .claude/skills/review/SKILL.md structure exactly but adapted" | task-004 | "Exactly but adapted" is contradictory — specify what changes |
| "Copy from backend-reviewer pattern" | task-005 | Backend-reviewer has `context_requirements.rules: []` but plan specifies `rules: ["agent-guidelines.md"]` — which one? |

### Knowledge Gap Check

- Bioinformatics domain expertise needed for body content quality (but acknowledged as boilerplate-to-be-expanded)
- agents-index.json structure: well-documented via backend-reviewer reference
- routing-schema.json four update locations: well-specified with line numbers

### Acceptance Criteria Completeness

All 8 tasks have clear acceptance criteria. Task-001 has 8 criteria, task-008 has 4 criteria referencing 20 sub-checks. No criteria rely on subjective judgment — all are verifiable via file existence, JSON parsing, or string matching.

---

## Specific Concerns Addressed

### Concern 1: "No spawned_by/can_spawn for leaf agents — correct given team-run's spawn path?"

**Answer: PARTIALLY CORRECT.** The decision to omit spawned_by/can_spawn is correct for team-run-only agents (matches existing code reviewer pattern). However, the plan contradicts this by adding agents to task_invocation_allowlist (see C-1). Fix: remove from allowlist.

### Concern 2: "Cost model: 4 Opus reviewers at $2.50-$5.00 each = $10-$20. Sustainable?"

**Answer: YES, with caveats.** Justified by domain expertise requirements and Haiku→Sonnet upgrade precedent. The max-4 cap and intelligent selection are appropriate mitigations. For heavy use, consider a future Sonnet fallback mode. See Cost-Benefit section.

### Concern 3: "Are /review architectural weaknesses being inherited?"

**Answer: YES, some.** Budget/model documentation inconsistencies in SKILL.md vs template, and missing partial-failure handling. See M-3 and m-3. The plan should mirror architecture, not documentation bugs.

### Concern 4: "subagent_type field consistency"

**Answer: WELL-HANDLED.** The plan specifies exact subagent_type strings, the validation gate (task-008 check #8) verifies cross-file consistency, and the specs.md tables are unambiguous. This is one of the plan's strengths.

### Concern 5: "Scaffolder for writing Opus agent frontmatter+body?"

**Answer: ADEQUATE WITH RISK.** Scaffolder can handle frontmatter (mechanical). Body content quality for bioinformatics review checklists is the risk. The "boilerplate to be expanded via braintrust" strategy makes this acceptable. See M-2.

### Concern 6: "effort: high in specs.md but NOT in implementation-plan.json"

**Answer: CONFIRMED GAP.** `effort: high` exists in zero agent files and zero agents-index.json entries across the entire system. It's a phantom field with no consumer. It appears in specs.md shared fields but was dropped from the implementation task descriptions. See M-1.

### Concern 7: "max_retries: 1 — sufficient?"

**Answer: YES.** At $2.50-$5.00 per Opus retry, max_retries: 1 is the correct cost/reliability tradeoff. The existing /review uses max_retries: 2 because Sonnet retries cost $0.50-$1.00 (5x cheaper). One retry provides basic reliability; more retries at Opus cost would be wasteful. If the retry also fails, team-run marks the member as failed — acceptable given domain review is advisory, not blocking.

### Concern 8: "No tests required for any task"

**Answer: ACCEPTABLE.** The validation gate (task-008, 20 checks) serves as the test suite for this configuration/wiring work. Runtime testing should be deferred to when body content is expanded. See Testing section.

---

## Commendations

1. **Thorough integration point inventory.** 59 integration points (9 per agent x 6 + 5 shared) explicitly tracked in specs.md. This level of accounting prevents missed wiring — the #1 failure mode for multi-agent registration.

2. **Sound architectural decision: no orchestrator.** Correctly identified that an orchestrator would create tier inversion (Sonnet orchestrating Opus reviewers) and doubled scope for no functional benefit. The team-run dispatch pattern is the right choice.

3. **Cost-aware design throughout.** max_retries: 1 (not 2), max 4 reviewers (not all 6), $5 cost ceiling per reviewer, $25 budget cap, intelligent selection algorithm. Every cost decision is explicitly reasoned with alternatives documented.

4. **Validation gate as final phase.** Task-008's 20-point checklist catches cross-file inconsistencies that individual task acceptance criteria cannot. This is the right pattern for multi-file configuration work.

5. **Explicit decision rationale table.** Every design decision (Opus tier, no orchestrator, max 4, leaf agents, new stdin schema, etc.) has rationale AND alternatives considered. This is exceptional specification quality.

---

## Recommendations

### High Priority

1. **Fix C-1:** Remove all 6 bioinformatics reviewer IDs from task-006's `task_invocation_allowlist` additions. These are team-run-only leaf agents matching the existing code reviewer pattern.

2. **Fix M-1:** Remove `effort: high` from specs.md shared fields. It has no consumer and would be a dead field.

### Medium Priority

3. **Address M-2:** Add explicit note to task-001 that body sections are BOILERPLATE placeholders for future braintrust expansion. Optionally upgrade agent from scaffolder to tech-docs-writer.

4. **Address M-3:** Add note to task-004 that review.json template values are authoritative over SKILL.md Phase 3 hardcoded values.

5. **Address M-4:** Consider splitting task-001 into 2 sub-tasks of 3 agents each to reduce scaffolder scope and isolate failures.

### Low Priority

6. **Address m-1:** Change task-005 insertion instruction from positional to anchor-based.

7. **Address m-3:** Document partial-success handling in SKILL.md task-004 (what happens when 1 of N reviewers fails).

---

## Final Sign-Off

**Reviewed By:** Staff Architect Critical Review
**Review Date:** 2026-04-09
**Review Duration:** ~15 minutes (including context reads)

**Conditions for Approval:**

- [x] C-1 addressed: Remove bioinformatics reviewers from task_invocation_allowlist
- [ ] M-1 acknowledged: Remove `effort: high` phantom field from specs.md

**Recommended Actions:**

1. Fix C-1 in task-006 description and acceptance criteria (remove allowlist step)
2. Fix M-1 in specs.md shared fields (remove `effort: high`)
3. Acknowledge M-2, M-3, M-4 — can fix during implementation
4. Proceed with implementation after C-1 fixed

**Post-Approval Monitoring:**

- Watch task-001 for scaffolder context limits — if it fails, split per M-4
- Verify task-005/006 JSON validity immediately after each edit (corruption is high-impact)
- After all tasks complete, manually invoke `/review-bioinformatics` on a test pipeline to verify end-to-end (not covered by validation gate)
