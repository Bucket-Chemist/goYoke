# Complete Claude Code CLI Interactive Elements Reference

Building a Go TUI wrapper for Claude Code requires intercepting and replicating **every interactive element** the CLI presents. This reference documents all permission prompts, user dialogs, event schemas, and edge cases—everything needed to create a pixel-perfect alternative interface.

## Permission system architecture

Claude Code evaluates permissions through a strict hierarchy: **Hooks (PreToolUse)** → **Permission Rules (deny/allow/ask)** → **Permission Mode** → **canUseTool callback**. Understanding this flow is essential for proper interception.

The SDK exposes four core permission modes via `--permission-mode`:

| Mode | Behavior |
|------|----------|
| `default` | Prompts on first use of each tool requiring permission |
| `acceptEdits` | Auto-approves file operations (Edit/Write); Bash still requires approval |
| `plan` | Read-only mode—Claude can analyze but not modify or execute |
| `bypassPermissions` | Skips ALL permission checks (dangerous, isolated environments only) |

---

## Complete prompt taxonomy

### Native permission prompts

Every tool that modifies state triggers a permission prompt in `default` mode. The prompts share a common structure but vary by tool type:

**Bash commands:**
```
Claude wants to run: npm install express
Allow? [y]es / [n]o / [a]lways for this command / [Tab] to add message
```

**File modifications (Edit/Write):**
```
Claude wants to edit: /src/app.ts
  - Line 15: const x = 1 → const x = 2
Allow? [y]es / [n]o / [a]lways for this session / [Tab] to add message
```

**WebFetch:**
```
Claude wants to fetch: https://api.example.com/data
Allow? [y]es / [n]o / [a]lways for this domain / [Tab] to add message
```

**MCP tools:**
```
Claude wants to use mcp__puppeteer__navigate
  Input: {"url": "https://example.com"}
Allow? [y]es / [n]o / [a]lways for this tool / [Tab] to add message
```

### Response options and scope

| Key | Action | Persistence Scope |
|-----|--------|------------------|
| `y` | Allow this specific use | One-time |
| `n` | Deny this specific use | One-time |
| `a` | Allow all future uses | **Varies by tool type** (see below) |
| `Tab` | Add message before deciding | Allows user feedback to Claude |

**"Accept all" scope by tool type:**
- **Bash commands**: Permanent per project+command pattern (stored in settings)
- **File operations (Edit/Write)**: Until session end only
- **WebFetch**: Per domain for session
- **MCP tools**: Per tool name, configurable destination

### Tools requiring vs. not requiring permission

| Tool | Permission Required | Notes |
|------|---------------------|-------|
| Bash | **Yes** | Always prompts unless allowed in rules |
| Edit | **Yes** | Auto-approved in `acceptEdits` mode |
| Write | **Yes** | Auto-approved in `acceptEdits` mode |
| WebFetch | **Yes** | Domain-based scoping |
| NotebookEdit | **Yes** | |
| MCP tools | **Yes** | First use per tool |
| Read | No | Read-only, no state change |
| Glob | No | Pattern searching only |
| Grep | No | Content searching only |
| LS | No | Directory listing only |
| Task (subagents) | No | Research agents |
| TodoRead/TodoWrite | No | Internal task tracking |

---

## Plan mode and multi-option selection

### Plan mode activation

Plan mode is a **read-only research phase** where Claude analyzes without executing. Activation methods:

1. CLI flag: `claude --permission-mode plan`
2. Keyboard: `Shift+Tab` twice (cycles: normal → acceptEdits → plan)
3. Settings: `{"permissions": {"defaultMode": "plan"}}`

### Plan execution dialog

When Claude completes planning and calls `exit_plan_mode`, users see:

```
╭───────────────────────────────────────────────────────────────╮
│  Claude has prepared a plan. Choose how to proceed:          │
│                                                              │
│  1. Execute with main Claude (auto-accept edits)             │
│  2. Execute with main Claude (manual approval)               │
│  3. No, keep planning                                        │
╰───────────────────────────────────────────────────────────────╯
```

The user presses `1`, `2`, or `3` to select.

### AskUserQuestion tool (multiple choice)

Claude uses this tool for clarification questions, presenting structured choices:

```typescript
interface AskUserQuestionInput {
  questions: Array<{
    question: string;      // Full question text
    header: string;        // Short label (max 12 chars)
    options: Array<{
      label: string;       // Option text (1-5 words)
      description: string; // Explanation
    }>;
    multiSelect: boolean;  // Allow multiple selections
  }>;
}
```

