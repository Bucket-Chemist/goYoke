# Part VI: Technical Specifications

## **Complete Reference for Implementation**

## **1. Threshold Reference Table**

All numeric thresholds derived from research, with justification and adjustment guidelines.

## **1.1 Routing Thresholds**

| **Threshold**     | **Value**     | **Justifcation**                 | **Adjustment Trigger**                  |
| ----------------- | ------------- | -------------------------------- | --------------------------------------- |
| Haiku             | Score <       | Simple tasks,                    | Increase if Haiku                       |
| ceiling           | 2             | minimal context                  | failures >5%                            |
| Sonnet<br>ceiling | Score<br>2-10 | Standard<br>development<br>work  | Adjust based on<br>cost/quality tradeof |
| Opus foor         | Score ><br>10 | Complex<br>reasoning<br>required | Decrease if over-<br>routing detected   |

| Gemini  |       | Tokens                 | Context window | Fixed (model                |
| ------- | ----- | ---------------------- | -------------- | --------------------------- |
| trigger |       | ><br>50,000            | limits         | constraint)                 |
| Gemini  | force | Tokens<br>><br>100,000 | Safety margin  | Fixed (model<br>constraint) |

## **1.2 Complexity Formula**

SCORE = (tokens / 10000) + (files / 5) + (modules × 2)

| **Component**             | **Weight**   |           |                                    | **Rationale**                |          |
| ------------------------- | ------------ | --------- | ---------------------------------- | ---------------------------- | -------- |
| Tokens                    | 1 per 10K    |           | Primary context consumption        |                              | metric   |
| Files                     | 1 per 5      |           | Cross-fle coordination overhead    |                              |          |
| Modules                   | 2 per module |           | Architectural complexity indicator |                              |          |
| **Example calculations:** |              |           |                                    |                              |          |
| **Scenario**              | **Tokens**   | **Files** |                                    | **Modules**<br>**Score**     | **Tier** |
| Simple fx                 | 5,000        | 1         |                                    | 1<br>0.5 + 0.2 + 2<br>= 2.7  | Sonnet   |
| Feature add               | 25,000       | 8         |                                    | 2<br>2.5 + 1.6 + 4<br>= 8.1  | Sonnet   |
| Refactor                  | 45,000       | 15        |                                    | 4<br>4.5 + 3.0 + 8<br>= 15.5 | Opus     |
| Codebase<br>analysis      | 150,000      | 50        |                                    | 10<br>15 + 10 + 20<br>= 45   | Gemini   |

**Security-Sensitive Code Detection:**

- **Implementation:** `.claude/scripts/calculate-complexity.sh` detects security keywords via `grep`
- **Keywords:** auth, crypto, token, secret, password, credential, jwt, oauth, session
- **Escalation:** If detected AND tier=haiku, force minimum tier=sonnet
- **Rationale:** Security code requires higher reasoning capability to avoid vulnerabilities

## **1.3 Pattern Detection Thresholds**

| **Threshold**                    | **Value**         | **Source**                       | **Purpose**                    |
| -------------------------------- | ----------------- | -------------------------------- | ------------------------------ |
| Min observations<br>for analysis | 200               | Statistical<br>power<br>research | Trigger<br>schema<br>discovery |
| Observations per<br>cluster      | 30                | 80% statistical<br>power         | Valid cluster<br>minimum       |
| Cluster count<br>formula         | N = 30 × k        | k = expected<br>clusters         | Sample size<br>planning        |
| Silhouette score<br>minimum      | 0.5               | Cluster validity<br>standard     | Pattern quality<br>gate        |
| Bootstrap stability              | 80%               | Reproducibility<br>standard      | Pattern<br>reliability         |
| Confdence interval               | 99% (α =<br>0.01) | Production ML<br>standard        | Decision<br>threshold          |

## **1.4 Autonomy Progression Thresholds**

| **Transition** | **Decisions**<br>**Required** | **Success**<br>**Rate** | **Rationale**                 |
| -------------- | ----------------------------- | ----------------------- | ----------------------------- |
| L1 → L2        | 100                           | N/A                     | Suficient observation<br>data |
| L2 → L3        | 200                           | 95%                     | Collaborator trust<br>earned  |
| L3 → L4        | 500                           | 98%                     | Consultant trust<br>earned    |
| L4 → L5        | Human-defned                  | 99%+                    | Domain-specifc full<br>trust  |

