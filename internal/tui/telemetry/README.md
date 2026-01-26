# Telemetry File Watchers

Real-time file watching infrastructure for GOgent-Fortress TUI. Monitors telemetry JSONL files and emits typed events for consumption by UI components.

## Overview

This package provides three levels of abstraction:

1. **JSONLWatcher** - Generic JSONL file watcher with offset tracking
2. **TelemetryWatcher** - Multi-file watcher for all telemetry streams
3. **TelemetryAggregator** - High-level state management with event correlation

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    TelemetryAggregator                      │
│  ┌───────────────────────────────────────────────────────┐  │
│  │            Agent State Management                     │  │
│  │  • Correlate spawn → complete events                  │  │
│  │  • Track active vs completed agents                   │  │
│  │  • Calculate durations, success rates                 │  │
│  └───────────────────────────────────────────────────────┘  │
│                           ▲                                 │
│                           │                                 │
│  ┌────────────────────────┴──────────────────────────────┐  │
│  │              TelemetryWatcher                         │  │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌─────────┐  │  │
│  │  │lifecycle │ │decisions │ │ updates  │ │  collab │  │  │
│  │  │ watcher  │ │ watcher  │ │ watcher  │ │ watcher │  │  │
│  │  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬────┘  │  │
│  └───────┼────────────┼────────────┼──────────────┼──────┘  │
│          │            │            │              │         │
│          ▼            ▼            ▼              ▼         │
│      JSONLWatcher  JSONLWatcher  JSONLWatcher  JSONLWatcher│
│          │            │            │              │         │
│          ▼            ▼            ▼              ▼         │
│      ┌────────────────────────────────────────────────┐    │
│      │            fsnotify.Watcher                    │    │
│      └────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
              ┌──────────────────────────┐
              │   Telemetry JSONL Files   │
              │  • agent-lifecycle.jsonl  │
              │  • routing-decisions.jsonl│
              │  • routing-decision-upda..│
              │  • agent-collaborations..  │
              └──────────────────────────┘
```

## Components

### JSONLWatcher

Low-level watcher for a single JSONL file.

**Features:**
- Tracks file offset (only reads new content)
- Handles file creation (waits if file doesn't exist)
- Handles file truncation/rotation
- Buffered channels (capacity: 100) to handle bursts
- Graceful error handling (malformed lines don't crash)
- Thread-safe

**Usage:**
```go
watcher, _ := NewJSONLWatcher("/path/to/file.jsonl", parseFunc)
watcher.Start()
defer watcher.Stop()

for event := range watcher.Events() {
    // Process event
}
```

### TelemetryWatcher

Manages multiple file watchers for all telemetry streams.

**Watched Files:**
- `$XDG_DATA_HOME/gogent-fortress/agent-lifecycle.jsonl`
- `$XDG_DATA_HOME/gogent-fortress/routing-decisions.jsonl`
- `$XDG_DATA_HOME/gogent-fortress/routing-decision-updates.jsonl`
- `$XDG_DATA_HOME/gogent-fortress/agent-collaborations.jsonl`

**Usage:**
```go
watcher, _ := NewTelemetryWatcher()
watcher.Start()
defer watcher.Stop()

events := watcher.Events()
for {
    select {
    case event := <-events.AgentLifecycle:
        // Handle lifecycle event
    case decision := <-events.RoutingDecisions:
        // Handle routing decision
    case update := <-events.DecisionUpdates:
        // Handle outcome update
    case err := <-events.Errors:
        // Handle error
    }
}
```

### TelemetryAggregator

High-level state manager with event correlation.

**Features:**
- Correlates agent spawn → complete events
- Tracks active vs completed agents
- Calculates durations and success rates
- Thread-safe state access
- Aggregated statistics

**Usage:**
```go
agg, _ := NewTelemetryAggregator()
agg.Start()
defer agg.Stop()

