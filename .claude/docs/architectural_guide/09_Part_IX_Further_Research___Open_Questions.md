# Part IX: Further Research & Open Questions

## **Exploration Priorities and Unresolved Architectural Decisions**

## **1. Immediate Research Priorities (Months 1-3)**

**1.1 Embedding Model Selection for Hybrid RAG**

**Question:** When BM25 proves insufficient, which embedding model provides best cost/performance tradeoff for local deployment?

**Candidates:** | Model | Dimensions | Speed | Quality | License | |——-| ————|——-|———|———| | all-MiniLM-L6-v2 | 384 | Fast | Good | Apache 2.0 | | bge-small-en-v1.5 | 384 | Fast | Better | MIT | | nomicembed-text-v1 | 768 | Medium | Best | Apache 2.0 | | mxbai-embedlarge | 1024 | Slow | Best | Apache 2.0 |

**Research tasks:** - [ ] Benchmark recall on sample memory corpus - [ ] Measure embedding generation latency on target hardware - [ ] Evaluate storage requirements per model - [ ] Test hybrid (BM25 + embedding) vs pure embedding

**Success criteria:** Identify model achieving >90% recall with <100ms embedding time per query. **Priority:** High (blocks Phase 4 hybrid retrieval)

## **1.2 sqlite-vec vs LanceDB Evaluation**

**Question:** Which vector storage solution best fits the local-first, minimal-dependency philosophy?

**Comparison:** | Aspect | sqlite-vec | LanceDB | |——–|————|———| | Dependencies | Zero (pure C) | Rust runtime | | Disk format | SQLite extension | Lance columnar | | Query speed | Good | Excellent | | Filtering | SQL native | Custom | | Maturity | Newer | Established | | Maintenance | Single developer | Funded company |

**Research tasks:** - [ ] Install and test both on target environment - [ ] Benchmark query performance at 500, 1000, 5000 vectors - [ ] Evaluate hybrid search capabilities (vector + metadata filter) - [ ] Assess maintenance/update burden

**Success criteria:** Select solution with <50ms query time, no external service dependencies.

**Priority:** Medium (supports Phase 4 but not blocking)

## **1.3 Gemini Rate Limit Characterization**

**Question:** What are actual rate limits and optimal concurrency for Gemini Flash in parallel worker scenarios?

**Known:** - Documented: 60 RPM for free tier - Unknown: Burst behavior, token-based limits, error recovery patterns

**Research tasks:** - [ ] Empirically test sustained throughput - [ ] Characterize rate limit error responses - [ ] Identify optimal backoff parameters - [ ] Test token-based vs request-based limits

**Success criteria:** Document reliable concurrency level and backoff strategy for parallel synthesis.

**Priority:** High (blocks Phase 4-5 parallel processing)

## **1.4 Context Compression Techniques**

**Question:** Can context compression reduce costs without degrading quality?

**Candidates:** | Technique | Compression | Quality Retention | Implementation | |———–|————-|——————-|—————-| | ACON framework | 26-54% | 95%+ | Medium | | LLMLingua | 20-40% | 90%+ | High | | Selective context | Variable | 85-95% | Low | | Summary caching | 50-80% | 80-90% | Medium |

**Research tasks:** - [ ] Implement selective context (drop low-relevance memory) - [ ] Test ACON on sample conversations - [ ] Measure quality degradation per compression level - [ ] Identify break-even point (cost savings vs quality loss)

**Success criteria:** Achieve 30% context reduction with <5% quality degradation.

**Priority:** Medium (cost optimization enhancement)

## **2. Medium-Term Exploration (Months 4-6)**

## **2.1 RouteLLM Integration Feasibility**

**Question:** Can RouteLLM’s learned routing provide better cost/quality tradeoff than formula-based routing?

**Background:** RouteLLM demonstrates 85% cost reduction while maintaining 95% performance through learned routing. Current formula is deterministic but may be suboptimal.

**Research approach:** 1. Collect routing decisions with outcomes (already planned in Phase 1) 2. Train RouteLLM classifier on historical decisions 3. A/B test learned routing vs formula routing 4. Measure cost and quality differential

**Considerations:** - Adds ML dependency (acceptable if isolated) - Requires training data accumulation period - May conflict with codeenforced routing philosophy

**Decision framework:** - If learned routing shows >15% cost improvement with <2% quality loss → integrate - If improvement <10% → maintain formula (simpler, more transparent) **Priority:** Medium (optimization, not critical path)

## **2.2 Constitutional AI Principles Adaptation**

**Question:** Can Constitutional AI principles improve agent selfcorrection and reduce human oversight needs?

**Concept:** Define principles as guardrails. Enable agents to selfcritique against principles. Train on revised outputs.

**Potential principles for GoGent:** 1. “Prefer lower-cost tier unless quality requirements demand escalation” 2. “Validate assumptions before proceeding with complex operations” 3. “Flag uncertainty explicitly rather than proceeding with low confidence” 4. “Preserve context boundaries in multi-document synthesis” 5. “Capture decision rationale for future learning”

