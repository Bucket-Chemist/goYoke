---
name: explore-add
description: This skill should be used when the user wants to add a custom skill to the spawner system. Invoke with /explore-add. Guides user through creating a skill with the spawner structure (skill.yaml, sharp-edges.yaml). Custom skills are saved to ~/.claude/custom-skills/ and take precedence over spawner skills during matching.
---

# Custom Skill Creator

## Overview

This skill enables users to add custom skills to the spawner system. Custom skills follow the spawner YAML structure and are stored in `~/.claude/custom-skills/`. They take precedence over spawner skills during matching in `/explore`.

## Invocation

- `/explore-add` - Start skill creation interactively

## Workflow

### Step 1: Category Selection

When invoked, show available categories and prompt:

```
[explore-add] Custom Skill Creator

Available categories (from spawner, or create new):
  ai, ai-agents, ai-tools, backend, biotech, blockchain, cli, climate,
  communications, community, creative, data, design, development, devops,
  education, enterprise, finance, frameworks, frontend, game-dev, hardware,
  integrations, legal, maker, marketing, mind, product, science, security,
  simulation, space, startup, strategy, testing, trading

Which category? (Enter name, or 'new [name]' to create)
```

### Step 2: Skill Content Input

Once category is selected:

```
[explore-add] Creating skill in category: [CATEGORY]

Paste your skill content. Supported formats:

**Format 1 - Frontmatter + Markdown (recommended):**
---
name: my-skill
description: What this skill does
triggers:
  - "trigger phrase 1"
  - "trigger phrase 2"
tags: [tag1, tag2]
---
# Skill Title

## What to Do
[patterns and guidance]

## What NOT to Do
[anti-patterns]

## Watch Out For
[sharp edges / gotchas]

**Format 2 - Plain description:**
Just describe what the skill covers and I'll structure it.

Paste content now (end with a line containing only 'END'):
```

### Step 3: Parse and Generate Structure

Parse the input and generate the spawner 4-file structure.

**If frontmatter format detected:**

```python
# Extract from frontmatter
id = slugify(name)  # e.g., "my-skill" from "My Skill"
name = frontmatter['name']
description = frontmatter['description']
triggers = frontmatter.get('triggers', [])
tags = frontmatter.get('tags', [])

# Extract from body
patterns = extract_sections('## What to Do', '## Patterns')
anti_patterns = extract_sections('## What NOT to Do', '## Anti-Patterns')
sharp_edges = extract_sections('## Watch Out For', '## Sharp Edges', '## Gotchas')
```

**If plain description:**

Use Claude's understanding to structure the content:
1. Identify the core purpose → name, description
2. Extract action verbs → triggers
3. Identify domain terms → tags
4. Separate do's and don'ts → patterns, anti_patterns
5. Identify warnings → sharp_edges

### Step 4: Generate skill.yaml

```yaml
id: [generated-id]
name: [Name]
version: 1.0.0
layer: 2
description: [description]

owns: []

pairs_with: []

requires: []

tags:
  - [tag1]
  - [tag2]

triggers:
  - [trigger1]
  - [trigger2]

identity: |
  [Generated persona based on skill domain]

patterns:
  - name: [Pattern Name]
    description: [What it does]
    when: [When to use]
    example: |
      [Code or guidance]

anti_patterns:
  - name: [Anti-Pattern Name]
    description: [What it is]
    why: [Why it's bad]
    instead: [What to do instead]

handoffs: []
```

### Step 5: Generate sharp-edges.yaml

If sharp edges were provided or can be inferred:

```yaml
sharp_edges:
  - id: [skill-id]-001
    summary: [Brief description]
    severity: [critical|high|medium|low]
    situation: [When this happens]
    why: |
      [Root cause explanation]
    solution: |
      [How to fix/avoid]
    symptoms:
      - [Observable sign 1]
      - [Observable sign 2]
    detection_pattern: '[regex pattern if applicable]'
```

If no sharp edges provided, create empty file:
```yaml
sharp_edges: []
```

### Step 6: Review Generated Files

Present the generated structure for review:

```
[explore-add] Generated skill structure:

**skill.yaml:**
  id: deslop
  name: Deslop
  category: workflow
  triggers: ["clean up ai code", "remove slop", "deslop"]
  tags: [code-quality, cleanup, ai-generated]
  patterns: 1 defined
  anti_patterns: 1 defined

**sharp-edges.yaml:**
  edges: 1 defined
  - [high] Over-aggressive removal of legitimate defensive code

**Files will be created at:**
  ~/.claude/custom-skills/workflow/deslop/
    ├── skill.yaml
    └── sharp-edges.yaml

Approve? (yes / edit / cancel)
```

### Step 7: Handle User Response

