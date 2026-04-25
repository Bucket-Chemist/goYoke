#!/bin/bash
# find-next-ticket.sh - Find next pending ticket respecting dependencies

set -euo pipefail

TICKETS_INDEX="$1"  # Path to tickets-index.json

if [[ ! -f "$TICKETS_INDEX" ]]; then
    echo "ERROR: tickets-index.json not found at $TICKETS_INDEX" >&2
    exit 1
fi

# Extract completed ticket IDs
completed_ids=$(jq -r '[.tickets[] | select(.status == "complete" or .status == "completed") | .id]' "$TICKETS_INDEX")

# Find first pending ticket with met dependencies
jq -r --argjson completed_ids "$completed_ids" '
.tickets[]
| select(.status == "pending")
| select(
    (.dependencies | length == 0) or
    (.dependencies | all(. as $dep | $completed_ids | index($dep) != null))
  )
| .id
' "$TICKETS_INDEX" | head -1

# Exit code 0 even if no ticket (empty output handled by caller)
