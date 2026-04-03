package routing

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EXPECTED_SCHEMA_VERSION is the version this code is built for.
const EXPECTED_SCHEMA_VERSION = "2.5.0"

// Schema represents the complete routing-schema.json v2.2.0 structure.
// This defines the tiered agent architecture for Claude Code.
type Schema struct {
	SchemaVersion        string                `json:"$schema"`
	Version              string                `json:"version"`
	Description          string                `json:"description"`
	Updated              string                `json:"updated"`
	Tiers                map[string]TierConfig `json:"tiers"`
	TierLevels           TierLevels            `json:"tier_levels"`
	DelegationCeiling    DelegationCeiling     `json:"delegation_ceiling"`
	ScoutProtocol        ScoutProtocol         `json:"scout_protocol"`
	EscalationRules      EscalationRules       `json:"escalation_rules"`
	CompoundTriggers     CompoundTriggers      `json:"compound_triggers"`
	CostThresholds       CostThresholds        `json:"cost_thresholds"`
	Override             Override              `json:"override"`
	SubagentTypesConfig  SubagentTypesConfig   `json:"subagent_types"`
	DelegationRules      DelegationRules       `json:"delegation_rules"`
	AgentSubagentMapping AgentSubagentMapping  `json:"agent_subagent_mapping"`
	BlockedPatterns      BlockedPatternsConfig `json:"blocked_patterns"`
	DirectImplCheck      DirectImplCheckConfig `json:"direct_impl_check"`
	MetaRules            MetaRules                     `json:"meta_rules"`
	BashBlockedBinaries  map[string]BashBlockedBinary  `json:"bash_blocked_binaries"`
}

// TierConfig defines configuration for each routing tier (haiku, sonnet, opus, etc.).
type TierConfig struct {
	Description               string              `json:"description"`
	Model                     string              `json:"model"`
	Thinking                  bool                `json:"thinking"`
	MaxThinkingBudget         int                 `json:"max_thinking_budget"`
	CostPer1KTokens           float64             `json:"cost_per_1k_tokens"`
	Patterns                  []string            `json:"patterns"`
	Tools                     []string            `json:"tools"`
	Invocation                string              `json:"invocation,omitempty"`
	TaskInvocationBlocked     bool                `json:"task_invocation_blocked,omitempty"`
	TaskInvocationAllowlist   []string            `json:"task_invocation_allowlist,omitempty"`
	EscalationProtocol        string              `json:"escalation_protocol,omitempty"`
	Thresholds                TierThresholds      `json:"thresholds"`
	Agents                    []string            `json:"agents"`
	Protocols                 map[string]Protocol `json:"protocols,omitempty"`
}

// TierThresholds defines limits for each tier.
type TierThresholds struct {
	MaxFiles          *int `json:"max_files"`
	MaxLines          *int `json:"max_lines"`
	MaxTokensEstimate *int `json:"max_tokens_estimate"`
	MinFiles          *int `json:"min_files,omitempty"`
	MinLines          *int `json:"min_lines,omitempty"`
	MinTokensEstimate *int `json:"min_tokens_estimate,omitempty"`
}

// TierLevels defines numeric levels for tier comparison in delegation ceiling.
type TierLevels struct {
	Description   string `json:"description"`
	Haiku         int    `json:"haiku"`
	HaikuThinking int    `json:"haiku_thinking"`
	Sonnet        int    `json:"sonnet"`
	Opus          int    `json:"opus"`
	External      int    `json:"external"`
}

// Protocol defines external model protocol configuration.
type Protocol struct {
	Model  string `json:"model"`
	Output string `json:"output"`
}

// DelegationCeiling controls which agents can be spawned via Task().
type DelegationCeiling struct {
	Description string            `json:"description"`
	File        string            `json:"file"`
	SetBy       string            `json:"set_by"`
	EnforcedBy  string            `json:"enforced_by"`
	Values      []string          `json:"values"`
	Note        string            `json:"note"`
	Override    string            `json:"override"`
	Calculation map[string]string `json:"calculation"`
}

