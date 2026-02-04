# Performance Benchmark Usage Guide

## Quick Start

### 1. Build the TUI

```bash
cd packages/tui
npm run build
```

### 2. Run Performance Benchmarks

```bash
npm run test:performance
```

This will:
- Spawn 5 process instances to measure cold start (median)
- Measure memory usage (idle and active states)
- Measure input latency (event loop responsiveness)
- Generate JSON output at `.claude/tmp/ts-benchmark-vitest.json`
- Run assertions against cutover targets

### 3. Review Results

```bash
cat ../../.claude/tmp/ts-benchmark-vitest.json
```

Example output:

```json
{
  "measured_at": "2026-02-04T12:34:56.789Z",
  "cold_start_ms": 450.23,
  "memory_idle_mb": 75.4,
  "memory_active_mb": 85.6,
  "input_latency_ms": 28.1,
  "passes_target": {
    "cold_start": true,
    "memory_idle": true,
    "memory_active": true,
    "input_latency": true
  },
  "passes_acceptable": {
    "cold_start": true,
    "memory_idle": true,
    "memory_active": true,
    "input_latency": true
  }
}
```

## Filling Out Cutover Sign-Off

### Step 1: Extract Values from JSON

```bash
# Use jq to extract values
cat ../../.claude/tmp/ts-benchmark-vitest.json | jq '{
  cold_start: .cold_start_ms,
  memory_idle: .memory_idle_mb,
  memory_active: .memory_active_mb,
  input_latency: .input_latency_ms
}'
```

### Step 2: Fill Section 2.5 of `cutover-signoff.md`

Location: `packages/tui/docs/cutover-signoff.md`

Find section "### Memory Profiling" and fill in:

```markdown
| Metric | Measured | Target | Status |
|--------|----------|--------|--------|
| Memory idle | 75.4MB | <80MB | ✅ |
| Memory active | 85.6MB | <100MB | ✅ |
| Cold start | 450.23ms | <500ms | ✅ |
| Input latency | 28.1ms | <32ms | ✅ |
```

### Step 3: Check Pass/Fail Status

All metrics must pass `passes_acceptable` for cutover approval:

```bash
# Check if all acceptable thresholds pass
cat ../../.claude/tmp/ts-benchmark-vitest.json | jq '.passes_acceptable | all'
```

Expected output: `true`

If any value is `false`, investigate before proceeding with cutover.

## Interpreting Results

### Cold Start Time

**What it measures:** Time from `node dist/index.js` execution to process ready.

**Target:** <500ms
**Acceptable:** <800ms
**Rollback trigger:** >800ms

If exceeds target:
- Check bundle size (`ls -lh dist/index.js`)
- Profile with `node --prof dist/index.js --version`
- Consider lazy loading modules

### Memory Idle

**What it measures:** RSS memory after 2s of idle startup.

**Target:** <80MB
**Acceptable:** <120MB
**Rollback trigger:** >120MB

If exceeds target:
- Profile with `node --inspect dist/index.js`
- Check for unnecessary dependencies in bundle
- Review store initialization size

### Memory Active

**What it measures:** RSS memory after simulated user activity (keypresses).

**Target:** <100MB
**Acceptable:** <150MB
**Rollback trigger:** >150MB

If exceeds target:
- Check for memory leaks (run sustained operation test)
- Profile with heap snapshots
- Review message history retention

### Input Latency

**What it measures:** Event loop response time for keypresses.

**Target:** <32ms
**Acceptable:** <50ms
**Rollback trigger:** >50ms

If exceeds target:
- Profile event loop blocking
- Check for synchronous I/O in render path
- Review React component render complexity

## CI/CD Integration

### GitHub Actions

```yaml
name: Performance Validation

on: [pull_request]

jobs:
  performance:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Node
        uses: actions/setup-node@v3
        with:
          node-version: '22'

      - name: Install dependencies
        run: npm ci
        working-directory: packages/tui

      - name: Build TUI
        run: npm run build
        working-directory: packages/tui

      - name: Run performance benchmarks
        run: npm run test:performance
        working-directory: packages/tui

      - name: Upload results
        uses: actions/upload-artifact@v3
        with:
          name: performance-results
          path: .claude/tmp/ts-benchmark-vitest.json

      - name: Check for rollback triggers
        run: |
          PASSES=$(cat .claude/tmp/ts-benchmark-vitest.json | jq '.passes_acceptable | all')
          if [ "$PASSES" != "true" ]; then
            echo "❌ Performance metrics exceed acceptable thresholds"
            exit 1
          fi
```

## Troubleshooting

### "dist/index.js not found"

```bash
npm run build
```

### Tests timeout

Increase timeout in `vitest.config.ts`:

```typescript
export default defineConfig({
  test: {
    testTimeout: 180000, // 3 minutes
  },
});
```

### Memory measurements fail on macOS/Linux

Ensure `ps` command is available:

```bash
which ps  # Should return /bin/ps or similar
```

### High variance between runs

Run benchmarks multiple times to verify stability:

```bash
for i in {1..3}; do
  echo "Run $i:"
  npm run test:performance
  cat ../../.claude/tmp/ts-benchmark-vitest.json | jq '.cold_start_ms'
done
```

Variance should be <10% between runs.

### Skip in development

```bash
# Skip expensive performance tests during development
SKIP_PERF=1 npm test
```

## Comparison with `npm run benchmark`

Two benchmark tools exist:

| Tool | Command | Output | Use Case |
|------|---------|--------|----------|
| `benchmark.ts` | `npm run benchmark` | Markdown report + JSON | Manual analysis, detailed reports |
| `benchmarks.test.ts` | `npm run test:performance` | Test assertions + JSON | CI/CD, cutover validation |

**Recommendation:** Use `test:performance` for cutover sign-off (has assertions against targets).

## Example: Complete Cutover Workflow

```bash
# 1. Build
npm run build

# 2. Run benchmarks
npm run test:performance

# 3. Check all pass
cat ../../.claude/tmp/ts-benchmark-vitest.json | jq '.passes_acceptable'

# 4. Extract values for sign-off
cat ../../.claude/tmp/ts-benchmark-vitest.json | jq '{
  cold_start: .cold_start_ms,
  memory_idle: .memory_idle_mb,
  memory_active: .memory_active_mb,
  input_latency: .input_latency_ms,
  measured_at: .measured_at
}'

# 5. Fill cutover-signoff.md Section 2.5

# 6. If all pass, proceed with cutover
```

## Reference

- **Cutover Checklist:** `packages/tui/docs/cutover-signoff.md`
- **Test Documentation:** `tests/performance/README.md`
- **Targets Source:** TUI-020 ticket (Performance Benchmarking)
