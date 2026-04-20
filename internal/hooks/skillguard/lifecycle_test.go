package skillguard

import (
	"os"

	"github.com/Bucket-Chemist/goYoke/pkg/config"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertToolAllowed verifies that a tool is allowed through the guard.
func assertToolAllowed(t *testing.T, guardPath, tool string) {
	t.Helper()
	output := legacyCheckGuard(tool, guardPath)
	assert.Equal(t, "{}", output, "tool %s should be allowed", tool)
}

// assertToolBlocked verifies that a tool is blocked by the guard.
func assertToolBlocked(t *testing.T, guardPath, tool, skill string) {
	t.Helper()
	output := legacyCheckGuard(tool, guardPath)
	assert.Contains(t, output, "blocked", "tool %s should be blocked during /%s", tool, skill)
	assert.Contains(t, output, skill)
}

// TestLifecycle_ReviewFull tests the complete lifecycle of a review skill guard.
func TestLifecycle_ReviewFull(t *testing.T) {
	tmpDir := t.TempDir()
	guardPath := filepath.Join(tmpDir, guardFileName)

	// Phase 1: Setup - Write review guard
	guard := &config.ActiveSkill{
		Skill:   "review",
		TeamDir: "/tmp/test-team",
		RouterAllowedTools: []string{
			"Task", "Bash", "Read", "Glob", "AskUserQuestion", "Skill",
		},
		CreatedAt: time.Now().Format(time.RFC3339),
	}
	writeGuardFile(t, guardPath, guard)

	// Phase 2: Active skill - Verify allowed tools
	assertToolAllowed(t, guardPath, "Task")
	assertToolAllowed(t, guardPath, "Bash")
	assertToolAllowed(t, guardPath, "Read")
	assertToolAllowed(t, guardPath, "Glob")
	assertToolAllowed(t, guardPath, "AskUserQuestion")
	assertToolAllowed(t, guardPath, "Skill")

	// Phase 3: Active skill - Verify blocked tools
	assertToolBlocked(t, guardPath, "Write", "review")
	assertToolBlocked(t, guardPath, "Edit", "review")

	// Phase 4: Cleanup - Remove guard file (simulating cleanup)
	require.NoError(t, os.Remove(guardPath))

	// Phase 5: Post-cleanup - Verify all tools pass through
	assertToolAllowed(t, guardPath, "Glob")
	assertToolAllowed(t, guardPath, "Write")
	assertToolAllowed(t, guardPath, "Edit")
}

// TestLifecycle_ImplementFull tests the complete lifecycle of an implement skill guard.
func TestLifecycle_ImplementFull(t *testing.T) {
	tmpDir := t.TempDir()
	guardPath := filepath.Join(tmpDir, guardFileName)

	// Phase 1: Setup - Write implement guard
	guard := &config.ActiveSkill{
		Skill:   "implement",
		TeamDir: "/tmp/test-team",
		RouterAllowedTools: []string{
			"Task", "Bash", "Read", "AskUserQuestion", "Skill",
		},
		CreatedAt: time.Now().Format(time.RFC3339),
	}
	writeGuardFile(t, guardPath, guard)

	// Phase 2: Active skill - Verify allowed tools
	assertToolAllowed(t, guardPath, "Task")
	assertToolAllowed(t, guardPath, "Bash")
	assertToolAllowed(t, guardPath, "Read")
	assertToolAllowed(t, guardPath, "AskUserQuestion")
	assertToolAllowed(t, guardPath, "Skill")

	// Phase 3: Active skill - Verify blocked tools
	assertToolBlocked(t, guardPath, "Glob", "implement")
	assertToolBlocked(t, guardPath, "Write", "implement")
	assertToolBlocked(t, guardPath, "Edit", "implement")

	// Phase 4: Cleanup - Remove guard file
	require.NoError(t, os.Remove(guardPath))

	// Phase 5: Post-cleanup - Verify all tools pass through
	assertToolAllowed(t, guardPath, "Glob")
	assertToolAllowed(t, guardPath, "Write")
}

// TestLifecycle_BraintrustFull tests the complete lifecycle of a braintrust skill guard.
func TestLifecycle_BraintrustFull(t *testing.T) {
	tmpDir := t.TempDir()
	guardPath := filepath.Join(tmpDir, guardFileName)

	// Phase 1: Setup - Write braintrust guard
	guard := &config.ActiveSkill{
		Skill:   "braintrust",
		TeamDir: "/tmp/test-team",
		RouterAllowedTools: []string{
			"Task", "Bash", "Read", "AskUserQuestion", "Skill",
		},
		CreatedAt: time.Now().Format(time.RFC3339),
	}
	writeGuardFile(t, guardPath, guard)

	// Phase 2: Active skill - Verify allowed tools
	assertToolAllowed(t, guardPath, "Task")
	assertToolAllowed(t, guardPath, "Bash")
	assertToolAllowed(t, guardPath, "Read")
	assertToolAllowed(t, guardPath, "AskUserQuestion")
	assertToolAllowed(t, guardPath, "Skill")

	// Phase 3: Active skill - Verify blocked tools
	assertToolBlocked(t, guardPath, "Glob", "braintrust")
	assertToolBlocked(t, guardPath, "Write", "braintrust")
	assertToolBlocked(t, guardPath, "Edit", "braintrust")
	assertToolBlocked(t, guardPath, "Grep", "braintrust")

	// Phase 4: Cleanup - Remove guard file
	require.NoError(t, os.Remove(guardPath))

	// Phase 5: Post-cleanup - Verify all tools pass through
	assertToolAllowed(t, guardPath, "Glob")
	assertToolAllowed(t, guardPath, "Write")
	assertToolAllowed(t, guardPath, "Edit")
}

// TestLifecycle_SkillSwitchOverwrite tests that one skill's guard cleanly replaces another's.
func TestLifecycle_SkillSwitchOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	guardPath := filepath.Join(tmpDir, guardFileName)

	// Phase 1: Setup review skill
	reviewGuard := &config.ActiveSkill{
		Skill:   "review",
		TeamDir: "/tmp/test-team",
		RouterAllowedTools: []string{
			"Task", "Bash", "Read", "Glob", "AskUserQuestion", "Skill",
		},
		CreatedAt: time.Now().Format(time.RFC3339),
	}
	writeGuardFile(t, guardPath, reviewGuard)

	// Phase 2: Verify review allows Glob
	assertToolAllowed(t, guardPath, "Glob")

	// Phase 3: Overwrite with braintrust skill
	braintrustGuard := &config.ActiveSkill{
		Skill:   "braintrust",
		TeamDir: "/tmp/test-team-2",
		RouterAllowedTools: []string{
			"Task", "Bash", "Read", "AskUserQuestion", "Skill",
		},
		CreatedAt: time.Now().Format(time.RFC3339),
	}
	writeGuardFile(t, guardPath, braintrustGuard)

	// Phase 4: Verify braintrust blocks Glob
	assertToolBlocked(t, guardPath, "Glob", "braintrust")

	// Phase 5: Verify braintrust allows its tools
	assertToolAllowed(t, guardPath, "Task")
	assertToolAllowed(t, guardPath, "Bash")
}

