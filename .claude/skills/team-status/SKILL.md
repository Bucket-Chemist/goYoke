---
name: team-status
description: Display detailed progress for running or completed orchestration teams
version: 1.0.0
---

# Team Status Skill

Display detailed progress for running or completed orchestration teams from the current session.

## Usage

```bash
/team-status                  # Show all teams in current session
/team-status <team-name>      # Show specific team
```

## Implementation Instructions

When this skill is invoked, follow these steps:

### 1. Parse Arguments

```javascript
const args = userInput.trim().split(/\s+/).slice(1); // Remove "/team-status"
const targetTeam = args.length > 0 ? args.join(' ') : null;
```

### 2. Discover Session Directory

```javascript
// Resolve gogent session directory from .gogent/current-session: env var → current-session marker → fallback
Bash({
    command: `project_root="$(git rev-parse --show-toplevel 2>/dev/null || echo .)"; gogent_session_dir="$(cat "$project_root/.gogent/current-session" 2>/dev/null)"; gogent_session_dir="${gogent_session_dir:-$project_root/.gogent/sessions/unknown}"; echo "$gogent_session_dir"`,
    description: "Resolve gogent session directory from .gogent/current-session"
})
// Verify it exists
Bash({command: `test -d "${gogentSessionDir}" && echo "exists" || echo "missing"`})
// If missing, error and exit
```

### 3. Find Team Directories

```javascript
Bash({
    command: `find ${gogentSessionDir}/teams -maxdepth 1 -type d -name "*.*" | sort`,
    description: "List all team directories"
})
// Each directory format: {timestamp}.{name}
```

### 4. Read Team Configurations

For each team directory (or just the target team if specified):

```javascript
Read({file_path: `${teamDir}/config.json`})
```

Parse the JSON structure:
- team_name: string
- workflow_type: "braintrust" | "review" | "implementation"
- status: "pending" | "running" | "completed" | "failed"
- background_pid: integer | null
- created_at, started_at, completed_at: ISO-8601 timestamps
- budget_max_usd, budget_remaining_usd: numbers
- waves[]: array of wave objects
  - wave_number: integer
  - description: string
  - members[]: array of agent objects
    - name: string
    - agent: string
    - status: "pending" | "running" | "completed" | "failed" | "retrying"
    - started_at, completed_at: ISO-8601 | null
    - cost_usd: number
    - retry_count: integer
    - error_message: string | null
    - process_pid: integer | null

### 5. Check Process Liveness

For teams with `background_pid` set:

```javascript
Bash({
    command: `kill -0 ${background_pid} 2>/dev/null && echo "alive" || echo "dead"`,
    description: "Check if background process is running"
})
```

For agents with `process_pid` set, same check.

### 6. Check Heartbeat Freshness

```javascript
Bash({
    command: `stat -c %Y ${teamDir}/heartbeat.json 2>/dev/null || echo "0"`,
    description: "Get heartbeat file modification timestamp"
})

const now = Math.floor(Date.now() / 1000);
const age = now - heartbeatTimestamp;
```

Interpret staleness:
- age < 30s: normal (no warning)
- age 30-60s: "⚠️ heartbeat: Xs ago (delayed)"
- age 60-120s: "⚠️ heartbeat: Xs ago (stale — process may be hung)"
- age > 120s: "⛔ heartbeat: Xs ago (likely dead)"

### 7. Format Output

For each team, produce output in this format:

```
Team: {team_name}
Workflow: {workflow_type}
Status: {STATUS_LABEL} {background_info}
{start_time_if_any}
Budget: ${budget_spent} / ${budget_max} ({percent}% remaining)

Wave {N}: {WAVE_STATUS} ({completed_count}/{total_count} agents)
  {status_icon} {agent_name:<12} [{status}]  Start: {time}  {duration_or_elapsed}  Cost: ${cost}  Retries: {count}
  ...

{heartbeat_warning_if_any}
```

**Status Labels:**
- Team status "running" + background_pid alive → "EXECUTING (background PID {pid})"
- Team status "running" + background_pid dead → "STALE (process {pid} died)"
- Team status "completed" → "COMPLETED"
- Team status "failed" → "FAILED"
- Team status "pending" → "PENDING"

**Wave Status:**
- All members completed → "COMPLETED"
- Any member running → "RUNNING"
- Any member failed → "FAILED"
- All members pending → "PENDING"

**Status Icons:**

Normal mode (Unicode):
- completed: ✓
- running: ⏳
- pending: ⏸
- failed: ✗
- retrying: 🔄

ASCII mode (when TERM=dumb or GOGENT_ASCII=1):
- completed: [OK]
- running: [..]
- pending: [--]
- failed: [XX]
- retrying: [>>]

**Time Formatting:**

