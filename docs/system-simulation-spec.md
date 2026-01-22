# GOgent-Fortress System Simulation Test Specification

> **Document Type:** Einstein Analysis Output
> **Generated:** 2026-01-22
> **Architecture Coverage:** ~33% (pkg/{routing,session,config,telemetry}, cmd/{validate,archive,aggregate})
> **Total LoC:** ~34,000 (including tests)

---

## Executive Summary

This document defines a simulation harness for end-to-end testing of the GOgent-Fortress hook system. The harness enables:

1. **Deterministic replay** - Known inputs produce verifiable outputs
2. **Randomized fuzzing** - Stress-test edge cases with varied payloads
3. **Integration verification** - Cross-package data flow validation
4. **Regression detection** - Catch behavioral changes between versions

---

## 1. Architecture Snapshot (Current State)

### 1.1 Package Dependency Graph

```
                                    ┌─────────────────┐
                                    │   cmd/validate  │
                                    │   (main.go)     │
                                    └────────┬────────┘
                                             │
┌─────────────────┐                          │                    ┌─────────────────┐
│  cmd/archive    │                          │                    │  cmd/aggregate  │
│   (main.go)     │                          │                    │   (main.go)     │
└────────┬────────┘                          │                    └────────┬────────┘
         │                                   │                             │
         ▼                                   ▼                             ▼
┌──────────────────────────────────────────────────────────────────────────────────┐
│                              pkg/routing                                          │
│  ┌─────────┐ ┌─────────┐ ┌───────────┐ ┌────────────────┐ ┌─────────────────────┐│
│  │ schema  │ │ events  │ │ validator │ │ task_validation│ │ subagent_validation ││
│  └────┬────┘ └────┬────┘ └─────┬─────┘ └───────┬────────┘ └──────────┬──────────┘│
│       │          │            │                │                     │           │
│  ┌────┴────┐ ┌───┴────┐ ┌─────┴──────┐ ┌──────┴───────┐ ┌───────────┴─────────┐ │
│  │ agents  │ │ stdin  │ │ violations │ │ delegation   │ │ transcript          │ │
│  │         │ │        │ │            │ │ ceiling      │ │                     │ │
│  └─────────┘ └────────┘ └────────────┘ └──────────────┘ └─────────────────────┘ │
└────────────────────────────────────────────────────────────────────────────────┬─┘
                                             │                                   │
                                             ▼                                   │
┌──────────────────────────────────────────────────────────────────────────────┐ │
│                              pkg/session                                      │ │
│  ┌──────────┐ ┌──────────┐ ┌────────────────┐ ┌───────────────┐ ┌──────────┐ │ │
│  │ handoff  │ │ metrics  │ │ handoff_       │ │ handoff_      │ │ query    │ │ │
│  │          │ │          │ │ artifacts      │ │ markdown      │ │          │ │ │
│  └──────────┘ └──────────┘ └────────────────┘ └───────────────┘ └──────────┘ │ │
│  ┌──────────┐ ┌──────────┐ ┌────────────────┐                                │ │
│  │ archive  │ │ events   │ │ violations_    │                                │ │
│  │          │ │          │ │ summary        │                                │ │
│  └──────────┘ └──────────┘ └────────────────┘                                │ │
└──────────────────────────────────────────────────────────────────────────────┘ │
                                             │                                   │
                                             ▼                                   │
┌──────────────────────────────────────────────────────────────────────────────┐ │
│                              pkg/telemetry                                    │ │
│  ┌─────────────┐ ┌──────────┐ ┌─────────────┐ ┌─────────┐                    │ │
│  │ invocations │ │ cost     │ │ escalations │ │ scout   │                    │ │
│  └─────────────┘ └──────────┘ └─────────────┘ └─────────┘                    │ │
└──────────────────────────────────────────────────────────────────────────────┘ │
                                             │                                   │
                                             ▼                                   │
┌──────────────────────────────────────────────────────────────────────────────┐ │
│                              pkg/config                                       │◄┘
│  ┌──────────┐ ┌──────────┐                                                   │
│  │ paths    │ │ tier     │                                                   │
│  └──────────┘ └──────────┘                                                   │
└──────────────────────────────────────────────────────────────────────────────┘
```

