# Specification: Programmatic Cross-Domain Interaction Detection for goyoke-team-prepare-synthesis

**Version:** 1.0
**Date:** 2026-04-13
**Status:** Proposed
**Author:** Router + Braintrust synthesis
**Implements:** Staff-bioinformatician v2 programmatic enforcement layer

---

## 1. Problem Statement

The staff-bioinformatician agent evaluates cross-domain interactions between wave 0 reviewer findings in `/review-bioinformatics`. Currently (v1), cross-domain interaction detection relies on a static lookup map in agent instructions — text-based, probabilistic, and susceptible to fabrication (the agent inventing connections not supported by reviewer outputs).

The enforcement architecture (`router-guidelines.md` §6.2) requires: **Declarative Rule → Programmatic Enforcement → Reference Documentation.** The static map is documentation without enforcement.

### Goal

Add programmatic interaction detection to `goyoke-team-prepare-synthesis` so that known cross-domain failure patterns are detected deterministically from wave 0 reviewer outputs, pre-computed, and included in `pre-synthesis.md` for the staff-bioinformatician to verify rather than discover.

### Success Criteria

1. Known cross-domain interactions detected with zero false negatives (all defined patterns matched when present)
2. Zero fabrication risk for programmatically-detected interactions (deterministic matching)
3. Staff-bioinformatician's thinking budget consumption reduced by 30%+ (verification vs discovery)
4. Interaction rules maintainable via JSON config without recompiling the Go binary
5. Novel interactions (not in rules) still discoverable by the staff-bioinformatician via "unmatched findings" section
6. Full test coverage: unit tests for pattern matching, integration tests with mock reviewer outputs, golden file tests for pre-synthesis.md generation

---

## 2. Architecture

### Current Flow

```
Wave 0 reviewers (parallel)
    ↓ produce stdout JSON files
goyoke-team-prepare-synthesis (inter-wave script)
    ↓ reads stdout files, produces
pre-synthesis.md (raw findings merged)
    ↓ read by
staff-bioinformatician (wave 1)
    ↓ discovers cross-domain interactions from scratch
stdout_staff-bioinformatician.json
```

### Proposed Flow

```
Wave 0 reviewers (parallel)
    ↓ produce stdout JSON files
goyoke-team-prepare-synthesis (MODIFIED)
    ↓ reads stdout files
    ↓ reads interaction-rules.json (NEW)
    ↓ matches findings against rules
    ↓ produces
pre-synthesis.md (ENHANCED)
    ├── Raw findings (existing)
    ├── Detected Interactions (NEW — programmatic matches)
    └── Unmatched Findings (NEW — for novel detection)
    ↓ read by
staff-bioinformatician (wave 1)
    ↓ VERIFIES detected interactions
    ↓ scans unmatched findings for NOVEL interactions
stdout_staff-bioinformatician.json
```

### File Locations

| File | Location | Purpose |
|------|----------|---------|
| `interaction-rules.json` | `~/.claude/schemas/teams/interaction-rules.json` | Declarative interaction patterns |
| `goyoke-team-prepare-synthesis` | `cmd/goyoke-team-prepare-synthesis/main.go` | Inter-wave Go binary |
| `pre-synthesis.md` | `{team_dir}/pre-synthesis.md` | Enhanced output with detected interactions |

---

## 3. Interaction Rules Schema

