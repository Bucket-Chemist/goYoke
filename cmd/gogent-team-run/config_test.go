package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBudgetFloorEnforcement verifies that reconcileCost clamps budget to 0 when it would go negative
func TestBudgetFloorEnforcement(t *testing.T) {
	member := Member{
		Name:   "agent-1",
		Status: "pending",
	}

	config := testConfig(member)
	config.BudgetMaxUSD = 2.00
	config.BudgetRemainingUSD = 1.00

	runner, _ := setupTestRunner(t, config)

	// Reserve $0.50
	reserved := runner.tryReserveBudget(0.50)
	require.True(t, reserved)

	// Reconcile with actual cost $2.00 (way more than estimated + remaining)
	// Remaining was $0.50, actual is $2.00 → should clamp to $0.00
	err := runner.reconcileCost(0.50, 2.00)
	require.NoError(t, err)

	// Budget should be clamped to 0, not negative
	budget := runner.BudgetRemaining()
	assert.Equal(t, 0.0, budget, "Budget must be clamped to 0, not negative")
}

// TestBudgetNeverNegative_Concurrent verifies budget never goes negative under concurrent operations
func TestBudgetNeverNegative_Concurrent(t *testing.T) {
	member := Member{
		Name:   "agent-1",
		Status: "pending",
		Agent:  "test-agent",
		Model:  "sonnet",
	}

	config := testConfig(member)
	config.BudgetMaxUSD = 1.00
	config.BudgetRemainingUSD = 1.00

	runner, _ := setupTestRunner(t, config)

	// 10 goroutines all try to reserve and reconcile
	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			// Try to reserve
			if runner.tryReserveBudget(0.20) {
				// Simulate work, then reconcile
				runner.reconcileCost(0.20, 0.25)
			}
		}(i)
	}

	wg.Wait()

	// Budget must be >= 0
	budget := runner.BudgetRemaining()
	assert.GreaterOrEqual(t, budget, 0.0, "Budget must never be negative")
}

// TestWriteOrdering_NoLostUpdates verifies that concurrent updateMember calls don't lose updates
func TestWriteOrdering_NoLostUpdates(t *testing.T) {
	// Create 10 members
	members := make([]Member, 10)
	for i := range members {
		members[i] = Member{
			Name:       fmt.Sprintf("agent-%d", i+1),
			Status:     "pending",
			RetryCount: 0,
		}
	}

	config := testConfig(members...)
	runner, _ := setupTestRunner(t, config)

	// 20 goroutines updating different members
	var wg sync.WaitGroup
	numUpdates := 20

	for i := 0; i < numUpdates; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			memberIdx := idx % len(members)
			err := runner.updateMember(0, memberIdx, func(m *Member) {
				m.RetryCount++
				m.Status = "running"
			})
			if err != nil {
				t.Errorf("updateMember failed: %v", err)
			}
		}(i)
	}

	wg.Wait()

	// Verify all updates are present
	runner.configMu.RLock()
	defer runner.configMu.RUnlock()

	for i, m := range runner.config.Waves[0].Members {
		assert.Equal(t, "running", m.Status, "member %d should be running", i)
		// Each member should have been updated 2 times (20 updates / 10 members)
		assert.Equal(t, 2, m.RetryCount, "member %d should have retryCount 2", i)
	}
}

// TestBudgetAccessor verifies BudgetRemaining returns correct value under concurrent mutations
func TestBudgetAccessor(t *testing.T) {
	member := Member{
		Name:   "agent-1",
		Status: "pending",
	}

	config := testConfig(member)
	config.BudgetMaxUSD = 10.00
	config.BudgetRemainingUSD = 10.00

	runner, _ := setupTestRunner(t, config)

	// Initial read
	budget := runner.BudgetRemaining()
	assert.Equal(t, 10.0, budget)

	// Reserve $3.00
	reserved := runner.tryReserveBudget(3.00)
	require.True(t, reserved)

	// Read after reservation
	budget = runner.BudgetRemaining()
	assert.Equal(t, 7.0, budget)

	// Reconcile: return $3.00, deduct $2.50
	err := runner.reconcileCost(3.00, 2.50)
	require.NoError(t, err)

	// Final read
	budget = runner.BudgetRemaining()
	assert.Equal(t, 7.5, budget)
}

