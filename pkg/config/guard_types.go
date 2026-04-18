package config

// ActiveSkill represents the v2 guard file written by goyoke-skill-guard when a skill is active.
// Shared between skill-guard (writer) and archive (reader/cleanup).
type ActiveSkill struct {
	FormatVersion      int    `json:"format_version"`
	Skill              string `json:"skill"`
	TeamDir            string `json:"team_dir"`
	RouterAllowedTools []string `json:"router_allowed_tools"`
	CreatedAt          string `json:"created_at"`
	SessionID          string `json:"session_id"`
	HolderPID          int    `json:"holder_pid,omitempty"`
	CCPID              int    `json:"cc_pid,omitempty"`
}
