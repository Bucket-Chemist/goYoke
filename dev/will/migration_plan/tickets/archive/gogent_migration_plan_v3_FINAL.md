# GOgent Fortress Migration Plan V3.1 (FINAL)

**Hybrid Daemon Architecture with Go Translation**

**Version**: 3.1 FINAL (Critical Review Applied)
**Date**: 2026-01-15
**Status**: APPROVED FOR IMPLEMENTATION
**Total Timeline**: 9 weeks + 1 day pre-work (3 phases)
**Risk Level**: LOW (post-review)
**Review Status:** ✅ Staff Architect Approved

---

## Document Change Log

**V3.0 → V3.1 Changes (Critical Review Fixes):**

- Added Pre-Work Phase (GOgent-000): Baseline measurement
- Added Error Message Standards section
- Added Logging Strategy section
- Fixed hardcoded /tmp paths → XDG with fallback
- Added Performance SLA definition
- Added Schema Version Validation
- Completed all schema struct definitions (Appendix A)
- Fixed circular dependencies in ticket ordering
- Added WSL2 testing requirements
- Deferred Phase 2 (TUI) to v1.1 as optional

---

## Executive Summary

**Current State**: Claude Code hook system implemented in Bash (400+ lines), functional but reaching maintainability limits.

**Destination**: Production-ready Go daemon with persistent supervision, crash recovery, and optional TUI observer.

**Strategy**: 3-phase approach that decouples **translation risk** (Bash → Go) from **architectural risk** (hook model → daemon model).

### Why This Approach Beats Alternatives

| Alternative                          | Risk | Timeline | Outcome                        |
| ------------------------------------ | ---- | -------- | ------------------------------ |
| **V2 Plan** (tmux scraping)          | High | 12 weeks | Fragile, no supervision        |
| **V3 Hook-Only** (skip daemon)       | Low  | 6 weeks  | No crash recovery, no scaling  |
| **Big Bang** (Go daemon immediately) | High | 9 weeks  | 2 simultaneous risks           |
| **This Plan** (phased hybrid)        | Low  | 9 weeks  | Proven patterns, safe rollback |

**Key Insight**: The current Bash system WORKS. Migrating to Go removes one problem (maintainability). Migrating to daemon removes another problem (supervision). These are **orthogonal risks** that should be tackled sequentially, not simultaneously.

---

## V2 → V3 Evolution

### V2 Flaws (Original Plan)

**Fatal Architectural Flaws**:

1. **Single Process Failure Domain** - TUI crash kills all sessions
2. **No Supervision Layer** - No health monitoring or stuck detection
3. **Volatile State** - Everything in TUI memory, not persistent
4. **No Crash Recovery** - No GUPP (Graceful Unexpected Process Poweroff)

**Technical Debt**:

- Tmux scraping (fragile, parsing overhead)
- Synchronous hook blocking (sessions wait on TUI)
- No graceful degradation

### V3 Breakthrough: Hook IPC

**Discovery**: Claude Code hooks already provide structured events via STDIN/STDOUT JSON. We don't need tmux scraping.

**Hook Events Available**:

```json
{
  "tool_name": "Task",
  "tool_input": {"model": "sonnet", "prompt": "AGENT: python-pro..."},
  "tool_response": {...},
  "session_id": "abc123",
  "timestamp": 1234567890
}
```

**Implication**: We can build a daemon that:

1. Listens on Unix socket for hook events
2. Maintains persistent session state
3. Provides TUI as observer (not controller)
4. Survives crashes independently

---

## Performance SLA (NEW)

**Baseline Requirements:**

- **Bash hooks current performance:** Measured in pre-work (GOgent-000)
- **Go hooks target:** ≤ Bash performance or <5ms p99 latency (whichever is lower)
- **Acceptable degradation:** +20% over Bash acceptable if ≤10ms
- **Unacceptable:** >10ms p99 latency

**Measurement:**

- 100-event benchmark corpus
- Measure: avg, p50, p95, p99 latency
- Tools: `time` command or Go benchmarking

