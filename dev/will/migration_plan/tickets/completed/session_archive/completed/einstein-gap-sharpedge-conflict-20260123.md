# GAP Analysis: SharpEdge Type Conflict Resolution

**Escalated:** 2026-01-23
**Escalated By:** User (via /einstein quick mode)
**Urgency:** IMMEDIATE - Blocking GOgent-037b completion
**Scope:** pkg/memory vs pkg/session SharpEdge types

---

## 1. Problem Statement

Two `SharpEdge` type definitions exist:

| Location | Package | Status | Schema |
|----------|---------|--------|--------|
| `pkg/session/handoff.go:54` | session | **COMMITTED** (GOgent-029 series) | v1.0+ versioned |
| `pkg/memory/sharp_edge.go:11` | memory | **UNTRACKED** (GOgent-037b WIP) | ad-hoc |

**Critical Issue:** These types have **incompatible JSON serialization**:
- `session.SharpEdge`: `json:"timestamp"`
- `memory.SharpEdge`: `json:"ts"`

This will cause silent data loss when serializing/deserializing across the system.

---

## 2. Primary Question

**Can we consolidate to a single SharpEdge type NOW without breaking tickets 037b-041c?**

Sub-questions:
1. Which type should be canonical?
2. What fields need to be added to support the ticket series?
3. Where should utility functions (ExtractCodeSnippet, etc.) live?
4. What is the minimal fix to unblock GOgent-037b?

---

## 3. Architectural Context

### 3.1 Established Schema (pkg/session/handoff.go)

```go
// Schema version for handoff format evolution
const HandoffSchemaVersion = "1.1"

type SharpEdge struct {
    // Existing fields (DO NOT CHANGE order or tags)
    File                string `json:"file"`
    ErrorType           string `json:"error_type"`
    ConsecutiveFailures int    `json:"consecutive_failures"`
    Context             string `json:"context,omitempty"`
    Timestamp           int64  `json:"timestamp"`

    // Extended fields (v1.0 compatible - all omitempty)
    ErrorMessage string `json:"error_message,omitempty"`
    Severity     string `json:"severity,omitempty"`
    Resolution   string `json:"resolution,omitempty"`
    ResolvedAt   int64  `json:"resolved_at,omitempty"`
}
```

**Design principles documented:**
- v1.0 backward compatibility via omitempty
- Old readers ignore new fields
- Field order stability for JSON marshaling
- Versioned schema evolution

### 3.2 New Type (pkg/memory/sharp_edge.go)

```go
type SharpEdge struct {
    Timestamp           int64  `json:"ts"`                        // ⚠️ INCOMPATIBLE
    Type                string `json:"type"`                      // NEW
    File                string `json:"file"`
    Tool                string `json:"tool"`                      // NEW
    ErrorType           string `json:"error_type"`
    ErrorMessage        string `json:"error_message,omitempty"`
    CodeSnippet         string `json:"code_snippet,omitempty"`   // NEW (037b)
    ConsecutiveFailures int    `json:"consecutive_failures"`
    Status              string `json:"status"`                    // NEW
}
```

**Why this was created:**
- GOgent-037b ticket spec included SharpEdge struct definition
- Agent didn't have visibility into existing pkg/session type
- Ticket showed `json:"ts"` tag (incompatible with established schema)

### 3.3 Ticket Series Dependencies

```
GOgent-037b (in_progress) ─┬─→ GOgent-037c (pending)
                           │
GOgent-035 (completed) ────┼─→ GOgent-037d (pending) ──→ GOgent-041 ──→ 041b, 041c
                           │
GOgent-038b (completed) ───┴─→ GOgent-038c (pending) ──→ GOgent-038d
```

**All pending tickets reference `pkg/memory/sharp_edge.go`:**
- 037c: Extends SharpEdge with AttemptedChange field
- 038c: Uses SharpEdge in FindSimilar() pattern matching
- 038d: Uses SharpEdge in GenerateBlockingResponse()

---

## 4. Field Requirements Analysis

### 4.1 Combined Field Set

