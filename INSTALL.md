# goYoke Installation

Single binary. Zero config. Just add to PATH and authenticate.

## Prerequisites

- [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) installed:
  ```bash
  npm install -g @anthropic-ai/claude-code
  ```
- Authentication token (one of):
  - `claude setup-token` (interactive — gives you a 1-year OAuth token)
  - API key from [console.anthropic.com](https://console.anthropic.com)

## Quick Start

```bash
# 1. Download binary for your platform (from GitHub Releases or CI artifacts)
chmod +x goyoke-*-linux-amd64   # or darwin-amd64 / darwin-arm64

# 2. Move to PATH
mv goyoke-*-linux-amd64 ~/.local/bin/goyoke

# 3. Set auth (pick one)
export CLAUDE_CODE_OAUTH_TOKEN="sk-ant-oat01-..."   # from claude setup-token
# OR
export ANTHROPIC_API_KEY="sk-ant-api03-..."          # from console.anthropic.com

# 4. Run
goyoke
```

## Platform Support

| Platform | Status | Binary Name |
|----------|--------|-------------|
| Linux x86_64 | Supported | `goyoke-<version>-linux-amd64` |
| macOS Intel | Supported | `goyoke-<version>-darwin-amd64` |
| macOS Apple Silicon | Supported | `goyoke-<version>-darwin-arm64` |
| Windows x86_64 | Supported | `goyoke-<version>-windows-amd64.exe` |

## Build from Source

```bash
git clone <repo-url>
cd goYoke
make defaults   # populate embedded config (requires jq)
make build      # builds to bin/goyoke
make install    # copies to ~/.local/bin/goyoke
```

## Getting an Auth Token

### Option A: OAuth Token (recommended for Claude Pro/Max subscribers)
```bash
claude setup-token
# Follow the URL, paste the token back
# Set in your shell profile:
echo 'export CLAUDE_CODE_OAUTH_TOKEN="sk-ant-oat01-..."' >> ~/.bashrc
```

### Option B: API Key (for API billing)
```bash
# Get from console.anthropic.com → Settings → API Keys
echo 'export ANTHROPIC_API_KEY="sk-ant-api03-..."' >> ~/.bashrc
```

## Verify Installation

```bash
goyoke version                    # prints version
echo '{}' | goyoke hook validate  # tests hook dispatch
goyoke                            # launches TUI
```

## How It Works

goYoke is a single multicall binary that handles all roles:
- `goyoke` — TUI mode (default)
- `goyoke mcp` — MCP server (spawned automatically by TUI)
- `goyoke hook <name>` — Hook handlers (fired by Claude CLI)
- `goyoke <utility>` — Utility commands (scout, team-run, etc.)

All configuration (agents, conventions, routing rules, hooks) is embedded in the binary. No external config files needed.

## Docker Testing

```bash
# Full isolation test (proves zero-install works)
CLAUDE_CODE_OAUTH_TOKEN="..." ./test/docker/run-naked-test.sh
```
