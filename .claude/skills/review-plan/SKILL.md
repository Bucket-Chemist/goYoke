---
name: review-plan
description: Critical review of implementation plans via staff architect agent
usage: /review-plan [file.md]
cost: $0.15-0.17 per invocation
---

# Review Plan Skill

## Purpose

Manual invocation of staff-architect-critical-review agent for critical review of implementation plans. Use when you want a second opinion on a plan before proceeding with implementation.

## Invocation

- `/review-plan` — Review `.claude/tmp/specs.md` (default, most common)
- `/review-plan <file.md>` — Review specific file (custom plans)

## When to Use

**Good Use Cases:**
- After `/explore` produces specs.md, before approving implementation
- Reviewing custom plan you wrote manually
- Second opinion on complex/risky architectural decisions
- Before committing to multi-day implementation effort

**When to Skip:**
- Trivial changes (single file, <50 LoC)
- Time-sensitive fixes (review adds 2-3 minutes)
- You've already reviewed manually and are confident

## Cost & Performance

- **Cost:** $0.15-0.17 per review (includes up to 2 scout spawns)
- **Duration:** 2-3 minutes (includes thinking time + scout calls)
- **Budget:** Paid from your session budget

## Workflow

### Phase 1: Validate Input

**Step 1:** Determine target file
```javascript
const targetFile = args.trim() || ".claude/tmp/specs.md"
```

**Step 2:** Validate file exists
```javascript
try {
  const fileCheck = Read({file_path: targetFile, limit: 1})
} catch (error) {
  Output(`[Error] File not found: ${targetFile}`)
  Output("\nUsage: /review-plan [file.md]")
  Output("Default: /review-plan reviews .claude/tmp/specs.md")
  exit()
}
```

**Step 3:** Read file to extract context (goal, complexity)
```javascript
const planContent = Read({file_path: targetFile, limit: 50})
// Extract goal from frontmatter or first paragraph
const goalMatch = planContent.match(/goal:(.*?)$/m)
const userGoal = goalMatch ? goalMatch[1].trim() : "Not specified"
```

---

### Phase 2: Invoke Review Agent

**Step 4:** Call staff-architect-critical-review
```javascript
Output(`[Review] Analyzing ${targetFile}...`)
Output("[Review] This will take 2-3 minutes (thinking + scout verification)")
Output(`[Review] Cost: ~$0.15-0.17`)

const reviewTask = Task({
  description: `Critical review of implementation plan: ${targetFile}`,
  subagent_type: "Explore",
  model: "sonnet",
  prompt: `AGENT: staff-architect-critical-review

1. TASK: Perform 7-layer critical review of ${targetFile}

2. EXPECTED OUTCOME:
   - review-critique.md with structured assessment
   - review-metadata.json with verdict and counts

3. REQUIRED SKILLS:
   - 7-layer critical review framework (agent.md)
   - Assumption extraction and challenge
   - Dependency graph validation
   - Failure mode analysis
   - Cost-benefit assessment
   - Testing coverage evaluation
   - Architecture smell detection
   - Contractor readiness assessment

4. REQUIRED TOOLS:
   - Read (plan file, codebase references)
   - Task (spawn haiku-scout if needed, max 2)
   - Write (review-critique.md, review-metadata.json)
   - Grep/Glob (verify claims in plan)

5. MUST DO:
   - Read entire plan file before starting critique
   - Apply ALL 7 layers in order (Assumptions → Dependencies → Failure Modes → Cost-Benefit → Testing → Architecture Smells → Contractor Readiness)
   - Quote specific sections when marking Critical issues
   - Spawn scout (max 2) if assumptions cannot be verified from plan alone
   - Include commendations for what was done well
   - Stay within 16K thinking budget
   - Generate both output files (critique + metadata)
   - Reference original user goal in critique

