/**
 * ScrollView primitive - line-aware scrollable container
 * Unlike Viewport (item-based), this scrolls by terminal lines to prevent overflow
 *
 * Key differences from Viewport:
 * - Scrolls by rendered lines, not items (messages can be multi-line)
 * - Uses marginTop offset + overflow:hidden for clipping
 * - Measures actual content height in terminal rows
 * - Prevents text overflow that causes garbled rendering
 *
 * Features:
 * - Keyboard: PageUp/PageDown (full page), Up/Down (line), Ctrl+A/E (top/bottom)
 * - Visual scrollbar with thumb/track (█/░)
 * - Mouse wheel scrolling (via SGR mouse mode)
 * - Click+drag on scrollbar for fast positioning
 */

import React, { useState, useEffect, useRef, useMemo, useCallback } from "react";
import { Box, Text, useInput, measureElement } from "ink";
import { useMouse } from "../../hooks/useMouse.js";
import type { MouseEvent as TermMouseEvent } from "../../hooks/useMouse.js";
import { colors } from "../../config/theme.js";
import { logger } from "../../utils/logger.js";

/** Lines scrolled per mouse wheel tick */
const WHEEL_SCROLL_LINES = 3;

export interface ScrollViewProps {
  /**
   * Available terminal rows for this container
   */
  height: number;

  /**
   * Available columns (for word wrap awareness)
   */
  width?: number;

  /**
   * Auto-scroll to bottom when new content arrives
   * @default true
   */
  autoScroll?: boolean;

  /**
   * Enable keyboard scrolling
   * @default false
   */
  focused?: boolean;

  /**
   * Allow scroll input (mouse wheel, PgUp/PgDn, Ctrl+A/E) independently of full focus.
   * When true, scrolling works even if `focused` is false (e.g., while a modal is active).
   * Arrow key scrolling still requires `focused` to be true.
   * @default undefined (follows `focused`)
   */
  scrollable?: boolean;

  /**
   * When true, only PageUp/PageDown + Ctrl+A/E are bound (not Up/Down arrows).
   * Use when the parent component owns Up/Down for another purpose (e.g. input history).
   * @default false
   */
  disableArrowKeys?: boolean;

  /**
   * Increment to force scroll-to-bottom (resets userScrolledUp).
   * Used when a modal appears so the user sees latest content.
   * @default 0
   */
  forceScrollToBottom?: number;

  /**
   * Content to render
   */
  children: React.ReactNode;
}

/**
 * Line-aware scrollable viewport with visual scrollbar
 * Renders children and scrolls by terminal lines rather than items
 */
