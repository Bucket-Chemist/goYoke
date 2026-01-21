# RLM Engine Implementation Plan

**Status**: Ready for Implementation
**Version**: 1.0
**Created**: 2026-01-17

---

## Quick Start

This directory contains the complete implementation plan for the **RLM (Recursive Language Model) Engine**, a system for processing massive contexts (6M-20M tokens) using iterative decomposition strategies.

### What is RLM?

RLM treats massive contexts as external environment variables that can be queried iteratively through a REPL (Read-Eval-Print Loop) interface, achieving 12-58 point improvements over direct long-context methods while remaining cost-competitive ($1.50-$2.50 vs $2.75+ for direct approaches).

**Key Innovation**: Instead of stuffing 10M tokens into an LLM's context window, RLM allows the LLM to query the context like a database:

```python
# Metaprompt executed in Starlark REPL
context = "...10M tokens..."  # External variable
chunk = context[0:100000]
result = llm_query("Find security issues in: " + chunk)
print(result)
FINAL(aggregated_results)
```

---

## Implementation Overview

### Technology Stack

- **Language**: Go 1.21+
- **REPL**: Starlark-go (Python-like embedded interpreter)
- **Integration**: External agent pattern (follows `gemini-slave` architecture)
- **Invocation**: Bash piping (`cat file | rlm-engine analyze "query"`)
- **Deployment**: Single binary, zero runtime dependencies

### Architecture Decision

