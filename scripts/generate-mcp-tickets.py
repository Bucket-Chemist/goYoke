#!/usr/bin/env python3
"""
Generate individual ticket files from MCP_IMPLEMENTATION_GUIDE.md

Parses the implementation guide and creates one markdown file per task
in .claude/tickets/mcp/ directory.
"""

import re
import os
from pathlib import Path
from typing import List, Dict, Optional

def slugify(text: str) -> str:
    """Convert text to filesystem-safe slug."""
    text = text.lower()
    text = re.sub(r'[^a-z0-9]+', '-', text)
    text = text.strip('-')
    return text

def parse_guide(guide_path: str) -> List[Dict[str, str]]:
    """Parse MCP_IMPLEMENTATION_GUIDE.md and extract tasks."""

    with open(guide_path, 'r') as f:
        content = f.read()

    tasks = []
    current_phase = None
    current_task = None
    in_task = False
    task_content = []

    for line in content.split('\n'):
        # Detect phase headers
        phase_match = re.match(r'^### (Phase \d+:.*)', line)
        if phase_match:
            current_phase = phase_match.group(1)
            continue

        # Detect task headers
        task_match = re.match(r'^\*\*Task (\d+\.\d+): (.+)\*\*$', line)
        if task_match:
            # Save previous task
            if current_task:
                current_task['content'] = '\n'.join(task_content)
                tasks.append(current_task)

            # Start new task
            task_id = task_match.group(1)
            task_name = task_match.group(2)
            current_task = {
                'id': task_id,
                'name': task_name,
                'phase': current_phase or 'Unknown Phase',
                'content': ''
            }
            task_content = []
            in_task = True
            continue

        # End task on section boundary
        if in_task and (line.startswith('---') or
                       line.startswith('### Phase') or
                       line.startswith('## ') or
                       re.match(r'^\*\*Task \d+\.\d+:', line)):
            if current_task:
                current_task['content'] = '\n'.join(task_content)
                tasks.append(current_task)
                current_task = None
                in_task = False
                task_content = []

        # Collect task content
        if in_task:
            task_content.append(line)

    # Save last task
    if current_task:
        current_task['content'] = '\n'.join(task_content)
        tasks.append(current_task)

    return tasks

def generate_ticket(task: Dict[str, str], num: int, output_dir: Path) -> Path:
    """Generate a single ticket markdown file."""

    # Create filename
    task_id_slug = task['id'].replace('.', '-')
    task_name_slug = slugify(task['name'])
    filename = f"{num:03d}-{task_id_slug}-{task_name_slug}.md"
    filepath = output_dir / filename

    # Build ticket content
    content = f"""# Task {task['id']}: {task['name']}

**Phase:** {task['phase']}
**Task ID:** {task['id']}
**Status:** Not Started

---

{task['content'].strip()}

---

## Status Tracking

- [ ] Task assigned to agent
- [ ] Dependencies reviewed
- [ ] Implementation started
- [ ] Code written
- [ ] Tests written
- [ ] Tests passing
- [ ] Code reviewed
- [ ] Documentation updated
- [ ] Task complete

## Notes

(Add implementation notes, blockers, or questions here)
"""

    # Write ticket file
    with open(filepath, 'w') as f:
        f.write(content)

    return filepath

def generate_index(tasks: List[Dict[str, str]], output_dir: Path):
    """Generate INDEX.md with all tickets organized by phase."""

    # Group tasks by phase
    phases = {}
    for i, task in enumerate(tasks, 1):
        phase = task['phase']
        if phase not in phases:
            phases[phase] = []

        task_id_slug = task['id'].replace('.', '-')
        task_name_slug = slugify(task['name'])
        filename = f"{i:03d}-{task_id_slug}-{task_name_slug}"

        phases[phase].append({
            'num': i,
            'id': task['id'],
            'name': task['name'],
            'filename': filename
        })

    # Build index content
    content = f"""# MCP Implementation Tickets

Generated from: `MCP_IMPLEMENTATION_GUIDE.md`
Date: {Path('MCP_IMPLEMENTATION_GUIDE.md').stat().st_mtime}
Total Tickets: {len(tasks)}

---

## Quick Start

```bash
# View all tickets
ls -1 .claude/tickets/mcp/*.md

# View a specific ticket
cat .claude/tickets/mcp/001-1-1-mcp-protocol-implementation.md

# Search tickets
grep -r "ask_user" .claude/tickets/mcp/

# Track progress
grep -r "Status: In Progress" .claude/tickets/mcp/
```

---

## Tickets by Phase

"""

    for phase, phase_tasks in phases.items():
        content += f"\n### {phase}\n\n"
        for task in phase_tasks:
            content += f"{task['num']:3d}. [{task['id']}] {task['name']}\n"
            content += f"     `.claude/tickets/mcp/{task['filename']}.md`\n\n"

    content += """
---

## Ticket Format

Each ticket includes:
- **Task ID and name** - Unique identifier
- **Phase** - Implementation phase (1-4)
- **Status** - Not Started, In Progress, Complete
- **Owner** - Assigned agent (go-pro, go-tui, etc.)
- **Complexity** - Low, Medium, High
- **Time estimate** - Expected days
- **Dependencies** - Blocked by which tasks
- **Subtasks** - Breakdown of work
- **Acceptance criteria** - Definition of done
- **Code examples** - Implementation guidance
- **Status tracking checklist** - Progress checkboxes

---

## Workflow

1. **Pick a ticket** from appropriate phase
2. **Review dependencies** - ensure blocked tasks are complete
3. **Assign to agent** - Update ticket with owner
4. **Update status** - Change to "In Progress"
5. **Implement** - Follow subtasks and code examples
6. **Test** - Verify acceptance criteria
7. **Update status** - Change to "Complete"
8. **Move to next ticket**

---

## Integration with Task System

To use with goyoke task tracking:

```bash
# In a future session, create tasks from tickets
for ticket in .claude/tickets/mcp/*.md; do
    # Parse ticket and create TaskCreate from it
    # (Can be automated with a script)
done
```
"""

    index_path = output_dir / 'INDEX.md'
    with open(index_path, 'w') as f:
        f.write(content)

    return index_path

def main():
    """Main entry point."""
    guide_path = 'MCP_IMPLEMENTATION_GUIDE.md'
    output_dir = Path('.claude/tickets/mcp')

    # Check guide exists
    if not Path(guide_path).exists():
        print(f"❌ Error: {guide_path} not found")
        print(f"   Run this script from project root: {Path.cwd()}")
        return 1

    # Create output directory
    output_dir.mkdir(parents=True, exist_ok=True)

    print(f"📖 Parsing {guide_path}...")
    tasks = parse_guide(guide_path)

    if not tasks:
        print("❌ No tasks found in guide")
        return 1

    print(f"✅ Found {len(tasks)} tasks")
    print(f"📝 Generating tickets in {output_dir}/...")

    # Generate ticket files
    for i, task in enumerate(tasks, 1):
        filepath = generate_ticket(task, i, output_dir)
        print(f"   {i:3d}. {filepath.name}")

    # Generate index
    print(f"\n📋 Generating index...")
    index_path = generate_index(tasks, output_dir)
    print(f"   {index_path}")

    print(f"\n✅ Complete! Generated {len(tasks)} tickets")
    print(f"\n📋 Next steps:")
    print(f"   cat {index_path}")
    print(f"   cat {output_dir}/001-1-1-mcp-protocol-implementation.md")

    return 0

if __name__ == '__main__':
    exit(main())
