#!/usr/bin/env bash
# Phase 2: Wire 7 bioinformatics agents into agents-index.json, routing-schema.json, CLAUDE.md
# Also creates the /review-bioinformatics SKILL.md
# Run from project root: bash scripts/wire-bioinformatics-agents.sh
set -euo pipefail

AGENTS_DIR="$HOME/.claude/agents"
INDEX="$AGENTS_DIR/agents-index.json"
SCHEMA="$HOME/.claude/routing-schema.json"
CLAUDE_MD="$HOME/.claude/CLAUDE.md"
SKILL_DIR="$HOME/.claude/skills/review-bioinformatics"

echo "=== Phase 2: Wiring 7 bioinformatics agents ==="

###############################################################################
# TASK 1: agents-index.json — add 7 entries + update model_tiers
###############################################################################
echo ""
echo "--- Task 1: agents-index.json ---"

# Build the 7 new agent entries as a JSON array
NEW_AGENTS=$(cat << 'JSON_EOF'
[
  {
    "id": "genomics-reviewer",
    "parallelization_template": "C",
    "name": "Genomics Reviewer",
    "model": "opus",
    "effortLevel": "high",
    "thinking": true,
    "thinking_budget": 32000,
    "tier": 3,
    "category": "bioinformatics-review",
    "path": "genomics-reviewer",
    "triggers": ["review genomics", "alignment review", "variant calling review", "genome assembly review", "VCF review", "sequencing review"],
    "tools": ["Read", "Glob", "Grep"],
    "cli_flags": {
      "allowed_tools": ["Read", "Glob", "Grep"],
      "additional_flags": ["--permission-mode", "delegate"]
    },
    "auto_activate": null,
    "conventions_required": ["python.md", "R.md"],
    "sharp_edges_count": 0,
    "description": "Genome assembly, variant calling, alignment, and sequence data format review. BWA/Bowtie2/STAR, GATK/bcftools, VCF/BAM/FASTA/FASTQ/GFF/GTF/BED.",
    "context_requirements": {
      "rules": ["agent-guidelines.md"],
      "conventions": {}
    },
    "default_acceptance_criteria": [
      "All files read before generating findings",
      "Findings include file:line references",
      "Severity classification follows bioinformatics review standards"
    ],
    "subagent_type": "Genomics Reviewer"
  },
  {
    "id": "proteomics-reviewer",
    "parallelization_template": "C",
    "name": "Proteomics Reviewer",
    "model": "opus",
    "effortLevel": "high",
    "thinking": true,
    "thinking_budget": 32000,
    "tier": 3,
    "category": "bioinformatics-review",
    "path": "proteomics-reviewer",
    "triggers": ["review proteomics", "protein identification review", "quantification review", "FDR review", "search engine review"],
    "tools": ["Read", "Glob", "Grep"],
    "cli_flags": {
      "allowed_tools": ["Read", "Glob", "Grep"],
      "additional_flags": ["--permission-mode", "delegate"]
    },
    "auto_activate": null,
    "conventions_required": ["python.md", "R.md"],
    "sharp_edges_count": 0,
    "description": "Mass spectrometry proteomics data processing review. Search engine config, FDR control, quantification (TMT/iTRAQ/LFQ/SILAC), mzML/mzXML, PSM scoring.",
    "context_requirements": {
      "rules": ["agent-guidelines.md"],
      "conventions": {}
    },
    "default_acceptance_criteria": [
      "All files read before generating findings",
      "Findings include file:line references",
      "Severity classification follows bioinformatics review standards"
    ],
    "subagent_type": "Proteomics Reviewer"
  },
  {
    "id": "proteogenomics-reviewer",
    "parallelization_template": "C",
    "name": "Proteogenomics Reviewer",
    "model": "opus",
    "effortLevel": "high",
    "thinking": true,
    "thinking_budget": 32000,
    "tier": 3,
    "category": "bioinformatics-review",
    "path": "proteogenomics-reviewer",
    "triggers": ["review proteogenomics", "custom database review", "novel peptide review", "variant peptide review", "splice junction review"],
    "tools": ["Read", "Glob", "Grep"],
    "cli_flags": {
      "allowed_tools": ["Read", "Glob", "Grep"],
      "additional_flags": ["--permission-mode", "delegate"]
    },
    "auto_activate": null,
    "conventions_required": ["python.md", "R.md"],
    "sharp_edges_count": 0,
    "description": "Proteogenomics pipeline review. Custom protein DB construction, novel peptide ID, variant peptides, splice junction peptides, ORF prediction.",
    "context_requirements": {
      "rules": ["agent-guidelines.md"],
      "conventions": {}
    },
    "default_acceptance_criteria": [
      "All files read before generating findings",
      "Findings include file:line references",
      "Severity classification follows bioinformatics review standards"
    ],
    "subagent_type": "Proteogenomics Reviewer"
  },
  {
    "id": "proteoform-reviewer",
    "parallelization_template": "C",
    "name": "Proteoform Reviewer",
    "model": "opus",
    "effortLevel": "high",
    "thinking": true,
    "thinking_budget": 32000,
    "tier": 3,
    "category": "bioinformatics-review",
    "path": "proteoform-reviewer",
    "triggers": ["review proteoform", "top-down review", "PTM analysis review", "intact mass review", "deconvolution review"],
    "tools": ["Read", "Glob", "Grep"],
    "cli_flags": {
      "allowed_tools": ["Read", "Glob", "Grep"],
      "additional_flags": ["--permission-mode", "delegate"]
    },
    "auto_activate": null,
    "conventions_required": ["python.md", "R.md"],
    "sharp_edges_count": 0,
    "description": "Top-down proteomics and proteoform analysis review. Intact mass, PTM combinatorics, proteoform families, deconvolution, sequence coverage.",
    "context_requirements": {
      "rules": ["agent-guidelines.md"],
      "conventions": {}
    },
    "default_acceptance_criteria": [
      "All files read before generating findings",
      "Findings include file:line references",
      "Severity classification follows bioinformatics review standards"
    ],
    "subagent_type": "Proteoform Reviewer"
  },
  {
    "id": "mass-spec-reviewer",
    "parallelization_template": "C",
    "name": "Mass Spectrometry Reviewer",
    "model": "opus",
    "effortLevel": "high",
    "thinking": true,
    "thinking_budget": 32000,
    "tier": 3,
    "category": "bioinformatics-review",
    "path": "mass-spec-reviewer",
    "triggers": ["review mass spec", "instrument review", "acquisition review", "raw data quality review", "spectral processing review", "DIA review", "DDA review"],
    "tools": ["Read", "Glob", "Grep"],
    "cli_flags": {
      "allowed_tools": ["Read", "Glob", "Grep"],
      "additional_flags": ["--permission-mode", "delegate"]
    },
    "auto_activate": null,
    "conventions_required": ["python.md"],
    "sharp_edges_count": 0,
    "description": "MS instrumentation and data acquisition review. DDA/DIA/PRM methods, calibration, raw data quality, vendor formats (Thermo/Bruker/SCIEX/Waters), spectral processing.",
    "context_requirements": {
      "rules": ["agent-guidelines.md"],
      "conventions": {}
    },
    "default_acceptance_criteria": [
      "All files read before generating findings",
      "Findings include file:line references",
      "Severity classification follows bioinformatics review standards"
    ],
    "subagent_type": "Mass Spectrometry Reviewer"
  },
  {
    "id": "bioinformatician-reviewer",
    "parallelization_template": "C",
    "name": "Bioinformatician Reviewer",
    "model": "opus",
    "effortLevel": "high",
    "thinking": true,
    "thinking_budget": 32000,
    "tier": 3,
    "category": "bioinformatics-review",
    "path": "bioinformatician-reviewer",
    "triggers": ["review bioinformatics", "pipeline review", "workflow review", "reproducibility review", "statistical methods review", "Nextflow review", "Snakemake review"],
    "tools": ["Read", "Glob", "Grep"],
    "cli_flags": {
      "allowed_tools": ["Read", "Glob", "Grep"],
      "additional_flags": ["--permission-mode", "delegate"]
    },
    "auto_activate": null,
    "conventions_required": ["python.md", "R.md"],
    "sharp_edges_count": 0,
    "description": "Pipeline architecture and methodology review. Nextflow/Snakemake/WDL workflows, Docker/Singularity/Conda reproducibility, statistics, data provenance.",
    "context_requirements": {
      "rules": ["agent-guidelines.md"],
      "conventions": {}
    },
    "default_acceptance_criteria": [
      "All files read before generating findings",
      "Findings include file:line references",
      "Severity classification follows bioinformatics review standards"
    ],
    "subagent_type": "Bioinformatician Reviewer"
  },
  {
    "id": "pasteur",
    "parallelization_template": "E",
    "name": "Pasteur",
    "model": "opus",
    "effortLevel": "high",
    "thinking": true,
    "thinking_budget": 32000,
    "tier": 3,
    "category": "bioinformatics-review",
    "path": "pasteur",
    "triggers": [],
    "tools": ["Read", "Glob", "Grep"],
    "cli_flags": {
      "allowed_tools": ["Read", "Glob", "Grep"],
      "additional_flags": ["--permission-mode", "delegate"]
    },
    "auto_activate": null,
    "sharp_edges_count": 0,
    "description": "Bioinformatics review synthesizer. Deduplicates findings across domain reviewers, identifies cross-domain contradictions, produces unified BLOCK/WARNING/APPROVE verdict.",
    "context_requirements": {
      "rules": ["agent-guidelines.md"],
      "conventions": {}
    },
    "default_acceptance_criteria": [
      "All reviewer stdout files read",
      "Findings deduplicated across reviewers",
      "Cross-domain issues identified",
      "Unified verdict with justification"
    ],
    "subagent_type": "Pasteur"
  }
]
JSON_EOF
)

