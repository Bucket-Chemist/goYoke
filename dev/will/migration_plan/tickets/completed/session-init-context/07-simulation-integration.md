# Session Init Context: Simulation Harness Integration

**File**: `07-simulation-integration.md`
**Tickets**: GOgent-067 to 072 (6 tickets)
**Phase**: Week 4 (Extension)
**Status**: Ready for Implementation
**Last Updated**: 2026-01-23
**Depends On**: 06-week4-load-routing-context-v2.md (GOgent-056 to 066)

---

## Navigation

- **Previous**: [06-week4-load-routing-context-v2.md](06-week4-load-routing-context-v2.md) - GOgent-056 to 066
- **Overview**: [00-overview.md](00-overview.md) - Testing strategy, rollback plan, standards

---

## Executive Summary

This document extends the simulation harness to cover the new `gogent-load-context` binary (SessionStart hook). The existing harness tests PreToolUse, PostToolUse, and SessionEnd hooks. We add SessionStart testing to complete the hook lifecycle coverage.

### Integration Strategy

| Level | What It Tests | New Coverage |
|-------|---------------|--------------|
| L1 - Unit Invariants | Single hook execution | SessionStart deterministic scenarios |
| L2 - Session Replay | Multi-turn sequences | SessionStart → PreToolUse transitions |
| L3 - Behavioral Properties | System invariants | Context injection correctness |
| L4 - Chaos Testing | Concurrent access | Parallel session initialization |

### New Components

```
test/simulation/
├── fixtures/
│   ├── deterministic/
│   │   └── sessionstart/          ← NEW: SessionStart test cases
│   │       ├── S001_startup.json
│   │       ├── S002_resume.json
│   │       └── ...
│   └── sessions/
│       └── session-init-flow.jsonl ← NEW: SessionStart in replay sequences
└── harness/
    ├── session_start_runner.go     ← NEW: SessionStart test runner
    └── session_start_invariants.go ← NEW: SessionStart invariants
```

### Dependencies

All tickets in this document depend on:
- GOgent-062: CLI Binary (main orchestrator)
- GOgent-063: Integration Tests
- GOgent-064: Ecosystem Test Suite

---

## GOgent-067: SessionStart Deterministic Fixtures

**Time**: 1.5 hours
**Dependencies**: GOgent-062 (gogent-load-context binary exists)
**Priority**: HIGH (blocks other simulation tickets)

**Task**:
Create deterministic test fixtures for SessionStart hook execution.

**Files Created**:
- `test/simulation/fixtures/deterministic/sessionstart/S001_startup_basic.json`
- `test/simulation/fixtures/deterministic/sessionstart/S002_resume_with_handoff.json`
- `test/simulation/fixtures/deterministic/sessionstart/S003_resume_no_handoff.json`
- `test/simulation/fixtures/deterministic/sessionstart/S004_pending_learnings.json`
- `test/simulation/fixtures/deterministic/sessionstart/S005_go_project.json`
- `test/simulation/fixtures/deterministic/sessionstart/S006_python_project.json`
- `test/simulation/fixtures/deterministic/sessionstart/S007_empty_input.json`
- `test/simulation/fixtures/deterministic/sessionstart/S008_invalid_json.json`
- `test/simulation/fixtures/deterministic/sessionstart/S009_unknown_type.json`
- `test/simulation/fixtures/deterministic/sessionstart/S010_git_dirty.json`

**Implementation**:

### S001_startup_basic.json
```json
{
  "id": "S001",
  "category": "sessionstart",
  "description": "Basic startup session with no special context",
  "setup": {
    "create_dirs": [".claude/memory"],
    "files": {},
    "env": {}
  },
  "input": {
    "type": "startup",
    "session_id": "test-s001",
    "hook_event_name": "SessionStart"
  },
  "expected": {
    "exit_code": 0,
    "stdout_contains": [
      "hookSpecificOutput",
      "SessionStart",
      "startup",
      "SESSION INITIALIZED"
    ],
    "stdout_not_contains": [
      "PREVIOUS SESSION HANDOFF",
      "ERROR"
    ],
    "files_created": [],
    "validate_json": true
  }
}
```

### S002_resume_with_handoff.json
```json
{
  "id": "S002",
  "category": "sessionstart",
  "description": "Resume session loads previous handoff",
  "setup": {
    "create_dirs": [".claude/memory"],
    "files": {
      ".claude/memory/last-handoff.md": "# Previous Session\n\nCompleted feature X."
    },
    "env": {}
  },
  "input": {
    "type": "resume",
    "session_id": "test-s002",
    "hook_event_name": "SessionStart"
  },
  "expected": {
    "exit_code": 0,
    "stdout_contains": [
      "resume",
      "PREVIOUS SESSION HANDOFF",
      "Completed feature X"
    ],
    "validate_json": true
  }
}
```

### S003_resume_no_handoff.json
```json
{
  "id": "S003",
  "category": "sessionstart",
  "description": "Resume session without handoff (graceful handling)",
  "setup": {
    "create_dirs": [".claude/memory"],
    "files": {},
    "env": {}
  },
  "input": {
    "type": "resume",
    "session_id": "test-s003",
    "hook_event_name": "SessionStart"
  },
  "expected": {
    "exit_code": 0,
    "stdout_contains": [
      "resume",
      "SESSION INITIALIZED"
    ],
    "stdout_not_contains": [
      "PREVIOUS SESSION HANDOFF",
      "ERROR"
    ],
    "validate_json": true
  }
}
```