**Why Starlark?**
- ✅ Python-like syntax (RLM paper's metaprompts translate directly)
- ✅ Hermetic execution (sandboxed, no file/network access)
- ✅ Zero runtime dependencies (pure Go, ~500KB binary addition)
- ✅ Production-hardened (used in Bazel, Google-maintained)

**Why NOT alternatives?**
- ❌ Yaegi (Go interpreter): Go syntax incompatible with Python metaprompts
- ❌ Embedded CPython: Violates Go convention #1 (zero runtime dependencies)

---

## Document Structure

| Document | Purpose | Start Here? |
|----------|---------|-------------|
| **README.md** (this file) | Overview and navigation | ✅ Yes |
| **RLM_IMPLEMENTATION_PLAN.md** | Complete technical architecture | ✅ Yes (for architects) |
| **tickets/00-overview.md** | Standards applying to all tickets | ✅ Yes (for developers) |
| **tickets/TICKET-TEMPLATE.md** | Required ticket structure | For reference |
| **docs/rlm_metaprompt_recommendation.md** | Original analysis and evaluation | For context |

---

## Implementation Phases

### Phase 1: Core Engine (Weeks 1-2)

**Goal**: Functional RLM engine with Starlark REPL

**Key Tickets** (15 total):
- RLM-001: Go module setup
- RLM-002 to RLM-005: Starlark embedding and built-ins
- RLM-006 to RLM-008: REPL loop implementation
- RLM-009 to RLM-011: Sub-LLM API integration (Claude Haiku)
- RLM-012 to RLM-015: Protocol handlers (analyze, compress, metaprompt)

**Deliverable**: `rlm-engine` binary that executes RLM metaprompts

---

### Phase 2: Routing Integration (Week 3)

**Goal**: Integrate RLM into Claude Code routing system

**Key Tickets** (5 total):
- RLM-016: Agent definition (`~/.claude/agents/rlm-engine/agent.yaml`)
- RLM-017: Update `routing-schema.json` with external tier entry
- RLM-018: Register routing triggers
- RLM-019: Bash integration testing
- RLM-020: Orchestrator delegation workflow

**Deliverable**: RLM automatically invoked for large-context queries

---

### Phase 3: Metaprompts & Testing (Week 4)

**Goal**: Production-quality metaprompts with corpus validation

**Key Tickets** (10 total):
- RLM-021 to RLM-023: Claude-optimized metaprompts
- RLM-024 to RLM-026: Gemini-optimized metaprompts
- RLM-027 to RLM-029: Corpus testing (6M, 8M, 11M tokens)
- RLM-030: Cost benchmarking

**Deliverable**: Proven metaprompt templates, validated performance

---

### Phase 4: Production Readiness (Week 5)

**Goal**: Production-ready, documented, installable

**Key Tickets** (5 total):
- RLM-031: Error handling standards
- RLM-032: Logging strategy
- RLM-033: Installation script
- RLM-034: Documentation (user guide, developer guide)
- RLM-035: Integration testing (all platforms)

**Deliverable**: Production-ready RLM system

---

## For Implementers

### Getting Started

1. **Read RLM_IMPLEMENTATION_PLAN.md** - Understand overall architecture
2. **Read tickets/00-overview.md** - Learn standards and conventions
3. **Start with RLM-001** - Initialize Go module
4. **Follow ticket sequence** - Dependencies are declared explicitly

### Standards to Follow

- **Error Format**: `[rlm-engine:component] What. Why. How.`
- **Logging**: Structured slog to `~/.gogent/rlm-engine.log`
- **File Paths**: XDG-compliant (no hardcoded `/tmp`)
- **Timeouts**: All long operations respect `context.Context`
- **Testing**: ≥80% coverage, race detector clean
- **Cross-compilation**: darwin/amd64, darwin/arm64, windows/amd64, linux/amd64

### Development Workflow

```bash
# 1. Clone repo and create RLM branch
git checkout -b feature/rlm-engine

# 2. Initialize module (RLM-001)
cd /path/to/gogent-fortress
go mod init github.com/yourusername/gogent-fortress

# 3. Add Starlark dependency (RLM-002)
go get go.starlark.net/starlark

# 4. Implement tickets sequentially
# ... follow ticket instructions ...

# 5. Test after each ticket
go test ./pkg/rlm -v
go test ./pkg/rlm -race

# 6. Cross-compile (before RLM-035)
make build-all

# 7. Integration test (RLM-035)
bash test/integration/test-rlm-routing.sh
```

---

## For Reviewers

### What to Verify

#### Phase 1 Completion Criteria

- [ ] `rlm-engine` binary compiles successfully
- [ ] Binary size <10MB
- [ ] Cross-compiles for all 4 platforms
- [ ] Starlark REPL executes paper's example metaprompts
- [ ] Built-ins (`context`, `llm_query`, `FINAL`) work correctly
- [ ] Sub-LLM API calls succeed with retries
- [ ] All 3 protocols functional (analyze, compress, metaprompt)
- [ ] Unit tests pass with ≥80% coverage

#### Phase 2 Completion Criteria

- [ ] `routing-schema.json` contains RLM entry
- [ ] Orchestrator routes large-context queries to RLM
- [ ] Bash invocation works: `cat file | rlm-engine analyze "query"`
- [ ] Output appears in `~/.claude/tmp/rlm-output.md`
- [ ] End-to-end test passes
- [ ] No regressions in existing routing

#### Phase 3 Completion Criteria

- [ ] 3+ metaprompt templates per protocol
- [ ] Tested with 6M, 8M, 11M token corpuses
- [ ] RLM accuracy meets or exceeds paper's claims
- [ ] Cost per invocation ≤ $2.50
- [ ] Corpus tests document actual vs expected performance
- [ ] Cost benchmarks show ROI

#### Phase 4 Completion Criteria

- [ ] Error messages follow standard format
- [ ] Structured logging to `~/.gogent/rlm-engine.log`
- [ ] Installation script works on clean system
- [ ] Documentation complete (4 guides)
- [ ] Integration tests pass on Linux, macOS, Windows
- [ ] System is user-facing ready

---

## Integration with Claude Code

### How RLM Fits In

```
User: "Analyze this 10M token codebase for security issues"
    ↓
Claude Orchestrator: Detects trigger "10M token"
    ↓
Routing System: Routes to external tier → rlm-engine
    ↓
Bash: cat codebase.tar.gz | rlm-engine analyze "security issues"
    ↓
RLM Engine:
  - Loads context into Starlark
  - Executes metaprompt in REPL loop
  - Makes 30-50 sub-queries to Claude Haiku
  - Returns markdown report
    ↓
Orchestrator: Reads output, synthesizes for user
```

### Routing Triggers

**Auto-invoke RLM when user query contains**:
- "multi-million token"
- "recursive analysis"
- "10M+ tokens"
- "extremely large context"
- "analyze massive codebase"

### Cost Model

| Component | Model | Avg Tokens | Cost |
|-----------|-------|------------|------|
| Root LLM | Claude Opus 4.5 | 15-20 iterations × 2K | $1.35-$1.80 |
| Sub-LLM | Claude Haiku 4.5 | 30-50 queries × 10K | $0.15-$0.25 |
| **Total** | - | - | **$1.50-$2.05** |

**Cost Ceiling**: $10.00 (hard stop)

---

## Research Context

### RLM Paper Summary

**Paper**: ["Recursively using Large Language Models to Compress Massive Contexts"](https://arxiv.org/abs/2501.09768)

**Key Results**:
- 12-58 point improvements over direct long-context methods
- Cost-competitive: $0.99 vs $1.50-$2.75
- Scales to 6M-11M tokens (10x beyond current context windows)

**Emergent Strategies** (Section 3.3):
1. **Peek-Filter-Dive**: Survey → Identify candidates → Deep analysis
2. **Prior-Based Probing**: Use earlier results to guide next queries
3. **Verification Loop**: Cross-check findings across multiple sections

**Our Implementation**:
- Uses Starlark instead of Python (hermetic, embeddable)
- Targets Claude API (paper used GPT-5)
- External agent pattern (follows existing `gemini-slave` architecture)
- Cost target: $1.50-$2.50 (accounting for Claude pricing)

---

## Risks & Mitigations

### Risk 1: Starlark Incompatibility

**Risk**: RLM metaprompts assume full Python; Starlark is a subset
**Mitigation**: Test paper's metaprompts in Phase 3, document incompatibilities, provide Starlark-ified versions

### Risk 2: Cost Overruns

**Risk**: RLM invocations exceed $10 budget
**Mitigation**: Hard $10 ceiling in code, warn at $5/$7.50, log all costs, provide estimates before execution

### Risk 3: Integration Complexity

**Risk**: Routing integration breaks existing workflows
**Mitigation**: External tier pattern is isolated, no Task() modifications, opt-in triggers, comprehensive testing

### Risk 4: Metaprompt Quality

**Risk**: Generated metaprompts produce poor results
**Mitigation**: Start with paper's proven strategies, test with real corpus, provide multiple templates, allow customization

---

## Timeline

| Phase | Duration | Tickets | Key Milestone |
|-------|----------|---------|---------------|
| **Phase 1** | 2 weeks | RLM-001 to RLM-015 | Core engine functional |
| **Phase 2** | 1 week | RLM-016 to RLM-020 | Routing integration |
| **Phase 3** | 1 week | RLM-021 to RLM-030 | Metaprompts validated |
| **Phase 4** | 3 days | RLM-031 to RLM-035 | Production-ready |
| **Total** | **~5 weeks** | **35 tickets** | - |

---

## Success Criteria

**Phase 1**: RLM engine executes metaprompts, makes sub-LLM calls, respects cost ceiling

**Phase 2**: Orchestrator automatically routes large-context queries to RLM

**Phase 3**: RLM accuracy ≥ paper's results, cost ≤ $2.50/invocation

**Phase 4**: Production-ready, documented, cross-platform, user-facing

---

## Related Documentation

- **Main Plan**: [RLM_IMPLEMENTATION_PLAN.md](RLM_IMPLEMENTATION_PLAN.md)
- **Standards**: [tickets/00-overview.md](tickets/00-overview.md)
- **Ticket Template**: [tickets/TICKET-TEMPLATE.md](tickets/TICKET-TEMPLATE.md)
- **Original Analysis**: [docs/rlm_metaprompt_recommendation.md](../docs/rlm_metaprompt_recommendation.md)
- **RLM Paper**: https://arxiv.org/abs/2501.09768
- **Starlark Spec**: https://github.com/google/starlark-go/blob/master/doc/spec.md

---

## Questions?

**Architecture Questions**: See RLM_IMPLEMENTATION_PLAN.md § Open Questions

**Implementation Questions**: See tickets/00-overview.md § Cross-References

**Starlark Questions**: See tickets/00-overview.md § Starlark Compatibility Guidelines

---

## Change Log

| Version | Date | Changes | Author |
|---------|------|---------|--------|
| 1.0 | 2026-01-17 | Initial implementation plan | Claude Sonnet 4.5 |

---

**Ready to begin?** Start with [RLM_IMPLEMENTATION_PLAN.md](RLM_IMPLEMENTATION_PLAN.md) for the complete technical architecture, then proceed to tickets/00-overview.md for implementation standards.
