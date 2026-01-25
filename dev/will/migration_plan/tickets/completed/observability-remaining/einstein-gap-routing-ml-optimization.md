# GAP Analysis: Routing Optimization & Agent Performance Benchmarking

**Document ID:** GAP-2026-01-25-routing-ml
**Escalated By:** Einstein + Staff-Architect Synthesis
**Primary Stakeholder Goals:**
1. Routing optimization - learn from outcomes to improve tier/agent selection
2. Agent performance benchmarking - determine which agent is best for what task type
3. Path of least resistance - identify optimal workflows with minimal friction
4. Team makeup optimization - discover best agent combinations for complex tasks

---

## 1. Executive Summary

This document synthesizes findings from Einstein and Staff-Architect critical reviews of GOgent-087 through GOgent-093, with focus on **data capture requirements for long-term ML-based routing optimization**.

**Core Finding:** Current ticket scope captures performance metrics but **misses the labeled outcome data** required for supervised learning on routing decisions. Without explicit success/failure correlation to routing choices, ML systems cannot learn "what works best."

**Recommendation:** Extend GOgent-087/088 to capture a **Routing Decision Audit Trail** and **Agent Outcome Correlation** data that enables future optimization.

---

## 2. Strategic Goals → Data Requirements Mapping

### 2.1 Goal: Routing Optimization (Learn from Outcomes)

**Question to Answer:** "Given this task, which tier/agent should I route to?"

**Required Data:**
| Field | Purpose | Captures |
|-------|---------|----------|
| `task_complexity_signals` | Input features for ML | What triggered the routing decision |
| `detected_patterns` | Pattern matching audit | Which keywords/patterns were detected |
| `selected_tier` | Action taken | haiku/sonnet/opus/external |
| `selected_agent` | Action taken | python-pro, orchestrator, etc. |
| `alternative_options` | Counterfactual | What else was considered |
| `outcome_success` | Reward signal | Did the task complete successfully? |
| `outcome_cost` | Reward signal | Actual tokens/$ consumed |
| `outcome_duration` | Reward signal | Time to completion |
| `escalation_required` | Negative signal | Did we need to escalate? |
| `retry_count` | Negative signal | How many attempts needed? |

**Gap:** Current system only logs **routing violations** (negative examples). No positive examples are captured.

### 2.2 Goal: Agent Performance Benchmarking

**Question to Answer:** "Which agent performs best for task type X?"

**Required Data:**
| Field | Purpose | Captures |
|-------|---------|----------|
| `agent_id` | Identification | Which agent executed |
| `task_type` | Classification | implementation, documentation, search, etc. |
| `task_domain` | Classification | Python, Go, R, infrastructure, etc. |
| `success_rate` | Performance metric | % of tasks completed without escalation |
| `avg_duration_by_type` | Efficiency metric | How long per task type |
| `avg_cost_by_type` | Cost metric | Tokens/$ per task type |
| `error_patterns` | Quality metric | What kinds of errors does this agent make? |
| `tool_preferences` | Behavior metric | Which tools does this agent favor? |

**Gap:** `AgentInvocation` captures agent + outcome but lacks task classification. Cannot group by "task type."

### 2.3 Goal: Path of Least Resistance

**Question to Answer:** "What's the optimal tool sequence for task type X?"

**Required Data:**
| Field | Purpose | Captures |
|-------|---------|----------|
| `tool_sequence` | Trajectory | Ordered list of tools used |
| `sequence_outcome` | Label | Did this sequence succeed? |
| `sequence_duration` | Efficiency | Total time for sequence |
| `backtrack_count` | Friction signal | How many times did we redo steps? |
| `tool_transitions` | Markov chain input | Tool A → Tool B frequency |
| `successful_patterns` | Pattern mining | Common sequences in successful sessions |

**Gap:** Current telemetry captures individual tool calls but not **sequences** or **trajectories**.

### 2.4 Goal: Team Makeup Optimization

