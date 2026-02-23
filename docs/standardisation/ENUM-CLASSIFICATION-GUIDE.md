# Enum Classification Guide

Principled guide for deciding when to use `enum` vs `string` in stdout schemas.

---

## The Core Principle

Constrained decoding with `--json-schema` enforces enum values at the token level. When a field has `"enum": ["high", "medium", "low"]`, the model can only produce one of those three strings. This is a hard constraint — no other value is possible.

This property is beneficial for protocol values but harmful for classification labels:

**Protocol values** are finite, machine-consumed, and have well-defined semantics that upstream/downstream code depends on. The correct value is determined by the state of execution, not by the model's judgment. These benefit from enum enforcement because:
- There is no "correct" value outside the list
- Downstream code branches on these values (status routing, verdict handling, etc.)
- A model that would produce a novel value (e.g., `"partial-complete"`) is producing a bug, not nuance

**Classification labels** are determined by the model's judgment about context-dependent qualities (confidence, severity, priority). The model should be free to express nuance. Forcing a choice from a closed set when the model would naturally say something more precise is an accuracy loss. With constrained decoding, the model must pick from the enum even if none of the options precisely fits.

| Field Type | Use | Reason |
|-----------|-----|--------|
| Closed protocol value | `enum` | Finite, machine-consumed, no valid alternatives |
| Classification label | `string` | Judgment-dependent, context-sensitive |
| Fixed single value | `const` | Only one correct value exists |

---

## Complete Inventory: Fields Kept as Enum

These fields use enum because they are **closed protocol values**: finite sets consumed by downstream code (team-run status routing, review orchestrator verdict handling, etc.).

| Schema | Field Path | Enum Values | Reason |
|--------|-----------|-------------|--------|
| All schemas | `status` | `complete`, `partial`, `failed` | Protocol completion state; team-run branches on this |
| `staff-architect.json` | `executive_assessment.verdict` | `APPROVE`, `APPROVE_WITH_CONDITIONS`, `REVISE`, `REJECT` | Protocol decision consumed by orchestrator |
| `beethoven.json` | `analysis_perspectives.staff_architect_verdict` | `APPROVE`, `APPROVE_WITH_CONDITIONS`, `REVISE`, `REJECT` | Echoed protocol decision (same closed set) |
| `reviewer.json` | `overall_assessment` | `APPROVE`, `WARNING`, `BLOCK` | Protocol assessment verdict |
| `worker.json` | `files_modified[].action` | `created`, `modified`, `deleted` | Protocol file operation type |
| `worker.json` | `acceptance_criteria_met[].status` | `met`, `not_met`, `partial` | Protocol criterion outcome |
| `einstein.json` | `first_principles_analysis.assumptions_challenged[].validity` | `valid`, `questionable`, `invalid` | Protocol validity classification (finite, well-defined) |
| `reviewer.json` | `structural_health_score.overall` | `A`, `B`, `C`, `D`, `F` | Protocol grade scale |
| `reviewer.json` | `structural_health_score.module_boundaries` | `A`, `B`, `C`, `D`, `F` | Protocol grade scale |
| `reviewer.json` | `structural_health_score.coupling` | `A`, `B`, `C`, `D`, `F` | Protocol grade scale |
| `reviewer.json` | `structural_health_score.testability` | `A`, `B`, `C`, `D`, `F` | Protocol grade scale |
| `reviewer.json` | `structural_health_score.extensibility` | `A`, `B`, `C`, `D`, `F` | Protocol grade scale |

---

## Complete Inventory: Fields Relaxed to String

These fields were previously defined with enum constraints but have been relaxed to plain `string`. The reason in each case: constrained decoding plus enum forces the model to pick from a closed set even when a more nuanced label would be more accurate.

### Einstein Schema (`schemas/stdout/einstein.json`)

| Field Path | Former Enum Values | Reason for Relaxation |
|-----------|-------------------|----------------------|
| `root_cause_analysis.identified_causes[].confidence` | `high`, `medium`, `low` | Subjective assessment; model may have fine-grained confidence |
| `novel_approaches[].feasibility` | `high`, `medium`, `low` | Context-dependent judgment |
| `open_questions[].importance` | `high`, `medium`, `low` | Subjective prioritization |

