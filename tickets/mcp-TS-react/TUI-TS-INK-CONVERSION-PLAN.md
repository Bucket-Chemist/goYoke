# TUI TypeScript/Ink Conversion Plan

**Status:** Planning (Reviewed)
**Created:** 2026-02-01
**Last Review:** 2026-02-01 (Staff Architect + Einstein)
**Scope:** Migrate GOgent-Fortress frontend from Go (Bubbletea) to TypeScript (Ink/React)

---

## 1. Executive Summary

### 1.1 What We're Doing

Replacing the Go-based TUI (`cmd/gofortress/`, `internal/tui/`, `internal/cli/`, `internal/callback/`) with a TypeScript implementation using Ink (React for terminals) and the official Claude Agent SDK.

### 1.2 Why

| Problem | Impact | Solution |
|---------|--------|----------|
| MCP SDK is TypeScript-native | Fighting upstream, manual JSON bridging | Use native SDK |
| Callback server complexity | Extra process, socket IPC, 230 LOC of bridge code | SDK handles MCP in-process |
| Two-stage JSON parsing | Verbose, error-prone event handling | TypeScript discriminated unions |
| Manual state management | 8+ variables, race conditions | React state + Zustand |

### 1.3 What Stays

All Go hooks remain unchanged:
- `gogent-load-context`
- `gogent-validate`
- `gogent-sharp-edge`
- `gogent-archive`
- `gogent-agent-endstate`

These are fast, well-tested, and communicate via filesystem—no code changes needed.

### 1.4 Expected Outcomes

| Metric | Current (Go) | Target (TS) |
|--------|--------------|-------------|
| Frontend LOC | ~4,280 | ~1,200 |
| Processes | 3 (TUI + Claude + MCP) | 2 (TUI + Claude) |
| MCP integration | Manual socket bridge | Native SDK |
| Type safety | Runtime only | Compile-time |

---

## 2. Current Architecture

### 2.1 Component Map

```
cmd/
├── gofortress/                    # TUI entry point
│   └── main.go                    # 300 LOC - socket cleanup, server start, tea.Program
├── gofortress-mcp-server/         # MCP stdio server
│   └── main.go                    # 250 LOC - 4 tools, callback client
└── debugtui/                      # Debug utility (deprecated - use Node debugging)

internal/
├── tui/
│   ├── layout/
│   │   ├── layout.go              # 350 LOC - 2-panel split, focus management
│   │   └── banner.go              # Status bar, session ID, cost
│   ├── claude/
│   │   ├── panel.go               # 600 LOC - conversation viewport, textarea, state machine
│   │   ├── modal.go               # 200 LOC - overlay system for MCP prompts
│   │   ├── output.go              # Message rendering
│   │   ├── input.go               # Input handling
│   │   └── events.go              # Event aggregation
│   ├── agents/
│   │   ├── model.go               # 400 LOC - AgentTree, AgentNode hierarchy
│   │   ├── view.go                # Tree rendering with viewport
│   │   ├── detail.go              # Selected agent details
│   │   └── picker.go              # Session picker modal
│   └── telemetry/
│       ├── aggregator.go          # Cost/event aggregation
│       └── watcher.go             # File system monitoring
├── cli/
│   ├── subprocess.go              # 500 LOC - ClaudeProcess, NDJSON streaming
│   ├── events.go                  # 250 LOC - Event types, two-stage parsing
│   ├── streams.go                 # NDJSON reader/writer
│   ├── restart.go                 # Auto-restart with backoff
│   ├── session.go                 # Session persistence
│   └── subagent.go                # Task event handling
├── callback/
│   ├── server.go                  # 230 LOC - Unix socket HTTP for TUI↔MCP
│   ├── client.go                  # MCP server's client to reach TUI
│   └── recovery.go                # Stale socket cleanup
└── mcp/
    └── config.go                  # MCP server discovery
```

### 2.2 Process Model

```
┌─────────────────┐     NDJSON      ┌─────────────────┐
│   gofortress    │◄───────────────►│  claude CLI     │
│   (Bubbletea)   │    stdin/out    │  subprocess     │
└────────┬────────┘                 └─────────────────┘
         │
         │ Unix socket HTTP
         │ /prompt, /confirm
         ▼
┌─────────────────┐     stdio       ┌─────────────────┐
│ callback.Server │◄───────────────►│ gofortress-mcp  │
│ (in TUI proc)   │                 │ -server (MCP)   │
└─────────────────┘                 └─────────────────┘
```

### 2.3 Data Flows

**User Input → Claude:**
```
textarea.Update(KeyMsg)
  → ClaudeProcess.Send(message)
  → NDJSON to claude stdin
```

**Claude Response → Display:**
```
claude stdout (NDJSON)
  → ClaudeProcess.Events() chan
  → panel.Update(cli.Event)
  → viewport.SetContent(rendered)
```

**MCP Prompt → User:**
```
claude calls ask_user tool
  → MCP server receives via stdio
  → callback.Client.SendPrompt()
  → HTTP POST to Unix socket
  → callback.Server queues to PromptChan
  → panel.ListenForPrompts() receives
  → modal.HandlePrompt() displays
  → user responds
  → callback.Server returns HTTP response
  → MCP server returns tool result
```

### 2.4 State Variables (panel.go)

```go
type PanelModel struct {
    focused        bool              // Panel focus
    streaming      bool              // Currently streaming response
    state          ProcessState      // Connecting/Ready/Streaming/Error/Stopped
    restartInfo    *RestartEvent     // Restart state
    currentModel   string            // Model name
    modal          ModalState        // Active modal
    callbackServer *callback.Server  // Server reference
    ctx            context.Context   // Cancellation
    messages       []Message         // Conversation history
    viewport       viewport.Model    // Scrollable area
    textarea       textarea.Model    // User input
    // ... more
}
```

