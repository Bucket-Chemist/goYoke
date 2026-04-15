# Bioinformatics Review Team — Architecture

How `/review-bioinformatics` executes end-to-end via `gogent-team-run`.

---

## Overview

The bioinformatics review team is a 2-wave pipeline orchestrated by `gogent-team-run` (a standalone Go binary). Wave 1 runs up to 6 domain-specialist Opus reviewers in parallel. An inter-wave Go script (`gogent-team-prepare-synthesis`) performs programmatic cross-referencing of findings. Wave 2 runs the Staff Bioinformatician, which applies a 7-layer synthesis framework to produce a unified verdict.

**Cost model:** $10-25 per review depending on reviewer count (3-6 reviewers at $2-5 each + synthesizer at $2-5).

---

## Execution Flow

```
/review-bioinformatics SKILL (router)
    │
    ├─ 1. Detect changed files (git diff --staged)
    ├─ 2. Classify domains (grep first 50 lines for indicators)
    ├─ 3. Select reviewers (top 4 by match + always bioinformatician)
    ├─ 4. Generate team_dir/:
    │       config.json
    │       stdin_{reviewer}.json  (per bioinformatics-reviewer.json schema)
    │       stdin_staff-bioinformatician.json
    │
    └─ 5. mcp__gofortress-interactive__team_run({team_dir})
                │
                ▼
        gogent-team-run (Go binary, background process)
                │
                ├─ Reads config.json → TeamRunner
                ├─ Writes PID file + heartbeat
                ├─ Notifies TUI via UDS
                │
                ▼
    ┌───────────────────────────────────────────────────────┐
    │                   WAVE 1 (parallel)                    │
    │                                                        │
    │  Budget check → reserve $5/member → spawn goroutines   │
    │                                                        │
    │  ┌────────────┐ ┌────────────┐ ┌────────────┐         │
    │  │ genomics-  │ │ proteomics-│ │proteogenom-│ ...      │
    │  │ reviewer   │ │ reviewer   │ │ics-reviewer│         │
    │  │            │ │            │ │            │         │
    │  │ claude -p  │ │ claude -p  │ │ claude -p  │         │
    │  │ --model    │ │ --model    │ │ --model    │         │
    │  │   opus     │ │   opus     │ │   opus     │         │
    │  │ < stdin    │ │ < stdin    │ │ < stdin    │         │
    │  │            │ │            │ │            │         │
    │  │ Read/Grep  │ │ Read/Grep  │ │ Read/Grep  │         │
    │  │ actual     │ │ actual     │ │ actual     │         │
    │  │ pipeline   │ │ pipeline   │ │ pipeline   │         │
    │  │ files      │ │ files      │ │ files      │         │
    │  │            │ │            │ │            │         │
    │  │  ┌──────┐  │ │  ┌──────┐  │ │  ┌──────┐  │         │
    │  │  │stdout│  │ │  │stdout│  │ │  │stdout│  │         │
    │  │  │ JSON │  │ │  │ JSON │  │ │  │ JSON │  │         │
    │  │  │+sharp│  │ │  │+sharp│  │ │  │+sharp│  │         │
    │  │  │edge  │  │ │  │edge  │  │ │  │edge  │  │         │
    │  │  │_ids  │  │ │  │_ids  │  │ │  │_ids  │  │         │
    │  │  └──┬───┘  │ │  └──┬───┘  │ │  └──┬───┘  │         │
    │  └─────┼──────┘ └─────┼──────┘ └─────┼──────┘         │
    │        ▼              ▼              ▼                  │
    │  stdout_genomics  stdout_proteo  stdout_proteogen  ... │
    │                                                        │
    │  sync.WaitGroup.Wait() — all must complete/fail/timeout│
    │                                                        │
    │  Partial fail (3/4 ok): continue to wave 2             │
    │  Total fail (0/N ok):   skip wave 2, abort             │
    └───────────────────────┬────────────────────────────────┘
                            │
                            ▼
    ┌───────────────────────────────────────────────────────┐
    │        INTER-WAVE: gogent-team-prepare-synthesis       │
    │                    (Go binary, NOT an LLM)             │
    │                                                        │
    │  1. Read all stdout_*.json from wave 1                 │
    │  2. Extract findings (JSON arrays with sharp_edge_ids) │
    │  3. Load interaction-rules.json from ~/.claude/         │
    │  4. DetectInteractions():                              │
    │                                                        │
    │     For each rule in Boundary Interaction Matrix:       │
    │       upstream_id:   massspec-spectral-centroiding      │
    │       downstream_id: proteoform-deconv-charge-cascade   │
    │       → Scan findings: upstream flagged?  YES/NO        │
    │       → Scan findings: downstream flagged? YES/NO       │
    │       → BOTH flagged → interaction DETECTED             │
    │         → severity escalation per algebra type          │
    │           (GATING / MULTIPLICATIVE / ADDITIVE / NEGATING)│
    │                                                        │
    │  5. Write pre-synthesis.md:                            │
    │     - Reviewer summary (who ran, finding counts)       │
    │     - Detected interactions with escalation notes      │
    │     - Dedup candidates (same file+line, 2+ agents)     │
    │                                                        │
    │  6. Write detected-interactions.json (sidecar)         │
    │  7. Update stdin_staff-bioinformatician.json:           │
    │     - wave_0_outputs: [{reviewer_id, stdout_path, status}]│
    │     - wave0_findings_path → pre-synthesis.md           │
    │     - detected_interactions_path → interactions.json    │
    └───────────────────────┬────────────────────────────────┘
                            │
                            ▼
    ┌───────────────────────────────────────────────────────┐
    │                   WAVE 2 (single agent)                │
    │                                                        │
    │  ┌──────────────────────────────────────────────────┐  │
    │  │        Staff Bioinformatician (Opus)              │  │
    │  │                                                   │  │
    │  │  READS:                                           │  │
    │  │    stdin_staff-bioinformatician.json               │  │
    │  │    pre-synthesis.md (programmatic pre-analysis)    │  │
    │  │    stdout_genomics-reviewer.json                   │  │
    │  │    stdout_proteomics-reviewer.json                 │  │
    │  │    stdout_proteogenomics-reviewer.json             │  │
    │  │    stdout_bioinformatician-reviewer.json           │  │
    │  │    (+ any other completed wave 1 stdout files)     │  │
    │  │                                                   │  │
    │  │  EXECUTES 7-LAYER FRAMEWORK:                      │  │
    │  │    L1: Information Integrity Chain                 │  │
    │  │        (trace data across reviewer boundaries)    │  │
    │  │    L2: Version & Reference Coherence               │  │
    │  │        (version matrix: all stages same build?)   │  │
    │  │    L3: Statistical Coherence                       │  │
    │  │        (FDR chain end-to-end, DB size x FDR)      │  │
    │  │    L4: Cross-Domain Finding Synthesis              │  │
    │  │        (dedup, causal chains, severity reclass)    │  │
    │  │    L5: Contradiction Resolution                    │  │
    │  │        (when reviewers disagree, who's right?)     │  │
    │  │    L6: Methodology Assessment                      │  │
    │  │        (is the overall approach scientifically     │  │
    │  │         appropriate for the study goals?)          │  │
    │  │    L7: Coverage, Completeness & Verdict            │  │
    │  │        (who didn't run? what stages uncovered?)    │  │
    │  │                                                   │  │
    │  │  Uses pre-synthesis.md interactions as INPUT:      │  │
    │  │    Programmatic detections confirmed/dismissed     │  │
    │  │    by Opus reasoning against actual findings       │  │
    │  │                                                   │  │
    │  │  OUTPUTS: stdout JSON                             │  │
    │  │    unified_verdict: BLOCK / WARNING / APPROVE      │  │
    │  │    7 layer assessments (PASS/CONCERN/FAIL)         │  │
    │  │    report_markdown (full human-readable report)    │  │
    │  │    causal_chains (with sharp_edge_id waypoints)    │  │
    │  │    methodology_assessment                          │  │
    │  │    reviewer_summary                                │  │
    │  └──────────────────────────────────────────────────┘  │
    │                                                        │
    │  → stdout_staff-bioinformatician.json                  │
    └───────────────────────┬────────────────────────────────┘
                            │
                            ▼
    ┌───────────────────────────────────────────────────────┐
    │                     COMPLETION                         │
    │                                                        │
    │  gogent-team-run:                                      │
    │    - status="completed" in config.json                 │
    │    - Releases PID file                                 │
    │    - Notifies TUI via UDS (toast)                      │
    │    - Exits                                             │
    │                                                        │
    │  User retrieves results:                               │
    │    /team-status  → progress summary                    │
    │    /team-result  → renders report_markdown             │
    └───────────────────────────────────────────────────────┘
```

