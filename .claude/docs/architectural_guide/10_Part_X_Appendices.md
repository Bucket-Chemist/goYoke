# Part X: Appendices

## **Reference Material and Quick Access Resources**

## **Appendix A: Complete System Diagrams**

**==> picture [268 x 454] intentionally omitted <==**

**----- Start of picture text -----**<br>
A.1 Full Architecture Overview<br>┌────────────────────────────────────────────────────────────────────<br>│ GoGent COMPLETE ARCHITECTURE<br>│<br>└────────────────────────────────────────────────────────────────────<br> ┌─────────────────┐<br> │ USER │<br> │ │<br> │ Terminal/IDE │<br> └────────┬────────┘<br> │<br> ▼<br>┌────────────────────────────────────────────────────────────────────<br>│ ORCHESTRATION LAYER<br>│<br>│<br>│<br>│<br>┌────────────────────────────────────────────────────────────────────<br>│<br>│ │ CLAUDE CODE ORCHESTRATOR<br>│ │<br>│ │<br>│ │<br>│ │<br>┌────────────────────────────────────────────────────────────────────<br>│ │<br>│ │ │ AGENT TIER REGISTRY<br>│ │ │<br>│ │ │<br>│ │ │<br>│ │ │ ┌─────────┐ ┌─────────┐ ┌─────────┐<br>┌─────────────────┐ │ │ │<br>│ │ │ │ HAIKU │ │ SONNET │ │ OPUS │ │ GEMINI<br>│ │ │ │<br>│ │ │ │ │ │ │ │ │ │ EXTERNAL<br>│ │ │ │<br>│ │ │ │ Scout │ │ Primary │ │ Complex │ │<br>│ │ │ │<br>│ │ │ │ Quick │ │ Work │ │ Tasks │ │ 1M Context<br>│ │ │ │<br>│ │ │ │ │ │ │ │ │ │ Large<br>Files │ │ │ │<br>│ │ │ │ $0.25/M │ │ $3/M │ │ $15/M │ │ ~$0.075/M<br>│ │ │ │<br>│ │ │ └─────────┘ └─────────┘ └─────────┘<br>└─────────────────┘ │ │ │<br>│ │ │<br>│ │ │<br>│ │<br>└────────────────────────────────────────────────────────────────────<br>│ │<br>│ │<br>│ │<br>│ │<br>┌────────────────────────────────────────────────────────────────────<br>│ │<br>│ │ │ SPECIALIZED SUBAGENTS<br>│ │ │<br>│ │ │<br>│ │ │<br>**----- End of picture text -----**<br>

**==> picture [268 x 533] intentionally omitted <==**

**----- Start of picture text -----**<br>

**==> picture [265 x 125] intentionally omitted <==**

**----- Start of picture text -----**<br>
A.2 Request Lifecycle Flow<br>┌────────────────────────────────────────────────────────────────────<br>│ REQUEST LIFECYCLE<br>│<br>└────────────────────────────────────────────────────────────────────<br>USER REQUEST<br> │<br> ▼<br>┌────────────────────────────────────────────────────────────────────<br>│ PHASE 1: INTENT CLASSIFICATION<br>│<br>│<br>│<br>**----- End of picture text -----**<br>

│ • Parse request

│

│ • Determine if explore workflow needed

│

│ • Identify task type and scope

│

└────────────────────────────────────────────────────────────────────

│ ▼

┌────────────────────────────────────────────────────────────────────

│ PHASE 2: SCOUT (Conditional)

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│ • Haiku agent enumerates scope

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│ • Count files, estimate tokens

│

│ • Identify module boundaries

│

│ • Output: scout_metrics.json

│

└────────────────────────────────────────────────────────────────────

│

▼

┌────────────────────────────────────────────────────────────────────

│ PHASE 3: COMPLEXITY CALCULATION

│

│

│

│ SCORE = (tokens/10000) + (files/5) + (modules×2)

**==> picture [4 x 6] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│

│

│ Output: complexity_score, recommended_tier

│

└────────────────────────────────────────────────────────────────────

│

▼

┌────────────────────────────────────────────────────────────────────

│ PHASE 4: ROUTING ENFORCEMENT

│

│

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│ PreToolUse.sh executes: │

│ • Check force-tier override

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│ • Validate against ceiling

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│ • Log decision

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│ • Exit 0 (PERMIT) or Exit 2 (BLOCK)

│

└────────────────────────────────────────────────────────────────────

│ ├──── BLOCK ────► Return to user with explanation │ ▼ PERMIT

