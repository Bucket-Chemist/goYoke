# TUI-CLI-01: Claude Subprocess Manager

> **Estimated Hours:** 3.0
> **Priority:** P0 - Foundation
> **Dependencies:** None
> **Phase:** 1 - Foundation

---

## Description

Implement a Go subprocess manager for embedding Claude CLI directly in the TUI. The subprocess communicates via NDJSON (newline-delimited JSON) streams on stdin/stdout using Claude's `--input-format stream-json` and `--output-format stream-json` flags.

**Key Flags:**
```bash
claude --print --verbose \
       --input-format stream-json \
       --output-format stream-json \
       --include-partial-messages \
       --session-id <uuid>
```

---

## Tasks

### 1. Create Subprocess Manager

**File:** `internal/cli/subprocess.go`

```go
package cli

import (
    "bufio"
    "encoding/json"
    "io"
    "os/exec"
    "sync"
)

type ClaudeProcess struct {
    cmd       *exec.Cmd
    stdin     io.WriteCloser
    stdout    io.ReadCloser
    stderr    io.ReadCloser
    sessionID string
    events    chan Event
    errors    chan error
    done      chan struct{}
    mu        sync.Mutex
    running   bool
}

type Config struct {
    ClaudePath      string        // Path to claude binary (default: "claude")
    SessionID       string        // Explicit session ID (generated if empty)
    SettingsPath    string        // Custom settings.json path
    WorkingDir      string        // Working directory for claude
    Verbose         bool          // Enable verbose output
    IncludePartial  bool          // Include partial messages for streaming
}

func NewClaudeProcess(cfg Config) (*ClaudeProcess, error)

func (cp *ClaudeProcess) Start() error

func (cp *ClaudeProcess) Stop() error

func (cp *ClaudeProcess) Send(message string) error

func (cp *ClaudeProcess) SendJSON(msg UserMessage) error

func (cp *ClaudeProcess) Events() <-chan Event

func (cp *ClaudeProcess) Errors() <-chan error

func (cp *ClaudeProcess) IsRunning() bool

func (cp *ClaudeProcess) SessionID() string
```

### 2. Create Stream Reader/Writer

**File:** `internal/cli/streams.go`

```go
package cli

// NDJSONReader reads newline-delimited JSON from an io.Reader
type NDJSONReader struct {
    scanner *bufio.Scanner
}

func NewNDJSONReader(r io.Reader) *NDJSONReader

func (nr *NDJSONReader) Read() ([]byte, error)

// NDJSONWriter writes newline-delimited JSON to an io.Writer
type NDJSONWriter struct {
    writer io.Writer
    mu     sync.Mutex
}

func NewNDJSONWriter(w io.Writer) *NDJSONWriter

func (nw *NDJSONWriter) Write(data interface{}) error
```

### 3. Implement Event Reading Goroutine

```go
func (cp *ClaudeProcess) readEvents() {
    reader := NewNDJSONReader(cp.stdout)

    for {
        select {
        case <-cp.done:
            return
        default:
            data, err := reader.Read()
            if err != nil {
                if err != io.EOF {
                    cp.errors <- err
                }
                return
            }

            event, err := ParseEvent(data)
            if err != nil {
                cp.errors <- fmt.Errorf("parse error: %w", err)
                continue
            }

            cp.events <- event
        }
    }
}
```

### 4. Implement Graceful Shutdown

```go
func (cp *ClaudeProcess) Stop() error {
    cp.mu.Lock()
    defer cp.mu.Unlock()

    if !cp.running {
        return nil
    }

    close(cp.done)

    // Close stdin to signal EOF
    cp.stdin.Close()

    // Wait for process with timeout
    done := make(chan error, 1)
    go func() {
        done <- cp.cmd.Wait()
    }()

    select {
    case err := <-done:
        cp.running = false
        return err
    case <-time.After(5 * time.Second):
        cp.cmd.Process.Kill()
        cp.running = false
        return fmt.Errorf("process killed after timeout")
    }
}
```

---

## Files to Create

| File | Purpose |
|------|---------|
| `internal/cli/subprocess.go` | Process lifecycle management |
| `internal/cli/streams.go` | NDJSON read/write utilities |
| `internal/cli/subprocess_test.go` | Unit tests |
| `internal/cli/streams_test.go` | Stream tests |

---

## Acceptance Criteria

- [ ] `ClaudeProcess` can start claude subprocess with correct flags
- [ ] Messages can be sent via `Send()` method
- [ ] Events received on `Events()` channel
- [ ] Errors received on `Errors()` channel
- [ ] `Stop()` gracefully shuts down process
- [ ] Session ID is preserved and accessible
- [ ] Process status queryable via `IsRunning()`
- [ ] Thread-safe message sending (mutex on stdin writes)
- [ ] Unit tests for stream reading/writing
- [ ] Integration test with mock claude binary

---

## Test Strategy

### Unit Tests
- NDJSON reader parses valid JSON lines
- NDJSON reader handles malformed JSON gracefully
- NDJSON writer formats output correctly
- Config validation works

### Integration Tests
Create a mock claude binary for testing:

**File:** `internal/cli/testdata/mock-claude.go`

```go
// Mock claude that echoes input as events
package main

import (
    "bufio"
    "encoding/json"
    "fmt"
    "os"
)

func main() {
    // Emit init event
    fmt.Println(`{"type":"system","subtype":"init","session_id":"test-123"}`)

    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        var msg map[string]interface{}
        json.Unmarshal(scanner.Bytes(), &msg)

        // Echo as assistant event
        response := map[string]interface{}{
            "type": "assistant",
            "message": map[string]interface{}{
                "content": []map[string]string{
                    {"type": "text", "text": fmt.Sprintf("Echo: %v", msg["content"])},
                },
            },
        }
        data, _ := json.Marshal(response)
        fmt.Println(string(data))
    }
}
```

---

## Notes

- Buffer size for event channel: 100 (prevent blocking on slow consumers)
- Use `bufio.Scanner` with custom buffer size for long lines
- Handle partial messages specially (they have incremental text)
- Log stderr to a separate channel or file for debugging
