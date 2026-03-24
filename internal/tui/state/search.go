// Package state provides shared, thread-safe state containers for the
// GOgent-Fortress TUI.
//
// This file defines the SearchResult and SearchSource types used by the
// unified cross-panel fuzzy search overlay (TUI-059).  Placing these types
// in state (rather than model) prevents a circular import: model imports
// agents, and agents must implement SearchSource — so SearchSource cannot
// live in model without creating a cycle.
package state

// SearchResult represents a single search match from any source.
type SearchResult struct {
	// Source identifies which panel produced this result
	// (e.g. "conversation", "agents", "settings").
	Source string
	// Label is the primary display text for the result.
	Label string
	// Detail is secondary display text (e.g. a message snippet, agent type).
	Detail string
	// Score is the relevance score used for sort ordering; higher is better.
	Score int
}

// SearchSource is implemented by components that can contribute results to the
// unified search overlay.
//
// Implementations must be safe to call from the Bubbletea Update goroutine.
// Search must return nil when query is empty.
type SearchSource interface {
	// Search executes a case-insensitive substring search for query and
	// returns matching SearchResult values.  An empty query must return nil.
	Search(query string) []SearchResult
}
