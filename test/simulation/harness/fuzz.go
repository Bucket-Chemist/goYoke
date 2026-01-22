package harness

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

// FuzzRunner executes randomized testing with invariant checking.
type FuzzRunner struct {
	config    SimulationConfig
	gen       Generator
	runner    *DefaultRunner
	results   []SimulationResult
	crashes   []FuzzCrash
	startTime time.Time
}

// FuzzCrash captures a failed fuzz iteration for replay.
type FuzzCrash struct {
	Seed             int64               `json:"seed"`
	Iteration        int                 `json:"iteration"`
	Category         string              `json:"category"` // "pretooluse" or "sessionend"
	Input            interface{}         `json:"input"`
	Output           string              `json:"output"`
	ExitCode         int                 `json:"exit_code"`
	FailedInvariants []InvariantResult   `json:"failed_invariants"`
	Timestamp        time.Time           `json:"timestamp"`
}

// FuzzSummary provides aggregate statistics for a fuzz run.
type FuzzSummary struct {
	TotalIterations   int           `json:"total_iterations"`
	PreToolUseCount   int           `json:"pretooluse_count"`
	SessionEndCount   int           `json:"sessionend_count"`
	PassedCount       int           `json:"passed_count"`
	CrashCount        int           `json:"crash_count"`
	InvariantFailures int           `json:"invariant_failures"`
	Duration          time.Duration `json:"duration"`
	Seed              int64         `json:"seed"`
}

// NewFuzzRunner creates a fuzz runner with the given configuration.
func NewFuzzRunner(cfg SimulationConfig, gen Generator, runner *DefaultRunner) *FuzzRunner {
	return &FuzzRunner{
		config:  cfg,
		gen:     gen,
		runner:  runner,
		results: make([]SimulationResult, 0, cfg.FuzzIterations),
		crashes: make([]FuzzCrash, 0),
	}
}

// RunFuzz executes the configured number of fuzz iterations.
func (f *FuzzRunner) RunFuzz() ([]SimulationResult, error) {
	f.startTime = time.Now()
	rng := rand.New(rand.NewSource(f.config.FuzzSeed))

	timeout := time.After(f.config.FuzzTimeout)

	for i := 0; i < f.config.FuzzIterations; i++ {
		select {
		case <-timeout:
			if f.config.Verbose {
				fmt.Printf("[FUZZ] Timeout after %d iterations\n", i)
			}
			return f.results, nil
		default:
		}

		seed := rng.Int63()

		// 70/30 split between PreToolUse and SessionEnd
		var result SimulationResult
		if rng.Float64() < 0.7 {
			result = f.fuzzPreToolUse(seed, i)
		} else {
			result = f.fuzzSessionEnd(seed, i)
		}

		f.results = append(f.results, result)

		if !result.Passed {
			if f.config.Verbose {
				fmt.Printf("[FUZZ] CRASH at iteration %d (seed: %d)\n", i, seed)
			}
		}
	}

	return f.results, nil
}

// fuzzPreToolUse runs a single PreToolUse fuzz iteration.
func (f *FuzzRunner) fuzzPreToolUse(seed int64, iteration int) SimulationResult {
	event := f.gen.RandomToolEvent(seed)

	scenario := Scenario{
		ID:       fmt.Sprintf("FUZZ-P-%d", iteration),
		Category: "pretooluse",
		Input:    event,
		Expected: ExpectedOutput{ExitCode: 0}, // Invariants will check specifics
	}

	result := f.runner.RunScenario(scenario)

	// Check invariants
	invariantResults := CheckInvariants(PreToolUseInvariants, event, result.Output, 0, f.config.TempDir)

	if !AllPassed(invariantResults) {
		result.Passed = false
		failed := FailedInvariants(invariantResults)
		result.Diff = formatInvariantFailures(failed)

		crash := FuzzCrash{
			Seed:             seed,
			Iteration:        iteration,
			Category:         "pretooluse",
			Input:            event,
			Output:           result.Output,
			ExitCode:         0,
			FailedInvariants: failed,
			Timestamp:        time.Now(),
		}
		f.crashes = append(f.crashes, crash)
		f.saveCrash(crash)
	}

	return result
}

