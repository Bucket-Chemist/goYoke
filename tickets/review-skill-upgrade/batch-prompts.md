# Review Skill Upgrade - Batch Implementation Prompts

**Plan File**: `/home/doktersmol/Documents/GOgent-Fortress/tickets/review-skill-upgrade/review-skill-upgrade-plan-v2.md`

**Usage**: Copy a batch prompt into a fresh Claude Code session. The agent will read the plan and implement exactly what's specified.

---

## Batch A: Phase 1 - Foundation

```
TASK: Implement review-skill-upgrade-plan-v2.md Phase 1

SPEC FILE: /home/doktersmol/Documents/GOgent-Fortress/tickets/review-skill-upgrade/review-skill-upgrade-plan-v2.md

PHASE: 1 (Foundation)
TASKS: 1.1 through 1.9

INSTRUCTIONS:
1. Read the spec file completely before starting
2. Implement each task EXACTLY as specified in the plan
3. Follow the code snippets verbatim - do not "improve" them
4. Run tests after implementing Go code (Task 1.1, 1.2)
5. Verify file paths exist before adding conventions_required (Tasks 1.4-1.6)

SUCCESS CRITERIA:
- [ ] pkg/config/paths.go has 3 new path helpers + getMLTelemetryPath
- [ ] pkg/config/paths_test.go has tests for new helpers
- [ ] cmd/gogent-sharp-edge/main.go includes all 7 new agents in list
- [ ] agents-index.json has sharp_edges_count for all agents listed in plan
- [ ] backend-reviewer, frontend-reviewer, standards-reviewer have conventions_required
- [ ] All 3 reviewer CLAUDE.md files have Sharp Edge Correlation section
- [ ] .claude/CLAUDE.md routing tables updated per Task 1.8
- [ ] impl-manager CLAUDE.md has Telemetry Relationship section (Task 1.9)
- [ ] All Go tests pass: go test ./pkg/config/... ./cmd/gogent-sharp-edge/...

DO NOT:
- Skip reading the full plan first
- Modify code beyond what the plan specifies
- Add "improvements" or refactoring not in the plan
- Skip writing tests for Go code
- Proceed if convention files don't exist (check first)
- Combine multiple tasks into a single commit (commit per logical unit)
```

---

## Batch B: Phase 2 - Core Telemetry

```
TASK: Implement review-skill-upgrade-plan-v2.md Phase 2

SPEC FILE: /home/doktersmol/Documents/GOgent-Fortress/tickets/review-skill-upgrade/review-skill-upgrade-plan-v2.md

PHASE: 2 (Core Telemetry)
TASKS: 2.0 through 2.3

DEPENDENCIES: Phase 1 must be complete (path helpers exist)

INSTRUCTIONS:
1. Read the spec file completely before starting
2. Verify Phase 1 is complete: check pkg/config/paths.go has the new helpers
3. Implement Task 2.0 FIRST (JSONL file locking) - other tasks depend on it
4. Use the EXACT code from the plan - the file locking logic is critical
5. Task 2.2 depends on Task 2.3 (registry validation)

SUCCESS CRITERIA:
- [ ] pkg/telemetry/jsonl_writer.go exists with AppendJSONL function
- [ ] pkg/telemetry/jsonl_writer_test.go exists with concurrent write test (100 goroutines)
- [ ] pkg/telemetry/review_finding.go exists with all structs and functions from plan
- [ ] pkg/telemetry/sharp_edge_hit.go exists with validation against registry
- [ ] pkg/telemetry/sharp_edge_registry.go exists with LoadSharpEdgeIDs, IsValidSharpEdgeID
- [ ] All tests pass: go test ./pkg/telemetry/...
- [ ] Concurrent write test verifies no JSONL corruption

DO NOT:
- Start before verifying Phase 1 completion
- Modify the file locking implementation (syscall.Flock pattern is intentional)
- Skip the concurrent write test - it's critical for data integrity
- Implement 2.2 before 2.3 (SharpEdgeHit needs registry validation)
- Use different import paths than specified in plan
```

---

## Batch C: Phase 3 - CLI Tools

