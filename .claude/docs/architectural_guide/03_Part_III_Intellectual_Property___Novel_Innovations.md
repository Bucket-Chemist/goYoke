# Part III: Intellectual Property & Novel Innovations

## **Four Architectural Patterns Representing Significant Innovation**

## **Overview**

GoGent introduces four architectural innovations that, to our knowledge, have no direct precedent in the multi-agent systems literature or commercial frameworks. These innovations address fundamental limitations in current approaches to AI agent orchestration:

| **Innovation**                  | **Problem**<br>**Addressed**                     | **Current**<br>**Approaches**               | **Our**<br>**Solution**                                    |
| ------------------------------- | ------------------------------------------------ | ------------------------------------------- | ---------------------------------------------------------- |
| Emergent<br>Schema<br>Discovery | Pre-defned<br>schemas<br>constrain<br>learning   | RLHF with fxed<br>feedback<br>categories    | Discover<br>schemas<br>from<br>accumulated<br>observations |
| Subagent<br>Spawning            | Fixed agent<br>populations limit<br>adaptability | Manual agent<br>creation                    | Generate<br>agents from<br>gap analysis                    |
|                                 |                                                  |                                             | System                                                     |
| Apprenticeship<br>Learning      | Human-in-the-<br>loop doesn’t<br>scale           | Humans<br>approve/reject<br>agent proposals | learns to<br>replicate<br>human                            |
|                                 |                                                  |                                             | decisions                                                  |
| Code-Enforced                   | LLM routing is                                   | LLM-based or                                | Bash<br>arithmetic                                         |
| Routing                         | non-<br>deterministic                            | keyword routing                             | enforces<br>routing                                        |

Each innovation is detailed below with architectural specifications, implementation guidance, novelty assessment, and competitive differentiation.

## **Innovation 1: Emergent Schema Discovery**

## **1.1 Concept Definition**

**Traditional approach:** Define a schema for capturing user feedback or system decisions, then collect data against that schema. The schema constrains what can be learned.

**Our approach:** Accumulate raw behavioral observations without schema constraints. Periodically analyze accumulated observations for patterns. When statistically significant clusters emerge, propose a schema that describes the discovered patterns. Human approves before activation.

This is **unsupervised learning applied to system architecture evolution** . The system discovers its own feedback categories rather than being told what to measure.

## **1.2 Architectural Specification**

┌────────────────────────────────────────────────────────────────────

│ EMERGENT SCHEMA DISCOVERY PIPELINE │

└────────────────────────────────────────────────────────────────────

PHASE 1: RAW OBSERVATION ACCUMULATION

┌────────────────────────────────────────────────────────────────────

- │

│ │ Every significant event is captured as a structured observation: │

│

│

- │ .claude/memory/observations/

│

- │ ├── 2026-01-13-session-001.jsonl

│

- │ ├── 2026-01-13-session-002.jsonl

│

- │ └── ...

│

- │

│

- │ Observation format (minimal schema, maximal flexibility):

│

- │ {

│

- │ "timestamp": "2026-01-13T10:23:45Z",

│

- │ "event_type": "human_override",

│

- │ "context": {

│

- │ "task": "refactor authentication",

│

│ "complexity_score": 8.5,

**==> picture [262 x 253] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>│ "recommended_tier": "sonnet",<br>│<br>│ "files_in_scope": 12<br>│<br>│ },<br>│<br>│ "action": {<br>│<br>│ "type": "tier_override",<br>│<br>│ "from": "sonnet",<br>│<br>│ "to": "opus",<br>│<br>│ "reason_provided": "security-critical code"<br>│<br>│ },<br>│<br>│ "outcome": {<br>│<br>│ "success": true,<br>│<br>│ "duration_seconds": 180<br>│<br>│ }<br>│<br>│ }<br>│<br>│<br>│<br>│ Key principle: Capture WHAT happened, not what category it<br>belongs to. │<br>│ Categories emerge later.<br>│<br>│<br>│<br>**----- End of picture text -----**<br>

