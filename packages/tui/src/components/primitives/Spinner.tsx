/**
 * Spinner primitive - loading indicator
 * Thin wrapper around ink-spinner with theme integration
 */

import React from "react";
import SpinnerLib from "ink-spinner";
import { Text } from "ink";
import { colors } from "../../config/theme.js";

export interface SpinnerProps {
  /**
   * Spinner animation type
   * @default "dots"
   */
  type?: "dots" | "line" | "arc" | "arrow" | "bouncingBar" | "bouncingBall";
}

/**
 * Animated spinner component for loading states
 */
export function Spinner({ type = "dots" }: SpinnerProps): JSX.Element {
  return (
    <Text color={colors.primary}>
      <SpinnerLib type={type} />
    </Text>
  );
}
