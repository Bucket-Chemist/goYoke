#!/bin/bash
# gather-scout-metrics.sh - Collect metrics for /plan skill scout phase
# Usage: gather-scout-metrics.sh [target-directory]

set -euo pipefail

TARGET="${1:-.}"

# Count source files
FILES=$(find "$TARGET" -type f \( -name "*.go" -o -name "*.py" -o -name "*.ts" -o -name "*.js" -o -name "*.r" -o -name "*.R" \) 2>/dev/null | wc -l)

# Count total lines
LINES=$(find "$TARGET" -type f \( -name "*.go" -o -name "*.py" -o -name "*.ts" -o -name "*.js" -o -name "*.r" -o -name "*.R" \) -exec wc -l {} + 2>/dev/null | tail -1 | awk '{print $1}' || echo "0")

# Estimate tokens (rough: 4 chars per token, 40 chars per line average)
TOKENS=$((LINES * 10))

echo "files=$FILES"
echo "lines=$LINES"
echo "tokens_estimate=$TOKENS"
