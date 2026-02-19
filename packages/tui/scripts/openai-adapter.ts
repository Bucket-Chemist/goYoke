#!/usr/bin/env -S npx tsx
/// <reference types="node" />

import OpenAI from "openai";
import * as readline from "readline";

// Protocol Mapping
// TUI Model Name -> OpenAI Model ID
const MODEL_MAPPING: Record<string, string> = {
  "gpt-4-turbo": "gpt-4-turbo-preview",
  "gpt-4": "gpt-4",
  "gpt-3.5-turbo": "gpt-3.5-turbo",
  // Default fallback
  "default": "gpt-4-turbo-preview",
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

// Global state
let currentModel = MODEL_MAPPING["default"];
let sessionId: string | null = null;
let currentMessageId: string | null = null;
let openaiClient: OpenAI | null = null;

// Debug logging
const debug = process.env.DEBUG_ADAPTER === "1";
function log(msg: string, ...args: any[]) {
  if (debug) {
    console.error(`[OPENAI-ADAPTER] ${msg}`, ...args);
  }
}

// Helper to send SDK event
function sendEvent(event: any) {
  console.log(JSON.stringify(event));
}

// Initialize OpenAI client
function initializeClient() {
  const apiKey = process.env.OPENAI_API_KEY;

  if (!apiKey) {
    log("ERROR: OPENAI_API_KEY environment variable not set");
    sendEvent({
      type: "error",
      error: {
        type: "authentication_error",
        message: "OPENAI_API_KEY environment variable is required"
      }
    });
    process.exit(1);
  }

  openaiClient = new OpenAI({ apiKey });
  log("OpenAI client initialized");
}

// Main Loop
async function main() {
  log("Starting OpenAI Adapter...");

  // Initialize OpenAI client
  initializeClient();

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
      sendEvent({
        type: "error",
        error: {
          type: "invalid_request_error",
          message: err instanceof Error ? err.message : String(err)
        }
      });
    }
  });

  // Handle process exit
  process.on("SIGINT", () => {
    log("Received SIGINT, exiting...");
    process.exit(0);
  });
}

async function handleUserMessage(sdkMsg: SDKUserMessage) {
  if (!openaiClient) {
    log("ERROR: OpenAI client not initialized");
    return;
  }

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

  // Determine model from env vars (MODEL takes precedence, then OPENAI_MODEL)
  const envModel = process.env.MODEL || process.env.OPENAI_MODEL;
  // Map friendly name to actual OpenAI model ID
  const mappedModel = envModel ? (MODEL_MAPPING[envModel] || envModel) : currentModel;

  log("Using OpenAI model:", mappedModel);

  try {
    // Send message_start event
    sendEvent({
      type: "message_start",
      message: {
        id: currentMessageId,
        type: "message",
        role: "assistant",
        content: [],
        model: mappedModel,
        stop_reason: null,
        stop_sequence: null,
        usage: { input_tokens: 0, output_tokens: 0 }
      }
    });

    // Send content_block_start event
    sendEvent({
      type: "content_block_start",
      index: 0,
      content_block: {
        type: "text",
        text: ""
      }
    });

    // Create streaming chat completion
    const stream = await openaiClient.chat.completions.create({
      model: mappedModel,
      messages: [{ role: "user", content: textContent }],
      stream: true,
    });

    let fullText = "";
    let inputTokens = 0;
    let outputTokens = 0;

    // Process stream chunks
    for await (const chunk of stream) {
      const delta = chunk.choices[0]?.delta?.content || "";

      if (delta) {
        fullText += delta;

        // Send content_block_delta event
        sendEvent({
          type: "content_block_delta",
          index: 0,
          delta: {
            type: "text_delta",
            text: delta
          }
        });
      }

      // Track usage if available (only in final chunk for some models)
      if (chunk.usage) {
        inputTokens = chunk.usage.prompt_tokens || 0;
        outputTokens = chunk.usage.completion_tokens || 0;
      }
    }

    log("Stream completed, total text length:", fullText.length);

    // Send content_block_stop event
    sendEvent({
      type: "content_block_stop",
      index: 0
    });

    // Send message_delta with stop_reason
    sendEvent({
      type: "message_delta",
      delta: {
        stop_reason: "end_turn",
        stop_sequence: null
      },
      usage: {
        output_tokens: outputTokens
      }
    });

    // Send message_stop event
    sendEvent({
      type: "message_stop"
    });

  } catch (error: any) {
    log("OpenAI API error:", error);

    // Determine error type
    let errorType = "api_error";
    let errorMessage = error.message || String(error);

    if (error.status === 401) {
      errorType = "authentication_error";
      errorMessage = "Invalid API key";
    } else if (error.status === 429) {
      errorType = "rate_limit_error";
      errorMessage = "Rate limit exceeded";
    } else if (error.status === 400) {
      errorType = "invalid_request_error";
    }

    sendEvent({
      type: "error",
      error: {
        type: errorType,
        message: errorMessage
      }
    });
  }
}

// Start
main().catch(err => {
  console.error("Adapter Fatal Error:", err);
  process.exit(1);
});
