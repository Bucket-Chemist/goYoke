# Einstein GAP Document: GOgent-028 Integration Analysis

**Generated**: 2026-01-19
**Session ID**: current
**Escalated By**: User (direct /einstein invocation)
**Analysis Type**: Comprehensive Integration & Dependency Review

---

## Primary Question

Does the GOgent-028 series implementation properly integrate with:
1. Existing hook capture mechanisms (session-archive.sh)?
2. Current Go-native implementations (pkg/session/*)?
3. CLI entry points (cmd/gogent-validate)?
4. The broader system architecture?

What dependencies, gaps, or forward-planning artifacts need to be created to ensure the 028a-n ticket series supplements GOgent-028 correctly?

---

## Context: What Has Been Implemented

### Completed (GOgent-027 & GOgent-028)

**Package: `pkg/session/`**

1. **events.go** (GOgent-027):
   - `SessionEvent` struct for SessionEnd hook parsing
   - `SessionMetrics` struct for statistics
   - `ParseSessionEvent()` with timeout handling

2. **metrics.go** (GOgent-027):
   - `CollectSessionMetrics()` reads from temp files
   - Counts tool calls from `/tmp/claude-tool-counter-*`
   - Counts errors from `/tmp/claude-error-patterns.jsonl`
   - Counts violations from routing violations log

3. **handoff.go** (GOgent-028):
   - **JSONL format** handoff (NOT markdown!)
   - `Handoff` struct with schema versioning
   - `GenerateHandoff()` appends to `.claude/memory/handoffs.jsonl`
   - `LoadHandoff()` reads most recent handoff
   - `LoadAllHandoffs()` for history

4. **handoff_artifacts.go** (GOgent-028):
   - `LoadArtifacts()` aggregates sharp edges, violations, error patterns
   - `SharpEdge`, `RoutingViolation`, `ErrorPattern` structs
   - JSONL parsing for all artifact types

5. **handoff_markdown.go** (GOgent-028):
   - `RenderMarkdown()` converts JSONL handoff → human-readable .md
   - Sections: Metrics, Sharp Edges, Violations, Actions
   - Separate from core JSONL storage

**Command: `cmd/gogent-validate/`**

- `main.go`: PreToolUse hook for routing validation
- Validates `Task` tool calls against routing schema
- Logs violations to JSONL
- **Integrated**: Already called by validate-routing.sh hook

### Current Hook Architecture (Bash)

**`~/.claude/hooks/session-archive.sh`** (SessionEnd trigger):

```bash
# Reads SessionEnd event from STDIN (JSON)
# Counts metrics from temp files
# Generates MARKDOWN handoff to .claude/memory/last-handoff.md
# Archives pending-learnings.jsonl and violations JSONL
# Cleans up temp files
```

**Status**: This is the CURRENT production hook that Claude Code calls.

---

## Critical Discovery: Format Mismatch

### The Problem

**GOgent-028 Implementation** (handoff.go:106):
```go
// Append to JSONL file
f, err := os.OpenFile(cfg.HandoffPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
```
→ Writes to `.claude/memory/handoffs.jsonl` (JSONL format)

**Current Bash Hook** (session-archive.sh:40):
```bash
cat > "$HANDOFF_FILE" << HANDOFF
# Session Handoff - $timestamp
...
HANDOFF
```
→ Writes to `.claude/memory/last-handoff.md` (Markdown format, overwrites)

**Load-routing-context.sh** (SessionStart hook):
```bash
# Presumably reads last-handoff.md
```

### Impact

1. **Go implementation writes JSONL** (machine-readable, append-only)
2. **Bash hook writes Markdown** (human-readable, overwrite)
3. **Two different paths**: `handoffs.jsonl` vs `last-handoff.md`
4. **No bridge** between formats currently

### Resolution Strategy

The Go implementation is CORRECT:
- JSONL for machine processing
- Markdown rendering via `handoff_markdown.go:RenderMarkdown()`
- Append-only history in JSONL
- Human-readable summary generated on-demand

**Required**: Replace bash hook with Go CLI that:
1. Generates JSONL handoff
2. Renders markdown for human consumption
3. Maintains both formats

---

## Integration Wire-In Analysis

### 1. Hook Capture Integration ✅ ALIGNED

**Current Flow**:
```
Claude Code SessionEnd → session-archive.sh → bash metrics collection → markdown generation
```

**Target Flow (028a-n)**:
```
Claude Code SessionEnd → gogent-archive (Go CLI) → pkg/session.GenerateHandoff() → JSONL + rendered MD
```

**Wire-in Requirements**:
- ✅ `SessionEvent` struct matches hook JSON schema
- ✅ `ParseSessionEvent()` handles STDIN with timeout
- ✅ `GenerateHandoff()` accepts config + metrics
- ❌ **GAP**: No `cmd/gogent-archive/main.go` exists yet
- ❌ **GAP**: No hook configuration to call Go CLI instead of bash

**Ticket Needed**: GOgent-028a - Build gogent-archive CLI

### 2. Metrics Collection Integration ⚠️ PARTIAL

**What Works**:
- ✅ `CollectSessionMetrics()` reads temp files correctly
- ✅ Paths align with bash hook expectations
- ✅ JSONL parsing for artifacts works

**What's Missing**:
- ❌ Tool call counters: Bash hook uses `wc -l /tmp/claude-tool-counter-*`
  - Go: `countToolCalls()` globs and sums
  - **Risk**: Bash globbing vs Go filepath.Glob() behavior
- ❌ Error log: Bash hook counts lines
  - Go: `countLogLines()` with empty line filtering
  - **Risk**: Off-by-one if bash doesn't filter empty lines
- ❌ **GAP**: No test validates bash/Go metrics match

**Ticket Needed**: GOgent-028b - Validate Metrics Parity (Bash vs Go)

### 3. Artifact Loading Integration ✅ SOLID

**LoadArtifacts() reads**:
1. `.claude/memory/pending-learnings.jsonl` → `SharpEdge[]`
2. Config violation path → `RoutingViolation[]`
3. `/tmp/claude-error-patterns.jsonl` → `ErrorPattern[]`

**Bash Hook Archives**:
1. Moves `pending-learnings.jsonl` to `session-archive/learnings-{timestamp}.jsonl`
2. Moves violations to `session-archive/violations-{timestamp}.jsonl`
3. Deletes temp files

**Conflict**:
- ❌ If bash hook MOVES files, Go handoff generation fails (files gone)
- ❌ If Go handoff runs AFTER bash archival, artifacts are missing
- ❌ Timing dependency not documented

**Resolution**:
- Go CLI must handle archival AFTER reading artifacts
- Or: Read artifacts, generate handoff, THEN archive
- Bash hook pattern must be replicated

**Ticket Needed**: GOgent-028c - Implement Artifact Archival in Go

### 4. CLI Entry Point Integration ❌ MISSING

**Current**:
- `cmd/gogent-validate/` for PreToolUse validation (working)

**Needed**:
- `cmd/gogent-archive/` for SessionEnd handoff generation
- Must match bash script behavior:
  - Read STDIN JSON
  - Collect metrics
  - Generate JSONL handoff
  - Render markdown
  - Archive artifacts
  - Output confirmation JSON

**Wire-in**:
```go
// cmd/gogent-archive/main.go (DOESN'T EXIST YET)
func main() {
    // Parse SessionEvent from STDIN
    event, _ := session.ParseSessionEvent(os.Stdin, 5*time.Second)

    // Collect metrics
    metrics, _ := session.CollectSessionMetrics(event.SessionID)

    // Generate handoff (JSONL)
    cfg := session.DefaultHandoffConfig(projectDir)
    session.GenerateHandoff(cfg, metrics)

    // Render markdown for humans
    handoff, _ := session.LoadHandoff(cfg.HandoffPath)
    session.RenderMarkdown(handoff, ".claude/memory/last-handoff.md")

    // Archive artifacts
    session.ArchiveArtifacts(cfg, event.SessionID)

    // Output confirmation
    fmt.Println(`{"hookSpecificOutput": {...}}`)
}
```

**Ticket Needed**: GOgent-028a - Build gogent-archive CLI (CRITICAL)

### 5. Hook Configuration Integration ❌ NOT ADDRESSED

**Current Hook Definition** (presumably in Claude Code config):
```toml
[hooks.SessionEnd]
command = "~/.claude/hooks/session-archive.sh"
```

**Target**:
```toml
[hooks.SessionEnd]
command = "gogent-archive"
# OR
command = "/path/to/GOgent-Fortress/cmd/gogent-archive/gogent-archive"
```

**Deployment Questions**:
1. Where does gogent-archive binary live after build?
2. Is it installed globally or project-local?
3. How does Claude Code discover the binary?
4. Fallback if binary missing?

**Ticket Needed**:
- GOgent-028d - Hook Registration & Deployment
- GOgent-028e - Installation & PATH Configuration

### 6. Markdown Rendering Integration ✅ IMPLEMENTED

**pkg/session/handoff_markdown.go**:
- `RenderMarkdown()` exists
- Converts JSONL `Handoff` → markdown
- Sections: Metrics, Sharp Edges (formatted), Violations, Actions

**Use Case**:
```go
handoff := session.LoadHandoff(".claude/memory/handoffs.jsonl")
session.RenderMarkdown(handoff, ".claude/memory/last-handoff.md")
```

**Status**: Ready to use, just needs CLI integration

### 7. Context Loading Integration ⚠️ UNVERIFIED

**Load-routing-context.sh** (SessionStart hook):
- Presumably reads `.claude/memory/last-handoff.md`
- Injects context into Claude session

**Questions**:
1. Does it expect markdown format specifically?
2. Can it consume JSONL instead?
3. Does it parse specific sections?
4. Is it Claude-Code-internal or another bash script?

**Risk**: If load-routing-context.sh parses markdown sections by header regex, changing format breaks context loading.

**Investigation Needed**:
```bash
cat ~/.claude/hooks/load-routing-context.sh
```

**Ticket Needed**: GOgent-028f - Verify Context Loading Compatibility

---

## Dependency Map

### Forward Dependencies (What 028 needs)

1. **GOgent-027 ✅ COMPLETE**:
   - Session event parsing
   - Metrics collection
   - All implemented

2. **GOgent-026 (if exists)**:
   - Session structs definition
   - Check: Does 026 exist or was it folded into 027?

### Backward Dependencies (What needs 028)

1. **GOgent-029 (Format Pending Learnings)**:
   - Depends on: `SharpEdge` struct from handoff.go
   - Depends on: `LoadArtifacts()` for reading learnings
   - Status: Can proceed with current implementation

2. **GOgent-030 (Format Violations)**:
   - Depends on: `RoutingViolation` struct
   - Depends on: `LoadArtifacts()`
   - Status: Can proceed

3. **Load-Routing-Context Hook**:
   - Depends on: Stable markdown format in `last-handoff.md`
   - Risk: Breaking change if markdown schema changes
   - Mitigation: 028f verification ticket

### Parallel Dependencies (Related systems)

1. **Routing Validation (gogent-validate)**:
   - Writes violations → `.claude/memory/routing-violations.jsonl`
   - Read by: `LoadArtifacts()` → handoff generation
   - Status: ✅ Wire-in works

2. **Sharp Edge Detection Hook (future)**:
   - Will write → `.claude/memory/pending-learnings.jsonl`
   - Read by: `LoadArtifacts()`
   - Status: ⚠️ Schema alignment required (028g ticket)

3. **Ticket System (.ticket-current)**:
   - Read by: `getActiveTicket()` in handoff.go:248
   - Wire-in: ✅ Works if file exists
   - Graceful degradation: Returns "" if missing

---

## Event Handler Integration Analysis

### Current Event Flow

```
┌─────────────────────────────────────────────────────────────┐
│                     Claude Code Session                     │
└────────────┬────────────────────────────────┬───────────────┘
             │                                │
    ┌────────▼────────┐              ┌───────▼────────┐
    │ PreToolUse      │              │ SessionEnd     │
    │ (Task validation)│              │ (Archive)      │
    └────────┬────────┘              └───────┬────────┘
             │                                │
    ┌────────▼─────────┐             ┌───────▼────────┐
    │ validate-routing.sh│            │session-archive.sh│
    └────────┬─────────┘             └───────┬────────┘
             │                                │
    ┌────────▼─────────┐             ┌───────▼────────┐
    │gogent-validate   │             │ BASH metrics   │
    │   (Go CLI)       │             │   collection   │
    └────────┬─────────┘             └───────┬────────┘
             │                                │
    ┌────────▼─────────┐             ┌───────▼────────┐
    │ ValidationResult │             │last-handoff.md │
    │   (JSON output)  │             │   (Markdown)   │
    └──────────────────┘             └────────────────┘
```

### Target Event Flow (028a-n)

```
┌─────────────────────────────────────────────────────────────┐
│                     Claude Code Session                     │
└────────────┬────────────────────────────────┬───────────────┘
             │                                │
    ┌────────▼────────┐              ┌───────▼────────┐
    │ PreToolUse      │              │ SessionEnd     │
    │ (Task validation)│              │ (Archive)      │
    └────────┬────────┘              └───────┬────────┘
             │                                │
    ┌────────▼─────────┐             ┌───────▼────────┐
    │ validate-routing.sh│            │ gogent-archive │
    │  (thin wrapper)  │             │   (Go CLI)     │ ← NEW
    └────────┬─────────┘             └───────┬────────┘
             │                                │
    ┌────────▼─────────┐             ┌───────▼────────────────┐
    │gogent-validate   │             │ pkg/session/           │
    │   (Go CLI)       │             │ - CollectSessionMetrics│
    └────────┬─────────┘             │ - GenerateHandoff      │
             │                        │ - LoadArtifacts        │
    ┌────────▼─────────┐             │ - RenderMarkdown       │
    │ ValidationResult │             │ - ArchiveArtifacts     │
    │   (JSON output)  │             └───────┬────────────────┘
    └──────────────────┘                     │
                                     ┌───────▼────────────────┐
                                     │ handoffs.jsonl (JSONL) │
                                     │ last-handoff.md (MD)   │
                                     └────────────────────────┘
```

### Integration Points

1. **Hook Invocation** (Claude Code → Go CLI):
   - ✅ JSON on STDIN (same pattern as gogent-validate)
   - ✅ JSON confirmation on STDOUT
   - ❌ Hook registration not documented

2. **Metrics Collection** (Go CLI → pkg/session):
   - ✅ `CollectSessionMetrics()` API exists
   - ⚠️ Parity with bash implementation unverified

3. **Handoff Generation** (pkg/session → filesystem):
   - ✅ JSONL append to `handoffs.jsonl`
   - ✅ Markdown render to `last-handoff.md`
   - ❌ Archival of artifacts not implemented

4. **Context Loading** (SessionStart → ???):
   - ⚠️ Unknown: Does load-routing-context.sh still work?
   - ⚠️ Unknown: Does it need JSONL or markdown?

---

## Output Format Integration

### Current Outputs (Bash Hook)

1. **last-handoff.md** (markdown, overwritten):
   ```markdown
   # Session Handoff - {timestamp}
   ## Session Metrics
   - Tool Calls: ~42
   - Errors: 5
   ...
   ## Pending Learnings
   - **file.go**: error_type (3 failures)
   ...
   ```

2. **Session Archives**:
   - `session-archive/session-{timestamp}.jsonl` (transcript)
   - `session-archive/learnings-{timestamp}.jsonl` (moved)
   - `session-archive/violations-{timestamp}.jsonl` (moved)

### New Outputs (Go Implementation)

1. **handoffs.jsonl** (JSONL, append-only):
   ```json
   {"schema_version":"1.0","timestamp":1234567890,"session_id":"abc","context":{...},"artifacts":{...},"actions":[...]}
   ```

2. **last-handoff.md** (rendered markdown):
   - Same format as bash version
   - Generated from JSONL via `RenderMarkdown()`

3. **Session Archives** (to be implemented):
   - Same paths as bash
   - Move logic in Go

### Compatibility

| Consumer | Expects | Current Output | Status |
|----------|---------|----------------|--------|
| Human readers | Markdown | ✅ last-handoff.md | OK |
| load-routing-context.sh | Markdown? | ✅ last-handoff.md | Needs verification |
| Machine processing | JSONL | ✅ handoffs.jsonl | NEW capability |
| Historical analysis | Archives | ❌ Not implemented | 028c ticket |

---

## Gaps Identified

### Critical Gaps (Block 028 completion)

1. **GAP-028-1: Missing gogent-archive CLI**
   - **Impact**: Can't replace bash hook without CLI entry point
   - **Blocker**: Yes - bash hook remains in production
   - **Ticket**: GOgent-028a

2. **GAP-028-2: Artifact Archival Not Implemented**
   - **Impact**: Artifacts not moved to session-archive/
   - **Blocker**: Yes - breaks multi-session history
   - **Ticket**: GOgent-028c

3. **GAP-028-3: Hook Registration Undefined**
   - **Impact**: Don't know how to deploy Go CLI as hook
   - **Blocker**: Yes - can't cut over to Go
   - **Ticket**: GOgent-028d

### Major Gaps (Degrade functionality)

4. **GAP-028-4: Metrics Parity Unverified**
   - **Impact**: Go metrics might differ from bash
   - **Risk**: Off-by-one errors, missing data
   - **Ticket**: GOgent-028b

5. **GAP-028-5: Context Loading Compatibility Unknown**
   - **Impact**: SessionStart hook might break
   - **Risk**: Context not injected properly
   - **Ticket**: GOgent-028f

6. **GAP-028-6: Sharp Edge Schema Alignment**
   - **Impact**: Future sharp-edge hook might write incompatible JSONL
   - **Risk**: Handoff generation fails on parse
   - **Ticket**: GOgent-028g

### Minor Gaps (Edge cases)

7. **GAP-028-7: Git Info Collection Stubbed**
   - **Impact**: `collectGitInfo()` returns empty struct (handoff.go:258)
   - **Risk**: Missing context in handoff
   - **Ticket**: GOgent-028h (low priority)

8. **GAP-028-8: No Error Recovery**
   - **Impact**: CLI crashes don't fall back to bash
   - **Risk**: Session end fails silently
   - **Ticket**: GOgent-028i (defensive)

---

## Ticket Series Design (028a-n)

### Tier 1: Critical Path (Blocking)

**GOgent-028a: Build gogent-archive CLI**
- **Priority**: CRITICAL
- **Time**: 2h
- **Dependencies**: GOgent-027, GOgent-028
- **Deliverables**:
  - `cmd/gogent-archive/main.go`
  - Parses SessionEnd JSON from STDIN
  - Calls `CollectSessionMetrics()` + `GenerateHandoff()`
  - Renders markdown via `RenderMarkdown()`
  - Outputs JSON confirmation
  - Tests: Mock STDIN, verify JSONL + MD outputs

**GOgent-028b: Validate Metrics Parity**
- **Priority**: CRITICAL
- **Time**: 1h
- **Dependencies**: GOgent-028a
- **Deliverables**:
  - Test: Run bash hook + Go CLI on same session
  - Compare metrics output
  - Fix any discrepancies (empty line handling, globbing)
  - Document differences (if unavoidable)

**GOgent-028c: Implement Artifact Archival**
- **Priority**: CRITICAL
- **Time**: 1.5h
- **Dependencies**: GOgent-028a
- **Deliverables**:
  - `session.ArchiveArtifacts(cfg, sessionID)` function
  - Moves pending-learnings.jsonl → `session-archive/learnings-{timestamp}.jsonl`
  - Moves violations → `session-archive/violations-{timestamp}.jsonl`
  - Copies transcript if available
  - Integration into gogent-archive CLI
  - Tests: Verify moves, graceful handling if missing

**GOgent-028d: Hook Registration & Deployment**
- **Priority**: CRITICAL
- **Time**: 1.5h
- **Dependencies**: GOgent-028a, 028b, 028c
- **Deliverables**:
  - Build script for gogent-archive
  - Installation instructions (Makefile target?)
  - Hook configuration example
  - Test: Register Go hook, trigger SessionEnd, verify handoff
  - Document rollback to bash hook

### Tier 2: Compatibility (Non-blocking but important)

**GOgent-028e: Installation & PATH Configuration**
- **Priority**: HIGH
- **Time**: 1h
- **Dependencies**: GOgent-028d
- **Deliverables**:
  - Makefile `install` target
  - Installs gogent-archive to `~/.local/bin/` or `/usr/local/bin/`
  - Updates project CLAUDE.md with installation steps
  - Uninstall target

**GOgent-028f: Verify Context Loading Compatibility**
- **Priority**: HIGH
- **Time**: 1h
- **Dependencies**: GOgent-028a
- **Deliverables**:
  - Read `~/.claude/hooks/load-routing-context.sh`
  - Test: Does it parse new markdown format?
  - Test: Does it work with JSONL if passed?
  - Document any required changes
  - If broken: Fix or propose 028f-1 ticket

**GOgent-028g: Sharp Edge Schema Alignment**
- **Priority**: MAJOR
- **Time**: 0.5h
- **Dependencies**: GOgent-028
- **Deliverables**:
  - Document `SharpEdge` JSON schema
  - Create `docs/schemas/sharp-edge.json` (JSON Schema)
  - Validation function: `ValidateSharpEdge([]byte) error`
  - Tests: Valid and invalid JSONL parsing

### Tier 3: Enhancements (Nice-to-have)

**GOgent-028h: Implement Git Info Collection**
- **Priority**: MINOR
- **Time**: 1h
- **Dependencies**: GOgent-028a
- **Deliverables**:
  - Replace stub in `collectGitInfo()`
  - Exec `git branch`, `git status --porcelain`
  - Parse output into `GitInfo` struct
  - Handle non-git directories gracefully
  - Tests: Mock git output, verify parsing

**GOgent-028i: Error Recovery & Fallback**
- **Priority**: MINOR
- **Time**: 1h
- **Dependencies**: GOgent-028d
- **Deliverables**:
  - Wrapper script: Try Go CLI, fall back to bash on failure
  - Logging: Why did Go CLI fail?
  - Graceful degradation: Partial handoff if artifacts missing
  - Tests: Simulate failures, verify fallback

**GOgent-028j: JSONL History Querying**
- **Priority**: MINOR
- **Time**: 1.5h
- **Dependencies**: GOgent-028
- **Deliverables**:
  - `gogent-archive list` - show all sessions
  - `gogent-archive show <session-id>` - render specific handoff
  - `gogent-archive stats` - aggregate metrics across sessions
  - Useful for retrospectives

### Tier 4: Observability (Future-proofing)

**GOgent-028k: Handoff Generation Metrics**
- **Priority**: MINOR
- **Time**: 0.5h
- **Dependencies**: GOgent-028a
- **Deliverables**:
  - Log handoff generation time
  - Log artifact counts (edges, violations, patterns)
  - Emit JSON to stderr for observability hooks
  - Helps diagnose performance issues

**GOgent-028l: Handoff Schema Versioning**
- **Priority**: MINOR
- **Time**: 1h
- **Dependencies**: GOgent-028
- **Deliverables**:
  - `HandoffSchemaVersion` already exists (handoff.go:15)
  - Migration logic: Read v1.0, convert if older
  - Backward compatibility tests
  - Future-proofs against schema changes

---

## Forward Planning Artifacts

### 1. Integration Test Plan

**File**: `test/integration/session-handoff-integration_test.go`

**Scope**:
- End-to-end: SessionEnd JSON → gogent-archive → JSONL + MD outputs
- Artifact loading: Pre-populate learnings/violations, verify in handoff
- Archival: Verify files moved to session-archive/
- Context loading: Simulate SessionStart, verify markdown readable

**Ticket**: GOgent-028m - Integration Test Suite (1.5h)

### 2. Deployment Runbook

**File**: `docs/deployment/gogent-archive-cutover.md`

**Contents**:
- Pre-cutover checklist
- Build and install instructions
- Hook configuration changes
- Validation steps (trigger SessionEnd, check outputs)
- Rollback procedure (restore bash hook)
- Troubleshooting guide

**Ticket**: GOgent-028n - Deployment Runbook (1h)

### 3. Architecture Decision Record

**File**: `docs/decisions/ADR-028-jsonl-handoff-format.md`

**Rationale**:
- Why JSONL over markdown-only?
- Benefits: Machine-readable, append-only history, queryable
- Tradeoffs: Dual-format complexity (JSONL + rendered MD)
- Migration strategy from bash markdown

**Ticket**: GOgent-028o - ADR Documentation (0.5h)

### 4. Schema Documentation

**File**: `docs/schemas/handoff-v1.0.json`

**Contents**:
- JSON Schema for `Handoff` struct
- Example valid JSONL
- Field descriptions
- Validation rules

**Ticket**: GOgent-028g (already planned)

### 5. Metrics Dashboard Spec

**File**: `docs/observability/session-metrics-dashboard.md`

**Vision**:
- Tool call trends over time
- Error rate analysis
- Routing violation patterns
- Sharp edge frequency

**Data Source**: `handoffs.jsonl` (queryable history)

**Ticket**: GOgent-040+ (post-028 series)

---

## Dependency Graph (Visual)

```
         ┌──────────────┐
         │  GOgent-027  │
         │  (Metrics)   │
         └──────┬───────┘
                │
         ┌──────▼───────┐
         │  GOgent-028  │
         │  (Handoff)   │
         └──────┬───────┘
                │
    ┌───────────┼───────────┐
    │           │           │
┌───▼────┐  ┌──▼─────┐  ┌──▼─────┐
│ 028a   │  │ 028b   │  │ 028c   │
│ CLI    │  │ Metrics│  │Archive │
└───┬────┘  └───┬────┘  └───┬────┘
    │           │           │
    └───────────┼───────────┘
                │
         ┌──────▼───────┐
         │   028d       │
         │   Hook Reg   │
         └──────┬───────┘
                │
    ┌───────────┼────────────┬──────────┐
    │           │            │          │
┌───▼────┐  ┌──▼─────┐  ┌───▼────┐ ┌──▼─────┐
│ 028e   │  │ 028f   │  │ 028g   │ │ 028h   │
│Install │  │Context │  │Schema  │ │Git Info│
└────────┘  └────────┘  └────────┘ └────────┘
                │
         ┌──────▼───────┐
         │   028m       │
         │  Integration │
         │     Test     │
         └──────────────┘
```

---

## Recommendation: Ticket Priorities

### Week 1 (Critical Path - 6.5h)
1. **GOgent-028a** - Build CLI (2h) - **START HERE**
2. **GOgent-028b** - Metrics Parity (1h)
3. **GOgent-028c** - Archival (1.5h)
4. **GOgent-028d** - Hook Registration (1.5h)
5. **GOgent-028e** - Installation (0.5h - quick win)

**Milestone**: Go-native session archival deployed, bash hook retired

### Week 2 (Compatibility - 2.5h)
6. **GOgent-028f** - Context Loading (1h)
7. **GOgent-028g** - Schema Alignment (0.5h)
8. **GOgent-028m** - Integration Tests (1h - reduced scope)

**Milestone**: Verified compatibility with SessionStart hook

### Week 3 (Enhancements - Optional)
9. **GOgent-028h** - Git Info (1h)
10. **GOgent-028i** - Error Recovery (1h)
11. **GOgent-028j** - JSONL Querying (1.5h)

**Milestone**: Production-hardened, queryable session history

### Documentation (Parallel track - 2h)
- **GOgent-028n** - Deployment Runbook (1h)
- **GOgent-028o** - ADR (0.5h)
- Update project CLAUDE.md (0.5h)

---

## Critical Insights

### 1. The Bash Hook is a Blueprint

The existing `session-archive.sh` is EXCELLENT documentation:
- It shows exactly what metrics to collect
- It shows the markdown format expected
- It shows archival behavior
- **Use it as acceptance criteria for Go CLI**

### 2. Dual Format is Intentional

JSONL + Markdown is correct:
- JSONL: Machine processing, history, queryable
- Markdown: Human consumption, hook compatibility
- Both generated from same source (Handoff struct)
- No data loss, maximum flexibility

### 3. The Hook Interface is Stable

SessionEnd hook interface:
```
Input: JSON on STDIN
Output: JSON confirmation on STDOUT
Side effects: Write files, archive artifacts
```

This is the contract. Go CLI must honor it exactly.

### 4. Metrics Parity is CRITICAL

If Go counts 42 tool calls but bash counted 45:
- Which is correct?
- How do we know?
- What breaks?

**Must have**: Test that compares bash vs Go on same session.

### 5. Context Loading is the Unknown

We haven't read `load-routing-context.sh` yet. Could be:
- Simple `cat last-handoff.md` (safe)
- Regex parsing of markdown sections (brittle)
- jq processing of JSONL (requires format change)

**Must investigate** before claiming compatibility.

---

## Answers to Original Questions

### Q1: Does this plan wire in with existing implementation?

**Answer**: PARTIALLY

- ✅ **Metrics collection**: Yes, `CollectSessionMetrics()` aligns
- ✅ **Artifact loading**: Yes, JSONL parsing works
- ✅ **Handoff generation**: Yes, core logic is sound
- ❌ **Hook capture**: No CLI to call yet (028a GAP)
- ❌ **Archival**: Not implemented (028c GAP)
- ⚠️ **Context loading**: Unknown compatibility (028f GAP)

### Q2: Does anything require revisiting?

**Answer**: YES - Minor adjustments needed

1. **Format Clarity**: Document that JSONL is primary, markdown is rendered
2. **Metrics Parity**: Add test to verify bash/Go equivalence (028b)
3. **Archival Timing**: Implement move logic AFTER handoff generation (028c)
4. **Hook Registration**: Define deployment process (028d)

**Nothing architecturally broken**, just missing glue code.

### Q3: How does this glue the system together?

**Answer**: Via CLI Entry Points

```
System Integration Points:
┌─────────────────────────────────────────────────┐
│            Claude Code Hook System              │
└────────┬──────────────────────────┬─────────────┘
         │                          │
    ┌────▼────────┐          ┌──────▼────────┐
    │PreToolUse   │          │SessionEnd     │
    │validate-    │          │gogent-        │
    │routing.sh   │          │archive        │ ← NEW
    └────┬────────┘          └──────┬────────┘
         │                          │
    ┌────▼────────┐          ┌──────▼────────────────┐
    │gogent-      │          │pkg/session/           │
    │validate     │          │  - Events             │
    └─────────────┘          │  - Metrics            │
                             │  - Handoff            │
                             │  - Artifacts          │
                             │  - Markdown           │
                             └──────┬────────────────┘
                                    │
                             ┌──────▼────────────────┐
                             │ Filesystem Outputs    │
                             │ - handoffs.jsonl      │
                             │ - last-handoff.md     │
                             │ - session-archive/    │
                             └───────────────────────┘
```

**The Glue**:
1. CLI binaries (gogent-validate, gogent-archive)
2. Shared pkg/session library
3. Hook configuration (TOML or bash wrappers)
4. File-based contracts (JSONL schemas)

### Q4: What tickets supplement GOgent-028?

**Answer**: 15 tickets in 4 tiers

**Tier 1 (Critical - 6.5h)**:
- 028a: CLI (2h)
- 028b: Metrics Parity (1h)
- 028c: Archival (1.5h)
- 028d: Hook Registration (1.5h)
- 028e: Installation (0.5h)

**Tier 2 (Compatibility - 2.5h)**:
- 028f: Context Loading (1h)
- 028g: Schema Alignment (0.5h)
- 028m: Integration Tests (1h)

**Tier 3 (Enhancements - 3.5h)**:
- 028h: Git Info (1h)
- 028i: Error Recovery (1h)
- 028j: JSONL Query (1.5h)

**Tier 4 (Docs - 2h)**:
- 028n: Runbook (1h)
- 028o: ADR (0.5h)
- CLAUDE.md update (0.5h)

**Total**: ~14.5h of work

### Q5: Are there other dependencies?

**Answer**: YES - External Systems

**Upstream Dependencies**:
1. **Claude Code Hook Interface**:
   - Must accept Go binary as hook command
   - Must pass JSON on STDIN correctly
   - Must handle JSON output
   - **Risk**: If hook interface changes, all breaks

2. **Filesystem Paths**:
   - `.claude/memory/` writable
   - `/tmp/claude-*` readable
   - **Risk**: Permissions, disk space

3. **Load-Routing-Context Hook**:
   - Must parse new markdown format
   - **Risk**: Unknown compatibility (needs 028f)

**Downstream Dependencies**:
1. **Future Sharp Edge Hook** (GOgent-034-040):
   - Will write pending-learnings.jsonl
   - Must match `SharpEdge` schema
   - Validated by: 028g

2. **Ticket System**:
   - Writes `.ticket-current`
   - Read by: `getActiveTicket()`
   - **Status**: Already integrated

3. **Routing Validation**:
   - Writes routing-violations.jsonl
   - Read by: `LoadArtifacts()`
   - **Status**: Already integrated

### Q6: Any GAPs in the system?

**Answer**: YES - 8 GAPs identified (see Gaps Identified section)

**Critical (blocking)**: 028-1, 028-2, 028-3
**Major (degrading)**: 028-4, 028-5, 028-6
**Minor (edge cases)**: 028-7, 028-8

All addressable via 028a-o ticket series.

### Q7: Forward planning documents needed?

**Answer**: 5 artifacts recommended

1. **Integration Test Plan** (028m)
2. **Deployment Runbook** (028n)
3. **Architecture Decision Record** (028o)
4. **Schema Documentation** (028g)
5. **Metrics Dashboard Spec** (future)

These ensure:
- Smooth deployment
- Future maintainability
- Schema stability
- Observability foundation

---

## Conclusion

### Summary

The GOgent-028 implementation is **architecturally sound** but **operationally incomplete**:

✅ **Strengths**:
- Core logic is correct (JSONL handoff, artifact loading)
- Dual-format approach (JSONL + Markdown) is elegant
- Integration points are designed (just not implemented)
- Schema versioning is forward-thinking

❌ **Gaps**:
- No CLI entry point (can't call from hook)
- No artifact archival (files don't move)
- No hook registration process (can't deploy)
- Unknown context loading compatibility

⚠️ **Risks**:
- Metrics parity unverified (might count differently)
- Context loading might break (haven't tested)
- Sharp edge schema might diverge (need validation)

### Recommended Next Steps

1. **Immediate** (Week 1): Implement 028a-e (critical path)
   - This gets Go CLI working end-to-end
   - Allows bash hook retirement

2. **Short-term** (Week 2): Implement 028f-g, 028m (compatibility)
   - Verifies SessionStart hook still works
   - Validates schema alignment for future work

3. **Long-term** (Week 3+): Implement 028h-j (enhancements)
   - Git info, error recovery, JSONL querying
   - Production hardening

4. **Documentation** (Parallel): 028n-o + CLAUDE.md update
   - Runbook for deployment
   - ADR for future reference

### Success Criteria

**Definition of Done** for 028 series:

- [ ] `gogent-archive` CLI built and installed
- [ ] SessionEnd hook calls Go CLI (not bash)
- [ ] JSONL handoff generated to `.claude/memory/handoffs.jsonl`
- [ ] Markdown rendered to `.claude/memory/last-handoff.md`
- [ ] Artifacts archived to `session-archive/`
- [ ] Metrics match bash hook output (±1% tolerance)
- [ ] SessionStart hook (load-routing-context.sh) still works
- [ ] Integration tests pass
- [ ] Deployment runbook exists
- [ ] Bash hook retired (archived for reference)

**Timeline**: 2-3 weeks for complete series (14.5h effort)

---

## Appendix: Existing vs Target Comparison

### Bash Hook (Current)

**Pros**:
- Works today
- Simple to understand
- Easy to debug (just cat the script)

**Cons**:
- Markdown-only (not machine-readable)
- Overwrites history (only "last" handoff)
- Bash globbing/counting fragile
- Can't query across sessions

### Go Implementation (Target)

**Pros**:
- JSONL history (queryable, append-only)
- Markdown generated (dual format)
- Type-safe metrics collection
- Testable (unit + integration)
- Reusable library (pkg/session)

**Cons**:
- More complex (CLI + lib + tests)
- Deployment overhead (build, install)
- Requires Go toolchain
- Must maintain compatibility

### Why Migrate?

1. **Queryability**: Answer "How many tool calls last 10 sessions?"
2. **Type Safety**: Catch schema errors at compile time
3. **Testability**: Unit tests > bash integration tests
4. **Reusability**: Other tools can import pkg/session
5. **Evolution**: JSON Schema versioning for future changes

**Bottom line**: Bash was great for prototyping. Go is required for production scale.

---

**Generated**: 2026-01-19
**Analysis Type**: Comprehensive Integration & Dependency Review
**Estimated Reading Time**: 25 minutes
**Recommended Action**: Implement 028a-e immediately, then 028f-g for compatibility verification
