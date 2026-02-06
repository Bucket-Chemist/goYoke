/**
 * ScrollView primitive - line-aware scrollable container
 * Unlike Viewport (item-based), this scrolls by terminal lines to prevent overflow
 *
 * Key differences from Viewport:
 * - Scrolls by rendered lines, not items (messages can be multi-line)
 * - Uses marginTop offset + overflow:hidden for clipping
 * - Measures actual content height in terminal rows
 * - Prevents text overflow that causes garbled rendering
 */

import React, { useState, useEffect, useRef, useMemo } from "react";
import { Box, Text, useInput, measureElement } from "ink";
import { colors } from "../../config/theme.js";
import { logger } from "../../utils/logger.js";

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
   * Content to render
   */
  children: React.ReactNode;
}

/**
 * Line-aware scrollable viewport
 * Renders children and scrolls by terminal lines rather than items
 */
export function ScrollView({
  height,
  width,
  autoScroll = true,
  focused = false,
  children,
}: ScrollViewProps): JSX.Element {
  const [scrollOffset, setScrollOffset] = useState(0);
  const [userScrolledUp, setUserScrolledUp] = useState(false);
  const contentRef = useRef<any>(null);
  const [contentHeight, setContentHeight] = useState(0);

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

  // Auto-scroll to bottom when content grows (if not manually scrolled up)
  // Simple rule: if user hasn't scrolled up, always show the latest content.
  useEffect(() => {
    if (autoScroll && !userScrolledUp) {
      setScrollOffset(maxOffset);
    }
  }, [maxOffset, autoScroll, userScrolledUp]);

  // Clamp scroll offset when max changes
  useEffect(() => {
    setScrollOffset((prev) => Math.min(prev, maxOffset));
  }, [maxOffset]);

  // Scroll controls (only active when focused)
  useInput(
    (input, key) => {
      if (!focused) return;

      let newOffset = scrollOffset;
      let scrolled = false;

      if (key.upArrow) {
        newOffset = Math.max(0, scrollOffset - 1);
        scrolled = true;
      } else if (key.downArrow) {
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
    { isActive: focused }
  );

  // Calculate scroll indicator values
  const showScrollIndicator = contentHeight > height;
  const visibleStart = scrollOffset + 1;
  const visibleEnd = Math.min(scrollOffset + height, contentHeight);
  const scrollPercentage =
    maxOffset > 0 ? Math.round((scrollOffset / maxOffset) * 100) : 100;

  return (
    <Box flexDirection="column" height={height}>
      {/* Content area with overflow clipping */}
      <Box
        flexDirection="column"
        flexGrow={1}
        overflow="hidden"
        height={height - (showScrollIndicator ? 1 : 0)}
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

      {/* Scroll position indicator */}
      {showScrollIndicator && (
        <Box justifyContent="flex-end">
          <Text color={colors.muted} dimColor>
            [lines {visibleStart}-{visibleEnd} of {contentHeight}] {scrollPercentage}%
            {userScrolledUp && " (paused)"}
          </Text>
        </Box>
      )}
    </Box>
  );
}
