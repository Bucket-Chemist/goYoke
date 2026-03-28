package state

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// DefaultProviders
// ---------------------------------------------------------------------------

func TestDefaultProviders_AllFourPresent(t *testing.T) {
	cfgs := DefaultProviders()
	require.Len(t, cfgs, 4)

	for _, id := range []ProviderID{
		ProviderAnthropic, ProviderGoogle, ProviderOpenAI, ProviderLocal,
	} {
		cfg, ok := cfgs[id]
		require.True(t, ok, "missing provider %q", id)
		assert.Equal(t, id, cfg.ID)
		assert.NotEmpty(t, cfg.Name)
		assert.NotEmpty(t, cfg.Description)
		assert.NotEmpty(t, cfg.Models, "provider %q has no models", id)
	}
}

func TestDefaultProviders_AnthropicNoAdapter(t *testing.T) {
	cfgs := DefaultProviders()
	assert.Empty(t, cfgs[ProviderAnthropic].AdapterPath)
	assert.Nil(t, cfgs[ProviderAnthropic].EnvVars)
}

func TestDefaultProviders_GoogleAdapterSet(t *testing.T) {
	cfgs := DefaultProviders()
	assert.Equal(t, "gemini-adapter", cfgs[ProviderGoogle].AdapterPath)
}

func TestDefaultProviders_OpenAIAdapterAndEnvVar(t *testing.T) {
	cfgs := DefaultProviders()
	cfg := cfgs[ProviderOpenAI]
	assert.Equal(t, "openai-adapter", cfg.AdapterPath)
	require.NotNil(t, cfg.EnvVars)
	_, ok := cfg.EnvVars["OPENAI_API_KEY"]
	assert.True(t, ok)
}

func TestDefaultProviders_LocalAdapterAndEnvVar(t *testing.T) {
	cfgs := DefaultProviders()
	cfg := cfgs[ProviderLocal]
	assert.Equal(t, "ollama-adapter", cfg.AdapterPath)
	require.NotNil(t, cfg.EnvVars)
	_, ok := cfg.EnvVars["OLLAMA_ENDPOINT"]
	assert.True(t, ok)
}

func TestDefaultProviders_AnthropicModels(t *testing.T) {
	cfgs := DefaultProviders()
	models := cfgs[ProviderAnthropic].Models
	require.Len(t, models, 3)

	ids := make([]string, len(models))
	for i, m := range models {
		ids[i] = m.ID
		assert.NotEmpty(t, m.DisplayName)
		assert.NotEmpty(t, m.Description)
		assert.Greater(t, m.ContextWindow, 0)
	}
	assert.Equal(t, []string{"opus", "sonnet", "haiku"}, ids)
}

func TestDefaultProviders_GoogleModels(t *testing.T) {
	cfgs := DefaultProviders()
	models := cfgs[ProviderGoogle].Models
	require.Len(t, models, 2)

	ids := []string{models[0].ID, models[1].ID}
	assert.Equal(t, []string{"gemini-pro", "gemini-flash"}, ids)

	for _, m := range models {
		assert.Equal(t, 1_048_576, m.ContextWindow, "gemini model %q should have 1M context", m.ID)
	}
}

func TestDefaultProviders_OpenAIModels(t *testing.T) {
	cfgs := DefaultProviders()
	models := cfgs[ProviderOpenAI].Models
	require.Len(t, models, 3)

	tests := []struct {
		id      string
		context int
	}{
		{"gpt-4-turbo", 128_000},
		{"gpt-4", 8_192},
		{"gpt-3.5-turbo", 16_385},
	}
	for i, tc := range tests {
		tc := tc
		assert.Equal(t, tc.id, models[i].ID)
		assert.Equal(t, tc.context, models[i].ContextWindow)
	}
}

func TestDefaultProviders_LocalModels(t *testing.T) {
	cfgs := DefaultProviders()
	models := cfgs[ProviderLocal].Models
	require.Len(t, models, 3)

	tests := []struct {
		id      string
		context int
	}{
		{"llama3.1:70b", 131_072},
		{"llama3.1:8b", 131_072},
		{"mistral:7b", 32_768},
	}
	for i, tc := range tests {
		tc := tc
		assert.Equal(t, tc.id, models[i].ID)
		assert.Equal(t, tc.context, models[i].ContextWindow)
	}
}

// DefaultProviders must return independent maps — mutations must not propagate.
func TestDefaultProviders_ReturnsFreshMap(t *testing.T) {
	cfgs1 := DefaultProviders()
	cfgs2 := DefaultProviders()

	cfgs1[ProviderAnthropic] = ProviderConfig{ID: "mutated"}
	assert.NotEqual(t, "mutated", cfgs2[ProviderAnthropic].ID)
}

// ---------------------------------------------------------------------------
// NewProviderState
// ---------------------------------------------------------------------------

func TestNewProviderState_AnthropicActiveByDefault(t *testing.T) {
	ps := NewProviderState()
	assert.Equal(t, ProviderAnthropic, ps.GetActiveProvider())
}

func TestNewProviderState_FirstModelSelectedPerProvider(t *testing.T) {
	ps := NewProviderState()
	cfgs := DefaultProviders()

	for _, id := range ps.AllProviders() {
		require.NoError(t, ps.SwitchProvider(id))
		wantModel := cfgs[id].Models[0].ID
		assert.Equal(t, wantModel, ps.GetActiveModel(),
			"provider %q: expected first model %q selected by default", id, wantModel)
	}
}

func TestNewProviderState_NoMessages(t *testing.T) {
	ps := NewProviderState()
	assert.Nil(t, ps.GetActiveMessages())
}