---

## 3. Target Architecture

### 3.1 Technology Stack

| Layer | Choice | Rationale |
|-------|--------|-----------|
| Runtime | Node.js 22+ | LTS, native ESM, fast startup |
| Language | TypeScript 5.6+ | Strict mode, SDK compatibility |
| TUI Framework | Ink 5.x | React patterns, active maintenance |
| State | Zustand 5.x | Lightweight, TypeScript-first, no boilerplate |
| MCP | @anthropic-ai/claude-agent-sdk | Official, native TypeScript, eliminates bridge |
| Schema | Zod 3.x | Runtime validation + static types |
| Build | esbuild | Fast bundling for production |
| Dev | tsx | Native TS execution, fast reload |

### 3.2 New Process Model

```
┌─────────────────────────────────────────────────────────┐
│                  gofortress-tui (Node.js)               │
│                                                         │
│  ┌─────────────────┐    ┌─────────────────────────────┐│
│  │  Ink/React TUI  │    │  Claude Agent SDK           ││
│  │                 │    │                             ││
│  │  <Layout/>      │◄──►│  query() streaming          ││
│  │  <ClaudePanel/> │    │  mcpServer (in-process)     ││
│  │  <AgentTree/>   │    │  tool handlers              ││
│  │  <Modal/>       │    │                             ││
│  └─────────────────┘    └─────────────────────────────┘│
└─────────────────────────────────────────────────────────┘
         │
         │ SDK manages subprocess internally
         ▼
┌─────────────────────────────────────────────────────────┐
│                    claude CLI (managed by SDK)          │
└─────────────────────────────────────────────────────────┘
```

**Key change:** MCP server runs in-process. No socket. No separate process.

### 3.3 Directory Structure

```
packages/
└── tui/
    ├── package.json
    ├── tsconfig.json
    ├── src/
    │   ├── index.tsx                 # Entry: render(<App />)
    │   ├── App.tsx                   # Root component, providers
    │   │
    │   ├── components/
    │   │   ├── Layout.tsx            # Two-panel split
    │   │   ├── Banner.tsx            # Status bar
    │   │   ├── ClaudePanel.tsx       # Conversation + input
    │   │   ├── AgentTree.tsx         # Delegation hierarchy
    │   │   ├── AgentDetail.tsx       # Selected agent info
    │   │   ├── Modal.tsx             # Overlay container
    │   │   ├── modals/
    │   │   │   ├── AskModal.tsx
    │   │   │   ├── ConfirmModal.tsx
    │   │   │   ├── InputModal.tsx
    │   │   │   └── SelectModal.tsx
    │   │   └── primitives/
    │   │       ├── Viewport.tsx      # Scrollable area
    │   │       ├── TextInput.tsx     # Styled input
    │   │       └── Spinner.tsx       # Loading indicator
    │   │
    │   ├── hooks/
    │   │   ├── useClaudeQuery.ts     # SDK query wrapper
    │   │   ├── useAgentTree.ts       # Tree state management
    │   │   ├── useKeymap.ts          # Key bindings
    │   │   ├── useTelemetry.ts       # File watcher for Go hook output
    │   │   └── useSession.ts         # Session persistence
    │   │
    │   ├── mcp/
    │   │   ├── server.ts             # createSdkMcpServer config
    │   │   └── tools/
    │   │       ├── askUser.ts
    │   │       ├── confirmAction.ts
    │   │       ├── requestInput.ts
    │   │       └── selectOption.ts
    │   │
    │   ├── store/
    │   │   ├── index.ts              # Combined store
    │   │   ├── slices/
    │   │   │   ├── messages.ts       # Conversation state
    │   │   │   ├── agents.ts         # Agent tree state
    │   │   │   ├── session.ts        # Session metadata
    │   │   │   ├── modal.ts          # Modal queue
    │   │   │   └── ui.ts             # Focus, streaming, etc.
    │   │   └── types.ts              # Store type definitions
    │   │
    │   ├── types/
    │   │   ├── events.ts             # Claude event types
    │   │   ├── mcp.ts                # MCP tool schemas
    │   │   └── agent.ts              # Agent node types
    │   │
    │   └── utils/
    │       ├── markdown.ts           # MD → terminal rendering
    │       ├── colors.ts             # Theme/styling
    │       ├── jsonl.ts              # JSONL incremental reader (offset tracking)
    │       └── format.ts             # Cost, duration formatting
    │
    ├── bin/
    │   └── gofortress-tui.js         # CLI entry with commander
    │
    └── tests/
        ├── components/
        ├── hooks/
        ├── integration/              # End-to-end flow tests
        └── mcp/
```

### 3.4 Component Hierarchy

```
<App>
  <StoreProvider>
    <Layout>
      <Banner />
      <Box flexDirection="row">
        <ClaudePanel />        {/* 70% width */}
        <Box flexDirection="column">
          <AgentTree />        {/* 60% height */}
          <AgentDetail />      {/* 40% height */}
        </Box>
      </Box>
      {modalQueue.length > 0 && (
        <Modal>
          <CurrentModal />     {/* Ask/Confirm/Input/Select */}
        </Modal>
      )}
    </Layout>
  </StoreProvider>
</App>
```

---

## 4. Implementation Specifications

### 4.1 MCP Server (Critical Path)

