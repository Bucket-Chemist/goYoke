# Part VII: Tradeoff Matrices & Decision Frameworks 

## **Structured Decision Support for Architectural Choices** 

## **1. RAG Implementation Tradeoffs** 

**1.1 Retrieval Method Comparison** 

|**Method**|**Recall (top-**<br>**20)**|**Latency**|**Dependencies**|**Best For**|
|---|---|---|---|---|
|||||<100|
|grep +<br>frontmatter|~75%|<10ms|None|fles,<br>exact|
|||||matches|
|||||100-1000|
|BM25|89%|20-50ms|rank_bm25|fles,|
|||||keyword|
|||||>1000|
|Vector||50-|Embedding|fles,|



**==> picture [200 x 47] intentionally omitted <==**

**----- Start of picture text -----**<br>
embeddings 91.7% 200ms model, DB semantic<br>Large<br>Hybrid(BM25 + 93-95% 100-300ms Both scale,mixed<br>vector)<br>queries<br>**----- End of picture text -----**<br>


_Source: XetHub benchmark, 2024_ 

**==> picture [209 x 370] intentionally omitted <==**

**----- Start of picture text -----**<br>
1.2 Decision Tree: When to Upgrade Retrieval<br>                             START<br>                               │<br>                               ▼<br>                   ┌───────────────────────┐<br>                   │ Memory file count?    │<br>                   └───────────┬───────────┘<br>                               │<br>             ┌─────────────────┼─────────────────┐<br>             │                 │                 │<br>             ▼                 ▼                 ▼<br>         < 100            100-500            > 500<br>             │                 │                 │<br>             ▼                 ▼                 ▼<br>   ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐<br>   │ grep + YAML     │ │ Implement BM25  │ │ Evaluate hybrid │<br>   │ frontmatter     │ │                 │ │ (BM25 + sqlite- │<br>   │                 │ │ ~4-6 hours      │ │ vec)            │<br>   │ Current state   │ │                 │ │                 │<br>   │ No action       │ │                 │ │ ~20-30 hours    │<br>   └─────────────────┘ └─────────────────┘ └─────────────────┘<br>                               │<br>                               ▼<br>                   ┌───────────────────────┐<br>                   │ Query types?          │<br>                   └───────────┬───────────┘<br>                               │<br>             ┌─────────────────┼─────────────────┐<br>             │                 │                 │<br>             ▼                 ▼                 ▼<br>        Exact match      Mixed keywords    Conceptual/<br>        "JWT refresh"    + concepts        semantic only<br>             │                 │                 │<br>             ▼                 ▼                 ▼<br>   ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐<br>   │ BM25 sufficient │ │ Hybrid optimal  │ │ Vector required │<br>   │                 │ │                 │ │                 │<br>   │ Stay with BM25  │ │ Add sqlite-vec  │ │ Full embedding  │<br>   │                 │ │ alongside BM25  │ │ pipeline        │<br>   └─────────────────┘ └─────────────────┘ └─────────────────┘<br>1.3 Implementation Effort vs. Benefit<br>Upgrade Effort Recall Improvement When Worth It<br>grep → 6 hours +14% Always (>100<br>BM25 files)<br>BM25 → 15 hours +3% >500 files,<br>sqlite-vec semantic queries<br>sqlite-vec→ full 20 hours +2% >2000 files, mixed<br>queries<br>hybrid<br>**----- End of picture text -----**<br>


**Recommendation:** Implement BM25 in Phase 2. Defer vector embeddings until memory exceeds 500 files or semantic retrieval failures exceed 10% of queries. 

## **2. Framework Adoption Tradeoffs** 

**2.1 Component-by-Component Assessment** 

