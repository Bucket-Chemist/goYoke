package main

import (
	"fmt"
	"sync"
	"time"
)

// SpawnStatus represents the lifecycle state of a background spawn.
type SpawnStatus string

const (
	SpawnStatusRunning   SpawnStatus = "running"
	SpawnStatusCompleted SpawnStatus = "completed"
	SpawnStatusFailed    SpawnStatus = "failed"
	SpawnStatusTimeout   SpawnStatus = "timeout"
)

// BackgroundSpawn holds the state and eventual result of a background spawn.
type BackgroundSpawn struct {
	AgentID   string
	Agent     string // agent type ID from agents-index.json
	Status    SpawnStatus
	StartTime time.Time
	Result    *SpawnAgentOutput // nil while running

	done      chan struct{} // closed when result is available
	closeOnce sync.Once    // guards against double-close panic
}

// SpawnSnapshot is a read-only copy of a BackgroundSpawn's public state.
// It omits internal synchronization primitives so it is safe to copy by value.
type SpawnSnapshot struct {
	AgentID   string
	Agent     string
	Status    SpawnStatus
	StartTime time.Time
	Result    *SpawnAgentOutput
}

// BackgroundSpawnStore is a thread-safe store for background spawn results.
// It supports non-blocking lookups and channel-based blocking waits.
type BackgroundSpawnStore struct {
	mu     sync.RWMutex
	spawns map[string]*BackgroundSpawn
}

// NewBackgroundSpawnStore creates an empty store.
func NewBackgroundSpawnStore() *BackgroundSpawnStore {
	return &BackgroundSpawnStore{
		spawns: make(map[string]*BackgroundSpawn),
	}
}

// Register creates a new background spawn entry with status=running and returns
// it. The caller should later call Complete to deliver the result and unblock
// any waiters.
func (s *BackgroundSpawnStore) Register(agentID, agentType string) *BackgroundSpawn {
	bs := &BackgroundSpawn{
		AgentID:   agentID,
		Agent:     agentType,
		Status:    SpawnStatusRunning,
		StartTime: time.Now(),
		done:      make(chan struct{}),
	}
	s.mu.Lock()
	s.spawns[agentID] = bs
	s.mu.Unlock()
	return bs
}

// Complete delivers the result for a background spawn, updates its status, and
// unblocks all waiters. It is idempotent — calling Complete on an already
// completed spawn is a no-op.
func (s *BackgroundSpawnStore) Complete(agentID string, result *SpawnAgentOutput) {
	s.mu.Lock()
	bs, ok := s.spawns[agentID]
	if !ok {
		s.mu.Unlock()
		return
	}
	// Already completed — no-op.
	if bs.Status != SpawnStatusRunning {
		s.mu.Unlock()
		return
	}
	bs.Result = result
	if result != nil && !result.Success {
		bs.Status = SpawnStatusFailed
	} else {
		bs.Status = SpawnStatusCompleted
	}
	s.mu.Unlock()

	// Close done channel outside the lock to unblock waiters. sync.Once
	// prevents a double-close panic if Complete is called concurrently.
	bs.closeOnce.Do(func() { close(bs.done) })
}

// CompleteTimeout marks a background spawn as timed out and unblocks waiters.
func (s *BackgroundSpawnStore) CompleteTimeout(agentID string, result *SpawnAgentOutput) {
	s.mu.Lock()
	bs, ok := s.spawns[agentID]
	if !ok {
		s.mu.Unlock()
		return
	}
	if bs.Status != SpawnStatusRunning {
		s.mu.Unlock()
		return
	}
	bs.Result = result
	bs.Status = SpawnStatusTimeout
	s.mu.Unlock()

	bs.closeOnce.Do(func() { close(bs.done) })
}

// Get returns a snapshot of the background spawn for the given agentID. The
// returned struct is a value copy — mutations do not affect the store. Returns
// false if the agentID is not registered.
func (s *BackgroundSpawnStore) Get(agentID string) (SpawnSnapshot, bool) {
	s.mu.RLock()
	bs, ok := s.spawns[agentID]
	if !ok {
		s.mu.RUnlock()
		return SpawnSnapshot{}, false
	}
	snap := SpawnSnapshot{
		AgentID:   bs.AgentID,
		Agent:     bs.Agent,
		Status:    bs.Status,
		StartTime: bs.StartTime,
		Result:    bs.Result,
	}
	s.mu.RUnlock()
	return snap, true
}

// Wait blocks until the background spawn completes or the timeout elapses.
// Returns the result on success, or an error if the agentID is unknown or the
// timeout is exceeded.
func (s *BackgroundSpawnStore) Wait(agentID string, timeout time.Duration) (*SpawnAgentOutput, error) {
	s.mu.RLock()
	bs, ok := s.spawns[agentID]
	if !ok {
		s.mu.RUnlock()
		return nil, fmt.Errorf("unknown spawn_id: %s", agentID)
	}
	// Grab the done channel while holding the read lock, then release.
	done := bs.done
	s.mu.RUnlock()

	select {
	case <-done:
		// Re-read under lock to get final result.
		s.mu.RLock()
		result := bs.Result
		s.mu.RUnlock()
		return result, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for spawn_id %s after %s", agentID, timeout)
	}
}

// List returns a snapshot of all background spawns. Each entry is a value copy.
func (s *BackgroundSpawnStore) List() []SpawnSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]SpawnSnapshot, 0, len(s.spawns))
	for _, bs := range s.spawns {
		out = append(out, SpawnSnapshot{
			AgentID:   bs.AgentID,
			Agent:     bs.Agent,
			Status:    bs.Status,
			StartTime: bs.StartTime,
			Result:    bs.Result,
		})
	}
	return out
}
