package model

import (
	"testing"
	"time"

	"github.com/Bucket-Chemist/goYoke/internal/tui/components/modals"
	"github.com/Bucket-Chemist/goYoke/internal/tui/config"
	"github.com/Bucket-Chemist/goYoke/pkg/harnessproto"
)

// TestPublishSnapshotDebounced_SuppressesWithinWindow verifies that rapid calls
// to publishSnapshotDebounced within the debounce window do not flood the store.
// This mirrors the streaming scenario where token-by-token AssistantEvents fire
// far more often than downstream consumers need updates.
func TestPublishSnapshotDebounced_SuppressesWithinWindow(t *testing.T) {
	m := NewAppModel()
	store := m.shared.snapshotStore

	var calls int
	store.Subscribe(func(_, _ harnessproto.SessionSnapshot) {
		calls++
	})

	// First call always goes through (lastPublishTime is zero).
	m.publishSnapshotDebounced()
	if calls != 1 {
		t.Fatalf("first debounced call should publish; got %d calls", calls)
	}

	// Subsequent calls within the 500 ms window are suppressed even when state
	// would have changed (simulated by flipping Streaming).
	m.statusLine.Streaming = true
	m.publishSnapshotDebounced()
	m.publishSnapshotDebounced()

	if calls != 1 {
		t.Errorf("calls within debounce window should be suppressed; got %d calls", calls)
	}

	// After advancing lastPublishTime past the window, the next call goes through.
	m.shared.lastPublishTime = time.Now().Add(-600 * time.Millisecond)
	m.publishSnapshotDebounced()

	if calls != 2 {
		t.Errorf("call after debounce window should publish; got %d calls", calls)
	}
}

// TestPublishSnapshot_ModalTransitionCapturedInStore verifies that publishSnapshot
// captures modal request and resolution transitions and that the store reflects
// the correct status and Pending descriptor at each stage.
func TestPublishSnapshot_ModalTransitionCapturedInStore(t *testing.T) {
	m := NewAppModel()
	store := m.shared.snapshotStore

	// Publish idle baseline.
	m.publishSnapshot()
	idleSnap := store.Latest()

	if idleSnap.Status != "idle" {
		t.Fatalf("idle snapshot Status = %q; want idle", idleSnap.Status)
	}
	if idleSnap.Pending != nil {
		t.Fatalf("idle snapshot Pending should be nil, got %+v", idleSnap.Pending)
	}

	// Activate a bridge modal request (mirrors handleBridgeModalRequest fallback path).
	km := config.DefaultKeyMap()
	mq := modals.NewModalQueue(km)
	ph := modals.NewPermissionHandler(&mq)
	m.shared.modalQueue = &mq
	m.shared.permHandler = ph
	_ = ph.HandleBridgeRequest("req-modal-1", "Please choose an option:", []string{"A", "B"})

	// Publish modal state.
	m.publishSnapshot()
	modalSnap := store.Latest()

	if modalSnap.Status != "waiting_modal" {
		t.Errorf("modal snapshot Status = %q; want waiting_modal", modalSnap.Status)
	}
	if modalSnap.Pending == nil {
		t.Fatal("modal snapshot Pending must be non-nil")
	}
	if modalSnap.Pending.Kind != "modal" {
		t.Errorf("modal snapshot Pending.Kind = %q; want modal", modalSnap.Pending.Kind)
	}
	if modalSnap.StateHash == idleSnap.StateHash {
		t.Error("StateHash should differ between idle and modal snapshots")
	}
}

// TestPublishSnapshot_PermissionTransitionCapturedInStore verifies that a
// permission gate request surfaces as Kind "permission" in the store.
func TestPublishSnapshot_PermissionTransitionCapturedInStore(t *testing.T) {
	m := NewAppModel()
	store := m.shared.snapshotStore

	km := config.DefaultKeyMap()
	mq := modals.NewModalQueue(km)
	ph := modals.NewPermissionHandler(&mq)
	m.shared.modalQueue = &mq
	m.shared.permHandler = ph

	// Publish idle baseline.
	m.publishSnapshot()
	idleSnap := store.Latest()

	// Activate a permission gate (mirrors handleCLIPermissionRequest).
	_ = ph.HandlePermGateRequest("req-perm-1", "Allow bash command?", []string{"Allow", "Deny", "Allow for Session"}, 30000)

	m.publishSnapshot()
	permSnap := store.Latest()

	if permSnap.Status != "waiting_permission" {
		t.Errorf("permission snapshot Status = %q; want waiting_permission", permSnap.Status)
	}
	if permSnap.Pending == nil {
		t.Fatal("permission snapshot Pending must be non-nil")
	}
	if permSnap.Pending.Kind != "permission" {
		t.Errorf("permission snapshot Pending.Kind = %q; want permission", permSnap.Pending.Kind)
	}
	if permSnap.StateHash == idleSnap.StateHash {
		t.Error("StateHash should differ between idle and permission snapshots")
	}
}

// TestSnapshotStore_InitializedByNewAppModel verifies that NewAppModel
// initialises the snapshotStore so publish calls never panic on a default model.
func TestSnapshotStore_InitializedByNewAppModel(t *testing.T) {
	m := NewAppModel()
	if m.shared == nil {
		t.Fatal("shared must be non-nil after NewAppModel")
	}
	if m.shared.snapshotStore == nil {
		t.Fatal("snapshotStore must be initialised by NewAppModel")
	}

	// publishSnapshot and publishSnapshotDebounced must not panic.
	m.publishSnapshot()
	m.publishSnapshotDebounced()

	// SnapshotStore getter must return the same store.
	if got := m.SnapshotStore(); got != m.shared.snapshotStore {
		t.Error("SnapshotStore() should return m.shared.snapshotStore")
	}
}

func TestPublishSnapshotPublic_PrimesStore(t *testing.T) {
	m := NewAppModel()

	if got := m.shared.snapshotStore.Latest(); !got.Timestamp.IsZero() {
		t.Fatal("new snapshot store should be empty before priming publish")
	}

	m.PublishSnapshotPublic()

	got := m.shared.snapshotStore.Latest()
	if got.Timestamp.IsZero() {
		t.Fatal("PublishSnapshotPublic should prime the snapshot store")
	}
	if got.Status != "idle" {
		t.Errorf("primed snapshot Status = %q; want idle", got.Status)
	}
	if !m.shared.lastPublishTime.IsZero() {
		t.Error("PublishSnapshotPublic should not consume the streaming debounce timer")
	}
}