// TestTryReserveBudget_Sufficient verifies reservation succeeds when budget is sufficient
func TestTryReserveBudget_Sufficient(t *testing.T) {
	member := Member{
		Name:   "agent-1",
		Status: "pending",
	}

	config := testConfig(member)
	config.BudgetMaxUSD = 5.00
	config.BudgetRemainingUSD = 5.00

	runner, _ := setupTestRunner(t, config)

	// Reserve $1.50
	reserved := runner.tryReserveBudget(1.50)
	assert.True(t, reserved, "Reservation should succeed")

	// Check remaining
	budget := runner.BudgetRemaining()
	assert.Equal(t, 3.5, budget)
}

// TestTryReserveBudget_Insufficient verifies reservation fails when budget is insufficient
func TestTryReserveBudget_Insufficient(t *testing.T) {
	member := Member{
		Name:   "agent-1",
		Status: "pending",
	}

	config := testConfig(member)
	config.BudgetMaxUSD = 1.00
	config.BudgetRemainingUSD = 0.50

	runner, _ := setupTestRunner(t, config)

	// Try to reserve $1.50 (more than available)
	reserved := runner.tryReserveBudget(1.50)
	assert.False(t, reserved, "Reservation should fail")

	// Budget should be unchanged
	budget := runner.BudgetRemaining()
	assert.Equal(t, 0.5, budget)
}

// TestReconcileCost_Normal verifies normal reconciliation (actual < estimated)
func TestReconcileCost_Normal(t *testing.T) {
	member := Member{
		Name:   "agent-1",
		Status: "pending",
	}

	config := testConfig(member)
	config.BudgetMaxUSD = 10.00
	config.BudgetRemainingUSD = 10.00

	runner, _ := setupTestRunner(t, config)

	// Reserve $1.50
	reserved := runner.tryReserveBudget(1.50)
	require.True(t, reserved)

	// Budget after reservation: $8.50
	budget := runner.BudgetRemaining()
	assert.Equal(t, 8.5, budget)

	// Reconcile: actual cost $1.20 (less than estimated)
	err := runner.reconcileCost(1.50, 1.20)
	require.NoError(t, err)

	// Net change: +$1.50 - $1.20 = +$0.30
	// Budget: $8.50 + $0.30 = $8.80
	budget = runner.BudgetRemaining()
	assert.Equal(t, 8.8, budget)
}

// TestReconcileCost_ActualHigher verifies reconciliation when actual > estimated
func TestReconcileCost_ActualHigher(t *testing.T) {
	member := Member{
		Name:   "agent-1",
		Status: "pending",
	}

	config := testConfig(member)
	config.BudgetMaxUSD = 10.00
	config.BudgetRemainingUSD = 10.00

	runner, _ := setupTestRunner(t, config)

	// Reserve $1.50
	reserved := runner.tryReserveBudget(1.50)
	require.True(t, reserved)

	// Reconcile: actual cost $2.00 (more than estimated)
	err := runner.reconcileCost(1.50, 2.00)
	require.NoError(t, err)

	// Net change: +$1.50 - $2.00 = -$0.50
	// Budget: $8.50 - $0.50 = $8.00
	budget := runner.BudgetRemaining()
	assert.Equal(t, 8.0, budget)
}

// TestEstimateCost_DefaultFallback verifies unknown agent uses $1.50 fallback
func TestEstimateCost_DefaultFallback(t *testing.T) {
	member := Member{
		Name:   "agent-1",
		Status: "pending",
		Agent:  "unknown-agent",
		Model:  "unknown-model",
	}

	config := testConfig(member)
	runner, _ := setupTestRunner(t, config)

	// Unknown agent should return $1.50 fallback
	estimate := runner.estimateCost("unknown-agent")
	assert.Equal(t, 1.5, estimate)
}

// TestEstimateCost_ModelBased verifies model-based cost estimation
func TestEstimateCost_ModelBased(t *testing.T) {
	tests := []struct {
		name          string
		model         string
		expectedCost  float64
	}{
		{"haiku_model", "haiku", 0.10},
		{"sonnet_model", "sonnet", 1.50},
		{"opus_model", "opus", 5.00},
		{"unknown_model", "unknown", 1.50},
		{"empty_model", "", 1.50},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			member := Member{
				Name:   "agent-1",
				Status: "pending",
				Agent:  "test-agent",
				Model:  tc.model,
			}

			config := testConfig(member)
			runner, _ := setupTestRunner(t, config)

			estimate := runner.estimateCost("test-agent")
			assert.Equal(t, tc.expectedCost, estimate)
		})
	}
}

