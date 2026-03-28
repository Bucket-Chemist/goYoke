package main

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestRegisterAndGet(t *testing.T) {
	store := NewBackgroundSpawnStore()
	store.Register("agent-1", "go-pro")

	snap, ok := store.Get("agent-1")
	if !ok {
		t.Fatal("expected agent-1 to be registered")
	}
	if snap.Status != SpawnStatusRunning {
		t.Errorf("expected status running, got %s", snap.Status)
	}
	if snap.Agent != "go-pro" {
		t.Errorf("expected agent type go-pro, got %s", snap.Agent)
	}
	if snap.Result != nil {
		t.Error("expected nil result while running")
	}
}

func TestGetUnknown(t *testing.T) {
	store := NewBackgroundSpawnStore()
	_, ok := store.Get("nonexistent")
	if ok {
		t.Error("expected false for unknown agentID")
	}
}

func TestCompleteSuccess(t *testing.T) {
	store := NewBackgroundSpawnStore()
	store.Register("agent-1", "go-pro")

	result := &SpawnAgentOutput{
		AgentID: "agent-1",
		Agent:   "go-pro",
		Success: true,
		Output:  "done",
		Cost:    0.05,
		Turns:   3,
	}
	store.Complete("agent-1", result)

	snap, ok := store.Get("agent-1")
	if !ok {
		t.Fatal("expected agent-1 to exist")
	}
	if snap.Status != SpawnStatusCompleted {
		t.Errorf("expected status completed, got %s", snap.Status)
	}
	if snap.Result == nil {
		t.Fatal("expected non-nil result")
	}
	if snap.Result.Output != "done" {
		t.Errorf("expected output 'done', got %q", snap.Result.Output)
	}
}

func TestCompleteFailed(t *testing.T) {
	store := NewBackgroundSpawnStore()
	store.Register("agent-1", "go-pro")

	result := &SpawnAgentOutput{
		AgentID: "agent-1",
		Agent:   "go-pro",
		Success: false,
		Error:   "something broke",
	}
	store.Complete("agent-1", result)

	snap, _ := store.Get("agent-1")
	if snap.Status != SpawnStatusFailed {
		t.Errorf("expected status failed, got %s", snap.Status)
	}
}

func TestCompleteTimeout(t *testing.T) {
	store := NewBackgroundSpawnStore()
	store.Register("agent-1", "go-pro")

	result := &SpawnAgentOutput{
		AgentID: "agent-1",
		Agent:   "go-pro",
		Success: false,
		Error:   "timed out",
	}
	store.CompleteTimeout("agent-1", result)

	snap, _ := store.Get("agent-1")
	if snap.Status != SpawnStatusTimeout {
		t.Errorf("expected status timeout, got %s", snap.Status)
	}
}

func TestDoubleCompleteNoPanic(t *testing.T) {
	store := NewBackgroundSpawnStore()
	store.Register("agent-1", "go-pro")

	result := &SpawnAgentOutput{Success: true, Output: "first"}
	store.Complete("agent-1", result)

	// Second complete should be a no-op, not panic.
	result2 := &SpawnAgentOutput{Success: true, Output: "second"}
	store.Complete("agent-1", result2)

	snap, _ := store.Get("agent-1")
	if snap.Result.Output != "first" {
		t.Errorf("expected first result to stick, got %q", snap.Result.Output)
	}
}

func TestCompleteUnknownNoPanic(t *testing.T) {
	store := NewBackgroundSpawnStore()
	// Should not panic.
	store.Complete("nonexistent", &SpawnAgentOutput{Success: true})
}

func TestWaitSuccess(t *testing.T) {
	store := NewBackgroundSpawnStore()
	store.Register("agent-1", "go-pro")

	expected := &SpawnAgentOutput{
		AgentID: "agent-1",
		Success: true,
		Output:  "result from background",
	}

	// Complete after a short delay.
	go func() {
		time.Sleep(50 * time.Millisecond)
		store.Complete("agent-1", expected)
	}()

	got, err := store.Wait("agent-1", 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Output != expected.Output {
		t.Errorf("expected output %q, got %q", expected.Output, got.Output)
	}
}

func TestWaitAlreadyComplete(t *testing.T) {
	store := NewBackgroundSpawnStore()
	store.Register("agent-1", "go-pro")
	store.Complete("agent-1", &SpawnAgentOutput{Success: true, Output: "already done"})

	// Wait on an already-completed spawn should return immediately.
	got, err := store.Wait("agent-1", 100*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Output != "already done" {
		t.Errorf("expected 'already done', got %q", got.Output)
	}
}

func TestWaitUnknownID(t *testing.T) {
	store := NewBackgroundSpawnStore()
	_, err := store.Wait("nonexistent", 100*time.Millisecond)
	if err == nil {
		t.Error("expected error for unknown spawn_id")
	}
}

func TestWaitTimeout(t *testing.T) {
	store := NewBackgroundSpawnStore()
	store.Register("agent-1", "go-pro")

	// Never complete — should hit timeout.
	_, err := store.Wait("agent-1", 50*time.Millisecond)
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestList(t *testing.T) {
	store := NewBackgroundSpawnStore()
	store.Register("agent-1", "go-pro")
	store.Register("agent-2", "go-tui")
	store.Complete("agent-2", &SpawnAgentOutput{Success: true})

	list := store.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(list))
	}

	found := map[string]SpawnStatus{}
	for _, bs := range list {
		found[bs.AgentID] = bs.Status
	}
	if found["agent-1"] != SpawnStatusRunning {
		t.Errorf("agent-1: expected running, got %s", found["agent-1"])
	}
	if found["agent-2"] != SpawnStatusCompleted {
		t.Errorf("agent-2: expected completed, got %s", found["agent-2"])
	}
}

func TestConcurrentAccess(t *testing.T) {
	store := NewBackgroundSpawnStore()
	const n = 50
	var wg sync.WaitGroup

	// Concurrent registers.
	wg.Add(n)
	for i := range n {
		go func(id string) {
			defer wg.Done()
			store.Register(id, "go-pro")
		}(fmt.Sprintf("agent-%d", i))
	}
	wg.Wait()

	// Concurrent completes.
	wg.Add(n)
	for i := range n {
		go func(id string) {
			defer wg.Done()
			store.Complete(id, &SpawnAgentOutput{Success: true, Output: id})
		}(fmt.Sprintf("agent-%d", i))
	}
	wg.Wait()

	// Verify all completed.
	list := store.List()
	if len(list) != n {
		t.Fatalf("expected %d entries, got %d", n, len(list))
	}
	for _, bs := range list {
		if bs.Status != SpawnStatusCompleted {
			t.Errorf("%s: expected completed, got %s", bs.AgentID, bs.Status)
		}
	}
}