|**Component**||**Adopt?**|**Rationale**|**Risk Level**|
|---|---|---|---|---|
|langchain-<br>anthropic|✅|Consider|Clean provider<br>abstraction|Low|
|langchain-openai|✅|Consider|Multi-provider<br>fexibility|Low|
|LlamaIndex<br>retrievers|✅|Consider|Advanced<br>retrieval<br>patterns|Low|
|LangGraph<br>StateGraph|❌|Avoid|Infrastructure<br>conficts|High|
|LangChain<br>memory|❌|Avoid|Token-<br>ineficient|Medium|
|LangChain<br>chains|❌|Avoid|Debugging<br>complexity|High|
|LangSmith|✅|Consider|Observability<br>value|Low|



## **2.2 Adoption Decision Tree** 

**==> picture [256 x 349] intentionally omitted <==**

**----- Start of picture text -----**<br>
                   ┌───────────────────────┐<br>                   │ Component considered? │<br>                   └───────────┬───────────┘<br>                               │<br>                               ▼<br>                   ┌───────────────────────┐<br>                   │ Does it require       │<br>                   │ PostgreSQL/Redis?     │<br>                   └───────────┬───────────┘<br>                               │<br>             ┌─────────────────┴─────────────────┐<br>             │ YES                               │ NO<br>             ▼                                   ▼<br>   ┌─────────────────┐               ┌───────────────────────┐<br>   │ REJECT          │               │ Does it handle state  │<br>   │                 │               │ we already manage?    │<br>   │ Conflicts with  │               └───────────┬───────────┘<br>   │ file-based      │                           │<br>   │ philosophy      │<br>┌─────────────────┴─────────────────┐<br>   └─────────────────┘         │ YES<br>│ NO<br>                               ▼<br>▼<br>                   ┌─────────────────┐<br>┌───────────────────────┐<br>                   │ REJECT          │               │ Is it<br>isolated/       │<br>                   │                 │               │ stateless<br>utility?    │<br>                   │ Duplicates      │<br>└───────────┬───────────┘<br>                   │ existing        │                           │<br>                   │ capability      │<br>┌─────────────────┴─────────────────┐<br>                   └─────────────────┘         │ YES<br>│ NO<br>                                               ▼<br>▼<br>                                   ┌─────────────────┐<br>┌─────────────────┐<br>                                   │ CONSIDER        │<br>│ EVALUATE        │<br>                                   │                 │<br>│ CAREFULLY       │<br>                                   │ Low risk,       │<br>│                 │<br>                                   │ useful utility  │<br>│ May introduce   │<br>                                   └─────────────────┘<br>│ coupling        │<br>**----- End of picture text -----**<br>


└─────────────────┘ 

## **2.3 Integration Complexity Ratings** 

|**Integration**|**Complexity**|**Hours**|**Reversibility**|
|---|---|---|---|
|langchain-anthropic adapter|Low|2-4|Easy|
|LangSmith tracing|Low|1-2|Easy|
|LlamaIndex BM25 retriever|Medium|8-12|Moderate|
|LlamaIndex hybrid search|Medium|15-20|Moderate|
|LangGraph for specifc<br>workfow|High|20-40|Dificult|
|Full LangChain adoption|Very High|80-<br>120|Very Dificult|



**Recommendation:** Adopt only isolated utilities (adapters, observability). Never adopt state management or orchestration components. 

## **3. Parallelization Tradeoffs** 

## **3.1 When Parallel Processing Helps** 

|**Task Type**|**Speedup**|**Quality**<br>**Impact**|**Recommendati**|
|---|---|---|---|
|Document extraction|1.4-3x|Neutral|✅Parallelize|
|Structured data parsing|2-4x|Neutral|✅Parallelize|
|Multi-fle analysis|1.5-2x|Slight<br>improvement|✅Parallelize|
|Creative generation|0.8-1.0x|Degraded|❌Sequential|
|Synthesis/summarization|1.0-1.5x|Risk of gaps|⚠Overlapping<br>windows|
|Reasoning chains|0.5-0.8x|Degraded|❌Sequential|



**==> picture [209 x 6] intentionally omitted <==**

