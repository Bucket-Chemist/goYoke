#!/usr/bin/env node
/**
 * Test: AsyncIterable Process Reuse
 *
 * Empirically validates whether passing an AsyncIterable<SDKUserMessage> as the
 * `prompt` parameter to query() keeps the CLI process alive between yields
 * (i.e., across conversation turns).
 *
 * Expected behavior: Process reused, msg2 latency << msg1 latency, session IDs match.
 * Failure modes: Process dies between yields, forcing fresh spawn with high latency.
 *
 * Usage:
 *   npx tsx scripts/test-async-iterable.ts [--skip-long-idle]
 */

import { query, type SDKMessage, type SDKUserMessage } from "@anthropic-ai/claude-agent-sdk";
import type { MessageParam } from "@anthropic-ai/sdk/resources/messages";

interface IdleTestResult {
  idle_seconds: number;
  msg1_latency_ms: number;
  msg2_latency_ms: number;
  session_ids_match: boolean;
  session_id_msg1: string;
  session_id_msg2: string;
  msg1_response: string;
  msg2_response: string;
  process_reused_inference: boolean;
  status: "pass" | "fail" | "error";
  error?: string;
}

interface TestResult {
  test: "async_iterable_process_reuse";
  tests: {
    short_idle: IdleTestResult | null;
    long_idle: IdleTestResult | null;
  };
  overall_verdict: "PROCESS_REUSED" | "PROCESS_NOT_REUSED" | "INCONCLUSIVE" | "ERROR";
  duration_ms: number;
}

/**
 * Coordination mechanism for generator and event consumer
 */
interface MessageCoordinator {
  text: string;
  yieldedAt: number; // Timestamp when generator yielded this message
  responsePromise: Promise<{
    latency_ms: number;
    session_id: string;
    response: string;
  }>;
  resolveResponse: (value: {
    latency_ms: number;
    session_id: string;
    response: string;
  }) => void;
  rejectResponse: (error: Error) => void;
}

/**
 * Create async generator that yields user messages with controlled timing
 */
async function* createMessageGenerator(
  coordinators: MessageCoordinator[]
): AsyncGenerator<SDKUserMessage, void, undefined> {
  for (const coordinator of coordinators) {
    const userMessage: SDKUserMessage = {
      type: 'user' as const,
      message: {
        role: 'user' as const,
        content: [
          {
            type: 'text' as const,
            text: coordinator.text,
          },
        ],
      } as MessageParam,
      parent_tool_use_id: null,
      session_id: '',
    };

    // Record timestamp just before yielding (for accurate latency measurement)
    coordinator.yieldedAt = performance.now();
    yield userMessage;

    // Wait for response to complete before yielding next message
    await coordinator.responsePromise;
  }
}

/**
 * Consume events from query, extract latency and session ID for each message
 */
async function consumeEvents(
  queryInstance: AsyncIterable<SDKMessage>,
  coordinators: MessageCoordinator[],
  signal: AbortSignal
): Promise<void> {
  let currentMessageIndex = 0;
  let firstTokenReceived = false;
  let firstTokenLatency = 0;
  let sessionId = "";
  const responseParts: string[] = [];

  try {
    for await (const event of queryInstance) {
      if (signal.aborted) {
        throw new Error("Aborted by timeout");
      }

      // Track session ID from system events
      if (event.type === "system" && "session_id" in event && typeof event.session_id === "string") {
        sessionId = event.session_id;
      }

      // Track first token for latency measurement (from yield time, NOT from previous result)
      if (event.type === "assistant" && "message" in event && !firstTokenReceived) {
        const coordinator = coordinators[currentMessageIndex];
        firstTokenLatency = coordinator ? performance.now() - coordinator.yieldedAt : 0;
        firstTokenReceived = true;
      }

      // Collect response text
      if (event.type === "assistant" && "message" in event) {
        const content = event.message.content;
        if (Array.isArray(content)) {
          for (const block of content) {
            if ("text" in block && typeof block.text === "string") {
              responseParts.push(block.text);
            }
          }
        }
      }

      // On result, resolve the current message coordinator
      if (event.type === "result") {
        const responseText = responseParts.join("").trim();

        if (currentMessageIndex < coordinators.length) {
          coordinators[currentMessageIndex].resolveResponse({
            latency_ms: firstTokenLatency, // Use first-token latency from yield time
            session_id: sessionId,
            response: responseText,
          });

          // Prepare for next message
          currentMessageIndex++;
          firstTokenReceived = false;
          firstTokenLatency = 0;
          responseParts.length = 0;
        }
      }

      // On error result, reject current coordinator
      if (event.type === "result" && "is_error" in event && event.is_error) {
        const errorMessages = "errors" in event && Array.isArray(event.errors)
          ? event.errors.join("; ")
          : "Unknown error";

        if (currentMessageIndex < coordinators.length) {
          coordinators[currentMessageIndex].rejectResponse(
            new Error(errorMessages)
          );
        }
        throw new Error(errorMessages);
      }
    }
  } catch (error) {
    // Reject any remaining coordinators
    for (let i = currentMessageIndex; i < coordinators.length; i++) {
      coordinators[i].rejectResponse(
        error instanceof Error ? error : new Error(String(error))
      );
    }
    throw error;
  }
}

/**
 * Run a single idle test
 */
