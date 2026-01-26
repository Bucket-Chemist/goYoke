# TUI Streaming Fix - GAP Document

## Problem Statement
When user types a message in the gofortress TUI and hits Enter, the UI shows [streaming] but never receives a response from Claude. The process appears stuck indefinitely.

## Root Cause Analysis: TIMING, Not Message Format

**Critical Discovery:** The stream-json message format is 100% correct. The issue is stdin timing/synchronization.

---

## Proof: Working vs Non-Working Tests

### Test 1: FIFO (WORKS)

```bash
# Create named pipe for controlled timing
rm -f /tmp/claude-fifo
mkfifo /tmp/claude-fifo

# Start Claude reading from FIFO in background
timeout 30 claude --print --verbose --input-format stream-json --output-format stream-json < /tmp/claude-fifo > /tmp/stdout.log 2>/tmp/stderr.log &
PID=$!

# Wait for Claude to initialize and start reading
sleep 1

# NOW send the message (Claude is ready to read)
echo '{"type":"user","message":{"role":"user","content":[{"type":"text","text":"Say BANANA"}]}}' > /tmp/claude-fifo

# Wait for response
sleep 10
kill $PID 2>/dev/null
wait $PID 2>/dev/null

# Check output
cat /tmp/stdout.log
```

**Result:**
```json
{"type":"system","subtype":"hook_started",...}
{"type":"system","subtype":"hook_response",...}
{"type":"system","subtype":"init","tools":["Task","Bash","Read",...],...}
{"type":"assistant","message":{"content":[{"type":"text","text":"BANANA"}],...}}
{"type":"result","subtype":"success","result":"BANANA","total_cost_usd":0.062,...}
```

### Test 2: Simple Pipe (FAILS)

```bash
echo '{"type":"user","message":{"role":"user","content":[{"type":"text","text":"Say BANANA"}]}}' | timeout 20 claude --print --verbose --input-format stream-json --output-format stream-json
```

**Result:** Empty output, exit code 0

### Test 3: Subshell with Sleep (FAILS)

```bash
(
echo '{"type":"user","message":{"role":"user","content":[{"type":"text","text":"hi"}]}}'
sleep 15
) | timeout 20 claude --print --verbose --input-format stream-json --output-format stream-json
```

**Result:** Empty output

### Why FIFO Works But Pipes Don't

1. **Pipe behavior:** When the writing side of a pipe closes (echo completes), EOF is sent to the reading side
2. **Claude's behavior:** With `--print` mode, Claude may check stdin state during initialization
3. **FIFO behavior:** The FIFO stays "open" because we write to it AFTER Claude starts reading
4. **Race condition:** With pipes, by the time Claude is ready to read user input, stdin may already be at EOF

---

## Verified Working Configuration

### CLI Flags (All Required)

```go
args := []string{
    "--print",                        // Required for non-interactive mode
    "--verbose",                      // REQUIRED when using --output-format=stream-json
    "--input-format", "stream-json",  // Accept JSON messages on stdin
    "--output-format", "stream-json", // Emit JSON events on stdout
    "--session-id", sessionID,        // Optional but recommended
}
```

**Important:** `--verbose` is MANDATORY with `--output-format=stream-json`. Without it:
```
Error: When using --print, --output-format=stream-json requires --verbose
```

### Message Format (Verified Working)

```json
{
  "type": "user",
  "message": {
    "role": "user",
    "content": [
      {"type": "text", "text": "YOUR MESSAGE HERE"}
    ]
  }
}
```

This matches `cli.UserMessage` struct in `internal/cli/events.go:79-90`.

---

## Current Implementation Analysis

### subprocess.go: Process Creation (Lines 140-205)

```go
// Build command arguments - THIS IS CORRECT
args := []string{
    "--print",
    "--verbose",           // Required for stream-json output
    "--debug-to-stderr",   // Keeps stdout clean for JSON
    "--input-format", "stream-json",
    "--output-format", "stream-json",
    "--session-id", sessionID,
}

cmd := exec.Command(cfg.ClaudePath, args...)

return &ClaudeProcess{
    cmd:           cmd,
    events:        make(chan Event, 100),        // Buffered
    errors:        make(chan error, 10),
    restartEvents: make(chan RestartEvent, 10),
    done:          make(chan struct{}),
    // ...
}, nil
```

### subprocess.go: Start() Method (Lines 207-280)

```go
func (cp *ClaudeProcess) Start() error {
    // Create pipes
    stdin, err := cp.cmd.StdinPipe()   // stdin pipe created
    stdout, err := cp.cmd.StdoutPipe() // stdout pipe created
    stderr, err := cp.cmd.StderrPipe() // stderr pipe created

    cp.stdin = stdin
    cp.stdout = stdout
    cp.stderr = stderr

    // Start the process
    if err := cp.cmd.Start(); err != nil {
        return err
    }

    // Create NDJSON writer for stdin
    cp.stdinWriter = NewNDJSONWriter(cp.stdin)

    // Start reading events in background
    go cp.readEvents()   // Reads from stdout
    go cp.readStderr()   // Reads from stderr
    go cp.monitorExit()  // Monitors process exit

    cp.running = true
    return nil
}
```