6. MUST NOT DO:
   - Rubber-stamp without finding issues (sharp-edge: rubber_stamping)
   - Add new requirements not in original goal (sharp-edge: scope_creep_during_review)
   - Exceed 2 scout spawns without exceptional justification
   - Mark Critical without quoting deficient section (sharp-edge: false_positive_blocking)
   - Nitpick style while missing architectural flaws (sharp-edge: missing_forest_for_trees)
   - Give vague recommendations without WHAT/WHY/HOW (sharp-edge: vague_recommendations)
   - Ignore scout data if scouts spawned (sharp-edge: ignoring_scout_data)

7. CONTEXT:
   Target file: ${targetFile}
   User goal: ${userGoal}
   Invocation: Manual (user requested via /review-plan)
   Scout metrics: ${scoutMetricsIfAvailable || "Not available"}

   Instructions for agent:
   - This is a manual review request, user wants thorough analysis
   - User will decide how to act on findings (no automatic iteration)
   - Be comprehensive but balanced (commend good practices too)
   - Focus on risks that could cause implementation failure or rework`
})
```

---

### Phase 3: Present Results

**Step 5:** Wait for review completion (with progress indicator)
```javascript
// Task() is blocking, but show user we're working
Output("[Review] Agent is analyzing plan (7-layer framework)...")
Output("[Review] Spawning scouts if verification needed...")
// reviewTask completes here
```

**Step 6:** Read outputs
```javascript
const critique = Read({file_path: ".claude/tmp/review-critique.md"})
const metadata = Read({file_path: ".claude/tmp/review-metadata.json"})
const meta = JSON.parse(metadata)
```

**Step 7:** Present critique to user
```javascript
Output("\n" + "=".repeat(80))
Output("CRITICAL REVIEW COMPLETE")
Output("=".repeat(80) + "\n")

Output(critique)

Output("\n" + "-".repeat(80))
Output("REVIEW METADATA")
Output("-".repeat(80))
Output(`Verdict: ${meta.verdict}`)
Output(`Confidence: ${meta.confidence}`)
Output(`Issues: ${meta.issue_counts.critical} Critical, ${meta.issue_counts.major} Major, ${meta.issue_counts.minor} Minor`)
Output(`Commendations: ${meta.commendations_count}`)
Output(`Scouts Used: ${meta.scouts_spawned.length}`)
Output(`Cost: $${meta.cost_estimate_usd.toFixed(2)}`)
Output(`Duration: ${meta.review_duration_minutes} minutes`)
Output(`Thinking Budget: ${meta.thinking_budget_used} / 16000 tokens`)

Output("\n" + "-".repeat(80))
Output("FILES SAVED")
Output("-".repeat(80))
Output(`Critique: .claude/tmp/review-critique.md (TTL: 7 days)`)
Output(`Metadata: .claude/tmp/review-metadata.json (TTL: 60 min)`)

Output("\n" + "=".repeat(80))
Output("NEXT STEPS")
Output("=".repeat(80))

if (meta.issue_counts.critical > 0) {
  Output(`WARNING: ${meta.issue_counts.critical} CRITICAL issues found - strongly recommend addressing before implementation`)
  Output("\nOptions:")
  Output("1. Revise plan manually to address Critical issues")
  Output("2. Re-run /explore with additional context")
  Output("3. Accept risk and proceed (document decision)")
} else if (meta.issue_counts.major > 0) {
  Output(`WARNING: ${meta.issue_counts.major} MAJOR issues found - consider addressing`)
  Output("\nOptions:")
  Output("1. Address Major issues now (reduces implementation risk)")
  Output("2. Proceed with caution (monitor these areas during implementation)")
  Output("3. Defer to post-implementation refinement")
} else {
  Output("OK: No Critical or Major issues found")
  Output("\nYou can proceed with implementation confidently.")
  if (meta.issue_counts.minor > 0) {
    Output(`\n${meta.issue_counts.minor} Minor issues noted - address when convenient.`)
  }
}

Output("\n")
```

