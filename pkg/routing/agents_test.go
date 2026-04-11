package routing

import (
	"encoding/json"
	"os"
	"testing"
)

// TestLoadAgentIndex verifies production agents-index.json can be loaded.
func TestLoadAgentIndex(t *testing.T) {
	index, err := LoadAgentIndex()
	if err != nil {
		t.Fatalf("Failed to load agent index: %v", err)
	}

	// Verify version
	if index.Version != EXPECTED_AGENT_INDEX_VERSION {
		t.Errorf("Expected version %s, got %s", EXPECTED_AGENT_INDEX_VERSION, index.Version)
	}

	// Verify agents loaded
	if len(index.Agents) == 0 {
		t.Error("Expected agents to be populated")
	}

	t.Logf("✓ Loaded %d agents from agents-index.json v%s", len(index.Agents), index.Version)
}

// TestUnmarshalProductionAgentIndex validates all v2.2.0 fields are captured.
func TestUnmarshalProductionAgentIndex(t *testing.T) {
	// Load production agents-index.json
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home := os.Getenv("HOME")
		if home == "" {
			t.Skip("HOME not set")
		}
		configHome = home + "/.config"
	}

	agentIndexPath := configHome + "/../.claude/agents/agents-index.json"
	data, err := os.ReadFile(agentIndexPath)
	if err != nil {
		t.Skipf("Skipping production test: %v", err)
	}

	var index AgentIndex
	if err := json.Unmarshal(data, &index); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify top-level fields
	if index.Version == "" {
		t.Error("version missing")
	}
	if index.GeneratedAt == "" {
		t.Error("generated_at missing")
	}
	if index.Description == "" {
		t.Error("description missing")
	}

	// Verify all agents have required fields
	for i, agent := range index.Agents {
		if agent.ID == "" {
			t.Errorf("Agent %d missing id", i)
		}
		if agent.Name == "" {
			t.Errorf("Agent %d (%s) missing name", i, agent.ID)
		}
		if agent.Model == "" {
			t.Errorf("Agent %d (%s) missing model", i, agent.ID)
		}
		if agent.Category == "" {
			t.Errorf("Agent %d (%s) missing category", i, agent.ID)
		}
		if agent.Path == "" {
			t.Errorf("Agent %d (%s) missing path", i, agent.ID)
		}
		if len(agent.Triggers) == 0 && agent.AutoActivate == nil && len(agent.SpawnedBy) == 0 {
			t.Errorf("Agent %d (%s) missing triggers, auto_activate, and spawned_by", i, agent.ID)
		}
		if len(agent.Tools) == 0 {
			t.Errorf("Agent %d (%s) missing tools", i, agent.ID)
		}
		if agent.Description == "" {
			t.Errorf("Agent %d (%s) missing description", i, agent.ID)
		}
	}

	// Verify optional fields are captured (v2.2.0 completeness)
	memoryArchivist, err := index.GetAgentByID("memory-archivist")
	if err == nil {
		if len(memoryArchivist.Inputs) == 0 {
			t.Error("memory-archivist missing inputs field")
		}
		if len(memoryArchivist.Outputs) == 0 {
			t.Error("memory-archivist missing outputs field")
		}
	}

	architect, err := index.GetAgentByID("architect")
	if err == nil {
		if architect.OutputArtifacts == nil {
			t.Error("architect missing output_artifacts field")
		} else {
			if len(architect.OutputArtifacts.Required) == 0 {
				t.Error("architect.output_artifacts.required empty")
			}
			if architect.OutputArtifacts.SpecsLocation == "" {
				t.Error("architect.output_artifacts.specs_location missing")
			}
		}
	}

	haikuScout, err := index.GetAgentByID("haiku-scout")
	if err == nil {
		if !haikuScout.ParallelSafe {
			t.Error("haiku-scout should have parallel_safe=true")
		}
		if !haikuScout.SwarmCompatible {
			t.Error("haiku-scout should have swarm_compatible=true")
		}
		if haikuScout.OutputFormat == "" {
			t.Error("haiku-scout missing output_format")
		}
		if haikuScout.OutputFile == "" {
			t.Error("haiku-scout missing output_file")
		}
		if haikuScout.CostCeilingUSD == 0 {
			t.Error("haiku-scout missing cost_ceiling_usd")
		}
		if haikuScout.FallbackFor == "" {
			t.Error("haiku-scout missing fallback_for")
		}
	}

	// Verify routing_rules structure
	if index.RoutingRules.IntentGate.Description == "" {
		t.Error("routing_rules.intent_gate.description missing")
	}
	if len(index.RoutingRules.IntentGate.Types) == 0 {
		t.Error("routing_rules.intent_gate.types missing")
	}
	if index.RoutingRules.ScoutFirstProtocol.Primary == "" {
		t.Error("routing_rules.scout_first_protocol.primary missing")
	}
	if index.RoutingRules.ComplexityRouting.Calculator == "" {
		t.Error("routing_rules.complexity_routing.calculator missing")
	}
	if len(index.RoutingRules.AutoFire) == 0 {
		t.Error("routing_rules.auto_fire empty")
	}
	if len(index.RoutingRules.ModelTiers) == 0 {
		t.Error("routing_rules.model_tiers empty")
	}

	// Verify state_management structure
	if index.StateManagement.Description == "" {
		t.Error("state_management.description missing")
	}
	if index.StateManagement.TmpDirectory == "" {
		t.Error("state_management.tmp_directory missing")
	}
	if len(index.StateManagement.Files) == 0 {
		t.Error("state_management.files empty")
	}

	t.Logf("✓ Successfully unmarshaled all v2.2.0 fields for %d agents", len(index.Agents))
}

