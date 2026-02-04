/**
 * Lifecycle management exports
 * Restart and graceful shutdown coordination
 */

export {
  RestartManager,
  getRestartManager,
  resetRestartManager,
  type RestartConfig,
} from "./restart.js";

export {
  onShutdown,
  initiateShutdown,
  setupSignalHandlers,
  registerChildProcessCleanup,
  isShutdownInProgress,
  resetShutdownState,
} from "./shutdown.js";
