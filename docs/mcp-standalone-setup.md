# gofortress-mcp-standalone Setup

## Installation

Build the binary:

```
go build -o bin/gofortress-mcp-standalone ./cmd/gofortress-mcp-standalone/
```

## Claude Code mcpServers Configuration

Add the following to `~/.claude/settings.json`:

```json
{
  "mcpServers": {
    "gofortress-standalone": {
      "command": "/absolute/path/to/bin/gofortress-mcp-standalone",
      "env": {
        "GOGENT_PROJECT_ROOT": "/absolute/path/to/GOgent-Fortress"
      }
    }
  }
}
```

Claude Code's `mcpServers` supports the `env` key for passing environment
variables to the server process. `GOGENT_PROJECT_ROOT` enables
`GetSessionDir()` in `pkg/routing/identity_loader.go` to resolve the
current-session marker file, which is required for session context injection.

## Environment Variables

| Variable | Set By | Purpose |
|----------|--------|---------|
| `GOGENT_PROJECT_ROOT` | mcpServers config | Project root for session dir resolution |
| `GOGENT_AGENTS_INDEX` | Optional override | Custom path to agents-index.json |
| `GOGENT_NESTING_LEVEL` | spawn_agent (auto) | Current nesting depth, incremented per spawn |
| `GOGENT_PARENT_AGENT` | spawn_agent (auto) | Agent ID of the parent process |
| `GOGENT_SPAWN_METHOD` | spawn_agent (auto) | Set to "mcp-cli" for CLI spawns |
| `GOGENT_SESSION_DIR` | Optional override | Direct session directory path |

## Coexistence with gofortress-mcp

Both MCP servers can be registered simultaneously in `settings.json`.

| Server | Tools | Requires TUI |
|--------|-------|-------------|
| `gofortress-mcp` | ask_user, confirm_action, request_input, select_option, spawn_agent, team_run, test_mcp_ping | Yes (GOFORTRESS_SOCKET) |
| `gofortress-mcp-standalone` | spawn_agent, test_mcp_ping | No |

When both are registered, Claude Code routes tool calls to the server that
provides the requested tool. If both provide `spawn_agent`, behavior depends
on Claude Code's MCP server selection. To avoid ambiguity:

- Use **only the standalone server** when running Claude Code outside the TUI.
- Use **both** when the TUI is running (interactive tools from gofortress-mcp,
  functional spawn_agent from standalone).

## Verifying the Server

Test that the server starts and responds to a ping:

```
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.0.1"}}}' | bin/gofortress-mcp-standalone
```
