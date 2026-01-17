# Phase 0 Overview - Standards and Strategy

**Version**: 1.1 FINAL (Critical Review Applied)
**Date**: 2026-01-15
**Status**: ✅ Staff Architect Approved

---

## Document Purpose

This document contains cross-cutting standards that apply to **ALL** Phase 0 tickets. Contractors should read this file completely before starting any implementation work.

**Contents**:
1. Change Log (v1.0 → v1.1)
2. Testing Strategy
3. Rollback Plan
4. Success Criteria
5. Error Handling Standards
6. Logging Strategy
7. Critical Files Priority

---

## Change Log (V1.0 → V1.1)

### Critical Fixes Applied

The following changes were made based on staff architect critical review:

| Ticket | Change | Review Issue | Impact |
|--------|--------|--------------|--------|
| **GOgent-000** | Added pre-work baseline measurement | C-2 | WITHOUT baseline, cannot verify Go doesn't regress |
| **GOgent-002b** | Complete all schema structs (no "omitted for brevity") | M-1 | Contractor needs complete definitions |
| **GOgent-004a/004c** | Split config loader to fix circular dependency | C-1 | Event parsing required schema, schema tests required events |
| **GOgent-008b** | Capture real event corpus during Week 1 | C-3 | 100 real events for regression testing |
| **GOgent-024b** | Wire validation orchestrator | Implied | Connect all validation checks |
| **GOgent-033** | Benchmark all hooks against baseline | Performance | Verify Go ≤ Bash latency |
| **GOgent-048b** | WSL2 testing coverage | M-8 | Ensure Windows compatibility |
| **All tickets** | File paths: /tmp → XDG with fallback | M-2 | Avoid noexec and reboot clearing |
| **All hooks** | Add stdin timeout (5s default) | M-6 | Prevent hanging hooks |
| **All tickets** | Formalize error message standards | Consistency | `[component] What. Why. How to fix.` |
| **All tickets** | Add logging strategy | Observability | Structured logs to ~/.gogent/hooks.log |

### Version Summary

- **v1.0**: Initial 50-ticket plan based on v2 migration plan
- **v1.1 FINAL**: +6 tickets, all critical/major review issues resolved

Total tickets: **56** (1 pre-work + 55 implementation)

---

## Testing Strategy

### 1. Unit Tests (Continuous - Every Ticket)

**Coverage Target**: ≥80% per package

**Test Naming**: `TestFunctionName_Scenario`

**Required Test Cases**:
- Valid input (happy path)
- Invalid input (error handling)
- Edge cases (empty strings, nil pointers, boundary values)
- Error conditions (file not found, permission denied, timeout)

**Example Structure**:
```go
func TestLoadRoutingSchema_ValidFile(t *testing.T) {
    schema, err := LoadRoutingSchema()
    if err != nil {
        t.Fatalf("Expected no error, got: %v", err)
    }
    if schema.Version == "" {
        t.Error("Expected version field to be populated")
    }
}

func TestLoadRoutingSchema_MissingFile(t *testing.T) {
    // Test with non-existent file path
    _, err := LoadRoutingSchemaFromPath("/tmp/nonexistent.json")
    if err == nil {
        t.Error("Expected error for missing file, got nil")
    }
    // Verify error message format
    if !strings.Contains(err.Error(), "[config]") {
        t.Errorf("Expected error with component tag, got: %v", err)
    }
}
```

**Run After Each Ticket**:
```bash
go test ./...
```

**Before Committing**:
```bash
go test ./... -cover
```

### 2. Integration Tests (Week 3 - GOgent-041 to 046)

**Purpose**: Test end-to-end workflows with real Claude Code event data

**Test Corpus**: 100 real events captured in GOgent-008b
- 25 Task events (various models/agents)
- 20 Read events
- 15 Write events
- 15 Edit events
- 10 Bash events
- 10 Glob events
- 5 Grep events

**Test Harness**:
```go
func TestValidateRouting_RealCorpus(t *testing.T) {
    // Load 100-event corpus
    corpus := loadEventCorpus("../../test/fixtures/event-corpus.json")

    for i, event := range corpus {
        // Run Go implementation
        goOutput := runGoValidation(event)

        // Run Bash implementation (parallel testing period)
        bashOutput := runBashValidation(event)

        // Compare outputs (byte-for-byte except timestamps)
        if !outputsMatch(goOutput, bashOutput) {
            t.Errorf("Event %d: Output mismatch\nGo:   %s\nBash: %s",
                i, goOutput, bashOutput)
        }
    }
}
```

**Pass Criteria**: 100% match on validation decisions (allow/block)

### 3. Regression Tests (Week 3 - GOgent-047)

