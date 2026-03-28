// Package state provides shared, thread-safe state containers for the
// GOgent-Fortress TUI. It has no dependency on the model, cli, bridge, or any
// Bubbletea packages, keeping the import graph acyclic.
package state

import (
	"fmt"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// ProviderID
// ---------------------------------------------------------------------------

// ProviderID identifies a supported LLM provider.
type ProviderID string

const (
	// ProviderAnthropic is the Anthropic Claude API (native, no adapter).
	ProviderAnthropic ProviderID = "anthropic"
	// ProviderGoogle is the Google Gemini API (requires gemini-adapter).
	ProviderGoogle ProviderID = "google"
	// ProviderOpenAI is the OpenAI GPT API (requires openai-adapter).
	ProviderOpenAI ProviderID = "openai"
	// ProviderLocal is local Ollama inference (requires ollama-adapter).
	ProviderLocal ProviderID = "local"
)

// ---------------------------------------------------------------------------
// ModelConfig
// ---------------------------------------------------------------------------

// ModelConfig describes a single model offered by a provider.
type ModelConfig struct {
	// ID is the canonical model identifier used in API calls.
	ID string
	// DisplayName is the short human-readable label shown in the UI.
	DisplayName string
	// Description is a one-line summary of the model's characteristics.
	Description string
	// ContextWindow is the maximum context size in tokens.
	ContextWindow int
}

// ---------------------------------------------------------------------------
// ProviderConfig
// ---------------------------------------------------------------------------

// ProviderConfig is the static definition for a single provider.
//
// The configs are immutable after construction; all mutations happen through
// ProviderState methods which track per-provider mutable state separately.
type ProviderConfig struct {
	// ID uniquely identifies the provider.
	ID ProviderID
	// Name is the display name of the provider.
	Name string
	// Description is a short summary of the provider.
	Description string
	// Models is the ordered list of models available from this provider.
	Models []ModelConfig
	// AdapterPath is the name of the adapter binary required to talk to this
	// provider. Empty for Anthropic (native support).
	AdapterPath string
	// EnvVars is the map of environment variable names to descriptions
	// required by this provider. May be nil for providers that use default
	// credentials resolution (e.g. Anthropic).
	EnvVars map[string]string
}

// ---------------------------------------------------------------------------
// DisplayMessage
// ---------------------------------------------------------------------------

// ToolBlock represents a tool invocation stored within a DisplayMessage.
// This is the canonical definition used by both the state and claude packages.
type ToolBlock struct {
	// Name is the tool name (e.g., "Read", "Bash", "Edit").
	Name string
	// ToolID is the unique identifier for this tool invocation (tool_use id).
	// Used to match incoming ToolResultMsg to the correct block.
	ToolID string
	// Input is a short human-readable summary of the tool input.
	Input string
	// Output is a short human-readable summary of the tool result.
	Output string
	// Success indicates tool result status: nil=pending, true=✓, false=✗.
	Success *bool
	// Expanded controls whether the full Input/Output is shown in the UI.
	// Always starts false on restore; transient UI state only.
	Expanded bool
}

// DisplayMessage is a single rendered message in a provider's conversation
// history. It contains only the information needed for per-provider message
// isolation and UI display.
type DisplayMessage struct {
	// Role is "user", "assistant", or "system".
	Role string
	// Content is the message body text.
	Content string
	// Timestamp is when the message was created.
	Timestamp time.Time
	// ToolBlocks holds any tool invocations embedded in an assistant message,
	// preserved across provider switches (TUI R-4).
	ToolBlocks []ToolBlock
}

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

var (
	// ErrProviderNotFound is returned when an operation targets an unknown provider ID.
	ErrProviderNotFound = fmt.Errorf("provider not found")
	// ErrModelNotFound is returned when a model ID does not belong to the provider.
	ErrModelNotFound = fmt.Errorf("model not found for provider")
)

// ---------------------------------------------------------------------------
// ProviderState
// ---------------------------------------------------------------------------

// ProviderState is a thread-safe container for all provider and per-provider
// mutable state within a single TUI session.
//
// The zero value is not usable; use NewProviderState instead.
//
// Concurrency model:
//   - Write methods (SwitchProvider, SetActiveModel, AppendMessage,
//     SetSessionID, SetProjectDir) acquire a full write lock (mu.Lock).
//   - Read methods (GetActiveProvider, GetActiveConfig, GetActiveModel,
//     GetActiveMessages, GetConfig, GetProviderForModel, AllProviders)
//     acquire a shared read lock (mu.RLock).
//
// The configs map is populated once during construction (NewProviderState) and
// is never mutated afterward. All mutable per-provider state lives in the
// messages, sessionIDs, models, and projectDirs maps.
type ProviderState struct {
	active     ProviderID
	configs    map[ProviderID]ProviderConfig
	messages   map[ProviderID][]DisplayMessage
	sessionIDs map[ProviderID]string
	models     map[ProviderID]string
	projectDirs map[ProviderID]string
	mu         sync.RWMutex
}

// NewProviderState allocates a ProviderState pre-loaded with all four
// providers. Anthropic is selected as the active provider, and each
// provider's first model is selected as the active model.
func NewProviderState() *ProviderState {
	cfgs := DefaultProviders()

	// Ordered provider list — determines which provider's first model is
	// the default selection.
	order := []ProviderID{
		ProviderAnthropic,
		ProviderGoogle,
		ProviderOpenAI,
		ProviderLocal,
	}

	models := make(map[ProviderID]string, len(cfgs))
	for _, id := range order {
		cfg := cfgs[id]
		if len(cfg.Models) > 0 {
			models[id] = cfg.Models[0].ID
		}
	}

	return &ProviderState{
		active:      ProviderAnthropic,
		configs:     cfgs,
		messages:    make(map[ProviderID][]DisplayMessage),
		sessionIDs:  make(map[ProviderID]string),
		models:      models,
		projectDirs: make(map[ProviderID]string),
	}
}

// ---------------------------------------------------------------------------
// Write methods
// ---------------------------------------------------------------------------

// SwitchProvider changes the active provider to id. All per-provider state
// (messages, session ID, model selection, project directory) is preserved.
//
// Returns ErrProviderNotFound if id is not a known provider.
func (ps *ProviderState) SwitchProvider(id ProviderID) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if _, ok := ps.configs[id]; !ok {
		return fmt.Errorf("switch provider %q: %w", id, ErrProviderNotFound)
	}
	ps.active = id
	return nil
}