```typescript
// src/mcp/server.ts
import { createSdkMcpServer, tool } from "@anthropic-ai/claude-agent-sdk";
import { z } from "zod";
import { useModalStore } from "../store/slices/modal";

export const mcpServer = createSdkMcpServer({
  name: "gofortress-interactive",
  version: "1.0.0",
  tools: [
    tool(
      "ask_user",
      "Ask the user a question with optional predefined options",
      {
        message: z.string(),
        options: z.array(z.string()).optional(),
        default: z.string().optional()
      },
      async (args) => {
        const response = await useModalStore.getState().enqueue({
          type: "ask",
          message: args.message,
          options: args.options,
          defaultValue: args.default
        });
        return { content: [{ type: "text", text: response.value }] };
      }
    ),
    // ... confirm_action, request_input, select_option
  ]
});
```

**Tool Specifications:**

| Tool | Input Schema | Output | Modal Type |
|------|--------------|--------|------------|
| `ask_user` | `{message, options?, default?}` | `{value: string}` | Text or buttons |
| `confirm_action` | `{action, destructive?}` | `{confirmed: bool, cancelled: bool}` | Yes/No |
| `request_input` | `{prompt, placeholder?}` | `{value: string}` | Text input |
| `select_option` | `{message, options[{label,value}]}` | `{selected: string, index: number}` | List |

### 4.2 Query Hook

```typescript
// src/hooks/useClaudeQuery.ts
import { query } from "@anthropic-ai/claude-agent-sdk";
import { useStore } from "../store";
import { mcpServer } from "../mcp/server";

export function useClaudeQuery() {
  const { addMessage, setStreaming, updateCost, addAgent } = useStore();

  const sendMessage = async (content: string) => {
    setStreaming(true);

    async function* messages() {
      yield {
        type: "user" as const,
        message: { role: "user" as const, content: [{ type: "text", text: content }] }
      };
    }

    try {
      for await (const event of query({
        prompt: messages(),
        options: {
          mcpServers: { "gofortress-interactive": mcpServer },
          allowedTools: [
            "mcp__gofortress-interactive__ask_user",
            "mcp__gofortress-interactive__confirm_action",
            "mcp__gofortress-interactive__request_input",
            "mcp__gofortress-interactive__select_option"
          ]
        }
      })) {
        handleEvent(event);
      }
    } finally {
      setStreaming(false);
    }
  };

  return { sendMessage };
}
```

### 4.3 Store Structure

```typescript
// src/store/types.ts
interface Message {
  id: string;
  role: "user" | "assistant" | "system";
  content: ContentBlock[];
  partial: boolean;
  timestamp: number;
}

interface Agent {
  id: string;
  parentId: string | null;
  model: string;
  tier: "haiku" | "sonnet" | "opus";
  status: "spawning" | "running" | "complete" | "error";
  description?: string;
  startTime: number;
  endTime?: number;
  tokenUsage?: { input: number; output: number };
}

interface ModalRequest<T = unknown> {
  id: string;
  type: "ask" | "confirm" | "input" | "select";
  payload: T;
  resolve: (response: ModalResponse) => void;
  reject: (error: Error) => void;
  timeout?: number;
}

interface Store {
  // Messages slice (in-memory only, not persisted to session files)
  messages: Message[];
  addMessage: (msg: Omit<Message, "id" | "timestamp">) => void;
  updateLastMessage: (content: ContentBlock[]) => void;

  // UI slice
  streaming: boolean;
  focusedPanel: "claude" | "agents";
  setStreaming: (s: boolean) => void;
  setFocusedPanel: (p: "claude" | "agents") => void;

  // Session slice (matches Go format exactly)
  sessionId: string | null;
  totalCost: number;
  tokenCount: { input: number; output: number };
  updateSession: (data: Partial<SessionData>) => void;

  // Agents slice
  agents: Map<string, Agent>;
  selectedAgentId: string | null;
  rootAgentId: string | null;
  addAgent: (agent: Omit<Agent, "startTime">) => void;
  updateAgent: (id: string, data: Partial<Agent>) => void;
  selectAgent: (id: string | null) => void;

  // Modal slice
  modalQueue: ModalRequest[];
  enqueue: <T>(request: Omit<ModalRequest<T>, "id" | "resolve" | "reject">) => Promise<ModalResponse>;
  dequeue: (id: string, response: ModalResponse) => void;
  cancel: (id: string) => void;
}
```

### 4.4 Key Components

**Layout.tsx:**
```typescript
export function Layout() {
  const { focusedPanel, setFocusedPanel, modalQueue } = useStore();

  useInput((input, key) => {
    if (key.tab) setFocusedPanel(focusedPanel === "claude" ? "agents" : "claude");
    if (key.escape && modalQueue.length === 0) process.exit(0);
  });

  return (
    <Box flexDirection="column" height="100%">
      <Banner />
      <Box flexDirection="row" flexGrow={1}>
        <Box width="70%"><ClaudePanel focused={focusedPanel === "claude"} /></Box>
        <Box width="30%" flexDirection="column">
          <Box height="60%"><AgentTree focused={focusedPanel === "agents"} /></Box>
          <Box height="40%"><AgentDetail /></Box>
        </Box>
      </Box>
      {modalQueue.length > 0 && <ModalOverlay request={modalQueue[0]} />}
    </Box>
  );
}
```

**ClaudePanel.tsx:**
```typescript
export function ClaudePanel({ focused }: { focused: boolean }) {
  const { messages, streaming } = useStore();
  const { sendMessage } = useClaudeQuery();
  const [input, setInput] = useState("");

  const handleSubmit = () => {
    if (input.trim() && !streaming) {
      sendMessage(input);
      setInput("");
    }
  };

  return (
    <Box flexDirection="column" borderStyle="single" borderColor={focused ? "cyan" : "gray"}>
      <Viewport items={messages} renderItem={renderMessage} />
      <Box borderStyle="single" borderColor="gray">
        <TextInput
          value={input}
          onChange={setInput}
          onSubmit={handleSubmit}
          placeholder={streaming ? "Waiting..." : "Type a message..."}
          disabled={streaming}
        />
        {streaming && <Spinner type="dots" />}
      </Box>
    </Box>
  );
}
```