export function ScrollView({
  height,
  width,
  autoScroll = true,
  focused = false,
  scrollable,
  disableArrowKeys = false,
  forceScrollToBottom = 0,
  children,
}: ScrollViewProps): JSX.Element {
  // Scroll input (mouse wheel, PgUp/PgDn, Ctrl+A/E) can be enabled independently
  // of full focus. Arrow keys still require `focused` to be true.
  const isScrollActive = scrollable ?? focused;
  const [scrollOffset, setScrollOffset] = useState(0);
  const [userScrolledUp, setUserScrolledUp] = useState(false);
  const contentRef = useRef<any>(null);
  const [contentHeight, setContentHeight] = useState(0);

  // Refs for stable mouse callback (avoids stale closures)
  const scrollOffsetRef = useRef(0);
  scrollOffsetRef.current = scrollOffset;
  const maxOffsetRef = useRef(0);
  const dragRef = useRef<{ y: number; offset: number } | null>(null);

  // Measure content height whenever children change
  useEffect(() => {
    if (contentRef.current) {
      try {
        const measured = measureElement(contentRef.current);
        const newHeight = measured.height;

        void logger.debug("ScrollView: content measured", {
          contentHeight: newHeight,
          viewportHeight: height,
          scrollOffset,
          userScrolledUp,
        });

        setContentHeight(newHeight);
      } catch (error) {
        // measureElement may fail during initial render
        void logger.warn("ScrollView: measurement failed", {
          error: String(error),
        });
      }
    }
  }, [children, height]);

  // Calculate max scroll offset
  const maxOffset = useMemo(() => {
    return Math.max(0, contentHeight - height);
  }, [contentHeight, height]);
  maxOffsetRef.current = maxOffset;

  // Auto-scroll to bottom when content grows (if not manually scrolled up)
  useEffect(() => {
    if (autoScroll && !userScrolledUp) {
      setScrollOffset(maxOffset);
    }
  }, [maxOffset, autoScroll, userScrolledUp]);

  // Clamp scroll offset when max changes
  useEffect(() => {
    setScrollOffset((prev) => Math.min(prev, maxOffset));
  }, [maxOffset]);

  // Force scroll to bottom when prop increments (e.g., modal appears)
  useEffect(() => {
    if (forceScrollToBottom > 0) {
      setUserScrolledUp(false);
      setScrollOffset(maxOffsetRef.current);
    }
  }, [forceScrollToBottom]);

  // Stable scroll helper — sets offset + tracks user scroll state
  const applyScroll = useCallback((target: number) => {
    const curMax = maxOffsetRef.current;
    const clamped = Math.max(0, Math.min(curMax, target));
    setScrollOffset(clamped);
    setUserScrolledUp(clamped < curMax);
  }, []);

  // Keyboard scroll controls
  // Scroll keys (PgUp/PgDn, Ctrl+A/E) activate when scrollable OR focused.
  // Arrow keys additionally require focused (they may be owned by the parent for other purposes).
  useInput(
    (input, key) => {
      let newOffset = scrollOffset;
      let scrolled = false;

      if (!disableArrowKeys && focused && key.upArrow) {
        newOffset = Math.max(0, scrollOffset - 1);
        scrolled = true;
      } else if (!disableArrowKeys && focused && key.downArrow) {
        newOffset = Math.min(maxOffset, scrollOffset + 1);
        scrolled = true;
      } else if (key.pageUp) {
        newOffset = Math.max(0, scrollOffset - height);
        scrolled = true;
      } else if (key.pageDown) {
        newOffset = Math.min(maxOffset, scrollOffset + height);
        scrolled = true;
      } else if (key.ctrl && input === "a") {
        // Home: Ctrl-A jump to top
        newOffset = 0;
        scrolled = true;
      } else if (key.ctrl && input === "e") {
        // End: Ctrl-E jump to bottom
        newOffset = maxOffset;
        scrolled = true;
        setUserScrolledUp(false); // Explicit jump to bottom re-enables auto-scroll
      }

      if (scrolled && newOffset !== scrollOffset) {
        setScrollOffset(newOffset);

        // Track if user scrolled away from bottom
        if (newOffset < maxOffset) {
          setUserScrolledUp(true);
        } else {
          setUserScrolledUp(false);
        }
      }
    },
    { isActive: isScrollActive }
  );

  // Mouse event handler — wheel scroll + click/drag on scrollbar
  const handleMouseEvent = useCallback(
    (event: TermMouseEvent) => {
      // Mouse wheel up
      if (event.button === 64 && event.isPress) {
        applyScroll(scrollOffsetRef.current - WHEEL_SCROLL_LINES);
        return;
      }
      // Mouse wheel down
      if (event.button === 65 && event.isPress) {
        applyScroll(scrollOffsetRef.current + WHEEL_SCROLL_LINES);
        return;
      }

      // Left click press — start drag
      if (event.button === 0 && event.isPress && !event.isDrag) {
        dragRef.current = { y: event.y, offset: scrollOffsetRef.current };
        return;
      }

      // Left button drag — scroll proportionally to Y delta
      if (event.isDrag && event.button === 0 && dragRef.current) {
        const deltaY = event.y - dragRef.current.y;
        // Ratio: dragging through full viewport height scrolls through full content
        const ratio = maxOffsetRef.current / (height || 1);
        const newOffset = Math.round(dragRef.current.offset + deltaY * ratio);
        applyScroll(newOffset);
        return;
      }

      // Left button release — end drag
      if (event.button === 0 && event.isRelease) {
        dragRef.current = null;
      }
    },
    [height, applyScroll]
  );

  useMouse({ isActive: isScrollActive, onMouseEvent: handleMouseEvent });

  // Scrollbar geometry
  const showScrollbar = contentHeight > height;
  const trackHeight = height;
  const thumbSize = showScrollbar
    ? Math.max(1, Math.round(trackHeight * (height / contentHeight)))
    : 0;
  const thumbPos =
    maxOffset > 0
      ? Math.round((scrollOffset / maxOffset) * (trackHeight - thumbSize))
      : 0;

  // Build scrollbar track string (single render, no per-row components)
  const scrollbarTrack = useMemo(() => {
    if (!showScrollbar) return "";
    return Array.from({ length: trackHeight }, (_, i) =>
      i >= thumbPos && i < thumbPos + thumbSize ? "\u2588" : "\u2591"
    ).join("\n");
  }, [showScrollbar, trackHeight, thumbPos, thumbSize]);

  return (
    <Box flexDirection="column" height={height}>
      <Box flexDirection="row" flexGrow={1}>
        {/* Content area with overflow clipping */}
        <Box
          flexDirection="column"
          flexGrow={1}
          overflow="hidden"
        >
          {/* Inner content box with negative marginTop for scrolling */}
          {/* flexShrink={0} is CRITICAL: without it, Yoga shrinks this box to fit
              the parent's height, making measureElement return the viewport height
              instead of the actual content height, so maxOffset=0 and scrolling
              never works. */}
          <Box
            ref={contentRef}
            flexDirection="column"
            flexShrink={0}
            marginTop={-scrollOffset}
          >
            {children}
          </Box>
        </Box>

        {/* Visual scrollbar — track (░) + thumb (█) */}
        {showScrollbar && (
          <Box width={1} flexDirection="column" flexShrink={0} overflow="hidden">
            <Text color={colors.muted}>{scrollbarTrack}</Text>
          </Box>
        )}
      </Box>

      {/* Auto-scroll paused indicator */}
      {userScrolledUp && (
        <Box
          justifyContent="center"
          paddingX={1}
        >
          <Text color={colors.warning}>
            ⏸ Auto-scroll paused • Press End to resume
          </Text>
        </Box>
      )}
    </Box>
  );
}
