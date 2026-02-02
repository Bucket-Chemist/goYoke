---
id: GOgent-106
title: Documentation Updates
description: **Task**:
status: pending
time_estimate: 1.5h
dependencies: ["GOgent-104"]
priority: high
week: 5
tags: ["deployment", "week-5"]
tests_required: true
acceptance_criteria_count: 8
---

### GOgent-106: Documentation Updates

**Time**: 1.5 hours
**Dependencies**: GOgent-104 (cutover complete)

**Task**:
Update project documentation to reflect Go implementation, including README, migration notes, and troubleshooting guide.

**Files**:

- `README.md` (update)
- `docs/migration-notes.md` (new)
- `docs/troubleshooting.md` (new)

**Implementation**:

**Update `README.md`:**

Add section after introduction:

````markdown
## Installation

### Prerequisites

- Go 1.21 or higher
- Bash 4.0 or higher (for installation scripts)
- Claude Code installed and configured

### Quick Install

```bash
# Clone repository
git clone https://github.com/yourusername/gogent-fortress.git
cd gogent-fortress

# Run installation script
./scripts/install.sh

# Run parallel testing (24 hours)
./scripts/parallel-test.sh

# If tests pass, cutover
./scripts/cutover.sh
```
````

### Manual Build

```bash
# Build binaries
go build -o bin/gogent-validate cmd/gogent-validate/main.go
go build -o bin/gogent-archive cmd/gogent-archive/main.go
go build -o bin/gogent-sharp-edge cmd/gogent-sharp-edge/main.go

# Install to ~/.gogent/bin
mkdir -p ~/.gogent/bin
cp bin/* ~/.gogent/bin/

# Create symlinks
ln -sf ~/.gogent/bin/gogent-validate ~/.claude/hooks/validate-routing
ln -sf ~/.gogent/bin/gogent-archive ~/.claude/hooks/session-archive
ln -sf ~/.gogent/bin/gogent-sharp-edge ~/.claude/hooks/sharp-edge-detector
```

## Architecture

GOgent Fortress implements three hooks for Claude Code:

1. **validate-routing** (`gogent-validate`): Enforces routing rules, blocks invalid operations
2. **session-archive** (`gogent-archive`): Generates handoff documents, archives session data
3. **sharp-edge-detector** (`gogent-sharp-edge`): Detects debugging loops, captures sharp edges

See [Architecture Documentation](docs/architecture.md) for details.

## Migration from Bash

If upgrading from Bash hooks, see [Migration Notes](docs/migration-notes.md).

## Troubleshooting

See [Troubleshooting Guide](docs/troubleshooting.md) for common issues.

````

**Create `docs/migration-notes.md`:**

```markdown
# Migration Notes: Bash → Go

**Date**: 2026-01-15
**Version**: Phase 0 Complete

---

## Summary

The GOgent Fortress hooks have been migrated from Bash to Go. This migration provides:

- **Performance**: ~50% faster hook execution (measured in GOgent-098)
- **Reliability**: Type-safe implementation reduces runtime errors
- **Maintainability**: Easier to extend and test
- **Compatibility**: 99%+ behavioral compatibility with Bash implementation

## What Changed

### User-Visible Changes

**None**. The Go implementation is designed to be a drop-in replacement with identical behavior.

### Internal Changes

1. **Implementation Language**: Bash → Go
2. **Binary Names**:
   - `validate-routing.sh` → `gogent-validate` (symlinked as `validate-routing`)
   - `session-archive.sh` → `gogent-archive` (symlinked as `session-archive`)
   - `sharp-edge-detector.sh` → `gogent-sharp-edge` (symlinked as `sharp-edge-detector`)
3. **Installation Location**: `~/.gogent/bin/` (symlinked from `~/.claude/hooks/`)

## Migration Process

### Automated Migration

```bash
# 1. Install Go hooks
./scripts/install.sh

# 2. Run parallel testing
./scripts/parallel-test.sh --duration 24

# 3. Review report
cat ~/.gogent/parallel-test-*/report.md

# 4. If tests pass, cutover
./scripts/cutover.sh

# 5. If issues, rollback
./scripts/rollback.sh
````

### Manual Migration

If you prefer manual control:

```bash
# 1. Backup existing hooks
mkdir -p ~/.claude/hooks/backup-manual
cp ~/.claude/hooks/validate-routing ~/.claude/hooks/backup-manual/
cp ~/.claude/hooks/session-archive ~/.claude/hooks/backup-manual/
cp ~/.claude/hooks/sharp-edge-detector ~/.claude/hooks/backup-manual/

# 2. Build Go binaries
go build -o ~/.gogent/bin/gogent-validate cmd/gogent-validate/main.go
go build -o ~/.gogent/bin/gogent-archive cmd/gogent-archive/main.go
go build -o ~/.gogent/bin/gogent-sharp-edge cmd/gogent-sharp-edge/main.go

# 3. Update symlinks
rm ~/.claude/hooks/validate-routing
ln -s ~/.gogent/bin/gogent-validate ~/.claude/hooks/validate-routing

rm ~/.claude/hooks/session-archive
ln -s ~/.gogent/bin/gogent-archive ~/.claude/hooks/session-archive

rm ~/.claude/hooks/sharp-edge-detector
ln -s ~/.gogent/bin/gogent-sharp-edge ~/.claude/hooks/sharp-edge-detector

# 4. Test
echo '{}' | ~/.claude/hooks/validate-routing
```

## Known Differences

### Minor Differences (Acceptable)

1. **Timestamps**: Go uses RFC3339 format, Bash used custom format
2. **Error Messages**: Slightly different wording (same meaning)
3. **JSON Formatting**: Go uses standard library (whitespace may differ)

### No Functional Differences

The following behaviors are **identical**:

- Routing validation logic
- Blocking decisions
- Violation logging
- Session metrics collection
- Handoff generation
- Sharp edge detection
- Failure counting

## Rollback Procedure

If you encounter issues:

```bash
# Automatic rollback
./scripts/rollback.sh

# Manual rollback
cp ~/.claude/hooks/backup-*/validate-routing ~/.claude/hooks/
cp ~/.claude/hooks/backup-*/session-archive ~/.claude/hooks/
cp ~/.claude/hooks/backup-*/sharp-edge-detector ~/.claude/hooks/
```

## Performance Improvements

Measured in GOgent-098 benchmarks:

| Hook                | Bash (p50) | Go (p50) | Improvement |
| ------------------- | ---------- | -------- | ----------- |
| validate-routing    | 4.2ms      | 2.1ms    | 50% faster  |
| session-archive     | 12.5ms     | 6.8ms    | 46% faster  |
| sharp-edge-detector | 3.8ms      | 1.9ms    | 50% faster  |

## FAQ

**Q: Do I need to change my Claude Code configuration?**
A: No. The hooks are drop-in replacements.

**Q: Will my existing violations and learnings be preserved?**
A: Yes. Go hooks read the same JSONL files as Bash.

**Q: Can I switch back to Bash if needed?**
A: Yes. Run `./scripts/rollback.sh`.

**Q: Does this work on WSL2?**
A: Yes. Tested in GOgent-101b.

**Q: When should I migrate?**
A: After parallel testing shows 99%+ match rate.

---

