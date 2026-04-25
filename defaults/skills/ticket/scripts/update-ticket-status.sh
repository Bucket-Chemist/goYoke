#!/bin/bash
# update-ticket-status.sh - Update ticket status in index atomically

set -euo pipefail

TICKETS_INDEX="$1"
TICKET_ID="$2"
NEW_STATUS="$3"  # pending|in_progress|completed|blocked

# Validate status
if [[ ! "$NEW_STATUS" =~ ^(pending|in_progress|completed|complete|blocked)$ ]]; then
    echo "ERROR: Invalid status '$NEW_STATUS'. Must be: pending, in_progress, completed, blocked" >&2
    exit 1
fi

# Normalize "complete" to "completed"
if [[ "$NEW_STATUS" == "complete" ]]; then
    NEW_STATUS="completed"
fi

# Atomic update using jq
temp_file=$(mktemp)
jq --arg id "$TICKET_ID" --arg status "$NEW_STATUS" \
   '(.tickets[] | select(.id == $id) | .status) = $status' \
   "$TICKETS_INDEX" > "$temp_file"

# Verify update worked
if ! grep -q "\"id\": \"$TICKET_ID\"" "$temp_file"; then
    echo "ERROR: Ticket $TICKET_ID not found in index" >&2
    rm "$temp_file"
    exit 1
fi

# Atomic move
mv "$temp_file" "$TICKETS_INDEX"
echo "Updated $TICKET_ID → $NEW_STATUS"
