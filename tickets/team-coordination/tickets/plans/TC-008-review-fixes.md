# TC-008 Review Fixes Addendum

**Generated**: 2026-02-07
**Source**: Parallel code review (backend, standards, architect)
**Status**: Approved for implementation

---

## Fix Index

| ID | Severity | Fix | Phase |
|----|----------|-----|-------|
| C1 | CRITICAL | Budget floor enforcement | Phase 1 (config.go) |
| C2 | CRITICAL | Unexport budgetRemainingUSD | Phase 1 (config.go) |
| C3 | CRITICAL | Cost extraction: missing = error, use estimate fallback | Phase 1 (cost.go) |
| C4 | CRITICAL | Serialize config writes with ordering mutex | Phase 1 (config.go) |
| C5 | CRITICAL | Reference TC-009 schemas in envelope builder | Phase 1 (envelope.go) |
| C6 | CRITICAL | Heartbeat interval = 10s (shared constant) | Phase 1 (heartbeat.go) |
| W1 | WARNING | Decompose Spawn() into 3 functions | Phase 2 (spawn.go) |
| W2 | WARNING | Path traversal protection for stdout/stdin | Phase 1 (validate.go) |
| W3 | WARNING | Validate empty task/context in envelope | Phase 1 (envelope.go) |
| W4 | WARNING | Least-privilege fallback tools (Read,Glob,Grep) | Phase 2 (spawn.go) |
| W5 | WARNING | Batch member status updates where possible | Phase 2 (spawn.go) |
| W6 | WARNING | Signal-safe PID registration | Phase 2 (spawn.go) |
| W7 | WARNING | Error classification: retryable vs fatal | Phase 2 (spawn.go) |

---

## Phase 1 Fixes (New Files + config.go Modifications)

### C1 + C2 + C4: Budget Safety (config.go)

**What**: Budget mutations are currently unprotected from direct access and have no write ordering.

**Changes to config.go**:

1. **Unexport `BudgetRemainingUSD`** in TeamConfig struct:
   ```go
   // BEFORE:
   BudgetRemainingUSD float64 `json:"budget_remaining_usd"`

   // AFTER: Keep JSON tag for serialization, but add accessor methods
   // NOTE: Because json.Marshal needs exported fields, keep exported BUT
   // add a comment-level prohibition + provide accessor methods.
   // True unexport breaks JSON marshaling, so use accessor pattern instead.
   ```

   **Practical approach**: Keep field exported (JSON needs it), but:
   - Add `BudgetRemaining() float64` read accessor (RLock)
   - Add comment: `// DO NOT mutate directly. Use tryReserveBudget()/reconcileCost() ONLY.`
   - Add compile-time test that greps for direct budget mutations

2. **Add write-ordering mutex** to TeamRunner:
   ```go
   type TeamRunner struct {
       // ... existing fields ...
       writeMu  sync.Mutex  // Serializes config writes (C4)
   }
   ```

3. **Update `updateMember()` and `SaveConfig()`** to acquire writeMu:
   ```go
   func (tr *TeamRunner) updateMember(waveIdx, memIdx int, fn func(*Member)) error {
       tr.writeMu.Lock()         // C4: serialize writes
       defer tr.writeMu.Unlock()

       tr.configMu.Lock()
       fn(&tr.config.Waves[waveIdx].Members[memIdx])
       configCopy := deepCopyConfig(tr.config)
       tr.configMu.Unlock()

       // Write to disk (serialized by writeMu)
       return writeConfigAtomic(tr.configPath, &configCopy)
   }
   ```

4. **Add floor enforcement** to `reconcileCost()`:
   ```go
   func (tr *TeamRunner) reconcileCost(estimated, actual float64) {
       tr.writeMu.Lock()
       defer tr.writeMu.Unlock()

       tr.configMu.Lock()
       tr.config.BudgetRemainingUSD += estimated
       tr.config.BudgetRemainingUSD -= actual
       if tr.config.BudgetRemainingUSD < 0 {
           log.Printf("[CRITICAL] Budget went negative ($%.4f), clamping to $0.00", tr.config.BudgetRemainingUSD)
           tr.config.BudgetRemainingUSD = 0
       }
       configCopy := deepCopyConfig(tr.config)
       tr.configMu.Unlock()

       writeConfigAtomic(tr.configPath, &configCopy)
   }
   ```

