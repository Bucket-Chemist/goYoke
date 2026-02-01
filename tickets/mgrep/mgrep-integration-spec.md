# mgrep Integration Specification for GOgent-Fortress

> **Version:** 1.0.0
> **Status:** Specification Complete - Ready for Implementation Planning
> **Author:** Einstein Analysis
> **Date:** 2026-02-01
> **Target Schema Version:** routing-schema v2.5.0

---

## 1. Executive Summary

### 1.1 What is mgrep?

mgrep is a semantic search CLI tool by Mixedbread AI that enables natural language queries over codebases. Unlike regex-based search (grep/ripgrep), mgrep uses semantic retrieval models to find **intent-matching** rather than **pattern-matching** results.

**Key Differentiator:** Agents can describe *what* they're looking for instead of *guessing patterns*.

### 1.2 Integration Goal

Transform GOgent-Fortress agents from pattern-matching to intent-understanding by integrating mgrep as an external context engine (following the gemini-slave architectural pattern).

### 1.3 Scope

| In Scope | Out of Scope |
|----------|--------------|
| mgrep as external engine via Bash | Native mgrep tool integration |
| Agent definition updates | Hook binary modifications (except telemetry) |
| Routing schema updates | Claude Code core changes |
| Telemetry for mgrep invocations | mgrep development/maintenance |
| `/mgrep` user-invoked skill | Automatic mgrep authentication |
| Fallback to Grep when unavailable | mgrep API wrapper library |

### 1.4 Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Token reduction on exploration | 40-60% | Compare pre/post on same queries |
| Scout routing accuracy | +40% improvement | Track tier changes after scout |
| False positive rate in search | -50% | Files returned but not used |
| Agent disambiguation iterations | -30% | Orchestrator re-routing events |

---

## 2. mgrep Technical Specification

### 2.1 Installation & Authentication

```bash
# Installation
npm install -g @mixedbread/mgrep

# Authentication (interactive - requires browser)
mgrep login

# Authentication (headless - for CI/hooks)
export MXBAI_API_KEY="your_key"
```

### 2.2 Project Indexing

```bash
# Index project (runs in background, watches for changes)
mgrep watch

# Manual sync before search
mgrep search -s "query"
```

### 2.3 CLI Invocation Patterns

| Mode | Command | Output | Use Case |
|------|---------|--------|----------|
| **Discover** | `mgrep "query" path -m N` | File list with context hints | codebase-search, scout |
| **Answer** | `mgrep "query" path -m N -a` | Synthesized answer | librarian, orchestrator |
| **Agentic** | `mgrep "query" path --agentic -a` | Multi-query deep analysis | orchestrator disambiguation |
| **Web Blended** | `mgrep --web "query" -a` | Internal + external results | librarian research |
| **Content** | `mgrep "query" path -m N -c` | Results with content preview | detailed discovery |

### 2.4 Key Flags Reference

| Flag | Description | Default |
|------|-------------|---------|
| `-m, --max-count` | Maximum results | 10 |
| `-c, --content` | Show result content | false |
| `-a, --answer` | Generate synthesized answer | false |
| `-w, --web` | Include web search results | false |
| `--agentic` | Multi-query refinement mode | false |
| `-s, --sync` | Sync files before search | false |
| `--no-rerank` | Disable result reranking | false |

### 2.5 Output Format

**Default (file list):**
```
src/auth/handler.go:45-67
src/middleware/session.go:12-34
pkg/jwt/validator.go:89-102
```

**With -a (answer mode):**
```
Based on the codebase, authentication is implemented in three layers:
1. JWT validation in pkg/jwt/validator.go
2. Session management in src/middleware/session.go
3. HTTP handler integration in src/auth/handler.go

[Files]
src/auth/handler.go:45-67
...
```

### 2.6 Environment Variables

| Variable | Purpose | GOgent Usage |
|----------|---------|--------------|
| `MXBAI_API_KEY` | Headless authentication | Required for hooks |
| `MXBAI_STORE` | Override store name | Optional |
| `MGREP_MAX_COUNT` | Default result limit | Set to 25 |
| `MGREP_RERANK` | Enable reranking | Leave as true |

### 2.7 Failure Modes

| Condition | Detection | Fallback |
|-----------|-----------|----------|
| mgrep not installed | `command -v mgrep` fails | Use Grep |
| Not authenticated | Exit code + "login required" | Use Grep |
| Network unavailable | Timeout after 30s | Use Grep |
| No index | "store not found" error | Run `mgrep watch` or use Grep |
| Empty results | Zero matches returned | Use Grep with relaxed pattern |

---

## 3. Routing Schema Changes

### 3.1 New External Engine Definition

**File:** `~/.claude/routing-schema.json`

**Add to `external.engines`:**