**Purpose**: Ensure Go output exactly matches Bash output

**Method**:
1. Feed same event through both Bash and Go hooks
2. Capture JSON output from both
3. Diff outputs (ignore timestamp fields)
4. Require 100% match

**Test Script**:
```bash
#!/bin/bash
# test/regression/compare-outputs.sh

BASH_HOOK="$HOME/.claude/hooks/validate-routing.sh"
GO_HOOK="./gogent-validate"

# Test each event from corpus
for event in $(cat test/fixtures/event-corpus.json | jq -c '.[]'); do
    # Run Bash hook
    bash_output=$(echo "$event" | $BASH_HOOK)

    # Run Go hook
    go_output=$(echo "$event" | $GO_HOOK)

    # Compare (strip timestamps)
    bash_normalized=$(echo "$bash_output" | jq 'del(.timestamp)')
    go_normalized=$(echo "$go_output" | jq 'del(.timestamp)')

    if [ "$bash_normalized" != "$go_normalized" ]; then
        echo "MISMATCH on event: $event"
        echo "Bash: $bash_normalized"
        echo "Go:   $go_normalized"
        exit 1
    fi
done

echo "✓ All 100 events match"
```

**Pass Criteria**: Zero mismatches

### 4. Performance Benchmarks (Week 3 - GOgent-033)

**Baseline**: Measured in GOgent-000 (pre-work)

**SLA**:
- **Target**: Go hooks ≤ Bash average latency
- **Acceptable**: +20% degradation (e.g., if Bash is 5ms, Go <6ms OK)
- **Unacceptable**: >10ms p99 latency

**Benchmark Test**:
```go
func BenchmarkValidateRouting(b *testing.B) {
    event := loadSampleEvent()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = ValidateRouting(event)
    }
}
```

**Run Benchmarks**:
```bash
go test -bench=. ./test/benchmark
```

**Comparison**:
```bash
# Run Go benchmark
go test -bench=BenchmarkValidateRouting -benchtime=100x ./test/benchmark

# Compare to Bash baseline from GOgent-000
cat ~/gogent-baseline/BASELINE.md
```

**Example Output**:
```
Bash (from GOgent-000):  Average 4.2ms per event
Go (from benchmark):    Average 3.8ms per event
Result:                 ✓ PASS (9.5% faster)
```

---

## Rollback Plan

### Trigger Conditions

Rollback to Bash hooks if:
1. **Critical bug** in Go implementation discovered during parallel testing
2. **Performance regression** >20% slower than Bash baseline
3. **Correctness issue**: Go produces different validation decisions than Bash
4. **Production incident**: Hooks causing Claude Code to hang/crash

### Immediate Rollback (< 5 minutes)

**Step 1**: Stop using Go hooks
```bash
cd ~/.claude/hooks

# Restore Bash scripts
mv validate-routing.go.bak validate-routing.sh
mv session-archive.go.bak session-archive.sh
mv sharp-edge-detector.go.bak sharp-edge-detector.sh

chmod +x *.sh
```

**Step 2**: Verify Bash hooks active
```bash
# Test hook manually
echo '{"tool_name":"Task","tool_input":{"model":"sonnet"},"session_id":"test"}' | \
    ~/.claude/hooks/validate-routing.sh

# Should output JSON decision
```

**Step 3**: Restart Claude Code sessions
```bash
# Kill any running sessions
pkill -f claude

# Start fresh session
claude
```

### Investigation Period (1 hour)

1. **Collect logs**: Check `~/.gogent/hooks.log` for error messages
2. **Review violations**: Check `~/.gogent/routing-violations.jsonl` for patterns
3. **Reproduce issue**: Use failing event from corpus to reproduce
4. **Identify root cause**: Bug in Go implementation? Missing edge case?

### Fix and Re-Deploy

1. **Fix bug** in Go code
2. **Add regression test** for the bug
3. **Re-test locally** with full corpus
4. **Re-deploy** after verification
5. **Monitor** for 24hrs before declaring success

### Risk Mitigation

**Parallel Testing Period** (GOgent-049):
- Run both Bash and Go hooks simultaneously for 24 hours
- Go hooks write decisions to separate log file
- Compare decisions post-facto (no impact on production)
- Only cutover if 100% match

This catches issues BEFORE full cutover, making rollback unnecessary.

---

## Success Criteria (Phase 0)

### Functional Requirements

- [ ] All 3 Go binaries (`gogent-validate`, `gogent-archive`, `gogent-sharp-edge`) built successfully
- [ ] Identical JSON output to Bash versions (byte-for-byte except timestamps)
- [ ] All unit tests pass: `go test ./...` (≥80 tests across all packages)
- [ ] All integration tests pass: 100 events from corpus validated correctly
- [ ] Regression tests pass: Output diff = 0 (100% match with Bash)

