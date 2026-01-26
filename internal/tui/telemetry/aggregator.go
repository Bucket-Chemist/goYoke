package telemetry

import (
	"fmt"
	"sync"
	"time"

	pkgtel "github.com/Bucket-Chemist/GOgent-Fortress/pkg/telemetry"
)

// AgentState represents the lifecycle state of a single agent
type AgentState struct {
	AgentID       string
	ParentAgent   string
	Tier          string
	TaskDesc      string
	SpawnEvent    *pkgtel.AgentLifecycleEvent
	CompleteEvent *pkgtel.AgentLifecycleEvent
	Status        string // "running", "completed", "error"
	Duration      time.Duration
	DecisionID    string
}

// TelemetryAggregator maintains high-level state by processing raw telemetry events.
// Correlates agent spawn/complete events, tracks active agents, calculates durations.
type TelemetryAggregator struct {
	watcher *TelemetryWatcher

	// State maps (thread-safe)
	mu               sync.RWMutex
	agents           map[string]*AgentState            // agentID -> state
	decisions        map[string]*pkgtel.RoutingDecision // decisionID -> decision
	decisionOutcomes map[string]*pkgtel.DecisionOutcomeUpdate

	// Derived state
	activeAgents    []*AgentState
	completedAgents []*AgentState

	// Control
	done chan struct{}
	wg   sync.WaitGroup
}

// NewTelemetryAggregator creates aggregator with embedded watcher
func NewTelemetryAggregator() (*TelemetryAggregator, error) {
	watcher, err := NewTelemetryWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create telemetry watcher: %w", err)
	}

	ta := &TelemetryAggregator{
		watcher:          watcher,
		agents:           make(map[string]*AgentState),
		decisions:        make(map[string]*pkgtel.RoutingDecision),
		decisionOutcomes: make(map[string]*pkgtel.DecisionOutcomeUpdate),
		activeAgents:     make([]*AgentState, 0),
		completedAgents:  make([]*AgentState, 0),
		done:             make(chan struct{}),
	}

	return ta, nil
}

// Start begins processing telemetry events
func (ta *TelemetryAggregator) Start() error {
	if err := ta.watcher.Start(); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}

	// Start processing goroutines
	ta.wg.Add(3)
	go ta.processLifecycleEvents()
	go ta.processDecisionEvents()
	go ta.processUpdateEvents()

	return nil
}

// processLifecycleEvents handles agent spawn/complete events
func (ta *TelemetryAggregator) processLifecycleEvents() {
	defer ta.wg.Done()
	events := ta.watcher.Events()

	for {
		select {
		case event := <-events.AgentLifecycle:
			if event == nil {
				return
			}
			ta.handleLifecycleEvent(event)
		case <-ta.done:
			return
		}
	}
}

// handleLifecycleEvent processes a single lifecycle event
func (ta *TelemetryAggregator) handleLifecycleEvent(event *pkgtel.AgentLifecycleEvent) {
	ta.mu.Lock()
	defer ta.mu.Unlock()

	// Get or create agent state
	state, exists := ta.agents[event.AgentID]
	if !exists {
		state = &AgentState{
			AgentID:     event.AgentID,
			ParentAgent: event.ParentAgent,
			Tier:        event.Tier,
			TaskDesc:    event.TaskDescription,
			DecisionID:  event.DecisionID,
			Status:      "running",
		}
		ta.agents[event.AgentID] = state
	}

	// Update state based on event type
	switch event.EventType {
	case "spawn":
		state.SpawnEvent = event
		state.Status = "running"
		ta.rebuildActiveLists()

	case "complete":
		state.CompleteEvent = event
		if event.Success != nil && *event.Success {
			state.Status = "completed"
		} else {
			state.Status = "error"
		}

		// Calculate duration
		if state.SpawnEvent != nil {
			duration := event.Timestamp - state.SpawnEvent.Timestamp
			state.Duration = time.Duration(duration) * time.Second
		}
		ta.rebuildActiveLists()

	case "error":
		state.CompleteEvent = event
		state.Status = "error"
		ta.rebuildActiveLists()
	}
}

// processDecisionEvents handles routing decision events
func (ta *TelemetryAggregator) processDecisionEvents() {
	defer ta.wg.Done()
	events := ta.watcher.Events()

	for {
		select {
		case decision := <-events.RoutingDecisions:
			if decision == nil {
				return
			}
			ta.handleDecision(decision)
		case <-ta.done:
			return
		}
	}
}