**Demotion triggers:** - Success rate drops 5% below promotion threshold - 3+ consecutive failures in category - Human requests demotion

## **1.5 Shadow Deployment Thresholds**

| **Threshold**                 | **Value**          | **Source**                     | **Purpose**            |
| ----------------------------- | ------------------ | ------------------------------ | ---------------------- |
| Min shadow<br>invocations     | 10                 | Statistical<br>signifcance     | Minimum<br>sample      |
| Success rate for<br>promotion | 90%                | Quality gate                   | Promotion<br>criterion |
| Error rate vs<br>baseline     | ≤ 0.1%<br>increase | Risk<br>management             | Safety margin          |
| Max shadow<br>duration        | 14 days            | Prevent<br>deployment<br>limbo | Time bound             |
| Trafic percentage<br>(shadow) | 0% live            | Shadow<br>defnition            | No user impact         |
| Trafic percentage<br>(canary) | 1-5%               | Progressive<br>rollout         | Initial exposure       |

## **1.6 Memory and Retrieval Thresholds**

|      | **Threshold** |     | **Value** | **Purpose** |
| ---- | ------------- | --- | --------- | ----------- |
| BM25 | top-k default | 5   |           | Balance     |

relevance/noise Curation trigger Performance concern Review for archival title, created, category, tags, summary

**==> picture [114 x 45] intentionally omitted <==**

**----- Start of picture text -----**<br>
Memory file limit warning 500 files<br>Memory file limit hard 1000 files<br>Stale memory threshold 90 days<br>Frontmatter requiredfields 5<br>**----- End of picture text -----**<br>

## **1.7 Cost Control Thresholds**

**==> picture [203 x 54] intentionally omitted <==**

**----- Start of picture text -----**<br>
Threshold Value Action<br>Session cost warning $5 Alert user<br>Session cost limit $10 Require confirmation<br>Weekly budget default $50 Review trigger<br>Opus calls per session warning 5 Review routing<br>Weekly review budget $2 Infrastructure investment<br>**----- End of picture text -----**<br>

## **1.8 Document Processing Thresholds**

| **Threshold**           | **Value**         | **Source**           | **Purpose**          |
| ----------------------- | ----------------- | -------------------- | -------------------- |
| Chunk size (factoid)    | 256-512<br>tokens | Industry<br>standard | Q&A retrieval        |
| Chunk size              | 1024+             | Industry             | Complex              |
| (analytical)            | tokens            | standard             | analysis             |
| Chunk size (default)    | 512 tokens        | Starting point       | General use          |
| Overlap percentage      | 10-20%            | NVIDIA<br>research   | Boundary<br>coverage |
| Overlap optimal         | 15%               | FinanceBench<br>2024 | Best recall          |
| Parallel workers<br>max | 5                 | Rate limit<br>safety | Gemini calls         |

## **2. Schema Specifications**

## **2.1 Observation Event Schema**

**==> picture [207 x 349] intentionally omitted <==**

**----- Start of picture text -----**<br>
{<br>"$schema": "http://json-schema.org/draft-07/schema#",<br>"title": "ObservationEvent",<br>"description": "Raw behavioral observation for pattern discovery",<br>"type": "object",<br>"required": ["timestamp", "event_type", "context", "action"],<br>"properties": {<br>"timestamp": {<br>"type": "string",<br>"format": "date-time",<br>"description": "ISO 8601 timestamp"<br>},<br>"event_type": {<br>"type": "string",<br>"enum": [<br>"human_override",<br>"plan_approval",<br>"plan_rejection",<br>"scope_modification",<br>"tier_escalation",<br>"tier_override",<br>"clarification_request",<br>"task_completion",<br>"task_failure",<br>"memory_retrieval",<br>"agent_handoff"<br>]<br>},<br>"context": {<br>"type": "object",<br>"properties": {<br>"task": {"type": "string"},<br>"complexity_score": {"type": "number"},<br>"recommended_tier": {"type": "string"},<br>"files_in_scope": {"type": "integer"},<br>"session_id": {"type": "string"},<br>"tags": {"type": "array", "items": {"type": "string"}}<br>}<br>},<br>"action": {<br>"type": "object",<br>"properties": {<br>"type": {"type": "string"},<br>"from": {"type": "string"},<br>"to": {"type": "string"},<br>"reason_provided": {"type": "string"}<br>},<br>"required": ["type"]<br>},<br>"outcome": {<br>"type": "object",<br>**----- End of picture text -----**<br>

"properties": { "success": {"type": "boolean"}, "duration_seconds": {"type": "integer"}, "error_message": {"type": "string"}

}

}

}

}

