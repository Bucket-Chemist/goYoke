# Specification: /review-bioinformatics Skill + 6 Opus Bioinformatics Reviewer Agents

## Context

- **Goal:** Create a bioinformatics-domain code review system mirroring the existing `/review` skill architecture. 6 Opus-tier specialist reviewers cover genomics, proteomics, proteogenomics, proteoforms, mass spectrometry, and general bioinformatics pipeline review.
- **Scope:** 6 agent frontmatter files, 1 skill, 1 team config template, 1 stdin schema, wiring across agents-index.json, routing-schema.json, and CLAUDE.md. Total: ~54 integration points.
- **Constraints:** Opus tier ($0.045/1K tokens), max 4 reviewers per invocation, bioinformatician-reviewer always included.

---

## Decisions

| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| **No orchestrator agent** | Skill handles orchestration via team-run dispatch (same as /review). An orchestrator would create a tier inversion (Sonnet orchestrating Opus reviewers) and add 9 integration points with no functional benefit. Bioinformatics review is deliberately invoked via slash command, not conversational. | `bioinformatics-review-orchestrator` (Sonnet) that spawns reviewers via spawn_agent. Rejected: tier inversion anti-pattern, doubles implementation scope, team-run already handles parallel dispatch. |
| **Opus tier for all 6 reviewers** | Bioinformatics review requires deep domain expertise, nuanced judgment about statistical methodology, and cross-domain reasoning (e.g., proteogenomics spans genomics + proteomics). Sonnet lacks depth for critique of FDR control methodology or variant calling pipeline design. | Sonnet tier with elevated thinking budget. Rejected: domain review quality is the primary value — cost savings not worth quality loss for a review workflow. |
| **Max 4 reviewers per invocation** | 6 Opus reviewers running simultaneously costs $15-30. Intelligent selection based on file content keeps cost at $5-20 while covering relevant domains. | Always run all 6 (comprehensive but expensive at $15-30 minimum). Rejected: most codebases touch 1-2 omics domains per review scope. |
| **bioinformatician-reviewer always included** | Pipeline architecture, reproducibility, and statistical methodology apply to ALL bioinformatics code regardless of omics domain. Mirrors standards-reviewer in /review (always runs). | Only when pipeline files detected. Rejected: even pure analysis scripts need reproducibility and statistics review. |
| **Read/Glob/Grep tools only** | Reviewers analyze code, they don't modify it. Read-only tool set prevents accidental writes and reduces cost (no Edit/Write token overhead). | Include Write for generating fix patches. Rejected: reviewers recommend, they don't implement. |
| **cost_ceiling: $5.00 per reviewer** | Opus with 32K thinking budget on a large pipeline codebase can consume significant tokens. $5 cap prevents runaway costs while allowing thorough review. | $2.50 (tighter). Rejected: Opus review of complex statistical methodology may legitimately need 50-60K output tokens. |
| **timeout_ms: 900000 (15 min)** | Opus reviewers need time for deep analysis — reading files, cross-referencing patterns, generating detailed findings. 15 min is generous but prevents hangs. | 600000 (10 min, matching /review). Rejected: Opus is slower than Sonnet, and bioinformatics files are often large config/pipeline files. |
| **max_retries: 1 (not 2)** | At $2.50-$5.00 per Opus invocation, a retry costs $5-10. Two retries would be $10-20 for a single reviewer. One retry balances reliability vs cost. | max_retries: 2 (matching /review). Rejected: /review uses Sonnet at ~$0.50-$1.00 per retry. Opus retries are 5x more expensive. |
| **Leaf agents (no spawned_by/can_spawn)** | These reviewers are spawned by team-run (CLI subprocess), not by spawn_agent. Team-run doesn't use spawned_by/can_spawn validation. Adding these fields would create misleading documentation. | spawned_by: ["router"]. Rejected: router doesn't spawn them directly — the skill launches team-run which spawns them. |
| **New stdin schema (not reuse reviewer.json)** | Bioinformatics review needs `omics_context` (domain, data formats, pipeline tools, workflow manager) and `pipeline_context` (containers, environment, organisms) — none of which exist in reviewer.json. Extending reviewer.json would break existing validators. | Extend reviewer.json with optional fields. Rejected: additionalProperties: false in reviewer.json means new fields would fail validation for existing reviewers. |

---

## Integration Points Inventory

### Per Agent (9 points × 6 agents = 54 total)