// SetActiveModel sets the selected model for the currently active provider.
//
// Returns ErrModelNotFound if modelID is not offered by the active provider.
func (ps *ProviderState) SetActiveModel(modelID string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	cfg := ps.configs[ps.active]
	for _, m := range cfg.Models {
		if m.ID == modelID {
			ps.models[ps.active] = modelID
			return nil
		}
	}
	return fmt.Errorf("set model %q on provider %q: %w", modelID, ps.active, ErrModelNotFound)
}

// AppendMessage appends msg to the message history of the active provider.
func (ps *ProviderState) AppendMessage(msg DisplayMessage) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.messages[ps.active] = append(ps.messages[ps.active], msg)
}

// SetSessionID records the CLI session ID for the active provider.
func (ps *ProviderState) SetSessionID(id string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.sessionIDs[ps.active] = id
}

// SetProjectDir records the project directory for the active provider.
func (ps *ProviderState) SetProjectDir(dir string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.projectDirs[ps.active] = dir
}

// SetActiveMessages replaces the entire message history for the active
// provider with a defensive copy of msgs. This is used by the provider
// switch flow to persist the current conversation before switching and to
// restore it when switching back.
//
// Passing nil or an empty slice clears the history for the active provider.
func (ps *ProviderState) SetActiveMessages(msgs []DisplayMessage) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if len(msgs) == 0 {
		ps.messages[ps.active] = nil
		return
	}
	cp := copyMessages(msgs)
	ps.messages[ps.active] = cp
}

// ---------------------------------------------------------------------------
// Read methods
// ---------------------------------------------------------------------------

// GetActiveSessionID returns the session ID for the active provider.
// Returns "" if no session ID has been set for this provider.
func (ps *ProviderState) GetActiveSessionID() string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.sessionIDs[ps.active]
}

// GetActiveProjectDir returns the project directory for the active provider.
// Returns "" if no project directory has been set for this provider.
func (ps *ProviderState) GetActiveProjectDir() string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.projectDirs[ps.active]
}

// ExportSessionIDs returns a copy of all per-provider session IDs for
// persistence. Only non-empty session IDs are included in the result.
// The returned map is safe to mutate; it does not alias internal state.
func (ps *ProviderState) ExportSessionIDs() map[ProviderID]string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	result := make(map[ProviderID]string)
	for k, v := range ps.sessionIDs {
		if v != "" {
			result[k] = v
		}
	}
	return result
}

// ImportSessionIDs populates session IDs from saved data (e.g. session resume).
// Existing non-empty values are NOT overwritten — this is additive.
// Entries for unknown providers are silently ignored.
func (ps *ProviderState) ImportSessionIDs(ids map[ProviderID]string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	for k, v := range ids {
		if _, ok := ps.configs[k]; !ok {
			continue // unknown provider — skip
		}
		if v != "" && ps.sessionIDs[k] == "" {
			ps.sessionIDs[k] = v
		}
	}
}

