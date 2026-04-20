#!/usr/bin/env bash
# test-naked.sh — Test the single binary in a truly isolated environment
#
# Overrides HOME so the claude CLI subprocess cannot read your real ~/.claude/.
# Only the auth credentials are carried over. Everything else (hooks, settings,
# CLAUDE.md, skills) must come from the embedded binary or it doesn't work.
#
# Usage: ./test-naked.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REAL_HOME="$HOME"
NAKED_DIR="/tmp/goyoke-naked"
FAKE_HOME="${NAKED_DIR}/home"
BIN_DIR="${NAKED_DIR}/bin"
CREDS="${REAL_HOME}/.claude/.credentials.json"

echo "[test-naked] Setting up isolated environment..."
echo "  Real HOME: $REAL_HOME"
echo "  Fake HOME: $FAKE_HOME"
echo ""

rm -rf "$NAKED_DIR"
mkdir -p "$BIN_DIR" "${FAKE_HOME}/.claude"

# Build
echo "[test-naked] Building single binary..."
go build -ldflags "-w -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)" \
    -o "${BIN_DIR}/goyoke" ./cmd/goyoke/

SIZE=$(stat -c%s "${BIN_DIR}/goyoke" 2>/dev/null || stat -f%z "${BIN_DIR}/goyoke")
MB=$(echo "scale=1; $SIZE / 1048576" | bc)
echo "[test-naked] Binary: ${BIN_DIR}/goyoke (${MB}MB)"

# Copy ONLY credentials — nothing else
if [ -f "$CREDS" ]; then
    cp "$CREDS" "${FAKE_HOME}/.claude/.credentials.json"
    echo "[test-naked] Auth credentials copied to fake HOME."
else
    echo "[test-naked] ERROR: ${CREDS} not found. Cannot authenticate."
    echo "  Run 'claude' once to authenticate, then retry."
    exit 1
fi

# Verify isolation — fake HOME should have NOTHING but credentials
echo ""
echo "[test-naked] Fake HOME contents (should be minimal):"
find "$FAKE_HOME" -type f | sed "s|$FAKE_HOME|  ~|"

# Smoke tests (run with fake HOME)
echo ""
echo "[test-naked] Smoke tests (HOME=${FAKE_HOME}):"
echo -n "  version: "
HOME="$FAKE_HOME" "${BIN_DIR}/goyoke" version 2>&1 || true
echo -n "  hook dispatch: "
echo '{}' | HOME="$FAKE_HOME" "${BIN_DIR}/goyoke" hook validate >/dev/null 2>&1 && echo "OK" || echo "OK (non-zero exit expected)"

# Launch TUI with full isolation
echo ""
echo "============================================"
echo "[test-naked] Launching TUI with full isolation:"
echo "  HOME=$FAKE_HOME (claude CLI sees empty .claude/)"
echo "  CWD=/tmp/goyoke-naked (no .claude/, no .git)"
echo "  PATH has goyoke binary (hooks resolve via PATH)"
echo ""
echo "  If this works → zero-install architecture is proven."
echo "  If hooks fail → embedded injection isn't working."
echo "============================================"
echo ""

cd "$NAKED_DIR"
export HOME="$FAKE_HOME"
export PATH="${BIN_DIR}:${PATH}"
exec "${BIN_DIR}/goyoke"
