# Phase 0 Tickets - Navigation Index

**Status**: ✅ All 56 tickets fully detailed and ready for implementation
**Last Updated**: 2026-01-15
**Progress**: 0/56 complete (0%)

---

## Quick Start

### For Contractors/Developers

1. **Read the overview**: [00-overview.md](00-overview.md) - Standards, testing strategy, rollback plan
2. **Start with prework**: [00-prework.md](00-prework.md) - GOgent-000 baseline measurement (MUST complete first)
3. **Use workflow automation**: `./scripts/ticket-workflow.sh next`
4. **Follow the prompts**: See [WORKFLOW.md](WORKFLOW.md) for Claude prompt templates

### Quick Commands

```bash
# Find next available ticket
./scripts/ticket-workflow.sh next

# List all pending tickets
./scripts/ticket-workflow.sh list

# Check progress
./scripts/ticket-workflow.sh status

# Start specific ticket
./scripts/ticket-workflow.sh GOgent-042
```

---

## File Structure

```
tickets/
├── INDEX.md                           ← YOU ARE HERE (navigation)
├── README.md                          ← Detailed overview and conventions
├── TICKET-TEMPLATE.md                 ← Template for any future tickets
├── WORKFLOW.md                        ← Workflow automation guide
├── PROGRESS.md                        ← Completion tracking
│
├── tickets-index.json                 ← Machine-readable ticket database
├── dependency-graph.mmd               ← Mermaid dependency visualization
│
├── 00-overview.md                     ← Cross-cutting standards (16KB)
├── 00-prework.md                      ← GOgent-000: Baseline (13KB)
│
├── 01-week1-foundation-events.md      ← GOgent-001 to 009 (32KB, 9 tickets)
├── 02-week1-overrides-permissions.md  ← GOgent-010 to 019 (45KB, 10 tickets)
├── 03-week1-validation-cli.md         ← GOgent-020 to 025 (38KB, 6 tickets)
│
├── 04-week2-session-archive.md        ← GOgent-026 to 033 (35KB, 8 tickets)
├── 05-week2-sharp-edge-memory.md      ← GOgent-034 to 040 (32KB, 7 tickets)
│
├── 06-week3-integration-tests.md      ← GOgent-041 to 047 (42KB, 7 tickets)
└── 07-week3-deployment-cutover.md     ← GOgent-048 to 055 (48KB, 8 tickets)
```

---

## Tickets by Week

### Week 0: Pre-Work (1 ticket, ~2 hours)

**File**: [00-prework.md](00-prework.md)

| Ticket | Title | Time | Priority |
|--------|-------|------|----------|
| **GOgent-000** | Baseline Measurement & Corpus Capture | 2h | Critical |

**⚠️ MUST complete GOgent-000 before starting any other tickets.**

---

### Week 1: Foundation & Validation (25 tickets, ~40 hours)

#### Week 1 Days 1-2: Foundation & Event Parsing (9 tickets)

**File**: [01-week1-foundation-events.md](01-week1-foundation-events.md)

| Ticket | Title | Time | Priority | Dependencies |
|--------|-------|------|----------|--------------|
| GOgent-001 | Go Module Setup | 1h | Critical | GOgent-000 |
| GOgent-002 | STDIN Timeout Reading | 1h | High | GOgent-001 |
| GOgent-003 | Parse ToolEvent JSON | 1.5h | Critical | GOgent-001, 002 |
| GOgent-004a | Load Routing Schema | 2h | Critical | GOgent-001, 003 |
| GOgent-004b | Read Current Tier | 1h | High | GOgent-004a |
| GOgent-005 | Parse Task Input | 1h | High | GOgent-002, 003 |
| GOgent-006 | XDG Path Resolution | 1h | Medium | GOgent-000 |
| GOgent-007 | Tool Permission Check | 1.5h | Critical | GOgent-004a, 004b |
| GOgent-008a | Hook Response JSON | 1h | Critical | GOgent-002 |
| GOgent-008b | Capture Event Corpus | 1h | High | GOgent-000, 002 |
| GOgent-009 | Error Message Format | 0.5h | Low | - |

**Total**: 12.5 hours

