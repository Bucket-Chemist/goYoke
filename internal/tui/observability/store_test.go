package observability_test

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Bucket-Chemist/goYoke/internal/tui/observability"
	"github.com/Bucket-Chemist/goYoke/pkg/harnessproto"
)

func snap(stateHash, publishHash, status string) harnessproto.SessionSnapshot {
	return harnessproto.SessionSnapshot{
		StateHash:   stateHash,
		PublishHash: publishHash,
		Status:      status,
	}
}

func TestUpdate_FirstUpdateAlwaysStored(t *testing.T) {
	store := observability.New()

	var calls int
	store.Subscribe(func(_, new harnessproto.SessionSnapshot) {
		calls++
	})

	store.Update(snap("s1", "p1", "idle"))

	if calls != 1 {
		t.Fatalf("expected 1 subscriber call after first Update, got %d", calls)
	}
	if got := store.Latest().StateHash; got != "s1" {
		t.Fatalf("Latest().StateHash = %q, want %q", got, "s1")
	}
}

// TestUpdate_DuplicateStateHashSuppressed verifies that a second Update with an
// identical StateHash does not call subscribers or overwrite Latest.
func TestUpdate_DuplicateStateHashSuppressed(t *testing.T) {
	store := observability.New()

	var calls int
	store.Subscribe(func(_, _ harnessproto.SessionSnapshot) {
		calls++
	})

	store.Update(snap("s1", "p1", "idle"))
	store.Update(snap("s1", "p1", "idle")) // duplicate
	store.Update(snap("s1", "p2", "idle")) // different PublishHash, same StateHash — still duplicate

	if calls != 1 {
		t.Fatalf("expected 1 call (first update only), got %d", calls)
	}
}

// TestUpdate_StateHashChanges_PublishHashSame demonstrates the core distinction:
// a StateHash change (streaming start) triggers the subscriber, but the
// PublishHash may remain the same because streaming onset is not
// notification-worthy.
func TestUpdate_StateHashChanges_PublishHashSame(t *testing.T) {
	store := observability.New()

	type notification struct {
		oldPublish string
		newPublish string
		stateChanged bool
	}
	var got []notification

	store.Subscribe(func(old, new harnessproto.SessionSnapshot) {
		got = append(got, notification{
			oldPublish:   old.PublishHash,
			newPublish:   new.PublishHash,
			stateChanged: old.StateHash != new.StateHash,
		})
	})

	// First update: idle, no notification context.
	store.Update(snap("state-idle", "pub-idle", "idle"))

	// Streaming starts: StateHash changes because Streaming flag toggled, but
	// PublishHash stays the same — streaming onset is not a human notification.
	store.Update(snap("state-streaming", "pub-idle", "streaming"))

	// Streaming completes and a new assistant message arrives: both hashes change.
	store.Update(snap("state-done", "pub-done", "idle"))

	if len(got) != 3 {
		t.Fatalf("expected 3 subscriber calls, got %d", len(got))
	}

	// Second notification: StateHash changed, PublishHash did NOT.
	if got[1].stateChanged != true {
		t.Error("second notification: StateHash should have changed")
	}
	if got[1].oldPublish != got[1].newPublish {
		t.Errorf("second notification: PublishHash should be unchanged; old=%q new=%q",
			got[1].oldPublish, got[1].newPublish)
	}

	// Third notification: both changed.
	if got[2].oldPublish == got[2].newPublish {
		t.Error("third notification: PublishHash should have changed")
	}
}

func TestLatest_BeforeUpdate_ReturnsZeroValue(t *testing.T) {
	store := observability.New()
	if got := store.Latest().StateHash; got != "" {
		t.Fatalf("empty store Latest().StateHash = %q, want empty", got)
	}
}

func TestSubscribe_UnsubscribeStopsNotifications(t *testing.T) {
	store := observability.New()

	var calls int
	unsub := store.Subscribe(func(_, _ harnessproto.SessionSnapshot) {
		calls++
	})

	store.Update(snap("s1", "p1", "idle"))
	unsub()
	store.Update(snap("s2", "p2", "streaming"))

	if calls != 1 {
		t.Fatalf("expected 1 call before unsub, got %d", calls)
	}
}

func TestSubscribe_UnsubscribeIdempotent(t *testing.T) {
	store := observability.New()
	unsub := store.Subscribe(func(_, _ harnessproto.SessionSnapshot) {})

	// Calling unsub multiple times must not panic.
	unsub()
	unsub()
}

func TestSubscribe_MultipleSubscribers(t *testing.T) {
	store := observability.New()

	var a, b int
	store.Subscribe(func(_, _ harnessproto.SessionSnapshot) { a++ })
	store.Subscribe(func(_, _ harnessproto.SessionSnapshot) { b++ })

	store.Update(snap("s1", "p1", "idle"))
	store.Update(snap("s2", "p2", "streaming"))

	if a != 2 || b != 2 {
		t.Fatalf("each subscriber should be called twice; a=%d b=%d", a, b)
	}
}

// TestUpdate_Concurrent exercises the store under concurrent reads and writes
// to detect data races when run with -race.
func TestUpdate_Concurrent(t *testing.T) {
	store := observability.New()

	var notified atomic.Int64
	store.Subscribe(func(_, _ harnessproto.SessionSnapshot) {
		notified.Add(1)
	})

	const writers = 4
	const updatesPerWriter = 50
	var wg sync.WaitGroup

	for w := range writers {
		wg.Go(func() {
			for i := range updatesPerWriter {
				hash := string(rune('a'+w)) + string(rune('0'+i%10))
				store.Update(snap(hash, hash, "idle"))
			}
		})
	}

	// Concurrent readers.
	for range 4 {
		wg.Go(func() {
			for range updatesPerWriter {
				_ = store.Latest()
			}
		})
	}

	wg.Wait()

	if notified.Load() == 0 {
		t.Fatal("no notifications received during concurrent test")
	}
}