async function runIdleTest(
  idleSeconds: number,
  signal: AbortSignal
): Promise<IdleTestResult> {
  try {
    // Create coordinators for two messages
    const coordinators: MessageCoordinator[] = [
      {
        text: "Reply with only the word ALPHA. Nothing else.",
        yieldedAt: 0,
        responsePromise: null as any,
        resolveResponse: null as any,
        rejectResponse: null as any,
      },
      {
        text: "Reply with only the word BETA. Nothing else.",
        yieldedAt: 0,
        responsePromise: null as any,
        resolveResponse: null as any,
        rejectResponse: null as any,
      },
    ];

    // Initialize promises
    for (const coordinator of coordinators) {
      coordinator.responsePromise = new Promise((resolve, reject) => {
        coordinator.resolveResponse = resolve;
        coordinator.rejectResponse = reject;
      });
    }

    // Insert idle delay before second message
    const originalResponsePromise = coordinators[0].responsePromise;
    coordinators[0].responsePromise = originalResponsePromise.then(async (result) => {
      // Wait for idle duration
      await new Promise((resolve) => setTimeout(resolve, idleSeconds * 1000));
      return result;
    });

    // Create generator
    const messageGenerator = createMessageGenerator(coordinators);

    // Start query with AsyncIterable prompt
    const queryInstance = query({
      prompt: messageGenerator,
      options: {
        abortController: { signal } as AbortController,
        allowedTools: [],
        model: "haiku",
        permissionMode: "bypassPermissions",
        allowDangerouslySkipPermissions: true,
      },
    });

    // Consume events (this will resolve coordinators as responses arrive)
    await consumeEvents(queryInstance, coordinators, signal);

    // Extract results
    const result1 = await coordinators[0].responsePromise;
    const result2 = await coordinators[1].responsePromise;

    // Calculate process reuse inference
    // Primary signal: latency delta > 1500ms (expected process startup overhead is 1.5-3.5s)
    // Secondary signal: session IDs match (same process, same session)
    // The old heuristic (msg2 < msg1 * 0.5) was too strict — API round-trip dominates both measurements
    const latencyDelta = result1.latency_ms - result2.latency_ms;
    const sessionMatch = result1.session_id === result2.session_id && result1.session_id !== "";
    const processReusedInference = sessionMatch && latencyDelta > 1500;

    return {
      idle_seconds: idleSeconds,
      msg1_latency_ms: Math.round(result1.latency_ms),
      msg2_latency_ms: Math.round(result2.latency_ms),
      session_ids_match: result1.session_id === result2.session_id,
      session_id_msg1: result1.session_id,
      session_id_msg2: result2.session_id,
      msg1_response: result1.response,
      msg2_response: result2.response,
      process_reused_inference: processReusedInference,
      status: "pass",
    };
  } catch (error) {
    return {
      idle_seconds: idleSeconds,
      msg1_latency_ms: 0,
      msg2_latency_ms: 0,
      session_ids_match: false,
      session_id_msg1: "",
      session_id_msg2: "",
      msg1_response: "",
      msg2_response: "",
      process_reused_inference: false,
      status: "error",
      error: error instanceof Error ? error.message : String(error),
    };
  }
}

/**
 * Main test execution
 */
async function main(): Promise<void> {
  const startTime = performance.now();
  const skipLongIdle = process.argv.includes("--skip-long-idle");

  // Create abort controller with overall timeout
  const overallTimeout = skipLongIdle ? 180000 : 300000; // 3min or 5min
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), overallTimeout);

  try {
    // Test 1: Short idle (5s)
    const shortIdleResult = await runIdleTest(5, controller.signal);

    // Test 2: Long idle (60s) - optional
    let longIdleResult: IdleTestResult | null = null;
    if (!skipLongIdle) {
      longIdleResult = await runIdleTest(60, controller.signal);
    }

    clearTimeout(timeout);

    // Determine overall verdict
    let overallVerdict: TestResult["overall_verdict"];

    if (shortIdleResult.status === "error" && (!longIdleResult || longIdleResult.status === "error")) {
      overallVerdict = "ERROR";
    } else if (shortIdleResult.status === "pass" && shortIdleResult.process_reused_inference) {
      if (!longIdleResult || longIdleResult.status === "error") {
        overallVerdict = "PROCESS_REUSED";
      } else if (longIdleResult.process_reused_inference) {
        overallVerdict = "PROCESS_REUSED";
      } else {
        overallVerdict = "INCONCLUSIVE";
      }
    } else if (shortIdleResult.status === "pass" && !shortIdleResult.process_reused_inference) {
      overallVerdict = "PROCESS_NOT_REUSED";
    } else {
      overallVerdict = "INCONCLUSIVE";
    }

    const duration = performance.now() - startTime;

    const testResult: TestResult = {
      test: "async_iterable_process_reuse",
      tests: {
        short_idle: shortIdleResult,
        long_idle: longIdleResult,
      },
      overall_verdict: overallVerdict,
      duration_ms: Math.round(duration),
    };

    console.log(JSON.stringify(testResult, null, 2));

    // Exit with appropriate code
    if (overallVerdict === "PROCESS_REUSED") {
      process.exit(0);
    } else {
      process.exit(1);
    }
  } catch (error) {
    clearTimeout(timeout);

    const duration = performance.now() - startTime;
    const errorMessage = error instanceof Error ? error.message : String(error);

    const testResult: TestResult = {
      test: "async_iterable_process_reuse",
      tests: {
        short_idle: {
          idle_seconds: 5,
          msg1_latency_ms: 0,
          msg2_latency_ms: 0,
          session_ids_match: false,
          session_id_msg1: "",
          session_id_msg2: "",
          msg1_response: "",
          msg2_response: "",
          process_reused_inference: false,
          status: "error",
          error: errorMessage,
        },
        long_idle: null,
      },
      overall_verdict: "ERROR",
      duration_ms: Math.round(duration),
    };

    console.log(JSON.stringify(testResult, null, 2));
    process.exit(1);
  }
}

main().catch((error) => {
  console.error("Unexpected crash:", error);
  process.exit(1);
});
