/**
 * Toast notification components
 * Displays temporary notifications with auto-dismiss
 */

import React from "react";
import { Box, Text } from "ink";
import { useStore } from "../store/index.js";
import { colors } from "../config/theme.js";

export function ToastContainer(): JSX.Element | null {
  const toasts = useStore((state) => state.toasts);

  if (toasts.length === 0) return null;

  const iconMap = {
    info: "ℹ",
    success: "✓",
    warning: "⚠",
    error: "✗",
  };

  const colorMap = {
    info: colors.primary,
    success: colors.success,
    warning: colors.warning,
    error: colors.error,
  };

  return (
    <Box flexDirection="column" position="absolute">
      {toasts.map((toast) => (
        <Box key={toast.id} borderStyle="round" borderColor={colorMap[toast.type]} paddingX={1} marginBottom={1}>
          <Text color={colorMap[toast.type]}>{iconMap[toast.type]} </Text>
          <Text wrap="wrap">{toast.message}</Text>
        </Box>
      ))}
    </Box>
  );
}