**Fallback:**
If Go >10ms: Profile, optimize, or accept and document limitation.

---

## Error Message Standards (NEW)

**All validation errors MUST follow this format:**

```
[component] What happened. Why it was blocked. How to fix.
```

**Examples:**

**Good:**

```
[validate-routing] Task(opus) blocked. Einstein requires GAP document workflow for cost control. Generate GAP: .claude/tmp/einstein-gap-{timestamp}.md, then run /einstein.
```

**Bad:**

```
Task blocked.
```

**Components:**

- `[validate-routing]`: Which hook/component
- `Task(opus) blocked`: What action failed
- `Einstein requires GAP...`: Why (policy/rule)
- `Generate GAP: ...`: How to fix (actionable)

**Logging:**
All errors logged to `~/.gogent/hooks.log` with:

- Timestamp (ISO 8601)
- Level (ERROR, WARN, INFO)
- Component name
- Full message

---

## Logging Strategy (NEW)

**Log Destinations:**

| Component        | Log File                               | Format     | Rotation             |
| ---------------- | -------------------------------------- | ---------- | -------------------- |
| All hooks        | `~/.gogent/hooks.log`                  | JSON lines | Keep last 1000 lines |
| Violations       | `/tmp/claude-routing-violations.jsonl` | JSON lines | Keep last 1000 lines |
| Daemon (Phase 1) | `~/.gogent/daemon.log`                 | JSON lines | Daily, keep 7 days   |
| TUI (Phase 2)    | `~/.gogent/tui.log`                    | JSON lines | Keep last 500 lines  |

**Log Format (JSON Lines):**

```json
{"ts":"2026-01-15T14:32:15Z","level":"ERROR","component":"validate-routing","msg":"Task(opus) blocked","session_id":"abc123","details":{...}}
```

**Implementation:**

- Package: `internal/logger` (structured logging)
- Library: Use `log/slog` (Go 1.21+) or `zerolog`
- Rotation: Manual (Phase 0), logrotate (Phase 1)

**User Debugging:**

```bash
# View recent errors
tail -n 50 ~/.gogent/hooks.log | jq 'select(.level=="ERROR")'

# View session violations
grep session-abc123 /tmp/claude-routing-violations.jsonl | jq .
```

---

## Schema Version Validation (NEW)

**Problem:** routing-schema.json format may change, breaking Go code.

**Solution:**

1. **Schema Version Field** (add to routing-schema.json):

```json
{
  "version": "2.1.0",
  "schema_version": "1.0",
  ...
}
```

2. **Go Validation** (pkg/config/loader.go):

```go
const EXPECTED_SCHEMA_VERSION = "1.0"

func LoadRoutingSchema() (*routing.Schema, error) {
    schema, err := parseSchema()
    if err != nil {
        return nil, err
    }

    if schema.SchemaVersion != EXPECTED_SCHEMA_VERSION {
        return nil, fmt.Errorf(
            "[config] Schema version mismatch. Expected %s, got %s. Update gogent binaries or routing-schema.json.",
            EXPECTED_SCHEMA_VERSION,
            schema.SchemaVersion,
        )
    }

    return schema, nil
}
```

3. **Failure Mode**: Fail fast at startup, log clear error, exit.

---

## File Path Standards (NEW)

**Problem:** Hardcoded /tmp paths may fail on noexec systems or multi-user environments.

**Solution: XDG Base Directory Standard with fallback**

**Tier file:**

```go
// OLD: /tmp/claude-current-tier
// NEW:
func GetTierFilePath() string {
    // Try XDG_RUNTIME_DIR (systemd standard)
    if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
        return filepath.Join(xdg, "gogent", "current-tier")
    }

    // Fallback: ~/.cache/gogent
    home, _ := os.UserHomeDir()
    cacheDir := filepath.Join(home, ".cache", "gogent")
    os.MkdirAll(cacheDir, 0755)
    return filepath.Join(cacheDir, "current-tier")
}
```

