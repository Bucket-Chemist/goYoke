# TypeScript TUI Cutover Sign-Off Checklist

**Purpose:** Comprehensive validation before switching from Go TUI to TypeScript TUI in production.

**Status:** Ready for execution

**Date of Sign-Off:** ___________________________

**Authorized By:** ___________________________

---

## Executive Summary

This checklist verifies that the TypeScript TUI migration meets all acceptance criteria across functionality, quality, performance, and compatibility. Only after all items pass should the cutover be authorized.

**Migration Context:**
- From: Go TUI (deprecated after cutover)
- To: TypeScript TUI (ink-based React interface)
- Duration of Go TUI retention: 90 days minimum post-cutover
- Rollback plan: Active for 90 days

---

## 1. Functionality Verification

### Core Message Flow

- [ ] Send message: User can send text messages
- [ ] Receive response: Messages received from Claude
- [ ] Message history: Full conversation visible in scrollback
- [ ] Cost tracking: Cost updates with each tool execution
- [ ] Message timestamp: Each message has readable timestamp

**Status:** PASS / FAIL

**Notes:**
```
[Any functionality gaps or deviations]
```

---

### Interactive Tools

- [ ] `ask_user`: Modal prompt appears, user can respond
- [ ] `ask_user`: Response captured and sent to agent
- [ ] `confirm_action`: Yes/No confirmation dialog works
- [ ] `confirm_action`: Choice correctly captured
- [ ] `request_input`: Input field accepts text
- [ ] `request_input`: Input sent to agent without modification
- [ ] `select_option`: List of options displays correctly
- [ ] `select_option`: Selection saved and returned

**Status:** PASS / FAIL

**Evidence:** [Link to test results or manual verification notes]

**Notes:**
```
[Any tool interaction issues]
```

---

### Agent Navigation

- [ ] Agent tree renders: Tree structure visible and readable
- [ ] Agent selection: Can click/navigate to agent in tree
- [ ] Agent expand/collapse: Can expand/collapse agent groups
- [ ] Agent icons: Icons display correctly for different agent types
- [ ] Agent status: Agent status (online/idle) updates
- [ ] Session context: Selected agent persists across interactions

**Status:** PASS / FAIL

**Notes:**
```
[Any agent tree navigation issues]
```

---

### Session Management

- [ ] List sessions: `--list` or menu shows all saved sessions
- [ ] Session resume: `--session <id>` loads previous session
- [ ] Auto-save: Cost and metrics saved after each tool execution
- [ ] Session switching: Can switch between sessions in same run
- [ ] New session creation: Typing in new session starts fresh
- [ ] Session history: Full message history loads on resume

**Status:** PASS / FAIL

**Evidence:** [Session IDs tested, number of sessions managed]

**Notes:**
```
[Any session management issues]
```

---

### Auto-Restart Capability

- [ ] Crash recovery: TUI restarts after unexpected termination
- [ ] Session recovery: Session state preserved after restart
- [ ] MCP reconnection: MCP client re-established after restart
- [ ] No data loss: Messages and cost preserved after restart

**Status:** PASS / FAIL

**Test Procedure:**
```
[How auto-restart was tested]
```

**Notes:**
```
[Any restart issues]
```

---

## 2. Quality Assurance

### TypeScript Compilation

```bash
cd packages/tui
npm run typecheck
```

- [ ] Zero TypeScript errors
- [ ] Zero TypeScript warnings
- [ ] Strict mode enabled: `strict: true` in tsconfig.json
- [ ] No `any` types in core files

**Output:**
```
[Paste typecheck output here]
```

**Status:** PASS / FAIL

---

### Code Quality: ESLint

```bash
npm run lint
```

- [ ] Zero ESLint errors
- [ ] Zero ESLint warnings
- [ ] All rules in `eslint.config.mjs` passing
- [ ] Naming conventions followed (camelCase, PascalCase for components)

**Output:**
```
[Paste lint output here]
```

**Status:** PASS / FAIL

---

### Test Coverage

```bash
npm test -- --coverage
```

**Coverage by category:**

| Category | Target | Actual | Status |
|----------|--------|--------|--------|
| Statements | >80% | ___% | ✅/❌ |
| Branches | >75% | ___% | ✅/❌ |
| Functions | >80% | ___% | ✅/❌ |
| Lines | >80% | ___% | ✅/❌ |

