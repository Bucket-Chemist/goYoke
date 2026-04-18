# Ticket Audit System Documentation

This directory contains complete documentation for the automated audit testing system integrated into the `/ticket` skill.

---

## Quick Navigation

### For Users (Get Started Fast)

Start here if you want to enable automated testing:

1. **[audit-quick-start.md](./audit-quick-start.md)** — 3-minute setup guide
   - Copy example config for your language
   - Verify tests pass
   - Enable audit
   - See first results

**Time to first audit**: 3-5 minutes

### For Developers (Deep Reference)

Complete technical reference for configuration:

1. **[audit-config-schema.md](./audit-config-schema.md)** — Full schema documentation
   - Root object structure
   - Language-specific test commands
   - Placeholder system
   - Default values
   - Output artifacts
   - Troubleshooting

### For Project Managers

See audit configuration in your project's CLAUDE.md:
- `/home/doktersmol/Documents/goYoke/CLAUDE.md`

---

## Documentation Structure

### audit-quick-start.md (286 lines)

User-friendly guide covering:
- **Setup (3 steps)**
  - Create configuration (copy example)
  - Verify tests work
  - Enable audit
- **First Run** — What happens when audit executes
- **Key Concepts** — Placeholders, non-blocking, language detection
- **Customization** — Common patterns
- **Troubleshooting** — Common issues + fixes
- **Examples** — Output structure for each language

**Read if**: You're setting up audit for the first time
**Time**: ~5 minutes

### audit-config-schema.md (611 lines)

Complete technical reference covering:
- **Quick Start** — Minimal working example
- **Schema Reference**
  - Root object
  - audit_config object
  - test_commands for each language (Go, Python, R, JavaScript)
- **Placeholder System** — {audit_dir}, {ticket_id}, {project_root}
- **Language Detection** — Priority order and rules
- **Output Artifacts** — Files generated per language
- **Default Values** — All language defaults
- **Migration Guide** — 3-step adoption path
- **Backward Compatibility** — Zero migration risk
- **Troubleshooting** — Issues with solutions
- **Examples** — 4 complete configs by language
- **Integration** — How audit fits in ticket workflow

**Read if**: You need to customize config or debug issues
**Time**: ~15 minutes (reference, not linear reading)

---

## Example Configurations

All examples are in `~/.claude/skills/ticket/examples/`:

| File | Use For | Test Commands |
|------|---------|---------------|
| `ticket-config-go.json` | Go projects | unit, integration, race, coverage |
| `ticket-config-python.json` | Python projects | pytest with coverage |
| `ticket-config-r.json` | R packages | testthat, optional coverage |
| `ticket-config-javascript.json` | Node.js/TypeScript | npm test |

**To use**: Copy to your project root as `.ticket-config.json`

---

## Supported Languages

### Go
- **Tests**: `go test`
- **Commands**: unit, integration (optional), race detection, coverage
- **Coverage**: `go tool cover` analysis
- **Defaults**: All configured

### Python
- **Tests**: pytest (via `uv run` for PEP 668)
- **Commands**: pytest with coverage flags
- **Coverage**: Extracted from pytest output
- **Defaults**: Configured

### R
- **Tests**: testthat (via `devtools::test()`)
- **Commands**: testthat, optional coverage (via `covr`)
- **Coverage**: Optional, requires covr package
- **Defaults**: Configured

### JavaScript/TypeScript
- **Tests**: npm test / npm run scripts
- **Commands**: npm test command
- **Coverage**: Auto-detected from Jest/Vitest
- **Defaults**: Configured

---

## Core Concepts

### Placeholder System

Commands support 3 placeholders:

| Placeholder | Value | Example |
|-------------|-------|---------|
| `{audit_dir}` | `.ticket-audits/{ticket_id}` | `.ticket-audits/FEAT-001` |
| `{ticket_id}` | Current ticket ID | `FEAT-001` |
| `{project_root}` | Project root | `/home/user/my-project` |

**Used in**: Go coverage, Python pytest, R coverage commands

### Language Detection

Automatic (no config needed):

1. Go: `go.mod` present
2. Python: `pyproject.toml` or `setup.py`
3. R: `DESCRIPTION` or `*.Rproj`
4. TypeScript: `package.json` + `tsconfig.json`
5. JavaScript: `package.json` alone

### Non-Blocking Behavior

Test failures do NOT prevent ticket completion:

```
[FAIL] Unit tests failed
[WARNING] Audit failed but completing ticket anyway
```

Fix tests in next iteration.

### Default Values

