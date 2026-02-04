# Rollback Rehearsal: TypeScript TUI ↔ Go TUI Migration

**Purpose:** Verify that sessions created in the TypeScript TUI can be resumed in the Go TUI (and vice versa) without data loss.

**Status:** Ready for execution

---

## Pre-Requisites

- [ ] Both TUI versions built and available
  - TypeScript: `npm run build` in `packages/tui/`
  - Go: Available in deprecated binary or git checkout
- [ ] Both commands available in PATH:
  - `gofortress-tui` (TypeScript version)
  - `gofortress-tui --legacy` (Go fallback, or separate binary)
- [ ] Session directory exists: `~/.claude/sessions/`
- [ ] All integration tests pass (see Test Suite section below)

---

## Test Execution Procedure

### Phase 1: Setup

1. Start fresh: Ensure no active sessions
   ```bash
   # Check current session directory
   ls ~/.claude/sessions/

   # Optional: backup existing sessions
   mkdir -p ~/.claude/sessions-backup
   cp ~/.claude/sessions/* ~/.claude/sessions-backup/ 2>/dev/null || true
   ```

2. Note test start time
   ```
   Test Start: [YYYY-MM-DD HH:MM:SS]
   Tester: [Your Name]
   ```

### Phase 2: Create Session in TypeScript TUI

1. Launch TypeScript TUI:
   ```bash
   ./packages/tui/bin/gofortress-tui
   ```

2. Interact with the TUI:
   - [ ] Verify UI renders correctly
   - [ ] Send at least 10 messages in conversation
   - [ ] Use at least 2-3 different tools/actions:
     - Request user input
     - Confirm action
     - Select option from list
   - [ ] Monitor cost accumulation (bottom right)
   - [ ] Note final cost value

3. Record session metrics:
   ```markdown
   ### TS TUI Metrics
   - Session ID: [copy from list view or logs]
   - Messages sent: [count]
   - Final cost: $[amount]
   - Tools used: [list tools]
   - Session duration: [HH:MM:SS]
   ```

4. Exit TypeScript TUI (Ctrl+C)

5. Verify session file was created:
   ```bash
   ls -la ~/.claude/sessions/<session-id>.json
   cat ~/.claude/sessions/<session-id>.json | jq .
   ```

   Expected output format:
   ```json
   {
     "id": "...",
     "name": "...",
     "created_at": "2026-02-XX...",
     "last_used": "2026-02-XX...",
     "cost": 0.XX,
     "tool_calls": XX
   }
   ```

### Phase 3: Rollback to Go TUI

1. Launch Go TUI with session resume:
   ```bash
   gofortress-tui --legacy --session <session-id>
   ```

2. Verify session loads correctly:
   - [ ] Session loads without errors
   - [ ] Message history visible (at least 10 messages)
   - [ ] Cost value preserved and displayed
   - [ ] Agent tree displays correctly
   - [ ] Can scroll through conversation

3. Resume conversation in Go TUI:
   - [ ] Send 2-3 new messages
   - [ ] Verify cost increases by expected amount
   - [ ] Verify new messages appear in history

4. Record Go TUI metrics:
   ```markdown
   ### Go TUI Metrics
   - Session loaded: [YES/NO]
   - Session data readable: [YES/NO]
   - Cost preserved: [YES/NO ($X.XX)]
   - Message count: [count]
   - New messages added: [count]
   - New cost: $[amount]
   - Navigation responsive: [YES/NO]
   ```

5. Exit Go TUI (Ctrl+C)

### Phase 4: Restore to TypeScript TUI

1. Launch TypeScript TUI with same session:
   ```bash
   ./packages/tui/bin/gofortress-tui --session <session-id>
   ```

2. Verify complete data integrity:
   - [ ] Session loads without errors
   - [ ] All previous messages present (from TS + new from Go)
   - [ ] Cost matches Go TUI value
   - [ ] Message count reflects all interactions
   - [ ] Can scroll through full conversation history
   - [ ] Can continue conversation from where Go left off

3. Send 1-2 final messages to verify continuity

4. Record final metrics:
   ```markdown
   ### TS TUI Restore Metrics
   - Session loaded: [YES/NO]
   - Message history complete: [YES/NO]
   - Cost preserved: [YES/NO ($X.XX)]
   - Total messages: [count]
   - Data integrity: [PASS/FAIL]
   - Can continue: [YES/NO]
   ```

5. Exit TypeScript TUI (Ctrl+C)

---

## Results Template

Copy this template and fill in actual values during test execution.

### Test Session: [Session ID]

**Date:** 2026-XX-XX
**Tester:** [Name]
**Start Time:** [HH:MM:SS]
**End Time:** [HH:MM:SS]
**Duration:** [HH:MM:SS]

---

#### Phase 1: TypeScript TUI Session Creation

| Metric | Value | Status |
|--------|-------|--------|
| Session created | [YES/NO] | ✅/❌ |
| Session ID | [id] | - |
| Messages sent | [count] | - |
| Tools used | [names] | - |
| Cost accumulated | $[amount] | - |
| Session file valid | [YES/NO] | ✅/❌ |

**Notes:**
```
[Observations, issues, or deviations]
```

---

#### Phase 2: Go TUI Rollback

