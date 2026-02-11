#!/usr/bin/env node
/**
 * Test: Query Cancellation on Concurrent Start
 *
 * Verifies if starting a second query() call cancels or interferes with the
 * first query that's still running. Tests SDK behavior when multiple queries
 * are active sequentially with overlap.
 *
 * Expected behavior: Both queries complete independently.
 * Failure modes: Query1 cancelled, Query1 errors, Query2 blocks until Query1 finishes.
 */

import { query, type SDKMessage } from "@anthropic-ai/claude-agent-sdk";

interface TestResult {
  test: "query_cancellation";
  query1_fate: "completed" | "cancelled" | "error" | "timeout";
  query2_fate: "completed" | "cancelled" | "error" | "timeout";
  query1_error?: string;
  query2_error?: string;
  query1_output: string;
  query2_output: string;
  query1_partial_output?: string;
  behavior:
    | "both_completed"
    | "query1_cancelled_query2_ok"
    | "both_failed"
    | "query2_blocked";
  duration_ms: number;
}

/**
 * Collect output from a query with status tracking
 */
async function collectQueryOutput(
  queryInstance: AsyncIterable<SDKMessage>,
  signal: AbortSignal,
  queryName: string
): Promise<{
  fate: "completed" | "cancelled" | "error" | "timeout";
  output: string;
  error?: string;
}> {
  const outputParts: string[] = [];
  let fate: "completed" | "cancelled" | "error" | "timeout" = "completed";
  let errorMessage: string | undefined;

  try {
    for await (const event of queryInstance) {
      if (signal.aborted) {
        fate = "timeout";
        break;
      }

      // Collect text from assistant messages (SDKAssistantMessage has .message.content)
      if (event.type === "assistant" && "message" in event) {
        const content = event.message.content;
        if (Array.isArray(content)) {
          for (const block of content) {
            if ("text" in block && typeof block.text === "string") {
              outputParts.push(block.text);
            }
          }
        }
      }
      // Also collect from result messages (SDKResultSuccess has .result)
      if (event.type === "result" && "result" in event && typeof event.result === "string") {
        outputParts.push(event.result);
      }
    }
  } catch (error) {
    const errorMsg = error instanceof Error ? error.message : String(error);

    // Distinguish cancellation from other errors
    if (errorMsg.includes("abort") || errorMsg.includes("cancel")) {
      fate = "cancelled";
    } else {
      fate = "error";
    }

    errorMessage = errorMsg;
  }

  return {
    fate,
    output: outputParts.join(""),
    error: errorMessage,
  };
}

/**
 * Wait for specified milliseconds
 */
function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * Main test execution
 */
async function main(): Promise<void> {
  const startTime = performance.now();

  // Create abort controller with 90s timeout
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 90000);

  try {
    // Start first query (should take time)
    const query1 = query({
      prompt: "Count from 1 to 20, one number per line",
      options: {
        abortController: controller,
        allowedTools: [],
        model: "haiku",
      },
    });

    // Start collecting query1 output
    const query1Promise = collectQueryOutput(query1, controller.signal, "query1");

    // Wait 2 seconds
    await sleep(2000);

    // Start second query while first is still running
    const query2 = query({
      prompt: "Respond with exactly: QUERY_TWO_DONE",
      options: {
        abortController: controller,
        allowedTools: [],
        model: "haiku",
      },
    });

    // Collect query2 output
    const query2Promise = collectQueryOutput(query2, controller.signal, "query2");

    // Wait for both to complete
    const [result1, result2] = await Promise.all([query1Promise, query2Promise]);

    clearTimeout(timeout);

    // Determine behavior
    let behavior: TestResult["behavior"];

    if (result1.fate === "completed" && result2.fate === "completed") {
      behavior = "both_completed";
    } else if (result1.fate === "cancelled" && result2.fate === "completed") {
      behavior = "query1_cancelled_query2_ok";
    } else if (
      result1.fate === "error" ||
      result2.fate === "error" ||
      (result1.fate === "cancelled" && result2.fate === "cancelled")
    ) {
      behavior = "both_failed";
    } else if (result2.fate === "timeout" && result1.fate === "completed") {
      behavior = "query2_blocked";
    } else {
      behavior = "both_failed";
    }

    const duration = performance.now() - startTime;

    const testResult: TestResult = {
      test: "query_cancellation",
      query1_fate: result1.fate,
      query2_fate: result2.fate,
      query1_error: result1.error,
      query2_error: result2.error,
      query1_output: result1.output.trim(),
      query2_output: result2.output.trim(),
      behavior,
      duration_ms: duration,
    };

    // If query1 was cancelled, include partial output
    if (result1.fate === "cancelled" && result1.output) {
      testResult.query1_partial_output = result1.output.trim();
    }

    console.log(JSON.stringify(testResult, null, 2));

    // Exit with appropriate code
    if (behavior === "both_completed" || behavior === "query1_cancelled_query2_ok") {
      process.exit(0);
    } else {
      process.exit(1);
    }
  } catch (error) {
    clearTimeout(timeout);

    const duration = performance.now() - startTime;
    const errorMessage = error instanceof Error ? error.message : String(error);

    const testResult: TestResult = {
      test: "query_cancellation",
      query1_fate: "error",
      query2_fate: "error",
      query1_error: errorMessage,
      query2_error: errorMessage,
      query1_output: "",
      query2_output: "",
      behavior: "both_failed",
      duration_ms: duration,
    };

    console.log(JSON.stringify(testResult, null, 2));
    process.exit(1);
  }
}

main().catch((error) => {
  console.error("Unexpected crash:", error);
  process.exit(1);
});
