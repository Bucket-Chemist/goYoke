# GOgent Team-Run Framework

## A Standardised Multi-Agent Orchestration System for LLM Workflows

**Version:** 1.0.0
**Status:** Production-validated (TC-013a review, TC-013b braintrust)
**Date:** 2026-02-08

---

# Part I: Framework Overview & Commercial Extensibility

## 1. What This Is

GOgent Team-Run is a **language-agnostic, schema-driven multi-agent orchestration framework** that turns any complex LLM task into a reproducible, observable, budget-controlled background process.

It solves the fundamental problem of multi-agent LLM systems: **how do you reliably coordinate multiple AI agents working on the same problem, track their costs, ensure structured outputs, and deliver results without blocking the user?**

### The Core Innovation

Most multi-agent systems treat orchestration as prompt engineering — one LLM spawns another and parses free-text responses. This approach is fragile, unobservable, and impossible to budget.

GOgent Team-Run introduces **industrial-grade orchestration primitives**:

| Primitive | What It Does | Analogy |
|-----------|-------------|---------|
| **Team Templates** | Declarative workflow definitions (JSON) | Kubernetes Deployment manifests |
| **Stdin/Stdout Contracts** | Typed I/O schemas per agent role | gRPC service definitions |
| **Wave Execution** | Dependency-ordered parallel groups | CI/CD pipeline stages |
| **Inter-Wave Scripts** | Data transformation between stages | ETL pipeline operators |
| **Budget Gates** | Pre-spawn cost reservation + reconciliation | Cloud resource quotas |
| **Process Registry** | PID tracking with graceful shutdown | Process supervisor (systemd) |
| **Envelope Builder** | Prompt construction from schema + stdin | Template engine with validation |

### What Makes This Commercially Viable

1. **Any LLM Backend**: The framework spawns agents as CLI processes. Any LLM with a CLI interface (Claude, GPT, Gemini, local models via Ollama) can be a worker. The `model` field in team configs is a string — swap the runtime, keep the orchestration.

2. **Any Workflow**: The wave + stdin/stdout contract pattern is generic. Two validated production workflows (code review, deep analysis) prove the pattern works for fundamentally different task types. Adding a new workflow requires only JSON schemas and a slash command — zero Go code changes.

3. **Any Scale**: Single-developer CLI tool today, team server tomorrow. The file-based coordination (JSON configs, stdin/stdout files) maps directly to object storage. The process registry maps to container orchestration. The budget system maps to organisational cost allocation.

4. **Observable by Default**: Every execution produces a team directory with complete audit trail — config, inputs, outputs, logs, costs, timing. No additional instrumentation needed.

5. **Deterministic Structure**: LLM outputs are notoriously unstructured. The envelope builder embeds the exact JSON schema into every agent's prompt, and the three-tier stdout extraction (direct JSON parse → code block extraction → raw text fallback) guarantees machine-readable outputs regardless of agent compliance level.

---

## 2. Architecture at a Glance

```
User invokes /skill
     |
     v
+------------------+
|    Router        |  Classifies request, dispatches to orchestrator
+------------------+
     |
     v
+------------------+
|  Orchestrator    |  Conducts interview, generates team config
|  (e.g. Mozart)   |  Writes: config.json + stdin_*.json files
+------------------+
     |
     v
+------------------+
|  gogent-team-run |  Go binary, runs as background daemon
|                  |
|  For each wave:  |
|    1. Reserve budget per member
|    2. Build prompt envelope (stdin + stdout schema)
|    3. Spawn agent CLI processes in parallel
|    4. Capture output, extract structured JSON
|    5. Reconcile actual cost against reservation
|    6. Run inter-wave script (if configured)
|    7. Advance to next wave
+------------------+
     |
     v
+------------------+
|  Team Directory  |  Complete audit trail on disk
|                  |
|  config.json     |  Status, budget, wave progress
|  stdin_*.json    |  Typed inputs per agent
|  stdout_*.json   |  Structured outputs per agent
|  runner.log      |  Execution log
|  pre-synthesis.md|  Inter-wave artifacts
+------------------+
     |
     v
+------------------+
| /team-status     |  Real-time progress monitoring
| /team-result     |  Final output retrieval
+------------------+
```

---

## 3. The Schema Stack