```json
{
  "external": {
    "engines": {
      "gemini-slave": { /* existing - unchanged */ },

      "mgrep": {
        "type": "semantic-search",
        "model": "mixedbread-semantic",
        "cost_model": {
          "type": "per-query",
          "estimated_cost_per_query": 0.001,
          "note": "Actual cost via Mixedbread billing"
        },
        "authentication": {
          "method": "api_key",
          "env_var": "MXBAI_API_KEY",
          "fallback": "device_login"
        },
        "availability_check": "command -v mgrep && mgrep --version",

        "invocation_modes": {
          "discover": {
            "command_template": "mgrep \"${query}\" ${path} -m ${count:-25}",
            "output_type": "file_list_with_hints",
            "timeout_ms": 30000,
            "agents": ["codebase-search", "haiku-scout", "review-orchestrator"]
          },
          "answer": {
            "command_template": "mgrep \"${query}\" ${path} -m ${count:-10} -a",
            "output_type": "synthesized_answer",
            "timeout_ms": 45000,
            "agents": ["librarian", "orchestrator", "architect", "memory-archivist"]
          },
          "agentic": {
            "command_template": "mgrep \"${query}\" ${path} --agentic -a",
            "output_type": "deep_analysis",
            "timeout_ms": 60000,
            "agents": ["orchestrator", "review-orchestrator"]
          },
          "web_blended": {
            "command_template": "mgrep --web \"${query}\" -a",
            "output_type": "internal_plus_external",
            "timeout_ms": 45000,
            "agents": ["librarian"]
          }
        },

        "selection_guidance": {
          "prefer_mgrep_when": [
            "Query describes intent (where is X implemented)",
            "Feature/concept discovery (how does Y work)",
            "Cross-module relationship detection",
            "Pattern finding without known keywords",
            "Onboarding/exploration tasks"
          ],
          "prefer_grep_when": [
            "Exact symbol/function tracing (func parseEvent)",
            "Known regex pattern matching",
            "Refactoring requiring exact locations",
            "mgrep unavailable or failed",
            "Simple filename/extension filtering"
          ]
        },

        "fallback": {
          "tool": "Grep",
          "triggers": [
            "mgrep_not_installed",
            "authentication_failed",
            "timeout_exceeded",
            "error_returned"
          ],
          "log_fallback": true
        }
      }
    }
  }
}
```

### 3.2 New Telemetry Types

**Add to schema (for documentation):**

```json
{
  "telemetry": {
    "mgrep_invocations": {
      "file": "${XDG_DATA_HOME}/gogent-fortress/mgrep-invocations.jsonl",
      "schema": "MgrepInvocation",
      "written_by": ["gogent-sharp-edge"],
      "retention": "30_days"
    },
    "mgrep_outcomes": {
      "file": "${XDG_DATA_HOME}/gogent-fortress/mgrep-outcomes.jsonl",
      "schema": "MgrepOutcome",
      "written_by": ["gogent-agent-endstate"],
      "retention": "30_days"
    }
  }
}
```

---

## 4. Agent Integration Specifications

### 4.1 Priority Tiers

| Priority | Agents | Rationale |
|----------|--------|-----------|
| **P0 - Critical** | codebase-search, haiku-scout | Foundation of all exploration |
| **P1 - High** | librarian, orchestrator | Synthesis and research agents |
| **P2 - Medium** | architect, review-orchestrator | Planning and review coordination |
| **P3 - Low** | *-reviewer, memory-archivist, *-pro | Context enhancement |

---

### 4.2 P0: codebase-search

**File:** `~/.claude/agents/codebase-search/agent.yaml`

**Current:**
```yaml
name: codebase-search
tools:
  - Glob
  - Grep
  - Read
```

**Updated:**
```yaml
name: codebase-search
description: >
  Fast file and code discovery specialist. Uses semantic search (mgrep) for
  intent-based queries and regex search (Grep) for exact pattern matching.
  Automatically selects the appropriate tool based on query type.

model: haiku

triggers:
  - "where is"
  - "find the"
  - "which files"
  - "locate"
  - "search for"
  - "find all"
  - "grep for"
  - "look for"
  - "how does"           # NEW: semantic trigger
  - "what implements"    # NEW: semantic trigger
  - "find code that"     # NEW: semantic trigger

tools:
  - Glob
  - Grep
  - Read
  - Bash  # NEW: for mgrep invocation

# NEW SECTION
mgrep_integration:
  enabled: true
  mode: discover
  default_count: 25

  tool_selection:
    use_mgrep:
      patterns:
        - "where is .* implemented"
        - "how does .* work"
        - "find .* that handles"
        - "what .* for"
        - "locate .* responsible"
      examples:
        - "where is authentication implemented"
        - "how does error handling work"
        - "find code that handles rate limiting"

    use_grep:
      patterns:
        - "grep for .*"
        - "find uses of .*"
        - "all references to .*"
        - "exact match .*"
      examples:
        - "grep for TODO"
        - "find uses of parseEvent"
        - "all references to pkg/routing"

    ambiguous:
      action: "prefer_mgrep_with_grep_fallback"

  invocation:
    command: |
      mgrep "${query}" ${path:-.} -m ${count:-25}
    parse_output: "file_list_with_line_ranges"

  fallback:
    condition: "mgrep fails or unavailable"
    action: |
      # Extract keywords from semantic query
      # Run grep with relaxed patterns
      grep -r "${keywords}" ${path:-.} --include="*.go" --include="*.py" -l

  output_transformation:
    mgrep_to_standard: |
      # Convert mgrep output to standard file:line format
      # src/auth/handler.go:45-67 → /absolute/path/src/auth/handler.go:45

output_requirements:
  - All paths must be absolute (starting with /)
  - Return structured results with file:line references
  - Include context hints from mgrep when available
  - Address underlying needs, not just literal requests
  - "If results exceed 20 files, recommend: 'Route to Orchestrator for triage.'"
```

