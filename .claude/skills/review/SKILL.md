---
name: review
description: Orchestrated multi-domain code review with severity-grouped findings and approval status
---

# Review Skill v2.0

## Purpose

Automated code review through coordinated specialist reviewers. Analyzes changed files, identifies relevant review domains, spawns reviewers via background team-run (default) or foreground orchestrator (fallback), and synthesizes findings into actionable report.

**What this skill does:**

1. **Detect** — Find changed files via git diff or specified scope
2. **Classify** — Identify languages and architectural layers present
3. **Select** — Choose relevant reviewers (backend, frontend, standards, architecture)
4. **Execute** — Dispatch reviewers via background team-run (default) or foreground orchestrator (fallback)
5. **Launch** — Start `gogent-team-run` in background, return immediately
6. **Synthesize** — Collect and group findings by severity
7. **Report** — Generate unified report with approval status

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

**Project setup:**
None required. Works in any git repository.

---

## Workflow

### Phase 1: Change Detection

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

### Phase 2.5: Dispatch Path Selection

Determine whether to use background team-run or foreground orchestrator:

```bash
# Check feature flag (default: true = use team-run)
use_team=$(jq -r '.use_team_pattern // true' ~/.claude/settings.json 2>/dev/null)

# Fallback if binary not available
if [[ "$use_team" == "true" ]] && ! command -v gogent-team-run &>/dev/null; then
    echo "[review] gogent-team-run not found, using foreground path"
    use_team="false"
fi

if [[ "$use_team" == "true" ]]; then
    echo "[review] Using background team-run dispatch"
    # Continue to Phase 3B
else
    echo "[review] Using foreground orchestrator dispatch"
    # Continue to Phase 3A
fi
```

**Feature flag:** `use_team_pattern` in `~/.claude/settings.json` (default: `true`).

**Fallback triggers:**
- `use_team_pattern: false` in settings.json
- `gogent-team-run` binary not found in PATH

---

### Phase 3A: Foreground Dispatch (Fallback)

**When:** `use_team == "false"` (feature flag off or binary not found)

This is the original foreground dispatch path. The router spawns review-orchestrator via Task, which coordinates reviewers using mcp__gofortress__spawn_agent. This path blocks the TUI for ~2-3 minutes.

Delegate to review-orchestrator:

**Task Invocation:**

```
Tool: Task
Description: Coordinate multi-domain code review
Subagent Type: Plan
Model: sonnet
Prompt:
  AGENT: review-orchestrator

  1. TASK: Coordinate code review for changed files

  2. FILES TO REVIEW:
     {files}

  3. REVIEWERS TO SPAWN:
     {reviewers}

  4. EXPECTED OUTCOME:
     - Spawn reviewers in parallel via mcp__gofortress__spawn_agent (Task() is blocked for sub-agents)
     - Each reviewer examines files in their domain
     - Collect findings from all reviewers
     - Synthesize into unified report
     - Assign approval status: APPROVED / WARNING / BLOCKED

  5. REQUIRED SKILLS:
     - Multi-agent coordination
     - Finding synthesis
     - Severity classification

  6. MUST DO:
     - Spawn each reviewer with relevant file subset
     - Wait for all reviewers to complete
     - Group findings by severity (critical, warning, info)
     - Ensure file:line references in all findings
     - Assign approval status based on severity

  7. MUST NOT DO:
     - Implement fixes (report only)
     - Skip critical issues
     - Generate vague findings without locations

  8. APPROVAL CRITERIA:
     - APPROVED: No critical or warning issues
     - WARNING: Warnings present, no critical issues
     - BLOCKED: Critical issues present

  9. OUTPUT FORMAT:
     {
       "status": "APPROVED|WARNING|BLOCKED",
       "summary": {
         "critical": N,
         "warnings": M,
         "info": K
       },
       "findings": [
         {
           "severity": "critical|warning|info",
           "file": "path/to/file",
           "line": N,
           "reviewer": "backend-reviewer",
           "message": "Description",
           "recommendation": "Fix suggestion"
         }
       ]
     }
```

---

### Phase 3B: Background Team-Run Dispatch (Default)

**When:** `use_team == "true"` (default)

This path generates config.json + stdin files directly, launches `gogent-team-run`, and returns immediately. No review-orchestrator LLM agent is involved. Results are retrieved later via `/team-result`.

#### Step 1: Create Team Directory

```bash
session_dir="${GOGENT_SESSION_DIR:-$HOME/.claude/sessions/$(date +%Y%m%d).$(uuidgen | cut -d- -f1)}"
team_dir="${session_dir}/teams/$(date +%s).code-review"
mkdir -p "$team_dir"
```

#### Step 2: Generate config.json

Read the review template from `.claude/schemas/teams/review.json` and populate with dynamic values. Only include selected reviewers in `waves[0].members[]`.