**Research approach:** 1. Define initial principle set (5-10 principles) 2. Implement self-critique prompt in agent definitions 3. Measure impact on routing violations, task failures, human overrides 4. Iterate on principle definitions

**Success criteria:** Reduce human overrides by 20% through principleguided self-correction.

**Priority:** Medium (quality improvement, supports autonomy progression)

## **2.3 Multi-Agent RLHF Implementation**

**Question:** How can captured human decisions be used to fine-tune agent behavior beyond pattern matching?

**Current approach:** Apprenticeship learning via pattern matching on captured decisions.

**RLHF enhancement:** Use decisions as preference data for reward model training.

**Challenges:** - Requires substantial decision volume (1000+) - Finetuning not available for API models - Proxy approaches (prompt optimization) may suffice

**Research approach:** 1. Accumulate decision corpus through Phases 2-3 2. Evaluate prompt optimization techniques (DSPy, automatic prompt engineering) 3. If effective: implement prompt evolution based on decision outcomes 4. If insufficient: document requirements for future fine-tuning capability

**Success criteria:** Measurable improvement in system recommendations matching human decisions. **Priority:** Low (long-term enhancement)

## **2.4 Distributed Execution Patterns**

**Question:** How should architecture evolve if workloads exceed singlemachine capacity?

**Current assumption:** All execution on developer machine (localfirst).

**Scaling scenarios:** 1. Multiple developers sharing memory (knowledge base) 2. Long-running background tasks (overnight analysis) 3. High-throughput batch processing

**Research areas:** - Shared memory synchronization (git-based already supports) - Background task queue (simple file-based job queue) - Distributed agent coordination (not needed near-term) **Priority:** Low (premature optimization risk)

## **3. Long-Term Research (Months 7-12)**

## **3.1 Full Automation Pathways**

**Question:** What is the realistic ceiling for autonomous operation? **Hypothesis:** Level 5 (Observer) autonomy is achievable for welldefined, low-risk decision categories. Most domains will plateau at Level 3-4.

**Research questions:** - What decision categories can safely reach L5? - What safeguards prevent catastrophic autonomous errors? - How to detect novel situations requiring human escalation? - What feedback loops maintain automation quality over time?

**Long-term experiments:** 1. Track L4 categories for extended periods (6+ months) 2. Measure drift in automated decision quality 3. Identify early warning indicators for degradation 4. Document domain-specific automation boundaries

**Priority:** Low (requires foundation from earlier phases)

## **3.2 Cross-Session Learning Optimization**

**Question:** How to maximize learning from limited human interaction across many sessions?

**Current:** Each session captures observations; weekly review synthesizes.

**Optimization opportunities:** - Real-time pattern matching (surface relevant history immediately) - Active learning (identify high-value decisions to capture explicitly) - Transfer learning across project contexts - Collaborative learning across multiple users (if applicable)

**Research approach:** 1. Instrument learning effectiveness metrics 2. Identify bottlenecks in current learning pipeline 3. Prototype targeted improvements 4. Measure impact on autonomy progression speed **Priority:** Low (optimization layer)

## **3.3 Emergent Capability Detection**

**Question:** How to detect when the system has developed capabilities not explicitly designed?

**Context:** Complex adaptive systems can exhibit emergent behaviors. GoGent’s evolution mechanisms (schema discovery, agent spawning) could produce unexpected capabilities.

**Monitoring needs:** - Capability inventory (what can each agent do?) - Usage pattern analysis (what is being used that wasn’t designed?) - Quality assessment of emergent behaviors - Safety evaluation of new capabilities **Priority:** Low (philosophical, long-term governance)

## **4. Academic References**

## **4.1 Key Papers to Monitor**

**Multi-Agent Systems:** - “MAST: A Multi-Agent System Testing Taxonomy” (UC Berkeley, 2025) — Failure mode classification - “MegaAgent: Autonomous Cooperation” (2024) — File-based state persistence patterns - “Agent2Agent Protocol” (Google/Linux Foundation) — Standardization efforts

**Retrieval and RAG:** - “Evaluation of Chunking Strategies” (NVIDIA FinanceBench, 2024) — Optimal chunking parameters - “ColBERT v2” (Stanford) — Late interaction retrieval - “Contextual Retrieval”

(Anthropic, 2024) — BM25 + embeddings hybrid

**Routing and Efficiency:** - “RouteLLM: Learning to Route LLMs” (2024) — Learned routing classification - “FrugalGPT” (Stanford, 2023) — Cascade routing patterns - “ACON: Adaptive Context Optimization” (2024) — Context compression

**Human-AI Collaboration:** - “Levels of Automation” (Sheridan & Verplank) — Autonomy taxonomy - “Constitutional AI” (Anthropic, 2022) — Principle-based self-correction - “Active Learning from Human Feedback” — Decision capture strategies

