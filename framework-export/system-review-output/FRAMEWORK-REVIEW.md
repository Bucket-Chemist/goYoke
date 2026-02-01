# GOgent-Fortress Framework Review

**Review Date:** 2026-02-01
**Framework Version:** 2.4.0 (routing-schema.json) / 2.3.0 (agents-index.json)
**Reviewer:** Claude Opus (Deep Research Mode)

---

## Executive Summary

The GOgent-Fortress agent framework demonstrates **mature architectural thinking** with sophisticated tiered routing, cost optimization, and sharp edge documentation. The 32-agent ecosystem with explicit subagent types, delegation ceilings, and scout protocols represents production-quality orchestration infrastructure. The framework's enforced conventions and hook-based validation exemplify the principle that "enforcement belongs in code, not documentation."

However, **critical gaps exist in convention currency**. The JavaScript/TypeScript stack conventions (react.md, typescript.md) are anchored to React 18 and TypeScript 5.0, missing transformative React 19 features (React Compiler, Server Components, Actions, use() hook) and TypeScript 5.5+ improvements (inferred type predicates, isolated declarations). Similarly, Go conventions lack Go 1.22+ patterns (range over integers, enhanced HTTP routing, the loop variable fix), and R conventions are missing S7 OOP and dplyr 1.1+ features. These gaps risk generating outdated code patterns in the framework's primary target languages.

