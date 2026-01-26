# GOfortress TUI - Debug Status Report

## What GOfortress Is Trying To Do

GOfortress is a Bubbletea-based TUI wrapper around Claude Code CLI. It aims to:
1. Spawn `claude` CLI as a subprocess with streaming JSON I/O
2. Display Claude's responses in a viewport
3. Accept user input via a textarea
4. Show agent tree and hook events in sidebars

## How It Communicates With Claude CLI

### Command Line Invocation

```bash
claude --print --verbose --input-format stream-json --output-format stream-json --session-id <uuid>
```

**Flags explained:**
- `--print` - Non-interactive mode (required for programmatic use)
- `--verbose` - Required when using `--output-format stream-json` with `--print`
- `--input-format stream-json` - Accept NDJSON on stdin
- `--output-format stream-json` - Emit NDJSON on stdout
- `--session-id <uuid>` - Session identifier (must be valid UUID)

### Message Format (stdin → claude)

We send messages as NDJSON (one JSON object per line):

```json
{"type":"user","message":{"role":"user","content":"your message here"}}
```

This format was determined by testing - Claude CLI requires:
- `type` field at top level (value: `"user"` or `"control"`)
- `message` object with `role` and `content`

### Event Format (claude → stdout)

Claude emits NDJSON events:

```json
{"type":"system","subtype":"init","session_id":"...","tools":[...],...}
{"type":"assistant","message":{"content":[{"type":"text","text":"..."}],...},...}
{"type":"result","is_error":false,"total_cost_usd":0.05,...}
```

## Current Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    main.go                               │
│  - Creates ClaudeProcess                                │
│  - Creates TUI components (PanelModel, AgentTree, etc.) │
│  - Runs tea.Program                                     │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│              internal/cli/subprocess.go                  │
│  ClaudeProcess:                                         │
│  - Spawns claude subprocess with exec.Command           │
│  - Creates stdin/stdout/stderr pipes                    │
│  - events channel (chan Event) for parsed events        │
│  - errors channel (chan error) for errors               │
│  - done channel (chan struct{}) for shutdown signal     │
│  - Goroutines: readEvents(), readStderr(), monitorRestart() │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│              internal/tui/claude/panel.go                │
│  PanelModel:                                            │
│  - Holds reference to ClaudeProcess                     │
│  - Init() returns waitForEvent() cmd to read events     │
│  - Update() handles cli.Event, re-subscribes            │
│  - View() renders viewport + textarea + sidebar         │
└─────────────────────────────────────────────────────────┘
```

## Fixes Applied So Far

### 1. Added --verbose flag (subprocess.go)
Claude CLI requires `--verbose` when using `--print` with `stream-json` output.

### 2. Fixed message format (events.go, subprocess.go)
Changed from `{"content":"..."}` to `{"type":"user","message":{"role":"user","content":"..."}}`.

### 3. Channel lifecycle fixes (subprocess.go)
- Removed `defer close(cp.events)` from readEvents()
- Removed `defer close(cp.errors)` from readStderr()
- Added `sync.Once` for channel closure in Stop()
- Fresh channels created on restart instead of reusing

### 4. Event pump (panel.go)
- Added `waitForEvent()` function to read from process.Events() channel
- Added `processStoppedMsg` type for graceful channel close handling
- Init() now starts event subscription
- Update() re-subscribes after each event

### 5. Other TUI fixes (panel.go, banner.go, input.go, events.go)
- Mouse handling added
- Dynamic sidebar width
- Banner width guards
- Input early returns

## Symptoms Still Occurring

User reports:
- `panic: send on closed channel`
- `panic: close of closed channel`
- TUI unresponsive
- Banner not showing
- Text input doesn't work

## Potential Remaining Issues

### 1. Binary Not Rebuilt?
The user may be running an old binary. Verify with:
```bash
go build -o /tmp/gofortress-new ./cmd/gofortress/
/tmp/gofortress-new
```

### 2. Race Condition in Channel Swap
When restart() swaps channels (`cp.events = newProc.events`), old goroutines may still have references to old channel pointers. The current design assumes goroutines read `cp.events` on each iteration, but there may be timing issues.

### 3. Process Exits Before Event Pump Starts
If claude exits immediately (e.g., auth issue, hooks failing), the event channel closes before the TUI can handle it properly.

### 4. Done Channel Not Closed on Non-Restart Exit
In `monitorRestart()`, if restart is disabled and process exits, the `done` channel is never closed. Reader goroutines may hang or cause issues.

### 5. Hook Failures
Your GOgent hooks (gogent-load-context, etc.) run on SessionStart. If they fail or output unexpected data, claude may exit.

### 6. TTY Requirements
Some claude operations may expect a real TTY. The `--print` flag should handle this, but there may be edge cases.

## Manual Testing Commands

### Test claude CLI directly:
```bash
SESSION=$(uuidgen)
echo '{"type":"user","message":{"role":"user","content":"say hi"}}' | \
  claude --print --verbose --input-format stream-json \
         --output-format stream-json --session-id "$SESSION"
```

### Test with kept-open stdin:
```bash
SESSION=$(uuidgen)
(echo '{"type":"user","message":{"role":"user","content":"say hi"}}'; sleep 30) | \
  claude --print --verbose --input-format stream-json \
         --output-format stream-json --session-id "$SESSION"
```

### Check if claude works at all:
```bash
claude --print "say hello"
```

### Check hooks:
```bash
claude --print --verbose "test" 2>&1 | head -50
```

## Key Files

| File | Purpose |
|------|---------|
| `cmd/gofortress/main.go` | Entry point, wires everything together |
| `internal/cli/subprocess.go` | Claude process management |
| `internal/cli/events.go` | Event types and UserMessage struct |
| `internal/cli/streams.go` | NDJSON reader/writer |
| `internal/tui/claude/panel.go` | Main Claude interaction panel |
| `internal/tui/claude/input.go` | Keyboard input handling |
| `internal/tui/claude/events.go` | Event handling and hook sidebar |
| `internal/tui/layout/layout.go` | Main layout with panels |
| `internal/tui/layout/banner.go` | Top banner with tabs |

## Questions To Research

1. What is the exact expected format for `--input-format stream-json`? Is there official documentation?
2. Does claude CLI have a `--debug` flag that shows more details?
3. Are there version-specific differences in the streaming JSON format?
4. What happens if hooks write to stdout during session start?
5. Is there a way to disable hooks for testing (`--no-hooks` or similar)?

## Claude Code Version

```bash
claude --version
```

Check if there's documentation at:
- `claude --help`
- https://docs.anthropic.com/claude-code
- Claude Code GitHub issues
