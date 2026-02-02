# Ticket Skill - Documentation

**Version**: 1.0
**Status**: Production
**Last Updated**: 2026-01-16

---

## Overview

The `/ticket` skill provides a systematic workflow for ticket-driven development:

- Auto-discovery of ticket systems in projects
- Dependency-aware ticket selection
- Schema validation (for properly formatted tickets)
- Planning decision routing (architect delegation when needed)
- Progress tracking with TaskCreate/TaskUpdate
- Acceptance criteria verification
- Automated commit generation

---

## File Structure

```
~/.claude/skills/ticket/
├── SKILL.md                          # Main skill orchestration (invoked by /ticket)
├── README.md                         # This file
└── scripts/
    ├── discover-project.sh           # Find ticket directory in project
    ├── find-next-ticket.sh           # Select next actionable ticket
    ├── check-planning-needed.py      # Determine if architect needed
    ├── validate-ticket-schema.py     # Validate ticket structure
    ├── verify-acceptance.py          # Check acceptance criteria completion
    ├── update-ticket-status.sh       # Update ticket status in index
    └── generate-commit-msg.sh        # Generate conventional commit message
```

---

## Ticket File Format

### **Recommended Format: Individual Ticket Files with Frontmatter**

The skill is designed for **individual ticket files** with YAML frontmatter:

```markdown
---
id: FEAT-001
title: Add user authentication
description: Implement JWT-based authentication with refresh tokens
status: pending
priority: high
time_estimate: 3h
dependencies: []
tags: [feature, security]
files_to_create:
  - pkg/auth/jwt.go
  - pkg/auth/jwt_test.go
acceptance_criteria_count: 7
---

## Task

Implement JWT-based authentication...

## Implementation

[Detailed implementation steps...]

## Acceptance Criteria

- [ ] JWT token generation implemented
- [ ] Token validation working
- [ ] Refresh token flow complete
- [ ] Tests passing (coverage ≥80%)
- [ ] Error handling for expired tokens
- [ ] Documentation updated
- [ ] Integration tests added

## Testing

[Test requirements...]
```

**Why this format?**

✅ **Schema validation works** - Scripts can validate frontmatter fields
✅ **Metadata queryable** - Can extract planning signals programmatically
✅ **Self-documenting** - Frontmatter shows ticket structure at a glance
✅ **Parser-friendly** - python-frontmatter handles parsing reliably
✅ **Version controllable** - Individual files = granular git history

### Alternative Format: Grouped Tickets (Limited Support)

The l-a-g-GO project uses **grouped tickets** (multiple tickets per .md file):

```markdown
# Week 1: Foundation & Event Parsing

### GoGent-001: Initialize Go Module

**Time**: 1h
**Dependencies**: GoGent-000
...

### GoGent-002: Define routing.Schema Struct

**Time**: 2h
**Dependencies**: GoGent-001
...
```

**Limitations with grouped format:**

⚠️ **Schema validation not available** - No frontmatter per ticket
⚠️ **Manual parsing required** - Can't use python-frontmatter
⚠️ **Planning heuristics limited** - Complexity signals harder to extract

**For grouped tickets:**

- `tickets-index.json` becomes single source of truth
- Schema validation skipped (assume index is correct)
- Planning decisions rely on explicit `planning_required` field in index

---

## Project Setup

### For New Projects (Recommended)

Use individual ticket files:

```bash
# 1. Create ticket directory
mkdir -p tickets/

# 2. Create tickets-index.json
cat > tickets/tickets-index.json << 'EOF'
{
  "metadata": {
    "project": "my-project",
    "version": "1.0"
  },
  "tickets": [
    {
      "id": "FEAT-001",
      "title": "First feature",
      "file": "FEAT-001.md",
      "status": "pending",
      "dependencies": []
    }
  ]
}
EOF

# 3. Create individual ticket file
cat > tickets/FEAT-001.md << 'EOF'
---
id: FEAT-001
title: First feature
description: Implement the first feature
status: pending
time_estimate: 2h
dependencies: []
tags: [feature]
---

## Acceptance Criteria
- [ ] Implementation complete
- [ ] Tests passing
EOF

# 4. Create project config (optional - enables auto-discovery)
cat > .ticket-config.json << 'EOF'
{
  "tickets_dir": "tickets",
  "project_name": "my-project"
}
EOF
```

### For Existing Projects (l-a-g-GO Pattern)

Grouped tickets work but with limitations:

```bash
# 1. Ensure tickets-index.json exists
ls migration_plan/finalised/tickets/tickets-index.json

# 2. Create .ticket-config.json at project root
cat > .ticket-config.json << 'EOF'
{
  "tickets_dir": "migration_plan/finalised/tickets",
  "project_name": "GoGent"
}
EOF

# 3. Note: Schema validation will be skipped
# Rely on tickets-index.json as source of truth
```

---

## Migration Path

### Converting Grouped Tickets → Individual Files

**Future task**: Create a migration script that:

1. Parses grouped ticket files (01-week1-foundation-events.md, etc.)
2. Extracts each ticket section
3. Generates individual .md files with frontmatter
4. Updates tickets-index.json to reference new files

**Script location**: `~/.claude/skills/ticket/scripts/migrate-to-individual.py` (TODO)

