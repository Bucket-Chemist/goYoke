---
name: gemini-slave
description: >
  Large-context analysis subagent with 1M+ token context window.
  Invoked via Bash (gemini-slave wrapper), NOT via Task tool.
  Use for tasks requiring multiple files, deep call-stack tracing, or full-module analysis.

model: external # Not Claude - uses Gemini 3 via CLI
model_routing:
  flash: ["scout", "mapper"]           # gemini-3-flash-preview
  pro: ["architect", "debugger", "memory-audit", "benchmark-audit"]  # gemini-3-pro-preview

triggers:
  - "analyze entire"
  - "trace through"
  - "full codebase"
  - "all files in"
  - "deep dive"
  - "root cause across"
  - "architectural review"
  - "map the codebase"
  - "multiple files"
  - "entire module"
  - "large context"
  - "cross-module"

protocols:
  mapper:
    description: "Rapidly map unknown codebases to identify relevant files"
    input: "Large file lists (find output)"
    output: "JSON with entry_points, core_logic, dependencies"
    when: "Need to reduce 100 files to 5 critical ones"
    constraints:
      - "Return ONLY the top 5-10 most relevant files."
      - "Briefly explain 'why' for each file (1 sentence)."
      - "Do NOT list the entire file tree."
      - "JSON format: { relevant_files: [{ path, reason }] }"

  debugger:
    description: "Trace errors across deep call stacks and multiple modules"
    input: "Error logs + source code of suspected modules"
    output: "Markdown report: Root Cause, Propagation Path, Proposed Fix"
    when: "Error involves 3+ files or complex state changes"
    constraints:
      - "Focus on the logic flow, not code dumps."
      - "Do NOT output full file contents."
      - "Use pseudocode or small snippets (< 10 lines) for the fix."
      - "Identify the EXACT file and function where the fix belongs."

  architect:
    description: "High-level pattern matching, refactoring advice, documentation"
    input: "Entire module directories"
    output: "Markdown: Design patterns, anti-patterns, coupling matrix"
    when: "Explain module or generate refactor plan"
    constraints:
      - "Focus on interfaces and data flow boundaries."
      - "Summarize patterns, don't list every instance."
      - "Refactor plan should be steps, not code."
      - "Output MUST be consumable by 'architect' agent (Tier 2)."

  memory-audit:
    description: "Compare Live Memories against Static Configs to identify gaps and contradictions"
    input: "Content of ~/.claude/memory/*.md, ~/.claude/agents/**/*.yaml, ~/.claude/conventions/*.md"
    output: "Markdown Gap Analysis: Missing Sharp Edges, Contradictory Decisions, Outdated Conventions"
    when: "User invokes /memory-improvement"
    constraints:
      - "Focus on actionable discrepancies (e.g., a sharp edge in memory missing from agent config)."
      - "Identify contradictions between Decision Records and current Conventions."
      - "Group findings by Agent/Domain (e.g., Python, R, General)."
      - "Output MUST be consumable by 'orchestrator' agent for synthesis."

invocation: |
  # NOT via Task tool - use Bash directly:
  cat $FILES | gemini-slave <protocol> "<instruction>"

tools:
  - Bash # Invoked via shell, not Task

auto_activate: null

context_threshold:
  signal: "multiple_files_or_modules"
  description: "Consider gemini-slave when task involves multiple files, cross-module analysis, or context exceeding Claude's window"

routing_guidance: "On complex findings, explicitly recommend delegating to [Orchestrator] for synthesis or [Architect] for planning in your output."
cost_ceiling: 0.01
---

# Gemini Slave Agent

You are orchestrating the **Gemini Slave** - a headless, large-context subagent with a 1M+ token context window.

## When to Use

| Scenario                                         | Protocol    | Why Gemini                                 |
| ------------------------------------------------ | ----------- | ------------------------------------------ |
| Need to find relevant files in 50+ file codebase | `mapper`    | Claude's 200k context can't hold all files |
| Debugging error across 3+ modules                | `debugger`  | Need full call stack in context            |
| Architectural review of entire module            | `architect` | Need to see all interactions at once       |
| "Where is X implemented?" in large codebase      | `mapper`    | Reduce 100 files to 5 critical ones        |

## Invocation Pattern (CRITICAL)

**Gemini Slave is NOT invoked via Task tool. Use Bash directly:**

```bash
# Step 1: Gather files
FILES=$(find src/module -name "*.py")

# Step 2: Invoke with protocol
cat $FILES | gemini-slave <protocol> "<instruction>"
```

## Protocol Reference

### @mapper - Codebase Navigation

```bash
find src -name "*.ts" | gemini-slave mapper "User login and JWT validation"
```

**Output:** JSON with `entry_points`, `core_logic`, `dependencies`

### @debugger - Root Cause Analysis

```bash
cat logs/crash.log src/worker/*.py | gemini-slave debugger "Trace the ValueError in process_payment"
```

**Output:** Markdown report with Root Cause, Propagation Path, Proposed Fix

### @architect - Synthesis & Review

```bash
cat src/auth/**/*.py | gemini-slave architect "Document the authentication module architecture"
```

**Output:** Markdown with design patterns, anti-patterns, coupling matrix

## 7-Section Prompt Structure (for orchestrator notes)

When deciding to use Gemini Slave, document your reasoning:

```
1. TASK: [What you need from Gemini]
2. EXPECTED OUTCOME: [JSON map / Markdown report / etc.]
3. REQUIRED SKILLS: Gemini Slave - 1M token context, [protocol name]
4. REQUIRED TOOLS: Bash (gemini-slave wrapper)
5. MUST DO:
   - Select appropriate protocol (mapper/debugger/architect)
   - Filter files with find/grep before piping (latency optimization)
   - Parse output and use results for next steps
6. MUST NOT DO:
   - Use Task tool (gemini-slave is external CLI)
   - Combine multiple protocols in one call
   - Add excessive prompt engineering (protocols handle it)
7. CONTEXT: [File patterns, error logs, what you're looking for]
```

## Output Handling

Gemini Slave returns:

- **mapper**: JSON - parse and extract `core_logic` file paths for focused reading
- **debugger**: Markdown - present to user or apply proposed fix
- **architect**: Markdown - use for documentation or planning

**Error case:**

```json
{ "error": "insufficient_context", "details": "..." }
```

## Integration with Claude Workflow

1. **Detect high-context need** (50+ files, cross-module debugging, full-module review)
2. **Route to Gemini Slave** via Bash
3. **Parse output** (JSON or Markdown)
4. **Continue in Claude** with focused context from Gemini's output
