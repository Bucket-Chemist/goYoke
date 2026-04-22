# goYoke Workflows Guide

A practical guide to using goYoke for day-to-day Claude Code work.

## What is goYoke?

goYoke enforces sane development workflows through compiled Go binaries that intercept Claude Code at runtime—not through prompt instructions that degrade over long conversations. It combines programmatic hooks, multi-tier agent routing, and a terminal UI to stop vibe coding and produce deliverables instead of chat.

**Why it exists:** Claude is powerful but easy to misuse. You can jump into implementation without a plan, lose context in long conversations, burn expensive Opus tokens on file searches, or end up with untested code. goYoke prevents these mistakes by making each step explicit: scout the problem, create a plan, get it reviewed, then implement.

**How it works:** When you describe a task, the router (an Opus-tier Claude instance) classifies your request and delegates to one of 78 specialized agents at the appropriate tier (Haiku for search, Sonnet for implementation, Opus for design). Each agent gets the same conventions, rules, and identity injected automatically. Every decision is logged. Sharp edges are captured. Sessions are archived with handoffs.

## Getting Started

New to goYoke? Here's what happens in your first 5 minutes.

### Prerequisites
You need Claude Code installed and authenticated. If you don't have it, install it first from [claude.ai/code](https://claude.ai/code).

### Your First Interaction

1. **Open a terminal** in your project directory and run:
   ```bash
   goyoke
   ```

2. **See the TUI startup** — you'll see a `[Session Init]` message printed to the console telling you what language was detected and what conventions are loaded. This happens automatically.

3. **See the input field** — the TUI shows a terminal interface with an input prompt waiting for your request.

4. **Type your request** in natural language:
   ```
   add error handling to the auth middleware
   ```

5. **The router classifies and delegates** — you'll see routing messages like:
   ```
   [ROUTING] → go-pro (Go implementation task)
   ```

6. **The agent works** — progress appears in real-time. The agent reads your code, understands conventions, implements the fix.

7. **Results appear in your codebase** — the code is written directly to your files. You review it like any other change.

8. **Next request?** Type another request in the input field. The router remembers context across the session.

The key difference from raw Claude: each request is atomic. One invocation, one agent, one deliverable. No chat pollution. No context degradation.

## Example: What It Looks Like

Here's what a real `/review-plan` workflow produces:

### Setup
You've created a plan for implementing a feature and saved it to `tickets/feature-x/PLAN.md`. Now you want expert review before implementation.

### Running the Workflow
```bash
/review-plan
```

### What Happens
1. The router spawns `staff-architect-critical-review` (an Opus-tier agent)
2. The agent reads your plan
3. The agent greps the codebase to verify your claims about affected files, existing patterns, and dependencies
4. The agent applies a 7-layer review framework: assumptions, dependencies, failure modes, cost-benefit, testing strategy, architecture smells, and contractor readiness

### What You Get Back
Structured output with a verdict and detailed findings:

```
Verdict: CONCERNS (3 critical, 5 major, 4 minor, 4 commendations)

Critical Issues:
- Plan references 6 files but grep found 13 more affected files in the codebase
  Affected: middleware/, handlers/, integrations/, schemas/
- Unix Domain Sockets not mentioned (TUI-MCP bridge would crash on Windows)
- Process group kill has no Windows equivalent (orphaned worker processes)

Major Issues:
- Error handling strategy incomplete for concurrent file operations
- Database migration path not specified for schema changes
- Dependency graph has implicit assumption about initialization order

Commendations:
- Excellent test coverage plan for edge cases
- Good backward compatibility thinking
- Clear rollback strategy
```

### Key Difference from "Ask Claude Directly"
The agent **verified** your plan against the actual codebase using grep. It didn't just read your assumptions—it checked them. It found 7 issues you didn't mention because it actually looked at the code. That's the leverage goYoke provides: agents have access to your actual project state, not just what you describe.

### Output
Results saved to:
- `review-critique.md` — full findings
- `review-metadata.json` — structured data
- Your session archive captures the decision

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

## Cost Estimates

Here's what typical workflows cost:

| Workflow | Agents Used | Typical Cost | Duration |
|----------|-------------|-------------|----------|
| Bug fix (direct) | 1 Sonnet agent | $0.05-0.15 | 1-3 min |
| /explore | 1 Haiku scout + 1 Opus architect | $0.10-0.30 | 2-4 min |
| /review | 4 Sonnet reviewers + 1 Sonnet orchestrator | $0.15-0.40 | 3-5 min |
| /review-plan | 1 Opus staff-architect | $0.20-0.30 | 2-3 min |
| /braintrust | Mozart + Einstein + Staff-Architect + Beethoven (all Opus) | $0.50-1.50 | 5-10 min |
| /plan-tickets | Scout + Planner + Architect + Review + Tickets (mixed tiers) | $2.00-5.00 | 10-20 min |
| /implement | Architect + background workers | $1.00-3.00 | 5-15 min |

Cost depends on codebase size and task complexity. The router automatically uses the cheapest tier that can handle each subtask.

**Rule of thumb:** A day of heavy goYoke usage costs $5-15. Without tier routing, the same work in raw Opus would cost 3-5x more.

## Anti-Patterns (What goYoke Prevents)

### The Vibe Coding Pattern
**Without goYoke:** "Just implement it." Claude implements in Opus ($0.045/1K tokens), floods the conversation with 500 lines of code, and loses context for follow-up questions. By the time you need clarification, the model has forgotten the original goal.

**With goYoke:** Router delegates to Sonnet-tier go-pro ($0.009/1K tokens). Implementation happens in a one-shot agent that exits when done. Router context stays clean for your next request. Cost: ~5x cheaper.

### The Context Pollution Pattern
**Without goYoke:** You ask Claude to implement a feature, then review it, then fix issues, then add tests. By turn 15, Claude is losing context of the original requirements. You're repeating yourself. Prompts get longer to compensate.

**With goYoke:** Each step is a separate agent invocation. The router remembers the requirements. Agents get fresh context with injected conventions. No degradation. No prompt inflation.

### The Wrong-Tier Pattern
**Without goYoke:** Every request goes to Opus regardless of complexity. File search? Opus. Grep for a pattern? Opus. Generate boilerplate? Opus. Paying $0.045/1K tokens for work that Haiku does for $0.0005/1K.

**With goYoke:** The router automatically selects the cheapest tier that can handle the task. Haiku for search (50x cheaper), Sonnet for implementation (5x cheaper), Opus only for architecture. You save 10-50x on the same result.

### The Skip-The-Plan Pattern
**Without goYoke:** Jump straight into implementation. Discover halfway through that the approach is wrong, conflicts with existing patterns, or misses edge cases. Start over. Waste $2 and 30 minutes.

**With goYoke:** /explore scouts the problem first ($0.10). /review-plan catches issues before implementation ($0.25). Total planning cost: $0.35. Saves $2+ in avoided rework. Every time.

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
