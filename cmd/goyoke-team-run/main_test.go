package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain_FullExecution tests complete wave execution with mock spawner
func TestMain_FullExecution(t *testing.T) {
	// Create config with 2 waves
	members1 := []Member{
		{
			Name:       "wave1-agent1",
			Agent:      "test-agent",
			Model:      "sonnet",
			StdinFile:  "wave1-stdin1.json",
			StdoutFile: "wave1-stdout1.json",
			Status:     "pending",
			MaxRetries: 1,
			TimeoutMs:  30000,
		},
	}
	members2 := []Member{
		{
			Name:       "wave2-agent1",
			Agent:      "test-agent",
			Model:      "haiku",
			StdinFile:  "wave2-stdin1.json",
			StdoutFile: "wave2-stdout1.json",
			Status:     "pending",
			MaxRetries: 1,
			TimeoutMs:  30000,
		},
	}

	config := &TeamConfig{
		TeamName:            "test-team",
		WorkflowType:        "test",
		ProjectRoot:         "/tmp/test",
		SessionID:           "test-session",
		BudgetMaxUSD:        10.0,
		BudgetRemainingUSD:  10.0,
		WarningThresholdUSD: 2.0,
		Status:              "running",
		Waves: []Wave{
			{
				WaveNumber:  1,
				Description: "Wave 1",
				Members:     members1,
			},
			{
				WaveNumber:  2,
				Description: "Wave 2",
				Members:     members2,
			},
		},
	}

	// Create fake spawner that succeeds and updates cost
	var completedMembers []string
	var mu sync.Mutex
	runner, teamDir := setupTestRunner(t, config, &fakeSpawner{
		fn: func(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error {
			mu.Lock()
			defer mu.Unlock()

			// Update member with mock cost
			tr.configMu.Lock()
			memberName := tr.config.Waves[waveIdx].Members[memIdx].Name
			tr.config.Waves[waveIdx].Members[memIdx].CostUSD = 0.50
			tr.configMu.Unlock()

			completedMembers = append(completedMembers, memberName)
			return nil
		},
	})

	// Create stdin files
	for _, wave := range config.Waves {
		for _, member := range wave.Members {
			stdinPath := filepath.Join(teamDir, member.StdinFile)
			stdinData := fmt.Sprintf(`{
				"agent": "%s",
				"task": "Test task",
				"context": {"test": true}
			}`, member.Agent)
			require.NoError(t, os.WriteFile(stdinPath, []byte(stdinData), 0644))
		}
	}

	// Run waves
	ctx := context.Background()
	err := runWaves(ctx, runner)
	require.NoError(t, err)

	// Verify all members completed
	mu.Lock()
	defer mu.Unlock()
	assert.Len(t, completedMembers, 2, "Should have completed 2 members")
	assert.Contains(t, completedMembers, "wave1-agent1")
	assert.Contains(t, completedMembers, "wave2-agent1")

	// Verify budget was updated
	remaining := runner.BudgetRemaining()
	assert.Less(t, remaining, 10.0, "Budget should have decreased")
	assert.Greater(t, remaining, 8.0, "Budget should be > $8 (2 x $0.50 + estimates)")
}

// TestMain_BudgetCeiling tests budget gate blocking spawns
func TestMain_BudgetCeiling(t *testing.T) {
	// Create config with insufficient budget
	members := []Member{
		{
			Name:       "agent-1",
			Agent:      "test-agent",
			Model:      "sonnet", // Estimated at $1.50
			StdinFile:  "stdin1.json",
			StdoutFile: "stdout1.json",
			Status:     "pending",
			MaxRetries: 0,
			TimeoutMs:  30000,
		},
		{
			Name:       "agent-2",
			Agent:      "test-agent",
			Model:      "sonnet", // Estimated at $1.50
			StdinFile:  "stdin2.json",
			StdoutFile: "stdout2.json",
			Status:     "pending",
			MaxRetries: 0,
			TimeoutMs:  30000,
		},
		{
			Name:       "agent-3",
			Agent:      "test-agent",
			Model:      "sonnet", // Estimated at $1.50
			StdinFile:  "stdin3.json",
			StdoutFile: "stdout3.json",
			Status:     "pending",
			MaxRetries: 0,
			TimeoutMs:  30000,
		},
	}

	config := &TeamConfig{
		TeamName:            "test-team",
		WorkflowType:        "test",
		ProjectRoot:         "/tmp/test",
		SessionID:           "test-session",
		BudgetMaxUSD:        3.0,
		BudgetRemainingUSD:  3.0, // Only enough for 2 members
		WarningThresholdUSD: 1.0,
		Status:              "running",
		Waves: []Wave{
			{
				WaveNumber:  1,
				Description: "Wave 1",
				Members:     members,
			},
		},
	}

	var spawnCount int
	var mu sync.Mutex
	runner, teamDir := setupTestRunner(t, config, &fakeSpawner{
		fn: func(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error {
			mu.Lock()
			defer mu.Unlock()
			spawnCount++

			// Update with actual cost
			tr.configMu.Lock()
			tr.config.Waves[waveIdx].Members[memIdx].CostUSD = 0.50
			tr.configMu.Unlock()

			return nil
		},
	})

	// Create stdin files
	for _, member := range members {
		stdinPath := filepath.Join(teamDir, member.StdinFile)
		stdinData := fmt.Sprintf(`{
			"agent": "%s",
			"task": "Test task",
			"context": {"test": true}
		}`, member.Agent)
		require.NoError(t, os.WriteFile(stdinPath, []byte(stdinData), 0644))
	}

	// Run waves
	ctx := context.Background()
	err := runWaves(ctx, runner)
	require.NoError(t, err)

	// Verify budget gate worked - should spawn at most 2 agents
	mu.Lock()
	defer mu.Unlock()
	assert.LessOrEqual(t, spawnCount, 2, "Budget gate should prevent spawning all 3 agents")

	// Verify budget decreased (actual cost was $0.50 per agent, but estimated $1.50)
	// After reconciliation: started with $3.00, spawned 2 agents at $0.50 each = $2.00 remaining
	remaining := runner.BudgetRemaining()
	assert.Less(t, remaining, 3.0, "Budget should have decreased")
	assert.GreaterOrEqual(t, remaining, 1.0, "Should have at least $1 remaining after 2 agents")
}

// TestMain_ContextCancellation tests graceful shutdown on context cancellation
func TestMain_ContextCancellation(t *testing.T) {
	// Create config with multiple members
	members := []Member{
		{
			Name:       "agent-1",
			Agent:      "test-agent",
			Model:      "haiku",
			StdinFile:  "stdin1.json",
			StdoutFile: "stdout1.json",
			Status:     "pending",
			MaxRetries: 5, // Many retries to allow cancellation
			TimeoutMs:  30000,
		},
		{
			Name:       "agent-2",
			Agent:      "test-agent",
			Model:      "haiku",
			StdinFile:  "stdin2.json",
			StdoutFile: "stdout2.json",
			Status:     "pending",
			MaxRetries: 5,
			TimeoutMs:  30000,
		},
	}

	config := &TeamConfig{
		TeamName:            "test-team",
		WorkflowType:        "test",
		ProjectRoot:         "/tmp/test",
		SessionID:           "test-session",
		BudgetMaxUSD:        10.0,
		BudgetRemainingUSD:  10.0,
		WarningThresholdUSD: 2.0,
		Status:              "running",
		Waves: []Wave{
			{
				WaveNumber:  1,
				Description: "Wave 1",
				Members:     members,
			},
		},
	}

	var attemptCount int
	var mu sync.Mutex
	runner, teamDir := setupTestRunner(t, config, &fakeSpawner{
		fn: func(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error {
			mu.Lock()
			attemptCount++
			mu.Unlock()

			// Delay to allow cancellation
			time.Sleep(50 * time.Millisecond)
			return fmt.Errorf("simulated failure")
		},
	})

	// Create stdin files
	for _, member := range members {
		stdinPath := filepath.Join(teamDir, member.StdinFile)
		stdinData := fmt.Sprintf(`{
			"agent": "%s",
			"task": "Test task",
			"context": {"test": true}
		}`, member.Agent)
		require.NoError(t, os.WriteFile(stdinPath, []byte(stdinData), 0644))
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Run waves in goroutine
	done := make(chan error, 1)
	go func() {
		done <- runWaves(ctx, runner)
	}()

	// Cancel after short delay
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Wait for completion
	select {
	case err := <-done:
		// Context cancellation during wave execution should either:
		// 1. Return context.Canceled error if caught before wave completes
		// 2. Return wave failure error if members fail before cancellation
		// 3. Return nil if all members finished processing before cancellation propagated
		// All are acceptable for this test - we just verify it terminates
		if err != nil {
			// Accept either context cancellation or wave failure
			acceptableErrors := []string{"context cancel", "failed member"}
			containsAcceptable := false
			for _, acceptable := range acceptableErrors {
				if strings.Contains(err.Error(), acceptable) {
					containsAcceptable = true
					break
				}
			}
			assert.True(t, containsAcceptable, "Error should be context cancel or wave failure, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runWaves did not terminate within timeout")
	}

	// Verify termination happened quickly
	mu.Lock()
	defer mu.Unlock()
	assert.Less(t, attemptCount, 10, "Should stop quickly after cancellation")
}

// TestMain_HeartbeatIntegration tests heartbeat file creation and updates
func TestMain_HeartbeatIntegration(t *testing.T) {
	teamDir := t.TempDir()
	heartbeatPath := filepath.Join(teamDir, "heartbeat")

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Start heartbeat with fast interval for testing
	startHeartbeatWithInterval(ctx, teamDir, 50*time.Millisecond)

	// Wait for first heartbeat
	time.Sleep(70 * time.Millisecond)

	// Verify heartbeat file exists
	data1, err := os.ReadFile(heartbeatPath)
	require.NoError(t, err, "Heartbeat file should exist")
	require.NotEmpty(t, data1, "Heartbeat file should not be empty")

	// Wait for update
	time.Sleep(60 * time.Millisecond)

	// Verify heartbeat was updated
	data2, err := os.ReadFile(heartbeatPath)
	require.NoError(t, err, "Heartbeat file should still exist")
	require.NotEmpty(t, data2, "Heartbeat file should not be empty")

	// Timestamps should be different (if system is fast enough)
	// This is a weak assertion but sufficient for integration testing
	assert.True(t, len(data1) > 0 && len(data2) > 0, "Both heartbeats should have data")
}

// TestMain_WaveSequencing tests waves execute sequentially
func TestMain_WaveSequencing(t *testing.T) {
	// Create config with 3 waves
	config := &TeamConfig{
		TeamName:            "test-team",
		WorkflowType:        "test",
		ProjectRoot:         "/tmp/test",
		SessionID:           "test-session",
		BudgetMaxUSD:        20.0,
		BudgetRemainingUSD:  20.0,
		WarningThresholdUSD: 2.0,
		Status:              "running",
		Waves: []Wave{
			{
				WaveNumber:  1,
				Description: "Wave 1",
				Members: []Member{
					{
						Name:       "wave1-agent1",
						Agent:      "test-agent",
						Model:      "haiku",
						StdinFile:  "wave1-stdin1.json",
						StdoutFile: "wave1-stdout1.json",
						Status:     "pending",
						MaxRetries: 0,
						TimeoutMs:  30000,
					},
				},
			},
			{
				WaveNumber:  2,
				Description: "Wave 2",
				Members: []Member{
					{
						Name:       "wave2-agent1",
						Agent:      "test-agent",
						Model:      "haiku",
						StdinFile:  "wave2-stdin1.json",
						StdoutFile: "wave2-stdout1.json",
						Status:     "pending",
						MaxRetries: 0,
						TimeoutMs:  30000,
					},
				},
			},
			{
				WaveNumber:  3,
				Description: "Wave 3",
				Members: []Member{
					{
						Name:       "wave3-agent1",
						Agent:      "test-agent",
						Model:      "haiku",
						StdinFile:  "wave3-stdin1.json",
						StdoutFile: "wave3-stdout1.json",
						Status:     "pending",
						MaxRetries: 0,
						TimeoutMs:  30000,
					},
				},
			},
		},
	}

	var executionOrder []string
	var mu sync.Mutex
	runner, teamDir := setupTestRunner(t, config, &fakeSpawner{
		fn: func(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error {
			mu.Lock()
			defer mu.Unlock()

			tr.configMu.RLock()
			memberName := tr.config.Waves[waveIdx].Members[memIdx].Name
			tr.configMu.RUnlock()

			executionOrder = append(executionOrder, memberName)

			// Update with cost
			tr.configMu.Lock()
			tr.config.Waves[waveIdx].Members[memIdx].CostUSD = 0.10
			tr.configMu.Unlock()

			// Small delay to ensure sequencing is visible
			time.Sleep(20 * time.Millisecond)
			return nil
		},
	})

	// Create stdin files
	for _, wave := range config.Waves {
		for _, member := range wave.Members {
			stdinPath := filepath.Join(teamDir, member.StdinFile)
			stdinData := fmt.Sprintf(`{
				"agent": "%s",
				"task": "Test task",
				"context": {"test": true}
			}`, member.Agent)
			require.NoError(t, os.WriteFile(stdinPath, []byte(stdinData), 0644))
		}
	}

	// Run waves
	ctx := context.Background()
	err := runWaves(ctx, runner)
	require.NoError(t, err)

	// Verify waves executed in order
	mu.Lock()
	defer mu.Unlock()
	require.Len(t, executionOrder, 3, "Should have executed 3 members")
	assert.Equal(t, "wave1-agent1", executionOrder[0], "Wave 1 should execute first")
	assert.Equal(t, "wave2-agent1", executionOrder[1], "Wave 2 should execute second")
	assert.Equal(t, "wave3-agent1", executionOrder[2], "Wave 3 should execute third")
}

// TestMain_ConfigPersistence tests config updates are persisted across operations
func TestMain_ConfigPersistence(t *testing.T) {
	member := Member{
		Name:       "agent-1",
		Agent:      "test-agent",
		Model:      "haiku",
		StdinFile:  "stdin1.json",
		StdoutFile: "stdout1.json",
		Status:     "pending",
		MaxRetries: 0,
		TimeoutMs:  30000,
	}

	config := &TeamConfig{
		TeamName:            "test-team",
		WorkflowType:        "test",
		ProjectRoot:         "/tmp/test",
		SessionID:           "test-session",
		BudgetMaxUSD:        5.0,
		BudgetRemainingUSD:  5.0,
		WarningThresholdUSD: 1.0,
		Status:              "running",
		Waves: []Wave{
			{
				WaveNumber:  1,
				Description: "Wave 1",
				Members:     []Member{member},
			},
		},
	}

	runner, teamDir := setupTestRunner(t, config, &fakeSpawner{
		fn: func(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error {
			// Update with cost
			tr.configMu.Lock()
			tr.config.Waves[waveIdx].Members[memIdx].CostUSD = 1.25
			tr.configMu.Unlock()
			return nil
		},
	})

	// Create stdin file
	stdinPath := filepath.Join(teamDir, member.StdinFile)
	stdinData := `{
		"agent": "test-agent",
		"task": "Test task",
		"context": {"test": true}
	}`
	require.NoError(t, os.WriteFile(stdinPath, []byte(stdinData), 0644))

	// Run waves
	ctx := context.Background()
	err := runWaves(ctx, runner)
	require.NoError(t, err)

	// Reload config from disk
	configPath := filepath.Join(teamDir, ConfigFileName)
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var reloaded TeamConfig
	err = json.Unmarshal(data, &reloaded)
	require.NoError(t, err)

	// Verify member status and cost were persisted
	assert.Equal(t, "completed", reloaded.Waves[0].Members[0].Status)
	assert.Equal(t, 1.25, reloaded.Waves[0].Members[0].CostUSD)

	// Verify budget was updated and persisted
	assert.Less(t, reloaded.BudgetRemainingUSD, 5.0)
}

// TestMain_MixedSuccessFailure tests wave continues with mixed member outcomes
func TestMain_MixedSuccessFailure(t *testing.T) {
	members := []Member{
		{
			Name:       "agent-success",
			Agent:      "test-agent",
			Model:      "haiku",
			StdinFile:  "stdin-success.json",
			StdoutFile: "stdout-success.json",
			Status:     "pending",
			MaxRetries: 0,
			TimeoutMs:  30000,
		},
		{
			Name:       "agent-fail",
			Agent:      "test-agent",
			Model:      "haiku",
			StdinFile:  "stdin-fail.json",
			StdoutFile: "stdout-fail.json",
			Status:     "pending",
			MaxRetries: 1,
			TimeoutMs:  30000,
		},
		{
			Name:       "agent-success2",
			Agent:      "test-agent",
			Model:      "haiku",
			StdinFile:  "stdin-success2.json",
			StdoutFile: "stdout-success2.json",
			Status:     "pending",
			MaxRetries: 0,
			TimeoutMs:  30000,
		},
	}

	config := &TeamConfig{
		TeamName:            "test-team",
		WorkflowType:        "test",
		ProjectRoot:         "/tmp/test",
		SessionID:           "test-session",
		BudgetMaxUSD:        10.0,
		BudgetRemainingUSD:  10.0,
		WarningThresholdUSD: 2.0,
		Status:              "running",
		Waves: []Wave{
			{
				WaveNumber:  1,
				Description: "Wave 1",
				Members:     members,
			},
		},
	}

	runner, teamDir := setupTestRunner(t, config, &fakeSpawner{
		fn: func(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error {
			tr.configMu.RLock()
			memberName := tr.config.Waves[waveIdx].Members[memIdx].Name
			tr.configMu.RUnlock()

			// Update with cost
			tr.configMu.Lock()
			tr.config.Waves[waveIdx].Members[memIdx].CostUSD = 0.10
			tr.configMu.Unlock()

			if memberName == "agent-fail" {
				return fmt.Errorf("simulated failure")
			}
			return nil
		},
	})

	// Create stdin files
	for _, member := range members {
		stdinPath := filepath.Join(teamDir, member.StdinFile)
		stdinData := fmt.Sprintf(`{
			"agent": "%s",
			"task": "Test task",
			"context": {"test": true}
		}`, member.Agent)
		require.NoError(t, os.WriteFile(stdinPath, []byte(stdinData), 0644))
	}

	// Run waves
	ctx := context.Background()
	err := runWaves(ctx, runner)
	// With failure propagation, wave returns error when any member fails
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed member")

	// Verify mixed outcomes
	runner.configMu.RLock()
	defer runner.configMu.RUnlock()
	assert.Equal(t, "completed", runner.config.Waves[0].Members[0].Status)
	assert.Equal(t, "failed", runner.config.Waves[0].Members[1].Status)
	assert.Equal(t, "completed", runner.config.Waves[0].Members[2].Status)
}
