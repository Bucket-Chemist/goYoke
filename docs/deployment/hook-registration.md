# Hook Registration: goyoke-archive

## Prerequisites

1. Build binary: `make build-archive`
2. Install to PATH: `make install-archive`
3. Verify: `which goyoke-archive` should show `~/.local/bin/goyoke-archive`

## Claude Code Hook Configuration

### Option 1: Global Installation (Recommended)

If `~/.local/bin` is in PATH:

```toml
# In Claude Code hook config (exact location varies)
[hooks.SessionEnd]
command = "goyoke-archive"
```

### Option 2: Absolute Path

If not in PATH:

```toml
[hooks.SessionEnd]
command = "/home/username/.local/bin/goyoke-archive"
```

### Option 3: Project-Local

For development/testing:

```toml
[hooks.SessionEnd]
command = "/path/to/goYoke/bin/goyoke-archive"
```

## Testing Hook Integration

```bash
# Simulate SessionEnd event
echo '{"session_id":"test-123","timestamp":1234567890,"hook_event_name":"SessionEnd"}' | goyoke-archive

# Expected output: JSON confirmation
# Expected side effects:
# - .claude/memory/handoffs.jsonl appended
# - .claude/memory/last-handoff.md created
# - Artifacts moved to session-archive/
```

## Rollback Procedure

If Go hook fails in production:

1. Edit hook config:
   ```toml
   [hooks.SessionEnd]
   command = "~/.claude/hooks/session-archive.sh"
   ```

2. Verify bash hook works:
   ```bash
   echo '{"session_id":"test","timestamp":123,"hook_event_name":"SessionEnd"}' | ~/.claude/hooks/session-archive.sh
   ```

3. Check output files created

4. Report Go hook failure for debugging
