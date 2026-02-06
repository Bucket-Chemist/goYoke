# Architecture Review Findings

## Summary
- **Total issues found**: 17
- **CRITICAL**: 4 | **HIGH**: 8 | **MEDIUM**: 4 | **LOW**: 1

## Findings

### [CRITICAL] Finding 1: TC-009 Is a Bottleneck Blocking Phase 2

**Tickets**: TC-008, TC-009
**Category**: dependency-graph
**Description**: TC-008 (Go binary, 5-7 days) lists TC-009 (team templates, 2-3 days) as a blocker. TC-009 is an extremely heavy ticket creating 15+ schema files with complete JSON specifications for 3 workflows. This creates a Phase 1→Phase 2 bottleneck where the largest Phase 1 ticket must fully complete before the largest Phase 2 ticket can start.

**Recommendation**: Split TC-009 into:
- TC-009a (1 day): Create minimal braintrust.json + review.json templates with basic structure
- TC-009b (1-2 days): Create full stdin/stdout schemas for all agents
- Have TC-008 start once TC-009a completes, then reference schemas as they become available

**Evidence**: TC-008 lines 5-6: "Blocked By: ... TC-009", TC-009 lines 5-7: "Effort: 2-3 days"

---

### [CRITICAL] Finding 2: CLI Output Format Completely Unverified

**Tickets**: TC-005, TC-008
**Category**: design-principle / missing-piece
**Description**: The entire cost tracking system (TC-008, TC-012, TC-013) depends on parsing `cost_usd` from `claude -p --output-format json`. No official documentation exists for this format. TC-005 itself states this is a HIGH-risk assumption. This is a load-bearing assumption with zero validation before implementation.

**Recommendation**: Before starting TC-008, immediately run TC-005 verification tests. Document actual JSON structure. If `cost_usd` field doesn't exist, this blocks TC-008, TC-012, TC-013 simultaneously.

**Evidence**: TC-005 lines 9-12: "No official documentation exists", TC-008 lines 275-280: depends on this

---

### [CRITICAL] Finding 3: Task Access Policy Has No Implementation Ticket

**Tickets**: TC-007, TC-001, TC-014
**Category**: contradiction
**Description**: TC-007 documents that team-spawned agents (Level 2) should have Task(haiku/sonnet) but not Task(opus). However, TC-007 states "This code change happens in a separate ticket (not TC-007)." The actual enforcement (modifying gogent-validate for Level 2) is not assigned to any of the 16 tickets. The feature will not work.

**Recommendation**: Create a new ticket to modify gogent-validate to selectively allow Task(haiku/sonnet) at Level 2 while blocking Task(opus). Link as blocker for TC-008 testing.

**Evidence**: TC-007 lines 99-127 (shows required changes), line 129 ("separate ticket"), no ticket implements this

---

### [CRITICAL] Finding 4: Mozart Interview Phase Not Specified

**Tickets**: TC-013, TC-009
**Category**: missing-piece
**Description**: TC-013 describes Mozart's team dispatch but the actual interview prompts, user interaction flow, and interview outputs are not specified. Interview outcome determines team composition, scout needs, and budget allocation. Without specifying the interview protocol, Mozart cannot reliably generate correct team configs.

**Recommendation**: TC-013 must include complete interview protocol: what questions, what decision points, what config fields depend on interview answers.

