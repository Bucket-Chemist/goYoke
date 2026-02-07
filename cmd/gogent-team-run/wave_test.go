package main

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunWaves_SingleWave tests a single wave with 2 members completing successfully
func TestRunWaves_SingleWave(t *testing.T) {
	members := []Member{
		{Name: "agent-1", Agent: "test-agent", Model: "haiku", Status: "pending", MaxRetries: 1},
		{Name: "agent-2", Agent: "test-agent", Model: "haiku", Status: "pending", MaxRetries: 1},
	}

	config := testConfig(members...)
	config.BudgetMaxUSD = 5.00
	config.BudgetRemainingUSD = 5.00

	var spawnCount atomic.Int32
	runner, _ := setupTestRunner(t, config, &fakeSpawner{
		fn: func(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error {
			spawnCount.Add(1)
			// Simulate cost extraction
			tr.configMu.Lock()
			tr.config.Waves[waveIdx].Members[memIdx].CostUSD = 0.05
			tr.configMu.Unlock()
			return nil
		},
	})

	ctx := context.Background()
	err := runWaves(ctx, runner)
	require.NoError(t, err)

	// Verify both members completed
	runner.configMu.RLock()
	defer runner.configMu.RUnlock()
	assert.Equal(t, "completed", runner.config.Waves[0].Members[0].Status)
	assert.Equal(t, "completed", runner.config.Waves[0].Members[1].Status)
	assert.Equal(t, int32(2), spawnCount.Load())
}

// TestRunWaves_Sequential tests that waves execute sequentially (wave 2 starts after wave 1)
func TestRunWaves_Sequential(t *testing.T) {
	config := &TeamConfig{
		TeamName:            "test-team",
		WorkflowType:        "test",
		ProjectRoot:         "/tmp/test",
		SessionID:           "test-session",
		Status:              "running",
		BudgetMaxUSD:        10.00,
		BudgetRemainingUSD:  10.00,
		Waves: []Wave{
			{
				WaveNumber:  1,
				Description: "Wave 1",
				Members: []Member{
					{Name: "wave1-agent", Agent: "test", Model: "haiku", Status: "pending", MaxRetries: 1},
				},
			},
			{
				WaveNumber:  2,
				Description: "Wave 2",
				Members: []Member{
					{Name: "wave2-agent", Agent: "test", Model: "haiku", Status: "pending", MaxRetries: 1},
				},
			},
		},
	}

	var executionOrder []string
	var mu sync.Mutex
	runner, _ := setupTestRunner(t, config, &fakeSpawner{
		fn: func(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error {
			mu.Lock()
			tr.configMu.RLock()
			name := tr.config.Waves[waveIdx].Members[memIdx].Name
			tr.configMu.RUnlock()
			executionOrder = append(executionOrder, name)
			mu.Unlock()

			// Simulate work
			time.Sleep(50 * time.Millisecond)

			tr.configMu.Lock()
			tr.config.Waves[waveIdx].Members[memIdx].CostUSD = 0.05
			tr.configMu.Unlock()
			return nil
		},
	})

	ctx := context.Background()
	err := runWaves(ctx, runner)
	require.NoError(t, err)

	// Verify wave 1 completed before wave 2 started
	mu.Lock()
	defer mu.Unlock()
	require.Len(t, executionOrder, 2)
	assert.Equal(t, "wave1-agent", executionOrder[0])
	assert.Equal(t, "wave2-agent", executionOrder[1])
}

// TestRunWaves_ParallelMembers tests concurrent execution within a wave
func TestRunWaves_ParallelMembers(t *testing.T) {
	members := []Member{
		{Name: "agent-1", Agent: "test", Model: "haiku", Status: "pending", MaxRetries: 1},
		{Name: "agent-2", Agent: "test", Model: "haiku", Status: "pending", MaxRetries: 1},
		{Name: "agent-3", Agent: "test", Model: "haiku", Status: "pending", MaxRetries: 1},
	}

	config := testConfig(members...)
	config.BudgetMaxUSD = 5.00
	config.BudgetRemainingUSD = 5.00

	var activeCount atomic.Int32
	var maxConcurrent atomic.Int32

	runner, _ := setupTestRunner(t, config, &fakeSpawner{
		fn: func(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error {
			// Track concurrent execution
			current := activeCount.Add(1)
			for {
				max := maxConcurrent.Load()
				if current <= max || maxConcurrent.CompareAndSwap(max, current) {
					break
				}
			}

			// Simulate work
			time.Sleep(50 * time.Millisecond)

			activeCount.Add(-1)

			tr.configMu.Lock()
			tr.config.Waves[waveIdx].Members[memIdx].CostUSD = 0.05
			tr.configMu.Unlock()
			return nil
		},
	})

	ctx := context.Background()
	err := runWaves(ctx, runner)
	require.NoError(t, err)

	// Verify at least 2 members ran concurrently
	assert.GreaterOrEqual(t, maxConcurrent.Load(), int32(2), "Members should execute in parallel")
}

// TestRunWaves_BudgetExhaustion tests that budget limits prevent excessive spawning
func TestRunWaves_BudgetExhaustion(t *testing.T) {
	members := []Member{
		{Name: "agent-1", Agent: "test-agent", Model: "sonnet", Status: "pending", MaxRetries: 1},
		{Name: "agent-2", Agent: "test-agent", Model: "sonnet", Status: "pending", MaxRetries: 1},
		{Name: "agent-3", Agent: "test-agent", Model: "sonnet", Status: "pending", MaxRetries: 1},
		{Name: "agent-4", Agent: "test-agent", Model: "sonnet", Status: "pending", MaxRetries: 1},
		{Name: "agent-5", Agent: "test-agent", Model: "sonnet", Status: "pending", MaxRetries: 1},
	}

	config := testConfig(members...)
	config.BudgetMaxUSD = 2.00
	config.BudgetRemainingUSD = 2.00 // Only enough for ~1 sonnet agent (estimated $1.50)

	var spawnCount atomic.Int32
	runner, _ := setupTestRunner(t, config, &fakeSpawner{
		fn: func(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error {
			spawnCount.Add(1)
			tr.configMu.Lock()
			tr.config.Waves[waveIdx].Members[memIdx].CostUSD = 1.20
			tr.configMu.Unlock()
			return nil
		},
	})

	ctx := context.Background()
	err := runWaves(ctx, runner)
	require.NoError(t, err)

	// Should spawn only 1-2 agents before budget gate blocks
	assert.LessOrEqual(t, spawnCount.Load(), int32(2), "Budget should limit spawns")
	assert.GreaterOrEqual(t, spawnCount.Load(), int32(1), "At least one should spawn")
}

// TestRunWaves_BudgetGate tests that insufficient budget blocks member spawn
func TestRunWaves_BudgetGate(t *testing.T) {
	members := []Member{
		{Name: "agent-1", Agent: "test-agent", Model: "opus", Status: "pending", MaxRetries: 1},
	}

	config := testConfig(members...)
	config.BudgetMaxUSD = 1.00
	config.BudgetRemainingUSD = 0.10 // Insufficient for opus ($5.00 estimated)

	var spawnCount atomic.Int32
	runner, _ := setupTestRunner(t, config, &fakeSpawner{
		fn: func(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error {
			spawnCount.Add(1)
			return nil
		},
	})

	ctx := context.Background()
	err := runWaves(ctx, runner)
	require.NoError(t, err)

	// No agents should spawn due to budget gate
	assert.Equal(t, int32(0), spawnCount.Load(), "Budget gate should block all spawns")

	// Verify member status is still pending
	runner.configMu.RLock()
	defer runner.configMu.RUnlock()
	assert.Equal(t, "pending", runner.config.Waves[0].Members[0].Status)
}

// TestRunWaves_ContextCancellation tests graceful shutdown on context cancellation
func TestRunWaves_ContextCancellation(t *testing.T) {
	// Use multiple waves to ensure context is checked between waves
	config := &TeamConfig{
		TeamName:            "test-team",
		WorkflowType:        "test",
		ProjectRoot:         "/tmp/test",
		SessionID:           "test-session",
		Status:              "running",
		BudgetMaxUSD:        10.00,
		BudgetRemainingUSD:  10.00,
		Waves: []Wave{
			{
				WaveNumber:  1,
				Description: "Wave 1",
				Members: []Member{
					{Name: "agent-1", Agent: "test", Model: "haiku", Status: "pending", MaxRetries: 1},
				},
			},
			{
				WaveNumber:  2,
				Description: "Wave 2",
				Members: []Member{
					{Name: "agent-2", Agent: "test", Model: "haiku", Status: "pending", MaxRetries: 1},
				},
			},
			{
				WaveNumber:  3,
				Description: "Wave 3",
				Members: []Member{
					{Name: "agent-3", Agent: "test", Model: "haiku", Status: "pending", MaxRetries: 1},
				},
			},
		},
	}

	var spawnCount atomic.Int32
	runner, _ := setupTestRunner(t, config, &fakeSpawner{
		fn: func(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error {
			spawnCount.Add(1)
			// Simulate work
			time.Sleep(50 * time.Millisecond)
			tr.configMu.Lock()
			tr.config.Waves[waveIdx].Members[memIdx].CostUSD = 0.05
			tr.configMu.Unlock()
			return nil
		},
	})

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after first wave completes
	go func() {
		time.Sleep(80 * time.Millisecond)
		cancel()
	}()

	err := runWaves(ctx, runner)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)

	// Should have spawned wave 1, but not all waves
	spawned := spawnCount.Load()
	assert.GreaterOrEqual(t, spawned, int32(1))
	assert.Less(t, spawned, int32(3), "Should stop spawning after cancellation")
}

// TestRunWaves_InterWaveScript tests inter-wave script execution
func TestRunWaves_InterWaveScript(t *testing.T) {
	scriptPath := "test-script.sh"
	config := &TeamConfig{
		TeamName:            "test-team",
		WorkflowType:        "test",
		ProjectRoot:         "/tmp/test",
		SessionID:           "test-session",
		Status:              "running",
		BudgetMaxUSD:        10.00,
		BudgetRemainingUSD:  10.00,
		Waves: []Wave{
			{
				WaveNumber:       1,
				Description:      "Wave with script",
				OnCompleteScript: &scriptPath,
				Members: []Member{
					{Name: "agent-1", Agent: "test", Model: "haiku", Status: "pending", MaxRetries: 1},
				},
			},
		},
	}

	runner, _ := setupTestRunner(t, config, &fakeSpawner{
		fn: func(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error {
			tr.configMu.Lock()
			tr.config.Waves[waveIdx].Members[memIdx].CostUSD = 0.05
			tr.configMu.Unlock()
			return nil
		},
	})

	ctx := context.Background()
	err := runWaves(ctx, runner)
	require.NoError(t, err)

	// Inter-wave script stub should not fail (TC-010 will implement)
	runner.configMu.RLock()
	defer runner.configMu.RUnlock()
	assert.Equal(t, "completed", runner.config.Waves[0].Members[0].Status)
}

// TestRunWaves_EmptyWave tests handling of wave with no members
func TestRunWaves_EmptyWave(t *testing.T) {
	config := &TeamConfig{
		TeamName:            "test-team",
		WorkflowType:        "test",
		ProjectRoot:         "/tmp/test",
		SessionID:           "test-session",
		Status:              "running",
		BudgetMaxUSD:        10.00,
		BudgetRemainingUSD:  10.00,
		Waves: []Wave{
			{
				WaveNumber:  1,
				Description: "Empty wave",
				Members:     []Member{}, // No members
			},
			{
				WaveNumber:  2,
				Description: "Wave with member",
				Members: []Member{
					{Name: "agent-1", Agent: "test", Model: "haiku", Status: "pending", MaxRetries: 1},
				},
			},
		},
	}

	var spawnCount atomic.Int32
	runner, _ := setupTestRunner(t, config, &fakeSpawner{
		fn: func(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error {
			spawnCount.Add(1)
			tr.configMu.Lock()
			tr.config.Waves[waveIdx].Members[memIdx].CostUSD = 0.05
			tr.configMu.Unlock()
			return nil
		},
	})

	ctx := context.Background()
	err := runWaves(ctx, runner)
	require.NoError(t, err)

	// Only wave 2 member should spawn
	assert.Equal(t, int32(1), spawnCount.Load())

	// Verify wave 2 member completed
	runner.configMu.RLock()
	defer runner.configMu.RUnlock()
	assert.Equal(t, "completed", runner.config.Waves[1].Members[0].Status)
}

// TestRunWaves_BudgetReconciliation tests budget reconciliation after spawn
func TestRunWaves_BudgetReconciliation(t *testing.T) {
	members := []Member{
		{Name: "agent-1", Agent: "test-agent", Model: "sonnet", Status: "pending", MaxRetries: 1},
	}

	config := testConfig(members...)
	config.BudgetMaxUSD = 10.00
	config.BudgetRemainingUSD = 10.00

	runner, _ := setupTestRunner(t, config, &fakeSpawner{
		fn: func(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error {
			// Simulate actual cost less than estimated
			tr.configMu.Lock()
			tr.config.Waves[waveIdx].Members[memIdx].CostUSD = 0.80 // Actual < $1.50 estimated
			tr.configMu.Unlock()
			return nil
		},
	})

	initialBudget := runner.BudgetRemaining()

	ctx := context.Background()
	err := runWaves(ctx, runner)
	require.NoError(t, err)

	// Budget should reflect actual cost deducted
	finalBudget := runner.BudgetRemaining()
	actualCost := runner.config.Waves[0].Members[0].CostUSD

	expectedRemaining := initialBudget - actualCost
	assert.InDelta(t, expectedRemaining, finalBudget, 0.01, "Budget should be reconciled with actual cost")
}

// TestRunWaves_BudgetFloor tests that budget never goes negative (C1 enforcement)
func TestRunWaves_BudgetFloor(t *testing.T) {
	members := []Member{
		{Name: "agent-1", Agent: "test-agent", Model: "sonnet", Status: "pending", MaxRetries: 1},
	}

	config := testConfig(members...)
	config.BudgetMaxUSD = 2.00
	config.BudgetRemainingUSD = 2.00

	runner, _ := setupTestRunner(t, config, &fakeSpawner{
		fn: func(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error {
			// Simulate actual cost exceeding budget
			tr.configMu.Lock()
			tr.config.Waves[waveIdx].Members[memIdx].CostUSD = 3.00 // Exceeds remaining
			tr.configMu.Unlock()
			return nil
		},
	})

	ctx := context.Background()
	err := runWaves(ctx, runner)
	require.NoError(t, err)

	// Budget should be clamped to 0, never negative
	finalBudget := runner.BudgetRemaining()
	assert.GreaterOrEqual(t, finalBudget, 0.0, "Budget must never go negative (C1 enforcement)")
}

// TestSpawnAndWaitWithBudget_InvalidIndices tests graceful handling of invalid indices
func TestSpawnAndWaitWithBudget_InvalidIndices(t *testing.T) {
	t.Parallel()
	member := Member{
		Name:       "agent-1",
		Agent:      "test-agent",
		Model:      "sonnet",
		Status:     "pending",
		MaxRetries: 1,
	}

	config := testConfig(member)
	runner, _ := setupTestRunner(t, config, &fakeSpawner{
		fn: func(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error {
			return nil
		},
	})

	ctx := context.Background()

	// Call with invalid indices - should not panic, just log and return gracefully
	spawnAndWaitWithBudget(ctx, runner, 99, 0, 1.50)

	// Verify original member is unchanged
	runner.configMu.RLock()
	status := runner.config.Waves[0].Members[0].Status
	runner.configMu.RUnlock()
	assert.Equal(t, "pending", status, "Original member should be unchanged")
}
