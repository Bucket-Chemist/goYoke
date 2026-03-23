# TUI-004 Spike: Side Channel IPC Protocol (MCP to TUI)

**Date:** 2026-03-23
**Go Version:** 1.25.5
**Transport:** Unix domain socket (UDS)
**Latency:** 56µs average (0.056ms) — 90x under 5ms target

---

## Executive Summary

**UDS IPC between Go MCP server and Go TUI works perfectly.** Round-trip latency is sub-millisecond (~56µs without modal delay). The JSON-over-UDS protocol is simple, reliable, and ready for production use in TUI-014/TUI-015.

---

## 1. Protocol Specification

### 1.1 Transport

- **Unix domain socket** at `$XDG_RUNTIME_DIR/gofortress-{pid}.sock`
- Newline-delimited JSON (one JSON object per `\n`)
- `json.NewEncoder`/`json.NewDecoder` over `net.Conn`
- Single persistent connection per session (MCP connects once at startup)

### 1.2 Request Types

```go
type IPCRequest struct {
    Type    string          `json:"type"`    // Request discriminator
    ID      string          `json:"id"`      // Unique request ID for correlation
    Payload json.RawMessage `json:"payload"` // Type-specific payload
}
```

| Type | Direction | Payload | Description |
|------|-----------|---------|-------------|
| `modal_request` | MCP → TUI | `{message, options[]}` | Show modal, return user choice |
| `agent_register` | MCP → TUI | `{agentId, agentType, ...}` | Register new subagent |
| `agent_update` | MCP → TUI | `{agentId, status, ...}` | Update agent status |
| `agent_activity` | MCP → TUI | `{agentId, tool, ...}` | Live agent tool activity |
| `toast` | MCP → TUI | `{message, level}` | Show notification toast |

### 1.3 Response Types

```go
type IPCResponse struct {
    Type    string          `json:"type"`    // Response discriminator
    ID      string          `json:"id"`      // Matches request ID
    Payload json.RawMessage `json:"payload"` // Type-specific payload
}
```

| Type | Direction | Payload | Description |
|------|-----------|---------|-------------|
| `modal_response` | TUI → MCP | `{value}` | User's modal selection |

### 1.4 Modal Request/Response (POC-validated)

**Request:**
```json
{"type":"modal_request","id":"req-1","message":"Allow Write to /path?","options":["Allow","Allow Always","Deny"]}
```

**Response:**
```json
{"type":"modal_response","id":"req-1","value":"Allow"}
```

---

## 2. Performance

### 2.1 Latency Results

| Test | Modal Delay | Avg RTT (MCP measured) | Avg RTT (TUI measured) | Transport Overhead |
|------|-------------|----------------------|----------------------|-------------------|
| With 10ms delay | 10ms | 10.213ms | 10.129ms | ~0.2ms |
| Zero delay | 0ms | 56µs | 8µs | ~56µs |

### 2.2 Per-Request Breakdown (Zero Delay)

```
req-1: 165µs (cold — first request, connection warm-up)
req-2:  57µs
req-3:  27µs
req-4:  19µs
req-5:  10µs (steady state)
```

**Conclusion:** UDS transport adds ~10-60µs per roundtrip. Well within the 100ms modal budget from TUI-018.

### 2.3 Production Estimate

With real Bubble Tea modal interaction:
- UDS transport: ~50µs
- Modal render + user interaction: 500ms-5s (user-dependent)
- Total: dominated by user interaction, transport is negligible

---

## 3. Connection Lifecycle

### 3.1 Startup Sequence

```
TUI process                          MCP process (child of Claude CLI)
───────────                          ────────────────────────────────
CleanupStaleSockets()
net.Listen("unix", sockPath)
os.Setenv("GOFORTRESS_SOCKET", sockPath)
                                     Read GOFORTRESS_SOCKET env var
Accept() [blocks]                    connectWithRetry(sockPath)
conn established ◄──────────────────────────────────
```

### 3.2 Connection Retry (MCP Side)

```go
// Exponential backoff: 100ms, 200ms, 400ms, 800ms, 1600ms (5 attempts)
func connectWithRetry(sockPath string) (net.Conn, error) {
    delay := 100 * time.Millisecond
    for attempt := 1; attempt <= 5; attempt++ {
        conn, err := net.Dial("unix", sockPath)
        if err == nil {
            return conn, nil
        }
        time.Sleep(delay)
        delay *= 2
    }
    return nil, fmt.Errorf("connect after 5 attempts")
}
```

**POC validated:** MCP connects on attempt 1 when TUI is ready. Retry logic handles race conditions at startup.

### 3.3 Socket Cleanup

| Event | Action |
|-------|--------|
| Normal exit | `defer os.Remove(sockPath)` |
| SIGINT/SIGTERM | Signal handler removes socket |
| Crash/SIGKILL | `CleanupStaleSockets()` on next startup (glob + PID liveness check) |

**POC validated:** No stale sockets after TUI exit.

---

## 4. Integration with Bubble Tea

### 4.1 Program.Send() Injection Pattern

```go
// Socket reader goroutine (runs outside Bubble Tea event loop)
go func() {
    dec := json.NewDecoder(conn)
    for {
        var req IPCRequest
        if err := dec.Decode(&req); err != nil {
            p.Send(connClosedMsg{})
            return
        }
        // Inject into Bubble Tea's event loop — safe, non-blocking
        p.Send(ipcRequestMsg{req: req, receivedAt: time.Now()})
    }
}()
```

### 4.2 Response from Update()

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case ipcRequestMsg:
        // Show modal, queue response...
        m.pendingModal = msg.req
        return m, showModalCmd(msg.req)
    case modalCompleteMsg:
        // User made selection — send response back over socket
        resp := IPCResponse{Type: "modal_response", ID: msg.reqID, Value: msg.value}
        json.NewEncoder(m.conn).Encode(resp)
        return m, nil
    }
}
```

**Note:** `Program.Send()` is thread-safe and non-blocking. The socket reader goroutine never touches model state directly — it only sends messages that the Bubble Tea event loop processes sequentially.

---

## 5. Implications for Production Tickets

| Ticket | Finding |
|--------|---------|
| **TUI-014** (MCP server) | Use `connectWithRetry` pattern for UDS client. Read `GOFORTRESS_SOCKET` from env. JSON encoder/decoder over single connection. |
| **TUI-015** (IPC bridge) | Use `net.Listen("unix", ...)` + `Accept()`. Socket reader goroutine with `p.Send()`. Cleanup via `defer` + signal handler + stale detection. |
| **TUI-018** (permissions) | Modal roundtrip budget is 100ms. Transport adds ~50µs — 2000x margin. Response latency is entirely user-dependent. |
| **TUI-034** (graceful shutdown) | Close socket connection → reader goroutine gets EOF → sends `connClosedMsg` → TUI shuts down. |

---

## 6. POC Binary Locations

```
cmd/gofortress-ipc-poc/tui/main.go  — TUI side (socket listener + responder)
cmd/gofortress-ipc-poc/mcp/main.go  — MCP side (socket client + requester)
```

Run with: `bin/gofortress-ipc-tui --delay=0` and `GOFORTRESS_SOCKET=<path> bin/gofortress-ipc-mcp`

---

_Generated by TUI-004 spike, 2026-03-23_