### 1.2 Data Flow Summary

| Flow | Entry Point | Processing | Exit Point |
|------|-------------|------------|------------|
| **PreToolUse** | `gogent-validate` STDIN | `routing.ValidateTask()` | STDOUT JSON |
| **SessionEnd** | `gogent-archive` STDIN | `session.GenerateHandoff()` | `.claude/memory/` files |
| **Aggregation** | `gogent-aggregate` CLI | `telemetry.ClusterInvocations*()` | STDOUT report |

### 1.3 Core Data Types

```go
// Input: PreToolUse hook event
type ToolEvent struct {
    ToolName      string                 // "Task", "Read", "Edit", etc.
    ToolInput     map[string]interface{} // Tool-specific payload
    SessionID     string                 // UUID-like identifier
    HookEventName string                 // "PreToolUse"
    CapturedAt    int64                  // Unix timestamp
}

// Input: Task tool_input (when ToolName == "Task")
type TaskInput struct {
    Model        string // "haiku", "sonnet", "opus"
    Prompt       string // Agent delegation prompt
    SubagentType string // "Explore", "general-purpose", "Plan", "Bash"
    Description  string // Task description
}

// Output: Validation result
type ValidationResult struct {
    Decision            string // "allow" or "block"
    Reason              string
    EinsteinBlocked     *TaskValidationResult
    ModelMismatch       string
    CeilingViolation    string
    SubagentTypeInvalid *SubagentTypeValidation
    Violations          []*Violation
}

// Input: SessionEnd event
type SessionEvent struct {
    SessionID     string
    HookEventName string // "SessionEnd"
    CapturedAt    int64
}

// Output: Session handoff
type Handoff struct {
    SchemaVersion string
    Timestamp     int64
    SessionID     string
    Context       SessionContext
    Artifacts     HandoffArtifacts
    Actions       []Action
}
```

---

## 2. Simulation Test Categories

### 2.1 Deterministic Scenarios

Fixed inputs with expected outputs for regression testing.

#### Scenario Matrix: PreToolUse (gogent-validate)

| ID | Category | Input Characteristics | Expected Output |
|----|----------|----------------------|-----------------|
| V001 | Pass-through | Non-Task tool (Read, Edit) | `{}` (allow) |
| V002 | Valid Task | Correct subagent_type mapping | `{decision: "allow"}` |
| V003 | Einstein Block | `model: "opus"` | `{decision: "block", reason: "Task(opus) blocked"}` |
| V004 | Subagent Mismatch | `agent: codebase-search, subagent_type: general-purpose` | `{decision: "block"}` |
| V005 | Ceiling Violation | `model: "sonnet"` when ceiling is `haiku` | `{decision: "block"}` |
| V006 | Model Warning | Model doesn't match agent-index spec | `{decision: "allow", modelMismatch: "..."}` |
| V007 | Unknown Agent | Agent not in routing-schema | `{decision: "block"}` |
| V008 | Empty Prompt | Task with empty prompt | `{decision: "block"}` |

#### Scenario Matrix: SessionEnd (gogent-archive)

| ID | Category | Input Characteristics | Expected Output |
|----|----------|----------------------|-----------------|
| S001 | Clean Session | Zero violations, zero sharp edges | Handoff with empty artifacts |
| S002 | With Violations | 3 routing violations logged | Handoff with violations array |
| S003 | With Sharp Edges | 2 sharp edges in pending-learnings | Handoff with sharp_edges array |
| S004 | Mixed Artifacts | Violations + sharp edges + user intents | Complete handoff document |
| S005 | Git Dirty | Uncommitted changes present | GitInfo.IsDirty = true |
| S006 | No Git | Non-git directory | GitInfo = {} |
| S007 | Active Ticket | .ticket-current present | Context.ActiveTicket set |
| S008 | Schema Migration | v1.0 handoff in file | Migrated to v1.1 |