### `interaction-rules.json`

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "version": "1.0.0",
  "description": "Cross-domain interaction detection rules for goyoke-team-prepare-synthesis",
  "rules": [
    {
      "id": "fdr-chain-inflation",
      "name": "FDR Chain Inflation from Custom Database",
      "algebra": "multiplicative",
      "severity_override": "critical",
      "description": "Database inflation detected by proteogenomics-reviewer combined with standard FDR methodology from proteomics-reviewer. Nominal 1% FDR becomes 3-5x actual for variant peptide class.",
      "condition": {
        "type": "requires_all",
        "matchers": [
          {
            "reviewer_pattern": "proteogenomics-reviewer",
            "sharp_edge_pattern": "proteogenomics-db-*",
            "severity_minimum": "warning"
          },
          {
            "reviewer_pattern": "proteomics-reviewer",
            "sharp_edge_pattern": "proteomics-fdr-*",
            "severity_minimum": "warning"
          }
        ]
      },
      "message_template": "Database inflation ({finding_a.sharp_edge_id} from {finding_a.reviewer}) combined with {finding_b.sharp_edge_id} from {finding_b.reviewer} — actual variant-class FDR may be {algebra_factor}x nominal. See Nesvizhskii 2014 framework for class-specific FDR.",
      "layer": 3,
      "tags": ["fdr", "search-space", "statistical-coherence"]
    },
    {
      "id": "version-coherence-break",
      "name": "Cross-Stage Version/Reference Mismatch",
      "algebra": "gating",
      "severity_override": "critical",
      "description": "Version or reference inconsistency detected across pipeline stages. Invalidates all downstream findings.",
      "condition": {
        "type": "requires_any",
        "matchers": [
          {
            "reviewer_pattern": "genomics-reviewer",
            "sharp_edge_pattern": "genomics-ref-*"
          },
          {
            "reviewer_pattern": "proteogenomics-reviewer",
            "sharp_edge_pattern": "proteogenomics-version-*"
          }
        ]
      },
      "message_template": "Version/reference inconsistency detected: {matched_findings}. All downstream identification and quantification findings may be invalidated.",
      "layer": 2,
      "tags": ["version", "reference", "gating"]
    },
    {
      "id": "spectral-quality-gates-identification",
      "name": "Spectral Quality Gating Identification Chain",
      "algebra": "gating",
      "severity_override": "warning",
      "description": "Mass-spec-reviewer flagged spectral quality issues. Proteomics-reviewer identification findings may be invalid if spectral input is compromised.",
      "condition": {
        "type": "requires_all",
        "matchers": [
          {
            "reviewer_pattern": "mass-spec-reviewer",
            "sharp_edge_pattern": "mass-spec-*",
            "severity_minimum": "critical"
          },
          {
            "reviewer_pattern": "proteomics-reviewer",
            "finding_present": true
          }
        ]
      },
      "message_template": "Spectral quality concern ({finding_a.sharp_edge_id}) may invalidate proteomics identification findings. Verify spectral quality before trusting identification results.",
      "layer": 1,
      "tags": ["spectral-quality", "gating", "identification"]
    },
    {
      "id": "normalization-mismatch-chain",
      "name": "Upstream Processing Invalidates Quantification",
      "algebra": "multiplicative",
      "severity_override": "warning",
      "description": "TMT ratio compression or normalization issue from proteomics-reviewer compounds with quantification-dependent downstream analysis.",
      "condition": {
        "type": "requires_all",
        "matchers": [
          {
            "reviewer_pattern": "proteomics-reviewer",
            "sharp_edge_pattern": "proteomics-quant-*",
            "severity_minimum": "warning"
          },
          {
            "reviewer_pattern": "proteomics-reviewer",
            "sharp_edge_pattern": "proteomics-stats-*",
            "severity_minimum": "info"
          }
        ]
      },
      "message_template": "Quantification issue ({finding_a.sharp_edge_id}) propagates to statistical testing ({finding_b.sharp_edge_id}). Fold changes and p-values may both be affected.",
      "layer": 3,
      "tags": ["quantification", "statistics", "chain"]
    },
    {
      "id": "header-format-protein-inference",
      "name": "FASTA Header Incompatibility Breaking Protein Inference",
      "algebra": "multiplicative",
      "severity_override": "critical",
      "description": "Custom FASTA headers from proteogenomics pipeline incompatible with search engine protein ID parsing. PSMs work but protein grouping fails.",
      "condition": {
        "type": "requires_all",
        "matchers": [
          {
            "reviewer_pattern": "proteogenomics-reviewer",
            "sharp_edge_pattern": "proteogenomics-fasta-*"
          },
          {
            "reviewer_pattern": "proteomics-reviewer",
            "sharp_edge_pattern": "proteomics-search-header-*"
          }
        ]
      },
      "message_template": "FASTA header format ({finding_a.sharp_edge_id}) incompatible with search engine parsing ({finding_b.sharp_edge_id}). PSMs match correctly but protein-level FDR and inference produce garbage.",
      "layer": 1,
      "tags": ["fasta", "header", "protein-inference"]
    },
    {
      "id": "variant-normalization-cascade",
      "name": "Missing Variant Normalization Cascading Through Pipeline",
      "algebra": "additive",
      "severity_override": "critical",
      "description": "Unnormalized VCF variants produce duplicate protein entries, inflating search database and degrading FDR.",
      "condition": {
        "type": "requires_all",
        "matchers": [
          {
            "reviewer_pattern": "genomics-reviewer",
            "sharp_edge_pattern": "genomics-vc-no-normalization"
          },
          {
            "reviewer_pattern": "proteogenomics-reviewer",
            "sharp_edge_pattern": "proteogenomics-fasta-dedup*"
          }
        ]
      },
      "message_template": "Missing variant normalization ({finding_a.reviewer}) causes duplicate protein entries ({finding_b.reviewer}), inflating search database. Root cause is upstream, not in protein generation.",
      "layer": 4,
      "tags": ["normalization", "cascade", "dedup"]
    },
    {
      "id": "mbr-no-spectral-in-diffex",
      "name": "MBR Proteins in Differential Expression",
      "algebra": "additive",
      "severity_override": "warning",
      "description": "MBR-transferred identifications with no spectral evidence included in statistical testing as if they were real measurements.",
      "condition": {
        "type": "requires_all",
        "matchers": [
          {
            "reviewer_pattern": "proteomics-reviewer",
            "sharp_edge_pattern": "proteomics-mbr-*"
          },
          {
            "reviewer_pattern": "proteomics-reviewer",
            "sharp_edge_pattern": "proteomics-stats-*",
            "severity_minimum": "info"
          }
        ]
      },
      "message_template": "MBR-transferred identifications ({finding_a.sharp_edge_id}) feeding into statistical testing. MBR-only proteins have no spectral evidence — differential expression results for these proteins are unreliable.",
      "layer": 3,
      "tags": ["mbr", "statistics", "identification-quantification-crossover"]
    },
    {
      "id": "multistage-fdr-with-custom-db",
      "name": "Multi-Stage Search FDR with Custom Database Construction",
      "algebra": "multiplicative",
      "severity_override": "critical",
      "description": "Multi-stage/iterative search where later stages construct custom variant databases. FDR from each stage is dependent, not independent.",
      "condition": {
        "type": "requires_all",
        "matchers": [
          {
            "reviewer_pattern": "proteomics-reviewer",
            "sharp_edge_pattern": "proteomics-fdr-multistage-*"
          },
          {
            "reviewer_pattern": "proteogenomics-reviewer",
            "sharp_edge_pattern": "proteogenomics-db-*"
          }
        ]
      },
      "message_template": "Multi-stage search ({finding_a.sharp_edge_id}) constructs custom variant database ({finding_b.sharp_edge_id}). Stage FDRs are dependent — overall FDR is NOT max(stage_FDRs). Actual FDR may be 3-5x nominal.",
      "layer": 3,
      "tags": ["multistage", "fdr", "custom-db"]
    },
    {
      "id": "dia-library-from-wrong-system",
      "name": "DIA Spectral Library Instrument Mismatch",
      "algebra": "multiplicative",
      "severity_override": "warning",
      "description": "DIA spectral library generated from different instrument or chromatography than current experiment. RT predictions miscalibrated.",
      "condition": {
        "type": "requires_all",
        "matchers": [
          {
            "reviewer_pattern": "mass-spec-reviewer",
            "finding_category": "acquisition",
            "finding_contains": "DIA"
          },
          {
            "reviewer_pattern": "proteomics-reviewer",
            "sharp_edge_pattern": "proteomics-dia-library-*"
          }
        ]
      },
      "message_template": "DIA acquisition detected ({finding_a.reviewer}) with library provenance concern ({finding_b.sharp_edge_id}). RT predictions may be miscalibrated for current chromatographic setup.",
      "layer": 1,
      "tags": ["dia", "library", "instrument"]
    },
    {
      "id": "pipeline-no-containers-version-drift",
      "name": "No Container Pinning Enables Silent Version Drift",
      "algebra": "additive",
      "severity_override": "warning",
      "description": "Bioinformatician-reviewer flags no container pinning. Combined with any version-sensitive finding from domain reviewers, this means the exact analysis cannot be reproduced.",
      "condition": {
        "type": "requires_all",
        "matchers": [
          {
            "reviewer_pattern": "bioinformatician-reviewer",
            "finding_category": "reproducibility"
          },
          {
            "reviewer_pattern": "*",
            "sharp_edge_pattern": "*-version-*"
          }
        ]
      },
      "message_template": "No container/environment pinning ({finding_a.reviewer}) combined with version-dependent finding ({finding_b.sharp_edge_id}). Pipeline results may not be reproducible — tool version drift could change results silently.",
      "layer": 2,
      "tags": ["reproducibility", "version", "containers"]
    }
  ]
}
```

---

## 4. Matching Algorithm

### 4.1 Finding Extraction

Parse each wave 0 reviewer's stdout JSON. Extract findings into a normalized structure:

```go
type ExtractedFinding struct {
    ReviewerID   string   // e.g., "proteomics-reviewer"
    FindingID    string   // e.g., "PROT-13"
    SharpEdgeID  string   // e.g., "proteomics-fdr-global-only"
    Severity     string   // "critical", "warning", "info"
    Category     string   // e.g., "fdr-control"
    File         string   // file path
    Line         int      // line number
    Title        string   // brief title
    Message      string   // full description
}
```

Findings without `sharp_edge_id` are included in the unmatched pool but cannot trigger sharp-edge-based rules. They CAN trigger `finding_category` or `finding_contains` matchers.

### 4.2 Pattern Matching

For each rule in `interaction-rules.json`:

```
MATCH rule against extracted findings:

  IF rule.condition.type == "requires_all":
    For each matcher in rule.condition.matchers:
      Find ANY finding that satisfies ALL matcher fields:
        - reviewer_pattern: glob match against finding.ReviewerID
        - sharp_edge_pattern: glob match against finding.SharpEdgeID
        - severity_minimum: finding.Severity >= matcher.severity_minimum
        - finding_category: finding.Category contains matcher.finding_category
        - finding_contains: finding.Message contains matcher.finding_contains
      If no finding matches this matcher → rule does NOT fire
    If ALL matchers satisfied → rule FIRES with matched findings

  IF rule.condition.type == "requires_any":
    For each matcher in rule.condition.matchers:
      Find ANY finding that satisfies ALL matcher fields
      If ANY matcher satisfied → rule FIRES with matched finding(s)
