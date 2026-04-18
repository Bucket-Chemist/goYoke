#!/bin/bash
set -euo pipefail

cat > new-tickets-11.json << 'NEWTICKETS'
[
  {"id": "goYoke-101", "title": "Installation Script", "description": "Create install.sh to automate CLI installation to ~/.local/bin", "file": "deployment-cutover/tickets/goYoke-101.md", "week": 5, "time_estimate": "3h", "dependencies": ["goYoke-100"], "priority": "high", "tags": ["installation", "week-5"], "tests_required": true, "acceptance_criteria_count": 10, "status": "pending"},
  {"id": "goYoke-101b", "title": "WSL2 Compatibility Testing", "description": "Test all CLIs on WSL2 environment", "file": "deployment-cutover/tickets/goYoke-101b.md", "week": 5, "time_estimate": "2h", "dependencies": ["goYoke-101"], "priority": "high", "tags": ["wsl2", "week-5"], "tests_required": true, "acceptance_criteria_count": 7, "status": "pending"},
  {"id": "goYoke-102", "title": "Parallel Testing Script", "description": "Create script to test all hooks in parallel", "file": "deployment-cutover/tickets/goYoke-102.md", "week": 5, "time_estimate": "2h", "dependencies": ["goYoke-101"], "priority": "high", "tags": ["testing", "week-5"], "tests_required": true, "acceptance_criteria_count": 7, "status": "pending"},
  {"id": "goYoke-103", "title": "Cutover Decision Workflow", "description": "Decision framework and checklist for Go cutover", "file": "deployment-cutover/tickets/goYoke-103.md", "week": 5, "time_estimate": "3h", "dependencies": ["goYoke-102", "goYoke-098"], "priority": "high", "tags": ["cutover", "week-5"], "tests_required": true, "acceptance_criteria_count": 9, "status": "pending"},
  {"id": "goYoke-104", "title": "Symlink Cutover Script", "description": "Script to atomically switch hooks from Bash to Go", "file": "deployment-cutover/tickets/goYoke-104.md", "week": 5, "time_estimate": "2h", "dependencies": ["goYoke-103"], "priority": "high", "tags": ["cutover", "week-5"], "tests_required": true, "acceptance_criteria_count": 6, "status": "pending"},
  {"id": "goYoke-105", "title": "Rollback Script and Testing", "description": "Rollback mechanism to revert to Bash hooks if issues arise", "file": "deployment-cutover/tickets/goYoke-105.md", "week": 5, "time_estimate": "2.5h", "dependencies": ["goYoke-104"], "priority": "high", "tags": ["cutover", "week-5"], "tests_required": true, "acceptance_criteria_count": 8, "status": "pending"},
  {"id": "goYoke-106", "title": "Documentation Updates", "description": "Update all documentation to reflect Go implementation", "file": "deployment-cutover/tickets/goYoke-106.md", "week": 5, "time_estimate": "4h", "dependencies": ["goYoke-104"], "priority": "high", "tags": ["deployment", "week-5"], "tests_required": true, "acceptance_criteria_count": 12, "status": "pending"},
  {"id": "goYoke-107", "title": "Performance Regression Monitoring", "description": "Set up ongoing performance monitoring post-cutover", "file": "deployment-cutover/tickets/goYoke-107.md", "week": 5, "time_estimate": "2h", "dependencies": ["goYoke-104"], "priority": "high", "tags": ["deployment", "week-5"], "tests_required": true, "acceptance_criteria_count": 7, "status": "pending"},
  {"id": "goYoke-108", "title": "Post-Cutover Validation Checklist", "description": "Comprehensive validation checklist for post-cutover verification", "file": "deployment-cutover/tickets/goYoke-108.md", "week": 5, "time_estimate": "3h", "dependencies": ["goYoke-106", "goYoke-107"], "priority": "high", "tags": ["deployment", "week-5"], "tests_required": true, "acceptance_criteria_count": 11, "status": "pending"}
]
NEWTICKETS

jq --slurpfile new new-tickets-11.json '
  .tickets = (
    [.tickets[] | select(.id < "goYoke-101")] +
    $new[0] +
    [.tickets[] | select(.id > "goYoke-108")]
  ) |
  .metadata.total_tickets = (.tickets | length) |
  .metadata.note = "Complete migration plan: agent-workflow-hooks (063-072), advanced-enforcement (075-083), observability (087-093), integration-tests (004c, 094-100), deployment (101-108)"
' tickets-index.json > tickets-index-updated.json

cp tickets-index.json tickets-index.json.backup-doc11
mv tickets-index-updated.json tickets-index.json
rm new-tickets-11.json

echo "✓ Updated index with goYoke-101, 101b, 102-108 (9 tickets)"
echo "✓ Final total tickets: $(jq '.metadata.total_tickets' tickets-index.json)"