### 2.2 Randomized Fuzzing Parameters

Parameters that can be varied to stress-test edge cases.

```yaml
fuzzing_parameters:
  tool_event:
    tool_name:
      distribution: weighted
      values:
        Task: 0.4
        Read: 0.2
        Edit: 0.15
        Write: 0.1
        Bash: 0.1
        Glob: 0.05
    session_id:
      distribution: uuid_v4
    captured_at:
      distribution: uniform
      range: [1704067200, 1767225600]  # 2024-01-01 to 2026-01-01 Unix

  task_input:
    model:
      distribution: weighted
      values:
        haiku: 0.5
        sonnet: 0.35
        opus: 0.1
        invalid: 0.05
    subagent_type:
      distribution: weighted
      values:
        Explore: 0.3
        general-purpose: 0.4
        Plan: 0.15
        Bash: 0.1
        invalid: 0.05
    agent:
      distribution: uniform
      values:
        - codebase-search
        - tech-docs-writer
        - python-pro
        - orchestrator
        - architect
        - haiku-scout
        - unknown-agent
    prompt_length:
      distribution: exponential
      mean: 500
      max: 10000

  session_metrics:
    tool_calls:
      distribution: poisson
      lambda: 25
    errors_logged:
      distribution: poisson
      lambda: 2
    routing_violations:
      distribution: poisson
      lambda: 1
```

---

## 3. Simulation Harness Design

### 3.1 Directory Structure

```
test/simulation/
├── harness/
│   ├── main.go              # Simulation runner
│   ├── generator.go         # Input generation (deterministic + random)
│   ├── validator.go         # Output validation
│   └── reporter.go          # Results aggregation
├── fixtures/
│   ├── deterministic/
│   │   ├── pretooluse/      # V001-V008 inputs
│   │   └── sessionend/      # S001-S008 inputs
│   ├── expected/
│   │   ├── pretooluse/      # Expected outputs
│   │   └── sessionend/      # Expected outputs
│   └── schemas/
│       ├── routing-schema.json      # Test schema
│       └── agents-index.json        # Test agents
├── fuzz/
│   ├── seeds/               # Seed corpus for fuzzing
│   └── crashes/             # Discovered failures
└── reports/
    └── .gitkeep
```

### 3.2 Core Harness Interface

```go
package harness

// SimulationConfig defines test execution parameters
type SimulationConfig struct {
    // Mode selection
    Mode string // "deterministic", "fuzz", "mixed"

    // Deterministic settings
    ScenarioFilter []string // Filter by ID prefix (e.g., "V00", "S00")

    // Fuzz settings
    FuzzIterations int     // Number of random iterations
    FuzzSeed       int64   // Random seed for reproducibility
    FuzzTimeout    time.Duration

    // Environment
    SchemaPath     string  // Path to test routing-schema.json
    AgentsPath     string  // Path to test agents-index.json
    TempDir        string  // Temporary directory for test artifacts

    // Output
    ReportFormat   string  // "json", "markdown", "tap"
    Verbose        bool
}

// Scenario represents a single test case
type Scenario struct {
    ID          string
    Category    string
    Description string
    Input       interface{}     // ToolEvent or SessionEvent
    Setup       SetupFunc       // Pre-test environment setup
    Expected    ExpectedOutput
    Teardown    TeardownFunc    // Post-test cleanup
}

// ExpectedOutput defines validation criteria
type ExpectedOutput struct {
    // For gogent-validate
    Decision     *string            // "allow" or "block"
    ReasonMatch  *regexp.Regexp     // Regex for reason field
    HasViolation *string            // Expected violation type

    // For gogent-archive
    HandoffFields map[string]interface{} // JSON path assertions
    FilesCreated  []string              // Expected file paths

    // For any
    ExitCode     int
    StderrMatch  *regexp.Regexp
}

// SimulationResult captures test execution outcome
type SimulationResult struct {
    ScenarioID  string
    Passed      bool
    Duration    time.Duration
    Input       string          // Serialized input
    Output      string          // Captured output
    Expected    string          // Expected output
    Diff        string          // If failed, the diff
    Error       error           // If execution error
}

// Runner executes simulations
type Runner interface {
    Run(cfg SimulationConfig) ([]SimulationResult, error)
    RunScenario(s Scenario) SimulationResult
}
```

