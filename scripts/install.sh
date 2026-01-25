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
    # Test actual cmd packages, not pkg/ (which doesn't exist)
    if ! go test ./cmd/... -v -timeout 3m; then
        log_error "Unit tests failed. Fix errors before installing."
        exit 1
    fi

    log_info "Running integration tests..."
    if ! go test ./test/integration/... -v -timeout 3m; then
        log_error "Integration tests failed. Fix errors before installing."
        exit 1
    fi

    log_info "Running benchmarks..."
    if ! go test -bench=. ./test/benchmark -benchtime=5s -timeout 3m; then
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

# Create bin directory if it doesn't exist
mkdir -p "$PROJECT_ROOT/bin"

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

    # Test binary responds to --help (safer than invalid JSON)
    if ! "$BINARY_PATH" --help &> /dev/null; then
        log_warn "Binary $BINARY_NAME does not support --help flag (expected for hooks)"
    else
        log_info "  ✓ $BINARY_NAME responds to --help"
    fi
done

log_info "✅ All binaries verified"

# Step 6: Create backup of existing hooks (only if they exist)
HOOKS=(
    "validate-routing"
    "session-archive"
    "sharp-edge-detector"
)

BACKUP_NEEDED=false
for hook in "${HOOKS[@]}"; do
    if [ -f "$HOOKS_DIR/$hook" ] || [ -L "$HOOKS_DIR/$hook" ]; then
        BACKUP_NEEDED=true
        break
    fi
done

if [ "$BACKUP_NEEDED" = true ]; then
    BACKUP_DIR="$HOOKS_DIR/backup-$(date +%Y%m%d-%H%M%S)"
    log_info "Backing up existing Bash hooks to $BACKUP_DIR..."
    mkdir -p "$BACKUP_DIR"

    for hook in "${HOOKS[@]}"; do
        if [ -f "$HOOKS_DIR/$hook" ] || [ -L "$HOOKS_DIR/$hook" ]; then
            cp -P "$HOOKS_DIR/$hook" "$BACKUP_DIR/$hook"
            log_info "  Backed up: $hook"
        fi
    done
else
    log_info "No existing hooks to backup (fresh install)"
fi

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
if [ "$BACKUP_NEEDED" = true ]; then
    log_info "Bash hooks backed up to:  $BACKUP_DIR/"
fi
log_info ""
log_info "Next steps:"
log_info "  1. Run parallel testing: ./scripts/parallel-test.sh"
log_info "  2. Monitor for 24 hours"
log_info "  3. If stable, run cutover: ./scripts/cutover.sh"
log_info "  4. If issues, rollback: ./scripts/rollback.sh"
log_info ""
log_warn "Do NOT manually symlink hooks yet. Use cutover.sh script."
log_info ""
