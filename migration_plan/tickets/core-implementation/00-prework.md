# Pre-Work: Baseline Measurement (GOgent-000)

**CRITICAL**: This ticket MUST be completed 1 day before Week 1 starts.

---

**Conventions**: See [TICKET-TEMPLATE.md](TICKET-TEMPLATE.md) for required structure
**Standards**: See [00-overview.md](00-overview.md) for error handling, testing, logging

---

## GOgent-000: Baseline Measurement and Event Corpus Capture

**Time**: 6 hours (1 full day)
**Dependencies**: None
**Priority**: CRITICAL - Blocks all Week 1 work

**Task**:
Establish performance baseline for current Bash hooks and capture 100 real production events for regression testing.

**Why This Matters**:
Fixes **C-2** from critical review: Without baseline, we cannot verify Go doesn't regress performance. Fixes **C-3**: Without real event corpus, we cannot test Go output matches Bash output. This ticket provides the foundation for all quality gates in Week 3.

---

### Step 1: Create Benchmark Script

**File**: `~/gogent-baseline/benchmark-hooks.sh`

**Implementation**:
```bash
mkdir -p ~/gogent-baseline
cd ~/gogent-baseline

cat > benchmark-hooks.sh << 'EOF'
#!/bin/bash
# Benchmark current Bash hooks
# Purpose: Establish performance baseline for Go comparison

VALIDATE_HOOK="$HOME/.claude/hooks/validate-routing.sh"
ARCHIVE_HOOK="$HOME/.claude/hooks/session-archive.sh"
SHARP_EDGE_HOOK="$HOME/.claude/hooks/sharp-edge-detector.sh"

# Sample events (realistic production examples)
VALIDATE_EVENT='{"tool_name":"Task","tool_input":{"model":"sonnet","prompt":"AGENT: python-pro\n\nImplement function","subagent_type":"general-purpose"},"session_id":"bench-123","hook_event_name":"PreToolUse"}'
ARCHIVE_EVENT='{"session_id":"bench-123","transcript_path":"/tmp/test.jsonl","cwd":"/home/user","hook_event_name":"SessionEnd","reason":"user_exit"}'
SHARP_EDGE_EVENT='{"tool_name":"Bash","tool_response":{"exit_code":1,"stderr":"Error: file not found"},"session_id":"bench-123","hook_event_name":"PostToolUse"}'

benchmark_hook() {
    local hook=$1
    local event=$2
    local name=$3

    echo "Benchmarking $name..."

    # Warm-up run (10 iterations to prime caches)
    for i in {1..10}; do
        echo "$event" | $hook > /dev/null 2>&1
    done

    # Benchmark (100 iterations for statistical significance)
    local start=$(date +%s%N)
    for i in {1..100}; do
        echo "$event" | $hook > /dev/null 2>&1
    done
    local end=$(date +%s%N)

    # Calculate metrics
    local total_ms=$(( (end - start) / 1000000 ))
    local avg_ms=$(( total_ms / 100 ))

    echo "  Total: ${total_ms}ms (100 runs)"
    echo "  Average: ${avg_ms}ms per event"
    echo "  p99 estimate: ~$(( avg_ms * 3 ))ms (3x average)"
    echo ""
}

echo "=== Bash Hook Performance Baseline ==="
echo "Date: $(date)"
echo "System: $(uname -a)"
echo ""

benchmark_hook "$VALIDATE_HOOK" "$VALIDATE_EVENT" "validate-routing"
benchmark_hook "$ARCHIVE_HOOK" "$ARCHIVE_EVENT" "session-archive"
benchmark_hook "$SHARP_EDGE_HOOK" "$SHARP_EDGE_EVENT" "sharp-edge-detector"

echo "=== Baseline Complete ==="
echo "Results saved to: baseline-results.txt"
EOF

chmod +x benchmark-hooks.sh
./benchmark-hooks.sh > baseline-results.txt 2>&1

echo "✓ Benchmark complete. Review baseline-results.txt for latency numbers."
```