| Field | session.SharpEdge | memory.SharpEdge | 037c | 038c/d | Canonical Type |
|-------|-------------------|------------------|------|--------|----------------|
| File | ✅ | ✅ | ✅ | ✅ | ✅ Required |
| ErrorType | ✅ | ✅ | ✅ | ✅ | ✅ Required |
| ConsecutiveFailures | ✅ | ✅ | ✅ | ✅ | ✅ Required |
| Context | ✅ omitempty | ❌ | ❌ | ❌ | ✅ Keep (backward compat) |
| Timestamp | ✅ `timestamp` | ✅ `ts` | - | - | ✅ Use `timestamp` |
| ErrorMessage | ✅ omitempty | ✅ omitempty | ✅ | ✅ | ✅ Required |
| Severity | ✅ omitempty | ❌ | ❌ | ❌ | ✅ Keep (backward compat) |
| Resolution | ✅ omitempty | ❌ | ❌ | ❌ | ✅ Keep (backward compat) |
| ResolvedAt | ✅ omitempty | ❌ | ❌ | ❌ | ✅ Keep (backward compat) |
| Type | ❌ | ✅ | ✅ | ❌ | ➕ Add (omitempty) |
| Tool | ❌ | ✅ | ✅ | ✅ | ➕ Add (omitempty) |
| CodeSnippet | ❌ | ✅ | ✅ | ✅ | ➕ Add (omitempty) |
| Status | ❌ | ✅ | ✅ | ✅ | ➕ Add (omitempty) |
| AttemptedChange | ❌ | ❌ | ✅ | ❌ | ➕ Add (omitempty) |

### 4.2 Proposed Unified Type

```go
// SharpEdge represents a debugging loop or gotcha discovered
// NOTE: Stays on schema v1.0 - all new fields are optional (omitempty)
type SharpEdge struct {
    // Existing fields (DO NOT CHANGE order or tags)
    File                string `json:"file"`
    ErrorType           string `json:"error_type"`
    ConsecutiveFailures int    `json:"consecutive_failures"`
    Context             string `json:"context,omitempty"`
    Timestamp           int64  `json:"timestamp"`

    // Extended fields (v1.0 compatible - all omitempty)
    ErrorMessage    string `json:"error_message,omitempty"`
    Severity        string `json:"severity,omitempty"`
    Resolution      string `json:"resolution,omitempty"`
    ResolvedAt      int64  `json:"resolved_at,omitempty"`

    // NEW FIELDS (v1.2 - GOgent-037b/c, 038c/d series)
    Type            string `json:"type,omitempty"`              // "sharp_edge"
    Tool            string `json:"tool,omitempty"`              // "Edit", "Write", "Bash"
    CodeSnippet     string `json:"code_snippet,omitempty"`      // 037b
    Status          string `json:"status,omitempty"`            // "pending_review", "resolved"
    AttemptedChange string `json:"attempted_change,omitempty"`  // 037c
}
```

---

## 5. Decision Matrix

| Option | Ticket Series Impact | Schema Impact | Effort | Risk |
|--------|---------------------|---------------|--------|------|
| **A. Extend session.SharpEdge, delete memory type** | ✅ None (update ticket specs) | ✅ v1.2 bump | Low | Low |
| **B. Keep memory.SharpEdge, update JSON tags** | ⚠️ None | ❌ Two sources of truth | Medium | Medium |
| **C. Create shared types package** | ⚠️ Update all imports | ✅ Clean separation | High | Medium |
| **D. Keep both types incompatible** | ❌ Data corruption | ❌ Breaking | None | **CRITICAL** |

---

## 6. Recommended Solution: Option A

### 6.1 Immediate Actions (Unblock 037b)

1. **Extend `pkg/session/handoff.go` SharpEdge type** with new fields
2. **Delete `pkg/memory/sharp_edge.go`** (untracked, safe)
3. **Move `ExtractCodeSnippet()` to `pkg/session/sharp_edge_utils.go`** (new file)
4. **Bump schema version** to "1.2"

### 6.2 Location Decision for Utility Functions

