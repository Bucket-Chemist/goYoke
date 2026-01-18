# Ticket Workflow Guide

**Purpose**: Systematic execution of Phase 0 tickets with Git workflow integration.

---

## Overview

This guide provides:
1. **Structured prompts** for Claude to systematically work through tickets
2. **Git workflow** for branch-per-ticket development
3. **Test automation** for validation
4. **PR workflow** for code review

---

## Quick Start

### Manual Workflow (Recommended for Learning)

```bash
# 1. Pick next ticket
cat tickets-index.json | jq '.tickets[] | select(.status=="pending") | select(.dependencies | all(. as $dep | any($dep == env.COMPLETED_TICKETS[]))) | [.id, .title, .time_estimate] | @tsv' | head -1

# 2. Create branch
git checkout -b gogent-XXX-description

# 3. Work on ticket (use prompt template below)

# 4. Run tests
go test ./...

# 5. Commit & push
git add .
git commit -m "GOgent-XXX: Description

- Implementation details
- Test coverage

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"

git push origin gogent-XXX-description

# 6. Create PR
gh pr create --title "GOgent-XXX: Description" \
  --body "$(cat PR-TEMPLATE.md)" \
  --label "phase-0,week-X"
```

### Automated Workflow

```bash
# Run automation script
./scripts/ticket-workflow.sh next

# Or specify ticket
./scripts/ticket-workflow.sh GOgent-001
```

---

## Prompt Templates

### Template 1: Start Ticket Work

Use this prompt to begin work on a ticket:

```
I'm working on Phase 0 ticket [TICKET_ID] from the gogent-fortress Go migration.

**Ticket**: [TICKET_ID] - [TITLE]
**File**: migration_plan/finalised/tickets/[FILE]
**Time Estimate**: [TIME]
**Dependencies**: [DEPENDENCIES] (all complete)

**Task**: Read the full ticket specification from [FILE], then implement according to the detailed requirements.

**Requirements**:
1. Read the complete ticket from the file listed above
2. Implement ALL code specified (no placeholders or "omitted for brevity")
3. Write complete tests as specified in the ticket
4. Follow error message format: [component] What. Why. How to fix.
5. Use XDG-compliant paths (no hardcoded /tmp)
6. Implement STDIN timeout handling where specified
7. Ensure acceptance criteria are met

**Output Format**:
1. Confirm ticket read and understood
2. Create all files specified in "files_to_create"
3. Implement complete code with no placeholders
4. Write all tests specified
5. Verify acceptance criteria
6. Report completion status

**Important**: This is copy-paste-ready implementation. No shortcuts, no TODOs, complete working code.

Begin by reading the ticket file, then proceed with implementation.
```

### Template 2: Verify Ticket Completion

Use this after implementing to verify completion:

```
Verify ticket [TICKET_ID] completion:

1. **Files Created**: Check all files in tickets-index.json "files_to_create" exist
2. **Tests Pass**: Run `go test ./...` and verify 100% pass rate
3. **Acceptance Criteria**: Review ticket acceptance_criteria_count items - all met?
4. **Code Quality**:
   - No placeholders or TODOs
   - Error messages follow [component] format
   - XDG paths used (no /tmp hardcoding)
   - STDIN timeouts implemented where required
5. **Test Coverage**: Run `go test -cover ./...` - coverage ≥80% for new packages?

Report:
- ✅ Pass or ❌ Fail for each category
- Any issues found
- Recommendation: Ready for commit? Or needs fixes?
```

### Template 3: Create Pull Request

Use this to generate PR body:

```
Generate a pull request description for ticket [TICKET_ID]:

**Template**:
```markdown
# GOgent-XXX: [Title]

**Week**: [WEEK] | **Day**: [DAY] | **Priority**: [PRIORITY]

## Summary

[1-2 sentence summary of what this ticket accomplishes]

## Implementation

- Created: [list files]
- Implemented: [key functions/types]
- Tests: [test files with coverage %]

## Dependencies

- Depends on: [list completed tickets]
- Blocks: [list tickets that can now start]

## Acceptance Criteria

- [x] [Criterion 1]
- [x] [Criterion 2]
...

## Testing

```bash
# Unit tests
go test ./[package] -v

# Coverage
go test ./[package] -cover
```

**Result**: All tests pass, coverage [X]%

## Files Changed

- `[file]` (+XXX lines): [description]

## Related

