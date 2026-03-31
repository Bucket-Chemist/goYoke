#!/usr/bin/env bash
# check-claude-writes.sh — CI check for residual .claude/ write paths in Go code.
# Exits 0 if clean, 1 if .claude/ write paths found in production code.
#
# Legitimate config READS are excluded:
#   .claude/agents/, .claude/routing-schema.json, .claude/conventions/,
#   .claude/rules/, .claude/CLAUDE.md, .claude/settings
#
# Test files (*_test.go) are excluded — they may reference .claude/ for
# backward-compat fixtures or enforcement test data.

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

# Pattern: Go filepath.Join or string literal referencing .claude/ with
# runtime state subdirectories (memory, sessions, current-session, tmp)
RUNTIME_PATTERN='\.claude.*(memory|sessions|current-session|/tmp)'

# Exclude patterns (config reads, not runtime writes)
EXCLUDE='agents/|routing-schema|conventions/|rules/|CLAUDE\.md|settings|\.claude/agents'

matches=$(rg --type go \
    --glob '!*_test.go' \
    --glob '!vendor/**' \
    --glob '!.claude/**' \
    --glob '!.archive/**' \
    --glob '!tickets/**' \
    "$RUNTIME_PATTERN" 2>/dev/null \
    | grep -Ev "$EXCLUDE" || true)

if [[ -n "$matches" ]]; then
    echo "ERROR: Residual .claude/ runtime write paths detected in production code:"
    echo ""
    echo "$matches"
    echo ""
    echo "These should use config.RuntimeDir() or config.ProjectMemoryDir() instead."
    echo "See pkg/config/paths.go for the migration API."
    exit 1
fi

echo "OK: No .claude/ runtime write paths in production Go code."
exit 0