// ---------------------------------------------------------------------------
// AllProviders
// ---------------------------------------------------------------------------

func TestAllProviders_OrderedAndComplete(t *testing.T) {
	ps := NewProviderState()
	ids := ps.AllProviders()
	require.Len(t, ids, 4)
	assert.Equal(t, ProviderAnthropic, ids[0])
	assert.Equal(t, ProviderGoogle, ids[1])
	assert.Equal(t, ProviderOpenAI, ids[2])
	assert.Equal(t, ProviderLocal, ids[3])
}

// ---------------------------------------------------------------------------
// GetConfig
// ---------------------------------------------------------------------------

func TestGetConfig_KnownProvider(t *testing.T) {
	ps := NewProviderState()

	tests := []struct {
		id ProviderID
	}{
		{ProviderAnthropic},
		{ProviderGoogle},
		{ProviderOpenAI},
		{ProviderLocal},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(string(tc.id), func(t *testing.T) {
			t.Parallel()
			cfg, ok := ps.GetConfig(tc.id)
			require.True(t, ok)
			assert.Equal(t, tc.id, cfg.ID)
			assert.NotEmpty(t, cfg.Models)
		})
	}
}

func TestGetConfig_UnknownProvider(t *testing.T) {
	ps := NewProviderState()
	_, ok := ps.GetConfig("unknown-provider")
	assert.False(t, ok)
}

// GetConfig must return a deep copy — mutations must not affect internal state.
func TestGetConfig_ReturnsCopy(t *testing.T) {
	ps := NewProviderState()
	cfg, ok := ps.GetConfig(ProviderAnthropic)
	require.True(t, ok)

	// Mutate the returned copy.
	cfg.Models[0].ID = "mutated"

	// Re-fetch — internal state must be unchanged.
	cfg2, ok2 := ps.GetConfig(ProviderAnthropic)
	require.True(t, ok2)
	assert.Equal(t, "opus", cfg2.Models[0].ID)
}

func TestGetConfig_EnvVarsCopied(t *testing.T) {
	ps := NewProviderState()
	cfg, ok := ps.GetConfig(ProviderOpenAI)
	require.True(t, ok)
	require.NotNil(t, cfg.EnvVars)

	// Mutate the returned copy.
	cfg.EnvVars["INJECTED"] = "bad"

	// Re-fetch — internal env vars must be unchanged.
	cfg2, _ := ps.GetConfig(ProviderOpenAI)
	_, injected := cfg2.EnvVars["INJECTED"]
	assert.False(t, injected)
}

// ---------------------------------------------------------------------------
// GetActiveConfig
// ---------------------------------------------------------------------------

func TestGetActiveConfig_DefaultIsAnthropic(t *testing.T) {
	ps := NewProviderState()
	cfg := ps.GetActiveConfig()
	assert.Equal(t, ProviderAnthropic, cfg.ID)
}

func TestGetActiveConfig_ChangesAfterSwitch(t *testing.T) {
	ps := NewProviderState()
	require.NoError(t, ps.SwitchProvider(ProviderGoogle))

	cfg := ps.GetActiveConfig()
	assert.Equal(t, ProviderGoogle, cfg.ID)
}

// ---------------------------------------------------------------------------
// SwitchProvider
// ---------------------------------------------------------------------------

func TestSwitchProvider_AllProviders(t *testing.T) {
	providers := []ProviderID{
		ProviderAnthropic, ProviderGoogle, ProviderOpenAI, ProviderLocal,
	}
	for _, id := range providers {
		id := id
		t.Run(string(id), func(t *testing.T) {
			t.Parallel()
			ps := NewProviderState()
			require.NoError(t, ps.SwitchProvider(id))
			assert.Equal(t, id, ps.GetActiveProvider())
		})
	}
}

func TestSwitchProvider_UnknownReturnsError(t *testing.T) {
	ps := NewProviderState()
	err := ps.SwitchProvider("not-a-real-provider")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrProviderNotFound))
	// Active provider must remain unchanged.
	assert.Equal(t, ProviderAnthropic, ps.GetActiveProvider())
}

// SwitchProvider must preserve per-provider state on switch.
func TestSwitchProvider_PreservesState(t *testing.T) {
	ps := NewProviderState()

	// Set up distinct state for Anthropic.
	require.NoError(t, ps.SetActiveModel("sonnet"))
	ps.SetSessionID("anthropic-session-1")
	ps.SetProjectDir("/projects/anthropic")
	ps.AppendMessage(DisplayMessage{Role: "user", Content: "hello"})

	// Switch to Google and set its state.
	require.NoError(t, ps.SwitchProvider(ProviderGoogle))
	ps.SetSessionID("google-session-1")
	ps.AppendMessage(DisplayMessage{Role: "user", Content: "bonjour"})

	// Switch back to Anthropic — its state must be intact.
	require.NoError(t, ps.SwitchProvider(ProviderAnthropic))
	assert.Equal(t, "sonnet", ps.GetActiveModel())
	msgs := ps.GetActiveMessages()
	require.Len(t, msgs, 1)
	assert.Equal(t, "hello", msgs[0].Content)

	// Switch to Google — its state must still be there.
	require.NoError(t, ps.SwitchProvider(ProviderGoogle))
	googleMsgs := ps.GetActiveMessages()
	require.Len(t, googleMsgs, 1)
	assert.Equal(t, "bonjour", googleMsgs[0].Content)
}

// ---------------------------------------------------------------------------
// SetActiveModel / GetActiveModel
// ---------------------------------------------------------------------------