```
TASK: Implement review-skill-upgrade-plan-v2.md Phase 3

SPEC FILE: /home/doktersmol/Documents/GOgent-Fortress/tickets/review-skill-upgrade/review-skill-upgrade-plan-v2.md

PHASE: 3 (CLI Tools and Orchestrator Integration)
TASKS: 3.1 through 3.3

DEPENDENCIES: Phase 2 must be complete (telemetry package exists)

INSTRUCTIONS:
1. Read the spec file completely before starting
2. Verify Phase 2 is complete: check pkg/telemetry/*.go files exist
3. Create CLI binaries exactly as specified
4. Update Makefile with telemetry-tools target
5. Task 3.3 modifies review-orchestrator CLAUDE.md - use exact format from plan

SUCCESS CRITERIA:
- [ ] cmd/gogent-log-review/main.go exists and builds
- [ ] cmd/gogent-update-review-outcome/main.go exists and builds
- [ ] Makefile has telemetry-tools target building both binaries
- [ ] bin/gogent-log-review binary works: echo '{"session_id":"test","findings":[]}' | ./bin/gogent-log-review
- [ ] bin/gogent-update-review-outcome binary works with --help
- [ ] .claude/agents/review-orchestrator/CLAUDE.md has Telemetry Requirements section
- [ ] go build ./cmd/... succeeds

DO NOT:
- Start before verifying Phase 2 completion
- Modify the JSON input/output schemas
- Skip the Makefile updates
- Change the CLI flag names or formats
- Add dependencies not in the plan
```

---

## Batch D: Phase 4 - Skill Integration

```
TASK: Implement review-skill-upgrade-plan-v2.md Phase 4

SPEC FILE: /home/doktersmol/Documents/GOgent-Fortress/tickets/review-skill-upgrade/review-skill-upgrade-plan-v2.md

PHASE: 4 (Skill Integration)
TASKS: 4.1 and 4.2

DEPENDENCIES: Phase 3 must be complete (CLI tools exist and build)

INSTRUCTIONS:
1. Read the spec file completely before starting
2. Verify Phase 3 is complete: check bin/gogent-log-review exists
3. Update SKILL.md files with exact bash snippets from plan
4. Preserve existing content - add new sections, don't replace
5. Test that skills still load correctly after modification

SUCCESS CRITERIA:
- [ ] .claude/skills/review/SKILL.md has Phase 4 telemetry integration
- [ ] .claude/skills/review/SKILL.md has ML Telemetry documentation section
- [ ] .claude/skills/review/SKILL.md references GOGENT_ENABLE_TELEMETRY env var
- [ ] .claude/skills/ticket/SKILL.md Phase 7.6 stores finding IDs
- [ ] .claude/skills/ticket/SKILL.md Phase 8 logs review outcomes
- [ ] Both skills have feature flag support (GOGENT_ENABLE_TELEMETRY=0 disables)

DO NOT:
- Start before verifying Phase 3 completion
- Remove existing SKILL.md content
- Change the telemetry file paths
- Skip the feature flag implementation (rollback capability)
- Hardcode paths - use variables as shown in plan
```

---

## Batch E: Phase 5 + 6 - Extended Features and Testing

```
TASK: Implement review-skill-upgrade-plan-v2.md Phases 5 and 6

SPEC FILE: /home/doktersmol/Documents/GOgent-Fortress/tickets/review-skill-upgrade/review-skill-upgrade-plan-v2.md

PHASES: 5 (Extended Features) and 6 (Testing)
TASKS: 5.1 and 6.1

DEPENDENCIES: Phase 2 must be complete (telemetry structs exist)

INSTRUCTIONS:
1. Read the spec file completely before starting
2. Add subcommands to existing gogent-ml-export binary
3. Add ReadReviewFindings and CalculateReviewStats functions
4. Create all test files with the setupTestDir helper pattern
5. Run full test suite after implementation

SUCCESS CRITERIA:
- [ ] cmd/gogent-ml-export/main.go has review-findings, review-stats, sharp-edge-hits subcommands
- [ ] pkg/telemetry/review_finding.go has ReadReviewFindings and CalculateReviewStats
- [ ] pkg/telemetry/sharp_edge_hit.go has ReadSharpEdgeHits
- [ ] pkg/telemetry/jsonl_writer_test.go exists with all 3 test cases from plan
- [ ] pkg/telemetry/review_finding_test.go exists with all test cases from plan
- [ ] pkg/telemetry/sharp_edge_hit_test.go exists
- [ ] pkg/telemetry/sharp_edge_registry_test.go exists with IsValidSharpEdgeID test
- [ ] All tests pass: go test ./pkg/telemetry/... ./cmd/gogent-ml-export/...
- [ ] Concurrent write test (100 goroutines) passes without JSONL corruption

DO NOT:
- Skip any of the test files
- Use different test isolation patterns than setupTestDir
- Skip the concurrent write stress test
- Modify test assertions to make failing tests pass
- Leave any test file empty or with placeholder content
```