// ScoutProtocol defines pre-routing reconnaissance configuration.
type ScoutProtocol struct {
	Description         string              `json:"description"`
	Primary             string              `json:"primary"`
	Fallback            string              `json:"fallback"`
	SelectionLogic      ScoutSelectionLogic `json:"selection_logic"`
	CostPerCallEstimate float64             `json:"cost_per_call_estimate"`
	Invocation          string              `json:"invocation"`
	OutputSchema        ScoutOutputSchema   `json:"output_schema"`
	WhenToUse           []string            `json:"when_to_use"`
	WhenToSkip          []string            `json:"when_to_skip"`
}

// ScoutSelectionLogic defines logic for choosing between scout types.
type ScoutSelectionLogic struct {
	HaikuScout  ScoutCriteria `json:"haiku_scout"`
	GeminiScout ScoutCriteria `json:"gemini_scout"`
}

// ScoutCriteria defines selection criteria for scout type.
type ScoutCriteria struct {
	MaxFiles  *int   `json:"max_files,omitempty"`
	MinFiles  *int   `json:"min_files,omitempty"`
	MaxTokens *int   `json:"max_tokens,omitempty"`
	MinTokens *int   `json:"min_tokens,omitempty"`
	Reason    string `json:"reason"`
}

// ScoutOutputSchema defines expected scout output structure.
type ScoutOutputSchema struct {
	ScopeMetrics          []string `json:"scope_metrics"`
	ComplexitySignals     []string `json:"complexity_signals"`
	RoutingRecommendation []string `json:"routing_recommendation"`
}

// EscalationRules defines tier-to-tier escalation triggers.
type EscalationRules struct {
	HaikuToHaikuThinking []string         `json:"haiku_to_haiku_thinking"`
	HaikuToSonnet        []string         `json:"haiku_to_sonnet"`
	SonnetToOpus         SonnetToOpusRule `json:"sonnet_to_opus"`
	AnyToExternal        []string         `json:"any_to_external"`
}

// SonnetToOpusRule defines special handling for Opus escalation.
type SonnetToOpusRule struct {
	Triggers     []string `json:"triggers"`
	Action       string   `json:"action"`
	Protocol     string   `json:"protocol"`
	OutputPath   string   `json:"output_path"`
	Notification string   `json:"notification"`
}

// CompoundTriggers defines multi-pattern escalation to orchestrator.
type CompoundTriggers struct {
	Description string     `json:"description"`
	Examples    [][]string `json:"examples"`
	Action      string     `json:"action"`
}

// CostThresholds defines cost ceilings for pre-execution phases.
type CostThresholds struct {
	ScoutMaxCost       float64 `json:"scout_max_cost"`
	ExplorationMaxCost float64 `json:"exploration_max_cost"`
	Description        string  `json:"description"`
}

// Override defines user escape hatch for routing.
type Override struct {
	Flag        string   `json:"flag"`
	Description string   `json:"description"`
	ValidTiers  []string `json:"valid_tiers"`
	AuditLog    string   `json:"audit_log"`
}

// SubagentTypesConfig wraps the subagent_types configuration with its description.
// These are informational groupings — each agent uses its specific CC type name
// from agent_subagent_mapping rather than these generic categories.
type SubagentTypesConfig struct {
	Description    string       `json:"description"`
	Exploration    SubagentType `json:"exploration"`
	Implementation SubagentType `json:"implementation"`
	External       SubagentType `json:"external"`
	Planning       SubagentType `json:"planning"`
	Analysis       SubagentType `json:"analysis"`
}

// SubagentType defines tool capabilities for an agent category.
type SubagentType struct {
	Description string   `json:"description"`
	Tools       []string `json:"tools"`
	AllowsWrite bool     `json:"allows_write"`
	Agents      []string `json:"agents"`
	Rationale   string   `json:"rationale"`
}

// FlexibleSubagentType supports both string and []string JSON unmarshaling
// for backwards compatibility with routing-schema.json.
//
// Accepts:
//   - "codebase-search": "Explore" (single string, backwards compat)
//   - "staff-architect-critical-review": ["Plan", "Explore"] (array, new multi-type)
type FlexibleSubagentType struct {
	types []string
}

