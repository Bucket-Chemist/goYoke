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

/**
 * Status values - union of V1 and V2
 */
export type AgentStatus =
  | "queued"      // New: waiting to spawn
  | "spawning"   // V1: CLI starting
  | "running"    // V1: executing
  | "streaming"  // New: producing output
  | "complete"   // V1: finished successfully
  | "error"      // V1: failed
  | "timeout";   // New: exceeded time limit

/**
 * Spawn method discriminator
 */
export type SpawnMethod = "task" | "mcp-cli";

// Agent interface with tree structure
/**
 * Legacy Agent interface (V1) - DO NOT MODIFY existing fields.
 * Kept for backward compatibility reference.
 */
export interface AgentV1 {
  id: string;
  parentId: string | null;
  model: string;
  tier: "haiku" | "sonnet" | "opus";
  status: AgentStatus;
  description?: string;
  startTime: number;
  endTime?: number;
  tokenUsage?: {
    input: number;
    output: number;
  };
}

/**
 * Extended Agent interface (V2) - All new fields are OPTIONAL.
 * This maintains backward compatibility with V1.
 */
export interface Agent extends AgentV1 {
  // Hierarchy (optional for V1 compatibility)
  agentType?: string;
  epicId?: string;
  depth?: number;
  childIds?: string[];

  // Spawning metadata (optional)
  spawnMethod?: "task" | "mcp-cli";
  spawnedBy?: string;
  prompt?: string;

  // Process info (for MCP-CLI spawns)
  pid?: number;
  queuedAt?: number;

  // Extended status (compatible with V1 status)
  // V1 status values still valid, these are additions
  // "queued" | "streaming" | "timeout" are new options

  // Output (optional)
  output?: string;
  streamBuffer?: string;
  error?: string;

  // Extended metrics (optional)
  cost?: number;
  turns?: number;
  toolCalls?: number;
}

/**
 * Input for creating a new agent
 */
export interface CreateAgentInput {
  // Required
  model: string;
  tier: "haiku" | "sonnet" | "opus";
  description: string;

  // Optional hierarchy
  parentId?: string | null;
  agentType?: string;
  epicId?: string;
  spawnMethod?: SpawnMethod;
  prompt?: string;
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
  contextWindow: {
    usedTokens: number;      // input + cache_creation + cache_read from last API call
    totalCapacity: number;   // contextWindow from ModelUsage (default 200000)
  };
  permissionMode: string;
  isCompacting: boolean;
  preferredModel: string | null;
  activeModel: string | null; // Actual model from SDK init event
  updateSession: (data: Partial<SessionData>) => void;
  incrementCost: (cost: number) => void;
  addTokens: (tokens: Partial<TokenCount>) => void;
  updateContextWindow: (usedTokens: number, totalCapacity: number) => void;
  setPermissionMode: (mode: string) => void;
  setCompacting: (compacting: boolean) => void;
  setPreferredModel: (model: string | null) => void;
  setActiveModel: (model: string | null) => void;
  isPlanMode: () => boolean;
  clearSession: () => void;
}

// Tab types
export type TabId = "chat" | "agent-config" | "team-config" | "telemetry";

export interface TabDefinition {
  id: TabId;
  label: string;
  shortcutKey: string; // Single lowercase letter for Alt+key
  shortcutIndex: number; // Position of underlined char in label
}

// UI slice
export interface UISlice {
  streaming: boolean;
  focusedPanel: "claude" | "agents";
  rightPanelMode: "agents" | "dashboard" | "settings" | "teams";
  activeTab: TabId;
  interruptQuery: (() => Promise<void>) | null;
  clearPendingMessage: (() => void) | null;
  setStreaming: (streaming: boolean) => void;
  setFocusedPanel: (panel: "claude" | "agents") => void;
  cycleRightPanel: () => void;
  setActiveTab: (tab: TabId) => void;
  setInterruptQuery: (fn: (() => Promise<void>) | null) => void;
  setClearPendingMessage: (fn: (() => void) | null) => void;
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

// Toast notification types
export interface Toast {
  id: string;
  message: string;
  type: "info" | "success" | "warning" | "error";
  createdAt: number;
}

export interface ToastSlice {
  toasts: Toast[];
  addToast: (message: string, type?: Toast["type"]) => void;
  removeToast: (id: string) => void;
}

// Team types - imported from hooks
export interface TeamSummary {
  dir: string;
  name: string;
  workflowType: string;
  status: "pending" | "running" | "completed" | "failed";
  backgroundPid: number | null;
  alive: boolean;
  budgetMax: number;
  budgetRemaining: number;
  startedAt: string | null;
  completedAt: string | null;
  totalCost: number;
  waveCount: number;
  currentWave: number;
  memberCount: number;
  completedMembers: number;
  failedMembers: number;
}

// Full team config structure (matches Go TeamConfig)
export interface TeamConfig {
  team_name: string;
  workflow_type: string;
  project_root: string;
  session_id: string;
  created_at: string;
  budget_max_usd: number;
  budget_remaining_usd: number;
  warning_threshold_usd: number;
  status: string;
  background_pid: number | null;
  started_at: string | null;
  completed_at: string | null;
  waves: TeamWave[];
}

export interface TeamWave {
  wave_number: number;
  description: string;
  members: TeamMember[];
  on_complete_script: string | null;
}

export interface TeamMember {
  name: string;
  agent: string;
  model: string;
  stdin_file: string;
  stdout_file: string;
  status: string;
  process_pid: number | null;
  exit_code: number | null;
  cost_usd: number;
  cost_status: string;
  error_message: string;
  retry_count: number;
  max_retries: number;
  timeout_ms: number;
  started_at: string | null;
  completed_at: string | null;
}

// Teams slice
export interface TeamsSlice {
  teams: TeamSummary[];
  selectedTeamDir: string | null;
  selectedTeamDetail: TeamConfig | null;
  /** @deprecated Use teams.filter(t => t.alive).length instead */
  backgroundTeamCount: number; // Derived for backward compat
  setTeams: (teams: TeamSummary[]) => void;
  selectTeam: (dir: string | null) => void;
  setTeamDetail: (config: TeamConfig | null) => void;
  /** @deprecated Use teams.filter(t => t.alive).length instead */
  setBackgroundTeamCount: (count: number) => void; // Deprecated but kept for compat
}

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
  TelemetrySlice &
  ToastSlice &
  TeamsSlice;