### 3.3 Generator Interface

```go
package harness

// Generator creates test inputs
type Generator interface {
    // Deterministic generation
    GenerateToolEvent(scenarioID string) (*routing.ToolEvent, error)
    GenerateSessionEvent(scenarioID string) (*session.SessionEvent, error)

    // Randomized generation
    RandomToolEvent(seed int64) *routing.ToolEvent
    RandomTaskInput(seed int64) *routing.TaskInput
    RandomSessionEvent(seed int64) *session.SessionEvent
    RandomSessionMetrics(seed int64) *session.SessionMetrics

    // Parameterized generation
    GenerateWithParams(params FuzzParams) interface{}
}

// FuzzParams controls random generation
type FuzzParams struct {
    ToolNameWeights      map[string]float64
    ModelWeights         map[string]float64
    SubagentTypeWeights  map[string]float64
    AgentList            []string
    PromptLengthMean     int
    PromptLengthMax      int
    ErrorRate            float64
    ViolationRate        float64
}
```

---

## 4. Deterministic Test Fixtures

### 4.1 PreToolUse Fixtures

#### V001: Pass-through Non-Task Tool

```json
// fixtures/deterministic/pretooluse/V001_passthrough.json
{
  "input": {
    "tool_name": "Read",
    "tool_input": {
      "file_path": "/home/user/project/main.go"
    },
    "session_id": "test-session-v001",
    "hook_event_name": "PreToolUse",
    "captured_at": 1705708800
  },
  "expected": {
    "stdout": "{}",
    "exit_code": 0
  }
}
```

#### V003: Einstein Block

```json
// fixtures/deterministic/pretooluse/V003_einstein_block.json
{
  "input": {
    "tool_name": "Task",
    "tool_input": {
      "model": "opus",
      "prompt": "AGENT: einstein\n\n1. TASK: Deep analysis of routing failure",
      "subagent_type": "general-purpose",
      "description": "Einstein deep analysis"
    },
    "session_id": "test-session-v003",
    "hook_event_name": "PreToolUse",
    "captured_at": 1705708800
  },
  "expected": {
    "decision": "block",
    "reason_contains": "Task(opus) blocked",
    "has_violation_type": "opus_task_blocked",
    "exit_code": 0
  }
}
```

#### V004: Subagent Type Mismatch

```json
// fixtures/deterministic/pretooluse/V004_subagent_mismatch.json
{
  "input": {
    "tool_name": "Task",
    "tool_input": {
      "model": "haiku",
      "prompt": "AGENT: codebase-search\n\nFind all Go files",
      "subagent_type": "general-purpose",
      "description": "Search codebase"
    },
    "session_id": "test-session-v004",
    "hook_event_name": "PreToolUse",
    "captured_at": 1705708800
  },
  "expected": {
    "decision": "block",
    "reason_contains": "Invalid subagent_type",
    "reason_contains_agent": "codebase-search",
    "reason_contains_required": "Explore",
    "exit_code": 0
  }
}
```

### 4.2 SessionEnd Fixtures

#### S001: Clean Session

```json
// fixtures/deterministic/sessionend/S001_clean_session.json
{
  "setup": {
    "create_dirs": [".claude/memory"],
    "files": {
      ".claude/memory/pending-learnings.jsonl": "",
      ".claude/memory/routing-violations.jsonl": ""
    },
    "env": {
      "GOGENT_PROJECT_DIR": "${TEMP_DIR}"
    }
  },
  "input": {
    "session_id": "test-session-s001",
    "hook_event_name": "SessionEnd",
    "captured_at": 1705708800
  },
  "expected": {
    "files_created": [
      ".claude/memory/handoffs.jsonl",
      ".claude/memory/last-handoff.md"
    ],
    "handoff_assertions": {
      "schema_version": "1.1",
      "artifacts.sharp_edges": [],
      "artifacts.routing_violations": [],
      "context.metrics.tool_calls": 0
    },
    "exit_code": 0
  }
}
```

