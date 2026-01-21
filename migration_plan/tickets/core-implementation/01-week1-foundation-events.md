# Week 1: Foundation & Event Parsing (GOgent-001 to 009)

**Days 1-2: Project Setup + Event Parsing**
**Total Tickets**: 9 (GOgent-001, 002, 002b, 003, 004a, 006, 007, 008, 008b, 009)
**Estimated Time**: ~11 hours

---

**Conventions**: See [TICKET-TEMPLATE.md](TICKET-TEMPLATE.md) for required structure
**Standards**: See [00-overview.md](00-overview.md) for error handling, testing, logging
**Pre-Work**: GOgent-000 in [00-prework.md](00-prework.md) MUST be complete before starting

---

## Day 1: Foundation (5 tickets, ~6.5 hours)

### GOgent-001: Initialize Go Module and Directory Structure

**Time**: 1 hour
**Dependencies**: GOgent-000 (must have baseline)
**Priority**: HIGH

**Task**:
Create Go module and directory structure for the gogent-fortress project.

**Steps**:

1. Navigate to `/home/doktersmol/Documents/gogent-fortress/`
2. Run `go version` - verify Go 1.21+ installed
3. Run `go mod init github.com/yourusername/gogent-fortress`
   - Replace "yourusername" with your GitHub username

4. Create directory structure:
```bash
mkdir -p cmd/gogent-validate
mkdir -p cmd/gogent-archive
mkdir -p cmd/gogent-sharp-edge
mkdir -p pkg/routing
mkdir -p pkg/session
mkdir -p pkg/memory
mkdir -p pkg/config
mkdir -p internal/logger
mkdir -p test/integration
mkdir -p test/benchmark
mkdir -p test/fixtures
mkdir -p scripts
```

5. Copy event corpus from baseline:
```bash
cp ~/gogent-baseline/event-corpus.json test/fixtures/
cp ~/gogent-baseline/BASELINE.md migration_plan/
```

6. Create .gitignore:
```bash
cat > .gitignore << 'EOF'
# Binaries
gogent-validate
gogent-archive
gogent-sharp-edge
gogent-daemon
gogent

# Test output
*.test
*.out
coverage.txt

# Temporary files
*.tmp
.cache/

# IDE
.vscode/
.idea/
*.swp
*.swo
EOF
```

**Acceptance Criteria**:
- [ ] `go.mod` exists with correct module path (your username substituted)
- [ ] `go.mod` specifies `go 1.21` or higher
- [ ] All directories created (verify with `tree -L 2`)
- [ ] `test/fixtures/event-corpus.json` exists (100 events)
- [ ] `migration_plan/BASELINE.md` exists
- [ ] `.gitignore` exists
- [ ] `go mod tidy` runs without errors (even though no code yet)

**Why This Matters**: Foundation for all subsequent work. Establishes Go module conventions and project structure. Event corpus is critical for testing.

---

### GOgent-002: Define routing.Schema Struct (Complete v2.2.0)

**Time**: 2 hours (revised from 1.5 hours)
**Dependencies**: GOgent-001
**Status**: ✅ IMPLEMENTED (see pkg/routing/schema.go)

**Task**:
Define complete `routing.Schema` struct matching routing-schema.json v2.2.0 with all 23+ struct types, security fields, and semantic validation.

**File**: `pkg/routing/schema.go` (448 lines, generated)

**Implementation Summary**:
Complete struct definitions generated from `~/.claude/routing-schema.json` v2.2.0 (342 lines). Includes all critical v2.2.0 fields identified in gap analysis:

**Key Structs Implemented**:
- `Schema` - Main routing schema (17 top-level keys)
- `TierConfig` - Tier configuration with all 5 tiers
- `TierLevels` - Numeric delegation ceiling levels
- `DelegationCeiling` - Security provenance (SetBy, EnforcedBy, Calculation)
- `SubagentTypesConfig` - All 4 subagent types with:
  - `AllowsWrite` (bool) - Security control for write permissions ✅
  - `RespectsAgentYaml` (bool) - Compliance control ✅
  - `UseFor` ([]string) - Agent compatibility list ✅
  - `Rationale` (string) - Configuration justification ✅
- `BlockedPattern` - Rich objects with Reason/Alternative/CostImpact ✅
- `ScoutProtocol`, `EscalationRules`, `CompoundTriggers`, `CostThresholds`
- 23 total struct types (no "omitted for brevity")

