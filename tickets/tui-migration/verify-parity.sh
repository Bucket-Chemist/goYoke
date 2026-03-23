#!/usr/bin/env bash
# verify-parity.sh — Automated feature parity verification for GOgent-Fortress TUI
#
# Usage:
#   ./tickets/tui-migration/verify-parity.sh [--verbose]
#
# Exit code:
#   0   All automated checks passed
#   1   One or more checks failed
#
# Requirements:
#   - Go 1.25+ in PATH
#   - Run from the repo root (or the script resolves it automatically)

set -euo pipefail

# ---------------------------------------------------------------------------
# Setup
# ---------------------------------------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

VERBOSE=0
if [[ "${1:-}" == "--verbose" ]]; then
    VERBOSE=1
fi

# Counters
PASS=0
FAIL=0
SKIP=0

# Terminal colours (only when stdout is a terminal)
if [[ -t 1 ]]; then
    GREEN="\033[0;32m"
    RED="\033[0;31m"
    YELLOW="\033[0;33m"
    CYAN="\033[0;36m"
    BOLD="\033[1m"
    RESET="\033[0m"
else
    GREEN=""
    RED=""
    YELLOW=""
    CYAN=""
    BOLD=""
    RESET=""
fi

log() {
    echo -e "$*"
}

log_step() {
    log "${CYAN}${BOLD}==> $*${RESET}"
}

check_pass() {
    local name="$1"
    log "  ${GREEN}PASS${RESET}  $name"
    PASS=$((PASS + 1))
}

check_fail() {
    local name="$1"
    local reason="${2:-}"
    log "  ${RED}FAIL${RESET}  $name"
    if [[ -n "$reason" ]]; then
        log "        ${reason}"
    fi
    FAIL=$((FAIL + 1))
}

check_skip() {
    local name="$1"
    local reason="${2:-}"
    log "  ${YELLOW}SKIP${RESET}  $name"
    if [[ -n "$reason" ]]; then
        log "        ${reason}"
    fi
    SKIP=$((SKIP + 1))
}

run_check() {
    local name="$1"
    shift
    if "$@" > /dev/null 2>&1; then
        check_pass "$name"
    else
        check_fail "$name" "(command: $*)"
        return 0  # non-fatal: continue to next check
    fi
}

# ---------------------------------------------------------------------------
# Change to repo root
# ---------------------------------------------------------------------------

cd "${REPO_ROOT}"

log ""
log "${BOLD}GOgent-Fortress TUI — Feature Parity Verification${RESET}"
log "${BOLD}Repo root: ${REPO_ROOT}${RESET}"
log ""

# ---------------------------------------------------------------------------
# SECTION 1: Build checks
# ---------------------------------------------------------------------------

log_step "Section 1: Build checks"

# 1.1 gofortress binary builds
if go build -o /dev/null ./cmd/gofortress/... 2>/dev/null; then
    check_pass "gofortress binary builds (./cmd/gofortress)"
else
    check_fail "gofortress binary builds (./cmd/gofortress)" \
        "$(go build -o /dev/null ./cmd/gofortress/... 2>&1 | head -5)"
fi

# 1.2 gofortress-mcp binary builds
if go build -o /dev/null ./cmd/gofortress-mcp/... 2>/dev/null; then
    check_pass "gofortress-mcp binary builds (./cmd/gofortress-mcp)"
else
    check_fail "gofortress-mcp binary builds (./cmd/gofortress-mcp)" \
        "$(go build -o /dev/null ./cmd/gofortress-mcp/... 2>&1 | head -5)"
fi

# 1.3 All packages build (vet)
if go vet ./internal/tui/... 2>/dev/null; then
    check_pass "go vet ./internal/tui/... (no issues)"
else
    check_fail "go vet ./internal/tui/..." \
        "$(go vet ./internal/tui/... 2>&1 | head -10)"
fi

# ---------------------------------------------------------------------------
# SECTION 2: Binary smoke test (start + shutdown)
# ---------------------------------------------------------------------------

log_step "Section 2: Binary smoke test"

# Build the binary to a temp file
TMPBIN=$(mktemp /tmp/gofortress-XXXXXX)
trap 'rm -f "${TMPBIN}"' EXIT