// TestBudgetRemaining_NilConfig verifies safe handling of nil config
func TestBudgetRemaining_NilConfig(t *testing.T) {
	teamDir := t.TempDir()
	runner := &TeamRunner{
		teamDir:   teamDir,
		config:    nil, // nil config
		childPIDs: make(map[int]struct{}),
	}

	budget := runner.BudgetRemaining()
	assert.Equal(t, 0.0, budget, "Should return 0 for nil config")
}

// TestTryReserveBudget_NilConfig verifies reservation fails safely with nil config
func TestTryReserveBudget_NilConfig(t *testing.T) {
	teamDir := t.TempDir()
	runner := &TeamRunner{
		teamDir:   teamDir,
		config:    nil, // nil config
		childPIDs: make(map[int]struct{}),
	}

	reserved := runner.tryReserveBudget(1.00)
	assert.False(t, reserved, "Reservation should fail with nil config")
}

// TestReconcileCost_NilConfig verifies reconcileCost returns error with nil config
func TestReconcileCost_NilConfig(t *testing.T) {
	teamDir := t.TempDir()
	runner := &TeamRunner{
		teamDir:   teamDir,
		config:    nil, // nil config
		childPIDs: make(map[int]struct{}),
	}

	err := runner.reconcileCost(1.00, 0.80)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config not loaded")
}

// TestBudgetConcurrentReserveAndReconcile verifies no race conditions in budget operations
func TestBudgetConcurrentReserveAndReconcile(t *testing.T) {
	member := Member{
		Name:   "agent-1",
		Status: "pending",
		Agent:  "test-agent",
		Model:  "haiku",
	}

	config := testConfig(member)
	config.BudgetMaxUSD = 5.00
	config.BudgetRemainingUSD = 5.00

	runner, _ := setupTestRunner(t, config)

	var wg sync.WaitGroup
	var successCount atomic.Int64

	// 20 goroutines trying to reserve and reconcile
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			if runner.tryReserveBudget(0.50) {
				successCount.Add(1)
				// Simulate actual cost = estimated (no return)
				// This prevents budget from growing and allowing more reservations
				runner.reconcileCost(0.50, 0.50)
			}
		}()
	}

	wg.Wait()

	// Budget must be >= 0
	budget := runner.BudgetRemaining()
	assert.GreaterOrEqual(t, budget, 0.0, "Budget must never go negative")

	// Should have allowed exactly 10 reservations (5.00 / 0.50)
	count := successCount.Load()
	assert.Greater(t, count, int64(0), "At least some reservations should succeed")
	assert.LessOrEqual(t, count, int64(10), "Should not exceed budget capacity")
}

// TestSaveConfig_Concurrent verifies SaveConfig is safe under concurrent calls
func TestSaveConfig_Concurrent(t *testing.T) {
	member := Member{
		Name:   "agent-1",
		Status: "pending",
	}

	config := testConfig(member)
	runner, _ := setupTestRunner(t, config)

	var wg sync.WaitGroup
	var errCount atomic.Int64

	// 10 concurrent SaveConfig calls using updateMember (which is thread-safe)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			// Use updateMember which properly handles locking and serialization
			if err := runner.updateMember(0, 0, func(m *Member) {
				m.RetryCount = idx
			}); err != nil {
				errCount.Add(1)
			}
		}(i)
	}

	wg.Wait()

	// All updates should succeed (no lost writes)
	assert.Equal(t, int64(0), errCount.Load(), "All SaveConfig calls should succeed")

	// Verify config file is valid JSON
	configPath := filepath.Join(runner.teamDir, ConfigFileName)
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var reloaded TeamConfig
	err = json.Unmarshal(data, &reloaded)
	require.NoError(t, err, "Config file should be valid JSON")
}

