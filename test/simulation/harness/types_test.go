package harness

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSimulationConfig_JSONRoundtrip(t *testing.T) {
	cfg := SimulationConfig{
		Mode:           "fuzz",
		ScenarioFilter: []string{"V001", "V002"},
		FuzzIterations: 500,
		FuzzSeed:       12345,
		FuzzTimeout:    10 * time.Minute,
		SchemaPath:     "/path/to/schema.json",
		ReportFormat:   "markdown",
		Verbose:        true,
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded SimulationConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Mode != cfg.Mode {
		t.Errorf("Mode mismatch: got %s, want %s", decoded.Mode, cfg.Mode)
	}
	if decoded.FuzzIterations != cfg.FuzzIterations {
		t.Errorf("FuzzIterations mismatch: got %d, want %d", decoded.FuzzIterations, cfg.FuzzIterations)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Mode != "deterministic" {
		t.Errorf("Expected default mode 'deterministic', got: %s", cfg.Mode)
	}
	if cfg.FuzzIterations != 1000 {
		t.Errorf("Expected default iterations 1000, got: %d", cfg.FuzzIterations)
	}
	if cfg.FuzzTimeout != 5*time.Minute {
		t.Errorf("Expected default timeout 5m, got: %v", cfg.FuzzTimeout)
	}
}

func TestExpectedOutput_JSONRoundtrip(t *testing.T) {
	decision := "block"
	expected := ExpectedOutput{
		Decision:      &decision,
		ReasonPattern: "opus.*blocked",
		FilesCreated:  []string{".claude/memory/handoffs.jsonl"},
		ExitCode:      0,
	}

	data, err := json.Marshal(expected)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded ExpectedOutput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if *decoded.Decision != "block" {
		t.Errorf("Decision mismatch: got %s, want block", *decoded.Decision)
	}
}

func TestSimulationResult_ErrorHandling(t *testing.T) {
	result := SimulationResult{
		ScenarioID: "V001",
		Passed:     false,
		ErrorMsg:   "timeout waiting for process",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	if string(data) == "" {
		t.Error("Expected non-empty JSON")
	}
}