**CLAUDE.md Addition for Agent:**

```markdown
## Tool Selection: mgrep vs Grep

Before searching, classify the query:

| Query Type | Tool | Example |
|------------|------|---------|
| **Intent-based** | mgrep | "where is authentication implemented" |
| **Concept discovery** | mgrep | "how does error handling work" |
| **Feature location** | mgrep | "find code that handles rate limiting" |
| **Exact symbol** | Grep | "find uses of parseEvent" |
| **Pattern match** | Grep | "grep for TODO comments" |
| **Refactoring** | Grep | "all imports of pkg/routing" |

**mgrep Invocation:**
```bash
mgrep "your natural language query" path/ -m 25
```

**Fallback to Grep if:**
- mgrep command not found
- Authentication error
- Timeout (>30s)
- Zero results (retry with Grep)
```

---

### 4.3 P0: haiku-scout

**File:** `~/.claude/agents/haiku-scout/agent.yaml`

**Current:**
```yaml
name: Haiku Scout
tools:
  - Read
  - Glob
  - Grep
  - Bash
output:
  file: .claude/tmp/scout_metrics.json
```

**Updated:**
```yaml
name: Haiku Scout
description: >
  Reconnaissance agent for scope assessment. Uses semantic search (mgrep) for
  conceptual scope detection, then mechanical metrics for quantification.
  Outputs both semantic scope and traditional metrics for routing decisions.

model: haiku
thinking:
  enabled: true
  budget: 2000

tier: 1
category: reconnaissance

triggers:
  - assess scope
  - count lines
  - estimate complexity
  - pre-route
  - scout
  - how big is
  - how many files

tools:
  - Read
  - Glob
  - Grep
  - Bash  # For mgrep AND wc/find commands

# NEW: Enhanced scout protocol
scout_protocol:
  phases:
    phase_1_semantic:
      description: "Assess conceptual scope via mgrep"
      condition: "mgrep available"
      command: |
        mgrep "${task_description}" ${path:-.} -m 50
      output_file: .claude/tmp/mgrep-scope.txt
      parse: |
        # Count files returned
        SEMANTIC_FILE_COUNT=$(wc -l < .claude/tmp/mgrep-scope.txt)
        # Extract unique directories
        SEMANTIC_DIRS=$(cut -d: -f1 .claude/tmp/mgrep-scope.txt | xargs dirname | sort -u)
      timeout_ms: 30000
      fallback: "skip to phase_2"

    phase_2_mechanical:
      description: "Gather LoC metrics on relevant files"
      input: |
        # If mgrep succeeded, scope to its files
        # Otherwise, use full path
        if [ -f .claude/tmp/mgrep-scope.txt ]; then
          FILES=$(cut -d: -f1 .claude/tmp/mgrep-scope.txt)
        else
          FILES=$(find ${path:-.} -type f \( -name "*.go" -o -name "*.py" -o -name "*.ts" \))
        fi
      metrics:
        - total_files
        - total_lines
        - estimated_tokens
        - import_density
        - cross_file_dependencies

    phase_3_synthesize:
      description: "Combine semantic + mechanical into routing recommendation"
      inputs:
        - .claude/tmp/mgrep-scope.txt (if exists)
        - mechanical metrics from phase_2
      output_file: .claude/tmp/scout_metrics.json
      schema: |
        {
          "semantic_scope": {
            "query": "${task_description}",
            "file_count": <int>,
            "directories_involved": [<strings>],
            "mgrep_available": <bool>
          },
          "mechanical_scope": {
            "total_files": <int>,
            "total_lines": <int>,
            "estimated_tokens": <int>
          },
          "routing_recommendation": {
            "recommended_tier": "haiku|sonnet|opus|external",
            "confidence": <float 0-1>,
            "reasoning": "<string>"
          }
        }

  tier_thresholds:
    haiku:
      max_semantic_files: 5
      max_total_lines: 500
    sonnet:
      max_semantic_files: 20
      max_total_lines: 5000
    external:
      min_semantic_files: 20
      min_total_lines: 5000

output:
  format: json
  schema: scout_report
  files:
    - .claude/tmp/scout_metrics.json
    - .claude/tmp/mgrep-scope.txt  # NEW

max_files: 50
max_tokens_per_file: 500
cost_ceiling_usd: 0.02
fallback_for: gemini-slave scout

integration:
  calculate_complexity:
    description: "After scout writes output, calculate-complexity.sh processes it"
    trigger: "automatic via explore workflow or manual"
    inputs:
      - .claude/tmp/scout_metrics.json
      - .claude/tmp/mgrep-scope.txt  # NEW: factor in semantic scope
    output: .claude/tmp/complexity_score
```

---

### 4.4 P1: librarian

**File:** `~/.claude/agents/librarian/agent.yaml`

