/**
 * PlanPreview - Right panel mode for showing plan .md content during ExitPlanMode approval
 *
 * Displayed in the right panel (30%) when ExitPlanMode fires.
 * Uses ScrollView with disableArrowKeys=true so the approval modal retains Up/Down for
 * option navigation. PageUp/PageDown, Ctrl+A/E, and mouse wheel still work for scrolling.
 */

import React, { useMemo } from "react";
import { Box, Text } from "ink";
import { basename } from "path";
import { useStore } from "../store/index.js";
import { ScrollView } from "./primitives/ScrollView.js";
import { renderMarkdown } from "../utils/markdown.js";
import { colors } from "../config/theme.js";

export interface PlanPreviewProps {
  /** Available terminal rows for the scrollable content area */
  scrollHeight: number;
  /** Panel width in columns — used to size the separator line */
  width?: number;
}

export function PlanPreview({ scrollHeight, width }: PlanPreviewProps): JSX.Element {
  const { planPreviewContent, planPreviewPath } = useStore();

  const fileName = planPreviewPath ? basename(planPreviewPath) : "plan.md";

  const renderedContent = useMemo(
    () => (planPreviewContent ? renderMarkdown(planPreviewContent) : ""),
    [planPreviewContent]
  );

  return (
    <Box flexDirection="column" flexGrow={1}>
      {/* Header */}
      <Box paddingX={1}>
        <Text color={colors.primary} bold>Plan </Text>
        <Text color="gray">{fileName}</Text>
      </Box>
      <Box paddingX={1}>
        <Text color="gray">{"─".repeat(Math.max(4, (width ?? 30) - 2))}</Text>
      </Box>

      {/* Scrollable plan content */}
      {planPreviewContent ? (
        <ScrollView
          height={Math.max(4, scrollHeight - 4)}
          autoScroll={false}
          focused={true}
          disableArrowKeys={true}
        >
          <Box paddingX={1}>
            <Text wrap="wrap">{renderedContent}</Text>
          </Box>
        </ScrollView>
      ) : (
        <Box flexGrow={1} justifyContent="center" alignItems="center">
          <Text color="gray">Loading plan...</Text>
        </Box>
      )}

      {/* Footer hints */}
      <Box paddingX={1}>
        <Text dimColor>PgUp/PgDn scroll  Ctrl+A/E top/bottom  wheel</Text>
      </Box>
    </Box>
  );
}