// UnmarshalJSON unmarshals either a string or []string into FlexibleSubagentType.
// Tries string first, falls back to []string if that fails.
func (f *FlexibleSubagentType) UnmarshalJSON(data []byte) error {
	// Try unmarshaling as single string first (backwards compatibility)
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		// Reject empty strings (including null, which unmarshals to "")
		if single == "" {
			return fmt.Errorf("FlexibleSubagentType must be string or []string, got null/empty")
		}
		f.types = []string{single}
		return nil
	}

	// Fall back to unmarshaling as []string (multi-type support)
	var multi []string
	if err := json.Unmarshal(data, &multi); err != nil {
		return fmt.Errorf("FlexibleSubagentType must be string or []string")
	}

	if len(multi) == 0 {
		return fmt.Errorf("FlexibleSubagentType array cannot be empty")
	}

	f.types = multi
	return nil
}

// Contains checks if the given subagent_type is in the allowed list.
func (f *FlexibleSubagentType) Contains(subagentType string) bool {
	for _, t := range f.types {
		if t == subagentType {
			return true
		}
	}
	return false
}

// GetAll returns all allowed subagent_types.
// For single-type entries, returns a slice containing that one type.
func (f *FlexibleSubagentType) GetAll() []string {
	result := make([]string, len(f.types))
	copy(result, f.types)
	return result
}

// Primary returns the first/only subagent_type.
// Useful for error messages and single-type compatibility.
func (f *FlexibleSubagentType) Primary() string {
	if len(f.types) == 0 {
		return ""
	}
	return f.types[0]
}

// NewFlexibleSubagentType creates a FlexibleSubagentType from one or more types.
// This is the recommended way to create FlexibleSubagentType in tests and code.
func NewFlexibleSubagentType(types ...string) FlexibleSubagentType {
	if len(types) == 0 {
		return FlexibleSubagentType{types: []string{}}
	}
	return FlexibleSubagentType{types: types}
}

// DelegationRules defines Task() tool permissions.
type DelegationRules struct {
	Description                  string   `json:"description"`
	TaskAlwaysAllowed            bool     `json:"task_always_allowed"`
	TierRestrictionsApplyTo      []string `json:"tier_restrictions_apply_to"`
	TierRestrictionsDoNotApplyTo []string `json:"tier_restrictions_do_not_apply_to"`
	Rationale                    string   `json:"rationale"`
}

// AgentSubagentMapping maps each agent to its required subagent_type.
type AgentSubagentMapping struct {
	Description                  string                `json:"description"`
	CodebaseSearch               FlexibleSubagentType  `json:"codebase-search"`
	HaikuScout                   FlexibleSubagentType  `json:"haiku-scout"`
	CodeReviewer                 FlexibleSubagentType  `json:"code-reviewer"`
	Librarian                    FlexibleSubagentType  `json:"librarian"`
	TechDocsWriter               FlexibleSubagentType  `json:"tech-docs-writer"`
	Scaffolder                   FlexibleSubagentType  `json:"scaffolder"`
	MemoryArchivist              FlexibleSubagentType  `json:"memory-archivist"`
	PythonPro                    FlexibleSubagentType  `json:"python-pro"`
	PythonUX                     FlexibleSubagentType  `json:"python-ux"`
	RPro                         FlexibleSubagentType  `json:"r-pro"`
	RShinyPro                    FlexibleSubagentType  `json:"r-shiny-pro"`
	GoPro                        FlexibleSubagentType  `json:"go-pro"`
	GoCLI                        FlexibleSubagentType  `json:"go-cli"`
	GoTUI                        FlexibleSubagentType  `json:"go-tui"`
	GoAPI                        FlexibleSubagentType  `json:"go-api"`
	GoConcurrent                 FlexibleSubagentType  `json:"go-concurrent"`
	TypescriptPro                FlexibleSubagentType  `json:"typescript-pro"`
	ReactPro                     FlexibleSubagentType  `json:"react-pro"`
	BackendReviewer              FlexibleSubagentType  `json:"backend-reviewer"`
	FrontendReviewer             FlexibleSubagentType  `json:"frontend-reviewer"`
	StandardsReviewer            FlexibleSubagentType  `json:"standards-reviewer"`
	ReviewOrchestrator           FlexibleSubagentType  `json:"review-orchestrator"`
	ImplManager                  FlexibleSubagentType  `json:"impl-manager"`
	Orchestrator                 FlexibleSubagentType  `json:"orchestrator"`
	Architect                    FlexibleSubagentType  `json:"architect"`
	Planner                      FlexibleSubagentType  `json:"planner"`
	PythonArchitect              FlexibleSubagentType  `json:"python-architect"`
	Einstein                     FlexibleSubagentType  `json:"einstein"`
	Mozart                       FlexibleSubagentType  `json:"mozart"`
	Beethoven                    FlexibleSubagentType  `json:"beethoven"`
	GeminiSlave                  FlexibleSubagentType  `json:"gemini-slave"`
	StaffArchitectCriticalReview FlexibleSubagentType  `json:"staff-architect-critical-review"`
}