**Updated:**
```yaml
name: librarian
description: >
  External documentation lookup and synthesis. Checks internal codebase patterns
  FIRST via mgrep, then external sources. Uses blended search for comprehensive
  research combining internal and external knowledge.

model: haiku
thinking:
  enabled: true
  budget: 4000

tools:
  - Read
  - WebFetch
  - WebSearch
  - Bash  # NEW: for mgrep
  - Grep  # NEW: fallback

triggers:
  - "how do I use"
  - "library docs"
  - "best practice"
  - "API reference"
  - "documentation for"
  - "what's the pattern for"
  - "example of"
  - "official docs"

# NEW: mgrep-first research strategy
research_strategy:
  step_1_internal:
    description: "Check if project has existing patterns/implementations"
    command: |
      mgrep "${topic} patterns in this codebase" . -m 10 -a
    condition: "mgrep available"
    output: "internal_patterns"

  step_2_assess:
    description: "Determine if internal patterns are sufficient"
    decision:
      sufficient: "Internal patterns cover the use case → return with caveats"
      partial: "Internal patterns exist but need external augmentation"
      none: "No internal patterns → proceed to external search"

  step_3_external:
    description: "Search external sources if needed"
    options:
      web_blended:
        command: |
          mgrep --web "${topic}" -a
        use_when: "Need both internal context and external best practices"
      web_only:
        tool: WebSearch
        use_when: "Topic is external library/API with no internal relevance"

  step_4_reconcile:
    description: "Compare internal practice to external recommendation"
    focus:
      - "Are we following best practices?"
      - "Should we adapt external pattern to our conventions?"
      - "What caveats apply specifically to this codebase?"

thinking_focus:
  - "What patterns exist internally?" # NEW
  - "Which external sources are most authoritative?"
  - "Do sources conflict? How to reconcile?"
  - "What's the recommended approach vs alternatives?"
  - "What caveats apply to this use case?"

output_format:
  - "Internal patterns found (if any)" # NEW
  - "Primary recommendation with rationale"
  - "Key caveats or gotchas"
  - "Link to authoritative source"
  - "If sources conflict on architecture, recommend: 'Route to Orchestrator for design decision.'"

escalate_to: orchestrator
escalation_triggers:
  - "Multiple conflicting best practices"
  - "Complex integration pattern spanning multiple libraries"
  - "Architecture-level decision required"
  - "Internal patterns conflict with external recommendations"  # NEW
```

---

### 4.5 P1: orchestrator

**File:** `~/.claude/agents/orchestrator/agent.yaml`

**Updated:**
```yaml
name: Orchestrator
description: >
  Handles ambiguous scope, cross-module planning, user interviews,
  design tradeoffs, and debugging loops. Uses semantic search for
  scope disambiguation before spawning specialist agents.

model: sonnet
thinking:
  enabled: true
  budget: 16000
  budget_complex: 24000

triggers:
  - ambiguous scope
  - cross-module planning
  - user interview
  - design decision
  - debugging loop
  - think through
  - analyze
  - architect
  - synthesize
  - synthesis
  - review findings
  - triage
  - interpret results

tools:
  - Read
  - Glob
  - Grep
  - Task
  - Bash  # NEW: for mgrep

# NEW: Semantic disambiguation protocol
disambiguation_protocol:
  step_1_semantic_scope:
    description: "Understand conceptual scope before mechanical search"
    condition: "Scope is ambiguous or spans modules"
    command: |
      mgrep --agentic "${task_description}" . -a
    output: "semantic_scope_analysis"
    parse:
      - "Modules/directories involved"
      - "Cross-module relationships"
      - "Recommended specialist agents"

  step_2_verify_scope:
    description: "Validate semantic findings with targeted grep"
    command: |
      # For each module identified, verify with grep
      grep -r "${key_terms}" ${identified_modules}
    purpose: "Confirm semantic results, identify false positives"

  step_3_spawn_informed:
    description: "Spawn agents with semantically-informed context"
    input: "Combined semantic + mechanical scope"
    output: "Agent invocations with accurate scope"

conventions_required: []
sharp_edges_count: 0

escalate_to: einstein
escalation_triggers:
  - "3 consecutive failures on same task"
  - "Scope spans 4+ modules with integration"
  - "Novel problem with no clear pattern"
  - "User requests deep analysis"
  - "Semantic and mechanical scope disagree significantly"  # NEW

failure_tracking:
  max_attempts: 3
  on_max_reached: "escalate_to_einstein"
```

---

### 4.6 P2: architect

**File:** `~/.claude/agents/architect/agent.yaml`

**Add to existing:**
```yaml
# ADD to tools section
tools:
  - Read
  - Write
  - Glob
  - Grep
  - Bash  # NEW: for mgrep

# ADD new section
pattern_discovery:
  description: "Use mgrep to find existing patterns before planning"
  commands:
    similar_implementations:
      command: mgrep "similar implementations to ${feature}" . -m 10 -a
      purpose: "Find patterns to follow"

    dependencies:
      command: mgrep "what depends on ${module}" . -m 20
      purpose: "Identify downstream impacts"

    conventions:
      command: mgrep "established patterns for ${pattern_type}" . -a
      purpose: "Ensure plan follows conventions"

  integration:
    input: "Scout report + pattern discovery"
    output: "More accurate implementation plan"
```

---

### 4.7 P2: review-orchestrator

**File:** `~/.claude/agents/review-orchestrator/agent.yaml`