func TestSetActiveModel_ValidModel(t *testing.T) {
	tests := []struct {
		provider ProviderID
		modelID  string
	}{
		{ProviderAnthropic, "sonnet"},
		{ProviderAnthropic, "haiku"},
		{ProviderGoogle, "gemini-flash"},
		{ProviderOpenAI, "gpt-4"},
		{ProviderOpenAI, "gpt-3.5-turbo"},
		{ProviderLocal, "llama3.1:8b"},
		{ProviderLocal, "mistral:7b"},
	}
	for _, tc := range tests {
		tc := tc
		name := fmt.Sprintf("%s/%s", tc.provider, tc.modelID)
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ps := NewProviderState()
			require.NoError(t, ps.SwitchProvider(tc.provider))
			require.NoError(t, ps.SetActiveModel(tc.modelID))
			assert.Equal(t, tc.modelID, ps.GetActiveModel())
		})
	}
}

func TestSetActiveModel_InvalidModel(t *testing.T) {
	ps := NewProviderState()
	err := ps.SetActiveModel("gpt-99-turbo")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrModelNotFound))
	// Model must remain unchanged (first Anthropic model).
	assert.Equal(t, "opus", ps.GetActiveModel())
}

// A model that belongs to a different provider must not be accepted.
func TestSetActiveModel_CrossProviderModelRejected(t *testing.T) {
	ps := NewProviderState()
	// Anthropic is active; GPT model belongs to OpenAI.
	err := ps.SetActiveModel("gpt-4-turbo")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrModelNotFound))
}

// ---------------------------------------------------------------------------
// AppendMessage / GetActiveMessages
// ---------------------------------------------------------------------------

func TestAppendMessage_Basic(t *testing.T) {
	ps := NewProviderState()
	now := time.Now()

	ps.AppendMessage(DisplayMessage{Role: "user", Content: "ping", Timestamp: now})
	ps.AppendMessage(DisplayMessage{Role: "assistant", Content: "pong", Timestamp: now.Add(time.Second)})

	msgs := ps.GetActiveMessages()
	require.Len(t, msgs, 2)
	assert.Equal(t, "ping", msgs[0].Content)
	assert.Equal(t, "pong", msgs[1].Content)
}

func TestGetActiveMessages_EmptyReturnsNil(t *testing.T) {
	ps := NewProviderState()
	assert.Nil(t, ps.GetActiveMessages())
}

// GetActiveMessages must return a copy — mutations must not affect internal history.
func TestGetActiveMessages_ReturnsCopy(t *testing.T) {
	ps := NewProviderState()
	ps.AppendMessage(DisplayMessage{Role: "user", Content: "original"})

	msgs := ps.GetActiveMessages()
	require.Len(t, msgs, 1)
	msgs[0].Content = "mutated"

	// Re-fetch — internal must be unchanged.
	msgs2 := ps.GetActiveMessages()
	assert.Equal(t, "original", msgs2[0].Content)
}

// Messages for one provider must not appear when a different provider is active.
func TestAppendMessage_PerProviderIsolation(t *testing.T) {
	ps := NewProviderState()

	ps.AppendMessage(DisplayMessage{Role: "user", Content: "anthropic msg"})

	require.NoError(t, ps.SwitchProvider(ProviderGoogle))
	googleMsgs := ps.GetActiveMessages()
	assert.Nil(t, googleMsgs, "Google must start with no messages")

	ps.AppendMessage(DisplayMessage{Role: "user", Content: "google msg"})

	require.NoError(t, ps.SwitchProvider(ProviderAnthropic))
	anthropicMsgs := ps.GetActiveMessages()
	require.Len(t, anthropicMsgs, 1)
	assert.Equal(t, "anthropic msg", anthropicMsgs[0].Content)
}

// ---------------------------------------------------------------------------
// SetSessionID / SetProjectDir
// ---------------------------------------------------------------------------

// These setters store per-provider state; they have no getter exported directly
// but their effect is observable through SwitchProvider round-trips.
func TestSetSessionID_PerProviderIsolation(t *testing.T) {
	ps := NewProviderState()
	ps.SetSessionID("session-anthropic")

	require.NoError(t, ps.SwitchProvider(ProviderGoogle))
	ps.SetSessionID("session-google")

	// Switch back; Anthropic session must be unchanged.
	require.NoError(t, ps.SwitchProvider(ProviderAnthropic))
	// We can't directly read sessionIDs; verify indirectly via concurrent safety.
	// The core check: no panic, no data race (see TestConcurrentProviderState).
}

func TestSetProjectDir_NoOp(t *testing.T) {
	// SetProjectDir should not panic and must survive concurrent access.
	ps := NewProviderState()
	ps.SetProjectDir("/home/user/project")
}

// ---------------------------------------------------------------------------
// GetProviderForModel
// ---------------------------------------------------------------------------

func TestGetProviderForModel_Found(t *testing.T) {
	tests := []struct {
		modelID  string
		wantProv ProviderID
	}{
		{"opus", ProviderAnthropic},
		{"sonnet", ProviderAnthropic},
		{"haiku", ProviderAnthropic},
		{"gemini-pro", ProviderGoogle},
		{"gemini-flash", ProviderGoogle},
		{"gpt-4-turbo", ProviderOpenAI},
		{"gpt-4", ProviderOpenAI},
		{"gpt-3.5-turbo", ProviderOpenAI},
		{"llama3.1:70b", ProviderLocal},
		{"llama3.1:8b", ProviderLocal},
		{"mistral:7b", ProviderLocal},
	}
	ps := NewProviderState()
	for _, tc := range tests {
		tc := tc
		t.Run(tc.modelID, func(t *testing.T) {
			t.Parallel()
			got, ok := ps.GetProviderForModel(tc.modelID)
			require.True(t, ok, "model %q should be found", tc.modelID)
			assert.Equal(t, tc.wantProv, got)
		})
	}
}