The framework has four schema layers, each building on the previous:

### Layer 1: Team Template (`schemas/teams/{workflow}.json`)

Defines the **execution topology** — how many waves, which agents in each wave, budget limits, timeouts.

```json
{
  "workflow_type": "my-workflow",
  "budget_max_usd": 10.0,
  "waves": [
    {
      "wave_number": 1,
      "members": [
        {"name": "analyst", "agent": "my-analyst", "model": "sonnet", ...},
        {"name": "reviewer", "agent": "my-reviewer", "model": "haiku", ...}
      ],
      "on_complete_script": "my-inter-wave-script"
    },
    {
      "wave_number": 2,
      "members": [
        {"name": "synthesiser", "agent": "my-synthesiser", "model": "opus", ...}
      ],
      "on_complete_script": null
    }
  ]
}
```

**Key properties per member:**
- `agent`: ID from the agent registry (maps to prompt file + CLI flags)
- `model`: LLM model to use (haiku/sonnet/opus or custom)
- `stdin_file` / `stdout_file`: I/O file names within team directory
- `timeout_ms`: Maximum execution time
- `max_retries`: Automatic retry on failure

### Layer 2: Stdin Schema (`schemas/stdin/{agent-type}.json`)

Defines the **typed input contract** for each agent role. JSON Schema with validation.

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "required": ["agent", "workflow", "context", "task_input"],
  "properties": {
    "agent": {"const": "my-analyst"},
    "workflow": {"const": "my-workflow"},
    "context": {
      "required": ["project_root", "team_dir"],
      "properties": {
        "project_root": {"type": "string", "pattern": "^/"},
        "team_dir": {"type": "string", "pattern": "^/"}
      }
    },
    "task_input": { ... },
    "description": {"type": "string"}
  }
}
```

**Universal fields** (every stdin schema):
- `agent`: Constant identifying the agent role
- `workflow`: Constant identifying the workflow type
- `context.project_root`: Absolute path to user's project
- `context.team_dir`: Absolute path to team execution directory
- `description`: Human-readable summary for envelope builder

### Layer 3: Stdin/Stdout Contract (`schemas/teams/stdin-stdout/{workflow}-{agent}.json`)

Defines **both** the input structure AND the expected output structure for a specific agent in a specific workflow. This is the contract that the envelope builder embeds into the agent's prompt.

```json
{
  "$comment": "Contract for my-analyst in my-workflow",
  "stdin": {
    "agent": "my-analyst",
    "workflow": "my-workflow",
    "task_input": { "question": "...", "context": "..." }
  },
  "stdout": {
    "analysis_id": "analyst-{timestamp}",
    "status": "complete|partial|failed",
    "findings": [ ... ],
    "metadata": { "duration_ms": 0 }
  }
}
```

The `stdout` section is what gets embedded in the agent's prompt as "Expected Output Format". The agent produces JSON matching this structure, which `writeStdoutFile()` extracts and writes to disk.

### Layer 4: Agent Registry (`agents-index.json`)

Defines **agent capabilities** — model, CLI flags, allowed tools, spawning relationships.

```json
{
  "id": "my-analyst",
  "model": "sonnet",
  "cli_flags": ["--permission-mode", "delegate", "--allowedTools", "Read,Glob,Grep"],
  "spawned_by": ["router", "my-orchestrator"],
  "can_spawn": []
}
```

### How the Layers Compose

```
Team Template         →  "Run analyst + reviewer in Wave 1, synthesiser in Wave 2"
  ↓
Stdin Schema          →  "Analyst expects these typed fields as input"
  ↓
Stdin/Stdout Contract →  "Analyst must produce JSON with these exact output fields"
  ↓
Agent Registry        →  "Analyst runs on Sonnet with Read,Glob,Grep tools"
  ↓
Envelope Builder      →  Combines stdin data + stdout schema into a single prompt
  ↓
Claude CLI            →  Executes the agent with the constructed prompt
  ↓
Stdout Extraction     →  Parses response, extracts JSON, writes stdout file
  ↓