┌────────────────────────────────────────────────────────────────────

│ PHASE 5: EXECUTION

│

│ │

│ Selected agent executes task

**==> picture [4 x 6] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│ • Each tool call triggers Pre/PostToolUse

│

│ • SubagentStop validates handoffs

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│ • Results captured

│

└────────────────────────────────────────────────────────────────────

│

▼

┌────────────────────────────────────────────────────────────────────

│ PHASE 6: MEMORY ARCHIVAL

│

│

│

│ • PostToolUse logs outcomes

│

│ • SessionEnd triggers archival

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│ • Memory archivist processes decisions

│

│ • Sharp edges captured

**==> picture [265 x 126] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>└────────────────────────────────────────────────────────────────────<br> │<br> ▼<br>OUTPUT TO USER<br>A.3 Weekly Review Process<br>┌────────────────────────────────────────────────────────────────────<br>│ WEEKLY REVIEW PROCESS<br>│<br>└────────────────────────────────────────────────────────────────────<br>TRIGGER: Monday 9 AM or manual /weekly-review<br>**----- End of picture text -----**<br>

**==> picture [265 x 527] intentionally omitted <==**

**----- Start of picture text -----**<br>
┌────────────────────────────────────────────────────────────────────<br>│ PHASE 1: AGGREGATION<br>│<br>│<br>│<br>│ Memory Synthesis Agent (Sonnet) collects:<br>│<br>│ • decisions/_.md (last 7 days)<br>│<br>│ • sharp-edges/_.md (last 7 days)<br>│<br>│ • session*summaries/\*.json<br>│<br>│ • routing_log*_.jsonl<br>│<br>│ • observations/_.jsonl<br>│<br>│<br>│<br>│ Output: weekly_synthesis.json<br>│<br>└────────────────────────────────────────────────────────────────────<br> │<br> ▼<br>┌────────────────────────────────────────────────────────────────────<br>│ PHASE 2: GAP ANALYSIS<br>│<br>│<br>│<br>│ Systems Architect Agent (Opus, extended thinking) analyzes:<br>│<br>│ • Recurring failure patterns (≥3 occurrences)<br>│<br>│ • Tasks exceeding budgets<br>│<br>│ • Human override patterns<br>│<br>│ • Context boundary issues<br>│<br>│<br>│<br>│ For each gap: config change OR new subagent?<br>│<br>│<br>│<br>│ Output: architect_report.md, recommendations.json<br>│<br>└────────────────────────────────────────────────────────────────────<br> │<br> ▼<br>┌────────────────────────────────────────────────────────────────────<br>│ PHASE 3: SCHEMA DISCOVERY (if observations ≥ 200)<br>│<br>│<br>│<br>│ Schema Discovery Agent analyzes observation patterns:<br>│<br>│ • Cluster detection (≥10 instances per cluster)<br>│<br>│ • Statistical validation (Silhouette > 0.5)<br>│<br>│ • Schema proposal generation<br>│<br>│<br>│<br>│ Output: pattern_analysis.json, proposed_schema (if patterns found)<br>│<br>└────────────────────────────────────────────────────────────────────<br> │<br> ▼<br>┌────────────────────────────────────────────────────────────────────<br>**----- End of picture text -----**<br>

│ PHASE 4: HUMAN INTERVIEW │ │ │ │ Orchestrator presents findings: │ │ • Gap analysis results │ │ • Recommendations with [Approve] [Modify] [Reject] │ │ • Schema proposals (if any) │ │ • Autonomy level assessments │ │ │ │ Human reviews and decides │ └──────────────────────────────────────────────────────────────────── │ ▼ ┌──────────────────────────────────────────────────────────────────── │ PHASE 5: IMPLEMENTATION │ │ │ │ Approved changes applied: │ │ • Config updates to routing-schema.json │ │ • New agents created in .claude/agents/ │ │ • Schemas activated (if approved) │ │ • Autonomy levels adjusted │ │ │ │ All changes logged in memory/decisions/ │

└────────────────────────────────────────────────────────────────────

**==> picture [209 x 6] intentionally omitted <==**

**==> picture [129 x 9] intentionally omitted <==**

**----- Start of picture text -----**<br>
A.4 Autonomy Progression Model<br>**----- End of picture text -----**<br>

┌────────────────────────────────────────────────────────────────────

**==> picture [265 x 306] intentionally omitted <==**

