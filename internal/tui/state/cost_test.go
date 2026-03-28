package state

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func budgetPtr(v float64) *float64 { return &v }

// ---------------------------------------------------------------------------
// NewCostTracker
// ---------------------------------------------------------------------------

func TestNewCostTracker_Defaults(t *testing.T) {
	ct := NewCostTracker()
	require.NotNil(t, ct)
	assert.Equal(t, 0.0, ct.GetSessionCost())
	assert.Nil(t, ct.GetBudgetUSD())
	assert.False(t, ct.IsOverBudget())
	assert.Empty(t, ct.GetAgentCosts())
}

// ---------------------------------------------------------------------------
// NewCostTrackerWithBudget
// ---------------------------------------------------------------------------

func TestNewCostTrackerWithBudget(t *testing.T) {
	ct := NewCostTrackerWithBudget(5.00)
	require.NotNil(t, ct)
	require.NotNil(t, ct.GetBudgetUSD())
	assert.Equal(t, 5.00, *ct.GetBudgetUSD())
	assert.Equal(t, 0.0, ct.GetSessionCost())
	assert.False(t, ct.IsOverBudget())
}

// ---------------------------------------------------------------------------
// UpdateSessionCost
// ---------------------------------------------------------------------------

// UpdateSessionCost replaces (does not accumulate) the session cost.
func TestUpdateSessionCost_Cumulative(t *testing.T) {
	ct := NewCostTracker()
	ct.UpdateSessionCost(1.00)
	assert.Equal(t, 1.00, ct.GetSessionCost())

	// Second call with a new cumulative total — must replace, not add.
	ct.UpdateSessionCost(2.50)
	assert.Equal(t, 2.50, ct.GetSessionCost())

	// Third call with a lower value (e.g., session reset on provider side).
	ct.UpdateSessionCost(0.10)
	assert.Equal(t, 0.10, ct.GetSessionCost())
}

// ---------------------------------------------------------------------------
// UpdateAgentCost
// ---------------------------------------------------------------------------

func TestUpdateAgentCost_Increments(t *testing.T) {
	ct := NewCostTracker()
	ct.UpdateAgentCost("agent-1", 0.05)
	ct.UpdateAgentCost("agent-1", 0.10)
	ct.UpdateAgentCost("agent-1", 0.03)

	costs := ct.GetAgentCosts()
	assert.InDelta(t, 0.18, costs["agent-1"], 1e-9)
}

func TestUpdateAgentCost_MultipleAgents(t *testing.T) {
	ct := NewCostTracker()
	ct.UpdateAgentCost("agent-A", 1.00)
	ct.UpdateAgentCost("agent-B", 0.50)
	ct.UpdateAgentCost("agent-A", 0.25)

	costs := ct.GetAgentCosts()
	assert.InDelta(t, 1.25, costs["agent-A"], 1e-9)
	assert.InDelta(t, 0.50, costs["agent-B"], 1e-9)
}

// ---------------------------------------------------------------------------
// CheckBudget
// ---------------------------------------------------------------------------

func TestCheckBudget_NoBudget(t *testing.T) {
	ct := NewCostTracker()
	ct.UpdateSessionCost(99.00)

	remaining, overBudget := ct.CheckBudget()
	assert.Equal(t, -1.0, remaining)
	assert.False(t, overBudget)
}

func TestCheckBudget_UnderBudget(t *testing.T) {
	ct := NewCostTrackerWithBudget(10.00)
	ct.UpdateSessionCost(3.75)

	remaining, overBudget := ct.CheckBudget()
	assert.InDelta(t, 6.25, remaining, 1e-9)
	assert.False(t, overBudget)
}

func TestCheckBudget_OverBudget(t *testing.T) {
	ct := NewCostTrackerWithBudget(5.00)
	ct.UpdateSessionCost(7.50)

	remaining, overBudget := ct.CheckBudget()
	assert.InDelta(t, -2.50, remaining, 1e-9)
	assert.True(t, overBudget)
}

// At exactly the budget boundary the session is NOT over budget.
func TestCheckBudget_ExactlyAtBudget(t *testing.T) {
	ct := NewCostTrackerWithBudget(5.00)
	ct.UpdateSessionCost(5.00)

	remaining, overBudget := ct.CheckBudget()
	assert.InDelta(t, 0.0, remaining, 1e-9)
	assert.False(t, overBudget)
}

// ---------------------------------------------------------------------------
// FormatCost
// ---------------------------------------------------------------------------

