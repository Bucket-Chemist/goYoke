---
name: teams
description: List all orchestration teams in the current session with summary status
version: 1.0.0
---

# Teams Skill

List all orchestration teams in the current session with summary status, grouped by status.

## Usage

```
/teams
```

No arguments required. Displays all teams in the current session.

## Implementation Instructions

Follow these steps to generate the team listing:

### Step 1: Discover Session Directory

1. Check environment variable `GOGENT_SESSION_DIR` first
2. If unset, fallback: find most recent session with `teams/` subdirectory under `~/.claude/sessions/`
3. Team directories are located at: `{session_dir}/teams/*/config.json`

If no session directory found:
```
No active session found. Start a session first or set GOGENT_SESSION_DIR.
```

If session found but no teams/ directory:
```
No teams found in current session.
```

### Step 2: Find All Team Directories

Use Bash to enumerate team directories:
```bash
find $SESSION_DIR/teams -mindepth 1 -maxdepth 1 -type d -name '*.*'
```

Expected format: `{timestamp}.{name}/config.json`

### Step 3: Read Each config.json

For each team directory, read `config.json` and extract:

| Field | Type | Purpose |
|-------|------|---------|
| `team_name` | string | Display name |
| `workflow_type` | "braintrust" \| "review" \| "implementation" | Workflow category |
| `status` | "pending" \| "running" \| "completed" \| "failed" | Current state |
| `created_at` | ISO-8601 | Creation timestamp |
| `background_pid` | integer \| null | Process ID if running in background |
| `budget_max_usd` | number | Maximum budget allocated |
| `budget_remaining_usd` | number | Remaining budget |
| `started_at` | ISO-8601 \| null | When execution started |
| `completed_at` | ISO-8601 \| null | When execution completed |
| `waves` | array | For wave progress calculation |

Skip teams with malformed JSON and log a warning.

### Step 4: Verify Process Status

For each team with `background_pid` set, verify the process is alive:

```bash
kill -0 $PID 2>/dev/null
```

- If exit code 0: Process is RUNNING
- If exit code non-zero but PID set: Process is STALE (mark as failed)

### Step 5: Calculate Derived Fields

For each team, calculate:

**Cost Spent:**
```
cost_spent = budget_max_usd - budget_remaining_usd
```

**Elapsed/Duration Time:**
- For RUNNING teams: `now - started_at`
- For COMPLETED teams: `completed_at - started_at`
- For FAILED teams: Use last known duration or "-"
- Format: < 60s → "45s", 60-3600s → "2m 14s", > 3600s → "1h 23m 14s"

**Wave Progress:**
- For RUNNING teams: Find first non-completed wave, display "Wave X/Y" where X is current wave number and Y is total waves
- For COMPLETED teams: Display "Complete"
- For FAILED teams: Display last wave attempted or "-"

### Step 6: Group Teams by Status

Group teams into these categories (display order):

1. **RUNNING** — Teams with active `background_pid` (verified alive)
2. **COMPLETED** — `status == "completed"`
3. **FAILED** — `status == "failed"` or stale PID
4. **CANCELLED** — `status == "cancelled"` or `status == "killed"`
5. **PENDING** — `status == "pending"`

Skip groups with 0 teams.

### Step 7: Sort Within Groups

Within each status group, sort reverse chronological (newest first) by `created_at`.

### Step 8: Format Output

**Header:**
```
Session: {session_id}
```

Extract `session_id` from session directory name (format: `YYMMDD.uuid-fragment`).

**Table Format:**

```
{STATUS_GROUP} ({count}):
  {team_name:<25} {workflow:<15} {progress:<12} {cost:>8} {duration:>12} {pid_or_dash}
```

Column spacing:
- Team name: 25 chars, left-aligned
- Workflow: 15 chars, left-aligned
- Progress: 12 chars, left-aligned
- Cost: 8 chars, right-aligned with $ prefix (e.g., "$3.42")
- Duration: 12 chars, right-aligned
- PID: Right-aligned, show "PID {number}" for running, "-" otherwise

**Footer:**
```
Total: {total_teams} teams | Running: {running_count} | Cost: ${total_cost}
```

Total cost format: 2 decimal places, sum of all teams' cost_spent.

**Empty State:**
```
No teams in current session. Use /braintrust, /review, or /ticket to start one.
```

### Step 9: Example Output

```
Session: 260206.abc123

RUNNING (1):
  braintrust-1738856422    braintrust      Wave 2/3      $3.42     2m 45s  PID 12345

COMPLETED (2):
  review-1738850011        review          Complete      $1.23     1m 32s  -
  impl-1738845678          implementation  Complete      $5.67     8m 14s  -

FAILED (1):
  braintrust-1738840000    braintrust      Wave 1/2      $0.52     FAILED  -

Total: 4 teams | Running: 1 | Cost: $10.84
```

## Error Handling

| Error | Response |
|-------|----------|
| Session dir not found | "No active session found. Start a session first or set GOGENT_SESSION_DIR." |
| No teams/ directory | "No teams found in current session." |
| Malformed config.json | Skip team, log warning to user |
| PID verification fails | Mark team as FAILED with stale indicator |

## Implementation Notes

- Use Bash tool for directory enumeration and PID verification
- Use Read tool for config.json files
- Calculate all derived fields before grouping
- Verify PIDs before categorizing as RUNNING
- Format monetary values to exactly 2 decimal places
- Use consistent timestamp parsing (ISO-8601)
- Handle missing fields gracefully (null checks)

## Cost Estimation

Typical execution:
- Bash (directory enumeration): ~$0.0001
- Read (5-10 config files): ~$0.001
- Processing and formatting: ~$0.0005
- **Total: ~$0.002 per invocation**