**Add to existing:**
```yaml
# ADD to tools section
tools:
  - Read
  - Glob
  - Grep
  - Task
  - Write
  - Bash  # NEW: for mgrep

# ADD new section
domain_detection:
  description: "Use semantic analysis to classify change domains"
  method: mgrep
  command: |
    mgrep "classify these changes: backend API, frontend UI, shared utilities, infrastructure" ${changed_files} -a
  output: |
    {
      "domains": ["backend", "frontend"],
      "confidence": 0.9,
      "reasoning": "Changes affect API handlers and React components"
    }
  fallback:
    method: pattern_matching
    rules:
      - "*.go in pkg/api/ → backend"
      - "*.tsx in components/ → frontend"
      - "*.go in cmd/ → infrastructure"

reviewer_selection:
  based_on: domain_detection
  mapping:
    backend: backend-reviewer
    frontend: frontend-reviewer
    both: [backend-reviewer, frontend-reviewer]
    infrastructure: standards-reviewer
```

---

### 4.8 P3: Reviewer Agents (backend-reviewer, frontend-reviewer, standards-reviewer)

**Add to each agent.yaml:**
```yaml
# ADD to tools section
tools:
  - Read
  - Glob
  - Grep
  - Bash  # NEW: for mgrep (optional)

# ADD new section
context_gathering:
  optional: true
  description: "Use mgrep to find related code for comparison"
  commands:
    similar_code:
      command: mgrep "similar ${file_type} in this codebase" ${directory} -m 5
      purpose: "Find patterns to compare against"

    established_patterns:
      command: mgrep "${focus_area} patterns" . -m 10 -a
      purpose: "Verify code follows established conventions"
```

---

### 4.9 P3: memory-archivist

**File:** `~/.claude/agents/memory-archivist/agent.yaml`

**Add to existing:**
```yaml
# ADD to tools section
tools:
  - Read
  - Write
  - Glob
  - Bash  # NEW: for mgrep

# MODIFY integration section
integration:
  rag_indexing:
    description: "Files in .claude/memory/ are searchable"
    query_methods:
      grep:
        - "grep -r 'type: decision' .claude/memory/"
        - "grep -r 'tags:.*{keyword}' .claude/memory/"
      mgrep:  # NEW
        - "mgrep 'decisions about {topic}' .claude/memory/ -a"
        - "mgrep 'sharp edges related to {pattern}' .claude/memory/ -m 10"

  deduplication:
    method: semantic  # NEW: changed from grep-based
    command: |
      mgrep "have we captured learnings about ${topic} before?" .claude/memory/ -a
    action:
      duplicate_found: "Skip archiving, reference existing"
      partial_match: "Update existing with new context"
      no_match: "Create new memory entry"
```

---

## 5. Telemetry Specification

### 5.1 New Data Types

**File:** `pkg/telemetry/mgrep_invocation.go`

```go
package telemetry

import (
    "time"
    "github.com/google/uuid"
)

// MgrepInvocation records a single mgrep command execution
type MgrepInvocation struct {
    InvocationID   string    `json:"invocation_id"`
    SessionID      string    `json:"session_id"`
    Timestamp      int64     `json:"timestamp"`

    // Context
    InvokingAgent  string    `json:"invoking_agent"`
    InvokingTool   string    `json:"invoking_tool"`  // Usually "Bash"

    // Command details
    Query          string    `json:"query"`
    Path           string    `json:"path"`
    Mode           string    `json:"mode"`  // discover, answer, agentic, web_blended
    Flags          []string  `json:"flags"`

    // Results
    ResultCount    int       `json:"result_count"`
    FilesReturned  []string  `json:"files_returned,omitempty"`
    DurationMs     int64     `json:"duration_ms"`

    // Status
    Success        bool      `json:"success"`
    ErrorMessage   string    `json:"error_message,omitempty"`
    FallbackUsed   bool      `json:"fallback_used"`
}

// MgrepOutcome records whether mgrep results were actually useful
type MgrepOutcome struct {
    InvocationID    string   `json:"invocation_id"`
    OutcomeTimestamp int64   `json:"outcome_timestamp"`

    // Usage metrics
    FilesActuallyRead []string `json:"files_actually_read"`
    ResultsUseful     bool     `json:"results_useful"`
    Precision         float64  `json:"precision"`  // files_read / files_returned

    // Token impact
    EstimatedTokensSaved int   `json:"estimated_tokens_saved,omitempty"`
}

// NewMgrepInvocation creates a new invocation record
func NewMgrepInvocation(sessionID, agent, query, path, mode string) *MgrepInvocation {
    return &MgrepInvocation{
        InvocationID:  uuid.New().String(),
        SessionID:     sessionID,
        Timestamp:     time.Now().Unix(),
        InvokingAgent: agent,
        InvokingTool:  "Bash",
        Query:         query,
        Path:          path,
        Mode:          mode,
    }
}
```

### 5.2 Logging Functions

**File:** `pkg/telemetry/mgrep_logging.go`

```go
package telemetry

import (
    "encoding/json"
    "os"
    "path/filepath"
)

// LogMgrepInvocation appends invocation to JSONL file
func LogMgrepInvocation(inv *MgrepInvocation) error {
    path := getMgrepInvocationsPath()
    return appendJSONL(path, inv)
}

// LogMgrepOutcome appends outcome to JSONL file
func LogMgrepOutcome(outcome *MgrepOutcome) error {
    path := getMgrepOutcomesPath()
    return appendJSONL(path, outcome)
}

// getMgrepInvocationsPath returns XDG_DATA_HOME/gogent-fortress/mgrep-invocations.jsonl
func getMgrepInvocationsPath() string {
    dataHome := os.Getenv("XDG_DATA_HOME")
    if dataHome == "" {
        home, _ := os.UserHomeDir()
        dataHome = filepath.Join(home, ".local", "share")
    }
    return filepath.Join(dataHome, "gogent-fortress", "mgrep-invocations.jsonl")
}

// getMgrepOutcomesPath returns XDG_DATA_HOME/gogent-fortress/mgrep-outcomes.jsonl
func getMgrepOutcomesPath() string {
    dataHome := os.Getenv("XDG_DATA_HOME")
    if dataHome == "" {
        home, _ := os.UserHomeDir()
        dataHome = filepath.Join(home, ".local", "share")
    }
    return filepath.Join(dataHome, "gogent-fortress", "mgrep-outcomes.jsonl")
}
```

