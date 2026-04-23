# Project Root Resolution

## Overview

`project_root` is resolved by the planner (Mozart, review-orchestrator, or impl-manager) at team creation time and written into `config.json`. The Go binary (`goyoke-team-run`) reads it and sets `cmd.Dir` for all spawned agent processes.

## Resolution Priority

1. **Explicit environment**: `GOYOKE_PROJECT_DIR`
2. **Claude CLI environment**: `CLAUDE_PROJECT_DIR`
3. **Current working directory**: `os.Getwd()`

## Implementation

### In Planning Agent (Mozart/Router)

```go
func resolveProjectRoot() string {
    if root := os.Getenv("GOYOKE_PROJECT_DIR"); root != "" {
        return root
    }
    if root := os.Getenv("CLAUDE_PROJECT_DIR"); root != "" {
        return root
    }
    root, _ := os.Getwd()
    return root
}
```

Write this to `config.json` at team creation.

### In Go Binary (goyoke-team-run)

Read from config.json and use for spawning:

```go
cmd := exec.Command("claude", args...)
cmd.Dir = config.ProjectRoot // Set working directory
```

## Validation

Team config schema enforces:
- Required field (cannot be omitted)
- Absolute path (starts with `/`)
- String type

Runtime validation in Go binary:

```go
if !filepath.IsAbs(config.ProjectRoot) {
    return fmt.Errorf("project_root must be absolute: %s", config.ProjectRoot)
}
if _, err := os.Stat(config.ProjectRoot); err != nil {
    return fmt.Errorf("project_root does not exist: %w", err)
}
```

## Agent Behavior

Agents receive `project_root` in their stdin context. Relative paths in the task description are resolved against this root. Tool calls (Read, Write, Glob, Bash) operate within this directory.

## Cross-Reference

- `cmd/goyoke-validate/main.go:66-74`: Same env var resolution pattern
- `spawnAgent.ts`: MCP-spawned agents inherit TUI's cwd (no explicit project_root)
- TC-006: Schema design for this field
- TC-008: Go binary implementation that uses this field
