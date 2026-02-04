# MCP Spawning Tickets Index

**Generated from**: mcp-spawning-v3.md
**Source Document**: Braintrust synthesis (Mozart → Einstein + Staff-Architect → Beethoven)
**Total Tickets**: 15
**Last Updated**: 2026-02-04 (dependency fix + MCP-SPAWN-015 added)

---

## Phase 0: Verification (GATE)

**Must pass before any Phase 1 work begins.**

| Ticket | Title | Time | Priority | Status |
|--------|-------|------|----------|--------|
| MCP-SPAWN-001 | MCP Tool Availability Verification | 2h | CRITICAL | pending |
| MCP-SPAWN-002 | CLI I/O Verification | 1h | HIGH | pending |
| MCP-SPAWN-003 | Mock CLI Infrastructure | 3h | CRITICAL | pending |

**Gate Decision**: If MCP-SPAWN-001 fails, halt and implement flat coordination fallback.

---

## Phase 1: Foundation

| Ticket | Title | Time | Priority | Status |
|--------|-------|------|----------|--------|
| MCP-SPAWN-004 | Environment Validation Pre-flight | 2h | CRITICAL | pending |
| MCP-SPAWN-005 | Process Registry and Cleanup | 3h | CRITICAL | pending |
| MCP-SPAWN-006 | Store Interface Extension | 2h | HIGH | pending |
| MCP-SPAWN-007 | gogent-validate Nesting Check | 2h | CRITICAL | pending |
| MCP-SPAWN-008 | spawn_agent Tool Implementation | 4h | CRITICAL | pending |

---

## Phase 2: Integration + Schema Alignment

**CRITICAL**: 013 (Relationship Validation) must complete BEFORE 010/011 (Orchestrator Updates).
This ensures orchestrators never run without validation guardrails.

| Ticket | Title | Time | Priority | Deps | Status |
|--------|-------|------|----------|------|--------|
| MCP-SPAWN-009 | MCP Server Registration | 1h | HIGH | [008] | pending |
| MCP-SPAWN-013 | Agent Relationship Validation | 3h | HIGH | [008] | pending |
| MCP-SPAWN-010 | Mozart Orchestrator Update | 2h | HIGH | [009, **013**] | pending |
| MCP-SPAWN-011 | Review-Orchestrator Update | 2h | HIGH | [009, **013**] | pending |
| MCP-SPAWN-014 | Delegation Enforcement Hook | 2h | HIGH | [007, 013] | pending |

---

## Phase 3: Testing & Documentation

| Ticket | Title | Time | Priority | Deps | Status |
|--------|-------|------|----------|------|--------|
| MCP-SPAWN-012 | Integration Testing & Docs | 4h | HIGH | [010, 011, **014**] | pending |
| MCP-SPAWN-015 | Validation Integration Tests | 3h | HIGH | [012, 014] | pending |

---

## Dependency Graph

```
Phase 0 (GATE):
  MCP-SPAWN-001 ─┬─► MCP-SPAWN-002 ─► MCP-SPAWN-003
                 │
                 └─► MCP-SPAWN-004

Phase 1:
  MCP-SPAWN-003 ─┬─► MCP-SPAWN-005
  MCP-SPAWN-004 ─┼─► MCP-SPAWN-006
                 └─► MCP-SPAWN-007
                          │
  All Phase 1 deps ──────►└─► MCP-SPAWN-008

Phase 2 (CRITICAL PATH - validation before orchestrators):
  MCP-SPAWN-008 ─┬─► MCP-SPAWN-009 ─────────────────┬─► MCP-SPAWN-010
                 │                                   │
                 └─► MCP-SPAWN-013 ─────────────────┼─► MCP-SPAWN-011
                                                    │
  MCP-SPAWN-007 ────────────────────────────────────┴─► MCP-SPAWN-014

  Note: 009 and 013 can run in PARALLEL (both depend only on 008)
  010/011 require BOTH 009 AND 013 to complete before starting

Phase 3:
  MCP-SPAWN-010 + 011 + 014 ─► MCP-SPAWN-012 ─► MCP-SPAWN-015
```

