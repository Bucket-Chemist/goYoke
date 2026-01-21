# GOgent Fortress: Go Translation of Claude Code Hooks

**Status:** Phase 0 - Go Translation (In Progress)
**Timeline:** 3 weeks + 1 day (pre-work complete ✅)
**Review Status:** ✅ Staff Architect Approved
**Version:** 3.1 FINAL

---

## What is GOgent Fortress?

GOgent Fortress is a phased migration project that translates the Claude Code hook system from Bash to Go, with the ultimate goal of building a production-ready daemon with persistent supervision and crash recovery.

**Current State:** Claude Code hooks implemented in Bash (~400 lines), functional but reaching maintainability limits.

**Destination:** Production-ready Go daemon with persistent supervision, crash recovery, and optional TUI observer.

**Strategy:** 3-phase approach that decouples **translation risk** (Bash → Go) from **architectural risk** (hook model → daemon model).

---

## Why This Approach?

| Alternative                      | Risk    | Timeline    | Outcome                            |
| -------------------------------- | ------- | ----------- | ---------------------------------- |
| Big Bang (Go daemon immediately) | High    | 9 weeks     | 2 simultaneous risks               |
| Hook-Only (skip daemon)          | Low     | 6 weeks     | No crash recovery, no scaling      |
| **This Plan (phased hybrid)**    | **Low** | **9 weeks** | **Proven patterns, safe rollback** |

**Key Insight:** The current Bash system WORKS. Migrating to Go removes one problem (maintainability). Migrating to daemon removes another problem (supervision). These are **orthogonal risks** that should be tackled sequentially, not simultaneously.

---

## Architecture Overview

### Phase 0: Go Translation (Current - Weeks 1-3)

**Objective:** 1:1 translation of Bash hooks to Go, zero architectural changes.

**What Gets Translated:**

```
Bash Hooks                    →  Go Binaries
─────────────────────────────────────────────────
validate-routing.sh (401 lines) → cmd/gogent-validate/main.go
session-archive.sh  (111 lines) → cmd/gogent-archive/main.go
sharp-edge-detector.sh (105 lines) → cmd/gogent-sharp-edge/main.go
```

**Benefits:**

- Type safety (catch errors at compile time)
- Faster execution (<2ms vs ~40ms for validate-routing)
- Easier to maintain and extend
- **Zero architectural risk** (same input/output)

**Rollback Plan:** Can run Go and Bash in parallel during testing, revert instantly if issues found.

### Phase 1: Daemon Architecture (Weeks 4-6)

**Objective:** Convert standalone hooks into supervised daemon with persistent state.

**Architecture:**

```
Claude Code Session
      ↓
  Hook (thin shim)
      ↓
  Unix Socket IPC
      ↓
  GOgent Daemon (Go)
  ├── Session Manager
  ├── Violation Logger
  ├── Health Monitor
  └── State Persistence
```

**Benefits:**

- Crash recovery (daemon restarts independently)
- Persistent session state (survives Claude Code crashes)
- Health monitoring and stuck detection
- Graceful degradation

### Phase 2: TUI Observer (Weeks 7-9) - Optional

**Objective:** Real-time dashboard for observing Claude Code activity.

**Features:**

- Live session monitoring
- Routing violation dashboard
- Agent activity visualization
- Performance metrics

**Risk:** Low (TUI is optional observer, not controller).

---

## Current Progress

### ✅ GOgent-000: Baseline Measurement (Complete)

**Deliverables:**

- Performance baseline: validate-routing (43ms), session-archive (19ms), sharp-edge-detector (36ms)
- Go corpus logger implemented (validates event schema before Week 1)
- 100-event synthetic test corpus
- Real event capture running via `~/.claude/hooks/zzz-corpus-logger`

**Performance SLA:**

- **Target:** Go hooks ≤ Bash average latency
- **Acceptable:** +20% degradation if ≤10ms p99
- **Unacceptable:** >10ms p99 latency

### 🚧 Week 1: Foundation & Events (Next - 9 tickets)

**Tickets:** GOgent-001 to GOgent-009

**Goals:**

