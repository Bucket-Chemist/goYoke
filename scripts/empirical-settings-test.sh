#!/usr/bin/env bash
# empirical-settings-test.sh — Verify Claude Code settings discovery for github-remodel plan
#
# PURPOSE: Empirically test 6 scenarios that the github-remodel cutover plan
# depends on. Results determine whether D-6 (settings.local.json) works, and
# whether the --settings flag is a viable alternative.
#
# USAGE: Run from a terminal OUTSIDE any active Claude Code session.
#   ./scripts/empirical-settings-test.sh
#
# REQUIRES:
#   - `claude` binary on PATH
#   - Either ANTHROPIC_API_KEY set or working OAuth/keychain auth
#   - jq (for JSON parsing)
#
# ESTIMATED COST: ~$0.02 (6 minimal claude -p calls)
# ESTIMATED TIME: ~2 minutes
#
# Each test creates an isolated environment, runs claude -p with a marker hook,
# and checks whether the hook fired.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_ROOT="/tmp/goyoke-empirical-$(date +%s)"
LOG_FILE="$TEST_ROOT/results.log"
MARKER_DIR="$TEST_ROOT/markers"
ORIGINAL_HOME="$HOME"
ORIGINAL_PATH="$PATH"
ORIGINAL_GOBIN="${GOBIN:-}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

cleanup() {
    echo ""
    echo "Results preserved at: $TEST_ROOT"
}
trap cleanup EXIT

mkdir -p "$TEST_ROOT" "$MARKER_DIR"

log() { echo -e "$1" | tee -a "$LOG_FILE"; }
pass() { log "${GREEN}  PASS${NC}: $1"; }
fail() { log "${RED}  FAIL${NC}: $1"; }
skip() { log "${YELLOW}  SKIP${NC}: $1"; }
info() { log "${CYAN}  INFO${NC}: $1"; }

# Generate a hook config JSON that writes a marker file when triggered
make_hook_settings() {
    local marker_name="$1"
    cat <<EOF
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": ".*",
        "hooks": [
          {
            "type": "command",
            "command": "echo FIRED > $MARKER_DIR/${marker_name}.marker"
          }
        ]
      }
    ]
  }
}
EOF
}

# Run claude -p with minimal prompt that triggers a tool use (PreToolUse hook)
# Args: $1=test_name, $2+=extra env/flags
run_claude_test() {
    local test_name="$1"
    shift
    local timeout=30

    # Use a prompt that forces tool use (Read triggers PreToolUse)
    # --output-format stream-json suppresses interactive output
    local exit_code=0
    timeout "$timeout" env "$@" claude -p \
        "Use the Read tool to read /etc/hostname, then say DONE" \
        --output-format stream-json --verbose \
        --permission-mode acceptEdits < /dev/null \
        > "$TEST_ROOT/${test_name}.stdout" 2>"$TEST_ROOT/${test_name}.stderr" || exit_code=$?

    if [ "$exit_code" -eq 124 ]; then
        info "$test_name timed out after ${timeout}s (possible interactive prompt)"
        return 1
    elif [ "$exit_code" -ne 0 ]; then
        info "$test_name exited with code $exit_code"
        # Check stderr for auth issues
        if grep -qi "auth\|login\|api.key\|unauthorized\|keychain" "$TEST_ROOT/${test_name}.stderr" 2>/dev/null; then
            info "Auth issue detected in stderr"
        fi
        return 1
    fi
    return 0
}

check_marker() {
    local marker_name="$1"
    [ -f "$MARKER_DIR/${marker_name}.marker" ]
}

# ═══════════════════════════════════════════════════════════════════
log ""
log "═══════════════════════════════════════════════════════════════"
log " goYoke Empirical Settings Test Suite"
log " $(date)"
log " Test root: $TEST_ROOT"
log "═══════════════════════════════════════════════════════════════"
log ""

# ── Pre-flight checks ──
if ! command -v claude &>/dev/null; then
    fail "claude binary not found on PATH"
    exit 1
fi
if ! command -v jq &>/dev/null; then
    fail "jq not found on PATH"
    exit 1
