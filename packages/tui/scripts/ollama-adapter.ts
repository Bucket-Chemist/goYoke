#!/usr/bin/env -S npx tsx
/// <reference types="node" />

import * as readline from "readline";

// Types for SDK Messages
interface SDKUserMessage {
  type: "user";
  message: {
    role: "user";
    content: Array<{ type: "text"; text: string }>;
  };
  session_id?: string;
}

// Types for Ollama API
interface OllamaMessage {
  role: "user" | "assistant";
  content: string;
}

interface OllamaChatRequest {
  model: string;
  messages: OllamaMessage[];
  stream: boolean;
}

interface OllamaChatResponse {
  model: string;
  created_at: string;
  message: {
    role: string;
    content: string;
  };
  done: boolean;
  total_duration?: number;
  load_duration?: number;
  prompt_eval_count?: number;
  eval_count?: number;
}

// Global state
let sessionId: string | null = null;
let currentMessageId: string | null = null;
let conversationHistory: OllamaMessage[] = [];

// Configuration
const OLLAMA_ENDPOINT = process.env.OLLAMA_ENDPOINT || "http://localhost:11434";
const DEFAULT_MODEL = "llama3.1:8b";

// Debug logging
const debug = process.env.DEBUG_ADAPTER === "1";
function log(msg: string, ...args: any[]) {
  if (debug) {
    console.error(`[OLLAMA ADAPTER] ${msg}`, ...args);
  }
}

// Helper to send SDK event
function sendEvent(event: any) {
  console.log(JSON.stringify(event));
}

// Check if Ollama server is running
async function checkOllamaServer(): Promise<boolean> {
  try {
    const response = await fetch(`${OLLAMA_ENDPOINT}/api/tags`, {
      method: "GET",
    });
    return response.ok;
  } catch (err) {
    return false;
  }
}

// Main Loop
async function main() {
  log("Starting Ollama Adapter...");
  log(`Ollama endpoint: ${OLLAMA_ENDPOINT}`);

  // Check if Ollama is running
  const isRunning = await checkOllamaServer();
  if (!isRunning) {
    console.error(`Error: Cannot connect to Ollama at ${OLLAMA_ENDPOINT}`);
    console.error("Please ensure Ollama is running with: ollama serve");
    process.exit(1);
  }

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
          type: "api_error",
          message: String(err),
        },
      });
    }
  });

  // Handle process exit
  process.on("SIGINT", () => {
    log("Received SIGINT, exiting...");
    process.exit(0);
  });

  // Send init event
  sendEvent({
    type: "system",
    subtype: "init",
    session_id: `ollama-${Date.now()}`,
    model: process.env.MODEL || process.env.OLLAMA_MODEL || DEFAULT_MODEL,
  });
}

async function handleUserMessage(sdkMsg: SDKUserMessage) {
  // Extract text content
  const textContent = sdkMsg.message.content
    .filter((c: any) => c.type === "text")
    .map((c: any) => c.text)
    .join("\n");

  log("Processing user message:", textContent.substring(0, 100) + "...");

  // Update session ID if provided
  if (sdkMsg.session_id) {
    sessionId = sdkMsg.session_id;
  }

  // Reset message state for new turn
  currentMessageId = `msg_${Date.now()}`;

  // Add user message to conversation history
  conversationHistory.push({
    role: "user",
    content: textContent,
  });

  // Determine model from env var or default
  const model = process.env.MODEL || process.env.OLLAMA_MODEL || DEFAULT_MODEL;
  log(`Using model: ${model}`);

  const requestBody: OllamaChatRequest = {
    model,
    messages: conversationHistory,
    stream: true,
  };

  try {
    // Send message_start event
    sendEvent({
      type: "message_start",
      message: {
        id: currentMessageId,
        type: "message",
        role: "assistant",
        content: [],
        model,
        stop_reason: null,
        usage: { input_tokens: 0, output_tokens: 0 },
      },
    });

    // Send content_block_start event
    const contentBlockId = `block_${Date.now()}`;
    sendEvent({
      type: "content_block_start",
      index: 0,
      content_block: {
        type: "text",
        text: "",
      },
    });

    // Make streaming request to Ollama
    const response = await fetch(`${OLLAMA_ENDPOINT}/api/chat`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(requestBody),
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`Ollama API error: ${response.status} ${errorText}`);
    }

    const reader = response.body?.getReader();
    if (!reader) {
      throw new Error("No response body from Ollama");
    }

    const decoder = new TextDecoder();
    let assistantContent = "";
    let totalDuration = 0;
    let promptEvalCount = 0;
    let evalCount = 0;

    // Process streaming response
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      const chunk = decoder.decode(value, { stream: true });
      const lines = chunk.split("\n").filter((line) => line.trim());

      for (const line of lines) {
        try {
          const data = JSON.parse(line) as OllamaChatResponse;

          if (data.message?.content) {
            const deltaText = data.message.content;
            assistantContent += deltaText;

            // Send content_block_delta event
            sendEvent({
              type: "content_block_delta",
              index: 0,
              delta: {
                type: "text_delta",
                text: deltaText,
              },
            });
          }

          if (data.done) {
            totalDuration = data.total_duration || 0;
            promptEvalCount = data.prompt_eval_count || 0;
            evalCount = data.eval_count || 0;
          }
        } catch (parseError) {
          log("Failed to parse chunk:", line);
        }
      }
    }

    // Add assistant response to conversation history
    conversationHistory.push({
      role: "assistant",
      content: assistantContent,
    });

    // Send content_block_stop event
    sendEvent({
      type: "content_block_stop",
      index: 0,
    });

    // Send message_stop event
    sendEvent({
      type: "message_stop",
    });

    // Send result event with usage stats
    const costPerToken = 0; // Ollama is free/local
    sendEvent({
      type: "result",
      status: "success",
      usage: {
        input_tokens: promptEvalCount,
        output_tokens: evalCount,
      },
      total_cost_usd: costPerToken,
    });

    log(`Completed. Input tokens: ${promptEvalCount}, Output tokens: ${evalCount}`);
  } catch (err: any) {
    log("Error calling Ollama API:", err);

    let errorMessage = String(err);
    let errorType = "api_error";

    // Handle specific error cases
    if (err.message?.includes("ECONNREFUSED")) {
      errorMessage = `Cannot connect to Ollama at ${OLLAMA_ENDPOINT}. Is Ollama running?`;
      errorType = "connection_error";
    } else if (err.message?.includes("model") && err.message?.includes("not found")) {
      errorMessage = `Model not found. Please pull the model first with: ollama pull ${process.env.MODEL || process.env.OLLAMA_MODEL || DEFAULT_MODEL}`;
      errorType = "invalid_request_error";
    }

    // Send error event
    sendEvent({
      type: "error",
      error: {
        type: errorType,
        message: errorMessage,
      },
    });

    // Send result event with error status
    sendEvent({
      type: "result",
      status: "error",
      usage: { input_tokens: 0, output_tokens: 0 },
      total_cost_usd: 0,
    });
  }
}

// Start
main().catch((err) => {
  console.error("Adapter Fatal Error:", err);
  process.exit(1);
});