**Violations log:**

```go
// OLD: /tmp/claude-routing-violations.jsonl
// NEW:
func GetViolationsLogPath() string {
    // Try XDG_CACHE_HOME
    if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
        return filepath.Join(xdg, "gogent", "routing-violations.jsonl")
    }

    // Fallback: ~/.cache/gogent
    home, _ := os.UserHomeDir()
    cacheDir := filepath.Join(home, ".cache", "gogent")
    os.MkdirAll(cacheDir, 0755)
    return filepath.Join(cacheDir, "routing-violations.jsonl")
}
```

**Session state (Phase 1):**

```go
// NEW: ~/.local/share/gogent/sessions/
func GetSessionDir() string {
    if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
        return filepath.Join(xdg, "gogent", "sessions")
    }

    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".local", "share", "gogent", "sessions")
}
```

**Benefits:**

- Respects system standards
- Works on noexec /tmp
- Multi-user safe
- Survives /tmp cleanup

---

## Pre-Work: Baseline Measurement (NEW)

**BEFORE contractor starts, run GOgent-000.**

### Purpose

Establish performance baseline and test corpus for regression testing.

### Tasks

**1. Benchmark Current Bash Hooks**

```bash
# Create benchmark script
cat > /tmp/benchmark-bash-hooks.sh << 'EOF'
#!/bin/bash
# Run 100 events through validate-routing.sh

HOOK="$HOME/.claude/hooks/validate-routing.sh"
CORPUS="/tmp/event-corpus-sample.json"

# Sample event
EVENT='{"tool_name":"Task","tool_input":{"model":"sonnet","prompt":"AGENT: python-pro"},"session_id":"bench-123"}'

# Warm-up
for i in {1..10}; do
    echo "$EVENT" | $HOOK > /dev/null 2>&1
done

# Benchmark
START=$(date +%s%N)
for i in {1..100}; do
    echo "$EVENT" | $HOOK > /dev/null 2>&1
done
END=$(date +%s%N)

TOTAL_MS=$(( (END - START) / 1000000 ))
AVG_MS=$(( TOTAL_MS / 100 ))

echo "Total: ${TOTAL_MS}ms"
echo "Average: ${AVG_MS}ms per event"
EOF

chmod +x /tmp/benchmark-bash-hooks.sh
/tmp/benchmark-bash-hooks.sh
```

**2. Capture Production Event Corpus**

```bash
# Add temporary logger hook
cat > ~/.claude/hooks/zzz-corpus-logger.sh << 'EOF'
#!/bin/bash
# Log all events to corpus file

CORPUS="$HOME/.cache/gogent/event-corpus.jsonl"
mkdir -p "$(dirname "$CORPUS")"

# Read stdin
stdin_content=$(cat)

# Append to corpus
echo "$stdin_content" >> "$CORPUS"

# Pass through (don't block)
echo "$stdin_content"
EOF

chmod +x ~/.claude/hooks/zzz-corpus-logger.sh

# Run Claude Code for 24hrs (or until 100 events captured)
# Then curate to 100 diverse events
```

**3. Document Baseline**
Create `migration_plan/BASELINE.md`:

```markdown
# Performance Baseline (Bash Hooks)

**Date:** 2026-01-XX
**System:** Linux 6.18.4-2-cachyos, 16GB RAM, AMD Ryzen

## Latency Measurements (100 events)

| Hook                   | Avg    | p50    | p95    | p99    |
| ---------------------- | ------ | ------ | ------ | ------ |
| validate-routing.sh    | 3.2ms  | 2.8ms  | 4.5ms  | 5.1ms  |
| session-archive.sh     | 12.5ms | 11.2ms | 15.3ms | 18.7ms |
| sharp-edge-detector.sh | 1.8ms  | 1.5ms  | 2.3ms  | 2.9ms  |

## Event Corpus

**File:** test/fixtures/event-corpus.json
**Events:** 100 total

- 25 Task events (10 sonnet, 8 haiku, 5 opus, 2 external)
- 20 Read events
- 15 Write events
- 15 Edit events
- 10 Bash events
- 10 Glob events
- 5 Grep events

## SLA Definition

**Target:** Go hooks ≤ Bash latency
**Acceptable:** +20% (validate-routing <6.1ms p99)
**Unacceptable:** >10ms p99
```

