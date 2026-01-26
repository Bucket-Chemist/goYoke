package routing

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// EXPECTED_SCHEMA_VERSION is the version this code is built for.
const EXPECTED_SCHEMA_VERSION = "2.2.0"

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
	MetaRules            MetaRules             `json:"meta_rules"`
}

// TierConfig defines configuration for each routing tier (haiku, sonnet, opus, etc.).
type TierConfig struct {
	Description           string              `json:"description"`
	Model                 string              `json:"model"`
	Thinking              bool                `json:"thinking"`
	MaxThinkingBudget     int                 `json:"max_thinking_budget"`
	CostPer1KTokens       float64             `json:"cost_per_1k_tokens"`
	Patterns              []string            `json:"patterns"`
	Tools                 []string            `json:"tools"`
	Invocation            string              `json:"invocation,omitempty"`
	TaskInvocationBlocked bool                `json:"task_invocation_blocked,omitempty"`
	EscalationProtocol    string              `json:"escalation_protocol,omitempty"`
	Thresholds            TierThresholds      `json:"thresholds"`
	Agents                []string            `json:"agents"`
	Protocols             map[string]Protocol `json:"protocols,omitempty"`
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
type SubagentTypesConfig struct {
	Description    string       `json:"description"`
	Explore        SubagentType `json:"Explore"`
	GeneralPurpose SubagentType `json:"general-purpose"`
	Bash           SubagentType `json:"Bash"`
	Plan           SubagentType `json:"Plan"`
}

// SubagentType defines tool capabilities for each subagent_type.
type SubagentType struct {
	Description       string   `json:"description"`
	Tools             []string `json:"tools"`
	AllowsWrite       bool     `json:"allows_write"`
	RespectsAgentYaml bool     `json:"respects_agent_yaml"`
	UseFor            []string `json:"use_for"`
	Rationale         string   `json:"rationale"`
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
	Description                  string `json:"description"`
	CodebaseSearch               string `json:"codebase-search"`
	HaikuScout                   string `json:"haiku-scout"`
	CodeReviewer                 string `json:"code-reviewer"`
	Librarian                    string `json:"librarian"`
	TechDocsWriter               string `json:"tech-docs-writer"`
	Scaffolder                   string `json:"scaffolder"`
	MemoryArchivist              string `json:"memory-archivist"`
	PythonPro                    string `json:"python-pro"`
	PythonUX                     string `json:"python-ux"`
	RPro                         string `json:"r-pro"`
	RShinyPro                    string `json:"r-shiny-pro"`
	GoPro                        string `json:"go-pro"`
	GoCLI                        string `json:"go-cli"`
	GoTUI                        string `json:"go-tui"`
	GoAPI                        string `json:"go-api"`
	GoConcurrent                 string `json:"go-concurrent"`
	Orchestrator                 string `json:"orchestrator"`
	Architect                    string `json:"architect"`
	Einstein                     string `json:"einstein"`
	GeminiSlave                  string `json:"gemini-slave"`
	StaffArchitectCriticalReview string `json:"staff-architect-critical-review"`
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

	// Fall back to XDG default if env var not set
	if schemaPath == "" {
		configHome := os.Getenv("XDG_CONFIG_HOME")
		if configHome == "" {
			home := os.Getenv("HOME")
			if home == "" {
				return nil, fmt.Errorf("[routing] HOME environment variable not set")
			}
			configHome = home + "/.config"
		}
		schemaPath = configHome + "/../.claude/routing-schema.json"
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
	validSubagentTypes := map[string]bool{
		"Explore":         true,
		"general-purpose": true,
		"Bash":            true,
		"Plan":            true,
	}

	// Check all agent mappings
	mappings := []string{
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
		s.AgentSubagentMapping.Orchestrator,
		s.AgentSubagentMapping.Architect,
		s.AgentSubagentMapping.Einstein,
		s.AgentSubagentMapping.GeminiSlave,
		s.AgentSubagentMapping.StaffArchitectCriticalReview,
	}

	for _, subagentType := range mappings {
		if subagentType != "" && !validSubagentTypes[subagentType] {
			return fmt.Errorf(
				"[routing] Invalid subagent_type reference: %q",
				subagentType,
			)
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

// GetSubagentTypeForAgent returns the required subagent_type for an agent.
// Returns an error if the agent is not in the mapping.
func (s *Schema) GetSubagentTypeForAgent(agentName string) (string, error) {
	mapping := map[string]string{
		"codebase-search":                 s.AgentSubagentMapping.CodebaseSearch,
		"haiku-scout":                     s.AgentSubagentMapping.HaikuScout,
		"code-reviewer":                   s.AgentSubagentMapping.CodeReviewer,
		"librarian":                       s.AgentSubagentMapping.Librarian,
		"tech-docs-writer":                s.AgentSubagentMapping.TechDocsWriter,
		"scaffolder":                      s.AgentSubagentMapping.Scaffolder,
		"memory-archivist":                s.AgentSubagentMapping.MemoryArchivist,
		"python-pro":                      s.AgentSubagentMapping.PythonPro,
		"python-ux":                       s.AgentSubagentMapping.PythonUX,
		"r-pro":                           s.AgentSubagentMapping.RPro,
		"r-shiny-pro":                     s.AgentSubagentMapping.RShinyPro,
		"go-pro":                          s.AgentSubagentMapping.GoPro,
		"go-cli":                          s.AgentSubagentMapping.GoCLI,
		"go-tui":                          s.AgentSubagentMapping.GoTUI,
		"go-api":                          s.AgentSubagentMapping.GoAPI,
		"go-concurrent":                   s.AgentSubagentMapping.GoConcurrent,
		"orchestrator":                    s.AgentSubagentMapping.Orchestrator,
		"architect":                       s.AgentSubagentMapping.Architect,
		"einstein":                        s.AgentSubagentMapping.Einstein,
		"gemini-slave":                    s.AgentSubagentMapping.GeminiSlave,
		"staff-architect-critical-review": s.AgentSubagentMapping.StaffArchitectCriticalReview,
	}

	subagentType, exists := mapping[agentName]
	if !exists || subagentType == "" {
		return "", fmt.Errorf("[routing] Unknown agent: %s", agentName)
	}
	return subagentType, nil
}

// GetSubagentType returns SubagentType configuration.
// Returns an error if the subagent_type does not exist.
func (s *Schema) GetSubagentType(subagentType string) (*SubagentType, error) {
	switch subagentType {
	case "Explore":
		return &s.SubagentTypesConfig.Explore, nil
	case "general-purpose":
		return &s.SubagentTypesConfig.GeneralPurpose, nil
	case "Bash":
		return &s.SubagentTypesConfig.Bash, nil
	case "Plan":
		return &s.SubagentTypesConfig.Plan, nil
	default:
		return nil, fmt.Errorf("[routing] Unknown subagent_type: %s", subagentType)
	}
}

// ValidateAgentSubagentPair checks if agent-subagent_type pairing is valid.
// Returns an error if the pairing violates the mapping in routing-schema.json.
func (s *Schema) ValidateAgentSubagentPair(agentName, subagentType string) error {
	requiredType, err := s.GetSubagentTypeForAgent(agentName)
	if err != nil {
		return err
	}

	if requiredType != subagentType {
		return fmt.Errorf(
			"[routing] Invalid subagent_type for agent %q: got %q, expected %q (enforced by routing-schema.json)",
			agentName,
			subagentType,
			requiredType,
		)
	}

	return nil
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
