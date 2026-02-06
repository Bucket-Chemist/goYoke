import React, { useState, useEffect, useRef } from "react";
import { Box, Text } from "ink";
import { useStore } from "../store/index.js";
import { colors } from "../config/theme.js";

/**
 * DashboardView component
 * Shows session statistics and agent counts
 */
export function DashboardView(): JSX.Element {
  const { activeModel, preferredModel, sessionId, totalCost, tokenCount, agents } = useStore();
  const startTime = useRef(Date.now());
  const [, setTick] = useState(0);

  useEffect(() => {
    const interval = setInterval(() => setTick(t => t + 1), 1000);
    return () => clearInterval(interval);
  }, []);

  const modelName = activeModel || preferredModel || "—";
  const elapsed = Math.floor((Date.now() - startTime.current) / 1000);
  const mins = Math.floor(elapsed / 60);
  const secs = elapsed % 60;

  const agentValues = Object.values(agents);
  const running = agentValues.filter(a => a.status === "running" || a.status === "streaming").length;
  const queued = agentValues.filter(a => a.status === "queued" || a.status === "spawning").length;
  const complete = agentValues.filter(a => a.status === "complete").length;
  const errors = agentValues.filter(a => a.status === "error").length;

  const formatTokens = (n: number): string => n >= 1000 ? `${(n / 1000).toFixed(1)}K` : String(n);

  return (
    <Box flexDirection="column" paddingX={1} paddingY={0}>
      <Text bold color={colors.primary}>Dashboard</Text>
      <Box marginTop={1} flexDirection="column">
        <Text><Text color={colors.muted}>Model:    </Text><Text bold>{modelName}</Text></Text>
        <Text><Text color={colors.muted}>Session:  </Text>{sessionId ? sessionId.slice(0, 8) : "—"}</Text>
        <Text><Text color={colors.muted}>Cost:     </Text><Text color={totalCost > 1 ? colors.warning : colors.success}>${totalCost.toFixed(2)}</Text></Text>
        <Text><Text color={colors.muted}>Tokens:   </Text>{formatTokens(tokenCount.input)} in / {formatTokens(tokenCount.output)} out</Text>
        <Text><Text color={colors.muted}>Duration: </Text>{mins}m {String(secs).padStart(2, "0")}s</Text>
      </Box>
      <Box marginTop={1} flexDirection="column">
        <Text bold color={colors.muted}>Agents</Text>
        <Text><Text color={colors.agentRunning}>  Running:  </Text>{running}</Text>
        <Text><Text color={colors.agentSpawning}>  Queued:   </Text>{queued}</Text>
        <Text><Text color={colors.agentComplete}>  Complete: </Text>{complete}</Text>
        <Text><Text color={colors.agentError}>  Errors:   </Text>{errors}</Text>
      </Box>
    </Box>
  );
}
