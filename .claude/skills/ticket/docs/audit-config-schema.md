# Audit Configuration Schema Reference

This document provides complete technical reference for configuring automated audit testing in the ticket workflow system.

**Latest Update**: 2026-01-18
**Applies to**: run-audit.sh v1.0+
**Status**: Production

---

## Quick Start

Add this to your project root `.ticket-config.json`:

```json
{
  "tickets_dir": "migration_plan/tickets",
  "audit_config": {
    "enabled": true,
    "test_commands": {
      "go": {
        "unit": "go test -v ./...",
        "race": "go test -race ./..."
      }
    }
  }
}
```

Then enable it when running `/ticket`:

```
[ticket] Running audit documentation...
[INFO] Audit complete
```

---

## Complete Schema Reference

### Root Object

```json
{
  "tickets_dir": "string",
  "project_name": "string (optional)",
  "audit_config": {
    "enabled": boolean,
    "test_commands": { ... }
  }
}
```

### audit_config Object

**Type**: Object
**Required**: Yes (if audit is needed)
**Properties**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `enabled` | boolean | Yes | Enable/disable audit system. Default: `false` |
| `test_commands` | object | No | Language-specific test command configurations |

---

### test_commands Structure

Language-specific test configurations. Only include the language(s) used in your project.

#### Go Test Commands

**Object Path**: `audit_config.test_commands.go`

