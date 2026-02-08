# Review-Orchestrator Team-Run Bridge Document

**Purpose:** Reconcile TC-013's review-orchestrator rewrite spec (lines 425-614) with actual binary interfaces, schema naming, and stdin structure.

**Source of truth hierarchy:**
1. Go source code (`cmd/gogent-team-run/*.go`)
2. TC-009 schemas (`schemas/stdin/reviewer.json`, `schemas/teams/review.json`)
3. This bridge doc
4. TC-013 inline examples (lowest — known stale)

---

## 1. Current -> Target Transition

| Aspect | Current (Foreground) | Target (Background) |
|--------|---------------------|---------------------|
| **Dispatch** | Router calls `Task(sonnet)` for review-orchestrator | Router generates config.json + stdin files directly |
| **Agent spawning** | `mcp__gofortress__spawn_agent` for 4 reviewers | `gogent-team-run <team-dir>` spawns `claude` CLI processes |
| **Orchestrator LLM** | Yes (review-orchestrator agent coordinates) | **None** — Go binary IS the orchestrator |
| **TUI blocking** | ~2-3 minutes | <10 seconds |
| **Result aggregation** | review-orchestrator synthesizes | `/team-result` slash command (TC-012) post-processes |
| **Progress tracking** | Manual | `/team-status` (TC-012) |

**Key insight:** In the background path, the review-orchestrator agent is eliminated entirely. The router generates config directly, the Go binary executes, and `/team-result` aggregates.

---

## 2. TC-013 Corrections Checklist

### Field Name Corrections

| TC-013 line(s) | TC-013 uses | Correct (config.go) | Fix |
|----------------|-------------|---------------------|-----|
| 488-498 | `agent_id` in config example | `agent` | Replace |
| 490 | `budget_total_usd` | `budget_max_usd` | Replace |
| 535 | `"agent_id": "backend-reviewer"` | `"agent": "backend-reviewer"` | Replace |

### Schema Path Corrections

| TC-013 line(s) | TC-013 references | Correct (TC-009) | Fix |
|----------------|-------------------|-------------------|-----|
| 535 | `https://gogent.dev/schemas/stdin/reviewer-v1.json` | `https://gogent-fortress/schemas/stdin/reviewer.json` | Replace |
| 1074 | `schemas/stdin/reviewer-v1.json` | `schemas/stdin/reviewer.json` | Replace |

### Launch Command Corrections

| TC-013 line(s) | TC-013 uses | Correct | Fix |
|----------------|-------------|---------|-----|
| 586 | `gogent-team-run "$team_dir/config.json" > "$team_dir/launch.log" 2>&1` | `gogent-team-run "$team_dir"` | Binary takes directory, handles own log redirection |
| 589-594 | `sleep 1` + manual PID check | Read `config.json` for `background_pid` (now written by daemon on startup) | Simplify verification |

### Stdin Structure Corrections

| TC-013 line(s) | TC-013 structure | Correct (reviewer.json schema) | Fix |
|----------------|-----------------|-------------------------------|-----|
| 543-549 | `context.diff_patch`, `context.files_to_review` | `review_scope.files[]`, `git_context.commit_message` | Restructure entirely |
| 555-561 | `review_focus` (array of strings) | `focus_areas` (object with domain-specific fields) | Replace |
| 562-568 | `output_requirements` | Not in stdin schema (prompt envelope handles output format) | Remove |
| (missing) | No `workflow` field | `"workflow": "review"` (required) | Add |
| (missing) | No `project_conventions` | `project_conventions` (required) | Add |

---

## 3. Complete Stdin Example (Validated Against `schemas/stdin/reviewer.json`)

```json
{
  "agent": "backend-reviewer",
  "workflow": "review",
  "context": {
    "project_root": "/home/user/Documents/GOgent-Fortress",
    "team_dir": "/home/user/.claude/sessions/20260208.a3f2/teams/1738876543.code-review"
  },
  "review_scope": {
    "files": [
      {
        "path": "cmd/gogent-team-run/wave.go",
        "language": "go",
        "category": "backend",
        "changed_lines": {"added": 15, "removed": 8},
        "is_new_file": false
      },
      {
        "path": "cmd/gogent-team-run/spawn.go",
        "language": "go",
        "category": "backend",
        "changed_lines": {"added": 3, "removed": 20},
        "is_new_file": false
      }
    ],
    "total_files": 2,
    "languages_detected": ["go"]
  },
  "git_context": {
    "commit_message": "fix: resolve budget double-reservation in team runner",
    "branch_name": "multiagent-dispatch",
    "related_tickets": ["TC-013-prereq"]
  },
  "focus_areas": {
    "security": false,
    "api_design": false,
    "concurrency": true,
    "error_handling": true
  },
  "project_conventions": {
    "language": "go",
    "conventions_file": "go.md",
    "test_pattern": "table-driven"
  }
}
```

