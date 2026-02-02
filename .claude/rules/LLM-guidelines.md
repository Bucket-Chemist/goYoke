---
paths:
  - "**/*"
---

# Guidelines for Maximizing Claude's Effectiveness

This document defines best practices for leveraging Claude models across all coding tasks. These patterns apply regardless of language (Python, R) or domain (ML, cloud, data science).

---

## Core Principles

### Context is Everything
Claude's effectiveness scales with context quality. Provide:
- **Complete type/class definitions** before asking for methods
- **Representative data samples** (structure, not full datasets)
- **Full error tracebacks** when debugging
- **Related code** that new code must integrate with
- **Constraints** (performance, memory, compatibility)

### Explicit Over Implicit
Never assume Claude knows your project conventions:
- Reference specific rule files: "Follow the conventions in R.md"
- State expected return types explicitly
- Specify edge cases that must be handled
- Declare performance requirements upfront

### Iterative Refinement Over One-Shot
Complex tasks benefit from staged approaches:
1. Skeleton/interface first
2. Implementation second
3. Tests third
4. Optimization fourth

---

## Task Specification Patterns

### The Complete Specification Pattern
For non-trivial functions, provide:
```
TASK: [What to build]
INPUTS: [Types, shapes, constraints]
OUTPUTS: [Expected return type/structure]
EDGE CASES: [What to handle]
INTEGRATES WITH: [Existing code/classes]
CONSTRAINTS: [Performance, memory, compatibility]
EXAMPLE: [Representative input → expected output]
```

### The Debugging Pattern
When asking Claude to debug:
```
OBSERVED: [What happened - include full error]
EXPECTED: [What should happen]
CONTEXT: [Relevant code, recent changes]
ATTEMPTED: [What you already tried]
```

### The Review Pattern
When requesting code review:
```
REVIEW AGAINST: [Specific rules/criteria]
FOCUS AREAS: [Performance, security, style, etc.]
CONSTRAINTS: [What cannot change]
```

---

## Leveraging Claude Code Features

### Plan Mode for Architecture
**MUST** use plan mode (`EnterPlanMode`) for:
- New feature implementations with multiple valid approaches
- Architectural decisions (state management, data flow)
- Multi-file refactoring
- Any task where user preference matters

Plan mode allows exploration before commitment.

### Parallel Agents for Research
Use the `Task` tool with multiple agents when:
- Researching best practices across sources
- Running multiple specialized code reviews
- Exploring different implementation approaches
- Gathering context from multiple codebases

```
Example: "Launch parallel agents to research Bubbletea tea.Model patterns
and lipgloss styling conventions, then synthesize recommendations"
```

### Todo Tracking for Complex Tasks
For multi-step implementations:
- Break down into discrete, verifiable steps
- Track progress explicitly
- Mark completions as you go
- Add discovered sub-tasks dynamically

---

## Verification and Self-Review

### Request Self-Review
After Claude generates code, ask:
- "Review this against the [language].md conventions"
- "Identify edge cases this doesn't handle"
- "What are the performance characteristics?"
- "Generate test cases for this function"

### Staged Verification
1. **Correctness**: "Does this logic handle [specific case]?"
2. **Style**: "Does this follow our naming conventions?"
3. **Performance**: "What's the complexity? Any bottlenecks?"
4. **Security**: "Any injection risks or data leaks?"
5. **Tests**: "Generate testthat/pytest cases for edge conditions"

### The Rubber Duck Pattern
Ask Claude to explain complex generated code:
- Forces verification of logic
- Surfaces implicit assumptions
- Identifies documentation gaps

---

## Domain-Specific Patterns

### Machine Learning (PyTorch/TensorFlow)

**Model Architecture Specification:**
```
ARCHITECTURE: [Model type - CNN, Transformer, etc.]
INPUT SHAPE: [Batch, channels, height, width] or [Batch, seq_len, features]
OUTPUT SHAPE: [Expected output dimensions]
LAYERS: [Key layer specifications]
LOSS: [Loss function and any weighting]
DEVICE: [CPU, CUDA, TPU considerations]
```