**Display example:**
```
Which framework should I use?

  [A] React - Component-based UI library
  [B] Vue - Progressive framework
  [C] Svelte - Compile-time framework

Select one (A/B/C):
```

---

## NDJSON event schema for permissions

### Streaming output activation

```bash
claude -p "query" --output-format stream-json
claude -p "query" --output-format stream-json --include-partial-messages
```

### Core event types

**System init (first event):**
```json
{
  "type": "system",
  "subtype": "init",
  "session_id": "abc123",
  "cwd": "/path/to/project",
  "tools": ["Read", "Write", "Edit", "Bash", "Grep", "Glob"],
  "model": "claude-sonnet-4-5-20250929",
  "permissionMode": "default",
  "mcp_servers": []
}
```

**User message:**
```json
{
  "type": "user",
  "session_id": "abc123",
  "message": {"role": "user", "content": "Analyze this codebase"},
  "parent_tool_use_id": null
}
```

**Assistant message with tool use:**
```json
{
  "type": "assistant",
  "uuid": "msg_123",
  "session_id": "abc123",
  "message": {
    "role": "assistant",
    "content": [
      {"type": "text", "text": "I'll read the file."},
      {"type": "tool_use", "id": "toolu_01ABC", "name": "Bash", "input": {"command": "npm test"}}
    ]
  }
}
```

**Tool result:**
```json
{
  "type": "user",
  "message": {
    "content": [{
      "type": "tool_result",
      "tool_use_id": "toolu_01ABC",
      "content": "All tests passed."
    }]
  }
}
```

**Result (final event):**
```json
{
  "type": "result",
  "subtype": "success",
  "session_id": "abc123",
  "duration_ms": 5000,
  "total_cost_usd": 0.0123,
  "num_turns": 2,
  "result": "Analysis complete.",
  "usage": {"input_tokens": 1500, "output_tokens": 450},
  "permission_denials": []
}
```

### Permission request hook event

When using hooks with `PermissionRequest`, Claude Code sends to stdin:

```json
{
  "session_id": "abc123",
  "transcript_path": "/Users/.../.claude/projects/.../transcript.jsonl",
  "cwd": "/Users/.../project",
  "permission_mode": "default",
  "hook_event_name": "PermissionRequest",
  "tool_name": "Bash",
  "tool_input": {"command": "npm install", "description": "Install dependencies"}
}
```

### Hook response format (stdout)

**Approve:**
```json
{"decision": "approve", "reason": "Safe command"}
```

**Block:**
```json
{"decision": "block", "reason": "Command not allowed by policy"}
```

**Extended response with input modification:**
```json
{
  "decision": "approve",
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "allow",
    "updatedInput": {"command": "npm install --production"}
  }
}
```

---

## SDK permission callback interface

### TypeScript canUseTool signature

```typescript
type CanUseTool = (
  toolName: string,
  input: ToolInput,
  options: {
    signal: AbortSignal;
    suggestions?: PermissionUpdate[];
  }
) => Promise<PermissionResult>;

type PermissionResult = 
  | {
      behavior: 'allow';
      updatedInput: ToolInput;
      updatedPermissions?: PermissionUpdate[];
    }
  | {
      behavior: 'deny';
      message: string;
      interrupt?: boolean;  // Stop conversation entirely if true
    };
```

### Permission update destinations

```typescript
type PermissionUpdate = {
  type: 'addRules';
  rules: Array<{toolName: string; ruleContent: string}>;
  behavior: 'allow' | 'deny' | 'ask';
  destination: 'session' | 'projectSettings' | 'localSettings' | 'userSettings';
};
```

### Tool input schemas

**Bash:**
```typescript
interface BashInput {
  command: string;
  description?: string;
  timeout?: number;        // Max 600000ms
  run_in_background?: boolean;
}
```

**Edit:**
```typescript
interface FileEditInput {
  file_path: string;
  old_string: string;
  new_string: string;
  replace_all?: boolean;
}
```

**Write:**
```typescript
interface FileWriteInput {
  file_path: string;
  content: string;
}
```

---

## Configuration and scoped permissions

### Settings file precedence (highest to lowest)

1. **Enterprise policies** (managed, cannot override)
2. **CLI arguments**
3. **Local project**: `.claude/settings.local.json` (gitignored)
4. **Shared project**: `.claude/settings.json`
5. **User settings**: `~/.claude/settings.json`