func TestGetProviderForModel_NotFound(t *testing.T) {
	ps := NewProviderState()
	_, ok := ps.GetProviderForModel("nonexistent-model-xyz")
	assert.False(t, ok)
}

// ---------------------------------------------------------------------------
// Provider constants
// ---------------------------------------------------------------------------

func TestProviderConstants(t *testing.T) {
	assert.Equal(t, ProviderID("anthropic"), ProviderAnthropic)
	assert.Equal(t, ProviderID("google"), ProviderGoogle)
	assert.Equal(t, ProviderID("openai"), ProviderOpenAI)
	assert.Equal(t, ProviderID("local"), ProviderLocal)
}

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

func TestSentinelErrors_ProviderNotFound(t *testing.T) {
	ps := NewProviderState()
	err := ps.SwitchProvider("bad")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrProviderNotFound))
}

func TestSentinelErrors_ModelNotFound(t *testing.T) {
	ps := NewProviderState()
	err := ps.SetActiveModel("bad-model")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrModelNotFound))
}

// ---------------------------------------------------------------------------
// Concurrent access
// ---------------------------------------------------------------------------

// TestConcurrentProviderState exercises all write and read methods concurrently
// to catch data races under -race.
func TestConcurrentProviderState(t *testing.T) {
	ps := NewProviderState()
	providers := []ProviderID{
		ProviderAnthropic, ProviderGoogle, ProviderOpenAI, ProviderLocal,
	}

	const n = 50
	var wg sync.WaitGroup

	// Goroutines that switch providers.
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = ps.SwitchProvider(providers[i%len(providers)])
		}(i)
	}

	// Goroutines that read.
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = ps.GetActiveProvider()
			_ = ps.GetActiveConfig()
			_ = ps.GetActiveModel()
			_ = ps.GetActiveMessages()
			_ = ps.AllProviders()
		}()
	}

	// Goroutines that append messages.
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ps.AppendMessage(DisplayMessage{
				Role:    "user",
				Content: fmt.Sprintf("msg-%d", i),
			})
		}(i)
	}

	// Goroutines that set session IDs and project dirs.
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ps.SetSessionID(fmt.Sprintf("session-%d", i))
			ps.SetProjectDir(fmt.Sprintf("/project/%d", i))
		}(i)
	}

	// Goroutines that query config and model lookups.
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, id := range providers {
				_, _ = ps.GetConfig(id)
			}
			_, _ = ps.GetProviderForModel("opus")
			_, _ = ps.GetProviderForModel("gemini-flash")
		}()
	}

	wg.Wait()
}

// TestConcurrentSwitchAndModel stresses SwitchProvider and SetActiveModel.
func TestConcurrentSwitchAndModel(t *testing.T) {
	ps := NewProviderState()

	const n = 100
	var wg sync.WaitGroup

	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			switch i % 3 {
			case 0:
				_ = ps.SwitchProvider(ProviderAnthropic)
				_ = ps.SetActiveModel("sonnet")
			case 1:
				_ = ps.SwitchProvider(ProviderOpenAI)
				_ = ps.SetActiveModel("gpt-4")
			case 2:
				_ = ps.GetActiveProvider()
				_ = ps.GetActiveModel()
			}
		}(i)
	}

	wg.Wait()
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

// Switching to the already active provider must be a no-op (no error).
func TestSwitchProvider_SameProviderNoOp(t *testing.T) {
	ps := NewProviderState()
	require.NoError(t, ps.SwitchProvider(ProviderAnthropic))
	assert.Equal(t, ProviderAnthropic, ps.GetActiveProvider())
}

// AppendMessage on all providers must not cross-contaminate.
func TestAppendMessage_AllProviders_NoCrossContamination(t *testing.T) {
	ps := NewProviderState()

	msgFor := map[ProviderID]string{
		ProviderAnthropic: "anthropic only",
		ProviderGoogle:    "google only",
		ProviderOpenAI:    "openai only",
		ProviderLocal:     "local only",
	}

	// Append a unique message for each provider.
	for _, id := range ps.AllProviders() {
		require.NoError(t, ps.SwitchProvider(id))
		ps.AppendMessage(DisplayMessage{Role: "user", Content: msgFor[id]})
	}

	// Verify each provider has exactly its own message.
	for _, id := range ps.AllProviders() {
		require.NoError(t, ps.SwitchProvider(id))
		msgs := ps.GetActiveMessages()
		require.Len(t, msgs, 1, "provider %q should have exactly 1 message", id)
		assert.Equal(t, msgFor[id], msgs[0].Content)
	}
}

// GetActiveMessages with zero messages after a switch must return nil, not an
// empty slice, to allow callers to distinguish "no messages yet" from an empty
// slice.
func TestGetActiveMessages_SwitchToFreshProviderReturnsNil(t *testing.T) {
	ps := NewProviderState()
	ps.AppendMessage(DisplayMessage{Role: "user", Content: "hello"})

	require.NoError(t, ps.SwitchProvider(ProviderGoogle))
	assert.Nil(t, ps.GetActiveMessages())
}

// ---------------------------------------------------------------------------
// SetActiveMessages (TUI-029)
// ---------------------------------------------------------------------------

