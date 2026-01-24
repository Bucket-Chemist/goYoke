#!/bin/bash
set -euo pipefail

cat > new-tickets-09.json << 'NEWTICKETS'
[
  {"id": "GOgent-087", "title": "PostToolUse Event for Benchmarking", "description": "Parse PostToolUse events to capture tool metrics", "file": "observability-remaining/tickets/GOgent-087.md", "week": 4, "time_estimate": "1.5h", "dependencies": ["GOgent-070"], "priority": "high", "tags": ["benchmark-logger", "week-4"], "tests_required": true, "acceptance_criteria_count": 6, "status": "pending"},
  {"id": "GOgent-088", "title": "Benchmark Metrics Logging", "description": "Log tool usage metrics for performance analysis", "file": "observability-remaining/tickets/GOgent-088.md", "week": 4, "time_estimate": "2h", "dependencies": ["GOgent-087"], "priority": "high", "tags": ["benchmark-logger", "week-4"], "tests_required": true, "acceptance_criteria_count": 8, "status": "pending"},
  {"id": "GOgent-089", "title": "Integration Tests for benchmark-logger", "description": "End-to-end tests for benchmark-logger workflow", "file": "observability-remaining/tickets/GOgent-089.md", "week": 4, "time_estimate": "1.5h", "dependencies": ["GOgent-088"], "priority": "high", "tags": ["benchmark-logger", "week-4"], "tests_required": true, "acceptance_criteria_count": 7, "status": "pending"},
  {"id": "GOgent-090", "title": "Build gogent-benchmark-logger CLI", "description": "Build CLI binary for benchmark-logger hook", "file": "observability-remaining/tickets/GOgent-090.md", "week": 4, "time_estimate": "1h", "dependencies": ["GOgent-089"], "priority": "high", "tags": ["benchmark-logger", "week-4"], "tests_required": true, "acceptance_criteria_count": 5, "status": "pending"},
  {"id": "GOgent-091", "title": "Investigate stop-gate.sh Purpose", "description": "Analyze stop-gate.sh hook to determine purpose and usage", "file": "observability-remaining/tickets/GOgent-091.md", "week": 4, "time_estimate": "1h", "dependencies": [], "priority": "high", "tags": ["stop-gate", "week-4"], "tests_required": true, "acceptance_criteria_count": 4, "status": "pending"},
  {"id": "GOgent-092", "title": "Stop-Gate Translation or Deprecation", "description": "Either translate stop-gate or document deprecation", "file": "observability-remaining/tickets/GOgent-092.md", "week": 4, "time_estimate": "0.5h", "dependencies": ["GOgent-091"], "priority": "high", "tags": ["stop-gate", "week-4"], "tests_required": true, "acceptance_criteria_count": 3, "status": "pending"},
  {"id": "GOgent-093", "title": "Final Documentation & Status Report", "description": "Generate migration status report and update documentation", "file": "observability-remaining/tickets/GOgent-093.md", "week": 4, "time_estimate": "2h", "dependencies": ["GOgent-090", "GOgent-092"], "priority": "high", "tags": ["observability", "week-4"], "tests_required": true, "acceptance_criteria_count": 8, "status": "pending"}
]
NEWTICKETS

jq --slurpfile new new-tickets-09.json '
  .tickets = (
    [.tickets[] | select(.id < "GOgent-087")] +
    $new[0] +
    [.tickets[] | select(.id > "GOgent-093")]
  ) |
  .metadata.total_tickets = (.tickets | length)
' tickets-index.json > tickets-index-updated.json

cp tickets-index.json tickets-index.json.backup-doc09
mv tickets-index-updated.json tickets-index.json
rm new-tickets-09.json

echo "✓ Updated index with GOgent-087 to 093 (7 tickets)"
