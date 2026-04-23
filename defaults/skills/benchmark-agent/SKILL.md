---
name: benchmark-agent
description: Evaluate goYoke agents against SkillsBench benchmarks via Harbor
triggers:
  - /benchmark-agent
  - benchmark agent
  - agent benchmark
  - skillsbench
---

# Benchmark Agent Skill

## Overview
Evaluates goYoke agents against SkillsBench tasks using Harbor's evaluation framework.
Measures whether agent identity injection (personality, conventions, rules) improves task performance.

## Usage
```
/benchmark-agent go-pro                           Run matching tasks with go-pro identity
/benchmark-agent go-pro --task fix-build-agentops  Run specific task
/benchmark-agent go-pro --raw                      Run without identity injection (baseline)
/benchmark-agent go-pro --compare                  Run both, show delta
/benchmark-agent --list                            Show agent-to-task mappings
/benchmark-agent go-pro --difficulty easy           Filter by difficulty
/benchmark-agent go-pro --dry-run                   Show what would run
/benchmark-agent go-pro --model claude-sonnet-4-6  Override model (default: from agents-index.json)
```

---

## Constants

```
GOYOKE_DIR="/home/doktersmol/Documents/goYoke"
BENCHMARK_TOOLS="${GOYOKE_DIR}/tools/benchmark"
DEFAULT_TASKS_DIR="/home/doktersmol/Documents/skillsbench/tasks-no-skills"
# DEFAULT_MODEL is resolved at runtime from agents-index.json via get_agent_model()
```

---

## Phase A: Parse Arguments

Parse the user's input to extract:

| Argument | Type | Default | Description |
|----------|------|---------|-------------|
| `agent_id` | positional | (required unless --list) | goYoke agent ID |
| `--task` | string | (all matching) | Run specific task by name |
| `--raw` | flag | false | Run without identity injection |
| `--compare` | flag | false | Run both with/without identity, show delta |
| `--list` | flag | false | Show available agent-to-task mappings |
| `--difficulty` | string | "hard" | Max difficulty: easy, medium, hard |
| `--dry-run` | flag | false | Show plan without executing |
| `--model` | string | from agents-index.json | Override model (resolved via `get_agent_model()` if not set) |
| `--tasks-dir` | string | DEFAULT_TASKS_DIR | Override SkillsBench tasks path |

### Model Resolution (if `--model` not provided)

```bash
PYTHONPATH=/home/doktersmol/Documents/goYoke/tools/benchmark \
  uv run python -c "
from identity_injector import get_agent_model
print(get_agent_model('${AGENT_ID}'))
"
```

This reads the agent's tier from agents-index.json and maps it to the current model ID.
The mapping (`MODEL_TIER_MAP`) lives in `identity_injector.py` -- update it there when new models release.

### If `--list` is set:

Run the following and display the output, then STOP:

```bash
PYTHONPATH=/home/doktersmol/Documents/goYoke/tools/benchmark \
  uv run python -c "
from task_matcher import scan_tasks, match_tasks, list_available_agents
from pathlib import Path
import json

tasks_dir = Path('/home/doktersmol/Documents/skillsbench/tasks-no-skills')
all_tasks = scan_tasks(tasks_dir)
print(f'SkillsBench tasks scanned: {len(all_tasks)}')
print()

for agent_id in list_available_agents():
    matched = match_tasks(agent_id, all_tasks)
    print(f'{agent_id}: {len(matched)} tasks')
    for t in matched[:5]:
        print(f'  - {t.name} [{t.difficulty}]')
    if len(matched) > 5:
        print(f'  ... and {len(matched) - 5} more')
    print()
"
```

---

## Phase B: Task Discovery

Find tasks matching the agent. If `--task` is specified, find just that task.

```bash
PYTHONPATH=/home/doktersmol/Documents/goYoke/tools/benchmark \
  uv run python -c "
from task_matcher import scan_tasks, match_tasks, filter_by_difficulty
from pathlib import Path
import json

TASKS_DIR = '${TASKS_DIR}'  # from parsed args, or DEFAULT_TASKS_DIR
AGENT_ID = '${AGENT_ID}'
SPECIFIC_TASK = '${SPECIFIC_TASK}'  # empty string if not set
MAX_DIFFICULTY = '${MAX_DIFFICULTY}'  # from --difficulty, default 'hard'

all_tasks = scan_tasks(Path(TASKS_DIR))

if SPECIFIC_TASK:
    matched = [t for t in all_tasks if t.name == SPECIFIC_TASK]
    if not matched:
        print(json.dumps({'error': f'Task {SPECIFIC_TASK} not found in {TASKS_DIR}'}))
    else:
        result = [{'path': str(t.path), 'name': t.name, 'difficulty': t.difficulty, 'category': t.category} for t in matched]
        print(json.dumps(result, indent=2))
else:
    matched = match_tasks(AGENT_ID, all_tasks)
    matched = filter_by_difficulty(matched, MAX_DIFFICULTY)
    result = [{'path': str(t.path), 'name': t.name, 'difficulty': t.difficulty, 'category': t.category} for t in matched]
    print(json.dumps(result, indent=2))
"
```

**Present the matched tasks to the user for confirmation before proceeding.**
Show: count, task names, difficulty distribution.

If `--dry-run` is set, show the plan and STOP here.

If zero tasks match, inform the user and suggest using `--task <name>` for manual selection.

---

## Phase C: Execute Harbor Runs

For each matched task, run Harbor. Set a results directory per run:

```bash
RESULTS_DIR="/tmp/benchmark-agent-$(date +%Y%m%d-%H%M%S)"
mkdir -p "${RESULTS_DIR}"
```

### With goYoke Identity Injection

```bash
PYTHONPATH=/home/doktersmol/Documents/goYoke/tools/benchmark \
  harbor run \
    -p "${TASK_PATH}" \
    --agent-import-path "goyoke_adapter:goYokeAgent" \
    --ak "agent_id=${AGENT_ID}" \
    -m "${MODEL}" \
    --jobs-dir "${RESULTS_DIR}/goyoke" \
    -n 1 -k 1
```

### Baseline (if --raw or --compare)

```bash
harbor run \
    -p "${TASK_PATH}" \
    -a claude-code \
    -m "${MODEL}" \
    --jobs-dir "${RESULTS_DIR}/baseline" \
    -n 1 -k 1
```

**Run tasks sequentially.** Each Harbor run is a Docker-containerized execution.

**Timeout:** Harbor tasks have their own timeout from task.toml (typically 600-1800s). Do not add additional timeout wrapping.

---

## Phase D: Collect Results

After all runs complete, collect results from the jobs directory.

Harbor writes results with this structure:
```
jobs-dir/
  {job-name}/
    trials/
      {task-name}/
        attempt-0/
          logs/
            agent/
              claude-code.txt
              trajectory.json
          reward.txt              # 0.0 or 1.0
          metadata.json
```

Parse results:

```bash
# Collect goYoke results
for reward_file in ${RESULTS_DIR}/goyoke/*/trials/*/attempt-0/reward.txt; do
    task_name=$(basename $(dirname $(dirname "$reward_file")))
    reward=$(cat "$reward_file" 2>/dev/null || echo "ERROR")
    echo "${task_name}: ${reward}"
done

# Collect baseline results (if --compare or --raw)
for reward_file in ${RESULTS_DIR}/baseline/*/trials/*/attempt-0/reward.txt; do
    task_name=$(basename $(dirname $(dirname "$reward_file")))
    reward=$(cat "$reward_file" 2>/dev/null || echo "ERROR")
    echo "${task_name}: ${reward}"
done
```

---

## Phase E: Generate Report

Generate a markdown report with the collected results.

### Report Template

````markdown
# Agent Benchmark Report: {agent_id}

**Model:** {model}
**Tasks:** {n_matched} matched, {n_run} executed
**Date:** {timestamp}
**Results Dir:** {results_dir}

## Results

| Task | Difficulty | goYoke | Baseline | Delta |
|------|-----------|--------|----------|-------|
| {task_name} | {difficulty} | {PASS/FAIL} | {PASS/FAIL/-} | {+1/0/-1/-} |

**Pass Rate:** goYoke {X}/{N} ({pct}%) | Baseline {Y}/{N} ({pct}%)

## Identity Injection Stats

```bash
# Get injection size
PYTHONPATH=/home/doktersmol/Documents/goYoke/tools/benchmark \
  uv run python -c "
from identity_injector import build_full_agent_context, load_agent_context_requirements
reqs = load_agent_context_requirements('${AGENT_ID}')
result = build_full_agent_context('${AGENT_ID}', reqs, [], '')
chars = len(result)
tokens_est = chars // 4  # rough estimate
print(f'Identity + conventions: {chars} chars (~{tokens_est} tokens)')
print(f'Context window usage: ~{tokens_est * 100 // 200000}%')
"
```

## Interpretation

{Brief analysis: Did identity injection help? Hurt? No difference?
Look at which tasks differ and whether the pattern makes sense.}
````

If `--compare` is not set, omit the Baseline and Delta columns.
If `--raw` is set, only show the Baseline column (no goYoke column).

---

## Error Handling

| Error | Action |
|-------|--------|
| Harbor not found | Print: "Harbor not installed. Install from https://github.com/laude-institute/harbor" |
| SkillsBench not found | Print: "SkillsBench not found at {tasks_dir}. Clone from https://github.com/benchflow-ai/skillsbench" |
| Docker not running | Print: "Docker is not running. Harbor requires Docker for task execution." |
| Agent not in mapping | Print: "Agent '{id}' has no task mapping. Use --list to see available agents, or --task to run a specific task." |
| Task not found | Print: "Task '{name}' not found in {tasks_dir}." |
| Harbor run fails | Log the error, continue with remaining tasks, mark as ERROR in report |

## Pre-flight Checks

Before executing any Harbor runs, verify:

```bash
# Check Harbor is installed
which harbor || echo "ERROR: harbor not found"

# Check Docker is running
docker info >/dev/null 2>&1 || echo "ERROR: Docker not running"

# Check SkillsBench exists
ls "${TASKS_DIR}/*/task.toml" >/dev/null 2>&1 || echo "ERROR: No tasks found in ${TASKS_DIR}"

# Check ANTHROPIC_API_KEY is set
[ -n "${ANTHROPIC_API_KEY}" ] || echo "WARNING: ANTHROPIC_API_KEY not set"
```

---

## Notes

- **tasks-no-skills/** is the default directory because our agents ARE the skill layer -- we test on raw tasks
- Identity injection runs on the HOST before Docker, not inside Docker (CLAUDE_CONFIG_DIR is overridden in container)
- Each Harbor run creates a Docker container, runs the agent, evaluates with pytest, writes reward.txt
- Results persist in RESULTS_DIR for later analysis
- The existing `/benchmark` skill tests routing compliance; this skill tests agent *effectiveness*