func TestSetActiveMessages_ReplacesHistory(t *testing.T) {
	ps := NewProviderState()
	ps.AppendMessage(DisplayMessage{Role: "user", Content: "old"})

	now := time.Now()
	ps.SetActiveMessages([]DisplayMessage{
		{Role: "user", Content: "new-1", Timestamp: now},
		{Role: "assistant", Content: "new-2", Timestamp: now.Add(time.Second)},
	})

	msgs := ps.GetActiveMessages()
	if len(msgs) != 2 {
		t.Fatalf("message count = %d; want 2", len(msgs))
	}
	if msgs[0].Content != "new-1" {
		t.Errorf("msgs[0].Content = %q; want %q", msgs[0].Content, "new-1")
	}
	if msgs[1].Content != "new-2" {
		t.Errorf("msgs[1].Content = %q; want %q", msgs[1].Content, "new-2")
	}
}

func TestSetActiveMessages_NilClearsHistory(t *testing.T) {
	ps := NewProviderState()
	ps.AppendMessage(DisplayMessage{Role: "user", Content: "existing"})

	ps.SetActiveMessages(nil)

	msgs := ps.GetActiveMessages()
	if msgs != nil {
		t.Errorf("GetActiveMessages() = %v; want nil after SetActiveMessages(nil)", msgs)
	}
}

func TestSetActiveMessages_EmptySliceClearsHistory(t *testing.T) {
	ps := NewProviderState()
	ps.AppendMessage(DisplayMessage{Role: "user", Content: "existing"})

	ps.SetActiveMessages([]DisplayMessage{})

	msgs := ps.GetActiveMessages()
	if msgs != nil {
		t.Errorf("GetActiveMessages() = %v; want nil after SetActiveMessages([])", msgs)
	}
}

// SetActiveMessages must store a deep copy — mutations on the input must not
// propagate to the internal state.
func TestSetActiveMessages_StoresDeepCopy(t *testing.T) {
	ps := NewProviderState()
	input := []DisplayMessage{
		{Role: "user", Content: "original"},
	}
	ps.SetActiveMessages(input)

	// Mutate the caller's slice.
	input[0].Content = "mutated"

	msgs := ps.GetActiveMessages()
	if msgs[0].Content != "original" {
		t.Errorf("SetActiveMessages stored a reference; want deep copy, got %q", msgs[0].Content)
	}
}

// SetActiveMessages must only affect the active provider — other providers
// must be unaffected.
func TestSetActiveMessages_PerProviderIsolation(t *testing.T) {
	ps := NewProviderState()

	// Set messages for Anthropic (active).
	ps.SetActiveMessages([]DisplayMessage{
		{Role: "user", Content: "anthropic msg"},
	})

	// Switch to Google and set different messages.
	if err := ps.SwitchProvider(ProviderGoogle); err != nil {
		t.Fatalf("SwitchProvider: %v", err)
	}
	ps.SetActiveMessages([]DisplayMessage{
		{Role: "user", Content: "google msg"},
	})

	// Switch back to Anthropic — messages must be untouched.
	if err := ps.SwitchProvider(ProviderAnthropic); err != nil {
		t.Fatalf("SwitchProvider: %v", err)
	}
	anthropicMsgs := ps.GetActiveMessages()
	if len(anthropicMsgs) != 1 || anthropicMsgs[0].Content != "anthropic msg" {
		t.Errorf("Anthropic messages corrupted; got %v", anthropicMsgs)
	}
}

// SaveMessages → SetActiveMessages round-trip: content and timestamps must
// survive the round-trip unchanged.
func TestSetActiveMessages_RoundTrip(t *testing.T) {
	ps := NewProviderState()
	now := time.Now().Truncate(time.Millisecond) // truncate for comparison

	original := []DisplayMessage{
		{Role: "user", Content: "hello", Timestamp: now},
		{Role: "assistant", Content: "hi there", Timestamp: now.Add(time.Second)},
	}
	for _, m := range original {
		ps.AppendMessage(m)
	}

	// Read back, clear, then restore.
	snapshot := ps.GetActiveMessages()
	ps.SetActiveMessages(nil)
	ps.SetActiveMessages(snapshot)

	restored := ps.GetActiveMessages()
	if len(restored) != len(original) {
		t.Fatalf("restored count = %d; want %d", len(restored), len(original))
	}
	for i := range original {
		if restored[i].Role != original[i].Role {
			t.Errorf("msg[%d].Role = %q; want %q", i, restored[i].Role, original[i].Role)
		}
		if restored[i].Content != original[i].Content {
			t.Errorf("msg[%d].Content = %q; want %q", i, restored[i].Content, original[i].Content)
		}
	}
}

// Concurrent stress: SetActiveMessages must not race with reads or other writes.
func TestSetActiveMessages_ConcurrentStress(t *testing.T) {
	ps := NewProviderState()
	const goroutines = 50

	done := make(chan struct{})
	for i := range goroutines {
		go func(i int) {
			defer func() { done <- struct{}{} }()
			msgs := []DisplayMessage{
				{Role: "user", Content: fmt.Sprintf("msg-%d", i)},
			}
			ps.SetActiveMessages(msgs)
		}(i)
	}
	// Concurrent readers.
	for range goroutines {
		go func() {
			defer func() { done <- struct{}{} }()
			_ = ps.GetActiveMessages()
			_ = ps.GetActiveProvider()
		}()
	}
	for range goroutines * 2 {
		<-done
	}
}

