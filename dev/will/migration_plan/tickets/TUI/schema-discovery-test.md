# User Event Schema Discovery Test

## Purpose
Capture the raw JSON structure of `type: "user"` events from Claude CLI to inform permission handling implementation.

## Build
```bash
go build -o ~/.local/bin/gofortress /home/doktersmol/Documents/GOgent-Fortress/cmd/gofortress
```

## Test Procedure

### 1. Run gofortress
```bash
gofortress
```

### 2. Send a command that triggers permission prompt
In the TUI input field, type:
```
> create a python script that prints hello world
```

### 3. Observe system message
The TUI should display:
```
🔍 Permission event detected. Raw data saved to: /tmp/user-event-{timestamp}.json
```

### 4. Examine captured JSON
```bash
# List captured events
ls -lt /tmp/user-event-*.json

# View the content
cat /tmp/user-event-{actual-timestamp}.json | jq .
```

## Expected Schema Elements

Based on existing event patterns in `internal/cli/events.go`, we expect the user event to contain:

### Minimum Expected Fields
```json
{
  "type": "user",
  "subtype": "permission" | "input" | "confirmation",
  ...
}
```

### Permission-Specific Fields (Hypothesis)
```json
{
  "type": "user",
  "subtype": "permission",
  "permission_type": "tool_use" | "file_write" | "bash_command",
  "tool_name": "Write" | "Bash" | "Edit",
  "file_path": "/path/to/file.py",
  "description": "Create a new Python script",
  "options": ["allow", "deny", "always_allow"],
  ...
}
```

## Schema Analysis Checklist

Once you have the JSON file, document:

- [ ] `type` field value (should be "user")
- [ ] `subtype` field value and possible values
- [ ] Permission-specific fields (what's being requested)
- [ ] Response mechanism fields (how to respond)
- [ ] Any session/context identifiers
- [ ] Optional vs required fields
- [ ] Nested structures (if any)

## Code Location

Instrumentation added to:
- **File**: `/home/doktersmol/Documents/GOgent-Fortress/internal/tui/claude/events.go`
- **Handler**: `case "user":` in `handleEvent()`
- **Lines**: Approximately 68-80

## Cleanup

After capturing the schema, the instrumentation code will be replaced with the actual permission handler implementation.

## Next Steps

1. Run test and capture JSON
2. Document schema in this file
3. Move to Task #2: Response Protocol Discovery
4. Implement production handler in Task #3