---

## Team Config Structure

Source template: `.claude/schemas/teams/review-bioinformatics.json`

```
review-bioinformatics.json
├── budget_max_usd: 30.0
├── waves:
│   ├── wave 1: "Parallel multi-domain review"
│   │   ├── genomics-reviewer       (opus, 15min timeout)
│   │   ├── proteomics-reviewer     (opus, 15min timeout)
│   │   ├── proteogenomics-reviewer (opus, 15min timeout)
│   │   ├── proteoform-reviewer     (opus, 15min timeout)
│   │   ├── mass-spec-reviewer      (opus, 15min timeout)
│   │   └── bioinformatician-reviewer (opus, 15min timeout, always-run)
│   │   on_complete_script: "gogent-team-prepare-synthesis"
│   │
│   └── wave 2: "7-layer cross-domain synthesis"
│       └── staff-bioinformatician  (opus, 15min timeout)
│           on_complete_script: null
```

The skill selects 2-4 domain reviewers based on detected file content. The bioinformatician-reviewer always runs. Unselected reviewers are removed from the config before launch.

---

## Stdin/Stdout Schemas

| Schema | Purpose | Used By |
|--------|---------|---------|
| `bioinformatics-reviewer.json` | Validates wave 1 reviewer stdin: agent ID, review scope, files, git context, omics domain, focus areas | All 6 domain reviewers |
| `review-bioinformatics-staff-bioinformatician.json` | Validates wave 2 synthesizer stdin: wave_0_outputs array, review scope summary, omics context | Staff Bioinformatician |

