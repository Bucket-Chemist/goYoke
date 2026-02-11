#!/usr/bin/env node
/**
 * Test: Stream Isolation
 *
 * Verifies if output streams from concurrent queries are properly isolated.
 * Tests if query1's output stream contains only query1's responses and
 * query2's stream contains only query2's responses (no cross-contamination).
 *
 * Expected behavior: Complete stream isolation, no interleaving.
 * Failure modes: Output mixed between streams, ALPHA in query2 or BETA in query1.
 */

import { query, type SDKMessage } from "@anthropic-ai/claude-agent-sdk";

interface TestResult {
  test: "stream_isolation";
  query1_status: "completed" | "error" | "timeout";
  query2_status: "completed" | "error" | "timeout";
  query1_contaminated: boolean;
  query2_contaminated: boolean;
  query1_text: string;
  query2_text: string;
  query1_error?: string;
  query2_error?: string;
  behavior: "isolated" | "contaminated" | "failed";
  contamination_details?: string;
  duration_ms: number;
}

/**
 * Collect all text output from a query
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
      prompt: "Repeat this exact string 5 times on separate lines: ALPHA",
      options: {
        abortController: controller,
        allowedTools: [],
        model: "haiku",
      },
    });

    const query2 = query({
      prompt: "Repeat this exact string 5 times on separate lines: BETA",
      options: {
        abortController: controller,
        allowedTools: [],
        model: "haiku",
      },
    });

    // Collect outputs concurrently
    const [result1, result2] = await Promise.all([
      collectQueryOutput(query1, controller.signal),
      collectQueryOutput(query2, controller.signal),
    ]);

    clearTimeout(timeout);

    // Check for contamination
    const query1Text = result1.output;
    const query2Text = result2.output;

    // Query1 should only contain ALPHA, not BETA
    const query1ContainsBeta = query1Text.includes("BETA");

    // Query2 should only contain BETA, not ALPHA
    const query2ContainsAlpha = query2Text.includes("ALPHA");

    const query1Contaminated = query1ContainsBeta;
    const query2Contaminated = query2ContainsAlpha;

    // Determine behavior
    let behavior: TestResult["behavior"];
    const bothCompleted = result1.status === "completed" && result2.status === "completed";

    if (!bothCompleted) {
      behavior = "failed";
    } else if (query1Contaminated || query2Contaminated) {
      behavior = "contaminated";
    } else {
      behavior = "isolated";
    }

    const duration = performance.now() - startTime;

    // Build contamination details
    let contaminationDetails: string | undefined;
    if (behavior === "contaminated") {
      const details: string[] = [];
      if (query1Contaminated) {
        details.push("Query1 contains BETA");
      }
      if (query2Contaminated) {
        details.push("Query2 contains ALPHA");
      }
      contaminationDetails = details.join("; ");
    }

    const testResult: TestResult = {
      test: "stream_isolation",
      query1_status: result1.status,
      query2_status: result2.status,
      query1_contaminated: query1Contaminated,
      query2_contaminated: query2Contaminated,
      query1_text: query1Text.trim(),
      query2_text: query2Text.trim(),
      query1_error: result1.error,
      query2_error: result2.error,
      behavior,
      contamination_details: contaminationDetails,
      duration_ms: duration,
    };

    console.log(JSON.stringify(testResult, null, 2));

    // Exit with appropriate code
    if (behavior === "isolated") {
      process.exit(0);
    } else {
      process.exit(1);
    }
  } catch (error) {
    clearTimeout(timeout);

    const duration = performance.now() - startTime;
    const errorMessage = error instanceof Error ? error.message : String(error);

    const testResult: TestResult = {
      test: "stream_isolation",
      query1_status: "error",
      query2_status: "error",
      query1_contaminated: false,
      query2_contaminated: false,
      query1_text: "",
      query2_text: "",
      query1_error: errorMessage,
      query2_error: errorMessage,
      behavior: "failed",
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
