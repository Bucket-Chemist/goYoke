#!/bin/bash
set -euo pipefail

# Ticket Extraction Script for Week 5 (integration-tests)
# Extracts individual tickets from 10-week5-integration-tests.md
# Special handling for GOgent-004c with letter suffix

SOURCE_FILE="10-week5-integration-tests.md"
OUTPUT_DIR="tickets"

mkdir -p "$OUTPUT_DIR"

echo "Scanning source file for tickets..."

# Pattern matches both GOgent-XXX and GOgent-XXXx (with letter suffix)
mapfile -t ticket_lines < <(grep -n "^### GOgent-" "$SOURCE_FILE" | cut -d: -f1)
mapfile -t ticket_ids < <(grep -o "^### GOgent-[0-9a-z]\+:" "$SOURCE_FILE" | sed 's/^### //; s/://')

echo "Found ${#ticket_ids[@]} tickets: ${ticket_ids[*]}"

for i in "${!ticket_ids[@]}"; do
    ticket_id="${ticket_ids[$i]}"
    start_line="${ticket_lines[$i]}"

    if [[ $i -lt $((${#ticket_ids[@]} - 1)) ]]; then
        end_line=$((${ticket_lines[$((i + 1))]} - 1))
    else
        cross_ref_line=$(grep -n "^## Cross-File References" "$SOURCE_FILE" | cut -d: -f1 || echo "")
        if [[ -n "$cross_ref_line" ]]; then
            end_line=$((cross_ref_line - 1))
        else
            end_line=$(wc -l < "$SOURCE_FILE")
        fi
    fi

    echo "Extracting $ticket_id (lines $start_line-$end_line)..."

    ticket_content=$(sed -n "${start_line},${end_line}p" "$SOURCE_FILE")
    title=$(echo "$ticket_content" | head -1 | sed 's/^### GOgent-[0-9a-z]\+: //')
    time=$(echo "$ticket_content" | grep "^\*\*Time\*\*:" | sed 's/\*\*Time\*\*: //' | sed 's/ hours\?/h/')
    deps=$(echo "$ticket_content" | grep "^\*\*Dependencies\*\*:" | sed 's/\*\*Dependencies\*\*: //')
    priority="high"

    if [[ "$deps" == "None"* ]] || [[ -z "$deps" ]]; then
        json_deps="[]"
    else
        json_deps=$(echo "$deps" | grep -o "GOgent-[0-9a-z]\+" | awk '{printf "\"%s\",", $0}' | sed 's/,$//')
        json_deps="[$json_deps]"
    fi

    criteria_count=$(echo "$ticket_content" | grep -c "^- \[ \]" || true)
    week=5

    # Determine tags based on ticket ID
    if [[ "$ticket_id" == "GOgent-004c" ]]; then
        tags="[\"config-tests\", \"week-5\", \"deferred\"]"
    elif [[ "$ticket_id" =~ GOgent-09[4-7] ]]; then
        tags="[\"integration-tests\", \"week-5\"]"
    elif [[ "$ticket_id" =~ GOgent-09[8-9] ]] || [[ "$ticket_id" == "GOgent-100" ]]; then
        tags="[\"performance\", \"week-5\"]"
    else
        tags="[\"week-5\"]"
    fi

    task_desc=$(echo "$ticket_content" | grep "^\*\*Task\*\*:" | sed 's/\*\*Task\*\*: //')

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
