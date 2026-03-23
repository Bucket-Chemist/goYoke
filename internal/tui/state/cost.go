// Package state provides shared, thread-safe state containers for the
// GOgent-Fortress TUI. It has no dependency on the model, cli, bridge, or any
// Bubbletea packages, keeping the import graph acyclic.
package state

import (
	"fmt"
	"sync"
)

// ---------------------------------------------------------------------------
// CostTracker
// ---------------------------------------------------------------------------

// CostTracker tracks cumulative session cost, per-agent costs, and optional
// budget enforcement for a single TUI session.
//
// The zero value is not usable; use NewCostTracker or NewCostTrackerWithBudget.
//
// Concurrency model:
//   - Write methods (UpdateSessionCost, UpdateAgentCost, SetBudget, Reset)
//     acquire a full write lock (mu.Lock).
//   - Read methods (GetSessionCost, GetAgentCosts, CheckBudget) acquire a
//     shared read lock (mu.RLock).
type CostTracker struct {
	// SessionCost is the cumulative session cost from the result event.
	// This is NOT per-message — it's the total from total_cost_usd.
	SessionCost float64

	// PerAgentCosts tracks cost per agent ID.
	PerAgentCosts map[string]float64

	// BudgetUSD is the session budget. nil means no budget set.
	BudgetUSD *float64

	// OverBudget is true when SessionCost exceeds BudgetUSD.
	OverBudget bool

	mu sync.RWMutex
}

// NewCostTracker allocates and returns a CostTracker with no budget set.
func NewCostTracker() *CostTracker {
	return &CostTracker{
		PerAgentCosts: make(map[string]float64),
	}
}

// NewCostTrackerWithBudget allocates and returns a CostTracker with the given
// USD budget pre-configured.
func NewCostTrackerWithBudget(budgetUSD float64) *CostTracker {
	ct := NewCostTracker()
	ct.BudgetUSD = &budgetUSD
	return ct
}

// ---------------------------------------------------------------------------
// Write methods
// ---------------------------------------------------------------------------

// UpdateSessionCost sets the cumulative session cost from a result event.
//
// NOTE: total_cost_usd from the CLI result event is CUMULATIVE — it replaces
// the previous value rather than adding to it. OverBudget is recomputed after
// every update.
func (ct *CostTracker) UpdateSessionCost(totalCostUSD float64) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.SessionCost = totalCostUSD
	ct.recomputeOverBudget()
}

// UpdateAgentCost increments the tracked cost for the agent identified by
// agentID. Each call adds cost to the running total for that agent.
func (ct *CostTracker) UpdateAgentCost(agentID string, cost float64) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.PerAgentCosts[agentID] += cost
}

// SetBudget sets or replaces the session budget. OverBudget is recomputed
// against the current SessionCost immediately.
func (ct *CostTracker) SetBudget(budgetUSD float64) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.BudgetUSD = &budgetUSD
	ct.recomputeOverBudget()
}

// Reset clears all cost data. After Reset, SessionCost is zero, PerAgentCosts
// is empty, OverBudget is false, and BudgetUSD is unchanged.
func (ct *CostTracker) Reset() {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.SessionCost = 0
	ct.PerAgentCosts = make(map[string]float64)
	ct.OverBudget = false
}

// recomputeOverBudget recalculates OverBudget based on the current
// SessionCost and BudgetUSD. Must be called with mu held for writing.
func (ct *CostTracker) recomputeOverBudget() {
	if ct.BudgetUSD == nil {
		ct.OverBudget = false
		return
	}
	ct.OverBudget = ct.SessionCost > *ct.BudgetUSD
}

// ---------------------------------------------------------------------------
// Read methods
// ---------------------------------------------------------------------------

// CheckBudget returns the remaining budget and whether the session is over
// budget.
//
// If no budget is set, remaining is -1 and overBudget is false.
// If a budget is set, remaining is (BudgetUSD - SessionCost); it may be
// negative when the session is over budget.
func (ct *CostTracker) CheckBudget() (remaining float64, overBudget bool) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	if ct.BudgetUSD == nil {
		return -1, false
	}
	return *ct.BudgetUSD - ct.SessionCost, ct.OverBudget
}

// GetSessionCost returns the current cumulative session cost.
func (ct *CostTracker) GetSessionCost() float64 {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	return ct.SessionCost
}

// GetAgentCosts returns a copy of the per-agent cost map. Callers may safely
// read the returned map without coordination; mutations do not affect the
// tracker's internal state.
func (ct *CostTracker) GetAgentCosts() map[string]float64 {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	cp := make(map[string]float64, len(ct.PerAgentCosts))
	for k, v := range ct.PerAgentCosts {
		cp[k] = v
	}
	return cp
}

// ---------------------------------------------------------------------------
// FormatCost
// ---------------------------------------------------------------------------

// FormatCost formats a cost value as a USD string for display.
//
// Formatting rules:
//   - cost == 0          → "$0.00"
//   - cost >= $0.01      → "$X.XX"   (2 decimal places)
//   - 0 < cost < $0.01  → "$X.XXXX" (4 decimal places)
//   - cost < 0           → "-$X.XX"  (negative costs for refunds/adjustments)
func FormatCost(cost float64) string {
	if cost == 0 {
		return "$0.00"
	}

	if cost < 0 {
		// Format the absolute value with 2 decimal places, then prepend "-$".
		abs := -cost
		if abs < 0.01 {
			return fmt.Sprintf("-$%.4f", abs)
		}
		return fmt.Sprintf("-$%.2f", abs)
	}

	// Positive cost.
	if cost < 0.01 {
		return fmt.Sprintf("$%.4f", cost)
	}
	return fmt.Sprintf("$%.2f", cost)
}
