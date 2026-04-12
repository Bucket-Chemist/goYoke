# Team Configuration Field Contracts

This document defines the field contracts for team configuration JSON files used by `gogent-team-run`.

## Top-Level Fields

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `team_name` | string | yes | Format: `{workflow}-{unix_timestamp}`. Used for directory naming. |
| `workflow_type` | enum | yes | One of: `braintrust`, `review`, `implementation` |
| `project_root` | string | yes | Absolute path to project. Must exist at runtime. |
| `session_id` | string | yes | UUID format. Links team to spawning session. |
| `created_at` | string | yes | ISO-8601 timestamp of team creation. |
| `background_pid` | int/null | yes | Initially `null`. Written by `gogent-team-run` after spawning background process. |
| `budget_max_usd` | float64 | yes | Maximum USD budget for entire team. **Top-level flat field** (not nested). |
| `budget_remaining_usd` | float64 | yes | Remaining budget. Decremented as agents complete. **Top-level flat field** (not nested). |
| `warning_threshold_usd` | float64 | yes | Budget warning threshold. Typically 80% of `budget_max_usd`. |
| `status` | enum | yes | Team status. Initially `pending`. Runtime: `running`, `completed`, `failed`. |
| `started_at` | int64/null | yes | Unix timestamp when team execution began. Initially `null`. |
| `completed_at` | int64/null | yes | Unix timestamp when team completed. Initially `null`. |
| `waves` | array | yes | Array of wave objects. Must have at least one wave. |

## Wave Fields

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `wave_number` | int | yes | 1-indexed. Must be sequential (1, 2, 3...). |
| `description` | string | yes | Human-readable description of wave purpose. |
| `members` | array | yes | Array of member objects. Must have at least one member. |
| `on_complete_script` | string/null | yes | Script to run after all wave members complete. `null` if no script. |

## Member Fields

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `name` | string | yes | Unique within team. Used for stdin/stdout filenames. Human-readable (e.g., "einstein", "backend-reviewer"). |
| `agent` | string | yes | Must exist in `agents-index.json`. Determines agent capabilities. (Renamed from `agent_id` to match Go struct.) |
| `model` | enum | yes | One of: `haiku`, `sonnet`, `opus`. Used for cost calculation. |
| `stdin_file` | string | yes | Filename for agent input. Convention: `stdin_{name}.json` (use `name`, not `agent`). |
| `stdout_file` | string | yes | Filename for agent output. Convention: `stdout_{name}.json` (use `name`, not `agent`). |
| `status` | enum | yes | Initially `pending`. Runtime values: `running`, `completed`, `failed`. |
| `process_pid` | int/null | yes | Initially `null`. Written by `gogent-team-run`. (Renamed from `pid` to match Go struct.) |
| `exit_code` | int/null | yes | Process exit code. Initially `null`. Written on completion. |
| `error_message` | string | yes | Error details if failed. Initially empty string. |
| `started_at` | int64/null | yes | Unix timestamp when member started. Initially `null`. |
| `completed_at` | int64/null | yes | Unix timestamp when member completed. Initially `null`. |
| `cost_usd` | float64 | yes | Initially `0.0`. Updated when agent completes. |
| `cost_status` | string | yes | Cost extraction status. Values: `""`, `"ok"`, `"unknown"`, `"error"`. |
| `retry_count` | int | yes | Initially `0`. Incremented on retry. |
| `max_retries` | int | yes | Maximum retry attempts. Typically `1` for Opus (expensive), `2` for Haiku/Sonnet. |
| `timeout_ms` | int | yes | Agent timeout in milliseconds. Typical values: 120000 (Haiku), 600000 (Sonnet), 600000 (Opus). |

## Budget Field Contract (Critical Decision)

Budget fields are **top-level flat** (`budget_max_usd`, `budget_remaining_usd`), NOT nested under a `budget` object.

**Rationale**: This resolves the TC-009/TC-008/TC-013 contradiction flagged by multiple reviewers. Flat structure:
- Simplifies JSON unmarshaling in Go
- Reduces nesting depth
- Matches existing GOgent convention patterns

**Example**:
```json
{
  "budget_max_usd": 5.0,
  "budget_remaining_usd": 5.0,
  "waves": [...]
}
```

**NOT**:
```json
{
  "budget": {
    "max_usd": 5.0,
    "remaining_usd": 5.0
  },
  "waves": [...]
}
```

## Field Rename History

| Old Name | New Name | Reason | Changed In |
|----------|----------|--------|------------|
| `agent_id` | `agent` | Match TC-008 Go struct `json:"agent"` tag | TC-006 |
| `pid` | `process_pid` | Match TC-008 Go struct `json:"process_pid"` tag | TC-006 |

