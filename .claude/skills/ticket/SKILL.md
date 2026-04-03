---
name: ticket
description: Ticket-driven workflow for structured implementation. Finds next ticket, validates readiness, spawns architect if needed, tracks progress.
---

# Ticket Skill v1.1

## Purpose

Turn ticket specifications into completed work through systematic workflow execution.

**What this skill does:**

1. **Discover** — Find ticket system in current project
2. **Select** — Choose next actionable ticket (dependencies met)
3. **Validate** — Check ticket structure and completeness
4. **Plan** — Delegate to architect if ticket needs implementation planning
5. **Track** — Monitor progress with TaskCreate/TaskUpdate and acceptance criteria
6. **Verify** — Ensure all acceptance criteria met before completion
7. **Audit** — Optionally run automated tests and generate documentation
8. **Complete** — Update status, generate commit, mark done

**What this skill does NOT do:**

- Implement code (delegates to language agents: go-pro, python-pro, etc.)
- Enforce routing rules (handled by validate-routing.sh hook)
- Create tickets (use project planning workflows)

---

## Invocation

- `/ticket` or `/ticket next` — Find and start next available ticket
- `/ticket status` — Show current ticket progress
- `/ticket verify` — Check if acceptance criteria met
- `/ticket complete` — Mark ticket done and update status

---

## Prerequisites

**Required tools:**

- `jq` (JSON manipulation)
- `python3` with `python-frontmatter` library
- `git` (for commit workflow)

**Install dependencies:**

```bash
pip install python-frontmatter
```

**Project setup:**
Either create `.ticket-config.json` at project root:

```json
{
  "tickets_dir": "implementation_plan/tickets",
  "project_name": "my-project"
}
```

Or use standard directory structure:

- `./implementation_plan/tickets/tickets-index.json`
- `./migration_plan/finalised/tickets/tickets-index.json`
- `./tickets/tickets-index.json`

---

## Workflow

### Phase 1: Project Discovery

Locate the ticket directory for this project:

```bash
tickets_dir=$(~/.claude/skills/ticket/scripts/discover-project.sh)
if [[ $? -ne 0 ]]; then
    echo "[ticket] ERROR: No ticket directory found."
    echo "[ticket] Create .ticket-config.json or tickets/ directory."
    exit 1
fi

echo "[ticket] Found tickets at: $tickets_dir"
```

**Discovery logic:**

1. Search for `.ticket-config.json` in current directory and ancestors
2. Fallback to git root + standard paths (implementation_plan/tickets/, migration_plan/finalised/tickets/, tickets/)
3. Error if nothing found

---

### Phase 2: Ticket Selection

Find the next actionable ticket (status=pending, dependencies met):

```bash
tickets_index="$tickets_dir/tickets-index.json"

if [[ ! -f "$tickets_index" ]]; then
    echo "[ticket] ERROR: tickets-index.json not found at $tickets_dir"
    exit 1
fi

ticket_id=$(~/.claude/skills/ticket/scripts/find-next-ticket.sh "$tickets_index")

if [[ -z "$ticket_id" ]]; then
    echo "[ticket] No actionable tickets found."
    echo "[ticket] Check if all dependencies are completed."
    exit 0
fi

echo "[ticket] Selected: $ticket_id"
```

**Selection logic:**

- Filter to `status == "pending"`
- Check all dependencies have `status == "completed"`
- Return first match (respects ticket order in JSON)

---

### Phase 3: Schema Validation

Validate ticket structure before proceeding:

```bash
# Find ticket file (check both .md in tickets_dir and subdirectories)
ticket_file=""
if [[ -f "$tickets_dir/$ticket_id.md" ]]; then
    ticket_file="$tickets_dir/$ticket_id.md"
elif [[ -f "$tickets_dir"/*"$ticket_id"*.md ]]; then
    ticket_file=$(find "$tickets_dir" -name "*$ticket_id*.md" | head -1)
else
    echo "[ticket] ERROR: Ticket file not found for $ticket_id"
    exit 1
fi

# Run validation
validation_result=$(~/.claude/skills/ticket/scripts/validate-ticket-schema.py \
    "$ticket_file" \
    "$tickets_index" 2>&1)

if [[ $? -ne 0 ]]; then
    echo "[ticket] Schema validation FAILED:"
    echo "$validation_result" | jq -r '.errors[]' 2>/dev/null || echo "$validation_result"
    exit 1
fi

# Show warnings if any
warnings=$(echo "$validation_result" | jq -r '.warnings[]' 2>/dev/null)
if [[ -n "$warnings" ]]; then
    echo "[ticket] Warnings:"
    echo "$warnings"
fi

echo "[ticket] Schema validation: PASS"
```

**Validates:**

- Required frontmatter fields (id, title, description, status, time_estimate, dependencies)
- Status enum values
- Acceptance criteria presence
- Dependency references (if provided)

---

### Phase 4: Planning Decision

Determine if ticket needs architect planning:

```bash
planning_check=$(~/.claude/skills/ticket/scripts/check-planning-needed.py "$ticket_file")
needs_planning=$(echo "$planning_check" | jq -r '.needs_planning')
reason=$(echo "$planning_check" | jq -r '.reason')
confidence=$(echo "$planning_check" | jq -r '.confidence')

echo "[ticket] Planning needed: $needs_planning"
echo "[ticket] Reason: $reason"
echo "[ticket] Confidence: $confidence"
```

**Decision logic (priority order):**

1. Explicit `needs_planning` field in frontmatter
2. "planning" tag presence
3. Complexity heuristic (files>3, time>2h, deps>2, multi-package)
4. Default: false (safe default)

---

### Phase 5: Planning Phase (Conditional)

If `needs_planning == true`, delegate to architect:

Use the Task tool to spawn architect agent:

**Task Invocation:**

```
Tool: Task
Description: Create implementation plan for {ticket_id}
Subagent Type: Plan
Model: sonnet
Prompt:
  AGENT: architect

  1. TASK: Review ticket {ticket_id} and create detailed implementation plan

  2. EXPECTED OUTCOME:
     - File-by-file implementation specifications
     - Dependency analysis
     - Risk assessment
     - Testing strategy

  3. REQUIRED SKILLS: System architecture, domain patterns

  4. REQUIRED TOOLS: Read, Write

  5. MUST DO:
     - Read ticket file: {ticket_file}
     - Read tickets-index.json for context
     - Create plan document
     - Identify blockers and dependencies

  6. MUST NOT DO:
     - Implement code (planning only)
     - Skip risk analysis

  7. CONTEXT:
     Ticket: {ticket_id}
     File: {ticket_file}
     Project: {project_name from config}
```

After architect completes, it produces:
- `specs.md` — human-readable plan for review
- `implementation-plan.json` — machine-readable plan for team-run automation

```
[ticket] Plan created.
Review and approve before proceeding.
```

If `needs_planning == false`, skip to Phase 6.

---

### Phase 5.5: Team-Run Dispatch (Conditional)

If the architect produced `implementation-plan.json` AND the `use_team_pattern` feature flag is enabled, dispatch via background team-run instead of foreground sequential execution.

**Decision logic:**

```bash
# Check feature flag
use_team=$(jq -r '.use_team_pattern // true' ~/.claude/settings.json 2>/dev/null)

# Check if implementation-plan.json exists
plan_file=".claude/tmp/implementation-plan.json"

if [[ "$use_team" == "true" ]] && [[ -f "$plan_file" ]]; then
    echo "[ticket] Team-run dispatch: implementation-plan.json found"

    # Determine team directory
    session_dir="${GOGENT_SESSION_DIR:-$(cat .claude/current-session 2>/dev/null)}"
    session_dir="${session_dir:-.claude/sessions/$(date +%Y%m%d-%H%M%S)}"
    team_dir="$session_dir/teams/$(date +%s).implementation"

    # Generate config + stdin files
    gogent-plan-impl \
        --plan="$plan_file" \
        --project-root="$(pwd)" \
        --output="$team_dir"

    if [[ $? -ne 0 ]]; then
        echo "[ticket] ERROR: gogent-plan-impl failed. Falling back to foreground."
        # Fall through to Phase 6 (foreground)
    else
        # Launch background execution
        result = mcp__gofortress-interactive__team_run({
            team_dir: "$team_dir",
            wait_for_start: true,
            timeout_ms: 10000
        })
        if result.success:
            background_pid = result.background_pid
            echo "[ticket] Team launched (PID $background_pid)"
            echo "[ticket] Monitor with: /team-status"
            echo "[ticket] Results with: /team-result"
            # EXIT — TUI returns to user, background handles execution
            exit 0
        else:
            echo "[ticket] ERROR: ${result.result}. Falling back to foreground."
    fi
fi

# If team-run not used, fall through to Phase 6 (foreground tracking)
```