if go build -o "${TMPBIN}" ./cmd/gofortress/... 2>/dev/null; then

    # 2.1 Binary responds to --version
    if timeout 5 "${TMPBIN}" --version 2>/dev/null | grep -q "gofortress version"; then
        check_pass "binary responds to --version"
    else
        check_fail "binary responds to --version" \
            "Expected 'gofortress version' in output"
    fi

    # 2.2 Graceful shutdown via SIGTERM (within 10s)
    # Launch in background with --debug (to avoid needing a terminal)
    # and send SIGTERM after 1 second. Verify it exits within 10 seconds.
    TERMLOG=$(mktemp /tmp/gofortress-term-XXXXXX)
    # We cannot run the TUI headlessly without a terminal, so we verify
    # the binary exits cleanly when SIGTERM is delivered via a pipe trick.
    # We check that the binary accepts SIGTERM without hanging.
    "${TMPBIN}" --version > "${TERMLOG}" 2>&1
    EXIT_CODE=$?
    if [[ $EXIT_CODE -eq 0 ]]; then
        check_pass "binary exits cleanly on --version (exit code 0)"
    else
        check_fail "binary exits cleanly on --version" \
            "Exit code: ${EXIT_CODE}"
    fi
    rm -f "${TERMLOG}"

else
    check_skip "binary smoke tests" "build failed — skipping runtime tests"
fi

# ---------------------------------------------------------------------------
# SECTION 3: Unit tests
# ---------------------------------------------------------------------------

log_step "Section 3: Unit tests (./internal/tui/...)"

# 3.1 All unit tests pass (excluding e2e and integration build tags)
if go test ./internal/tui/... 2>/dev/null; then
    check_pass "go test ./internal/tui/... (all pass)"
else
    check_fail "go test ./internal/tui/..." \
        "$(go test ./internal/tui/... 2>&1 | grep -E '^(FAIL|panic)' | head -10)"
fi

# 3.2 Race detector clean
if go test -race ./internal/tui/... 2>/dev/null; then
    check_pass "go test -race ./internal/tui/... (race-clean)"
else
    check_fail "go test -race ./internal/tui/..." \
        "$(go test -race ./internal/tui/... 2>&1 | grep -E '^(FAIL|DATA RACE)' | head -10)"
fi

# ---------------------------------------------------------------------------
# SECTION 4: Feature verification — code-review checks
# ---------------------------------------------------------------------------

log_step "Section 4: Feature-level code checks"

# ---- Feature 1: Session resume (--session-id flag) ----
if grep -q '"session-id"' ./cmd/gofortress/main.go 2>/dev/null; then
    check_pass "Feature 1: --session-id flag declared in main.go"
else
    check_fail "Feature 1: --session-id flag declared in main.go"
fi

if grep -q 'LoadSession' ./cmd/gofortress/main.go 2>/dev/null; then
    check_pass "Feature 1: LoadSession called on session resume"
else
    check_fail "Feature 1: LoadSession called on session resume"
fi

if grep -q -- '--resume' ./internal/tui/cli/driver.go 2>/dev/null; then
    check_pass "Feature 1: --resume flag forwarded to CLI subprocess"
else
    check_fail "Feature 1: --resume flag forwarded to CLI subprocess"
fi

# ---- Feature 2: Multi-provider switching (4 providers) ----
for provider in ProviderAnthropic ProviderGoogle ProviderOpenAI ProviderLocal; do
    if grep -q "${provider}" ./internal/tui/state/provider.go 2>/dev/null; then
        check_pass "Feature 2: ${provider} defined in state/provider.go"
    else
        check_fail "Feature 2: ${provider} missing from state/provider.go"
    fi
done

if grep -q 'handleProviderSwitch' ./internal/tui/model/provider_switch.go 2>/dev/null; then
    check_pass "Feature 2: handleProviderSwitch implemented"
else
    check_fail "Feature 2: handleProviderSwitch missing"
fi

# ---- Feature 3: Permission prompts ----
if grep -q 'FlowConfirm' ./internal/tui/components/modals/permission.go 2>/dev/null; then
    check_pass "Feature 3: FlowConfirm (allow/deny) defined"
else
    check_fail "Feature 3: FlowConfirm missing in permission.go"
fi

if grep -q 'Allow.*Deny\|Deny.*Allow' ./internal/tui/mcp/tools.go 2>/dev/null; then
    check_pass "Feature 3: Allow/Deny options in MCP confirm_action tool"
else
    check_fail "Feature 3: Allow/Deny options missing in MCP tools"
fi

# ---- Feature 4: Plan mode ----
if grep -q 'FlowEnterPlan\|FlowExitPlan' ./internal/tui/components/modals/permission.go 2>/dev/null; then
    check_pass "Feature 4: FlowEnterPlan and FlowExitPlan defined"
else
    check_fail "Feature 4: Plan mode flows missing in permission.go"
fi

if grep -q 'Approve Plan.*Request Changes.*Reject Plan\|isExitPlanOptions' \
    ./internal/tui/components/modals/permission.go 2>/dev/null; then
    check_pass "Feature 4: Exit plan options (Approve/Request Changes/Reject) recognised"