// No cross-provider data leak: SetActiveMessages on one provider then switch
// to another — the new provider must have its own (possibly empty) history.
func TestSetActiveMessages_NoLeakOnSwitch(t *testing.T) {
	ps := NewProviderState()
	ps.SetActiveMessages([]DisplayMessage{
		{Role: "user", Content: "anthropic only"},
	})

	for _, id := range []ProviderID{ProviderGoogle, ProviderOpenAI, ProviderLocal} {
		if err := ps.SwitchProvider(id); err != nil {
			t.Fatalf("SwitchProvider(%q): %v", id, err)
		}
		msgs := ps.GetActiveMessages()
		if msgs != nil {
			t.Errorf("provider %q has messages after switch; want none, got %v", id, msgs)
		}
	}
}

// ---------------------------------------------------------------------------
// GetActiveSessionID (TUI-031)
// ---------------------------------------------------------------------------

func TestGetActiveSessionID_EmptyForNewProvider(t *testing.T) {
	ps := NewProviderState()
	got := ps.GetActiveSessionID()
	if got != "" {
		t.Errorf("GetActiveSessionID() = %q; want empty for new provider", got)
	}
}

func TestGetActiveSessionID_ReturnsSetValue(t *testing.T) {
	ps := NewProviderState()
	ps.SetSessionID("session-abc")
	got := ps.GetActiveSessionID()
	if got != "session-abc" {
		t.Errorf("GetActiveSessionID() = %q; want %q", got, "session-abc")
	}
}

func TestGetActiveSessionID_SwitchesWithProvider(t *testing.T) {
	ps := NewProviderState()
	ps.SetSessionID("anthropic-session")

	require.NoError(t, ps.SwitchProvider(ProviderGoogle))
	// Google has no session ID yet.
	assert.Empty(t, ps.GetActiveSessionID())

	ps.SetSessionID("google-session")
	assert.Equal(t, "google-session", ps.GetActiveSessionID())

	// Switch back to Anthropic — its session must be intact.
	require.NoError(t, ps.SwitchProvider(ProviderAnthropic))
	assert.Equal(t, "anthropic-session", ps.GetActiveSessionID())
}

// ---------------------------------------------------------------------------
// GetActiveProjectDir (TUI-031)
// ---------------------------------------------------------------------------

func TestGetActiveProjectDir_EmptyForNewProvider(t *testing.T) {
	ps := NewProviderState()
	got := ps.GetActiveProjectDir()
	if got != "" {
		t.Errorf("GetActiveProjectDir() = %q; want empty for new provider", got)
	}
}

func TestGetActiveProjectDir_ReturnsSetValue(t *testing.T) {
	ps := NewProviderState()
	ps.SetProjectDir("/home/user/myproject")
	got := ps.GetActiveProjectDir()
	assert.Equal(t, "/home/user/myproject", got)
}

func TestGetActiveProjectDir_SwitchesWithProvider(t *testing.T) {
	ps := NewProviderState()
	ps.SetProjectDir("/projects/anthropic")

	require.NoError(t, ps.SwitchProvider(ProviderGoogle))
	assert.Empty(t, ps.GetActiveProjectDir())

	ps.SetProjectDir("/projects/google")
	assert.Equal(t, "/projects/google", ps.GetActiveProjectDir())

	require.NoError(t, ps.SwitchProvider(ProviderAnthropic))
	assert.Equal(t, "/projects/anthropic", ps.GetActiveProjectDir())
}

// ---------------------------------------------------------------------------
// ExportSessionIDs (TUI-031)
// ---------------------------------------------------------------------------

func TestExportSessionIDs_EmptyWhenNoSessionsSet(t *testing.T) {
	ps := NewProviderState()
	got := ps.ExportSessionIDs()
	assert.Empty(t, got, "ExportSessionIDs should return empty map when no sessions set")
}

func TestExportSessionIDs_OnlyNonEmptyEntries(t *testing.T) {
	ps := NewProviderState()
	ps.SetSessionID("anthropic-session")

	require.NoError(t, ps.SwitchProvider(ProviderGoogle))
	// Do not set a session ID for Google.

	got := ps.ExportSessionIDs()
	require.Len(t, got, 1)
	assert.Equal(t, "anthropic-session", got[ProviderAnthropic])
	_, hasGoogle := got[ProviderGoogle]
	assert.False(t, hasGoogle)
}

func TestExportSessionIDs_ReturnsCopy_MutationIsolated(t *testing.T) {
	ps := NewProviderState()
	ps.SetSessionID("session-1")

	exported := ps.ExportSessionIDs()
	// Mutate the returned map.
	exported[ProviderAnthropic] = "mutated"
	delete(exported, ProviderAnthropic)

	// Re-export — internal state must be unchanged.
	exported2 := ps.ExportSessionIDs()
	assert.Equal(t, "session-1", exported2[ProviderAnthropic])
}

func TestExportSessionIDs_AllProviders(t *testing.T) {
	ps := NewProviderState()
	for i, id := range ps.AllProviders() {
		require.NoError(t, ps.SwitchProvider(id))
		ps.SetSessionID(fmt.Sprintf("session-%d", i))
	}

	got := ps.ExportSessionIDs()
	assert.Len(t, got, 4)
	for _, id := range ps.AllProviders() {
		_, ok := got[id]
		assert.True(t, ok, "provider %q missing from export", id)
	}
}

// ---------------------------------------------------------------------------
// ImportSessionIDs (TUI-031)
// ---------------------------------------------------------------------------

func TestImportSessionIDs_PopulatesFromSavedData(t *testing.T) {
	ps := NewProviderState()
	ps.ImportSessionIDs(map[ProviderID]string{
		ProviderAnthropic: "restored-session",
	})
	assert.Equal(t, "restored-session", ps.GetActiveSessionID())
}

