Revised Ticket Sequence (Go-Native)

GOgent-028a: Session Monitoring Daemon

---

id: GOgent-028a  
 title: "Session Monitoring Daemon (Go-Native)"  
 description: "Implement Go daemon to monitor Claude Code sessions and trigger handoff
generation"  
 status: pending  
 dependencies: [GOgent-028]  
 time_estimate: "2.0h"  
 priority: CRITICAL

---

## Implementation Strategy

**Option A (Recommended): Signal-Based**

- Daemon runs as child process of Claude Code
- Receives SIGTERM on session end
- Generates handoff in signal handler  


**Option B: Polling**

- Daemon polls for Claude Code process
- Generates handoff when process exits  


**Option C: File Watcher**

- Daemon watches `/tmp/claude-sessions/` for markers
- Generates handoff on session-end file creation  


## Acceptance Criteria

- [ ] `cmd/gogent/main.go` implements daemon
- [ ] `pkg/daemon/daemon.go` implements session monitoring
- [ ] Daemon starts with `gogent daemon start`
- [ ] Daemon detects session end (via signal/polling/watcher)
- [ ] Daemon calls `session.GenerateHandoff()` natively
- [ ] Graceful shutdown on SIGINT/SIGTERM
- [ ] No bash scripts involved
- [ ] Daemon logs to stderr
- [ ] Tests: Mock session end, verify handoff generated  


## Files

- `cmd/gogent/main.go` - Daemon entry point
- `pkg/daemon/daemon.go` - Session monitoring logic
- `pkg/daemon/daemon_test.go` - Daemon tests  


GOgent-028b: Session Context Loading (Go-Native)

---

id: GOgent-028b  
 title: "Session Context Loading (Go-Native)"  
 description: "Load previous handoff at session start via Go library integration"  
 status: pending  
 dependencies: [GOgent-028a]  
 time_estimate: "1.0h"  
 priority: HIGH

---

## Implementation Strategy

**Option A (Recommended): Daemon Hook**

- Daemon detects session start
- Loads handoff via `session.LoadHandoff()`
- Writes summary to `/tmp/claude-session-context.txt`
- Claude Code reads this file (if supported)  


**Option B: CLI Integration**

- Claude Code calls `gogent context load` at start
- CLI prints summary to stdout
- Claude Code captures and displays  


**Option C: IPC Integration**

- Daemon exposes IPC endpoint
- Claude Code queries daemon for context
- Daemon responds with handoff summary  


## Acceptance Criteria

- [ ] Daemon detects session start
- [ ] Daemon loads last handoff via `session.LoadHandoff()`
- [ ] Daemon formats summary (actions, learnings, etc.)
- [ ] Context injected into Claude Code session
- [ ] Graceful handling if no handoff exists
- [ ] No bash scripts involved
- [ ] Tests: Mock session start, verify context loaded  


---

Metrics Collection (Go-Native)

The metrics gap I identified earlier also needs a Go-native solution:

GOgent-027-REVISION: Metrics Serialization

// pkg/session/metrics.go

type MetricsCollector struct {  
 mu sync.Mutex  
 toolCalls int  
 errors int  
 violations int  
 startTime time.Time  
 }

func NewMetricsCollector() \*MetricsCollector {  
 return &MetricsCollector{  
 startTime: time.Now(),  
 }  
 }

func (m \*MetricsCollector) RecordToolCall(toolName string) {  
 m.mu.Lock()  
 defer m.mu.Unlock()  
 m.toolCalls++  
 }

func (m \*MetricsCollector) RecordError() {  
 m.mu.Lock()  
 defer m.mu.Unlock()  
 m.errors++  
 }

func (m *MetricsCollector) Snapshot() *SessionMetrics {  
 m.mu.Lock()  
 defer m.mu.Unlock()

      return &SessionMetrics{
          SessionID:      generateSessionID(),
          StartTime:      m.startTime.Unix(),
          EndTime:        time.Now().Unix(),
          ToolCallCount:  m.toolCalls,
          ErrorCount:     m.errors,
          ViolationCount: m.violations,
      }

}

