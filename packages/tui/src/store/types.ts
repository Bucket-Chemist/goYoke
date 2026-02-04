/**
 * Type definitions for Zustand store
 * Central state management for TUI application
 */

// Content block types (matching Anthropic SDK structure)
export interface TextContent {
  type: "text";
  text: string;
}

export interface ToolUseContent {
  type: "tool_use";
  id: string;
  name: string;
  input: Record<string, unknown>;
}

export interface ToolResultContent {
  type: "tool_result";
  tool_use_id: string;
  content: string;
  is_error?: boolean;
}

export type ContentBlock = TextContent | ToolUseContent | ToolResultContent;

// Message interface
export interface Message {
  id: string;
  role: "user" | "assistant" | "system";
  content: ContentBlock[];
  partial: boolean;
  timestamp: number;
}

// Agent interface with tree structure
export interface Agent {
  id: string;
  parentId: string | null;
  model: string;
  tier: "haiku" | "sonnet" | "opus";
  status: "spawning" | "running" | "complete" | "error";
  description?: string;
  startTime: number;
  endTime?: number;
  tokenUsage?: {
    input: number;
    output: number;
  };
}

// Session data (matches Go format exactly)
export interface SessionData {
  id: string;
  name?: string;
  created_at: string; // ISO8601
  last_used: string; // ISO8601
  cost: number;
  tool_calls: number;
}

// Token count structure
export interface TokenCount {
  input: number;
  output: number;
}

// Messages slice
export interface MessagesSlice {
  messages: Message[];
  addMessage: (msg: Omit<Message, "id" | "timestamp">) => void;
  updateLastMessage: (content: ContentBlock[]) => void;
  clearMessages: () => void;
}

// Agents slice
export interface AgentsSlice {
  agents: Record<string, Agent>;
  selectedAgentId: string | null;
  rootAgentId: string | null;
  addAgent: (agent: Omit<Agent, "startTime">) => void;
  updateAgent: (id: string, data: Partial<Agent>) => void;
  selectAgent: (id: string | null) => void;
  getAgentChildren: (id: string) => Agent[];
  clearAgents: () => void;
}

// Session slice (Go format compatible)
export interface SessionSlice {
  sessionId: string | null;
  totalCost: number;
  tokenCount: TokenCount;
  updateSession: (data: Partial<SessionData>) => void;
  incrementCost: (cost: number) => void;
  addTokens: (tokens: Partial<TokenCount>) => void;
  clearSession: () => void;
}

// UI slice
export interface UISlice {
  streaming: boolean;
  focusedPanel: "claude" | "agents";
  setStreaming: (streaming: boolean) => void;
  setFocusedPanel: (panel: "claude" | "agents") => void;
}

// Input history slice (ephemeral - not persisted)
export interface InputSlice {
  inputHistory: string[]; // Max 100 entries
  inputHistoryIndex: number; // -1 = not navigating, 0+ = position
  addToHistory: (input: string) => void;
  navigateHistory: (direction: "up" | "down") => string | null;
  resetHistoryIndex: () => void;
  clearHistory: () => void;
}

// Modal slice (ephemeral - not persisted)
export interface ModalSlice {
  modalQueue: Array<{
    id: string;
    type: "ask" | "confirm" | "input" | "select";
    payload: unknown;
    resolve: (response: ModalResponse) => void;
    reject: (error: Error) => void;
    timeout?: number;
    timeoutId?: NodeJS.Timeout;
  }>;
  enqueue: <T>(
    request: Omit<
      {
        id: string;
        type: "ask" | "confirm" | "input" | "select";
        payload: T;
        resolve: (response: ModalResponse) => void;
        reject: (error: Error) => void;
        timeout?: number;
        timeoutId?: NodeJS.Timeout;
      },
      "id" | "resolve" | "reject" | "timeoutId"
    >
  ) => Promise<ModalResponse>;
  dequeue: (id: string, response: ModalResponse) => void;
  cancel: (id: string) => void;
}

export type ModalResponse =
  | { type: "ask"; value: string }
  | { type: "confirm"; confirmed: boolean; cancelled: boolean }
  | { type: "input"; value: string }
  | { type: "select"; selected: string; index: number };

// Import telemetry types
import type { TelemetrySlice } from "./slices/telemetry.js";
export type { TelemetrySlice, RoutingDecision, Handoff, SharpEdge } from "./slices/telemetry.js";

// Combined store type
export type Store = MessagesSlice &
  AgentsSlice &
  SessionSlice &
  UISlice &
  InputSlice &
  ModalSlice &
  TelemetrySlice;