### Staff-Architect Schema (`schemas/stdout/staff-architect.json`)

| Field Path | Former Enum Values | Reason for Relaxation |
|-----------|-------------------|----------------------|
| `executive_assessment.confidence` | `high`, `medium`, `low` | Subjective; reviewer may want to express "very high" or nuance |
| `issue_register[].severity` | `critical`, `major`, `minor` | Context-dependent label; not a protocol branch point |
| `issue_register[].layer` | `assumptions`, `dependencies`, `failure_modes`, `cost_benefit`, `testing`, `architecture_smells`, `contractor_readiness` | Review layer taxonomy may expand; model should not be blocked from describing a novel layer |
| `failure_mode_analysis[].probability` | `high`, `medium`, `low` | Subjective assessment |
| `failure_mode_analysis[].impact` | `high`, `medium`, `low` | Subjective assessment |

### Beethoven Schema (`schemas/stdout/beethoven.json`)

| Field Path | Former Enum Values | Reason for Relaxation |
|-----------|-------------------|----------------------|
| `convergence_points[].confidence` | `high`, `medium`, `low` | Subjective synthesis confidence |
| `divergence_resolution[].confidence` | `high`, `medium`, `low` | Subjective resolution confidence |
| `risk_assessment[].probability` | `high`, `medium`, `low` | Subjective risk assessment |
| `risk_assessment[].impact` | `high`, `medium`, `low` | Subjective risk assessment |
| `risk_assessment[].source` | `einstein`, `staff-architect`, `synthesis` | New source types (e.g., a third analyst) may be added |
| `assumptions_to_validate[].source` | `einstein`, `staff-architect`, `both` | Same reasoning as above |
| `assumptions_to_validate[].priority` | `high`, `medium`, `low` | Subjective prioritization |
| `open_questions[].importance` | `high`, `medium`, `low` | Subjective prioritization |
| `open_questions[].source` | `einstein`, `staff-architect`, `synthesis` | Source taxonomy may expand |

### Reviewer Schema (`schemas/stdout/reviewer.json`)

| Field Path | Former Enum Values | Reason for Relaxation |
|-----------|-------------------|----------------------|
| `reviewer` | `backend-reviewer`, `frontend-reviewer`, `standards-reviewer`, `architect-reviewer` | New reviewer agents may be added; enum would require schema update per new agent |
| `findings[].severity` | `critical`, `warning`, `info` | Context-dependent label; not a downstream branch point |

---

## Decision Checklist for New Fields

When adding a field to a stdout schema, answer these questions:

1. **Will downstream code branch on this value?**
   If yes (e.g., `if status == "failed" { ... }`), use `enum`.

2. **Is the set of valid values permanently closed?**
   If the set might grow as the system evolves (new agents, new review layers, new source types), use `string`.

3. **Is the value determined by execution state or model judgment?**
   Execution state (did the task complete?) → `enum`.
   Model judgment (how confident is the model?) → `string`.

4. **Is there exactly one correct value for any given run?**
   If yes and it never changes, use `const` (e.g., `schema_id`).

5. **Is this a grading system with a universally understood scale?**
   Academic letter grades (A/B/C/D/F) → `enum` is appropriate because the scale is canonical and closed.

---

## Why This Matters for Constrained Decoding

Without constrained decoding, enum constraints in a schema are validation-only: the model may produce `"very high"` and a validator catches it post-hoc. With constrained decoding, the token sampler enforces the constraint during generation. The model is forced to produce `"high"` even if its internal state would lead it to produce `"very high"` or `"high-to-certain"`.

For protocol fields, this is the correct behavior. For classification labels, this is accuracy loss — the model picks the closest approximation from a restricted set rather than expressing its actual assessment.

The enum/string split in GOgent-Fortress schemas reflects this distinction: constrained decoding is used as a reliability mechanism for protocol fields, not as a normalization mechanism for judgment fields.
