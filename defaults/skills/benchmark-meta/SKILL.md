---
name: benchmark-meta
description: Analyze benchmark trends across multiple commits. Weekly/monthly cadence for system optimization.
triggers:
  - /benchmark-meta
  - analyze benchmarks
  - benchmark trends
  - cross-commit analysis
---

# Benchmark Meta-Analysis Skill

## Overview
Analyzes benchmark results across multiple commits/runs to identify:
- Cost trends (improving or degrading)
- Routing accuracy changes
- Impact of specific commits on performance
- Anti-patterns (changes that hurt metrics)
- Improvement opportunities

**Cost Warning:** This skill uses Opus for deep synthesis. Run sparingly (weekly/monthly).

## Usage
```
/benchmark-meta                    # Analyze last 5 runs
/benchmark-meta --commits=10       # Analyze last 10 runs
/benchmark-meta --since=2026-01-01 # Analyze runs since date
/benchmark-meta --compare a]b      # Compare two specific runs
```

---

## Phase 1: Gather Historical Data

**Step 1: Locate benchmark runs**
```bash
RUNS_DIR="$HOME/.claude/benchmarks/runs"
COMMITS=${COMMITS:-5}

# Get most recent N runs
RUNS=$(ls -1t "$RUNS_DIR" | head -$COMMITS)

if [[ -z "$RUNS" ]]; then
    echo "ERROR: No benchmark runs found in $RUNS_DIR"
    exit 1
fi

echo "Analyzing $(echo "$RUNS" | wc -l) benchmark runs..."
```

**Step 2: Extract summary data**
```bash
# Collect all summaries
SUMMARIES=""
for run in $RUNS; do
    RUN_PATH="$RUNS_DIR/$run"
    if [[ -f "$RUN_PATH/summary.json" ]]; then
        SUMMARIES="$SUMMARIES\n=== $run ===\n$(cat $RUN_PATH/summary.json)"
    fi
done
```

**Step 3: Collect git context**
```bash
# Get git diffs between benchmark commits
COMMITS_LIST=""
for run in $RUNS; do
    COMMIT=$(cat "$RUNS_DIR/$run/commit.txt" 2>/dev/null || echo "unknown")
    COMMITS_LIST="$COMMITS_LIST $COMMIT"
done

# Generate inter-commit diffs
DIFFS=""
prev_commit=""
for commit in $COMMITS_LIST; do
    if [[ -n "$prev_commit" && "$commit" != "unknown" && "$prev_commit" != "unknown" ]]; then
        DIFF=$(git diff --stat "$prev_commit" "$commit" 2>/dev/null || echo "diff unavailable")
        DIFFS="$DIFFS\n=== $prev_commit → $commit ===\n$DIFF"
    fi
    prev_commit=$commit
done
```

**Step 4: Collect memory artifacts (optional)**
```bash
# Include relevant memory if available
MEMORY_CONTEXT=""
if [[ -d "$HOME/.claude/memory" ]]; then
    # Recent decisions
    if [[ -f "$HOME/.claude/memory/decisions/session-decisions.jsonl" ]]; then
        MEMORY_CONTEXT="$MEMORY_CONTEXT\n=== Recent Decisions ===\n$(tail -20 $HOME/.claude/memory/decisions/session-decisions.jsonl)"
    fi
    # Sharp edges
    for f in "$HOME/.claude/memory/sharp-edges"/*.md; do
        [[ -f "$f" ]] && MEMORY_CONTEXT="$MEMORY_CONTEXT\n=== Sharp Edge: $(basename $f) ===\n$(head -30 $f)"
    done
fi
```

---

## Phase 2: Opus Deep Analysis

**This is expensive. Reserve for weekly/monthly reviews.**

```javascript
Task({
  description: "Cross-commit benchmark meta-analysis",
  subagent_type: "Explore",
  model: "opus",
  prompt: `AGENT: einstein

You are a Senior Systems Architect reviewing benchmark trends across multiple runs.

## INPUT DATA

### Benchmark Summaries (chronological)
${SUMMARIES}

### Git Diffs Between Runs
${DIFFS}

### Memory Artifacts (if available)
${MEMORY_CONTEXT}

### Current Routing Schema
$(cat ~/.claude/routing-schema.json)

