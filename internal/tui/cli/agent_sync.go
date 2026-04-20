package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bucket-Chemist/goYoke/internal/tui/state"
	"github.com/Bucket-Chemist/goYoke/internal/tui/util"
)

// ---------------------------------------------------------------------------
// AgentSyncResult
// ---------------------------------------------------------------------------

// AgentSyncResult holds the outcomes of processing a single CLI event.
// Registered, Updated, and Activity contain the IDs of agents that were
// affected; callers use these to drive downstream Bubbletea messages.
type AgentSyncResult struct {
	// Registered contains the IDs of newly registered agents.
	Registered []string
	// Updated contains the IDs of agents whose status changed.
	Updated []string
	// Activity contains the IDs of agents whose activity was set.
	Activity []string
}

// ---------------------------------------------------------------------------
// taskInputSchema
// ---------------------------------------------------------------------------

// taskInputSchema is the expected JSON shape of a Task tool_use Input field.
type taskInputSchema struct {
	Description  string `json:"description"`
	SubagentType string `json:"subagent_type"`
	Model        string `json:"model"`
	Prompt       string `json:"prompt"`
}

// ---------------------------------------------------------------------------
// SyncAssistantEvent
// ---------------------------------------------------------------------------

// SyncAssistantEvent processes an AssistantEvent and returns mutations to apply.
//
// It scans ev.Message.Content for:
//   - tool_use blocks where Name == "Task": parses agent metadata and calls
//     registry.Register. The tool_use ID becomes the canonical agent ID.
//   - tool_use blocks where Name != "Task" and ev.ParentToolUseID != nil:
//     activity for the parent subagent; calls registry.SetActivity.
//
// Root-level non-Task tool_use blocks (ParentToolUseID == nil) are skipped
// because the root agent is the CLI process itself, not a tracked subagent.
func SyncAssistantEvent(ev AssistantEvent, registry *state.AgentRegistry) AgentSyncResult {
	var result AgentSyncResult

	for _, block := range ev.Message.Content {
		if block.Type != "tool_use" {
			continue
		}

		if block.Name == "Task" {
			agent, ok := ParseTaskInput(block.Input)
			if !ok {
				continue
			}

			// Use the tool_use ID as the canonical agent ID.
			agent.ID = block.ID

			// Link to parent agent if this is a subagent spawn.
			if ev.ParentToolUseID != nil {
				agent.ParentID = *ev.ParentToolUseID
			}

			agent.Status = state.StatusRunning
			agent.StartedAt = time.Now()

			if err := registry.Register(agent); err != nil {
				// ErrDuplicateAgent: skip silently — idempotent.
				continue
			}

			result.Registered = append(result.Registered, agent.ID)
			continue
		}

		// Non-Task tool_use: activity for the owning subagent.
		if ev.ParentToolUseID == nil {
			// Root-level tool use — not tracked as subagent activity.
			continue
		}

		root, _ := os.Getwd()
		activities := ExtractToolActivities(block, root)
		for _, activity := range activities {
			registry.AppendActivity(*ev.ParentToolUseID, activity)
		}
		if len(activities) > 0 {
			result.Activity = append(result.Activity, *ev.ParentToolUseID)
		}
	}

	return result
}

// ---------------------------------------------------------------------------
// SyncUserEvent
// ---------------------------------------------------------------------------

// SyncUserEvent processes a UserEvent (tool results) and returns mutations.
//
// It scans ev.Message.Content for tool_result blocks whose ToolUseID matches a
// registered agent ID. When found and the agent is StatusRunning:
//   - IsError → StatusError
//   - Otherwise → StatusComplete
//
// If ev.ParentToolUseID is non-nil the tool result belongs to a subagent turn;
// the parent's activity is cleared to signal it is idle again.
//
// Orphaned tool_result blocks (ToolUseID not in registry) are ignored.
func SyncUserEvent(ev UserEvent, registry *state.AgentRegistry) AgentSyncResult {
	var result AgentSyncResult

	for _, block := range ev.Message.Content {
		if block.Type != "tool_result" {
			continue
		}

		id := block.ToolUseID
		if id == "" {
			continue
		}

		agent := registry.Get(id)
		if agent == nil {
			// Orphaned tool_result — not a tracked agent.
			continue
		}

		if agent.Status != state.StatusRunning {
			continue
		}

		targetStatus := state.StatusComplete
		if block.IsError {
			targetStatus = state.StatusError
		}

		if err := registry.Update(id, func(a *state.Agent) {
			a.Status = targetStatus
		}); err != nil {
			// Invalid transition or agent gone — skip.
			continue
		}

		result.Updated = append(result.Updated, id)
	}

	// If this result belongs to a subagent turn, clear the parent's activity.
	if ev.ParentToolUseID != nil {
		parentID := *ev.ParentToolUseID
		if registry.Get(parentID) != nil {
			registry.AppendActivity(parentID, state.AgentActivity{
				Type:      "idle",
				Timestamp: time.Now(),
			})
			result.Activity = append(result.Activity, parentID)
		}
	}

	return result
}

// ---------------------------------------------------------------------------
// ParseTaskInput
// ---------------------------------------------------------------------------

