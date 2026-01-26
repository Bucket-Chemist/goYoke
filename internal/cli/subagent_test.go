package cli

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestSubagentManager_RegisterUnregister tests the registration lifecycle
func TestSubagentManager_RegisterUnregister(t *testing.T) {
	t.Run("register new agent succeeds", func(t *testing.T) {
		mgr := NewSubagentManager(Config{})
		cfg := SubagentConfig{
			Name:        "test-agent",
			Description: "Test agent",
			Tier:        "haiku",
		}

		err := mgr.Register(cfg)
		if err != nil {
			t.Errorf("Register() error = %v, want nil", err)
		}

		// Verify it's in registry
		got, exists := mgr.Get("test-agent")
		if !exists {
			t.Error("Get() agent not found after Register()")
		}
		if got.Name != cfg.Name {
			t.Errorf("Get() name = %q, want %q", got.Name, cfg.Name)
		}
	})

	t.Run("register duplicate agent returns error", func(t *testing.T) {
		mgr := NewSubagentManager(Config{})
		cfg := SubagentConfig{
			Name:        "duplicate",
			Description: "First",
		}

		// Register once
		if err := mgr.Register(cfg); err != nil {
			t.Fatalf("Register() first call error = %v", err)
		}

		// Try to register again
		cfg2 := SubagentConfig{
			Name:        "duplicate",
			Description: "Second",
		}
		err := mgr.Register(cfg2)
		if err == nil {
			t.Error("Register() duplicate should return error, got nil")
		}
		if err.Error() != `agent "duplicate" already registered` {
			t.Errorf("Register() error = %v, want duplicate error", err)
		}
	})

	t.Run("register with empty name returns error", func(t *testing.T) {
		mgr := NewSubagentManager(Config{})
		cfg := SubagentConfig{
			Name:        "",
			Description: "Empty name",
		}

		err := mgr.Register(cfg)
		if err == nil {
			t.Error("Register() with empty name should return error, got nil")
		}
		if err.Error() != "agent name cannot be empty" {
			t.Errorf("Register() error = %v, want empty name error", err)
		}
	})

	t.Run("unregister existing agent succeeds", func(t *testing.T) {
		mgr := NewSubagentManager(Config{})
		cfg := SubagentConfig{Name: "to-remove"}

		mgr.Register(cfg)
		err := mgr.Unregister("to-remove")
		if err != nil {
			t.Errorf("Unregister() error = %v, want nil", err)
		}

		// Verify it's gone
		_, exists := mgr.Get("to-remove")
		if exists {
			t.Error("Get() agent still exists after Unregister()")
		}
	})

	t.Run("unregister non-existent agent returns error", func(t *testing.T) {
		mgr := NewSubagentManager(Config{})
		err := mgr.Unregister("does-not-exist")
		if err == nil {
			t.Error("Unregister() non-existent should return error, got nil")
		}
	})
}

// TestSubagentManager_GetList tests retrieval operations
func TestSubagentManager_GetList(t *testing.T) {
	t.Run("get returns correct config", func(t *testing.T) {
		mgr := NewSubagentManager(Config{})
		cfg := SubagentConfig{
			Name:        "getter",
			Description: "Get test",
			Tier:        "sonnet",
			MaxTurns:    5,
		}
		mgr.Register(cfg)

		got, exists := mgr.Get("getter")
		if !exists {
			t.Fatal("Get() returned false, want true")
		}
		if got.Description != cfg.Description {
			t.Errorf("Get() description = %q, want %q", got.Description, cfg.Description)
		}
		if got.Tier != cfg.Tier {
			t.Errorf("Get() tier = %q, want %q", got.Tier, cfg.Tier)
		}
		if got.MaxTurns != cfg.MaxTurns {
			t.Errorf("Get() max_turns = %d, want %d", got.MaxTurns, cfg.MaxTurns)
		}
	})

	t.Run("get non-existent returns false", func(t *testing.T) {
		mgr := NewSubagentManager(Config{})
		_, exists := mgr.Get("missing")
		if exists {
			t.Error("Get() non-existent returned true, want false")
		}
	})

	t.Run("list returns all registered agents", func(t *testing.T) {
		mgr := NewSubagentManager(Config{})
		agents := []SubagentConfig{
			{Name: "agent1", Description: "First"},
			{Name: "agent2", Description: "Second"},
			{Name: "agent3", Description: "Third"},
		}

		for _, cfg := range agents {
			mgr.Register(cfg)
		}

		list := mgr.List()
		if len(list) != len(agents) {
			t.Errorf("List() returned %d agents, want %d", len(list), len(agents))
		}

		// Verify all agents are in the list (order doesn't matter)
		found := make(map[string]bool)
		for _, cfg := range list {
			found[cfg.Name] = true
		}
		for _, expected := range agents {
			if !found[expected.Name] {
				t.Errorf("List() missing agent %q", expected.Name)
			}
		}
	})

	t.Run("list empty returns empty slice", func(t *testing.T) {
		mgr := NewSubagentManager(Config{})
		list := mgr.List()
		if len(list) != 0 {
			t.Errorf("List() on empty manager returned %d agents, want 0", len(list))
		}
	})
}