Budget Reconciliation →  Extracts actual cost from CLI output, updates config
```

---

## 4. Extensibility Surface

### Adding a New Workflow (Zero Code Changes)

A workflow is defined entirely by JSON schemas. To add a new workflow:

| Step | Artifact | Purpose |
|------|----------|---------|
| 1 | `schemas/teams/{workflow}.json` | Team template (waves, members, budgets) |
| 2 | `schemas/stdin/{agent-type}.json` | Input schema per agent role (if new role) |
| 3 | `schemas/teams/stdin-stdout/{workflow}-{agent}.json` | I/O contract per agent (stdin example + stdout schema) |
| 4 | `skills/{workflow}/SKILL.md` | Slash command definition |
| 5 | `agents/{agent}/` | Agent prompt file (if new agent) |

**No Go binary changes required.** The team-run binary reads schemas at runtime.

### Adding a New Agent Role (Zero Code Changes)

An agent role is defined by its prompt file and registry entry:

| Step | Artifact | Purpose |
|------|----------|---------|
| 1 | `agents/{agent}/{agent}.md` | Agent behaviour prompt |
| 2 | Entry in `agents-index.json` | Model, tools, CLI flags, spawn relationships |
| 3 | `schemas/stdin/{agent}.json` | Input schema (if unique) |

### Adding an Inter-Wave Script

Inter-wave scripts transform Wave N outputs into Wave N+1 inputs. They are standalone binaries that:

1. Accept one argument: the team directory path
2. Read `stdout_*.json` files from the completed wave
3. Write transformation artifacts (e.g., `pre-synthesis.md`)
4. Exit 0 on success, non-zero on failure

The team-run binary calls them automatically between waves when `on_complete_script` is set.

### Extension Points Summary

| Extension | Requires Code? | Requires Schema? | Example |
|-----------|---------------|-------------------|---------|
| New workflow | No | Yes (3-5 JSON files) | `/deep-debug`, `/security-audit` |
| New agent role | No | Yes (1-2 JSON files) | `security-reviewer`, `ux-analyst` |
| New inter-wave script | Yes (Go binary) | No | `gogent-team-merge-findings` |
| New model backend | Yes (spawn.go) | No | OpenAI, local Ollama |
| New budget strategy | Yes (cost.go) | No | Per-org limits, usage tiers |
| New output format | No | Yes (stdout contract) | XML, YAML, custom |

---

## 5. Production Validation Evidence

### Review Workflow (TC-013a)

| Metric | Result |
|--------|--------|
| Config generation pass rate | 100% (11/11 files across 3 scenarios) |
| Structured stdout compliance | 100% (`$schema` field present, JSON parseable) |
| Cost extraction accuracy | Exact (CLI JSON array parsing) |
| Budget reconciliation | Correct ($2.00 → $1.93 after $0.068 haiku run) |
| End-to-end runtime | 38 seconds (1 haiku reviewer) |

### Braintrust Workflow (TC-013b)

| Metric | Result |
|--------|--------|
| Wave 1 parallel execution | Confirmed (2 Opus agents, same-second start) |
| Inter-wave script | `gogent-team-prepare-synthesis` produced 45KB `pre-synthesis.md` |
| Wave 2 synthesis | Beethoven read pre-synthesis, produced structured output |
| Structured stdout compliance | 100% (all 3 Opus agents: `$schema` present, full contract compliance) |
| Total cost | $2.48 (Einstein $0.95, Staff-Arch $1.13, Beethoven $0.40) |
| Total runtime | 6.5 minutes (Wave 1: 3.7min parallel, Wave 2: 2.8min) |
| Budget tracking | Accurate (reserved $15, actual $2.48, remaining $13.52) |

### Key Insight: Cost Estimates vs Actuals

The framework's conservative $5.00/Opus estimate resulted in 83% overestimation. Real-world costs per agent were $0.40-$1.13. This means the budget system provides significant safety margin while actual costs are far lower than expected — a strong commercial selling point.

---

## 6. Commercial Positioning

### For Individual Developers

- **One command** to run a multi-agent analysis (`/braintrust "my question"`)
- **Background execution** — work continues while agents think
- **Cost visibility** — know exactly what each agent costs before and after
- **Audit trail** — every input, output, and decision logged to disk

### For Teams & Organisations

- **Standardised workflows** — every team member runs the same analysis pipeline
- **Budget controls** — organisational spending limits per workflow, per user, per day
- **Custom agents** — add domain-specific reviewers (compliance, security, accessibility)
- **Custom workflows** — compose agents into pipelines matching internal processes
- **Reproducibility** — re-run any analysis from its team directory (stdin files are frozen inputs)

### For Platform Builders

- **White-label orchestration** — the framework is model-agnostic and runtime-agnostic
- **Schema marketplace** — share and discover workflow templates
- **Metered execution** — built-in cost tracking enables pay-per-analysis pricing
- **Multi-tenant** — file-based isolation, no shared state between team runs

---

# Part II: Implementation Guide — Creating a New Multi-Agent Skill

## 7. Step-by-Step: Building a `/security-audit` Skill

This section walks through creating a complete new multi-agent skill from scratch, demonstrating every extension point.

### 7.1 Define the Workflow

**Goal:** A security audit that runs 3 specialised reviewers in parallel (OWASP, dependency, secrets), then synthesises findings.

**Wave structure:**
```
Wave 1 (parallel):  owasp-reviewer + dependency-reviewer + secrets-scanner
Wave 2 (sequential): security-synthesiser (reads all Wave 1 outputs)
```

### 7.2 Create the Team Template

**File:** `schemas/teams/security-audit.json`

```json
{
  "$schema": "./team-config.json",
  "version": "1.0.0",
  "team_name": "security-audit-{timestamp}",
  "workflow_type": "security-audit",
  "project_root": "/absolute/path",
  "session_id": "{uuid}",
  "created_at": "{ISO-8601}",
  "background_pid": null,
  "budget_max_usd": 5.0,
  "budget_remaining_usd": 5.0,
  "warning_threshold_usd": 4.0,
  "status": "pending",
  "started_at": null,
  "completed_at": null,
  "waves": [
    {
      "wave_number": 1,
      "description": "Parallel security analysis",
      "members": [
        {
          "name": "owasp-reviewer",
          "agent": "owasp-reviewer",
          "model": "sonnet",
          "stdin_file": "stdin_owasp-reviewer.json",
          "stdout_file": "stdout_owasp-reviewer.json",
          "status": "pending",
          "process_pid": null,
          "exit_code": null,
          "cost_usd": 0.0,
          "cost_status": "",
          "error_message": "",
          "retry_count": 0,
          "max_retries": 2,
          "timeout_ms": 180000,
          "started_at": null,
          "completed_at": null
        },
        {
          "name": "dependency-reviewer",
          "agent": "dependency-reviewer",
          "model": "haiku",
          "stdin_file": "stdin_dependency-reviewer.json",
          "stdout_file": "stdout_dependency-reviewer.json",
          "status": "pending",
          "process_pid": null,
          "exit_code": null,
          "cost_usd": 0.0,
          "cost_status": "",
          "error_message": "",
          "retry_count": 0,
          "max_retries": 2,
          "timeout_ms": 120000,
          "started_at": null,
          "completed_at": null
        },
        {
          "name": "secrets-scanner",
          "agent": "secrets-scanner",
          "model": "haiku",
          "stdin_file": "stdin_secrets-scanner.json",
          "stdout_file": "stdout_secrets-scanner.json",
          "status": "pending",
          "process_pid": null,
          "exit_code": null,
          "cost_usd": 0.0,
          "cost_status": "",
          "error_message": "",
          "retry_count": 0,
          "max_retries": 2,
          "timeout_ms": 120000,
          "started_at": null,
          "completed_at": null
        }
      ],
      "on_complete_script": "gogent-team-merge-findings"
    },
    {
      "wave_number": 2,
      "description": "Security finding synthesis and risk scoring",
      "members": [
        {
          "name": "security-synthesiser",
          "agent": "security-synthesiser",
          "model": "sonnet",
          "stdin_file": "stdin_security-synthesiser.json",
          "stdout_file": "stdout_security-synthesiser.json",
          "status": "pending",
          "process_pid": null,
          "exit_code": null,
          "cost_usd": 0.0,
          "cost_status": "",
          "error_message": "",
          "retry_count": 0,
          "max_retries": 1,
          "timeout_ms": 600000,
          "started_at": null,
          "completed_at": null
        }
      ],
      "on_complete_script": null
    }
  ]
}
```

**Design decisions:**
- Wave 1 reviewers use haiku/sonnet (cheap, parallel) — OWASP needs reasoning so gets sonnet
- Wave 2 synthesiser uses sonnet (needs to reason across findings)
- `on_complete_script` merges Wave 1 findings into a unified input for the synthesiser
- Budget $5.00 is generous for 3 haiku/sonnet + 1 sonnet run
- `max_retries: 2` for reviewers (idempotent), `1` for synthesiser (not idempotent)

### 7.3 Create Stdin Schemas

**File:** `schemas/stdin/security-reviewer.json` (shared by all 3 Wave 1 agents)

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "Security Reviewer Stdin Schema",
  "type": "object",
  "required": ["agent", "workflow", "context", "audit_scope", "focus_area"],
  "properties": {
    "agent": {
      "type": "string",
      "enum": ["owasp-reviewer", "dependency-reviewer", "secrets-scanner"]
    },
    "workflow": {
      "type": "string",
      "const": "security-audit"
    },
    "context": {
      "type": "object",
      "required": ["project_root", "team_dir"],
      "properties": {
        "project_root": {"type": "string", "pattern": "^/"},
        "team_dir": {"type": "string", "pattern": "^/"}
      }
    },
    "audit_scope": {
      "type": "object",
      "required": ["files", "languages"],
      "properties": {
        "files": {
          "type": "array",
          "items": {
            "type": "object",
            "required": ["path", "language"],
            "properties": {
              "path": {"type": "string"},
              "language": {"type": "string"},
              "contains_user_input": {"type": "boolean"},
              "contains_auth_logic": {"type": "boolean"},
              "contains_crypto": {"type": "boolean"}
            }
          }
        },
        "languages": {"type": "array", "items": {"type": "string"}},
        "entry_points": {"type": "array", "items": {"type": "string"}},
        "external_dependencies": {"type": "array", "items": {"type": "string"}}
      }
    },
    "focus_area": {
      "type": "object",
      "description": "Varies by agent — OWASP categories, dependency CVEs, or secrets patterns"
    },
    "description": {
      "type": "string"
    }
  },
  "additionalProperties": false
}
```