**Methods Implemented**:
- `LoadSchema()` - XDG-compliant loading from `~/.claude/routing-schema.json`
- `Validate()` - Semantic validation (version check, tier names, reference integrity) ✅
- `GetTier()`, `GetTierLevel()`, `GetSubagentTypeForAgent()`
- `GetSubagentType()`, `ValidateAgentSubagentPair()`

**Acceptance Criteria**:
- [x] `pkg/routing/schema.go` exists (448 lines)
- [x] All 23+ struct types defined (no omissions)
- [x] Critical v2.2.0 security fields present (AllowsWrite, RespectsAgentYaml)
- [x] BlockedPattern is struct (not string) with Reason/Alternative/CostImpact
- [x] DelegationCeiling includes SetBy/EnforcedBy/Calculation metadata
- [x] Schema version constant: `EXPECTED_SCHEMA_VERSION = "2.2.0"`
- [x] Validate() method checks version, tier names, reference integrity
- [x] Concrete types used ([]string for Tools, not interface{})
- [x] `go build ./pkg/routing` succeeds
- [x] Tests in schema_test.go pass (71 assertions)

**Why This Matters**:
- Fixes M-1 (incomplete structs) from critical review
- Implements v2.2.0 schema (current production version)
- Adds security controls (AllowsWrite, RespectsAgentYaml)
- Core data model for all routing logic - must match production schema exactly

---

### GOgent-002b: Complete All Schema Struct Definitions

**Time**: N/A (merged into GOgent-002)
**Dependencies**: GOgent-002
**Priority**: HIGH (fixes M-1)
**Status**: ✅ SUPERSEDED - Completed as part of GOgent-002

**Task**:
Complete all remaining nested struct definitions for routing-schema.json. No "omitted for brevity" - contractor needs complete types.

**Resolution**:
This ticket was **merged into GOgent-002** during implementation. The original plan split struct definition into two tickets (basic + complete), but the actual implementation generated all 23 struct types in a single pass using the production routing-schema.json v2.2.0 as the source of truth.

**What was implemented in GOgent-002**:
- All 23+ struct types (not split across tickets)
- All critical v2.2.0 fields from gap analysis:
  - `SubagentType.AllowsWrite`, `RespectsAgentYaml`, `UseFor`, `Rationale`
  - `BlockedPattern` as struct with Reason/Alternative/CostImpact
  - `DelegationCeiling` with SetBy/EnforcedBy/Calculation
- Concrete types throughout (no interface{} overuse)
- Semantic validation via `Validate()` method
- Complete test suite (71 assertions in schema_test.go)

**Files Created**:
- `pkg/routing/schema.go` (448 lines) - All structs + methods
- `pkg/routing/schema_test.go` (377 lines) - Comprehensive tests

**Acceptance Criteria**: (all met in GOgent-002)
- [x] All struct types defined (23 types total)
- [x] No "omitted for brevity" comments remain
- [x] `go build ./pkg/routing` succeeds
- [x] Can unmarshal production routing-schema.json without errors
- [x] TestUnmarshalProductionSchema validates all v2.2.0 fields
- [x] Schema version validation implemented

**Why This Matters**: Fixes M-1 (incomplete structs). All definitions completed in single comprehensive implementation matching production v2.2.0 schema.

---

### GOgent-002c: Schema Semantic Validation (NEW)

**Time**: N/A (implemented as part of GOgent-002)
**Dependencies**: GOgent-002
**Priority**: HIGH (security validation)
**Status**: ✅ IMPLEMENTED (see schema.go:Validate() method)

**Task**:
Add semantic validation to ensure loaded schema is internally consistent and matches expected version.

**File**: `pkg/routing/schema.go` (already implemented)

**Implementation Summary**:
Semantic validation was implemented as part of GOgent-002. The `Validate()` method performs runtime checks beyond JSON unmarshaling to ensure schema integrity.

**Validation Rules Implemented**:

1. **Version Check**:
   ```go
   const EXPECTED_SCHEMA_VERSION = "2.2.0"

   if s.Version != EXPECTED_SCHEMA_VERSION {
       return fmt.Errorf("[routing] Schema version mismatch...")
   }
   ```

2. **Tier Name Validation**:
   - Valid tiers: `haiku`, `haiku_thinking`, `sonnet`, `opus`, `external`
   - Rejects unknown tier names
   - Ensures tier_levels reference existing tiers