// TestSubagentManager_Spawn tests process creation
func TestSubagentManager_Spawn(t *testing.T) {
	t.Run("spawn unknown agent returns error", func(t *testing.T) {
		mgr := NewSubagentManager(Config{ClaudePath: "claude"})
		ctx := context.Background()

		_, err := mgr.Spawn(ctx, "unknown-agent")
		if err == nil {
			t.Error("Spawn() unknown agent should return error, got nil")
		}
		if err.Error() != `unknown agent: "unknown-agent"` {
			t.Errorf("Spawn() error = %v, want unknown agent error", err)
		}
	})

	t.Run("spawn returns existing if already running", func(t *testing.T) {
		// This test would require mocking ClaudeProcess or a test harness
		// Skip for now as it requires subprocess management
		t.Skip("Requires subprocess mocking infrastructure")
	})

	t.Run("spawn creates new process", func(t *testing.T) {
		// This test would require mocking ClaudeProcess or a test harness
		t.Skip("Requires subprocess mocking infrastructure")
	})
}

// TestSubagentManager_Query tests sending prompts to agents
func TestSubagentManager_Query(t *testing.T) {
	t.Run("query non-running agent returns error", func(t *testing.T) {
		mgr := NewSubagentManager(Config{})
		cfg := SubagentConfig{Name: "not-running"}
		mgr.Register(cfg)

		_, err := mgr.Query("not-running", "test prompt")
		if err == nil {
			t.Error("Query() non-running agent should return error, got nil")
		}
		if err.Error() != `agent "not-running" not running` {
			t.Errorf("Query() error = %v, want not running error", err)
		}
	})

	t.Run("query non-existent agent returns error", func(t *testing.T) {
		mgr := NewSubagentManager(Config{})
		_, err := mgr.Query("does-not-exist", "test prompt")
		if err == nil {
			t.Error("Query() non-existent agent should return error, got nil")
		}
	})
}

// TestSubagentManager_Stop tests process termination
func TestSubagentManager_Stop(t *testing.T) {
	t.Run("stop non-existent agent is no-op", func(t *testing.T) {
		mgr := NewSubagentManager(Config{})
		err := mgr.Stop("does-not-exist")
		if err != nil {
			t.Errorf("Stop() non-existent should return nil, got %v", err)
		}
	})

	t.Run("stop removes agent from sessions", func(t *testing.T) {
		// This test would require mocking ClaudeProcess
		t.Skip("Requires subprocess mocking infrastructure")
	})
}

// TestSubagentManager_StopAll tests bulk termination
func TestSubagentManager_StopAll(t *testing.T) {
	t.Run("stopall on empty manager is no-op", func(t *testing.T) {
		mgr := NewSubagentManager(Config{})
		mgr.StopAll() // Should not panic
	})

	t.Run("stopall terminates all running agents", func(t *testing.T) {
		// This test would require mocking ClaudeProcess
		t.Skip("Requires subprocess mocking infrastructure")
	})
}

// TestSubagentManager_IsRunning tests status checks
func TestSubagentManager_IsRunning(t *testing.T) {
	t.Run("is_running non-existent returns false", func(t *testing.T) {
		mgr := NewSubagentManager(Config{})
		if mgr.IsRunning("does-not-exist") {
			t.Error("IsRunning() non-existent returned true, want false")
		}
	})

	t.Run("is_running registered but not spawned returns false", func(t *testing.T) {
		mgr := NewSubagentManager(Config{})
		cfg := SubagentConfig{Name: "registered-only"}
		mgr.Register(cfg)

		if mgr.IsRunning("registered-only") {
			t.Error("IsRunning() registered-only returned true, want false")
		}
	})
}

// TestSubagentManager_RunningAgents tests listing running agents
func TestSubagentManager_RunningAgents(t *testing.T) {
	t.Run("running_agents on empty returns empty slice", func(t *testing.T) {
		mgr := NewSubagentManager(Config{})
		running := mgr.RunningAgents()
		if len(running) != 0 {
			t.Errorf("RunningAgents() returned %d, want 0", len(running))
		}
	})

	t.Run("running_agents returns only running processes", func(t *testing.T) {
		// This test would require mocking ClaudeProcess
		t.Skip("Requires subprocess mocking infrastructure")
	})
}