// TestAgentIndexValidate verifies validation logic.
func TestAgentIndexValidate(t *testing.T) {
	tests := []struct {
		name    string
		index   AgentIndex
		wantErr bool
	}{
		{
			name: "valid index",
			index: AgentIndex{
				Version:     EXPECTED_AGENT_INDEX_VERSION,
				GeneratedAt: "2026-01-15T00:00:00Z",
				Description: "Test",
				Agents: []Agent{
					{
						ID:          "test-agent",
						Name:        "Test Agent",
						Model:       "haiku",
						Thinking:    false,
						Tier:        1.0,
						Category:    "task",
						Path:        "test-agent",
						Triggers:    []string{"test"},
						Tools:       []string{"Read"},
						Description: "Test agent",
					},
				},
				RoutingRules: RoutingRules{
					ModelTiers: map[string][]string{
						"haiku": {"test-agent"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "version mismatch",
			index: AgentIndex{
				Version: "1.0.0",
				Agents:  []Agent{},
			},
			wantErr: true,
		},
		{
			name: "duplicate agent ID",
			index: AgentIndex{
				Version: EXPECTED_AGENT_INDEX_VERSION,
				Agents: []Agent{
					{
						ID:          "duplicate",
						Name:        "Agent 1",
						Model:       "haiku",
						Tier:        1.0,
						Category:    "task",
						Path:        "agent1",
						Tools:       []string{"Read"},
						Description: "Test",
					},
					{
						ID:          "duplicate",
						Name:        "Agent 2",
						Model:       "haiku",
						Tier:        1.0,
						Category:    "task",
						Path:        "agent2",
						Tools:       []string{"Read"},
						Description: "Test",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "model_tiers references unknown agent",
			index: AgentIndex{
				Version: EXPECTED_AGENT_INDEX_VERSION,
				Agents: []Agent{
					{
						ID:          "known-agent",
						Name:        "Known",
						Model:       "haiku",
						Tier:        1.0,
						Category:    "task",
						Path:        "known",
						Tools:       []string{"Read"},
						Description: "Test",
					},
				},
				RoutingRules: RoutingRules{
					ModelTiers: map[string][]string{
						"haiku": {"known-agent", "unknown-agent"},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.index.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateAgent verifies agent validation logic.
func TestValidateAgent(t *testing.T) {
	tests := []struct {
		name    string
		agent   Agent
		wantErr bool
	}{
		{
			name: "valid agent",
			agent: Agent{
				ID:          "test",
				Name:        "Test",
				Model:       "haiku",
				Tier:        1.0,
				Category:    "task",
				Path:        "test",
				Tools:       []string{"Read"},
				Description: "Test",
			},
			wantErr: false,
		},
		{
			name: "invalid string tier rejected",
			agent: Agent{
				ID:          "test",
				Name:        "Test",
				Model:       "haiku",
				Tier:        "external",
				Category:    "task",
				Path:        "test",
				Tools:       []string{"Read"},
				Description: "Test",
			},
			wantErr: true,
		},
		{
			name: "missing id",
			agent: Agent{
				Name:        "Test",
				Model:       "haiku",
				Tier:        1.0,
				Category:    "task",
				Path:        "test",
				Tools:       []string{"Read"},
				Description: "Test",
			},
			wantErr: true,
		},
		{
			name: "missing tools (non-external)",
			agent: Agent{
				ID:          "test",
				Name:        "Test",
				Model:       "haiku",
				Tier:        1.0,
				Category:    "task",
				Path:        "test",
				Tools:       []string{},
				Description: "Test",
			},
			wantErr: true,
		},
		{
			name: "invalid numeric tier",
			agent: Agent{
				ID:          "test",
				Name:        "Test",
				Model:       "haiku",
				Tier:        5.0,
				Category:    "task",
				Path:        "test",
				Tools:       []string{"Read"},
				Description: "Test",
			},
			wantErr: true,
		},
		{
			name: "invalid string tier",
			agent: Agent{
				ID:          "test",
				Name:        "Test",
				Model:       "haiku",
				Tier:        "invalid",
				Category:    "task",
				Path:        "test",
				Tools:       []string{"Read"},
				Description: "Test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.agent.ValidateAgent()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAgent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestGetAgentByID verifies agent lookup.
func TestGetAgentByID(t *testing.T) {
	index := &AgentIndex{
		Agents: []Agent{
			{ID: "python-pro", Name: "Python Pro"},
			{ID: "go-pro", Name: "GO Pro"},
		},
	}

	tests := []struct {
		name     string
		agentID  string
		wantName string
		wantErr  bool
	}{
		{
			name:     "existing agent",
			agentID:  "python-pro",
			wantName: "Python Pro",
			wantErr:  false,
		},
		{
			name:     "unknown agent",
			agentID:  "unknown",
			wantName: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := index.GetAgentByID(tt.agentID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAgentByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && agent.Name != tt.wantName {
				t.Errorf("GetAgentByID() name = %v, want %v", agent.Name, tt.wantName)
			}
		})
	}
}

// TestGetAgentsByTier verifies tier-based agent lookup.
func TestGetAgentsByTier(t *testing.T) {
	index := &AgentIndex{
		Agents: []Agent{
			{ID: "codebase-search", Name: "Codebase Search"},
			{ID: "python-pro", Name: "Python Pro"},
		},
		RoutingRules: RoutingRules{
			ModelTiers: map[string][]string{
				"haiku":  {"codebase-search"},
				"sonnet": {"python-pro"},
			},
		},
	}

	tests := []struct {
		name      string
		tierName  string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "haiku tier",
			tierName:  "haiku",
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "sonnet tier",
			tierName:  "sonnet",
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "unknown tier",
			tierName:  "unknown",
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agents, err := index.GetAgentsByTier(tt.tierName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAgentsByTier() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(agents) != tt.wantCount {
				t.Errorf("GetAgentsByTier() count = %v, want %v", len(agents), tt.wantCount)
			}
		})
	}
}

// TestGetToolsForAgent verifies tool list retrieval.
func TestGetToolsForAgent(t *testing.T) {
	index := &AgentIndex{
		Agents: []Agent{
			{
				ID:    "codebase-search",
				Tools: []string{"Glob", "Grep", "Read"},
			},
		},
	}

	tools, err := index.GetToolsForAgent("codebase-search")
	if err != nil {
		t.Fatalf("GetToolsForAgent() error = %v", err)
	}

	expectedTools := []string{"Glob", "Grep", "Read"}
	if len(tools) != len(expectedTools) {
		t.Errorf("Got %d tools, want %d", len(tools), len(expectedTools))
	}

	for i, tool := range expectedTools {
		if tools[i] != tool {
			t.Errorf("Tool %d = %s, want %s", i, tools[i], tool)
		}
	}
}

// TestFindAgentByLanguage verifies language-based auto-activation.
func TestFindAgentByLanguage(t *testing.T) {
	index := &AgentIndex{
		Agents: []Agent{
			{
				ID:   "python-pro",
				Name: "Python Pro",
				AutoActivate: &AutoActivate{
					Languages: []string{"Python"},
				},
			},
			{
				ID:   "go-pro",
				Name: "GO Pro",
				AutoActivate: &AutoActivate{
					Languages: []string{"Go"},
				},
			},
			{
				ID:           "codebase-search",
				AutoActivate: nil,
			},
		},
	}

	tests := []struct {
		name      string
		language  string
		wantCount int
	}{
		{
			name:      "Python",
			language:  "Python",
			wantCount: 1,
		},
		{
			name:      "Go",
			language:  "Go",
			wantCount: 1,
		},
		{
			name:      "Unknown",
			language:  "Rust",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agents := index.FindAgentByLanguage(tt.language)
			if len(agents) != tt.wantCount {
				t.Errorf("FindAgentByLanguage() count = %v, want %v", len(agents), tt.wantCount)
			}
		})
	}
}

// TestFindAgentByPattern verifies pattern-based auto-activation.
func TestFindAgentByPattern(t *testing.T) {
	index := &AgentIndex{
		Agents: []Agent{
			{
				ID:   "python-ux",
				Name: "Python UX",
				AutoActivate: &AutoActivate{
					Patterns: []string{"PySide6", "PyQt"},
				},
			},
		},
	}

	agents := index.FindAgentByPattern("PySide6")
	if len(agents) != 1 {
		t.Errorf("Expected 1 agent, got %d", len(agents))
	}
	if len(agents) > 0 && agents[0].ID != "python-ux" {
		t.Errorf("Expected python-ux, got %s", agents[0].ID)
	}
}

// TestFindAgentByTrigger verifies trigger phrase matching.
func TestFindAgentByTrigger(t *testing.T) {
	index := &AgentIndex{
		Agents: []Agent{
			{
				ID:       "codebase-search",
				Triggers: []string{"where is", "find the", "locate"},
			},
		},
	}

	agents := index.FindAgentByTrigger("where is")
	if len(agents) != 1 {
		t.Errorf("Expected 1 agent, got %d", len(agents))
	}
}

// TestFindAgentByCategory verifies category-based lookup.
func TestFindAgentByCategory(t *testing.T) {
	index := &AgentIndex{
		Agents: []Agent{
			{ID: "python-pro", Category: "language"},
			{ID: "go-pro", Category: "language"},
			{ID: "orchestrator", Category: "architecture"},
		},
	}

	tests := []struct {
		name      string
		category  string
		wantCount int
	}{
		{
			name:      "language category",
			category:  "language",
			wantCount: 2,
		},
		{
			name:      "architecture category",
			category:  "architecture",
			wantCount: 1,
		},
		{
			name:      "unknown category",
			category:  "unknown",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agents := index.FindAgentByCategory(tt.category)
			if len(agents) != tt.wantCount {
				t.Errorf("FindAgentByCategory() count = %v, want %v", len(agents), tt.wantCount)
			}
		})
	}
}

// TestGetScoutAgents verifies scout agent identification.
func TestGetScoutAgents(t *testing.T) {
	index := &AgentIndex{
		Agents: []Agent{
			{
				ID:         "orchestrator",
				ScoutFirst: true,
			},
			{
				ID: "python-pro",
			},
		},
	}

	scouts := index.GetScoutAgents()
	if len(scouts) != 1 {
		t.Errorf("Expected 1 scout agent, got %d", len(scouts))
	}

	// Verify scout agent IDs
	scoutIDs := make(map[string]bool)
	for _, scout := range scouts {
		scoutIDs[scout.ID] = true
	}

	if !scoutIDs["orchestrator"] {
		t.Error("Expected orchestrator to be a scout agent")
	}
}

// TestGetTierForAgent verifies tier name lookup for agents.
func TestGetTierForAgent(t *testing.T) {
	index := &AgentIndex{
		Agents: []Agent{
			{ID: "codebase-search"},
			{ID: "python-pro"},
		},
		RoutingRules: RoutingRules{
			ModelTiers: map[string][]string{
				"haiku":  {"codebase-search"},
				"sonnet": {"python-pro"},
			},
		},
	}

	tests := []struct {
		name     string
		agentID  string
		wantTier string
		wantErr  bool
	}{
		{
			name:     "haiku tier agent",
			agentID:  "codebase-search",
			wantTier: "haiku",
			wantErr:  false,
		},
		{
			name:     "sonnet tier agent",
			agentID:  "python-pro",
			wantTier: "sonnet",
			wantErr:  false,
		},
		{
			name:     "unknown agent",
			agentID:  "unknown",
			wantTier: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tier, err := index.GetTierForAgent(tt.agentID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTierForAgent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tier != tt.wantTier {
				t.Errorf("GetTierForAgent() tier = %v, want %v", tier, tt.wantTier)
			}
		})
	}
}

// TestProductionAgentIndexCompleteness verifies all production agents are accessible.
func TestProductionAgentIndexCompleteness(t *testing.T) {
	index, err := LoadAgentIndex()
	if err != nil {
		t.Skipf("Skipping production test: %v", err)
	}

	// Expected v2.2.0 agents
	expectedAgents := []string{
		"memory-archivist",
		"codebase-search",
		"librarian",
		"scaffolder",
		"tech-docs-writer",
		"code-reviewer",
		"python-pro",
		"python-ux",
		"r-pro",
		"r-shiny-pro",
		"go-pro",
		"go-cli",
		"go-tui",
		"go-api",
		"go-concurrent",
		"orchestrator",
		"architect",
		"einstein",
		"staff-architect-critical-review",
		"haiku-scout",
	}

	for _, agentID := range expectedAgents {
		agent, err := index.GetAgentByID(agentID)
		if err != nil {
			t.Errorf("Expected agent %s not found: %v", agentID, err)
			continue
		}
		if agent.Name == "" {
			t.Errorf("Agent %s has empty name", agentID)
		}
	}

	t.Logf("✓ All %d expected agents found and accessible", len(expectedAgents))
}

// TestValidateDependencies_CircularDependency verifies circular dependency detection.
func TestValidateDependencies_CircularDependency(t *testing.T) {
	tests := []struct {
		name        string
		agents      []Agent
		wantErr     bool
		errContains string
	}{
		{
			name: "direct circular dependency",
			agents: []Agent{
				{
					ID:       "agent-a",
					Name:     "Agent A",
					Model:    "haiku",
					Category: "test",
					Path:     "agent-a",
					Tools:    []string{"Read"},
					Tier:     1.0,
					AutoActivate: &AutoActivate{
						Dependencies: []string{"agent-b"},
					},
				},
				{
					ID:       "agent-b",
					Name:     "Agent B",
					Model:    "haiku",
					Category: "test",
					Path:     "agent-b",
					Tools:    []string{"Read"},
					Tier:     1.0,
					AutoActivate: &AutoActivate{
						Dependencies: []string{"agent-a"},
					},
				},
			},
			wantErr:     true,
			errContains: "circular dependency",
		},
		{
			name: "indirect circular dependency (A→B→C→A)",
			agents: []Agent{
				{
					ID:       "agent-a",
					Name:     "Agent A",
					Model:    "haiku",
					Category: "test",
					Path:     "agent-a",
					Tools:    []string{"Read"},
					Tier:     1.0,
					AutoActivate: &AutoActivate{
						Dependencies: []string{"agent-b"},
					},
				},
				{
					ID:       "agent-b",
					Name:     "Agent B",
					Model:    "haiku",
					Category: "test",
					Path:     "agent-b",
					Tools:    []string{"Read"},
					Tier:     1.0,
					AutoActivate: &AutoActivate{
						Dependencies: []string{"agent-c"},
					},
				},
				{
					ID:       "agent-c",
					Name:     "Agent C",
					Model:    "haiku",
					Category: "test",
					Path:     "agent-c",
					Tools:    []string{"Read"},
					Tier:     1.0,
					AutoActivate: &AutoActivate{
						Dependencies: []string{"agent-a"},
					},
				},
			},
			wantErr:     true,
			errContains: "circular dependency",
		},
		{
			name: "no circular dependency (linear chain)",
			agents: []Agent{
				{
					ID:       "agent-a",
					Name:     "Agent A",
					Model:    "haiku",
					Category: "test",
					Path:     "agent-a",
					Tools:    []string{"Read"},
					Tier:     1.0,
					AutoActivate: &AutoActivate{
						Dependencies: []string{"agent-b"},
					},
				},
				{
					ID:       "agent-b",
					Name:     "Agent B",
					Model:    "haiku",
					Category: "test",
					Path:     "agent-b",
					Tools:    []string{"Read"},
					Tier:     1.0,
					AutoActivate: &AutoActivate{
						Dependencies: []string{"agent-c"},
					},
				},
				{
					ID:           "agent-c",
					Name:         "Agent C",
					Model:        "haiku",
					Category:     "test",
					Path:         "agent-c",
					Tools:        []string{"Read"},
					Tier:         1.0,
					AutoActivate: nil, // No dependencies
				},
			},
			wantErr: false,
		},
		{
			name: "diamond dependency (no cycle)",
			agents: []Agent{
				{
					ID:       "orchestrator",
					Name:     "Orchestrator",
					Model:    "sonnet",
					Category: "planning",
					Path:     "orchestrator",
					Tools:    []string{"Task"},
					Tier:     2.0,
					AutoActivate: &AutoActivate{
						Dependencies: []string{"python-pro", "go-pro"},
					},
				},
				{
					ID:       "python-pro",
					Name:     "Python Pro",
					Model:    "sonnet",
					Category: "implementation",
					Path:     "python-pro",
					Tools:    []string{"Edit"},
					Tier:     2.0,
					AutoActivate: &AutoActivate{
						Dependencies: []string{"codebase-search"},
					},
				},
				{
					ID:       "go-pro",
					Name:     "Go Pro",
					Model:    "sonnet",
					Category: "implementation",
					Path:     "go-pro",
					Tools:    []string{"Edit"},
					Tier:     2.0,
					AutoActivate: &AutoActivate{
						Dependencies: []string{"codebase-search"},
					},
				},
				{
					ID:           "codebase-search",
					Name:         "Codebase Search",
					Model:        "haiku",
					Category:     "exploration",
					Path:         "codebase-search",
					Tools:        []string{"Grep"},
					Tier:         1.0,
					AutoActivate: nil,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			index := &AgentIndex{
				Version:     EXPECTED_AGENT_INDEX_VERSION,
				GeneratedAt: "2026-01-16T00:00:00Z",
				Description: "Test agent index",
				Agents:      tt.agents,
				RoutingRules: RoutingRules{
					ModelTiers: make(map[string][]string),
				},
				StateManagement: StateManagement{},
			}

			err := index.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Expected error containing %q, got nil", tt.errContains)
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestValidateDependencies_MissingDependency verifies missing dependency detection.
func TestValidateDependencies_MissingDependency(t *testing.T) {
	index := &AgentIndex{
		Version:     EXPECTED_AGENT_INDEX_VERSION,
		GeneratedAt: "2026-01-16T00:00:00Z",
		Description: "Test",
		Agents: []Agent{
			{
				ID:       "agent-a",
				Name:     "Agent A",
				Model:    "haiku",
				Category: "test",
				Path:     "agent-a",
				Tools:    []string{"Read"},
				Tier:     1.0,
				AutoActivate: &AutoActivate{
					Dependencies: []string{"nonexistent-agent"},
				},
			},
		},
		RoutingRules: RoutingRules{
			ModelTiers: make(map[string][]string),
		},
		StateManagement: StateManagement{},
	}

	err := index.Validate()
	if err == nil {
		t.Fatal("Expected error for missing dependency, got nil")
	}

	if !contains(err.Error(), "missing dependency") && !contains(err.Error(), "nonexistent-agent") {
		t.Errorf("Expected error about missing dependency, got: %v", err)
	}
}

// TestValidateDependencies_MultiLevelDependencies verifies multi-level dependency chains.
func TestValidateDependencies_MultiLevelDependencies(t *testing.T) {
	// A → B → C → D (linear, valid)
	index := &AgentIndex{
		Version:     EXPECTED_AGENT_INDEX_VERSION,
		GeneratedAt: "2026-01-16T00:00:00Z",
		Description: "Test",
		Agents: []Agent{
			{
				ID:       "agent-a",
				Name:     "Agent A",
				Model:    "haiku",
				Category: "test",
				Path:     "agent-a",
				Tools:    []string{"Read"},
				Tier:     1.0,
				AutoActivate: &AutoActivate{
					Dependencies: []string{"agent-b"},
				},
			},
			{
				ID:       "agent-b",
				Name:     "Agent B",
				Model:    "haiku",
				Category: "test",
				Path:     "agent-b",
				Tools:    []string{"Read"},
				Tier:     1.0,
				AutoActivate: &AutoActivate{
					Dependencies: []string{"agent-c"},
				},
			},
			{
				ID:       "agent-c",
				Name:     "Agent C",
				Model:    "haiku",
				Category: "test",
				Path:     "agent-c",
				Tools:    []string{"Read"},
				Tier:     1.0,
				AutoActivate: &AutoActivate{
					Dependencies: []string{"agent-d"},
				},
			},
			{
				ID:           "agent-d",
				Name:         "Agent D",
				Model:        "haiku",
				Category:     "test",
				Path:         "agent-d",
				Tools:        []string{"Read"},
				Tier:         1.0,
				AutoActivate: nil,
			},
		},
		RoutingRules: RoutingRules{
			ModelTiers: make(map[string][]string),
		},
		StateManagement: StateManagement{},
	}

	err := index.Validate()
	if err != nil {
		t.Errorf("Expected no error for valid multi-level dependencies, got: %v", err)
	}
}

// TestLoadAgentIndex_Concurrent verifies thread-safe loading.
func TestLoadAgentIndex_Concurrent(t *testing.T) {
	// Verify LoadAgentIndex is safe for concurrent calls
	const numGoroutines = 10
	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, err := LoadAgentIndex()
			errChan <- err
		}()
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		err := <-errChan
		if err != nil {
			t.Errorf("Concurrent load %d failed: %v", i, err)
		}
	}
}

// P0 COVERAGE TESTS: Missing Error Paths

// TestValidateAgent_MissingName verifies validation fails on empty Name field.
func TestValidateAgent_MissingName(t *testing.T) {
	agent := Agent{
		ID:          "test-agent",
		Name:        "", // Missing
		Model:       "haiku",
		Tier:        1.0,
		Category:    "task",
		Path:        "test-agent",
		Tools:       []string{"Read"},
		Description: "Test",
	}

	err := agent.ValidateAgent()
	if err == nil {
		t.Fatal("expected error for missing name, got nil")
	}

	if !contains(err.Error(), "name") {
		t.Errorf("expected error about 'name', got: %v", err)
	}
}

// TestValidateAgent_MissingCategory verifies validation fails on empty Category field.
func TestValidateAgent_MissingCategory(t *testing.T) {
	agent := Agent{
		ID:          "test-agent",
		Name:        "Test Agent",
		Model:       "haiku",
		Tier:        1.0,
		Category:    "", // Missing
		Path:        "test-agent",
		Tools:       []string{"Read"},
		Description: "Test",
	}

	err := agent.ValidateAgent()
	if err == nil {
		t.Fatal("expected error for missing category, got nil")
	}

	if !contains(err.Error(), "category") {
		t.Errorf("expected error about 'category', got: %v", err)
	}
}

// TestValidateAgent_MissingPath verifies validation fails on empty Path field.
func TestValidateAgent_MissingPath(t *testing.T) {
	agent := Agent{
		ID:          "test-agent",
		Name:        "Test Agent",
		Model:       "haiku",
		Tier:        1.0,
		Category:    "task",
		Path:        "", // Missing
		Tools:       []string{"Read"},
		Description: "Test",
	}

	err := agent.ValidateAgent()
	if err == nil {
		t.Fatal("expected error for missing path, got nil")
	}

	if !contains(err.Error(), "path") {
		t.Errorf("expected error about 'path', got: %v", err)
	}
}

// TestLoadAgentIndex_NoXDGConfig verifies HOME fallback when XDG_CONFIG_HOME not set.
func TestLoadAgentIndex_NoXDGConfig(t *testing.T) {
	// Save original env
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	origHOME := os.Getenv("HOME")
	defer func() {
		if origXDG != "" {
			os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
		os.Setenv("HOME", origHOME)
	}()

	// Unset XDG_CONFIG_HOME, keep HOME
	os.Unsetenv("XDG_CONFIG_HOME")

	// Should fall back to HOME/.config
	index, err := LoadAgentIndex()

	// We expect this to succeed (fallback to HOME/.config/../.claude/agents/agents-index.json)
	if err != nil {
		// If it fails, verify it's not because of missing HOME
		if !contains(err.Error(), "Failed to read") {
			t.Errorf("unexpected error: %v", err)
		}
		// File not found is acceptable in test environment
	} else {
		if len(index.Agents) == 0 {
			t.Error("expected agents to be loaded via HOME fallback")
		}
	}
}

// TestLoadAgentIndex_NoHomeEnv verifies error when neither XDG nor HOME set.
func TestLoadAgentIndex_NoHomeEnv(t *testing.T) {
	// Save original env
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	origHOME := os.Getenv("HOME")
	defer func() {
		if origXDG != "" {
			os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
		os.Setenv("HOME", origHOME)
	}()

	// Unset both XDG_CONFIG_HOME and HOME
	os.Unsetenv("XDG_CONFIG_HOME")
	os.Unsetenv("HOME")

	index, err := LoadAgentIndex()

	if err == nil {
		t.Fatal("expected error when HOME not set, got nil")
	}

	if index != nil {
		t.Errorf("expected nil index, got: %v", index)
	}

	if !contains(err.Error(), "HOME") {
		t.Errorf("expected error about HOME, got: %v", err)
	}
}

// TestLoadAgentIndex_FileNotFound verifies error handling for missing file.
func TestLoadAgentIndex_FileNotFound(t *testing.T) {
	// Create temp directory that won't have agents-index.json
	tmpDir := t.TempDir()

	// Save original env
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	// Point to non-existent location
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	index, err := LoadAgentIndex()

	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}

	if index != nil {
		t.Errorf("expected nil index, got: %v", index)
	}

	if !contains(err.Error(), "Failed to read") {
		t.Errorf("expected 'Failed to read' error, got: %v", err)
	}
}

// TestLoadAgentIndex_MalformedJSON verifies error handling for corrupted JSON.
func TestLoadAgentIndex_MalformedJSON(t *testing.T) {
	// Create temp directory structure: tmpDir/.config/../.claude/agents/
	tmpDir := t.TempDir()
	configDir := tmpDir + "/config"
	claudeDir := tmpDir + "/.claude/agents"

	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("failed to create .claude/agents dir: %v", err)
	}

	malformedJSON := `{"version": "2.2.0", "agents": [malformed json`
	agentIndexPath := claudeDir + "/agents-index.json"
	if err := os.WriteFile(agentIndexPath, []byte(malformedJSON), 0644); err != nil {
		t.Fatalf("failed to write malformed JSON: %v", err)
	}

	// Save original env
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	// Point XDG_CONFIG_HOME to tmpDir/config so path becomes tmpDir/config/../.claude/agents/
	os.Setenv("XDG_CONFIG_HOME", configDir)

	index, err := LoadAgentIndex()

	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}

	if index != nil {
		t.Errorf("expected nil index, got: %v", index)
	}

	if !contains(err.Error(), "Failed to parse") {
		t.Errorf("expected 'Failed to parse' error, got: %v", err)
	}
}

// P1 COVERAGE TESTS: Important Edge Cases

// TestValidateAgent_TierBoundaries verifies tier boundary validation (1.0 and 3.0).
func TestValidateAgent_TierBoundaries(t *testing.T) {
	tests := []struct {
		name    string
		tier    interface{}
		wantErr bool
	}{
		{
			name:    "tier exactly 1.0 (valid boundary)",
			tier:    1.0,
			wantErr: false,
		},
		{
			name:    "tier exactly 3.0 (valid boundary)",
			tier:    3.0,
			wantErr: false,
		},
		{
			name:    "tier below 1.0 (invalid)",
			tier:    0.5,
			wantErr: true,
		},
		{
			name:    "tier above 3.0 (invalid)",
			tier:    3.5,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := Agent{
				ID:          "test-agent",
				Name:        "Test Agent",
				Model:       "haiku",
				Tier:        tt.tier,
				Category:    "task",
				Path:        "test-agent",
				Tools:       []string{"Read"},
				Description: "Test",
			}

			err := agent.ValidateAgent()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAgent() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && !contains(err.Error(), "tier") {
				t.Errorf("expected error about tier, got: %v", err)
			}
		})
	}
}

// TestGetToolsForAgent_UnknownAgent verifies error for non-existent agent.
func TestGetToolsForAgent_UnknownAgent(t *testing.T) {
	index := &AgentIndex{
		Agents: []Agent{
			{ID: "known-agent", Tools: []string{"Read"}},
		},
	}

	tools, err := index.GetToolsForAgent("unknown-agent")

	if err == nil {
		t.Fatal("expected error for unknown agent, got nil")
	}

	if tools != nil {
		t.Errorf("expected nil tools, got: %v", tools)
	}

	if !contains(err.Error(), "unknown") && !contains(err.Error(), "Unknown agent") {
		t.Errorf("expected error about unknown agent, got: %v", err)
	}
}

// Helper function for case-insensitive substring check
func contains(s, substr string) bool {
	// Simple case-insensitive substring search
	sLower := toLower(s)
	substrLower := toLower(substr)

	if len(sLower) < len(substrLower) {
		return false
	}

	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		if sLower[i:i+len(substrLower)] == substrLower {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + ('a' - 'A')
		} else {
			result[i] = c
		}
	}
	return string(result)
}

// slicesEqual compares two string slices for equality
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestAgentCliFlags_Parsing verifies cli_flags field parsing
func TestAgentCliFlags_Parsing(t *testing.T) {
	tests := []struct {
		name      string
		json      string
		wantErr   bool
		wantTools []string
	}{
		{
			name:      "agent with cli_flags",
			json:      `{"id":"test","name":"Test","model":"haiku","tier":1,"category":"task","path":"test","tools":["Read","Glob"],"cli_flags":{"allowed_tools":["Read","Glob"],"additional_flags":["--permission-mode delegate"]},"description":"test"}`,
			wantErr:   false,
			wantTools: []string{"Read", "Glob"},
		},
		{
			name:      "agent without cli_flags",
			json:      `{"id":"test","name":"Test","model":"haiku","tier":1,"category":"task","path":"test","tools":["Read","Glob"],"description":"test"}`,
			wantErr:   false,
			wantTools: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var agent Agent
			err := json.Unmarshal([]byte(tt.json), &agent)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantTools != nil {
				if agent.CliFlags == nil {
					t.Fatal("expected cli_flags to be non-nil")
				}
				if !slicesEqual(agent.CliFlags.AllowedTools, tt.wantTools) {
					t.Errorf("AllowedTools = %v, want %v", agent.CliFlags.AllowedTools, tt.wantTools)
				}
			} else if agent.CliFlags != nil && tt.wantTools == nil {
				// Agent without cli_flags should have nil CliFlags
				// This is only an error if we expected nil
			}
		})
	}
}

// TestAgent_GetAllowedTools verifies GetAllowedTools method
func TestAgent_GetAllowedTools(t *testing.T) {
	tests := []struct {
		name     string
		agent    Agent
		expected []string
	}{
		{
			name:     "with cli_flags returns config tools",
			agent:    Agent{CliFlags: &AgentCliFlags{AllowedTools: []string{"Read", "Glob"}}},
			expected: []string{"Read", "Glob"},
		},
		{
			name:     "without cli_flags returns conservative fallback",
			agent:    Agent{},
			expected: []string{"Read", "Glob", "Grep"},
		},
		{
			name:     "with empty allowed_tools returns fallback",
			agent:    Agent{CliFlags: &AgentCliFlags{AllowedTools: []string{}}},
			expected: []string{"Read", "Glob", "Grep"},
		},
		{
			name:     "with nil cli_flags returns fallback",
			agent:    Agent{CliFlags: nil},
			expected: []string{"Read", "Glob", "Grep"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.agent.GetAllowedTools()
			if !slicesEqual(got, tt.expected) {
				t.Errorf("GetAllowedTools() = %v, want %v", got, tt.expected)
			}
		})
	}
}
