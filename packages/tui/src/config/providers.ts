/**
 * Provider configuration system
 * Central definitions for Anthropic, Google, OpenAI, and Local (Ollama) providers
 */

import type { ProviderDefinition, ProviderId } from "../store/types.js";
import { fileURLToPath } from "url";
import { dirname, join } from "path";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

export const PROVIDERS: Record<ProviderId, ProviderDefinition> = {
  anthropic: {
    id: "anthropic",
    name: "Anthropic",
    description: "Claude models (Opus, Sonnet, Haiku)",
    models: [
      {
        id: "opus",
        displayName: "Opus",
        description: "Most capable - deep reasoning, complex tasks",
        contextWindow: 200000,
      },
      {
        id: "sonnet",
        displayName: "Sonnet",
        description: "Balanced - quality and speed",
        contextWindow: 200000,
      },
      {
        id: "haiku",
        displayName: "Haiku",
        description: "Fastest - simple tasks, low cost",
        contextWindow: 200000,
      },
    ],
  },

  google: {
    id: "google",
    name: "Google",
    description: "Gemini models (Pro, Flash)",
    adapterPath: join(__dirname, "..", "scripts", "gemini-adapter.ts"),
    models: [
      {
        id: "gemini-pro",
        displayName: "Gemini 3 Pro",
        description: "Powerful - 1M+ token context",
        contextWindow: 1000000,
      },
      {
        id: "gemini-flash",
        displayName: "Gemini 3 Flash",
        description: "Fast - large context, quick responses",
        contextWindow: 1000000,
      },
    ],
  },

  openai: {
    id: "openai",
    name: "OpenAI",
    description: "GPT models (GPT-4, GPT-4 Turbo)",
    adapterPath: join(__dirname, "..", "scripts", "openai-adapter.ts"),
    envVars: { OPENAI_API_KEY: process.env["OPENAI_API_KEY"] || "" },
    models: [
      {
        id: "gpt-4-turbo",
        displayName: "GPT-4 Turbo",
        description: "Latest GPT-4 - 128K context",
        contextWindow: 128000,
      },
      {
        id: "gpt-4",
        displayName: "GPT-4",
        description: "Standard GPT-4 - 8K context",
        contextWindow: 8192,
      },
      {
        id: "gpt-3.5-turbo",
        displayName: "GPT-3.5 Turbo",
        description: "Fast and cheap - 16K context",
        contextWindow: 16384,
      },
    ],
  },

  local: {
    id: "local",
    name: "Local",
    description: "Ollama models (Llama, Mistral, etc.)",
    adapterPath: join(__dirname, "..", "scripts", "ollama-adapter.ts"),
    models: [
      {
        id: "llama3.1:70b",
        displayName: "Llama 3.1 70B",
        description: "Powerful open model - 128K context",
        contextWindow: 128000,
      },
      {
        id: "llama3.1:8b",
        displayName: "Llama 3.1 8B",
        description: "Fast open model - 128K context",
        contextWindow: 128000,
      },
      {
        id: "mistral:7b",
        displayName: "Mistral 7B",
        description: "Efficient open model - 32K context",
        contextWindow: 32768,
      },
    ],
  },
};

/**
 * Helper to get provider by model ID
 * @param modelId - The model identifier
 * @returns The provider ID that owns this model
 */
export function getProviderForModel(modelId: string): ProviderId {
  for (const [providerId, provider] of Object.entries(PROVIDERS)) {
    if (provider.models.some((m) => m.id === modelId)) {
      return providerId as ProviderId;
    }
  }
  return "anthropic"; // Default fallback
}