- Go module initialization
- Event schema structs (validated by corpus logger)
- STDIN reading with timeout (M-6 fix)
- XDG-compliant path resolution (M-2 fix)
- Error message standards: `[component] What. Why. How to fix.`

---

## Repository Structure

```
GOgent-Fortress/
├── cmd/                          # Binary entry points
│   ├── gogent-validate/          # validate-routing.sh → Go
│   ├── gogent-archive/           # session-archive.sh → Go
│   └── gogent-sharp-edge/        # sharp-edge-detector.sh → Go
├── pkg/                         # Public packages
│   ├── routing/                 # Validation logic
│   ├── config/                  # Schema loading
│   ├── session/                 # Session management
│   └── memory/                  # Sharp edge detection
├── internal/                    # Private packages
│   └── logger/                  # Structured logging
├── test/
│   ├── fixtures/
│   │   └── event-corpus.json    # 100-event test corpus
│   ├── integration/             # Integration tests
│   └── benchmark/               # Performance benchmarks
├── dev/tools/corpus-logger/     # Pre-work: Real event capture
│   ├── main.go                  # Event capture implementation
│   ├── main_test.go            # Tests (50% coverage)
│   └── install.sh              # Hook installer
└── migration_plan/
    └── BASELINE.md              # Performance baseline documentation
```

---

## Building & Testing

### Prerequisites

- Go 1.23+
- jq (for corpus curation)
- Claude Code installed

### Build Commands

```bash
# Build all binaries
make build-all

# Build single binary
go build -o bin/gogent-validate ./cmd/gogent-validate

# Run tests
go test ./...

# Run tests with coverage
go test ./... -cover

# Run tests with race detector
go test -race ./...

# Run benchmarks
go test -bench=. ./test/benchmark
```

### Installation

```bash
# Phase 0: Install Go hooks (replaces Bash)
./scripts/install.sh

# Rollback to Bash hooks
./scripts/rollback.sh
```

---

## Event Schema

All hooks receive events via STDIN in JSON format:

```go
type HookEvent struct {
    ToolName      string                 `json:"tool_name"`
    ToolInput     map[string]interface{} `json:"tool_input,omitempty"`
    ToolResponse  map[string]interface{} `json:"tool_response,omitempty"`
    SessionID     string                 `json:"session_id"`
    HookEventName string                 `json:"hook_event_name"`
    CapturedAt    int64                  `json:"captured_at"`
}
```

**PreToolUse Example:**

```json
{
  "tool_name": "Task",
  "tool_input": {
    "model": "sonnet",
    "prompt": "AGENT: python-pro\n\nImplement function",
    "subagent_type": "general-purpose"
  },
  "session_id": "abc-123",
  "hook_event_name": "PreToolUse"
}
```

**PostToolUse Example:**

```json
{
  "tool_name": "Bash",
  "tool_response": {
    "exit_code": 1,
    "stderr": "Error: file not found"
  },
  "session_id": "abc-123",
  "hook_event_name": "PostToolUse"
}
```

---

## Standards & Conventions

### Error Message Format

**Required:** `[component] What happened. Why it was blocked/failed. How to fix.`

**Examples:**

✅ **Good:**

```
[validate-routing] Task(opus) blocked. Einstein requires GAP document workflow for cost control. Generate GAP: .claude/tmp/einstein-gap-{timestamp}.md, then run /einstein.
```

❌ **Bad:**

```
Task blocked.
```

### File Paths (XDG Compliance)

**Never use hardcoded `/tmp` paths.** Priority:

1. `$XDG_RUNTIME_DIR/gogent/` (session-specific, auto-cleaned)
2. `$XDG_CACHE_HOME/gogent/` (user-configurable)
3. `~/.cache/gogent/` (standard fallback)

**Example:**

```go
func GetGOgentDir() string {
    if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
        return filepath.Join(xdg, "gogent")
    }
    if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
        return filepath.Join(xdg, "gogent")
    }
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".cache", "gogent")
}
```

### STDIN Timeout

**All hooks MUST implement 5-second timeout on STDIN reads** (fixes M-6):

