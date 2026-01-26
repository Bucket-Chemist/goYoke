# TUI-TELEM-01: Real-Time File Watchers

> **Estimated Hours:** 2.0
> **Priority:** P1 - Foundation
> **Dependencies:** TUI-INFRA-01
> **Phase:** 1 - Foundation

---

## Description

Implement file watching infrastructure to enable real-time updates in the TUI. The TUI needs to react to changes in telemetry JSONL files as hooks write to them.

**Files to Watch:**
- `agent-lifecycle.jsonl` - Agent spawn/complete events
- `routing-decisions.jsonl` - Routing decisions
- `routing-violations.jsonl` - Violations
- `tool-counter-*.log` - Tool count

---

## Tasks

### 1. Create Watcher Interface

**File:** `internal/tui/telemetry/watcher.go`

```go
package telemetry

import (
    "context"
    "time"

    tea "github.com/charmbracelet/bubbletea"
)

// WatchEvent represents a change detected in a telemetry file
type WatchEvent struct {
    Source    string      // File path
    EventType string      // "append", "modify", "create"
    Data      interface{} // Parsed data (type depends on source)
    Timestamp time.Time
}

// Watcher monitors telemetry files for changes
type Watcher struct {
    paths     map[string]string // name -> path
    watchers  map[string]*FileWatcher
    events    chan WatchEvent
    ctx       context.Context
    cancel    context.CancelFunc
}

func NewWatcher() *Watcher

func (w *Watcher) AddPath(name, path string) error

func (w *Watcher) Start() error

func (w *Watcher) Stop()

func (w *Watcher) Events() <-chan WatchEvent

// WatchCmd returns a tea.Cmd that listens for watch events
func (w *Watcher) WatchCmd() tea.Cmd {
    return func() tea.Msg {
        event := <-w.Events()
        return WatchEventMsg(event)
    }
}
```

### 2. Implement JSONL File Watcher

**File:** `internal/tui/telemetry/jsonl_watcher.go`

```go
package telemetry

import (
    "bufio"
    "encoding/json"
    "io"
    "os"

    "github.com/fsnotify/fsnotify"
)

// JSONLWatcher watches a JSONL file and emits new lines
type JSONLWatcher struct {
    path     string
    file     *os.File
    offset   int64
    watcher  *fsnotify.Watcher
    events   chan interface{}
    parser   func([]byte) (interface{}, error)
}

func NewJSONLWatcher(path string, parser func([]byte) (interface{}, error)) (*JSONLWatcher, error) {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return nil, err
    }

    jw := &JSONLWatcher{
        path:    path,
        watcher: watcher,
        events:  make(chan interface{}, 100),
        parser:  parser,
    }

    return jw, nil
}

func (jw *JSONLWatcher) Start() error {
    // Open file and seek to end
    file, err := os.Open(jw.path)
    if err != nil {
        if os.IsNotExist(err) {
            // File doesn't exist yet - wait for creation
            jw.offset = 0
        } else {
            return err
        }
    } else {
        jw.file = file
        // Seek to end - only watch new lines
        offset, _ := file.Seek(0, io.SeekEnd)
        jw.offset = offset
    }

    // Add file (or directory if file doesn't exist) to watcher
    dir := filepath.Dir(jw.path)
    if err := jw.watcher.Add(dir); err != nil {
        return err
    }

    go jw.watch()

    return nil
}

func (jw *JSONLWatcher) watch() {
    for {
        select {
        case event, ok := <-jw.watcher.Events:
            if !ok {
                return
            }

            if event.Name != jw.path {
                continue
            }

            if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
                jw.readNewLines()
            }

        case err, ok := <-jw.watcher.Errors:
            if !ok {
                return
            }
            // Log error but continue
            fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)
        }
    }
}

func (jw *JSONLWatcher) readNewLines() {
    if jw.file == nil {
        file, err := os.Open(jw.path)
        if err != nil {
            return
        }
        jw.file = file
    }

    // Seek to last read position
    jw.file.Seek(jw.offset, io.SeekStart)

    scanner := bufio.NewScanner(jw.file)
    for scanner.Scan() {
        line := scanner.Bytes()
        if len(line) == 0 {
            continue
        }

        if jw.parser != nil {
            parsed, err := jw.parser(line)
            if err == nil {
                jw.events <- parsed
            }
        } else {
            // Raw JSON
            var data interface{}
            if json.Unmarshal(line, &data) == nil {
                jw.events <- data
            }
        }
    }

    // Update offset
    jw.offset, _ = jw.file.Seek(0, io.SeekCurrent)
}

func (jw *JSONLWatcher) Events() <-chan interface{} {
    return jw.events
}

func (jw *JSONLWatcher) Stop() {
    jw.watcher.Close()
    if jw.file != nil {
        jw.file.Close()
    }
}
```

