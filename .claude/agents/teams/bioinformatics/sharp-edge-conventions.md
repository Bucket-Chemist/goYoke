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

---

## proteoform (proteoform-reviewer)

Categories: `deconv` (deconvolution), `ptm` (PTM localization), `assign` (proteoform assignment), `mass` (mass discrimination), `coverage` (sequence coverage).

| ID | Severity | Description |
|----|----------|-------------|
| `proteoform-deconv-charge-cascade` | critical | Charge state assignment error producing phantom proteoforms at harmonic masses |
| `proteoform-deconv-harmonic-artifact` | critical | Harmonic artifacts from dominant species at mass/N ratios |
| `proteoform-deconv-em-local-optima` | critical | EM deconvolution merges overlapping proteoforms into chimeric mass |
| `proteoform-deconv-regularization` | critical | UniDec stiffness miscalibrated — FP/FN tradeoff |
| `proteoform-deconv-psf-mismatch` | critical | UniDec mzsig doesn't match instrument peak width |
| `proteoform-deconv-resolution-mismatch` | warning | Deconvolution resolution doesn't match instrument capability |
| `proteoform-deconv-intensity-bias` | warning | EM intensity biased toward abundant species in overlapping envelopes |
| `proteoform-ptm-no-fragment-evidence` | critical | PTM localization claimed without flanking fragment coverage |
| `proteoform-ptm-combinatorial-explosion` | critical | Unbounded PTM combinatorial search — FDR unreliable |
| `proteoform-mass-adduct-as-ptm` | critical | Metal adducts misclassified as PTMs |
| `proteoform-mass-truncation-as-diversity` | warning | In vitro degradation reported as proteoform diversity |
| `proteoform-assign-fdr-wrong-level` | critical | Bottom-up PSM FDR applied to proteoform-spectrum matches |
| `proteoform-assign-small-db-fdr` | warning | Target-decoy FDR unreliable on small databases |
| `proteoform-assign-mass-coincidence` | warning | PTM identity from mass alone when multiple mods same delta |
| `proteoform-coverage-internal-fragments` | warning | Internal fragments misassigned as terminal ions |

---

## massspec (mass-spec-reviewer)

Categories: `spectral` (spectral processing), `cal` (calibration), `acq` (acquisition), `inst` (instrument), `data` (data handling).

| ID | Severity | Description |
|----|----------|-------------|
| `massspec-spectral-centroiding` | critical | Double centroiding or wrong algorithm — split peaks corrupt downstream |
| `massspec-cal-mass-accuracy` | critical | Mass accuracy outside instrument spec — GATES downstream |
| `massspec-cal-mass-drift` | warning | Mass accuracy drift over acquisition sequence |
| `massspec-cal-no-lockmass` | critical | Lock mass configured but not applied, or absent |
| `massspec-acq-mode-mismatch` | warning | Acquisition mode doesn't match experimental goal |
| `massspec-acq-collision-energy` | critical | Wrong collision energy or fragmentation mode |
| `massspec-acq-dda-exclusion` | warning | DDA dynamic exclusion misconfigured |
| `massspec-acq-dia-window` | warning | DIA window scheme inappropriate for precursor density |
| `massspec-acq-dia-cycle-time` | critical | DIA cycle time too long — <6 data points per peak |
| `massspec-acq-tmt-reporter-sn` | warning | TMT reporter S/N inadequate or co-isolation not assessed |
| `massspec-acq-sps-ms3` | critical | SPS-MS3 misconfigured — wrong notches or collision energy |
| `massspec-inst-resolution-mismatch` | warning | Resolution inappropriate for scan type |
| `massspec-inst-agc-injection` | warning | AGC/injection time imbalance |
| `massspec-cal-rt-stability` | warning | RT instability or no RT standards |
| `massspec-cal-no-qc` | warning | No QC samples in acquisition sequence |
| `massspec-data-conversion-fidelity` | critical | Lossy conversion or wrong bit encoding |
| `massspec-data-centroid-profile` | critical | Centroid/profile mode mismatch |

---

## bioinfo (bioinformatician-reviewer)

Categories: `repro` (reproducibility), `arch` (architecture), `stat` (statistics), `resource` (resource management), `audit` (data provenance).

| ID | Severity | Description |
|----|----------|-------------|
| `bioinfo-repro-mutable-tag` | critical | Container image by mutable tag, not SHA256 digest |
| `bioinfo-repro-unlocked-env` | critical | Conda/pip/renv without version lockfile |
| `bioinfo-repro-no-engine-version` | warning | Workflow engine version not pinned |
| `bioinfo-repro-mutable-reference` | critical | Reference data fetched from mutable URL at runtime |
| `bioinfo-repro-no-seed` | critical | Random seed not set for stochastic processes |
| `bioinfo-repro-mutable-base` | warning | Dockerfile FROM without SHA256 digest |
| `bioinfo-arch-silent-fail` | critical | Pipeline continues after step failure |
| `bioinfo-arch-resume-stale` | critical | Resume reuses stale outputs from different params |
| `bioinfo-arch-no-validation` | warning | No input validation before processing |
| `bioinfo-arch-retry-unbounded` | warning | Retry without maxRetries bound |
| `bioinfo-arch-race-condition` | critical | Parallel execution race on shared temp files |
| `bioinfo-arch-non-atomic` | warning | Non-atomic output writes |
| `bioinfo-arch-wdl-portability` | warning | WDL runtime uses backend-specific attributes |
| `bioinfo-stat-no-mtc` | critical | Multiple testing correction absent |
| `bioinfo-stat-wrong-test` | warning | Test assumptions not verified computationally |
| `bioinfo-stat-no-effect-size` | warning | P-values without effect sizes |
| `bioinfo-resource-no-memory` | warning | No memory declaration on processes |
| `bioinfo-resource-no-cleanup` | warning | Intermediate files never cleaned |
| `bioinfo-audit-no-versions` | warning | Software versions not recorded |
| `bioinfo-audit-no-params` | warning | Run parameters not logged |