```go
func ReadStdin(timeout time.Duration) ([]byte, error) {
    ch := make(chan []byte, 1)
    errCh := make(chan error, 1)

    go func() {
        data, err := io.ReadAll(os.Stdin)
        if err != nil {
            errCh <- err
            return
        }
        ch <- data
    }()

    select {
    case data := <-ch:
        return data, nil
    case err := <-errCh:
        return nil, fmt.Errorf("reading stdin: %w", err)
    case <-time.After(timeout):
        return nil, fmt.Errorf("stdin read timeout after %v", timeout)
    }
}
```

### Logging

**Structured logs to `~/.gogent/hooks.log`:**

```json
{
  "ts": "2026-01-15T14:32:15Z",
  "level": "ERROR",
  "component": "validate-routing",
  "msg": "Task(opus) blocked",
  "session_id": "abc123",
  "details": {...}
}
```

**Debugging:**

```bash
# View recent errors
tail -n 50 ~/.gogent/hooks.log | jq 'select(.level=="ERROR")'

# View session violations
grep session-abc123 ~/.cache/gogent/routing-violations.jsonl | jq .
```

---

## Testing Strategy

### Unit Tests (Every Ticket)

**Coverage Target:** ≥80% per package

**Test Naming:** `TestFunctionName_Scenario`

**Required Cases:**

- Valid input (happy path)
- Invalid input (error handling)
- Edge cases (empty strings, nil pointers)
- Error conditions (file not found, timeout)

**Run after each ticket:**

```bash
go test ./...
```

### Integration Tests (Week 3)

**100-event corpus replay:**

```bash
# Run all events through hooks, compare Go vs Bash output
go test ./test/integration/... -v
```

**Regression tests:**

```bash
# Verify 100% match with Bash output
go test ./test/regression/... -v
```

### Performance Benchmarks (Week 3)

**Target:** Go hooks ≤ Bash average latency

```bash
go test -bench=. ./test/benchmark

# Compare against baseline
./scripts/compare-baseline.sh
```

---

## Rollback Plan

**If Go hooks cause issues:**

1. **Immediate rollback** (<5 minutes):

   ```bash
   ./scripts/rollback.sh
   ```

   Restores Bash hooks from backup.

2. **Parallel testing** (24 hours):

   ```bash
   ./scripts/parallel-test.sh
   ```

   Runs Go and Bash side-by-side, compares outputs.

3. **GO/NO-GO decision** (Week 3, Day 3):
   - ✅ **GO**: 100% corpus match, performance ≤ baseline → cutover
   - ❌ **NO-GO**: Differences found → investigate, fix, or rollback

---

## Documentation

### For Contributors

- **Migration Plan:** `/migration_plan/gogent_migration_plan_v3_FINAL.md`
- **Ticket Details:** `/migration_plan/finalised/tickets/`
- **Critical Review:** `/migration_plan/finalised/CRITICAL_REVIEW.md`
- **Baseline:** `/migration_plan/BASELINE.md`

### For Users

- **Installation:** (TBD - Week 3)
- **Troubleshooting:** (TBD - Week 3)
- **Configuration:** (TBD - Week 3)

---

## Support & Contributing

### Reporting Issues

**Phase 0 bugs:**

- Include: Go version, OS, full error message
- Attach: logs from `~/.gogent/hooks.log`
- Provide: minimal reproduction case

### Development Workflow

1. Create branch: `gogent-XXX-description`
2. Implement ticket following `TICKET-TEMPLATE.md`
3. Run tests: `go test ./...`
4. Commit with format:

   ```
   GOgent-XXX: Title

   - Implementation detail 1
   - Implementation detail 2
   - Test coverage: XX%

   Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
   ```

5. Push and create PR
6. Wait for review
7. Merge to main

---

## License

Copyright © 2025 William Klare. All rights reserved.

This software and its associated documentation, architecture, and design are proprietary.
No license is granted to use, copy, modify, or distribute any part of this codebase without
explicit written permission from the author.

---

## Acknowledgments

- **Claude Code Team** - Original hook system design
- **Staff Architect** - Critical review and approval
- **go-pro agent** - Corpus logger implementation

---

**Current Phase:** 0 (Go Translation)
**Current Ticket:** GOgent-000 ✅ Complete
**Next Ticket:** GOgent-001 (Go Module Setup)
**Timeline:** On track for 3-week Phase 0 completion
