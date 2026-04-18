#!/bin/bash
set -euo pipefail

cat > new-tickets-09.json << 'NEWTICKETS'
[
  {"id": "goYoke-087", "title": "PostToolUse Event for Benchmarking", "description": "Parse PostToolUse events to capture tool metrics", "file": "observability-remaining/tickets/goYoke-087.md", "week": 4, "time_estimate": "1.5h", "dependencies": ["goYoke-070"], "priority": "high", "tags": ["benchmark-logger", "week-4"], "tests_required": true, "acceptance_criteria_count": 6, "status": "pending"},
  {"id": "goYoke-088", "title": "Benchmark Metrics Logging", "description": "Log tool usage metrics for performance analysis", "file": "observability-remaining/tickets/goYoke-088.md", "week": 4, "time_estimate": "2h", "dependencies": ["goYoke-087"], "priority": "high", "tags": ["benchmark-logger", "week-4"], "tests_required": true, "acceptance_criteria_count": 8, "status": "pending"},
  {"id": "goYoke-089", "title": "Integration Tests for benchmark-logger", "description": "End-to-end tests for benchmark-logger workflow", "file": "observability-remaining/tickets/goYoke-089.md", "week": 4, "time_estimate": "1.5h", "dependencies": ["goYoke-088"], "priority": "high", "tags": ["benchmark-logger", "week-4"], "tests_required": true, "acceptance_criteria_count": 7, "status": "pending"},
  {"id": "goYoke-090", "title": "Build goyoke-benchmark-logger CLI", "description": "Build CLI binary for benchmark-logger hook", "file": "observability-remaining/tickets/goYoke-090.md", "week": 4, "time_estimate": "1h", "dependencies": ["goYoke-089"], "priority": "high", "tags": ["benchmark-logger", "week-4"], "tests_required": true, "acceptance_criteria_count": 5, "status": "pending"},
  {"id": "goYoke-091", "title": "Investigate stop-gate.sh Purpose", "description": "Analyze stop-gate.sh hook to determine purpose and usage", "file": "observability-remaining/tickets/goYoke-091.md", "week": 4, "time_estimate": "1h", "dependencies": [], "priority": "high", "tags": ["stop-gate", "week-4"], "tests_required": true, "acceptance_criteria_count": 4, "status": "pending"},
  {"id": "goYoke-092", "title": "Stop-Gate Translation or Deprecation", "description": "Either translate stop-gate or document deprecation", "file": "observability-remaining/tickets/goYoke-092.md", "week": 4, "time_estimate": "0.5h", "dependencies": ["goYoke-091"], "priority": "high", "tags": ["stop-gate", "week-4"], "tests_required": true, "acceptance_criteria_count": 3, "status": "pending"},
  {"id": "goYoke-093", "title": "Final Documentation & Status Report", "description": "Generate migration status report and update documentation", "file": "observability-remaining/tickets/goYoke-093.md", "week": 4, "time_estimate": "2h", "dependencies": ["goYoke-090", "goYoke-092"], "priority": "high", "tags": ["observability", "week-4"], "tests_required": true, "acceptance_criteria_count": 8, "status": "pending"}
]
NEWTICKETS

jq --slurpfile new new-tickets-09.json '
  .tickets = (
    [.tickets[] | select(.id < "goYoke-087")] +
    $new[0] +
    [.tickets[] | select(.id > "goYoke-093")]
  ) |
  .metadata.total_tickets = (.tickets | length)
' tickets-index.json > tickets-index-updated.json

cp tickets-index.json tickets-index.json.backup-doc09
mv tickets-index-updated.json tickets-index.json
rm new-tickets-09.json

echo "✓ Updated index with goYoke-087 to 093 (7 tickets)"