// BlockedPatternsConfig contains patterns that should never be used.
type BlockedPatternsConfig struct {
	Description string           `json:"description"`
	Patterns    []BlockedPattern `json:"patterns"`
}

// BlockedPattern represents a forbidden pattern with guidance.
type BlockedPattern struct {
	Pattern     string `json:"pattern"`
	Reason      string `json:"reason"`
	Alternative string `json:"alternative"`
	CostImpact  string `json:"cost_impact"`
}

// DirectImplCheckConfig configures detection of direct implementation instead of delegation.
type DirectImplCheckConfig struct {
	Description              string   `json:"description"`
	Enabled                  bool     `json:"enabled"`
	WriteThresholdLines      int      `json:"write_threshold_lines"`
	EditThresholdLines       int      `json:"edit_threshold_lines"`
	ImplementationExtensions []string `json:"implementation_extensions"`
	ImplementationPaths      []string `json:"implementation_paths"`
	ExcludedPatterns         []string `json:"excluded_patterns"`
}

// MetaRules defines meta-enforcement rules.
type MetaRules struct {
	DocumentationTheater DocumentationTheater `json:"documentation_theater"`
}

// BashBlockedBinary defines a binary that must not be invoked directly via Bash.
type BashBlockedBinary struct {
	Reason   string `json:"reason"`
	Redirect string `json:"redirect"`
}

// DocumentationTheater defines detection of unenforceable imperatives.
type DocumentationTheater struct {
	Description       string   `json:"description"`
	DetectionPatterns []string `json:"detection_patterns"`
	TargetFiles       []string `json:"target_files"`
	Enforcement       string   `json:"enforcement"`
	Guidance          string   `json:"guidance"`
}

// LoadSchema loads and validates routing-schema.json.
// Priority: GOGENT_ROUTING_SCHEMA env var > XDG config directory default.
// Returns an error if file is missing, malformed, or version mismatch detected.
func LoadSchema() (*Schema, error) {
	schemaPath := os.Getenv("GOGENT_ROUTING_SCHEMA")

	// If explicit path not set, try project-specific or XDG default
	if schemaPath == "" {
		// Priority 1: GOGENT_PROJECT_DIR (test isolation)
		if projectDir := os.Getenv("GOGENT_PROJECT_DIR"); projectDir != "" {
			path := filepath.Join(projectDir, ".claude", "routing-schema.json")
			if _, err := os.Stat(path); err == nil {
				schemaPath = path
			}
		}

		// Priority 2: XDG default
		if schemaPath == "" {
			configHome := os.Getenv("XDG_CONFIG_HOME")
			if configHome == "" {
				home := os.Getenv("HOME")
				if home == "" {
					return nil, fmt.Errorf("[routing] HOME environment variable not set")
				}
				configHome = filepath.Join(home, ".config")
			}
			schemaPath = filepath.Join(configHome, "..", ".claude", "routing-schema.json")
		}
	}

	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("[routing] Failed to read routing-schema.json from %s: %w", schemaPath, err)
	}

	var schema Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("[routing] Failed to parse routing-schema.json: %w", err)
	}

	// Validate schema version
	if err := schema.Validate(); err != nil {
		return nil, err
	}

	return &schema, nil
}

