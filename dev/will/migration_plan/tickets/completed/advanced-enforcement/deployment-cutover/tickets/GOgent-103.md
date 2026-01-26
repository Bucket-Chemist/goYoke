---
id: GOgent-103
title: Cutover Decision Workflow
description: **Task**:
status: pending
time_estimate: 1h
dependencies: ["GOgent-102"]
priority: high
week: 5
tags: ["cutover", "week-5"]
tests_required: true
acceptance_criteria_count: 36
---

### GOgent-103: Cutover Decision Workflow

**Time**: 1 hour
**Dependencies**: GOgent-102 (parallel testing complete)

**Task**:
Define cutover decision criteria and workflow. Document GO/NO-GO checklist.

**File**: `docs/cutover-decision.md`

**Implementation**:

````markdown
# Cutover Decision Workflow

**Purpose**: Define criteria and process for switching from Bash to Go hooks.

---

## Pre-Cutover Checklist

### 1. Testing Complete

- [ ] All unit tests pass: `go test ./pkg/... -v`
- [ ] All integration tests pass: `go test ./test/integration/... -v`
- [ ] All regression tests pass: `go test ./test/regression -v`
- [ ] Performance benchmarks meet targets (<5ms p99, <10MB memory)
- [ ] WSL2 compatibility verified: `./test/wsl2/wsl2_test.sh`

### 2. Parallel Testing Results

- [ ] Parallel testing ran for 24+ hours: `./scripts/parallel-test.sh`
- [ ] Processed 100+ real events during parallel testing
- [ ] Match rate ≥99% (Go output matches Bash)
- [ ] Reviewed all differences logged in `differences.log`
- [ ] All critical differences resolved (blocking, violations, etc.)
- [ ] Minor differences (timestamps, formatting) are acceptable

### 3. Installation Verified

- [ ] Installation script tested: `./scripts/install.sh --test-only`
- [ ] Binaries built successfully
- [ ] Binaries installed to `~/.gogent/bin/`
- [ ] Bash hooks backed up to `~/.claude/hooks/backup-*/`

### 4. Rollback Prepared

- [ ] Rollback script tested: `./scripts/rollback.sh --dry-run`
- [ ] Rollback process documented
- [ ] Backup hooks verified (can be restored)
- [ ] Rollback can complete in <5 minutes

### 5. Documentation Complete

- [ ] README updated with Go installation instructions
- [ ] Migration notes documented
- [ ] Known issues documented (if any)
- [ ] User-facing changes documented (none expected)

---

## GO/NO-GO Decision Criteria

### GO Criteria (Proceed with Cutover)

All of the following MUST be true:

1. **Testing**: All test categories pass (unit, integration, regression, performance)
2. **Parallel Testing**: Match rate ≥99% over 24+ hours with 100+ events
3. **Critical Differences**: Zero unresolved critical differences
4. **Installation**: Successful installation on 2+ test machines
5. **Rollback**: Rollback tested and works in <5 minutes
6. **WSL2**: Passes WSL2 compatibility tests (if WSL2 users exist)

### NO-GO Criteria (Do NOT Proceed)

Any of the following triggers NO-GO:

1. **Test Failures**: Any test category fails
2. **Low Match Rate**: Match rate <95% in parallel testing
3. **Critical Differences**: Unresolved differences in blocking logic, violation logging, or session archival
4. **Performance Regression**: Latency >5ms p99 or memory >10MB
5. **Installation Issues**: Cannot install successfully on test machines
6. **Rollback Issues**: Rollback fails or takes >5 minutes

### CAUTION Criteria (Proceed with Extra Monitoring)

Proceed but monitor closely if:

1. **Match Rate**: 95-99% (good but not perfect)
2. **Minor Differences**: Acceptable differences (timestamps, whitespace, formatting)
3. **New Environment**: First deployment to production
4. **Limited Testing**: <100 events in parallel testing (but all match)

---

## Cutover Process

### Phase 1: Preparation (T-24 hours)

1. Review parallel testing report
2. Convene decision meeting with team
3. Make GO/NO-GO decision based on criteria above
4. If GO: Schedule cutover window
5. If NO-GO: Document blockers and re-test timeline

### Phase 2: Cutover Execution (T+0 hours)

**Estimated Duration**: 15 minutes

1. **Notify Users** (if multi-user system)
   ```bash
   wall "GOgent Fortress maintenance: Switching hooks to Go implementation"
   ```
````

2. **Final Backup**

   ```bash
   ./scripts/backup-hooks.sh
   ```

3. **Execute Cutover**

   ```bash
   ./scripts/cutover.sh
   ```

4. **Verify Symlinks**

   ```bash
   ls -la ~/.claude/hooks/ | grep gogent
   ```