**Option A1: Keep in pkg/session/**
- Pros: Single package for all SharpEdge-related code
- Cons: session package gets larger

**Option A2: Create pkg/session/sharpedge/ subpackage**
- Pros: Clear organization
- Cons: Import path changes

**Recommendation: Option A1** - Keep in pkg/session/ for simplicity

### 6.3 Ticket Spec Updates Required

| Ticket | Change |
|--------|--------|
| 037b | Change `pkg/memory/sharp_edge.go` → `pkg/session/handoff.go` + `pkg/session/sharp_edge_utils.go` |
| 037c | Change `pkg/memory/sharp_edge.go` → `pkg/session/handoff.go` |
| 038c | Change `pkg/memory/pattern_matching.go` → `pkg/session/pattern_matching.go` or use `session.SharpEdge` |
| 038d | Change `pkg/memory/responses.go` → `pkg/session/responses.go` or use `session.SharpEdge` |

---

## 7. Implementation Plan

### Phase 1: Fix Type (NOW)

```bash
# 1. Add new fields to session.SharpEdge
# Edit pkg/session/handoff.go

# 2. Create utility file
# Create pkg/session/sharp_edge_utils.go with:
#   - ExtractCodeSnippet()
#   - ExtractAttemptedChange() (stub for 037c)

# 3. Delete duplicate
rm pkg/memory/sharp_edge.go
rm pkg/memory/sharp_edge_test.go  # if exists

# 4. Update schema version
# HandoffSchemaVersion = "1.2"

# 5. Run tests
go test ./pkg/session/...
```

### Phase 2: Update Tickets (After fix)

Update ticket markdown files to reference correct package:
- `pkg/memory/sharp_edge.go` → `pkg/session/handoff.go`
- `pkg/memory/pattern_matching.go` → `pkg/session/pattern_matching.go`

### Phase 3: Complete 037b

With unified type in place, complete remaining 037b acceptance criteria:
- [x] ExtractCodeSnippet implemented (move from memory to session)
- [x] CodeSnippet field exists
- [ ] Tests passing
- [ ] Coverage ≥80%

---

## 8. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Existing code uses memory.SharpEdge | **None** (grep verified) | N/A | Already verified |
| Ticket specs reference wrong package | **Certain** | Low | Update ticket MD files |
| Tests reference wrong package | **None** | N/A | Tests are in pkg/memory/ which we're deleting |
| Backward compat break | **None** | N/A | All new fields are omitempty |

---

## 9. Answer to Primary Question

**YES, we can and MUST consolidate NOW.**

### Why it's safe:
1. `pkg/memory/sharp_edge.go` is **untracked** - no production dependency
2. `session.SharpEdge` has **explicit versioning** - adding fields is safe
3. No existing code imports `memory.SharpEdge` - **verified via grep**
4. All new fields use `omitempty` - **backward compatible**

### Why it's necessary:
1. JSON tag mismatch (`ts` vs `timestamp`) will cause **silent data loss**
2. Proceeding creates **technical debt** and **schema fragmentation**
3. Ticket series (037c-041c) all assume `pkg/memory` - fixing now prevents cascade

### Minimal fix to unblock:
```go
// Add to pkg/session/handoff.go SharpEdge struct:
Type            string `json:"type,omitempty"`
Tool            string `json:"tool,omitempty"`
CodeSnippet     string `json:"code_snippet,omitempty"`
Status          string `json:"status,omitempty"`
```

Then move `ExtractCodeSnippet()` from `pkg/memory/` to `pkg/session/`.

---

## 10. Conclusion

| Question | Answer |
|----------|--------|
| Is this a conflict? | **YES** - incompatible JSON tags |
| Does it break 037b-041c? | **NO** - if we fix now (all are pending/in_progress) |
| Simple fix available? | **YES** - extend session type, delete memory type |
| Risk of fixing now? | **LOW** - untracked file, no dependencies |
| Risk of NOT fixing? | **HIGH** - data corruption, schema fragmentation |

**Recommendation: Execute fix immediately, then resume GOgent-037b.**

---

## Anti-Scope

This GAP analysis does NOT cover:
- Redesigning the SharpEdge schema
- Moving to a shared types package
- Changing JSON tag conventions
- Modifying existing committed code behavior

Focus is strictly on: **Unify types, unblock 037b, prevent data corruption.**
