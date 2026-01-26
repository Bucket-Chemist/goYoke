# GOfortress TUI Implementation Plan: Adopting claude-code-go Patterns

**Date:** 2026-01-26
**Analysis by:** Einstein (Opus 4.5)
**Status:** Implementation Plan Ready

---

## Executive Summary

After analyzing both GOfortress and the `lancekrogers/claude-code-go` repository, I've identified that GOfortress has **already solved most of the hard problems** but has a few critical gaps that the claude-code-go patterns can fill. The key insight is that claude-code-go's architecture is fundamentally different (functional/prompt-based vs long-running process), but specific patterns can be adapted.

### Key Findings

| Issue | Current GOfortress Status | claude-code-go Pattern | Effort |
|-------|---------------------------|------------------------|--------|
| Message format | ✅ **Already correct** - `[]ContentBlock` | Same format | None |
| Missing result events (Bug #1920) | ❌ No timeout handling | Retry + timeout patterns | Medium |
| WaitGroup coordination | ✅ Implemented but incomplete | N/A (different architecture) | Low |
| Error classification | ❌ Basic string matching | Structured `ClaudeError` types | Medium |
| Rate limit handling | ❌ Not implemented | Retry with extracted delay | Medium |
| Plugin/lifecycle hooks | ❌ Not implemented | Plugin interface pattern | Optional |
| Process state machine | ⚠️ Implicit in goroutines | Could be explicit | Optional |

---

## Part 1: Critical Fixes (Do These First)

### 1.1 Fix WaitGroup Usage in restart() (P0)

**Problem:** The `restart()` function in `subprocess.go` calls `newProc.startProcess()` which starts goroutines, but `startProcess()` doesn't use the WaitGroup from the parent process. Also, there's a race: the old process's goroutines might still be draining when the new process starts.

**Current Code Issue (subprocess.go:518-568):**
```go
func (cp *ClaudeProcess) restart() error {
    // ...creates newProc...

    // Start new process - but this doesn't wait for old goroutines to finish!
    if err := newProc.startProcess(); err != nil {
        return fmt.Errorf("start new process: %w", err)
    }

    // Replaces internal state with newProc's channels
    cp.events = newProc.events   // Old events channel may still have writers!
```

**Fix:**
```go
func (cp *ClaudeProcess) restart() error {
    // Wait for previous generation's goroutines to complete pipe reading
    cp.readWg.Wait()

    // Increment generation AFTER old goroutines finish
    newGen := cp.incrementGeneration()

    // ... rest of restart logic ...
}
```

**Files to modify:** `internal/cli/subprocess.go`
**Estimated lines changed:** ~15

---

### 1.2 Add Result Event Timeout (P0 - Bug #1920 Workaround)

**Problem:** Claude CLI Bug #1920 means the `{"type":"result",...}` event sometimes never arrives. GOfortress blocks forever on the event pump.

**claude-code-go pattern:** Uses context with timeout on all CLI invocations.

**Implementation:**

Create a new file `internal/cli/timeout.go`:
```go
package cli

import (
    "context"
    "time"
)

// ResultTimeout is the maximum time to wait for a result event
// after the last assistant event. Set conservatively high because
// tool execution can be slow.
const ResultTimeout = 5 * time.Minute

// TimeoutConfig allows customizing timeout behavior
type TimeoutConfig struct {
    ResultTimeout     time.Duration // Max wait for result after last activity
    InactivityTimeout time.Duration // Max wait with no events at all
}

func DefaultTimeoutConfig() TimeoutConfig {
    return TimeoutConfig{
        ResultTimeout:     5 * time.Minute,
        InactivityTimeout: 2 * time.Minute,
    }
}
```

Modify `readEvents()` to track last event time and check for timeout:
```go
func (cp *ClaudeProcess) readEvents() {
    myGen := cp.currentGeneration()
    reader := NewNDJSONReader(cp.stdout)
    lastEventTime := time.Now()

    for {
        // ... existing shutdown checks ...

        // Add inactivity timeout check
        if time.Since(lastEventTime) > cp.config.Timeout.InactivityTimeout {
            cp.errors <- fmt.Errorf("inactivity timeout: no events for %v",
                cp.config.Timeout.InactivityTimeout)
            return
        }

        // ... existing read logic ...

        // Update last event time on successful read
        lastEventTime = time.Now()
    }
}
```

**Files to create:** `internal/cli/timeout.go`
**Files to modify:** `internal/cli/subprocess.go`, `internal/cli/restart.go` (add to Config)
**Estimated lines:** ~60 new, ~25 modified

---

### 1.3 Add `--no-hooks` Flag for Testing (P1)

**Problem:** GOfortress hooks can interfere with TUI testing, creating a chicken-and-egg problem.

**Implementation:** Add to Config struct and argument building:
```go
type Config struct {
    // ... existing fields ...

    // NoHooks disables Claude Code hooks for testing
    NoHooks bool
}

// In NewClaudeProcess, add to args slice:
if cfg.NoHooks {
    args = append(args, "--no-hooks")
}
```

**Files to modify:** `internal/cli/subprocess.go`
**Estimated lines changed:** ~5

---

## Part 2: Error Handling Improvements (Medium Priority)

### 2.1 Structured Error Types

**claude-code-go pattern:** Defines 10 error categories with `ClaudeError` struct containing Type, Code, Message, IsRetryable(), and RetryDelay().

**Implementation:** Create `internal/cli/errors.go`:

```go
package cli

import (
    "fmt"
    "regexp"
    "strings"
    "time"
)

// ErrorType categorizes Claude CLI errors for handling decisions
type ErrorType int

const (
    ErrorUnknown ErrorType = iota
    ErrorAuthentication
    ErrorRateLimit
    ErrorPermission
    ErrorNetwork
    ErrorTimeout
    ErrorSession
    ErrorMCP
)

// ClaudeError provides structured error information
type ClaudeError struct {
    Type     ErrorType
    Code     int       // Exit code
    Message  string    // Error message
    Original error     // Wrapped error
    Stderr   string    // Raw stderr output
}

func (e *ClaudeError) Error() string {
    return fmt.Sprintf("claude error (type=%d, code=%d): %s", e.Type, e.Code, e.Message)
}

// IsRetryable returns true if this error type warrants retry
func (e *ClaudeError) IsRetryable() bool {
    switch e.Type {
    case ErrorRateLimit, ErrorNetwork, ErrorTimeout:
        return true
    case ErrorMCP:
        // MCP connection errors are retryable, config errors are not
        return strings.Contains(strings.ToLower(e.Message), "connection")
    }
    return false
}

// RetryDelay returns the recommended delay before retry
func (e *ClaudeError) RetryDelay() time.Duration {
    switch e.Type {
    case ErrorRateLimit:
        // Try to extract retry-after header
        if delay := extractRetryAfter(e.Stderr); delay > 0 {
            return delay
        }
        return 60 * time.Second
    case ErrorNetwork, ErrorTimeout:
        return 5 * time.Second
    case ErrorMCP:
        return 3 * time.Second
    }
    return 0
}

// ParseError classifies stderr output into a structured error
func ParseError(stderr string, exitCode int) *ClaudeError {
    lower := strings.ToLower(stderr)

    err := &ClaudeError{
        Code:   exitCode,
        Stderr: stderr,
    }

    switch {
    case containsAny(lower, "rate limit", "too many requests", "429", "quota exceeded"):
        err.Type = ErrorRateLimit
        err.Message = "API rate limit exceeded"
    case containsAny(lower, "authentication", "api key", "unauthorized", "401"):
        err.Type = ErrorAuthentication
        err.Message = "Authentication failed"
    case containsAny(lower, "permission", "denied", "forbidden", "403"):
        err.Type = ErrorPermission
        err.Message = "Permission denied"
    case containsAny(lower, "network", "connection", "dns", "timeout"):
        err.Type = ErrorNetwork
        err.Message = "Network error"
    case containsAny(lower, "mcp", "server"):
        err.Type = ErrorMCP
        err.Message = "MCP server error"
    default:
        err.Type = ErrorUnknown
        err.Message = firstLine(stderr)
    }

    return err
}

// Helper functions
func containsAny(s string, substrings ...string) bool {
    for _, sub := range substrings {
        if strings.Contains(s, sub) {
            return true
        }
    }
    return false
}

var retryAfterRe = regexp.MustCompile(`retry[- ]?after[:\s]+(\d+)`)

func extractRetryAfter(s string) time.Duration {
    matches := retryAfterRe.FindStringSubmatch(strings.ToLower(s))
    if len(matches) >= 2 {
        // Parse seconds
        var seconds int
        fmt.Sscanf(matches[1], "%d", &seconds)
        return time.Duration(seconds) * time.Second
    }
    return 0
}

func firstLine(s string) string {
    if idx := strings.Index(s, "\n"); idx >= 0 {
        return s[:idx]
    }
    return s
}
```

**Files to create:** `internal/cli/errors.go`, `internal/cli/errors_test.go`
**Estimated lines:** ~150

---

### 2.2 Integrate Error Classification into readStderr()

Modify `readStderr()` to use the new error classification:

```go
func (cp *ClaudeProcess) readStderr() {
    // ... existing code ...

    // Instead of:
    // cp.errors <- fmt.Errorf("stderr: %s", string(data))

    // Use structured errors:
    claudeErr := ParseError(string(data), 0)
    cp.errors <- claudeErr
}
```

And in `monitorRestart()`, use error type for restart decisions:

```go
func (cp *ClaudeProcess) monitorRestart() {
    err := cp.cmd.Wait()

    // Classify exit error
    var claudeErr *ClaudeError
    if exitErr, ok := err.(*exec.ExitError); ok {
        claudeErr = ParseError(stderrBuffer.String(), exitErr.ExitCode())
        claudeErr.Original = err
    }

    // Don't restart on authentication errors - they won't self-heal
    if claudeErr != nil && claudeErr.Type == ErrorAuthentication {
        cp.sendExitEvent(claudeErr, false)
        return
    }

    // ... rest of restart logic ...
}
```

---

## Part 3: TUI Improvements (Lower Priority)

### 3.1 Handle processStoppedMsg with Restart Context

**Current issue:** When the process stops, the TUI just sets `streaming = false`. It doesn't communicate WHY it stopped or if a restart is pending.

**Fix:** Subscribe to RestartEvents channel in addition to Events:

```go
// In panel.go Init():
func (m PanelModel) Init() tea.Cmd {
    return tea.Batch(
        textarea.Blink,
        waitForEvent(m.process.Events()),
        waitForRestartEvent(m.process.RestartEvents()), // NEW
    )
}

func waitForRestartEvent(events <-chan cli.RestartEvent) tea.Cmd {
    return func() tea.Msg {
        event, ok := <-events
        if !ok {
            return nil
        }
        return event
    }
}

// In Update():
case cli.RestartEvent:
    if msg.Reason == "max_restarts_exceeded" {
        m.showError("Process crashed and could not restart")
    } else {
        m.showInfo(fmt.Sprintf("Restarting in %v...", msg.NextDelay))
    }
    cmds = append(cmds, waitForRestartEvent(m.process.RestartEvents()))
```

**Files to modify:** `internal/tui/claude/panel.go`
**Estimated lines:** ~30

---

### 3.2 Add Status Indicator to TUI

Show process state clearly:

```go
type ProcessState int

const (
    StateConnecting ProcessState = iota
    StateReady
    StateStreaming
    StateRestarting
    StateStopped
    StateError
)

func (m PanelModel) View() string {
    // Add status indicator to header
    statusIcon := map[ProcessState]string{
        StateConnecting: "🔄",
        StateReady:      "🟢",
        StateStreaming:  "💭",
        StateRestarting: "♻️",
        StateStopped:    "⬛",
        StateError:      "🔴",
    }[m.state]

    header := headerStyle.Render(fmt.Sprintf(
        "%s Claude Code - Session: %s  Cost: $%.2f",
        statusIcon,
        truncate(m.sessionID, 8),
        m.cost,
    ))
    // ...
}
```

---

## Part 4: Optional Enhancements (Future Work)

### 4.1 SubagentManager - Agent Tree Interaction (PROMOTED TO P1)

The agent tree view (`internal/tui/agents/`) is **read-only** - it displays agents from telemetry but cannot:
- Spawn new specialized agents
- Query/interact with running agents
- Stop or restart agents
- Fork agent sessions

The claude-code-go `SubagentManager` pattern fills this gap perfectly.

**Current gap (detail.go:73-76):**
```go
// Keyboard hints show "[q] Query agent" but nothing implements it!
b.WriteString(hintStyle.Render("[q] Query agent"))
```

**Implementation: `internal/cli/subagent.go`**

```go
package cli

import (
    "context"
    "fmt"
    "sync"
)

// SubagentConfig defines a specialized agent's configuration
type SubagentConfig struct {
    Name         string   // Unique identifier (e.g., "security-reviewer")
    Description  string   // What this agent does
    SystemPrompt string   // Custom system prompt
    AllowedTools []string // Tool whitelist
    Model        string   // "haiku", "sonnet", "opus" (empty = inherit)
    MaxTurns     int      // Turn limit (0 = unlimited)
}

// PresetAgents provides factory functions for common agent types
var PresetAgents = map[string]func() SubagentConfig{
    "security-reviewer": func() SubagentConfig {
        return SubagentConfig{
            Name:         "security-reviewer",
            Description:  "Analyzes code for security vulnerabilities",
            SystemPrompt: "You are a security expert. Focus on OWASP Top 10, injection risks, auth issues.",
            AllowedTools: []string{"Read", "Grep", "Glob"},
            Model:        "sonnet",
            MaxTurns:     10,
        }
    },
    "code-reviewer": func() SubagentConfig {
        return SubagentConfig{
            Name:         "code-reviewer",
            Description:  "Reviews code quality, patterns, and architecture",
            SystemPrompt: "You are a senior engineer. Focus on readability, maintainability, and idioms.",
            AllowedTools: []string{"Read", "Grep", "Glob"},
            Model:        "sonnet",
            MaxTurns:     10,
        }
    },
    "test-analyst": func() SubagentConfig {
        return SubagentConfig{
            Name:         "test-analyst",
            Description:  "Analyzes test coverage and suggests improvements",
            SystemPrompt: "You are a testing expert. Focus on coverage gaps and edge cases.",
            AllowedTools: []string{"Read", "Grep", "Glob", "Bash(go test*)"},
            Model:        "sonnet",
            MaxTurns:     15,
        }
    },
}

// SubagentManager manages specialized agent instances
type SubagentManager struct {
    registry map[string]SubagentConfig
    sessions map[string]*ClaudeProcess // agentName -> active process
    mu       sync.RWMutex
    baseCfg  Config // Inherited config from parent process
}

// NewSubagentManager creates a manager with preset agents registered
func NewSubagentManager(baseCfg Config) *SubagentManager {
    sm := &SubagentManager{
        registry: make(map[string]SubagentConfig),
        sessions: make(map[string]*ClaudeProcess),
        baseCfg:  baseCfg,
    }

    // Register presets
    for name, factory := range PresetAgents {
        sm.Register(factory())
        _ = name // unused but makes intent clear
    }

    return sm
}

// Register adds an agent configuration to the registry
func (sm *SubagentManager) Register(cfg SubagentConfig) error {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    if _, exists := sm.registry[cfg.Name]; exists {
        return fmt.Errorf("agent %q already registered", cfg.Name)
    }

    sm.registry[cfg.Name] = cfg
    return nil
}

// List returns all registered agent names with descriptions
func (sm *SubagentManager) List() []SubagentConfig {
    sm.mu.RLock()
    defer sm.mu.RUnlock()

    result := make([]SubagentConfig, 0, len(sm.registry))
    for _, cfg := range sm.registry {
        result = append(result, cfg)
    }
    return result
}

// Spawn creates and starts a new agent process
func (sm *SubagentManager) Spawn(ctx context.Context, agentName string) (*ClaudeProcess, error) {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    cfg, exists := sm.registry[agentName]
    if !exists {
        return nil, fmt.Errorf("unknown agent: %q", agentName)
    }

    // Check if already running
    if proc, running := sm.sessions[agentName]; running && proc.IsRunning() {
        return proc, nil // Return existing
    }

    // Build process config from base + agent overrides
    procCfg := sm.baseCfg
    // Note: Would need to extend Config to support system prompt, allowed tools, etc.
    // This is where the integration work happens

    proc, err := NewClaudeProcess(procCfg)
    if err != nil {
        return nil, fmt.Errorf("create process for %s: %w", agentName, err)
    }

    if err := proc.Start(); err != nil {
        return nil, fmt.Errorf("start process for %s: %w", agentName, err)
    }

    sm.sessions[agentName] = proc
    return proc, nil
}

// Query sends a prompt to a running agent and returns the events channel
func (sm *SubagentManager) Query(agentName, prompt string) (<-chan Event, error) {
    sm.mu.RLock()
    proc, exists := sm.sessions[agentName]
    sm.mu.RUnlock()

    if !exists || !proc.IsRunning() {
        return nil, fmt.Errorf("agent %q not running", agentName)
    }

    if err := proc.Send(prompt); err != nil {
        return nil, fmt.Errorf("send to %s: %w", agentName, err)
    }

    return proc.Events(), nil
}

// Stop terminates a running agent
func (sm *SubagentManager) Stop(agentName string) error {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    proc, exists := sm.sessions[agentName]
    if !exists {
        return nil // Already stopped
    }

    delete(sm.sessions, agentName)
    return proc.Stop()
}

// StopAll terminates all running agents
func (sm *SubagentManager) StopAll() {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    for name, proc := range sm.sessions {
        proc.Stop()
        delete(sm.sessions, name)
    }
}
```

**TUI Integration (`internal/tui/agents/manager_view.go`):**

```go
// Add to Model struct
type Model struct {
    // ... existing fields ...
    subagentMgr *cli.SubagentManager
}

// Handle 'q' key to query selected agent
case "q":
    if m.selectedID != "" {
        return m, m.queryAgent()
    }

// Handle 's' key to spawn new agent (shows picker)
case "s":
    return m, m.showAgentPicker()

func (m Model) queryAgent() tea.Cmd {
    return func() tea.Msg {
        // This would open an input prompt and send to the agent
        return QueryAgentMsg{AgentID: m.selectedID}
    }
}
```

**Why this is P1 not P4:**
- The tree view UI is already built but non-functional
- Users expect `[q] Query agent` to work
- This enables the core TUI value proposition: managing multiple Claude agents visually

**Integration with GOgent Agent System:**

Your existing GOgent routing system (`agents-index.json`) defines agents like `go-pro`, `go-tui`, `security-reviewer`, etc. The SubagentManager can load these definitions:

```go
// Load from agents-index.json
func LoadAgentsFromIndex(path string) ([]SubagentConfig, error) {
    // Parse agents-index.json
    // Convert each entry to SubagentConfig
    // This makes TUI agents consistent with CLI routing
}

// Example: Spawn go-tui agent from TUI
proc, err := subagentMgr.Spawn(ctx, "go-tui")
// Sends prompts with Bubbletea-specific system prompt
```

This creates a unified agent experience:
- CLI routing uses `agents-index.json` for dispatch
- TUI uses same definitions via SubagentManager
- Telemetry tree shows all agents regardless of spawn source

**Files to create:**
- `internal/cli/subagent.go` (~200 lines)
- `internal/cli/subagent_test.go` (~150 lines)
- `internal/tui/agents/picker.go` (agent selection UI, ~100 lines)

**Files to modify:**
- `internal/tui/agents/view.go` (integrate SubagentManager)
- `internal/cli/subprocess.go` (extend Config for system prompt, allowed tools)

---

### 4.2 Budget Tracking (from claude-code-go)

claude-code-go tracks cumulative cost across invocations. GOfortress already captures `total_cost_usd` from ResultEvents.

**Enhancement:** Persist cost across restarts:
```go
type SessionMetrics struct {
    TotalCostUSD float64
    TotalTokens  int
    TurnCount    int
}
```

---

### 4.3 MCP Config Builder (from claude-code-go)

The `MCPConfigBuilder` pattern is useful if GOfortress needs programmatic MCP setup. Currently, GOfortress relies on Claude Code's own MCP configuration.

**Recommendation:** Defer unless MCP integration becomes a priority.

---

## Implementation Phases

### Phase 1: Critical Fixes (Unblock TUI)
1. Fix WaitGroup in restart() - 1 hour
2. Add result timeout handling - 2 hours
3. Add `--no-hooks` flag - 15 minutes
4. **Test:** Run TUI with `--no-hooks`, verify messages flow

### Phase 2: Error Handling
1. Create errors.go with structured types - 2 hours
2. Integrate into subprocess lifecycle - 1 hour
3. Add tests - 1 hour

### Phase 3: SubagentManager (Agent Tree Interaction)
1. Create `internal/cli/subagent.go` with registry pattern - 3 hours
2. Add preset agents (security-reviewer, code-reviewer, test-analyst) - 1 hour
3. Extend Config struct for system prompt, allowed tools - 1 hour
4. Create agent picker TUI component - 2 hours
5. Wire up `[q] Query agent` in tree view - 1 hour
6. Add tests - 2 hours

### Phase 4: TUI Polish
1. Subscribe to RestartEvents - 1 hour
2. Add status indicator - 30 minutes
3. Test restart scenarios

### Phase 5: Documentation
1. Update GOFORTRESS-HANDOVER.md with fixes
2. Create debugging guide

---

## Verification Commands

After implementing Phase 1, test with:

```bash
# Build
go build ./cmd/gofortress/

# Test without hooks (isolates TUI from GOgent hooks)
echo '{"type":"user","message":{"role":"user","content":[{"type":"text","text":"say hello"}]}}' | \
  claude --print --verbose --debug-to-stderr --no-hooks \
         --input-format stream-json --output-format stream-json \
         --session-id $(uuidgen)

# Test with TUI
./gofortress --no-hooks
```

---

## Files to Create/Modify Summary

| File | Action | Priority |
|------|--------|----------|
| `internal/cli/subprocess.go` | Modify (WaitGroup, --no-hooks, Config extension) | P0 |
| `internal/cli/timeout.go` | Create | P0 |
| `internal/cli/errors.go` | Create | P1 |
| `internal/cli/errors_test.go` | Create | P1 |
| `internal/cli/subagent.go` | Create (SubagentManager) | P1 |
| `internal/cli/subagent_test.go` | Create | P1 |
| `internal/tui/agents/picker.go` | Create (agent selector UI) | P1 |
| `internal/tui/agents/view.go` | Modify (integrate SubagentManager) | P1 |
| `internal/tui/claude/panel.go` | Modify (RestartEvents) | P2 |
| `internal/tui/claude/events.go` | Modify (state handling) | P2 |

---

## Conclusion

The GOfortress TUI has **correct foundations** - the message format is right, channel management is mostly correct, and the restart policy is well-designed. The issues are:

1. **Timing bugs** - WaitGroup not covering restart scenarios
2. **Missing defensive measures** - No timeout for Bug #1920
3. **Weak error classification** - Can't make smart retry decisions
4. **Read-only agent tree** - Beautiful UI with no interaction capability

The claude-code-go patterns for error handling, retry logic, AND the SubagentManager are directly applicable. The SubagentManager is particularly valuable because:
- Your agent tree view already exists and looks good
- The `[q] Query agent` hint promises functionality that doesn't exist
- Managing multiple specialized agents is a core differentiator for GOfortress

The plugin system and MCP builder are less urgent given GOfortress's architecture.

**Total estimated effort:**
- Phases 1-2 (critical fixes + errors): 8 hours
- Phase 3 (SubagentManager): 10 hours
- Phase 4 (polish): 2 hours
- **Total: ~20 hours**