**If "yes":**
1. Create directory: `~/.claude/custom-skills/{category}/{skill-id}/`
2. Write skill.yaml
3. Write sharp-edges.yaml
4. Regenerate index: Run `python3 ~/.claude/skills/explore/scripts/generate_index.py`
5. Confirm success

**If "edit":**
```
Which part to edit?
1. Basic info (name, description, triggers, tags)
2. Patterns
3. Anti-patterns
4. Sharp edges

Enter number:
```

Then allow inline editing of that section.

**If "cancel":**
```
[explore-add] Cancelled. No files created.
```

### Step 8: Confirm Success

```
[explore-add] Skill created successfully!

Location: ~/.claude/custom-skills/workflow/deslop/
Files:
  - skill.yaml
  - sharp-edges.yaml

Index updated: 419 skills (418 spawner + 1 custom)

The skill will appear in future /explore sessions matching:
  Triggers: "clean up ai code", "remove slop", "deslop"
  Tags: code-quality, cleanup, ai-generated
```

## Example: Adding the Deslop Skill

```
User: /explore-add

Claude: [explore-add] Custom Skill Creator

        Available categories: ai, backend, frontend, workflow, ...

        Which category?

User: workflow

Claude: [explore-add] Creating skill in category: workflow

        Paste your skill content (end with 'END'):

User: ---
      name: deslop
      description: Remove AI-generated code slop from a branch
      triggers:
        - "clean up ai code"
        - "remove slop"
        - "deslop"
      tags: [code-quality, cleanup, ai-generated]
      ---
      # Remove AI Code Slop

      Check the diff against main and remove all AI-generated slop.

      ## What to Remove
      - Extra comments that a human wouldn't add
      - Extra defensive checks abnormal for the codebase
      - Casts to `any` to get around type issues
      - Inline imports in Python (move to top)
      - Style inconsistent with the file

      ## Process
      1. Get diff: `git diff main...HEAD`
      2. Review each file for slop patterns
      3. Remove slop while preserving legitimate changes
      4. Report summary

      ## Watch Out For
      - Over-aggressive removal of legitimate defensive code
      - Removing checks that exist elsewhere in similar paths
      END

Claude: [explore-add] Generated skill structure:

        **skill.yaml:**
          id: deslop
          name: Deslop
          triggers: ["clean up ai code", "remove slop", "deslop"]
          tags: [code-quality, cleanup, ai-generated]
          patterns: 1 (diff-based review process)
          anti_patterns: 0

        **sharp-edges.yaml:**
          edges: 1
          - [high] Over-aggressive removal

        Approve? (yes / edit / cancel)

User: yes

Claude: [explore-add] Creating files...

        Created: ~/.claude/custom-skills/workflow/deslop/skill.yaml
        Created: ~/.claude/custom-skills/workflow/deslop/sharp-edges.yaml

        Regenerating index...
        Index updated: 419 skills (418 spawner + 1 custom)

        [explore-add] Skill 'deslop' added successfully!

        It will match in /explore when you mention:
          - "clean up ai code"
          - "remove slop"
          - "deslop"
```

## Editing Existing Custom Skills

To edit an existing custom skill:

```
/explore-add edit [skill-id]
```

This will:
1. Read the existing skill files
2. Present them for editing
3. Save changes and regenerate index

## Deleting Custom Skills

To delete a custom skill:

```
/explore-add delete [skill-id]
```

This will:
1. Confirm the skill exists in custom-skills/
2. Ask for confirmation
3. Delete the directory
4. Regenerate index

## Custom Skills Precedence

During skill matching in `/explore`:
1. Custom skills (`~/.claude/custom-skills/`) are searched FIRST
2. Then spawner skills (`~/.spawner/skills/`)
3. If a custom skill has the same ID as a spawner skill, the custom one wins

This allows you to:
- Override spawner skills with your own versions
- Add domain-specific skills not in spawner
- Customize patterns for your workflow

## Minimal Viable Skill

The absolute minimum for a custom skill:

```yaml
# skill.yaml
id: my-skill
name: My Skill
version: 1.0.0
layer: 2
description: What this skill does

triggers:
  - "my trigger phrase"

tags:
  - my-tag
```

Everything else is optional but recommended.

## Tips for Good Skills

1. **Triggers should be specific** - "django authentication" not just "auth"
2. **Tags should be searchable** - common terms people would use
3. **Patterns should have examples** - show, don't just tell
4. **Sharp edges should have detection patterns** - enable reactive warnings
5. **Description should explain WHEN to use** - not just what it does

## Not For
- Understanding existing skills (use /dummies-guide instead)
- Extending agent definitions with domain expertise (use /schema-extend instead)
- Modifying goYoke system configuration (edit settings.json or CLAUDE.md directly)
