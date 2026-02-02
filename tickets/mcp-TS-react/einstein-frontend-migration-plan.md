# GOgent-Fortress Frontend Migration: Architectural Assessment & Plan

**Generated:** 2026-02-01
**Author:** Einstein Analysis (Opus)
**Purpose:** Detailed guide for `/plan` command processing

---

## Executive Summary

This document provides a comprehensive architectural map for migrating the GOgent-Fortress frontend from Go (Bubbletea/lipgloss) to TypeScript/React while preserving the high-performance Go hooks infrastructure.

**Recommendation:** TypeScript with React + Ink (terminal React renderer) for the TUI, and the official Claude Agent SDK (`@anthropic-ai/claude-agent-sdk`) for MCP integration.

---

## Part 1: Current Architecture Analysis

### 1.1 What Exists Today

```
┌─────────────────────────────────────────────────────────────────┐
│                    CURRENT GO ARCHITECTURE                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │           cmd/gofortress/main.go                          │   │
│  │           (TUI Entry Point - 300 LOC)                     │   │
│  │  • Socket cleanup                                         │   │
│  │  • Callback server start                                  │   │
│  │  • MCP server spawn                                       │   │
│  │  • Claude subprocess orchestration                        │   │
│  │  • Bubbletea program initialization                       │   │
│  └──────────────────────────────────────────────────────────┘   │
│                              │                                   │
│           ┌──────────────────┼──────────────────┐               │
│           ▼                  ▼                  ▼               │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────┐     │
│  │internal/tui/│    │internal/cli/│    │cmd/gofortress-  │     │
│  │  layout/    │    │             │    │  mcp-server/    │     │
│  │  claude/    │    │subprocess.go│    │                 │     │
│  │  agents/    │    │events.go    │    │MCP stdio server │     │
│  │  telemetry/ │    │streams.go   │    │4 tools:         │     │
│  │             │    │restart.go   │    │ • ask_user      │     │
│  │~2000 LOC    │    │~1500 LOC    │    │ • confirm_action│     │
│  └─────────────┘    └─────────────┘    │ • request_input │     │
│                                         │ • select_option │     │
│                                         │  ~250 LOC       │     │
│                                         └─────────────────┘     │
│                              │                                   │
│                              ▼                                   │
│                    ┌─────────────────┐                          │
│                    │internal/callback│                          │
│                    │                 │                          │
│                    │Unix socket HTTP │                          │
│                    │for TUI↔MCP      │                          │
│                    │bridge ~230 LOC  │                          │
│                    └─────────────────┘                          │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 Component Inventory

| Component | Location | LOC | Pain Level | Migration Priority |
|-----------|----------|-----|------------|-------------------|
| TUI Layout | internal/tui/layout/ | ~350 | Medium | HIGH |
| Claude Panel | internal/tui/claude/ | ~600 | HIGH | HIGH |
| Agent Tree | internal/tui/agents/ | ~400 | Medium | HIGH |
| Modal System | internal/tui/claude/modal.go | ~200 | HIGH | HIGH |
| CLI Subprocess | internal/cli/ | ~1500 | Medium | MEDIUM |
| MCP Server | cmd/gofortress-mcp-server/ | ~250 | HIGH | HIGH |
| Callback Bridge | internal/callback/ | ~230 | HIGH | ELIMINATE |
| Event Parsing | internal/cli/events.go | ~250 | HIGH | MEDIUM |

### 1.3 Identified Pain Points

1. **Manual JSON marshaling everywhere** - Go requires explicit struct definitions for every event type
2. **Two-stage event parsing** - Parse once to get type, then parse again for specific fields
3. **Complex state machine** - 8+ state variables in PanelModel with multiple sources of truth
4. **Custom modal system** - Built from scratch, no library support
5. **Socket bridge complexity** - Unix socket HTTP server just to communicate between processes
6. **Lifecycle fragility** - Multiple shutdown paths, signal propagation issues

---

## Part 2: Target Architecture

### 2.1 Recommended Stack

| Layer | Technology | Rationale |
|-------|------------|-----------|
| **TUI Renderer** | [Ink](https://github.com/vadimdemedes/ink) | React for terminals, mature ecosystem |
| **State Management** | Zustand or Jotai | Lightweight, TypeScript-first |
| **MCP Integration** | `@anthropic-ai/claude-agent-sdk` | Official SDK, native TypeScript |
| **Build Tool** | tsx + esbuild | Fast builds, native TS execution |
| **Go Hooks** | UNCHANGED | Keep all `gogent-*` binaries |

### 2.2 Why Ink Over Alternatives

| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **Ink (React)** | React mental model, component reuse, huge ecosystem, active maintenance | Requires Node.js runtime | ✅ RECOMMENDED |
| Blessed | Pure Node, no React | Dated, minimal maintenance | ❌ |
| Terminal-kit | Full-featured | Complex API, learning curve | ❌ |
| Electron | Full GUI capabilities | Overkill for TUI, heavy | ❌ |

### 2.3 Target Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                    NEW TYPESCRIPT ARCHITECTURE                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │              packages/tui/src/index.tsx                         │ │
│  │              (Main Entry Point)                                 │ │
│  │                                                                 │ │
│  │  import { render } from 'ink';                                  │ │
│  │  import { App } from './App';                                   │ │
│  │  render(<App />);                                               │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                              │                                       │
│           ┌──────────────────┼──────────────────┐                   │
│           ▼                  ▼                  ▼                   │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────────┐   │
│  │ components/     │ │ hooks/          │ │ mcp/                │   │
│  │                 │ │                 │ │                     │   │
│  │ <Layout />      │ │ useClaudeQuery  │ │ createSdkMcpServer  │   │
│  │ <ClaudePanel /> │ │ useAgentTree    │ │                     │   │
│  │ <AgentTree />   │ │ useModalState   │ │ Tools:              │   │
│  │ <Modal />       │ │ useTelemetry    │ │ • ask_user          │   │
│  │ <Banner />      │ │                 │ │ • confirm_action    │   │
│  │ <Detail />      │ │                 │ │ • request_input     │   │
│  └─────────────────┘ └─────────────────┘ │ • select_option     │   │
│                                          └─────────────────────┘   │
│                              │                                       │
│                              ▼                                       │
│                    ┌─────────────────────┐                          │
│                    │ store/              │                          │
│                    │                     │                          │
│                    │ Zustand store:      │                          │
│                    │ • messages[]        │                          │
│                    │ • agents{}          │                          │
│                    │ • sessionState      │                          │
│                    │ • modalQueue        │                          │
│                    └─────────────────────┘                          │
│                                                                      │
│  ════════════════════════════════════════════════════════════════   │
│                     PROCESS BOUNDARY                                 │
│  ════════════════════════════════════════════════════════════════   │
│                                                                      │
│                    ┌─────────────────────┐                          │
│                    │ GO HOOKS (UNCHANGED)│                          │
│                    │                     │                          │
│                    │ • gogent-load-context                          │
│                    │ • gogent-validate                              │
│                    │ • gogent-sharp-edge                            │
│                    │ • gogent-archive                               │
│                    │ • gogent-agent-endstate                        │
│                    └─────────────────────┘                          │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 2.4 Key Architectural Changes

| Current (Go) | Target (TypeScript) | Improvement |
|--------------|---------------------|-------------|
| Unix socket callback server | **ELIMINATED** - SDK handles MCP in-process | -230 LOC, -1 process |
| Manual NDJSON parsing | SDK streaming events | Native async iteration |
| Two-stage event parsing | TypeScript discriminated unions | Type-safe at compile time |
| Bubbletea message passing | React state + hooks | Familiar patterns |
| Custom modal system | React components + portals | Composable, testable |
| Process spawning | SDK's `query()` function | Managed lifecycle |

---

## Part 3: Migration Strategy

### 3.1 Phased Approach

```
Phase 1: Foundation (1 week)
├─► Set up TypeScript project structure
├─► Install dependencies (ink, claude-agent-sdk, zustand)
├─► Create basic App shell with Ink
└─► Verify Go hooks still work via subprocess

