package harness

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBehavioralInvariant_B1_RequiredFields(t *testing.T) {
	invariant := BehavioralInvariantByID("B1")
	if invariant == nil {
		t.Fatal("B1 invariant not found")
	}

	tests := []struct {
		name       string
		sharpEdges []map[string]interface{}
		wantPass   bool
	}{
		{
			name:       "empty edges - pass",
			sharpEdges: []map[string]interface{}{},
			wantPass:   true,
		},
		{
			name: "valid edge - pass",
			sharpEdges: []map[string]interface{}{
				{
					"error_type":          "TypeError",
					"consecutive_failures": float64(3),
					"timestamp":           float64(1705000000),
				},
			},
			wantPass: true,
		},
		{
			name: "missing error_type - fail",
			sharpEdges: []map[string]interface{}{
				{
					"consecutive_failures": float64(3),
					"timestamp":            float64(1705000000),
				},
			},
			wantPass: false,
		},
		{
			name: "failures below threshold - fail",
			sharpEdges: []map[string]interface{}{
				{
					"error_type":          "TypeError",
					"consecutive_failures": float64(2),
					"timestamp":           float64(1705000000),
				},
			},
			wantPass: false,
		},
		{
			name: "missing timestamp - fail",
			sharpEdges: []map[string]interface{}{
				{
					"error_type":          "TypeError",
					"consecutive_failures": float64(3),
				},
			},
			wantPass: false,
		},
		{
			name: "string timestamp - pass",
			sharpEdges: []map[string]interface{}{
				{
					"error_type":          "TypeError",
					"consecutive_failures": float64(3),
					"timestamp":           "2026-01-23T10:00:00Z",
				},
			},
			wantPass: true,
		},
		{
			name: "ts field instead of timestamp - pass",
			sharpEdges: []map[string]interface{}{
				{
					"error_type":          "TypeError",
					"consecutive_failures": float64(3),
					"ts":                  float64(1705000000),
				},
			},
			wantPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &BehavioralContext{
				SharpEdges: tt.sharpEdges,
				Config:     DefaultBehavioralConfig(),
			}

			passed, msg := invariant.Check(ctx)
			if passed != tt.wantPass {
				t.Errorf("B1 Check() = %v, want %v (msg: %s)", passed, tt.wantPass, msg)
			}
		})
	}
}

func TestBehavioralInvariant_B4_SchemaVersion(t *testing.T) {
	invariant := BehavioralInvariantByID("B4")
	if invariant == nil {
		t.Fatal("B4 invariant not found")
	}

	tests := []struct {
		name     string
		handoff  map[string]interface{}
		wantPass bool
	}{
		{
			name:     "no handoff - pass (vacuously)",
			handoff:  nil,
			wantPass: true,
		},
		{
			name:     "empty handoff - pass (vacuously)",
			handoff:  map[string]interface{}{},
			wantPass: true,
		},
		{
			name: "correct version - pass",
			handoff: map[string]interface{}{
				"schema_version": "1.2",
			},
			wantPass: true,
		},
		{
			name: "wrong version - fail",
			handoff: map[string]interface{}{
				"schema_version": "1.0",
			},
			wantPass: false,
		},
		{
			name: "missing version - fail",
			handoff: map[string]interface{}{
				"session_id": "test",
			},
			wantPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &BehavioralContext{
				Handoff: tt.handoff,
				Config:  DefaultBehavioralConfig(),
			}

			passed, msg := invariant.Check(ctx)
			if passed != tt.wantPass {
				t.Errorf("B4 Check() = %v, want %v (msg: %s)", passed, tt.wantPass, msg)
			}
		})
	}
}

