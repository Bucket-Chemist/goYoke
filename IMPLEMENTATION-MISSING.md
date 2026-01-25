# Implementation Missing - GOgent-Fortress

This document tracks partially completed tickets with missing prerequisite functionality.

---

## GOgent-101: ML Telemetry Integration Tests

**Status**: Tests implemented, awaiting prerequisite functionality
**Test File**: `test/integration/ml_telemetry_test.go` ✅ (609 lines, 7 test functions)
**Current Results**: 3/7 tests pass, 4 fail due to missing ML capture

### What's Complete ✅

- Test suite fully implemented (609 lines)
- 7 test functions covering all scenarios
- Race detector clean (zero data races)
- Proper test isolation with t.TempDir()
- Realistic corpus generation helpers
- Comprehensive validation logic

### What's Missing ❌

#### 1. ML Telemetry Capture in gogent-validate Hook

**Blocker**: Tests expect `routing-decisions.jsonl` to be created automatically by the validate hook.

**Required Implementation**:
```go
// In pkg/validate/handler.go or similar
func captureRoutingDecision(event HookEvent) error {
    projectDir := event.ProjectDir
    telemetryDir := filepath.Join(projectDir, ".gogent")
    os.MkdirAll(telemetryDir, 0755)

    decisionsFile := filepath.Join(telemetryDir, "routing-decisions.jsonl")

    decision := map[string]interface{}{
        "timestamp": time.Now().Unix(),
        "tool_name": event.ToolName,
        "routing_decision": event.Decision,
        "session_id": event.SessionID,
        "ml_duration_ms": event.DurationMs,
        "ml_input_tokens": event.InputTokens,
        "ml_output_tokens": event.OutputTokens,
    }

    data, _ := json.Marshal(decision)
    f, _ := os.OpenFile(decisionsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    defer f.Close()
    f.Write(append(data, '\n'))

    return nil
}
```

**Files Affected**:
- `cmd/gogent-validate/main.go`
- `pkg/validate/handler.go` (new file?)
- `pkg/telemetry/capture.go` (new package?)

**Tests Blocked**:
- `TestMLTelemetry_RoutingDecisionCapture` ❌
- `TestMLTelemetry_DecisionUpdates` ✅ (passes with mock data)
- `TestMLTelemetry_ConcurrentWrites` ❌

---

#### 2. ML Telemetry Capture in gogent-sharp-edge Hook

**Blocker**: Tests expect `agent-collaborations.jsonl` to be created when agents collaborate.

**Required Implementation**:
```go
// In pkg/sharp-edge/handler.go or similar
func captureCollaboration(event HookEvent) error {
    projectDir := event.ProjectDir
    telemetryDir := filepath.Join(projectDir, ".gogent")

    collaborationsFile := filepath.Join(telemetryDir, "agent-collaborations.jsonl")

    collab := map[string]interface{}{
        "agent_name": event.AgentName,
        "action": event.Action,
        "timestamp": time.Now().Unix(),
        "session_id": event.SessionID,
    }

    data, _ := json.Marshal(collab)
    f, _ := os.OpenFile(collaborationsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    defer f.Close()
    f.Write(append(data, '\n'))

    return nil
}
```

**Files Affected**:
- `cmd/gogent-sharp-edge/main.go`
- `pkg/sharp-edge/handler.go` (new file?)
- `pkg/telemetry/capture.go` (shared package)

**Tests Blocked**:
- `TestMLTelemetry_CollaborationTracking` ❌

---

#### 3. gogent-ml-export CLI Implementation

**Blocker**: Tests expect `gogent-ml-export` binary to convert JSONL to CSV/JSON datasets.

**Required Implementation**:
```bash
# Command interface
gogent-ml-export training-dataset --output ./ml-export/

# Expected output files:
# - routing-decisions.csv
# - tool-sequences.json
# - agent-collaborations.csv
# - metadata.json
```

**CSV Format** (routing-decisions.csv):
```csv
timestamp,tool_name,routing_decision,session_id,ml_duration_ms,ml_input_tokens,ml_output_tokens
1706198400,Read,allow,session-1,150,1000,500
```

**Metadata Format** (metadata.json):
```json
{
  "export_timestamp": 1706198400,
  "decision_count": 15,
  "collaboration_count": 3,
  "source_files": {
    "routing_decisions": ".gogent/routing-decisions.jsonl",
    "collaborations": ".gogent/agent-collaborations.jsonl"
  }
}
```

**Files Affected**:
- `cmd/gogent-ml-export/main.go` (new CLI)
- `pkg/export/csv.go` (new package)
- `pkg/export/reconcile.go` (new package)

**Tests Blocked**:
- `TestMLTelemetry_ExportReconciliation` ❌

---

### Test Results Summary

| Test Function | Status | Blocker |
|---------------|--------|---------|
| TestMLTelemetry_RoutingDecisionCapture | ❌ FAIL | Missing ML capture in validate hook |
| TestMLTelemetry_DecisionUpdates | ✅ PASS | Works with existing logic |
| TestMLTelemetry_ConcurrentWrites | ❌ FAIL | Missing ML capture in validate hook |
| TestMLTelemetry_CollaborationTracking | ❌ FAIL | Missing collaboration capture in sharp-edge hook |
| TestMLTelemetry_ExportReconciliation | ❌ FAIL | Missing gogent-ml-export CLI |
| TestMLTelemetry_RaceConditionDetection | ✅ PASS | Race detector clean |
| TestMLTelemetry_SequenceIntegrity | ✅ PASS | Validates field propagation |

**Current**: 3/7 tests pass (43%)
**When Prerequisites Complete**: 7/7 tests expected to pass (100%)

---

### Implementation Priority

1. **High**: ML capture in gogent-validate (blocks 3 tests)
2. **Medium**: Collaboration capture in gogent-sharp-edge (blocks 1 test)
3. **Low**: gogent-ml-export CLI (blocks 1 test, but needed for actual ML training)

### Estimated Effort

- ML capture implementation: ~2 hours
- Collaboration capture: ~1 hour
- gogent-ml-export CLI: ~3 hours
- **Total**: ~6 hours to unblock all tests

---

## Next Steps

1. Implement ML capture in hooks (GOgent-089b dependency)
2. Re-run tests: `go test ./test/integration -v -run TestMLTelemetry`
3. Verify race detector: `go test -race ./test/integration -run TestMLTelemetry`
4. Measure coverage: `go test -coverprofile=coverage.out ./test/integration -run TestMLTelemetry`
5. Run ecosystem tests: `make test-ecosystem`

---

**Document Created**: 2026-01-25
**Last Updated**: 2026-01-25
**Ticket**: GOgent-101
**Related Tickets**: GOgent-089b (ML Export CLI), GOgent-094 (test harness)
