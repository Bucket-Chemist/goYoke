---
name: braintrust
description: >
  Multi-perspective deep analysis workflow. Invokes Mozart (orchestrator) who
  conducts clarification interview, spawns scouts for reconnaissance, then
  dispatches Einstein (theoretical) and Staff-Architect (practical) in parallel.
  Beethoven synthesizes orthogonal analyses into standardized output document.
replaces: einstein
version: 1.0.0
---

# Braintrust Skill v1.0

## Purpose

Braintrust is the premium analysis workflow for complex problems requiring both theoretical depth and practical rigor. It replaces the standalone `/einstein` skill with a multi-agent orchestrated approach.

**What this skill does:**
1. **Mozart** (Opus) - Intake, interview, scout, decompose problem
2. **Einstein** (Opus) - Theoretical analysis: root cause, frameworks, first principles
3. **Staff-Architect** (Opus) - Practical review: 7-layer framework, risks, implementation
4. **Beethoven** (Opus) - Synthesize orthogonal analyses into unified document

**What this skill does NOT do:**
- Implement code (analysis only, delegate execution after)
- Skip user confirmation (Mozart always confirms before heavy Opus spend)
- Produce vague outputs (standardized document format)

---

## Invocation

| Command | Behavior |
|---------|----------|
| `/braintrust` | Start with problem statement prompt |
| `/braintrust "question"` | Quick mode with inline problem |
| `/braintrust path/to/gap.md` | Process existing GAP document |

---

## Workflow

```
┌─────────────────────────────────────────────────────────────────────┐
│                        /braintrust                                  │
│                     User Invocation                                 │
└───────────────────────────┬─────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    MOZART (Opus)                                    │
│              Problem Decomposition Orchestrator                     │
│                                                                     │
│  Phase 1: INTAKE                                                    │
│    - Parse user input (question, GAP doc, or raw problem)           │
│                                                                     │
│  Phase 2: INTERVIEW (if needed)                                     │
│    - Conduct clarification interview (max 3 questions)              │
│                                                                     │
│  Phase 3: RECONNAISSANCE                                            │
│    - Spawn scouts (haiku) to gather scope metrics                   │
│    - Assemble Problem Brief                                         │
│                                                                     │
│  Phase 4: CONFIRMATION CHECKPOINT                                   │
│    - Present Problem Brief to user                                  │
│    - User approves before heavy Opus spend                          │
│                                                                     │
│  Phase 5: ORTHOGONAL DISPATCH (parallel)                            │
│    ┌─────────────────────┐    ┌─────────────────────────────────┐  │
│    │  EINSTEIN (Opus)    │    │  STAFF-ARCHITECT-CR (Opus)      │  │
│    │  Theoretical        │    │  Practical/Implementation       │  │
│    │  - Root cause       │    │  - 7-layer review               │  │
│    │  - Conceptual model │    │  - Risk assessment              │  │
│    │  - Novel approaches │    │  - Failure modes                │  │
│    │  - First principles │    │  - Contractor readiness         │  │
│    └──────────┬──────────┘    └─────────────────┬───────────────┘  │
│               │                                  │                  │
│               └─────────────┬────────────────────┘                  │
│                             │                                       │
│  Phase 6: HANDOFF TO BEETHOVEN                                      │
│    - Collect both analyses                                          │
│    - Pass to Beethoven with Problem Brief                           │
└─────────────────────────────┬───────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                   BEETHOVEN (Opus)                                  │
│                  Composer / Synthesizer                             │
│                                                                     │
│  - Identify convergences (where both agree)                         │
│  - Highlight divergences (where they disagree)                      │
│  - Resolve contradictions with higher-order reasoning               │
│  - Synthesize unified recommendation                                │
│  - Produce standardized Braintrust Analysis Document                │
└─────────────────────────────┬───────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│              BRAINTRUST ANALYSIS DOCUMENT                           │
│              .claude/braintrust/analysis-{timestamp}.md             │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Execution

When `/braintrust` is invoked:

### Step 1: Invoke Mozart

```javascript
Task({
  description: "Mozart: Braintrust problem decomposition",
  subagent_type: "Plan",
  model: "opus",
  prompt: `AGENT: mozart

BRAINTRUST INVOCATION

USER INPUT: {user_input}
INPUT TYPE: {raw_problem | gap_document | inline_question}

Execute full Braintrust workflow:
1. Parse input
2. Interview if needed (max 3 questions)
3. Spawn scouts for reconnaissance
4. Assemble Problem Brief
5. Confirm with user before proceeding
6. Dispatch Einstein + Staff-Architect in parallel
7. Collect analyses
8. Invoke Beethoven for synthesis
9. Return final analysis document path`
});
```

### Step 2: Mozart Handles Everything

Mozart orchestrates the entire workflow internally:
- Spawns scouts (haiku)
- Spawns Einstein (opus)
- Spawns Staff-Architect-Critical-Review (opus)
- Spawns Beethoven (opus)
- Returns final document path

### Step 3: Return to User

```
[Braintrust] Analysis complete.
[Braintrust] Output: .claude/braintrust/analysis-{timestamp}.md
[Braintrust] Agents: Mozart → Einstein + Staff-Architect → Beethoven
```

---

## Output Format

The final Braintrust Analysis document includes:

```markdown
# Braintrust Analysis: {Title}

