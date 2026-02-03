/**
 * InputModal component - text input with placeholder
 * - Enter submits input
 * - Uses TextInput primitive for robust input handling
 */

import React, { useState } from "react";
import { Box, Text } from "ink";
import type { ModalRequest, ModalResponse, InputPayload } from "../../store/slices/modal.js";
import { TextInput } from "../primitives/TextInput.js";

interface InputModalProps {
  request: ModalRequest<InputPayload>;
  onComplete: (response: ModalResponse) => void;
}

export function InputModal({ request, onComplete }: InputModalProps): JSX.Element {
  const payload = request.payload as InputPayload;
  const [value, setValue] = useState("");

  const handleSubmit = () => {
    onComplete({ type: "input", value });
  };

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