**Training Loop Requests:**
- Specify batch size, gradient accumulation steps
- State mixed precision requirements (fp16/bf16)
- Define checkpointing strategy
- Specify distributed training needs (DDP, FSDP)

**Data Pipeline Requests:**
- Input data format and source
- Augmentation requirements
- Preprocessing steps
- Memory constraints (streaming vs. in-memory)

**Debugging ML Code:**
- Always include tensor shapes at failure point
- Provide device placement info
- Include gradient flow concerns
- State memory usage observations

### Cloud Storage (GCS)

**Authentication Context:**
- Service account vs. user credentials
- Environment (local dev, Cloud Run, GKE, Vertex AI)
- Required IAM permissions

**Data Transfer Patterns:**
- Batch size for parallel operations
- Resumable upload requirements
- Streaming vs. download-then-process
- Cost considerations (egress, operations)

**Path Handling:**
- Always use `gs://bucket/path` URI format
- Clarify blob vs. prefix operations
- State whether recursive operations needed

### Large Data Processing

**Memory Constraints:**
- State available memory
- Specify chunking requirements
- Indicate streaming needs
- Define acceptable memory/speed tradeoffs

**Parallelization Context:**
- Number of available cores/workers
- I/O vs. CPU bound nature
- Shared state requirements
- Progress tracking needs

### Go Development (Primary Language)

**Code Structure Specification:**
```
PACKAGE: [package name and responsibility]
IMPORTS: [expected dependencies]
TYPES: [structs, interfaces to define]
FUNCTIONS: [public API with signatures]
ERROR HANDLING: [error types, wrapping strategy]
TESTS: [table-driven test expectations]
```

**Interface Design:**
- Keep interfaces small (1-3 methods)
- Define interfaces at the point of use, not implementation
- Accept interfaces, return concrete types
- Example: `io.Reader` over custom `DataSource`

**Error Handling Patterns:**
- Always handle errors explicitly (no `_` for errors)
- Wrap errors with context: `fmt.Errorf("loading config: %w", err)`
- Define sentinel errors for expected conditions
- Use error types for errors that need inspection

**Concurrency Specification:**
```
PATTERN: [fan-out/fan-in, worker pool, pipeline]
GOROUTINES: [number and responsibility]
SYNCHRONIZATION: [channels, mutex, errgroup]
CANCELLATION: [context.Context usage]
ERROR PROPAGATION: [how errors surface]
```

**Common Go Agents:**
| Agent | Use For | Key Patterns |
|-------|---------|--------------|
| `go-pro` | Core implementation | Idiomatic Go, error handling |
| `go-cli` | CLI apps (Cobra) | Flags, subcommands, help text |
| `go-tui` | TUI apps (Bubbletea) | tea.Model, tea.Cmd, lipgloss |
| `go-api` | HTTP clients/servers | Context, retry, rate limiting |
| `go-concurrent` | Concurrency | errgroup, channels, context |

**Testing Patterns:**
- Table-driven tests for multiple cases
- Subtests with `t.Run()` for organization
- Test helpers with `t.Helper()` marking
- `testdata/` directory for fixtures
- Example: `TestParseConfig/valid_json`, `TestParseConfig/missing_field`

**GOgent-Fortress Specific:**
- All hook binaries read JSON from STDIN with 5s timeout
- Output JSON to STDOUT for Claude Code consumption
- Use `pkg/routing` for schema validation
- Use `pkg/session` for handoff operations
- JSONL files are append-only (never rewrite)

---

## Anti-Patterns to Avoid

### Vague Specifications
| Bad | Good |
|-----|------|
| "Make it faster" | "Reduce complexity from O(n²) to O(n log n)" |
| "Handle errors" | "Catch ConnectionError, retry 3x with exponential backoff" |
| "Add logging" | "Log at INFO level with structured fields: user_id, operation, duration" |
| "Make it work with big data" | "Must handle 10M rows with <8GB RAM using chunked processing" |

