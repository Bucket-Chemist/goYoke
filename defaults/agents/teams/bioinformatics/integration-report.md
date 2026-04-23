# Bioinformatics Review Team — Integration Report

**Date:** 2026-04-14
**Scope:** Expansion of proteoform-reviewer, mass-spec-reviewer, bioinformatician-reviewer from boilerplate to production quality
**Related:** Staff-bioinformatician 7-layer framework, sharp-edge-conventions.md, review-bioinformatics team config

---

## 1. Staff-Bioinformatician Boundary Interaction Matrix — MODIFICATIONS

These 5 entries currently use vague string references. Replace with concrete sharp_edge_ids:

| Entry # | Old Vague Reference | Replacement ID(s) | Interaction Type | Rationale |
|---------|--------------------|--------------------|-----------------|-----------|
| 25 | `mass-spec: MS3/FAIMS acquisition` | `massspec-acq-sps-ms3`, `massspec-inst-resolution-mismatch` | negating | MS3 mitigates TMT compression at hardware level; FAIMS affects charge-state population |
| 26 | `mass-spec: DIA acquisition` | `massspec-acq-dia-window`, `massspec-acq-dia-cycle-time` | multiplicative | DIA acquisition problems × mismatched library compound identification failure |
| 27 | `mass-spec: centroiding quality` | `massspec-spectral-centroiding`, `massspec-cal-mass-accuracy` | multiplicative | Poor centroiding × wrong tolerance = noise matches as PSMs |
| 28 | `bioinformatician: container reference consistency` | `bioinfo-repro-mutable-tag`, `bioinfo-repro-mutable-base` | gating | Container reference ≠ pipeline reference invalidates all downstream analyses |
| 29 | `proteoform: PTM site assignment` | `proteoform-ptm-no-fragment-evidence`, `proteoform-mass-adduct-as-ptm` | additive | Variant creates/destroys PTM site not modeled in proteoform analysis |

---

## 2. Staff-Bioinformatician Boundary Interaction Matrix — ADDITIONS

New cross-domain entries to add to the matrix:

### Sequential Boundaries (new)

| # | Upstream ID | Downstream ID | Type | Mechanism |
|---|---|---|---|---|
| 30 | `massspec-spectral-centroiding` | `proteoform-deconv-charge-cascade` | gating | Poor centroiding → deconvolution charge state assignment fails → phantom proteoforms |
| 31 | `massspec-cal-mass-accuracy` | `proteoform-deconv-psf-mismatch` | gating | Mass accuracy drift → PSF model invalid → systematic mass errors in all deconvolved masses |
| 32 | `massspec-acq-collision-energy` | `proteoform-ptm-no-fragment-evidence` | multiplicative | Wrong fragmentation energy → poor fragment coverage → PTM localization fails |
| 33 | `bioinfo-repro-mutable-tag` | `proteogenomics-version-vep-pyensembl` | gating | Container drift → VEP cache version change → different protein sequences |
| 34 | `bioinfo-repro-mutable-reference` | `genomics-ref-wrong-build` | gating | Reference fetched from mutable URL → wrong genome build silently loaded |
| 35 | `bioinfo-arch-silent-fail` | `proteomics-fdr-global-only` | multiplicative | Silent sample dropout × global FDR → FDR calibrated on incomplete sample set |

### Non-Sequential Boundaries (new)

| # | Boundary | Reviewers | Key IDs | What Can Break |
|---|---|---|---|---|
| 36 | Spectral quality × deconvolution | mass-spec × proteoform | `massspec-cal-mass-drift` + `proteoform-deconv-em-local-optima` | Mass drift during acquisition shifts charge state envelopes → EM converges to wrong local optimum |
| 37 | Container × reference consistency | bioinformatician × genomics | `bioinfo-repro-mutable-tag` + `genomics-ref-wrong-build` | Container bundles different reference build than pipeline config specifies |
| 38 | Pipeline error × quantification | bioinformatician × proteomics | `bioinfo-arch-silent-fail` + `proteomics-quant-mnar-as-mcar` | Silent sample dropout creates MNAR pattern that MCAR imputation masks as random |

