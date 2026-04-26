// Package model defines shared state types for the goYoke TUI.
// This file wires the snapshot builder (HL-003) and observability store (HL-004)
// into the Bubbletea event loop. Two helpers manage when snapshots are published:
//
//   - publishSnapshot: unconditional publish; for discrete state transitions.
//   - publishSnapshotDebounced: rate-limited to one publish per 500 ms; for
//     high-frequency streaming events so token-by-token ticks do not flood the
//     store or downstream sinks.
//
// Hash-based deduplication is handled by SnapshotStore.Update; callers must NOT
// add their own dedup logic.
package model

import (
	"time"

	"github.com/Bucket-Chemist/goYoke/internal/tui/observability"
)

// streamingPublishDebounce is the minimum gap between successive snapshot
// publications during a streaming turn.
const streamingPublishDebounce = 500 * time.Millisecond

// publishSnapshot builds the current session snapshot and writes it to the
// store. Call this at the end of any handler that produces a meaningful state
// transition (modal request, result completion, agent update, etc.).
//
// Hash deduplication is handled by the store; callers do not need to compare
// hashes before calling.
func (m AppModel) publishSnapshot() {
	if m.shared == nil || m.shared.snapshotStore == nil {
		return
	}
	snap := m.BuildHarnessSnapshot()
	m.shared.snapshotStore.Update(snap)
	m.shared.lastPublishTime = time.Now()
}

// publishSnapshotDebounced publishes a snapshot only when at least
// streamingPublishDebounce has elapsed since the previous publication.
// Use this for AssistantEvent (streaming token) handlers where the call
// rate far exceeds the human-perceptible update rate.
//
// The first call always publishes (lastPublishTime is zero on startup).
func (m AppModel) publishSnapshotDebounced() {
	if m.shared == nil || m.shared.snapshotStore == nil {
		return
	}
	if !m.shared.lastPublishTime.IsZero() && time.Since(m.shared.lastPublishTime) < streamingPublishDebounce {
		return
	}
	m.publishSnapshot()
}

// SnapshotStore returns the observability store holding the latest session
// snapshot. External components (control server, relay) may call Subscribe on
// the returned store to receive state-change notifications without adding
// Discord-specific or relay-specific wiring to the model package.
//
// Returns nil when the model was not constructed via NewAppModel.
func (m AppModel) SnapshotStore() *observability.SnapshotStore {
	if m.shared == nil {
		return nil
	}
	return m.shared.snapshotStore
}