Phase 2: MCP Integration (1 week)
├─► Implement MCP server with Agent SDK
├─► Define 4 tools (ask_user, confirm, input, select)
├─► Test tool invocation from Claude
└─► Remove Go MCP server + callback bridge

Phase 3: TUI Components (2 weeks)
├─► Port Layout component
├─► Port Claude Panel (messages, input)
├─► Port Agent Tree visualization
├─► Port Modal system
└─► Port Banner/telemetry

Phase 4: State & Events (1 week)
├─► Implement Zustand store
├─► Wire up event streaming
├─► Handle session persistence
└─► Implement restart logic

Phase 5: Polish & Cutover (1 week)
├─► Integration testing
├─► Performance comparison
├─► Documentation
└─► Archive Go frontend code
```

### 3.2 Project Structure

```
packages/
├── tui/                          # New TypeScript TUI
│   ├── package.json
│   ├── tsconfig.json
│   ├── src/
│   │   ├── index.tsx             # Entry point
│   │   ├── App.tsx               # Root component
│   │   ├── components/
│   │   │   ├── Layout.tsx
│   │   │   ├── ClaudePanel.tsx
│   │   │   ├── AgentTree.tsx
│   │   │   ├── Modal.tsx
│   │   │   ├── Banner.tsx
│   │   │   └── Detail.tsx
│   │   ├── hooks/
│   │   │   ├── useClaudeQuery.ts
│   │   │   ├── useAgentTree.ts
│   │   │   ├── useModalQueue.ts
│   │   │   └── useTelemetry.ts
│   │   ├── mcp/
│   │   │   ├── server.ts         # MCP tool definitions
│   │   │   └── tools.ts          # Tool implementations
│   │   ├── store/
│   │   │   ├── index.ts
│   │   │   ├── messages.ts
│   │   │   ├── agents.ts
│   │   │   └── session.ts
│   │   └── types/
│   │       ├── events.ts
│   │       └── mcp.ts
│   └── bin/
│       └── gofortress-tui        # Compiled/bundled executable
│
├── hooks/                        # EXISTING - unchanged
│   └── cmd/
│       ├── gogent-load-context/
│       ├── gogent-validate/
│       ├── gogent-sharp-edge/
│       └── gogent-archive/
│
└── deprecated/                   # Archive Go frontend
    ├── cmd/gofortress/
    ├── cmd/gofortress-mcp-server/
    └── internal/
        ├── tui/
        ├── cli/
        └── callback/
