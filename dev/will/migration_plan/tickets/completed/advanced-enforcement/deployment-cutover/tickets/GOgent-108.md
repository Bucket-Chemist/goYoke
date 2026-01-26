---
id: GOgent-108
title: Post-Cutover Validation Checklist
description: **Task**:
status: pending
time_estimate: 1h
dependencies: ["GOgent-104"]
priority: high
week: 5
tags: ["deployment", "week-5"]
tests_required: true
acceptance_criteria_count: 25
---

### GOgent-108: Post-Cutover Validation Checklist

**Time**: 1 hour
**Dependencies**: GOgent-104 (cutover complete), 48 hours elapsed

**Task**:
Create validation checklist and script to verify cutover success after 48-hour monitoring period.

**File**: `scripts/validate-cutover.sh`

**Implementation**:

```bash
#!/bin/bash
# Post-cutover validation script
# Usage: ./scripts/validate-cutover.sh

set -e

HOOKS_DIR="${HOME}/.claude/hooks"
GOgent_DIR="${HOME}/.gogent"
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo "=========================================="
echo "Post-Cutover Validation"
echo "=========================================="
echo ""

VALIDATION_PASSED=true

# Check 1: Verify hooks are Go binaries
echo "[1/10] Verifying hooks are Go binaries..."

HOOKS=(
    "validate-routing:gogent-validate"
    "session-archive:gogent-archive"
    "sharp-edge-detector:gogent-sharp-edge"
)

for hook in "${HOOKS[@]}"; do
    HOOK_NAME="${hook%%:*}"
    BINARY_NAME="${hook##*:}"

    if [ ! -L "$HOOKS_DIR/$HOOK_NAME" ]; then
        echo -e "  ${RED}✗${NC} $HOOK_NAME is not a symlink"
        VALIDATION_PASSED=false
    else
        TARGET=$(readlink "$HOOKS_DIR/$HOOK_NAME")
        if echo "$TARGET" | grep -q "$BINARY_NAME"; then
            echo -e "  ${GREEN}✓${NC} $HOOK_NAME → Go binary"
        else
            echo -e "  ${RED}✗${NC} $HOOK_NAME points to: $TARGET"
            VALIDATION_PASSED=false
        fi
    fi
done

# Check 2: No critical errors in logs (last 48 hours)
echo ""
echo "[2/10] Checking for critical errors..."

CRITICAL_ERRORS=0

if [ -f "$GOgent_DIR/hooks.log" ]; then
    # Look for panic, fatal, critical keywords
    CRITICAL_ERRORS=$(grep -iE "panic|fatal|critical" "$GOgent_DIR/hooks.log" | wc -l)

    if [ "$CRITICAL_ERRORS" -eq 0 ]; then
        echo -e "  ${GREEN}✓${NC} No critical errors found"
    else
        echo -e "  ${RED}✗${NC} $CRITICAL_ERRORS critical errors found"
        echo "  Review: grep -iE 'panic|fatal|critical' $GOgent_DIR/hooks.log"
        VALIDATION_PASSED=false
    fi
else
    echo -e "  ${YELLOW}⚠${NC} Hooks log not found"
fi

# Check 3: Handoffs generated
echo ""
echo "[3/10] Checking session handoffs..."

HANDOFF_FILE="${HOME}/.claude/memory/last-handoff.md"

if [ -f "$HANDOFF_FILE" ]; then
    HANDOFF_AGE_HOURS=$(( ($(date +%s) - $(stat -c %Y "$HANDOFF_FILE" 2>/dev/null || stat -f %m "$HANDOFF_FILE")) / 3600 ))

    if [ "$HANDOFF_AGE_HOURS" -lt 72 ]; then
        echo -e "  ${GREEN}✓${NC} Recent handoff generated ($HANDOFF_AGE_HOURS hours ago)"
    else
        echo -e "  ${YELLOW}⚠${NC} Last handoff is $HANDOFF_AGE_HOURS hours old"
    fi
else
    echo -e "  ${YELLOW}⚠${NC} No handoff file found (no sessions ended?)"
fi

# Check 4: Violations logged correctly
echo ""
echo "[4/10] Checking violation logging..."

VIOLATIONS_LOG="$GOgent_DIR/routing-violations.jsonl"

if [ -f "$VIOLATIONS_LOG" ]; then
    # Verify JSONL format
    if jq . "$VIOLATIONS_LOG" >/dev/null 2>&1; then
        VIOLATION_COUNT=$(wc -l < "$VIOLATIONS_LOG")
        echo -e "  ${GREEN}✓${NC} Violations logged correctly ($VIOLATION_COUNT entries)"
    else
        echo -e "  ${RED}✗${NC} Violations log has invalid JSON"
        VALIDATION_PASSED=false
    fi
else
    echo -e "  ${GREEN}✓${NC} No violations (or no violations log)"
fi

# Check 5: Performance acceptable
echo ""
echo "[5/10] Checking performance..."

TEST_EVENT='{"hook_event_name":"PreToolUse","tool_name":"Read"}'

for hook in validate-routing session-archive sharp-edge-detector; do
    START=$(date +%s%N)
    echo "$TEST_EVENT" | "$HOOKS_DIR/$hook" > /dev/null 2>&1
    END=$(date +%s%N)

    DURATION_MS=$(( (END - START) / 1000000 ))

    if [ "$DURATION_MS" -lt 10 ]; then
        echo -e "  ${GREEN}✓${NC} $hook: ${DURATION_MS}ms"
    elif [ "$DURATION_MS" -lt 50 ]; then
        echo -e "  ${YELLOW}⚠${NC} $hook: ${DURATION_MS}ms (slower than baseline)"
    else
        echo -e "  ${RED}✗${NC} $hook: ${DURATION_MS}ms (too slow)"
        VALIDATION_PASSED=false
    fi
done

# Check 6: No user-reported issues
echo ""
echo "[6/10] Checking for user reports..."
echo "  Manual check required: Have users reported issues?"
echo "  Review issue tracker, emails, chat"

# Check 7: Sharp edges captured
echo ""
echo "[7/10] Checking sharp edge capture..."

LEARNINGS_FILE="${HOME}/.claude/memory/pending-learnings.jsonl"

if [ -f "$LEARNINGS_FILE" ]; then
    if jq . "$LEARNINGS_FILE" >/dev/null 2>&1; then
        EDGE_COUNT=$(wc -l < "$LEARNINGS_FILE")
        echo -e "  ${GREEN}✓${NC} Sharp edges captured ($EDGE_COUNT entries)"
    else
        echo -e "  ${RED}✗${NC} Pending learnings has invalid JSON"
        VALIDATION_PASSED=false
    fi
else
    echo -e "  ${GREEN}✓${NC} No sharp edges detected (or no learnings file)"
fi

# Check 8: Rollback still available
echo ""
echo "[8/10] Verifying rollback available..."

BACKUP_DIRS=$(find "$HOOKS_DIR" -maxdepth 1 -name "backup-*" -type d 2>/dev/null | wc -l)

if [ "$BACKUP_DIRS" -gt 0 ]; then
    echo -e "  ${GREEN}✓${NC} $BACKUP_DIRS backup(s) available"
else
    echo -e "  ${RED}✗${NC} No backups found"
    VALIDATION_PASSED=false
fi

# Check 9: Memory usage acceptable
echo ""
echo "[9/10] Checking memory usage..."

for hook in gogent-validate gogent-archive gogent-sharp-edge; do
    if pgrep -f "$hook" > /dev/null; then
        MEM_KB=$(ps aux | grep "$hook" | grep -v grep | awk '{sum+=$6} END {print sum}')
        MEM_MB=$(( MEM_KB / 1024 ))

        if [ "$MEM_MB" -lt 10 ]; then
            echo -e "  ${GREEN}✓${NC} $hook: ${MEM_MB}MB"
        else
            echo -e "  ${YELLOW}⚠${NC} $hook: ${MEM_MB}MB (higher than expected)"
        fi
    else
        echo "  ℹ️  $hook: not currently running"
    fi
done

# Check 10: System stable
echo ""
echo "[10/10] Overall system stability..."

# Calculate uptime since cutover (approximate)
UPTIME_HOURS=$(( ($(date +%s) - $(stat -c %Y "$HOOKS_DIR/validate-routing" 2>/dev/null || stat -f %m "$HOOKS_DIR/validate-routing")) / 3600 ))

if [ "$UPTIME_HOURS" -lt 48 ]; then
    echo -e "  ${YELLOW}⚠${NC} System running for only $UPTIME_HOURS hours (recommend 48+ hours)"
else
    echo -e "  ${GREEN}✓${NC} System stable for $UPTIME_HOURS hours"
fi

# Final verdict
echo ""
echo "=========================================="

if [ "$VALIDATION_PASSED" = true ]; then
    echo -e "${GREEN}✅ CUTOVER SUCCESSFUL${NC}"
    echo ""
    echo "Go implementation is working correctly."
    echo "Cutover can be declared complete."
    echo ""
    echo "Next steps:"
    echo "  1. Document lessons learned"
    echo "  2. Remove old backups (after 1 week)"
    echo "  3. Update team documentation"
else
    echo -e "${RED}❌ VALIDATION FAILED${NC}"
    echo ""
    echo "Issues detected. Review failures above."
    echo ""
    echo "Options:"
    echo "  1. Investigate and fix issues"
    echo "  2. Rollback if issues are critical: ./scripts/rollback.sh"
fi

echo "=========================================="

exit $([ "$VALIDATION_PASSED" = true ] && echo 0 || echo 1)
```

