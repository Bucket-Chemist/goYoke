# Review Skill Upgrade - Batch Implementation Prompts

**Source Plan**: `tickets/review-skill-upgrade/review-skill-upgrade-plan-v2.md`
**Generated**: 2026-02-02
**Total Batches**: 6 across 4 sessions

---

## Session 1: Foundation

### Batch A: Path Helpers + Agent List + JSON Updates

```
Implement Review Skill Upgrade - Batch A (Tasks 1.1-1.3)

SPEC FILE: /home/doktersmol/Documents/GOgent-Fortress/tickets/review-skill-upgrade/review-skill-upgrade-plan-v2.md

BATCH: A - Tasks 1.1, 1.2, 1.3

---

## Task 1.1: Add Path Helpers for Telemetry Files

**File**: `pkg/config/paths.go`

Add these functions after existing path helpers:

```go
// GetReviewFindingsPathWithProjectDir returns path for review findings log
func GetReviewFindingsPathWithProjectDir() string {
    return getMLTelemetryPath("review-findings.jsonl")
}

// GetReviewOutcomesPathWithProjectDir returns path for review outcome updates
func GetReviewOutcomesPathWithProjectDir() string {
    return getMLTelemetryPath("review-outcomes.jsonl")
}

// GetSharpEdgeHitsPathWithProjectDir returns path for sharp edge correlation log
func GetSharpEdgeHitsPathWithProjectDir() string {
    return getMLTelemetryPath("sharp-edge-hits.jsonl")
}
```

If `getMLTelemetryPath` helper doesn't exist, add it:

```go
func getMLTelemetryPath(filename string) string {
    if dir := os.Getenv("GOGENT_PROJECT_DIR"); dir != "" {
        return filepath.Join(dir, ".claude", "memory", filename)
    }
    dataHome := os.Getenv("XDG_DATA_HOME")
    if dataHome == "" {
        home := os.Getenv("HOME")
        dataHome = filepath.Join(home, ".local", "share")
    }
    return filepath.Join(dataHome, "gogent-fortress", filename)
}
```

Add tests to `pkg/config/paths_test.go`.

---

## Task 1.2: Update gogent-sharp-edge Agent List

**File**: `cmd/gogent-sharp-edge/main.go`
**Location**: `getAgentDirectories()` function

Add these agents to the agents slice:
- "typescript-pro"
- "react-pro"
- "backend-reviewer"
- "frontend-reviewer"
- "standards-reviewer"
- "review-orchestrator"
- "impl-manager"

---

## Task 1.3: Add sharp_edges_count to agents-index.json

**File**: `.claude/agents/agents-index.json`

Add `sharp_edges_count` field to these agent entries:

| Agent | Count | Source |
|-------|-------|--------|
| typescript-pro | (count from sharp-edges.yaml) | |
| react-pro | (count from sharp-edges.yaml) | |
| backend-reviewer | (count from sharp-edges.yaml) | |
| frontend-reviewer | (count from sharp-edges.yaml) | |
| standards-reviewer | (count from sharp-edges.yaml) | |
| review-orchestrator | 4 | Created in v2.5.0 |
| orchestrator | 12 | 7 existing + 5 new |
| planner | 4 | Created in v2.5.0 |
| gemini-slave | 4 | Created in v2.5.0 |
| memory-archivist | 3 | Created in v2.5.0 |
| haiku-scout | 2 | Created in v2.5.0 |
| impl-manager | 7 | Created in v2.5.0 |

First, count actual entries in each sharp-edges.yaml file to verify.

---

## TODO

- [x] Read existing paths.go to understand patterns
- [x] Add 3 new path helper functions
- [x] Add getMLTelemetryPath helper if missing
- [x] Add tests for new path helpers
- [x] Update getAgentDirectories() with 7 new agents
- [x] Count sharp edges in each agent's sharp-edges.yaml
- [x] Add sharp_edges_count to agents-index.json entries
- [x] Run `go build ./...`
- [x] Run `go test ./pkg/config/...`
- [x] Run `go test ./cmd/gogent-sharp-edge/...`

## DO NOT

- Do NOT modify any agent YAML files
- Do NOT modify routing-schema.json
- Do NOT modify ARCHITECTURE.md
- Do NOT create new files outside pkg/config and cmd/gogent-sharp-edge
- Do NOT change existing path helper function signatures

---

## VERIFICATION

```bash
go build ./...
go test ./pkg/config/...
go test ./cmd/gogent-sharp-edge/...
```

## COMMIT MESSAGE

```
feat(telemetry): Add review telemetry path helpers and agent list updates