**When team-run is used:**
- TUI returns to user in <15 seconds
- `gogent-team-run` executes waves in the background
- Tasks within a wave run in parallel
- Wave failure propagation: if any task in Wave N fails, Wave N+1 members get status "skipped"
- Monitor via `/team-status`, collect results via `/team-result`

**When team-run is NOT used (fallback):**
- `use_team_pattern: false` in settings.json
- `implementation-plan.json` not produced by architect
- `gogent-plan-impl` binary not found or fails
- Single-task tickets (no specs.md / no multi-task plan)

---

### Phase 6: Implementation Tracking (Foreground Fallback)

Update ticket status and create tasks for tracking:

```bash
# Update status to in_progress
~/.claude/skills/ticket/scripts/update-ticket-status.sh \
    "$tickets_index" \
    "$ticket_id" \
    "in_progress"

echo "[ticket] Status updated: in_progress"

# Save current ticket to state file
echo "$ticket_id" > "$tickets_dir/.current-ticket"
```

**Create tasks from acceptance criteria:**

Parse the ticket file and extract acceptance criteria checkboxes, then create tasks.

For each criterion, use TaskCreate:

```javascript
TaskCreate({
  subject: "[Criterion text]",
  description: "From ticket {ticket_id}: [full criterion details]",
  activeForm: "Working on [criterion]...",
});
```

Example output:

```
[ticket] Created 3 tasks from acceptance criteria:
  - Task #1: Implement function X
  - Task #2: Write tests for Y
  - Task #3: Update documentation
```

---

### Phase 7: Completion Verification

After implementation work, verify acceptance criteria:

```bash
acceptance=$(~/.claude/skills/ticket/scripts/verify-acceptance.py "$ticket_file")
all_complete=$(echo "$acceptance" | jq -r '.all_complete')
completed=$(echo "$acceptance" | jq -r '.completed')
total=$(echo "$acceptance" | jq -r '.total')

echo "[ticket] Acceptance criteria: $completed/$total complete"

if [[ "$all_complete" != "true" ]]; then
    pending=$(echo "$acceptance" | jq -r '.pending[]')
    echo "[ticket] Pending criteria:"
    echo "$pending" | sed 's/^/  - /'
    echo ""
    echo "[ticket] Continue working or mark complete anyway?"
    exit 0
fi

echo "[ticket] All acceptance criteria met ✓"
```

---

### Phase 7.5: Audit Documentation (Optional)

After acceptance criteria verification, optionally run automated audit tests and documentation:

```bash
# Check if audit is enabled in config
config_file=$(find . -name ".ticket-config.json" -maxdepth 3 | head -1)
audit_enabled="false"

if [[ -f "$config_file" ]]; then
    audit_enabled=$(jq -r '.audit_config.enabled // false' "$config_file")
fi

if [[ "$audit_enabled" == "true" ]]; then
    echo "[ticket] Running audit documentation..."
    if ~/.claude/skills/ticket/scripts/run-audit.sh --ticket-id="$ticket_id"; then
        echo "[ticket] ✓ Audit complete"
    else
        echo "[ticket] ⚠️ Audit failed (non-blocking, continuing)"
    fi
fi
```

**What the audit does:**

- Executes language-specific test suites (unit, integration, race detection)
- Generates coverage reports
- Creates implementation summary document
- Logs all results to `.ticket-audits/{ticket_id}/`

