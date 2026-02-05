/**
 * Cost Tracking for MCP Agent Spawning
 *
 * Tracks costs from spawned agents and aggregates to parent session.
 * Per MCP-SPAWN-012 R10: Cost Attribution Strategy
 */

export interface SpawnCost {
  agentId: string;
  agentType: string;
  cost: number;
  tokens: {
    input: number;
    output: number;
  };
  turns: number;
  timestamp: number;
}

export interface SessionCostSummary {
  sessionId: string;
  directCosts: number;
  spawnCosts: SpawnCost[];
  totalCost: number;
  startTime: number;
  endTime?: number;
}

/**
 * Session-level cost tracker (singleton per session)
 */
class SessionCostTracker {
  private sessionId: string;
  private directCosts: number = 0;
  private spawnCosts: SpawnCost[] = [];
  private startTime: number = Date.now();
  private endTime?: number;

  constructor(sessionId: string) {
    this.sessionId = sessionId;
  }

  /**
   * Add cost from a spawned agent via CLI
   */
  addSpawnCost(cost: Omit<SpawnCost, "timestamp">): void {
    this.spawnCosts.push({
      ...cost,
      timestamp: Date.now(),
    });
  }

  /**
   * Add direct cost from current session (e.g., Task() calls)
   */
  addDirectCost(cost: number): void {
    this.directCosts += cost;
  }

  /**
   * Get all spawn costs
   */
  getSpawnCosts(): readonly SpawnCost[] {
    return [...this.spawnCosts];
  }

  /**
   * Get total session cost (direct + spawn)
   */
  getSessionTotal(): number {
    const spawnTotal = this.spawnCosts.reduce((sum, c) => sum + c.cost, 0);
    return this.directCosts + spawnTotal;
  }

  /**
   * Get session summary
   */
  getSummary(): SessionCostSummary {
    return {
      sessionId: this.sessionId,
      directCosts: this.directCosts,
      spawnCosts: [...this.spawnCosts],
      totalCost: this.getSessionTotal(),
      startTime: this.startTime,
      endTime: this.endTime,
    };
  }

  /**
   * Mark session as ended
   */
  end(): void {
    this.endTime = Date.now();
  }

  /**
   * Reset tracker (for testing)
   */
  reset(): void {
    this.directCosts = 0;
    this.spawnCosts = [];
    this.startTime = Date.now();
    this.endTime = undefined;
  }

  /**
   * Format session summary for output
   */
  formatSummary(): string {
    const summary = this.getSummary();
    const lines: string[] = [
      "Session Cost Summary:",
      `├── Router direct costs: $${summary.directCosts.toFixed(4)}`,
      "├── Spawn costs:",
    ];

    if (summary.spawnCosts.length === 0) {
      lines.push("│   └── (none)");
    } else {
      const grouped = new Map<string, SpawnCost[]>();
      for (const cost of summary.spawnCosts) {
        const existing = grouped.get(cost.agentType) || [];
        existing.push(cost);
        grouped.set(cost.agentType, existing);
      }

      const entries = Array.from(grouped.entries());
      entries.forEach(([agentType, costs], idx) => {
        const totalCost = costs.reduce((sum, c) => sum + c.cost, 0);
        const isLast = idx === entries.length - 1;
        const prefix = isLast ? "└──" : "├──";
        lines.push(`│   ${prefix} ${agentType}: $${totalCost.toFixed(4)} (${costs.length}x)`);
      });
    }

    lines.push(`└── Total: $${summary.totalCost.toFixed(4)}`);

    if (summary.endTime) {
      const duration = summary.endTime - summary.startTime;
      lines.push(`Duration: ${(duration / 1000).toFixed(1)}s`);
    }

    return lines.join("\n");
  }
}

// Global tracker instance (per-session)
let globalTracker: SessionCostTracker | null = null;

/**
 * Get or create the session cost tracker
 */
export function getSessionCostTracker(sessionId?: string): SessionCostTracker {
  if (!globalTracker) {
    const id = sessionId || `session-${Date.now()}`;
    globalTracker = new SessionCostTracker(id);
  }
  return globalTracker;
}

/**
 * Reset the global tracker (for testing or new session)
 */
export function resetSessionCostTracker(sessionId?: string): void {
  const id = sessionId || `session-${Date.now()}`;
  globalTracker = new SessionCostTracker(id);
}

/**
 * End the current session and return summary
 */
export function endSession(): SessionCostSummary {
  if (!globalTracker) {
    throw new Error("No active session to end");
  }
  globalTracker.end();
  return globalTracker.getSummary();
}