- Add GetReviewFindingsPathWithProjectDir()
- Add GetReviewOutcomesPathWithProjectDir()
- Add GetSharpEdgeHitsPathWithProjectDir()
- Add 7 new agents to gogent-sharp-edge
- Add sharp_edges_count to agents-index.json entries

Tasks 1.1, 1.2, 1.3 from review-skill-upgrade-plan-v2.md
```
```

---

### Batch B: YAML Config + CLAUDE.md Updates

```
Implement Review Skill Upgrade - Batch B (Tasks 1.4-1.9)

SPEC FILE: /home/doktersmol/Documents/GOgent-Fortress/tickets/review-skill-upgrade/review-skill-upgrade-plan-v2.md

BATCH: B - Tasks 1.4, 1.5, 1.6, 1.7, 1.8, 1.9

---

## Task 1.4: Add conventions_required to backend-reviewer

**File**: `.claude/agents/backend-reviewer/agent.yaml`

Add after `tools:` section:
```yaml
conventions_required:
  - go.md
  - python.md
  - typescript.md
```

---

## Task 1.5: Add conventions_required to frontend-reviewer

**File**: `.claude/agents/frontend-reviewer/agent.yaml`

Add after `tools:` section:
```yaml
conventions_required:
  - react.md
  - typescript.md
```

---

## Task 1.6: Add conventions_required to standards-reviewer

**File**: `.claude/agents/standards-reviewer/agent.yaml`

Add after `tools:` section:
```yaml
conventions_required:
  - go.md
  - python.md
  - typescript.md
  - react.md
```

---

## Task 1.7: Add Sharp Edge Pattern Matching to Reviewers

**Files**:
- `.claude/agents/backend-reviewer/CLAUDE.md`
- `.claude/agents/frontend-reviewer/CLAUDE.md`
- `.claude/agents/standards-reviewer/CLAUDE.md`

Add this section to each CLAUDE.md:

```markdown
## Sharp Edge Correlation

When identifying issues, check if they match known sharp edge patterns from sharp-edges.yaml.

For each finding that matches a sharp edge:
1. Include `sharp_edge_id` in output (must be valid ID from registry)
2. Use the exact symptom description
3. Reference the documented solution

**Output format for correlated findings**:
```json
{
  "severity": "critical",
  "file": "path/to/file.go",
  "line": 45,
  "message": "Issue description",
  "sharp_edge_id": "sql-injection",
  "recommendation": "Use parameterized queries"
}
```
```

Also list the available sharp edge IDs from that agent's sharp-edges.yaml file.

---

## Task 1.8: Update CLAUDE.md Routing Tables

**File**: `.claude/CLAUDE.md`

1. Verify these entries exist in "Tier 1.5: Haiku + Thinking" table:
   - backend-reviewer, frontend-reviewer, standards-reviewer

2. Verify these entries exist in "Tier 2: Sonnet" table:
   - typescript-pro, react-pro, review-orchestrator

3. ADD this entry to "Tier 2: Sonnet" table if missing:
```markdown
| implement from specs, execute todos, implement plan | `impl-manager` | Plan |
```

4. Verify `/review` is in "Slash Commands" table

5. Update Schema version reference if it says v2.2.0 (should be v2.5.0)

---

## Task 1.9: Document impl-manager Telemetry Relationship

**File**: `.claude/agents/impl-manager/CLAUDE.md`

Add this section:

```markdown
## Telemetry Relationship