### 3. Create Aggregated Telemetry Watcher

**File:** `internal/tui/telemetry/aggregator.go`

```go
package telemetry

import (
    "github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
    pkgtel "github.com/Bucket-Chemist/GOgent-Fortress/pkg/telemetry"
)

// TelemetryAggregator watches all telemetry sources
type TelemetryAggregator struct {
    lifecycleWatcher *JSONLWatcher
    decisionWatcher  *JSONLWatcher
    violationWatcher *JSONLWatcher

    updates chan TelemetryUpdate
}

type TelemetryUpdate struct {
    Type string // "lifecycle", "decision", "violation"
    Data interface{}
}

// Lifecycle event message for TUI
type LifecycleEventMsg pkgtel.AgentLifecycleEvent

// Routing decision message for TUI
type RoutingDecisionMsg pkgtel.RoutingDecision

// Violation message for TUI
type ViolationMsg struct {
    Type    string
    Agent   string
    Message string
}

func NewTelemetryAggregator() (*TelemetryAggregator, error) {
    ta := &TelemetryAggregator{
        updates: make(chan TelemetryUpdate, 100),
    }

    // Lifecycle watcher
    lifecyclePath := config.GetAgentLifecyclePathWithProjectDir()
    lw, err := NewJSONLWatcher(lifecyclePath, parseLifecycleEvent)
    if err != nil {
        return nil, err
    }
    ta.lifecycleWatcher = lw

    // Decision watcher
    decisionPath := config.GetRoutingDecisionsPathWithProjectDir()
    dw, err := NewJSONLWatcher(decisionPath, parseRoutingDecision)
    if err != nil {
        return nil, err
    }
    ta.decisionWatcher = dw

    return ta, nil
}

func (ta *TelemetryAggregator) Start() error {
    if err := ta.lifecycleWatcher.Start(); err != nil {
        return err
    }
    if err := ta.decisionWatcher.Start(); err != nil {
        return err
    }

    // Forward events to unified channel
    go ta.forward()

    return nil
}

func (ta *TelemetryAggregator) forward() {
    for {
        select {
        case event := <-ta.lifecycleWatcher.Events():
            ta.updates <- TelemetryUpdate{Type: "lifecycle", Data: event}
        case event := <-ta.decisionWatcher.Events():
            ta.updates <- TelemetryUpdate{Type: "decision", Data: event}
        }
    }
}

func (ta *TelemetryAggregator) Updates() <-chan TelemetryUpdate {
    return ta.updates
}

// WatchCmd returns a tea.Cmd for Bubble Tea integration
func (ta *TelemetryAggregator) WatchCmd() tea.Cmd {
    return func() tea.Msg {
        update := <-ta.Updates()
        switch update.Type {
        case "lifecycle":
            if event, ok := update.Data.(*pkgtel.AgentLifecycleEvent); ok {
                return LifecycleEventMsg(*event)
            }
        case "decision":
            if event, ok := update.Data.(*pkgtel.RoutingDecision); ok {
                return RoutingDecisionMsg(*event)
            }
        }
        return nil
    }
}
```

---

## Files to Create

| File | Purpose |
|------|---------|
| `internal/tui/telemetry/watcher.go` | Base watcher interface |
| `internal/tui/telemetry/jsonl_watcher.go` | JSONL-specific watcher |
| `internal/tui/telemetry/aggregator.go` | Unified telemetry stream |
| `internal/tui/telemetry/watcher_test.go` | Unit tests |

---

## Dependencies

Add to `go.mod`:
```
github.com/fsnotify/fsnotify v1.7.0
```

---

## Acceptance Criteria

- [ ] `JSONLWatcher` detects new lines appended to file
- [ ] Watcher handles file creation (file doesn't exist on start)
- [ ] Watcher parses JSONL lines correctly
- [ ] `TelemetryAggregator` combines all telemetry sources
- [ ] `WatchCmd()` returns tea.Cmd for Bubble Tea integration
- [ ] Watcher only reads new lines (not full file on each change)
- [ ] Graceful handling of malformed JSON lines
- [ ] Unit tests with temp files

---

## Notes

- Use `fsnotify` for cross-platform file watching
- Buffer channel to prevent blocking hook writes
- Seek to end on start - don't replay historical events
- Handle file rotation/truncation gracefully
- Consider debouncing rapid writes
