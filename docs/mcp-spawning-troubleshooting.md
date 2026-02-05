# MCP Agent Spawning Troubleshooting Guide

Complete troubleshooting reference for the MCP spawn_agent tool.

**Source:** MCP-SPAWN-012 Documentation Requirements

---

## Common Issues

### 1. "spawn_agent tool not found"

**Cause**: MCP server not registered or GOGENT_MCP_SPAWN_ENABLED=false

**Solution**:
```bash
# Check MCP server status
ps aux | grep gofortress

# Verify environment variable
echo $GOGENT_MCP_SPAWN_ENABLED  # Should be "true"

# Restart MCP server
pkill -f gofortress
# Restart Claude CLI session
```

**Verification**:
```bash
# Check if MCP server is listening
lsof -i :3000  # Or whatever port is configured
```

---

### 2. "agents-index.json not found"

**Cause**: Missing agents index file

**Solution**:
```bash
# Check file exists
ls -la ~/.claude/agents/agents-index.json

# If missing, regenerate
cd ~/.claude/agents && ./generate-index.sh

# Verify contents
cat ~/.claude/agents/agents-index.json | jq '.agents | length'
```

**Prevention**: Add agents-index.json generation to pre-commit hook

---

### 3. "Permission denied on hooks"

**Cause**: Hook binaries not executable

**Solution**:
```bash
# Fix hook permissions
chmod +x ~/.claude/hooks/gogent-*

# Verify
ls -la ~/.claude/hooks/gogent-*

# All should show -rwxr-xr-x
```

---

### 4. "Nesting level exceeded"

**Cause**: Trying to spawn from level 2+ (max depth is 10)

**Error Message**:
```json
{
  "success": false,
  "error": "Maximum nesting depth (10) exceeded. Current level: 10. Cannot spawn sub-agent at this depth.",
  "errorCode": "E_MAX_DEPTH_EXCEEDED"
}
```

**Solution**:
This is intentional enforcement. Redesign workflow to avoid deep nesting.

**Workaround**: Use orchestrator pattern instead of recursive spawning:

```
❌ WRONG (Deep Nesting):
Router → Mozart → Einstein → Sub-Einstein → Sub-Sub-Einstein

✅ RIGHT (Flat Orchestration):
Router → Mozart → [Einstein, Staff-Architect, Beethoven] (parallel)
```

---

### 5. "Timeout exceeded"

**Cause**: Agent took longer than configured timeout

**Error Message**:
```json
{
  "success": false,
  "error": "Agent timed out after 300000ms"
}
```

**Solution**:

1. **Increase timeout** in spawn_agent call:
   ```javascript
   mcp__gofortress__spawn_agent({
     agent: "einstein",
     prompt: "...",
     timeout: 600000  // 10 minutes instead of 5
   });
   ```

2. **Check if agent is blocked** on user input:
   ```bash
   # Review agent logs
   tail -f /tmp/spawn-einstein-*.log

   # Look for AskUserQuestion or modal prompts
   ```

3. **Review agent logs** in `/tmp/spawn-{agent-id}-{timestamp}.log`:
   ```bash
   # Find recent spawn logs
   ls -lt /tmp/spawn-*.log | head -5

   # Check for infinite loops or blocking operations
   grep -i "waiting\|blocked\|stuck" /tmp/spawn-*.log
   ```

---

### 6. "Invalid JSON in agent output"

**Cause**: CLI output contains non-JSON text

**Error Manifestation**:
Agent succeeds but output is unparsed raw text instead of structured JSON.

**Solution**:

1. **Check CLI version**:
   ```bash
   claude --version
   # Should be latest version
   ```

2. **Review raw CLI output** in spawn logs:
   ```bash
   tail -100 /tmp/spawn-{agent-id}-*.log
   ```

3. **Agent may have printed debug output** - update agent to use proper logging:
   ```javascript
   // ❌ WRONG: Prints to stdout
   console.log("Debug message");

   // ✅ RIGHT: Uses logger
   logger.debug("Debug message", { context });
   ```

---

### 7. "Spawn validation failed"

**Cause**: Parent agent not allowed to spawn requested child agent

