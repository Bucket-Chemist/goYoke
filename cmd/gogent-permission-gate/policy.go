package main

// Classification constants returned by Classify.
const (
	classAutoAllow     = "auto_allow"
	classNeedsApproval = "needs_approval"
	classSkip          = "skip"
)

// Policy defines per-tool classification rules for the permission gate.
type Policy struct {
	// AutoAllow is the list of tool names that are always permitted without
	// prompting the user.
	AutoAllow []string
	// NeedsApproval is the list of tool names that require explicit user
	// approval via the TUI modal.
	NeedsApproval []string
	// Skip is the list of tool names that bypass the gate entirely (neither
	// allowed nor blocked by this hook).
	Skip []string
	// Default is the classification applied when a tool name does not appear
	// in any of the above lists.  Valid values are the class* constants.
	Default string
}

// defaultPolicy is the hardcoded permission policy.  No external config file
// is required; the policy is compiled into the binary.
var defaultPolicy = Policy{
	AutoAllow: []string{
		"Read", "Glob", "Grep", "TodoWrite", "EnterPlanMode",
		"ExitPlanMode", "WebSearch", "WebFetch", "ToolSearch",
		"AskUserQuestion", "Skill", "Write", "Edit", "NotebookEdit",
	},
	NeedsApproval: []string{},
	Skip:          []string{"Task", "Agent", "Bash"},
	Default:       classAutoAllow,
}

// Classify returns the classification for toolName according to p.
// It checks NeedsApproval first, then Skip, then AutoAllow, then falls back to
// Default.  The precedence ensures that explicit approval requirements are
// never silently bypassed by the auto-allow list.
func (p *Policy) Classify(toolName string) string {
	for _, t := range p.NeedsApproval {
		if t == toolName {
			return classNeedsApproval
		}
	}
	for _, t := range p.Skip {
		if t == toolName {
			return classSkip
		}
	}
	for _, t := range p.AutoAllow {
		if t == toolName {
			return classAutoAllow
		}
	}
	return p.Default
}