**Checklist Document**: `docs/post-cutover-checklist.md`

```markdown
# Post-Cutover Validation Checklist

**Purpose**: Verify cutover success after 48-hour monitoring period.

---

## Automated Checks (via script)

Run: `./scripts/validate-cutover.sh`

- [ ] Hooks are Go binaries (symlinks point to ~/.gogent/bin/)
- [ ] No critical errors in logs (no panic/fatal/critical)
- [ ] Session handoffs generated recently
- [ ] Violations logged correctly (valid JSONL)
- [ ] Performance acceptable (<10ms per hook)
- [ ] Sharp edges captured (valid JSONL)
- [ ] Rollback backups available
- [ ] Memory usage <10MB per hook
- [ ] System stable for 48+ hours

## Manual Checks

- [ ] No user-reported issues (check issue tracker, emails, chat)
- [ ] No "hook timed out" errors in Claude Code sessions
- [ ] Routing logic behaves correctly (spot check violations log)
- [ ] Session continuity works (handoffs preserve context)
- [ ] Sharp edge blocking works (3 failures → block)

## Performance Comparison

Compare to baseline from GOgent-000:

- [ ] validate-routing latency: \_**\_ms (baseline: \_\_**ms)
- [ ] session-archive latency: \_**\_ms (baseline: \_\_**ms)
- [ ] sharp-edge latency: \_**\_ms (baseline: \_\_**ms)

## Success Criteria

All of the following MUST be true:

1. ✅ Automated validation script passes
2. ✅ No critical user-reported issues
3. ✅ Performance meets or exceeds baseline
4. ✅ System stable for 48+ hours

## If Validation Fails

1. **Minor Issues**: Fix and re-validate in 24 hours
2. **Major Issues**: Rollback immediately: `./scripts/rollback.sh`

## Declaration

**Cutover Status**: [ ] SUCCESS / [ ] ROLLBACK REQUIRED

**Date**: ******\_\_\_\_******
**Validated By**: ******\_\_\_\_******
**Notes**:

---

**Next Steps After Success**:

1. Document lessons learned (use template in cutover-decision.md)
2. Schedule backup cleanup (1 week retention)
3. Update team documentation
4. Close cutover tracking ticket
```

