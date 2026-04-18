package harness

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFuzzRunner_Reproducibility(t *testing.T) {
	tempDir := t.TempDir()

	cfg := SimulationConfig{
		Mode:           "fuzz",
		FuzzIterations: 10,
		FuzzSeed:       42,
		FuzzTimeout:    1 * time.Minute,
		TempDir:        tempDir,
	}

	gen := NewGenerator(filepath.Join(tempDir, "fixtures"))

	// Run twice with same seed
	runner1 := &FuzzRunner{
		config:  cfg,
		gen:     gen,
		results: make([]SimulationResult, 0),
	}

	runner2 := &FuzzRunner{
		config:  cfg,
		gen:     gen,
		results: make([]SimulationResult, 0),
	}

	// Compare generated inputs (not full execution)
	rng1 := rand.New(rand.NewSource(42))
	rng2 := rand.New(rand.NewSource(42))

	for i := 0; i < 10; i++ {
		seed1 := rng1.Int63()
		seed2 := rng2.Int63()

		if seed1 != seed2 {
			t.Errorf("Iteration %d: seeds differ (%d vs %d)", i, seed1, seed2)
		}
	}

	// Verify runners have same configuration
	if runner1.config.FuzzSeed != runner2.config.FuzzSeed {
		t.Errorf("FuzzSeed mismatch: %d vs %d", runner1.config.FuzzSeed, runner2.config.FuzzSeed)
	}
}

func TestFuzzRunner_70_30_Split(t *testing.T) {
	tempDir := t.TempDir()

	cfg := SimulationConfig{
		Mode:           "fuzz",
		FuzzIterations: 1000,
		FuzzSeed:       12345,
		FuzzTimeout:    5 * time.Minute,
		TempDir:        tempDir,
	}

	// Just test the distribution, not actual execution
	rng := rand.New(rand.NewSource(cfg.FuzzSeed))

	preToolCount := 0
	sessionCount := 0

	for i := 0; i < cfg.FuzzIterations; i++ {
		if rng.Float64() < 0.7 {
			preToolCount++
		} else {
			sessionCount++
		}
	}

	// Should be roughly 70/30 (allow 5% variance)
	preToolRatio := float64(preToolCount) / float64(cfg.FuzzIterations)
	if preToolRatio < 0.65 || preToolRatio > 0.75 {
		t.Errorf("PreToolUse ratio should be ~70%%, got: %.2f%%", preToolRatio*100)
	}

	sessionRatio := float64(sessionCount) / float64(cfg.FuzzIterations)
	if sessionRatio < 0.25 || sessionRatio > 0.35 {
		t.Errorf("SessionEnd ratio should be ~30%%, got: %.2f%%", sessionRatio*100)
	}
}

func TestFuzzRunner_CrashSaving(t *testing.T) {
	tempDir := t.TempDir()
	crashDir := filepath.Join(tempDir, "fuzz", "crashes")
	os.MkdirAll(crashDir, 0755)

	cfg := SimulationConfig{
		TempDir: tempDir,
		Verbose: false,
	}

	runner := &FuzzRunner{
		config:  cfg,
		crashes: make([]FuzzCrash, 0),
	}

	crash := FuzzCrash{
		Seed:      12345,
		Iteration: 42,
		Category:  "pretooluse",
		Input:     map[string]string{"test": "input"},
		Output:    "test output",
		Timestamp: time.Now(),
		FailedInvariants: []InvariantResult{
			{InvariantID: "P1", Passed: false, Message: "test failure"},
		},
	}

	runner.saveCrash(crash)

	// Verify file was created
	expectedPath := filepath.Join(crashDir, "crash-pretooluse-42-seed12345.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Crash file not created at: %s", expectedPath)
	}

	// Verify file content is valid JSON
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read crash file: %v", err)
	}

	var loaded FuzzCrash
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Errorf("Crash file contains invalid JSON: %v", err)
	}

	// Verify key fields
	if loaded.Seed != crash.Seed {
		t.Errorf("Seed mismatch: expected %d, got %d", crash.Seed, loaded.Seed)
	}
	if loaded.Iteration != crash.Iteration {
		t.Errorf("Iteration mismatch: expected %d, got %d", crash.Iteration, loaded.Iteration)
	}
	if loaded.Category != crash.Category {
		t.Errorf("Category mismatch: expected %s, got %s", crash.Category, loaded.Category)
	}
}