```json
{
  "unit": "go test -v ./...",
  "integration": "go test -v -tags=integration ./...",
  "race": "go test -race ./...",
  "coverage": "go test -coverprofile={audit_dir}/coverage.out ./..."
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `unit` | string | `go test -v ./...` | Run unit tests with verbose output |
| `integration` | string | (none) | Optional: Run integration tests. Omit to skip |
| `race` | string | `go test -race ./...` | Run race detector (catches concurrent access bugs) |
| `coverage` | string | `go test -coverprofile={audit_dir}/coverage.out ./...` | Generate coverage profile |

**Examples**:

```json
{
  "unit": "go test -v -count=5 ./...",
  "race": "go test -race -timeout=30s ./...",
  "coverage": "go test -coverprofile={audit_dir}/coverage.out -count=1 ./..."
}
```

#### Python Test Commands

**Object Path**: `audit_config.test_commands.python`

```json
{
  "pytest": "uv run pytest --cov=src --cov-branch --cov-report=term"
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `pytest` | string | `uv run pytest --cov=src --cov-branch --cov-report=term` | Run pytest with coverage |

**Notes**:
- Uses `uv run` for PEP 668 compatible environments (Arch Linux, etc.)
- Include `--cov` flags to generate coverage reports
- Coverage data extracted from pytest output automatically

**Examples**:

```json
{
  "pytest": "uv run pytest -xvs tests/ --cov=mypackage --cov-report=term-missing"
}
```

#### R Test Commands

**Object Path**: `audit_config.test_commands.r`

```json
{
  "testthat": "Rscript -e \"devtools::test()\"",
  "coverage": ""
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `testthat` | string | `Rscript -e "devtools::test()"` | Run testthat test suite |
| `coverage` | string | (none) | Optional: Generate coverage with covr package. Omit to skip |

**Examples**:

```json
{
  "testthat": "Rscript -e \"devtools::test()\"",
  "coverage": "Rscript -e \"covr::report(covr::package_coverage())\""
}
```

#### JavaScript/TypeScript Test Commands

**Object Path**: `audit_config.test_commands.javascript`

```json
{
  "test": "npm test"
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `test` | string | `npm test` | Run npm test suite with coverage if configured in package.json |

**Notes**:
- Coverage extracted from Jest/Vitest output if available
- Coverage should be configured in `package.json` or test runner config
- Works with TypeScript projects (automatic transpilation)

**Examples**:

```json
{
  "test": "npm run test:coverage"
}
```

---

## Placeholder System

Test commands support runtime placeholders that are automatically replaced:

| Placeholder | Replacement | Example |
|-------------|------------|---------|
| `{audit_dir}` | Audit output directory path | `.ticket-audits/FEAT-001` |
| `{ticket_id}` | Current ticket ID | `FEAT-001` |
| `{project_root}` | Project root directory | `/home/user/my-project` |

**Usage in Commands**:

```json
{
  "go": {
    "coverage": "go test -coverprofile={audit_dir}/coverage.out ./..."
  }
}
```

When ticket `FEAT-001` runs audit in `/home/user/my-project`, this becomes:

```bash
go test -coverprofile=.ticket-audits/FEAT-001/coverage.out ./...
```

---

## Language Detection

The audit system automatically detects your project language (order of precedence):

1. **Go**: `go.mod` present
2. **Python**: `pyproject.toml` OR `setup.py` present
3. **R**: `DESCRIPTION` OR `*.Rproj` files present
4. **TypeScript**: `package.json` AND `tsconfig.json` present
5. **JavaScript**: `package.json` alone present

No manual configuration needed—detection is automatic.

---

## Output Artifacts

After audit completes, results are written to `.ticket-audits/{ticket_id}/`:

### Go Projects

| File | Description |
|------|-------------|
| `unit-tests.log` | Unit test execution output |
| `integration-tests.log` | Integration test output (if configured) |
| `race-detector.log` | Race detector scan results |
| `coverage.out` | Go coverage profile (binary format) |
| `coverage-report.txt` | Coverage analysis per function |
| `coverage-summary.txt` | Total coverage percentage |
| `implementation-summary.md` | Human-readable ticket completion summary |

### Python Projects

| File | Description |
|------|-------------|
| `pytest-output.log` | Pytest execution output with coverage |
| `coverage-summary.txt` | Coverage percentage extracted from pytest |
| `implementation-summary.md` | Human-readable ticket completion summary |

### R Projects

| File | Description |
|------|-------------|
| `testthat-output.log` | Testthat execution output |
| `coverage-report.txt` | Coverage report (if configured) |
| `implementation-summary.md` | Human-readable ticket completion summary |

### JavaScript/TypeScript Projects

| File | Description |
|------|-------------|
| `npm-test-output.log` | npm test execution output |
| `coverage-summary.txt` | Coverage summary (if available) |
| `implementation-summary.md` | Human-readable ticket completion summary |

---

## Default Values

If `test_commands` is not specified, the system uses language-appropriate defaults:

**Go**:
```json
{
  "unit": "go test -v ./...",
  "race": "go test -race ./...",
  "coverage": "go test -coverprofile={audit_dir}/coverage.out ./..."
}
```

**Python**:
```json
{
  "pytest": "uv run pytest --cov=src --cov-branch --cov-report=term"
}
```

**R**:
```json
{
  "testthat": "Rscript -e \"devtools::test()\""
}
```

**JavaScript**:
```json
{
  "test": "npm test"
}
```

To override defaults, include only the fields you need to customize—the system merges your config with defaults.

---

## Migration Guide

### For Existing Projects (3 Steps)

#### Step 1: Create Minimal Config (Disabled by Default)

Add `.ticket-config.json` to project root with audit disabled:

```json
{
  "tickets_dir": "migration_plan/tickets",
  "audit_config": {
    "enabled": false
  }
}
```

**Why**: Establishes config structure without any changes to workflow.
**Next**: Commit and test that ticket workflow still works.

#### Step 2: Enable with Defaults

Enable audit with auto-detected language and default test commands:

```json
{
  "tickets_dir": "migration_plan/tickets",
  "audit_config": {
    "enabled": true
  }
}
```

**Why**: Adopts automated testing without custom configuration.
**Testing**: Run `/ticket verify` and check `.ticket-audits/` output.
**If tests fail**: Likely due to missing test infrastructure (no tests, no go.mod, etc.). See Troubleshooting.

#### Step 3: Customize Commands (Optional)

Override defaults for project-specific needs:

```json
{
  "tickets_dir": "migration_plan/tickets",
  "audit_config": {
    "enabled": true,
    "test_commands": {
      "go": {
        "unit": "go test -v -timeout=10s ./...",
        "coverage": "go test -coverprofile={audit_dir}/coverage.out -count=1 ./..."
      }
    }
  }
}
```

**Why**: Fine-tune test behavior for your specific project needs.
**Common Customizations**:
- Add `-timeout=Xs` for slow test suites
- Add `-count=5` for flaky test detection
- Add test tags: `-tags=integration`
- Change coverage target: `--cov=mypackage` instead of `src`

---

## Backward Compatibility

The audit system is **fully backward compatible**:

- ❌ No `.ticket-config.json` file → Audit silently skipped
- ✅ `.ticket-config.json` exists but `audit_config` missing → Audit skipped
- ✅ `audit_config` present but `enabled: false` → Audit skipped
- ✅ `enabled: true` → Audit executes with appropriate defaults

**Migration Risk**: Zero. You can enable audit incrementally without affecting existing tickets.

---

## Troubleshooting

### Config/Setup Issues

#### "Error: Invalid JSON in config file"

**Cause**: Syntax error in `.ticket-config.json`

**Fix**: Validate JSON syntax:
```bash
jq empty .ticket-config.json
```

Shows exact line/character of syntax error.

#### "Error: Could not detect project language"

**Cause**: Project root missing standard language indicator files

**Fix**: Add the appropriate indicator file:
- **Go**: Create `go.mod` (or run `go mod init github.com/user/project`)
- **Python**: Create `pyproject.toml` (or `setup.py`)
- **R**: Create `DESCRIPTION` (or `.Rproj` file)
- **JavaScript**: Create `package.json` (or `package-lock.json`)

#### "Audit disabled in config. Skipping audit"

**Cause**: `enabled: false` in `audit_config`

**Fix**: Change to `"enabled": true` in `.ticket-config.json`

### Test Execution Issues

#### "Unit tests failed (see audit-dir/unit-tests.log)"

**Cause**: Test suite has failures or infrastructure missing

**Fix**:
1. Review log: `cat .ticket-audits/{TICKET_ID}/unit-tests.log`
2. Common issues:
   - Missing test files → Create `*_test.go` files
   - Missing dependencies → Run `go mod tidy` / `npm install`
   - Wrong test command → Customize in config (Step 3 above)
3. Run tests manually to verify: `go test -v ./...`

#### "Coverage generation failed"

**Cause**: Test runner doesn't support coverage or missing tools

**Fix**:
- **Go**: Install go tools: `go install golang.org/x/tools/cmd/cover@latest`
- **Python**: Install coverage: `uv pip install pytest-cov`
- **R**: Install covr: `install.packages("covr")`
- **JavaScript**: Ensure Jest/Vitest configured with coverage in `package.json`

### Audit Output Issues

#### "Created minimal summary (template not found)"

**Cause**: Not a critical error. Template missing but audit completed.

**Info**: See `implementation-summary.md` generated in audit directory.

#### "Audit failed (non-blocking, continuing)"

**Cause**: Test execution had non-zero exit code

**Info**: This does NOT block ticket completion. See logs for details.

---

## Configuration Examples by Project Type

### Example 1: Go Project with Integration Tests

```json
{
  "tickets_dir": "migration_plan/tickets",
  "project_name": "api-server",
  "audit_config": {
    "enabled": true,
    "test_commands": {
      "go": {
        "unit": "go test -v -timeout=30s ./...",
        "integration": "go test -v -tags=integration -timeout=60s ./tests/integration",
        "race": "go test -race -timeout=30s ./...",
        "coverage": "go test -coverprofile={audit_dir}/coverage.out -count=1 ./..."
      }
    }
  }
}
```

### Example 2: Python Project with Pytest

```json
{
  "tickets_dir": "migration_plan/tickets",
  "project_name": "data-pipeline",
  "audit_config": {
    "enabled": true,
    "test_commands": {
      "python": {
        "pytest": "uv run pytest tests/ -xvs --cov=src --cov-branch --cov-report=term-missing"
      }
    }
  }
}
```

### Example 3: R Package with Testthat and Covr

```json
{
  "tickets_dir": "migration_plan/tickets",
  "project_name": "analytics-pkg",
  "audit_config": {
    "enabled": true,
    "test_commands": {
      "r": {
        "testthat": "Rscript -e \"devtools::test()\"",
        "coverage": "Rscript -e \"covr::report(covr::package_coverage())\""
      }
    }
  }
}
```

### Example 4: Node.js Project with Jest

```json
{
  "tickets_dir": "migration_plan/tickets",
  "project_name": "frontend-app",
  "audit_config": {
    "enabled": true,
    "test_commands": {
      "javascript": {
        "test": "npm run test:coverage"
      }
    }
  }
}
```

---

## Integration with Ticket Workflow

The audit system integrates into Phase 7.5 of the `/ticket` skill:

1. **User completes ticket implementation** (acceptance criteria met)
2. **`/ticket verify` called** (user or automated)
3. **Acceptance criteria verified** (must pass)
4. **Audit executes** (if `enabled: true`)
5. **Tests run** (language-specific)
6. **Results logged** (`.ticket-audits/{ticket_id}/`)
7. **Summary generated** (human-readable markdown)
8. **Ticket completion continues** (audit failures are non-blocking)

The audit is **optional** and **non-blocking**—ticket completion does NOT depend on test success.

---

## Performance Notes

### Typical Execution Times

| Language | Typical Duration | Factors |
|----------|-----------------|---------|
| Go | 5-30 seconds | Test count, race detection overhead |
| Python | 10-60 seconds | Test count, coverage instrumentation |
| R | 15-90 seconds | Test count, R startup time |
| JavaScript | 5-30 seconds | Test runner, browser automation (if used) |

### Optimization Tips

1. **Use short timeouts** (Go: `timeout=10s` for quick tests)
2. **Skip integration tests** if not needed in config
3. **Exclude slow tests** with tags/markers
4. **Run parallel tests** where supported (Go: built-in, Python: `pytest-xdist`)

---

## Security Considerations

### Test Command Injection

The `test_commands` values are passed to the shell. Be cautious with user-provided values:

```json
{
  "go": {
    "unit": "go test -v ./..."
  }
}
```

This is safe. This is NOT:

```json
{
  "unit": "go test -v $(user_input)"
}
```

**Best Practice**: Only use static, vetted test commands.

### File System Access

Audit operations:
- Read: Project source files, test files
- Write: `.ticket-audits/{ticket_id}/` directory only
- Execute: Test runners in project root

No access to other directories or credentials.

---

## References

- **Ticket Skill**: `~/.claude/skills/ticket/SKILL.md`
- **Run Audit Script**: `~/.claude/skills/ticket/scripts/run-audit.sh`
- **Example Configs**: `~/.claude/skills/ticket/examples/`

---

**Document Version**: 1.0
**Last Updated**: 2026-01-18
**Maintained By**: System