**Status:** PASS (all >80%) / FAIL

**Notes:**
```
[Any low-coverage areas or untested edge cases]
```

---

### Unit Test Results

```bash
npm test
```

- [ ] All unit tests passing
- [ ] No flaky tests
- [ ] No skipped tests
- [ ] Error handling tests comprehensive

**Test Summary:**
```
[Paste test summary output here]
Example:
Test Files  3 passed (3)
Tests       42 passed (42)
Start       12:34:56
Duration    1.23s
```

**Status:** PASS / FAIL

---

### Integration Test Results

```bash
npm run test:integration
```

**Test suites:**

- [ ] `session.test.ts` - Session CRUD and format validation: PASS/FAIL
- [ ] `mcp.test.ts` - MCP tool invocation: PASS/FAIL
- [ ] `e2e.test.ts` - End-to-end workflows: PASS/FAIL
- [ ] `concurrent.test.ts` - Concurrent operations: PASS/FAIL
- [ ] `errors.test.ts` - Error handling and recovery: PASS/FAIL

**Summary:**
```
[Paste integration test results here]
```

**Status:** PASS (all suites) / FAIL

---

### Memory Profiling

```bash
npm run benchmark
```

**Memory metrics:**

| Metric | Measured | Target | Status |
|--------|----------|--------|--------|
| Memory idle | ___MB | <80MB | ✅/❌ |
| Memory active | ___MB | <100MB | ✅/❌ |
| Cold start | ___ms | <500ms | ✅/❌ |
| Input latency | ___ms | <32ms | ✅/❌ |

**Comparison to Go baseline:**
```
[Notes on performance vs Go version]
```

**Status:** PASS (all within target) / FAIL

**Reference:** See `performance-report.md` for detailed analysis

---

### No Memory Leaks

- [ ] Sustained operation test: TUI runs for 30+ minutes without memory increase
- [ ] Memory usage stable: No gradual memory growth
- [ ] Cleanup verified: Session files cleaned on exit
- [ ] Resource cleanup: All timers/listeners cleaned

**Test Evidence:**
```
[Describe sustained operation test and results]
```

**Status:** PASS / FAIL

---

## 3. Performance Verification

### Reference Baseline (Go TUI)

| Metric | Go Baseline | Notes |
|--------|-------------|-------|
| Cold start | ~4ms | Compiled binary |
| Memory idle | N/A | Not measured for Go |
| Input latency | ~7ms | Native event handling |

**Source:** Compare against `performance-report.md`

---

### TypeScript TUI Performance

| Metric | Measured | Target | Status |
|--------|----------|--------|--------|
| Cold start | ___ms | <500ms | ✅/❌ |
| Memory idle | ___MB | <80MB | ✅/❌ |
| Memory active | ___MB | <100MB | ✅/❌ |
| Input latency | ___ms | <32ms | ✅/❌ |

**Test Environment:**
```
Hardware: [CPU, RAM, OS]
Node version: [version]
Package manager: [npm/yarn/pnpm]
```

**Status:** PASS (all within target) / FAIL

---

### Load Testing

- [ ] 50 messages in quick succession: No lag or freezing
- [ ] Large message handling: 5000+ character messages display correctly
- [ ] Rapid navigation: Switching between agents 20+ times
- [ ] Sustained operation: 30+ minutes without degradation

**Results:**
```
[Load test results and observations]
```

**Status:** PASS / FAIL

---

## 4. Compatibility Verification

### Terminal Emulator Support

Test in at least 5 different terminal emulators:

| Terminal | Version | Works | Issues |
|----------|---------|-------|--------|
| xterm | [version] | ✅/❌ | [notes] |
| GNOME Terminal | [version] | ✅/❌ | [notes] |
| Alacritty | [version] | ✅/❌ | [notes] |
| Kitty | [version] | ✅/❌ | [notes] |
| iTerm2 (macOS) | [version] | ✅/❌ | [notes] |

**Status:** PASS (works in 5+) / FAIL

**Notes:**
```
[Terminal-specific issues or limitations]
```

---

### Terminal Features

- [ ] Colors render correctly (256-color support)
- [ ] Unicode characters display (box drawing, arrows, etc.)
- [ ] Mouse support (if applicable)
- [ ] Resize handling: Can resize terminal during run
- [ ] Scrollback: Can scroll back through history

**Status:** PASS / FAIL

**Notes:**
```
[Terminal feature issues]
```