**Acceptance Criteria**:

- [ ] Validation script checks all critical success factors
- [ ] Script returns exit code 0 if passing, 1 if failing
- [ ] Checklist document covers automated and manual checks
- [ ] Performance comparison to baseline included
- [ ] Clear SUCCESS / ROLLBACK REQUIRED decision output
- [ ] Script can run unattended (for automation)
- [ ] Human-readable output with color coding
- [ ] Manual checklist prompts for user feedback verification

**Why This Matters**: Formal validation after 48 hours ensures cutover was truly successful. Checklist prevents premature success declaration.

---

## Summary

Week 3 Part 2 completes Phase 0 with deployment and cutover workflow:

- **GOgent-101**: Automated installation script
- **GOgent-101b**: WSL2 compatibility testing (M-8)
- **GOgent-102**: 24-hour parallel testing
- **GOgent-103**: Cutover decision workflow with GO/NO-GO criteria
- **GOgent-104**: Atomic symlink cutover script
- **GOgent-105**: Rollback script (<5min) with testing
- **GOgent-106**: Documentation updates (README, migration notes, troubleshooting)
- **GOgent-107**: Performance regression monitoring (health check)
- **GOgent-108**: Post-cutover validation after 48 hours

**Deployment Safety**:

- Automated scripts reduce human error
- Parallel testing catches edge cases before cutover
- Rollback tested and ready (<5 min recovery)
- Clear GO/NO-GO criteria prevent premature cutover
- 48-hour validation ensures stability

**Success Metrics**:

- Zero critical errors post-cutover
- Performance meets baseline (<5ms p99)
- 99%+ match rate in parallel testing
- User-transparent migration (no behavior changes)

---

**File Status**: ✅ Complete
**Tickets**: 8 (GOgent-101, 048b, 049-055)
**Detail Level**: Full implementation + complete cutover workflow
**Phase 0 Status**: ✅ ALL TICKETS COMPLETE (GOgent-000 through GOgent-108)
