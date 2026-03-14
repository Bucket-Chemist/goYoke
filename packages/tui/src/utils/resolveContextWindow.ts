/**
 * Derive context window capacity from a model ID string.
 *
 * The SDK reports the real capacity in modelUsage.contextWindow — use that as
 * the source of truth whenever available. This function is a best-effort
 * fallback for the window between session connect and first SDK response.
 */
export function resolveContextWindow(modelId: string): number {
  if (modelId.includes("[1m]")) return 1_000_000;
  if (modelId.includes("haiku")) return 200_000;
  return 200_000;
}
