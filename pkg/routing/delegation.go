package routing

import (
	"fmt"
	"sync"
)

// AgentDelegationConfig holds delegation requirements for an agent.
// Matches must_delegate and min_delegations fields in agents-index.json.
type AgentDelegationConfig struct {
	MustDelegate   bool `json:"must_delegate"`
	MinDelegations int  `json:"min_delegations"`
}

// agentsIndexCache stores the loaded agents index to avoid repeated file reads.
var agentsIndexCache *AgentIndex
var agentsIndexMutex sync.RWMutex

// GetAgentDelegationConfig retrieves delegation requirements for the specified agent.
// Returns zero-value config if agent not found or has no delegation requirements.
// This function never errors - unknown agents are treated as having no requirements.
func GetAgentDelegationConfig(agentID string) AgentDelegationConfig {
	index, err := LoadAgentsIndexCached()
	if err != nil {
		// Fail-open: If we can't load config, assume no delegation requirement
		return AgentDelegationConfig{MustDelegate: false, MinDelegations: 0}
	}

	agent, err := index.GetAgentByID(agentID)
	if err != nil {
		// Unknown agent - no delegation requirement
		return AgentDelegationConfig{MustDelegate: false, MinDelegations: 0}
	}

	// Extract delegation config from agent struct fields
	return AgentDelegationConfig{
		MustDelegate:   agent.MustDelegate,
		MinDelegations: agent.MinDelegations,
	}
}

// LoadAgentsIndexCached loads the agents index with caching.
// Subsequent calls return the cached index without re-reading the file.
// Thread-safe via RWMutex.
func LoadAgentsIndexCached() (*AgentIndex, error) {
	// Fast path: read lock to check cache
	agentsIndexMutex.RLock()
	if agentsIndexCache != nil {
		cached := agentsIndexCache
		agentsIndexMutex.RUnlock()
		return cached, nil
	}
	agentsIndexMutex.RUnlock()

	// Slow path: write lock to load and cache
	agentsIndexMutex.Lock()
	defer agentsIndexMutex.Unlock()

	// Double-check after acquiring write lock
	if agentsIndexCache != nil {
		return agentsIndexCache, nil
	}

	// Load from disk
	index, err := LoadAgentIndex()
	if err != nil {
		return nil, err
	}

	// Cache for future calls
	agentsIndexCache = index
	return index, nil
}

// ClearAgentsIndexCache clears the cached agents index.
// Used by tests to force re-loading of modified index files.
func ClearAgentsIndexCache() {
	agentsIndexMutex.Lock()
	defer agentsIndexMutex.Unlock()
	agentsIndexCache = nil
}

// ValidateDelegationRequirement checks if an agent met its delegation requirements.
// Returns error if agent must delegate but didn't spawn enough child agents.
// Returns nil if requirements are met or agent has no delegation requirement.
func ValidateDelegationRequirement(agentID string, childCount int) error {
	config := GetAgentDelegationConfig(agentID)

	// No delegation requirement
	if !config.MustDelegate {
		return nil
	}

	// Check minimum delegation count
	if childCount < config.MinDelegations {
		return fmt.Errorf(
			"agent %q must delegate to at least %d child agents (actual: %d)",
			agentID,
			config.MinDelegations,
			childCount,
		)
	}

	return nil
}

// BlockResponseForDelegation creates a SubagentStop block response for delegation violations.
// Includes structured hook-specific output with required delegations and suggestions.
func BlockResponseForDelegation(agentID string, required, actual int) *HookResponse {
	reason := fmt.Sprintf(
		"Agent %q requires at least %d delegations but only spawned %d child agents. "+
			"This agent must coordinate work through specialized sub-agents rather than implementing directly.",
		agentID,
		required,
		actual,
	)

	suggestion := fmt.Sprintf(
		"Review %q implementation to ensure it delegates to specialized agents. "+
			"Check agents-index.json for can_spawn list and ensure Task() calls are present.",
		agentID,
	)

	response := NewBlockResponse("SubagentStop", reason)
	response.AddField("agentId", agentID)
	response.AddField("requiredDelegations", required)
	response.AddField("actualDelegations", actual)
	response.AddField("suggestion", suggestion)
	response.AddField("permissionDecision", "deny")
	response.AddField("permissionDecisionReason", reason)

	return response
}

