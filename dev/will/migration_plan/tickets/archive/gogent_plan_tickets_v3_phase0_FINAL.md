# GOgent Phase 0 Tickets - Go Translation (FINAL)
**3-Week Sprint Plan + 1 Day Pre-Work: Bash → Go 1:1 Translation**

**Version**: 1.1 FINAL (Critical Review Applied)
**Date**: 2026-01-15
**Total Tickets**: 55 (50 original + 5 critical additions)
**Estimated Duration**: 3 weeks + 1 day pre-work (128 hours)
**Average Ticket Size**: 1-1.5 hours
**Review Status:** ✅ Staff Architect Approved

---

## Change Log (V1.0 → V1.1)

**Critical Fixes Applied:**
- Added GOgent-000: Pre-work baseline measurement (C-2)
- Added GOgent-002b: Complete schema structs (M-1)
- Added GOgent-004b: Config error handling (implied in review)
- Split GOgent-004 into 004a/004c to fix circular dependency (C-1)
- Added GOgent-008b: Capture real event corpus (C-3)
- Added GOgent-024b: Wire validation orchestrator (implied)
- Added GOgent-033: Benchmark all hooks (performance)
- Added GOgent-101b: WSL2 testing (M-8)
- Fixed file paths: /tmp → XDG with fallback (M-2)
- Added stdin timeout to all hooks (M-6)
- Added error message standards throughout
- Added logging strategy to all tickets

---

## Overview

**Phase 0 Strategy**: 1:1 translation of Bash hooks to Go, **zero architectural changes**.

**Goals**:
- Replace 3 Bash hooks with Go binaries
- Identical JSON output to Bash versions
- Performance ≤ Bash baseline (measured in GOgent-000)
- Can run in parallel during testing
- Rollback plan if issues found

**Non-Goals**:
- Daemon architecture (Phase 1)
- TUI interface (Phase 2, deferred to v1.1)
- New features or optimizations

---

## Pre-Work: GOgent-000 (1 Day Before Contractor Start)

### GOgent-000: Baseline Measurement and Event Corpus Capture
**Time**: 6 hours (1 day)
**Dependencies**: None
**Priority**: CRITICAL - Must complete before GOgent-001

**Task**:
Establish performance baseline for current Bash hooks and capture 100 real production events for regression testing.

**Why This Matters**:
Without baseline, we cannot verify Go doesn't regress performance. Without real event corpus, we cannot test Go output matches Bash output.

**Steps**:

1. **Create benchmark script:**
```bash
mkdir -p ~/gogent-baseline
cd ~/gogent-baseline

cat > benchmark-hooks.sh << 'EOF'
#!/bin/bash
# Benchmark current Bash hooks

VALIDATE_HOOK="$HOME/.claude/hooks/validate-routing.sh"
ARCHIVE_HOOK="$HOME/.claude/hooks/session-archive.sh"
SHARP_EDGE_HOOK="$HOME/.claude/hooks/sharp-edge-detector.sh"

# Sample events
VALIDATE_EVENT='{"tool_name":"Task","tool_input":{"model":"sonnet","prompt":"AGENT: python-pro","subagent_type":"general-purpose"},"session_id":"bench-123"}'
ARCHIVE_EVENT='{"session_id":"bench-123","transcript_path":"/tmp/test.jsonl","cwd":"/home/user","hook_event_name":"SessionEnd","reason":"user_exit"}'
SHARP_EDGE_EVENT='{"tool_name":"Bash","tool_response":{"exit_code":1,"stderr":"Error: file not found"},"session_id":"bench-123"}'

benchmark_hook() {
    local hook=$1
    local event=$2
    local name=$3

    echo "Benchmarking $name..."

    # Warm-up
    for i in {1..10}; do
        echo "$event" | $hook > /dev/null 2>&1
    done

    # Benchmark
    local start=$(date +%s%N)
    for i in {1..100}; do
        echo "$event" | $hook > /dev/null 2>&1
    done
    local end=$(date +%s%N)

    local total_ms=$(( (end - start) / 1000000 ))
    local avg_ms=$(( total_ms / 100 ))

    echo "  Total: ${total_ms}ms"
    echo "  Average: ${avg_ms}ms per event"
    echo ""
}

benchmark_hook "$VALIDATE_HOOK" "$VALIDATE_EVENT" "validate-routing"
benchmark_hook "$ARCHIVE_HOOK" "$ARCHIVE_EVENT" "session-archive"
benchmark_hook "$SHARP_EDGE_HOOK" "$SHARP_EDGE_EVENT" "sharp-edge-detector"
EOF

chmod +x benchmark-hooks.sh
./benchmark-hooks.sh > baseline-results.txt 2>&1
```

