# Einstein Analysis: 029 Series Rescoping

**Resolved**: 2026-01-21
**GAP Source**: `.claude/tmp/einstein-gap-029-rescope-analysis.md`

---

## Executive Summary

The 029 series tickets are **architecturally compatible** with 028k-o but need **moderate rescoping** to clarify integration patterns. Key decisions: (1) Use schema v1.1 with additive-only changes, (2) CLI artifact commands run **parallel** to existing session commands (not replacing), (3) Follow the existing dual-write pattern where artifacts accumulate in separate JSONL files and get embedded into per-session handoffs. Most tickets need only documentation clarification; 029b and 029d need structural changes to their CLI design.

## Root Cause Analysis

The 029 series was designed with implicit assumptions that weren't aligned with how 028k-o actually landed:

1. **Assumption**: CLI would pivot to artifact-centric
   **Reality**: CLI is session-centric with artifact filtering flags

2. **Assumption**: Schema extension requires version bump
   **Reality**: Optional fields with `omitempty` are backward-compatible within v1.0

3. **Assumption**: Separate files vs embedded is a choice
   **Reality**: The existing system already does BOTH (dual-write pattern)

These aren't conflicts—they're integration points that weren't explicitly documented.

## Recommended Solutions

### Q1: Schema Strategy → **v1.1 (Additive)**

**Decision**: Use schema v1.1 for the combined 029 series changes.

**Rationale**:
- All proposed changes are additive (new optional fields, new array types)
- JSON `omitempty` ensures old readers ignore new fields
- New readers handle old data (missing fields = zero values)
- ADR-028's `migrateHandoff()` is only needed for breaking changes

**Implementation**:
```go
// pkg/session/handoff.go
const HandoffSchemaVersion = "1.1"  // Bump when 029c lands (not before)

// migrateHandoff() additions:
case "1.0":
    // v1.0 → v1.1: No migration needed, just defaults
    var h Handoff
    json.Unmarshal(data, &h)
    h.SchemaVersion = "1.1"  // Upgrade marker
    // New arrays default to empty, new fields default to zero
    return &h, nil
```

**When to bump**:
- 029, 029a: Stay on v1.0 (optional field additions)
- 029c: Bump to v1.1 (structural extension with new artifact arrays)

### Q2: CLI Architecture → **Option A: Parallel Namespaces**

**Decision**: Artifact commands coexist alongside session commands.

**Final CLI Structure**:
```
gogent-archive                         # Hook mode (unchanged)
gogent-archive list [--filters]        # Session list (unchanged)
gogent-archive show <id>               # Session detail (unchanged)
gogent-archive stats                   # Session aggregates (unchanged)
gogent-archive sharp-edges [--filters] # NEW: Cross-session artifact view
gogent-archive user-intents [--filters]# NEW: Cross-session artifact view
gogent-archive decisions [--filters]   # NEW: Cross-session artifact view
gogent-archive preferences [--filters] # NEW: Cross-session artifact view
gogent-archive performance [--filters] # NEW: Cross-session artifact view
gogent-archive aggregate [--force]     # NEW: 029f weekly rotation
```

### Q3: File Architecture → **Option C: Hybrid (Already in Use)**

**Decision**: Continue the existing dual-write pattern.

**Extended Pattern** (029 series):
```
Session runtime:
  Hooks append to → pending-learnings.jsonl   (existing)
                 → routing-violations.jsonl   (existing)
                 → user-intents.jsonl         (NEW - 029a)
                 → decisions.jsonl            (NEW - 029c)
                 → preferences.jsonl          (NEW - 029c)
                 → performance.jsonl          (NEW - 029c)

SessionEnd:
  gogent-archive reads ALL files
              ↓
  Embeds ALL artifacts into Handoff struct
              ↓
  Appends to handoffs.jsonl
              ↓
  Renders last-handoff.md (extended sections)

Weekly:
  gogent-aggregate rotates old files → archive/
```

### Q4: Ticket-by-Ticket Rescoping

| Ticket | Rescoping | Changes Required |
|--------|-----------|------------------|
| **029** | MINOR | Add `omitempty` tags to new SharpEdge fields. Clarify stays on v1.0. |
| **029a** | MINOR | Follow pending-learnings.jsonl pattern. Add `UserIntentsPath` to HandoffConfig. |
| **029b** | MODERATE | Rewrite as "parallel commands" not "replacement". |
| **029c** | MINOR | Add `omitempty` to new arrays. Bump to v1.1. |
| **029d** | MODERATE | Same as 029b—parallel commands pattern. |
| **029e** | NONE | Clean extension, no conflict. |
| **029f** | MINOR | Verify complete file list for aggregation. |

## Follow-Up Actions

- [ ] Update 029.md: Add schema compatibility notes, `omitempty` requirement
- [ ] Update 029a.md: Add file pattern documentation, HandoffConfig extension
- [ ] **Rewrite 029b.md**: Major reframe as parallel CLI commands
- [ ] Update 029c.md: Schema v1.1 marker, file path additions
- [ ] **Rewrite 029d.md**: Align with 029b parallel pattern
- [ ] Verify 029f.md: Complete file list for aggregation
- [ ] Update tickets-index.json: Mark rescoping complete

---

**Status**: RESOLVED
**Estimated Rescoping Effort**: 2-3 hours (mostly documentation updates)
