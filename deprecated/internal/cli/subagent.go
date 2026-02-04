package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// SubagentConfig defines a specialized agent's configuration
type SubagentConfig struct {
	// Name is the unique identifier for this agent (e.g., "security-reviewer")
	Name string

	// Description explains what this agent does (shown in TUI picker)
	Description string

	// SystemPrompt is the custom system prompt for this agent
	SystemPrompt string

	// AppendPrompt appends to the default system prompt instead of replacing
	AppendPrompt string

	// AllowedTools is the whitelist of permitted tools
	// Empty means all tools allowed
	AllowedTools []string

	// DisallowedTools is the blacklist of forbidden tools
	DisallowedTools []string

	// Model overrides the default model ("haiku", "sonnet", "opus")
	// Empty means inherit from parent
	Model string

	// MaxTurns limits the number of agentic turns
	// 0 means unlimited
	MaxTurns int

	// Tier is the routing tier for display purposes
	Tier string
}

// SubagentManager manages specialized agent instances
type SubagentManager struct {
	registry map[string]SubagentConfig
	sessions map[string]*ClaudeProcess
	mu       sync.RWMutex
	baseCfg  Config
}

// NewSubagentManager creates a new manager with the given base configuration
func NewSubagentManager(baseCfg Config) *SubagentManager {
	return &SubagentManager{
		registry: make(map[string]SubagentConfig),
		sessions: make(map[string]*ClaudeProcess),
		baseCfg:  baseCfg,
	}
}

// Register adds an agent configuration to the registry
// Returns error if an agent with the same name already exists
func (sm *SubagentManager) Register(cfg SubagentConfig) error {
	if cfg.Name == "" {
		return fmt.Errorf("agent name cannot be empty")
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.registry[cfg.Name]; exists {
		return fmt.Errorf("agent %q already registered", cfg.Name)
	}

	sm.registry[cfg.Name] = cfg
	return nil
}

// Unregister removes an agent from the registry
// Also stops the agent if running
func (sm *SubagentManager) Unregister(name string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.registry[name]; !exists {
		return fmt.Errorf("agent %q not registered", name)
	}

	// Stop if running
	if proc, running := sm.sessions[name]; running {
		proc.Stop()
		delete(sm.sessions, name)
	}

	delete(sm.registry, name)
	return nil
}

// Get returns the configuration for a registered agent
func (sm *SubagentManager) Get(name string) (SubagentConfig, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	cfg, exists := sm.registry[name]
	return cfg, exists
}

// List returns all registered agent configurations
func (sm *SubagentManager) List() []SubagentConfig {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make([]SubagentConfig, 0, len(sm.registry))
	for _, cfg := range sm.registry {
		result = append(result, cfg)
	}
	return result
}

// Spawn creates and starts a new agent process
// Returns existing process if agent is already running
func (sm *SubagentManager) Spawn(ctx context.Context, agentName string) (*ClaudeProcess, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	cfg, exists := sm.registry[agentName]
	if !exists {
		return nil, fmt.Errorf("unknown agent: %q", agentName)
	}

	// Return existing if running
	if proc, running := sm.sessions[agentName]; running && proc.IsRunning() {
		return proc, nil
	}

	// Build process config from base + agent overrides
	procCfg := sm.buildConfig(cfg)

	proc, err := NewClaudeProcess(procCfg)
	if err != nil {
		return nil, fmt.Errorf("create process for %s: %w", agentName, err)
	}

	if err := proc.Start(); err != nil {
		return nil, fmt.Errorf("start process for %s: %w", agentName, err)
	}

	sm.sessions[agentName] = proc
	return proc, nil
}

// buildConfig creates a Config from base config + agent overrides
func (sm *SubagentManager) buildConfig(agent SubagentConfig) Config {
	cfg := sm.baseCfg

	// Apply agent-specific overrides
	if agent.SystemPrompt != "" {
		cfg.SystemPrompt = agent.SystemPrompt
		cfg.AppendPrompt = "" // Clear append if system is set
	} else if agent.AppendPrompt != "" {
		cfg.AppendPrompt = agent.AppendPrompt
	}

	if len(agent.AllowedTools) > 0 {
		cfg.AllowedTools = agent.AllowedTools
	}
	if len(agent.DisallowedTools) > 0 {
		cfg.DisallowedTools = agent.DisallowedTools
	}

	if agent.MaxTurns > 0 {
		cfg.MaxTurns = agent.MaxTurns
	}

	if agent.Model != "" {
		cfg.Model = agent.Model
	}

	return cfg
}

// Query sends a prompt to a running agent
// Returns the events channel for receiving responses
func (sm *SubagentManager) Query(agentName, prompt string) (<-chan Event, error) {
	sm.mu.RLock()
	proc, exists := sm.sessions[agentName]
	sm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("agent %q not running", agentName)
	}

	if !proc.IsRunning() {
		return nil, fmt.Errorf("agent %q process stopped", agentName)
	}

	if err := proc.Send(prompt); err != nil {
		return nil, fmt.Errorf("send to %s: %w", agentName, err)
	}

	return proc.Events(), nil
}

