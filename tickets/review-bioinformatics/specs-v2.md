# Specification v2: /review-bioinformatics Skill + 7 Opus Bioinformatics Agents

**Supersedes:** specs.md (v1)
**Review fixes applied:** C-1, M-1 (overridden), M-3, M-4, m-1, m-3
**New additions:** Pasteur synthesizer agent, braintrust expansion pipeline

---

## Context

- **Goal:** Create a bioinformatics-domain code review system mirroring the existing `/review` skill architecture. 6 Opus-tier specialist reviewers cover genomics, proteomics, proteogenomics, proteoforms, mass spectrometry, and general bioinformatics pipeline review. 1 Opus-tier pasteur synthesizer produces a unified report from reviewer findings.
- **Scope:** 7 agent frontmatter files, 1 skill, 1 team config template, 2 stdin schemas, wiring across agents-index.json, routing-schema.json, and CLAUDE.md. Total: ~68 integration points.
- **Constraints:** Opus tier ($0.045/1K tokens), max 4 reviewers per invocation + pasteur, bioinformatician-reviewer always included. Budget cap $30.

---

## Decisions

| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| **No orchestrator agent** | Skill handles orchestration via team-run dispatch (same as /review). An orchestrator would create a tier inversion (Sonnet orchestrating Opus reviewers) and add 9 integration points with no functional benefit. | `bioinformatics-review-orchestrator` (Sonnet). Rejected: tier inversion, doubled scope, team-run already handles parallel dispatch. |
| **Opus tier for all 7 agents** | Bioinformatics review requires deep domain expertise. Sonnet lacks depth for FDR methodology critique or variant calling pipeline design. Pasteur needs Opus for cross-domain synthesis quality. | Sonnet tier with elevated thinking. Rejected: domain review quality is primary value. |
| **Max 4 reviewers per invocation** | 6 Opus reviewers + pasteur costs $17.50-$30. Intelligent selection keeps cost at $10-$25 while covering relevant domains. | Always run all 6. Rejected: most codebases touch 1-2 omics domains per review. |
| **bioinformatician-reviewer always included** | Pipeline architecture, reproducibility, and statistical methodology apply to ALL bioinformatics code. Mirrors standards-reviewer in /review. | Only when pipeline files detected. Rejected: even pure analysis scripts need reproducibility review. |
| **Read/Glob/Grep tools only** | Reviewers and pasteur analyze, they don't modify. Read-only tool set prevents accidental writes. | Include Write for fix patches. Rejected: reviewers recommend, not implement. |
| **cost_ceiling: $5.00 per agent** | Opus with 32K thinking on large pipelines can consume significant tokens. $5 cap prevents runaway costs. | $2.50. Rejected: Opus review of complex statistical methodology may need 50-60K output tokens. |
| **Pasteur synthesizer as wave 2** | Without synthesis, users get 2-4 independent reports with duplicated findings and no cross-domain analysis. Pasteur deduplicates, resolves contradictions, and produces actionable unified verdict. | No synthesizer (user reads individual reports). Rejected: cross-domain issues invisible, duplicated findings waste user time. Manual synthesis by router rejected: router shouldn't do domain-expert work. |
| **Pasteur timeout: 600000 (10 min)** | Synthesis is faster than initial review — pasteur reads structured outputs, not raw code. 10 min is sufficient. | 900000 (same as reviewers). Rejected: pasteur processes structured data, not raw code exploration. |
| **effort: high in frontmatter** | `effort` is an officially supported Claude Code frontmatter field consumed by CC itself. Staff-architect flagged it as phantom (M-1), but this was incorrect — CC reads it. All existing Opus agents (beethoven, einstein, mozart) use it. | Remove effort field. Rejected: CC consumes it, and consistency with existing Opus agents. |
| **No agents on task_invocation_allowlist** | C-1 fix. These are leaf agents spawned by team-run CLI subprocess, not by Task()/spawn_agent. Existing code reviewers (backend-reviewer, etc.) are NOT on the allowlist. Adding them would create an unguarded $5/call invocation path contradicting leaf-agent design. | Add to allowlist for future flexibility. Rejected: creates security/cost risk with no current need. Add when needed. |
| **Split agent creation 2x3** | M-4 fix. Single task creating 6 agents risks scaffolder context overflow (~3000-word description, 12-18 output files). Split into 2 parallel tasks of 3 agents each: reduces per-task scope, enables parallel execution, isolates failures. | Single task for all 6. Rejected: single point of failure, context overflow risk. |
| **Separate pasteur stdin schema** | Pasteur receives fundamentally different input (wave 0 stdout file paths) vs reviewers (code files, git context). A shared schema would require extensive optional fields defeating validation purpose. | Extend bioinformatics-reviewer.json. Rejected: additionalProperties:false blocks extension, and pasteur's input structure is structurally different. |
| **Budget cap $30 (up from $25)** | Adding pasteur ($2.50-$5.00) to 4 reviewers ($10-$20) totals $12.50-$25. $30 cap provides headroom for retries. | Keep $25 (tight). Rejected: $25 doesn't accommodate 4 reviewers + pasteur + 1 retry. |