**----- Start of picture text -----**<br>
│ AUTONOMY LEVEL PROGRESSION<br>│<br>└────────────────────────────────────────────────────────────────────<br> INCREASING AUTOMATION<br> ─────────────────────►<br>┌─────────────┐ ┌─────────────┐ ┌─────────────┐<br>┌─────────────┐ ┌─────────────┐<br>│ LEVEL 1 │ │ LEVEL 2 │ │ LEVEL 3 │ │ LEVEL 4<br>│ │ LEVEL 5 │<br>│ │ │ │ │ │ │<br>│ │ │<br>│ OPERATOR │──►│COLLABORATOR │──►│ CONSULTANT │──►│ APPROVER<br>│──►│ OBSERVER │<br>│ │ │ │ │ │ │<br>│ │ │<br>│ Human does │ │ System │ │ System auto-│ │ System<br>│ │ System runs │<br>│ everything │ │ suggests, │ │ approves │ │ handles<br>│ │ autonomously│<br>│ │ │ human │ │ high-conf │ │ routine,<br>│ │ in defined │<br>│ System │ │ confirms │ │ decisions │ │ human<br>│ │ domain │<br>│ observes │ │ │ │ │ │ reviews<br>│ │ │<br>│ │ │ │ │ │ │ summary<br>│ │ Human │<br>│ 0% auto │ │ 0% auto │ │ ~60% auto │ │ ~85% auto<br>│ │ ~95% auto │<br>└─────────────┘ └─────────────┘ └─────────────┘<br>└─────────────┘ └─────────────┘<br> │ │ │ │<br>│<br> │ │ │ │<br>│<br> ▼ ▼ ▼ ▼<br>▼<br> 100 decisions 200 decisions 500 decisions 1000+<br>decisions Domain-specific<br> captured 95% success 98% success 99%+ success<br>Human approves<br>**----- End of picture text -----**<br>

## **Appendix B: Threshold Quick Reference**

## **B.1 Routing Thresholds**

| **Parameter**  |                  | **Value**     | **Formula/Rule** |
| -------------- | ---------------- | ------------- | ---------------- |
| Complexity     | (tokens/10000)   | + (files/5) + |                  |
| score          | (modules×2)      |               |                  |
| Haiku ceiling  | Score < 2        |               |                  |
| Sonnet ceiling | Score 2-10       |               |                  |
| Opus foor      | Score > 10       |               |                  |
| Gemini trigger | Tokens > 50,000  |               |                  |
| Gemini force   | Tokens > 100,000 |               |                  |

## **B.2 Cost Thresholds**

| **Parameter**        | **Value**   | **Action**          |
| -------------------- | ----------- | ------------------- |
| Session warning      | $5          | Alert user          |
| Session limit        | $10         | Require confrmation |
| Weekly budget        | $50         | Review trigger      |
| Weekly review budget | $2.50       | Infrastructure cost |
| Shadow cost ceiling  | 2× expected | Auto-terminate      |

## **B.3 Pattern Detection Thresholds**

| **Parameter**       | **Value**    | **Purpose**        |
| ------------------- | ------------ | ------------------ |
| Min observations    | 200          | Trigger analysis   |
| Per-cluster minimum | 30           | Statistical power  |
| Silhouette score    | > 0.5        | Cluster validity   |
| Bootstrap stability | 80%          | Reproducibility    |
| Confdence level     | 99% (α=0.01) | Decision threshold |

## **B.4 Autonomy Thresholds**

| **Transition** | **Decisions** | **Success Rate** |
| -------------- | ------------- | ---------------- |
| L1 → L2        | 100           | N/A              |
| L2 → L3        | 200           | 95%              |
| L3 → L4        | 500           | 98%              |
| L4 → L5        | 1000+         | 99%+             |

## **B.5 Shadow Deployment Thresholds**

| **Parameter**               | **Value**       |
| --------------------------- | --------------- |
| Min invocations             | 10              |
| Success rate                | ≥ 90%           |
| Error rate vs baseline      | ≤ 0.1% increase |
| Max duration                | 14 days         |
| **B.6 Document Processing** |                 |
| **Parameter**               | **Value**       |
| Chunk size (default)        | 512 tokens      |
| Chunk overlap               | 15%             |
| Max parallel workers        | 5               |
| Overlap window size         | 40,000 tokens   |

## **Appendix C: Glossary**

