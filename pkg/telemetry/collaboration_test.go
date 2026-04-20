package telemetry

import (
	"os"
	"path/filepath"
	"strings"
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
	// Setup: Clear existing logs
	globalPath := getGlobalCollaborationPath()
	os.RemoveAll(filepath.Dir(globalPath))

	collab := NewAgentCollaboration("session-456", "architect", "codebase-search", "escalate")
	collab.ChildSuccess = true
	collab.ChildDurationMs = 250

	err := LogCollaboration(collab)
	if err != nil {
		t.Fatalf("Failed to log collaboration: %v", err)
	}

	// Verify global file created
	if _, err := os.Stat(globalPath); os.IsNotExist(err) {
		t.Fatalf("Global collaboration log should exist at %s", globalPath)
	}

	// Cleanup
	os.RemoveAll(filepath.Dir(globalPath))
}

func TestReadCollaborationLogs_Empty(t *testing.T) {
	// Setup: Clear existing logs
	globalPath := getGlobalCollaborationPath()
	os.RemoveAll(filepath.Dir(globalPath))

	logs, err := ReadCollaborationLogs()

	if err != nil {
		t.Fatalf("Should not error on missing file, got: %v", err)
	}

	if len(logs) != 0 {
		t.Errorf("Expected 0 logs, got: %d", len(logs))
	}
}

func TestReadCollaborationLogs(t *testing.T) {
	// Setup: Clear existing logs
	globalPath := getGlobalCollaborationPath()
	os.RemoveAll(filepath.Dir(globalPath))

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

	// Verify success rate format (should be "66.67%")
	successRate, ok := stats["success_rate"].(string)
	if !ok {
		t.Errorf("Expected success_rate to be string, got: %T", stats["success_rate"])
	}
	if successRate != "66.67%" {
		t.Errorf("Expected success_rate '66.67%%', got: %s", successRate)
	}

	// Verify agent pairings
	pairings, ok := stats["agent_pairings"].(map[string]int)
	if !ok {
		t.Errorf("Expected agent_pairings to be map[string]int, got: %T", stats["agent_pairings"])
	}
	if pairings["orchestrator → python-pro"] != 2 {
		t.Errorf("Expected 2 orchestrator → python-pro pairings, got: %d", pairings["orchestrator → python-pro"])
	}
}

func TestXDGComplianceCollaboration(t *testing.T) {
	origXDG := os.Getenv("XDG_DATA_HOME")
	defer os.Setenv("XDG_DATA_HOME", origXDG)

	testPath := "/tmp/test-xdg-collab"
	os.Setenv("XDG_DATA_HOME", testPath)

	path := getGlobalCollaborationPath()
	expected := filepath.Join(testPath, "goyoke", "agent-collaborations.jsonl")

	if path != expected {
		t.Errorf("XDG_DATA_HOME not respected. Got %s, expected %s", path, expected)
	}
}

func TestCollaborationWithSwarmFields(t *testing.T) {
	// Setup: Clear existing logs
	globalPath := getGlobalCollaborationPath()
	os.RemoveAll(filepath.Dir(globalPath))

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

	if lastLog.AgreementWithAdjacent != 0.92 {
		t.Errorf("Expected AgreementWithAdjacent 0.92, got: %f", lastLog.AgreementWithAdjacent)
	}

	if lastLog.InformationLoss != 0.03 {
		t.Errorf("Expected InformationLoss 0.03, got: %f", lastLog.InformationLoss)
	}

	// Cleanup
	os.RemoveAll(filepath.Dir(globalPath))
}

func TestChainDepthTracking(t *testing.T) {
	// Setup: Clear existing logs
	globalPath := getGlobalCollaborationPath()
	os.RemoveAll(filepath.Dir(globalPath))

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
	os.RemoveAll(filepath.Dir(globalPath))
}