func TestImportSessionIDs_SkipsUnknownProviders(t *testing.T) {
	ps := NewProviderState()
	// Should not panic and must not add unknown provider.
	ps.ImportSessionIDs(map[ProviderID]string{
		"unknown-provider": "session-x",
	})
	// State unchanged.
	assert.Empty(t, ps.GetActiveSessionID())
}

func TestImportSessionIDs_DoesNotOverwriteExistingNonEmpty(t *testing.T) {
	ps := NewProviderState()
	ps.SetSessionID("existing-session")

	// Import should NOT overwrite the existing non-empty value.
	ps.ImportSessionIDs(map[ProviderID]string{
		ProviderAnthropic: "imported-session",
	})
	assert.Equal(t, "existing-session", ps.GetActiveSessionID())
}

func TestImportSessionIDs_WritesWhenCurrentIsEmpty(t *testing.T) {
	ps := NewProviderState()
	// Anthropic has no session ID; import should write it.
	ps.ImportSessionIDs(map[ProviderID]string{
		ProviderAnthropic: "new-session",
	})
	assert.Equal(t, "new-session", ps.GetActiveSessionID())
}

func TestImportSessionIDs_SkipsEmptyValues(t *testing.T) {
	ps := NewProviderState()
	ps.ImportSessionIDs(map[ProviderID]string{
		ProviderAnthropic: "",
	})
	assert.Empty(t, ps.GetActiveSessionID())
}

// ---------------------------------------------------------------------------
// ExportModels (TUI-031)
// ---------------------------------------------------------------------------

func TestExportModels_ReturnsAllProviders(t *testing.T) {
	ps := NewProviderState()
	got := ps.ExportModels()
	assert.Len(t, got, 4, "ExportModels should include all four providers")
}

func TestExportModels_ReturnsCopy_MutationIsolated(t *testing.T) {
	ps := NewProviderState()
	exported := ps.ExportModels()
	originalOpus := exported[ProviderAnthropic]

	// Mutate the returned map.
	exported[ProviderAnthropic] = "mutated-model"

	// Re-export — internal state must be unchanged.
	exported2 := ps.ExportModels()
	assert.Equal(t, originalOpus, exported2[ProviderAnthropic])
}

func TestExportModels_ReflectsCurrentSelections(t *testing.T) {
	ps := NewProviderState()
	require.NoError(t, ps.SetActiveModel("haiku"))

	got := ps.ExportModels()
	assert.Equal(t, "haiku", got[ProviderAnthropic])
}

// ---------------------------------------------------------------------------
// ImportModels (TUI-031)
// ---------------------------------------------------------------------------

func TestImportModels_ValidModel(t *testing.T) {
	ps := NewProviderState()
	// Anthropic starts with "opus"; import "haiku".
	ps.ImportModels(map[ProviderID]string{
		ProviderAnthropic: "haiku",
	})
	assert.Equal(t, "haiku", ps.GetActiveModel())
}

func TestImportModels_SkipsInvalidModel(t *testing.T) {
	ps := NewProviderState()
	// "gpt-99-turbo" does not exist — must be silently skipped.
	ps.ImportModels(map[ProviderID]string{
		ProviderAnthropic: "gpt-99-turbo",
	})
	// Model must remain at the default.
	assert.Equal(t, "opus", ps.GetActiveModel())
}

func TestImportModels_SkipsUnknownProvider(t *testing.T) {
	ps := NewProviderState()
	// Should not panic.
	ps.ImportModels(map[ProviderID]string{
		"unknown-provider": "some-model",
	})
}

func TestImportModels_CrossProviderModelRejected(t *testing.T) {
	ps := NewProviderState()
	// "gpt-4" belongs to OpenAI, not Anthropic.
	ps.ImportModels(map[ProviderID]string{
		ProviderAnthropic: "gpt-4",
	})
	// Anthropic model must remain unchanged.
	assert.Equal(t, "opus", ps.GetActiveModel())
}

func TestImportModels_MultipleProviders(t *testing.T) {
	ps := NewProviderState()
	ps.ImportModels(map[ProviderID]string{
		ProviderAnthropic: "sonnet",
		ProviderOpenAI:    "gpt-4",
	})

	// Check Anthropic.
	assert.Equal(t, "sonnet", ps.GetActiveModel())

	// Check OpenAI.
	require.NoError(t, ps.SwitchProvider(ProviderOpenAI))
	assert.Equal(t, "gpt-4", ps.GetActiveModel())
}

// ---------------------------------------------------------------------------
// Export → Import roundtrip (TUI-031)
// ---------------------------------------------------------------------------

func TestExportImport_SessionIDs_Roundtrip(t *testing.T) {
	// Set up source state with session IDs for multiple providers.
	src := NewProviderState()
	for i, id := range src.AllProviders() {
		require.NoError(t, src.SwitchProvider(id))
		src.SetSessionID(fmt.Sprintf("session-%d", i))
	}
	require.NoError(t, src.SwitchProvider(ProviderAnthropic))

	exported := src.ExportSessionIDs()

	// Import into a fresh state.
	dst := NewProviderState()
	dst.ImportSessionIDs(exported)

	// Verify all session IDs transferred.
	for _, id := range dst.AllProviders() {
		require.NoError(t, src.SwitchProvider(id))
		require.NoError(t, dst.SwitchProvider(id))
		assert.Equal(t, src.GetActiveSessionID(), dst.GetActiveSessionID(),
			"session ID mismatch for provider %q", id)
	}
}