## Executive Summary
## Problem Statement
## Analysis Perspectives
   ### Einstein (Theoretical)
   ### Staff-Architect (Practical)
## Convergence Points
## Divergence Resolution
## Unified Recommendations
## Implementation Pathway
## Risk Assessment
## Open Questions
## Appendix: Full Analyses
## Metadata
```

---

## Cost Model

| Agent | Typical Tokens | Estimated Cost |
|-------|----------------|----------------|
| Mozart (orchestration) | 15,000 in / 5,000 out | ~$0.75 |
| Scouts (2x haiku) | 2,000 each | ~$0.01 |
| Einstein | 20,000 in / 8,000 out | ~$1.10 |
| Staff-Architect | 20,000 in / 6,000 out | ~$1.00 |
| Beethoven | 25,000 in / 10,000 out | ~$1.25 |
| **Total** | - | **~$4.10** |

**Note**: No cost ceiling. Quality over cost for Braintrust.

---

## State Files

| File | Written By | Read By | Purpose |
|------|------------|---------|---------|
| `.claude/braintrust/problem-brief-*.md` | Mozart | Einstein, Staff-Architect, Beethoven | Problem decomposition |
| `.claude/braintrust/analysis-*.md` | Beethoven | User | Final output |
| `.claude/braintrust/mozart-log-*.jsonl` | Mozart | Telemetry | Workflow tracking |
| `.claude/tmp/scout_metrics.json` | Scouts | Mozart | Reconnaissance data |

---

## Comparison: Einstein vs Braintrust

| Aspect | Old /einstein | New /braintrust |
|--------|--------------|-----------------|
| Agents | 1 (Einstein) | 4 (Mozart, Einstein, Staff-Arch, Beethoven) |
| Perspectives | Single deep analysis | Theoretical + Practical orthogonal |
| Interview | None (GAP doc required) | Built-in clarification |
| Confirmation | Pre-flight check | Full problem brief review |
| Output | Analysis markdown | Standardized synthesis document |
| Cost | ~$0.92 | ~$4.10 |
| Use case | Bounded escalations | Complex problem workshopping |

---

## When to Use Braintrust

**Use Braintrust for:**
- Complex architectural decisions
- Problems with both theoretical and practical dimensions
- Situations requiring multiple perspectives
- High-stakes decisions worth the cost
- Thought workshopping and whiteboarding
- Problems where implementation concerns matter

**Don't use Braintrust for:**
- Simple debugging (use regular agents)
- Single-perspective analysis (old einstein pattern still works via escalation)
- Time-sensitive issues (4 Opus agents take time)
- Well-understood problems (overkill)

---

## Migration from /einstein

The `/einstein` skill is replaced by `/braintrust`. However:

1. **Escalation protocol still works**: Agents can still generate GAP documents
2. **GAP documents as input**: `/braintrust path/to/gap.md` processes existing GAPs
3. **Legacy triggers preserved**: "escalate to einstein" still routes appropriately

For simple escalations that don't need full multi-perspective analysis, the GAP document pattern remains valid - just invoke `/braintrust` with the GAP path.

---

## Skill Metadata

```yaml
skill_id: braintrust
version: 1.0.0
replaces: einstein
tier: 3
model: opus (all agents)
agents_invoked:
  - mozart
  - einstein
  - staff-architect-critical-review
  - beethoven
cost_estimate: "$4-6 per invocation"
output_location: ".claude/braintrust/"
user_confirmation: required (Phase 4)
```