fi
info "claude: $(command -v claude)"
info "claude version: $(claude --version 2>/dev/null || echo 'unknown')"
info "HOME: $HOME"
info "ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY:+SET (hidden)}${ANTHROPIC_API_KEY:-NOT SET}"
log ""

# ═══════════════════════════════════════════════════════════════════
# E-1: BASELINE — --settings flag with hook injection (HIGHEST PRIORITY)
#
# This tests the alternative to D-6. If this works, the TUI can pass
# hooks via --settings instead of writing settings.local.json to disk.
# Distribution plan Test 5 confirmed this merges additively.
# ═══════════════════════════════════════════════════════════════════
log "─── E-1: --settings flag with hook injection ───"
E1_SETTINGS="$TEST_ROOT/e1-settings.json"
make_hook_settings "e1" > "$E1_SETTINGS"
E1_DIR="$TEST_ROOT/e1-workdir"
mkdir -p "$E1_DIR"

info "Testing: claude -p --settings <file> with marker hook"
E1_EXIT=0
timeout 30 bash -c "
    cd '$E1_DIR'
    claude -p 'Use the Read tool to read /etc/hostname, then say DONE' \
        --settings '$E1_SETTINGS' \
        --output-format stream-json --verbose \
        --permission-mode acceptEdits < /dev/null
" > "$TEST_ROOT/e1.stdout" 2>"$TEST_ROOT/e1.stderr" || E1_EXIT=$?

if check_marker "e1"; then
    pass "--settings flag correctly registers and fires hooks"
    E1_RESULT="PASS"
else
    if [ "$E1_EXIT" -eq 124 ]; then
        fail "--settings test timed out (possible interactive prompt)"
    elif [ "$E1_EXIT" -ne 0 ]; then
        fail "--settings test failed (exit=$E1_EXIT). Check $TEST_ROOT/e1.stderr"
    else
        fail "--settings flag did NOT fire the marker hook"
    fi
    E1_RESULT="FAIL"
fi
log ""

# ═══════════════════════════════════════════════════════════════════
# E-2: Does CC discover ~/.claude/settings.local.json at USER-GLOBAL scope?
#
# This tests D-6's assumption. We create a clean HOME with ONLY
# settings.local.json (no settings.json) and check if CC discovers it.
# ═══════════════════════════════════════════════════════════════════
log "─── E-2: ~/.claude/settings.local.json user-global discovery ───"
E2_HOME="$TEST_ROOT/e2-home"
E2_WORKDIR="$TEST_ROOT/e2-workdir"
mkdir -p "$E2_HOME/.claude" "$E2_WORKDIR"
make_hook_settings "e2" > "$E2_HOME/.claude/settings.local.json"

info "Testing: HOME=$E2_HOME with ~/.claude/settings.local.json only"
info "(No settings.json, no project .claude/)"
E2_EXIT=0
timeout 30 bash -c "
    export HOME='$E2_HOME'
    export PATH='$ORIGINAL_PATH'
    ${ORIGINAL_GOBIN:+export GOBIN='$ORIGINAL_GOBIN'}
    cd '$E2_WORKDIR'
    claude -p 'Use the Read tool to read /etc/hostname, then say DONE' \
        --output-format stream-json --verbose \
        --permission-mode acceptEdits < /dev/null
" > "$TEST_ROOT/e2.stdout" 2>"$TEST_ROOT/e2.stderr" || E2_EXIT=$?

if check_marker "e2"; then
    pass "CC discovers ~/.claude/settings.local.json at user-global scope"
    E2_RESULT="PASS"
else
    if [ "$E2_EXIT" -eq 124 ]; then
        fail "Test timed out — possible first-launch interactive prompt with clean HOME"
        info "Check $TEST_ROOT/e2.stderr for onboarding messages"
    elif [ "$E2_EXIT" -ne 0 ]; then
        fail "Claude exited with code $E2_EXIT (auth? onboarding?). Check $TEST_ROOT/e2.stderr"
    else
        fail "CC does NOT discover ~/.claude/settings.local.json at user-global scope"
        info "D-6 assumption is FALSE — settings.local.json approach needs redesign"
    fi
    E2_RESULT="FAIL"
fi
log ""