3. **Reference Integrity**:
   - Agent-to-subagent mappings reference existing subagent_types
   - Blocked patterns reference valid tiers
   - Escalation rules reference defined agents

**Query Methods** (for routing logic):
- `GetTier(tierName string)` - Retrieve tier configuration
- `GetTierLevel(tierName string)` - Get numeric level for delegation ceiling
- `GetSubagentTypeForAgent(agentName string)` - Agent → subagent_type lookup
- `GetSubagentType(subagentType string)` - Subagent type configuration
- `ValidateAgentSubagentPair(agent, subagent string)` - Pairing validation

**Test Coverage**:
- `TestSchemaValidate()` - Version mismatch, invalid tiers, broken references
- `TestValidateAgentSubagentPair()` - Correct vs incorrect pairings
- `TestGetTier()`, `TestGetTierLevel()` - Query method validation

**Acceptance Criteria**:
- [x] Schema.Validate() method exists
- [x] Version validated against EXPECTED_SCHEMA_VERSION constant
- [x] Tier names validated against known values
- [x] Agent-subagent mapping reference integrity checked
- [x] Clear error messages for inconsistencies (with "[routing]" prefix)
- [x] Query methods tested (GetTier, GetTierLevel, etc.)
- [x] Tests cover version mismatch, invalid tiers, broken references

**Why This Matters**:
- Catches configuration errors at load time (not during execution)
- Prevents routing to non-existent tiers or agents
- Validates security-critical fields (AllowsWrite, delegation ceiling)
- Provides clear error messages for troubleshooting misconfigurations

**Note**: This ticket was originally planned as separate work, but was integrated into GOgent-002 implementation for completeness.

---

### GOgent-003: Define routing.AgentIndex and Config Structs

**Time**: 1 hour
**Dependencies**: GOgent-002b
**Status**: ✅ IMPLEMENTED (see pkg/routing/agents.go)

**Task**:
Define structs for agents-index.json parsing with complete v2.2.0 fields and query methods.

**Files**:
- `pkg/routing/agents.go` (593 lines)
- `pkg/routing/agents_test.go` (804 lines)

**Imports**:
```go
package routing

import (
    "encoding/json"
)
```

**Struct Definition**:
```go
// AgentIndex represents agents-index.json structure
type AgentIndex struct {
    Version      string        `json:"version"`
    GeneratedAt  string        `json:"generated_at"`
    Description  string        `json:"description"`
    Agents       []Agent       `json:"agents"`
    RoutingRules RoutingRules  `json:"routing_rules"`
    StateManagement StateManagement `json:"state_management"`
}

type Agent struct {
    ID                  string      `json:"id"`
    Name                string      `json:"name"`
    Model               string      `json:"model"`
    Thinking            bool        `json:"thinking"`
    ThinkingBudget      int         `json:"thinking_budget,omitempty"`
    ThinkingBudgetComplex int       `json:"thinking_budget_complex,omitempty"`
    Tier                interface{} `json:"tier"` // Can be float64 or string
    Category            string      `json:"category"`
    Path                string      `json:"path"`
    Triggers            []string    `json:"triggers"`
    Tools               []string    `json:"tools"`
    AutoActivate        interface{} `json:"auto_activate"` // Can be null or object
    ConventionsRequired []string    `json:"conventions_required,omitempty"`
    SharpEdgesCount     int         `json:"sharp_edges_count,omitempty"`
    Description         string      `json:"description"`
}

type RoutingRules struct {
    IntentGate        IntentGateConfig        `json:"intent_gate"`
    ScoutFirstProtocol ScoutFirstProtocolConfig `json:"scout_first_protocol"`
    ComplexityRouting ComplexityRoutingConfig `json:"complexity_routing"`
    AutoFire          map[string]string       `json:"auto_fire"`
    ModelTiers        map[string][]string     `json:"model_tiers"`
}

type IntentGateConfig struct {
    Description string   `json:"description"`
    Triggers    []string `json:"triggers"`
}

type ScoutFirstProtocolConfig struct {
    Description string   `json:"description"`
    Triggers    []string `json:"triggers"`
}

type ComplexityRoutingConfig struct {
    Description string                 `json:"description"`
    Tiers       map[string]interface{} `json:"tiers"`
}

type StateManagement struct {
    Description string                 `json:"description"`
    Files       map[string]interface{} `json:"files"`
}
```

