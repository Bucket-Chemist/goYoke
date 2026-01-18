# GOgent-Fortress Claude Configuration

This document describes the Claude Code configuration for the GOgent-Fortress migration project.

---

## Project Type

**Language**: Go
**Framework**: CLI + TUI (Bubble Tea)
**Infrastructure**: Ticket-driven workflow with automated testing

---

## Ticket System

This project uses the `/ticket` skill for structured implementation. See [Ticket Workflow Guide](#ticket-workflow-guide) below.

### Quick Commands

```bash
# Start next available ticket
/ticket next

# Check current status
/ticket status

# Verify acceptance criteria met
/ticket verify

# Complete ticket and generate commit
/ticket complete
```

### Configuration

Ticket system configuration is in `.ticket-config.json`:

```json
{
  "tickets_dir": "migration_plan/tickets",
  "project_name": "GOgent-Fortress",
  "audit_config": {
    "enabled": true,
    "test_commands": {
      "go": {
        "unit": "go test -v ./...",
        "race": "go test -race ./...",
        "coverage": "go test -coverprofile={audit_dir}/coverage.out ./..."
      }
    }
  }
}
```

---

## Audit System

The ticket system includes automated audit testing (Phase 7.5). After completing ticket work and verifying acceptance criteria:

1. Tests execute automatically (`go test -v`, race detection, coverage)
2. Results logged to `.ticket-audits/{ticket_id}/`
3. Implementation summary generated
4. Ticket completion proceeds (audit failures are non-blocking)

### Audit Configuration

Enable/disable in `.ticket-config.json`:

```json
{
  "audit_config": {
    "enabled": true,
    "test_commands": { ... }
  }
}
```

### Output Artifacts

After audit completes:

```
.ticket-audits/
├── {ticket_id}/
│   ├── unit-tests.log          # Go test output
│   ├── race-detector.log        # Race condition scan
│   ├── coverage.out             # Binary coverage profile
│   ├── coverage-report.txt      # Per-function coverage
│   ├── coverage-summary.txt     # Total coverage %
│   └── implementation-summary.md # Human-readable summary
```

### Documentation

See complete audit configuration reference:
- **Schema**: `~/.claude/skills/ticket/docs/audit-config-schema.md`
- **Examples**: `~/.claude/skills/ticket/examples/`

---

## Ticket Workflow Guide

### Phase 1-3: Discovery → Validation

When you run `/ticket next`:
1. Discovers ticket directory (from `.ticket-config.json`)
2. Selects next pending ticket (dependencies met)
3. Validates ticket schema (frontmatter, acceptance criteria)

### Phase 4-5: Planning Decision

System decides if ticket needs architect planning:
- Explicit `needs_planning` field in ticket frontmatter, OR
- Complexity heuristics (4+ files, 3+ dependencies, multi-package)

If planning needed, orchestrator spawns architect agent automatically.

### Phase 6: Implementation Tracking

Ticket status updated to `in_progress` and TodoWrite created from acceptance criteria.

### Phase 7: Verification

After work completed, run `/ticket verify`:
- Checks acceptance criteria checkboxes
- Confirms all are marked complete
- Shows pending items if any

### Phase 7.5: Audit Documentation

If `audit_config.enabled: true`:
- Executes language-specific tests (unit, integration, race detection)
- Generates coverage reports
- Creates implementation summary document
- **Non-blocking**: Test failures do NOT prevent completion

### Phase 8: Completion

Run `/ticket complete`:
- Generates commit message from ticket title + description
- Shows preview for confirmation
- Commits and updates ticket status to `completed`
- Clears current ticket state

---

## Key Files

| File | Purpose |
|------|---------|
| `.ticket-config.json` | Ticket system configuration (tickets_dir, audit settings) |
| `migration_plan/tickets/tickets-index.json` | Ticket registry and status tracking |
| `migration_plan/tickets/*.md` | Individual ticket specifications |
| `.ticket-audits/{ticket_id}/` | Audit results and test logs |
| `CLAUDE.md` | This file - Claude configuration reference |

---

## Conventions

Go projects in this system follow:
- **Naming**: Go project layout with `cmd/`, `pkg/`, `internal/`
- **Testing**: Standard `*_test.go` files with go test
- **Coverage**: Generated via `go test -coverprofile`
- **Race Detection**: Enabled via `go test -race`

See `~/.claude/conventions/go.md` for complete Go style guide.

---

## Testing

Tests are run manually with:

```bash
# Unit tests
go test -v ./...

# Race detector (catches concurrent access bugs)
go test -race ./...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

Tests are also automated when completing tickets (if audit enabled).

---

## Contact

For questions about:
- **Ticket workflow**: See `~/.claude/skills/ticket/SKILL.md`
- **Audit configuration**: See `~/.claude/skills/ticket/docs/audit-config-schema.md`
- **Go conventions**: See `~/.claude/conventions/go.md`
- **Global configuration**: See `~/.claude/CLAUDE.md`

---

**Project**: GOgent-Fortress
**Status**: Active Development (Ticket-Driven)
**Last Updated**: 2026-01-18
**Maintained By**: System