```

### 4.3 Glob Matching

Sharp edge ID patterns use standard glob syntax:
- `proteomics-fdr-*` matches `proteomics-fdr-global-only`, `proteomics-fdr-multistage-dependent`, etc.
- `*-version-*` matches any reviewer's version-related sharp edge
- `proteogenomics-db-*` matches `proteogenomics-db-inflation`, `proteogenomics-db-decoy-strategy`, etc.

Reviewer patterns:
- `proteomics-reviewer` matches exactly
- `*` matches any reviewer

### 4.4 Severity Comparison

Severity ordering for `severity_minimum` filter:

```
info < warning < critical
```

`severity_minimum: "warning"` matches findings with severity "warning" OR "critical".

### 4.5 Algebra Types

The `algebra` field defines how matched findings interact for severity reclassification:

| Algebra | Meaning | Example |
|---------|---------|---------|
| `additive` | Two independent findings compound — severity escalates by 1 level | Two warnings from different reviewers → staff-bio may reclassify as critical |
| `multiplicative` | Upstream finding multiplies downstream impact — severity is max(findings) or override | DB inflation × standard FDR = actual FDR much worse than either finding alone |
| `negating` | One finding mitigates another — severity may decrease | Proteomics flags MBR risk, but bioinformatician confirms MBR proteins excluded from DE |
| `gating` | Upstream finding invalidates downstream findings entirely | Build mismatch → all identification and quantification findings are suspect |

The `severity_override` field in the rule takes precedence when set. Otherwise, algebra determines the reclassification.

---

## 5. Output Format

### 5.1 Enhanced pre-synthesis.md

The existing `pre-synthesis.md` content is preserved. Three new sections are appended:

```markdown
---

