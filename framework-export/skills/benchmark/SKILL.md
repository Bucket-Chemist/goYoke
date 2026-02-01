---
name: benchmark
description: Run gold standard prompts against the system and generate compliance audit report
triggers:
  - /benchmark
  - benchmark system
  - run benchmarks
  - test routing
---

# Benchmark Skill

## Overview
Runs gold standard prompts from `.claude/benchmarks/suite.yaml` against the current system configuration. Captures detailed metrics via hooks and generates an audit report using Gemini for large-context analysis.

## Usage
```
/benchmark                    # Run full suite
/benchmark [prompt_id]        # Run specific prompt
/benchmark --runs=5           # Multiple runs for variance analysis
/benchmark --quick            # Run only tier-1 tests (fast validation)
```

---

## Phase 1: Initialize Benchmark Mode

**Step 1: Set benchmark flag**
```bash
# Create benchmark flag (activates benchmark-logger hook)
touch /tmp/claude-benchmark-active

# Create run directory with timestamp
RUN_ID=$(date +%Y%m%d-%H%M%S)
RUN_DIR="$HOME/.claude/benchmarks/runs/$RUN_ID"
mkdir -p "$RUN_DIR"

# Record git commit for version tracking
git rev-parse HEAD > "$RUN_DIR/commit.txt" 2>/dev/null || echo "no-git" > "$RUN_DIR/commit.txt"

# Copy current config state
cp ~/.claude/routing-schema.json "$RUN_DIR/config-snapshot.json" 2>/dev/null || true
```

**Step 2: Load test suite**
```bash
# Parse suite.yaml
SUITE_FILE="$HOME/.claude/benchmarks/suite.yaml"
if [[ ! -f "$SUITE_FILE" ]]; then
    echo "ERROR: No benchmark suite found at $SUITE_FILE"
    exit 1
fi
```

---

## Phase 2: Execute Prompts

For each prompt in the suite (or specified prompt_id):

**Step 1: Prepare prompt context**
```bash
PROMPT_ID="<from suite>"
PROMPT_DIR="$RUN_DIR/$PROMPT_ID"
mkdir -p "$PROMPT_DIR"

# Clear trace log for fresh capture
rm -f /tmp/claude-benchmark-current/trace.jsonl
mkdir -p /tmp/claude-benchmark-current
```

**Step 2: Execute the prompt**
- Run the prompt through normal Claude Code flow
- Hooks automatically capture all tool calls, costs, agent invocations
- Wait for completion

**Step 3: Capture results**
```bash
# Copy trace to run directory
cp /tmp/claude-benchmark-current/trace.jsonl "$PROMPT_DIR/" 2>/dev/null || true
cp /tmp/claude-benchmark-current/metrics.json "$PROMPT_DIR/" 2>/dev/null || true

# Record completion status
echo "{\"prompt_id\":\"$PROMPT_ID\",\"completed\":true,\"timestamp\":\"$(date -Iseconds)\"}" > "$PROMPT_DIR/status.json"
```

**Step 4: Variance runs (if --runs=N specified)**
```bash
for run in $(seq 1 $RUNS); do
    # Repeat execution
    # Capture to $PROMPT_DIR/run-$run/
done
```

---

## Phase 3: Aggregate Metrics (Local - No LLM Cost)

**Step 1: Compute per-prompt metrics**
```bash
for prompt_dir in "$RUN_DIR"/*/; do
    [[ -f "$prompt_dir/trace.jsonl" ]] || continue
    
    jq -s '{
        total_cost: (map(.cost_usd) | add),
        total_tokens: (map(.tokens_in + .tokens_out) | add),
        tool_calls: length,
        models_used: (map(.model) | unique),
        agents_invoked: (map(select(.agent != "none")) | map(.agent) | unique),
        errors: (map(select(.is_error == true)) | length),
        duration_seconds: ((.[length-1].ts // 0) - (.[0].ts // 0))
    }' "$prompt_dir/trace.jsonl" > "$prompt_dir/computed_metrics.json"
done
```