# ═══════════════════════════════════════════════════════════════════
# E-3: Does CC discover <project>/.claude/settings.local.json?
#
# Baseline verification that project-level settings.local.json works.
# This is how goYoke currently works (via symlink).
# ═══════════════════════════════════════════════════════════════════
log "─── E-3: Project-level .claude/settings.local.json ───"
E3_DIR="$TEST_ROOT/e3-project"
mkdir -p "$E3_DIR/.claude"
make_hook_settings "e3" > "$E3_DIR/.claude/settings.local.json"

info "Testing: .claude/settings.local.json in project directory"
E3_EXIT=0
timeout 30 bash -c "
    cd '$E3_DIR'
    claude -p 'Use the Read tool to read /etc/hostname, then say DONE' \
        --output-format stream-json --verbose \
        --permission-mode acceptEdits < /dev/null
" > "$TEST_ROOT/e3.stdout" 2>"$TEST_ROOT/e3.stderr" || E3_EXIT=$?

if check_marker "e3"; then
    pass "CC discovers project-level .claude/settings.local.json"
    E3_RESULT="PASS"
else
    if [ "$E3_EXIT" -ne 0 ]; then
        fail "Claude exited with code $E3_EXIT. Check $TEST_ROOT/e3.stderr"
    else
        fail "CC does NOT discover project-level settings.local.json"
    fi
    E3_RESULT="FAIL"
fi
log ""

# ═══════════════════════════════════════════════════════════════════
# E-4: Clean HOME behavior — first-launch onboarding
#
# Tests what happens with a completely clean HOME. No config at all.
# Does CC prompt interactively? What files does it create?
# ═══════════════════════════════════════════════════════════════════
log "─── E-4: First-launch behavior with clean HOME ───"
E4_HOME="$TEST_ROOT/e4-home"
E4_WORKDIR="$TEST_ROOT/e4-workdir"
mkdir -p "$E4_HOME" "$E4_WORKDIR"

info "Testing: completely clean HOME (no .claude/ at all)"
info "Running claude --version first (minimal, non-interactive)"
E4_VERSION_EXIT=0
timeout 10 bash -c "
    export HOME='$E4_HOME'
    export PATH='$ORIGINAL_PATH'
    claude --version
" > "$TEST_ROOT/e4-version.stdout" 2>"$TEST_ROOT/e4-version.stderr" || E4_VERSION_EXIT=$?

if [ "$E4_VERSION_EXIT" -eq 0 ]; then
    pass "claude --version works with clean HOME"
    info "Version: $(cat "$TEST_ROOT/e4-version.stdout")"
else
    fail "claude --version failed with clean HOME (exit=$E4_VERSION_EXIT)"
fi

# Check what files were created
if [ -d "$E4_HOME/.claude" ]; then
    info "Files created in \$HOME/.claude/:"
    find "$E4_HOME/.claude" -type f 2>/dev/null | head -20 | while read -r f; do
        info "  $(basename "$f") ($(stat -c%s "$f" 2>/dev/null || stat -f%z "$f" 2>/dev/null) bytes)"
    done
    E4_CREATES_DIR="YES"
else
    info "No .claude/ directory created by claude --version"
    E4_CREATES_DIR="NO"
fi

# Now try claude -p (might trigger onboarding)
info "Testing: claude -p with clean HOME"
E4_EXIT=0
timeout 20 bash -c "
    export HOME='$E4_HOME'
    export PATH='$ORIGINAL_PATH'
    cd '$E4_WORKDIR'
    claude -p 'say hello' \
        --output-format stream-json --verbose \
        --permission-mode acceptEdits < /dev/null
" > "$TEST_ROOT/e4.stdout" 2>"$TEST_ROOT/e4.stderr" || E4_EXIT=$?

if [ "$E4_EXIT" -eq 0 ]; then
    pass "claude -p works with clean HOME (no interactive blocking)"
    E4_RESULT="PASS"
elif [ "$E4_EXIT" -eq 124 ]; then
    fail "claude -p TIMED OUT with clean HOME (interactive prompt?)"
    info "stderr tail:"
    tail -5 "$TEST_ROOT/e4.stderr" 2>/dev/null | while read -r line; do info "  $line"; done
    E4_RESULT="TIMEOUT"
else
    fail "claude -p failed with clean HOME (exit=$E4_EXIT)"
    E4_RESULT="FAIL"