**Implementation Summary**:
Complete structs defined from production agents-index.json v2.2.0. Includes all 20+ optional fields:

**Key Structs Implemented**:
- `AgentIndex` - Top-level with Agents, RoutingRules, StateManagement
- `Agent` - Complete with all v2.2.0 fields:
  - Core: ID, Name, Model, Thinking, ThinkingBudget, ThinkingBudgetComplex, Tier, Category, Path, Triggers, Tools, Description
  - Optional: AutoActivate, Inputs, Outputs, ConventionsRequired, SharpEdgesCount, AutoFire, ScoutFirst
  - Planning: OutputArtifacts, InputSources
  - External: Invocation, Protocols, StateFiles
  - Performance: CostPerInvocation, ParallelSafe, SwarmCompatible, OutputFormat, OutputFile, CostCeilingUSD, FallbackFor
- `AutoActivate` - Languages, Patterns, Dependencies, FilePatterns
- `OutputArtifacts` - Required outputs + SpecsLocation
- `StateFiles` - ScoutOutput, ComplexityScore
- `RoutingRules` - IntentGate, ScoutFirstProtocol, ComplexityRouting, AutoFire, ModelTiers
- `StateManagement` - TmpDirectory, Files (with TTL), Cleanup

**Methods Implemented**:
- `LoadAgentIndex()` - XDG-compliant loading
- `Validate()` - Version check, ID uniqueness, reference integrity
- `ValidateAgent()` - Required fields, tier validation
- `GetAgentByID()`, `GetAgentsByTier()`, `GetToolsForAgent()`
- `FindAgentByLanguage()`, `FindAgentByPattern()`, `FindAgentByTrigger()`, `FindAgentByCategory()`
- `GetScoutAgents()`, `GetTierForAgent()`

**Acceptance Criteria**:
- [x] `pkg/routing/agents.go` exists (593 lines) with complete struct definitions
- [x] All 20+ optional fields captured (no data loss)
- [x] Can unmarshal production agents-index.json v2.2.0 (21 agents)
- [x] `go build ./pkg/routing` succeeds
- [x] TestUnmarshalProductionAgentIndex validates all v2.2.0 fields
- [x] All query methods tested (11 test functions, 804 lines)
- [x] Zero data loss from production agents-index.json

**Why This Matters**: Foundation for agent validation and tier lookups in Task() calls. Must capture ALL v2.2.0 fields to prevent data loss during routing validation.

---

### GOgent-004a: Implement Config Loader (LoadRoutingSchema, LoadAgentsIndex)

**Time**: 1.5 hours
**Dependencies**: GOgent-002b, GOgent-003
**Priority**: HIGH (fixes C-1 circular dependency)

**Task**:
Implement functions to load routing-schema.json and agents-index.json from disk. **WITHOUT comprehensive tests** (tests come in GOgent-004c after event parsing is complete).

**File**: `pkg/config/loader.go`

**Imports**:
```go
package config

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"

    "github.com/yourusername/gogent-fortress/pkg/routing"
)
```

**Constants**:
```go
const EXPECTED_SCHEMA_VERSION = "1.0"
```

**Function Implementations**:
```go
// LoadRoutingSchema loads routing-schema.json from ~/.claude/
func LoadRoutingSchema() (*routing.Schema, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return nil, fmt.Errorf("[config] Failed to get home dir: %w", err)
    }

    path := filepath.Join(homeDir, ".claude", "routing-schema.json")
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("[config] Failed to read routing schema at %s: %w. Ensure .claude/ directory exists.", path, err)
    }

    var schema routing.Schema
    if err := json.Unmarshal(data, &schema); err != nil {
        return nil, fmt.Errorf("[config] Failed to parse routing schema: %w. Check JSON syntax.", err)
    }

    // Validate schema version
    if schema.SchemaVersion != EXPECTED_SCHEMA_VERSION {
        return nil, fmt.Errorf(
            "[config] Schema version mismatch. Expected %s, got %s. Update gogent binaries or routing-schema.json.",
            EXPECTED_SCHEMA_VERSION,
            schema.SchemaVersion,
        )
    }

    return &schema, nil
}

// LoadAgentsIndex loads agents-index.json from ~/.claude/agents/
func LoadAgentsIndex() (*routing.AgentIndex, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return nil, fmt.Errorf("[config] Failed to get home dir: %w", err)
    }

    path := filepath.Join(homeDir, ".claude", "agents", "agents-index.json")
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("[config] Failed to read agents index at %s: %w. Ensure .claude/agents/ exists.", path, err)
    }

    var index routing.AgentIndex
    if err := json.Unmarshal(data, &index); err != nil {
        return nil, fmt.Errorf("[config] Failed to parse agents index: %w. Check JSON syntax.", err)
    }

    return &index, nil
}
```