**Question to Answer:** "What agent combinations work best for complex tasks?"

**Required Data:**
| Field | Purpose | Captures |
|-------|---------|----------|
| `delegation_chain` | Collaboration graph | orchestrator → python-pro → codebase-search |
| `chain_outcome` | Label | Did the collaboration succeed? |
| `handoff_friction` | Quality metric | Information loss at handoffs |
| `parallel_agents` | Topology | Which agents ran concurrently |
| `agent_interaction_graph` | Network analysis | Who spawned whom, with what context |
| `bottleneck_agent` | Optimization target | Which agent in chain caused delays |

**Gap:** `parent_task_id` exists in `AgentInvocation` but **chain reconstruction** requires post-processing. No explicit collaboration metrics.

---

## 3. Current State Analysis

### 3.1 What We Already Capture (pkg/telemetry)

```go
// AgentInvocation - EXISTS
type AgentInvocation struct {
    Timestamp       string   // ✓ When
    SessionID       string   // ✓ Session correlation
    Agent           string   // ✓ Which agent
    Model           string   // ✓ Model used
    Tier            string   // ✓ Tier used
    DurationMs      int64    // ✓ Performance
    InputTokens     int      // ✓ Cost input
    OutputTokens    int      // ✓ Cost input
    Success         bool     // ✓ Outcome (binary)
    ErrorType       string   // ✓ Failure classification
    TaskDescription string   // ✓ Task context (truncated)
    ParentTaskID    string   // ✓ Delegation chain link
    ToolsUsed       []string // ✓ Tools in this invocation
}
```

**Coverage:** ~60% of routing optimization needs, ~40% of agent benchmarking needs.

### 3.2 What We Already Capture (pkg/session)

```go
// RoutingViolation - EXISTS (negative examples only)
type RoutingViolation struct {
    Agent         string // Which agent violated
    ViolationType string // What rule was broken
    ExpectedTier  string // What should have been used
    ActualTier    string // What was actually used
    Timestamp     int64
}
```

**Coverage:** Only captures failures. No positive routing decisions logged.

### 3.3 What's Missing (The Gap)

| Need | Status | Priority |
|------|--------|----------|
| Routing decision audit (ALL decisions) | **MISSING** | P0 |
| Task type classification | **MISSING** | P0 |
| Tool sequence capture | **MISSING** | P0 |
| Delegation chain reconstruction | Partial (needs aggregation) | P1 |
| Agent collaboration metrics | **MISSING** | P1 |
| Counterfactual alternatives | **MISSING** | P2 |

---

## 4. Proposed Schema Extensions

### 4.1 RoutingDecision (NEW - P0)

Capture EVERY routing decision, not just violations.

```go
// pkg/telemetry/routing_decision.go

// RoutingDecision captures a single routing choice for ML training
type RoutingDecision struct {
    // Identity
    DecisionID  string `json:"decision_id"`  // UUID
    Timestamp   int64  `json:"timestamp"`
    SessionID   string `json:"session_id"`

    // Input Context (Features for ML)
    TaskDescription    string   `json:"task_description"`    // First 500 chars
    TaskType           string   `json:"task_type"`           // "implementation", "search", "documentation", "debug"
    TaskDomain         string   `json:"task_domain"`         // "python", "go", "r", "infrastructure"
    DetectedPatterns   []string `json:"detected_patterns"`   // ["implement", "refactor"] from schema
    ContextWindowUsed  int      `json:"context_window_used"` // Tokens in context
    SessionToolCount   int      `json:"session_tool_count"`  // Tools used so far in session
    RecentSuccessRate  float64  `json:"recent_success_rate"` // Last 10 operations

    // Decision Made (Action)
    SelectedTier      string   `json:"selected_tier"`       // haiku, sonnet, opus, external
    SelectedAgent     string   `json:"selected_agent"`      // python-pro, orchestrator, etc.
    AlternativeTiers  []string `json:"alternative_tiers"`   // What else was considered
    AlternativeAgents []string `json:"alternative_agents"`  // What else matched
    Confidence        float64  `json:"confidence"`          // 0.0-1.0 routing confidence

    // Override Information
    WasOverridden    bool   `json:"was_overridden"`     // User used --force-tier
    OverrideReason   string `json:"override_reason,omitempty"`

    // Outcome (Reward Signal - populated after execution)
    OutcomeSuccess     bool    `json:"outcome_success,omitempty"`
    OutcomeDurationMs  int64   `json:"outcome_duration_ms,omitempty"`
    OutcomeCost        float64 `json:"outcome_cost,omitempty"`
    EscalationRequired bool    `json:"escalation_required,omitempty"`
    RetryCount         int     `json:"retry_count,omitempty"`

    // Correlation
    InvocationID string `json:"invocation_id,omitempty"` // Links to AgentInvocation
}

// Storage: ~/.gogent/routing-decisions.jsonl
// Also: <project>/.claude/memory/routing-decisions.jsonl
```

