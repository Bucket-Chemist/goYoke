# Lisan al-Gaib: Technical Manual & System Guide

**Version:** 3.0 (Deep Dive)
**Status:** Operational
**Role:** Definitive Technical Reference

---

## 1. System Philosophy: The Hybrid Swarm

Lisan al-Gaib is a **Tiered Multi-Agent System** that orchestrates two distinct intelligence classes into a cohesive unit. It solves the "Reasoning vs. Context" trade-off by enforcing a strict division of labor.

### The Problem
*   **High Reasoning (Claude 3.5 Sonnet/Opus):** Excellent at coding and architecture, but expensive ($15/M tokens) and context-limited (200k effective).
*   **High Context (Gemini 2.0 Flash):** Massive context window (1M+ tokens), fast, and cheap ($0.10/M tokens), but lower reasoning capability for subtle architectural decisions.

### The Solution: "Brain & Brawn" Architecture
The system projects itself into the user's workspace via two hidden nodes:
1.  **The Brain (`~/.claude/`):** A high-reasoning, low-context hypervisor. It handles planning, architecture, synthesis, and precise implementation.
2.  **The Brawn (`~/.gemini-slave/`):** A high-context, low-reasoning engine. It handles scouting, mapping, bulk analysis, and large-scale pattern matching.

---

## 2. System Anatomy

### 2.1 The Brain: Claude Node (`~/.claude/`)
This directory acts as the **Orchestrator**. It contains the "conscious" agents that interact with the user and write code.

*   **`CLAUDE.md`**: The System Prompt/Hypervisor. It loads conventions and enforces the routing gates.
*   **`routing-schema.json`**: The "Constitution". Defines cost thresholds, tiers (`haiku`, `sonnet`, `opus`, `external`), and allowed tools per tier.
*   **`agents/`**: The Specialized Personas.
    *   **`orchestrator`**: The default router.
    *   **`architect`**: (Sonnet) Generates `specs.md` plans.
    *   **`tech-docs-writer`**: (Haiku+Thinking) Updates documentation.
    *   **`python-pro`**: (Sonnet) Implements Python code.
    *   **`memory-archivist`**: (Haiku) Compresses session state into long-term memory.
*   **`hooks/`**: Bash scripts that enforce rules *before* the LLM can act.
    *   **`validate-routing.sh`**: Blocks Opus from doing Haiku work.
    *   **`sharp-edge-detector.sh`**: Monitors for debugging loops (3+ failures).
    *   **`attention-gate.sh`**: Injects reminders to keep the model focused.

### 2.2 The Brawn: Gemini Node (`~/.gemini-slave/`)
This directory defines the **Context Engine**. Gemini is never "chatted" with; it is invoked via CLI protocols to perform heavy data processing.

*   **`GEMINI.md`**: The System Prompt. Defines the identity "Gemini-3-Pro-Headless" and strictly forbids conversational filler.
*   **`protocols/`**: The "Subagents" of the Brawn. These are distinct instruction sets for specific heavy-lifting tasks.

#### The Gemini Protocols (Subagents)
1.  **`scout`**: ("The Surveyor")
    *   **Role:** Rapidly assess scope (Files, Lines, Tokens) to inform routing.
    *   **Architecture:** **Bash-First Metrics** - Uses `gather-scout-metrics.sh` for deterministic counting, LLM only for pattern classification.
    *   **Metric Sources:**
        *   **PRIMARY:** Bash shell commands (exact counts via `wc`, `find`, `grep`)
        *   **FALLBACK:** LLM estimation (legacy mode when Bash metrics unavailable)
    *   **Output:** JSON metrics + Routing Recommendation (`haiku`, `sonnet`, `external`).
    *   **Security Detection:** Automatically detects auth/crypto/token keywords and escalates to minimum `sonnet` tier.
    *   **Nuance:** Deterministic metric gathering prevents LLM hallucination in routing decisions.
2.  **`mapper`**: ("The Cartographer")
    *   **Role:** Read 50+ files and generate a navigational map.
    *   **Output:** JSON AST-like map (Entry Points, Core Logic, Dependencies).
    *   **Nuance:** Filters noise to help the Architect focus on critical paths.
3.  **`debugger`**: ("The Tracer")
    *   **Role:** Trace a specific error through the entire call stack across modules.
    *   **Output:** Markdown Root Cause Analysis (RCA).
    *   **Nuance:** Can ingest the entire codebase to find where a variable was mutated 10 files ago.
4.  **`architect`**: ("The Analyst")
    *   **Role:** Review module architecture for patterns and anti-patterns.
    *   **Output:** Refactoring strategy (Phase 1, Phase 2...).