**Example transformation**:

```
# Before (grouped)
01-week1-foundation-events.md
├── GoGent-001
├── GoGent-002
└── GoGent-003

# After (individual)
tickets/
├── GoGent-001.md (with frontmatter)
├── GoGent-002.md (with frontmatter)
└── GoGent-003.md (with frontmatter)
```

---

## Dependencies

**System requirements:**

- `bash` ≥4.0
- `jq` (JSON query tool)
- `python3` ≥3.7
- `git` (for commit workflow)

**Python packages:**

- `python-frontmatter` (required for schema validation)

**Installation (Arch Linux with externally-managed Python):**

```bash
# Use generic-python environment
~/.generic-python/bin/pip install python-frontmatter

# Or for other systems
pip install python-frontmatter
```

**Note**: Python scripts use shebang `#!/home/doktersmol/.generic-python/bin/python3` for Arch Linux compatibility. Adjust if using different Python installation.

---

## Usage Examples

### Start Next Ticket

```bash
cd ~/my-project
/ticket next

# Output:
# [ticket] Found tickets at: /home/user/my-project/tickets
# [ticket] Selected: FEAT-001
# [ticket] Schema validation: PASS
# [ticket] Planning needed: false
# [ticket] Status updated: in_progress
# [ticket] Created 5 tasks from acceptance criteria
```

### Check Status

```bash
/ticket status

# Output:
# [ticket] Current: FEAT-001
# [ticket] Progress: 3/5 criteria complete
# [ticket] Pending:
#   - Tests passing
#   - Documentation updated
```

### Verify and Complete

```bash
/ticket verify

# Output:
# [ticket] Acceptance criteria: 5/5 complete ✓

/ticket complete

# Output:
# [ticket] Generated commit message:
# feat: FEAT-001 - First feature
# ...
# Commit and complete ticket? (y/n)
```

---

## Script Details

### discover-project.sh

Locates ticket directory using:

1. `.ticket-config.json` in current dir or ancestors
2. Git root + standard paths (implementation_plan/tickets/, migration_plan/finalised/tickets/, tickets/)

### find-next-ticket.sh

Queries tickets-index.json for first pending ticket with all dependencies completed.

### check-planning-needed.py

Determines if architect planning required via:

1. Explicit `needs_planning` frontmatter field
2. "planning" tag presence
3. Complexity heuristic (files>3, time>2h, deps>2, multi-package)
4. Default: false

**Requires**: Individual ticket files with frontmatter

### validate-ticket-schema.py

Validates ticket structure:

- Required frontmatter fields (id, title, description, status, time_estimate, dependencies)
- Status enum values (pending|in_progress|completed|blocked)
- Acceptance criteria presence
- Dependency references (if index provided)

**Requires**: Individual ticket files with frontmatter

### verify-acceptance.py

Parses markdown checkboxes to determine completion:

- Counts total criteria
- Counts completed (- [x])
- Lists pending criteria

**Works with**: Both individual and grouped ticket formats

### update-ticket-status.sh

Atomically updates ticket status in tickets-index.json using jq.

### generate-commit-msg.sh

Creates conventional commit message from ticket metadata.

**Requires**: Individual ticket files with frontmatter

---

## Known Limitations

### Current Version (1.0)

1. **Schema validation requires individual ticket files**
   - Grouped ticket format (l-a-g-GO) bypasses validation
   - Mitigation: Trust tickets-index.json as source of truth

2. **Planning heuristics limited without frontmatter**
   - Complexity signals need frontmatter fields
   - Mitigation: Use explicit `planning_required` in tickets-index.json

3. **No migration script yet**
   - Manual conversion from grouped → individual format
   - Mitigation: Document conversion pattern in this README

4. **Python path hardcoded**
   - Shebangs point to `~/.generic-python/bin/python3`
   - Mitigation: Update shebangs if using different Python installation

---

## Future Enhancements

- [ ] Migration script: grouped tickets → individual files
- [ ] Support for alternative frontmatter formats (TOML, JSON)
- [ ] Interactive ticket creation wizard (`/ticket new`)
- [ ] Dependency graph visualization
- [ ] Time tracking and estimation accuracy
- [ ] Multi-project ticket aggregation
- [ ] Integration with GitHub Issues/Projects

---

## Troubleshooting

**"python-frontmatter not installed"**

```bash
~/.generic-python/bin/pip install python-frontmatter
```

**"Schema validation FAILED" on grouped tickets**

- Expected for l-a-g-GO format
- Skip validation, rely on tickets-index.json
- Consider migrating to individual ticket files

**"No actionable tickets found"**

- Check dependencies are marked "completed" in tickets-index.json
- Verify tickets have `status: "pending"`

**Python script fails to execute**

- Check shebang points to correct Python installation
- Verify python-frontmatter is installed in that environment

---

## Contributing

When updating the skill:

1. Maintain backward compatibility with grouped ticket format
2. Document breaking changes in this README
3. Update SKILL.md if workflow changes
4. Test on both individual and grouped ticket projects
5. Increment version number

---

**Maintained By**: Claude Code System
**License**: Internal Use
**Support**: See ~/.claude/docs/ for general skill documentation