### 5.3 Detection in gogent-sharp-edge

**File:** `cmd/gogent-sharp-edge/main.go`

Add mgrep detection to PostToolUse handler:

```go
// detectMgrepInvocation checks if a Bash command is an mgrep call
func detectMgrepInvocation(event *routing.PostToolEvent) *telemetry.MgrepInvocation {
    if event.ToolName != "Bash" {
        return nil
    }

    command, ok := event.ToolInput["command"].(string)
    if !ok || !strings.HasPrefix(strings.TrimSpace(command), "mgrep ") {
        return nil
    }

    // Parse mgrep command
    inv := telemetry.NewMgrepInvocation(
        event.SessionID,
        extractAgentFromContext(event),
        extractMgrepQuery(command),
        extractMgrepPath(command),
        extractMgrepMode(command),
    )

    // Parse result count from output
    if output, ok := event.ToolResponse["output"].(string); ok {
        inv.ResultCount = countMgrepResults(output)
        inv.FilesReturned = parseMgrepFiles(output)
    }

    // Check for errors
    if exitCode, ok := event.ToolResponse["exit_code"].(float64); ok && exitCode != 0 {
        inv.Success = false
        inv.ErrorMessage = extractErrorMessage(event.ToolResponse)
    } else {
        inv.Success = true
    }

    return inv
}
```

### 5.4 ML Export Enhancement

**File:** `cmd/gogent-ml-export/main.go`

Add new export commands:

```go
// Add to subcommands
case "mgrep-invocations":
    return exportMgrepInvocations(outputPath, sinceDate)
case "mgrep-outcomes":
    return exportMgrepOutcomes(outputPath)
case "mgrep-stats":
    return printMgrepStats()
```

**Stats output format:**
```
mgrep Invocation Statistics
============================
Total invocations: 342
By mode:
  discover: 198 (57.9%)
  answer: 89 (26.0%)
  agentic: 41 (12.0%)
  web_blended: 14 (4.1%)

By agent:
  codebase-search: 156
  haiku-scout: 78
  orchestrator: 52
  librarian: 34
  ...

Success rate: 94.2%
Average duration: 1.2s
Fallback rate: 5.8%

Precision (files read / files returned): 0.68
Estimated token savings: 124,500
```

---

## 6. User-Invoked Skill: /mgrep

### 6.1 Skill Definition

**File:** `~/.claude/skills/mgrep/SKILL.md`

```markdown
# /mgrep Skill

## Purpose

Semantic code discovery using mgrep. Use when you need to find code by
describing what it does, not what patterns it matches.

## Invocation

| Command | Behavior |
|---------|----------|
| `/mgrep [query]` | Search current directory |
| `/mgrep [query] [path]` | Search specific path |
| `/mgrep -a [query]` | Search and synthesize answer |
| `/mgrep --agentic [query]` | Deep multi-query analysis |
| `/mgrep --web [query]` | Include external web results |

## Examples

```bash
# Find where authentication is implemented
/mgrep "where is authentication implemented"

# Understand how error handling works (with synthesis)
/mgrep -a "how does error handling work in this codebase"

# Deep analysis of module interactions
/mgrep --agentic "how do the routing and validation modules interact"

# Research with external sources
/mgrep --web "best practices for Go error wrapping"
```

## Workflow

1. **Parse arguments** - Extract query, path, flags
2. **Check availability** - Verify mgrep installed and authenticated
3. **Execute search** - Run mgrep with appropriate flags
4. **Format output** - Present results with file:line references
5. **Optionally read** - Offer to read top results

## Output Format

```
mgrep results for: "where is authentication implemented"
Path: ./

Files (12 matches):
1. pkg/auth/handler.go:45-67
   → JWT token validation and session creation
2. pkg/middleware/session.go:12-34
   → Session middleware for request authentication
3. cmd/api/routes.go:89-102
   → Route registration with auth middleware
...

[Enter number to read file, or 'a' for synthesized answer]
```

## Fallback

If mgrep is unavailable:
```
[mgrep] Not available (not installed or not authenticated)
[mgrep] Falling back to grep...
[mgrep] Tip: Run 'npm install -g @mixedbread/mgrep && mgrep login' to enable semantic search
```
```

---

## 7. Configuration Files

### 7.1 .mgrepignore

**File:** Project root `.mgrepignore`

```gitignore
# Standard ignores (in addition to .gitignore)

# Build artifacts
dist/
build/
*.exe
*.dll
*.so
*.dylib

# Dependencies
node_modules/
vendor/
.venv/

# IDE
.idea/
.vscode/
*.swp

# GOgent-specific
.claude/tmp/
.claude/session-archive/
.claude/statsig/

# Sensitive (if any)
.env
*.key
*.pem
credentials.*

# Large binaries
*.zip
*.tar.gz
*.wasm
```

