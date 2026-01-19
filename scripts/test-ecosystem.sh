#!/usr/bin/env bash
# test-ecosystem.sh - Comprehensive test wrapper for GOgent-Fortress ecosystem
#
# This script runs all tests in the correct order with proper reporting.
# Goal: Replace 15+ manual test invocations with a single command.
#
# Usage:
#   ./test-ecosystem.sh
#
# Audit Trail:
#   Test outputs are saved to test/audit/GOgent-XXX/ for persistent tracking.
#   Ticket label is auto-detected from:
#     1. ENV var: GOgent_TICKET (e.g., export GOgent_TICKET=003)
#     2. Git branch name (e.g., feature/GOgent-003 -> 003)
#     3. Fallback: Current date (YYYY-MM-DD)

set -euo pipefail

# ANSI color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RESET='\033[0m'

# Detect ticket label for audit trail
detect_ticket_label() {
    # Priority 1: Explicit ENV var
    if [[ -n "${GOgent_TICKET:-}" ]]; then
        echo "GOgent-${GOgent_TICKET}"
        return
    fi

    # Priority 2: Git branch name (feature/GOgent-003 -> GOgent-003)
    if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
        BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "")
        if [[ "$BRANCH" =~ GOgent-([0-9]+[a-z]?) ]]; then
            echo "GOgent-${BASH_REMATCH[1]}"
            return
        fi
    fi

    # Priority 3: Fallback to current date
    date +%Y-%m-%d
}

# Set up audit directory
TICKET_LABEL=$(detect_ticket_label)
AUDIT_DIR="test/audit/${TICKET_LABEL}"
mkdir -p "$AUDIT_DIR"

# Record timestamp
date -Iseconds > "$AUDIT_DIR/timestamp.txt"

echo -e "${BLUE}========================================${RESET}"
echo -e "${BLUE}GOgent-Fortress Ecosystem Test Suite${RESET}"
echo -e "${BLUE}========================================${RESET}"
echo -e "${BLUE}Audit Label: ${TICKET_LABEL}${RESET}"
echo ""

# Track overall status
FAILED=0

# Step 1: Run unit tests
echo -e "${YELLOW}[1/4] Running unit tests...${RESET}"
if go test -v ./pkg/routing/... 2>&1 | tee "$AUDIT_DIR/unit-tests.log"; then
    echo -e "${GREEN}✅ Unit tests passed${RESET}"
else
    echo -e "${RED}❌ Unit tests failed${RESET}"
    FAILED=1
fi
echo ""

# Step 2: Run integration tests
echo -e "${YELLOW}[2/4] Running integration tests...${RESET}"
if go test -v -run 'TestEcosystem_' ./pkg/routing 2>&1 | tee "$AUDIT_DIR/integration-tests.log"; then
    echo -e "${GREEN}✅ Integration tests passed${RESET}"

    # Extract and display ecosystem test results
    echo ""
    echo -e "${BLUE}Ecosystem Test Results:${RESET}"
    grep '✅' "$AUDIT_DIR/integration-tests.log" || true
else
    echo -e "${RED}❌ Integration tests failed${RESET}"
    FAILED=1
fi
echo ""

# Step 3: Run race detector
echo -e "${YELLOW}[3/4] Running race detector...${RESET}"
if go test -race ./pkg/routing/... 2>&1 | tee "$AUDIT_DIR/race-detector.log"; then
    echo -e "${GREEN}✅ Race detector clean${RESET}"
else
    echo -e "${RED}❌ Race conditions detected${RESET}"
    FAILED=1
fi
echo ""

# Step 4: Generate coverage report
echo -e "${YELLOW}[4/4] Generating coverage report...${RESET}"
if go test -coverprofile="$AUDIT_DIR/coverage.out" ./pkg/routing/...; then
    COVERAGE=$(go tool cover -func="$AUDIT_DIR/coverage.out" | grep total | awk '{print $3}')
    echo -e "${GREEN}✅ Coverage: ${COVERAGE}${RESET}"

    # Check coverage threshold
    COVERAGE_NUM=$(echo "$COVERAGE" | sed 's/%//')
    if (( $(echo "$COVERAGE_NUM >= 80" | bc -l) )); then
        echo -e "${GREEN}   Coverage meets ≥80% requirement${RESET}"
    else
        echo -e "${RED}   WARNING: Coverage below 80% threshold${RESET}"
        FAILED=1
    fi

    # Save coverage summary
    echo "Coverage: ${COVERAGE}" > "$AUDIT_DIR/coverage-summary.txt"
else
    echo -e "${RED}❌ Coverage generation failed${RESET}"
    FAILED=1
fi
echo ""

# Update symlink to latest audit
ln -sfn "$TICKET_LABEL" test/audit/latest

# Final summary
echo -e "${BLUE}========================================${RESET}"
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ ALL ECOSYSTEM TESTS PASSED${RESET}"
    echo -e "${BLUE}📁 Audit saved to: ${AUDIT_DIR}${RESET}"
    echo -e "${BLUE}========================================${RESET}"
    exit 0
else
    echo -e "${RED}❌ SOME TESTS FAILED${RESET}"
    echo -e "${BLUE}📁 Audit saved to: ${AUDIT_DIR}${RESET}"
    echo -e "${BLUE}========================================${RESET}"
    exit 1
fi