## **2.2 Decision Capture Schema**

{

"$schema": "http://json-schema.org/draft-07/schema#", "title": "Decision", "description": "Captured decision for apprenticeship learning",

"type": "object",

"required": [ "schema_version", "decision_id",

"timestamp",

"session_id", "decision_category", "context", "human_decision", "learning_metadata"

], "properties": { "schema_version": { "type": "string", "const": "1.0.0"

}, "decision_id": { "type": "string", "format": "uuid"

}, "timestamp": { "type": "string", "format": "date-time"

}, "session_id": { "type": "string" }, "decision_category": { "type": "string", "enum": [ "routing_override", "scope_modification", "plan_approval", "plan_rejection", "human_escalation", "task_delegation", "memory_curation", "agent_selection"

]

}, "context": { "type": "object", "properties": { "task_type": {"type": "string"}, "task_description": {"type": "string"}, "complexity_score": {"type": "number"}, "files_in_scope": {"type": "integer"}, "estimated_tokens": {"type": "integer"}, "tags": {"type": "array", "items": {"type": "string"}}

}

}, "system_recommendation": { "type": "object", "properties": { "action": {"type": "string"}, "confidence": {"type": "number", "minimum": 0, "maximum": 1}, "reasoning": {"type": "string"} } }, "human_decision": { "type": "object", "required": ["action"], "properties": { "action": {"type": "string"}, "reasoning_provided": {"type": "string"}, "time_to_decide_seconds": {"type": "integer"}

}

}, "outcome": { "type": "object", "properties": { "task_success": {"type": "boolean"}, "quality_rating": {"type": "integer", "minimum": 1, "maximum": 5}, "issues_encountered": {"type": "array", "items": {"type": "string"}}, "would_recommend_same": {"type": "boolean"} } }, "learning_metadata": {

"type": "object", "required": ["autonomy_level_at_time"], "properties": { "autonomy_level_at_time": {"type": "integer", "minimum": 1, "maximum": 5}, "pattern_match_candidates": {"type": "array", "items": {"type": "string"}}, "should_automate_similar": {"type": "boolean"}, "requires_human_always": {"type": "boolean"} }

}

}

}

**==> picture [111 x 9] intentionally omitted <==**

**----- Start of picture text -----**<br>
2.3 Agent Definition Schema<br>**----- End of picture text -----**<br>

{

"$schema": "http://json-schema.org/draft-07/schema#", "title": "AgentDefinition", "description": "Subagent configuration and metadata", "type": "object", "required": ["name", "version", "tier", "purpose"], "properties": { "name": { "type": "string", "pattern": "^[a-z][a-z0-9-]*$", "maxLength": 50 }, "version": { "type": "string", "pattern": "^\\d+\\.\\d+\\.\\d+$" }, "status": { "type": "string", "enum": ["proposed", "shadow", "active", "deprecated", "archived"], "default": "proposed" }, "tier": { "type": "string", "enum": ["haiku", "sonnet", "opus", "gemini"] }, "purpose": { "type": "string", "maxLength": 500 }, "created": { "type": "string", "format": "date" }, "trigger_conditions": { "type": "array", "items": { "type": "object", "properties": { "condition": {"type": "string"}, "value": {} } } }, "configuration": { "type": "object", "additionalProperties": **true** }, "input_schema": { "type": "object", "additionalProperties": {"type": "string"} }, "output_schema": { "type": "object", "additionalProperties": {"type": "string"} }, "constraints": { "type": "object", "properties": { "max_input_tokens": {"type": "integer"}, "max_output_tokens": {"type": "integer"}, "timeout_minutes": {"type": "integer"}, "extended_thinking": {"type": "boolean"} } }, "metrics": { "type": "object", "properties": { "invocations": {"type": "integer", "default": 0}, "successes": {"type": "integer", "default": 0}, "failures": {"type": "integer", "default": 0}, "avg_duration_seconds": {"type": "number"}, "avg_cost_usd": {"type": "number"} } } } }

**2.4 Memory File Schema (YAML Frontmatter)**

**==> picture [265 x 626] intentionally omitted <==**

**----- Start of picture text -----**<br>