// handleDecision stores routing decision
func (ta *TelemetryAggregator) handleDecision(decision *pkgtel.RoutingDecision) {
	ta.mu.Lock()
	defer ta.mu.Unlock()

	ta.decisions[decision.DecisionID] = decision
}

// processUpdateEvents handles decision outcome updates
func (ta *TelemetryAggregator) processUpdateEvents() {
	defer ta.wg.Done()
	events := ta.watcher.Events()

	for {
		select {
		case update := <-events.DecisionUpdates:
			if update == nil {
				return
			}
			ta.handleUpdate(update)
		case <-ta.done:
			return
		}
	}
}

// handleUpdate stores decision outcome update
func (ta *TelemetryAggregator) handleUpdate(update *pkgtel.DecisionOutcomeUpdate) {
	ta.mu.Lock()
	defer ta.mu.Unlock()

	// Keep latest update per decision ID
	existing, exists := ta.decisionOutcomes[update.DecisionID]
	if !exists || update.UpdateTimestamp > existing.UpdateTimestamp {
		ta.decisionOutcomes[update.DecisionID] = update
	}
}

// rebuildActiveLists rebuilds active/completed agent lists (caller must hold lock)
func (ta *TelemetryAggregator) rebuildActiveLists() {
	ta.activeAgents = make([]*AgentState, 0)
	ta.completedAgents = make([]*AgentState, 0)

	for _, state := range ta.agents {
		if state.Status == "running" {
			ta.activeAgents = append(ta.activeAgents, state)
		} else {
			ta.completedAgents = append(ta.completedAgents, state)
		}
	}
}

// GetActiveAgents returns currently running agents (thread-safe)
func (ta *TelemetryAggregator) GetActiveAgents() []*AgentState {
	ta.mu.RLock()
	defer ta.mu.RUnlock()

	result := make([]*AgentState, len(ta.activeAgents))
	copy(result, ta.activeAgents)
	return result
}

// GetCompletedAgents returns completed/error agents (thread-safe)
func (ta *TelemetryAggregator) GetCompletedAgents() []*AgentState {
	ta.mu.RLock()
	defer ta.mu.RUnlock()

	result := make([]*AgentState, len(ta.completedAgents))
	copy(result, ta.completedAgents)
	return result
}

// GetAgentState returns state for specific agent (thread-safe)
func (ta *TelemetryAggregator) GetAgentState(agentID string) (*AgentState, bool) {
	ta.mu.RLock()
	defer ta.mu.RUnlock()

	state, exists := ta.agents[agentID]
	if !exists {
		return nil, false
	}

	// Return copy to avoid race conditions
	stateCopy := *state
	return &stateCopy, true
}

// GetDecision returns routing decision by ID (thread-safe)
func (ta *TelemetryAggregator) GetDecision(decisionID string) (*pkgtel.RoutingDecision, bool) {
	ta.mu.RLock()
	defer ta.mu.RUnlock()

	decision, exists := ta.decisions[decisionID]
	return decision, exists
}

// GetDecisionOutcome returns latest outcome for decision (thread-safe)
func (ta *TelemetryAggregator) GetDecisionOutcome(decisionID string) (*pkgtel.DecisionOutcomeUpdate, bool) {
	ta.mu.RLock()
	defer ta.mu.RUnlock()

	outcome, exists := ta.decisionOutcomes[decisionID]
	return outcome, exists
}

// Stats returns aggregated statistics
func (ta *TelemetryAggregator) Stats() AggregatorStats {
	ta.mu.RLock()
	defer ta.mu.RUnlock()

	stats := AggregatorStats{
		TotalAgents:     len(ta.agents),
		ActiveAgents:    len(ta.activeAgents),
		CompletedAgents: len(ta.completedAgents),
		TotalDecisions:  len(ta.decisions),
	}

	// Calculate success rate
	var successCount int
	for _, state := range ta.completedAgents {
		if state.Status == "completed" {
			successCount++
		}
	}

	if len(ta.completedAgents) > 0 {
		stats.SuccessRate = float64(successCount) / float64(len(ta.completedAgents))
	}

	return stats
}

// AggregatorStats provides high-level metrics
type AggregatorStats struct {
	TotalAgents     int
	ActiveAgents    int
	CompletedAgents int
	TotalDecisions  int
	SuccessRate     float64
}

// Stop gracefully shuts down aggregator
func (ta *TelemetryAggregator) Stop() error {
	close(ta.done)
	ta.wg.Wait()
	return ta.watcher.Stop()
}
