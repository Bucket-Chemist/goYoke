#!/bin/bash

# Verification Script for GOgent-010 to GOgent-019 Discrepancy Fixes
# Purpose: Verify all 21 discrepancies have been resolved
# Date: 2026-01-17

# Don't use set -e as we want to collect all failures, not exit on first

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INDEX_FILE="$SCRIPT_DIR/tickets-index.json"
DOC_FILE="$SCRIPT_DIR/02-week1-overrides-permissions.md"

echo "=========================================="
echo "Discrepancy Verification for GOgent-010 to GOgent-019"
echo "=========================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
PASS=0
FAIL=0

# Function to check a value
check_value() {
    local ticket="$1"
    local field="$2"
    local expected="$3"
    local actual="$4"

    if [[ "$actual" == "$expected" ]]; then
        echo -e "${GREEN}✓${NC} $ticket $field: $actual"
        ((PASS++))
    else
        echo -e "${RED}✗${NC} $ticket $field: Expected '$expected', Got '$actual'"
        ((FAIL++))
    fi
}

# Extract JSON field using jq
get_json_field() {
    local ticket="$1"
    local field="$2"
    jq -r ".tickets[] | select(.id == \"$ticket\") | .$field" "$INDEX_FILE"
}

# Extract JSON array field
get_json_array() {
    local ticket="$1"
    local field="$2"
    jq -r ".tickets[] | select(.id == \"$ticket\") | .$field | join(\", \")" "$INDEX_FILE"
}

echo "=== CRITICAL: Filename Mismatches (7 issues) ==="
echo ""

# GOgent-010: File conflict fix
echo "GOgent-010: Override Flags and XDG Paths"
check_value "GOgent-010" "title" "Implement Override Flags and XDG Paths" "$(get_json_field GOgent-010 title)"
check_value "GOgent-010" "files_to_create" "pkg/routing/overrides.go, pkg/config/paths.go, pkg/routing/overrides_test.go" "$(get_json_array GOgent-010 files_to_create)"
check_value "GOgent-010" "dependencies" "GOgent-007" "$(get_json_array GOgent-010 dependencies)"
echo ""

# GOgent-013: scout.go → metrics.go
echo "GOgent-013: Scout Metrics Loading"
check_value "GOgent-013" "files_to_create" "pkg/routing/metrics.go, pkg/routing/metrics_test.go" "$(get_json_array GOgent-013 files_to_create)"
check_value "GOgent-013" "time_estimate" "2h" "$(get_json_field GOgent-013 time_estimate)"
echo ""

# GOgent-014: Extend metrics.go
echo "GOgent-014: Metrics Freshness Check"
check_value "GOgent-014" "files_to_create" "pkg/routing/metrics.go (extend)" "$(get_json_array GOgent-014 files_to_create)"
check_value "GOgent-014" "time_estimate" "1.5h" "$(get_json_field GOgent-014 time_estimate)"
check_value "GOgent-014" "acceptance_criteria_count" "6" "$(get_json_field GOgent-014 acceptance_criteria_count)"
echo ""

# GOgent-015: Time estimate only
echo "GOgent-015: Tier Update from Complexity"
check_value "GOgent-015" "time_estimate" "2h" "$(get_json_field GOgent-015 time_estimate)"
echo ""

# GOgent-016: Filename + time
echo "GOgent-016: Complexity Routing Tests"
check_value "GOgent-016" "files_to_create" "test/integration/complexity_routing_test.go" "$(get_json_array GOgent-016 files_to_create)"
check_value "GOgent-016" "time_estimate" "1.5h" "$(get_json_field GOgent-016 time_estimate)"
check_value "GOgent-016" "acceptance_criteria_count" "6" "$(get_json_field GOgent-016 acceptance_criteria_count)"
echo ""

# GOgent-017: tool_permissions → permissions
echo "GOgent-017: Tool Permission Checks"
check_value "GOgent-017" "files_to_create" "pkg/routing/permissions.go, pkg/routing/permissions_test.go" "$(get_json_array GOgent-017 files_to_create)"
check_value "GOgent-017" "time_estimate" "2.5h" "$(get_json_field GOgent-017 time_estimate)"
check_value "GOgent-017" "acceptance_criteria_count" "6" "$(get_json_field GOgent-017 acceptance_criteria_count)"
echo ""

# GOgent-018: Extend permissions.go
echo "GOgent-018: Wildcard Tools Handling"
check_value "GOgent-018" "files_to_create" "pkg/routing/permissions.go (extend)" "$(get_json_array GOgent-018 files_to_create)"
check_value "GOgent-018" "acceptance_criteria_count" "7" "$(get_json_field GOgent-018 acceptance_criteria_count)"
echo ""

# GOgent-019: Filename
echo "GOgent-019: Tool Permission Tests"
check_value "GOgent-019" "files_to_create" "test/integration/tool_permissions_test.go" "$(get_json_array GOgent-019 files_to_create)"
check_value "GOgent-019" "time_estimate" "2h" "$(get_json_field GOgent-019 time_estimate)"
check_value "GOgent-019" "acceptance_criteria_count" "7" "$(get_json_field GOgent-019 acceptance_criteria_count)"
echo ""

echo "=========================================="
echo "Verification Summary"
echo "=========================================="
echo -e "${GREEN}Passed:${NC} $PASS checks"
echo -e "${RED}Failed:${NC} $FAIL checks"
echo ""

if [[ $FAIL -eq 0 ]]; then
    echo -e "${GREEN}✓ All 21 discrepancies resolved!${NC}"
    echo ""
    echo "Next steps:"
    echo "  1. Review changes: git diff tickets-index.json"
    echo "  2. Commit updates: git add tickets-index.json && git commit -m 'fix: resolve GOgent-010 to 019 index discrepancies'"
    echo "  3. Proceed with Week 2 ticket review"
    exit 0
else
    echo -e "${RED}✗ Some discrepancies remain. Please review and fix.${NC}"
    exit 1
fi
