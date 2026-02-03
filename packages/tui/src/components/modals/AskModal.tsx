/**
 * AskModal component - displays question with optional button options
 * - If options provided: arrow key navigation, Enter selects
 * - If no options: free text input, Enter submits
 */

import React, { useState } from "react";
import { Box, Text, useInput } from "ink";
import type { ModalRequest, ModalResponse, AskPayload } from "../../store/slices/modal.js";
import { colors } from "../../config/theme.js";

interface AskModalProps {
  request: ModalRequest<AskPayload>;
  onComplete: (response: ModalResponse) => void;
}

export function AskModal({ request, onComplete }: AskModalProps): JSX.Element {
  const payload = request.payload as AskPayload;
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [inputValue, setInputValue] = useState(payload.defaultValue || "");

  const hasOptions = payload.options && payload.options.length > 0;

  useInput(
    (input, key) => {
      if (hasOptions) {
        // Arrow navigation mode
        if (key.upArrow) {
          setSelectedIndex((prev) =>
            prev > 0 ? prev - 1 : (payload.options?.length || 1) - 1
          );
        } else if (key.downArrow) {
          setSelectedIndex((prev) =>
            prev < (payload.options?.length || 1) - 1 ? prev + 1 : 0
          );
        } else if (key.return) {
          const value = payload.options?.[selectedIndex] || "";
          onComplete({ type: "ask", value });
        }
      } else {
        // Free text input mode
        if (key.return) {
          onComplete({ type: "ask", value: inputValue });
        } else if (key.backspace || key.delete) {
          setInputValue((prev) => prev.slice(0, -1));
        } else if (input && !key.ctrl && !key.meta) {
          setInputValue((prev) => prev + input);
        }
      }
    },
    { isActive: true }
  );

  return (
    <Box flexDirection="column" gap={1}>
      <Text>{payload.message}</Text>

      {hasOptions ? (
        <Box flexDirection="column" marginTop={1}>
          {payload.options?.map((option, index) => (
            <Box key={index}>
              <Text color={index === selectedIndex ? colors.primary : colors.muted}>
                {index === selectedIndex ? "▶ " : "  "}
                {option}
              </Text>
            </Box>
          ))}
        </Box>
      ) : (
        <Box marginTop={1}>
          <Text color={colors.muted}>&gt; </Text>
          <Text>{inputValue}</Text>
          <Text color={colors.primary}>▎</Text>
        </Box>
      )}

      <Box marginTop={1}>
        <Text dimColor>
          {hasOptions ? "↑↓ Navigate • Enter Select" : "Enter Submit"} • Esc Cancel
        </Text>
      </Box>
    </Box>
  );
}
