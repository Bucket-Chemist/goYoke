# goYoke Workflows Guide

A practical guide to using goYoke for day-to-day Claude Code work.

## What is goYoke?

goYoke enforces sane development workflows through compiled Go binaries that intercept Claude Code at runtime—not through prompt instructions that degrade over long conversations. It combines programmatic hooks, multi-tier agent routing, and a terminal UI to stop vibe coding and produce deliverables instead of chat.

**Why it exists:** Claude is powerful but easy to misuse. You can jump into implementation without a plan, lose context in long conversations, burn expensive Opus tokens on file searches, or end up with untested code. goYoke prevents these mistakes by making each step explicit: scout the problem, create a plan, get it reviewed, then implement.

**How it works:** When you describe a task, the router (an Opus-tier Claude instance) classifies your request and delegates to one of 78 specialized agents at the appropriate tier (Haiku for search, Sonnet for implementation, Opus for design). Each agent gets the same conventions, rules, and identity injected automatically. Every decision is logged. Sharp edges are captured. Sessions are archived with handoffs.

## Core Concepts

### Router
The entry point to goYoke. It receives your request, decides what type of work it is, and delegates to the right agent. The router **never implements code directly**—it classifies and coordinates.

### Agents
Specialized Claude instances trained on specific domains. There are 78 agents across 4 tiers:
- **Haiku**: File search, scope assessment, lightweight tasks ($0.0005/1K tokens)
- **Haiku+Thinking**: Scaffolding, docs, simple reviews (~$0.001/1K tokens)
- **Sonnet**: Implementation, refactoring, tests ($0.009/1K tokens)
- **Opus**: Architecture, planning, deep analysis ($0.045/1K tokens)

Agents auto-load language-specific conventions and operate atomically—one invocation, one deliverable.

### Hooks
Go binaries that intercept Claude Code events (SessionStart, PreToolUse, PostToolUse, SubagentStop, SessionEnd) to enforce behavior. They validate routing, track failures, warn on anti-patterns, and collect telemetry. No hook bypasses are allowed.

### Skills
Slash commands that orchestrate multi-agent workflows. They combine scouts, planners, architects, and reviewers into reusable pipelines. For example, `/explore` runs a scout for reconnaissance and an architect for analysis. `/braintrust` invokes Mozart to interview you, then Einstein and Staff-Architect in parallel for perspective, then Beethoven to synthesize findings.

### Conventions
Language-specific coding standards (Go, Python, React, Rust, TypeScript, R) auto-loaded based on file context. Every agent sees the same conventions, so output is consistent.

### Sharp Edges
Known failure patterns captured across sessions. If an agent fails 3+ times on the same problem, execution stops and you're notified to escalate to `/braintrust` for deep analysis.

## Typical Workflows

### Small Bug Fix
Just describe it. The router handles trivial fixes directly or delegates to the language-specific agent (go-pro, python-pro, react-pro, etc.).

```
User: "The login button on the dashboard doesn't work"
Router: Delegates to go-pro or react-pro depending on file context
Agent: Implements the fix directly
```

### Feature Implementation (Full Pipeline)

For substantial work, use the full planning → review → implement pipeline:

```bash
/explore "add user authentication"     # Scout + architect: understand scope, create specs
/review-plan                           # Staff architect: review plan for soundness
/refine-plan                           # Harmonizer: incorporate review feedback
/implement                             # Architect plans, workers execute in background
```

Output: Tested, reviewed code ready to merge.

### Complex Design Decision

When you're unsure about the approach:

```bash
/braintrust "should we cache in Redis or use a database?"
```

Output: Multi-perspective analysis with recommendation.

Mozart interviews you for context, Einstein analyzes from first principles, Staff-Architect adds practical constraints, Beethoven synthesizes a recommendation.

### Code Review Before Merge

```bash
/review
```

Four specialist reviewers (backend, frontend, standards, architecture) examine the code in parallel, then synthesize findings. Output: severity-grouped findings with approval recommendation.

### Planning a Feature from Scratch

```bash
/plan-tickets "build a notification system"
```

Full pipeline: Scout → Planner → Architect → Staff Reviewer → [optional] Harmonizer → Ticket generation.

Output: Jira-ready tickets with dependency graphs and effort estimates.

