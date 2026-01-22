# Simulation Test Harness Tickets Specification

> **Source:** Architect breakdown of `docs/system-simulation-spec.md`
> **Generated:** 2026-01-22
> **Tickets:** GOgent-030l through GOgent-030v (11 tickets)
> **Total Estimated Hours:** 26.0

---

## Dependency Graph

```
030l (directory structure)
  ↓
030m (core interfaces)
  ↓
030n (generator) ────────┐
  ↓                      ↓
030o (runner)         030p (PreToolUse fixtures)
  ↓                      ↓
030r (invariants)     030q (SessionEnd fixtures)
  ↓
030s (fuzz runner)
  ↓
030t (reporter)
  ↓
030u (CLI entry)
  ↓
030v (CI/CD workflow)
```

---

## Ticket Specifications

### GOgent-030l: Directory Structure & Test Schemas

```yaml
ticket_id: GOgent-030l
title: "Directory Structure & Test Schemas"
status: pending
dependencies: []
estimated_hours: 1.0
phase: 3
priority: MEDIUM
```

**Description:** Create the foundational directory structure for the simulation test harness and test versions of routing-schema.json and agents-index.json.

**Files to Create:**
- `test/simulation/harness/.gitkeep`
- `test/simulation/fixtures/deterministic/pretooluse/.gitkeep`
- `test/simulation/fixtures/deterministic/sessionend/.gitkeep`
- `test/simulation/fixtures/expected/pretooluse/.gitkeep`
- `test/simulation/fixtures/expected/sessionend/.gitkeep`
- `test/simulation/fixtures/schemas/routing-schema.json`
- `test/simulation/fixtures/schemas/agents-index.json`
- `test/simulation/fuzz/seeds/.gitkeep`
- `test/simulation/fuzz/crashes/.gitkeep`
- `test/simulation/reports/.gitkeep`

**Acceptance Criteria:**
- [ ] All directory structure created per spec Section 3.1
- [ ] Test schemas provide isolated environment
- [ ] Test schemas define einstein blocking, subagent type mappings
- [ ] `make test-ecosystem` passes

---

### GOgent-030m: Core Harness Interfaces

```yaml
ticket_id: GOgent-030m
title: "Core Harness Interfaces"
status: pending
dependencies: [GOgent-030l]
estimated_hours: 2.0
phase: 3
priority: MEDIUM
```

**Description:** Define core Go interfaces: SimulationConfig, Scenario, ExpectedOutput, SimulationResult, Runner interface.

**Files to Create:**
- `test/simulation/harness/types.go`
- `test/simulation/harness/types_test.go`

**Key Types (from spec Section 3.2):**

```go
type SimulationConfig struct {
    Mode           string        `json:"mode"`            // "deterministic", "fuzz", "mixed"
    ScenarioFilter []string      `json:"scenario_filter"`
    FuzzIterations int           `json:"fuzz_iterations"`
    FuzzSeed       int64         `json:"fuzz_seed"`
    FuzzTimeout    time.Duration `json:"fuzz_timeout"`
    SchemaPath     string        `json:"schema_path"`
    AgentsPath     string        `json:"agents_path"`
    TempDir        string        `json:"temp_dir"`
    ReportFormat   string        `json:"report_format"`   // "json", "markdown", "tap"
    Verbose        bool          `json:"verbose"`
}

type Scenario struct {
    ID          string
    Category    string
    Description string
    Input       interface{}
    Setup       SetupFunc
    Expected    ExpectedOutput
    Teardown    TeardownFunc
}

type ExpectedOutput struct {
    Decision     *string
    ReasonMatch  *regexp.Regexp
    HasViolation *string
    HandoffFields map[string]interface{}
    FilesCreated  []string
    ExitCode     int
    StderrMatch  *regexp.Regexp
}

type SimulationResult struct {
    ScenarioID  string
    Passed      bool
    Duration    time.Duration
    Input       string
    Output      string
    Expected    string
    Diff        string
    Error       error
}

type Runner interface {
    Run(cfg SimulationConfig) ([]SimulationResult, error)
    RunScenario(s Scenario) SimulationResult
}
```