### Wave 1 stdin (per reviewer)
```json
{
  "agent": "genomics-reviewer",
  "workflow": "review-bioinformatics",
  "context": { "project_root": "/abs/path", "team_dir": "/abs/path" },
  "review_scope": {
    "files": [{ "path": "pipeline/align.nf", "language": "nextflow", ... }],
    "total_files": 5,
    "languages_detected": ["nextflow", "python"]
  },
  "git_context": { "commit_message": "...", "branch_name": "..." },
  "focus_areas": { "alignment": true, "variant_calling": true, ... },
  "omics_context": {
    "primary_domain": "genomics",
    "data_formats_detected": ["FASTQ", "BAM", "VCF"],
    "pipeline_tools_detected": ["BWA", "GATK", "Nextflow"],
    "workflow_manager": "nextflow"
  }
}
```

### Wave 2 stdin (synthesizer, after inter-wave script updates it)
```json
{
  "agent": "staff-bioinformatician",
  "workflow": "review-bioinformatics",
  "wave_0_outputs": [
    { "reviewer_id": "genomics-reviewer", "stdout_file_path": "/abs/stdout_genomics.json", "status": "completed" },
    { "reviewer_id": "proteomics-reviewer", "stdout_file_path": "/abs/stdout_proteomics.json", "status": "completed" },
    { "reviewer_id": "bioinformatician-reviewer", "stdout_file_path": "/abs/stdout_bioinfo.json", "status": "completed" }
  ],
  "wave0_findings_path": "/abs/pre-synthesis.md",
  "detected_interactions_path": "/abs/detected-interactions.json"
}
```

