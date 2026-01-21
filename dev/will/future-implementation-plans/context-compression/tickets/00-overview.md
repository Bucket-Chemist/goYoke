# RLM Engine - Standards and Strategy

**Version**: 1.0
**Date**: 2026-01-17
**Status**: Draft

---

## Document Purpose

This document contains cross-cutting standards that apply to **ALL** RLM implementation tickets. Developers should read this file completely before starting any implementation work.

**Contents**:
1. Testing Strategy
2. Error Handling Standards
3. Logging Strategy
4. Cost Tracking Requirements
5. Starlark Compatibility Guidelines
6. Cross-Compilation Requirements
7. Integration Testing Strategy
8. Rollback Plan

---

## Testing Strategy

### 1. Unit Tests (Continuous - Every Ticket)

**Coverage Target**: ≥80% per package

**Test Naming**: `TestFunctionName_Scenario`

**Required Test Cases**:
- Valid input (happy path)
- Invalid input (error handling)
- Edge cases (empty strings, nil pointers, boundary values)
- Error conditions (API failure, timeout, malformed input)

**Example Structure**:
```go
func TestExecuteREPL_ValidMetaprompt(t *testing.T) {
    engine := NewEngine(testConfig)
    result, err := engine.ExecuteREPL(context.Background(), "valid metaprompt", "test query")

    if err != nil {
        t.Fatalf("Expected no error, got: %v", err)
    }

    if result.Output == "" {
        t.Error("Expected non-empty output")
    }
}

func TestExecuteREPL_TimeoutExceeded(t *testing.T) {
    engine := NewEngine(testConfig)
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
    defer cancel()

    _, err := engine.ExecuteREPL(ctx, "slow metaprompt", "test query")

    if err == nil {
        t.Error("Expected timeout error, got nil")
    }

    if !errors.Is(err, context.DeadlineExceeded) {
        t.Errorf("Expected DeadlineExceeded, got: %v", err)
    }

    // Verify error message format
    if !strings.Contains(err.Error(), "[rlm-engine]") {
        t.Errorf("Expected error with component tag, got: %v", err)
    }
}
```

**Run After Each Ticket**:
```bash
go test ./pkg/rlm/...
```

**Before Committing**:
```bash
go test ./... -cover
go test ./... -race  # Race detector MANDATORY
```

---

### 2. Starlark Integration Tests (Weekly)

**Purpose**: Verify RLM metaprompts execute correctly in Starlark

**Test Corpus**: Metaprompt templates from paper
- Chunk-and-aggregate
- Peek-filter-dive
- Prior-based probing
- Verification loop

**Test Harness**:
```go
func TestStarlarkMetaprompt_ChunkAndAggregate(t *testing.T) {
    // Load metaprompt template
    template, err := os.ReadFile("../../templates/chunk-and-aggregate.star")
    require.NoError(t, err)

    // Create Starlark thread with RLM built-ins
    thread := &starlark.Thread{Name: "test"}
    predeclared := CreateRLMBuiltins(testContext, testLLMClient)

    // Execute metaprompt
    _, err = starlark.ExecFile(thread, "test.star", template, predeclared)
    require.NoError(t, err)

    // Verify FINAL was called
    assert.True(t, predeclared["FINAL"].Called())
}
```

---

### 3. Corpus Testing (Phase 3)

**Purpose**: Test RLM with real large contexts (6M-11M tokens)

**Test Cases**:
1. **Security Analysis**: 8M token codebase, find vulnerabilities
2. **API Extraction**: 10M token docs, extract all endpoints
3. **Architecture Documentation**: 6M token codebase, generate system diagram

**Pass Criteria**:
- Completes within timeout (30 minutes)
- Cost ≤ $5.00 per test
- Accuracy improvement vs direct long-context (measured qualitatively)

**Test Procedure**:
```bash
# Generate test corpus
make generate-corpus SIZE=8M

# Run RLM analysis
cat test/fixtures/corpus-8M.txt | \
  ./rlm-engine analyze "Find all SQL injection vulnerabilities" \
  > test/output/security-analysis-rlm.md

# Compare to baseline (direct long-context)
# Manual review of output quality
```

---

### 4. Integration Tests (Phase 4 - RLM-035)

**Purpose**: Test end-to-end workflow with Claude Code routing

**Test Scenarios**:
1. Orchestrator detects trigger → routes to RLM
2. RLM executes → writes output
3. Orchestrator reads output → synthesizes for user

