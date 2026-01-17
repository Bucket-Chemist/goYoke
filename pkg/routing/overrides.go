package routing

import (
	"regexp"
)

// OverrideFlags represents parsed override flags from prompt
type OverrideFlags struct {
	ForceTier       string // e.g., "haiku", "sonnet", "opus"
	ForceDelegation string // e.g., "haiku", "sonnet"
}

// ParseOverrides extracts --force-* flags from Task prompt
func ParseOverrides(prompt string) *OverrideFlags {
	flags := &OverrideFlags{}

	// Match --force-tier=VALUE
	tierRe := regexp.MustCompile(`--force-tier=(\w+)`)
	if match := tierRe.FindStringSubmatch(prompt); len(match) > 1 {
		flags.ForceTier = match[1]
	}

	// Match --force-delegation=VALUE
	delegationRe := regexp.MustCompile(`--force-delegation=(\w+)`)
	if match := delegationRe.FindStringSubmatch(prompt); len(match) > 1 {
		flags.ForceDelegation = match[1]
	}

	return flags
}

// HasOverrides returns true if any overrides are present
func (o *OverrideFlags) HasOverrides() bool {
	return o.ForceTier != "" || o.ForceDelegation != ""
}