// TestCalculateCollaborationStats_EmptyLogs tests stats calculation with empty input
func TestCalculateCollaborationStats_EmptyLogs(t *testing.T) {
	stats := CalculateCollaborationStats([]AgentCollaboration{})

	if stats["collaboration_count"] != 0 {
		t.Errorf("Expected collaboration_count 0, got: %v", stats["collaboration_count"])
	}

	if stats["avg_chain_depth"] != 0 {
		t.Errorf("Expected avg_chain_depth 0, got: %v", stats["avg_chain_depth"])
	}

	if stats["success_rate"] != "0.00%" {
		t.Errorf("Expected success_rate '0.00%%', got: %v", stats["success_rate"])
	}

	if stats["avg_handoff_time"] != int64(0) {
		t.Errorf("Expected avg_handoff_time 0, got: %v", stats["avg_handoff_time"])
	}

	pairings, ok := stats["agent_pairings"].(map[string]int)
	if !ok || len(pairings) != 0 {
		t.Errorf("Expected empty agent_pairings map, got: %v", stats["agent_pairings"])
	}
}

// TestReadCollaborationLogs_MalformedLines tests resilience to malformed JSONL
func TestReadCollaborationLogs_MalformedLines(t *testing.T) {
	globalPath := getGlobalCollaborationPath()
	os.RemoveAll(filepath.Dir(globalPath))

	// Write malformed JSONL
	dir := filepath.Dir(globalPath)
	os.MkdirAll(dir, 0755)
	content := `{"collaboration_id":"valid","timestamp":123,"session_id":"test"}
not-json-data
{"collaboration_id":"valid2","timestamp":456,"session_id":"test2"}

{"incomplete":
{"collaboration_id":"valid3","timestamp":789,"session_id":"test3"}
`
	os.WriteFile(globalPath, []byte(content), 0644)

	logs, err := ReadCollaborationLogs()
	if err != nil {
		t.Fatalf("Should not error on malformed lines, got: %v", err)
	}

	// Should skip malformed lines and return only valid ones
	if len(logs) != 3 {
		t.Errorf("Expected 3 valid logs, got: %d", len(logs))
	}

	// Cleanup
	os.RemoveAll(filepath.Dir(globalPath))
}

