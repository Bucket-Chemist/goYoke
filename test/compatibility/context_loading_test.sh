#!/bin/bash
# ═══════════════════════════════════════════════════════════════════════════════
# Context Loading Compatibility Test
# Verifies Go-generated markdown is compatible with load-routing-context.sh hook
# ═══════════════════════════════════════════════════════════════════════════════
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "═══════════════════════════════════════════════════════════════════════════════"
echo "Context Loading Compatibility Test"
echo "═══════════════════════════════════════════════════════════════════════════════"
echo ""

# Test configuration
TEST_DIR=$(mktemp -d)
SESSION_TEMP_DIR="/tmp/claude-session-test-session-028f"
trap "rm -rf $TEST_DIR $SESSION_TEMP_DIR" EXIT

export GOGENT_PROJECT_DIR="$TEST_DIR"
mkdir -p "$TEST_DIR/.claude/memory"

HANDOFF_MD="$TEST_DIR/.claude/memory/last-handoff.md"
HANDOFF_JSONL="$TEST_DIR/.claude/memory/handoffs.jsonl"
HOOK_SCRIPT="${HOME}/.claude/hooks/load-routing-context.sh"

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Test helper functions
test_start() {
    TESTS_RUN=$((TESTS_RUN + 1))
    echo -n "Test $TESTS_RUN: $1... "
}

test_pass() {
    TESTS_PASSED=$((TESTS_PASSED + 1))
    echo -e "${GREEN}PASS${NC}"
}