### Performance Requirements

- [ ] Hook execution latency ≤ baseline measured in GOgent-000
- [ ] Target: <5ms p99 latency per hook execution
- [ ] Memory usage: <10MB per process
- [ ] CPU usage: <1% when idle

### Operational Requirements

- [ ] Installation script works: `scripts/install.sh` succeeds on clean system
- [ ] Parallel testing runs for 24hrs without issues (GOgent-049)
- [ ] Rollback plan documented and tested (can revert in <5 minutes)
- [ ] Error messages follow standard format (100% compliance)
- [ ] All logs written to `~/.gogent/hooks.log` (structured JSON)
- [ ] WSL2 compatibility verified (GOgent-048b)

### Quality Requirements

- [ ] Code coverage ≥80% across all packages
- [ ] No TODOs or placeholders in production code
- [ ] All error paths tested
- [ ] All edge cases handled (empty input, missing files, invalid JSON, etc.)
- [ ] Documentation complete (README, inline comments for non-obvious logic)

### Verification Checklist

Before declaring Phase 0 complete:

1. **Build all binaries**:
   ```bash
   go build -o gogent-validate ./cmd/gogent-validate
   go build -o gogent-archive ./cmd/gogent-archive
   go build -o gogent-sharp-edge ./cmd/gogent-sharp-edge
   ```

2. **Run all tests**:
   ```bash
   go test ./... -cover
   ```

3. **Run benchmarks**:
   ```bash
   go test -bench=. ./test/benchmark
   ```

4. **Run regression tests**:
   ```bash
   ./test/regression/compare-outputs.sh
   ```

5. **Test installation**:
   ```bash
   ./scripts/install.sh
   ```

6. **Verify hooks work**:
   ```bash
   echo '{"tool_name":"Task","tool_input":{"model":"sonnet"},"session_id":"test"}' | \
       ~/.claude/hooks/validate-routing
   ```

---

## Error Handling Standards

### Error Message Format

**Required**: `[component] What happened. Why it was blocked/failed. How to fix.`

**Components**:
- `[component]`: Source of error (e.g., `config`, `event-parser`, `validate-routing`)
- **What**: Specific error that occurred
- **Why**: Reason for failure/block
- **How to fix**: Actionable guidance for user

### Examples

#### Good Error Messages ✅

```go
// Missing file
return fmt.Errorf("[config] Failed to read routing schema at %s: %w. Ensure .claude/ directory exists.", path, err)

// Invalid JSON
return fmt.Errorf("[event-parser] Failed to parse JSON: %w. Check STDIN format: %s", err, string(data[:100]))

// Validation blocked
return fmt.Errorf("[validate-routing] Task(opus) blocked. Einstein requires GAP document workflow for cost control. Generate GAP: .claude/tmp/einstein-gap-{timestamp}.md, then run /einstein.")

// Timeout
return fmt.Errorf("[event-parser] STDIN read timeout after %v. Hook may be stuck waiting for input.", timeout)

// Schema version mismatch
return fmt.Errorf("[config] Schema version mismatch. Expected %s, got %s. Update gogent binaries or routing-schema.json.", expected, actual)
```

#### Bad Error Messages ❌

```go
// No context
return errors.New("config error")

// No guidance
return fmt.Errorf("failed to load")

// Unclear component
return fmt.Errorf("error: %v", err)

// Missing "how to fix"
return fmt.Errorf("file not found: %s", path)
```

### Error Wrapping

Use `%w` to wrap errors for stack traces:

```go
data, err := os.ReadFile(path)
if err != nil {
    return nil, fmt.Errorf("[config] Failed to read file: %w", err)
}
```

### Error Testing

Every error path MUST have a test:

```go
func TestLoadRoutingSchema_MissingFile(t *testing.T) {
    _, err := LoadRoutingSchemaFromPath("/tmp/nonexistent.json")

    if err == nil {
        t.Error("Expected error for missing file, got nil")
    }

    // Verify error message format
    errMsg := err.Error()
    if !strings.Contains(errMsg, "[config]") {
        t.Errorf("Expected component tag, got: %s", errMsg)
    }
    if !strings.Contains(errMsg, "Ensure .claude/") {
        t.Errorf("Expected guidance, got: %s", errMsg)
    }
}
```

### GOgent-009 Implementation Note

**Original Plan (tickets-index.json v1.0):** Create `pkg/errors/format.go` with centralized error formatter.

**Implemented Approach (v1.1):** Distributed `fmt.Errorf` convention following documented standard.

