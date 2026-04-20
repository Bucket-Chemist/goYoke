# Installing a New Subagent in goYoke

> **Version:** 1.0
> **Author:** Einstein Analysis
> **Last Updated:** 2026-01-25
> **Prerequisite:** goYoke framework installed (see INSTALL-GUIDE.md)

---

## Table of Contents

1. [Overview](#1-overview)
2. [Agent Architecture](#2-agent-architecture)
3. [Quick Start (5-Minute Setup)](#3-quick-start-5-minute-setup)
4. [Detailed Configuration](#4-detailed-configuration)
5. [Tier Selection Guide](#5-tier-selection-guide)
6. [Subagent Type Selection](#6-subagent-type-selection)
7. [Registration & Enforcement](#7-registration--enforcement)
8. [Testing Your Agent](#8-testing-your-agent)
9. [Common Patterns](#9-common-patterns)
10. [Troubleshooting](#10-troubleshooting)
11. [Checklist](#11-checklist)

---

## 1. Overview

### What is a Subagent?

A subagent is a specialized Claude instance with:
- **Defined scope** (what it can do)
- **Tool permissions** (what tools it can use)
- **Model tier** (how much reasoning power it has)
- **Conventions** (what rules it follows)

### Files Required

| File | Purpose | Required? |
|------|---------|-----------|
| `~/.claude/agents/{name}/agent.yaml` | Configuration | **YES** |
| `~/.claude/agents/{name}/agent.md` | Detailed documentation | **YES** |
| `~/.claude/agents/{name}/sharp-edges.yaml` | Known pitfalls | Recommended |
| Update `~/.claude/agents/agents-index.json` | Register in index | **YES** |
| Update `~/.claude/routing-schema.json` | Subagent type mapping | **YES** |
| `~/.claude/conventions/{lang}.md` | Language conventions | If needed |

---

## 2. Agent Architecture

### Tier Hierarchy

```
┌─────────────────────────────────────────────────────────────────┐
│                         OPUS (Tier 4)                           │
│  Deep analysis, complex reasoning                               │
│  Cost: $0.045/1K tokens | Thinking: 32K                        │
│  Invocation: /einstein ONLY (Task blocked)                      │
│  Agents: einstein                                               │
├─────────────────────────────────────────────────────────────────┤
│                        SONNET (Tier 3)                          │
│  Implementation, refactoring, debugging                         │
│  Cost: $0.009/1K tokens | Thinking: 16K                        │
│  Agents: python-pro, go-pro, orchestrator, architect           │
├─────────────────────────────────────────────────────────────────┤
│                   HAIKU + THINKING (Tier 1.5)                   │
│  Structured tasks, documentation, review                        │
│  Cost: $0.001/1K tokens | Thinking: 6K                         │
│  Agents: scaffolder, tech-docs-writer, librarian               │
├─────────────────────────────────────────────────────────────────┤
│                        HAIKU (Tier 1)                           │
│  Mechanical work, search, formatting                            │
│  Cost: $0.0005/1K tokens | Thinking: None                      │
│  Agents: codebase-search, haiku-scout                          │
├─────────────────────────────────────────────────────────────────┤
│                       EXTERNAL (Tier 0)                         │
│  Large context (1M+ tokens)                                     │
│  Cost: — | External: deprecated                                │
│  (deprecated — no agents)                                       │
└─────────────────────────────────────────────────────────────────┘
```

### Subagent Types

| Type | Write Access | Use For |
|------|--------------|---------|
| `Explore` | NO | Read-only reconnaissance (search, review) |
| `general-purpose` | YES | Implementation, documentation, edits |
| `Plan` | YES | Architecture, orchestration, planning |
| `Bash` | NO | External process execution (deprecated) |

---

## 3. Quick Start (5-Minute Setup)

### Step 1: Create Agent Directory

```bash
mkdir -p ~/.claude/agents/my-new-agent
```

### Step 2: Create agent.yaml

```yaml
# ~/.claude/agents/my-new-agent/agent.yaml
name: my-new-agent
description: >
  Brief description of what this agent does and when to use it.

model: sonnet          # haiku, sonnet, or opus
thinking:
  enabled: true
  budget: 10000        # tokens for reasoning

triggers:
  - "pattern1"
  - "pattern2"
  - "when user says this"

tools:
  - Read
  - Write
  - Edit
  - Bash
  - Grep
  - Glob

conventions_required:
  - go.md              # optional: load specific conventions

failure_tracking:
  max_attempts: 3
  on_max_reached: "escalate_to_orchestrator"
```

### Step 3: Create agent.md

```markdown
# ~/.claude/agents/my-new-agent/agent.md

# My New Agent

## Purpose
What this agent does.

## When to Use
- Situation 1
- Situation 2

## Behavior
How it should operate.

## Output Format
What results look like.
```

### Step 4: Create sharp-edges.yaml

```yaml
# ~/.claude/agents/my-new-agent/sharp-edges.yaml
edges: []
# Add entries as you discover pitfalls
```

### Step 5: Register in agents-index.json

Add to `~/.claude/agents/agents-index.json`:

```json
{
  "id": "my-new-agent",
  "parallelization_template": "B",
  "name": "My New Agent",
  "model": "sonnet",
  "thinking": true,
  "thinking_budget": 10000,
  "tier": 2,
  "category": "task",
  "path": "my-new-agent",
  "triggers": ["pattern1", "pattern2"],
  "tools": ["Read", "Write", "Edit", "Bash", "Grep", "Glob"],
  "auto_activate": null,
  "conventions_required": ["go.md"],
  "description": "Brief description"
}
```

### Step 6: Register in routing-schema.json

Add to `~/.claude/routing-schema.json` → `agent_subagent_mapping`:

```json
{
  "agent_subagent_mapping": {
    "existing-agents": "...",
    "my-new-agent": "general-purpose"
  }
}
```

### Step 7: Verify

```bash
# Check subagent type mapping
jq '.agent_subagent_mapping["my-new-agent"]' ~/.claude/routing-schema.json
# Expected: "general-purpose"

# Check agent config exists
cat ~/.claude/agents/my-new-agent/agent.yaml
```

---

## 4. Detailed Configuration

### agent.yaml Full Schema

```yaml
# REQUIRED FIELDS
name: string                    # Agent identifier (kebab-case)
description: string             # Purpose and usage
model: haiku|sonnet|opus        # Base model

# REASONING (required for sonnet/opus, optional for haiku)
thinking:
  enabled: true|false
  budget: 2000-32000            # Base thinking tokens
  budget_refactor: 14000        # Optional: higher for refactoring
  budget_debug: 18000           # Optional: higher for debugging

# ACTIVATION TRIGGERS
triggers:
  - "keyword1"                  # What phrases trigger this agent
  - "keyword2"

# TOOL PERMISSIONS
tools:
  - Read                        # Read files
  - Write                       # Create new files
  - Edit                        # Modify existing files
  - Bash                        # Run shell commands
  - Glob                        # Find files by pattern
  - Grep                        # Search file contents
  - Task                        # Spawn other agents
  - WebFetch                    # Fetch web content
  - WebSearch                   # Search the web
  - AskUserQuestion             # Ask user for input

# OPTIONAL: Auto-activation (for language-specific agents)
auto_activate:
  languages:
    - Python
    - Go
    - R
  patterns:
    - "**/cmd/**/main.go"
    - "**/*.py"

# OPTIONAL: Required conventions
conventions_required:
  - go.md
  - python.md

# OPTIONAL: Focus areas (documentation for the agent)
focus_areas:
  - "Area 1"
  - "Area 2"

# OPTIONAL: Failure handling
failure_tracking:
  max_attempts: 3
  on_max_reached: "escalate_to_orchestrator"
```

### agent.md Structure

```markdown
# Agent Name

## Purpose
One paragraph explaining what this agent does.

## Domain
- Specific technologies
- Specific problem types

## When to Use
- Trigger scenario 1
- Trigger scenario 2
- Trigger scenario 3

## When NOT to Use
- Anti-pattern 1
- Anti-pattern 2

## Required Behavior

### Tool Selection
Describe which tools to prefer and why.

### Output Format
Describe expected output structure.

### Parallelization
Describe how to handle multiple tasks.

## Failure Handling
What to do when things go wrong.

## Guardrails Checklist
- [ ] Check 1
- [ ] Check 2
- [ ] Check 3

## Examples

### Good Example
```
Input: ...
Output: ...
```

### Bad Example (What NOT to do)
```
Input: ...
Wrong: ...
```
```

### sharp-edges.yaml Structure

```yaml
edges:
  - name: descriptive_name
    severity: critical|high|medium|low
    description: |
      What the problem is and why it matters.
    mitigation: |
      How to avoid or fix the problem.
    detection: |
      How to recognize this is happening.
    example:
      wrong: |
        // Code that triggers the problem
        wrong_pattern()
      right: |
        // Code that avoids the problem
        correct_pattern()

  - name: another_edge
    severity: high
    description: Second pitfall
    mitigation: How to avoid it
```

---

## 5. Tier Selection Guide

### Decision Tree

```
Is the task mechanical (search, format, count)?
├─ YES → HAIKU (Tier 1)
│        thinking: false
│        budget: 0
│
└─ NO → Does it require structured output but simple reasoning?
        ├─ YES → HAIKU + THINKING (Tier 1.5)
        │        thinking: true
        │        budget: 4000-6000
        │
        └─ NO → Does it require implementation, refactoring, or debugging?
                ├─ YES → SONNET (Tier 2-3)
                │        thinking: true
                │        budget: 10000-18000
                │
                └─ NO → Is it deep analysis, security, or architectural?
                        └─ YES → OPUS (Tier 4)
                                 NOTE: Use /einstein, NOT Task()
```

### Tier Mapping

| Task Type | Tier | Model | Thinking | Budget |
|-----------|------|-------|----------|--------|
| File search, grep | 1 | haiku | false | 0 |
| Documentation | 1.5 | haiku | true | 4000 |
| Scaffolding | 1.5 | haiku | true | 4000 |
| Code review | 1.5 | haiku | true | 6000 |
| Implementation | 2 | sonnet | true | 10000 |
| Refactoring | 2.5 | sonnet | true | 14000 |
| Debugging | 3 | sonnet | true | 18000 |
| Deep analysis | 4 | opus | true | 32000 |

---

## 6. Subagent Type Selection

### Quick Reference

| Agent Purpose | Subagent Type | Write Access |
|---------------|---------------|--------------|
| Search files | `Explore` | NO |
| Read code for review | `Explore` | NO |
| Research libraries | `Explore` | NO |
| Write documentation | `general-purpose` | YES |
| Implement features | `general-purpose` | YES |
| Scaffold code | `general-purpose` | YES |
| Plan architecture | `Plan` | YES |
| Coordinate agents | `Plan` | YES |
| Run external tools | `Bash` | NO |

### Choosing the Right Type

```
Does the agent need to modify files?
├─ NO → Does it coordinate other agents?
│       ├─ YES → Plan
│       └─ NO → Does it run external processes?
│               ├─ YES → Bash
│               └─ NO → Explore
│
└─ YES → Does it plan and coordinate?
         ├─ YES → Plan
         └─ NO → general-purpose
```

### Subagent Type Capabilities

#### Explore (Read-Only)
```yaml
tools: [Read, Glob, Grep, Bash]
allows_write: false
# Use for: codebase-search, code-reviewer, librarian
```

#### general-purpose (Full Access)
```yaml
tools: ["*"]  # All tools available
allows_write: true
# Use for: python-pro, scaffolder, tech-docs-writer
```

#### Plan (Coordination)
```yaml
tools: [Read, Glob, Grep, Write, Task, AskUserQuestion]
allows_write: true
# Use for: orchestrator, architect
```

#### Bash (External)
```yaml
tools: [Bash, Read]
allows_write: false
# (deprecated — no agents in this category)
```

---

## 7. Registration & Enforcement

### agents-index.json Entry

```json
{
  "id": "my-agent",
  "parallelization_template": "B",
  "name": "Human-Readable Name",
  "model": "sonnet",
  "thinking": true,
  "thinking_budget": 10000,
  "tier": 2,
  "category": "task|language|architecture|review|memory",
  "path": "my-agent",
  "triggers": ["trigger1", "trigger2"],
  "tools": ["Read", "Write", "Edit"],
  "auto_activate": {
    "languages": ["Python"],
    "patterns": ["**/*.py"]
  },
  "conventions_required": ["python.md"],
  "inputs": [".claude/tmp/some-file"],
  "outputs": [".claude/memory/some-output/"],
  "description": "What this agent does"
}
```

### Parallelization Templates

| Template | Pattern | Use For |
|----------|---------|---------|
| A | Sequential scan | codebase-search |
| B | Parallel execution | scaffolder |
| C | Layer-based | Implementation agents |
| D | Fan-out/fan-in | Orchestrator |
| E | Batch processing | Memory archivist |
| F | Streaming | External processes |

### routing-schema.json Mapping

Add to the `agent_subagent_mapping` section:

```json
{
  "agent_subagent_mapping": {
    "codebase-search": "Explore",
    "python-pro": "general-purpose",
    "orchestrator": "Plan",
    "my-new-agent": "general-purpose"  // ADD THIS
  }
}
```

### Enforcement Mechanism

The `validate-routing.sh` hook enforces subagent type mappings:

```bash
# When you call Task():
Task({
  subagent_type: "general-purpose",  # MUST match routing-schema.json
  model: "sonnet",
  prompt: "AGENT: my-new-agent\n\n..."
})

# If mismatch:
# ❌ BLOCKED: "my-new-agent requires general-purpose, got Explore"
```

---

## 8. Testing Your Agent

### Step 1: Verify Configuration

```bash
# Check agent.yaml is valid
cat ~/.claude/agents/my-new-agent/agent.yaml

# Check routing-schema.json mapping
jq '.agent_subagent_mapping["my-new-agent"]' ~/.claude/routing-schema.json
```

### Step 2: Test Invocation

In a Claude session:

```javascript
// Test direct invocation
Task({
  description: "Test my-new-agent",
  subagent_type: "general-purpose",  // Must match routing-schema.json
  model: "sonnet",
  prompt: `AGENT: my-new-agent

1. TASK: Perform a simple test task
2. EXPECTED OUTCOME: Verification that agent works
3. REQUIRED TOOLS: Read
4. MUST DO: Report success
5. MUST NOT DO: Make actual changes
6. CONTEXT: Testing new agent installation`
})
```

### Step 3: Check for Violations

```bash
# View routing violations
cat /tmp/claude-routing-violations.jsonl

# Check for errors
tail -20 /tmp/claude-routing-violations.jsonl | jq .
```

### Step 4: Test with Force Override

If blocked, test with override:

```javascript
Task({
  description: "Test with override",
  subagent_type: "general-purpose",
  model: "sonnet",
  prompt: `AGENT: my-new-agent --force-delegation=sonnet

  ... task details ...`
})
```

---

## 9. Common Patterns

### Pattern 1: Language Implementation Agent

```yaml
# Example: rust-pro
name: rust-pro
description: Expert Rust development with ownership patterns
model: sonnet
thinking:
  enabled: true
  budget: 10000
  budget_debug: 18000
auto_activate:
  languages: [Rust]
triggers: [implement, refactor, cargo, "rust code"]
tools: [Read, Write, Edit, Bash, Grep, Glob]
conventions_required: [rust.md]
failure_tracking:
  max_attempts: 3
  on_max_reached: escalate_to_orchestrator
```

Registration:
```json
// agents-index.json
{"id": "rust-pro", "model": "sonnet", "tier": 2, ...}

// routing-schema.json
"rust-pro": "general-purpose"
```

### Pattern 2: Read-Only Analysis Agent

```yaml
# Example: security-scanner
name: security-scanner
description: Scans code for security vulnerabilities
model: sonnet
thinking:
  enabled: true
  budget: 14000
triggers: [security, vulnerability, audit, CVE]
tools: [Read, Grep, Glob]  # NO Write/Edit
conventions_required: []
```

Registration:
```json
// agents-index.json
{"id": "security-scanner", "model": "sonnet", "tier": 2, ...}

// routing-schema.json
"security-scanner": "Explore"  // Read-only
```

### Pattern 3: External Integration Agent

```yaml
# Example: openai-bridge
name: openai-bridge
description: Bridges to OpenAI API for specific tasks
model: haiku
thinking: false
triggers: [openai, gpt-4, dalle]
tools: [Bash, Read]
```

Registration:
```json
// routing-schema.json
"openai-bridge": "Bash"
```

### Pattern 4: Documentation Specialist

```yaml
# Example: api-docs-writer
name: api-docs-writer
description: Generates OpenAPI/Swagger documentation
model: haiku
thinking:
  enabled: true
  budget: 6000
triggers: [openapi, swagger, api docs, endpoints]
tools: [Read, Write, Edit, Glob, Grep]
conventions_required: [api-docs.md]
```

Registration:
```json
// routing-schema.json
"api-docs-writer": "general-purpose"  // Needs Write access
```

---

## 10. Troubleshooting

### Error: "Invalid subagent_type for agent X"

**Cause:** Mismatch between Task() call and routing-schema.json

**Fix:**
```bash
# Check correct subagent_type
jq '.agent_subagent_mapping["my-agent"]' ~/.claude/routing-schema.json

# Use the correct type in Task()
Task({subagent_type: "correct-type", ...})
```

### Error: "Agent not found"

**Cause:** Agent not registered in agents-index.json

**Fix:**
1. Add entry to `~/.claude/agents/agents-index.json`
2. Verify agent.yaml exists at `~/.claude/agents/{name}/agent.yaml`

### Error: "Delegation ceiling exceeded"

**Cause:** Trying to spawn agent above current ceiling

**Fix:**
```javascript
// Add override to prompt
Task({
  prompt: "AGENT: my-agent --force-delegation=sonnet\n\n..."
})
```

### Error: Agent has no tools

**Cause:** Tools not listed in agent.yaml or using wrong subagent_type

**Fix:**
1. Check tools array in agent.yaml
2. Verify subagent_type allows those tools:
   - `Explore` → Read, Glob, Grep, Bash only
   - `general-purpose` → All tools
   - `Plan` → Read, Glob, Grep, Write, Task, AskUserQuestion
   - `Bash` → Bash, Read only

### Agent Not Auto-Activating

**Cause:** `auto_activate` not configured or patterns don't match

**Fix:**
```yaml
auto_activate:
  languages:
    - Python  # Must match exactly
  patterns:
    - "**/*.py"  # Glob patterns
```

### Sharp Edge Not Captured

**Cause:** sharp-edges.yaml not being read

**Fix:**
1. Verify file exists: `~/.claude/agents/{name}/sharp-edges.yaml`
2. Check YAML syntax is valid
3. Restart session to reload

---

## 11. Checklist

### Before Creating

- [ ] Identified the tier (haiku/haiku+thinking/sonnet/opus)
- [ ] Chose subagent_type (Explore/general-purpose/Plan/Bash)
- [ ] Listed required tools
- [ ] Identified trigger patterns
- [ ] Determined if auto-activation needed

### File Creation

- [ ] Created `~/.claude/agents/{name}/` directory
- [ ] Created `agent.yaml` with all required fields
- [ ] Created `agent.md` with documentation
- [ ] Created `sharp-edges.yaml` (can be empty initially)

### Registration

- [ ] Added entry to `~/.claude/agents/agents-index.json`
- [ ] Added mapping to `~/.claude/routing-schema.json` → `agent_subagent_mapping`
- [ ] Verified with: `jq '.agent_subagent_mapping["{name}"]' ~/.claude/routing-schema.json`

### Testing

- [ ] Tested Task() invocation with correct subagent_type
- [ ] Verified no routing violations in `/tmp/claude-routing-violations.jsonl`
- [ ] Tested trigger patterns activate agent
- [ ] Tested auto-activation (if configured)
- [ ] Tested failure handling escalates correctly

### Documentation

- [ ] Documented in agent.md when to use
- [ ] Documented in agent.md when NOT to use
- [ ] Added to project documentation if project-specific

---

## Quick Reference Card

```
┌─────────────────────────────────────────────────────────────────┐
│                  NEW AGENT QUICK REFERENCE                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. mkdir ~/.claude/agents/{name}                              │
│                                                                 │
│  2. Create agent.yaml:                                         │
│     name: {name}                                                │
│     model: haiku|sonnet                                        │
│     thinking: {enabled: true, budget: NNNN}                    │
│     triggers: [...]                                            │
│     tools: [...]                                               │
│                                                                 │
│  3. Create agent.md (documentation)                            │
│                                                                 │
│  4. Create sharp-edges.yaml (edges: [])                        │
│                                                                 │
│  5. Add to agents-index.json:                                  │
│     {"id": "{name}", "model": "...", ...}                      │
│                                                                 │
│  6. Add to routing-schema.json → agent_subagent_mapping:       │
│     "{name}": "general-purpose|Explore|Plan|Bash"              │
│                                                                 │
│  7. Test: Task({subagent_type: "...", prompt: "AGENT: ..."})  │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│  SUBAGENT TYPE CHEAT SHEET:                                    │
│  • Explore       → Read-only (search, review)                  │
│  • general-purpose → Full access (implement, docs)              │
│  • Plan          → Coordination (orchestrator)                 │
│  • Bash          → External processes (deprecated)             │
└─────────────────────────────────────────────────────────────────┘
```

---

**Document Version:** 1.0
**Last Updated:** 2026-01-25
**Related Documents:**
- `INSTALL-GUIDE.md` - Framework installation
- `~/.claude/routing-schema.json` - Tier and routing definitions
- `~/.claude/agents/agents-index.json` - Agent registry
