#!/usr/bin/env bash
# dev-setup.sh — Set up goYoke development environment
#
# Run once after cloning the repo. Builds all binaries, creates the
# ~/.claude symlink, and generates a settings.json with correct paths.
#
# Usage: ./scripts/dev-setup.sh
#
# Prerequisites: Go 1.25+, Claude Code installed and authenticated

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BIN_DIR="${PROJECT_ROOT}/bin"
CLAUDE_DIR="${PROJECT_ROOT}/.claude"

echo "[dev-setup] goYoke development environment setup"
echo "  Project root: $PROJECT_ROOT"
echo ""

# --- 1. Build all binaries ---
echo "=== Step 1: Building binaries ==="
mkdir -p "$BIN_DIR"
go build -o "$BIN_DIR/" ./cmd/...
BINARY_COUNT=$(ls "$BIN_DIR" | wc -l)
echo "  Built $BINARY_COUNT binaries to $BIN_DIR"
echo ""

# --- 2. Symlink ~/.claude ---
echo "=== Step 2: ~/.claude symlink ==="
if [ -L "$HOME/.claude" ]; then
    CURRENT_TARGET=$(readlink "$HOME/.claude")
    if [ "$CURRENT_TARGET" = "$CLAUDE_DIR" ]; then
        echo "  Symlink already points to this repo"
    else
        echo "  WARNING: ~/.claude already symlinks to: $CURRENT_TARGET"
        echo "  To switch to this repo, run:"
        echo "    ln -sfn $CLAUDE_DIR $HOME/.claude"
        echo "  Skipping (won't overwrite existing symlink)"
    fi
elif [ -d "$HOME/.claude" ]; then
    echo "  WARNING: ~/.claude is a real directory (not a symlink)"
    echo "  Back it up and create symlink manually:"
    echo "    mv ~/.claude ~/.claude.backup"
    echo "    ln -s $CLAUDE_DIR $HOME/.claude"
    echo "  Skipping"
else
    ln -s "$CLAUDE_DIR" "$HOME/.claude"
    echo "  Created: ~/.claude -> $CLAUDE_DIR"
fi
echo ""

# --- 3. Generate settings.json ---
echo "=== Step 3: Hook configuration ==="
SETTINGS_FILE="${CLAUDE_DIR}/settings.json"

if [ -f "$SETTINGS_FILE" ]; then
    echo "  settings.json already exists — skipping generation"
    echo "  To regenerate: ./scripts/dev-setup.sh --regen-settings"
else
    echo "  Generating settings.json with local binary paths..."
fi

REGEN=false
for arg in "$@"; do
    [ "$arg" = "--regen-settings" ] && REGEN=true
done

if [ ! -f "$SETTINGS_FILE" ] || [ "$REGEN" = true ]; then
    cat > "$SETTINGS_FILE" << JSONEOF
{
  "permissions": {
    "allow": [],
    "deny": []
  },
  "hooks": {
    "SessionStart": [
      {
        "matcher": "startup|resume|clear|compact",
        "hooks": [
          {
            "type": "command",
            "command": "${BIN_DIR}/goyoke-load-context",
            "timeout": 10
          }
        ]
      }
    ],
    "PreToolUse": [
      {
        "matcher": ".*",
        "hooks": [
          {
            "type": "command",
            "command": "${BIN_DIR}/goyoke-skill-guard",
            "timeout": 5
          }
        ]
      },
      {
        "matcher": "Task|Agent",
        "hooks": [
          {
            "type": "command",
            "command": "${BIN_DIR}/goyoke-validate",
            "timeout": 10
          }
        ]
      },
      {
        "matcher": "Write|Edit",
        "hooks": [
          {
            "type": "command",
            "command": "${BIN_DIR}/goyoke-direct-impl-check",
            "timeout": 5
          }
        ]
      },
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "${BIN_DIR}/goyoke-permission-gate",
            "timeout": 5
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": ".*",
        "hooks": [
          {
            "type": "command",
            "command": "${BIN_DIR}/goyoke-sharp-edge",
            "timeout": 5
          }
        ]
      }
    ],
    "SubagentStop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "${BIN_DIR}/goyoke-agent-endstate",
            "timeout": 10
          },
          {
            "type": "command",
            "command": "${BIN_DIR}/goyoke-orchestrator-guard",
            "timeout": 5
          }
        ]
      }
    ],
    "SessionEnd": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "${BIN_DIR}/goyoke-archive",
            "timeout": 15
          }
        ]
      }
    ],
    "ConfigChange": [
      {
        "matcher": "user|project|local settings",
        "hooks": [
          {
            "type": "command",
            "command": "${BIN_DIR}/goyoke-config-guard",
            "timeout": 5
          }
        ]
      }
    ],
    "InstructionsLoaded": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "${BIN_DIR}/goyoke-instructions-audit",
            "timeout": 5
          }
        ]
      }
    ]
  }
}
JSONEOF
    echo "  Generated: $SETTINGS_FILE"
    echo "  Hook binaries point to: $BIN_DIR/"
fi
echo ""

# --- 4. MCP server config ---
echo "=== Step 4: MCP config ==="
MCP_CONFIG="${CLAUDE_DIR}/mcp.json"
if [ -f "$MCP_CONFIG" ]; then
    echo "  mcp.json already exists"
else
    cat > "$MCP_CONFIG" << JSONEOF
{
  "mcpServers": {
    "goyoke-interactive": {
      "command": "${BIN_DIR}/goyoke-mcp",
      "args": [],
      "env": {}
    }
  }
}
JSONEOF
    echo "  Generated: $MCP_CONFIG"
fi
echo ""

# --- 5. Summary ---
echo "=== Setup complete ==="
echo ""
echo "  Binaries:     $BIN_DIR/ ($BINARY_COUNT binaries)"
echo "  Config:       $CLAUDE_DIR/"
echo "  Symlink:      ~/.claude -> $CLAUDE_DIR"
echo "  Hooks:        $SETTINGS_FILE"
echo "  MCP:          $MCP_CONFIG"
echo ""
echo "  To launch the TUI:    $BIN_DIR/goyoke"
echo "  To use with claude:   claude  (hooks fire automatically)"
echo ""
echo "  After pulling changes: make build"
echo "  To rebuild everything: ./scripts/dev-setup.sh --regen-settings"