// ExportModels returns a copy of all per-provider active model selections for
// persistence. All providers are included, even those using their default model.
// The returned map is safe to mutate; it does not alias internal state.
func (ps *ProviderState) ExportModels() map[ProviderID]string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	result := make(map[ProviderID]string, len(ps.models))
	for k, v := range ps.models {
		result[k] = v
	}
	return result
}

// ImportModels populates model selections from saved data.
// Only imports for known providers with models that belong to that provider.
// Unknown providers and invalid model IDs are silently ignored.
func (ps *ProviderState) ImportModels(models map[ProviderID]string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	for provID, modelID := range models {
		cfg, ok := ps.configs[provID]
		if !ok {
			continue
		}
		for _, m := range cfg.Models {
			if m.ID == modelID {
				ps.models[provID] = modelID
				break
			}
		}
	}
}

// GetActiveProvider returns the ID of the currently active provider.
func (ps *ProviderState) GetActiveProvider() ProviderID {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	return ps.active
}

// GetActiveConfig returns a copy of the configuration for the active provider.
func (ps *ProviderState) GetActiveConfig() ProviderConfig {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	return ps.copyConfig(ps.configs[ps.active])
}

// GetActiveModel returns the selected model ID for the active provider.
func (ps *ProviderState) GetActiveModel() string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	return ps.models[ps.active]
}

// GetActiveMessages returns a copy of the message history for the active
// provider. The returned slice is safe to read without coordination; mutations
// do not affect the internal history, including any ToolBlocks slices.
func (ps *ProviderState) GetActiveMessages() []DisplayMessage {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	msgs := ps.messages[ps.active]
	if len(msgs) == 0 {
		return nil
	}
	return copyMessages(msgs)
}

// GetConfig returns a copy of the configuration for the specified provider.
// The second return value is false if id is not a known provider.
func (ps *ProviderState) GetConfig(id ProviderID) (ProviderConfig, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	cfg, ok := ps.configs[id]
	if !ok {
		return ProviderConfig{}, false
	}
	return ps.copyConfig(cfg), true
}

// GetProviderForModel searches all provider configs for modelID and returns
// the first matching provider ID. The second return value is false when no
// provider offers the given model.
func (ps *ProviderState) GetProviderForModel(modelID string) (ProviderID, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	for _, id := range orderedProviderIDs() {
		cfg, ok := ps.configs[id]
		if !ok {
			continue
		}
		for _, m := range cfg.Models {
			if m.ID == modelID {
				return id, true
			}
		}
	}
	return "", false
}

// GetMessages returns a copy of the message history for the specified provider.
// The returned slice is safe to read without coordination; mutations do not
// affect the internal history. Returns nil when the provider has no history or
// is unknown.
func (ps *ProviderState) GetMessages(provider ProviderID) []DisplayMessage {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	msgs := ps.messages[provider]
	if len(msgs) == 0 {
		return nil
	}
	return copyMessages(msgs)
}

// ExportAllMessages returns a deep copy of all per-provider message histories
// for persistence. Only providers with at least one message are included.
// The returned map and its slices are safe to mutate; they do not alias
// internal state.
func (ps *ProviderState) ExportAllMessages() map[ProviderID][]DisplayMessage {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	result := make(map[ProviderID][]DisplayMessage)
	for k, msgs := range ps.messages {
		if len(msgs) > 0 {
			result[k] = copyMessages(msgs)
		}
	}
	return result
}

