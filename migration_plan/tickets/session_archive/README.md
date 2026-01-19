# Session Archive Ticket Files

This directory contains individual ticket specification files for GOgent-026 through GOgent-040, including base implementation tickets and enhancement tickets.

## File Organization

### Base Implementation Tickets (15 files)
Week 2 session archive and sharp edge detection translation from Bash to Go:

**Session Archive (026-033)**:
- `026.md` - Define Session Event Structs
- `027.md` - Implement Session Metrics Collection
- `028.md` - Implement Handoff Document Generation
- `029.md` - Format Pending Learnings
- `030.md` - Format Routing Violations Summary
- `031.md` - Implement File Archival
- `032.md` - Session Archive Integration Tests
- `033.md` - Build gogent-archive CLI

**Sharp Edge Detection (034-040)**:
- `034.md` - Define PostToolUse Event Structs
- `035.md` - Implement Failure Detection
- `036.md` - Implement Consecutive Failure Tracking
- `037.md` - Implement Sharp Edge Capture
- `038.md` - Implement Hook Response Generation
- `039.md` - Sharp Edge Detection Integration Tests
- `040.md` - Build gogent-sharp-edge CLI

### Enhancement Tickets (17 files)
Critical and major enhancements addressing GAP analysis findings:

**Work Package 1: Transcript Parsing (Phase 2 - Critical)**:
- `027b.md` - Implement ParseTranscript() for Semantic Analysis
- `027c.md` - Implement AnalyzeToolDistribution()
- `027d.md` - Implement DetectPhases() for Session Characterization

**Work Package 2: Adaptive Handoff (Phase 2 - Critical)**:
- `028b.md` - Implement GenerateAdaptiveHandoff()
- `028c.md` - Implement DetectWorkInProgress()
- `028d.md` - Implement GenerateResumeGuidance()

**Work Package 3: Enriched Sharp Edges (Phase 2 - Critical)**:
- `029b.md` - Capture Full Error Messages in Sharp Edges
- `037b.md` - Extract 5-Line Code Context Window
- `037c.md` - Log Attempted Changes from Tool Input

**Work Package 4: Violation Clustering (Phase 3 - Major)**:
- `030b.md` - Cluster Violations by Type
- `030c.md` - Cluster Violations by Agent
- `030d.md` - Analyze Violation Temporal Trends

**Work Package 5: Refined Failure Tracking (Phase 3 - Major)**:
- `036b.md` - Track Failures by File + Error Type
- `036c.md` - Extract Function Name from Stack Traces (Optional)

**Work Package 6: Pattern-Aware Responses (Phase 3 - Major)**:
- `038b.md` - Load and Index Sharp-Edges YAML
- `038c.md` - Implement Pattern Similarity Matching
- `038d.md` - Inject Remediation Suggestions in Blocking Response

## Implementation Phases

### Phase 1: Base Implementation (24 hours)
Implement tickets 026-040 as-written using `/ticket` workflow.

**Status**: Ready for implementation
**Priority**: CRITICAL - implement immediately

### Phase 2: Critical Enhancements (7 hours)
Implement Work Packages 1-3 (027b-d, 028b-d, 029b, 037b-c).

**Dependencies**: Phase 1 complete
**Value**: +200% improvement in handoff quality
**Status**: Recommended for Week 2 completion

### Phase 3: Major Enhancements (4.5 hours)
Implement Work Packages 4-6 (030b-d, 036b-c, 038b-d).

**Dependencies**: Phase 2 complete
**Value**: Prevents false positives, enables pattern-aware responses
**Status**: Recommended for early Week 3

## Total Effort Estimate

| Phase | Tickets | Time | Cumulative |
|-------|---------|------|------------|
| Phase 1 (Base) | 15 tickets | 24h | 24h |
| Phase 2 (Critical) | 9 tickets | 7h | 31h |
| Phase 3 (Major) | 8 tickets | 4.5h | 35.5h |
| **Total** | **32 tickets** | **35.5h** | - |

## Usage with /ticket Workflow

All ticket files follow the GOgent-Fortress ticket template structure and are compatible with the `/ticket` skill.

To begin implementation:
```bash
/ticket next  # Will start with 026.md
```

After each ticket completion, run `/ticket next` to proceed to the next pending ticket in dependency order.

## References

- **Source**: `migration_plan/tickets/04-week2-session-archive.md` (base tickets 026-033)
- **Source**: `migration_plan/tickets/05-week2-sharp-edge-memory.md` (base tickets 034-040)
- **Source**: `migration_plan/gap-analysis-week2.md` (enhancement specifications)
- **Analysis**: `migration_plan/tickets/gogent-026-040-review.md` (architectural review)

---

**Created**: 2026-01-19
**Status**: All 32 tickets ready for implementation
