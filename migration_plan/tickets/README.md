# GOgent Phase 0 Tickets - Detailed Implementation Guide

**Version**: 1.1 FINAL
**Date**: 2026-01-15
**Total Tickets**: 56 (1 pre-work + 55 implementation)
**Estimated Duration**: 3 weeks + 1 day pre-work (~128 hours)
**Status**: ✅ Staff Architect Approved | 🔄 Implementation: Week 1 (3/9 tickets complete)

---

## 📍 Current Status

**Last Updated**: 2026-01-16
**Progress**: GOgent-002 complete (schema v2.2.0)
**Next**: GOgent-003 (AgentIndex structs)

👉 **[See SESSION-HANDOFF.md for current implementation status and next steps](SESSION-HANDOFF.md)**

---

## Quick Navigation

| File | Tickets | Time | Description |
|------|---------|------|-------------|
| [TICKET-TEMPLATE.md](TICKET-TEMPLATE.md) | N/A | N/A | **READ THIS FIRST** - Required structure for all tickets |
| [00-overview.md](00-overview.md) | N/A | N/A | Testing strategy, rollback plan, error standards |
| [00-prework.md](00-prework.md) | GOgent-000 | 6h | **MUST COMPLETE BEFORE WEEK 1** - Baseline measurement |
| [01-week1-foundation-events.md](01-week1-foundation-events.md) | GOgent-001 to 009 | ~11h | Go setup, schema structs, event parsing |
| [02-week1-overrides-permissions.md](02-week1-overrides-permissions.md) | GOgent-010 to 019 | ~13h | Escape hatches, tool permissions, complexity |
| [03-week1-validation-cli.md](03-week1-validation-cli.md) | GOgent-020 to 025 | ~12h | Task validation, opus blocking, CLI build |
| [04-week2-session-archive.md](04-week2-session-archive.md) | GOgent-026 to 033 | ~12h | session-archive hook translation |
| [05-week2-sharp-edge-memory.md](05-week2-sharp-edge-memory.md) | GOgent-034 to 040 | ~12h | sharp-edge-detector hook translation |
| [06-week3-integration-tests.md](06-week3-integration-tests.md) | GOgent-041 to 047 | ~14h | Integration tests, corpus validation |
| [07-week3-deployment-cutover.md](07-week3-deployment-cutover.md) | GOgent-048 to 055 | ~14h | Benchmarking, installation, cutover |

---

## How to Use This Directory

### For Contractors

1. **Start Here**: Read [TICKET-TEMPLATE.md](TICKET-TEMPLATE.md) to understand required structure
2. **Review Standards**: Read [00-overview.md](00-overview.md) for testing, error handling, rollback
3. **Pre-Work**: Complete [00-prework.md](00-prework.md) (GOgent-000) BEFORE starting Week 1
4. **Sequential Implementation**: Follow files in order (01 → 02 → 03 → 04 → 05 → 06 → 07)
5. **Dependencies**: Check each ticket's "Dependencies" field before starting
6. **Acceptance Criteria**: Every ticket has testable acceptance criteria - check ALL boxes
7. **Tests**: Run `go test ./...` after EACH ticket completion

### For Reviewers

1. **Template Compliance**: Check tickets against [TICKET-TEMPLATE.md](TICKET-TEMPLATE.md)
2. **No Shortcuts**: Verify NO "implement logic here", NO "omitted for brevity"
3. **Complete Code**: All code blocks should be copy-paste ready
4. **Error Standards**: All errors follow `[component] What. Why. How.` format
5. **XDG Paths**: No hardcoded `/tmp` paths (M-2 fix)
6. **STDIN Timeouts**: All hook STDIN reads have timeout (M-6 fix)

### For Project Managers