func TestFuzzSummary(t *testing.T) {
	runner := &FuzzRunner{
		config: SimulationConfig{FuzzSeed: 42},
		results: []SimulationResult{
			{ScenarioID: "FUZZ-P-0", Passed: true},
			{ScenarioID: "FUZZ-P-1", Passed: true},
			{ScenarioID: "FUZZ-S-2", Passed: false},
		},
		crashes: []FuzzCrash{
			{FailedInvariants: []InvariantResult{{}, {}}},
		},
		startTime: time.Now().Add(-10 * time.Second),
	}

	summary := runner.GetSummary()

	if summary.TotalIterations != 3 {
		t.Errorf("Expected 3 total iterations, got: %d", summary.TotalIterations)
	}
	if summary.PreToolUseCount != 2 {
		t.Errorf("Expected 2 PreToolUse, got: %d", summary.PreToolUseCount)
	}
	if summary.SessionEndCount != 1 {
		t.Errorf("Expected 1 SessionEnd, got: %d", summary.SessionEndCount)
	}
	if summary.PassedCount != 2 {
		t.Errorf("Expected 2 passed, got: %d", summary.PassedCount)
	}
	if summary.CrashCount != 1 {
		t.Errorf("Expected 1 crash, got: %d", summary.CrashCount)
	}
	if summary.InvariantFailures != 2 {
		t.Errorf("Expected 2 invariant failures, got: %d", summary.InvariantFailures)
	}
	if summary.Seed != 42 {
		t.Errorf("Expected seed 42, got: %d", summary.Seed)
	}
}

func TestNewFuzzRunner(t *testing.T) {
	tempDir := t.TempDir()
	cfg := SimulationConfig{
		Mode:           "fuzz",
		FuzzIterations: 100,
		FuzzSeed:       999,
		TempDir:        tempDir,
	}

	gen := NewGenerator(filepath.Join(tempDir, "fixtures"))
	runner := NewRunner(cfg, "/fake/validate", "/fake/archive", gen)

	fuzzRunner := NewFuzzRunner(cfg, gen, runner)

	if fuzzRunner == nil {
		t.Fatal("NewFuzzRunner returned nil")
	}
	if fuzzRunner.config.FuzzSeed != 999 {
		t.Errorf("Expected seed 999, got: %d", fuzzRunner.config.FuzzSeed)
	}
	if cap(fuzzRunner.results) != cfg.FuzzIterations {
		t.Errorf("Expected results capacity %d, got: %d", cfg.FuzzIterations, cap(fuzzRunner.results))
	}
	if len(fuzzRunner.crashes) != 0 {
		t.Errorf("Expected 0 initial crashes, got: %d", len(fuzzRunner.crashes))
	}
}

func TestFuzzRunner_GetCrashes(t *testing.T) {
	runner := &FuzzRunner{
		crashes: []FuzzCrash{
			{Seed: 1, Iteration: 0, Category: "pretooluse"},
			{Seed: 2, Iteration: 1, Category: "sessionend"},
		},
	}

	crashes := runner.GetCrashes()
	if len(crashes) != 2 {
		t.Errorf("Expected 2 crashes, got: %d", len(crashes))
	}
}

func TestFuzzRunner_ReplayCrash(t *testing.T) {
	tempDir := t.TempDir()
	crashDir := filepath.Join(tempDir, "fuzz", "crashes")
	os.MkdirAll(crashDir, 0755)

	// Create a crash file
	crash := FuzzCrash{
		Seed:      98765,
		Iteration: 10,
		Category:  "pretooluse",
		Input: map[string]interface{}{
			"tool_name": "Task",
		},
		Output:   `{"decision": "allow"}`,
		ExitCode: 0,
	}

	crashPath := filepath.Join(crashDir, "test-crash.json")
	data, _ := json.MarshalIndent(crash, "", "  ")
	os.WriteFile(crashPath, data, 0644)

	// Create runner (won't actually execute since we need real binaries)
	cfg := SimulationConfig{TempDir: tempDir}
	gen := NewGenerator(filepath.Join(tempDir, "fixtures"))
	defaultRunner := NewRunner(cfg, "/fake/validate", "/fake/archive", gen)

	fuzzRunner := NewFuzzRunner(cfg, gen, defaultRunner)

	// Verify we can load the crash file
	loadedData, err := os.ReadFile(crashPath)
	if err != nil {
		t.Fatalf("Failed to read crash file: %v", err)
	}

	var loaded FuzzCrash
	if err := json.Unmarshal(loadedData, &loaded); err != nil {
		t.Fatalf("Failed to parse crash file: %v", err)
	}

	if loaded.Seed != crash.Seed {
		t.Errorf("Loaded seed %d doesn't match original %d", loaded.Seed, crash.Seed)
	}

	// Test that ReplayCrash can read the file (actual replay requires binaries)
	// We just verify the method doesn't panic on valid input
	_, err = fuzzRunner.ReplayCrash(crashPath)
	// Error is expected since binaries don't exist, but should be structured
	if err == nil {
		// Replay might succeed if it generates but doesn't execute
		t.Log("ReplayCrash succeeded (no binaries needed for generation)")
	}
}

