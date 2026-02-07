# TC-008 Implementation Plan: gogent-team-run Go Binary Completion

**Status**: Ready for Implementation
**Estimated Effort**: 5-7 days
**Phase**: Phase 2: Go Binary Implementation
**Priority**: CRITICAL

---

## Executive Summary

Complete the `gogent-team-run` Go binary by implementing 5 new files and modifying 2 existing files. The binary orchestrates background multi-agent execution with wave-based coordination, budget gates, and retry logic. All infrastructure (daemon, config, spawn framework) is proven through existing tests.

**Key Achievement**: Transform the placeholder main loop into a production-ready team orchestrator.

---

## Current State Analysis

### Completed Components (Proven via Tests)

| File | Lines | Status | Test Coverage |
|------|-------|--------|---------------|
| `daemon.go` | 274 | ✅ COMPLETE | 317 lines (daemon_test.go) |
| `config.go` | 248 | ✅ COMPLETE | Tested via spawn_test.go |
| `spawn.go` | 119 | ⚠️ PARTIAL | 551 lines (spawn_test.go) |
| `main.go` | 159 | ⚠️ PARTIAL | N/A (integration tests) |

**What Works Now**:
- PID file acquisition with double-start prevention
- Signal cascade (SIGTERM → SIGKILL) to child process groups
- Atomic config writes (write-tmp-rename pattern with unique suffixes)
- Mutex-protected config access via `updateMember()`
- Iterative retry loop with error history accumulation
- Child process registration/tracking
- Output redirection to runner.log
- Stdin daemonization

**What's Stubbed**:
1. `spawn.go:22` - `claudeSpawner.Spawn()` returns "not yet implemented"
2. `main.go:135-159` - `runWaves()` is a 5-minute sleep placeholder

### Missing Components

**5 New Files** (0 lines → ~600 lines total):
1. `wave.go` - Wave scheduler (~120 lines)
2. `envelope.go` - Prompt envelope builder (~80 lines)
3. `cost.go` - Cost extraction from CLI output (~90 lines)
4. `validate.go` - Stdout envelope validation (~60 lines)
5. `heartbeat.go` - Background heartbeat loop (~40 lines)

**6 New Test Files** (~800 lines total):
1. `wave_test.go` - Wave execution tests
2. `envelope_test.go` - Envelope building tests
3. `cost_test.go` - Cost extraction tests
4. `validate_test.go` - Validation tests
5. `heartbeat_test.go` - Heartbeat tests
6. `main_test.go` - End-to-end integration

**2 Modifications**:
1. `spawn.go` - Implement `claudeSpawner.Spawn()` (~150 lines)
2. `main.go` - Wire up `runWaves()`, heartbeat, signal handling (~50 lines modification)

---

## Design Decisions

### 1. Budget Management Pattern

**Decision**: Two-phase budget pattern (reserve → reconcile) with mutex-protected operations.

**Rationale**: 
- Prevents race conditions during parallel spawns
- Conservative estimation prevents budget overrun
- Actual cost reconciliation ensures accurate tracking

**Implementation**:
```go
// Phase 1: Reserve budget before spawn (mutex-protected)
func (tr *TeamRunner) tryReserveBudget(estimatedCost float64) bool {
    tr.configMu.Lock()
    defer tr.configMu.Unlock()
    if tr.config.BudgetRemainingUSD < estimatedCost {
        return false  // Budget gate blocks spawn
    }
    tr.config.BudgetRemainingUSD -= estimatedCost
    tr.writeConfigAtomic()
    return true
}

// Phase 2: Reconcile after spawn completes (return reservation, deduct actual)
func (tr *TeamRunner) reconcileCost(estimatedCost, actualCost float64) {
    tr.configMu.Lock()
    defer tr.configMu.Unlock()
    tr.config.BudgetRemainingUSD += estimatedCost  // Return reservation
    tr.config.BudgetRemainingUSD -= actualCost     // Deduct actual
    tr.writeConfigAtomic()
}
```

**Budget Estimation** (conservative fallback):
```go
func estimateCost(agentID string) float64 {
    index := loadAgentsIndex()
    agent := index[agentID]
    
    // Model-based estimation
    defaults := map[string]float64{
        "opus": 1.50,    // 30K tokens at $0.05/1K
        "sonnet": 0.30,  // 30K tokens at $0.01/1K
        "haiku": 0.05,   // 30K tokens at $0.0015/1K
    }
    
    if cost, ok := defaults[agent.Model]; ok {
        return cost
    }
    return defaults["opus"]  // Conservative fallback
}
```

**Critical Review Finding** (Item 13): ALL budget mutations MUST go through these two functions. Direct `tr.configMu.Lock()` for budget changes is PROHIBITED.

### 2. Prompt Envelope Structure

**Decision**: Build envelopes from stdin JSON with agent capabilities notice.