**Output artifacts:**

- `unit-tests.log` — Unit test execution results
- `coverage.out` — Coverage data (Go projects)
- `coverage-report.txt` — Coverage analysis
- `coverage-summary.txt` — Total coverage percentage
- `implementation-summary.md` — Human-readable summary with test results and metadata

**Behavior notes:**

- **Non-blocking**: Audit failures do NOT prevent ticket completion
- **Backward compatible**: If `.ticket-config.json` missing or `audit_config.enabled` is false, audit is silently skipped
- **Language-agnostic**: Automatically detects project language (Go, Python, R, JavaScript/TypeScript)
- **Configurable**: Test commands can be customized in `.ticket-config.json` under `audit_config.test_commands`

**Example audit configuration:**

```json
{
  "tickets_dir": "migration_plan/tickets",
  "project_name": "GOgent-Fortress",
  "audit_config": {
    "enabled": true,
    "test_commands": {
      "go": {
        "unit": "go test -v ./...",
        "race": "go test -race ./...",
        "coverage": "go test -coverprofile={audit_dir}/coverage.out ./..."
      }
    }
  }
}
```

**When to enable audit:**

- Projects with established test suites
- Tickets requiring coverage verification
- Teams wanting automated documentation of implementation quality

**When to skip audit:**

- Prototyping or spike work
- Documentation-only changes
- Projects without test infrastructure

---

### Phase 7.6: Code Review (Blocking)

After audit, run code review if enabled:

```bash
# Check if code review is enabled
config_file=$(find . -name ".ticket-config.json" -maxdepth 3 | head -1)
review_enabled="false"

if [[ -f "$config_file" ]]; then
    review_enabled=$(jq -r '.audit_config.code_review.enabled // false' "$config_file")
fi

if [[ "$review_enabled" == "true" ]]; then
    echo "[ticket] Running code review..."

    # Get changed files for this ticket
    changed_files=$(git diff --name-only HEAD~1)

    # Write changed files to temp file for review-orchestrator
    echo "$changed_files" > .claude/tmp/review-scope.txt

    # Invoke review-orchestrator via Task
    # Task invocation with review-orchestrator agent
    # The orchestrator will:
    # 1. Read files from review-scope.txt
    # 2. Classify languages and select reviewers
    # 3. Spawn reviewers in parallel
    # 4. Synthesize findings to .claude/tmp/review-result.json

    # Wait for review to complete and read results
    review_status=$(jq -r '.status' .claude/tmp/review-result.json)
    critical_count=$(jq -r '.summary.critical' .claude/tmp/review-result.json)
    warning_count=$(jq -r '.summary.warnings' .claude/tmp/review-result.json)

    if [[ "$review_status" == "BLOCKED" ]]; then
        echo "[ticket] ❌ Code review FAILED - critical issues found"
        echo "[ticket] Critical issues: $critical_count"

        # Display critical findings
        jq -r '.findings[] | select(.severity == "critical") |
               "  [\(.file):\(.line)] \(.message)\n    → \(.recommendation)"' \
               .claude/tmp/review-result.json

        echo "[ticket] Fix issues before completing ticket"
        exit 1
    elif [[ "$review_status" == "WARNING" ]]; then
        echo "[ticket] ⚠️ Code review passed with warnings"
        echo "[ticket] Warnings: $warning_count"

        # Display warnings but continue
        jq -r '.findings[] | select(.severity == "warning") |
               "  [\(.file):\(.line)] \(.message)"' \
               .claude/tmp/review-result.json
    else
        echo "[ticket] ✓ Code review passed (APPROVED)"
    fi

    # Store finding IDs for outcome tracking (if telemetry enabled)
    if [[ "${GOGENT_ENABLE_TELEMETRY:-1}" == "1" ]] && [[ -f .claude/tmp/review-telemetry.json ]]; then
        jq -r '.finding_ids[]' .claude/tmp/review-telemetry.json > "$tickets_dir/.review-findings-$ticket_id" 2>/dev/null || true
    fi
fi
```

**Code review behavior:**