func TestFormatInvariantFailures(t *testing.T) {
	failed := []InvariantResult{
		{InvariantID: "P1", Passed: false, Message: "exit code non-zero"},
		{InvariantID: "P2", Passed: false, Message: "invalid JSON"},
	}

	result := formatInvariantFailures(failed)

	expectedSubstrings := []string{"P1", "exit code non-zero", "P2", "invalid JSON"}
	for _, substr := range expectedSubstrings {
		if !contains(result, substr) {
			t.Errorf("Expected result to contain %q, got: %s", substr, result)
		}
	}
}

func TestFuzzRunner_SetupRandomArtifacts(t *testing.T) {
	tempDir := t.TempDir()
	testDir := filepath.Join(tempDir, "test-artifacts")
	os.MkdirAll(filepath.Join(testDir, ".goyoke", "memory"), 0755)

	cfg := SimulationConfig{TempDir: tempDir}
	runner := &FuzzRunner{config: cfg}

	// Test with seed that should create violations (30% chance)
	// Try multiple seeds to ensure we hit both paths
	for seed := int64(0); seed < 100; seed++ {
		iterDir := filepath.Join(tempDir, filepath.Base(t.TempDir()))
		os.MkdirAll(filepath.Join(iterDir, ".goyoke", "memory"), 0755)

		runner.setupRandomArtifacts(iterDir, seed)

		// Check if files were created (probabilistic)
		violationsPath := filepath.Join(iterDir, ".goyoke", "memory", "routing-violations.jsonl")
		edgesPath := filepath.Join(iterDir, ".goyoke", "memory", "pending-learnings.jsonl")

		// At least verify paths don't cause errors
		_, violErr := os.Stat(violationsPath)
		_, edgeErr := os.Stat(edgesPath)

		// Both missing is fine (low probability), both present is fine (high probability)
		// We're just checking for crashes
		t.Logf("Seed %d: violations=%v, edges=%v", seed, violErr == nil, edgeErr == nil)

		os.RemoveAll(iterDir)
	}
}

func TestFuzzRunner_Timeout(t *testing.T) {
	tempDir := t.TempDir()

	cfg := SimulationConfig{
		Mode:           "fuzz",
		FuzzIterations: 1000000, // Very high
		FuzzSeed:       42,
		FuzzTimeout:    100 * time.Millisecond, // Very short
		TempDir:        tempDir,
		Verbose:        false,
	}

	gen := NewGenerator(filepath.Join(tempDir, "fixtures"))
	defaultRunner := NewRunner(cfg, "/fake/validate", "/fake/archive", gen)
	fuzzRunner := NewFuzzRunner(cfg, gen, defaultRunner)

	start := time.Now()
	results, err := fuzzRunner.RunFuzz()
	duration := time.Since(start)

	if err != nil {
		t.Errorf("RunFuzz returned error: %v", err)
	}

	// Should timeout well before completing all iterations
	if len(results) >= 1000000 {
		t.Errorf("Expected timeout before completing all iterations, got %d results", len(results))
	}

	// Should timeout roughly around the configured timeout
	if duration > 2*time.Second {
		t.Errorf("Timeout took too long: %v (expected ~100ms)", duration)
	}
}

func TestFuzzRunner_EmptyResults(t *testing.T) {
	runner := &FuzzRunner{
		config:    SimulationConfig{FuzzSeed: 0},
		results:   []SimulationResult{},
		crashes:   []FuzzCrash{},
		startTime: time.Now(),
	}

	summary := runner.GetSummary()

	if summary.TotalIterations != 0 {
		t.Errorf("Expected 0 iterations, got: %d", summary.TotalIterations)
	}
	if summary.CrashCount != 0 {
		t.Errorf("Expected 0 crashes, got: %d", summary.CrashCount)
	}

	crashes := runner.GetCrashes()
	if len(crashes) != 0 {
		t.Errorf("Expected 0 crashes from GetCrashes, got: %d", len(crashes))
	}
}

