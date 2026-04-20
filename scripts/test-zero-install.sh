#!/usr/bin/env bash
# test-zero-install.sh — Zero-install integration test (consumer-perspective E2E)
#
# Validates the full zero-install path: go install + goyoke works without
# manual ~/.claude/ setup. BYO-Claude: user has Claude Code authenticated.
#
# Usage: ./scripts/test-zero-install.sh [--skip-claude]
#   --skip-claude  Skip step 2 (requires working Claude auth)
#
# Dependencies: jq, rsync, go
# Exit 0 on pass, exit 1 on any failure.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SKIP_CLAUDE=false

for arg in "$@"; do
    case "$arg" in
        --skip-claude) SKIP_CLAUDE=true ;;
    esac
done

PASS=0
FAIL=0
SKIP=0

check() {
    local desc="$1"; shift
    if "$@" >/dev/null 2>&1; then
        echo "  PASS: $desc"
        PASS=$((PASS + 1))
    else
        echo "  FAIL: $desc"
        FAIL=$((FAIL + 1))
    fi
}

TEMP_DIR=$(mktemp -d)
TEMP_PUBLIC="${TEMP_DIR}/public-repo"
TEMP_BIN="${TEMP_DIR}/bin"
mkdir -p "$TEMP_BIN"
trap 'rm -rf "$TEMP_DIR"' EXIT

echo "[test-zero-install] Starting integration test..."
echo "  Project root: $PROJECT_ROOT"
echo "  Temp dir: $TEMP_DIR"

# ============================================================
# STEP 1: Build from simulated public repo
# ============================================================
echo ""
echo "=== Step 1: Build single binary from simulated public repo ==="

echo "  Syncing public content..."
rsync -a \
    --exclude='.claude/' \
    --exclude='.goyoke/' \
    --exclude='.env' \
    --exclude='tickets/' \
    --exclude='.git/' \
    --exclude='bin/' \
    "$PROJECT_ROOT/" "$TEMP_PUBLIC/"

# Ensure defaults/ is populated
if [ ! -f "${TEMP_PUBLIC}/defaults/agents/agents-index.json" ]; then
    echo "  Copying defaults/ from project..."
    cp -r "${PROJECT_ROOT}/defaults/"* "${TEMP_PUBLIC}/defaults/" 2>/dev/null || true
fi

echo "  Building goyoke binary from public repo..."
if (cd "$TEMP_PUBLIC" && go build -o "$TEMP_BIN/goyoke" ./cmd/goyoke 2>&1); then
    echo "  PASS: goyoke binary builds from public repo"
    PASS=$((PASS + 1))
else
    echo "  FAIL: build failed from public repo"
    FAIL=$((FAIL + 1))
fi

check "goyoke binary exists" test -f "${TEMP_BIN}/goyoke"
check "goyoke binary is executable" test -x "${TEMP_BIN}/goyoke"

# ============================================================
# STEP 2: Hook injection test (requires Claude auth)
# ============================================================
echo ""
echo "=== Step 2: Hook injection test ==="
if [ "$SKIP_CLAUDE" = true ]; then
    echo "  SKIP: --skip-claude flag set"
    SKIP=$((SKIP + 1))
else
    MARKER_DIR=$(mktemp -d)
    MARKER_FILE="${MARKER_DIR}/hook-fired"
    MARKER_SETTINGS="${MARKER_DIR}/marker-settings.json"

    cat > "$MARKER_SETTINGS" <<JSONEOF
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "startup",
        "hooks": [
          {
            "type": "command",
            "command": "touch ${MARKER_FILE}",
            "timeout": 5
          }
        ]
      }
    ]
  }
}
JSONEOF

    echo "  Running claude with --settings injection..."
    if timeout 30 claude -p 'respond with OK' --settings "$MARKER_SETTINGS" --output-format stream-json --verbose --permission-mode plan < /dev/null >/dev/null 2>&1; then
        if [ -f "$MARKER_FILE" ]; then
            echo "  PASS: --settings hook injection works (marker file created)"
            PASS=$((PASS + 1))
        else
            echo "  FAIL: claude ran but hook didn't fire (no marker file)"
            FAIL=$((FAIL + 1))
        fi
    else
        echo "  FAIL: claude command failed (auth issue?)"
        FAIL=$((FAIL + 1))
    fi
    rm -rf "$MARKER_DIR"
fi

# ============================================================
# STEP 3: Settings template validation
# ============================================================
echo ""
echo "=== Step 3: Settings template validation ==="

TEMPLATE="${PROJECT_ROOT}/defaults/settings-template.json"
check "settings-template.json exists" test -f "$TEMPLATE"

