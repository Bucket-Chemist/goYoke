# Architecture Decision: {{TITLE}}

**Decision ID:** {{UUID}}
**Date:** {{ISO_DATE}}
**Requested by:** {{REQUESTING_AGENT}}
**Status:** DECIDED | PENDING_CLARIFICATION | ESCALATED

---

## Decision Summary

{{2-3 sentence summary of what was decided and why. Be specific about the choice made.}}

---

## Context

### Problem Statement

{{What prompted this decision? What question needs answering?}}

### Constraints

| Dimension | Constraint |
|-----------|------------|
| **Hardware** | {{memory limits, target device, VRAM}} |
| **Performance** | {{latency target, throughput requirement}} |
| **Compatibility** | {{ONNX export, framework versions, existing patterns}} |
| **Timeline** | {{urgency, iteration timeline}} |

### Existing Patterns

{{Reference to similar decisions or patterns in the codebase. Include file paths.}}

---

## Options Considered

### Option 1: {{Name}}

**Description:**
{{What this approach entails, in 2-3 sentences.}}

**Key Pattern:**
```python
# Core implementation approach
{{code snippet showing the pattern}}
```

| Dimension | Assessment |
|-----------|------------|
| Complexity | {{Low / Medium / High}} |
| Risk | {{Low / Medium / High}} |
| Compute Cost | {{O(?) with concrete numbers for typical input}} |
| Memory Cost | {{O(?) with concrete numbers}} |
| Accuracy Impact | {{Expected effect on accuracy}} |

**Pros:**
- {{Pro 1}}
- {{Pro 2}}
- {{Pro 3}}

**Cons:**
- {{Con 1}}
- {{Con 2}}

---

### Option 2: {{Name}}

**Description:**
{{What this approach entails}}

**Key Pattern:**
```python
{{code snippet}}
```

| Dimension | Assessment |
|-----------|------------|
| Complexity | {{Low / Medium / High}} |
| Risk | {{Low / Medium / High}} |
| Compute Cost | {{O(?)}} |
| Memory Cost | {{O(?)}} |
| Accuracy Impact | {{Expected effect}} |

**Pros:**
- {{Pro 1}}
- {{Pro 2}}

**Cons:**
- {{Con 1}}
- {{Con 2}}

---

### Option 3: {{Name}} (if applicable)

{{Same structure as above}}

---

## Decision

**Selected:** Option {{N}}: {{Name}}

### Rationale

{{Detailed justification with specific evidence. Reference the analysis above.}}

{{Why this option wins over the alternatives. Be specific.}}

### What We're Sacrificing

{{Explicit acknowledgment of tradeoffs accepted. What does this choice cost us?}}

### Why the Tradeoff is Acceptable

{{Justification for why the sacrifice is worth it in this context.}}

---

## Implementation Guidance

### Files to Modify

| File | Change Type | Description |
|------|-------------|-------------|
| `{{path/to/file.py}}` | Create / Modify | {{Brief description of change}} |
| `{{path/to/other.py}}` | Modify | {{Brief description}} |

### Code Patterns to Follow

**Pattern 1: {{description}}**
```python
{{Concrete code pattern to implement}}
```

**Pattern 2: {{description}}**
```python
{{Another pattern if needed}}
```

### Integration Points

- {{Where this connects to existing code}}
- {{Dependencies to be aware of}}
- {{Order of implementation if it matters}}

### Testing Strategy

- {{How to test the implementation}}
- {{What edge cases to cover}}
- {{Performance benchmarks to run}}

---

## Validation Criteria

Implementation is complete when:

- [ ] {{Specific, verifiable criterion 1}}
- [ ] {{Specific, verifiable criterion 2}}
- [ ] {{Specific, verifiable criterion 3}}
- [ ] All tests pass
- [ ] Code review completed

---

## Follow-up Considerations

- {{Future decision that may be affected by this choice}}
- {{Technical debt to address later}}
- {{Monitoring or metrics to implement}}
- {{Documentation to update}}

---

## Metadata

```json
{
  "decision_id": "{{UUID}}",
  "timestamp": "{{ISO_DATE}}",
  "requesting_agent": "{{AGENT}}",
  "options_considered": ["{{opt1}}", "{{opt2}}", "{{opt3}}"],
  "selected_option": "{{selected}}",
  "confidence": {{0.0-1.0}},
  "affected_files": [
    "{{file1}}",
    "{{file2}}"
  ],
  "estimated_loc_change": {{number}},
  "conventions_referenced": [
    "python.md",
    "python-ml.md"
  ],
  "scouts_spawned": [
    "{{codebase-search if used}}",
    "{{haiku-scout if used}}"
  ]
}
```