**Acceptance Criteria:**
- [ ] All types defined with JSON tags
- [ ] SetupFunc/TeardownFunc types defined
- [ ] `go test ./test/simulation/harness/...` passes

---

### GOgent-030n: Generator Implementation

```yaml
ticket_id: GOgent-030n
title: "Generator Implementation"
status: pending
dependencies: [GOgent-030m]
estimated_hours: 3.0
phase: 3
priority: MEDIUM
```

**Description:** Implement Generator interface for deterministic and randomized test inputs.

**Files to Create:**
- `test/simulation/harness/generator.go`
- `test/simulation/harness/generator_test.go`

**Key Interface (from spec Section 3.3):**

```go
type Generator interface {
    GenerateToolEvent(scenarioID string) (*routing.ToolEvent, error)
    GenerateSessionEvent(scenarioID string) (*session.SessionEvent, error)
    RandomToolEvent(seed int64) *routing.ToolEvent
    RandomTaskInput(seed int64) *routing.TaskInput
    RandomSessionEvent(seed int64) *session.SessionEvent
    RandomSessionMetrics(seed int64) *session.SessionMetrics
    GenerateWithParams(params FuzzParams) interface{}
}

type FuzzParams struct {
    ToolNameWeights     map[string]float64
    ModelWeights        map[string]float64
    SubagentTypeWeights map[string]float64
    AgentList           []string
    PromptLengthMean    int
    PromptLengthMax     int
    ErrorRate           float64
    ViolationRate       float64
}
```

**Acceptance Criteria:**
- [ ] Deterministic generation loads from fixtures
- [ ] Random generation uses seeded RNG
- [ ] FuzzParams allows distribution overrides
- [ ] Tests verify reproducibility

---

### GOgent-030o: Runner Implementation

```yaml
ticket_id: GOgent-030o
title: "Runner Implementation"
status: pending
dependencies: [GOgent-030m, GOgent-030n]
estimated_hours: 3.5
phase: 3
priority: MEDIUM
```

**Description:** Implement scenario execution engine that runs CLI commands, captures output, validates results.

**Files to Create:**
- `test/simulation/harness/runner.go`
- `test/simulation/harness/runner_test.go`

**Acceptance Criteria:**
- [ ] RunScenario() executes single scenario
- [ ] Calls Setup func before execution
- [ ] Pipes input to STDIN of CLI commands
- [ ] Captures STDOUT, STDERR, exit code
- [ ] Validates output against ExpectedOutput
- [ ] Calls Teardown func after execution
- [ ] Generates diff on validation failure
- [ ] Timeout protection for hung processes

---

### GOgent-030p: PreToolUse Fixtures (V001-V008)

```yaml
ticket_id: GOgent-030p
title: "PreToolUse Fixtures (V001-V008)"
status: pending
dependencies: [GOgent-030l, GOgent-030n]
estimated_hours: 2.5
phase: 3
priority: MEDIUM
```

**Description:** Create 8 deterministic test fixtures for gogent-validate scenarios.

**Fixtures to Create (spec Section 4.1):**

| ID | Category | Input | Expected |
|----|----------|-------|----------|
| V001 | Pass-through | Non-Task tool (Read) | `{}` |
| V002 | Valid Task | Correct subagent_type | `{decision: "allow"}` |
| V003 | Einstein Block | `model: "opus"` | `{decision: "block"}` |
| V004 | Subagent Mismatch | codebase-search + general-purpose | block |
| V005 | Ceiling Violation | sonnet when ceiling=haiku | block |
| V006 | Model Warning | Model mismatch | allow + warning |
| V007 | Unknown Agent | Unknown agent name | block |
| V008 | Empty Prompt | Empty prompt field | block |

**Files to Create:**
- `test/simulation/fixtures/deterministic/pretooluse/V001_passthrough.json`
- `test/simulation/fixtures/deterministic/pretooluse/V002_valid_task.json`
- `test/simulation/fixtures/deterministic/pretooluse/V003_einstein_block.json`
- `test/simulation/fixtures/deterministic/pretooluse/V004_subagent_mismatch.json`
- `test/simulation/fixtures/deterministic/pretooluse/V005_ceiling_violation.json`
- `test/simulation/fixtures/deterministic/pretooluse/V006_model_warning.json`
- `test/simulation/fixtures/deterministic/pretooluse/V007_unknown_agent.json`
- `test/simulation/fixtures/deterministic/pretooluse/V008_empty_prompt.json`
- Corresponding expected files in `fixtures/expected/pretooluse/`