func TestBehavioralInvariant_B5_FailureTrackerAccuracy(t *testing.T) {
	invariant := BehavioralInvariantByID("B5")
	if invariant == nil {
		t.Fatal("B5 invariant not found")
	}

	tests := []struct {
		name       string
		tracker    []map[string]interface{}
		sharpEdges []map[string]interface{}
		wantPass   bool
	}{
		{
			name:       "empty - pass",
			tracker:    nil,
			sharpEdges: nil,
			wantPass:   true,
		},
		{
			name: "matching counts - pass",
			tracker: []map[string]interface{}{
				{"file": "test.py", "error_type": "TypeError"},
				{"file": "test.py", "error_type": "TypeError"},
				{"file": "test.py", "error_type": "TypeError"},
			},
			sharpEdges: []map[string]interface{}{
				{"file": "test.py", "error_type": "TypeError", "consecutive_failures": float64(3)},
			},
			wantPass: true,
		},
		{
			name: "mismatched counts - fail",
			tracker: []map[string]interface{}{
				{"file": "test.py", "error_type": "TypeError"},
				{"file": "test.py", "error_type": "TypeError"},
				{"file": "test.py", "error_type": "TypeError"},
			},
			sharpEdges: []map[string]interface{}{
				{"file": "test.py", "error_type": "TypeError", "consecutive_failures": float64(1)},
			},
			wantPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &BehavioralContext{
				FailureTrackerLog: tt.tracker,
				SharpEdges:        tt.sharpEdges,
				Config:            DefaultBehavioralConfig(),
			}

			passed, msg := invariant.Check(ctx)
			if passed != tt.wantPass {
				t.Errorf("B5 Check() = %v, want %v (msg: %s)", passed, tt.wantPass, msg)
			}
		})
	}
}

func TestBehavioralInvariant_B6_BlockingAtThreshold(t *testing.T) {
	invariant := BehavioralInvariantByID("B6")
	if invariant == nil {
		t.Fatal("B6 invariant not found")
	}

	tests := []struct {
		name       string
		sharpEdges []map[string]interface{}
		wantPass   bool
	}{
		{
			name:       "no edges - pass",
			sharpEdges: nil,
			wantPass:   true,
		},
		{
			name: "edge at threshold - pass",
			sharpEdges: []map[string]interface{}{
				{"consecutive_failures": float64(3)},
			},
			wantPass: true,
		},
		{
			name: "edge above threshold - pass",
			sharpEdges: []map[string]interface{}{
				{"consecutive_failures": float64(5)},
			},
			wantPass: true,
		},
		{
			name: "edge below threshold - fail",
			sharpEdges: []map[string]interface{}{
				{"consecutive_failures": float64(2)},
			},
			wantPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &BehavioralContext{
				SharpEdges: tt.sharpEdges,
				Config:     DefaultBehavioralConfig(),
			}

			passed, msg := invariant.Check(ctx)
			if passed != tt.wantPass {
				t.Errorf("B6 Check() = %v, want %v (msg: %s)", passed, tt.wantPass, msg)
			}
		})
	}
}

func TestBehavioralInvariant_B7_JSONLValidity(t *testing.T) {
	invariant := BehavioralInvariantByID("B7")
	if invariant == nil {
		t.Fatal("B7 invariant not found")
	}

	t.Run("valid files - pass", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "b7-test-")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		os.MkdirAll(filepath.Join(tmpDir, ".claude", "memory"), 0755)
		os.WriteFile(filepath.Join(tmpDir, ".claude", "memory", "pending-learnings.jsonl"),
			[]byte(`{"valid":"json"}`+"\n"), 0644)

		ctx := &BehavioralContext{
			TempDir: tmpDir,
			Config:  DefaultBehavioralConfig(),
		}

		passed, msg := invariant.Check(ctx)
		if !passed {
			t.Errorf("B7 Check() = false, want true (msg: %s)", msg)
		}
	})

	t.Run("invalid JSON - fail", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "b7-test-")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		os.MkdirAll(filepath.Join(tmpDir, ".claude", "memory"), 0755)
		os.WriteFile(filepath.Join(tmpDir, ".claude", "memory", "pending-learnings.jsonl"),
			[]byte(`{not valid json}`+"\n"), 0644)

		ctx := &BehavioralContext{
			TempDir: tmpDir,
			Config:  DefaultBehavioralConfig(),
		}

		passed, msg := invariant.Check(ctx)
		if passed {
			t.Errorf("B7 Check() = true, want false (msg: %s)", msg)
		}
	})

	t.Run("missing files - pass", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "b7-test-")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		ctx := &BehavioralContext{
			TempDir: tmpDir,
			Config:  DefaultBehavioralConfig(),
		}

		passed, msg := invariant.Check(ctx)
		if !passed {
			t.Errorf("B7 Check() = false, want true (msg: %s)", msg)
		}
	})
}

