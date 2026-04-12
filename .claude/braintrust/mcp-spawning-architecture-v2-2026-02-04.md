# MCP-Based Agent Spawning Architecture v2

**Date**: 2026-02-04
**Status**: REVISED - CLI-only approach
**Supersedes**: mcp-spawning-discovery-2026-02-04.md
**Impact**: Enables full orchestrator pattern with process oversight

---

## Executive Summary

We discovered that subagents can call MCP tools but cannot use the Task tool. Initial proposal suggested using SDK `query()` for spawning, but this was **flawed**:

| Approach | Tools | API Key | Project Context |
|----------|-------|---------|-----------------|
| SDK query() | ❌ None | ⚠️ Required | ❌ None |
| CLI spawning | ✅ Full | ✅ Uses Claude Code | ✅ Full |

**Revised approach**: All agent spawning via `claude` CLI, orchestrated through MCP tools in the TUI. This provides full tool access, no API key management, and integrates with existing TUI infrastructure for visualization and oversight.

---

## Part 1: Theoretical Framework

### 1.1 The Spawning Problem

Claude Code's architecture creates a capability asymmetry:

```
Level 0 (Router)     → Has Task tool     → Can spawn Level 1
Level 1 (Subagent)   → NO Task tool      → Cannot spawn Level 2
Level 2 (Would-be)   → N/A               → Cannot exist via Task
```

This breaks orchestrator patterns where Mozart needs to spawn Einstein, or review-orchestrator needs to spawn reviewers.

### 1.2 The MCP Bridge

MCP tools exist outside Claude Code's tool restriction system:

```
Claude Code Tool Restrictions    MCP Tool Access
┌─────────────────────────┐     ┌─────────────────────────┐
│ Router: Full tools      │     │ Router: MCP available   │
│ Subagent: Limited tools │     │ Subagent: MCP available │
│ Sub-sub: Cannot exist   │     │ Sub-sub: MCP available* │
└─────────────────────────┘     └─────────────────────────┘
                                 * If spawned via CLI
```

