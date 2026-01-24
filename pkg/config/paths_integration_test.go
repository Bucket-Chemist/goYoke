package config

import (
	"testing"
)

func TestAttentionGateWorkflow_ReminderAt10(t *testing.T) {
	// Use t.TempDir() for isolation (NOT global COUNTER_FILE)
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// Increment to 10 (should trigger reminder)
	var count int
	var err error
	for i := 0; i < 10; i++ {
		count, err = GetToolCountAndIncrement()
		if err != nil {
			t.Fatalf("Increment %d failed: %v", i+1, err)
		}
	}

	if count != 10 {
		t.Errorf("Expected count 10, got: %d", count)
	}

	if !ShouldRemind(count) {
		t.Error("Should trigger reminder at tool #10")
	}

	if ShouldFlush(count) {
		t.Error("Should NOT trigger flush at tool #10 (threshold is 20)")
	}
}

func TestAttentionGateWorkflow_FlushAt20(t *testing.T) {
	// Use t.TempDir() for isolation
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// Increment to 20 (should trigger both reminder and flush)
	var count int
	var err error
	for i := 0; i < 20; i++ {
		count, err = GetToolCountAndIncrement()
		if err != nil {
			t.Fatalf("Increment %d failed: %v", i+1, err)
		}
	}

	if count != 20 {
		t.Errorf("Expected count 20, got: %d", count)
	}

	if !ShouldRemind(count) {
		t.Error("Should trigger reminder at tool #20 (multiple of 10)")
	}

	if !ShouldFlush(count) {
		t.Error("Should trigger flush at tool #20")
	}
}

func TestAttentionGateWorkflow_MultipleThresholds(t *testing.T) {
	// Use t.TempDir() for isolation
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	tests := []struct {
		targetCount  int
		shouldRemind bool
		shouldFlush  bool
		description  string
	}{
		{9, false, false, "Before first reminder"},
		{10, true, false, "First reminder"},
		{19, false, false, "Before first flush"},
		{20, true, true, "First flush + reminder"},
		{30, true, false, "Second reminder"},
		{40, true, true, "Second flush + reminder"},
	}

	var count int
	for _, tc := range tests {
		// Increment to target
		for count < tc.targetCount {
			count, _ = GetToolCountAndIncrement()
		}

		if ShouldRemind(count) != tc.shouldRemind {
			t.Errorf("%s: ShouldRemind(%d) = %v, expected %v",
				tc.description, count, ShouldRemind(count), tc.shouldRemind)
		}

		if ShouldFlush(count) != tc.shouldFlush {
			t.Errorf("%s: ShouldFlush(%d) = %v, expected %v",
				tc.description, count, ShouldFlush(count), tc.shouldFlush)
		}
	}
}