**==> picture [265 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
└────────────────────────────────────────────────────────────────────<br>**----- End of picture text -----**<br>

**==> picture [262 x 171] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br> │ Trigger: 200+<br>observations<br> │ OR manual /schema-<br>discovery<br> ▼<br>PHASE 2: PATTERN DETECTION<br>┌────────────────────────────────────────────────────────────────────<br>│<br>│<br>│ Schema Discovery Agent (Sonnet with extended thinking)<br>│<br>│<br>│<br>│ Input: All observations from past N days/sessions<br>│<br>│<br>│<br>│ Task:<br>│<br>│ "Analyze these behavioral observations. Identify recurring<br>patterns. │<br>│ A valid pattern requires:<br>│<br>**----- End of picture text -----**<br>

┌────────────────────────────────────────────────────────────────────

│ - At least 10 instances

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│ - Distinguishable characteristics from other patterns

**==> picture [192 x 21] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>│ - Predictive value (pattern membership predicts outcome or<br>action) │<br>**----- End of picture text -----**<br>

│

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│ For each discovered pattern, provide:

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│ - Proposed name

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│ - Defining characteristics

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│ - Instance count

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│ - Confidence score

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│ - Suggested schema fields to capture this pattern"

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│ Statistical validation:

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│ - Minimum 30 observations per cluster for 80% statistical power

**==> picture [4 x 34] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>│<br>│<br>**----- End of picture text -----**<br>

│ - Silhouette score > 0.5 for cluster validity

│ - Bootstrap stability > 80% reproducibility

**==> picture [262 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│ Output: pattern_analysis.json

**==> picture [265 x 424] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>│ {<br>│<br>│ "analysis_date": "2026-01-13",<br>│<br>│ "observations_analyzed": 234,<br>│<br>│ "patterns_discovered": [<br>│<br>│ {<br>│<br>│ "name": "scope_reduction",<br>│<br>│ "instance_count": 47,<br>│<br>│ "confidence": 0.87,<br>│<br>│ "characteristics": [<br>│<br>│ "Human modified plan to reduce scope",<br>│<br>│ "Original scope > 5 files",<br>│<br>│ "Reduction typically 40-60%"<br>│<br>│ ],<br>│<br>│ "suggested_schema_fields": {<br>│<br>│ "original_scope": "integer",<br>│<br>│ "final_scope": "integer",<br>│<br>│ "reduction_reason": "enum[complexity|time|risk]"<br>│<br>│ }<br>│<br>│ },<br>│<br>│ {<br>│<br>│ "name": "tier_override_up",<br>│<br>│ "instance_count": 31,<br>│<br>│ ...<br>│<br>│ }<br>│<br>│ ],<br>│<br>│ "schema_proposal": { ... }<br>│<br>│ }<br>│<br>│<br>│<br>└────────────────────────────────────────────────────────────────────<br> │<br> │ Human review required<br> ▼<br>**----- End of picture text -----**<br>

PHASE 3: SCHEMA PROPOSAL & APPROVAL

**==> picture [265 x 21] intentionally omitted <==**

**----- Start of picture text -----**<br>
┌────────────────────────────────────────────────────────────────────<br>│<br>**----- End of picture text -----**<br>

**==> picture [4 x 20] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>│<br>**----- End of picture text -----**<br>

│ Orchestrator presents findings to human:

│

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│ " � SCHEMA DISCOVERY RESULTS

**==> picture [4 x 6] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│ I analyzed 234 observations and found 4 distinct patterns:

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│

**==> picture [4 x 6] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│ 1. scope_reduction (47 instances, 87% confidence)

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│ You frequently reduce task scope when > 5 files involved

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│

**==> picture [4 x 6] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│ 2. tier_override_up (31 instances, 82% confidence)

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│ You escalate to Opus for security-related code

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│

│

│ 3. clarification_request (28 instances, 79% confidence)

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│ You ask for clarification on ambiguous requirements

│

│

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

- │ 4. quick_approval (89 instances, 94% confidence)

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

- │ Simple tasks are approved without modification

**==> picture [4 x 6] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

- │

**==> picture [80 x 21] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>patterns │<br>**----- End of picture text -----**<br>

- │ I propose activating a decision schema to capture these

- │ more precisely. This will enable me to:

**==> picture [4 x 6] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

- │ - Predict when you'll want scope reduction

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

- │ - Auto-escalate security-critical code │

- │ - Eventually automate quick_approval decisions

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

- │

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

- │ [View full schema] [Approve] [Modify] [Reject]"

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

- │

**==> picture [186 x 34] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>│ Human response determines next action:<br>│<br>│ - Approve: Schema activated, future observations captured<br>against it │<br>**----- End of picture text -----**<br>

- │ Human response determines next action:

- │ - Modify: Human edits schema, then activates

**==> picture [265 x 75] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>│ - Reject: Continue raw observation accumulation<br>│<br>│<br>│<br>└────────────────────────────────────────────────────────────────────<br> │<br> │ On approval<br> ▼<br>PHASE 4: SCHEMA ACTIVATION & BACKFILL<br>**----- End of picture text -----**<br>

┌────────────────────────────────────────────────────────────────────

- │

**==> picture [4 x 6] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

- │ 1. Create schema definition file:

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

- │ .claude/schemas/decisions-v1.0.json

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

- │

**==> picture [4 x 6] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

- │ 2. Update observation capture to use new schema

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

- │

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

- │ 3. Backfill existing observations (async, Haiku-tier):

**==> picture [186 x 34] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>│ - Classify each historical observation into discovered<br>patterns │<br>│ - Add pattern_classification field<br>│<br>**----- End of picture text -----**<br>

- │ - Add pattern_classification field

- │ - Maintain original raw data

**==> picture [4 x 6] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

- │

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

- │ 4. Enable pattern-based features:

- │ │ - Predictive suggestions ("Based on pattern, you may want to...") │

- │ - Confidence tracking per pattern

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

- │ - Autonomy level candidates identified

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

- │

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

└────────────────────────────────────────────────────────────────────

**==> picture [209 x 6] intentionally omitted <==**

## **1.3 Novelty Assessment**

**Existing approaches in RLHF/feedback systems:**

| **System**            | **Schema Approach**                              | **Limitation**                            |
| --------------------- | ------------------------------------------------ | ----------------------------------------- |
| OpenAI RLHF           | Pre-defned<br>helpfulness/harmlessness<br>scales | Categories fxed at<br>design time         |
| Constitutional<br>AI  | Pre-defned principles                            | Principles must be<br>articulated upfront |
| LangSmith<br>feedback | User-defned categories                           | Categories manually<br>specifed           |
| Thumbs<br>up/down     | Binary feedback                                  | No nuance, no<br>category discovery       |

**Our differentiation:**

1. **No a priori categories** — The system discovers what matters by observing behavior

2. **Statistical validation** — Patterns must meet significance

- thresholds, not just exist

3. **Human approval gate** — Discovery is automated, activation is human-controlled

4. **Backfill capability** — Historical data becomes more valuable over time

**Prior art search findings:** No published systems implement emergent schema discovery for agent feedback. The closest analog is unsupervised topic modeling (LDA, etc.) applied to text, but not applied to behavioral observation streams in agent systems.

- **1.4 Competitive Advantage**

1. **Adapts to any workflow** — The system learns YOUR patterns, not generic patterns

2. **Reveals unknown unknowns** — Discovers behavioral patterns the user wasn’t aware of

3. **Reduces schema maintenance** — No need to manually update feedback categories

4. **Enables organic evolution** — New patterns emerge as usage evolves

- **1.5 Implementation Priority**

**Phase 4 in roadmap (Months 4-5)**

Dependencies: - Observation capture infrastructure (Phase 2) - Memory file standardization (Phase 2) - Weekly review process (Phase 3)

## **Innovation 2: Gap Analysis → Subagent Spawning**

- **2.1 Concept Definition**

**Traditional approach:** Multi-agent systems have fixed agent populations defined at design time. Adding capabilities requires manual agent development.

**Our approach:** Weekly architecture review identifies capability gaps from actual usage patterns. For gaps that cannot be resolved by configuration changes, the system generates complete subagent definitions ready for deployment. Humans approve before activation.

This creates a **generative architecture** — the system grows capabilities based on observed deficiencies rather than anticipated needs.

## **2.2 Architectural Specification**

┌────────────────────────────────────────────────────────────────────

│ GAP ANALYSIS → SUBAGENT SPAWNING │

└────────────────────────────────────────────────────────────────────

TRIGGER: Weekly review OR manual /gap-analysis command PHASE 1: GAP IDENTIFICATION

┌────────────────────────────────────────────────────────────────────

- │

│ │

- │ Systems Architect Agent (Opus-tier, extended thinking)

- │

│ │

- │ Inputs:

- │ - Past week's session logs

│

- │ - Memory/decisions/ and sharp-edges/

│

- │ - Routing violations log

│

- │ - Human override history

│

- │ - Cost breakdown by tier

│

- │

│

- │ Analysis tasks:

│

- │ 1. Identify recurring failure patterns (≥3 occurrences)

│

- │ 2. Flag tasks exceeding time/cost budgets repeatedly

│ │

- │ 3. Detect human overrides suggesting missing capabilities

- │ 4. Find context boundary issues (synthesis gaps)

│ │

│ 5. Identify routing inefficiencies

│

**==> picture [262 x 75] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>│ For each gap, determine:<br>│<br>│ - Can this be resolved by configuration change?<br>│<br>│ - Does this require new subagent capability?<br>│<br>│ - Is this a one-off or recurring pattern?<br>│<br>│<br>│<br>**----- End of picture text -----**<br>

└────────────────────────────────────────────────────────────────────

│

▼

PHASE 2: RECOMMENDATION GENERATION

┌────────────────────────────────────────────────────────────────────

│

│ │ Gap Classification: │ │ │ │ ┌──────────────────────────────────────────────────────────────────── │ │ │ Gap Type │ Resolution │ Output │ │ │ ├───────────────────────┼─────────────────────────┼────────────────── │ │ │ Routing threshold │ Config change │ routingschema.json diff │ │ │ │ Missing agent type │ Subagent spawn │ Agent definition files │ │ │ │ Memory retrieval │ Query enhancement │ querymemory.sh update │ │ │ │ Context limits │ Gemini integration │ Protocol addition │ │ │ │ User workflow │ Skill addition │ SKILL.md template │ │ │

└────────────────────────────────────────────────────────────────────

│

│

│ │ For subagent spawning, generate: │ │ .claude/agents/[agent-name]/ │ │ ├── agent.md # Agent definition and purpose │ │ ├── agent.yaml # Configuration (tier, triggers, constraints) │ │ └── CLAUDE.md # Instructions for this agent │ │ │ └──────────────────────────────────────────────────────────────────── │ ▼

PHASE 3: HUMAN REVIEW & APPROVAL

┌────────────────────────────────────────────────────────────────────

│

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

**==> picture [262 x 137] intentionally omitted <==**

**----- Start of picture text -----**<br>
│ Orchestrator presents recommendations:<br>│<br>│<br>│<br>│ " � GAP ANALYSIS RESULTS<br>│<br>│<br>│<br>│ This week I identified 3 capability gaps:<br>│<br>│<br>│<br>│ GAP 1: Document synthesis failures (4 occurrences)<br>│<br>│ Root cause: Context boundaries causing information loss<br>│<br>│ Recommendation: Spawn 'parallel-summarizer' subagent swarm<br>│<br>│ [View agent definition] [Approve] [Modify] [Reject]<br>│<br>**----- End of picture text -----**<br>

│

│

│ GAP 2: Excessive Opus routing (12 tasks could have been Sonnet)

│

│ Root cause: Complexity threshold too aggressive

**==> picture [4 x 34] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>│<br>│<br>**----- End of picture text -----**<br>

│ Recommendation: Raise opus threshold from 10 to 12

│ [View diff] [Approve] [Modify] [Reject]

│

**==> picture [262 x 76] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>│ GAP 3: Memory retrieval misses (3 relevant decisions not found)<br>│<br>│ Root cause: Keyword matching insufficient for conceptual queries<br>│<br>│ Recommendation: Add BM25 indexing to query-memory.sh<br>│<br>│ [View implementation] [Approve] [Modify] [Reject]"<br>│<br>│<br>│<br>**----- End of picture text -----**<br>

└────────────────────────────────────────────────────────────────────

│

▼

PHASE 4: SHADOW DEPLOYMENT (For new agents)

┌────────────────────────────────────────────────────────────────────

│

**==> picture [262 x 21] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>│ New agents enter shadow mode before production:<br>│<br>**----- End of picture text -----**<br>

│

**==> picture [262 x 260] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>│ agents-index.json entry:<br>│<br>│ {<br>│<br>│ "parallel-summarizer": {<br>│<br>│ "status": "shadow",<br>│<br>│ "created": "2026-01-13",<br>│<br>│ "shadow_start": "2026-01-13",<br>│<br>│ "promotion_criteria": {<br>│<br>│ "min_invocations": 10,<br>│<br>│ "success_rate_threshold": 0.9,<br>│<br>│ "max_shadow_duration_days": 14<br>│<br>│ },<br>│<br>│ "shadow_metrics": {<br>│<br>│ "invocations": 0,<br>│<br>│ "successes": 0,<br>│<br>│ "failures": 0<br>│<br>│ }<br>│<br>│ }<br>│<br>│ }<br>│<br>│<br>**----- End of picture text -----**<br>

**==> picture [4 x 6] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│ Shadow behavior:

**==> picture [259 x 158] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>│ - Agent runs in parallel with existing solution<br>│<br>│ - Output compared but not used<br>│<br>│ - Metrics tracked for promotion decision<br>│<br>│ - Human can force-promote or reject<br>│<br>│<br>│<br>│ Promotion criteria (from research):<br>│<br>│ - Minimum 10 invocations<br>│<br>│ - Success rate ≥ 90%<br>│<br>│ - Error rate ≤ 0.1% above baseline<br>│<br>│ - Maximum 14 days in shadow<br>│<br>│<br>│<br>**----- End of picture text -----**<br>

│

│

**==> picture [265 x 6] intentionally omitted <==**

**----- Start of picture text -----**<br>
└────────────────────────────────────────────────────────────────────<br>**----- End of picture text -----**<br>

│ ▼

PHASE 5: LIFECYCLE MANAGEMENT

┌────────────────────────────────────────────────────────────────────

│

│

│ Agent lifecycle states:

│

│

│ │ proposed → shadow → active → deprecated → archived │

│

│

│ Deprecation triggers:

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│ - Agent unused for 30 days → flag for review

**==> picture [4 x 20] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>│<br>**----- End of picture text -----**<br>

│ - Success rate < 50% over 20+ invocations → flag for removal

│ - Superseded by better agent → deprecate

│

│

│ │

│ Population control:

│ - Maximum 15 active agents (research: 3-4 is optimal, but specialized │

│ agents don't all run simultaneously)

│ │ │

│ - New agent spawn requires deprecation proposal if at limit

│

└────────────────────────────────────────────────────────────────────

**==> picture [209 x 6] intentionally omitted <==**

## **2.3 Example: Spawned Agent Definition**

_# .claude/agents/parallel-summarizer/agent.yaml_

name **:** parallel-summarizer version **:** 1.0.0 status **:** shadow created **:** 2026-01-13 purpose **:** | Handle document synthesis tasks that exceed single-context limits by spawning parallel Gemini workers with overlapping windows.

tier **:** gemini trigger_conditions **:**

**-** document_tokens > 50000

- task_type **:** "synthesis"

**-** single_pass_failed **:** true

configuration **:** window_size **:** 40000 overlap_percentage **:** 15 max_parallel_workers **:** 5 synthesis_model **:** gemini-flash

- input_schema **:** document_path **:** string synthesis_goal **:** string output_format **:** enum[summary, analysis, extraction]

output_schema **:** synthesis_result **:** string source_coverage **:** float confidence_score **:** float

constraints **:**

**-** max_total_tokens **:** 500000

**-** max_workers **:** 5 **-** timeout_minutes **:** 10

## **2.4 Novelty Assessment**

## **Existing approaches:**

**Agent System Adaptation Method Population** AutoGPT Fixed Manual configuration CrewAI Fixed Manual agent definition LangGraph Fixed Graph structure changes Multi-agent Fixed Developer extends frameworks codebase

**Our differentiation:**

1. **Automated gap detection** — System identifies when it needs new

- capabilities

- 2. **Complete agent generation** — Not just suggestions, but deployable definitions

3. **Shadow validation** — New agents prove themselves before

- production

4. **Lifecycle management** — Agents can be deprecated as needs

evolve

**Prior art search findings:** No published systems implement automated subagent generation from gap analysis. The closest analog is AutoML for model architecture search, but not applied to agent architecture.

- **2.5 Competitive Advantage**

1. **Self-healing architecture** — System addresses its own deficiencies

2. **Reduced maintenance burden** — No manual agent development for common gaps

3. **Organic capability growth** — Architecture evolves with usage patterns

4. **Risk-managed deployment** — Shadow mode prevents untested agents from causing harm

- **2.6 Implementation Priority**

**Phase 5 in roadmap (Months 6-7)**

Dependencies: - Weekly review process (Phase 3) - Systems Architect agent (Phase 3) - Agent registry infrastructure (Phase 1)

## **Innovation 3: Apprenticeship Learning Model (“Agent-in-the-Loop” Inversion)**

- **3.1 Concept Definition**

**Traditional human-in-the-loop:**

Agent proposes action → Human approves/rejects → Agent executes (or not)

The agent is the actor; the human is the gatekeeper.

**Our approach (Apprenticeship Learning):**

Human acts → System observes → Patterns emerge → Agent learns to replicate → Human validates agent's replication → Agent assumes routine decisions → Human handles novel cases

The human is the teacher; the agent is the apprentice learning to become the human (for routine decisions).

## **3.2 Architectural Specification**

**==> picture [265 x 34] intentionally omitted <==**

**----- Start of picture text -----**<br>
┌────────────────────────────────────────────────────────────────────<br>│ APPRENTICESHIP LEARNING MODEL<br>│<br>└────────────────────────────────────────────────────────────────────<br>**----- End of picture text -----**<br>

CONCEPT: The system learns to replicate the human orchestrator's decision-making for routine decisions, progressively reducing human involvement.

┌────────────────────────────────────────────────────────────────────

│ │ │ LEARNING PROGRESSION │

**==> picture [262 x 171] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>│<br>│ LEVEL 1: OPERATOR<br>│<br>│<br>┌────────────────────────────────────────────────────────────────────<br> │<br>│ │ Human approves ALL decisions<br>│ │<br>│ │ System observes and captures<br>│ │<br>│ │ Decision patterns begin accumulating<br>│ │<br>│ │ No automation<br>│ │<br>│ │<br>│ │<br>│ │ Duration: Until 100+ decisions captured<br>│ │<br>│<br>└────────────────────────────────────────────────────────────────────<br> │<br>│ │<br>│<br>│ ▼ Pattern detection<br>**----- End of picture text -----**<br>

**==> picture [415 x 666] intentionally omitted <==**

**----- Start of picture text -----**<br>
triggers │<br>│ LEVEL 2: COLLABORATOR<br>│<br>│<br>┌────────────────────────────────────────────────────────────────────<br> │<br>│ │ System suggests decisions based on patterns<br>│ │<br>│ │ Human confirms, modifies, or rejects<br>│ │<br>│ │ Confirmation/rejection feeds back to learning<br>│ │<br>│ │ Still no autonomous action<br>│ │<br>│ │<br>│ │<br>│ │ Duration: Until 95% success rate on suggestions over 200<br>decisions │ │<br>│<br>└────────────────────────────────────────────────────────────────────<br> │<br>│ │<br>│<br>│ ▼ Success threshold met<br>│<br>│ LEVEL 3: CONSULTANT<br>│<br>│<br>┌────────────────────────────────────────────────────────────────────<br> │<br>│ │ System auto-approves HIGH-CONFIDENCE decisions (>95% pattern<br>match) │ │<br>│ │ Human consulted on MEDIUM-CONFIDENCE decisions<br>│ │<br>│ │ Human decides on LOW-CONFIDENCE / novel situations<br>│ │<br>│ │ All auto-decisions logged for audit<br>│ │<br>│ │<br>│ │<br>│ │ Category-specific: Different decision types progress<br>independently │ │<br>│ │ Duration: Until 98% success rate over 500 decisions in category<br>│ │<br>│<br>└────────────────────────────────────────────────────────────────────<br> │<br>│ │<br>│<br>│ ▼ Extended success<br>demonstrated │<br>│ LEVEL 4: APPROVER<br>│<br>│<br>┌────────────────────────────────────────────────────────────────────<br> │<br>│ │ System handles routine decisions autonomously<br>│ │<br>│ │ Human reviews summary reports (daily/weekly)<br>│ │<br>│ │ Human intervenes only on flagged anomalies<br>│ │<br>│ │ Human approves CRITICAL decisions (security, architecture)<br>│ │<br>│ │<br>│ │<br>│ │ Duration: Indefinite with periodic validation<br>│ │<br>│<br>└────────────────────────────────────────────────────────────────────<br> │<br>│ │<br>│<br>│ ▼ Full trust established<br>│<br>│ LEVEL 5: OBSERVER<br>│<br>│<br>┌────────────────────────────────────────────────────────────────────<br> │<br>│ │ System operates autonomously for defined domain<br>│ │<br>│ │ Human observes via dashboard/reports<br>│ │<br>│ │ Human can intervene at any time<br>│ │<br>│ │ System flags when it encounters novel situations<br>│ │<br>│ │<br>│ │<br>│ │ Note: Level 5 only for specific, well-defined decision<br>categories │ │<br>│ │ Most domains remain at Level 3-4<br>│ │<br>│<br>└────────────────────────────────────────────────────────────────────<br> │<br>**----- End of picture text -----**<br>

│ │

└────────────────────────────────────────────────────────────────────

**==> picture [209 x 6] intentionally omitted <==**

## **3.3 Decision Capture Schema**

To enable apprenticeship learning, decisions must be captured with sufficient context for pattern matching:

**==> picture [220 x 318] intentionally omitted <==**

**----- Start of picture text -----**<br>
{<br>"decision_id": "uuid",<br>"timestamp": "2026-01-13T10:23:45Z",<br>"session_id": "session-uuid",<br>"decision_category": "routing_override",<br>"context": {<br>"task_type": "code_modification",<br>"task_description": "Refactor authentication module",<br>"complexity_score": 8.5,<br>"files_in_scope": 12,<br>"estimated_tokens": 35000,<br>"tags": ["security", "authentication", "refactor"]<br>},<br>"system_recommendation": {<br>"action": "route_to_sonnet",<br>"confidence": 0.82,<br>"reasoning": "Complexity score 8.5 within sonnet range"<br>},<br>"human_decision": {<br>"action": "route_to_opus",<br>"reasoning_provided": "Security-critical code needs careful<br>review",<br>"time_to_decide_seconds": 12<br>},<br>"outcome": {<br>"task_success": true ,<br>"quality_rating": null ,<br>"issues_encountered": [],<br>"would_recommend_same_decision": null<br>},<br>"learning_metadata": {<br>"autonomy_level_at_time": 2,<br>"pattern_match_candidates": ["security_escalation",<br>"complexity_override"],<br>"should_automate_similar": false ,<br>"requires_human_always": false<br>}<br>}<br>3.4 Autonomy Level Tracking<br>**----- End of picture text -----**<br>

**==> picture [232 x 268] intentionally omitted <==**

**----- Start of picture text -----**<br>

# .claude/autonomy-levels.yaml<br>schema_version : 1.0.0<br>updated : 2026-01-13<br>categories :<br>quick_approval :<br>current_level : 3 # CONSULTANT - auto-approving high-confidence<br>decision_count : 127<br>success_rate : 0.98<br>last_human_override : 2026-01-10<br>auto_approve_threshold : 0.95<br>routing_decisions :<br>current_level : 2 # COLLABORATOR - suggesting but human confirms<br>decision_count : 89<br>success_rate : 0.91<br>promotion_blocked_reason : "success rate below 95% threshold"<br>security_decisions :<br>current_level : 1 # OPERATOR - human approves all<br>decision_count : 23<br>success_rate : null # Not enough data<br>notes : "Security decisions will likely remain human-approved"<br>max_allowed_level : 2 # Never fully automate<br>scope_modifications :<br>current_level : 2 # COLLABORATOR<br>decision_count : 47<br>success_rate : 0.87<br>promotion_blocked_reason : "success rate below 95% threshold"<br>3.5 Novelty Assessment<br>Existing approaches:<br>System Learning Model Human Role<br>Standard<br>RLHF Human rates outputs Feedback provider<br>**----- End of picture text -----**<br>

ConstitutionalAI Human defines principles Rule definer Active Human labels uncertain Data labeler Learning cases Imitation Human demonstrates Demonstrator Learning behavior (batch)

**Our differentiation:**

1. **Continuous apprenticeship** — Learning happens during normal use, not separate training

2. **Category-specific progression** — Different decision types progress independently

3. **Transparent level tracking** — User always knows automation level per category

4. **Reversible autonomy** — Levels can decrease if success rate drops

5. **Graceful degradation** — Novel situations always surface to human

**Prior art search findings:** Apprenticeship learning exists in robotics (learning from demonstration), but applying this model to AI orchestration with progressive autonomy levels is novel. The closest analog is Levels of Automation taxonomies from human factors engineering, but those describe states, not transitions.

## **3.6 Competitive Advantage**

1. **Personalized automation** — System learns YOUR decisionmaking, not generic rules

2. **Earned trust** — Automation is earned through demonstrated competence

3. **Maintained oversight** — Human never fully out of the loop; can intervene anytime

4. **Category-specific control** — Automate routine decisions while keeping critical ones human-controlled

- **3.7 Implementation Priority**

## **Phase 6 in roadmap (Months 8-10)**

Dependencies: - Decision capture schema (Phase 2) - Pattern detection (Phase 4) - Weekly review with autonomy assessment (Phase 3)

## **Innovation 4: Code-Enforced Deterministic Routing**

## **4.1 Concept Definition**

**Traditional approach:** Use the LLM itself to decide which model tier to use, or use keyword matching.

**Our approach:** Bash scripts calculate complexity scores using arithmetic formulas. The code enforces tier ceilings. The LLM cannot route itself to expensive tiers regardless of what it “decides.”

## **4.2 Implementation**

_#!/bin/bash_

- _# calculate-complexity.sh_

METRICS_FILE="$HOME/.claude/tmp/scout_metrics.json"

_# Extract values from scout report_

TOKENS=$(jq '.scout_report.scope_metrics.estimated_tokens' "$METRICS_FILE") FILES=$(jq '.scout_report.scope_metrics.total_files' "$METRICS_FILE") MODULES=$(jq '.scout_report.complexity_signals.module_count // 1' "$METRICS_FILE")

_# Calculate complexity score using ARITHMETIC, not LLM_

_interpretation_ SCORE=$(echo "scale=2; ($TOKENS / 10000) + ($FILES / 5) + ($MODULES \* 2)" **|** bc)

_# Determine tier using NUMERIC COMPARISON, not LLM judgment_ **if ((** $(echo "$SCORE < 2" **|** bc -l) **)); then** TIER="haiku" **elif ((** $(echo "$SCORE < 10" **|** bc -l) **)); then** TIER="sonnet"

**else** TIER="opus" **fi**

_# Force Gemini for large context regardless of score_ **if ((** TOKENS > 50000 **)); then** TIER="gemini" **fi**

_# Write results (code decides, not LLM)_ echo "$SCORE" > "$HOME/.claude/tmp/complexity_score" echo "$TIER" > "$HOME/.claude/tmp/recommended_tier"

## **4.3 Why This Matters**

| **Risk**      | **LLM-Based Routing**                    | **Code-Enforced Routing**         |
| ------------- | ---------------------------------------- | --------------------------------- |
| Prompt        | Attacker can trick LLM                   | Code ignores prompts              |
| injection     | into expensive tier                      | entirely                          |
| Cost overrun  | LLM may rationalize<br>expensive choices | Arithmetic doesn’t<br>rationalize |
| Inconsistency | Same task may route<br>diferently        | Deterministic for same<br>inputs  |
| Auditability  | “Why did it choose<br>Opus?” — unclear   | Formula is explicit and<br>logged |
| Latency       | 100-500ms for routing<br>decision        | <10ms fle I/O                     |

## **4.4 Novelty Assessment**

**Existing approaches:**

| **Framework**           | **Routing Method**               |
| ----------------------- | -------------------------------- |
| LangChain               | LLM-based or manual              |
| AutoGPT                 | LLM self-selects                 |
| Claude native           | Single model, no routing         |
| RouteLLM                | ML classifer (still model-based) |
| **Our diferentiation:** |                                  |

Code-enforced routing using bash arithmetic is, to our knowledge, unique in production multi-agent systems. The pattern of using shell scripts as an enforcement layer between user intent and model selection is novel.

- **4.5 Competitive Advantage**

1. **Predictable costs** — Formula determines cost, not LLM mood

2. **Zero routing latency** — No LLM call needed for routing

3. **Fully auditable** — Every routing decision traceable to formula

4. **Manipulation-resistant** — Prompt injection cannot affect routing

5. **Simple to adjust** — Change thresholds in one file

## **5. IP Protection Recommendations**

- **5.1 Documentation Requirements**

To establish prior art and potential patent claims:

1. **Timestamp all documents** — This guide serves as dated documentation

2. **Maintain git history** — All implementation changes tracked 3. **Publish technical descriptions** — Consider blog posts or technical papers

3. **Record conception dates** — Brain dump messages are dated evidence

## **5.2 Provisional Patent Candidates**

| **Innovation**             | **Patentability Assessment**                 | **Priority** |
| -------------------------- | -------------------------------------------- | ------------ |
| Emergent                   |                                              |              |
| Schema                     | High — Novel method, clear utility           | High         |
| Discovery                  |                                              |              |
| Subagent<br>Spawning       | High — Novel method, clear utility           | High         |
| Apprenticeship<br>Learning | Medium — Builds on existing<br>concepts      | Medium       |
| Code-Enforced<br>Routing   | Medium — Novel application,<br>simple method | Low          |

- **5.3 Trade Secret Considerations**

Some implementation details may be better protected as trade secrets than patents: Specific threshold values and their derivation Observation capture heuristics Pattern detection algorithms tuned to specific domains

## **5.4 Recommended Actions**

1. **Immediate:** File provisional patent applications for Innovations 1 and 2

2. **Near-term:** Document implementation details sufficient for reproduction

3. **Ongoing:** Maintain dated development records

4. **Consider:** Academic publication for prior art establishment

## **Summary**

These four innovations represent genuine architectural novelty in the multi-agent systems space:

1. **Emergent Schema Discovery** — The system discovers what to measure, not just how to measure it

2. **Subagent Spawning** — The architecture grows capabilities based on observed needs

3. **Apprenticeship Learning** — The system learns to become the human orchestrator

4. **Code-Enforced Routing** — Deterministic control that LLMs cannot circumvent

Together, they create an architecture that is **self-evolving** — it improves through use rather than requiring manual enhancement. This positions GoGent as a fundamentally different approach to AI orchestration, one where the human trains the system through normal use rather than explicit configuration.