5. **Test Basic Operation**

   ```bash
   # Test validate-routing
   echo '{"hook_event_name":"PreToolUse","tool_name":"Read"}' | ~/.claude/hooks/validate-routing

   # Should return valid JSON
   ```

6. **Monitor First Session**
   - Start new Claude Code session
   - Verify hooks execute without errors
   - Check log: `tail -f ~/.gogent/hooks.log`

### Phase 3: Monitoring (T+1 to T+48 hours)

**Monitoring Checklist:**

- [ ] **Hour 1**: Check hooks.log for errors every 15 minutes
- [ ] **Hour 4**: Review violation logs for unexpected entries
- [ ] **Hour 24**: Run health check script: `./scripts/health-check.sh`
- [ ] **Hour 48**: Final review, declare cutover successful or rollback

**What to Monitor:**

1. **Hook Errors**: Check `~/.gogent/hooks.log` for Go panic stacktraces or errors
2. **Claude Code Errors**: Check if users report "hook timed out" or similar
3. **Violation Logs**: Verify violations logged correctly
4. **Performance**: Check if hooks feel slower (user perception)
5. **Session Handoffs**: Verify handoff files generated correctly

### Phase 4: Success Declaration or Rollback

**Success Criteria** (after 48 hours):

- No critical errors in logs
- No user reports of hook failures
- Performance feels equivalent to Bash
- Violation logging works correctly
- Session handoffs generated successfully

**If Successful:**

```bash
# Mark cutover as successful
touch ~/.gogent/cutover-success-$(date +%Y%m%d)

# Document lessons learned
./scripts/post-cutover-notes.sh
```

**If Issues Detected:**

```bash
# Rollback immediately
./scripts/rollback.sh

# Document issues
./scripts/post-rollback-notes.sh
```

---

## Communication Plan

### Before Cutover

**Email Template:**

```
Subject: GOgent Fortress Cutover Scheduled - [DATE] [TIME]

The Claude Code hooks will be upgraded from Bash to Go on [DATE] at [TIME].

Expected Downtime: 15 minutes
Impact: Minimal (hooks will continue working)
Rollback Plan: Available if issues detected

No action required from users.

Questions? Contact: [YOUR_EMAIL]
```

### During Cutover

- Post status updates every 30 minutes
- Report completion within 15 minutes

### After Cutover

**Success Email:**

```
Subject: GOgent Fortress Cutover Complete

The Claude Code hooks have been successfully upgraded to Go.

No user action required. Everything should work as before.

Report any issues to: [YOUR_EMAIL]
```

---

## Rollback Triggers

Rollback immediately if any of these occur during monitoring:

1. **Hook Failures**: ≥3 hook execution errors in logs
2. **Timeouts**: Claude Code reports "hook timed out" multiple times
3. **Blocking Errors**: Validation incorrectly blocks valid operations
4. **Data Loss**: Session handoffs not generated
5. **Performance**: User reports of "hooks feel slow"
6. **Panic**: Any Go panic stacktrace in logs

---

## Decision Meeting Agenda

**Duration**: 30 minutes

1. **Review Testing Results** (10 min)
   - Unit/integration/regression test results
   - Performance benchmark results
   - Parallel testing report

2. **Review Readiness Checklist** (10 min)
   - Go through pre-cutover checklist item by item
   - Address any incomplete items

3. **GO/NO-GO Decision** (5 min)
   - Apply decision criteria
   - Vote: GO or NO-GO
   - If NO-GO: Define timeline for re-evaluation

4. **Plan Execution** (5 min)
   - Assign cutover executor
   - Assign monitor (watches logs during cutover)
   - Schedule monitoring check-ins

---

## Post-Cutover Review

**1 Week After Cutover:**

- Review logs for patterns
- Survey users for feedback
- Document lessons learned
- Update this document based on experience

**Template: Lessons Learned**

```markdown
# Cutover Lessons Learned

**Date**: [DATE]
**Duration**: [ACTUAL_DURATION]
**Issues**: [COUNT]

## What Went Well

- [Item]
- [Item]

## What Could Improve

- [Item]
- [Item]

## Action Items for Next Time

- [Item]
- [Item]
```

---

**Document Status**: ✅ Ready for Use
**Last Updated**: 2026-01-15
**Owner**: [YOUR_NAME]

````

**Acceptance Criteria**:
- [ ] Document defines clear GO/NO-GO criteria
- [ ] Checklist covers all pre-cutover requirements
- [ ] Cutover process broken into phases with timelines
- [ ] Monitoring checklist covers first 48 hours
- [ ] Rollback triggers clearly defined
- [ ] Communication plan includes email templates
- [ ] Decision meeting agenda structured
- [ ] Post-cutover review process defined
- [ ] Document reviewed by team lead

**Why This Matters**: Cutover decision cannot be subjective. Clear criteria and process ensure safe transition with minimal risk.

---