# Required fields<br>title : string # Brief descriptive title (max 100 chars)<br>created : date # YYYY-MM-DD format<br>category : enum # decisions | sharp-edges | facts |<br>preferences | observations<br>tags : array[string] # Searchable tags (max 10)<br>summary : string # One-line searchable summary (max 200<br>chars)<br># Optional fields<br>updated : date # Last modification date<br>related : array[string] # Relative paths to related files<br>confidence : enum # high | medium | low<br>status : enum # active | deprecated | archived (default:<br>active)<br>source : string # Session ID or "manual"<br>expires : date # For time-sensitive facts<br>integrity :<br>content_hash : string # sha256:... of body content<br>last_verified : datetime<br>Validation rules: - title required, max 100 characters - created<br>required, valid ISO date - category required, must be enum value -<br>tags required, 1-10 items, each max 30 chars - summary required, max<br>200 characters<br>3. Hook Implementation Specifications<br>3.1 Hook Lifecycle<br>┌────────────────────────────────────────────────────────────────────<br>│ HOOK EXECUTION ORDER<br>│<br>└────────────────────────────────────────────────────────────────────<br>SESSION START<br> │<br> ▼<br>┌─────────────────┐<br>│ SessionStart.sh │ ← Load context, verify environment<br>│ (optional) │<br>└────────┬────────┘<br> │<br> ▼<br> ┌─────────┐<br> │ TASK │<br> │ LOOP │◄──────────────────────────────────────────┐<br> └────┬────┘ │<br> │ │<br> ▼ │<br>┌─────────────────┐ │<br>│ PreToolUse.sh │ ← Routing enforcement, validation │<br>│ (required) │ │<br>└────────┬────────┘ │<br> │ │<br> ┌────┴────┐ │<br> │ exit 0 │ PERMIT │<br> │ exit 2 │ BLOCK ─────────────────────────────┐ │<br> └────┬────┘ │ │<br> │ │ │<br> ▼ │ │<br>┌─────────────────┐ │ │<br>│ TOOL EXECUTION │ │ │<br>└────────┬────────┘ │ │<br> │ │ │<br> ▼ │ │<br>┌─────────────────┐ │ │<br>│ PostToolUse.sh │ ← Log outcome, capture metrics │ │<br>│ (required) │ │ │<br>└────────┬────────┘ │ │<br> │ │ │<br> ▼ │ │<br>┌─────────────────┐ │ │<br>│ SubagentStop.sh │ ← Validate handoff (if agent) │ │<br>│ (conditional) │ │ │<br>└────────┬────────┘ │ │<br> │ │ │<br> └─────────────────────────────────────────┴──────┘<br> │<br> ┌─────────────────────────────────────────┘<br> │<br> ▼<br>┌─────────────────┐<br>│ SessionEnd.sh │ ← Summary, archival trigger<br>│ (optional) │<br>└─────────────────┘<br>**----- End of picture text -----**<br>

**3.2 PreToolUse.sh Specification Purpose:** Enforce routing decisions, validate requests, log decisions

**Exit codes:** | Code | Meaning | Behavior | |——|———|———-| | 0 | PERMIT | Continue with tool execution | | 1 | ERROR | Hook failure, blocks execution | | 2 | BLOCK | Routing violation, blocks with message |

**Environment variables available:**

CLAUDE*TOOL_NAME *# Name of tool being invoked* CLAUDE_SESSION_ID *# Current session identifier* CLAUDE_REQUESTED_TIER *# Tier requested (if applicable)_ CLAUDE_WORKING_DIR _# Current working directory\_

**Required behavior:** 1. Read complexity score from state file 2. Compare requested operation to tier ceiling 3. Log decision to routing_log.jsonl 4. Return appropriate exit code **Template:**

_#!/bin/bash_ set -euo pipefail

_# Configuration_ STATE_DIR="$HOME/.claude/tmp" LOG_FILE="$STATE_DIR/routing_log.jsonl"

_# Read state_ SCORE=$(cat "$STATE_DIR/complexity_score" 2>/dev/null **||** echo "5") CEILING=$(cat "$STATE_DIR/recommended_tier" 2>/dev/null **||** echo "sonnet")

_# Enforcement logic_

_# [implementation here]_

_# Log decision_ echo "{...}" >> "$LOG_FILE"

exit $EXIT_CODE

## **3.3 PostToolUse.sh Specification**

**Purpose:** Capture outcomes, update metrics, detect anomalies