---

## Error Handling

**File Not Found:**
```javascript
if (!fileExists) {
  Output("[Error] File not found: " + targetFile)
  Output("\nDid you run /explore yet? That creates .claude/tmp/specs.md")
  Output("\nOr specify a custom file: /review-plan path/to/custom.md")
  exit()
}
```

**Review Agent Failure:**
```javascript
try {
  const reviewTask = Task({...})
} catch (error) {
  Output("[Error] Review agent failed: " + error.message)
  Output("\nThis is rare. Possible causes:")
  Output("- Plan file is malformed (not readable)")
  Output("- Agent timeout (plan too large, >10K lines)")
  Output("- System error (retry once)")
  Output("\nRetry: /review-plan " + targetFile)
}
```

**Output File Missing:**
```javascript
if (!critiqueExists) {
  Output("[Error] Review agent did not produce critique file")
  Output("\nThis indicates an agent execution issue.")
  Output("Check: .claude/tmp/review-critique.md exists?")
  Output("\nReport this issue with: agent=staff-architect-critical-review, file=" + targetFile)
}
```

---

## Examples

### Example 1: Review Default specs.md

```bash
# After running explore
/explore "Add user authentication with OAuth"
# ... architect produces specs.md ...

# Review the plan
/review-plan

# Output:
# [Review] Analyzing .claude/tmp/specs.md...
# [Review] This will take 2-3 minutes...
# [Review] Cost: ~$0.15-0.17
# [Review] Agent is analyzing plan...
# [Review] Spawning scouts if verification needed...
#
# ================================================================================
# CRITICAL REVIEW COMPLETE
# ================================================================================
#
# [Full critique output here]
#
# WARNING: 1 CRITICAL issues found - strongly recommend addressing
```

### Example 2: Review Custom Plan

```bash
# User wrote custom plan
vim my-custom-plan.md

# Review it
/review-plan my-custom-plan.md

# Output: Same format, targeting my-custom-plan.md
```

### Example 3: File Not Found

```bash
/review-plan nonexistent.md

# Output:
# [Error] File not found: nonexistent.md
#
# Usage: /review-plan [file.md]
# Default: /review-plan reviews .claude/tmp/specs.md
```

---

## Cost Breakdown

| Component | Model | Cost |
|-----------|-------|------|
| Review agent | Sonnet+16K | ~$0.15 |
| Scout 1 (if spawned) | Haiku+2K | ~$0.01 |
| Scout 2 (if spawned) | Haiku+2K | ~$0.01 |
| **Total** | | **$0.15-0.17** |

**Budget Source:** Paid from your session budget (same pool as /explore, /commit, etc.)

---

## Limitations

**Not a Replacement for Human Review:**
- Agent catches common anti-patterns and architectural flaws
- May miss domain-specific issues requiring deep expertise
- Use as second opinion, not sole reviewer

**Limited Context:**
- Agent reviews plan file only (+ optional scout verification)
- Doesn't have full conversation history
- Provide context in plan if important

**Manual Follow-Up:**
- Agent provides critique, doesn't automatically fix issues
- You decide how to address findings
- No automatic iteration loops

---

## Tips

**Get Better Reviews:**
1. **Add context to plan:** Include user goal, constraints, assumptions in frontmatter
2. **Run after explore:** scout_metrics.json provides additional context
3. **Be specific:** Plans with specific file paths and code structure get more targeted feedback

**Interpret Results:**
- **CRITICAL:** Address before implementation (high risk of rework)
- **MAJOR:** Address if time permits (medium risk)
- **MINOR:** Note for future (low risk)
- **Commendations:** Keep doing this (patterns to replicate)

**When to Re-Review:**
- After significant plan changes (e.g., revised 3+ phases)
- If first review had CRITICAL issues and you addressed them
- Cost: $0.15 per review, budget accordingly