// Validate performs semantic validation on the loaded schema.
// Checks version compatibility, tier name validity, and reference integrity.
func (s *Schema) Validate() error {
	// Version check
	if s.Version != EXPECTED_SCHEMA_VERSION {
		return fmt.Errorf(
			"[routing] Schema version mismatch: expected %s, got %s. Update code or schema.",
			EXPECTED_SCHEMA_VERSION,
			s.Version,
		)
	}

	// Tier name validation
	validTiers := map[string]bool{
		"haiku":          true,
		"haiku_thinking": true,
		"sonnet":         true,
		"opus":           true,
		"external":       true,
	}

	for tierName := range s.Tiers {
		if !validTiers[tierName] {
			return fmt.Errorf(
				"[routing] Invalid tier name %q. Valid tiers: haiku, haiku_thinking, sonnet, opus, external",
				tierName,
			)
		}
	}

	// Agent-to-subagent mapping reference integrity
	// Each agent now maps to its CC-specific type name
	validSubagentTypes := map[string]bool{
		"Codebase Search":               true,
		"Haiku Scout":                   true,
		"Code Reviewer":                 true,
		"Librarian":                     true,
		"Tech Docs Writer":              true,
		"Scaffolder":                    true,
		"Memory Archivist":              true,
		"Python Pro":                    true,
		"Python UX (PySide6)":           true,
		"R Pro":                         true,
		"R Shiny Pro":                   true,
		"GO Pro":                        true,
		"GO CLI (Cobra)":                true,
		"GO TUI (Bubbletea)":            true,
		"GO API (HTTP Client)":          true,
		"GO Concurrent":                 true,
		"TypeScript Pro":                true,
		"React Pro":                     true,
		"Backend Reviewer":              true,
		"Frontend Reviewer":             true,
		"Standards Reviewer":            true,
		"Review Orchestrator":           true,
		"Implementation Manager":        true,
		"Orchestrator":                  true,
		"Architect":                     true,
		"Planner":                       true,
		"Python ML Architect":           true,
		"Einstein":                      true,
		"Mozart":                        true,
		"Beethoven":                     true,
		"Gemini Slave":                  true,
		"Staff Architect Critical Review": true,
	}

	// Check all agent mappings (each field is now FlexibleSubagentType)
	mappings := []FlexibleSubagentType{
		s.AgentSubagentMapping.CodebaseSearch,
		s.AgentSubagentMapping.HaikuScout,
		s.AgentSubagentMapping.CodeReviewer,
		s.AgentSubagentMapping.Librarian,
		s.AgentSubagentMapping.TechDocsWriter,
		s.AgentSubagentMapping.Scaffolder,
		s.AgentSubagentMapping.MemoryArchivist,
		s.AgentSubagentMapping.PythonPro,
		s.AgentSubagentMapping.PythonUX,
		s.AgentSubagentMapping.RPro,
		s.AgentSubagentMapping.RShinyPro,
		s.AgentSubagentMapping.GoPro,
		s.AgentSubagentMapping.GoCLI,
		s.AgentSubagentMapping.GoTUI,
		s.AgentSubagentMapping.GoAPI,
		s.AgentSubagentMapping.GoConcurrent,
		s.AgentSubagentMapping.TypescriptPro,
		s.AgentSubagentMapping.ReactPro,
		s.AgentSubagentMapping.BackendReviewer,
		s.AgentSubagentMapping.FrontendReviewer,
		s.AgentSubagentMapping.StandardsReviewer,
		s.AgentSubagentMapping.ReviewOrchestrator,
		s.AgentSubagentMapping.ImplManager,
		s.AgentSubagentMapping.Orchestrator,
		s.AgentSubagentMapping.Architect,
		s.AgentSubagentMapping.Planner,
		s.AgentSubagentMapping.PythonArchitect,
		s.AgentSubagentMapping.Einstein,
		s.AgentSubagentMapping.Mozart,
		s.AgentSubagentMapping.Beethoven,
		s.AgentSubagentMapping.GeminiSlave,
		s.AgentSubagentMapping.StaffArchitectCriticalReview,
	}

	// Validate each FlexibleSubagentType's allowed types
	for _, flexType := range mappings {
		types := flexType.GetAll()
		for _, subagentType := range types {
			if subagentType != "" && !validSubagentTypes[subagentType] {
				return fmt.Errorf(
					"[routing] Invalid subagent_type reference: %q",
					subagentType,
				)
			}
		}
	}

	return nil
}

// GetTier returns TierConfig for the given tier name.
// Returns an error if the tier does not exist.
func (s *Schema) GetTier(tierName string) (*TierConfig, error) {
	tier, exists := s.Tiers[tierName]
	if !exists {
		return nil, fmt.Errorf("[routing] Unknown tier: %s", tierName)
	}
	return &tier, nil
}