**Step 2: Generate summary**
```bash
# Aggregate all prompt metrics into single summary
jq -s '{
    run_id: "'$RUN_ID'",
    total_prompts: length,
    total_cost: (map(.total_cost) | add),
    avg_cost_per_prompt: ((map(.total_cost) | add) / length),
    total_errors: (map(.errors) | add),
    all_models: (map(.models_used) | flatten | unique),
    all_agents: (map(.agents_invoked) | flatten | unique)
}' "$RUN_DIR"/*/computed_metrics.json > "$RUN_DIR/summary.json"
```

---

## Phase 4: Audit with Gemini (Single Large-Context Call)

**Pipeline enforcement:** Use gemini-slave for audit to leverage 1M token context.

```bash
# Concatenate all context for Gemini
{
    echo "=== BENCHMARK SUITE DEFINITION ==="
    cat ~/.claude/benchmarks/suite.yaml
    
    echo -e "\n=== RUN SUMMARY ==="
    cat "$RUN_DIR/summary.json"
    
    echo -e "\n=== PER-PROMPT METRICS ==="
    for f in "$RUN_DIR"/*/computed_metrics.json; do
        echo "--- $(dirname $f | xargs basename) ---"
        cat "$f"
    done
    
    echo -e "\n=== EXECUTION TRACES (sampled) ==="
    for f in "$RUN_DIR"/*/trace.jsonl; do
        echo "--- $(dirname $f | xargs basename) ---"
        head -50 "$f"  # Sample first 50 events per prompt
    done
    
} | gemini-slave benchmark-audit "Score this benchmark run against expected metrics. Identify violations and provide recommendations."
```

Save Gemini output:
```bash
# Gemini output goes to stdout, capture it
GEMINI_OUTPUT=$(... | gemini-slave benchmark-audit "...")
echo "$GEMINI_OUTPUT" > "$RUN_DIR/audit_report.md"
```

---

## Phase 5: Generate Final Report

**Output file:** `__benchmark_[commit]_[timestamp].md`

```markdown
# Benchmark Report

**Run ID:** $RUN_ID
**Commit:** $(cat $RUN_DIR/commit.txt)
**Timestamp:** $(date -Iseconds)

## Executive Summary
[From Gemini audit]

## Scores
| Metric | Score | Target |
|--------|-------|--------|
| Cost Accuracy | X% | 100% |
| Routing Accuracy | X% | 100% |
| Attention Retention | X% | 100% |
| Failure Recovery | X% | 100% |

## Per-Prompt Results
[Table of each prompt with pass/fail]

## Violations
[List from Gemini audit]

## Recommendations
[Prioritized from Gemini audit]

## Raw Data
- Traces: $RUN_DIR/*/trace.jsonl
- Metrics: $RUN_DIR/*/computed_metrics.json
- Gemini Audit: $RUN_DIR/audit_report.md
```

---

## Phase 6: Cleanup

```bash
# Remove benchmark flag
rm -f /tmp/claude-benchmark-active

# Clean temp files
rm -rf /tmp/claude-benchmark-current

# Output report path
echo "Benchmark complete. Report: $RUN_DIR/__benchmark_report.md"
```

---

## Orchestrator Handoff (Optional)

If violations are found, delegate to orchestrator for review:

```javascript
Task({
  description: "Review benchmark findings and propose improvements",
  subagent_type: "Explore",
  model: "sonnet",
  prompt: `AGENT: orchestrator

TASK: Review benchmark audit findings and create improvement plan

CONTEXT:
- Benchmark run: $RUN_ID
- Violations found: [list from audit]
- Key metrics: [from summary.json]

EXPECTED OUTPUT:
1. Prioritized list of fixes
2. Specific file changes needed (CLAUDE.md, routing-schema.json, agent configs)
3. Estimated impact on benchmark scores

DO NOT: Implement changes directly. Output plan for user approval.`
})
```

---

## Quick Reference

| Command | Action |
|---------|--------|
| `/benchmark` | Full suite run |
| `/benchmark haiku-file-search` | Single prompt |
| `/benchmark --runs=3` | 3x runs for variance |
| `/benchmark --quick` | Tier-1 only |
| `/benchmark --audit-only` | Re-run Gemini audit on existing run |