## **3.2 Parallelization Decision Matrix** 

**==> picture [265 x 362] intentionally omitted <==**

**----- Start of picture text -----**<br>
┌────────────────────────────────────────────────────────────────────<br>│                        PARALLELIZATION DECISION MATRIX<br>│<br>└────────────────────────────────────────────────────────────────────<br>                             Task requires coherent reasoning?<br>                             ─────────────────────────────────<br>                                        │<br>                   ┌────────────────────┴────────────────────┐<br>                   │ YES                                     │ NO<br>                   ▼                                         ▼<br>       ┌─────────────────────┐<br>┌─────────────────────┐<br>       │ SEQUENTIAL          │               │ Document size?<br>│<br>       │                     │<br>└──────────┬──────────┘<br>       │ Do not parallelize  │                          │<br>       │ reasoning tasks     │<br>┌────────────────┼────────────────┐<br>       └─────────────────────┘         │                │<br>│<br>                                       ▼                ▼<br>▼<br>                                  < 50K tokens    50K-500K tokens<br>> 500K tokens<br>                                       │                │<br>│<br>                                       ▼                ▼<br>▼<br>                           ┌───────────────┐ ┌───────────────┐<br>┌───────────────┐<br>                           │ SINGLE PASS   │ │ PARALLEL      │ │<br>CHUNKED       │<br>                           │               │ │ OVERLAPPING   │ │<br>PARALLEL      │<br>                           │ No overhead   │ │ WINDOWS       │ │ +<br>HIERARCHICAL│<br>                           │ benefit       │ │               │ │<br>REDUCE        │<br>                           │               │ │ 3-5 workers   │ │<br>│<br>                           └───────────────┘ │ 15% overlap   │ │ 5+<br>workers    │<br>                                             └───────────────┘ │<br>15% overlap   │<br>                                                               │<br>Multi-stage   │<br>└───────────────┘<br>**----- End of picture text -----**<br>


## **3.3 Overlapping Window Pattern Specification** 

**==> picture [207 x 122] intentionally omitted <==**

**----- Start of picture text -----**<br>
When to use:  Document synthesis where boundary gaps would cause<br>information loss<br>Configuration:<br>overlapping_windows :<br>window_size : 40000   # tokens per worker<br>overlap :  15%         # 6000 tokens overlap<br>max_workers : 5<br>boundary_markers :<br>start_continuation : true<br>end_continuation : true<br>key_entities_at_boundaries : true<br>synthesis :<br>method :  hierarchical_reduce<br>max_reduce_stages : 3<br>**----- End of picture text -----**<br>


**==> picture [185 x 60] intentionally omitted <==**

**----- Start of picture text -----**<br>
Cost calculation:<br>Single pass:     100K tokens × $0.075/1M = $0.0075<br>Parallel (5x):   5 × 46K tokens × $0.075/1M = $0.0173<br>Overhead:        2.3x cost for boundary handling<br>Break-even point:  Only parallelize when: - Time savings > 2x<br>(latency-sensitive) - OR document > 200K tokens (required for<br>processing)<br>**----- End of picture text -----**<br>


## **3.4 Rate Limit Management** 

|**Provider**|**Rate**<br>**Limit**|**Recommended**<br>**Concurrency**|**Backof Strategy**|
|---|---|---|---|
|Gemini<br>Flash|60 RPM|5 workers|Exponential (1s,<br>2s, 4s)|
|Claude<br>API|50 RPM|4 workers|Exponential (2s,<br>4s, 8s)|



|OpenAI|500<br>RPM|10|workers|Linear (1s<br>increments)|
|---|---|---|---|---|



## _# Semaphore pattern for rate limiting_ **import** asyncio 

- GEMINI_SEMAPHORE = asyncio.Semaphore(5) 

**async def** rate_limited_call(func, *args): **async with** GEMINI_SEMAPHORE: **return await** func(*args) 