// agentMapping returns a map of agent name to its FlexibleSubagentType pointer.
// This is the single source of truth for agent-to-subagent-type lookups.
// Adding a new agent requires updating only this method.
func (s *Schema) agentMapping() map[string]*FlexibleSubagentType {
	return map[string]*FlexibleSubagentType{
		"codebase-search":                 &s.AgentSubagentMapping.CodebaseSearch,
		"haiku-scout":                     &s.AgentSubagentMapping.HaikuScout,
		"code-reviewer":                   &s.AgentSubagentMapping.CodeReviewer,
		"librarian":                       &s.AgentSubagentMapping.Librarian,
		"tech-docs-writer":                &s.AgentSubagentMapping.TechDocsWriter,
		"scaffolder":                      &s.AgentSubagentMapping.Scaffolder,
		"memory-archivist":                &s.AgentSubagentMapping.MemoryArchivist,
		"python-pro":                      &s.AgentSubagentMapping.PythonPro,
		"python-ux":                       &s.AgentSubagentMapping.PythonUX,
		"r-pro":                           &s.AgentSubagentMapping.RPro,
		"r-shiny-pro":                     &s.AgentSubagentMapping.RShinyPro,
		"go-pro":                          &s.AgentSubagentMapping.GoPro,
		"go-cli":                          &s.AgentSubagentMapping.GoCLI,
		"go-tui":                          &s.AgentSubagentMapping.GoTUI,
		"go-api":                          &s.AgentSubagentMapping.GoAPI,
		"go-concurrent":                   &s.AgentSubagentMapping.GoConcurrent,
		"typescript-pro":                  &s.AgentSubagentMapping.TypescriptPro,
		"react-pro":                       &s.AgentSubagentMapping.ReactPro,
		"backend-reviewer":                &s.AgentSubagentMapping.BackendReviewer,
		"frontend-reviewer":               &s.AgentSubagentMapping.FrontendReviewer,
		"standards-reviewer":              &s.AgentSubagentMapping.StandardsReviewer,
		"review-orchestrator":             &s.AgentSubagentMapping.ReviewOrchestrator,
		"impl-manager":                    &s.AgentSubagentMapping.ImplManager,
		"orchestrator":                    &s.AgentSubagentMapping.Orchestrator,
		"architect":                       &s.AgentSubagentMapping.Architect,
		"planner":                         &s.AgentSubagentMapping.Planner,
		"python-architect":                &s.AgentSubagentMapping.PythonArchitect,
		"einstein":                        &s.AgentSubagentMapping.Einstein,
		"mozart":                          &s.AgentSubagentMapping.Mozart,
		"beethoven":                       &s.AgentSubagentMapping.Beethoven,
		"gemini-slave":                    &s.AgentSubagentMapping.GeminiSlave,
		"staff-architect-critical-review": &s.AgentSubagentMapping.StaffArchitectCriticalReview,
	}
}

// GetAllowedSubagentTypes returns all allowed subagent_types for an agent.
// For agents with multi-type support, returns all types.
// For single-type agents, returns a slice containing that one type.
// Returns an error if the agent is not in the mapping.
func (s *Schema) GetAllowedSubagentTypes(agentName string) ([]string, error) {
	flexType, exists := s.agentMapping()[agentName]
	if !exists || flexType == nil {
		return nil, fmt.Errorf("[routing] Unknown agent: %s", agentName)
	}

	types := flexType.GetAll()
	if len(types) == 0 {
		return nil, fmt.Errorf("[routing] Agent %s has no subagent types defined", agentName)
	}

	return types, nil
}

// GetSubagentTypeForAgent returns the primary subagent_type for an agent.
// For backwards compatibility, this returns the first type in multi-type mappings.
//
// Deprecated: Use GetAllowedSubagentTypes to get all allowed types.
// This function only returns the primary type and may not represent the full set of allowed types.
func (s *Schema) GetSubagentTypeForAgent(agentName string) (string, error) {
	flexType, exists := s.agentMapping()[agentName]
	if !exists || flexType == nil {
		return "", fmt.Errorf("[routing] Unknown agent: %s", agentName)
	}

	primaryType := flexType.Primary()
	if primaryType == "" {
		return "", fmt.Errorf("[routing] Agent %s has no subagent types defined", agentName)
	}

	return primaryType, nil
}

