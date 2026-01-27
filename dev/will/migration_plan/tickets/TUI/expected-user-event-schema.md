# Expected User Event Schema

## Hypothesis Based on Existing Event Patterns

Looking at `/home/doktersmol/Documents/GOgent-Fortress/internal/cli/events.go`, we can predict the likely structure:

---

## Existing Event Patterns

### SystemEvent (reference)
```go
type SystemEvent struct {
    Event                          // Base: Type, Subtype, Raw
    HookID     string              // Unique identifier
    HookName   string              // Human-readable name
    CWD        string              // Context
    SessionID  string              // Session correlation
    Tools      []string            // Related tools
    Model      string              // Model context
    ExitCode   int                 // Status
    Stdout     string              // Output
}
```

### AssistantEvent (reference)
```go
type AssistantEvent struct {
    Event                          // Base: Type, Subtype, Raw
    Message   AssistantMessage     // Nested content
    SessionID string               // Session correlation
    Partial   bool                 // Streaming indicator
}
```

### ResultEvent (reference)
```go
type ResultEvent struct {
    Event                          // Base: Type, Subtype, Raw
    IsError      bool              // Status
    DurationMs   int64             // Metrics
    Result       string            // Output
    SessionID    string            // Session correlation
    TotalCostUSD float64           // Cost tracking
}
```

---

## Predicted UserEvent Structure

### Minimal Structure (Highly Likely)
```json
{
  "type": "user",
  "subtype": "permission" | "input" | "confirmation",
  "session_id": "abc123...",
  "timestamp": "2026-01-27T07:22:00Z"
}
```

### Permission Subtype (Most Likely for File Operations)
```json
{
  "type": "user",
  "subtype": "permission",
  "session_id": "abc123...",
  "permission_type": "tool_use" | "file_write" | "bash_command",
  "tool_name": "Write" | "Edit" | "Bash",
  "tool_input": {
    "file_path": "/path/to/script.py",
    "content": "#!/usr/bin/env python3\nprint('hello world')\n",
    "description": "Create a new Python script"
  },
  "message": "Claude wants to create a new file: /path/to/script.py",
  "options": ["allow", "deny", "always_allow", "never_allow"],
  "default": "deny",
  "timeout_ms": 30000
}
```

### Response Format (Hypothesis)

**Option 1: Via stdin (like UserMessage)**
```json
{
  "type": "user_response",
  "permission_id": "perm-123",
  "action": "allow" | "deny" | "always_allow" | "never_allow"
}
```

**Option 2: Special permission response format**
```json
{
  "type": "permission_response",
  "session_id": "abc123...",
  "action": "allow",
  "remember": false
}
```

---

## Key Questions to Answer from Captured JSON

### Structure Questions
1. What is the exact `subtype` for permission events?
2. Is there a `permission_id` or similar correlation field?
3. How is the tool/action being requested encoded?
4. Are there nested objects or flat structure?

### Response Mechanism Questions
5. Is there a `response_format` or `expected_response` field?
6. Does the event include instructions on how to respond?
7. Is response via stdin write, or separate API call?
8. Is there a timeout specified?

### Content Questions
9. What user-facing message is provided (for display)?
10. Are there multiple permission types (file, command, network)?
11. Is there a security context (user, workspace, trust level)?
12. Are there default actions or suggestions?

### Session Questions
13. Is `session_id` present for correlation?
14. Are there any sequence numbers or request IDs?
15. Is there conversation context included?

---

## Comparison with UserMessage (Outgoing)

**We send this format TO Claude:**
```go
type UserMessage struct {
    Type    string      `json:"type"`     // "user"
    Message UserContent `json:"message"`  // Nested content
}

type UserContent struct {
    Role    string         `json:"role"`    // "user"
    Content []ContentBlock `json:"content"` // Array of blocks
}
```

**Hypothesis:** Incoming user events might mirror this structure:
```json
{
  "type": "user",
  "subtype": "permission",
  "message": {
    "role": "system",
    "content": [
      {
        "type": "permission_request",
        "permission_type": "file_write",
        "file_path": "/path/to/file.py",
        "action": "create"
      }
    ]
  }
}
```

---

## Analysis Checklist

After capturing the actual JSON, verify:

### Required Fields
- [ ] `type` is "user"
- [ ] `subtype` exists and documents its value
- [ ] Session identifier field present
- [ ] Permission target clearly specified

### Response Protocol
- [ ] Response mechanism documented
- [ ] Response format specified
- [ ] Allowed values enumerated
- [ ] Correlation ID for matching request/response

### Content
- [ ] User-facing message suitable for display
- [ ] All necessary context to make allow/deny decision
- [ ] Security implications clear
- [ ] Default behavior specified

### Edge Cases
- [ ] What happens on timeout?
- [ ] What happens if response is malformed?
- [ ] Can permissions be batched?
- [ ] Are there permission levels/scopes?

---

## Schema Definition Template

Once captured, add the Go struct here:

```go
// UserEvent represents permission requests from Claude CLI
type UserEvent struct {
    Event
    // TODO: Add fields based on captured JSON
    // SessionID     string `json:"session_id"`
    // PermissionID  string `json:"permission_id,omitempty"`
    // ...
}
```

And add the parser:

```go
// AsUser attempts to parse the event as UserEvent.
// Returns error if the event is not a user event or is malformed.
func (e Event) AsUser() (*UserEvent, error) {
    if e.Type != "user" {
        return nil, fmt.Errorf("event type is %q, not user", e.Type)
    }
    var ue UserEvent
    if err := json.Unmarshal(e.Raw, &ue); err != nil {
        return nil, fmt.Errorf("unmarshal user event: %w", err)
    }
    return &ue, nil
}
```

---

## Next Steps After Capture

1. **Document actual schema** in this file
2. **Add UserEvent struct** to `internal/cli/events.go`
3. **Implement AsUser() parser**
4. **Update events_test.go** with test cases
5. **Validate response mechanism** (Task #2)

---

**Status**: Awaiting manual test execution to capture actual JSON.