// TestSubagentManager_ConcurrentAccess tests thread safety
func TestSubagentManager_ConcurrentAccess(t *testing.T) {
	t.Run("concurrent register operations", func(t *testing.T) {
		mgr := NewSubagentManager(Config{})
		var wg sync.WaitGroup
		numGoroutines := 10

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				cfg := SubagentConfig{
					Name:        fmt.Sprintf("agent-%d", id),
					Description: fmt.Sprintf("Agent %d", id),
				}
				if err := mgr.Register(cfg); err != nil {
					t.Errorf("concurrent Register() failed: %v", err)
				}
			}(i)
		}

		wg.Wait()

		// Verify all agents were registered
		list := mgr.List()
		if len(list) != numGoroutines {
			t.Errorf("concurrent Register() resulted in %d agents, want %d", len(list), numGoroutines)
		}
	})

	t.Run("concurrent read operations", func(t *testing.T) {
		mgr := NewSubagentManager(Config{})
		agents := []SubagentConfig{
			{Name: "reader-1", Description: "First"},
			{Name: "reader-2", Description: "Second"},
			{Name: "reader-3", Description: "Third"},
		}

		for _, cfg := range agents {
			mgr.Register(cfg)
		}

		var wg sync.WaitGroup
		numReaders := 20

		for i := 0; i < numReaders; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				// Read operations should not panic or race
				mgr.List()
				mgr.Get("reader-1")
				mgr.IsRunning("reader-2")
				mgr.RunningAgents()
			}()
		}

		wg.Wait()
	})

	t.Run("concurrent mixed operations", func(t *testing.T) {
		mgr := NewSubagentManager(Config{})
		var wg sync.WaitGroup

		// Writer goroutine
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 5; i++ {
				cfg := SubagentConfig{Name: fmt.Sprintf("mixed-%d", i)}
				mgr.Register(cfg)
				time.Sleep(1 * time.Millisecond)
			}
		}()

		// Reader goroutines
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					mgr.List()
					time.Sleep(1 * time.Millisecond)
				}
			}()
		}

		wg.Wait()
	})
}

// TestSubagentManager_UnregisterRunningAgent tests cleanup behavior
func TestSubagentManager_UnregisterRunningAgent(t *testing.T) {
	t.Run("unregister stops running agent", func(t *testing.T) {
		// This test would require mocking ClaudeProcess to verify Stop() is called
		t.Skip("Requires subprocess mocking infrastructure")
	})
}

// TestSubagentManager_BuildConfig tests configuration building
func TestSubagentManager_BuildConfig(t *testing.T) {
	t.Run("buildconfig inherits base config", func(t *testing.T) {
		baseCfg := Config{
			ClaudePath: "/custom/path/claude",
			WorkingDir: "/custom/workdir",
			Verbose:    true,
			NoHooks:    true,
		}
		mgr := NewSubagentManager(baseCfg)

		agentCfg := SubagentConfig{
			Name:         "test",
			SystemPrompt: "Custom prompt",
			Model:        "haiku",
		}

		procCfg := mgr.buildConfig(agentCfg)

		// Verify base config is inherited
		if procCfg.ClaudePath != baseCfg.ClaudePath {
			t.Errorf("buildConfig() ClaudePath = %q, want %q", procCfg.ClaudePath, baseCfg.ClaudePath)
		}
		if procCfg.WorkingDir != baseCfg.WorkingDir {
			t.Errorf("buildConfig() WorkingDir = %q, want %q", procCfg.WorkingDir, baseCfg.WorkingDir)
		}
		if procCfg.Verbose != baseCfg.Verbose {
			t.Errorf("buildConfig() Verbose = %v, want %v", procCfg.Verbose, baseCfg.Verbose)
		}
		if procCfg.NoHooks != baseCfg.NoHooks {
			t.Errorf("buildConfig() NoHooks = %v, want %v", procCfg.NoHooks, baseCfg.NoHooks)
		}

		// Note: Agent-specific overrides will be tested once Config
		// has fields for SystemPrompt, Model, etc. (Task #8)
	})
}

