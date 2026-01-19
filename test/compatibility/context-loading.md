# Context Loading Parsing Logic

This document explains how the `load-routing-context.sh` hook parses handoff markdown and verifies compatibility with Go-generated output.

## Hook Parsing Behavior

### Location
`~/.claude/hooks/load-routing-context.sh`

### Parsing Method
**Simple extraction with `head -30`** (line 39)

```bash
if [[ "$session_type" == "resume" ]] && [[ -f "$HANDOFF_FILE" ]]; then
    handoff_summary=$(head -30 "$HANDOFF_FILE" 2>/dev/null || echo "No handoff available")
    context_parts+=("PREVIOUS SESSION HANDOFF:\n$handoff_summary")
fi
```

### Key Characteristics

| Aspect | Details |
|--------|---------|
| **Parsing Type** | Plain text extraction (no regex, no jq) |
| **Lines Read** | First 30 lines only |
| **Format Dependency** | None - any text format works |
| **Section Extraction** | No - entire 30 lines displayed as-is |
| **Breaking Changes Risk** | **Very Low** - only requires readable text in first 30 lines |

### What the Hook Does

1. **Reads first 30 lines** of `.claude/memory/last-handoff.md`
2. **Injects verbatim** into additionalContext field
3. **Displays to user** at session start

The hook does NOT:
- Parse specific markdown sections
- Extract structured data
- Validate format
- Search for specific headers

## Compatibility Requirements

For Go-generated markdown to be compatible, it must:

1. ✅ **Place critical information in first 30 lines**
   - Session Context (ID, project, ticket)
   - Session Metrics (tool calls, errors, violations)
   - Git State (branch, dirty status)

2. ✅ **Use human-readable format**
   - Standard markdown headers
   - Bulleted lists
   - No raw JSON or Go structs

3. ✅ **Avoid Go-specific artifacts**
   - No `nil` or `<nil>`
   - No `[]interface{}` or `map[string]interface{}`
   - No Go stack traces

## Go Markdown Generator Analysis

### File
`pkg/session/handoff_markdown.go`

### Structure
```go
func RenderHandoffMarkdown(h *Handoff) string {
    // Header
    "# Session Handoff - {timestamp}"

    // Session Context (lines 3-10)
    "## Session Context"
    "- **Session ID**: ..."
    "- **Project**: ..."
    "- **Active Ticket**: ..."

    // Session Metrics (lines 12-16)
    "## Session Metrics"
    "- **Tool Calls**: ..."
    "- **Errors Logged**: ..."
    "- **Routing Violations**: ..."

    // Git Info (lines 18-30)
    "## Git State"
    "- **Branch**: ..."
    "- **Status**: ..."
    "- **Uncommitted Files**: ..."

    // Sharp Edges (after line 30)
    // Routing Violations (after line 30)
    // Actions (after line 30)
}
```

### Compatibility Assessment

✅ **Fully Compatible**

- Critical sections (Context, Metrics, Git) appear in lines 3-30
- Sharp Edges and Routing Violations may appear after line 30, but these are supplementary
- Format is clean markdown with no Go artifacts
- Timestamps use human-readable format (`2006-01-02 15:04:05`)

## Testing Strategy

### Test Script
`test/compatibility/context_loading_test.sh`

### Test Phases

#### Phase 1: Generation
- Create mock session data
- Run `gogent-archive` to generate markdown
- Verify output file exists

#### Phase 2: Structure Analysis
- Check for required section headers
- Verify sections appear in correct order
- Confirm critical sections in first 30 lines

#### Phase 3: Hook Simulation
- Simulate `head -30` extraction
- Verify extracted content contains session info
- Check for metrics and violation counts

#### Phase 4: Content Verification
- Validate sharp edge details
- Verify routing violation information
- Check git state formatting

#### Phase 5: Format Compatibility
- Ensure no JSON leaked into markdown
- Verify standard markdown list syntax
- Check timestamps are human-readable
- Confirm no Go-specific artifacts

### Success Criteria

All tests must pass:
- ✅ Markdown generated successfully
- ✅ All expected sections present
- ✅ Critical info in first 30 lines
- ✅ Content is human-readable
- ✅ No Go artifacts in output
- ✅ Hook simulation successful

## Potential Breaking Changes

### What Would Break Compatibility

| Change | Impact | Likelihood |
|--------|--------|------------|
| Moving Session Context below line 30 | Critical info not visible to hook | ❌ Low - template order is stable |
| Adding raw JSON to markdown | Poor readability | ❌ Low - Go struct marshaling is controlled |
| Using Go nil/interface{} in output | Confusing output | ❌ Low - string formatting handles this |
| Changing to non-markdown format | Hook expects text | ❌ Very Low - markdown is requirement |

### What Would NOT Break Compatibility

| Change | Impact | Safe? |
|--------|--------|-------|
| Adding sections after line 30 | Not visible to hook, but safe | ✅ Yes |
| Changing order of sections after line 30 | Hook doesn't parse them | ✅ Yes |
| Adding more detail to existing sections | May push content past line 30 | ⚠️ Monitor line count |
| Formatting changes (bold, bullets) | Hook displays as-is | ✅ Yes |

## Monitoring

### Verification Points

When modifying `handoff_markdown.go`:

1. **Run compatibility test**:
   ```bash
   bash test/compatibility/context_loading_test.sh
   ```

2. **Check line count of critical sections**:
   ```bash
   head -30 .claude/memory/last-handoff.md | wc -l
   ```

3. **Verify Git State appears early**:
   ```bash
   head -30 .claude/memory/last-handoff.md | grep "Git State"
   ```

4. **Ensure no Go artifacts**:
   ```bash
   grep -E "(nil|<nil>|\[\]interface|map\[string\])" .claude/memory/last-handoff.md
   ```

### Continuous Integration

The compatibility test is integrated into the ecosystem test suite:

```bash
make test-ecosystem  # Includes context_loading_test.sh
```

This ensures any changes to `handoff_markdown.go` are automatically validated against hook expectations.

## Conclusion

The `load-routing-context.sh` hook uses a **very simple parsing strategy** (first 30 lines only), which makes compatibility extremely robust. The Go markdown generator is fully compatible because:

1. Critical information appears in first 30 lines
2. Format is clean, readable markdown
3. No Go-specific artifacts are present
4. Section ordering matches hook expectations

**Risk Level**: ✅ **VERY LOW** - Simple extraction method is inherently stable.

## References

- Hook implementation: `~/.claude/hooks/load-routing-context.sh:39`
- Go markdown generator: `pkg/session/handoff_markdown.go`
- Test script: `test/compatibility/context_loading_test.sh`
- Compatibility documentation: `docs/compatibility/sessionstart-hook.md`
