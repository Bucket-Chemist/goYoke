# CLI Package - Claude Subprocess Manager

This package provides a Go subprocess manager for embedding Claude CLI directly in applications. It handles process lifecycle, NDJSON stream communication, and event parsing.

## Overview

The Claude CLI supports NDJSON (newline-delimited JSON) streaming via the `--input-format stream-json` and `--output-format stream-json` flags. This package wraps that functionality in a thread-safe Go interface.

## Components

### ClaudeProcess

Main process manager that handles:
- Process lifecycle (start, stop, restart)
- NDJSON stdin/stdout/stderr pipes
- Event and error channels
- Thread-safe message sending
- Running state tracking
- Graceful shutdown with timeout

### Config

Configuration for process initialization:
- `ClaudePath` - Path to claude binary (default: "claude")
- `SessionID` - Explicit session ID (auto-generated if empty)
- `SettingsPath` - Custom settings.json path
- `WorkingDir` - Working directory for subprocess
- `Verbose` - Enable verbose output (--verbose flag)
- `IncludePartial` - Include partial messages for streaming

### NDJSONReader/Writer

Thread-safe NDJSON stream utilities:
- `NDJSONReader` - Reads newline-delimited JSON from io.Reader
- `NDJSONWriter` - Writes newline-delimited JSON to io.Writer
- Custom buffer size (1MB) for long Claude responses
- Mutex-protected writes for concurrent access

## Usage

### Basic Example

```go
package main

import (
    "fmt"
    "time"

    "github.com/Bucket-Chemist/GOgent-Fortress/internal/cli"
)

func main() {
    // Create configuration
    cfg := cli.Config{
        ClaudePath: "claude",
        Verbose:    true,
    }

    // Create process
    proc, err := cli.NewClaudeProcess(cfg)
    if err != nil {
        panic(err)
    }

    // Start subprocess
    if err := proc.Start(); err != nil {
        panic(err)
    }
    defer proc.Stop()

    // Send message
    proc.Send("Hello Claude!")

    // Receive events
    for {
        select {
        case event := <-proc.Events():
            fmt.Printf("Event type: %s\n", event.Type)
            // Process event...

        case err := <-proc.Errors():
            fmt.Printf("Error: %v\n", err)

        case <-time.After(30 * time.Second):
            fmt.Println("Timeout")
            return
        }
    }
}
```

### Custom Session ID

```go
cfg := cli.Config{
    SessionID: "my-custom-session-123",
}

proc, _ := cli.NewClaudeProcess(cfg)
fmt.Println(proc.SessionID()) // "my-custom-session-123"
```

### Structured Messages

```go
msg := cli.UserMessage{
    Content: "Explain quantum computing",
}

proc.SendJSON(msg)
```

### Process Status

```go
if proc.IsRunning() {
    fmt.Println("Process is running")
}

// Graceful shutdown (5 second timeout)
proc.Stop()
```

## Architecture

### Process Lifecycle

1. **Create**: `NewClaudeProcess(cfg)` builds command with flags
2. **Start**: `Start()` spawns subprocess, starts event reader goroutine
3. **Communicate**: `Send()` writes to stdin, `Events()` reads from stdout
4. **Stop**: `Stop()` closes stdin, waits 5s, then SIGKILL if needed

### Thread Safety

- `Send()` and `SendJSON()` use mutex-protected writes
- `Events()` and `Errors()` return buffered channels
- `IsRunning()` uses mutex for state access
- Multiple goroutines can safely call all methods

### Graceful Shutdown

Stop sequence:
1. Close `done` channel to signal goroutines
2. Close stdin to send EOF to subprocess
3. Wait up to 5 seconds for clean exit
4. If timeout, send SIGKILL
5. Close event and error channels

### Event Reading

Background goroutine (`readEvents`):
1. Read NDJSON lines from stdout
2. Parse into Event structs
3. Send to buffered events channel (size 100)
4. Handle errors and EOF gracefully
5. Respect shutdown signal via done channel

### Error Handling

Errors are sent to the errors channel:
- Stderr output (prefixed with "stderr:")
- JSON parse errors (prefixed with "parse error:")
- Read errors (IO failures)
- Process exit errors

## Event Types (Placeholder)

This package uses placeholder `Event` and `UserMessage` types. Full event parsing is implemented in GOgent-114.

Current placeholder:
```go
type Event struct {
    Type    string                 `json:"type"`
    RawData map[string]interface{} `json:"-"`
}
```

## Testing

### Unit Tests

Run unit tests for streams and config:
```bash
go test ./internal/cli/...
```

### Integration Tests

Integration tests use a mock claude binary (`testdata/mock-claude`) that:
- Emits init event with session ID
- Echoes input messages as assistant events
- Handles stdin EOF gracefully

Build mock:
```bash
cd internal/cli/testdata
go build -o mock-claude mock-claude.go
```

Run integration tests:
```bash
go test -v ./internal/cli/...
```

### Race Detection

Always run with race detector during development:
```bash
go test -race ./internal/cli/...
```

## Channel Buffering

- **Events channel**: Buffer size 100 to prevent blocking on slow consumers
- **Errors channel**: Buffer size 10 for error bursts

This prevents the event reader goroutine from blocking and ensures events are not lost during processing delays.

## Timeouts

- **Graceful shutdown**: 5 seconds before SIGKILL
- **Read operations**: Non-blocking with select pattern
- Callers should implement their own timeouts when reading from channels

## Claude CLI Flags

The subprocess is started with these flags:

```bash
claude --print \
       --input-format stream-json \
       --output-format stream-json \
       --session-id <uuid> \
       [--verbose] \
       [--include-partial-messages] \
       [--settings <path>]
```

## Dependencies

- `github.com/google/uuid` - Session ID generation
- Standard library only for core functionality

## Related Tickets

- **GOgent-110 (TUI-CLI-01)**: This implementation
- **GOgent-114 (TUI-CLI-02)**: Full event type parsing
- **GOgent-115 (TUI-CLI-03)**: Conversation state tracking
- **TUI-CLI-04**: Error handling strategies
- **TUI-CLI-05**: Metrics and monitoring

## Future Enhancements

The following will be added in subsequent tickets:

1. **Full Event Parsing** (GOgent-114)
   - Complete event type definitions
   - System, assistant, tool use events
   - Partial message handling

2. **State Management** (GOgent-115)
   - Conversation history tracking
   - Turn management
   - Session persistence

3. **Error Recovery** (TUI-CLI-04)
   - Automatic restart on crash
   - Backoff strategies
   - Health checks

4. **Metrics** (TUI-CLI-05)
   - Message latency tracking
   - Token usage monitoring
   - Error rate metrics

## Notes

- This package is in `internal/` and cannot be imported by external projects
- The subprocess inherits the parent's environment
- Stdin writes are line-buffered (NDJSON requirement)
- Stdout/stderr are read with 1MB custom buffer for long responses