// Query state
activeAgents := agg.GetActiveAgents()
stats := agg.Stats()
```

## Event Types

### AgentLifecycleEvent
```go
type AgentLifecycleEvent struct {
    EventID         string
    SessionID       string
    Timestamp       int64
    EventType       string  // "spawn" | "complete" | "error"
    AgentID         string
    ParentAgent     string
    Tier            string
    TaskDescription string
    DecisionID      string
    Success         *bool   // For complete/error events
    DurationMs      *int64  // For complete events
}
```

### RoutingDecision
```go
type RoutingDecision struct {
    DecisionID      string
    Timestamp       int64
    TaskDescription string
    SelectedTier    string
    SelectedAgent   string
    Confidence      float64
}
```

### DecisionOutcomeUpdate
```go
type DecisionOutcomeUpdate struct {
    DecisionID        string
    OutcomeSuccess    bool
    OutcomeDurationMs int64
    OutcomeCost       float64
    UpdateTimestamp   int64
}
```

## Performance Characteristics

### Efficiency
- **Offset tracking**: Only new content is read on each event
- **Buffered channels**: 100-event buffer prevents blocking hook writes
- **Minimal allocations**: Reuses buffers where possible
- **No polling**: Uses fsnotify for event-driven updates

### Scalability
- **Large files**: Only reads delta (current offset → end)
- **High throughput**: Buffered channels handle bursts
- **Multiple watchers**: Each watcher runs in separate goroutine

### Benchmarks
```
BenchmarkJSONLWatcher_SingleLine-8   1000   1.2ms/op
```

## Edge Cases Handled

### File Doesn't Exist
- Watches parent directory
- Starts watching file when created
- Reads content from beginning

### File Truncation
- Detects offset > file size
- Resets to beginning
- Continues reading new content

### Malformed JSON
- Parse errors logged but don't crash
- Continues processing subsequent lines
- Errors available on error channel

### Rapid Writes
- Buffered channels (100 capacity)
- Multiple lines per write event batched
- Non-blocking emission (drops if full)

### Graceful Shutdown
- Close done channel
- Wait for goroutines (sync.WaitGroup)
- Close fsnotify watcher
- Close all output channels

## Integration with GOgent-109

Uses telemetry types from `pkg/telemetry`:
- `AgentLifecycleEvent`
- `RoutingDecision`
- `DecisionOutcomeUpdate`

Uses path resolution from `pkg/config`:
- `GetAgentLifecyclePathWithProjectDir()`
- `GetRoutingDecisionsPathWithProjectDir()`
- `GetRoutingDecisionUpdatesPathWithProjectDir()`
- `GetCollaborationsPathWithProjectDir()`

## Testing

### Unit Tests
- File creation/deletion handling
- Offset tracking
- Malformed JSON handling
- File truncation
- Multiple rapid writes

### Integration Tests
- End-to-end with real telemetry files
- Multi-file watcher coordination
- Aggregator state correlation

### Running Tests
```bash
go test -v ./internal/tui/telemetry
go test -race ./internal/tui/telemetry  # Race detector
go test -bench=. ./internal/tui/telemetry  # Benchmarks
```

## Dependencies

- `github.com/fsnotify/fsnotify v1.7.0` - File system notifications
- `github.com/stretchr/testify v1.11.1` - Test assertions

## Future Enhancements (GOgent-115)

Next ticket will add:
- Agent tree visualization (parent → child relationships)
- Historical event replay
- Time-series metrics (agents/second, success rate over time)
- WebSocket integration for remote monitoring

## Verification

Acceptance criteria from TUI-TELEM-01:

- [x] TelemetryWatcher can watch multiple JSONL files
- [x] fsnotify integration for file change detection
- [x] Read new events from current file offset (not entire file)
- [x] Parse JSONL events and emit on typed channels
- [x] Handle file rotation/truncation gracefully
- [x] Thread-safe event emission
- [x] Graceful shutdown (close channels, stop goroutines)
- [x] Unit tests for file watching logic
- [x] Integration test: write to file, verify event received

## See Also

- `dev/will/migration_plan/tickets/TUI/TUI-TELEM-01.md` - Original ticket spec
- `pkg/telemetry/` - Telemetry event definitions
- `pkg/config/paths.go` - Path resolution
