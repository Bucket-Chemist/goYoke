# Part II: Technical Architecture

## **Comprehensive System Reference**

## **1. System Topology**

## **1.1 Complete Component Map**

┌────────────────────────────────────────────────────────────────────

│ GoGent SYSTEM TOPOLOGY │

**==> picture [415 x 667] intentionally omitted <==**

**----- Start of picture text -----**<br>

protocols/ │ │ PostToolUse.sh │ │ .json │ │ sharp-edges/ │ │ handover.md │ │ SubagentStop.sh │ │ │ │ facts/ │ │ │ │ SessionEnd.sh │ │ complexity* │ │ preferences/ │ │ config.yaml │ │ │ │ score │ │ observations/ │ │ │ │ validate- │ │ │ │ │ │ CLI interface │ │ routing.sh │ │ recommended* │ │ YAML frontmatter│ │ via pipes │ │ │ │ tier │ │ + markdown body │ │ │ │ calculate- │ │ │ │ │ │ │ │ complexity.sh │ │ handoff.json │ │ BM25 indexable │ │ │ │ │ │ │ │ │ │ │ │ attention- │ │ routing_log │ │ git versioned │ │ │ │ gate.sh │ │ .jsonl │ │ │ │ │ │ │ │ │ │ │ │ │ └─────────────────┘ └─────────────────┘ └─────────────────┘ └─────────────────┘

## **1.2 Component Responsibilities**

| **Component**                       | **Primary**<br>**Responsibility**                 | **Secondary**<br>**Responsibilities**  |
| ----------------------------------- | ------------------------------------------------- | -------------------------------------- |
| **Claude Code**<br>**Orchestrator** | Conversation<br>management, task<br>decomposition | Tool invocation, context<br>management |
| **Hook Layer**                      | Routing enforcement,<br>validation                | Logging, metrics capture,<br>blocking  |
| **State Layer**                     | Inter-agent<br>communication                      | Checkpointing, handof<br>persistence   |
| **Memory**                          | Long-term learning                                | Pattern retrieval, decision            |
| **Layer**                           | storage                                           | history                                |
| **External**                        | Large context                                     | Codebase analysis,                     |
| **Layer**                           | operations                                        | document synthesis                     |

## **1.3 File System Layout**

- ~/.claude/ ├── hooks/ │ ├── PreToolUse.sh # Routing enforcement (runs before every tool) │ ├── PostToolUse.sh # Outcome logging (runs after every tool) │ ├── SubagentStop.sh # Subagent completion validation │ └── SessionEnd.sh # Session cleanup, memory archival trigger │ ├── scripts/ │ ├── calculate-complexity.sh # Complexity scoring formula │ ├── validate-routing.sh # Tier permission checker │ ├── attention-gate.sh # Focus management │ └── query-memory.sh # Memory retrieval interface │ ├── agents/

- │ ├── haiku-scout/

- │ │ ├── agent.md # Agent definition

- │ │ └── agent.yaml # Configuration

- │ ├── architect/

- │ ├── memory-archivist/

- │ └── [spawned-agents]/ # Future dynamically created agents │ ├── skills/ │ └── explore/ │ └── SKILL.md # Explore workflow definition │ ├── tmp/ # Ephemeral state (cleared per session) │ ├── scout_metrics.json │ ├── complexity_score │ ├── recommended_tier │ └── handoff.json │ ├── memory/ # Persistent learning (git versioned)

- │ ├── decisions/ # Approved architectural decisions

- │ ├── sharp-edges/ # Known pitfalls and workarounds

- │ ├── facts/ # Verified project facts

- │ ├── preferences/ # User preferences and patterns │ └── observations/ # Raw behavioral observations (future) │

- ├── schemas/ # Schema definitions (future)

│ └── [version]/

│

- ├── settings.json # Global configuration ├── routing-schema.json # Routing rules and thresholds └── agents-index.json # Agent registry with status


- ├── protocols/

- │ ├── handover-protocol.md # Claude→Gemini handoff format

- │ ├── codebase-analysis.md # Large codebase handling

- │ └── document-synthesis.md # Multi-document summarization └── config.yaml # Gemini CLI configuration

## **2. Tier Structure**

## **2.1 Tier Definitions**

**Tier 0: Haiku (Scout/Quick Operations)**

**Purpose:** Fast, cheap reconnaissance and simple tasks **Cost:** $0.25/1M input, $1.25/1M output **Context:** 200K tokens **Latency:** Fastest

**Use cases:** - Initial codebase reconnaissance (scout protocol) - Simple file lookups - Quick validation checks - Formatting and linting decisions - Yes/no classification tasks