impl-manager produces `impl-violations.jsonl` during implementation, which is conceptually related to but distinct from `review-findings.jsonl`:

| Telemetry File | Written By | Phase | Purpose |
|----------------|------------|-------|---------|
| `impl-violations.jsonl` | impl-manager | During implementation | Real-time convention enforcement |
| `review-findings.jsonl` | review-orchestrator | Post-implementation | Code review feedback |

**Key Differences:**
- **impl-violations**: Caught DURING implementation, can block task completion
- **review-findings**: Caught AFTER implementation, advisory only
```

---

## TODO

- [x] Add conventions_required to backend-reviewer/agent.yaml
- [x] Add conventions_required to frontend-reviewer/agent.yaml
- [x] Add conventions_required to standards-reviewer/agent.yaml
- [x] Add Sharp Edge Correlation section to backend-reviewer/CLAUDE.md
- [x] Add Sharp Edge Correlation section to frontend-reviewer/CLAUDE.md
- [x] Add Sharp Edge Correlation section to standards-reviewer/CLAUDE.md
- [x] List available sharp_edge_ids in each CLAUDE.md
- [x] Verify/update .claude/CLAUDE.md routing tables
- [x] Add impl-manager to Tier 2 Sonnet table
- [x] Update schema version reference to v2.5.0
- [x] Add telemetry relationship section to impl-manager/CLAUDE.md

## DO NOT

- Do NOT modify Go code
- Do NOT modify routing-schema.json
- Do NOT modify ARCHITECTURE.md
- Do NOT change agent model/tier settings
- Do NOT remove existing content from CLAUDE.md files

---

## VERIFICATION

- All YAML files parse correctly (no syntax errors)
- All agent.yaml files have valid structure
- CLAUDE.md files render correctly in markdown

## COMMIT MESSAGE

```
feat(agents): Add conventions and sharp edge correlation to reviewers

- Add conventions_required to backend/frontend/standards reviewers
- Add Sharp Edge Correlation section to reviewer CLAUDE.md files
- Update .claude/CLAUDE.md routing tables with impl-manager
- Document impl-manager telemetry relationship

Tasks 1.4-1.9 from review-skill-upgrade-plan-v2.md
```
```

---

## Session 2: Core Telemetry

### Batch C: Telemetry Structs + Registry

```
Implement Review Skill Upgrade - Batch C (Tasks 2.0-2.3)

SPEC FILE: /home/doktersmol/Documents/GOgent-Fortress/tickets/review-skill-upgrade/review-skill-upgrade-plan-v2.md

BATCH: C - Tasks 2.0, 2.1, 2.2, 2.3

---

## Task 2.0: Implement JSONL Append with File Locking

**File**: `pkg/telemetry/jsonl_writer.go` (NEW)

Create new file with this implementation:

```go
package telemetry

import (
    "fmt"
    "os"
    "path/filepath"
    "syscall"
)