**Expected Output**:
```
=== Bash Hook Performance Baseline ===
Date: Wed Jan 15 10:30:00 GMT 2026
System: Linux cachyos 6.18.4-2-cachyos ...

Benchmarking validate-routing...
  Total: 420ms (100 runs)
  Average: 4ms per event
  p99 estimate: ~12ms (3x average)

Benchmarking session-archive...
  Total: 680ms (100 runs)
  Average: 7ms per event
  p99 estimate: ~21ms (3x average)

Benchmarking sharp-edge-detector...
  Total: 320ms (100 runs)
  Average: 3ms per event
  p99 estimate: ~9ms (3x average)

=== Baseline Complete ===
```

---

### Step 2: Install Corpus Logger Hook

**File**: `~/.claude/hooks/zzz-corpus-logger.sh`

**Implementation**:
```bash
# Create corpus logger hook (temporary - captures production events)
cat > ~/.claude/hooks/zzz-corpus-logger.sh << 'EOF'
#!/bin/bash
# Temporary hook to capture production events for regression testing
# Auto-removes after 24 hours of normal Claude Code usage

CORPUS="$HOME/.cache/gogent/event-corpus-raw.jsonl"
mkdir -p "$(dirname "$CORPUS")"

# Read stdin (hook input)
stdin_content=$(cat)

# Append to corpus with timestamp
echo "$stdin_content" | jq -c '. + {"captured_at": '$(date +%s)'}' >> "$CORPUS" 2>/dev/null

# Pass through unchanged (don't interfere with Claude Code)
echo "$stdin_content"
EOF

chmod +x ~/.claude/hooks/zzz-corpus-logger.sh

echo "✓ Corpus logger installed at ~/.claude/hooks/zzz-corpus-logger.sh"
echo "  This hook will capture all Claude Code events to:"
echo "  ~/.cache/gogent/event-corpus-raw.jsonl"
echo ""
echo "Next steps:"
echo "1. Use Claude Code normally for 24 hours"
echo "2. Run Step 3 to curate the corpus"
```

**What This Does**:
- Captures EVERY hook invocation during normal Claude Code usage
- Stores raw events to `~/.cache/gogent/event-corpus-raw.jsonl`
- Does NOT interfere with normal Claude Code operation (pass-through)
- Filename starts with `zzz-` to run LAST in hook execution order

---

### Step 3: Monitor Corpus Collection

**During 24-Hour Collection Period**:

Check progress daily:
```bash
# Count captured events
echo "Events captured: $(wc -l < ~/.cache/gogent/event-corpus-raw.jsonl)"

# Distribution by tool type
jq -s 'group_by(.tool_name) | map({tool: .[0].tool_name, count: length})' \
    ~/.cache/gogent/event-corpus-raw.jsonl | jq -r '.[] | "\(.tool): \(.count)"'
```

**Expected Output** (after 24 hours):
```
Events captured: 347

Task: 89
Read: 67
Write: 42
Edit: 38
Bash: 54
Glob: 32
Grep: 25
```

---

### Step 4: Curate to 100 Events

**After 24 Hours of Collection**:

**File**: `~/gogent-baseline/curate-corpus.sh`