**Why this ordering matters:**
- 013 validates spawn relationships (`spawned_by`, `can_spawn`, `max_delegations`)
- 010/011 are orchestrators that USE spawn_agent
- If 010/011 ran before 013, orchestrators would spawn without guardrails
- 014 validates delegation requirements at completion time
- 012 needs 014 complete to test full workflow with enforcement
- 015 tests the validation failure paths specifically

**Parallelism Opportunity:**
- 009 (MCP Server Registration) and 013 (Relationship Validation) can run concurrently
- Both depend only on 008, with no dependencies on each other
- Estimated time savings: ~2 hours if executed in parallel

---

## Effort Summary

| Phase | Hours | Priority Items |
|-------|-------|----------------|
| Phase 0 | 6h | MCP verification (GATE) |
| Phase 1 | 13h | spawn_agent + infrastructure |
| Phase 2 | 10h | Orchestrator + relationship validation |
| Phase 3 | 7h | Testing + docs + validation tests |
| **Total** | **36h** | ~4.5 days focused |

**With 50% buffer**: 20-27 days realistic timeline.

---

## Schema Alignment

Tickets MCP-SPAWN-013 and MCP-SPAWN-014 align with agent-relationships-schema.json:

| Schema Field | Validated By | When | Enforcement |
|--------------|--------------|------|-------------|
| `spawned_by` | MCP-SPAWN-013 | Spawn time | Block spawn |
| `can_spawn` | MCP-SPAWN-013 | Spawn time | Block spawn |
| `max_delegations` | MCP-SPAWN-013 | Spawn time | Block spawn |
| `must_delegate` | MCP-SPAWN-014 | Completion | Block completion |
| `min_delegations` | MCP-SPAWN-014 | Completion | Block completion |

---

## Ticket Dependency Matrix

| Ticket | Depends On | Blocks |
|--------|------------|--------|
| 001 | - | 002, 004, 007 |
| 002 | 001 | 003 |
| 003 | 002 | 005, 008 |
| 004 | 001 | 005, 006, 007 |
| 005 | 004 | 008 |
| 006 | 004 | 008 |
| 007 | 001 | 014 |
| 008 | 003, 004, 005, 006 | 009, 013 |
| 009 | 008 | 010, 011, 013 |
| 010 | 009, **013** | 012 |
| 011 | 009, **013** | 012 |
| 012 | 010, 011, **014** | 015 |
| 013 | 008 | 010, 011, 014 |
| 014 | 007, 013 | 012 |
| 015 | 012, 014 | - |

---

## Quick Start

```bash
# Start with Phase 0 Gate ticket
cat MCP-SPAWN-001.md

# After Phase 0 passes, work through Phase 1
cat MCP-SPAWN-004.md
# ... etc

# IMPORTANT: In Phase 2, complete 013 BEFORE 010/011
cat MCP-SPAWN-013.md  # Do this first!
cat MCP-SPAWN-010.md  # Then this
cat MCP-SPAWN-011.md  # And this
```

---

## Critical Success Criteria

1. **Phase 0 Gate**: MCP tools must be accessible from Task()-spawned subagents
2. **Phase 1 Complete**: spawn_agent tool working with mock CLI tests passing
3. **Phase 2 Complete**: Validation in place BEFORE orchestrators updated
4. **Phase 3 Complete**:
   - 3 successful real Braintrust runs without intervention
   - Validation failure paths tested (015)

---

## Change Log

| Date | Change |
|------|--------|
| 2026-02-04 | Initial ticket series from mcp-spawning-v3.md |
| 2026-02-04 | **CRITICAL FIX**: Added 013 as dependency for 010/011 |
| 2026-02-04 | **CRITICAL FIX**: Added 014 as dependency for 012 |
| 2026-02-04 | Added MCP-SPAWN-015 for validation integration tests |