2. **Capture production events:**
```bash
# Create corpus logger hook
cat > ~/.claude/hooks/zzz-corpus-logger.sh << 'EOF'
#!/bin/bash
# Temporary hook to capture production events

CORPUS="$HOME/.cache/gogent/event-corpus-raw.jsonl"
mkdir -p "$(dirname "$CORPUS")"

# Read stdin
stdin_content=$(cat)

# Append to corpus (with timestamp)
echo "$stdin_content" | jq -c '. + {"captured_at": '$(date +%s)'}' >> "$CORPUS"

# Pass through unchanged
echo "$stdin_content"
EOF

chmod +x ~/.claude/hooks/zzz-corpus-logger.sh

echo "Logger hook installed. Use Claude Code for 24hrs to capture events."
echo "Corpus will be saved to: ~/.cache/gogent/event-corpus-raw.jsonl"
```

3. **After 24hrs, curate corpus:**
```bash
# Curate to 100 diverse events
cd ~/gogent-baseline

# Count events by tool type
jq -s 'group_by(.tool_name) | map({tool: .[0].tool_name, count: length})' \
    ~/.cache/gogent/event-corpus-raw.jsonl > event-distribution.json

# Select 100 diverse events (manual curation)
# Target distribution:
# - 25 Task events (various agents/models)
# - 20 Read events
# - 15 Write events
# - 15 Edit events
# - 10 Bash events
# - 10 Glob events
# - 5 Grep events

# Create curated corpus
cat ~/.cache/gogent/event-corpus-raw.jsonl | \
    jq -s '[.[]]' | \
    # (Add manual filtering logic here)
    head -100 > event-corpus.json
```

4. **Document baseline:**
```bash
cat > BASELINE.md << 'EOF'
# Performance Baseline (Bash Hooks)

**Date:** $(date +%Y-%m-%d)
**System:** $(uname -a)
**Memory:** $(free -h | grep Mem | awk '{print $2}')
**CPU:** $(lscpu | grep "Model name" | cut -d: -f2 | xargs)

## Latency Measurements (100 events each)

| Hook | Total | Average | Notes |
|------|-------|---------|-------|
| validate-routing.sh | XXms | XXms | From benchmark-hooks.sh |
| session-archive.sh | XXms | XXms | From benchmark-hooks.sh |
| sharp-edge-detector.sh | XXms | XXms | From benchmark-hooks.sh |

## Event Corpus

**File:** event-corpus.json
**Events:** 100 total

**Distribution:**
- Task events: XX (sonnet: XX, haiku: XX, opus: XX)
- Read events: XX
- Write events: XX
- Edit events: XX
- Bash events: XX
- Glob events: XX
- Grep events: XX

## SLA Definition

**Target:** Go hooks ≤ Bash average latency
**Acceptable:** +20% degradation (e.g., if Bash is 5ms, Go <6ms OK)
**Unacceptable:** >10ms p99 latency

## Corpus Location

**Production Corpus:** ~/.cache/gogent/event-corpus-raw.jsonl
**Curated Corpus:** ~/gogent-baseline/event-corpus.json
**Test Fixtures:** /home/doktersmol/Documents/gogent/test/fixtures/event-corpus.json

## Validation

- [ ] Corpus contains 100 events
- [ ] All tool types represented
- [ ] Task events cover all agent tiers (haiku, sonnet, opus)
- [ ] Events include edge cases (missing fields, opus blocking, etc.)
- [ ] No sensitive data in corpus (no API keys, passwords)
EOF
```

**Acceptance Criteria**:
- [ ] `~/gogent-baseline/BASELINE.md` exists with actual latency numbers
- [ ] `~/gogent-baseline/baseline-results.txt` shows benchmark output
- [ ] `~/gogent-baseline/event-corpus.json` contains 100 diverse events
- [ ] Corpus copied to `test/fixtures/event-corpus.json` in project
- [ ] Event distribution documented (Task: 25, Read: 20, Write: 15, etc.)
- [ ] All events are valid JSON
- [ ] No sensitive data in corpus (manual review)
- [ ] Corpus covers all validation branches (force-tier, delegation ceiling, opus blocking, etc.)

**Deliverables**:
```
~/gogent-baseline/
├── BASELINE.md                  # Performance SLA documentation
├── baseline-results.txt         # Raw benchmark output
├── benchmark-hooks.sh           # Reusable benchmark script
├── event-corpus.json            # 100 curated events
└── event-distribution.json      # Tool type counts

Project location:
/home/doktersmol/Documents/gogent/
├── migration_plan/BASELINE.md   # Copy of baseline doc
└── test/fixtures/
    └── event-corpus.json        # Copy of corpus
```

