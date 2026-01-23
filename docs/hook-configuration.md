# Hook Configuration

This document shows how to configure GOgent-Fortress hooks in Claude Code.

## Settings.json Configuration

Add to your Claude Code settings (`~/.claude/settings.json`):

```json
{
  "hooks": {
    "SessionStart": {
      "command": "gogent-load-context"
    },
    "PreToolUse": {
      "command": "gogent-validate"
    },
    "PostToolUse": {
      "command": "gogent-sharp-edge",
      "tools": ["Bash", "Edit", "Write"]
    },
    "SessionEnd": {
      "command": "gogent-archive"
    }
  }
}
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GOGENT_PROJECT_DIR` | Override project directory | Current working directory |
| `CLAUDE_PROJECT_DIR` | Fallback project directory | Current working directory |
| `GOGENT_ROUTING_SCHEMA` | Path to routing-schema.json | `~/.claude/routing-schema.json` |
| `XDG_CACHE_HOME` | XDG cache directory | `~/.cache` |

## Verifying Installation

```bash
# Check binaries are installed
which gogent-load-context gogent-validate gogent-archive

# Test SessionStart hook manually
echo '{"type":"startup","session_id":"test","hook_event_name":"SessionStart"}' | gogent-load-context

# Expected output: JSON with hookSpecificOutput containing session context
```

## Troubleshooting

### Hook not executing
- Verify binary is in PATH: `which gogent-load-context`
- Check permissions: `ls -la $(which gogent-load-context)`
- Test manually with echo | pipe

### Missing routing schema
- Expected at: `~/.claude/routing-schema.json`
- Hook will warn but continue without routing summary

### Tool counter not created
- Check XDG_CACHE_HOME or ~/.cache/gogent/ permissions
- Non-fatal - session continues but attention-gate won't work
