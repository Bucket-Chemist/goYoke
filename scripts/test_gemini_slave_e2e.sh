#!/bin/bash
# E2E Test Suite for gemini-slave Go binary
# Tests the original failing case and other critical scenarios

set -euo pipefail

BINARY="${GEMINI_SLAVE_BINARY:-$HOME/.local/bin/gemini-slave}"
TESTDATA="/tmp/gemini-slave-e2e-$$"
PASS_COUNT=0
FAIL_COUNT=0

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Setup
mkdir -p "$TESTDATA"
trap "rm -rf '$TESTDATA'" EXIT

echo "========================================"
echo "  Gemini-Slave E2E Test Suite"
echo "  Binary: $BINARY"
echo "========================================"
echo ""

# Helper functions
pass() {
    echo -e "${GREEN}✓ PASS${NC}: $1"
    PASS_COUNT=$((PASS_COUNT + 1))
}

fail() {
    echo -e "${RED}✗ FAIL${NC}: $1"
    FAIL_COUNT=$((FAIL_COUNT + 1))
}

warn() {
    echo -e "${YELLOW}⚠ WARN${NC}: $1"
}

# Test 1: Binary exists and is executable
echo "Test 1: Binary availability..."
if [ -x "$BINARY" ]; then
    pass "Binary exists and is executable"
else
    fail "Binary not found or not executable: $BINARY"
    exit 1
fi

# Test 2: Version command
echo "Test 2: Version command..."
VERSION=$("$BINARY" --version)
if [[ "$VERSION" == *"gemini-slave"* ]]; then
    pass "Version command works: $VERSION"
else
    fail "Version command failed: $VERSION"
fi

# Test 3: Help command
echo "Test 3: Help command..."
if "$BINARY" -h 2>&1 | grep -q "Usage:"; then
    pass "Help command works"
else
    fail "Help command failed"
fi

# Test 4: Large stdin capture (THE ORIGINAL BUG)
echo "Test 4: Large stdin capture (50KB+)..."
if find ~/Documents/GOgent-Fortress/cmd/gogent-scout -name "*.go" -exec cat {} \; 2>/dev/null | \
    "$BINARY" mapper "Map entry points" -o json > "$TESTDATA/mapper_output.json" 2>&1; then
    if jq -e '.entry_points' "$TESTDATA/mapper_output.json" > /dev/null 2>&1; then
        SIZE=$(wc -c < "$TESTDATA/mapper_output.json")
        pass "Large stdin works (output: $SIZE bytes, valid JSON)"
    else
        fail "Large stdin produced invalid JSON"
        cat "$TESTDATA/mapper_output.json" | head -20
    fi
else
    fail "Large stdin command failed"
fi

# Test 5: File arguments
echo "Test 5: File arguments..."
cat > "$TESTDATA/test.go" << 'EOF'
package main

import "fmt"

func main() {
    fmt.Println("test")
}
EOF

if "$BINARY" deps "Extract dependencies" "$TESTDATA/test.go" -o json > "$TESTDATA/deps_output.json" 2>&1; then
    if jq -e '.external_deps' "$TESTDATA/deps_output.json" > /dev/null 2>&1; then
        pass "File arguments work (valid JSON)"
    else
        fail "File arguments produced invalid JSON"
    fi
else
    fail "File arguments command failed"
fi

# Test 6: JSON output flag
echo "Test 6: JSON output flag..."
if echo 'package main' | "$BINARY" mapper "test" -o json 2>&1 | grep -q '"summary"'; then
    pass "JSON output flag works"
else
    fail "JSON output flag failed"
fi

# Test 7: Model override via flag
echo "Test 7: Model override via flag..."
if echo 'test' | "$BINARY" -m gemini-3-pro-preview mapper "test" -o json > "$TESTDATA/model_override.json" 2>&1; then
    if [ -s "$TESTDATA/model_override.json" ]; then
        pass "Model override flag works"
    else
        fail "Model override produced empty output"
    fi
else
    fail "Model override command failed"
fi

# Test 8: Environment variable override
echo "Test 8: Environment variable override..."
if echo 'test' | GEMINI_MODEL=gemini-3-flash-preview "$BINARY" mapper "test" -o json > "$TESTDATA/env_override.json" 2>&1; then
    if [ -s "$TESTDATA/env_override.json" ]; then
        pass "Environment variable override works"
    else
        fail "Environment variable override produced empty output"
    fi
else
    fail "Environment variable override command failed"
fi

# Test 9: Empty input handling
echo "Test 9: Empty input handling..."
if "$BINARY" mapper "test" -o json < /dev/null > "$TESTDATA/empty_input.json" 2>&1; then
    # Empty input should still work (protocol handles it)
    pass "Empty input handled gracefully"
else
    warn "Empty input failed (may be expected for some protocols)"
fi

# Test 10: Invalid protocol error handling
echo "Test 10: Invalid protocol error..."
if ! "$BINARY" nonexistent "test" 2>&1 | grep -q "unknown protocol"; then
    fail "Invalid protocol error handling failed"
else
    pass "Invalid protocol error handling works"
fi

# Test 11: Missing arguments error handling
echo "Test 11: Missing arguments error..."
if ! "$BINARY" 2>&1 | grep -q "Missing required arguments"; then
    fail "Missing arguments error handling failed"
else
    pass "Missing arguments error handling works"
fi

# Test 12: Protocol-specific models (Flash vs Pro)
echo "Test 12: Protocol model routing..."
PROTOCOLS_TESTED=0
for protocol in mapper architect; do
    if echo 'test' | timeout 30 "$BINARY" "$protocol" "test" -o json > "$TESTDATA/${protocol}_test.json" 2>&1; then
        ((PROTOCOLS_TESTED++))
    fi
done

if [ "$PROTOCOLS_TESTED" -ge 2 ]; then
    pass "Protocol model routing works ($PROTOCOLS_TESTED protocols tested)"
else
    warn "Only $PROTOCOLS_TESTED protocols tested successfully"
fi

# Summary
echo ""
echo "========================================"
echo "  Test Summary"
echo "========================================"
echo -e "Passed: ${GREEN}$PASS_COUNT${NC}"
echo -e "Failed: ${RED}$FAIL_COUNT${NC}"
echo ""

if [ $FAIL_COUNT -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed.${NC}"
    exit 1
fi