---

### Go TUI Compatibility (Format & Hooks)

**Session Format Compatibility:**

- [ ] Sessions created in TS readable by Go: PASS/FAIL
- [ ] Sessions created in Go readable by TS: PASS/FAIL
- [ ] Field names are `snake_case`: VERIFIED
- [ ] Date format is ISO 8601 with Z: VERIFIED
- [ ] All required fields present: VERIFIED
- [ ] No extra camelCase variants: VERIFIED

**Test Details:**
```
[Session IDs tested, verification steps]
```

**Status:** PASS / FAIL

---

**MCP Hook Compatibility:**

- [ ] Go MCP hooks function with TS TUI: PASS/FAIL
- [ ] Tool execution triggers hooks: PASS/FAIL
- [ ] Hooks complete without errors: PASS/FAIL
- [ ] Hook outputs processed correctly: PASS/FAIL

**Status:** PASS / FAIL

---

### CLI Flag Compatibility

- [ ] `--session <id>`: Resume specific session
- [ ] `--list`: List available sessions
- [ ] `--legacy`: Fallback to Go (if available)
- [ ] `--verbose`: Enable verbose logging
- [ ] `--version`: Display version info
- [ ] `-h/--help`: Display help text

**Status:** PASS (all flags work) / FAIL

**Notes:**
```
[Any CLI compatibility issues]
```

---

## 5. Data Integrity & Rollback

### Rollback Rehearsal Results

Reference: `rollback-rehearsal.md`

- [ ] Rollback rehearsal completed: YES/NO
- [ ] Session loads in Go: YES/NO
- [ ] Cost preserved: YES/NO
- [ ] Message history intact: YES/NO
- [ ] No data loss: YES/NO

**Session ID Tested:** `[session-id]`

**Rehearsal Date:** ___________________________

**Tester:** ___________________________

**Status:** PASS / FAIL

**See rollback-rehearsal.md for detailed results**

---

### Data Format Validation

- [ ] Session files use snake_case: VERIFIED
- [ ] Timestamps are ISO 8601: VERIFIED
- [ ] No undefined/null required fields: VERIFIED
- [ ] Float precision preserved (cost): VERIFIED
- [ ] Optional fields handled: VERIFIED

**Sample Session Checked:**
```json
[Paste sample session file here]
```

**Status:** PASS / FAIL

---

## 6. Documentation Complete

### README & Getting Started

- [ ] README updated with TS TUI instructions
- [ ] Installation steps accurate
- [ ] Build instructions work end-to-end
- [ ] Running TUI section covers all scenarios

**Location:** `/packages/tui/README.md`

**Status:** PASS / FAIL

---

### Migration Notes

- [ ] Migration notes added to documentation
- [ ] Data compatibility explained
- [ ] Rollback procedure documented
- [ ] 90-day retention of Go TUI explained

**Location:** `/packages/tui/docs/migration-notes.md` or README section

**Status:** PASS / FAIL

---

### Performance Documentation

- [ ] Performance report up to date
- [ ] Metrics include TS vs Go comparison
- [ ] Baseline and targets documented
- [ ] Methodology explained

**Location:** `/packages/tui/docs/performance-report.md`

**Status:** PASS / FAIL

---

### Operational Documentation

- [ ] Session persistence documented: `session-persistence.md` ✅/❌
- [ ] Terminal compatibility documented: `terminal-compatibility.md` ✅/❌
- [ ] Keyboard shortcuts documented: `keyboard-handling.md` ✅/❌
- [ ] Error handling documented: `error-boundary-usage.md` ✅/❌
- [ ] Benchmarking procedures documented: `benchmarking.md` ✅/❌

**Status:** PASS (all documented) / FAIL

---

## 7. Risk Assessment

### Known Limitations

Document any known issues or limitations:

```markdown
1. [Limitation]: [Impact] [Mitigation]
2. [Limitation]: [Impact] [Mitigation]
3. [Limitation]: [Impact] [Mitigation]
```

**Risk Level:** LOW / MEDIUM / HIGH

---

### Rollback Triggers

The following conditions require rollback to Go TUI:

- [ ] Session format incompatibility discovered
- [ ] Data loss during normal operation
- [ ] Memory leaks causing performance degradation
- [ ] Terminal compatibility issues in critical environment
- [ ] MCP hook failures causing tool execution failure
- [ ] Cost tracking inaccuracies