#### S004: Mixed Artifacts

```json
// fixtures/deterministic/sessionend/S004_mixed_artifacts.json
{
  "setup": {
    "create_dirs": [".claude/memory"],
    "files": {
      ".claude/memory/pending-learnings.jsonl": "{\"file\":\"pkg/main.go\",\"error_type\":\"type_mismatch\",\"consecutive_failures\":3,\"timestamp\":1705708000}\n",
      ".claude/memory/routing-violations.jsonl": "{\"agent\":\"python-pro\",\"violation_type\":\"ceiling_exceeded\",\"timestamp\":1705708100}\n{\"agent\":\"orchestrator\",\"violation_type\":\"subagent_mismatch\",\"timestamp\":1705708200}\n",
      ".claude/memory/user-intents.jsonl": "{\"question\":\"Should I use Haiku or Sonnet?\",\"response\":\"Sonnet\",\"source\":\"ask_user\",\"timestamp\":1705708300}\n"
    }
  },
  "input": {
    "session_id": "test-session-s004",
    "hook_event_name": "SessionEnd",
    "captured_at": 1705709000
  },
  "expected": {
    "handoff_assertions": {
      "artifacts.sharp_edges.length": 1,
      "artifacts.routing_violations.length": 2,
      "artifacts.user_intents.length": 1,
      "actions.length_gte": 2
    }
  }
}
```

---

## 5. Randomized Test Protocol

### 5.1 Fuzz Test Implementation

```go
package harness

import (
    "math/rand"
    "time"
)

// FuzzRunner executes randomized tests
type FuzzRunner struct {
    config  SimulationConfig
    gen     Generator
    runner  Runner
    results []SimulationResult
}

// RunFuzz executes N random iterations
func (f *FuzzRunner) RunFuzz() ([]SimulationResult, error) {
    rng := rand.New(rand.NewSource(f.config.FuzzSeed))

    for i := 0; i < f.config.FuzzIterations; i++ {
        seed := rng.Int63()

        // Decide test type (70% PreToolUse, 30% SessionEnd)
        if rng.Float64() < 0.7 {
            result := f.fuzzPreToolUse(seed)
            f.results = append(f.results, result)
        } else {
            result := f.fuzzSessionEnd(seed)
            f.results = append(f.results, result)
        }

        // Check for crashes (save to corpus)
        if !f.results[len(f.results)-1].Passed {
            f.saveCrash(f.results[len(f.results)-1])
        }
    }

    return f.results, nil
}

func (f *FuzzRunner) fuzzPreToolUse(seed int64) SimulationResult {
    event := f.gen.RandomToolEvent(seed)

    // Property-based assertions
    scenario := Scenario{
        ID:          fmt.Sprintf("FUZZ-PRE-%d", seed),
        Category:    "fuzz",
        Description: "Randomized PreToolUse event",
        Input:       event,
        Expected: ExpectedOutput{
            // Invariants that must always hold
            ExitCode: 0, // Should never crash
            // Output must be valid JSON
            // If Task + opus → decision must be "block"
            // If non-Task → output must be "{}"
        },
    }

    return f.runner.RunScenario(scenario)
}

func (f *FuzzRunner) fuzzSessionEnd(seed int64) SimulationResult {
    event := f.gen.RandomSessionEvent(seed)
    metrics := f.gen.RandomSessionMetrics(seed)

    scenario := Scenario{
        ID:          fmt.Sprintf("FUZZ-END-%d", seed),
        Category:    "fuzz",
        Input:       event,
        Setup: func(tmpDir string) error {
            // Create random artifact files
            return setupRandomArtifacts(tmpDir, seed)
        },
        Expected: ExpectedOutput{
            ExitCode: 0,
            // Handoff must be valid JSON
            // schema_version must be "1.1"
            // Artifacts counts must match input files
        },
    }

    return f.runner.RunScenario(scenario)
}
```

