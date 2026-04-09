---
name: schema-extend
description: Extend a boilerplate agent with deep domain expertise via braintrust, or refine an already-expanded agent with a targeted Opus pass
---

# Schema Extend Skill v1.0

## Purpose

Extends agent `.md` files from boilerplate scaffolds into production-quality agents with deep domain expertise. Two modes:

- **Boilerplate → Production** (`/schema-extend {agent-id}`): Full braintrust workflow (Mozart → Einstein + Staff-Architect → Beethoven). Deep domain research, structural consistency review, synthesized expansion. ~$3-5.
- **Production → Refined** (`/schema-extend {agent-id} --refine`): Single Opus agent pass for targeted improvements to an already-expanded agent. ~$1-2.

**What this skill does:**

1. **Detect** — Read the agent .md file, classify as boilerplate or already-expanded
2. **Analyze** — Extract domain focus from frontmatter (description, focus_areas, triggers, category)
3. **Assemble** — Build the expansion/refinement prompt automatically from agent metadata
4. **Dispatch** — Spawn Mozart (braintrust mode) or single Opus agent (refine mode)
5. **Present** — Return expanded content for user review
6. **Apply** — User decides whether to apply (the skill does NOT auto-write)

**What this skill does NOT do:**

- Auto-apply changes (user reviews and approves first)
- Create new agents (use scaffolder for that)
- Modify frontmatter (body content only)

---

## Invocation

| Command | Behavior |
|---------|----------|
| `/schema-extend genomics-reviewer` | Full braintrust expansion (boilerplate → prod) |
| `/schema-extend genomics-reviewer --refine` | Single Opus refinement pass (prod → deeper) |
| `/schema-extend genomics-reviewer --refine "add more PTM examples"` | Targeted refinement with focus hint |
| `/schema-extend --list` | List agents and their expansion status |

---

## Prerequisites

**Required:**
- Agent must exist in `.claude/agents/{agent-id}/{agent-id}.md`
- Agent must be registered in `agents-index.json`
- For braintrust mode: Mozart must have `interactive: true` in agents-index.json

---

## Workflow

### Step 1: Validate Agent Exists

```javascript
const agentId = args.split(/\s+/)[0];
const refineMode = args.includes("--refine");
const listMode = args.includes("--list");

if (listMode) {
    // List all agents with expansion status — see Phase: List Mode below
    return;
}

if (!agentId) {
    Output("[schema-extend] Usage: /schema-extend {agent-id} [--refine] [\"focus hint\"]");
    return;
}

const agentFile = `.claude/agents/${agentId}/${agentId}.md`;
Read({ file_path: agentFile });
```

### Step 2: Detect Expansion Status

Read the agent .md file and classify:

```
BOILERPLATE indicators (any 2+ = boilerplate):
  - Review Checklist has < 15 items (count "- [ ]" lines)
  - Severity Classification has < 3 examples per level
  - No Sharp Edge ID table
  - Body < 200 lines (after frontmatter)
  - Contains "placeholder" or "boilerplate" in comments

EXPANDED indicators (any 2+ = already expanded):
  - Review Checklist has 15+ items
  - Severity Classification has 5+ examples per level
  - Sharp Edge ID table present with 5+ entries
  - Body > 300 lines
```

Report status:
```
[schema-extend] Agent: {agent-id}
[schema-extend] Status: BOILERPLATE | EXPANDED
[schema-extend] Mode: braintrust (full expansion) | refine (targeted pass)
```

If agent is boilerplate and `--refine` was passed, warn:
```
[schema-extend] WARNING: Agent is still boilerplate. Use without --refine for full expansion first.
```

If agent is expanded and no `--refine`, auto-switch:
```
[schema-extend] Agent already expanded. Switching to --refine mode.
```

### Step 3: Extract Domain Metadata

Read frontmatter fields to auto-generate the expansion prompt:

```yaml
# From agent frontmatter:
description: → domain summary for Einstein
focus_areas: → specific review topics
triggers: → domain keywords
category: → agent category (bioinformatics-review, review, etc.)
conventions_required: → language context
```

