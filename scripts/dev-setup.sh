#!/usr/bin/env bash
# dev-setup.sh — Set up goYoke development environment
#
# Uses the same zero-install path as public distribution:
# embedded defaults, --settings hook injection, no ~/.claude/ writes.
#
# Usage: ./scripts/dev-setup.sh
#
# Prerequisites: Go 1.25+, jq, Claude Code installed and authenticated

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "[dev-setup] goYoke development environment setup"
echo ""

# --- 1. Check prerequisites ---
echo "=== Prerequisites ==="
command -v go >/dev/null 2>&1 || { echo "  FAIL: Go not found. Install Go 1.25+"; exit 1; }
command -v jq >/dev/null 2>&1 || { echo "  FAIL: jq not found. Install jq"; exit 1; }
echo "  Go: $(go version | awk '{print $3}')"
echo "  jq: $(jq --version)"
echo ""

# --- 2. Generate defaults + build ---
echo "=== Building ==="
make defaults
make dist
echo ""

# --- 3. Install to PATH ---
echo "=== Installing to ~/.local/bin ==="
mkdir -p ~/.local/bin
cp bin/* ~/.local/bin/
BINARY_COUNT=$(ls bin/ | wc -l)
echo "  Installed $BINARY_COUNT binaries to ~/.local/bin/"

if ! echo "$PATH" | grep -q "\.local/bin"; then
    echo ""
    echo "  WARNING: ~/.local/bin is not in your PATH"
    echo "  Add to your shell profile:"
    echo "    export PATH=\"\$HOME/.local/bin:\$PATH\""
fi
echo ""

# --- 4. Verify ---
echo "=== Verification ==="
if command -v goyoke >/dev/null 2>&1; then
    echo "  goyoke: found at $(which goyoke)"
else
    echo "  goyoke: NOT on PATH (check step above)"
fi
if command -v goyoke-validate >/dev/null 2>&1; then
    echo "  goyoke-validate: found"
else
    echo "  goyoke-validate: NOT on PATH"
fi
echo ""

# --- 5. Summary ---
echo "=== Ready ==="
echo ""
echo "  Launch:     goyoke"
echo "  How it works:"
echo "    - Embedded defaults provide agents, conventions, rules, schemas"
echo "    - Hooks injected via --settings flag (no ~/.claude/ writes)"
echo "    - CLAUDE.md injected via additionalContext at session start"
echo "    - Your ~/.claude/ is untouched — add files there to override defaults"
echo ""
echo "  After pulling changes:"
echo "    make dist && cp bin/* ~/.local/bin/"
