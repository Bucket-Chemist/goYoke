# SessionStart Hook Compatibility

This document verifies that Go-generated handoff markdown is compatible with the `load-routing-context.sh` SessionStart hook.

## Overview

**Hook**: `~/.claude/hooks/load-routing-context.sh`
**Trigger**: SessionStart (startup, resume)
**Purpose**: Inject routing schema and previous session handoff context

## Hook Behavior Analysis

### Parsing Method

The hook uses **simple text extraction** rather than structured parsing:

```bash
# Line 39 of load-routing-context.sh
if [[ "$session_type" == "resume" ]] && [[ -f "$HANDOFF_FILE" ]]; then
    handoff_summary=$(head -30 "$HANDOFF_FILE" 2>/dev/null || echo "No handoff available")
    context_parts+=("PREVIOUS SESSION HANDOFF:\n$handoff_summary")
fi
```

**Key Characteristics**:
- Reads **first 30 lines only** of `.claude/memory/last-handoff.md`
- No regex parsing or section extraction
- Displays content verbatim in additionalContext field
- Works with any text format

### Sections Used

The hook doesn't parse specific sections - it reads the entire first 30 lines. However, for effective context loading, the following should appear early:

| Section | Expected Position | Purpose |
|---------|-------------------|---------|
| Session Context | Lines 3-10 | Session ID, project, active ticket |
| Session Metrics | Lines 12-16 | Tool calls, errors, violations |
| Git State | Lines 18-30 | Branch, dirty status, uncommitted files |
| Sharp Edges | After line 30 | Debugging failures (supplementary) |
| Routing Violations | After line 30 | Tier mismatches (supplementary) |

### Breaking Changes

**What would break compatibility**:
- Moving Session Context below line 30
- Outputting raw JSON instead of markdown
- Including Go-specific artifacts (`nil`, `map[string]interface{}`)
- Changing file location (must be `.claude/memory/last-handoff.md`)

**What is safe**:
- Adding sections after line 30
- Reordering sections after line 30
- Changing formatting (bold, bullets, etc.)
- Adding more detail to existing sections (if line count stays manageable)

## Go Markdown Format Compatibility

### Implementation

**File**: `pkg/session/handoff_markdown.go`
**Function**: `RenderHandoffMarkdown(h *Handoff) string`

### Format Structure

```markdown
# Session Handoff - 2026-01-20 14:23:45

## Session Context

- **Session ID**: test-session-028f
- **Project**: /home/doktersmol/Documents/goYoke
- **Active Ticket**: goYoke-028f
- **Phase**: compatibility-testing

## Session Metrics

- **Tool Calls**: 42
- **Errors Logged**: 2
- **Routing Violations**: 1

## Git State

- **Branch**: master
- **Status**: Uncommitted changes present
- **Uncommitted Files**:
  - test/compatibility/context_loading_test.sh
  - docs/compatibility/sessionstart-hook.md

## Sharp Edges

- **pkg/session/handoff.go**: type_mismatch (3 consecutive failures)
  - Context: JSON unmarshaling failed - expected struct, got map

## Routing Violations

- **tech-docs-writer**: wrong_subagent_type (expected: general-purpose, actual: Explore)

## Immediate Actions

1. Fix sharp edge in handoff.go
   - Resolve type mismatch
```

### Compatibility Assessment

✅ **FULLY COMPATIBLE**

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Critical info in first 30 lines | ✅ | Session Context, Metrics, Git State all appear before line 30 |
| Human-readable format | ✅ | Standard markdown with headers and bullets |
| No Go artifacts | ✅ | Struct formatting uses strings, not Go internal types |
| Timestamp readability | ✅ | Uses `2006-01-02 15:04:05` format, not Unix epoch |
| Markdown list syntax | ✅ | Uses `- **Field**: value` pattern consistently |
| Proper section ordering | ✅ | Most important sections appear first |

### Line Count Analysis

Typical line distribution:
- Lines 1-2: Header
- Lines 3-10: Session Context (~7 lines)
- Lines 12-16: Session Metrics (~5 lines)
- Lines 18-30: Git State (~12 lines)
- Lines 31+: Sharp Edges, Routing Violations, Actions

**Margin**: ~8 lines remaining in first 30 for expansion.

## Testing

### Test Script

**Location**: `test/compatibility/context_loading_test.sh`

### Test Coverage

The compatibility test verifies:

1. **Generation Phase**
   - `goyoke-archive` CLI is installed
   - Mock session data can be created
   - Markdown generation succeeds

2. **Structure Phase**
   - All expected section headers present
   - Sections appear in correct order
   - Critical sections within first 30 lines

3. **Hook Simulation Phase**
   - `head -30` extraction works
   - Extracted content contains session info
   - Metrics and violation counts are readable

4. **Content Verification Phase**
   - Sharp edge details are complete
   - Routing violation information is present
   - Git state includes uncommitted files

5. **Format Compatibility Phase**
   - No JSON leaked into markdown
   - Standard markdown list syntax used
   - Timestamps are human-readable
   - No Go-specific artifacts present

