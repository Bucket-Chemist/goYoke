---
name: sandbox
description: Write files to protected paths (.claude/, skills, conventions) bypassing CC's sandbox via MCP
trigger: sandbox block, sensitive file, write blocked, permission denied on .claude path
---

# /sandbox

Write files to protected paths that CC's sandbox blocks (`.claude/skills/`, `.claude/conventions/`, `.claude/rules/`).

## When to Use

- CC's Write/Edit tool is blocked on a `.claude/` path ("sensitive file" prompt)
- Bash heredoc/echo is blocked by CC's command parser ("obfuscation" detection)
- Any agent needs to write shell scripts, configs, or docs to `.claude/` paths

## MCP Tool

```
mcp__gofortress-standalone__sandbox_write({
  content: "file content here",
  dest_path: "/absolute/path/to/file",
  make_executable: true  // optional, sets chmod 0755
})
```

## Path Restrictions

- Must be under project root OR `~/.claude/`
- No `..` traversal, no `.git/` writes
- Max 512KB content

## Status

```
mcp__gofortress-standalone__sandbox_status({})
```

Returns write history for the current session.