**Test Requirements (config_test.go)**:
- `TestBudgetFloorEnforcement`: reconcileCost with actual > estimated + remaining, verify budget = 0
- `TestBudgetNeverNegative_Concurrent`: 10 goroutines doing tryReserveBudget + reconcileCost, assert budget >= 0 always
- `TestWriteOrdering_NoLostUpdates`: 20 concurrent updateMember calls, verify ALL mutations present in final config
- `TestBudgetAccessor`: BudgetRemaining() returns correct value under concurrent mutations
- `TestBudgetDirectMutationProhibited`: grep source for direct `BudgetRemainingUSD =` outside allowed functions

**Success Criteria**:
- [ ] `go test -race` passes with 0 warnings on all budget tests
- [ ] Budget never goes negative in 100-iteration concurrent test
- [ ] No lost writes in 20-concurrent-update test
- [ ] Source scan test catches any direct budget mutation

---

### C3: Cost Extraction Safety (cost.go)

**What**: Missing cost field should be treated as error, not warning. Use estimated cost as conservative fallback.

**Implementation**:
```go
type CostResult struct {
    Cost   float64
    Status CostStatus
    Err    error
}

type CostStatus string

const (
    CostOK      CostStatus = "ok"       // Extracted successfully
    CostFallback CostStatus = "fallback" // Used estimated cost (missing field)
    CostError   CostStatus = "error"    // JSON parse failed
)

func extractCostFromCLIOutput(output []byte) CostResult {
    var result map[string]interface{}
    if err := json.Unmarshal(output, &result); err != nil {
        return CostResult{
            Cost:   0,
            Status: CostError,
            Err:    fmt.Errorf("CLI output not valid JSON: %w", err),
        }
    }

    for _, field := range []string{"cost_usd", "total_cost_usd", "usage.cost_usd"} {
        if val, ok := getNestedFloat(result, field); ok {
            return CostResult{Cost: val, Status: CostOK}
        }
    }

    // C3 FIX: Missing field = error, caller uses estimated cost as fallback
    return CostResult{
        Cost:   0,
        Status: CostFallback,
        Err:    fmt.Errorf("no cost field found in CLI output (checked: cost_usd, total_cost_usd, usage.cost_usd)"),
    }
}
```

**Caller side** (in spawn.go):
```go
costResult := extractCostFromCLIOutput(stdoutBuf.Bytes())
actualCost := costResult.Cost
if costResult.Status == CostFallback {
    log.Printf("[WARN] cost: %v for %s — using estimated cost $%.2f as fallback",
        costResult.Err, member.Name, estimatedCost)
    actualCost = estimatedCost  // Conservative: assume full estimate
}
```

**Test Requirements (cost_test.go)**:
- `TestExtractCost_TopLevel`: `{"cost_usd": 2.45}` → CostOK, 2.45
- `TestExtractCost_TotalField`: `{"total_cost_usd": 1.80}` → CostOK, 1.80
- `TestExtractCost_Nested`: `{"usage": {"cost_usd": 0.50}}` → CostOK, 0.50
- `TestExtractCost_MissingField`: `{"status": "ok"}` → CostFallback, 0, non-nil error
- `TestExtractCost_InvalidJSON`: `not json` → CostError, 0, non-nil error
- `TestExtractCost_EmptyObject`: `{}` → CostFallback
- `TestExtractCost_NegativeValue`: `{"cost_usd": -5.0}` → CostOK, -5.0 (caller validates)
- `TestExtractCost_LargeValue`: `{"cost_usd": 999.99}` → CostOK, 999.99
- `TestExtractCost_ZeroCost`: `{"cost_usd": 0}` → CostOK, 0
- `TestExtractCost_FloatPrecision`: `{"cost_usd": 0.000001}` → CostOK, 0.000001
- `TestGetNestedFloat_DeepPath`: 3-level nesting works
- `TestGetNestedFloat_MissingIntermediate`: partial path returns false

**Success Criteria**:
- [ ] All 12 test cases pass
- [ ] CostFallback status returned when no cost field found
- [ ] CostError status returned when JSON parse fails
- [ ] Caller receives status to decide fallback behavior

---

### C5 + W3: Envelope Builder (envelope.go)

**What**: Build prompt from stdin JSON, validate required fields, reference TC-009 schemas.

**Prerequisites**: Read these schemas before implementing:
- `.claude/schemas/teams/stdin-stdout/` (all files in directory)
- `.claude/schemas/teams/team-config.json`