---

## 3. Staff-Bioinformatician Causal Chain Library — ADDITIONS

6 new causal chains involving the 3 expanded agents:

### Chain 6: Centroiding → Deconvolution Artifacts → False Proteoforms

- **Path:** `massspec-spectral-centroiding` → `proteoform-deconv-charge-cascade` → `proteoform-assign-fdr-wrong-level`
- **Algebra:** GATING at first step → MULTIPLICATIVE at downstream
- **Mechanism:** Poor centroiding splits or merges peaks at the spectral level. Deconvolution algorithms assign charge states based on inter-peak spacing — corrupted spacing produces phantom charge state envelopes. These generate fictitious proteoform masses that enter the PrSM database and inflate false discovery. Unlike database search (which tolerates some mass error), deconvolution is mathematically non-recoverable from centroiding errors.
- **Systemic severity:** critical

### Chain 7: Container Drift → VEP Change → Different Proteins → Search Invalidation

- **Path:** `bioinfo-repro-mutable-tag` → `proteogenomics-version-vep-pyensembl` → proteogenomics protein generation affected → `proteomics-fdr-global-only` compromised
- **Algebra:** GATING at first step, MULTIPLICATIVE at downstream
- **Mechanism:** Container image tag (e.g., `ensemblorg/ensembl-vep:110.1`) is mutable. Maintainer rebuilds with updated VEP cache (110.1→110.2). VEP annotations use different transcript models. PyEnsembl resolves transcript IDs to different exon structures. Proteins built from wrong gene models. Custom database contains different proteins. Search results and FDR calibrated against wrong target distribution. No individual reviewer sees the full chain.
- **Systemic severity:** critical

### Chain 8: Resolution Mismatch → Charge State Confusion → Artifact Proteoforms

- **Path:** `massspec-inst-resolution-mismatch` → `proteoform-deconv-charge-cascade` → `proteoform-mass-adduct-as-ptm`
- **Algebra:** MULTIPLICATIVE
- **Mechanism:** Insufficient MS1 resolution prevents separation of adjacent charge state isotope envelopes. Overlapping envelopes are deconvolved as single species at intermediate mass. The mass error from merged envelopes falls in the range of common adduct/PTM masses (+22 Da Na+, +42 Da acetylation), leading to false PTM assignment on artifact masses.
- **Systemic severity:** critical

### Chain 9: Silent Sample Dropout → Biased Quantification → False Differential Expression

