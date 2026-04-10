# Sharp Edge ID Conventions

> Append-only. IDs referenced in telemetry are permanent — never rename or recycle.
> Each reviewer expansion appends its block below.

## Format

```
{domain}-{category}-{issue}
```

- **domain**: Reserved prefix per reviewer (see table below)
- **category**: Broad failure class (3-5 per domain)
- **issue**: Specific failure mode (kebab-case)

## Reserved Domain Prefixes

| Prefix | Reviewer | Domain |
|--------|----------|--------|
| `genomics` | genomics-reviewer | DNA alignment, variant calling, annotation (WGS/WES/Panel) |
| `proteomics` | proteomics-reviewer | Protein identification, quantification, FDR control |
| `proteogenomics` | proteogenomics-reviewer | Custom databases, novel peptide discovery |
| `proteoform` | proteoform-reviewer | Top-down, intact mass, PTM analysis |
| `massspec` | mass-spec-reviewer | Instrument settings, acquisition, calibration |
| `bioinfo` | bioinformatician-reviewer | Pipeline architecture, reproducibility, statistics |
| `backend` | backend-reviewer | API security, database patterns, auth |

## Category Guidelines

Each domain should define 3-6 categories covering its major failure classes.
Categories should be short (1 word preferred): `ref`, `align`, `vc`, `anno`, `fmt`.

Example: `genomics-ref-wrong-build`, `genomics-vc-no-normalization`, `proteomics-quant-no-fdr`

## Rules

1. Max 20 IDs per reviewer. If you need more, consolidate.
2. Every ID must map to at least one checklist item.
3. Every checklist item must map to at least one ID.
4. IDs are referenced in telemetry JSON `sharp_edge_id` field — never rename.
5. When adding IDs for a new reviewer, append a new section below with the reviewer name as header.

---

## backend (backend-reviewer)

| ID | Severity | Description |
|----|----------|-------------|
| `backend-sec-sql-injection` | critical | SQL queries with string concatenation |
| `backend-sec-command-injection` | critical | Shell execution with user input |
| `backend-sec-auth-bypass` | critical | Missing auth on sensitive endpoints |
| `backend-sec-hardcoded-secrets` | critical | Secrets in source code |
| `backend-sec-insecure-deser` | critical | Unsafe pickle/eval/YAML |
| `backend-data-missing-validation` | high | Unvalidated user input |
| `backend-data-n-plus-one` | high | Database query in loop |
| `backend-api-missing-rate-limits` | high | Public endpoints without limits |
| `backend-sec-exposed-stacktrace` | high | Debug info leaked to client |
| `backend-err-missing-context` | medium | Generic errors without logging |

---

## genomics (genomics-reviewer)

Categories: `ref` (reference genome), `align` (alignment), `vc` (variant calling), `anno` (annotation), `fmt` (format compliance), `xstage` (cross-stage consistency).

| ID | Severity | Description |
|----|----------|-------------|
| `genomics-ref-wrong-build` | critical | Mismatched genome build (hg19/hg38/T2T) across pipeline steps |
| `genomics-ref-missing-index` | critical | Reference/BAM/VCF missing required index files (.fai/.dict/.bai/.tbi) |
| `genomics-align-no-mapq` | warning | No mapping quality threshold applied before variant calling |
| `genomics-align-wrong-aligner` | warning | Aligner inappropriate for sequencing data type |
| `genomics-align-no-dedup` | warning | Duplicate marking step missing from pipeline |
| `genomics-align-no-readgroups` | critical | BAM files missing @RG read group information |
| `genomics-align-no-bqsr` | warning | BQSR skipped without documented justification |
| `genomics-vc-wrong-caller` | critical | Variant caller mismatched to variant type or sample context |
| `genomics-vc-no-normalization` | critical | Variants not normalized before annotation/comparison |
| `genomics-vc-no-multiallelic` | critical | Multi-allelic sites not decomposed before per-allele analysis |
| `genomics-vc-no-filter` | warning | No hard filters or VQSR applied to variant callset |
| `genomics-vc-no-gq-filter` | warning | No genotype quality threshold in variant filtering |
| `genomics-anno-build-mismatch` | critical | Annotation database/cache build doesn't match reference genome |
| `genomics-anno-no-transcript` | warning | No transcript selection strategy defined (MANE/canonical) |
| `genomics-fmt-invalid-vcf` | warning | VCF violates 4.3+ spec (malformed header or INFO/FORMAT fields) |
| `genomics-fmt-bad-coordinates` | warning | Coordinate system error — 0-based/1-based mismatch across formats |
| `genomics-xstage-contig-mismatch` | critical | Inconsistent contig naming (chr1 vs 1) across pipeline steps |
| `genomics-align-no-contamination` | critical | No contamination estimation step (VerifyBamID2 / CalculateContamination) |
| `genomics-vc-wrong-dv-model` | critical | DeepVariant model type does not match sequencing data type |
| `genomics-vc-wrong-ploidy` | warning | Sex chromosome or mitochondrial ploidy not handled (default diploid) |
