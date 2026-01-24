#!/bin/bash
set -euo pipefail

cat > new-tickets-08.json << 'NEWTICKETS'
[
  {"id": "GOgent-075", "title": "SubagentStop Event Parsing for Orchestrator", "description": "Detect orchestrator/architect completion events", "file": "advanced-enforcement/tickets/GOgent-075.md", "week": 5, "time_estimate": "1.5h", "dependencies": ["GOgent-063"], "priority": "high", "tags": ["orchestrator-guard", "week-5"], "tests_required": true, "acceptance_criteria_count": 6, "status": "pending"},
  {"id": "GOgent-076", "title": "Transcript Analysis & Task Tracking", "description": "Scan transcripts for background tasks and TaskOutput collections", "file": "advanced-enforcement/tickets/GOgent-076.md", "week": 5, "time_estimate": "2.5h", "dependencies": ["GOgent-075"], "priority": "high", "tags": ["orchestrator-guard", "week-5"], "tests_required": true, "acceptance_criteria_count": 8, "status": "pending"},
  {"id": "GOgent-077", "title": "Blocking Response Generation", "description": "Generate blocking response if background tasks uncollected", "file": "advanced-enforcement/tickets/GOgent-077.md", "week": 5, "time_estimate": "2h", "dependencies": ["GOgent-076"], "priority": "high", "tags": ["orchestrator-guard", "week-5"], "tests_required": true, "acceptance_criteria_count": 7, "status": "pending"},
  {"id": "GOgent-078", "title": "Integration Tests for orchestrator-guard", "description": "End-to-end tests for orchestrator-guard workflow", "file": "advanced-enforcement/tickets/GOgent-078.md", "week": 5, "time_estimate": "1.5h", "dependencies": ["GOgent-077"], "priority": "high", "tags": ["orchestrator-guard", "week-5"], "tests_required": true, "acceptance_criteria_count": 7, "status": "pending"},
  {"id": "GOgent-079", "title": "Build gogent-orchestrator-guard CLI", "description": "Build CLI binary for orchestrator-guard hook", "file": "advanced-enforcement/tickets/GOgent-079.md", "week": 5, "time_estimate": "1.5h", "dependencies": ["GOgent-078"], "priority": "high", "tags": ["orchestrator-guard", "week-5"], "tests_required": true, "acceptance_criteria_count": 6, "status": "pending"},
  {"id": "GOgent-080", "title": "PreToolUse Event Parsing", "description": "Parse PreToolUse events to detect documentation theater", "file": "advanced-enforcement/tickets/GOgent-080.md", "week": 5, "time_estimate": "1.5h", "dependencies": ["GOgent-063"], "priority": "high", "tags": ["doc-theater", "week-5"], "tests_required": true, "acceptance_criteria_count": 6, "status": "pending"},
  {"id": "GOgent-081", "title": "Pattern Detection for Documentation Theater", "description": "Detect patterns indicating documentation theater (markdown writes without implementation)", "file": "advanced-enforcement/tickets/GOgent-081.md", "week": 5, "time_estimate": "2.5h", "dependencies": ["GOgent-080"], "priority": "high", "tags": ["doc-theater", "week-5"], "tests_required": true, "acceptance_criteria_count": 8, "status": "pending"},
  {"id": "GOgent-082", "title": "Integration Tests for doc-theater", "description": "End-to-end tests for doc-theater detection", "file": "advanced-enforcement/tickets/GOgent-082.md", "week": 5, "time_estimate": "1.5h", "dependencies": ["GOgent-081"], "priority": "high", "tags": ["doc-theater", "week-5"], "tests_required": true, "acceptance_criteria_count": 7, "status": "pending"},
  {"id": "GOgent-083", "title": "Build gogent-doc-theater CLI", "description": "Build CLI binary for doc-theater hook", "file": "advanced-enforcement/tickets/GOgent-083.md", "week": 5, "time_estimate": "1.5h", "dependencies": ["GOgent-082"], "priority": "high", "tags": ["doc-theater", "week-5"], "tests_required": true, "acceptance_criteria_count": 6, "status": "pending"}
]
NEWTICKETS

jq --slurpfile new new-tickets-08.json '
  .tickets = (
    [.tickets[] | select(.id < "GOgent-075")] +
    $new[0] +
    [.tickets[] | select(.id > "GOgent-083" or (.id | test("GOgent-08[4-9]")))]
  ) |
  .metadata.total_tickets = (.tickets | length)
' tickets-index.json > tickets-index-updated.json

cp tickets-index.json tickets-index.json.backup-doc08
mv tickets-index-updated.json tickets-index.json
rm new-tickets-08.json

echo "✓ Updated index with GOgent-075 to 083 (9 tickets)"
