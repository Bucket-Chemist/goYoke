#!/bin/bash
# Session Archive Hook with Go/Bash Fallback
# Purpose: Provides production safety net during migration period
# Usage: Configured as SessionEnd hook in claude-code config
set -euo pipefail

# Read STDIN into temp file (allows retry)
TMPFILE=$(mktemp)
trap 'rm -f "$TMPFILE"' EXIT
cat > "$TMPFILE"

# Ensure log directory exists
mkdir -p ~/.gogent

# Try Go implementation first
if command -v gogent-archive &>/dev/null; then
    if gogent-archive < "$TMPFILE" 2>/dev/null; then
        # Success
        echo "[INFO] Go hook succeeded" >&2
        exit 0
    else
        EXIT_CODE=$?
        echo "[ERROR] Go hook failed with exit code $EXIT_CODE" >&2
        echo "$(date): gogent-archive failed (exit $EXIT_CODE)" >> ~/.gogent/hook-failures.log
    fi
else
    echo "[WARN] gogent-archive not found in PATH" >&2
fi

# Fallback to bash hook
echo "[INFO] Falling back to bash hook" >&2
if [[ -f ~/.claude/hooks/session-archive.sh ]]; then
    ~/.claude/hooks/session-archive.sh < "$TMPFILE"
    echo "$(date): Bash fallback succeeded" >> ~/.gogent/hook-failures.log
    exit 0
else
    echo "[ERROR] Bash hook not found at ~/.claude/hooks/session-archive.sh" >&2
    exit 1
fi
