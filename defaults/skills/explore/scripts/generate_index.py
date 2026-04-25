#!/usr/bin/env python3
"""
Skill Index Generator

Generates skills-index.json from spawner skills repository
and custom skills directory.

Usage:
    python generate_index.py [--spawner PATH] [--custom PATH] [--output PATH]

Default paths:
    --spawner: ~/.spawner/skills/
    --custom:  ~/.claude/custom-skills/
    --output:  ~/.claude/skills-index.json
"""

import json
import os
import sys
from pathlib import Path
from datetime import datetime
import argparse

# Try to import yaml, fall back to basic parsing if not available
try:
    import yaml
    HAS_YAML = True
except ImportError:
    HAS_YAML = False


def parse_yaml_basic(content: str) -> dict:
    """Basic YAML parser for simple key-value structures when PyYAML unavailable."""
    result = {}
    current_key = None
    current_list = None

    for line in content.split('\n'):
        stripped = line.strip()
        if not stripped or stripped.startswith('#'):
            continue

        # Check for list item
        if stripped.startswith('- '):
            if current_list is not None:
                current_list.append(stripped[2:].strip().strip('"\''))
            continue

        # Check for key-value
        if ':' in stripped:
            parts = stripped.split(':', 1)
            key = parts[0].strip()
            value = parts[1].strip() if len(parts) > 1 else ''

            if value == '' or value == '|':
                # Start of list or multiline
                current_key = key
                current_list = []
                result[key] = current_list
            elif value.startswith('[') and value.endswith(']'):
                # Inline list
                items = value[1:-1].split(',')
                result[key] = [i.strip().strip('"\'') for i in items if i.strip()]
                current_list = None
            else:
                result[key] = value.strip('"\'')
                current_list = None

    return result


def load_yaml(path: Path) -> dict:
    """Load a YAML file, return empty dict if missing."""
    if not path.exists():
        return {}

    content = path.read_text(encoding='utf-8')

    if HAS_YAML:
        try:
            return yaml.safe_load(content) or {}
        except yaml.YAMLError:
            return {}
    else:
        return parse_yaml_basic(content)


def count_sharp_edges(sharp_edges_data: dict) -> tuple:
    """Count total, critical, and high severity sharp edges."""
    edges = sharp_edges_data.get('sharp_edges', [])
    if not isinstance(edges, list):
        edges = []

    total = len(edges)
    critical = sum(1 for e in edges if isinstance(e, dict) and e.get('severity') == 'critical')
    high = sum(1 for e in edges if isinstance(e, dict) and e.get('severity') == 'high')

    return total, critical, high


def process_skill_directory(skill_dir: Path, category: str, source: str) -> dict:
    """Process a single skill directory and return index entry."""
    skill_yaml = load_yaml(skill_dir / 'skill.yaml')
    sharp_edges_yaml = load_yaml(skill_dir / 'sharp-edges.yaml')
    collaboration_yaml = load_yaml(skill_dir / 'collaboration.yaml')
    validations_yaml = load_yaml(skill_dir / 'validations.yaml')

    skill_id = skill_yaml.get('id')
    if not skill_id:
        return None

    total_edges, critical_edges, high_edges = count_sharp_edges(sharp_edges_yaml)

    # Get pairs_with from either skill.yaml or collaboration.yaml
    pairs_with = skill_yaml.get('pairs_with', [])
    if not pairs_with and collaboration_yaml:
        collab_pairs = collaboration_yaml.get('pairs_with', [])
        if isinstance(collab_pairs, list):
            # Extract skill names if it's a list of dicts
            pairs_with = []
            for p in collab_pairs:
                if isinstance(p, dict):
                    pairs_with.append(p.get('skill', ''))
                elif isinstance(p, str):
                    pairs_with.append(p)

    # Ensure lists are actually lists
    triggers = skill_yaml.get('triggers', [])
    if not isinstance(triggers, list):
        triggers = [triggers] if triggers else []

    tags = skill_yaml.get('tags', [])
    if not isinstance(tags, list):
        tags = [tags] if tags else []

    if not isinstance(pairs_with, list):
        pairs_with = [pairs_with] if pairs_with else []

    # Get description/summary
    summary = skill_yaml.get('description', '')
    if not summary:
        summary = skill_yaml.get('summary', '')

    # Check for validations
    validations = validations_yaml.get('validations', [])
    has_validations = isinstance(validations, list) and len(validations) > 0

    return {
        'id': skill_id,
        'category': category,
        'name': skill_yaml.get('name', skill_id),
        'triggers': triggers,
        'tags': tags,
        'pairs_with': [p for p in pairs_with if p],  # Filter empty
        'summary': summary[:200] if summary else '',  # Truncate for index
        'sharp_edges_count': total_edges,
        'sharp_edges_critical': critical_edges,
        'sharp_edges_high': high_edges,
        'has_validations': has_validations,
        'path': f"{category}/{skill_id}",
        'source': source
    }