### 5.2 Property-Based Invariants

These properties must hold for ALL inputs:

```go
// PreToolUse invariants
var PreToolUseInvariants = []Invariant{
    {
        Name: "never_crash",
        Check: func(input *ToolEvent, output string, exitCode int) bool {
            return exitCode == 0
        },
    },
    {
        Name: "valid_json_output",
        Check: func(input *ToolEvent, output string, exitCode int) bool {
            var v interface{}
            return json.Unmarshal([]byte(output), &v) == nil
        },
    },
    {
        Name: "non_task_passthrough",
        Check: func(input *ToolEvent, output string, exitCode int) bool {
            if input.ToolName != "Task" {
                return output == "{}" || output == "{}\n"
            }
            return true // Not applicable
        },
    },
    {
        Name: "opus_always_blocked",
        Check: func(input *ToolEvent, output string, exitCode int) bool {
            if input.ToolName == "Task" {
                if model, ok := input.ToolInput["model"].(string); ok && model == "opus" {
                    var result map[string]interface{}
                    json.Unmarshal([]byte(output), &result)
                    return result["decision"] == "block"
                }
            }
            return true
        },
    },
    {
        Name: "decision_is_allow_or_block",
        Check: func(input *ToolEvent, output string, exitCode int) bool {
            if input.ToolName != "Task" {
                return true
            }
            var result map[string]interface{}
            if json.Unmarshal([]byte(output), &result) != nil {
                return false
            }
            if decision, ok := result["decision"].(string); ok {
                return decision == "allow" || decision == "block"
            }
            return true // No decision field is valid for pass-through
        },
    },
}

// SessionEnd invariants
var SessionEndInvariants = []Invariant{
    {
        Name: "never_crash",
        Check: func(input *SessionEvent, output string, exitCode int) bool {
            return exitCode == 0
        },
    },
    {
        Name: "handoff_created",
        Check: func(input *SessionEvent, output string, exitCode int) bool {
            // Check handoffs.jsonl was created/appended
            return fileExists(filepath.Join(os.Getenv("GOGENT_PROJECT_DIR"), ".claude/memory/handoffs.jsonl"))
        },
    },
    {
        Name: "schema_version_current",
        Check: func(input *SessionEvent, output string, exitCode int) bool {
            handoff := loadLatestHandoff()
            return handoff.SchemaVersion == "1.1"
        },
    },
    {
        Name: "markdown_created",
        Check: func(input *SessionEvent, output string, exitCode int) bool {
            return fileExists(filepath.Join(os.Getenv("GOGENT_PROJECT_DIR"), ".claude/memory/last-handoff.md"))
        },
    },
}
```

---

## 6. Execution Commands

### 6.1 Run Deterministic Tests

```bash
# Run all deterministic scenarios
go test -tags=simulation ./test/simulation/...

# Run specific category
go run ./test/simulation/harness -mode=deterministic -filter="V00"

# Run with verbose output
go run ./test/simulation/harness -mode=deterministic -verbose
```

### 6.2 Run Fuzz Tests

```bash
# Run 1000 random iterations with fixed seed
go run ./test/simulation/harness -mode=fuzz -iterations=1000 -seed=12345

# Run with timeout
go run ./test/simulation/harness -mode=fuzz -iterations=10000 -timeout=5m

# Replay a specific crash
go run ./test/simulation/harness -replay=test/simulation/fuzz/crashes/crash-42.json
```

### 6.3 Run Mixed Mode

```bash
# Deterministic + 500 fuzz iterations
go run ./test/simulation/harness -mode=mixed -fuzz-iterations=500
```

### 6.4 Generate Report

```bash
# JSON report
go run ./test/simulation/harness -mode=mixed -report=json > reports/simulation-$(date +%Y%m%d).json

# Markdown report
go run ./test/simulation/harness -mode=mixed -report=markdown > reports/simulation-$(date +%Y%m%d).md
```

---