| Metric | Value | Status |
|--------|-------|--------|
| Session loaded | [YES/NO] | ✅/❌ |
| Error on load | [none/describe] | - |
| Message history present | [YES/NO] | ✅/❌ |
| Cost preserved | $[amount] (matched: ✓/✗) | ✅/❌ |
| Conversation resumed | [YES/NO] | ✅/❌ |
| New messages added | [count] | - |
| Cost after: | $[amount] | - |
| Navigation responsive | [YES/NO] | ✅/❌ |

**Notes:**
```
[Any issues, UI glitches, or unexpected behavior]
```

---

#### Phase 3: TypeScript TUI Restore

| Metric | Value | Status |
|--------|-------|--------|
| Session loaded | [YES/NO] | ✅/❌ |
| Error on load | [none/describe] | - |
| Total messages | [count] | - |
| Messages from both TUIs | [YES/NO] | ✅/❌ |
| Cost matches Go | $[amount] (matched: ✓/✗) | ✅/❌ |
| Can continue | [YES/NO] | ✅/❌ |
| UI responsiveness | [good/slow/freeze] | ✅/❌ |

**Notes:**
```
[Any issues with data integrity or functionality]
```

---

## Pass/Fail Criteria

### PASS Requirements

All of the following must be true:

1. **Session Creation**: Session created in TS with valid JSON format
2. **Session Loading**: Session loads in Go without errors
3. **Data Preservation**: All messages and cost preserved during rollback
4. **Conversation Continuation**: Can add messages in Go and restore to TS
5. **Data Integrity**: All messages from both TUIs visible after restore
6. **Cost Accuracy**: Cost values preserved and correctly incremented
7. **No Errors**: No crashes, hangs, or data corruption during transitions

### FAIL Triggers

Any of the following triggers automatic rollback to Go TUI:

- Session fails to load in Go
- Message history corrupted or missing
- Cost values inconsistent
- Application crash during transitions
- Data loss during save/restore cycle

---

## Integration Test Suite

Before executing rollback rehearsal, verify all tests pass.

### Run Test Suite

```bash
cd packages/tui

# Run all tests
npm test

# Run only integration tests
npm run test:integration

# Run specific integration test
npm test -- session.test.ts

# Run with verbose output
npm test -- --reporter=verbose
```

### Expected Test Results

All of the following test suites should pass:

- [ ] `session.test.ts` - Session CRUD operations
- [ ] `e2e.test.ts` - End-to-end workflows
- [ ] `mcp.test.ts` - MCP tool integration
- [ ] `concurrent.test.ts` - Concurrent operations
- [ ] `errors.test.ts` - Error handling

### Required Metrics

After test execution, verify:

```bash
# Check test coverage (target: >80%)
npm test -- --coverage

# Verify TypeScript compilation
npm run typecheck

# Check for linting issues
npm run lint
```

**Expected output:**
- All tests passing
- Coverage >80%
- Zero TypeScript errors
- Zero linting warnings

---

## Quality Gate Verification

### TypeScript Compilation

```bash
cd packages/tui
npm run typecheck
```

Expected: Zero errors or warnings

### ESLint Compliance

```bash
npm run lint
```

Expected: No violations

### Test Coverage

```bash
npm test -- --coverage
```

Expected: >80% coverage

### Memory Profiling

```bash
npm run benchmark
```

Expected: Metrics within target ranges (see performance-report.md)

---

## Rollback Decision Tree

### If Session Won't Load in Go

**Error Type: File Not Found**
- Verify session file exists: `ls ~/.claude/sessions/<id>.json`
- Verify file is readable: `cat ~/.claude/sessions/<id>.json`
- Verify JSON is valid: `cat ~/.claude/sessions/<id>.json | jq .`
- Action: Fix file format, retry

**Error Type: Invalid Format**
- Verify all required fields present: `id, created_at, last_used, cost, tool_calls`
- Verify date format: ISO 8601 with `Z` suffix
- Verify field names are `snake_case` (not `camelCase`)
- Action: Restore from backup, re-run TS TUI, retry

**Error Type: Crash on Load**
- Check for memory issues: `free -h`
- Check system logs: `journalctl -n 50`
- Try loading older session
- Action: Investigate system resource constraints

### If Messages Missing After Rollback

- Messages should be preserved in Go session file
- If missing: Compare `created_at` timestamps
- Action: Restore from backup, investigate save logic

### If Cost Doesn't Match

- Verify cost is not being reset on load
- Check for floating-point precision issues
- Manually verify last_used timestamp was updated
- Action: Investigate cost tracking in session store

---

## Sign-Off

**Rollback Rehearsal**: PASS / FAIL

**Signed by:** ___________________________

**Date:** ___________________________

**Notes/Issues:**
```
[Any issues discovered during rehearsal]
[Recommendations for before cutover]
```

---

## Post-Test Checklist

After rollback rehearsal:

- [ ] All test results documented
- [ ] Results match pass/fail criteria
- [ ] No data loss during transitions
- [ ] No crashes or hangs observed
- [ ] Performance within acceptable range
- [ ] Handoff to cutover checklist complete
- [ ] Go TUI deprecated/ archived (if passing)

---

## Reference

- **Session Format:** See `session-persistence.md` for file format specification
- **Performance Baseline:** See `performance-report.md` for expected metrics
- **Integration Tests:** `tests/integration/session.test.ts`
- **Test Commands:** See `package.json` scripts section

---

*Generated for: TUI-022 - Integration Tests & Rollback Rehearsal*
*Migration Phase: 8 (Cutover Validation)*