// TestSubagentManager_EdgeCases tests edge case handling
func TestSubagentManager_EdgeCases(t *testing.T) {
	t.Run("register agent with all fields populated", func(t *testing.T) {
		mgr := NewSubagentManager(Config{})
		cfg := SubagentConfig{
			Name:            "full-config",
			Description:     "Agent with all fields",
			SystemPrompt:    "You are a specialist",
			AppendPrompt:    "Additional instructions",
			AllowedTools:    []string{"Read", "Write"},
			DisallowedTools: []string{"Bash"},
			Model:           "sonnet",
			MaxTurns:        10,
			Tier:            "sonnet",
		}

		if err := mgr.Register(cfg); err != nil {
			t.Fatalf("Register() full config error = %v", err)
		}

		got, exists := mgr.Get("full-config")
		if !exists {
			t.Fatal("Get() full config not found")
		}

		// Verify all fields are preserved
		if got.Description != cfg.Description {
			t.Errorf("Description = %q, want %q", got.Description, cfg.Description)
		}
		if got.SystemPrompt != cfg.SystemPrompt {
			t.Errorf("SystemPrompt = %q, want %q", got.SystemPrompt, cfg.SystemPrompt)
		}
		if got.AppendPrompt != cfg.AppendPrompt {
			t.Errorf("AppendPrompt = %q, want %q", got.AppendPrompt, cfg.AppendPrompt)
		}
		if len(got.AllowedTools) != len(cfg.AllowedTools) {
			t.Errorf("AllowedTools length = %d, want %d", len(got.AllowedTools), len(cfg.AllowedTools))
		}
		if len(got.DisallowedTools) != len(cfg.DisallowedTools) {
			t.Errorf("DisallowedTools length = %d, want %d", len(got.DisallowedTools), len(cfg.DisallowedTools))
		}
		if got.Model != cfg.Model {
			t.Errorf("Model = %q, want %q", got.Model, cfg.Model)
		}
		if got.MaxTurns != cfg.MaxTurns {
			t.Errorf("MaxTurns = %d, want %d", got.MaxTurns, cfg.MaxTurns)
		}
		if got.Tier != cfg.Tier {
			t.Errorf("Tier = %q, want %q", got.Tier, cfg.Tier)
		}
	})

	t.Run("register agent with minimal config", func(t *testing.T) {
		mgr := NewSubagentManager(Config{})
		cfg := SubagentConfig{
			Name: "minimal",
		}

		if err := mgr.Register(cfg); err != nil {
			t.Fatalf("Register() minimal config error = %v", err)
		}

		got, exists := mgr.Get("minimal")
		if !exists {
			t.Fatal("Get() minimal config not found")
		}
		if got.Name != "minimal" {
			t.Errorf("Name = %q, want %q", got.Name, "minimal")
		}
	})

	t.Run("list preserves agent order independence", func(t *testing.T) {
		mgr := NewSubagentManager(Config{})
		names := []string{"zebra", "alpha", "beta", "gamma"}

		for _, name := range names {
			mgr.Register(SubagentConfig{Name: name})
		}

		// Call List multiple times, verify all agents present
		for i := 0; i < 3; i++ {
			list := mgr.List()
			if len(list) != len(names) {
				t.Errorf("List() iteration %d returned %d agents, want %d", i, len(list), len(names))
			}

			found := make(map[string]bool)
			for _, cfg := range list {
				found[cfg.Name] = true
			}
			for _, expected := range names {
				if !found[expected] {
					t.Errorf("List() iteration %d missing agent %q", i, expected)
				}
			}
		}
	})
}

// TestSubagentManager_NewInstance tests manager creation
func TestSubagentManager_NewInstance(t *testing.T) {
	t.Run("new manager with empty config", func(t *testing.T) {
		mgr := NewSubagentManager(Config{})
		if mgr == nil {
			t.Fatal("NewSubagentManager() returned nil")
		}
		if len(mgr.List()) != 0 {
			t.Error("new manager should have empty registry")
		}
		if len(mgr.RunningAgents()) != 0 {
			t.Error("new manager should have no running agents")
		}
	})

	t.Run("new manager with base config", func(t *testing.T) {
		baseCfg := Config{
			ClaudePath: "/test/claude",
			WorkingDir: "/test/dir",
			Verbose:    true,
		}
		mgr := NewSubagentManager(baseCfg)
		if mgr == nil {
			t.Fatal("NewSubagentManager() returned nil")
		}

		// Base config is internal, but we can verify it's used in buildConfig
		testAgent := SubagentConfig{Name: "test"}
		procCfg := mgr.buildConfig(testAgent)
		if procCfg.ClaudePath != baseCfg.ClaudePath {
			t.Errorf("base config not applied: ClaudePath = %q, want %q", procCfg.ClaudePath, baseCfg.ClaudePath)
		}
	})
}

