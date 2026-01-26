# TUI Tickets JSON Construction Summary

**Generated:** 2026-01-26
**Task:** Construct complete JSON ticket entries for all 13 TUI tickets
**Output File:** `/home/doktersmol/Documents/GOgent-Fortress/dev/will/migration_plan/tickets/TUI/tui-tickets-json-entries.json`

---

## Overview

Successfully constructed 13 complete ticket JSON entries ready for insertion into `tickets-index.json`. All tickets follow the established schema and maintain proper dependency/blocking relationships.

---

## Ticket ID Mapping

| TUI ID | GOgent ID | Title | Week | Priority |
|--------|-----------|-------|------|----------|
| TUI-INFRA-01 | GOgent-109 | Agent Lifecycle Telemetry | 6 | critical |
| TUI-CLI-01 | GOgent-110 | CLI Subprocess Management | 6 | critical |
| TUI-PERF-01 | GOgent-111 | Performance Dashboard Shell | 6 | high |
| TUI-CLI-01a | GOgent-112 | Auto-Restart on Panic | 6 | high |
| TUI-TELEM-01 | GOgent-113 | File Watchers for Telemetry | 6 | high |
| TUI-CLI-02 | GOgent-114 | Event System Integration | 7 | critical |
| TUI-AGENT-01 | GOgent-115 | Agent Tree Model | 7 | high |
| TUI-AGENT-02 | GOgent-116 | Tree View Component | 7 | high |
| TUI-AGENT-03 | GOgent-117 | Agent Detail Sidebar | 7 | medium |
| TUI-CLI-03 | GOgent-118 | Claude Conversation Panel | 8 | high |
| TUI-CLI-04 | GOgent-119 | 70/30 Layout Integration | 8 | high |
| TUI-MAIN-01 | GOgent-120 | Persistent Banner | 8 | high |
| TUI-CLI-05 | GOgent-121 | Session Management | 8 | medium |

---

## Week Assignments

**Rationale:** TUI README specifies 3 weeks (Weeks 1-3). Mapped to weeks 6-8 because tickets-index.json currently goes through week 5.

| Week | Phase | Tickets | Total Hours |
|------|-------|---------|-------------|
| **6** | Infrastructure + Foundation | GOgent-109, 110, 111, 112, 113 | 10.0h |
| **7** | Event System + Agent Tree | GOgent-114, 115, 116, 117 | 8.0h |
| **8** | Integration + Session | GOgent-118, 119, 120, 121 | 9.0h |
| **Total** | | **13 tickets** | **27.0h** |

---

## Priority Distribution

| Priority | Count | Tickets |
|----------|-------|---------|
| **critical** | 3 | GOgent-109, 110, 114 |
| **high** | 8 | GOgent-111, 112, 113, 115, 116, 118, 119, 120 |
| **medium** | 2 | GOgent-117, 121 |

---

## Dependency Validation

### Starting Points (No Dependencies)
- GOgent-109 (TUI-INFRA-01)
- GOgent-110 (TUI-CLI-01)
- GOgent-111 (TUI-PERF-01)

### Critical Path
Longest dependency chain (6 tickets):
```
GOgent-110 → GOgent-114 → GOgent-118 → GOgent-119 → GOgent-120 → GOgent-121
(CLI-01)   (CLI-02)      (CLI-03)      (CLI-04)      (MAIN-01)    (CLI-05)
```

### Parallel Execution Opportunities

**Week 6 (3 parallel streams):**
- Stream 1: GOgent-109 → GOgent-113
- Stream 2: GOgent-110 → GOgent-112
- Stream 3: GOgent-111 (standalone)

**Week 7 (2 parallel streams):**
- Stream 1: GOgent-114 (after GOgent-110)
- Stream 2: GOgent-115 → GOgent-116 → GOgent-117 (after GOgent-109, GOgent-113)

**Week 8 (sequential):**
- GOgent-118 → GOgent-119 → GOgent-120
- GOgent-121 (after GOgent-112, can run in parallel with GOgent-120)

---

## Field Completeness

All 13 tickets include:

✅ **Required Fields:**
- `id`, `title`, `description`, `status`, `priority`, `time_estimate`
- `week`, `dependencies`, `tags`, `tests_required`

