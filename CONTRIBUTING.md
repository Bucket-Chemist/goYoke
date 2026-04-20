# Contributing to goYoke

goYoke welcomes contributions — especially new agents, team configurations, and conventions that benefit the broader Claude Code community.

## What You Can Contribute

### Agents

An agent is a Claude Code subagent with a defined role, model tier, trigger patterns, and optional sharp edge documentation. Each agent lives in its own directory:

```
agents/
  my-agent/
    my-agent.md         # Identity prompt (YAML frontmatter + markdown body)
    sharp-edges.yaml    # Known failure patterns and mitigations (optional)
```

**Agent frontmatter:**

```yaml
---
name: My Agent
description: One-line description of what this agent does
model: sonnet          # haiku, sonnet, or opus
tier: 2                # 1 (haiku), 1.5 (haiku+thinking), 2 (sonnet), 3 (opus)
category: implementation
triggers:
  - "my domain keyword"
  - "another trigger phrase"
tools:
  - Read
  - Write
  - Edit
  - Bash
  - Glob
  - Grep
spawned_by:
  - router
---

# My Agent

You are a specialist in [domain]. Your role is to [what you do].

## Guidelines

- [Domain-specific instructions]
```

**To add an agent:**

1. Create the agent directory under `agents/`
2. Write the identity markdown with frontmatter
3. Add the agent entry to `agents-index.json` with an `"id"`, `"name"`, `"model"`, `"tier"`, `"triggers"`, and `"tools"` fields
4. Optionally add `sharp-edges.yaml` documenting known failure patterns
5. Open a PR

### Team Configurations

A team configuration defines a reproducible multi-agent workflow with typed stdin/stdout contracts. Teams are the mechanism for complex workflows like code review, deep analysis, and parallel implementation.

**Team directory structure:**

```
teams/
  my-workflow/
    config.json                    # Team topology (waves, members, budget)
    stdin/
      wave1-agent-a.json           # Typed input for agent A
      wave1-agent-b.json           # Typed input for agent B
      wave2-synthesizer.json       # Input for Wave 2 (receives Wave 1 output)
```

**config.json format:**

```json
{
  "team_name": "my-workflow",
  "workflow_type": "my-workflow",
  "budget_max_usd": 3.00,
  "waves": [
    {
      "wave_number": 1,
      "description": "Parallel analysis",
      "members": [
        {"name": "wave1-agent-a", "agent": "my-agent-a", "model": "sonnet", "status": "pending"},
        {"name": "wave1-agent-b", "agent": "my-agent-b", "model": "sonnet", "status": "pending"}
      ]
    },
    {
      "wave_number": 2,
      "description": "Synthesis",
      "members": [
        {"name": "wave2-synthesizer", "agent": "my-synthesizer", "model": "opus", "status": "pending"}
      ]
    }
  ]
}
```

**Stdin/stdout schemas** (optional but recommended) go in `schemas/teams/stdin-stdout/`:

```json
{
  "$comment": "Contract for my-workflow agents",
  "stdin": {
    "task": "string — what to analyze",
    "context": "string — relevant codebase context"
  },
  "stdout": {
    "status": "string — completed|failed",
    "summary": "string — human-readable summary",
    "findings": "array — structured results"
  }
}
```

Schemas ensure reproducibility — the same team config with the same stdin produces structurally consistent output regardless of which LLM instance runs it.

**To add a team workflow:**

1. Create the team directory with config.json and stdin/ files
2. Create the stdin-stdout schema in `schemas/teams/stdin-stdout/`
3. Ensure all referenced agents exist (or include them in your PR)
4. Document the workflow in your PR description
5. Open a PR

### Conventions

Conventions are language-specific coding guidelines auto-loaded based on file context. They live in `conventions/`:

```
conventions/
  my-language.md    # Coding conventions for a language or framework
```

Conventions are markdown files that get injected into agent prompts when working with matching file types. If you're adding conventions for a new language, also update the convention auto-loading patterns in the routing config.

### Skills

Skills are slash-command workflows defined in `skills/`:

```
skills/
  my-skill/
    SKILL.md    # Skill definition with YAML frontmatter
```

Skills define multi-step workflows that the router executes. They're invoked via `/my-skill` in the TUI.

## Development Setup

```bash
git clone https://github.com/Bucket-Chemist/goYoke.git
cd goYoke
make dist           # Generate defaults + build all binaries
make test           # Run test suite
make test-defaults  # Validate distribution content
```

## Pull Request Guidelines

- **One concern per PR** — a new agent, a new team config, or a convention update
- **Include tests** if your contribution includes Go code changes
- **Test your agent** by running it through the TUI before submitting
- **Document sharp edges** — if your agent has known failure patterns, add a `sharp-edges.yaml`
- **Respect the tier system** — don't use Opus for tasks that Sonnet handles. Cost matters.

## Code Style

Go code follows the conventions in `conventions/go.md`. Key points:

- Explicit error handling (no panics in library code)
- Table-driven tests
- Small interfaces, composition over inheritance
- `go vet` and `go test -race` must pass

## Questions?

Open an issue or start a discussion. We're happy to help you figure out the right agent tier, team topology, or convention structure for your contribution.

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
