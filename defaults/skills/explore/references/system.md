# Enhanced Explore System - Technical Documentation

**Version:** 1.0.0
**Created:** 2026-01-08

This document provides complete technical documentation for the enhanced `/explore` system with spawner skill integration.

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Directory Structure](#directory-structure)
3. [Spawner Skill Schema](#spawner-skill-schema)
4. [Index System](#index-system)
5. [Matching Algorithm](#matching-algorithm)
6. [Sharp Edges System](#sharp-edges-system)
7. [Convention Conflict Resolution](#convention-conflict-resolution)
8. [Custom Skills](#custom-skills)
9. [Maintenance & Troubleshooting](#maintenance--troubleshooting)

---

## Architecture Overview

The enhanced `/explore` system integrates three knowledge layers:

```
┌─────────────────────────────────────────────────────────────────┐
│                      USER REQUEST                               │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                 ~/.claude/CLAUDE.md (Orchestrator)              │
│  - Session initialization                                       │
│  - Language detection                                           │
│  - Convention loading                                           │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                Layer 1: USER CONVENTIONS                        │
│  ~/.claude/conventions/                                         │
│  - python.md, R.md, R-shiny.md, R-golem.md                     │
│  - Language-specific coding standards                           │
│  - ALWAYS LOADED (override everything)                          │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                Layer 2: SPAWNER SKILLS                          │
│  ~/.spawner/skills/ (418 skills, 35 categories)                │
│  - Domain expertise (backend, frontend, ai, etc.)              │
│  - Patterns, anti-patterns, sharp edges                        │
│  - LOADED ON-DEMAND based on task matching                     │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                Layer 3: CUSTOM SKILLS                           │
│  ~/.claude/custom-skills/                                       │
│  - User-added skills                                            │
│  - Same format as spawner                                       │
│  - PRECEDENCE over spawner (searched first)                    │
└─────────────────────────────────────────────────────────────────┘
```

### Data Flow

1. User invokes `/explore`
2. Orchestrator runs session init (language detection, conventions)
3. User provides goal
4. `/explore` reads index, matches skills
5. Interview refines skill selection
6. User approves skill set
7. Full skill YAMLs loaded
8. Sharp edges surfaced as warnings
9. Convention conflicts detected
10. Plan mode entered with full context

---

## Directory Structure

```
~/.spawner/
└── skills/                        # Spawner repository (git clone)
    ├── ai/                        # Category directory
    │   ├── llm-whisperer/         # Skill directory
    │   │   ├── skill.yaml         # Core definition
    │   │   ├── sharp-edges.yaml   # Gotchas
    │   │   ├── validations.yaml   # Code checks
    │   │   └── collaboration.yaml # Relationships
    │   └── ...
    ├── backend/
    ├── frontend/
    ├── ... (35 categories)
    └── skills-index.json          # Authoritative index copy

~/.claude/
├── CLAUDE.md                      # Orchestrator (unchanged)
├── conventions/                   # Your rules (unchanged)
│   ├── python.md
│   ├── R.md
│   └── ...
├── skills/
│   ├── explore/                   # Enhanced skill
│   │   ├── SKILL.md               # Main workflow
│   │   ├── scripts/
│   │   │   └── generate_index.py  # Index generator
│   │   └── references/
│   │       └── system.md          # This file
│   ├── explore-add/               # Subskill
│   │   └── SKILL.md
│   ├── init-auto/                 # Existing skill
│   └── dummies-guide/             # Existing skill
├── custom-skills/                 # User-added skills
│   └── {category}/
│       └── {skill-id}/
│           ├── skill.yaml
│           └── sharp-edges.yaml
├── skills-index.json              # Cache of index
└── rules/
    └── LLM-guidelines.md          # Always loaded
```

---

## Spawner Skill Schema

### skill.yaml

```yaml
# Required fields
id: string                    # Unique identifier (kebab-case)
name: string                  # Human-readable name
version: string               # Semantic version (e.g., "1.0.0")
layer: integer                # 1=Core, 2=Integration, 3=Polish
description: string           # Brief summary (used in index)

# Discovery fields
triggers: array[string]       # Phrases that activate the skill
tags: array[string]           # Searchable keywords

# Relationship fields
owns: array[string]           # Domains this skill manages
pairs_with: array[string]     # Complementary skills
requires: array[string]       # Dependencies (load first)

# Content fields
identity: string              # Persona/expertise description
patterns: array               # Recommended approaches
  - name: string
    description: string
    when: string              # When to use
    example: string           # Code/guidance

anti_patterns: array          # Things to avoid
  - name: string
    description: string
    why: string               # Why it's bad
    instead: string           # What to do instead

handoffs: array               # Skill transitions
  - trigger: string           # Activation pattern
    to: string                # Target skill ID
    context: string           # What to pass
```

### sharp-edges.yaml

```yaml
sharp_edges: array
  - id: string                # Unique ID (e.g., "backend-001")
    summary: string           # Brief description
    severity: string          # critical | high | medium | low
    situation: string         # When this happens
    why: string               # Root cause (multiline)
    solution: string          # How to fix (multiline, with code)
    symptoms: array[string]   # Observable signs
    detection_pattern: string # Regex for reactive detection
```

### Severity Levels

| Severity | Description | Planning Behavior |
|----------|-------------|-------------------|
| critical | Data loss, security holes, production crashes | Always surface |
| high | Significant bugs, performance issues | Always surface |
| medium | Minor bugs, code smell | Surface on request |
| low | Style issues, minor optimizations | Don't surface |

---

## Index System

### Index Location

- **Cache:** `~/.claude/skills-index.json`
- **Authoritative:** `~/.spawner/skills/skills-index.json`

Both are identical; cache exists for faster access.

### Index Format

```json
{
  "version": "1.0.0",
  "generated_at": "2026-01-08T15:12:00Z",
  "sources": {
    "spawner": "/home/user/.spawner/skills",
    "custom": "/home/user/.claude/custom-skills"
  },
  "stats": {
    "total_skills": 419,
    "spawner_skills": 418,
    "custom_skills": 1,
    "categories": 35
  },
  "skills": [
    {
      "id": "backend",
      "category": "backend",
      "name": "Backend Engineering",
      "triggers": ["backend", "api", "database", ...],
      "tags": ["backend", "api", "database", ...],
      "pairs_with": ["frontend", "devops", ...],
      "summary": "World-class backend engineering...",
      "sharp_edges_count": 12,
      "sharp_edges_critical": 5,
      "sharp_edges_high": 4,
      "has_validations": true,
      "path": "backend/backend",
      "source": "spawner"
    },
    ...
  ],
  "categories": {
    "backend": {"count": 10, "source": "spawner"},
    "workflow": {"count": 1, "source": "custom", "has_custom": true},
    ...
  }
}
```

### Index Regeneration

```bash
# Manual regeneration
python3 ~/.claude/skills/explore/scripts/generate_index.py

# With custom paths
python3 ~/.claude/skills/explore/scripts/generate_index.py \
  --spawner ~/.spawner/skills \
  --custom ~/.claude/custom-skills \
  --output ~/.claude/skills-index.json
```

### Auto-generation Trigger

The index is regenerated automatically when:
1. First `/explore` invocation and index doesn't exist
2. Index file is corrupted/unparseable
3. After `/explore-add` creates a new skill

---

## Matching Algorithm

### Scoring Weights

| Match Type | Weight | Description |
|------------|--------|-------------|
| Exact trigger in goal | +10.0 | Trigger phrase found verbatim |
| Tag in goal tokens | +3.0 | Tag word found in goal |
| Category matches language | +2.0 | Skill category contains detected language |
| Summary keyword overlap | +0.5 | Goal word appears in summary |
| pairs_with bonus | +1.5 | Skill pairs with already-matched skill |

### Pseudocode

```python
def match_skills(goal: str, index: dict, language: str = None) -> list:
    goal_lower = goal.lower()
    goal_tokens = set(goal_lower.split())
    scores = []

    for skill in index['skills']:
        score = 0.0

        # Trigger matching (highest weight)
        for trigger in skill['triggers']:
            if trigger.lower() in goal_lower:
                score += 10.0

        # Tag matching
        for tag in skill['tags']:
            if tag.lower() in goal_tokens:
                score += 3.0

        # Language bonus
        if language and language.lower() in skill['category'].lower():
            score += 2.0

        # Summary overlap
        summary_lower = skill['summary'].lower()
        for word in goal_tokens:
            if len(word) > 3 and word in summary_lower:
                score += 0.5

        if score > 0:
            scores.append((skill, score))

    # Sort by score, take top 8
    scores.sort(key=lambda x: x[1], reverse=True)
    candidates = scores[:8]

    # Second pass: pairs_with bonus
    top_ids = {s['id'] for s, _ in candidates[:3]}
    for i, (skill, score) in enumerate(candidates):
        bonus = sum(1.5 for p in skill.get('pairs_with', []) if p in top_ids)
        candidates[i] = (skill, score + bonus)

    # Final sort, return top 5
    candidates.sort(key=lambda x: x[1], reverse=True)
    return candidates[:5]
```

### No-Match Behavior

If no skills score > 0:
- Announce: "No specific skill matches found"
- Proceed with user conventions only
- Do not block the workflow

---

## Sharp Edges System

### Planning Phase (Proactive)

When skills are loaded, critical and high severity edges are surfaced:

```
[explore] PLANNING WARNINGS - Sharp Edges to Watch:

CRITICAL:
  [skill-id: edge-id] Summary
  → Mitigation guidance

HIGH:
  [skill-id: edge-id] Summary
  → Mitigation guidance
```

### Implementation Phase (Reactive)

During code writing, if `detection_pattern` matches:

```
[explore] SHARP EDGE WARNING

[skill: edge-id] detected in current code

[Explanation of what was detected]

Mitigation: [How to fix]

Continue anyway? (y/n/fix)
```

### Detection Pattern Syntax

Uses standard regex. Common patterns:

```yaml
# Loop with database call
detection_pattern: 'for\\s*\\([^)]*\\)\\s*\\{[^}]*await.*find'

# External call in transaction
detection_pattern: '\\$transaction[^}]*(?:fetch|axios|http)'

# Check-then-act pattern
detection_pattern: 'if\\s*\\([^)]*balance[^)]*\\)[^}]*update'
```

---

## Convention Conflict Resolution

### Detection

Conflicts are detected when:
1. A spawner skill pattern contradicts a user convention rule
2. Both are relevant to the current task

### Resolution Flow

```
[explore] CONVENTION CONFLICT DETECTED

Your convention (from ~/.claude/conventions/python.md):
  "[extracted rule]"

Spawner skill (backend) recommends:
  "[contradicting pattern]"

Rationale: [why spawner suggests this]

My assessment: [which is better and why]

Options:
1. Keep your convention
2. Update convention to spawner's approach
3. Use spawner for this project only
```

### Option 2: Knowledge Compound

If user chooses to update convention:
1. Edit the relevant convention file
2. Add the new rule with explanation
3. Announce: `[Knowledge Compound] Updated ~/.claude/conventions/[file]`

This creates a compounding knowledge base.

---

## Custom Skills

### Location

```
~/.claude/custom-skills/{category}/{skill-id}/
```

### Precedence

Custom skills are searched BEFORE spawner skills:
1. Index includes both with `source` field
2. Matching algorithm processes custom first
3. Same-ID custom skill overrides spawner

### Creation

Via `/explore-add`:
1. Select category
2. Paste content
3. Review generated YAML
4. Approve to save

### Minimum Structure

```yaml
# skill.yaml (minimum)
id: my-skill
name: My Skill
version: 1.0.0
layer: 2
description: What this skill does
triggers:
  - "my trigger"
tags:
  - my-tag
```

---

## Maintenance & Troubleshooting

### Update Spawner Skills

```bash
cd ~/.spawner/skills
git pull
python3 ~/.claude/skills/explore/scripts/generate_index.py
```

### Index Problems

**Symptom:** "/explore can't find skills"

```bash
# Check index exists
ls -la ~/.claude/skills-index.json

# Regenerate if needed
python3 ~/.claude/skills/explore/scripts/generate_index.py

# Check output
cat ~/.claude/skills-index.json | python3 -m json.tool | head -20
```

**Symptom:** "Index parse error"

```bash
# Validate JSON
python3 -c "import json; json.load(open('$HOME/.claude/skills-index.json'))"

# If corrupt, regenerate
rm ~/.claude/skills-index.json
python3 ~/.claude/skills/explore/scripts/generate_index.py
```

### Skill Not Loading

1. Check skill exists: `ls ~/.spawner/skills/{category}/{skill-id}/`
2. Check skill.yaml valid: `python3 -c "import yaml; yaml.safe_load(open('skill.yaml'))"`
3. Check path in index matches actual location

### Sharp Edges Not Appearing

1. Check sharp-edges.yaml exists for the skill
2. Check severity is "critical" or "high"
3. Check edges array is not empty

### Custom Skill Not Matching

1. Verify file saved to correct location
2. Check index was regenerated after creation
3. Verify triggers match what you're typing

### PyYAML Not Installed

The index generator works without PyYAML using a basic parser, but for best results:

```bash
pip install pyyaml
```

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2026-01-08 | Initial release |