**Routing trigger:** Complexity score < 2

**Tier 1: Sonnet (Standard Operations)**

- **Purpose:** Primary workhorse for most development tasks **Cost:** $3/1M input, $15/1M output **Context:** 200K tokens **Latency:** Fast

**Use cases:** - Architecture planning - Code generation and modification - Code review - Documentation writing - Multi-file refactoring (up to 10 files) - Standard problem-solving

**Routing trigger:** Complexity score 2-10

**Tier 2: Opus (Complex Operations)**

**Purpose:** Most capable reasoning for difficult problems **Cost:** $15/1M input, $75/1M output **Context:** 200K tokens **Latency:** Slower **Use cases:** - Complex architectural decisions - Edge case reasoning - Cross-cutting refactors - Critical security reviews - Novel problemsolving

**Routing trigger:** Complexity score > 10, or explicit --force-tier opus

**External: Gemini Flash (Large Context Operations)**

**Purpose:** Operations requiring context beyond Claude’s window **Cost:** ~$0.075/1M input (free tier available) **Context:** 1M tokens **Latency:** Variable

**Use cases:** - Full codebase analysis - Large document synthesis - Multi-file context assembly - Repository-wide search and analysis **Routing trigger:** Estimated tokens > 50K, or explicit Gemini protocol invocation

## **2.2 Tier Selection Matrix**

| **Task Type**             | **Files** | **Complexity**    | **Recommended**<br>**Tier** |
| ------------------------- | --------- | ----------------- | --------------------------- |
| Simple lookup             | 1         | Low               | Haiku                       |
| Bug fx                    | 1-3       | Low-Medium        | Sonnet                      |
| Feature<br>implementation | 3-10      | Medium            | Sonnet                      |
| Architecture refactor     | 5-15      | High              | Opus                        |
| Codebase analysis         | 20+       | Context-<br>bound | Gemini                      |
| Security audit            | Variable  | Critical          | Opus                        |
| Documentation             | Variable  | Low-Medium        | Sonnet                      |

## **3. Request Lifecycle**

## **3.1 Six-Phase Request Processing**

┌──────────────────────────────────────────────────────────────────── │ COMPLETE REQUEST LIFECYCLE

## │

└────────────────────────────────────────────────────────────────────

PHASE 1: INTENT CLASSIFICATION

┌──────────────────────────────────────────────────────────────────── │ │ │ User Input ──→ Parse Intent ──→ Determine if Explore Workflow Needed │ │ │ │ Keywords: "explore", "investigate", "analyze codebase", "understand" │ │ → Triggers full explore workflow │ │ │ │ Direct tasks: "fix bug", "add feature", "write test" │ │ → May skip to Phase 3 with direct routing │ │ │ └──────────────────────────────────────────────────────────────────── │ ▼ PHASE 2: SCOUT (Conditional)

┌────────────────────────────────────────────────────────────────────

**==> picture [265 x 458] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>│<br>│ Haiku Scout Agent performs reconnaissance:<br>│<br>│<br>│<br>│ 1. Count files in scope<br>│<br>│ 2. Estimate token requirements<br>│<br>│ 3. Identify module boundaries<br>│<br>│ 4. Detect cross-file dependencies<br>│<br>│ 5. Flag complexity indicators<br>│<br>│<br>│<br>│ Output: .claude/tmp/scout_metrics.json<br>│<br>│ {<br>│<br>│ "scout_report": {<br>│<br>│ "scope_metrics": {<br>│<br>│ "total_files": 15,<br>│<br>│ "estimated_tokens": 45000<br>│<br>│ },<br>│<br>│ "complexity_signals": {<br>│<br>│ "cross_file_dependencies": 8,<br>│<br>│ "module_count": 3<br>│<br>│ }<br>│<br>│ }<br>│<br>│ }<br>│<br>│<br>│<br>└────────────────────────────────────────────────────────────────────<br> │<br> ▼<br>PHASE 3: COMPLEXITY CALCULATION<br>┌────────────────────────────────────────────────────────────────────<br>│<br>│<br>│ calculate-complexity.sh reads scout_metrics.json and applies<br>formula: │<br>│<br>│<br>│<br>┌────────────────────────────────────────────────────────────────────<br> │<br>│ │<br>│ │<br>│ │ SCORE = (tokens / 10000) + (files / 5) + (modules × 2)<br>│ │<br>│ │<br>**----- End of picture text -----**<br>

│ │

**==> picture [265 x 240] intentionally omitted <==**

