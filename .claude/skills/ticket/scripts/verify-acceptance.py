#!/home/doktersmol/.generic-python/bin/python3
"""
Verify all acceptance criteria checkboxes are marked complete.

Parses markdown for checkbox patterns:
- "- [ ]" = incomplete
- "- [x]" or "- [X]" = complete
"""

import sys
import json
from pathlib import Path
from typing import Dict, Any, List
import re

try:
    import frontmatter
except ImportError:
    print(json.dumps({"error": "python-frontmatter not installed. Run: pip install python-frontmatter"}), file=sys.stderr)
    sys.exit(1)


def extract_checkboxes(content: str) -> Dict[str, Any]:
    """
    Extract and analyze checkbox completion status.

    Args:
        content: Markdown content

    Returns:
        Dict with all_complete, total, completed, pending
    """
    # Pattern to match checkboxes with their text
    # Matches: "- [ ] Some task" or "- [x] Done task"
    checkbox_pattern = r'^\s*-\s+\[([ xX])\]\s+(.+)$'

    matches = re.finditer(checkbox_pattern, content, re.MULTILINE)

    total = 0
    completed = 0
    pending = []

    for match in matches:
        total += 1
        checkbox_state = match.group(1)
        checkbox_text = match.group(2).strip()

        if checkbox_state.lower() == 'x':
            completed += 1
        else:
            pending.append(checkbox_text)

    return {
        "all_complete": total > 0 and completed == total,
        "total": total,
        "completed": completed,
        "pending": pending
    }


def verify_acceptance(ticket_file: str) -> Dict[str, Any]:
    """
    Verify acceptance criteria completion.

    Args:
        ticket_file: Path to ticket markdown file

    Returns:
        Dict with all_complete, total, completed, pending
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

    return extract_checkboxes(post.content)


def main():
    if len(sys.argv) < 2:
        print(json.dumps({"error": "Usage: verify-acceptance.py <ticket-file>"}), file=sys.stderr)
        sys.exit(1)

    ticket_file = sys.argv[1]
    result = verify_acceptance(ticket_file)

    if "error" in result:
        print(json.dumps(result), file=sys.stderr)
        sys.exit(1)

    print(json.dumps(result))

    # Exit code: 0 if all complete, 1 if any pending
    sys.exit(0 if result['all_complete'] else 1)


if __name__ == '__main__':
    main()