#### Week 1 Days 3-5: Overrides & Permissions (10 tickets)

**File**: [02-week1-overrides-permissions.md](02-week1-overrides-permissions.md)

| Ticket | Title | Time | Priority | Dependencies |
|--------|-------|------|----------|--------------|
| GOgent-010 | Parse Override Flags | 1.5h | Medium | - |
| GOgent-011 | Violation Logging | 1.5h | High | GOgent-010 |
| GOgent-012 | Escape Hatch Tests | 1h | Medium | GOgent-011 |
| GOgent-013 | Scout Metrics Loading | 1.5h | Medium | GOgent-003, 004a |
| GOgent-014 | Metrics Freshness Check | 1h | Low | GOgent-013 |
| GOgent-015 | Tier Update from Complexity | 1h | Medium | GOgent-013, 014 |
| GOgent-016 | Complexity Routing Tests | 1h | Medium | GOgent-015 |
| GOgent-017 | Tool Permission Checks | 1.5h | Critical | GOgent-007 |
| GOgent-018 | Wildcard Tools Handling | 1h | Medium | GOgent-017 |
| GOgent-019 | Tool Permission Tests | 1h | High | GOgent-017, 018 |

**Total**: 12 hours

#### Week 1 Days 6-7: Task Validation & CLI (6 tickets)

**File**: [03-week1-validation-cli.md](03-week1-validation-cli.md)

| Ticket | Title | Time | Priority | Dependencies |
|--------|-------|------|----------|--------------|
| GOgent-020 | Einstein/Opus Blocking | 1.5h | Critical | GOgent-005 |
| GOgent-021 | Model Mismatch Warnings | 1h | Low | - |
| GOgent-022 | Delegation Ceiling Enforcement | 1.5h | Medium | - |
| GOgent-023 | Subagent_type Validation | 1.5h | High | GOgent-005 |
| GOgent-024 | Task Validation Tests | 2h | High | GOgent-020, 022, 023 |
| GOgent-024b | Wire Validation Orchestrator | 2h | Critical | GOgent-007, 008a, 020, 024 |
| GOgent-025 | Build gogent-validate CLI | 1h | Critical | GOgent-024b |

**Total**: 10.5 hours

**Week 1 Total**: 35 hours

---

### Week 2: Session & Memory Management (15 tickets, ~28 hours)

#### Week 2 Days 1-3: Session Archive Translation (8 tickets)

**File**: [04-week2-session-archive.md](04-week2-session-archive.md)

| Ticket | Title | Time | Priority | Dependencies |
|--------|-------|------|----------|--------------|
| GOgent-026 | Parse SessionEnd Event | 1h | High | - |
| GOgent-027 | Collect Session Metrics | 2h | High | GOgent-006, 026 |
| GOgent-028 | Generate Handoff Document | 2h | Critical | GOgent-026, 027 |
| GOgent-029 | Format Pending Learnings | 1h | Medium | - |
| GOgent-030 | Format Violations Summary | 1.5h | Medium | GOgent-011 |
| GOgent-031 | Archive Session Files | 1.5h | High | - |
| GOgent-032 | Session Archive Integration Test | 1.5h | High | GOgent-028, 029, 030, 031 |
| GOgent-033 | Build gogent-archive CLI | 1h | Critical | GOgent-032 |

**Total**: 11.5 hours

#### Week 2 Days 4-6: Sharp Edge Detection (7 tickets)

**File**: [05-week2-sharp-edge-memory.md](05-week2-sharp-edge-memory.md)

| Ticket | Title | Time | Priority | Dependencies |
|--------|-------|------|----------|--------------|
| GOgent-034 | Parse PostToolUse Event | 1h | High | - |
| GOgent-035 | Failure Detection Logic | 2h | Critical | GOgent-034 |
| GOgent-036 | Consecutive Failure Tracking | 2h | Critical | GOgent-034, 035 |
| GOgent-037 | Sharp Edge Capture | 1.5h | High | GOgent-036 |
| GOgent-038 | Hook Response Generation | 1.5h | Critical | GOgent-008a, 036 |
| GOgent-039 | Sharp Edge Integration Tests | 2h | High | GOgent-037, 038 |
| GOgent-040 | Build gogent-sharp-edge CLI | 1h | Critical | GOgent-039 |