# Add the 7 agents to agents[] array
jq --argjson new "$NEW_AGENTS" '.agents += $new' "$INDEX" > "$INDEX.tmp" && mv "$INDEX.tmp" "$INDEX"
echo "  Added 7 agents to agents[] array"

# Add to routing_rules.model_tiers.opus
jq '.routing_rules.model_tiers.opus += ["genomics-reviewer", "proteomics-reviewer", "proteogenomics-reviewer", "proteoform-reviewer", "mass-spec-reviewer", "bioinformatician-reviewer", "pasteur"]' "$INDEX" > "$INDEX.tmp" && mv "$INDEX.tmp" "$INDEX"
echo "  Added 7 agents to model_tiers.opus"

# Validate
jq . "$INDEX" > /dev/null 2>&1 && echo "  JSON valid: OK" || echo "  JSON valid: FAIL"
echo "  Total agents: $(jq '.agents | length' "$INDEX")"

###############################################################################
# TASK 2: routing-schema.json — 3 locations (NO allowlist per C-1)
###############################################################################
echo ""
echo "--- Task 2: routing-schema.json (3 locations, NO allowlist per C-1) ---"

# Location 1: agent_subagent_mapping
jq '.agent_subagent_mapping += {
  "genomics-reviewer": "Genomics Reviewer",
  "proteomics-reviewer": "Proteomics Reviewer",
  "proteogenomics-reviewer": "Proteogenomics Reviewer",
  "proteoform-reviewer": "Proteoform Reviewer",
  "mass-spec-reviewer": "Mass Spectrometry Reviewer",
  "bioinformatician-reviewer": "Bioinformatician Reviewer",
  "pasteur": "Pasteur"
}' "$SCHEMA" > "$SCHEMA.tmp" && mv "$SCHEMA.tmp" "$SCHEMA"
echo "  Location 1: agent_subagent_mapping — 7 entries added"