## ANALYSIS REQUIRED

### 1. Trend Analysis
- **Cost Trend**: Is total cost per benchmark improving, degrading, or stable?
- **Routing Accuracy Trend**: Are correct tier selections increasing?
- **Error Rate Trend**: Are failures decreasing?
- Quantify with specific numbers and % changes.

### 2. Change Impact Assessment
For each commit transition:
| From → To | Files Changed | Cost Impact | Routing Impact | Verdict |
|-----------|---------------|-------------|----------------|---------|

Classify each change as:
- ✅ **Beneficial**: Improved metrics
- ⚠️ **Neutral**: No significant change
- ❌ **Harmful**: Degraded metrics

### 3. Anti-System Modifications
Identify changes that:
- Increased cost without improving quality
- Broke routing patterns that were working
- Added complexity without benefit
- Caused regression in attention retention

For each, recommend: REVERT, FIX, or INVESTIGATE

### 4. Improvement Opportunities
Based on patterns across all runs:
- Which prompts consistently underperform?
- Which agents are over/under-utilized?
- Where is tier leakage most common?
- What routing rules could be tightened?

Provide SPECIFIC, ACTIONABLE recommendations with file paths.

### 5. Benchmark Suite Gaps
What behaviors observed in traces aren't covered by current gold standard prompts?
Suggest 3-5 new prompts to add to suite.yaml.

### 6. System Health Score
Rate the system 1-10 on:
- Cost Efficiency: __/10
- Routing Precision: __/10  
- Attention Stability: __/10
- Error Recovery: __/10
- Overall: __/10

## OUTPUT FORMAT

# Benchmark Meta-Analysis Report

**Analysis Period:** [first run] → [last run]
**Commits Analyzed:** [count]
**Generated:** [timestamp]

## Executive Summary
[2-3 sentence overall assessment]

## Trend Charts (ASCII)
[Show cost and accuracy trends if possible]

## Change Impact Matrix
[Table from section 2]

## Anti-Patterns Detected
[List with severity and recommendations]

## Improvement Roadmap
[Prioritized list: HIGH/MED/LOW with specific actions]

## Benchmark Suite Recommendations
[New prompts to add]

## System Health Scorecard
[Scores with brief justification]

## Appendix: Raw Metrics
[Key numbers for reference]`
})
```

---

## Phase 3: Generate Report

**Output file:** `__benchmark_meta_[date].md`

Save the Opus analysis output to:
```bash
OUTPUT_FILE="$HOME/.claude/benchmarks/__benchmark_meta_$(date +%Y%m%d).md"
# Save Opus output to this file
```

---

## Phase 4: Archive & Notify

```bash
# Archive the meta-analysis
cp "$OUTPUT_FILE" "$RUNS_DIR/meta-analyses/" 2>/dev/null || mkdir -p "$RUNS_DIR/meta-analyses" && cp "$OUTPUT_FILE" "$RUNS_DIR/meta-analyses/"

# Output location
echo "Meta-analysis complete: $OUTPUT_FILE"
echo ""
echo "Key actions from this analysis:"
# Extract HIGH priority items
grep -A2 "HIGH" "$OUTPUT_FILE" | head -10
```

---

## Scheduling Recommendation

| Cadence | Trigger | Analysis Depth |
|---------|---------|----------------|
| After each benchmark | Automatic | Single-run Gemini audit only |
| Weekly | Manual `/benchmark-meta` | Last 5 runs, Opus synthesis |
| Monthly | Manual `/benchmark-meta --commits=20` | Full month, comprehensive |
| After major refactor | Manual | Compare before/after specifically |

---

## Cost Estimation

| Component | Estimated Cost |
|-----------|----------------|
| Data gathering | ~$0.00 (local) |
| Opus analysis | ~$0.50-1.50 |
| Total per meta-analysis | ~$0.50-1.50 |

Run sparingly to avoid cost accumulation.

---

## Quick Reference

| Command | Scope | Model | Est. Cost |
|---------|-------|-------|-----------|
| `/benchmark-meta` | Last 5 runs | Opus | ~$0.75 |
| `/benchmark-meta --commits=10` | Last 10 runs | Opus | ~$1.00 |
| `/benchmark-meta --compare X Y` | 2 runs | Sonnet | ~$0.15 |