fi

# Check final state of .claude/ dir
if [ -d "$E4_HOME/.claude" ]; then
    E4_FILE_COUNT=$(find "$E4_HOME/.claude" -type f 2>/dev/null | wc -l)
    info "After claude -p: $E4_FILE_COUNT files in \$HOME/.claude/"
fi
log ""

# ═══════════════════════════════════════════════════════════════════
# E-5: --settings + clean HOME combined (the actual zero-install path)
#
# This tests the proposed alternative to D-6: TUI passes hooks via
# --settings flag, user has clean HOME. Does everything work together?
# ═══════════════════════════════════════════════════════════════════
log "─── E-5: --settings with clean HOME (zero-install simulation) ───"
E5_HOME="$TEST_ROOT/e5-home"
E5_WORKDIR="$TEST_ROOT/e5-workdir"
E5_SETTINGS="$TEST_ROOT/e5-settings.json"
mkdir -p "$E5_HOME" "$E5_WORKDIR"
make_hook_settings "e5" > "$E5_SETTINGS"

info "Testing: --settings <file> with clean HOME (no ~/.claude/)"
E5_EXIT=0
timeout 30 bash -c "
    export HOME='$E5_HOME'
    export PATH='$ORIGINAL_PATH'
    ${ORIGINAL_GOBIN:+export GOBIN='$ORIGINAL_GOBIN'}
    cd '$E5_WORKDIR'
    claude -p 'Use the Read tool to read /etc/hostname, then say DONE' \
        --settings '$E5_SETTINGS' \
        --output-format stream-json --verbose \
        --permission-mode acceptEdits < /dev/null
" > "$TEST_ROOT/e5.stdout" 2>"$TEST_ROOT/e5.stderr" || E5_EXIT=$?

if check_marker "e5"; then
    pass "--settings works with clean HOME — zero-install hook injection viable"
    E5_RESULT="PASS"
else
    if [ "$E5_EXIT" -eq 124 ]; then
        fail "Timed out — onboarding blocks even with --settings"
    elif [ "$E5_EXIT" -ne 0 ]; then
        fail "Failed (exit=$E5_EXIT). Check $TEST_ROOT/e5.stderr"
    else
        fail "--settings did not fire hooks with clean HOME"
    fi
    E5_RESULT="FAIL"
fi
log ""

# ═══════════════════════════════════════════════════════════════════
# E-6: Non-embedded hook graceful degradation
#
# Tests the 2 hooks that read config files (sharp-edge, skill-guard)
# with no config available. Verifies no panics and correct exit codes.
# ═══════════════════════════════════════════════════════════════════
log "─── E-6: Non-embedded hook graceful degradation ───"
E6_HOME="$TEST_ROOT/e6-home"
mkdir -p "$E6_HOME"
E6_ALL_PASS=true

# Mock PostToolUse event for sharp-edge
MOCK_POST_EVENT='{"hook_event_name":"PostToolUse","tool_name":"Bash","session_id":"test-session","cwd":"/tmp"}'

# Mock PreToolUse event for skill-guard and others
MOCK_PRE_EVENT='{"hook_event_name":"PreToolUse","tool_name":"Bash","session_id":"test-session","cwd":"/tmp"}'

HOOK_BINARIES=(
    "goyoke-sharp-edge:PostToolUse:$MOCK_POST_EVENT"
    "goyoke-skill-guard:PreToolUse:$MOCK_PRE_EVENT"
    "goyoke-direct-impl-check:PreToolUse:$MOCK_PRE_EVENT"
    "goyoke-permission-gate:PreToolUse:$MOCK_PRE_EVENT"
    "goyoke-orchestrator-guard:SubagentStop:$MOCK_PRE_EVENT"
    "goyoke-instructions-audit:InstructionsLoaded:$MOCK_PRE_EVENT"
)