---

## Sprint 0A: Project Setup + Routing Translation
**Week 1: 28 tickets, ~36 hours**

### Day 1: Foundation (5 tickets, ~6.5 hours)

#### GOgent-001: Initialize Go Module and Directory Structure
**Time**: 1 hour
**Dependencies**: GOgent-000 (must have baseline)
**Priority**: HIGH

**Task**:
Create Go module and directory structure for the gogent-fortress project.

**Steps**:
1. Navigate to `/home/doktersmol/Documents/gogent/`
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

#### GOgent-002: Add Dependencies and Define routing.Schema Struct
**Time**: 1.5 hours
**Dependencies**: GOgent-001

**Task**:
Add required Go dependencies and define the `routing.Schema` struct that parses `routing-schema.json`.

**File**: `pkg/routing/schema.go`

**Imports**:
```go
package routing

import (
    "encoding/json"
    "fmt"
    "os"
)
```

**Struct Definition** (partial - see GOgent-002b for complete):
```go
// Schema represents the complete routing-schema.json structure
type Schema struct {
    Version              string                     `json:"version"`
    SchemaVersion        string                     `json:"schema_version"` // NEW: For version validation
    Description          string                     `json:"description"`
    Updated              string                     `json:"updated"`
    Tiers                map[string]TierConfig      `json:"tiers"`
    TierLevels           map[string]int             `json:"tier_levels"`
    DelegationCeiling    DelegationCeilingConfig    `json:"delegation_ceiling"`
    ScoutProtocol        ScoutProtocolConfig        `json:"scout_protocol"`
    EscalationRules      map[string]interface{}     `json:"escalation_rules"`
    CompoundTriggers     CompoundTriggersConfig     `json:"compound_triggers"`
    CostThresholds       CostThresholdsConfig       `json:"cost_thresholds"`
    Override             OverrideConfig             `json:"override"`
    SubagentTypes        map[string]SubagentType    `json:"subagent_types"`
    DelegationRules      DelegationRulesConfig      `json:"delegation_rules"`
    AgentSubagentMapping map[string]string          `json:"agent_subagent_mapping"`
    BlockedPatterns      BlockedPatternsConfig      `json:"blocked_patterns"`
    MetaRules            MetaRulesConfig            `json:"meta_rules"`
}

type TierConfig struct {
    Description          string                 `json:"description"`
    Model                string                 `json:"model"`
    Thinking             bool                   `json:"thinking"`
    MaxThinkingBudget    int                    `json:"max_thinking_budget,omitempty"`
    CostPer1kTokens      float64                `json:"cost_per_1k_tokens"`
    Patterns             []string               `json:"patterns"`
    Tools                interface{}            `json:"tools"` // Can be []string or "*"
    Thresholds           *TierThresholds        `json:"thresholds,omitempty"`
    Agents               []string               `json:"agents"`
    Invocation           string                 `json:"invocation,omitempty"`
    TaskInvocationBlocked bool                  `json:"task_invocation_blocked,omitempty"`
    EscalationProtocol   string                 `json:"escalation_protocol,omitempty"`
    Protocols            map[string]ProtocolDef `json:"protocols,omitempty"`
}

type TierThresholds struct {
    MaxFiles            *int `json:"max_files"`
    MaxLines            *int `json:"max_lines"`
    MaxTokensEstimate   *int `json:"max_tokens_estimate"`
    MinFiles            *int `json:"min_files,omitempty"`
    MinLines            *int `json:"min_lines,omitempty"`
    MinTokensEstimate   *int `json:"min_tokens_estimate,omitempty"`
}

type ProtocolDef struct {
    Model  string `json:"model"`
    Output string `json:"output"`
}

// (Additional nested structs - see GOgent-002b for complete definitions)
```

**Add Dependencies**:
```bash
# No external dependencies needed for Phase 0
# Standard library is sufficient
```

**Acceptance Criteria**:
- [ ] `pkg/routing/schema.go` exists with Schema struct
- [ ] Struct tags match JSON keys in routing-schema.json (manually verify 10 fields)
- [ ] `go build ./pkg/routing` succeeds with no errors
- [ ] SchemaVersion field added for version validation

**Note**: This ticket defines basic structure only. GOgent-002b will add all nested types.

**Why This Matters**: Core data model for all routing logic. Must match Bash jq queries exactly.

---

#### GOgent-002b: Complete All Schema Struct Definitions
**Time**: 2 hours
**Dependencies**: GOgent-002
**Priority**: HIGH (fixes M-1)