**Fixture Format:**
```json
{
  "input": {
    "tool_name": "Task",
    "tool_input": {...},
    "session_id": "test-session-xxx",
    "hook_event_name": "PreToolUse",
    "captured_at": 1705708800
  },
  "expected": {
    "decision": "block",
    "reason_contains": "...",
    "exit_code": 0
  }
}
```

---

### GOgent-030q: SessionEnd Fixtures (S001-S008)

```yaml
ticket_id: GOgent-030q
title: "SessionEnd Fixtures (S001-S008)"
status: pending
dependencies: [GOgent-030l, GOgent-030n]
estimated_hours: 2.5
phase: 3
priority: MEDIUM
```

**Description:** Create 8 deterministic test fixtures for gogent-archive scenarios.

**Fixtures to Create (spec Section 4.2):**

| ID | Category | Setup | Expected |
|----|----------|-------|----------|
| S001 | Clean Session | Zero artifacts | Empty handoff |
| S002 | With Violations | 3 violations logged | violations array |
| S003 | With Sharp Edges | 2 sharp edges | sharp_edges array |
| S004 | Mixed Artifacts | Multiple types | Complete handoff |
| S005 | Git Dirty | Uncommitted changes | GitInfo.IsDirty=true |
| S006 | No Git | Non-git directory | GitInfo={} |
| S007 | Active Ticket | .ticket-current | Context.ActiveTicket |
| S008 | Schema Migration | v1.0 handoff | Migrated to v1.1 |

**Files to Create:**
- `test/simulation/fixtures/deterministic/sessionend/S001_clean_session.json`
- `test/simulation/fixtures/deterministic/sessionend/S002_with_violations.json`
- `test/simulation/fixtures/deterministic/sessionend/S003_with_sharp_edges.json`
- `test/simulation/fixtures/deterministic/sessionend/S004_mixed_artifacts.json`
- `test/simulation/fixtures/deterministic/sessionend/S005_git_dirty.json`
- `test/simulation/fixtures/deterministic/sessionend/S006_no_git.json`
- `test/simulation/fixtures/deterministic/sessionend/S007_active_ticket.json`
- `test/simulation/fixtures/deterministic/sessionend/S008_schema_migration.json`
- Corresponding expected files

**Fixture Format:**
```json
{
  "setup": {
    "create_dirs": [".claude/memory"],
    "files": {
      ".claude/memory/routing-violations.jsonl": "..."
    },
    "env": {"GOGENT_PROJECT_DIR": "${TEMP_DIR}"}
  },
  "input": {
    "session_id": "test-session-xxx",
    "hook_event_name": "SessionEnd",
    "captured_at": 1705708800
  },
  "expected": {
    "files_created": [".claude/memory/handoffs.jsonl"],
    "handoff_assertions": {"schema_version": "1.1"},
    "exit_code": 0
  }
}
```

---

### GOgent-030r: Invariant Definitions

```yaml
ticket_id: GOgent-030r
title: "Invariant Definitions"
status: pending
dependencies: [GOgent-030m]
estimated_hours: 2.0
phase: 3
priority: MEDIUM
```

**Description:** Define property-based invariants that must hold for ALL inputs (spec Section 5.2).

**Files to Create:**
- `test/simulation/harness/invariants.go`
- `test/simulation/harness/invariants_test.go`

**PreToolUse Invariants (5):**
1. `never_crash` - exit code == 0
2. `valid_json_output` - JSON unmarshal succeeds
3. `non_task_passthrough` - non-Task tools return `{}`
4. `opus_always_blocked` - Task(opus) always blocked
5. `decision_is_allow_or_block` - Task decisions are valid

**SessionEnd Invariants (4):**
1. `never_crash` - exit code == 0
2. `handoff_created` - handoffs.jsonl exists
3. `schema_version_current` - version is "1.1"
4. `markdown_created` - last-handoff.md exists

