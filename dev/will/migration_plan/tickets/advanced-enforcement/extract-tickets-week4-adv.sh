#!/bin/bash
set -euo pipefail

# Ticket Extraction Script for Week 4 (advanced-enforcement)
# Extracts individual tickets from 08-week4-advanced-enforcement.md

SOURCE_FILE="08-week4-advanced-enforcement.md"
OUTPUT_DIR="tickets"

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Extract ticket metadata and boundaries
echo "Scanning source file for tickets..."

# Get all ticket headers and their line numbers (### GOgent- pattern)
mapfile -t ticket_lines < <(grep -n "^### GOgent-" "$SOURCE_FILE" | cut -d: -f1)
mapfile -t ticket_ids < <(grep -o "^### GOgent-[0-9]\+:" "$SOURCE_FILE" | sed 's/^### //; s/://')

echo "Found ${#ticket_ids[@]} tickets: ${ticket_ids[*]}"

# Extract each ticket
for i in "${!ticket_ids[@]}"; do
    ticket_id="${ticket_ids[$i]}"
    start_line="${ticket_lines[$i]}"

    # Determine end line (start of next ticket or special markers)
    if [[ $i -lt $((${#ticket_ids[@]} - 1)) ]]; then
        end_line=$((${ticket_lines[$((i + 1))]} - 1))
    else
        # Find the "Cross-File References" section or end of file
        cross_ref_line=$(grep -n "^## Cross-File References" "$SOURCE_FILE" | cut -d: -f1 || echo "")
        if [[ -n "$cross_ref_line" ]]; then
            end_line=$((cross_ref_line - 1))
        else
            end_line=$(wc -l < "$SOURCE_FILE")
        fi
    fi

    echo "Extracting $ticket_id (lines $start_line-$end_line)..."

    # Extract ticket content
    ticket_content=$(sed -n "${start_line},${end_line}p" "$SOURCE_FILE")

    # Parse metadata from content
    title=$(echo "$ticket_content" | head -1 | sed 's/^### GOgent-[0-9]\+: //')
    time=$(echo "$ticket_content" | grep "^\*\*Time\*\*:" | sed 's/\*\*Time\*\*: //' | sed 's/ hours\?/h/')
    deps=$(echo "$ticket_content" | grep "^\*\*Dependencies\*\*:" | sed 's/\*\*Dependencies\*\*: //')

    # Default priority to high
    priority="high"

    # Convert dependencies to JSON array format
    if [[ "$deps" == "None"* ]] || [[ -z "$deps" ]]; then
        json_deps="[]"
    else
        # Extract GOgent-XXX patterns
        json_deps=$(echo "$deps" | grep -o "GOgent-[0-9a-z]\+" | awk '{printf "\"%s\",", $0}' | sed 's/,$//')
        json_deps="[$json_deps]"
    fi

    # Count acceptance criteria checkboxes
    criteria_count=$(echo "$ticket_content" | grep -c "^- \[ \]" || true)

    # Week 5 for all these tickets
    week=5

    # Determine tags based on ticket range
    if [[ "$ticket_id" =~ GOgent-07[5-9] ]]; then
        tags="[\"orchestrator-guard\", \"week-5\"]"
    elif [[ "$ticket_id" =~ GOgent-08[0-3] ]]; then
        tags="[\"doc-theater\", \"week-5\"]"
    else
        tags="[\"week-5\"]"
    fi

    # Extract task description
    task_desc=$(echo "$ticket_content" | grep "^\*\*Task\*\*:" | sed 's/\*\*Task\*\*: //')

    # Write ticket file with frontmatter
    output_file="$OUTPUT_DIR/${ticket_id}.md"

    cat > "$output_file" << EOF
---
id: $ticket_id
title: $title
description: $task_desc
status: pending
time_estimate: $time
dependencies: $json_deps
priority: $priority
week: $week
tags: $tags
tests_required: true
acceptance_criteria_count: $criteria_count
---

$ticket_content
EOF

    echo "  → Created $output_file"
done

echo ""
echo "Extraction complete! Created ${#ticket_ids[@]} ticket files in $OUTPUT_DIR/"