**Test Script**:
```bash
#!/bin/bash
# test/integration/test-rlm-routing.sh

set -e

echo "[Test] Creating large context file..."
cat test/fixtures/large-codebase.tar.gz > /tmp/test-context.txt

echo "[Test] Simulating orchestrator invocation..."
cat /tmp/test-context.txt | \
  rlm-engine analyze "Find security issues" \
  > ~/.claude/tmp/rlm-output.md

echo "[Test] Verifying output exists..."
if [[ ! -f ~/.claude/tmp/rlm-output.md ]]; then
    echo "FAIL: Output file not created"
    exit 1
fi

echo "[Test] Verifying output format..."
if ! grep -q "# RLM Analysis" ~/.claude/tmp/rlm-output.md; then
    echo "FAIL: Output missing expected header"
    exit 1
fi

echo "PASS: Integration test successful"
```

---

## Error Handling Standards

### Error Message Format

```
[rlm-engine] What happened. Why it was blocked/failed. How to fix.
```

**Examples**:
- ✅ GOOD: `[rlm-engine] REPL iteration 15 exceeded 5-minute timeout. Context size (12M tokens) too large. Try smaller context or increase timeout with --timeout flag.`
- ✅ GOOD: `[rlm-engine] Sub-LLM API call failed after 3 retries. Network error: connection refused. Check internet connection and Claude API status.`
- ❌ BAD: `timeout`
- ❌ BAD: `failed to execute`

### Error Types

| Error Class | Component Tag | Example |
|-------------|---------------|---------|
| **REPL Execution** | `[rlm-engine:repl]` | `[rlm-engine:repl] Starlark syntax error at line 42: unexpected EOF. Check metaprompt template syntax.` |
| **API Errors** | `[rlm-engine:api]` | `[rlm-engine:api] Claude API rate limit exceeded. Retry in 60 seconds or reduce sub-query frequency.` |
| **Cost Errors** | `[rlm-engine:cost]` | `[rlm-engine:cost] Cost ceiling ($10.00) exceeded at $10.23. Execution halted. Review query complexity.` |
| **Timeout Errors** | `[rlm-engine:timeout]` | `[rlm-engine:timeout] Total execution time (35 min) exceeded 30-minute limit. Context too large for RLM.` |
| **Protocol Errors** | `[rlm-engine:protocol]` | `[rlm-engine:protocol] Unknown protocol 'invalid'. Valid protocols: analyze, compress, metaprompt.` |

### Error Handling Pattern

```go
// In all RLM packages
func ExecuteREPL(ctx context.Context, metaprompt, query string) (*REPLResult, error) {
    // Validate inputs
    if metaprompt == "" {
        return nil, fmt.Errorf("[rlm-engine] Metaprompt empty. Required for REPL execution. Provide valid metaprompt template.")
    }

    // Execute with timeout
    select {
    case result := <-resultCh:
        if result.err != nil {
            return nil, fmt.Errorf("[rlm-engine:repl] Execution failed: %w. Check metaprompt syntax and LLM responses.", result.err)
        }
        return result.output, nil
    case <-ctx.Done():
        return nil, fmt.Errorf("[rlm-engine:timeout] Context cancelled: %w. REPL iteration may have hung.", ctx.Err())
    }
}
```

### Graceful Degradation

When errors occur mid-execution:
1. **Log error** with full context
2. **Preserve partial results** (if any)
3. **Return best-effort output** with error flag
4. **Include cost incurred** in response

```go
type REPLResult struct {
    Output          string
    PartialOutput   string  // Non-empty if error occurred mid-execution
    IterationCount  int     // How many iterations completed
    CostIncurred    float64 // Total API cost
    Error           error   // Non-nil if failed
    ErrorIteration  int     // Which iteration failed (0 if N/A)
}
```

---

## Logging Strategy

### Log Destination

```
~/.gogent/rlm-engine.log
```

**Rotation**: Daily, keep last 7 days

**Format**: Structured JSON (one entry per line)

### Log Levels

| Level | Usage |
|-------|-------|
| **DEBUG** | Iteration details, Starlark execution trace |
| **INFO** | API calls, cost tracking, protocol selection |
| **WARN** | Retry attempts, approaching cost ceiling |
| **ERROR** | Failures, timeouts, API errors |

### Log Entry Structure

```json
{
  "timestamp": "2026-01-17T10:30:45Z",
  "level": "INFO",
  "component": "rlm-engine:repl",
  "message": "REPL iteration 5 completed",
  "iteration": 5,
  "cost_so_far": 1.23,
  "sub_queries": 3,
  "duration_ms": 2450
}
```

### Logging Example