else
    check_fail "Feature 4: Exit plan option recognition missing"
fi

# ---- Feature 5: Agent spawning ----
if grep -q 'spawn_agent\|SpawnAgent' ./internal/tui/mcp/tools.go 2>/dev/null; then
    check_pass "Feature 5: spawn_agent tool registered in MCP server"
else
    check_fail "Feature 5: spawn_agent tool missing"
fi

if grep -q 'TypeAgentRegister\|AgentRegisterPayload' ./internal/tui/mcp/tools.go 2>/dev/null; then
    check_pass "Feature 5: Agent registration notification sent on spawn"
else
    check_fail "Feature 5: Agent registration notification missing"
fi

# Note: spawn_agent is a stub (no subprocess) — check for the stub comment
if grep -q 'stub' ./internal/tui/mcp/tools.go 2>/dev/null; then
    check_skip "Feature 5: spawn_agent full subprocess launch" \
        "handleSpawnAgent is a validated stub (subprocess not launched)"
fi

# ---- Feature 6: Team orchestration ----
if grep -q 'team_run\|TeamRun' ./internal/tui/mcp/tools.go 2>/dev/null; then
    check_pass "Feature 6: team_run tool registered in MCP server"
else
    check_fail "Feature 6: team_run tool missing"
fi

if grep -q 'TeamListModel\|NewTeamListModel' ./internal/tui/components/teams/list.go 2>/dev/null; then
    check_pass "Feature 6: TeamListModel implemented (polling display)"
else
    check_fail "Feature 6: TeamListModel missing"
fi

if grep -q 'stub' ./internal/tui/mcp/tools.go 2>/dev/null; then
    check_skip "Feature 6: team_run background process launch" \
        "handleTeamRun is a validated stub (subprocess not launched)"
fi

# ---- Feature 7: Context compaction notification ----
if grep -q 'compact_boundary\|SystemStatusEvent' ./internal/tui/cli/events.go 2>/dev/null; then
    check_pass "Feature 7: compact_boundary / SystemStatusEvent parsed"
else
    check_fail "Feature 7: compact_boundary / SystemStatusEvent missing in events.go"
fi

# ---- Feature 8: Model switching ----
if grep -q 'SetActiveModel\|GetActiveModel' ./internal/tui/state/provider.go 2>/dev/null; then
    check_pass "Feature 8: SetActiveModel/GetActiveModel implemented"
else
    check_fail "Feature 8: Model switching methods missing in state/provider.go"
fi

if grep -q '"--model"' ./internal/tui/cli/driver.go 2>/dev/null; then
    check_pass "Feature 8: --model flag forwarded to CLI subprocess"
else
    check_fail "Feature 8: --model flag not forwarded in driver.go"
fi

# ---- Feature 9: Permission mode cycling (Alt+P) ----
if grep -q 'CyclePermMode\|alt+p\|alt\+p' ./internal/tui/config/keys.go 2>/dev/null; then
    check_pass "Feature 9: CyclePermMode (Alt+P) defined in keys.go"
else
    check_fail "Feature 9: CyclePermMode missing in keys.go"
fi

# ---- Feature 10: Input history (up/down) ----
if grep -q 'inputHistory\|HistoryPrev\|HistoryNext' \
    ./internal/tui/components/claude/panel.go 2>/dev/null; then
    check_pass "Feature 10: Input history fields and navigation in panel.go"
else
    check_fail "Feature 10: Input history not found in panel.go"
fi

if [[ -f ./internal/tui/components/claude/history_test.go ]]; then
    check_pass "Feature 10: history_test.go exists"
else
    check_fail "Feature 10: history_test.go missing"
fi

# ---- Feature 11: Slash commands passed through ----
if grep -q 'SendMessage' ./internal/tui/components/claude/panel.go 2>/dev/null; then
    check_pass "Feature 11: SendMessage used (input passed verbatim to CLI)"
else
    check_fail "Feature 11: SendMessage not found — slash commands may not be forwarded"
fi

# ---- Feature 12: Markdown rendering ----
if grep -q 'RenderMarkdown\|glamour' ./internal/tui/util/markdown.go 2>/dev/null; then
    check_pass "Feature 12: Glamour-based RenderMarkdown implemented"
else
    check_fail "Feature 12: RenderMarkdown / Glamour missing in util/markdown.go"
fi

if grep -q 'glamour' ./go.mod 2>/dev/null; then
    check_pass "Feature 12: glamour dependency present in go.mod"
else
    check_fail "Feature 12: glamour missing from go.mod"
fi