**Success Criteria:**

- [ ] BASELINE.md exists with latency measurements
- [ ] test/fixtures/event-corpus.json exists with 100 diverse events
- [ ] Events cover all tool types and tiers
- [ ] Corpus is curated (duplicates removed, edge cases included)

---

## Phase 0: Go Translation (Weeks 1-3 + 1 day pre-work)

**Objective**: 1:1 translation of Bash hooks to Go, **zero architectural changes**.

### Why Start Here?

**Benefits**:

- Removes Bash dependency
- Type safety (catch errors at compile time)
- Faster execution (<2ms vs <5ms)
- Easier to maintain and extend
- **Zero architectural risk** (same input/output)

**Rollback Plan**: Can run Go and Bash in parallel during testing, revert if issues found.

### What Gets Translated?

#### 1. validate-routing.sh → cmd/gogent-validate/main.go

**Current Behavior** (400 lines):

- Parses tool events from STDIN (JSON)
- Implements escape hatches (`--force-tier`, `--force-delegation`)
- Complexity-based routing (reads scout_metrics.json)
- Tool permission checks (tier restrictions)
- Task delegation validation:
  - Einstein/Opus blocking (GAP-003b)
  - Model mismatch warnings
  - Delegation ceiling enforcement (GAP-007)
  - Subagent_type enforcement
- Logs all violations to violations.jsonl
- Outputs JSON to STDOUT (allow/block/warn)

**Go Implementation**:

```go
// cmd/gogent-validate/main.go
package main

import (
    "encoding/json"
    "io"
    "os"
    "time"

    "github.com/yourusername/gogent-fortress/pkg/routing"
    "github.com/yourusername/gogent-fortress/pkg/config"
    "github.com/yourusername/gogent-fortress/internal/logger"
)

func main() {
    // Initialize logger
    logger.Init(logger.Config{
        File:  config.GetLogPath(),
        Level: "INFO",
    })

    // Set stdin timeout (30 seconds)
    stdin := make(chan []byte, 1)
    go func() {
        data, _ := io.ReadAll(os.Stdin)
        stdin <- data
    }()

    select {
    case data := <-stdin:
        processEvent(data)
    case <-time.After(30 * time.Second):
        logger.Error("validate-routing", "Stdin timeout after 30s", nil)
        outputError("Stdin timeout")
        os.Exit(1)
    }
}

func processEvent(data []byte) {
    // Load schemas
    schema, err := config.LoadRoutingSchema()
    if err != nil {
        logger.Error("validate-routing", "Failed to load routing schema", err)
        outputError(err.Error())
        os.Exit(1)
    }

    agents, err := config.LoadAgentsIndex()
    if err != nil {
        logger.Error("validate-routing", "Failed to load agents index", err)
        outputError(err.Error())
        os.Exit(1)
    }

    // Parse event from STDIN
    event, err := routing.ParseToolEvent(data)
    if err != nil {
        logger.Error("validate-routing", "Failed to parse event", err)
        outputError("Invalid JSON input")
        os.Exit(1)
    }

    // Validate
    result := routing.Validate(event, schema, agents)

    // Output to STDOUT
    output, _ := json.Marshal(result)
    os.Stdout.Write(output)
}

func outputError(msg string) {
    result := routing.ValidationResult{
        HookSpecificOutput: &routing.HookOutput{
            HookEventName:     "PreToolUse",
            AdditionalContext: "[validate-routing] " + msg,
        },
    }
    output, _ := json.Marshal(result)
    os.Stdout.Write(output)
}
```

**Package Structure**:

```
pkg/routing/
├── schema.go         # RoutingSchema, TierConfig, AgentIndex structs
├── events.go         # ToolEvent, TaskInput parsing
├── validation.go     # Core validation orchestrator
├── escape.go         # --force-tier, --force-delegation
├── complexity.go     # Scout metrics, tier calculation
├── permissions.go    # Tool permission checks
├── task.go           # Task delegation validation
├── opus.go           # Einstein/Opus blocking
├── ceiling.go        # Delegation ceiling enforcement
├── subagent.go       # Subagent_type validation
└── logger.go         # Violation logging to JSONL
```

_(Full implementation details in Phase 0 Tickets document)_

#### 2. session-archive.sh → cmd/gogent-archive/main.go

**Current Behavior** (111 lines):

- Reads session info from STDIN (JSON)
- Counts session metrics (tools, errors, violations)
- Generates handoff markdown document
- Extracts pending learnings from JSONL
- Summarizes routing violations
- Archives files (transcript, learnings, violations)
- Outputs confirmation to STDOUT

_(Full implementation in separate tickets document)_

#### 3. sharp-edge-detector.sh → cmd/gogent-sharp-edge/main.go

**Current Behavior** (105 lines):

- Parses tool response from STDIN (JSON)
- Detects errors (exit codes, error keywords)
- Logs to error-patterns.jsonl
- Counts recent failures on same file (5min window)
- If ≥3 failures: logs to pending-learnings.jsonl and blocks
- If 2 failures: warns

_(Full implementation in separate tickets document)_

### Success Criteria (Phase 0)

- [ ] All 3 Bash hooks replaced with Go binaries
- [ ] Go binaries produce **identical JSON output** to Bash versions
- [ ] Performance: Go execution ≤ Bash baseline (measured in GOgent-000)
- [ ] Integration tests pass (100 real events from corpus)
- [ ] Regression tests pass (output diff = 0)
- [ ] Can roll back to Bash if issues found
- [ ] All logs written to ~/.gogent/hooks.log
- [ ] Schema version validated on startup

### Testing Strategy (Phase 0)

**Unit Tests**: Every function in pkg/ has tests
**Integration Tests**: Real Claude Code events → compare Go vs Bash output
**Regression Tests**: 100-event corpus → diff outputs (byte-for-byte match)
**Performance Benchmarks**: Latency measurements (must be ≤ baseline)
**Parallel Testing**: Run Go and Bash simultaneously for 24hrs, monitor discrepancies

---

## Phase 1: Daemon Foundation (Weeks 4-6)

**Objective**: Build persistent daemon with supervision, **without breaking existing hooks**.

### Gastown Patterns Applied

**From Gastown's proven architecture**:

1. **Persistent Daemon Process** (replaces "run TUI or die")
   - Unix socket listener (`~/.local/share/gogent/hook.sock`)
   - Session map (in-memory + persistent)
   - Event routing to handlers
   - Graceful shutdown (SIGTERM, SIGINT)

2. **Session State Persistence** (GUPP - Graceful Unexpected Process Poweroff)
   - Write `~/.local/share/gogent/sessions/<id>.json` on every event
   - Read on daemon startup
   - PID tracking (verify process still alive)
   - Recover active sessions after crash

3. **Supervision Loop** (Witness pattern)
   - 30s tick
   - Check `session.LastEvent` timestamp
   - Detect stuck (>30min idle)
   - Log alerts
   - Optional: kill stuck sessions

### Why Gastown Patterns Work

**GUPP (Graceful Unexpected Process Poweroff):**

- **Problem:** Application crashes lose in-memory state
- **Solution:** Persist state to disk on every mutation
- **Cost:** File I/O overhead (~1ms per write)
- **Benefit:** Complete recovery after crash
- **Gastown Evidence:** 2+ years production, zero data loss from crashes

**Witness (Supervision Pattern):**