**ML Usage:**
- Features: TaskType, TaskDomain, DetectedPatterns, ContextWindowUsed, RecentSuccessRate
- Action: SelectedTier, SelectedAgent
- Reward: OutcomeSuccess, OutcomeCost, EscalationRequired

### 4.2 ToolEvent Extension (P0)

Extend the proposed ToolEvent with sequence tracking.

```go
// pkg/telemetry/tool_event.go

type ToolEvent struct {
    // ... existing AgentInvocation fields ...

    // Tool-specific
    ToolName     string `json:"tool_name"`
    ToolCategory string `json:"tool_category"` // file, execution, search, task, web

    // Sequence tracking (NEW)
    SequenceIndex    int      `json:"sequence_index"`     // Position in session (0, 1, 2...)
    PreviousTools    []string `json:"previous_tools"`     // Last 5 tools
    PreviousOutcomes []bool   `json:"previous_outcomes"`  // Success of last 5

    // Trajectory correlation (NEW)
    TaskBatchID string `json:"task_batch_id,omitempty"` // Groups related tool calls
    IsRetry     bool   `json:"is_retry"`                // Was this a retry of failed operation?
    RetryOf     string `json:"retry_of,omitempty"`      // EventID of original attempt
}
```

**Trajectory Reconstruction:**
```python
# Post-processing to extract successful patterns
df = pd.read_json('tool-events.jsonl', lines=True)
successful_sequences = df[df['success'] == True].groupby('task_batch_id')['tool_name'].apply(list)
# → ["Glob", "Read", "Edit", "Read"] patterns
```

### 4.3 AgentCollaboration (NEW - P1)

Capture delegation chains and collaboration patterns.

```go
// pkg/telemetry/collaboration.go

// AgentCollaboration captures a delegation relationship
type AgentCollaboration struct {
    CollaborationID string `json:"collaboration_id"` // UUID
    Timestamp       int64  `json:"timestamp"`
    SessionID       string `json:"session_id"`

    // Delegation relationship
    ParentAgent     string `json:"parent_agent"`     // orchestrator
    ChildAgent      string `json:"child_agent"`      // python-pro
    DelegationType  string `json:"delegation_type"`  // "spawn", "escalate", "parallel"

    // Context transfer
    ContextSize     int    `json:"context_size"`     // Tokens passed to child
    TaskDescription string `json:"task_description"` // What was delegated

    // Outcome
    ChildSuccess    bool   `json:"child_success"`
    ChildDurationMs int64  `json:"child_duration_ms"`
    HandoffFriction string `json:"handoff_friction,omitempty"` // "context_loss", "misunderstanding"

    // Chain position
    ChainDepth int    `json:"chain_depth"` // 0 = root, 1 = first delegation, etc.
    RootTaskID string `json:"root_task_id"` // Original task that spawned chain
}

// Storage: ~/.gogent/agent-collaborations.jsonl
```