**Migration Support**: [GitHub Issues](https://github.com/yourusername/gogent/issues)

````

**Create `docs/troubleshooting.md`:**

```markdown
# Troubleshooting Guide

Common issues and solutions for GOgent Fortress Go hooks.

---

## Hook Execution Errors

### Symptom: "hook timed out" in Claude Code

**Cause**: Hook taking >5 seconds to execute.

**Solution**:
1. Check hook logs: `tail -f ~/.gogent/hooks.log`
2. Look for errors or slow operations
3. Verify routing schema is valid: `jq . ~/.claude/routing-schema.json`
4. If corrupt, restore from backup: `cp ~/.claude/routing-schema.json.bak ~/.claude/routing-schema.json`

### Symptom: "permission denied" when running hooks

**Cause**: Binaries not executable.

**Solution**:
```bash
chmod +x ~/.gogent/bin/*
````

### Symptom: "no such file or directory" for hook

**Cause**: Symlinks broken.

**Solution**:

```bash
# Verify symlinks
ls -la ~/.claude/hooks/ | grep gogent

# Recreate if needed
./scripts/cutover.sh
```

---

## Installation Issues

### Symptom: `go: command not found`

**Cause**: Go not installed.

**Solution**:

```bash
# Install Go
wget https://go.dev/dl/go1.21.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

### Symptom: Tests fail during installation

**Cause**: Dependencies missing or code issues.

**Solution**:

1. Check test output for specific errors
2. Install missing dependencies: `go mod download`
3. If persistent, skip tests: `./scripts/install.sh --skip-tests`
4. Report issue with test output

---

## Runtime Issues

### Symptom: Hooks blocking valid operations

**Cause**: Routing schema too restrictive or mismatched tier.

**Solution**:

1. Check current tier: `cat ~/.gogent/current-tier`
2. Check routing schema: `jq '.tiers' ~/.claude/routing-schema.json`
3. Use override flag: `--force-tier=sonnet` in prompt
4. Review violation log: `tail ~/.gogent/routing-violations.jsonl`

### Symptom: Sharp edge not captured after failures

**Cause**: Failure count not reaching threshold (default 3).

**Solution**:

1. Check error log: `tail ~/.gogent/error-patterns.jsonl`
2. Verify failures logged
3. Check threshold: Default is 3 consecutive failures on same file
4. Check pending learnings: `cat ~/.claude/memory/pending-learnings.jsonl`

### Symptom: Session handoff not generated

**Cause**: session-archive hook not executing or failing.

**Solution**:

1. Check hooks log: `grep session-archive ~/.gogent/hooks.log`
2. Verify SessionEnd events firing
3. Check handoff location: `ls ~/.claude/memory/last-handoff.md`
4. Run manually: `echo '{"hook_event_name":"SessionEnd","session_id":"test"}' | ~/.claude/hooks/session-archive`

---

## Performance Issues

### Symptom: Hooks feel slow

**Cause**: Routing schema very large or disk I/O issues.

**Solution**:

1. Run benchmark: `go test -bench=. ./test/benchmark`
2. Check if p99 latency >5ms
3. Check disk space: `df -h ~/.gogent`
4. Reduce log file sizes: `truncate -s 0 ~/.gogent/error-patterns.jsonl`

### Symptom: High memory usage

**Cause**: Large log files or memory leak.

**Solution**:

1. Check memory: `ps aux | grep gogent`
2. Rotate log files: `mv ~/.gogent/hooks.log ~/.gogent/hooks.log.old`
3. Report if >10MB per hook process

---

## Data Issues

### Symptom: Violations not logged

**Cause**: Violations log path issue or permissions.

**Solution**:

1. Check log file: `ls -la ~/.gogent/routing-violations.jsonl`
2. Verify writable: `touch ~/.gogent/test && rm ~/.gogent/test`
3. Check disk space: `df -h ~/.gogent`

### Symptom: Corpus events not parsed

**Cause**: Invalid JSON in corpus file.

**Solution**:

1. Validate JSON: `cat corpus.jsonl | jq . >/dev/null`
2. Look for parse errors
3. Remove invalid lines

---

## WSL2-Specific Issues

### Symptom: Path not found errors on Windows paths

**Cause**: WSL2 path translation needed.

**Solution**:

- Windows paths should use `/mnt/c/...` format
- Verify: `ls /mnt/c/Users`

### Symptom: Line ending issues (CRLF)

**Cause**: Windows-style line endings in scripts.

**Solution**:

```bash
# Convert to LF
dos2unix ~/.claude/hooks/*
```

---

## Rollback Scenarios

### When to Rollback

Rollback immediately if:

1. ≥3 hook execution errors in 1 hour
2. Blocking valid operations repeatedly
3. Session handoffs not generating
4. Any Go panic in logs

### How to Rollback

```bash
# Automatic
./scripts/rollback.sh

# Manual
cp ~/.claude/hooks/backup-cutover-*/validate-routing ~/.claude/hooks/
cp ~/.claude/hooks/backup-cutover-*/session-archive ~/.claude/hooks/
cp ~/.claude/hooks/backup-cutover-*/sharp-edge-detector ~/.claude/hooks/
```

---

## Getting Help

1. **Check logs**: `~/.gogent/hooks.log`
2. **Check violations**: `~/.gogent/routing-violations.jsonl`
3. **Run health check**: `./scripts/health-check.sh`
4. **Report issue**: [GitHub Issues](https://github.com/yourusername/gogent/issues)

Include in bug reports:

- Hook logs (`~/.gogent/hooks.log`)
- Error output
- Steps to reproduce
- Operating system (Linux, WSL2, etc.)
- Go version: `go version`

````

**Acceptance Criteria**:
- [ ] README updated with Go installation instructions
- [ ] Migration notes document created
- [ ] Troubleshooting guide covers common issues
- [ ] Documentation includes WSL2-specific guidance
- [ ] FAQ section answers common questions
- [ ] All documentation reviewed for accuracy
- [ ] Links between documents work correctly
- [ ] Code examples tested and verified

**Why This Matters**: Good documentation reduces support burden and helps users self-serve solutions to common problems.

---
