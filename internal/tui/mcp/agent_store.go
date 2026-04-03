package mcp

import (
	"log/slog"
	"sync"
	"time"
)

// AgentState represents the lifecycle state of a spawned agent.
type AgentState string

const (
	AgentStateRunning  AgentState = "running"
	AgentStateComplete AgentState = "complete"
	AgentStateError    AgentState = "error"
)

const (
	// completedEntryTTL is how long completed/errored entries are kept
	// before eviction. 30 minutes gives callers ample time to retrieve
	// results while preventing unbounded memory growth.
	completedEntryTTL = 30 * time.Minute

	// evictionInterval is how often the background reaper runs.
	evictionInterval = 5 * time.Minute

	// maxConcurrentSpawns limits how many claude subprocesses run
	// simultaneously. Set high enough to be effectively unlimited for
	// normal use, but prevents runaway fork-bombs from buggy callers.
	maxConcurrentSpawns = 10
)

// agentEntry holds the result of a spawned agent subprocess.
type agentEntry struct {
	AgentID   string
	Agent     string
	State     AgentState
	Output    string
	Error     string
	Cost      float64
	Turns     int
	Duration  string
	StartedAt time.Time
	DoneAt    time.Time
}

// AgentStore is a thread-safe in-memory store for async agent results.
// spawn_agent writes entries when launching; the background goroutine
// updates them on completion. get_agent_result reads them.
//
// Completed entries are automatically evicted after completedEntryTTL
// by a background reaper goroutine.
//
// The SpawnSem semaphore limits concurrent subprocess launches to
// maxConcurrentSpawns to avoid Anthropic API rate-limit (429) errors.
type AgentStore struct {
	mu      sync.RWMutex
	entries map[string]*agentEntry
	// done channels are signaled when an agent completes, allowing
	// get_agent_result to block efficiently instead of polling.
	done map[string]chan struct{}
	// stop signals the reaper goroutine to exit.
	stop chan struct{}
	// SpawnSem is a counting semaphore that limits concurrent subprocess
	// launches. Goroutines send to acquire a slot and receive to release.
	SpawnSem chan struct{}
}

// NewAgentStore creates a new empty store and starts the background
// reaper that evicts completed entries older than completedEntryTTL.
func NewAgentStore() *AgentStore {
	s := &AgentStore{
		entries:  make(map[string]*agentEntry),
		done:     make(map[string]chan struct{}),
		stop:     make(chan struct{}),
		SpawnSem: make(chan struct{}, maxConcurrentSpawns),
	}
	go s.reapLoop()
	return s
}

// reapLoop periodically removes completed/errored entries that have
// exceeded completedEntryTTL. Running entries are never evicted.
func (s *AgentStore) reapLoop() {
	ticker := time.NewTicker(evictionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.evictExpired()
		case <-s.stop:
			return
		}
	}
}

// evictExpired removes completed entries older than completedEntryTTL.
func (s *AgentStore) evictExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	evicted := 0
	for id, entry := range s.entries {
		if entry.State == AgentStateRunning {
			continue
		}
		if now.Sub(entry.DoneAt) > completedEntryTTL {
			delete(s.entries, id)
			delete(s.done, id)
			evicted++
		}
	}
	if evicted > 0 {
		slog.Info("agent_store: evicted expired entries", "count", evicted)
	}
}

// Stop shuts down the background reaper. Safe to call multiple times.
func (s *AgentStore) Stop() {
	select {
	case <-s.stop:
		// Already stopped.
	default:
		close(s.stop)
	}
}

// Register creates a new running entry and returns a done channel.
func (s *AgentStore) Register(agentID, agentType string) <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan struct{})
	s.entries[agentID] = &agentEntry{
		AgentID:   agentID,
		Agent:     agentType,
		State:     AgentStateRunning,
		StartedAt: time.Now(),
	}
	s.done[agentID] = ch
	return ch
}

// Complete marks an agent as done (success or error) and signals waiters.
func (s *AgentStore) Complete(agentID string, output string, errMsg string, cost float64, turns int, duration string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.entries[agentID]
	if !ok {
		return
	}

	// Guard against double-complete (e.g. timeout race).
	if entry.State != AgentStateRunning {
		return
	}

	entry.Output = output
	entry.Error = errMsg
	entry.Cost = cost
	entry.Turns = turns
	entry.Duration = duration
	entry.DoneAt = time.Now()

	if errMsg != "" {
		entry.State = AgentStateError
	} else {
		entry.State = AgentStateComplete
	}

	// Signal any waiters.
	if ch, ok := s.done[agentID]; ok {
		close(ch)
	}
}

// Get returns a copy of the entry, or nil if not found.
func (s *AgentStore) Get(agentID string) *agentEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.entries[agentID]
	if !ok {
		return nil
	}
	// Return a copy to avoid races.
	cp := *entry
	return &cp
}

// DoneChan returns the done channel for an agent, or nil if not found.
func (s *AgentStore) DoneChan(agentID string) <-chan struct{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.done[agentID]
}

// Len returns the number of entries (for testing).
func (s *AgentStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}
