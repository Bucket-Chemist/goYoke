/**
 * useMouse hook — terminal mouse event handling for Ink
 *
 * Enables SGR mouse mode (1006) with button-event + button-motion tracking.
 * Parses mouse escape sequences from raw stdin and exposes them via callback.
 *
 * SGR protocol: \x1b[<button;col;rowM (press) / \x1b[<button;col;rowm (release)
 * Button codes: 0=left, 1=middle, 2=right, 64=wheelUp, 65=wheelDown
 * Bit 32 set = motion event (drag)
 *
 * Mouse tracking is ref-counted: enabled when first consumer mounts,
 * disabled when last consumer unmounts. Safe for multiple ScrollView instances.
 *
 * Interception strategy:
 * Ink reads stdin via `readable` event + `stdin.read()` (pull/paused mode),
 * NOT via `data` events. We must intercept BOTH paths:
 * - `stdin.read()` — for Ink's paused-mode pull reads
 * - `stdin.emit('data')` — for any flowing-mode consumers
 * Mouse sequences are stripped before downstream consumers see the data.
 */

import { useEffect, useRef } from "react";

export interface MouseEvent {
  /** 1-indexed terminal column */
  x: number;
  /** 1-indexed terminal row */
  y: number;
  /** Button code: 0=left, 1=middle, 2=right, 64=wheelUp, 65=wheelDown */
  button: number;
  /** True on button press */
  isPress: boolean;
  /** True on button release */
  isRelease: boolean;
  /** True if this is a motion event (drag) */
  isDrag: boolean;
}

export interface UseMouseOptions {
  /** Whether event callbacks fire. Mouse tracking stays enabled regardless. */
  isActive?: boolean;
  /** Callback for mouse events */
  onMouseEvent?: (event: MouseEvent) => void;
}

// SGR mouse event regex: \x1b[<button;col;row(M|m)
const SGR_MOUSE_RE = /\x1b\[<(\d+);(\d+);(\d+)([Mm])/g;

// Module-level ref count — mouse mode stays enabled while any consumer is mounted
let mouseRefCount = 0;

// Interceptor state — installed once when mouse tracking is first enabled
// eslint-disable-next-line @typescript-eslint/no-explicit-any
type EmitFn = (event: string | symbol, ...args: any[]) => boolean;
// eslint-disable-next-line @typescript-eslint/no-explicit-any
type ReadFn = (size?: number) => any;
let originalEmit: EmitFn | null = null;
let originalRead: ReadFn | null = null;

// Registered mouse callbacks — dispatched by interceptors
const mouseCallbacks = new Set<(event: MouseEvent) => void>();

/**
 * Parse SGR mouse sequences from a string and dispatch to all registered callbacks.
 */
function parseAndDispatchMouse(str: string): void {
  SGR_MOUSE_RE.lastIndex = 0;
  let match;
  while ((match = SGR_MOUSE_RE.exec(str)) !== null) {
    const rawButton = parseInt(match[1]!, 10);
    const mouseEvent: MouseEvent = {
      x: parseInt(match[2]!, 10),
      y: parseInt(match[3]!, 10),
      button: rawButton & ~32, // strip motion bit
      isPress: match[4] === "M",
      isRelease: match[4] !== "M",
      isDrag: (rawButton & 32) !== 0,
    };
    for (const cb of mouseCallbacks) {
      cb(mouseEvent);
    }
  }
}

/**
 * Strip SGR mouse sequences from a string.
 * Returns the cleaned string (may be empty if input was all mouse data).
 */
function stripMouseSequences(str: string): string {
  return str.replace(SGR_MOUSE_RE, "");
}

/**
 * stdin.read() interceptor — handles Ink's paused-mode pull reads.
 *
 * Ink uses `readable` event + `stdin.read()` to pull data. We intercept
 * read() to parse mouse events and strip sequences before Ink sees them.
 * If a chunk is entirely mouse data, we recurse to get the next chunk
 * so Ink's while loop doesn't see empty data.
 */
function stdinReadInterceptor(size?: number): string | Buffer | null {
  const chunk = originalRead!(size);
  if (chunk === null) return null;

  const str = typeof chunk === "string" ? chunk : chunk.toString("utf8");
  if (!SGR_MOUSE_RE.test(str)) return chunk; // fast path: no mouse data

  parseAndDispatchMouse(str);
  const cleaned = stripMouseSequences(str);

  if (cleaned.length === 0) {
    // Entire chunk was mouse data — try next chunk from buffer
    // so Ink's while((chunk = read()) !== null) loop continues correctly
    return stdinReadInterceptor(size);
  }

  return cleaned;
}

/**
 * stdin.emit() interceptor — handles flowing-mode data events.
 *
 * If any consumer puts stdin into flowing mode (via .on('data')),
 * this strips mouse sequences from data events before they propagate.
 */
function stdinEmitInterceptor(
  this: NodeJS.ReadStream,
  event: string | symbol,
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  ...args: any[]
): boolean {
  if (event !== "data" || !originalEmit) {
    return originalEmit!.call(this, event, ...args);
  }

  const raw = args[0];
  const str =
    raw instanceof Buffer
      ? raw.toString("utf8")
      : typeof raw === "string"
        ? raw
        : String(raw);

  if (!SGR_MOUSE_RE.test(str)) {
    return originalEmit.call(this, event, ...args); // fast path
  }

  parseAndDispatchMouse(str);
  const cleaned = stripMouseSequences(str);

  if (cleaned.length > 0) {
    return originalEmit.call(this, event, cleaned);
  }

  // All data was mouse sequences — suppress entirely
  return true;
}

function enableMouseTracking(): void {
  if (mouseRefCount === 0) {
    // Install both interceptors before enabling mouse mode
    originalRead = process.stdin.read.bind(process.stdin) as ReadFn;
    originalEmit = process.stdin.emit.bind(process.stdin) as EmitFn;
    process.stdin.read = stdinReadInterceptor as ReadFn;
    process.stdin.emit = stdinEmitInterceptor as EmitFn;
    process.stdout.write("\x1b[?1000h\x1b[?1002h\x1b[?1006h");
  }
  mouseRefCount++;
}

function disableMouseTracking(): void {
  mouseRefCount--;
  if (mouseRefCount <= 0) {
    mouseRefCount = 0;
    process.stdout.write("\x1b[?1000l\x1b[?1002l\x1b[?1006l");
    // Restore originals
    if (originalRead) {
      process.stdin.read = originalRead as NodeJS.ReadStream["read"];
      originalRead = null;
    }
    if (originalEmit) {
      process.stdin.emit = originalEmit as NodeJS.ReadStream["emit"];
      originalEmit = null;
    }
  }
}

/**
 * Hook to enable terminal mouse tracking and parse SGR mouse events.
 *
 * Mouse mode is enabled on mount and disabled on unmount (ref-counted).
 * Events are only dispatched when `isActive` is true.
 */
export function useMouse({ isActive = true, onMouseEvent }: UseMouseOptions): void {
  const callbackRef = useRef(onMouseEvent);
  callbackRef.current = onMouseEvent;

  // Enable mouse tracking on mount (ref-counted, independent of isActive)
  useEffect(() => {
    enableMouseTracking();
    return () => disableMouseTracking();
  }, []);

  // Register/unregister mouse callback with the interceptors
  useEffect(() => {
    if (!isActive) return;

    const handler = (event: MouseEvent): void => {
      callbackRef.current?.(event);
    };

    mouseCallbacks.add(handler);
    return () => {
      mouseCallbacks.delete(handler);
    };
  }, [isActive]);
}
