// Package observability provides a thread-safe store for the latest harness
// SessionSnapshot and a subscription mechanism for downstream sinks.
//
// The store does not compute hashes itself. Callers must populate StateHash
// and PublishHash on the snapshot (e.g. via model.BuildHarnessSnapshot) before
// calling Update. The store uses these fields for duplicate suppression.
//
// StateHash changes on every operator-visible state mutation.
// PublishHash changes only when a human notification is warranted.
// Downstream sinks may compare the two to decide whether to act.
package observability

import (
	"sync"

	"github.com/Bucket-Chemist/goYoke/pkg/harnessproto"
)

// SnapshotStore holds the most recent SessionSnapshot and notifies registered
// subscribers whenever the StateHash changes.
//
// Its zero value is not usable; use New.
type SnapshotStore struct {
	mu          sync.RWMutex
	current     harnessproto.SessionSnapshot
	initialized bool
	subs        []sub
	nextID      int
}

type sub struct {
	id int
	fn func(old, new harnessproto.SessionSnapshot)
}

// New returns a ready-to-use SnapshotStore.
func New() *SnapshotStore {
	return &SnapshotStore{}
}

// Update stores snap when its StateHash differs from the currently held
// snapshot (or when the store has never been written to). All registered
// subscribers are called outside the write lock with (old, new).
//
// Callers must ensure snap.StateHash and snap.PublishHash are populated before
// calling Update; an empty StateHash is treated as a valid (zero) hash value.
func (s *SnapshotStore) Update(snap harnessproto.SessionSnapshot) {
	s.mu.Lock()
	if s.initialized && s.current.StateHash == snap.StateHash {
		s.mu.Unlock()
		return
	}
	old := s.current
	s.current = snap
	s.initialized = true

	// Copy subscriber slice so callbacks run without holding the lock.
	fns := make([]func(harnessproto.SessionSnapshot, harnessproto.SessionSnapshot), len(s.subs))
	for i, sub := range s.subs {
		fns[i] = sub.fn
	}
	s.mu.Unlock()

	for _, fn := range fns {
		fn(old, snap)
	}
}

// Latest returns the most recently stored SessionSnapshot. Returns the zero
// value before the first Update call.
func (s *SnapshotStore) Latest() harnessproto.SessionSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current
}

// Subscribe registers fn to be called after every Update that stores a new
// snapshot. fn receives the previous and new snapshots as (old, new).
//
// Subscribers run synchronously in the calling goroutine of Update, outside
// the store's write lock. Long-running work should be dispatched to a
// goroutine inside fn.
//
// Subscribe returns an unsubscribe function; calling it removes the subscriber.
// The unsubscribe function is safe to call multiple times.
func (s *SnapshotStore) Subscribe(fn func(old, new harnessproto.SessionSnapshot)) func() {
	s.mu.Lock()
	id := s.nextID
	s.nextID++
	s.subs = append(s.subs, sub{id: id, fn: fn})
	s.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			for i, sub := range s.subs {
				if sub.id == id {
					s.subs = append(s.subs[:i], s.subs[i+1:]...)
					return
				}
			}
		})
	}
}