### 4.5 Telemetry Integration

Go hooks write to filesystem. TypeScript reads with offset tracking:

```typescript
// src/utils/jsonl.ts
interface JsonlReader {
  path: string;
  offset: number;
  inode: number;
}

export async function readNewLines(reader: JsonlReader): Promise<{ lines: object[], newOffset: number }> {
  const stat = await fs.stat(reader.path);

  // File was rotated/replaced - reset offset
  if (stat.ino !== reader.inode) {
    reader.offset = 0;
    reader.inode = stat.ino;
  }

  // No new content
  if (stat.size <= reader.offset) {
    return { lines: [], newOffset: reader.offset };
  }

  // Read only new bytes
  const handle = await fs.open(reader.path, 'r');
  const buffer = Buffer.alloc(stat.size - reader.offset);
  await handle.read(buffer, 0, buffer.length, reader.offset);
  await handle.close();

  const content = buffer.toString('utf-8');
  const lines = content.trim().split('\n').filter(Boolean).map(JSON.parse);

  return { lines, newOffset: stat.size };
}
```

```typescript
// src/hooks/useTelemetry.ts
import { watch } from "chokidar";
import { readNewLines, JsonlReader } from "../utils/jsonl";

const readers: Record<string, JsonlReader> = {
  routingDecisions: {
    path: `${process.env.XDG_DATA_HOME}/gogent-fortress/routing-decisions.jsonl`,
    offset: 0,
    inode: 0
  },
  handoffs: {
    path: `${process.env.HOME}/.claude/memory/handoffs.jsonl`,
    offset: 0,
    inode: 0
  }
};

export function useTelemetry() {
  const { updateTelemetry } = useStore();

  useEffect(() => {
    const watcher = watch(Object.values(readers).map(r => r.path), { persistent: true });

    watcher.on("change", async (path) => {
      const key = Object.keys(readers).find(k => readers[k].path === path);
      if (!key) return;

      const { lines, newOffset } = await readNewLines(readers[key]);
      readers[key].offset = newOffset;

      for (const line of lines) {
        updateTelemetry(key, line);
      }
    });

    return () => watcher.close();
  }, []);
}
```

---

## 5. Migration Phases

### Phase 0: Validation Sprint (GATE)

**Duration:** 2 days
**Dependencies:** None
**Purpose:** Verify assumptions before committing resources

**Deliverables:**
- [ ] Install `@anthropic-ai/claude-agent-sdk` in test project
- [ ] Verify `query()`, `createSdkMcpServer()`, `tool()` exports exist and work
- [ ] Test MCP tool invocation with simple handler
- [ ] Measure Go TUI baseline metrics:
  - Cold start time
  - Memory at idle
  - Memory under load
  - Input-to-display latency
- [ ] Verify Go hooks work with Node.js parent process
- [ ] Test Ink complex layout (nested flex, dynamic resize)

**Acceptance Criteria:**
```bash
# SDK verification
cd test-project
npm install @anthropic-ai/claude-agent-sdk
npx tsx test-sdk.ts  # Must successfully call query() with mcpServer

# Baseline metrics captured
cat metrics/go-baseline.json
# { "coldStart": 187, "memoryIdle": 28, "memoryActive": 52, "inputLatency": 12 }
```

**Gate Decision:**
- All pass → Proceed to Phase 1
- SDK APIs missing → HALT, redesign architecture
- Hook issues → Document workaround, proceed with caution

---

### Phase 1: Project Foundation

**Duration:** 3 days
**Dependencies:** Phase 0 PASS
**Deliverables:**
- [ ] Create `packages/tui/` directory
- [ ] Initialize `package.json` with dependencies
- [ ] Configure `tsconfig.json` (strict mode)
- [ ] Set up esbuild for production builds
- [ ] Configure tsx for development
- [ ] Create basic `<App />` that renders "Hello"
- [ ] Verify Go hooks work when TUI spawns Claude
- [ ] Spike: Test complex Ink layout (2-panel, resize handling)

**Acceptance Criteria:**
```bash
cd packages/tui && npm run dev
# Renders "Hello" in terminal
# Ctrl+C exits cleanly
# Resize terminal → layout adjusts
```

---

### Phase 2: Store & State

**Duration:** 3 days
**Dependencies:** Phase 1
**Deliverables:**
- [ ] Implement Zustand store with all slices
- [ ] Message state (add, update partial)
- [ ] Agent state (tree operations)
- [ ] Session state (cost, tokens) - **matches Go format exactly**
- [ ] Modal queue with Promise-based resolution
- [ ] UI state (focus, streaming)

**Acceptance Criteria:**
- Messages persist across component re-renders
- Partial messages update in place (no flicker)
- Session file format identical to Go: `{id, name?, created_at, last_used, cost, tool_calls}`

---

### Phase 3: Core Components + Modal System

**Duration:** 6 days
**Dependencies:** Phase 2
**Deliverables:**
- [ ] `<Layout />` - two-panel split, focus management
- [ ] `<Banner />` - session ID, cost, status
- [ ] `<ClaudePanel />` - viewport + input
- [ ] `<Viewport />` - scrollable message list with position persistence
- [ ] `<TextInput />` - styled input field
- [ ] Message rendering (markdown → terminal)
- [ ] `<ModalOverlay />` - centered overlay
- [ ] `<AskModal />` - question + options/text
- [ ] `<ConfirmModal />` - yes/no
- [ ] `<InputModal />` - text input
- [ ] `<SelectModal />` - list selection
- [ ] Keyboard handling (enter, escape, arrows)
- [ ] Timeout handling for modals