## Programmatic Cross-Domain Interaction Detection

> Generated by goyoke-team-prepare-synthesis v{version} using interaction-rules.json v{rules_version}.
> {n_rules} rules evaluated against {n_findings} findings from {n_reviewers} reviewers.
> {n_matched} interactions detected, {n_unmatched} findings unmatched.

### Detected Interactions

#### CRITICAL: fdr-chain-inflation (multiplicative)

**Rule:** FDR Chain Inflation from Custom Database

| Source | Reviewer | Sharp Edge | Severity | Finding |
|--------|----------|-----------|----------|---------|
| A | proteogenomics-reviewer | proteogenomics-db-inflation | warning | Search database 4.7x larger than reference proteome |
| B | proteomics-reviewer | proteomics-fdr-global-only | warning | Only PSM-level FDR applied at 1% |

**Interaction:** Database inflation (proteogenomics-db-inflation) combined with proteomics-fdr-global-only — actual variant-class FDR may be 4.7x nominal.

**Algebra:** multiplicative — upstream database size multiplies downstream FDR impact.

**Staff-bioinformatician action:** VERIFY this interaction. Check whether the FDR was computed on the full inflated DB or on separated classes. If global FDR on inflated DB → confirm CRITICAL. If class-specific FDR applied → this interaction may be negated.