**Validation notes:**
- `agent` field: enum value from `["backend-reviewer", "frontend-reviewer", "standards-reviewer", "architect-reviewer"]`
- `workflow` field: const `"review"`
- `context`: requires absolute paths (`^/` pattern)
- `review_scope.files[]`: each file needs `path`, `language`, `category`, `changed_lines`, `is_new_file`
- `git_context`: requires `commit_message`
- `focus_areas`: object (domain-specific, not string array)
- `project_conventions`: object (not string array)

**What `buildPromptEnvelope` validates** (envelope.go):
- `agent` non-empty (used for `AGENT:` header)
- `context` non-empty object
- `task` OR `description` non-empty string

**Note:** The reviewer stdin schema requires `review_scope` etc., but `buildPromptEnvelope` does NOT validate these. The stdin JSON is passed through to the agent as-is in the envelope. Schema compliance is the config generator's responsibility, enforced via external `ajv validate` in CI, not at runtime.

---

## 4. Correct Launch Sequence

```bash
# 1. Create team directory
session_dir=".claude/sessions/${GOGENT_SESSION_ID:-$(date +%Y%m%d).$(uuidgen | cut -d- -f1)}"
team_dir="${session_dir}/teams/$(date +%s).code-review"
mkdir -p "$team_dir"

# 2. Generate config.json
# Copy template, fill dynamic fields
# (LLM or lightweight script — no helper binary needed for review)

# 3. Generate stdin files (one per reviewer)
# Each must comply with schemas/stdin/reviewer.json required fields

# 4. Launch
gogent-team-run "$team_dir"
# Binary handles:
#   - PID file (gogent-team-run.pid)
#   - Log redirection (runner.log)
#   - Setsid (session leader)
#   - Writes background_pid + status to config.json

# 5. Verify (read back config.json)
sleep 2
background_pid=$(jq -r '.background_pid' "$team_dir/config.json")
if [[ -z "$background_pid" || "$background_pid" == "null" ]]; then
  echo "[ERROR] Team launch failed. Check $team_dir/runner.log"
  exit 1
fi
echo "[review] Team launched (PID $background_pid). Use /team-status to track."
```

---

## 5. Integration Notes

### No Inter-Wave Script
Review uses a single wave (Wave 1) with all reviewers in parallel. `on_complete_script` is `null`. No `gogent-team-prepare-synthesis` needed.

### Result Aggregation
Happens in `/team-result` slash command (TC-012), NOT in the Go binary. When user runs `/team-result`:
1. Read all `stdout_*.json` files from team directory
2. Extract findings from each reviewer
3. Merge and group by severity (critical, warning, info)
4. Compute approval status (APPROVED/WARNING/BLOCKED)
5. Output report

### Reviewer Model Selection
Per `review.json` template:
- `backend-reviewer`: haiku (120s timeout)
- `frontend-reviewer`: haiku (120s timeout)
- `standards-reviewer`: haiku (120s timeout)
- `architect-reviewer`: sonnet (300s timeout)

### Dynamic Reviewer Selection
Not all 4 reviewers run every time. The router selects reviewers based on changed file types:
- `.go`, `.py` backend files -> `backend-reviewer`
- `.tsx`, `.ts` UI files -> `frontend-reviewer`
- All files -> `standards-reviewer` (always included)
- 5+ files OR cross-module changes -> `architect-reviewer`

Config.json `waves[0].members[]` should only include selected reviewers.

---

## 6. Test Cases

### Test Case A: Happy Path (4 reviewers, single wave)

**Setup:**
- Stage changes to `handler.go` (backend) + `Button.tsx` (frontend)
- Generate config with all 4 reviewers

**Expected:**
- Config.json has 4 members in Wave 1
- `gogent-team-run` spawns 4 claude CLI processes in parallel
- All 4 complete within timeout
- Each writes `stdout_*.json` with findings
- `/team-result` aggregates findings by severity
- Approval status based on highest severity

### Test Case B: Empty Diff (abort before launch)

**Setup:**
- No staged changes (`git diff --staged` returns empty)

**Expected:**
- Router outputs "No files to review"
- No team directory created
- No `gogent-team-run` invoked
- Return time: <1 second

---

## Reference Files

| What | Where |
|------|-------|
| Config struct | `cmd/gogent-team-run/config.go:18-62` |
| Envelope builder | `cmd/gogent-team-run/envelope.go:57-129` |
| Wave execution | `cmd/gogent-team-run/wave.go:13-61` |
| Spawn + retry | `cmd/gogent-team-run/spawn.go:340-447` |
| Reviewer stdin schema | `.claude/schemas/stdin/reviewer.json` |
| Review team template | `.claude/schemas/teams/review.json` |
| Review skill | `.claude/skills/review/SKILL.md` |
