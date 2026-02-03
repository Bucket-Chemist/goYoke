/**
 * Event type definitions for Claude Agent SDK query events
 * These events are emitted during async iteration of query() results
 */

// System event - session initialization with metadata
export interface SystemEvent {
  type: "system";
  session_id: string;
  model: string;
  tools?: Array<{
    name: string;
    description?: string;
  }>;
  mcp_servers?: string[];
  agents?: Array<{
    id: string;
    model: string;
    tier: "haiku" | "sonnet" | "opus";
    description?: string;
  }>;
  skills?: string[];
}

// Assistant event - streaming or complete message from Claude
export interface AssistantEvent {
  type: "assistant";
  message: string;
  parent_tool_use_id?: string;
  session_id: string;
  content_blocks?: Array<{
    type: "text" | "tool_use" | "tool_result";
    text?: string;
    id?: string;
    name?: string;
    input?: Record<string, unknown>;
  }>;
}

// Result event - final outcome with usage statistics
export interface ResultEvent {
  type: "result";
  is_error: boolean;
  error_message?: string;
  duration_ms: number;
  num_turns: number;
  total_cost_usd: number;
  usage: {
    input_tokens: number;
    output_tokens: number;
    total_tokens: number;
  };
}

// Union type for all query events
export type QueryEvent = SystemEvent | AssistantEvent | ResultEvent;

// Error classification for better error handling
export type ErrorType =
  | "network"
  | "auth"
  | "rate_limit"
  | "invalid_request"
  | "server_error"
  | "timeout"
  | "unknown";

export interface ClassifiedError {
  type: ErrorType;
  message: string;
  originalError?: unknown;
  retryable: boolean;
}
