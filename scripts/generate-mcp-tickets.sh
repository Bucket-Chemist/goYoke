#!/bin/bash
# Generate individual ticket files from MCP_IMPLEMENTATION_GUIDE.md

set -euo pipefail

GUIDE="MCP_IMPLEMENTATION_GUIDE.md"
TICKETS_DIR=".claude/tickets/mcp"
COUNTER=0

# Create tickets directory
mkdir -p "$TICKETS_DIR"

echo "Generating MCP implementation tickets from $GUIDE..."

# Extract tasks using awk to parse markdown structure
awk '
BEGIN {
    in_task = 0
    task_num = 0
    phase_num = 0
}

# Detect phase headers
/^### Phase [0-9]:/ {
    phase_num++
    phase_name = $0
    sub(/^### /, "", phase_name)
    next
}

# Detect task headers
/^\*\*Task [0-9]+\.[0-9]+:/ {
    # Save previous task if exists
    if (in_task) {
        print content > filename
        close(filename)
    }

    in_task = 1
    task_num++

    # Extract task ID and name
    task_line = $0
    sub(/^\*\*Task /, "", task_line)
    sub(/\*\*$/, "", task_line)

    split(task_line, parts, ": ")
    task_id = parts[1]
    task_name = parts[2]

    # Create filename
    gsub(/\./, "-", task_id)
    gsub(/[^a-zA-Z0-9-]/, "-", task_name)
    gsub(/--+/, "-", task_name)
    task_name = tolower(task_name)
    filename = sprintf(".claude/tickets/mcp/%03d-%s-%s.md", task_num, task_id, task_name)

    # Start content
    content = "# " task_line "\n\n"
    content = content "**Phase:** " phase_name "\n"
    content = content "**Task ID:** " task_id "\n\n"
    content = content "---\n\n"
    next
}

# Collect task content until next task or phase
in_task && (/^---$/ || /^### Phase/ || /^## / || /^\*\*Task/) {
    # End of current task
    print content > filename
    close(filename)
    in_task = 0

    # If this is a new task header, process it
    if (/^\*\*Task/) {
        # Handle this line in next iteration
        $0 = $0  # Keep the line for processing
    }
}

# Add lines to current task content
in_task {
    content = content $0 "\n"
}

END {
    # Save last task
    if (in_task) {
        print content > filename
        close(filename)
    }
}
' "$GUIDE"

# Count generated tickets
TICKET_COUNT=$(find "$TICKETS_DIR" -name "*.md" -type f | wc -l)

echo "✅ Generated $TICKET_COUNT ticket files in $TICKETS_DIR/"

# Create index file
cat > "$TICKETS_DIR/INDEX.md" <<EOF
# MCP Implementation Tickets

Generated from: \`$GUIDE\`
Date: $(date +%Y-%m-%d)
Total Tickets: $TICKET_COUNT

## Phases

### Phase 1: Foundation (Weeks 1-2)
$(ls "$TICKETS_DIR"/001-*.md "$TICKETS_DIR"/002-*.md "$TICKETS_DIR"/003-*.md "$TICKETS_DIR"/004-*.md 2>/dev/null | sed 's|.claude/tickets/mcp/||' | sed 's/\.md$//' | sed 's/^/- /')

### Phase 2: Interactive Prompts (Weeks 3-4)
$(ls "$TICKETS_DIR"/005-*.md "$TICKETS_DIR"/006-*.md "$TICKETS_DIR"/007-*.md "$TICKETS_DIR"/008-*.md "$TICKETS_DIR"/009-*.md "$TICKETS_DIR"/010-*.md 2>/dev/null | sed 's|.claude/tickets/mcp/||' | sed 's/\.md$//' | sed 's/^/- /')

### Phase 3: Production Hardening (Weeks 5-6)
$(ls "$TICKETS_DIR"/011-*.md "$TICKETS_DIR"/012-*.md "$TICKETS_DIR"/013-*.md "$TICKETS_DIR"/014-*.md 2>/dev/null | sed 's|.claude/tickets/mcp/||' | sed 's/\.md$//' | sed 's/^/- /')

### Phase 4: Extensibility (Weeks 7-8)
$(ls "$TICKETS_DIR"/015-*.md "$TICKETS_DIR"/016-*.md "$TICKETS_DIR"/017-*.md "$TICKETS_DIR"/018-*.md 2>/dev/null | sed 's|.claude/tickets/mcp/||' | sed 's/\.md$//' | sed 's/^/- /')

## Usage

\`\`\`bash
# View a ticket
cat .claude/tickets/mcp/001-1-1-mcp-protocol-implementation.md

# List all tickets
ls -1 .claude/tickets/mcp/*.md

# Search for specific task
grep -r "ask_user" .claude/tickets/mcp/
\`\`\`

## Ticket Format

Each ticket includes:
- Task ID and name
- Phase information
- Owner (agent assignment)
- Complexity and time estimate
- Dependencies
- Subtasks
- Acceptance criteria
- Code examples (where applicable)
EOF

echo "✅ Created index file: $TICKETS_DIR/INDEX.md"
echo ""
echo "📋 Next steps:"
echo "   1. Review tickets: cat $TICKETS_DIR/INDEX.md"
echo "   2. Pick a ticket: cat $TICKETS_DIR/001-1-1-mcp-protocol-implementation.md"
echo "   3. Start implementation: Assign to appropriate agent"