// fuzzSessionEnd runs a single SessionEnd fuzz iteration.
func (f *FuzzRunner) fuzzSessionEnd(seed int64, iteration int) SimulationResult {
	event := f.gen.RandomSessionEvent(seed)

	// Create temp directory for this iteration
	iterDir := filepath.Join(f.config.TempDir, fmt.Sprintf("fuzz-%d", iteration))
	os.MkdirAll(filepath.Join(iterDir, ".claude", "memory"), 0755)

	// Set up random artifacts based on seed
	f.setupRandomArtifacts(iterDir, seed)

	scenario := Scenario{
		ID:       fmt.Sprintf("FUZZ-S-%d", iteration),
		Category: "sessionend",
		Input:    event,
		Expected: ExpectedOutput{ExitCode: 0},
		Setup: func(cfg SimulationConfig) error {
			cfg.TempDir = iterDir
			return nil
		},
		// Note: No Teardown here - cleanup happens after invariant check
	}

	// Temporarily update config for this iteration (both FuzzRunner and Runner)
	origTempDir := f.config.TempDir
	f.config.TempDir = iterDir
	f.runner.SetTempDir(iterDir)

	result := f.runner.RunScenario(scenario)

	// Check invariants (BEFORE cleanup so files still exist)
	invariantResults := CheckInvariants(SessionEndInvariants, event, result.Output, 0, iterDir)

	if !AllPassed(invariantResults) {
		result.Passed = false
		failed := FailedInvariants(invariantResults)
		result.Diff = formatInvariantFailures(failed)

		crash := FuzzCrash{
			Seed:             seed,
			Iteration:        iteration,
			Category:         "sessionend",
			Input:            event,
			Output:           result.Output,
			ExitCode:         0,
			FailedInvariants: failed,
			Timestamp:        time.Now(),
		}
		f.crashes = append(f.crashes, crash)
		f.saveCrash(crash)
	} else {
		// Only cleanup passing iterations (keep failed ones for debugging)
		os.RemoveAll(iterDir)
	}

	f.config.TempDir = origTempDir
	f.runner.SetTempDir(origTempDir)

	return result
}

// setupRandomArtifacts creates random session artifacts for testing.
func (f *FuzzRunner) setupRandomArtifacts(dir string, seed int64) {
	rng := rand.New(rand.NewSource(seed))

	// Random violations
	if rng.Float64() < 0.3 {
		violationCount := rng.Intn(5) + 1
		var violations string
		for i := 0; i < violationCount; i++ {
			violations += fmt.Sprintf(`{"timestamp":"2026-01-22T10:00:00Z","agent":"test-%d","violation":"test"}`+"\n", i)
		}
		os.WriteFile(filepath.Join(dir, ".claude", "memory", "routing-violations.jsonl"), []byte(violations), 0644)
	}

	// Random sharp edges
	if rng.Float64() < 0.2 {
		edgeCount := rng.Intn(3) + 1
		var edges string
		for i := 0; i < edgeCount; i++ {
			edges += fmt.Sprintf(`{"type":"sharp_edge","file":"test-%d.go","pattern":"test","timestamp":"2026-01-22T10:00:00Z"}`+"\n", i)
		}
		os.WriteFile(filepath.Join(dir, ".claude", "memory", "pending-learnings.jsonl"), []byte(edges), 0644)
	}
}

// saveCrash writes a crash to the crashes directory for replay.
func (f *FuzzRunner) saveCrash(crash FuzzCrash) {
	crashDir := filepath.Join(f.config.TempDir, "fuzz", "crashes")
	os.MkdirAll(crashDir, 0755)

	filename := fmt.Sprintf("crash-%s-%d-seed%d.json", crash.Category, crash.Iteration, crash.Seed)
	path := filepath.Join(crashDir, filename)

	data, err := json.MarshalIndent(crash, "", "  ")
	if err != nil {
		if f.config.Verbose {
			fmt.Printf("[FUZZ] Failed to marshal crash: %v\n", err)
		}
		return
	}

	os.WriteFile(path, data, 0644)
}

// GetSummary returns aggregate statistics for the fuzz run.
func (f *FuzzRunner) GetSummary() FuzzSummary {
	summary := FuzzSummary{
		TotalIterations: len(f.results),
		Seed:            f.config.FuzzSeed,
		Duration:        time.Since(f.startTime),
	}

	for _, r := range f.results {
		if len(r.ScenarioID) >= 6 && r.ScenarioID[:6] == "FUZZ-P" {
			summary.PreToolUseCount++
		} else {
			summary.SessionEndCount++
		}

		if r.Passed {
			summary.PassedCount++
		}
	}

	summary.CrashCount = len(f.crashes)

	for _, c := range f.crashes {
		summary.InvariantFailures += len(c.FailedInvariants)
	}

	return summary
}

// GetCrashes returns all captured crashes.
func (f *FuzzRunner) GetCrashes() []FuzzCrash {
	return f.crashes
}

// ReplayCrash re-executes a specific crash file.
func (f *FuzzRunner) ReplayCrash(crashPath string) (SimulationResult, error) {
	data, err := os.ReadFile(crashPath)
	if err != nil {
		return SimulationResult{}, fmt.Errorf("read crash file: %w", err)
	}

	var crash FuzzCrash
	if err := json.Unmarshal(data, &crash); err != nil {
		return SimulationResult{}, fmt.Errorf("parse crash file: %w", err)
	}

	var result SimulationResult
	if crash.Category == "pretooluse" {
		result = f.fuzzPreToolUse(crash.Seed, crash.Iteration)
	} else {
		result = f.fuzzSessionEnd(crash.Seed, crash.Iteration)
	}

	return result, nil
}

// formatInvariantFailures creates a human-readable failure summary.
func formatInvariantFailures(failed []InvariantResult) string {
	var output string
	for _, f := range failed {
		output += fmt.Sprintf("[%s] %s\n", f.InvariantID, f.Message)
	}
	return output
}
