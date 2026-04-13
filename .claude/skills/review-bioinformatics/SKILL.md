---
name: review-bioinformatics
description: Bioinformatics pipeline and omics data processing review with domain-specialist Opus reviewers and Staff Bioinformatician 7-layer synthesis
---

# Review Bioinformatics Skill v1.0

## Purpose

Bioinformatics-domain code review through coordinated Opus-tier specialist reviewers. Analyzes changed files, detects omics domains, spawns relevant reviewers via background team-run, then synthesizes findings via Staff Bioinformatician 7-layer review (wave 2).

**What this skill does:**

1. **Detect** — Find changed files via git diff or specified scope
2. **Classify** — Identify bioinformatics file types and omics domains
3. **Select** — Choose relevant reviewers (max 4) + always include bioinformatician-reviewer
4. **Execute** — Dispatch reviewers (wave 0) + staff-bioinformatician (wave 1) via background team-run
5. **Launch** — Start `gogent-team-run` in background, return immediately

**What this skill does NOT do:**

- Implement fixes (generates recommendations only)
- Review non-bioinformatics code (use `/review` for that)
- Replace domain expert review (supplements, doesn't replace)

---

## Invocation

- `/review-bioinformatics` — Review all staged changes
- `/review-bioinformatics --all` — Review all uncommitted changes
- `/review-bioinformatics --scope=<glob>` — Review specific files
- `/review-bioinformatics path/to/pipeline` — Review specific path

---

## Prerequisites

**Required tools:**

- `git` (for change detection)
- `jq` (JSON processing)
- `gogent-team-run` (team execution)

---

## Workflow

### Phase 1: Detect Changes

Same detection pattern as `/review`:

```bash
review_scope="staged"  # default
# Supports --all, --scope=<glob>, explicit path
files=$(git diff --staged --name-only)
```

### Phase 2: Classify Files and Detect Domains

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
| `.toml` | toml | config |

#### Domain Detection Heuristics

Scan first 50 lines of each file for domain indicators:

**genomics-reviewer indicators:**
BWA, Bowtie2, STAR, samtools, bcftools, GATK, picard, FASTA, FASTQ, BAM, CRAM, VCF, BED, GFF, GTF, alignment, variant

**proteomics-reviewer indicators:**
MaxQuant, Comet, MSFragger, Percolator, mzML, mzXML, pepXML, protXML, PSM, FDR, peptide identification, protein inference

**proteogenomics-reviewer indicators:**
custom database, novel peptide, variant peptide, splice junction, ORF prediction, SAAV (requires ALSO genomics OR proteomics indicators)

**proteoform-reviewer indicators:**
intact mass, deconvolution, proteoform, top-down, PTM combinatorial, TopPIC, ProSight, FLASHDeconv

**mass-spec-reviewer indicators:**
DDA, DIA, PRM, SRM, MRM, Thermo, Bruker, SCIEX, Waters, Orbitrap, TOF, calibration, acquisition

**bioinformatician-reviewer:** ALWAYS included (pipeline architecture, reproducibility, statistics)

### Phase 3: Select Reviewers

1. Score each domain reviewer by indicator match count
2. Always include bioinformatician-reviewer
3. Include top-scoring domain reviewers up to max 4 total
4. Minimum 2 reviewers per invocation

### Phase 4: Generate Team Config and Stdin Files

1. Read template from `.claude/schemas/teams/review-bioinformatics.json`
2. Filter waves[0].members to only selected reviewers
3. waves[1] (staff-bioinformatician) always included
4. Generate stdin files per `.claude/schemas/stdin/bioinformatics-reviewer.json` for each reviewer
5. Generate stdin file per review-bioinformatics-staff-bioinformatician.json schema for staff-bioinformatician
6. Write config.json and all stdin files to team directory

**Team directory:** `{gogent_session_dir}/teams/{timestamp}.bioinformatics-review/`

**IMPORTANT:** Template values in review-bioinformatics.json are authoritative. Do NOT copy budget/timeout values from the /review SKILL.md (those are stale).

### Phase 5: Launch and Return

```
result = mcp__gofortress-interactive__team_run({
    team_dir: "$team_dir",
    wait_for_start: true,
    timeout_ms: 10000
})
```

Output summary and return immediately:

```
[review-bioinformatics] Review team launched in background
  Reviewers: {selected reviewers}
  Synthesizer: staff-bioinformatician (wave 2)
  Files: {count} files across {domain-count} domains
  Team: {team_dir}
  PID: {pid}

Use /team-status to check progress
Use /team-result to view findings when complete
```

---

## Per-Reviewer Focus Areas (in stdin)

| Reviewer | focus_areas |
|----------|-------------|
| genomics-reviewer | `{alignment: true, variant_calling: true, reference_handling: true, format_compliance: true, annotation: true}` |
| proteomics-reviewer | `{search_parameters: true, fdr_control: true, quantification: true, statistics: true}` |
| proteogenomics-reviewer | `{database_construction: true, novel_peptide_validation: true, variant_peptides: true, coordinate_mapping: true}` |
| proteoform-reviewer | `{deconvolution: true, ptm_localization: true, proteoform_families: true, intact_mass: true, sequence_coverage: true}` |
| mass-spec-reviewer | `{acquisition_method: true, instrument_parameters: true, calibration: true, data_conversion: true, spectral_processing: true}` |
| bioinformatician-reviewer | `{reproducibility: true, pipeline_architecture: true, statistics: true, resource_management: true, provenance: true}` |

---

## Cost Model

| Component | Model | Est. Tokens | Cost |
|-----------|-------|-------------|------|
| Detection + Classification | Bash | 0 | $0.00 |
| Config generation | Router | ~2K | $0.00 |
| Per Opus Reviewer | Opus | 30-60K | $2.50-$5.00 |
| Staff Bioinformatician (synthesis) | Opus | 20-40K | $2.50-$5.00 |
| **Typical (3 reviewers + staff-bioinformatician)** | | 110-220K | **$10.00-$20.00** |
| **Maximum (4 reviewers + staff-bioinformatician)** | | 140-280K | **$12.50-$25.00** |
| Budget cap | | | **$30.00** |

---

## Partial Failure Handling

If one or more wave 0 reviewers fail:
- Staff Bioinformatician synthesizes from available results
- Failed reviewers noted prominently in Staff Bioinformatician's report
- Caveat added: "Review incomplete — N of M reviewers completed"
- Consider WARNING status due to incomplete coverage

If Staff Bioinformatician fails:
- Individual reviewer stdout files are still available via `/team-result`
- No cross-domain synthesis, but domain-specific findings are intact

---

## State Files

| File | Purpose | Format |
|------|---------|--------|
| `{team_dir}/config.json` | Team execution config | JSON |
| `{team_dir}/stdin_*.json` | Per-reviewer/staff-bioinformatician input | JSON |
| `{team_dir}/stdout_*.json` | Per-reviewer/staff-bioinformatician output | JSON |
| `{team_dir}/runner.log` | Execution log | Text |

---

## Troubleshooting

**"No bioinformatics files detected"**
- Ensure files have bioinformatics imports/references
- Use `--scope=<glob>` to specify files explicitly

**"Reviewer not found"**
- Ensure agents-index.json includes all 7 bioinformatics agents
- Check routing-schema.json has correct mappings

**"Team launch failed"**
- Check `$team_dir/runner.log` for errors
- Verify `gogent-team-run` is built and in PATH
- Validate `$team_dir/config.json` with `jq .`

---

## Example Session

```bash
$ git status
On branch feature/new-variant-pipeline
Changes to be committed:
  modified:   pipeline/alignment.nf
  modified:   pipeline/variant_calling.nf
  new file:   scripts/annotate_variants.py

$ /review-bioinformatics

[review-bioinformatics] Found 3 files to review
[review-bioinformatics] Detected domains: genomics (alignment, variant calling)
[review-bioinformatics] Selected reviewers: genomics-reviewer, bioinformatician-reviewer
[review-bioinformatics] Synthesizer: staff-bioinformatician (wave 2)

[review-bioinformatics] Review team launched in background
  Reviewers: genomics-reviewer bioinformatician-reviewer
  Synthesizer: staff-bioinformatician
  Files: 3 files across 1 domain
  Team: .gogent/sessions/.../teams/1712649600.bioinformatics-review
  PID: 54321

Use /team-status to check progress
Use /team-result to view findings when complete
```

---

**Skill Version:** 1.0
**Last Updated:** 2026-04-09
