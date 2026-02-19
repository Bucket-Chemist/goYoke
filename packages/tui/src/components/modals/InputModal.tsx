/**
 * InputModal component - text input with placeholder
 * - Enter submits input
 * - Escape warns before discarding unsaved text
 * - Uses TextInput primitive for robust input handling
 */

import React, { useState } from "react";
import { Box, Text, useInput } from "ink";
import type { ModalRequest, ModalResponse, InputPayload } from "../../store/slices/modal.js";
import { TextInput } from "../primitives/TextInput.js";

interface InputModalProps {
  request: ModalRequest<InputPayload>;
  onComplete: (response: ModalResponse) => void;
  onCancel: () => void;
}

export function InputModal({ request, onComplete, onCancel }: InputModalProps): JSX.Element {
  const payload = request.payload as InputPayload;
  const [value, setValue] = useState("");
  const [showConfirm, setShowConfirm] = useState(false);

  const hasUnsavedText = value.trim().length > 0;

  const handleSubmit = () => {
    onComplete({ type: "input", value });
  };

  const handleEscape = () => {
    if (hasUnsavedText && !showConfirm) {
      setShowConfirm(true);
    } else {
      onCancel();
    }
  };

  // Handle Escape key
  useInput((input, key) => {
    if (key.escape) {
      handleEscape();
    }
  }, { isActive: !showConfirm });

  // Handle confirmation dialog input
  useInput((input, key) => {
    if (key.return || input.toLowerCase() === "y") {
      onCancel();
    } else if (input.toLowerCase() === "n") {
      setShowConfirm(false);
    }
  }, { isActive: showConfirm });

  if (showConfirm) {
    return (
      <Box flexDirection="column" gap={1}>
        <Text color="yellow">⚠ Discard typed text?</Text>
        <Box marginTop={1}>
          <Text>You have unsaved text: "{value}"</Text>
        </Box>
        <Box marginTop={1}>
          <Text dimColor>Y Discard • N Continue editing</Text>
        </Box>
      </Box>
    );
  }

  return (
    <Box flexDirection="column" gap={1}>
      <Text>{payload.prompt}</Text>

      <Box marginTop={1}>
        <TextInput
          value={value}
          onChange={setValue}
          onSubmit={handleSubmit}
          placeholder={payload.placeholder || "Type here..."}
          focused={true}
        />
      </Box>

      <Box marginTop={1}>
        <Text dimColor>Enter Submit • Esc Cancel</Text>
      </Box>
    </Box>
  );
}