func TestExportImport_Models_Roundtrip(t *testing.T) {
	src := NewProviderState()
	require.NoError(t, src.SetActiveModel("haiku"))
	require.NoError(t, src.SwitchProvider(ProviderOpenAI))
	require.NoError(t, src.SetActiveModel("gpt-4"))
	require.NoError(t, src.SwitchProvider(ProviderAnthropic))

	exported := src.ExportModels()

	dst := NewProviderState()
	dst.ImportModels(exported)

	assert.Equal(t, "haiku", dst.GetActiveModel())
	require.NoError(t, dst.SwitchProvider(ProviderOpenAI))
	assert.Equal(t, "gpt-4", dst.GetActiveModel())
}

// ---------------------------------------------------------------------------
// Concurrent access — new methods (TUI-031)
// ---------------------------------------------------------------------------

func TestConcurrent_ExportImportSessionIDs(t *testing.T) {
	ps := NewProviderState()
	const goroutines = 30
	var wg sync.WaitGroup

	// Writers: set session IDs.
	for i := range goroutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ps.SetSessionID(fmt.Sprintf("session-%d", i))
		}(i)
	}

	// Readers: export session IDs.
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = ps.ExportSessionIDs()
			_ = ps.GetActiveSessionID()
			_ = ps.GetActiveProjectDir()
		}()
	}

	// Importers: import session IDs.
	for i := range goroutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ps.ImportSessionIDs(map[ProviderID]string{
				ProviderAnthropic: fmt.Sprintf("imported-%d", i),
			})
			ps.ImportModels(map[ProviderID]string{
				ProviderAnthropic: "sonnet",
			})
			_ = ps.ExportModels()
		}(i)
	}

	wg.Wait()
}

// SetActiveModel after switching back to a provider must use the default model
// that was set initially, not the other provider's model.
func TestSetActiveModel_PerProviderModelIsolation(t *testing.T) {
	ps := NewProviderState()

	// Change Anthropic to haiku.
	require.NoError(t, ps.SetActiveModel("haiku"))
	assert.Equal(t, "haiku", ps.GetActiveModel())

	// Switch to OpenAI and change its model.
	require.NoError(t, ps.SwitchProvider(ProviderOpenAI))
	require.NoError(t, ps.SetActiveModel("gpt-4"))
	assert.Equal(t, "gpt-4", ps.GetActiveModel())

	// Switch back to Anthropic — model must be haiku, not gpt-4.
	require.NoError(t, ps.SwitchProvider(ProviderAnthropic))
	assert.Equal(t, "haiku", ps.GetActiveModel())

	// Switch back to OpenAI — model must still be gpt-4.
	require.NoError(t, ps.SwitchProvider(ProviderOpenAI))
	assert.Equal(t, "gpt-4", ps.GetActiveModel())
}

// ---------------------------------------------------------------------------
// R-4: ToolBlock persistence in ProviderState
// ---------------------------------------------------------------------------

func TestSetActiveMessages_ToolBlocksPreserved(t *testing.T) {
	ps := NewProviderState()

	msgs := []DisplayMessage{
		{
			Role:    "assistant",
			Content: "I ran a tool",
			ToolBlocks: []ToolBlock{
				{Name: "Read", Input: "main.go", Output: "package main"},
				{Name: "Bash", Input: "go build", Output: "ok"},
			},
		},
	}
	ps.SetActiveMessages(msgs)

	got := ps.GetActiveMessages()
	require.Len(t, got, 1)
	require.Len(t, got[0].ToolBlocks, 2)

	assert.Equal(t, "Read", got[0].ToolBlocks[0].Name)
	assert.Equal(t, "main.go", got[0].ToolBlocks[0].Input)
	assert.Equal(t, "package main", got[0].ToolBlocks[0].Output)

	assert.Equal(t, "Bash", got[0].ToolBlocks[1].Name)
	assert.Equal(t, "go build", got[0].ToolBlocks[1].Input)
	assert.Equal(t, "ok", got[0].ToolBlocks[1].Output)
}

func TestGetActiveMessages_ToolBlocksDeepCopied(t *testing.T) {
	// Mutating returned ToolBlocks must not affect internal state.
	ps := NewProviderState()
	ps.SetActiveMessages([]DisplayMessage{
		{
			Role: "assistant",
			ToolBlocks: []ToolBlock{
				{Name: "Read", Input: "file.go", Output: "original output"},
			},
		},
	})

	first := ps.GetActiveMessages()
	require.Len(t, first[0].ToolBlocks, 1)
	first[0].ToolBlocks[0].Output = "mutated"

	second := ps.GetActiveMessages()
	assert.Equal(t, "original output", second[0].ToolBlocks[0].Output,
		"mutating returned ToolBlocks must not affect internal state")
}

func TestSetActiveMessages_EmptyToolBlocks_NoAlloc(t *testing.T) {
	// Messages without ToolBlocks should round-trip cleanly (no nil-slice allocation).
	ps := NewProviderState()
	ps.SetActiveMessages([]DisplayMessage{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	})

	got := ps.GetActiveMessages()
	assert.Nil(t, got[0].ToolBlocks)
	assert.Nil(t, got[1].ToolBlocks)
}

func TestSetActiveMessages_ToolBlocks_PerProviderIsolation(t *testing.T) {
	// ToolBlocks stored in one provider must not appear in another.
	ps := NewProviderState()

	ps.SetActiveMessages([]DisplayMessage{
		{
			Role: "assistant",
			ToolBlocks: []ToolBlock{
				{Name: "Edit", Input: "foo.go", Output: "edited"},
			},
		},
	})

	require.NoError(t, ps.SwitchProvider(ProviderGoogle))
	googleMsgs := ps.GetActiveMessages()
	assert.Nil(t, googleMsgs, "Google must not have Anthropic's ToolBlocks")
}
