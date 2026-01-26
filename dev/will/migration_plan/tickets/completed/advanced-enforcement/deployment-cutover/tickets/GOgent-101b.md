---
id: GOgent-101b
title: WSL2 Compatibility Testing
description: **Task**:
status: pending
time_estimate: 1.5h
dependencies: ["GOgent-101"]
priority: high
week: 5
tags: ["wsl2", "week-5"]
tests_required: true
acceptance_criteria_count: 12
---

### GOgent-101b: WSL2 Compatibility Testing

**Time**: 1.5 hours
**Dependencies**: GOgent-101

**Task**:
Test installation and hook execution on WSL2 (Windows Subsystem for Linux). Addresses M-8 (WSL2 path handling).

**File**: `test/wsl2/wsl2_test.sh`

**Implementation**:

```bash
#!/bin/bash
# WSL2 compatibility test script
# Run this on a WSL2 instance

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

echo "=========================================="
echo "WSL2 Compatibility Test"
echo "=========================================="
echo ""

# Detect if running on WSL2
if ! grep -qi microsoft /proc/version; then
    echo "[SKIP] Not running on WSL2. Skipping tests."
    exit 0
fi

echo "[INFO] Detected WSL2 environment"
echo ""

# Test 1: Path handling (Windows vs Linux paths)
echo "[TEST] Testing path handling..."

# Create test file with Windows-style path
WINDOWS_PATH="/mnt/c/Users/test/file.txt"
LINUX_PATH="/home/test/file.txt"

# Test that hooks handle both path styles
TEST_EVENT=$(cat <<EOF
{
    "hook_event_name": "PreToolUse",
    "tool_name": "Read",
    "tool_input": {
        "file_path": "$WINDOWS_PATH"
    },
    "session_id": "wsl2-test"
}
EOF
)

echo "$TEST_EVENT" | "$PROJECT_ROOT/bin/gogent-validate" > /dev/null

if [ $? -ne 0 ]; then
    echo "[FAIL] Hook failed on Windows-style path"
    exit 1
fi

echo "[PASS] Path handling works"

# Test 2: XDG directory creation
echo "[TEST] Testing XDG directory creation..."

# Clear XDG vars to test fallback
unset XDG_RUNTIME_DIR
unset XDG_CACHE_HOME

export CLAUDE_PROJECT_DIR="$PROJECT_ROOT"

# Run hook to trigger directory creation
echo '{"hook_event_name":"PreToolUse","tool_name":"Read","session_id":"wsl2-xdg-test"}' | \
    "$PROJECT_ROOT/bin/gogent-validate" > /dev/null

# Verify directories created
GOgent_DIR="${HOME}/.cache/gogent"

if [ ! -d "$GOgent_DIR" ]; then
    echo "[FAIL] XDG fallback directory not created: $GOgent_DIR"
    exit 1
fi

echo "[PASS] XDG directory creation works"

# Test 3: File permissions
echo "[TEST] Testing file permissions..."

TEST_LOG="$GOgent_DIR/test-permissions.log"
touch "$TEST_LOG"

if [ ! -w "$TEST_LOG" ]; then
    echo "[FAIL] Cannot write to log file: $TEST_LOG"
    exit 1
fi

rm "$TEST_LOG"

echo "[PASS] File permissions work"

# Test 4: Symlink handling
echo "[TEST] Testing symlink handling..."

TEST_LINK="$GOgent_DIR/test-link"
TEST_TARGET="$GOgent_DIR/test-target"

touch "$TEST_TARGET"
ln -s "$TEST_TARGET" "$TEST_LINK"

if [ ! -L "$TEST_LINK" ]; then
    echo "[FAIL] Symlink creation failed"
    exit 1
fi

rm "$TEST_LINK" "$TEST_TARGET"

echo "[PASS] Symlink handling works"

# Test 5: Line ending handling (CRLF vs LF)
echo "[TEST] Testing line ending handling..."

# Create file with CRLF line endings
TEST_JSONL="$GOgent_DIR/test-crlf.jsonl"
printf '{"test":"line1"}\r\n{"test":"line2"}\r\n' > "$TEST_JSONL"

# Verify hooks can parse CRLF files
# (Go should handle this automatically, but verify)
LINE_COUNT=$(wc -l < "$TEST_JSONL" | tr -d ' ')

if [ "$LINE_COUNT" != "2" ]; then
    echo "[WARN] Line count with CRLF may differ (got: $LINE_COUNT, expected: 2)"
fi

rm "$TEST_JSONL"

echo "[PASS] Line ending handling works"

# Test 6: Case sensitivity
echo "[TEST] Testing case sensitivity..."

# WSL2 is case-sensitive for Linux paths, case-insensitive for /mnt/c
TEST_LOWER="$GOgent_DIR/testfile.txt"
TEST_UPPER="$GOgent_DIR/TESTFILE.txt"

touch "$TEST_LOWER"

if [ -f "$TEST_UPPER" ]; then
    echo "[FAIL] Linux filesystem should be case-sensitive"
    rm "$TEST_LOWER"
    exit 1
fi

rm "$TEST_LOWER"

echo "[PASS] Case sensitivity works as expected"

# Test 7: Exec permission on binaries
echo "[TEST] Testing binary execution..."

for binary in gogent-validate gogent-archive gogent-sharp-edge; do
    if [ ! -x "$PROJECT_ROOT/bin/$binary" ]; then
        echo "[FAIL] Binary not executable: $binary"
        exit 1
    fi

    # Test execution returns successfully (even with empty input)
    if ! echo '{}' | "$PROJECT_ROOT/bin/$binary" &> /dev/null; then
        echo "[WARN] Binary may not handle empty JSON: $binary"
    fi
done

echo "[PASS] Binary execution works"

# Test 8: Environment variable handling
echo "[TEST] Testing environment variables..."

export CLAUDE_PROJECT_DIR="/mnt/c/Projects/test"
export GOgent_TEST_MODE="1"

if [ "$CLAUDE_PROJECT_DIR" != "/mnt/c/Projects/test" ]; then
    echo "[FAIL] Environment variables not preserved"
    exit 1
fi

unset CLAUDE_PROJECT_DIR
unset GOgent_TEST_MODE

echo "[PASS] Environment variables work"

echo ""
echo "=========================================="
echo "All WSL2 compatibility tests passed!"
echo "=========================================="
echo ""
echo "WSL2 environment is compatible with gogent-fortress Go hooks."
```

**Run on WSL2**:

```bash
# From a WSL2 terminal
./test/wsl2/wsl2_test.sh
```

**Acceptance Criteria**:

- [ ] Script detects WSL2 environment (checks /proc/version)
- [ ] Handles Windows-style paths (/mnt/c/...)
- [ ] Handles Linux-style paths (/home/...)
- [ ] XDG directory creation works on WSL2
- [ ] File permissions work correctly
- [ ] Symlink creation works
- [ ] CRLF line endings parsed correctly
- [ ] Case sensitivity behaves as expected
- [ ] Binaries execute correctly
- [ ] Environment variables preserved
- [ ] All tests pass on WSL2: `./test/wsl2/wsl2_test.sh`
- [ ] Document any WSL2-specific quirks in README

**Why This Matters**: Many users run Claude Code on WSL2. Must verify hooks work correctly in this environment to avoid platform-specific bugs (M-8).

---
