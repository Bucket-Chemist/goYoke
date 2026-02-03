/**
 * InputModal component - text input with placeholder
 * - Enter submits input
 * - Displays placeholder when input is empty
 */

import React, { useState } from "react";
import { Box, Text, useInput } from "ink";
import type { ModalRequest, ModalResponse, InputPayload } from "../../store/slices/modal.js";
import { colors } from "../../config/theme.js";

interface InputModalProps {
  request: ModalRequest<InputPayload>;
  onComplete: (response: ModalResponse) => void;
}

export function InputModal({ request, onComplete }: InputModalProps): JSX.Element {
  const payload = request.payload as InputPayload;
  const [value, setValue] = useState("");

  useInput(
    (input, key) => {
      if (key.return) {
        onComplete({ type: "input", value });
      } else if (key.backspace || key.delete) {
        setValue((prev) => prev.slice(0, -1));
      } else if (input && !key.ctrl && !key.meta) {
        setValue((prev) => prev + input);
      }
    },
    { isActive: true }
  );

  return (
    <Box flexDirection="column" gap={1}>
      <Text>{payload.prompt}</Text>

      <Box marginTop={1}>
        <Text color={colors.muted}>&gt; </Text>
        {value ? (
          <Text>{value}</Text>
        ) : (
          <Text dimColor>{payload.placeholder || "Type here..."}</Text>
        )}
        <Text color={colors.primary}>▎</Text>
      </Box>

      <Box marginTop={1}>
        <Text dimColor>Enter Submit • Esc Cancel</Text>
      </Box>
    </Box>
  );
}
