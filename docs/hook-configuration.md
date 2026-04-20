---
title: Hook Configuration
type: guide
tags: [hooks, configuration, settings]
related: [concepts/hook-system]
created: 2026-04-18
---
# Hook Configuration

This document shows how to configure goYoke hooks in Claude Code.

## Settings.json Configuration

Add to your Claude Code settings (`~/.claude/settings.json`):

```json
{
  "hooks": {
    "SessionStart": {
      "command": "goyoke-load-context"
    },
    "PreToolUse": {
      "command": "goyoke-validate"
    },
    "PostToolUse": {
      "command": "goyoke-sharp-edge",
      "tools": ["Bash", "Edit", "Write"]
    },
    "SessionEnd": {
      "command": "goyoke-archive"
    }
  }
}
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GOYOKE_PROJECT_DIR` | Override project directory | Current working directory |
| `CLAUDE_PROJECT_DIR` | Fallback project directory | Current working directory |
| `GOYOKE_ROUTING_SCHEMA` | Path to routing-schema.json | `~/.claude/routing-schema.json` |
| `XDG_CACHE_HOME` | XDG cache directory | `~/.cache` |

## Verifying Installation

```bash
# Check binaries are installed
which goyoke-load-context goyoke-validate goyoke-archive

# Test SessionStart hook manually
echo '{"type":"startup","session_id":"test","hook_event_name":"SessionStart"}' | goyoke-load-context

# Expected output: JSON with hookSpecificOutput containing session context
```

## Troubleshooting

### Hook not executing
- Verify binary is in PATH: `which goyoke-load-context`
- Check permissions: `ls -la $(which goyoke-load-context)`
- Test manually with echo | pipe

### Missing routing schema
- Expected at: `~/.claude/routing-schema.json`
- Hook will warn but continue without routing summary

### Tool counter not created
- Check XDG_CACHE_HOME or ~/.cache/goyoke/ permissions
- Non-fatal - session continues but attention-gate won't work


---

## See Also

- [[concepts/hook-system]] — Hook lifecycle and event types
- [[ARCHITECTURE#2. Hook Event Flow]] — Event flow diagrams
- [[concepts/routing-tiers]] — How hooks enforce tier selection