**Basic Test** (minimal - full tests in GOgent-004c):
```go
// pkg/config/loader_test.go
package config

import (
    "testing"
)

func TestLoadRoutingSchema_Basic(t *testing.T) {
    schema, err := LoadRoutingSchema()
    if err != nil {
        t.Fatalf("Failed to load routing schema: %v", err)
    }

    if schema.Version == "" {
        t.Error("Expected version field to be populated")
    }
}

func TestLoadAgentsIndex_Basic(t *testing.T) {
    index, err := LoadAgentsIndex()
    if err != nil {
        t.Fatalf("Failed to load agents index: %v", err)
    }

    if len(index.Agents) == 0 {
        t.Error("Expected agents to be populated")
    }
}
```

**Acceptance Criteria**:
- [ ] `pkg/config/loader.go` exists with both functions
- [ ] Functions return non-nil Schema and AgentIndex for valid files
- [ ] Functions return clear error messages for missing/invalid files
- [ ] Schema version validation implemented
- [ ] Error messages follow format: "[component] What happened. Why. How to fix."
- [ ] Basic tests pass: `go test ./pkg/config`

**Why This Matters**: Central config loading used by all validation logic. Split from GOgent-004 to break circular dependency with event parsing.

---

## Day 2: Event Parsing (5 tickets, ~7 hours)

### GOgent-006: Define ToolEvent Structs

**Time**: 1 hour
**Dependencies**: GOgent-001

**Task**:
Define Go structs to parse JSON events received via STDIN from Claude Code hooks.

**File**: `pkg/routing/events.go`

**Imports**:
```go
package routing

import (
    "encoding/json"
)
```

**Struct Definitions**:
```go
// ToolEvent represents the JSON received on STDIN during PreToolUse hooks
type ToolEvent struct {
    ToolName      string                 `json:"tool_name"`
    ToolInput     map[string]interface{} `json:"tool_input"`
    SessionID     string                 `json:"session_id"`
    HookEventName string                 `json:"hook_event_name"`
    CWD           string                 `json:"cwd"`
    Timestamp     int64                  `json:"timestamp,omitempty"`
}

// TaskInput represents the tool_input when tool_name = "Task"
type TaskInput struct {
    Model         string `json:"model,omitempty"`
    Prompt        string `json:"prompt"`
    SubagentType  string `json:"subagent_type,omitempty"`
    Description   string `json:"description,omitempty"`
    MaxTurns      int    `json:"max_turns,omitempty"`
}

// SessionArchiveEvent represents SessionEnd events for session-archive hook
type SessionArchiveEvent struct {
    SessionID      string `json:"session_id"`
    TranscriptPath string `json:"transcript_path"`
    CWD            string `json:"cwd"`
    HookEventName  string `json:"hook_event_name"`
    Reason         string `json:"reason,omitempty"`
}

// PostToolEvent represents PostToolUse events for sharp-edge-detector
type PostToolEvent struct {
    ToolName      string                 `json:"tool_name"`
    ToolResponse  map[string]interface{} `json:"tool_response"`
    SessionID     string                 `json:"session_id"`
    HookEventName string                 `json:"hook_event_name"`
    FilePath      string                 `json:"file_path,omitempty"` // For Edit/Write tools
}
```

**Acceptance Criteria**:
- [ ] `pkg/routing/events.go` exists with all event types
- [ ] Struct tags match Claude Code JSON format
- [ ] `go build ./pkg/routing` succeeds

**Why This Matters**: Event parsing is foundation for all hooks. Must handle all event types Claude Code sends.

---

### GOgent-007: Implement ParseToolEvent Function

**Time**: 1.5 hours
**Dependencies**: GOgent-006

**Task**:
Parse JSON from STDIN into ToolEvent struct with error handling.

**File**: `pkg/routing/events.go` (continued)