| **Term**             | **Defnition**                                                                    |
| -------------------- | -------------------------------------------------------------------------------- |
| **Agent**            | An LLM confgured with specifc instructions<br>and constraints for a task type    |
| **Apprenticeship**   | Learning to replicate human decision-                                            |
| **Learning**         | making through observation                                                       |
| **BM25**             | Best Matching 25, a probabilistic ranking<br>function for text retrieval         |
| **Complexity Score** | Numeric value calculated from tokens, fles,<br>and modules to determine routing  |
| **Emergent Schema**  | A data schema discovered from accumulated<br>observations rather than pre-defned |
| **Explore Workfow**  | 7-phase structured process for investigating<br>and modifying codebases          |
| **Gap Analysis**     | Systematic identifcation of capability<br>defciencies based on observed failures |
| **Handof**           | Transfer of task context and state between<br>agents                             |
|                      | Bash script executed at specifc lifecycle                                        |

points (PreToolUse, PostToolUse, etc.)

## **Hook**

| **Hook**             | points (PreToolUse, PostToolUse, etc.)                                        |
| -------------------- | ----------------------------------------------------------------------------- |
| **HITL**             | Human-in-the-Loop; pattern where humans<br>approve or reject agent proposals  |
| **Memory Archivist** | Agent responsible for processing session<br>outcomes into persistent memory   |
| **Observation**      | Raw behavioral event captured for pattern<br>analysis                         |
| **Routing**          | Decision process for selecting which agent<br>tier handles a task             |
| **Scout**            | Haiku agent performing reconnaissance on<br>codebase before main task         |
| **Shadow**           | Running new agent in parallel without                                         |
| **Deployment**       | afecting production, for validation                                           |
| **Sharp Edge**       | Known pitfall or unexpected behavior<br>documented for future avoidance       |
| **Subagent**         | Automatic generation of new agent                                             |
| **Spawning**         | defnitions from gap analysis                                                  |
| **Tier**             | Model capability level (Haiku < Sonnet <<br>Opus)                             |
| **Weekly Review**    | Automated process for analyzing system<br>behavior and proposing improvements |

## **Appendix D: Research Source Index**

## **Primary Research Documents**

| **ID** | **Title**                                               | **Key Findings Used**                                                                                          |
| ------ | ------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------- |
| R1     | Hybrid Multi-Agent<br>Architecture<br>Evaluation        | MAST taxonomy (41-86.7% failure<br>rate), RouteLLM (85% cost<br>reduction), BM25 recall<br>benchmarks          |
| R2     | Evidence-Based<br>Thresholds for Multi-<br>Agent Design | Pattern detection thresholds (30×k<br>formula), autonomy progression<br>criteria, shadow deployment<br>metrics |
| R3     | LangChain/LangGraph<br>vs Bespoke RAG<br>Analysis       | Framework overhead (14ms<br>latency, 50% token increase),<br>selective adoption<br>recommendations             |

## **External Sources Referenced**

| **Topic**                   | **Source**                  | **Finding**                                    |
| --------------------------- | --------------------------- | ---------------------------------------------- |
| BM25 vs<br>Embeddings       | XetHub Benchmark<br>2024    | 89% vs 91.7% recall                            |
| Chunk Overlap               | NVIDIA<br>FinanceBench 2024 | 15% optimal                                    |
| Multi-Agent<br>Coordination | MAST Taxonomy               | 35% failures from inter-<br>agent misalignment |
|                             |                             | 85% cost reduction,                            |
| Routing Eficiency           | RouteLLM Paper              | 95% performance                                |
|                             |                             | retention                                      |
| Context<br>Compression      | ACON Framework              | 26-54% reduction,<br>95%+ accuracy             |
| Autonomy Levels             | Sheridan & Verplank         | 5-level automation<br>taxonomy                 |

## **Appendix E: Verification Commands**

## **E.1 Installation Verification**

_# Verify Claude Code installation_ claude --version

- _# Verify hooks directory exists_ ls -la ~/.claude/hooks/

_# Verify hook executability_ **[[** -x ~/.claude/hooks/PreToolUse.sh **]] &&** echo "PreToolUse OK" **||** echo "PreToolUse NOT EXECUTABLE"

_# Verify required tools_ command -v jq >/dev/null **&&** echo "jq OK" **||** echo "jq MISSING" command -v bc >/dev/null **&&** echo "bc OK" **||** echo "bc MISSING"

_# Verify telemetry enabled_ env **|** grep CLAUDE_CODE_ENABLE_TELEMETRY

## **E.2 Component Health Checks**

_# Test complexity calculation_ echo '{"scout_report":{"scope_metrics":

**==> picture [226 x 162] intentionally omitted <==**

