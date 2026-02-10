package routing

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

// EXPECTED_AGENT_INDEX_VERSION is the version this code is built for.
const EXPECTED_AGENT_INDEX_VERSION = "2.6.0"

// AgentIndex represents the complete agents-index.json v2.2.0 structure.
// This defines the agent catalog for Claude Code routing and auto-activation.
type AgentIndex struct {
	Version         string          `json:"version"`
	GeneratedAt     string          `json:"generated_at"`
	Description     string          `json:"description"`
	Agents          []Agent         `json:"agents"`
	RoutingRules    RoutingRules    `json:"routing_rules"`
	StateManagement StateManagement `json:"state_management"`
}

// AgentCliFlags represents CLI spawning configuration for an agent.
type AgentCliFlags struct {
	AllowedTools    []string `json:"allowed_tools"`
	AdditionalFlags []string `json:"additional_flags,omitempty"`
}

// Agent represents a single agent definition with complete v2.2.0 fields.
type Agent struct {
	ID                    string          `json:"id"`
	Name                  string          `json:"name"`
	Model                 string          `json:"model"`
	Thinking              bool            `json:"thinking"`
	ThinkingBudget        int             `json:"thinking_budget,omitempty"`
	ThinkingBudgetComplex int             `json:"thinking_budget_complex,omitempty"`
	Tier                  any             `json:"tier"` // Can be float64 (1.5) or string ("external")
	Category              string          `json:"category"`
	Path                  string          `json:"path"`
	Triggers              []string        `json:"triggers"`
	Tools                 []string        `json:"tools"`
	CliFlags              *AgentCliFlags  `json:"cli_flags,omitempty"`
	AutoActivate          *AutoActivate   `json:"auto_activate"` // Can be null or object
	Inputs                []string        `json:"inputs,omitempty"`
	Outputs               []string        `json:"outputs,omitempty"`
	ConventionsRequired   []string        `json:"conventions_required,omitempty"`
	SharpEdgesCount       int             `json:"sharp_edges_count,omitempty"`
	Description           string          `json:"description"`
	AutoFire              []string        `json:"auto_fire,omitempty"`
	ScoutFirst            bool            `json:"scout_first,omitempty"`
	MustDelegate          bool            `json:"must_delegate,omitempty"`
	MinDelegations        int             `json:"min_delegations,omitempty"`
	OutputArtifacts       *OutputArtifacts `json:"output_artifacts,omitempty"`
	InputSources          []string        `json:"input_sources,omitempty"`
	Invocation            string          `json:"invocation,omitempty"`
	Protocols             []string        `json:"protocols,omitempty"`
	StateFiles            *StateFiles     `json:"state_files,omitempty"`
	CostPerInvocation     string          `json:"cost_per_invocation,omitempty"`
	ParallelSafe          bool            `json:"parallel_safe,omitempty"`
	SwarmCompatible       bool            `json:"swarm_compatible,omitempty"`
	OutputFormat          string          `json:"output_format,omitempty"`
	OutputFile            string          `json:"output_file,omitempty"`
	CostCeilingUSD        float64         `json:"cost_ceiling_usd,omitempty"`
	FallbackFor           string          `json:"fallback_for,omitempty"`
	SpawnedBy             []string        `json:"spawned_by,omitempty"`
	CanSpawn              []string        `json:"can_spawn,omitempty"`
}

