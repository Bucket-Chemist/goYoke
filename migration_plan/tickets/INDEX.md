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
./scripts/ticket-workflow.sh GOgent-095
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
├── 06-week3-load-routing-context.md   ← GOgent-056 to 062 (35KB, 7 tickets)
├── 07-week3-agent-workflow-hooks.md   ← GOgent-063 to 074 (55KB, 12 tickets)
├── 08-week4-advanced-enforcement.md   ← GOgent-075 to 086 (43KB, 12 tickets)
├── 09-week4-observability-remaining.md ← GOgent-087 to 093 (24KB, 7 tickets)
│
├── 10-week5-integration-tests.md      ← GOgent-094 to 100+ (42KB+, 13-14 tickets) ⚠️ REFACTORED
└── 11-week5-deployment-cutover.md     ← GOgent-101 to 108 (48KB+, 8 tickets) ⚠️ REFACTORED
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
| GOgent-094 | Test Harness for Corpus Replay | 2h | Critical | GOgent-000, 008b |
| GOgent-095 | Integration Tests for validate-routing | 2h | High | GOgent-025, 041 |
| GOgent-096 | Integration Tests for session-archive | 1.5h | High | GOgent-033, 041 |
| GOgent-097 | Integration Tests for sharp-edge-detector | 1.5h | High | GOgent-040, 041 |
| GOgent-098 | Performance Benchmarks | 2h | High | GOgent-094, 042, 043, 044 |
| GOgent-099 | End-to-End Workflow Tests | 2h | High | GOgent-095, 043, 044 |
| GOgent-100 | Regression Tests (Go vs Bash) | 2h | Critical | GOgent-000, 041 |

**Total**: 14 hours

#### Week 3 Days 4-7: Deployment & Cutover (8 tickets)

**File**: [07-week3-deployment-cutover.md](07-week3-deployment-cutover.md)

| Ticket | Title | Time | Priority | Dependencies |
|--------|-------|------|----------|--------------|
| GOgent-101 | Installation Script | 2h | Critical | (all above complete) |
| GOgent-101b | WSL2 Compatibility Testing | 1.5h | Medium | GOgent-101 |
| GOgent-102 | Parallel Testing (24hrs) | 2h | Critical | GOgent-101 |
| GOgent-103 | Cutover Decision Workflow | 1h | Critical | GOgent-102 |
| GOgent-104 | Symlink Cutover Script | 1h | Critical | GOgent-101, 050 |
| GOgent-105 | Rollback Script & Testing | 1h | Critical | GOgent-104 |
| GOgent-106 | Documentation Updates | 1.5h | High | GOgent-104 |
| GOgent-107 | Performance Regression Monitoring | 1h | High | GOgent-104 |
| GOgent-108 | Post-Cutover Validation Checklist | 1h | Critical | GOgent-104 |

**Total**: 12 hours

**Week 3 Total**: 26 hours

---

### Week 3-4: Additional Hook Implementations (38 tickets, ~57 hours)

#### Week 3 Days 3-4: Session Initialization (7 tickets)

**File**: [06-week3-load-routing-context.md](06-week3-load-routing-context.md)

| Ticket | Title | Time | Priority | Dependencies |
|--------|-------|------|----------|--------------|
| GOgent-056 | SessionStart Event Parsing | 1.5h | Critical | GOgent-002 |
| GOgent-057 | Routing Schema Loading | 1.5h | Critical | GOgent-004a |
| GOgent-058 | Handoff Document Loading | 1h | High | GOgent-056 |
| GOgent-059 | Pending Learnings & Git Integration | 1.5h | Medium | - |
| GOgent-060 | Project Type Detection | 1.5h | High | - |
| GOgent-061 | Session Context Response | 1.5h | Critical | GOgent-056-060 |
| GOgent-062 | Build gogent-load-context CLI | 1.5h | Critical | GOgent-061 |

**Total**: 11 hours

#### Week 3 Days 5-7 & Week 4 Days 1-2: Agent Workflow Hooks (12 tickets)

**File**: [07-week3-agent-workflow-hooks.md](07-week3-agent-workflow-hooks.md)

| Ticket | Title | Time | Priority | Dependencies |
|--------|-------|------|----------|--------------|
| GOgent-063 | SubagentStop Event Parsing | 1.5h | High | GOgent-002 |
| GOgent-064 | Agent Type Detection | 1.5h | Critical | GOgent-063 |
| GOgent-065 | Tier-Specific Response Templates | 2h | Critical | GOgent-064 |
| GOgent-066 | Decision Logging | 1h | Medium | GOgent-065 |
| GOgent-067 | Agent Endstate Integration Tests | 1.5h | High | GOgent-066 |
| GOgent-068 | Build gogent-agent-endstate CLI | 1.5h | Critical | GOgent-067 |
| GOgent-069 | Tool Counter Management | 1.5h | Critical | GOgent-056 |
| GOgent-070 | Routing Reminder Injection | 1h | High | GOgent-069 |
| GOgent-071 | Auto-Flush Logic | 2h | Critical | GOgent-070 |
| GOgent-072 | Archive Generation | 1.5h | Medium | GOgent-071 |
| GOgent-073 | Attention Gate Integration Tests | 1.5h | High | GOgent-072 |
| GOgent-074 | Build gogent-attention-gate CLI | 1.5h | Critical | GOgent-073 |