### Missing Context
| Bad | Good |
|-----|------|
| "Fix this function" | "Fix this function [full code]. Error: [full traceback]. Expected: [behavior]" |
| "Add a method to MyClass" | "Add a method to MyClass [class definition]. Must integrate with [related code]" |
| "Optimize the model" | "Optimize: current 2.3s/batch on V100, target <1s. Bottleneck appears in attention layer" |

### Skipping Verification
| Bad | Good |
|-----|------|
| Accept first output | "Review this against our style guide before I use it" |
| Assume edge cases handled | "What edge cases does this not handle?" |
| Trust performance claims | "Profile this and identify actual bottlenecks" |

---

## Prompting Strategies by Task Type

### New Feature Implementation
1. Start with plan mode for architecture
2. Define interfaces/contracts first
3. Implement core logic
4. Add error handling
5. Generate tests
6. Self-review against rules

### Bug Fixing
1. Provide full error context
2. Share relevant code sections
3. State what you've tried
4. Ask for root cause analysis
5. Request fix with explanation
6. Ask for regression test

### Code Review
1. Specify review criteria explicitly
2. Request structured feedback (severity, location, suggestion)
3. Ask for specific rule violations
4. Request improvement alternatives

### Refactoring
1. State refactoring goals (performance, readability, testability)
2. Define what must not change (API, behavior)
3. Request incremental changes
4. Ask for verification strategy

### Performance Optimization
1. Provide profiling data
2. State performance targets
3. Specify acceptable tradeoffs
4. Request complexity analysis
5. Ask for benchmarking approach

---

## Context Window Optimization

### What to Include
- Full class/type definitions for code being modified
- Representative sample data (structure, not volume)
- Error messages with complete tracebacks
- Related functions that must integrate
- Relevant configuration/constants

### What to Summarize
- Large datasets → representative samples + schema
- Long files → relevant sections + structure overview
- History → key decisions and constraints

### What to Reference
- Rule files by name: "per go.md conventions"
- Previous conversation context: "as discussed above"
- External docs: use WebFetch tool or MCP-provided fetch tools

---

## Multi-Model Strategy

### CRITICAL: Tiered Model Routing

Use `Task(model: "haiku")` or `Task(model: "sonnet")` to delegate work to cheaper models. Only keep quality-critical tasks in Opus.

### When to Use Different Models

| Task Type | Model | Rationale |
|-----------|-------|-----------|
| **OPUS (Quality Critical)** | | |
| Interview/requirements gathering | Opus | Quality of questions determines outcome |
| Planning/architecture | Opus | Complex tradeoffs need depth |
| Cross-domain synthesis | Opus | Connecting 5+ sources needs reasoning |
| Conflict judgment | Opus | Requires nuanced assessment |
| MCP API calls | Opus | Direct API, delegation overhead exceeds savings |
| **SONNET (Reasoning, Familiar)** | | |
| Go implementation (go-pro, go-tui, go-cli) | Sonnet | Needs reasoning, follows Go idioms |
| Code understanding | Sonnet | Needs reasoning but standard patterns |
| Core implementation | Sonnet | Following established patterns |
| Single-domain analysis | Sonnet | Focused analysis, not cross-cutting |
| Documentation generation | Sonnet | Structured output with reasoning |
| Concurrency design (go-concurrent) | Sonnet | Channel patterns, error propagation |
| **HAIKU (Mechanical Work)** | | |
| File discovery (glob, find, ls) | Haiku | Pure file operations |
| Pattern extraction (grep, regex) | Haiku | Mechanical matching |
| Keyword extraction | Haiku | Text parsing |
| Result formatting | Haiku | Structured output, no reasoning |
| Skill/index loading | Haiku | File reading |
| Boilerplate generation | Haiku | Template following |
| Sharp edge detection | Haiku | Pattern matching against known list |
| Code review (style only) | Haiku | Convention checking, no design judgment |