Also read:
- `agents-index.json` entry for the agent (triggers, description)
- The reference reviewer for structural patterns:
  - If category is `bioinformatics-review` → read `backend-reviewer.md` as pattern
  - If category is `review` → read `backend-reviewer.md` as pattern
  - Otherwise → read the most structurally similar agent in the same category

### Step 4: Assemble Prompt

#### Braintrust Mode (boilerplate → prod)

```
Expand the body content of the {agent-name} agent (.claude/agents/{agent-id}/{agent-id}.md).

CURRENT STATE: The agent has boilerplate body sections that were scaffolded from a reference pattern. The domain-specific content is placeholder-quality.

DOMAIN: {description from frontmatter}
FOCUS AREAS: {focus_areas from frontmatter, joined with newlines}
TRIGGERS: {triggers from frontmatter, joined with commas}
CATEGORY: {category from frontmatter}

GOAL: Replace boilerplate with deep domain expertise:

1. REVIEW CHECKLIST: Expand to 20-30 domain-specific checks organized by priority. Each check should include:
   - Why this check matters (consequence of getting it wrong)
   - What to look for in code (specific function calls, parameter values, file operations)
   - Common mistakes (patterns that look correct but are wrong)

2. SEVERITY CLASSIFICATION: Expand with 5-10 specific examples per severity level, grounded in real failure modes. Each example should name the specific tool/method/parameter involved.

3. SHARP EDGE CORRELATION: Create a domain-specific sharp edge ID table with 10+ semantic IDs ({domain}-{issue} format).

4. IDENTITY: Refine the role statement. What distinguishes an expert in this domain from a generalist?

CONSTRAINTS:
- Keep Output Format and Parallelization sections unchanged
- Keep the CRITICAL file-reading warning unchanged
- All checklist items must be verifiable by reading code
- Severity classifications must be actionable from code alone

REFERENCE: Read the current agent file and {reference-agent} for structural patterns.
```

#### Refine Mode (prod → deeper)

```
Refine the body content of the {agent-name} agent (.claude/agents/{agent-id}/{agent-id}.md).

CURRENT STATE: The agent has been expanded with domain expertise. This is a refinement pass.

{if focus_hint provided:}
FOCUS: {focus_hint}
{else:}
FOCUS: Identify gaps, strengthen weak sections, add missing edge cases.
{end}

DOMAIN: {description from frontmatter}

TASKS:
1. Review each checklist item — are there missing checks that a domain expert would include?
2. Review severity examples — are there real-world failure modes not yet covered?
3. Review sharp edge IDs — are there common pitfalls missing?
4. Verify technical accuracy — are tool names, parameter names, and methodology descriptions current?
5. Check for consistency with other agents in the same category

OUTPUT: Provide the refined sections as replacement text, clearly marking what changed and why.

CONSTRAINTS:
- Do NOT reduce existing content (only add or improve)
- Keep structural format unchanged
- All additions must be code-verifiable
```

### Step 5: Dispatch

#### Braintrust Mode

Spawn Mozart with the assembled prompt. Mozart handles the full workflow:

```javascript
mcp__gofortress-interactive__spawn_agent({
    agent: "mozart",
    description: `Schema extension: ${agentId} (braintrust)`,
    prompt: assembledPrompt,
    model: "opus",
    timeout: 900000,  // 15 min for full braintrust
});
```

Output:
```
[schema-extend] Braintrust launched for {agent-id}
[schema-extend] Mozart will interview, scout, then dispatch Einstein + Staff-Architect → Beethoven
[schema-extend] Estimated cost: $3-5
[schema-extend] Use /team-status to track progress
[schema-extend] When complete, review output and apply with:
[schema-extend]   /schema-extend {agent-id} --apply
```

#### Refine Mode

Spawn a single Opus agent (use architect for analytical depth):

```javascript
mcp__gofortress-interactive__spawn_agent({
    agent: "architect",
    description: `Schema refinement: ${agentId}`,
    prompt: assembledPrompt,
    model: "opus",
    timeout: 600000,  // 10 min for single pass
});
```

