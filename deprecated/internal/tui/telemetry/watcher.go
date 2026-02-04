package telemetry

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
	pkgtel "github.com/Bucket-Chemist/GOgent-Fortress/pkg/telemetry"
)

// TelemetryWatcher manages multiple JSONL file watchers for different telemetry streams.
// Provides unified event channels for consumption by TUI components.
type TelemetryWatcher struct {
	// Individual file watchers
	lifecycleWatcher *JSONLWatcher
	decisionsWatcher *JSONLWatcher
	updatesWatcher   *JSONLWatcher
	collabWatcher    *JSONLWatcher

	// Unified output channels (typed)
	lifecycleEvents chan *pkgtel.AgentLifecycleEvent
	decisionEvents  chan *pkgtel.RoutingDecision
	updateEvents    chan *pkgtel.DecisionOutcomeUpdate
	errors          chan error

	// Control
	done chan struct{}
	wg   sync.WaitGroup
}

// TelemetryEvents provides read-only access to typed event channels
type TelemetryEvents struct {
	AgentLifecycle   <-chan *pkgtel.AgentLifecycleEvent
	RoutingDecisions <-chan *pkgtel.RoutingDecision
	DecisionUpdates  <-chan *pkgtel.DecisionOutcomeUpdate
	Errors           <-chan error
}

// NewTelemetryWatcher creates a new watcher for all telemetry files.
// Uses config.GetXXXPathWithProjectDir() for correct path resolution.
func NewTelemetryWatcher() (*TelemetryWatcher, error) {
	tw := &TelemetryWatcher{
		lifecycleEvents: make(chan *pkgtel.AgentLifecycleEvent, 100),
		decisionEvents:  make(chan *pkgtel.RoutingDecision, 100),
		updateEvents:    make(chan *pkgtel.DecisionOutcomeUpdate, 100),
		errors:          make(chan error, 50),
		done:            make(chan struct{}),
	}

	// Create lifecycle watcher
	lifecyclePath := config.GetAgentLifecyclePathWithProjectDir()
	lw, err := NewJSONLWatcher(lifecyclePath, parseLifecycleEvent)
	if err != nil {
		return nil, fmt.Errorf("failed to create lifecycle watcher: %w", err)
	}
	tw.lifecycleWatcher = lw

	// Create routing decisions watcher
	decisionsPath := config.GetRoutingDecisionsPathWithProjectDir()
	dw, err := NewJSONLWatcher(decisionsPath, parseRoutingDecision)
	if err != nil {
		return nil, fmt.Errorf("failed to create decisions watcher: %w", err)
	}
	tw.decisionsWatcher = dw

	// Create decision updates watcher
	updatesPath := config.GetRoutingDecisionUpdatesPathWithProjectDir()
	uw, err := NewJSONLWatcher(updatesPath, parseDecisionUpdate)
	if err != nil {
		return nil, fmt.Errorf("failed to create updates watcher: %w", err)
	}
	tw.updatesWatcher = uw

	// Create collaborations watcher (optional - not in original spec)
	collabPath := config.GetCollaborationsPathWithProjectDir()
	cw, err := NewJSONLWatcher(collabPath, nil) // No parser for now
	if err != nil {
		return nil, fmt.Errorf("failed to create collaborations watcher: %w", err)
	}
	tw.collabWatcher = cw

	return tw, nil
}

// Start begins watching all telemetry files and forwarding events
func (tw *TelemetryWatcher) Start() error {
	// Start all watchers
	if err := tw.lifecycleWatcher.Start(); err != nil {
		return fmt.Errorf("failed to start lifecycle watcher: %w", err)
	}

	if err := tw.decisionsWatcher.Start(); err != nil {
		return fmt.Errorf("failed to start decisions watcher: %w", err)
	}

	if err := tw.updatesWatcher.Start(); err != nil {
		return fmt.Errorf("failed to start updates watcher: %w", err)
	}

	if err := tw.collabWatcher.Start(); err != nil {
		return fmt.Errorf("failed to start collaborations watcher: %w", err)
	}

	// Start forwarding goroutines
	tw.wg.Add(4)
	go tw.forwardLifecycleEvents()
	go tw.forwardDecisionEvents()
	go tw.forwardUpdateEvents()
	go tw.forwardErrors()

	return nil
}

// forwardLifecycleEvents forwards lifecycle events to typed channel
func (tw *TelemetryWatcher) forwardLifecycleEvents() {
	defer tw.wg.Done()
	for {
		select {
		case event := <-tw.lifecycleWatcher.Events():
			if event == nil {
				return
			}
			if typed, ok := event.(*pkgtel.AgentLifecycleEvent); ok {
				select {
				case tw.lifecycleEvents <- typed:
				case <-tw.done:
					return
				}
			}
		case <-tw.done:
			return
		}
	}
}

