# gogent-scout

Unified pre-routing reconnaissance binary with smart backend selection.

## Overview

`gogent-scout` provides scope assessment and routing recommendations before task delegation. It automatically selects between native Go scout (fast, basic metrics) and Gemini-backed scout (semantic analysis) based on project complexity.

## Features

- **Smart Backend Routing**: Automatically selects optimal backend based on scope
- **Multi-Factor Scoring**: Considers file count, lines, file sizes, and language diversity
- **Graceful Fallback**: Falls back through native → gemini → synthetic on failures
- **Always Valid JSON**: Guaranteed JSON output even on total failure
- **Atomic File Writes**: Safe concurrent execution with atomic output writes
- **Fast Native Scout**: <100ms for small projects (p50: ~40-50μs for 5 files)

## Usage

```bash
# Basic usage
gogent-scout <target> "<instruction>"

# With output file
gogent-scout ./pkg "Assess auth module" --output=.claude/tmp/scout_metrics.json

# Pipe file list
find src -name "*.go" | gogent-scout - "Analyze Go codebase"
```

## Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `SCOUT_BACKEND` | auto | Force backend: `native` or `gemini` |
| `SCOUT_THRESHOLD` | 40 | Routing score threshold (0-100) |
| `SCOUT_OUTPUT` | - | Output file path (alternative to --output) |

## Backend Selection

### Routing Score Calculation

Score is computed from:
- **File count** (40% weight): 20 files = max
- **Total lines** (30% weight): 5000 lines = max
- **Max file size** (20% weight): 1000 lines = max
- **Language count** (10% weight): 4 languages = max

### Backend Decision

| Score | Backend | Capabilities |
|-------|---------|-------------|
| < 40 | **Native** | File count, line count, language detection, test detection |
| ≥ 40 | **Gemini** | Import analysis, dependency graph, semantic complexity |

**Example scores:**
- 8 files, 2000 lines, max 400 lines, 1 language → Score 38 (native)
- 15 files, 3000 lines, max 500 lines, 2 languages → Score 63 (gemini)

## Output Schema

```json
{
  "schema_version": "1.0",
  "backend": "native|gemini|native_fallback|synthetic_fallback",
  "target": "/path/to/target",
  "timestamp": "2026-02-02T10:00:00Z",

  "scope_metrics": {
    "total_files": 15,
    "total_lines": 2340,
    "estimated_tokens": 23400,
    "languages": ["go", "md"],
    "file_types": {".go": 12, ".md": 3},
    "max_file_lines": 450,
    "files_over_500_lines": 0
  },

  "complexity_signals": {
    "available": false,
    "import_density": null,
    "cross_file_dependencies": null,
    "test_coverage_present": true,
    "note": "Semantic analysis unavailable - basic metrics only"
  },

  "routing_recommendation": {
    "recommended_tier": "sonnet",
    "confidence": "high",
    "reasoning": "15 files, 2340 lines, single language"
  },

  "key_files": [
    {"path": "pkg/routing/schema.go", "lines": 450, "relevance": "Largest file"}
  ],

  "warnings": []
}
```

## Fallback Chain

1. **Primary Backend**: Selected based on routing score
2. **Opposite Backend**: Tries alternative if primary fails
3. **Synthetic Report**: Conservative estimates if both fail (always succeeds)

## Examples

### Small Project (Native Scout)
```bash
$ gogent-scout ./cmd/gogent-scout "Assess implementation"
# Backend: native
# Score: 38 (8 files, 1353 lines)
# Recommendation: sonnet
# Latency: ~50ms
```

### Large Project (Gemini Scout)
```bash
$ gogent-scout ./cmd "Assess all binaries"
# Backend: gemini (falls back to native if unavailable)
# Score: 95 (41 files, 21454 lines)
# Recommendation: external
```

### Force Native Backend
```bash
$ SCOUT_BACKEND=native gogent-scout ./pkg/routing "Quick scan"
# Backend: native (forced)
# Skips routing score calculation
```

## Tier Recommendations

| Files | Lines | Tokens | Tier | Rationale |
|-------|-------|--------|------|-----------|
| < 5 | < 500 | < 5K | `haiku` | Small scope, mechanical work |
| ≤ 15 | < 2000 | < 20K | `sonnet` | Medium scope, reasoning required |
| > 15 | > 2000 | > 50K | `external` | Large scope, use gemini-slave mapper |

## Performance

**Benchmarks** (AMD Ryzen AI 7 350):
- 5 files: ~40-50μs (p50), <100μs (p99) ✅ **Target: <100ms**
- 15 files: ~117μs (p50)
- 37 files: ~150ms (native fallback after Gemini timeout)

## Testing

```bash
# Run tests
go test ./cmd/gogent-scout/ -v

# Run benchmarks
go test ./cmd/gogent-scout/ -bench=. -benchmem

# Coverage
go test ./cmd/gogent-scout/ -cover
```

## Integration

### From Router (CLAUDE.md)
```javascript
// Pattern 1: Scout → Route → Execute
[SCOUTING] Spawn gogent-scout
Bash({command: "gogent-scout ./pkg 'Assess scope'"})
// Read .claude/tmp/scout_metrics.json
// Route based on recommended_tier
```

### From /explore Skill
```bash
# Scout step automatically uses gogent-scout
gogent-scout "${TARGET_DIR}" "Pre-routing reconnaissance"
```

## Error Handling

- **Target not found**: Exit 2
- **Invalid arguments**: Exit 1, print usage
- **Backend failures**: Fallback chain, always exit 0 with valid JSON
- **Timeout**: 30s for Gemini delegate

## Implementation Details

- **Language Detection**: By file extension mapping
- **Test Detection**: Pattern matching (_test.go, test_*.py, etc.)
- **Vendor Skipping**: Ignores vendor/, node_modules/, hidden dirs
- **Atomic Writes**: Write to .tmp, then rename for safety

## Specification

Full specification: `.claude/tmp/gogent-scout-spec-v2.md`

## Related Components

- `pkg/telemetry/scout.go`: Scout recommendation logging
- `routing-schema.json`: Scout protocol configuration
- `agents-index.json`: gogent-scout agent definition