**File:** `schemas/stdin/security-synthesiser.json`

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "Security Synthesiser Stdin Schema",
  "type": "object",
  "required": ["agent", "workflow", "context", "merged_findings_path"],
  "properties": {
    "agent": {"type": "string", "const": "security-synthesiser"},
    "workflow": {"type": "string", "const": "security-audit"},
    "context": {
      "type": "object",
      "required": ["project_root", "team_dir"],
      "properties": {
        "project_root": {"type": "string", "pattern": "^/"},
        "team_dir": {"type": "string", "pattern": "^/"}
      }
    },
    "merged_findings_path": {
      "type": "string",
      "pattern": "^/",
      "description": "Path to merged findings from Wave 1, generated by inter-wave script"
    },
    "description": {"type": "string"}
  },
  "additionalProperties": false
}
```

### 7.4 Create Stdin/Stdout Contracts

One contract file per agent per workflow. These define both the example input AND the required output structure.

**File:** `schemas/teams/stdin-stdout/security-audit-owasp.json`

```json
{
  "$comment": "Contract for owasp-reviewer in security-audit workflow",
  "stdin": {
    "agent": "owasp-reviewer",
    "workflow": "security-audit",
    "description": "OWASP Top 10 security review",
    "context": {
      "project_root": "/path/to/project",
      "team_dir": "/path/to/team/dir"
    },
    "audit_scope": {
      "files": [
        {"path": "src/auth/handler.go", "language": "go", "contains_user_input": true, "contains_auth_logic": true}
      ],
      "languages": ["go"],
      "entry_points": ["cmd/server/main.go"],
      "external_dependencies": ["github.com/golang-jwt/jwt/v5"]
    },
    "focus_area": {
      "owasp_categories": [
        "A01:2021-Broken Access Control",
        "A02:2021-Cryptographic Failures",
        "A03:2021-Injection",
        "A07:2021-Identification and Authentication Failures"
      ]
    }
  },
  "stdout": {
    "reviewer": "owasp-reviewer",
    "status": "complete|partial|failed",
    "overall_risk": "CRITICAL|HIGH|MEDIUM|LOW|NONE",
    "findings": [
      {
        "id": "OWASP-1",
        "category": "A01:2021-Broken Access Control",
        "severity": "CRITICAL|HIGH|MEDIUM|LOW|INFO",
        "title": "Brief vulnerability title",
        "description": "Detailed description of the vulnerability",
        "file": "relative/path/to/file.go",
        "line": 42,
        "cwe": "CWE-XXX",
        "evidence": "Code snippet or reasoning",
        "impact": "What an attacker could achieve",
        "remediation": "Specific fix recommendation",
        "references": ["https://owasp.org/..."]
      }
    ],
    "files_audited": 0,
    "owasp_coverage": {
      "categories_checked": ["A01", "A02", "A03"],
      "categories_with_findings": ["A03"]
    },
    "metadata": {
      "duration_ms": 0,
      "thinking_budget_used": 0
    }
  }
}
```

Similar contracts for `security-audit-dependency.json` and `security-audit-secrets.json`, each with domain-specific stdout fields.

**File:** `schemas/teams/stdin-stdout/security-audit-security-synthesiser.json`

```json
{
  "$comment": "Contract for security-synthesiser in security-audit workflow",
  "stdin": {
    "agent": "security-synthesiser",
    "workflow": "security-audit",
    "description": "Synthesise security findings into risk-scored report",
    "context": {
      "project_root": "/path/to/project",
      "team_dir": "/path/to/team/dir"
    },
    "merged_findings_path": "/path/to/team/dir/merged-findings.md"
  },
  "stdout": {
    "report_id": "security-audit-{timestamp}",
    "status": "complete",
    "executive_summary": "Overall security posture assessment",
    "risk_score": {
      "overall": "CRITICAL|HIGH|MEDIUM|LOW",
      "by_category": {
        "owasp": "HIGH",
        "dependencies": "MEDIUM",
        "secrets": "LOW"
      }
    },
    "critical_findings": [
      {
        "id": "CRIT-1",
        "source": "owasp-reviewer|dependency-reviewer|secrets-scanner",
        "title": "Finding title",
        "risk_score": 9.5,
        "remediation_priority": 1,
        "estimated_fix_effort": "1 hour"
      }
    ],
    "recommendations": {
      "immediate": ["Fix within 24 hours"],
      "short_term": ["Fix within 1 week"],
      "long_term": ["Architectural improvements"]
    },
    "compliance_impact": {
      "affected_standards": ["SOC2", "PCI-DSS"],
      "blocking_deployment": true
    },
    "metadata": {
      "findings_from_owasp": 0,
      "findings_from_dependency": 0,
      "findings_from_secrets": 0,
      "total_findings": 0,
      "duration_ms": 0
    }
  }
}
```

### 7.5 Create Agent Prompt Files

**File:** `agents/owasp-reviewer/owasp-reviewer.md`

```yaml
---
id: owasp-reviewer
name: OWASP Security Reviewer
description: Reviews code for OWASP Top 10 vulnerabilities
model: sonnet
tier: 2
category: security
subagent_type: Explore
tools:
  - Read
  - Glob
  - Grep
