---
name: memory-improvement
description: Triggers a comprehensive audit of the system's memory. Compares "Live Memories" (decisions, sharp edges, facts) against "Static Configurations" (agents, conventions) to identify gaps, contradictions, and opportunities for improvement. Routes findings to the Orchestrator for synthesis.
---

# Memory Improvement Protocol

## Overview


**Workflow:**
1.  **Gather:** Collect all memory files, agent configs, and conventions.
3.  **Synthesize:** Delegate to `orchestrator` to review findings.
4.  **Plan:** (Optional) Orchestrator creates update tasks.

---

## Phase 1: Context Gathering (Bash)

**Step 1: Verify Memory Store exists**
If `~/.claude/memory/` is empty, abort with a warning.

**Step 2: Collect Files**
We need to pipe the following into Gemini:
*   `~/.claude/memory/**/*.md` (The "Live" Truth)
*   `~/.claude/agents/**/*.yaml` (The "Configured" Truth)
*   `~/.claude/conventions/*.md` (The "Rules")

```bash
# Verify memory exists
if [ -z "$(ls -A ~/.claude/memory 2>/dev/null)" ]; then
  echo "Memory store is empty. No audit possible."
  exit 1
fi

# Collect files
FILES=$(find ~/.claude/memory ~/.claude/agents ~/.claude/conventions -type f \( -name "*.md" -o -name "*.yaml" \) -not -path "*/.git/*")
```

---

## Phase 2: The Memory Audit (Gemini Pipeline)

**Step 1: Execute Gemini Protocol**

```bash
```

---

## Phase 3: Synthesis & Handoff (Orchestrator)

**Pipeline Enforcement:**
The output from Gemini will be complex. You **MUST** delegate to `orchestrator`.

```javascript
Task({
  description: "Synthesize Memory Audit Findings",
  subagent_type: "Explore",
  model: "sonnet",
  prompt: `AGENT: orchestrator
  1. TASK: Review the Gemini Memory Audit and synthesize an improvement plan
  2. EXPECTED OUTCOME: A list of specific config updates (e.g., "Add Sharp Edge X to Python Agent")
  3. REQUIRED SKILLS: Orchestrator - synthesis, system architecture
  4. REQUIRED TOOLS: Read, Task
  5. MUST DO:
     - Validate the Gemini findings (do they make sense?)
     - Prioritize "High Severity" gaps (e.g., missing sharp edges)
     - Propose concrete edits to YAML/MD files
  6. MUST NOT DO: Blindly apply changes without validation
  7. CONTEXT: [Insert Gemini Output Here]`
})
```

---

## Phase 4: Execution (Optional)

If the Orchestrator proposes concrete changes, you may proceed to:
1.  **Drafting:** Use `tech-docs-writer` to update conventions.
2.  **Configuring:** Use `write_file` (or relevant agent) to update Agent YAMLs.

*Note: Always ask for user confirmation before applying system-wide config changes.*
