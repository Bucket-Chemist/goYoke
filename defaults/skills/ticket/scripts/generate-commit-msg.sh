#!/bin/bash
# generate-commit-msg.sh - Generate commit message from ticket

set -euo pipefail

TICKET_FILE="$1"
TICKET_ID="${2:-}"

if [[ ! -f "$TICKET_FILE" ]]; then
    echo "ERROR: Ticket file not found: $TICKET_FILE" >&2
    exit 1
fi

# Extract frontmatter fields using Python
read -r ticket_id title description tags <<< $(python3 -c "
import sys
try:
    import frontmatter
    post = frontmatter.load('$TICKET_FILE')
    m = post.metadata
    print(m.get('id', 'UNKNOWN'), m.get('title', 'Unknown'), m.get('description', 'Unknown'), ','.join(m.get('tags', [])))
except Exception as e:
    print('UNKNOWN Unknown Unknown none', file=sys.stderr)
    sys.exit(1)
" 2>/dev/null || echo "UNKNOWN Unknown Unknown none")

# GRACEFUL FALLBACK: If frontmatter extraction failed and ticket_id provided, extract from tickets-index.json
if [[ "$ticket_id" == "UNKNOWN" && -n "$TICKET_ID" ]]; then
    # Find tickets-index.json (could be in same dir as ticket file or .ticket-config.json)
    ticket_dir=$(dirname "$TICKET_FILE")
    index_file="$ticket_dir/tickets-index.json"

    if [[ ! -f "$index_file" ]]; then
        # Try discovering from .ticket-config.json
        if [[ -f ".ticket-config.json" ]]; then
            tickets_dir=$(jq -r '.tickets_dir' .ticket-config.json)
            index_file="$tickets_dir/tickets-index.json"
        fi
    fi

    if [[ -f "$index_file" ]]; then
        # Extract from tickets-index.json
        ticket_data=$(jq -r ".tickets[] | select(.id == \"$TICKET_ID\")" "$index_file")

        if [[ -n "$ticket_data" ]]; then
            ticket_id="$TICKET_ID"
            title=$(echo "$ticket_data" | jq -r '.title // "Unknown"')
            description=$(echo "$ticket_data" | jq -r '.description // "Unknown"')
            tags=$(echo "$ticket_data" | jq -r '.tags // [] | join(",")')

            # Add files_to_create to description if available
            files_to_create=$(echo "$ticket_data" | jq -r '.files_to_create // [] | join(", ")')
            if [[ -n "$files_to_create" && "$files_to_create" != "null" ]]; then
                description="$description

Files created: $files_to_create"
            fi
        fi
    fi
fi

# Determine commit type from tags
commit_type="feat"
if [[ "$tags" =~ bugfix ]]; then
    commit_type="fix"
elif [[ "$tags" =~ test ]]; then
    commit_type="test"
elif [[ "$tags" =~ docs ]]; then
    commit_type="docs"
fi

# Generate message
cat <<EOF
$commit_type: $ticket_id - $title

$description

Ticket-Id: $ticket_id

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
