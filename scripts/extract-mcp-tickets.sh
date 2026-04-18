#!/bin/bash
# extract-mcp-tickets.sh
#
# Extracts individual ticket files from MCP_IMPLEMENTATION_GUIDE_V2.md
# Creates properly formatted markdown files with YAML frontmatter
#
# Usage: ./scripts/extract-mcp-tickets.sh [--dry-run]
#
# Output directory: tickets/mcp/

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
SOURCE_FILE="$PROJECT_ROOT/tickets/mcp/MCP_IMPLEMENTATION_GUIDE_V2.md"
OUTPUT_DIR="$PROJECT_ROOT/tickets/mcp"
DRY_RUN=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case "$1" in
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--dry-run]"
            exit 1
            ;;
    esac
done

# Check source file exists
if [[ ! -f "$SOURCE_FILE" ]]; then
    echo "Error: Source file not found: $SOURCE_FILE"
    exit 1
fi

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Temporary files
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

echo "Extracting tickets from: $SOURCE_FILE"
echo "Output directory: $OUTPUT_DIR"
echo ""

# Extract tickets using awk
awk '
BEGIN {
    in_ticket = 0
    ticket_count = 0
}

# Match ticket header: #### goYoke-MCP-XXX: Title
/^#### goYoke-MCP-[0-9]+:/ {
    # If we were capturing a ticket, write it
    if (in_ticket && ticket_id != "") {
        write_ticket()
    }

    # Parse ticket ID and title
    ticket_id = $2
    gsub(/:$/, "", ticket_id)

    # Extract title (everything after the ID:)
    title = $0
    sub(/^#### goYoke-MCP-[0-9]+: /, "", title)

    in_ticket = 1
    ticket_content = ""
    time_est = ""
    priority = ""
    deps = ""
    ticket_count++

    next
}

# End of ticket: --- line (but not the first one right after header)
/^---$/ && in_ticket {
    write_ticket()
    in_ticket = 0
    next
}

# Capture ticket content
in_ticket {
    ticket_content = ticket_content $0 "\n"

    # Extract metadata from content
    if ($0 ~ /^\*\*Time:\*\*/) {
        time_est = $0
        gsub(/^\*\*Time:\*\* /, "", time_est)
    }
    if ($0 ~ /^\*\*Priority:\*\*/) {
        priority = $0
        # First remove the prefix
        gsub(/^\*\*Priority:\*\* /, "", priority)
        # Then remove any parenthetical (priority may be "HIGH (critical path)")
        gsub(/ *\(.*\).*$/, "", priority)
    }
    if ($0 ~ /^\*\*Dependencies:\*\*/) {
        deps = $0
        gsub(/^\*\*Dependencies:\*\* /, "", deps)
    }
}

function write_ticket() {
    if (ticket_id == "") return

    # Normalize priority - extract just the priority level
    if (priority == "") priority = "MEDIUM"
    priority = toupper(priority)
    # Handle patterns like "CRITICAL (blocks Phase 3)" or "HIGH (critical path)"
    if (priority ~ /CRITICAL/) priority = "CRITICAL"
    else if (priority ~ /HIGH/) priority = "HIGH"
    else if (priority ~ /MEDIUM/) priority = "MEDIUM"
    else if (priority ~ /LOW/) priority = "LOW"
    else priority = "MEDIUM"

    # Create filename
    filename = TEMP_DIR "/" ticket_id ".md"

    # Write file with frontmatter
    print "---" > filename
    print "id: " ticket_id >> filename
    print "title: \"" title "\"" >> filename
    print "time: \"" time_est "\"" >> filename
    print "priority: " priority >> filename
    print "dependencies: \"" deps "\"" >> filename
    print "status: pending" >> filename
    print "---" >> filename
    print "" >> filename
    print "# " ticket_id ": " title >> filename
    print "" >> filename
    print ticket_content >> filename

    close(filename)

    print "  Extracted: " ticket_id " (" priority ")"
}

END {
    # Handle last ticket if file doesnt end with ---
    if (in_ticket && ticket_id != "") {
        write_ticket()
    }
    print ""
    print "Total tickets extracted: " ticket_count
}
' TEMP_DIR="$TEMP_DIR" "$SOURCE_FILE"

# Move files to output directory (or just list in dry-run mode)
echo ""
if [[ "$DRY_RUN" == "true" ]]; then
    echo "DRY RUN - would create these files:"
    ls -1 "$TEMP_DIR"/*.md 2>/dev/null | while read -r f; do
        basename "$f"
    done
else
    # Copy files to output directory
    count=0
    for f in "$TEMP_DIR"/*.md; do
        if [[ -f "$f" ]]; then
            filename=$(basename "$f")
            cp "$f" "$OUTPUT_DIR/$filename"
            count=$((count + 1))
        fi
    done
    echo "Created $count ticket files in $OUTPUT_DIR"

    # Create index file
    INDEX_FILE="$OUTPUT_DIR/TICKET_INDEX.md"
    echo "# MCP Implementation Tickets" > "$INDEX_FILE"
    echo "" >> "$INDEX_FILE"
    echo "Generated: $(date -Iseconds)" >> "$INDEX_FILE"
    echo "Source: MCP_IMPLEMENTATION_GUIDE_V2.md" >> "$INDEX_FILE"
    echo "" >> "$INDEX_FILE"
    echo "## Ticket Summary" >> "$INDEX_FILE"
    echo "" >> "$INDEX_FILE"
    echo "| ID | Title | Priority | Time | Dependencies |" >> "$INDEX_FILE"
    echo "|:---|:------|:---------|:-----|:-------------|" >> "$INDEX_FILE"

    # Parse each ticket file for index
    for f in "$OUTPUT_DIR"/goYoke-MCP-*.md; do
        if [[ -f "$f" ]]; then
            # Extract frontmatter values
            id=$(grep "^id:" "$f" | head -1 | sed 's/^id: //')
            title=$(grep "^title:" "$f" | head -1 | sed 's/^title: "//' | sed 's/"$//')
            priority=$(grep "^priority:" "$f" | head -1 | sed 's/^priority: //')
            time=$(grep "^time:" "$f" | head -1 | sed 's/^time: "//' | sed 's/"$//')
            deps=$(grep "^dependencies:" "$f" | head -1 | sed 's/^dependencies: "//' | sed 's/"$//')

            echo "| $id | $title | $priority | $time | $deps |" >> "$INDEX_FILE"
        fi
    done

    echo "" >> "$INDEX_FILE"
    echo "## Tickets by Priority" >> "$INDEX_FILE"
    echo "" >> "$INDEX_FILE"

    for prio in CRITICAL HIGH MEDIUM LOW; do
        tickets=$(grep -l "^priority: $prio" "$OUTPUT_DIR"/goYoke-MCP-*.md 2>/dev/null | wc -l || true)
        if [[ $tickets -gt 0 ]]; then
            echo "### $prio ($tickets)" >> "$INDEX_FILE"
            echo "" >> "$INDEX_FILE"
            grep -l "^priority: $prio" "$OUTPUT_DIR"/goYoke-MCP-*.md 2>/dev/null | while read -r f; do
                id=$(grep "^id:" "$f" | head -1 | sed 's/^id: //')
                title=$(grep "^title:" "$f" | head -1 | sed 's/^title: "//' | sed 's/"$//')
                echo "- [$id](./$id.md): $title" >> "$INDEX_FILE"
            done
            echo "" >> "$INDEX_FILE"
        fi
    done

    echo "Created index: $INDEX_FILE"
fi

echo ""
echo "Done!"
