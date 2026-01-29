# Compound Engineering Plugin Integration Guide

**For GOgent-Fortress Multi-Agent Architecture**

Version: 1.0.0
Date: 2026-01-30
Plugin Version: compound-engineering (via Compound Marketplace)

---

## Table of Contents

1. [What is Compound Engineering?](#what-is-compound-engineering)
2. [Installation](#installation)
3. [Architecture Comparison](#architecture-comparison)
4. [Compatibility Analysis](#compatibility-analysis)
5. [Integration Strategy](#integration-strategy)
6. [Workflow Command Reference](#workflow-command-reference)
7. [Agent Overlap & Routing](#agent-overlap--routing)
8. [MCP Server: Context7](#mcp-server-context7)
9. [Skills Deep Dive](#skills-deep-dive)
10. [Configuration Recipes](#configuration-recipes)
11. [Operational Patterns](#operational-patterns)
12. [Troubleshooting](#troubleshooting)

---

## What is Compound Engineering?

### Philosophy

> "Each unit of engineering work should make subsequent units easier—not harder."

Traditional development accumulates technical debt. Compound Engineering inverts this through an **80/20 split**: 80% planning and review, 20% execution.

### Core Cycle

```
Plan → Work → Review → Compound → Repeat
  │       │        │         │
  │       │        │         └─► Document learnings to docs/solutions/
  │       │        └─► Multi-agent parallel review before merge
  │       └─► Execute with worktrees and task tracking
  └─► Transform ideas into structured implementation plans
```

### Plugin Contents

| Component | Count | Purpose |
|-----------|-------|---------|
| Agents | 27 | Specialized reviewers, researchers, designers |
| Commands | 20 | 5 core workflow + 15 utility |
| Skills | 14 | Architecture guides, tools, automation |
| MCP Servers | 1 | Context7 (framework docs) |

---

## Installation

### Step 1: Install via Claude Code

```bash
/plugin marketplace add https://github.com/EveryInc/compound-engineering-plugin
/plugin install compound-engineering
```

### Step 2: Add Context7 MCP Server

The bundled Context7 MCP server may not auto-load. Add manually to `~/.claude/settings.json`:

```json
{
  "mcpServers": {
    "context7": {
      "type": "http",
      "url": "https://mcp.context7.com/mcp"
    }
  }
}
```

### Step 3: Install Browser Automation (Optional)

For `/test-browser` and `agent-browser` skill:

```bash
npm install -g agent-browser
agent-browser install  # Downloads Chromium
```

### Step 4: Set Gemini API Key (Optional)

For `/workflows:plan` research and `gemini-imagegen` skill:

```bash
export GEMINI_API_KEY="your-api-key"
```

---

## Architecture Comparison

### GOgent-Fortress vs Compound Engineering

| Aspect | GOgent-Fortress | Compound Engineering |
|--------|-----------------|----------------------|
| **Philosophy** | Routing efficiency, cost optimization | Knowledge compounding, thorough review |
| **Enforcement** | Hook binaries (deterministic) | Prompt-based (probabilistic) |
| **Tier Model** | haiku → sonnet → opus → gemini | Not explicitly tiered |
| **Agent Control** | routing-schema.json + hooks | Plugin-managed |
| **State** | .claude/tmp/*.json, memory/ | docs/solutions/, docs/brainstorms/ |
| **Review** | Single code-reviewer (Haiku) | 14 specialized reviewers (parallel) |
| **Planning** | architect + orchestrator | /workflows:plan + brainstorm |

### Complementary Strengths

**GOgent excels at:**
- Cost-efficient routing (Haiku for mechanical work)
- Hook-enforced constraints (cannot be bypassed)
- ML telemetry and pattern learning
- Scout-first for unknown scope
- Einstein escalation protocol

**Compound excels at:**
- Thorough multi-agent review (14 reviewers in parallel)
- Knowledge documentation (docs/solutions/)
- Rails/Ruby expertise (DHH-style, Kieran-style reviewers)
- Framework docs via Context7 MCP
- Brainstorming workflow

---

## Compatibility Analysis

### No Conflicts (Safe to Use Immediately)

| Plugin Component | GOgent Interaction | Status |
|------------------|-------------------|--------|
| `/workflows:plan` | Complements architect agent | ✅ Compatible |
| `/workflows:work` | Complements orchestrator | ✅ Compatible |
| `/workflows:compound` | New capability (knowledge capture) | ✅ No overlap |
| Context7 MCP | Complements librarian agent | ✅ Compatible |
| Most review agents | Different from code-reviewer | ✅ Compatible |

### Potential Friction Points

| Area | Issue | Mitigation |
|------|-------|------------|
| **Reviewer naming** | Plugin has `code-simplicity-reviewer`; GOgent has `code-reviewer` | Different tools - no collision |
| **Task() validation** | Plugin spawns agents without subagent_type | `gogent-validate` may warn on unexpected patterns |
| **Model selection** | Plugin doesn't use tiered model routing | Plugin agents run at default model, not cost-optimized |
| **Scout protocol** | Plugin does local research; GOgent uses gemini-slave scout | Use GOgent scout BEFORE `/workflows:plan` for large tasks |

### Hook Interactions

**PreToolUse (gogent-validate):**
- Plugin's `Task` calls don't specify `subagent_type`
- Validation hook may inject warnings
- **Impact:** Cosmetic warnings only; execution continues

**PostToolUse (gogent-sharp-edge):**
- Plugin tool usage counted in telemetry
- Routing reminders still fire every 10 tools
- **Impact:** Normal behavior; no conflicts

**SubagentStop (gogent-agent-endstate):**
- Plugin agents recorded in decision outcomes
- **Impact:** Good - provides telemetry on plugin agent usage

---

## Integration Strategy

### Recommended Hybrid Workflow

```
User Request
     │
     ├─► Is this a Rails/Ruby task? ──YES──► Use Compound workflows
     │
     ├─► Is this a Go task? ──YES──► Use GOgent routing (go-pro, go-tui, etc.)
     │
     ├─► Unknown scope? ──YES──► Scout first (GOgent), then decide
     │
     ├─► Multi-agent review needed? ──YES──► /workflows:review (Compound)
     │
     ├─► Document a solution? ──YES──► /workflows:compound
     │
     └─► Otherwise ──► Standard GOgent routing
```

### When to Use Compound

1. **Thorough code review** - Use `/workflows:review` for 14-agent parallel analysis
2. **Knowledge capture** - Use `/workflows:compound` after solving problems
3. **Framework research** - Use Context7 MCP for Rails, React, Next.js docs
4. **Rails development** - Use DHH/Kieran reviewers for Rails-specific patterns
5. **Brainstorming** - Use `/workflows:brainstorm` before planning complex features

### When to Use GOgent

1. **Go development** - Route to go-pro, go-tui, go-cli, go-concurrent
2. **Cost optimization** - Haiku scouts before expensive work
3. **Large context** - gemini-slave for 50K+ token analysis
4. **Enforced constraints** - Hook-based blocking (not bypassable)
5. **ML pattern learning** - Telemetry feeds future routing improvements

---

## Workflow Command Reference

### /workflows:brainstorm

**Purpose:** Explore requirements before committing to a plan.

**GOgent equivalent:** orchestrator (user interview mode)

**When to use:**
- Feature is vague or has multiple valid approaches
- User isn't sure what they want
- Domain is unfamiliar

**Output:** `docs/brainstorms/{topic}.md`

**Integration tip:** Run brainstorm BEFORE `/workflows:plan`. The plan command checks for recent brainstorms and uses them as context.

---

### /workflows:plan

**Purpose:** Transform ideas into implementation plans.

**GOgent equivalent:** architect (produces specs.md)

**Phases:**
1. Idea refinement (or use existing brainstorm)
2. Local research (parallel agents)
3. External research (if needed - uses Gemini)
4. Plan generation

**Output:** Structured markdown plan with checkboxes

**Integration tip:**
- For Go projects, run `scout` first via GOgent, then `/workflows:plan`
- The plan command's local research complements GOgent's codebase-search

---

### /workflows:work

**Purpose:** Execute plans systematically.

**GOgent equivalent:** orchestrator + implementation agents

**Key features:**
- Git worktree management (parallel development)
- Todo tracking with task dependencies
- Incremental commits at logical boundaries

**Integration tip:**
- Works well with GOgent's TaskCreate/TaskUpdate tools
- Respects your existing git workflow
- Branch naming follows your conventions

---

### /workflows:review

**Purpose:** Multi-agent code review before merge.

**GOgent equivalent:** code-reviewer (single agent, Haiku)

**Agents launched in parallel:**

| Category | Agents |
|----------|--------|
| Style | kieran-rails-reviewer, dhh-rails-reviewer, code-simplicity-reviewer |
| Architecture | architecture-strategist, pattern-recognition-specialist |
| Security | security-sentinel |
| Performance | performance-oracle |
| Data | data-integrity-guardian, data-migration-expert |
| History | git-history-analyzer |
| Agent-native | agent-native-reviewer |

**When to use:** Before merging significant PRs, especially Rails code.

**Integration tip:** This is MORE thorough than GOgent's code-reviewer. Use for important PRs; use GOgent's code-reviewer for quick style checks.

---

### /workflows:compound

**Purpose:** Document solved problems for future reference.

**GOgent equivalent:** memory-archivist (archives to .claude/memory/)

**Creates:** `docs/solutions/{category}/{filename}.md`

**Categories auto-detected:**
- build-errors/
- test-failures/
- runtime-errors/
- performance-issues/
- database-issues/
- security-issues/
- ui-bugs/
- integration-issues/
- logic-errors/

**Integration tip:**
- GOgent archives to `.claude/memory/` (Claude-readable)
- Compound archives to `docs/solutions/` (human/team-readable)
- Use BOTH for different audiences

---

## Agent Overlap & Routing

### Review Agents (Compound)

None of these conflict with GOgent's `code-reviewer`:

| Agent | Specialty | Use When |
|-------|-----------|----------|
| `kieran-rails-reviewer` | Strict Rails conventions | Rails PRs |
| `dhh-rails-reviewer` | 37signals style | Rails PRs |
| `kieran-python-reviewer` | Python conventions | Python PRs |
| `kieran-typescript-reviewer` | TypeScript conventions | TS PRs |
| `security-sentinel` | Vulnerability detection | Any security-sensitive code |
| `performance-oracle` | Perf analysis | Optimization work |
| `data-integrity-guardian` | DB migrations | Schema changes |
| `code-simplicity-reviewer` | Minimal complexity | Final review pass |

### Research Agents (Compound)

These complement GOgent's `librarian`:

| Agent | Specialty |
|-------|-----------|
| `best-practices-researcher` | External patterns |
| `framework-docs-researcher` | Framework docs |
| `repo-research-analyst` | Repository conventions |
| `git-history-analyzer` | Code evolution |

### GOgent Agents (No Plugin Equivalent)

| GOgent Agent | Keep Using For |
|--------------|----------------|
| `go-pro` | Go implementation |
| `go-tui` | Bubbletea TUI |
| `go-cli` | Cobra CLI |
| `go-concurrent` | Concurrency patterns |
| `haiku-scout` | Cheap scope assessment |
| `gemini-slave` | Large context (50K+) |
| `einstein` | Deep analysis (via /einstein) |

---

## MCP Server: Context7

### What It Provides

**Tools:**
- `resolve-library-id` - Find library ID for a framework
- `get-library-docs` - Fetch documentation

**Supported frameworks (100+):**
Rails, React, Next.js, Vue, Django, Laravel, FastAPI, Express, Svelte, and more.

### Usage Pattern

```
User: "How do I implement form validation in Rails 8?"

Claude: [Uses Context7 MCP]
  1. resolve-library-id("rails")
  2. get-library-docs(library_id, topic="form validation")
  3. Returns relevant Rails 8 documentation
```

### Integration with GOgent

Context7 complements `librarian` agent:

| Source | Use For |
|--------|---------|
| Context7 MCP | Framework-specific docs (Rails, React, etc.) |
| Librarian | General library research, GitHub permalinks |

**Recommendation:** Use Context7 for framework docs; use librarian for library internals and GitHub source reading.

---

## Skills Deep Dive

### Most Useful Skills for Your Stack

#### 1. git-worktree

**What:** Manage Git worktrees for parallel development.

**Why useful:** Work on multiple features simultaneously without stashing.

```bash
# Invoke skill
skill: git-worktree

# Creates isolated worktree for feature branch
# Keeps main branch clean
```

**GOgent integration:** Works with any branch workflow. Worktrees are git-native.

#### 2. compound-docs

**What:** Capture solved problems as searchable documentation.

**Why useful:** Team knowledge persistence.

**GOgent integration:** Complements memory-archivist:
- `compound-docs` → `docs/solutions/` (human-readable, team-shared)
- `memory-archivist` → `.claude/memory/` (Claude-readable, session state)

#### 3. agent-native-architecture

**What:** Build AI agents using prompt-native patterns.

**Why useful:** Designing your GOgent agents to be action + context parity compliant.

**Key principle:** Every feature should be achievable by an AI agent, not just humans.

#### 4. skill-creator / create-agent-skills

**What:** Guide for creating Claude Code skills.

**Why useful:** Extending GOgent-Fortress with new skills.

#### 5. gemini-imagegen

**What:** Generate and edit images via Gemini API.

**Requirements:**
- `GEMINI_API_KEY` environment variable
- Python: `pip install google-genai pillow`

**Not available in GOgent:** New capability.

#### 6. rclone

**What:** Upload files to S3, R2, B2, cloud storage.

**Why useful:** Deployment artifacts, backup.

---

## Configuration Recipes

### Recipe 1: Enable Context7 for All Projects

Add to `~/.claude/settings.json`:

```json
{
  "mcpServers": {
    "context7": {
      "type": "http",
      "url": "https://mcp.context7.com/mcp"
    }
  }
}
```

### Recipe 2: Integrate Compound Docs with GOgent Memory

Create a post-compound hook to notify memory-archivist:

```bash
# .claude/hooks/post-compound.sh
#!/bin/bash
# After /workflows:compound creates a doc, also log to GOgent memory

DOC_PATH="$1"
if [ -f "$DOC_PATH" ]; then
  # Extract title from YAML frontmatter
  TITLE=$(head -20 "$DOC_PATH" | grep "^title:" | cut -d: -f2 | xargs)

  # Append to pending learnings
  echo "{\"type\":\"solution\",\"title\":\"$TITLE\",\"path\":\"$DOC_PATH\",\"timestamp\":\"$(date -Is)\"}" \
    >> .claude/memory/pending-learnings.jsonl
fi
```

### Recipe 3: Route Rails Tasks to Compound, Go Tasks to GOgent

Add to your workflow decision prompt:

```markdown
## Routing Override

When the task involves:
- Rails, Ruby, ActiveRecord, ERB → Use `/workflows:*` commands
- Go, Cobra, Bubbletea, JSONL → Use GOgent routing (go-pro, go-tui, etc.)
- Unknown scope → Scout first (GOgent), then decide
```

### Recipe 4: Parallel Review (GOgent Scout + Compound Review)

For significant PRs:

```
1. [GOgent Scout] Assess scope
   - gemini-slave scout | calculate-complexity.sh
   - Produces: .claude/tmp/scout_metrics.json

2. [Compound Review] Multi-agent analysis
   - /workflows:review [PR number]
   - 14 agents in parallel

3. [GOgent Archive] Capture learnings
   - memory-archivist archives review outcomes
```

---

## Operational Patterns

### Pattern 1: Thorough Feature Development

```
/workflows:brainstorm "user authentication with OAuth"
   ↓ (creates docs/brainstorms/user-auth-oauth.md)
/workflows:plan "implement OAuth authentication"
   ↓ (uses brainstorm, creates plan.md)
/workflows:work "plan.md"
   ↓ (executes plan with worktrees, tracks todos)
/workflows:review [PR#]
   ↓ (14-agent parallel review)
/workflows:compound "OAuth implementation complete"
   ↓ (documents to docs/solutions/)
```

### Pattern 2: Quick Go Implementation (GOgent Only)

```
User: "Add a --verbose flag to the CLI"

[ROUTING] → go-cli (Cobra flag implementation)

Task(go-cli): Add verbose flag
   ↓
Output: Implementation complete

[No need for Compound workflows - single-file change]
```

### Pattern 3: Hybrid Large Feature

```
User: "Refactor the authentication module"

1. [GOgent Scout] Assess scope
   gemini-slave scout → "Large scope: 15 files, recommend sonnet tier"

2. [GOgent Architect] Create plan
   Task(architect) → specs.md with todos

3. [Compound Work] Execute with worktrees
   /workflows:work specs.md → Isolated worktree, tracked progress

4. [Compound Review] Pre-merge analysis
   /workflows:review PR# → 14-agent review

5. [GOgent Archive + Compound Capture]
   - memory-archivist → .claude/memory/
   - /workflows:compound → docs/solutions/
```

### Pattern 3: Framework Learning

```
User: "How do I use Hotwire Turbo frames in Rails?"

1. [Context7 MCP] Fetch Rails Turbo docs
   resolve-library-id("rails") → get-library-docs(turbo frames)

2. [Compound Researcher] Deepen understanding
   Task(framework-docs-researcher) → Best practices

3. [GOgent Librarian] Find examples
   Task(librarian) → GitHub permalinks to real implementations
```

---

## Troubleshooting

### Issue: "Context7 MCP not loading"

**Symptom:** `resolve-library-id` tool not available.

**Fix:** Add manually to settings.json (see Recipe 1).

### Issue: "gogent-validate warns about subagent_type"

**Symptom:** Warnings when plugin spawns agents.

**Why:** Plugin doesn't use GOgent's subagent_type convention.

**Impact:** Cosmetic only. Execution continues.

**Optional fix:** Suppress warnings for plugin agents in gogent-validate.

### Issue: "Duplicate review effort"

**Symptom:** Using both GOgent code-reviewer AND /workflows:review.

**Fix:** Choose one:
- Quick checks → GOgent code-reviewer (Haiku, cheap)
- Thorough PR review → /workflows:review (14 agents, comprehensive)

### Issue: "Knowledge spread across two systems"

**Symptom:** Some learnings in `.claude/memory/`, others in `docs/solutions/`.

**Why:** Different purposes:
- `.claude/memory/` - Claude-readable, session state
- `docs/solutions/` - Human-readable, team knowledge

**Recommendation:** Use both. Run `/workflows:compound` for team docs, let memory-archivist handle session state automatically.

### Issue: "Plugin agents run at wrong model tier"

**Symptom:** Plugin agents use default model, not cost-optimized routing.

**Why:** Plugin doesn't implement tiered model selection.

**Impact:** May cost more than equivalent GOgent routing.

**Recommendation:** Use plugin for Rails/review (where expertise matters), use GOgent for Go (where cost optimization + expertise both apply).

---

## Summary: When to Use What

| Situation | Use |
|-----------|-----|
| Go implementation | GOgent (go-pro, go-tui, go-cli) |
| Rails implementation | Compound (kieran/dhh reviewers) |
| Quick code check | GOgent (code-reviewer) |
| Thorough PR review | Compound (/workflows:review) |
| Large context analysis | GOgent (gemini-slave) |
| Framework docs lookup | Compound (Context7 MCP) |
| Library source reading | GOgent (librarian) |
| Document a solution | Compound (/workflows:compound) |
| Archive session state | GOgent (memory-archivist) |
| Deep analysis | GOgent (/einstein) |
| Feature brainstorming | Compound (/workflows:brainstorm) |
| Implementation planning | Either (architect vs /workflows:plan) |
| Work execution | Compound (/workflows:work) for worktree isolation |

---

## Appendix: Plugin Agent Reference

### Review Agents (14)

| Agent | Language Focus |
|-------|----------------|
| agent-native-reviewer | AI agent accessibility |
| architecture-strategist | Architecture compliance |
| code-simplicity-reviewer | Minimalism |
| data-integrity-guardian | DB migrations |
| data-migration-expert | ID mappings, rollback |
| deployment-verification-agent | Go/No-Go checklists |
| dhh-rails-reviewer | 37signals Rails style |
| kieran-rails-reviewer | Strict Rails conventions |
| kieran-python-reviewer | Python conventions |
| kieran-typescript-reviewer | TypeScript conventions |
| pattern-recognition-specialist | Anti-patterns |
| performance-oracle | Performance |
| security-sentinel | Security vulnerabilities |
| julik-frontend-races-reviewer | JS race conditions |

### Research Agents (4)

| Agent | Focus |
|-------|-------|
| best-practices-researcher | External patterns |
| framework-docs-researcher | Framework docs |
| git-history-analyzer | Code evolution |
| repo-research-analyst | Repo conventions |

### Design Agents (3)

| Agent | Focus |
|-------|-------|
| design-implementation-reviewer | Figma match |
| design-iterator | UI refinement |
| figma-design-sync | Figma sync |

### Workflow Agents (5)

| Agent | Focus |
|-------|-------|
| bug-reproduction-validator | Bug repro |
| every-style-editor | Every.to style |
| lint | Ruby/ERB linting |
| pr-comment-resolver | PR comment fixes |
| spec-flow-analyzer | User flow gaps |

### Docs Agents (1)

| Agent | Focus |
|-------|-------|
| ankane-readme-writer | Ankane-style READMEs |

---

*This guide will be updated as the plugin evolves. Last verified against compound-engineering plugin at commit 250212a.*
