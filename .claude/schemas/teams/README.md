# Team Configuration Templates

This directory contains template files for multi-agent team coordination workflows in GOgent-Fortress.

## Purpose

Team templates define the structure and execution parameters for coordinated multi-agent workflows. Each template specifies:
- Agent roles and models
- Wave-based execution order (dependencies)
- Budget constraints
- Input/output file contracts
- Retry and timeout policies

## Current Templates

### `braintrust.json`
**Workflow**: Parallel deep analysis + synthesis
**Pattern**: 2 waves — Wave 1 (einstein + staff-architect parallel analysis), Wave 2 (beethoven synthesis)
**Budget**: $5.00
**Use Case**: Complex architectural decisions, intractable design problems

### `review.json`
**Workflow**: Parallel multi-domain code review
**Pattern**: 1 wave — 4 reviewers (backend, frontend, standards, architect) execute in parallel
**Budget**: $2.00
**Use Case**: Comprehensive code review with domain-specific perspectives

### `implementation.json`
**Workflow**: Task DAG-driven implementation with wave-based dependency resolution
**Pattern**: N waves — tasks grouped by dependency order, workers execute in parallel within each wave
**Budget**: $10.00
**Use Case**: Multi-file implementation from specs.md or ticket task DAGs

## Stdin/Stdout Schemas

The `stdin-stdout/` directory contains comprehensive JSON contracts for each agent type. These define exactly what input each agent receives and what structured output it produces.

### Braintrust Agents

| Agent | Schema | Input | Output |
|-------|--------|-------|--------|
| Einstein | `braintrust-einstein.json` | Problem brief, analysis axes, codebase context, scout findings | Root cause analysis, conceptual frameworks, novel approaches, assumptions, open questions |
| Staff-Architect | `braintrust-staff-architect.json` | Plan to review, 7-layer focus areas, codebase context, scout metrics | Issue register, assumption register, dependency analysis, failure modes, verdict |
| Beethoven | `braintrust-beethoven.json` | Problem brief, Einstein's full output, Staff-Architect's full output | Convergence/divergence resolution, unified recommendations, risk assessment, implementation pathway |

### Review Agents

| Agent | Schema | Input | Output |
|-------|--------|-------|--------|
| Backend | `review-backend.json` | Backend files with categories, security/API/data/concurrency focus areas | Findings by severity, security summary, sharp_edge_id correlation |
| Frontend | `review-frontend.json` | Frontend files with component tree, a11y/hooks/perf/memory focus areas | Findings by severity, accessibility summary |
| Standards | `review-standards.json` | Source files, naming/complexity/DRY/structure focus areas | Findings by severity, complexity score |
| Architect | `review-architect.json` | Changed files + direct imports, coupling/boundary/testability focus | Findings by severity, structural health score (A-F) |

All reviewers share a common findings structure: `severity`, `category`, `file`, `line`, `message`, `recommendation`, `sharp_edge_id`.

### Implementation Agents

| Agent | Schema | Input | Output |
|-------|--------|-------|--------|
| Worker | `implementation-worker.json` | Task description, acceptance criteria, conventions, codebase context | Files modified, tests written, acceptance criteria status, build status |

## Usage

Templates are instantiated by the router when spawning a team:

1. Router selects template based on workflow type
2. Template is copied to `sessions/{YYMMDD}.{id}/teams/{timestamp}.{workflow}/config.json`
3. Placeholders (`{timestamp}`, `{uuid}`, etc.) are filled
4. Orchestrator (Mozart/review-orchestrator) populates stdin files from codebase context
5. `gogent-team-run` reads config, spawns agents with stdin, collects stdout
6. Go binary writes final output files from agent stdout JSON

## Field Contract

See `common-types.md` for complete field definitions, types, and validation rules.

## Critical Design Decisions

**Budget fields**: Top-level flat (`budget_max_usd`, `budget_remaining_usd`), NOT nested.

**Tool permissions**: Agents in team pattern use `Read`, `Glob`, `Grep` only. No `Write`/`Edit` needed — agents output JSON to stdout, Go binary handles file writes. See `tickets/team-coordination/PERMISSION-DESIGN.md`.

**Two-layer parsing**: CLI output is a JSON array (Layer 1: cost extraction). Agent response is inside `result.Result` string (Layer 2: agent-specific stdout schema). See `cmd/gogent-team-run/docs/claude-cli-output-format.md`.

## Related Tickets

- **TC-008**: Core orchestration (reads these templates, implements Go structs)
- **TC-005**: CLI output format (Layer 1 parsing spec)
- **TC-001**: Permission design (allowed tools per agent)
- **TC-002**: Concurrency design (mutex-protected config updates)
- **TC-014**: cli_flags in agents-index.json (per-agent tool lists)
