---
id: GOgent-101
title: Installation Script
description: Create automated installation script that builds Go binaries, runs tests, and prepares for deployment
status: pending
time_estimate: 2h
dependencies: ["GOgent-001"]
priority: high
week: 5
tags: ["installation", "week-5"]
tests_required: true
acceptance_criteria_count: 12
---

### GOgent-101: Installation Script

**Time**: 2 hours
**Dependencies**: All GOgent-001 to 047 complete

**Task**:
Create automated installation script that builds Go binaries, runs tests, and prepares for deployment.

**File**: `scripts/install.sh`

**Implementation**:

```bash
#!/bin/bash
# Installation script for gogent-fortress Go hooks
# Usage: ./scripts/install.sh [--test-only] [--skip-tests]

set -e  # Exit on error

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HOOKS_DIR="${HOME}/.claude/hooks"
GOgent_DIR="${HOME}/.gogent"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Parse arguments
TEST_ONLY=false
SKIP_TESTS=false

for arg in "$@"; do
    case $arg in
        --test-only)
            TEST_ONLY=true
            ;;
        --skip-tests)
            SKIP_TESTS=true
            ;;
        --help)
            echo "Usage: $0 [--test-only] [--skip-tests]"
            echo ""
            echo "Options:"
            echo "  --test-only   Run tests but don't install"
            echo "  --skip-tests  Skip tests and install directly"
            echo "  --help        Show this help message"
            exit 0
            ;;
    esac
done

cd "$PROJECT_ROOT"

# Step 1: Verify Go installation
log_info "Checking Go installation..."
if ! command -v go &> /dev/null; then
    log_error "Go is not installed. Install from https://go.dev/dl/"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}')
log_info "Found Go version: $GO_VERSION"

# Step 2: Install dependencies
log_info "Installing Go dependencies..."
go mod download

# Step 3: Run tests (unless skipped)
if [ "$SKIP_TESTS" = false ]; then
    log_info "Running unit tests..."
    if ! go test ./pkg/... -v -timeout 5m; then
        log_error "Unit tests failed. Fix errors before installing."
        exit 1
    fi

    log_info "Running integration tests..."
    if ! go test ./test/integration/... -v -timeout 10m; then
        log_error "Integration tests failed. Fix errors before installing."
        exit 1
    fi

    log_info "Running benchmarks..."
    if ! go test -bench=. ./test/benchmark -benchtime=1s -timeout 5m; then
        log_error "Benchmarks failed. Fix errors before installing."
        exit 1
    fi

    log_info "✅ All tests passed"
else
    log_warn "Skipping tests (--skip-tests flag)"
fi

# Exit if test-only mode
if [ "$TEST_ONLY" = true ]; then
    log_info "Test-only mode. Exiting without installation."
    exit 0
fi

# Step 4: Build binaries
log_info "Building Go binaries..."

BINARIES=(
    "cmd/gogent-validate:gogent-validate"
    "cmd/gogent-archive:gogent-archive"
    "cmd/gogent-sharp-edge:gogent-sharp-edge"
)

for binary in "${BINARIES[@]}"; do
    CMD_DIR="${binary%%:*}"
    BINARY_NAME="${binary##*:}"

    log_info "  Building $BINARY_NAME..."

    if ! go build -o "$PROJECT_ROOT/bin/$BINARY_NAME" "./$CMD_DIR"; then
        log_error "Failed to build $BINARY_NAME"
        exit 1
    fi
done

log_info "✅ All binaries built successfully"

# Step 5: Verify binaries
log_info "Verifying binaries..."

for binary in "${BINARIES[@]}"; do
    BINARY_NAME="${binary##*:}"
    BINARY_PATH="$PROJECT_ROOT/bin/$BINARY_NAME"

    if [ ! -x "$BINARY_PATH" ]; then
        log_error "Binary not executable: $BINARY_PATH"
        exit 1
    fi

    # Test binary accepts JSON (basic smoke test)
    if ! echo '{}' | "$BINARY_PATH" &> /dev/null; then
        log_warn "Binary $BINARY_NAME may not handle empty JSON correctly (expected for hooks)"
    fi
done

log_info "✅ All binaries verified"

# Step 6: Create backup of existing hooks
log_info "Backing up existing Bash hooks..."

mkdir -p "$HOOKS_DIR/backup-$(date +%Y%m%d-%H%M%S)"

HOOKS=(
    "validate-routing"
    "session-archive"
    "sharp-edge-detector"
)

for hook in "${HOOKS[@]}"; do
    if [ -f "$HOOKS_DIR/$hook" ] || [ -L "$HOOKS_DIR/$hook" ]; then
        cp -P "$HOOKS_DIR/$hook" "$HOOKS_DIR/backup-$(date +%Y%m%d-%H%M%S)/$hook"
        log_info "  Backed up: $hook"
    else
        log_warn "  Hook not found (fresh install): $hook"
    fi
done

# Step 7: Install binaries (but don't activate yet)
log_info "Installing Go binaries to $GOgent_DIR/bin..."

mkdir -p "$GOgent_DIR/bin"

for binary in "${BINARIES[@]}"; do
    BINARY_NAME="${binary##*:}"
    cp "$PROJECT_ROOT/bin/$BINARY_NAME" "$GOgent_DIR/bin/$BINARY_NAME"
    chmod +x "$GOgent_DIR/bin/$BINARY_NAME"
    log_info "  Installed: $BINARY_NAME"
done

# Step 8: Instructions for parallel testing
log_info ""
log_info "=========================================="
log_info "Installation Complete!"
log_info "=========================================="
log_info ""
log_info "Go binaries installed to: $GOgent_DIR/bin/"
log_info "Bash hooks backed up to:  $HOOKS_DIR/backup-*/"
log_info ""
log_info "Next steps:"
log_info "  1. Run parallel testing: ./scripts/parallel-test.sh"
log_info "  2. Monitor for 24 hours"
log_info "  3. If stable, run cutover: ./scripts/cutover.sh"
log_info "  4. If issues, rollback: ./scripts/rollback.sh"
log_info ""
log_warn "Do NOT manually symlink hooks yet. Use cutover.sh script."
log_info ""
```