**Implementation**:
```go
type StdinEnvelope struct {
    Schema      string            `json:"$schema"`
    Task        string            `json:"task"`
    Context     string            `json:"context"`
    Constraints []string          `json:"constraints,omitempty"`
    Files       map[string]string `json:"files,omitempty"`
    // Additional fields discovered from TC-009 schemas
}

func buildPromptEnvelope(teamDir string, member *Member) (string, error) {
    // 1. Validate stdin path is within teamDir (W2 path traversal)
    stdinPath := filepath.Join(teamDir, member.StdinFile)
    if err := validatePathWithinDir(stdinPath, teamDir); err != nil {
        return "", fmt.Errorf("stdin path security: %w", err)
    }

    // 2. Read and parse stdin
    stdinData, err := os.ReadFile(stdinPath)
    if err != nil {
        return "", fmt.Errorf("read stdin file %s: %w", member.StdinFile, err)
    }

    var stdin StdinEnvelope
    if err := json.Unmarshal(stdinData, &stdin); err != nil {
        return "", fmt.Errorf("parse stdin JSON: %w", err)
    }

    // 3. W3: Validate required fields
    if stdin.Task == "" {
        return "", fmt.Errorf("stdin: task field is empty")
    }
    if stdin.Context == "" {
        return "", fmt.Errorf("stdin: context field is empty")
    }

    // 4. Build envelope with capabilities notice (TC-007 nesting level)
    envelope := fmt.Sprintf(...)
    return envelope, nil
}

// Shared path traversal validator (used by envelope + validate)
func validatePathWithinDir(targetPath, baseDir string) error {
    absTarget, err := filepath.Abs(targetPath)
    if err != nil {
        return fmt.Errorf("resolve target path: %w", err)
    }
    absBase, err := filepath.Abs(baseDir)
    if err != nil {
        return fmt.Errorf("resolve base dir: %w", err)
    }
    if !strings.HasPrefix(absTarget, absBase+string(filepath.Separator)) && absTarget != absBase {
        return fmt.Errorf("path %s escapes base directory %s", targetPath, baseDir)
    }
    return nil
}
```

**Test Requirements (envelope_test.go)**:
- `TestBuildEnvelope_ValidStdin`: full valid stdin → correct envelope format
- `TestBuildEnvelope_EmptyTask`: task="" → error
- `TestBuildEnvelope_EmptyContext`: context="" → error
- `TestBuildEnvelope_MissingFile`: stdin file doesn't exist → error
- `TestBuildEnvelope_InvalidJSON`: bad JSON → error
- `TestBuildEnvelope_PathTraversal`: `"../../../etc/passwd"` → error
- `TestBuildEnvelope_CapabilitiesNotice`: output contains nesting level notice
- `TestBuildEnvelope_AgentName`: output contains correct AGENT: line
- `TestValidatePathWithinDir`: 5+ cases (relative, absolute, traversal, symlink)

**Success Criteria**:
- [ ] Path traversal blocked for both stdin and stdout paths
- [ ] Empty task/context rejected with clear error
- [ ] Capabilities notice included in every envelope
- [ ] All TC-009 stdin fields handled (read schemas first)

---

### W2: Stdout Validation (validate.go)

**What**: Validate stdout JSON envelope with path traversal protection.

**Implementation**:
```go
func validateStdout(stdoutPath string, teamDir string) error {
    // W2: Path traversal protection
    if err := validatePathWithinDir(stdoutPath, teamDir); err != nil {
        return fmt.Errorf("stdout path security: %w", err)
    }

    data, err := os.ReadFile(stdoutPath)
    if err != nil {
        return fmt.Errorf("read stdout file: %w", err)
    }

    if len(data) == 0 {
        return fmt.Errorf("stdout file is empty")
    }

    var stdout StdoutEnvelope
    if err := json.Unmarshal(data, &stdout); err != nil {
        return fmt.Errorf("parse stdout JSON: %w", err)
    }

    if stdout.Schema == "" {
        return fmt.Errorf("missing $schema field in stdout")
    }
    if stdout.Status == "" {
        return fmt.Errorf("missing status field in stdout")
    }

    return nil
}

type StdoutEnvelope struct {
    Schema  string                 `json:"$schema"`
    Status  string                 `json:"status"`
    Content map[string]interface{} `json:"content"`
}
```

