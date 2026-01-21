# Einstein GAP Document: User Intent Capture Architecture

**Generated**: 2026-01-21
**Escalated By**: Orchestrator during 029b completion
**Severity**: HIGH - Foundation for behavioral learning system
**Ticket Context**: GOgent-029a defined schema, capture mechanism undefined

---

## 1. Problem Statement

The GOgent learning system has **read infrastructure** for user intents but **no write mechanism**:

- `UserIntent` struct exists (`pkg/session/handoff_artifacts.go`)
- `QueryUserIntents()` API exists (`pkg/session/query.go`)
- `gogent-archive user-intents` CLI exists (`cmd/gogent-archive/main.go`)
- **Nothing writes to `user-intents.jsonl`**

### The Deeper Question

Beyond "how do we capture AskUserQuestion responses," the user asks:

> "What would be considered useful metadata or keywords to extract such that we can learn from a user's behavior?"

This elevates the problem from plumbing (hook implementation) to **behavioral intelligence architecture**:

1. What user behaviors are worth capturing?
2. What metadata enables pattern recognition across sessions?
3. How do we structure intents for agentic consumption?
4. What keywords/categories enable useful aggregation?

---

## 2. Current State Analysis

### Existing UserIntent Schema (from 029a)

```go
type UserIntent struct {
    Timestamp   int64  `json:"timestamp"`              // When captured
    Question    string `json:"question"`               // What was asked
    Response    string `json:"response"`               // User's answer
    Confidence  string `json:"confidence"`             // "explicit", "inferred", "default"
    Context     string `json:"context,omitempty"`      // Why this was asked
    Source      string `json:"source"`                 // "ask_user", "hook_prompt", "manual"
    ActionTaken string `json:"action_taken,omitempty"` // What we did with the response
}
```

### Valid Sources (where intents can originate)

```go
var ValidIntentSources = map[string]bool{
    "ask_user":    true, // AskUserQuestion tool
    "hook_prompt": true, // Hook-injected prompt
    "manual":      true, // Manually recorded
}
```

### Capture Points Identified

| Source | Trigger | Data Available |
|--------|---------|----------------|
| `ask_user` | `PostToolUse` for `AskUserQuestion` | Question, response, options offered |
| `hook_prompt` | Hook `additionalContext` prompts | Injected guidance, user reaction |
| `manual` | Explicit user statement | Free-form preference expression |

### Related Ticket Architecture

```
029 (SharpEdge Schema)  ─────────────────────────────────────────────┐
029a (UserIntent Schema) ────────────────────────────────────────────┤
029b (CLI for both) ─────────────────────────────────────────────────┤
                                                                      │
037 (SharpEdge Capture) ──── PostToolUse hook for Edit/Bash errors   │
037b (Code Context)     ──── Extract surrounding code lines          ├── Capture Layer
037c (Tool Input Log)   ──── Log what was attempted                  │
??? (UserIntent Capture) ─── THIS GAP                                │
                                                                      │
029c-f (Extended artifacts + aggregation) ───────────────────────────┘
```

---

## 3. Behavioral Learning Dimensions

### What Behaviors Matter?

| Behavior Type | Example | Learning Value |
|---------------|---------|----------------|
| **Routing preferences** | "Use sonnet for this" | Default tier selection |
| **Tool preferences** | "Don't use sed, use Edit" | Tool selection hints |
| **Style preferences** | "More concise responses" | Output formatting |
| **Workflow preferences** | "Always run tests after edit" | Task sequencing |
| **Domain knowledge** | "We use pytest not unittest" | Project context |
| **Correction patterns** | "No, I meant X not Y" | Disambiguation signals |
| **Approval patterns** | "Yes, proceed" vs detailed review | Trust calibration |
| **Rejection patterns** | "Stop", "Cancel", "Wrong approach" | Anti-patterns |

### Metadata for Pattern Recognition

**Temporal metadata:**
- Session ID (cluster behaviors by session)
- Timestamp (time-of-day patterns)
- Turn number (early vs late session behavior)
- Time since last interaction (engagement rhythm)

**Contextual metadata:**
- Current file/directory being worked on
- Current task (from TodoWrite if active)
- Recent tool sequence (what led to this question)
- Error state (was this triggered by failure?)

**Semantic metadata:**
- Intent category (routing, style, domain, workflow, correction)
- Keywords extracted from response
- Sentiment signal (positive/negative/neutral)
- Specificity level (vague vs precise)

**Outcome metadata:**
- Was the intent honored?
- Did it lead to success or failure?
- Was it referenced again later?

---

## 4. Capture Architecture Options

### Option A: Minimal Capture (Just AskUserQuestion)

```
PostToolUse(AskUserQuestion) → Extract Q&A → Write UserIntent
```

**Pros:** Simple, focused, low noise
**Cons:** Misses implicit preferences, corrections, hook interactions

### Option B: Multi-Source Capture

```
PostToolUse(AskUserQuestion) ─┬─→ UserIntent (source: ask_user)
PreToolUse(any) + user msg   ─┼─→ UserIntent (source: hook_prompt)
Explicit detection patterns  ─┴─→ UserIntent (source: manual)
```

**Pros:** Comprehensive behavioral data
**Cons:** Higher complexity, potential noise

### Option C: Enriched Capture with Classification

```
Capture Event → Classify Intent → Extract Keywords → Enrich Metadata → Write
```