**Total**: 11 hours

**Week 2 Total**: 22.5 hours

---

### Week 3: Testing & Deployment (15 tickets, ~26 hours)

#### Week 3 Days 1-3: Integration & Regression Tests (7 tickets)

**File**: [06-week3-integration-tests.md](06-week3-integration-tests.md)

| Ticket | Title | Time | Priority | Dependencies |
|--------|-------|------|----------|--------------|
| GOgent-004c | Config Circular Dependency Tests | 1h | Medium | GOgent-004a, 004b |
| GOgent-041 | Test Harness for Corpus Replay | 2h | Critical | GOgent-000, 008b |
| GOgent-042 | Integration Tests for validate-routing | 2h | High | GOgent-025, 041 |
| GOgent-043 | Integration Tests for session-archive | 1.5h | High | GOgent-033, 041 |
| GOgent-044 | Integration Tests for sharp-edge-detector | 1.5h | High | GOgent-040, 041 |
| GOgent-045 | Performance Benchmarks | 2h | High | GOgent-041, 042, 043, 044 |
| GOgent-046 | End-to-End Workflow Tests | 2h | High | GOgent-042, 043, 044 |
| GOgent-047 | Regression Tests (Go vs Bash) | 2h | Critical | GOgent-000, 041 |

**Total**: 14 hours

#### Week 3 Days 4-7: Deployment & Cutover (8 tickets)

**File**: [07-week3-deployment-cutover.md](07-week3-deployment-cutover.md)

| Ticket | Title | Time | Priority | Dependencies |
|--------|-------|------|----------|--------------|
| GOgent-048 | Installation Script | 2h | Critical | (all above complete) |
| GOgent-048b | WSL2 Compatibility Testing | 1.5h | Medium | GOgent-048 |
| GOgent-049 | Parallel Testing (24hrs) | 2h | Critical | GOgent-048 |
| GOgent-050 | Cutover Decision Workflow | 1h | Critical | GOgent-049 |
| GOgent-051 | Symlink Cutover Script | 1h | Critical | GOgent-048, 050 |
| GOgent-052 | Rollback Script & Testing | 1h | Critical | GOgent-051 |
| GOgent-053 | Documentation Updates | 1.5h | High | GOgent-051 |
| GOgent-054 | Performance Regression Monitoring | 1h | High | GOgent-051 |
| GOgent-055 | Post-Cutover Validation Checklist | 1h | Critical | GOgent-051 |

**Total**: 12 hours

**Week 3 Total**: 26 hours

---

## Total Time Estimate

| Phase | Tickets | Hours | Days (8hr) |
|-------|---------|-------|------------|
| Pre-Work | 1 | 2 | 0.25 |
| Week 1 | 25 | 35 | 4.4 |
| Week 2 | 15 | 22.5 | 2.8 |
| Week 3 | 15 | 26 | 3.3 |
| **Total** | **56** | **85.5** | **10.7** |

**Actual Duration**: 3 weeks (15 work days) with buffer for testing, reviews, and integration.

---

## Critical Path

The following tickets MUST be completed before others can begin:

```
GOgent-000 (baseline)
  ↓
GOgent-001 (Go module)
  ↓
GOgent-002, 003, 004a (foundation)
  ↓
GOgent-004b, 007, 008a (core validation)
  ↓
GOgent-024b (orchestrator)
  ↓
GOgent-025 (gogent-validate CLI)
  ↓
GOgent-033, 040 (other CLIs)
  ↓
GOgent-041 (test harness)
  ↓
GOgent-042, 043, 044, 047 (integration/regression)
  ↓
GOgent-048, 049 (installation, parallel test)
  ↓
GOgent-050, 051 (cutover decision & execution)
```

**Total Critical Path**: ~18 tickets, ~30 hours

---

## Machine-Readable Data

### JSON Index

[tickets-index.json](tickets-index.json) contains:
- All 56 tickets with complete metadata
- Dependency graph (machine-readable)
- Status tracking (pending/complete)
- File paths, branches, labels
- Time estimates and priorities