**Function Implementation**:
```go
import (
    "bufio"
    "fmt"
    "io"
    "time"
)

// ParseToolEvent reads JSON from STDIN and parses into ToolEvent
func ParseToolEvent(r io.Reader, timeout time.Duration) (*ToolEvent, error) {
    // Create buffered reader
    reader := bufio.NewReader(r)

    // Read with timeout (fixes M-6)
    type result struct {
        data []byte
        err  error
    }

    ch := make(chan result, 1)
    go func() {
        data, err := io.ReadAll(reader)
        ch <- result{data, err}
    }()

    select {
    case res := <-ch:
        if res.err != nil {
            return nil, fmt.Errorf("[event-parser] Failed to read STDIN: %w. Ensure hook is receiving JSON input.", res.err)
        }

        var event ToolEvent
        if err := json.Unmarshal(res.data, &event); err != nil {
            return nil, fmt.Errorf("[event-parser] Failed to parse JSON: %w. Check STDIN format: %s", err, string(res.data[:min(100, len(res.data))]))
        }

        return &event, nil

    case <-time.After(timeout):
        return nil, fmt.Errorf("[event-parser] STDIN read timeout after %v. Hook may be stuck waiting for input.", timeout)
    }
}

// ParseTaskInput extracts Task-specific parameters from tool_input
func ParseTaskInput(toolInput map[string]interface{}) (*TaskInput, error) {
    data, err := json.Marshal(toolInput)
    if err != nil {
        return nil, fmt.Errorf("[event-parser] Failed to marshal tool_input: %w", err)
    }

    var taskInput TaskInput
    if err := json.Unmarshal(data, &taskInput); err != nil {
        return nil, fmt.Errorf("[event-parser] Failed to parse Task input: %w. Check tool_input structure.", err)
    }

    return &taskInput, nil
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
```

**Acceptance Criteria**:
- [ ] `ParseToolEvent()` reads JSON from STDIN successfully
- [ ] Function returns clear errors for invalid JSON
- [ ] Timeout prevents hanging on missing input (default 5s)
- [ ] `ParseTaskInput()` extracts Task parameters correctly
- [ ] Error messages follow format: "[component] What. Why. How to fix."

**Why This Matters**: Robust parsing prevents silent failures. Timeout fixes M-6 (hanging hooks).

---

### GOgent-008: Event Parsing Unit Tests

**Time**: 1.5 hours
**Dependencies**: GOgent-007

**Task**:
Create comprehensive unit tests for event parsing with real Claude Code JSON examples.

**File**: `pkg/routing/events_test.go`

**Test Implementation**:
```go
package routing

import (
    "strings"
    "testing"
    "time"
)

func TestParseToolEvent_ValidTask(t *testing.T) {
    json := `{
        "tool_name": "Task",
        "tool_input": {
            "model": "sonnet",
            "prompt": "AGENT: python-pro\n\nImplement function",
            "subagent_type": "general-purpose"
        },
        "session_id": "test-123",
        "hook_event_name": "PreToolUse"
    }`

    reader := strings.NewReader(json)
    event, err := ParseToolEvent(reader, 5*time.Second)

    if err != nil {
        t.Fatalf("Expected no error, got: %v", err)
    }

    if event.ToolName != "Task" {
        t.Errorf("Expected tool_name Task, got: %s", event.ToolName)
    }

    if event.SessionID != "test-123" {
        t.Errorf("Expected session_id test-123, got: %s", event.SessionID)
    }
}

func TestParseToolEvent_InvalidJSON(t *testing.T) {
    json := `{"invalid": json}`

    reader := strings.NewReader(json)
    _, err := ParseToolEvent(reader, 5*time.Second)

    if err == nil {
        t.Error("Expected error for invalid JSON, got nil")
    }

    // Verify error message includes diagnostic info
    if !strings.Contains(err.Error(), "[event-parser]") {
        t.Errorf("Expected error to include component tag, got: %v", err)
    }
}

func TestParseTaskInput_Complete(t *testing.T) {
    toolInput := map[string]interface{}{
        "model":          "haiku",
        "prompt":         "Search for auth files",
        "subagent_type":  "Explore",
        "description":    "Find authentication",
    }

    taskInput, err := ParseTaskInput(toolInput)
    if err != nil {
        t.Fatalf("Expected no error, got: %v", err)
    }

    if taskInput.Model != "haiku" {
        t.Errorf("Expected model haiku, got: %s", taskInput.Model)
    }

    if taskInput.SubagentType != "Explore" {
        t.Errorf("Expected subagent_type Explore, got: %s", taskInput.SubagentType)
    }
}

func TestParseToolEvent_Timeout(t *testing.T) {
    // Reader that never provides data
    reader := &slowReader{delay: 10 * time.Second}

    _, err := ParseToolEvent(reader, 100*time.Millisecond)

    if err == nil {
        t.Error("Expected timeout error, got nil")
    }

    if !strings.Contains(err.Error(), "timeout") {
        t.Errorf("Expected timeout error, got: %v", err)
    }
}

// slowReader simulates slow STDIN (for timeout testing)
type slowReader struct {
    delay time.Duration
}

func (r *slowReader) Read(p []byte) (n int, err error) {
    time.Sleep(r.delay)
    return 0, nil
}
```

