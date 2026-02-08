# Impl-Manager Team-Run Bridge Document

**Purpose:** Reconcile TC-013's impl-manager rewrite spec (lines 617-930) with actual binary interfaces, schema naming, and stdin structure.

**Source of truth hierarchy:**
1. Go source code (`cmd/gogent-team-run/*.go`)
2. TC-009 schemas (`schemas/stdin/worker.json`, `schemas/teams/implementation.json`)
3. This bridge doc
4. TC-013 inline examples (lowest — known stale)

---

## 1. Current -> Target Transition

| Aspect | Current (Foreground) | Target (Background) |
|--------|---------------------|---------------------|
| **Dispatch** | Router calls `Task(sonnet)` for impl-manager | Router parses specs.md, computes waves, generates config |
| **Agent spawning** | `Task()` for each worker sequentially | `gogent-team-run <team-dir>` spawns workers per wave |
| **Wave computation** | Impl-manager LLM computes at runtime | Computed at config generation time (Kahn's algorithm) |
| **TUI blocking** | ~5-15 minutes | <15 seconds |
| **Progress tracking** | TaskCreate/TaskUpdate calls | `/team-status` (TC-012) reads config.json |

**Key difference:** Wave computation happens at config generation time, not at execution time. The Go binary executes a pre-computed wave plan.

---

## 2. TC-013 Corrections Checklist

### Field Name Corrections

| TC-013 line(s) | TC-013 uses | Correct (config.go) | Fix |
|----------------|-------------|---------------------|-----|
| 688, 694, 702 | `agent_id` in member objects | `agent` | Replace |
| 828-865 | `members[].wave` (flat field) | Members nested in `waves[].members[]` | Restructure |

### Schema Path Corrections

| TC-013 line(s) | TC-013 references | Correct (TC-009) | Fix |
|----------------|-------------------|-------------------|-----|
| 878 | `https://gogent.dev/schemas/stdin/implementation-worker-v1.json` | `https://gogent-fortress/schemas/stdin/worker.json` | Replace |
| 1075 | `schemas/stdin/implementation-worker-v1.json` | `schemas/stdin/worker.json` | Replace |

### Launch Command Corrections

| TC-013 line(s) | TC-013 uses | Correct | Fix |
|----------------|-------------|---------|-----|
| 911 | `gogent-team-run "$team_dir/config.json" > "$team_dir/launch.log" 2>&1` | `gogent-team-run "$team_dir"` | Binary takes directory, handles own log |

### Stdin Structure Corrections

| TC-013 line(s) | TC-013 structure | Correct (worker.json schema) | Fix |
|----------------|-----------------|------------------------------|-----|
| 882 | `"agent_id": "go-pro"` | `"agent": "go-pro"` | Replace |
| 883 | `"task_id": "TC-001"` (top-level) | `task.task_id` (nested in task object) | Restructure |
| 884-887 | `task.title`, `task.description`, `task.files_to_modify` | `task.task_id`, `task.subject`, `task.description`, `task.acceptance_criteria` | Restructure |
| 889-892 | `context.specs_excerpt`, `context.conventions` (strings) | `conventions` object (separate required field), `implementation_scope` object | Restructure |
| 894-895 | `reads_from` array | Not in schema | Remove |
| 896-902 | `output_requirements` | Not in schema (envelope handles output) | Remove |
| (missing) | No `workflow` field | `"workflow": "implementation"` (required) | Add |
| (missing) | No `implementation_scope` | Required: `target_packages`, `related_files`, `tests_required` | Add |
| (missing) | No `codebase_context` | Required field | Add |

### Config Structure Corrections

TC-013 lines 828-865 use a **flat member structure** with `"wave": 0` on each member. The actual config uses **nested waves**:

**TC-013 (WRONG):**
```json
{
  "members": [
    {"name": "TC-001", "agent_id": "go-pro", "wave": 0, ...},
    {"name": "TC-002", "agent_id": "go-pro", "wave": 1, ...}
  ]
}
```

**Correct (config.go):**
```json
{
  "waves": [
    {
      "wave_number": 1,
      "members": [
        {"name": "TC-001", "agent": "go-pro", ...}
      ]
    },
    {
      "wave_number": 2,
      "members": [
        {"name": "TC-002", "agent": "go-pro", ...}
      ]
    }
  ]
}
```

Note: `wave_number` is 1-indexed (per `implementation.json` template), not 0-indexed as TC-013 assumes.

---

## 3. Complete Config.json Example (Nested Wave Structure)

```json
{
  "team_name": "implementation-1738876543",
  "workflow_type": "implementation",
  "project_root": "/home/user/Documents/GOgent-Fortress",
  "session_id": "20260208.a3f2",
  "created_at": "2026-02-08T14:35:43Z",
  "background_pid": null,
  "budget_max_usd": 10.0,
  "budget_remaining_usd": 10.0,
  "warning_threshold_usd": 8.0,
  "status": "pending",
  "started_at": null,
  "completed_at": null,
  "waves": [
    {
      "wave_number": 1,
      "description": "Foundation tasks (no dependencies)",
      "members": [
        {
          "name": "task-001",
          "agent": "go-pro",
          "model": "sonnet",
          "stdin_file": "stdin_task-001.json",
          "stdout_file": "stdout_task-001.json",
          "status": "pending",
          "process_pid": null,
          "exit_code": null,
          "cost_usd": 0.0,
          "cost_status": "",
          "error_message": "",
          "retry_count": 0,
          "max_retries": 2,
          "timeout_ms": 300000,
          "started_at": null,
          "completed_at": null
        },
        {
          "name": "task-002",
          "agent": "go-cli",
          "model": "sonnet",
          "stdin_file": "stdin_task-002.json",
          "stdout_file": "stdout_task-002.json",
          "status": "pending",
          "process_pid": null,
          "exit_code": null,
          "cost_usd": 0.0,
          "cost_status": "",
          "error_message": "",
          "retry_count": 0,
          "max_retries": 2,
          "timeout_ms": 300000,
          "started_at": null,
          "completed_at": null
        }
      ],
      "on_complete_script": null
    },
    {
      "wave_number": 2,
      "description": "Tasks depending on wave 1",
      "members": [
        {
          "name": "task-003",
          "agent": "go-pro",
          "model": "sonnet",
          "stdin_file": "stdin_task-003.json",
          "stdout_file": "stdout_task-003.json",
          "status": "pending",
          "process_pid": null,
          "exit_code": null,
          "cost_usd": 0.0,
          "cost_status": "",
          "error_message": "",
          "retry_count": 0,
          "max_retries": 2,
          "timeout_ms": 300000,
          "started_at": null,
          "completed_at": null
        }
      ],
      "on_complete_script": null
    }
  ]
}
```

---

## 4. Complete Worker Stdin Example (Validated Against `schemas/stdin/worker.json`)

```json
{
  "agent": "go-pro",
  "workflow": "implementation",
  "context": {
    "project_root": "/home/user/Documents/GOgent-Fortress",
    "team_dir": "/home/user/.claude/sessions/20260208.a3f2/teams/1738876543.implementation"
  },
  "task": {
    "task_id": "task-001",
    "subject": "Implement authentication handler",
    "description": "Create JWT-based auth handler with table-driven tests per specs section 3.2. Handler should validate tokens, extract claims, and return appropriate HTTP status codes.",
    "acceptance_criteria": [
      "Handler validates JWT tokens with RS256 algorithm",
      "Returns 401 for expired/invalid tokens",
      "Extracts user_id claim and passes to context",
      "Table-driven tests cover all error paths"
    ],
    "blocked_by": [],
    "blocks": ["task-003"]
  },
  "implementation_scope": {
    "target_packages": ["internal/handlers"],
    "related_files": [
      {"path": "internal/handlers/middleware.go", "relevance": "Existing middleware patterns to follow"},
      {"path": "internal/auth/jwt.go", "relevance": "JWT utility functions to use"}
    ],
    "tests_required": true,
    "build_verification": "go build ./..."
  },
  "conventions": {
    "language": "go",
    "conventions_file": "go.md",
    "error_handling": "explicit error returns, no panics",
    "test_pattern": "table-driven"
  },
  "codebase_context": {
    "architecture_notes": "Handlers follow http.HandlerFunc pattern with middleware chaining",
    "patterns_to_follow": ["explicit error handling", "context propagation", "structured logging"],
    "anti_patterns": ["global state", "init() functions", "panic-based error handling"]
  }
}
```

**Required fields per schema:**
- `agent`: dynamic (e.g., `go-pro`, `python-pro`, `go-cli`)
- `workflow`: const `"implementation"`
- `context`: `project_root` + `team_dir` (absolute paths)
- `task`: `task_id`, `subject`, `description`, `acceptance_criteria`
- `implementation_scope`: `target_packages`, `related_files`, `tests_required`
- `conventions`: `language`, `conventions_file`
- `codebase_context`: object (optional fields)

---

## 5. Inter-Wave Handling

Implementation waves typically do NOT use `on_complete_script`. Wave sequencing is handled by `gogent-team-run` internally — wave N+1 waits for all wave N members to complete.

**Failure propagation:** If a member in wave N fails (after retries), the current implementation continues to wave N+1. TC-013 test case 8 expects that wave N+1 does NOT execute when a dependency fails. This behavior is NOT currently implemented in `wave.go` — `runWaves` iterates all waves unconditionally. If dependency-failure-stops-downstream is required, `runWaves` needs modification (scope for TC-013c, not this bridge doc).

---

## 6. Helper Binary Consolidation

TC-013 proposes 4 binaries: `gogent-team-init-review`, `gogent-team-init-impl`, `gogent-parse-specs`, `gogent-compute-waves`.

**Recommendation:** Merge into 1 binary: `gogent-plan-impl`

```bash
gogent-plan-impl --specs=.claude/tmp/specs.md --output="$team_dir"
```

This binary:
1. Parses specs.md to extract tasks (replaces `gogent-parse-specs`)
2. Computes wave DAG via Kahn's algorithm (replaces `gogent-compute-waves`)
3. Generates config.json + stdin files (replaces `gogent-team-init-impl`)

**Review workflow does NOT need a helper binary.** The LLM generates config directly (simple, fixed structure).

---

## 7. Correct Launch Sequence

```bash
# 1. Parse specs.md and generate team directory
gogent-plan-impl \
  --specs=".claude/tmp/specs.md" \
  --project-root="$(pwd)" \
  --output="$team_dir"

# 2. Launch
gogent-team-run "$team_dir"

# 3. Verify
sleep 2
background_pid=$(jq -r '.background_pid' "$team_dir/config.json")
if [[ -z "$background_pid" || "$background_pid" == "null" ]]; then
  echo "[ERROR] Team launch failed. Check $team_dir/runner.log"
  exit 1
fi

echo "[ticket] Implementation team dispatched (PID $background_pid)"
echo "[ticket] Waves: $(jq '.waves | length' "$team_dir/config.json")"
echo "[ticket] Tasks: $(jq '[.waves[].members[]] | length' "$team_dir/config.json")"
```

---

## 8. Test Cases

### Test Case A: Happy Path (5 tasks, 3 waves)

**Setup:**
- specs.md with 5 tasks:
  - Wave 1: task-001, task-002 (no deps)
  - Wave 2: task-003 (blocked by task-001)
  - Wave 3: task-004, task-005 (blocked by task-003)

**Expected:**
- `gogent-plan-impl` generates config.json with 3 waves
- Wave 1 members run in parallel (verify via timestamps)
- Wave 2 waits for Wave 1 completion
- All 5 tasks complete with `status: "completed"`
- `/team-status` shows correct wave progression

### Test Case B: Circular Dependency (error at config generation)

**Setup:**
- specs.md with: task-001 blocked_by task-002, task-002 blocked_by task-001

**Expected:**
- `gogent-plan-impl` detects cycle (Kahn's algorithm: zero-indegree set becomes empty while nodes remain)
- Error: "Circular dependency detected among: task-001, task-002"
- No team directory created
- No `gogent-team-run` invoked

---

## Reference Files

| What | Where |
|------|-------|
| Config struct | `cmd/gogent-team-run/config.go:18-62` |
| Wave execution | `cmd/gogent-team-run/wave.go:13-61` |
| Spawn + retry | `cmd/gogent-team-run/spawn.go:340-447` |
| Worker stdin schema | `.claude/schemas/stdin/worker.json` |
| Implementation template | `.claude/schemas/teams/implementation.json` |
| Kahn's algorithm spec | TC-013 lines 742-793 (algorithm is correct, context is stale) |
