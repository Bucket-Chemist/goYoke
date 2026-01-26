## **Lisan al-Gaib** 

Architecture & Implementation Guide 

v1.0 — January 2026 

A Self-Evolving Multi-Agent Orchestration System 

## **Part I: Executive Preamble** 

## **Lisan al-Gaib: A Self-Evolving Multi-Agent Orchestration System** 

**1. Vision Statement** 

Lisan al-Gaib is a hybrid multi-agent orchestration architecture that combines Claude Code’s tiered intelligence (Haiku/Sonnet/Opus) with Gemini Flash’s million-token context window, unified by deterministic bash-hook enforcement and file-based state management. The system is designed to evolve: it observes human orchestration decisions, discovers behavioral patterns, and progressively automates routine workflows while maintaining human oversight for critical decisions. 

**The core thesis:** Most AI agent frameworks optimize for developer convenience at the cost of transparency, cost control, and auditability. Lisan al-Gaib inverts this priority—transparency and determinism come first, with sophistication emerging from composition rather than abstraction. 

**What makes this different:** 

1. **Code-enforced routing** — Bash scripts calculate complexity scores and enforce tier selection. The LLM cannot route itself to expensive tiers; the code decides. This eliminates prompt injection risks in routing and provides deterministic cost control. 

2. **File-based memory** — All state is human-readable YAML and JSON files. No database dependencies, no opaque vector stores. grep and find are valid debugging tools. Git tracks all changes. 

3. **Hybrid context architecture** — Claude handles orchestration and generation within its native context limits; Gemini handles large-context operations (codebase analysis, document synthesis) via CLI pipes. Each model operates in its optimal regime. 

4. **Progressive autonomy** — The system captures human decisions, identifies patterns, and proposes automation. Humans approve. Over time, routine decisions automate while novel situations maintain human oversight. 

This is not a framework. It is an architecture pattern with concrete implementation—a set of interlocking components designed to grow more capable through use. 

## **2. Architecture at a Glance** 

┌──────────────────────────────────────────────────────────────────── │                           LISAN AL-GAIB v2.1 │ │                    Hybrid Multi-Agent Orchestration │ └──────────────────────────────────────────────────────────────────── ┌─────────────┐ │    USER     │ └──────┬──────┘ │ ▼ ┌──────────────────────────────────────────────────────────────────── │                         CLAUDE CODE ORCHESTRATOR │ │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐ ┌─────────────┐        │ │  │   HAIKU     │  │   SONNET    │  │    OPUS     │  │   GEMINI │        │ │  │  Scout/QA   │  │  Architect  │  │   Complex   │  │  External │        │ │  │  $0.25/1M   │  │   $3/1M     │  │   $15/1M    │  │   1M ctx │        │ │  └─────────────┘  └─────────────┘  └─────────────┘ └─────────────┘        │ └───────────────────────────────┬──────────────────────────────────── 

│ ┌───────────────────────┼───────────────────────┐ │                       │                       │ ▼                       ▼                       ▼ ┌───────────────┐       ┌───────────────┐       ┌───────────────┐ │  BASH HOOKS   │       │  STATE FILES  │       │    MEMORY     │ │               │       │               │       │               │ │ PreToolUse    │       │ scout_metrics │       │ decisions/    │ │ PostToolUse   │       │ complexity    │       │ sharp-edges/  │ │ SubagentStop  │       │ routing_log   │       │ facts/        │ │ SessionEnd    │       │ handoff.json  │       │ preferences/  │ │               │       │               │       │               │ │ ENFORCEMENT   │       │ .claude/tmp/  │       │ .claude/      │ │    LAYER      │       │               │       │   memory/     │ └───────────────┘       └───────────────┘       └───────────────┘ 

**Component Summary** 

|**Component**|**Role**|**Implementation**|**Cost Profle**|
|---|---|---|---|
|**Claude**<br>**Haiku**|Scout, quick<br>QA, simple<br>tasks|Native subagent|$0.25/1M<br>tokens|
|**Claude**<br>**Sonnet**|Architecture,<br>planning,<br>standard work|Primary tier|$3/1M tokens|
||Complex|||
|**Claude**<br>**Opus**|reasoning,<br>critical|Escalation tier|$15/1M<br>tokens|
||decisions|||
||Large context|||
|**Gemini**|(>100K),|CLI pipe|~$0.075/1M|
|**Flash**|codebase|integration|tokens|
||analysis|||
||Routing|||
|**Bash Hooks**|enforcement,<br>validation,|Shell scripts in<br>.claude/hooks/|Zero marginal<br>cost|
||logging|||
|**State Files**|Inter-agent<br>communication,<br>checkpoints|JSON/YAML in<br>.claude/tmp/|Zero marginal<br>cost|
|**Memory**<br>**System**|Persistent<br>learning,<br>decision history|Markdown/YAML<br>in.claude/memory/|Zero marginal<br>cost|



## **Data Flow Summary** 

