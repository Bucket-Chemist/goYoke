/**
 * TextInput primitive - styled input wrapper
 * Wraps ink-text-input with theme integration and submit handling
 */

import React from "react";
import TextInputLib from "ink-text-input";
import { Box, Text } from "ink";
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
   * Called when Enter is pressed
   */
  onSubmit: () => void;

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
      <Box borderStyle={borders.input} borderColor={colors.muted} paddingX={1}>
        <Text color={colors.muted}>{placeholder}</Text>
      </Box>
    );
  }

  return (
    <Box
      borderStyle={borders.input}
      borderColor={focused ? colors.focused : colors.unfocused}
      paddingX={1}
    >
      <TextInputLib
        value={value}
        onChange={onChange}
        onSubmit={onSubmit}
        placeholder={placeholder}
        focus={focused}
      />
    </Box>
  );
}
