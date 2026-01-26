---
id: GOgent-105
title: Rollback Script and Testing
description: **Task**:
status: pending
time_estimate: 1h
dependencies: ["GOgent-104"]
priority: high
week: 5
tags: ["cutover", "week-5"]
tests_required: true
acceptance_criteria_count: 11
---

### GOgent-105: Rollback Script and Testing

**Time**: 1 hour
**Dependencies**: GOgent-104

**Task**:
Create rollback script that restores Bash hooks in <5 minutes. Test rollback process.

**File**: `scripts/rollback.sh`

**Implementation**:

```bash
#!/bin/bash
# Rollback script - restore Bash hooks
# Usage: ./scripts/rollback.sh [--dry-run] [--backup-dir DIR]

set -e

HOOKS_DIR="${HOME}/.claude/hooks"
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

DRY_RUN=false
BACKUP_DIR=""

for arg in "$@"; do
    case $arg in
        --dry-run)
            DRY_RUN=true
            ;;
        --backup-dir)
            shift
            BACKUP_DIR="$1"
            shift
            ;;
    esac
done

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

if [ "$DRY_RUN" = true ]; then
    log_warn "DRY RUN MODE - No changes will be made"
fi

# Find most recent backup if not specified
if [ -z "$BACKUP_DIR" ]; then
    BACKUP_DIR=$(ls -dt "$HOOKS_DIR"/backup-* 2>/dev/null | head -n 1)

    if [ -z "$BACKUP_DIR" ]; then
        log_error "No backup found in $HOOKS_DIR/backup-*"
        log_error "Specify backup with --backup-dir"
        exit 1
    fi

    log_info "Using most recent backup: $BACKUP_DIR"
fi

# Verify backup exists
if [ ! -d "$BACKUP_DIR" ]; then
    log_error "Backup directory not found: $BACKUP_DIR"
    exit 1
fi

# Verify backup contains hooks
HOOKS=(
    "validate-routing"
    "session-archive"
    "sharp-edge-detector"
)

for hook in "${HOOKS[@]}"; do
    if [ ! -e "$BACKUP_DIR/$hook" ]; then
        log_warn "Hook not found in backup: $hook"
    fi
done

log_info "Backup verified: $BACKUP_DIR"

# Rollback hooks
log_info "Rolling back hooks..."

for hook in "${HOOKS[@]}"; do
    HOOK_PATH="$HOOKS_DIR/$hook"
    BACKUP_PATH="$BACKUP_DIR/$hook"

    if [ ! -e "$BACKUP_PATH" ]; then
        log_warn "  Skipping $hook (not in backup)"
        continue
    fi

    log_info "  Restoring: $hook"

    if [ "$DRY_RUN" = false ]; then
        # Remove current hook (Go symlink)
        rm -f "$HOOK_PATH"

        # Restore from backup
        cp -P "$BACKUP_PATH" "$HOOK_PATH"

        # Ensure executable
        chmod +x "$HOOK_PATH"

        log_info "    ✓ Restored from backup"
    else
        log_info "    [DRY RUN] Would restore from backup"
    fi
done

# Verify rollback
log_info "Verifying rollback..."

for hook in "${HOOKS[@]}"; do
    HOOK_PATH="$HOOKS_DIR/$hook"

    if [ "$DRY_RUN" = false ]; then
        if [ ! -e "$HOOK_PATH" ]; then
            log_error "  ✗ $hook not found after rollback"
            exit 1
        fi

        # Check if it's Bash script (not Go binary symlink)
        if file "$HOOK_PATH" | grep -q "Bourne-Again shell script"; then
            log_info "  ✓ $hook is Bash script"
        elif [ -L "$HOOK_PATH" ]; then
            TARGET=$(readlink "$HOOK_PATH")
            if echo "$TARGET" | grep -q "\.sh$"; then
                log_info "  ✓ $hook symlinks to Bash script"
            else
                log_warn "  ⚠️  $hook still points to Go binary: $TARGET"
            fi
        else
            log_warn "  ⚠️  $hook type unclear (check manually)"
        fi
    fi
done

# Test hooks
log_info "Testing rolled-back hooks..."

TEST_EVENT='{"hook_event_name":"PreToolUse","tool_name":"Read"}'

for hook in "${HOOKS[@]}"; do
    HOOK_PATH="$HOOKS_DIR/$hook"

    if [ "$DRY_RUN" = false ] && [ -x "$HOOK_PATH" ]; then
        if echo "$TEST_EVENT" | "$HOOK_PATH" > /dev/null 2>&1; then
            log_info "  ✓ $hook responds correctly"
        else
            log_warn "  ⚠️  $hook may have issues"
        fi
    fi
done

log_info ""
log_info "=========================================="
log_info "Rollback Complete!"
log_info "=========================================="
log_info ""

if [ "$DRY_RUN" = false ]; then
    log_info "Bash hooks restored from: $BACKUP_DIR"
    log_info ""
    log_info "Hooks now running Bash implementation:"
    ls -la "$HOOKS_DIR" | grep -E "validate-routing|session-archive|sharp-edge-detector"

    log_info ""
    log_info "Next steps:"
    log_info "  1. Verify hooks work in new Claude Code session"
    log_info "  2. Document rollback reason"
    log_info "  3. Fix Go implementation issues"
    log_info "  4. Re-test before next cutover attempt"
else
    log_info "Dry run complete. No changes made."
    log_info "Run without --dry-run to execute rollback."
fi
```

