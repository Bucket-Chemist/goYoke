# Parallel Tool Calling Templates - Orchestrator Implementation Guide

**Created by**: Einstein Analysis
**Date**: 2026-01-21
**Purpose**: Archetype-specific parallelization patterns for agent system enhancement
**Status**: Ready for orchestrator implementation

---

## Overview

This document provides 5 archetype-specific templates for parallel tool calling guidance. Each template is designed for a specific agent cognitive profile and includes:

- When to parallelize (and when NOT to)
- Concrete code examples
- Failure mode awareness
- Guardrail requirements

**CRITICAL**: Do NOT apply a single uniform template to all agents. Each archetype has different parallelization safety requirements.

---

## Implementation Sequence

### Phase 1: Low-Risk Archetypes (Week 1)

- Template A: Search Agents (`codebase-search`, `haiku-scout`)
- Template B: Scaffolding Agents (`scaffolder`)
- **Risk**: Minimal, **Benefit**: 5-10x speedup

### Phase 2: Medium-Risk Archetypes (Weeks 2-3)

- Template C: Analysis Agents (`code-reviewer`, `tech-docs-writer`)
- **Risk**: Medium, **Benefit**: 3-4x speedup

### Phase 3: High-Risk Archetypes (Weeks 4-7)

- Template D: Implementation Agents (`python-pro`, `go-pro`, `r-pro`, `r-shiny-pro`, `python-ux`, `go-cli`)
- **Risk**: High, **Benefit**: 3-10x speedup
- **Requires**: Dependency graph tracking, layer enforcement

### Phase 4: Constrained Archetypes (Week 8)

- Template E: Orchestration Agents (`orchestrator`, `architect`)
- Template F: Deep Reasoning Agents (`einstein`)
- **Risk**: Low, **Benefit**: Maintain correctness

---

# Template A: Mandatory Parallelization (Search Agents)

## Target Agents

- `codebase-search`
- `librarian` (already has guidance - validate consistency)
- `haiku-scout`

## Cognitive Profile

- **Model**: Haiku (0-2K thinking budget)
- **Operation Type**: Read-only searches
- **Parallelism Tolerance**: Very high (no state mutation)
- **Failure Recovery**: Simple (retry failed read)

## Template Content

```markdown
---

## PARALLELIZATION: MANDATORY

**All read operations MUST be batched in a single message.** Sequential reads are a performance violation.

### Performance Impact

Sequential (OLD):
```

Message 1: Read file1 (5s)
Message 2: Read file2 (5s)
Message 3: Read file3 (5s)
Total: 15 seconds

```

Parallel (REQUIRED):
```

Message 1: Read file1, Read file2, Read file3 (5s)
Total: 5 seconds

````

**Result: 3x faster**

---

### Correct Pattern (MANDATORY)

✅ **ALWAYS do this:**
```python
# Batch ALL reads in one message
Read(src/auth.py)
Read(src/models/user.py)
Read(tests/test_auth.py)
Read(config/settings.py)

# Batch ALL searches in one message
Grep("class.*User", glob="*.py")
Grep("def.*authenticate", glob="*.py")
Grep("import.*jwt", glob="*.py")
````

❌ **NEVER do this:**

```python
# Sequential reads (FORBIDDEN)
Read(src/auth.py)
# ... analyze ...
Read(src/models/user.py)  # VIOLATION: Should have batched with first read
```

---

### When to Parallelize

**Always.** Search agents have no valid reason for sequential reads.

**Exception**: None. If you think you need sequential reads, you are misunderstanding the task.

---

### Failure Handling

If any read in a batch fails:

1. **Retry the entire batch** (reads are idempotent)
2. If retry fails, **retry the failed read individually** to isolate error
3. Report which specific file/pattern failed

**Example**:

```
Batch: Read(file1), Read(file2), Read(file3)
Result: file2 failed with "permission denied"

Action:
1. Report: "Failed to read file2: permission denied"
2. Continue with file1 and file3 results
3. Suggest: "Check file permissions on file2"
```

---

### Guardrails

**Tool Layer Enforcement** (to be implemented):

- Detect sequential Read/Grep calls in search agents
- Auto-batch if agent forgets
- Flag performance violations in agent logs

**Agent Self-Check**:
Before sending message, verify:

- [ ] All reads are in ONE message (not spread across multiple)
- [ ] No sequential read calls
- [ ] Batch size is reasonable (<50 files to avoid timeout)

---

### Examples by Use Case

**Finding all authentication-related files:**

```python
# CORRECT: Parallel search
Grep("auth", glob="*.py", output_mode="files_with_matches")
Grep("login", glob="*.py", output_mode="files_with_matches")
Grep("session", glob="*.py", output_mode="files_with_matches")

# Then read all found files in parallel
Read(result1_file1)
Read(result1_file2)
Read(result2_file1)
# ... etc
```

**Understanding a module's structure:**

```python
# CORRECT: Read all module files at once
Glob("src/auth/*.py")  # Get file list
# Then read all:
Read(src/auth/__init__.py)
Read(src/auth/models.py)
Read(src/auth/views.py)
Read(src/auth/utils.py)
```

**Finding usage patterns:**

```python
# CORRECT: Parallel pattern searches
Grep("import requests", glob="*.py", output_mode="content", -n=True)
Grep("requests.get", glob="*.py", output_mode="content", -n=True)
Grep("requests.post", glob="*.py", output_mode="content", -n=True)
```

---

### Anti-Patterns to Avoid

❌ **Sequential exploration:**

```python
# Read one file
Read(file1)
# Decide what to read next based on file1
Read(file2)  # WRONG: Should have read both initially
```

**Fix**: Read ALL potentially relevant files upfront. Over-reading is better than under-reading for search agents.

❌ **Conditional reading:**

```python
# Read main file
Read(main.py)
# If main.py mentions tests, read tests
if "test" in main_content:
    Read(tests/test_main.py)  # WRONG: Should have read both initially
