---
id: GOgent-102
title: Parallel Testing Script
description: **Task**:
status: pending
time_estimate: 2h
dependencies: ["GOgent-101"]
priority: high
week: 5
tags: ["testing", "week-5"]
tests_required: true
acceptance_criteria_count: 11
---

### GOgent-102: Parallel Testing Script

**Time**: 2 hours
**Dependencies**: GOgent-101

**Task**:
Run Go and Bash hooks in parallel for 24 hours, comparing outputs and monitoring for issues.

**File**: `scripts/parallel-test.sh`

**Implementation**:

```bash
#!/bin/bash
# Parallel testing script - runs Go and Bash hooks side-by-side
# Usage: ./scripts/parallel-test.sh [--duration HOURS]

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HOOKS_DIR="${HOME}/.claude/hooks"
GOgent_BIN="${HOME}/.gogent/bin"
TEST_LOG_DIR="${HOME}/.gogent/parallel-test-$(date +%Y%m%d-%H%M%S)"

mkdir -p "$TEST_LOG_DIR"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1" | tee -a "$TEST_LOG_DIR/parallel-test.log"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" | tee -a "$TEST_LOG_DIR/parallel-test.log"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$TEST_LOG_DIR/parallel-test.log"
}

# Parse duration argument (default 24 hours)
DURATION_HOURS=24

for arg in "$@"; do
    case $arg in
        --duration)
            shift
            DURATION_HOURS="$1"
            shift
            ;;
    esac
done

log_info "Starting parallel testing for $DURATION_HOURS hours"
log_info "Test logs: $TEST_LOG_DIR"

# Verify binaries exist
if [ ! -x "$GOgent_BIN/gogent-validate" ]; then
    log_error "Go binary not found. Run ./scripts/install.sh first"
    exit 1
fi

if [ ! -f "$HOOKS_DIR/validate-routing" ]; then
    log_error "Bash hook not found: $HOOKS_DIR/validate-routing"
    exit 1
fi

# Create test corpus (capture real events)
CORPUS_FILE="$TEST_LOG_DIR/parallel-test-corpus.jsonl"

log_info "Capturing live events from Claude Code sessions..."
log_info "Events will be logged to: $CORPUS_FILE"

# Hook into Claude Code event stream (if available)
# For testing, we can monitor the session transcripts
TRANSCRIPT_DIR="${HOME}/.claude/projects"

# Start monitoring loop
START_TIME=$(date +%s)
END_TIME=$((START_TIME + DURATION_HOURS * 3600))

EVENT_COUNT=0
DIFF_COUNT=0

log_info "Monitoring events until: $(date -d @$END_TIME)"

while [ $(date +%s) -lt $END_TIME ]; do
    # Find recent transcript files (last 5 minutes)
    RECENT_TRANSCRIPTS=$(find "$TRANSCRIPT_DIR" -name "*.jsonl" -mmin -5 2>/dev/null || true)

    if [ -n "$RECENT_TRANSCRIPTS" ]; then
        # Extract events from recent transcripts
        for transcript in $RECENT_TRANSCRIPTS; do
            # Read events from transcript (last 10 lines to avoid re-processing)
            tail -n 10 "$transcript" 2>/dev/null | while read -r line; do
                # Skip empty lines
                [ -z "$line" ] && continue

                # Parse event
                HOOK_EVENT=$(echo "$line" | jq -r '.hook_event_name // empty' 2>/dev/null || true)

                if [ -n "$HOOK_EVENT" ]; then
                    # Run through both Go and Bash implementations
                    GO_OUTPUT=$(echo "$line" | "$GOgent_BIN/gogent-validate" 2>&1 || true)
                    BASH_OUTPUT=$(echo "$line" | "$HOOKS_DIR/validate-routing" 2>&1 || true)

                    # Compare outputs
                    if [ "$GO_OUTPUT" != "$BASH_OUTPUT" ]; then
                        DIFF_COUNT=$((DIFF_COUNT + 1))

                        log_warn "Output difference detected!"
                        echo "Event: $line" >> "$TEST_LOG_DIR/differences.log"
                        echo "Go:   $GO_OUTPUT" >> "$TEST_LOG_DIR/differences.log"
                        echo "Bash: $BASH_OUTPUT" >> "$TEST_LOG_DIR/differences.log"
                        echo "---" >> "$TEST_LOG_DIR/differences.log"
                    fi

                    EVENT_COUNT=$((EVENT_COUNT + 1))

                    # Log event to corpus
                    echo "$line" >> "$CORPUS_FILE"
                fi
            done
        done
    fi

    # Report progress every hour
    CURRENT_TIME=$(date +%s)
    ELAPSED_HOURS=$(( (CURRENT_TIME - START_TIME) / 3600 ))

    if [ $((ELAPSED_HOURS % 1)) -eq 0 ] && [ $ELAPSED_HOURS -gt 0 ]; then
        log_info "Progress: $ELAPSED_HOURS/$DURATION_HOURS hours, $EVENT_COUNT events, $DIFF_COUNT differences"
    fi

    # Sleep for 60 seconds before next check
    sleep 60
done

# Generate report
log_info "Parallel testing complete"
log_info "Generating report..."

cat > "$TEST_LOG_DIR/report.md" <<EOF
# Parallel Testing Report

**Duration**: $DURATION_HOURS hours
**Start**: $(date -d @$START_TIME)
**End**: $(date -d @$END_TIME)

## Summary

- **Total Events**: $EVENT_COUNT
- **Differences Detected**: $DIFF_COUNT
- **Match Rate**: $(awk "BEGIN {printf \"%.2f\", (($EVENT_COUNT - $DIFF_COUNT) / $EVENT_COUNT) * 100}")%

## Status

EOF

if [ $DIFF_COUNT -eq 0 ]; then
    echo "✅ **PASS** - All events matched between Go and Bash implementations" >> "$TEST_LOG_DIR/report.md"
    log_info "✅ All events matched!"
elif [ $DIFF_COUNT -lt $((EVENT_COUNT / 100)) ]; then
    echo "⚠️ **CAUTION** - <1% difference rate. Review differences before cutover." >> "$TEST_LOG_DIR/report.md"
    log_warn "⚠️ Minor differences detected (<1%)"
else
    echo "❌ **FAIL** - Significant differences detected. Do not proceed with cutover." >> "$TEST_LOG_DIR/report.md"
    log_error "❌ Significant differences detected!"
fi

cat >> "$TEST_LOG_DIR/report.md" <<EOF

## Files

- Corpus: $CORPUS_FILE
- Differences: $TEST_LOG_DIR/differences.log
- Full Log: $TEST_LOG_DIR/parallel-test.log

## Next Steps

EOF

if [ $DIFF_COUNT -eq 0 ]; then
    echo "1. Review report: $TEST_LOG_DIR/report.md" >> "$TEST_LOG_DIR/report.md"
    echo "2. Proceed with cutover: ./scripts/cutover.sh" >> "$TEST_LOG_DIR/report.md"
else
    echo "1. Review differences: $TEST_LOG_DIR/differences.log" >> "$TEST_LOG_DIR/report.md"
    echo "2. Fix issues in Go implementation" >> "$TEST_LOG_DIR/report.md"
    echo "3. Re-run parallel testing" >> "$TEST_LOG_DIR/report.md"
fi

log_info "Report saved: $TEST_LOG_DIR/report.md"
log_info ""
cat "$TEST_LOG_DIR/report.md"
```

**Acceptance Criteria**:

- [ ] Script accepts `--duration HOURS` parameter (default 24)
- [ ] Monitors Claude Code session transcripts for events
- [ ] Runs each event through both Go and Bash hooks
- [ ] Compares outputs and logs differences
- [ ] Saves all events to corpus file
- [ ] Reports progress every hour
- [ ] Generates final report with statistics
- [ ] Report includes match rate percentage
- [ ] Differences logged to separate file for review
- [ ] Clear PASS/CAUTION/FAIL status based on difference rate
- [ ] Instructions for next steps based on results

**Why This Matters**: 24-hour parallel testing catches edge cases that unit/integration tests miss. Ensures Go hooks behave identically to Bash in production.

---