# Location 2: tiers.opus.agents
jq '.tiers.opus.agents += ["genomics-reviewer", "proteomics-reviewer", "proteogenomics-reviewer", "proteoform-reviewer", "mass-spec-reviewer", "bioinformatician-reviewer", "pasteur"]' "$SCHEMA" > "$SCHEMA.tmp" && mv "$SCHEMA.tmp" "$SCHEMA"
echo "  Location 2: tiers.opus.agents — 7 IDs added"

# Location 3: subagent_types — new bioinformatics_review category
jq '.subagent_types.bioinformatics_review = {
  "description": "Opus-tier bioinformatics domain review and synthesis — read-only analysis of omics pipelines and data processing code",
  "tools": ["Read", "Glob", "Grep"],
  "allows_write": false,
  "agents": ["genomics-reviewer", "proteomics-reviewer", "proteogenomics-reviewer", "proteoform-reviewer", "mass-spec-reviewer", "bioinformatician-reviewer", "pasteur"],
  "rationale": "Bioinformatics review agents analyze code quality and methodology but do not modify files"
}' "$SCHEMA" > "$SCHEMA.tmp" && mv "$SCHEMA.tmp" "$SCHEMA"
echo "  Location 3: subagent_types.bioinformatics_review — category added"

# C-1 verification: ensure NONE of the 7 are in task_invocation_allowlist
for agent_id in genomics-reviewer proteomics-reviewer proteogenomics-reviewer proteoform-reviewer mass-spec-reviewer bioinformatician-reviewer pasteur; do
  if jq -e ".tiers.opus.task_invocation_allowlist | index(\"$agent_id\")" "$SCHEMA" > /dev/null 2>&1; then
    echo "  C-1 VIOLATION: $agent_id found in allowlist!"
  fi
done
echo "  C-1 verified: no agents in task_invocation_allowlist"

# Validate
jq . "$SCHEMA" > /dev/null 2>&1 && echo "  JSON valid: OK" || echo "  JSON valid: FAIL"