```

---

## Part 4: Implementation Details

### 4.1 MCP Server Implementation

```typescript
// packages/tui/src/mcp/server.ts

import { createSdkMcpServer, tool } from "@anthropic-ai/claude-agent-sdk";
import { z } from "zod";
import { useModalStore } from "../store/modal";

export const mcpServer = createSdkMcpServer({
  name: "gofortress-interactive",
  version: "1.0.0",
  tools: [
    tool(
      "ask_user",
      "Ask the user a question with optional predefined options",
      {
        message: z.string().describe("The question to ask"),
        options: z.array(z.string()).optional().describe("Predefined answer options"),
        default: z.string().optional().describe("Default answer")
      },
      async (args) => {
        // This runs in-process - no socket needed!
        const response = await useModalStore.getState().showQuestion({
          type: "ask",
          message: args.message,
          options: args.options,
          default: args.default
        });

        return {
          content: [{
            type: "text",
            text: response.answer
          }]
        };
      }
    ),

    tool(
      "confirm_action",
      "Request user confirmation for an action",
      {
        action: z.string().describe("Description of the action to confirm"),
        destructive: z.boolean().optional().describe("Whether action is destructive")
      },
      async (args) => {
        const response = await useModalStore.getState().showConfirm({
          action: args.action,
          destructive: args.destructive ?? false
        });

        return {
          content: [{
            type: "text",
            text: JSON.stringify({
              confirmed: response.confirmed,
              cancelled: response.cancelled
            })
          }]
        };
      }
    ),

    tool(
      "request_input",
      "Request text input from the user",
      {
        prompt: z.string().describe("Input prompt"),
        placeholder: z.string().optional().describe("Placeholder text")
      },
      async (args) => {
        const response = await useModalStore.getState().showInput({
          prompt: args.prompt,
          placeholder: args.placeholder
        });

        return {
          content: [{
            type: "text",
            text: response.value
          }]
        };
      }
    ),

    tool(
      "select_option",
      "Let user select from a list of options",
      {
        message: z.string().describe("Selection prompt"),
        options: z.array(z.object({
          label: z.string(),
          value: z.string()
        })).describe("Available options")
      },
      async (args) => {
        const response = await useModalStore.getState().showSelect({
          message: args.message,
          options: args.options
        });

        return {
          content: [{
            type: "text",
            text: JSON.stringify({
              selected: response.selected,
              index: response.index
            })
          }]
        };
      }
    )
  ]
});
```

### 4.2 Main Query Hook

```typescript
// packages/tui/src/hooks/useClaudeQuery.ts

import { query, ClaudeEvent } from "@anthropic-ai/claude-agent-sdk";
import { useCallback, useEffect } from "react";
import { useStore } from "../store";
import { mcpServer } from "../mcp/server";