**----- Start of picture text -----**<br>
{"total_files":10,"estimated_tokens":25000},"complexity_signals":<br>{"module_count":2}}}' > ~/.claude/tmp/scout_metrics.json<br>~/.claude/scripts/calculate-complexity.sh<br>cat ~/.claude/tmp/complexity_score # Should output ~7.5<br># Test SubagentStop validation<br>echo '{"status":"completed","context":<br>{"task_summary":"test","success_criteria":["done"]}}' ><br>~/.claude/tmp/handoff.json<br>~/.claude/hooks/SubagentStop.sh test_agent<br>echo "Exit code: $?" # Should be 0<br># Test memory query<br>python3 ~/.claude/scripts/query-memory-bm25.py "test query" --json<br>E.3 State File Validation<br># Validate all state files<br>python3 ~/.claude/scripts/validate-state.py<br># Check routing log integrity<br>jq empty ~/.claude/tmp/routing_log.jsonl 2>/dev/null && echo<br>"Routing log valid" || echo "Routing log invalid"<br>**----- End of picture text -----**<br>

**==> picture [156 x 73] intentionally omitted <==**

**----- Start of picture text -----**<br>

# Verify memory file format<br>for f in ~/.claude/memory/**/\*.md ; do<br>if head -1 "$f" | grep -q "^---$" ; then<br>echo "✓ $f has frontmatter"<br>else<br>echo "✗ $f missing frontmatter"<br>fi<br>done<br>E.4 Integration Tests<br>**----- End of picture text -----\*\*<br>

**==> picture [125 x 41] intentionally omitted <==**

**----- Start of picture text -----**<br>

# Full hook chain test<br>~/.claude/scripts/calculate-complexity.sh<br>~/.claude/hooks/PreToolUse.sh<br>echo "PreToolUse exit: $?"<br>~/.claude/hooks/PostToolUse.sh<br>echo "PostToolUse exit: $?"<br>**----- End of picture text -----**<br>

**==> picture [232 x 162] intentionally omitted <==**

**----- Start of picture text -----**<br>

# Memory retrieval test<br>python3 ~/.claude/scripts/query-memory-bm25.py "authentication" -k 3<br># Cost report generation<br>~/.claude/scripts/generate-cost-report.sh<br>E.5 Diagnostic Commands<br># View recent routing decisions<br>tail -20 ~/.claude/tmp/routing_log.jsonl | jq .<br># Summarize session costs<br>jq -s 'map(.cost.estimated_cost_usd // 0) | add'<br>~/.claude/tmp/routing_log.jsonl<br># Count memory files<br>find ~/.claude/memory -name "\*.md" | wc -l<br># Check agent registry<br>cat ~/.claude/agents-index.json | jq '.[] | {name, status}'<br># View autonomy levels<br>cat ~/.claude/autonomy-levels.yaml<br>**----- End of picture text -----**<br>

## **Appendix F: File Templates**

**F.1 Memory File Template**

---

title **:** "[Brief descriptive title]" created **:** YYYY-MM-DD category **:** decisions _# or sharp-edges, facts, preferences_ tags **: [** tag1 **,** tag2 **,** tag3 **]** status **:** active summary **:** "One-line searchable summary" ---

_## Context_ **[** Why this was created **]** _## Content_

**[** Main content **]**

_## Related_

**- [[** related-file-1.md **]]**

**- [[** related-file-2.md **]]**

## **F.2 Agent Definition Template**

_# agent.yaml_ name **:** agent-name version **:** 1.0.0 status **:** proposed tier **:** sonnet purpose **:** | Brief description of what this agent does. triggers **: -** condition **:** "trigger condition" configuration **:** key **:** value constraints **:** max_input_tokens **:** 50000 max_output_tokens **:** 10000 timeout_minutes **:** 5

**==> picture [102 x 9] intentionally omitted <==**

**----- Start of picture text -----**<br>
F.3 Decision Log Template<br>**----- End of picture text -----**<br>

--title **:** "Decision: [Topic]" created **:** YYYY-MM-DD category **:** decisions tags **: [** architecture **,** topic **]** status **:** active summary **:** "One-line summary" --- _## Context_ Why this decision was needed. _## Options Considered_ 1. Option A 2. Option B

_## Decision_ What was chosen. _## Rationale_ Why this choice. _## Consequences_ What changes.

**==> picture [129 x 36] intentionally omitted <==**

**----- Start of picture text -----**<br>
Document Version History<br>Version Date Changes<br>1.0.0 2026-01-13 Initial release<br>**----- End of picture text -----**<br>

_End of GoGent Architecture & Implementation Guide_