// AppendJSONL safely appends a line to a JSONL file with file locking
// to prevent corruption from concurrent writes.
func AppendJSONL(path string, data []byte) error {
    // Ensure parent directory exists
    dir := filepath.Dir(path)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return fmt.Errorf("mkdir: %w", err)
    }

    // Open file for append
    f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return fmt.Errorf("open: %w", err)
    }
    defer f.Close()

    // Acquire exclusive lock (blocks until available)
    if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
        return fmt.Errorf("flock: %w", err)
    }
    defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

    // Write data with newline
    if _, err := f.Write(append(data, '\n')); err != nil {
        return fmt.Errorf("write: %w", err)
    }

    return nil
}
```

**Tests**: Create `pkg/telemetry/jsonl_writer_test.go` with:
- TestAppendJSONL_Basic
- TestAppendJSONL_CreatesDir
- TestAppendJSONL_Concurrent (100 goroutines)

---

## Task 2.1: Create ReviewFinding Telemetry Struct

**File**: `pkg/telemetry/review_finding.go` (NEW)

See spec file for complete implementation including:
- ReviewFinding struct
- ReviewOutcomeUpdate struct
- NewReviewFinding()
- LogReviewFinding()
- UpdateReviewFindingOutcome()
- LookupFindingTimestamp()
- ReadReviewFindings()

**Dependencies**:
- github.com/google/uuid
- pkg/config (for path helpers from Task 1.1)

**Tests**: Create `pkg/telemetry/review_finding_test.go`

---

## Task 2.2: Create SharpEdgeHit Telemetry Struct

**File**: `pkg/telemetry/sharp_edge_hit.go` (NEW)

See spec file for complete implementation including:
- SharpEdgeHit struct
- NewSharpEdgeHit() (validates against registry)
- LogSharpEdgeHit()

**Tests**: Create `pkg/telemetry/sharp_edge_hit_test.go`

---

## Task 2.3: Create Sharp Edge ID Registry

**File**: `pkg/telemetry/sharp_edge_registry.go` (NEW)

See spec file for complete implementation including:
- SharpEdgesYAML struct
- LoadSharpEdgeIDs()
- IsValidSharpEdgeID()
- GetAllSharpEdgeIDs()

**Tests**: Create `pkg/telemetry/sharp_edge_registry_test.go`

---

## TODO

- [x] Create pkg/telemetry/jsonl_writer.go
- [x] Create pkg/telemetry/jsonl_writer_test.go with concurrent test
- [x] Create pkg/telemetry/review_finding.go with all structs/functions
- [x] Create pkg/telemetry/review_finding_test.go
- [x] Create pkg/telemetry/sharp_edge_hit.go
- [x] Create pkg/telemetry/sharp_edge_hit_test.go
- [x] Create pkg/telemetry/sharp_edge_registry.go
- [x] Create pkg/telemetry/sharp_edge_registry_test.go
- [x] Add github.com/google/uuid to go.mod if needed
- [x] Run go build ./...
- [x] Run go test ./pkg/telemetry/...
- [x] Verify concurrent write test passes (no corruption)

## DO NOT

- Do NOT modify existing telemetry files (routing_decision.go, etc.)
- Do NOT modify agent YAML files
- Do NOT modify hook binaries
- Do NOT use global state without sync.Once protection
- Do NOT skip the file locking in AppendJSONL

---

## VERIFICATION

```bash
go mod tidy
go build ./...
go test ./pkg/telemetry/... -v
go test ./pkg/telemetry/... -race  # Race detector
```

## COMMIT MESSAGE

```
feat(telemetry): Add review finding and sharp edge telemetry structs

- Add AppendJSONL with file locking for concurrent writes
- Add ReviewFinding and ReviewOutcomeUpdate structs
- Add SharpEdgeHit struct with registry validation
- Add SharpEdgeRegistry for validating sharp_edge_id values
- Add comprehensive tests including concurrent write test

Tasks 2.0-2.3 from review-skill-upgrade-plan-v2.md
```
```

---

## Session 3: CLI Tools + Integration

### Batch D: CLI Binaries

