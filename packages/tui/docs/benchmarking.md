# Performance Benchmarking Guide

## Quick Start

```bash
# Build the TUI
npm run build

# Run benchmark
npm run benchmark
```

## What Gets Measured

| Metric | Method | Target | Acceptable |
|--------|--------|--------|------------|
| Cold start | Median of 5 `node dist/index.js` runs | < 500ms | < 800ms |
| Memory (idle) | RSS after 2s startup | < 80MB | < 120MB |
| Memory (active) | RSS after simulated activity | < 100MB | < 150MB |
| Input latency | Event loop response time | < 32ms | < 50ms |

## Output Files

- **Raw data**: `.claude/tmp/ts-benchmark.json`
- **Report**: `docs/performance-report.md`

## Interpreting Results

### Status Indicators

- ✅ **Pass**: Within target range
- ⚠️ **Target Exceeded**: Exceeds target but acceptable
- ❌ **Unacceptable**: Rollback trigger

### Comparison with Go Baseline

The benchmark compares TypeScript implementation against the Go baseline from TUI-002:

- **Cold start**: Node.js is ~40x slower (expected - interpreted vs compiled)
- **Memory**: Node.js uses more memory (expected - V8 overhead)
- **Input latency**: Node.js is ~3x slower (acceptable for event-driven UI)

## Known Limitations

1. **--version flag**: Not implemented, so cold start measures basic node startup
2. **TTY requirement**: Memory measurements use estimates when TUI can't allocate TTY
3. **Platform-specific**: Results vary by system load and Node.js version

## Troubleshooting

### "dist/index.js not found"

```bash
npm run build
```

### "Go baseline not found"

The benchmark works without the baseline but won't show comparisons. To generate:

```bash
cd ../..  # Go to repo root
# Run TUI-002 benchmark if available
```

### Exit code 1

If any metric exceeds **acceptable** threshold, benchmark exits with code 1 to trigger rollback evaluation.

## Continuous Monitoring

Run benchmark:
- After major refactoring
- Before releases
- When investigating performance regressions

Track trends in `.claude/tmp/ts-benchmark.json` over time.

---

*See also: [Performance Report](./performance-report.md)*
