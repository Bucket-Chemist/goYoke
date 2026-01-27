# Claude CLI Permission Handling in GOgent-Fortress

## Problem Statement

When using Claude CLI in `--output-format=stream-json` mode, tool use (Write, Edit, Bash) triggers permission checks. Initial hypothesis was that we could handle these interactively via TUI.

## Findings

### Permission Event Structure

Permission denials appear as `user` events with `tool_result` errors:

```json
{
  "type": "user",
  "message": {
    "role": "user",
    "content": [{
      "type": "tool_result",
      "content": "Claude requested permissions to write to /path/to/file, but you haven't granted it yet.",
      "is_error": true,
      "tool_use_id": "toolu_..."
    }]
  },
  "tool_use_result": "Error: Claude requested permissions...",
  "session_id": "...",
  "uuid": "..."
}
```

### Permission Modes Testing

Tested all Claude CLI permission modes with `--output-format=stream-json`:

| Flag | Behavior | Event Type |
|------|----------|------------|
| `--permission-mode default` | Sends error event, blocks action | `user` event with error |
| `--permission-mode delegate` | **Still sends error event** | `user` event with error |
| `--permission-mode bypassPermissions` | Not effective in stream-json | Same as default |
| `--permission-mode dontAsk` | Not tested (likely same) | N/A |

**Key Discovery:** The `--permission-mode` flag appears to be designed for interactive CLI sessions, not `stream-json` mode. In stream-json mode, permission events are always **error notifications**, not **permission requests**.

### Why Interactive Permission Handling Doesn't Work

1. **No Request Events:** Claude CLI never sends a "permission request" event that we could respond to
2. **Error Events Only:** By the time we see the event, it's already a denial
3. **No Response Protocol:** There's no documented way to send permission approval back to Claude CLI in stream-json mode

The `init` event shows:
```json
"permissionMode": "default"
```
Even when we specify `--permission-mode delegate`, suggesting the flag is ignored in stream-json mode.

## Solution: Pre-Approval via AllowedTools

### Implementation

Instead of interactive permission handling, pre-approve tools at subprocess creation:

```go
cfg := cli.Config{
    ClaudePath:   "claude",
    AllowedTools: []string{"Bash", "Read", "Write", "Edit", "Glob", "Grep", "Task", "TaskOutput"},
    // ...
}
```

This maps to Claude CLI flags:
```bash
claude --allowed-tools Bash --allowed-tools Read --allowed-tools Write ...
```

### Benefits

1. **No Permission Dialogs:** Tools are pre-approved, actions execute immediately
2. **Predictable Behavior:** User knows upfront what Claude can do
3. **Simple Implementation:** No complex event handling needed
4. **Matches CLI Design:** `--allowed-tools` is the intended mechanism for automation

### Code Changes

**File:** `internal/cli/subprocess.go`
- Already has `AllowedTools []string` field in Config (lines 62-65)
- Already wired into args building (lines 171-173)

**File:** `cmd/gofortress/main.go`
- Set default AllowedTools for new sessions (lines 99-106)

```go
AllowedTools: []string{"Bash", "Read", "Write", "Edit", "Glob", "Grep", "Task", "TaskOutput"},
```

### Future Enhancements

Could add UI controls for:
1. **Runtime tool allowlist editing:** Let user grant/revoke tools during session
2. **Per-session policies:** Different tool sets for different tasks
3. **Audit trail:** Log when Claude uses powerful tools (Bash, Write)
4. **Confirmations for sensitive tools:** Prompt before Bash execution

These would require subprocess restart with new `AllowedTools` config.

## Alternative Approaches Considered

### Approach 1: Interactive Permission Handling ❌

**Hypothesis:** Use `--permission-mode delegate` to get permission request events, respond with approval messages.

**Result:** Failed. No request events exist in stream-json mode.

**Evidence:**
```bash
claude --permission-mode delegate --output-format stream-json
# Still sends error events, not request events
```

### Approach 2: Permission Event Response Protocol ❌

**Hypothesis:** Send user messages back to approve denied tool uses.

**Result:** No documented protocol exists. Error events don't include a response mechanism.

**Evidence:** Searched Claude CLI docs, tested various message formats, no response mechanism found.

### Approach 3: AllowedTools Pre-Approval ✅

**Implementation:** Pre-configure allowed tools at subprocess start.

**Result:** Success. Clean, documented, intended mechanism.

## Testing

### Manual Test Steps

1. Build: `go build -o ~/.local/bin/gofortress ./cmd/gofortress`
2. Run: `gofortress`
3. Command: "create a python file that prints hello"
4. Expected: File created immediately, no permission prompt
5. Verify: Check file exists at specified path

### Automated Test Plan

```go
func TestAllowedToolsPreApproval(t *testing.T) {
    cfg := cli.Config{
        AllowedTools: []string{"Write"},
        NoHooks: true,
    }

    proc, _ := cli.NewClaudeProcess(cfg)
    proc.Start()

    proc.Send("create /tmp/test.py with print('hello')")

    // Wait for completion
    event := <-proc.Events()

    // Should NOT see permission error
    assert.NotEqual(t, "user", event.Type)

    // Should see successful tool_result
    // ...
}
```

## Recommendations

1. **Use AllowedTools by default:** Pre-approve common safe tools
2. **Consider security:** Be cautious with Bash pre-approval
3. **Future UI:** Add tool allowlist editor in TUI settings
4. **Documentation:** Update user guide with allowed tools explanation

## References

- Claude CLI documentation: `claude --help`
- Config struct: `internal/cli/subprocess.go:18-79`
- Subprocess args: `internal/cli/subprocess.go:139-191`
- Main config: `cmd/gofortress/main.go:92-106`

## Captured Event Examples

See: `/tmp/user-event-*.json` from testing sessions

Example event:
```json
{
  "type": "user",
  "message": {
    "role": "user",
    "content": [{
      "type": "tool_result",
      "content": "Claude requested permissions to write to /home/user/hello.py, but you haven't granted it yet.",
      "is_error": true,
      "tool_use_id": "toolu_01JPe1RANUVkiuGDj8gzDuXn"
    }]
  },
  "tool_use_result": "Error: Claude requested permissions to write to /home/user/hello.py, but you haven't granted it yet."
}
```

This is a **denial**, not a **request**.