**Acceptance Criteria:**
- Tab switches focus between panels
- Messages scroll correctly, position preserved on resize
- Input captures keystrokes
- Cost updates in banner
- Modals render centered over content
- Escape cancels modal
- Enter submits modal
- Timeout returns default/cancels

---

### Phase 4: MCP Integration

**Duration:** 4 days
**Dependencies:** Phase 3 (modals must exist for tool testing)
**Deliverables:**
- [ ] Implement `mcpServer` with 4 tools
- [ ] Create `useClaudeQuery` hook
- [ ] Wire up streaming event handling
- [ ] Error classification mapping (Go error types → TS)
- [ ] Test: Claude can call `ask_user`, response returns correctly
- [ ] Test: All 4 tools work end-to-end
- [ ] Test: Concurrent tool calls handled correctly

**Acceptance Criteria:**
```
User: "Ask me what color I like"
Claude: [calls ask_user tool]
[Modal appears with question]
User: [types "blue"]
Claude: "You said you like blue!"
```

---

### Phase 5: Agent Visualization

**Duration:** 3 days
**Dependencies:** Phase 4
**Deliverables:**
- [ ] `<AgentTree />` - hierarchical tree view
- [ ] `<AgentDetail />` - selected agent info
- [ ] Tree navigation (up/down/enter)
- [ ] Status indicators (running/complete/error)

**Acceptance Criteria:**
- Agent spawns appear in tree
- Completion updates status
- Selection shows detail panel
- Tree handles 20+ agents without performance issues

---

### Phase 6: Session & Persistence

**Duration:** 2 days
**Dependencies:** Phase 2
**Deliverables:**
- [ ] Session list (`--list` flag)
- [ ] Session resume (`--session ID`)
- [ ] Session file read/write (**Go format preserved**)
- [ ] Restart logic with exponential backoff
- [ ] Graceful shutdown handler (coordinate with Go hooks)

**Acceptance Criteria:**
- `gofortress-tui --list` shows sessions
- `gofortress-tui --session abc` resumes
- Session file readable by Go TUI (rollback compatibility)
- Crash → auto-restart with delay
- Max 3 restart attempts before giving up

---

### Phase 7: Telemetry & Polish

**Duration:** 4 days
**Dependencies:** Phase 6
**Deliverables:**
- [ ] File watchers for Go hook output (offset-tracking JSONL reader)
- [ ] Cost aggregation from events
- [ ] Handoff display from last session
- [ ] Error boundaries (React error boundary component)
- [ ] Terminal compatibility testing (see matrix below)
- [ ] Performance benchmarking vs. Go baseline

**Terminal Compatibility Matrix:**

| Terminal | Linux | macOS | Windows | Pass Criteria |
|----------|-------|-------|---------|---------------|
| iTerm2 | - | Test | - | Full color, mouse, resize |
| Alacritty | Test | Test | Test | Full color, mouse, resize |
| Kitty | Test | Test | - | Full color, mouse, resize |
| macOS Terminal | - | Test | - | 256 color, resize |
| Windows Terminal | - | - | Test | Full color, resize |
| GNOME Terminal | Test | - | - | Full color, mouse, resize |

**Acceptance Criteria:**
- Routing decisions appear in real-time (within 500ms of file change)
- Works in all terminals in matrix with degraded features noted
- Performance within targets (see Section 10.2)

---

### Phase 8: Cutover

**Duration:** 2 days
**Dependencies:** All phases
**Deliverables:**
- [ ] Move Go frontend to `deprecated/`
- [ ] Update `Makefile` / build scripts
- [ ] Update README
- [ ] Integration test suite passes
- [ ] Performance comparison documented
- [ ] Rollback rehearsal completed

**Acceptance Criteria:**
- `make build` produces new TUI
- Old TUI accessible via `--legacy`
- No regression in functionality
- Rollback tested: can switch back to Go TUI and resume session

---

## 6. File-by-File Migration Map

### 6.1 Files to Replace

| Go File | TS Replacement | Notes |
|---------|----------------|-------|
| `cmd/gofortress/main.go` | `src/index.tsx` | Entry point |
| `cmd/gofortress-mcp-server/main.go` | `src/mcp/server.ts` | In-process, no binary |
| `internal/tui/layout/layout.go` | `src/components/Layout.tsx` | React component |
| `internal/tui/layout/banner.go` | `src/components/Banner.tsx` | React component |
| `internal/tui/claude/panel.go` | `src/components/ClaudePanel.tsx` | Simpler with hooks |
| `internal/tui/claude/modal.go` | `src/components/Modal.tsx` + `modals/*` | React portals |
| `internal/tui/claude/output.go` | `src/utils/markdown.ts` | MD rendering |
| `internal/tui/claude/input.go` | ink-text-input wrapper | Library |
| `internal/tui/agents/model.go` | `src/store/slices/agents.ts` | Zustand slice |
| `internal/tui/agents/view.go` | `src/components/AgentTree.tsx` | React component |
| `internal/tui/agents/detail.go` | `src/components/AgentDetail.tsx` | React component |
| `internal/cli/subprocess.go` | SDK `query()` | Managed by SDK |
| `internal/cli/events.go` | `src/types/events.ts` | TypeScript types |
| `internal/cli/streams.go` | SDK handles | Not needed |
| `internal/cli/restart.go` | `src/hooks/useSession.ts` | Simpler |
| `internal/callback/server.go` | **DELETED** | Not needed |
| `internal/callback/client.go` | **DELETED** | Not needed |