- **Problem:** Long-running processes can hang indefinitely
- **Solution:** Periodic health check loop, timeout detection
- **Cost:** Negligible (30s tick, <0.1% CPU)
- **Benefit:** Stuck processes detected automatically
- **Gastown Evidence:** Detects hung SSH connections, dead subprocesses

**Unix Sockets:**

- **Problem:** Need IPC between daemon and hooks/TUI
- **Solution:** Local domain sockets (not network TCP)
- **Cost:** <0.1ms per message
- **Benefit:** Fast, secure, no network exposure
- **Gastown Evidence:** Handles 1000+ messages/sec

### Architecture

```
┌─────────────────────────────────────────────────────┐
│ Claude Code Session                                 │
│ ┌─────────────┐                                     │
│ │  PreToolUse │ ──STDIN──> gogent-validate           │
│ └─────────────┘                 │                   │
│                                 ▼                   │
│                         Validation logic            │
│                                 │                   │
│                                 ▼                   │
│                     Send to daemon (optional)       │
│                                 │                   │
└─────────────────────────────────┼───────────────────┘
                                  │
                                  ▼
                    ┌──────────────────────────┐
                    │   GOgent Daemon Process   │
                    │  ~/.local/share/gogent/   │
                    │       hook.sock          │
                    │                          │
                    │  Session State (persist) │
                    │  ├─ session-abc123.json  │
                    │  ├─ session-def456.json  │
                    │  └─ session-ghi789.json  │
                    │                          │
                    │  Supervision Loop        │
                    │  └─ 30s tick, stuck det. │
                    └─────────┬────────────────┘
                              │
                              ▼
                    ┌──────────────────────────┐
                    │   TUI Observer(s)        │
                    │  (OPTIONAL - v1.1)       │
                    │                          │
                    │  Read-only state view    │
                    │  Send commands (kill)    │
                    └──────────────────────────┘
```

### Hook Integration (Phase 1)

**Before (Phase 0)**: Hooks do all work directly
**After (Phase 1)**: Hooks send to daemon socket, daemon handles logic

```go
// cmd/gogent-validate/main.go refactor
func processEvent(data []byte) {
    // Parse event
    event, _ := routing.ParseToolEvent(data)

    // Send to daemon socket (if daemon running)
    // Non-blocking: if daemon down, hooks still work
    go func() {
        conn, err := net.Dial("unix", config.GetDaemonSocketPath())
        if err == nil {
            defer conn.Close()
            json.NewEncoder(conn).Encode(map[string]interface{}{
                "type":       "tool_event",
                "session_id": event.SessionID,
                "timestamp":  time.Now().Unix(),
                "event":      event,
            })
        }
    }()

    // Still do validation (hooks remain functional)
    schema, _ := config.LoadRoutingSchema()
    agents, _ := config.LoadAgentsIndex()
    result := routing.Validate(event, schema, agents)

    output, _ := json.Marshal(result)
    os.Stdout.Write(output)
}
```

**Key Design Decision**: Hooks don't DEPEND on daemon. Daemon is purely additive (supervision, TUI). Hooks work standalone.

### Daemon API

**Unix Socket Protocol** (`~/.local/share/gogent/hook.sock`):

```json
// Event sent by hooks
{
  "type": "tool_event",
  "session_id": "abc123",
  "timestamp": 1234567890,
  "event": {...}  // Original ToolEvent
}

// Response (optional, hooks don't wait)
{
  "status": "received",
  "session_updated": true
}
```

**State Files** (`~/.local/share/gogent/sessions/<id>.json`):

```json
{
  "session_id": "abc123",
  "pid": 12345,
  "started_at": 1234567890,
  "last_event_at": 1234567999,
  "tool_count": 42,
  "violation_count": 3,
  "status": "active" // active, stuck, completed
}
```

### Graceful Shutdown (NEW)

**SIGTERM/SIGINT Handler:**