// GetSubagentType returns SubagentType configuration for an informational category.
// Returns an error if the category does not exist.
func (s *Schema) GetSubagentType(category string) (*SubagentType, error) {
	switch category {
	case "exploration":
		return &s.SubagentTypesConfig.Exploration, nil
	case "implementation":
		return &s.SubagentTypesConfig.Implementation, nil
	case "external":
		return &s.SubagentTypesConfig.External, nil
	case "planning":
		return &s.SubagentTypesConfig.Planning, nil
	case "analysis":
		return &s.SubagentTypesConfig.Analysis, nil
	default:
		return nil, fmt.Errorf("[routing] Unknown subagent category: %s", category)
	}
}

// ValidateAgentSubagentPair checks if agent-subagent_type pairing is valid.
// Returns an error if the pairing violates the mapping in routing-schema.json.
func (s *Schema) ValidateAgentSubagentPair(agentName, subagentType string) error {
	allowedTypes, err := s.GetAllowedSubagentTypes(agentName)
	if err != nil {
		return err
	}

	// Check if subagentType is in the allowed list
	for _, allowedType := range allowedTypes {
		if allowedType == subagentType {
			return nil
		}
	}

	return fmt.Errorf(
		"[routing] Invalid subagent_type for agent %q: expected one of %v, got %q (enforced by routing-schema.json)",
		agentName,
		allowedTypes,
		subagentType,
	)
}

// GetTierLevel returns numeric tier level for comparison.
// Returns an error if the tier does not exist in tier_levels.
func (s *Schema) GetTierLevel(tierName string) (int, error) {
	switch tierName {
	case "haiku":
		return s.TierLevels.Haiku, nil
	case "haiku_thinking":
		return s.TierLevels.HaikuThinking, nil
	case "sonnet":
		return s.TierLevels.Sonnet, nil
	case "opus":
		return s.TierLevels.Opus, nil
	case "external":
		return s.TierLevels.External, nil
	default:
		return 0, fmt.Errorf("[routing] No tier level defined for: %s", tierName)
	}
}

// FormatTierSummary generates a concise routing tier summary for session context.
// Limits patterns to first 3 and tools to first 4 to prevent context bloat.
// Format:
//
//	ROUTING TIERS ACTIVE:
//	  • haiku: patterns=[...] → tools=[...]
//	  • sonnet: patterns=[...] → tools=[...]
//
//	DELEGATION CEILING: Set by {SetBy}
func (s *Schema) FormatTierSummary() string {
	var sb strings.Builder
	sb.WriteString("ROUTING TIERS ACTIVE:\n")

	// Process tiers in order
	tierOrder := []string{"haiku", "haiku_thinking", "sonnet", "opus", "external"}

	for _, tierName := range tierOrder {
		tier, exists := s.Tiers[tierName]
		if !exists {
			continue
		}

		// Truncate patterns to first 3
		patterns := tier.Patterns
		patternsStr := ""
		if len(patterns) > 3 {
			patternsStr = strings.Join(patterns[:3], ", ") + "..."
		} else if len(patterns) > 0 {
			patternsStr = strings.Join(patterns, ", ")
		}

		// Truncate tools to first 4
		tools := tier.Tools
		toolsStr := ""
		if len(tools) > 4 {
			toolsStr = strings.Join(tools[:4], ", ") + "..."
		} else if len(tools) > 0 {
			toolsStr = strings.Join(tools, ", ")
		}

		// Format tier line
		sb.WriteString(fmt.Sprintf("  • %s: patterns=[%s] → tools=[%s]\n",
			tierName, patternsStr, toolsStr))
	}

	// Add delegation ceiling
	sb.WriteString(fmt.Sprintf("\nDELEGATION CEILING: Set by %s\n", s.DelegationCeiling.SetBy))

	return sb.String()
}

// LoadAndFormatSchemaSummary loads routing schema and returns formatted summary.
// Returns a friendly message if schema file is missing (not an error).
// Returns actual errors only for JSON parsing or validation failures.
func LoadAndFormatSchemaSummary() (string, error) {
	schema, err := LoadSchema()
	if err != nil {
		// Check if error is due to missing file
		errMsg := err.Error()
		if strings.Contains(errMsg, "no such file") || strings.Contains(errMsg, "not found") {
			return "[No routing schema found - using defaults]\n", nil
		}
		// Return other errors (parsing, validation)
		return "", err
	}

	return schema.FormatTierSummary(), nil
}