---

# OWASP Security Reviewer

You are a security analyst specialising in OWASP Top 10 vulnerabilities.

## Your Task

Review the provided source files for security vulnerabilities. Focus on
the OWASP categories specified in your input.

## How to Work

1. Read each file in `audit_scope.files`
2. For files with `contains_user_input: true`, focus on injection attacks
3. For files with `contains_auth_logic: true`, focus on access control
4. For files with `contains_crypto: true`, focus on cryptographic failures
5. Report each finding with file, line, CWE, and remediation

## Output

Produce structured JSON matching your stdout contract.
```

### 7.6 Register Agents

Add entries to `agents-index.json`:

```json
{
  "id": "owasp-reviewer",
  "model": "sonnet",
  "cli_flags": ["--permission-mode", "delegate", "--allowedTools", "Read,Glob,Grep"],
  "spawned_by": ["router"],
  "can_spawn": []
},
{
  "id": "dependency-reviewer",
  "model": "haiku",
  "cli_flags": ["--permission-mode", "delegate", "--allowedTools", "Read,Glob,Grep,Bash"],
  "spawned_by": ["router"],
  "can_spawn": []
},
{
  "id": "secrets-scanner",
  "model": "haiku",
  "cli_flags": ["--permission-mode", "delegate", "--allowedTools", "Read,Glob,Grep"],
  "spawned_by": ["router"],
  "can_spawn": []
},
{
  "id": "security-synthesiser",
  "model": "sonnet",
  "cli_flags": ["--permission-mode", "delegate", "--allowedTools", "Read,Glob,Grep"],
  "spawned_by": ["router"],
  "can_spawn": []
}
```

### 7.7 Create the Inter-Wave Script

**File:** `cmd/gogent-team-merge-findings/main.go`

A Go binary that:
1. Reads `stdout_owasp-reviewer.json`, `stdout_dependency-reviewer.json`, `stdout_secrets-scanner.json`
2. Extracts findings from each, normalises severity
3. Writes `merged-findings.md` with all findings grouped by severity
4. Exits 0 (or gracefully degrades with `(unavailable: ...)` if any input is missing)

This follows the exact same pattern as `gogent-team-prepare-synthesis` — the braintrust inter-wave script that is already production-validated.

### 7.8 Create the Slash Command

**File:** `skills/security-audit/SKILL.md`

```yaml
---
name: security-audit
description: Multi-agent security audit with OWASP, dependency, and secrets review
version: 1.0.0
---