**Total**: 18 hours

#### Week 4 Days 3-6: Advanced Enforcement (12 tickets)

**File**: [08-week4-advanced-enforcement.md](08-week4-advanced-enforcement.md)

| Ticket | Title | Time | Priority | Dependencies |
|--------|-------|------|----------|--------------|
| GOgent-075 | Transcript Parsing | 1.5h | Critical | GOgent-002 |
| GOgent-076 | Background Task Detection | 2h | Critical | GOgent-075 |
| GOgent-077 | Task Collection Verification | 1.5h | Critical | GOgent-076 |
| GOgent-078 | Blocking Response Generation | 1h | High | GOgent-077 |
| GOgent-079 | Orchestrator Guard Integration Tests | 1.5h | High | GOgent-078 |
| GOgent-080 | Build gogent-orchestrator-guard CLI | 1.5h | Critical | GOgent-079 |
| GOgent-081 | PreToolUse Event Parsing | 1.5h | High | GOgent-002 |
| GOgent-082 | File Path Extraction | 1h | Medium | GOgent-081 |
| GOgent-083 | Enforcement Pattern Matching | 1.5h | Critical | GOgent-082 |
| GOgent-084 | Warning Response Generation | 1h | Medium | GOgent-083 |
| GOgent-085 | Doc Theater Integration Tests | 1.5h | High | GOgent-084 |
| GOgent-086 | Build gogent-doc-theater CLI | 1.5h | Critical | GOgent-085 |

**Total**: 18 hours

#### Week 4 Days 7 & Week 5 Days 1-2: Observability & Remaining (7 tickets)

**File**: [09-week4-observability-remaining.md](09-week4-observability-remaining.md)

| Ticket | Title | Time | Priority | Dependencies |
|--------|-------|------|----------|--------------|
| GOgent-087 | Benchmark Event Parsing | 1.5h | Medium | GOgent-002 |
| GOgent-088 | Timing Capture & Metrics | 1.5h | Medium | GOgent-087 |
| GOgent-089 | Benchmark JSONL Logging | 1h | Low | GOgent-088 |
| GOgent-090 | Build gogent-benchmark CLI | 1h | Medium | GOgent-089 |
| GOgent-091 | stop-gate.sh Investigation | 1.5h | Low | - |
| GOgent-092 | stop-gate Translation (if needed) | 2h | Low | GOgent-091 |
| GOgent-093 | stop-gate Integration Tests | 1.5h | Low | GOgent-092 |

**Total**: 10 hours

**Week 3-4 Total**: 57 hours

---

### Week 5: Testing & Deployment (15 tickets, ~38-42 hours)

⚠️ **REFACTORED: Originally weeks 6-7, now expanded to test/deploy ALL 7 hooks**

#### Week 5 Days 3-5: Integration & Regression Tests (13-14 tickets)

**File**: [10-week5-integration-tests.md](10-week5-integration-tests.md)

Original 7 tickets (GOgent-094-100) PLUS 6 new integration test tickets (GOgent-100b-100g) for new hooks.

**See file for complete ticket details.**

**Total**: ~22-24 hours

#### Week 5 Days 6-7: Deployment & Cutover (8 tickets, expanded scope)

**File**: [11-week5-deployment-cutover.md](11-week5-deployment-cutover.md)

Same 8 tickets (GOgent-101-055) but each expanded to handle ALL 7 hooks instead of just 3.

**See file for complete refactoring details.**

**Total**: ~16-18 hours

**Week 5 Total**: ~38-42 hours

---

## Total Time Estimate

| Phase | Tickets | Hours | Days (8hr) |
|-------|---------|-------|------------|
| Pre-Work | 1 | 2 | 0.25 |
| Week 1 | 25 | 35 | 4.4 |
| Week 2 | 15 | 22.5 | 2.8 |
| Week 3-4 | 45 | 68 | 8.5 |
| Week 5 | 21-22 | 38-42 | 4.8-5.3 |
| **Total** | **107-108** | **165.5-169.5** | **20.7-21.2** |

**Actual Duration**: ~5 weeks (25 work days) with buffer for testing, reviews, and integration.

**Note**: Weeks 3-4 add 38 new hook implementation tickets (GOgent-056 to 093). Week 5 expands original testing/deployment with 6-7 additional tickets for comprehensive coverage.

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
GOgent-094 (test harness)
  ↓
GOgent-095, 043, 044, 047 (integration/regression)
  ↓
GOgent-101, 049 (installation, parallel test)
  ↓
GOgent-103, 051 (cutover decision & execution)
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