if [ -f "$TEMPLATE" ]; then
    HOOK_COUNT=$(jq '[.hooks | to_entries[] | .value[] | .hooks[]] | length' "$TEMPLATE" 2>/dev/null || echo 0)
    echo "  Found $HOOK_COUNT hook entries"
    check "at least 10 hook entries" test "$HOOK_COUNT" -ge 10

    # Verify all commands use multicall format (goyoke hook <name>)
    COMMANDS=$(jq -r '[.hooks | to_entries[] | .value[] | .hooks[] | .command // empty] | unique | .[]' "$TEMPLATE" 2>/dev/null)
    ALL_MULTICALL=true
    for cmd in $COMMANDS; do
        if [[ "$cmd" != goyoke\ hook\ * ]]; then
            echo "  WARN: non-multicall command: $cmd"
            ALL_MULTICALL=false
        fi
    done
    if [ "$ALL_MULTICALL" = true ]; then
        echo "  PASS: all hook commands use multicall format (goyoke hook <name>)"
        PASS=$((PASS + 1))
    else
        echo "  FAIL: some commands don't use multicall format"
        FAIL=$((FAIL + 1))
    fi

    # Verify goyoke binary can dispatch each hook command
    GOYOKE="${TEMP_BIN}/goyoke"
    HOOK_DISPATCH_OK=true
    for cmd in $COMMANDS; do
        # Extract hook name from "goyoke hook <name>"
        HOOK_NAME="${cmd#goyoke hook }"
        MOCK_EVENT='{"hook_event_name":"test","tool_name":"Bash","session_id":"test"}'
        if ! echo "$MOCK_EVENT" | "$GOYOKE" hook "$HOOK_NAME" >/dev/null 2>&1; then
            # Exit code 1 is OK (hook ran but rejected event), only panic/crash is bad
            OUTPUT=$(echo "$MOCK_EVENT" | "$GOYOKE" hook "$HOOK_NAME" 2>&1 || true)
            if echo "$OUTPUT" | grep -q 'panic:'; then
                echo "  FAIL: hook '$HOOK_NAME' panicked"
                HOOK_DISPATCH_OK=false
            fi
        fi
    done
    if [ "$HOOK_DISPATCH_OK" = true ]; then
        echo "  PASS: all hooks dispatch without panic"
        PASS=$((PASS + 1))
    else
        FAIL=$((FAIL + 1))
    fi
fi

# ============================================================
# STEP 4: Multicall dispatch verification
# ============================================================
echo ""
echo "=== Step 4: Multicall dispatch ==="

GOYOKE="${TEMP_BIN}/goyoke"

# Test version subcommand
VERSION_OUT=$("$GOYOKE" version 2>&1 || true)
if echo "$VERSION_OUT" | grep -qi 'version\|dev'; then
    echo "  PASS: goyoke version works"
    PASS=$((PASS + 1))
else
    echo "  FAIL: goyoke version unexpected output: $VERSION_OUT"
    FAIL=$((FAIL + 1))
fi

# Test hook dispatch with mock event
MOCK_EVENT='{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"echo test"},"session_id":"test"}'
HOOKS="skill-guard direct-impl-check permission-gate orchestrator-guard config-guard instructions-audit"

for hook in $HOOKS; do
    STDERR_OUT=$(echo "$MOCK_EVENT" | "$GOYOKE" hook "$hook" 2>&1 >/dev/null || true)

    if echo "$STDERR_OUT" | grep -q 'panic:'; then
        echo "  FAIL: hook $hook panicked"
        FAIL=$((FAIL + 1))
    else
        echo "  PASS: hook $hook degrades gracefully"
        PASS=$((PASS + 1))
    fi
done

# ============================================================
# STEP 5: Build verification
# ============================================================
echo ""
echo "=== Step 5: Build verification ==="

# Single binary size check
BIN_PATH="${TEMP_BIN}/goyoke"
if [ -f "$BIN_PATH" ]; then
    SIZE=$(stat -c%s "$BIN_PATH" 2>/dev/null || stat -f%z "$BIN_PATH")
    MB=$(echo "scale=1; $SIZE / 1048576" | bc)
    echo "  goyoke: ${MB}MB"
    # Should be under 40MB (dev build without stripping)
    check "binary size under 40MB" test "$SIZE" -lt 41943040
fi

# No private agents
PRIVATE_IN_INDEX=$(jq -r '.agents[] | select(.distribution == "private") | .id' "${PROJECT_ROOT}/defaults/agents/agents-index.json" 2>/dev/null || true)
if [ -z "$PRIVATE_IN_INDEX" ]; then
    echo "  PASS: no private agents in defaults index"
    PASS=$((PASS + 1))
else
    echo "  FAIL: private agents in defaults index: $PRIVATE_IN_INDEX"
    FAIL=$((FAIL + 1))
fi

# ============================================================
# Summary
# ============================================================
echo ""
echo "=== Results ==="
echo "  Passed: $PASS"
echo "  Failed: $FAIL"
echo "  Skipped: $SKIP"

if [ "$FAIL" -gt 0 ]; then
    echo ""
    echo "ZERO-INSTALL TEST FAILED"
    exit 1
fi

echo ""
echo "ZERO-INSTALL TEST PASSED"
exit 0