// TestLifecycle_CleanupOnError tests that cleanup works correctly after error conditions.
func TestLifecycle_CleanupOnError(t *testing.T) {
	tmpDir := t.TempDir()
	guardPath := filepath.Join(tmpDir, guardFileName)

	// Phase 1: Setup guard
	guard := &config.ActiveSkill{
		Skill:   "review",
		TeamDir: "/tmp/test-team",
		RouterAllowedTools: []string{
			"Task", "Bash",
		},
		CreatedAt: time.Now().Format(time.RFC3339),
	}
	writeGuardFile(t, guardPath, guard)

	// Phase 2: Verify guard is active
	assertToolBlocked(t, guardPath, "Write", "review")

	// Phase 3: Simulate error cleanup - remove guard file
	require.NoError(t, os.Remove(guardPath))

	// Phase 4: Verify all tools pass through (no guard = no blocking)
	assertToolAllowed(t, guardPath, "Task")
	assertToolAllowed(t, guardPath, "Bash")
	assertToolAllowed(t, guardPath, "Write")
	assertToolAllowed(t, guardPath, "Edit")
	assertToolAllowed(t, guardPath, "Glob")
	assertToolAllowed(t, guardPath, "Grep")
}

// TestLifecycle_StalenessAutoCleanup tests that stale guards are automatically deleted.
func TestLifecycle_StalenessAutoCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	guardPath := filepath.Join(tmpDir, guardFileName)

	// Phase 1: Setup guard with CreatedAt 31 minutes in the past
	staleTime := time.Now().Add(-31 * time.Minute)
	guard := &config.ActiveSkill{
		Skill:   "review",
		TeamDir: "/tmp/test-team",
		RouterAllowedTools: []string{
			"Task", "Bash",
		},
		CreatedAt: staleTime.Format(time.RFC3339),
	}
	writeGuardFile(t, guardPath, guard)

	// Phase 2: Verify guard file exists
	_, err := os.Stat(guardPath)
	require.NoError(t, err, "guard file should exist before staleness check")

	// Phase 3: Call legacyCheckGuard - should auto-delete stale guard
	output := legacyCheckGuard("Write", guardPath)
	assert.Equal(t, "{}", output, "stale guard should be deleted, tool allowed")

	// Phase 4: Verify guard file was deleted
	_, err = os.Stat(guardPath)
	assert.True(t, os.IsNotExist(err), "stale guard file should be deleted")

	// Phase 5: Verify subsequent calls still return {} (no guard)
	output = legacyCheckGuard("Write", guardPath)
	assert.Equal(t, "{}", output, "after deletion, all tools should pass through")

	output = legacyCheckGuard("Edit", guardPath)
	assert.Equal(t, "{}", output, "after deletion, all tools should pass through")
}
