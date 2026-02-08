---
name: team-result
description: Display final output from a completed orchestration team
version: 1.0.0
---

# Team Result Display Skill

Display the final output from a completed orchestration team by reading stdout JSON files.

## Usage

```
/team-result <team-name>
```

**Arguments:**
- `team-name` (REQUIRED): Name of the team to display results for

## Execution Steps

### 1. Parse Arguments

Extract the team-name from the invocation. If missing, output:
```
Error: team-name is required
Usage: /team-result <team-name>
```

### 2. Discover Session Directory

1. Check environment variable `GOGENT_SESSION_DIR`
2. If unset, fallback: Find most recent session directory with a `teams/` subdirectory under `~/.claude/sessions/`
3. Team directory path: `{session_dir}/teams/{team_name}/`

If no session found:
```
Error: No session with teams found. Has a team been started?
```

### 3. Read Team Configuration

Read `{team_dir}/config.json` to determine:
- `workflow_type`: "braintrust", "review", or "implementation"
- `status`: "planning", "running", "completed", "failed"
- Cost and duration metadata

If config not found:
```
Error: Team '{team_name}' not found in session
```

If status is not "completed":
```
Team '{team_name}' is still {status}. Use /team-status to check progress.
```

### 4. Workflow-Specific Output

Based on `workflow_type`, format output as specified below.

---

## BRAINTRUST WORKFLOW

**Read:** `stdout_beethoven.json` (the final synthesis agent)

**JSON Schema:**
```
{
  executive_summary: string
  problem_statement: {
    original: string
    clarified: string
    success_criteria: string[]
  }
  analysis_perspectives: {
    einstein_summary: string
    einstein_strengths: string[]
    staff_architect_verdict: string
    staff_architect_strengths: string[]
  }
  convergence_points: [{
    topic: string
    einstein_view: string
    staff_architect_view: string
    synthesis: string
    confidence: string
  }]
  divergence_resolution: [{
    topic: string
    einstein_position: string
    staff_architect_position: string
    resolution: string
    reasoning: string
    confidence: string
    unresolved: boolean
  }]
  unified_recommendations: {
    primary: {
      recommendation: string
      rationale: string
      supporting_evidence: string[]
      implementation_phases: [{
        phase: string
        steps: string[]
        success_metrics: string[]
      }]
    }
    secondary: [{
      recommendation: string
      rationale: string
      when_to_consider: string
    }]
    not_recommended: [{
      approach: string
      reasoning: string
    }]
  }
  risk_assessment: [{
    risk: string
    probability: string
    impact: string
    source: string
    mitigation: string
    owner: string
  }]
  open_questions: [{
    question: string
    importance: string
    suggested_next_step: string
  }]
}
```

**Output Format:**

```markdown
# Braintrust Analysis: {team_name}

## Executive Summary

{executive_summary}

## Problem Statement

**Original:** {problem_statement.original}

**Clarified:** {problem_statement.clarified}

**Success Criteria:**
{for each criterion in success_criteria}
- {criterion}

## Analysis Perspectives

### Einstein (Theoretical)
{einstein_summary}

**Strengths:**
{for each strength in einstein_strengths}
- {strength}

### Staff Architect (Critical)
{staff_architect_verdict}

**Strengths:**
{for each strength in staff_architect_strengths}
- {strength}

## Convergence Points

{for each point in convergence_points}
### {topic}
**Synthesis:** {synthesis}
**Confidence:** {confidence}

{if empty: "No explicit convergence points identified."}

## Divergence Resolution

{for each divergence in divergence_resolution}
### {topic}
**Einstein Position:** {einstein_position}
**Staff Architect Position:** {staff_architect_position}

**Resolution:** {resolution}
**Reasoning:** {reasoning}
**Confidence:** {confidence}
{if unresolved: "⚠️ UNRESOLVED"}

{if empty: "No divergences required resolution."}

## Unified Recommendations

### Primary Recommendation

**{primary.recommendation}**

**Rationale:** {primary.rationale}

**Supporting Evidence:**
{for each evidence in primary.supporting_evidence}
- {evidence}

**Implementation Phases:**
{for each phase in primary.implementation_phases}
#### Phase: {phase.phase}
**Steps:**
{for each step in phase.steps}
- {step}
**Success Metrics:**
{for each metric in phase.success_metrics}
- {metric}

### Secondary Recommendations

{for each rec in secondary}
**{rec.recommendation}**
- Rationale: {rec.rationale}
- When to consider: {rec.when_to_consider}

{if empty: "No secondary recommendations."}

### Not Recommended

{for each not_rec in not_recommended}
- **{not_rec.approach}**: {not_rec.reasoning}

{if empty: "No approaches explicitly rejected."}

## Risk Assessment

{if risks exist}
| Risk | Probability | Impact | Mitigation | Owner |
|------|-------------|---------|------------|-------|
{for each risk in risk_assessment}
| {risk.risk} | {risk.probability} | {risk.impact} | {risk.mitigation} | {risk.owner} |
{else}
No risks identified.

## Open Questions

{for each question in open_questions}
### {question.question}
- **Importance:** {question.importance}
- **Next Step:** {question.suggested_next_step}

{if empty: "No open questions."}

---
Cost: ${total_cost} | Duration: {duration} | Agents: {agent_count}
```