- **Path:** `bioinfo-arch-silent-fail` → incomplete sample set → `proteomics-quant-mnar-as-mcar` → `proteomics-fdr-global-only`
- **Algebra:** MULTIPLICATIVE at first step → ADDITIVE downstream
- **Mechanism:** Nextflow `errorStrategy 'ignore'` causes failed samples to emit empty output channels. `.collect()` aggregates N-1 samples. If the dropped sample is from one treatment group, the missing data pattern is MNAR (not at random — it's missing because the sample failed). MCAR imputation fills the gap with population mean, creating false abundance estimates. Differential expression detects "significant" differences that are artifacts of the dropout + imputation.
- **Systemic severity:** critical

### Chain 10: Unpinned Environment → Tool Version Change → Different Statistical Results

- **Path:** `bioinfo-repro-unlocked-env` → tool behavior change → `bioinfo-stat-no-mtc` (or any downstream statistical analysis)
- **Algebra:** GATING
- **Mechanism:** Unpinned conda environment resolves `scipy>=1.10` to 1.10 in January and 1.14 in June. Statistical test implementation changes between versions (e.g., default alternative hypothesis, tie handling, continuity correction). Results differ between runs with no code change. If multiple testing correction was borderline (p-values near threshold), version change flips significance calls.
- **Systemic severity:** warning (results differ but both may be valid)

### Chain 11: Mass Accuracy Drift → Database Search Mismatch → Identification Failure

- **Path:** `massspec-cal-mass-drift` → `massspec-cal-no-lockmass` → `proteomics-search-precursor-tolerance`
- **Algebra:** ADDITIVE → MULTIPLICATIVE
- **Mechanism:** Mass accuracy drifts over multi-hour acquisition (no lock mass correction). Early samples within ±3 ppm, late samples at ±15 ppm. Search engine tolerance set to ±10 ppm. Late-run precursor masses fall outside tolerance window — peptides present in sample but not matched. Appears as "fewer identifications in later samples" but the root cause is calibration drift, not biology.
- **Systemic severity:** warning (early results valid; late results degraded)

---

## 4. Staff-Bioinformatician Coverage Matrix — Density Upgrades

| Pipeline Stage | Previous Density | New Density | Contributing Agents | New Checks |
|---|---|---|---|---|
| Spectral processing | MEDIUM | **HIGH** | mass-spec (PRIMARY: 32 checks), proteoform (deconvolution: 12 checks) | 44 checks across 2 reviewers with cross-reference IDs |
| Proteoform assignment | LOW (conditional) | **MEDIUM** | proteoform (PRIMARY: 31 checks) | 31 checks including FDR, PTM localization, family assignment |
| Pipeline reproducibility | MEDIUM | **HIGH** | bioinformatician (PRIMARY: 30 checks) | 30 checks across NF/SM/WDL with 3-WM parity |
| RT alignment / MBR | LOW | **LOW-MEDIUM** | mass-spec (`massspec-cal-rt-stability`), proteomics (MBR config) | RT stability check added; MBR interaction clarified |
| In silico digestion | LOW | **LOW** | No change — still falls between proteogenomics (generates) and proteomics (searches) | Structural blind spot remains |

---

## 5. FDR Detection Matrix — Scope Note

**New entry (proteoform FDR):**

| Upstream Finding (proteoform) | Downstream Finding (proteomics) | Failure Type | Severity | What Breaks |
|---|---|---|---|---|
| `proteoform-assign-small-db-fdr` (small DB, unstable FDR) | N/A (proteoform FDR is independent) | FDR RELIABILITY | warning | Proteoform-level FDR unreliable on <500-entry databases; reported "0% FDR" may be statistical artifact |

**Scope exclusion confirmed:** Mass-spec-reviewer and bioinformatician-reviewer do NOT contribute FDR Detection Matrix entries. FDR chains involve identification→statistics stages, not instrument or architecture stages. Mass-spec affects FDR indirectly through spectral quality (already covered by GATING in Boundary Interaction Matrix entries 30-31).

---

## 6. Sharp Edge ID Summary

| Agent | Prefix | Count | Categories |
|---|---|---|---|
| proteoform-reviewer | `proteoform-` | 15 | deconv (7), ptm (2), mass (2), assign (3), coverage (1) |
| mass-spec-reviewer | `massspec-` | 17 | spectral (2), cal (5), acq (7), inst (2), data (2) |
| bioinformatician-reviewer | `bioinfo-` | 20 | repro (6), arch (7), stat (3), resource (2), audit (2) |
| **Total new IDs** | | **52** | |

**ID uniqueness verified:** No collisions with existing IDs from genomics-reviewer (20), proteomics-reviewer (13), or proteogenomics-reviewer (24). Total system-wide: 109 sharp_edge_ids.

---

## 7. Remaining Structural Blind Spots

| Stage | Gap | Recommended Action |
|---|---|---|
| In silico digestion | Falls between proteogenomics (generates) and proteomics (searches). Neither reviewer deeply checks enzyme consistency or cleavage site handling. | Address in proteogenomics-reviewer refine pass (check #49 exists but coverage still LOW) |
| RNA-seq proteogenomics | Expression-informed filtering only checked when RNA-seq present. No reviewer covers RNA-seq quality (alignment, quantification). | Future: transcriptomics-reviewer agent |
| Native MS complex analysis | proteoform-reviewer has [DESIGN] check for native vs denaturing but no deep native MS complex stoichiometry analysis | Future: native-ms specialist or proteoform-reviewer refine pass |