// Stop terminates a running agent
func (sm *SubagentManager) Stop(agentName string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	proc, exists := sm.sessions[agentName]
	if !exists {
		return nil // Already stopped, not an error
	}

	delete(sm.sessions, agentName)
	return proc.Stop()
}

// StopAll terminates all running agents
func (sm *SubagentManager) StopAll() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for name, proc := range sm.sessions {
		proc.Stop()
		delete(sm.sessions, name)
	}
}

// IsRunning checks if an agent is currently running
func (sm *SubagentManager) IsRunning(agentName string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	proc, exists := sm.sessions[agentName]
	return exists && proc.IsRunning()
}

// RunningAgents returns the names of all currently running agents
func (sm *SubagentManager) RunningAgents() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make([]string, 0, len(sm.sessions))
	for name, proc := range sm.sessions {
		if proc.IsRunning() {
			result = append(result, name)
		}
	}
	return result
}

// PresetAgents provides factory functions for common agent types
var PresetAgents = map[string]func() SubagentConfig{
	"security-reviewer": SecurityReviewerAgent,
	"code-reviewer":     CodeReviewerAgent,
	"test-analyst":      TestAnalystAgent,
	"go-pro":            GoProAgent,
	"go-tui":            GoTuiAgent,
	"go-cli":            GoCliAgent,
	"go-concurrent":     GoConcurrentAgent,
}

// SecurityReviewerAgent returns configuration for security analysis
func SecurityReviewerAgent() SubagentConfig {
	return SubagentConfig{
		Name:        "security-reviewer",
		Description: "Analyzes code for security vulnerabilities (OWASP Top 10, injection, auth issues)",
		AppendPrompt: `You are a security expert. Focus on:
- OWASP Top 10 vulnerabilities
- Injection risks (SQL, command, XSS)
- Authentication and authorization issues
- Secrets and credential exposure
- Input validation gaps`,
		AllowedTools: []string{"Read", "Grep", "Glob"},
		Model:        "sonnet",
		MaxTurns:     10,
		Tier:         "sonnet",
	}
}

// CodeReviewerAgent returns configuration for code quality review
func CodeReviewerAgent() SubagentConfig {
	return SubagentConfig{
		Name:        "code-reviewer",
		Description: "Reviews code quality, patterns, and architecture",
		AppendPrompt: `You are a senior engineer conducting code review. Focus on:
- Readability and maintainability
- Design patterns and anti-patterns
- Error handling completeness
- Test coverage gaps
- Performance considerations`,
		AllowedTools: []string{"Read", "Grep", "Glob"},
		Model:        "sonnet",
		MaxTurns:     10,
		Tier:         "sonnet",
	}
}