### S004_pending_learnings.json
```json
{
  "id": "S004",
  "category": "sessionstart",
  "description": "Session with pending learnings shows warning",
  "setup": {
    "create_dirs": [".claude/memory"],
    "files": {
      ".claude/memory/pending-learnings.jsonl": "{\"ts\":1234,\"file\":\"test.go\",\"error_type\":\"nil_pointer\"}\n{\"ts\":1235,\"file\":\"main.go\",\"error_type\":\"type_mismatch\"}\n"
    },
    "env": {}
  },
  "input": {
    "type": "startup",
    "session_id": "test-s004",
    "hook_event_name": "SessionStart"
  },
  "expected": {
    "exit_code": 0,
    "stdout_contains": [
      "PENDING LEARNINGS",
      "2 sharp edge"
    ],
    "validate_json": true
  }
}
```

### S005_go_project.json
```json
{
  "id": "S005",
  "category": "sessionstart",
  "description": "Go project detection",
  "setup": {
    "create_dirs": [],
    "files": {
      "go.mod": "module test"
    },
    "env": {}
  },
  "input": {
    "type": "startup",
    "session_id": "test-s005",
    "hook_event_name": "SessionStart"
  },
  "expected": {
    "exit_code": 0,
    "stdout_contains": [
      "PROJECT TYPE",
      "go"
    ],
    "validate_json": true
  }
}
```

### S006_python_project.json
```json
{
  "id": "S006",
  "category": "sessionstart",
  "description": "Python project detection",
  "setup": {
    "create_dirs": [],
    "files": {
      "pyproject.toml": "[project]\nname = \"test\""
    },
    "env": {}
  },
  "input": {
    "type": "startup",
    "session_id": "test-s006",
    "hook_event_name": "SessionStart"
  },
  "expected": {
    "exit_code": 0,
    "stdout_contains": [
      "PROJECT TYPE",
      "python"
    ],
    "validate_json": true
  }
}
```

### S007_empty_input.json
```json
{
  "id": "S007",
  "category": "sessionstart",
  "description": "Empty input produces error response",
  "setup": {
    "create_dirs": [],
    "files": {},
    "env": {}
  },
  "input_raw": "",
  "expected": {
    "exit_code": 1,
    "stdout_contains": [
      "ERROR",
      "Empty STDIN"
    ],
    "validate_json": true
  }
}
```

### S008_invalid_json.json
```json
{
  "id": "S008",
  "category": "sessionstart",
  "description": "Invalid JSON input produces error response",
  "setup": {
    "create_dirs": [],
    "files": {},
    "env": {}
  },
  "input_raw": "not valid json at all",
  "expected": {
    "exit_code": 1,
    "stdout_contains": [
      "ERROR",
      "Failed to parse JSON"
    ],
    "validate_json": true
  }
}
```

### S009_unknown_type.json
```json
{
  "id": "S009",
  "category": "sessionstart",
  "description": "Unknown session type produces error",
  "setup": {
    "create_dirs": [],
    "files": {},
    "env": {}
  },
  "input": {
    "type": "invalid_type",
    "session_id": "test-s009",
    "hook_event_name": "SessionStart"
  },
  "expected": {
    "exit_code": 1,
    "stdout_contains": [
      "ERROR",
      "Invalid session type"
    ],
    "validate_json": true
  }
}
```

### S010_git_dirty.json
```json
{
  "id": "S010",
  "category": "sessionstart",
  "description": "Git repository with uncommitted changes",
  "setup": {
    "create_dirs": [],
    "files": {},
    "env": {},
    "git_init": true,
    "git_dirty": ["uncommitted-file.txt"]
  },
  "input": {
    "type": "startup",
    "session_id": "test-s010",
    "hook_event_name": "SessionStart"
  },
  "expected": {
    "exit_code": 0,
    "stdout_contains": [
      "GIT:",
      "Uncommitted"
    ],
    "validate_json": true
  }
}
```

**Acceptance Criteria**:
- [ ] 10 deterministic fixtures created in `test/simulation/fixtures/deterministic/sessionstart/`
- [ ] Each fixture has: id, category, description, setup, input, expected
- [ ] Fixtures cover: startup, resume, project detection, error handling, git status
- [ ] `setup.files` creates project-specific files
- [ ] `expected.stdout_contains` validates context injection content
- [ ] `expected.validate_json` ensures valid JSON output
- [ ] All fixtures pass when harness is updated (GOgent-068)

**Test Deliverables**:
- [ ] Fixture files created: 10
- [ ] JSON schema validation: ✅

**Why This Matters**: Deterministic fixtures form the foundation of L1 testing. They ensure each code path in gogent-load-context is exercised.

---

## GOgent-068: SessionStart Harness Runner

**Time**: 2 hours
**Dependencies**: GOgent-067 (fixtures exist)
**Priority**: HIGH

**Task**:
Extend simulation harness to execute SessionStart scenarios.

**File**: `test/simulation/harness/session_start_runner.go` (new file)

**Implementation**:
```go
package harness

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// SessionStartScenario represents a SessionStart test case loaded from fixtures.
type SessionStartScenario struct {
	ID          string               `json:"id"`
	Category    string               `json:"category"`
	Description string               `json:"description"`
	Setup       SessionStartSetup    `json:"setup"`
	Input       interface{}          `json:"input"`
	InputRaw    string               `json:"input_raw,omitempty"`
	Expected    SessionStartExpected `json:"expected"`
}

// SessionStartSetup defines test environment configuration.
type SessionStartSetup struct {
	CreateDirs []string          `json:"create_dirs,omitempty"`
	Files      map[string]string `json:"files,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	GitInit    bool              `json:"git_init,omitempty"`
	GitDirty   []string          `json:"git_dirty,omitempty"`
}