5.  **`memory-audit`**: ("The Librarian")
    *   **Role:** Compare "Live Memory" (`.claude/memory/`) against "Static Config" (`.claude/agents/`) to find drift.
    *   **Output:** Drift Analysis Report.
6.  **`benchmark-audit`**: ("The Scorekeeper")
    *   **Role:** Score a benchmark run against `suite.yaml` metrics.
    *   **Output:** Compliance Report (Cost, Routing Accuracy).

---

## 3. The "Reverse Dispatch" Artifact

A unique capability of this architecture is **Bidirectional Autonomy**.

### Forward Dispatch (Standard)
Claude (Brain) needs to understand a large codebase.
*   **Action:** Claude runs `find src | gemini-slave mapper "Map the auth system"`.
*   **Result:** Gemini reads the files and returns a JSON map.

### Reverse Dispatch (The Artifact)
Gemini (Brawn) finds something that requires reasoning while processing.
*   **Scenario:** While mapping, Gemini sees a critical function lacking documentation.
*   **Action:** Gemini (via CLI) calls `gemini-dispatch tech-docs "Generate docstrings for src/auth.py"`.
*   **Mechanism:** The `gemini-dispatch` binary spins up a specific Claude agent (`tech-docs-writer`), executes the task, and returns the result to Gemini's context.

This turns the hierarchy into a **Swarm**, where the Context Engine can autonomously command the Reasoning Engine for surgical precision.

---

## 4. The Routing Engine & Gates

Lisan al-Gaib enforces cost discipline via **Code-Enforced Routing**.

### 4.1 The Complexity Score

**Workflow:**
1. **Scout gathers metrics** via `gather-scout-metrics.sh` (deterministic Bash)
2. **Scout produces JSON** with exact counts (no LLM estimation)
3. **`calculate-complexity.sh` computes score** from JSON metrics

**Formula:** `Score = (Tokens/10k) + (Files/5) + (Modules*2)`

| Score | Tier | Model | Triggered By |
|-------|------|-------|--------------|
| < 2 | Haiku | Claude 3 Haiku | "Find", "Search", "Count" |
| 2-10 | Sonnet | Claude 3.5 Sonnet | "Implement", "Refactor" |
| > 10 | External | Gemini 2.0 Flash | "Map", "Audit", "Trace" |

**Special Cases:**
- **Tokens > 50K:** Force `external` regardless of score
- **Security-Sensitive Code:** Force minimum `sonnet` tier (detects: auth, crypto, token, secret, password, credential, jwt, oauth, session)

### 4.2 The Gates
*   **`scout-selection-gate`**: (Logical, inside `/explore`)
    *   **Files <= 3:** Route to **Haiku Scout**. (Trivial checks, low latency).
    *   **Files > 3:** Route to **Gemini Scout**. (Aggressive offloading for Context Hygiene & Cost).
*   **`validate-routing.sh`**: Fires pre-tool-use. Enforces two layers of routing discipline:
    *   **Layer 1 (Complexity Check):** Checks `complexity_score` vs. current model. Blocks execution if mismatched.
    *   **Layer 2 (Subagent Type Check):** NEW - Validates that agents are invoked with correct subagent_type (see 4.3 below).
*   **`sharp-edge-detector.sh`**: Fires post-tool-use.
    *   **Detection:** Checks for `Error`, `Exception`, `Failed` in output.
    *   **Logic:** If 3 failures on the same file in 5 mins -> **STOP**.
    *   **Action:** Logs to `pending-learnings.jsonl` for archival.

### 4.3 Subagent Type Enforcement (GAP-002 Resolution)

**Problem:** Custom agents (like `tech-docs-writer`) are configured with Write/Edit tools in `agent.yaml`, but when invoked via `Task(subagent_type: "Explore")`, the system-level constraints on `Explore` force read-only mode, overriding agent.yaml permissions.

**Solution:** Moved subagent_type selection from **documentation** (prone to human error) to **programmatic enforcement** (automated verification).

#### The Architecture (Three Layers)

**Layer 1: Schema Definition** (`routing-schema.json`)
- New section `subagent_types` defines tool capabilities for each type:
  - `Explore`: Read-only (tools: Read, Glob, Grep, Bash)
  - `general-purpose`: Full write access (tools: *)
  - `Plan`: Planning mode (tools: Read, Glob, Grep, Write, Task, AskUserQuestion)
  - `Bash`: Shell piping only (tools: Bash, Read)
- New section `agent_subagent_mapping` defines correct subagent_type for each agent:
  - Read-only agents (e.g., `codebase-search`) → `Explore`
  - Write-requiring agents (e.g., `tech-docs-writer`, `scaffolder`) → `general-purpose`
  - Planning agents (e.g., `orchestrator`, `architect`) → `Plan`