// TestPresetAgents tests all preset agent factory functions
func TestPresetAgents(t *testing.T) {
	for name, factory := range PresetAgents {
		t.Run(name, func(t *testing.T) {
			cfg := factory()

			// Verify name matches
			if cfg.Name != name {
				t.Errorf("factory name mismatch: got %q, want %q", cfg.Name, name)
			}

			// Verify description is populated
			if cfg.Description == "" {
				t.Error("Description is empty")
			}

			// Verify tier is populated
			if cfg.Tier == "" {
				t.Error("Tier is empty")
			}

			// Verify at least one prompt is set
			if cfg.SystemPrompt == "" && cfg.AppendPrompt == "" {
				t.Error("Both SystemPrompt and AppendPrompt are empty")
			}

			// Verify MaxTurns is positive
			if cfg.MaxTurns <= 0 {
				t.Errorf("MaxTurns = %d, want > 0", cfg.MaxTurns)
			}

			// Verify Model is set
			if cfg.Model == "" {
				t.Error("Model is empty")
			}
		})
	}
}

// TestPresetAgents_SpecificConfigurations tests specific preset configs
func TestPresetAgents_SpecificConfigurations(t *testing.T) {
	t.Run("security-reviewer has read-only tools", func(t *testing.T) {
		cfg := SecurityReviewerAgent()

		// Should have Read, Grep, Glob for analysis
		if len(cfg.AllowedTools) == 0 {
			t.Error("AllowedTools is empty")
		}

		// Should not have Write or Edit (read-only)
		for _, tool := range cfg.AllowedTools {
			if tool == "Write" || tool == "Edit" {
				t.Errorf("SecurityReviewerAgent should be read-only, found %s", tool)
			}
		}
	})

	t.Run("code-reviewer has read-only tools", func(t *testing.T) {
		cfg := CodeReviewerAgent()

		// Should be read-only
		for _, tool := range cfg.AllowedTools {
			if tool == "Write" || tool == "Edit" {
				t.Errorf("CodeReviewerAgent should be read-only, found %s", tool)
			}
		}
	})

	t.Run("test-analyst allows bash for test runs", func(t *testing.T) {
		cfg := TestAnalystAgent()

		// Should allow Bash for running tests
		hasBash := false
		for _, tool := range cfg.AllowedTools {
			if tool == "Bash(go test*)" {
				hasBash = true
				break
			}
		}
		if !hasBash {
			t.Error("TestAnalystAgent should allow 'Bash(go test*)'")
		}
	})

	t.Run("go-pro has write permissions", func(t *testing.T) {
		cfg := GoProAgent()

		// Should have Read, Write, Edit for implementation
		hasWrite := false
		hasEdit := false
		for _, tool := range cfg.AllowedTools {
			if tool == "Write" {
				hasWrite = true
			}
			if tool == "Edit" {
				hasEdit = true
			}
		}
		if !hasWrite {
			t.Error("GoProAgent should allow Write")
		}
		if !hasEdit {
			t.Error("GoProAgent should allow Edit")
		}
	})

	t.Run("go-tui has write permissions and go bash", func(t *testing.T) {
		cfg := GoTuiAgent()

		// Should have implementation tools
		hasWrite := false
		hasBash := false
		for _, tool := range cfg.AllowedTools {
			if tool == "Write" {
				hasWrite = true
			}
			if tool == "Bash(go *)" {
				hasBash = true
			}
		}
		if !hasWrite {
			t.Error("GoTuiAgent should allow Write")
		}
		if !hasBash {
			t.Error("GoTuiAgent should allow 'Bash(go *)'")
		}
	})

	t.Run("go-cli has appropriate MaxTurns", func(t *testing.T) {
		cfg := GoCliAgent()

		// CLI work is moderately complex, not as much as TUI
		if cfg.MaxTurns < 10 || cfg.MaxTurns > 20 {
			t.Errorf("GoCliAgent MaxTurns = %d, expected between 10 and 20", cfg.MaxTurns)
		}
	})

	t.Run("go-concurrent has high MaxTurns", func(t *testing.T) {
		cfg := GoConcurrentAgent()

		// Concurrency is complex, needs more turns
		if cfg.MaxTurns < 20 {
			t.Errorf("GoConcurrentAgent MaxTurns = %d, expected >= 20 for complex concurrency work", cfg.MaxTurns)
		}
	})
}