## **4.2 Research Groups to Follow**

| **Group**             | **Focus**                     | **Relevance**          |
| --------------------- | ----------------------------- | ---------------------- |
| Anthropic<br>Research | AI safety,<br>capabilities    | Primary model provider |
| Google<br>DeepMind    | Multi-agent,<br>large context | Gemini integration     |
| Stanford HAI          | Human-AI<br>interaction       | Autonomy frameworks    |
| UC Berkeley AI        | Agent<br>architectures        | Failure taxonomies     |
| LangChain<br>Research | Framework<br>patterns         | Anti-patterns to avoid |

## **4.3 Conference Tracks of Interest**

| **Conference** | **Track**                      | **Timing** |
| -------------- | ------------------------------ | ---------- |
| NeurIPS        | AI Agents workshop             | December   |
| ICML           | Multi-agent systems            | July       |
| ACL            | Retrieval-augmented generation | Summer     |
| CHI            | Human-AI collaboration         | Spring     |
| AAAI           | Agent architectures            | February   |

## **5. Open Design Questions**

## **5.1 Unresolved Architectural Decisions**

**Question 1: Memory Expiration Policy** - Should facts automatically expire? - How to handle contradictory facts (old vs new)? - What triggers re-verification of established facts?

**Current thinking:** Facts should have optional expires field. Contradictions surface in weekly review. Verification via explicit user confirmation or external validation.

**Question 2: Agent Communication Protocol** - Direct message passing vs shared state file? - Synchronous vs asynchronous handoffs? - How to handle agent timeout mid-task?

**Current thinking:** File-based (current approach) is simpler and auditable. Async by default. Timeout triggers handoff failure and human escalation.

**Question 3: Schema Evolution Coordination** - Who decides when to evolve schemas? - How to coordinate across active sessions? - What’s the rollback strategy?

**Current thinking:** Weekly review proposes; human approves. Session restart required for schema changes. Git-based rollback.

**Question 4: Multi-User Memory Sharing** - Should memory be shared across team members? - How to resolve conflicting preferences? - What’s the privacy model?

**Current thinking:** Out of scope for solo dev. Git-based sharing technically possible. Would require explicit namespacing.

## **5.2 Experimentation Recommendations**

**A/B Testing Candidates:**

| **Experiment**               | **Hypothesis**                          | **Metric**           | **Priority** |
| ---------------------------- | --------------------------------------- | -------------------- | ------------ |
| Opus threshold<br>10 vs 12   | Higher threshold<br>reduces cost        | Cost per<br>session  | High         |
| Scout-frst vs<br>direct      | Scout saves<br>money on simple<br>tasks | Cost,<br>latency     | Medium       |
| BM25 k=3 vs k=5<br>vs k=10   | More results<br>improve context         | Task<br>success      | Medium       |
| Weekly vs<br>biweekly review | Less frequent<br>review is<br>suficient | Learning<br>velocity | Low          |

**Recommended approach:** 1. Implement simple A/B infrastructure (config flag + logging) 2. Run experiments for minimum 2 weeks each 3. Measure with statistical significance 4. Document results in memory/decisions/

## **5.3 Parking Lot (Ideas to Revisit)**

| **Idea**                | **Why Parked**            | **Revisit When** |
| ----------------------- | ------------------------- | ---------------- |
| Voice interface         | Complexity, unclear value | User requests    |
| IDE integration         | Scope creep               | Core stable      |
| Cloud backup            | Conficts with local-frst  | User requests    |
| Real-time collaboration | Multi-user complexity     | Team adoption    |
| Mobile interface        | Platform dependency       | User requests    |

## **6. Research Roadmap Summary**

┌──────────────────────────────────────────────────────────────────── │ RESEARCH TIMELINE │

└────────────────────────────────────────────────────────────────────

MONTHS 1-3 (Immediate) MONTHS 4-6 (Medium-Term)

───────────────────────── ───────────────────────── • Embedding model selection • RouteLLM integration study • sqlite-vec vs LanceDB • Constitutional AI adaptation • Gemini rate limit testing • Prompt optimization experiments • Context compression pilots • Multi-agent RLHF feasibility MONTHS 7-12 (Long-Term) ONGOING ───────────────────────── ───────────────────────── • Full automation boundaries • Academic literature monitoring • Cross-session learning • A/B testing program • Emergent capability detection • Open questions resolution • Distributed patterns (if needed) • Conference attendance

## **7. Decision Log Template**

For tracking research outcomes:

_# .claude/memory/decisions/YYYY-MM-DD-research-finding.md_ --title **:** "Research: [Topic]" created **:** YYYY-MM-DD category **:** decisions tags **: [** research **,** topic-area **]** status **:** active summary **:** "Research finding and decision" ---

_## Research Question_ What we were trying to learn. _## Methodology_ How we investigated. _## Findings_ What we discovered. _## Decision_ What we're doing based on findings. _## Open Questions_ What remains unresolved. _## References_ Sources consulted.