**Rationale:** For a 3-binary system with 55 error sites, the distributed approach is:
- More idiomatic Go (matches `fmt.Errorf` best practices)
- Simpler to maintain (zero dependencies, no import overhead)
- Already implemented across all tickets (55+ compliant call sites)
- Verifiable through testing (error format checks in all test suites)

**Convention Enforcement:**
1. **Code Review**: All PRs checked for `[component] What. Why. How.` format
2. **Test Coverage**: Every error path must verify message format (see examples above)
3. **CI Verification**: `grep -r 'fmt\.Errorf' | grep -v '\[.*\]'` flags non-compliant errors

**Future:** If error sites grow to >200 or structured logging becomes critical, revisit centralized approach in Phase 1.

---

## Logging Strategy

### Log File Location

**Path**: `~/.gogent/hooks.log`

**Format**: Structured JSON (one entry per line - JSONL)

### Log Levels

- **ERROR**: Hook failures, validation blocks, critical issues
- **WARN**: Deprecation warnings, override flags used, near-threshold conditions
- **INFO**: Successful validations, normal operations (default level)
- **DEBUG**: Detailed execution flow (disabled in production)

### Log Entry Structure

```json
{
  "timestamp": "2026-01-15T10:30:45Z",
  "level": "ERROR",
  "component": "validate-routing",
  "message": "Task(opus) blocked - GAP required",
  "session_id": "abc-123",
  "tool_name": "Task",
  "model": "opus",
  "reason": "task_invocation_blocked",
  "decision": "block"
}
```

### Implementation

```go
package logger

import (
    "encoding/json"
    "os"
    "time"
)

type LogEntry struct {
    Timestamp string                 `json:"timestamp"`
    Level     string                 `json:"level"`
    Component string                 `json:"component"`
    Message   string                 `json:"message"`
    Fields    map[string]interface{} `json:"fields,omitempty"`
}

func Error(component, message string, fields map[string]interface{}) {
    log("ERROR", component, message, fields)
}

func log(level, component, message string, fields map[string]interface{}) {
    entry := LogEntry{
        Timestamp: time.Now().Format(time.RFC3339),
        Level:     level,
        Component: component,
        Message:   message,
        Fields:    fields,
    }

    data, _ := json.Marshal(entry)
    logFile := getLogPath() // ~/.gogent/hooks.log

    f, _ := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    defer f.Close()
    f.Write(append(data, '\n'))
}
```

### Usage in Hooks

```go
// Log validation decision
logger.Info("validate-routing", "Task validation complete", map[string]interface{}{
    "session_id":     event.SessionID,
    "tool_name":      event.ToolName,
    "model":          taskInput.Model,
    "decision":       "allow",
    "agent":          extractedAgent,
    "tier":           determinedTier,
})

// Log error
logger.Error("config", "Failed to load routing schema", map[string]interface{}{
    "path":  schemaPath,
    "error": err.Error(),
})
```

---

## Critical Files Priority

Based on complexity and risk, implement in this order:

### Week 1 (Highest Priority)

1. **pkg/routing/schema.go** - Foundation for everything (GOgent-002/002b)
2. **pkg/config/loader.go** - Config loading + version validation (GOgent-004a)
3. **pkg/routing/events.go** - Event parsing (GOgent-006/007)
4. **pkg/routing/validation.go** - Orchestrator - most complex (GOgent-020-024)
5. **cmd/gogent-validate/main.go** - Hook entry point (GOgent-025)

### Week 2 (Medium Priority)

6. **pkg/session/archive.go** - Session archival (GOgent-026-033)
7. **pkg/memory/detector.go** - Sharp edge detection (GOgent-034-040)
8. **cmd/gogent-archive/main.go** - Archive hook entry (GOgent-033)
9. **cmd/gogent-sharp-edge/main.go** - Sharp edge hook entry (GOgent-040)

### Week 3 (Testing Priority)

10. **test/integration/validate_test.go** - Quality gate (GOgent-041-046)
11. **test/benchmark/hooks_bench.go** - Performance gate (GOgent-033)
12. **scripts/install.sh** - Deployment (GOgent-048)

---

## Cross-References

- **Ticket Template**: [TICKET-TEMPLATE.md](TICKET-TEMPLATE.md) - Required structure
- **Navigation**: [README.md](README.md) - File index and quick find
- **Critical Review**: [../CRITICAL_REVIEW.md](../CRITICAL_REVIEW.md) - Issues that drove decisions
- **Migration Plan**: [../gogent_migration_plan_v3_FINAL.md](../gogent_migration_plan_v3_FINAL.md) - Architecture

---

**Last Updated**: 2026-01-15
**Version**: 1.1 FINAL
**Status**: ✅ Ready for implementation
