---
name: implement
description: Plan and implement a feature end-to-end. Spawns architect for planning, then dispatches workers via background team-run.
---

# /implement Skill v2.0

## Purpose

Single command to go from feature description to working code:

```
/implement "Add a health-check HTTP endpoint at /healthz"
```

**Pipeline:** Architect → goyoke-plan-impl → goyoke-team-run (background)

**Returns to user in <15 seconds after architect completes.**

---

## Invocation

```bash
/implement "<feature description>"
/implement                          # Prompts for description
```

---

## Workflow

When `/implement` is invoked, the `goyoke-skill-guard` PreToolUse hook has already:
- Created the team directory (`{goyoke_session_dir}/teams/{timestamp}.implementation/`)
- Written `active-skill.json` with guard restrictions + `team_dir` path
- Restricted the router to: Task, Bash, Read, AskUserQuestion, Skill

The Router executes the following steps:

### 1. Extract Feature Description

```javascript
const description = args.trim();
if (!description) {
    // Ask user for feature description
    AskUserQuestion({
        questions: [{
            question: "What feature do you want to implement?",
            header: "Feature",
            options: [
                {label: "Describe it", description: "I'll type a feature description"}
            ],
            multiSelect: false
        }]
    });
    // Use the response as description
}
```

Output:
```
[implement] Feature: {description}
```

### 2. Read Team Directory from Guard File

```javascript
Read({ file_path: `${goyoke_session_dir}/active-skill.json` })
// Extract team_dir from JSON response
```

The `goyoke_session_dir` is resolved by reading `{project_root}/.goyoke/current-session`. The project root can be found via `git rev-parse --show-toplevel` or `GOYOKE_PROJECT_ROOT` env var.

### 3. Spawn Architect

Invoke the architect agent to produce `implementation-plan.json` and `specs.md`. The architect writes these to `SESSION_DIR/` (session-level artifacts, not team_dir).

```javascript
Task({
    description: "Architect: plan implementation",
    subagent_type: "Plan",
    model: "sonnet",
    prompt: `AGENT: architect

TASK: Plan the implementation of the following feature and produce implementation-plan.json + specs.md.

FEATURE DESCRIPTION:
${description}

PROJECT CONTEXT:
- Project root: ${process.cwd()}
- Language: detected from project (check go.mod, pyproject.toml, package.json, DESCRIPTION)
- Conventions: load from ~/.claude/conventions/ based on detected language

REQUIRED OUTPUTS (ALL THREE MANDATORY):
1. SESSION_DIR/implementation-plan.json — Machine-readable plan (write FIRST)
2. SESSION_DIR/specs.md — Human-readable plan with decisions and risk register
3. write_todos — Task list derived from implementation phases

INSTRUCTIONS:
- Read the architect agent instructions at ~/.claude/agents/architect/architect.md
- Explore the codebase to understand existing patterns before planning
- Task descriptions in implementation-plan.json must be COMPLETE — workers cannot ask clarifying questions
- Each task needs: agent ID, target_packages, related_files, blocked_by, acceptance_criteria
- Keep it focused — only plan what was asked for
- Write outputs to SESSION_DIR/ (available as environment variable)

CONSTRAINTS:
- Do NOT implement code (planning only)
- Do NOT skip implementation-plan.json
- Maximum 2 clarifying questions if scope is ambiguous`
});
```

After architect completes:
```
[implement] Plan created: SESSION_DIR/implementation-plan.json
[implement] Specs: SESSION_DIR/specs.md
```

### 4. Validate Plan Exists

```bash
goyoke_session_dir="$(cat "$(git rev-parse --show-toplevel 2>/dev/null || echo .)/.goyoke/current-session" 2>/dev/null)"
goyoke_session_dir="${goyoke_session_dir:-.goyoke/sessions/$(date +%Y%m%d-%H%M%S)}"
plan_file="$goyoke_session_dir/implementation-plan.json"

if [[ ! -f "$plan_file" ]]; then
    echo "[implement] ERROR: Architect did not produce implementation-plan.json"
    echo "[implement] Check $goyoke_session_dir/specs.md for details"
    rm -f "$goyoke_session_dir/active-skill.json"
    # STOP — do not proceed
fi

# Quick validation
task_count=$(jq '.tasks | length' "$plan_file")
wave_info=$(jq -r '.tasks | group_by(.blocked_by | length) | map(length) | @json' "$plan_file")

echo "[implement] Plan: $task_count tasks"
```

