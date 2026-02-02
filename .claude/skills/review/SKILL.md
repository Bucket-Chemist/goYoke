---
name: review
description: Orchestrated multi-domain code review with severity-grouped findings and approval status
---

# Review Skill v1.0

## Purpose

Automated code review through coordinated specialist reviewers. Analyzes changed files, identifies relevant review domains, spawns reviewers in parallel, and synthesizes findings into actionable report.

**What this skill does:**
1. **Detect** — Find changed files via git diff or specified scope
2. **Classify** — Identify languages and architectural layers present
3. **Select** — Choose relevant reviewers (backend, frontend, standards, architecture)
4. **Execute** — Spawn reviewers in parallel via review-orchestrator
5. **Synthesize** — Collect and group findings by severity
6. **Report** — Generate unified report with approval status

**What this skill does NOT do:**
- Implement fixes (generates recommendations only)
- Enforce routing rules (handled by hooks)
- Replace human review (supplements, doesn't replace)

---

## Invocation

- `/review` — Review all staged changes (git diff --staged)
- `/review --all` — Review all uncommitted changes (git diff HEAD)
- `/review --scope=<glob>` — Review specific files (e.g., "**/*.go")
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

### Phase 2: Language Classification

Identify languages and architectural layers:

```bash
# Group files by extension
declare -A langs
while IFS= read -r file; do
    ext="${file##*.}"
    langs["$ext"]=1
done <<< "$files"

# Map to review domains
reviewers=()
if [[ -n "${langs[go]}" ]]; then
    reviewers+=("backend-reviewer")
fi
if [[ -n "${langs[ts]}" || -n "${langs[tsx]}" || -n "${langs[jsx]}" ]]; then
    reviewers+=("frontend-reviewer")
fi
# Always include standards reviewer
reviewers+=("standards-reviewer")
# Always include architecture reviewer
reviewers+=("architect-reviewer")

echo "[review] Selected reviewers: ${reviewers[*]}"
```

---

### Phase 3: Orchestrated Review

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
     - Spawn reviewers in parallel via Task
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

### Phase 4: Report Generation

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

| Phase | Model | Est. Tokens | Cost |
|-------|-------|-------------|------|
| Detection | Bash | 0 | $0.000 |
| Classification | Bash | 0 | $0.000 |
| Orchestrator | Sonnet | 8-12K | $0.07-$0.11 |
| Backend Reviewer | Haiku+Think | 3-5K | $0.003-$0.005 |
| Frontend Reviewer | Haiku+Think | 3-5K | $0.003-$0.005 |
| Standards Reviewer | Haiku+Think | 3-5K | $0.003-$0.005 |
| Architect Reviewer | Sonnet | 8-12K | $0.07-$0.11 |
| **Total (4 reviewers)** | | 28-42K | **$0.15-$0.24** |

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

| File | Purpose | Format |
|------|---------|--------|
| `.claude/tmp/review-result.json` | Review findings and status | JSON with status, summary, findings |
| `.claude/tmp/review-scope.txt` | Files under review | Line-separated file paths |

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

---

## ML Telemetry

This skill logs all findings to ML telemetry for downstream analysis:

| File | Purpose |
|------|---------|
| `$XDG_DATA_HOME/gogent-fortress/review-findings.jsonl` | All review findings |
| `$XDG_DATA_HOME/gogent-fortress/sharp-edge-hits.jsonl` | Sharp edge correlations |
| `.claude/tmp/review-telemetry.json` | Session telemetry output (finding IDs) |

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

**Skill Version**: 1.1
**Last Updated**: 2026-02-02
**Maintained By**: System