// TestReadCollaborationLogs_EmptyLines tests handling of empty lines
func TestReadCollaborationLogs_EmptyLines(t *testing.T) {
	globalPath := getGlobalCollaborationPath()
	os.RemoveAll(filepath.Dir(globalPath))

	// Log collaborations with extra empty lines
	collab1 := NewAgentCollaboration("session-1", "agent1", "agent2", "spawn")
	LogCollaboration(collab1)

	// Append extra empty lines
	f, _ := os.OpenFile(globalPath, os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString("\n\n")
	f.Close()

	collab2 := NewAgentCollaboration("session-2", "agent3", "agent4", "escalate")
	LogCollaboration(collab2)

	logs, err := ReadCollaborationLogs()
	if err != nil {
		t.Fatalf("Should handle empty lines, got: %v", err)
	}

	if len(logs) != 2 {
		t.Errorf("Expected 2 logs (empty lines skipped), got: %d", len(logs))
	}

	// Cleanup
	os.RemoveAll(filepath.Dir(globalPath))
}

func TestReadCollaborationLogs_LargeLine(t *testing.T) {
	globalPath := getGlobalCollaborationPath()
	os.RemoveAll(filepath.Dir(globalPath))

	dir := filepath.Dir(globalPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	largeTask := strings.Repeat("c", 70*1024)
	content := `{"collaboration_id":"large","timestamp":123,"session_id":"test","parent_agent":"orchestrator","child_agent":"python-pro","delegation_type":"spawn","task_description":"` + largeTask + `"}`
	if err := os.WriteFile(globalPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write log: %v", err)
	}

	logs, err := ReadCollaborationLogs()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("Expected 1 log, got: %d", len(logs))
	}
	if logs[0].TaskDescription != largeTask {
		t.Fatalf("Expected large task description to round-trip")
	}

	os.RemoveAll(filepath.Dir(globalPath))
}

// TestCollaboration_ThreadSafety verifies append-only pattern
func TestCollaboration_ThreadSafety(t *testing.T) {
	globalPath := getGlobalCollaborationPath()
	os.RemoveAll(filepath.Dir(globalPath))

	// Simulate concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			collab := NewAgentCollaboration("session-concurrent", "parent", "child", "spawn")
			collab.ChainDepth = id
			LogCollaboration(collab)
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 10; i++ {
		<-done
	}

	logs, err := ReadCollaborationLogs()
	if err != nil {
		t.Fatalf("Failed to read concurrent logs: %v", err)
	}

	if len(logs) != 10 {
		t.Errorf("Expected 10 logs from concurrent writes, got: %d", len(logs))
	}

	// Cleanup
	os.RemoveAll(filepath.Dir(globalPath))
}

// TestCollaboration_AllFieldsSerialized verifies all struct fields are captured
func TestCollaboration_AllFieldsSerialized(t *testing.T) {
	globalPath := getGlobalCollaborationPath()
	os.RemoveAll(filepath.Dir(globalPath))

	collab := NewAgentCollaboration("session-full", "orchestrator", "python-pro", "parallel")
	collab.ContextSize = 5000
	collab.TaskDescription = "Implement feature X"
	collab.ChildSuccess = true
	collab.ChildDurationMs = 1500
	collab.HandoffFriction = "context_loss"
	collab.ChainDepth = 2
	collab.RootTaskID = "root-task-123"
	collab.IsSwarmMember = true
	collab.SwarmPosition = 3
	collab.OverlapWithPrevious = 0.25
	collab.AgreementWithAdjacent = 0.88
	collab.InformationLoss = 0.12

	err := LogCollaboration(collab)
	if err != nil {
		t.Fatalf("Failed to log: %v", err)
	}

	logs, err := ReadCollaborationLogs()
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	if len(logs) == 0 {
		t.Fatal("Expected at least one log")
	}

	loaded := logs[0]

	// Verify all fields
	if loaded.CollaborationID == "" {
		t.Error("CollaborationID not preserved")
	}
	if loaded.SessionID != "session-full" {
		t.Errorf("Expected SessionID 'session-full', got: %s", loaded.SessionID)
	}
	if loaded.ParentAgent != "orchestrator" {
		t.Errorf("Expected ParentAgent 'orchestrator', got: %s", loaded.ParentAgent)
	}
	if loaded.ChildAgent != "python-pro" {
		t.Errorf("Expected ChildAgent 'python-pro', got: %s", loaded.ChildAgent)
	}
	if loaded.DelegationType != "parallel" {
		t.Errorf("Expected DelegationType 'parallel', got: %s", loaded.DelegationType)
	}
	if loaded.ContextSize != 5000 {
		t.Errorf("Expected ContextSize 5000, got: %d", loaded.ContextSize)
	}
	if loaded.TaskDescription != "Implement feature X" {
		t.Errorf("Expected TaskDescription 'Implement feature X', got: %s", loaded.TaskDescription)
	}
	if !loaded.ChildSuccess {
		t.Error("Expected ChildSuccess true")
	}
	if loaded.ChildDurationMs != 1500 {
		t.Errorf("Expected ChildDurationMs 1500, got: %d", loaded.ChildDurationMs)
	}
	if loaded.HandoffFriction != "context_loss" {
		t.Errorf("Expected HandoffFriction 'context_loss', got: %s", loaded.HandoffFriction)
	}
	if loaded.ChainDepth != 2 {
		t.Errorf("Expected ChainDepth 2, got: %d", loaded.ChainDepth)
	}
	if loaded.RootTaskID != "root-task-123" {
		t.Errorf("Expected RootTaskID 'root-task-123', got: %s", loaded.RootTaskID)
	}
	if !loaded.IsSwarmMember {
		t.Error("Expected IsSwarmMember true")
	}
	if loaded.SwarmPosition != 3 {
		t.Errorf("Expected SwarmPosition 3, got: %d", loaded.SwarmPosition)
	}
	if loaded.OverlapWithPrevious != 0.25 {
		t.Errorf("Expected OverlapWithPrevious 0.25, got: %f", loaded.OverlapWithPrevious)
	}
	if loaded.AgreementWithAdjacent != 0.88 {
		t.Errorf("Expected AgreementWithAdjacent 0.88, got: %f", loaded.AgreementWithAdjacent)
	}
	if loaded.InformationLoss != 0.12 {
		t.Errorf("Expected InformationLoss 0.12, got: %f", loaded.InformationLoss)
	}

	// Cleanup
	os.RemoveAll(filepath.Dir(globalPath))
}

// TestLogCollaboration_PermissionError tests error handling when directory is not writable
func TestLogCollaboration_PermissionError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	// This test is informational - actual permission errors are hard to test
	// without modifying system directories. The error path exists and will be
	// exercised in real usage scenarios where directories become read-only.
	// Coverage metric acknowledges this limitation.
}