| # | Integration Point | Location | Notes |
|---|-------------------|----------|-------|
| 1 | Agent frontmatter file | `.claude/agents/{id}/{id}.md` | Complete frontmatter + body sections |
| 2 | agents-index.json entry | `.claude/agents/agents-index.json` agents[] | Full agent object with all fields |
| 3 | agents-index.json relationships | Same file | No spawned_by/can_spawn for leaf reviewers |
| 4 | agents-index.json model_tiers | Same file, `routing_rules.model_tiers.opus` | Add to opus array |
| 5 | routing-schema.json agent_subagent_mapping | `.claude/routing-schema.json` ~line 556 | Map id → subagent_type string |
| 6 | routing-schema.json tiers.opus.agents | Same file ~line 210 | Add to agents array |
| 7 | routing-schema.json task_invocation_allowlist | Same file ~line 194 | Add to allowlist for direct spawning |
| 8 | routing-schema.json subagent_types | Same file ~line 433 | New bioinformatics_review category |
| 9 | CLAUDE.md dispatch table | `.claude/CLAUDE.md` | Tier 3 subsection |

### Additional (non-per-agent)

| # | Integration Point | Location |
|---|-------------------|----------|
| 10 | Skill SKILL.md | `.claude/skills/review-bioinformatics/SKILL.md` |
| 11 | Team config template | `.claude/schemas/teams/review-bioinformatics.json` |
| 12 | Stdin schema | `.claude/schemas/stdin/bioinformatics-reviewer.json` |
| 13 | CLAUDE.md slash command | `.claude/CLAUDE.md` Slash Commands table |
| 14 | Sharp edges files (×6) | `.claude/agents/{id}/sharp-edges.yaml` |

**Total integration points: 54 (per-agent) + 5 (shared) = 59**

---

## Per-Agent Frontmatter Specification

### Shared Fields (all 6 agents)

```yaml
model: opus
effort: high
thinking:
  enabled: true
  budget: 32000
tier: 3
category: bioinformatics-review
tools:
  - Read
  - Glob
  - Grep
failure_tracking:
  max_attempts: 2
  on_max_reached: "report_incomplete"
cost_ceiling: 5.00
```

### Agent-Specific Fields

#### 1. genomics-reviewer

| Field | Value |
|-------|-------|
| id | `genomics-reviewer` |
| name | Genomics Reviewer |
| subagent_type | Genomics Reviewer |
| description | Genome assembly, variant calling, alignment, and sequence data format review. BWA/Bowtie2/STAR, GATK/bcftools, VCF/BAM/FASTA/FASTQ/GFF/GTF/BED. |
| triggers | review genomics, alignment review, variant calling review, genome assembly review, VCF review, sequencing review |
| conventions_required | python.md, R.md |
| focus_areas | Alignment accuracy, Variant calling methodology, Reference genome handling, File format compliance, Annotation pipeline correctness |

#### 2. proteomics-reviewer

| Field | Value |
|-------|-------|
| id | `proteomics-reviewer` |
| name | Proteomics Reviewer |
| subagent_type | Proteomics Reviewer |
| description | Mass spec proteomics data processing review. Search engine config, FDR control, quantification (TMT/iTRAQ/LFQ/SILAC), mzML/mzXML, PSM scoring. |
| triggers | review proteomics, protein identification review, quantification review, FDR review, search engine review |
| conventions_required | python.md, R.md |
| focus_areas | Search engine parameters, FDR control methodology, Quantification design, Statistical testing |

#### 3. proteogenomics-reviewer

| Field | Value |
|-------|-------|
| id | `proteogenomics-reviewer` |
| name | Proteogenomics Reviewer |
| subagent_type | Proteogenomics Reviewer |
| description | Proteogenomics pipeline review. Custom protein DB construction, novel peptide ID, variant peptides, splice junction peptides, ORF prediction. |
| triggers | review proteogenomics, custom database review, novel peptide review, variant peptide review |
| conventions_required | python.md, R.md |
| focus_areas | Database construction methodology, Novel peptide validation, Variant peptide identification, Splice junction detection, ORF prediction quality |

#### 4. proteoform-reviewer

| Field | Value |
|-------|-------|
| id | `proteoform-reviewer` |
| name | Proteoform Reviewer |
| subagent_type | Proteoform Reviewer |
| description | Top-down proteomics and proteoform analysis review. Intact mass analysis, PTM combinatorics, proteoform families, deconvolution, sequence coverage. |
| triggers | review proteoform, top-down review, PTM analysis review, intact mass review, deconvolution review |
| conventions_required | python.md, R.md |
| focus_areas | Deconvolution algorithm selection, PTM localization confidence, Proteoform family assignment, Intact mass accuracy, Sequence coverage |

#### 5. mass-spec-reviewer