If `test_commands` not specified, system uses language defaults:
- Go: unit, race, coverage
- Python: pytest with coverage
- R: testthat
- JavaScript: npm test

---

## Integration with Ticket Workflow

Audit runs in **Phase 7.5** of `/ticket` skill:

1. Phases 1-7: Ticket discovery → verification
2. **Phase 7.5: Audit** (if enabled)
   - Execute tests
   - Generate coverage
   - Create summary
3. Phase 8: Completion

**Status**: Optional, non-blocking, fully backward compatible

---

## Migration Path (3 Steps)

### Step 1: Create Config (Disabled)
```json
{
  "audit_config": { "enabled": false }
}
```
Establish structure, no changes.

### Step 2: Enable with Defaults
```json
{
  "audit_config": { "enabled": true }
}
```
Adopt automation, use language defaults.

### Step 3: Customize (Optional)
```json
{
  "audit_config": {
    "enabled": true,
    "test_commands": {
      "go": { ... }
    }
  }
}
```
Project-specific fine-tuning.

**Risk**: Zero. Enable incrementally.

---

## Output Artifacts

After audit completes, `.ticket-audits/{ticket_id}/` contains:

**All Languages**:
- `timestamp.txt` — Execution timestamp
- `implementation-summary.md` — Human-readable summary

**Go**:
- `unit-tests.log` — Unit test results
- `race-detector.log` — Race condition scan
- `coverage.out` — Binary coverage profile
- `coverage-report.txt` — Per-function analysis
- `coverage-summary.txt` — Total percentage

**Python**:
- `pytest-output.log` — Full pytest output
- `coverage-summary.txt` — Coverage extracted

**R**:
- `testthat-output.log` — Test results
- `coverage-report.txt` — Coverage (if enabled)

**JavaScript**:
- `npm-test-output.log` — Test output
- `coverage-summary.txt` — Coverage (if configured)

---

## Setup Checklist

### Before Enabling Audit

- [ ] Project has test infrastructure (`*_test.go`, `tests/`, etc.)
- [ ] Tests pass manually (`go test`, `pytest`, etc.)
- [ ] Language indicator file exists (`go.mod`, `pyproject.toml`, etc.)
- [ ] Copy example config for your language
- [ ] Verify language detection works

### Enabling Audit

- [ ] Set `"enabled": true` in `.ticket-config.json`
- [ ] Commit config to git
- [ ] Run `/ticket verify` on next ticket
- [ ] Review `.ticket-audits/{ticket_id}/` output

### (Optional) Customization

- [ ] Identify test command changes needed
- [ ] Update `test_commands` section
- [ ] Test manually first
- [ ] Re-run `/ticket verify`

---

## Troubleshooting Overview

### Setup Issues
- JSON syntax error → Validate with `jq empty`
- Language not detected → Add indicator file (go.mod, pyproject.toml, etc.)
- Audit disabled → Set `enabled: true`

### Test Issues
- Tests fail → Run tests manually, fix first
- Coverage missing → Install coverage tool
- Wrong command → Customize in `test_commands`

### Output Issues
- Minimal summary → Template missing (non-critical)
- Audit failed → Check logs in `.ticket-audits/{ticket_id}/`

See **audit-quick-start.md** and **audit-config-schema.md** for detailed solutions.

---

## References

| Document | Purpose |
|----------|---------|
| SKILL.md | Ticket skill workflow (phases 1-8) |
| run-audit.sh | Implementation of audit system |
| audit-quick-start.md | 3-minute user guide |
| audit-config-schema.md | Complete technical reference |
| goYoke/CLAUDE.md | Project-specific configuration |

---

## Getting Help

### Quick Questions
→ See **audit-quick-start.md** (setup, basics)

### Configuration Details
→ See **audit-config-schema.md** (complete reference)

### Project-Specific
→ See `/home/doktersmol/Documents/goYoke/CLAUDE.md`

### Issues
→ See Troubleshooting in **audit-config-schema.md**

---

## Summary

The audit system provides **optional, non-blocking automated testing** integrated into `/ticket` workflow:

- **Setup**: 3 minutes (copy config, verify tests, enable)
- **Languages**: Go, Python, R, JavaScript/TypeScript
- **Features**: Test execution, coverage reports, summary generation
- **Risk**: Zero (backward compatible, opt-in)
- **Blocking**: No (audit failures don't prevent completion)

---

**Documentation Version**: 1.0
**Applies To**: run-audit.sh v1.0+, /ticket skill v1.1+
**Last Updated**: 2026-01-18
**Maintained By**: System