###############################################################################
# TASK 3: CLAUDE.md — dispatch table + slash command
###############################################################################
echo ""
echo "--- Task 3: CLAUDE.md ---"

# Insert new Tier 3 bioinformatics review subsection after line 227 (after llm-inference-architect row)
# Find the exact line
INSERT_AFTER=$(grep -n "llm-inference-architect.*LLM Inference Architect" "$CLAUDE_MD" | tail -1 | cut -d: -f1)
if [ -n "$INSERT_AFTER" ]; then
  sed -i "${INSERT_AFTER}a\\
\\
### Tier 3: Opus (Bioinformatics Review — team-run only)\\
\\
| Trigger Patterns | Agent | subagent_type |\\
| --- | --- | --- |\\
| review genomics, alignment, variant calling, VCF | \`genomics-reviewer\` | Genomics Reviewer |\\
| review proteomics, FDR, quantification, search engine | \`proteomics-reviewer\` | Proteomics Reviewer |\\
| review proteogenomics, custom database, novel peptide | \`proteogenomics-reviewer\` | Proteogenomics Reviewer |\\
| review proteoform, top-down, PTM, intact mass, deconvolution | \`proteoform-reviewer\` | Proteoform Reviewer |\\
| review mass spec, instrument, acquisition, DDA, DIA | \`mass-spec-reviewer\` | Mass Spectrometry Reviewer |\\
| review bioinformatics, pipeline, workflow, reproducibility | \`bioinformatician-reviewer\` | Bioinformatician Reviewer |\\
| (wave 2 synthesizer — spawned by team-run only) | \`pasteur\` | Pasteur |" "$CLAUDE_MD"
  echo "  Dispatch table: bioinformatics review section inserted after line $INSERT_AFTER"
else
  echo "  WARN: Could not find llm-inference-architect line for insertion"
fi

# Add /review-bioinformatics to slash commands table (after /review line)
REVIEW_LINE=$(grep -n "| \`/review\`" "$CLAUDE_MD" | head -1 | cut -d: -f1)
if [ -n "$REVIEW_LINE" ]; then
  sed -i "${REVIEW_LINE}a\\
| \`/review-bioinformatics\` | Bioinformatics domain review with Opus specialist reviewers (6 domains + Pasteur synthesis) |" "$CLAUDE_MD"
  echo "  Slash commands: /review-bioinformatics added after line $REVIEW_LINE"
else
  echo "  WARN: Could not find /review line for slash command insertion"
fi

###############################################################################
# TASK 4: Create SKILL.md
###############################################################################
echo ""
echo "--- Task 4: SKILL.md ---"

mkdir -p "$SKILL_DIR"
cat > "$SKILL_DIR/SKILL.md" << 'SKILL_EOF'
---
name: review-bioinformatics
description: Bioinformatics pipeline and omics data processing review with domain-specialist Opus reviewers and Pasteur synthesis
---

# Review Bioinformatics Skill v1.0

## Purpose

Bioinformatics-domain code review through coordinated Opus-tier specialist reviewers. Analyzes changed files, detects omics domains, spawns relevant reviewers via background team-run, then synthesizes findings via Pasteur (wave 2).

**What this skill does:**

1. **Detect** — Find changed files via git diff or specified scope
2. **Classify** — Identify bioinformatics file types and omics domains
3. **Select** — Choose relevant reviewers (max 4) + always include bioinformatician-reviewer
4. **Execute** — Dispatch reviewers (wave 0) + pasteur (wave 1) via background team-run
5. **Launch** — Start `goyoke-team-run` in background, return immediately

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
- `goyoke-team-run` (team execution)

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
3. waves[1] (pasteur) always included
4. Generate stdin files per `.claude/schemas/stdin/bioinformatics-reviewer.json` for each reviewer
5. Generate stdin file per `.claude/schemas/stdin/bioinformatics-pasteur.json` for pasteur
6. Write config.json and all stdin files to team directory

**Team directory:** `{goyoke_session_dir}/teams/{timestamp}.bioinformatics-review/`

**IMPORTANT:** Template values in review-bioinformatics.json are authoritative. Do NOT copy budget/timeout values from the /review SKILL.md (those are stale).

### Phase 5: Launch and Return

```
result = mcp__goyoke-interactive__team_run({
    team_dir: "$team_dir",
    wait_for_start: true,
    timeout_ms: 10000
})
```

Output summary and return immediately:

```
[review-bioinformatics] Review team launched in background
  Reviewers: {selected reviewers}
  Synthesizer: pasteur (wave 2)
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
| Pasteur (synthesis) | Opus | 20-40K | $2.50-$5.00 |
| **Typical (3 reviewers + pasteur)** | | 110-220K | **$10.00-$20.00** |
| **Maximum (4 reviewers + pasteur)** | | 140-280K | **$12.50-$25.00** |
| Budget cap | | | **$30.00** |

