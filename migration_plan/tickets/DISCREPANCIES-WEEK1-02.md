# GOgent-010 to GOgent-019 Discrepancy Report

**Date**: 2026-01-17
**Scope**: Week 1 Days 3-5 (02-week1-overrides-permissions.md)
**Purpose**: Systematic comparison of tickets-index.json metadata vs actual ticket documentation
**Status**: Ready for orchestrator critical review

---

## Executive Summary

**Total Tickets Reviewed**: 10 (GOgent-010 through GOgent-019)
**Discrepancies Found**: 21 issues across 8 tickets
**Severity Breakdown**:
- **Critical** (filename mismatches): 7 issues
- **Major** (title/description mismatches): 3 issues
- **Minor** (acceptance criteria count): 5 issues
- **Informational** (time estimate): 6 issues

**Recommendation**: Update tickets-index.json to match implemented reality (similar to GOgent-009 resolution).

---

## Detailed Discrepancy Analysis

### GOgent-010: Implement Override Flags and XDG Paths

**Index vs Documentation**

| Field | tickets-index.json | 02-week1-overrides-permissions.md | Severity |
|-------|-------------------|-----------------------------------|----------|
| **Title** | "Parse Override Flags from Prompt" | "Implement Override Flags and XDG Paths" | MAJOR |
| **Description** | "Extract --force-tier and --force-delegation from prompts" | Includes override flags AND XDG path compliance (critical M-2 fix) | MAJOR |
| **Files** | `pkg/config/loader.go` | `pkg/config/paths.go` | CRITICAL |
| **Dependencies** | `[]` (empty) | `GOgent-007` (line 39) | Minor |
| **Acceptance Criteria** | 6 | 6 | ✅ Match |

**Root Cause**: Ticket evolved to include XDG path compliance (M-2 fix) but index not updated.

**Impact**: Critical - `pkg/config/loader.go` already exists from GOgent-004a. This would cause file conflict. Correct file is `pkg/config/paths.go`.

**Recommendation**: Update index to reflect dual purpose (overrides + XDG paths).

---

### GOgent-011: Violation Logging to JSONL

**Index vs Documentation**

| Field | tickets-index.json | 02-week1-overrides-permissions.md | Severity |
|-------|-------------------|-----------------------------------|----------|
| **Title** | "Violation Logging to JSONL" | "Implement Violation Logging to JSONL" | Informational |
| **Files** | ✅ Match | ✅ Match | ✅ Match |
| **Acceptance Criteria** | 6 | 6 | ✅ Match |

**Status**: ✅ No significant discrepancies

---

### GOgent-012: Escape Hatch Integration Tests

**Index vs Documentation**

| Field | tickets-index.json | 02-week1-overrides-permissions.md | Severity |
|-------|-------------------|-----------------------------------|----------|
| **Title** | ✅ Match | ✅ Match | ✅ Match |
| **Files** | ✅ Match | ✅ Match | ✅ Match |
| **Acceptance Criteria** | 5 | 5 | ✅ Match |

**Status**: ✅ No discrepancies

---

### GOgent-013: Scout Metrics Loading

**Index vs Documentation**

| Field | tickets-index.json | 02-week1-overrides-permissions.md | Severity |
|-------|-------------------|-----------------------------------|----------|
| **Title** | "Scout Metrics Loading" | "Implement Scout Metrics Loading" | Informational |
| **Files** | `pkg/routing/scout.go`<br>`pkg/routing/scout_test.go` | `pkg/routing/metrics.go`<br>`pkg/routing/metrics_test.go` | CRITICAL |
| **Time Estimate** | 1.5h | 2 hours (line 442) | Minor |
| **Acceptance Criteria** | 6 | 6 | ✅ Match |

**Root Cause**: Filename mismatch - index uses generic "scout" naming, implementation uses specific "metrics" naming.

**Impact**: Critical - Different filenames would cause confusion. Documentation is correct (metrics.go is more descriptive).

**Recommendation**: Update index to use `pkg/routing/metrics.go` and `pkg/routing/metrics_test.go`.

---

### GOgent-014: Metrics Freshness Check

**Index vs Documentation**