---

## Inter-Wave Script: `gogent-team-prepare-synthesis`

This Go binary is the critical bridge between wave 1 and wave 2. It runs deterministic code (not an LLM) to pre-analyze findings before the Staff Bioinformatician sees them.

### What it does

1. **Extracts** all JSON findings from each wave 1 reviewer's stdout file
2. **Loads** interaction rules from `~/.claude/` (the Boundary Interaction Matrix encoded as machine-readable rules)
3. **Detects** cross-domain interactions programmatically by matching `sharp_edge_id` pairs against the rules
4. **Writes** `pre-synthesis.md` — a structured summary that the Staff Bioinformatician reads before starting Layer 1

### Why concrete sharp_edge_ids matter

The interaction detection is ID-based string matching:

```
Rule: { upstream: "massspec-spectral-centroiding", downstream: "proteoform-deconv-charge-cascade", type: "gating" }

Finding from mass-spec-reviewer:   { sharp_edge_id: "massspec-spectral-centroiding", severity: "critical" }
Finding from proteoform-reviewer:  { sharp_edge_id: "proteoform-deconv-charge-cascade", severity: "critical" }

→ BOTH present → Interaction DETECTED → type: gating → escalation note in pre-synthesis.md
```

This is why the expansion replaced vague strings like `"mass-spec: centroiding quality"` with concrete IDs — the programmatic detection cannot match free-text descriptions, only exact ID strings.

---

## Budget & Cost Management

| Component | Mechanism |
|-----------|-----------|
| **Reserve** | Before spawning each member, `tryReserveBudget(estimated)` deducts the estimate |
| **Reconcile** | After member completes, actual cost extracted from Claude CLI output; surplus returned to pool |
| **Gate** | If remaining budget < estimated cost, member spawn is blocked (not killed) |
| **Warning** | When remaining budget crosses below `warning_threshold_usd`, a toast fires to the TUI |
| **Ceiling** | `budget_max_usd: 30.0` is the total pool for all waves combined |

Typical costs per member:
- Domain reviewer (Opus, 15min): $2-5 depending on file count and checklist depth
- Staff Bioinformatician (Opus, 15min): $2-5 depending on number of wave 1 findings

---

## Health Monitoring

`gogent-team-run` monitors each spawned agent via its NDJSON output stream:

| Threshold | Status | Action |
|-----------|--------|--------|
| 30s between outputs | normal | Continue monitoring |
| 90s (stallWarningThreshold) | stall_warning | Logged, TUI activity shows yellow |
| 150s (3x warning) | stalled | Logged, counter incremented — NOT killed (Opus thinking can be legitimately silent) |
| timeout_ms reached | timeout | SIGTERM → 5s grace → SIGKILL. Member marked status="timeout" |

Heartbeat file updated every 30s with timestamp — external monitors (systemd, cron) can detect a hung `gogent-team-run` process.

---

## Failure Modes & Recovery

| Scenario | Behavior | User Impact |
|----------|----------|-------------|
| 1 reviewer times out | Wave 1 partial success → wave 2 proceeds | Staff-bio notes missing reviewer at Layer 7; coverage gap flagged |
| All reviewers fail | Wave 1 total failure → wave 2 skipped | Team status shows failure; no synthesis produced |
| Staff-bio times out | Wave 2 fails | Individual reviewer stdout files still available via `/team-result` |
| Budget exhausted mid-wave | Remaining members not spawned | Partial results from already-completed members available |
| `gogent-team-run` crashes | PID file left behind; heartbeat stale | `/team-status` detects stale heartbeat; manual cleanup needed |
| Inter-wave script fails | Wave 2 proceeds with un-enriched stdin | Staff-bio gets wave_0_outputs but no pre-synthesis.md; Layer 1 less informed |

