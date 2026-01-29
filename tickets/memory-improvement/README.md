# Memory Improvement Tickets

Tickets for enhancing the GOgent-Fortress memory and learning systems.

## Active Tickets

| ID | Title | Priority | Status | Time Est |
|----|-------|----------|--------|----------|
| [GOgent-MEM-001](./GOgent-MEM-001.md) | Structured Problem Capture for Memory Improvement | HIGH | pending | 8-10 hrs |

## Ticket Summary

### GOgent-MEM-001: Structured Problem Capture

**Purpose:** Extend `gogent-sharp-edge` hook to capture structured problem-solution data when problems are resolved, feeding into `/memory-improvement` for automatic sharp-edge recommendations.

**Key Features:**
- Session-scoped pending resolution tracking
- Consecutive success threshold (2) for resolution confirmation
- File locking for concurrent write safety
- TTL-based cleanup of stale pending resolutions
- Gemini integration for pattern detection and sharp-edge recommendations
- Automatic rotation of solved-problems.jsonl

**Review Status:** Staff-architect 7-layer review COMPLETE
- 3 BLOCKER issues resolved
- 5 HIGH issues resolved
- 4 MEDIUM issues addressed (2 deferred to v1.1)

**Dependencies:** `github.com/gofrs/flock` for file locking

## Architecture Context

This work integrates with:
- `cmd/gogent-sharp-edge/` - PostToolUse hook (modified)
- `cmd/gogent-archive/` - SessionEnd hook (extended for cleanup)
- `pkg/telemetry/` - New solved_problem.go and classification.go
- `~/.claude/skills/memory-improvement/` - Gemini prompt updates
- `.claude/memory/solved-problems.jsonl` - New output file

## Related Documentation

- [Compound Engineering Integration Guide](../.claude/guides/compound-engineering-integration.md)
- [Routing Schema](../.claude/routing-schema.json)
- [Memory Improvement Skill](../.claude/skills/memory-improvement/SKILL.md)
