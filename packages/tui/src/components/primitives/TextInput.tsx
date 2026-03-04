/**
 * TextInput primitive - styled input wrapper
 * Wraps TextInputCore with theme integration (borders, colors)
 * TextInputCore replaces ink-text-input to fix delete key border tearing
 */

import React from "react";
import { Box, Text } from "ink";
import { TextInputCore } from "./TextInputCore.js";
import { colors, borders } from "../../config/theme.js";

export interface TextInputProps {
  /**
   * Current input value
   */
  value: string;

  /**
   * Called when input value changes
   */
  onChange: (value: string) => void;

  /**
   * Called when Enter is pressed (optional - useKeymap handles this in most cases)
   */
  onSubmit?: () => void;

  /**
   * Placeholder text when empty
   */
  placeholder?: string;

  /**
   * Whether input is disabled
   * @default false
   */
  disabled?: boolean;

  /**
   * Whether input has focus
   * @default true
   */
  focused?: boolean;
}

/**
 * Styled text input component with submit handling
 * Captures keystrokes and submits on Enter
 */
export function TextInput({
  value,
  onChange,
  onSubmit,
  placeholder = "Type a message...",
  disabled = false,
  focused = true,
}: TextInputProps): JSX.Element {
  if (disabled) {
    return (
      <Box borderStyle={borders.input} borderColor={colors.muted} paddingX={1} width="100%">
        <Text color={colors.muted}>{placeholder}</Text>
      </Box>
    );
  }

  return (
    <Box
      borderStyle={borders.input}
      borderColor={focused ? colors.focused : colors.unfocused}
      paddingX={1}
      width="100%"
    >
      <TextInputCore
        value={value}
        onChange={onChange}
        onSubmit={onSubmit}
        placeholder={placeholder}
        focused={focused}
      />
    </Box>
  );
}
