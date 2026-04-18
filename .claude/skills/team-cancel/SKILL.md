---
name: team-cancel
description: Gracefully stop a running orchestration team
version: 1.0.0
---

# Team Cancel

Gracefully stops a running orchestration team by sending SIGTERM to the goyoke-team-run process. The process implements a graceful shutdown cascade: it catches SIGTERM, forwards it to all child Claude processes, waits up to 5 seconds for children to exit, sends SIGKILL to any remaining children, and cleans up.

## Usage

```bash
/team-cancel <team-name>
```

**Arguments:**
- `<team-name>` (REQUIRED): Name of the team to cancel (e.g., "braintrust-1738856422")

## Implementation

When invoked, follow these steps:

### 1. Parse Arguments

Team name is REQUIRED. If not provided:
```
Error: Team name is required
Usage: /team-cancel <team-name>
```

### 2. Discover Session Directory

Check in order:
1. Environment variable `GOYOKE_SESSION_DIR`
2. Read `{project_root}/.goyoke/current-session` marker file
3. Fallback: `{project_root}/.goyoke/sessions/unknown`

If no session found:
```
Error: No active session directory found
```

### 3. Locate Team Directory

Team directory pattern: `{goyoke_session_dir}/teams/{team_name}/`

If directory doesn't exist:
```bash
# List available teams
ls -1 {goyoke_session_dir}/teams/ 2>/dev/null || echo "none"
```

Output:
```
Error: Team not found: [name]
Available teams: [list from ls, or "none"]
```

### 4. Read Team Configuration

Read `{team_dir}/config.json`:

```bash
cat {team_dir}/config.json
```

Extract `background_pid` field. If null or missing:
```
Team is not running (no background PID recorded)
```
Exit with success (no action needed).

### 5. Verify Process Exists

Check if PID is still alive:
```bash
kill -0 ${PID} 2>/dev/null
```

If exit code != 0:
```
Team process not found (PID ${PID} is stale). Updating config...
```

Update config.json:
```bash
~/.generic-python/bin/python3 -c "
import json
import sys
from pathlib import Path
from datetime import datetime

config_path = Path('${team_dir}/config.json')
config = json.loads(config_path.read_text())
config['status'] = 'failed'
config['completed_at'] = datetime.now().isoformat()
config['background_pid'] = None
config_path.write_text(json.dumps(config, indent=2))
print('Config updated. Team marked as failed.')
"
```
Exit with success.

### 6. Send SIGTERM

```
Cancelling team: {team_name} (PID {background_pid})
Sent SIGTERM...
Waiting for graceful shutdown...
```

Send SIGTERM:
```bash
kill -TERM ${PID}
```

### 7. Wait for Graceful Shutdown

Check every 1 second for up to 10 seconds:

```bash
for i in {1..10}; do
  if ! kill -0 ${PID} 2>/dev/null; then
    echo "Team stopped successfully."
    break
  fi
  sleep 1
  if [ $i -eq 10 ]; then
    echo "Warning: Team did not stop gracefully after 10 seconds."
  fi
done
```

### 8. Handle Force Kill (if needed)

If process still alive after 10 seconds, ask user:
```
Warning: Team did not stop gracefully. Force kill? (y/n)
```

Wait for user input. If user confirms with "y" or "yes":
```bash
kill -9 ${PID}
echo "Sent SIGKILL."
sleep 1
if ! kill -0 ${PID} 2>/dev/null; then
  echo "Team forcefully terminated."
  # Mark as "killed" in config (step 9)
fi
```

If user declines:
```
Team cancel aborted. Process ${PID} still running.
```
Exit without updating config.

### 9. Update Configuration

After successful shutdown (graceful or forced):

```bash
~/.generic-python/bin/python3 -c "
import json
import sys
from pathlib import Path
from datetime import datetime

config_path = Path('${team_dir}/config.json')
config = json.loads(config_path.read_text())

# Set status based on shutdown type
status = '${shutdown_type}'  # 'cancelled' or 'killed'
config['status'] = status
config['completed_at'] = datetime.now().isoformat()
config['background_pid'] = None

config_path.write_text(json.dumps(config, indent=2))
"
```

Pass `shutdown_type`:
- `"cancelled"` for graceful SIGTERM shutdown
- `"killed"` for SIGKILL forced shutdown
- `"failed"` for stale PID (handled in step 5)

### 10. Display Final Status

Read final config and display summary:

```bash
~/.generic-python/bin/python3 -c "
import json
from pathlib import Path
from datetime import datetime

config_path = Path('${team_dir}/config.json')
config = json.loads(config_path.read_text())

# Format duration
started = datetime.fromisoformat(config['started_at'])
completed = datetime.fromisoformat(config['completed_at'])
duration_sec = int((completed - started).total_seconds())

if duration_sec < 60:
    duration_str = f'{duration_sec}s'
elif duration_sec < 3600:
    mins = duration_sec // 60
    secs = duration_sec % 60
    duration_str = f'{mins}m {secs}s'
else:
    hours = duration_sec // 3600
    mins = (duration_sec % 3600) // 60
    secs = duration_sec % 60
    duration_str = f'{hours}h {mins}m {secs}s'

# Display summary
print(f\"\"\"
Final status:
- Status: {config['status'].upper()}
- Cost: \${config['total_cost']:.2f} / \${config['cost_limit']:.2f}
- Duration: {duration_str}
\"\"\")

# Display wave status
waves = config['waves']
for wave in waves:
    completed_count = sum(1 for a in wave['agents'] if a['status'] == 'completed')
    total_count = len(wave['agents'])
    status = wave['status'].upper()

    running_agents = [a['name'] for a in wave['agents'] if a['status'] == 'running']
    running_note = f', {running_agents[0]} was running' if running_agents else ''

    print(f\"- Wave {wave['wave_number']}: {status} ({completed_count}/{total_count}{running_note})\")
"
```

## Signal Cascade Behavior

When goyoke-team-run receives SIGTERM:
1. Catches signal via Go signal.Notify
2. Sends SIGTERM to all child Claude processes (tracked in ProcessRegistry)
3. Waits up to 5 seconds for children to exit
4. Sends SIGKILL to any remaining children
5. Cleans up PID file at `{team_dir}/.pid`
6. Updates metrics and exits

This allows in-flight agents to complete their current tool call before terminating.

## Example Output

```
Cancelling team: braintrust-1738856422 (PID 12345)
Sent SIGTERM...
Waiting for graceful shutdown...
Team stopped successfully.

Final status:
- Status: CANCELLED
- Cost: $2.86 / $15.00
- Duration: 3m 12s
- Wave 1: COMPLETED (2/2)
- Wave 2: CANCELLED (0/1, beethoven was running)
```

## Error Cases

| Condition | Output |
|-----------|--------|
| No team name | "Error: Team name is required" |
| Team not found | "Error: Team not found: {name}. Available teams: {list}" |
| No PID recorded | "Team is not running (no background PID recorded)" |
| Stale PID | "Team process not found (PID ${PID} is stale). Updating config..." |
| Won't terminate | "Warning: Team did not stop gracefully. Force kill? (y/n)" |

## Configuration Updates

The skill updates `config.json` with:
- **status**: "cancelled" (graceful), "killed" (forced), or "failed" (stale PID)
- **completed_at**: ISO-8601 timestamp
- **background_pid**: null

## Notes

- Graceful shutdown allows running agents to finish their current tool call
- Force kill (SIGKILL) immediately terminates all processes
- Stale PIDs (process already dead) are marked as "failed"
- Cost accounting includes partial work before cancellation