Output:
```
[schema-extend] Opus refinement pass launched for {agent-id}
[schema-extend] Estimated cost: $1-2
[schema-extend] Result will contain refined sections for review
```

### Step 6: Return

The skill returns immediately after dispatch. User checks results via the agent's output or `/team-result` (braintrust mode).

The output from braintrust/architect contains the expanded/refined body sections. The user reviews and manually applies them to the agent .md file (or asks the router to do it).

---

## List Mode

`/schema-extend --list` scans all agents and reports expansion status:

```
[schema-extend] Agent Expansion Status

Category: bioinformatics-review
  genomics-reviewer        BOILERPLATE  (12 checklist items, 0 sharp edges)
  proteomics-reviewer      BOILERPLATE  (10 checklist items, 0 sharp edges)
  proteogenomics-reviewer  BOILERPLATE  (11 checklist items, 0 sharp edges)
  proteoform-reviewer      BOILERPLATE  (9 checklist items, 0 sharp edges)
  mass-spec-reviewer       BOILERPLATE  (10 checklist items, 0 sharp edges)
  bioinformatician-reviewer BOILERPLATE (11 checklist items, 0 sharp edges)
  pasteur                  BOILERPLATE  (synthesis agent, 0 sharp edges)

Category: review
  backend-reviewer         EXPANDED     (24 checklist items, 10 sharp edges)
  frontend-reviewer        EXPANDED     (18 checklist items, 8 sharp edges)
  standards-reviewer       EXPANDED     (15 checklist items, 0 sharp edges)
  architect-reviewer       EXPANDED     (12 checklist items, 0 sharp edges)
```

Detection logic:
- Count `- [ ]` lines in body = checklist items
- Count rows in Sharp Edge table = sharp edges
- Body line count after frontmatter
- Report as BOILERPLATE or EXPANDED based on thresholds

---

## Cost Model

| Mode | Agents Involved | Cost |
|------|----------------|------|
| Braintrust (boilerplate → prod) | Mozart + Einstein + Staff-Architect + Beethoven | $3-5 |
| Refine (prod → deeper) | Single Opus (architect) | $1-2 |
| List | Router only (file reads) | $0 |

---

## Troubleshooting

**"Agent not found"**
- Check `.claude/agents/{agent-id}/{agent-id}.md` exists
- Verify agent is in agents-index.json

**"Mozart has no MCP access"**
- Ensure Mozart has `interactive: true` in agents-index.json
- Rebuild TUI binary if recently changed

**"Braintrust timed out"**
- Default timeout is 15 minutes
- Large agents or complex domains may need more time
- Check `/team-status` for progress

**"Refine mode didn't change much"**
- Provide a specific focus hint: `/schema-extend {id} --refine "add edge cases for DIA quantification"`
- Already well-expanded agents may need targeted prompts

---

## Examples

### Example 1: Expand Boilerplate Agent

```bash
$ /schema-extend genomics-reviewer

[schema-extend] Agent: genomics-reviewer
[schema-extend] Status: BOILERPLATE (12 checklist items, 0 sharp edges)
[schema-extend] Mode: braintrust (full expansion)
[schema-extend] Domain: Genome assembly, variant calling, alignment...
[schema-extend] Braintrust launched for genomics-reviewer
[schema-extend] Estimated cost: $3-5
```

### Example 2: Refine Expanded Agent

```bash
$ /schema-extend backend-reviewer --refine "add GraphQL-specific security checks"

[schema-extend] Agent: backend-reviewer
[schema-extend] Status: EXPANDED (24 checklist items, 10 sharp edges)
[schema-extend] Mode: refine (targeted Opus pass)
[schema-extend] Focus: add GraphQL-specific security checks
[schema-extend] Opus refinement pass launched
[schema-extend] Estimated cost: $1-2
```

### Example 3: List All Agents

```bash
$ /schema-extend --list

[schema-extend] Agent Expansion Status
  ...
```

---

**Skill Version:** 1.0
**Last Updated:** 2026-04-09