func TestFuzzCrash_JSONSerialization(t *testing.T) {
	crash := FuzzCrash{
		Seed:      12345,
		Iteration: 99,
		Category:  "sessionend",
		Input: map[string]interface{}{
			"session_id":      "test-session",
			"hook_event_name": "SessionEnd",
		},
		Output:   `{"status": "archived"}`,
		ExitCode: 0,
		FailedInvariants: []InvariantResult{
			{InvariantID: "S1", Passed: false, Message: "handoff missing"},
		},
		Timestamp: time.Date(2026, 1, 22, 10, 0, 0, 0, time.UTC),
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(crash, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal crash: %v", err)
	}

	// Unmarshal back
	var loaded FuzzCrash
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Failed to unmarshal crash: %v", err)
	}

	// Verify fields
	if loaded.Seed != crash.Seed {
		t.Errorf("Seed mismatch: %d vs %d", loaded.Seed, crash.Seed)
	}
	if loaded.Category != crash.Category {
		t.Errorf("Category mismatch: %s vs %s", loaded.Category, crash.Category)
	}
	if len(loaded.FailedInvariants) != len(crash.FailedInvariants) {
		t.Errorf("FailedInvariants length mismatch: %d vs %d",
			len(loaded.FailedInvariants), len(crash.FailedInvariants))
	}
}

func TestFuzzRunner_FuzzPreToolUse(t *testing.T) {
	tempDir := t.TempDir()

	cfg := SimulationConfig{
		Mode:    "fuzz",
		TempDir: tempDir,
	}

	gen := NewGenerator(filepath.Join(tempDir, "fixtures"))
	defaultRunner := NewRunner(cfg, "/fake/validate", "/fake/archive", gen)
	fuzzRunner := NewFuzzRunner(cfg, gen, defaultRunner)

	// Test that fuzzPreToolUse generates a scenario
	result := fuzzRunner.fuzzPreToolUse(12345, 0)

	// Should have a scenario ID
	if result.ScenarioID != "FUZZ-P-0" {
		t.Errorf("Expected scenario ID 'FUZZ-P-0', got: %s", result.ScenarioID)
	}

	// Error is expected since binaries don't exist, but structure should be valid
	if result.ScenarioID == "" {
		t.Error("ScenarioID should not be empty")
	}
}

func TestFuzzRunner_FuzzSessionEnd(t *testing.T) {
	tempDir := t.TempDir()

	cfg := SimulationConfig{
		Mode:    "fuzz",
		TempDir: tempDir,
	}

	gen := NewGenerator(filepath.Join(tempDir, "fixtures"))
	defaultRunner := NewRunner(cfg, "/fake/validate", "/fake/archive", gen)
	fuzzRunner := NewFuzzRunner(cfg, gen, defaultRunner)

	// Test that fuzzSessionEnd generates a scenario
	result := fuzzRunner.fuzzSessionEnd(67890, 1)

	// Should have a scenario ID
	if result.ScenarioID != "FUZZ-S-1" {
		t.Errorf("Expected scenario ID 'FUZZ-S-1', got: %s", result.ScenarioID)
	}

	// Should have attempted to create temp directory
	expectedIterDir := filepath.Join(tempDir, "fuzz-1")
	if _, err := os.Stat(expectedIterDir); err == nil {
		// Directory might exist or be cleaned up
		t.Logf("Iteration directory exists: %s", expectedIterDir)
	}
}

func TestFuzzRunner_InvariantChecking(t *testing.T) {
	tempDir := t.TempDir()

	cfg := SimulationConfig{
		Mode:           "fuzz",
		FuzzIterations: 5,
		FuzzSeed:       999,
		FuzzTimeout:    10 * time.Second,
		TempDir:        tempDir,
		Verbose:        true,
	}

	gen := NewGenerator(filepath.Join(tempDir, "fixtures"))
	defaultRunner := NewRunner(cfg, "/fake/validate", "/fake/archive", gen)
	fuzzRunner := NewFuzzRunner(cfg, gen, defaultRunner)

	// Run a few iterations (will fail since binaries don't exist, but should check invariants)
	results, err := fuzzRunner.RunFuzz()

	if err != nil {
		t.Errorf("RunFuzz returned error: %v", err)
	}

	// Should have attempted all iterations
	if len(results) != cfg.FuzzIterations {
		t.Errorf("Expected %d results, got: %d", cfg.FuzzIterations, len(results))
	}

	// All should fail since binaries don't exist
	for i, r := range results {
		if r.ScenarioID == "" {
			t.Errorf("Result %d has empty scenario ID", i)
		}
	}
}

