# Braintrust Analysis: SCOPE-v3.md Review (2026-03-31)

> **Status:** complete
> **Total team cost:** Einstein $2.62 + Staff-Architect $1.25 + Beethoven $0.80 = **$4.67**

## Executive Summary

SCOPE-v3.md is a fundamentally sound implementation blueprint that requires targeted corrections before implementation begins. The .claude/ to .gogent/ migration (2026-03-31, 40 days after the spec was written) is the single largest invalidation event, creating a vault path naming collision and stale path assumptions. The most critical finding — shared by both analyses — is that the agents-index.json entry is missing 8+ required v2.6.0 schema fields, which would block all agent-delegated implementation. The hook integration architecture is underspecified (standalone Go binary vs library import). Overall, ~85% of the spec remains valid; the 15% requiring updates clusters around paths, schema completeness, and hook wiring. All issues are fixable in an estimated 3-4 hours of spec revision before implementation begins.

## Convergence Points

### agents-index.json entry is critically incomplete

**Confidence:** high

- **Einstein:** Entry missing parallelization_template, cli_flags, inputs, outputs, must_delegate, effortLevel, path, description, spawned_by. Phase 1 estimate of 30-60 min is 2-3x too low.
- **Staff-Architect:** C-1 (Critical): Missing 8+ required fields. Agent unreachable via MCP spawn_agent. Effort to fix: 30 minutes by copying from existing Tier 2 agent.
- **Synthesis:** Both agree this is the single highest-priority fix. The entry must be expanded to full v2.6.0 schema before any implementation begins. Staff-Architect's approach (copy from go-pro and customize) is the most efficient path.

### Vault path .gogent-vault/.gogent/ creates naming confusion

**Confidence:** high

- **Einstein:** Semantic collision: .gogent/ now means 'runtime I/O' at project level. Reusing inside .gogent-vault/ creates confusion. Recommends renaming to .graphdb/ or moving to .gogent/graphdb/.
- **Staff-Architect:** M-2 (Major): Cognitive overhead and potential glob/gitignore conflicts. Recommends renaming to .gogent-vault/.data/ or .gogent-vault/.store/, or moving to .gogent/memory/graph.db.
- **Synthesis:** Strong agreement on the problem. Resolution on naming differs (see divergence). The nested .gogent/ directory must be renamed regardless of chosen approach.

### Hook integration architecture is underspecified

**Confidence:** high

- **Einstein:** gogent-load-context extension is architecturally sound, but integration point should reference .gogent/memory/ not .claude/memory/. Latency measurements likely still valid.
- **Staff-Architect:** M-1 (Major): Two valid approaches exist (extend cmd/ binary to import internal/hooks, or create new hook binary) but spec doesn't choose one. Phase 4 blocked without this decision.
- **Synthesis:** Both agree the hook integration concept is sound but the spec needs to specify the concrete wiring. Staff-Architect's analysis is more actionable here — the spec must choose between extending gogent-load-context or creating a new binary.

### Proposed internal/ package structure is viable

**Confidence:** high

- **Einstein:** No naming conflicts with existing lifecycle/, teamconfig/, tui/. Go package system handles this cleanly. Aspiration (category 2) — valid.
- **Staff-Architect:** A-5 verified: Glob confirmed no existing directories at proposed paths. Minor concern about memory/ vs graphstore/ responsibility overlap (m-4).
- **Synthesis:** Both confirm the package structure works. The responsibility split between internal/memory/ and internal/graphstore/ is a minor design smell worth noting but not blocking.

### Spec is fundamentally sound — issues are completeness and drift, not architecture

**Confidence:** high

- **Einstein:** The spec's architectural decisions (interface-first, phased rollout, pluggable strategies) are solid. Problems are path staleness and schema drift, not design flaws.
- **Staff-Architect:** APPROVE_WITH_CONDITIONS. Commends interface-first design, pragmatic complexity management, and sharp edges documentation. Issues are fixable before implementation.
- **Synthesis:** Both analyses converge on a positive overall assessment. The spec needs a v3.1 revision for path corrections and schema completeness, not a redesign.

### Hook latency budget needs re-verification

