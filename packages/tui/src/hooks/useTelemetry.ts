/**
 * Telemetry hook - watches Go hook output files for real-time updates
 * Monitors routing decisions, handoffs, and sharp edges
 */

import { watch } from "chokidar";
import { useEffect } from "react";
import { createReader, readNewLines } from "../utils/jsonl.js";
import { useStore } from "../store/index.js";
import { logger } from "../utils/logger.js";

const XDG_DATA_HOME = process.env["XDG_DATA_HOME"] ?? `${process.env["HOME"]}/.local/share`;

const readers = {
  routingDecisions: createReader(`${XDG_DATA_HOME}/gogent/routing-decisions.jsonl`),
  handoffs: createReader(`${process.env["HOME"]}/.claude/memory/handoffs.jsonl`),
  sharpEdges: createReader(`${XDG_DATA_HOME}/gogent/sharp-edges.jsonl`)
};

/**
 * Watch telemetry files for changes and update store
 * Updates are processed within 500ms of file change (chokidar default)
 */
export function useTelemetry(): void {
  const updateTelemetry = useStore((state) => state.updateTelemetry);

  useEffect(() => {
    const paths = Object.values(readers).map(r => r.path);

    logger.info("Starting telemetry watchers", { paths });

    const watcher = watch(paths, {
      persistent: true,
      ignoreInitial: true,
      awaitWriteFinish: {
        stabilityThreshold: 100,
        pollInterval: 50
      }
    });

    watcher.on("change", async (path) => {
      const key = Object.keys(readers).find(
        k => readers[k as keyof typeof readers].path === path
      ) as keyof typeof readers | undefined;

      if (!key) {
        logger.warn("Unknown file changed", { path });
        return;
      }

      try {
        const reader = readers[key];
        const { lines, newOffset } = await readNewLines(reader);
        reader.offset = newOffset;

        logger.debug(`Read ${lines.length} new lines from ${key}`, { path, newOffset });

        for (const line of lines) {
          updateTelemetry(key, line);
        }
      } catch (error) {
        logger.error(`Failed to read telemetry file: ${key}`, {
          error: error instanceof Error ? error.message : String(error),
          path
        });
      }
    });

    watcher.on("error", (error) => {
      logger.error("Telemetry watcher error", {
        error: error instanceof Error ? error.message : String(error)
      });
    });

    return () => {
      logger.info("Stopping telemetry watchers");
      watcher.close();
    };
  }, [updateTelemetry]);
}