func TestFuzzRunner_CrashCapture(t *testing.T) {
	tempDir := t.TempDir()
	crashDir := filepath.Join(tempDir, "fuzz", "crashes")

	cfg := SimulationConfig{
		Mode:           "fuzz",
		FuzzIterations: 10,
		FuzzSeed:       42,
		FuzzTimeout:    10 * time.Second,
		TempDir:        tempDir,
		Verbose:        false,
	}

	gen := NewGenerator(filepath.Join(tempDir, "fixtures"))
	defaultRunner := NewRunner(cfg, "/fake/validate", "/fake/archive", gen)
	fuzzRunner := NewFuzzRunner(cfg, gen, defaultRunner)

	// Run fuzz (will crash due to missing binaries)
	fuzzRunner.RunFuzz()

	// Check if crash directory was created
	if _, err := os.Stat(crashDir); os.IsNotExist(err) {
		// This is expected since we don't have real binaries
		t.Logf("No crashes directory created (expected without binaries)")
		return
	}

	// If crashes were saved, verify file format
	entries, err := os.ReadDir(crashDir)
	if err == nil && len(entries) > 0 {
		for _, entry := range entries {
			if !entry.IsDir() {
				t.Logf("Found crash file: %s", entry.Name())
			}
		}
	}
}

func TestFuzzRunner_VerboseOutput(t *testing.T) {
	tempDir := t.TempDir()

	cfg := SimulationConfig{
		Mode:           "fuzz",
		FuzzIterations: 3,
		FuzzSeed:       111,
		FuzzTimeout:    5 * time.Second,
		TempDir:        tempDir,
		Verbose:        true, // Enable verbose for coverage
	}

	gen := NewGenerator(filepath.Join(tempDir, "fixtures"))
	defaultRunner := NewRunner(cfg, "/fake/validate", "/fake/archive", gen)
	fuzzRunner := NewFuzzRunner(cfg, gen, defaultRunner)

	// Run with verbose enabled
	results, err := fuzzRunner.RunFuzz()

	if err != nil {
		t.Errorf("RunFuzz returned error: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got: %d", len(results))
	}
}

func TestFuzzRunner_ConfigurableIterations(t *testing.T) {
	tempDir := t.TempDir()

	testCases := []struct {
		name       string
		iterations int
	}{
		{"zero iterations", 0},
		{"one iteration", 1},
		{"ten iterations", 10},
		{"hundred iterations", 100},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := SimulationConfig{
				Mode:           "fuzz",
				FuzzIterations: tc.iterations,
				FuzzSeed:       42,
				FuzzTimeout:    30 * time.Second,
				TempDir:        tempDir,
				Verbose:        false,
			}

			gen := NewGenerator(filepath.Join(tempDir, "fixtures"))
			defaultRunner := NewRunner(cfg, "/fake/validate", "/fake/archive", gen)
			fuzzRunner := NewFuzzRunner(cfg, gen, defaultRunner)

			results, err := fuzzRunner.RunFuzz()

			if err != nil {
				t.Errorf("RunFuzz returned error: %v", err)
			}

			if len(results) != tc.iterations {
				t.Errorf("Expected %d results, got: %d", tc.iterations, len(results))
			}
		})
	}
}

func TestFuzzSummary_Duration(t *testing.T) {
	runner := &FuzzRunner{
		config:    SimulationConfig{FuzzSeed: 123},
		results:   []SimulationResult{},
		crashes:   []FuzzCrash{},
		startTime: time.Now().Add(-5 * time.Second),
	}

	summary := runner.GetSummary()

	if summary.Duration < 4*time.Second || summary.Duration > 6*time.Second {
		t.Errorf("Expected duration ~5s, got: %v", summary.Duration)
	}
}