func TestLoadBehavioralContext(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ctx-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	os.MkdirAll(filepath.Join(tmpDir, ".claude", "memory"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, ".gogent"), 0755)

	os.WriteFile(filepath.Join(tmpDir, ".claude", "memory", "pending-learnings.jsonl"),
		[]byte(`{"file":"a.py","error_type":"Error"}`+"\n"+`{"file":"b.py","error_type":"Error"}`+"\n"), 0644)

	os.WriteFile(filepath.Join(tmpDir, ".claude", "memory", "handoffs.jsonl"),
		[]byte(`{"schema_version":"1.0"}`+"\n"+`{"schema_version":"1.2"}`+"\n"), 0644)

	os.WriteFile(filepath.Join(tmpDir, ".gogent", "failure-tracker.jsonl"),
		[]byte(`{"file":"a.py"}`+"\n"), 0644)

	ctx, err := LoadBehavioralContext(tmpDir, DefaultBehavioralConfig())
	if err != nil {
		t.Fatalf("LoadBehavioralContext failed: %v", err)
	}

	// Verify sharp edges loaded
	if len(ctx.SharpEdges) != 2 {
		t.Errorf("SharpEdges: got %d, want 2", len(ctx.SharpEdges))
	}

	// Verify handoff is last entry
	if ctx.Handoff["schema_version"] != "1.2" {
		t.Errorf("Handoff version: got %v, want 1.1", ctx.Handoff["schema_version"])
	}

	// Verify tracker loaded
	if len(ctx.FailureTrackerLog) != 1 {
		t.Errorf("FailureTrackerLog: got %d, want 1", len(ctx.FailureTrackerLog))
	}
}

func TestCheckBehavioralInvariants(t *testing.T) {
	// Valid context - all should pass
	ctx := &BehavioralContext{
		TempDir: "",
		SharpEdges: []map[string]interface{}{
			{
				"error_type":          "TypeError",
				"consecutive_failures": float64(3),
				"timestamp":           float64(1705000000),
			},
		},
		Handoff: map[string]interface{}{
			"schema_version": "1.2",
		},
		Config: DefaultBehavioralConfig(),
	}

	results := CheckBehavioralInvariants(ctx)

	// Should have results for all invariants
	if len(results) != len(BehavioralInvariants) {
		t.Errorf("Results count: got %d, want %d", len(results), len(BehavioralInvariants))
	}

	// Count passes
	passCount := 0
	for _, r := range results {
		if r.Passed {
			passCount++
		}
	}

	// B1, B4, B5, B6 should pass with this context (B7 may fail without TempDir)
	if passCount < 4 {
		t.Errorf("Pass count: got %d, want >= 4", passCount)
		for _, r := range results {
			if !r.Passed {
				t.Logf("  Failed: %s - %s", r.InvariantID, r.Message)
			}
		}
	}
}

func TestBehavioralInvariantByID(t *testing.T) {
	// Test finding existing invariants
	ids := []string{"B1", "B4", "B5", "B6", "B7"}
	for _, id := range ids {
		inv := BehavioralInvariantByID(id)
		if inv == nil {
			t.Errorf("Invariant %s not found", id)
		}
		if inv != nil && inv.ID != id {
			t.Errorf("Invariant ID mismatch: got %s, want %s", inv.ID, id)
		}
	}

	// Test non-existent
	inv := BehavioralInvariantByID("B99")
	if inv != nil {
		t.Errorf("Expected nil for B99, got %v", inv)
	}
}
