# Agent Reference Table

Complete reference for all Claude Code agents, their subagent_types, models, and usage.

## Quick Lookup by Subagent Type

### Explore (Read-Only)
Read-only codebase exploration and analysis. No write permissions.

| Agent | Model | Domain | When to Use | If Wrong |
|-------|-------|--------|-------------|----------|
| `codebase-search` | Haiku | File/code discovery | Finding files, grep patterns, "where is X" | Will error: "requires Explore, not general-purpose" |
| `haiku-scout` | Haiku | Scope assessment | Pre-route reconnaissance for unknown scope | Will error: "requires Explore" |
| `code-reviewer` | Haiku+Think | Code Review | Review style, simple bugs, "review this" | Will error: "requires Explore" |
| `librarian` | Haiku+Think | External research | Library docs, best practices, "how do I use X" | Will error: "requires Explore" |

### general-purpose (Full Write)
Full tool access for implementation and documentation. Respects agent.yaml definitions.

| Agent | Model | Domain | When to Use | If Wrong |
|-------|-------|--------|-------------|----------|
| `scaffolder` | Haiku+Think | Boilerplate | New class, new module, scaffold, template | Will error: "requires general-purpose, not Explore" |
| `tech-docs-writer` | Haiku+Think | Documentation | README, guides, API docs, system-guide | Will error: "requires general-purpose, not Explore" |
| `python-pro` | Sonnet+Think | Python implementation | Any Python code writing | Will error: "requires general-purpose, not Plan" |
| `python-ux` | Sonnet+Think | PySide6/Qt | GUI code, widgets, Qt patterns | Will error: "requires general-purpose" |
| `r-pro` | Sonnet+Think | R implementation | Any R code writing | Will error: "requires general-purpose" |
| `r-shiny-pro` | Sonnet+Think | Shiny apps | Modules, reactives, Shiny-specific | Will error: "requires general-purpose" |
| `memory-archivist` | Haiku+Think | Memory management | Archive session learnings | Will error: "requires general-purpose" |
| `staff-architect-critical-review` | Sonnet+Think (16K) | Plan review | Critical review of implementation plans using 7-layer framework. Manual invocation via /review-plan. Spawns scouts for verification. $0.15-0.17 per review. | Will error: "requires general-purpose" |
| `einstein` | Opus+Think | Deep Analysis | Last resort, complex reasoning, intractable problems | Will error: "requires general-purpose" |

### Plan (Coordination)
Architecture and planning mode with controlled write access. Can spawn other agents.

| Agent | Model | Domain | When to Use | If Wrong |
|-------|-------|--------|-------------|----------|
| `orchestrator` | Sonnet+Think | Architecture & Planning | Ambiguous scope, interviews, debug coordination | Will error: "requires Plan, not general-purpose" |
| `architect` | Sonnet+Think | Implementation Planning | Detailed multi-file plans, dependency analysis | Will error: "requires Plan, not general-purpose" |

### Bash (External Processes)
Command execution specialist for external processes. Shell piping only.

| Agent | Model | Domain | When to Use | If Wrong |
|-------|-------|--------|-------------|----------|
| `gemini-slave` | Gemini 2.0 Flash | Large-context analysis | Multiple files, cross-module, 1M+ token context | Will error: "requires Bash, not Explore" |

## Complete Mapping Reference

Use this table to quickly find the correct subagent_type for any agent.

```json
{
  "Explore": ["codebase-search", "haiku-scout", "code-reviewer", "librarian"],
  "general-purpose": ["scaffolder", "tech-docs-writer", "python-pro", "python-ux", "r-pro", "r-shiny-pro", "memory-archivist", "staff-architect-critical-review", "einstein"],
  "Plan": ["orchestrator", "architect"],
  "Bash": ["gemini-slave"]
}
```

## Agent Details

### codebase-search

- **Tier**: Haiku (read-only)
- **Subagent Type**: `Explore`
- **Model**: Haiku
- **Domain**: File/code discovery
- **When to use**: Finding files, grep patterns, "where is X", "find all occurrences"
- **Example**:
  ```javascript
  Task({
    description: "Find all authentication files",
    subagent_type: "Explore",
    model: "haiku",
    prompt: "AGENT: codebase-search\n\n1. TASK: Find auth files..."
  })
  ```