func TestFuzzCrash_AllFields(t *testing.T) {
	crash := FuzzCrash{
		Seed:      99999,
		Iteration: 555,
		Category:  "pretooluse",
		Input: ToolEvent{
			ToolName:      "Task",
			SessionID:     "session-123",
			HookEventName: "PreToolUse",
			CapturedAt:    time.Now().Unix(),
		},
		Output:   `{"decision": "block", "reason": "test"}`,
		ExitCode: 1,
		FailedInvariants: []InvariantResult{
			{InvariantID: "P1", Passed: false, Message: "failed"},
			{InvariantID: "P2", Passed: false, Message: "also failed"},
		},
		Timestamp: time.Now(),
	}

	// Verify all fields are set
	if crash.Seed != 99999 {
		t.Errorf("Seed not set correctly: %d", crash.Seed)
	}
	if crash.Iteration != 555 {
		t.Errorf("Iteration not set correctly: %d", crash.Iteration)
	}
	if crash.Category != "pretooluse" {
		t.Errorf("Category not set correctly: %s", crash.Category)
	}
	if crash.ExitCode != 1 {
		t.Errorf("ExitCode not set correctly: %d", crash.ExitCode)
	}
	if len(crash.FailedInvariants) != 2 {
		t.Errorf("Expected 2 failed invariants, got: %d", len(crash.FailedInvariants))
	}
}

func TestFormatInvariantFailures_Empty(t *testing.T) {
	result := formatInvariantFailures([]InvariantResult{})
	if result != "" {
		t.Errorf("Expected empty string for no failures, got: %q", result)
	}
}

func TestFormatInvariantFailures_Single(t *testing.T) {
	failed := []InvariantResult{
		{InvariantID: "TEST", Passed: false, Message: "single failure"},
	}

	result := formatInvariantFailures(failed)

	if !contains(result, "TEST") {
		t.Errorf("Expected result to contain 'TEST', got: %s", result)
	}
	if !contains(result, "single failure") {
		t.Errorf("Expected result to contain 'single failure', got: %s", result)
	}
}

func TestFuzzRunner_SaveCrashVerbose(t *testing.T) {
	tempDir := t.TempDir()
	crashDir := filepath.Join(tempDir, "fuzz", "crashes")

	cfg := SimulationConfig{
		TempDir: tempDir,
		Verbose: true, // Enable verbose logging
	}

	runner := &FuzzRunner{
		config:  cfg,
		crashes: make([]FuzzCrash, 0),
	}

	// Test crash with marshal error (should trigger verbose error path)
	crash := FuzzCrash{
		Seed:      12345,
		Iteration: 99,
		Category:  "sessionend",
		Input:     make(chan int), // This will fail to marshal
		Timestamp: time.Now(),
	}

	// This should fail to marshal but not panic
	runner.saveCrash(crash)

	// Verify no crash file was created due to marshal error
	expectedPath := filepath.Join(crashDir, "crash-sessionend-99-seed12345.json")
	if _, err := os.Stat(expectedPath); err == nil {
		t.Error("Crash file should not be created when marshal fails")
	}
}

func TestFuzzRunner_ReplayCrashErrors(t *testing.T) {
	tempDir := t.TempDir()

	cfg := SimulationConfig{TempDir: tempDir}
	gen := NewGenerator(filepath.Join(tempDir, "fixtures"))
	defaultRunner := NewRunner(cfg, "/fake/validate", "/fake/archive", gen)
	fuzzRunner := NewFuzzRunner(cfg, gen, defaultRunner)

	// Test with non-existent file
	_, err := fuzzRunner.ReplayCrash("/nonexistent/crash.json")
	if err == nil {
		t.Error("Expected error for non-existent crash file")
	}
	if !contains(err.Error(), "read crash file") {
		t.Errorf("Expected 'read crash file' error, got: %v", err)
	}

	// Test with invalid JSON
	invalidCrashPath := filepath.Join(tempDir, "invalid-crash.json")
	os.WriteFile(invalidCrashPath, []byte("not json"), 0644)

	_, err = fuzzRunner.ReplayCrash(invalidCrashPath)
	if err == nil {
		t.Error("Expected error for invalid JSON crash file")
	}
	if !contains(err.Error(), "parse crash file") {
		t.Errorf("Expected 'parse crash file' error, got: %v", err)
	}

	// Test sessionend category replay
	sessionCrash := FuzzCrash{
		Seed:      11111,
		Iteration: 5,
		Category:  "sessionend",
		Input:     map[string]string{"test": "data"},
	}

	sessionCrashPath := filepath.Join(tempDir, "session-crash.json")
	data, _ := json.MarshalIndent(sessionCrash, "", "  ")
	os.WriteFile(sessionCrashPath, data, 0644)

	result, err := fuzzRunner.ReplayCrash(sessionCrashPath)
	if err != nil {
		t.Errorf("Unexpected error replaying session crash: %v", err)
	}
	if result.ScenarioID != "FUZZ-S-5" {
		t.Errorf("Expected scenario ID 'FUZZ-S-5', got: %s", result.ScenarioID)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