**Exit codes:** | Code | Meaning | Behavior | |——|———|———-| | 0 | SUCCESS | Continue normally | | 1 | ERROR | Log error, continue | **Environment variables available:**

CLAUDE*TOOL_NAME *# Name of tool that executed* CLAUDE_TOOL_EXIT_CODE *# Exit code from tool* CLAUDE_SESSION_ID *# Current session identifier* CLAUDE_TOOL_DURATION *# Execution time in milliseconds\_

**Required behavior:** 1. Read last routing log entry 2. Update with actual outcome 3. Detect sharp edges (errors, unusual patterns) 4. Queue observations for learning

## **3.4 SubagentStop.sh Specification**

**Purpose:** Validate agent completion, verify handoff integrity

**Exit codes:** | Code | Meaning | Behavior | |——|———|———-| | 0 | VALID | Handoff accepted | | 1 | ERROR | Hook failure | | 2 | INVALID | Handoff rejected, blocks |

**Validation checklist:** - [ ] Handoff file exists and is valid JSON - [ ] Required fields present (task_summary, success_criteria) - [ ] Referenced artifacts exist - [ ] Status is “completed” (warn if not) - [ ] Schema version compatible

## **3.5 SessionEnd.sh Specification**

**Purpose:** Generate session summary, trigger archival processes

**Exit codes:** | Code | Meaning | Behavior | |——|———|———-| | 0 | SUCCESS | Clean shutdown | | 1 | ERROR | Log error, continue shutdown |

**Required behavior:** 1. Aggregate session metrics from routing log 2. Generate session summary JSON 3. Archive routing log with session ID 4. Trigger memory archivist (background) 5. Clean up temporary state files

## **4. State File Specifications**

## **4.1 scout_metrics.json**

**Location:** ~/.claude/tmp/scout_metrics.json **Lifetime:** Per-task (refreshed on each scout operation) **Max age:** 5 minutes before considered stale

**Metric Sources (Bash-First Architecture):**

- **PRIMARY:** `~/.claude/scripts/gather-scout-metrics.sh` provides deterministic counts via shell commands (`wc`, `find`, `grep`)
- **FALLBACK:** LLM estimation (legacy mode when Bash metrics unavailable)
- **LLM Role:** Pattern classification (`import_density`), key file identification, confidence assessment
- **NOT LLM Role:** Counting files/lines/tokens (Bash does this deterministically)

{ "schema_version": "1.0.0", "generated_at": "2026-01-13T10:23:45Z", "scout_agent": "haiku",

"scout_report": { "scope_metrics": { "total_files": 15, "file_types": {

".py": 10,

".md": 3, ".json": 2 }, "estimated_tokens": 45000, "largest_file": { "path": "src/auth/handlers.py", "tokens": 8500

}

}, "complexity_signals": { "cross_file_dependencies": 8, "module_count": 3, "circular_imports": 0, "test_coverage_files": 5 }, "recommendations": { "suggested_tier": "sonnet", "gemini_offload_candidates": [], "risk_factors": ["large handlers.py may need splitting"] } }

}

**4.2 complexity_score**

**Location:** ~/.claude/tmp/complexity_score **Lifetime:** Per-task **Format:** Plain text, single decimal number

13.50

**4.3 recommended_tier**

**Location:** ~/.claude/tmp/recommended_tier **Lifetime:** Per-task **Format:** Plain text, tier name

opus

## **4.4 handoff.json**

**Location:** ~/.claude/tmp/handoff.json **Lifetime:** Per-agent-transition **Max size:** 100KB

{ "schema_version": "1.0.0", "handoff_id": "550e8400-e29b-41d4-a716-446655440000", "from_agent": "architect", "to_agent": "executor", "created_at": "2026-01-13T10:25:00Z", "status": "completed", "context": { "task_summary": "Implement OAuth refresh token rotation", "files_in_scope": [ "src/auth/tokens.py", "src/auth/middleware.py"

], "critical_constraints": [ "Must maintain backward compatibility with existing tokens", "Refresh rotation must be atomic" ], "success_criteria": [ "All existing tests pass", "New rotation tests added", "No breaking API changes"

]

}, "artifacts": { "specs_path": ".claude/tmp/specs.md", "scout_metrics_path": ".claude/tmp/scout_metrics.json"

},

"metadata": { "estimated_tokens": 25000, "estimated_duration_minutes": 15, "tier_ceiling": "sonnet"

}

}

## **4.5 routing_log.jsonl**