### Codebase Exploration (Unknown Scope)

When you need to understand before acting:

```bash
/explore "what logging do we have across the system?"
```

Scout reconnaissance finds the relevant files. Architect synthesizes findings. Output: structured analysis, not chat.

## Skills Quick Reference

| Command | What It Does | Best For |
|---------|-------------|----------|
| `/explore` | Scout + architect analysis | Unknown scope, need to understand before acting |
| `/braintrust` | Mozart → Einstein + Staff-Architect → Beethoven | Complex design decisions, intractable problems |
| `/review` | 4-domain code review + synthesis | Code quality gates, pre-merge validation |
| `/review-plan` | Staff architect critical review | After creating a plan, before implementing |
| `/refine-plan` | Incorporate review feedback | Automate plan fixes from review findings |
| `/implement` | Architect plans, workers execute | Ready to build, plan is approved |
| `/plan-tickets` | Full planning pipeline | Starting a feature from scratch |
| `/ticket` | Ticket-driven implementation | Working through a backlog |
| `/cleanup` | 8-domain code hygiene | Technical debt, duplication, dead code |
| `/codebase-map` | Generate architecture documentation | Onboarding, understanding structure |
| `/teams` | List running teams | Monitor background work |
| `/team-status` | Detailed team progress | Check background implementation status |
| `/team-result` | Collect completed work | Get final output from background work |
| `/team-cancel` | Stop a running team | Abort background work |

## Agent Tiers Explained

**Use the right tier for the job.** Haiku saves money on search. Sonnet handles reasoning. Opus handles architecture. Mixing tiers wrong wastes tokens.

| Tier | Model | Cost | Used For | Example Agents |
|------|-------|------|----------|----------------|
| Haiku | Claude 3.5 Haiku | $0.0005/1K tokens | File search, scope assessment, counting lines | `codebase-search`, `haiku-scout` |
| Haiku+Thinking | Haiku with thinking | ~$0.001/1K tokens | Scaffolding, docs, simple code review | `scaffolder`, `tech-docs-writer`, `code-reviewer` |
| Sonnet | Claude 3.5 Sonnet | $0.009/1K tokens | Implementation, refactoring, testing, CLI work, TUI work | `go-pro`, `python-pro`, `react-pro`, `go-cli`, `go-tui` |
| Opus | Claude 4 Opus | $0.045/1K tokens | Architecture decisions, planning, deep analysis, synthesis | `architect`, `planner`, `staff-architect`, `einstein` |

**Rule of thumb:** Scout with Haiku before committing Opus time. A $0.02 scout can prevent a $0.50 mis-routing.

## How Enforcement Works

goYoke uses Go binaries (not prompts) to enforce behavior:

- **goyoke-validate** blocks wrong-tier delegation. Task(opus) is blocked unless it's an allowlisted planning agent. Use `/braintrust` instead.
- **goyoke-sharp-edge** tracks failures. After 3+ consecutive failures on the same problem, execution stops and you're notified to escalate.
- **goyoke-direct-impl-check** warns when the router tries to code directly. Routing is about classification, not implementation.
- **goyoke-orchestrator-guard** ensures background tasks are collected. If you spawn background work, you must collect results before synthesis.
- **goyoke-load-context** injects conventions and restores session state at session start. This is why agents stay consistent.

These aren't suggestions—they're enforced in binaries before Claude even sees the request.

## Customization

### Add a Custom Agent

Create `.claude/agents/my-agent/agent.yaml`:

```yaml
agent_id: my-agent
name: My Custom Agent
model: sonnet
trigger_patterns:
  - "my domain pattern"
  - "my keywords"
description: What this agent does
```

Deploy with `goyoke install --agent my-agent`.

### Add Custom Conventions

Create `.claude/conventions/my-lang.md` with coding standards for your language. goYoke auto-loads it based on file context.

### Add a Custom Skill

Use `/explore-add` to scaffold a new skill that orchestrates multiple agents into a reusable workflow.

## Next Steps

- **First time?** Run `goyoke` to start the TUI.
- **Learn by example?** Try `/explore` on an unknown module.
- **Deep dive?** See [ARCHITECTURE.md](ARCHITECTURE.md) for system design.
- **Setting up your project?** See [INSTALL-GUIDE.md](INSTALL-GUIDE.md).
