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

  // Determine model from env var (passed by SessionManager) or default
  const envModel = process.env.GEMINI_MODEL;
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
    // Ensure we send a result event to close the turn if Gemini didn't
    sendEvent({
      type: "result", // SDK expects result to finish turn
      status: code === 0 ? "success" : "error",
      usage: { input_tokens: 0, output_tokens: 0 },
      total_cost_usd: 0
    });
  });
}

function mapAndEmitEvent(gEvent: GeminiEvent) {
  // log("Mapping Gemini event:", gEvent.type);

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
        sendEvent({
          type: "assistant", // SDK event type
          message: {
            id: currentMessageId, // Persist ID for deltas
            role: "assistant",
            content: [{ type: "text", text: gEvent.content || "" }]
          }
        });
      }
      break;
    
    case "tool_use":
       sendEvent({
         type: "assistant",
         message: {
           id: currentMessageId,
           role: "assistant",
           content: [{
             type: "tool_use",
             id: gEvent.tool_id || "call_" + Date.now(),
             name: gEvent.tool_name,
             input: gEvent.parameters
           }]
         }
       });
       break;

    case "tool_result":
        sendEvent({
          type: "user",
          message: {
             role: "user",
             content: [{
               type: "tool_result",
               tool_use_id: gEvent.tool_id,
               content: gEvent.output
             }]
          }
        });
        break;

    case "result":
      // Handled by close event fallback usually, but if received, good.
      break;
      
    case "error":
      // Optional: send error event
      break;
  }
}

// Start
main().catch(err => {
  console.error("Adapter Fatal Error:", err);
  process.exit(1);
});