def scan_skills_directory(base_path: Path, source: str) -> tuple:
    """Scan a skills directory and return skills list and categories dict."""
    skills = []
    categories = {}

    if not base_path.exists():
        return skills, categories

    # Skip non-category directories
    skip_dirs = {'.git', '.github', 'scripts', 'benchmarks', 'node_modules'}

    for category_dir in sorted(base_path.iterdir()):
        if not category_dir.is_dir():
            continue
        if category_dir.name in skip_dirs:
            continue
        if category_dir.name.startswith('.'):
            continue

        category = category_dir.name
        category_count = 0

        for skill_dir in sorted(category_dir.iterdir()):
            if not skill_dir.is_dir():
                continue

            # Check if it has skill.yaml
            if not (skill_dir / 'skill.yaml').exists():
                continue

            skill_entry = process_skill_directory(skill_dir, category, source)
            if skill_entry:
                skills.append(skill_entry)
                category_count += 1

        if category_count > 0:
            categories[category] = {
                'count': category_count,
                'source': source
            }

    return skills, categories


def generate_index(spawner_path: Path, custom_path: Path, output_path: Path):
    """Generate the skills index JSON file."""

    print(f"Scanning spawner skills: {spawner_path}")
    spawner_skills, spawner_categories = scan_skills_directory(spawner_path, 'spawner')
    print(f"  Found {len(spawner_skills)} skills in {len(spawner_categories)} categories")

    print(f"Scanning custom skills: {custom_path}")
    custom_skills, custom_categories = scan_skills_directory(custom_path, 'custom')
    print(f"  Found {len(custom_skills)} skills in {len(custom_categories)} categories")

    # Merge: custom skills override spawner skills with same ID
    custom_ids = {s['id'] for s in custom_skills}
    merged_skills = custom_skills + [s for s in spawner_skills if s['id'] not in custom_ids]

    # Merge categories
    merged_categories = {**spawner_categories}
    for cat, info in custom_categories.items():
        if cat in merged_categories:
            merged_categories[cat]['count'] += info['count']
            merged_categories[cat]['has_custom'] = True
        else:
            merged_categories[cat] = info

    index = {
        'version': '1.0.0',
        'generated_at': datetime.utcnow().isoformat() + 'Z',
        'sources': {
            'spawner': str(spawner_path),
            'custom': str(custom_path)
        },
        'stats': {
            'total_skills': len(merged_skills),
            'spawner_skills': len(spawner_skills),
            'custom_skills': len(custom_skills),
            'categories': len(merged_categories)
        },
        'skills': merged_skills,
        'categories': merged_categories
    }

    # Ensure output directory exists
    output_path.parent.mkdir(parents=True, exist_ok=True)

    # Write index (compact JSON for token efficiency)
    with open(output_path, 'w', encoding='utf-8') as f:
        json.dump(index, f, separators=(',', ':'))

    print(f"\nGenerated index: {output_path}")
    print(f"  Total skills: {len(merged_skills)}")
    print(f"  Spawner: {len(spawner_skills)}")
    print(f"  Custom: {len(custom_skills)}")
    print(f"  Categories: {len(merged_categories)}")

    # Also write to spawner directory as authoritative copy
    spawner_index = spawner_path / 'skills-index.json'
    with open(spawner_index, 'w', encoding='utf-8') as f:
        json.dump(index, f, separators=(',', ':'))
    print(f"  Authoritative copy: {spawner_index}")


def main():
    parser = argparse.ArgumentParser(description='Generate skills index')
    parser.add_argument('--spawner', type=Path,
                        default=Path.home() / '.spawner' / 'skills',
                        help='Path to spawner skills directory')
    parser.add_argument('--custom', type=Path,
                        default=Path.home() / '.claude' / 'custom-skills',
                        help='Path to custom skills directory')
    parser.add_argument('--output', type=Path,
                        default=Path.home() / '.claude' / 'skills-index.json',
                        help='Output path for index JSON')

    args = parser.parse_args()

    generate_index(args.spawner, args.custom, args.output)


if __name__ == '__main__':
    main()
