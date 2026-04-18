# Quick Reference: Permission Handling in goyoke

## TL;DR

**goyoke pre-approves common tools so Claude can work without permission prompts.**

Allowed tools:
- `Bash` - Run shell commands
- `Read` - Read files
- `Write` - Create/overwrite files
- `Edit` - Edit existing files
- `Glob` - Find files by pattern
- `Grep` - Search file contents
- `Task` - Spawn sub-agents
- `TaskOutput` - Collect agent results

## What This Means for You

✅ **Good:**
- Claude can create files immediately
- No interruption for permissions
- Fast, automated workflows
- Predictable behavior

⚠️ **Be Aware:**
- Claude can modify any file in working directory
- Claude can execute shell commands
- Review suggested changes before running
- Use version control (git) for safety

## How It Works

When goyoke starts Claude CLI, it uses:
```bash
claude --allowed-tools Bash --allowed-tools Read --allowed-tools Write ...
```

This pre-authorizes tools for the session.

## Security Considerations

### Safe Tools
- `Read`, `Glob`, `Grep` - Read-only, safe
- `Task`, `TaskOutput` - Meta-operations, safe

### Powerful Tools
- `Write` - Can create/overwrite files
- `Edit` - Can modify existing files
- `Bash` - Can run arbitrary commands

### Best Practices

1. **Use git:** Always have uncommitted changes backed up
2. **Review first:** Read Claude's plan before executing
3. **Work in safe directories:** Don't run in system folders
4. **Ask for explanations:** "Explain what this Bash command does"

## Future Features

Coming soon:
- [ ] UI to view allowed tools
- [ ] Option to revoke tools during session
- [ ] Confirmation prompt before Bash execution
- [ ] Audit log of all tool uses
- [ ] Per-session tool policies

## Troubleshooting

### "Permission denied" errors

If you see permission errors:
1. Check file/directory permissions: `ls -la`
2. Verify working directory: Not system-protected location
3. Check disk space: `df -h`

These are **OS permission errors**, not Claude permission errors.

### Want to restrict tools?

Currently requires code change. Edit `cmd/goyoke/main.go:101`:
```go
AllowedTools: []string{"Read", "Glob", "Grep"}, // Read-only mode
```

Then rebuild:
```bash
go build -o ~/.local/bin/goyoke ./cmd/goyoke
```

### Want to allow more tools?

See Claude CLI documentation:
```bash
claude --help | grep -A 5 "allowed-tools"
```

Add to AllowedTools list and rebuild.

## Technical Details

For developers, see:
- `docs/PERMISSION_HANDLING.md` - Full investigation report
- `docs/TASK-2-COMPLETION-REPORT.md` - Implementation details
- `internal/cli/subprocess.go:62-65` - AllowedTools field
- `cmd/goyoke/main.go:101` - Default configuration