```go
package rlm

import (
    "log/slog"
    "os"
    "path/filepath"
)

var logger *slog.Logger

func init() {
    // Create log directory
    home, _ := os.UserHomeDir()
    logDir := filepath.Join(home, ".gogent")
    os.MkdirAll(logDir, 0755)

    // Open log file
    logFile, err := os.OpenFile(
        filepath.Join(logDir, "rlm-engine.log"),
        os.O_CREATE|os.O_WRONLY|os.O_APPEND,
        0644,
    )
    if err != nil {
        log.Fatal(err)
    }

    // Create structured logger
    logger = slog.New(slog.NewJSONHandler(logFile, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }))
}

func LogREPLIteration(iteration int, cost float64, subQueries int, duration time.Duration) {
    logger.Info("REPL iteration completed",
        slog.String("component", "rlm-engine:repl"),
        slog.Int("iteration", iteration),
        slog.Float64("cost_so_far", cost),
        slog.Int("sub_queries", subQueries),
        slog.Int64("duration_ms", duration.Milliseconds()),
    )
}
```

---

## Cost Tracking Requirements

### Cost Ceiling

**Hard Limit**: $10.00 per invocation

**Warning Thresholds**:
- $5.00: Log warning
- $7.50: Log warning + user notification (if interactive)
- $10.00: **HARD STOP** - halt execution immediately

### Token Counting

**Must track**:
1. Root LLM tokens (Opus/Sonnet for metaprompt)
2. Sub-LLM tokens (Haiku for queries)
3. Input tokens vs output tokens
4. Cached vs non-cached tokens (if applicable)

**Formula**:
```
Total Cost = (Root Input Tokens × $0.045/1K) +
             (Root Output Tokens × $0.180/1K) +
             (Sub Input Tokens × $0.0005/1K) +
             (Sub Output Tokens × $0.002/1K)
```

### Cost Tracking Implementation

```go
package rlm

type CostTracker struct {
    RootInputTokens  int
    RootOutputTokens int
    SubInputTokens   int
    SubOutputTokens  int
    CachedTokens     int
}

func (ct *CostTracker) TotalCost() float64 {
    rootCost := (float64(ct.RootInputTokens) * 0.045 / 1000) +
                (float64(ct.RootOutputTokens) * 0.180 / 1000)
    subCost := (float64(ct.SubInputTokens) * 0.0005 / 1000) +
               (float64(ct.SubOutputTokens) * 0.002 / 1000)
    return rootCost + subCost
}

func (ct *CostTracker) CheckCeiling() error {
    cost := ct.TotalCost()

    if cost >= 5.00 && cost < 7.50 {
        logger.Warn("Cost approaching ceiling", slog.Float64("cost", cost))
    }

    if cost >= 7.50 && cost < 10.00 {
        logger.Warn("Cost near ceiling", slog.Float64("cost", cost))
    }

    if cost >= 10.00 {
        return fmt.Errorf("[rlm-engine:cost] Cost ceiling ($10.00) exceeded at $%.2f. Execution halted.", cost)
    }

    return nil
}
```

### Cost Reporting

**Output Format** (appended to markdown result):
```markdown
---

## Cost Report

- **Root LLM**: 35,420 input tokens, 8,230 output tokens = $1.89
- **Sub-LLM**: 245,000 input tokens, 12,300 output tokens = $0.15
- **Total**: $2.04
- **Iterations**: 12
- **Duration**: 8m 23s
```

---

## Starlark Compatibility Guidelines

### Supported Python Features

| Feature | Starlark Support | Example |
|---------|------------------|---------|
| **Variables** | ✅ Full | `x = 10` |
| **Functions** | ✅ Full | `def foo(a, b): return a + b` |
| **Lists** | ✅ Full | `items = [1, 2, 3]` |
| **Dicts** | ✅ Full | `d = {"key": "value"}` |
| **Strings** | ✅ Full | `s = "hello"[0:2]` |
| **Loops** | ✅ Full | `for i in range(10): ...` |
| **Conditionals** | ✅ Full | `if x > 5: ...` |
| **List comprehensions** | ✅ Partial | `[x*2 for x in items]` (simple only) |
| **Imports** | ❌ None | N/A - Starlark is sandboxed |
| **File I/O** | ❌ None | N/A - Hermetic execution |
| **Classes** | ❌ None | Use functions instead |
| **Exceptions** | ❌ Limited | `fail()` instead of `raise` |

### Unsupported Patterns

**From paper's metaprompts** that need conversion:

| Python Pattern | Starlark Alternative |
|----------------|----------------------|
| `import json` | Use string concatenation |
| `with open(...) as f:` | Context is pre-loaded in `context` variable |
| `raise ValueError(...)` | `fail("error message")` |
| `class MyClass:` | Use dict or functions |
| `try: ... except:` | Check return values explicitly |

### Metaprompt Conversion Example

**Python (from paper)**:
```python
import json

results = []
for chunk in chunks:
    response = llm_query(f"Analyze: {chunk}")
    results.append(json.loads(response))

with open("output.json", "w") as f:
    json.dump(results, f)
```

**Starlark (for RLM engine)**:
```python
# No imports needed - JSON is strings

results = []
for chunk in chunks:
    response = llm_query("Analyze: " + chunk)
    results.append(response)  # Keep as string

# Output via FINAL, not file write
FINAL("\n".join(results))
```

### Testing Starlark Compatibility

Every metaprompt template MUST pass:
```bash
# Validate Starlark syntax
go run cmd/rlm-validate-metaprompt/main.go templates/chunk-and-aggregate.star

# Test execution with mock context
go test ./pkg/rlm -run TestMetaprompt_ChunkAndAggregate
```

---

## Cross-Compilation Requirements

### Target Platforms

**MUST compile for**:
- darwin/amd64 (macOS Intel)
- darwin/arm64 (macOS Apple Silicon)
- windows/amd64 (Windows 64-bit)
- linux/amd64 (Linux 64-bit)

### Makefile Targets

```makefile
BINARY_NAME=rlm-engine
VERSION=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X main.version=${VERSION} -s -w"

.PHONY: build build-all clean test

build:
	go build ${LDFLAGS} -o ${BINARY_NAME} ./cmd/rlm-engine

build-all:
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-darwin-amd64 ./cmd/rlm-engine
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-darwin-arm64 ./cmd/rlm-engine
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-windows-amd64.exe ./cmd/rlm-engine
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-linux-amd64 ./cmd/rlm-engine

clean:
	rm -f ${BINARY_NAME}
	rm -rf dist/

test:
	go test -race -v ./...
```

### Binary Size Target

**Target**: <10MB per binary (including Starlark)

**Optimization Flags**:
- `-s`: Omit symbol table
- `-w`: Omit DWARF debug info

**Verify**:
```bash
make build-all
ls -lh dist/
# Should show all binaries <10MB
```

---

## Integration Testing Strategy

### Phase 4 Requirements (RLM-035)

**Test Coverage**:
1. Bash invocation with stdin piping
2. Bash invocation with file input
3. Output file creation and format
4. Error code propagation
5. Cost ceiling enforcement
6. Timeout enforcement
7. Routing trigger detection
8. Cross-platform compatibility

### Test Matrix

| Platform | Bash Shell | Test Command | Pass Criteria |
|----------|------------|--------------|---------------|
| **Linux** | bash 5.1+ | `cat test.txt \| rlm-engine analyze "query"` | Exit 0, output exists |
| **macOS** | zsh/bash | Same | Exit 0, output exists |
| **Windows** | Git Bash | Same | Exit 0, output exists |
| **Windows** | WSL2 | Same | Exit 0, output exists |

### Automated Test Suite

```bash
#!/bin/bash
# test/integration/cross-platform-test.sh

set -e

PLATFORM=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

echo "[Test] Platform: $PLATFORM $ARCH"

# Test 1: Stdin piping
echo "[Test 1] Testing stdin piping..."
echo "test context" | ./rlm-engine analyze "test query" > /tmp/rlm-test-1.md
[[ -f /tmp/rlm-test-1.md ]] || (echo "FAIL: Output not created"; exit 1)

# Test 2: File input
echo "[Test 2] Testing file input..."
echo "test context" > /tmp/rlm-test-input.txt
./rlm-engine analyze --context-file /tmp/rlm-test-input.txt "test query" > /tmp/rlm-test-2.md
[[ -f /tmp/rlm-test-2.md ]] || (echo "FAIL: Output not created"; exit 1)

# Test 3: Error handling
echo "[Test 3] Testing error handling..."
./rlm-engine invalid-protocol "query" 2>&1 | grep -q "\[rlm-engine:protocol\]" || \
    (echo "FAIL: Error format incorrect"; exit 1)

# Test 4: Cost ceiling
echo "[Test 4] Testing cost ceiling..."
# (requires mock LLM client for testing)

echo "PASS: All integration tests passed on $PLATFORM $ARCH"
```

---

## Rollback Plan

### If Phase 1 Fails

**Symptoms**:
- Starlark integration doesn't work
- RLM metaprompts fail to execute
- Binary size >50MB