✅ **Optional Fields (All Populated):**
- `file` - Path to TUI/*.md source
- `blocks` - Calculated from dependency reciprocity
- `git_branch` - Generated from ID and title slug
- `pr_labels` - Phase-appropriate labels
- `files_to_create` - Extracted from ticket specifications
- `acceptance_criteria_count` - Counted from AC sections

✅ **Consistency:**
- `day` - All set to `null` (not yet scheduled)
- `status` - All set to `"pending"`

---

## Tags Strategy

Each ticket has 5-6 tags covering:

1. **Domain:** `tui` (all tickets)
2. **Category:** `infrastructure`, `foundation`, `agent-tree`, `integration`, etc.
3. **Technology:** `bubbletea`, `telemetry`, `cli`, `subprocess`, etc.
4. **Week:** `week-6`, `week-7`, `week-8`
5. **Phase:** `phase-0-infrastructure`, `phase-1-foundation`, etc.

---

## Files to Create Summary

**Total:** 42 files across 13 tickets

| Category | Count | Examples |
|----------|-------|----------|
| **Core Implementation** | 20 | `pkg/telemetry/agent_lifecycle.go`, `internal/cli/subprocess.go` |
| **Tests** | 13 | `*_test.go` files |
| **UI Components** | 9 | `internal/tui/agents/view.go`, `internal/tui/claude/panel.go` |

**Breakdown by Package:**
- `pkg/telemetry/` - 2 files
- `pkg/config/` - modifications (not new files)
- `internal/cli/` - 10 files
- `internal/tui/telemetry/` - 4 files
- `internal/tui/agents/` - 6 files
- `internal/tui/claude/` - 4 files
- `internal/tui/performance/` - 3 files
- `internal/tui/main/` - 4 files
- `internal/tui/session/` - 2 files

---

## PR Labels

Consistent labeling strategy across all tickets:

**Common Labels:**
- `tui` - All tickets
- Phase labels: `phase-0`, `phase-1`, `phase-2`, `phase-3`, `phase-4`, `phase-5`

**Category Labels:**
- `infrastructure`, `foundation`, `event-system`, `agent-tree`, `integration`, `session-management`
- `telemetry`, `cli`, `bubbletea`, `layout`, `navigation`, `enhancement`, `polish`

---

## Acceptance Criteria Summary

| Ticket | AC Count | Notes |
|--------|----------|-------|
| GOgent-109 | 8 | Infrastructure setup |
| GOgent-110 | 10 | Most complex (subprocess management) |
| GOgent-111 | 9 | Dashboard shell |
| GOgent-112 | 9 | Error handling focus |
| GOgent-113 | 8 | File watching |
| GOgent-114 | 6 | Event parsing |
| GOgent-115 | 7 | Data model |
| GOgent-116 | 8 | Tree view component |
| GOgent-117 | 6 | Detail sidebar |
| GOgent-118 | 8 | Claude panel |
| GOgent-119 | 6 | Layout integration |
| GOgent-120 | 6 | Banner component |
| GOgent-121 | 7 | Session management |
| **Average** | **7.5** | |

---

## Validation Checklist

✅ All 13 tickets constructed
✅ IDs sequential (GOgent-109 through GOgent-121)
✅ Dependencies translated to GOgent IDs
✅ Blocks arrays reciprocal to dependencies
✅ Week assignments logical (6-8)
✅ Priority mappings preserved from source
✅ Time estimates extracted from headers
✅ File paths point to TUI/*.md
✅ Git branches follow naming convention
✅ PR labels phase-appropriate
✅ Tags comprehensive
✅ No circular dependencies
✅ All required fields present
✅ JSON valid (ready for insertion)

---

## Next Steps

1. **Insert into tickets-index.json**
   - Append to `tickets` array
   - Update metadata (total_tickets, total_weeks)
   - Validate JSON schema

2. **Update cross-references**
   - Verify no existing tickets reference TUI-* IDs
   - Update any GOgent-109+ forward references

3. **Validate dependencies**
   - Run dependency graph validator
   - Check for circular dependencies
   - Verify all referenced IDs exist

4. **Update week metadata**
   - Extend `total_weeks` to 8
   - Document TUI implementation phase

---

## Construction Methodology

1. **Source Data Collection:**
   - Read all 13 TUI/*.md ticket files
   - Extracted metadata from frontmatter headers
   - Counted acceptance criteria sections

2. **ID Assignment:**
   - Used mapping from tui-gogent-mapping.json
   - Sequential GOgent-109 through GOgent-121

3. **Week Calculation:**
   - TUI README specifies Weeks 1-3
   - Mapped to weeks 6-8 (after existing week 5 tickets)

4. **Field Generation:**
   - `git_branch`: Slugified `id-title`
   - `pr_labels`: Phase + category tags
   - `tags`: Domain + category + week + phase
   - `files_to_create`: Extracted from "Files to Create" sections

5. **Validation:**
   - Dependency reciprocity check
   - JSON schema compliance
   - Field completeness verification

---

**Status:** ✅ COMPLETE - Ready for insertion into tickets-index.json