| Field | Value |
|-------|-------|
| id | `mass-spec-reviewer` |
| name | Mass Spectrometry Reviewer |
| subagent_type | Mass Spectrometry Reviewer |
| description | MS instrumentation and data acquisition review. DDA/DIA/PRM methods, calibration, raw data quality, vendor formats (Thermo/Bruker/SCIEX/Waters), spectral processing. |
| triggers | review mass spec, instrument review, acquisition review, raw data quality review, DIA review |
| conventions_required | python.md |
| focus_areas | Acquisition method suitability, Instrument parameter optimization, Calibration and QC, Vendor-specific data handling, Spectral processing |

#### 6. bioinformatician-reviewer

| Field | Value |
|-------|-------|
| id | `bioinformatician-reviewer` |
| name | Bioinformatician Reviewer |
| subagent_type | Bioinformatician Reviewer |
| description | Pipeline architecture and methodology review. Nextflow/Snakemake/WDL workflows, Docker/Singularity/Conda reproducibility, statistics, data provenance. |
| triggers | review bioinformatics, pipeline review, workflow review, reproducibility review, statistical methods review |
| conventions_required | python.md, R.md |
| focus_areas | Workflow reproducibility, Pipeline architecture, Statistical methodology, Resource management, Data provenance |

---

## Skill Specification

### /review-bioinformatics

**Location:** `.claude/skills/review-bioinformatics/SKILL.md`

**Mirrors:** `/review` skill (`.claude/skills/review/SKILL.md`) with bioinformatics adaptations.

#### Invocation

```
/review-bioinformatics           # Staged changes (bioinformatics files)
/review-bioinformatics --all     # All uncommitted changes
/review-bioinformatics --scope=<glob>  # Specific files
/review-bioinformatics path/     # Specific directory
```

#### File Classification

| Extension | Language | Category |
|-----------|----------|----------|
| `.nf` | nextflow | pipeline |
| `.smk` | snakemake | pipeline |
| `.wdl` | wdl | pipeline |
| `.cwl` | cwl | pipeline |
| `.py` | python | data-processing |
| `.R` | r | statistical-analysis |
| `.config` | config | config |
| `.yaml`/`.yml` | yaml | config |
| `.sh` | bash | pipeline |

#### Reviewer Selection Algorithm

1. Scan files for domain indicators (imports, tool references, data format mentions)
2. Score each reviewer by number of indicator matches
3. Always include `bioinformatician-reviewer`
4. Include top-scoring domain reviewers up to max 4 total
5. Minimum 2 reviewers per invocation

#### Cost Model

| Component | Model | Est. Tokens | Cost |
|-----------|-------|-------------|------|
| Detection + Classification | Bash | 0 | $0.00 |
| Config generation | Router | ~2K | $0.00 |
| Per Opus Reviewer | Opus | 30-60K | $2.50-$5.00 |
| **Typical (3 reviewers)** | | 90-180K | **$7.50-$15.00** |
| **Maximum (4 reviewers)** | | 120-240K | **$10.00-$20.00** |
| Budget cap | | | **$25.00** |

---

## Schema Specifications

### Team Config Template

**Location:** `.claude/schemas/teams/review-bioinformatics.json`

Mirrors `review.json` with:
- `team_name`: `"review-bioinformatics-{timestamp}"`
- `workflow_type`: `"review-bioinformatics"`
- `budget_max_usd`: 25.0
- `warning_threshold_usd`: 20.0
- 6 members, all `model: "opus"`, `timeout_ms: 900000`, `max_retries: 1`
- Team directory: `{gogent_session_dir}/teams/{timestamp}.bioinformatics-review/`

### Stdin Schema

**Location:** `.claude/schemas/stdin/bioinformatics-reviewer.json`

Extends `reviewer.json` with:
- `omics_context` (required): `primary_domain`, `data_formats_detected`, `pipeline_tools_detected`, `workflow_manager`
- `pipeline_context` (optional): `container_technology`, `execution_environment`, `sample_count`, `organisms`
- `workflow`: const `"review-bioinformatics"`
- `agent`: enum of 6 bioinformatics reviewer IDs

---

## Implementation Phases

### Phase 1: Agent Foundations (Parallel)

- **Tasks:** task-001, task-002, task-003
- **Files:** 6 agent .md files, 6 sharp-edges.yaml, 1 team config, 1 stdin schema
- **Dependencies:** None
- **Risk:** Low — boilerplate creation, well-defined patterns
- **Validation:** Files exist, YAML frontmatter parses, JSON validates with jq
- **Rollback:** Delete created directories and files

### Phase 2: Skill + Wiring (Parallel after Phase 1)