// SessionStartExpected defines success criteria.
type SessionStartExpected struct {
	ExitCode         int      `json:"exit_code"`
	StdoutContains   []string `json:"stdout_contains,omitempty"`
	StdoutNotContain []string `json:"stdout_not_contains,omitempty"`
	StderrContains   []string `json:"stderr_contains,omitempty"`
	FilesCreated     []string `json:"files_created,omitempty"`
	ValidateJSON     bool     `json:"validate_json"`
}

// SessionStartRunner executes SessionStart test scenarios.
type SessionStartRunner struct {
	binaryPath  string
	fixturesDir string
	verbose     bool
}

// NewSessionStartRunner creates a runner for SessionStart tests.
func NewSessionStartRunner(binaryPath, fixturesDir string) *SessionStartRunner {
	return &SessionStartRunner{
		binaryPath:  binaryPath,
		fixturesDir: fixturesDir,
	}
}

// SetVerbose enables verbose output.
func (r *SessionStartRunner) SetVerbose(v bool) {
	r.verbose = v
}

// LoadScenarios loads all SessionStart scenarios from fixtures directory.
func (r *SessionStartRunner) LoadScenarios() ([]SessionStartScenario, error) {
	scenarioDir := filepath.Join(r.fixturesDir, "deterministic", "sessionstart")

	entries, err := os.ReadDir(scenarioDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No scenarios is valid
		}
		return nil, fmt.Errorf("read scenario dir: %w", err)
	}

	var scenarios []SessionStartScenario
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(scenarioDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}

		var scenario SessionStartScenario
		if err := json.Unmarshal(data, &scenario); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}

		scenarios = append(scenarios, scenario)
	}

	return scenarios, nil
}

// RunAll executes all loaded scenarios.
func (r *SessionStartRunner) RunAll(scenarios []SessionStartScenario) []SimulationResult {
	var results []SimulationResult

	for _, scenario := range scenarios {
		result := r.RunScenario(scenario)
		results = append(results, result)

		if r.verbose {
			status := "PASS"
			if !result.Passed {
				status = "FAIL"
			}
			fmt.Printf("[%s] %s: %s (%v)\n", status, scenario.ID, scenario.Description, result.Duration)
		}
	}

	return results
}

