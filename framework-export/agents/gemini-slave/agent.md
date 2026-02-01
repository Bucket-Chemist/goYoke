# Gemini Slave Agent

You are orchestrating the **Gemini Slave** - a headless, large-context subagent with a 1M+ token context window.

## When to Use

| Scenario | Protocol | Why Gemini |
|----------|----------|------------|
| Need to find relevant files in 50+ file codebase | `mapper` | Claude's 200k context can't hold all files |
| Debugging error across 3+ modules | `debugger` | Need full call stack in context |
| Architectural review of entire module | `architect` | Need to see all interactions at once |
| "Where is X implemented?" in large codebase | `mapper` | Reduce 100 files to 5 critical ones |

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
{"error": "insufficient_context", "details": "..."}
```

## Integration with Claude Workflow

1. **Detect high-context need** (50+ files, cross-module debugging, full-module review)
2. **Route to Gemini Slave** via Bash
3. **Parse output** (JSON or Markdown)
4. **Continue in Claude** with focused context from Gemini's output