export function useClaudeQuery() {
  const { addMessage, setStreaming, updateCost, addAgent } = useStore();

  const sendMessage = useCallback(async (content: string) => {
    setStreaming(true);

    // Create streaming input
    async function* generateMessages() {
      yield {
        type: "user" as const,
        message: {
          role: "user" as const,
          content: [{ type: "text", text: content }]
        }
      };
    }

    try {
      for await (const event of query({
        prompt: generateMessages(),
        options: {
          mcpServers: {
            "gofortress-interactive": mcpServer
          },
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
  }, []);

  const handleEvent = (event: ClaudeEvent) => {
    switch (event.type) {
      case "assistant":
        addMessage({
          role: "assistant",
          content: event.message.content,
          partial: event.partial
        });
        break;

      case "result":
        updateCost(event.total_cost_usd);
        break;

      case "task":
        if (event.subtype === "spawn") {
          addAgent({
            id: event.agent_id,
            parent: event.parent_id,
            model: event.model,
            status: "running"
          });
        }
        break;
    }
  };

  return { sendMessage };
}
```

### 4.3 Layout Component

```typescript
// packages/tui/src/components/Layout.tsx

import React from "react";
import { Box, Text, useInput, useApp } from "ink";
import { ClaudePanel } from "./ClaudePanel";
import { AgentTree } from "./AgentTree";
import { Detail } from "./Detail";
import { Banner } from "./Banner";
import { Modal } from "./Modal";
import { useStore } from "../store";

export function Layout() {
  const { focusedPanel, setFocusedPanel, modalQueue } = useStore();
  const { exit } = useApp();

  useInput((input, key) => {
    if (key.escape) {
      exit();
    }
    if (key.tab) {
      setFocusedPanel(focusedPanel === "claude" ? "agents" : "claude");
    }
  });

  return (
    <Box flexDirection="column" height="100%">
      {/* Banner */}
      <Banner />

      {/* Main content */}
      <Box flexDirection="row" flexGrow={1}>
        {/* Left panel: Claude conversation (70%) */}
        <Box width="70%" flexDirection="column">
          <ClaudePanel focused={focusedPanel === "claude"} />
        </Box>

        {/* Right panel: Agent tree + detail (30%) */}
        <Box width="30%" flexDirection="column">
          <Box height="60%">
            <AgentTree focused={focusedPanel === "agents"} />
          </Box>
          <Box height="40%">
            <Detail />
          </Box>
        </Box>
      </Box>

      {/* Modal overlay */}
      {modalQueue.length > 0 && (
        <Modal modal={modalQueue[0]} />
      )}
    </Box>
  );
}
```

### 4.4 Zustand Store

```typescript
// packages/tui/src/store/index.ts

import { create } from "zustand";

interface Message {
  role: "user" | "assistant" | "system";
  content: any[];
  partial?: boolean;
  timestamp: number;
}

interface Agent {
  id: string;
  parent?: string;
  model: string;
  status: "spawning" | "running" | "complete" | "error";
  startTime: number;
  endTime?: number;
}

interface ModalRequest {
  id: string;
  type: "ask" | "confirm" | "input" | "select";
  payload: any;
  resolve: (response: any) => void;
}

interface Store {
  // Messages
  messages: Message[];
  addMessage: (msg: Omit<Message, "timestamp">) => void;

  // Streaming state
  streaming: boolean;
  setStreaming: (s: boolean) => void;

  // Session
  sessionId: string | null;
  cost: number;
  updateCost: (c: number) => void;

  // Agents
  agents: Record<string, Agent>;
  selectedAgent: string | null;
  addAgent: (a: Omit<Agent, "startTime">) => void;
  updateAgent: (id: string, updates: Partial<Agent>) => void;
  selectAgent: (id: string | null) => void;

  // Focus
  focusedPanel: "claude" | "agents";
  setFocusedPanel: (p: "claude" | "agents") => void;

  // Modals
  modalQueue: ModalRequest[];
  showQuestion: (payload: any) => Promise<any>;
  showConfirm: (payload: any) => Promise<any>;
  showInput: (payload: any) => Promise<any>;
  showSelect: (payload: any) => Promise<any>;
  resolveModal: (id: string, response: any) => void;
}

export const useStore = create<Store>((set, get) => ({
  messages: [],
  addMessage: (msg) => set((s) => ({
    messages: msg.partial
      ? [...s.messages.slice(0, -1), { ...msg, timestamp: Date.now() }]
      : [...s.messages, { ...msg, timestamp: Date.now() }]
  })),

  streaming: false,
  setStreaming: (streaming) => set({ streaming }),

  sessionId: null,
  cost: 0,
  updateCost: (cost) => set({ cost }),

  agents: {},
  selectedAgent: null,
  addAgent: (a) => set((s) => ({
    agents: { ...s.agents, [a.id]: { ...a, startTime: Date.now() } }
  })),
  updateAgent: (id, updates) => set((s) => ({
    agents: { ...s.agents, [id]: { ...s.agents[id], ...updates } }
  })),
  selectAgent: (selectedAgent) => set({ selectedAgent }),

  focusedPanel: "claude",
  setFocusedPanel: (focusedPanel) => set({ focusedPanel }),

  modalQueue: [],
  showQuestion: (payload) => new Promise((resolve) => {
    const id = crypto.randomUUID();
    set((s) => ({
      modalQueue: [...s.modalQueue, { id, type: "ask", payload, resolve }]
    }));
  }),
  showConfirm: (payload) => new Promise((resolve) => {
    const id = crypto.randomUUID();
    set((s) => ({
      modalQueue: [...s.modalQueue, { id, type: "confirm", payload, resolve }]
    }));
  }),
  showInput: (payload) => new Promise((resolve) => {
    const id = crypto.randomUUID();
    set((s) => ({
      modalQueue: [...s.modalQueue, { id, type: "input", payload, resolve }]
    }));
  }),
  showSelect: (payload) => new Promise((resolve) => {
    const id = crypto.randomUUID();
    set((s) => ({
      modalQueue: [...s.modalQueue, { id, type: "select", payload, resolve }]
    }));
  }),
  resolveModal: (id, response) => {
    const modal = get().modalQueue.find((m) => m.id === id);
    if (modal) {
      modal.resolve(response);
      set((s) => ({
        modalQueue: s.modalQueue.filter((m) => m.id !== id)
      }));
    }
  }
}));
```

---

## Part 5: Integration with Go Hooks

### 5.1 Hook Communication Pattern

The Go hooks communicate via environment variables and file system:

```
┌────────────────────────────────────────────────────────────────┐
│                  TypeScript TUI Process                         │
│                                                                 │
│  1. Spawns Claude Code as subprocess                            │
│  2. Claude Code loads hooks via its own mechanism               │
│  3. Hooks write to ~/.claude/tmp/ and XDG_DATA_HOME             │
│  4. TUI reads from those locations for telemetry                │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
          │
          │ Claude Code subprocess
          ▼
┌────────────────────────────────────────────────────────────────┐
│                    Claude Code Process                          │
│                                                                 │
│  Hook events:                                                   │
│  • SessionStart → gogent-load-context                           │
│  • PreToolUse → gogent-validate                                 │
│  • PostToolUse → gogent-sharp-edge                              │
│  • SessionEnd → gogent-archive                                  │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
          │
          │ File system (not IPC)
          ▼
┌────────────────────────────────────────────────────────────────┐
│                    Shared File Locations                        │
│                                                                 │
│  ~/.claude/tmp/scout_metrics.json                               │
│  ~/.claude/memory/handoffs.jsonl                                │
│  $XDG_DATA_HOME/gogent/routing-decisions.jsonl         │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
```

### 5.2 Telemetry Watcher Hook

```typescript
// packages/tui/src/hooks/useTelemetry.ts

import { useEffect } from "react";
import { watch } from "chokidar";
import { readFile } from "fs/promises";
import { useStore } from "../store";

const TELEMETRY_PATHS = {
  routingDecisions: `${process.env.XDG_DATA_HOME}/gogent/routing-decisions.jsonl`,
  handoffs: `${process.env.HOME}/.claude/memory/handoffs.jsonl`,
  scoutMetrics: `${process.env.HOME}/.claude/tmp/scout_metrics.json`
};

export function useTelemetry() {
  const { updateTelemetry } = useStore();

  useEffect(() => {
    const watcher = watch(Object.values(TELEMETRY_PATHS), {
      persistent: true,
      ignoreInitial: false
    });

    watcher.on("change", async (path) => {
      try {
        const content = await readFile(path, "utf-8");

        if (path.endsWith(".jsonl")) {
          // Parse last line for latest entry
          const lines = content.trim().split("\n");
          const latest = JSON.parse(lines[lines.length - 1]);
          updateTelemetry(path, latest);
        } else {
          // Parse as single JSON
          updateTelemetry(path, JSON.parse(content));
        }
      } catch (err) {
        // File may be in-flight, ignore
      }
    });

    return () => watcher.close();
  }, []);
}
```

---

## Part 6: Dependencies & package.json

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
    "build": "esbuild src/index.tsx --bundle --platform=node --outfile=dist/index.js",
    "start": "node dist/index.js",
    "typecheck": "tsc --noEmit"
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
    "commander": "^12.0.0"
  },
  "devDependencies": {
    "@types/node": "^22.0.0",
    "@types/react": "^18.3.0",
    "esbuild": "^0.24.0",
    "tsx": "^4.19.0",
    "typescript": "^5.6.0"
  }
}
```

---

## Part 7: Risk Assessment & Mitigations

### 7.1 Risks

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Agent SDK API changes | High | Low | Pin SDK version, monitor changelog |
| Ink rendering quirks | Medium | Medium | Extensive terminal testing matrix |
| Node.js runtime dependency | Low | Certain | Bundle with pkg or use Bun |
| Performance regression | Medium | Low | Benchmark against Go implementation |
| Hook integration failures | High | Low | Integration test suite |

### 7.2 Rollback Plan

1. Keep Go frontend in `deprecated/` for 3 months
2. Maintain feature parity checklist
3. Add `--legacy-tui` flag to fallback during transition
4. Document any breaking changes

---

## Part 8: Success Criteria

### 8.1 Functional Requirements

- [ ] All 4 MCP tools work (ask, confirm, input, select)
- [ ] Claude conversation displays correctly
- [ ] Agent tree shows delegation hierarchy
- [ ] Session persistence works
- [ ] Restart logic functions
- [ ] Keyboard navigation intact
- [ ] Mouse support (click panels)
- [ ] Cost/token tracking accurate

### 8.2 Non-Functional Requirements

- [ ] Startup time < 500ms (Go was ~200ms)
- [ ] Memory usage < 100MB (Go was ~30MB)
- [ ] No visible input lag
- [ ] Works in: iTerm2, Alacritty, Kitty, macOS Terminal, Windows Terminal

### 8.3 Code Quality

- [ ] TypeScript strict mode enabled
- [ ] No `any` types except at boundaries
- [ ] Test coverage > 80%
- [ ] ESLint clean
- [ ] Documentation complete

---

## Part 9: Next Steps for /plan

This document is ready for `/plan` processing. Recommended breakdown:

1. **Ticket: Project Scaffolding**
   - Create packages/tui directory structure
   - Initialize package.json, tsconfig.json
   - Set up build pipeline (tsx, esbuild)

2. **Ticket: MCP Server Implementation**
   - Implement 4 tools with Agent SDK
   - Unit test each tool
   - Integration test with Claude

3. **Ticket: Core Components**
   - Layout, ClaudePanel, Banner
   - Input handling, message display
   - Viewport scrolling

4. **Ticket: Agent Visualization**
   - AgentTree component
   - Detail panel
   - Tree navigation

5. **Ticket: Modal System**
   - Modal component
   - Queue management in store
   - All 4 modal types

6. **Ticket: State Management**
   - Zustand store implementation
   - Event handling
   - Session persistence

7. **Ticket: Telemetry Integration**
   - File watchers for Go hook outputs
   - Cost tracking
   - Handoff display

8. **Ticket: Polish & Testing**
   - Terminal compatibility testing
   - Performance benchmarking
   - Documentation

---

## Appendix A: Reference Links

- [Ink Documentation](https://github.com/vadimdemedes/ink)
- [Claude Agent SDK](https://platform.claude.com/docs/en/agent-sdk)
- [Zustand](https://github.com/pmndrs/zustand)
- [MCP Protocol](https://modelcontextprotocol.io)

---

## Appendix B: Current vs Target LOC Estimate

| Component | Current Go LOC | Target TS LOC | Change |
|-----------|----------------|---------------|--------|
| Entry point | 300 | 50 | -250 |
| TUI components | 2000 | 800 | -1200 |
| MCP server | 250 | 150 | -100 |
| Callback bridge | 230 | 0 | -230 |
| CLI subprocess | 1500 | 200 | -1300 |
| **Total** | **4280** | **1200** | **-72%** |

*Note: LOC reduction comes from SDK handling complexity, React patterns, and eliminating the callback bridge entirely.*