1. **User request** enters the orchestrator 2. **Haiku scout** (optional) performs reconnaissance on large codebases 

3. **Complexity calculator** (bash) scores the task: (tokens/10000) + (files/5) + (modules*2) 

4. **PreToolUse hook** enforces tier ceiling based on score 

5. **Selected agent** executes with appropriate context 

6. **PostToolUse hook** logs outcomes, captures decisions 7. **Memory archivist** (periodic) processes decisions into persistent memory 

8. **Weekly review** synthesizes learnings, proposes system improvements 

## **3. Competitive Differentiation** 

**vs. LangChain/LangGraph** 

|**Dimension**|**LangChain/LangGraph**|**Lisan al-Gaib**|
|---|---|---|
|**Routing**|LLM-based or manual|Code-enforced<br>(bash arithmetic)|
|**State**|PostgreSQL + Redis<br>required|File-based<br>(YAML/JSON)|
|**Debugging**|Graph visualization, traces|grep, fnd, git dif|
|**Dependencies**|Heavy (framework +<br>infrastructure)|Minimal (bash, jq,<br>bc)|
|**Latency**<br>**overhead**|~14ms per graph invocation|<1ms (fle I/O)|
|**Token**<br>**overhead**|15-50% increase<br>documented|Controlled (no<br>framework<br>prompts)|
|**Tiered**<br>**routing**|Not native, must implement|First-class feature|
|**Lock-in risk**|High (API instability history)|Zero (standard<br>tools)|



**Key finding from framework analysis:** LangChain consumed 2.7x expected costs in documented cases due to hidden multi-call patterns. Teams that migrated from LangChain to bespoke solutions reported being “happier and more productive.” 

**vs. Pure API Integration** 

|**Dimension**|**Direct API Calls**|**Lisan al-Gaib**|
|---|---|---|
|**Cost control**|Manual, per-call|Automatic tier<br>routing|
|**Context**<br>**management**|Developer responsibility|Hybrid architecture<br>handles|
|**Memory/learning**|None|File-based RAG +<br>evolution|
|**Human**<br>**oversight**|Ad-hoc|Structured (weekly<br>review, approvals)|
|**Scalability**|Linear cost growth|Optimized routing<br>reduces costs|



## **Cost Efficiency Thesis** 

Research on intelligent routing (RouteLLM, FrugalGPT) demonstrates that **85% cost reduction** is achievable while maintaining 95% of toptier performance through appropriate model selection. Lisan al-Gaib implements this via: 

1. **Scout-first protocol** — Haiku assesses complexity before expensive operations 

2. **Complexity-based routing** — Deterministic formula prevents over-escalation 

3. **Cascade fallback** — Start cheap, escalate only on verified need 4. **External offload** — Gemini handles context-heavy work at lower cost 

Target session cost: **$3-5** for typical development sessions touching 5- 20 files. 

## **4. Intellectual Property Summary** 

Lisan al-Gaib introduces **four novel architectural patterns** that represent significant innovations in multi-agent system design. These should be considered for IP protection. 

## **Innovation 1: Emergent Schema Discovery** 

**What it is:** Rather than pre-defining schemas for decision capture, the system accumulates raw behavioral observations. When sufficient data exists (200+ observations, 3+ detectable clusters), a specialized subagent analyzes patterns and _proposes_ a schema. Humans approve before activation. 

**Why it’s novel:** Existing RLHF approaches require pre-defined feedback schemas. This inverts the process—the schema emerges from observed behavior, similar to unsupervised learning applied to system architecture evolution. 

**Competitive advantage:** Systems using this pattern adapt to user workflows rather than forcing users into predefined categories. 

## **Innovation 2: Gap Analysis → Subagent Spawning** 

**What it is:** Weekly architecture review identifies capability gaps (e.g., “document synthesis failed 4 times due to context boundaries”). For gaps that cannot be resolved by configuration changes, the system generates subagent templates—complete agent definitions ready for deployment. 

**Why it’s novel:** Multi-agent systems typically have fixed agent populations. This creates a _generative architecture_ that grows capabilities based on observed deficiencies. 

**Competitive advantage:** The system becomes more capable over time without manual architecture redesign. 

## **Innovation 3: Apprenticeship Learning Model** 

## **(“Agent-in-the-Loop” Inversion)** 

**What it is:** Traditional human-in-the-loop has agents propose, humans approve. This system inverts the model: humans act, the system observes and learns. Over time, a “QC subagent” emerges that replicates human decision-making. The human progressively steps back as the agent demonstrates competence in specific decision categories. 

**Why it’s novel:** This is apprenticeship learning applied to AI orchestration. The agent learns to _become_ the human orchestrator for routine decisions. 

**Competitive advantage:** Achieves automation without sacrificing the nuanced decision-making that humans provide during the learning period. 

**Innovation 4: Code-Enforced Deterministic Routing** 

**What it is:** Routing decisions are made by bash scripts using arithmetic formulas, not by LLMs interpreting prompts. The LLM cannot convince itself to use a more expensive tier—the code enforces the ceiling. 