**Team Makeup Analysis:**
```sql
-- Which agent pairs have highest success rate?
SELECT
    parent_agent,
    child_agent,
    COUNT(*) as collaborations,
    AVG(CASE WHEN child_success THEN 1 ELSE 0 END) as success_rate
FROM agent_collaborations
GROUP BY parent_agent, child_agent
ORDER BY success_rate DESC;

-- Result: orchestrator → python-pro: 94% success
--         architect → go-pro: 91% success
--         orchestrator → codebase-search: 89% success
```

### 4.4 TaskClassification Helper (P0)

Auto-classify tasks for ML labeling.

```go
// pkg/telemetry/task_classifier.go

// ClassifyTask extracts task type and domain from description
func ClassifyTask(description string) (taskType, taskDomain string) {
    // Task type detection
    taskType = "unknown"
    typePatterns := map[string][]string{
        "implementation": {"implement", "create", "add", "build", "write"},
        "search":         {"find", "search", "locate", "where", "which"},
        "documentation":  {"document", "readme", "docstring", "comment"},
        "debug":          {"debug", "fix", "error", "issue", "bug"},
        "refactor":       {"refactor", "clean", "restructure", "reorganize"},
        "review":         {"review", "check", "audit", "validate"},
        "test":           {"test", "verify", "assert", "coverage"},
    }

    descLower := strings.ToLower(description)
    for tType, patterns := range typePatterns {
        for _, p := range patterns {
            if strings.Contains(descLower, p) {
                taskType = tType
                break
            }
        }
    }

    // Domain detection
    taskDomain = "unknown"
    domainPatterns := map[string][]string{
        "python":         {"python", ".py", "pip", "pytest", "pyproject"},
        "go":             {"go", "golang", ".go", "cobra", "bubbletea"},
        "r":              {"r ", "shiny", "golem", ".r", "tidyverse"},
        "javascript":     {"javascript", "typescript", ".js", ".ts", "npm"},
        "infrastructure": {"docker", "kubernetes", "ci/cd", "deploy", "terraform"},
        "documentation":  {"readme", "docs", "markdown", ".md"},
    }

    for domain, patterns := range domainPatterns {
        for _, p := range patterns {
            if strings.Contains(descLower, p) {
                taskDomain = domain
                break
            }
        }
    }

    return taskType, taskDomain
}
```

---

## 5. Data Flow Architecture

### 5.1 Capture Points

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Claude Code Session                          │
└─────────────────────────────────────────────────────────────────────┘
                                    │
        ┌───────────────────────────┼───────────────────────────┐
        │                           │                           │
        ▼                           ▼                           ▼
┌───────────────┐         ┌───────────────┐         ┌───────────────┐
│  PreToolUse   │         │ PostToolUse   │         │ SubagentStop  │
│  (validate)   │         │ (sharp-edge)  │         │ (endstate)    │
└───────┬───────┘         └───────┬───────┘         └───────┬───────┘
        │                         │                         │
        ▼                         ▼                         ▼
┌───────────────┐         ┌───────────────┐         ┌───────────────┐
│RoutingDecision│         │  ToolEvent    │         │AgentCollaboration│
│  (if Task)    │         │  (always)     │         │  (always)     │
└───────┬───────┘         └───────┬───────┘         └───────┬───────┘
        │                         │                         │
        └─────────────────────────┼─────────────────────────┘
                                  │
                                  ▼
                    ┌─────────────────────────┐
                    │   ~/.gogent/ (global)   │
                    │   .claude/memory/       │
                    │   (project-scoped)      │
                    └─────────────────────────┘
                                  │
                                  ▼
                    ┌─────────────────────────┐
                    │   SessionEnd Archive    │
                    │   - Aggregate metrics   │
                    │   - Update handoff      │
                    │   - Feed to Gemini audit│
                    └─────────────────────────┘