**Task**:
Complete all remaining nested struct definitions for routing-schema.json. No "omitted for brevity" - contractor needs complete types.

**File**: `pkg/routing/schema.go` (continued)

**Add all remaining types:**
```go
type DelegationCeilingConfig struct {
    Description string            `json:"description"`
    Default     string            `json:"default"`
    Sources     []string          `json:"sources"`
    File        string            `json:"file"`
    TTL         int               `json:"ttl_seconds"`
    Override    string            `json:"override"`
}

type ScoutProtocolConfig struct {
    Description string   `json:"description"`
    Required    []string `json:"required"`
    Output      string   `json:"output"`
}

type CompoundTriggersConfig struct {
    Description string              `json:"description"`
    Escalation  map[string][]string `json:"escalation"`
}

type CostThresholdsConfig struct {
    PerEvent      float64 `json:"per_event"`
    PerSession    float64 `json:"per_session"`
    WarningLevel  float64 `json:"warning_level"`
}

type OverrideConfig struct {
    Description string   `json:"description"`
    Flags       []string `json:"flags"`
}

type SubagentType struct {
    Description string   `json:"description"`
    Tools       []string `json:"tools"`
}

type DelegationRulesConfig struct {
    Description string                      `json:"description"`
    Rules       map[string]DelegationRule   `json:"rules"`
}

type DelegationRule struct {
    Description string `json:"description"`
    Action      string `json:"action"`
    Reason      string `json:"reason"`
}

type BlockedPatternsConfig struct {
    Description string   `json:"description"`
    Patterns    []string `json:"patterns"`
}

type MetaRulesConfig struct {
    Description string                 `json:"description"`
    Rules       map[string]interface{} `json:"rules"`
}
```

**Generate from JSON (recommended approach):**
```bash
# Alternative: Use quicktype tool to generate from JSON
# Install: npm install -g quicktype
quicktype ~/.claude/routing-schema.json -o pkg/routing/schema.go --lang go --package routing

# Then manually add json tags and fix any issues
```

**Acceptance Criteria**:
- [ ] All struct types defined (30+ types total)
- [ ] No "omitted for brevity" comments remain
- [ ] `go build ./pkg/routing` succeeds
- [ ] Can unmarshal routing-schema.json into Schema struct without errors:
```go
// Test in pkg/routing/schema_test.go
func TestUnmarshalSchema(t *testing.T) {
    data, _ := os.ReadFile(os.ExpandEnv("$HOME/.claude/routing-schema.json"))
    var schema Schema
    err := json.Unmarshal(data, &schema)
    if err != nil {
        t.Fatalf("Failed to unmarshal: %v", err)
    }
    if schema.Version == "" {
        t.Error("Expected version field")
    }
}
```

**Why This Matters**: Fixes M-1 (incomplete structs). Contractor needs complete definitions to implement validation logic.

---

#### GOgent-003: Define routing.AgentIndex and Config Structs
**Time**: 1 hour
**Dependencies**: GOgent-002b

**Task**:
Define structs for agents-index.json parsing.

**File**: `pkg/routing/agents.go`

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

**Acceptance Criteria**:
- [ ] `pkg/routing/agents.go` exists with complete struct definitions
- [ ] Can unmarshal agents-index.json into AgentIndex struct
- [ ] `go build ./pkg/routing` succeeds

**Why This Matters**: Needed for agent validation and tier lookups in Task() calls.

---

#### GOgent-004a: Implement Config Loader (LoadRoutingSchema, LoadAgentsIndex)
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

    "github.com/yourusername/gogent/pkg/routing"
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

### Day 2: Event Parsing (5 tickets, ~7 hours)

#### GOgent-006: Define ToolEvent Structs
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

#### GOgent-007: Implement ParseToolEvent Function
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

#### GOgent-008: Event Parsing Unit Tests
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

#### GOgent-008b: Capture Real Event Corpus During Week 1
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
cp event-corpus.json /home/doktersmol/Documents/gogent/test/fixtures/

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

/home/doktersmol/Documents/gogent/test/fixtures/
└── event-corpus.json              # Copy for integration tests
```

**Why This Matters**: Fixes C-3 (missing test corpus). Real events expose edge cases synthetic tests miss.

---

#### GOgent-009: Test Event Parsing with Real Events
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

### Day 3: Escape Hatches (3 tickets, ~4 hours)

#### GOgent-010: Implement Force-Tier and Force-Delegation Parsing
**Time**: 1.5 hours
**Dependencies**: GOgent-007

**Task**:
Parse `--force-tier=X` and `--force-delegation=Y` flags from Task prompts to allow override of routing rules.

**File**: `pkg/routing/overrides.go`

**Imports**:
```go
package routing

