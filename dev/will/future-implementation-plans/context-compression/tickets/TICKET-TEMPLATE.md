# RLM Ticket Template - Required Structure

**Version**: 1.0
**Last Updated**: 2026-01-17
**Purpose**: Enforce consistent detail level across all RLM implementation tickets

---

## Mandatory Ticket Structure

Every RLM ticket MUST follow this exact structure. NO shortcuts, NO "implement logic here", NO "omitted for brevity".

```markdown
#### RLM-XXX: [Descriptive Title]

**Time**: X hours
**Dependencies**: RLM-YYY, RLM-ZZZ (or "None")
**Phase**: 1|2|3|4
**Priority**: HIGH/MEDIUM/LOW

**Task**:
[One clear sentence describing what needs to be done]

**File**: `exact/path/to/file.go` (or multiple files if ticket spans multiple)

**Imports**:
```go
package packagename

import (
    "context"
    "fmt"

    "go.starlark.net/starlark"
    "github.com/yourusername/gogent/pkg/rlm"
)
```

**Implementation**:
```go
// COMPLETE, PRODUCTION-READY CODE
// Developer should be able to copy-paste this directly

func ExampleFunction(ctx context.Context, param1 string) (*Result, error) {
    // Validate inputs
    if param1 == "" {
        return nil, fmt.Errorf("[rlm-engine:component] Parameter param1 empty. Required for X. Provide non-empty value.")
    }

    // Check context
    select {
    case <-ctx.Done():
        return nil, fmt.Errorf("[rlm-engine:timeout] Context cancelled: %w", ctx.Err())
    default:
    }

    // Main logic with error handling
    result, err := doWork(param1)
    if err != nil {
        return nil, fmt.Errorf("[rlm-engine:component] Failed to do work: %w. Check input format.", err)
    }

    // Log success
    logger.Info("Operation completed",
        slog.String("component", "rlm-engine:component"),
        slog.String("param1", param1),
    )

    return result, nil
}
```

**Tests**:
```go
package packagename

import (
    "context"
    "testing"
    "time"
)

func TestExampleFunction_ValidInput(t *testing.T) {
    ctx := context.Background()
    result, err := ExampleFunction(ctx, "valid")

    if err != nil {
        t.Fatalf("Expected no error, got: %v", err)
    }

    if result == nil {
        t.Error("Expected non-nil result")
    }
}

func TestExampleFunction_EmptyParam(t *testing.T) {
    ctx := context.Background()
    _, err := ExampleFunction(ctx, "")

    if err == nil {
        t.Error("Expected error for empty param, got nil")
    }

    if !strings.Contains(err.Error(), "[rlm-engine:component]") {
        t.Errorf("Expected error with component tag, got: %v", err)
    }
}

func TestExampleFunction_ContextCancellation(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    cancel()  // Cancel immediately

    _, err := ExampleFunction(ctx, "valid")

    if err == nil {
        t.Error("Expected context cancellation error, got nil")
    }

    if !strings.Contains(err.Error(), "[rlm-engine:timeout]") {
        t.Errorf("Expected timeout error, got: %v", err)
    }
}
```

**Acceptance Criteria**:
- [ ] Specific, testable requirement 1
- [ ] Specific, testable requirement 2
- [ ] All error messages follow `[rlm-engine:component] What. Why. How.` format
- [ ] Context timeout is respected
- [ ] Structured logging to ~/.gogent/rlm-engine.log
- [ ] Tests pass: `go test ./pkg/rlm`
- [ ] Race detector clean: `go test -race ./pkg/rlm`
- [ ] Code coverage ≥80%

**Why This Matters**:
[Context explaining why this ticket exists, what problem it solves, how it fits into the RLM architecture]
```

---

## RLM-Specific Requirements

### 1. Starlark Integration

If ticket involves Starlark code execution:

```go
// Create thread with RLM built-ins
thread := &starlark.Thread{
    Name: "rlm",
    Print: func(_ *starlark.Thread, msg string) {
        printBuffer.WriteString(msg + "\n")
    },
}

predeclared := starlark.StringDict{
    "context":   starlark.String(contextData),
    "llm_query": starlark.NewBuiltin("llm_query", llmQueryFunc),
    "print":     starlark.NewBuiltin("print", printFunc),
    "FINAL":     starlark.NewBuiltin("FINAL", finalFunc),
}

// Execute Starlark code
_, err := starlark.ExecFile(thread, "metaprompt.star", code, predeclared)
if err != nil {
    return fmt.Errorf("[rlm-engine:starlark] Execution failed: %w. Check metaprompt syntax.", err)
}
```

---

### 2. Cost Tracking

If ticket involves API calls:

```go
// Track token usage
costTracker.AddRootTokens(inputTokens, outputTokens)
costTracker.AddSubTokens(inputTokens, outputTokens)

// Check cost ceiling
if err := costTracker.CheckCeiling(); err != nil {
    return fmt.Errorf("[rlm-engine:cost] %w", err)
}

// Log cost
logger.Info("API call completed",
    slog.String("component", "rlm-engine:api"),
    slog.Float64("cost_so_far", costTracker.TotalCost()),
)
```

---

### 3. REPL Iteration

If ticket involves REPL loop:

```go
for iteration := 1; iteration <= maxIterations; iteration++ {
    // Check context timeout
    select {
    case <-ctx.Done():
        return nil, fmt.Errorf("[rlm-engine:timeout] Context cancelled at iteration %d: %w", iteration, ctx.Err())
    default:
    }

    // Log iteration start
    logger.Debug("REPL iteration starting",
        slog.Int("iteration", iteration),
        slog.Int("max_iterations", maxIterations),
    )

    // Execute iteration
    result, err := executeIteration(ctx, iteration, ...)
    if err != nil {
        return nil, fmt.Errorf("[rlm-engine:repl] Iteration %d failed: %w", iteration, err)
    }

    // Check for FINAL call
    if result.IsFinal {
        logger.Info("REPL completed",
            slog.Int("iterations", iteration),
            slog.Float64("total_cost", costTracker.TotalCost()),
        )
        return result, nil
    }
}
```

---

### 4. Protocol Handling

If ticket implements a protocol:

```go
type Protocol interface {
    Name() string
    Execute(ctx context.Context, contextData, query string) (*ProtocolResult, error)
    LoadMetaprompt() (string, error)
}

type AnalyzeProtocol struct {
    config *Config
}

func (p *AnalyzeProtocol) Name() string {
    return "analyze"
}

func (p *AnalyzeProtocol) Execute(ctx context.Context, contextData, query string) (*ProtocolResult, error) {
    // Load metaprompt template
    metaprompt, err := p.LoadMetaprompt()
    if err != nil {
        return nil, fmt.Errorf("[rlm-engine:protocol] Failed to load metaprompt for protocol %s: %w", p.Name(), err)
    }

    // Execute REPL loop
    result, err := ExecuteREPL(ctx, metaprompt, contextData, query)
    if err != nil {
        return nil, fmt.Errorf("[rlm-engine:protocol] Protocol %s failed: %w", p.Name(), err)
    }

    return result, nil
}
```

---

## Required Conventions

### Error Message Format

```
[rlm-engine:component] What happened. Why it was blocked/failed. How to fix.
```

| Component | Usage |
|-----------|-------|
| `[rlm-engine]` | General RLM errors |
| `[rlm-engine:repl]` | REPL execution errors |
| `[rlm-engine:api]` | Claude API errors |
| `[rlm-engine:cost]` | Cost ceiling errors |
| `[rlm-engine:timeout]` | Timeout errors |
| `[rlm-engine:protocol]` | Protocol errors |
| `[rlm-engine:starlark]` | Starlark execution errors |

---

### File Path Conventions

```go
// CORRECT: XDG-compliant
func GetRLMLogPath() string {
    home, _ := os.UserHomeDir()
    logDir := filepath.Join(home, ".gogent")
    os.MkdirAll(logDir, 0755)
    return filepath.Join(logDir, "rlm-engine.log")
}

// WRONG: Hardcoded /tmp
func GetRLMLogPath() string {
    return "/tmp/rlm-engine.log"  // NEVER do this
}
```

---

### Context Timeout Handling

```go
// CORRECT: Check context before expensive operations
func LongOperation(ctx context.Context) error {
    select {
    case <-ctx.Done():
        return fmt.Errorf("[rlm-engine:timeout] Context cancelled: %w", ctx.Err())
    default:
    }

    // Proceed with operation
    return doWork()
}

// WRONG: Ignore context
func LongOperation(ctx context.Context) error {
    return doWork()  // No context check
}
```

---

### Structured Logging

```go
// CORRECT: Structured logging with slog
logger.Info("REPL iteration completed",
    slog.String("component", "rlm-engine:repl"),
    slog.Int("iteration", 5),
    slog.Float64("cost_so_far", 1.23),
    slog.Int("sub_queries", 3),
)

// WRONG: Unstructured logging
log.Printf("iteration 5 completed, cost: 1.23")
```

---

## Anti-Patterns (FORBIDDEN)

### ❌ Pseudocode or Placeholders

```go
// BAD
func ExecuteREPL() error {
    // TODO: Implement REPL logic here
    return nil
}

// GOOD
func ExecuteREPL(ctx context.Context, metaprompt, query string) (*REPLResult, error) {
    // [Complete implementation with error handling]
}
```

---

### ❌ Missing Error Context

```go
// BAD
if err != nil {
    return err
}

// GOOD
if err != nil {
    return fmt.Errorf("[rlm-engine:repl] Failed to execute iteration %d: %w. Check metaprompt syntax.", iteration, err)
}
```

---

### ❌ Incomplete Test Coverage

```go
// BAD - only tests happy path
func TestExecuteREPL(t *testing.T) {
    result, _ := ExecuteREPL(ctx, "valid", "query")
    // only checks success
}

// GOOD - tests multiple scenarios
func TestExecuteREPL_ValidInput(t *testing.T) { ... }
func TestExecuteREPL_InvalidMetaprompt(t *testing.T) { ... }
func TestExecuteREPL_TimeoutExceeded(t *testing.T) { ... }
func TestExecuteREPL_CostCeilingExceeded(t *testing.T) { ... }
```

---

## Checklist Before Marking Ticket Complete

- [ ] All code is production-ready (no TODOs)
- [ ] Error messages follow RLM format
- [ ] Context timeout is respected
- [ ] Structured logging implemented
- [ ] File paths are XDG-compliant
- [ ] Tests cover valid input, invalid input, edge cases, errors
- [ ] Test coverage ≥80%
- [ ] Race detector passes
- [ ] Cross-compilation succeeds (if applicable)
- [ ] Acceptance criteria met
- [ ] "Why This Matters" explains context

---

## Cross-References

- **Standards**: See [00-overview.md](00-overview.md)
- **Main Plan**: See [../RLM_IMPLEMENTATION_PLAN.md](../RLM_IMPLEMENTATION_PLAN.md)
- **Go Conventions**: See `/home/doktersmol/.claude/conventions/go.md`

---

**Remember**: If you find yourself writing "implement X here", you haven't provided enough detail. Every ticket should be copy-paste ready for a contractor who has never seen the codebase.