## 7. Expected Output Formats

### 7.1 JSON Report

```json
{
  "run_id": "sim-20260122-143052",
  "config": {
    "mode": "mixed",
    "fuzz_iterations": 500,
    "fuzz_seed": 12345
  },
  "summary": {
    "total": 508,
    "passed": 506,
    "failed": 2,
    "duration_ms": 4523
  },
  "deterministic_results": [
    {
      "scenario_id": "V001",
      "passed": true,
      "duration_ms": 12
    }
  ],
  "fuzz_results": {
    "iterations": 500,
    "crashes": 0,
    "invariant_failures": 2,
    "failures": [
      {
        "seed": 987654,
        "invariant": "decision_is_allow_or_block",
        "input": "...",
        "output": "..."
      }
    ]
  }
}
```

### 7.2 Markdown Report

```markdown
# Simulation Report: sim-20260122-143052

## Summary

| Metric | Value |
|--------|-------|
| Total Tests | 508 |
| Passed | 506 |
| Failed | 2 |
| Duration | 4.52s |

## Deterministic Results

| ID | Category | Result | Duration |
|----|----------|--------|----------|
| V001 | Pass-through | ✅ | 12ms |
| V002 | Valid Task | ✅ | 15ms |
| V003 | Einstein Block | ✅ | 14ms |
...

## Fuzz Results

- **Iterations:** 500
- **Crashes:** 0
- **Invariant Failures:** 2

### Failures

#### Failure 1: FUZZ-PRE-987654

**Invariant:** `decision_is_allow_or_block`

<details>
<summary>Input</summary>

```json
{...}
```

</details>

<details>
<summary>Output</summary>

```json
{...}
```

</details>
```

---

## 8. Future Implementation Integration

### 8.1 Adding New Packages

When adding new packages to GOgent-Fortress:

1. **Update Architecture Diagram** (Section 1.1)
   - Add new package box with dependencies
   - Update data flow arrows

2. **Define New Fixtures** (Section 4)
   ```
   fixtures/deterministic/<hook_type>/
   └── <NewPackageID>_<scenario>.json
   ```

3. **Add Invariants** (Section 5.2)
   ```go
   var NewPackageInvariants = []Invariant{
       {Name: "...", Check: func(...) bool {...}},
   }
   ```

4. **Register in Harness**
   ```go
   // harness/main.go
   func init() {
       RegisterPackage("newpackage", NewPackageInvariants)
   }
   ```

### 8.2 Adding New CLI Commands

When adding new cmd/ binaries:

1. **Create Fixture Directory**
   ```
   fixtures/deterministic/newcmd/
   └── NC001_basic.json
   ```

2. **Define Expected Behavior Matrix**
   - Document all input/output combinations
   - Specify exit codes and error messages

3. **Add to Runner**
   ```go
   // harness/runner.go
   func (r *Runner) runNewCmd(scenario Scenario) SimulationResult {
       cmd := exec.Command("newcmd")
       // ...
   }
   ```

### 8.3 Adding New Telemetry Types

When adding new telemetry (e.g., `pkg/telemetry/newtype.go`):

1. **Add Generator Method**
   ```go
   func (g *Generator) RandomNewType(seed int64) *telemetry.NewType {
       // ...
   }
   ```

2. **Add Aggregate Test Scenario**
   ```json
   // fixtures/deterministic/aggregate/A00X_newtype.json
   {
       "input_files": {...},
       "expected_aggregation": {...}
   }
   ```

### 8.4 Schema Version Upgrades

When bumping handoff schema version:

1. **Add Migration Test**
   ```json
   // fixtures/deterministic/sessionend/S00X_migration_vY_to_vZ.json
   {
       "setup": {
           "files": {
               ".claude/memory/handoffs.jsonl": "{\"schema_version\":\"Y.0\",...}"
           }
       },
       "expected": {
           "handoff_assertions": {
               "schema_version": "Z.0"
           }
       }
   }
   ```

2. **Verify Backward Compatibility**
   - Old readers must ignore new fields
   - New readers must handle missing fields