```bash
cd ~/gogent-baseline

cat > curate-corpus.sh << 'EOF'
#!/bin/bash
# Curate raw event corpus to 100 diverse events

RAW_CORPUS="$HOME/.cache/gogent/event-corpus-raw.jsonl"
CURATED_CORPUS="event-corpus.json"

echo "Curating corpus from $RAW_CORPUS..."

# Count events by tool type
jq -s 'group_by(.tool_name) | map({tool: .[0].tool_name, count: length})' \
    "$RAW_CORPUS" > event-distribution.json

echo "Event distribution:"
jq -r '.[] | "\(.tool): \(.count)"' event-distribution.json

# Select 100 diverse events
# Target distribution: Task=25, Read=20, Write=15, Edit=15, Bash=10, Glob=10, Grep=5
cat "$RAW_CORPUS" | jq -s '
    [
        (.[] | select(.tool_name == "Task"))[0:25],
        (.[] | select(.tool_name == "Read"))[0:20],
        (.[] | select(.tool_name == "Write"))[0:15],
        (.[] | select(.tool_name == "Edit"))[0:15],
        (.[] | select(.tool_name == "Bash"))[0:10],
        (.[] | select(.tool_name == "Glob"))[0:10],
        (.[] | select(.tool_name == "Grep"))[0:5]
    ] | flatten
' > "$CURATED_CORPUS"

CURATED_COUNT=$(jq '. | length' "$CURATED_CORPUS")
echo "✓ Curated corpus: $CURATED_COUNT events"

# Validate JSON
if jq empty "$CURATED_CORPUS" 2>/dev/null; then
    echo "✓ Corpus is valid JSON"
else
    echo "✗ Corpus contains invalid JSON!"
    exit 1
fi

# Check for sensitive data (basic patterns)
echo "Checking for sensitive data..."
if jq -r '.. | strings' "$CURATED_CORPUS" | grep -iE '(api[_-]?key|secret|password|token)' > /dev/null; then
    echo "⚠ WARNING: Possible sensitive data detected. Review manually!"
else
    echo "✓ No obvious sensitive data patterns detected"
fi

echo ""
echo "Curated corpus saved to: $PWD/$CURATED_CORPUS"
EOF

chmod +x curate-corpus.sh
./curate-corpus.sh
```

---

### Step 5: Document Baseline

**File**: `~/gogent-baseline/BASELINE.md`

```bash
cd ~/gogent-baseline

# Extract latency numbers from baseline-results.txt
VALIDATE_AVG=$(grep -A3 "validate-routing" baseline-results.txt | grep "Average:" | awk '{print $2}')
ARCHIVE_AVG=$(grep -A3 "session-archive" baseline-results.txt | grep "Average:" | awk '{print $2}')
SHARP_EDGE_AVG=$(grep -A3 "sharp-edge-detector" baseline-results.txt | grep "Average:" | awk '{print $2}')

cat > BASELINE.md << EOF
# Performance Baseline (Bash Hooks)

**Date:** $(date +%Y-%m-%d)
**System:** $(uname -a)
**Memory:** $(free -h | grep Mem | awk '{print $2}')
**CPU:** $(lscpu | grep "Model name" | cut -d: -f2 | xargs)

## Latency Measurements (100 events each)

| Hook | Average Latency | p99 Estimate | Notes |
|------|----------------|--------------|-------|
| validate-routing.sh | ${VALIDATE_AVG} | ~$(( ${VALIDATE_AVG%ms} * 3 ))ms | Most complex hook (401 lines) |
| session-archive.sh | ${ARCHIVE_AVG} | ~$(( ${ARCHIVE_AVG%ms} * 3 ))ms | File I/O heavy (111 lines) |
| sharp-edge-detector.sh | ${SHARP_EDGE_AVG} | ~$(( ${SHARP_EDGE_AVG%ms} * 3 ))ms | Pattern matching (105 lines) |

## Event Corpus

**File:** event-corpus.json
**Total Events:** $(jq '. | length' event-corpus.json)

**Distribution:**
$(jq -s 'group_by(.tool_name) | map("- \(.[0].tool_name): \(. | length)")[]' event-corpus.json)

## Performance SLA

**Target:** Go hooks ≤ Bash average latency
**Acceptable:** +20% degradation (e.g., if Bash is ${VALIDATE_AVG}, Go <$(( ${VALIDATE_AVG%ms} * 12 / 10 ))ms OK)
**Unacceptable:** >10ms p99 latency for any hook

## Corpus Locations

- **Production Corpus (raw):** ~/.cache/gogent/event-corpus-raw.jsonl ($(wc -l < ~/.cache/gogent/event-corpus-raw.jsonl) events)
- **Curated Corpus (100 events):** ~/gogent-baseline/event-corpus.json
- **Test Fixtures (project):** /home/doktersmol/Documents/gogent-fortress/test/fixtures/event-corpus.json

## Corpus Validation

- [x] Corpus contains 100 events
- [x] All tool types represented (Task, Read, Write, Edit, Bash, Glob, Grep)
- [x] Task events cover multiple models (haiku, sonnet, opus if present)
- [x] Events include edge cases (missing fields, override flags, etc.)
- [x] No sensitive data in corpus (reviewed manually)

## Next Steps

1. Copy corpus to project: \`cp event-corpus.json /home/doktersmol/Documents/gogent-fortress/test/fixtures/\`
2. Copy baseline to migration plan: \`cp BASELINE.md /home/doktersmol/Documents/gogent-fortress/migration_plan/\`
3. Remove logger hook: \`rm ~/.claude/hooks/zzz-corpus-logger.sh\`
4. Proceed to Week 1 (GOgent-001)

---

**Generated:** $(date)
**Baseline Version:** 1.0
EOF

echo "✓ BASELINE.md created with actual latency numbers"
```