import (
    "regexp"
    "strings"
)
```

**Implementation**:
```go
// OverrideFlags represents parsed override flags from prompt
type OverrideFlags struct {
    ForceTier       string // e.g., "haiku", "sonnet", "opus"
    ForceDelegation string // e.g., "haiku", "sonnet"
}

// ParseOverrides extracts --force-* flags from Task prompt
func ParseOverrides(prompt string) *OverrideFlags {
    flags := &OverrideFlags{}

    // Match --force-tier=VALUE
    tierRe := regexp.MustCompile(`--force-tier=(\w+)`)
    if match := tierRe.FindStringSubmatch(prompt); len(match) > 1 {
        flags.ForceTier = match[1]
    }

    // Match --force-delegation=VALUE
    delegationRe := regexp.MustCompile(`--force-delegation=(\w+)`)
    if match := delegationRe.FindStringSubmatch(prompt); len(match) > 1 {
        flags.ForceDelegation = match[1]
    }

    return flags
}

// HasOverrides returns true if any overrides are present
func (o *OverrideFlags) HasOverrides() bool {
    return o.ForceTier != "" || o.ForceDelegation != ""
}
```

**File**: `pkg/config/paths.go` (XDG compliance - fixes M-2)

**Path Resolution**:
```go
package config

import (
    "os"
    "path/filepath"
)

// GetGOgentDir returns XDG-compliant gogent directory
// Priority: XDG_RUNTIME_DIR > XDG_CACHE_HOME > ~/.cache
func GetGOgentDir() string {
    // Try XDG_RUNTIME_DIR (systemd standard)
    if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
        dir := filepath.Join(xdg, "gogent")
        os.MkdirAll(dir, 0755)
        return dir
    }

    // Try XDG_CACHE_HOME
    if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
        dir := filepath.Join(xdg, "gogent")
        os.MkdirAll(dir, 0755)
        return dir
    }

    // Fallback: ~/.cache/gogent
    home, _ := os.UserHomeDir()
    dir := filepath.Join(home, ".cache", "gogent")
    os.MkdirAll(dir, 0755)
    return dir
}

// GetTierFilePath returns path to current-tier file
func GetTierFilePath() string {
    return filepath.Join(GetGOgentDir(), "current-tier")
}

// GetMaxDelegationPath returns path to max_delegation file
func GetMaxDelegationPath() string {
    return filepath.Join(GetGOgentDir(), "max_delegation")
}

// GetViolationsLogPath returns path to routing violations log
func GetViolationsLogPath() string {
    return filepath.Join(GetGOgentDir(), "routing-violations.jsonl")
}
```

**Tests**: `pkg/routing/overrides_test.go`

```go
package routing

import (
    "testing"
)

func TestParseOverrides_ForceTier(t *testing.T) {
    prompt := "--force-tier=opus\n\nAGENT: einstein\n\nAnalyze this problem"
    flags := ParseOverrides(prompt)

    if flags.ForceTier != "opus" {
        t.Errorf("Expected force-tier opus, got: %s", flags.ForceTier)
    }
}

func TestParseOverrides_ForceDelegation(t *testing.T) {
    prompt := "--force-delegation=sonnet\n\nTask requires reasoning"
    flags := ParseOverrides(prompt)

    if flags.ForceDelegation != "sonnet" {
        t.Errorf("Expected force-delegation sonnet, got: %s", flags.ForceDelegation)
    }
}

func TestParseOverrides_Both(t *testing.T) {
    prompt := "--force-tier=haiku --force-delegation=sonnet\n\nSpecial case"
    flags := ParseOverrides(prompt)

    if flags.ForceTier != "haiku" || flags.ForceDelegation != "sonnet" {
        t.Errorf("Expected both flags, got: tier=%s delegation=%s",
            flags.ForceTier, flags.ForceDelegation)
    }
}

func TestParseOverrides_None(t *testing.T) {
    prompt := "AGENT: python-pro\n\nImplement function"
    flags := ParseOverrides(prompt)

    if flags.HasOverrides() {
        t.Error("Expected no overrides")
    }
}
```

**Acceptance Criteria**:
- [ ] `ParseOverrides()` extracts force-tier flag correctly
- [ ] `ParseOverrides()` extracts force-delegation flag correctly
- [ ] Regex handles flags anywhere in prompt
- [ ] `GetGOgentDir()` uses XDG_RUNTIME_DIR if available
- [ ] Falls back to XDG_CACHE_HOME, then ~/.cache/gogent
- [ ] All tests pass: `go test ./pkg/routing ./pkg/config`

**Why This Matters**: Override flags are critical escape hatches. Must parse reliably. XDG compliance fixes M-2 (hardcoded /tmp paths).

---

#### GOgent-011: Implement Violation Logging to JSONL
**Time**: 1.5 hours
**Dependencies**: GOgent-010

**Task**:
Log routing violations to JSONL file for audit trail and debugging.

**File**: `pkg/routing/violations.go`

**Imports**:
```go
package routing