**Actions**:
1. Document specific failure mode
2. Assess alternatives:
   - Option A: Embedded Python via CGo (violates constraints)
   - Option B: Simpler templating (no REPL, just string substitution)
   - Option C: Abort RLM, focus on direct long-context optimization
3. Consult with user before proceeding

**Rollback**: No system changes yet, safe to abandon

---

### If Phase 2 Fails

**Symptoms**:
- Routing integration breaks existing workflows
- External agent pattern doesn't work as expected
- Performance unacceptable (>5min for simple queries)

**Actions**:
1. Revert `routing-schema.json` changes
2. Remove `agents/rlm-engine/` directory
3. Remove binary from `~/.local/bin/`
4. Document lessons learned

**Rollback Time**: <10 minutes

---

### If Phase 3 Fails

**Symptoms**:
- RLM accuracy worse than paper's claims
- Costs exceed $10/invocation frequently
- Metaprompts produce gibberish

**Actions**:
1. Analyze why performance doesn't match paper
2. Options:
   - Tune metaprompts (extend Phase 3)
   - Adjust cost ceiling
   - Document limitations
3. Decide if RLM is viable for this use case

**Rollback**: Mark as experimental, document limitations

---

### If Phase 4 Fails

**Symptoms**:
- Cross-platform compilation issues
- Integration tests fail on Windows/macOS
- Installation script breaks

**Actions**:
1. Fix platform-specific issues (extend Phase 4)
2. Mark unsupported platforms explicitly
3. Document workarounds

**Rollback**: RLM still functional on supported platforms

---

## Conventions (Must Apply to ALL Tickets)

### File Path Conventions

**ALWAYS** use XDG Base Directory compliance:
- Logs: `~/.gogent/rlm-engine.log`
- Cache: `$XDG_CACHE_HOME/rlm/` or `~/.cache/rlm/`
- Config: `~/.claude/agents/rlm-engine/`

**NEVER** hardcode `/tmp` paths (can be noexec, cleared on reboot)

```go
func GetRLMCacheDir() string {
    if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
        dir := filepath.Join(xdg, "rlm")
        os.MkdirAll(dir, 0755)
        return dir
    }

    home, _ := os.UserHomeDir()
    dir := filepath.Join(home, ".cache", "rlm")
    os.MkdirAll(dir, 0755)
    return dir
}
```

---

### Timeout Handling

**MANDATORY**: All long-running operations MUST respect context timeout

```go
func ExecuteREPL(ctx context.Context, ...) (*REPLResult, error) {
    for iteration := 1; iteration <= maxIterations; iteration++ {
        // Check context before expensive operation
        select {
        case <-ctx.Done():
            return nil, fmt.Errorf("[rlm-engine:timeout] Context cancelled at iteration %d: %w", iteration, ctx.Err())
        default:
        }

        // Execute iteration with timeout
        result, err := executeIteration(ctx, ...)
        if err != nil {
            return nil, err
        }
    }
}
```

---

### Go Version and Module

- **Go Version**: 1.21+
- **Module Path**: `github.com/yourusername/gogent-fortress`

---

## Checklist: Before Marking Ticket Complete

- [ ] All code is production-ready (no TODOs, no placeholders)
- [ ] Error messages follow `[rlm-engine:component] What. Why. How.` format
- [ ] Logging uses structured slog to `~/.gogent/rlm-engine.log`
- [ ] File paths use XDG compliance (no hardcoded `/tmp`)
- [ ] Long operations respect context.Context timeout
- [ ] Tests are complete with ≥80% coverage
- [ ] Race detector passes: `go test -race ./...`
- [ ] Cross-compilation succeeds for all 4 platforms
- [ ] Acceptance criteria are met
- [ ] "Why This Matters" section explains context
- [ ] Dependencies are listed correctly

---

## Cross-References

- **Main Plan**: See [../RLM_IMPLEMENTATION_PLAN.md](../RLM_IMPLEMENTATION_PLAN.md)
- **Ticket Template**: See [TICKET-TEMPLATE.md](TICKET-TEMPLATE.md)
- **Go Conventions**: See `/home/doktersmol/.claude/conventions/go.md`
- **RLM Paper**: [arXiv:2501.09768](https://arxiv.org/abs/2501.09768)
- **Starlark Spec**: [github.com/google/starlark-go](https://github.com/google/starlark-go/blob/master/doc/spec.md)

---

**Remember**: These standards apply to EVERY ticket. If you find yourself violating a standard, either fix the code or propose an update to this document with justification.