```
Implement Review Skill Upgrade - Batch D (Tasks 3.1-3.2)

SPEC FILE: /home/doktersmol/Documents/GOgent-Fortress/tickets/review-skill-upgrade/review-skill-upgrade-plan-v2.md

BATCH: D - Tasks 3.1, 3.2

---

## Task 3.1: Create gogent-log-review Binary

**File**: `cmd/gogent-log-review/main.go` (NEW)

Create CLI that:
1. Reads JSON from stdin (ReviewInput struct)
2. Logs each finding via telemetry.LogReviewFinding()
3. Logs sharp edge hits via telemetry.LogSharpEdgeHit()
4. Validates sharp_edge_id against registry
5. Outputs JSON summary (LogOutput struct)

See spec file for complete implementation.

**Input JSON format**:
```json
{
  "session_id": "...",
  "review_scope": "staged",
  "files_reviewed": 5,
  "findings": [...]
}
```

**Output JSON format**:
```json
{
  "logged": 3,
  "finding_ids": ["uuid1", "uuid2", "uuid3"],
  "sharp_edge_hits": 1,
  "invalid_edge_ids": []
}
```

---

## Task 3.2: Create gogent-update-review-outcome Binary

**File**: `cmd/gogent-update-review-outcome/main.go` (NEW)

Create CLI that:
1. Accepts flags: --finding-id, --resolution, --ticket-id, --commit
2. Validates resolution (fixed, wontfix, false_positive, deferred)
3. Looks up original finding timestamp for accurate resolution_ms
4. Calls telemetry.UpdateReviewFindingOutcome()

See spec file for complete implementation.

**Usage**:
```bash
gogent-update-review-outcome --finding-id=abc123 --resolution=fixed --commit=def456
```

---

## Makefile Updates

Add to Makefile:

```makefile
.PHONY: telemetry-tools
telemetry-tools: gogent-log-review gogent-update-review-outcome

gogent-log-review:
	go build -o bin/gogent-log-review ./cmd/gogent-log-review

gogent-update-review-outcome:
	go build -o bin/gogent-update-review-outcome ./cmd/gogent-update-review-outcome
```

---

## TODO

- [x] Create cmd/gogent-log-review/main.go
- [x] Create cmd/gogent-update-review-outcome/main.go
- [x] Add Makefile targets for new binaries
- [x] Build binaries successfully
- [x] Test gogent-log-review with sample JSON input
- [x] Test gogent-update-review-outcome with sample flags
- [x] Verify invalid sharp_edge_id is reported in output

## DO NOT

- Do NOT install binaries to ~/.local/bin yet (just build to bin/)
- Do NOT modify existing hook binaries
- Do NOT modify pkg/telemetry (should be done in Batch C)
- Do NOT add timeout handling (keep simple for v1)

---

## VERIFICATION

```bash
go build ./cmd/gogent-log-review/...
go build ./cmd/gogent-update-review-outcome/...

# Test gogent-log-review
echo '{"session_id":"test","review_scope":"staged","files_reviewed":1,"findings":[{"severity":"warning","reviewer":"backend-reviewer","category":"security","file":"test.go","line":1,"message":"test"}]}' | ./bin/gogent-log-review

# Test gogent-update-review-outcome
./bin/gogent-update-review-outcome --finding-id=test123 --resolution=fixed
```

## COMMIT MESSAGE

```
feat(cli): Add gogent-log-review and gogent-update-review-outcome binaries

- Add gogent-log-review: logs review findings to telemetry
- Add gogent-update-review-outcome: updates finding resolutions
- Add Makefile targets for telemetry-tools

Tasks 3.1-3.2 from review-skill-upgrade-plan-v2.md
```
```

---

### Batch E: Skill Integration

```
Implement Review Skill Upgrade - Batch E (Tasks 3.3, 4.1, 4.2)

SPEC FILE: /home/doktersmol/Documents/GOgent-Fortress/tickets/review-skill-upgrade/review-skill-upgrade-plan-v2.md

BATCH: E - Tasks 3.3, 4.1, 4.2

---

## Task 3.3: Integrate Telemetry into review-orchestrator

**File**: `.claude/agents/review-orchestrator/CLAUDE.md`

Add to "Synthesis Phase" section:

```markdown
### Telemetry Requirements

After collecting findings, ensure output includes telemetry-compatible format:

```json
{
  "session_id": "[from context or generate UUID]",
  "status": "BLOCKED",
  "summary": { "critical": 2, "warnings": 3, "info": 1 },
  "findings": [
    {
      "severity": "critical",
      "reviewer": "backend-reviewer",
      "category": "security",
      "file": "src/api/handler.go",
      "line": 45,
      "message": "SQL injection via string concatenation",
      "recommendation": "Use parameterized queries",
      "sharp_edge_id": "sql-injection"
    }
  ]
}
```

**Required fields per finding**:
- `severity`: critical, warning, info
- `reviewer`: Which specialist found it
- `category`: security, performance, accessibility, maintainability, etc.
- `file`: Full file path
- `line`: Line number (0 if not applicable)
- `message`: Issue description
- `recommendation`: Fix suggestion
- `sharp_edge_id`: If matches known pattern (optional, must be valid ID)

**IMPORTANT**: The `session_id` field is REQUIRED for telemetry correlation.
```

---

## Task 4.1: Update /review Skill for Telemetry

**File**: `.claude/skills/review/SKILL.md`

Add to Phase 4 (Report Generation):

```markdown
### Phase 4: Report Generation and Telemetry

After orchestrator completes:

```bash
# Read review result
review_result=$(cat .claude/tmp/review-result.json)

# Check if telemetry is enabled (default: enabled)
if [[ "${GOGENT_ENABLE_TELEMETRY:-1}" == "1" ]]; then
    # Build telemetry input
    telemetry_input=$(jq -n \
        --arg sid "$(echo "$review_result" | jq -r '.session_id // "unknown"')" \
        --arg scope "${review_scope:-staged}" \
        --argjson files "$(echo "$files" | wc -l)" \
        --argjson findings "$(echo "$review_result" | jq '.findings')" \
        '{session_id: $sid, review_scope: $scope, files_reviewed: $files, findings: $findings}')

    # Log to ML telemetry (non-blocking)
    echo "$telemetry_input" | gogent-log-review > .claude/tmp/review-telemetry.json 2>/dev/null || true
fi
```
```

Add new section:

```markdown
## ML Telemetry

This skill logs all findings to ML telemetry for downstream analysis:

| File | Purpose |
|------|---------|
| `$XDG_DATA_HOME/gogent-fortress/review-findings.jsonl` | All review findings |
| `$XDG_DATA_HOME/gogent-fortress/sharp-edge-hits.jsonl` | Sharp edge correlations |
| `.claude/tmp/review-telemetry.json` | Session telemetry output |

### Disabling Telemetry

Set `GOGENT_ENABLE_TELEMETRY=0` to disable telemetry logging.
```

---

## Task 4.2: Update /ticket Phase 7.6 for Outcome Tracking

**File**: `.claude/skills/ticket/SKILL.md`

Update Phase 7.6 (Code Review):

```markdown
### Phase 7.6: Code Review (Blocking)

After audit, run code review if enabled:

```bash
# ... existing review invocation code ...

# Store finding IDs for outcome tracking
if [[ "${GOGENT_ENABLE_TELEMETRY:-1}" == "1" ]] && [[ -f .claude/tmp/review-telemetry.json ]]; then
    jq -r '.finding_ids[]' .claude/tmp/review-telemetry.json > "$tickets_dir/.review-findings-$ticket_id" 2>/dev/null || true
fi
```
```

Update Phase 8 (Completion):

```markdown
### Phase 8: Completion Workflow

