#!/bin/bash
set -euo pipefail

# Ticket Extraction Script for Week 4 (observability-remaining)
# Extracts individual tickets from 09-week4-observability-remaining.md

SOURCE_FILE="09-week4-observability-remaining.md"
OUTPUT_DIR="tickets"

mkdir -p "$OUTPUT_DIR"

echo "Scanning source file for tickets..."

mapfile -t ticket_lines < <(grep -n "^### GOgent-" "$SOURCE_FILE" | cut -d: -f1)
mapfile -t ticket_ids < <(grep -o "^### GOgent-[0-9]\+:" "$SOURCE_FILE" | sed 's/^### //; s/://')

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
    title=$(echo "$ticket_content" | head -1 | sed 's/^### GOgent-[0-9]\+: //')
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
    week=4

    if [[ "$ticket_id" =~ GOgent-08[7-9] ]] || [[ "$ticket_id" == "GOgent-090" ]]; then
        tags="[\"benchmark-logger\", \"week-4\"]"
    elif [[ "$ticket_id" =~ GOgent-09[1-2] ]]; then
        tags="[\"stop-gate\", \"week-4\"]"
    else
        tags="[\"observability\", \"week-4\"]"
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
