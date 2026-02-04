# Timeout Handling (Bug #1920 Workaround)

## Problem

Claude CLI Bug #1920: The `{"type":"result",...}` event sometimes never arrives after tool execution, causing the TUI to block forever waiting for it in the `readEvents()` loop.

## Solution

Configurable timeout mechanism that triggers recovery when no events are received for a specified duration.

## Configuration

```go
type TimeoutConfig struct {
    // ResultTimeout is max time to wait for result after last activity
    ResultTimeout time.Duration

    // InactivityTimeout is max time with no events at all
    InactivityTimeout time.Duration
}
```

### Defaults

```go
DefaultTimeoutConfig() = TimeoutConfig{
    ResultTimeout:     5 * time.Minute,  // Tool execution can be slow
    InactivityTimeout: 2 * time.Minute,  // But silence is suspicious
}
```

### Usage

```go
// Use defaults
proc, err := cli.NewClaudeProcess(cli.Config{
    ClaudePath: "claude",
})

// Custom timeouts
proc, err := cli.NewClaudeProcess(cli.Config{
    ClaudePath: "claude",
    Timeout: cli.TimeoutConfig{
        ResultTimeout:     10 * time.Minute,
        InactivityTimeout: 3 * time.Minute,
    },
})

// Disable timeout (not recommended)
proc, err := cli.NewClaudeProcess(cli.Config{
    ClaudePath: "claude",
    Timeout: cli.TimeoutConfig{
        InactivityTimeout: 0,  // No timeout
    },
})
```

## Behavior

1. **Last Event Time Tracking**: `readEvents()` tracks `lastEventTime` on every successful event read
2. **Timeout Check**: Before each read attempt, checks if `time.Since(lastEventTime) > InactivityTimeout`
3. **Timeout Error**: If timeout exceeded, sends error to `errors` channel:
   ```
   timeout: no events for 2m0s
   ```
4. **Graceful Exit**: `readEvents()` goroutine returns, allowing TUI to recover

## Design Notes

- **100ms Context Timeout**: Short timeout for responsiveness during read operations
- **Inactivity Timeout**: Separate, longer timeout tracking time since LAST successful event
- **Generation Check**: Timeout errors only sent if goroutine matches current generation (prevents stale timeout errors after restart)
- **Non-blocking**: Error sent to buffered errors channel, doesn't block if channel is full

## Testing

See `internal/cli/timeout_test.go` and `internal/cli/subprocess_test.go`:

- `TestDefaultTimeoutConfig()`: Verifies default values
- `TestTimeoutConfig_CustomValues()`: Custom configuration
- `TestClaudeProcess_TimeoutErrorMessage()`: Error message format and duration
- `TestClaudeProcess_NoTimeoutDuringActiveStreaming()`: No false positives during normal operation
- `TestClaudeProcess_TimeoutDisabledWhenZero()`: Zero value disables timeout

## Trade-offs

| Timeout Value | Pros | Cons |
|---------------|------|------|
| Short (< 1 min) | Fast recovery from hangs | May timeout during legitimate slow tool execution |
| Medium (2-5 min) | Balanced | Default choice |
| Long (> 5 min) | Patient for very slow tools | User may force-quit before timeout |
| Disabled (0) | Never false positive | TUI hangs forever on Bug #1920 |

## Future Enhancements

1. **ResultTimeout**: Currently only InactivityTimeout is implemented. ResultTimeout would track time since last `{"type":"tool_result"}` event specifically.
2. **Adaptive Timeout**: Learn typical tool execution times and adjust timeout dynamically.
3. **Per-Tool Timeout**: Different timeouts for different tools (e.g., longer for `WebSearch`, shorter for `Read`).
4. **Timeout Event**: Structured timeout event instead of generic error string.

## Related

- Bug #1920: https://github.com/anthropics/claude-code/issues/1920
- `internal/cli/subprocess.go`: `readEvents()` implementation
- `internal/cli/events.go`: Event type definitions