// AllowResponseForDelegation creates a SubagentStop allow response for successful delegation.
// Includes telemetry-friendly structured output for ML analysis.
func AllowResponseForDelegation(agentID string, required, actual int) *HookResponse {
	reason := fmt.Sprintf(
		"Agent %q met delegation requirements (%d delegations, required: %d)",
		agentID,
		actual,
		required,
	)

	response := &HookResponse{
		Decision: DecisionApprove,
		Reason:   reason,
		HookSpecificOutput: map[string]interface{}{
			"hookEventName":            "SubagentStop",
			"agentId":                  agentID,
			"requiredDelegations":      required,
			"actualDelegations":        actual,
			"permissionDecision":       "allow",
			"permissionDecisionReason": reason,
		},
	}

	return response
}

// ValidateDelegationFromTranscript validates delegation requirements using transcript analysis.
// This is the primary entry point for the orchestrator-guard hook.
//
// Process:
// 1. Parse agent metadata from transcript
// 2. Get delegation config for agent
// 3. Validate child count meets requirements
// 4. Return block/allow response
//
// Fail-open behavior: If any step fails (parsing, config load), returns allow response.
func ValidateDelegationFromTranscript(transcriptPath string) (*HookResponse, error) {
	// Parse transcript for agent metadata
	metadata, err := ParseTranscriptForMetadata(transcriptPath)
	if err != nil {
		// Fail-open: Can't parse transcript, allow completion
		return &HookResponse{
			Decision: DecisionApprove,
			Reason:   fmt.Sprintf("Transcript parsing failed: %v", err),
			HookSpecificOutput: map[string]interface{}{
				"hookEventName":            "SubagentStop",
				"permissionDecision":       "allow",
				"permissionDecisionReason": "Fail-open: transcript parsing error",
			},
		}, nil
	}

	// Validate delegation requirement
	config := GetAgentDelegationConfig(metadata.AgentID)
	if !config.MustDelegate {
		// No delegation requirement - allow
		return &HookResponse{
			Decision: DecisionApprove,
			Reason:   fmt.Sprintf("Agent %q has no delegation requirement", metadata.AgentID),
			HookSpecificOutput: map[string]interface{}{
				"hookEventName":            "SubagentStop",
				"agentId":                  metadata.AgentID,
				"permissionDecision":       "allow",
				"permissionDecisionReason": "No delegation requirement",
			},
		}, nil
	}

	// Count child agents from transcript
	childCount, err := CountChildAgentsFromTranscript(transcriptPath)
	if err != nil {
		// Fail-open: Can't count children, allow completion
		return &HookResponse{
			Decision: DecisionApprove,
			Reason:   fmt.Sprintf("Child counting failed: %v", err),
			HookSpecificOutput: map[string]interface{}{
				"hookEventName":            "SubagentStop",
				"agentId":                  metadata.AgentID,
				"permissionDecision":       "allow",
				"permissionDecisionReason": "Fail-open: child counting error",
			},
		}, nil
	}

	// Validate delegation count
	if childCount < config.MinDelegations {
		return BlockResponseForDelegation(metadata.AgentID, config.MinDelegations, childCount), nil
	}

	return AllowResponseForDelegation(metadata.AgentID, config.MinDelegations, childCount), nil
}

// CountChildAgentsFromTranscript counts Task() invocations in the transcript.
// Returns the number of child agents spawned by analyzing tool usage.
// Uses the existing ParseTranscript function which handles JSONL format.
func CountChildAgentsFromTranscript(transcriptPath string) (int, error) {
	events, err := ParseTranscript(transcriptPath)
	if err != nil {
		return 0, fmt.Errorf("failed to parse transcript: %w", err)
	}

	// Count Task tool invocations
	count := 0
	for _, event := range events {
		if event.ToolName == "Task" {
			count++
		}
	}

	return count, nil
}
