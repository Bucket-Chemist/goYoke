/**
 * Shared team formatting utilities
 * Extracted from TeamList.tsx and TeamDetail.tsx to eliminate duplication
 */

import { colors, icons } from "../config/theme.js";

/**
 * Format milliseconds to human-readable duration
 */
export function formatDuration(ms: number): string {
  if (ms < 1000) {
    return `${ms}ms`;
  }
  if (ms < 60000) {
    return `${(ms / 1000).toFixed(1)}s`;
  }
  const minutes = Math.floor(ms / 60000);
  const seconds = ((ms % 60000) / 1000).toFixed(0);
  return `${minutes}m ${seconds}s`;
}

/**
 * Format elapsed time from ISO timestamp to now
 */
export function formatElapsed(startedAt: string | null): string {
  if (!startedAt) return "—";

  const start = new Date(startedAt).getTime();
  const now = Date.now();
  const elapsed = now - start;

  return formatDuration(elapsed);
}

/**
 * Get team status display icon and color
 */
export function getTeamStatusDisplay(
  status: string,
  alive: boolean
): { icon: string; color: string } {
  if (status === "completed") {
    return { icon: icons.agentComplete, color: colors.agentComplete };
  }
  if (status === "failed") {
    return { icon: icons.agentError, color: colors.agentError };
  }
  if (alive || status === "running") {
    return { icon: icons.agentRunning, color: colors.agentRunning };
  }
  return { icon: icons.agentSpawning, color: colors.agentSpawning };
}

/**
 * Get member status icon
 */
export function getMemberStatusIcon(status: string): string {
  switch (status) {
    case "completed":
      return icons.agentComplete;
    case "failed":
      return icons.agentError;
    case "running":
      return icons.agentRunning;
    default:
      return icons.agentSpawning;
  }
}

/**
 * Get member status color
 */
export function getMemberStatusColor(status: string): string {
  switch (status) {
    case "completed":
      return colors.agentComplete;
    case "failed":
      return colors.agentError;
    case "running":
      return colors.agentRunning;
    default:
      return colors.muted;
  }
}

/**
 * Get team status color for TeamDetail
 */
export function getTeamStatusColor(status: string): string {
  switch (status) {
    case "completed":
      return colors.success;
    case "failed":
      return colors.error;
    case "running":
      return colors.warning;
    default:
      return colors.muted;
  }
}
