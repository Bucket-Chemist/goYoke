---
id: GOgent-088b
title: Agent Collaboration Tracking
description: Implement AgentCollaboration struct and logging for team makeup optimization
status: pending
time_estimate: 2h
dependencies: ["GOgent-088"]
priority: high
week: 4
tags: ["ml-optimization", "collaboration-tracking", "week-4"]
tests_required: true
acceptance_criteria_count: 8
---

### GOgent-088b: Agent Collaboration Tracking

**Time**: 2 hours
**Dependencies**: GOgent-088

**Task**:
Implement collaboration tracking for team makeup optimization and delegation chain analysis.

**File**: `pkg/telemetry/collaboration.go`

**Imports**:
```go
package telemetry

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "time"

    "github.com/google/uuid"
)
```

**Implementation**:
```go
// AgentCollaboration captures a delegation relationship for ML analysis
type AgentCollaboration struct {
    CollaborationID string `json:"collaboration_id"` // UUID
    Timestamp       int64  `json:"timestamp"`
    SessionID       string `json:"session_id"`

    // Delegation relationship
    ParentAgent     string `json:"parent_agent"`     // orchestrator, architect, etc.
    ChildAgent      string `json:"child_agent"`      // python-pro, codebase-search, etc.
    DelegationType  string `json:"delegation_type"`  // "spawn", "escalate", "parallel"

    // Context transfer
    ContextSize     int    `json:"context_size"`     // Tokens passed to child
    TaskDescription string `json:"task_description"` // What was delegated (truncated)

    // Outcome
    ChildSuccess    bool   `json:"child_success"`
    ChildDurationMs int64  `json:"child_duration_ms"`
    HandoffFriction string `json:"handoff_friction,omitempty"` // "context_loss", "misunderstanding", "none"

    // Chain position
    ChainDepth int    `json:"chain_depth"` // 0 = root, 1 = first delegation, etc.
    RootTaskID string `json:"root_task_id"` // Original task that spawned chain

    // Swarm coordination (Addendum A.3)
    IsSwarmMember         bool    `json:"is_swarm_member,omitempty"`
    SwarmPosition         int     `json:"swarm_position,omitempty"`
    OverlapWithPrevious   float64 `json:"overlap_with_previous,omitempty"`
    AgreementWithAdjacent float64 `json:"agreement_with_adjacent,omitempty"`
    InformationLoss       float64 `json:"information_loss,omitempty"`
}

// NewAgentCollaboration creates a new collaboration record
func NewAgentCollaboration(sessionID, parentAgent, childAgent, delegationType string) *AgentCollaboration {
    return &AgentCollaboration{
        CollaborationID: uuid.New().String(),
        Timestamp:       time.Now().Unix(),
        SessionID:       sessionID,
        ParentAgent:     parentAgent,
        ChildAgent:      childAgent,
        DelegationType:  delegationType,
    }
}

// LogCollaboration writes collaboration to JSONL storage
func LogCollaboration(collab *AgentCollaboration) error {
    globalPath := getGlobalCollaborationPath()

    data, err := json.Marshal(collab)
    if err != nil {
        return fmt.Errorf("[collaboration] Failed to marshal: %w", err)
    }

    dir := filepath.Dir(globalPath)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return fmt.Errorf("[collaboration] Failed to create directory: %w", err)
    }

    f, err := os.OpenFile(globalPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return fmt.Errorf("[collaboration] Failed to open log: %w", err)
    }
    defer f.Close()

    if _, err := f.WriteString(string(data) + "\n"); err != nil {
        return fmt.Errorf("[collaboration] Failed to write: %w", err)
    }

    return nil
}

// getGlobalCollaborationPath returns XDG-compliant global collaboration log path
func getGlobalCollaborationPath() string {
    xdgData := os.Getenv("XDG_DATA_HOME")
    if xdgData == "" {
        home, _ := os.UserHomeDir()
        xdgData = filepath.Join(home, ".local", "share")
    }
    return filepath.Join(xdgData, "gogent", "agent-collaborations.jsonl")
}

// ReadCollaborationLogs reads all collaboration logs from the global path
func ReadCollaborationLogs() ([]AgentCollaboration, error) {
    logPath := getGlobalCollaborationPath()

    data, err := os.ReadFile(logPath)
    if os.IsNotExist(err) {
        return []AgentCollaboration{}, nil
    }
    if err != nil {
        return nil, fmt.Errorf("[collaboration] Failed to read logs: %w", err)
    }

    var logs []AgentCollaboration
    offset := 0
    content := string(data)

    for {
        // Find next newline
        newlineIdx := -1
        for i := offset; i < len(content); i++ {
            if content[i] == '\n' {
                newlineIdx = i
                break
            }
        }

        if newlineIdx == -1 {
            if offset < len(content) {
                newlineIdx = len(content)
            } else {
                break
            }
        }

        line := content[offset:newlineIdx]
        if line == "" {
            offset = newlineIdx + 1
            continue
        }

        var log AgentCollaboration
        if err := json.Unmarshal([]byte(line), &log); err != nil {
            // Skip malformed lines
            offset = newlineIdx + 1
            continue
        }

        logs = append(logs, log)
        offset = newlineIdx + 1
    }

    return logs, nil
}

// CalculateCollaborationStats returns aggregate metrics for agent collaborations
func CalculateCollaborationStats(logs []AgentCollaboration) map[string]interface{} {
    if len(logs) == 0 {
        return map[string]interface{}{
            "collaboration_count": 0,
            "avg_chain_depth":     0,
            "success_rate":        0.0,
            "avg_handoff_time":    0,
        }
    }

    var successCount int
    var totalChainDepth int
    var totalDuration int64
    agentPairings := make(map[string]int)

    for _, log := range logs {
        if log.ChildSuccess {
            successCount++
        }
        totalChainDepth += log.ChainDepth
        totalDuration += log.ChildDurationMs

        pairing := log.ParentAgent + " → " + log.ChildAgent
        agentPairings[pairing]++
    }

    successRate := float64(successCount) / float64(len(logs)) * 100
    avgChainDepth := totalChainDepth / len(logs)
    avgDuration := totalDuration / int64(len(logs))

    return map[string]interface{}{
        "collaboration_count": len(logs),
        "avg_chain_depth":     avgChainDepth,
        "success_rate":        fmt.Sprintf("%.2f%%", successRate),
        "avg_handoff_time":    avgDuration,
        "agent_pairings":      agentPairings,
    }
}
```