---

## File Layout (team directory)

```
{session_dir}/teams/{timestamp}.bioinformatics-review/
├── config.json                              ← team config (live-updated with status/cost)
├── stdin_genomics-reviewer.json             ← wave 1 reviewer input
├── stdin_proteomics-reviewer.json
├── stdin_proteogenomics-reviewer.json
├── stdin_bioinformatician-reviewer.json
├── stdin_staff-bioinformatician.json        ← wave 2 synthesizer input (updated by inter-wave)
├── stdout_genomics-reviewer.json            ← wave 1 reviewer output (JSON findings)
├── stdout_proteomics-reviewer.json
├── stdout_proteogenomics-reviewer.json
├── stdout_bioinformatician-reviewer.json
├── stdout_staff-bioinformatician.json       ← FINAL OUTPUT (7-layer report + verdict)
├── stream_genomics-reviewer.ndjson          ← raw NDJSON stream (for debugging/replay)
├── stream_proteomics-reviewer.ndjson
├── ...
├── pre-synthesis.md                         ← inter-wave script output (programmatic analysis)
├── detected-interactions.json               ← inter-wave script output (machine-readable)
├── runner.log                               ← gogent-team-run execution log
├── heartbeat                                ← timestamp file for external health monitoring
└── gogent-team-run.pid                      ← PID file (removed on clean exit)
```

---

## Agent Team Composition

### Wave 1: Domain Reviewers (all Opus-tier, parallel)

| Agent | Domain | Checklist Items | Sharp Edge IDs | Always Run? |
|-------|--------|:-:|:-:|:-:|
| genomics-reviewer | Alignment, variant calling, annotation | 30 | 20 | No |
| proteomics-reviewer | Search engine, FDR, quantification | 41 | 14 | No |
| proteogenomics-reviewer | Custom DB, novel peptides, variant peptides | 61 | 41 | No |
| proteoform-reviewer | Deconvolution, PTM, intact mass | 31 | 15 | No |
| mass-spec-reviewer | Instrumentation, acquisition, calibration | 32 | 17 | No |
| bioinformatician-reviewer | Pipeline architecture, reproducibility, stats | 30 | 20 | **Yes** |

### Wave 2: Synthesizer (Opus-tier, sequential)

| Agent | Role | Framework | Output |
|-------|------|-----------|--------|
| staff-bioinformatician | Cross-domain synthesis | 7-layer review | BLOCK / WARNING / APPROVE verdict |

### Cross-Domain Wiring

The staff-bioinformatician's 7-layer framework connects reviewers via:

- **38 Boundary Interaction Matrix entries** — sharp_edge_id pairs at reviewer boundaries
- **11 Causal Chain Library entries** — multi-step failure cascades across 2-3 reviewers
- **FDR Detection Matrix** — DB inflation x FDR coupling patterns
- **Coverage Matrix** — which pipeline stages are checked by which reviewers

All wiring uses concrete `sharp_edge_id` values (not free-text descriptions), enabling programmatic detection by `gogent-team-prepare-synthesis`.

---

## Related Files

| File | Purpose |
|------|---------|
| `.claude/skills/review-bioinformatics/SKILL.md` | Skill definition (invocation, detection, reviewer selection) |
| `.claude/schemas/teams/review-bioinformatics.json` | Team config template |
| `.claude/schemas/stdin/bioinformatics-reviewer.json` | Wave 1 stdin schema |
| `.claude/schemas/stdin/review-bioinformatics-staff-bioinformatician.json` | Wave 2 stdin schema |
| `.claude/agents/teams/bioinformatics/sharp-edge-conventions.md` | ID naming conventions (append-only) |
| `.claude/agents/teams/bioinformatics/integration-report.md` | Cross-agent integration documentation |
| `cmd/gogent-team-run/` | Team runner Go source |
| `cmd/gogent-team-prepare-synthesis/` | Inter-wave script Go source |
