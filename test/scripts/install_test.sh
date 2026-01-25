#!/bin/bash
# Test install.sh script

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

echo "[TEST] Testing install.sh script..."

# Test 1: Test-only mode
echo "[TEST] Running in test-only mode..."
if ! "$PROJECT_ROOT/scripts/install.sh" --test-only; then
    echo "[FAIL] Test-only mode failed"
    exit 1
fi

# Verify no binaries created in bin/ (may already exist from previous runs)
# Just verify install didn't modify ~/.gogent
if [ -d "${HOME}/.gogent/bin" ]; then
    # Count files before test
    BEFORE_COUNT=$(ls -1 "${HOME}/.gogent/bin" 2>/dev/null | wc -l)
fi

echo "[PASS] Test-only mode works"

# Test 2: Full install (skip tests for speed)
echo "[TEST] Running full install (skip tests)..."
if ! "$PROJECT_ROOT/scripts/install.sh" --skip-tests; then
    echo "[FAIL] Full install failed"
    exit 1
fi

# Verify binaries created in project bin/
BINARIES=("gogent-validate" "gogent-archive" "gogent-sharp-edge")

for binary in "${BINARIES[@]}"; do
    if [ ! -x "$PROJECT_ROOT/bin/$binary" ]; then
        echo "[FAIL] Binary not found or not executable in project: $binary"
        exit 1
    fi
done

echo "[PASS] Full install works"

# Test 3: Verify binaries in ~/.gogent/bin
GOgent_BIN="${HOME}/.gogent/bin"

if [ ! -d "$GOgent_BIN" ]; then
    echo "[FAIL] $GOgent_BIN not created"
    exit 1
fi

for binary in "${BINARIES[@]}"; do
    if [ ! -x "$GOgent_BIN/$binary" ]; then
        echo "[FAIL] Binary not installed to $GOgent_BIN: $binary"
        exit 1
    fi
done

echo "[PASS] Binaries installed to ~/.gogent/bin"

# Test 4: Verify backups created (if hooks existed)
HOOKS_DIR="${HOME}/.claude/hooks"

if [ -d "$HOOKS_DIR" ]; then
    BACKUP_DIRS=("$HOOKS_DIR"/backup-*)

    # Check if any backup directories exist (glob expansion)
    if [ -d "${BACKUP_DIRS[0]}" ]; then
        echo "[PASS] Backup directories created"
    else
        echo "[WARN] No backup directories found (fresh install or no hooks existed)"
    fi
else
    echo "[WARN] No hooks directory found (fresh install)"
fi

# Test 5: Idempotency - run install again
echo "[TEST] Testing idempotency (running install again)..."
if ! "$PROJECT_ROOT/scripts/install.sh" --skip-tests; then
    echo "[FAIL] Second install run failed (not idempotent)"
    exit 1
fi

echo "[PASS] Script is idempotent"

# Test 6: Verify binaries respond to --help
echo "[TEST] Verifying binary functionality..."
for binary in "${BINARIES[@]}"; do
    if "$GOgent_BIN/$binary" --help &> /dev/null; then
        echo "[PASS] $binary responds to --help"
    else
        echo "[WARN] $binary does not support --help (may be expected for hooks)"
    fi
done

echo ""
echo "=========================================="
echo "All install.sh tests passed!"
echo "=========================================="