**Template Structure**:
```
AGENT: {agent_id}

## Your Task

{task from stdin}

## Context

{context from stdin}

## Your Capabilities

You are spawned via gogent-team-run at nesting level 2.

**Available delegation**:
- ✅ Task(model: "haiku") - For mechanical tasks
- ✅ Task(model: "sonnet") - For focused analysis
- ❌ Task(model: "opus") - Blocked by gogent-validate

**Important**: Always specify model explicitly in Task() calls.

## Expected Output

Write structured JSON to {stdout_file} following the stdout schema.

## Constraints

- Use absolute paths only
- {additional constraints from stdin}
```

**Rationale**:
- Capabilities notice prevents accidental Task(opus) calls
- Explicit model requirement prevents CLI default (which may be opus)
- Structured output requirement enables downstream validation

### 3. Cost Extraction Strategy

**Decision**: Multi-field fallback with graduated error handling.

**Search Order**:
1. `cost_usd` (top-level)
2. `total_cost_usd` (alternate top-level)
3. `usage.cost_usd` (nested)

**Error Levels**:
- **error**: JSON parse failed, CLI output is corrupted
- **warning**: Valid JSON but no cost field (proceed with cost=0)

**Rationale**:
- Graduated severity prevents blocking on missing cost (non-critical)
- Multiple fields handle CLI output format variations
- Logs provide debugging context without crashing

**Implementation** (`cost.go`):
```go
type CostError struct {
    Level     string // "warn" | "error"
    Message   string
    RawOutput string // First 500 chars for debugging
}

func extractCostFromCLIOutput(output []byte) (float64, *CostError) {
    var result map[string]interface{}
    if err := json.Unmarshal(output, &result); err != nil {
        return 0, &CostError{
            Level:     "error",
            Message:   fmt.Sprintf("CLI output is not valid JSON: %v", err),
            RawOutput: string(output[:min(len(output), 500)]),
        }
    }
    
    // Try multiple fields
    for _, field := range []string{"cost_usd", "total_cost_usd", "usage.cost_usd"} {
        if val, ok := getNestedFloat(result, field); ok {
            return val, nil  // Success
        }
    }
    
    // No cost field found - warning, not error
    return 0, &CostError{
        Level:     "warn",
        Message:   "no cost field found in CLI JSON output",
        RawOutput: string(output[:min(len(output), 500)]),
    }
}
```

### 4. Wave Execution Pattern

**Decision**: Sequential waves, parallel members, budget gates before each spawn.

**Flow**:
```
for each wave:
    check_budget_ceiling()
    
    for each member (parallel):
        estimated = estimateCost(member.agent)
        if !tryReserveBudget(estimated):
            log_budget_gate_block()
            break  // Stop spawning this wave
        
        spawn_in_goroutine(member, estimated)
    
    wait_for_wave_completion()
    
    if wave.OnCompleteScript != "":
        run_inter_wave_script()
```

**Key Properties**:
- Members within a wave run concurrently (fan-out)
- Waves execute sequentially (Wave 2 waits for Wave 1)
- Budget check is atomic (mutex-protected)
- Early exit on budget exhaustion (preserves remaining members)

### 5. Claude CLI Invocation

**Decision**: Use `--allowedTools` from `agents-index.json` + belt-and-suspenders `--permission-mode delegate`.

**CLI Args Construction**:
```go
func buildCLIArgs(agentConfig *AgentConfig) []string {
    args := []string{
        "-p",  // Pipe mode (stdin/stdout)
        "--output-format", "json",  // For cost extraction
    }
    
    // Load allowed tools from agents-index.json (TC-014)
    if len(agentConfig.CLIFlags.AllowedTools) > 0 {
        args = append(args, "--allowed-tools", 
            strings.Join(agentConfig.CLIFlags.AllowedTools, ","))
    }
    
    // Additional flags from agent config
    args = append(args, agentConfig.CLIFlags.AdditionalFlags...)
    
    return args
}
```

**Environment Variables**:
```go
cmd.Env = append(os.Environ(),
    "GOGENT_NESTING_LEVEL=2",           // For Task() validation (TC-007)
    "GOGENT_PROJECT_ROOT=" + projectRoot, // From config (TC-006)
)
```

**Rationale**:
- `--allowedTools` provides primary enforcement
- `--permission-mode delegate` is redundant but harmless (belt-and-suspenders)
- Nesting level enables Task(haiku/sonnet) while blocking Task(opus)

### 6. Process Group Isolation

**Decision**: Use `Setsid: true` for each spawned Claude process.

**Implementation**:
```go
cmd.SysProcAttr = &syscall.SysProcAttr{
    Setsid: true,  // Create new session (becomes process group leader)
}
```

**Rationale**:
- Prevents terminal signals (Ctrl+C) from reaching children
- Enables process group kill (`kill -TERM -pid` sends to entire group)
- Matches proven pattern from daemon_test.go (integration_test.go:191)

