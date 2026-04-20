# Agent Guidelines

Guidelines for subagents executing tasks within goYoke.

## 1. Coding Discipline

These principles apply to all coding work, before routing considerations.

### 1.1 Surface Assumptions Before Committing

Before implementing non-trivial code:

**State what you're assuming.** If the request could reasonably be interpreted
multiple ways, name them. Don't silently pick one and hope it's right.

**Push back when warranted.** If a simpler approach exists, say so. If the
requested approach has significant tradeoffs, surface them. Technical honesty
serves the user better than compliance.

**Stop when genuinely confused.** Name what's unclear. One targeted question
beats three attempts that miss the mark.

This doesn't mean interrogating the user on every detail. Use judgment:
- Obvious intent → proceed
- Ambiguous with reasonable default → state assumption, proceed
- Ambiguous with no clear default → ask

### 1.2 Solve the Actual Problem

Write code that addresses what was asked. Resist the pull toward:

- **Speculative features** - functionality nobody requested
- **Premature abstraction** - generalizing one-off code "for later"
- **Defensive overkill** - error handling for scenarios that can't occur

That said, use judgment. Sometimes a small abstraction genuinely clarifies.
Sometimes adjacent error handling prevents real bugs. The test isn't "was
this explicitly requested?" but "does this serve the user's actual goal?"

**The checks:**

1. If you wrote 200 lines and it could be 50, rewrite it.

2. Ask yourself: "Would a senior engineer say this is overcomplicated?"
   If yes, simplify.

3. If you added code beyond what was asked, ask why. "Might be useful
   someday" → reconsider. "Genuinely makes this clearer or more correct"
   → proceed.

### 1.3 Parallelize Independent Operations

When executing a task, identify operations that don't depend on each other
and execute them in a single message with multiple tool calls.

**How to parallelize:** Include multiple tool invocations in the same response.
The runtime executes them concurrently and returns all results together.

**Parallelize (no dependencies between calls):**
```javascript
// GOOD: Single message, multiple independent reads
Read({file_path: "/src/auth/handler.go"})
Read({file_path: "/src/auth/middleware.go"})
Read({file_path: "/src/auth/types.go"})
// All three execute concurrently, results return together
```

**Don't parallelize (output informs next input):**
```javascript
// These MUST be sequential - each depends on the previous
Read({file_path: "/src/config.go"})        // Need content first
// ...wait for result...
Edit({file_path: "/src/config.go", ...})   // Edit depends on read
// ...wait for result...
Bash({command: "go build ./..."})          // Build depends on edit
```

**Common parallelizable patterns:**

| Scenario | Parallel Calls |
|----------|----------------|
| Understanding a module | Read 3-5 related files |
| Finding patterns | Multiple Grep/Glob searches |
| Validating changes | Bash(go vet) + Bash(go test) + Bash(golangci-lint) |
| Exploring structure | Glob for *.go + Glob for *_test.go + Read go.mod |

**The check:** Before making a tool call, ask: "Do I need the result of a
previous call to determine this call's parameters?" If no → batch it.

---

## 2. Parallel Agent Management

### 2.1 Background vs Foreground

| Pattern | Use When | Mechanism |
|---------|----------|-----------|
| **Foreground (default)** | Next step depends on this output | `Task({...})` |
| **Background** | Independent work, will collect later | `Bash({..., run_in_background: true})` |
| **Parallel foreground** | Multiple independent, need all before continuing | Multiple `Task()` in same message |

### 2.2 MANDATORY: Background Task Collection

**Enforcement:** `goyoke-orchestrator-guard` (SubagentStop hook) blocks orchestrator completion when background tasks remain uncollected.

**If you spawn background tasks, you MUST:**

1. Track every task_id returned
2. Before ANY final output or synthesis:
   ```javascript
   TaskOutput({task_id: "bg-task-1", block: true})
   TaskOutput({task_id: "bg-task-2", block: true})
   ```
3. NEVER conclude orchestration with uncollected background tasks

**Violation Pattern (BLOCKED by hook):**
```javascript
Bash({..., run_in_background: true})  // Spawned
Bash({..., run_in_background: true})  // Spawned
// ... do other work ...
// Output synthesis WITHOUT calling TaskOutput → BLOCKED by goyoke-orchestrator-guard
```

### 2.3 Fan-Out, Fan-In Pattern

For parallel information gathering:

```javascript
// 1. FAN-OUT: Spawn all tasks
const task1 = Task({...})  // Returns task_id
const task2 = Task({...})  // Returns task_id
const task3 = Task({...})  // Returns task_id

// 2. FAN-IN: Collect all results (MANDATORY)
const result1 = TaskOutput({task_id: task1, block: true})
const result2 = TaskOutput({task_id: task2, block: true})
const result3 = TaskOutput({task_id: task3, block: true})

// 3. SYNTHESIZE: Only after all collected
// Now proceed with synthesis
```

---

## 3. Output Quality

### 3.1 Self-Verification

Before returning output to user:
1. Does it answer the actual question?
2. Does it follow relevant conventions?
3. Are there obvious errors?
4. Would a quick code-reviewer pass help?

### 3.2 Critic Pattern (Optional)

For important outputs, invoke quick review:
```javascript
Task({
  model: "haiku",
  prompt: "Review this output for obvious errors: [output]"
})
```

Cost: ~$0.005. Worth it for user-facing deliverables.

---

## 4. Core Principles

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

## 5. Task Specification Patterns

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

## 6. Verification and Self-Review

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

## 7. Domain-Specific Patterns

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

---

## 8. Effective Feedback Loops

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

## 9. Anti-Patterns to Avoid

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

### System-Level Anti-Patterns (Agent Behavior)

| Anti-Pattern | Why Bad | Correct Approach |
|--------------|---------|------------------|
| Retrying identically after failure | Wastes tokens, won't help | Modify approach (different tool, smaller scope, more context) |
| Using Sonnet for file search | 50x cost waste | Use Haiku/codebase-search |
| Spawning background tasks without collecting | Orphaned work, blocked by goyoke-orchestrator-guard | Always call TaskOutput before final synthesis |
| Ignoring hook injections | Misses guidance/enforcement | Read and follow additionalContext from hooks |
| Skipping scout on unknown scope | Potential mis-routing to expensive tier | Scout first with haiku-scout or goyoke-scout |

---

## 10. Completion Checklist

- [ ] All background tasks collected?
- [ ] Routing tier was appropriate?
- [ ] No obvious errors in output?
- [ ] Sharp edges documented if any?
- [ ] Conventions followed?
- [ ] User's actual question answered?

---

**Remember:** Your effectiveness is bounded by coding discipline and routing discipline. Overcomplicated code or wrong tier = wasted effort + suboptimal output.