**Test Requirements (validate_test.go)**:
- `TestValidateStdout_Valid`: all fields present → nil
- `TestValidateStdout_MissingSchema`: no $schema → error
- `TestValidateStdout_MissingStatus`: no status → error
- `TestValidateStdout_InvalidJSON`: bad JSON → error
- `TestValidateStdout_EmptyFile`: 0 bytes → error
- `TestValidateStdout_FileNotFound`: missing file → error
- `TestValidateStdout_PathTraversal`: "../../../" → error
- `TestValidateStdout_ExtraFields`: extra fields allowed → nil

**Success Criteria**:
- [ ] Path traversal blocked
- [ ] All required fields validated
- [ ] Extra fields don't cause errors

---

### C6: Heartbeat (heartbeat.go)

**What**: 10-second heartbeat (per TC-012, not 30s from original spec).

**Implementation**:
```go
const HeartbeatInterval = 10 * time.Second  // C6: Per TC-012 requirement (was 30s)

func (tr *TeamRunner) startHeartbeat(ctx context.Context) {
    heartbeatPath := filepath.Join(tr.teamDir, "heartbeat")

    // Touch immediately on start
    writeHeartbeat(heartbeatPath)

    go func() {
        ticker := time.NewTicker(HeartbeatInterval)
        defer ticker.Stop()

        for {
            select {
            case <-ticker.C:
                writeHeartbeat(heartbeatPath)
            case <-ctx.Done():
                return
            }
        }
    }()
}

func writeHeartbeat(path string) {
    now := time.Now().Unix()
    os.WriteFile(path, []byte(fmt.Sprintf("%d\n", now)), 0644)
}
```

**Test Requirements (heartbeat_test.go)**:
- `TestHeartbeat_FileCreated`: heartbeat file exists after start
- `TestHeartbeat_PeriodicTouch`: mtime updates within expected interval (use 100ms ticker for tests)
- `TestHeartbeat_ContextCancellation`: goroutine stops when ctx cancelled
- `TestHeartbeat_Interval`: verify HeartbeatInterval constant = 10s

**Success Criteria**:
- [ ] HeartbeatInterval = 10 * time.Second (shared constant)
- [ ] File touched immediately on start
- [ ] Goroutine exits cleanly on context cancellation

---

## Phase 2 Fixes (spawn.go + wave.go)

### W1 + W4 + W5 + W6 + W7: Spawn Decomposition (spawn.go)

**What**: Decompose 17-step Spawn() into 3 focused functions. Add error classification, least-privilege fallback, signal-safe PID registration.

**Decomposition**:
```go
// Step 1: Prepare all inputs (no side effects)
func (s *claudeSpawner) prepareSpawn(tr *TeamRunner, waveIdx, memIdx int) (*spawnConfig, error)

// Step 2: Execute process (side effects: fork, register PID)
func (s *claudeSpawner) executeSpawn(ctx context.Context, tr *TeamRunner, cfg *spawnConfig) (*spawnResult, error)

// Step 3: Process results (cost extraction, validation, member update)
func (s *claudeSpawner) finalizeSpawn(tr *TeamRunner, waveIdx, memIdx int, result *spawnResult, estimated float64) error

// Public entry point delegates to the three phases
func (s *claudeSpawner) Spawn(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error {
    cfg, err := s.prepareSpawn(tr, waveIdx, memIdx)
    if err != nil {
        return fmt.Errorf("prepare: %w", err)
    }

    result, err := s.executeSpawn(ctx, tr, cfg)
    if err != nil {
        return fmt.Errorf("execute: %w", err)
    }

    return s.finalizeSpawn(tr, waveIdx, memIdx, result, cfg.estimatedCost)
}
```

**W4: Least-privilege fallback**:
```go
// Default fallback is READ-ONLY when agents-index.json unavailable
var defaultFallbackTools = []string{"Read", "Glob", "Grep"}
```

**W7: Error classification**:
```go
func isRetryableError(err error) bool {
    if err == nil { return false }
    if errors.Is(err, context.DeadlineExceeded) { return true }    // Timeout: retry
    if errors.Is(err, context.Canceled) { return false }           // Cancelled: stop
    var exitErr *exec.ExitError
    if errors.As(err, &exitErr) { return true }                    // Non-zero exit: retry
    if errors.Is(err, exec.ErrNotFound) { return false }           // CLI missing: fatal
    if errors.Is(err, os.ErrPermission) { return false }           // Permission: fatal
    return true  // Default: retry (conservative)
}
```

