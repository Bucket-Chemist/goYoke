# Frontend Review Findings

## Summary
- **Total issues found**: 8
- **CRITICAL**: 1 | **HIGH**: 3 | **MEDIUM**: 3 | **LOW**: 1

## Findings

### [CRITICAL] TC-015: Phase 1 SDK Concurrency Investigation Missing From Acceptance Criteria

**Tickets**: TC-015
**Category**: blocking-dependency
**Description**: TC-015 implementation depends critically on answering: "Does Claude Agent SDK support concurrent `query()` calls?" This is documented in "Open Questions" section (Q1) but NOT in the Acceptance Checklist. If the SDK does not support concurrent queries, the entire Phase 2 refactoring becomes invalid. Alternative approaches (separate processes, SDK upgrade request, redesign) are needed.

**Recommendation**: Move Phase 1 investigation into Acceptance Criteria section AND add it as a blocking dependency:
```
- [ ] PHASE 1 INVESTIGATION COMPLETE: SDK concurrency semantics documented
  - Confirmed: SDK supports concurrent query() calls, OR
  - Documented: SDK limitation + alternative approach identified
```

Make it clear that Phase 2 cannot begin without this answer.

**Evidence**: TC-015 section "Phase 1: Understand SDK Constraints" describes investigation but it's not in acceptance criteria (line 505-519).

---

### [HIGH] TC-015: Root Cause Analysis Line Numbers Are Approximate But Analysis Is Sound

**Tickets**: TC-015
**Category**: root-cause-analysis
**Description**: TC-015's root cause analysis claims specific line numbers. Upon verification against actual useClaudeQuery.ts code:
- Line 152-153 CORRECT (isStreaming state + streamingRef)
- Line 600-603 CORRECT (guard blocks concurrent sendMessage)
- Line 629-631 CORRECT (setIsStreaming + streamingRef both set)
- Line 575-578 OFF BY 1 (actually 575-578 in handleResultEvent, Einstein said 576-578)
- Line 730 EXACTLY CORRECT (for await loop)

The analysis is fundamentally correct: `streamingRef.current` acts as a global mutex that blocks user input during any active query.

**Recommendation**: Update TC-015 acceptance criteria to note: "Line number references are approximate; actual lines verified to match logic flow exactly." This prevents implementers from searching for exact line matches during code review.

**Evidence**: Lines 600-603, 629-631, 575-578, 730 in useClaudeQuery.ts all verified.

---

### [HIGH] TC-015: Event Flow Missing Sequence Diagram

**Tickets**: TC-015
**Category**: concurrency-model
**Description**: TC-015 correctly identifies application-level freeze (not event loop). However, the exact event flow causing the freeze should be clearer with a diagram: `Mozart query → MCP tool invoke → Einstein CLI → (freeze here for 30s) → completion event → exit await loop`. During Einstein's 30+ second execution, streamingRef.current === true blocks all new user messages.

**Recommendation**: Add a sequence diagram showing the exact event flow that causes the freeze. Make it explicit that the guard at line 600 is the single point of failure.

**Evidence**: useClaudeQuery.ts line 600: `if (streamingRef.current) { return; }`, line 335-361: handleBuiltinToolUse processes tools within event stream.

---

### [HIGH] TC-012: Slash Commands Should Validate Session Directory Discovery

**Tickets**: TC-012
**Category**: ux-robustness
**Description**: TC-012 slash commands depend on session directory discovery via `GOGENT_SESSION_DIR`. The ticket documents the design decision but doesn't specify what happens if: env var unset (fallback documented), env var points to invalid path (no error handling), multiple sessions active (no precedence), session dir moved/deleted.

**Recommendation**: Add to TC-012 acceptance criteria:
- Skills handle missing GOGENT_SESSION_DIR gracefully (fallback to scan)
- Skills handle invalid session dir path gracefully (error message, not crash)
- TUI integration verified: process.env.GOGENT_SESSION_DIR set during session init

**Evidence**: TC-012 "Design Decision: Session Directory Discovery" section.

---

### [MEDIUM] TC-012: Status Indicators May Not Render on Non-ANSI Terminals

**Tickets**: TC-012
**Category**: accessibility
**Description**: TC-012 defines status indicators using Unicode symbols (✓, ⏳, ⏸, ✗, 🔄). These may not render correctly on limited terminals, SSH sessions without UTF-8, or screen readers.

**Recommendation**: Add fallback ASCII indicators: ✓/[OK], ⏳/[..], ⏸/[||], ✗/[XX]. Document in output formatting conventions.

**Evidence**: TC-012 "Output Format" section uses Unicode symbols without fallback.

---

### [MEDIUM] TC-012: Multi-Reviewer Conflict Resolution Unclear

**Tickets**: TC-012
**Category**: data-aggregation
**Description**: `/team-result` for review workflow aggregates findings by severity but doesn't specify conflict resolution when reviewers disagree on severity for the same issue.

**Recommendation**: Add conflict resolution logic: use highest severity reported, show attribution (which reviewer(s) reported it), if disagreement show both severities.

**Evidence**: TC-012 "Output Format for Review" shows severity grouping but not conflict resolution.

---

### [MEDIUM] TC-009: Stdin Schemas Missing Optional Field Indicators

**Tickets**: TC-009
**Category**: schema-clarity
**Description**: TC-009 defines stdin schemas but JSON schema doesn't explicitly mark optional fields. For example, `reads_from.scout_metrics` may not exist if scouts didn't run. The schema should clarify required vs optional.

**Recommendation**: Update stdin schemas to explicitly mark optional fields. Add to agent prompt envelopes: "If optional fields are missing, proceed with available context."

**Evidence**: TC-009 stdin schemas use "required" array inconsistently.

---

### [LOW] StatusLine Doesn't Reflect Background Team Status

**Tickets**: TC-012, TC-015
**Category**: ux-consistency
**Description**: After TC-015 concurrent queries and TC-013 orchestrator rewrites, background teams are tracked in config.json, not TUI store. StatusLine shows streaming state but not background team activity. After `/braintrust` dispatches a team, StatusLine shows "0 running" because agents are in background process, not store.

**Recommendation**: Add to TC-012 and TC-015 acceptance criteria: StatusLine accurately reflects background team status after /braintrust.

**Evidence**: StatusLine.tsx lines 159-171 compute agentCounts from store only.

---

## Cross-Ticket Dependencies

| From → To | Type | Risk |
|-----------|------|------|
| TC-012 → TC-015 | Blocking | /team-status won't work if concurrent queries fail |
| TC-015 → TC-013 | Blocking | Orchestrator rewrites depend on TUI responsiveness |
| TC-009 → TC-008 | Blocking | Stdin schemas must be implemented before Go binary |
| TC-012 → StatusLine | Enhancement | Background team visibility depends on StatusLine update |

**Overall Assessment**: Frontend ticket suite is production-ready for Phase 1 (MVP) with TC-015 SDK concurrency investigation as the critical gate.