- **Fully blocking**: Critical issues prevent ticket completion (exit 1)
- **Warnings allowed**: Warnings are displayed but don't block
- **Controlled by**: `audit_config.code_review.enabled` in `.ticket-config.json`
- **Reviewers selected**: Based on file types (backend-reviewer, frontend-reviewer, standards-reviewer)
- **Output**: Findings written to `.claude/tmp/review-result.json`

**Example configuration with code review:**

```json
{
  "tickets_dir": "migration_plan/tickets",
  "project_name": "GOgent-Fortress",
  "audit_config": {
    "enabled": true,
    "code_review": {
      "enabled": true,
      "block_on_critical": true
    },
    "test_commands": {
      "go": {
        "unit": "go test -v ./...",
        "race": "go test -race ./...",
        "coverage": "go test -coverprofile={audit_dir}/coverage.out ./..."
      }
    }
  }
}
```

**When to enable code review:**

- Projects requiring consistent code quality
- Team environments with coding standards
- Security-sensitive codebases
- Large refactoring efforts

**When to disable code review:**

- Solo prototyping
- Emergency hotfixes
- Documentation-only tickets

---

### Phase 8: Completion Workflow

After acceptance criteria, optional audit, and optional code review verification, mark ticket complete and generate commit:

```bash
# Generate commit message
commit_msg=$(~/.claude/skills/ticket/scripts/generate-commit-msg.sh "$ticket_file")

# Show commit message preview
echo "[ticket] Generated commit message:"
echo "---"
echo "$commit_msg"
echo "---"

# Git operations (user confirms first)
echo "Commit and complete ticket? (y/n)"
# Await user confirmation, then:

git add .
git commit -m "$commit_msg"

# Update status to completed
~/.claude/skills/ticket/scripts/update-ticket-status.sh \
    "$tickets_index" \
    "$ticket_id" \
    "completed"

# Clear current ticket state
rm -f "$tickets_dir/.current-ticket"

# Log review outcomes if code review was run (if telemetry enabled)
if [[ "${GOGENT_ENABLE_TELEMETRY:-1}" == "1" ]] && [[ -f "$tickets_dir/.review-findings-$ticket_id" ]]; then
    commit_hash=$(git rev-parse HEAD 2>/dev/null || echo "")
    while IFS= read -r finding_id; do
        gogent-update-review-outcome \
            --finding-id="$finding_id" \
            --resolution="fixed" \
            --ticket-id="$ticket_id" \
            --commit="$commit_hash" 2>/dev/null || true
    done < "$tickets_dir/.review-findings-$ticket_id"
    rm -f "$tickets_dir/.review-findings-$ticket_id"
fi

echo "[ticket] ✓ Ticket $ticket_id completed"

# Show next available tickets
echo ""
echo "[ticket] Next available tickets:"
~/.claude/skills/ticket/scripts/find-next-ticket.sh "$tickets_index" | head -3
```

---

## State Files

| File                          | Purpose                       | Format                                            |
| ----------------------------- | ----------------------------- | ------------------------------------------------- |
| `.ticket-config.json`         | Project configuration         | JSON with tickets_dir, project_name, audit_config |
| `tickets-index.json`          | Ticket registry and status    | JSON with metadata and tickets array              |
| `.current-ticket`             | Current ticket ID             | Plain text (e.g., "GoGent-002")                   |
| `plans/{ticket-id}-plan.md`   | Implementation plans          | Markdown from architect                           |
| `implementation-plan.json`    | Machine-readable plan         | JSON (architect dual-output)                      |
| `teams/{ts}.implementation/`  | Team-run working directory    | config.json + stdin/stdout files                  |
| `.ticket-audits/{ticket-id}/` | Audit artifacts               | Directory with test logs, coverage data, summary  |

---

## Cost Model

### Foreground Path (no team-run)

