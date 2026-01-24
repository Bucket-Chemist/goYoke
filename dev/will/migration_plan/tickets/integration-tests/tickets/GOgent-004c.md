---
id: GOgent-004c
title: Config Circular Dependency Tests
description: **Task**:
status: pending
time_estimate: 1h
dependencies: ["GOgent-004a","GOgent-004b"]
priority: high
week: 5
tags: ["config-tests", "week-5", "deferred"]
tests_required: true
acceptance_criteria_count: 7
---

### GOgent-004c: Config Circular Dependency Tests

**Time**: 1 hour
**Dependencies**: GOgent-004a, GOgent-004b (from Week 1)

**Task**:
Complete config package tests by adding circular dependency detection and multi-agent config loading tests. Deferred from Week 1 to avoid blocking event parsing work.

**File**: `pkg/config/loader_test.go` (append to existing tests)

**Implementation**:

Add test cases to existing test file:

```go
// Circular dependency detection test
func TestLoadAgentConfig_CircularDependency(t *testing.T) {
	// Create temp agent configs with circular dependency
	tmpDir := t.TempDir()

	// Agent A requires Agent B
	agentAConfig := `{
		"agent_id": "agent-a",
		"requires": ["agent-b"],
		"tier": "haiku"
	}`

	// Agent B requires Agent A (circular)
	agentBConfig := `{
		"agent_id": "agent-b",
		"requires": ["agent-a"],
		"tier": "haiku"
	}`

	os.MkdirAll(filepath.Join(tmpDir, "agents", "agent-a"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "agents", "agent-b"), 0755)

	os.WriteFile(filepath.Join(tmpDir, "agents", "agent-a", "agent.json"), []byte(agentAConfig), 0644)
	os.WriteFile(filepath.Join(tmpDir, "agents", "agent-b", "agent.json"), []byte(agentBConfig), 0644)

	// Attempt to load agent-a should detect circular dependency
	_, err := LoadAgentConfig(filepath.Join(tmpDir, "agents", "agent-a"))
	if err == nil {
		t.Fatal("Expected error for circular dependency, got nil")
	}

	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("Expected 'circular' in error message, got: %v", err)
	}
}

// Multi-level dependency resolution test
func TestLoadAgentConfig_MultiLevelDependencies(t *testing.T) {
	tmpDir := t.TempDir()

	// Create agent chain: C → B → A
	agentAConfig := `{
		"agent_id": "agent-a",
		"tier": "haiku",
		"tools_allowed": ["Read", "Glob"]
	}`

	agentBConfig := `{
		"agent_id": "agent-b",
		"requires": ["agent-a"],
		"tier": "haiku_thinking"
	}`

	agentCConfig := `{
		"agent_id": "agent-c",
		"requires": ["agent-b"],
		"tier": "sonnet"
	}`

	os.MkdirAll(filepath.Join(tmpDir, "agents", "agent-a"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "agents", "agent-b"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "agents", "agent-c"), 0755)

	os.WriteFile(filepath.Join(tmpDir, "agents", "agent-a", "agent.json"), []byte(agentAConfig), 0644)
	os.WriteFile(filepath.Join(tmpDir, "agents", "agent-b", "agent.json"), []byte(agentBConfig), 0644)
	os.WriteFile(filepath.Join(tmpDir, "agents", "agent-c", "agent.json"), []byte(agentCConfig), 0644)

	// Load agent-c should load all dependencies
	config, err := LoadAgentConfig(filepath.Join(tmpDir, "agents", "agent-c"))
	if err != nil {
		t.Fatalf("Failed to load multi-level dependencies: %v", err)
	}

	if config.AgentID != "agent-c" {
		t.Errorf("Expected agent-c, got: %s", config.AgentID)
	}

	// Verify dependencies loaded
	if len(config.Requires) != 1 || config.Requires[0] != "agent-b" {
		t.Errorf("Expected requires=[agent-b], got: %v", config.Requires)
	}
}

// Missing dependency test
func TestLoadAgentConfig_MissingDependency(t *testing.T) {
	tmpDir := t.TempDir()

	// Agent references non-existent dependency
	agentConfig := `{
		"agent_id": "test-agent",
		"requires": ["nonexistent-agent"],
		"tier": "haiku"
	}`

	os.MkdirAll(filepath.Join(tmpDir, "agents", "test-agent"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "agents", "test-agent", "agent.json"), []byte(agentConfig), 0644)

	_, err := LoadAgentConfig(filepath.Join(tmpDir, "agents", "test-agent"))
	if err == nil {
		t.Fatal("Expected error for missing dependency, got nil")
	}

	if !strings.Contains(err.Error(), "nonexistent-agent") {
		t.Errorf("Expected 'nonexistent-agent' in error, got: %v", err)
	}
}

// Concurrent config loading test (verify thread safety)
func TestLoadAgentConfig_Concurrent(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple agent configs
	for i := 0; i < 10; i++ {
		agentID := fmt.Sprintf("agent-%d", i)
		agentConfig := fmt.Sprintf(`{
			"agent_id": "%s",
			"tier": "haiku",
			"tools_allowed": ["Read"]
		}`, agentID)

		os.MkdirAll(filepath.Join(tmpDir, "agents", agentID), 0755)
		os.WriteFile(filepath.Join(tmpDir, "agents", agentID, "agent.json"), []byte(agentConfig), 0644)
	}

	// Load all configs concurrently
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			agentID := fmt.Sprintf("agent-%d", index)
			_, err := LoadAgentConfig(filepath.Join(tmpDir, "agents", agentID))
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent load failed: %v", err)
	}
}
```

**Acceptance Criteria**:
- [ ] `TestLoadAgentConfig_CircularDependency` detects circular dependencies
- [ ] Error message includes "circular" for circular dependency detection
- [ ] `TestLoadAgentConfig_MultiLevelDependencies` loads transitive dependencies
- [ ] `TestLoadAgentConfig_MissingDependency` reports missing dependencies
- [ ] `TestLoadAgentConfig_Concurrent` verifies thread-safe config loading
- [ ] All tests pass: `go test ./pkg/config -v`
- [ ] Coverage for config package ≥80%

**Why This Matters**: Deferred from Week 1 to unblock event parsing work. Config loading must handle complex dependency graphs without deadlocks or crashes. Thread safety critical for production use.

---
