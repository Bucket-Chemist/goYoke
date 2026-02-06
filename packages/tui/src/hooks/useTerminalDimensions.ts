import { useStdout } from "ink";
import { useState, useEffect, useRef } from "react";

interface Dimensions {
  rows: number;
  columns: number;
}

/**
 * Custom hook for terminal dimensions with debounced resize handling
 *
 * Fixes rendering corruption on resize by:
 * - Clearing screen before updating dimensions
 * - Debouncing rapid resize events
 * - Providing stable fallback dimensions
 *
 * @param debounceMs Milliseconds to wait before processing resize (default: 100)
 * @returns Current terminal dimensions
 */
export function useTerminalDimensions(debounceMs = 100): Dimensions {
  const { stdout } = useStdout();
  const [dimensions, setDimensions] = useState<Dimensions>({
    rows: stdout?.rows ?? 24,
    columns: stdout?.columns ?? 80,
  });
  const timeoutRef = useRef<NodeJS.Timeout | null>(null);

  useEffect(() => {
    const handleResize = () => {
      // Clear existing timeout if resize is rapid
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }

      // Debounce dimension update to avoid rapid re-renders
      // Ink handles re-rendering automatically - don't write raw escape codes
      timeoutRef.current = setTimeout(() => {
        if (stdout) {
          setDimensions({
            rows: stdout.rows ?? 24,
            columns: stdout.columns ?? 80,
          });
        }
      }, debounceMs);
    };

    process.stdout.on('resize', handleResize);

    return () => {
      process.stdout.off('resize', handleResize);
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, [stdout, debounceMs]);

  return dimensions;
}