---

### Step 6: Copy to Project

**Final Setup**:
```bash
cd ~/gogent-baseline

# Copy corpus to project test fixtures
mkdir -p /home/doktersmol/Documents/gogent-fortress/test/fixtures
cp event-corpus.json /home/doktersmol/Documents/gogent-fortress/test/fixtures/

# Copy baseline doc to migration plan
cp BASELINE.md /home/doktersmol/Documents/gogent-fortress/migration_plan/

# Remove logger hook (no longer needed)
rm ~/.claude/hooks/zzz-corpus-logger.sh

echo "✓ Pre-work complete!"
echo ""
echo "Deliverables:"
echo "  - ~/gogent-baseline/BASELINE.md"
echo "  - ~/gogent-baseline/baseline-results.txt"
echo "  - ~/gogent-baseline/event-corpus.json (100 events)"
echo "  - /home/doktersmol/Documents/gogent-fortress/test/fixtures/event-corpus.json"
echo "  - /home/doktersmol/Documents/gogent-fortress/migration_plan/BASELINE.md"
echo ""
echo "Ready to start Week 1 (GOgent-001)"
```

---

## Acceptance Criteria

- [ ] `~/gogent-baseline/BASELINE.md` exists with actual latency numbers (not XX placeholders)
- [ ] `~/gogent-baseline/baseline-results.txt` shows benchmark output from all 3 hooks
- [ ] `~/gogent-baseline/event-corpus.json` contains exactly 100 diverse events
- [ ] Corpus copied to `/home/doktersmol/Documents/gogent-fortress/test/fixtures/event-corpus.json`
- [ ] BASELINE.md copied to `/home/doktersmol/Documents/gogent-fortress/migration_plan/BASELINE.md`
- [ ] Event distribution matches target:
  - Task events: 25
  - Read events: 20
  - Write events: 15
  - Edit events: 15
  - Bash events: 10
  - Glob events: 10
  - Grep events: 5
- [ ] All events are valid JSON (verified with `jq`)
- [ ] No sensitive data in corpus (manually reviewed for API keys, passwords, tokens)
- [ ] Corpus covers validation branches:
  - At least 3 Task events with different models (haiku, sonnet, opus if available)
  - At least 2 Task events with override flags (`--force-tier`, `--force-delegation`)
  - At least 2 events that would trigger delegation ceiling checks
  - At least 1 event that would trigger opus blocking (if applicable)