---

## Batch F: Phase 7 - Optional Features

```
TASK: Implement review-skill-upgrade-plan-v2.md Phase 7

SPEC FILE: /home/doktersmol/Documents/GOgent-Fortress/tickets/review-skill-upgrade/review-skill-upgrade-plan-v2.md

PHASE: 7 (Nice-to-Have Features)
TASKS: 7.1 and 7.2

DEPENDENCIES: Phases 1-6 should be complete

INSTRUCTIONS:
1. Read the spec file completely before starting
2. These are OPTIONAL features - confirm with user before implementing
3. Extend existing structs - don't create duplicates
4. Cross-session detection requires reading multiple JSONL files

SUCCESS CRITERIA:
- [ ] ReviewOutcomeUpdate has UserFeedback and FeedbackComment fields
- [ ] cmd/gogent-review-feedback/main.go exists and builds
- [ ] pkg/telemetry/sharp_edge_candidate.go exists with detection algorithm
- [ ] cmd/gogent-aggregate/main.go extended with candidate detection
- [ ] Tests exist for new functionality
- [ ] All tests pass

DO NOT:
- Implement without user confirmation (these are optional)
- Create duplicate struct definitions
- Skip similarity matching logic in candidate detection
- Leave candidate threshold hardcoded without configuration option
```

---

## Verification Batch: Final Checks

```
TASK: Verify review-skill-upgrade-plan-v2.md implementation

SPEC FILE: /home/doktersmol/Documents/GOgent-Fortress/tickets/review-skill-upgrade/review-skill-upgrade-plan-v2.md

INSTRUCTIONS:
1. Read the Verification Checklist section at the end of the plan
2. Run through each checkbox systematically
3. Report any failures with specific details

VERIFICATION COMMANDS:
# Build all binaries
make all

# Run all tests
go test ./pkg/... ./cmd/...

# Verify binaries exist
ls -la bin/gogent-log-review bin/gogent-update-review-outcome

# Test telemetry logging
echo '{"session_id":"verify","review_scope":"test","files_reviewed":1,"findings":[{"severity":"info","reviewer":"test","category":"test","file":"test.go","line":1,"message":"test"}]}' | ./bin/gogent-log-review

# Verify JSONL output
cat ~/.local/share/gogent-fortress/review-findings.jsonl | tail -1 | jq .

# Test outcome update
./bin/gogent-update-review-outcome --finding-id=test --resolution=fixed

# Verify agents-index.json has sharp_edges_count
jq '.[].sharp_edges_count' .claude/agents/agents-index.json | head -20

# Verify conventions exist
ls .claude/conventions/typescript.md .claude/conventions/react.md

SUCCESS CRITERIA: All items in plan's Verification Checklist pass

REPORT FORMAT:
For each failing check, provide:
- Which check failed
- What was expected
- What was found
- Suggested fix
```

---

## Notes for Implementers

1. **Always read the plan first** - The plan is authoritative. These prompts are guardrails.

2. **Verify dependencies** - Each batch depends on previous batches. Don't skip ahead.

3. **Commit frequently** - One commit per task or logical unit. Makes rollback easier.

4. **Tests are mandatory** - No Go code without corresponding tests. The plan specifies exact test cases.

5. **Feature flag** - GOGENT_ENABLE_TELEMETRY=0 must disable all telemetry. This is the rollback mechanism.

6. **Don't improve** - Implement exactly what's in the plan. Improvements belong in a follow-up PR.
