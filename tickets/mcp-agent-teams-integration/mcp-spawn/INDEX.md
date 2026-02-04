# MCP Spawning Tickets Index

**Generated from**: mcp-spawning-v3.md
**Source Document**: Braintrust synthesis (Mozart → Einstein + Staff-Architect → Beethoven)
**Total Tickets**: 14

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

| Ticket | Title | Time | Priority | Status |
|--------|-------|------|----------|--------|
| MCP-SPAWN-009 | MCP Server Registration | 1h | HIGH | pending |
| MCP-SPAWN-010 | Mozart Orchestrator Update | 2h | HIGH | pending |
| MCP-SPAWN-011 | Review-Orchestrator Update | 2h | HIGH | pending |
| MCP-SPAWN-013 | Agent Relationship Validation | 3h | HIGH | pending |
| MCP-SPAWN-014 | Delegation Enforcement Hook | 2h | HIGH | pending |

---

## Phase 3: Testing & Documentation

| Ticket | Title | Time | Priority | Status |
|--------|-------|------|----------|--------|
| MCP-SPAWN-012 | Integration Testing & Docs | 4h | HIGH | pending |

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

Phase 2 (Integration + Schema):
  MCP-SPAWN-008 ─► MCP-SPAWN-009 ─┬─► MCP-SPAWN-010
                                  ├─► MCP-SPAWN-011
                                  └─► MCP-SPAWN-013 ─► MCP-SPAWN-014

Phase 3:
  MCP-SPAWN-010 + 011 + 014 ─► MCP-SPAWN-012
```

---

## Effort Summary

| Phase | Hours | Priority Items |
|-------|-------|----------------|
| Phase 0 | 6h | MCP verification (GATE) |
| Phase 1 | 13h | spawn_agent + infrastructure |
| Phase 2 | 10h | Orchestrator + relationship validation |
| Phase 3 | 4h | Testing + docs |
| **Total** | **33h** | ~4 days focused |

**With 50% buffer**: 18-24 days realistic timeline.

## Schema Alignment

Tickets MCP-SPAWN-013 and MCP-SPAWN-014 align with agent-relationships-schema.json:

| Schema Field | Validated By | Enforcement |
|--------------|--------------|-------------|
| `spawned_by` | MCP-SPAWN-013 | Block spawn |
| `can_spawn` | MCP-SPAWN-013 | Block spawn |
| `max_delegations` | MCP-SPAWN-013 | Block spawn |
| `must_delegate` | MCP-SPAWN-014 | Block completion |
| `min_delegations` | MCP-SPAWN-014 | Block completion |

---

## Quick Start

```bash
# Start with Phase 0 Gate ticket
cat MCP-SPAWN-001.md

# After Phase 0 passes, work through Phase 1
cat MCP-SPAWN-004.md
# ... etc
```

---

## Critical Success Criteria

1. **Phase 0 Gate**: MCP tools must be accessible from Task()-spawned subagents
2. **Phase 1 Complete**: spawn_agent tool working with mock CLI tests passing
3. **Phase 2 Complete**: Braintrust and /review workflows use spawn_agent
4. **Phase 3 Complete**: 3 successful real Braintrust runs without intervention