- **Critical Path**: GOgent-000 → Week 1 → Week 2 → Week 3 (sequential)
- **Checkpoints**: End of Week 1, End of Week 2, Mid-Week 3 (GO/NO-GO)
- **Rollback**: See [00-overview.md](00-overview.md#rollback-plan)
- **Success Criteria**: See [00-overview.md](00-overview.md#success-criteria)

---

## Dependency Graph

```
GOgent-000 (Pre-Work: Baseline + Corpus)
    ↓
GOgent-001 (Go Module Init)
    ↓
GOgent-002 (Schema Structs) → GOgent-002b (Complete Structs)
    ↓                              ↓
GOgent-003 (Agent Structs)          ↓
    ↓                              ↓
GOgent-004a (Config Loader) ←──────┘
    ↓
GOgent-006 (Event Structs)
    ↓
GOgent-007 (Event Parser)
    ↓
GOgent-008 (Parser Tests) → GOgent-008b (Corpus Capture)
    ↓                           ↓
GOgent-009 (Real Event Tests) ←─┘
    ↓
GOgent-004c (Complete Config Tests - after event parsing)
    ↓
GOgent-010 (Override Parsing) → GOgent-011 (Violation Logging) → GOgent-012 (Integration Test)
    ↓
GOgent-013 (Scout Metrics) → GOgent-014 (Freshness) → GOgent-015 (Tier Update) → GOgent-016 (Tests)
    ↓
GOgent-017 (Tool Permissions) → GOgent-018 (Wildcard) → GOgent-019 (Tests)
    ↓
GOgent-020 (Opus Blocking) → GOgent-021 (Model Mismatch) → GOgent-022 (Ceiling) → GOgent-023 (Subagent Type)
    ↓
GOgent-024 (Task Tests) → GOgent-024b (Wire Orchestrator) → GOgent-025 (Build CLI)
    ↓
[Week 2: Session Archive - GOgent-026 to 033]
    ↓
[Week 2: Sharp Edge - GOgent-034 to 040]
    ↓
[Week 3: Integration - GOgent-041 to 047]
    ↓
[Week 3: Deployment - GOgent-048 to 055]
```

---

## Timeline Summary

| Week | Phase | Tickets | Hours | Key Deliverables |
|------|-------|---------|-------|------------------|
| Pre  | Baseline | 1 (GOgent-000) | 6h | Performance baseline, 100-event corpus |
| 1    | Routing Translation | 25 (001-025) | 36h | gogent-validate binary, complete routing logic |
| 2    | Session/Memory | 15 (026-040) | 24h | gogent-archive, gogent-sharp-edge binaries |
| 3    | Testing/Cutover | 15 (041-055) | 28h | Integration tests, benchmarks, production cutover |

**Total**: 3 weeks + 1 day = ~94 hours implementation + 34 hours testing/QA = **128 hours**

---

## Critical Milestones

### ✅ Pre-Work Complete (Before Week 1)
- [ ] GOgent-000: Baseline measured, corpus captured (100 events)
- [ ] `~/gogent-baseline/BASELINE.md` exists with latency numbers
- [ ] `test/fixtures/event-corpus.json` exists

### 🔄 Week 1 Checkpoint (Friday) - IN PROGRESS
- [ ] GOgent-001 to 025 complete
  - [x] GOgent-001: Go module initialized
  - [x] GOgent-002: Schema structs complete (v2.2.0, all 23 types)
  - [x] GOgent-002b: Merged into GOgent-002
  - [x] GOgent-002c: Semantic validation (part of GOgent-002)
  - [ ] GOgent-003: AgentIndex structs (NEXT)
  - [ ] GOgent-004a: Config loader
  - [ ] GOgent-006-009: Event parsing
  - [ ] GOgent-010-025: Validation logic
- [ ] `gogent-validate` binary compiles and runs
- [x] Week 1 tests pass for completed tickets (`go test ./pkg/routing`)
- [ ] Event corpus tests pass (100% success rate)

### ✅ Week 2 Checkpoint (Friday)
- [ ] GOgent-026 to 040 complete
- [ ] `gogent-archive` and `gogent-sharp-edge` binaries compile
- [ ] All Week 2 unit tests pass
- [ ] Session archival tested with real JSONL

### ✅ Week 3 Mid-Point (Wednesday - GO/NO-GO Decision)
- [ ] GOgent-041 to 047 complete (integration tests)
- [ ] 100-event corpus regression test: 100% match Bash output
- [ ] Performance benchmark: ≤ baseline from GOgent-000
- [ ] **Decision**: Proceed to cutover or rollback?

### ✅ Week 3 Complete (Friday - Production Cutover)
- [ ] GOgent-048 to 055 complete
- [ ] Installation script tested on clean system
- [ ] Parallel testing (24hrs) successful
- [ ] Hooks switched to Go binaries
- [ ] Rollback plan tested and documented

---

## Conventions (Apply to ALL Tickets)

### Error Message Format
**Required**: `[component] What happened. Why it was blocked/failed. How to fix.`

**Examples**:
- ✅ `[config] Failed to read routing schema at ~/.claude/routing-schema.json: file not found. Ensure .claude/ directory exists.`
- ✅ `[validate-routing] Task(opus) blocked. Einstein requires GAP document workflow for cost control. Generate GAP: .claude/tmp/einstein-gap-{timestamp}.md, then run /einstein.`
- ❌ `config error` (no context)
- ❌ `failed to load` (no guidance)

### File Paths (XDG Compliance)
**Priority**: `XDG_RUNTIME_DIR > XDG_CACHE_HOME > ~/.cache/gogent`

**Never** hardcode `/tmp` (fixes M-2 from critical review)

```go
func GetGOgentDir() string {
    if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
        return filepath.Join(xdg, "gogent")
    }
    if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
        return filepath.Join(xdg, "gogent")
    }
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".cache", "gogent")
}
```

### STDIN Timeout
**All** hooks reading STDIN MUST have 5-second timeout (fixes M-6)

```go
func ParseToolEvent(r io.Reader, timeout time.Duration) (*ToolEvent, error) {
    ch := make(chan result, 1)
    go func() {
        data, err := io.ReadAll(r)
        ch <- result{data, err}
    }()

    select {
    case res := <-ch:
        return parseData(res.data)
    case <-time.After(timeout):
        return nil, fmt.Errorf("[event-parser] STDIN read timeout after %v. Hook may be stuck.", timeout)
    }
}
```

### Test Standards
- **Coverage**: ≥80% per package
- **Naming**: `TestFunctionName_Scenario`
- **Test Types**: Valid input, invalid input, edge cases, error conditions
- **Run After Each Ticket**: `go test ./...`

---

## Quick Find Reference

### Tickets by Category

**Foundation & Setup**:
- GOgent-000: Baseline measurement (pre-work)
- GOgent-001: Go module initialization
- GOgent-002/002b: Schema struct definitions

**Event Parsing**:
- GOgent-006: Event structs
- GOgent-007: Event parser with timeout
- GOgent-008/008b/009: Event parsing tests + corpus

**Routing Logic**:
- GOgent-010-012: Override flags (escape hatches)
- GOgent-013-016: Complexity routing
- GOgent-017-019: Tool permissions
- GOgent-020-023: Task validation (opus blocking, ceiling, subagent_type)

**CLI Build**:
- GOgent-025: Build gogent-validate binary

**Session Management**:
- GOgent-026-033: session-archive translation

**Memory/Sharp Edges**:
- GOgent-034-040: sharp-edge-detector translation

**Testing & Deployment**:
- GOgent-041-047: Integration tests
- GOgent-048-055: Benchmarking, installation, cutover

---

## File Size Guide

Each ticket file is sized for easy consumption:
- **~6-10 tickets** per file
- **~1500-2500 lines** per file (including code blocks)
- **~20-40 pages** when printed

This ensures:
- ✅ Easy to navigate (not overwhelming)
- ✅ Can review one file in 1-2 hours
- ✅ Printable for offline review
- ✅ Git-friendly diffs

---

## Quality Checklist (Before Contractor Handoff)

### Template Compliance
- [ ] Every ticket follows [TICKET-TEMPLATE.md](TICKET-TEMPLATE.md) structure
- [ ] No "implement logic here" placeholders
- [ ] No "omitted for brevity" shortcuts
- [ ] All imports are complete
- [ ] All code is copy-paste ready

### Standards Compliance
- [ ] All error messages follow `[component] What. Why. How.` format
- [ ] All file paths use XDG compliance (no `/tmp`)
- [ ] All STDIN reads have timeout
- [ ] All tests have ≥80% coverage targets

### Completeness
- [ ] All 56 tickets have full implementation detail
- [ ] All dependencies are listed correctly
- [ ] All acceptance criteria are specific and testable
- [ ] All "Why This Matters" sections provide context

---

## Support & References

### Primary Documents
- **Critical Review**: [../CRITICAL_REVIEW.md](../CRITICAL_REVIEW.md) - Issues that drove design decisions
- **Migration Plan**: [../gogent_migration_plan_v3_FINAL.md](../gogent_migration_plan_v3_FINAL.md) - Architecture and strategy
- **Ticket Index**: [../gogent_plan_tickets_v3_phase0_FINAL.md](../gogent_plan_tickets_v3_phase0_FINAL.md) - High-level overview

### Bash Reference Implementations
- `~/.claude/hooks/validate-routing.sh` (401 lines) - Source for GOgent-001 to 025
- `~/.claude/hooks/session-archive.sh` (111 lines) - Source for GOgent-026 to 033
- `~/.claude/hooks/sharp-edge-detector.sh` (105 lines) - Source for GOgent-034 to 040

### Configuration Files
- `~/.claude/routing-schema.json` - Routing rules (source of truth)
- `~/.claude/agents/agents-index.json` - Agent definitions

---

## FAQ

**Q: Can I skip GOgent-000 (pre-work)?**
A: No. Without baseline and corpus, you cannot verify Go doesn't regress performance or output.

**Q: Can I implement tickets out of order?**
A: No. Dependencies are strict. GOgent-002 requires GOgent-001, GOgent-007 requires GOgent-006, etc.

**Q: What if a ticket's time estimate is wrong?**
A: Estimates are for 1-2 hour chunks. If a ticket takes >3 hours, it may be blocked by missing dependency or misunderstanding. Review acceptance criteria and ask questions.

**Q: Can I combine multiple tickets?**
A: No. Each ticket has specific acceptance criteria that must be validated independently.

**Q: What does "copy-paste ready" mean?**
A: Contractor should be able to copy code from ticket directly into file without modifications. No placeholders, no TODOs.

**Q: How do I handle "yourusername" in import paths?**
A: Replace with your GitHub username during GOgent-001 (Go module init). All tickets reference this variable.

---

**Version History**:
- v1.0: Initial ticket breakdown (50 tickets)
- v1.1 FINAL: Applied critical review fixes (+6 tickets: 002b, 004c, 008b, 024b, 033, 048b)

**Last Updated**: 2026-01-15
**Status**: ✅ Ready for contractor assignment