```

### 5.2 Storage Schema

```
~/.gogent/
├── routing-decisions.jsonl      # NEW: All routing decisions with outcomes
├── tool-events.jsonl            # NEW: All tool calls with sequences
├── agent-collaborations.jsonl   # NEW: Delegation chains
├── agent-invocations.jsonl      # EXISTS: Agent-level metrics
└── failure-tracker.jsonl        # EXISTS: Debugging loops

<project>/.claude/memory/
├── routing-decisions.jsonl      # Project-scoped mirror
├── tool-events.jsonl            # Project-scoped mirror
├── agent-collaborations.jsonl   # Project-scoped mirror
├── agent-invocations.jsonl      # EXISTS
├── handoffs.jsonl               # EXISTS: Session summaries
└── pending-learnings.jsonl      # EXISTS: Sharp edges
```

### 5.3 Aggregation for ML Training

```go
// Weekly batch job (or Gemini audit)

type MLTrainingDataset struct {
    // Routing optimization dataset
    RoutingExamples []struct {
        Features RoutingFeatures
        Action   RoutingAction
        Reward   RoutingReward
    }

    // Agent benchmarking dataset
    AgentBenchmarks map[string]AgentPerformance // agent_id → metrics

    // Sequence patterns
    SuccessfulSequences [][]string // [[Glob, Read, Edit], [Search, Read, Read]]
    FailureSequences    [][]string

    // Collaboration graphs
    CollaborationEdges []CollaborationEdge
}

type RoutingFeatures struct {
    TaskType          string
    TaskDomain        string
    DetectedPatterns  []string
    ContextWindowUsed int
    RecentSuccessRate float64
}

type RoutingAction struct {
    SelectedTier  string
    SelectedAgent string
}

type RoutingReward struct {
    Success   bool
    Cost      float64
    Duration  int64
    Escalated bool
}

type AgentPerformance struct {
    Agent            string
    TotalInvocations int
    ByTaskType       map[string]TaskTypeMetrics
    ByTaskDomain     map[string]DomainMetrics
    TopTools         []string
    CommonErrors     []string
}
```

---

## 6. Implementation Roadmap

### Phase 1: Foundation (GOgent-087/088 Revision)

**Week 1:**
1. Create `pkg/telemetry/tool_event.go` with sequence tracking
2. Create `pkg/telemetry/task_classifier.go`
3. Add `PreviousTools`, `SequenceIndex` to ToolEvent
4. Integrate into `gogent-sharp-edge` PostToolUse handler

**Deliverables:**
- ToolEvent with sequence data captured
- Task classification on every event
- Dual-write to global + project storage

### Phase 2: Routing Decision Capture (NEW TICKET)

**Week 2:**
1. Create `pkg/telemetry/routing_decision.go`
2. Modify `gogent-validate` to log ALL routing decisions
3. Implement outcome correlation (link decision → invocation result)
4. Add to handoff artifacts

**Deliverables:**
- RoutingDecision logged on every Task() call
- Outcome populated after SubagentStop
- Queryable via `gogent-archive routing-decisions`

### Phase 3: Collaboration Tracking (NEW TICKET)

**Week 3:**
1. Create `pkg/telemetry/collaboration.go`
2. Modify `gogent-agent-endstate` to log collaborations
3. Reconstruct delegation chains
4. Calculate handoff friction metrics

**Deliverables:**
- AgentCollaboration logged on every SubagentStop
- Chain depth tracking
- Parent-child success correlation

### Phase 4: Aggregation & Analysis (NEW TICKET)

**Week 4:**
1. Create `cmd/gogent-ml-export/main.go`
2. Implement feature extraction from JSONL
3. Generate training datasets
4. Integrate with Gemini benchmark-audit protocol

**Deliverables:**
- ML-ready CSV/Parquet export
- Weekly aggregation script
- Gemini audit integration

---

## 7. Minimal Viable Data Capture (Immediate)

If full implementation is too heavy, capture **at minimum**:

### Absolutely Essential (P0)

```go
// In every ToolEvent:
type ToolEventMinimal struct {
    ToolName      string   `json:"tool_name"`
    Success       bool     `json:"success"`
    DurationMs    int64    `json:"duration_ms"`
    PreviousTools []string `json:"previous_tools"` // Last 5
    TaskType      string   `json:"task_type"`      // Auto-classified
}