**Evidence**: TC-013 lines 27-47 (describes dispatch but interview is vague), line 945 (references mozart.md but doesn't show content)

---

### [HIGH] Finding 5: Stdin/Stdout File Naming Inconsistency

**Tickets**: TC-008, TC-009, TC-013
**Category**: contradiction
**Description**: Three different naming conventions appear:
- TC-009: `stdin_einstein.json` (agent ID)
- TC-008: `stdin_{name}.json` where name is member.Name
- TC-013: For implementation, member.Name = "TC-001", so file is `stdin_TC-001.json`

This creates ambiguity for implementation workflow where multiple instances of the same agent type might run.

**Recommendation**: Define naming scheme explicitly: braintrust uses agent ID (unique), implementation uses task ID (unique). Document in TC-009 "File Naming Conventions" section.

**Evidence**: TC-009 lines 88, 147; TC-008 line 219; TC-013 line 575

---

### [HIGH] Finding 6: Budget Field Names Contradict Across Schemas

**Tickets**: TC-009, TC-008, TC-013
**Category**: contradiction
**Description**: Three different budget field structures:
- TC-009: nested `"budget": { "max_total_usd", "budget_remaining_usd" }`
- TC-008: flat `BudgetMaxUSD float64`, `BudgetRemainingUSD float64`
- TC-013: `budget_total_usd` and `budget_remaining_usd`

Nested vs flat will cause JSON unmarshaling failures.

**Recommendation**: Standardize on ONE schema. Add validation test: unmarshal sample config.json from TC-013 into TeamConfig struct from TC-008.

**Evidence**: TC-009 lines 73-76 (nested), TC-008 lines 45-47 (flat), TC-013 lines 114-119

---

### [HIGH] Finding 7: Project Root Resolution Underspecified

**Tickets**: TC-006, TC-013, TC-008
**Category**: design-principle
**Description**: TC-006 documents resolution priority (env vars, fallback to pwd) but TC-013 uses `pwd` which is wrong if user is in a subdirectory. No method validates project_root is correct.

**Recommendation**: Implement: check env var → check git root → ask user. Validate detected root contains expected files (go.mod, package.json).

**Evidence**: TC-006 lines 98-117, TC-013 lines 61-65 (vague), TC-013 line 294 (`pwd` too simplistic)

---

### [HIGH] Finding 8: Wave Computation Algorithm Not Specified for Edge Cases

**Tickets**: TC-013, TC-011
**Category**: missing-piece
**Description**: TC-013 describes `gogent-compute-waves` but no algorithm for circular dependencies, partial dependency resolution, or dynamic task discovery. The wave computation binary could fail catastrophically on edge cases.

**Recommendation**: Specify topological sort with cycle detection. Define behavior for missing dependencies (abort? skip?). Add test cases to TC-011.

**Evidence**: TC-013 lines 519-523, line 937 ("detect cycles" but no algorithm)

---

### [HIGH] Finding 9: Heartbeat Timeout Logic Not Specified

**Tickets**: TC-012, TC-008
**Category**: missing-piece
**Description**: TC-012 describes staleness detection (60s) but doesn't define remediation: what happens if stale but process alive? Auto-kill or warn? What if agent legitimately takes 60+ seconds?

**Recommendation**: Define timeout policy: 60s warning, 120s strong warning, 180s auto-cancel. Document what user should do.

**Evidence**: TC-012 lines 84-91, TC-008 lines 415-434

---

### [HIGH] Finding 10: Config.json Atomic Write Recovery Not Specified

**Tickets**: TC-008, TC-004
**Category**: design-principle
**Description**: TC-008 uses write-tmp-then-rename but doesn't specify: what if tmp file left behind on SIGKILL? What if reader and writer race? What about abandoned .tmp files?

**Recommendation**: Document recovery semantics. On startup, check for abandoned .tmp files. TC-012 must handle ENOENT during config reads.

**Evidence**: TC-008 lines 107-125

---

### [HIGH] Finding 11: TC-011 Missing TC-003 Dependency

**Tickets**: TC-011, TC-003
**Category**: dependency-graph
**Description**: TC-011 (unit tests) blocks on TC-002 but should also block on TC-003 (retry fix) since it tests retry behavior that TC-003 implements.

**Recommendation**: Add TC-003 to TC-011's blocked_by list.

**Evidence**: TC-011 tests retry scenarios that TC-003 defines.

---

### [HIGH] Finding 12: TC-013 Orchestrator Modifications Need Separate Design Docs

**Tickets**: TC-013
**Category**: missing-piece
**Description**: TC-013 covers 3 major orchestrator rewrites but only shows skeleton/pseudocode. Actual prompt templates should be in separate design documents for each rewrite.

**Recommendation**: Create Mozart-rewrite.md, ReviewOrch-rewrite.md, ImplMgr-rewrite.md. Include before/after prompt examples. Add test cases showing prompt output produces correct config.json.

**Evidence**: TC-013 lines 49-103, 229-399, 422-656

---

### [MEDIUM] Finding 13: TC-016 PID Check Assumes Same Machine

**Tickets**: TC-016, TC-004
**Category**: design-principle
**Description**: PID-based liveness check only works on same machine. PID reuse by OS is possible.

**Recommendation**: Document limitation. Add timestamp to PID file for staleness detection independent of PID reuse.

**Evidence**: TC-016 lines 62-71

---

### [MEDIUM] Finding 14: TC-014 Fallback Defaults Differ Between Spawn Paths

**Tickets**: TC-014, TC-008, TC-001
**Category**: design-principle
**Description**: Missing `cli_flags` in agents-index.json gets different fallback defaults: TUI path gives full tools (Read, Write, Glob, Grep, Bash, Edit), Go binary gives read-only (Read, Glob, Grep). Inconsistency masks bugs.

**Recommendation**: Use same fallback across all paths. Or require cli_flags to be explicitly defined (fail at startup if missing).

**Evidence**: TC-014 lines 419-427

---

### [MEDIUM] Finding 15: TC-015 SDK Concurrency Is a Complete Unknown

**Tickets**: TC-015
**Category**: missing-piece
**Description**: TC-015's entire design depends on SDK concurrency support which is explicitly "UNANSWERED." If SDK doesn't support concurrency, the whole ticket approach is invalid.

**Recommendation**: Create blocking investigation ticket before any TC-015 work. 1-2 day investigation task.

**Evidence**: TC-015 lines 452-467

---

### [MEDIUM] Finding 16: No Build/Deployment Story

**Tickets**: All
**Category**: missing-piece
**Description**: No ticket covers how the new Go binaries get built and distributed. The project has existing binaries in cmd/ but no ticket specifies Makefile changes, go install paths, or binary distribution.

**Recommendation**: Add to TC-008 acceptance criteria: Makefile updated, `make build` produces all new binaries.

**Evidence**: No Makefile or build instructions in any ticket.

---

### [LOW] Finding 17: TC-010 Markdown Output Not Validated for Beethoven

**Tickets**: TC-010, TC-009, TC-013
**Category**: missing-piece
**Description**: TC-010 generates pre-synthesis.md but no validation against Beethoven's expected structure.

**Recommendation**: Add validation test: generate pre-synthesis.md, verify expected sections present.

**Evidence**: TC-010 lines 160-217, TC-009 stdin/beethoven.json

---

## Dependency Graph Assessment

**Valid**: PARTIAL (Major issues found)

### Issues Found:
1. **Bottleneck**: TC-009 (2-3 days) blocks TC-008 — split TC-009 into MVP + extensions
2. **Missing ticket**: No ticket modifies gogent-validate for Level 2 Task enforcement
3. **Missing ticket**: No SDK concurrency investigation ticket blocking TC-015
4. **Missing dependency**: TC-011 should block on TC-003 (not just TC-002)

### Dependency Verification:
- TC-003 → TC-002 ✅
- TC-008 → TC-001, TC-002, TC-004, TC-006, TC-009, TC-014 ✅ (but TC-009 is bottleneck)
- TC-010 → TC-008 ✅
- TC-011 → TC-002 ✅ (should also → TC-003)
- TC-012 → TC-008 ✅
- TC-013 → TC-012 ✅
- TC-015 independent ✅ (but needs SDK investigation)
- TC-016 → TC-008 ✅

---

## Risk Matrix

| Integration Point | Risk Level | Mitigation |
|-------------------|------------|-----------|
| CLI output format (TC-005 → TC-008) | CRITICAL | Verify immediately before Phase 2 |
| gogent-validate enforcement (TC-007) | CRITICAL | Create new ticket, must complete before TC-008 testing |
| Schema bottleneck (TC-009 → TC-008) | CRITICAL | Split TC-009 into TC-009a + TC-009b |
| Mozart interview protocol (TC-013) | CRITICAL | Specify complete interview protocol |
| File naming conventions (TC-009/TC-008/TC-013) | HIGH | Standardize and document |
| Budget field structure (TC-009/TC-008/TC-013) | HIGH | Choose flat or nested, validate |
| Project root resolution (TC-006/TC-013) | HIGH | Implement proper detection |
| Wave computation edge cases (TC-013) | HIGH | Specify algorithm, test cycles |
| Heartbeat timeout policy (TC-008/TC-012) | HIGH | Define remediation thresholds |
| SDK concurrency (TC-015) | CRITICAL | Investigate before any TC-015 work |

**Overall Assessment**: Architecture is sound but specifications have 4 critical blockers and 8 high-risk integration points that will cause implementation failures if not resolved first.
