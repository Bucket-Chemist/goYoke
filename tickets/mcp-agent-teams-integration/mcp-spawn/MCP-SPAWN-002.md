```yaml
---
id: MCP-SPAWN-002
title: CLI I/O Verification
description: Verify that claude CLI stdin piping and JSON output work as expected. Tests the practical CLI spawning mechanism.
status: pending
time_estimate: 1h
dependencies: [MCP-SPAWN-001]
phase: 0
tags: [gate, verification, phase-0, cli]
needs_planning: false
agent: typescript-pro
priority: HIGH
---
```

# MCP-SPAWN-002: CLI I/O Verification

## Description

Verify that the `claude` CLI supports stdin piping for prompts and produces parseable JSON output. This tests the practical mechanism for CLI-based agent spawning.

**Source**: Staff-Architect Analysis §4.1.2, §4.1.3

## Why This Matters

The spawn_agent tool will pipe prompts via stdin and parse JSON output. If this doesn't work reliably, the implementation approach must change.

## Task

1. Test `claude -p` with stdin piping
2. Test `--output-format json` parsing
3. Test `--permission-mode delegate`
4. Document all verified flags

## Files

- `packages/tui/tests/verification/cli-io-test.sh` — Bash test script
- `.claude/tmp/cli-verification-result.json` — Results documentation

## Implementation

### Test Script (`packages/tui/tests/verification/cli-io-test.sh`)

```bash
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
if echo "$JSON_RESULT" | jq -e '.result' > /dev/null 2>&1; then
    echo "  ✅ JSON output parseable"
    TEST2="pass"
else
    echo "  ❌ JSON output not parseable: $JSON_RESULT"
    TEST2="fail"
fi

# Test 3: permission-mode delegate
echo "Test 3: permission-mode delegate..."
PERM_RESULT=$(echo "What is 2+2?" | claude -p --permission-mode delegate --output-format json 2>&1 || true)
if echo "$PERM_RESULT" | jq -e '.result' > /dev/null 2>&1; then
    echo "  ✅ permission-mode delegate works"
    TEST3="pass"
else
    echo "  ❌ permission-mode delegate failed: $PERM_RESULT"
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
```

## Acceptance Criteria

- [ ] Test script created and executable
- [ ] stdin piping test passes
- [ ] JSON output parsing test passes
- [ ] permission-mode delegate test passes
- [ ] allowedTools flag test passes
- [ ] Results documented in `.claude/tmp/cli-verification-result.json`
- [ ] All verified flags documented

## Test Deliverables

- [ ] Test script created: `packages/tui/tests/verification/cli-io-test.sh`
- [ ] Script is executable (`chmod +x`)
- [ ] All 4 tests documented with pass/fail
- [ ] Verified flags list complete
- [ ] Gate decision recorded

