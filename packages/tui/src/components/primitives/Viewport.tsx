/**
 * Viewport primitive - scrollable list container
 * Handles scrolling, resize, and position tracking for message lists
 */

import React, { useState, useEffect, useMemo } from "react";
import { Box, Text, useInput } from "ink";
import { colors } from "../../config/theme.js";

export interface ViewportProps<T> {
  /**
   * Array of items to render
   */
  items: T[];

  /**
   * Render function for each item
   */
  renderItem: (item: T, index: number) => React.ReactNode;

  /**
   * Viewport height (number of visible items)
   * @default 10
   */
  height?: number;

  /**
   * Whether viewport has focus for scroll controls
   * @default false
   */
  focused?: boolean;

  /**
   * Auto-scroll to bottom when new items added
   * @default true
   */
  autoScroll?: boolean;
}

/**
 * Scrollable viewport for rendering lists
 * Features:
 * - Arrow key scrolling (when focused)
 * - Auto-scroll to bottom (configurable)
 * - Scroll position preserved on resize
 * - Page up/down support
 */
export function Viewport<T>({
  items,
  renderItem,
  height = 10,
  focused = false,
  autoScroll = true,
}: ViewportProps<T>): JSX.Element {
  const [scrollOffset, setScrollOffset] = useState(0);

  // Auto-scroll to bottom when new items arrive
  useEffect(() => {
    if (autoScroll && items.length > height) {
      setScrollOffset(Math.max(0, items.length - height));
    }
  }, [items.length, height, autoScroll]);

  // Clamp scroll offset when items or height changes
  const maxOffset = useMemo(() => {
    return Math.max(0, items.length - height);
  }, [items.length, height]);

  useEffect(() => {
    setScrollOffset((prev) => Math.min(prev, maxOffset));
  }, [maxOffset]);

  // Scroll controls (only active when focused)
  useInput(
    (input, key) => {
      if (!focused) return;

      if (key.upArrow) {
        setScrollOffset((prev) => Math.max(0, prev - 1));
      } else if (key.downArrow) {
        setScrollOffset((prev) => Math.min(maxOffset, prev + 1));
      } else if (key.pageUp) {
        setScrollOffset((prev) => Math.max(0, prev - height));
      } else if (key.pageDown) {
        setScrollOffset((prev) => Math.min(maxOffset, prev + height));
      }
    },
    { isActive: focused }
  );

  // Slice visible items
  const visibleItems = useMemo(() => {
    return items.slice(scrollOffset, scrollOffset + height);
  }, [items, scrollOffset, height]);

  // Scroll indicator
  const showScrollIndicator = items.length > height;
  const scrollPercentage =
    maxOffset > 0 ? Math.round((scrollOffset / maxOffset) * 100) : 100;

  return (
    <Box flexDirection="column" flexGrow={1}>
      {/* Message list */}
      <Box flexDirection="column" flexGrow={1}>
        {visibleItems.length === 0 ? (
          <Text color={colors.muted} italic>
            No messages yet. Start a conversation!
          </Text>
        ) : (
          visibleItems.map((item, index) => (
            <Box key={scrollOffset + index} flexDirection="column">
              {renderItem(item, scrollOffset + index)}
            </Box>
          ))
        )}
      </Box>

      {/* Scroll indicator */}
      {showScrollIndicator && (
        <Box justifyContent="flex-end">
          <Text color={colors.muted} dimColor>
            [{scrollOffset + 1}-{Math.min(scrollOffset + height, items.length)}/
            {items.length}] {scrollPercentage}%
          </Text>
        </Box>
      )}
    </Box>
  );
}