**Status:** Triggers understood and documented

---

### Contingency Plan

If rollback required:

1. Switch to Go TUI: `gofortress-tui --legacy --session <id>`
2. Check session loads without errors
3. Verify data integrity after switch
4. Notify stakeholders
5. Investigate cause in TS version
6. Plan remediation

**Contact for Rollback:** [Name, Contact Info]

---

## 8. Sign-Off Authorization

### Technical Lead Sign-Off

**Name:** ___________________________

**Title:** ___________________________

**Date:** ___________________________

**Signature:** ___________________________

**Notes:**
```
[Any technical concerns or final observations]
```

---

### Project Lead Sign-Off

**Name:** ___________________________

**Title:** ___________________________

**Date:** ___________________________

**Signature:** ___________________________

**Notes:**
```
[Project-level sign-off notes]
```

---

### QA Lead Sign-Off

**Name:** ___________________________

**Title:** ___________________________

**Date:** ___________________________

**Signature:** ___________________________

**Testing Summary:**
```
[QA final summary of testing performed]
```

---

## 9. Pre-Cutover Checklist

Before authorizing cutover, verify:

- [ ] All functionality tests PASS
- [ ] All quality gates PASS
- [ ] Performance metrics within target
- [ ] Compatibility verified in 5+ terminals
- [ ] Rollback rehearsal PASS
- [ ] All documentation complete and accurate
- [ ] All sign-offs completed
- [ ] No open issues blocking cutover
- [ ] Go TUI backed up to deprecated/
- [ ] Team trained on new TS TUI
- [ ] Support plan in place for 90-day monitoring

**Pre-Cutover Checklist:** PASS / FAIL

**Authorized for Cutover:** YES / NO

---

## 10. Post-Cutover Monitoring

### Week 1 Monitoring

- [ ] No emergency rollback triggers activated
- [ ] Session creation/resumption working
- [ ] Cost tracking accurate
- [ ] User reports: No critical issues
- [ ] Performance stable

**Status:** ✅ / ⚠️ / ❌

**Notes:**
```
[Any issues or observations from early use]
```

---

### Week 2-4 Monitoring

- [ ] Memory usage stable under load
- [ ] No data loss incidents
- [ ] All features working as expected
- [ ] Performance consistent
- [ ] Terminal compatibility confirmed

**Status:** ✅ / ⚠️ / ❌

---

### 90-Day Review

At 90 days post-cutover:

- [ ] Confirm no critical issues found
- [ ] All performance metrics stable
- [ ] User adoption successful
- [ ] Ready to archive Go TUI

**Status:** PASS / FAIL

**Next Steps:** [Remove Go TUI or extend monitoring period]

---

## Appendices

### A. Test Environment

**Hardware:**
- CPU: ___________________________
- RAM: ___________________________
- Disk: ___________________________
- OS: ___________________________

**Software:**
- Node version: ___________________________
- npm version: ___________________________
- TypeScript version: ___________________________

---

### B. Test Evidence

Attach or reference:
- [ ] Test run output (all tests)
- [ ] Coverage report
- [ ] Performance benchmark results
- [ ] Rollback rehearsal results
- [ ] Terminal compatibility matrix

---

### C. Reference Documents

- `packages/tui/README.md` - Usage and installation
- `packages/tui/docs/rollback-rehearsal.md` - Rollback procedure
- `packages/tui/docs/performance-report.md` - Performance analysis
- `packages/tui/docs/session-persistence.md` - Session format specification
- `packages/tui/docs/terminal-compatibility.md` - Terminal support matrix
- `packages/tui/docs/keyboard-handling.md` - Keyboard shortcuts

---

### D. Contact & Escalation

**Technical Lead:** [Name] - [Phone/Email]

**Project Lead:** [Name] - [Phone/Email]

**On-Call Support:** [Process for 90-day monitoring period]

---

## Cutover Decision

After all items reviewed:

### AUTHORIZED FOR CUTOVER: YES / NO

**Cutover Date:** ___________________________

**Cutover Window:** ___________________________

**Rollback Contact:** ___________________________

**Success Criteria Met:** YES / NO

---

**Document Version:** 1.0
**Last Updated:** 2026-02-04
**Migration Phase:** 8 (Cutover Validation)
**Ticket:** TUI-022

*For questions or issues, reference rollback-rehearsal.md or contact technical lead.*