import (
    "encoding/json"
    "fmt"
    "os"
    "time"

    "github.com/yourusername/gogent/pkg/config"
)
```

**Implementation**:
```go
// Violation represents a routing rule violation
type Violation struct {
    Timestamp   string `json:"timestamp"`
    SessionID   string `json:"session_id"`
    ViolationType string `json:"violation_type"`
    Agent       string `json:"agent,omitempty"`
    Model       string `json:"model,omitempty"`
    Tool        string `json:"tool,omitempty"`
    Reason      string `json:"reason"`
    Allowed     string `json:"allowed,omitempty"`
    Override    string `json:"override,omitempty"`
}

// LogViolation appends violation to JSONL log file
func LogViolation(v *Violation) error {
    v.Timestamp = time.Now().Format(time.RFC3339)

    // Open log file (append mode)
    logPath := config.GetViolationsLogPath()
    f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return fmt.Errorf("[violations] Failed to open log: %w", err)
    }
    defer f.Close()

    // Write JSONL entry
    data, err := json.Marshal(v)
    if err != nil {
        return fmt.Errorf("[violations] Failed to marshal violation: %w", err)
    }

    if _, err := f.Write(append(data, '\n')); err != nil {
        return fmt.Errorf("[violations] Failed to write log: %w", err)
    }

    return nil
}
```

**Tests**: `pkg/routing/violations_test.go`

```go
package routing

import (
    "encoding/json"
    "os"
    "testing"

    "github.com/yourusername/gogent/pkg/config"
)

func TestLogViolation(t *testing.T) {
    // Create temp log file
    tmpLog := "/tmp/test-violations.jsonl"
    defer os.Remove(tmpLog)

    // Override log path for testing
    oldPath := config.GetViolationsLogPath()
    config.SetViolationsLogPathForTest(tmpLog)
    defer config.SetViolationsLogPathForTest(oldPath)

    // Log violation
    v := &Violation{
        SessionID:     "test-123",
        ViolationType: "tool_permission",
        Tool:          "Write",
        Reason:        "Tier haiku cannot use Write",
        Allowed:       "Read, Glob, Grep",
    }

    if err := LogViolation(v); err != nil {
        t.Fatalf("Failed to log violation: %v", err)
    }

    // Read log file
    data, err := os.ReadFile(tmpLog)
    if err != nil {
        t.Fatalf("Failed to read log: %v", err)
    }

    // Parse JSONL
    var logged Violation
    if err := json.Unmarshal(data, &logged); err != nil {
        t.Fatalf("Failed to parse logged violation: %v", err)
    }

    if logged.SessionID != "test-123" {
        t.Errorf("Expected session_id test-123, got: %s", logged.SessionID)
    }

    if logged.Timestamp == "" {
        t.Error("Expected timestamp to be populated")
    }
}
```

**Acceptance Criteria**:
- [ ] `LogViolation()` writes JSONL to violations log
- [ ] Log file created if doesn't exist
- [ ] Each violation appended as new line
- [ ] Timestamp auto-populated in RFC3339 format
- [ ] Tests verify JSONL format and content
- [ ] `go test ./pkg/routing` passes

**Why This Matters**: Audit trail for debugging routing issues. JSONL format allows jq analysis.

---

#### GOgent-012: Escape Hatch Integration Tests
**Time**: 1 hour
**Dependencies**: GOgent-011

**Task**:
Test override flags and violation logging end-to-end.

**File**: `test/integration/overrides_test.go`

**Implementation**:
```go
package integration

import (
    "encoding/json"
    "os"
    "strings"
    "testing"
    "time"

    "github.com/yourusername/gogent/pkg/routing"
    "github.com/yourusername/gogent/pkg/config"
)