**Pros:** Structured for learning, queryable by category
**Cons:** Classification logic complexity, potential misclassification

---

## 5. Schema Extension Considerations

### Current Schema Gaps

The existing `UserIntent` struct may need extension for behavioral learning:

```go
// Potential extensions (for Einstein to evaluate)
type UserIntent struct {
    // Existing fields...

    // Behavioral classification
    Category    string   `json:"category,omitempty"`    // routing, style, domain, workflow, correction
    Keywords    []string `json:"keywords,omitempty"`    // Extracted terms for aggregation
    Sentiment   string   `json:"sentiment,omitempty"`   // positive, negative, neutral

    // Contextual metadata
    SessionID   string   `json:"session_id,omitempty"`  // For session clustering
    TurnNumber  int      `json:"turn_number,omitempty"` // Position in conversation
    CurrentFile string   `json:"current_file,omitempty"`// Active file context
    CurrentTask string   `json:"current_task,omitempty"`// From TodoWrite if active

    // Outcome tracking
    Honored     *bool    `json:"honored,omitempty"`     // Was intent respected?
    OutcomeNote string   `json:"outcome_note,omitempty"`// What happened
}
```

### Backward Compatibility

Any extension must use `omitempty` for backward compatibility with existing schema version.

---

## 6. Integration Points

### Hook System

```toml
# Potential hook configuration
[hooks.PostToolUse]
matcher = "AskUserQuestion"
command = "gogent-capture-intent"
```

### Existing Capture Patterns

**From 037 (Sharp Edge Capture):**
```
PostToolUse(Edit/Bash) → Detect failure → Extract context → Write SharpEdge
```

User intent capture should follow parallel pattern:
```
PostToolUse(AskUserQuestion) → Extract Q&A → Classify → Write UserIntent
```

### Query Integration

The `QueryUserIntents()` function already supports filters. New filters may be needed:

```go
type UserIntentFilters struct {
    // Existing
    Source     *string
    Confidence *string
    HasAction  bool
    Since      *int64
    Limit      int

    // Potential additions
    Category   *string   // Filter by intent category
    Keywords   []string  // Filter by keyword presence
    SessionID  *string   // Filter by session
    Honored    *bool     // Filter by outcome
}
```

---

## 7. Primary Questions for Einstein

### Architecture Questions

1. **What capture scope is appropriate?**
   - Minimal (AskUserQuestion only)?
   - Multi-source (ask_user + hook_prompt + manual detection)?
   - What's the noise/signal tradeoff?

2. **What metadata enables useful behavioral learning?**
   - Which fields justify schema extension?
   - What's extractable vs requiring LLM classification?
   - How do we avoid over-engineering?

3. **How should intents be classified?**
   - Fixed category taxonomy vs free-form?
   - Keyword extraction approach?
   - Should classification happen at capture or query time?

### Implementation Questions

4. **Where does this fit in the ticket sequence?**
   - New ticket GOgent-037d?
   - Extension of 037c (Tool Input Log)?
   - Separate series for behavioral learning?

5. **What's the MVP vs full vision?**
   - Phase 1: Basic AskUserQuestion capture
   - Phase 2: Classification + metadata enrichment
   - Phase 3: Multi-source capture
   - Phase 4: Outcome tracking

6. **How does this integrate with 029f weekly aggregation?**
   - Intent patterns in weekly summary?
   - Preference drift detection?
   - Behavioral trend reporting?

---

## 8. Constraints

1. **Schema backward compatibility**: Extensions must use `omitempty`
2. **Hook performance**: Capture must not add latency to tool execution
3. **Storage efficiency**: Cannot capture everything; must be selective
4. **Privacy consideration**: User responses may contain sensitive content
5. **Agentic queryability**: Must support efficient filtered retrieval
6. **Human reviewability**: Weekly summaries must be meaningful

---

## 9. Anti-Scope

**DO NOT address:**
- Implementation details of capture hook code
- CLI UX for viewing intents (already done in 029b)
- Performance optimization specifics
- Privacy/security policy decisions

**FOCUS ON:**
- Behavioral learning architecture
- Metadata schema design
- Classification taxonomy
- Capture scope decision
- Ticket structure and sequencing

---

## 10. Success Criteria

Einstein analysis should provide:

1. **Recommended capture scope** (minimal/multi-source/enriched)
2. **Schema extension specification** (which fields to add)
3. **Classification taxonomy** (intent categories and keywords)
4. **Metadata priority ranking** (must-have vs nice-to-have)
5. **Phased implementation plan** (MVP → full vision)
6. **Ticket specification** (GOgent-037d or alternative)
7. **Integration guidance** (how this fits with 029 series and 037 series)

---

## 11. Reference Materials

### Files to Review

- `pkg/session/handoff_artifacts.go` - UserIntent struct definition
- `pkg/session/query.go` - QueryUserIntents() implementation
- `migration_plan/tickets/session_archive/completed/029a.md` - Original UserIntent ticket
- `migration_plan/tickets/session_archive/029f.md` - Weekly aggregation spec

### Architectural Context

- Dual-write pattern (JSONL source + formatted output)
- Query API pattern (filters + pagination)
- Hook capture pattern (PostToolUse → detect → extract → write)

---

**GAP Document Ready**

Run `/einstein` to process this escalation.
