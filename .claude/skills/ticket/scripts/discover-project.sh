#!/bin/bash
# discover-project.sh - Find ticket root for current project

set -euo pipefail

# 1. Check for .ticket-config.json in current directory or ancestors
current_dir="$PWD"
while [[ "$current_dir" != "/" ]]; do
    if [[ -f "$current_dir/.ticket-config.json" ]]; then
        # Extract tickets_dir from config
        tickets_dir=$(jq -r '.tickets_dir // "tickets"' "$current_dir/.ticket-config.json")
        echo "$current_dir/$tickets_dir"
        exit 0
    fi
    current_dir=$(dirname "$current_dir")
done

# 2. Fallback: Check if we're in a git repo
if git_root=$(git rev-parse --show-toplevel 2>/dev/null); then
    # Search for tickets/ or migration_plan/tickets/
    if [[ -d "$git_root/migration_plan/tickets" ]]; then
        echo "$git_root/migration_plan/tickets"
        exit 0
    elif [[ -d "$git_root/implementation_plan/tickets" ]]; then
        echo "$git_root/implementation_plan/tickets"
        exit 0
    elif [[ -d "$git_root/tickets" ]]; then
        echo "$git_root/tickets"
        exit 0
    fi
fi

# 3. No ticket directory found
echo "ERROR: No ticket directory found. Create .ticket-config.json or tickets/ directory." >&2
exit 1