// ParseTaskInput extracts agent metadata from a Task tool_use Input JSON.
// It returns a partially populated Agent and true on success, or a zero-value
// Agent and false when the input cannot be decoded or is missing required fields.
//
// Populated fields: Description, AgentType, Model, Tier.
// The caller is responsible for setting ID, ParentID, Status, and StartedAt.
func ParseTaskInput(input json.RawMessage) (state.Agent, bool) {
	if len(input) == 0 {
		return state.Agent{}, false
	}

	var schema taskInputSchema
	if err := json.Unmarshal(input, &schema); err != nil {
		return state.Agent{}, false
	}

	agent := state.Agent{
		Description: schema.Description,
		AgentType:   normaliseAgentType(schema.SubagentType),
		Model:       schema.Model,
		Tier:        modelToTier(schema.Model),
	}

	return agent, true
}

// normaliseAgentType converts a subagent_type string (e.g. "GO Pro") to the
// lowercase kebab-case ID form (e.g. "go-pro") used in agents-index.json.
func normaliseAgentType(raw string) string {
	if raw == "" {
		return ""
	}
	return strings.ToLower(strings.ReplaceAll(raw, " ", "-"))
}

// modelToTier maps a model name string to its cost tier label.
func modelToTier(model string) string {
	lower := strings.ToLower(model)
	switch {
	case strings.Contains(lower, "haiku"):
		return "haiku"
	case strings.Contains(lower, "opus"):
		return "opus"
	case strings.Contains(lower, "sonnet"):
		return "sonnet"
	default:
		return lower
	}
}

// ---------------------------------------------------------------------------
// stripProjectRoot / splitCompoundCommand
// ---------------------------------------------------------------------------

// stripProjectRoot returns path relative to root. If path is outside root, if
// root is empty, or if filepath.Rel fails, the original path is returned unchanged.
func stripProjectRoot(path, root string) string {
	if root == "" || path == "" {
		return path
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	// A ".." prefix means the path escapes the root — return unchanged.
	if strings.HasPrefix(rel, "..") {
		return path
	}
	return rel
}

// splitCompoundCommand splits a shell command on "&&" and returns each trimmed
// non-empty part.
func splitCompoundCommand(cmd string) []string {
	parts := strings.Split(cmd, "&&")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// ExtractToolActivities
// ---------------------------------------------------------------------------

// ExtractToolActivities creates AgentActivity items from a non-Task tool_use
// block. The activity Type is always "tool_use".
//
// For Bash commands containing "&&" each sub-command becomes its own entry,
// allowing the rolling activity buffer to show individual steps.
//
// root is the project root directory used to strip absolute path prefixes from
// file-path targets (Read, Write, Edit). Pass an empty string to disable
// stripping.
func ExtractToolActivities(block ContentBlock, root string) []state.AgentActivity {
	now := time.Now()

	// Bash compound commands: split on && and return one activity per part.
	if block.Name == "Bash" && len(block.Input) > 0 {
		var fields struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal(block.Input, &fields); err == nil && strings.Contains(fields.Command, "&&") {
			parts := splitCompoundCommand(fields.Command)
			if len(parts) > 1 {
				activities := make([]state.AgentActivity, 0, len(parts))
				for _, part := range parts {
					truncated := util.Truncate(part, 80)
					activities = append(activities, state.AgentActivity{
						Type:      "tool_use",
						ToolName:  "Bash",
						ToolID:    block.ID,
						Target:    truncated,
						Preview:   "Bash: " + truncated,
						Timestamp: now,
					})
				}
				return activities
			}
		}
	}

	// Default: single activity.
	rawTarget := extractToolTarget(block.Name, block.Input)

	// Strip project root for tools whose target is a file path.
	target := rawTarget
	switch block.Name {
	case "Read", "Write", "Edit":
		target = stripProjectRoot(rawTarget, root)
	}

	preview := block.Name
	if target != "" {
		preview = block.Name + ": " + target
	}

	return []state.AgentActivity{{
		Type:      "tool_use",
		ToolName:  block.Name,
		ToolID:    block.ID,
		Target:    target,
		Preview:   preview,
		Timestamp: now,
	}}
}

// ---------------------------------------------------------------------------
// extractToolTarget
// ---------------------------------------------------------------------------

// extractToolTarget parses the tool input JSON and returns a human-readable
// target string (file path, command, pattern, URL, etc.) appropriate for the
// named tool.
//
// For unrecognised tools it falls back to the tool name itself. Long strings
// are truncated to 80 characters to keep one-line previews readable.
func extractToolTarget(toolName string, input json.RawMessage) string {
	if len(input) == 0 {
		return toolName
	}

	// Generic container for all known input shapes — unmarshal once.
	var fields struct {
		FilePath string `json:"file_path"`
		Command  string `json:"command"`
		Pattern  string `json:"pattern"`
		URL      string `json:"url"`
		Query    string `json:"query"`
	}
	if err := json.Unmarshal(input, &fields); err != nil {
		return toolName
	}

	var target string
	switch toolName {
	case "Read", "Write", "Edit":
		target = fields.FilePath
	case "Bash":
		target = util.Truncate(fields.Command, 80)
	case "Grep":
		target = fields.Pattern
	case "Glob":
		target = fields.Pattern
	case "WebFetch":
		target = fields.URL
	case "WebSearch":
		target = fields.Query
	default:
		target = toolName
	}

	return target
}

