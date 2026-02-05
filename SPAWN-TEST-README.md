# MCP spawn_agent Testing Guide

## Quick Start

```bash
cd ~/Documents/GOgent-Fortress
./test-mcp-spawn.sh
```

This launches a tmux session with 4 panes showing:
1. **TUI Output** (top-left) - The running TUI
2. **Process Monitor** (top-right) - Active Claude processes
3. **Nesting Watch** (bottom-left) - Nesting level tracking
4. **Event Log** (bottom-right) - Spawn events in real-time

## Tmux Controls

| Key | Action |
|-----|--------|
| `Ctrl+B` then `←→↑↓` | Navigate between panes |
| `Ctrl+B` then `d` | Detach (keeps running) |
| `Ctrl+B` then `[` | Scroll mode (q to exit) |
| Type in TUI pane | Send prompts to Claude |

## Test Prompts

### Test 1: Simple Single Agent Spawn (Easiest)

Copy-paste this into the TUI pane:

```
Please use the spawn_agent tool to spawn a codebase-search agent to find all
TypeScript files in packages/tui/src/mcp that contain the word "spawn".

Use these parameters:
- agent: "codebase-search"
- model: "haiku"
- description: "Find spawn references in MCP code"
```

**What to watch:**
- Process Monitor: Should show +1 process, then back to baseline
- Nesting Watch: Should show LEVEL:1 appear and disappear
- Duration: ~5-10 seconds
- Cost: ~$0.01-0.02

### Test 2: Quick Haiku Verification

```
Use spawn_agent to spawn a haiku agent with this simple task:
"Count from 1 to 5 and return the numbers as a list"

Parameters:
- agent: "general-purpose"
- model: "haiku"
- description: "Quick test"
```

**What to watch:**
- Fastest test (<5 seconds)
- Minimal cost (<$0.01)
- Single process spike

### Test 3: Nested Orchestrator (Advanced)

```
I want to test nested agent spawning. Please use spawn_agent to spawn a
review-orchestrator that will review the MCP-SPAWN-009 implementation.

Files to review:
- packages/tui/src/mcp/server.ts
- packages/tui/src/index.tsx
- packages/tui/src/mcp/server.test.ts

Use:
- agent: "review-orchestrator"
- model: "sonnet"
- description: "Review MCP-SPAWN-009 implementation"
```

**What to watch:**
- Process Monitor: Should show 2-5 processes (orchestrator + reviewers)
- Nesting Watch: LEVEL:1 (orchestrator), LEVEL:2 (reviewers)
- Duration: ~30-60 seconds
- Cost: ~$0.30-0.50

## Expected Output Format

When spawn_agent completes, you should see JSON like:

```json
{
  "agentId": "abc-123-def-456",
  "agent": "codebase-search",
  "success": true,
  "output": "Found 3 files:\n- spawnAgent.ts\n- server.ts\n- server.test.ts",
  "cost": 0.015,
  "turns": 2,
  "duration": 3421,
  "truncated": false
}
```

## Success Indicators

✅ **spawn_agent tool called successfully**
- Check TUI output for tool invocation
- No "tool not found" errors

✅ **Process spawned correctly**
- Process Monitor shows increase in process count
- Nesting Watch shows GOGENT_NESTING_LEVEL incremented

✅ **Clean termination**
- Process count returns to baseline
- No orphaned processes (`ps aux | grep "claude -p"`)

✅ **Valid output returned**
- JSON structure with agentId, success, output
- Cost and duration populated

## Troubleshooting

### "Tool spawn_agent not found"
```bash
# Check feature flag
echo $GOGENT_MCP_SPAWN_ENABLED  # Should be empty or "true"

# If "false", unset it:
unset GOGENT_MCP_SPAWN_ENABLED

# Restart TUI (Ctrl+C in TUI pane, then):
npm start
```

### Processes not appearing in monitor
```bash
# Verify TUI is running
ps aux | grep "node.*dist/index.js"

# Manually check for claude processes
ps aux | grep "claude -p"
```

### Orphaned processes after test
```bash
# Kill all Claude spawned processes
pkill -f "claude -p"

# Verify cleanup
ps aux | grep claude
```

### TUI won't start
```bash
# Rebuild TUI
cd ~/Documents/GOgent-Fortress/packages/tui
npm run build
npm start
```

## Log Analysis

After testing, analyze the session:

```bash
cd ~/Documents/GOgent-Fortress/.test-spawn-logs
./analyze-session.sh
```

View logs:
```bash
# TUI output
cat tui_output_*.log

# Process activity
cat processes_*.log

# All logs
ls -lh
```

## Advanced: Manual Monitoring

If you prefer manual monitoring outside tmux:

**Terminal 1: Start TUI**
```bash
cd ~/Documents/GOgent-Fortress/packages/tui
npm start
```

**Terminal 2: Monitor processes**
```bash
watch -n 0.5 'ps aux | grep claude | grep -v grep | wc -l'
```

**Terminal 3: Watch nesting levels**
```bash
while true; do
  ps e -o pid,cmd | grep GOGENT_NESTING_LEVEL | grep -v grep
  sleep 1
done
```

## Cleanup

Stop the test session:
```bash
# Detach first (Ctrl+B then d) if attached, then:
tmux kill-session -t mcp-spawn-test

# Or kill all tmux sessions:
tmux kill-server
```

Remove logs:
```bash
rm -rf ~/Documents/GOgent-Fortress/.test-spawn-logs
```

## What Each Test Validates

| Test | Validates |
|------|-----------|
| Simple spawn | Basic spawn_agent functionality, CLI invocation |
| Haiku test | Fast execution, minimal cost, proper cleanup |
| Orchestrator | Nested spawning, Level 2+ agents, parallel execution |

## Expected Behavior Summary

```
Level 0 (TUI)
  └─ spawn_agent → Level 1 (spawned CLI agent)
       └─ Task tool → Level 2 (sub-agents if orchestrator)
```

- **Level 0:** TUI running MCP server with spawn_agent tool
- **Level 1:** Spawned Claude CLI process with normal tool access
- **Level 2:** Sub-agents spawned by Level 1 using Task tool
- **Max depth:** 10 levels (enforced by validateNestingDepth)

## Reference

- Implementation: `packages/tui/src/mcp/tools/spawnAgent.ts`
- Tests: `packages/tui/src/mcp/server.test.ts`
- Ticket: `tickets/mcp-agent-teams-integration/mcp-spawn/MCP-SPAWN-009.md`