**Use for**:
- Automation scripts
- Progress tracking
- Dependency resolution
- PR generation

### Dependency Graph

[dependency-graph.mmd](dependency-graph.mmd) - Mermaid diagram showing:
- All ticket dependencies
- Critical path highlighted
- Color-coded by priority

**Render with**:
```bash
mmdc -i dependency-graph.mmd -o dependency-graph.png
```

---

## Workflow Automation

### Interactive Workflow

1. **Find next ticket**: `./scripts/ticket-workflow.sh next`
2. **Create branch**: (automatic)
3. **Get Claude prompt**: (automatic - copy/paste to Claude)
4. **Implement ticket**: Follow prompt instructions
5. **Run tests**: `go test ./...`
6. **Commit**: Standard format (see WORKFLOW.md)
7. **Create PR**: `gh pr create --fill`
8. **Mark complete**: Update tickets-index.json

### Automated Commands

```bash
# List available tickets
./scripts/ticket-workflow.sh list

# Check progress
./scripts/ticket-workflow.sh status

# Start specific ticket
./scripts/ticket-workflow.sh GOgent-025
```

**Full Documentation**: See [WORKFLOW.md](WORKFLOW.md)

---

## Quality Standards

All tickets must meet these standards (enforced in acceptance criteria):

### Code Quality
- ✅ No placeholders or "omitted for brevity"
- ✅ Complete, copy-paste-ready implementation
- ✅ Error format: `[component] What. Why. How to fix.`
- ✅ XDG-compliant paths (no `/tmp` hardcoding)
- ✅ STDIN timeout handling (5s default)

### Testing
- ✅ Test coverage ≥80% for new packages
- ✅ All tests pass: `go test ./...`
- ✅ Integration tests for complete workflows
- ✅ Regression tests (Go vs Bash comparison)

### Documentation
- ✅ Each ticket has "Why This Matters" section
- ✅ Acceptance criteria checklist
- ✅ Cross-references to related tickets
- ✅ Complete code comments

**Full Standards**: See [00-overview.md](00-overview.md)

---

## Progress Tracking

### Current Status

**Phase 0 Implementation**: Not started
**Completed Tickets**: 0 / 56 (0%)
**Estimated Completion**: 3 weeks from start

### By Week

- Week 0 (Prework): 0 / 1 (0%)
- Week 1: 0 / 25 (0%)
- Week 2: 0 / 15 (0%)
- Week 3: 0 / 15 (0%)

### Critical Path

- Critical tickets: 0 / 18 completed (0%)

**Live Progress**: See [PROGRESS.md](PROGRESS.md)

---

## Getting Started

### Prerequisites

- Go 1.21+
- Git
- jq (for automation scripts)
- Access to Claude Code with hooks enabled

### First Steps

1. **Read overview**: [00-overview.md](00-overview.md)
2. **Read template**: [TICKET-TEMPLATE.md](TICKET-TEMPLATE.md)
3. **Complete prework**: [00-prework.md](00-prework.md) - GOgent-000
4. **Start Week 1**: [01-week1-foundation-events.md](01-week1-foundation-events.md) - GOgent-001

### For Claude Sessions

Use the workflow automation:

```bash
./scripts/ticket-workflow.sh next
```

This will:
1. Find the next available ticket (dependencies met)
2. Create Git branch
3. Generate Claude prompt with full ticket context
4. List files to create

Copy the prompt into Claude and follow the instructions.

---

## Support & Questions

- **Ticket Details**: Read the detailed ticket files (00-07 series)
- **Workflow Help**: See [WORKFLOW.md](WORKFLOW.md)
- **Standards**: See [00-overview.md](00-overview.md)
- **Template**: See [TICKET-TEMPLATE.md](TICKET-TEMPLATE.md)
- **Progress**: See [PROGRESS.md](PROGRESS.md)

---

## Document History

| Date | Version | Changes |
|------|---------|---------|
| 2026-01-15 | 1.0 | Initial navigation index created |
| 2026-01-15 | 1.1 | Added automation scripts and workflow |

---

**Status**: ✅ Ready for Implementation
**Maintainer**: Project Team
**Last Review**: 2026-01-15
