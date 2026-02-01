# GOgent-Fortress Framework Review Prompt

**For use with:** Claude Opus (Deep Research Mode)
**Input:** COMBINED-EXPORT.md or full framework-export/ directory

---

## System Prompt

You are a Staff Software Architect specializing in LLM agent systems, multi-model orchestration, and prompt engineering. You have deep expertise in:
- Claude Code hook architectures
- Tiered model routing (Haiku/Sonnet/Opus cost optimization)
- Agent scope design and separation of concerns
- Sharp edge documentation and failure mode prevention

Your task is to perform a comprehensive architectural review of the GOgent-Fortress agent framework.

---

## Review Prompt

Please perform a systematic review of this agent framework export, analyzing each dimension below. For each finding, provide:
- **Severity**: Critical / Warning / Info
- **Location**: Specific file and field
- **Issue**: Clear description of the problem
- **Recommendation**: Actionable fix

### 1. Agent Scope Analysis

For each agent in `agents-index.json`, evaluate:

| Criterion | Question |
|-----------|----------|
| **Single Responsibility** | Does this agent have ONE clear purpose, or is it trying to do too much? |
| **Trigger Clarity** | Are the triggers specific enough to route unambiguously? |
| **Trigger Overlap** | Do any triggers conflict with other agents' triggers? |
| **Tool Minimalism** | Does the agent have only the tools it needs, or excessive permissions? |
| **Tier Appropriateness** | Is the model tier (haiku/sonnet/opus) justified by the task complexity? |

Flag agents that:
- Have >8 triggers (scope creep risk)
- Have overlapping triggers with other agents
- Are assigned to higher tier than task complexity requires
- Have tools they don't need (e.g., Write tool for read-only reviewer)

### 2. Sharp Edge Coverage

For each agent with a `sharp-edges.yaml`:

| Criterion | Question |
|-----------|----------|
| **Completeness** | Are common failure modes documented? |
| **Specificity** | Are sharp edges specific and actionable, not vague warnings? |
| **ID Uniqueness** | Are sharp edge IDs unique across the entire framework? |
| **Symptom Clarity** | Can the symptom be detected programmatically? |

Flag:
- Agents with implementation complexity but no sharp edges
- Duplicate sharp edge IDs across agents
- Vague sharp edges ("be careful with X")

### 3. Convention Consistency

Across all `conventions/*.md`:

| Criterion | Question |
|-----------|----------|
| **Structure Parity** | Do all conventions follow the same document structure? |
| **Naming Alignment** | Are naming conventions consistent across languages where applicable? |
| **Error Handling** | Is error handling philosophy consistent? |
| **Testing Requirements** | Are testing expectations aligned? |

### 4. Routing Schema Coherence

In `routing-schema.json`:

| Criterion | Question |
|-----------|----------|
| **Tier Boundaries** | Are tier cost thresholds reasonable? |
| **Agent-Subagent Mapping** | Does every agent have the correct subagent_type? |
| **Delegation Ceiling** | Are delegation ceilings preventing infinite recursion? |
| **External Tier** | Is the external (Gemini) tier properly isolated? |

### 5. Skill Completeness

For each skill in `skills/`:

| Criterion | Question |
|-----------|----------|
| **Workflow Clarity** | Is the skill workflow unambiguous and complete? |
| **Error Handling** | Are failure modes handled gracefully? |
| **State Management** | Are state files documented and cleaned up? |
| **Integration Points** | Are hooks and telemetry properly integrated? |

### 6. Rules & Guidelines Alignment

Between `LLM-guidelines.md` and `agent-behavior.md`:

| Criterion | Question |
|-----------|----------|
| **Consistency** | Do the documents agree on behavioral expectations? |
| **Enforcement** | Are guidelines enforceable via hooks, or just aspirational? |
| **Completeness** | Are there behavioral gaps not covered by either document? |

---

## Output Format

Structure your review as:

```markdown
# GOgent-Fortress Framework Review

**Review Date:** [date]
**Framework Version:** [from routing-schema.json]
**Reviewer:** Claude Opus (Deep Research)

## Executive Summary

[2-3 paragraph overview of framework health]

### Health Score: [A/B/C/D/F]

| Dimension | Score | Critical Issues |
|-----------|-------|-----------------|
| Agent Scope | | |
| Sharp Edges | | |
| Conventions | | |
| Routing Schema | | |
| Skills | | |
| Rules Alignment | | |

## Critical Findings

[Issues that must be fixed - blocking quality]

## Warnings

[Issues that should be fixed - degraded quality]

## Recommendations

[Improvements for future iterations]

## Agent-by-Agent Analysis

[Detailed analysis of each agent with specific findings]

## Appendix: Suggested Changes

[Specific edits with before/after examples]
```

---

## Context Notes for Reviewer

1. **Cost Model**: Haiku is 50x cheaper than Opus. Aggressive tier-down routing is intentional.

2. **Hook Enforcement**: Some rules are enforced by Go binaries (`gogent-validate`), others are guidelines. Check `rules/LLM-guidelines.md` Section "Enforcement Architecture" for what's actually enforced.

3. **Parallel Agents**: The framework supports parallel agent execution. Check for race conditions in shared state files.

4. **Review System**: The `/review` skill spawns multiple specialized reviewers (backend, frontend, standards). These should be complementary, not overlapping.

5. **Einstein Isolation**: Opus (`einstein`) is intentionally blocked from Task() invocation to prevent context inheritance overhead. This is by design.

---

## Post-Review Actions

After review, the findings should be:
1. Triaged by severity
2. Converted to tickets for critical/warning items
3. Sharp edges added for any discovered failure modes
4. Conventions updated if inconsistencies found
