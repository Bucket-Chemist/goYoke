#!/usr/bin/env python3
"""Fix id + subagent_type in all agent frontmatter files.

Uses line-based frontmatter manipulation (no YAML dependency).
Source of truth: routing-schema.json agent_subagent_mapping.
"""
import json
import os
import sys
from pathlib import Path

HOME = Path.home()
AGENTS_DIR = HOME / ".claude" / "agents"
SCHEMA = HOME / ".claude" / "routing-schema.json"

def load_mapping():
    with open(SCHEMA) as f:
        schema = json.load(f)
    mapping = schema.get("agent_subagent_mapping", {})
    mapping.pop("description", None)
    return mapping

def parse_frontmatter(lines):
    """Find frontmatter boundaries. Returns (start, end) line indices."""
    dashes = []
    for i, line in enumerate(lines):
        if line.strip() == "---":
            dashes.append(i)
        if len(dashes) == 2:
            break
    if len(dashes) < 2:
        return None, None
    return dashes[0], dashes[1]

def get_fm_field(lines, start, end, field):
    """Get value of a simple field in frontmatter (single line, no nesting)."""
    prefix = f"{field}:"
    for i in range(start + 1, end):
        stripped = lines[i].strip()
        if stripped.startswith(prefix):
            val = stripped[len(prefix):].strip()
            return i, val
    return None, None

def fix_file(agent_id, expected_sat):
    filepath = AGENTS_DIR / agent_id / f"{agent_id}.md"
    if not filepath.exists():
        return f"  SKIP: {filepath} not found"

    with open(filepath, "r") as f:
        lines = f.readlines()

    start, end = parse_frontmatter(lines)
    if start is None:
        return f"  SKIP: {agent_id} has no frontmatter"

    changes = []

    # --- Fix id ---
    id_line, id_val = get_fm_field(lines, start, end, "id")
    if id_line is None:
        # Insert id as first line after opening ---
        lines.insert(start + 1, f"id: {agent_id}\n")
        end += 1  # Shift end marker
        changes.append(f"added id")
    elif id_val != agent_id:
        lines[id_line] = f"id: {agent_id}\n"
        changes.append(f"fixed id ({id_val} -> {agent_id})")

    # --- Fix subagent_type ---
    sat_line, sat_val = get_fm_field(lines, start, end, "subagent_type")
    if sat_line is None:
        # Find best insertion point: after category, or after tier, or after model
        insert_after = None
        for field in ["category", "tier", "model"]:
            idx, _ = get_fm_field(lines, start, end, field)
            if idx is not None:
                insert_after = idx
                break
        if insert_after is not None:
            lines.insert(insert_after + 1, f"subagent_type: {expected_sat}\n")
            end += 1
            changes.append(f"added subagent_type: {expected_sat}")
        else:
            # Last resort: insert before closing ---
            lines.insert(end, f"subagent_type: {expected_sat}\n")
            changes.append(f"added subagent_type: {expected_sat} (before closing ---)")
    elif sat_val != expected_sat:
        lines[sat_line] = f"subagent_type: {expected_sat}\n"
        changes.append(f"fixed subagent_type ({sat_val} -> {expected_sat})")

    if changes:
        with open(filepath, "w") as f:
            f.writelines(lines)
        return f"  {agent_id}: {', '.join(changes)}"
    else:
        return f"  {agent_id}: OK"

def main():
    mapping = load_mapping()
    print(f"Loaded {len(mapping)} agents from routing-schema mapping\n")

    for agent_id in sorted(mapping.keys()):
        expected_sat = mapping[agent_id]
        result = fix_file(agent_id, expected_sat)
        print(result)

    # Verification
    print("\n=== Verification ===")
    failures = 0
    for agent_id in sorted(mapping.keys()):
        expected_sat = mapping[agent_id]
        filepath = AGENTS_DIR / agent_id / f"{agent_id}.md"
        if not filepath.exists():
            continue

        with open(filepath) as f:
            lines = f.readlines()
        start, end = parse_frontmatter(lines)
        if start is None:
            continue

        _, id_val = get_fm_field(lines, start, end, "id")
        _, sat_val = get_fm_field(lines, start, end, "subagent_type")

        issues = []
        if id_val != agent_id:
            issues.append(f"id='{id_val}'(want '{agent_id}')")
        if sat_val != expected_sat:
            issues.append(f"sat='{sat_val}'(want '{expected_sat}')")
        if issues:
            print(f"  FAIL: {agent_id} - {' '.join(issues)}")
            failures += 1

    if failures == 0:
        print("  PASS: All frontmatter id + subagent_type aligned")
    else:
        print(f"\n  {failures} agents still misaligned")
    return failures

if __name__ == "__main__":
    sys.exit(main())
