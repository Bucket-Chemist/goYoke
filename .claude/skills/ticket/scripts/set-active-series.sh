#!/bin/bash
# set-active-series.sh - Set, clear, or list ticket series
#
# Usage:
#   set-active-series.sh list                 # List all series with status
#   set-active-series.sh set <series-name>    # Set active series (name or relative path)
#   set-active-series.sh clear                # Clear active series (auto-discover)
#   set-active-series.sh current              # Show current active series
#   set-active-series.sh rebuild              # Rebuild the root index

set -euo pipefail

SCRIPTS_DIR="$(cd "$(dirname "$0")" && pwd)"

find_tickets_root() {
    local current_dir="$PWD"
    while [[ "$current_dir" != "/" ]]; do
        if [[ -f "$current_dir/.ticket-config.json" ]]; then
            local base_dir
            base_dir=$(jq -r '.tickets_root // empty' "$current_dir/.ticket-config.json" 2>/dev/null)
            if [[ -n "$base_dir" ]]; then
                echo "$current_dir/$base_dir"
                return 0
            fi
        fi
        current_dir=$(dirname "$current_dir")
    done

    local git_root
    if git_root=$(git rev-parse --show-toplevel 2>/dev/null); then
        if [[ -d "$git_root/tickets" ]]; then
            echo "$git_root/tickets"
            return 0
        fi
    fi

    echo "ERROR: No tickets/ directory found" >&2
    return 1
}

# list_series_from_root_index uses the root index for a full recursive listing
list_series_from_root_index() {
    local tickets_root="$1"
    local root_index="$tickets_root/tickets-root-index.json"

    [[ -f "$root_index" ]] || return 1

    local active=""
    if [[ -f "$tickets_root/.active-series" ]]; then
        active=$(cat "$tickets_root/.active-series")
    fi

    jq -r '.series[] | [.name, .path, (.total|tostring), (.pending|tostring), (.completed|tostring)] | @tsv' "$root_index" | \
    while IFS=$'\t' read -r name path total pending completed; do
        local marker=""
        if [[ "$name" == "$active" ]] || [[ "$path" == "$active" ]]; then
            marker=" <- active"
        fi
        printf "  %-30s %s total, %s pending, %s completed%s\n" \
            "$name ($path)" "$total" "$pending" "$completed" "$marker"
    done
    return 0
}

# list_series_shallow uses the original one-level-deep glob
list_series_shallow() {
    local tickets_root="$1"

    local active=""
    if [[ -f "$tickets_root/.active-series" ]]; then
        active=$(cat "$tickets_root/.active-series")
    fi

    for index in "$tickets_root"/*/tickets-index.json; do
        [[ -f "$index" ]] || continue
        local series_dir series_name total pending completed marker
        series_dir=$(dirname "$index")
        series_name=$(basename "$series_dir")
        total=$(jq '.tickets | length' "$index")
        pending=$(jq '[.tickets[] | select(.status == "pending")] | length' "$index")
        completed=$(jq '[.tickets[] | select(.status == "complete" or .status == "completed")] | length' "$index")
        marker=""
        if [[ "$series_name" == "$active" ]]; then
            marker=" <- active"
        fi
        printf "  %-30s %d total, %d pending, %d completed%s\n" \
            "$series_name" "$total" "$pending" "$completed" "$marker"
    done
}

list_series() {
    local tickets_root="$1"
    # Prefer root index (includes nested series), fall back to shallow glob
    list_series_from_root_index "$tickets_root" 2>/dev/null || list_series_shallow "$tickets_root"
}

# resolve_series_name looks up a name in the root index to find its relative path
resolve_series_name() {
    local tickets_root="$1"
    local name="$2"
    local root_index="$tickets_root/tickets-root-index.json"

    # Direct match: name is already a valid path
    if [[ -f "$tickets_root/$name/tickets-index.json" ]]; then
        echo "$name"
        return 0
    fi

    # Look up by name in root index
    if [[ -f "$root_index" ]]; then
        local mapped_path
        mapped_path=$(jq -r --arg name "$name" \
            '.series[] | select(.name == $name) | .path' \
            "$root_index" 2>/dev/null | head -1)
        if [[ -n "$mapped_path" ]] && [[ -f "$tickets_root/$mapped_path/tickets-index.json" ]]; then
            echo "$mapped_path"
            return 0
        fi
    fi

    return 1
}

action="${1:-}"
arg="${2:-}"
tickets_root=$(find_tickets_root) || exit 1

case "$action" in
    list)
        echo "Ticket series in $tickets_root:"
        echo ""
        list_series "$tickets_root"
        ;;
    set)
        if [[ -z "$arg" ]]; then
            echo "Usage: set-active-series.sh set <series-name>" >&2
            exit 1
        fi
        resolved=$(resolve_series_name "$tickets_root" "$arg") || {
            echo "ERROR: Series '$arg' not found." >&2
            echo "" >&2
            echo "Available series:" >&2
            list_series "$tickets_root" >&2
            exit 1
        }
        echo "$resolved" > "$tickets_root/.active-series"
        echo "Active series set to: $arg"
        echo "Resolved path: $tickets_root/$resolved"
        ;;
    clear)
        rm -f "$tickets_root/.active-series"
        echo "Active series cleared. Discovery will auto-select next actionable series."
        ;;
    current)
        if [[ -f "$tickets_root/.active-series" ]]; then
            active=$(cat "$tickets_root/.active-series")
            echo "Active series: $active"
            echo "Tickets dir: $tickets_root/$active"
        else
            echo "No active series set. Discovery will auto-select."
        fi
        ;;
    rebuild)
        "$SCRIPTS_DIR/rebuild-index.sh" "$tickets_root"
        ;;
    *)
        echo "Usage: set-active-series.sh {list|set <name>|clear|current|rebuild}" >&2
        exit 1
        ;;
esac