- **If you use wrong subagent_type**: Will fail with "codebase-search requires Explore"

### haiku-scout

- **Tier**: Haiku (read-only)
- **Subagent Type**: `Explore`
- **Model**: Haiku
- **Domain**: Scope assessment
- **When to use**: Pre-routing reconnaissance, assessing refactoring scope, "how big is this task"
- **Example**:
  ```javascript
  Task({
    description: "Scout src/ for refactoring scope",
    subagent_type: "Explore",
    model: "haiku",
    prompt: "AGENT: haiku-scout\n\nSCOUT TARGET: src/"
  })
  ```
- **If you use wrong subagent_type**: Will fail with "haiku-scout requires Explore"

### code-reviewer

- **Tier**: Haiku+Thinking (read-only)
- **Subagent Type**: `Explore`
- **Model**: Haiku with thinking
- **Domain**: Code Review
- **When to use**: Review style, simple bugs, "review this code against conventions"
- **Example**:
  ```javascript
  Task({
    description: "Review Python code for style violations",
    subagent_type: "Explore",
    model: "haiku",
    prompt: "AGENT: code-reviewer\n\n1. TASK: Review this against python.md..."
  })
  ```
- **If you use wrong subagent_type**: Will fail with "code-reviewer requires Explore"

### librarian

- **Tier**: Haiku+Thinking (read-only)
- **Subagent Type**: `Explore`
- **Model**: Haiku with thinking
- **Domain**: External research
- **When to use**: Library docs, best practices, "how do I use X", external APIs
- **Example**:
  ```javascript
  Task({
    description: "Research PyTorch best practices",
    subagent_type: "Explore",
    model: "haiku",
    prompt: "AGENT: librarian\n\n1. TASK: Find PyTorch best practices..."
  })
  ```
- **If you use wrong subagent_type**: Will fail with "librarian requires Explore"

### tech-docs-writer

- **Tier**: Haiku+Thinking (write access)
- **Subagent Type**: `general-purpose`
- **Model**: Haiku with thinking
- **Domain**: Documentation
- **When to use**: README, guides, API docs, system-guide updates
- **Example**:
  ```javascript
  Task({
    description: "Update API documentation",
    subagent_type: "general-purpose",
    model: "haiku",
    prompt: "AGENT: tech-docs-writer\n\n1. TASK: Update docs/api.md..."
  })
  ```
- **If you use wrong subagent_type**: Will fail with "tech-docs-writer requires general-purpose"

### scaffolder

- **Tier**: Haiku+Thinking (write access)
- **Subagent Type**: `general-purpose`
- **Model**: Haiku with thinking
- **Domain**: Boilerplate
- **When to use**: New class, new module, scaffold, template generation
- **Example**:
  ```javascript
  Task({
    description: "Scaffold new React component",
    subagent_type: "general-purpose",
    model: "haiku",
    prompt: "AGENT: scaffolder\n\n1. TASK: Create new component..."
  })
  ```
- **If you use wrong subagent_type**: Will fail with "scaffolder requires general-purpose"

### python-pro

- **Tier**: Sonnet+Thinking (write access)
- **Subagent Type**: `general-purpose`
- **Model**: Sonnet with thinking
- **Domain**: Python implementation
- **When to use**: Any Python code writing, refactoring, debugging
- **Example**:
  ```javascript
  Task({
    description: "Implement data pipeline",
    subagent_type: "general-purpose",
    model: "sonnet",
    prompt: "AGENT: python-pro\n\n1. TASK: Build data pipeline..."
  })
  ```
- **If you use wrong subagent_type**: Will fail with "python-pro requires general-purpose"

### python-ux

- **Tier**: Sonnet+Thinking (write access)
- **Subagent Type**: `general-purpose`
- **Model**: Sonnet with thinking
- **Domain**: PySide6/Qt GUI
- **When to use**: GUI code, widgets, Qt patterns, PySide6 implementation
- **Example**:
  ```javascript
  Task({
    description: "Build PySide6 dialog",
    subagent_type: "general-purpose",
    model: "sonnet",
    prompt: "AGENT: python-ux\n\n1. TASK: Create dialog widget..."
  })
  ```
