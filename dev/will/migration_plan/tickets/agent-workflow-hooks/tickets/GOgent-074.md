---
id: GOgent-074
title: Update Systems Architecture for Merged PostToolUse Handler
status: pending
time_estimate: 0.5h
dependencies: ["GOgent-072"]
priority: medium
week: 4
tags: ["documentation", "week-4"]
tests_required: false
acceptance_criteria_count: 4
---

### GOgent-074: Update Systems Architecture for Merged PostToolUse Handler

**Time**: 0.5 hours
**Dependencies**: GOgent-072 (merged CLI implementation)

**Task**:
Update documentation to reflect merged PostToolUse handler.

**File**: `docs/systems-architecture-overview.md`

**Changes**:

1. Update hook event flow diagram:
```
PostToolUse ──→ gogent-sharp-edge ──→ Failure tracking
                                  ├──→ Counter increment
                                  ├──→ Reminder injection (every 10)
                                  └──→ Auto-flush (every 20)
```

2. Remove reference to separate gogent-attention-gate CLI

3. Document combined behavior:
- Sharp edge detection (existing)
- Tool counter management (new)
- Routing compliance reminders (new)
- Pending learnings auto-flush (new)

4. Add configuration section for thresholds

**Acceptance Criteria**:
- [ ] Hook event flow diagram updated
- [ ] Single PostToolUse handler documented
- [ ] Counter/reminder/flush behavior explained
- [ ] Configuration options documented

---
