package permissiongate

// Classification constants returned by Classify.
const (
	ClassAutoAllow     = "auto_allow"
	ClassNeedsApproval = "needs_approval"
	ClassSkip          = "skip"
)

// Policy defines per-tool classification rules for the permission gate.
type Policy struct {
	AutoAllow     []string
	NeedsApproval []string
	Skip          []string
	Default       string
}

// DefaultPolicy is the hardcoded permission policy compiled into the binary.
var DefaultPolicy = Policy{
	AutoAllow: []string{
		"Read", "Glob", "Grep", "TodoWrite", "EnterPlanMode",
		"ExitPlanMode", "WebSearch", "WebFetch", "ToolSearch",
		"AskUserQuestion", "Skill", "Write", "Edit", "NotebookEdit",
	},
	NeedsApproval: []string{},
	Skip:          []string{"Task", "Agent", "Bash"},
	Default:       ClassAutoAllow,
}

// Classify returns the classification for toolName according to p.
func (p *Policy) Classify(toolName string) string {
	for _, t := range p.NeedsApproval {
		if t == toolName {
			return ClassNeedsApproval
		}
	}
	for _, t := range p.Skip {
		if t == toolName {
			return ClassSkip
		}
	}
	for _, t := range p.AutoAllow {
		if t == toolName {
			return ClassAutoAllow
		}
	}
	return p.Default
}
