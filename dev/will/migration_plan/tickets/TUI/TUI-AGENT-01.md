# TUI-AGENT-01: Agent Tree Data Model

> **Estimated Hours:** 2.0
> **Priority:** P1 - Agent Tree
> **Dependencies:** TUI-INFRA-01, TUI-TELEM-01
> **Phase:** 3 - Agent Tree

---

## Description

Create the data model for tracking agent delegation tree with spawn/complete correlation.

---

## Tasks

### 1. Define Agent Status and Node Types

**File:** `internal/tui/agents/model.go`

```go
package agents

import (
    "sync"
    "time"

    "github.com/Bucket-Chemist/GOgent-Fortress/pkg/telemetry"
)

type AgentStatus string

const (
    StatusSpawning  AgentStatus = "spawning"
    StatusRunning   AgentStatus = "running"
    StatusCompleted AgentStatus = "completed"
    StatusFailed    AgentStatus = "failed"
)

// AgentNode represents a single agent in the delegation tree
type AgentNode struct {
    ID              string        // Event ID from lifecycle
    AgentID         string        // "python-pro", "orchestrator", etc.
    ParentID        string        // "" for terminal-spawned
    Tier            string        // "haiku", "sonnet", "opus"
    Status          AgentStatus
    TaskDescription string
    SpawnTime       time.Time
    CompleteTime    *time.Time
    Duration        time.Duration
    Success         *bool
    Children        []*AgentNode
}
```

### 2. Implement Agent Tree Container

```go
// AgentTree tracks all agents in the current session
type AgentTree struct {
    mu       sync.RWMutex
    Root     *AgentNode            // Virtual root (terminal)
    ByID     map[string]*AgentNode // Lookup by event ID
    ByAgent  map[string][]*AgentNode // Lookup by agent name (may have multiple)
    Pending  []*AgentNode          // Spawned but not completed
    SessionID string
}

func NewAgentTree(sessionID string) *AgentTree {
    root := &AgentNode{
        ID:       "terminal",
        AgentID:  "terminal",
        Status:   StatusRunning,
        Children: make([]*AgentNode, 0),
    }

    return &AgentTree{
        Root:      root,
        ByID:      map[string]*AgentNode{"terminal": root},
        ByAgent:   make(map[string][]*AgentNode),
        Pending:   make([]*AgentNode, 0),
        SessionID: sessionID,
    }
}
```

### 3. Implement Spawn Handling

```go
// HandleSpawn processes a spawn lifecycle event
func (t *AgentTree) HandleSpawn(event telemetry.AgentLifecycleEvent) {
    t.mu.Lock()
    defer t.mu.Unlock()

    node := &AgentNode{
        ID:              event.EventID,
        AgentID:         event.AgentID,
        ParentID:        event.ParentAgent,
        Tier:            event.Tier,
        Status:          StatusSpawning,
        TaskDescription: event.TaskDescription,
        SpawnTime:       time.Unix(event.Timestamp, 0),
        Children:        make([]*AgentNode, 0),
    }

    t.ByID[node.ID] = node
    t.ByAgent[node.AgentID] = append(t.ByAgent[node.AgentID], node)
    t.Pending = append(t.Pending, node)

    // Link to parent (or root if parent not found)
    if parent, ok := t.ByID[node.ParentID]; ok {
        parent.Children = append(parent.Children, node)
    } else {
        t.Root.Children = append(t.Root.Children, node)
    }
}
```

### 4. Implement Complete Handling

```go
// HandleComplete processes a complete lifecycle event
func (t *AgentTree) HandleComplete(event telemetry.AgentLifecycleEvent) {
    t.mu.Lock()
    defer t.mu.Unlock()

    // Find matching pending node by AgentID (LIFO - most recent spawn)
    for i := len(t.Pending) - 1; i >= 0; i-- {
        node := t.Pending[i]
        if node.AgentID == event.AgentID {
            now := time.Unix(event.Timestamp, 0)
            node.CompleteTime = &now
            node.Duration = now.Sub(node.SpawnTime)
            node.Success = event.Success

            if event.Success != nil && *event.Success {
                node.Status = StatusCompleted
            } else {
                node.Status = StatusFailed
            }

            // Remove from pending
            t.Pending = append(t.Pending[:i], t.Pending[i+1:]...)
            return
        }
    }
}
```

### 5. Implement Status Update and Accessors

```go
// UpdateRunningStatus marks spawning agents as running after threshold
func (t *AgentTree) UpdateRunningStatus(threshold time.Duration) {
    t.mu.Lock()
    defer t.mu.Unlock()

    now := time.Now()
    for _, node := range t.Pending {
        if node.Status == StatusSpawning && now.Sub(node.SpawnTime) > threshold {
            node.Status = StatusRunning
        }
    }
}

// GetPending returns all pending (non-completed) agents
func (t *AgentTree) GetPending() []*AgentNode {
    t.mu.RLock()
    defer t.mu.RUnlock()

    result := make([]*AgentNode, len(t.Pending))
    copy(result, t.Pending)
    return result
}

// GetAll returns all agents (flattened)
func (t *AgentTree) GetAll() []*AgentNode {
    t.mu.RLock()
    defer t.mu.RUnlock()

    var result []*AgentNode
    var traverse func(*AgentNode)
    traverse = func(n *AgentNode) {
        if n.ID != "terminal" {
            result = append(result, n)
        }
        for _, child := range n.Children {
            traverse(child)
        }
    }
    traverse(t.Root)
    return result
}
```

---

## Files to Create

| File | Purpose |
|------|---------|
| `internal/tui/agents/model.go` | Agent tree data structures |
| `internal/tui/agents/model_test.go` | Unit tests for spawn/complete correlation |

---

## Acceptance Criteria

- [ ] `AgentTree` tracks all agents in session
- [ ] `HandleSpawn` adds agent to tree and pending list
- [ ] `HandleComplete` updates status and removes from pending
- [ ] Parent-child relationships correctly maintained
- [ ] Thread-safe with mutex protection
- [ ] `UpdateRunningStatus` transitions spawning -> running
- [ ] Unit tests for spawn/complete correlation

---

## Test Strategy

### Unit Tests
- Spawn single agent, verify tree structure
- Spawn multiple agents, verify pending list
- Complete agent, verify status and duration
- Nested agents (orchestrator -> python-pro)
- Concurrent access safety

### Edge Cases
- Complete event for unknown agent (no-op)
- Multiple spawns of same agent type
- Out-of-order events
