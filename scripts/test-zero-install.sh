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
echo "=== Step 1: Build from simulated public repo ==="

echo "  Syncing public content..."
rsync -a \
    --exclude='.claude/' \
    --exclude='.goyoke/' \
    --exclude='.env' \
    --exclude='tickets/' \
    --exclude='.git/' \
    --exclude='bin/' \
    --exclude='cmd/goyoke-archive/.goyoke/' \
    --exclude='cmd/goyoke-sharp-edge/.goyoke/' \
    --exclude='pkg/telemetry/.goyoke/' \
    "$PROJECT_ROOT/" "$TEMP_PUBLIC/"

# Ensure defaults/ is populated
if [ ! -f "${TEMP_PUBLIC}/defaults/agents/agents-index.json" ]; then
    echo "  Copying defaults/ from project..."
    cp -r "${PROJECT_ROOT}/defaults/"* "${TEMP_PUBLIC}/defaults/" 2>/dev/null || true
fi

echo "  Building binaries from public repo..."
if (cd "$TEMP_PUBLIC" && go build -o "$TEMP_BIN/" ./cmd/... 2>&1); then
    echo "  PASS: all binaries build from public repo"
    PASS=$((PASS + 1))
else
    echo "  FAIL: build failed from public repo"
    FAIL=$((FAIL + 1))
fi

BINARY_COUNT=$(ls "$TEMP_BIN" 2>/dev/null | wc -l)
echo "  Built $BINARY_COUNT binaries"
check "at least 20 binaries built" test "$BINARY_COUNT" -ge 20

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

    # Verify no absolute paths in hook commands
    ABS_PATHS=$(jq -r '[.hooks | to_entries[] | .value[] | .hooks[] | .command // empty] | .[]' "$TEMPLATE" 2>/dev/null | grep '/' || true)
    if [ -z "$ABS_PATHS" ]; then
        echo "  PASS: all hook commands use binary names (no absolute paths)"
        PASS=$((PASS + 1))
    else
        echo "  FAIL: absolute paths found in hook commands: $ABS_PATHS"
        FAIL=$((FAIL + 1))
    fi

    # Verify each referenced binary exists in temp bin
    COMMANDS=$(jq -r '[.hooks | to_entries[] | .value[] | .hooks[] | .command // empty] | unique | .[]' "$TEMPLATE" 2>/dev/null)
    MISSING_BINS=""
    for cmd in $COMMANDS; do
        if [ ! -f "${TEMP_BIN}/${cmd}" ]; then
            MISSING_BINS="${MISSING_BINS} ${cmd}"
        fi
    done
    if [ -z "$MISSING_BINS" ]; then
        echo "  PASS: all hook binaries found in build output"
        PASS=$((PASS + 1))
    else
        echo "  FAIL: missing hook binaries:${MISSING_BINS}"
        FAIL=$((FAIL + 1))
    fi
fi

# ============================================================
# STEP 4: Non-embedded hook degradation
# ============================================================
echo ""
echo "=== Step 4: Hook degradation (no config) ==="

HOOKS="goyoke-skill-guard goyoke-direct-impl-check goyoke-permission-gate goyoke-orchestrator-guard goyoke-config-guard goyoke-instructions-audit"

for hook in $HOOKS; do
    BIN="${TEMP_BIN}/${hook}"
    if [ ! -f "$BIN" ]; then
        echo "  SKIP: $hook not built"
        SKIP=$((SKIP + 1))
        continue
    fi

    MOCK_EVENT='{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"echo test"},"session_id":"test"}'
    STDERR_OUT=$("$BIN" <<< "$MOCK_EVENT" 2>&1 >/dev/null || true)
    EXIT_CODE=$?

    if echo "$STDERR_OUT" | grep -q 'panic:'; then
        echo "  FAIL: $hook panicked"
        FAIL=$((FAIL + 1))
    elif [ "$EXIT_CODE" -gt 1 ]; then
        echo "  FAIL: $hook exit code $EXIT_CODE (>1)"
        FAIL=$((FAIL + 1))
    else
        echo "  PASS: $hook degrades gracefully (exit $EXIT_CODE)"
        PASS=$((PASS + 1))
    fi
done

# ============================================================
# STEP 5: Build verification
# ============================================================
echo ""
echo "=== Step 5: Build verification ==="

# Embedded binary sizes
EMBEDDED_BINS="goyoke goyoke-mcp goyoke-team-run goyoke-load-context goyoke-validate"
for bin in $EMBEDDED_BINS; do
    BIN_PATH="${TEMP_BIN}/${bin}"
    if [ -f "$BIN_PATH" ]; then
        SIZE=$(stat -c%s "$BIN_PATH" 2>/dev/null || stat -f%z "$BIN_PATH")
        MB=$(echo "scale=1; $SIZE / 1048576" | bc)
        echo "  $bin: ${MB}MB"
    fi
done

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