### Routing Enforcement

When using `/explore` or similar workflows:

1. **ALWAYS announce routing** with `[ROUTING] → Model (reason)`
2. **Use Task tool** with explicit `model: "haiku"` or `model: "sonnet"`
3. **Never use Glob/Grep/Read directly** for exploration - spawn Haiku scouts
4. **Stay in Opus** only for interview, planning, synthesis, and judgment

### Cost Impact

Aggressive tiered routing saves ~70% on exploration workflows:
- Haiku: ~$0.0005/1k tokens (50x cheaper than Opus)
- Sonnet: ~$0.009/1k tokens (5x cheaper than Opus)
- Opus: ~$0.045/1k tokens (baseline)

### Parallel Agent Patterns

For complex research tasks, launch multiple Haiku scouts in parallel:
```
- Haiku Scout 1: File discovery (glob patterns)
- Haiku Scout 2: Pattern extraction (grep)
- Haiku Scout 3: Code snippet extraction
→ Sonnet Analyst: Synthesize findings
→ Opus Main: Make architectural decisions
```

For Go implementation tasks, consider:
```
- Haiku Scout: Find existing patterns (grep for similar interfaces)
- go-pro (Sonnet): Implement core logic
- code-reviewer (Haiku): Verify conventions
```

---

## Effective Feedback Loops

### When Output Isn't Right
Instead of: "That's wrong, try again"
Use: "The output [specific issue]. The constraint is [constraint]. Consider [hint]"

### Building on Previous Output
Instead of: Starting fresh
Use: "Keep [what worked], but modify [specific part] to [desired change]"

### Incremental Complexity
Instead of: Full implementation request
Use:
1. "Create the interface/skeleton"
2. "Implement the core logic"
3. "Add error handling for [cases]"
4. "Optimize [specific bottleneck]"

---

## Checklist: Before Asking Claude

- [ ] Have I provided complete type/class definitions?
- [ ] Have I included representative examples?
- [ ] Have I stated constraints explicitly?
- [ ] Have I referenced relevant rule files?
- [ ] Have I specified what "done" looks like?
- [ ] Am I using plan mode for complex tasks?
- [ ] Should I break this into smaller requests?

---

## Enforcement Architecture

### The Anti-Pattern: Documentation Theater

**Definition:** Adding imperative enforcement language ("MUST NOT", "NEVER", "BLOCKED") to CLAUDE.md or other documentation files, creating the illusion of enforcement without any actual mechanism.

**Why it fails:**
- Text instructions are probabilistic suggestions, not deterministic rules
- Attention to early instructions degrades over long conversations
- No mechanism exists to actually BLOCK a tool call via text
- Creates false confidence that behavioral problems are "solved"
- CLAUDE.md becomes bloated with unenforceable imperatives

### The Correct Pattern: Declarative → Programmatic → Reference

**Three components, in order:**

1. **Declarative Rule** (`routing-schema.json`)
   - Single source of truth for what's allowed/blocked
   - Parsed by hooks at runtime
   - Example: `"task_invocation_blocked": true`

2. **Programmatic Enforcement** (Go hook binary, e.g., `gogent-validate`)
   - Actually runs before/after tool use
   - Can block, warn, or modify behavior
   - Example: Check schema rule, return `routing.BlockResponse()` with reason

3. **Reference Documentation** (`CLAUDE.md`)
   - Points to enforcement, doesn't replace it
   - Example: "Blocked by gogent-validate (PreToolUse hook)"
   - Provides context for WHY, not enforcement of WHAT

### Decision Tree: Where Does This Go?

