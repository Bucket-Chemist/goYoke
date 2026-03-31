# Bootstrap: Braintrust Analysis of .gogent/ Migration

## Context

The `.gogent/` runtime I/O migration requires deep architectural analysis. The normal `/braintrust` workflow is broken because it relies on the same `.claude/` write paths that are blocked. This document describes how to **manually bootstrap a braintrust analysis** by acting as Mozart (the orchestrator) and routing all I/O to a writable `.gogent/` directory.

Additionally, the TUI has a critical UX bug: **Bash permission prompts are invisible** because Claude Code CLI's permission system is terminal-based, not stream-json-based. The `acceptEdits` permission mode auto-approves file edits but still requires interactive terminal approval for risky Bash commands. Since the TUI pipes stdin/stdout via stream-json, there is no terminal for the approval prompt — the CLI blocks forever and the TUI sees no event. This must be fixed before any Bash-dependent workflow (including team-run launch) works from the TUI.

---

## Problem 1: Braintrust Is Broken

### Root Cause

The `/braintrust` skill workflow writes to `.claude/` at multiple points:

1. **`gogent-skill-guard`** (PreToolUse hook) creates `{session_dir}/teams/{ts}.braintrust/` and writes `active-skill.json` — both under `.claude/sessions/`
2. **Mozart** (spawned agent) writes `config.json`, `stdin_*.json`, `problem-brief.md` to the team directory — under `.claude/sessions/`
3. **Router** reads `active-skill.json` and launches `gogent-team-run` pointing at the team dir

Steps 1 and 2 fail because CC agents cannot write to `.claude/` paths. Step 3 never executes.

### Why Go Binaries Are Not Affected

`gogent-team-run` and `gogent-team-prepare-synthesis` use `os.WriteFile()` in Go — they are not subject to Claude Code's sandbox. They can write anywhere on the filesystem. The sandbox only applies to CC's own tools (Write, Edit, Bash) used by agents within a CC session.

This means: **if we can get a valid team directory with config.json and stdin files written to a non-.claude/ path, team-run will work fine.**

---

## Problem 2: TUI Permission Prompt Not Visible

### Root Cause

The NDJSON catalog (TUI-003 spike) documents this explicitly:

> `"control_request for permissions" | ❌ Does not exist (TUI-001 finding)`

Claude Code CLI does **not** emit a stream-json event when it needs permission approval. Instead, it uses an internal terminal-based prompt. When the TUI runs Claude CLI via `--input-format stream-json --output-format stream-json`, the CLI's stdin is a pipe, not a TTY. The permission prompt:

1. Is written to an internal terminal mechanism (not stdout NDJSON)
2. Waits for terminal input (not stdin JSON)
3. Never receives a response → **blocks forever**
4. The TUI sees no event → **appears as a silent hang**

### Evidence

- `internal/tui/cli/driver.go:282-283`: TUI starts CLI with `--permission-mode acceptEdits`
- `acceptEdits` auto-approves Write/Edit but still prompts for Bash commands Claude considers risky
- `internal/tui/cli/events.go:384-426`: Parser handles 6 event types — none is a permission request
- The binary execution (`gogent-team-run`) is exactly the kind of command that triggers a Bash permission prompt

### Impact

**Any Bash command that CC considers "risky" silently hangs in the TUI.** This includes:
- Executing custom binaries (`gogent-team-run`, `gogent-team-prepare-synthesis`)
- Any `nohup` or background process launch
- Commands the CLI hasn't seen before in this session

### Fix Options