---

## Integration Points Inventory

### Per Agent (9 points x 7 agents = 63 total)

| # | Integration Point | Location | Notes |
|---|-------------------|----------|-------|
| 1 | Agent frontmatter file | `.claude/agents/{id}/{id}.md` | Complete frontmatter + body sections |
| 2 | agents-index.json entry | `.claude/agents/agents-index.json` agents[] | Full agent object with all fields |
| 3 | agents-index.json relationships | Same file | No spawned_by/can_spawn for leaf agents |
| 4 | agents-index.json model_tiers | Same file, `routing_rules.model_tiers.opus` | Add to opus array |
| 5 | routing-schema.json agent_subagent_mapping | `.claude/routing-schema.json` ~line 556 | Map id -> subagent_type string |
| 6 | routing-schema.json tiers.opus.agents | Same file ~line 210 | Add to agents array |
| 7 | routing-schema.json subagent_types | Same file ~line 433 | bioinformatics_review category |
| 8 | CLAUDE.md dispatch table | `.claude/CLAUDE.md` | Tier 3 subsection |
| 9 | Sharp-edges.yaml | `.claude/agents/{id}/sharp-edges.yaml` | Empty initial file |

**Note:** Integration point #7 from v1 (task_invocation_allowlist) has been REMOVED per C-1 fix. Leaf agents are not added to the allowlist.

### Additional (non-per-agent)

| # | Integration Point | Location |
|---|-------------------|----------|
| 10 | Skill SKILL.md | `.claude/skills/review-bioinformatics/SKILL.md` |
| 11 | Team config template | `.claude/schemas/teams/review-bioinformatics.json` |
| 12 | Reviewer stdin schema | `.claude/schemas/stdin/bioinformatics-reviewer.json` |
| 13 | Pasteur stdin schema | `.claude/schemas/stdin/bioinformatics-pasteur.json` |
| 14 | CLAUDE.md slash command | `.claude/CLAUDE.md` Slash Commands table |

**Total integration points: 63 (per-agent) + 5 (shared) = 68**

---

## Per-Agent Frontmatter Specification

### Shared Fields (all 7 agents)

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
  on_max_reached: "report_incomplete"  # reviewers
  # pasteur uses: "output_raw_findings_with_caveat"
