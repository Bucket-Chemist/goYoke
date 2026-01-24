#!/bin/bash
set -euo pipefail

# Ticket Extraction Script
# Extracts individual tickets from 06-week4-load-routing-context-v2.md

SOURCE_FILE="06-week4-load-routing-context-v2.md"
OUTPUT_DIR="tickets"

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Extract ticket metadata and boundaries
echo "Scanning source file for tickets..."

# Get all ticket headers and their line numbers
mapfile -t ticket_lines < <(grep -n "^## GOgent-" "$SOURCE_FILE" | cut -d: -f1)
mapfile -t ticket_ids < <(grep -o "^## GOgent-[0-9]\+:" "$SOURCE_FILE" | sed 's/^## //; s/://')

echo "Found ${#ticket_ids[@]} tickets: ${ticket_ids[*]}"

# Extract each ticket
for i in "${!ticket_ids[@]}"; do
    ticket_id="${ticket_ids[$i]}"
    start_line="${ticket_lines[$i]}"

    # Determine end line (start of next ticket or end of file)
    if [[ $i -lt $((${#ticket_ids[@]} - 1)) ]]; then
        end_line=$((${ticket_lines[$((i + 1))]} - 1))
    else
        end_line=$(wc -l < "$SOURCE_FILE")
    fi

    echo "Extracting $ticket_id (lines $start_line-$end_line)..."

    # Extract ticket content
    ticket_content=$(sed -n "${start_line},${end_line}p" "$SOURCE_FILE")

    # Parse metadata from content
    title=$(echo "$ticket_content" | head -1 | sed 's/^## GOgent-[0-9]\+: //')
    time=$(echo "$ticket_content" | grep "^\*\*Time\*\*:" | sed 's/\*\*Time\*\*: //' | sed 's/ hours\?/h/' | sed 's/ hour/h/')
    deps=$(echo "$ticket_content" | grep "^\*\*Dependencies\*\*:" | sed 's/\*\*Dependencies\*\*: //')
    priority=$(echo "$ticket_content" | grep "^\*\*Priority\*\*:" | sed 's/\*\*Priority\*\*: //' | cut -d' ' -f1)

    # Convert dependencies to YAML array
    if [[ "$deps" == "None"* ]] || [[ "$deps" == "GOgent-062"* ]]; then
        if [[ "$deps" == "None"* ]]; then
            yaml_deps="[]"
        else
            # Parse dependency list
            yaml_deps=$(echo "$deps" | sed 's/GOgent-/\n  - GOgent-/g' | sed '1d' | sed 's/, GOgent-/\n  - GOgent-/g')
            yaml_deps="[$yaml_deps]"
        fi
    else
        yaml_deps=$(echo "$deps" | sed 's/GOgent-/GOgent-/g' | sed 's/, / /g' | awk '{for(i=1;i<=NF;i++) printf "  - %s\n", $i}')
        if [[ -n "$yaml_deps" ]]; then
            yaml_deps="[\n$yaml_deps]"
        else
            yaml_deps="[]"
        fi
    fi

    # Count acceptance criteria checkboxes
    criteria_count=$(echo "$ticket_content" | grep -c "^- \[ \]" || true)

    # Determine week (056-066 = week 4, 067-070 = week 4 part B)
    week=4

    # Write ticket file with frontmatter
    output_file="$OUTPUT_DIR/${ticket_id}.md"

    cat > "$output_file" << EOF
---
id: $ticket_id
title: $title
description: $(echo "$ticket_content" | grep "^\*\*Task\*\*:" | sed 's/\*\*Task\*\*: //')
status: pending
time_estimate: $time
dependencies: $yaml_deps
priority: $priority
week: $week
tags:
  - session-start
  - week-4
tests_required: true
acceptance_criteria_count: $criteria_count
---

$ticket_content
EOF

    echo "  → Created $output_file"
done

echo ""
echo "Extraction complete! Created ${#ticket_ids[@]} ticket files in $OUTPUT_DIR/"