```
Is this enforcement of a behavior?
│
├─ YES: Can it be detected programmatically?
│   │
│   ├─ YES: What kind of enforcement?
│   │   │
│   │   ├─ Block action → routing-schema.json rule
│   │   │                 + gogent-validate check (Go binary)
│   │   │                 + CLAUDE.md reference
│   │   │
│   │   ├─ Require action → Hook injects reminder at trigger
│   │   │                   + CLAUDE.md documents workflow
│   │   │
│   │   └─ Warn on pattern → PreToolUse hook with warning
│   │                        + CLAUDE.md notes the check
│   │
│   └─ NO: Is it methodology guidance?
│       │
│       ├─ YES → LLM-guidelines.md (this file)
│       │
│       └─ NO → agent-behavior.md or conventions/*.md
│
└─ NO: Is this describing existing system behavior?
    │
    ├─ YES → CLAUDE.md (gates, workflows, triggers)
    │
    └─ NO → Probably doesn't need to be written
```

### What Goes Where: Quick Reference

| Need | ❌ Wrong | ✅ Right |
|------|----------|----------|
| Block a tool pattern | "You MUST NOT use X" in CLAUDE.md | `routing-schema.json` rule + `gogent-validate` enforcement + CLAUDE.md reference |
| Require pre-check | "ALWAYS check Y first" in CLAUDE.md | Hook injects reminder at trigger point |
| Prevent anti-pattern | "NEVER do Z" in CLAUDE.md | This section in LLM-guidelines.md + warning hook |
| Document workflow | Gates 1-5 in CLAUDE.md | ✅ Appropriate (this IS documentation) |
| Agent-specific rule | In CLAUDE.md | `agents/*/sharp-edges.yaml` or `agents/*/{agent-name}.md` (unified frontmatter) |

### Pre-Commit Checklist for CLAUDE.md Edits

Before adding enforcement-style language to CLAUDE.md:

- [ ] Is this DESCRIPTION of existing behavior, or ENFORCEMENT of new behavior?
- [ ] If enforcement: Is it implemented in a hook FIRST?
- [ ] Does CLAUDE.md text REFERENCE the hook (file + line), not REPLACE it?
- [ ] Are there any new "MUST", "NEVER", "BLOCKED" without corresponding code?
- [ ] Would this still work if the LLM ignores this paragraph?

If any answer is wrong, implement programmatic enforcement first.

### What CLAUDE.md IS For

✅ **Appropriate content:**
- Gates (workflow checkpoints with structure)
- Trigger tables (pattern → agent mapping)
- System constraints (Arch Linux, Python paths)
- References ("See hook X for enforcement")
- Context loading (conventions, skills)

❌ **Inappropriate content:**
- Behavioral blocking ("MUST NOT use X")
- Imperative requirements without enforcement
- Rules that depend on LLM "remembering"
- Anything that fails silently when ignored

### Example: Correct vs Incorrect

**Scenario:** Need to prevent Task(opus) invocations

❌ **Incorrect (documentation theater):**
```markdown
## Gate 6: Einstein Protection

**You MUST NOT invoke Einstein via Task tool.**
**This is BLOCKED. Use /einstein slash command instead.**
```

✅ **Correct (layered enforcement):**

1. `routing-schema.json`:
```json
"opus": {
  "task_invocation_blocked": true,
  "blocked_reason": "60K+ token inheritance overhead"
}
```

2. `cmd/gogent-validate/main.go`:
```go
if event.Task != nil && event.Task.Model == "opus" {
    return routing.BlockResponse(
        "Task(opus) blocked by gogent-validate. Use /einstein instead.",
    )
}
```

3. `CLAUDE.md`:
```markdown
## Gate 6: Einstein Escalation

Einstein invocation via Task tool is blocked by `gogent-validate` (PreToolUse hook).
See `routing-schema.json` → `opus.task_invocation_blocked`.

When Einstein triggers fire, use `escalate_to_einstein` protocol instead.
Reference: `~/.claude/skills/einstein/SKILL.md`
```

The CLAUDE.md version describes and references; it doesn't pretend to enforce.

---

**Remember:** Claude's output quality is bounded by input quality. Invest in context.