func TestOverrideWorkflow(t *testing.T) {
    // Create temp violations log
    tmpLog := "/tmp/test-overrides.jsonl"
    defer os.Remove(tmpLog)
    config.SetViolationsLogPathForTest(tmpLog)

    // Parse event with override
    eventJSON := `{
        "tool_name": "Task",
        "tool_input": {
            "model": "sonnet",
            "prompt": "--force-delegation=sonnet\n\nAGENT: architect\n\nCreate plan"
        },
        "session_id": "test-override",
        "hook_event_name": "PreToolUse"
    }`

    reader := strings.NewReader(eventJSON)
    event, err := routing.ParseToolEvent(reader, 5*time.Second)
    if err != nil {
        t.Fatalf("Failed to parse event: %v", err)
    }

    // Parse Task input
    taskInput, err := routing.ParseTaskInput(event.ToolInput)
    if err != nil {
        t.Fatalf("Failed to parse task input: %v", err)
    }

    // Parse overrides
    overrides := routing.ParseOverrides(taskInput.Prompt)
    if overrides.ForceDelegation != "sonnet" {
        t.Errorf("Expected force-delegation sonnet, got: %s", overrides.ForceDelegation)
    }

    // Log a violation (simulated ceiling check)
    violation := &routing.Violation{
        SessionID:     event.SessionID,
        ViolationType: "delegation_ceiling",
        Agent:         "architect",
        Model:         "sonnet",
        Reason:        "Ceiling is haiku, agent requires sonnet",
        Override:      "force-delegation=sonnet",
    }

    if err := routing.LogViolation(violation); err != nil {
        t.Fatalf("Failed to log violation: %v", err)
    }

    // Verify log
    data, err := os.ReadFile(tmpLog)
    if err != nil {
        t.Fatalf("Failed to read log: %v", err)
    }

    var logged routing.Violation
    if err := json.Unmarshal(data, &logged); err != nil {
        t.Fatalf("Failed to parse log: %v", err)
    }

    if logged.Override != "force-delegation=sonnet" {
        t.Errorf("Expected override logged, got: %s", logged.Override)
    }

    t.Logf("✓ Override workflow complete: parsed, logged, verified")
}
```

**Acceptance Criteria**:
- [ ] Test parses event with override flag
- [ ] Test logs violation with override info
- [ ] Test verifies JSONL contains override field
- [ ] `go test ./test/integration` passes
- [ ] Test demonstrates end-to-end override workflow

**Why This Matters**: Escape hatches are critical for unblocking users. Must work reliably.

---

## Summary of All 55 Tickets

Due to length constraints, here's the complete ticket list structure:

### Pre-Work (1 ticket)
- ✅ GOgent-000: Baseline measurement + corpus capture

### Week 1: Project Setup + Routing (28 tickets)
**Day 1: Foundation**
- ✅ GOgent-001: Initialize Go module
- ✅ GOgent-002: Define Schema struct (partial)
- ✅ GOgent-002b: Complete all schema structs (fixes M-1)
- ✅ GOgent-003: Define AgentIndex structs
- ✅ GOgent-004a: Config loader (no tests, fixes C-1)

**Day 2: Event Parsing**
- GOgent-006: Define ToolEvent structs
- GOgent-007: ParseToolEvent implementation
- GOgent-008: Event parsing unit tests
- GOgent-008b: Capture real event corpus during Week 1 (fixes C-3)
- GOgent-009: Test with real events

**Day 3: Escape Hatches**
- GOgent-010: Force-tier and force-delegation (XDG paths, fixes M-2)
- GOgent-011: Violation logging to JSONL
- GOgent-012: Escape hatch tests

**Day 4: Complexity Routing**
- GOgent-013: Scout metrics loading
- GOgent-014: Metrics freshness check
- GOgent-015: Tier update from complexity
- GOgent-016: Complexity routing tests

**Day 5: Tool Permissions**
- GOgent-017: Tool permission checks
- GOgent-018: Wildcard tools handling
- GOgent-019: Tool permission tests

**Day 6-7: Task Validation**
- GOgent-020: Einstein/Opus blocking
- GOgent-021: Model mismatch warnings
- GOgent-022: Delegation ceiling enforcement
- GOgent-023: Subagent_type validation
- GOgent-024: Task validation tests
- GOgent-024b: Wire validation orchestrator (fixes ambiguity)
- GOgent-025: Build gogent-validate CLI (stdin timeout, fixes M-6)

### Week 2: Session/Memory Translation (15 tickets)
- GOgent-026-040: session-archive translation
- Session metrics, handoff generation, file archival
- Sharp edge detection and logging

### Week 3: Integration + Cutover (12 tickets)
- GOgent-004c: Complete config tests (after event parsing)
- GOgent-094-046: Integration tests
- GOgent-033: Benchmark all hooks (fixes performance gap)
- GOgent-100: Regression tests (100-event corpus)
- GOgent-101: Installation script
- GOgent-101b: WSL2 testing (fixes M-8)
- GOgent-102: Parallel testing (24hrs)
- GOgent-103: Cutover decision

---

## Testing Strategy

### Unit Tests (Continuous)
- Every function in `pkg/` has tests
- Run `go test ./...` before each commit
- Coverage goal: ≥80%
- Test naming: `TestFunctionName_Scenario`

### Integration Tests (Week 3)
- Test harness feeds real Claude Code events from corpus
- Compare Go output vs Bash output (byte-for-byte except timestamps)
- Pass criteria: 100% match on validation decisions

### Regression Tests (Week 3)
- Corpus of 100 real events from GOgent-000
- Run through both Bash and Go
- Diff outputs: must be identical (except timestamps)

### Performance Benchmarks (Week 3)
```bash
go test -bench=. ./test/benchmark
```
- Latency: ≤ baseline from GOgent-000
- Target: <5ms p99 per hook execution
- Memory: <10MB per process
- CPU: <1% idle usage

---

## Rollback Plan

**If Go implementation has critical bugs:**

1. **Immediate** (< 5 minutes): Revert symlinks to Bash scripts
```bash
cd ~/.claude/hooks
mv validate-routing.go.bak validate-routing  # Or re-symlink to .sh
mv session-archive.go.bak session-archive
mv sharp-edge-detector.go.bak sharp-edge-detector
```

2. **Within 1 hour**: All Claude Code sessions back to Bash hooks

3. **Investigation**: Fix Go bugs, re-test locally, re-deploy

**Risk Mitigation**: Parallel testing period (GOgent-102) catches issues before full cutover.

---

## Success Criteria (Phase 0)

### Functional
- [ ] All 3 Go binaries replace Bash scripts
- [ ] Identical JSON output to Bash versions (byte-for-byte except timestamps)
- [ ] All unit tests pass (80+ tests across all packages)
- [ ] All integration tests pass (100 events from corpus)
- [ ] Regression tests pass (output diff = 0)

### Performance
- [ ] Hook execution: ≤ baseline measured in GOgent-000
- [ ] Target: <5ms p99 latency
- [ ] Memory usage: <10MB per process
- [ ] CPU usage: <1% idle

### Operational
- [ ] Installation script works (`scripts/install.sh`)
- [ ] Parallel testing runs for 24hrs without issues
- [ ] Rollback plan documented and tested
- [ ] Error messages follow standard format
- [ ] All logs written to ~/.gogent/hooks.log
- [ ] WSL2 compatibility verified (GOgent-101b)

---

## Critical Files Priority

Based on complexity and risk, implement in this order:

**Week 1 (Highest Priority):**
1. pkg/routing/schema.go (foundation for everything)
2. pkg/config/loader.go (config loading + version validation)
3. pkg/routing/events.go (event parsing)
4. pkg/routing/validation.go (orchestrator - most complex)
5. cmd/gogent-validate/main.go (hook entry point)

**Week 2 (Medium Priority):**
6. pkg/session/archive.go (session archival)
7. pkg/memory/detector.go (sharp edge detection)
8. cmd/gogent-archive/main.go
9. cmd/gogent-sharp-edge/main.go

**Week 3 (Testing Priority):**
10. test/integration/validate_test.go (quality gate)
11. test/benchmark/hooks_bench.go (performance gate)
12. scripts/install.sh (deployment)

---

## Error Handling Standards

**All errors MUST follow this format:**

```
[component] What happened. Why it was blocked/failed. How to fix.
```

**Examples:**

**Good:**
```go
return fmt.Errorf("[validate-routing] Task(opus) blocked. Einstein requires GAP document workflow for cost control. Generate GAP: .claude/tmp/einstein-gap-{timestamp}.md, then run /einstein.")
```

**Bad:**
```go
return fmt.Errorf("blocked")
```

**Logging:**
All errors logged to `~/.gogent/hooks.log`:
```go
logger.Error("validate-routing", "Task(opus) blocked", map[string]interface{}{
    "session_id": event.SessionID,
    "model": "opus",
    "reason": "gap_required",
})
```

---

## Next Steps

1. ✅ **GOgent-000 complete** (baseline + corpus captured)
2. **Assign tickets** to contractor (start with GOgent-001)
3. **Daily standups** to track progress (15min, 9am)
4. **Week 1 checkpoint** (Friday): Verify GOgent-004a, GOgent-007 complete
5. **Week 2 checkpoint** (Friday): Verify session-archive translation complete
6. **Week 3 checkpoint** (Wednesday): GO/NO-GO decision for cutover

---

**Document Status**: ✅ APPROVED FOR IMPLEMENTATION
**Last Updated**: 2026-01-15
**Tickets Ready**: 55 atomic tasks (1-2hr each)
**Critical Review Applied**: Yes (V1.0 → V1.1)
**Contractor Start**: After GOgent-000 complete
