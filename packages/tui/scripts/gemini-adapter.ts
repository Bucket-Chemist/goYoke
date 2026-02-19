#!/usr/bin/env -S npx tsx
/// <reference types="node" />

import { spawn } from "child_process";
import * as readline from "readline";

// Protocol Mapping
// TUI Model Name -> Gemini CLI Flag
const MODEL_MAPPING: Record<string, string> = {
  "gemini-flash": "gemini-3-flash-preview",
  "gemini-pro": "gemini-3-pro-preview",
  // Default fallback
  "default": "gemini-3-pro-preview",
};

// Types for SDK Messages (Simplified)
interface SDKUserMessage {
  type: "user";
  message: {
    role: "user";
    content: Array<{ type: "text"; text: string }>;
  };
  session_id?: string;
}

// Types for Gemini Events (Simplified)
interface GeminiEvent {
  type: "init" | "message" | "tool_use" | "tool_result" | "result" | "error";
  session_id?: string;
  model?: string;
  role?: string;
  content?: string | any[];
  delta?: boolean;
  tool_name?: string;
  tool_id?: string;
  parameters?: any;
  status?: string;
  output?: string;
  stats?: any;
  timestamp?: string;
}

// Global state
let currentModel = MODEL_MAPPING["default"];
let sessionId: string | null = null;
let currentMessageId: string | null = null;
let messageStarted = false;
let contentBlockStarted = false;
let currentBlockIndex = 0;

// Debug logging
const debug = process.env.DEBUG_ADAPTER === "1";
function log(msg: string, ...args: any[]) {
  if (debug) {
    console.error(`[ADAPTER] ${msg}`, ...args);
  }
}

// Helper to send SDK event
function sendEvent(event: any) {
  // log("Sending event to SDK:", JSON.stringify(event));
  console.log(JSON.stringify(event));
}

// Main Loop
async function main() {
  log("Starting Gemini Adapter...");

  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
    terminal: false,
  });

  rl.on("line", async (line) => {
    try {
      log("Received line from SDK:", line);
      if (!line.trim()) return;

      const message = JSON.parse(line) as SDKUserMessage;

      if (message.type === "user") {
        await handleUserMessage(message);
      }
    } catch (err) {
      log("Error processing input:", err);
    }
  });

  // Handle process exit
  process.on("SIGINT", () => {
    log("Received SIGINT, exiting...");
    process.exit(0);
  });
}

async function handleUserMessage(sdkMsg: SDKUserMessage) {
  // Extract text content
  const textContent = sdkMsg.message.content
    .filter((c: any) => c.type === "text")
    .map((c: any) => c.text)
    .join("\n");

  log("Processing user message:", textContent);

  // Update session ID if provided
  if (sdkMsg.session_id) {
    sessionId = sdkMsg.session_id;
  }

  // Reset message state for new turn
  currentMessageId = "msg_" + Date.now();
  messageStarted = false;
  contentBlockStarted = false;
  currentBlockIndex = 0;

  // Determine model from env var (passed by SessionManager) or default
  // Check both MODEL (new) and GEMINI_MODEL (backward compat)
  const envModel = process.env.MODEL || process.env.GEMINI_MODEL;
  // Map friendly name (e.g. 'gemini-pro') to CLI flag if needed
  const mappedModel = envModel ? (MODEL_MAPPING[envModel] || envModel) : currentModel;

  const args = [
    "--output-format", "stream-json",
    "--prompt", textContent,
    "-m", mappedModel
  ];
  
  // Append session ID if available (assuming flag support, otherwise ignore)
  // if (sessionId) args.push("--session-id", sessionId);

  log("Spawning gemini with args:", args);

  const gemini = spawn("/usr/bin/gemini", args, {
    env: { ...process.env },
  });

  const geminiRl = readline.createInterface({
    input: gemini.stdout,
    terminal: false,
  });

  geminiRl.on("line", (line) => {
    if (!line.trim()) return;
    try {
      const gEvent = JSON.parse(line) as GeminiEvent;
      mapAndEmitEvent(gEvent);
    } catch (e) {
      log("Failed to parse Gemini output:", line);
    }
  });

  gemini.stderr.on("data", (data) => {
    log("Gemini stderr:", data.toString());
  });

  gemini.on("close", (code) => {
    log("Gemini process exited with code:", code);

    // Close any open content blocks/messages
    if (contentBlockStarted) {
      sendEvent({
        type: "content_block_stop",
        index: currentBlockIndex
      });
      contentBlockStarted = false;
    }

    if (messageStarted) {
      sendEvent({
        type: "message_stop"
      });
      messageStarted = false;
    }

    // Ensure we send a result event to close the turn if Gemini didn't
    sendEvent({
      type: "result", // SDK expects result to finish turn
      status: code === 0 ? "success" : "error",
      usage: { input_tokens: 0, output_tokens: 0 },
      total_cost_usd: 0
    });
  });
}