**Confidence:** high

- **Einstein:** 28ms baseline and 72ms available budget are plausible but measured pre-migration. SQLite cold-start benchmark should be added to Phase 0.
- **Staff-Architect:** A-4 unverified. Recommends re-profiling on post-migration codebase. Lists as high-priority recommendation with 30-minute effort.
- **Synthesis:** Both flag this as an unverified assumption. Add SQLite cold-start latency benchmark to Phase 0 validation tasks alongside existing nomic-embed-text and goldmark checks.

## Unified Recommendations

### Implementation Phases

**Phase 1:** Critical and Major fixes: Expand agents-index.json entry to v2.6.0 schema, specify hook integration architecture, rename vault DB path to .gogent/graphdb/, fix spawned_by/can_spawn bidirectional relationships

Decision points:
- Hook integration: extend cmd/gogent-load-context/ to import internal/hooks, or create new cmd/gogent-memory-loader/ binary?
- Confirm .gogent/graphdb/ as database location (vs .gogent-vault/.store/)

Success criteria:
- [ ] agents-index.json entry has all v2.6.0 required fields
- [ ] Hook integration path specified with exact import paths and settings.json registration
- [ ] Vault structure diagram updated with .gogent/graphdb/ path
- [ ] spawned_by includes router, orchestrator, impl-manager

**Phase 2:** Comprehensive path audit: grep SCOPE-v3 for all .claude/ references, categorize as config (keep) vs runtime I/O (change to .gogent/) vs vault (keep at .gogent-vault/), apply corrections

Decision points:
- Verify which agents-index.json .claude/ paths are declarative vs runtime-active

Success criteria:
- [ ] Zero stale .claude/ runtime I/O paths remain in spec
- [ ] JSONL migration section (Section 12) references .gogent/memory/ paths
- [ ] All path references verified against current codebase state

**Phase 3:** Minor fixes and additions: add go.mod dependency commands, add _test.go expectations to package structure, add Phase 4 rollback procedure, add SQLite cold-start benchmark to Phase 0

Decision points:
- Phase 4 rollback: feature flag in gogent-load-context or separate binary switching?

Success criteria:
- [ ] Phase 0 includes SQLite cold-start latency validation
- [ ] Phase 1/2 prerequisites include go get commands
- [ ] Section 19 includes _test.go file expectations
- [ ] Phase 4 has documented rollback procedure

**Phase 4:** Re-profile hook latency on post-migration codebase to validate 72ms budget assumption before greenlighting implementation

Decision points:
- If baseline has increased significantly: switch to async injection model or optimize existing hooks first?

Success criteria:
- [ ] Hook latency profiled on current codebase
- [ ] Available budget confirmed >= 50ms for memory retrieval
- [ ] If budget insufficient: async injection path documented as alternative

### Not Recommended

- **Begin implementation without addressing the agents-index.json schema gap (C-1)** — The agent would be unreachable via spawn_agent. Every spawn attempt would fail gogent-validate schema validation. This is a hard blocker, not a soft issue.
- **Keep .gogent-vault/.gogent/ naming despite the collision** — Post-migration, .gogent/ has established runtime I/O semantics. Reusing the name inside .gogent-vault/ creates confusion for developers, glob patterns, and gitignore rules. The cost of renaming is trivial; the cost of confusion is ongoing.
- **Skip hook latency re-verification and assume 72ms budget is still valid** — The measurement is 40 days old. The migration touched 23 files. If the budget has shrunk below 50ms, the entire synchronous retrieval model breaks and Phase 4 needs redesign. A 30-minute verification prevents potential Phase 4 failure.

## Open Questions

- [high] Should the hook integration use cmd/gogent-load-context/ extension (import internal/hooks) or a new binary cmd/gogent-memory-loader/?
- [high] What is the intended long-term relationship between .gogent-vault/ (Obsidian content) and .gogent/ (runtime I/O)?
- [medium] Should the Phase 4 rollback use a feature flag in gogent-load-context or a separate binary that can be swapped in settings.json?
- [medium] Has go-db-architect.md (the detailed agent spec) been updated for the .gogent/ migration?