### Permission rule syntax

```json
{
  "permissions": {
    "allow": [
      "Read",
      "Bash(npm run test:*)",
      "Bash(git:*)",
      "Edit(/src/**/*.ts)",
      "WebFetch(domain:github.com)",
      "mcp__puppeteer"
    ],
    "deny": [
      "Read(.env*)",
      "Bash(rm -rf:*)",
      "Bash(sudo:*)"
    ],
    "ask": [
      "Bash(git push:*)"
    ]
  }
}
```

**Pattern matching rules:**
- Bash uses **prefix matching** with `:*` wildcard at end only
- File paths use **gitignore-style globs**: `**`, `*`, `?`
- MCP tools: **No wildcards**—use `mcp__server` for all tools from a server
- Evaluation order: **deny → allow → ask** (first match wins)

---

## Visual elements and keyboard shortcuts

### Mode cycling with Shift+Tab

```
Normal Mode → Auto-Accept Mode → Plan Mode → (back to Normal)
```

The UI displays the current mode prominently. In Go TUI, display this in a status bar.

### Essential keybindings

| Shortcut | Action |
|----------|--------|
| `Enter` | Submit prompt |
| `Shift+Enter` | New line in prompt |
| `Escape` | Interrupt Claude's response |
| `Esc Esc` | Rewind to edit previous prompt |
| `Ctrl+C` | Cancel current operation |
| `Ctrl+O` | Toggle verbose/thinking mode |
| `Ctrl+G` | Open plan in external editor |
| `Ctrl+L` | Clear screen |
| `Shift+Tab` | Cycle permission modes |

### Status line components

A customizable status line can show:
- Current model (◆ Opus / ◇ Sonnet / ○ Haiku)
- Working directory and git branch
- Token usage: `◔ 35k/200k` (with visual progress)
- Session cost: `$2.50` (green <$2, yellow $2-10, red >$10)
- Session duration

---

## Edge cases for robust implementation

### Ctrl+C and Escape handling

**Intended behavior:**
- `Escape`: Interrupts streaming response
- `Ctrl+C`: Cancels current operation (does NOT exit)
- Double `Escape`: Rewinds to edit previous prompt

**Known issues to handle:**
- Interruption may not stop active tool execution immediately
- Agent may continue executing despite interrupt signal
- Windows: Ctrl+C can corrupt session state

### Non-TTY and closed stdin

```go
// Check for interactive TTY
if !term.IsTerminal(int(os.Stdin.Fd())) {
    // Use -p print mode, not interactive REPL
    // Permission prompts won't work—use --permission-prompt-tool
}
```

### Session resume edge cases

If a session ends with pending `tool_use` (no `tool_result`), resume may fail with "No messages returned." Handle gracefully by offering to start fresh.

### 60-second timeout

The `canUseTool` callback must return within **60 seconds** or Claude retries with a different approach. Implement timeout handling in your TUI.

---

## Go implementation patterns

### Parsing NDJSON events

```go
type StreamEvent struct {
    Type      string          `json:"type"`
    Subtype   string          `json:"subtype,omitempty"`
    SessionID string          `json:"session_id"`
    Message   *MessageContent `json:"message,omitempty"`
    UUID      string          `json:"uuid,omitempty"`
}

type MessageContent struct {
    Role    string         `json:"role"`
    Content []ContentBlock `json:"content"`
}

type ContentBlock struct {
    Type  string          `json:"type"`
    Text  string          `json:"text,omitempty"`
    ID    string          `json:"id,omitempty"`
    Name  string          `json:"name,omitempty"`
    Input json.RawMessage `json:"input,omitempty"`
}

func ParseStream(reader io.Reader) (<-chan StreamEvent, <-chan error) {
    events := make(chan StreamEvent)
    errs := make(chan error, 1)
    
    go func() {
        defer close(events)
        scanner := bufio.NewScanner(reader)
        for scanner.Scan() {
            var event StreamEvent
            if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
                errs <- err
                return
            }
            events <- event
        }
    }()
    
    return events, errs
}
```

### Permission prompt rendering

