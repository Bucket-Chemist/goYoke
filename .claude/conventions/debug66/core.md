---
description: Debug 66 core principles and orchestration. Triggered by "execute debug 66" or "debug 66". Read this first, then language-specific rules.
globs:
alwaysApply: false
---

# Debug 66 - Core Principles

## Philosophy

Debug 66 instruments code to answer: **"What happened, in what order, with what data?"**

The trace should tell a complete story readable by both humans and AI, enabling root cause identification without re-running with breakpoints.

## Orchestration Protocol

When "execute debug 66" is invoked:

1. **Identify targets:** Functions, methods, classes, or code blocks suspected of causing issues
2. **Determine instrumentation level:** Default to Level 2 (Standard) unless specified
3. **Read language-specific rule** and apply instrumentation patterns
4. **Preserve original logic:** Instrumentation must not alter behavior
5. **Provide removal instructions:** Comment markers for easy cleanup

## Universal Instrumentation Points

### 1. Boundary Tracing
```
ENTER → Log function name, timestamp, call context
EXIT  → Log return value (type + summary), duration
```

### 2. Argument Inspection
- Log type, shape/length, and representative sample
- For collections: length + first/last elements
- For objects: key attributes or `toString()` summary

### 3. Data State Checkpoints
Before critical operations (transforms, I/O, API calls), log:
- Variable name
- Dimensions/length
- Type/class
- Sample values (head/tail)
- Null/NA/NaN counts where applicable

### 4. Control Flow Markers
- Which branch taken (if/else/switch)
- Loop iteration identifiers
- Early returns with reason

### 5. Error Context (Hybrid Approach)
Wrap suspicious blocks to **capture context, then re-raise**:
```
TRY:
    [original code]
CATCH error:
    LOG "ERROR in [location]: [error message]"
    LOG "State at failure: [relevant variables]"
    RE-RAISE error  # Preserve original stack trace
```

## Output Format Standards

### Prefix Convention
```
[D66] ─── ENTER functionName ───────────────────
[D66]   ARG: paramName = <summary>
[D66]   STEP: description of action
[D66]   DATA: varName | type | dims | sample
[D66]   BRANCH: condition → TRUE/FALSE
[D66]   ITER: loop [i/n] processing: identifier
[D66]   RESULT: calledFunction → <summary>
[D66]   ERROR: message | state snapshot
[D66] ─── EXIT functionName (duration) → <return summary> ───
```

### Indentation
- Base level: no indent
- Nested calls: +2 spaces per depth level
- Use consistent prefix `[D66]` for easy grep filtering

## Instrumentation Levels

### Level 1 - Light
- Entry/exit with timing
- Arguments (type + length only)
- Return value summary
- Errors with context

### Level 2 - Standard (Default)
- Everything in Level 1
- Major logic steps (before/after)
- Conditional branch taken
- Loop iteration markers (every Nth or key iterations)
- Results from important function calls

### Level 3 - Deep
- Everything in Level 2
- Full data state inspection (head, structure, summary stats)
- All loop iterations
- Memory/resource usage where available
- Intermediate computation results
- Stack depth tracking

## Cleanup Protocol

All instrumentation should be marked for easy removal:

```
# [D66:START] ─────────────────────────
<instrumentation code>
# [D66:END] ───────────────────────────
```

Provide cleanup command: `grep -v "D66" file` or language-specific equivalent.

## Anti-Patterns to Avoid

1. **Don't log sensitive data** (passwords, tokens, PII)
2. **Don't log inside tight loops** without sampling (performance)
3. **Don't alter return values** or control flow
4. **Don't swallow exceptions** - always re-raise after logging
5. **Don't log binary data** - log metadata only (size, type, hash)
