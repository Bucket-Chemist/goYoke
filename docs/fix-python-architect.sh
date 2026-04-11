#!/usr/bin/env bash
set -euo pipefail

FILE="$HOME/.claude/agents/python-architect/python-architect.md"
echo "Fixing subagent_type in $FILE"
sed -i 's/^subagent_type: \["Plan", "Explore"\].*$/subagent_type: Python ML Architect/' "$FILE"

echo "=== Verification ==="
grep '^subagent_type:' "$FILE"
