---
id: PERM-008
title: "Integration Tests: end-to-end permission gate"
status: pending
dependencies: [PERM-001, PERM-002, PERM-003, PERM-004, PERM-005, PERM-006, PERM-007]
time_estimate: "1.5h"
phase: 7
tags: [plan-generated, phase-7, testing, integration]
needs_planning: false
---

# PERM-008: Integration Tests — end-to-end permission gate

## Description

Comprehensive integration and manual test suite for the complete permission
gate feature. Validates the full flow from hook binary through UDS bridge to
TUI modal and back.

## Acceptance Criteria

- [ ] Integration test: UDS server + hook client + response flow (end-to-end)
- [ ] Integration test: inject CLIPermissionRequestMsg → modal → response → bridge resolve
- [ ] Integration test: dispatch() routes permission_gate_request correctly (not dropped)
- [ ] Manual test: TUI → Bash → modal → Allow → command executes
- [ ] Manual test: TUI → Bash → modal → Deny → command blocked
- [ ] Manual test: TUI → Bash → modal → timeout (30s) → auto-deny + toast
- [ ] Manual test: Write/Edit tool → NO modal (auto-allowed)
- [ ] Manual test: Phase 5a without 5b → hook registered but Bash still CLI-denied (harmless)
- [ ] Negative test: invalid GOFORTRESS_SOCKET → hook auto-denies
- [ ] Negative test: stale socket (connect refused) → hook auto-denies

## Files

- `cmd/gogent-permission-gate/main_test.go` — Hook binary unit/integration tests
- `internal/tui/bridge/server_test.go` — Bridge permission gate tests

## Test Matrix

| # | Scenario | Input | Expected | Type |
|---|----------|-------|----------|------|
| 1 | Bash → Allow | Bash tool_use | Command executes | Manual |
| 2 | Bash → Deny | Bash tool_use | tool_result = error | Manual |
| 3 | Bash → Allow for Session | Bash tool_use | Executes + cache written | Manual |
| 4 | Bash after session-allow | Bash tool_use | No modal, auto-allow | Manual |
| 5 | Write tool | Write tool_use | No modal, auto-allow | Manual |
| 6 | Edit tool | Edit tool_use | No modal, auto-allow | Manual |
| 7 | Timeout | No user response in 30s | Auto-deny + toast | Manual |
| 8 | Bad socket | GOFORTRESS_SOCKET=/bad | Auto-deny | Unit |
| 9 | No socket var | unset GOFORTRESS_SOCKET | Auto-deny | Unit |
| 10 | Bridge dispatch | permission_gate_request msg | Routes to handlePermGate | Integration |
| 11 | Bridge round-trip | Request → channel → resolve | Response sent | Integration |
| 12 | Modal round-trip | CLIPermissionRequestMsg → modal → response | ResolvePermGate called | Integration |

## Rollback Verification

After all tests pass, verify rollback:
1. Remove hook from settings.json
2. Revert allowedTools in driver.go
3. Restart TUI
4. Confirm: Bash commands handled by CLI internal permission (no modal)

---

_Generated from: Revised Architecture v2, Integration Testing_