cost_ceiling: 5.00
```

### Domain Reviewers (6 agents)

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
| triggers | review proteogenomics, custom database review, novel peptide review, variant peptide review, splice junction review |
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
| triggers | review mass spec, instrument review, acquisition review, raw data quality review, spectral processing review, DIA review, DDA review |
| conventions_required | python.md |
| focus_areas | Acquisition method suitability, Instrument parameter optimization, Calibration and QC, Vendor-specific data handling, Spectral processing |

#### 6. bioinformatician-reviewer

| Field | Value |
|-------|-------|
| id | `bioinformatician-reviewer` |
| name | Bioinformatician Reviewer |
| subagent_type | Bioinformatician Reviewer |
| description | Pipeline architecture and methodology review. Nextflow/Snakemake/WDL workflows, Docker/Singularity/Conda reproducibility, statistics, data provenance. |
| triggers | review bioinformatics, pipeline review, workflow review, reproducibility review, statistical methods review, Nextflow review, Snakemake review |
| conventions_required | python.md, R.md |
| focus_areas | Workflow reproducibility, Pipeline architecture, Statistical methodology, Resource management, Data provenance |

### 7. pasteur (Synthesizer)

| Field | Value |
|-------|-------|
| id | `pasteur` |
| name | Pasteur |
| subagent_type | Pasteur |
| description | Bioinformatics review synthesizer. Reads wave 0 reviewer outputs, deduplicates findings, identifies cross-domain contradictions and dependencies, prioritizes by systemic impact, produces unified report with BLOCK/WARNING/APPROVE verdict. |
| triggers | (none — empty array, spawned by team-run wave 1 only) |
| conventions_required | python.md, R.md |
| focus_areas | Cross-domain finding deduplication, Contradiction identification, Dependency chain analysis, Systemic impact prioritization, Unified verdict generation |
| failure_tracking.on_max_reached | output_raw_findings_with_caveat |
| parallelization_template | F (fully sequential — waits for all wave 0 outputs) |

**Pasteur design rationale:** Mirrors Beethoven's synthesis role in /braintrust but specialized for bioinformatics review. Beethoven synthesizes Einstein (theoretical) + Staff-Architect (practical) analyses into a unified Braintrust Analysis document. Pasteur synthesizes N domain reviewer outputs into a unified bioinformatics review report. Same pattern: receive orthogonal analyses, deduplicate, resolve contradictions, produce unified output with verdict.

**Key differences from Beethoven:**
- Input: N domain reviewer stdout files (vs 2 fixed analyses)
- Output: Unified review report with BLOCK/WARNING/APPROVE (vs Braintrust Analysis document)
- Verdict: Binary (block/warn/approve) vs nuanced (recommendations, open questions)
- Scope: Bioinformatics domain only (vs any domain)

---

## Pasteur Agent Full Specification

### Frontmatter

```yaml
---
id: pasteur
name: Pasteur
description: >
  Bioinformatics review synthesizer. Reads all domain reviewer outputs from wave 0,
  deduplicates findings, identifies cross-domain contradictions and dependencies,
  prioritizes by systemic impact, and produces a unified review report with
  BLOCK/WARNING/APPROVE verdict.

model: opus
effort: high
thinking:
  enabled: true
  budget: 32000

tier: 3
category: bioinformatics-review
subagent_type: Pasteur

triggers: []

tools:
  - Read
  - Glob
  - Grep

conventions_required:
  - python.md
  - R.md

focus_areas:
  - Cross-domain finding deduplication
  - Contradiction identification between reviewers
  - Dependency chain analysis across omics domains
  - Systemic impact prioritization
  - Unified verdict generation

failure_tracking:
  max_attempts: 2
  on_max_reached: "output_raw_findings_with_caveat"