func TestFormatCost_Table(t *testing.T) {
	tests := []struct {
		name  string
		cost  float64
		want  string
	}{
		{"zero", 0.0, "$0.00"},
		{"small positive (4 dp)", 0.0042, "$0.0042"},
		{"exactly one cent boundary (4 dp)", 0.0099, "$0.0099"},
		{"at one cent (2 dp)", 0.01, "$0.01"},
		{"normal", 1.23, "$1.23"},
		{"large", 100.00, "$100.00"},
		{"negative large", -2.50, "-$2.50"},
		{"negative small (4 dp)", -0.0042, "-$0.0042"},
		{"negative zero", -0.0, "$0.00"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, FormatCost(tc.cost))
		})
	}
}

// ---------------------------------------------------------------------------
// GetAgentCosts — returns copy
// ---------------------------------------------------------------------------

func TestGetAgentCosts_ReturnsCopy(t *testing.T) {
	ct := NewCostTracker()
	ct.UpdateAgentCost("agent-X", 1.00)

	// Mutate the returned map.
	costs := ct.GetAgentCosts()
	costs["agent-X"] = 999.00
	costs["injected"] = 42.00

	// The tracker's internal state must be unaffected.
	internal := ct.GetAgentCosts()
	assert.InDelta(t, 1.00, internal["agent-X"], 1e-9)
	assert.NotContains(t, internal, "injected")
}

// ---------------------------------------------------------------------------
// SetBudget
// ---------------------------------------------------------------------------

func TestSetBudget_Updates(t *testing.T) {
	ct := NewCostTracker()
	assert.Nil(t, ct.GetBudgetUSD())

	ct.SetBudget(10.00)
	require.NotNil(t, ct.GetBudgetUSD())
	assert.Equal(t, 10.00, *ct.GetBudgetUSD())

	// Update to a lower budget that is now exceeded by the existing cost.
	ct.UpdateSessionCost(8.00)
	ct.SetBudget(5.00)
	assert.True(t, ct.IsOverBudget())

	// Raise the budget back above cost — OverBudget must clear.
	ct.SetBudget(20.00)
	assert.False(t, ct.IsOverBudget())
}

// ---------------------------------------------------------------------------
// Reset
// ---------------------------------------------------------------------------

func TestReset_ClearsAll(t *testing.T) {
	ct := NewCostTrackerWithBudget(10.00)
	ct.UpdateSessionCost(15.00) // sets overBudget = true
	ct.UpdateAgentCost("agent-A", 5.00)
	ct.UpdateAgentCost("agent-B", 3.00)

	require.True(t, ct.IsOverBudget(), "pre-condition: should be over budget")

	ct.Reset()

	assert.Equal(t, 0.0, ct.GetSessionCost())
	assert.Empty(t, ct.GetAgentCosts())
	assert.False(t, ct.IsOverBudget())

	// Budget itself must be preserved after Reset.
	require.NotNil(t, ct.GetBudgetUSD())
	assert.Equal(t, 10.00, *ct.GetBudgetUSD())
}

// ---------------------------------------------------------------------------
// Concurrent access
// ---------------------------------------------------------------------------

// TestConcurrentAccess verifies that multiple goroutines can call Update/Get
// methods simultaneously without data races. Run with -race.
func TestConcurrentAccess(t *testing.T) {
	ct := NewCostTrackerWithBudget(100.00)

	const goroutines = 50
	var wg sync.WaitGroup

	// Writers: UpdateSessionCost.
	for i := range goroutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ct.UpdateSessionCost(float64(i) * 0.01)
		}(i)
	}

	// Writers: UpdateAgentCost.
	for i := range goroutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			agentID := "agent-concurrent"
			ct.UpdateAgentCost(agentID, float64(i)*0.001)
		}(i)
	}

	// Readers: GetSessionCost.
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = ct.GetSessionCost()
		}()
	}

	// Readers: GetAgentCosts.
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = ct.GetAgentCosts()
		}()
	}

	// Readers: CheckBudget.
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = ct.CheckBudget()
		}()
	}

	// Writers: SetBudget.
	for i := range goroutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ct.SetBudget(float64(i+1) * 10.0)
		}(i)
	}

	wg.Wait()

	// Post-condition: no panic, no race. State values are nondeterministic
	// due to concurrent writes, but the tracker must still be structurally
	// sound (no nil map, no inconsistent OverBudget when budget is nil).
	remaining, overBudget := ct.CheckBudget()
	if ct.GetBudgetUSD() == nil {
		assert.Equal(t, -1.0, remaining)
		assert.False(t, overBudget)
	}
}