---

#### WARNING: spectral-quality-gates-identification (gating)

[... same format ...]

---

### Unmatched Findings

These findings did not trigger any programmatic interaction rule. The staff-bioinformatician should evaluate them for NOVEL cross-domain interactions not yet codified in the rules.

| Reviewer | Sharp Edge | Severity | Finding |
|----------|-----------|----------|---------|
| proteomics-reviewer | proteomics-quant-tmt-no-compression | warning | TMT ratio compression not acknowledged |
| bioinformatician-reviewer | (none) | info | No random seed set for stochastic processes |

### Interaction Detection Summary

| Metric | Value |
|--------|-------|
| Rules evaluated | 10 |
| Interactions detected | 3 |
| Critical interactions | 1 |
| Warning interactions | 2 |
| Findings matched | 6 of 24 |
| Findings unmatched | 18 |
| Reviewers with interactions | 3 of 5 |
```

### 5.2 Machine-Readable Sidecar

In addition to the markdown in pre-synthesis.md, produce a JSON sidecar file `{team_dir}/detected-interactions.json`:

```json
{
  "version": "1.0.0",
  "rules_version": "1.0.0",
  "timestamp": "2026-04-13T14:30:00Z",
  "summary": {
    "rules_evaluated": 10,
    "interactions_detected": 3,
    "findings_total": 24,
    "findings_matched": 6,
    "findings_unmatched": 18,
    "reviewers_with_findings": 5,
    "reviewers_with_interactions": 3
  },
  "detected_interactions": [
    {
      "rule_id": "fdr-chain-inflation",
      "algebra": "multiplicative",
      "severity": "critical",
      "matched_findings": [
        {
          "reviewer": "proteogenomics-reviewer",
          "sharp_edge_id": "proteogenomics-db-inflation",
          "original_severity": "warning",
          "finding_id": "PG-24"
        },
        {
          "reviewer": "proteomics-reviewer",
          "sharp_edge_id": "proteomics-fdr-global-only",
          "original_severity": "warning",
          "finding_id": "PROT-13"
        }
      ],
      "message": "Database inflation combined with standard FDR..."
    }
  ],
  "unmatched_findings": [
    {
      "reviewer": "proteomics-reviewer",
      "sharp_edge_id": "proteomics-quant-tmt-no-compression",
      "severity": "warning",
      "finding_id": "PROT-30"
    }
  ]
}
```

This sidecar is consumed by the staff-bioinformatician's stdin (add `detected_interactions_path` field to the stdin schema) and can also be used for telemetry/ML analysis.

---

## 6. Implementation Plan

### 6.1 New Files

| File | Purpose |
|------|---------|
| `~/.claude/schemas/teams/interaction-rules.json` | Declarative interaction patterns |
| `cmd/goyoke-team-prepare-synthesis/interactions.go` | Interaction detection logic |
| `cmd/goyoke-team-prepare-synthesis/interactions_test.go` | Unit tests for matching |
| `cmd/goyoke-team-prepare-synthesis/testdata/interaction-rules.json` | Test fixture rules |
| `cmd/goyoke-team-prepare-synthesis/testdata/mock-wave0/` | Mock reviewer outputs for integration tests |
| `cmd/goyoke-team-prepare-synthesis/testdata/golden/` | Expected pre-synthesis.md outputs |

### 6.2 Modified Files

| File | Change |
|------|--------|
| `cmd/goyoke-team-prepare-synthesis/main.go` | Load interaction-rules.json, run detection after finding extraction, append to pre-synthesis.md |
| `~/.claude/schemas/teams/stdin-stdout/review-bioinformatics-staff-bioinformatician.json` | Add `detected_interactions_path` field to stdin |
| `~/.claude/agents/staff-bioinformatician/staff-bioinformatician.md` | Update Layer 4 to reference programmatic detection, simplify from "discover" to "verify + novel detect" |

### 6.3 Implementation Order

1. **`interactions.go`** — Core matching engine (types, pattern matching, rule loading)
2. **`interactions_test.go`** — Unit tests (see §7)
3. **`interaction-rules.json`** — Initial 10 rules (from §3 examples)
4. **`main.go` integration** — Load rules, extract findings, detect, append to pre-synthesis.md
5. **Mock data + golden tests** — Integration tests with realistic reviewer outputs
6. **Stdin schema update** — Add `detected_interactions_path`
7. **Agent update** — Simplify Layer 4 instructions

---

## 7. Testing Strategy

### 7.1 Unit Tests (`interactions_test.go`)

#### Pattern Matching Tests

| Test | Input | Expected |
|------|-------|----------|
| `TestGlobMatch_ExactID` | pattern `proteomics-fdr-global-only`, id `proteomics-fdr-global-only` | match |
| `TestGlobMatch_Wildcard` | pattern `proteomics-fdr-*`, id `proteomics-fdr-global-only` | match |
| `TestGlobMatch_DoubleWildcard` | pattern `*-version-*`, id `proteogenomics-version-vep-pyensembl` | match |
| `TestGlobMatch_NoMatch` | pattern `proteomics-fdr-*`, id `proteomics-quant-tmt-no-compression` | no match |
| `TestGlobMatch_ReviewerWildcard` | pattern `*`, reviewer `proteomics-reviewer` | match |
| `TestGlobMatch_EmptyID` | pattern `proteomics-*`, id `""` | no match |

#### Severity Comparison Tests

| Test | Minimum | Finding | Expected |
|------|---------|---------|----------|
| `TestSeverity_ExactMatch` | `warning` | `warning` | pass |
| `TestSeverity_HigherPasses` | `warning` | `critical` | pass |
| `TestSeverity_LowerFails` | `warning` | `info` | fail |
| `TestSeverity_NoMinimum` | `""` | `info` | pass |

#### Rule Matching Tests

| Test | Rule Type | Findings | Expected |
|------|-----------|----------|----------|
| `TestRequiresAll_BothPresent` | requires_all with 2 matchers | Both matching findings present | fires |
| `TestRequiresAll_OneMissing` | requires_all with 2 matchers | Only 1 matching finding | does not fire |
| `TestRequiresAll_SeverityBelowMin` | requires_all, severity_minimum=warning | Finding has severity=info | does not fire |
| `TestRequiresAny_OnePresent` | requires_any with 2 matchers | 1 matching finding | fires |
| `TestRequiresAny_NonePresent` | requires_any with 2 matchers | No matching findings | does not fire |
| `TestRequiresAll_MultipleMatchesFirstWins` | requires_all | 3 findings match matcher 1, 1 matches matcher 2 | fires with first matching finding from each matcher |

#### Edge Case Tests

| Test | Scenario | Expected |
|------|----------|----------|
| `TestEmptyFindings` | No findings from any reviewer | No interactions, all rules unevaluated |
| `TestEmptyRules` | No rules in config | No interactions detected, all findings unmatched |
| `TestFindingWithoutSharpEdge` | Finding has `sharp_edge_id: null` | Cannot match sharp_edge_pattern rules, but CAN match finding_category/finding_contains rules |
| `TestSameReviewerBothSides` | Rule requires findings from same reviewer | fires (e.g., proteomics MBR + proteomics stats) |
| `TestDuplicateSharpEdge` | Two findings with same sharp_edge_id from different reviewers | Both considered, first match used |
| `TestMalformedRulesFile` | Invalid JSON in interaction-rules.json | Graceful error, skip interaction detection, log warning, proceed with basic pre-synthesis |
| `TestMissingRulesFile` | interaction-rules.json not found | Skip interaction detection silently, produce standard pre-synthesis.md |

#### Algebra Tests

| Test | Algebra | Input Severities | Expected Reclassification |
|------|---------|-----------------|--------------------------|
| `TestAlgebra_Additive` | additive | warning + warning | critical (escalated) |
| `TestAlgebra_Multiplicative` | multiplicative | warning + info | warning (max) or severity_override |
| `TestAlgebra_Gating` | gating | critical + warning | critical (upstream gates downstream) |
| `TestAlgebra_Negating` | negating | warning + info(mitigating) | info (downgraded) |
| `TestAlgebra_SeverityOverride` | any, severity_override=critical | info + info | critical (override wins) |

### 7.2 Integration Tests (Mock Wave 0 Outputs)

#### Test Fixtures

Create mock wave 0 stdout files that represent realistic reviewer outputs:

**Fixture Set 1: VCF-VEP-to-fasta pipeline (standard)**

```
testdata/mock-wave0/fixture-vcf-vep/
  stdout_genomics-reviewer.json       ← 5 findings, includes genomics-ref-wrong-build
  stdout_proteogenomics-reviewer.json ← 8 findings, includes proteogenomics-version-vep-pyensembl
  stdout_proteomics-reviewer.json     ← 6 findings, includes proteomics-fdr-global-only
  stdout_bioinformatician-reviewer.json ← 3 findings, includes reproducibility concern