**Template fields to populate:**
- `team_name`: `"review-$(date +%Y%m%d-%H%M%S)"`
- `workflow_type`: `"review"`
- `project_root`: `$(git rev-parse --show-toplevel)`
- `session_id`: basename of `$GOGENT_SESSION_DIR`
- `created_at`: `$(date -u +%Y-%m-%dT%H:%M:%SZ)`
- `budget_max_usd`: `2.0`
- `budget_remaining_usd`: `2.0`
- `warning_threshold_usd`: `1.6`
- `status`: `"pending"`
- `background_pid`: `null`
- `started_at`: `null`
- `completed_at`: `null`

**Per-member fields** (from review.json template, filtered to selected reviewers):
- `model`: `"haiku"` for backend/frontend/standards, `"sonnet"` for architect
- `timeout_ms`: `120000` for haiku reviewers, `300000` for architect
- `max_retries`: `2`
- `stdin_file`: `"stdin_{reviewer-name}.json"`
- `stdout_file`: `"stdout_{reviewer-name}.json"`
- All runtime fields: `null`/`0`/`""`/`"pending"`

Write to `$team_dir/config.json`.

#### Step 3: Generate Stdin Files

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

#### Step 4: Launch

```bash
gogent-team-run "$team_dir"
```

No output redirection. No config.json path argument. The binary handles:
- PID file creation
- Log redirection to `runner.log`
- Session leadership (setsid)
- Writing `background_pid` to config.json

#### Step 5: Verify Launch

```bash
sleep 2
background_pid=$(jq -r '.background_pid' "$team_dir/config.json")
if [[ -z "$background_pid" || "$background_pid" == "null" ]]; then
    echo "[review] ERROR: Team launch failed. Check $team_dir/runner.log"
    echo "[review] Falling back to foreground path..."
    # Fall through to Phase 3A
else
    echo "[review] Team launched (PID $background_pid)"
    echo "[review] Use /team-status to track progress"
    echo "[review] Use /team-result when complete to see findings"
fi
```

#### Step 6: Return to User

For background path, output summary and return immediately:

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

**For foreground path (Phase 3A):** Display results inline as shown below.

**For background path (Phase 3B):** Phase 4 is skipped — results come from `/team-result` after the team completes. The review skill returns immediately after Phase 3B Step 6.

#### Foreground Report (Phase 3A only)

After orchestrator completes, display results:

```bash
review_result=$(cat .claude/tmp/review-result.json)
status=$(echo "$review_result" | jq -r '.status')
critical=$(echo "$review_result" | jq -r '.summary.critical')
warnings=$(echo "$review_result" | jq -r '.summary.warnings')
info=$(echo "$review_result" | jq -r '.summary.info')

echo ""
echo "╔══════════════════════════════════════╗"
echo "║        CODE REVIEW REPORT             ║"
echo "╚══════════════════════════════════════╝"
echo ""
echo "Status: $status"
echo "Critical: $critical"
echo "Warnings: $warnings"
echo "Info: $info"
echo ""

if [[ "$critical" -gt 0 ]]; then
    echo "═══ CRITICAL ISSUES ═══"
    echo "$review_result" | jq -r '.findings[] | select(.severity == "critical") | "  [\(.file):\(.line)] \(.message)\n    → \(.recommendation)"'
    echo ""
fi

if [[ "$warnings" -gt 0 ]]; then
    echo "═══ WARNINGS ═══"
    echo "$review_result" | jq -r '.findings[] | select(.severity == "warning") | "  [\(.file):\(.line)] \(.message)\n    → \(.recommendation)"'
    echo ""
fi

if [[ "$info" -gt 0 ]]; then
    echo "═══ INFO ═══"
    echo "$review_result" | jq -r '.findings[] | select(.severity == "info") | "  [\(.file):\(.line)] \(.message)"'
    echo ""
fi

# Exit code matches approval status
case "$status" in
    APPROVED) exit 0 ;;
    WARNING) exit 0 ;;
    BLOCKED) exit 1 ;;
esac
```

---

### Telemetry Logging

After collecting review results, log to ML telemetry (non-blocking):

