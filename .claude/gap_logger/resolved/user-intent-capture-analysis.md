# Einstein Analysis: User Intent Capture Architecture

**Resolved**: 2026-01-21
**GAP Document**: `.claude/tmp/einstein-gap-user-intent-capture.md`
**Escalated By**: Orchestrator during 029b completion
**Severity**: HIGH - Foundation for behavioral learning system

---

## Executive Summary

The GOgent system has a critical asymmetry: **it learns from failures (sharp edges) but not from successes (user guidance)**. The existing `UserIntent` schema is adequate for MVP but needs strategic extension for genuine behavioral learning. My recommendation is a **phased approach** starting with minimal capture, evolving toward enriched behavioral intelligence.

---

## 1. Recommended Capture Scope

**Verdict: Start Minimal, Evolve to Multi-Source**

### Phase 1: MVP (GOgent-037d)
Capture **only** `AskUserQuestion` tool responses. This is:
- High signal (explicit user decisions)
- Clean trigger (PostToolUse hook)
- Zero ambiguity (structured Q&A format)

### Phase 2: Enriched (GOgent-038 series)
Add classification and metadata after MVP proves value.

### Phase 3: Multi-Source (Future)
Hook prompts and implicit detection only after behavioral patterns emerge from Phase 1/2 data.

**Rationale:** Starting multi-source creates noise before you understand signal. AskUserQuestion responses are the purest behavioral data—start there.

---

## 2. Schema Extension Specification

### Must-Have Fields (Phase 1)

```go
type UserIntent struct {
    // Existing - keep all
    Timestamp   int64  `json:"timestamp"`
    Question    string `json:"question"`
    Response    string `json:"response"`
    Confidence  string `json:"confidence"`
    Context     string `json:"context,omitempty"`
    Source      string `json:"source"`
    ActionTaken string `json:"action_taken,omitempty"`

    // NEW - Phase 1 (cheap to capture)
    SessionID   string `json:"session_id,omitempty"`   // From hook context
    ToolContext string `json:"tool_context,omitempty"` // Previous tool if relevant
}
```

### Should-Have Fields (Phase 2)

```go
    // NEW - Phase 2 (requires classification)
    Category    string   `json:"category,omitempty"`    // See taxonomy below
    Keywords    []string `json:"keywords,omitempty"`    // Extracted at capture
```

### Nice-to-Have Fields (Phase 3+)

```go
    // NEW - Phase 3+ (requires outcome tracking)
    Honored     *bool    `json:"honored,omitempty"`     // Did we follow it?
    OutcomeNote string   `json:"outcome_note,omitempty"`// What happened
    TurnNumber  int      `json:"turn_number,omitempty"` // Position in session
```

**What NOT to add:**
- `CurrentFile` / `CurrentTask` — Too noisy, often irrelevant to the intent itself
- `Sentiment` — Requires LLM classification, low ROI
- `Specificity` — Subjective, hard to define consistently

---

## 3. Classification Taxonomy

### Intent Categories (Fixed, Not Free-Form)

| Category | Trigger Patterns | Example |
|----------|------------------|---------|
| `routing` | tier, model, agent, delegate, use sonnet/haiku/opus | "Use haiku for this" |
| `tooling` | tool, use X not Y, prefer, don't use | "Use Edit not sed" |
| `style` | concise, verbose, format, output, response style | "Shorter responses" |
| `workflow` | always, never, after X do Y, sequence, order | "Run tests after edit" |
| `domain` | we use, our project, this codebase, convention | "We use pytest" |
| `correction` | no, wrong, I meant, actually, not that | "No, the other file" |
| `approval` | yes, proceed, go ahead, looks good, confirmed | "Yes, do it" |
| `rejection` | stop, cancel, abort, wrong approach, undo | "Stop, wrong direction" |

**Classification approach:** Pattern matching at capture time, not LLM inference. Simple keyword detection is sufficient and deterministic.

### Keyword Extraction

Extract **nouns and tool names** from user response:
- File paths mentioned
- Tool names (Edit, Bash, Task)
- Model names (sonnet, haiku, opus)
- Test frameworks (pytest, jest, go test)
- Commands mentioned

**Implementation:** Regex + allowlist, not NLP. Keep it simple.

---

## 4. Metadata Priority Ranking

| Priority | Field | Rationale |
|----------|-------|-----------|
| **P0** | SessionID | Clusters behaviors, enables session-level analysis |
| **P0** | Category | Enables filtered queries ("show me all routing preferences") |
| **P1** | Keywords | Enables search across intents |
| **P1** | ToolContext | Links intent to triggering situation |
| **P2** | Honored | Outcome tracking for learning effectiveness |
| **P2** | TurnNumber | Early vs late session patterns |
| **P3** | Sentiment | Low value, high complexity |
| **P3** | CurrentFile | Often noise, rarely meaningful |

---

## 5. Phased Implementation Plan

### Phase 1: MVP Capture (GOgent-037d)
**Scope:** Basic AskUserQuestion capture with SessionID

**Deliverables:**
- PostToolUse hook for AskUserQuestion
- Extract question, response, options from tool result
- Add SessionID from hook context
- Write to user-intents.jsonl
- No classification yet

**Acceptance criteria:**
- `gogent-archive user-intents` shows captured data
- SessionID populated correctly
- No latency impact on tool execution

### Phase 2: Classification (GOgent-038)
**Scope:** Add category and keyword extraction

**Deliverables:**
- Category classifier (pattern matching)
- Keyword extractor (regex + allowlist)
- Schema extended with Category, Keywords
- QueryUserIntents extended with Category filter

**Acceptance criteria:**
- 90%+ of intents correctly categorized
- Keywords extracted without LLM calls
- Query by category works