cost_ceiling: 5.00
---
```

### Body Outline

1. **CRITICAL: File Reading Required** — must read ALL wave 0 stdout files before synthesis
2. **Identity** — synthesizer role, what it does/doesn't do
3. **Input Structure** — wave_0_outputs array, review_scope_summary, omics_context
4. **Synthesis Framework** — 5 steps: Extract, Deduplicate, Cross-Reference, Prioritize, Verdict
5. **Output Format** — Human-readable unified report + telemetry JSON
6. **Parallelization** — batch all wave 0 file reads in single message
7. **Constraints** — read-only, no new findings, no agent spawning
8. **Anti-Patterns** — concatenation vs synthesis, favoring reviewers, ignoring contradictions
9. **Quick Checklist** — pre-completion verification

### Pasteur Stdin Schema

**Location:** `.claude/schemas/stdin/bioinformatics-pasteur.json`

Distinct from reviewer stdin because pasteur receives:
- `wave_0_outputs`: array of `{reviewer_id, stdout_file_path, status}` — paths to wave 0 stdout files
- `review_scope_summary`: lightweight scope metadata (total_files, languages, categories)
- `omics_context`: same structure as reviewer stdin (primary_domain, data_formats, pipeline_tools, workflow_manager)

Does NOT receive: full review_scope with file list, git_context, focus_areas, project_conventions (these are in the reviewer outputs pasteur reads).

### Pasteur Output

**Human-readable:** Unified review report with sections: Verdict, Executive Summary, Critical Issues, Cross-Domain Issues, Warnings, Suggestions, Reviewer Agreement Matrix, Coverage Summary.

**Telemetry JSON:**
```json
{
  "synthesizer": "pasteur",
  "verdict": "BLOCK|WARNING|APPROVE",
  "total_findings": 0,
  "deduplicated_findings": 0,
  "critical_count": 0,
  "warning_count": 0,
  "info_count": 0,
  "cross_domain_issues": 0,
  "contradictions_found": 0,
  "contradictions_resolved": 0,
  "reviewers_synthesized": [],
  "coverage": {"files_reviewed": 0, "files_total": 0}
}
```

---

## Team Config Specification (2 Waves)

**Location:** `.claude/schemas/teams/review-bioinformatics.json`

| Field | Value |
|-------|-------|
| team_name | `review-bioinformatics-{timestamp}` |
| workflow_type | `review-bioinformatics` |
| budget_max_usd | 30.0 |
| warning_threshold_usd | 25.0 |

### Wave 0: Domain Reviewers (parallel)

| Property | Value |
|----------|-------|
| wave_number | 1 |
| description | Parallel domain-specific bioinformatics code review |
| members | 6 template entries (skill selects 2-4 at runtime) |
| model | opus (all members) |
| timeout_ms | 900000 (15 min) |
| max_retries | 1 |

### Wave 1: Pasteur Synthesizer (sequential)

| Property | Value |
|----------|-------|
| wave_number | 2 |
| description | Synthesis of domain reviewer findings into unified report |
| members | 1 entry: pasteur |
| model | opus |
| timeout_ms | 600000 (10 min) |
| max_retries | 1 |

---

## Skill Specification

### /review-bioinformatics

**Location:** `.claude/skills/review-bioinformatics/SKILL.md`

6-step workflow: Detect -> Classify -> Select -> Execute -> Launch -> Synthesize

**Authoritative values from review-bioinformatics.json template (M-3 fix):**
- budget_max_usd: 30.0 (NOT 2.0 or 10.0 from stale /review SKILL.md)
- warning_threshold_usd: 25.0
- Reviewer timeout: 900000
- Pasteur timeout: 600000
- All models: opus
- All max_retries: 1

**Partial failure handling (m-3 fix):**
- 1 of N wave 0 reviewers fails: pasteur synthesizes available outputs, notes incomplete coverage
- All wave 0 reviewers fail: report error, no synthesis attempted
- Pasteur fails: present individual wave 0 reports concatenated with caveat

---

## Bidirectional Spawn Relationships

All 7 agents are **leaf agents** spawned by `gogent-team-run` (CLI subprocess), not by `spawn_agent`. Therefore:

- **No `spawned_by` field** in agents-index.json entries
- **No `can_spawn` field** in agents-index.json entries
- **No `task_invocation_allowlist` additions** (C-1 fix)
- **No existing agent needs `can_spawn` updates**

---

## Implementation Phases

### Phase 1: Agent Foundations (All Parallel)

- **Tasks:** task-001, task-002, task-003, task-004, task-005, task-006
- **Files:** 7 agent .md files, 7 sharp-edges.yaml, 1 team config, 2 stdin schemas
- **Dependencies:** None (all tasks are independent)
- **Risk:** Low — boilerplate creation, well-defined patterns
- **Validation:** Files exist, YAML frontmatter parses, JSON validates with jq
- **Rollback:** Delete created directories and files

### Phase 2: Skill + Wiring (Parallel after Phase 1)

- **Tasks:** task-007, task-008, task-009, task-010
- **Files:** SKILL.md, agents-index.json (edits), routing-schema.json (edits), CLAUDE.md (edits)
- **Dependencies:** task-001/002/003 (need agent IDs); task-007 also needs 004/005/006 (skill references schemas)
- **Risk:** Medium — editing shared JSON files risks breaking existing agent routing if edits corrupt JSON
- **Validation:** jq validation on both JSON files, grep for all 7 agent IDs in each file
- **Rollback:** git checkout for JSON/MD files, delete for new directories

### Phase 3: Validation Gate (Sequential after Phase 2)

- **Tasks:** task-011
- **Files:** All files (read-only verification)
- **Dependencies:** All Phase 2 tasks
- **Risk:** None — read-only validation
- **Validation:** 23-point checklist
- **Rollback:** N/A

### Phase 4: Braintrust Expansion (Template — not executed)

- **Documented below** — user follows manually for each of 7 agents
- **Dependencies:** Phase 3 validation passes
- **Risk:** Medium — each /braintrust invocation costs $3-$5
- **Estimated cost:** 7 agents x $3-$5 = $21-$35 total

---

## Dependency Graph

```
Phase 1 (all parallel, no dependencies):

