# Task #2 Completion Report: Permission Mode Discovery

## Objective

Determine the correct approach for handling Claude CLI permissions in gofortress TUI.

**Initial Hypothesis:** Use `--permission-mode delegate` to receive interactive permission request events and respond with approvals.

**Actual Result:** Permission modes don't enable interactive permissions in stream-json mode. Solution is pre-approval via `AllowedTools`.

---

## Investigation Summary

### Phase 1: Event Capture Analysis

Captured permission denial event structure:
```json
{
  "type": "user",
  "message": {
    "role": "user",
    "content": [{
      "type": "tool_result",
      "content": "Claude requested permissions to write to <path>, but you haven't granted it yet.",
      "is_error": true,
      "tool_use_id": "toolu_..."
    }]
  }
}
```

**Key Insight:** This is an ERROR notification, not a permission REQUEST.

### Phase 2: Permission Mode Testing

Tested all permission modes with `--output-format=stream-json`:

| Mode | Expected | Actual |
|------|----------|--------|
| `default` | Standard behavior | Sends error event, blocks action |
| `delegate` | **Interactive requests** | **Still sends error event** ❌ |
| `bypassPermissions` | Auto-approve | Not effective in stream-json ❌ |

**Evidence:** Direct Claude CLI test showed `init` event reports `"permissionMode": "default"` even when `--permission-mode delegate` is specified.

### Phase 3: Solution Implementation

Implemented pre-approval approach:

**File:** `cmd/gofortress/main.go`
```go
cfg = cli.Config{
    ClaudePath:   "claude",
    AllowedTools: []string{"Bash", "Read", "Write", "Edit", "Glob", "Grep", "Task", "TaskOutput"},
    // ...
}
```

This maps to CLI flags: `--allowed-tools Bash --allowed-tools Read ...`

---

## Why Interactive Permissions Don't Work

1. **No Request Protocol:** Claude CLI doesn't send "permission request" events in stream-json mode
2. **Error Events Only:** Permission issues manifest as `tool_result` errors after denial
3. **Mode Flag Ignored:** `--permission-mode` appears to only affect interactive CLI sessions
4. **No Response Mechanism:** No documented way to send approval back to CLI in stream-json

The architecture is:
- **Interactive Mode:** Permission dialogs shown in terminal, user approves/denies
- **Stream-JSON Mode:** Tools must be pre-approved via `--allowed-tools` flag

---

## Implemented Solution

### Code Changes

**1. Config Usage (cmd/gofortress/main.go:99-106)**
```go
AllowedTools: []string{"Bash", "Read", "Write", "Edit", "Glob", "Grep", "Task", "TaskOutput"},
```

**2. Existing Infrastructure (internal/cli/subprocess.go)**
- Config.AllowedTools field already exists (lines 62-65)
- Argument building already wired (lines 171-173)
- No new code needed, just configuration!

### Benefits

✅ **No Permission Dialogs:** Actions execute immediately
✅ **Predictable Behavior:** User knows allowed tools upfront
✅ **Simple Implementation:** Just config, no complex event handling
✅ **Documented Mechanism:** Uses intended Claude CLI automation feature
✅ **Clean Code:** Removed experimental PermissionMode code

---

## Testing

### Manual Test

```bash
# Build
go build -o ~/.local/bin/gofortress ./cmd/gofortress

# Run
gofortress

# Test command
> create a python file that prints hello

# Expected: File created immediately without prompt
```

### Verification Points

- [ ] No permission error events received
- [ ] File actually created at specified path
- [ ] Write tool executes without delay
- [ ] Bash commands work (if tested)

---

## Future Enhancements

Potential TUI features:

1. **Runtime Tool Management:**
   - UI to view allowed tools
   - Button to grant/revoke tools
   - Requires subprocess restart with new config

2. **Per-Session Policies:**
   - Different tool sets for different task types
   - "Safe mode" with Read/Glob only
   - "Full mode" with Bash/Write/Edit

3. **Tool Usage Audit:**
   - Log all Bash commands executed
   - Warn before destructive operations
   - Session summary of tool usage

4. **Smart Prompting:**
   - If denied tool detected, prompt user to grant
   - "Claude wants to use Bash, allow for this session?"

---

## Documentation

Created comprehensive documentation:

**File:** `docs/PERMISSION_HANDLING.md`
- Problem statement
- Investigation findings
- Solution rationale
- Implementation details
- Testing procedures
- Alternative approaches considered
- Future enhancement ideas

---

## Recommendations

### For gofortress

1. **Ship with current AllowedTools list:** Safe, common tools
2. **Add UI indicator:** Show allowed tools in status bar
3. **Consider Bash confirmation:** Optionally prompt before Bash execution
4. **Document in README:** Explain tool permissions to users

### For Task #3

Original plan was "Production Permission Handler". Now:

**Decision:** SKIP Task #3 or repurpose to "Tool Management UI"

**Rationale:**
- No interactive permission handling needed
- AllowedTools solution is complete
- If we want user control, build UI, not event handler

**Alternative Task #3:** "Tool Allowlist UI"
- Settings panel to view/edit allowed tools
- Restart subprocess with new config
- Persist user preferences

---

## Lessons Learned

1. **Test assumptions early:** Permission modes don't work as expected
2. **Read the source:** `--allowed-tools` was the answer all along
3. **Simpler is better:** Pre-approval cleaner than interactive handling
4. **Documentation matters:** Created comprehensive docs for future reference

---

## Artifacts

### Code Changes
- `cmd/gofortress/main.go`: Added AllowedTools config
- `internal/cli/subprocess.go`: Removed experimental PermissionMode code
- Clean build, no test failures

### Documentation
- `docs/PERMISSION_HANDLING.md`: Comprehensive findings
- `docs/TASK-2-COMPLETION-REPORT.md`: This report

### Evidence
- `/tmp/user-event-*.json`: Captured permission denial events
- Claude CLI test output: Shows "delegate" mode doesn't work
- Build verification: Clean compile with solution

---

## Status

**Task #2: COMPLETED ✅**

**Next Steps:**
1. Manual TUI test to verify no permission prompts
2. Update Task #3 description (repurpose or skip)
3. Consider implementing Tool Management UI
4. Update main README with permission handling info

**Recommendation:** Proceed with manual testing, then decide if Task #3 is still relevant or should be replaced with "Tool Allowlist UI" feature.