### Running Tests

```bash
# Run compatibility test only
bash test/compatibility/context_loading_test.sh

# Run full ecosystem test suite (includes compatibility)
make test-ecosystem
```

### Expected Output

```
═══════════════════════════════════════════════════════════════════════════════
Context Loading Compatibility Test
═══════════════════════════════════════════════════════════════════════════════

Phase 1: Generate Go handoff markdown
───────────────────────────────────────────────────────────────────────────────
Test 1: goyoke-archive CLI is installed... PASS
Test 2: Create mock session event... PASS
Test 3: Create mock handoffs.jsonl with sample data... PASS
Test 4: Run goyoke-archive to generate markdown... PASS
Test 5: Verify last-handoff.md was created... PASS

Phase 2: Analyze Go-generated markdown structure
───────────────────────────────────────────────────────────────────────────────
Test 6: Markdown contains header... PASS
Test 7: Markdown contains Session Context section... PASS
Test 8: Markdown contains Session Metrics section... PASS
Test 9: Markdown contains Git State section... PASS
Test 10: Markdown contains Sharp Edges section... PASS
Test 11: Markdown contains Routing Violations section... PASS
Test 12: Critical sections appear in first 30 lines... PASS

Phase 3: Simulate load-routing-context.sh parsing
───────────────────────────────────────────────────────────────────────────────
Test 13: Hook script exists... PASS
Test 14: Simulate hook's head -30 extraction... PASS
Test 15: Extracted content contains session info... PASS
Test 16: Extracted content contains tool call count... PASS
Test 17: Extracted content contains routing violations count... PASS
Test 18: Extracted content is readable markdown... PASS

Phase 4: Content verification
───────────────────────────────────────────────────────────────────────────────
Test 19: Sharp edge details are present... PASS
Test 20: Routing violation details are present... PASS
Test 21: Git state shows uncommitted files... PASS
Test 22: Actions section is present... PASS

Phase 5: Format compatibility checks
───────────────────────────────────────────────────────────────────────────────
Test 23: No JSONL content leaked into markdown... PASS
Test 24: Markdown uses standard list syntax... PASS
Test 25: Timestamps are human-readable... PASS
Test 26: No Go-specific artifacts in output... PASS

═══════════════════════════════════════════════════════════════════════════════
Test Summary
═══════════════════════════════════════════════════════════════════════════════
Tests run:    26
Tests passed: 26
Tests failed: 0

✅ COMPATIBILITY VERIFIED
```

## Fixes Applied

**None required.** Go markdown generator is fully compatible out-of-the-box.

## Maintenance

### When Modifying handoff_markdown.go

Before committing changes:

1. **Run compatibility test**:
   ```bash
   bash test/compatibility/context_loading_test.sh
   ```

2. **Check line count**:
   ```bash
   head -30 .claude/memory/last-handoff.md | wc -l
   ```

3. **Verify critical sections visible**:
   ```bash
   head -30 .claude/memory/last-handoff.md | grep -E "Session Context|Session Metrics|Git State"
   ```

4. **Ensure no Go artifacts**:
   ```bash
   grep -E "(nil|<nil>|\[\]interface|map\[string\])" .claude/memory/last-handoff.md
   ```

### Monitoring Guidelines

| Metric | Threshold | Action if Exceeded |
|--------|-----------|-------------------|
| Critical sections line count | >28 lines | Condense formatting or move less critical info later |
| Go artifacts in output | Any instance | Fix struct formatting in handoff_markdown.go |
| Test failures | Any failure | Debug and fix before merging |
| Hook read time | >100ms | Optimize file I/O or reduce content |

## Conclusion

### Compatibility Status

✅ **FULLY COMPATIBLE**

The Go-generated handoff markdown works seamlessly with the `load-routing-context.sh` SessionStart hook because:

1. **Hook uses simple extraction** - No complex parsing to break
2. **Critical info appears early** - All important sections in first 30 lines
3. **Format is clean markdown** - Human-readable with no Go artifacts
4. **Section ordering is optimal** - Most important context loads first

### Risk Assessment

**Risk Level**: ✅ **VERY LOW**

The simple `head -30` extraction method is inherently stable. Breaking changes would require:
- Completely restructuring markdown output order (unlikely)
- Moving to non-text format (blocked by requirements)
- Introducing Go-internal representations (caught by tests)

### Confidence

**High confidence** that switching from Bash to Go handoff generation will NOT break the SessionStart hook:

- ✅ 26 compatibility tests pass
- ✅ Hook simulation successful
- ✅ Format matches expectations
- ✅ No Go artifacts present
- ✅ Critical info in first 30 lines

## References

- **Hook Implementation**: `~/.claude/hooks/load-routing-context.sh:39`
- **Go Markdown Generator**: `pkg/session/handoff_markdown.go`
- **Compatibility Test**: `test/compatibility/context_loading_test.sh`
- **Parsing Logic Documentation**: `test/compatibility/context-loading.md`
- **Ticket**: `migration_plan/tickets/session_archive/028f.md`
