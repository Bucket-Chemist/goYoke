---
id: GOgent-107
title: Performance Regression Monitoring
description: **Task**:
status: pending
time_estimate: 1h
dependencies: ["GOgent-104"]
priority: high
week: 5
tags: ["deployment", "week-5"]
tests_required: true
acceptance_criteria_count: 10
---

### GOgent-107: Performance Regression Monitoring

**Time**: 1 hour
**Dependencies**: GOgent-104 (cutover complete)

**Task**:
Create monitoring script that detects performance regressions post-cutover.

**File**: `scripts/health-check.sh`

**Implementation**:

```bash
#!/bin/bash
# Health check script - monitor hook performance
# Usage: ./scripts/health-check.sh

set -e

HOOKS_DIR="${HOME}/.claude/hooks"
GOgent_LOG="${HOME}/.gogent/hooks.log"
VIOLATIONS_LOG="${HOME}/.gogent/routing-violations.jsonl"
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo "=========================================="
echo "GOgent Fortress Health Check"
echo "=========================================="
echo ""

# Check 1: Hooks exist and are executable
echo "[CHECK] Hook binaries..."

HOOKS=(
    "validate-routing"
    "session-archive"
    "sharp-edge-detector"
)

HOOKS_OK=true

for hook in "${HOOKS[@]}"; do
    if [ ! -x "$HOOKS_DIR/$hook" ]; then
        echo -e "  ${RED}✗${NC} $hook not found or not executable"
        HOOKS_OK=false
    else
        echo -e "  ${GREEN}✓${NC} $hook exists and executable"
    fi
done

# Check 2: Hook performance (basic smoke test)
echo ""
echo "[CHECK] Hook performance..."

TEST_EVENT='{"hook_event_name":"PreToolUse","tool_name":"Read"}'

for hook in "${HOOKS[@]}"; do
    START=$(date +%s%N)

    if echo "$TEST_EVENT" | "$HOOKS_DIR/$hook" > /dev/null 2>&1; then
        END=$(date +%s%N)
        DURATION_MS=$(( (END - START) / 1000000 ))

        if [ $DURATION_MS -lt 10 ]; then
            echo -e "  ${GREEN}✓${NC} $hook responds in ${DURATION_MS}ms"
        elif [ $DURATION_MS -lt 50 ]; then
            echo -e "  ${YELLOW}⚠${NC} $hook responds in ${DURATION_MS}ms (slower than expected)"
        else
            echo -e "  ${RED}✗${NC} $hook responds in ${DURATION_MS}ms (too slow)"
        fi
    else
        echo -e "  ${RED}✗${NC} $hook failed smoke test"
    fi
done

# Check 3: Recent errors in logs
echo ""
echo "[CHECK] Recent errors..."

if [ -f "$GOgent_LOG" ]; then
    ERROR_COUNT=$(grep -c "ERROR" "$GOgent_LOG" 2>/dev/null || echo "0")

    if [ "$ERROR_COUNT" -eq 0 ]; then
        echo -e "  ${GREEN}✓${NC} No errors in hooks.log"
    elif [ "$ERROR_COUNT" -lt 5 ]; then
        echo -e "  ${YELLOW}⚠${NC} $ERROR_COUNT errors in hooks.log (review recommended)"
    else
        echo -e "  ${RED}✗${NC} $ERROR_COUNT errors in hooks.log (investigate immediately)"
    fi
else
    echo -e "  ${YELLOW}⚠${NC} hooks.log not found"
fi

# Check 4: Violation rate
echo ""
echo "[CHECK] Routing violations..."

if [ -f "$VIOLATIONS_LOG" ]; then
    VIOLATION_COUNT=$(wc -l < "$VIOLATIONS_LOG" 2>/dev/null || echo "0")

    # Check violations in last hour
    if command -v jq &> /dev/null; then
        ONE_HOUR_AGO=$(date -d '1 hour ago' +%s 2>/dev/null || date -v -1H +%s)
        RECENT_VIOLATIONS=$(jq -r "select(.timestamp > \"$(date -d @$ONE_HOUR_AGO -Iseconds)\") | .timestamp" "$VIOLATIONS_LOG" 2>/dev/null | wc -l)

        if [ "$RECENT_VIOLATIONS" -eq 0 ]; then
            echo -e "  ${GREEN}✓${NC} No violations in last hour"
        elif [ "$RECENT_VIOLATIONS" -lt 10 ]; then
            echo -e "  ${YELLOW}⚠${NC} $RECENT_VIOLATIONS violations in last hour"
        else
            echo -e "  ${RED}✗${NC} $RECENT_VIOLATIONS violations in last hour (high rate)"
        fi
    else
        echo -e "  ${YELLOW}⚠${NC} jq not installed, cannot analyze violation rate"
    fi

    echo "  Total violations logged: $VIOLATION_COUNT"
else
    echo -e "  ${GREEN}✓${NC} No violations log (clean)"
fi

# Check 5: Disk space
echo ""
echo "[CHECK] Disk space..."

GOgent_DIR="${HOME}/.gogent"

if [ -d "$GOgent_DIR" ]; then
    DISK_USAGE=$(du -sh "$GOgent_DIR" | awk '{print $1}')
    echo "  GOgent directory size: $DISK_USAGE"

    # Check if >100MB
    DISK_USAGE_KB=$(du -sk "$GOgent_DIR" | awk '{print $1}')

    if [ "$DISK_USAGE_KB" -gt 102400 ]; then
        echo -e "  ${YELLOW}⚠${NC} Directory exceeds 100MB (consider log rotation)"
    else
        echo -e "  ${GREEN}✓${NC} Disk usage normal"
    fi
fi

# Check 6: Log file sizes
echo ""
echo "[CHECK] Log file sizes..."

LOG_FILES=(
    "$GOgent_LOG"
    "$VIOLATIONS_LOG"
    "${HOME}/.gogent/error-patterns.jsonl"
)

for log in "${LOG_FILES[@]}"; do
    if [ -f "$log" ]; then
        SIZE=$(du -h "$log" | awk '{print $1}')
        LINE_COUNT=$(wc -l < "$log")

        echo "  $(basename $log): $SIZE ($LINE_COUNT lines)"

        # Warn if >10MB
        SIZE_KB=$(du -k "$log" | awk '{print $1}')
        if [ "$SIZE_KB" -gt 10240 ]; then
            echo -e "    ${YELLOW}⚠${NC} Large log file (consider truncating)"
        fi
    fi
done

# Summary
echo ""
echo "=========================================="

if [ "$HOOKS_OK" = true ] && [ "$ERROR_COUNT" -lt 5 ]; then
    echo -e "${GREEN}System Health: GOOD${NC}"
    echo "No immediate action required."
else
    echo -e "${YELLOW}System Health: NEEDS ATTENTION${NC}"
    echo "Review warnings above and investigate issues."
fi

echo "=========================================="
echo ""
echo "For detailed logs: tail -f $GOgent_LOG"
echo "For violations: cat $VIOLATIONS_LOG"
````

**Cron Job Setup**:

```bash
# Add to crontab for hourly checks
# crontab -e

# Run health check every hour, log to file
0 * * * * /path/to/gogent/scripts/health-check.sh >> ~/gogent-health-$(date +\%Y\%m\%d).log 2>&1
```

**Acceptance Criteria**:

- [ ] Script checks hook binaries exist and are executable
- [ ] Smoke tests each hook with sample event
- [ ] Reports response times (<10ms good, 10-50ms warning, >50ms error)
- [ ] Counts recent errors in hooks.log
- [ ] Analyzes violation rate (last hour)
- [ ] Checks disk space usage
- [ ] Reports log file sizes
- [ ] Overall health status (GOOD / NEEDS ATTENTION)
- [ ] Can be run manually or via cron
- [ ] Output is human-readable with color coding

**Why This Matters**: Post-cutover monitoring catches performance regressions early. Regular health checks ensure system stays healthy long-term.

---