```go
type PermissionRequest struct {
    ToolName  string
    ToolInput json.RawMessage
    SessionID string
}

func RenderPermissionPrompt(req PermissionRequest) string {
    switch req.ToolName {
    case "Bash":
        var input struct{ Command string `json:"command"` }
        json.Unmarshal(req.ToolInput, &input)
        return fmt.Sprintf("Claude wants to run: %s\nAllow? [y]es / [n]o / [a]lways / [Tab] message", input.Command)
    
    case "Edit", "Write":
        var input struct{ FilePath string `json:"file_path"` }
        json.Unmarshal(req.ToolInput, &input)
        return fmt.Sprintf("Claude wants to modify: %s\nAllow? [y]es / [n]o / [a]lways for session / [Tab] message", input.FilePath)
    
    default:
        return fmt.Sprintf("Claude wants to use %s\nAllow? [y]es / [n]o", req.ToolName)
    }
}
```

### Sending permission responses

```go
type PermissionResponse struct {
    Behavior           string                 `json:"behavior"`
    UpdatedInput       map[string]interface{} `json:"updated_input,omitempty"`
    Message            string                 `json:"message,omitempty"`
    UpdatedPermissions []PermissionUpdate     `json:"updated_permissions,omitempty"`
}

type PermissionUpdate struct {
    Type        string `json:"type"`
    Rules       []Rule `json:"rules"`
    Behavior    string `json:"behavior"`
    Destination string `json:"destination"`
}

func AllowTool(input map[string]interface{}) PermissionResponse {
    return PermissionResponse{
        Behavior:     "allow",
        UpdatedInput: input,
    }
}

func AllowWithPersistence(input map[string]interface{}, toolName, pattern, dest string) PermissionResponse {
    return PermissionResponse{
        Behavior:     "allow",
        UpdatedInput: input,
        UpdatedPermissions: []PermissionUpdate{{
            Type:        "addRules",
            Rules:       []Rule{{ToolName: toolName, RuleContent: pattern}},
            Behavior:    "allow",
            Destination: dest, // "session", "projectSettings", "localSettings"
        }},
    }
}

func DenyTool(reason string) PermissionResponse {
    return PermissionResponse{
        Behavior: "deny",
        Message:  reason,
    }
}
```

### Handling "Tab to add message"

When user presses Tab during a permission prompt, capture additional input and include it:

```go
func HandleTabMessage(baseResponse PermissionResponse, userMessage string) PermissionResponse {
    // The message is passed back to Claude as context
    // This is typically handled by the SDK internally
    // In hook mode, include in systemMessage field
    return baseResponse
}
```

### Mode state machine

```go
type PermissionMode int

const (
    ModeDefault PermissionMode = iota
    ModeAcceptEdits
    ModePlan
)

func (m PermissionMode) Next() PermissionMode {
    return (m + 1) % 3
}

func (m PermissionMode) String() string {
    return []string{"Normal", "Auto-Accept", "Plan"}[m]
}

func (m PermissionMode) CLIFlag() string {
    return []string{"default", "acceptEdits", "plan"}[m]
}
```

---

## Complete event flow reference

```
SESSION START
│
├─► system (init) ─────────────────► Parse session_id, tools, model
│
├─► user (prompt) ─────────────────► Echo user's query
│
│   ┌─── TURN LOOP ───────────────────────────────────────────┐
│   │                                                          │
│   ├─► stream_event* ────────────► Render streaming tokens    │
│   │   (if --include-partial-messages)                        │
│   │                                                          │
│   ├─► assistant ────────────────► Check for tool_use blocks  │
│   │       │                                                  │
│   │       ├─► No tool_use ──────► Continue streaming         │
│   │       │                                                  │
│   │       └─► Has tool_use ─────► PERMISSION CHECK           │
│   │               │                                          │
│   │               ├─► Allowed by rules ──► Execute           │
│   │               │                                          │
│   │               └─► Needs approval ───► PROMPT USER        │
│   │                       │                                  │
│   │                       ├─► y ────► Execute tool           │
│   │                       ├─► n ────► Deny, inform Claude    │
│   │                       ├─► a ────► Allow + persist rule   │
│   │                       └─► Tab ──► Get message, re-prompt │
│   │                                                          │
│   ├─► user (tool_result) ───────► Show tool output           │
│   │                                                          │
│   └─── (repeat until Claude done) ──────────────────────────┘
│
└─► result ────────────────────────► Session complete
        │
        └─► Parse: cost, tokens, duration, permission_denials
```

This reference provides the complete foundation for building a Go TUI that fully replicates Claude Code's interactive experience. The key implementation priorities are: **NDJSON event parsing**, **permission prompt rendering with correct scope handling**, and **proper keyboard interrupt handling** for the edge cases documented above.