## **4. Autonomy Level Tradeoffs** 

## **4.1 Risk vs. Efficiency by Level** 

||**Level**|**Human**<br>**Efort**|**Error Rate**<br>**Risk**|**Suitable Tasks**|
|---|---|---|---|---|
|L1|Operator|100%|Baseline|All (initial)|
|L2<br>Collaborator||50-70%|Baseline|Suggestions visible|
|L3<br>Consultant||20-40%|+0.5-1%|High-confdence<br>routine|
|L4|Approver|5-15%|+1-2%|Well-defned routine|
|L5|Observer|<5%|+2-5%|Fully automated<br>domain|



## **4.2 Category-Specific Recommendations** 

|**Decision Category**|**Max**<br>**Autonomy**|**Rationale**|
|---|---|---|
|Quick approval (simple<br>tasks)|L5|Low risk, high frequency|
|Routing decisions|L4|Formula-based,<br>auditable|
|Scope modifcations|L3|Requires judgment|
|Security-related|L2|High consequence|
|Architecture changes|L2|High consequence|
|Memory curation|L3|Reversible, moderate<br>impact|



## **4.3 Promotion vs. Demotion Thresholds** 

┌──────────────────────────────────────────────────────────────────── │                         AUTONOMY LEVEL TRANSITIONS │ └──────────────────────────────────────────────────────────────────── 

PROMOTION (requires ALL):                    DEMOTION (requires ANY): 

───────────────────────────────────────────────────────────────────── 

L1 → L2                                      L2 → L1 • 100 decisions in category                  • Success rate drops below 80% • No failures in last 20                     • 3+ consecutive failures • Human requests demotion L2 → L3                                      L3 → L2 • 200 decisions in category                  • Success rate drops below 90% • 95% success rate                           • 2+ consecutive failures • No failures in last 50                     • Human requests demotion L3 → L4                                      L4 → L3 • 500 decisions in category                  • Success rate drops below 95% • 98% success rate                           • Any failure on highpriority task • No failures in last 100                    • Human requests demotion • Human explicitly approves L4 → L5                                      L5 → L4 • 1000+ decisions in category                • Any failure • 99%+ success rate                          • Automatic on first error • Domain fully defined                       • Human requests demotion • Human explicitly approves 

## **4.4 Rollback Triggers** 

|**Trigger**|**Action**|**Recovery**|
|---|---|---|
|Single failure at L5|Demote to L4|50 successes to re-<br>promote|
|3 failures at L4|Demote to L3|100 successes to re-<br>promote|
|Success rate drops<br>5%|Demote one level|Rebuild success history|
|Human override|Immediate<br>demotion|Per human specifcation|
|Schema change|Reset to L2|Re-learn patterns|



## **5. Cost Optimization Tradeoffs** 

## **5.1 Tier Routing Sensitivity Analysis** 

|**Routing**<br>**Strategy**|**Avg**<br>**Cost/Session**|**Quality**<br>**Impact**|**When to Use**|
|---|---|---|---|
|Always Haiku|$0.25-0.50|-30%<br>quality|Never (too<br>aggressive)|
|Formula-based<br>(current)|$3-5|Baseline|Default|
|Always Sonnet|$5-10|+5% quality|Quality-critical<br>periods|
|Aggressive Opus|$15-30|+10%<br>quality|Complex<br>refactors only|



## **5.2 Cost Reduction Levers** 

|**Lever**|**Cost**<br>**Impact**|**Implementation**<br>**Efort**|**Risk**|
|---|---|---|---|
|Scout-frst protocol|-20-40%|Already implemented|Low|
|Stricter Opus<br>threshold|-15-25%|1 hour (confg<br>change)|Medium|
|Memory caching|-10-20%|8-12 hours|Low|
|Request batching|-5-15%|4-6 hours|Low|
|Context<br>compression|-20-30%|20-30 hours|Medium|



## **5.3 Weekly Review Budget Allocation** 

