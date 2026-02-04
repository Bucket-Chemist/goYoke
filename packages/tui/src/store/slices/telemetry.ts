/**
 * Telemetry store slice - manages real-time telemetry data from Go hooks
 * Tracks routing decisions, session handoffs, and sharp edges
 */

import type { StateCreator } from "zustand";
import type { Store } from "../types.js";

// Routing decision from gogent-validate
export interface RoutingDecision {
  timestamp: string;
  agent: string;
  model: string;
  subagent_type: string;
  tier: string;
  cost_estimate: number;
  recommended_tier?: string;
}

// Session handoff from gogent-archive
export interface Handoff {
  timestamp: string;
  session_id: string;
  summary: string;
  pending_tasks: string[];
  sharp_edges: string[];
  context: Record<string, unknown>;
}

// Sharp edge from gogent-sharp-edge
export interface SharpEdge {
  timestamp: string;
  pattern: string;
  file: string;
  description: string;
  failure_count: number;
}

export interface TelemetrySlice {
  routingDecisions: RoutingDecision[];
  lastHandoff: Handoff | null;
  sharpEdges: SharpEdge[];
  totalRoutingCost: number;
  updateTelemetry: (key: "routingDecisions" | "handoffs" | "sharpEdges", data: unknown) => void;
  clearTelemetry: () => void;
}

export const createTelemetrySlice: StateCreator<
  Store,
  [],
  [],
  TelemetrySlice
> = (set, _get) => ({
  routingDecisions: [],
  lastHandoff: null,
  sharpEdges: [],
  totalRoutingCost: 0,

  updateTelemetry: (key, data) => {
    set((state: Store) => {
      switch (key) {
        case "routingDecisions": {
          const decision = data as RoutingDecision;
          return {
            routingDecisions: [...state.routingDecisions, decision],
            totalRoutingCost: state.totalRoutingCost + (decision.cost_estimate || 0),
          };
        }
        case "handoffs": {
          const handoff = data as Handoff;
          return {
            lastHandoff: handoff,
          };
        }
        case "sharpEdges": {
          const edge = data as SharpEdge;
          return {
            sharpEdges: [...state.sharpEdges, edge],
          };
        }
        default:
          return state;
      }
    });
  },

  clearTelemetry: () => {
    set({
      routingDecisions: [],
      lastHandoff: null,
      sharpEdges: [],
      totalRoutingCost: 0,
    });
  },
});