---

## REVIEW WORKFLOW

**Read ALL reviewer stdout files:**
- `stdout_backend-reviewer.json`
- `stdout_frontend-reviewer.json`
- `stdout_standards-reviewer.json`
- `stdout_architecture-reviewer.json`

(Read whichever exist)

**JSON Schema (per reviewer):**
```
{
  reviewer: string
  status: string
  overall_assessment: string
  summary: string
  findings: [{
    id: string
    severity: "CRITICAL" | "HIGH" | "MEDIUM" | "LOW" | "INFO"
    category: string
    file: string
    line: number
    title: string
    message: string
    impact: string
    recommendation: string
  }]
}
```

**CONFLICT RESOLUTION:**

When multiple reviewers report the same issue (same file + line + category):
1. Use the HIGHEST severity
2. Show attribution: "[CRITICAL] (reported by: backend, architecture)"
3. If severities differ by 2+ levels (e.g., CRITICAL vs LOW), add note: "⚠️ severity disputed — see individual reviews"

**Output Format:**

```markdown
# Code Review: {team_name}

## Overall Assessment

{combine overall_assessment from all reviewers}

{for each reviewer}
### {reviewer}
{summary}

## Findings by Severity

{group all findings by severity, deduplicated}

### CRITICAL ({count} findings)

{group by category}
#### {category}
{for each finding in category}
- **{file}:{line}** - {title}
  - **Message:** {message}
  - **Impact:** {impact}
  - **Recommendation:** {recommendation}
  - **Reported by:** {comma-separated reviewer list}
  {if severity disputed: "⚠️ severity disputed — see individual reviews"}

### HIGH ({count} findings)
{same structure}

### MEDIUM ({count} findings)
{same structure}

### LOW ({count} findings)
{same structure}

### INFO ({count} findings)
{same structure}

{if no findings: "✅ No issues found."}

---
Cost: ${total_cost} | Duration: {duration} | Reviewers: {reviewer_count}
```

---

## IMPLEMENTATION WORKFLOW

**Read stdout files for all workers in final wave.**

Each worker stdout has:
```
{
  content: {
    deliverables: [{
      file: string
      description: string
      phase: string (optional)
    }]
  }
}
```

**Output Format:**

```markdown
# Implementation Result: {team_name}

## Deliverables

{group deliverables by phase if present, otherwise list all}

{for each phase}
### Phase: {phase}
{for each deliverable in phase}
- **{file}**: {description}

{if no phases}
{for each deliverable}
- **{file}**: {description}

## Modified Files

{unique list of all files across all deliverables}
- {file}

---
Cost: ${total_cost} | Duration: {duration} | Workers: {worker_count}
```

---

## Error Handling

### Missing stdout file
```
Agent {agent_name} did not produce output (status: {status from config})
```

### Malformed JSON
```
Warning: Agent {agent_name} output is malformed.
Raw content excerpt:
{first 200 chars}
...
Suggest checking file manually: {file_path}
```

### Missing required fields
```
Warning: Output incomplete - missing required field: {field_name}
Displaying partial results...
```

### Team not completed
```
Team '{team_name}' is still {status}. Use /team-status to check progress.
```

---

## Formatting Conventions

**Cost:** Display with 2 decimal places and $ prefix: `$12.34`

**Duration:**
- < 60s: `"45s"`
- 60-3599s: `"2m 14s"`
- ≥ 3600s: `"1h 23m 14s"`

**Agent/Reviewer/Worker Count:** Integer count of unique agents that produced output.

---

## Implementation Notes

1. Use `Read` tool to access JSON files
2. Parse JSON with error handling for malformed content
3. For review workflow, implement deduplication by checking `file + line + category` for matching issues
4. Extract cost/duration from config.json metadata
5. Format duration using the conventions above
6. Handle missing optional fields gracefully (display "Not provided" or skip section)
