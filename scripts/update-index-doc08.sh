#!/bin/bash
set -euo pipefail

cat > new-tickets-08.json << 'NEWTICKETS'
[
  {"id": "goYoke-075", "title": "SubagentStop Event Parsing for Orchestrator", "description": "Detect orchestrator/architect completion events", "file": "advanced-enforcement/tickets/goYoke-075.md", "week": 5, "time_estimate": "1.5h", "dependencies": ["goYoke-063"], "priority": "high", "tags": ["orchestrator-guard", "week-5"], "tests_required": true, "acceptance_criteria_count": 6, "status": "pending"},
  {"id": "goYoke-076", "title": "Transcript Analysis & Task Tracking", "description": "Scan transcripts for background tasks and TaskOutput collections", "file": "advanced-enforcement/tickets/goYoke-076.md", "week": 5, "time_estimate": "2.5h", "dependencies": ["goYoke-075"], "priority": "high", "tags": ["orchestrator-guard", "week-5"], "tests_required": true, "acceptance_criteria_count": 8, "status": "pending"},
  {"id": "goYoke-077", "title": "Blocking Response Generation", "description": "Generate blocking response if background tasks uncollected", "file": "advanced-enforcement/tickets/goYoke-077.md", "week": 5, "time_estimate": "2h", "dependencies": ["goYoke-076"], "priority": "high", "tags": ["orchestrator-guard", "week-5"], "tests_required": true, "acceptance_criteria_count": 7, "status": "pending"},
  {"id": "goYoke-078", "title": "Integration Tests for orchestrator-guard", "description": "End-to-end tests for orchestrator-guard workflow", "file": "advanced-enforcement/tickets/goYoke-078.md", "week": 5, "time_estimate": "1.5h", "dependencies": ["goYoke-077"], "priority": "high", "tags": ["orchestrator-guard", "week-5"], "tests_required": true, "acceptance_criteria_count": 7, "status": "pending"},
  {"id": "goYoke-079", "title": "Build goyoke-orchestrator-guard CLI", "description": "Build CLI binary for orchestrator-guard hook", "file": "advanced-enforcement/tickets/goYoke-079.md", "week": 5, "time_estimate": "1.5h", "dependencies": ["goYoke-078"], "priority": "high", "tags": ["orchestrator-guard", "week-5"], "tests_required": true, "acceptance_criteria_count": 6, "status": "pending"},
  {"id": "goYoke-080", "title": "PreToolUse Event Parsing", "description": "Parse PreToolUse events to detect documentation theater", "file": "advanced-enforcement/tickets/goYoke-080.md", "week": 5, "time_estimate": "1.5h", "dependencies": ["goYoke-063"], "priority": "high", "tags": ["doc-theater", "week-5"], "tests_required": true, "acceptance_criteria_count": 6, "status": "pending"},
  {"id": "goYoke-081", "title": "Pattern Detection for Documentation Theater", "description": "Detect patterns indicating documentation theater (markdown writes without implementation)", "file": "advanced-enforcement/tickets/goYoke-081.md", "week": 5, "time_estimate": "2.5h", "dependencies": ["goYoke-080"], "priority": "high", "tags": ["doc-theater", "week-5"], "tests_required": true, "acceptance_criteria_count": 8, "status": "pending"},
  {"id": "goYoke-082", "title": "Integration Tests for doc-theater", "description": "End-to-end tests for doc-theater detection", "file": "advanced-enforcement/tickets/goYoke-082.md", "week": 5, "time_estimate": "1.5h", "dependencies": ["goYoke-081"], "priority": "high", "tags": ["doc-theater", "week-5"], "tests_required": true, "acceptance_criteria_count": 7, "status": "pending"},
  {"id": "goYoke-083", "title": "Build goyoke-doc-theater CLI", "description": "Build CLI binary for doc-theater hook", "file": "advanced-enforcement/tickets/goYoke-083.md", "week": 5, "time_estimate": "1.5h", "dependencies": ["goYoke-082"], "priority": "high", "tags": ["doc-theater", "week-5"], "tests_required": true, "acceptance_criteria_count": 6, "status": "pending"}
]
NEWTICKETS

jq --slurpfile new new-tickets-08.json '
  .tickets = (
    [.tickets[] | select(.id < "goYoke-075")] +
    $new[0] +
    [.tickets[] | select(.id > "goYoke-083" or (.id | test("goYoke-08[4-9]")))]
  ) |
  .metadata.total_tickets = (.tickets | length)
' tickets-index.json > tickets-index-updated.json

cp tickets-index.json tickets-index.json.backup-doc08
mv tickets-index-updated.json tickets-index.json
rm new-tickets-08.json

echo "✓ Updated index with goYoke-075 to 083 (9 tickets)"