```go
// cmd/gogent-daemon/main.go
func main() {
    daemon := NewDaemon()

    // Setup signal handler
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

    go func() {
        <-sigChan
        logger.Info("daemon", "Received shutdown signal, cleaning up...")

        // Flush all session state
        daemon.FlushAllSessions()

        // Close unix socket
        daemon.socket.Close()

        // Exit
        os.Exit(0)
    }()

    daemon.Run()
}

func (d *Daemon) FlushAllSessions() {
    for _, session := range d.sessions {
        session.WriteState()
    }
}
```

### Success Criteria (Phase 1)

- [ ] Daemon starts and listens on Unix socket
- [ ] Sessions persist across daemon restarts (GUPP)
- [ ] Supervision loop detects stuck sessions (>30min idle)
- [ ] Daemon crash doesn't affect active Claude Code sessions
- [ ] Hooks remain functional if daemon is down
- [ ] Performance: Socket IPC adds <1ms latency
- [ ] Graceful shutdown handler implemented
- [ ] Corrupted session files handled (skip and log)

---

## Phase 2: TUI Observer (DEFERRED TO v1.1)

**Decision**: Phase 2 (TUI) is **optional** and can be deferred to v1.1.

**Rationale:**

- Phase 0 and 1 provide complete functionality (hooks + daemon)
- TUI is pure UX polish, not core system
- Saves 3 weeks for faster MVP delivery
- Can be developed in parallel with production use of Phases 0-1

**If implemented in v1.1:**

- Bubbletea-based observer
- Read-only state view from daemon
- Dwarf Fortress aesthetic
- Multiple TUI connections supported
- TUI crash doesn't affect daemon

---

## Risk Analysis

### Phase 0 Risks (LOW - post-review)

| Risk                        | Probability | Impact | Mitigation                                              |
| --------------------------- | ----------- | ------ | ------------------------------------------------------- |
| Go output differs from Bash | Low         | High   | Regression tests, parallel testing, GOgent-000 baseline |
| Performance regression      | Low         | Medium | Benchmarks, <5ms requirement, measured baseline         |
| JSON parsing bugs           | Low         | High   | Unit tests, real event corpus (100 events)              |
| Circular dependencies       | Mitigated   | High   | Fixed in ticket ordering (GOgent-004 split)             |

**Overall Phase 0 Risk**: **LOW** (1:1 translation, proven patterns, critical issues fixed)

### Phase 1 Risks (MEDIUM)

| Risk                       | Probability | Impact | Mitigation                                                  |
| -------------------------- | ----------- | ------ | ----------------------------------------------------------- |
| Socket IPC adds latency    | Low         | Medium | Async send, hooks don't wait                                |
| GUPP recovery fails        | Low         | High   | Integration tests, manual crash tests, skip corrupted files |
| Session PID tracking fails | Low         | High   | Test on Linux, validate /proc                               |
| Daemon becomes bottleneck  | Low         | High   | Goroutines per handler, no blocking                         |

**Overall Phase 1 Risk**: **MEDIUM** (new architecture, but Gastown-proven)

### Phase 2 Risks (DEFERRED)

TUI deferred to v1.1, risk postponed.

---

## Success Criteria (Overall)

### Performance

- [ ] Hook execution: ≤ Bash baseline (measured in GOgent-000)
- [ ] Socket IPC overhead: <1ms (Phase 1)
- [ ] Target: <5ms p99 latency for all hooks

### Reliability

- [ ] Daemon survives crashes (GUPP recovery)
- [ ] Sessions persist across daemon restarts
- [ ] Hooks work without daemon (graceful degradation)
- [ ] Corrupted session files handled gracefully

### Quality

- [ ] 100% unit test coverage for pkg/ (target: ≥80% actual)
- [ ] Integration tests pass (real Claude Code events)
- [ ] Regression tests pass (output diff = 0 except timestamps)
- [ ] Performance benchmarks pass (≤ baseline)

### Operational

- [ ] Installation script (compile + symlink)
- [ ] Systemd service (auto-start daemon, Phase 1)
- [ ] Documentation (architecture, operations, troubleshooting)
- [ ] Rollback plan documented and tested
- [ ] WSL2 compatibility verified

