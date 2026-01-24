# Coverage Gap Analysis Report
**Generated**: 2026-01-24
**Project**: GOgent-Fortress
**Overall Coverage**: 76.5%
**Target**: >90%

---

## Executive Summary

Current project-wide coverage is **76.5%**, requiring a **13.5 percentage point increase** to reach the 90% target.

### Coverage by Module

| Module | Coverage | Status | Priority |
|--------|----------|--------|----------|
| pkg/routing | 95.1% | ✅ EXCELLENT | Maintain |
| pkg/telemetry | 95.3% | ✅ EXCELLENT | Maintain |
| cmd/gogent-aggregate | 91.5% | ✅ GOOD | Maintain |
| pkg/session | 89.0% | 🟡 CLOSE | Low priority |
| pkg/memory | 89.8% | 🟡 CLOSE | Low priority |
| cmd/gogent-archive | 76.9% | 🟠 NEEDS WORK | Medium |
| pkg/config | 74.7% | 🟠 NEEDS WORK | Medium |
| test/simulation/harness | 65.0% | 🔴 CRITICAL | High |
| cmd/gogent-validate | 56.9% | 🔴 CRITICAL | High |
| cmd/gogent-capture-intent | 52.6% | 🔴 CRITICAL | High |
| cmd/gogent-sharp-edge | 18.8% | 🔴 CRITICAL | Critical |
| test/simulation/harness/cmd/harness | 14.5% | 🔴 CRITICAL | Critical |
| cmd/gogent-load-context | 0.0% | 🔴 CRITICAL | Critical |

---

## Critical Gaps (Immediate Action Required)

### 1. cmd/gogent-load-context (0.0%)
**Impact**: HIGH - Core SessionStart hook functionality
**Estimated Effort**: 4-6 hours

**Missing Tests**:
- main() entry point
- outputError() function
- All load-context logic paths

**Recommendation**: Create integration tests covering:
- Valid SessionStart event processing
- Invalid JSON handling
- File read errors
- Context assembly logic

---

### 2. cmd/gogent-sharp-edge (18.8%)
**Impact**: HIGH - Sharp edge detection critical for system learning
**Estimated Effort**: 6-8 hours

**Missing Tests** (from coverage report):
- getAgentDirectories() - 0.0%
- main() - 0.0%
- outputWarning() - 0.0%

**Covered**:
- writePendingLearning() - 69.2%
- getProjectDir() - 66.7%
- outputJSON() - 100.0%

**Recommendation**: Add tests for:
- Agent directory discovery with multiple paths
- Warning output formatting
- main() integration with different event inputs

---

### 3. test/simulation/harness/cmd/harness (14.5%)
**Impact**: MEDIUM - Testing infrastructure (meta-coverage issue)
**Estimated Effort**: 8-12 hours

**Missing Tests**:
- main() - 0.0%
- runFuzz() - 0.0%
- runMixed() - 0.0%
- runReplay() - 0.0%
- runSessionReplay() - 0.0%
- runBehavioral() - 0.0%
- runChaos() - 0.0%
- All report writing functions - 0.0%

**Recommendation**: 
- **Option A**: Add integration tests for harness CLI
- **Option B**: Exclude from coverage (test-only infrastructure)
- **Recommended**: Option B with documentation

---

### 4. cmd/gogent-capture-intent (52.6%)
**Impact**: MEDIUM - Intent tracking for SessionStart
**Estimated Effort**: 3-4 hours

**Missing Tests**:
- main() - 0.0%
- run() - 0.0%
- appendIntent() - 57.9% (partial)

**Covered**:
- extractIntent() - 90.5% ✓

**Recommendation**: Add integration tests for:
- Intent extraction from user prompts
- JSONL appending with file locking
- Error handling for file write failures

---

### 5. cmd/gogent-validate (56.9%)
**Impact**: HIGH - Routing validation is critical
**Estimated Effort**: 2-3 hours

**Missing Tests**:
- main() - 0.0%

**Covered**:
- parseEvent() - 100.0% ✓
- outputResult() - 100.0% ✓
- outputError() - 100.0% ✓

**Recommendation**: Add integration tests for main() with:
- Valid PreToolUse events
- Invalid JSON inputs
- STDIN timeout scenarios

---

## Medium Priority Gaps

### 6. test/simulation/harness (65.0%)
**Impact**: LOW - Test infrastructure
**Estimated Effort**: 12-16 hours

**Major Gaps**:
- chaos_runner.go: NewChaosRunner, Run, runAgent, simulateFailure - all 0.0%
- session_replayer.go: Most functions 0.0%
- generator.go: GenerateSessionEvent, RandomSessionMetrics - 0.0%

**Recommendation**:
- **Option A**: Comprehensive harness testing (expensive)
- **Option B**: Mark as test-infrastructure, exclude from coverage
- **Recommended**: Option B - Document why tests are excluded

---

### 7. cmd/gogent-archive (76.9%)
**Impact**: MEDIUM - Session archival
**Estimated Effort**: 4-5 hours

**Major Gaps**:
- generateWeeklySummary() - 0.0%
- parseSinceFilter() - 60.0%
- filterBetween() - 54.5%
- showSession() - 60.0%
- getProjectDir() - 66.7%

**Recommendation**: Add tests for:
- Weekly summary generation
- Date filter parsing (since/between)
- Session display formatting

---