- **If you use wrong subagent_type**: Will fail with "python-ux requires general-purpose"

### r-pro

- **Tier**: Sonnet+Thinking (write access)
- **Subagent Type**: `general-purpose`
- **Model**: Sonnet with thinking
- **Domain**: R implementation
- **When to use**: Any R code writing, refactoring, debugging
- **Example**:
  ```javascript
  Task({
    description: "Implement statistical analysis",
    subagent_type: "general-purpose",
    model: "sonnet",
    prompt: "AGENT: r-pro\n\n1. TASK: Build analysis pipeline..."
  })
  ```
- **If you use wrong subagent_type**: Will fail with "r-pro requires general-purpose"

### r-shiny-pro

- **Tier**: Sonnet+Thinking (write access)
- **Subagent Type**: `general-purpose`
- **Model**: Sonnet with thinking
- **Domain**: Shiny apps
- **When to use**: Modules, reactives, Shiny-specific code, interactive apps
- **Example**:
  ```javascript
  Task({
    description: "Build Shiny reactive module",
    subagent_type: "general-purpose",
    model: "sonnet",
    prompt: "AGENT: r-shiny-pro\n\n1. TASK: Create reactive module..."
  })
  ```
- **If you use wrong subagent_type**: Will fail with "r-shiny-pro requires general-purpose"

### orchestrator

- **Tier**: Sonnet+Thinking (coordination)
- **Subagent Type**: `Plan`
- **Model**: Sonnet with thinking
- **Domain**: Architecture & Planning
- **When to use**: Ambiguous scope, user interviews, debug coordination, multi-agent workflows
- **Example**:
  ```javascript
  Task({
    description: "Plan refactoring strategy",
    subagent_type: "Plan",
    model: "sonnet",
    prompt: "AGENT: orchestrator\n\n1. TASK: Interview user and plan..."
  })
  ```
- **If you use wrong subagent_type**: Will fail with "orchestrator requires Plan"

### architect

- **Tier**: Sonnet+Thinking (coordination)
- **Subagent Type**: `Plan`
- **Model**: Sonnet with thinking
- **Domain**: Implementation Planning
- **When to use**: Detailed multi-file plans, dependency analysis, design reviews
- **Example**:
  ```javascript
  Task({
    description: "Create implementation roadmap",
    subagent_type: "Plan",
    model: "sonnet",
    prompt: "AGENT: architect\n\n1. TASK: Build detailed implementation plan..."
  })
  ```
- **If you use wrong subagent_type**: Will fail with "architect requires Plan"

### einstein

- **Tier**: Opus+Thinking (write access)
- **Subagent Type**: `general-purpose`
- **Model**: Opus with thinking
- **Domain**: Deep Analysis
- **When to use**: Last resort, complex reasoning, intractable problems, security reviews
- **Example**:
  ```javascript
  Task({
    description: "Deep security audit",
    subagent_type: "general-purpose",
    model: "opus",
    prompt: "AGENT: einstein\n\n1. TASK: Security audit..."
  })
  ```
- **If you use wrong subagent_type**: Will fail with "einstein requires general-purpose"

### gemini-slave

- **Tier**: External (bash execution only)
- **Subagent Type**: `Bash`
- **Model**: Gemini 2.0 Flash
- **Domain**: Large-context analysis
- **When to use**: Multiple files, cross-module, 1M+ token context, architectural review
- **Note**: Uses Bash, NOT Task tool. Invoke via piping:
  ```bash
  FILES=$(find src/ -name "*.py")
  cat $FILES | gemini-slave mapper "Find core logic"
  ```
- **If you use wrong subagent_type**: Will fail with "gemini-slave requires Bash"

### staff-architect-critical-review

- **Tier**: Sonnet+Thinking (write access, 16K budget)
- **Subagent Type**: `general-purpose`
- **Model**: Sonnet with thinking
- **Domain**: Plan review and critical analysis
- **When to use**: Critical review of implementation plans before committing to implementation, post-/explore validation, second opinion on complex architectural decisions
- **Invocation**: Manual via `/review-plan` slash command (NOT Task tool)
  ```bash
  /review-plan                    # Review default specs.md
  /review-plan path/to/plan.md    # Review custom plan file
  ```
