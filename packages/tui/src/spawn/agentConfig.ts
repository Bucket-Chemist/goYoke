import * as fs from "fs";
import * as path from "path";

/**
 * Relationship fields from agents-index.json
 */
export interface AgentRelationships {
  id: string;
  spawned_by?: string[];
  can_spawn?: string[];
  must_delegate?: boolean;
  min_delegations?: number;
  max_delegations?: number;
  inputs?: string[];
  outputs?: string[];
  outputs_to?: string[];
  invoked_by?: string;
}

/**
 * Full agent config from agents-index.json
 */
export interface AgentConfig extends AgentRelationships {
  name: string;
  model: string;
  effortLevel?: "low" | "medium" | "high";
  tier: number | string;
  triggers?: string[];
  tools?: string[];
  description?: string;
}

interface AgentsIndex {
  version: string;
  agents: AgentConfig[];
}

// Cache for agents-index.json
let cachedIndex: AgentsIndex | null = null;
let cacheTime: number = 0;
const CACHE_TTL_MS = 60000; // 1 minute

/**
 * Get the path to agents-index.json
 */
function getAgentsIndexPath(): string {
  // Check standard locations
  const locations = [
    path.join(process.cwd(), ".claude", "agents", "agents-index.json"),
    path.join(process.env["HOME"] || "", ".claude", "agents", "agents-index.json"),
  ];

  for (const loc of locations) {
    if (fs.existsSync(loc)) {
      return loc;
    }
  }

  throw new Error(
    "[agentConfig] agents-index.json not found. Checked: " + locations.join(", ")
  );
}

/**
 * Load agents-index.json with caching.
 */
export function loadAgentsIndex(): AgentsIndex {
  const now = Date.now();

  // Return cached if still valid
  if (cachedIndex && now - cacheTime < CACHE_TTL_MS) {
    return cachedIndex;
  }

  const indexPath = getAgentsIndexPath();
  const content = fs.readFileSync(indexPath, "utf-8");
  cachedIndex = JSON.parse(content) as AgentsIndex;
  cacheTime = now;

  return cachedIndex;
}

/**
 * Get config for a specific agent by ID.
 */
export function getAgentConfig(agentId: string): AgentConfig | null {
  const index = loadAgentsIndex();
  return index.agents.find((a) => a.id === agentId) || null;
}

/**
 * Clear the cache (for testing).
 */
export function clearAgentConfigCache(): void {
  cachedIndex = null;
  cacheTime = 0;
}