Daemon integration:

func (d \*Daemon) Run(ctx context.Context) error {  
 // Start metrics collection  
 d.metrics = session.NewMetricsCollector()

      // Monitor tool calls (how? see below)
      go d.monitorToolCalls()

      // ... session monitoring ...

}

func (d \*Daemon) monitorToolCalls() {  
 // Option A: Parse Claude Code logs in real-time  
 // Option B: Watch for tool execution markers  
 // Option C: IPC with Claude Code  
 }

---

The Critical Question: Claude Code Integration

Everything depends on: How does gogent interact with Claude Code?

Discovery Required

1. Does Claude Code expose lifecycle hooks?


    - Session start event?
    - Session end event?
    - Tool execution events?

2. Does Claude Code support IPC?


    - Unix sockets?
    - Named pipes?
    - RPC interface?

3. Does Claude Code write observable artifacts?


    - Log files?
    - Marker files?
    - Process signals?

4. Can we extend Claude Code?


    - Plugin system?
    - Extension API?
    - Source modification?


Without this information, we're guessing.

---

Recommended Investigation Path

Step 1: Discover Claude Code's Extension Points

# Check for IPC sockets

ls -la /tmp/claude-\*

# Check for plugin directories

ls -la ~/.claude/plugins/

# Check process tree

pstree -p | grep claude

# Check for configuration

cat ~/.claude/config.toml

# Check for logs

ls -la ~/.claude/logs/

Step 2: Test Simple Integration

// Test: Can we detect when Claude Code starts/stops?  
 package main

import (  
 "fmt"  
 "os"  
 "time"  
 )

func main() {  
 fmt.Println("Waiting for Claude Code...")

      for {
          // Check if Claude Code is running
          if isRunning, pid := detectClaudeCode(); isRunning {
              fmt.Printf("Claude Code detected (PID: %d)\n", pid)

              // Wait for it to exit
              waitForExit(pid)

              fmt.Println("Claude Code exited, generating handoff...")
              // Generate handoff here
              break
          }

          time.Sleep(1 * time.Second)
      }

}

func detectClaudeCode() (bool, int) {  
 // Implementation: ps aux | grep claude-code  
 return false, 0  
 }

func waitForExit(pid int) {  
 // Implementation: wait for PID to disappear  
 }

Step 3: Build Minimal Daemon

Once we understand the integration points, build the minimal daemon that:

1. Starts when Claude Code starts
2. Monitors session lifecycle
3. Generates handoff on exit
4. All in Go, zero bash  


---

Answer to Your Pushback

You're correct: Bash hooks are wrong for a Go migration.

The right solution:

1. Go daemon (gogent daemon) runs continuously
2. Monitors Claude Code session lifecycle (signal/poll/IPC)
3. Calls Go libraries directly (session.GenerateHandoff())
4. Zero bash scripts (except maybe systemd unit file for daemon auto-start)  


What I need to know:

- How does Claude Code expose session lifecycle?
- Can we hook into it programmatically?
- Or do we need to poll/watch for external signals?  


Next step: Investigate Claude Code's architecture to design the proper integration  
 points.

Bottom line: You're right to push back. Go-native means Go owns the lifecycle, not  
 bash scripts calling Go binaries.

● Ran 1 stop hook  
 ⎿ Stop hook error: Failed with non-blocking status code: /bin/sh: line 1:  
 /home/doktersmol/Documents/GOgent-Fortress/.claude/hooks/stop-gate.sh: No such  
 file or directory

✻ Sautéed for 1m 11s

❯ please route this question to /einstein as a subagent - does this plan wire in  
 with my existing implementation and hook capture? does anything else require  
 revisiting as a result? ultrathink and then please route the full plan to a  
 series of .md tickets to amend as 0.28a....i

❯ /einstein - does this plan wire in with my existing implementation and hook capture?  
does anything else require revisiting as a result? ultrathink and then please route the
full plan to a series of .md tickets to amend as 0.28a....i. please route your response
to orchestrator for handoff and ticket creation
