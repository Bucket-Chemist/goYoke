#!/usr/bin/env node
/**
 * Test: Concurrent Query Execution
 *
 * Verifies if two concurrent query() calls from the same process can complete
 * successfully without interference. Tests if SDK properly isolates concurrent
 * sessions or if they interfere with each other.
 *
 * Expected behavior: Both queries complete successfully with clean output.
 * Failure modes: One/both fail, output interleaving, subprocess conflicts.
 */

import { query, type SDKMessage } from "@anthropic-ai/claude-agent-sdk";

interface TestResult {
  test: "concurrent_queries";
  query1_status: "completed" | "error" | "timeout";
  query2_status: "completed" | "error" | "timeout";
  query1_error?: string;
  query2_error?: string;
  query1_output: string;
  query2_output: string;
  behavior:
    | "both_completed"
    | "one_failed"
    | "both_failed"
    | "timeout"
    | "interleaved";
  duration_ms: number;
}

/**
 * Collect all text output from a query's async generator
 */
async function collectQueryOutput(
  queryInstance: AsyncIterable<SDKMessage>,
  signal: AbortSignal
): Promise<{ status: "completed" | "error" | "timeout"; output: string; error?: string }> {
  const outputParts: string[] = [];
  let status: "completed" | "error" | "timeout" = "completed";
  let errorMessage: string | undefined;

  try {
    for await (const event of queryInstance) {
      // Check abort signal
      if (signal.aborted) {
        status = "timeout";
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
    status = "error";
    errorMessage = error instanceof Error ? error.message : String(error);
  }

  return {
    status,
    output: outputParts.join(""),
    error: errorMessage,
  };
}

/**
 * Main test execution
 */
async function main(): Promise<void> {
  const startTime = performance.now();

  // Create abort controller with 60s timeout
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 60000);

  try {
    // Start both queries concurrently
    const query1 = query({
      prompt: "Respond with exactly: QUERY_ONE_COMPLETE",
      options: {
        abortController: controller,
        allowedTools: [],
        model: "haiku",
      },
    });

    const query2 = query({
      prompt: "Respond with exactly: QUERY_TWO_COMPLETE",
      options: {
        abortController: controller,
        allowedTools: [],
        model: "haiku",
      },
    });

    // Collect both concurrently
    const [result1, result2] = await Promise.all([
      collectQueryOutput(query1, controller.signal),
      collectQueryOutput(query2, controller.signal),
    ]);

    clearTimeout(timeout);

    // Determine behavior
    let behavior: TestResult["behavior"];
    const output1 = result1.output.trim();
    const output2 = result2.output.trim();

    if (result1.status === "completed" && result2.status === "completed") {
      // Check for interleaving
      const query1HasQuery2Text = output1.includes("QUERY_TWO_COMPLETE");
      const query2HasQuery1Text = output2.includes("QUERY_ONE_COMPLETE");

      if (query1HasQuery2Text || query2HasQuery1Text) {
        behavior = "interleaved";
      } else {
        behavior = "both_completed";
      }
    } else if (result1.status === "error" || result2.status === "error") {
      if (result1.status === "error" && result2.status === "error") {
        behavior = "both_failed";
      } else {
        behavior = "one_failed";
      }
    } else if (result1.status === "timeout" || result2.status === "timeout") {
      behavior = "timeout";
    } else {
      behavior = "both_failed";
    }

    const duration = performance.now() - startTime;

    const testResult: TestResult = {
      test: "concurrent_queries",
      query1_status: result1.status,
      query2_status: result2.status,
      query1_error: result1.error,
      query2_error: result2.error,
      query1_output: output1,
      query2_output: output2,
      behavior,
      duration_ms: duration,
    };

    // Print result as JSON
    console.log(JSON.stringify(testResult, null, 2));

    // Exit with appropriate code
    if (behavior === "both_completed" || behavior === "interleaved") {
      process.exit(0);
    } else {
      process.exit(1);
    }
  } catch (error) {
    clearTimeout(timeout);

    const duration = performance.now() - startTime;
    const errorMessage = error instanceof Error ? error.message : String(error);

    const testResult: TestResult = {
      test: "concurrent_queries",
      query1_status: "error",
      query2_status: "error",
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