### 6.2 Files to Keep (unchanged)

```
cmd/
├── gogent-load-context/
├── gogent-validate/
├── gogent-sharp-edge/
├── gogent-archive/
├── gogent-agent-endstate/
└── [all other gogent-* binaries]
```

### 6.3 Files to Archive

```
deprecated/
├── cmd/
│   ├── gofortress/
│   ├── gofortress-mcp-server/
│   └── debugtui/
└── internal/
    ├── tui/
    ├── cli/
    ├── callback/
    └── mcp/
```

---

## 7. Dependencies

### 7.1 package.json

```json
{
  "name": "@gofortress/tui",
  "version": "1.0.0",
  "type": "module",
  "main": "dist/index.js",
  "bin": {
    "gofortress-tui": "./bin/gofortress-tui.js"
  },
  "scripts": {
    "dev": "tsx watch src/index.tsx",
    "build": "esbuild src/index.tsx --bundle --platform=node --target=node22 --outfile=dist/index.js",
    "start": "node dist/index.js",
    "typecheck": "tsc --noEmit",
    "test": "vitest",
    "test:integration": "vitest run tests/integration",
    "lint": "eslint src/"
  },
  "dependencies": {
    "@anthropic-ai/claude-agent-sdk": "^1.0.0",
    "ink": "^5.0.0",
    "ink-text-input": "^6.0.0",
    "ink-select-input": "^6.0.0",
    "ink-spinner": "^5.0.0",
    "react": "^18.3.0",
    "zustand": "^5.0.0",
    "zod": "^3.23.0",
    "chokidar": "^4.0.0",
    "commander": "^12.0.0",
    "marked": "^15.0.0",
    "marked-terminal": "^7.0.0"
  },
  "devDependencies": {
    "@types/node": "^22.0.0",
    "@types/react": "^18.3.0",
    "esbuild": "^0.24.0",
    "tsx": "^4.19.0",
    "typescript": "^5.6.0",
    "vitest": "^2.0.0",
    "eslint": "^9.0.0",
    "@typescript-eslint/eslint-plugin": "^8.0.0",
    "ink-testing-library": "^4.0.0"
  }
}
```

### 7.2 tsconfig.json

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "lib": ["ES2022"],
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "resolveJsonModule": true,
    "declaration": true,
    "declarationMap": true,
    "sourceMap": true,
    "outDir": "./dist",
    "rootDir": "./src",
    "jsx": "react-jsx",
    "jsxImportSource": "react",
    "noUncheckedIndexedAccess": true,
    "noImplicitOverride": true,
    "noPropertyAccessFromIndexSignature": true
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules", "dist"]
}
```

---

## 8. Risk Register

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| SDK API changes | Low | High | Pin version, monitor changelog, Phase 0 verification |
| Ink rendering bugs | Medium | Medium | Test matrix (Section 7.7), fallback to simpler layout |
| Node.js startup latency | Medium | Low | Benchmark in Phase 0, consider Bun if >800ms |
| Hook integration failure | Low | High | Integration test in Phase 0, keep Go TUI for 90 days |
| Memory leaks in watchers | Medium | Low | Cleanup in useEffect returns, monitor in Phase 7 |
| Modal race conditions | Medium | Medium | Queue-based with Promise resolution, mutex if needed |
| **NEW:** Hook filesystem race | Medium | Medium | Graceful shutdown handler, signal coordination |
| **NEW:** Ink layout performance | Low | Medium | Profile in Phase 1 spike, simplify if needed |
| **NEW:** Session format migration | N/A | N/A | **RESOLVED:** Keep Go format (see Section 12) |

---

## 9. Testing Strategy

### 9.1 Unit Tests

```typescript
// tests/store/messages.test.ts
describe("messages slice", () => {
  it("adds message with timestamp", () => {
    const { addMessage, messages } = useStore.getState();
    addMessage({ role: "user", content: [{ type: "text", text: "hello" }], partial: false });
    expect(messages).toHaveLength(1);
    expect(messages[0].timestamp).toBeDefined();
  });

  it("updates partial message in place", () => {
    // ...
  });
});

// tests/utils/jsonl.test.ts
describe("JSONL incremental reader", () => {
  it("tracks offset across reads", async () => {
    // Write initial content
    await fs.writeFile(testPath, '{"a":1}\n');
    const reader = createReader(testPath);

    const first = await readNewLines(reader);
    expect(first.lines).toHaveLength(1);

    // Append new content
    await fs.appendFile(testPath, '{"b":2}\n');

    const second = await readNewLines(reader);
    expect(second.lines).toHaveLength(1);
    expect(second.lines[0]).toEqual({ b: 2 });
  });
});
```

### 9.2 Component Tests

```typescript
// tests/components/Modal.test.tsx
import { render } from "ink-testing-library";

describe("<AskModal />", () => {
  it("renders question text", () => {
    const { lastFrame } = render(
      <AskModal message="What color?" options={["Red", "Blue"]} onSubmit={() => {}} />
    );
    expect(lastFrame()).toContain("What color?");
  });

  it("submits on Enter", async () => {
    const onSubmit = vi.fn();
    const { stdin } = render(<AskModal message="?" onSubmit={onSubmit} />);
    await stdin.write("\r");
    expect(onSubmit).toHaveBeenCalled();
  });
});

