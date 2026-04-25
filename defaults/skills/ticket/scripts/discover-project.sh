#!/bin/bash
# discover-project.sh - Find ticket root for current project
#
# Supports:
#   1. .ticket-config.json with tickets_dir (direct path to series)
#   2. .ticket-config.json with tickets_root + .active-series file
#   3. Single-series: tickets/tickets-index.json at git root
#   4. Multi-series: tickets-root-index.json (fast) or shallow glob (fallback)

set -euo pipefail

SCRIPTS_DIR="$(cd "$(dirname "$0")" && pwd)"

# resolve_from_root_index uses tickets-root-index.json for fast O(1) lookup.
# Returns the best series directory path.
resolve_from_root_index() {
    local tickets_root="$1"
    local root_index="$tickets_root/tickets-root-index.json"

    [[ -f "$root_index" ]] || return 1

    # 1. If .active-series is set, resolve name → path via root index
    if [[ -f "$tickets_root/.active-series" ]]; then
        local active
        active=$(cat "$tickets_root/.active-series")

        # Try direct path first (backward compat: .active-series stores a relative path)
        if [[ -f "$tickets_root/$active/tickets-index.json" ]]; then
            echo "$tickets_root/$active"
            return 0
        fi

        # Look up by name in root index
        local mapped_path
        mapped_path=$(jq -r --arg name "$active" \
            '.series[] | select(.name == $name) | .path' \
            "$root_index" 2>/dev/null | head -1)
        if [[ -n "$mapped_path" ]] && [[ -f "$tickets_root/$mapped_path/tickets-index.json" ]]; then
            echo "$tickets_root/$mapped_path"
            return 0
        fi

        echo "WARNING: .active-series points to '$active' but no index found. Auto-selecting." >&2
    fi

    # 2. Pick best series from root index: most pending tickets with actionable next ticket
    local best_path=""
    while IFS= read -r series_path; do
        [[ -n "$series_path" ]] || continue
        local full_path="$tickets_root/$series_path"
        [[ -f "$full_path/tickets-index.json" ]] || continue

        local next_ticket
        next_ticket=$("$SCRIPTS_DIR/find-next-ticket.sh" "$full_path/tickets-index.json" 2>/dev/null) || true
        if [[ -n "$next_ticket" ]]; then
            echo "$full_path"
            return 0
        fi
    done < <(jq -r '[.series[] | select(.pending > 0)] | sort_by(-.pending) | .[].path' "$root_index" 2>/dev/null)

    # 3. Fallback: any series with pending from root index (even if deps block next ticket)
    local fallback_path
    fallback_path=$(jq -r '[.series[] | select(.pending > 0)] | .[0].path // empty' "$root_index" 2>/dev/null)
    if [[ -n "$fallback_path" ]] && [[ -f "$tickets_root/$fallback_path/tickets-index.json" ]]; then
        echo "$tickets_root/$fallback_path"
        return 0
    fi

    return 1
}

# resolve_multi_series takes a tickets/ directory that has subdirectories
# containing tickets-index.json files. Returns the best series directory.
#
# Priority:
#   1. tickets-root-index.json (fast, supports nested paths)
#   2. tickets/.active-series file (explicit user choice)
#   3. First series with an actionable pending ticket (most recently modified)
#   4. First series with any pending tickets
resolve_multi_series() {
    local tickets_root="$1"

    # Try root index first (handles nested series, fast lookup)
    result=$(resolve_from_root_index "$tickets_root") && { echo "$result"; return 0; }

    # Fallback: shallow glob (original behavior, one level deep only)

    # 1. Check .active-series file
    if [[ -f "$tickets_root/.active-series" ]]; then
        local active
        active=$(cat "$tickets_root/.active-series")
        if [[ -f "$tickets_root/$active/tickets-index.json" ]]; then
            echo "$tickets_root/$active"
            return 0
        fi
        echo "WARNING: .active-series points to '$active' but no index found. Auto-selecting." >&2
    fi

    # 2. Scan for series with actionable tickets (most recently modified first)
    local best_series=""
    local best_mtime=0

    for index in "$tickets_root"/*/tickets-index.json; do
        [[ -f "$index" ]] || continue
        local series_dir
        series_dir=$(dirname "$index")
        local next_ticket
        next_ticket=$("$SCRIPTS_DIR/find-next-ticket.sh" "$index" 2>/dev/null) || true
        if [[ -n "$next_ticket" ]]; then
            local mtime
            mtime=$(stat -c %Y "$index" 2>/dev/null || stat -f %m "$index" 2>/dev/null || echo 0)
            if [[ "$mtime" -gt "$best_mtime" ]]; then
                best_mtime="$mtime"
                best_series="$series_dir"
            fi
        fi
    done

    if [[ -n "$best_series" ]]; then
        echo "$best_series"
        return 0
    fi

    # 3. Fallback: any series with pending tickets (even if deps not met)
    for index in "$tickets_root"/*/tickets-index.json; do
        [[ -f "$index" ]] || continue
        local pending_count
        pending_count=$(jq '[.tickets[] | select(.status == "pending")] | length' "$index" 2>/dev/null || echo 0)
        if [[ "$pending_count" -gt 0 ]]; then
            dirname "$index"
            return 0
        fi
    done

    return 1
}

# 1. Check for .ticket-config.json in current directory or ancestors
current_dir="$PWD"
while [[ "$current_dir" != "/" ]]; do
    if [[ -f "$current_dir/.ticket-config.json" ]]; then
        # Check for direct tickets_dir (legacy: points to specific series)
        tickets_dir=$(jq -r '.tickets_dir // empty' "$current_dir/.ticket-config.json")
        if [[ -n "$tickets_dir" ]]; then
            full_path="$current_dir/$tickets_dir"
            if [[ -f "$full_path/tickets-index.json" ]]; then
                echo "$full_path"
                exit 0
            fi
            # tickets_dir set but no index - might be stale or a tickets root
            if [[ -d "$full_path" ]]; then
                result=$(resolve_multi_series "$full_path") && { echo "$result"; exit 0; }
            fi
        fi

        # Check for tickets_root (new: points to parent of series dirs)
        tickets_root=$(jq -r '.tickets_root // empty' "$current_dir/.ticket-config.json")
        if [[ -n "$tickets_root" ]]; then
            full_root="$current_dir/$tickets_root"
            if [[ -d "$full_root" ]]; then
                result=$(resolve_multi_series "$full_root") && { echo "$result"; exit 0; }
            fi
        fi
    fi
    current_dir=$(dirname "$current_dir")
done

# 2. Fallback: Check if we're in a git repo
if git_root=$(git rev-parse --show-toplevel 2>/dev/null); then
    for candidate in \
        "$git_root/migration_plan/tickets" \
        "$git_root/implementation_plan/tickets" \
        "$git_root/tickets"; do

        if [[ ! -d "$candidate" ]]; then
            continue
        fi

        # Direct index exists - single series
        if [[ -f "$candidate/tickets-index.json" ]]; then
            echo "$candidate"
            exit 0
        fi

        # Multi-series: subdirectories with their own indices
        result=$(resolve_multi_series "$candidate") && { echo "$result"; exit 0; }
    done
fi

# 3. No ticket directory found
echo "ERROR: No ticket directory found. Create .ticket-config.json or tickets/ directory." >&2
exit 1
