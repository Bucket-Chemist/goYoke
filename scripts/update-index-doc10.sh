#!/bin/bash
set -euo pipefail

cat > new-tickets-10.json << 'NEWTICKETS'
[
  {"id": "GOgent-004c", "title": "Config Circular Dependency Tests", "description": "Test circular dependency detection in config loading (deferred from Week 1)", "file": "integration-tests/tickets/GOgent-004c.md", "week": 5, "time_estimate": "2h", "dependencies": ["GOgent-004"], "priority": "high", "tags": ["config-tests", "week-5", "deferred"], "tests_required": true, "acceptance_criteria_count": 7, "status": "pending"},
  {"id": "GOgent-094", "title": "Test Harness for Event Corpus Replay", "description": "Create test harness to replay hook event corpus for regression testing", "file": "integration-tests/tickets/GOgent-094.md", "week": 5, "time_estimate": "3h", "dependencies": [], "priority": "high", "tags": ["integration-tests", "week-5"], "tests_required": true, "acceptance_criteria_count": 10, "status": "pending"},
  {"id": "GOgent-095", "title": "Integration Tests for validate-routing Hook", "description": "End-to-end tests for validate-routing hook", "file": "integration-tests/tickets/GOgent-095.md", "week": 5, "time_estimate": "2.5h", "dependencies": ["GOgent-094"], "priority": "high", "tags": ["integration-tests", "week-5"], "tests_required": true, "acceptance_criteria_count": 8, "status": "pending"},
  {"id": "GOgent-096", "title": "Integration Tests for session-archive Hook", "description": "End-to-end tests for session-archive hook", "file": "integration-tests/tickets/GOgent-096.md", "week": 5, "time_estimate": "2.5h", "dependencies": ["GOgent-094"], "priority": "high", "tags": ["integration-tests", "week-5"], "tests_required": true, "acceptance_criteria_count": 8, "status": "pending"},
  {"id": "GOgent-097", "title": "Integration Tests for sharp-edge-detector Hook", "description": "End-to-end tests for sharp-edge-detector hook", "file": "integration-tests/tickets/GOgent-097.md", "week": 5, "time_estimate": "2.5h", "dependencies": ["GOgent-094"], "priority": "high", "tags": ["integration-tests", "week-5"], "tests_required": true, "acceptance_criteria_count": 8, "status": "pending"},
  {"id": "GOgent-098", "title": "Performance Benchmarks", "description": "Benchmark Go vs Bash hook performance", "file": "integration-tests/tickets/GOgent-098.md", "week": 5, "time_estimate": "3h", "dependencies": ["GOgent-090"], "priority": "high", "tags": ["performance", "week-5"], "tests_required": true, "acceptance_criteria_count": 10, "status": "pending"},
  {"id": "GOgent-099", "title": "End-to-End Workflow Integration Tests", "description": "Full workflow integration tests across all hooks", "file": "integration-tests/tickets/GOgent-099.md", "week": 5, "time_estimate": "3h", "dependencies": ["GOgent-095", "GOgent-096", "GOgent-097"], "priority": "high", "tags": ["integration-tests", "week-5"], "tests_required": true, "acceptance_criteria_count": 9, "status": "pending"},
  {"id": "GOgent-100", "title": "Regression Tests (Go vs Bash Comparison)", "description": "Regression testing to ensure Go hooks match Bash behavior", "file": "integration-tests/tickets/GOgent-100.md", "week": 5, "time_estimate": "3h", "dependencies": ["GOgent-099"], "priority": "high", "tags": ["performance", "week-5"], "tests_required": true, "acceptance_criteria_count": 10, "status": "pending"}
]
NEWTICKETS

jq --slurpfile new new-tickets-10.json '
  .tickets = (
    [.tickets[] | select(.id < "GOgent-004c" and .id != "GOgent-004")] +
    [.tickets[] | select(.id == "GOgent-004")] +
    [$new[0][0]] +
    [.tickets[] | select(.id > "GOgent-004c" and .id < "GOgent-094")] +
    ($new[0][1:]) +
    [.tickets[] | select(.id > "GOgent-100")]
  ) |
  .metadata.total_tickets = (.tickets | length)
' tickets-index.json > tickets-index-updated.json

cp tickets-index.json tickets-index.json.backup-doc10
mv tickets-index-updated.json tickets-index.json
rm new-tickets-10.json

echo "✓ Updated index with GOgent-004c and GOgent-094 to 100 (8 tickets)"