**Error Message**:
```json
{
  "success": false,
  "error": "Spawn validation failed: Parent 'python-pro' cannot spawn child 'einstein'",
  "validationErrors": [
    {
      "code": "E_UNAUTHORIZED_SPAWN",
      "message": "Parent 'python-pro' cannot spawn child 'einstein'"
    }
  ]
}
```

**Solution**:

Check delegation rules in agent definition. Example from `mozart.md`:

```yaml
delegation:
  can_spawn:
    - haiku-scout
    - codebase-search
    - einstein
    - staff-architect-critical-review
    - beethoven
  cannot_spawn:
    - mozart  # Cannot spawn self
    - orchestrator
```

**Fix**: Either update delegation rules or use different parent agent.

---

### 8. "Output truncated - exceeded 10MB limit"

**Cause**: Agent output exceeded MAX_BUFFER_SIZE (10MB)

**Error Manifestation**:
```json
{
  "success": true,
  "output": "... [OUTPUT TRUNCATED - exceeded 10MB limit]",
  "truncated": true
}
```

**Solution**:

1. **Reduce output size** - agent should write large data to files, not stdout:
   ```javascript
   // ❌ WRONG: Output 10MB JSON to stdout
   console.log(JSON.stringify(largeData));

   // ✅ RIGHT: Write to file, return path
   Write({
     file_path: "/tmp/large-data.json",
     content: JSON.stringify(largeData)
   });
   console.log(JSON.stringify({
     result: "Data written",
     path: "/tmp/large-data.json"
   }));
   ```

2. **Use streaming** for large datasets instead of buffering

---

## Debugging Workflow

### Step 1: Check MCP Server Logs

```bash
# Tail MCP server logs
tail -f /tmp/mcp-gofortress.log

# Look for:
# - Tool registration errors
# - Connection failures
# - Runtime exceptions
```

### Step 2: Check Spawn Logs

```bash
# List recent spawn logs
ls -lt /tmp/spawn-*.log | head -10

# View specific spawn log
tail -f /tmp/spawn-mozart-2026-02-05T14-30-00.log
```

### Step 3: Test Spawn Manually

```bash
# From Claude CLI, test Braintrust workflow
claude -p << 'EOF'
Run /braintrust with this test problem:
"Design a caching strategy for API responses"
EOF
```

### Step 4: Verify Environment

```bash
# Check all GOGENT variables
env | grep GOGENT

# Should see:
# GOGENT_NESTING_LEVEL=0
# GOGENT_MCP_SPAWN_ENABLED=true
# GOGENT_PARENT_AGENT=(not set at level 0)
```

### Step 5: Check Agent Index

```bash
# Verify agent exists
cat ~/.claude/agents/agents-index.json | jq '.agents[] | select(.id == "einstein")'

# Should return agent definition
```

---

## Environment Setup Verification

### Prerequisites Checklist

- [ ] Node.js ≥20.x installed (`node --version`)
- [ ] npm ≥10.x installed (`npm --version`)
- [ ] Claude CLI installed (`claude --version`)
- [ ] Go ≥1.21 installed (`go version`)
- [ ] `.claude` directory exists (`ls ~/.claude`)
- [ ] Hooks are executable (`ls -la ~/.claude/hooks/`)
- [ ] MCP server binary exists (`ls ~/.claude/bin/gofortress`)

### Environment Variables

Create `.env.local` in project root:

```bash
# Required for MCP spawning
GOGENT_NESTING_LEVEL=0
GOGENT_MCP_SPAWN_ENABLED=true

# Optional: Test mode
GOGENT_TEST_MODE=false

# Optional: Custom paths
CLAUDE_PROJECT_DIR=/path/to/project
XDG_DATA_HOME=~/.local/share
XDG_RUNTIME_DIR=/run/user/$(id -u)
```

Load environment:
```bash
source .env.local
```

---

## Common Error Patterns

### Pattern 1: Agent Spawns but Immediately Fails

**Symptoms**:
- Agent ID returned
- Success: false
- Error: "Exit code 1"

**Debug Steps**:
1. Check agent's own logs in `/tmp/spawn-{agent-id}-*.log`
2. Look for missing dependencies or configuration
3. Verify agent has access to required tools
4. Check if agent's model is available (e.g., opus quota exceeded)