// RunScenario executes a single SessionStart scenario.
func (r *SessionStartRunner) RunScenario(scenario SessionStartScenario) SimulationResult {
	start := time.Now()
	result := SimulationResult{
		ScenarioID: scenario.ID,
	}

	// Create isolated temp directory
	tempDir, err := os.MkdirTemp("", "sessionstart-"+scenario.ID+"-")
	if err != nil {
		result.Error = err
		result.ErrorMsg = fmt.Sprintf("create temp dir: %v", err)
		result.Duration = time.Since(start)
		return result
	}
	defer os.RemoveAll(tempDir)

	// Setup environment
	if err := r.setupEnvironment(tempDir, scenario.Setup); err != nil {
		result.Error = err
		result.ErrorMsg = fmt.Sprintf("setup: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	// Prepare input
	var inputData []byte
	if scenario.InputRaw != "" {
		inputData = []byte(scenario.InputRaw)
	} else if scenario.Input != nil {
		inputData, _ = json.Marshal(scenario.Input)
	}
	result.Input = string(inputData)

	// Execute binary
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, r.binaryPath)
	cmd.Stdin = bytes.NewReader(inputData)
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(), "GOGENT_PROJECT_DIR="+tempDir)

	// Add custom env vars
	for k, v := range scenario.Setup.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		result.Error = err
		result.ErrorMsg = fmt.Sprintf("exec: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	result.Output = stdout.String()

	// Validate exit code
	if exitCode != scenario.Expected.ExitCode {
		result.Error = fmt.Errorf("exit code: got %d, want %d", exitCode, scenario.Expected.ExitCode)
		result.ErrorMsg = result.Error.Error()
		result.Duration = time.Since(start)
		return result
	}

	// Validate stdout contains
	for _, want := range scenario.Expected.StdoutContains {
		if !strings.Contains(result.Output, want) {
			result.Error = fmt.Errorf("stdout missing: %q", want)
			result.ErrorMsg = result.Error.Error()
			result.Duration = time.Since(start)
			return result
		}
	}

	// Validate stdout does not contain
	for _, notWant := range scenario.Expected.StdoutNotContain {
		if strings.Contains(result.Output, notWant) {
			result.Error = fmt.Errorf("stdout contains forbidden: %q", notWant)
			result.ErrorMsg = result.Error.Error()
			result.Duration = time.Since(start)
			return result
		}
	}

	// Validate JSON output
	if scenario.Expected.ValidateJSON {
		var parsed interface{}
		if err := json.Unmarshal([]byte(result.Output), &parsed); err != nil {
			result.Error = fmt.Errorf("invalid JSON output: %v", err)
			result.ErrorMsg = result.Error.Error()
			result.Duration = time.Since(start)
			return result
		}
	}

	// Validate files created
	for _, relPath := range scenario.Expected.FilesCreated {
		fullPath := filepath.Join(tempDir, relPath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			result.Error = fmt.Errorf("expected file not created: %s", relPath)
			result.ErrorMsg = result.Error.Error()
			result.Duration = time.Since(start)
			return result
		}
	}

	result.Passed = true
	result.Duration = time.Since(start)
	return result
}

// setupEnvironment creates directories, files, and git state as specified.
func (r *SessionStartRunner) setupEnvironment(tempDir string, setup SessionStartSetup) error {
	// Create directories
	for _, dir := range setup.CreateDirs {
		fullPath := filepath.Join(tempDir, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	// Create files
	for relPath, content := range setup.Files {
		fullPath := filepath.Join(tempDir, relPath)

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return fmt.Errorf("mkdir for %s: %w", relPath, err)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("write %s: %w", relPath, err)
		}
	}

	// Initialize git repo if requested
	if setup.GitInit {
		cmd := exec.Command("git", "init")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git init: %w", err)
		}

		// Configure git user (required for commits)
		exec.Command("git", "config", "user.email", "test@example.com").Run()
		exec.Command("git", "config", "user.name", "Test").Run()

		// Create initial commit
		exec.Command("git", "commit", "--allow-empty", "-m", "Initial commit").Run()

		// Create dirty files if specified
		for _, filename := range setup.GitDirty {
			fullPath := filepath.Join(tempDir, filename)
			if err := os.WriteFile(fullPath, []byte("dirty content"), 0644); err != nil {
				return fmt.Errorf("create dirty file %s: %w", filename, err)
			}
		}
	}

	return nil
}
```

**Tests**: `test/simulation/harness/session_start_runner_test.go`

```go
package harness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSessionStartRunner_LoadScenarios(t *testing.T) {
	// Create temp fixtures
	tmpDir := t.TempDir()
	scenarioDir := filepath.Join(tmpDir, "deterministic", "sessionstart")
	os.MkdirAll(scenarioDir, 0755)

	// Write test scenario
	scenario := SessionStartScenario{
		ID:          "TEST001",
		Category:    "sessionstart",
		Description: "Test scenario",
		Expected: SessionStartExpected{
			ExitCode: 0,
		},
	}
	data, _ := json.Marshal(scenario)
	os.WriteFile(filepath.Join(scenarioDir, "TEST001.json"), data, 0644)

	runner := NewSessionStartRunner("/bin/true", tmpDir)
	scenarios, err := runner.LoadScenarios()

	if err != nil {
		t.Fatalf("LoadScenarios failed: %v", err)
	}

	if len(scenarios) != 1 {
		t.Errorf("Expected 1 scenario, got %d", len(scenarios))
	}

	if scenarios[0].ID != "TEST001" {
		t.Errorf("Expected ID TEST001, got %s", scenarios[0].ID)
	}
}

func TestSessionStartRunner_SetupEnvironment(t *testing.T) {
	runner := NewSessionStartRunner("/bin/true", ".")

	tmpDir := t.TempDir()
	setup := SessionStartSetup{
		CreateDirs: []string{".claude/memory"},
		Files: map[string]string{
			"test.txt": "hello world",
			".claude/memory/handoff.md": "# Handoff",
		},
	}

	if err := runner.setupEnvironment(tmpDir, setup); err != nil {
		t.Fatalf("setupEnvironment failed: %v", err)
	}

	// Verify directories
	if _, err := os.Stat(filepath.Join(tmpDir, ".claude", "memory")); os.IsNotExist(err) {
		t.Error("Directory .claude/memory not created")
	}

	// Verify files
	if _, err := os.Stat(filepath.Join(tmpDir, "test.txt")); os.IsNotExist(err) {
		t.Error("File test.txt not created")
	}

	content, _ := os.ReadFile(filepath.Join(tmpDir, "test.txt"))
	if string(content) != "hello world" {
		t.Errorf("File content mismatch: %s", content)
	}
}

func TestSessionStartRunner_SetupEnvironment_GitInit(t *testing.T) {
	runner := NewSessionStartRunner("/bin/true", ".")

	tmpDir := t.TempDir()
	setup := SessionStartSetup{
		GitInit:  true,
		GitDirty: []string{"dirty.txt"},
	}

	if err := runner.setupEnvironment(tmpDir, setup); err != nil {
		t.Fatalf("setupEnvironment failed: %v", err)
	}

	// Verify git directory exists
	if _, err := os.Stat(filepath.Join(tmpDir, ".git")); os.IsNotExist(err) {
		t.Error(".git directory not created")
	}

	// Verify dirty file exists
	if _, err := os.Stat(filepath.Join(tmpDir, "dirty.txt")); os.IsNotExist(err) {
		t.Error("dirty.txt not created")
	}
}
```

**Acceptance Criteria**:
- [ ] `SessionStartRunner` struct created with `binaryPath` and `fixturesDir`
- [ ] `LoadScenarios()` reads all `.json` files from `fixtures/deterministic/sessionstart/`
- [ ] `RunScenario()` creates isolated temp dir, sets up environment, executes binary
- [ ] Validates: exit code, stdout contains/not-contains, JSON validity, files created
- [ ] `setupEnvironment()` handles directories, files, git init, git dirty state
- [ ] Tests verify scenario loading, environment setup, git initialization
- [ ] `go test ./test/simulation/harness/...` passes

**Test Deliverables**:
- [ ] Test file created: `test/simulation/harness/session_start_runner_test.go`
- [ ] Number of test functions: 3
- [ ] Tests passing: ✅

**Why This Matters**: The runner provides the execution engine for SessionStart testing. Without it, fixtures cannot be validated.

---

## GOgent-069: SessionStart Makefile Integration

**Time**: 0.5 hours
**Dependencies**: GOgent-068 (runner exists)
**Priority**: HIGH

**Task**:
Add Makefile targets for SessionStart simulation testing.

**File**: `Makefile` (extend existing)

**Implementation**:
```makefile
# Add to existing Makefile simulation section

# SessionStart simulation targets
test-simulation-sessionstart:
	@echo "Running SessionStart deterministic tests..."
	@mkdir -p test/simulation/reports
	go run ./test/simulation/harness/cmd/harness \
		-mode=deterministic \
		-category=sessionstart \
		-report=tap \
		-output=test/simulation/reports

test-simulation-sessionstart-verbose:
	@echo "Running SessionStart tests (verbose)..."
	go run ./test/simulation/harness/cmd/harness \
		-mode=deterministic \
		-category=sessionstart \
		-verbose \
		-report=tap

# Update build-all to include load-context
build-all: build-validate build-archive build-sharp-edge build-capture-intent build-load-context
	@echo "✅ All binaries built"

# Build load-context binary
build-load-context:
	@echo "Building gogent-load-context binary..."
	go build -o bin/gogent-load-context ./cmd/gogent-load-context
	@echo "✅ Binary created at bin/gogent-load-context"

# Install load-context
install-load-context: build-load-context
	@echo "Installing gogent-load-context to ~/.local/bin/..."
	mkdir -p ~/.local/bin
	cp bin/gogent-load-context ~/.local/bin/gogent-load-context
	chmod +x ~/.local/bin/gogent-load-context
	@echo "✅ Installed to ~/.local/bin/gogent-load-context"
```

**Update harness main.go to support category filter**:
```go
// In test/simulation/harness/cmd/harness/main.go

// Add flag
var categoryFilter = flag.String("category", "", "Run only scenarios in this category (pretooluse, posttooluse, sessionstart)")

// In mode selection
if *categoryFilter == "sessionstart" {
    runner := harness.NewSessionStartRunner(loadContextPath, cfg.FixturesDir)
    runner.SetVerbose(cfg.Verbose)
    scenarios, err := runner.LoadScenarios()
    if err != nil {
        log.Fatalf("Load scenarios: %v", err)
    }
    results := runner.RunAll(scenarios)
    // ... report results
}
```

**Acceptance Criteria**:
- [ ] `make test-simulation-sessionstart` runs SessionStart tests
- [ ] `make test-simulation-sessionstart-verbose` runs with verbose output
- [ ] `make build-all` includes `build-load-context`
- [ ] `make build-load-context` builds the binary
- [ ] `make install-load-context` installs to `~/.local/bin`
- [ ] Harness CLI supports `-category=sessionstart` filter

**Test Deliverables**:
- [ ] `make test-simulation-sessionstart` executes without error
- [ ] All SessionStart fixtures pass

**Why This Matters**: Makefile targets provide standard developer interface for running tests locally before CI.

---

## GOgent-070: SessionStart Behavioral Invariants

**Time**: 1.5 hours
**Dependencies**: GOgent-068 (runner exists)
**Priority**: MEDIUM

**Task**:
Define behavioral invariants for SessionStart hook execution.

**File**: `test/simulation/harness/session_start_invariants.go` (new file)

**Implementation**:
```go
package harness

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SessionStartInvariant defines a property that must hold for all SessionStart executions.
type SessionStartInvariant struct {
	ID    string
	Name  string
	Check func(output string, scenario SessionStartScenario) (bool, string)
}

// SessionStartInvariants defines properties that must hold across SessionStart scenarios.
var SessionStartInvariants = []SessionStartInvariant{
	{
		// SS1: All successful responses are valid JSON
		ID:   "SS1",
		Name: "output_is_valid_json",
		Check: func(output string, scenario SessionStartScenario) (bool, string) {
			if scenario.Expected.ExitCode != 0 {
				// Error responses should also be valid JSON
			}

			var parsed interface{}
			if err := json.Unmarshal([]byte(output), &parsed); err != nil {
				return false, fmt.Sprintf("invalid JSON: %v", err)
			}
			return true, ""
		},
	},
	{
		// SS2: Response contains required hook structure
		ID:   "SS2",
		Name: "response_has_hook_structure",
		Check: func(output string, scenario SessionStartScenario) (bool, string) {
			var response map[string]interface{}
			if err := json.Unmarshal([]byte(output), &response); err != nil {
				return false, fmt.Sprintf("parse error: %v", err)
			}

			hookOutput, ok := response["hookSpecificOutput"].(map[string]interface{})
			if !ok {
				return false, "missing hookSpecificOutput"
			}

			if _, ok := hookOutput["hookEventName"]; !ok {
				return false, "missing hookEventName in hookSpecificOutput"
			}

			if _, ok := hookOutput["additionalContext"]; !ok {
				return false, "missing additionalContext in hookSpecificOutput"
			}

			return true, ""
		},
	},
	{
		// SS3: hookEventName is always "SessionStart"
		ID:   "SS3",
		Name: "hook_event_name_is_sessionstart",
		Check: func(output string, scenario SessionStartScenario) (bool, string) {
			var response map[string]interface{}
			if err := json.Unmarshal([]byte(output), &response); err != nil {
				return true, "" // Skip if can't parse
			}

			hookOutput, ok := response["hookSpecificOutput"].(map[string]interface{})
			if !ok {
				return true, "" // Skip if structure missing
			}

			eventName, _ := hookOutput["hookEventName"].(string)
			if eventName != "SessionStart" {
				return false, fmt.Sprintf("hookEventName=%q, want SessionStart", eventName)
			}

			return true, ""
		},
	},
	{
		// SS4: Startup sessions do not include handoff
		ID:   "SS4",
		Name: "startup_excludes_handoff",
		Check: func(output string, scenario SessionStartScenario) (bool, string) {
			// Only check startup sessions
			input, ok := scenario.Input.(map[string]interface{})
			if !ok {
				return true, "" // Skip non-standard input
			}

			sessionType, _ := input["type"].(string)
			if sessionType != "startup" {
				return true, "" // Only check startup
			}

			if strings.Contains(output, "PREVIOUS SESSION HANDOFF") {
				return false, "startup session should not include handoff"
			}

			return true, ""
		},
	},
	{
		// SS5: Resume sessions include handoff if available
		ID:   "SS5",
		Name: "resume_includes_handoff_when_present",
		Check: func(output string, scenario SessionStartScenario) (bool, string) {
			// Only check resume sessions with handoff in setup
			input, ok := scenario.Input.(map[string]interface{})
			if !ok {
				return true, ""
			}

			sessionType, _ := input["type"].(string)
			if sessionType != "resume" {
				return true, ""
			}

			// Check if setup includes handoff file
			if _, hasHandoff := scenario.Setup.Files[".claude/memory/last-handoff.md"]; !hasHandoff {
				return true, "" // No handoff to include
			}

			if !strings.Contains(output, "PREVIOUS SESSION HANDOFF") {
				return false, "resume session with handoff should include PREVIOUS SESSION HANDOFF"
			}

			return true, ""
		},
	},
	{
		// SS6: Error responses indicate ERROR in context
		ID:   "SS6",
		Name: "error_responses_indicate_error",
		Check: func(output string, scenario SessionStartScenario) (bool, string) {
			if scenario.Expected.ExitCode == 0 {
				return true, "" // Only check error cases
			}

			var response map[string]interface{}
			if err := json.Unmarshal([]byte(output), &response); err != nil {
				return false, "error response is not valid JSON"
			}

			hookOutput, _ := response["hookSpecificOutput"].(map[string]interface{})
			context, _ := hookOutput["additionalContext"].(string)

			if !strings.Contains(context, "ERROR") {
				return false, "error response should contain ERROR indicator"
			}

			return true, ""
		},
	},
	{
		// SS7: Project type detection is consistent
		ID:   "SS7",
		Name: "project_type_in_context",
		Check: func(output string, scenario SessionStartScenario) (bool, string) {
			// If setup has go.mod, output should mention go
			if _, hasGoMod := scenario.Setup.Files["go.mod"]; hasGoMod {
				if !strings.Contains(strings.ToLower(output), "go") {
					return false, "go.mod present but 'go' not in output"
				}
			}

			// If setup has pyproject.toml, output should mention python
			if _, hasPyproject := scenario.Setup.Files["pyproject.toml"]; hasPyproject {
				if !strings.Contains(strings.ToLower(output), "python") {
					return false, "pyproject.toml present but 'python' not in output"
				}
			}

			return true, ""
		},
	},
}

// CheckSessionStartInvariants runs all invariants against a scenario result.
func CheckSessionStartInvariants(output string, scenario SessionStartScenario) []InvariantResult {
	var results []InvariantResult

	for _, inv := range SessionStartInvariants {
		passed, message := inv.Check(output, scenario)
		results = append(results, InvariantResult{
			InvariantID: inv.ID,
			Passed:      passed,
			Message:     message,
		})
	}

	return results
}
```

**Tests**: `test/simulation/harness/session_start_invariants_test.go`

```go
package harness

import (
	"testing"
)

func TestSessionStartInvariant_SS1_ValidJSON(t *testing.T) {
	inv := SessionStartInvariants[0] // SS1

	// Valid JSON
	passed, msg := inv.Check(`{"hookSpecificOutput":{"hookEventName":"SessionStart"}}`, SessionStartScenario{})
	if !passed {
		t.Errorf("SS1 should pass for valid JSON: %s", msg)
	}

	// Invalid JSON
	passed, _ = inv.Check("not json", SessionStartScenario{})
	if passed {
		t.Error("SS1 should fail for invalid JSON")
	}
}

func TestSessionStartInvariant_SS4_StartupExcludesHandoff(t *testing.T) {
	inv := SessionStartInvariants[3] // SS4

	scenario := SessionStartScenario{
		Input: map[string]interface{}{"type": "startup"},
	}

	// Without handoff mention - should pass
	passed, _ := inv.Check(`{"hookSpecificOutput":{"additionalContext":"SESSION INITIALIZED"}}`, scenario)
	if !passed {
		t.Error("SS4 should pass when startup has no handoff")
	}

	// With handoff mention - should fail
	passed, msg := inv.Check(`{"hookSpecificOutput":{"additionalContext":"PREVIOUS SESSION HANDOFF"}}`, scenario)
	if passed {
		t.Errorf("SS4 should fail when startup mentions handoff: %s", msg)
	}
}

func TestCheckSessionStartInvariants(t *testing.T) {
	output := `{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"test"}}`
	scenario := SessionStartScenario{}

	results := CheckSessionStartInvariants(output, scenario)

	if len(results) != len(SessionStartInvariants) {
		t.Errorf("Expected %d results, got %d", len(SessionStartInvariants), len(results))
	}

	for _, r := range results {
		if r.InvariantID == "" {
			t.Error("Result missing InvariantID")
		}
	}
}
```

**Acceptance Criteria**:
- [ ] 7 behavioral invariants defined for SessionStart
- [ ] SS1: Output is valid JSON
- [ ] SS2: Response has hookSpecificOutput structure
- [ ] SS3: hookEventName is "SessionStart"
- [ ] SS4: Startup sessions exclude handoff
- [ ] SS5: Resume sessions include handoff when available
- [ ] SS6: Error responses indicate ERROR
- [ ] SS7: Project type detection is consistent
- [ ] `CheckSessionStartInvariants()` runs all invariants
- [ ] Tests verify invariant logic
- [ ] `go test ./test/simulation/harness/...` passes

**Test Deliverables**:
- [ ] Test file created: `test/simulation/harness/session_start_invariants_test.go`
- [ ] Number of test functions: 3
- [ ] Tests passing: ✅

**Why This Matters**: Invariants ensure SessionStart hook behavior is consistent across all scenarios. They catch regressions early.

---

## GOgent-071: GitHub Actions Workflow Update

**Time**: 1 hour
**Dependencies**: GOgent-069 (Makefile targets exist)
**Priority**: HIGH

**Task**:
Update GitHub Actions workflows to include SessionStart simulation testing.

**File**: `.github/workflows/simulation-behavioral.yml` (extend existing)

**Implementation**:
```yaml
# Add to existing simulation-behavioral.yml

# Update build step to include load-context
- name: Build CLIs
  run: make build-validate build-archive build-sharp-edge build-load-context

# Add SessionStart to L1 Unit Invariants
- name: Run SessionStart Deterministic Tests
  run: make test-simulation-sessionstart

# Update the behavioral tests job to run SessionStart invariants
behavioral-properties:
  name: L3 Behavioral Properties
  runs-on: ubuntu-latest
  needs: session-replay
  if: github.event_name == 'pull_request' || github.event_name == 'workflow_dispatch'
  timeout-minutes: 20

  steps:
    # ... existing steps ...

    - name: Build CLIs
      run: make build-validate build-archive build-sharp-edge build-load-context

    - name: Run Behavioral Tests (including SessionStart)
      run: make test-simulation-behavioral

    - name: Run SessionStart Invariant Tests
      run: |
        go run ./test/simulation/harness/cmd/harness \
          -mode=behavioral \
          -category=sessionstart \
          -report=json \
          -output=test/simulation/reports
```

**Create new workflow file**: `.github/workflows/simulation-sessionstart.yml`

```yaml
# Session Initialization Testing
# Tests gogent-load-context (SessionStart hook)

name: Session Start Simulation

on:
  push:
    branches: [master]
    paths:
      - 'cmd/gogent-load-context/**'
      - 'pkg/session/**'
      - 'test/simulation/fixtures/deterministic/sessionstart/**'
  pull_request:
    branches: [master]
    paths:
      - 'cmd/gogent-load-context/**'
      - 'pkg/session/**'
      - 'test/simulation/fixtures/deterministic/sessionstart/**'
  workflow_dispatch:

env:
  GO_VERSION: '1.25'

jobs:
  sessionstart-tests:
    name: SessionStart Simulation
    runs-on: ubuntu-latest
    timeout-minutes: 15

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}

      - name: Cache Go modules
        uses: actions/cache@v4
        with:
          path: |
            ~/.cache/go-build
            ~/go/pkg/mod
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
          restore-keys: |
            ${{ runner.os }}-go-

      - name: Build gogent-load-context
        run: make build-load-context

      - name: Run SessionStart Deterministic Tests
        run: make test-simulation-sessionstart

      - name: Run SessionStart Invariants
        run: |
          go run ./test/simulation/harness/cmd/harness \
            -mode=behavioral \
            -category=sessionstart \
            -report=json \
            -output=test/simulation/reports

      - name: Upload Reports
        uses: actions/upload-artifact@v4
        if: always()
        with:
          name: sessionstart-reports-${{ github.run_number }}
          path: test/simulation/reports/
          retention-days: 14
          if-no-files-found: ignore

  sessionstart-status:
    name: SessionStart Status
    runs-on: ubuntu-latest
    needs: [sessionstart-tests]
    if: always()

    steps:
      - name: Check Results
        run: |
          if [ "${{ needs.sessionstart-tests.result }}" != "success" ]; then
            echo "::error::SessionStart tests failed"
            exit 1
          fi
          echo "✅ SessionStart simulation passed!"
