# Team Coordination Tickets Index

**Source**: `IMPLEMENTATION-PLAN-FINAL.md` (Braintrust synthesis) + `REVIEW-SYNTHESIS.md` (review findings)
**Total Tickets**: 21

---

## Phase 0: Foundation Fixes

| Ticket | Title | Priority | Effort | Blocked By |
|--------|-------|----------|--------|------------|
| TC-001 | Replace `--permission-mode delegate` with `--allowedTools` | CRITICAL | 0.5d | none |
| TC-002 | Design Mutex-Protected Config Access | CRITICAL | 0.5d | none |

## Phase 1: Schema Design

| Ticket | Title | Priority | Effort | Blocked By |
|--------|-------|----------|--------|------------|
| TC-006 | Add `project_root` to Team Config Schema | HIGH | 0.5d | none |
| TC-007 | Document Task() Access for Team-Spawned Agents | MEDIUM | 0.5d | none |
| TC-009 | Design All Three Team Templates | HIGH | 2-3d | none |
| TC-009a | **Minimal Team Templates (MVP)** | **CRITICAL** | 1d | none |
| TC-014 | Add `cli_flags` to agents-index.json | MEDIUM | 1d | none |
| TC-017 | **gogent-validate Level 2 Enforcement** | **CRITICAL** | 1d | none |
| TC-018 | **Mozart Interview Protocol** | **CRITICAL** | 1d | none |

## Phase 2: Go Binary Implementation

| Ticket | Title | Priority | Effort | Blocked By |
|--------|-------|----------|--------|------------|
| TC-004 | Implement Full Daemon Pattern in Go Binary | HIGH | 1d | none |
| TC-008 | Implement `gogent-team-run` Go Binary | CRITICAL | 5-7d | TC-001, TC-002, TC-004, TC-006, TC-009a, TC-014, TC-017 |
| TC-003 | Fix Recursive Retry WaitGroup Panic | HIGH | incl. in TC-008 | TC-002 |
| TC-005 | Verify and Document CLI JSON Output Format | HIGH | 0.5d | none |
| TC-010 | Rewrite Inter-Wave Script as Go Binary | HIGH | 1-2d | TC-008 |
| TC-011 | Unit Tests for Go Binary | HIGH | 1-2d | TC-002, TC-003 |

## Phase 3: Slash Commands

| Ticket | Title | Priority | Effort | Blocked By |
|--------|-------|----------|--------|------------|
| TC-012 | Implement Team Slash Commands | HIGH | 2-3d | TC-008 |

## Phase 4: Orchestrator Rewrites

| Ticket | Title | Priority | Effort | Blocked By |
|--------|-------|----------|--------|------------|
| TC-013 | Rewrite Orchestrator Prompts for Team Pattern | HIGH | 3-5d | TC-012 |

## Independent

| Ticket | Title | Priority | Effort | Blocked By |
|--------|-------|----------|--------|------------|
| TC-015 | TUI Concurrent Query Support | MEDIUM | 3-5d | TC-019 |
| TC-016 | Duplicate Launch Prevention via PID File | LOW | 0.5d | TC-008 |
| TC-019 | **SDK Concurrency Investigation** | **CRITICAL** | 1-2d | none |
| TC-020 | Orchestrator Rewrite Design Docs | MEDIUM | 2-3d | none |

---

## Enrichment Status

| Group | Tickets | Enrichment Agent | Status |
|-------|---------|------------------|--------|
| Design specs | TC-001, TC-002, TC-006, TC-007 | architect | pending |
| Schema/config | TC-009, TC-009a, TC-014 | architect + go-pro | **TC-009a created (review)** |
| Critical blockers | TC-017, TC-018, TC-019 | go-pro / react-pro | **created (review)** |
| Go binary core | TC-003, TC-004, TC-005, TC-008, TC-010, TC-011 | go-pro | **TC-008, TC-011 enriched (review)** |
| Slash commands | TC-012 | go-cli | **enriched (review)** |
| Orchestrator rewrites | TC-013, TC-020 | architect | **TC-013 enriched, TC-020 created (review)** |
| TUI concurrency | TC-015, TC-019 | react-pro | **TC-015 enriched, TC-019 created (review)** |
| PID lockfile | TC-016 | go-pro | **enriched (review)** |

---

## Dependency Graph

```
CRITICAL BLOCKERS (Week 1, parallel):
  TC-005 (verify CLI output) ──────────────────┐
  TC-009a (minimal templates) ─────────────────┐│
  TC-017 (validate Level 2) ──────────────────┐││
  TC-018 (Mozart interview) ──► TC-013        │││
                                              │││
PHASE 0-1 (parallel with blockers):          │││
  TC-001 (permission flags)  ─────────────────┤││
  TC-002 (mutex)  ────────────────────────────┤││
  TC-006 (projectRoot)  ─────────────────────┤││
  TC-014 (cli_flags in agents-index)  ───────┤││
  TC-004 (daemon pattern)  ─────────────────┐│││
  TC-007, TC-009 (full schemas)  ───────────┤│││
                                            ▼▼▼▼
                                     TC-008 (Go binary)
                                            │
                                            ├──► TC-003 (retry fix, inside binary)
                                            │
                                            ▼
                                     TC-010 (inter-wave Go binary)
                                            │
                                            ▼
                                     TC-011 (unit tests) ◄── TC-003
                                            │
                                            ▼
                                     TC-012 (slash commands)
                                            │
                                            ▼
                                     TC-013 (orchestrator rewrites) ◄── TC-018, TC-020

Independent:
  TC-019 (SDK investigation) ──► TC-015 (TUI concurrency)
  TC-016 (duplicate launch prevention) ──► TC-008
  TC-020 (orchestrator design docs) ──► TC-013
```
