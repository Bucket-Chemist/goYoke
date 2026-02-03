/**
 * ConfirmModal component - Yes/No confirmation with destructive action styling
 * - Y/N keyboard shortcuts
 * - Enter submits current selection (default: No for safety)
 * - Destructive actions highlighted in red
 */

import React, { useState } from "react";
import { Box, Text, useInput } from "ink";
import type { ModalRequest, ModalResponse, ConfirmPayload } from "../../store/slices/modal.js";
import { colors } from "../../config/theme.js";

interface ConfirmModalProps {
  request: ModalRequest<ConfirmPayload>;
  onComplete: (response: ModalResponse) => void;
}

export function ConfirmModal({ request, onComplete }: ConfirmModalProps): JSX.Element {
  const payload = request.payload as ConfirmPayload;
  const [selected, setSelected] = useState<"yes" | "no">("no"); // Default to No for safety

  useInput(
    (input, key) => {
      if (key.return) {
        onComplete({
          type: "confirm",
          confirmed: selected === "yes",
          cancelled: false,
        });
      } else if (input === "y" || input === "Y") {
        setSelected("yes");
      } else if (input === "n" || input === "N") {
        setSelected("no");
      } else if (key.leftArrow || key.rightArrow) {
        setSelected((prev) => (prev === "yes" ? "no" : "yes"));
      }
    },
    { isActive: true }
  );

  const actionColor = payload.destructive ? colors.error : colors.warning;

  return (
    <Box flexDirection="column" gap={1}>
      <Text color={actionColor}>{payload.action}</Text>

      <Box marginTop={1} gap={2}>
        <Box>
          <Text
            color={selected === "yes" ? colors.success : colors.muted}
            bold={selected === "yes"}
          >
            {selected === "yes" ? "[Yes]" : " Yes "}
          </Text>
        </Box>
        <Box>
          <Text
            color={selected === "no" ? colors.primary : colors.muted}
            bold={selected === "no"}
          >
            {selected === "no" ? "[No]" : " No "}
          </Text>
        </Box>
      </Box>

      <Box marginTop={1}>
        <Text dimColor>Y/N or ←→ Navigate • Enter Confirm • Esc Cancel</Text>
      </Box>
    </Box>
  );
}