```

Expected interactions:
- `version-coherence-break` (gating) — genomics-ref + proteogenomics-version
- `fdr-chain-inflation` (multiplicative) — if DB inflation present

**Fixture Set 2: DIA proteomics pipeline (clean)**

```
testdata/mock-wave0/fixture-dia-clean/
  stdout_mass-spec-reviewer.json      ← 1 info finding
  stdout_proteomics-reviewer.json     ← 2 info findings
  stdout_bioinformatician-reviewer.json ← 1 info finding
```

Expected interactions: NONE (clean pipeline should trigger zero rules)

**Fixture Set 3: Multi-domain critical failure**

```
testdata/mock-wave0/fixture-critical/
  stdout_genomics-reviewer.json       ← no normalization finding
  stdout_proteogenomics-reviewer.json ← duplicate proteins + DB inflation
  stdout_proteomics-reviewer.json     ← global FDR only + MBR enabled
  stdout_mass-spec-reviewer.json      ← spectral quality critical
```

Expected interactions:
- `variant-normalization-cascade` (additive)
- `fdr-chain-inflation` (multiplicative)
- `spectral-quality-gates-identification` (gating)
- `mbr-no-spectral-in-diffex` (additive)

**Fixture Set 4: Partial reviewer failure**

```
testdata/mock-wave0/fixture-partial/
  stdout_genomics-reviewer.json       ← normal findings
  stdout_proteomics-reviewer.json     ← status: "failed"