for entry in "${HOOK_BINARIES[@]}"; do
    IFS=':' read -r binary event mock_event <<< "$entry"

    if ! command -v "$binary" &>/dev/null; then
        skip "$binary not on PATH"
        continue
    fi

    E6_EXIT=0
    timeout 10 bash -c "
        export HOME='$E6_HOME'
        export PATH='$ORIGINAL_PATH'
        echo '$mock_event' | $binary
    " > "$TEST_ROOT/e6-${binary}.stdout" 2>"$TEST_ROOT/e6-${binary}.stderr" || E6_EXIT=$?

    if [ "$E6_EXIT" -eq 0 ]; then
        pass "$binary exits cleanly with no config (exit=0)"
    elif [ "$E6_EXIT" -eq 124 ]; then
        fail "$binary TIMED OUT with no config"
        E6_ALL_PASS=false
    else
        # Exit code 1 from some hooks is acceptable (e.g., stdin parse timeout)
        # But panic (exit 2) or signal death (exit > 128) is not
        if [ "$E6_EXIT" -gt 1 ]; then
            fail "$binary exited with code $E6_EXIT (possible panic)"
            E6_ALL_PASS=false
        else
            info "$binary exited with code $E6_EXIT (acceptable — likely stdin timeout)"
        fi
    fi

    # Check for panic in stderr
    if grep -q "panic:" "$TEST_ROOT/e6-${binary}.stderr" 2>/dev/null; then
        fail "$binary PANICKED on zero-install!"
        grep "panic:" "$TEST_ROOT/e6-${binary}.stderr" | head -3 | while read -r line; do
            info "  $line"
        done
        E6_ALL_PASS=false
    fi

    # Check for config warnings (informational)
    if grep -qi "warning.*config\|warning.*agents\|warning.*read" "$TEST_ROOT/e6-${binary}.stderr" 2>/dev/null; then
        info "$binary logged config-missing warning (expected)"
    fi
done

if $E6_ALL_PASS; then
    E6_RESULT="PASS"
else
    E6_RESULT="PARTIAL"
fi
log ""

# ═══════════════════════════════════════════════════════════════════
# RESULTS SUMMARY
# ═══════════════════════════════════════════════════════════════════
log "═══════════════════════════════════════════════════════════════"
log " RESULTS SUMMARY"
log "═══════════════════════════════════════════════════════════════"
log ""
log " E-1  --settings flag hook injection:       ${E1_RESULT:-SKIPPED}"
log " E-2  ~/.claude/settings.local.json scope:  ${E2_RESULT:-SKIPPED}"
log " E-3  Project settings.local.json:          ${E3_RESULT:-SKIPPED}"
log " E-4  Clean HOME onboarding:                ${E4_RESULT:-SKIPPED}"
log " E-5  --settings + clean HOME combined:     ${E5_RESULT:-SKIPPED}"
log " E-6  Hook graceful degradation:            ${E6_RESULT:-SKIPPED}"
log ""

# Decision matrix
log " DECISION MATRIX:"
log ""
if [ "${E1_RESULT:-}" = "PASS" ] && [ "${E5_RESULT:-}" = "PASS" ]; then
    log "${GREEN} RECOMMENDED: Use --settings flag for hook injection (E-1 + E-5 both pass)${NC}"
    log " The TUI can pass hooks via --settings without any disk writes."
    log " This is cleaner than D-6 (settings.local.json) and avoids C-1 entirely."
    log ""
    log " Implementation: Add SettingsPath or SettingsJSON to CLIDriverOpts."
    log " In buildArgs(), append: --settings <generated-hook-config>"
elif [ "${E2_RESULT:-}" = "PASS" ]; then
    log "${YELLOW} FALLBACK: D-6 approach works — ~/.claude/settings.local.json is discovered${NC}"
    log " The github-remodel plan's original approach is viable."
elif [ "${E3_RESULT:-}" = "PASS" ]; then
    log "${YELLOW} ALTERNATIVE: Use project-level settings.local.json${NC}"
    log " Write to <cwd>/.claude/settings.local.json — pollutes user project but works."
else
    log "${RED} PROBLEM: No hook injection mechanism verified!${NC}"
    log " Consider --setting-sources project with managed workspace (distribution plan approach)."
fi
log ""
if [ "${E4_RESULT:-}" = "TIMEOUT" ]; then
    log "${RED} WARNING: Clean HOME triggers interactive prompt — must seed onboarding markers${NC}"
fi
log ""
log " Full output: $TEST_ROOT/"
log " Results log: $LOG_FILE"
log "═══════════════════════════════════════════════════════════════"