- **Cost**: $0.15-0.17 per review
- **Duration**: 2-3 minutes
- **Scout spawns**: 0-2 (for verification)
- **7-Layer Framework**:
  1. Assumption Register (extracts implicit assumptions, assesses risk)
  2. Dependency Mapping (circular dependencies, missing deps)
  3. Failure Mode Analysis (partial failure scenarios)
  4. Cost-Benefit Assessment (complexity justified)
  5. Testing Coverage (critical paths tested)
  6. Architecture Smell Detection (God Components, coupling)
  7. Contractor Readiness (mid-level engineer can implement)
- **Output**: Verdict (APPROVE, APPROVE_WITH_CONDITIONS, CONCERNS, CRITICAL_ISSUES) + structured findings by severity
- **If you use wrong subagent_type**: Will fail with "staff-architect-critical-review requires general-purpose"

### memory-archivist

- **Tier**: Haiku+Thinking (write access)
- **Subagent Type**: `general-purpose`
- **Model**: Haiku with thinking
- **Domain**: Memory management
- **When to use**: Archive session learnings, save decisions to memory
- **Example**:
  ```javascript
  Task({
    description: "Archive session learnings",
    subagent_type: "general-purpose",
    model: "haiku",
    prompt: "AGENT: memory-archivist\n\n1. TASK: Archive this session..."
  })
  ```
- **If you use wrong subagent_type**: Will fail with "memory-archivist requires general-purpose"

## Model Tier Reference

| Model | Cost | Thinking | Best For |
|-------|------|----------|----------|
| Haiku | $0.0005/1k | Optional | Mechanical work, file search, counting |
| Haiku+Thinking | $0.001/1k | 6K budget | Documentation, scaffolding, simple reasoning |
| Sonnet | $0.009/1k | 16K budget | Implementation, refactoring, multi-file work |
| Sonnet+Thinking | $0.009/1k | 16K budget | Planning, architecture, synthesis |
| Opus | $0.045/1k | 32K budget | Deep analysis, security, novel problems |
| Gemini 2.0 Flash | $0.0001/1k | None | Large context (1M tokens), codebase analysis |

## Tool Permissions by Subagent Type

| Subagent Type | Tools | Write | Reason |
|---|---|---|---|
| `Explore` | Read, Glob, Grep, Bash | NO | Reconnaissance only |
| `general-purpose` | All (per agent.yaml) | YES | Implementation and docs |
| `Plan` | Read, Glob, Grep, Write, Task, Ask | YES | Planning and coordination |
| `Bash` | Bash, Read | NO | External process execution only |

## Verification Commands

### Check any agent's subagent_type

```bash
jq '.agent_subagent_mapping["agent-name"]' ~/.claude/routing-schema.json
```

### List all agents by subagent_type

```bash
jq 'to_entries | group_by(.value) | map({(.[0].value): map(.key)}) | add' \
  ~/.claude/routing-schema.json
```

### Check subagent_type tools

```bash
jq '.subagent_types["Explore"]' ~/.claude/routing-schema.json
```

## Related Documentation

- **Routing Enforcement**: `/home/doktersmol/.claude/docs/architecture/routing-enforcement.md`
- **Hook Documentation**: `/home/doktersmol/.claude/docs/hooks/validate-routing.md`
- **Main Guide**: `/home/doktersmol/.claude/CLAUDE.md` (Agent Routing section)
- **Routing Schema**: `/home/doktersmol/.claude/routing-schema.json`

## FAQ

**Q: How do I know if I'm using the right subagent_type?**
A: Find your agent in this table and check the "Subagent Type" column.

**Q: What's the difference between subagent_type and model?**
A: Model (haiku, sonnet, opus) is the LLM size. Subagent_type is the tool permissions. You need both.

**Q: Can two agents have the same subagent_type?**
A: Yes, many do. For example, both `scaffolder` and `tech-docs-writer` use `general-purpose`.

**Q: What if my agent isn't listed here?**
A: Either it's new (check `/home/doktersmol/.claude/routing-schema.json`), or it's custom (check `~/.claude/agents/[name]/agent.yaml`).
