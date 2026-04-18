#!/bin/bash
set -euo pipefail

# Generate JSON entries for new tickets
echo "Generating new ticket entries..."

cat > new-tickets.json << 'NEWTICKETS'
[
  {
    "id": "goYoke-063",
    "title": "Define SubagentStop Event Structs",
    "description": "Parse SubagentStop events and detect agent completion type",
    "file": "agent-workflow-hooks/tickets/goYoke-063.md",
    "week": 4,
    "time_estimate": "1.5h",
    "dependencies": ["goYoke-056"],
    "priority": "high",
    "tags": ["agent-endstate", "week-4"],
    "tests_required": true,
    "acceptance_criteria_count": 7,
    "status": "pending"
  },
  {
    "id": "goYoke-064",
    "title": "Tier-Specific Response Generation",
    "description": "Generate appropriate follow-up responses based on agent class and tier",
    "file": "agent-workflow-hooks/tickets/goYoke-064.md",
    "week": 4,
    "time_estimate": "2h",
    "dependencies": ["goYoke-063"],
    "priority": "high",
    "tags": ["agent-endstate", "week-4"],
    "tests_required": true,
    "acceptance_criteria_count": 9,
    "status": "pending"
  },
  {
    "id": "goYoke-065",
    "title": "Endstate Logging & Decision Storage",
    "description": "Store endstate decisions in JSONL format for analysis and audit trail",
    "file": "agent-workflow-hooks/tickets/goYoke-065.md",
    "week": 4,
    "time_estimate": "1.5h",
    "dependencies": ["goYoke-064"],
    "priority": "high",
    "tags": ["agent-endstate", "week-4"],
    "tests_required": true,
    "acceptance_criteria_count": 8,
    "status": "pending"
  },
  {
    "id": "goYoke-066",
    "title": "Integration Tests for agent-endstate",
    "description": "Comprehensive tests covering event parsing → response generation → logging workflow",
    "file": "agent-workflow-hooks/tickets/goYoke-066.md",
    "week": 4,
    "time_estimate": "1.5h",
    "dependencies": ["goYoke-065"],
    "priority": "high",
    "tags": ["agent-endstate", "week-4"],
    "tests_required": true,
    "acceptance_criteria_count": 6,
    "status": "pending"
  },
  {
    "id": "goYoke-067",
    "title": "Build goyoke-agent-endstate CLI",
    "description": "Build CLI binary that reads SubagentStop events and generates follow-up responses",
    "file": "agent-workflow-hooks/tickets/goYoke-067.md",
    "week": 4,
    "time_estimate": "1.5h",
    "dependencies": ["goYoke-066"],
    "priority": "high",
    "tags": ["agent-endstate", "week-4"],
    "tests_required": true,
    "acceptance_criteria_count": 7,
    "status": "pending"
  },
  {
    "id": "goYoke-068",
    "title": "Tool Counter Management",
    "description": "Manage persistent tool call counter for attention-gate triggering",
    "file": "agent-workflow-hooks/tickets/goYoke-068.md",
    "week": 4,
    "time_estimate": "1.5h",
    "dependencies": ["goYoke-056"],
    "priority": "high",
    "tags": ["attention-gate", "week-4"],
    "tests_required": true,
    "acceptance_criteria_count": 8,
    "status": "pending"
  },
  {
    "id": "goYoke-069",
    "title": "Reminder & Flush Logic",
    "description": "Generate routing compliance reminders and auto-flush pending learnings",
    "file": "agent-workflow-hooks/tickets/goYoke-069.md",
    "week": 4,
    "time_estimate": "2h",
    "dependencies": ["goYoke-068"],
    "priority": "high",
    "tags": ["attention-gate", "week-4"],
    "tests_required": true,
    "acceptance_criteria_count": 8,
    "status": "pending"
  },
  {
    "id": "goYoke-070",
    "title": "PostToolUse Event Parsing",
    "description": "Parse PostToolUse events that trigger attention-gate",
    "file": "agent-workflow-hooks/tickets/goYoke-070.md",
    "week": 4,
    "time_estimate": "1.5h",
    "dependencies": ["goYoke-056"],
    "priority": "high",
    "tags": ["attention-gate", "week-4"],
    "tests_required": true,
    "acceptance_criteria_count": 5,
    "status": "pending"
  },
  {
    "id": "goYoke-071",
    "title": "Integration Tests for attention-gate",
    "description": "End-to-end tests for tool counter → reminder/flush workflow",
    "file": "agent-workflow-hooks/tickets/goYoke-071.md",
    "week": 4,
    "time_estimate": "1.5h",
    "dependencies": ["goYoke-070"],
    "priority": "high",
    "tags": ["attention-gate", "week-4"],
    "tests_required": true,
    "acceptance_criteria_count": 8,
    "status": "pending"
  },
  {
    "id": "goYoke-072",
    "title": "Build goyoke-attention-gate CLI",
    "description": "Build CLI binary for attention-gate hook",
    "file": "agent-workflow-hooks/tickets/goYoke-072.md",
    "week": 4,
    "time_estimate": "1.5h",
    "dependencies": ["goYoke-071"],
    "priority": "high",
    "tags": ["attention-gate", "week-4"],
    "tests_required": true,
    "acceptance_criteria_count": 7,
    "status": "pending"
  }
]
NEWTICKETS

# Update the index
echo "Updating tickets-index.json..."

# Remove old 063-070 entries and insert new 063-072 entries
jq --slurpfile new new-tickets.json '
  .tickets = (
    [.tickets[] | select(.id < "goYoke-063")] +
    $new[0] +
    [.tickets[] | select(.id > "goYoke-070")]
  ) |
  .metadata.total_tickets = (.tickets | length) |
  .metadata.note = "Includes agent-workflow-hooks tickets (063-072): agent-endstate and attention-gate hooks"
' tickets-index.json > tickets-index-updated.json

# Backup and replace
cp tickets-index.json tickets-index.json.backup
mv tickets-index-updated.json tickets-index.json

echo "✓ Updated tickets-index.json (backup: tickets-index.json.backup)"
echo "✓ Added 10 new tickets (goYoke-063 to goYoke-072)"

# Cleanup
rm new-tickets.json