- Ticket: `migration_plan/finalised/tickets/[FILE]#GOgent-XXX`
- Depends: #[PR numbers for dependencies]

---

🤖 Generated with [Claude Code](https://claude.com/claude-code)
```
```

### Template 4: Multi-Session Resume

Use this to resume work across sessions:

```
**Context**: Resuming Phase 0 gogent-fortress migration work.

**Current Status**:
- Completed tickets: [list GOgent-XXX completed]
- Current branch: [branch name]
- Last work: [brief summary of what was last done]

**Next Task**: Continue with ticket [TICKET_ID] or move to next ticket.

**Instructions**:
1. Check git status to see uncommitted work
2. If work in progress: Review and complete current ticket
3. If ticket complete: Commit, push, create PR
4. If no work in progress: Check tickets-index.json for next available ticket (status="pending", dependencies met)
5. Report next recommended action

Begin by assessing current state.
```

---

## Git Workflow

### Branch Naming Convention

```
gogent-XXX-short-description
```

Examples:
- `gogent-000-baseline-measurement`
- `gogent-001-go-module-setup`
- `gogent-025-gogent-validate-cli`

### Commit Message Format

```
GOgent-XXX: Title in sentence case

- Key implementation point 1
- Key implementation point 2
- Test coverage: XX%

[Optional: Additional context or notes]

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
```

### PR Labels

Use labels from `tickets-index.json`:
- `phase-0` (all PRs)
- `week-1`, `week-2`, `week-3`
- Specific tags: `setup`, `parsing`, `testing`, `cli`, etc.
- Priority: `critical`, `high`, `medium`, `low`

### PR Review Checklist

Before merging:
- [ ] All tests pass: `go test ./...`
- [ ] Coverage ≥80% for new packages
- [ ] No placeholders or TODOs
- [ ] Error messages follow format
- [ ] XDG paths used (no /tmp)
- [ ] Documentation updated if needed
- [ ] Acceptance criteria met

---

## Test Workflow

### Running Tests

```bash
# All tests
go test ./...

# Specific package
go test ./pkg/routing -v

# With coverage
go test ./... -cover

# Integration tests only
go test ./test/integration/... -v

# Benchmarks
go test -bench=. ./test/benchmark
```

### Test Log Format

Append test results to `/tests/`:

```bash
# Run tests and save log
go test ./... -v 2>&1 | tee tests/gogent-XXX-tests.log

# Add to PR
git add tests/gogent-XXX-tests.log
git commit -m "Add test logs for GOgent-XXX"
```

### Test Failure Protocol

If tests fail:
1. **Do not commit** until fixed
2. Review error messages
3. Fix issue
4. Re-run tests
5. Verify all pass before proceeding

---

## Status Tracking

### Update tickets-index.json

After completing a ticket:

```bash
# Mark ticket as complete
jq '.tickets |= map(if .id == "GOgent-XXX" then .status = "complete" else . end)' \
  tickets-index.json > tmp.json && mv tmp.json tickets-index.json

# Commit status update
git add tickets-index.json
git commit -m "Mark GOgent-XXX as complete"
```

### Check Next Available Ticket

```bash
# Find tickets with met dependencies
./scripts/next-ticket.sh

# Or manually query
cat tickets-index.json | jq -r '
  .tickets[] |
  select(.status == "pending") |
  select(.dependencies | length == 0 or all(. as $dep | any(.tickets[].id == $dep and .tickets[].status == "complete"))) |
  "\(.id): \(.title) (\(.time_estimate))"
' | head -5
```

---

## Progress Visualization

### Generate Progress Report

```bash
# Quick stats
echo "Completed: $(jq '[.tickets[] | select(.status=="complete")] | length' tickets-index.json) / 56"

# By week
for week in 0 1 2 3; do
  echo "Week $week: $(jq "[.tickets[] | select(.week==$week and .status==\"complete\")] | length" tickets-index.json) / $(jq "[.tickets[] | select(.week==$week)] | length" tickets-index.json)"
done

# Critical path progress
jq -r '.tickets[] | select(.priority=="critical") | "\(.id): \(.status)"' tickets-index.json
```

### Visualize Dependency Graph

```bash
# Render Mermaid diagram
mmdc -i dependency-graph.mmd -o dependency-graph.png

# Or view in browser (if mermaid-cli installed)
```

---

## Common Patterns

### Pattern: Start New Ticket

```bash
# 1. Find next ticket
TICKET=$(./scripts/next-ticket.sh)
echo "Next: $TICKET"

# 2. Create branch
git checkout main
git pull origin main
git checkout -b $(echo "$TICKET" | jq -r '.git_branch')

# 3. Use Template 1 prompt with ticket details

# 4. Implement according to ticket spec

# 5. Test
go test ./...

# 6. Commit
git add .
git commit -m "$(echo "$TICKET" | jq -r '.id + ": " + .title')"

# 7. Push & PR
git push origin HEAD
gh pr create --fill
```

### Pattern: Resume After Break

```bash
# 1. Check status
git status
git log -1 --oneline

# 2. Check current ticket
CURRENT_BRANCH=$(git branch --show-current)
CURRENT_TICKET=$(echo "$CURRENT_BRANCH" | grep -oP 'gogent-\d+')

# 3. Use Template 4 (Multi-Session Resume)

# 4. Continue work or move to next
```

### Pattern: Run Integration Tests

```bash
# 1. Build binaries
go build -o bin/gogent-validate cmd/gogent-validate/main.go
go build -o bin/gogent-archive cmd/gogent-archive/main.go
go build -o bin/gogent-sharp-edge cmd/gogent-sharp-edge/main.go

# 2. Run integration tests
go test ./test/integration/... -v

# 3. Save logs
go test ./test/integration/... -v 2>&1 | tee tests/integration-$(date +%Y%m%d).log
```

---

## Troubleshooting

### Issue: Dependencies Not Met

```bash
# Check which dependencies are blocking
TICKET_ID="GOgent-XXX"
jq --arg id "$TICKET_ID" -r '
  .tickets[] |
  select(.id == $id) |
  .dependencies[] as $dep |
  .tickets[] |
  select(.id == $dep) |
  "\(.id): \(.status)"
' tickets-index.json
```

**Solution**: Complete blocking tickets first, or check if status needs updating.

### Issue: Test Failing

1. Read error message carefully
2. Check if related to previous tickets (dependencies)
3. Review ticket acceptance criteria
4. Use Template 2 (Verify Completion) to identify gaps
5. Fix and re-test

### Issue: Merge Conflict

```bash
# 1. Fetch latest
git fetch origin main

# 2. Rebase onto main
git rebase origin/main

# 3. Resolve conflicts
# Edit files, then:
git add .
git rebase --continue

# 4. Force push (rebased)
git push origin HEAD --force-with-lease
```

---

## Best Practices

1. **One ticket at a time**: Don't start new tickets until current is complete and merged
2. **Complete implementation**: No placeholders, no TODOs, working code only
3. **Test before commit**: `go test ./...` must pass 100%
4. **Document decisions**: Add comments for non-obvious logic
5. **Small commits**: Commit logical units, not entire ticket at once
6. **Update status**: Mark tickets complete in tickets-index.json
7. **Preserve logs**: Save test logs to `/tests/` for reference

---

## Multi-Session Strategy

For work spanning multiple sessions:

### Session End

1. Commit WIP if incomplete: `git commit -m "WIP: GOgent-XXX partial implementation"`
2. Push branch: `git push origin HEAD`
3. Note next steps in commit message or GitHub issue comment
4. Update personal notes on what was last done

### Session Start

1. Use Template 4 (Multi-Session Resume)
2. Check `git status` and `git log`
3. Review uncommitted changes
4. Continue where left off or start new ticket

### State Tracking

Create a `SESSION-STATE.md` file:

```markdown
# Session State

**Last Updated**: 2026-01-XX HH:MM

## Current Work
- Branch: gogent-XXX-description
- Status: [in-progress/testing/ready-for-pr]
- Next: [what to do next]

## Completed This Session
- GOgent-XXX: Description
- GOgent-YYY: Description

## Blocked On
- None / [description of blocker]

## Notes
- [any relevant context for next session]
```

---

## Quick Reference

| Command | Purpose |
|---------|---------|
| `./scripts/next-ticket.sh` | Find next available ticket |
| `go test ./...` | Run all tests |
| `go test -cover ./...` | Test with coverage |
| `gh pr create --fill` | Create PR from commits |
| `git log --oneline -10` | Recent commits |
| `jq '.tickets[] \| select(.status=="pending")' tickets-index.json` | List pending tickets |

---

**Document Status**: ✅ Ready for Use
**Last Updated**: 2026-01-15
**Maintained By**: Project Team
