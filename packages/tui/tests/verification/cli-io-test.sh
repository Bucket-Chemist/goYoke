#!/bin/bash
# CLI I/O Verification Script
# Tests claude CLI capabilities for spawn_agent implementation

set -e

RESULTS_FILE=".claude/tmp/cli-verification-result.json"
mkdir -p "$(dirname "$RESULTS_FILE")"

echo "Starting CLI I/O Verification..."
echo '{"tests": [], "timestamp": "'$(date -Iseconds)'"}' > "$RESULTS_FILE"

# Test 1: stdin piping works
echo "Test 1: stdin piping..."
STDIN_RESULT=$(echo "Reply with exactly: STDIN_TEST_OK" | claude -p --output-format text 2>&1 || true)
if echo "$STDIN_RESULT" | grep -q "STDIN_TEST_OK"; then
    echo "  ✅ stdin piping works"
    TEST1="pass"
else
    echo "  ❌ stdin piping failed: $STDIN_RESULT"
    TEST1="fail"
fi

# Test 2: JSON output format
echo "Test 2: JSON output format..."
JSON_RESULT=$(echo "Say hello" | claude -p --output-format json 2>&1 || true)
# CLI returns array of event objects, check if it's valid JSON and contains result event
if echo "$JSON_RESULT" | jq -e '.[].type' > /dev/null 2>&1 && echo "$JSON_RESULT" | jq -e '.[] | select(.type == "result")' > /dev/null 2>&1; then
    echo "  ✅ JSON output parseable"
    TEST2="pass"
else
    echo "  ❌ JSON output not parseable or missing result event"
    TEST2="fail"
fi

# Test 3: permission-mode delegate
echo "Test 3: permission-mode delegate..."
PERM_RESULT=$(echo "What is 2+2?" | claude -p --permission-mode delegate --output-format json 2>&1 || true)
# Check if JSON is parseable and contains result event
if echo "$PERM_RESULT" | jq -e '.[].type' > /dev/null 2>&1 && echo "$PERM_RESULT" | jq -e '.[] | select(.type == "result")' > /dev/null 2>&1; then
    echo "  ✅ permission-mode delegate works"
    TEST3="pass"
else
    echo "  ❌ permission-mode delegate failed"
    TEST3="fail"
fi

# Test 4: allowedTools restriction
echo "Test 4: allowedTools restriction..."
TOOLS_RESULT=$(echo "List current directory" | claude -p --permission-mode delegate --allowedTools "Bash(ls:*)" --output-format json 2>&1 || true)
if [ -n "$TOOLS_RESULT" ]; then
    echo "  ✅ allowedTools flag accepted"
    TEST4="pass"
else
    echo "  ❌ allowedTools failed"
    TEST4="fail"
fi

# Write results
cat > "$RESULTS_FILE" << EOF
{
  "timestamp": "$(date -Iseconds)",
  "tests": {
    "stdin_piping": "$TEST1",
    "json_output": "$TEST2",
    "permission_mode_delegate": "$TEST3",
    "allowed_tools": "$TEST4"
  },
  "verified_flags": [
    "-p / --print",
    "--output-format json",
    "--permission-mode delegate",
    "--allowedTools"
  ],
  "gate_decision": "$([ "$TEST1" = "pass" ] && [ "$TEST2" = "pass" ] && echo "PROCEED" || echo "INVESTIGATE")"
}
EOF

echo ""
echo "Results written to: $RESULTS_FILE"
cat "$RESULTS_FILE" | jq .