Durations:
- < 60s: "{seconds}s" (e.g., "45s")
- 60-3600s: "{minutes}m {seconds}s" (e.g., "2m 14s")
- >= 3600s: "{hours}h {minutes}m {seconds}s" (e.g., "1h 23m 14s")

Timestamps:
- Absolute: "2026-02-06 14:30:22" (ISO format, 24-hour, local time)
- Relative: "{time} ago" where time uses duration format

Costs:
- Always 2 decimal places with $ prefix: "$3.42"

Budget percentage:
- `((budget_remaining_usd / budget_max_usd) * 100).toFixed(0)`

### 8. Error Handling

**Team not found (when specific team requested):**
```
Error: Team not found: {team_name}
Available teams:
  - {team1_name}
  - {team2_name}
  ...
```

**No teams in session:**
```
No teams in current session.

Use /braintrust, /review, or /ticket to start an orchestration team.
```

**Session directory invalid:**
```
Error: Session directory not found: {path}

The session may have been moved or deleted. Check GOGENT_SESSION_DIR environment variable.
```

**Config parsing error:**
```
Error: Failed to parse config for team {team_name}: {error_message}
```

### 9. Multi-Team Display

When showing all teams (no specific target):

```
=== Orchestration Teams (Session: {session_id}) ===

{team1_output}

---

{team2_output}

---

Total: {count} teams
```

### 10. ASCII Detection

Before formatting, detect ASCII mode:

```javascript
Bash({command: `echo "$TERM"`})
Bash({command: `echo "${GOGENT_ASCII:-0}"`})

const asciiMode = (term === "dumb") || (gogentAscii === "1");
```

Use appropriate icon set based on `asciiMode`.

## Example Output

### Single Team (Running)

```
Team: braintrust-1738856422
Workflow: braintrust
Status: EXECUTING (background PID 12345)
Started: 2026-02-06 14:30:22
Budget: $3.42 / $15.00 (77% remaining)

Wave 1: COMPLETED (2/2 agents)
  ✓ einstein      [completed]  Start: 14:31:05  Duration: 2m 14s  Cost: $1.52  Retries: 0
  ✓ staff-arch    [completed]  Start: 14:31:05  Duration: 1m 58s  Cost: $1.34  Retries: 0

Wave 2: RUNNING (0/1 agents)
  ⏳ beethoven    [running]    Start: 14:33:24  Elapsed: 45s      Cost: $0.56  Retries: 0

Next check: heartbeat updated 3s ago
```

### Multiple Teams

```
=== Orchestration Teams (Session: 260206.001) ===

Team: review-1738855000
Workflow: review
Status: COMPLETED
Started: 2026-02-06 14:10:00
Completed: 2026-02-06 14:15:42
Budget: $2.18 / $10.00 (78% remaining)

Wave 1: COMPLETED (3/3 agents)
  ✓ backend-rev   [completed]  Duration: 3m 22s  Cost: $0.84  Retries: 0
  ✓ frontend-rev  [completed]  Duration: 2m 58s  Cost: $0.72  Retries: 0
  ✓ standards-rev [completed]  Duration: 3m 10s  Cost: $0.62  Retries: 0

---

Team: braintrust-1738856422
Workflow: braintrust
Status: EXECUTING (background PID 12345)
Started: 2026-02-06 14:30:22
Budget: $3.42 / $15.00 (77% remaining)

Wave 1: COMPLETED (2/2 agents)
  ✓ einstein      [completed]  Duration: 2m 14s  Cost: $1.52  Retries: 0
  ✓ staff-arch    [completed]  Duration: 1m 58s  Cost: $1.34  Retries: 0

Wave 2: RUNNING (0/1 agents)
  ⏳ beethoven    [running]    Elapsed: 45s      Cost: $0.56  Retries: 0

⚠️ heartbeat: 3s ago

---

Total: 2 teams
```

### Team with Errors

```
Team: implementation-1738857000
Workflow: implementation
Status: FAILED
Started: 2026-02-06 14:50:00
Failed: 2026-02-06 14:52:15
Budget: $1.20 / $20.00 (94% remaining)

Wave 1: FAILED (1/2 agents)
  ✓ architect     [completed]  Duration: 1m 45s  Cost: $0.92  Retries: 0
  ✗ python-pro    [failed]     Duration: 30s     Cost: $0.28  Retries: 2
    Error: Build failed: syntax error in models/attention.py line 42

⛔ heartbeat: 125s ago (likely dead)
```

## Notes

- This skill is READ-ONLY. It does not modify team state.
- Process checks use `kill -0` which only checks existence, not health.
- Heartbeat freshness is the best indicator of active progress.
- Cost tracking depends on agents reporting their usage accurately.
- All timestamps assume local timezone unless otherwise specified.