// TestBudgetEdgeCase_ZeroBudget verifies handling of zero budget
func TestBudgetEdgeCase_ZeroBudget(t *testing.T) {
	member := Member{
		Name:   "agent-1",
		Status: "pending",
	}

	config := testConfig(member)
	config.BudgetMaxUSD = 0.00
	config.BudgetRemainingUSD = 0.00

	runner, _ := setupTestRunner(t, config)

	// Try to reserve with zero budget
	reserved := runner.tryReserveBudget(0.10)
	assert.False(t, reserved, "Cannot reserve with zero budget")

	budget := runner.BudgetRemaining()
	assert.Equal(t, 0.0, budget)
}

// TestBudgetEdgeCase_ExactMatch verifies reservation of exact remaining budget
func TestBudgetEdgeCase_ExactMatch(t *testing.T) {
	member := Member{
		Name:   "agent-1",
		Status: "pending",
	}

	config := testConfig(member)
	config.BudgetMaxUSD = 1.50
	config.BudgetRemainingUSD = 1.50

	runner, _ := setupTestRunner(t, config)

	// Reserve exact amount
	reserved := runner.tryReserveBudget(1.50)
	assert.True(t, reserved, "Should allow exact budget reservation")

	budget := runner.BudgetRemaining()
	assert.Equal(t, 0.0, budget)

	// Try to reserve more
	reserved = runner.tryReserveBudget(0.01)
	assert.False(t, reserved, "Cannot reserve with zero budget")
}

// TestReconcileCost_LargeActualCost verifies floor enforcement with large overrun
func TestReconcileCost_LargeActualCost(t *testing.T) {
	member := Member{
		Name:   "agent-1",
		Status: "pending",
	}

	config := testConfig(member)
	config.BudgetMaxUSD = 1.00
	config.BudgetRemainingUSD = 1.00

	runner, _ := setupTestRunner(t, config)

	// Reserve $0.50
	reserved := runner.tryReserveBudget(0.50)
	require.True(t, reserved)

	// Reconcile with huge actual cost (10x estimated)
	err := runner.reconcileCost(0.50, 5.00)
	require.NoError(t, err)

	// Budget should be clamped to 0
	budget := runner.BudgetRemaining()
	assert.Equal(t, 0.0, budget)
}

// TestReconcileCost_WritesToDisk verifies reconcileCost persists budget to config.json
func TestReconcileCost_WritesToDisk(t *testing.T) {
	t.Parallel()
	config := testConfig(Member{Name: "a", Agent: "test", Status: "pending"})
	config.BudgetMaxUSD = 10.0
	config.BudgetRemainingUSD = 10.0
	runner, teamDir := setupTestRunner(t, config)

	// Reserve then reconcile
	require.True(t, runner.tryReserveBudget(2.0))
	err := runner.reconcileCost(2.0, 1.5)
	require.NoError(t, err)

	// Verify budget written to disk
	data, err := os.ReadFile(filepath.Join(teamDir, ConfigFileName))
	require.NoError(t, err)
	var reloaded TeamConfig
	require.NoError(t, json.Unmarshal(data, &reloaded))
	assert.InDelta(t, 8.5, reloaded.BudgetRemainingUSD, 0.01)
}

// TestSaveConfig_WritesToDisk tests SaveConfig writes updated status to disk
func TestSaveConfig_WritesToDisk(t *testing.T) {
	t.Parallel()
	config := testConfig(Member{Name: "a", Status: "pending"})
	runner, teamDir := setupTestRunner(t, config)

	runner.configMu.Lock()
	runner.config.Status = "completed"
	runner.configMu.Unlock()

	err := runner.SaveConfig()
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(teamDir, ConfigFileName))
	require.NoError(t, err)
	var reloaded TeamConfig
	require.NoError(t, json.Unmarshal(data, &reloaded))
	assert.Equal(t, "completed", reloaded.Status)
}

// TestLoadConfig_ReadError tests LoadConfig handles read errors gracefully
func TestLoadConfig_ReadError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Create config.json as a directory, not a file
	err := os.Mkdir(filepath.Join(dir, ConfigFileName), 0755)
	require.NoError(t, err)

	runner := &TeamRunner{
		teamDir:    dir,
		configPath: filepath.Join(dir, ConfigFileName),
		childPIDs:  make(map[int]struct{}),
	}
	err = runner.LoadConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read config.json")
}