| Phase                | Model      | Est. Tokens | Cost            |
| -------------------- | ---------- | ----------- | --------------- |
| Discovery            | Bash       | 0           | $0.000          |
| Selection            | Bash       | 0           | $0.000          |
| Validation           | Python     | 0           | $0.000          |
| Planning (if needed) | Sonnet     | 10-15K      | $0.09-$0.14     |
| Tracking             | TaskCreate | 0           | $0.000          |
| Verification         | Python     | 0           | $0.000          |
| Audit (if enabled)   | Bash       | 0           | $0.000          |
| Completion           | Bash       | 0           | $0.000          |
| **Total per ticket** |            | 10-15K      | **$0.09-$0.14** |

### Team-Run Path (background)

| Phase                   | Model       | Est. Tokens | Cost              |
| ----------------------- | ----------- | ----------- | ----------------- |
| Discovery–Validation    | Bash/Python | 0           | $0.000            |
| Planning                | Sonnet      | 10-15K      | $0.09-$0.14       |
| gogent-plan-impl        | Go binary   | 0           | $0.000            |
| Worker agents (per task)| Sonnet      | 10-20K each | $0.09-$0.18 each  |
| **Total (3-task plan)** |             | 40-75K      | **$0.36-$0.68**   |

**Cost savings vs manual workflow:** ~40% (foreground), ~60% (team-run: parallelism reduces wall-clock time)

---

## Memory Integration

The /ticket skill automatically updates project state:

- Current ticket tracked in `.current-ticket`
- Status updates in `tickets-index.json`
- Plans archived in `plans/` directory

On session resume, the skill reads `.current-ticket` to restore context.

---

## Troubleshooting

**"No ticket directory found"**

- Create `.ticket-config.json` at project root
- Or ensure `tickets/tickets-index.json` exists

**"Schema validation FAILED"**

- Check ticket has required frontmatter fields
- Verify acceptance criteria exist (markdown checkboxes)
- Run validation manually: `validate-ticket-schema.py ticket.md`

**"No actionable tickets found"**

- Check if dependencies are marked as "completed" in tickets-index.json
- Verify `status == "pending"` for expected tickets

**"Planning needed but no architect available"**

- Architect requires Plan subagent_type
- Ensure routing-schema.json includes architect → Plan mapping

**"Audit failed (non-blocking)"**

- Check test logs in `.ticket-audits/{ticket-id}/unit-tests.log`
- Review configuration in `.ticket-config.json` under `audit_config.test_commands`
- Verify test infrastructure is set up (go.mod, pyproject.toml, etc.)
- Run audit manually: `~/.claude/skills/ticket/scripts/run-audit.sh --ticket-id={ticket-id}`
- Note: Audit failures do NOT prevent ticket completion

**"Audit disabled/skipped"**

- Enable in `.ticket-config.json`: `"audit_config": { "enabled": true }`
- If missing, audit is skipped by default (backward compatible behavior)

---

## Example Session

```bash
$ cd ~/my-project
$ /ticket next

[ticket] Found tickets at: /home/user/my-project/implementation_plan/tickets
[ticket] Selected: FEAT-001
[ticket] Schema validation: PASS
[ticket] Planning needed: true
[ticket] Reason: High complexity (4 files, 3h estimate)
[ROUTING] → architect (implementation planning for FEAT-001)

[architect returns plan]

[ticket] Status updated: in_progress
[ticket] Created 8 tasks from acceptance criteria

[... implementation work happens ...]

$ /ticket verify
[ticket] Acceptance criteria: 8/8 complete ✓
[ticket] Running audit documentation...
[INFO] Starting audit for ticket: FEAT-001
[INFO] Detected language: go
[INFO] Phase 2: Executing tests...
[PASS] Unit tests passed
[PASS] Race detector passed
[INFO] Total coverage: 87.3%
[INFO] Phase 3: Generating implementation summary...
[ticket] ✓ Audit complete

$ /ticket complete
[ticket] Generated commit message:
---
feat: FEAT-001 - Add user authentication

Implement JWT-based authentication system with refresh tokens

Ticket-Id: FEAT-001

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
---
Commit and complete ticket? (y/n) y

[ticket] ✓ Ticket FEAT-001 completed

[ticket] Next available tickets:
FEAT-002
FEAT-003
```

---

**Skill Version**: 1.2
**Last Updated**: 2026-02-08
**Maintained By**: System