### 5. Generate Team Config

Use the hook-provided `team_dir` from `active-skill.json` (read in Step 2). No `mkdir -p` needed — the hook already created the directory.

```bash
# team_dir was extracted from active-skill.json in Step 2
goyoke-plan-impl \
    --plan="$plan_file" \
    --project-root="$(pwd)" \
    --output="$team_dir"

if [[ $? -ne 0 ]]; then
    echo "[implement] ERROR: goyoke-plan-impl failed"
    echo "[implement] Check plan validity: jq . $plan_file"
    rm -f "$goyoke_session_dir/active-skill.json"
    # STOP
fi

echo "[implement] Team config: $team_dir"
```

### 6. Launch Background Execution

```
result = mcp__goyoke-interactive__team_run({
    team_dir: "$team_dir",
    wait_for_start: true,
    timeout_ms: 10000
})
if !result.success:
    echo "[implement] ERROR: ${result.result}"
    echo "[implement] Check: $team_dir/runner.log"
else:
    background_pid = result.background_pid
    # Read wave structure for display
    wave_summary=$(jq -r '.waves[] | "  Wave \(.wave_number): \(.members | length) workers [\(.members | map(.agent) | unique | join(", "))]"' "$team_dir/config.json")
    budget=$(jq -r '.budget_max_usd' "$team_dir/config.json")

    echo ""
    echo "[implement] Team launched (PID $background_pid)"
    echo "[implement] Budget: \$$budget"
    echo "$wave_summary"
    echo ""
    echo "[implement] Monitor:  /team-status"
    echo "[implement] Results:  /team-result"
    echo "[implement] Cancel:   /team-cancel"
```

### 7. Remove Skill Guard

```bash
rm -f "$goyoke_session_dir/active-skill.json"
```

---

## Example Session

```
> /implement "Add a goyoke-version binary that prints version and build time, with --json flag"

[implement] Feature: Add a goyoke-version binary that prints version and build time, with --json flag
[implement] Planning...

[architect explores codebase, creates plan]

[implement] Plan created: SESSION_DIR/implementation-plan.json
[implement] Specs: SESSION_DIR/specs.md
[implement] Plan: 2 tasks

[implement] Team config: .goyoke/sessions/20260209-143000/teams/1770551000.implementation

[implement] Team launched (PID 12345)
[implement] Budget: $10.00
  Wave 1: 1 workers [go-pro]
  Wave 2: 1 workers [go-pro]

[implement] Monitor:  /team-status
[implement] Results:  /team-result
[implement] Cancel:   /team-cancel
```

---

## Error Handling

| Error | Cause | Resolution |
|-------|-------|------------|
| Architect didn't produce JSON | Feature too vague or architect confused | Check specs.md for details, re-run with clearer description |
| goyoke-plan-impl failed | Invalid plan JSON (bad task_ids, circular deps) | Check error output, fix plan manually or re-run architect |
| Team failed to start | Binary not found or permission issue | Verify `goyoke-team-run` is built: `go build -o ./goyoke-team-run ./cmd/goyoke-team-run/` |
| Workers can't write files | Missing Write/Edit in allowed tools | Verify `augmentToolsForImplementation()` in spawn.go |

---

## Notes

- Architect runs in foreground (needs to explore codebase, may ask 1-2 clarifying questions)
- Team-run executes in background (no interaction needed)
- Workers within a wave run in parallel
- Wave failure propagation: failed wave N → subsequent waves skipped
- The architect decides the DAG structure via `blocked_by` relationships
- `goyoke-plan-impl` computes parallel waves via Kahn's algorithm

---

**Skill Version**: 2.0
**Last Updated**: 2026-02-10
**Maintained By**: System