// On every Task() routing decision:
type RoutingDecisionMinimal struct {
    SelectedTier   string `json:"selected_tier"`
    SelectedAgent  string `json:"selected_agent"`
    TaskType       string `json:"task_type"`
    OutcomeSuccess bool   `json:"outcome_success"` // Populated after
}
```

### "Nice to Have" (P1)

- `AlternativeTiers` / `AlternativeAgents` for counterfactual analysis
- `ContextWindowUsed` for context budget optimization
- `ChainDepth` for delegation pattern analysis

### "Future" (P2)

- `Confidence` scoring
- `HandoffFriction` classification
- Full `MLTrainingExample` schema

---

## 8. Queries You'll Be Able to Run

### Routing Optimization

```sql
-- Which tier works best for implementation tasks?
SELECT
    selected_tier,
    AVG(CASE WHEN outcome_success THEN 1 ELSE 0 END) as success_rate,
    AVG(outcome_cost) as avg_cost,
    COUNT(*) as sample_size
FROM routing_decisions
WHERE task_type = 'implementation'
GROUP BY selected_tier
ORDER BY success_rate DESC;
```

### Agent Benchmarking

```sql
-- Best agent for Python implementation?
SELECT
    agent,
    AVG(CASE WHEN success THEN 1 ELSE 0 END) as success_rate,
    AVG(duration_ms) as avg_duration,
    COUNT(*) as invocations
FROM agent_invocations
WHERE task_domain = 'python' AND task_type = 'implementation'
GROUP BY agent
ORDER BY success_rate DESC, avg_duration ASC;
```

### Path of Least Resistance

```python
# Most common successful tool sequences
from collections import Counter

successful_sessions = df[df['session_success'] == True]
sequences = successful_sessions.groupby('session_id')['tool_name'].apply(tuple)
Counter(sequences).most_common(10)

# → [('Glob', 'Read', 'Edit'): 234,
#    ('Read', 'Edit', 'Bash'): 189,
#    ('Grep', 'Read', 'Read', 'Edit'): 156]
```

### Team Makeup

```sql
-- Best orchestrator → specialist pairings
SELECT
    parent_agent,
    child_agent,
    AVG(CASE WHEN child_success THEN 1 ELSE 0 END) as success_rate,
    COUNT(*) as collaborations