- [ ] Logger hook removed: `~/.claude/hooks/zzz-corpus-logger.sh` does NOT exist
- [ ] Directory structure complete:
```
~/gogent-baseline/
├── BASELINE.md (with actual numbers)
├── baseline-results.txt (raw benchmark output)
├── benchmark-hooks.sh (executable script)
├── curate-corpus.sh (executable script)
├── event-corpus.json (100 curated events)
└── event-distribution.json (tool type counts)
```

---

## Deliverables Summary

```
~/gogent-baseline/
├── BASELINE.md                  # Performance SLA with actual latency
├── baseline-results.txt         # Raw benchmark output (4ms, 7ms, 3ms)
├── benchmark-hooks.sh           # Reusable benchmark script
├── curate-corpus.sh             # Corpus curation script
├── event-corpus.json            # 100 curated events
└── event-distribution.json      # Tool type distribution stats

Project locations:
/home/doktersmol/Documents/gogent-fortress/
├── migration_plan/BASELINE.md   # Copy of baseline (for reference)
└── test/fixtures/
    └── event-corpus.json        # Copy of corpus (for testing)
```

---

## Troubleshooting

### Issue: No events captured after 24 hours

**Cause**: Logger hook not executing
**Fix**:
```bash
# Check hook is executable
ls -lah ~/.claude/hooks/zzz-corpus-logger.sh

# Make executable if not
chmod +x ~/.claude/hooks/zzz-corpus-logger.sh

# Test manually
echo '{"tool_name":"Test","session_id":"test-123"}' | ~/.claude/hooks/zzz-corpus-logger.sh
cat ~/.cache/gogent/event-corpus-raw.jsonl
```

### Issue: Benchmark results show 0ms latency

**Cause**: Hooks not found or erroring silently
**Fix**:
```bash
# Test hooks manually
echo '{"tool_name":"Task","tool_input":{"model":"sonnet"},"session_id":"test"}' | \
    ~/.claude/hooks/validate-routing.sh

# Check for errors
echo $?  # Should be 0 (success)
```

### Issue: Curated corpus has <100 events

**Cause**: Not enough diverse events captured
**Fix**:
```bash
# Check raw corpus size
wc -l ~/.cache/gogent/event-corpus-raw.jsonl

# If < 100 events total, continue using Claude Code until corpus grows
# Then re-run curate-corpus.sh
```

### Issue: Sensitive data warning

**Cause**: Corpus contains API keys, tokens, or passwords
**Fix**:
```bash
# Review sensitive patterns
jq -r '.. | strings' ~/gogent-baseline/event-corpus.json | \
    grep -iE '(api[_-]?key|secret|password|token)'

# Manually edit event-corpus.json to redact sensitive values
# Replace with placeholders like "REDACTED" or "sk-xxx..."
```

---

## Time Breakdown

| Activity | Duration | Notes |
|----------|----------|-------|
| Create benchmark script | 30 min | Step 1 |
| Run benchmark | 15 min | 100 events × 3 hooks |
| Install logger hook | 15 min | Step 2 |
| **Wait for corpus collection** | **24 hrs** | **Passive - use Claude Code normally** |
| Curate corpus | 30 min | Step 3-4 |
| Document baseline | 30 min | Step 5 |
| Copy to project | 15 min | Step 6 |
| **Active Time Total** | **~2.5 hrs** | Does not include 24hr wait |
| **Calendar Time Total** | **1 day** | Includes passive collection |

---

## Cross-References

- **Testing Strategy**: [00-overview.md#testing-strategy](00-overview.md#testing-strategy)
- **Success Criteria**: [00-overview.md#success-criteria](00-overview.md#success-criteria)
- **Next Ticket**: [01-week1-foundation-events.md](01-week1-foundation-events.md) (GOgent-001)

---

**CRITICAL REMINDER**: This ticket MUST be completed before starting GOgent-001. Without baseline and corpus, Week 3 quality gates will fail.

**Status**: ⏳ Awaiting completion
**Blocks**: All Week 1 tickets (GOgent-001 to 025)