// Helper to chunk text for streaming
function chunkText(text: string, chunkSize = 50): string[] {
  if (!text) return [];
  const chunks: string[] = [];
  for (let i = 0; i < text.length; i += chunkSize) {
    chunks.push(text.slice(i, i + chunkSize));
  }
  return chunks;
}

// Helper to ensure message_start is sent
function ensureMessageStart() {
  if (!messageStarted) {
    log("Emitting message_start");
    sendEvent({
      type: "message_start",
      message: {
        id: currentMessageId,
        type: "message",
        role: "assistant",
        content: [],
        model: currentModel,
        stop_reason: null,
        stop_sequence: null,
        usage: { input_tokens: 0, output_tokens: 0 }
      }
    });
    messageStarted = true;
  }
}

// Helper to ensure content_block_start is sent
function ensureContentBlockStart() {
  if (!contentBlockStarted) {
    log("Emitting content_block_start for block", currentBlockIndex);
    sendEvent({
      type: "content_block_start",
      index: currentBlockIndex,
      content_block: {
        type: "text",
        text: ""
      }
    });
    contentBlockStarted = true;
  }
}

function mapAndEmitEvent(gEvent: GeminiEvent) {
  log("Mapping Gemini event:", gEvent.type);

  switch (gEvent.type) {
    case "init":
      sendEvent({
        type: "system",
        subtype: "init",
        session_id: gEvent.session_id || sessionId || `gemini-${Date.now()}`,
        model: gEvent.model
      });
      break;

    case "message":
      if (gEvent.role === "assistant") {
        const textContent = typeof gEvent.content === "string" ? gEvent.content : "";

        // Ensure message_start is sent first
        ensureMessageStart();
        ensureContentBlockStart();

        // Chunk the text and send as deltas
        const chunks = chunkText(textContent);
        log(`Emitting ${chunks.length} content_block_delta chunks`);

        for (const chunk of chunks) {
          sendEvent({
            type: "content_block_delta",
            index: currentBlockIndex,
            delta: {
              type: "text_delta",
              text: chunk
            }
          });
        }
      }
      break;

    case "tool_use":
      // Ensure message started
      ensureMessageStart();

      // Close any existing text content block
      if (contentBlockStarted) {
        sendEvent({
          type: "content_block_stop",
          index: currentBlockIndex
        });
        contentBlockStarted = false;
        currentBlockIndex++;
      }

      // Send tool_use as a content block
      sendEvent({
        type: "content_block_start",
        index: currentBlockIndex,
        content_block: {
          type: "tool_use",
          id: gEvent.tool_id || "call_" + Date.now(),
          name: gEvent.tool_name,
          input: gEvent.parameters || {}
        }
      });

      sendEvent({
        type: "content_block_stop",
        index: currentBlockIndex
      });

      currentBlockIndex++;
      break;

    case "tool_result":
      // Tool results should come from TUI side, not adapter
      // But if Gemini sends them, log and ignore
      log("Ignoring tool_result from Gemini (should be handled by TUI)");
      break;

    case "result":
      // Close any open blocks/messages
      if (contentBlockStarted) {
        sendEvent({
          type: "content_block_stop",
          index: currentBlockIndex
        });
        contentBlockStarted = false;
      }

      if (messageStarted) {
        sendEvent({
          type: "message_stop"
        });
        messageStarted = false;
      }
      break;

    case "error":
      log("Gemini error event:", gEvent);
      // Could emit error event here if needed
      break;
  }
}

// Start
main().catch(err => {
  console.error("Adapter Fatal Error:", err);
  process.exit(1);
});