See [Section: Fixing TUI Permission Prompts](#fixing-tui-permission-prompts) below.

---

## Bootstrap Procedure: Manual Braintrust

Since braintrust is broken and TUI permission prompts are invisible, we bootstrap the analysis by:

1. **Router acts as Mozart** — skips the interview/scout phase (reconnaissance already done)
2. **Router writes all files to `.gogent/`** — outside CC sandbox
3. **Team-run is launched from a terminal** (not from within CC) — bypasses permission issue

### Prerequisites

- [ ] GOgent-Fortress project binaries built: `bin/gogent-team-run`, `bin/gogent-team-prepare-synthesis`
- [ ] A terminal session with PATH or absolute binary paths available
- [ ] `.gogent/` directory writable at project root

### Step-by-Step

#### Step 1: Create Team Directory

```bash
TEAM_DIR="/home/doktersmol/Documents/GOgent-Fortress/.gogent/braintrust-$(date +%s)"
mkdir -p "$TEAM_DIR"
```

#### Step 2: Write config.json

The config must follow the schema at `.claude/schemas/teams/braintrust.json`. Critical fields:

```json
{
  "version": "1.0.0",
  "team_name": "braintrust-{timestamp}",
  "workflow_type": "braintrust",
  "project_root": "/home/doktersmol/Documents/GOgent-Fortress",
  "session_id": "bootstrap-{timestamp}",
  "created_at": "{ISO-8601}",
  "background_pid": null,
  "budget_max_usd": 100.0,
  "budget_remaining_usd": 100.0,
  "warning_threshold_usd": 80.0,
  "status": "pending",
  "started_at": null,
  "completed_at": null,
  "waves": [
    {
      "wave_number": 1,
      "description": "Parallel theoretical and practical analysis",
      "members": [
        {
          "name": "einstein",
          "agent": "einstein",
          "model": "opus",
          "stdin_file": "stdin_einstein.json",
          "stdout_file": "stdout_einstein.json",
          "status": "pending",
          "process_pid": null,
          "exit_code": null,
          "cost_usd": 0.0,
          "cost_status": "",
          "error_message": "",
          "retry_count": 0,
          "max_retries": 1,
          "timeout_ms": 1200000
        },
        {
          "name": "staff-architect",
          "agent": "staff-architect-critical-review",
          "model": "opus",
          "stdin_file": "stdin_staff-architect.json",
          "stdout_file": "stdout_staff-arch.json",
          "status": "pending",
          "process_pid": null,
          "exit_code": null,
          "cost_usd": 0.0,
          "cost_status": "",
          "error_message": "",
          "retry_count": 0,
          "max_retries": 1,
          "timeout_ms": 1200000
        }
      ],
      "on_complete_script": "/home/doktersmol/Documents/GOgent-Fortress/bin/gogent-team-prepare-synthesis"
    },
    {
      "wave_number": 2,
      "description": "Synthesis of orthogonal analyses",
      "members": [
        {
          "name": "beethoven",
          "agent": "beethoven",
          "model": "opus",
          "stdin_file": "stdin_beethoven.json",
          "stdout_file": "stdout_beethoven.json",
          "status": "pending",
          "process_pid": null,
          "exit_code": null,
          "cost_usd": 0.0,
          "cost_status": "",
          "error_message": "",
          "retry_count": 0,
          "max_retries": 1,
          "timeout_ms": 1200000
        }
      ],
      "on_complete_script": null
    }
  ]
}
```

**Critical details:**
- `on_complete_script` MUST be an absolute path — `gogent-team-prepare-synthesis` is not in PATH
- `timeout_ms: 1200000` = 20 minutes per agent
- `budget_max_usd: 100.0` — generous budget for opus agents
- `project_root` must be the absolute path to GOgent-Fortress (team-run uses this as `cmd.Dir`)

#### Step 3: Write stdin_einstein.json

Follow the schema at `.claude/schemas/teams/stdin-stdout/braintrust-einstein.json`.

Key sections to populate:
- `problem_brief` — the full migration problem statement with scope, constraints, complexity signals
- `codebase_context` — key files with excerpts from reconnaissance (`pkg/session/session_dir.go`, `internal/tui/session/persistence.go`, `cmd/gogent-skill-guard/main.go`, `cmd/gogent-team-run/spawn.go`, `cmd/gogent-load-context/main.go`)
- `analysis_axes` — conceptual focus questions, novel angles, first principles to challenge
- `output_instructions` — MUST tell Einstein to output JSON to stdout only, NOT use Write tool

**The `output_instructions.critical` field is essential:**
```json
"critical": "Your ENTIRE output must be a single JSON object conforming to the stdout section of this schema. gogent-team-run captures your process stdout as your result file. Do NOT use the Write() tool. Output JSON to stdout only."
```

#### Step 4: Write stdin_staff-architect.json

Follow the schema at `.claude/schemas/teams/stdin-stdout/braintrust-staff-architect.json`.

Key sections:
- `problem_brief` — same problem statement as Einstein
- `plan_to_review` — the proposed Option D migration plan with phases, assumptions, directory structure
- `codebase_context` — focus on existing technical debt and fragile patterns
- `review_focus` — all 7 layers (assumptions, dependencies, failure_modes, cost_benefit, testing, architecture_smells, contractor_readiness)
- `output_instructions` — same stdout-only instruction as Einstein

#### Step 5: Write stdin_beethoven.json

Follow the schema at `.claude/schemas/teams/stdin-stdout/braintrust-beethoven.json`.

Key sections:
- `problem_brief` — condensed version with success criteria
- `pre_synthesis_path` — absolute path to `{team_dir}/pre-synthesis.md` (generated by `gogent-team-prepare-synthesis` after wave 1)
- `output_instructions` — stdout-only, tell Beethoven to Read pre-synthesis.md via the Read tool

#### Step 6: Write problem-brief.md

A human-readable version of the problem brief for reference. Not consumed by team-run directly, but agents may Read it from team_dir.

#### Step 7: Launch team-run FROM A TERMINAL

**Do NOT launch from within the TUI or Claude Code session.** The TUI cannot display Bash permission prompts (see Problem 2 above).

Open a separate terminal and run:

```bash
cd /home/doktersmol/Documents/GOgent-Fortress
bin/gogent-team-run "$TEAM_DIR"
```

The binary will:
1. Daemonize (setsid, redirect output to `runner.log`)
2. Acquire PID file
3. Write `background_pid` and `status: "running"` to config.json
4. Spawn Einstein + Staff-Architect in parallel (Wave 1)
5. Execute `bin/gogent-team-prepare-synthesis "$TEAM_DIR"` after Wave 1 completes
6. Spawn Beethoven (Wave 2)
7. Write `status: "completed"` to config.json

#### Step 8: Monitor Progress

From any terminal:

```bash
# Watch config.json for status updates
watch -n 5 'jq "{status, waves: [.waves[] | {wave: .wave_number, members: [.members[] | {name, status, cost_usd}]}]}" "$TEAM_DIR/config.json"'

# Tail the runner log
tail -f "$TEAM_DIR/runner.log"

# Watch for stream output
tail -f "$TEAM_DIR/stream_einstein.ndjson"
```

From within a CC session (if TUI is running):
```
/team-status    # if the skill can read from .gogent/ paths
```

#### Step 9: Collect Results

After completion (status: "completed" in config.json):

```bash
# Einstein's theoretical analysis
jq . "$TEAM_DIR/stdout_einstein.json"

# Staff-Architect's practical review
jq . "$TEAM_DIR/stdout_staff-arch.json"

# Pre-synthesis (merged wave 1 outputs)
cat "$TEAM_DIR/pre-synthesis.md"

# Beethoven's final synthesis
jq . "$TEAM_DIR/stdout_beethoven.json"

# Cost summary
jq '{total_budget: .budget_max_usd, remaining: .budget_remaining_usd, spent: (.budget_max_usd - .budget_remaining_usd), per_agent: [.waves[].members[] | {name, cost_usd, status}]}' "$TEAM_DIR/config.json"
```

---

## Pre-Built Team Directory

A complete, ready-to-launch team directory has already been created at:

```
/home/doktersmol/Documents/GOgent-Fortress/.gogent/braintrust-1774939872/
├── config.json              # Team execution configuration
├── problem-brief.md         # Human-readable problem brief
├── stdin_einstein.json      # Einstein's input (theoretical analysis)
├── stdin_staff-architect.json  # Staff-Architect's input (practical review)
└── stdin_beethoven.json     # Beethoven's input (synthesis)
```

**To launch immediately:**

```bash
cd /home/doktersmol/Documents/GOgent-Fortress
bin/gogent-team-run .gogent/braintrust-1774939872
```

---

## Fixing TUI Permission Prompts

### The Core Problem

Claude Code CLI's permission system is terminal-based, not stream-json-based. When the CLI needs Bash approval in `acceptEdits` mode, it writes to an internal terminal prompt and waits for TTY input. Since the TUI pipes stdin/stdout via JSON, the prompt goes nowhere and the CLI blocks forever.

### Fix Option A: Use `--permission-mode bypassPermissions` (REJECTED)

This flag doesn't exist in Claude Code CLI. The available modes are: `default`, `acceptEdits`, `plan`, `bypassPermissions` — but `bypassPermissions` may not be available or may be a CC internal mode.

### Fix Option B: Allowlist Custom Binaries via Settings (RECOMMENDED — SHORT-TERM)

Claude Code allows configuring allowed Bash commands in `settings.json`. Adding GOgent binaries to the allowlist means they won't trigger permission prompts:

**File: `~/.claude/settings.json`** (or project-level `.claude/settings.json`):

```json
{
  "permissions": {
    "allow": [
      "Bash(bin/gogent-team-run:*)",
      "Bash(bin/gogent-team-prepare-synthesis:*)",
      "Bash(nohup:*)",
      "Bash(/home/doktersmol/Documents/GOgent-Fortress/bin/gogent-team-run:*)",
      "Bash(/home/doktersmol/Documents/GOgent-Fortress/bin/gogent-team-prepare-synthesis:*)"
    ]
  }
}
```

**Verification:** After adding, the TUI should be able to launch team-run without hanging.

### Fix Option C: Intercept Permission Requests in TUI (RECOMMENDED — LONG-TERM)

The Claude Code CLI source (not available to us) may have an undocumented mechanism for permission handling in stream-json mode. Investigation steps:

1. **Check CC CLI docs** for any `--auto-approve` or `--non-interactive` flag
2. **Inspect CC source** (if accessible) for how `permission_request` events are emitted in non-TTY mode
3. **Test hypothesis**: Run `claude --input-format stream-json --output-format stream-json --permission-mode default` in a pipe and trigger a Bash command — observe what happens on stdout/stderr

If the CLI does emit a permission request event in some form:
- Add a new event type to `internal/tui/cli/events.go`
- Add a handler in the TUI that shows an approval dialog
- Send the response back via stdin JSON

If it does NOT emit any event:
- File a feature request with Anthropic for stream-json permission event support
- Use Fix Option B as the permanent workaround

### Fix Option D: Switch to `bypassPermissions` Mode (IF AVAILABLE)

Check if `--permission-mode bypassPermissions` is a valid option:

```bash
claude --help 2>&1 | grep -A5 "permission-mode"
```

If available, change `internal/tui/cli/driver.go:279-282`:

```go
permMode := d.opts.PermissionMode
if permMode == "" {
    permMode = "bypassPermissions"  // Changed from "acceptEdits"
}
```

**Risk:** This auto-approves ALL tool calls. The TUI would need its own permission layer for safety.

### Recommended Path

1. **Immediate (today):** Add bin paths to `settings.json` allowlist (Fix B) → unblocks team-run from TUI
2. **Short-term (this week):** Investigate if CC CLI has a non-TTY permission mechanism (Fix C investigation)
3. **Medium-term:** If no CC mechanism exists, build TUI-side permission layer with `bypassPermissions` + custom approval UI (Fix D + custom UI)

---

## Checklist

- [ ] Fix TUI permission prompt (settings.json allowlist) — BLOCKING
- [ ] Launch braintrust from terminal using pre-built team directory
- [ ] Monitor progress via runner.log and config.json
- [ ] Collect Einstein + Staff-Architect + Beethoven outputs
- [ ] Synthesize findings into migration implementation plan
- [ ] Create implementation tickets from braintrust analysis

---

## Files Referenced

| File | Purpose |
|------|---------|
| `.claude/schemas/teams/braintrust.json` | Config schema template |
| `.claude/schemas/teams/stdin-stdout/braintrust-einstein.json` | Einstein stdin/stdout contract |
| `.claude/schemas/teams/stdin-stdout/braintrust-staff-architect.json` | Staff-Architect stdin/stdout contract |
| `.claude/schemas/teams/stdin-stdout/braintrust-beethoven.json` | Beethoven stdin/stdout contract |
| `cmd/gogent-team-run/spawn.go` | Agent spawning (path-agnostic) |
| `cmd/gogent-team-run/wave.go` | Wave execution + inter-wave scripts |
| `cmd/gogent-team-prepare-synthesis/main.go` | Pre-synthesis markdown generation |
| `internal/tui/cli/driver.go` | TUI CLI driver (permission mode config) |
| `internal/tui/cli/events.go` | NDJSON event parser (missing permission events) |
| `tickets/completed/tui-migration/spike-results/ndjson-catalog.md` | Event catalog confirming no permission event type |