# Security Audit Skill

## Invocation

| Command | Behaviour |
|---------|-----------|
| `/security-audit` | Audit all changed files |
| `/security-audit src/auth/` | Audit specific directory |

## Execution

1. Router scans changed files (or specified path)
2. Classifies files by security relevance
3. Generates team config + stdin files
4. Launches `gogent-team-run` in background
5. Returns immediately with team ID

## Monitoring

- `/team-status` — see which reviewers have completed
- `/team-result` — view final synthesised security report
```

### 7.9 Schema Resolution

The envelope builder resolves stdout schemas via candidate-based lookup:

```
resolveStdoutSchema("security-audit", "owasp-reviewer")
  → Try: security-audit-owasp-reviewer.json    ✓ (exact match)

resolveStdoutSchema("security-audit", "security-synthesiser")
  → Try: security-audit-security-synthesiser.json  ✓ (exact match)
```

If the agent ID has a suffix like `-reviewer`, the builder also tries the stripped form:

```
resolveStdoutSchema("security-audit", "owasp-reviewer")
  → Try: security-audit-owasp-reviewer.json    ✓ (exact)
  → Would also try: security-audit-owasp.json  (suffix-stripped, not needed)
  → Would also try: security-audit-worker.json (generic fallback)
```

### 7.10 Directory Structure After Execution

```
teams/20260208.150000.security-audit/
  config.json                          Status, budget, wave progress
  stdin_owasp-reviewer.json            OWASP reviewer input
  stdin_dependency-reviewer.json       Dependency reviewer input
  stdin_secrets-scanner.json           Secrets scanner input
  stdin_security-synthesiser.json      Synthesiser input
  stdout_owasp-reviewer.json           OWASP findings (structured JSON)
  stdout_dependency-reviewer.json      Dependency findings (structured JSON)
  stdout_secrets-scanner.json          Secrets findings (structured JSON)
  merged-findings.md                   Inter-wave: merged from all 3
  stdout_security-synthesiser.json     Final report (structured JSON)
  runner.log                           Execution log
  heartbeat                            Liveness indicator
