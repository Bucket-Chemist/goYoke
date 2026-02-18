/**
 * TaskBoard - compact full-width strip showing Claude tasks and team progress.
 * Renders above StatusLine when activeTab === 'chat'.
 * Max 8 rows (including border supplied by Layout).
 */

import React from "react";
import { Box, Text } from "ink";
import { useStore } from "../store/index.js";
import { colors } from "../config/theme.js";
import type { TeamSummary } from "../store/types.js";

interface TodoItem {
  content: string;
  status: string;
  activeForm?: string;
}

function useTaskBoardData(): { todos: TodoItem[]; teams: TeamSummary[]; sessionId: string | null; ownerLabel: string } {
  const messages = useStore((s) => s.messages);
  const teams = useStore((s) => s.teams);
  const sessionId = useStore((s) => s.sessionId);

  // Find last TodoWrite block across all messages (last write wins)
  let lastTodos: TodoItem[] = [];
  for (const msg of messages) {
    for (const block of msg.content) {
      if (block.type === "tool_use" && block.name === "TodoWrite") {
        const todos = (block.input as Record<string, unknown>)["todos"];
        if (Array.isArray(todos)) {
          lastTodos = todos as TodoItem[];
        }
      }
    }
  }

  // Determine owner by scanning messages in reverse for the most recent Task()
  // invocation. The prompt field contains "AGENT: <agent-id>" as its first line.
  // Falls back to "claude" when tasks come from the root session directly.
  let ownerLabel = "claude";
  outer: for (let i = messages.length - 1; i >= 0; i--) {
    const msg = messages[i];
    if (!msg) continue;
    for (const block of msg.content) {
      if (block.type === "tool_use" && block.name === "Task") {
        const prompt = (block.input as Record<string, unknown>)["prompt"];
        if (typeof prompt === "string") {
          const match = prompt.match(/^AGENT:\s*(\S+)/m);
          if (match?.[1]) {
            ownerLabel = match[1];
            break outer;
          }
        }
        // Fallback: use the description field if no AGENT: line
        const desc = (block.input as Record<string, unknown>)["description"];
        if (typeof desc === "string" && desc.trim()) {
          ownerLabel = desc.split(" ")[0]?.toLowerCase() ?? "claude";
          break outer;
        }
      }
    }
  }

  return { todos: lastTodos, teams, sessionId, ownerLabel };
}

function todoIcon(status: string): string {
  if (status === "completed") return "✓";
  if (status === "in_progress") return "▶";
  return "○";
}

function todoColor(status: string): string {
  if (status === "completed") return colors.success;
  if (status === "in_progress") return colors.warning;
  return colors.muted;
}

function memberIcon(status: string, alive: boolean): string {
  if (status === "completed") return "✓";
  if (status === "failed") return "✗";
  if (alive && (status === "running" || status === "pending")) return "▶";
  return "○";
}

function memberColor(status: string): string {
  if (status === "completed") return colors.success;
  if (status === "failed") return colors.error;
  if (status === "running") return colors.warning;
  return colors.muted;
}

function truncate(str: string, maxLen: number): string {
  return str.length > maxLen ? str.slice(0, maxLen - 1) + "…" : str;
}

export function TaskBoard({ width, tab = "active" }: { width?: number; tab?: "active" | "done" }): JSX.Element {
  const { todos, teams, sessionId, ownerLabel } = useTaskBoardData();

  const activeTodos = todos.filter((t) => t.status !== "completed");
  const doneTodos = todos.filter((t) => t.status === "completed");
  const activeTeams = teams.filter((t) => t.alive || t.status === "running");
  const displayTodos = tab === "active" ? activeTodos : doneTodos;
  const maxW = (width ?? 80) - 4;

  const hasAnything = todos.length > 0 || teams.length > 0;
  const sessionHash = sessionId ? sessionId.slice(0, 8) : "local";

  return (
    <Box flexDirection="column" width={width} overflow="hidden">
      {/* Header row with tabs */}
      <Box flexDirection="row" paddingX={1}>
        <Text bold color={colors.primary}>Tasks </Text>
        <Text
          color={tab === "active" ? colors.focused : colors.muted}
          bold={tab === "active"}
          underline={tab === "active"}
        >
          [Active]
        </Text>
        <Text> </Text>
        <Text
          color={tab === "done" ? colors.focused : colors.muted}
          bold={tab === "done"}
          underline={tab === "done"}
        >
          [Done]
        </Text>
        {activeTodos.length > 0 && (
          <Text color={colors.muted} dimColor>  ({activeTodos.length} active)</Text>
        )}
      </Box>

      {!hasAnything && (
        <Box paddingX={2}>
          <Text color={colors.muted} dimColor>No active tasks</Text>
        </Box>
      )}

      {/* Claude session tasks — tree: Session owner header + indented tasks */}
      {displayTodos.length > 0 && (
        <>
          {/* Owner row */}
          <Box paddingX={1}>
            <Text color={colors.muted}>├─ </Text>
            <Text color={colors.focused}>{ownerLabel}</Text>
            <Text color={colors.muted} dimColor> ({sessionHash})</Text>
          </Box>
          {displayTodos.slice(0, 4).map((todo, idx) => {
            const isLast = idx === Math.min(displayTodos.length, 4) - 1;
            return (
              <Box key={`todo-${idx}`} paddingX={1} overflow="hidden">
                <Text color={colors.muted}>{isLast ? "   └─ " : "   ├─ "}</Text>
                <Text color={todoColor(todo.status)}>{todoIcon(todo.status)} </Text>
                <Text
                  color={todo.status === "completed" ? colors.muted : colors.assistantMessage}
                  dimColor={todo.status === "completed"}
                >
                  {truncate(todo.content, maxW - 14)}
                </Text>
                {todo.status === "in_progress" && todo.activeForm && (
                  <Text color={colors.warning} dimColor> ({truncate(todo.activeForm, 18)})</Text>
                )}
              </Box>
            );
          })}
          {displayTodos.length > 4 && (
            <Box paddingX={1}>
              <Text color={colors.muted} dimColor>   └─ … +{displayTodos.length - 4} more (Alt+B for Done tab)</Text>
            </Box>
          )}
        </>
      )}

      {/* Active teams — tree: team owner header + member counts */}
      {tab === "active" && activeTeams.slice(0, 2).map((team, tIdx) => (
        <Box key={team.dir} flexDirection="column" overflow="hidden">
          <Box paddingX={1}>
            <Text color={colors.muted}>{tIdx === 0 && displayTodos.length === 0 ? "├─ " : "├─ "}</Text>
            <Text color={colors.accent}>Team: </Text>
            <Text color={colors.assistantMessage}>{truncate(team.name, 28)}</Text>
          </Box>
          <Box paddingX={1}>
            <Text color={colors.muted}>   └─ </Text>
            <Text color={team.status === "running" ? colors.warning : colors.muted}>
              {memberIcon(team.status, team.alive)}{" "}
            </Text>
            <Text color={colors.muted} dimColor>
              {team.completedMembers}/{team.memberCount} members
              {team.failedMembers > 0 ? ` · ${team.failedMembers}✗` : ""}
              {" · "}{team.status}
            </Text>
          </Box>
        </Box>
      ))}
    </Box>
  );
}