## Naming Conventions

### `name` vs `agent`

- **`name`**: Human-readable identifier unique within the team. Used for:
  - stdin/stdout filenames (`stdin_{name}.json`)
  - User-facing status messages
  - Log entries

- **`agent`**: References entry in `agents-index.json`. Determines:
  - Agent capabilities and prompt
  - Spawning relationships
  - Validation rules
  - Allowed tools (from PERMISSION-DESIGN.md)

**Example**:
```json
{
  "name": "staff-architect",
  "agent": "staff-architect-critical-review",
  "stdin_file": "stdin_staff-architect.json"
}
```

Note: `stdin_file` uses `name`, not `agent`.

## Stdin/Stdout Contracts

Each agent type has a comprehensive stdin/stdout JSON schema in `stdin-stdout/`:

| Agent | Schema File | Key Input | Key Output |
|-------|-------------|-----------|------------|
| Einstein | `braintrust-einstein.json` | Problem brief, analysis axes, codebase context | Root cause analysis, frameworks, novel approaches |
| Staff-Architect | `braintrust-staff-architect.json` | Plan to review, 7-layer focus areas | Issue register, assumption register, verdict |
| Beethoven | `braintrust-beethoven.json` | Problem brief + both Wave 1 outputs | Convergence/divergence resolution, unified recommendations |
| Backend Reviewer | `review-backend.json` | Backend files, security/API/data focus | Findings with severity, security summary |
| Frontend Reviewer | `review-frontend.json` | Frontend files, a11y/hooks/perf focus | Findings with severity, accessibility summary |
| Standards Reviewer | `review-standards.json` | Source files, naming/complexity/DRY focus | Findings with severity, complexity score |
| Architect Reviewer | `review-architect.json` | Changed files + imports, coupling/boundary focus | Findings with severity, structural health score |

### Data Flow

**Braintrust workflow**:
```
Mozart → stdin_einstein.json → Einstein → stdout_einstein.json
Mozart → stdin_staff-architect.json → Staff-Architect → stdout_staff-architect.json
gogent-team-prepare-synthesis → stdin_beethoven.json (includes both Wave 1 outputs)
stdin_beethoven.json → Beethoven → stdout_beethoven.json → Go binary writes analysis.md
```

**Review workflow**:
```
Orchestrator → stdin_backend-reviewer.json → Backend → stdout_backend-reviewer.json
Orchestrator → stdin_frontend-reviewer.json → Frontend → stdout_frontend-reviewer.json
Orchestrator → stdin_standards-reviewer.json → Standards → stdout_standards-reviewer.json
Orchestrator → stdin_architect-reviewer.json → Architect → stdout_architect-reviewer.json
Go binary collects all stdout files → writes unified review report
```

### Two-Layer Parsing

The Go binary parses CLI output in two layers:
1. **Layer 1**: CLI wrapper JSON array → result event → `total_cost_usd` (cost tracking)
2. **Layer 2**: `result.Result` string → agent-specific stdout schema → file writes

See `cmd/gogent-team-run/docs/claude-cli-output-format.md` for Layer 1.
See `stdin-stdout/*.json` files for Layer 2.

## Validation Requirements

Implementations must validate:

1. **Structural**:
   - All required fields present
   - Correct types for all fields
   - Wave numbers sequential starting from 1
   - Member names unique within team

2. **Referential**:
   - `agent_id` exists in `agents-index.json`
   - `project_root` exists on filesystem
   - `workflow_type` matches one of the defined enums

3. **Runtime**:
   - `budget_remaining_usd <= budget_max_usd`
   - `retry_count <= max_retries`
   - `status` transitions are valid (pending -> running -> completed/failed)

## File Locations

Templates are stored in:
```
.claude/schemas/teams/
  ├── braintrust.json
  ├── review.json
  ├── implementation.json
  ├── common-types.md (this file)
  ├── README.md
  ├── PROJECT-ROOT-RESOLUTION.md
  └── stdin-stdout/
      ├── braintrust-einstein.json
      ├── braintrust-staff-architect.json
      ├── braintrust-beethoven.json
      ├── review-backend.json
      ├── review-frontend.json
      ├── review-standards.json
      ├── review-architect.json
      └── implementation-worker.json
```

Instantiated team configs are written to:
```
sessions/{YYMMDD}.{id}/teams/{timestamp}.{workflow_type}/
  ├── config.json
  ├── stdin_einstein.json
  ├── stdout_einstein.json
  └── ...
```