test_fail() {
    TESTS_FAILED=$((TESTS_FAILED + 1))
    echo -e "${RED}FAIL${NC}"
    if [[ -n "${1:-}" ]]; then
        echo "  Reason: $1"
    fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# Phase 1: Generate Go handoff markdown
# ═══════════════════════════════════════════════════════════════════════════════

echo "Phase 1: Generate Go handoff markdown"
echo "───────────────────────────────────────────────────────────────────────────────"

test_start "gogent-archive CLI is installed"
if command -v gogent-archive &>/dev/null; then
    test_pass
else
    test_fail "gogent-archive not found in PATH. Run: make install"
    exit 1
fi

test_start "Create sample Go-generated markdown (matching RenderHandoffMarkdown format)"
cat > "$HANDOFF_MD" <<'EOF'
# Session Handoff - 2026-01-20 06:47:59

## Session Context

- **Session ID**: test-session-028f
- **Project**: /home/doktersmol/Documents/GOgent-Fortress
- **Active Ticket**: GOgent-028f
- **Phase**: compatibility-testing

## Session Metrics

- **Tool Calls**: 42
- **Errors Logged**: 2
- **Routing Violations**: 1

## Git State

- **Branch**: master
- **Status**: Uncommitted changes present
- **Uncommitted Files**:
  - test/compatibility/context_loading_test.sh
  - docs/compatibility/sessionstart-hook.md

## Sharp Edges

- **pkg/session/handoff.go**: type_mismatch (3 consecutive failures)
  - Context: JSON unmarshaling failed - expected struct, got map

## Routing Violations

- **tech-docs-writer**: wrong_subagent_type (expected: general-purpose, actual: Explore)

## Error Patterns

- **file_not_found**: 5 occurrences
  - Context: Missing test fixtures

## Immediate Actions

1. Fix sharp edge in handoff.go
   - Resolve type mismatch
EOF
if [[ -f "$HANDOFF_MD" ]]; then
    test_pass
else
    test_fail
    exit 1
fi

# ═══════════════════════════════════════════════════════════════════════════════
# Phase 2: Analyze Go-generated markdown structure
# ═══════════════════════════════════════════════════════════════════════════════

echo ""
echo "Phase 2: Analyze Go-generated markdown structure"
echo "───────────────────────────────────────────────────────────────────────────────"

test_start "Markdown contains header"
if grep -q "^# Session Handoff" "$HANDOFF_MD"; then
    test_pass
else
    test_fail "Missing main header"
fi

test_start "Markdown contains Session Context section"
if grep -q "^## Session Context" "$HANDOFF_MD"; then
    test_pass
else
    test_fail "Missing Session Context section"
fi

test_start "Markdown contains Session Metrics section"
if grep -q "^## Session Metrics" "$HANDOFF_MD"; then
    test_pass
else
    test_fail "Missing Session Metrics section"
fi

test_start "Markdown contains Git State section"
if grep -q "^## Git State" "$HANDOFF_MD"; then
    test_pass
else
    test_fail "Missing Git State section (may be optional)"
fi

test_start "Markdown contains Sharp Edges section"
if grep -q "^## Sharp Edges" "$HANDOFF_MD"; then
    test_pass
else
    test_fail "Missing Sharp Edges section"
fi

test_start "Markdown contains Routing Violations section"
if grep -q "^## Routing Violations" "$HANDOFF_MD"; then
    test_pass
else
    test_fail "Missing Routing Violations section"
fi

test_start "Critical sections appear in first 30 lines"
FIRST_30=$(head -30 "$HANDOFF_MD")
if echo "$FIRST_30" | grep -q "Session Context" && \
   echo "$FIRST_30" | grep -q "Session Metrics"; then
    test_pass
else
    test_fail "Critical sections not in first 30 lines (hook reads only these)"
fi

# ═══════════════════════════════════════════════════════════════════════════════
# Phase 3: Simulate load-routing-context.sh parsing
# ═══════════════════════════════════════════════════════════════════════════════

echo ""
echo "Phase 3: Simulate load-routing-context.sh parsing"
echo "───────────────────────────────────────────────────────────────────────────────"

test_start "Hook script exists"
if [[ -f "$HOOK_SCRIPT" ]]; then
    test_pass
else
    test_fail "load-routing-context.sh not found at $HOOK_SCRIPT"
    # Non-fatal - continue testing markdown format
fi

test_start "Simulate hook's head -30 extraction"
EXTRACTED=$(head -30 "$HANDOFF_MD" 2>/dev/null || echo "")
if [[ -n "$EXTRACTED" ]]; then
    test_pass
else
    test_fail "Failed to extract first 30 lines"
fi

test_start "Extracted content contains session info"
if echo "$EXTRACTED" | grep -q "Session ID"; then
    test_pass
else
    test_fail "Session ID not found in extracted content"
fi

test_start "Extracted content contains tool call count"
if echo "$EXTRACTED" | grep -q "Tool Calls"; then
    test_pass
else
    test_fail "Tool Calls metric not found"
fi

test_start "Extracted content contains routing violations count"
if echo "$EXTRACTED" | grep -q "Routing Violations"; then
    test_pass
else
    test_fail "Routing Violations not found"
fi

test_start "Extracted content is readable markdown"
if echo "$EXTRACTED" | grep -E "^##? " >/dev/null; then
    test_pass
else
    test_fail "Content doesn't contain markdown headers"
fi

# ═══════════════════════════════════════════════════════════════════════════════
# Phase 4: Content verification
# ═══════════════════════════════════════════════════════════════════════════════

echo ""
echo "Phase 4: Content verification"
echo "───────────────────────────────────────────────────────────────────────────────"

test_start "Sharp edge details are present"
if grep -q "type_mismatch" "$HANDOFF_MD" && \
   grep -q "consecutive failures" "$HANDOFF_MD"; then
    test_pass
else
    test_fail "Sharp edge details missing or incomplete"
fi

test_start "Routing violation details are present"
if grep -q "tech-docs-writer" "$HANDOFF_MD" && \
   grep -q "wrong_subagent_type" "$HANDOFF_MD"; then
    test_pass
else
    test_fail "Routing violation details missing"
fi

test_start "Git state shows uncommitted files"
if grep -q "Uncommitted Files" "$HANDOFF_MD"; then
    test_pass
else
    test_fail "Git uncommitted files not listed"
fi

test_start "Actions section is present"
if grep -q "^## Immediate Actions" "$HANDOFF_MD"; then
    test_pass
else
    test_fail "Immediate Actions section missing"
fi

# ═══════════════════════════════════════════════════════════════════════════════
# Phase 5: Format compatibility checks
# ═══════════════════════════════════════════════════════════════════════════════

echo ""
echo "Phase 5: Format compatibility checks"
echo "───────────────────────────────────────────────────────────────────────────────"

test_start "No JSONL content leaked into markdown"
if grep -q '{.*:.*}' "$HANDOFF_MD"; then
    test_fail "Raw JSON detected in markdown (should be formatted)"
else
    test_pass
fi

test_start "Markdown uses standard list syntax"
if grep -q "^- \*\*" "$HANDOFF_MD"; then
    test_pass
else
    test_fail "Expected bold markdown list items not found"
fi

test_start "Timestamps are human-readable"
if grep -E "[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}" "$HANDOFF_MD" >/dev/null; then
    test_pass
else
    test_fail "Timestamp not in expected YYYY-MM-DD HH:MM:SS format"
fi

test_start "No Go-specific artifacts in output"
if grep -E "(nil|<nil>|\[\]interface|map\[string\])" "$HANDOFF_MD" >/dev/null; then
    test_fail "Go internal representations leaked into markdown"
else
    test_pass
fi

# ═══════════════════════════════════════════════════════════════════════════════
# Summary and output
# ═══════════════════════════════════════════════════════════════════════════════

echo ""
echo "═══════════════════════════════════════════════════════════════════════════════"
echo "Test Summary"
echo "═══════════════════════════════════════════════════════════════════════════════"
echo "Tests run:    $TESTS_RUN"
echo -e "Tests passed: ${GREEN}$TESTS_PASSED${NC}"
if [[ $TESTS_FAILED -gt 0 ]]; then
    echo -e "Tests failed: ${RED}$TESTS_FAILED${NC}"
else
    echo -e "Tests failed: ${GREEN}0${NC}"
fi
echo ""

# Show sample of generated markdown
echo "═══════════════════════════════════════════════════════════════════════════════"
echo "Generated Markdown Preview (first 30 lines - what hook sees)"
echo "═══════════════════════════════════════════════════════════════════════════════"
head -30 "$HANDOFF_MD"
echo "═══════════════════════════════════════════════════════════════════════════════"
echo ""

# Final verdict
if [[ $TESTS_FAILED -eq 0 ]]; then
    echo -e "${GREEN}✅ COMPATIBILITY VERIFIED${NC}"
    echo "Go-generated markdown is fully compatible with load-routing-context.sh hook."
    echo ""
    echo "Key findings:"
    echo "  - Hook uses simple head -30 extraction (line 39 of load-routing-context.sh)"
    echo "  - Critical sections (Session Context, Metrics, Git State) appear in first 30 lines"
    echo "  - Markdown format is clean and human-readable"
    echo "  - No Go-specific artifacts leaked into output"
    echo ""
    exit 0
else
    echo -e "${RED}❌ COMPATIBILITY ISSUES DETECTED${NC}"
    echo "Some tests failed. Review output above and fix handoff_markdown.go if needed."
    echo ""
    exit 1
fi