**----- Start of picture text -----**<br>
│ │ Example: 45000 tokens, 15 files, 3 modules<br>│ │<br>│ │ SCORE = 4.5 + 3.0 + 6.0 = 13.5<br>│ │<br>│ │<br>│ │<br>│<br>└────────────────────────────────────────────────────────────────────<br> │<br>│<br>│<br>│ Output files:<br>│<br>│ • .claude/tmp/complexity_score → "13.50"<br>│<br>│ • .claude/tmp/recommended_tier → "opus"<br>│<br>│<br>│<br>│ Threshold mapping:<br>│<br>│ • Score < 2 → haiku<br>│<br>│ • Score 2-10 → sonnet<br>│<br>│ • Score > 10 → opus<br>│<br>│ • Tokens > 50K → FORCE gemini (regardless of score)<br>│<br>│<br>│<br>└────────────────────────────────────────────────────────────────────<br> │<br> ▼<br>**----- End of picture text -----**<br>

PHASE 4: ROUTING ENFORCEMENT

**==> picture [265 x 411] intentionally omitted <==**

**----- Start of picture text -----**<br>
┌────────────────────────────────────────────────────────────────────<br>│<br>│<br>│ PreToolUse.sh hook executes BEFORE any tool invocation:<br>│<br>│<br>│<br>│<br>┌────────────────────────────────────────────────────────────────────<br> │<br>│ │<br>│ │<br>│ │ 1. Check for --force-tier escape hatch<br>│ │<br>│ │ └─ If present and valid, PERMIT regardless of score<br>│ │<br>│ │<br>│ │<br>│ │ 2. Check scout_metrics freshness (< 5 minutes)<br>│ │<br>│ │ └─ If stale, trigger re-scout or use conservative estimate<br>│ │<br>│ │<br>│ │<br>│ │ 3. Read recommended_tier from state file<br>│ │<br>│ │<br>│ │<br>│ │ 4. Compare requested operation tier to ceiling<br>│ │<br>│ │ └─ If operation tier ≤ ceiling: PERMIT (exit 0)<br>│ │<br>│ │ └─ If operation tier > ceiling: BLOCK (exit 2)<br>│ │<br>│ │<br>│ │<br>│ │ 5. Log routing decision to routing_log.jsonl<br>│ │<br>│ │<br>│ │<br>│<br>└────────────────────────────────────────────────────────────────────<br> │<br>│<br>│<br>│ Exit codes:<br>│<br>│ • 0 = PERMIT (continue with tool execution)<br>│<br>│ • 1 = ERROR (hook failure, blocks execution)<br>│<br>│ • 2 = BLOCK (routing violation, blocks with message)<br>│<br>│<br>│<br>└────────────────────────────────────────────────────────────────────<br> │<br> ▼<br>**----- End of picture text -----**<br>

PHASE 5: EXECUTION

┌────────────────────────────────────────────────────────────────────

**==> picture [262 x 150] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>│<br>│ Selected agent executes the task:<br>│<br>│<br>│<br>│ • Receives full context (user request + relevant files + memory)<br>│<br>│ • Performs reasoning and tool invocations<br>│<br>│ • Each tool invocation triggers PreToolUse/PostToolUse hooks<br>│<br>│ • Produces output (code, documentation, analysis)<br>│<br>│<br>│<br>│ For explore workflow, this is the multi-phase execution:<br>│<br>│ Architect → [Approval] → Execute phases → Archive<br>│<br>│<br>│<br>**----- End of picture text -----**<br>

└──────────────────────────────────────────────────────────────────── │

▼

PHASE 6: MEMORY ARCHIVAL

**==> picture [265 x 217] intentionally omitted <==**

**----- Start of picture text -----**<br>
┌────────────────────────────────────────────────────────────────────<br>│<br>│<br>│ PostToolUse.sh and SessionEnd.sh capture outcomes:<br>│<br>│<br>│<br>│ 1. Log tool invocation results<br>│<br>│ 2. Capture any human overrides or corrections<br>│<br>│ 3. Detect sharp edges (errors, unexpected behaviors)<br>│<br>│ 4. Queue decisions for memory archivist processing<br>│<br>│<br>│<br>│ Memory Archivist (periodic/triggered):<br>│<br>│ • Processes queued items<br>│<br>│ • Extracts learnings into appropriate memory directories<br>│<br>│ • Updates facts/, decisions/, sharp-edges/<br>│<br>│ • Commits changes to git<br>│<br>│<br>│<br>└────────────────────────────────────────────────────────────────────<br>**----- End of picture text -----**<br>

## **3.2 Phase Timing**

**==> picture [209 x 83] intentionally omitted <==**