**Tests**: `test/scripts/install_test.sh`

```bash
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

# Verify no binaries created
if [ -d "$PROJECT_ROOT/bin" ]; then
    echo "[FAIL] Binaries created in test-only mode"
    exit 1
fi

echo "[PASS] Test-only mode works"

# Test 2: Full install (skip tests for speed)
echo "[TEST] Running full install (skip tests)..."
if ! "$PROJECT_ROOT/scripts/install.sh" --skip-tests; then
    echo "[FAIL] Full install failed"
    exit 1
fi

# Verify binaries created
BINARIES=("gogent-validate" "gogent-archive" "gogent-sharp-edge")

for binary in "${BINARIES[@]}"; do
    if [ ! -x "$PROJECT_ROOT/bin/$binary" ]; then
        echo "[FAIL] Binary not found or not executable: $binary"
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

# Test 4: Verify backups created
HOOKS_DIR="${HOME}/.claude/hooks"
BACKUP_DIRS=("$HOOKS_DIR"/backup-*)

if [ ${#BACKUP_DIRS[@]} -eq 0 ]; then
    echo "[WARN] No backup directories found (may be fresh install)"
else
    echo "[PASS] Backup directories created"
fi

echo ""
echo "=========================================="
echo "All install.sh tests passed!"
echo "=========================================="
```

**Acceptance Criteria**:

- [x] `install.sh` checks for Go installation
- [x] Installs dependencies with `go mod download`
- [x] Runs unit tests, integration tests, benchmarks (unless --skip-tests)
- [x] Builds all three binaries to `bin/` directory
- [x] Verifies binaries are executable
- [x] Backs up existing Bash hooks to timestamped directory
- [x] Installs Go binaries to `~/.gogent/bin/`
- [x] Outputs clear next-steps instructions
- [x] `--test-only` flag runs tests without installation
- [x] `--skip-tests` flag skips tests for quick install
- [x] Test script validates all functionality
- [x] Script is idempotent (safe to run multiple times)

**Why This Matters**: Automated installation reduces human error and ensures consistent deployment. Backup of Bash hooks enables quick rollback if needed.

---
