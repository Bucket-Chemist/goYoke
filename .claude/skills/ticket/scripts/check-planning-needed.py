#!/home/doktersmol/.generic-python/bin/python3
"""
Check if a ticket requires architect planning based on complexity signals.

Priority order:
1. Explicit frontmatter field (needs_planning: true/false)
2. Tag presence ("planning" tag)
3. Complexity signals (files, time, dependencies, packages)
4. Default: false
"""

import sys
import json
from pathlib import Path
from typing import Dict, Any, Tuple
import re

try:
    import frontmatter
except ImportError:
    print(json.dumps({"error": "python-frontmatter not installed. Run: pip install python-frontmatter"}), file=sys.stderr)
    sys.exit(1)


def parse_time_estimate(time_str: str) -> float:
    """
    Convert time estimate string to hours.

    Args:
        time_str: Time string like "2h", "30m", "1.5h"

    Returns:
        Hours as float
    """
    if not time_str:
        return 0.0

    # Match patterns like "2h", "30m", "1.5h"
    hour_match = re.match(r'(\d+\.?\d*)h', time_str.lower())
    minute_match = re.match(r'(\d+)m', time_str.lower())

    if hour_match:
        return float(hour_match.group(1))
    elif minute_match:
        return float(minute_match.group(1)) / 60.0

    return 0.0


def count_unique_packages(files_to_create: list) -> int:
    """
    Count unique package paths from files_to_create list.

    Args:
        files_to_create: List of file paths

    Returns:
        Number of unique package directories
    """
    if not files_to_create:
        return 0

    packages = set()
    for file_path in files_to_create:
        # Extract package pattern: pkg/*/
        match = re.search(r'pkg/([^/]+)/', str(file_path))
        if match:
            packages.add(match.group(1))

    return len(packages)


def calculate_complexity_score(metadata: Dict[str, Any]) -> Tuple[int, list]:
    """
    Calculate complexity score based on multiple signals.

    Args:
        metadata: Ticket frontmatter metadata

    Returns:
        Tuple of (score, reasons list)
    """
    score = 0
    reasons = []

    # Signal 1: files_to_create > 3
    files_to_create = metadata.get('files_to_create', [])
    if isinstance(files_to_create, list) and len(files_to_create) > 3:
        score += 1
        reasons.append(f"files_to_create={len(files_to_create)} (>3)")

    # Signal 2: time_estimate > 2h
    time_estimate = metadata.get('time_estimate', '')
    hours = parse_time_estimate(time_estimate)
    if hours > 2.0:
        score += 1
        reasons.append(f"time_estimate={hours}h (>2h)")

    # Signal 3: dependencies > 2
    dependencies = metadata.get('dependencies', [])
    if isinstance(dependencies, list) and len(dependencies) > 2:
        score += 1
        reasons.append(f"dependencies={len(dependencies)} (>2)")

    # Signal 4: multiple packages
    num_packages = count_unique_packages(files_to_create)
    if num_packages > 1:
        score += 1
        reasons.append(f"packages={num_packages} (>1)")

    return score, reasons


def check_planning_needed(ticket_file: str) -> Dict[str, Any]:
    """
    Determine if ticket needs planning based on priority rules.

    Args:
        ticket_file: Path to ticket markdown file

    Returns:
        Dict with needs_planning, reason, confidence
    """
    ticket_path = Path(ticket_file)

    if not ticket_path.exists():
        return {
            "error": f"Ticket file not found: {ticket_file}"
        }

    try:
        with open(ticket_path, 'r', encoding='utf-8') as f:
            post = frontmatter.load(f)
    except Exception as e:
        return {
            "error": f"Failed to parse ticket: {str(e)}"
        }

    metadata = post.metadata

    # Priority 1: Explicit frontmatter field
    if 'needs_planning' in metadata:
        needs_planning = bool(metadata['needs_planning'])
        return {
            "needs_planning": needs_planning,
            "reason": f"Explicit frontmatter field: needs_planning={needs_planning}",
            "confidence": "explicit"
        }

    # Priority 2: Tag presence
    tags = metadata.get('tags', [])
    if isinstance(tags, list) and 'planning' in tags:
        return {
            "needs_planning": True,
            "reason": "Tag 'planning' found in frontmatter",
            "confidence": "heuristic"
        }

    # Priority 3: Complexity signals
    complexity_score, complexity_reasons = calculate_complexity_score(metadata)

    if complexity_score >= 2:
        return {
            "needs_planning": True,
            "reason": f"High complexity (score={complexity_score}/4): {', '.join(complexity_reasons)}",
            "confidence": "heuristic"
        }

    # Priority 4: Default
    if complexity_score > 0:
        reason = f"Low complexity (score={complexity_score}/4)"
        if complexity_reasons:
            reason += f": {', '.join(complexity_reasons)}"
    else:
        reason = "No complexity signals detected"

    return {
        "needs_planning": False,
        "reason": f"Default ({reason})",
        "confidence": "default"
    }


def main():
    if len(sys.argv) < 2:
        print(json.dumps({"error": "Usage: check-planning-needed.py <ticket-file>"}), file=sys.stderr)
        sys.exit(1)

    ticket_file = sys.argv[1]
    result = check_planning_needed(ticket_file)

    if "error" in result:
        print(json.dumps(result), file=sys.stderr)
        sys.exit(1)

    print(json.dumps(result))
    sys.exit(0)


if __name__ == '__main__':
    main()