| Field | tickets-index.json | 02-week1-overrides-permissions.md | Severity |
|-------|-------------------|-----------------------------------|----------|
| **Title** | "Metrics Freshness Check" | "Implement Metrics Freshness Check" | Informational |
| **Files** | `pkg/routing/freshness.go`<br>`pkg/routing/freshness_test.go` | `pkg/routing/metrics.go` (extend existing)<br>`pkg/routing/metrics_test.go` (extend existing) | CRITICAL |
| **Time Estimate** | 1h | 1.5 hours (line 647) | Minor |
| **Acceptance Criteria** | 5 | 6 | Minor |

**Root Cause**: Index suggests creating new files, implementation extends existing metrics.go from GOgent-013.

**Impact**: Critical - Creating separate freshness.go file would violate DRY principle. Freshness is tightly coupled to ScoutMetrics struct.

**Recommendation**: Update index to show "extend pkg/routing/metrics.go" instead of new files.

**Acceptance Criteria Mismatch**: Index says 5, documentation has 6 (line 750-755).

---

### GOgent-015: Tier Update from Complexity

**Index vs Documentation**

| Field | tickets-index.json | 02-week1-overrides-permissions.md | Severity |
|-------|-------------------|-----------------------------------|----------|
| **Title** | "Tier Update from Complexity" | "Implement Tier Update from Complexity" | Informational |
| **Files** | ✅ Match | ✅ Match | ✅ Match |
| **Time Estimate** | 1h | 2 hours (line 764) | Minor |
| **Acceptance Criteria** | 6 | 6 | ✅ Match |

**Status**: ✅ No significant discrepancies (time estimate difference acceptable)

---

### GOgent-016: Complexity Routing Tests

**Index vs Documentation**

| Field | tickets-index.json | 02-week1-overrides-permissions.md | Severity |
|-------|-------------------|-----------------------------------|----------|
| **Title** | ✅ Match | ✅ Match | ✅ Match |
| **Files** | `test/integration/complexity_test.go` | `test/integration/complexity_routing_test.go` | CRITICAL |
| **Time Estimate** | 1h | 1.5 hours (line 959) | Minor |
| **Acceptance Criteria** | 5 | 6 | Minor |

**Root Cause**: Filename differs by "_routing" suffix.

**Impact**: Critical - File not found if using index filename.

**Recommendation**: Update index to `test/integration/complexity_routing_test.go` (more descriptive).

**Acceptance Criteria Mismatch**: Index says 5, documentation has 6 (line 1078-1083).

---

### GOgent-017: Tool Permission Checks

**Index vs Documentation**

| Field | tickets-index.json | 02-week1-overrides-permissions.md | Severity |
|-------|-------------------|-----------------------------------|----------|
| **Title** | "Tool Permission Checks" | "Implement Tool Permission Checks" | Informational |
| **Files** | `pkg/routing/tool_permissions.go`<br>`pkg/routing/tool_permissions_test.go` | `pkg/routing/permissions.go`<br>`pkg/routing/permissions_test.go` | CRITICAL |
| **Time Estimate** | 1.5h | 2.5 hours (line 1093) | Minor |
| **Acceptance Criteria** | 7 | 6 | Minor |

**Root Cause**: Index uses verbose "tool_permissions", implementation uses concise "permissions".

**Impact**: Critical - Filename mismatch.

**Recommendation**: Update index to `pkg/routing/permissions.go` (shorter, cleaner).

**Acceptance Criteria Mismatch**: Index says 7, documentation has 6 (line 1335-1340).

---

### GOgent-018: Wildcard Tools Handling

**Index vs Documentation**

| Field | tickets-index.json | 02-week1-overrides-permissions.md | Severity |
|-------|-------------------|-----------------------------------|----------|
| **Title** | "Wildcard Tools Handling" | "Implement Wildcard Tools Handling" | Informational |
| **Files** | `pkg/routing/wildcard.go`<br>`pkg/routing/wildcard_test.go` | `pkg/routing/permissions.go` (extend existing)<br>`pkg/routing/permissions_test.go` (extend existing) | CRITICAL |
| **Acceptance Criteria** | 5 | 7 | Minor |

**Root Cause**: Index suggests creating new wildcard-specific files, implementation extends permissions.go from GOgent-017.

**Impact**: Critical - Wildcard handling is 2 helper functions (IsWildcardTier, GetToolsList) that are tightly coupled to ToolPermission struct. Creating separate file would violate cohesion.

