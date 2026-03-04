# Scientific Review Skill Specification

**Ticket ID**: BINF-001
**Status**: Design Complete - Awaiting Implementation
**Priority**: High
**Created**: 2026-02-25
**Author**: Router + Einstein (Orthogonal Analysis)

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Scientific Domain Context](#2-scientific-domain-context)
3. [Skill Architecture](#3-skill-architecture)
4. [Agent Class: Scientific](#4-agent-class-scientific)
5. [Wave Composition](#5-wave-composition)
6. [Senior-Staff-Scientist Orchestrator](#6-senior-staff-scientist-orchestrator)
7. [Scientific Reviewer Agents](#7-scientific-reviewer-agents)
8. [Stdin/Stdout Contract Schemas](#8-stdinstdout-contract-schemas)
9. [Team Config Template](#9-team-config-template)
10. [Anti-Pattern Detection Matrix](#10-anti-pattern-detection-matrix)
11. [Cost Model](#11-cost-model)
12. [Extensibility Framework](#12-extensibility-framework)
13. [Implementation Checklist](#13-implementation-checklist)
14. [Future Scientific Domains](#14-future-scientific-domains)

---

## 1. Executive Summary

### Purpose

The `/scientific-review` skill provides automated multi-perspective scientific review for bioinformatics and computational biology applications. It orchestrates domain-expert reviewer agents to validate scientific correctness, biological plausibility, and computational reproducibility.

### Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| **Model Tier** | Sonnet+Thinking (ALL reviewers) | Scientific review requires reasoning, not mechanical checking |
| **Wave 3** | python-pro | Provide actionable code feedback based on scientific findings |
| **Wave 4** | senior-staff-scientist | Final synthesis with full context from all reviewers |
| **Agent Class** | New `scientific` class | Distinct from code reviewers; extensible for future domains |
| **Orchestrator** | senior-staff-scientist (Opus) | Interview, scope assessment, team composition, synthesis |

### Cost Estimate

| Configuration | Estimated Cost |
|---------------|----------------|
| Full Team (5 reviewers + python-pro + synthesis) | ~$4.50 - $6.00 |
| Focused Review (2-3 reviewers) | ~$2.50 - $3.50 |

---

## 2. Scientific Domain Context

### Target Application: Variant-to-Peptide Pipeline

The initial implementation targets the VCF-VEP-to-fasta codebase, which performs:

```
VCF (variants)
    ↓
VEP (annotation)
    ↓
PyEnsembl (protein sequence generation)
    ↓
In Silico Digestion (enzymatic cleavage)
    ↓
Peptide Database (MS-compatible FASTA)
```

### Pipeline Stages as Review Domains

| Stage | Scientific Domain | Key Files | Primary Reviewer |
|-------|-------------------|-----------|------------------|
| 1. Variant Input | Genomics/Sequencing QC | `src/vep/parser.py` | genomics-reviewer |
| 2. Annotation | Functional Genomics | `src/vep/utils.py` | genomics-reviewer |
| 3. Protein Generation | Molecular Biology | `src/vep/protein_generator.py` | protein-biochemist |
| 4. Digestion | Proteomics/Mass Spec | `src/digest.py` | proteomics-reviewer |
| 5. Database | Computational Proteomics | `src/vep/fasta_writer.py` | proteomics-reviewer |
| Cross-cutting | Algorithm Correctness | `src/vep/pipeline.py` | computational-biologist |
| Cross-cutting | Reproducibility | All config/logs | reproducibility-auditor |

### Critical Scientific Concerns

1. **Reference Genome Concordance**: VCF, VEP, and PyEnsembl must use same assembly (GRCh37 vs GRCh38)
2. **Strand Handling**: Minus-strand gene variants must be reverse-complemented correctly
3. **Coordinate Mapping**: Genomic position → cDNA offset → protein position must be accurate
4. **Consequence Validation**: VEP-predicted effects must match observed sequence changes
5. **MS Compatibility**: Generated peptides must be detectable by mass spectrometry

---

## 3. Skill Architecture

### Three-Layer Design (Following Team-Run Pattern)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       LAYER 1: SKILL (SKILL.md)                              │
│  User-facing entry point                                                     │
│  - Parse user request (/scientific-review)                                   │
│  - Spawn senior-staff-scientist orchestrator                                 │
│  - Launch gogent-team-run in background                                      │
│  - Return monitoring instructions                                            │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                LAYER 2: GO ORCHESTRATION (gogent-team-run)                   │
│  Background execution engine                                                 │
│  - Wave-by-wave agent spawning                                               │
│  - Budget tracking with cost reconciliation                                  │
│  - Inter-wave synthesis (gogent-team-prepare-synthesis)                      │
│  - Process management                                                        │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│               LAYER 3: JSON SCHEMA CONTRACTS                                 │
│  Typed interfaces between components                                         │
│  - Stdin schemas: What each scientific reviewer receives                     │
│  - Stdout schemas: Structured findings from each reviewer                    │
│  - Config schema: Team state, waves, budget                                  │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Execution Flow

```
User: /scientific-review

    │
    ├─► [gogent-skill-guard] Creates team_dir, writes active-skill.json
    │
    ├─► [Router] Reads active-skill.json, spawns senior-staff-scientist (Opus)
    │
    ├─► [senior-staff-scientist]
    │       ├─► Conducts interview (scope, concerns, files)
    │       ├─► Assesses codebase against ALL scientific reviewers
    │       ├─► Selects appropriate reviewer team
    │       ├─► Writes config.json + stdin files
    │       └─► Returns to router
    │
    ├─► [Router] Validates config.json, launches gogent-team-run
    │
    ├─► [gogent-team-run] (Background)
    │       ├─► Wave 1: Parallel scientific reviewers
    │       ├─► Wave 2: Remaining reviewers (if any)
    │       ├─► Wave 3: python-pro (code feedback)
    │       └─► Wave 4: senior-staff-scientist (synthesis)
    │
    └─► [User]
            ├─► /team-status (check progress)
            └─► /team-result (view findings)
```

---

## 4. Agent Class: Scientific

### New Agent Classification

The `/scientific-review` skill introduces a new agent class: **scientific**. These agents are distinct from code reviewers in that they assess scientific correctness, not just code quality.

```json
{
  "class": "scientific",
  "description": "Domain-expert agents for scientific validation",
  "characteristics": {
    "model": "sonnet",
    "thinking_budget": "10-16K tokens",
    "output_format": "structured findings with severity",
    "domain_knowledge": "specialized scientific discipline"
  }
}
```

### Scientific vs Code Reviewers

| Aspect | Code Reviewers | Scientific Reviewers |
|--------|----------------|---------------------|
| **Focus** | Style, patterns, bugs | Correctness, plausibility, validity |
| **Domain** | Programming languages | Scientific disciplines |
| **Model** | Haiku (style) / Sonnet (logic) | Sonnet+Thinking (ALL) |
| **Output** | Code-level findings | Scientific findings with evidence |
| **Examples** | backend-reviewer, standards-reviewer | genomics-reviewer, protein-biochemist |

### Agent Registry Structure

All scientific agents will be registered in `agents-index.json` with:

```json
{
  "id": "genomics-reviewer",
  "class": "scientific",
  "domain": "bioinformatics",
  "subdomain": "genomics",
  "model": "sonnet",
  "tier": 2,
  "thinking_budget": 12000,
  "description": "Validates VCF quality, VEP annotation accuracy, and genomic data integrity",
  "cli_flags": {
    "allowed_tools": ["Read", "Glob", "Grep", "Bash"],
    "permission_mode": "delegate"
  },
  "spawned_by": ["router", "senior-staff-scientist"],
  "can_spawn": []
}
```

---

## 5. Wave Composition

### Four-Wave Structure

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  WAVE 1: Foundation Reviews (Parallel)                                       │
│  ─────────────────────────────────────                                       │
│  Duration: ~60-90 seconds                                                    │
│                                                                              │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────────────┐   │
│  │ genomics-reviewer│  │ reproducibility- │  │ protein-biochemist       │   │
│  │ (Sonnet+Thinking)│  │ auditor          │  │ (Sonnet+Thinking)        │   │
│  │                  │  │ (Sonnet+Thinking)│  │                          │   │
│  │ VCF quality      │  │ Versions         │  │ Sequence plausibility    │   │
│  │ VEP annotation   │  │ Determinism      │  │ Consequence consistency  │   │
│  │ Reference match  │  │ Audit trail      │  │ Domain effects           │   │
│  └──────────────────┘  └──────────────────┘  └──────────────────────────┘   │
│                                                                              │
│  on_complete: null (no inter-wave synthesis needed)                          │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  WAVE 2: Domain-Specific Reviews (Parallel, receives Wave 1 context)         │
│  ──────────────────────────────────────────────────────────────────          │
│  Duration: ~60-90 seconds                                                    │
│                                                                              │
│  ┌──────────────────────────┐  ┌────────────────────────────────────────┐   │
│  │ proteomics-reviewer      │  │ computational-biologist                │   │
│  │ (Sonnet+Thinking)        │  │ (Sonnet+Thinking)                      │   │
│  │                          │  │                                        │   │
│  │ MS compatibility         │  │ Algorithm correctness                  │   │
│  │ Peptide properties       │  │ Coordinate mapping                     │   │
│  │ Database design          │  │ Strand handling                        │   │
│  │ Digestion validation     │  │ Edge case analysis                     │   │
│  └──────────────────────────┘  └────────────────────────────────────────┘   │
│                                                                              │
│  on_complete: gogent-team-prepare-synthesis (merges all findings)            │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  WAVE 3: Code Feedback (Sequential, receives all scientific findings)        │
│  ───────────────────────────────────────────────────────────────────         │
│  Duration: ~90-120 seconds                                                   │
│                                                                              │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │ python-pro (Sonnet+Thinking)                                         │   │
│  │                                                                       │   │
│  │ Receives: pre-synthesis.md with all scientific findings              │   │
│  │                                                                       │   │
│  │ Task: Translate scientific findings into actionable code feedback    │   │
│  │       - Where in code to fix each issue                              │   │
│  │       - Suggested implementation patterns                            │   │
│  │       - Priority ordering for fixes                                  │   │
│  │       - Test cases to add                                            │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  on_complete: null                                                           │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  WAVE 4: Synthesis (Sequential, receives all outputs)                        │
│  ─────────────────────────────────────────────────────                       │
│  Duration: ~60-90 seconds                                                    │
│                                                                              │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │ senior-staff-scientist (Opus)                                        │   │
│  │                                                                       │   │
│  │ Receives: All stdout files + python-pro feedback                     │   │
│  │                                                                       │   │
│  │ Task: Final synthesis                                                │   │
│  │       - Aggregate findings by severity                               │   │
│  │       - Identify convergences (multiple reviewers flagged same)      │   │
│  │       - Resolve conflicts between reviewers                          │   │
│  │       - Produce final Scientific Review Report                       │   │
│  │       - Determine PASS / WARNING / BLOCK status                      │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  on_complete: null (final output)                                            │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Wave Dependencies

| Wave | Depends On | Reads From | Writes To |
|------|------------|------------|-----------|
| Wave 1 | None | stdin files, source code | stdout_*.json |
| Wave 2 | Wave 1 | stdin files, source code, Wave 1 stdout | stdout_*.json |
| Wave 3 | Waves 1+2 | pre-synthesis.md | stdout_python-pro.json |
| Wave 4 | Waves 1+2+3 | All stdout files | stdout_senior-staff-scientist.json |

### Reviewer Selection Logic

The senior-staff-scientist assesses the codebase and selects reviewers based on:

| Trigger | Reviewer Selected |
|---------|-------------------|
| `.vcf`, `.vcf.gz`, VCF parsing code | genomics-reviewer |
| VEP output, annotation code | genomics-reviewer |
| Protein sequences, FASTA generation | protein-biochemist |
| Peptide generation, digestion code | proteomics-reviewer |
| Complex algorithms, coordinate mapping | computational-biologist |
| Any scientific code | reproducibility-auditor (always) |

---

## 6. Senior-Staff-Scientist Orchestrator

### Role

The `senior-staff-scientist` acts as the scientific lead, analogous to Mozart in braintrust. Responsibilities:

1. **Intake**: Parse user request and understand scope
2. **Interview**: Conduct structured clarification
3. **Assessment**: Evaluate codebase against ALL scientific reviewers
4. **Team Selection**: Choose appropriate reviewer composition
5. **Synthesis**: Final review with full context (Wave 4)

### Interview Protocol

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  INTERVIEW PROTOCOL: Senior-Staff-Scientist                                  │
│  ──────────────────────────────────────────                                  │
│                                                                              │
│  Q1: SCOPE ASSESSMENT (ALWAYS)                                               │
│  ────────────────────────────                                                │
│  "What aspect of the pipeline are you seeking review for?"                   │
│                                                                              │
│  Options:                                                                    │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │ ○ Full pipeline review (all stages, comprehensive)                  │    │
│  │ ○ Variant input & annotation (VCF, VEP stages)                      │    │
│  │ ○ Protein generation (sequence reconstruction)                      │    │
│  │ ○ Peptide database (digestion, MS compatibility)                    │    │
│  │ ○ Specific concern (describe below)                                 │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
│  Maps to: reviewer_selection[], review_focus                                 │
│                                                                              │
│  ────────────────────────────────────────────────────────────────────────    │
│                                                                              │
│  Q2: PRIMARY CONCERNS (ALWAYS)                                               │
│  ─────────────────────────────                                               │
│  "What are your primary scientific concerns?"                                │
│                                                                              │
│  Options (multi-select):                                                     │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │ ☐ Data integrity (reference genome, version concordance)            │    │
│  │ ☐ Algorithm correctness (coordinate mapping, strand handling)       │    │
│  │ ☐ Biological plausibility (sequences make biological sense)         │    │
│  │ ☐ MS compatibility (peptides are detectable)                        │    │
│  │ ☐ Reproducibility (versions tracked, deterministic output)          │    │
│  │ ☐ All of the above                                                  │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
│  Maps to: focus_areas{} in each reviewer's stdin                             │
│                                                                              │
│  ────────────────────────────────────────────────────────────────────────    │
│                                                                              │
│  Q3: RELEVANT FILES (ALWAYS)                                                 │
│  ───────────────────────────                                                 │
│  "Which files or directories should be reviewed?"                            │
│                                                                              │
│  Options:                                                                    │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │ ○ Entire src/ directory                                             │    │
│  │ ○ Specific paths: [user provides]                                   │    │
│  │ ○ Let me scout first (spawns haiku scout)                           │    │
│  │ ○ Recent changes only (git diff)                                    │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
│  Maps to: relevant_files[] in each reviewer's stdin                          │
│                                                                              │
│  ────────────────────────────────────────────────────────────────────────    │
│                                                                              │
│  Q4: BUDGET (CONDITIONAL - if user has cost concerns)                        │
│  ───────────────────────────────────────────────────                         │
│  "Default budget is $6.00 for full review. Adjust?"                          │
│                                                                              │
│  Validation: Min $2.00, Max $15.00                                           │
│  Maps to: budget_max_usd, budget_remaining_usd                               │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Codebase Assessment

Before team selection, senior-staff-scientist MUST assess codebase against ALL available scientific reviewers:

```python
# Pseudocode for assessment
assessment = {
    "genomics_relevant": check_for_patterns([
        "*.vcf", "*.vcf.gz", "VEP", "VCF", "variant", "annotation",
        "CHROM", "POS", "REF", "ALT", "QUAL"
    ]),
    "protein_relevant": check_for_patterns([
        "protein", "sequence", "amino", "translation", "transcript",
        "FASTA", "missense", "frameshift"
    ]),
    "proteomics_relevant": check_for_patterns([
        "peptide", "digest", "trypsin", "cleavage", "mass_spec",
        "enzyme", "missed_cleavage"
    ]),
    "algorithm_relevant": check_for_patterns([
        "coordinate", "offset", "strand", "complement", "mapping",
        "cDNA", "genomic", "position"
    ]),
    "reproducibility_relevant": True  # ALWAYS
}

# Select reviewers based on assessment
selected_reviewers = []
for domain, relevant in assessment.items():
    if relevant:
        selected_reviewers.append(DOMAIN_TO_REVIEWER[domain])
```

### Agent Definition

```json
{
  "id": "senior-staff-scientist",
  "class": "scientific",
  "role": "orchestrator",
  "domain": "bioinformatics",
  "subdomain": "multi-domain",
  "model": "opus",
  "tier": 3,
  "thinking_budget": 24000,
  "description": "Senior scientific lead who orchestrates multi-domain scientific review. Conducts intake interview, assesses codebase, selects reviewer team, and synthesizes final report.",
  "cli_flags": {
    "allowed_tools": ["Read", "Glob", "Grep", "Bash", "Write", "AskUserQuestion"],
    "permission_mode": "delegate"
  },
  "spawned_by": ["router"],
  "can_spawn": ["genomics-reviewer", "protein-biochemist", "proteomics-reviewer", "computational-biologist", "reproducibility-auditor"]
}
```

---

## 7. Scientific Reviewer Agents

### 7.1 genomics-reviewer

**Domain**: Genomics / Sequencing QC / Variant Annotation

**Focus Areas**:
- VCF file quality and format compliance
- VEP annotation accuracy
- Reference genome concordance (GRCh37 vs GRCh38)
- Variant quality score interpretation
- REF allele validation

**What to Look For**:
| Check | Detection Method | Severity |
|-------|------------------|----------|
| Reference genome mismatch | Check chromosome naming (chr1 vs 1), REF allele warnings | CRITICAL |
| VEP version skew | Compare VEP cache version to annotation request | CRITICAL |
| Low QUAL variants included | Check QUAL threshold vs industry standards | WARNING |
| Missing annotations | Count variants with missing consequence | WARNING |
| Multi-allelic handling | Check if multi-allelic sites split correctly | INFO |

**Agent Definition**:
```json
{
  "id": "genomics-reviewer",
  "class": "scientific",
  "domain": "bioinformatics",
  "subdomain": "genomics",
  "model": "sonnet",
  "tier": 2,
  "thinking_budget": 12000,
  "description": "Validates VCF quality, VEP annotation accuracy, reference genome concordance, and genomic data integrity for variant calling pipelines.",
  "cli_flags": {
    "allowed_tools": ["Read", "Glob", "Grep", "Bash"],
    "permission_mode": "delegate"
  },
  "spawned_by": ["router", "senior-staff-scientist"],
  "can_spawn": []
}
```

---

### 7.2 protein-biochemist

**Domain**: Molecular Biology / Protein Structure

**Focus Areas**:
- Protein sequence plausibility
- Consequence type consistency
- Reading frame correctness
- Domain effect assessment
- Stop codon handling

**What to Look For**:
| Check | Detection Method | Severity |
|-------|------------------|----------|
| Consequence mismatch | missense_variant but no AA change | CRITICAL |
| Implausible protein length | Proteins <10 or >35000 amino acids | WARNING |
| Start-lost full proteins | start_lost producing full-length protein | WARNING |
| Missing stop translation | Proteins containing internal * | CRITICAL |
| Frameshift miscalculation | In-frame reported as frameshift or vice versa | CRITICAL |

**Agent Definition**:
```json
{
  "id": "protein-biochemist",
  "class": "scientific",
  "domain": "bioinformatics",
  "subdomain": "molecular-biology",
  "model": "sonnet",
  "tier": 2,
  "thinking_budget": 12000,
  "description": "Validates protein sequence biological plausibility, consequence type consistency, and reading frame correctness for protein generation pipelines.",
  "cli_flags": {
    "allowed_tools": ["Read", "Glob", "Grep", "Bash"],
    "permission_mode": "delegate"
  },
  "spawned_by": ["router", "senior-staff-scientist"],
  "can_spawn": []
}
```

---

### 7.3 proteomics-reviewer

**Domain**: Proteomics / Mass Spectrometry

**Focus Areas**:
- Mass spectrometry compatibility
- Peptide property validation
- Database design assessment
- Digestion rule correctness
- Variant peptide detection

**What to Look For**:
| Check | Detection Method | Severity |
|-------|------------------|----------|
| Undetectable peptides | All peptides <6 or >50 AA | CRITICAL |
| Missing variant peptides | has_snp column shows 0% SNP peptides | CRITICAL |
| Database bloat | 100x more sequences than variants | WARNING |
| Wrong cleavage rules | Trypsin pattern incorrect | CRITICAL |
| Missed cleavage handling | missed_cleavages parameter ignored | WARNING |

**Agent Definition**:
```json
{
  "id": "proteomics-reviewer",
  "class": "scientific",
  "domain": "bioinformatics",
  "subdomain": "proteomics",
  "model": "sonnet",
  "tier": 2,
  "thinking_budget": 12000,
  "description": "Validates mass spectrometry compatibility, peptide properties, database design, and digestion correctness for proteomics database generation.",
  "cli_flags": {
    "allowed_tools": ["Read", "Glob", "Grep", "Bash"],
    "permission_mode": "delegate"
  },
  "spawned_by": ["router", "senior-staff-scientist"],
  "can_spawn": []
}
```

---

### 7.4 computational-biologist

**Domain**: Computational Biology / Algorithm Verification

**Focus Areas**:
- Algorithm correctness
- Coordinate mapping validation
- Strand handling verification
- Edge case analysis
- Performance assessment

**What to Look For**:
| Check | Detection Method | Severity |
|-------|------------------|----------|
| Strand sign error | Minus-strand not reverse-complemented | CRITICAL |
| Off-by-one coordinate | REF allele mismatch, position errors | CRITICAL |
| cDNA offset miscalculation | spliced_offset returns wrong value | CRITICAL |
| Edge case failures | Indels at exon boundaries fail | WARNING |
| Performance bottlenecks | O(n²) algorithms in hot paths | INFO |

**Agent Definition**:
```json
{
  "id": "computational-biologist",
  "class": "scientific",
  "domain": "bioinformatics",
  "subdomain": "computational-biology",
  "model": "sonnet",
  "tier": 2,
  "thinking_budget": 14000,
  "description": "Validates algorithm correctness, coordinate mapping, strand handling, and edge case behavior for computational biology pipelines.",
  "cli_flags": {
    "allowed_tools": ["Read", "Glob", "Grep", "Bash"],
    "permission_mode": "delegate"
  },
  "spawned_by": ["router", "senior-staff-scientist"],
  "can_spawn": []
}
```

---

### 7.5 reproducibility-auditor

**Domain**: Computational Reproducibility

**Focus Areas**:
- Version tracking
- Deterministic output
- Audit trail completeness
- Environment documentation
- Configuration validation

**What to Look For**:
| Check | Detection Method | Severity |
|-------|------------------|----------|
| Missing version info | No Ensembl release in output | WARNING |
| Hardcoded paths | /home/, /Users/ in code | WARNING |
| Non-deterministic output | Random without seed | CRITICAL |
| Untracked dependencies | Missing from requirements | WARNING |
| Missing environment docs | No environment.yml | INFO |

**Agent Definition**:
```json
{
  "id": "reproducibility-auditor",
  "class": "scientific",
  "domain": "bioinformatics",
  "subdomain": "reproducibility",
  "model": "sonnet",
  "tier": 2,
  "thinking_budget": 10000,
  "description": "Validates computational reproducibility including version tracking, deterministic output, and audit trail completeness.",
  "cli_flags": {
    "allowed_tools": ["Read", "Glob", "Grep", "Bash"],
    "permission_mode": "delegate"
  },
  "spawned_by": ["router", "senior-staff-scientist"],
  "can_spawn": []
}
```

---

## 8. Stdin/Stdout Contract Schemas

### 8.1 Common Stdin Fields (All Scientific Reviewers)

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Scientific Reviewer Common Stdin",
  "type": "object",
  "required": ["agent", "workflow", "context", "description", "review_scope", "focus_areas"],
  "properties": {
    "agent": {
      "type": "string",
      "description": "Agent ID from agents-index.json"
    },
    "workflow": {
      "type": "string",
      "enum": ["scientific-review"]
    },
    "context": {
      "type": "object",
      "required": ["project_root", "team_dir"],
      "properties": {
        "project_root": {"type": "string"},
        "team_dir": {"type": "string"},
        "pipeline_type": {
          "type": "string",
          "enum": ["variant-to-peptide", "rnaseq", "metagenomics", "custom"]
        }
      }
    },
    "description": {
      "type": "string",
      "description": "Human-readable task description"
    },
    "review_scope": {
      "type": "object",
      "required": ["files"],
      "properties": {
        "files": {
          "type": "array",
          "items": {
            "type": "object",
            "properties": {
              "path": {"type": "string"},
              "stage": {"type": "string"},
              "priority": {"type": "string", "enum": ["high", "medium", "low"]}
            }
          }
        },
        "total_files": {"type": "integer"},
        "pipeline_stages": {
          "type": "array",
          "items": {"type": "string"}
        }
      }
    },
    "focus_areas": {
      "type": "object",
      "description": "Reviewer-specific focus areas (boolean flags)"
    },
    "prior_findings": {
      "type": "object",
      "description": "Findings from Wave 1 reviewers (for Wave 2+)",
      "properties": {
        "critical_issues": {"type": "array"},
        "warnings": {"type": "array"},
        "affected_files": {"type": "array"}
      }
    }
  }
}
```

### 8.2 Common Stdout Fields (All Scientific Reviewers)

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Scientific Reviewer Common Stdout",
  "type": "object",
  "required": ["$schema", "status", "metadata", "findings", "summary"],
  "properties": {
    "$schema": {
      "type": "string",
      "description": "Must match filename without .json (e.g., scientific-review-genomics-reviewer)"
    },
    "status": {
      "type": "string",
      "enum": ["complete", "partial", "failed"]
    },
    "metadata": {
      "type": "object",
      "required": ["thinking_budget_used", "files_reviewed", "review_duration_ms"],
      "properties": {
        "thinking_budget_used": {"type": "integer"},
        "files_reviewed": {"type": "integer"},
        "review_duration_ms": {"type": "integer"}
      }
    },
    "findings": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["id", "severity", "category", "title", "description"],
        "properties": {
          "id": {
            "type": "string",
            "pattern": "^[A-Z]{3}-[0-9]{3}$",
            "description": "Unique finding ID (e.g., GEN-001, PRO-002)"
          },
          "severity": {
            "type": "string",
            "enum": ["critical", "warning", "info"]
          },
          "category": {
            "type": "string",
            "description": "Scientific category (e.g., 'reference_concordance', 'strand_handling')"
          },
          "title": {
            "type": "string",
            "description": "Brief title (<80 chars)"
          },
          "description": {
            "type": "string",
            "description": "Detailed explanation with scientific context"
          },
          "evidence": {
            "type": "object",
            "properties": {
              "file": {"type": "string"},
              "line": {"type": "integer"},
              "code_snippet": {"type": "string"},
              "observation": {"type": "string"}
            }
          },
          "scientific_impact": {
            "type": "string",
            "description": "Why this matters scientifically"
          },
          "recommendation": {
            "type": "string",
            "description": "How to address this finding"
          }
        }
      }
    },
    "summary": {
      "type": "object",
      "required": ["total_findings", "by_severity", "assessment"],
      "properties": {
        "total_findings": {"type": "integer"},
        "by_severity": {
          "type": "object",
          "properties": {
            "critical": {"type": "integer"},
            "warning": {"type": "integer"},
            "info": {"type": "integer"}
          }
        },
        "assessment": {
          "type": "string",
          "enum": ["pass", "warning", "block"],
          "description": "Reviewer's overall assessment"
        },
        "key_concerns": {
          "type": "array",
          "items": {"type": "string"}
        }
      }
    }
  }
}
```

### 8.3 Reviewer-Specific Stdin Extensions

#### genomics-reviewer

```json
{
  "focus_areas": {
    "vcf_quality": true,
    "vep_annotation": true,
    "reference_concordance": true,
    "qual_thresholds": true,
    "multi_allelic": false
  },
  "genomics_context": {
    "expected_genome_build": "GRCh38",
    "expected_vep_version": "115",
    "qual_threshold_used": 30
  }
}
```

#### protein-biochemist

```json
{
  "focus_areas": {
    "sequence_plausibility": true,
    "consequence_consistency": true,
    "reading_frame": true,
    "domain_effects": true,
    "stop_codon_handling": true
  },
  "protein_context": {
    "expected_length_range": [10, 35000],
    "consequence_types_present": ["missense_variant", "frameshift_variant"]
  }
}
```

#### proteomics-reviewer

```json
{
  "focus_areas": {
    "ms_compatibility": true,
    "peptide_properties": true,
    "database_design": true,
    "digestion_validation": true,
    "variant_peptide_detection": true
  },
  "proteomics_context": {
    "target_mass_spec": "Orbitrap",
    "enzyme_used": "trypsin",
    "missed_cleavages": 2,
    "peptide_length_range": [6, 40]
  }
}
```

#### computational-biologist

```json
{
  "focus_areas": {
    "algorithm_correctness": true,
    "coordinate_mapping": true,
    "strand_handling": true,
    "edge_case_analysis": true,
    "performance_assessment": false
  },
  "computational_context": {
    "coordinate_system": "1-based",
    "strand_representation": "+/-"
  }
}
```

#### reproducibility-auditor

```json
{
  "focus_areas": {
    "version_tracking": true,
    "determinism": true,
    "audit_trail": true,
    "environment_docs": true,
    "config_validation": true
  },
  "reproducibility_context": {
    "expected_versions": {
      "ensembl_release": "115",
      "python": "3.12"
    }
  }
}
```

### 8.4 python-pro Stdin (Wave 3)

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Python Pro Code Feedback Stdin",
  "type": "object",
  "required": ["agent", "workflow", "context", "scientific_findings", "code_feedback_scope"],
  "properties": {
    "agent": {
      "type": "string",
      "enum": ["python-pro"]
    },
    "workflow": {
      "type": "string",
      "enum": ["scientific-review"]
    },
    "context": {
      "type": "object",
      "properties": {
        "project_root": {"type": "string"},
        "team_dir": {"type": "string"},
        "pre_synthesis_path": {
          "type": "string",
          "description": "Path to pre-synthesis.md with all scientific findings"
        }
      }
    },
    "scientific_findings": {
      "type": "object",
      "description": "Aggregated findings from scientific reviewers",
      "properties": {
        "critical": {"type": "array"},
        "warnings": {"type": "array"},
        "affected_files": {
          "type": "array",
          "items": {
            "type": "object",
            "properties": {
              "path": {"type": "string"},
              "findings": {"type": "array"}
            }
          }
        }
      }
    },
    "code_feedback_scope": {
      "type": "object",
      "properties": {
        "provide_fixes": {
          "type": "boolean",
          "description": "Whether to suggest specific code fixes"
        },
        "suggest_tests": {
          "type": "boolean",
          "description": "Whether to suggest test cases"
        },
        "prioritize_fixes": {
          "type": "boolean",
          "description": "Whether to prioritize fixes by scientific impact"
        }
      }
    }
  }
}
```

### 8.5 python-pro Stdout (Wave 3)

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Python Pro Code Feedback Stdout",
  "type": "object",
  "required": ["$schema", "status", "code_feedback"],
  "properties": {
    "$schema": {
      "type": "string",
      "enum": ["scientific-review-python-pro"]
    },
    "status": {
      "type": "string",
      "enum": ["complete", "partial", "failed"]
    },
    "code_feedback": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["finding_id", "file", "feedback_type", "description"],
        "properties": {
          "finding_id": {
            "type": "string",
            "description": "References scientific finding (e.g., GEN-001)"
          },
          "file": {"type": "string"},
          "line_range": {
            "type": "object",
            "properties": {
              "start": {"type": "integer"},
              "end": {"type": "integer"}
            }
          },
          "feedback_type": {
            "type": "string",
            "enum": ["fix_required", "improvement_suggested", "test_needed", "refactor_recommended"]
          },
          "description": {"type": "string"},
          "suggested_fix": {
            "type": "object",
            "properties": {
              "current_code": {"type": "string"},
              "proposed_code": {"type": "string"},
              "explanation": {"type": "string"}
            }
          },
          "suggested_tests": {
            "type": "array",
            "items": {
              "type": "object",
              "properties": {
                "test_name": {"type": "string"},
                "test_description": {"type": "string"},
                "test_code": {"type": "string"}
              }
            }
          },
          "priority": {
            "type": "string",
            "enum": ["high", "medium", "low"]
          }
        }
      }
    },
    "summary": {
      "type": "object",
      "properties": {
        "total_feedback_items": {"type": "integer"},
        "fixes_required": {"type": "integer"},
        "tests_suggested": {"type": "integer"},
        "estimated_effort": {
          "type": "string",
          "enum": ["trivial", "small", "medium", "large"]
        }
      }
    }
  }
}
```

### 8.6 senior-staff-scientist Synthesis Stdout (Wave 4)

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Senior Staff Scientist Synthesis Stdout",
  "type": "object",
  "required": ["$schema", "status", "review_status", "executive_summary", "findings_by_domain", "convergence_analysis", "recommendations"],
  "properties": {
    "$schema": {
      "type": "string",
      "enum": ["scientific-review-synthesis"]
    },
    "status": {
      "type": "string",
      "enum": ["complete", "partial", "failed"]
    },
    "review_status": {
      "type": "string",
      "enum": ["PASS", "WARNING", "BLOCK"],
      "description": "Overall review determination"
    },
    "executive_summary": {
      "type": "string",
      "description": "2-3 paragraph summary for non-technical stakeholders"
    },
    "findings_by_domain": {
      "type": "object",
      "properties": {
        "genomics": {"$ref": "#/definitions/domain_findings"},
        "protein_biochemistry": {"$ref": "#/definitions/domain_findings"},
        "proteomics": {"$ref": "#/definitions/domain_findings"},
        "computational": {"$ref": "#/definitions/domain_findings"},
        "reproducibility": {"$ref": "#/definitions/domain_findings"}
      }
    },
    "convergence_analysis": {
      "type": "object",
      "properties": {
        "multi_reviewer_issues": {
          "type": "array",
          "description": "Issues flagged by 2+ reviewers",
          "items": {
            "type": "object",
            "properties": {
              "issue": {"type": "string"},
              "flagged_by": {"type": "array", "items": {"type": "string"}},
              "severity_consensus": {"type": "string"}
            }
          }
        },
        "conflicts": {
          "type": "array",
          "description": "Where reviewers disagreed",
          "items": {
            "type": "object",
            "properties": {
              "issue": {"type": "string"},
              "reviewer_a": {"type": "string"},
              "position_a": {"type": "string"},
              "reviewer_b": {"type": "string"},
              "position_b": {"type": "string"},
              "resolution": {"type": "string"}
            }
          }
        }
      }
    },
    "recommendations": {
      "type": "object",
      "properties": {
        "must_fix_before_use": {
          "type": "array",
          "items": {"type": "string"}
        },
        "should_fix_before_production": {
          "type": "array",
          "items": {"type": "string"}
        },
        "nice_to_have": {
          "type": "array",
          "items": {"type": "string"}
        }
      }
    },
    "code_feedback_summary": {
      "type": "object",
      "description": "Summary of python-pro recommendations",
      "properties": {
        "total_fixes": {"type": "integer"},
        "high_priority_fixes": {"type": "integer"},
        "tests_to_add": {"type": "integer"}
      }
    }
  },
  "definitions": {
    "domain_findings": {
      "type": "object",
      "properties": {
        "reviewer": {"type": "string"},
        "assessment": {"type": "string"},
        "critical_count": {"type": "integer"},
        "warning_count": {"type": "integer"},
        "key_findings": {"type": "array", "items": {"type": "string"}}
      }
    }
  }
}
```

---

## 9. Team Config Template

Location: `~/.claude/schemas/teams/scientific-review.json`

```json
{
  "$schema": "./team-config.json",
  "version": "1.0.0",
  "team_name": "scientific-review-TIMESTAMP",
  "workflow_type": "scientific-review",
  "project_root": "",
  "session_id": "",
  "created_at": "",
  "budget_max_usd": 6.0,
  "budget_remaining_usd": 6.0,
  "warning_threshold_usd": 4.8,
  "status": "pending",
  "background_pid": null,
  "started_at": null,
  "completed_at": null,

  "waves": [
    {
      "wave_number": 1,
      "description": "Foundation scientific reviews (parallel)",
      "status": "pending",
      "started_at": null,
      "completed_at": null,
      "on_complete_script": null,
      "members": [
        {
          "member_id": "genomics-reviewer",
          "agent": "genomics-reviewer",
          "model": "sonnet",
          "description": "Validate VCF quality, VEP annotation, reference concordance",
          "status": "pending",
          "process_pid": null,
          "exit_code": null,
          "cost_estimated_usd": 0.60,
          "cost_actual_usd": 0.0,
          "cost_status": "",
          "error_message": "",
          "retry_count": 0,
          "max_retries": 2,
          "timeout_ms": 180000,
          "stdin_file": "stdin_genomics-reviewer.json",
          "stdout_file": "stdout_genomics-reviewer.json",
          "started_at": null,
          "completed_at": null,
          "blocked_by": [],
          "blocks": []
        },
        {
          "member_id": "reproducibility-auditor",
          "agent": "reproducibility-auditor",
          "model": "sonnet",
          "description": "Validate version tracking, determinism, audit trail",
          "status": "pending",
          "process_pid": null,
          "exit_code": null,
          "cost_estimated_usd": 0.50,
          "cost_actual_usd": 0.0,
          "cost_status": "",
          "error_message": "",
          "retry_count": 0,
          "max_retries": 2,
          "timeout_ms": 180000,
          "stdin_file": "stdin_reproducibility-auditor.json",
          "stdout_file": "stdout_reproducibility-auditor.json",
          "started_at": null,
          "completed_at": null,
          "blocked_by": [],
          "blocks": []
        },
        {
          "member_id": "protein-biochemist",
          "agent": "protein-biochemist",
          "model": "sonnet",
          "description": "Validate sequence plausibility, consequence consistency",
          "status": "pending",
          "process_pid": null,
          "exit_code": null,
          "cost_estimated_usd": 0.60,
          "cost_actual_usd": 0.0,
          "cost_status": "",
          "error_message": "",
          "retry_count": 0,
          "max_retries": 2,
          "timeout_ms": 180000,
          "stdin_file": "stdin_protein-biochemist.json",
          "stdout_file": "stdout_protein-biochemist.json",
          "started_at": null,
          "completed_at": null,
          "blocked_by": [],
          "blocks": []
        }
      ]
    },
    {
      "wave_number": 2,
      "description": "Domain-specific reviews with Wave 1 context (parallel)",
      "status": "pending",
      "started_at": null,
      "completed_at": null,
      "on_complete_script": "gogent-team-prepare-synthesis",
      "members": [
        {
          "member_id": "proteomics-reviewer",
          "agent": "proteomics-reviewer",
          "model": "sonnet",
          "description": "Validate MS compatibility, peptide properties, database design",
          "status": "pending",
          "process_pid": null,
          "exit_code": null,
          "cost_estimated_usd": 0.60,
          "cost_actual_usd": 0.0,
          "cost_status": "",
          "error_message": "",
          "retry_count": 0,
          "max_retries": 2,
          "timeout_ms": 180000,
          "stdin_file": "stdin_proteomics-reviewer.json",
          "stdout_file": "stdout_proteomics-reviewer.json",
          "started_at": null,
          "completed_at": null,
          "blocked_by": [],
          "blocks": []
        },
        {
          "member_id": "computational-biologist",
          "agent": "computational-biologist",
          "model": "sonnet",
          "description": "Validate algorithm correctness, coordinate mapping, strand handling",
          "status": "pending",
          "process_pid": null,
          "exit_code": null,
          "cost_estimated_usd": 0.70,
          "cost_actual_usd": 0.0,
          "cost_status": "",
          "error_message": "",
          "retry_count": 0,
          "max_retries": 2,
          "timeout_ms": 180000,
          "stdin_file": "stdin_computational-biologist.json",
          "stdout_file": "stdout_computational-biologist.json",
          "started_at": null,
          "completed_at": null,
          "blocked_by": [],
          "blocks": []
        }
      ]
    },
    {
      "wave_number": 3,
      "description": "Code feedback based on scientific findings",
      "status": "pending",
      "started_at": null,
      "completed_at": null,
      "on_complete_script": null,
      "members": [
        {
          "member_id": "python-pro",
          "agent": "python-pro",
          "model": "sonnet",
          "description": "Translate scientific findings into actionable code feedback",
          "status": "pending",
          "process_pid": null,
          "exit_code": null,
          "cost_estimated_usd": 0.80,
          "cost_actual_usd": 0.0,
          "cost_status": "",
          "error_message": "",
          "retry_count": 0,
          "max_retries": 2,
          "timeout_ms": 300000,
          "stdin_file": "stdin_python-pro.json",
          "stdout_file": "stdout_python-pro.json",
          "started_at": null,
          "completed_at": null,
          "blocked_by": [],
          "blocks": []
        }
      ]
    },
    {
      "wave_number": 4,
      "description": "Final synthesis by senior staff scientist",
      "status": "pending",
      "started_at": null,
      "completed_at": null,
      "on_complete_script": null,
      "members": [
        {
          "member_id": "senior-staff-scientist-synthesis",
          "agent": "senior-staff-scientist",
          "model": "opus",
          "description": "Synthesize all findings, resolve conflicts, produce final report",
          "status": "pending",
          "process_pid": null,
          "exit_code": null,
          "cost_estimated_usd": 1.50,
          "cost_actual_usd": 0.0,
          "cost_status": "",
          "error_message": "",
          "retry_count": 0,
          "max_retries": 2,
          "timeout_ms": 600000,
          "stdin_file": "stdin_senior-staff-scientist-synthesis.json",
          "stdout_file": "stdout_senior-staff-scientist-synthesis.json",
          "started_at": null,
          "completed_at": null,
          "blocked_by": [],
          "blocks": []
        }
      ]
    }
  ]
}
```

---

## 10. Anti-Pattern Detection Matrix

### Critical Patterns (BLOCK if found)

| ID | Pattern | Domain | Detection | Reviewer |
|----|---------|--------|-----------|----------|
| CRIT-001 | Reference genome mismatch | Genomics | `chr1` vs `1` naming mismatch | genomics-reviewer |
| CRIT-002 | Strand sign error | Algorithm | Minus-strand not RC'd | computational-biologist |
| CRIT-003 | Off-by-one coordinate | Algorithm | REF allele mismatch | computational-biologist |
| CRIT-004 | Consequence mismatch | Protein | missense but no AA change | protein-biochemist |
| CRIT-005 | Missing internal stop | Protein | `*` in protein middle | protein-biochemist |
| CRIT-006 | No variant peptides | Proteomics | `has_snp` always False | proteomics-reviewer |
| CRIT-007 | Non-deterministic output | Reproducibility | Different outputs on re-run | reproducibility-auditor |

### Warning Patterns (WARNING status)

| ID | Pattern | Domain | Detection | Reviewer |
|----|---------|--------|-----------|----------|
| WARN-001 | Ensembl version skew | Genomics | VEP vs PyEnsembl mismatch | genomics-reviewer |
| WARN-002 | Low QUAL threshold | Genomics | QUAL < 20 accepted | genomics-reviewer |
| WARN-003 | Micro-proteins | Protein | Length < 10 AA | protein-biochemist |
| WARN-004 | Giant proteins | Protein | Length > 35000 AA | protein-biochemist |
| WARN-005 | Database bloat | Proteomics | 100x sequences vs variants | proteomics-reviewer |
| WARN-006 | Missing version info | Reproducibility | No Ensembl release in output | reproducibility-auditor |
| WARN-007 | Hardcoded paths | Reproducibility | `/home/`, `/Users/` in code | reproducibility-auditor |

### Info Patterns (Noted but not blocking)

| ID | Pattern | Domain | Detection | Reviewer |
|----|---------|--------|-----------|----------|
| INFO-001 | Multi-allelic not split | Genomics | Multiple ALT alleles | genomics-reviewer |
| INFO-002 | Performance bottleneck | Algorithm | O(n²) in hot path | computational-biologist |
| INFO-003 | Missing environment docs | Reproducibility | No environment.yml | reproducibility-auditor |

---

## 11. Cost Model

### Per-Agent Cost Estimates (Sonnet+Thinking)

| Agent | Input Tokens | Output Tokens | Thinking | Est. Cost |
|-------|--------------|---------------|----------|-----------|
| senior-staff-scientist (intake) | 10,000 | 4,000 | 16,000 | $0.75 |
| genomics-reviewer | 15,000 | 4,000 | 12,000 | $0.60 |
| reproducibility-auditor | 12,000 | 3,000 | 10,000 | $0.50 |
| protein-biochemist | 15,000 | 4,000 | 12,000 | $0.60 |
| proteomics-reviewer | 15,000 | 4,000 | 12,000 | $0.60 |
| computational-biologist | 18,000 | 5,000 | 14,000 | $0.70 |
| python-pro | 20,000 | 6,000 | 12,000 | $0.80 |
| senior-staff-scientist (synthesis) | 30,000 | 8,000 | 20,000 | $1.50 |

### Configuration Costs

| Configuration | Agents | Est. Cost | Use Case |
|---------------|--------|-----------|----------|
| **Full Review** | All 5 reviewers + python-pro + synthesis | $5.30 - $6.50 | Comprehensive pipeline review |
| **Variant Focus** | genomics + computational + reproducibility + synthesis | $3.00 - $3.50 | VCF/VEP validation |
| **Protein Focus** | protein-biochemist + proteomics + synthesis | $2.50 - $3.00 | Sequence generation review |
| **Algorithm Only** | computational-biologist + python-pro | $1.50 - $2.00 | Code correctness check |

### Budget Defaults

```json
{
  "budget_max_usd": 6.00,
  "budget_remaining_usd": 6.00,
  "warning_threshold_usd": 4.80,
  "minimum_budget_usd": 2.00,
  "maximum_budget_usd": 15.00
}
```

---

## 12. Extensibility Framework

### Adding New Scientific Domains

The `/scientific-review` skill is designed to support future scientific domains beyond bioinformatics. The framework follows a pattern of domain → subdomain → reviewer:

```
scientific (class)
├── bioinformatics (domain)
│   ├── genomics (subdomain) → genomics-reviewer
│   ├── proteomics (subdomain) → proteomics-reviewer
│   ├── molecular-biology (subdomain) → protein-biochemist
│   ├── computational-biology (subdomain) → computational-biologist
│   └── reproducibility (subdomain) → reproducibility-auditor
│
├── cheminformatics (domain) [FUTURE]
│   ├── molecular-docking → docking-reviewer
│   ├── qsar → qsar-reviewer
│   └── chemical-properties → chemist-reviewer
│
├── clinical-informatics (domain) [FUTURE]
│   ├── ehr-data → clinical-data-reviewer
│   ├── phenotyping → phenotype-reviewer
│   └── cohort-analysis → epidemiologist-reviewer
│
└── imaging (domain) [FUTURE]
    ├── medical-imaging → radiology-reviewer
    ├── microscopy → microscopy-reviewer
    └── image-analysis → cv-reviewer
```

### Steps to Add a New Subdomain

1. **Define the agent** in `agents-index.json`:
   ```json
   {
     "id": "new-subdomain-reviewer",
     "class": "scientific",
     "domain": "bioinformatics",
     "subdomain": "new-subdomain",
     "model": "sonnet",
     "tier": 2,
     "thinking_budget": 12000,
     ...
   }
   ```

2. **Create stdin/stdout contract** in `schemas/teams/stdin-stdout/scientific-review-new-subdomain-reviewer.json`

3. **Update senior-staff-scientist assessment logic** to detect new subdomain patterns

4. **Add anti-pattern detection matrix** entries for the new subdomain

5. **Update team config template** to include the new reviewer in appropriate wave

### Pattern Detection Registry

For extensibility, anti-patterns should be registered in a machine-readable format:

Location: `~/.claude/schemas/scientific-review/anti-patterns.json`

```json
{
  "version": "1.0.0",
  "patterns": {
    "bioinformatics": {
      "genomics": [
        {
          "id": "CRIT-001",
          "name": "reference_genome_mismatch",
          "severity": "critical",
          "detection": {
            "method": "regex",
            "pattern": "chr[0-9XYM]+ vs [0-9XYM]+",
            "files": ["*.vcf", "*.py"]
          },
          "message": "Reference genome naming convention mismatch (chr1 vs 1)"
        }
      ],
      "proteomics": [
        {
          "id": "CRIT-006",
          "name": "no_variant_peptides",
          "severity": "critical",
          "detection": {
            "method": "column_check",
            "column": "has_snp",
            "condition": "all_false",
            "files": ["*_peptides.tsv"]
          },
          "message": "No peptides contain the variant position (has_snp always False)"
        }
      ]
    }
  }
}
```

---

## 13. Implementation Checklist

### Phase 1: Agent Registration

- [ ] Add `genomics-reviewer` to `agents-index.json`
- [ ] Add `protein-biochemist` to `agents-index.json`
- [ ] Add `proteomics-reviewer` to `agents-index.json`
- [ ] Add `computational-biologist` to `agents-index.json`
- [ ] Add `reproducibility-auditor` to `agents-index.json`
- [ ] Add `senior-staff-scientist` to `agents-index.json`
- [ ] Add `class: scientific` field to schema

### Phase 2: Schema Creation

- [ ] Create `schemas/teams/scientific-review.json` (team config template)
- [ ] Create `schemas/teams/stdin-stdout/scientific-review-genomics-reviewer.json`
- [ ] Create `schemas/teams/stdin-stdout/scientific-review-protein-biochemist.json`
- [ ] Create `schemas/teams/stdin-stdout/scientific-review-proteomics-reviewer.json`
- [ ] Create `schemas/teams/stdin-stdout/scientific-review-computational-biologist.json`
- [ ] Create `schemas/teams/stdin-stdout/scientific-review-reproducibility-auditor.json`
- [ ] Create `schemas/teams/stdin-stdout/scientific-review-python-pro.json`
- [ ] Create `schemas/teams/stdin-stdout/scientific-review-synthesis.json`
- [ ] Create `schemas/scientific-review/anti-patterns.json`

### Phase 3: Agent Definition Files

- [ ] Create `agents/scientific/genomics-reviewer/genomics-reviewer.md`
- [ ] Create `agents/scientific/protein-biochemist/protein-biochemist.md`
- [ ] Create `agents/scientific/proteomics-reviewer/proteomics-reviewer.md`
- [ ] Create `agents/scientific/computational-biologist/computational-biologist.md`
- [ ] Create `agents/scientific/reproducibility-auditor/reproducibility-auditor.md`
- [ ] Create `agents/scientific/senior-staff-scientist/senior-staff-scientist.md`

### Phase 4: Skill Implementation

- [ ] Create `skills/scientific-review/SKILL.md`
- [ ] Add to skill registry
- [ ] Test interview protocol
- [ ] Test team selection logic
- [ ] Test wave execution
- [ ] Test synthesis output

### Phase 5: Integration Testing

- [ ] Test against VCF-VEP-to-fasta codebase
- [ ] Validate anti-pattern detection
- [ ] Verify cost estimates
- [ ] Document findings format

---

## 14. Future Scientific Domains

### Near-Term Additions (Q2 2026)

| Domain | Use Case | Reviewers Needed |
|--------|----------|------------------|
| **RNA-Seq Analysis** | Gene expression, differential analysis | `rnaseq-reviewer`, `statistics-reviewer` |
| **Metagenomics** | Microbiome analysis | `taxonomy-reviewer`, `diversity-reviewer` |
| **Structural Biology** | Protein structure prediction | `structure-reviewer`, `docking-reviewer` |

### Medium-Term Additions (Q3-Q4 2026)

| Domain | Use Case | Reviewers Needed |
|--------|----------|------------------|
| **Clinical Genomics** | Variant interpretation, ClinVar | `clinical-genetics-reviewer`, `pathogenicity-reviewer` |
| **Pharmacogenomics** | Drug response prediction | `pharmacologist-reviewer`, `pgx-reviewer` |
| **Multi-omics Integration** | Cross-platform analysis | `integration-reviewer`, `network-reviewer` |

### Long-Term Vision

The `/scientific-review` skill should evolve into a comprehensive scientific validation framework that can review any computational biology pipeline by:

1. **Auto-detecting domain** from file types and code patterns
2. **Selecting appropriate reviewers** from the scientific agent registry
3. **Applying domain-specific anti-pattern detection**
4. **Producing standardized, actionable reports**

This positions GOgent-Fortress as the go-to AI assistant for scientific software development, where scientific rigor is as important as code quality.

---

## Appendix A: Reference Files

### VCF-VEP-to-fasta Codebase Structure

```
/home/doktersmol/Documents/VCF-VEP-to-fasta/
├── src/
│   ├── vep/
│   │   ├── parser.py          # VCF/VEP file parsing
│   │   ├── protein_generator.py # Protein sequence generation
│   │   ├── fasta_writer.py    # FASTA output
│   │   ├── pipeline.py        # Main pipeline orchestration
│   │   └── utils.py           # Helper functions
│   ├── digest.py              # In silico digestion engine
│   ├── validation/            # Validation utilities
│   └── ...
├── tools/                     # CLI entry points
├── tests/                     # Test suite
└── data/                      # Sample data
```

### Key Files for Each Reviewer

| Reviewer | Primary Files | Secondary Files |
|----------|---------------|-----------------|
| genomics-reviewer | `src/vep/parser.py`, `src/vep/utils.py` | `*.vcf` samples |
| protein-biochemist | `src/vep/protein_generator.py` | Generated FASTAs |
| proteomics-reviewer | `src/digest.py` | Peptide TSVs |
| computational-biologist | `src/vep/pipeline.py`, `src/vep/protein_generator.py` | All core modules |
| reproducibility-auditor | `environment.yml`, logs, config | All files |

---

## Appendix B: Example Output

### Scientific Review Report (Synthesized)

```markdown
# Scientific Review Report

**Pipeline**: VCF-VEP-to-fasta v1.0
**Review Date**: 2026-02-25
**Status**: ⚠️ WARNING

## Executive Summary

The variant-to-peptide pipeline is functionally correct but has two warnings
that should be addressed before production use:

1. **WARN-001**: Ensembl version mismatch between VEP cache (v114) and
   PyEnsembl (v115) could cause annotation inconsistencies
2. **WARN-006**: Missing Ensembl release version in FASTA headers reduces
   reproducibility

No critical issues were identified. The pipeline correctly handles strand
orientation, coordinate mapping, and enzymatic digestion.

## Findings by Domain

### Genomics (genomics-reviewer)
- Assessment: WARNING
- Critical: 0, Warning: 1, Info: 0
- Key Finding: WARN-001 - Ensembl version skew detected

### Protein Biochemistry (protein-biochemist)
- Assessment: PASS
- Critical: 0, Warning: 0, Info: 1
- All sequences biologically plausible

### Proteomics (proteomics-reviewer)
- Assessment: PASS
- Critical: 0, Warning: 0, Info: 0
- Peptides MS-compatible, variant peptides present

### Computational (computational-biologist)
- Assessment: PASS
- Critical: 0, Warning: 0, Info: 0
- Coordinate mapping and strand handling correct

### Reproducibility (reproducibility-auditor)
- Assessment: WARNING
- Critical: 0, Warning: 1, Info: 1
- Key Finding: WARN-006 - Missing version in FASTA headers

## Recommendations

### Must Fix Before Production
1. Align Ensembl versions (VEP cache and PyEnsembl)
2. Add Ensembl release to FASTA headers

### Nice to Have
1. Add environment.yml to repository

## Code Feedback (python-pro)

### Fix 1: Add Ensembl Version to FASTA Headers
- File: `src/vep/fasta_writer.py`
- Priority: High
- Suggested fix provided in stdout_python-pro.json
```

---

**Document Version**: 1.0.0
**Last Updated**: 2026-02-25
**Next Review**: After Phase 1 Implementation