**Location:** ~/.claude/tmp/routing_log.jsonl **Lifetime:** Per-session (archived on SessionEnd) **Format:** JSON Lines (one JSON object per line)

{"timestamp":"2026-0113T10:23:45Z","session_id":"abc123","decision_id":"uuid","tool_name":"Edit","routing": {"complexity_score":8.5,"calculated_tier":"sonnet","requested_tier":"opus","final_tier":"sonnet","decis exceeds ceiling"},"cost": {"estimated_tokens":35000,"estimated_cost_usd":0.105},"outcome": {"actual_tokens": **null** ,"actual_cost_usd": **null** ,"task_success": **null** }}

## **5. Directory Structure**

## **5.1 Complete Annotated Tree**

- ~/.claude/ ├── hooks/ # Lifecycle hooks (executable scripts) │ ├── PreToolUse.sh # REQUIRED: Routing enforcement │ ├── PostToolUse.sh # REQUIRED: Outcome capture │ ├── SubagentStop.sh # OPTIONAL: Handoff validation │ ├── SessionStart.sh # OPTIONAL: Context loading │ └── SessionEnd.sh # OPTIONAL: Summary/archival │

- ├── scripts/ # Utility scripts

- │ ├── calculate-complexity.sh # Complexity formula

implementation

- │ ├── validate-routing.sh # Tier permission validation

- │ ├── validate-state.py # Schema validation

- │ ├── query-memory-bm25.py # Memory retrieval

- │ ├── log-observation.py # Observation capture

- │ ├── generate-cost-report.sh # Cost aggregation

- │ ├── weekly-review.sh # Review orchestrator

- │ └── verify-memory-integrity.sh # Integrity checking

│

- ├── agents/ # Agent definitions

- │ ├── haiku-scout/

- │ │ ├── agent.yaml # Configuration

- │ │ ├── agent.md # Description

- │ │ └── CLAUDE.md # Instructions

- │ ├── architect/

- │ ├── memory-archivist/

- │ ├── memory-synthesis/

- │ ├── systems-architect/

- │ ├── schema-discovery/ # Phase 4

- │ └── [spawned-agents]/ # Phase 5+

│ │

- ├── skills/ # Workflow definitions

- │ └── explore/

- │ └── SKILL.md

- ├── schemas/ # Schema definitions

- │ ├── state/ # State file schemas

- │ │ ├── scout_metrics.py

- │ │ ├── handoff.py

- │ │ └── routing_log.py

- │ ├── memory/ # Memory schemas

- │ │ ├── decision.py

- │ │ └── observation.py

- │ └── [version]/ # Versioned schemas (Phase 4+) │ ├── tmp/ # Ephemeral state (cleared per session)

- │ ├── scout_metrics.json

- │ ├── complexity_score

- │ ├── recommended_tier

- │ ├── handoff.json

- │ ├── routing_log.jsonl

- │ ├── specs.md

- │ ├── validation_log.jsonl

- │ ├── session_summaries/

- │ │ └── [session_id].json

- │ ├── reviews/

- │ │ └── [date]/

- │ │ ├── synthesis.json

- │ │ ├── architect_report.md

- │ │ └── recommendations.json

- │ └── metrics/ │ └── cost_report.json

- │ ├── memory/ # Persistent learning (git versioned)

- │ ├── decisions/ # Architectural decisions

- │ │ └── YYYY-MM-DD-topic.md

- │ ├── sharp-edges/ # Known pitfalls

- │ │ └── YYYY-MM-DD-issue.md

- │ ├── facts/ # Verified project facts

- │ │ └── topic.md

- │ ├── preferences/ # User preferences

- │ │ └── category.md

- │ ├── observations/ # Raw behavioral observations

- │ │ └── YYYY-MM-DD-observations.jsonl

- │ ├── audit.jsonl # Memory access audit log │ └── index.json # Searchable index (generated) │

- ├── docs/ # Documentation

- │ ├── memory-format.md # Memory file standard

- │ ├── schema-approval.md # Schema approval workflow │ └── review-interview.md # Human interview template │

- ├── settings.json # Global configuration

- ├── routing-schema.json # Routing rules and thresholds ├── agents-index.json # Agent registry with lifecycle └── autonomy-levels.yaml # Autonomy state per category

- ~/.gemini-slave/ # Gemini integration ├── protocols/

│ ├── handover-protocol.md

- │ ├── codebase-analysis.md

- │ └── document-synthesis.md └── config.yaml