**Implementation:**
```go
type Invariant struct {
    Name  string
    Check func(input interface{}, output string, exitCode int) bool
}

var PreToolUseInvariants = []Invariant{...}
var SessionEndInvariants = []Invariant{...}
```

---

### GOgent-030s: Fuzz Runner Implementation

```yaml
ticket_id: GOgent-030s
title: "Fuzz Runner Implementation"
status: pending
dependencies: [GOgent-030n, GOgent-030o, GOgent-030r]
estimated_hours: 3.0
phase: 3
priority: MEDIUM
```

**Description:** Implement randomized fuzzing with seed control, crash corpus capture, invariant checking.

**Files to Create:**
- `test/simulation/harness/fuzz.go`
- `test/simulation/harness/fuzz_test.go`

**Implementation (from spec Section 5.1):**
```go
type FuzzRunner struct {
    config  SimulationConfig
    gen     Generator
    runner  Runner
    results []SimulationResult
}

func (f *FuzzRunner) RunFuzz() ([]SimulationResult, error) {
    rng := rand.New(rand.NewSource(f.config.FuzzSeed))

    for i := 0; i < f.config.FuzzIterations; i++ {
        seed := rng.Int63()

        if rng.Float64() < 0.7 {
            result := f.fuzzPreToolUse(seed)
            f.results = append(f.results, result)
        } else {
            result := f.fuzzSessionEnd(seed)
            f.results = append(f.results, result)
        }

        if !f.results[len(f.results)-1].Passed {
            f.saveCrash(f.results[len(f.results)-1])
        }
    }
    return f.results, nil
}
```

**Acceptance Criteria:**
- [ ] 70/30 split PreToolUse/SessionEnd
- [ ] All invariants checked per execution
- [ ] Failed inputs saved to fuzz/crashes/
- [ ] Crash filename includes seed for replay

---

### GOgent-030t: Reporter Implementation

```yaml
ticket_id: GOgent-030t
title: "Reporter Implementation"
status: pending
dependencies: [GOgent-030m]
estimated_hours: 2.5
phase: 3
priority: MEDIUM
```

**Description:** Implement result aggregation and report generation (JSON, Markdown, TAP).

**Files to Create:**
- `test/simulation/harness/reporter.go`
- `test/simulation/harness/reporter_test.go`

**JSON Report Format (spec Section 7.1):**
```json
{
  "run_id": "sim-20260122-143052",
  "config": {...},
  "summary": {
    "total": 508,
    "passed": 506,
    "failed": 2,
    "duration_ms": 4523
  },
  "deterministic_results": [...],
  "fuzz_results": {
    "iterations": 500,
    "crashes": 0,
    "invariant_failures": 2,
    "failures": [...]
  }
}
```

**Markdown Report Format (spec Section 7.2):**
- Summary table
- Deterministic results table
- Fuzz results with collapsible failure details

---

### GOgent-030u: CLI Entry Point

```yaml
ticket_id: GOgent-030u
title: "CLI Entry Point"
status: pending
dependencies: [GOgent-030m, GOgent-030n, GOgent-030o, GOgent-030p, GOgent-030q, GOgent-030r, GOgent-030s, GOgent-030t]
estimated_hours: 2.5
phase: 3
priority: MEDIUM
```

**Description:** Implement main.go CLI with flag parsing and orchestration.

**Files to Create:**
- `test/simulation/harness/main.go`

**CLI Flags (spec Section 6):**
- `-mode` : deterministic, fuzz, mixed (default: deterministic)
- `-filter` : scenario ID prefix filter
- `-iterations` : fuzz iteration count (default: 1000)
- `-seed` : reproducible random seed
- `-timeout` : fuzz timeout duration (default: 5m)
- `-report` : output format json/markdown/tap (default: json)
- `-verbose` : detailed output
- `-replay` : replay specific crash file

**Usage Examples:**
```bash
go run ./test/simulation/harness -mode=deterministic
go run ./test/simulation/harness -mode=fuzz -iterations=1000 -seed=12345
go run ./test/simulation/harness -mode=mixed -report=markdown > report.md
go run ./test/simulation/harness -replay=fuzz/crashes/crash-42.json
```

