/**
 * ProviderTabs component - horizontal provider navigation
 * Allows switching between AI providers (Anthropic, Google, OpenAI, Local)
 * Uses Shift+Tab keybinding for provider cycling
 */

import React from "react";
import { Box, Text } from "ink";
import { colors } from "../config/theme.js";
import { useStore } from "../store/index.js";
import { useKeymap, type KeyBinding } from "../hooks/useKeymap.js";
import type { ProviderId } from "../store/types.js";

/**
 * Provider display order
 */
const PROVIDER_ORDER: ProviderId[] = ["anthropic", "google", "openai", "local"];

/**
 * Provider display names
 */
const PROVIDER_NAMES: Record<ProviderId, string> = {
  anthropic: "Anthropic",
  google: "Google",
  openai: "OpenAI",
  local: "Local",
};

interface ProviderTabsProps {
  enabled?: boolean;
}

/**
 * ProviderTabs component - displays providers horizontally with Shift+Tab cycling
 *
 * @param enabled - Whether keyboard shortcuts are active (default: true)
 *
 * @example
 * ```tsx
 * <ProviderTabs enabled={!streaming} />
 * ```
 */
export function ProviderTabs({ enabled = true }: ProviderTabsProps): JSX.Element {
  const activeProvider = useStore((state) => state.activeProvider);
  const setActiveProvider = useStore((state) => state.setActiveProvider);

  // Cycle to next provider on Shift+Tab
  const cycleProvider = (): void => {
    const currentIndex = PROVIDER_ORDER.indexOf(activeProvider);
    const nextIndex = (currentIndex + 1) % PROVIDER_ORDER.length;
    setActiveProvider(PROVIDER_ORDER[nextIndex]!);
  };

  // Build keyboard binding for Shift+Tab
  const bindings: KeyBinding[] = [
    {
      key: "tab",
      shift: true,
      action: cycleProvider,
      description: "Switch to next provider",
    },
  ];

  useKeymap(bindings, enabled);

  return (
    <Box gap={2}>
      {PROVIDER_ORDER.map((providerId) => {
        const isActive = activeProvider === providerId;

        return (
          <Text
            key={providerId}
            color={isActive ? colors.primary : colors.muted}
            bold={isActive}
          >
            {isActive && "● "}
            {PROVIDER_NAMES[providerId]}
          </Text>
        );
      })}
      <Text color={colors.muted} dimColor>
        (Shift+Tab to switch)
      </Text>
    </Box>
  );
}