// tests/components/Layout.test.tsx
describe("<Layout />", () => {
  it("handles terminal resize", async () => {
    const { lastFrame, rerender } = render(<Layout />, { columns: 80, rows: 24 });
    expect(lastFrame()).toContain("Claude"); // Panel visible

    rerender(<Layout />, { columns: 40, rows: 24 });
    // Should gracefully degrade or show message
  });
});
```

### 9.3 Integration Tests

```typescript
// tests/integration/mcp.test.ts
describe("MCP tools", () => {
  it("ask_user returns user response", async () => {
    // Mock modal response
    useModalStore.getState().enqueue = async () => ({ value: "blue" });

    const tool = mcpServer.tools.find(t => t.name === "ask_user");
    const result = await tool.handler({ message: "Color?", options: [] });

    expect(result.content[0].text).toBe("blue");
  });
});

// tests/integration/session.test.ts
describe("Session persistence", () => {
  it("writes Go-compatible session format", async () => {
    await saveSession({ id: "test-123", cost: 0.05, toolCalls: 10 });

    const content = await fs.readFile(sessionPath, "utf-8");
    const parsed = JSON.parse(content);

    // Must match Go format exactly
    expect(parsed).toHaveProperty("id");
    expect(parsed).toHaveProperty("created_at");
    expect(parsed).toHaveProperty("last_used");
    expect(parsed).toHaveProperty("cost");
    expect(parsed).toHaveProperty("tool_calls");
  });

  it("can be read by Go TUI", async () => {
    // Write session with TS
    await saveSession({ id: "test-456", cost: 0.10 });

    // Verify Go can read it (spawn Go binary)
    const result = await exec("./deprecated/gofortress --list");
    expect(result.stdout).toContain("test-456");
  });
});

// tests/integration/e2e.test.ts
describe("End-to-end flow", () => {
  it("sends message and displays response", async () => {
    // Full flow test with mocked Claude subprocess
  });

  it("handles tool call with modal interaction", async () => {
    // Tool call → modal → response → continuation
  });
});
```

### 9.4 Test Coverage Requirements

| Area | Minimum Coverage | Notes |
|------|------------------|-------|
| Store slices | 90% | Critical state management |
| MCP tools | 100% | Must work perfectly |
| Components | 80% | Visual testing is hard |
| Utils | 90% | JSONL reader, markdown, etc. |
| Hooks | 70% | Side effects hard to test |
| **Overall** | **80%** | Enforced in CI |

---

## 10. Success Metrics

### 10.1 Functional Parity

| Feature | Status |
|---------|--------|
| Send/receive messages | |
| Streaming display | |
| ask_user tool | |
| confirm_action tool | |
| request_input tool | |
| select_option tool | |
| Agent tree display | |
| Agent selection | |
| Session list | |
| Session resume | |
| Auto-restart | |
| Cost tracking | |
| Keyboard navigation | |

### 10.2 Performance Targets

| Metric | Go Baseline | TS Target | Acceptable | Measured In |
|--------|-------------|-----------|------------|-------------|
| Cold start | TBD (Phase 0) | <500ms | <800ms | Phase 7 |
| Memory idle | TBD (Phase 0) | <80MB | <120MB | Phase 7 |
| Memory active | TBD (Phase 0) | <100MB | <150MB | Phase 7 |
| Input latency | TBD (Phase 0) | <32ms | <50ms | Phase 7 |
| Event processing | TBD (Phase 0) | <10ms | <20ms | Phase 7 |

### 10.3 Quality Gates

- [ ] TypeScript strict mode, zero `any` in business logic
- [ ] ESLint clean (no warnings)
- [ ] Test coverage >80% (branch coverage)
- [ ] No known memory leaks
- [ ] Works in 5+ terminal emulators (see matrix)
- [ ] Session format compatible with Go TUI (rollback test passes)

---

## 11. Rollback Plan

### 11.1 Rollback Triggers

| Trigger | Threshold | Action |
|---------|-----------|--------|
| Startup time | >800ms consistently | Roll back, investigate |
| Memory usage | >150MB idle | Roll back, investigate |
| User-reported input lag | >3 reports of >50ms | Roll back, investigate |
| Session corruption | Any data loss | Immediate rollback |
| Tool failures | >5% failure rate | Roll back, investigate |

### 11.2 Rollback Procedure

1. **Phase 1-3 failure:** Delete `packages/tui/`, continue with Go
2. **Phase 4-6 failure:** Freeze TS work, evaluate specific blockers
3. **Phase 7-8 failure:** Keep TS in parallel, add `--legacy` flag for Go TUI
4. **Post-cutover issues:**
   - `git revert` the cutover commit
   - Restore Go TUI as primary
   - TS TUI becomes `--experimental`

### 11.3 Rollback Compatibility

**Session format preservation ensures:**
- Sessions created in TS TUI can be opened in Go TUI
- Sessions created in Go TUI can be opened in TS TUI
- No data migration required for rollback

**Rollback rehearsal (required before Phase 8):**
1. Create session in TS TUI, add 10+ messages
2. Stop TS TUI
3. Resume same session in Go TUI
4. Verify: session loads, cost preserved, can continue conversation

Keep Go frontend in `deprecated/` for minimum 90 days post-cutover.

---

## 12. Resolved Questions

Previously open questions, now resolved:

| Question | Resolution | Rationale |
|----------|------------|-----------|
| **Bundling strategy** | Single file with esbuild | Faster startup, simpler deployment |
| **Session file format** | **Keep Go format exactly** | Enables rollback without data migration |
| **Restart policy** | Same exponential backoff as Go | Proven approach, consistency |
| **Debug mode** | Deprecate debug panel, use Node.js debugging | Standard tooling, less code |
| **Theming** | Hardcode colors initially | Can add config later if needed |

### 12.1 Session Format Decision (Critical)

**Decision:** Preserve Go session format exactly.

**Go session file format:**
```json
{
  "id": "uuid-here",
  "name": "optional-name",
  "created_at": "2026-02-01T10:00:00Z",
  "last_used": "2026-02-01T11:30:00Z",
  "cost": 0.42,
  "tool_calls": 127
}
```

**What this means:**
- TS TUI reads/writes this exact format
- Messages are NOT stored in session files (in-memory only)
- Claude CLI handles conversation persistence via its own session mechanism
- Rollback to Go TUI works without data migration

**What this does NOT mean:**
- We can't add TS-specific metadata later (use separate file if needed)
- Messages are lost on restart (Claude CLI preserves via `--session`)

---

## 13. Timeline Summary

| Phase | Duration | Running Total | Key Deliverable |
|-------|----------|---------------|-----------------|
| 0: Validation | 2 days | 2 days | SDK verified, Go baseline measured |
| 1: Foundation | 3 days | 5 days | Basic Ink app, hooks work |
| 2: Store | 3 days | 8 days | State management complete |
| 3: Components + Modals | 6 days | 14 days | Full UI, modal system |
| 4: MCP | 4 days | 18 days | Tools working end-to-end |
| 5: Agents | 3 days | 21 days | Tree visualization |
| 6: Session | 2 days | 23 days | Persistence, restart |
| 7: Polish | 4 days | 27 days | Telemetry, testing, performance |
| 8: Cutover | 2 days | **29 days** | Production ready |

**Note:** 20% buffer built into estimates. Excludes weekends.

---

## Appendix A: Quick Reference

### A.1 Commands

```bash
# Development
cd packages/tui
npm install
npm run dev