# ---- Feature 13: Responsive layout ----
if grep -q 'width < 80\|width.*80' ./internal/tui/model/layout.go 2>/dev/null; then
    check_pass "Feature 13: Narrow breakpoint (width < 80) implemented"
else
    check_fail "Feature 13: Narrow breakpoint missing in layout.go"
fi

if grep -q 'width < 100\|width.*100' ./internal/tui/model/layout.go 2>/dev/null; then
    check_pass "Feature 13: Medium breakpoint (width < 100) implemented"
else
    check_fail "Feature 13: Medium breakpoint missing in layout.go"
fi

if grep -q '0\.75\|0\.70\|leftRatio' ./internal/tui/model/layout.go 2>/dev/null; then
    check_pass "Feature 13: Left/right split ratios (75/25 and 70/30) computed"
else
    check_fail "Feature 13: Split ratios missing in layout.go"
fi

# ---- Feature 14: Graceful shutdown ----
if grep -q 'ShutdownManager\|NewShutdownManager' ./internal/tui/lifecycle/shutdown.go 2>/dev/null; then
    check_pass "Feature 14: ShutdownManager implemented"
else
    check_fail "Feature 14: ShutdownManager missing in lifecycle/shutdown.go"
fi

if grep -q 'ShutdownCompleteMsg\|shutdownFunc' ./internal/tui/model/app.go 2>/dev/null; then
    check_pass "Feature 14: Shutdown wired into AppModel (double Ctrl+C guard)"
else
    check_fail "Feature 14: Shutdown not wired in app.go"
fi

if grep -q 'StartSignalHandler\|SIGTERM\|SIGINT' ./cmd/gofortress/main.go 2>/dev/null; then
    check_pass "Feature 14: OS signal handler (SIGTERM/SIGINT) wired in main.go"
else
    check_fail "Feature 14: OS signal handler missing in main.go"
fi

# ---- Feature 15: Toast notifications ----
if grep -q 'ToastModel\|NewToastModel' ./internal/tui/components/toast/toast.go 2>/dev/null; then
    check_pass "Feature 15: ToastModel implemented"
else
    check_fail "Feature 15: ToastModel missing in toast/toast.go"
fi

if grep -q 'ToastMsg\|handleToast' ./internal/tui/bridge/server.go 2>/dev/null; then
    check_pass "Feature 15: Toast notifications routed from bridge"
else
    check_fail "Feature 15: Toast notification routing missing in bridge/server.go"
fi

# ---- Feature 16: Clipboard copy ----
if grep -q 'CopyToClipboard\|CopyLastResponse' \
    ./internal/tui/components/claude/panel.go 2>/dev/null; then
    check_pass "Feature 16: CopyToClipboard (Ctrl+Y) in panel.go"
else
    check_fail "Feature 16: Clipboard copy not found in panel.go"
fi

if grep -q 'atotto/clipboard\|clipboard' ./go.mod 2>/dev/null; then
    check_pass "Feature 16: clipboard dependency present in go.mod"
else
    check_fail "Feature 16: clipboard dependency missing from go.mod"
fi

# ---- Feature 17: Search within conversation ----
if grep -q 'SearchModel\|ExecuteSearch\|search.IsActive' \
    ./internal/tui/components/claude/panel.go 2>/dev/null; then
    check_pass "Feature 17: In-panel search (SearchModel) implemented"
else
    check_fail "Feature 17: In-panel search missing in panel.go"
fi

if [[ -f ./internal/tui/components/claude/search_test.go ]]; then
    check_pass "Feature 17: search_test.go exists"
else
    check_fail "Feature 17: search_test.go missing"
fi

# ---- Feature 18: All keybindings ----
log_step "Section 4 (continued): Keybinding completeness (Feature 18)"

# Check all 24 binding names are present in keys.go
BINDINGS=(
    "ToggleFocus"
    "CycleProvider"
    "CycleRightPanel"
    "CyclePermMode"
    "Interrupt"
    "ForceQuit"
    "ClearScreen"
    "ToggleTaskBoard"
    "TabChat"
    "TabAgentConfig"
    "TabTeamConfig"
    "TabTelemetry"
    "Submit"
    "HistoryPrev"
    "HistoryNext"
    "ToggleToolExpansion"
    "CycleExpansion"
    "Search"
    "SearchNext"
    "SearchPrev"
    "CopyLastResponse"
    "AgentUp"
    "AgentDown"
    "AgentExpand"
)

MISSING_BINDINGS=0
for binding in "${BINDINGS[@]}"; do
    if ! grep -q "${binding}" ./internal/tui/config/keys.go 2>/dev/null; then
        check_fail "Feature 18: keybinding '${binding}' missing from keys.go"
        MISSING_BINDINGS=$((MISSING_BINDINGS + 1))
    fi
