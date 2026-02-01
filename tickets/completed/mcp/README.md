# MCP Implementation Tickets

This directory contains individual tickets extracted from `MCP_IMPLEMENTATION_GUIDE.md`.

## Quick Start

```bash
# View the index
cat .claude/tickets/mcp/INDEX.md

# View a specific ticket
cat .claude/tickets/mcp/001-1-1-mcp-protocol-implementation.md

# List all tickets
ls -1 .claude/tickets/mcp/*.md

# Search for specific content
grep -r "ask_user" .claude/tickets/mcp/

# Track progress
grep -r "Status:" .claude/tickets/mcp/ | grep -v "Not Started"
```

## Directory Structure

```
.claude/tickets/mcp/
├── README.md                            # This file
├── INDEX.md                             # Master index of all tickets
├── 001-1-1-mcp-protocol-implementation.md
├── 002-1-2-unix-socket-transport.md
├── 003-1-3-tool-registry.md
└── ... (18 tickets total)
```

## Ticket Numbering

Format: `{sequence}-{task-id}-{task-name-slug}.md`

- **Sequence:** 001-018 (execution order)
- **Task ID:** Phase.Task (e.g., 1.1, 2.3)
- **Name Slug:** Lowercase, hyphenated task name

## Using Tickets in Future Sessions

### Option 1: Reference Ticket Directly

```bash
# In a new session
> Read ticket 001 and implement it

# I'll read:
cat .claude/tickets/mcp/001-1-1-mcp-protocol-implementation.md

# Then implement following the ticket's guidance
```

### Option 2: Create Tasks from Tickets

```bash
# Future: Script to convert tickets to TaskCreate
for ticket in .claude/tickets/mcp/*.md; do
    # Parse ticket metadata
    # Create TaskCreate() from it
done
```

### Option 3: Manual Task Creation

```bash
# In session
> Create tasks for Phase 1

# I'll read tickets 001-004 and create TaskCreate for each
```

## Regenerating Tickets

If `MCP_IMPLEMENTATION_GUIDE.md` is updated:

```bash
# From project root
python3 scripts/generate-mcp-tickets.py

# Or use bash version
./scripts/generate-mcp-tickets.sh
```

## Ticket Format

Each ticket contains:

```markdown
# Task {ID}: {Name}

**Phase:** Phase X: Description
**Task ID:** X.Y
**Status:** Not Started | In Progress | Complete

---

{Task content from guide including:}
- Owner (agent assignment)
- Complexity
- Time estimate
- Dependencies
- Subtasks
- Acceptance criteria
- Code examples

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

(Space for implementation notes)
```

## Workflow Example

**Session 1: Phase 1, Task 1**

```
User: Implement Task 1.1 - MCP Protocol

Assistant reads:
.claude/tickets/mcp/001-1-1-mcp-protocol-implementation.md