// AllProviders returns the canonical ordered list of all provider IDs.
func (ps *ProviderState) AllProviders() []ProviderID {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	return orderedProviderIDs()
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// copyConfig returns a deep copy of cfg so callers cannot mutate internal
// slices or maps.
func (ps *ProviderState) copyConfig(cfg ProviderConfig) ProviderConfig {
	cp := cfg

	if cfg.Models != nil {
		cp.Models = make([]ModelConfig, len(cfg.Models))
		copy(cp.Models, cfg.Models)
	}

	if cfg.EnvVars != nil {
		cp.EnvVars = make(map[string]string, len(cfg.EnvVars))
		for k, v := range cfg.EnvVars {
			cp.EnvVars[k] = v
		}
	}

	return cp
}

// copyMessages returns a deep copy of msgs. Each DisplayMessage's ToolBlocks
// slice is independently copied so callers cannot mutate internal state.
func copyMessages(msgs []DisplayMessage) []DisplayMessage {
	cp := make([]DisplayMessage, len(msgs))
	for i, msg := range msgs {
		cp[i] = msg
		if len(msg.ToolBlocks) > 0 {
			cp[i].ToolBlocks = make([]ToolBlock, len(msg.ToolBlocks))
			copy(cp[i].ToolBlocks, msg.ToolBlocks)
		}
	}
	return cp
}

// orderedProviderIDs returns the canonical display order of all provider IDs.
// This must be a stable, consistent order used across AllProviders and
// GetProviderForModel.
func orderedProviderIDs() []ProviderID {
	return []ProviderID{
		ProviderAnthropic,
		ProviderGoogle,
		ProviderOpenAI,
		ProviderLocal,
	}
}

// ---------------------------------------------------------------------------
// DefaultProviders
// ---------------------------------------------------------------------------

// DefaultProviders returns the hardcoded provider configurations for all four
// supported providers. The returned map is freshly allocated on each call and
// is safe to mutate.
//
// Model context windows use the following defaults:
//   - Anthropic: 200 000 tokens
//   - Google Gemini: 1 048 576 tokens (1M+)
//   - GPT-4 Turbo: 128 000 tokens
//   - GPT-4: 8 192 tokens
//   - GPT-3.5 Turbo: 16 385 tokens
//   - Llama 3.1 (both): 131 072 tokens (128K)
//   - Mistral 7B: 32 768 tokens (32K)
func DefaultProviders() map[ProviderID]ProviderConfig {
	return map[ProviderID]ProviderConfig{
		ProviderAnthropic: {
			ID:          ProviderAnthropic,
			Name:        "Anthropic",
			Description: "Claude models — native provider, no adapter required",
			AdapterPath: "",
			EnvVars:     nil,
			Models: []ModelConfig{
				{
					ID:            "opus",
					DisplayName:   "Opus",
					Description:   "Most capable - deep reasoning, complex tasks",
					ContextWindow: 200_000,
				},
				{
					ID:            "sonnet",
					DisplayName:   "Sonnet",
					Description:   "Balanced - quality and speed",
					ContextWindow: 200_000,
				},
				{
					ID:            "haiku",
					DisplayName:   "Haiku",
					Description:   "Fastest - simple tasks, low cost",
					ContextWindow: 200_000,
				},
			},
		},

		ProviderGoogle: {
			ID:          ProviderGoogle,
			Name:        "Google",
			Description: "Gemini models — large context window",
			AdapterPath: "gemini-adapter",
			EnvVars:     nil,
			Models: []ModelConfig{
				{
					ID:            "gemini-pro",
					DisplayName:   "Gemini 3 Pro",
					Description:   "Powerful - 1M+ token context",
					ContextWindow: 1_048_576,
				},
				{
					ID:            "gemini-flash",
					DisplayName:   "Gemini 3 Flash",
					Description:   "Fast - large context, quick responses",
					ContextWindow: 1_048_576,
				},
			},
		},

		ProviderOpenAI: {
			ID:          ProviderOpenAI,
			Name:        "OpenAI",
			Description: "GPT models via OpenAI API",
			AdapterPath: "openai-adapter",
			EnvVars: map[string]string{
				"OPENAI_API_KEY": "API key for authenticating with the OpenAI API",
			},
			Models: []ModelConfig{
				{
					ID:            "gpt-4-turbo",
					DisplayName:   "GPT-4 Turbo",
					Description:   "Latest GPT-4 - 128K context",
					ContextWindow: 128_000,
				},
				{
					ID:            "gpt-4",
					DisplayName:   "GPT-4",
					Description:   "Standard GPT-4 - 8K context",
					ContextWindow: 8_192,
				},
				{
					ID:            "gpt-3.5-turbo",
					DisplayName:   "GPT-3.5 Turbo",
					Description:   "Fast and cheap - 16K context",
					ContextWindow: 16_385,
				},
			},
		},

		ProviderLocal: {
			ID:          ProviderLocal,
			Name:        "Local / Ollama",
			Description: "Local inference via Ollama",
			AdapterPath: "ollama-adapter",
			EnvVars: map[string]string{
				"OLLAMA_ENDPOINT": "Ollama server URL (default: http://localhost:11434)",
			},
			Models: []ModelConfig{
				{
					ID:            "llama3.1:70b",
					DisplayName:   "Llama 3.1 70B",
					Description:   "Powerful open model - 128K context",
					ContextWindow: 131_072,
				},
				{
					ID:            "llama3.1:8b",
					DisplayName:   "Llama 3.1 8B",
					Description:   "Fast open model - 128K context",
					ContextWindow: 131_072,
				},
				{
					ID:            "mistral:7b",
					DisplayName:   "Mistral 7B",
					Description:   "Efficient open model - 32K context",
					ContextWindow: 32_768,
				},
			},
		},
	}
}