```

Expected behavior: Skip failed reviewer, detect interactions only from available outputs, note incomplete coverage in summary.

#### Integration Test Cases

| Test | Fixture | Assertion |
|------|---------|-----------|
| `TestIntegration_VCFVEPPipeline` | fixture-vcf-vep | Detects version-coherence-break; pre-synthesis.md contains "Detected Interactions" section |
| `TestIntegration_CleanPipeline` | fixture-dia-clean | Zero interactions detected; pre-synthesis.md contains "No interactions detected" |
| `TestIntegration_CriticalFailure` | fixture-critical | Detects 4 interactions; severity_override applied correctly |
| `TestIntegration_PartialFailure` | fixture-partial | Gracefully handles failed reviewer; summary notes incomplete coverage |
| `TestIntegration_NoRulesFile` | Any fixture, missing rules file | Standard pre-synthesis.md produced without interaction section |

### 7.3 Golden File Tests

For each fixture set, maintain a golden `expected-pre-synthesis.md` and `expected-detected-interactions.json`. The test compares actual output against golden files.

```go
func TestGolden_VCFVEPPipeline(t *testing.T) {
    actual := runSynthesis(t, "testdata/mock-wave0/fixture-vcf-vep")
    golden := readGolden(t, "testdata/golden/vcf-vep-pre-synthesis.md")
    if diff := cmp.Diff(golden, actual); diff != "" {
        t.Errorf("pre-synthesis.md mismatch (-want +got):\n%s", diff)
    }
}
```

Golden files should be checked into version control. When rules change, golden files are regenerated and reviewed.

### 7.4 Compliance Tests

Verify that the programmatic detection produces the same results as the staff-bioinformatician would discover manually:

| Test | Method | Assertion |
|------|--------|-----------|
| `TestCompliance_KnownInteractionsDetected` | Run fixture-critical through both programmatic detection AND staff-bioinformatician agent | All programmatically-detected interactions also found by agent |
| `TestCompliance_NoFalsePositives` | Run fixture-dia-clean through programmatic detection | Zero interactions detected (clean pipeline should not trigger false alarms) |
| `TestCompliance_UnmatchedFindingsComplete` | Run any fixture | Every finding not in a detected interaction appears in unmatched list |

### 7.5 Regression Tests

After deploying interaction detection, capture the first 5 real `/review-bioinformatics` runs as regression fixtures. Compare programmatic detections against staff-bioinformatician's actual cross-domain findings. Track:

| Metric | Target |
|--------|--------|
| True positive rate (programmatic detects what agent found) | >90% for known patterns |
| False positive rate (programmatic detects what agent didn't find) | <5% |
| Novel detection rate (agent finds interactions not in rules) | Track for rule expansion |
| Thinking budget reduction | >30% compared to pre-programmatic runs |

---

## 8. Maintenance

### Adding New Rules

1. Identify the cross-domain pattern from a real review
2. Determine the sharp_edge_ids involved
3. Classify the algebra type
4. Add to `interaction-rules.json`
5. Create a test fixture that triggers the new rule
6. Add golden file for the fixture
7. Run full test suite

### Rule Deprecation

When a rule no longer applies (e.g., a reviewer merges a check into another):
1. Set `"deprecated": true` on the rule
2. Deprecated rules are logged but not matched
3. Remove after 2 release cycles

### Sharp Edge ID Evolution

When wave 0 reviewers add/rename/remove sharp edge IDs:
1. `interaction-rules.json` patterns should use wildcards where possible (`proteomics-fdr-*`)
2. Exact-match patterns must be updated when sharp edge IDs change
3. CI step: validate that all exact-match patterns in rules resolve to at least one sharp edge ID in current reviewer definitions

---

## 9. Performance

Expected performance characteristics:

| Metric | Expected | Constraint |
|--------|----------|-----------|
| Rule evaluation time | <100ms for 50 rules × 100 findings | Inter-wave script should not add perceptible latency |
| Memory | <10MB for rule matching | Findings fit comfortably in memory |
| File I/O | Read 6 stdout files + 1 rules file, write 2 output files | Standard filesystem operations |

No optimization needed — the matching is O(rules × findings × matchers), which is trivially fast for expected scales (10-50 rules, 10-100 findings, 2-4 matchers per rule).

---

## 10. Migration Path

### Phase 1 (Current — v1)

Static interaction map in staff-bioinformatician agent instructions. Agent discovers interactions from scratch.

### Phase 2 (This Spec — v2)

Programmatic detection in `goyoke-team-prepare-synthesis`. Agent verifies pre-computed interactions + discovers novel ones.

### Phase 3 (Future — v3)

ML-assisted interaction detection. Train a classifier on historical review data (detected interactions, staff-bioinformatician novel findings, user feedback) to propose new rules. Human reviews and promotes to `interaction-rules.json`.

---

## 11. Open Questions

1. **Should negating rules reduce severity or just annotate?** If a negating rule fires, should it modify the original finding's severity in the output, or just add a note that the severity may be mitigated?

2. **Should the staff-bioinformatician trust programmatic detections unconditionally?** Or should it always verify? Current spec requires verification, but this adds thinking budget cost. If programmatic detection has >99% precision, verification may be unnecessary for high-confidence rules.

3. **Should rules have a confidence score?** High-confidence rules (well-established interactions like FDR chain inflation) could skip verification, while low-confidence rules (newer, less tested patterns) require verification.

4. **CI validation:** Should a CI step validate that all sharp_edge_ids in interaction rules exist in current reviewer definitions? This would catch stale rules after reviewer evolution.
