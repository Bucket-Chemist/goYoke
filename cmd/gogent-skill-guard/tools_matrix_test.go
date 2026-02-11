package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestToolsMatrix(t *testing.T) {
	allTools := []string{
		"Task", "Bash", "Read", "Write", "Edit", "Glob", "Grep",
		"AskUserQuestion", "Skill", "NotebookEdit", "WebFetch", "WebSearch",
	}

	type skillDef struct {
		allowed []string
	}

	skills := map[string]skillDef{
		"braintrust": {allowed: []string{"Task", "Bash", "Read", "AskUserQuestion", "Skill"}},
		"review":     {allowed: []string{"Task", "Bash", "Read", "Glob", "AskUserQuestion", "Skill"}},
		"implement":  {allowed: []string{"Task", "Bash", "Read", "AskUserQuestion", "Skill"}},
	}

	contains := func(slice []string, item string) bool {
		for _, s := range slice {
			if s == item {
				return true
			}
		}
		return false
	}

	for skillName, skillCfg := range skills {
		t.Run(skillName, func(t *testing.T) {
			for _, tool := range allTools {
				t.Run(tool, func(t *testing.T) {
					// Create fresh guard for each subtest
					tmpDir := t.TempDir()
					guardPath := filepath.Join(tmpDir, guardFileName)

					guard := &ActiveSkill{
						Skill:              skillName,
						TeamDir:            "/tmp/test-team",
						RouterAllowedTools: skillCfg.allowed,
						CreatedAt:          time.Now().UTC().Format(time.RFC3339),
					}
					writeGuardFile(t, guardPath, guard)

					output := handleGuardMode(tool, guardPath)

					if contains(skillCfg.allowed, tool) {
						assert.Equal(t, "{}", output,
							"tool %s should be ALLOWED for /%s", tool, skillName)
					} else {
						assert.Contains(t, output, "blocked",
							"tool %s should be BLOCKED for /%s", tool, skillName)
						assert.Contains(t, output, skillName,
							"block message should mention skill name %s", skillName)
					}
				})
			}
		})
	}
}