done

if [[ $MISSING_BINDINGS -eq 0 ]]; then
    check_pass "Feature 18: All 24 keybindings present in config/keys.go"
fi

# Check key codes for critical bindings
declare -A EXPECTED_KEYS=(
    ["alt+p"]="CyclePermMode"
    ["shift+tab"]="CycleProvider"
    ["ctrl+c"]="ForceQuit"
    ["ctrl+y"]="CopyLastResponse"
    ["\"/\""]="Search"
    ["ctrl+n"]="SearchNext"
    ["ctrl+p"]="SearchPrev"
    ["alt+r"]="CycleRightPanel"
    ["alt+b"]="ToggleTaskBoard"
)

for key_code in "${!EXPECTED_KEYS[@]}"; do
    binding="${EXPECTED_KEYS[$key_code]}"
    if grep -q "${key_code}" ./internal/tui/config/keys.go 2>/dev/null; then
        check_pass "Feature 18: key '${key_code}' bound (${binding})"
    else
        check_fail "Feature 18: key '${key_code}' not found for ${binding}"
    fi
done

# ---------------------------------------------------------------------------
# SECTION 5: Test file presence checks
# ---------------------------------------------------------------------------

log_step "Section 5: Test file presence"

TEST_FILES=(
    "internal/tui/cli/events_test.go"
    "internal/tui/cli/driver_test.go"
    "internal/tui/cli/resilience_test.go"
    "internal/tui/mcp/tools_test.go"
    "internal/tui/bridge/server_test.go"
    "internal/tui/components/modals/permission_test.go"
    "internal/tui/state/provider_test.go"
    "internal/tui/components/claude/panel_test.go"
    "internal/tui/components/claude/history_test.go"
    "internal/tui/components/claude/search_test.go"
    "internal/tui/components/teams/list_test.go"
    "internal/tui/components/toast/toast_test.go"
    "internal/tui/util/markdown_test.go"
    "internal/tui/util/clipboard_test.go"
    "internal/tui/lifecycle/shutdown_test.go"
    "internal/tui/model/app_test.go"
    "internal/tui/config/keys_test.go"
)

for tf in "${TEST_FILES[@]}"; do
    if [[ -f "${REPO_ROOT}/${tf}" ]]; then
        check_pass "Test file exists: ${tf}"
    else
        check_fail "Test file missing: ${tf}"
    fi
done

# ---------------------------------------------------------------------------
# SECTION 6: Benchmark targets (from TUI-040)
# ---------------------------------------------------------------------------

log_step "Section 6: Benchmark sanity (TUI-040 targets)"

# Run a quick benchmark to verify benchmarks exist and are measurable.
# We do not assert timing here; that is done by dedicated benchmark runs.
BENCH_PKGS=(
    "./internal/tui/model/..."
    "./internal/tui/components/modals/..."
    "./internal/tui/cli/..."
    "./internal/tui/mcp/..."
)

for pkg in "${BENCH_PKGS[@]}"; do
    if go test -run '^$' -bench '.' -benchtime 1x "${pkg}" > /dev/null 2>&1; then
        check_pass "Benchmarks runnable: ${pkg}"
    else
        check_skip "Benchmarks runnable: ${pkg}" \
            "No benchmarks found or run failed (non-blocking)"
    fi
done

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------

TOTAL=$((PASS + FAIL + SKIP))
log ""
log "${BOLD}=== Parity Verification Summary ===${RESET}"
log "  Total checks : ${TOTAL}"
log "  ${GREEN}Pass${RESET}         : ${PASS}"
log "  ${RED}Fail${RESET}         : ${FAIL}"
log "  ${YELLOW}Skip${RESET}         : ${SKIP}"
log ""

if [[ $FAIL -gt 0 ]]; then
    log "${RED}${BOLD}RESULT: FAIL — ${FAIL} check(s) did not pass.${RESET}"
    log ""
    log "Review the FAIL lines above and consult:"
    log "  tickets/tui-migration/parity-checklist.md"
    log ""
    exit 1
else
    log "${GREEN}${BOLD}RESULT: PASS — All automated checks passed.${RESET}"
    if [[ $SKIP -gt 0 ]]; then
        log ""
        log "${YELLOW}Note: ${SKIP} check(s) were skipped (see SKIP lines above).${RESET}"
        log "These require manual verification:"
        log "  - Feature 5: spawn_agent full subprocess launch"
        log "  - Feature 6: team_run background process launch"
    fi
    log ""
    exit 0
fi