**Acceptance Criteria**:
- [ ] `go test ./pkg/routing` passes all tests
- [ ] Tests cover: valid JSON, invalid JSON, timeout, missing fields
- [ ] Tests use real Claude Code JSON examples
- [ ] Coverage ≥80% for events.go

**Why This Matters**: Parsing bugs cause silent hook failures. Comprehensive tests prevent regressions.

---

### GOgent-008b: Capture Real Event Corpus During Week 1

**Time**: 30 minutes (passive - runs during development)
**Dependencies**: GOgent-000
**Priority**: HIGH (fixes C-3)

**Task**:
Install corpus logger hook to capture 100 real Claude Code events during Week 1 development work. These events will be used for regression testing.

**File**: Already created in GOgent-000 as `~/.claude/hooks/zzz-corpus-logger.sh`

**Steps**:

1. **Verify logger is active**:
```bash
# Check if corpus logger hook exists
ls -lah ~/.claude/hooks/zzz-corpus-logger.sh

# Verify it's executable
chmod +x ~/.claude/hooks/zzz-corpus-logger.sh

# Check corpus collection
wc -l ~/.cache/gogent/event-corpus-raw.jsonl
```

2. **Monitor corpus growth during Week 1**:
```bash
# Run this daily to check progress
echo "Events captured: $(wc -l < ~/.cache/gogent/event-corpus-raw.jsonl)"
jq -s 'group_by(.tool_name) | map({tool: .[0].tool_name, count: length})' \
    ~/.cache/gogent/event-corpus-raw.jsonl
```

3. **End of Week 1: Curate to 100 events**:
```bash
cd ~/gogent-baseline

# Count events by type
jq -s 'group_by(.tool_name) | map({tool: .[0].tool_name, count: length})' \
    ~/.cache/gogent/event-corpus-raw.jsonl > event-distribution-week1.json

# Select 100 diverse events (target distribution from GOgent-000)
# Task: 25, Read: 20, Write: 15, Edit: 15, Bash: 10, Glob: 10, Grep: 5

# Create curated corpus
cat ~/.cache/gogent/event-corpus-raw.jsonl | \
    jq -s '[
        (.[] | select(.tool_name == "Task"))[0:25],
        (.[] | select(.tool_name == "Read"))[0:20],
        (.[] | select(.tool_name == "Write"))[0:15],
        (.[] | select(.tool_name == "Edit"))[0:15],
        (.[] | select(.tool_name == "Bash"))[0:10],
        (.[] | select(.tool_name == "Glob"))[0:10],
        (.[] | select(.tool_name == "Grep"))[0:5]
    ] | flatten' > event-corpus.json

# Copy to project
cp event-corpus.json /home/doktersmol/Documents/gogent-fortress/test/fixtures/

# Remove logger hook (no longer needed)
rm ~/.claude/hooks/zzz-corpus-logger.sh
```

**Acceptance Criteria**:
- [ ] Corpus logger hook active during Week 1
- [ ] ≥100 events captured by end of Week 1
- [ ] Curated corpus in `test/fixtures/event-corpus.json`
- [ ] Distribution matches target (Task: 25, Read: 20, etc.)
- [ ] No sensitive data in corpus (manual review)
- [ ] Corpus includes edge cases (opus blocking, ceiling violations, etc.)
- [ ] Logger hook removed after curation

**Deliverables**:
```
~/gogent-baseline/
├── event-corpus.json              # 100 curated events
└── event-distribution-week1.json  # Type distribution stats

/home/doktersmol/Documents/gogent-fortress/test/fixtures/
└── event-corpus.json              # Copy for integration tests
```