// TestPresetAgents_PromptContent tests that prompts contain expected keywords
func TestPresetAgents_PromptContent(t *testing.T) {
	tests := []struct {
		name     string
		factory  func() SubagentConfig
		keywords []string
	}{
		{
			name:     "security-reviewer",
			factory:  SecurityReviewerAgent,
			keywords: []string{"security", "OWASP", "injection"},
		},
		{
			name:     "code-reviewer",
			factory:  CodeReviewerAgent,
			keywords: []string{"review", "patterns", "maintainability"},
		},
		{
			name:     "test-analyst",
			factory:  TestAnalystAgent,
			keywords: []string{"test", "coverage", "edge case"},
		},
		{
			name:     "go-pro",
			factory:  GoProAgent,
			keywords: []string{"Go", "error handling", "interfaces"},
		},
		{
			name:     "go-tui",
			factory:  GoTuiAgent,
			keywords: []string{"Bubbletea", "tea.Model", "lipgloss"},
		},
		{
			name:     "go-cli",
			factory:  GoCliAgent,
			keywords: []string{"Cobra", "Flag", "subcommand"},
		},
		{
			name:     "go-concurrent",
			factory:  GoConcurrentAgent,
			keywords: []string{"goroutine", "channel", "errgroup"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.factory()
			prompt := cfg.SystemPrompt + cfg.AppendPrompt

			for _, keyword := range tt.keywords {
				// Case-insensitive search
				found := false
				for i := 0; i < len(prompt); i++ {
					if i+len(keyword) <= len(prompt) {
						substr := prompt[i : i+len(keyword)]
						if len(substr) == len(keyword) {
							match := true
							for j := 0; j < len(keyword); j++ {
								c1 := substr[j]
								c2 := keyword[j]
								// Simple case-insensitive comparison
								if c1 >= 'A' && c1 <= 'Z' {
									c1 = c1 + 32
								}
								if c2 >= 'A' && c2 <= 'Z' {
									c2 = c2 + 32
								}
								if c1 != c2 {
									match = false
									break
								}
							}
							if match {
								found = true
								break
							}
						}
					}
				}
				if !found {
					t.Errorf("prompt missing expected keyword %q", keyword)
				}
			}
		})
	}
}

// TestSubagentManager_RegisterPresets tests bulk preset registration
func TestSubagentManager_RegisterPresets(t *testing.T) {
	t.Run("register all presets succeeds", func(t *testing.T) {
		mgr := NewSubagentManager(Config{})

		err := mgr.RegisterPresets()
		if err != nil {
			t.Fatalf("RegisterPresets() error = %v, want nil", err)
		}

		// Verify all presets are registered
		list := mgr.List()
		if len(list) != len(PresetAgents) {
			t.Errorf("RegisterPresets() registered %d agents, want %d", len(list), len(PresetAgents))
		}

		// Verify each preset by name
		for name := range PresetAgents {
			cfg, exists := mgr.Get(name)
			if !exists {
				t.Errorf("preset %q not found after RegisterPresets()", name)
			}
			if cfg.Name != name {
				t.Errorf("preset %q has name %q", name, cfg.Name)
			}
		}
	})

	t.Run("register presets twice returns error", func(t *testing.T) {
		mgr := NewSubagentManager(Config{})

		// First registration
		if err := mgr.RegisterPresets(); err != nil {
			t.Fatalf("first RegisterPresets() error = %v", err)
		}

		// Second registration should fail
		err := mgr.RegisterPresets()
		if err == nil {
			t.Error("second RegisterPresets() should return error, got nil")
		}
	})

	t.Run("register presets with conflicting custom agent", func(t *testing.T) {
		mgr := NewSubagentManager(Config{})

		// Register a custom agent with the same name as a preset
		custom := SubagentConfig{
			Name:        "go-pro", // Conflicts with preset
			Description: "Custom agent",
		}
		if err := mgr.Register(custom); err != nil {
			t.Fatalf("Register() custom agent error = %v", err)
		}

		// RegisterPresets should fail
		err := mgr.RegisterPresets()
		if err == nil {
			t.Error("RegisterPresets() should fail when preset name conflicts, got nil")
		}
	})
}

// TestNewSubagentManagerWithPresets tests convenience constructor
func TestNewSubagentManagerWithPresets(t *testing.T) {
	t.Run("creates manager with presets", func(t *testing.T) {
		baseCfg := Config{
			ClaudePath: "/test/claude",
			WorkingDir: "/test/dir",
		}

		mgr, err := NewSubagentManagerWithPresets(baseCfg)
		if err != nil {
			t.Fatalf("NewSubagentManagerWithPresets() error = %v, want nil", err)
		}
		if mgr == nil {
			t.Fatal("NewSubagentManagerWithPresets() returned nil manager")
		}

		// Verify presets are registered
		list := mgr.List()
		if len(list) != len(PresetAgents) {
			t.Errorf("NewSubagentManagerWithPresets() registered %d agents, want %d", len(list), len(PresetAgents))
		}

		// Verify base config is preserved
		testAgent := SubagentConfig{Name: "test-verify-base"}
		procCfg := mgr.buildConfig(testAgent)
		if procCfg.ClaudePath != baseCfg.ClaudePath {
			t.Errorf("base config not preserved: ClaudePath = %q, want %q", procCfg.ClaudePath, baseCfg.ClaudePath)
		}
	})
}