### Pattern 2: Agent Hangs/Never Returns

**Symptoms**:
- Agent spawns
- Never completes (times out)
- Logs show agent is "waiting for user input"

**Debug Steps**:
1. Check if agent uses AskUserQuestion - this blocks until user responds
2. Review agent's prompt for ambiguous instructions
3. Check if agent entered infinite loop
4. Verify agent's stop condition is reachable

### Pattern 3: Cost Not Attributed

**Symptoms**:
- Agent completes successfully
- Session cost summary missing spawn costs

**Debug Steps**:
1. Verify CLI output includes `cost_usd` field:
   ```bash
   grep -i "cost_usd" /tmp/spawn-*.log
   ```

2. Check if cost tracker is initialized:
   ```javascript
   const tracker = getSessionCostTracker();
   console.log(tracker.getSummary());
   ```

3. Verify parseCliOutput extracts cost correctly

---

## Performance Debugging

### Slow Spawn Times

**Normal spawn times**:
- Haiku agents: <5 seconds
- Sonnet agents: 10-30 seconds
- Opus agents: 30-120 seconds

**If exceeding these**:

1. **Check model availability**:
   ```bash
   # Test API latency
   time claude -p "Hello" --model opus
   ```

2. **Check agent's tool usage**:
   ```bash
   # Count tool calls in spawn log
   grep -c "tool_use" /tmp/spawn-*.log
   ```

3. **Reduce agent scope** - overly broad prompts cause long runtimes

---

## Emergency Recovery

### Kill Stuck Spawns

```bash
# Find all spawned claude processes
ps aux | grep "claude.*--output-format json"

# Kill specific spawn
kill -TERM <pid>

# Force kill if needed
kill -KILL <pid>
```

### Reset MCP State

```bash
# Stop MCP server
pkill -f gofortress

# Clear spawn registry
rm -f /tmp/spawn-registry-*.json

# Restart Claude CLI session
# MCP server will auto-restart
```

### Clear Spawn Logs

```bash
# Remove old spawn logs
find /tmp -name "spawn-*.log" -mtime +7 -delete

# Keep last 50 only
ls -t /tmp/spawn-*.log | tail -n +51 | xargs rm -f
```

---

## Architecture Reference

See `tickets/mcp-agent-teams-integration/mcp-spawn/mcp-spawning-v3.md` for full architecture.

**Key Components**:
- **spawnAgent.ts** - MCP tool implementation
- **relationshipValidation.ts** - Parent-child validation
- **processRegistry.ts** - Process lifecycle tracking
- **tracker.ts** - Cost attribution

**Data Flow**:
```
1. spawn_agent tool called
2. Validate nesting depth
3. Validate parent-child relationship
4. Spawn CLI process with environment
5. Collect stdout/stderr
6. Parse JSON output
7. Extract cost data
8. Add to session tracker
9. Return result to caller
```

---

## Getting Help

If issue persists:

1. **Collect diagnostic info**:
   ```bash
   # Save environment
   env | grep GOGENT > /tmp/gogent-env.txt

   # Save recent spawn logs
   tar czf /tmp/spawn-logs.tar.gz /tmp/spawn-*.log

   # Save MCP server log
   cp /tmp/mcp-gofortress.log /tmp/mcp-diagnostics.log
   ```

2. **Check for known issues** in `tickets/mcp-agent-teams-integration/mcp-spawn/`

3. **Create minimal reproduction** using `/braintrust` skill

4. **Document**:
   - What you tried
   - Expected result
   - Actual result
   - Full error message
   - Spawn log excerpt

---

## Related Documentation

- **MCP Spawning Architecture**: `tickets/mcp-agent-teams-integration/mcp-spawn/mcp-spawning-v3.md`
- **Mozart Agent Definition**: `~/.claude/agents/mozart/mozart.md`
- **Review-Orchestrator Definition**: `~/.claude/agents/review-orchestrator/review-orchestrator.md`
- **CLAUDE.md MCP Section**: `~/.claude/CLAUDE.md` (search for "MCP Agent Spawning")
- **Cost Tracking**: `packages/tui/src/cost/tracker.ts`