```

**Acceptance Criteria**:
- [ ] `simulation-behavioral.yml` builds `gogent-load-context`
- [ ] L1 includes SessionStart deterministic tests
- [ ] L3 includes SessionStart invariant tests
- [ ] New `simulation-sessionstart.yml` workflow created
- [ ] Path filters trigger on changes to `cmd/gogent-load-context/`, `pkg/session/`, fixtures
- [ ] Artifacts uploaded on completion
- [ ] Status check gates merge

**Test Deliverables**:
- [ ] Workflow files valid YAML
- [ ] CI runs without error on push

**Why This Matters**: GitHub Actions integration ensures SessionStart tests run on every PR, catching regressions before merge.

---

## GOgent-072: Session Replay SessionStart Events

**Time**: 1.5 hours
**Dependencies**: GOgent-068 (runner exists)
**Priority**: MEDIUM

**Task**:
Extend session replay to support SessionStart events in multi-turn sequences.

**File**: `test/simulation/harness/session_replayer.go` (extend existing)

**Implementation**:

Add SessionStart support to `ReplayEvent`:
```go
// In session_replayer.go

// HookType constants
const (
	HookTypePreToolUse  = "PreToolUse"
	HookTypePostToolUse = "PostToolUse"
	HookTypeSessionStart = "SessionStart"
	HookTypeSessionEnd  = "SessionEnd"
)