```bash
# Check if telemetry is enabled (default: enabled)
if [[ "${GOGENT_ENABLE_TELEMETRY:-1}" == "1" ]]; then
    # Extract session_id from context or generate one
    session_id="${CLAUDE_SESSION_ID:-$(uuidgen 2>/dev/null || cat /proc/sys/kernel/random/uuid)}"

    # Build telemetry input from review result
    review_scope="${review_scope:-staged}"
    files_count=$(echo "$files" | wc -l | tr -d ' ')

    # Pipe to gogent-log-review (non-blocking, errors to /dev/null)
    echo "$review_result" | jq --arg sid "$session_id" --arg scope "$review_scope" --argjson files "$files_count" \
        '{session_id: $sid, review_scope: $scope, files_reviewed: $files, findings: .findings}' \
        | gogent-log-review > .claude/tmp/review-telemetry.json 2>/dev/null || true
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
| **Background Path (team-run)**|             |             |                 |
| Config generation             | Router      | ~2K         | $0.000 (inline) |
| Backend Reviewer              | Haiku       | 3-5K        | $0.003-$0.005   |
| Frontend Reviewer             | Haiku       | 3-5K        | $0.003-$0.005   |
| Standards Reviewer            | Haiku       | 3-5K        | $0.003-$0.005   |
| Architect Reviewer            | Sonnet      | 8-12K       | $0.07-$0.11     |
| **Background Total (4 rev.)** |             | 17-27K      | **$0.08-$0.13** |
| **Foreground Path (fallback)**|             |             |                 |
| Orchestrator                  | Sonnet      | 8-12K       | $0.07-$0.11     |
| Backend Reviewer              | Haiku+Think | 3-5K        | $0.003-$0.005   |
| Frontend Reviewer             | Haiku+Think | 3-5K        | $0.003-$0.005   |
| Standards Reviewer            | Haiku+Think | 3-5K        | $0.003-$0.005   |
| Architect Reviewer            | Sonnet      | 8-12K       | $0.07-$0.11     |
| **Foreground Total (4 rev.)** |             | 28-42K      | **$0.15-$0.24** |

**Cost per file reviewed:** ~$0.03-$0.06

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
        echo "[ticket] ❌ Code review FAILED - critical issues found"
        echo "[ticket] Fix issues before completing ticket"
        exit 1
    fi

    echo "[ticket] ✓ Code review passed"
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
| `.claude/tmp/review-result.json` | Review findings and status | JSON with status, summary, findings |
| `.claude/tmp/review-scope.txt`   | Files under review         | Line-separated file paths           |

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
[review] Selected reviewers: backend-reviewer standards-reviewer architect-reviewer
[ROUTING] → review-orchestrator (multi-domain code review)

[review-orchestrator spawns reviewers...]
[backend-reviewer analyzing internal/api/handler.go...]
[standards-reviewer analyzing all files...]
[architect-reviewer analyzing structural patterns...]

[All reviewers complete]

╔══════════════════════════════════════╗
║        CODE REVIEW REPORT             ║
╚══════════════════════════════════════╝

Status: WARNING
Critical: 0
Warnings: 2
Info: 1

═══ WARNINGS ═══
  [internal/api/handler.go:45] Missing error check on database query
    → Add error handling: if err != nil { return err }

  [internal/models/user.go:23] Exported field without documentation
    → Add godoc comment for Email field

═══ INFO ═══
  [internal/api/handler_test.go:12] Test table format recommended
    → Consider using table-driven test pattern

$ # Address warnings and re-review
$ /review
[review] Status: APPROVED ✓
```

**Background path example:**

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
  Team: /home/user/.claude/sessions/20260208.a1b2c3d4/teams/1738901234.code-review
  PID: 12345

Use /team-status to check progress
Use /team-result to view findings when complete

$ /team-status
[team-status] Team: code-review
Status: running
Started: 5 seconds ago
Progress: 1/2 complete

Wave 1:
  ✓ backend-reviewer (completed 3s ago)
  ⟳ standards-reviewer (running, 2s elapsed)

$ /team-result
[team-result] Team: code-review (completed 8 seconds ago)

╔══════════════════════════════════════╗
║        CODE REVIEW REPORT             ║
╚══════════════════════════════════════╝

Status: WARNING
Critical: 0
Warnings: 2
Info: 1

═══ WARNINGS ═══
  [internal/api/handler.go:45] Missing error check on database query
    → Add error handling: if err != nil { return err }

  [internal/models/user.go:23] Exported field without documentation
    → Add godoc comment for Email field

═══ INFO ═══
  [internal/api/handler_test.go:12] Test table format recommended
    → Consider using table-driven test pattern
```

---

## ML Telemetry

This skill logs all findings to ML telemetry for downstream analysis:

| File                                          | Purpose                                |
| --------------------------------------------- | -------------------------------------- |
| `$XDG_DATA_HOME/gogent/review-findings.jsonl` | All review findings                    |
| `$XDG_DATA_HOME/gogent/sharp-edge-hits.jsonl` | Sharp edge correlations                |
| `.claude/tmp/review-telemetry.json`           | Session telemetry output (finding IDs) |

Telemetry is non-blocking - skill continues even if logging fails.

### Disabling Telemetry

Set `GOGENT_ENABLE_TELEMETRY=0` to disable telemetry logging:

```bash
GOGENT_ENABLE_TELEMETRY=0 /review
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

**Skill Version**: 2.0
**Last Updated**: 2026-02-08
**Maintained By**: System