// TestLoadAgentsFromFile tests loading custom agents from JSON
func TestLoadAgentsFromFile(t *testing.T) {
	t.Run("load valid JSON succeeds", func(t *testing.T) {
		// Create temporary file with valid JSON
		tmpFile := t.TempDir() + "/agents.json"
		jsonData := `[
			{
				"Name": "custom-agent-1",
				"Description": "First custom agent",
				"Tier": "haiku",
				"MaxTurns": 5
			},
			{
				"Name": "custom-agent-2",
				"Description": "Second custom agent",
				"Tier": "sonnet",
				"MaxTurns": 10
			}
		]`
		if err := os.WriteFile(tmpFile, []byte(jsonData), 0644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		// Load agents
		configs, err := LoadAgentsFromFile(tmpFile)
		if err != nil {
			t.Fatalf("LoadAgentsFromFile() error = %v, want nil", err)
		}

		// Verify count
		if len(configs) != 2 {
			t.Errorf("LoadAgentsFromFile() loaded %d agents, want 2", len(configs))
		}

		// Verify first agent
		if configs[0].Name != "custom-agent-1" {
			t.Errorf("configs[0].Name = %q, want %q", configs[0].Name, "custom-agent-1")
		}
		if configs[0].Description != "First custom agent" {
			t.Errorf("configs[0].Description = %q, want %q", configs[0].Description, "First custom agent")
		}
		if configs[0].MaxTurns != 5 {
			t.Errorf("configs[0].MaxTurns = %d, want 5", configs[0].MaxTurns)
		}

		// Verify second agent
		if configs[1].Name != "custom-agent-2" {
			t.Errorf("configs[1].Name = %q, want %q", configs[1].Name, "custom-agent-2")
		}
	})

	t.Run("load non-existent file returns error", func(t *testing.T) {
		_, err := LoadAgentsFromFile("/nonexistent/path/agents.json")
		if err == nil {
			t.Error("LoadAgentsFromFile() on non-existent file should return error, got nil")
		}
	})

	t.Run("load invalid JSON returns error", func(t *testing.T) {
		tmpFile := t.TempDir() + "/invalid.json"
		invalidJSON := `{"Name": "incomplete"`
		if err := os.WriteFile(tmpFile, []byte(invalidJSON), 0644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		_, err := LoadAgentsFromFile(tmpFile)
		if err == nil {
			t.Error("LoadAgentsFromFile() on invalid JSON should return error, got nil")
		}
	})

	t.Run("load empty array succeeds", func(t *testing.T) {
		tmpFile := t.TempDir() + "/empty.json"
		emptyJSON := `[]`
		if err := os.WriteFile(tmpFile, []byte(emptyJSON), 0644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		configs, err := LoadAgentsFromFile(tmpFile)
		if err != nil {
			t.Errorf("LoadAgentsFromFile() on empty array error = %v, want nil", err)
		}
		if len(configs) != 0 {
			t.Errorf("LoadAgentsFromFile() returned %d configs, want 0", len(configs))
		}
	})
}

// TestSubagentManager_BuildConfig_Complete tests full buildConfig implementation
func TestSubagentManager_BuildConfig_Complete(t *testing.T) {
	t.Run("buildconfig applies all agent overrides", func(t *testing.T) {
		baseCfg := Config{
			ClaudePath: "/base/claude",
			WorkingDir: "/base/workdir",
			Verbose:    false,
		}
		mgr := NewSubagentManager(baseCfg)

		agentCfg := SubagentConfig{
			Name:            "security-agent",
			SystemPrompt:    "You are a security expert",
			AllowedTools:    []string{"Read", "Grep"},
			DisallowedTools: []string{"Bash", "WebFetch"},
			Model:           "sonnet",
			MaxTurns:        5,
		}

		procCfg := mgr.buildConfig(agentCfg)

		// Verify base config is inherited
		assert.Equal(t, baseCfg.ClaudePath, procCfg.ClaudePath)
		assert.Equal(t, baseCfg.WorkingDir, procCfg.WorkingDir)
		assert.Equal(t, baseCfg.Verbose, procCfg.Verbose)

		// Verify agent overrides are applied
		assert.Equal(t, "You are a security expert", procCfg.SystemPrompt)
		assert.Equal(t, []string{"Read", "Grep"}, procCfg.AllowedTools)
		assert.Equal(t, []string{"Bash", "WebFetch"}, procCfg.DisallowedTools)
		assert.Equal(t, "sonnet", procCfg.Model)
		assert.Equal(t, 5, procCfg.MaxTurns)
	})

	t.Run("buildconfig inherits base when agent fields empty", func(t *testing.T) {
		baseCfg := Config{
			ClaudePath:   "/base/claude",
			SystemPrompt: "Base prompt",
			Model:        "haiku",
			MaxTurns:     10,
		}
		mgr := NewSubagentManager(baseCfg)

		agentCfg := SubagentConfig{
			Name: "minimal-agent",
			// No overrides
		}

		procCfg := mgr.buildConfig(agentCfg)

		// Should inherit base config
		assert.Equal(t, baseCfg.ClaudePath, procCfg.ClaudePath)
		assert.Equal(t, baseCfg.SystemPrompt, procCfg.SystemPrompt)
		assert.Equal(t, baseCfg.Model, procCfg.Model)
		assert.Equal(t, baseCfg.MaxTurns, procCfg.MaxTurns)
	})

	t.Run("buildconfig SystemPrompt clears AppendPrompt", func(t *testing.T) {
		baseCfg := Config{
			AppendPrompt: "Base append",
		}
		mgr := NewSubagentManager(baseCfg)

		agentCfg := SubagentConfig{
			Name:         "override-agent",
			SystemPrompt: "Override system",
		}

		procCfg := mgr.buildConfig(agentCfg)

		// SystemPrompt should be set, AppendPrompt should be cleared
		assert.Equal(t, "Override system", procCfg.SystemPrompt)
		assert.Equal(t, "", procCfg.AppendPrompt)
	})

	t.Run("buildconfig AppendPrompt when no SystemPrompt", func(t *testing.T) {
		baseCfg := Config{}
		mgr := NewSubagentManager(baseCfg)

		agentCfg := SubagentConfig{
			Name:         "append-agent",
			AppendPrompt: "Additional context",
		}

		procCfg := mgr.buildConfig(agentCfg)

		// AppendPrompt should be set, SystemPrompt should be empty
		assert.Equal(t, "", procCfg.SystemPrompt)
		assert.Equal(t, "Additional context", procCfg.AppendPrompt)
	})

	t.Run("buildconfig partial overrides", func(t *testing.T) {
		baseCfg := Config{
			ClaudePath: "/base/claude",
			Model:      "haiku",
			MaxTurns:   20,
		}
		mgr := NewSubagentManager(baseCfg)

		agentCfg := SubagentConfig{
			Name:     "partial-agent",
			Model:    "sonnet", // Override only model
			MaxTurns: 0,        // Don't override MaxTurns
		}

		procCfg := mgr.buildConfig(agentCfg)

		// Model should be overridden
		assert.Equal(t, "sonnet", procCfg.Model)
		// MaxTurns should inherit from base (0 means don't override)
		assert.Equal(t, 20, procCfg.MaxTurns)
		// ClaudePath should inherit
		assert.Equal(t, "/base/claude", procCfg.ClaudePath)
	})

	t.Run("buildconfig tool lists override completely", func(t *testing.T) {
		baseCfg := Config{
			AllowedTools:    []string{"Read", "Write"},
			DisallowedTools: []string{"Bash"},
		}
		mgr := NewSubagentManager(baseCfg)

		agentCfg := SubagentConfig{
			Name:            "tool-agent",
			AllowedTools:    []string{"Read", "Grep", "Edit"},
			DisallowedTools: []string{"WebFetch"},
		}

		procCfg := mgr.buildConfig(agentCfg)

		// Agent tool lists should completely replace base lists
		assert.Equal(t, []string{"Read", "Grep", "Edit"}, procCfg.AllowedTools)
		assert.Equal(t, []string{"WebFetch"}, procCfg.DisallowedTools)
	})

	t.Run("buildconfig empty tool lists inherit from base", func(t *testing.T) {
		baseCfg := Config{
			AllowedTools:    []string{"Read", "Write"},
			DisallowedTools: []string{"Bash"},
		}
		mgr := NewSubagentManager(baseCfg)

		agentCfg := SubagentConfig{
			Name: "no-tools-agent",
			// No tool overrides
		}

		procCfg := mgr.buildConfig(agentCfg)

		// Should inherit base tool lists
		assert.Equal(t, baseCfg.AllowedTools, procCfg.AllowedTools)
		assert.Equal(t, baseCfg.DisallowedTools, procCfg.DisallowedTools)
	})
}