## **5.2 File Naming Conventions**

| **Location**             | **Pattern**                    | **Example**                       |
| ------------------------ | ------------------------------ | --------------------------------- |
| Memory<br>decisions      | YYYY-MM-DD-topic.md            | 2026-01-13-jwt-<br>strategy.md    |
| Memory<br>sharp-edges    | YYYY-MM-DD-issue.md            | 2026-01-13-circular-<br>import.md |
| Memory<br>facts          | topic.md                       | project-structure.md              |
| Observations             | YYYY-MM-DD-observations.jsonl  | 2026-01-13-<br>observations.jsonl |
| Session<br>summaries     | [session_id].json              | abc123.json                       |
| Archived<br>routing logs | routing*log*[session_id].jsonl | routing_log_abc123.js             |
| Review<br>directories    | YYYY-MM-DD/                    | 2026-01-13/                       |

**==> picture [209 x 6] intentionally omitted <==**

## **5.3 Lifecycle Management**

| **Directory**          | **Retention**   | **Cleanup Trigger** |
| ---------------------- | --------------- | ------------------- |
| tmp/                   | Session         | SessionEnd hook     |
| tmp/session_summaries/ | 30 days         | Weekly cleanup      |
| tmp/reviews/           | 90 days         | Monthly cleanup     |
| memory/observations/   | Until processed | Schema discovery    |
| memory/decisions/      | Permanent       | Manual archival     |
| memory/sharp-edges/    | Permanent       | Manual archival     |

## **6. API Contracts**

## **6.1 Gemini CLI Interface**

## **Invocation pattern:**

**==> picture [209 x 330] intentionally omitted <==**

**----- Start of picture text -----**<br>
gemini-cli <command> [options]<br>Commands:<br>Command Purpose Key Options<br>--input-dir, --output, --<br>analyze Codebase analysis max-tokens<br>Document<br>summarize summarization --input, --output, --format<br>Multi-document<br>synthesize synthesis --inputs, --output<br>Example invocations:<br># Codebase analysis<br>gemini-cli analyze \<br>--protocol codebase-analysis \<br>--input-dir ./src \<br>--output .claude/tmp/analysis.json \<br>--max-tokens 100000<br># Document synthesis<br>gemini-cli synthesize \<br>--inputs ".claude/tmp/summary\_\*.json" \<br>--output .claude/tmp/synthesis.md \<br>--format markdown<br>Output format:<br>{<br>"status": "success|error",<br>"output_path": "/path/to/output",<br>"metrics": {<br>"input_tokens": 45000,<br>"output_tokens": 3000,<br>"duration_seconds": 12<br>},<br>"error": null<br>}<br>6.2 Inter-Agent Message Format<br>Standard message envelope:<br>{<br>"message_id": "uuid",<br>"timestamp": "ISO8601",<br>**----- End of picture text -----**<br>

"from_agent": "agent-name", "to_agent": "agent-name", "message_type": "request|response|error", "correlation_id": "uuid", "payload": {}, "metadata": { "tier": "sonnet", "priority": "normal|high", "timeout_seconds": 300

}

}

## **6.3 Memory Query Interface**

**Function signature:**

**def** query_memory( query: str, top_k: int = 5, category: Optional[str] = None, tags: Optional[List[str]] = None, min_confidence: Optional[str] = None, since: Optional[date] = None ) -> List[MemoryResult]

**CLI interface:**

python3 ~/.claude/scripts/query-memory-bm25.py \ "authentication JWT refresh" \ --top-k 5 \ --category decisions \

--json

**Result format:**

[

{

"path": "decisions/2026-01-13-jwt-strategy.md", "title": "Decision: JWT Token Refresh Strategy", "category": "decisions", "tags": ["authentication", "jwt", "security"], "score": 8.234, "preview": "Implemented sliding window refresh with..." } ]

## **Summary**

This technical specification provides the concrete implementation details needed to build the GoGent architecture:

1. **Thresholds** — All numeric values with justification and adjustment guidance

2. **Schemas** — Complete JSON Schema definitions for all data structures

3. **Hooks** — Specification for each lifecycle hook with exit codes and behavior

4. **State files** — Format and location for all ephemeral state 5. **Directory structure** — Complete annotated file tree with conventions

5. **APIs** — Contracts for Gemini CLI, inter-agent messaging, and memory queries

These specifications should be treated as the authoritative reference during implementation. Deviations should be documented and justified.
