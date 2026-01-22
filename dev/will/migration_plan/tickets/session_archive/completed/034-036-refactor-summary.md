# Ticket Series 034-036: Sharp Edge Detection - Architecture Summary

**Status**: Refactored for zero overlap, clean layering
**Generated**: 2026-01-23

---

## Ticket Dependency Chain

```
034 (Detection)
 │
 ├──→ 036 (Tracking)
 │     │
 │     └──→ 036c (Function Extraction - OPTIONAL)
 │
 └──→ 035 (CLI Orchestration)
       ├── Depends on: 034 ✅
       └── Depends on: 036 ✅
```

---

## Architectural Layering

### Layer 1: Detection (pkg/routing)

**Ticket**: 034
**Package**: `pkg/routing/failure.go`
**Question**: "Is this tool response a failure?"

```go
// Input: PostToolEvent from hook
// Output: FailureInfo or nil

failure := routing.DetectFailure(event)
// → {File: "test.py", ErrorType: "typeerror", Timestamp: 1234}
```

**Responsibilities:**
- Parse `PostToolEvent.ToolResponse`
- Detect explicit `success: false`
- Detect non-zero exit codes
- Detect error patterns (Python errors, generic keywords)
- Extract file path from event

**Does NOT:**
- Track consecutive failures
- Write to disk
- Make hook decisions

---

### Layer 2: Tracking (pkg/memory)

**Ticket**: 036 (unified with 036b)
**Package**: `pkg/memory/failure_tracking.go`
**Question**: "How many times has THIS error failed on THIS file recently?"

```go
// Input: FailureKey{FilePath, ErrorType, Function}
// Output: count within time window

tracker := memory.DefaultFailureTracker()
key := memory.FailureKey{
    FilePath:  "test.py",
    ErrorType: "typeerror",
}

tracker.LogFailure(key, "Bash")
count, _ := tracker.CountRecentFailures(key)
// → 3 (if this is the 3rd typeerror on test.py)
```

**Responsibilities:**
- Persist failures to `~/.gogent/failure-tracker.jsonl`
- Count recent failures using composite key
- Sliding time window (default: 300s)
- Configurable threshold (default: 3)