---

## Decision Rationale

### Why 3 Phases?

**De-risks by separating concerns**:

1. **Phase 0**: Prove Go can replace Bash (translation risk only)
2. **Phase 1**: Prove daemon architecture works (architectural risk only)
3. **Phase 2**: Add polish (UX risk only) - **DEFERRED**

**Alternative (Big Bang)**: Go daemon immediately = 2 simultaneous risks = higher chance of failure.

### Why Go?

- **Type safety**: Catch errors at compile time
- **Performance**: <2ms execution (vs <5ms Bash)
- **Concurrency**: Goroutines for daemon handlers
- **Single binary**: Easy distribution
- **Ecosystem**: JSON parsing, Unix sockets, testing tools
- **Modern**: Go 1.21+ with generics, slog, context

### Why Daemon?

**Problems solved**:

- Crash recovery (GUPP)
- Supervision (detect stuck sessions)
- Scaling (multiple sessions)
- Observability (future TUI, logs)

**Gastown proves it works**: 2+ years in production.

### Why Not V2 Plan?

**V2 Fatal Flaws**:

- Single process failure domain (TUI crash = all sessions die)
- No supervision (can't detect stuck sessions)
- Volatile state (no GUPP)
- Tmux scraping (fragile, overhead)

**V3 Fixes All**:

- Daemon survives TUI crash
- Supervision loop (Witness pattern)
- Persistent state (GUPP)
- Hook IPC (structured events)

---

## Timeline Summary

| Phase           | Duration        | Risk      | Deliverable                       |
| --------------- | --------------- | --------- | --------------------------------- |
| **Pre-Work**    | 1 day           | LOW       | Baseline measurement (GOgent-000) |
| **Phase 0**     | 3 weeks         | LOW       | Go hooks (1:1 Bash replacement)   |
| **Phase 1**     | 3 weeks         | MEDIUM    | Daemon (supervision + GUPP)       |
| **Phase 2**     | DEFERRED        | -         | TUI (v1.1 feature)                |
| **Total (MVP)** | 6 weeks + 1 day | De-risked | Production-ready system           |

---

## Next Steps

1. ✅ **Review approved** (Staff Architect sign-off received)
2. **Run GOgent-000** (pre-work: baseline + corpus) - 1 day
3. **Create tickets** from migration_plan/finalised/gogent_plan_tickets_v3_phase0_FINAL.md
4. **Assign contractor** (tickets ready for Monday start)
5. **Begin Sprint 0A** (Week 1: Project setup + routing translation)

---

## Appendix A: Complete Schema Definitions

See separate file: `migration_plan/finalised/SCHEMA_COMPLETE.go`

Contains all struct definitions with complete nesting (400+ lines).

---

## Appendix B: Glossary

**Terms used in this plan:**

| Term                   | Definition                                                                    |
| ---------------------- | ----------------------------------------------------------------------------- |
| **Complexity Score**   | Numeric value (0-100) from scout metrics indicating task difficulty           |
| **Delegation Ceiling** | Maximum tier allowed for Task() spawning (haiku/haiku_thinking/sonnet)        |
| **Scout Metrics**      | JSON output from haiku-scout containing file count, LoC, estimated tokens     |
| **GUPP**               | Graceful Unexpected Process Poweroff - Gastown pattern for crash recovery     |
| **Witness Pattern**    | Supervision loop that monitors process health (Gastown pattern)               |
| **Hook IPC**           | Inter-Process Communication between hooks and daemon via Unix sockets         |
| **XDG**                | XDG Base Directory Specification - Linux standard for config/cache/data paths |
| **p99 latency**        | 99th percentile latency - 99% of events complete within this time             |

---

**Document Status**: ✅ APPROVED FOR IMPLEMENTATION
**Last Updated**: 2026-01-15
**Approved By**: Staff Solutions Architect
**Critical Review Applied**: Yes (V3.0 → V3.1)
**Implementation Start**: After GOgent-000 complete
