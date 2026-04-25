# Audit System Quick Start Guide

Get automated testing running in 3 minutes.

---

## 1. Create Configuration (30 seconds)

Copy the example config for your language to your project root:

**Go Project**:
```bash
cp ~/.claude/skills/ticket/examples/ticket-config-go.json .ticket-config.json
```

**Python Project**:
```bash
cp ~/.claude/skills/ticket/examples/ticket-config-python.json .ticket-config.json
```

**R Project**:
```bash
cp ~/.claude/skills/ticket/examples/ticket-config-r.json .ticket-config.json
```

**JavaScript/TypeScript Project**:
```bash
cp ~/.claude/skills/ticket/examples/ticket-config-javascript.json .ticket-config.json
```

---

## 2. Verify Setup (1 minute)

Check that your project has test infrastructure:

**Go**:
```bash
ls go.mod                    # Should exist
ls *_test.go                 # Should have test files
go test -v ./...             # Should run successfully
```

**Python**:
```bash
ls pyproject.toml            # Should exist (or setup.py)
ls tests/                    # Should have test directory
uv run pytest tests/ -v      # Should run successfully
```

**R**:
```bash
ls DESCRIPTION               # Should exist
ls tests/testthat/           # Should have test directory
Rscript -e "devtools::test()" # Should run successfully
```

**JavaScript**:
```bash
ls package.json              # Should exist
npm test                     # Should run successfully
```

If any tests fail, fix test infrastructure before enabling audit.

---

## 3. Enable Audit (30 seconds)

Edit `.ticket-config.json` and set `enabled: true`:

```json
{
  "tickets_dir": "migration_plan/tickets",
  "audit_config": {
    "enabled": true
  }
}
```

Save and commit.

---

## 4. Run First Audit (1 minute)

When completing a ticket, the audit runs automatically:

```bash
$ /ticket verify
[ticket] Acceptance criteria: 5/5 complete ✓
[ticket] Running audit documentation...
[INFO] Starting audit for ticket: FEAT-001
[INFO] Detected language: go
[INFO] Phase 2: Executing tests...
[PASS] Unit tests passed
[PASS] Race detector passed
[INFO] Total coverage: 85.2%
[INFO] Phase 3: Generating implementation summary...
[ticket] ✓ Audit complete
```

Results saved to `.ticket-audits/FEAT-001/`:
- `unit-tests.log` — Test output
- `race-detector.log` — Concurrency checks (Go)
- `coverage-report.txt` — Coverage analysis
- `implementation-summary.md` — Human-readable summary

---

## Key Concepts

### Placeholders

Test commands support runtime substitution:

| Placeholder | Becomes |
|-------------|---------|
| `{audit_dir}` | `.ticket-audits/{ticket_id}` |
| `{ticket_id}` | Current ticket ID (e.g., `FEAT-001`) |
| `{project_root}` | Project root directory path |

Example:
```json
{
  "go": {
    "coverage": "go test -coverprofile={audit_dir}/coverage.out ./..."
  }
}
```

### Non-Blocking

Audit failures do NOT prevent ticket completion. If tests fail:

```
[FAIL] Unit tests failed (see .ticket-audits/FEAT-001/unit-tests.log)
[WARNING] Audit failed but completing ticket anyway
```

You can still mark ticket complete. Fix tests in next iteration.

### Language Detection

No manual configuration needed. System detects language from:
- Go: `go.mod` file
- Python: `pyproject.toml` or `setup.py`
- R: `DESCRIPTION` file
- JavaScript: `package.json` file

---

## Customization

### Override Default Commands

The examples use defaults. To customize:

```json
{
  "go": {
    "unit": "go test -v -timeout=30s ./...",
    "race": "go test -race -count=5 ./...",
    "coverage": "go test -coverprofile={audit_dir}/coverage.out -count=1 ./..."
  }
}
```

Common customizations:
- Add `-timeout=Xs` for slow tests
- Add `-count=5` for flaky test detection
- Add test tags: `-tags=integration`
- Change coverage package: `--cov=mypackage`

### Add Integration Tests (Go)

```json
{
  "go": {
    "integration": "go test -v -tags=integration ./tests/integration"
  }
}
```

### Add Coverage (R)

```json
{
  "r": {
    "coverage": "Rscript -e \"covr::report(covr::package_coverage())\""
  }
}
```

See `~/.claude/skills/ticket/docs/audit-config-schema.md` for complete reference.

---

## Troubleshooting

### "Error: Could not detect project language"

**Cause**: Missing language indicator file

**Fix**: Add required file:
- Go: `go mod init github.com/user/project`
- Python: Create `pyproject.toml` or `setup.py`
- R: Create `DESCRIPTION`
- JavaScript: Create `package.json`

### "Unit tests failed"

**Cause**: Test infrastructure missing or tests broken

**Fix**:
1. Run tests manually: `go test -v ./...` (or equivalent)
2. Fix test failures first
3. Then re-run `/ticket`

### "Audit disabled in config"

**Cause**: `enabled: false` in `.ticket-config.json`

**Fix**: Change to `enabled: true`

### "No `.ticket-config.json` found"

**Cause**: Config file missing

**Fix**: Copy example config for your language (see Step 1 above)

---

## What Happens When Audit Runs

1. **Phase 1**: Create `.ticket-audits/{ticket_id}/` directory
2. **Phase 2**: Execute tests (language-specific)
   - Go: unit, integration, race detection, coverage
   - Python: pytest with coverage
   - R: testthat with optional coverage
   - JavaScript: npm test with optional coverage
3. **Phase 3**: Generate human-readable summary
   - Extract test results
   - Format coverage reports
   - Create `implementation-summary.md`

All failures are non-blocking. Audit never prevents ticket completion.

---

## Output Examples

### Successful Go Audit

```
.ticket-audits/FEAT-001/
├── unit-tests.log           # 15 tests passed
├── race-detector.log        # No races detected
├── coverage.out             # Binary format
├── coverage-report.txt      # Per-function coverage
├── coverage-summary.txt     # "85.2%"
└── implementation-summary.md # Full summary
```

### Python with Coverage

```
.ticket-audits/FEAT-002/
├── pytest-output.log        # 23 tests passed, 89% coverage
├── coverage-summary.txt     # Coverage line extracted
└── implementation-summary.md # Full summary
```

---

## Next Steps

1. Copy config for your language
2. Verify tests pass manually
3. Run `/ticket verify` on next ticket
4. Check `.ticket-audits/` results
5. Optionally customize commands (see Customization section)

---

**Learn More**: See `~/.claude/skills/ticket/docs/audit-config-schema.md` for complete reference.