### 8. pkg/config (74.7%)
**Impact**: MEDIUM - Configuration utilities
**Estimated Effort**: 2-3 hours

**Major Gaps**:
- GetGOgentDir() - 56.5%
- IncrementToolCount() - 73.1%
- GetToolCount() - 76.9%

**Recommendation**: Add tests for:
- GetGOgentDir() with missing HOME env
- Tool counter edge cases (concurrent increments)
- File permission errors

---

## Low Priority Gaps

### 9. pkg/session (89.0%)
**Impact**: LOW - Already close to target
**Estimated Effort**: 3-4 hours

**Specific Gaps**:
- archive.go: moveFile() - 30.0%
- archive.go: copyFile() - 66.7%
- handoff_markdown.go: FormatWeeklyIntentSummary() - 64.0%
- intent_aggregation.go: AggregateWeeklyIntents() - 72.7%
- intent_outcomes.go: analyzeRoutingIntent() - 57.1%

**Recommendation**: Focus on moveFile/copyFile error paths:
- Permission denied scenarios
- Cross-device moves
- Partial copy failures

---

### 10. pkg/memory (89.8%)
**Impact**: LOW - Already close to target
**Estimated Effort**: 2-3 hours

**Specific Gaps**:
- failure_tracking.go: rewriteFailures() - 47.6%
- getStoragePath() - 66.7%

**Recommendation**: Add tests for:
- Concurrent failure log rewrites
- Storage path expansion edge cases

---

## Roadmap to 90% Coverage

### Phase 1: Critical Fixes (Est. 20-25 hours)
**Target**: Bring all <60% modules above 60%
**Estimated Coverage Gain**: +8-10 percentage points

1. ✅ cmd/gogent-load-context: 0% → 80% (+80pp local)
2. ✅ cmd/gogent-sharp-edge: 18.8% → 75% (+56.2pp local)
3. ✅ cmd/gogent-capture-intent: 52.6% → 80% (+27.4pp local)
4. ✅ cmd/gogent-validate: 56.9% → 85% (+28.1pp local)

### Phase 2: Medium Priority (Est. 6-8 hours)
**Target**: Bring 70-80% modules above 85%
**Estimated Coverage Gain**: +3-4 percentage points

1. ✅ cmd/gogent-archive: 76.9% → 85% (+8.1pp local)
2. ✅ pkg/config: 74.7% → 85% (+10.3pp local)

### Phase 3: Polish (Est. 5-6 hours)
**Target**: Bring 85-90% modules above 90%
**Estimated Coverage Gain**: +1-2 percentage points

1. ✅ pkg/session: 89.0% → 92% (+3pp local)
2. ✅ pkg/memory: 89.8% → 92% (+2.2pp local)

### Total Estimated Effort: 31-39 hours

**Decision Point**: test/simulation/harness modules
- **Option A**: Include (add 12-16 hours)
- **Option B**: Exclude as test-infrastructure (saves time, document decision)
- **Recommended**: Option B

---

## Implementation Priorities

### Immediate (This Week)
1. cmd/gogent-load-context - Integration tests
2. cmd/gogent-sharp-edge - main() and agent directory tests
3. cmd/gogent-validate - main() integration tests

### Next Week
4. cmd/gogent-capture-intent - Intent extraction tests
5. cmd/gogent-archive - Weekly summary + filter tests
6. pkg/config - GetGOgentDir edge cases

### Following Week
7. pkg/session - moveFile/copyFile error paths
8. pkg/memory - rewriteFailures concurrent scenarios

---

## Exclusion Recommendations

Consider excluding from coverage requirements:
1. **test/simulation/harness/** - Testing infrastructure (meta-tests)
2. **test/simulation/harness/cmd/harness/** - Test CLI tooling
3. **All main() functions** - Entry points (hard to test meaningfully)

**Rationale**: Test infrastructure testing creates circular dependencies and limited value.

**Alternative**: Document coverage exclusions in `.coverignore` or equivalent.

---

## Quick Wins (High ROI)

These provide maximum coverage gain for minimum effort:

| Item | Current | Target | Gain | Effort | ROI |
|------|---------|--------|------|--------|-----|
| cmd/gogent-load-context | 0.0% | 80% | +80pp | 4h | ⭐⭐⭐⭐⭐ |
| cmd/gogent-validate main() | 0.0% | 100% | +43pp | 2h | ⭐⭐⭐⭐⭐ |
| cmd/gogent-sharp-edge main() | 0.0% | 100% | +81pp | 3h | ⭐⭐⭐⭐ |
| pkg/session moveFile() | 30% | 90% | +60pp | 1h | ⭐⭐⭐⭐ |

**Start Here**: Focus on main() functions and integration tests.

---

## Notes

- Coverage data generated from: `go test -coverprofile=coverage-full.out ./...`
- Detailed function coverage: `coverage-full-report.txt`
- Project uses Go test coverage (not external tools)
- Current audit system (GOgent-063) generates per-ticket coverage reports

---

## Recommended Next Steps

1. **Review this report** with team
2. **Decide on test/simulation/harness** inclusion/exclusion
3. **Create tickets** for Phase 1 critical fixes
4. **Set coverage target** in CI/CD (suggest 85% initially, then 90%)
5. **Add coverage gate** to prevent regression

---

**Report Generated By**: Coverage analysis script
**Data Source**: `go test -coverprofile=coverage-full.out ./...`
**Last Updated**: 2026-01-24