# Production build
npm run build
./bin/gofortress-tui

# Testing
npm test
npm run test:integration
npm run typecheck
npm run lint

# With flags
gofortress-tui --session abc123
gofortress-tui --list
gofortress-tui --verbose
gofortress-tui --legacy  # Falls back to Go TUI
```

### A.2 Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `XDG_DATA_HOME` | Telemetry location | `~/.local/share` |
| `GOFORTRESS_SOCKET` | Legacy (not used) | - |
| `DEBUG` | Enable debug logging | `false` |
| `NO_COLOR` | Disable colors | `false` |

### A.3 Key Bindings

| Key | Action |
|-----|--------|
| Tab | Switch panel focus |
| Escape | Cancel modal / Exit |
| Enter | Submit input |
| Up/Down | Navigate tree/list |
| Ctrl+C | Force quit |
| Ctrl+L | Clear screen |

---

## Appendix B: Architecture Decision Records

### ADR-001: React/Ink over raw terminal

**Decision:** Use Ink (React for terminals) instead of raw ANSI codes or blessed.

**Rationale:**
- Component model matches mental model
- Reusable primitives
- Active maintenance
- Testing library available

**Consequences:**
- Node.js runtime required
- ~5MB dependency footprint
- Learning curve for non-React developers

### ADR-002: Zustand over Redux/MobX

**Decision:** Use Zustand for state management.

**Rationale:**
- Zero boilerplate
- TypeScript-first
- Works with React hooks naturally
- Small bundle size (~1KB)

**Consequences:**
- Less structured than Redux
- No time-travel debugging out of box
- Team must learn Zustand patterns

### ADR-003: In-process MCP over IPC

**Decision:** Run MCP server in-process using SDK, not as separate binary.

**Rationale:**
- Eliminates callback server (230 LOC)
- Removes socket complexity
- Tool handlers can directly access React state
- SDK manages lifecycle

**Consequences:**
- Tighter coupling between TUI and MCP
- Can't run MCP server standalone (not needed)
- Must ensure tool handlers are non-blocking

### ADR-004: Preserve Go Session Format

**Decision:** Keep session file format identical to Go implementation.

**Rationale:**
- Enables rollback without data migration
- Reduces migration risk
- Simpler implementation (no converter needed)

**Consequences:**
- Can't store additional TS-specific metadata in session file
- Messages remain in-memory only (Claude CLI handles persistence)

---

## Appendix C: Review History

### Staff Architect Review (2026-02-01)

**Reviewer:** staff-architect-critical-review (Sonnet)
**Overall Assessment:** CONCERN (multiple issues identified)

**Showstoppers Identified:**
1. S1: SDK API Verification - APIs assumed without proof
2. S2: Session Format Decision - Format change blocks rollback
3. S3: Phase Dependency Cycle - Phase 2 needs Phase 6

**Yellow Flags:**
- Y1: Go Hook Compatibility - No testing strategy
- Y2: JSONL Incremental Read - No offset tracking design
- Y3: Duration Estimates - 30-50% underestimated
- Y4: Terminal Compatibility - No test matrix
- Y5: Rollback Data Loss - Tied to S2

### Einstein Resolution (2026-02-01)

**Reviewer:** Einstein (Opus)
**Assessment:** Plan viable with amendments

**Resolutions:**
- S1: **NOT a showstopper** - SDK APIs exist per fetched documentation. Add Phase 0 verification spike (2 days).
- S2: **Resolved** - Adopt Option A: Keep Go session format exactly. No migration needed.
- S3: **Resolved** - Reorder phases: Merge Modals into Phase 3, MCP becomes Phase 4.

**Amendments Applied:**
- Added Phase 0: Validation Sprint (2 days)
- Reordered phases to fix dependency cycle
- Adopted session format Option A
- Added JSONL offset-tracking reader design
- Added terminal compatibility matrix
- Added 20% buffer to duration estimates
- Expanded risk register with new risks
- Added rollback triggers and rehearsal requirement

---

*End of document. Ready for /plan processing.*