FROM agent_collaborations
WHERE parent_agent = 'orchestrator'
GROUP BY parent_agent, child_agent
HAVING COUNT(*) > 10
ORDER BY success_rate DESC;
```

---

## 9. Revised Ticket Recommendations

Based on this GAP analysis, revise tickets as follows:

### GOgent-087-REVISED: ToolEvent with Sequence Tracking
- Add `PreviousTools`, `SequenceIndex`, `TaskType`, `TaskDomain`
- Integrate task classifier
- **Time:** 2h (was 1h)

### GOgent-088-REVISED: Tool Event Logging with ML Fields
- Dual-write architecture
- XDG compliance
- Include minimal routing decision capture on Task() events
- **Time:** 1.5h (was 1.5h)

### NEW: GOgent-087b: Routing Decision Capture
- Full `RoutingDecision` struct
- Log on PreToolUse for Task()
- Correlate outcome on SubagentStop
- **Time:** 2h

### NEW: GOgent-088b: Collaboration Tracking
- `AgentCollaboration` struct
- Log on SubagentStop
- Chain reconstruction
- **Time:** 1.5h

### NEW: GOgent-089b: ML Export CLI
- `gogent-ml-export` command
- CSV/Parquet output
- Feature extraction
- **Time:** 2h

---

## 10. Success Metrics

After implementation, you should be able to:

| Metric | Target | How to Measure |
|--------|--------|----------------|
| Routing decision coverage | 100% | All Task() calls logged |
| Outcome correlation | >95% | Decisions linked to invocation results |
| Task classification accuracy | >85% | Manual audit sample |
| Sequence capture | 100% | PreviousTools populated |
| Collaboration tracking | >90% | SubagentStop events logged |

---

## 11. Document Sign-Off

**Analysis Prepared By:** Einstein + Staff-Architect Synthesis
**Date:** 2026-01-25
**Status:** Ready for Implementation Planning

**Recommended Next Step:**
1. Revise GOgent-087/088 per Section 9
2. Create GOgent-087b/088b/089b tickets
3. Implement Phase 1-4 roadmap

---

## Addendum A: Understanding Task Evaluation Extensions

**Added:** 2026-01-25
**Purpose:** Minimal field extensions to evaluate document understanding and codebase understanding tasks, including swarm-based multi-agent coordination with overlapping context and border stitching.

### A.1 Additional Task Types (TaskClassifier)

Add to existing `typePatterns` in `task_classifier.go`:

```go
// Understanding task types
"document_understanding": {"summarize", "extract", "analyze document", "key points", "what does this say", "main arguments"},
"codebase_understanding": {"how does", "explain the", "trace through", "architecture", "map the", "what is the structure"},
"synthesis":              {"synthesize", "combine", "merge findings", "consolidate", "bring together"},
```

**Cost:** 0 additional bytes - already classifying tasks.

---

### A.2 Understanding Quality Fields (RoutingDecision)

Add 4 optional fields to `RoutingDecision` struct for understanding task outcomes:

```go
// Understanding quality (populated post-execution, omitempty for non-understanding tasks)
UnderstandingCompleteness float64 `json:"understanding_completeness,omitempty"` // % of target covered (0.0-1.0)
UnderstandingAccuracy     float64 `json:"understanding_accuracy,omitempty"`     // If verifiable against ground truth
SynthesisCoherence        float64 `json:"synthesis_coherence,omitempty"`        // Logical flow score (0.0-1.0)
RequiredFollowUp          bool    `json:"required_follow_up,omitempty"`         // Needed additional passes?
```

**Cost:** ~20 bytes per routing decision (only populated for understanding tasks).

**Usage:**
```sql
-- Best tier for document understanding?
SELECT selected_tier,
       AVG(understanding_completeness) as coverage,
       AVG(synthesis_coherence) as coherence
FROM routing_decisions
WHERE task_type = 'document_understanding'
GROUP BY selected_tier;
```

---

### A.3 Swarm Coordination Fields (AgentCollaboration)

Add 5 optional fields to `AgentCollaboration` struct for multi-agent swarm evaluation:

```go
// Swarm coordination (populated when agent is part of parallel swarm, omitempty otherwise)
IsSwarmMember         bool    `json:"is_swarm_member,omitempty"`          // True if parallel worker
SwarmPosition         int     `json:"swarm_position,omitempty"`           // 0, 1, 2... position in sequence
OverlapWithPrevious   float64 `json:"overlap_with_previous,omitempty"`    // 0.0-1.0 context overlap ratio
AgreementWithAdjacent float64 `json:"agreement_with_adjacent,omitempty"`  // Border agreement rate (0.0-1.0)
InformationLoss       float64 `json:"information_loss,omitempty"`         // % lost at stitch point (0.0-1.0)
```

**Cost:** ~30 bytes per collaboration event (only populated for swarm members).

**Usage:**
```sql
-- What overlap % minimizes information loss?
SELECT
    ROUND(overlap_with_previous * 10) / 10 as overlap_bucket,
    AVG(agreement_with_adjacent) as avg_agreement,
    AVG(information_loss) as avg_loss
FROM agent_collaborations
WHERE is_swarm_member = true
GROUP BY overlap_bucket
ORDER BY overlap_bucket;