### 7.2 Environment Setup

**File:** Add to user shell profile or `.envrc`

```bash
# mgrep configuration for GOgent-Fortress

# API key for headless operation (if not using device login)
# export MXBAI_API_KEY="your_key_here"

# Default result count
export MGREP_MAX_COUNT=25

# Keep reranking enabled for better results
export MGREP_RERANK=1

# GOgent integration flag
export GOGENT_MGREP_ENABLED=1
```

---

## 8. Implementation Phases

### Phase 1: Foundation (Week 1)

**Deliverables:**
1. [ ] Install mgrep, test authentication
2. [ ] Create `/mgrep` skill with full SKILL.md
3. [ ] Add mgrep engine definition to routing-schema.json
4. [ ] Create .mgrepignore template
5. [ ] Document in CLAUDE.md (reference only, not enforcement)

**Validation:**
- `/mgrep "test query"` works from any project
- Fallback to grep works when mgrep unavailable

### Phase 2: Critical Agents (Weeks 2-3)

**Deliverables:**
1. [ ] Update codebase-search/agent.yaml
2. [ ] Update codebase-search/CLAUDE.md with tool selection guide
3. [ ] Update haiku-scout/agent.yaml with semantic phase
4. [ ] Update calculate-complexity.sh to consume mgrep-scope.txt
5. [ ] Add pkg/telemetry/mgrep_invocation.go
6. [ ] Add mgrep detection to gogent-sharp-edge

**Validation:**
- codebase-search uses mgrep for intent queries
- Scout produces both semantic and mechanical scope
- Telemetry captures mgrep invocations

### Phase 3: Research Agents (Week 4)

**Deliverables:**
1. [ ] Update librarian/agent.yaml with internal-first strategy
2. [ ] Update orchestrator/agent.yaml with disambiguation protocol
3. [ ] Update architect/agent.yaml with pattern discovery

**Validation:**
- Librarian checks internal patterns before external search
- Orchestrator uses semantic scope for disambiguation

### Phase 4: Review Pipeline (Week 5)

**Deliverables:**
1. [ ] Update review-orchestrator/agent.yaml with domain detection
2. [ ] Update reviewer agents with optional context gathering

**Validation:**
- Review-orchestrator correctly classifies backend vs frontend
- Reviewers can find similar code for comparison

### Phase 5: Telemetry & Polish (Week 6)

**Deliverables:**
1. [ ] Add mgrep-invocations and mgrep-outcomes export
2. [ ] Add mgrep-stats command
3. [ ] Update ARCHITECTURE.md with mgrep integration
4. [ ] Update memory-archivist with semantic deduplication
5. [ ] Create benchmark prompts for mgrep effectiveness

**Validation:**
- `gogent-ml-export mgrep-stats` produces meaningful output
- Documentation is complete

---

## 9. Risk Mitigations

### 9.1 Fallback Strategy

Every mgrep integration MUST have grep fallback:

```yaml
mgrep_integration:
  enabled: true
  fallback:
    tool: Grep
    triggers:
      - command_not_found
      - authentication_failed
      - timeout_exceeded
      - error_returned
      - zero_results
    log_fallback: true
```

### 9.2 Availability Check

Before any mgrep invocation:

```bash
# Quick availability check
if ! command -v mgrep &> /dev/null; then
    echo "[mgrep] Not installed, using grep fallback"
    USE_GREP=1
elif ! mgrep --version &> /dev/null; then
    echo "[mgrep] Not authenticated, using grep fallback"
    USE_GREP=1
fi
```

### 9.3 Privacy Documentation

Add to CLAUDE.md:

```markdown
## mgrep Privacy Notice

mgrep indexes code to Mixedbread's cloud service for semantic search.

**What is indexed:**
- File contents matching patterns (respects .gitignore and .mgrepignore)
- File paths and structure

**What is NOT indexed:**
- Files in .mgrepignore
- Files in .gitignore
- Directories: .claude/tmp/, .claude/session-archive/, node_modules/, vendor/

**To disable mgrep:**
```bash
export GOGENT_MGREP_ENABLED=0
```
```

### 9.4 Cost Visibility

Add to session cost summary (via gogent-archive):

```json
{
  "session_costs": {
    "claude_api": 0.45,
    "mgrep_queries": 0.012,
    "gemini_slave": 0.003,
    "total": 0.465
  }
}
```

---

## 10. Testing Specification

### 10.1 Unit Tests

**File:** `pkg/telemetry/mgrep_invocation_test.go`

```go
func TestNewMgrepInvocation(t *testing.T) {
    inv := NewMgrepInvocation("session-123", "codebase-search", "find auth", ".", "discover")

    assert.NotEmpty(t, inv.InvocationID)
    assert.Equal(t, "session-123", inv.SessionID)
    assert.Equal(t, "codebase-search", inv.InvokingAgent)
    assert.Equal(t, "find auth", inv.Query)
    assert.Equal(t, "discover", inv.Mode)
}

func TestParseMgrepCommand(t *testing.T) {
    tests := []struct {
        command  string
        expected MgrepInvocation
    }{
        {
            command: `mgrep "where is auth" pkg/ -m 25`,
            expected: MgrepInvocation{Query: "where is auth", Path: "pkg/", Mode: "discover"},
        },
        {
            command: `mgrep "how does X work" . -a`,
            expected: MgrepInvocation{Query: "how does X work", Path: ".", Mode: "answer"},
        },
        {
            command: `mgrep --agentic "complex query" . -a`,
            expected: MgrepInvocation{Query: "complex query", Path: ".", Mode: "agentic"},
        },
    }
    // ...
}
```

