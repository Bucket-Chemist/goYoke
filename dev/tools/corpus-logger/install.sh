#!/usr/bin/env bash
# Installation script for corpus-logger hook

set -euo pipefail

HOOK_DIR="${HOME}/.claude/hooks"
HOOK_NAME="zzz-corpus-logger"
BINARY="corpus-logger"

# Check if binary exists
if [[ ! -f "${BINARY}" ]]; then
    echo "Error: ${BINARY} not found. Run 'go build -o corpus-logger main.go' first."
    exit 1
fi

# Create hooks directory if it doesn't exist
mkdir -p "${HOOK_DIR}"

# Copy binary to hooks directory
cp "${BINARY}" "${HOOK_DIR}/${HOOK_NAME}"
chmod +x "${HOOK_DIR}/${HOOK_NAME}"

echo "✓ Installed ${HOOK_NAME} to ${HOOK_DIR}/"
echo ""
echo "Output location:"
if [[ -n "${XDG_RUNTIME_DIR:-}" ]]; then
    echo "  ${XDG_RUNTIME_DIR}/gogent/event-corpus-raw.jsonl"
elif [[ -n "${XDG_CACHE_HOME:-}" ]]; then
    echo "  ${XDG_CACHE_HOME}/gogent/event-corpus-raw.jsonl"
else
    echo "  ${HOME}/.cache/gogent/event-corpus-raw.jsonl"
fi
echo ""
echo "The hook will capture all Claude Code events to this file."