// forwardDecisionEvents forwards routing decision events to typed channel
func (tw *TelemetryWatcher) forwardDecisionEvents() {
	defer tw.wg.Done()
	for {
		select {
		case event := <-tw.decisionsWatcher.Events():
			if event == nil {
				return
			}
			if typed, ok := event.(*pkgtel.RoutingDecision); ok {
				select {
				case tw.decisionEvents <- typed:
				case <-tw.done:
					return
				}
			}
		case <-tw.done:
			return
		}
	}
}

// forwardUpdateEvents forwards decision update events to typed channel
func (tw *TelemetryWatcher) forwardUpdateEvents() {
	defer tw.wg.Done()
	for {
		select {
		case event := <-tw.updatesWatcher.Events():
			if event == nil {
				return
			}
			if typed, ok := event.(*pkgtel.DecisionOutcomeUpdate); ok {
				select {
				case tw.updateEvents <- typed:
				case <-tw.done:
					return
				}
			}
		case <-tw.done:
			return
		}
	}
}

// forwardErrors collects errors from all watchers
func (tw *TelemetryWatcher) forwardErrors() {
	defer tw.wg.Done()
	for {
		select {
		case err := <-tw.lifecycleWatcher.Errors():
			if err == nil {
				continue
			}
			select {
			case tw.errors <- fmt.Errorf("lifecycle watcher: %w", err):
			case <-tw.done:
				return
			}
		case err := <-tw.decisionsWatcher.Errors():
			if err == nil {
				continue
			}
			select {
			case tw.errors <- fmt.Errorf("decisions watcher: %w", err):
			case <-tw.done:
				return
			}
		case err := <-tw.updatesWatcher.Errors():
			if err == nil {
				continue
			}
			select {
			case tw.errors <- fmt.Errorf("updates watcher: %w", err):
			case <-tw.done:
				return
			}
		case err := <-tw.collabWatcher.Errors():
			if err == nil {
				continue
			}
			select {
			case tw.errors <- fmt.Errorf("collaborations watcher: %w", err):
			case <-tw.done:
				return
			}
		case <-tw.done:
			return
		}
	}
}

// Events returns read-only access to all event channels
func (tw *TelemetryWatcher) Events() TelemetryEvents {
	return TelemetryEvents{
		AgentLifecycle:   tw.lifecycleEvents,
		RoutingDecisions: tw.decisionEvents,
		DecisionUpdates:  tw.updateEvents,
		Errors:           tw.errors,
	}
}

// Stop gracefully stops all watchers and closes channels
func (tw *TelemetryWatcher) Stop() error {
	// Signal shutdown
	close(tw.done)

	// Stop all watchers
	var errors []error

	if err := tw.lifecycleWatcher.Stop(); err != nil {
		errors = append(errors, fmt.Errorf("lifecycle watcher: %w", err))
	}

	if err := tw.decisionsWatcher.Stop(); err != nil {
		errors = append(errors, fmt.Errorf("decisions watcher: %w", err))
	}

	if err := tw.updatesWatcher.Stop(); err != nil {
		errors = append(errors, fmt.Errorf("updates watcher: %w", err))
	}

	if err := tw.collabWatcher.Stop(); err != nil {
		errors = append(errors, fmt.Errorf("collaborations watcher: %w", err))
	}

	// Wait for forwarding goroutines to finish
	tw.wg.Wait()

	// Close output channels
	close(tw.lifecycleEvents)
	close(tw.decisionEvents)
	close(tw.updateEvents)
	close(tw.errors)

	// Return any errors
	if len(errors) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errors)
	}

	return nil
}

// Parse functions for converting JSONL to typed events

func parseLifecycleEvent(data []byte) (interface{}, error) {
	var event pkgtel.AgentLifecycleEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to parse lifecycle event: %w", err)
	}
	return &event, nil
}

func parseRoutingDecision(data []byte) (interface{}, error) {
	var decision pkgtel.RoutingDecision
	if err := json.Unmarshal(data, &decision); err != nil {
		return nil, fmt.Errorf("failed to parse routing decision: %w", err)
	}
	return &decision, nil
}

func parseDecisionUpdate(data []byte) (interface{}, error) {
	var update pkgtel.DecisionOutcomeUpdate
	if err := json.Unmarshal(data, &update); err != nil {
		return nil, fmt.Errorf("failed to parse decision update: %w", err)
	}
	return &update, nil
}