**W5: Reduce status updates** — batch into prepareSpawn (set running) and finalizeSpawn (set completed/failed).

**W6: Signal-safe PID** — registerChild immediately after cmd.Start() succeeds, before any other work.

**Test Requirements**:
- `TestPrepareSpawn_ValidConfig`: produces correct spawnConfig
- `TestPrepareSpawn_MissingStdin`: returns error
- `TestPrepareSpawn_FallbackTools`: when agent config unavailable, uses Read,Glob,Grep only
- `TestExecuteSpawn_MockCLI`: bash script mock → success
- `TestExecuteSpawn_Timeout`: context deadline → retryable error
- `TestExecuteSpawn_CLINotFound`: missing binary → non-retryable
- `TestFinalizeSpawn_CostOK`: cost extracted, member updated
- `TestFinalizeSpawn_CostFallback`: missing cost → uses estimated
- `TestFinalizeSpawn_ValidationFail`: invalid stdout → error
- `TestIsRetryableError`: table-driven with all error types
- `TestSpawn_FullLifecycle_MockCLI`: prepare → execute → finalize end-to-end

**Success Criteria**:
- [ ] Spawn() is 3 functions, each independently testable
- [ ] Non-retryable errors (ErrNotFound, ErrPermission) fail immediately without retry
- [ ] Fallback tools are Read, Glob, Grep (read-only)
- [ ] PID registered before any post-Start() work
- [ ] Member status updated max 2x per attempt (running, completed/failed)

---

### Wave Scheduler (wave.go)

**What**: Sequential waves, parallel members, budget gates using safe budget functions.

**Key Rule**: ALL budget access via `tryReserveBudget()` / `reconcileCost()` / `BudgetRemaining()`. Never touch `BudgetRemainingUSD` directly.

**Implementation follows original plan** with these additions:
- `runInterWaveScript()` stubs with log + TODO for TC-010
- Budget gate uses `tryReserveBudget()` (C1/C2 safe functions)
- Wave completion logs total cost for the wave

**Test Requirements (wave_test.go)**:
- `TestRunWaves_SingleWave`: 1 wave, 2 members → both complete
- `TestRunWaves_Sequential`: wave 2 starts only after wave 1 completes
- `TestRunWaves_ParallelMembers`: members within wave run concurrently
- `TestRunWaves_BudgetExhaustion`: $0.50 budget, 5 members → only 1-2 spawn
- `TestRunWaves_BudgetGate`: insufficient budget → member skipped with log
- `TestRunWaves_ContextCancellation`: cancel mid-wave → graceful stop
- `TestRunWaves_InterWaveScript`: stub script runs between waves
- `TestRunWaves_EmptyWave`: wave with 0 members → skip

**Success Criteria**:
- [ ] Waves execute sequentially (verified with execution ordering)
- [ ] Members within wave execute in parallel (verified with timing)
- [ ] Budget gate prevents spawns when budget insufficient
- [ ] Budget never goes negative across concurrent wave execution

---

## Phase 3: Integration (main.go)

**What**: Wire runWaves(), heartbeat, agent validation into main.go.

**Changes to main.go**:
1. Replace `runWaves()` placeholder with real call
2. Add `startHeartbeat(ctx)` after daemon setup
3. Add `validateAgents(config)` before wave execution
4. Add `runner.cleanupOrphanProcesses()` in shutdown path

**Test Requirements (main_test.go)**:
- `TestMain_FullExecution_MockCLI`: 2-wave config, mock agents → success
- `TestMain_BudgetCeiling`: $0.50 budget → stops early
- `TestMain_SignalHandling`: SIGTERM → clean shutdown within 10s
- `TestMain_HeartbeatIntegration`: heartbeat file exists during execution

**Success Criteria**:
- [ ] `go build ./cmd/gogent-team-run` exits 0
- [ ] Mock CLI integration test passes
- [ ] SIGTERM kills all children within 10s
- [ ] Heartbeat file touched every 10s during execution

---

## Quality Gates (Final)

- [ ] `go test ./cmd/gogent-team-run/...` — all pass
- [ ] `go test -race ./cmd/gogent-team-run/...` — 0 warnings
- [ ] `go vet ./cmd/gogent-team-run/...` — clean
- [ ] `golangci-lint run ./cmd/gogent-team-run/...` — 0 errors
- [ ] Coverage >= 80% overall, >= 95% for budget/concurrency code