task-001 (agents A)    task-002 (agents B)    task-003 (pasteur)
task-004 (team config) task-005 (reviewer stdin) task-006 (pasteur stdin)
    |                      |                      |
    +----------+-----------+----------+-----------+
               |                      |
Phase 2 (parallel, depend on Phase 1):

task-008 (agents-index) ----+
task-009 (routing-schema) --+-- depend on task-001, task-002, task-003
task-010 (CLAUDE.md) -------+
task-007 (SKILL.md) --------+-- depends on ALL Phase 1 tasks
    |            |          |          |
    +------------+----------+----------+
                 |
Phase 3 (sequential, depends on Phase 2):

task-011 (validation gate) -- depends on task-007, task-008, task-009, task-010

Phase 4 (manual, depends on Phase 3):

/braintrust expansion x7 (documented template, not automated)
```

**Parallelizable groups:**
- Wave 1: task-001, task-002, task-003, task-004, task-005, task-006 (all independent)
- Wave 2: task-007 (needs all Wave 1), task-008/009/010 (need task-001/002/003)
- Wave 3: task-011 (needs all Wave 2)

**Critical path:** task-001 or task-002 -> task-008 -> task-011 (agent creation is the longest Phase 1 task due to file count, blocking Phase 2 wiring)

---

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| JSON corruption in agents-index.json | Medium | High — breaks ALL agent routing | jq validation after every edit; git checkout rollback |
| JSON corruption in routing-schema.json | Medium | High — breaks routing enforcement | jq validation after every edit; git checkout rollback |
| subagent_type mismatch (frontmatter vs routing-schema vs agents-index) | Medium | Medium — agent fails to spawn | 3-way validation in gate (check 13) |
| Scaffolder context overflow on 3-agent tasks | Low | Medium — incomplete agent files | Split into 2x3 (M-4 fix), each task manageable |
| Pasteur fails to synthesize (bad wave 0 outputs) | Low | Medium — no unified report | Graceful fallback: concatenate individual reports with caveat |
| CLAUDE.md edit breaks existing dispatch table | Low | High — all agent routing affected | Targeted Edit insertions, not full-file rewrites |
| Opus reviewers too expensive for routine use | Low | Medium — users avoid skill | Max 4 reviewers, $30 budget cap, intelligent selection |
| Reviewer generates hallucinated findings | Medium | High — false positives undermine trust | CRITICAL file-reading section in every agent; Opus reduces vs Haiku/Sonnet |
| Stale SKILL.md values copied from /review | Medium | Medium — budget/timeout inconsistencies | M-3 fix: explicit note that template JSON is authoritative |
| Dead fields in frontmatter | Low | Low — no functional impact | Validation gate check 23 scans for forbidden fields |
| Pasteur stdin schema mismatch with team-run output | Low | Medium — pasteur can't parse input | Schema specifies exact wave_0_outputs structure matching team-run output format |

---

## Cost Model (Updated)

### Per Invocation

| Component | Model | Est. Tokens | Cost |
|-----------|-------|-------------|------|
| Detection + Classification | Bash | 0 | $0.00 |
| Config generation | Router | ~2K | $0.00 |
| Per Opus Reviewer | Opus | 30-60K | $2.50-$5.00 |
| Pasteur Synthesizer | Opus | 20-40K | $2.50-$5.00 |
| **Typical (3 reviewers + pasteur)** | | **110-220K** | **$10.00-$20.00** |
| **Maximum (4 reviewers + pasteur)** | | **140-280K** | **$12.50-$25.00** |
| **Budget cap** | | | **$30.00** |

### Implementation Cost (This Plan)

| Phase | Tasks | Agent Tier | Est. Cost |
|-------|-------|-----------|-----------|
| Phase 1: Agent Foundations | 6 tasks | Scaffolder (Haiku) | $0.05-$0.10 |
| Phase 2: Skill + Wiring | 4 tasks | Scaffolder/Tech-Docs (Haiku) | $0.03-$0.06 |
| Phase 3: Validation | 1 task | Code-Reviewer (Haiku) | $0.01-$0.02 |
| **Total implementation** | **11 tasks** | | **$0.09-$0.18** |

### Braintrust Expansion Cost (Future)

| Agent | /braintrust cost | Notes |
|-------|-----------------|-------|
| genomics-reviewer | $3-$5 | Mozart + Einstein + Staff-Architect + Beethoven |
| proteomics-reviewer | $3-$5 | Same workflow |
| proteogenomics-reviewer | $3-$5 | Same workflow |
| proteoform-reviewer | $3-$5 | Same workflow |
| mass-spec-reviewer | $3-$5 | Same workflow |
| bioinformatician-reviewer | $3-$5 | Same workflow |
| pasteur | $3-$5 | Same workflow |
| **Total expansion** | **$21-$35** | Run sequentially over multiple sessions |

---

## Testing Strategy

### Validation Gate (task-011): 23 Checks

| Category | Checks | What They Verify |
|----------|--------|-----------------|
| File Existence | 1-6 | All 7 agent .md, 7 sharp-edges.yaml, team config, 2 stdin schemas, SKILL.md exist |
| JSON Validity | 7-11 | agents-index.json, routing-schema.json, team config, both stdin schemas parse with jq |
| ID Consistency | 12 | Frontmatter id matches directory name for all 7 agents |
| Cross-File Consistency | 13-19 | subagent_type 3-way match, agents-index completeness, routing-schema locations, C-1 verification |
| Documentation | 20-21 | CLAUDE.md dispatch table and slash command |
| Schema | 22 | Team config has 2 waves with correct member counts |
| Dead Fields | 23 | No forbidden frontmatter fields in any agent file |

### Runtime Testing (Future — after braintrust expansion)

Not covered by this plan. After body content is expanded, manually test:
- `/review-bioinformatics` on a real bioinformatics pipeline
- Verify team-run spawns agents correctly
- Verify pasteur receives wave 0 outputs
- Verify reviewer selection algorithm picks correct domains
- Verify partial failure handling (kill one reviewer mid-run)

---

## Braintrust Expansion Pipeline

### Overview

After the boilerplate scaffold is complete and validated (Phase 3), each of the 7 agents should be expanded with deep domain content via `/braintrust`. This is a repeatable pipeline the user follows manually for each agent.

### Per-Agent Expansion Workflow

For each agent:

1. User invokes `/braintrust` with agent-specific expansion prompt
2. Mozart interviews for domain requirements, spawns scouts to read current agent body
3. Einstein does deep domain research (bioinformatics best practices, common pitfalls, severity classifications)
4. Staff-Architect reviews for consistency with review framework and other reviewer agents
5. Beethoven synthesizes into expanded agent body content
6. User applies result to the agent .md file body (replacing boilerplate sections)

### Expansion Prompt Templates

#### Template for Domain Reviewers (agents 1-6)

```
Expand the body content of the {agent-name} agent (.claude/agents/{agent-id}/{agent-id}.md).

