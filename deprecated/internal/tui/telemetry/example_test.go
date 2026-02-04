package telemetry_test

import (
	"fmt"
	"log"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/deprecated/internal/tui/telemetry"
)

// Example_basicWatcher demonstrates basic usage of TelemetryWatcher
func Example_basicWatcher() {
	// Create watcher (uses config paths automatically)
	watcher, err := telemetry.NewTelemetryWatcher()
	if err != nil {
		log.Fatal(err)
	}

	// Start watching
	if err := watcher.Start(); err != nil {
		log.Fatal(err)
	}
	defer watcher.Stop()

	// Get event channels
	events := watcher.Events()

	// Process events in goroutine
	go func() {
		for {
			select {
			case event := <-events.AgentLifecycle:
				if event.EventType == "spawn" {
					fmt.Printf("Agent spawned: %s (%s)\n", event.AgentID, event.Tier)
				} else if event.EventType == "complete" {
					fmt.Printf("Agent completed: %s\n", event.AgentID)
				}

			case decision := <-events.RoutingDecisions:
				fmt.Printf("Routing: %s -> %s\n", decision.TaskDescription, decision.SelectedAgent)

			case update := <-events.DecisionUpdates:
				fmt.Printf("Decision outcome: %s, success=%v\n", update.DecisionID, update.OutcomeSuccess)

			case err := <-events.Errors:
				log.Printf("Watcher error: %v\n", err)
			}
		}
	}()

	// Keep running
	time.Sleep(1 * time.Hour)
}

// Example_aggregator demonstrates using the aggregator for state tracking
func Example_aggregator() {
	// Create aggregator (includes watcher)
	agg, err := telemetry.NewTelemetryAggregator()
	if err != nil {
		log.Fatal(err)
	}

	if err := agg.Start(); err != nil {
		log.Fatal(err)
	}
	defer agg.Stop()

	// Query aggregated state
	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		// Get current active agents
		activeAgents := agg.GetActiveAgents()
		fmt.Printf("Active agents: %d\n", len(activeAgents))
		for _, agent := range activeAgents {
			fmt.Printf("  - %s: %s (tier: %s)\n", agent.AgentID, agent.TaskDesc, agent.Tier)
		}

		// Get stats
		stats := agg.Stats()
		fmt.Printf("Stats: %d total, %d active, %.1f%% success\n",
			stats.TotalAgents, stats.ActiveAgents, stats.SuccessRate*100)
	}
}

// Example_customWatcher demonstrates creating a custom JSONL watcher
func Example_customWatcher() {
	// Custom parse function for specific event type
	parseFunc := func(data []byte) (interface{}, error) {
		// Parse custom format here
		return string(data), nil
	}

	watcher, err := telemetry.NewJSONLWatcher("/path/to/file.jsonl", parseFunc)
	if err != nil {
		log.Fatal(err)
	}

	if err := watcher.Start(); err != nil {
		log.Fatal(err)
	}
	defer watcher.Stop()

	// Read events
	for event := range watcher.Events() {
		fmt.Printf("Event: %v\n", event)
	}
}