**----- Start of picture text -----**<br>
Phase Typical Variable Factors<br>Duration<br>Intent Classification <100ms Orchestrator native<br>Scout 2-5s Codebase size<br>Complexity <10ms Pure bash arithmetic<br>Calculation<br>Routing Enforcement <5ms File I/O only<br>Execution 5s-5min Task complexity<br>Memory Archival 1-10s Background, non-<br>blocking<br>**----- End of picture text -----**<br>

**4. Routing Enforcement Flow**

## **4.1 Decision Tree**

**==> picture [180 x 82] intentionally omitted <==**

**----- Start of picture text -----**<br>
┌─────────────────┐<br> │ Tool Request │<br> │ Initiated │<br> └────────┬────────┘<br> │<br> ▼<br> ┌─────────────────┐<br> │ --force-tier │<br> │ flag set? │<br> └────────┬────────┘<br> │<br> ┌──────────────────┴──────────────────┐<br>**----- End of picture text -----**<br>

**==> picture [223 x 513] intentionally omitted <==**

**----- Start of picture text -----**<br>
│ YES │ NO<br> ▼ ▼<br> ┌─────────────────┐<br>┌─────────────────┐<br> │ Validate tier │ │ Scout metrics<br>│<br> │ is permitted │ │ exist & fresh?<br>│<br> │ (user auth) │ │ (< 5 min old)<br>│<br> └────────┬────────┘<br>└────────┬────────┘<br> │ │<br> ┌─────────┴─────────┐<br>┌─────────────┴─────────────┐<br> │ VALID │ INVALID │ YES<br>│ NO<br> ▼ ▼ ▼<br>▼<br> ┌─────────┐ ┌─────────┐ ┌─────────────────┐<br>┌─────────────────┐<br> │ PERMIT │ │ BLOCK │ │ Read complexity │ │<br>Trigger scout │<br> │ (log │ │ (invalid│ │ from state file │ │ OR<br>use default │<br> │ override)│ │ escape) │ └────────┬────────┘ │<br>conservative │<br> └─────────┘ └─────────┘ │ │<br>estimate │<br> ▼<br>└────────┬────────┘<br> ┌─────────────────┐<br>│<br> │ calculate- │<br>│<br> │ complexity.sh<br>│◄─────────────────┘<br> │ (if needed) │<br> └────────┬────────┘<br> │<br> ▼<br> ┌─────────────────┐<br> │ Read │<br> │ recommended\_ │<br> │ tier │<br> └────────┬────────┘<br> │<br> ▼<br> ┌─────────────────┐<br> │ Requested tier │<br> │ ≤ recommended? │<br> └────────┬────────┘<br> │<br> ┌──────────────────┴──────────────────┐<br> │ YES │<br>NO<br> ▼ ▼<br> ┌─────────┐<br>┌─────────┐<br> │ PERMIT │ │ BLOCK<br>│<br> │ exit 0 │ │ exit 2<br>│<br> │ │ │<br>+message│<br> └─────────┘<br>└─────────┘<br> │ │<br> └──────────────┬───────────────────────┘<br> ▼<br> ┌─────────────────┐<br> │ Log decision to │<br> │ routing_log. │<br> │ jsonl │<br> └─────────────────┘<br>**----- End of picture text -----**<br>

**4.2 Routing Log Format**

{ "timestamp": "2026-01-13T10:23:45Z", "session_id": "abc123", "tool_name": "Edit", "requested_tier": "opus", "calculated_tier": "sonnet", "complexity_score": 8.5, "scout_metrics": { "files": 12, "tokens": 35000,

"modules": 2

}, "decision": "BLOCK", "reason": "requested_tier exceeds calculated ceiling", "force_override": **false** }

## **5. The Explore Workflow**

## **5.1 Seven-Phase Breakdown**

**==> picture [265 x 151] intentionally omitted <==**

**----- Start of picture text -----**<br>
┌────────────────────────────────────────────────────────────────────<br>│ EXPLORE WORKFLOW (7 PHASES)<br>│<br>└────────────────────────────────────────────────────────────────────<br>PHASE 1: ACKNOWLEDGE<br>┌────────────────────────────────────────────────────────────────────<br>│ • Orchestrator confirms understanding of user's goal<br>│<br>│ • Clarifies scope if ambiguous<br>│<br>│ • Establishes success criteria<br>│<br>│ Duration: Interactive (user dependent)<br>│<br>└────────────────────────────────────────────────────────────────────<br> │<br> ▼<br>**----- End of picture text -----**<br>

PHASE 2: SCOUT

**==> picture [265 x 130] intentionally omitted <==**