- **Tasks:** task-004, task-005, task-006, task-007
- **Files:** SKILL.md, agents-index.json (edits), routing-schema.json (edits), CLAUDE.md (edits)
- **Dependencies:** task-001 (need agent IDs), task-002/003 (skill references schemas)
- **Risk:** Medium — editing shared JSON files (agents-index, routing-schema) risks breaking existing agent routing if edits corrupt JSON structure
- **Validation:** jq validation on both JSON files, grep for all 6 agent IDs in each file
- **Rollback:** `git checkout .claude/agents/agents-index.json .claude/routing-schema.json .claude/CLAUDE.md`

### Phase 3: Validation Gate (Sequential after Phase 2)

- **Tasks:** task-008
- **Files:** All files (read-only verification)
- **Dependencies:** All Phase 1 + Phase 2 tasks
- **Risk:** None — read-only validation
- **Validation:** 20-point checklist (see task-008 description)
- **Rollback:** N/A (validation doesn't modify files)

---

## Dependency Graph

```
task-001 (agent files)     task-002 (team config)     task-003 (stdin schema)
    │                           │                           │
    ├──────────────────────────┼───────────────────────────┤
    │                           │                           │
    v                           v                           v
task-005 (agents-index)    task-004 (SKILL.md)         task-006 (routing-schema)
task-007 (CLAUDE.md)            │                           │
    │                           │                           │
    └──────────────────────────┼───────────────────────────┘
                                │
                                v
                        task-008 (validation gate)
```

**Parallelizable groups:**
- Wave 1: task-001, task-002, task-003 (all independent)
- Wave 2: task-004, task-005, task-006, task-007 (all depend on task-001; task-004 also on 002, 003)
- Wave 3: task-008 (depends on all of Wave 2)

---

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| JSON corruption in agents-index.json | Medium | High — breaks ALL agent routing | jq validation after every edit; git checkout for rollback |
| JSON corruption in routing-schema.json | Medium | High — breaks routing enforcement | jq validation after every edit; git checkout for rollback |
| subagent_type mismatch between frontmatter and routing-schema | Medium | Medium — agent fails to spawn with cryptic error | Validation gate checks exact string match across all 3 locations |
| Dead fields in frontmatter | Low | Low — no functional impact but violates constraints | Grep-based scan in validation gate |
| Opus reviewers too expensive for typical use | Low | Medium — users avoid the skill | Max 4 reviewers, intelligent selection, budget cap at $25 |
| File classification misses bioinformatics files | Medium | Medium — relevant files excluded from review | Broad extension list + import-based heuristic detection |
| Team-run timeout for Opus reviewers | Low | Medium — review hangs, user waits | 15-minute timeout with graceful failure reporting |
| Reviewer generates hallucinated findings | Medium | High — false positives undermine trust | CRITICAL: File Reading Required section in every agent; Opus reduces hallucination risk vs Haiku/Sonnet |
| CLAUDE.md edit breaks existing dispatch table | Low | High — routing for all agents affected | Targeted Edit insertions, not full-file rewrites |

---

## Bidirectional Spawn Relationships

These reviewers are **leaf agents** spawned by `gogent-team-run` (CLI subprocess), not by `spawn_agent`. Therefore:

- **No `spawned_by` field** in agents-index.json entries
- **No `can_spawn` field** in agents-index.json entries
- **No existing agent needs `can_spawn` updates** for these reviewers

The team-run binary reads agent ID from config.json, looks up the agent in agents-index.json, builds context via `buildFullAgentContext()`, and spawns `claude -p`. This bypasses spawn_agent validation entirely.

If direct spawn_agent invocation is needed later (e.g., from a future bioinformatics-review-orchestrator), add:
- `spawned_by: ["bioinformatics-review-orchestrator"]` to each reviewer
- `can_spawn: [all 6 reviewer IDs]` to the orchestrator

---

## Success Criteria

- [ ] All 6 agent frontmatter files created with valid YAML and complete body sections
- [ ] Team config template validates with jq and has 6 Opus members
- [ ] Stdin schema is valid JSON Schema with omics_context and pipeline_context fields
- [ ] Skill SKILL.md follows /review structure with bioinformatics adaptations
- [ ] agents-index.json has 6 new entries, parses with jq, existing entries intact
- [ ] routing-schema.json updated in all 4 locations, parses with jq, existing entries intact
- [ ] CLAUDE.md has dispatch table subsection and slash command entry
- [ ] Validation gate passes all 20 checks with zero failures
- [ ] No dead frontmatter fields anywhere
- [ ] Total estimated cost per typical invocation: $7.50-$15.00 (3 reviewers)