```

---

## 8. Implementation Checklist for Any New Skill

```
PRE-IMPLEMENTATION
  [ ] Define wave structure (how many waves, which agents, parallelism)
  [ ] Choose models per agent (haiku for mechanical, sonnet for reasoning, opus for complex)
  [ ] Estimate budget (use $0.07/haiku, $0.50/sonnet, $1.50/opus as baselines)
  [ ] Decide if inter-wave script is needed

SCHEMA CREATION (no code required)
  [ ] Team template:     schemas/teams/{workflow}.json
  [ ] Stdin schemas:     schemas/stdin/{agent-type}.json (one per unique role)
  [ ] Stdout contracts:  schemas/teams/stdin-stdout/{workflow}-{agent}.json (one per agent)

AGENT CREATION (no code required)
  [ ] Prompt files:      agents/{agent}/{agent}.md
  [ ] Registry entries:  agents-index.json (model, flags, tools, spawn relationships)

SKILL CREATION (no code required)
  [ ] Skill definition:  skills/{workflow}/SKILL.md

OPTIONAL: INTER-WAVE SCRIPT (requires Go code)
  [ ] Binary:            cmd/gogent-team-{script-name}/main.go
  [ ] Build & install:   go install && ln -sf ~/go/bin/{name} ~/.local/bin/
  [ ] Graceful fallback: Handle missing/malformed inputs without crashing