**----- Start of picture text -----**<br>
┌────────────────────────────────────────────────────────────────────<br>│ Agent: Haiku Scout (or Gemini for large codebases)<br>│<br>│<br>│<br>│ Tasks:<br>│<br>│ • Enumerate files in scope<br>│<br>│ • Estimate token requirements<br>│<br>│ • Identify module structure<br>│<br>│ • Map dependencies<br>│<br>│ • Flag complexity indicators<br>│<br>│<br>**----- End of picture text -----**<br>

**==> picture [259 x 35] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>│ Output: scout_metrics.json → Feeds Phase 3<br>│<br>│ Duration: 2-30s depending on codebase size<br>│<br>**----- End of picture text -----**<br>

**==> picture [265 x 6] intentionally omitted <==**

**----- Start of picture text -----**<br>
└────────────────────────────────────────────────────────────────────<br>**----- End of picture text -----**<br>

**==> picture [129 x 21] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br> ▼<br>PHASE 3: ROUTE<br>**----- End of picture text -----**<br>

**==> picture [265 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
┌────────────────────────────────────────────────────────────────────<br>**----- End of picture text -----**<br>

**==> picture [262 x 6] intentionally omitted <==**

**----- Start of picture text -----**<br>
│ Agent: None (bash calculation)<br>**----- End of picture text -----**<br>

**==> picture [265 x 233] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>│<br>│<br>│ Decision table:<br>│<br>│<br>┌─────────────────┬────────────────┬─────────────────────────────────<br> │<br>│ │ Confidence │ Token Estimate │ Route To<br>│ │<br>│<br>├─────────────────┼────────────────┼─────────────────────────────────<br> │<br>│ │ High (>0.8) │ < 30K │ Sonnet Architect<br>│ │<br>│ │ High (>0.8) │ 30K - 100K │ Opus Architect<br>│ │<br>│ │ High (>0.8) │ > 100K │ Gemini → then Sonnet/Opus<br>│ │<br>│ │ Low (<0.8) │ Any │ Request clarification OR<br>Opus │ │<br>│ │ Any │ > 500K │ FORCE Gemini with chunking<br>│ │<br>│<br>└─────────────────┴────────────────┴─────────────────────────────────<br> │<br>│<br>│<br>│ Duration: <100ms<br>│<br>└────────────────────────────────────────────────────────────────────<br> │<br> ▼<br>**----- End of picture text -----**<br>

PHASE 4: ARCHITECT

**==> picture [265 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
┌────────────────────────────────────────────────────────────────────<br>**----- End of picture text -----**<br>

│ Agent: Sonnet or Opus (based on routing)

│ │ │ │ Tasks: │ │ • Analyze scout report │ │ • Create implementation plan │ │ • Break into discrete phases │ │ • Assign agent type per phase │ │ • Estimate cost and duration │ │ │ │ Output: specs.md with structure: │ │ `│ │  # Implementation Plan │ │  ## Phase 1: [Name] │ │  - Agent: sonnet │ │  - Files: [list] │ │  - Tasks: [list] │ │  - Estimated tokens: X │ │  ## Phase 2: [Name] │ │  ... │ │ ` │ │ │ │ Duration: 10-60s │ └──────────────────────────────────────────────────────────────────── │ ▼ PHASE 5: APPROVE ┌──────────────────────────────────────────────────────────────────── │ Agent: Human │ │ │ │ User reviews specs.md and: │

│ • APPROVE → Continue to Phase 6

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>**----- End of picture text -----**<br>

│ • MODIFY → Edit specs.md, return to Phase 4 for replan │ │ • REJECT → Abort workflow, capture reason in sharp-edges/ │ │ │ │ Duration: User dependent (could be immediate or hours) │

└────────────────────────────────────────────────────────────────────

│ ▼

PHASE 6: EXECUTE

┌────────────────────────────────────────────────────────────────────

│ Agents: Per-phase as specified in specs.md

│ │ │ │ Agent selection by task type: │ │ ┌─────────────────────────────┬─────────────┬──────────────────────── │ │ │ Task Type │ Agent │ Rationale │ │ │ ├─────────────────────────────┼─────────────┼──────────────────────── │ │ │ File creation (simple) │ Sonnet │ Standard generation │ │ │ │ File creation (complex) │ Opus │ Architectural decisions │ │ │ │ Refactoring (< 5 files) │ Sonnet │ Bounded scope │ │ │ │ Refactoring (> 5 files) │ Opus │ Cross-cutting concerns │ │ │ │ Test writing │ Sonnet │ Pattern-based

**==> picture [265 x 458] intentionally omitted <==**

**----- Start of picture text -----**<br>
│ │<br>│ │ Documentation │ Sonnet │ Standard prose<br>│ │<br>│ │ Security review │ Opus │ Critical analysis<br>│ │<br>│ │ Large file analysis │ Gemini │ Context<br>requirements │ │<br>│<br>└─────────────────────────────┴─────────────┴────────────────────────<br> │<br>│<br>│<br>│ Each phase execution:<br>│<br>│ 1. Load phase context from specs.md<br>│<br>│ 2. PreToolUse hook validates tier permission<br>│<br>│ 3. Execute with selected agent<br>│<br>│ 4. PostToolUse hook logs outcome<br>│<br>│ 5. Validate completion (SubagentStop hook)<br>│<br>│ 6. Proceed to next phase or flag for human intervention<br>│<br>│<br>│<br>│ Duration: Varies by task complexity (minutes to hours)<br>│<br>└────────────────────────────────────────────────────────────────────<br> │<br> ▼<br>PHASE 7: ARCHIVE<br>┌────────────────────────────────────────────────────────────────────<br>│ Agent: Memory Archivist (Haiku or Sonnet)<br>│<br>│<br>│<br>│ Sources processed:<br>│<br>│ • specs.md (architectural decisions)<br>│<br>│ • Execution logs (what worked, what failed)<br>│<br>│ • Human modifications (corrections, overrides)<br>│<br>│ • Sharp edge detections (errors encountered)<br>│<br>│<br>│<br>│ Outputs:<br>│<br>│ • .claude/memory/decisions/[date]-[topic].md<br>│<br>│ • .claude/memory/sharp-edges/[date]-[issue].md<br>│<br>│ • .claude/memory/facts/[topic].md (updates)<br>│<br>│<br>│<br>│ Duration: 5-30s (background, non-blocking)<br>│<br>└────────────────────────────────────────────────────────────────────<br>**----- End of picture text -----**<br>

**==> picture [95 x 11] intentionally omitted <==**

**----- Start of picture text -----**<br> 6. Memory Pipeline<br>**----- End of picture text -----**<br>

**6.1 Memory Architecture**

**==> picture [265 x 151] intentionally omitted <==**

**----- Start of picture text -----**<br>
┌────────────────────────────────────────────────────────────────────<br>│ MEMORY PIPELINE<br>│<br>└────────────────────────────────────────────────────────────────────<br>SOURCES PROCESSING<br>STORAGE<br>─────────────────────────────────────────────────────────────────────<br>┌─────────────────┐<br>│ Architect │<br>│ specs.md │──┐<br>└─────────────────┘ │<br> │<br>┌─────────────────┐ │ ┌─────────────────┐<br>┌─────────────────┐<br>│ Sharp Edge │ │ │ │ │<br>decisions/ │<br>│ Detector │──┼────────→│ MEMORY │────────→│<br>**----- End of picture text -----**<br>

**==> picture [415 x 666] intentionally omitted <==**

**----- Start of picture text -----**<br>
│<br>└─────────────────┘ │ │ ARCHIVIST │ │ 2026-<br>01-13- │<br> │ │ │ │ auth-<br>refactor │<br>┌─────────────────┐ │ │ (Haiku/Sonnet) │ │ .md<br>│<br>│ Session │──┤ │ │<br>└─────────────────┘<br>│ Context │ │ │ • Extracts │<br>└─────────────────┘ │ │ learnings │<br>┌─────────────────┐<br> │ │ • Categorizes │ │ sharp-<br>edges/ │<br>┌─────────────────┐ │ │ • Formats with │────────→│<br>│<br>│ Human │──┤ │ frontmatter │ │ jwt-<br>expiry- │<br>│ Corrections │ │ │ • Commits to │ │<br>gotcha.md │<br>└─────────────────┘ │ │ git │<br>└─────────────────┘<br> │ │ │<br>┌─────────────────┐ │ └─────────────────┘<br>┌─────────────────┐<br>│ Tool Execution │──┘ │ facts/<br>│<br>│ Logs │ ────→│<br>│<br>└─────────────────┘ │<br>project- │<br> │<br>structure.md │<br>└─────────────────┘<br>┌─────────────────┐<br> │<br>preferences/ │<br> ────→│<br>│<br> │ coding-<br>style │<br> │ .md<br>│<br>└─────────────────┘<br> RETRIEVAL<br> ─────────────────────────────<br> ┌─────────────────┐<br> │ query-memory.sh │<br> │ │<br> │ • BM25 search │<br> │ • Frontmatter │<br> │ filtering │<br> │ • Recency │<br> │ weighting │<br> └────────┬────────┘<br> │<br> ▼<br> ┌─────────────────┐<br> │ Context │<br> │ Assembly │──────→ Agent receives<br>relevant memory<br> └─────────────────┘<br>6.2 Memory File Format<br>All memory files use YAML frontmatter with markdown body:<br>---<br>title : "Decision: JWT Token Refresh Strategy"<br>created : 2026-01-13<br>updated : 2026-01-13<br>category : architecture<br>tags : [ authentication , jwt , security ]<br>related : [ ./oauth-implementation.md , ../sharp-edges/jwt-expiry-<br>gotcha.md ]<br>status : active<br>confidence : high<br>source : explore-session-2026-01-13<br>summary : "Chose sliding window refresh over fixed expiry for better<br>UX"<br>---<br>## Context<br>During authentication refactor, needed to decide between fixed JWT<br>expiry<br>(requiring re-login) and sliding window refresh (extending on<br>activity).<br>## Decision<br>**----- End of picture text -----**<br>

Implemented sliding window refresh with **:**

- 15-minute access token

- 7-day refresh token

- Refresh on any authenticated request within 5 minutes of expiry

- _## Rationale_

- Better user experience (no unexpected logouts during active

sessions)

- Security acceptable for this application's threat model

- Aligns with OAuth 2.0 best practices

- _## Consequences_

- Must handle token refresh failures gracefully

- Need to implement refresh token rotation

- Slightly more complex token validation logic

## **6.3 Retrieval Mechanism**

Current implementation uses grep-based search with YAML

frontmatter filtering:

- _# query-memory.sh simplified logic_

- search_memory() **{** local query="$1" local category="${2:-}"

**==> picture [189 x 96] intentionally omitted <==**

**----- Start of picture text -----**<br>

# Find files matching category filter<br>if [ -n "$category" ] ; then<br>files=$(grep -l "category: $category"<br>~/.claude/memory/**/*.md)<br>else<br>files=$(find ~/.claude/memory -name "\*.md")<br>fi<br># BM25-style term matching (simplified)<br>for file in $files ; do<br>score=$(calculate_bm25_score "$file" "$query")<br>echo "$score $file"<br>done | sort -rn | head -10<br>}<br>**----- End of picture text -----**<br>

**Planned upgrade path:** Implement proper BM25 via rank_bm25 Python library, with optional sqlite-vec for semantic search when pattern matching proves insufficient.

## **7. Inter-Agent Communication**

## **7.1 State File Specifications**

**scout_metrics.json**

**==> picture [189 x 212] intentionally omitted <==**

**----- Start of picture text -----**<br>
{<br>"schema_version": "1.0.0",<br>"generated_at": "2026-01-13T10:23:45Z",<br>"scout_agent": "haiku",<br>"scout_report": {<br>"scope_metrics": {<br>"total_files": 15,<br>"file_types": {<br>".py": 10,<br>".md": 3,<br>".json": 2<br>},<br>"estimated_tokens": 45000,<br>"largest_file": {<br>"path": "src/auth/handlers.py",<br>"tokens": 8500<br>}<br>},<br>"complexity_signals": {<br>"cross_file_dependencies": 8,<br>"module_count": 3,<br>"circular_imports": 0,<br>"test_coverage_files": 5<br>},<br>"recommendations": {<br>"suggested_tier": "sonnet",<br>"gemini_offload_candidates": [],<br>"risk_factors": ["large handlers.py may need splitting"]<br>}<br>}<br>}<br>**----- End of picture text -----**<br>

**handoff.json (Inter-agent handoff)**

**==> picture [4 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
{<br>**----- End of picture text -----**<br>

"schema_version": "1.0.0", "handoff_id": "uuid-here", "from_agent": "architect", "to_agent": "executor",

**==> picture [226 x 171] intentionally omitted <==**

**----- Start of picture text -----**<br>
"created_at": "2026-01-13T10:25:00Z",<br>"context": {<br>"task_summary": "Implement OAuth refresh token rotation",<br>"files_in_scope": ["src/auth/tokens.py",<br>"src/auth/middleware.py"],<br>"critical_constraints": [<br>"Must maintain backward compatibility with existing tokens",<br>"Refresh rotation must be atomic"<br>],<br>"success_criteria": [<br>"All existing tests pass",<br>"New rotation tests added",<br>"No breaking API changes"<br>]<br>},<br>"artifacts": {<br>"specs_path": ".claude/tmp/specs.md",<br>"scout_metrics_path": ".claude/tmp/scout_metrics.json"<br>},<br>"metadata": {<br>"estimated_tokens": 25000,<br>"estimated_duration_minutes": 15,<br>"tier_ceiling": "sonnet"<br>}<br>}<br>**----- End of picture text -----**<br>

## **7.2 Handoff Protocol**

**==> picture [183 x 191] intentionally omitted <==**

**----- Start of picture text -----**<br>
┌─────────────────┐ ┌─────────────────┐<br>│ Agent A │ │ Agent B │<br>│ (Architect) │ │ (Executor) │<br>└────────┬────────┘ └────────┬────────┘<br> │ │<br> │ 1. Complete task │<br> │ │<br> │ 2. Write handoff.json │<br> │ - Task summary │<br> │ - Context │<br> │ - Constraints │<br> │ - Success criteria │<br> │ │<br> │ 3. Write completion signal │<br> ├─────────────────────────────────────→│<br> │ │<br> │ 4. Read handoff.json<br> │ │<br> │ 5. Validate schema<br> │ │<br> │ 6. Load context<br> │ │<br> │ 7. Execute task<br> │ │<br> │ 8. SubagentStop hook │<br> │◄────────────────────────────────────┤<br> │ validates completion │<br> │ │<br>**----- End of picture text -----**<br>

## **8. External Engine Integration (Gemini)**

## **8.1 Integration Architecture**

**==> picture [265 x 204] intentionally omitted <==**

**----- Start of picture text -----**<br>
┌────────────────────────────────────────────────────────────────────<br>│ GEMINI EXTERNAL ENGINE INTEGRATION<br>│<br>└────────────────────────────────────────────────────────────────────<br>┌─────────────────┐ ┌─────────────────┐<br>┌─────────────────┐<br>│ Claude Code │ │ Gemini CLI │ │ Gemini<br>API │<br>│ Orchestrator │────────→│ Wrapper │────────→│ (Flash<br>2.0) │<br>│ │ pipe │ │ HTTP │<br>│<br>│ • Detects need │ │ • Formats │ │ • 1M<br>context │<br>│ • Prepares │ │ request │ │ •<br>Processes │<br>│ context │ │ • Handles │ │ • Returns<br>│<br>│ • Parses │◄────────│ response │◄────────│<br>response │<br>│ response │ pipe │ • Manages │ HTTP │<br>│<br>│ │ │ errors │ │<br>│<br>└─────────────────┘ └─────────────────┘<br>└─────────────────┘<br>**----- End of picture text -----**<br>

**8.2 Invocation Patterns**

**Pattern 1: Large Codebase Analysis**

- _# Orchestrator detects token estimate > 50K_

- _# Invokes Gemini via CLI pipe_ gemini-cli analyze \

--protocol codebase-analysis \ --input-dir ./src \ --output .claude/tmp/codebase_analysis.json \ --max-tokens 100000

**Pattern 2: Document Synthesis (Parallel Workers)**

- _# For documents exceeding single-call limits_

_# Spin up parallel Gemini workers with overlapping windows_ **for** chunk **in** $(split_document_with_overlap "$DOCUMENT" 40000 10000) **; do** gemini-cli summarize \ --input "$chunk" \ --output ".claude/tmp/summary_${i}.json" **& done** wait _# Synthesis phase_ gemini-cli synthesize \ --inputs ".claude/tmp/summary\_\*.json" \ --output ".claude/tmp/final_synthesis.md"

- **8.3 Handover Protocol Format**
  - # Gemini Handover Protocol

## Task _[_ Clear description of what Gemini should accomplish _]_ ## Context

- _[_ Relevant background information _]_

- ## Input Files

- path/to/file1.py - path/to/file2.py ## Constraints - Maximum output tokens: 10000 - Focus areas: _[_ specific aspects _]_ - Exclude: _[_ what to ignore _]_

## Expected Output Format

_[_ Structured format specification _]_

## Success Criteria

- [ ] Criterion 1 - [ ] Criterion 2

## **8.4 When to Route to Gemini**

**Condition Action** Token estimate > 50,000 Route to Gemini Token estimate > 100,000 FORCE Gemini Files > 20 Consider Gemini scout first Full repo analysis requested Gemini required Document synthesis > 30 pages Parallel Gemini workers

## **Summary**

This technical architecture provides:

1. **Deterministic control** — Bash hooks enforce routing without LLM interpretation

2. **Transparent state** — All inter-agent communication via humanreadable files

3. **Optimal model allocation** — Each model tier serves its appropriate use cases

4. **Scalable context** — Gemini integration handles workloads beyond Claude’s limits

5. **Persistent learning** — Memory pipeline captures and retrieves organizational knowledge

The architecture is designed for evolution. The following sections describe the novel innovations that enable progressive autonomy and the implementation roadmap for realizing the full vision.