CURRENT STATE: The agent has boilerplate body sections (Identity, Review Checklist, Severity Classification, Output Format, Parallelization, Constraints, Quick Checklist) that were scaffolded from the backend-reviewer.md pattern. The domain-specific content is placeholder-quality.

GOAL: Replace boilerplate with deep domain expertise:

1. REVIEW CHECKLIST: Expand to 20-30 domain-specific checks organized by priority. Each check should reference specific tools, parameters, file formats, or methodologies. Include:
   - Why this check matters (biological/analytical consequence of getting it wrong)
   - What to look for in code (specific function calls, parameter values, file operations)
   - Common mistakes (patterns that look correct but are wrong)

2. SEVERITY CLASSIFICATION: Expand with 5-10 specific examples per severity level, grounded in real bioinformatics failure modes. Each example should name the specific tool/method/parameter involved.

3. SHARP EDGE CORRELATION: Create a domain-specific sharp edge ID table (like backend-reviewer's sql-injection, auth-bypass table). IDs should be semantic ({domain}-{issue}, e.g., genomics-wrong-reference-build, proteomics-no-fdr-control).

4. IDENTITY: Refine the role statement with domain expertise depth. What distinguishes an expert {domain} reviewer from a generalist?

CONSTRAINTS:
- Keep Output Format and Parallelization sections unchanged (they follow the standard reviewer pattern)
- Keep the CRITICAL file-reading warning unchanged
- All checklist items must be verifiable by reading code (not by running experiments)
- Severity classifications must be actionable (reviewer can determine severity from code alone)

REFERENCE: Read the current agent file at .claude/agents/{agent-id}/{agent-id}.md and the backend-reviewer at .claude/agents/backend-reviewer/backend-reviewer.md for structural patterns.
```

**Per-agent customization (replace `{placeholders}`):**

| Agent | agent-id | agent-name | Domain Focus for Einstein |
|-------|----------|------------|---------------------------|
| 1 | genomics-reviewer | Genomics Reviewer | NGS alignment, variant calling pipelines, reference genome management, VCF spec compliance |
| 2 | proteomics-reviewer | Proteomics Reviewer | MS-based proteomics search engines, FDR control, quantification methods, statistical testing |
| 3 | proteogenomics-reviewer | Proteogenomics Reviewer | Custom database construction, novel peptide validation, variant peptide mapping, FDR in expanded search spaces |
| 4 | proteoform-reviewer | Proteoform Reviewer | Top-down proteomics, spectral deconvolution, PTM localization, proteoform family analysis |
| 5 | mass-spec-reviewer | Mass Spectrometry Reviewer | Instrument parameters, DDA/DIA/PRM acquisition, calibration QC, vendor format handling |
| 6 | bioinformatician-reviewer | Bioinformatician Reviewer | Workflow managers, reproducibility, statistical methodology, resource management, data provenance |

#### Template for Pasteur

```
Expand the body content of the Pasteur synthesizer agent (.claude/agents/pasteur/pasteur.md).

CURRENT STATE: The agent has boilerplate synthesis framework (Extract, Deduplicate, Cross-Reference, Prioritize, Verdict) scaffolded from the Beethoven pattern. The bioinformatics-specific synthesis logic is placeholder-quality.

GOAL: Replace boilerplate with domain-aware synthesis expertise:

1. SYNTHESIS FRAMEWORK: Expand each step with bioinformatics-specific logic:
   - Extract: How to parse domain reviewer outputs, what metadata to capture per finding
   - Deduplicate: Domain-aware dedup rules (e.g., "wrong FDR" flagged by both proteomics and proteogenomics reviewers is ONE finding, not two)
   - Cross-Reference: Bioinformatics-specific cross-domain patterns (e.g., reference genome mismatch affects both genomics and proteogenomics pipelines)
   - Prioritize: Domain-aware impact scoring (e.g., irreproducibility is systemic, wrong enzyme is single-pipeline)
   - Verdict: Clear criteria for BLOCK vs WARNING vs APPROVE in bioinformatics context

2. CROSS-DOMAIN ISSUE CATALOG: Enumerate known cross-domain issues:
   - Reference genome consistency across pipeline stages
   - FDR control in expanded search spaces (proteogenomics)
   - Reproducibility across pipeline steps
   - Statistical methodology consistency
   - Data format compatibility between tools

3. CONTRADICTION RESOLUTION: Domain-specific rules for when reviewers disagree (e.g., mass-spec-reviewer says DIA is fine, proteomics-reviewer says quantification is problematic — how to resolve)

CONSTRAINTS:
- Keep Output Format structure unchanged (unified report + telemetry JSON)
- All synthesis must be grounded in reviewer outputs (no new code analysis)
- Verdict must be traceable to specific findings

REFERENCE: Read the current pasteur file at .claude/agents/pasteur/pasteur.md and Beethoven at .claude/agents/beethoven/beethoven.md for synthesis patterns.
```

### Expected Braintrust Output Format

Each `/braintrust` invocation produces a Braintrust Analysis document at `.claude/braintrust/analysis-{timestamp}.md` containing:
- Expanded body sections ready to paste into the agent .md file
- Einstein's domain research (theoretical analysis of bioinformatics best practices)
- Staff-Architect's consistency review (does expanded content match other reviewers' structure?)
- Beethoven's synthesis (unified expansion recommendation)

### Success Criteria for Expanded Agent

After applying braintrust output to an agent .md file:
- [ ] Review Checklist has 20+ domain-specific checks with "why it matters" explanations
- [ ] Severity Classification has 5+ examples per level, each naming specific tools/methods
- [ ] Sharp Edge ID table has 10+ domain-specific IDs
- [ ] Identity section demonstrates domain expertise depth
- [ ] No structural changes to Output Format or Parallelization sections
- [ ] File still has valid YAML frontmatter (frontmatter untouched)
- [ ] All checklist items are code-verifiable (not experiment-dependent)

### Estimated Cost Per Expansion

| Braintrust Agent | Model | Est. Cost |
|-----------------|-------|-----------|
| Mozart (interview + dispatch) | Opus | $0.50-$1.00 |
| Einstein (domain research) | Opus | $1.00-$2.00 |
| Staff-Architect (consistency) | Opus | $0.50-$1.00 |
| Beethoven (synthesis) | Opus | $0.50-$1.00 |
| **Total per agent** | | **$2.50-$5.00** |

---

## Review Fixes Summary

| Fix ID | Status | What Changed |
|--------|--------|-------------|
| **C-1** | FIXED | Removed all agents from task_invocation_allowlist. task-009 now updates 3 locations (not 4). Validation gate check 17 explicitly verifies no new agents in allowlist. |
| **M-1** | OVERRIDDEN | Kept `effort: high`. It is a CC-consumed field, not phantom. Staff-architect's grep found 0 matches because existing Opus agents (beethoven, einstein, mozart) use it in frontmatter but it's parsed by CC, not by GOgent code. The field stays. |
| **M-2** | ACKNOWLEDGED | Body sections are BOILERPLATE placeholders. Task descriptions explicitly state this. Braintrust expansion pipeline (Phase 4) provides the mechanism for domain content upgrade. |
| **M-3** | FIXED | task-007 (SKILL.md) contains explicit note: "Use review-bioinformatics.json template values as authoritative, NOT /review SKILL.md hardcoded values." |
| **M-4** | FIXED | task-001 split into task-001 (genomics, proteomics, proteogenomics) and task-002 (proteoform, mass-spec, bioinformatician). Both run in parallel. |
| **m-1** | FIXED | task-008 insertion instruction changed from positional to anchor-based: "Insert before entry with id: review-orchestrator" |
| **m-2** | ACKNOWLEDGED | Validation gate (task-011) still uses code-reviewer. The 23 checks are mechanical (file existence, JSON parsing, string matching) — haiku tier is appropriate. |
| **m-3** | FIXED | Partial failure handling documented in SKILL.md (task-007) and specs-v2.md. |

---

## Success Criteria

### Implementation (Phase 1-3)

- [ ] All 7 agent frontmatter files created with valid YAML and complete boilerplate body sections
- [ ] Team config template has 2 waves (6 reviewers + pasteur), validates with jq
- [ ] Reviewer stdin schema has omics_context, pipeline_context, 6-agent enum
- [ ] Pasteur stdin schema has wave_0_outputs, review_scope_summary, omics_context
- [ ] Skill SKILL.md follows /review structure with 6-step workflow including Synthesize step
- [ ] agents-index.json has 7 new entries, parses with jq, existing entries intact
- [ ] routing-schema.json updated in 3 locations (NOT allowlist), parses with jq, existing entries intact
- [ ] CLAUDE.md has dispatch table subsection (7 agents) and slash command entry
- [ ] Validation gate passes all 23 checks with zero failures
- [ ] No dead frontmatter fields in any agent file
- [ ] No new agents in task_invocation_allowlist (C-1 verified)
- [ ] Budget cap documented as $30.00 (includes pasteur overhead)

### Braintrust Expansion (Phase 4 — future)

- [ ] All 7 agents have 20+ domain-specific review checklist items
- [ ] All 7 agents have 5+ severity examples per level
- [ ] All 7 agents have 10+ sharp edge IDs
- [ ] Pasteur has domain-aware cross-reference and deduplication rules
- [ ] Total expansion cost within $21-$35 budget
