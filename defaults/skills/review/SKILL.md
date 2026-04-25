---
name: review
description: Orchestrated multi-domain code review with severity-grouped findings and approval status
---

# Review Skill v3.0

## Purpose

Automated code review through coordinated specialist reviewers. Analyzes changed files, identifies relevant review domains, spawns reviewers via background team-run, and provides findings via `/team-result`.

**What this skill does:**

1. **Detect** — Find changed files via git diff or specified scope
2. **Classify** — Identify languages and architectural layers present
3. **Select** — Choose relevant reviewers (backend, frontend, standards, architecture)
4. **Execute** — Dispatch reviewers via background team-run
5. **Launch** — Start `goyoke-team-run` in background, return immediately

**What this skill does NOT do:**

- Implement fixes (generates recommendations only)
- Enforce routing rules (handled by hooks)
- Replace human review (supplements, doesn't replace)

---

## Invocation

- `/review` — Review all staged changes (git diff --staged)
- `/review --all` — Review all uncommitted changes (git diff HEAD)
- `/review --scope=<glob>` — Review specific files (e.g., "\*_/_.go")
- `/review path/to/file` — Review specific file or directory

---

## Prerequisites

**Required tools:**

- `git` (for change detection)
- `jq` (JSON processing)
- `goyoke-team-run` (team execution)

**Project setup:**
None required. Works in any git repository.

---

## Workflow

When `/review` is invoked, the Router must first set up the skill environment
by calling `prepare_skill` before following these instructions.

The `goyoke_session_dir` lives under `{project_root}/.goyoke/sessions/`, NOT `.claude/sessions/`. It is resolved by reading `{project_root}/.goyoke/current-session`.
- Restricted the router to: Task, Bash, Read, AskUserQuestion, Skill

The Router executes the following steps:

### Phase 1: Read Guard File and Detect Changes

#### Step 1: Read Team Directory from Guard File

```javascript
mcp__goyoke-interactive__prepare_skill({ skill: "review" })
// Returns: { team_dir, guard_active, router_allowed_tools, tui_translation }
// Extract team_dir from the response.
// If tui_translation is non-empty, follow it for all Task() calls below.
//
// Non-TUI fallback: Bash({ command: "goyoke-skill-guard --setup review" })
//
// ERROR HANDLING: If prepare_skill returns an error or guard_active is false:
//   Log warning, generate team_dir manually, continue without guard, skip release.
```

The `goyoke_session_dir` is resolved by reading `{project_root}/.goyoke/current-session`. The project root can be found via `git rev-parse --show-toplevel` or `GOYOKE_PROJECT_ROOT` env var.

#### Step 2: Detect Changed Files

Determine which files to review:

```bash
review_scope="staged"  # default
if [[ "$1" == "--all" ]]; then
    review_scope="all"
elif [[ "$1" == --scope=* ]]; then
    review_scope="glob"
    glob_pattern="${1#--scope=}"
elif [[ -n "$1" ]]; then
    review_scope="explicit"
    review_target="$1"
fi

case "$review_scope" in
    staged)
        files=$(git diff --staged --name-only)
        ;;
    all)
        files=$(git diff HEAD --name-only)
        ;;
    glob)
        files=$(find . -type f -path "$glob_pattern")
        ;;
    explicit)
        if [[ -d "$review_target" ]]; then
            files=$(find "$review_target" -type f)
        else
            files="$review_target"
        fi
        ;;
esac

if [[ -z "$files" ]]; then
    echo "[review] No files to review."
    mcp__goyoke-interactive__prepare_skill({ skill: "review", release: true })
# Non-TUI fallback: Bash({ command: "goyoke-skill-guard --release" })
    exit 0
fi

echo "[review] Found $(echo "$files" | wc -l) files to review"
```

---

### Phase 2: Reviewer Selection

Identify languages and select relevant reviewers:

```bash
# Group files by extension
declare -A langs
declare -A categories
file_count=0
while IFS= read -r file; do
    ext="${file##*.}"
    langs["$ext"]=1
    file_count=$((file_count + 1))
done <<< "$files"

# Detect cross-module (files from 3+ distinct top-level directories)
module_count=$(echo "$files" | cut -d/ -f1 | sort -u | wc -l)

# Map to review domains
reviewers=()
if [[ -n "${langs[go]}" || -n "${langs[py]}" ]]; then
    reviewers+=("backend-reviewer")
fi
if [[ -n "${langs[ts]}" || -n "${langs[tsx]}" || -n "${langs[jsx]}" ]]; then
    reviewers+=("frontend-reviewer")
fi
# Always include standards reviewer
reviewers+=("standards-reviewer")
# Include architect-reviewer only for large or cross-module changes
if [[ "$file_count" -ge 5 || "$module_count" -ge 3 ]]; then
    reviewers+=("architect-reviewer")
fi

echo "[review] Selected reviewers: ${reviewers[*]}"
```

---

### Phase 3: Background Team-Run Dispatch

#### Step 1: Generate config.json

Read the review template from `.claude/schemas/teams/review.json` and populate with dynamic values. Only include selected reviewers in `waves[0].members[]`.

**Template fields to populate:**
- `team_name`: `"review-$(date +%Y%m%d-%H%M%S)"`
- `workflow_type`: `"review"`
- `project_root`: `$(git rev-parse --show-toplevel)`
- `session_id`: basename of `$GOYOKE_SESSION_DIR`
- `created_at`: `$(date -u +%Y-%m-%dT%H:%M:%SZ)`
- `budget_max_usd`: `5.0`
- `budget_remaining_usd`: `5.0`
- `warning_threshold_usd`: `4.0`
- `status`: `"pending"`
- `background_pid`: `null`
- `started_at`: `null`
- `completed_at`: `null`

**Per-member fields** (from review.json template, filtered to selected reviewers):
- `model`: `"haiku"` for backend/frontend/standards, `"sonnet"` for architect
- `timeout_ms`: `600000` for all reviewers (10 min)
- `max_retries`: `2`
- `stdin_file`: `"stdin_{reviewer-name}.json"`
- `stdout_file`: `"stdout_{reviewer-name}.json"`
- All runtime fields: `null`/`0`/`""`/`"pending"`

Write to `$team_dir/config.json`.

#### Step 2: Generate Stdin Files

For each selected reviewer, generate a stdin JSON file compliant with `schemas/stdin/reviewer.json`.

**Required fields (all reviewers):**

```json
{
  "agent": "{reviewer-name}",
  "workflow": "review",
  "description": "Review {domain} code changes",
  "context": {
    "project_root": "{absolute project root}",
    "team_dir": "{absolute team directory}"
  },
  "review_scope": {
    "files": [
      {
        "path": "{relative file path}",
        "language": "{language}",
        "category": "{category}",
        "changed_lines": {"added": N, "removed": M},
        "is_new_file": false
      }
    ],
    "total_files": N,
    "languages_detected": ["{languages}"]
  },
  "git_context": {
    "commit_message": "{from git log -1 or staged changes summary}",
    "branch_name": "{current branch}"
  },
  "focus_areas": {},
  "project_conventions": {}
}
```

**Per-reviewer focus_areas:**

| Reviewer | focus_areas |
|----------|-------------|
| backend-reviewer | `{"security": true, "api_design": true, "concurrency": true, "error_handling": true}` |
| frontend-reviewer | `{"accessibility": true, "performance": true, "state_management": true, "component_design": true}` |
| standards-reviewer | `{"naming": true, "structure": true, "complexity": true, "dry_kiss_yagni": true}` |
| architect-reviewer | `{"module_boundaries": true, "coupling": true, "design_patterns": true, "change_impact": true}` |

**Per-reviewer project_conventions:**

Detect from project context:
- `language`: primary language of files being reviewed
- `conventions_file`: matching conventions file (e.g., `"go.md"`)

**File classification for review_scope:**

| Extension | Language | Category |
|-----------|----------|----------|
| `.go` | `go` | `backend` |
| `.py` | `python` | `backend` |
| `.ts` | `typescript` | `frontend` or `backend` (check path) |
| `.tsx` | `typescript` | `frontend` |
| `.jsx` | `javascript` | `frontend` |
| `.md` | `markdown` | `docs` |
| `.json` | `json` | `config` |
| `.yaml`/`.yml` | `yaml` | `config` |

**Changed lines detection:**

```bash
# Get per-file line counts from git diff
git diff --staged --numstat | while read added removed file; do
    echo "{\"path\": \"$file\", \"added\": $added, \"removed\": $removed}"
done
```

Write each stdin file to `$team_dir/stdin_{reviewer-name}.json`.

#### Step 3: Launch

```
result = mcp__goyoke-interactive__team_run({
    team_dir: "$team_dir",
    wait_for_start: true,
    timeout_ms: 10000
})
if !result.success:
    echo "[review] ERROR: ${result.result}"
    mcp__goyoke-interactive__prepare_skill({ skill: "review", release: true })
# Non-TUI fallback: Bash({ command: "goyoke-skill-guard --release" })
    exit 1
background_pid = result.background_pid
echo "[review] Team launched (PID $background_pid)"
echo "[review] Use /team-status to track progress"
echo "[review] Use /team-result when complete to see findings"
```

#### Step 5: Remove Skill Guard

```bash
mcp__goyoke-interactive__prepare_skill({ skill: "review", release: true })
# Non-TUI fallback: Bash({ command: "goyoke-skill-guard --release" })
```

#### Step 6: Return to User

Output summary and return immediately:

```
[review] Review team launched in background
  Reviewers: {reviewer-list}
  Files: {file-count} files across {language-count} languages
  Team: {team_dir}
  PID: {background_pid}

Use /team-status to check progress
Use /team-result to view findings when complete
```

TUI returns to user within ~5 seconds.

---

### Phase 4: Report Generation

Results come from `/team-result` after the team completes. The review skill returns immediately after Phase 3 Step 6.

---

### Telemetry Logging

After collecting review results, log to ML telemetry (non-blocking):

```bash
# Check if telemetry is enabled (default: enabled)
if [[ "${GOYOKE_ENABLE_TELEMETRY:-1}" == "1" ]]; then
    # Extract session_id from context or generate one
    session_id="${CLAUDE_SESSION_ID:-$(uuidgen 2>/dev/null || cat /proc/sys/kernel/random/uuid)}"

    # Build telemetry input from review result
    review_scope="${review_scope:-staged}"
    files_count=$(echo "$files" | wc -l | tr -d ' ')

    # Pipe to goyoke-log-review (non-blocking, errors to /dev/null)
    echo "$review_result" | jq --arg sid "$session_id" --arg scope "$review_scope" --argjson files "$files_count" \
        '{session_id: $sid, review_scope: $scope, files_reviewed: $files, findings: .findings}' \
        | goyoke-log-review > .claude/tmp/review-telemetry.json 2>/dev/null || true
fi
```

**Note:** Telemetry logging is non-blocking and fails silently. Review skill continues regardless of telemetry success.

---

## Reviewer Specializations

### backend-reviewer

**Focus areas:**

- API design and contracts
- Data layer patterns
- Error handling
- Concurrency safety
- Resource management

**Languages:** Go, Python, backend TypeScript

**Severity mapping:**

- Critical: SQL injection, race conditions, resource leaks
- Warning: Inefficient algorithms, missing error checks
- Info: Style preferences, optimization opportunities

### frontend-reviewer

**Focus areas:**

- Component architecture
- State management
- Hook usage patterns
- Performance (memo, callbacks)
- Accessibility

**Languages:** TypeScript, React, Ink

**Severity mapping:**

- Critical: XSS vulnerabilities, infinite loops, memory leaks
- Warning: Missing memoization, prop drilling, missing keys
- Info: Component naming, file organization

### standards-reviewer

**Focus areas:**

- Naming conventions
- Code organization
- Documentation
- Test coverage
- Consistency with codebase patterns

**Languages:** All

**Severity mapping:**

- Critical: None (standards reviewer never blocks)
- Warning: Convention violations, missing docs
- Info: Style suggestions, minor improvements

### architect-reviewer

**Focus areas:**

- Module boundaries and cohesion
- Dependency health (circular deps, coupling)
- Design patterns (god objects, leaky abstractions)
- Change impact and testability
- Structural anti-patterns

**Languages:** All

**Severity mapping:**

- Critical: Circular dependencies, god modules, leaky abstractions
- Warning: High fan-out, tight coupling, missing abstractions
- Info: Interface extraction opportunities, testability improvements

---

## Cost Model

| Phase                         | Model       | Est. Tokens | Cost            |
| ----------------------------- | ----------- | ----------- | --------------- |
| Detection                     | Bash        | 0           | $0.000          |
| Classification                | Bash        | 0           | $0.000          |
| Config generation             | Router      | ~2K         | $0.000 (inline) |
| Backend Reviewer              | Sonnet      | 20-40K      | $0.50-$1.00     |
| Frontend Reviewer             | Sonnet      | 20-40K      | $0.50-$1.00     |
| Standards Reviewer            | Sonnet      | 20-40K      | $0.50-$1.00     |
| Architect Reviewer            | Sonnet      | 30-50K      | $0.80-$1.20     |
| **Total (4 reviewers)**       |             | 90-170K     | **$2.30-$4.20** |

**Cost per file reviewed:** ~$0.15-$0.30

**Note:** Reviewers upgraded from Haiku to Sonnet (2026-02-25) to ensure actual file reading.
Haiku reviewers were generating hallucinated findings without using Read tools.

**Parallelization savings:** ~40% faster than sequential review

---

## Integration with /ticket

The /review skill integrates with /ticket workflow as Phase 7.6 (blocking):

```bash
# In ticket workflow after audit
review_enabled=$(jq -r '.audit_config.code_review.enabled // false' "$config_file")

if [[ "$review_enabled" == "true" ]]; then
    echo "[ticket] Running code review..."

    # Get changed files for this ticket
    changed_files=$(git diff --name-only HEAD~1)

    # Run review
    /review --scope="$changed_files"

    if [[ $? -ne 0 ]]; then
        echo "[ticket] Code review FAILED - critical issues found"
        echo "[ticket] Fix issues before completing ticket"
        exit 1
    fi

    echo "[ticket] Code review passed"
fi
```

**Configuration example:**

```json
{
  "tickets_dir": "tickets/",
  "project_name": "my-project",
  "audit_config": {
    "enabled": true,
    "code_review": {
      "enabled": true,
      "block_on_critical": true
    }
  }
}
```

---

## State Files

| File                             | Purpose                    | Format                              |
| -------------------------------- | -------------------------- | ----------------------------------- |
| `{team_dir}/config.json`        | Team execution config      | JSON with waves, members, budget    |
| `{team_dir}/stdin_*.json`       | Per-reviewer input         | JSON per reviewer schema            |
| `{team_dir}/stdout_*.json`      | Per-reviewer output        | JSON with findings                  |
| `{team_dir}/runner.log`         | Execution log              | Plain text                          |

`{team_dir}` = `{goyoke_session_dir}/teams/{timestamp}.code-review/` (created by `goyoke-skill-guard` hook)

---

## Troubleshooting

**"No files to review"**

- Check git status - are there staged changes?
- Use `--all` to include unstaged changes
- Use `--scope=<glob>` to specify files explicitly

**"Reviewer not found"**

- Ensure agents-index.json includes reviewer agents
- Check routing-schema.json has correct mappings

**"Review failed with no findings"**

- Check reviewer output in task logs
- Verify reviewers have access to files
- Ensure file paths are absolute

**"False positives in review"**

- Review is advisory - human judgment still required
- Use findings as guidance, not absolute truth
- Consider adding project-specific review guidelines

**"Team launch failed"**

- Check `$team_dir/runner.log` for errors
- Verify `goyoke-team-run` is built and in PATH
- Check `$team_dir/config.json` is valid JSON: `jq . "$team_dir/config.json"`

---

## Example Session

```bash
$ git status
On branch feature/new-api
Changes to be committed:
  modified:   internal/api/handler.go
  modified:   internal/models/user.go
  new file:   internal/api/handler_test.go

$ /review

[review] Found 3 files to review
[review] Selected reviewers: backend-reviewer standards-reviewer
[review] Using background team-run dispatch

[review] Review team launched in background
  Reviewers: backend-reviewer standards-reviewer
  Files: 3 files across 1 languages
  Team: /home/user/project/.goyoke/sessions/20260208.a1b2c3d4/teams/1738901234.code-review
  PID: 12345

Use /team-status to check progress
Use /team-result to view findings when complete

$ /team-status
[team-status] Team: code-review
Status: running
Started: 5 seconds ago
Progress: 1/2 complete

Wave 1:
  backend-reviewer (completed 3s ago)
  standards-reviewer (running, 2s elapsed)

$ /team-result
[team-result] Team: code-review (completed 8 seconds ago)

Status: WARNING
Critical: 0
Warnings: 2
Info: 1

WARNINGS:
  [internal/api/handler.go:45] Missing error check on database query
    Add error handling: if err != nil { return err }

  [internal/models/user.go:23] Exported field without documentation
    Add godoc comment for Email field

INFO:
  [internal/api/handler_test.go:12] Test table format recommended
    Consider using table-driven test pattern
```

---

## ML Telemetry

This skill logs all findings to ML telemetry for downstream analysis:

| File                                          | Purpose                                |
| --------------------------------------------- | -------------------------------------- |
| `$XDG_DATA_HOME/goyoke/review-findings.jsonl` | All review findings                    |
| `$XDG_DATA_HOME/goyoke/sharp-edge-hits.jsonl` | Sharp edge correlations                |
| `.claude/tmp/review-telemetry.json`           | Session telemetry output (finding IDs) |

Telemetry is non-blocking - skill continues even if logging fails.

### Disabling Telemetry

Set `GOYOKE_ENABLE_TELEMETRY=0` to disable telemetry logging:

```bash
GOYOKE_ENABLE_TELEMETRY=0 /review
```

### Telemetry Schema

Each review session logs:

- **session_id**: Unique session identifier
- **review_scope**: staged | all | glob | explicit
- **files_reviewed**: Number of files reviewed
- **findings**: Array of findings with severity, file, line, reviewer, message

This data feeds ML models for:

- Pattern recognition (which findings appear most often)
- Sharp edge detection (which code patterns trigger critical findings)
- Review effectiveness (do findings prevent bugs)
- Reviewer accuracy (false positive rates)

---

**Skill Version**: 3.0
**Last Updated**: 2026-02-10
**Maintained By**: System

## Not For
- Bioinformatics pipeline review (use /review-bioinformatics instead)
- Implementation plan review (use /review-plan instead)
- Quick spot-checks on a single file (use code-reviewer agent directly)
- Implementing fixes for review findings (use /implement or /ticket)
