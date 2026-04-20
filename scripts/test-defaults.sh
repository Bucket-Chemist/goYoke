#!/usr/bin/env bash
# test-defaults.sh — Smoke test for generate-defaults.sh output
#
# Validates that defaults/ contains the expected public content
# and no private content leaked through.
#
# Exit 0 on pass, exit 1 on any failure.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DEST="${PROJECT_ROOT}/defaults"

PASS=0
FAIL=0

check() {
    local desc="$1"
    shift
    if "$@" >/dev/null 2>&1; then
        echo "  PASS: $desc"
        PASS=$((PASS + 1))
    else
        echo "  FAIL: $desc"
        FAIL=$((FAIL + 1))
    fi
}

check_not() {
    local desc="$1"
    shift
    if ! "$@" >/dev/null 2>&1; then
        echo "  PASS: $desc"
        PASS=$((PASS + 1))
    else
        echo "  FAIL: $desc"
        FAIL=$((FAIL + 1))
    fi
}

echo "[test-defaults] Running smoke tests..."

# 1. Critical files exist
echo ""
echo "=== Critical Files ==="
check "agents-index.json exists" test -f "${DEST}/agents/agents-index.json"
check "routing-schema.json exists" test -f "${DEST}/routing-schema.json"
check "CLAUDE.md exists" test -f "${DEST}/CLAUDE.md"
check "settings-template.json exists" test -f "${DEST}/settings-template.json"

# 2. Agent count (at least 40 of 46 expected public agents)
echo ""
echo "=== Agent Count ==="
AGENT_COUNT=$(find "${DEST}/agents" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | wc -l)
echo "  Found $AGENT_COUNT agent directories"
check "at least 40 public agents" test "$AGENT_COUNT" -ge 40

# 3. No private agents
echo ""
echo "=== Private Agent Exclusion ==="
PRIVATE_AGENTS="genomics-reviewer proteomics-reviewer proteogenomics-reviewer proteoform-reviewer mass-spec-reviewer bioinformatician-reviewer staff-bioinformatician pasteur"
for agent in $PRIVATE_AGENTS; do
    check_not "no ${agent} directory" test -d "${DEST}/agents/${agent}"
done

# 4. No private conventions
echo ""
echo "=== Private Convention Exclusion ==="
check_not "no python-datasci.md" test -f "${DEST}/conventions/python-datasci.md"
check_not "no python-ml.md" test -f "${DEST}/conventions/python-ml.md"

# 5. No dotfiles (except .gitkeep)
echo ""
echo "=== Dotfile Check ==="
DOTFILES=$(find "${DEST}" -name ".*" -not -name ".gitkeep" -not -path "${DEST}" 2>/dev/null || true)
if [ -z "$DOTFILES" ]; then
    echo "  PASS: no dotfiles found"
    PASS=$((PASS + 1))
else
    echo "  FAIL: dotfiles found: $DOTFILES"
    FAIL=$((FAIL + 1))
fi

# 6. No distribution=private in agents-index.json
echo ""
echo "=== Index Content ==="
PRIVATE_IN_INDEX=$(jq -r '.agents[] | select(.distribution == "private") | .id' "${DEST}/agents/agents-index.json" 2>/dev/null || true)
if [ -z "$PRIVATE_IN_INDEX" ]; then
    echo "  PASS: no distribution=private agents in index"
    PASS=$((PASS + 1))
else
    echo "  FAIL: private agents in index: $PRIVATE_IN_INDEX"
    FAIL=$((FAIL + 1))
fi

# 7. Build check
echo ""
echo "=== Build Check ==="
if go build ./defaults/... 2>&1; then
    echo "  PASS: go build ./defaults/... succeeds"
    PASS=$((PASS + 1))
else
    echo "  FAIL: go build ./defaults/... failed"
    FAIL=$((FAIL + 1))
fi

# Summary
echo ""
echo "=== Results ==="
echo "  Passed: $PASS"
echo "  Failed: $FAIL"

if [ "$FAIL" -gt 0 ]; then
    echo ""
    echo "SMOKE TEST FAILED"
    exit 1
fi

echo ""
echo "SMOKE TEST PASSED"
exit 0
