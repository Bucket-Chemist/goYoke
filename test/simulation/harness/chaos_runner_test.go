package harness

import (
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultChaosConfig(t *testing.T) {
	cfg := DefaultChaosConfig()

	if cfg.NumAgents != 10 {
		t.Errorf("NumAgents: got %d, want 10", cfg.NumAgents)
	}
	if cfg.FailuresPerAgent != 20 {
		t.Errorf("FailuresPerAgent: got %d, want 20", cfg.FailuresPerAgent)
	}
	if cfg.SharedKeyRatio != 0.3 {
		t.Errorf("SharedKeyRatio: got %f, want 0.3", cfg.SharedKeyRatio)
	}
	if cfg.MaxFailures != 3 {
		t.Errorf("MaxFailures: got %d, want 3", cfg.MaxFailures)
	}
}

func TestChaosRunner_GenerateAssignments(t *testing.T) {
	tests := []struct {
		name        string
		numAgents   int
		sharedRatio float64
		wantShared  int
		wantUnique  int
	}{
		{
			name:        "10 agents 30% shared",
			numAgents:   10,
			sharedRatio: 0.3,
			wantShared:  3,
			wantUnique:  7,
		},
		{
			name:        "10 agents 0% shared",
			numAgents:   10,
			sharedRatio: 0.0,
			wantShared:  0,
			wantUnique:  10,
		},
		{
			name:        "10 agents 100% shared",
			numAgents:   10,
			sharedRatio: 1.0,
			wantShared:  10,
			wantUnique:  0,
		},
		{
			name:        "5 agents 50% shared",
			numAgents:   5,
			sharedRatio: 0.5,
			wantShared:  2, // min 2 for sharing
			wantUnique:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ChaosConfig{
				NumAgents:      tt.numAgents,
				SharedKeyRatio: tt.sharedRatio,
			}
			runner := &ChaosRunner{config: cfg}
			rng := rand.New(rand.NewSource(42))

			assignments := runner.generateAssignments(rng)

			if len(assignments) != tt.numAgents {
				t.Errorf("Assignment count: got %d, want %d", len(assignments), tt.numAgents)
			}

			// Count shared vs unique
			shared, unique := 0, 0
			for _, a := range assignments {
				if a.IsShared {
					shared++
				} else {
					unique++
				}
			}

			if shared != tt.wantShared {
				t.Errorf("Shared count: got %d, want %d", shared, tt.wantShared)
			}
			if unique != tt.wantUnique {
				t.Errorf("Unique count: got %d, want %d", unique, tt.wantUnique)
			}

			// Verify shared agents use same key
			var sharedKey string
			for _, a := range assignments {
				if a.IsShared {
					key := a.File + ":" + a.ErrorType
					if sharedKey == "" {
						sharedKey = key
					} else if key != sharedKey {
						t.Errorf("Shared agents use different keys: %s vs %s", key, sharedKey)
					}
				}
			}
		})
	}
}

func TestChaosRunner_GenerateAssignments_Deterministic(t *testing.T) {
	cfg := ChaosConfig{
		NumAgents:      10,
		SharedKeyRatio: 0.5,
	}
	runner := &ChaosRunner{config: cfg}

	// Same seed should produce same assignments
	rng1 := rand.New(rand.NewSource(12345))
	rng2 := rand.New(rand.NewSource(12345))

	a1 := runner.generateAssignments(rng1)
	a2 := runner.generateAssignments(rng2)

	for i := range a1 {
		if a1[i].File != a2[i].File || a1[i].ErrorType != a2[i].ErrorType || a1[i].IsShared != a2[i].IsShared {
			t.Errorf("Assignment %d differs with same seed", i)
		}
	}
}

func TestChaosRunner_SetupTempDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "chaos-setup-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	runner := &ChaosRunner{
		tempDir: tmpDir,
	}

	if err := runner.setupTempDir(); err != nil {
		t.Fatalf("setupTempDir failed: %v", err)
	}

	// Check directories exist
	expectedDirs := []string{
		filepath.Join(tmpDir, ".gogent", "memory"),
		filepath.Join(tmpDir, ".gogent"),
	}

	for _, dir := range expectedDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Directory not created: %s", dir)
		}
	}
}

func TestChaosRunner_BuildEnv(t *testing.T) {
	runner := &ChaosRunner{
		config: ChaosConfig{
			MaxFailures: 5,
			SchemaPath:  "/path/to/schema.json",
			AgentsPath:  "/path/to/agents.json",
		},
		tempDir: "/tmp/chaos",
	}

	env := runner.buildEnv()

	// Convert to map for easy checking
	envMap := make(map[string]string)
	for _, e := range env {
		for i := 0; i < len(e); i++ {
			if e[i] == '=' {
				envMap[e[:i]] = e[i+1:]
				break
			}
		}
	}

	if envMap["GOGENT_PROJECT_DIR"] != "/tmp/chaos" {
		t.Errorf("GOGENT_PROJECT_DIR: got %q", envMap["GOGENT_PROJECT_DIR"])
	}
	if envMap["GOGENT_MAX_FAILURES"] != "5" {
		t.Errorf("GOGENT_MAX_FAILURES: got %q, want %q", envMap["GOGENT_MAX_FAILURES"], "5")
	}
	if envMap["GOGENT_ROUTING_SCHEMA"] != "/path/to/schema.json" {
		t.Errorf("GOGENT_ROUTING_SCHEMA: got %q", envMap["GOGENT_ROUTING_SCHEMA"])
	}
}