// executeEvent handles event execution based on hook type
func (r *SessionReplayer) executeEvent(event ReplayEvent, tempDir string) (string, error) {
	switch event.HookType {
	case HookTypeSessionStart:
		return r.executeSessionStart(event, tempDir)
	case HookTypePreToolUse:
		return r.executePreToolUse(event, tempDir)
	case HookTypePostToolUse:
		return r.executePostToolUse(event, tempDir)
	case HookTypeSessionEnd:
		return r.executeSessionEnd(event, tempDir)
	default:
		return "", fmt.Errorf("unknown hook type: %s", event.HookType)
	}
}

// executeSessionStart runs gogent-load-context
func (r *SessionReplayer) executeSessionStart(event ReplayEvent, tempDir string) (string, error) {
	// Build input JSON
	input := map[string]interface{}{
		"type":            event.ToolInput["type"],
		"session_id":      event.ToolInput["session_id"],
		"hook_event_name": "SessionStart",
	}
	inputData, _ := json.Marshal(input)

	// Execute binary
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, r.loadContextPath)
	cmd.Stdin = bytes.NewReader(inputData)
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(), "GOGENT_PROJECT_DIR="+tempDir)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("execute: %w", err)
	}

	return string(output), nil
}
```

**Create session replay fixture**: `test/simulation/fixtures/sessions/session-init-flow.jsonl`

```json
{"ts":1,"hook_type":"SessionStart","tool_name":"","tool_input":{"type":"startup","session_id":"flow-001"},"expected_decision":""}
{"ts":2,"hook_type":"PreToolUse","tool_name":"Task","tool_input":{"subagent_type":"Explore","prompt":"AGENT: codebase-search\n\nFind all go files"},"expected_decision":"allow"}
{"ts":3,"hook_type":"PostToolUse","tool_name":"Bash","tool_input":{"file_path":"test.go","command":"go build"},"tool_response":{"error":"undefined: foo"},"success":false,"expected_decision":""}
{"ts":4,"hook_type":"PostToolUse","tool_name":"Bash","tool_input":{"file_path":"test.go","command":"go build"},"tool_response":{"error":"undefined: foo"},"success":false,"expected_decision":""}
{"ts":5,"hook_type":"PostToolUse","tool_name":"Bash","tool_input":{"file_path":"test.go","command":"go build"},"tool_response":{"error":"undefined: foo"},"success":false,"expected_decision":"block"}
```

**Create resume session fixture**: `test/simulation/fixtures/sessions/session-resume-flow.jsonl`

```json
{"ts":1,"hook_type":"SessionStart","tool_name":"","tool_input":{"type":"resume","session_id":"flow-002"},"expected_decision":""}
{"ts":2,"hook_type":"PreToolUse","tool_name":"Read","tool_input":{"file_path":"main.go"},"expected_decision":"allow"}
```

**Acceptance Criteria**:
- [ ] `HookTypeSessionStart` constant added
- [ ] `executeSessionStart()` method implemented in SessionReplayer
- [ ] Session replay handles SessionStart events in sequences
- [ ] `session-init-flow.jsonl` fixture tests startup → tools → failure loop
- [ ] `session-resume-flow.jsonl` fixture tests resume session
- [ ] `make test-simulation-replay` includes SessionStart events
- [ ] `go test ./test/simulation/harness/...` passes

**Test Deliverables**:
- [ ] Fixtures created: 2
- [ ] Session replay integration tests pass

**Why This Matters**: Session replay testing validates complete session lifecycles. SessionStart support ensures context injection is tested in realistic multi-turn scenarios.

---

## Summary: Integration Testing Architecture

After completing GOgent-067 to 072:

```
                    ┌─────────────────────────────────────────┐
                    │         GitHub Actions CI/CD            │
                    │                                         │
                    │  ┌─────────┐ ┌─────────┐ ┌───────────┐ │
                    │  │ L1 Unit │ │L2 Replay│ │L3 Behavior│ │
                    │  │Invariant│ │ Session │ │ Properties│ │
                    │  └────┬────┘ └────┬────┘ └─────┬─────┘ │
                    │       │           │            │        │
                    └───────┼───────────┼────────────┼────────┘
                            │           │            │
                    ┌───────▼───────────▼────────────▼────────┐
                    │         Simulation Harness              │
                    │                                         │
                    │  ┌──────────────────────────────────┐  │
                    │  │        SessionStartRunner         │  │
                    │  │   LoadScenarios() RunScenario()   │  │
                    │  └───────────────┬──────────────────┘  │
                    │                  │                      │
                    │  ┌───────────────▼──────────────────┐  │
                    │  │     SessionStartInvariants        │  │
                    │  │  SS1-SS7 (output validation)      │  │
                    │  └──────────────────────────────────┘  │
                    │                                         │
                    └─────────────────────────────────────────┘
                                       │
                    ┌──────────────────▼──────────────────────┐
                    │            Fixtures                     │
                    │                                         │
                    │  fixtures/deterministic/sessionstart/   │
                    │    S001_startup_basic.json              │
                    │    S002_resume_with_handoff.json        │
                    │    ...                                  │
                    │                                         │
                    │  fixtures/sessions/                     │
                    │    session-init-flow.jsonl              │
                    │    session-resume-flow.jsonl            │
                    │                                         │
                    └─────────────────────────────────────────┘
```

### Ticket Dependencies

```
GOgent-067 (Fixtures) ─────┬──────▶ GOgent-068 (Runner)
                           │                │
                           │                ▼
                           │       GOgent-069 (Makefile)
                           │                │
                           ├───────────────▶├──────▶ GOgent-070 (Invariants)
                           │                │
                           │                ▼
                           └───────▶ GOgent-071 (GitHub Actions)
                                            │
                                            ▼
                                   GOgent-072 (Session Replay)
```

### Estimated Total Time

| Ticket | Time |
|--------|------|
| GOgent-067 | 1.5h |
| GOgent-068 | 2.0h |
| GOgent-069 | 0.5h |
| GOgent-070 | 1.5h |
| GOgent-071 | 1.0h |
| GOgent-072 | 1.5h |
| **Total** | **8.0h** |

### Success Criteria

All tickets complete when:
- [ ] `make test-simulation-sessionstart` passes
- [ ] `make test-simulation-behavioral` includes SessionStart
- [ ] GitHub Actions `simulation-sessionstart.yml` workflow passes
- [ ] Session replay supports SessionStart events
- [ ] All 7 SessionStart invariants pass on all fixtures