---

## Partial Failure Handling

If one or more wave 0 reviewers fail:
- Pasteur synthesizes from available results
- Failed reviewers noted prominently in Pasteur's report
- Caveat added: "Review incomplete — N of M reviewers completed"
- Consider WARNING status due to incomplete coverage

If Pasteur fails:
- Individual reviewer stdout files are still available via `/team-result`
- No cross-domain synthesis, but domain-specific findings are intact

---

## State Files

| File | Purpose | Format |
|------|---------|--------|
| `{team_dir}/config.json` | Team execution config | JSON |
| `{team_dir}/stdin_*.json` | Per-reviewer/pasteur input | JSON |
| `{team_dir}/stdout_*.json` | Per-reviewer/pasteur output | JSON |
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
- Verify `goyoke-team-run` is built and in PATH
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
[review-bioinformatics] Synthesizer: pasteur (wave 2)

[review-bioinformatics] Review team launched in background
  Reviewers: genomics-reviewer bioinformatician-reviewer
  Synthesizer: pasteur
  Files: 3 files across 1 domain
  Team: .goyoke/sessions/.../teams/1712649600.bioinformatics-review
  PID: 54321

Use /team-status to check progress
Use /team-result to view findings when complete
```

---

**Skill Version:** 1.0
**Last Updated:** 2026-04-09
SKILL_EOF
echo "  SKILL.md created at $SKILL_DIR/SKILL.md"

###############################################################################
# VERIFICATION
###############################################################################
echo ""
echo "=== Phase 2 Verification ==="

echo ""
echo "--- JSON validity ---"
jq . "$INDEX" > /dev/null 2>&1 && echo "  agents-index.json: VALID" || echo "  agents-index.json: INVALID"
jq . "$SCHEMA" > /dev/null 2>&1 && echo "  routing-schema.json: VALID" || echo "  routing-schema.json: INVALID"

echo ""
echo "--- agents-index.json checks ---"
echo "  Total agents: $(jq '.agents | length' "$INDEX")"
for agent_id in genomics-reviewer proteomics-reviewer proteogenomics-reviewer proteoform-reviewer mass-spec-reviewer bioinformatician-reviewer pasteur; do
  exists=$(jq --arg id "$agent_id" '[.agents[] | select(.id == $id)] | length' "$INDEX")
  [ "$exists" -eq 1 ] && echo "  $agent_id: present" || echo "  $agent_id: MISSING"
done
echo "  model_tiers.opus count: $(jq '.routing_rules.model_tiers.opus | length' "$INDEX")"

echo ""
echo "--- routing-schema.json checks ---"
for agent_id in genomics-reviewer proteomics-reviewer proteogenomics-reviewer proteoform-reviewer mass-spec-reviewer bioinformatician-reviewer pasteur; do
  mapping=$(jq -r --arg id "$agent_id" '.agent_subagent_mapping[$id] // "MISSING"' "$SCHEMA")
  echo "  mapping $agent_id: $mapping"
done
echo "  bioinformatics_review category: $(jq 'has("subagent_types") and (.subagent_types | has("bioinformatics_review"))' "$SCHEMA")"
echo "  C-1 check (should be 0): $(jq '[.tiers.opus.task_invocation_allowlist[] | select(. == "genomics-reviewer" or . == "proteomics-reviewer" or . == "proteogenomics-reviewer" or . == "proteoform-reviewer" or . == "mass-spec-reviewer" or . == "bioinformatician-reviewer" or . == "pasteur")] | length' "$SCHEMA")"

echo ""
echo "--- CLAUDE.md checks ---"
grep -c "review-bioinformatics" "$CLAUDE_MD" && echo "  /review-bioinformatics mentions found" || echo "  MISSING from CLAUDE.md"
grep -c "pasteur" "$CLAUDE_MD" && echo "  pasteur mentions found" || echo "  pasteur MISSING from CLAUDE.md"
grep -c "genomics-reviewer" "$CLAUDE_MD" && echo "  genomics-reviewer found in dispatch" || echo "  genomics-reviewer MISSING from dispatch"

echo ""
echo "--- SKILL.md check ---"
[ -f "$SKILL_DIR/SKILL.md" ] && echo "  SKILL.md: EXISTS" || echo "  SKILL.md: MISSING"

echo ""
echo "=== Phase 2 Complete ==="