### 10.2 Integration Tests

**File:** `test/integration/mgrep_integration_test.go`

```go
func TestMgrepFallbackToGrep(t *testing.T) {
    // Temporarily disable mgrep
    os.Setenv("GOGENT_MGREP_ENABLED", "0")
    defer os.Unsetenv("GOGENT_MGREP_ENABLED")

    // Invoke codebase-search
    result := invokeCodebaseSearch("where is authentication")

    // Should have used grep fallback
    assert.True(t, result.FallbackUsed)
    assert.NotEmpty(t, result.Files)
}

func TestScoutSemanticPhase(t *testing.T) {
    // Run scout with mgrep enabled
    result := invokeHaikuScout("refactor authentication module")

    // Should have both semantic and mechanical scope
    assert.FileExists(t, ".claude/tmp/mgrep-scope.txt")
    assert.FileExists(t, ".claude/tmp/scout_metrics.json")

    metrics := loadScoutMetrics()
    assert.NotNil(t, metrics.SemanticScope)
    assert.True(t, metrics.SemanticScope.MgrepAvailable)
}
```

### 10.3 Benchmark Prompts

**File:** `test/benchmarks/mgrep_effectiveness.md`

```markdown
# mgrep Effectiveness Benchmarks

## Semantic Discovery Prompts

| Prompt | Expected Result | Grep Difficulty |
|--------|-----------------|-----------------|
| "where is authentication implemented" | auth/, session/, jwt/ | Hard (many false positives) |
| "how does error handling work" | All error wrapping patterns | Hard (no single keyword) |
| "find rate limiting code" | middleware/, api/throttle | Medium |
| "what handles user notifications" | notifications/, email/, push/ | Hard |

## Comparison Metrics

For each prompt, measure:
1. Files returned by mgrep
2. Files returned by grep (best-effort pattern)
3. Files actually relevant
4. Precision: relevant / returned
5. Recall: found / total relevant

## Token Impact

For each prompt, measure:
1. Tokens to read all grep results
2. Tokens to read all mgrep results
3. Tokens to read only relevant files
4. Savings: (grep_tokens - mgrep_tokens) / grep_tokens
```

---

## 11. Success Criteria

### 11.1 Functional Requirements

| Requirement | Validation |
|-------------|------------|
| mgrep invocable from all P0 agents | Manual test each agent |
| Fallback to grep works | Disable mgrep, verify grep used |
| Telemetry captures invocations | Check mgrep-invocations.jsonl |
| /mgrep skill works | User can invoke directly |
| Scout produces semantic scope | Check mgrep-scope.txt exists |

### 11.2 Performance Requirements

| Metric | Target | Measurement |
|--------|--------|-------------|
| mgrep latency | <3s for typical query | Telemetry DurationMs |
| Fallback latency | <100ms detection | Time to first grep |
| Precision improvement | >50% vs grep | Benchmark comparison |
| Token reduction | >40% on exploration | Compare pre/post |

### 11.3 Reliability Requirements

| Requirement | Validation |
|-------------|------------|
| No agent blocked by mgrep failure | All agents have fallback |
| Telemetry doesn't block execution | Async logging |
| Privacy controls work | .mgrepignore respected |
| Cost tracking accurate | Verify against Mixedbread billing |

---

## 12. Appendix: File Manifest

### New Files

| File | Purpose |
|------|---------|
| `~/.claude/skills/mgrep/SKILL.md` | User-invoked skill definition |
| `pkg/telemetry/mgrep_invocation.go` | Telemetry data types |
| `pkg/telemetry/mgrep_logging.go` | Logging functions |
| `docs/mgrep-integration-spec.md` | This specification |
| `.mgrepignore` | Index exclusion patterns |

### Modified Files

| File | Changes |
|------|---------|
| `~/.claude/routing-schema.json` | Add mgrep engine definition |
| `~/.claude/agents/codebase-search/agent.yaml` | Add mgrep integration |
| `~/.claude/agents/haiku-scout/agent.yaml` | Add semantic phase |
| `~/.claude/agents/librarian/agent.yaml` | Add internal-first strategy |
| `~/.claude/agents/orchestrator/agent.yaml` | Add disambiguation protocol |
| `~/.claude/agents/architect/agent.yaml` | Add pattern discovery |
| `~/.claude/agents/review-orchestrator/agent.yaml` | Add domain detection |
| `~/.claude/agents/memory-archivist/agent.yaml` | Add semantic dedup |
| `~/.claude/CLAUDE.md` | Add mgrep reference section |
| `cmd/gogent-sharp-edge/main.go` | Add mgrep detection |
| `cmd/gogent-ml-export/main.go` | Add mgrep export commands |
| `docs/ARCHITECTURE.md` | Add mgrep integration section |

---

**End of Specification**

*This document is ready for use with `/plan` to generate implementation tickets.*