**Does NOT:**
- Parse tool events (that's Layer 1)
- Write pending-learnings.jsonl (that's Layer 3)
- Generate hook responses (that's Layer 3)

---

### Layer 3: Orchestration (cmd/gogent-sharp-edge)

**Ticket**: 035
**Package**: `cmd/gogent-sharp-edge/main.go`
**Question**: "Wire everything together and produce hook output"

```go
// 1. Parse event (routing.ParsePostToolEvent)
// 2. Detect failure (routing.DetectFailure)
// 3. Track failure (memory.LogFailure + CountRecentFailures)
// 4. Decide hook action (block/warn/pass)
// 5. Write pending-learning if threshold reached
// 6. Return hook JSON
```

**Responsibilities:**
- Read STDIN (PostToolUse event)
- Coordinate detection + tracking layers
- Decide threshold behavior
- Write `pending-learnings.jsonl` (SharpEdge schema)
- Output hook-compliant JSON

**Does NOT:**
- Implement detection logic (imports pkg/routing)
- Implement tracking logic (imports pkg/memory)

---

## Zero Overlap Verification

| Feature | Layer 1 (034) | Layer 2 (036) | Layer 3 (035) |
|---------|---------------|---------------|---------------|
| **Detect failures** | ✅ | - | - |
| **Track consecutive** | - | ✅ | - |
| **Composite key** | - | ✅ | - |
| **Write tracker log** | - | ✅ | - |
| **Parse events** | ✅ | - | Uses Layer 1 |
| **Write pending-learnings** | - | - | ✅ |
| **Hook output** | - | - | ✅ |

**No duplication** ✅

---

## Schema Alignment

### Internal: FailureTracker Log

`~/.gogent/failure-tracker.jsonl`:
```jsonl
{"timestamp":1705708800,"file":"test.py","tool":"Bash","error_type":"typeerror"}
```

### Output: Pending Learnings

`.claude/memory/pending-learnings.jsonl`:
```jsonl
{"file":"test.py","error_type":"typeerror","consecutive_failures":3,"timestamp":1705708800,"context":"Tool: Bash"}
```

Matches `pkg/session.SharpEdge` struct:
- ✅ `file` (string)
- ✅ `error_type` (string)
- ✅ `consecutive_failures` (int, ≥3)
- ✅ `timestamp` (Unix epoch int64)
- ✅ `context` (string, optional)

---

## Environment Variables (Standardized)

| Variable | Default | Used By | Purpose |
|----------|---------|---------|---------|
| `GOGENT_MAX_FAILURES` | 3 | pkg/memory | Threshold for capture |
| `GOGENT_FAILURE_WINDOW` | 300 | pkg/memory | Time window (seconds) |
| `GOGENT_PROJECT_DIR` | cwd | cmd/gogent-sharp-edge | Project root |

**Prefix**: `GOGENT_*` (not `CLAUDE_*`) - this is GOgent-Fortress, not Claude Code

---

## Storage Paths (Standardized)

| File | Location | Purpose |
|------|----------|---------|
| Failure tracker | `~/.gogent/failure-tracker.jsonl` | Internal tracking state |
| Pending learnings | `.claude/memory/pending-learnings.jsonl` | Sharp edges awaiting archive |

**Why `~/.gogent/`:**
- Persists across reboots (unlike `/tmp`)
- User-specific
- Follows XDG conventions
- Dedicated namespace for GOgent suite

---

## Composite Key Prevents False Positives

### Without Composite Key (File-Only)

```
File: main.py
- Line 10: TypeError (attempt 1)
- Line 50: ImportError (attempt 2)
- Line 100: ValueError (attempt 3)
→ 3 failures on main.py → SHARP EDGE TRIGGERED ❌ (false positive)
```

### With Composite Key (File + Error)

```
File: main.py, Error: TypeError → Count: 1
File: main.py, Error: ImportError → Count: 1
File: main.py, Error: ValueError → Count: 1
→ No threshold reached ✅ (correct)

File: main.py, Error: TypeError (3 consecutive attempts)
→ SHARP EDGE TRIGGERED ✅ (correct - genuine debugging loop)
```

---

## Simulation Test Coverage

| Test | Purpose | Validates |
|------|---------|-----------|
| **F001_single_failure** | 1 failure → no capture | Threshold = 3 works |
| **F002_threshold_reached** | 3 same errors → capture | Composite key + threshold |
| **F003_mixed_errors** | 3 different errors → no capture | Composite key prevents false positive |
| **F004_schema_compliance** | Output passes `ValidateSharpEdge()` | Schema alignment |

---

## Implementation Order

1. **034** → Failure detection logic
   - Time: 2.0h
   - Deliverable: `pkg/routing/failure.go`

2. **036** → Failure tracking logic (includes composite key)
   - Time: 2.5h
   - Deliverable: `pkg/memory/failure_tracking.go`
   - Depends on: None (standalone package)

3. **035** → CLI orchestration
   - Time: 2.5h
   - Deliverable: `cmd/gogent-sharp-edge/main.go` + simulation tests
   - Depends on: 034, 036

4. **036c** (OPTIONAL) → Function extraction
   - Time: 0.5h
   - Deliverable: `ExtractFunctionFromStackTrace()` in `pkg/memory/`
   - Depends on: 036
   - Can be deferred

**Total**: 7.5 hours (7.0h without optional 036c)

---

## CI/CD Integration

### Makefile Targets

```makefile
build-sharp-edge:
	go build -o bin/gogent-sharp-edge ./cmd/gogent-sharp-edge

test-simulation-posttooluse:
	go run ./test/simulation/harness/cmd/harness \
		-mode=deterministic \
		-scenarios=posttooluse \
		-report=tap
```

### GitHub Actions

```yaml
- name: Build CLIs
  run: make build-validate build-archive build-sharp-edge

- name: Run Simulation Tests
  run: |
    go run ./test/simulation/harness/cmd/harness \
      -mode=deterministic \
      -scenarios=pretooluse,sessionend,posttooluse \
      -report=json
```

---

## Post-Implementation Verification

### Step 1: Unit Tests

```bash
go test ./pkg/routing -run Failure    # 034
go test ./pkg/memory -run Failure     # 036
go test ./cmd/gogent-sharp-edge       # 035
```

### Step 2: Simulation Tests

```bash
make test-simulation
# Should show:
# - V001-V008 (validate) ✅
# - S001-S008 (sessionend) ✅
# - F001-F004 (posttooluse) ✅ (NEW)
```

### Step 3: Integration Test

```bash
# Install
make install

# Test manually
echo '{"tool_name":"Bash","tool_input":{"command":"python test.py"},"tool_response":{"exit_code":1,"output":"TypeError"},"session_id":"test","hook_event_name":"PostToolUse","captured_at":1705708800}' | gogent-sharp-edge

# Should output: {} (first failure, below threshold)

# Run 2 more times to hit threshold...
# Third run should output block response + write pending-learnings.jsonl
```

### Step 4: Hook Integration

Update `~/.claude/settings.json`:
```json
"PostToolUse": [
  {
    "matcher": "Bash|Edit|Write|Task",
    "hooks": [
      {
        "type": "command",
        "command": "gogent-sharp-edge",
        "timeout": 5
      }
    ]
  }
]
```

### Step 5: End-to-End Validation

1. Trigger 3 failures on same file with same error
2. Verify block message appears
3. Check `~/.gogent/failure-tracker.jsonl` has entries
4. Check `.claude/memory/pending-learnings.jsonl` has SharpEdge
5. Run `gogent-archive` at session end
6. Verify handoff includes sharp edge
7. Verify archive has `learnings-{timestamp}.jsonl`

---

## Cleanup After Implementation

```bash
# Remove old bash script
mv ~/.claude/hooks/sharp-edge-detector.sh ~/.claude/hooks/sharp-edge-detector.sh.bak

# Verify no references remain
grep -r "sharp-edge-detector.sh" ~/.claude/
# (should find only .bak file)
```

---

## Migration from Original 036 Series

| Original | Unified | Status |
|----------|---------|--------|
| GOgent-036 | GOgent-036 | **REPLACED** - Now includes composite key from start |
| GOgent-036b | (merged into 036) | **OBSOLETE** - Functionality absorbed |
| GOgent-036c | GOgent-036c | **KEPT** - Optional function extraction |

**Key Change**: Original 036 series had 3 tickets with incremental refinement. Unified approach implements composite key from the start (036), with optional function extraction as 036c.

---

## Architecture Principles

1. **Separation of Concerns**
   - Detection ≠ Tracking ≠ Orchestration
   - Each layer has ONE job

2. **Composability**
   - pkg/routing and pkg/memory are libraries
   - Can be used by other CLIs
   - CLI is thin orchestration layer

3. **Testability**
   - Pure functions in pkg/
   - Integration tests via simulation harness
   - Unit tests for each layer

4. **Schema Consistency**
   - One canonical format (SharpEdge)
   - Validation enforced (ValidateSharpEdge)
   - Simulation tests catch drift

5. **Zero Duplication**
   - No overlapping implementations
   - Each feature lives in exactly ONE place

---

## Questions & Answers

**Q: Why not keep tracker in cmd/gogent-sharp-edge/?**
A: Violates composability. If we add `gogent-tui` later, it would need tracking too. pkg/memory is reusable.

**Q: Why composite key instead of file-only?**
A: Prevents false positives. 3 different errors on one file ≠ debugging loop.

**Q: Why not include function from the start?**
A: Function extraction is heuristic and fragile. File+Error is deterministic. 036c adds function as optional refinement.

**Q: Why `~/.gogent/` instead of `/tmp/`?**
A: Persistence. Reboot wipes `/tmp`, but failures across sessions are valuable context.

**Q: Why GOGENT_* prefix instead of CLAUDE_*?**
A: This is GOgent-Fortress, not Claude Code core. Clear namespace ownership.

---

## Einstein Review (2026-01-23)

### Issues Identified and Resolved

| Issue | Severity | Resolution |
|-------|----------|------------|
| `pkg/memory/` package missing | 🔴 Critical | Created `pkg/memory/doc.go` with package documentation |
| `formatExitCode()` bug in 034 | 🔴 Critical | Fixed to use `fmt.Sprintf("exit_code_%d", code)` with semantic mappings |
| Simulation harness missing posttooluse | 🔴 Critical | Added explicit harness update requirements to 035 ticket |
| 035 dependency missing 036 | 🟡 Moderate | Updated frontmatter: `dependencies: [GOgent-034-R, GOgent-036]` |
| Extra closing brace in 035 main.go | 🟡 Moderate | Removed stray `}` from code sample |

### Verification Matrix

| Component | Status | Notes |
|-----------|--------|-------|
| `pkg/memory/doc.go` | ✅ Created | Package now exists |
| 034 formatExitCode | ✅ Fixed | Returns semantic names + numeric fallback |
| 035 dependencies | ✅ Fixed | Both 034 and 036 listed |
| 035 harness requirements | ✅ Added | Explicit code changes documented |
| Schema alignment | ✅ Verified | All use `timestamp` (not `ts`) |
| Env var prefix | ✅ Verified | All use `GOGENT_*` |

### Implementation Ready

All critical issues resolved. Safe to proceed with implementation.

---

**Status**: Ready for implementation ✅
**Next Step**: Implement tickets in order: 034 → 036 → 035 → 036c (optional)
**Einstein Review**: Complete (2026-01-23)