**Tests**: `pkg/telemetry/collaboration_test.go`

```go
package telemetry

import (
    "os"
    "path/filepath"
    "testing"
)

func TestNewAgentCollaboration(t *testing.T) {
    collab := NewAgentCollaboration("session-123", "orchestrator", "python-pro", "spawn")

    if collab.CollaborationID == "" {
        t.Error("CollaborationID should be populated with UUID")
    }

    if collab.SessionID != "session-123" {
        t.Errorf("Expected SessionID 'session-123', got: %s", collab.SessionID)
    }

    if collab.ParentAgent != "orchestrator" {
        t.Errorf("Expected ParentAgent 'orchestrator', got: %s", collab.ParentAgent)
    }

    if collab.ChildAgent != "python-pro" {
        t.Errorf("Expected ChildAgent 'python-pro', got: %s", collab.ChildAgent)
    }

    if collab.DelegationType != "spawn" {
        t.Errorf("Expected DelegationType 'spawn', got: %s", collab.DelegationType)
    }

    if collab.Timestamp == 0 {
        t.Error("Timestamp should be populated")
    }
}

func TestLogCollaboration_GlobalPath(t *testing.T) {
    collab := NewAgentCollaboration("session-456", "architect", "codebase-search", "escalate")
    collab.ChildSuccess = true
    collab.ChildDurationMs = 250

    err := LogCollaboration(collab)
    if err != nil {
        t.Fatalf("Failed to log collaboration: %v", err)
    }

    // Verify global file created
    globalPath := getGlobalCollaborationPath()
    if _, err := os.Stat(globalPath); os.IsNotExist(err) {
        t.Fatalf("Global collaboration log should exist at %s", globalPath)
    }

    // Cleanup
    os.RemoveAll(filepath.Dir(globalPath))
}

func TestReadCollaborationLogs_Empty(t *testing.T) {
    logs, err := ReadCollaborationLogs()

    if err != nil {
        t.Fatalf("Should not error on missing file, got: %v", err)
    }

    if len(logs) != 0 {
        t.Errorf("Expected 0 logs, got: %d", len(logs))
    }
}

func TestReadCollaborationLogs(t *testing.T) {
    // Log multiple collaborations
    for i := 0; i < 3; i++ {
        collab := NewAgentCollaboration("session-789", "orchestrator", "python-pro", "spawn")
        collab.ChildSuccess = i%2 == 0
        collab.ChildDurationMs = int64(100 + i*50)
        collab.ChainDepth = i
        LogCollaboration(collab)
    }

    logs, err := ReadCollaborationLogs()

    if err != nil {
        t.Fatalf("Failed to read: %v", err)
    }

    if len(logs) != 3 {
        t.Errorf("Expected 3 logs, got: %d", len(logs))
    }

    // Verify fields populated
    if logs[0].SessionID != "session-789" {
        t.Errorf("Expected SessionID populated in read logs")
    }

    // Cleanup
    globalPath := getGlobalCollaborationPath()
    os.RemoveAll(filepath.Dir(globalPath))
}

func TestCalculateCollaborationStats(t *testing.T) {
    logs := []AgentCollaboration{
        {
            ParentAgent:     "orchestrator",
            ChildAgent:      "python-pro",
            ChildSuccess:    true,
            ChildDurationMs: 100,
            ChainDepth:      0,
        },
        {
            ParentAgent:     "architect",
            ChildAgent:      "codebase-search",
            ChildSuccess:    true,
            ChildDurationMs: 200,
            ChainDepth:      1,
        },
        {
            ParentAgent:     "orchestrator",
            ChildAgent:      "python-pro",
            ChildSuccess:    false,
            ChildDurationMs: 150,
            ChainDepth:      0,
        },
    }

    stats := CalculateCollaborationStats(logs)

    if stats["collaboration_count"] != 3 {
        t.Errorf("Expected 3 collaborations, got: %v", stats["collaboration_count"])
    }

    if stats["avg_chain_depth"] != 0 {
        t.Errorf("Expected avg_chain_depth 0, got: %v", stats["avg_chain_depth"])
    }

    if stats["avg_handoff_time"] != int64(150) {
        t.Errorf("Expected avg_handoff_time 150, got: %v", stats["avg_handoff_time"])
    }
}

func TestXDGComplianceCollaboration(t *testing.T) {
    origXDG := os.Getenv("XDG_DATA_HOME")
    defer os.Setenv("XDG_DATA_HOME", origXDG)

    testPath := "/tmp/test-xdg-collab"
    os.Setenv("XDG_DATA_HOME", testPath)

    path := getGlobalCollaborationPath()
    expected := filepath.Join(testPath, "gogent", "agent-collaborations.jsonl")

    if path != expected {
        t.Errorf("XDG_DATA_HOME not respected. Got %s, expected %s", path, expected)
    }
}

func TestCollaborationWithSwarmFields(t *testing.T) {
    collab := NewAgentCollaboration("session-swarm", "orchestrator", "python-pro", "parallel")
    collab.IsSwarmMember = true
    collab.SwarmPosition = 1
    collab.OverlapWithPrevious = 0.15
    collab.AgreementWithAdjacent = 0.92
    collab.InformationLoss = 0.03

    err := LogCollaboration(collab)
    if err != nil {
        t.Fatalf("Failed to log collaboration with swarm fields: %v", err)
    }

    logs, err := ReadCollaborationLogs()
    if err != nil {
        t.Fatalf("Failed to read: %v", err)
    }

    if len(logs) == 0 {
        t.Fatal("Expected at least one log entry")
    }

    lastLog := logs[len(logs)-1]
    if !lastLog.IsSwarmMember {
        t.Error("IsSwarmMember should be true")
    }

    if lastLog.SwarmPosition != 1 {
        t.Errorf("Expected SwarmPosition 1, got: %d", lastLog.SwarmPosition)
    }

    if lastLog.OverlapWithPrevious != 0.15 {
        t.Errorf("Expected OverlapWithPrevious 0.15, got: %f", lastLog.OverlapWithPrevious)
    }

    // Cleanup
    globalPath := getGlobalCollaborationPath()
    os.RemoveAll(filepath.Dir(globalPath))
}

func TestChainDepthTracking(t *testing.T) {
    // Simulate a delegation chain
    depths := []int{0, 1, 2}
    for _, depth := range depths {
        collab := NewAgentCollaboration("session-chain", "root", "child", "spawn")
        collab.ChainDepth = depth
        collab.RootTaskID = "task-root-001"
        LogCollaboration(collab)
    }

    logs, err := ReadCollaborationLogs()
    if err != nil {
        t.Fatalf("Failed to read: %v", err)
    }

    if len(logs) != 3 {
        t.Errorf("Expected 3 chain entries, got: %d", len(logs))
    }

    for i, log := range logs {
        if log.ChainDepth != depths[i] {
            t.Errorf("Log %d: expected ChainDepth %d, got: %d", i, depths[i], log.ChainDepth)
        }
        if log.RootTaskID != "task-root-001" {
            t.Errorf("Log %d: expected RootTaskID 'task-root-001', got: %s", i, log.RootTaskID)
        }
    }

    // Cleanup
    globalPath := getGlobalCollaborationPath()
    os.RemoveAll(filepath.Dir(globalPath))
}
```

**Acceptance Criteria**:
- [x] `AgentCollaboration` struct implemented with all GAP 4.3 fields
- [x] Swarm coordination fields included (Addendum A.3)
- [x] `NewAgentCollaboration()` creates records with UUID and timestamp
- [x] `LogCollaboration()` writes to XDG-compliant global path
- [x] Chain depth tracking works correctly
- [x] Parent-child success correlation captured
- [x] `ReadCollaborationLogs()` parses logs correctly
- [x] Thread-safe update pattern implemented (append-only or file locking)
- [x] `go test ./pkg/telemetry` passes

**Thread Safety Note**: If collaboration outcomes need updates after initial logging, follow the append-only pattern from GOgent-087b to avoid race conditions under parallel agent execution.

**Why This Matters**: Collaboration tracking enables ML-based team makeup optimization by capturing delegation patterns, success rates by agent pairing, and swarm coordination metrics for parallel agent workflows. This data feeds into routing ML model training per GAP Section 4.3 and Addendum A.3.

---
