#!/home/doktersmol/.generic-python/bin/python3
"""
Validate ticket schema before workflow execution.

Validates:
- Required frontmatter fields
- Field types and formats
- Acceptance criteria presence
- Dependency references (optional)
"""

import sys
import json
from pathlib import Path
from typing import Dict, Any, List, Tuple
import re

try:
    import frontmatter
except ImportError:
    print(json.dumps({"error": "python-frontmatter not installed. Run: pip install python-frontmatter"}), file=sys.stderr)
    sys.exit(1)


REQUIRED_FIELDS = {
    'id': str,
    'title': str,
    'description': str,
    'status': str,
    'time_estimate': str,
    'dependencies': list,
}

VALID_STATUSES = ['pending', 'in_progress', 'completed', 'blocked']


def validate_frontmatter(metadata: Dict[str, Any]) -> Tuple[List[str], List[str]]:
    """
    Validate frontmatter fields.

    Args:
        metadata: Ticket frontmatter metadata

    Returns:
        Tuple of (errors, warnings)
    """
    errors = []
    warnings = []

    # Check required fields exist
    for field, expected_type in REQUIRED_FIELDS.items():
        if field not in metadata:
            errors.append(f"Missing required field: {field}")
            continue

        value = metadata[field]

        # Type checking
        if not isinstance(value, expected_type):
            errors.append(f"Field '{field}' must be {expected_type.__name__}, got {type(value).__name__}")
            continue

        # Non-empty string validation
        if expected_type == str and not value.strip():
            errors.append(f"Field '{field}' cannot be empty")

    # Validate status enum
    if 'status' in metadata:
        status = metadata['status']
        if isinstance(status, str) and status not in VALID_STATUSES:
            errors.append(f"Invalid status '{status}'. Must be one of: {', '.join(VALID_STATUSES)}")

    # Validate time_estimate format
    if 'time_estimate' in metadata:
        time_str = metadata['time_estimate']
        if isinstance(time_str, str):
            # Accept patterns like "2h", "30m", "1.5h"
            if not re.match(r'^\d+\.?\d*[hm]$', time_str.lower()):
                errors.append(f"Invalid time_estimate format '{time_str}'. Expected format: '2h' or '30m'")

    # Validate dependencies is array
    if 'dependencies' in metadata:
        deps = metadata['dependencies']
        if isinstance(deps, list):
            for dep in deps:
                if not isinstance(dep, str):
                    warnings.append(f"Dependency '{dep}' should be a string")

    return errors, warnings


def validate_acceptance_criteria(content: str) -> Tuple[List[str], List[str]]:
    """
    Validate acceptance criteria presence in markdown body.

    Args:
        content: Ticket markdown content

    Returns:
        Tuple of (errors, warnings)
    """
    errors = []
    warnings = []

    # Look for checkbox patterns: "- [ ]" or "- [x]"
    checkbox_pattern = r'^\s*-\s+\[([ xX])\]'
    checkboxes = re.findall(checkbox_pattern, content, re.MULTILINE)

    if not checkboxes:
        errors.append("No acceptance criteria checkboxes found. Expected markdown checkboxes: '- [ ]' or '- [x]'")

    return errors, warnings


def validate_dependencies(metadata: Dict[str, Any], tickets_index_path: str) -> List[str]:
    """
    Validate dependency references against tickets index (optional).

    Args:
        metadata: Ticket frontmatter metadata
        tickets_index_path: Path to tickets-index.json

    Returns:
        List of warnings
    """
    warnings = []

    if not tickets_index_path or not Path(tickets_index_path).exists():
        return warnings

    try:
        with open(tickets_index_path, 'r', encoding='utf-8') as f:
            tickets_index = json.load(f)
    except Exception as e:
        warnings.append(f"Could not load tickets index: {str(e)}")
        return warnings

    dependencies = metadata.get('dependencies', [])
    if not isinstance(dependencies, list):
        return warnings

    # Extract ticket IDs from index
    valid_ticket_ids = set()
    for ticket in tickets_index.get('tickets', []):
        if 'id' in ticket:
            valid_ticket_ids.add(ticket['id'])

    # Check each dependency
    for dep in dependencies:
        if isinstance(dep, str) and dep not in valid_ticket_ids:
            warnings.append(f"Dependency '{dep}' not found in tickets index")

    return warnings


def validate_ticket(ticket_file: str, tickets_index_path: str = None) -> Dict[str, Any]:
    """
    Validate ticket schema.

    Args:
        ticket_file: Path to ticket markdown file
        tickets_index_path: Optional path to tickets-index.json

    Returns:
        Dict with valid, errors, warnings
    """
    ticket_path = Path(ticket_file)

    if not ticket_path.exists():
        return {
            "valid": False,
            "errors": [f"Ticket file not found: {ticket_file}"],
            "warnings": []
        }

    try:
        with open(ticket_path, 'r', encoding='utf-8') as f:
            post = frontmatter.load(f)
    except Exception as e:
        return {
            "valid": False,
            "errors": [f"Failed to parse ticket: {str(e)}"],
            "warnings": []
        }

    all_errors = []
    all_warnings = []

    # Validate frontmatter
    fm_errors, fm_warnings = validate_frontmatter(post.metadata)
    all_errors.extend(fm_errors)
    all_warnings.extend(fm_warnings)

    # Validate acceptance criteria
    ac_errors, ac_warnings = validate_acceptance_criteria(post.content)
    all_errors.extend(ac_errors)
    all_warnings.extend(ac_warnings)

    # Validate dependencies (optional)
    if tickets_index_path:
        dep_warnings = validate_dependencies(post.metadata, tickets_index_path)
        all_warnings.extend(dep_warnings)

    return {
        "valid": len(all_errors) == 0,
        "errors": all_errors,
        "warnings": all_warnings
    }


def main():
    if len(sys.argv) < 2:
        print(json.dumps({"error": "Usage: validate-ticket-schema.py <ticket-file> [tickets-index.json]"}), file=sys.stderr)
        sys.exit(1)

    ticket_file = sys.argv[1]
    tickets_index_path = sys.argv[2] if len(sys.argv) > 2 else None

    result = validate_ticket(ticket_file, tickets_index_path)

    print(json.dumps(result))

    # Exit code: 0 if valid, 1 if invalid
    sys.exit(0 if result['valid'] else 1)


if __name__ == '__main__':
    main()