---

### GOgent-030v: CI/CD Workflow Integration

```yaml
ticket_id: GOgent-030v
title: "CI/CD Workflow Integration"
status: pending
dependencies: [GOgent-030u]
estimated_hours: 1.5
phase: 3
priority: MEDIUM
```

**Description:** Create GitHub Actions workflow for simulation tests.

**Files to Create:**
- `.github/workflows/simulation.yml`
- Update `Makefile` with `test-simulation` target

**Workflow (spec Section 8.6):**
```yaml
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
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - run: make build
      - run: go run ./test/simulation/harness -mode=deterministic
      - run: go run ./test/simulation/harness -mode=fuzz -iterations=1000 -seed=${{ github.run_number }}
      - uses: actions/upload-artifact@v4
        with:
          name: simulation-report
          path: test/simulation/reports/
          retention-days: 30
```

**Makefile Target:**
```makefile
.PHONY: test-simulation
test-simulation: build
	go run ./test/simulation/harness -mode=mixed -iterations=500
```

---

## tickets-index.json Entries

Add these 11 entries to `dev/will/migration_plan/tickets/tickets-index.json`:

```json
{"id": "GOgent-030l", "title": "Directory Structure & Test Schemas", "file": "tickets/session_archive/030l.md", "time_estimate": "1.0h", "dependencies": [], "status": "pending"},
{"id": "GOgent-030m", "title": "Core Harness Interfaces", "file": "tickets/session_archive/030m.md", "time_estimate": "2.0h", "dependencies": ["GOgent-030l"], "status": "pending"},
{"id": "GOgent-030n", "title": "Generator Implementation", "file": "tickets/session_archive/030n.md", "time_estimate": "3.0h", "dependencies": ["GOgent-030m"], "status": "pending"},
{"id": "GOgent-030o", "title": "Runner Implementation", "file": "tickets/session_archive/030o.md", "time_estimate": "3.5h", "dependencies": ["GOgent-030m", "GOgent-030n"], "status": "pending"},
{"id": "GOgent-030p", "title": "PreToolUse Fixtures (V001-V008)", "file": "tickets/session_archive/030p.md", "time_estimate": "2.5h", "dependencies": ["GOgent-030l", "GOgent-030n"], "status": "pending"},
{"id": "GOgent-030q", "title": "SessionEnd Fixtures (S001-S008)", "file": "tickets/session_archive/030q.md", "time_estimate": "2.5h", "dependencies": ["GOgent-030l", "GOgent-030n"], "status": "pending"},
{"id": "GOgent-030r", "title": "Invariant Definitions", "file": "tickets/session_archive/030r.md", "time_estimate": "2.0h", "dependencies": ["GOgent-030m"], "status": "pending"},
{"id": "GOgent-030s", "title": "Fuzz Runner Implementation", "file": "tickets/session_archive/030s.md", "time_estimate": "3.0h", "dependencies": ["GOgent-030n", "GOgent-030o", "GOgent-030r"], "status": "pending"},
{"id": "GOgent-030t", "title": "Reporter Implementation", "file": "tickets/session_archive/030t.md", "time_estimate": "2.5h", "dependencies": ["GOgent-030m"], "status": "pending"},
{"id": "GOgent-030u", "title": "CLI Entry Point", "file": "tickets/session_archive/030u.md", "time_estimate": "2.5h", "dependencies": ["GOgent-030m", "GOgent-030n", "GOgent-030o", "GOgent-030p", "GOgent-030q", "GOgent-030r", "GOgent-030s", "GOgent-030t"], "status": "pending"},
{"id": "GOgent-030v", "title": "CI/CD Workflow Integration", "file": "tickets/session_archive/030v.md", "time_estimate": "1.5h", "dependencies": ["GOgent-030u"], "status": "pending"}
```

---

## Next Steps

In a fresh session:
1. Read this file
2. Read `docs/system-simulation-spec.md` for detailed reference
3. Create 11 individual ticket files from the specifications above
4. Update tickets-index.json with the entries