```

**Fix**: Read both files in parallel initially. Conditional logic comes AFTER reading, not before.

---

````

---

# Template B: Directory-Aware Parallelization (Scaffolding Agents)

## Target Agents
- `scaffolder`

## Cognitive Profile
- **Model**: Haiku+ (2-6K thinking budget)
- **Operation Type**: Batch file creation
- **Parallelism Tolerance**: High (creates new files, minimal conflicts)
- **Failure Recovery**: Simple (retry failed creation)

## Template Content

```markdown
---

## PARALLELIZATION: DIRECTORY-AWARE

**File creation operations MUST respect directory hierarchy.** Create parent directories before children.

### Performance Impact

Sequential (OLD):
````

Write **init**.py (5s)
Write core/**init**.py (5s)
Write core/models.py (5s)
Write tests/**init**.py (5s)
Write tests/test_core.py (5s)
Total: 25 seconds

```

Directory-Layered (REQUIRED):
```

Layer 0: Write **init**.py (5s)
Layer 1: Write core/**init**.py, Write tests/**init**.py (5s)
Layer 2: Write core/models.py, Write tests/test_core.py (5s)
Total: 15 seconds

````

**Result: 1.7x faster + guaranteed correctness**

---

### Correct Pattern (MANDATORY)

✅ **Directory depth layering:**
```python
# Layer 0: Root package files (depth 0)
Write(my_package/__init__.py, content=root_init)
Write(my_package/version.py, content=version_file)

# [WAIT for Layer 0 to complete]

# Layer 1: Subdirectory package files (depth 1)
Write(my_package/core/__init__.py, content=core_init)
Write(my_package/utils/__init__.py, content=utils_init)
Write(my_package/tests/__init__.py, content=tests_init)

# [WAIT for Layer 1 to complete]

# Layer 2: Module files (depth 2)
Write(my_package/core/models.py, content=models)
Write(my_package/core/api.py, content=api)
Write(my_package/utils/helpers.py, content=helpers)
Write(my_package/tests/test_core.py, content=tests)
````

❌ **NEVER mix depths in same batch:**

```python
# WRONG: Mixing depth 0 and depth 2
Write(my_package/__init__.py, content=root)
Write(my_package/core/models.py, content=models)  # VIOLATION: Child before parent
```

---

### Depth Calculation

**Depth = number of slashes in path after base directory**

Examples:

```
my_package/__init__.py          → Depth 0 (base)
my_package/core/__init__.py     → Depth 1 (one subdir)
my_package/core/models.py       → Depth 2 (subdir + file)
my_package/core/sub/utils.py    → Depth 3 (two subdirs + file)
```

**Rule**: Depth N operations MUST complete before Depth N+1 operations start.

---

### Automatic Layering (Tool Layer Feature)

**You don't need to calculate depths manually.** The tool layer will:

1. Calculate depth for each Write() call
2. Group calls by depth
3. Execute groups in depth order
4. Wait for each group to complete before starting next

**Your job**: Declare all Write() operations. Tool layer handles sequencing.

Example:

```python
# You write:
Write(pkg/__init__.py)
Write(pkg/core/models.py)      # Depth 2
Write(pkg/utils/__init__.py)   # Depth 1
Write(pkg/core/__init__.py)    # Depth 1

# Tool layer reorders:
# Batch 0 (depth 0): pkg/__init__.py
# Batch 1 (depth 1): pkg/core/__init__.py, pkg/utils/__init__.py
# Batch 2 (depth 2): pkg/core/models.py
```

---

### Special Case: **init**.py Files

`__init__.py` files are special in Python:

- **Must exist before other files in same directory**
- **Always depth N for directory at depth N**

**Rule**: If creating directory `foo/bar/`, create `foo/bar/__init__.py` BEFORE creating `foo/bar/module.py`.

Tool layer enforces this automatically.

---

### Failure Handling

If a write fails in Layer N:

1. **Rollback entire Layer N** (remove any files created in this layer)
2. **Stop execution** (do not proceed to Layer N+1)
3. **Report failure** with context of which layer failed

**Example**:

```
Layer 1 batch:
- Write pkg/core/__init__.py → SUCCESS
- Write pkg/utils/__init__.py → FAIL (permission denied)

Action:
1. Delete pkg/core/__init__.py (rollback layer)
2. Stop (do not create Layer 2 files)
3. Report: "Layer 1 failed: pkg/utils/__init__.py permission denied"
```

**Rationale**: Partial directory structures are worse than no directory structure. All-or-nothing per layer.

---

### Examples by Use Case

**Python package scaffold:**

```python
# Declare all files (tool layer will layer them)
Write(myapp/__init__.py, content=root_init)
Write(myapp/core/__init__.py, content=core_init)
Write(myapp/core/models.py, content=models)
Write(myapp/core/api.py, content=api)
Write(myapp/tests/__init__.py, content=test_init)
Write(myapp/tests/test_models.py, content=tests)
Write(myapp/config.py, content=config)

# Tool layer executes as:
# Layer 0: myapp/__init__.py, myapp/config.py
# Layer 1: myapp/core/__init__.py, myapp/tests/__init__.py
# Layer 2: myapp/core/models.py, myapp/core/api.py, myapp/tests/test_models.py
```

**Nested directory structure:**

```python
# Complex nesting
Write(app/__init__.py)
Write(app/api/__init__.py)
Write(app/api/v1/__init__.py)
Write(app/api/v1/endpoints.py)
Write(app/api/v2/__init__.py)
Write(app/api/v2/endpoints.py)

# Tool layer executes as:
# Layer 0: app/__init__.py
# Layer 1: app/api/__init__.py
# Layer 2: app/api/v1/__init__.py, app/api/v2/__init__.py
# Layer 3: app/api/v1/endpoints.py, app/api/v2/endpoints.py
```

**Multiple root files:**

```python
# Same depth, parallel OK
Write(setup.py, content=setup)
Write(README.md, content=readme)
Write(requirements.txt, content=reqs)
Write(.gitignore, content=gitignore)

# All depth 0, execute in parallel
```

---

### Anti-Patterns to Avoid

❌ **Creating child before parent:**

```python
Write(app/core/models.py)  # WRONG: app/core/ doesn't exist yet
Write(app/core/__init__.py)
```

**Fix**: Declare `__init__.py` before other files in directory.

❌ **Manual sequencing instead of declarative:**

```python
# Create parent
Write(app/__init__.py)
# Wait...
# Create child
Write(app/core/__init__.py)
```

**Fix**: Declare all writes at once, let tool layer sequence.

---

````

---

# Template C: Tiered Parallelization (Analysis Agents)

## Target Agents
- `code-reviewer`
- `tech-docs-writer`

## Cognitive Profile
- **Model**: Haiku+ to Sonnet+ (4-16K thinking budget)
- **Operation Type**: Multi-source reading for synthesis
- **Parallelism Tolerance**: High (mostly reads)
- **Failure Recovery**: Medium (can continue with partial context)

## Template Content

```markdown
---

## PARALLELIZATION: TIERED

**Read operations fall into two tiers: CRITICAL and OPTIONAL.** Critical reads must succeed; optional reads enable better analysis but aren't required.

### Performance Impact

Sequential (OLD):
````

Read critical1 (5s)
Read critical2 (5s)
Read optional1 (5s)
Read optional2 (5s)
Total: 20 seconds

```

Tiered Parallel (REQUIRED):
```

Read critical1, critical2, optional1, optional2 (5s)
Handle failures gracefully
Total: 5 seconds

````

**Result: 4x faster + graceful degradation**

---

### Correct Pattern (MANDATORY)

✅ **Tag reads by priority, batch together:**
```python
# All reads in one message, tagged by priority
Read(src/auth.py, priority="critical")       # Must have for review
Read(tests/test_auth.py, priority="optional") # Nice to have for context
Read(docs/auth.md, priority="optional")       # Helpful but not required

# If critical read fails → Abort with error
# If optional read fails → Log caveat and continue
````

❌ **Sequential reads without priority:**

```python
Read(src/auth.py)
# ... analyze ...
Read(tests/test_auth.py)  # WRONG: Should have batched, should have tagged priority
```

---

### Priority Levels

**CRITICAL** (must succeed):

- Primary source code being analyzed
- Configuration files that affect understanding
- Schema/model definitions
- Files explicitly requested by user

**OPTIONAL** (nice to have):

- Test files (for context)
- Documentation (for comparison)
- Related but not directly analyzed files
- Historical context (git history, old versions)

**Rule of thumb**: If analysis cannot proceed without this file, it's CRITICAL. Otherwise, OPTIONAL.

---

### Failure Handling by Priority

**CRITICAL read fails:**

```
Action:
1. ABORT analysis
2. Report: "Cannot analyze src/auth.py: file not found"
3. Suggest: "Check file path or restore from backup"
4. Do NOT attempt partial analysis
```

**OPTIONAL read fails:**

```
Action:
1. CONTINUE analysis with available files
2. Log caveat: "Analysis based on src/auth.py only. Tests unavailable for comparison."
3. Note limitations in output
4. Suggest: "Review would be more thorough with test coverage"
```

---

### Output Caveats

When optional reads fail, include caveats in your analysis:

**Example: Code Review Output**

```markdown
## Code Review: src/auth.py

**Context Limitations**: Test file tests/test_auth.py not available. Review focuses on code structure only, cannot verify test coverage.

### Findings

1. Authentication logic in `verify_password()` uses bcrypt correctly ✓
2. Session management in `create_session()` uses secure random tokens ✓
3. **CAVEAT**: Cannot verify if these functions are tested (test file unavailable)

### Recommendations

- Consider adding integration tests for authentication flow
- **Note**: Recommendation based on code review only. Existing tests may already cover this.
```

**Pattern**: Be explicit about what you DIDN'T analyze due to missing optional context.

---

### Examples by Use Case

**Code review with full context:**

```python
# Batch all sources
Read(src/api.py, priority="critical")          # Primary review target
Read(tests/test_api.py, priority="optional")   # Test coverage context
Read(docs/api.md, priority="optional")         # Documentation comparison
Read(src/models.py, priority="optional")       # Dependency context

# If all succeed: Comprehensive review
# If tests missing: Review code + note lack of test verification
# If docs missing: Review code + note documentation may be outdated
```

**Documentation update:**

```python
# Batch all sources
Read(src/endpoints.py, priority="critical")    # Source of truth for API
Read(docs/api-reference.md, priority="critical") # Document being updated
Read(tests/test_endpoints.py, priority="optional") # Usage examples
Read(CHANGELOG.md, priority="optional")        # Historical context

# If critical files missing: Cannot proceed
# If optional files missing: Update docs based on source code only
```

**Multi-file analysis:**

```python
# Reviewing authentication system across multiple files
Read(src/auth/login.py, priority="critical")
Read(src/auth/session.py, priority="critical")
Read(src/auth/permissions.py, priority="critical")
Read(tests/test_auth.py, priority="optional")
Read(docs/security.md, priority="optional")

# If any critical file missing: Report which file is missing, abort
# If optional files missing: Proceed with code-only review
```

---

### Anti-Patterns to Avoid

❌ **Treating all reads as critical:**

```python
Read(src/auth.py, priority="critical")
Read(tests/test_auth.py, priority="critical")  # WRONG: Tests are optional context
```

**Fix**: Reserve "critical" for must-have files only. Most context files are optional.

❌ **Not including caveats for missing optional files:**

```
# Output says "Code review complete"
# But tests were missing and not mentioned
```

**Fix**: Always note what context was missing.

❌ **Aborting on optional failure:**

```python
# Test file missing
if test_file_failed:
    abort_analysis()  # WRONG: Should continue with caveat
```

**Fix**: Continue analysis, add caveat to output.

---

````

---

# Template D: Layer-Based Parallelization (Implementation Agents)

## Target Agents
- `python-pro`
- `go-pro`
- `r-pro`
- `r-shiny-pro`
- `python-ux`
- `go-cli`
- `go-tui`
- `go-api`
- `go-concurrent`

## Cognitive Profile
- **Model**: Sonnet+ (10-16K thinking budget)
- **Operation Type**: Multi-file implementation with dependencies
- **Parallelism Tolerance**: Medium (writes can conflict)
- **Failure Recovery**: Complex (partial state is dangerous)

## Template Content

```markdown
---

## PARALLELIZATION: LAYER-BASED

**Write operations MUST respect dependency layers.** Files that depend on other files must be written in separate layers.

### Performance Impact

Sequential (OLD):
````

Write models.py (5s)
Write api.py (5s) # Depends on models.py
Write tests.py (5s) # Depends on both
Total: 15 seconds

```

Layer-Based (REQUIRED):
```

Layer 0: Write models.py (5s)
Layer 1: Write api.py (5s)
Layer 2: Write tests.py (5s)
Total: 15 seconds (BUT CORRECT + safer for parallel within layers)

```

**For independent files:**
```

Layer 0: Write models.py, Write utils.py, Write config.py (5s)
Total: 5 seconds (3x faster than sequential)

````

---

### Dependency Declaration

**Before writing any files, declare dependencies:**

```python
# Dependency graph declaration
dependencies = {
    "models.py": [],                    # No dependencies
    "utils.py": [],                     # No dependencies
    "api.py": ["models.py"],           # Depends on models
    "tests/test_api.py": ["api.py", "models.py"],  # Depends on both
    "tests/test_models.py": ["models.py"]          # Depends on models
}

# Tool layer calculates layers:
# Layer 0: models.py, utils.py (no dependencies)
# Layer 1: api.py (depends on layer 0)
# Layer 2: tests/test_api.py, tests/test_models.py (depend on layer 0-1)
````

---

### Writing by Layer

**Layer 0: Foundation files (no dependencies)**

```python
Write(models.py, content=models_code)
Write(utils.py, content=utils_code)
Write(constants.py, content=constants)
```

**Layer 1: Files that depend on Layer 0**

```python
Write(api.py, content=api_code)  # Imports from models.py
Write(services.py, content=services)  # Imports from utils.py
```

**Layer 2: Files that depend on Layer 1**

```python
Write(tests/test_api.py, content=api_tests)  # Imports api.py
Write(integration/app.py, content=app)  # Imports api.py, services.py
```

**Rule**: Layer N writes execute in parallel. Layer N+1 waits for Layer N to complete.

---

### Automatic Layer Calculation

**You don't need to manually order writes.** Declare dependencies, tool layer sequences automatically.

```python
# You declare:
dependencies = {
    "c.py": ["a.py", "b.py"],  # c depends on a and b
    "b.py": ["a.py"],          # b depends on a
    "a.py": []                 # a has no deps
}

# Tool layer calculates:
# Layer 0: a.py
# Layer 1: b.py (depends on layer 0)
# Layer 2: c.py (depends on layer 1)
```

**Topological sort** handles complex dependency graphs automatically.

---

### Dependency Detection Strategies

**Strategy 1: Import Analysis (Recommended)**

```python
# Read files first to detect imports
Read(existing_models.py)
# Tool layer extracts: "from models import User, Post"

# Automatically build dependency graph:
dependencies = detect_imports([existing_models.py, new_api.py])
```

**Strategy 2: Manual Declaration**

```python
# You know the dependencies from design
dependencies = {
    "api.py": ["models.py"],
    "tests.py": ["api.py", "models.py"]
}
```

**Strategy 3: Conservative Layering**

```python
# If unsure, layer by file type:
# Layer 0: Models
# Layer 1: Business logic
# Layer 2: API endpoints
# Layer 3: Tests
```

---

### Failure Handling

**If any write in Layer N fails:**

1. **Rollback entire Layer N** (delete files created in this layer)
2. **Stop execution** (do not proceed to Layer N+1)
3. **Report failure** with dependency context

**Example:**

```
Layer 1 writes:
- Write api.py (depends on models.py) → SUCCESS
- Write services.py (depends on utils.py) → FAIL (import error)

Action:
1. Delete api.py (rollback Layer 1)
2. Stop (Layer 2 depends on Layer 1, cannot proceed)
3. Report: "Layer 1 failed: services.py import error. Missing utils.py dependency?"
```

**Rationale**: Partial implementation is worse than no implementation. All-or-nothing per layer.

---

### Special Case: Test Files

**Tests often depend on source files but not vice versa.**

**Pattern**: Write tests in final layer, parallelize all test files together.

```python
dependencies = {
    # Source files (layers 0-2)
    "models.py": [],
    "api.py": ["models.py"],

    # Tests (layer 3, all parallel)
    "tests/test_models.py": ["models.py"],
    "tests/test_api.py": ["api.py", "models.py"],
    "tests/fixtures.py": ["models.py"]
}

# Layer 0: models.py
# Layer 1: api.py
# Layer 2: (empty)
# Layer 3: All three test files in parallel
```

---

### Examples by Use Case

**Creating a new API feature:**

```python
# Phase 1: Declare dependencies
dependencies = {
    "models/user.py": [],                           # Foundation
    "api/endpoints.py": ["models/user.py"],        # Uses models
    "tests/test_endpoints.py": ["api/endpoints.py", "models/user.py"],
    "tests/fixtures.py": ["models/user.py"]
}

# Phase 2: Write by layers (tool layer auto-sequences)
Write(models/user.py, content=user_model)
Write(api/endpoints.py, content=endpoints)
Write(tests/test_endpoints.py, content=tests)
Write(tests/fixtures.py, content=fixtures)

# Execution:
# Layer 0: models/user.py
# Layer 1: api/endpoints.py
# Layer 2: tests/test_endpoints.py, tests/fixtures.py (parallel)
```

**Refactoring with dependencies:**

```python
# Refactor: Split models.py into user.py and post.py
dependencies = {
    "models/user.py": [],          # New file, no deps
    "models/post.py": [],          # New file, no deps
    "api/users.py": ["models/user.py"],   # Update to use new model
    "api/posts.py": ["models/post.py"],   # Update to use new model
}

# Edit old files that import models:
Edit(api/old_endpoints.py, old="from models import", new="from models.user import")

# Execution:
# Layer 0: models/user.py, models/post.py (parallel)
# Layer 1: api/users.py, api/posts.py (parallel), Edit api/old_endpoints.py
```

**Complex multi-package implementation:**

```python
dependencies = {
    # Package A
    "pkg_a/__init__.py": [],
    "pkg_a/core.py": [],

    # Package B (depends on A)
    "pkg_b/__init__.py": [],
    "pkg_b/utils.py": ["pkg_a/core.py"],

    # Package C (depends on A and B)
    "pkg_c/__init__.py": [],
    "pkg_c/api.py": ["pkg_a/core.py", "pkg_b/utils.py"],

    # Tests (depend on all)
    "tests/test_integration.py": ["pkg_a/core.py", "pkg_b/utils.py", "pkg_c/api.py"]
}

# Tool layer calculates 4 layers, executes accordingly
```

---

### Anti-Patterns to Avoid

❌ **Writing dependent files in parallel:**

```python
# WRONG: api.py depends on models.py
Write(models.py, content=models)
Write(api.py, content=api)  # Imports models.py - race condition!
```

**Fix**: Declare dependency, let tool layer sequence.

❌ **Not declaring dependencies:**

```python
# Assuming tool layer will "figure it out"
Write(many_files_without_declaring_dependencies)
```

**Fix**: Always declare dependencies explicitly or use import analysis.

❌ **Overly conservative sequencing:**

```python
# Making everything sequential "to be safe"
Write(file1)
# wait
Write(file2)  # Actually independent of file1, could be parallel
```

**Fix**: Declare actual dependencies only. Independent files parallelize automatically.

---

````

---

# Template E: Constrained Parallelization (Orchestration Agents)

## Target Agents
- `orchestrator`
- `architect`

## Cognitive Profile
- **Model**: Sonnet+ (12-16K thinking budget)
- **Operation Type**: Coordination and planning
- **Parallelism Tolerance**: Low for agent spawning, high for reads
- **Failure Recovery**: Complex (coordination failures cascade)

## Template Content

```markdown
---

## PARALLELIZATION: CONSTRAINED

**Read operations: Parallelize freely. Agent spawning: Sequential only.** Coordination requires single-threaded decision making.

### Performance Impact

**Information Gathering (GAINS FROM PARALLELIZATION):**

Sequential (OLD):
````

Read file1 (5s)
Read file2 (5s)
Grep pattern1 (5s)
Grep pattern2 (5s)
Total: 20 seconds

```

Parallel (REQUIRED):
```

Read file1, file2, Grep pattern1, pattern2 (5s)
Total: 5 seconds

```

**Result: 4x faster**

**Agent Coordination (NO PARALLELIZATION):**

```

# MUST remain sequential

Task(agent1) # Completes, returns result

# Analyze result

Task(agent2) # Based on agent1's output

````

**Parallel spawning is FORBIDDEN** (creates conflicting instructions)

---

### Correct Pattern: Information Gathering

✅ **Parallelize all reads for decision making:**
```python
# Gather context in parallel
Read(src/api.py)
Read(src/models.py)
Read(tests/test_api.py)
Grep("TODO", glob="*.py", output_mode="files_with_matches")
Grep("FIXME", glob="*.py", output_mode="content")

# Analyze gathered information
# Make decision: Which agent to spawn?
````

❌ **Sequential information gathering:**

```python
Read(src/api.py)
# Analyze
Read(src/models.py)  # WRONG: Should have batched with first read
```

---

### Correct Pattern: Agent Coordination

✅ **Sequential agent spawning:**

```python
# Spawn one agent
result1 = Task(codebase-search, prompt="Find authentication files")

# Analyze result1
# Make next decision based on result1

# Spawn next agent
result2 = Task(python-pro, prompt="Implement auth based on: {result1}")
```

❌ **FORBIDDEN: Speculative parallel spawning:**

```python
# DO NOT DO THIS:
Task(codebase-search, prompt="Find Python files")  # Spawn
Task(codebase-search, prompt="Find Go files")      # Spawn in parallel

# WRONG: Creates two parallel searches, wastes resources
```

❌ **FORBIDDEN: Parallel agent decisions:**

```python
# DO NOT DO THIS:
Task(python-pro, prompt="Implement approach A")
Task(python-pro, prompt="Implement approach B")

# WRONG: Creates conflicting implementations
```

---

### When to Parallelize

**✅ DO parallelize:**

- Reading multiple files for context
- Running multiple searches for information
- Gathering data from multiple sources
- Reading documentation + code simultaneously

**❌ DO NOT parallelize:**

- Spawning multiple agents
- Making multiple Task() calls
- Speculative execution (spawning agents "just in case")
- Parallel decision paths

---

### Why Agent Spawning Must Be Sequential

**Problem**: Multiple agents receiving instructions in parallel create coordination conflicts.

**Example of conflict:**

```python
# Parallel spawn (BAD)
Task(python-pro, "Refactor auth.py to use JWT")
Task(python-pro, "Refactor auth.py to use sessions")

# Result:
# - Both agents edit auth.py
# - Conflicting implementations
# - File is corrupted
```

**Correct approach:**

```python
# Decision first
chosen_approach = analyze_requirements()

# Single spawn
Task(python-pro, f"Refactor auth.py to use {chosen_approach}")
```

---

### Decision-Making Pattern

**Phase 1: Information Gathering (PARALLEL)**

```python
# Gather all needed information at once
Read(file1), Read(file2), Read(file3)
Grep(pattern1), Grep(pattern2)
```

**Phase 2: Analysis (SEQUENTIAL, in your thinking)**

```
Analyze gathered information:
- What type of task is this?
- Which agent is appropriate?
- What are the requirements?
```

**Phase 3: Execution (SEQUENTIAL)**

```python
# Spawn ONE agent based on analysis
Task(chosen_agent, prompt=constructed_prompt)
```

**Phase 4: Iteration (SEQUENTIAL)**

```python
# Analyze agent result
# If more work needed, spawn NEXT agent
Task(next_agent, prompt=next_prompt)
```

---

### Examples by Use Case

**Code exploration and implementation:**

```python
# Phase 1: Parallel information gathering
Read(src/auth.py)
Read(src/models/user.py)
Read(tests/test_auth.py)
Grep("authentication", glob="*.py", output_mode="files_with_matches")

# Phase 2: Sequential analysis
# (in thinking: What needs to be done? → Refactor authentication)

# Phase 3: Sequential execution
Task(python-pro, prompt="Refactor authentication to use JWT based on files: src/auth.py, src/models/user.py")
```

**Documentation review and update:**

```python
# Phase 1: Parallel information gathering
Read(docs/api.md)
Read(src/api/endpoints.py)
Read(tests/test_endpoints.py)

# Phase 2: Sequential analysis
# (in thinking: Docs are outdated, need update)

# Phase 3: Sequential execution
Task(tech-docs-writer, prompt="Update docs/api.md based on current implementation in src/api/endpoints.py")
```

**Multi-stage workflow:**

```python
# Stage 1: Find files
files = Task(codebase-search, "Find all authentication-related files")

# Stage 2: Review files (based on Stage 1 result)
review = Task(code-reviewer, f"Review these files for security: {files}")

# Stage 3: Implement fixes (based on Stage 2 result)
Task(python-pro, f"Fix security issues identified: {review}")

# Sequential chain, each stage depends on previous
```

---

### Anti-Patterns to Avoid

❌ **Speculative agent spawning:**

```python
# "I'll spawn both and pick the better result"
Task(python-pro, "Approach A")
Task(python-pro, "Approach B")  # WRONG: Wastes resources, creates conflicts
```

**Fix**: Make decision FIRST, then spawn ONE agent.

❌ **Sequential information gathering:**

```python
Read(file1)
# Decide what to read next
Read(file2)  # WRONG: Should have read both initially
```

**Fix**: Read all potentially relevant files in parallel, THEN decide.

❌ **Mixing information gathering with spawning:**

```python
Read(file1)
Task(agent1)  # WRONG: Might need file2 for agent1's context
Read(file2)
```

**Fix**: Gather ALL information first, THEN spawn agents with complete context.

---

````

---

# Template F: No Parallelization (Deep Reasoning Agents)

## Target Agents
- `einstein`

## Cognitive Profile
- **Model**: Opus+ (16-32K thinking budget)
- **Operation Type**: Deep integrative reasoning
- **Parallelism Tolerance**: Zero (reasoning is inherently sequential)
- **Failure Recovery**: N/A (doesn't fail, goes deeper)

## Template Content

```markdown
---

## PARALLELIZATION: FORBIDDEN

**All operations must be sequential.** Deep reasoning requires building integrated understanding step-by-step.

### Why Parallelization Is Harmful for Deep Reasoning

**Parallel reads fragment context:**
````

Read(source1), Read(source2), Read(source3)
→ Receive three isolated pieces of information
→ Must reconstruct relationships retroactively
→ Lose opportunity for integrative thinking during reading

```

**Sequential reads enable integration:**
```

Read(source1)
→ Think: What does this mean? What are the implications?

Read(source2)
→ Think: How does this relate to source1? What's the connection?

Read(source3)
→ Think: How do all three fit together? What's the synthesis?

````

**Result**: Sequential reading produces DEEPER understanding, not just faster reading.

---

### Correct Pattern (MANDATORY)

✅ **Sequential reading with thinking checkpoints:**
```python
# Read first source
Read(source1)

[THINK: Understand source1 deeply]
- What is the core concept?
- What are the implications?
- What questions does this raise?

# Read second source
Read(source2)

[THINK: Integrate with source1]
- How does this relate to what I learned from source1?
- Does this confirm, contradict, or extend source1?
- What new connections emerge?

# Read third source
Read(source3)

[THINK: Synthesize all three]
- What is the overarching pattern?
- How do all pieces fit together?
- What insights emerge from the whole that weren't visible in parts?
````

❌ **FORBIDDEN: Parallel reading:**

```python
Read(source1), Read(source2), Read(source3)
# WRONG: Receives isolated fragments, misses integration opportunities
```

---

### Reading Strategy by Complexity

**Simple problem (2-3 sources):**

```python
Read(source1)
[Think: Extract key concepts]

Read(source2)
[Think: Compare and contrast with source1]

Read(source3)
[Think: Synthesize final understanding]
```

**Complex problem (5+ sources):**

```python
Read(primary_source)
[Think: Establish foundation]

Read(supporting_source1)
[Think: How does this elaborate on the foundation?]

Read(supporting_source2)
[Think: How does this provide a different perspective?]

Read(contradictory_source)
[Think: Where does this conflict? Why? Which is more credible?]

Read(synthesis_source)
[Think: How does this help resolve conflicts? What's the bigger picture?]
```

---

### Integration Thinking Prompts

After each read, ask yourself:

**For source N=1 (first source):**

- What is the core claim or concept?
- What evidence supports it?
- What assumptions underlie it?
- What questions remain unanswered?

**For source N>1 (subsequent sources):**

- How does this relate to previous sources?
- Does it confirm, refute, or qualify previous understanding?
- What new information does it add?
- What contradictions or tensions emerge?
- How do I reconcile different perspectives?

**For final source (synthesis):**

- What is the synthesized understanding across all sources?
- What patterns emerged that weren't visible in individual sources?
- What remains uncertain or debated?
- What are the practical implications of this synthesis?

---

### Why This Differs from Other Agents

**Other agents**: Optimize for SPEED (read 10 files in 5 seconds)

**Einstein**: Optimize for DEPTH (read 3 files with full integration in 30 seconds)

**Tradeoff**: Einstein takes longer, but produces insights other agents cannot.

**Example**:

**Fast agent** (parallel reads):

- Reads 10 papers in 5 seconds
- Summarizes each independently
- Lists findings

**Einstein** (sequential + integration):

- Reads 3 papers in 30 seconds
- Identifies contradiction between paper 1 and paper 2
- Recognizes paper 3 resolves contradiction
- Synthesizes novel insight not stated in any paper

**Result**: Einstein produces QUALITATIVELY different output, not just slower output.

---

### When to Use Einstein

**Use Einstein when:**

- Problem has been escalated (other agents failed)
- Deep analysis needed (not just information gathering)
- Multiple perspectives need synthesis
- Contradictions need resolution
- Novel insights required

**Do NOT use Einstein when:**

- Simple information lookup
- Straightforward implementation
- Speed is more important than depth
- Problem doesn't require synthesis

---

### Examples by Use Case

**Analyzing architectural tradeoffs:**

```python
Read(architecture_proposal_A.md)
[Think: What are the strengths and weaknesses of approach A?]

Read(architecture_proposal_B.md)
[Think: How does B differ from A? Where do they conflict?]

Read(system_constraints.md)
[Think: Given constraints, which approach is more viable? Why?]

Read(team_experience.md)
[Think: Given team skills, which approach is more feasible?]

Synthesize:
- Approach A is theoretically superior but requires expertise we lack
- Approach B is pragmatic given constraints
- Hybrid approach possible: Start with B, migrate to A over time
```

**Debugging complex system failure:**

```python
Read(error_logs.txt)
[Think: What is the symptom? When did it start?]

Read(recent_changes.diff)
[Think: What changed that could cause this?]

Read(system_architecture.md)
[Think: How do components interact? Where could failure propagate?]

Read(similar_incident_postmortem.md)
[Think: Is this the same root cause? Different? Related?]

Synthesize:
- Error manifests in component C
- Root cause is in component A (2 layers removed)
- Recent change in A introduced race condition
- Race propagates to C under specific load pattern
- Similar to incident from 6 months ago, different trigger
```

---

### Anti-Patterns to Avoid

❌ **Speed-optimizing Einstein:**

```python
# "I'll make Einstein faster with parallel reads"
Read(s1), Read(s2), Read(s3)  # WRONG: Defeats Einstein's purpose
```

**Fix**: If speed matters more than depth, use a different agent.

❌ **Shallow integration:**

```python
Read(source1)
Read(source2)  # WRONG: No thinking between reads
Read(source3)
# Summarize all three
```

**Fix**: Force thinking checkpoint after EACH read.

❌ **Using Einstein for simple lookup:**

```python
# "What files handle authentication?"
# WRONG: This is codebase-search's job, not Einstein's
```

**Fix**: Use appropriate agent for task complexity.

---

```

---

# Implementation Checklist for Orchestrator

## Phase 1: Low-Risk Templates (Week 1)

- [ ] Apply Template A to `codebase-search`
- [ ] Apply Template A to `haiku-scout`
- [ ] Verify Template A on `librarian` (should match existing)
- [ ] Apply Template B to `scaffolder`
- [ ] Test: Create 10-file project structure, verify depth layering
- [ ] Test: Parallel reads in search agents, verify 3-5x speedup
- [ ] Monitor for new failure modes

## Phase 2: Medium-Risk Templates (Weeks 2-3)

- [ ] Apply Template C to `code-reviewer`
- [ ] Apply Template C to `tech-docs-writer`
- [ ] Test: Code review with missing test files, verify graceful degradation
- [ ] Test: Docs update with missing source files, verify caveats in output
- [ ] Monitor for failure mode: Analysis without noting missing context

## Phase 3: High-Risk Templates (Weeks 4-7)

- [ ] Implement dependency graph tracking in tool layer
- [ ] Implement layer-based execution in tool layer
- [ ] Apply Template D to `python-pro`
- [ ] Apply Template D to `go-pro`
- [ ] Apply Template D to `r-pro`
- [ ] Apply Template D to `r-shiny-pro`
- [ ] Apply Template D to `python-ux`
- [ ] Apply Template D to `go-cli`
- [ ] Test: Multi-file implementation with dependencies, verify layering
- [ ] Test: Dependency conflict detection, verify rollback
- [ ] Monitor for failure mode: Broken code from incorrect layering

## Phase 4: Constrained Templates (Week 8)

- [ ] Apply Template E to `orchestrator`
- [ ] Apply Template E to `architect`
- [ ] Apply Template F to `einstein`
- [ ] Test: Orchestrator with parallel information gathering, verify speedup
- [ ] Test: Orchestrator blocks parallel Task() calls, verify enforcement
- [ ] Test: Einstein sequential reads, verify integration thinking
- [ ] Monitor for failure mode: Coordination conflicts

## Validation Gates

After EACH phase:
- [ ] Run 10+ test cases with new templates
- [ ] Measure performance improvement vs baseline
- [ ] Check for new failure modes in logs
- [ ] Verify output quality hasn't degraded
- [ ] Get user feedback on agent behavior

**Do NOT proceed to next phase if validation fails.**

---

# Tool Layer Enhancements Required

## For Template B (Scaffolder)
- [ ] Implement depth calculation for file paths
- [ ] Implement automatic depth-based layering
- [ ] Implement layer rollback on failure

## For Template C (Analysts)
- [ ] Implement priority tagging for Read() calls
- [ ] Implement partial success handling
- [ ] Implement caveat tracking and reporting

## For Template D (Implementation)
- [ ] Implement dependency graph tracking
- [ ] Implement topological sort for layer calculation
- [ ] Implement import analysis for automatic dependency detection
- [ ] Implement layer-based rollback

## For Template E (Orchestrator)
- [ ] Implement block on parallel Task() calls
- [ ] Implement warning on speculative spawning

## For Template F (Einstein)
- [ ] Implement block on parallel reads (sequential enforcement)
- [ ] Implement thinking checkpoint prompts

---

# Success Metrics

Track these metrics before and after implementation:

| Metric | Baseline | Target | Phase |
|--------|----------|--------|-------|
| Search agent avg task time | 15-20s | 3-5s | Phase 1 |
| Scaffolder avg creation time (10 files) | 50s | 15s | Phase 1 |
| Code review avg time | 30s | 8-10s | Phase 2 |
| Multi-file implementation time | 60s | 20-30s | Phase 3 |
| Orchestrator information gathering | 25s | 5-8s | Phase 4 |
| New failure modes introduced | 0 | 0 | All phases |
| Output quality regression | 0% | 0% | All phases |

---

**End of Implementation Guide**

**Status**: Ready for orchestrator review and phased rollout
**Estimated Total Implementation Time**: 8 weeks
**Expected System-Wide Performance Improvement**: 3-10x for applicable operations
**Risk Level**: Controlled (phased approach with validation gates)
```