func TestContainsString(t *testing.T) {
	slice := []string{"apple", "banana", "cherry"}

	if !containsString(slice, "banana") {
		t.Error("Expected true for 'banana'")
	}
	if containsString(slice, "grape") {
		t.Error("Expected false for 'grape'")
	}
	if containsString(nil, "anything") {
		t.Error("Expected false for nil slice")
	}
}

func TestChaosReport_PassedWhenNoErrors(t *testing.T) {
	report := ChaosReport{
		TotalAgents: 10,
		TotalEvents: 200,
		Errors:      nil,
	}

	// When building report, Passed should be true if no errors
	// This tests the logic we use in buildReport
	report.Passed = len(report.Errors) == 0

	if !report.Passed {
		t.Error("Expected Passed=true when no errors")
	}
}

func TestChaosReport_FailedWhenErrors(t *testing.T) {
	report := ChaosReport{
		TotalAgents: 10,
		TotalEvents: 200,
		Errors:      []string{"something went wrong"},
	}

	report.Passed = len(report.Errors) == 0

	if report.Passed {
		t.Error("Expected Passed=false when errors exist")
	}
}

func TestJSONLCheckResult(t *testing.T) {
	result := JSONLCheckResult{
		FilesChecked:   2,
		TotalLines:     100,
		InvalidLines:   0,
		CorruptedFiles: nil,
	}

	if result.InvalidLines != 0 {
		t.Errorf("InvalidLines: got %d, want 0", result.InvalidLines)
	}

	// With corruption
	result.InvalidLines = 5
	result.CorruptedFiles = []string{"file1.jsonl"}

	if len(result.CorruptedFiles) != 1 {
		t.Errorf("CorruptedFiles count: got %d, want 1", len(result.CorruptedFiles))
	}
}

func TestChaosRunner_ValidateJSONL_EmptyFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "chaos-jsonl-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	runner := &ChaosRunner{tempDir: tmpDir}
	runner.setupTempDir()

	// Empty files (not created yet)
	result, err := runner.validateJSONL()
	if err != nil {
		t.Fatalf("validateJSONL failed: %v", err)
	}

	if result.FilesChecked != 0 {
		t.Errorf("FilesChecked: got %d, want 0 (files don't exist)", result.FilesChecked)
	}
}

func TestChaosRunner_ValidateJSONL_ValidFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "chaos-jsonl-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	runner := &ChaosRunner{tempDir: tmpDir}
	runner.setupTempDir()

	// Create valid JSONL
	os.WriteFile(filepath.Join(tmpDir, ".gogent", "failure-tracker.jsonl"),
		[]byte(`{"key":"value"}`+"\n"+`{"key2":"value2"}`+"\n"), 0644)

	result, err := runner.validateJSONL()
	if err != nil {
		t.Fatalf("validateJSONL failed: %v", err)
	}

	if result.FilesChecked != 1 {
		t.Errorf("FilesChecked: got %d, want 1", result.FilesChecked)
	}
	if result.TotalLines != 2 {
		t.Errorf("TotalLines: got %d, want 2", result.TotalLines)
	}
	if result.InvalidLines != 0 {
		t.Errorf("InvalidLines: got %d, want 0", result.InvalidLines)
	}
}

func TestChaosRunner_ValidateJSONL_InvalidFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "chaos-jsonl-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	runner := &ChaosRunner{tempDir: tmpDir}
	runner.setupTempDir()

	// Create JSONL with invalid line
	os.WriteFile(filepath.Join(tmpDir, ".gogent", "failure-tracker.jsonl"),
		[]byte(`{"valid":"json"}`+"\n"+`{not valid json}`+"\n"), 0644)

	result, err := runner.validateJSONL()

	// Should return error for corruption
	if err == nil {
		t.Error("Expected error for corrupted JSONL")
	}

	if result.InvalidLines != 1 {
		t.Errorf("InvalidLines: got %d, want 1", result.InvalidLines)
	}
}

func TestKeyAssignment(t *testing.T) {
	ka := KeyAssignment{
		File:      "test.go",
		ErrorType: "TestError",
		IsShared:  true,
	}

	if ka.File != "test.go" {
		t.Errorf("File: got %q", ka.File)
	}
	if !ka.IsShared {
		t.Error("Expected IsShared=true")
	}
}

func TestChaosResult(t *testing.T) {
	result := ChaosResult{
		AgentID:     1,
		File:        "test.go",
		ErrorType:   "Error",
		IsShared:    true,
		FinalCount:  5,
		BlockedAt:   3,
		WriteErrors: 0,
		TotalCalls:  20,
	}

	if result.BlockedAt != 3 {
		t.Errorf("BlockedAt: got %d, want 3", result.BlockedAt)
	}
	if result.TotalCalls != 20 {
		t.Errorf("TotalCalls: got %d, want 20", result.TotalCalls)
	}
}