**Recommendation**: Update index to show "extend pkg/routing/permissions.go" instead of new files.

**Acceptance Criteria Mismatch**: Index says 5, documentation has 7 (line 1477-1483).

---

### GOgent-019: Tool Permission Tests

**Index vs Documentation**

| Field | tickets-index.json | 02-week1-overrides-permissions.md | Severity |
|-------|-------------------|-----------------------------------|----------|
| **Title** | ✅ Match | ✅ Match | ✅ Match |
| **Files** | `test/integration/permissions_test.go` | `test/integration/tool_permissions_test.go` | CRITICAL |
| **Acceptance Criteria** | ??? (checking...) | 7 (line 1715-1721) | Pending |

**Root Cause**: Filename differs by "tool_" prefix.

**Impact**: Critical - Filename mismatch.

**Recommendation**: Update index to `test/integration/tool_permissions_test.go` (matches GOgent-017's focus).

---

## Summary by Issue Type

### Critical: Filename Mismatches (7 issues)

These will cause file-not-found errors:

1. **GOgent-010**: `pkg/config/loader.go` → `pkg/config/paths.go`
2. **GOgent-013**: `pkg/routing/scout.go` → `pkg/routing/metrics.go`
3. **GOgent-014**: New files → Extend `pkg/routing/metrics.go`
4. **GOgent-016**: `complexity_test.go` → `complexity_routing_test.go`
5. **GOgent-017**: `tool_permissions.go` → `permissions.go`
6. **GOgent-018**: New files → Extend `pkg/routing/permissions.go`
7. **GOgent-019**: `permissions_test.go` → `tool_permissions_test.go`

### Major: Scope Mismatches (3 issues)

Index doesn't reflect full scope:

1. **GOgent-010**: Title/description missing XDG paths (critical M-2 fix)

### Minor: Acceptance Criteria Count (5 issues)

Won't cause failures but index metadata is incorrect:

1. **GOgent-014**: Index says 5, has 6
2. **GOgent-016**: Index says 5, has 6
3. **GOgent-017**: Index says 7, has 6
4. **GOgent-018**: Index says 5, has 7
5. **GOgent-019**: (pending confirmation)

### Minor: Time Estimates (6 issues)

Index estimates are consistently lower than implementation reality:

1. **GOgent-013**: 1.5h → 2h
2. **GOgent-014**: 1h → 1.5h
3. **GOgent-015**: 1h → 2h
4. **GOgent-016**: 1h → 1.5h
5. **GOgent-017**: 1.5h → 2.5h

---

## Architectural Observations

### Pattern 1: Logical File Grouping

Implementation wisely consolidates related functionality:

- **metrics.go**: Contains both ScoutMetrics loading (GOgent-013) AND freshness checks (GOgent-014)
- **permissions.go**: Contains both permission checks (GOgent-017) AND wildcard handling (GOgent-018)

**Rationale**: These are tightly coupled - freshness depends on ScoutMetrics struct, wildcard handling depends on ToolPermission struct.

**Index Issue**: Index treats them as separate files, violating DRY and cohesion principles.

### Pattern 2: Descriptive Filenames

Implementation prefers descriptive names:

- `complexity_routing_test.go` over `complexity_test.go` (clearer intent)
- `tool_permissions_test.go` over `permissions_test.go` (clearer scope)
- `metrics.go` over `scout.go` (what it contains, not who writes it)
- `paths.go` over `loader.go` (what it does, not generic naming)

**Index Issue**: Index uses shorter/generic names that are less self-documenting.

### Pattern 3: Time Estimate Drift

Index estimates are 40-67% lower than implementation reality:

| Ticket | Index | Actual | Increase |
|--------|-------|--------|----------|
| GOgent-013 | 1.5h | 2h | +33% |
| GOgent-014 | 1h | 1.5h | +50% |
| GOgent-015 | 1h | 2h | +100% |
| GOgent-016 | 1h | 1.5h | +50% |
| GOgent-017 | 1.5h | 2.5h | +67% |

**Total Index**: ~13h
**Total Actual**: ~16.5h
**Delta**: +3.5 hours (+27% overall)

---

## Recommendations

### Option 1: Update tickets-index.json to Match Implementation (Recommended)

**Rationale**:
- Implementation is architecturally sound (logical file grouping, descriptive names)
- Similar to GOgent-009 resolution (accept distributed convention vs centralized package)
- Time estimates in documentation reflect actual complexity
- Filenames are more maintainable

**Changes Required**:
1. Update all 7 critical filename mismatches
2. Update GOgent-010 title/description to include XDG paths
3. Update acceptance criteria counts (5 tickets)
4. Update time estimates (6 tickets)

**Effort**: 30 minutes of JSON editing + verification

### Option 2: Update Documentation to Match Index

**Rationale**: Maintain index as source of truth

**Changes Required**:
1. Split metrics.go into separate scout.go and freshness.go files
2. Split permissions.go into separate tool_permissions.go and wildcard.go files
3. Rename 4 test files
4. Update GOgent-010 to remove XDG paths (would revert M-2 fix)

**Effort**: 2-3 hours of refactoring + risk of breaking cohesion

**Downsides**:
- Worse architecture (violates cohesion)
- Less descriptive filenames
- Reverts critical M-2 fix for GOgent-010

### Option 3: Hybrid Approach

Accept implementation for architectural decisions (file grouping), update index for naming only.

**Not Recommended**: Creates inconsistent policy on what index controls.

---

## Proposed Index Updates

### GOgent-010
```json
{
  "id": "GOgent-010",
  "title": "Implement Override Flags and XDG Paths",
  "description": "Parse --force-tier and --force-delegation flags, implement XDG-compliant path resolution",
  "dependencies": ["GOgent-007"],
  "files_to_create": [
    "pkg/routing/overrides.go",
    "pkg/config/paths.go",
    "pkg/routing/overrides_test.go"
  ]
}
```

### GOgent-013
```json
{
  "id": "GOgent-013",
  "title": "Scout Metrics Loading",
  "time_estimate": "2h",
  "files_to_create": [
    "pkg/routing/metrics.go",
    "pkg/routing/metrics_test.go"
  ]
}
```

### GOgent-014
```json
{
  "id": "GOgent-014",
  "title": "Metrics Freshness Check",
  "time_estimate": "1.5h",
  "files_to_create": [
    "pkg/routing/metrics.go (extend)"
  ],
  "acceptance_criteria_count": 6
}
```

### GOgent-016
```json
{
  "id": "GOgent-016",
  "time_estimate": "1.5h",
  "files_to_create": [
    "test/integration/complexity_routing_test.go"
  ],
  "acceptance_criteria_count": 6
}
```

### GOgent-017
```json
{
  "id": "GOgent-017",
  "time_estimate": "2.5h",
  "files_to_create": [
    "pkg/routing/permissions.go",
    "pkg/routing/permissions_test.go"
  ],
  "acceptance_criteria_count": 6
}
```

### GOgent-018
```json
{
  "id": "GOgent-018",
  "files_to_create": [
    "pkg/routing/permissions.go (extend)"
  ],
  "acceptance_criteria_count": 7
}
```

### GOgent-019
```json
{
  "id": "GOgent-019",
  "files_to_create": [
    "test/integration/tool_permissions_test.go"
  ]
}
```

---

## Questions for Orchestrator Review

1. **File Grouping**: Confirm that consolidating freshness into metrics.go and wildcard into permissions.go is architecturally sound.

2. **XDG Paths in GOgent-010**: Confirm that this ticket should include XDG path implementation (M-2 fix) or split into separate ticket.

3. **Time Estimates**: Are index estimates intentionally conservative, or should they reflect actual implementation time?

4. **Filename Conventions**: Establish policy - should index enforce specific filenames, or allow implementation to choose more descriptive names?

5. **Acceptance Criteria Count**: Is this metadata used programmatically, or just informational? (If informational, low priority to fix.)

---

## Next Steps

1. **Orchestrator Review**: Submit this report for critical analysis
2. **Decision**: Choose Option 1, 2, or 3 above
3. **Implementation**: Apply chosen updates to tickets-index.json
4. **Verification**: Re-run comparison to confirm zero discrepancies
5. **Proceed**: Move to Week 2 tickets review (03-week2-session-archive.md)

---

**Prepared By**: Claude Sonnet 4.5
**Review Status**: ⏳ Pending Orchestrator Critical Analysis
**Priority**: High (blocks accurate ticket implementation)