VALIDATION
  [ ] Create test team directory with manual config + stdin files
  [ ] Run gogent-team-run against test directory
  [ ] Verify all stdout files contain structured JSON with $schema field
  [ ] Verify inter-wave script produces expected artifacts
  [ ] Verify budget tracking is accurate (reservation vs actual)
  [ ] Run full end-to-end via slash command
```

---

## 9. Framework Primitives Reference

### Budget System

| Function | Location | Purpose |
|----------|----------|---------|
| `estimateCost(model)` | config.go | Returns conservative cost estimate per model |
| `tryReserveBudget(member)` | cost.go | Atomic pre-spawn budget reservation |
| `reconcileCost(member, actual)` | cost.go | Post-spawn adjustment (release over-reservation) |
| `budget_max_usd` | config.json | Hard ceiling (team fails if exceeded) |
| `warning_threshold_usd` | config.json | Early warning (logged, not blocking) |

### Process Management

| Component | Purpose |
|-----------|---------|
| Process Registry | Track all child PIDs, enable clean shutdown |
| Session Leader | Each agent gets own process group (`Setsid: true`) |
| SIGTERM → SIGKILL | Graceful shutdown with escalation timeout |
| PID File | Prevent duplicate team-run instances |
| Heartbeat File | Liveness detection for `/team-status` |

### Stdout Extraction (Three-Tier)

| Tier | Method | When Used |
|------|--------|-----------|
| 1 | Direct JSON parse | Agent response is pure JSON |
| 2 | Code block extraction | Agent wraps JSON in ```json ... ``` |
| 3 | Raw text fallback | Agent ignores format instructions — wraps in `{"raw_output": true, "result": "..."}` |

### Envelope Builder

The prompt sent to each agent is constructed by `buildPromptEnvelope()`:

```
AGENT: {agent-id}

{description from stdin}

## Context
Project: {project_root}
Team: {team_dir}

## Task Input
{formatted stdin fields}

## Expected Output Format
Your response MUST be a single JSON code block...

```json
{stdout schema from contract}
```

Include "$schema": "{schema-name}" in your JSON output.
```

---

## 10. Roadmap: From CLI to Platform

### Phase 1: CLI Framework (Current — Validated)
- Single-user, local execution
- File-based coordination
- Manual workflow creation via JSON schemas

### Phase 2: Workflow SDK
- CLI tool to scaffold new workflows: `gogent-workflow init security-audit`
- Schema validation at creation time
- Template library for common patterns (review, analysis, pipeline)

### Phase 3: Team Server
- HTTP API wrapping `gogent-team-run`
- WebSocket for real-time status updates
- Multi-user with per-user budget quotas
- Team directory stored in object storage (S3/GCS)

### Phase 4: Workflow Marketplace
- Publish/discover workflow templates
- Version pinning for reproducibility
- Usage analytics and cost benchmarking
- Community-contributed agent roles and inter-wave scripts

### Phase 5: Enterprise
- SSO and RBAC for workflow access
- Audit logging and compliance reporting
- SLA-based agent selection (latency vs cost vs quality)
- Multi-model routing (use cheapest model that meets quality threshold)

---

## Appendix A: Validated Workflow Reference

### Review Workflow

```
Waves:     1 (parallel)
Agents:    backend-reviewer (haiku), frontend-reviewer (haiku),
           standards-reviewer (haiku), architect-reviewer (sonnet)
Inter-wave: none
Budget:    $2.00
Runtime:   ~40s
Use case:  Code review with domain-specialised reviewers
```

### Braintrust Workflow

```
Waves:     2 (parallel → sequential)
Agents:    Wave 1: einstein (opus) + staff-architect (opus)
           Wave 2: beethoven (opus)
Inter-wave: gogent-team-prepare-synthesis (merges Wave 1 → pre-synthesis.md)
Budget:    $16.00 (actual ~$2.50)
Runtime:   ~6.5 minutes
Use case:  Deep multi-perspective analysis of complex problems
```