**Rollback Test Script**: `test/scripts/rollback_test.sh`

```bash
#!/bin/bash
# Test rollback.sh script

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
HOOKS_DIR="${HOME}/.claude/hooks"

echo "[TEST] Testing rollback.sh script..."

# Setup: Create fake backup
FAKE_BACKUP="$HOOKS_DIR/backup-test-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$FAKE_BACKUP"

# Create fake Bash hooks in backup
for hook in validate-routing session-archive sharp-edge-detector; do
    echo "#!/bin/bash" > "$FAKE_BACKUP/$hook"
    echo "echo 'Fake Bash hook: $hook'" >> "$FAKE_BACKUP/$hook"
    chmod +x "$FAKE_BACKUP/$hook"
done

echo "[TEST] Created fake backup: $FAKE_BACKUP"

# Test 1: Dry run
echo "[TEST] Testing --dry-run..."
if ! "$PROJECT_ROOT/scripts/rollback.sh" --dry-run --backup-dir "$FAKE_BACKUP"; then
    echo "[FAIL] Dry run failed"
    exit 1
fi

echo "[PASS] Dry run works"

# Test 2: Actual rollback
echo "[TEST] Testing actual rollback..."

# First, backup current hooks
CURRENT_BACKUP="$HOOKS_DIR/backup-before-rollback-test"
mkdir -p "$CURRENT_BACKUP"

for hook in validate-routing session-archive sharp-edge-detector; do
    if [ -e "$HOOKS_DIR/$hook" ]; then
        cp -P "$HOOKS_DIR/$hook" "$CURRENT_BACKUP/$hook"
    fi
done

# Execute rollback
if ! "$PROJECT_ROOT/scripts/rollback.sh" --backup-dir "$FAKE_BACKUP"; then
    echo "[FAIL] Rollback failed"
    exit 1
fi

# Verify hooks restored
for hook in validate-routing session-archive sharp-edge-detector; do
    if [ ! -f "$HOOKS_DIR/$hook" ]; then
        echo "[FAIL] Hook not restored: $hook"
        exit 1
    fi

    # Check it's the fake one
    if ! grep -q "Fake Bash hook" "$HOOKS_DIR/$hook"; then
        echo "[FAIL] Hook not from backup: $hook"
        exit 1
    fi
done

echo "[PASS] Rollback works"

# Cleanup: Restore original hooks
for hook in validate-routing session-archive sharp-edge-detector; do
    if [ -e "$CURRENT_BACKUP/$hook" ]; then
        cp -P "$CURRENT_BACKUP/$hook" "$HOOKS_DIR/$hook"
    fi
done

rm -rf "$FAKE_BACKUP" "$CURRENT_BACKUP"

echo ""
echo "=========================================="
echo "All rollback tests passed!"
echo "=========================================="
```

**Acceptance Criteria**:

- [ ] Script finds most recent backup automatically
- [ ] Accepts `--backup-dir` to specify backup location
- [ ] Verifies backup exists and contains hooks
- [ ] Removes Go symlinks
- [ ] Restores Bash hooks from backup
- [ ] Ensures restored hooks are executable
- [ ] Tests hooks respond to sample events
- [ ] `--dry-run` flag works
- [ ] Rollback completes in <5 minutes
- [ ] Test script validates rollback functionality
- [ ] Rollback tested successfully before cutover

**Why This Matters**: Rollback must work reliably under pressure. Testing rollback before cutover ensures it's ready if needed.

---