```bash
# ... existing completion code ...

# Log review outcomes if code review was run
if [[ "${GOGENT_ENABLE_TELEMETRY:-1}" == "1" ]] && [[ -f "$tickets_dir/.review-findings-$ticket_id" ]]; then
    commit_hash=$(git rev-parse HEAD 2>/dev/null || echo "")
    while IFS= read -r finding_id; do
        gogent-update-review-outcome \
            --finding-id="$finding_id" \
            --resolution="fixed" \
            --ticket-id="$ticket_id" \
            --commit="$commit_hash" 2>/dev/null || true
    done < "$tickets_dir/.review-findings-$ticket_id"
    rm -f "$tickets_dir/.review-findings-$ticket_id"
fi
```
```

---

## TODO

- [x] Add telemetry requirements section to review-orchestrator/CLAUDE.md
- [x] Add telemetry integration to /review SKILL.md Phase 4
- [x] Add ML Telemetry documentation section to /review SKILL.md
- [x] Add finding ID storage to /ticket SKILL.md Phase 7.6
- [x] Add outcome logging to /ticket SKILL.md Phase 8
- [x] Verify bash syntax is correct
- [x] Verify jq commands are valid

## DO NOT

- Do NOT modify Go code
- Do NOT modify agent.yaml files
- Do NOT modify existing workflow phases (only add to them)
- Do NOT make telemetry blocking (always use `|| true`)
- Do NOT remove existing content from SKILL.md files

---

## VERIFICATION

- All SKILL.md files render correctly in markdown
- Bash snippets have valid syntax
- jq commands are valid

## COMMIT MESSAGE

```
feat(skills): Integrate ML telemetry into /review and /ticket workflows

- Add telemetry output format to review-orchestrator
- Add telemetry logging to /review skill Phase 4
- Add finding ID tracking to /ticket skill Phase 7.6
- Add outcome logging to /ticket skill Phase 8
- Add GOGENT_ENABLE_TELEMETRY feature flag

Tasks 3.3, 4.1, 4.2 from review-skill-upgrade-plan-v2.md
```
```

---

## Session 4: Polish

### Batch F: Export Support + Tests

```
Implement Review Skill Upgrade - Batch F (Tasks 5.1, 6.1)

SPEC FILE: /home/doktersmol/Documents/GOgent-Fortress/tickets/review-skill-upgrade/review-skill-upgrade-plan-v2.md

BATCH: F - Tasks 5.1, 6.1

---

## Task 5.1: Add gogent-ml-export Review Support

**File**: `cmd/gogent-ml-export/main.go`

Add new subcommands:

```go
case "review-findings":
    findings, err := telemetry.ReadReviewFindings()
    if err != nil {
        return err
    }
    return outputJSON(findings, outputPath)

case "review-stats":
    findings, _ := telemetry.ReadReviewFindings()
    stats := telemetry.CalculateReviewStats(findings)
    return outputJSON(stats, "")

case "sharp-edge-hits":
    hits, err := telemetry.ReadSharpEdgeHits()
    if err != nil {
        return err
    }
    return outputJSON(hits, outputPath)
```

**Add to pkg/telemetry/review_finding.go**:

```go
// ReadReviewFindings reads all findings from storage
func ReadReviewFindings() ([]ReviewFinding, error) {
    path := config.GetReviewFindingsPathWithProjectDir()
    // Read JSONL file line by line
    // Unmarshal each line into ReviewFinding
    // Return slice
}

