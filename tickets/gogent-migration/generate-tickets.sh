#!/usr/bin/env bash
# Generate individual ticket .md files from implementation-plan.json
set -euo pipefail

PLAN=".gogent/sessions/20260331-plan-tickets-gogent-migration/implementation-plan.json"
TICKET_DIR="tickets/gogent-migration/tickets"
INDEX="tickets/gogent-migration/tickets-index.json"

mkdir -p "$TICKET_DIR"

# Phase mapping
phase_for_task() {
    case "$1" in
        1|2) echo 1 ;;
        3|4|5|6|7|8|9|10) echo 2 ;;
        11|12|13|14|15|16|17) echo 3 ;;
        18|19|20) echo 4 ;;
        21|22|23) echo 5 ;;
    esac
}

task_count=$(jq '.tasks | length' "$PLAN")
echo "Generating $task_count tickets..."

# Start index
printf '{"version":"1.0","project":"MIG","generated_by":"/plan-tickets v1.0 (braintrust-sourced)","generated_at":"2026-03-31T09:44:00Z","tickets":[\n' > "$INDEX"

for i in $(seq 0 $((task_count - 1))); do
    num=$((i + 1))
    mig_id=$(printf "MIG-%03d" "$num")
    phase=$(phase_for_task "$num")
    title=$(jq -r ".tasks[$i].subject" "$PLAN")
    desc=$(jq -r ".tasks[$i].description" "$PLAN")
    agent=$(jq -r ".tasks[$i].agent" "$PLAN")

    # Dependencies as "MIG-001, MIG-003" format
    deps=$(jq -r "[.tasks[$i].blocked_by[] | ltrimstr(\"task-\") | tonumber] | map(\"MIG-\" + (if . < 10 then \"00\" + tostring elif . < 100 then \"0\" + tostring else tostring end)) | join(\", \")" "$PLAN")
    deps_json=$(jq -c "[.tasks[$i].blocked_by[] | ltrimstr(\"task-\") | tonumber] | map(\"MIG-\" + (if . < 10 then \"00\" + tostring elif . < 100 then \"0\" + tostring else tostring end))" "$PLAN" 2>/dev/null || echo "[]")

    # Acceptance criteria
    criteria=$(jq -r ".tasks[$i].acceptance_criteria // [] | map(\"- [ ] \" + .) | join(\"\n\")" "$PLAN")

    # Related files (plain text, no backticks in jq)
    files=$(jq -r ".tasks[$i].related_files // [] | map(\"- \" + .path + \" -- \" + .relevance) | join(\"\n\")" "$PLAN")

    # Size based on description length
    desc_len=${#desc}
    if [[ $desc_len -lt 400 ]]; then size="S"
    elif [[ $desc_len -lt 1200 ]]; then size="M"
    else size="L"
    fi

    # Write ticket file (use quoted heredoc to prevent expansion issues)
    cat > "$TICKET_DIR/$mig_id.md" <<EOF
---
id: $mig_id
title: "$title"
status: pending
dependencies: [$deps]
phase: $phase
tags: [plan-generated, phase-$phase, gogent-migration]
needs_planning: false
agent: $agent
size: $size
---

# $mig_id: $title

## Description

$desc

## Acceptance Criteria

$criteria

## Files

$files

## Context

Phase $phase of the .claude/ to .gogent/ runtime I/O migration.
Source: Braintrust analysis + Architect specs.

---

_Generated from: .gogent/sessions/20260331-plan-tickets-gogent-migration/specs.md Phase ${phase}_
EOF

    # Add to index
    comma=","
    [[ $i -eq $((task_count - 1)) ]] && comma=""
    title_json=$(jq -c ".tasks[$i].subject" "$PLAN")
    printf '{"id":"%s","title":%s,"status":"pending","phase":%d,"dependencies":%s,"file":"tickets/gogent-migration/tickets/%s.md","size":"%s"}%s\n' \
        "$mig_id" "$title_json" "$phase" "$deps_json" "$mig_id" "$size" "$comma" >> "$INDEX"

    echo "  $mig_id: $title"
done

echo ']}' >> "$INDEX"

# Pretty-print the index
jq '.' "$INDEX" > "${INDEX}.tmp" && mv "${INDEX}.tmp" "$INDEX"

echo ""
echo "Done! Generated $task_count tickets in $TICKET_DIR/"
echo "Index: $INDEX"
echo "Overview: tickets/gogent-migration/overview.md"
