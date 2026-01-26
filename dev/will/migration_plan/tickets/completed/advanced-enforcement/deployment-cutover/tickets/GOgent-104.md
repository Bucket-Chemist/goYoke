---
id: GOgent-104
title: Symlink Cutover Script
description: **Task**:
status: pending
time_estimate: 1h
dependencies: ["GOgent-103"]
priority: high
week: 5
tags: ["cutover", "week-5"]
tests_required: true
acceptance_criteria_count: 9
---

### GOgent-104: Symlink Cutover Script

**Time**: 1 hour
**Dependencies**: GOgent-103 (decision made)

**Task**:
Create script that atomically switches symlinks from Bash to Go hooks.

**File**: `scripts/cutover.sh`

**Implementation**:

```bash
#!/bin/bash
# Cutover script - switch from Bash to Go hooks
# Usage: ./scripts/cutover.sh [--dry-run]

set -e

HOOKS_DIR="${HOME}/.claude/hooks"
GOgent_BIN="${HOME}/.gogent/bin"
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

DRY_RUN=false

for arg in "$@"; do
    case $arg in
        --dry-run)
            DRY_RUN=true
            ;;
    esac
done

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

if [ "$DRY_RUN" = true ]; then
    log_warn "DRY RUN MODE - No changes will be made"
fi

# Verify Go binaries exist
BINARIES=(
    "gogent-validate"
    "gogent-archive"
    "gogent-sharp-edge"
)

for binary in "${BINARIES[@]}"; do
    if [ ! -x "$GOgent_BIN/$binary" ]; then
        echo "ERROR: Binary not found: $GOgent_BIN/$binary"
        echo "Run ./scripts/install.sh first"
        exit 1
    fi
done

log_info "All Go binaries verified"

# Create final backup
BACKUP_DIR="$HOOKS_DIR/backup-cutover-$(date +%Y%m%d-%H%M%S)"

if [ "$DRY_RUN" = false ]; then
    mkdir -p "$BACKUP_DIR"
fi

# Cutover hooks
HOOKS=(
    "validate-routing:gogent-validate"
    "session-archive:gogent-archive"
    "sharp-edge-detector:gogent-sharp-edge"
)

for hook in "${HOOKS[@]}"; do
    HOOK_NAME="${hook%%:*}"
    BINARY_NAME="${hook##*:}"

    HOOK_PATH="$HOOKS_DIR/$HOOK_NAME"
    BINARY_PATH="$GOgent_BIN/$BINARY_NAME"

    log_info "Cutting over: $HOOK_NAME → $BINARY_NAME"

    if [ "$DRY_RUN" = false ]; then
        # Backup existing hook
        if [ -e "$HOOK_PATH" ]; then
            cp -P "$HOOK_PATH" "$BACKUP_DIR/$HOOK_NAME"
            log_info "  Backed up: $HOOK_NAME"
        fi

        # Remove old symlink/file
        rm -f "$HOOK_PATH"

        # Create new symlink
        ln -s "$BINARY_PATH" "$HOOK_PATH"

        log_info "  Linked: $HOOK_NAME → $BINARY_PATH"
    else
        log_info "  [DRY RUN] Would link: $HOOK_NAME → $BINARY_PATH"
    fi
done

# Verify symlinks
log_info "Verifying symlinks..."

for hook in "${HOOKS[@]}"; do
    HOOK_NAME="${hook%%:*}"
    BINARY_NAME="${hook##*:}"

    HOOK_PATH="$HOOKS_DIR/$HOOK_NAME"
    BINARY_PATH="$GOgent_BIN/$BINARY_NAME"

    if [ "$DRY_RUN" = false ]; then
        if [ ! -L "$HOOK_PATH" ]; then
            echo "ERROR: $HOOK_NAME is not a symlink"
            exit 1
        fi

        LINK_TARGET=$(readlink "$HOOK_PATH")

        if [ "$LINK_TARGET" != "$BINARY_PATH" ]; then
            echo "ERROR: $HOOK_NAME points to wrong target: $LINK_TARGET"
            exit 1
        fi

        log_info "  ✓ $HOOK_NAME → $BINARY_PATH"
    fi
done

# Test hooks
log_info "Testing hooks..."

TEST_EVENT='{"hook_event_name":"PreToolUse","tool_name":"Read","tool_input":{"file_path":"/tmp/test"},"session_id":"cutover-test"}'

for hook in "${HOOKS[@]}"; do
    HOOK_NAME="${hook%%:*}"
    HOOK_PATH="$HOOKS_DIR/$HOOK_NAME"

    if [ "$DRY_RUN" = false ]; then
        # Test hook accepts JSON
        if ! echo "$TEST_EVENT" | "$HOOK_PATH" > /dev/null 2>&1; then
            log_warn "  ⚠️  $HOOK_NAME may not handle test event (check logs)"
        else
            log_info "  ✓ $HOOK_NAME responds to test event"
        fi
    fi
done

log_info ""
log_info "=========================================="
log_info "Cutover Complete!"
log_info "=========================================="
log_info ""

if [ "$DRY_RUN" = false ]; then
    log_info "Hooks now point to Go binaries:"
    ls -la "$HOOKS_DIR" | grep -E "validate-routing|session-archive|sharp-edge-detector"

    log_info ""
    log_info "Next steps:"
    log_info "  1. Start a new Claude Code session"
    log_info "  2. Monitor logs: tail -f ~/.gogent/hooks.log"
    log_info "  3. If issues: ./scripts/rollback.sh"
    log_info ""
    log_info "Backup location: $BACKUP_DIR"
else
    log_info "Dry run complete. No changes made."
    log_info "Run without --dry-run to execute cutover."
fi
````

**Acceptance Criteria**:

- [ ] Script verifies all Go binaries exist before cutover
- [ ] Creates timestamped backup before making changes
- [ ] Atomically updates all symlinks (remove old, create new)
- [ ] Verifies symlinks point to correct targets
- [ ] Tests hooks respond to sample events
- [ ] `--dry-run` flag shows what would happen without making changes
- [ ] Outputs clear success message with next steps
- [ ] Lists symlinks after cutover for verification
- [ ] Idempotent (safe to run multiple times)

**Why This Matters**: Atomic cutover ensures minimal downtime. Verification step catches symlink errors before users encounter them.

---
