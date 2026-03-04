/**
 * SelectModal component - arrow-navigable list with number shortcuts
 * - Arrow keys navigate options
 * - Number keys (1-9) for quick select
 * - Enter selects current option
 *
 * Split layout: scrollable message (top) + fixed options panel (bottom)
 * Options are ALWAYS visible regardless of message length.
 */

import React, { useState } from "react";
import { Box, Text, useInput } from "ink";
import type { ModalRequest, ModalResponse, SelectPayload } from "../../store/slices/modal.js";
import { colors } from "../../config/theme.js";
import { ScrollView } from "../primitives/ScrollView.js";

interface SelectModalProps {
  request: ModalRequest<SelectPayload>;
  onComplete: (response: ModalResponse) => void;
  /** Available terminal rows for scrollable content area (provided by ModalOverlay) */
  contentHeight?: number;
}

export function SelectModal({ request, onComplete, contentHeight }: SelectModalProps): JSX.Element {
  const payload = request.payload as SelectPayload;
  const [selectedIndex, setSelectedIndex] = useState(0);

  useInput(
    (input, key) => {
      if (key.return) {
        const option = payload.options[selectedIndex];
        if (option) {
          onComplete({
            type: "select",
            selected: option.value,
            index: selectedIndex,
          });
        }
      } else if (key.upArrow) {
        setSelectedIndex((prev) =>
          prev > 0 ? prev - 1 : payload.options.length - 1
        );
      } else if (key.downArrow) {
        setSelectedIndex((prev) =>
          prev < payload.options.length - 1 ? prev + 1 : 0
        );
      } else if (input && /^[1-9]$/.test(input)) {
        // Number key shortcuts (1-9)
        const index = parseInt(input, 10) - 1;
        if (index < payload.options.length) {
          const option = payload.options[index];
          if (option) {
            onComplete({
              type: "select",
              selected: option.value,
              index,
            });
          }
        }
      }
    },
    { isActive: true }
  );

  const helpText = contentHeight !== undefined
    ? "↑↓ or 1-9 Navigate • PgUp/PgDn Scroll • Enter Select • Esc Cancel"
    : "↑↓ or 1-9 Navigate • Enter Select • Esc Cancel";

  // Content region: message text
  const contentRegion = (
    <Text wrap="wrap">{payload.message}</Text>
  );

  // Options region: option list + help text
  const optionsRegion = (
    <Box flexDirection="column" flexShrink={0}>
      <Box flexDirection="column">
        {payload.options.map((option, index) => {
          const isSelected = index === selectedIndex;
          const numberKey = index < 9 ? `${index + 1}` : " ";

          return (
            <Box key={index}>
              <Text color={colors.muted}>{numberKey}. </Text>
              <Text color={isSelected ? colors.primary : colors.muted}>
                {isSelected ? "▶ " : "  "}
                {option.label}
              </Text>
            </Box>
          );
        })}
      </Box>

      <Box marginTop={1}>
        <Text dimColor>{helpText}</Text>
      </Box>
    </Box>
  );

  // Split layout: scrollable message (top) + fixed options (bottom)
  if (contentHeight !== undefined && contentHeight > 0) {
    return (
      <Box flexDirection="column">
        {/* TOP: Scrollable message region */}
        <ScrollView
          height={contentHeight}
          autoScroll={false}
          focused={false}
          disableArrowKeys={true}
        >
          {contentRegion}
        </ScrollView>

        {/* Separator */}
        <Box>
          <Text dimColor>{"─".repeat(44)}</Text>
        </Box>

        {/* BOTTOM: Fixed options panel — never clipped */}
        {optionsRegion}
      </Box>
    );
  }

  // Fallback: no split layout (full-screen mode or no height constraint)
  return (
    <Box flexDirection="column" gap={1}>
      {contentRegion}
      {optionsRegion}
    </Box>
  );
}