// CalculateReviewStats returns aggregate metrics
func CalculateReviewStats(findings []ReviewFinding) map[string]interface{} {
    // Group by severity, reviewer, category
    // Calculate fix rates, average resolution time
    return map[string]interface{}{
        "total_findings": len(findings),
        "by_severity": map[string]int{...},
        "by_reviewer": map[string]int{...},
        "by_category": map[string]int{...},
    }
}
```

**Add to pkg/telemetry/sharp_edge_hit.go**:

```go
// ReadSharpEdgeHits reads all hits from storage
func ReadSharpEdgeHits() ([]SharpEdgeHit, error) {
    path := config.GetSharpEdgeHitsPathWithProjectDir()
    // Read JSONL file line by line
    // Return slice
}
```

---

## Task 6.1: Comprehensive Test Coverage

Ensure all test files exist and pass:

**Required test files**:
- `pkg/telemetry/jsonl_writer_test.go`
- `pkg/telemetry/review_finding_test.go`
- `pkg/telemetry/sharp_edge_hit_test.go`
- `pkg/telemetry/sharp_edge_registry_test.go`

**Test isolation helper** (add to each test file if missing):

```go
func setupTestDir(t *testing.T) func() {
    t.Helper()
    dir := t.TempDir()
    t.Setenv("GOGENT_PROJECT_DIR", dir)

    os.MkdirAll(filepath.Join(dir, ".claude", "memory"), 0755)
    os.MkdirAll(filepath.Join(dir, ".claude", "agents"), 0755)

    return func() {}
}
```

**Critical test cases**:
- TestAppendJSONL_Concurrent (100 goroutines, verify no corruption)
- TestIsValidSharpEdgeID (valid and invalid IDs)
- TestLogReviewFinding (write and read back)
- TestLookupFindingTimestamp (accurate lookup)

---

## TODO

- [x] Add ReadReviewFindings() to review_finding.go
- [x] Add CalculateReviewStats() to review_finding.go
- [x] Add ReadSharpEdgeHits() to sharp_edge_hit.go
- [x] Add review-findings subcommand to gogent-ml-export
- [x] Add review-stats subcommand to gogent-ml-export
- [x] Add sharp-edge-hits subcommand to gogent-ml-export
- [x] Verify all test files exist
- [x] Add setupTestDir helper to all test files
- [x] Run full test suite
- [x] Run race detector tests
- [x] Rebuild gogent-ml-export binary

## DO NOT

- Do NOT modify existing gogent-ml-export subcommands
- Do NOT change existing telemetry file paths
- Do NOT skip test isolation (always use GOGENT_PROJECT_DIR)
- Do NOT remove existing tests

---

## VERIFICATION

```bash
go build ./...
go test ./pkg/telemetry/... -v
go test ./pkg/telemetry/... -race
go test ./cmd/gogent-ml-export/... -v

# Test new subcommands
./bin/gogent-ml-export review-findings
./bin/gogent-ml-export review-stats
./bin/gogent-ml-export sharp-edge-hits
```

## COMMIT MESSAGE

```
feat(telemetry): Add review telemetry export and comprehensive tests

- Add ReadReviewFindings() and CalculateReviewStats()
- Add ReadSharpEdgeHits()
- Add review-findings, review-stats, sharp-edge-hits to gogent-ml-export
- Add comprehensive test coverage with concurrent write tests

Tasks 5.1, 6.1 from review-skill-upgrade-plan-v2.md
```
```

---

## Final Verification Checklist

After all batches complete, run this verification:

```bash
# Build everything
go build ./...
make telemetry-tools

# Run all tests
go test ./... -v
go test ./pkg/telemetry/... -race

# Verify binaries work
echo '{"session_id":"test","review_scope":"staged","files_reviewed":1,"findings":[]}' | ./bin/gogent-log-review
./bin/gogent-update-review-outcome --finding-id=test --resolution=fixed
./bin/gogent-ml-export review-stats

# Verify agents-index.json is valid JSON
jq . .claude/agents/agents-index.json > /dev/null

# Verify all YAML files parse
for f in .claude/agents/*/agent.yaml; do
  python3 -c "import yaml; yaml.safe_load(open('$f'))" || echo "FAIL: $f"
done
```

---

## Summary

| Session | Batch | Tasks | Agent | Est. Commits |
|---------|-------|-------|-------|--------------|
| 1 | A | 1.1-1.3 | go-pro | 1 |
| 1 | B | 1.4-1.9 | general-purpose | 1 |
| 2 | C | 2.0-2.3 | go-pro | 1 |
| 3 | D | 3.1-3.2 | go-pro | 1 |
| 3 | E | 3.3, 4.1-4.2 | general-purpose | 1 |
| 4 | F | 5.1, 6.1 | go-pro | 1 |

**Total: 6 commits across 4 sessions**