The sharp edge documentation is **exemplary for project-specific learnings** (particularly go-tui's subprocess channel guards), but requires expansion for modern framework features. Agent trigger overlap presents moderate routing ambiguity risk, particularly around generic terms like "implement" and "refactor."

### Health Score: B+

| Dimension | Score | Critical Issues |
|-----------|-------|-----------------|
| Agent Scope | B | 3 agents with >8 triggers; moderate trigger overlap |
| Sharp Edges | B+ | Missing React 19/TS 5.5+ sharp edges; excellent project-specific learnings |
| Conventions | C+ | **React 18 (not 19)**, TS 5.0 (not 5.5+), Go 1.21 (not 1.22+), R missing S7/dplyr 1.1+ |
| Routing Schema | A | Excellent tier boundaries, delegation ceiling, scout protocol |
| Skills | A- | Comprehensive coverage; well-structured workflows |
| Rules Alignment | A | Strong consistency between LLM-guidelines.md and agent-behavior.md |

---

## Critical Findings

### CRIT-001: React Conventions Missing React 19 Features
**Severity:** Critical
**Location:** `conventions/react.md`
**Issue:** The convention targets "React 18+" but lacks React 19 patterns that fundamentally change development practices:
- No React Compiler guidance (automatic memoization makes manual useMemo/useCallback often unnecessary)
- No Server Components patterns (`'use server'`, `'use client'` directives)
- No Actions/Forms patterns (useActionState, useOptimistic, useFormStatus)
- No `use()` hook documentation
- No metadata API (native `<title>`, `<meta>` hoisting)

**Impact:** Agents may generate outdated patterns (manual memoization everywhere, no server/client boundary awareness).

**Recommendation:** Add "## React 19+ Features" section covering:
```markdown
## React 19+ Features

### React Compiler
When React Compiler is enabled, manual memoization is largely unnecessary:
- The compiler automatically optimizes re-renders
- Remove new useMemo/useCallback calls (keep existing ones)
- Components MUST follow Rules of React (pure rendering, no prop mutation)

### Server Components (Default)
Components without 'use client' are Server Components by default:
- Can access databases, file systems, API credentials directly
- Cannot use hooks (useState, useEffect) or event handlers
- Use 'use client' at the top of files needing interactivity

### Server Actions
\`\`\`tsx
'use server'
async function createItem(formData: FormData) {
  await db.insert({ name: formData.get('name') });
  revalidatePath('/items');
}
\`\`\`

### Form Handling
\`\`\`tsx
function Form() {
  const [state, formAction, isPending] = useActionState(createItem, null);
  return (
    <form action={formAction}>
      <SubmitButton /> {/* useFormStatus MUST be in child component */}
    </form>
  );
}
\`\`\`
```

---

### CRIT-002: TypeScript Conventions Missing 5.5+ Features
**Severity:** Critical
**Location:** `conventions/typescript.md`
**Issue:** Convention targets "TypeScript 5.0+" but lacks critical 5.5+ features:
- No inferred type predicates (`.filter()` now correctly narrows types)
- No `NoInfer` utility type (5.4)
- No isolated declarations mode documentation
- No `using` keyword for resource management (5.2)

**Impact:** Agents may write unnecessary explicit type predicates and miss performance optimizations from isolated declarations.

**Recommendation:** Add section:
```markdown
## TypeScript 5.5+ Features

### Inferred Type Predicates (5.5)
TypeScript now infers type predicates automatically:
\`\`\`typescript
// No longer need explicit type predicate
const numbers = items.filter(x => x !== undefined);
// Type: T[] (not (T | undefined)[] anymore)
\`\`\`

### NoInfer Utility Type (5.4)
Prevents inference from specific type positions:
\`\`\`typescript
function createConfig<T extends string>(
  options: T[],
  defaultOption?: NoInfer<T>
) {}
\`\`\`

### Isolated Declarations Mode (5.5)
Enable for faster parallel builds in monorepos:
\`\`\`json
{ "compilerOptions": { "isolatedDeclarations": true } }
\`\`\`
Requires explicit return types on all exports.
```

---

### CRIT-003: Go Conventions Missing 1.22+ Features
**Severity:** Critical
**Location:** `conventions/go.md`
**Issue:** Convention lacks Go 1.22+ patterns:
- No range over integers (`for i := range 10`)
- No enhanced HTTP routing (`mux.HandleFunc("GET /users/{id}", handler)`)
- No `slog` structured logging (Go 1.21)
- Loop variable capture examples still show manual `item := item` without noting Go 1.22 fixed this

**Impact:** Agents generate verbose code where modern Go provides cleaner patterns.

**Recommendation:** Add section:
```markdown
## Go 1.22+ Features

### Range Over Integers
\`\`\`go
// PREFERRED (Go 1.22+)
for i := range 10 {
    fmt.Println(i) // 0-9
}

// LEGACY
for i := 0; i < 10; i++ {
    fmt.Println(i)
}
\`\`\`

### Enhanced HTTP Routing
\`\`\`go
mux := http.NewServeMux()
mux.HandleFunc("GET /users/{id}", getUser)
mux.HandleFunc("POST /users", createUser)
mux.HandleFunc("/files/{path...}", serveFiles) // Wildcard
\`\`\`

### Structured Logging (slog)
\`\`\`go
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
logger.Info("request",
    "method", r.Method,
    "path", r.URL.Path,
    "duration_ms", elapsed.Milliseconds(),
)
\`\`\`

### Loop Variable Fix (1.22)
Go 1.22 creates new loop variables per iteration. Manual capture (`item := item`) 
is no longer necessary but remains valid for backward compatibility.
\`\`\`
```

---

### CRIT-004: R Conventions Missing S7 and dplyr 1.1+
**Severity:** Critical
**Location:** `conventions/R.md`
**Issue:** Convention lacks modern R patterns:
- No S7 OOP system (the new R Consortium standard superseding S3/S4/R6)
- No dplyr 1.1+ features (`.by` argument, `reframe()`, `pick()`, `join_by()`)
- No `%||%` null coalescing operator (R 4.4)

**Impact:** Agents generate legacy OOP patterns and verbose grouped operations.

**Recommendation:** Add sections:
```markdown
## S7 OOP System (R 4.3+)

S7 is the new OOP standard from R Consortium, superseding S3/S4/R6 for new code:
\`\`\`r
library(S7)

Range <- new_class("Range",
  properties = list(
    start = class_double,
    end = class_double
  ),
  validator = function(self) {
    if (self@end < self@start) "@end must be >= @start"
  }
)

x <- Range(start = 1, end = 10)
x@start  # Property access with @
\`\`\`

## dplyr 1.1+ Features

### .by Argument (Preferred over group_by)
\`\`\`r
# PREFERRED
df |> summarise(total = sum(value), .by = category)

# LEGACY
df |> group_by(category) |> summarise(total = sum(value)) |> ungroup()
\`\`\`

### reframe() for Multi-Row Results
\`\`\`r
# summarise() now warns for multi-row results; use reframe()
df |> reframe(quantile_df(height), .by = species)
\`\`\`

### pick() for Column Selection in Mutate
\`\`\`r
df |> mutate(row_sum = rowSums(pick(starts_with("x"))))
\`\`\`
```

---

## Warnings

### WARN-001: React Sharp Edges Missing React 19 Pitfalls
**Severity:** High
**Location:** `agents/react-pro/sharp-edges.yaml`
**Issue:** Missing React 19-specific sharp edges:
- `use()` hook cannot be called in try/catch blocks
- `use()` requires stable promise references (not created during render)
- `useFormStatus` must be in child component of form
- Server Components cannot use hooks or event handlers

**Recommendation:** Add sharp edges:
```yaml
- id: use-hook-try-catch
  severity: critical
  category: hooks
  description: "use() hook cannot be called inside try/catch"
  symptom: "Runtime error when using use() in try/catch"
  solution: |
    WRONG:
    try {
      const data = use(promise);
    } catch (e) {}
    
    CORRECT: Use Error Boundary components instead

- id: use-hook-stable-promise
  severity: critical
  category: hooks
  description: "use() requires stable promise references"
  symptom: "Infinite loop when promise created during render"
  solution: |
    WRONG:
    function Component() {
      const data = use(fetch('/api')); // New promise each render!
    }
    
    CORRECT:
    const dataPromise = fetch('/api'); // Outside component or cached
    function Component() {
      const data = use(dataPromise);
    }

- id: useformstatus-child-component
  severity: high
  category: hooks
  description: "useFormStatus must be called in child of <form>"
  symptom: "pending is always false"
  solution: |
    WRONG:
    function Form() {
      const { pending } = useFormStatus(); // Won't work
      return <form>...</form>;
    }
    
    CORRECT:
    function SubmitButton() {
      const { pending } = useFormStatus(); // In child component
      return <button disabled={pending}>Submit</button>;
    }
```

---

### WARN-002: Agent Trigger Overlap
**Severity:** Medium
**Location:** `agents/agents-index.json`
**Issue:** Several triggers appear in multiple agents, creating routing ambiguity:

| Trigger | Agents |
|---------|--------|
| "implement" | python-pro, go-pro, typescript-pro |
| "refactor" | python-pro, go-pro, r-pro, typescript-pro |
| "test" | python-pro, r-pro |
| "review" | code-reviewer, backend-reviewer, frontend-reviewer, standards-reviewer |

**Impact:** Without file-type auto-activation, these triggers may route to wrong agent.

**Recommendation:** 
1. Ensure auto_activate patterns take precedence over generic triggers
2. Add language qualifiers to generic triggers or rely on file context
3. Document routing priority explicitly in CLAUDE.md

---

### WARN-003: Agents with >8 Triggers (Scope Creep Risk)
**Severity:** Medium
**Location:** `agents/agents-index.json`
**Issue:** Three agents exceed 8 triggers:
- `tech-docs-writer`: 11 triggers
- `r-pro`: 11 triggers  
- `python-ux`: 11 triggers
- `orchestrator`: 13 triggers

**Recommendation:** 
- Split `tech-docs-writer` into `readme-writer` and `api-docs-writer`
- Review if all `orchestrator` triggers are necessary vs delegating to sub-agents

---

### WARN-004: Version Mismatch Between Index and Schema
**Severity:** Low
**Location:** `agents/agents-index.json` (v2.3.0) vs `core/routing-schema.json` (v2.4.0)
**Issue:** Version numbers don't match, suggesting potential sync issues.

**Recommendation:** Unify version numbers and consider a single VERSION file.

---

### WARN-005: Missing PySide6 6.9+ Patterns
**Severity:** Medium
**Location:** `agents/python-ux/sharp-edges.yaml`, `conventions/python.md`
**Issue:** PySide6 6.9+ introduces `pyproject.toml` support (replacing deprecated `.pyproject`). Convention doesn't mention this.

**Recommendation:** Add note about pyproject.toml support in PySide6 6.9+.

---

## Recommendations

### REC-001: Add React Compiler Convention Section
**Priority:** High
**Rationale:** React Compiler fundamentally changes memoization patterns. Without guidance, agents may:
- Generate unnecessary useMemo/useCallback calls
- Not flag Rules of React violations (state mutation, impure render)

**Action:** Add comprehensive React Compiler section to react.md with examples of when memoization is/isn't needed.

---

### REC-002: Add TanStack Query to State Management Guidance
**Priority:** Medium
**Location:** `conventions/react.md`
**Rationale:** Current convention mentions Zustand but lacks TanStack Query for server state.

**Action:** Add state management decision tree:
```markdown
| State Type | Tool |
|------------|------|
| Server/async data | TanStack Query |
| Global client state | Zustand |
| Local component state | useState |
| Form state | React Hook Form or useActionState |
```

---

### REC-003: Add Ink v5/v6 Patterns
**Priority:** Medium
**Location:** `conventions/react.md`
**Rationale:** Ink TUI section exists but may need updates for v5/v6 patterns and React 18 concurrent features.

**Action:** Verify Ink version targeting and update patterns accordingly.

---

### REC-004: Add Go errgroup.SetLimit Pattern
**Priority:** Low
**Location:** `conventions/go.md`
**Rationale:** errgroup.SetLimit (Go 1.21) is missing from concurrency patterns.

**Action:** Add:
```go
g, ctx := errgroup.WithContext(ctx)
g.SetLimit(10) // Limit concurrent goroutines
```

---

### REC-005: Add Sharp Edge for Go 1.22 Loop Variable Change
**Priority:** Medium
**Location:** `agents/go-pro/sharp-edges.yaml`
**Rationale:** Go 1.22's loop variable fix is a behavioral change that could confuse developers expecting old behavior.

**Action:** Add sharp edge noting the change and when manual capture is still needed (pre-1.22 compatibility).

---

### REC-006: Standardize Convention Document Structure
**Priority:** Low
**Issue:** Conventions have varying structures. Some have frontmatter with paths, others don't.

**Action:** Create convention template with standardized sections:
1. Frontmatter (paths, version targeting)
2. System Constraints
3. Version-Specific Features
4. Core Patterns
5. Anti-Patterns
6. Testing
7. Configuration

---

## Agent-by-Agent Analysis

### Tier: Haiku (Cost: $0.0005/1k tokens)

| Agent | Triggers | Issues | Notes |
|-------|----------|--------|-------|
| codebase-search | 8 | None | Well-scoped read-only agent |
| haiku-scout | 8 | None | Good fallback for gemini-slave |

### Tier: Haiku Thinking (Cost: $0.001/1k tokens)

| Agent | Triggers | Issues | Notes |
|-------|----------|--------|-------|
| memory-archivist | 8 | None | Clear archival scope |
| scaffolder | 8 | None | Appropriate for boilerplate |
| librarian | 8 | None | Good research scope |
| tech-docs-writer | **11** | Scope creep | Consider splitting |
| code-reviewer | 6 | None | Well-scoped |
| backend-reviewer | 5 | None | Appropriate specialization |
| frontend-reviewer | 5 | None | Appropriate specialization |
| standards-reviewer | 5 | None | Appropriate specialization |

### Tier: Sonnet (Cost: $0.009/1k tokens)

| Agent | Triggers | Issues | Notes |
|-------|----------|--------|-------|
| python-pro | 9 | Generic triggers | Auto-activate mitigates |
| python-ux | **11** | Scope creep | Many Qt-specific triggers |
| r-pro | **11** | Scope creep | Many domain triggers |
| r-shiny-pro | 8 | None | Well-scoped |
| go-pro | 10 | Border case | Consider reducing |
| go-cli | 9 | None | Cobra-specific scope |
| go-tui | 8 | None | **Excellent sharp edges** |
| go-api | 9 | None | HTTP client scope |
| go-concurrent | 10 | Border case | Concurrency-specific |
| typescript-pro | 10 | Border case | Auto-activate mitigates |
| react-pro | **11** | Missing React 19 | **Critical convention gap** |
| orchestrator | **13** | Highest trigger count | Review necessity |
| review-orchestrator | 5 | None | Coordination agent |
| staff-architect-critical-review | 4 | None | Specialized review |

### Tier: Opus (Cost: $0.045/1k tokens)

| Agent | Triggers | Issues | Notes |
|-------|----------|--------|-------|
| architect | 10 | None | Justified by complexity |
| planner | 7 | None | Strategic scope |
| einstein | 4 | None | Appropriate escalation only |

### Tier: External (Gemini)

| Agent | Triggers | Issues | Notes |
|-------|----------|--------|-------|
| gemini-slave | 13 | High count OK | Large context specialist |

---

## Appendix: Suggested Changes

### Change 1: Update react.md React Version Target

**Before:**
```markdown
## System Constraints (CRITICAL)

**This system targets React 18+ with functional components and TypeScript.**
```

**After:**
```markdown
## System Constraints (CRITICAL)

**This system targets React 18+/19 with functional components and TypeScript.**

React 19 introduces:
- React Compiler (automatic memoization)
- Server Components and Server Actions
- New hooks: use(), useActionState, useOptimistic, useFormStatus
- Native metadata handling (<title>, <meta> in components)

See "React 19+ Features" section for migration patterns.
```

---

### Change 2: Update typescript.md Version Target

**Before:**
```markdown
## System Constraints (CRITICAL)

**This system targets TypeScript 5.0+ with strict mode ALWAYS enabled.**
```

**After:**
```markdown
## System Constraints (CRITICAL)

**This system targets TypeScript 5.5+ with strict mode ALWAYS enabled.**

TypeScript 5.5 key features:
- Inferred type predicates (.filter() now narrows correctly)
- Isolated declarations mode for faster builds
- NoInfer utility type (5.4)

See "TypeScript 5.5+ Features" section for patterns.
```

---

### Change 3: Add React 19 Sharp Edges

**File:** `agents/react-pro/sharp-edges.yaml`

**Add:**
```yaml
- id: server-component-hooks
  severity: critical
  category: react-19
  description: "Server Components cannot use hooks"
  symptom: "Build error or runtime crash when using useState/useEffect in Server Component"
  solution: |
    Server Components (no 'use client' directive) cannot use:
    - useState, useEffect, useRef, etc.
    - Event handlers (onClick, onChange, etc.)
    - Browser-only APIs
    
    Add 'use client' at top of file for interactive components.
  auto_inject: true

- id: react-compiler-rules
  severity: high
  category: react-19
  description: "React Compiler requires Rules of React compliance"
  symptom: "Compiler errors or incorrect optimization"
  solution: |
    When React Compiler is enabled, components MUST:
    - Be pure (same props = same output)
    - Never mutate props or state
    - Keep side effects outside render
    
    The compiler will skip components that violate these rules.
  auto_inject: true
```

---

### Change 4: Update go.md Loop Variable Section

**Before:**
```markdown
1. **Loop variable capture in goroutines**
   ```go
   // WRONG: All goroutines see same value
   for _, item := range items {
       go func() {
           process(item)  // BUG: item changes
       }()
   }
   
   // CORRECT: Capture variable
   for _, item := range items {
       item := item  // Capture
       go func() {
           process(item)
       }()
   }
   ```
```

**After:**
```markdown
1. **Loop variable capture in goroutines**
   
   **Go 1.22+ Fixed This!** Each iteration creates new variables automatically.
   
   ```go
   // Go 1.22+: Works correctly without manual capture
   for _, item := range items {
       go func() {
           process(item)  // Now works correctly
       }()
   }
   
   // Pre-1.22 or for explicit clarity: Manual capture
   for _, item := range items {
       item := item  // Still valid, required for Go <1.22
       go func() {
           process(item)
       }()
   }
   ```
   
   **Note:** Manual capture remains valid and may improve clarity. 
   Required when targeting Go <1.22 compatibility.
```

---

### Change 5: Add R S7 Section

**File:** `conventions/R.md`

**Add after OOP section:**
```markdown
## S7 OOP System (PREFERRED for New Code)

S7 is the R Consortium's new OOP system, designed to supersede S3/S4/R6.
**Use S7 for all new OOP code in R 4.3+.**

### Defining Classes

```r
library(S7)

# Simple class
Person <- new_class("Person",
  properties = list(
    name = class_character,
    age = class_integer
  )
)

# With validation
PositiveNumber <- new_class("PositiveNumber",
  properties = list(
    value = class_double
  ),
  validator = function(self) {
    if (self@value <= 0) "value must be positive"
  }
)

# With inheritance
Employee <- new_class("Employee",
  parent = Person,
  properties = list(
    employee_id = class_character,
    department = class_character
  )
)
```

### Property Access

```r
person <- Person(name = "Alice", age = 30L)
person@name        # "Alice"
person@age <- 31L  # Setter
```

### When to Use Which OOP System

| System | Use For |
|--------|---------|
| **S7** | All new OOP code (preferred) |
| S4 | Bioconductor packages requiring S4 |
| R6 | Reference semantics, mutable state |
| S3 | Simple method dispatch only |
```

---

## Post-Review Actions

1. **Immediate (Critical):**
   - [ ] Update react.md with React 19 features
   - [ ] Update typescript.md with TypeScript 5.5+ features
   - [ ] Update go.md with Go 1.22+ features
   - [ ] Update R.md with S7 and dplyr 1.1+
   - [ ] Add React 19 sharp edges to react-pro

2. **Short-term (Warnings):**
   - [ ] Review agent trigger overlap; add auto_activate priority documentation
   - [ ] Consider splitting tech-docs-writer
   - [ ] Sync version numbers between agents-index.json and routing-schema.json

3. **Long-term (Recommendations):**
   - [ ] Standardize convention document structure
   - [ ] Add TanStack Query to state management guidance
   - [ ] Update Ink patterns for v5/v6

---

*Review completed 2026-02-01. Framework demonstrates strong architectural foundations with targeted updates needed for 2024-2025 language ecosystem evolution.*
