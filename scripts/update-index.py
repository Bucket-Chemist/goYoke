#!/usr/bin/env python3
"""Update tickets-index.json with extracted session-init-context tickets."""

import json
import re
from pathlib import Path

# Ticket metadata from source document analysis
TICKETS = [
    {"id": "goYoke-056", "title": "SessionStart Event Struct & Parser", "time": "1h", "deps": [], "files": ["pkg/session/events.go", "pkg/session/session_start_test.go"], "priority": "high", "ac_count": 16},
    {"id": "goYoke-057", "title": "Tool Counter Initialization", "time": "0.5h", "deps": ["goYoke-056"], "files": ["pkg/config/paths.go", "pkg/config/paths_test.go"], "priority": "high", "ac_count": 14},
    {"id": "goYoke-058", "title": "Routing Schema Summary Formatter", "time": "1h", "deps": [], "files": ["pkg/routing/schema.go", "pkg/routing/schema_test.go"], "priority": "medium", "ac_count": 13},
    {"id": "goYoke-059", "title": "Handoff Document Loader", "time": "1h", "deps": [], "files": ["pkg/session/context_loader.go", "pkg/session/context_loader_test.go"], "priority": "medium", "ac_count": 17},
    {"id": "goYoke-060", "title": "Project Type Detection", "time": "1.5h", "deps": [], "files": ["pkg/session/project_detection.go", "pkg/session/project_detection_test.go"], "priority": "medium", "ac_count": 22},
    {"id": "goYoke-061", "title": "Session Context Response Generator", "time": "1.5h", "deps": ["goYoke-056", "goYoke-058", "goYoke-059", "goYoke-060"], "files": ["pkg/session/context_response.go", "pkg/session/context_response_test.go"], "priority": "high", "ac_count": 15},
    {"id": "goYoke-062", "title": "CLI Binary - Main Orchestrator", "time": "1.5h", "deps": ["goYoke-056", "goYoke-057", "goYoke-061"], "files": ["cmd/goyoke-load-context/main.go", "cmd/goyoke-load-context/main_test.go"], "priority": "high", "ac_count": 16},
    {"id": "goYoke-063", "title": "Integration Tests", "time": "1h", "deps": ["goYoke-062"], "files": ["test/integration/session_start_test.go"], "priority": "medium", "ac_count": 10},
    {"id": "goYoke-064", "title": "Makefile Updates", "time": "0.5h", "deps": ["goYoke-062"], "files": ["Makefile"], "priority": "low", "ac_count": 3},
    {"id": "goYoke-065", "title": "Documentation Update", "time": "1h", "deps": ["goYoke-064"], "files": ["docs/systems-architecture-overview.md"], "priority": "low", "ac_count": 4},
    {"id": "goYoke-066", "title": "Hook Configuration Template", "time": "0.5h", "deps": ["goYoke-064"], "files": ["docs/hook-configuration.md"], "priority": "medium", "ac_count": 5},
    {"id": "goYoke-067", "title": "Extend Runner with SessionStart Category", "time": "1.5h", "deps": ["goYoke-062"], "files": ["harness/runner.go", "harness/runner_test.go"], "priority": "medium", "ac_count": 13},
    {"id": "goYoke-068", "title": "Create SessionStart Test Fixtures", "time": "2h", "deps": ["goYoke-067"], "files": ["fixtures/sessionstart/*.json"], "priority": "medium", "ac_count": 12},
    {"id": "goYoke-069", "title": "Update Harness CLI for SessionStart", "time": "1h", "deps": ["goYoke-067"], "files": ["cmd/harness/main.go"], "priority": "low", "ac_count": 6},
    {"id": "goYoke-070", "title": "GitHub Actions Workflow Update", "time": "1.5h", "deps": ["goYoke-068", "goYoke-069"], "files": [".github/workflows/*.yml"], "priority": "low", "ac_count": 8},
]

def calculate_blocks(tickets):
    """Build blocks relationships from dependencies."""
    blocks_map = {t["id"]: [] for t in tickets}

    for ticket in tickets:
        for dep in ticket["deps"]:
            if dep in blocks_map:
                blocks_map[dep].append(ticket["id"])

    return blocks_map

def main():
    # Read existing index
    index_path = Path("../tickets-index.json")
    with open(index_path) as f:
        index = json.load(f)

    # Calculate blocks relationships
    blocks_map = calculate_blocks(TICKETS)

    # Add new tickets to index
    for ticket_meta in TICKETS:
        ticket_entry = {
            "id": ticket_meta["id"],
            "title": ticket_meta["title"],
            "description": ticket_meta["title"],  # Will be refined from markdown
            "file": f"session-init-context/tickets/{ticket_meta['id']}.md",
            "week": 4,
            "time_estimate": ticket_meta["time"],
            "dependencies": ticket_meta["deps"],
            "blocks": blocks_map[ticket_meta["id"]],
            "priority": ticket_meta["priority"],
            "tags": ["session-start", "week-4", "phase-0"],
            "files_to_create": ticket_meta["files"],
            "tests_required": True,
            "git_branch": f"{ticket_meta['id'].lower()}-{ticket_meta['title'].lower().replace(' ', '-')}",
            "pr_labels": ["session-start", "week-4", "phase-0"],
            "acceptance_criteria_count": ticket_meta["ac_count"],
            "status": "pending"
        }

        index["tickets"].append(ticket_entry)

    # Update metadata
    index["metadata"]["total_tickets"] = 127  # Was 112, adding 15
    index["metadata"]["note"] += " + 15 session-init-context tickets (056-070)"

    # Write updated index
    with open(index_path, 'w') as f:
        json.dump(index, f, indent=2)

    print(f"✓ Added {len(TICKETS)} tickets to index")
    print(f"✓ Updated total_tickets: 112 → 127")
    print(f"✓ Updated metadata note")

if __name__ == "__main__":
    main()
