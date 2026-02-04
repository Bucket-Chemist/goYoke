/**
 * Restart manager with exponential backoff
 * Handles automatic restart on crash with configurable retry logic
 */

export interface RestartConfig {
  maxAttempts: number;
  initialDelayMs: number;
  maxDelayMs: number;
  backoffMultiplier: number;
  successfulRunThresholdMs: number; // Reset backoff after this long
}

const DEFAULT_CONFIG: RestartConfig = {
  maxAttempts: 3,
  initialDelayMs: 1000,
  maxDelayMs: 30000,
  backoffMultiplier: 2,
  successfulRunThresholdMs: 60000, // 60 seconds
};

export class RestartManager {
  private attempts = 0;
  private lastAttemptTime = 0;
  private lastSuccessfulStartTime = 0;

  constructor(private config: RestartConfig = DEFAULT_CONFIG) {}

  /**
   * Check if restart should be attempted
   * Returns false if max attempts exceeded
   */
  shouldRestart(): boolean {
    return this.attempts < this.config.maxAttempts;
  }

  /**
   * Calculate delay for next restart attempt
   * Uses exponential backoff capped at maxDelayMs
   */
  getDelay(): number {
    const delay = Math.min(
      this.config.initialDelayMs *
        Math.pow(this.config.backoffMultiplier, this.attempts),
      this.config.maxDelayMs
    );
    return delay;
  }

  /**
   * Record a restart attempt
   * Updates attempt counter and timestamp
   */
  recordAttempt(): void {
    this.attempts++;
    this.lastAttemptTime = Date.now();
  }

  /**
   * Mark successful start
   * Used to detect if app ran long enough to reset backoff
   */
  recordSuccessfulStart(): void {
    this.lastSuccessfulStartTime = Date.now();
  }

  /**
   * Check if backoff should reset based on run duration
   * Resets if app ran longer than successfulRunThresholdMs
   */
  checkAndResetIfSuccessful(): void {
    if (this.lastSuccessfulStartTime === 0) {
      return;
    }

    const runDuration = Date.now() - this.lastSuccessfulStartTime;
    if (runDuration >= this.config.successfulRunThresholdMs) {
      this.reset();
    }
  }

  /**
   * Reset attempt counter
   * Called after successful run or manual reset
   */
  reset(): void {
    this.attempts = 0;
    this.lastAttemptTime = 0;
  }

  /**
   * Get current restart state for debugging
   */
  getState(): {
    attempts: number;
    maxAttempts: number;
    lastAttemptTime: number;
    canRestart: boolean;
    nextDelay: number;
  } {
    return {
      attempts: this.attempts,
      maxAttempts: this.config.maxAttempts,
      lastAttemptTime: this.lastAttemptTime,
      canRestart: this.shouldRestart(),
      nextDelay: this.getDelay(),
    };
  }
}

/**
 * Global restart manager instance
 * Singleton pattern for app-wide restart coordination
 */
let globalRestartManager: RestartManager | null = null;

export function getRestartManager(
  config?: Partial<RestartConfig>
): RestartManager {
  if (!globalRestartManager) {
    globalRestartManager = new RestartManager(
      config ? { ...DEFAULT_CONFIG, ...config } : DEFAULT_CONFIG
    );
  }
  return globalRestartManager;
}

/**
 * Reset global restart manager
 * Primarily for testing
 */
export function resetRestartManager(): void {
  globalRestartManager = null;
}
