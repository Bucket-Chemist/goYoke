/**
 * TabBar component - horizontal tab navigation with Alt+key hotkeys
 * Uses useKeymap for keyboard bindings and Zustand for active tab state
 */

import React from "react";
import { Box, Text } from "ink";
import { colors } from "../config/theme.js";
import { useStore } from "../store/index.js";
import { useKeymap, type KeyBinding } from "../hooks/useKeymap.js";
import type { TabDefinition, TabId } from "../store/types.js";

/**
 * Tab configuration with hotkey definitions
 */
const TAB_CONFIG: TabDefinition[] = [
  { id: "chat", label: "Chat", shortcutKey: "c", shortcutIndex: 0 },
  { id: "agent-config", label: "Agent Config", shortcutKey: "a", shortcutIndex: 0 },
  { id: "team-config", label: "Team Config", shortcutKey: "t", shortcutIndex: 0 },
  { id: "telemetry", label: "Telemetry", shortcutKey: "y", shortcutIndex: 9 },
];

interface TabBarProps {
  enabled?: boolean;
}

/**
 * Renders a tab label with underlined shortcut character
 */
function TabLabel({
  label,
  shortcutIndex,
}: {
  label: string;
  shortcutIndex: number;
}): JSX.Element {
  const before = label.slice(0, shortcutIndex);
  const shortcut = label[shortcutIndex];
  const after = label.slice(shortcutIndex + 1);

  return (
    <>
      {before}
      <Text underline>{shortcut}</Text>
      {after}
    </>
  );
}

/**
 * TabBar component - displays tabs horizontally with Alt+key shortcuts
 *
 * @param enabled - Whether keyboard shortcuts are active (default: true)
 *
 * @example
 * ```tsx
 * <TabBar enabled={!streaming} />
 * ```
 */
export function TabBar({ enabled = true }: TabBarProps): JSX.Element {
  const activeTab = useStore((state) => state.activeTab);
  const setActiveTab = useStore((state) => state.setActiveTab);

  // Build keyboard bindings for Alt+key shortcuts
  const bindings: KeyBinding[] = TAB_CONFIG.map((tab) => ({
    key: tab.shortcutKey,
    meta: true,
    action: () => setActiveTab(tab.id),
    description: `Switch to ${tab.label} tab`,
  }));

  useKeymap(bindings, enabled);

  return (
    <Box gap={2}>
      {TAB_CONFIG.map((tab) => {
        const isActive = activeTab === tab.id;

        return (
          <Text
            key={tab.id}
            color={isActive ? colors.primary : colors.muted}
            bold={isActive}
          >
            <TabLabel label={tab.label} shortcutIndex={tab.shortcutIndex} />
          </Text>
        );
      })}
    </Box>
  );
}