-- Best agent pairs for swarm work?
SELECT parent_agent, child_agent,
       AVG(agreement_with_adjacent) as border_agreement,
       AVG(information_loss) as info_loss
FROM agent_collaborations
WHERE is_swarm_member = true
GROUP BY parent_agent, child_agent
ORDER BY border_agreement DESC;
```

---

### A.4 Understanding Context Fields (ToolEvent)

Add 3 optional fields to `ToolEvent` struct for understanding task context:

```go
// Understanding task context (omitempty for code generation tasks)
TargetSize       int64   `json:"target_size,omitempty"`        // Total tokens/pages being understood
CoverageAchieved float64 `json:"coverage_achieved,omitempty"`  // What % of target was analyzed (0.0-1.0)
EntitiesFound    int     `json:"entities_found,omitempty"`     // Key items/concepts extracted
```

**Cost:** ~15 bytes per tool event (only populated for understanding tasks).

**Usage:**
```sql
-- Agent efficiency on large documents?
SELECT agent,
       AVG(coverage_achieved) as avg_coverage,
       AVG(entities_found) as avg_entities,
       AVG(duration_ms) as avg_time
FROM tool_events
WHERE task_type = 'document_understanding' AND target_size > 50000
GROUP BY agent;
```

---

### A.5 Summary: 12 New Optional Fields

| Struct | New Fields | Purpose | Storage Cost |
|--------|------------|---------|--------------|
| TaskClassifier | 3 patterns | Identify understanding vs code gen | 0 bytes |
| RoutingDecision | 4 fields | Understanding quality outcomes | ~20 bytes |
| AgentCollaboration | 5 fields | Swarm coordination & border agreement | ~30 bytes |
| ToolEvent | 3 fields | Understanding coverage/extraction | ~15 bytes |

**Total additional cost:** ~65 bytes per event (only for understanding tasks, all fields `omitempty`)

---

### A.6 Key Queries Enabled

**Routing for Understanding Tasks:**
```sql
SELECT selected_tier, selected_agent,
       AVG(understanding_completeness) as completeness,
       AVG(synthesis_coherence) as coherence,
       AVG(outcome_cost) as cost
FROM routing_decisions
WHERE task_type IN ('document_understanding', 'codebase_understanding')
GROUP BY selected_tier, selected_agent
ORDER BY completeness DESC, cost ASC;
```

**Optimal Swarm Configuration:**
```sql
SELECT
    COUNT(DISTINCT child_agent) as swarm_size,
    AVG(overlap_with_previous) as avg_overlap,
    AVG(agreement_with_adjacent) as avg_border_agreement,
    AVG(information_loss) as avg_info_loss
FROM agent_collaborations
WHERE is_swarm_member = true
GROUP BY root_task_id
HAVING COUNT(*) > 3;
```

**Best Team for Document Synthesis:**
```sql
SELECT parent_agent as orchestrator,
       child_agent as worker,
       AVG(child_success::int) as success_rate,
       AVG(agreement_with_adjacent) as border_quality
FROM agent_collaborations ac
JOIN routing_decisions rd ON ac.root_task_id = rd.decision_id
WHERE rd.task_type = 'document_understanding'
  AND ac.is_swarm_member = true
GROUP BY parent_agent, child_agent
HAVING COUNT(*) > 5
ORDER BY border_quality DESC;
```

---

### A.7 No New Tickets Required

These 12 fields integrate into the existing ticket revisions:

| Ticket | Additional Work |
|--------|-----------------|
| GOgent-087-REVISED | Add 3 understanding task types to classifier |
| GOgent-087b | Add 4 understanding quality fields to RoutingDecision |
| GOgent-088b | Add 5 swarm fields to AgentCollaboration |
| GOgent-087-REVISED | Add 3 context fields to ToolEvent |

**Estimated additional time:** +1h total across existing tickets.

---

*End of Addendum A*

---

*This GAP document should be archived to `.claude/gap_logger/` after review.*