### subprocess.go: Send() Method (Lines 331-339)

```go
func (cp *ClaudeProcess) Send(message string) error {
    return cp.SendJSON(UserMessage{
        Type: "user",
        Message: UserContent{
            Role: "user",
            Content: []ContentBlock{
                {Type: "text", Text: message},
            },
        },
    })
}

func (cp *ClaudeProcess) SendJSON(data interface{}) error {
    cp.mu.Lock()
    defer cp.mu.Unlock()

    if !cp.running {
        return fmt.Errorf("process not running")
    }

    return cp.stdinWriter.Write(data)  // Writes JSON + newline
}
```

### streams.go: NDJSONWriter (Lines 49-79)

```go
type NDJSONWriter struct {
    writer io.Writer
    mu     sync.Mutex
}

func (nw *NDJSONWriter) Write(data interface{}) error {
    nw.mu.Lock()
    defer nw.mu.Unlock()

    jsonData, err := json.Marshal(data)
    if err != nil {
        return err
    }

    // Write JSON followed by newline
    jsonData = append(jsonData, '\n')
    _, err = nw.writer.Write(jsonData)
    return err
    // NOTE: No explicit Flush() call!
}
```

### subprocess.go: readEvents() Loop (Lines 408-497)

```go
func (cp *ClaudeProcess) readEvents() {
    reader := NewNDJSONReader(cp.stdout)

    for {
        // Check for shutdown
        select {
        case <-cp.done:
            return
        default:
        }

        // Read with 100ms timeout context
        ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
        dataChan := make(chan []byte, 1)
        errChan := make(chan error, 1)

        go func() {
            data, err := reader.Read()
            if err != nil {
                errChan <- err
            } else {
                dataChan <- data
            }
        }()

        select {
        case <-ctx.Done():
            cancel()
            continue  // Timeout, try again
        case readErr = <-errChan:
            cancel()
            if readErr != nil && readErr != io.EOF {
                cp.errors <- fmt.Errorf("read error: %w", readErr)
            }
            return  // EXIT ON EOF OR ERROR
        case data = <-dataChan:
            cancel()
            // Parse and send event
            event, err := parseEvent(data)
            if err != nil {
                cp.errors <- fmt.Errorf("parse error: %w", err)
                continue
            }
            cp.events <- event
        }
    }
}
```

---

## Suspect Issues

### Issue 1: No Flush After Write

`NDJSONWriter.Write()` calls `nw.writer.Write()` but never flushes. If `cp.stdin` is buffered, the message may sit in a buffer and never reach Claude.

**Fix:**
```go
func (nw *NDJSONWriter) Write(data interface{}) error {
    // ... marshal and write ...

    // Flush if writer supports it
    if flusher, ok := nw.writer.(interface{ Flush() error }); ok {
        return flusher.Flush()
    }
    return err
}
```

### Issue 2: Possible Early Process Exit

With `--print` mode, Claude may:
1. Start up
2. Emit init events
3. Check stdin for input
4. If no input immediately available, exit or hang

**Evidence:** In my FIFO test, Claude waited for input. In pipe tests, it exited silently.

### Issue 3: Channel Replacement on Restart

In `subprocess.go:728-729`:
```go
cp.events = newProc.events        // Fresh channel after restart
cp.errors = newProc.errors        // Fresh channel after restart
```

The TUI's `waitForEvent()` is still blocked on the OLD channel. It will never receive events from the new channel.

**Fix:** Either:
- Don't replace channels, copy events from new to old
- Add channel-change notification mechanism
- Have TUI re-fetch channel reference periodically

### Issue 4: TUI Event Flow

```
main.go:
  process.Start()           // Process starts, readEvents() goroutine begins
  NewPanelModel(process)    // Panel created
  tea.NewProgram().Run()    // TUI starts

panel.go Init():
  waitForEvent(m.process.Events())  // Blocks on channel

User types, hits Enter:
  handleInput() → sendMessage() → process.Send()

Expected:
  Send() writes to stdin → Claude reads → Claude responds → stdout → readEvents() → channel → TUI

Actual:
  Send() writes to stdin → ??? (never reaches Claude or response never read)
```

---

## Task List with Details

### Task 1: Verify CLI Streaming ✅ COMPLETE
- Confirmed stream-json format works with FIFO test
- Confirmed all required flags
- Confirmed message format

### Task 2: Fix stdin Write/Flush Issue
**Files:** `internal/cli/streams.go`, `internal/cli/subprocess.go`