**Why This Matters**: Fixes C-3 (missing test corpus). Real events expose edge cases synthetic tests miss.

---

### GOgent-009: Test Event Parsing with Real Events

**Time**: 1 hour
**Dependencies**: GOgent-008, GOgent-008b

**Task**:
Test event parsing against the captured real event corpus to verify it handles all production cases.

**File**: `pkg/routing/events_integration_test.go`

**Test Implementation**:
```go
package routing

import (
    "encoding/json"
    "os"
    "strings"
    "testing"
    "time"
)

func TestParseToolEvent_RealCorpus(t *testing.T) {
    // Load real event corpus
    corpusPath := "../../test/fixtures/event-corpus.json"
    data, err := os.ReadFile(corpusPath)
    if err != nil {
        t.Skipf("Skipping corpus test: %v", err)
    }

    var events []json.RawMessage
    if err := json.Unmarshal(data, &events); err != nil {
        t.Fatalf("Failed to parse corpus: %v", err)
    }

    // Parse each event
    successCount := 0
    for i, rawEvent := range events {
        reader := strings.NewReader(string(rawEvent))
        event, err := ParseToolEvent(reader, 5*time.Second)

        if err != nil {
            t.Errorf("Event %d failed to parse: %v", i, err)
            continue
        }

        // Validate required fields
        if event.ToolName == "" {
            t.Errorf("Event %d missing tool_name", i)
        }
        if event.SessionID == "" {
            t.Errorf("Event %d missing session_id", i)
        }

        successCount++
    }

    // Require 100% success rate
    if successCount != len(events) {
        t.Errorf("Only %d/%d events parsed successfully", successCount, len(events))
    } else {
        t.Logf("✓ Successfully parsed all %d real events", successCount)
    }
}

func TestParseTaskInput_RealCorpus(t *testing.T) {
    // Load corpus and filter Task events
    corpusPath := "../../test/fixtures/event-corpus.json"
    data, err := os.ReadFile(corpusPath)
    if err != nil {
        t.Skipf("Skipping corpus test: %v", err)
    }

    var events []ToolEvent
    if err := json.Unmarshal(data, &events); err != nil {
        t.Fatalf("Failed to parse corpus: %v", err)
    }

    // Parse Task events
    taskCount := 0
    for i, event := range events {
        if event.ToolName != "Task" {
            continue
        }

        taskInput, err := ParseTaskInput(event.ToolInput)
        if err != nil {
            t.Errorf("Event %d Task input parse failed: %v", i, err)
            continue
        }

        // Validate Task input has prompt
        if taskInput.Prompt == "" {
            t.Errorf("Event %d Task missing prompt", i)
        }

        taskCount++
    }

    t.Logf("✓ Successfully parsed %d Task events", taskCount)
}
```

**Acceptance Criteria**:
- [ ] `go test ./pkg/routing -run Integration` passes
- [ ] 100% of corpus events parse successfully
- [ ] All Task events extract TaskInput correctly
- [ ] Test logs success rate and counts
- [ ] Failures include diagnostic info (event index, error)

**Why This Matters**: Real production events have edge cases synthetic tests miss. Validates parser robustness.

---

## Summary: Days 1-2 Complete

**Deliverables**:
- ✅ Go module initialized with proper structure
- ✅ Complete schema and agent struct definitions (no "omitted for brevity")
- ✅ Config loader with version validation
- ✅ Event parsing with timeout protection
- ✅ Comprehensive test coverage (unit + integration)
- ✅ Real event corpus captured for regression testing

**Next Steps**: Proceed to [02-week1-overrides-permissions.md](02-week1-overrides-permissions.md) (GOgent-010 to 019)

---

**Cross-References**:
- **Overview**: [00-overview.md](00-overview.md) - Testing strategy, error standards
- **Template**: [TICKET-TEMPLATE.md](TICKET-TEMPLATE.md) - Required structure
- **Pre-Work**: [00-prework.md](00-prework.md) - GOgent-000 baseline
- **Navigation**: [README.md](README.md) - File index

---

**Status**: ✅ Ready for implementation
**Last Updated**: 2026-01-15
**Tickets**: 9 (GOgent-001, 002, 002b, 003, 004a, 006, 007, 008, 008b, 009)