// AutoActivate defines conditions for agent auto-activation.
type AutoActivate struct {
	Languages    []string `json:"languages,omitempty"`
	Patterns     []string `json:"patterns,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
	FilePatterns []string `json:"file_patterns,omitempty"`
}

// OutputArtifacts defines required outputs for planning agents.
type OutputArtifacts struct {
	Required       []string `json:"required"`
	SpecsLocation  string   `json:"specs_location,omitempty"`
}

// StateFiles defines state file locations for external agents.
type StateFiles struct {
	ScoutOutput      string `json:"scout_output,omitempty"`
	ComplexityScore  string `json:"complexity_score,omitempty"`
}

// RoutingRules defines routing behavior configuration.
type RoutingRules struct {
	IntentGate         IntentGate         `json:"intent_gate"`
	ScoutFirstProtocol ScoutFirstProtocol `json:"scout_first_protocol"`
	ComplexityRouting  ComplexityRouting  `json:"complexity_routing"`
	AutoFire           map[string]string  `json:"auto_fire"`
	ModelTiers         map[string][]string `json:"model_tiers"`
}

// IntentGate defines pre-classification rules.
type IntentGate struct {
	Description string       `json:"description"`
	Types       []IntentType `json:"types"`
}

// IntentType represents a message intent classification.
type IntentType struct {
	Type   string `json:"type"`
	Signal string `json:"signal"`
	Action string `json:"action"`
}

// ScoutFirstProtocol defines pre-routing reconnaissance configuration.
type ScoutFirstProtocol struct {
	Description string   `json:"description"`
	Triggers    []string `json:"triggers"`
	SkipWhen    []string `json:"skip_when"`
	Primary     string   `json:"primary"`
	Fallback    string   `json:"fallback"`
	Output      string   `json:"output"`
}

// ComplexityRouting defines complexity-based tier selection.
type ComplexityRouting struct {
	Description    string                 `json:"description"`
	Calculator     string                 `json:"calculator"`
	Thresholds     map[string]Threshold   `json:"thresholds"`
	ForceExternalIf string                `json:"force_external_if"`
}

// Threshold defines tier selection thresholds.
type Threshold struct {
	MaxScore int `json:"max_score,omitempty"`
	MinScore int `json:"min_score,omitempty"`
}

// StateManagement defines file-based state passing.
type StateManagement struct {
	Description  string              `json:"description"`
	TmpDirectory string              `json:"tmp_directory"`
	Files        map[string]StateFile `json:"files"`
	Cleanup      Cleanup             `json:"cleanup"`
}

// StateFile defines state file metadata.
type StateFile struct {
	WrittenBy  []string `json:"written_by"`
	ReadBy     []string `json:"read_by"`
	TTLMinutes *int     `json:"ttl_minutes"` // Can be null
	ArchivedTo string   `json:"archived_to,omitempty"`
}

// Cleanup defines state cleanup rules.
type Cleanup struct {
	Trigger string `json:"trigger"`
	Action  string `json:"action"`
}

// LoadAgentIndex loads and validates agents-index.json.
// Priority: GOGENT_AGENTS_INDEX env var > GOGENT_PROJECT_DIR > XDG config directory default.
// Returns an error if file is missing, malformed, or version mismatch detected.
func LoadAgentIndex() (*AgentIndex, error) {
	agentIndexPath := os.Getenv("GOGENT_AGENTS_INDEX")

	// If explicit path not set, try project-specific or XDG default
	if agentIndexPath == "" {
		// Priority 1: GOGENT_PROJECT_DIR (test isolation)
		if projectDir := os.Getenv("GOGENT_PROJECT_DIR"); projectDir != "" {
			path := filepath.Join(projectDir, ".claude", "agents", "agents-index.json")
			if _, err := os.Stat(path); err == nil {
				agentIndexPath = path
			}
		}

		// Priority 2: XDG default
		if agentIndexPath == "" {
			configHome := os.Getenv("XDG_CONFIG_HOME")
			if configHome == "" {
				home := os.Getenv("HOME")
				if home == "" {
					return nil, fmt.Errorf("[routing] HOME environment variable not set")
				}
				configHome = filepath.Join(home, ".config")
			}
			agentIndexPath = filepath.Join(configHome, "..", ".claude", "agents", "agents-index.json")
		}
	}

	data, err := os.ReadFile(agentIndexPath)
	if err != nil {
		return nil, fmt.Errorf("[routing] Failed to read agents-index.json from %s: %w", agentIndexPath, err)
	}

	var index AgentIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("[routing] Failed to parse agents-index.json: %w", err)
	}

	// Validate agent index version
	if err := index.Validate(); err != nil {
		return nil, err
	}

	return &index, nil
}

// Validate performs semantic validation on the loaded agent index.
// Checks version compatibility, agent ID uniqueness, and reference integrity.
func (a *AgentIndex) Validate() error {
	// Version check
	if a.Version != EXPECTED_AGENT_INDEX_VERSION {
		return fmt.Errorf(
			"[routing] Agent index version mismatch: expected %s, got %s. Update code or agents-index.json.",
			EXPECTED_AGENT_INDEX_VERSION,
			a.Version,
		)
	}

	// Agent ID uniqueness check
	seen := make(map[string]bool)
	for _, agent := range a.Agents {
		if seen[agent.ID] {
			return fmt.Errorf(
				"[routing] Duplicate agent ID: %s",
				agent.ID,
			)
		}
		seen[agent.ID] = true

		// Validate agent has required fields
		if err := agent.ValidateAgent(); err != nil {
			return fmt.Errorf("[routing] Agent %s validation failed: %w", agent.ID, err)
		}
	}

	// Validate model tier mappings reference existing agents
	for tier, agentIDs := range a.RoutingRules.ModelTiers {
		for _, agentID := range agentIDs {
			found := false
			for _, agent := range a.Agents {
				if agent.ID == agentID {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf(
					"[routing] Model tier %q references unknown agent: %s",
					tier,
					agentID,
				)
			}
		}
	}

	// Validate AutoActivate.Dependencies for circular references
	if err := a.ValidateDependencies(); err != nil {
		return err
	}

	return nil
}

// ValidateAgent performs validation on individual agent configuration.
func (ag *Agent) ValidateAgent() error {
	// Required fields
	if ag.ID == "" {
		return fmt.Errorf("agent missing required field: id")
	}
	if ag.Name == "" {
		return fmt.Errorf("agent missing required field: name")
	}
	if ag.Model == "" {
		return fmt.Errorf("agent missing required field: model")
	}
	if ag.Category == "" {
		return fmt.Errorf("agent missing required field: category")
	}
	if ag.Path == "" {
		return fmt.Errorf("agent missing required field: path")
	}
	if len(ag.Tools) == 0 && ag.Model != "external" {
		return fmt.Errorf("agent missing required field: tools (unless external)")
	}

	// Tier validation
	switch tier := ag.Tier.(type) {
	case float64:
		// Valid numeric tier
		if tier < 1 || tier > 3 {
			return fmt.Errorf("invalid numeric tier: %v (must be 1-3)", tier)
		}
	case string:
		// Valid string tier (only "external" allowed)
		if tier != "external" {
			return fmt.Errorf("invalid string tier: %q (only 'external' allowed)", tier)
		}
	default:
		return fmt.Errorf("tier must be float64 or string, got %T", ag.Tier)
	}

	return nil
}

// GetAgentByID returns the agent with the specified ID.
// Returns an error if the agent does not exist.
func (a *AgentIndex) GetAgentByID(agentID string) (*Agent, error) {
	for i := range a.Agents {
		if a.Agents[i].ID == agentID {
			return &a.Agents[i], nil
		}
	}
	return nil, fmt.Errorf("[routing] Unknown agent: %s", agentID)
}

// GetAgentsByTier returns all agents in the specified tier.
// Returns an error if the tier does not exist in model_tiers.
func (a *AgentIndex) GetAgentsByTier(tierName string) ([]*Agent, error) {
	agentIDs, exists := a.RoutingRules.ModelTiers[tierName]
	if !exists {
		return nil, fmt.Errorf("[routing] Unknown tier: %s", tierName)
	}

	agents := make([]*Agent, 0, len(agentIDs))
	for _, agentID := range agentIDs {
		agent, err := a.GetAgentByID(agentID)
		if err != nil {
			return nil, err
		}
		agents = append(agents, agent)
	}
	return agents, nil
}

// GetToolsForAgent returns the tool list for the specified agent.
// Returns an error if the agent does not exist.
func (a *AgentIndex) GetToolsForAgent(agentID string) ([]string, error) {
	agent, err := a.GetAgentByID(agentID)
	if err != nil {
		return nil, err
	}
	return agent.Tools, nil
}

// FindAgentByLanguage returns agents that auto-activate for the given language.
// Returns empty slice if no matching agents found.
func (a *AgentIndex) FindAgentByLanguage(language string) []*Agent {
	matches := make([]*Agent, 0)
	for i := range a.Agents {
		if a.Agents[i].AutoActivate != nil {
			if slices.Contains(a.Agents[i].AutoActivate.Languages, language) {
				matches = append(matches, &a.Agents[i])
			}
		}
	}
	return matches
}

// FindAgentByPattern returns agents that auto-activate for the given pattern.
// Returns empty slice if no matching agents found.
func (a *AgentIndex) FindAgentByPattern(pattern string) []*Agent {
	matches := make([]*Agent, 0)
	for i := range a.Agents {
		if a.Agents[i].AutoActivate != nil {
			if slices.Contains(a.Agents[i].AutoActivate.Patterns, pattern) {
				matches = append(matches, &a.Agents[i])
			}
		}
	}
	return matches
}

// FindAgentByTrigger returns agents with the specified trigger phrase.
// Returns empty slice if no matching agents found.
func (a *AgentIndex) FindAgentByTrigger(trigger string) []*Agent {
	matches := make([]*Agent, 0)
	for i := range a.Agents {
		if slices.Contains(a.Agents[i].Triggers, trigger) {
			matches = append(matches, &a.Agents[i])
		}
	}
	return matches
}

// FindAgentByCategory returns all agents in the specified category.
// Returns empty slice if no matching agents found.
func (a *AgentIndex) FindAgentByCategory(category string) []*Agent {
	matches := make([]*Agent, 0)
	for i := range a.Agents {
		if a.Agents[i].Category == category {
			matches = append(matches, &a.Agents[i])
		}
	}
	return matches
}

// GetScoutAgents returns all agents with scout_first=true or in scout protocols.
// Returns empty slice if no scout agents found.
func (a *AgentIndex) GetScoutAgents() []*Agent {
	matches := make([]*Agent, 0)
	for i := range a.Agents {
		// Check scout_first flag
		if a.Agents[i].ScoutFirst {
			matches = append(matches, &a.Agents[i])
			continue
		}
		// Check if agent has scout protocol
		if slices.Contains(a.Agents[i].Protocols, "scout") {
			matches = append(matches, &a.Agents[i])
		}
	}
	return matches
}

// GetTierForAgent returns the tier name for an agent by looking up model_tiers.
// Returns an error if the agent is not found in any tier.
func (a *AgentIndex) GetTierForAgent(agentID string) (string, error) {
	for tierName, agentIDs := range a.RoutingRules.ModelTiers {
		if slices.Contains(agentIDs, agentID) {
			return tierName, nil
		}
	}
	return "", fmt.Errorf("[routing] Agent %s not found in any tier", agentID)
}

// ValidateDependencies checks for circular dependencies in agent AutoActivate.Dependencies.
// Uses depth-first search to detect cycles in the dependency graph.
func (a *AgentIndex) ValidateDependencies() error {
	// Build dependency graph
	depGraph := make(map[string][]string)
	for _, agent := range a.Agents {
		if agent.AutoActivate != nil && len(agent.AutoActivate.Dependencies) > 0 {
			depGraph[agent.ID] = agent.AutoActivate.Dependencies
		}
	}

	// Check each agent for circular dependencies
	for _, agent := range a.Agents {
		visited := make(map[string]bool)
		recStack := make(map[string]bool)

		if hasCycle(agent.ID, depGraph, visited, recStack) {
			return fmt.Errorf(
				"[routing] Circular dependency detected starting from agent %q. "+
					"Agent dependency chains must be acyclic. "+
					"Fix: Remove circular reference in AutoActivate.Dependencies",
				agent.ID,
			)
		}
	}

	// Validate all dependencies exist
	for _, agent := range a.Agents {
		if agent.AutoActivate == nil {
			continue
		}
		for _, depID := range agent.AutoActivate.Dependencies {
			found := false
			for _, a := range a.Agents {
				if a.ID == depID {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf(
					"[routing] Agent %q references missing dependency: %s. "+
						"Ensure dependency exists in agents-index.json",
					agent.ID,
					depID,
				)
			}
		}
	}

	return nil
}

// hasCycle performs DFS to detect cycles in dependency graph.
func hasCycle(agentID string, graph map[string][]string, visited, recStack map[string]bool) bool {
	visited[agentID] = true
	recStack[agentID] = true

	// Check all dependencies
	if deps, exists := graph[agentID]; exists {
		for _, dep := range deps {
			if !visited[dep] {
				if hasCycle(dep, graph, visited, recStack) {
					return true
				}
			} else if recStack[dep] {
				// Found a back edge (cycle)
				return true
			}
		}
	}

	recStack[agentID] = false
	return false
}

// GetAllowedTools returns CLI-permitted tools for this agent.
// Returns cli_flags.allowed_tools if configured, otherwise conservative read-only fallback.
func (ag *Agent) GetAllowedTools() []string {
	if ag.CliFlags != nil && len(ag.CliFlags.AllowedTools) > 0 {
		return ag.CliFlags.AllowedTools
	}
	return []string{"Read", "Glob", "Grep"}
}