By implementing a `spawn_agent` MCP tool that invokes the `claude` CLI, we create a spawning pathway that:
1. Is available to subagents (MCP tools are accessible)
2. Produces full-capability agents (CLI gives full tools)
3. Requires no API key (uses Claude Code's auth)
4. Integrates with existing infrastructure (TUI, hooks, telemetry)

### 1.3 Process Hierarchy Model

We adopt an **Epic → Parent → Child** model borrowed from project management:

```
Epic: The top-level workflow (e.g., /braintrust, /review)
  │
  Parent: Orchestrator agent spawned by router
  │   │
  │   Child: Specialist agents spawned by parent via MCP
  │   │   │
  │   │   Grandchild: Agents spawned by children (if needed)
```

Each agent in the hierarchy has:
- `epicId`: The workflow that initiated everything
- `parentId`: Direct parent agent (null for router-spawned)
- `spawnMethod`: "task" (router) or "mcp-cli" (subagent)
- `depth`: Nesting level (0 = router, 1 = parent, 2 = child, etc.)

---

## Part 2: Integration with Current TUI Architecture

### 2.1 Current TUI State (from TUI-TS-INK-CONVERSION-PLAN.md)

The TUI already has infrastructure for agent tracking:

```typescript
// Current agent interface (src/store/slices/agents.ts)
interface Agent {
  id: string;
  parentId: string | null;
  model: string;
  tier: "haiku" | "sonnet" | "opus";
  status: "spawning" | "running" | "complete" | "error";
  description?: string;
  startTime: number;
  endTime?: number;
  tokenUsage?: { input: number; output: number };
}
```

**What exists:**
- `<AgentTree />` component for hierarchical visualization
- `<AgentDetail />` for selected agent info
- Zustand store with agents slice
- Telemetry integration watching Go hook output

**What's missing:**
- MCP spawn_agent tool
- Epic tracking
- Spawn method tracking
- CLI process management
- Real-time output streaming
- Deeper nesting support

### 2.2 Enhanced Agent Interface

```typescript
// Enhanced agent interface (src/store/types.ts)
interface Agent {
  // Identity
  id: string;
  agentType: string;              // e.g., "einstein", "backend-reviewer"

  // Hierarchy
  epicId: string;                 // Top-level workflow ID
  parentId: string | null;        // Direct parent
  depth: number;                  // Nesting level
  childIds: string[];             // Children spawned by this agent

  // Spawning
  spawnMethod: "task" | "mcp-cli";
  spawnedBy: string;              // Agent ID that spawned this
  prompt: string;                 // Original prompt (for debugging)

  // Execution
  model: "haiku" | "sonnet" | "opus";
  status: "queued" | "spawning" | "running" | "streaming" | "complete" | "error" | "timeout";
  pid?: number;                   // CLI process ID (for mcp-cli spawns)

  // Timing
  queuedAt: number;
  startTime?: number;
  endTime?: number;

  // Output
  output?: string;                // Final output text
  streamBuffer?: string;          // Streaming output buffer
  error?: string;                 // Error message if failed

  // Metrics
  tokenUsage?: { input: number; output: number };
  cost?: number;
  turns?: number;
  toolCalls?: number;
}

interface Epic {
  id: string;
  type: string;                   // "braintrust", "review", "ticket", etc.
  rootAgentId: string;            // First agent in the epic
  agentIds: string[];             // All agents in this epic
  status: "running" | "complete" | "error";
  startTime: number;
  endTime?: number;
  totalCost?: number;
}
```

### 2.3 Enhanced Store Structure

```typescript
// src/store/slices/agents.ts
interface AgentsSlice {
  // Agents
  agents: Map<string, Agent>;
  agentsByEpic: Map<string, Set<string>>;  // epicId → agent IDs

  // Epics
  epics: Map<string, Epic>;
  activeEpicId: string | null;

  // Selection
  selectedAgentId: string | null;
  expandedAgentIds: Set<string>;           // For tree view

  // Actions
  createEpic: (type: string) => string;
  addAgent: (agent: Omit<Agent, 'id' | 'queuedAt'>) => string;
  updateAgent: (id: string, updates: Partial<Agent>) => void;
  appendOutput: (id: string, chunk: string) => void;
  completeAgent: (id: string, result: AgentResult) => void;
  failAgent: (id: string, error: string) => void;

  // Queries
  getAgentTree: (rootId: string) => AgentTreeNode;
  getEpicAgents: (epicId: string) => Agent[];
  getAgentChildren: (agentId: string) => Agent[];
  getAgentAncestors: (agentId: string) => Agent[];
}
```

---

## Part 3: MCP spawn_agent Implementation

### 3.1 Tool Definition

```typescript
// src/mcp/tools/spawnAgent.ts
import { tool } from "@anthropic-ai/claude-agent-sdk";
import { z } from "zod";
import { spawn, ChildProcess } from "child_process";
import * as fs from "fs/promises";
import * as path from "path";
import { useAgentsStore } from "../store/slices/agents";
import { useEpicsStore } from "../store/slices/epics";

// Process registry for management
const activeProcesses = new Map<string, ChildProcess>();

export const spawnAgentTool = tool(
  "spawn_agent",
  `Spawn a Claude Code subagent with full tool access.

  The spawned agent runs as a real Claude Code CLI process with:
  - Full tool access (Read, Write, Bash, Glob, Grep, Edit, etc.)
  - Project context (CLAUDE.md, working directory, git state)
  - Hook integration (gogent-validate, gogent-sharp-edge, etc.)

  Use this to delegate work from orchestrator agents to specialists.`,
  {
    agent: z.string().describe("Agent ID from agents-index.json (e.g., 'einstein', 'backend-reviewer')"),
    description: z.string().describe("Brief description for logging and display"),
    prompt: z.string().describe("Full prompt including AGENT: header and task details"),
    model: z.enum(["haiku", "sonnet", "opus"]).optional().describe("Model override (defaults to agent definition)"),
    epicId: z.string().optional().describe("Epic ID for grouping related agents"),
    parentId: z.string().optional().describe("Parent agent ID (auto-detected if possible)"),
    maxTurns: z.number().default(30).describe("Maximum conversation turns"),
    timeout: z.number().default(600000).describe("Timeout in milliseconds"),
    allowedTools: z.array(z.string()).optional().describe("Restrict available tools"),
    stream: z.boolean().default(true).describe("Stream output in real-time"),
  },
  async (args, context) => {
    const store = useAgentsStore.getState();

    // Determine epic (create if not provided)
    const epicId = args.epicId || store.activeEpicId || store.createEpic("ad-hoc");

    // Determine parent (from context if available)
    const parentId = args.parentId || context?.agentId || null;
    const parentAgent = parentId ? store.agents.get(parentId) : null;
    const depth = parentAgent ? parentAgent.depth + 1 : 1;

    // Create agent record
    const agentId = store.addAgent({
      agentType: args.agent,
      epicId,
      parentId,
      depth,
      childIds: [],
      spawnMethod: "mcp-cli",
      spawnedBy: parentId || "mcp-tool",
      prompt: args.prompt,
      model: args.model || "sonnet",
      status: "queued",
    });

    // Update parent's children list
    if (parentId) {
      store.updateAgent(parentId, {
        childIds: [...(parentAgent?.childIds || []), agentId],
      });
    }

    try {
      // Spawn CLI process
      const result = await spawnCliAgent(agentId, args, store);

      return {
        content: [{
          type: "text",
          text: JSON.stringify({
            agentId,
            agent: args.agent,
            epicId,
            success: true,
            output: result.output,
            cost: result.cost,
            turns: result.turns,
            toolCalls: result.toolCalls,
          })
        }]
      };
    } catch (error) {
      store.failAgent(agentId, error.message);

      return {
        content: [{
          type: "text",
          text: JSON.stringify({
            agentId,
            agent: args.agent,
            epicId,
            success: false,
            error: error.message,
          })
        }]
      };
    }
  }
);

async function spawnCliAgent(
  agentId: string,
  args: SpawnArgs,
  store: AgentsStore
): Promise<AgentResult> {
  // Write prompt to temp file (handles complex prompts with quotes/newlines)
  const promptFile = path.join('/tmp', `claude-spawn-${agentId}.txt`);
  await fs.writeFile(promptFile, args.prompt, 'utf-8');

  // Build CLI arguments
  const cliArgs = buildCliArgs(args, promptFile);

  return new Promise((resolve, reject) => {
    store.updateAgent(agentId, { status: "spawning" });

    const proc = spawn('claude', cliArgs, {
      cwd: process.cwd(),
      shell: true,
      stdio: ['pipe', 'pipe', 'pipe'],
      env: {
        ...process.env,
        GOGENT_PARENT_AGENT: agentId,
        GOGENT_EPIC_ID: args.epicId,
      },
    });

    // Register process for management
    activeProcesses.set(agentId, proc);
    store.updateAgent(agentId, {
      status: "running",
      startTime: Date.now(),
      pid: proc.pid,
    });

    let stdout = '';
    let stderr = '';

    // Handle streaming output
    proc.stdout.on('data', (data) => {
      const chunk = data.toString();
      stdout += chunk;

      if (args.stream) {
        store.appendOutput(agentId, chunk);
        store.updateAgent(agentId, { status: "streaming" });
      }
    });

    proc.stderr.on('data', (data) => {
      stderr += data.toString();
    });

    // Timeout handling
    const timer = setTimeout(() => {
      proc.kill('SIGTERM');
      activeProcesses.delete(agentId);
      store.updateAgent(agentId, { status: "timeout" });
      reject(new Error(`Agent ${args.agent} timed out after ${args.timeout}ms`));
    }, args.timeout);

    proc.on('close', async (code) => {
      clearTimeout(timer);
      activeProcesses.delete(agentId);

      // Clean up prompt file
      await fs.unlink(promptFile).catch(() => {});

      if (code === 0) {
        const result = parseCliOutput(stdout);
        store.completeAgent(agentId, result);
        resolve(result);
      } else {
        const error = `Exit code ${code}: ${stderr || 'Unknown error'}`;
        store.failAgent(agentId, error);
        reject(new Error(error));
      }
    });

    proc.on('error', (err) => {
      clearTimeout(timer);
      activeProcesses.delete(agentId);
      store.failAgent(agentId, err.message);
      reject(err);
    });
  });
}

function buildCliArgs(args: SpawnArgs, promptFile: string): string[] {
  const cliArgs = [
    '-p', `"$(cat ${promptFile})"`,
    '--output-format', 'stream-json',
    '--dangerously-skip-permissions',
    '--max-turns', String(args.maxTurns),
  ];

  if (args.model) {
    cliArgs.push('--model', args.model);
  }

  if (args.allowedTools?.length) {
    cliArgs.push('--allowedTools', `"${args.allowedTools.join(',')}"`);
  }

  return cliArgs;
}

function parseCliOutput(stdout: string): AgentResult {
  // Parse NDJSON stream output
  const lines = stdout.trim().split('\n').filter(Boolean);
  let output = '';
  let cost = 0;
  let turns = 0;
  let toolCalls = 0;

  for (const line of lines) {
    try {
      const event = JSON.parse(line);

      if (event.type === 'assistant' && event.message?.content) {
        for (const block of event.message.content) {
          if (block.type === 'text') {
            output += block.text;
          } else if (block.type === 'tool_use') {
            toolCalls++;
          }
        }
      }

      if (event.type === 'result') {
        cost = event.cost_usd || 0;
        turns = event.num_turns || 0;
      }
    } catch {
      // Non-JSON line, append as output
      output += line + '\n';
    }
  }

  return { output, cost, turns, toolCalls };
}

// Process management functions
export function killAgent(agentId: string): boolean {
  const proc = activeProcesses.get(agentId);
  if (proc) {
    proc.kill('SIGTERM');
    activeProcesses.delete(agentId);
    return true;
  }
  return false;
}

export function getActiveAgents(): string[] {
  return Array.from(activeProcesses.keys());
}
```

### 3.2 Parallel Spawning Tool

For orchestrators that need to spawn multiple agents simultaneously:

```typescript
// src/mcp/tools/spawnAgentsParallel.ts
export const spawnAgentsParallelTool = tool(
  "spawn_agents_parallel",
  "Spawn multiple agents in parallel and wait for all to complete.",
  {
    agents: z.array(z.object({
      agent: z.string(),
      description: z.string(),
      prompt: z.string(),
      model: z.enum(["haiku", "sonnet", "opus"]).optional(),
    })),
    epicId: z.string().optional(),
    parentId: z.string().optional(),
    failFast: z.boolean().default(false).describe("Stop all if one fails"),
  },
  async (args, context) => {
    const store = useAgentsStore.getState();
    const epicId = args.epicId || store.activeEpicId || store.createEpic("parallel-spawn");

    // Spawn all agents concurrently
    const promises = args.agents.map(agentArgs =>
      spawnAgentTool.handler({
        ...agentArgs,
        epicId,
        parentId: args.parentId,
        stream: true,
      }, context)
    );

    if (args.failFast) {
      // Fail immediately if any agent fails
      const results = await Promise.all(promises);
      return { content: [{ type: "text", text: JSON.stringify(results) }] };
    } else {
      // Wait for all, collect successes and failures
      const results = await Promise.allSettled(promises);
      const summary = results.map((r, i) => ({
        agent: args.agents[i].agent,
        status: r.status,
        result: r.status === 'fulfilled' ? r.value : null,
        error: r.status === 'rejected' ? r.reason.message : null,
      }));
      return { content: [{ type: "text", text: JSON.stringify(summary) }] };
    }
  }
);
```

---

## Part 4: Visualization Integration

**Agent relationship schemas for validation and visualization:**
- `.claude/schemas/agent-relationships-schema.json` - Formal field definitions
- `.claude/schemas/agent-relationships-examples.md` - Per-agent examples

### 4.1 Enhanced AgentTree Component

```typescript
// src/components/AgentTree.tsx
import React from 'react';
import { Box, Text } from 'ink';
import { useAgentsStore } from '../store/slices/agents';

interface AgentTreeNode {
  agent: Agent;
  children: AgentTreeNode[];
}

function buildTree(agents: Map<string, Agent>, rootId: string | null): AgentTreeNode[] {
  const children = Array.from(agents.values())
    .filter(a => a.parentId === rootId)
    .sort((a, b) => a.queuedAt - b.queuedAt);

  return children.map(agent => ({
    agent,
    children: buildTree(agents, agent.id),
  }));
}

export function AgentTree({ focused }: { focused: boolean }) {
  const { agents, selectedAgentId, selectAgent, expandedAgentIds, toggleExpanded } = useAgentsStore();
  const tree = buildTree(agents, null);

  return (
    <Box flexDirection="column" borderStyle="single" borderColor={focused ? "cyan" : "gray"}>
      <Box paddingX={1}>
        <Text bold>Agent Hierarchy</Text>
      </Box>
      <Box flexDirection="column" paddingX={1}>
        {tree.map(node => (
          <TreeNode
            key={node.agent.id}
            node={node}
            depth={0}
            selectedId={selectedAgentId}
            expandedIds={expandedAgentIds}
            onSelect={selectAgent}
            onToggle={toggleExpanded}
          />
        ))}
      </Box>
    </Box>
  );
}

function TreeNode({ node, depth, selectedId, expandedIds, onSelect, onToggle }: TreeNodeProps) {
  const { agent, children } = node;
  const isSelected = agent.id === selectedId;
  const isExpanded = expandedIds.has(agent.id);
  const hasChildren = children.length > 0;

  const statusIcon = getStatusIcon(agent.status);
  const spawnIcon = agent.spawnMethod === 'mcp-cli' ? '⚡' : '→';
  const indent = '  '.repeat(depth);

  return (
    <>
      <Box>
        <Text color={isSelected ? "cyan" : undefined}>
          {indent}
          {hasChildren ? (isExpanded ? '▼ ' : '▶ ') : '  '}
          {statusIcon} {spawnIcon} {agent.agentType}
          <Text dimColor> ({agent.model})</Text>
          {agent.cost && <Text color="yellow"> ${agent.cost.toFixed(4)}</Text>}
        </Text>
      </Box>
      {isExpanded && children.map(child => (
        <TreeNode
          key={child.agent.id}
          node={child}
          depth={depth + 1}
          selectedId={selectedId}
          expandedIds={expandedIds}
          onSelect={onSelect}
          onToggle={onToggle}
        />
      ))}
    </>
  );
}

function getStatusIcon(status: Agent['status']): string {
  switch (status) {
    case 'queued': return '⏳';
    case 'spawning': return '🔄';
    case 'running': return '🏃';
    case 'streaming': return '📡';
    case 'complete': return '✅';
    case 'error': return '❌';
    case 'timeout': return '⏰';
    default: return '?';
  }
}
```

### 4.2 Enhanced AgentDetail with Relationship Validation

```typescript
// src/components/AgentDetail.tsx
import React from 'react';
import { Box, Text, Newline } from 'ink';
import { useAgentsStore } from '../store/slices/agents';
import { useAgentConfig } from '../hooks/useAgentConfig';

export function AgentDetail() {
  const { agents, selectedAgentId } = useAgentsStore();
  const agent = selectedAgentId ? agents.get(selectedAgentId) : null;
  const agentConfig = useAgentConfig(agent?.agentType);

  if (!agent) {
    return (
      <Box borderStyle="single" borderColor="gray" padding={1}>
        <Text dimColor>Select an agent to view details</Text>
      </Box>
    );
  }

  const duration = agent.endTime
    ? ((agent.endTime - (agent.startTime || 0)) / 1000).toFixed(1)
    : agent.startTime
      ? ((Date.now() - agent.startTime) / 1000).toFixed(1)
      : '-';

  return (
    <Box flexDirection="column" borderStyle="single" borderColor="gray" padding={1}>
      <Box>
        <Text bold color="cyan">{agent.agentType}</Text>
        <Text dimColor> • {agent.model} • {agent.status}</Text>
      </Box>

      <Newline />

      {/* Hierarchy */}
      <Box flexDirection="column">
        <Text dimColor>Hierarchy:</Text>
        <Text>  Epic: {agent.epicId.slice(0, 8)}</Text>
        <Text>  Parent: {agent.parentId?.slice(0, 8) || 'router'}</Text>
        <Text>  Depth: {agent.depth}</Text>
        <Text>  Children: {agent.childIds.length}</Text>
        <Text>  Spawn: {agent.spawnMethod}</Text>
      </Box>

      <Newline />

      {/* Relationships from agents-index.json */}
      {agentConfig && (
        <Box flexDirection="column">
          <Text dimColor>Relationships (schema):</Text>
          {agentConfig.spawned_by && (
            <Text>  Spawned by: {agentConfig.spawned_by.join(', ')}</Text>
          )}
          {agentConfig.can_spawn && (
            <Text>  Can spawn: {agentConfig.can_spawn.join(', ') || 'none'}</Text>
          )}
          {agentConfig.outputs_to && (
            <Text>  Outputs to: {agentConfig.outputs_to.join(', ')}</Text>
          )}
        </Box>
      )}

      <Newline />

      {/* Validation */}
      <Box flexDirection="column">
        <Text dimColor>Validation:</Text>
        {renderValidation(agent, agentConfig)}
      </Box>

      <Newline />

      {/* Metrics */}
      <Box flexDirection="column">
        <Text dimColor>Metrics:</Text>
        <Text>  Duration: {duration}s</Text>
        <Text>  Turns: {agent.turns || '-'}</Text>
        <Text>  Tool calls: {agent.toolCalls || '-'}</Text>
        <Text>  Cost: ${agent.cost?.toFixed(4) || '-'}</Text>
        {agent.pid && <Text>  PID: {agent.pid}</Text>}
      </Box>

      {agent.error && (
        <>
          <Newline />
          <Text color="red">Error: {agent.error}</Text>
        </>
      )}
    </Box>
  );
}

function renderValidation(agent: Agent, config: AgentConfig | null) {
  const checks: JSX.Element[] = [];

  // Check spawned_by
  if (config?.spawned_by && agent.parentId) {
    const parentType = getAgentType(agent.parentId);
    const allowed = config.spawned_by.includes(parentType) || config.spawned_by.includes('any');
    checks.push(
      <Text key="parent" color={allowed ? "green" : "yellow"}>
        {allowed ? "  ✓" : "  ⚠️"} Parent: {parentType}
      </Text>
    );
  }

  // Check must_delegate
  if (config?.must_delegate) {
    const met = agent.childIds.length >= (config.min_delegations || 1);
    checks.push(
      <Text key="delegate" color={met ? "green" : "red"}>
        {met ? "  ✓" : "  ❌"} Delegation: {agent.childIds.length}/{config.min_delegations}
      </Text>
    );
  }

  return checks.length > 0 ? checks : <Text dimColor>  No constraints</Text>;
}
```

### 4.3 Epic Overview Component

```typescript
// src/components/EpicOverview.tsx
export function EpicOverview() {
  const { epics, agents, activeEpicId } = useAgentsStore();
  const activeEpic = activeEpicId ? epics.get(activeEpicId) : null;

  if (!activeEpic) return null;

  const epicAgents = activeEpic.agentIds.map(id => agents.get(id)).filter(Boolean);
  const running = epicAgents.filter(a => ['running', 'streaming'].includes(a.status)).length;
  const complete = epicAgents.filter(a => a.status === 'complete').length;
  const failed = epicAgents.filter(a => ['error', 'timeout'].includes(a.status)).length;
  const totalCost = epicAgents.reduce((sum, a) => sum + (a.cost || 0), 0);

  return (
    <Box borderStyle="round" borderColor="magenta" paddingX={1}>
      <Text bold color="magenta">Epic: {activeEpic.type}</Text>
      <Text> • </Text>
      <Text color="blue">{running} running</Text>
      <Text> • </Text>
      <Text color="green">{complete} done</Text>
      {failed > 0 && <Text color="red"> • {failed} failed</Text>}
      <Text> • </Text>
      <Text color="yellow">${totalCost.toFixed(4)}</Text>
    </Box>
  );
}
```

---

## Part 5: Hook Integration

### 5.1 Environment Variables for Spawned Agents

The MCP spawn_agent passes environment variables to track hierarchy:

```bash
GOGENT_PARENT_AGENT=<parent-agent-id>
GOGENT_EPIC_ID=<epic-id>
GOGENT_SPAWN_METHOD=mcp-cli
GOGENT_DEPTH=<nesting-depth>
```

### 5.2 Enhanced gogent-validate Hook

Update to recognize MCP-spawned agents:

```go
// cmd/gogent-validate/main.go additions

func main() {
    // ... existing code ...

    // Check if this is an MCP-spawned agent
    parentAgent := os.Getenv("GOGENT_PARENT_AGENT")
    epicID := os.Getenv("GOGENT_EPIC_ID")
    spawnMethod := os.Getenv("GOGENT_SPAWN_METHOD")

    // Log with hierarchy context
    if parentAgent != "" {
        lifecycle := telemetry.NewAgentLifecycleEvent(
            event.SessionID,
            "spawn",
            extractAgentFromPrompt(taskInput.Prompt),
            parentAgent,  // Now tracks actual parent, not just "terminal"
            taskInput.Model,
            taskInput.Prompt,
            decisionID,
        )
        lifecycle.EpicID = epicID
        lifecycle.SpawnMethod = spawnMethod
        lifecycle.Depth = getDepthFromEnv()

        telemetry.LogAgentLifecycle(lifecycle)
    }
}
```

### 5.3 Enhanced Telemetry Schema

```go
// pkg/telemetry/types.go additions

type AgentLifecycleEvent struct {
    // ... existing fields ...

    EpicID      string `json:"epic_id,omitempty"`
    SpawnMethod string `json:"spawn_method,omitempty"` // "task" or "mcp-cli"
    Depth       int    `json:"depth,omitempty"`
    ParentAgent string `json:"parent_agent,omitempty"`
}

type EpicEvent struct {
    Timestamp   time.Time `json:"timestamp"`
    EpicID      string    `json:"epic_id"`
    EpicType    string    `json:"epic_type"`  // "braintrust", "review", "ticket"
    Event       string    `json:"event"`      // "start", "agent_added", "complete", "error"
    AgentID     string    `json:"agent_id,omitempty"`
    TotalAgents int       `json:"total_agents,omitempty"`
    TotalCost   float64   `json:"total_cost,omitempty"`
}
```

### 5.4 Telemetry File Watching

The TUI watches telemetry files for real-time updates:

```typescript
// src/hooks/useTelemetry.ts
import { watch } from 'chokidar';
import { useAgentsStore } from '../store/slices/agents';

const LIFECYCLE_PATH = `${process.env.XDG_DATA_HOME}/gogent/agent-lifecycle.jsonl`;

export function useTelemetry() {
  const { updateAgent, addAgent } = useAgentsStore();

  useEffect(() => {
    const watcher = watch(LIFECYCLE_PATH, { persistent: true });

    watcher.on('change', async () => {
      const newEvents = await readNewLines(LIFECYCLE_PATH);

      for (const event of newEvents) {
        if (event.event === 'spawn' && event.spawn_method === 'mcp-cli') {
          // MCP-spawned agent detected via telemetry
          // (backup if store update missed)
        }

        if (event.event === 'complete' || event.event === 'error') {
          updateAgent(event.agent_id, {
            status: event.event === 'complete' ? 'complete' : 'error',
            endTime: new Date(event.timestamp).getTime(),
            cost: event.cost,
          });
        }
      }
    });

    return () => watcher.close();
  }, []);
}
```

---

## Part 6: Workflow Examples

### 6.1 Braintrust via MCP Spawning

```
User: /braintrust "Analyze the orchestrator spawning problem"
           │
           ▼
Router spawns Mozart (via Task)
           │
           ▼
┌──────────────────────────────────────────────────────────────┐
│ Mozart (Level 1, spawnMethod: "task")                        │
│                                                              │
│ 1. Creates Epic: "braintrust-2026-02-04-abc123"              │
│                                                              │
│ 2. Gathers context:                                          │
│    - Reads relevant files                                    │
│    - Runs searches                                           │
│    - Compiles problem brief                                  │
│                                                              │
│ 3. Spawns Einstein via MCP:                                  │
│    mcp__gofortress__spawn_agent({                            │
│      agent: "einstein",                                      │
│      epicId: "braintrust-2026-02-04-abc123",                 │
│      prompt: "AGENT: einstein\n\n[problem brief]\n\n..."     │
│    })                                                        │
│              │                                               │
│              ▼                                               │
│    ┌─────────────────────────────────────────────────────┐   │
│    │ Einstein (Level 2, spawnMethod: "mcp-cli")          │   │
│    │ - Full tool access via CLI                          │   │
│    │ - Can read additional files if needed               │   │
│    │ - Returns theoretical analysis                      │   │
│    └─────────────────────────────────────────────────────┘   │
│                                                              │
│ 4. Spawns Staff-Architect via MCP (parallel with Einstein):  │
│    mcp__gofortress__spawn_agent({                            │
│      agent: "staff-architect-critical-review",               │
│      epicId: "braintrust-2026-02-04-abc123",                 │
│      prompt: "AGENT: staff-architect...\n\n[problem brief]"  │
│    })                                                        │
│              │                                               │
│              ▼                                               │
│    ┌─────────────────────────────────────────────────────┐   │
│    │ Staff-Architect (Level 2, spawnMethod: "mcp-cli")   │   │
│    │ - Full tool access via CLI                          │   │
│    │ - Can read files, check implementation              │   │
│    │ - Returns practical review                          │   │
│    └─────────────────────────────────────────────────────┘   │
│                                                              │
│ 5. Collects both analyses                                    │
│                                                              │
│ 6. Spawns Beethoven via MCP:                                 │
│    mcp__gofortress__spawn_agent({                            │
│      agent: "beethoven",                                     │
│      epicId: "braintrust-2026-02-04-abc123",                 │
│      prompt: "AGENT: beethoven\n\n[both analyses]\n\n..."    │
│    })                                                        │
│              │                                               │
│              ▼                                               │
│    ┌─────────────────────────────────────────────────────┐   │
│    │ Beethoven (Level 2, spawnMethod: "mcp-cli")         │   │
│    │ - Synthesizes analyses                              │   │
│    │ - Writes final document                             │   │
│    │ - Returns path to output                            │   │
│    └─────────────────────────────────────────────────────┘   │
│                                                              │
│ 7. Returns final synthesis to user                           │
└──────────────────────────────────────────────────────────────┘
```

### 6.2 Review-Orchestrator via MCP Spawning

```
User: /review
           │
           ▼
Router spawns review-orchestrator (via Task)
           │
           ▼
┌──────────────────────────────────────────────────────────────┐
│ review-orchestrator (Level 1, spawnMethod: "task")           │
│                                                              │
│ 1. Creates Epic: "review-2026-02-04-xyz789"                  │
│                                                              │
│ 2. Analyzes files to determine domains                       │
│                                                              │
│ 3. Spawns reviewers in PARALLEL via MCP:                     │
│    mcp__gofortress__spawn_agents_parallel({                  │
│      epicId: "review-2026-02-04-xyz789",                     │
│      agents: [                                               │
│        { agent: "backend-reviewer", prompt: "..." },         │
│        { agent: "frontend-reviewer", prompt: "..." },        │
│        { agent: "standards-reviewer", prompt: "..." },       │
│        { agent: "architect-reviewer", prompt: "..." },       │
│      ]                                                       │
│    })                                                        │
│              │                                               │
│              ▼ (parallel)                                    │
│    ┌──────────────┐ ┌──────────────┐ ┌──────────────┐       │
│    │ backend      │ │ frontend     │ │ standards    │ ...   │
│    │ reviewer     │ │ reviewer     │ │ reviewer     │       │
│    │ (mcp-cli)    │ │ (mcp-cli)    │ │ (mcp-cli)    │       │
│    └──────────────┘ └──────────────┘ └──────────────┘       │
│                                                              │
│ 4. Collects all findings                                     │
│                                                              │
│ 5. Deduplicates and prioritizes                              │
│                                                              │
│ 6. Writes unified report                                     │
│                                                              │
│ 7. Returns to user with APPROVE/WARNING/BLOCK                │
└──────────────────────────────────────────────────────────────┘
```

---

## Part 7: TUI Visualization in Action

### 7.1 During Braintrust Execution

```
┌─ Claude Panel ─────────────────────────┬─ Agent Hierarchy ─────────────────┐
│                                        │                                    │
│ [Mozart] Starting Braintrust analysis  │ Epic: braintrust (running)         │
│ for orchestrator spawning problem...   │                                    │
│                                        │ ▼ 🏃 → mozart (opus)               │
│ Gathering context from:                │   ├─ 📡 ⚡ einstein (opus)         │
│ - GAP document                         │   ├─ 📡 ⚡ staff-architect (opus)  │
│ - TUI plan                             │   └─ ⏳ ⚡ beethoven (queued)      │
│ - Agent definitions                    │                                    │
│                                        ├────────────────────────────────────┤
│ [Einstein streaming...]                │ Agent Detail                       │
│ The fundamental issue stems from...    │                                    │
│                                        │ einstein • opus • streaming        │
│ [Staff-Architect streaming...]         │                                    │
│ From a practical standpoint...         │ Hierarchy:                         │
│                                        │   Epic: braintrust                 │
│                                        │   Parent: mozart                   │
│                                        │   Depth: 2                         │
│                                        │   Spawn: mcp-cli                   │
│                                        │                                    │
│                                        │ Metrics:                           │
│                                        │   Duration: 12.3s                  │
│                                        │   Turns: 3                         │
│                                        │   Cost: $0.0234                    │
│                                        │   PID: 48291                       │
│                                        │                                    │
└────────────────────────────────────────┴────────────────────────────────────┘
│ Epic: braintrust • 2 running • 1 done • 1 queued • $0.0891                  │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 7.2 Tree Notation Legend

```
Status Icons:
  ⏳ queued     - Waiting to spawn
  🔄 spawning   - CLI process starting
  🏃 running    - Executing
  📡 streaming  - Producing output
  ✅ complete   - Finished successfully
  ❌ error      - Failed
  ⏰ timeout    - Exceeded time limit

Spawn Method Icons:
  →  task       - Spawned by router via Task tool
  ⚡ mcp-cli    - Spawned via MCP CLI spawning
```

---

## Part 8: Migration Plan

### Phase 1: Foundation (Part of TUI Phase 4)
- [ ] Implement spawn_agent MCP tool
- [ ] Implement spawn_agents_parallel MCP tool
- [ ] Add enhanced Agent interface to store
- [ ] Add Epic tracking to store

### Phase 2: Visualization
- [ ] Enhance AgentTree with spawn method icons
- [ ] Enhance AgentDetail with hierarchy info
- [ ] Add EpicOverview component
- [ ] Add streaming output display

### Phase 3: Hook Integration
- [ ] Add environment variables to spawned CLIs
- [ ] Update gogent-validate for hierarchy tracking
- [ ] Update telemetry schema
- [ ] Add telemetry file watching

### Phase 4: Workflow Updates
- [ ] Update Mozart to use MCP spawning
- [ ] Update review-orchestrator to use MCP spawning
- [ ] Update impl-manager to use MCP spawning
- [ ] Update /braintrust skill documentation
- [ ] Update /review skill documentation

### Phase 5: Testing
- [ ] Test single agent MCP spawn
- [ ] Test parallel spawning
- [ ] Test nested spawning (3+ levels)
- [ ] Test timeout handling
- [ ] Test error propagation
- [ ] Test kill functionality
- [ ] Performance benchmarking

---

## Part 9: Considerations & Risks

### 9.1 Security Considerations

| Risk | Mitigation |
|------|------------|
| `--dangerously-skip-permissions` bypasses all checks | Only use in trusted environments; consider sandboxing |
| Spawned agents have full file access | Use `--allowedTools` to restrict per agent type |
| Prompt injection via MCP args | Sanitize prompt content; use temp files |
| Resource exhaustion via spawning | Implement max concurrent agents; timeout enforcement |

### 9.2 Performance Considerations

| Factor | Impact | Mitigation |
|--------|--------|------------|
| CLI startup overhead | ~5-10s per spawn | Parallel spawning; batching |
| Process count | Memory usage | Max concurrent limit |
| Output buffering | Memory for large outputs | Streaming; max buffer size |
| Disk I/O for prompts | Minor latency | /tmp cleanup; tmpfs if available |

### 9.3 Limitations

1. **Requires TUI**: MCP spawning only works when TUI is the interface
2. **No raw CLI support**: Users running `claude` directly can't use this
3. **Platform dependency**: Assumes Unix-like CLI spawning (may need Windows adaptation)
4. **Debugging complexity**: Nested CLI processes harder to debug than Task spawning

---

## Part 10: Conclusion

### What We Achieved

1. **Identified the real constraint**: Task tool unavailable to subagents
2. **Found viable workaround**: MCP tools + CLI spawning
3. **Designed complete architecture**: From MCP tool to visualization
4. **Preserved full capabilities**: CLI spawning gives full tool access
5. **No API key requirement**: Uses Claude Code's existing auth
6. **Integrated with existing infrastructure**: TUI, hooks, telemetry

### The Key Insight

MCP tools bridge the capability gap because:
- They're available to ALL agents (including subagents)
- Their handlers run in YOUR process (the TUI)
- Your process can spawn real Claude Code CLIs
- CLIs have full tool access

This creates an indirect spawning pathway that bypasses Claude Code's Task tool restriction while maintaining full agent capabilities.

### Next Steps

1. Implement spawn_agent tool in TUI Phase 4
2. Test with simple orchestration scenarios
3. Migrate Braintrust/review/impl-manager to MCP spawning
4. Monitor for issues and iterate

---

## Metadata

```yaml
document_id: mcp-spawning-architecture-v2-2026-02-04
supersedes: mcp-spawning-discovery-2026-02-04.md
status: revised
approach: cli-only
validated:
  - MCP tools accessible from subagents: true
  - CLI spawning provides full tools: true (by design)
  - No API key required: true
implementation_phase: TUI Phase 4
blocks: ["braintrust", "review-orchestrator", "impl-manager"]
estimated_effort: "5-7 days within TUI development"
```