### Phase 3: Weekly Integration (GOgent-038b)
**Scope:** User intent patterns in 029f weekly aggregation

**Deliverables:**
- Intent summary in weekly report
- Preference drift detection (compare week-over-week)
- Category distribution metrics

### Phase 4: Outcome Tracking (GOgent-038c)
**Scope:** Track whether intents were honored

**Deliverables:**
- Honored field populated (requires session-end analysis)
- OutcomeNote for notable cases
- Query by outcome

---

## 6. Ticket Specification: GOgent-037d

```yaml
ticket_id: GOgent-037d
title: "User Intent Capture Hook"
status: pending
dependencies: [GOgent-029a, GOgent-037]
estimated_hours: 2.0
phase: 2
priority: HIGH
```

### Description

Implement PostToolUse hook to capture user intents from AskUserQuestion tool responses. This completes the write side of the user intent system (029a provided read side).

### Acceptance Criteria

- [ ] PostToolUse hook triggers on AskUserQuestion tool completion
- [ ] Hook extracts: question, response, options (if multi-choice)
- [ ] Hook extracts SessionID from hook context
- [ ] UserIntent written to `.claude/memory/user-intents.jsonl`
- [ ] Source field set to "ask_user"
- [ ] Confidence field set based on response type (explicit for selection, inferred for free-form)
- [ ] Context field captures why the question was asked (from tool input)
- [ ] `gogent-archive user-intents` displays captured intents
- [ ] Hook execution < 50ms (no latency impact)
- [ ] Tests for hook extraction logic
- [ ] `make test-ecosystem` passes

### Implementation Notes

**Hook configuration:**
```toml
[hooks.PostToolUse]
matcher = "AskUserQuestion"
command = "gogent-capture-intent"
```

**Data extraction from AskUserQuestion result:**
```go
type AskUserQuestionResult struct {
    Questions []struct {
        Question string   `json:"question"`
        Options  []string `json:"options,omitempty"`
    } `json:"questions"`
    Answers map[string]string `json:"answers"`
}
```

**SessionID source:** Available in hook input JSON under `session_id` field.

---

## 7. Integration Guidance

### Relationship to 029 Series

```
029 (SharpEdge Schema) ──────────────────────────────────┐
029a (UserIntent Schema) ────────────────────────────────┤
029b (CLI for both) ─────────────────────────────────────┤ Schema + Query + CLI
029c-f (Extended artifacts) ─────────────────────────────┘

037 (SharpEdge Capture) ─────────────────────────────────┐
037b (Code Context) ─────────────────────────────────────┤
037c (Tool Input Log) ───────────────────────────────────┤ Capture Hooks
037d (UserIntent Capture) ← THIS ────────────────────────┘

038 (Intent Classification) ─────────────────────────────┐
038b (Weekly Intent Summary) ────────────────────────────┤ Behavioral Learning
038c (Outcome Tracking) ─────────────────────────────────┘
```

### Relationship to 037 Series

GOgent-037d fits naturally in the capture hook series:
- 037: Captures failures (sharp edges)
- 037b: Enriches with code context
- 037c: Logs attempted changes
- **037d: Captures successes (user guidance)**

This creates symmetry: the system learns from both failures AND user direction.

### Weekly Aggregation (029f)

After 037d ships, 029f should include:
```markdown
## User Intents This Week

**Total Captured:** 23 intents across 8 sessions

**By Category:**
- routing: 5 (21%)
- domain: 7 (30%)
- workflow: 4 (17%)
- approval: 7 (30%)

**Notable Preferences:**
- "Use pytest not unittest" (3 sessions)
- "Run tests after edit" (2 sessions)
- "Prefer Edit over sed" (2 sessions)
```

---

## 8. Key Architectural Decisions

### Decision 1: Capture at Hook Level, Not Agent Level

**Chosen:** Hook-based capture (PostToolUse)
**Alternative:** Agent-based capture (each agent writes intents)
**Rationale:** Centralized, consistent, no agent modification needed

### Decision 2: Classification at Capture Time

**Chosen:** Classify during capture (pattern matching)
**Alternative:** Classify at query time (LLM-based)
**Rationale:** Deterministic, fast, no LLM cost per query

### Decision 3: Fixed Category Taxonomy

**Chosen:** 8 fixed categories
**Alternative:** Free-form tags
**Rationale:** Enables aggregation, prevents tag sprawl, queryable

### Decision 4: Defer Outcome Tracking

**Chosen:** Phase 3+ implementation
**Alternative:** Track outcomes immediately
**Rationale:** Requires session-end analysis, complex, low MVP value

---

## 9. Success Metrics

After GOgent-037d ships, measure:

| Metric | Target | Measurement |
|--------|--------|-------------|
| Capture rate | >95% of AskUserQuestion | Compare tool calls to intents |
| Latency impact | <50ms | Hook execution time |
| Query utility | Users use CLI | Usage of `gogent-archive user-intents` |
| Data quality | Parseable JSON | Validation on read |

After GOgent-038 ships, measure:
| Metric | Target | Measurement |
|--------|--------|-------------|
| Classification accuracy | >90% | Manual review sample |
| Keyword relevance | >80% useful | Manual review sample |
| Weekly summary value | User reads it | Engagement tracking |

---

## 10. Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Hook latency | User experience | Async write, benchmark |
| Over-capture noise | Storage, query noise | Start minimal, add sources later |
| Misclassification | Wrong aggregation | Simple patterns, manual review |
| Privacy concerns | Sensitive data logged | Document clearly, user control |

---

## Resolution Status

**Analysis Complete** - 2026-01-21

**Next Actions:**
1. Create ticket GOgent-037d in tickets-index.json
2. Create ticket spec file at `migration_plan/tickets/037d.md`
3. Optionally create 038 series tickets for future phases
4. Archive original GAP document
