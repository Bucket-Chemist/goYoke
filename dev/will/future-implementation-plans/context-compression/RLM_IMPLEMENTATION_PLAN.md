# RLM Engine Implementation Plan

**Project**: Recursive Language Model (RLM) Engine for Large-Context Processing
**Version**: 1.0
**Date**: 2026-01-17
**Status**: Draft - Awaiting Review

---

## Executive Summary

This document outlines the complete implementation plan for an RLM (Recursive Language Model) engine as described in the research paper ["Recursively using Large Language Models to Compress Massive Contexts"](https://arxiv.org/abs/2501.09768). The RLM paradigm treats massive contexts (6M-11M tokens) as external environment variables that can be queried iteratively, achieving 12-58 point improvements over direct long-context methods while remaining cost-competitive ($0.99 vs $1.50-$2.75).

### Core Innovation

Traditional approach: Stuff entire context into LLM's window (limited to 2M tokens, degrades at scale)
RLM approach: Context as external data structure, iteratively queried via REPL environment

### Implementation Strategy

**Technology Choice**: Starlark-go (Python-like language embedded in Go)
**Integration Pattern**: External agent (follows existing `gemini-slave` architecture)
**Invocation**: Bash piping (zero Task() overhead)
**Cross-compilation**: Single binary for darwin/amd64, darwin/arm64, windows/amd64, linux/amd64

**Why Starlark over alternatives**:
- ✅ Python-like syntax (RLM metaprompts translate directly)
- ✅ Hermetic execution (no file system, network, syscall access)
- ✅ Zero runtime dependencies (pure Go, ~500KB binary addition)
- ✅ Google-maintained (used in Bazel, production-hardened)
- ❌ NOT Yaegi (Go interpreter): Limited to Go syntax, breaks RLM paper's Python examples
- ❌ NOT embedded CPython: Violates Go convention #1 (zero runtime dependencies)

---

## Problem Statement

### Current Limitations

1. **Context Window Ceiling**: Claude Opus 4.5 (200K tokens), Gemini 1.5 Pro (2M tokens)
2. **Performance Degradation**: Accuracy drops significantly beyond 500K tokens (RULER benchmark)
3. **Cost Inefficiency**: Processing 6M+ tokens in single call is prohibitively expensive
4. **Pattern Mismatch**: Humans decompose large information tasks; LLMs process monolithically

### RLM Solution

**Key Insight**: Treat context as a database to be queried, not a monolith to be read

**Metaprompt Strategy**:
```python
# Starlark-compatible Python-like syntax
context = "...6M token codebase..."  # External variable
chunk = context[0:100000]
result = llm_query("Find security issues in: " + chunk)
print(result)
FINAL(result)
```

**Emergent Patterns** (from paper Section 3.3):
1. **Peek-Filter-Dive**: Survey → Identify candidates → Deep analysis
2. **Prior-Based Probing**: Use earlier results to guide next queries
3. **Verification Loop**: Cross-check findings across multiple sections

---

## Architecture Overview

### System Diagram

```
┌────────────────────────────────────────────────────────────┐
│                    Claude Code (Sonnet)                    │
│                      Orchestrator                          │
└─────────────────────────┬──────────────────────────────────┘
                          │
                          │ Detects trigger:
                          │ "analyze 10M token codebase"
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│              Routing System (schema.json)                    │
│  Trigger: ["multi-million token", "recursive analysis"]     │
│  Action: Route to external tier → rlm-engine                │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│           Bash Invocation (follows gemini-slave)            │
│  cat large_context.txt | rlm-engine analyze "query"         │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌──────────────────────────────────────────────────────────────┐
│                    RLM Engine (Go + Starlark)                │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  1. Load context into Starlark `context` variable     │  │
│  │  2. Load metaprompt template (Claude/Gemini-specific) │  │
│  │  3. Execute REPL loop (max 20 iterations):            │  │
│  │     - Call Claude API with metaprompt                 │  │
│  │     - Extract ```repl code blocks                     │  │
│  │     - Execute Starlark code                           │  │
│  │     - llm_query() makes sub-LLM calls (Haiku)         │  │
│  │     - Accumulate print() output                       │  │
│  │     - Check for FINAL() call                          │  │
│  │  4. Return markdown report                            │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                              │
│  Cost Tracking: $0.50-$10.00 per invocation                 │
│  Output: ~/.claude/tmp/rlm-output.md                        │
└──────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│              Orchestrator Synthesis                          │
│  Reads output, presents findings to user                    │
└─────────────────────────────────────────────────────────────┘
```

### Integration Points

| Component | File Path | Purpose |
|-----------|-----------|---------|
| **RLM Binary** | `cmd/rlm-engine/main.go` | Entry point, protocol routing, Bash integration |
| **Core Engine** | `pkg/rlm/engine.go` | REPL loop, Starlark interpreter management |
| **Built-ins** | `pkg/rlm/builtins.go` | Starlark functions: `context`, `llm_query`, `FINAL` |
| **Metaprompts** | `pkg/rlm/metaprompts.go` | Template engine (Claude/Gemini-optimized) |
| **Cost Tracking** | `pkg/rlm/cost.go` | Token counting, budget limits, reporting |
| **Agent Definition** | `~/.claude/agents/rlm-engine/agent.yaml` | Routing metadata, triggers, protocols |
| **Routing Schema** | `~/.claude/routing-schema.json` | External tier registration, trigger patterns |
| **Metaprompt Templates** | `~/.claude/agents/rlm-engine/templates/*.star` | Claude/Gemini-specific metaprompts |

---

## Implementation Phases

### Phase 1: Core Engine (Weeks 1-2)

**Tickets**: RLM-001 to RLM-015
**Duration**: 2 weeks
**Goal**: Functional RLM engine with Starlark REPL

#### Key Deliverables

1. **Go Module Setup** (RLM-001)
   - Initialize `cmd/rlm-engine` and `pkg/rlm`
   - Add Starlark-go dependency: `go.starlark.net/starlark`
   - Cross-compilation targets in Makefile

2. **Starlark Embedding** (RLM-002 to RLM-005)
   - Interpreter initialization
   - RLM built-in functions: `context`, `llm_query`, `print`, `FINAL`, `FINAL_VAR`
   - Starlark code execution with error handling
   - Variable accumulation across iterations

3. **REPL Loop** (RLM-006 to RLM-008)
   - Iteration limit (20 max)
   - Code block extraction from LLM responses
   - Print output accumulation
   - FINAL detection and termination

4. **Sub-LLM Integration** (RLM-009 to RLM-011)
   - Claude API client (Haiku 4.5 for sub-queries)
   - Request/response handling
   - Error handling and retries (3 max)
   - Response caching (optional)

5. **Protocol Handlers** (RLM-012 to RLM-015)
   - `analyze`: General recursive analysis
   - `compress`: Context compression for long conversations
   - `metaprompt`: Metaprompt generation from corpus
   - Protocol router in `main.go`

---

### Phase 2: Routing Integration (Week 3)

**Tickets**: RLM-016 to RLM-020
**Duration**: 1 week
**Goal**: RLM engine integrated into Claude Code routing system

#### Key Deliverables

1. **Agent Definition** (RLM-016)
   - Create `~/.claude/agents/rlm-engine/agent.yaml`
   - Define triggers: ["multi-million token", "recursive analysis", "10M+ tokens"]
   - Specify protocols: ["analyze", "compress", "metaprompt"]
   - Document cost ceiling ($10.00 per invocation)

2. **Routing Schema Update** (RLM-017)
   - Add RLM to `routing-schema.json` external tier
   - Register agent-to-subagent mapping (external/Bash)
   - Define escalation rules (any → external for large context)

3. **Trigger Registration** (RLM-018)
   - Update `agents-index.json` with rlm-engine entry
   - Add routing patterns to orchestrator knowledge
   - Document invocation examples in agent.yaml

4. **Bash Integration Testing** (RLM-019)
   - Test stdin piping: `cat file | rlm-engine analyze "query"`
   - Test file input: `rlm-engine analyze --context-file path "query"`
   - Test output redirection: `> ~/.claude/tmp/rlm-output.md`
   - Verify error codes and stderr handling

5. **Orchestrator Delegation** (RLM-020)
   - Document orchestrator → rlm-engine workflow
   - Create example delegation prompts
   - Test end-to-end: user query → orchestrator → RLM → synthesis

---

### Phase 3: Metaprompts & Testing (Week 4)

**Tickets**: RLM-021 to RLM-030
**Duration**: 1 week
**Goal**: Production-quality metaprompts and comprehensive testing

#### Key Deliverables

1. **Claude-Optimized Metaprompts** (RLM-021 to RLM-023)
   - Chunk-and-aggregate strategy (RLM-021)
   - Peek-filter-dive strategy (RLM-022)
   - Verification loop strategy (RLM-023)
   - Based on paper Section 3.2 guidance

2. **Gemini-Optimized Metaprompts** (RLM-024 to RLM-026)
   - Similar strategies adapted for Gemini's strengths
   - Larger chunk sizes (Gemini handles longer contexts)
   - Less frequent sub-queries

3. **Corpus Testing** (RLM-027 to RLM-029)
   - Generate test corpus: 6M, 8M, 11M token contexts
   - Test case: Security vulnerability analysis
   - Test case: API pattern extraction
   - Test case: Architecture documentation generation
   - Compare RLM vs direct long-context approaches

4. **Cost Benchmarking** (RLM-030)
   - Measure actual costs per protocol
   - Compare to paper's reported costs ($0.99)
   - Document cost ceiling thresholds
   - Generate cost reports

---

### Phase 4: Production Readiness (Week 5)

**Tickets**: RLM-031 to RLM-035
**Duration**: 3 days
**Goal**: Production-ready, documented, installable

#### Key Deliverables

1. **Error Handling Standards** (RLM-031)
   - Standardized error format: `[rlm-engine] What. Why. How.`
   - Timeout handling (5min max per iteration)
   - Graceful degradation (fallback to last good result)
   - Retry logic for API failures

2. **Logging Strategy** (RLM-032)
   - Structured logs to `~/.gogent/rlm-engine.log`
   - Log levels: DEBUG, INFO, WARN, ERROR
   - Iteration tracking (step 1/20, cost so far)
   - Performance metrics (tokens/sec, API latency)

3. **Installation Script** (RLM-033)
   - `make install-rlm` target
   - Copy binary to `~/.local/bin/rlm-engine`
   - Install agent.yaml to `~/.claude/agents/rlm-engine/`
   - Install metaprompt templates
   - Verify routing-schema.json update

4. **Documentation** (RLM-034)
   - User guide: When to use RLM vs direct approaches
   - Developer guide: Adding new protocols
   - Metaprompt guide: Writing custom strategies
   - Troubleshooting guide: Common errors

5. **Integration Testing** (RLM-035)
   - End-to-end test: User query → RLM → output
   - Test all protocols with real corpus
   - Verify routing triggers work
   - Verify cost ceiling enforcement
   - Cross-platform testing (Linux, macOS, Windows)

---

## Technical Specifications

### Starlark Built-ins API

```python
# Available in all RLM metaprompts

# 1. context (string): The massive context provided by user
context: str  # 6M-11M tokens, immutable

# 2. llm_query(prompt: str) -> str
#    Makes a sub-LLM API call (Claude Haiku 4.5)
#    Max 3 retries on failure
#    Raises error if all retries fail
result = llm_query("What is X in this chunk: " + chunk)

# 3. print(value: Any) -> None
#    Accumulates output, shown to root LLM in next iteration
print("Found 12 security issues in module A")

# 4. FINAL(answer: str) -> NoReturn
#    Terminates REPL loop, returns answer to orchestrator
FINAL("The codebase has 45 security vulnerabilities...")

# 5. FINAL_VAR(variable: Any) -> NoReturn
#    Terminates with the value of a variable
FINAL_VAR(aggregated_results)

# Standard Python/Starlark operations available:
# - String slicing: context[0:1000]
# - String concatenation: "prompt: " + context
# - Lists: results = []
# - Loops: for i in range(10): ...
# - Functions: def analyze_chunk(c): ...
```

### Protocol Specifications

#### 1. `analyze` Protocol

**Purpose**: General recursive analysis of large contexts

**Usage**:
```bash
cat large_codebase.tar.gz | rlm-engine analyze "Find all SQL injection vulnerabilities"
```

**Metaprompt Strategy**: Chunk-and-aggregate (divide into 100K chunks, analyze each, synthesize)

**Output Format**: Markdown report with findings

**Cost**: $0.80-$2.50 per invocation (depends on context size)

---

#### 2. `compress` Protocol

**Purpose**: Context compression for long conversations

**Usage**:
```bash
cat conversation_history.txt | rlm-engine compress "Summarize key decisions"
```

**Metaprompt Strategy**: Peek-filter-dive (scan for important sections, deep-dive on those)

**Output Format**: Compressed markdown summary (<10% of original size)

**Cost**: $0.50-$1.20 per invocation

---

#### 3. `metaprompt` Protocol

**Purpose**: Generate metaprompts from corpus examples

**Usage**:
```bash
cat metaprompt_corpus.json | rlm-engine metaprompt "Create strategy for API extraction"
```

**Metaprompt Strategy**: Prior-based probing (analyze existing patterns, generate new)

**Output Format**: Starlark metaprompt template

**Cost**: $1.00-$3.00 per invocation

---

### Cost Model

| Component | Model | Cost per 1K tokens | Usage Pattern |
|-----------|-------|-------------------|---------------|
| **Root LLM** | Claude Opus 4.5 | $0.045 | 15-20 iterations × 2K tokens = $1.35-$1.80 |
| **Sub-LLM** | Claude Haiku 4.5 | $0.0005 | 30-50 queries × 10K tokens = $0.15-$0.25 |
| **Total** | - | - | **$1.50-$2.05 per invocation** |

**Cost Ceiling**: $10.00 per invocation (enforced in code)

**Comparison** (from paper Table 1):
- RLM (GPT-5 + GPT-5-mini): $0.99
- Direct GPT-5 (1M tokens): $1.50
- Gemini 1.5 Pro (2M tokens): $2.75

Our implementation is slightly higher due to Claude pricing but within acceptable range.

---

### Performance Targets

| Metric | Target | Rationale |
|--------|--------|-----------|
| **Max Iterations** | 20 | Prevents runaway loops, paper uses 15 avg |
| **Per-Iteration Timeout** | 5 minutes | Allows for API latency, prevents hangs |
| **Total Timeout** | 30 minutes | 20 iterations × 5min/iter + overhead |
| **Context Size** | 6M-20M tokens | Paper tested up to 11M, we extend slightly |
| **Output Size** | <500KB | Manageable for orchestrator to synthesize |
| **Cache Hit Rate** | >30% | Sub-queries often repeat, caching helps |

---

## Risks & Mitigations

### Risk 1: Starlark Incompatibility with Python

**Risk**: RLM metaprompts from paper assume full Python; Starlark is subset
**Impact**: Metaprompts fail to execute, require rewriting
**Probability**: MEDIUM (Starlark covers 80% of Python basics)
**Mitigation**:
- Test paper's metaprompts in Starlark during RLM-021
- Document incompatibilities in `COMPATIBILITY.md`
- Provide Starlark-ified versions of all metaprompts
- Keep incompatibilities minimal (no imports, no file I/O)

---

### Risk 2: Cost Overruns

**Risk**: RLM invocations exceed $10 budget, surprise user
**Impact**: Angry user, system distrust
**Probability**: LOW (cost tracking built-in)
**Mitigation**:
- Enforce $10 ceiling in code (hard stop)
- Warn at $5, $7.50 thresholds
- Log all costs to structured log
- Provide cost estimates before execution (RLM-030)

---

### Risk 3: Integration Complexity

**Risk**: Routing integration breaks existing workflows
**Impact**: Claude Code system instability
**Probability**: LOW (external agent pattern is isolated)
**Mitigation**:
- RLM uses external tier (no Task() tool modifications)
- No changes to existing agents (pure addition)
- Routing triggers are opt-in (explicit keywords)
- Comprehensive integration testing (RLM-035)

---

### Risk 4: Metaprompt Quality

**Risk**: Generated metaprompts produce poor results
**Impact**: RLM accuracy below paper's claims
**Probability**: MEDIUM (metaprompt engineering is hard)
**Mitigation**:
- Start with paper's proven strategies
- Test against real corpus (RLM-027 to RLM-029)
- Provide multiple metaprompt templates per protocol
- Allow users to customize metaprompts
- Document metaprompt debugging techniques (RLM-034)

---

## Success Criteria

### Phase 1 Success

- [ ] RLM engine compiles to single binary (<10MB)
- [ ] Cross-compiles for all 4 targets (darwin/amd64, darwin/arm64, windows/amd64, linux/amd64)
- [ ] Starlark REPL executes paper's example metaprompts
- [ ] Built-ins (`context`, `llm_query`, `FINAL`) work correctly
- [ ] Sub-LLM API calls succeed with proper error handling
- [ ] All 3 protocols (analyze, compress, metaprompt) are functional
- [ ] Unit tests pass with ≥80% coverage

---

### Phase 2 Success

- [ ] `routing-schema.json` contains RLM external tier entry
- [ ] Orchestrator routes large-context queries to RLM
- [ ] Bash invocation works: `cat file | rlm-engine analyze "query"`
- [ ] Output appears in `~/.claude/tmp/rlm-output.md`
- [ ] End-to-end test passes: user query → orchestrator → RLM → synthesis
- [ ] No regressions in existing routing system

---

### Phase 3 Success

- [ ] 3+ metaprompt templates per protocol
- [ ] Tested with 6M, 8M, 11M token corpuses
- [ ] RLM accuracy ≥ paper's reported results (12-58 point improvement)
- [ ] Cost per invocation ≤ $2.50 (vs paper's $0.99, accounting for Claude pricing)
- [ ] Corpus tests document actual performance vs expectations
- [ ] Cost benchmarks show ROI vs direct long-context approaches

---

### Phase 4 Success

- [ ] Error messages follow `[rlm-engine] What. Why. How.` format
- [ ] Structured logging to `~/.gogent/rlm-engine.log`
- [ ] Installation script works on clean system
- [ ] Documentation covers user guide, developer guide, troubleshooting
- [ ] Integration tests pass on Linux, macOS, Windows
- [ ] System is production-ready and user-facing

---

## Timeline

| Phase | Duration | Start | End | Key Milestones |
|-------|----------|-------|-----|----------------|
| **Phase 1** | 2 weeks | Week 1 | Week 2 | Core engine functional |
| **Phase 2** | 1 week | Week 3 | Week 3 | Routing integration complete |
| **Phase 3** | 1 week | Week 4 | Week 4 | Metaprompts tested with corpus |
| **Phase 4** | 3 days | Week 5 | Week 5 | Production-ready |
| **Total** | 4 weeks + 3 days | - | - | - |

---

## Dependencies

### External Libraries

```go
// go.mod
module github.com/yourusername/gogent-fortress

go 1.21

require (
    go.starlark.net/starlark v0.0.0-20240123000000-0123456789ab
    github.com/anthropics/anthropic-sdk-go v0.1.0  // Claude API client
)
```

### System Requirements

- Go 1.21+ (for development)
- Zero runtime requirements (single static binary)
- Supported platforms: Linux, macOS (Intel/ARM), Windows

### Claude Code System Requirements

- `routing-schema.json` v2.2.0+
- `agents-index.json` with external tier support
- Bash shell for invocation
- `~/.claude/` directory structure

---

## Open Questions

### Q1: Should we support Gemini as sub-LLM?

**Context**: Paper uses GPT-5-mini for sub-queries; we use Claude Haiku 4.5
**Options**:
- A: Claude Haiku only (simplicity)
- B: Configurable sub-LLM (Claude Haiku or Gemini Flash)
- C: Protocol-specific sub-LLM (Claude for reasoning, Gemini for large chunks)

**Recommendation**: Start with A (RLM-009), defer B to post-v1.0

---

### Q2: How to handle metaprompt versioning?

**Context**: Metaprompts will evolve; users may customize
**Options**:
- A: Ship templates, users modify in `~/.claude/agents/rlm-engine/templates/`
- B: Version metaprompts (v1, v2), allow selection via flag
- C: Git-tracked metaprompts, users fork and modify

**Recommendation**: Start with A (RLM-021), document customization in RLM-034

---

### Q3: Should RLM cache sub-query results?

**Context**: Paper mentions 30%+ queries are duplicates
**Options**:
- A: No caching (simplicity, correct but slower)
- B: In-memory cache per invocation (fast, correct, lost after exit)
- C: Persistent cache (fastest, but cache invalidation is hard)

**Recommendation**: Start with A (RLM-009), add B in RLM-011 if performance is issue

---

## Related Documents

- **Tickets**: See `tickets/` directory for individual implementation tickets
- **Standards**: See `tickets/00-overview.md` for cross-cutting standards
- **Template**: See `tickets/TICKET-TEMPLATE.md` for ticket structure
- **Index**: See `tickets/tickets-index.json` for metadata
- **Paper**: [Recursively using Large Language Models to Compress Massive Contexts](https://arxiv.org/abs/2501.09768)
- **Starlark Spec**: [Starlark Language Specification](https://github.com/google/starlark-go/blob/master/doc/spec.md)
- **Go Conventions**: `/home/doktersmol/.claude/conventions/go.md`

---

## Approval

**Created By**: Claude Sonnet 4.5
**Reviewed By**: Pending
**Approved By**: Pending
**Date**: 2026-01-17

---

## Change Log

| Version | Date | Changes | Author |
|---------|------|---------|--------|
| 1.0 | 2026-01-17 | Initial implementation plan | Claude Sonnet 4.5 |

---

**Next Steps**:
1. Review this plan with team
2. Adjust timeline/scope based on feedback
3. Begin Phase 1 implementation (RLM-001)
4. Set up weekly progress reviews