// TestAnalystAgent returns configuration for test analysis
func TestAnalystAgent() SubagentConfig {
	return SubagentConfig{
		Name:        "test-analyst",
		Description: "Analyzes test coverage and suggests improvements",
		AppendPrompt: `You are a testing expert. Focus on:
- Test coverage gaps
- Edge case identification
- Test quality and assertions
- Mocking strategies
- Integration test opportunities`,
		AllowedTools: []string{"Read", "Grep", "Glob", "Bash(go test*)"},
		Model:        "sonnet",
		MaxTurns:     15,
		Tier:         "sonnet",
	}
}

// GoProAgent returns configuration for Go implementation
func GoProAgent() SubagentConfig {
	return SubagentConfig{
		Name:        "go-pro",
		Description: "Go implementation specialist (structs, interfaces, error handling)",
		AppendPrompt: `You are a Go expert. Follow Go idioms:
- Explicit error handling (no _ for errors)
- Small interfaces at point of use
- Accept interfaces, return concrete types
- Table-driven tests
- Composition over inheritance`,
		AllowedTools: []string{"Read", "Write", "Edit", "Grep", "Glob", "Bash(go *)"},
		Model:        "sonnet",
		MaxTurns:     20,
		Tier:         "sonnet",
	}
}

// GoTuiAgent returns configuration for TUI development
func GoTuiAgent() SubagentConfig {
	return SubagentConfig{
		Name:        "go-tui",
		Description: "Bubbletea/lipgloss TUI specialist",
		AppendPrompt: `You are a Bubbletea TUI expert. Focus on:
- tea.Model implementation patterns
- tea.Cmd for async operations
- lipgloss styling
- Component composition
- Keyboard navigation`,
		AllowedTools: []string{"Read", "Write", "Edit", "Grep", "Glob", "Bash(go *)"},
		Model:        "sonnet",
		MaxTurns:     20,
		Tier:         "sonnet",
	}
}

// GoCliAgent returns configuration for CLI development
func GoCliAgent() SubagentConfig {
	return SubagentConfig{
		Name:        "go-cli",
		Description: "Cobra CLI specialist (flags, subcommands)",
		AppendPrompt: `You are a Cobra CLI expert. Focus on:
- Subcommand organization
- Flag handling and validation
- Help text and documentation
- Environment variable integration
- Shell completion`,
		AllowedTools: []string{"Read", "Write", "Edit", "Grep", "Glob", "Bash(go *)"},
		Model:        "sonnet",
		MaxTurns:     15,
		Tier:         "sonnet",
	}
}

// GoConcurrentAgent returns configuration for concurrency work
func GoConcurrentAgent() SubagentConfig {
	return SubagentConfig{
		Name:        "go-concurrent",
		Description: "Go concurrency specialist (goroutines, channels, errgroup)",
		AppendPrompt: `You are a Go concurrency expert. Focus on:
- Goroutine lifecycle management
- Channel patterns (fan-out, fan-in, pipeline)
- errgroup for parallel operations
- sync.Mutex and sync.RWMutex
- Context cancellation
- Race condition prevention`,
		AllowedTools: []string{"Read", "Write", "Edit", "Grep", "Glob", "Bash(go *)"},
		Model:        "sonnet",
		MaxTurns:     20,
		Tier:         "sonnet",
	}
}

// RegisterPresets registers all preset agents with the manager
func (sm *SubagentManager) RegisterPresets() error {
	for name, factory := range PresetAgents {
		if err := sm.Register(factory()); err != nil {
			return fmt.Errorf("register preset %s: %w", name, err)
		}
	}
	return nil
}

// NewSubagentManagerWithPresets creates a manager with all presets registered
func NewSubagentManagerWithPresets(baseCfg Config) (*SubagentManager, error) {
	sm := NewSubagentManager(baseCfg)
	if err := sm.RegisterPresets(); err != nil {
		return nil, err
	}
	return sm, nil
}

// LoadAgentsFromFile loads agent configs from a JSON file
// This allows consistency with agents-index.json format
func LoadAgentsFromFile(path string) ([]SubagentConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var configs []SubagentConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}

	return configs, nil
}