**Actions:**
1. Add `Sync()` or `Flush()` call after `NDJSONWriter.Write()`
2. Verify `cmd.StdinPipe()` isn't buffering unexpectedly
3. Add debug logging to confirm message bytes written

**Test:**
```go
func TestSendFlushes(t *testing.T) {
    // Create process, start
    // Send message
    // Verify bytes actually written to stdin pipe
}
```

### Task 3: Add Subprocess Streaming Integration Tests
**Files:** `internal/cli/subprocess_test.go`, `internal/cli/testdata/mock-claude.go`

**Actions:**
1. Update mock-claude.go to properly parse UserMessage format
2. Add test that verifies: Start → receive init → Send → receive response
3. Consider test with real Claude using FIFO pattern

**Current mock issue:** mock-claude.go expects `content` at top level, but UserMessage has nested structure.

### Task 4: Fix Channel Replacement on Restart
**Files:** `internal/cli/subprocess.go`

**Problem:** Line 728-729 replaces channels, breaking existing subscriptions.

**Options:**
A. Don't replace channels - forward events from new process to existing channels
B. Add `OnChannelChange(callback)` mechanism
C. Close old channels properly so TUI detects and re-subscribes

**Recommended:** Option A - simpler, no TUI changes needed

### Task 5: Add Restart Channel Subscription Tests
**Files:** `internal/cli/subprocess_test.go`, `internal/tui/claude/panel_test.go`

**Test scenarios:**
1. Process crashes → restart → events still received
2. Multiple restarts → channel still works
3. Max restarts exceeded → proper error state

### Task 6: TUI End-to-End Integration Test
**Files:** `internal/tui/claude/integration_test.go`

**Test flow:**
1. Create MockClaudeProcess
2. Create PanelModel
3. Simulate user input (KeyMsg Enter)
4. Verify Send() called
5. Inject mock event into channel
6. Verify panel updates

---

## Debug Strategy for Next Session

### Step 1: Add Logging

```go
// In subprocess.go Send()
func (cp *ClaudeProcess) SendJSON(data interface{}) error {
    log.Printf("[DEBUG] SendJSON called with: %+v", data)
    err := cp.stdinWriter.Write(data)
    log.Printf("[DEBUG] SendJSON write result: %v", err)
    return err
}

// In readEvents()
case data = <-dataChan:
    log.Printf("[DEBUG] readEvents received: %s", string(data))
```

### Step 2: Test with Real Claude

```go
func TestRealClaudeStreaming(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping real Claude test")
    }

    cfg := Config{
        ClaudePath: "claude",
        NoHooks:    true,  // Skip GOgent hooks for clean test
    }

    proc, err := NewClaudeProcess(cfg)
    require.NoError(t, err)

    err = proc.Start()
    require.NoError(t, err)
    defer proc.Stop()

    // Wait for init event
    select {
    case event := <-proc.Events():
        t.Logf("Init event: %s/%s", event.Type, event.Subtype)
    case <-time.After(10 * time.Second):
        t.Fatal("Timeout waiting for init")
    }

    // Send message
    err = proc.Send("Say HELLO")
    require.NoError(t, err)

    // Wait for response
    for {
        select {
        case event := <-proc.Events():
            t.Logf("Event: %s/%s", event.Type, event.Subtype)
            if event.Type == "result" {
                return // Success!
            }
        case err := <-proc.Errors():
            t.Fatalf("Error: %v", err)
        case <-time.After(30 * time.Second):
            t.Fatal("Timeout waiting for response")
        }
    }
}
```

---

## Key Files Reference

| File | Purpose |
|------|---------|
| `internal/cli/subprocess.go` | Process lifecycle, Send(), readEvents(), restart logic |
| `internal/cli/streams.go` | NDJSONReader/Writer for stdin/stdout |
| `internal/cli/events.go` | Event types, UserMessage struct, parseEvent() |
| `internal/cli/errors.go` | ClaudeError type, ParseError() |
| `internal/tui/claude/panel.go` | TUI model, Init(), Update(), waitForEvent() |
| `internal/tui/claude/input.go` | handleInput(), sendMessage() |
| `internal/tui/claude/output.go` | handleEvent(), appendStreamingText() |
| `cmd/gofortress/main.go` | Application entry point |

## Reference: claude-code-go Implementation

Located at `/tmp/claude-code-go/` (cloned during analysis).

Their approach is **one-shot**: start process with prompt in args, read response, exit. They don't maintain persistent stdin for multi-turn.

```go
// Their streaming.go - passes prompt as CLI arg, not stdin
args := BuildArgs(prompt, &streamOpts)  // prompt included in args
cmd := execCommand(ctx, c.BinPath, args...)
// Then just reads stdout until EOF
```

Our approach requires persistent stdin for multi-turn conversation, which is more complex but necessary for TUI interaction.