**Why it’s novel:** Most agent frameworks rely on LLM-based routing, which is susceptible to prompt injection, inconsistent interpretation, and cost overruns. Code-enforced routing is deterministic, auditable, and manipulation-resistant. 

**Competitive advantage:** Predictable costs, auditable decisions, zero routing latency, security against prompt injection attacks on routing. 

## **5. Current Maturity & Capabilities** 

**What Works Today (v2.1)** 

|**Capability**|**Status**|**Notes**|
|---|---|---|
|Tiered agent routing|✅Production|Haiku/Sonnet/Opus<br>+ Gemini|
|Complexity calculation|✅Production|Bash formula<br>implemented|
|PreToolUse enforcement|✅Production|Tier ceiling<br>enforced|
|Scout-frst protocol|✅Production|Haiku<br>reconnaissance|
|Gemini CLI integration|✅Production|Large context<br>ofoad|
|File-based state|✅Production|JSON/YAML in<br>.claude/tmp/|
|Memory directory<br>structure|✅Production|decisions/, sharp-<br>edges/, etc.|
|Cross-platform install|✅Production|Bash + PowerShell<br>installers|
|Explore workfow (7<br>phases)|✅Production|Full<br>implementation|



## **Partial Implementation** 

|**Capability**|**Status**|**Gap**|
|---|---|---|
|||Needs|
|PostToolUse logging|�Partial|structured<br>decision|
|||capture|
|||Hook|
|||exists,|
|SubagentStop validation|�Partial|validation|
|||logic|
|||needed|
|||grep-based,|
|Memory retrieval|�Partial|needs<br>BM25|
|||upgrade|
|||Manual|
|Weekly review|�Partial|process,<br>needs|
|||automation|



**Not Yet Implemented** 

|**Capability**|**Status**|**Roadmap Position**|
|---|---|---|
|Observation accumulation|❌Planned|Phase 4 (Months 4-5)|
|Emergent schema discovery|❌Planned|Phase 4 (Months 4-5)|
|Gap analysis automation|❌Planned|Phase 5 (Months 6-7)|
|Subagent spawning|❌Planned|Phase 5 (Months 6-7)|
|Autonomy level progression|❌Planned|Phase 6 (Months 8-10)|
|Self-improving loop|❌Planned|Phase 7 (Months 11-12)|



**Production Readiness Assessment** 

**Current state:** Solid foundation for cost-controlled multi-agent development work. The routing and enforcement layers are production-ready. The evolution mechanisms (the novel IP) are designed but not implemented. 

**Risk level:** Low for current capabilities; the architecture is conservative and failure modes are well-understood. 

**Scaling readiness:** Tested with sessions touching 5-20 files. Architecture supports larger scale; Gemini offload handles context growth. 

## **6. Strategic Positioning** 

## **Target Use Cases** 

1. **Cost-sensitive AI-assisted development** — Teams with budget constraints who need intelligent routing to control costs while maintaining quality for complex tasks. 

2. **Auditable AI workflows** — Organizations requiring explainable AI decisions with full audit trails (regulated industries, securitysensitive environments). 

3. **Long-running development relationships** — Solo developers or small teams where the AI assistant should learn and improve over months/years of collaboration. 

4. **Hybrid model architectures** — Teams wanting to leverage multiple AI providers (Anthropic + Google) without framework lock-in. 

## **Development Philosophy** 

1. **Transparency over abstraction** — Every decision is traceable to human-readable files. 

2. **Composition over inheritance** — Components are independent and composable; no deep inheritance hierarchies. 

3. **Conservative automation** — Automate only what has been proven safe; maintain human oversight for novel situations. 

4. **Local-first** — No mandatory cloud dependencies; everything runs on developer machines. 

5. **Incremental evolution** — The system grows capabilities through observation and adaptation, not wholesale replacement. 

**12-Month Vision** 

By end of year, Lisan al-Gaib will be a **self-evolving development assistant** that: 

- Recognizes your decision patterns and proposes automation for routine choices Spawns specialized subagents when capability gaps are identified Maintains transparent human oversight for all critical decisions Operates at predictable, optimized costs through intelligent routing 

- Accumulates organizational knowledge in searchable, versioncontrolled memory 

The human orchestrator progressively shifts from operator to supervisor to consultant—not because the AI “takes over,” but because the AI has _learned_ the human’s approach and can be trusted with routine execution. 

## **Summary** 

Lisan al-Gaib represents a fundamentally different approach to AI agent orchestration: 

- **Not a framework** — An architecture pattern with concrete implementation **Not opaque** — Every decision traceable in human-readable files **Not static** — Designed to evolve through observation and adaptation **Not expensive** — Intelligent routing controls costs without sacrificing capability 

The four novel innovations—emergent schema discovery, generative subagent spawning, apprenticeship learning, and code-enforced routing—position this architecture as a unique entry in the multiagent systems space, with potential for significant intellectual property value. 

The following sections provide the technical depth required for implementation. 