|**Component**|**Budget**|**Rationale**|
|---|---|---|
|Memory Synthesis (Sonnet)|$0.30|Aggregation task|
|Systems Architect (Opus)|$1.50|Complex analysis|
|Schema Discovery (Sonnet)|$0.50|Pattern detection|
|Bufer|$0.20|Unexpected needs|
|**Total**|**$2.50**|Infrastructure investment|



## **5.4 Cost Monitoring Decision Framework** 

**==> picture [138 x 20] intentionally omitted <==**

**----- Start of picture text -----**<br>
                   ┌───────────────────────┐<br>                   │ Session cost exceeded │<br>                   │ threshold?            │<br>**----- End of picture text -----**<br>


## └───────────┬───────────┘ 

│ 

**==> picture [198 x 130] intentionally omitted <==**

**----- Start of picture text -----**<br>
             ┌─────────────────┼─────────────────┐<br>             │ No              │ $5 warning      │ $10 limit<br>             ▼                 ▼                 ▼<br>   ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐<br>   │ Continue        │ │ Alert user      │ │ Require         │<br>   │ normally        │ │                 │ │ confirmation    │<br>   │                 │ │ "Session cost   │ │                 │<br>   │                 │ │ at $X. Continue │ │ "Session at $10 │<br>   │                 │ │ or review?"     │ │ Confirm to      │<br>   │                 │ │                 │ │ continue"       │<br>   └─────────────────┘ └─────────────────┘ └─────────────────┘<br>                                                   │<br>                                                   ▼<br>                                       ┌───────────────────────┐<br>                                       │ Log for weekly review │<br>                                       │ Analyze routing       │<br>                                       │ decisions that led    │<br>                                       │ to high cost          │<br>                                       └───────────────────────┘<br>**----- End of picture text -----**<br>


## **5.5 Shadow Deployment Cost Ceiling** 

**Scenario Max Shadow Cost Rationale** $5 over shadow Bounded New agent validation period investment Routing threshold $2 per A/B Quick validation 

|test||comparison||
|---|---|---|---|
|Schema<br>test|migration|$3 per migration|One-time cost|



**Rule:** Shadow deployments that exceed 2x expected cost are autoterminated and flagged for review. 

## **6. Decision Framework Summary** 

## **6.1 Quick Reference Decision Table** 

|**Decision**|**Default**<br>**Choice**|**Reconsider When**|
|---|---|---|
|Retrieval method|BM25|>500 fles or semantic<br>failures|
|Framework<br>adoption|Bespoke|Never for orchestration|
|Parallelization|Sequential|>50K tokens OR extraction<br>task|
|Autonomy level|L2 start|Success thresholds met|
|Cost optimization|Formula<br>routing|Budget pressure|



## **6.2 Reversibility Assessment** 

|**Decision**|**Reversibility**|**Lock-in Risk**|
|---|---|---|
|Add BM25 retrieval|Easy|None|
|Add vector embeddings|Moderate|Data migration|
|Adopt LangGraph|Dificult|High|
|Increase autonomy|Easy|None|
|Decrease thresholds|Easy|None|
|Shadow new agent|Easy|None|
|Promote agent to production|Moderate|Dependent systems|



## **6.3 Decision Logging** 

All significant architectural decisions should be logged: 

_# .claude/memory/decisions/YYYY-MM-DD-decision-topic.md_ --title **:** "Decision: [Brief title]" created **:** YYYY-MM-DD category **:** decisions tags **: [** architecture **,** tradeoff **,** relevant-area **]** status **:** active summary **:** "One-line summary of what was decided" --- 

_## Context_ 

Why this decision was needed. 

_## Options Considered_ 1. Option A - pros/cons 2. Option B - pros/cons 

_## Decision_ What was chosen. 

_## Rationale_ Why this option was selected. 

_## Consequences_ What changes as a result. 

_## Review Trigger_ When to revisit this decision. 