### 8.5 Integration Test Hooks

For cross-package integration:

```go
// test/simulation/harness/integration.go

// IntegrationScenario chains multiple CLI invocations
type IntegrationScenario struct {
    ID    string
    Steps []IntegrationStep
}

type IntegrationStep struct {
    Cmd      string                 // "validate", "archive", "aggregate"
    Input    interface{}
    Expected ExpectedOutput
    PassTo   string                 // Next step receives this output
}

// Example: Full session lifecycle
var SessionLifecycleScenario = IntegrationScenario{
    ID: "INT001_full_lifecycle",
    Steps: []IntegrationStep{
        {
            Cmd:   "validate",
            Input: TaskEvent{Model: "sonnet", ...},
            Expected: ExpectedOutput{Decision: ptr("allow")},
        },
        {
            Cmd:   "validate",
            Input: TaskEvent{Model: "opus", ...},
            Expected: ExpectedOutput{Decision: ptr("block")},
        },
        {
            Cmd:   "archive",
            Input: SessionEndEvent{...},
            Expected: ExpectedOutput{
                FilesCreated: []string{".claude/memory/handoffs.jsonl"},
            },
        },
        {
            Cmd: "aggregate",
            Input: AggregateRequest{Since: "1h"},
            Expected: ExpectedOutput{
                StdoutContains: "Total Sessions: 1",
            },
        },
    },
}
```

### 8.6 CI/CD Integration

```yaml
# .github/workflows/simulation.yml
name: Simulation Tests

on:
  push:
    branches: [master]
  pull_request:

jobs:
  simulation:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Build
        run: make build

      - name: Run Deterministic Tests
        run: go run ./test/simulation/harness -mode=deterministic

      - name: Run Fuzz Tests
        run: go run ./test/simulation/harness -mode=fuzz -iterations=1000 -seed=${{ github.run_number }}

      - name: Upload Report
        uses: actions/upload-artifact@v4
        with:
          name: simulation-report
          path: reports/
```

### 8.7 Performance Benchmarking Extension

```go
// Future: Add performance tracking to simulation
type PerformanceMetrics struct {
    Scenario      string
    P50Latency    time.Duration
    P99Latency    time.Duration
    MemoryPeakMB  float64
    Throughput    float64 // ops/sec
}

func (r *Runner) BenchmarkScenario(s Scenario, iterations int) PerformanceMetrics {
    // Run N iterations, collect timing
    // Return percentile statistics
}
```

---

## 9. Appendix: Agent-Subagent Mapping Reference

Used for V004-style subagent mismatch tests:

| Agent | Required subagent_type |
|-------|----------------------|
| codebase-search | Explore |
| haiku-scout | Explore |
| code-reviewer | Explore |
| librarian | Explore |
| tech-docs-writer | general-purpose |
| scaffolder | general-purpose |
| memory-archivist | general-purpose |
| python-pro | general-purpose |
| python-ux | general-purpose |
| r-pro | general-purpose |
| r-shiny-pro | general-purpose |
| go-pro | general-purpose |
| go-cli | general-purpose |
| go-tui | general-purpose |
| go-api | general-purpose |
| go-concurrent | general-purpose |
| orchestrator | Plan |
| architect | Plan |
| einstein | general-purpose |
| gemini-slave | Bash |

---

## 10. Quick Start

```bash
# 1. Create test directory structure
mkdir -p test/simulation/{harness,fixtures/{deterministic/{pretooluse,sessionend},expected,schemas},fuzz/{seeds,crashes},reports}

# 2. Copy fixtures from this spec
# (Copy JSON blocks from Section 4 to appropriate files)

# 3. Implement harness/main.go (skeleton provided in Section 3.2)

# 4. Run first test
go run ./test/simulation/harness -mode=deterministic -filter="V001" -verbose

# 5. Iterate on failures
```

---

**Document Version:** 1.0
**Architecture Coverage:** 33% (see Section 1.1 for covered packages)
**Next Update:** After implementing pkg/aggregate or adding new cmd/