**Layer 2: Hook Enforcement** (`validate-routing.sh`)
- Added validation logic (lines 169-227) that:
  1. Extracts target agent from Task prompt (`AGENT: agent-id` pattern)
  2. Looks up correct subagent_type from schema's `agent_subagent_mapping`
  3. Compares provided vs. required subagent_type
  4. **BLOCKS** if write-requiring agent invoked with read-only subagent_type
  5. **WARNS** for other mismatches
  6. Logs violations to `/tmp/claude-routing-violations.jsonl`

**Layer 3: Updated Task Invocation Pattern** (`CLAUDE.md`)
- Changed from: `subagent_type: "Explore", // ALWAYS "Explore"`
- To: `subagent_type: "[from routing-schema.json agent_subagent_mapping]", // ENFORCED`
- Agents no longer rely on documentation; schema is single source of truth.

#### Why This Matters

**Before (Documentation-Based):**
- CLAUDE.md says "use Explore for custom agents"
- Model interprets instructions (or doesn't follow them)
- Agent receives read-only constraint despite agent.yaml permissions
- Silent failure: Agent reports "I cannot write" despite being configured to write
- Debugging takes hours

**After (Schema-Based):**
- routing-schema.json defines agent → subagent_type mapping
- validate-routing.sh enforces before execution
- Wrong subagent_type is caught immediately with clear error
- Costs $0 (prevents failed execution), saves debugging tokens
- Consistent behavior across all sessions

#### Common Subagent Type Mappings

| Agent | Subagent Type | Reason |
|-------|---|---|
| `codebase-search` | `Explore` | Read-only reconnaissance |
| `tech-docs-writer` | `general-purpose` | Writes/edits documentation |
| `scaffolder` | `general-purpose` | Generates boilerplate code |
| `python-pro` | `general-purpose` | Implements Python code |
| `orchestrator` | `Plan` | Coordination and planning |
| `architect` | `Plan` | Generates architecture specs |

See `routing-schema.json` for complete mapping.

---

## 5. The Memory Pipeline

Memory is a structured filesystem, not a vector DB. This allows for precise RAG and manual debugging.

### 5.1 Structure (`~/.claude/memory/`)
*   **`decisions/`**: `specs.md` files from the Architect. ("Why we did X")
*   **`sharp-edges/`**: Learned anti-patterns. ("What broke")
*   **`facts/`**: Project truths. ("The DB is Postgres 14")
*   **`preferences/`**: User style guides.

### 5.2 The Archivist Loop
1.  **Session:** `sharp-edge-detector` logs errors to `pending-learnings.jsonl`.
2.  **End:** `memory-archivist` (Haiku) wakes up.
    *   Reads `pending-learnings.jsonl`.
    *   Compresses them into permanent `sharp-edges/YYYY-MM-DD-error.md`.
    *   Moves `specs.md` to `decisions/`.
3.  **Audit:** Periodically, `gemini-slave memory-audit` runs to ensure these learnings are codified into agent configs (`agent.yaml`).

---

## 6. The `/explore` Workflow (Full Trace)

When a user asks: `/explore "Refactor Auth"`

1.  **Ack:** System acknowledges goal.
2.  **Scout (Gemini):**
    *   `find src | gemini-slave scout "Refactor Auth"`
    *   Output: `scout_metrics.json` (Files: 15, Tokens: 45k).
3.  **Math:** `calculate-complexity.sh` -> Score: 12.5.
4.  **Route:** Score > 10 -> **Tier: External**.
5.  **Map (Gemini):**
    *   `find src | gemini-slave mapper "Map Auth"`
    *   Output: `map.json` (Entry points, dependencies).
6.  **Plan (Claude - Architect):**
    *   Reads `scout_metrics.json` and `map.json`.
    *   Writes `specs.md` (Phased Plan).
7.  **Approve:** User confirms.
8.  **Execute (Claude - Python Pro):**
    *   Implements Phase 1.
    *   `validate-routing.sh` ensures it stays on Sonnet.
9.  **Archive (Haiku - Archivist):**
    *   Moves `specs.md` to Memory.

---

## 7. Benchmarking

The system self-validates using `suite.yaml`.

*   **Command:** `.claude/hooks/benchmark-logger.sh run suite.yaml`
*   **Auditor:** `gemini-slave benchmark-audit` scores the run.
*   **Metrics:** Cost Efficiency, Routing Accuracy, Attention Retention.

---

## 8. Summary

**Lisan al-Gaib** is an operational doctrine that:
1.  **Separates** Reasoning (Claude) from Context (Gemini).
2.  **Enforces** this separation via Bash hooks.
3.  **Learns** via a structured Memory Pipeline.
4.  **Collaborates** via Bidirectional Dispatch.

This structure ensures that for any given task, the **most cost-effective capable agent** is always the one executing it.
