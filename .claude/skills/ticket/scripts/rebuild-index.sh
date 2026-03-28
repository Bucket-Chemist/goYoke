#!/bin/bash
# rebuild-index.sh - Scan for all tickets-index.json files and build root index
#
# Usage:
#   rebuild-index.sh                    # Auto-detect tickets root
#   rebuild-index.sh /path/to/tickets   # Explicit tickets root
#
# Output: tickets-root-index.json at the tickets root

set -euo pipefail

SCRIPTS_DIR="$(cd "$(dirname "$0")" && pwd)"

find_tickets_root() {
    local current_dir="$PWD"
    while [[ "$current_dir" != "/" ]]; do
        if [[ -f "$current_dir/.ticket-config.json" ]]; then
            local tickets_root
            tickets_root=$(jq -r '.tickets_root // empty' "$current_dir/.ticket-config.json" 2>/dev/null)
            if [[ -n "$tickets_root" ]]; then
                echo "$current_dir/$tickets_root"
                return 0
            fi
            local tickets_dir
            tickets_dir=$(jq -r '.tickets_dir // empty' "$current_dir/.ticket-config.json" 2>/dev/null)
            if [[ -n "$tickets_dir" ]]; then
                local full="$current_dir/$tickets_dir"
                if [[ -d "$full" ]]; then
                    dirname "$full"
                    return 0
                fi
            fi
        fi
        current_dir=$(dirname "$current_dir")
    done

    local git_root
    if git_root=$(git rev-parse --show-toplevel 2>/dev/null); then
        for candidate in "$git_root/tickets" "$git_root/implementation_plan/tickets" "$git_root/migration_plan/tickets"; do
            if [[ -d "$candidate" ]]; then
                echo "$candidate"
                return 0
            fi
        done
    fi

    echo "ERROR: No tickets root found" >&2
    return 1
}

tickets_root="${1:-}"
if [[ -z "$tickets_root" ]]; then
    tickets_root=$(find_tickets_root) || exit 1
fi

# Normalise to absolute path
tickets_root=$(cd "$tickets_root" && pwd)

index_file="$tickets_root/tickets-root-index.json"
timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build series array by scanning recursively
series_json="[]"
while IFS= read -r idx_path; do
    [[ -f "$idx_path" ]] || continue

    series_dir=$(dirname "$idx_path")
    prefix="$tickets_root/"
    rel_path=${series_dir#$prefix}
    series_name=$(basename "$series_dir")

    total=$(jq '.tickets | length' "$idx_path" 2>/dev/null || echo 0)
    pending=$(jq '[.tickets[] | select(.status == "pending")] | length' "$idx_path" 2>/dev/null || echo 0)
    completed=$(jq '[.tickets[] | select(.status == "complete" or .status == "completed")] | length' "$idx_path" 2>/dev/null || echo 0)
    in_progress=$(jq '[.tickets[] | select(.status == "in_progress")] | length' "$idx_path" 2>/dev/null || echo 0)

    series_json=$(echo "$series_json" | jq \
        --arg name "$series_name" \
        --arg path "$rel_path" \
        --argjson total "$total" \
        --argjson pending "$pending" \
        --argjson completed "$completed" \
        --argjson in_progress "$in_progress" \
        '. + [{
            "name": $name,
            "path": $path,
            "total": $total,
            "pending": $pending,
            "completed": $completed,
            "in_progress": $in_progress
        }]')
done < <(find "$tickets_root" -name "tickets-index.json" -not -name "tickets-root-index.json" 2>/dev/null | sort)

# Write root index
jq -n \
    --argjson version 1 \
    --arg generated_at "$timestamp" \
    --arg tickets_root "$tickets_root" \
    --argjson series "$series_json" \
    '{
        "version": $version,
        "generated_at": $generated_at,
        "tickets_root": $tickets_root,
        "series": $series
    }' > "$index_file"

# Summary
total_series=$(echo "$series_json" | jq 'length')
active_series=$(echo "$series_json" | jq '[.[] | select(.pending > 0)] | length')
echo "Root index rebuilt: $index_file"
echo "  Total series: $total_series"
echo "  Active (with pending): $active_series"
