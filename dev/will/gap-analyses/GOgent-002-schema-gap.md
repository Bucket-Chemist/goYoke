# GAP Analysis: GOgent-002/002b & Routing Schema Implementation

**Date:** 2026-01-16
**Target:** `routing-schema.json` vs. Ticket Definitions (GOgent-002/002b)
**Status:** CRITICAL DISCREPANCIES FOUND

## 1. Executive Summary

The current definition of tickets `GOgent-002` and `GOgent-002b` in `01-week1-foundation-events.md` is **insufficient** and **factually incorrect** when compared to the actual production `routing-schema.json` (v2.2.0).

Proceeding with the current ticket definitions would result in:
1.  **Data Loss:** Critical fields like `subagent_types.allows_write` and `meta_rules` enforcement patterns would be ignored.
2.  **Runtime Errors:** The Go struct tags in the plan do not match the JSON types (e.g., `Tools` is often an array, but implied as varying types in loose descriptions).
3.  **Security Gaps:** Missing fields like `enforced_by` and `set_by` in `delegation_ceiling` obscure the security provenance of the configuration.

**Recommendation:** Halt implementation of GOgent-002. Rewrite the ticket specifications immediately to reflect the v2.2.0 schema documented below.

## 2. Detailed Schema Gap Analysis

The following table highlights the critical differences between the proposed `GOgent-002b` struct definitions and the actual JSON schema.

| Component | Proposed in GOgent-002b | Actual `routing-schema.json` (v2.2.0) | Impact |
| :--- | :--- | :--- | :--- |
| **TierConfig.Tools** | `interface{}` or `[]string` | `[]string` (Consistent string array) | The schema uses `"*"` as a string inside the array for Opus, not as a raw string type. Go type should be `[]string`. |
| **SubagentType** | `Description`, `Tools` | **+** `AllowsWrite` (bool), `RespectsAgentYaml` (bool), `UseFor` ([]string), `Rationale` (string) | **CRITICAL**. Missing `AllowsWrite` is a major security flaw. The Go agent would not know if a subagent is read-only. |
| **DelegationCeiling** | `Description`, `Default`, `Sources`, `File`... | `Description`, `File`, `SetBy`, `EnforcedBy`, `Values`, `Note`, `Override`, `Calculation` (Map) | The actual schema tracks *who* enforces the ceiling and *how* it is calculated. |
| **MetaRules** | `Rules` (map[string]interface{}) | Map of specific `MetaRule` objects: `DetectionPatterns`, `TargetFiles`, `Enforcement`, `Guidance` | The "interface{}" approach is lazy and dangerous. We need concrete types for rule enforcement logic. |
| **BlockedPatterns** | `Patterns` ([]string) | `Patterns` is a **list of objects**, each containing `pattern`, `reason`, `alternative`, `cost_impact`. | **CRITICAL**. The current plan treats patterns as simple strings. The actual schema is rich objects providing user feedback. |
| **CostThresholds** | `PerEvent`, `PerSession` | `ScoutMaxCost`, `ExplorationMaxCost` | The cost model in the plan (per session) does not match the actual schema (phase-based caps). |

## 3. Critical Analysis of Implementation Plan (002/002b)

### Is it substantial enough?
**No.** The current plan represents a "happy path" translation that ignores the complexity accumulated in the Bash implementation (v2.2).

### Specific Failings:
1.  **Type Safety Illusion:** The plan relies heavily on `map[string]interface{}` for "future proofing," but this defeats the purpose of moving to Go (type safety). The schema is stable enough to demand concrete types.
2.  **Validation Logic Missing:** The tickets define the *structs* but miss the *semantic validation* required by the new fields (e.g., checking `valid_tiers` in overrides, or verifying `subagent_type` compatibility).
3.  **Version Drift:** The tickets describe a generic version of the schema (likely v1.0 or v1.5), while the system is running v2.2.0.

## 4. Revised Implementation Plan (Actionable)

We must update `GOgent-002` and `GOgent-002b` to reflect the following concrete Go structures.

### Step 1: Update `pkg/routing/schema.go` Specification

The structs must be defined as follows (abbreviated for clarity, see full implementation for tags):

```go
type Schema struct {
    Version           string                  `json:"version"`
    Tiers             map[string]TierConfig   `json:"tiers"`
    // ... (standard fields)
    SubagentTypes     map[string]SubagentType `json:"subagent_types"` // Critical fix
    BlockedPatterns   BlockedPatternsConfig   `json:"blocked_patterns"` // Critical fix
}

type SubagentType struct {
    Description       string   `json:"description"`
    Tools             []string `json:"tools"`
    AllowsWrite       bool     `json:"allows_write"`       // NEW: Security control
    RespectsAgentYaml bool     `json:"respects_agent_yaml"` // NEW: Compliance control
    UseFor            []string `json:"use_for"`
}

type BlockedPattern struct {
    Pattern     string `json:"pattern"`     // Regex string
    Reason      string `json:"reason"`      // For user error message
    Alternative string `json:"alternative"` // Suggestion
    CostImpact  string `json:"cost_impact"` // Justification
}
```

### Step 2: Update Ticket `GOgent-002b`

**New Acceptance Criteria:**
*   [ ] `SubagentType` struct includes `AllowsWrite` and `RespectsAgentYaml` fields.
*   [ ] `BlockedPatterns` parses as a slice of structs, not strings.
*   [ ] `TierConfig.Tools` is strictly `[]string` (handle `["*"]` logic in code, not type system).
*   [ ] JSON unmarshaling test uses the **actual** `~/.claude/routing-schema.json` from the environment, not a fixture, to ensure production compatibility.

### Step 3: Immediate Next Actions

1.  **Do not** generate code based on the old `01-week1-foundation-events.md` description.
2.  **Generate** the `pkg/routing/schema.go` file using the `routing-schema.json` found in `~/.claude/` as the ground truth.
3.  **Verify** the parser against the live file immediately (Integration Test first).

## 5. Conclusion

The "fragmented" nature of the boundaries observed is a result of the documentation lagging behind the rapid evolution of the Bash prototype. The Go implementation must skip the "planned" state and jump directly to the "actual" state (v2.2.0).

**Verdict:** The plan for 002/002b is **NOT substantial enough** and requires the specific schema updates detailed above before coding begins.