### 7. Stdout Validation

**Decision**: Minimal schema validation (presence checks, not deep validation).

**Checks**:
1. File exists and is readable
2. Valid JSON
3. Required fields present: `$schema`, `status`

**Non-Goals**:
- Schema version checking (future work)
- Content validation (agent's responsibility)
- Field type validation (trust the agent)

**Rationale**:
- Validation failure triggers retry (agent gets another chance)
- Deep validation creates brittle coupling
- Agents are trusted to produce correct output

---

## Implementation Phases

### Phase 1: Core Wave Execution (Days 1-2)

**Objective**: Implement wave scheduler and integrate with existing spawn framework.

**Files**:
1. **`wave.go`** (~120 lines)
   - `runWaves(ctx, tr) error` - Main wave loop
   - `runWave(ctx, tr, waveIdx) error` - Single wave execution
   - `runInterWaveScript(scriptPath) error` - TC-010 hook point
   - Budget gate integration
   - Error aggregation

2. **`wave_test.go`** (~200 lines)
   - Sequential wave execution (Wave 2 waits for Wave 1)
   - Parallel member execution within wave
   - Budget gate blocking (mid-wave exhaustion)
   - Inter-wave script execution
   - Context cancellation propagation

**Integration Point**: Wire `runWaves()` into `main.go` (replace placeholder).

**Test Strategy**:
```go
// Test: Sequential waves
func TestRunWaves_Sequential(t *testing.T) {
    config := &TeamConfig{
        Waves: []Wave{
            {WaveNumber: 1, Members: []Member{{Name: "w1m1"}}},
            {WaveNumber: 2, Members: []Member{{Name: "w2m1"}}},
        },
    }
    
    // Track execution order
    var order []string
    spawner := &fakeSpawner{
        fn: func(ctx, tr, waveIdx, memIdx) error {
            order = append(order, tr.config.Waves[waveIdx].Members[memIdx].Name)
            return nil
        },
    }
    
    runWaves(ctx, runner)
    
    // Assert: w1m1 completes before w2m1 starts
    assert.Equal(t, []string{"w1m1", "w2m1"}, order)
}
```

**Risks**:
- Budget gate logic is complex (mutex + early exit)
- Inter-wave script execution is untested (TC-010 dependency)

**Mitigation**:
- Use table-driven tests with varied budget scenarios
- Stub inter-wave script (echo script that touches a file)

### Phase 2: Prompt Envelope & Cost Extraction (Days 2-3)

**Objective**: Build prompt envelopes from stdin JSON and extract cost from CLI output.

**Files**:
1. **`envelope.go`** (~80 lines)
   - `buildPromptEnvelope(teamDir, member) (string, error)`
   - Stdin JSON parsing
   - Capabilities notice injection
   - Template rendering

2. **`envelope_test.go`** (~150 lines)
   - Valid stdin parsing
   - Missing fields (should error)
   - Template rendering correctness
   - Capabilities notice presence

3. **`cost.go`** (~90 lines)
   - `extractCostFromCLIOutput(output []byte) (float64, *CostError)`
   - `getNestedFloat(m map[string]interface{}, path string) (float64, bool)`
   - Multi-field search
   - Graduated error levels

4. **`cost_test.go`** (~150 lines)
   - Valid cost extraction (all 3 field variants)
   - Invalid JSON (should error)
   - Missing cost field (should warn)
   - Nested field extraction

**Integration Point**: `claudeSpawner.Spawn()` uses both functions.

**Test Strategy**:
```go
// Test: Cost extraction with nested field
func TestExtractCost_NestedField(t *testing.T) {
    output := []byte(`{"usage": {"cost_usd": 2.45}, "status": "ok"}`)
    
    cost, err := extractCostFromCLIOutput(output)
    
    assert.Nil(t, err)
    assert.Equal(t, 2.45, cost)
}

// Test: Missing cost field (warning, not error)
func TestExtractCost_MissingField(t *testing.T) {
    output := []byte(`{"status": "ok", "result": "done"}`)
    
    cost, err := extractCostFromCLIOutput(output)
    
    assert.NotNil(t, err)
    assert.Equal(t, "warn", err.Level)
    assert.Equal(t, 0.0, cost)
}
```

**Risks**:
- Stdin schemas vary by workflow (braintrust vs review vs implementation)
- CLI output format may change

**Mitigation**:
- Read multiple stdin schema examples (done in Phase 0)
- Multi-field fallback for cost extraction
- Log raw output on error (first 500 chars)

### Phase 3: Stdout Validation & Heartbeat (Day 3)

**Objective**: Validate agent output and implement heartbeat file touch loop.

**Files**:
1. **`validate.go`** (~60 lines)
   - `validateStdout(stdoutPath string) error`
   - JSON parsing
   - Required field checks

2. **`validate_test.go`** (~100 lines)
   - Valid stdout (should pass)
   - Invalid JSON (should fail)
   - Missing required fields (should fail)

3. **`heartbeat.go`** (~40 lines)
   - `startHeartbeat(ctx, teamDir)`
   - 30-second ticker loop
   - Context cancellation

4. **`heartbeat_test.go`** (~80 lines)
   - Heartbeat file creation
   - Periodic touch (verify mtime updates)
   - Context cancellation stops loop

**Integration Point**: 
- `validateStdout()` called in `claudeSpawner.Spawn()` before marking success
- `startHeartbeat()` called in `main()` after PID file acquisition

**Test Strategy**:
```go
// Test: Heartbeat periodic touch
func TestHeartbeat_PeriodicTouch(t *testing.T) {
    teamDir := t.TempDir()
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    startHeartbeat(ctx, teamDir)
    
    // Wait for multiple beats
    time.Sleep(2 * time.Second)
    
    // Verify file exists and was touched recently
    heartbeatPath := filepath.Join(teamDir, "heartbeat")
    info, err := os.Stat(heartbeatPath)
    require.NoError(t, err)
    assert.WithinDuration(t, time.Now(), info.ModTime(), 1*time.Second)
}
```

**Risks**:
- Heartbeat goroutine may not stop cleanly on context cancellation

**Mitigation**:
- Test context cancellation explicitly
- Use `select` with `ctx.Done()` in ticker loop

### Phase 4: Claude CLI Spawning (Days 4-5)

**Objective**: Implement `claudeSpawner.Spawn()` with full lifecycle (spawn → wait → cost → validate).

**Files**:
1. **`spawn.go` (MODIFY)** - Replace stub at line 22 with ~150 lines

**Implementation**:
```go
func (s *claudeSpawner) Spawn(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error {
    // 1. Load member config snapshot
    tr.configMu.RLock()
    member := tr.config.Waves[waveIdx].Members[memIdx]
    projectRoot := tr.config.ProjectRoot
    tr.configMu.RUnlock()
    
    // 2. Build prompt envelope
    envelope, err := buildPromptEnvelope(tr.teamDir, &member)
    if err != nil {
        return fmt.Errorf("build envelope: %w", err)
    }
    
    // 3. Load agent config for CLI flags
    agentConfig, err := loadAgentConfig(member.Agent)
    if err != nil {
        log.Printf("[WARN] Failed to load agent config for %s: %v", member.Agent, err)
        // Default fallback
        agentConfig = &AgentConfig{
            CLIFlags: CLIFlags{
                AllowedTools: []string{"Read", "Write", "Edit", "Bash", "Glob", "Grep"},
                AdditionalFlags: []string{"--permission-mode", "delegate"},
            },
        }
    }
    
    // 4. Build CLI args
    args := buildCLIArgs(agentConfig)
    
    // 5. Set timeout
    timeout := time.Duration(member.TimeoutMs) * time.Millisecond
    if timeout == 0 {
        timeout = 10 * time.Minute
    }
    cmdCtx, cmdCancel := context.WithTimeout(ctx, timeout)
    defer cmdCancel()
    
    // 6. Create command
    cmd := exec.CommandContext(cmdCtx, "claude", args...)
    cmd.Dir = projectRoot
    cmd.Stdin = strings.NewReader(envelope)
    cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
    
    // 7. Capture stdout for cost extraction
    var stdoutBuf bytes.Buffer
    cmd.Stdout = &stdoutBuf
    cmd.Stderr = os.Stderr  // Preserve stderr for debugging
    
    // 8. Set environment
    cmd.Env = append(os.Environ(),
        "GOGENT_NESTING_LEVEL=2",
        "GOGENT_PROJECT_ROOT="+projectRoot,
    )
    
    // 9. Start process
    if err := cmd.Start(); err != nil {
        return fmt.Errorf("start claude: %w", err)
    }
    
    // 10. Register child PID
    pid := cmd.Process.Pid
    tr.registerChild(pid)
    defer tr.unregisterChild(pid)
    
    // 11. Update member with PID
    if err := tr.updateMember(waveIdx, memIdx, func(m *Member) {
        m.ProcessPID = &pid
    }); err != nil {
        log.Printf("[WARN] Failed to update PID for %s: %v", member.Name, err)
    }
    
    // 12. Wait for completion
    err = cmd.Wait()
    
    // 13. Check timeout
    if cmdCtx.Err() == context.DeadlineExceeded {
        return fmt.Errorf("timeout after %v", timeout)
    }
    
    // 14. Check exit error
    if err != nil {
        return fmt.Errorf("claude exited with error: %w", err)
    }
    
    // 15. Extract cost
    cost, costErr := extractCostFromCLIOutput(stdoutBuf.Bytes())
    if costErr != nil {
        if costErr.Level == "error" {
            log.Printf("[ERROR] cost: %s for %s\n  → %s", 
                costErr.Message, member.Name, costErr.RawOutput[:min(len(costErr.RawOutput), 200)])
        } else {
            log.Printf("[WARN] cost: %s for %s", costErr.Message, member.Name)
        }
    }
    
    // 16. Update member with cost
    if err := tr.updateMember(waveIdx, memIdx, func(m *Member) {
        m.CostUSD = cost
        if costErr == nil {
            m.CostStatus = "ok"
        } else if costErr.Level == "warn" {
            m.CostStatus = "unknown"
        } else {
            m.CostStatus = "error"
        }
    }); err != nil {
        log.Printf("[WARN] Failed to update cost for %s: %v", member.Name, err)
    }
    
    // 17. Validate stdout
    stdoutPath := filepath.Join(tr.teamDir, member.StdoutFile)
    if err := validateStdout(stdoutPath); err != nil {
        return fmt.Errorf("validate stdout: %w", err)
    }
    
    return nil
}
```

**Test Strategy**: Extend existing `spawn_test.go` with real CLI invocation tests (use mock Claude scripts).

**Risks**:
- Most complex function (17 steps)
- Many error paths to test
- Real CLI dependency

**Mitigation**:
- Mock Claude CLI with bash scripts that write test JSON
- Test each error path with table-driven tests
- Use existing `fakeSpawner` for unit tests, real CLI for integration

### Phase 5: Main Integration & Heartbeat (Day 5)

**Objective**: Wire all components into `main.go` and verify end-to-end flow.

**Files**:
1. **`main.go` (MODIFY)** - Replace `runWaves()` placeholder at line 135

**Changes**:
```go
func main() {
    // ... existing PID file, redirect, daemonize ...
    
    runner, err := NewTeamRunner(teamDir)
    if err != nil {
        log.Fatalf("Failed to initialize TeamRunner: %v", err)
    }
    
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // NEW: Start heartbeat
    startHeartbeat(ctx, teamDir)
    
    // NEW: Validate agents exist in agents-index.json
    if err := validateAgents(runner.config); err != nil {
        log.Fatalf("Validate agents: %v", err)
    }
    
    // NEW: Update config with runner PID
    if err := runner.updateMember(0, 0, func(m *Member) {
        pid := os.Getpid()
        runner.config.BackgroundPID = &pid
    }); err != nil {
        log.Fatalf("Update config with PID: %v", err)
    }
    
    // Signal handler (unchanged)
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
    
    // NEW: Execute waves in goroutine
    doneCh := make(chan error, 1)
    go func() {
        doneCh <- runWaves(ctx, runner)
    }()
    
    // Wait for completion or signal (unchanged)
    select {
    case err := <-doneCh:
        if err != nil {
            log.Printf("Wave execution failed: %v", err)
            os.Exit(1)
        }
        log.Printf("All waves completed successfully")
    case sig := <-sigCh:
        log.Printf("Received signal %s, shutting down gracefully", sig)
        cancel()
        runner.killAllChildren()
        select {
        case <-doneCh:
        case <-time.After(shutdownTimeout):
            log.Printf("Shutdown timeout exceeded")
        }
    }
}
```

**Test Strategy**:
- Integration test with 2-wave config (3 agents total)
- Mock Claude scripts that write valid stdout JSON
- Verify config.json updates after each agent completion
- Signal test (SIGTERM mid-execution)

### Phase 6: Testing & Quality Gates (Days 6-7)

**Objective**: Achieve 80%+ coverage, pass all quality gates, manual verification.

**Test Files**:
1. **`main_test.go`** (~150 lines)
   - End-to-end with real binary
   - Signal handling (SIGTERM, SIGINT)
   - Budget exhaustion
   - Double-start prevention

2. **Extend existing tests** with real CLI scenarios

**Quality Gates** (from TC-008.md):
- [ ] `go build ./cmd/gogent-team-run` exits 0
- [ ] `go test -race ./cmd/gogent-team-run` passes with 0 warnings
- [ ] `golangci-lint run ./cmd/gogent-team-run` passes
- [ ] At least 3 successful executions with mock Claude CLI
- [ ] Budget gate prevents runaway (test with $0.50 budget)
- [ ] SIGTERM kills all children within 10 seconds
- [ ] Heartbeat file touched every 30s
- [ ] Agent failure → retry → success path tested
- [ ] Wave 2 waits for Wave 1 completion
- [ ] Atomic config.json writes (verified with kill -9 during write)

**Manual Tests**:
1. **Two-wave braintrust simulation**:
   ```bash
   # Create team config
   mkdir -p /tmp/test-team
   cat > /tmp/test-team/config.json <<EOF
   {
     "team_name": "test-braintrust",
     "workflow_type": "braintrust",
     "project_root": "$PWD",
     "session_id": "test-001",
     "created_at": "2026-02-07T00:00:00Z",
     "budget_max_usd": 5.0,
     "budget_remaining_usd": 5.0,
     "warning_threshold_usd": 1.0,
     "status": "pending",
     "waves": [
       {
         "wave_number": 1,
         "description": "Analysis wave",
         "members": [
           {
             "name": "einstein",
             "agent": "einstein",
             "model": "opus",
             "stdin_file": "stdin_einstein.json",
             "stdout_file": "stdout_einstein.json",
             "status": "pending",
             "cost_usd": 0,
             "retry_count": 0,
             "max_retries": 1,
             "timeout_ms": 600000
           }
         ]
       },
       {
         "wave_number": 2,
         "description": "Synthesis wave",
         "members": [
           {
             "name": "beethoven",
             "agent": "beethoven",
             "model": "opus",
             "stdin_file": "stdin_beethoven.json",
             "stdout_file": "stdout_beethoven.json",
             "status": "pending",
             "cost_usd": 0,
             "retry_count": 0,
             "max_retries": 1,
             "timeout_ms": 600000
           }
         ]
       }
     ]
   }
   EOF
   
   # Create mock stdin files
   echo '{"task": "Test task", "context": "Test context"}' > /tmp/test-team/stdin_einstein.json
   echo '{"task": "Synthesize", "context": "Wave 1 results"}' > /tmp/test-team/stdin_beethoven.json
   
   # Run team
   gogent-team-run /tmp/test-team
   
   # Verify:
   # 1. runner.log shows sequential wave execution
   # 2. config.json shows both members "completed"
   # 3. stdout files exist and are valid JSON
   # 4. budget_remaining_usd is updated
   ```

2. **Budget exhaustion test**:
   - Config with 5 members, budget of $0.50
   - Estimated cost per member: $0.30
   - Expected: Only 1 member spawns, rest blocked at budget gate
   - Verify: runner.log shows budget gate messages

3. **Signal handling test**:
   - Start team with long-running mock agents (sleep 60)
   - Send SIGTERM after 5 seconds
   - Verify: All children killed within 10 seconds
   - Verify: PID file removed, heartbeat file frozen

---

## Integration Points

### Inputs (Dependencies)

| Ticket | Provides | Used In |
|--------|----------|---------|
| TC-001 | `--allowedTools` CLI flag pattern | `buildCLIArgs()` |
| TC-002 | Mutex-protected config updates | `updateMember()`, `tryReserveBudget()` |
| TC-003 | Iterative retry loop | `spawnAndWait()` |
| TC-004 | Daemon lifecycle functions | `main()` |
| TC-005 | Cost extraction function | `extractCostFromCLIOutput()` |
| TC-006 | `project_root` field in config | `cmd.Dir = projectRoot` |
| TC-007 | `GOGENT_NESTING_LEVEL=2` env var | `cmd.Env` |
| TC-009 | config.json schema | `TeamConfig` struct |
| TC-014 | `cli_flags` field in agents-index.json | `loadAgentConfig()` |

### Outputs (Feeds Into)

| Ticket | Consumes | How |
|--------|----------|-----|
| TC-010 | Inter-wave script hook | `runInterWaveScript()` placeholder |
| TC-011 | Unit test patterns | Test suite serves as reference |
| TC-012 | config.json structure | Slash commands read background_pid, status |
| TC-013 | Working binary | Orchestrator rewrites invoke via Bash |

---

## Risk Assessment

### High Risks

1. **Budget Gate Logic Complexity**
   - **Risk**: Race condition in budget check + reservation
   - **Likelihood**: Medium
   - **Impact**: High (runaway costs)
   - **Mitigation**: Atomic operations via mutex, extensive race detector testing

2. **CLI Output Format Variability**
   - **Risk**: Cost extraction fails due to format changes
   - **Likelihood**: Medium
   - **Impact**: Medium (budget tracking inaccurate)
   - **Mitigation**: Multi-field fallback, graduated error levels, log raw output

3. **Process Group Kill Failures**
   - **Risk**: Stubborn children survive SIGKILL
   - **Likelihood**: Low
   - **Impact**: High (orphaned processes)
   - **Mitigation**: Process group isolation (Setsid), SIGTERM → SIGKILL escalation, proven via integration_test.go

### Medium Risks

4. **Stdin Schema Variability**
   - **Risk**: Envelope builder breaks on unexpected stdin format
   - **Likelihood**: Medium
   - **Impact**: Medium (spawn fails)
   - **Mitigation**: Lenient parsing, default fallbacks, schema examples in test fixtures

5. **Heartbeat Goroutine Leak**
   - **Risk**: Heartbeat continues after context cancellation
   - **Likelihood**: Low
   - **Impact**: Low (minor resource leak)
   - **Mitigation**: Explicit context.Done() check, defer ticker.Stop()

### Low Risks

6. **Inter-Wave Script Execution**
   - **Risk**: TC-010 script fails, blocks wave progression
   - **Likelihood**: Low (TC-010 not implemented yet)
   - **Impact**: Medium
   - **Mitigation**: Stub implementation logs and continues, full implementation in TC-010

---

## Testing Strategy

### Unit Tests (Per-File Coverage)

| File | Test Focus | Key Tests |
|------|------------|-----------|
| `wave.go` | Wave coordination | Sequential waves, parallel members, budget gates |
| `envelope.go` | Prompt building | Valid/invalid stdin, capabilities notice presence |
| `cost.go` | Cost extraction | All field variants, invalid JSON, missing field |
| `validate.go` | Stdout validation | Valid/invalid JSON, missing fields |
| `heartbeat.go` | Periodic file touch | Touch frequency, context cancellation |
| `spawn.go` | CLI lifecycle | Full spawn → wait → cost → validate flow |

### Integration Tests

1. **Full wave execution** (main_test.go)
   - 2 waves, 3 members
   - Mock Claude scripts (bash)
   - Verify sequential waves, parallel members

2. **Budget exhaustion** (main_test.go)
   - 5 members, insufficient budget
   - Verify budget gate blocks mid-wave

3. **Signal handling** (main_test.go)
   - SIGTERM mid-execution
   - Verify all children killed, PID file removed

4. **Double-start prevention** (daemon_test.go - already exists)
   - Verify second start fails with ErrTeamAlreadyRunning

### Race Detector Tests

```bash
go test -race ./cmd/gogent-team-run/...
```

**Critical Coverage**:
- Concurrent `updateMember()` calls (config_test.go:434-474 proven)
- Parallel spawn in `runWaves()`
- Budget reservation during concurrent spawns

### Coverage Target

- **Overall**: 80%+
- **Financial code** (budget functions): 95%+
- **Concurrency code** (mutexes, goroutines): 95%+

---

## Makefile Integration

**New Targets** (add to Makefile):
```makefile
# Team coordination binaries
build-team-tools: build-gogent-team-run build-gogent-team-prepare-synthesis

build-gogent-team-run:
	@echo "Building gogent-team-run..."
	@go build -o bin/gogent-team-run ./cmd/gogent-team-run

build-gogent-team-prepare-synthesis:
	@echo "Building gogent-team-prepare-synthesis..."
	@go build -o bin/gogent-team-prepare-synthesis ./cmd/gogent-team-prepare-synthesis

install-team-tools: build-team-tools
	@echo "Installing team coordination tools to ~/.local/bin..."
	@cp bin/gogent-team-run ~/.local/bin/
	@cp bin/gogent-team-prepare-synthesis ~/.local/bin/
	@echo "✓ Team tools installed"

test-team:
	@echo "Running team coordination tests..."
	@go test -v ./cmd/gogent-team-run/...
	@go test -race ./cmd/gogent-team-run/...
```

**Modified Targets**:
```makefile
build: build-hooks build-tui build-team-tools

install: install-archive install-aggregate install-load-context install-team-tools

test: test-ecosystem test-team
```

---

## Acceptance Checklist

### Build & Compilation
- [ ] `make build-gogent-team-run` succeeds
- [ ] `go build ./cmd/gogent-team-run` exits 0
- [ ] Binary runs without panics

### Test Coverage
- [ ] `go test ./cmd/gogent-team-run` all pass
- [ ] `go test -race ./cmd/gogent-team-run` 0 warnings
- [ ] Overall coverage ≥ 80%
- [ ] Financial code coverage ≥ 95%
- [ ] Concurrency code coverage ≥ 95%

### Quality Gates
- [ ] `golangci-lint run ./cmd/gogent-team-run` passes
- [ ] `go vet ./cmd/gogent-team-run` clean
- [ ] All test files use table-driven tests

### Functional Requirements
- [ ] At least 3 successful executions with mock Claude CLI
- [ ] Budget ceiling prevents runaway (tested with $0.50 budget)
- [ ] Budget remaining updated after each agent
- [ ] Budget gate blocks new spawns when exhausted
- [ ] SIGTERM kills all children within 10 seconds
- [ ] PID file written on startup, removed on clean exit
- [ ] Heartbeat file touched every 30s

### Retry Logic
- [ ] Agent failure → retry once → success marked "completed"
- [ ] Agent failure → all retries exhausted → marked "failed"
- [ ] Retry count accurate in config.json
- [ ] Error history accumulated across retries

### Concurrency
- [ ] Atomic config.json writes (verified with kill -9 during write)
- [ ] Wave 2 waits for Wave 1 completion
- [ ] Members within wave execute in parallel
- [ ] No race warnings in any test

### Agent Capabilities (TC-007)
- [ ] `GOGENT_NESTING_LEVEL=2` set in spawned agent env
- [ ] Task(haiku/sonnet) allowed (not tested in TC-008, deferred to TC-007 validator changes)
- [ ] Task(opus) blocked (deferred to TC-007)

### Cost Tracking
- [ ] Cost extraction accurate for all 3 CLI output field variants
- [ ] Missing cost field logs warning but continues
- [ ] Invalid JSON logs error with raw output (first 500 chars)
- [ ] Budget remaining never goes negative

### Manual Verification
- [ ] Two-wave braintrust simulation completes successfully
- [ ] Budget exhaustion test blocks correctly
- [ ] Signal handling test terminates cleanly
- [ ] Team survives TUI terminal close (daemon detachment verified)

---

## Critical Files for Implementation

The following files are most critical for implementing this plan:

1. **`/home/doktersmol/Documents/GOgent-Fortress/cmd/gogent-team-run/spawn.go`**
   - **Reason**: Contains the stubbed `claudeSpawner.Spawn()` that must be implemented (line 22)
   - **Pattern**: Use existing `spawnAndWait()` retry loop as integration model
   - **Risk**: Most complex function (17 steps), highest bug likelihood

2. **`/home/doktersmol/Documents/GOgent-Fortress/cmd/gogent-team-run/config.go`**
   - **Reason**: Budget management functions (`tryReserveBudget()`, `reconcileCost()`) must be added here
   - **Pattern**: Follow existing `updateMember()` mutex pattern (lines 183-227)
   - **Risk**: Race conditions if mutex discipline violated

3. **`/home/doktersmol/Documents/GOgent-Fortress/cmd/gogent-team-run/main.go`**
   - **Reason**: Entry point integration for `runWaves()`, heartbeat, and signal handling
   - **Pattern**: Existing daemon lifecycle (lines 13-131) provides proven template
   - **Risk**: Incorrect goroutine coordination between runWaves and signal handler

4. **`/home/doktersmol/Documents/GOgent-Fortress/.claude/schemas/teams/stdin-stdout/braintrust-einstein.json`**
   - **Reason**: Reference schema for stdin envelope structure
   - **Pattern**: Template for `buildPromptEnvelope()` implementation
   - **Risk**: Schema variability across workflows (braintrust vs review vs implementation)

5. **`/home/doktersmol/Documents/GOgent-Fortress/cmd/gogent-team-run/spawn_test.go`**
   - **Reason**: Proven test patterns for spawn logic (551 lines, comprehensive coverage)
   - **Pattern**: `fakeSpawner` pattern (lines 35-41) for unit tests, real CLI for integration
   - **Risk**: None (reference only, exemplary quality)

---

## Success Criteria Summary

**Minimum Viable Product**:
1. Binary compiles and runs without crashes
2. Executes 2-wave config with 3 mock agents successfully
3. Budget gates prevent overspend
4. SIGTERM terminates cleanly
5. Config.json updates correctly throughout lifecycle

**Production Ready**:
1. All acceptance checklist items ✓
2. 80%+ test coverage
3. 3+ manual verification runs
4. Zero race warnings
5. Documentation complete (this plan + inline comments)

---

## Implementation Order Dependency Graph

```
Phase 1 (wave.go) ← Phase 4 (spawn.go implementation)
           ↓
Phase 2 (envelope.go, cost.go) → Phase 4
           ↓
Phase 3 (validate.go, heartbeat.go) → Phase 4
           ↓
Phase 4 (spawn.go) → Phase 5 (main.go)
           ↓
Phase 5 (main.go) → Phase 6 (testing)
```

**Critical Path**: Phase 2 → Phase 4 → Phase 5 (envelope/cost must complete before spawn implementation).

---

## Notes for Implementer

1. **Start with Phase 2** (envelope/cost) before Phase 1 (wave). The spawn implementation needs these functions, and they're independent (easier to test in isolation).

2. **Use existing tests as templates**. The `spawn_test.go` and `daemon_test.go` files demonstrate the project's test style. Follow these patterns.

3. **Budget functions are security-critical**. The review finding (Item 13) emphasizes: ALL budget mutations MUST go through `tryReserveBudget()` and `reconcileCost()`. Direct mutex locking for budget is forbidden.

4. **Log verbosely**. The error message convention (from TC-008.md:774-791) should be followed:
   ```
   [LEVEL] component: message
     → actionable next step
   ```

5. **Test with mock Claude first**. Write bash scripts that mimic Claude CLI behavior (read stdin, write stdout JSON, exit 0). This enables testing without API costs.

6. **Race detector is mandatory**. Run `go test -race` after every significant change. The concurrent `updateMember` bug (spawn_test.go:433-474) proves why.

7. **Heartbeat is low-priority**. If time is tight, stub `startHeartbeat()` with a TODO. It's for monitoring, not correctness.

8. **Inter-wave scripts are future work**. The `runInterWaveScript()` function can log and return nil. TC-010 will implement the real behavior.

---

**Plan Status**: Ready for Implementation
**Estimated Timeline**: 5-7 days (with contingency for Phase 4 complexity)
**Next Step**: Implement Phase 2 (envelope.go, cost.go) with full test coverage
