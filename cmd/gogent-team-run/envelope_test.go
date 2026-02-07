package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildEnvelope_ValidStdin(t *testing.T) {
	teamDir := t.TempDir()

	// Create valid stdin file
	stdinPath := filepath.Join(teamDir, "stdin_worker.json")
	stdinContent := `{
  "$schema": "v1",
  "agent": "go-pro",
  "context": {
    "project_root": "/home/user/project",
    "team_dir": "/home/user/project/teams/test"
  },
  "task": "Implement the handler function",
  "constraints": ["Must use error wrapping", "Must be thread-safe"]
}`
	require.NoError(t, os.WriteFile(stdinPath, []byte(stdinContent), 0644))

	member := &Member{
		Name:       "worker-1",
		Agent:      "go-pro",
		StdinFile:  "stdin_worker.json",
		StdoutFile: "stdout_worker.json",
	}

	envelope, err := buildPromptEnvelope(teamDir, member)
	require.NoError(t, err)

	// Verify envelope contains all expected sections
	assert.Contains(t, envelope, "AGENT: go-pro")
	assert.Contains(t, envelope, "Stdin Envelope")
	assert.Contains(t, envelope, "Implement the handler function")
	assert.Contains(t, envelope, "nesting level 2")
	assert.Contains(t, envelope, `Task(model: "haiku")`)
	assert.Contains(t, envelope, `Task(model: "sonnet")`)
	assert.Contains(t, envelope, `Task(model: "opus") — BLOCKED`)
	assert.Contains(t, envelope, "Must use error wrapping")
	assert.Contains(t, envelope, "Must be thread-safe")
}

func TestBuildEnvelope_EmptyTask(t *testing.T) {
	teamDir := t.TempDir()

	// Create stdin with empty task field
	stdinPath := filepath.Join(teamDir, "stdin_empty_task.json")
	stdinContent := `{
  "$schema": "v1",
  "agent": "go-pro",
  "context": {
    "project_root": "/home/user/project"
  },
  "task": ""
}`
	require.NoError(t, os.WriteFile(stdinPath, []byte(stdinContent), 0644))

	member := &Member{
		Name:       "worker-1",
		Agent:      "go-pro",
		StdinFile:  "stdin_empty_task.json",
		StdoutFile: "stdout.json",
	}

	_, err := buildPromptEnvelope(teamDir, member)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task field is empty")
}

func TestBuildEnvelope_EmptyContext(t *testing.T) {
	teamDir := t.TempDir()

	// Create stdin with empty context field
	stdinPath := filepath.Join(teamDir, "stdin_empty_context.json")
	stdinContent := `{
  "$schema": "v1",
  "agent": "go-pro",
  "context": {},
  "task": "Do something"
}`
	require.NoError(t, os.WriteFile(stdinPath, []byte(stdinContent), 0644))

	member := &Member{
		Name:       "worker-1",
		Agent:      "go-pro",
		StdinFile:  "stdin_empty_context.json",
		StdoutFile: "stdout.json",
	}

	_, err := buildPromptEnvelope(teamDir, member)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context field is empty")
}

func TestBuildEnvelope_MissingFile(t *testing.T) {
	teamDir := t.TempDir()

	member := &Member{
		Name:       "worker-1",
		Agent:      "go-pro",
		StdinFile:  "nonexistent.json",
		StdoutFile: "stdout.json",
	}

	_, err := buildPromptEnvelope(teamDir, member)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read stdin file")
}

func TestBuildEnvelope_InvalidJSON(t *testing.T) {
	teamDir := t.TempDir()

	// Create stdin with invalid JSON
	stdinPath := filepath.Join(teamDir, "stdin_bad.json")
	stdinContent := `{invalid json content`
	require.NoError(t, os.WriteFile(stdinPath, []byte(stdinContent), 0644))

	member := &Member{
		Name:       "worker-1",
		Agent:      "go-pro",
		StdinFile:  "stdin_bad.json",
		StdoutFile: "stdout.json",
	}

	_, err := buildPromptEnvelope(teamDir, member)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse stdin JSON")
}

func TestBuildEnvelope_PathTraversal(t *testing.T) {
	teamDir := t.TempDir()

	// Create a file outside teamDir to attempt path traversal
	outsideDir := t.TempDir()
	evilPath := filepath.Join(outsideDir, "evil.json")
	require.NoError(t, os.WriteFile(evilPath, []byte(`{"task":"hack"}`), 0644))

	member := &Member{
		Name:       "worker-1",
		Agent:      "go-pro",
		StdinFile:  "../../../" + evilPath,
		StdoutFile: "stdout.json",
	}

	_, err := buildPromptEnvelope(teamDir, member)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "escapes")
}

func TestBuildEnvelope_CapabilitiesNotice(t *testing.T) {
	teamDir := t.TempDir()

	stdinPath := filepath.Join(teamDir, "stdin.json")
	stdinContent := `{
  "agent": "einstein",
  "context": {"project_root": "/test"},
  "task": "Analyze the problem"
}`
	require.NoError(t, os.WriteFile(stdinPath, []byte(stdinContent), 0644))

	member := &Member{
		Name:       "einstein-1",
		Agent:      "einstein",
		StdinFile:  "stdin.json",
		StdoutFile: "stdout.json",
	}

	envelope, err := buildPromptEnvelope(teamDir, member)
	require.NoError(t, err)

	// Verify capabilities notice is present
	assert.Contains(t, envelope, "nesting level 2")
	assert.Contains(t, envelope, "Your Capabilities")
	assert.Contains(t, envelope, "Available delegation")
}

func TestBuildEnvelope_AgentName(t *testing.T) {
	teamDir := t.TempDir()

	stdinPath := filepath.Join(teamDir, "stdin.json")
	stdinContent := `{
  "agent": "backend-reviewer",
  "context": {"project_root": "/test"},
  "task": "Review the API handlers"
}`
	require.NoError(t, os.WriteFile(stdinPath, []byte(stdinContent), 0644))

	member := &Member{
		Name:       "reviewer-1",
		Agent:      "backend-reviewer",
		StdinFile:  "stdin.json",
		StdoutFile: "stdout.json",
	}

	envelope, err := buildPromptEnvelope(teamDir, member)
	require.NoError(t, err)

	// Verify agent name in header
	assert.Contains(t, envelope, "AGENT: backend-reviewer")
}

func TestBuildEnvelope_DescriptionFallback(t *testing.T) {
	teamDir := t.TempDir()

	// Use 'description' instead of 'task'
	stdinPath := filepath.Join(teamDir, "stdin.json")
	stdinContent := `{
  "agent": "go-pro",
  "context": {"project_root": "/test"},
  "description": "Implement feature using description field"
}`
	require.NoError(t, os.WriteFile(stdinPath, []byte(stdinContent), 0644))

	member := &Member{
		Name:       "worker-1",
		Agent:      "go-pro",
		StdinFile:  "stdin.json",
		StdoutFile: "stdout.json",
	}

	envelope, err := buildPromptEnvelope(teamDir, member)
	require.NoError(t, err)

	assert.Contains(t, envelope, "Implement feature using description field")
}

func TestValidatePathWithinDir(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T) (targetPath, baseDir string)
		expectError bool
		errorText   string
	}{
		{
			name: "relative path within base",
			setupFunc: func(t *testing.T) (string, string) {
				baseDir := t.TempDir()
				targetPath := filepath.Join(baseDir, "subdir", "file.txt")
				return targetPath, baseDir
			},
			expectError: false,
		},
		{
			name: "absolute path within base",
			setupFunc: func(t *testing.T) (string, string) {
				baseDir := t.TempDir()
				targetPath := filepath.Join(baseDir, "file.txt")
				return targetPath, baseDir
			},
			expectError: false,
		},
		{
			name: "exact base directory",
			setupFunc: func(t *testing.T) (string, string) {
				baseDir := t.TempDir()
				return baseDir, baseDir
			},
			expectError: false,
		},
		{
			name: "path traversal attempt",
			setupFunc: func(t *testing.T) (string, string) {
				baseDir := t.TempDir()
				targetPath := filepath.Join(baseDir, "..", "..", "etc", "passwd")
				return targetPath, baseDir
			},
			expectError: true,
			errorText:   "escapes",
		},
		{
			name: "absolute path outside base",
			setupFunc: func(t *testing.T) (string, string) {
				baseDir := t.TempDir()
				outsideDir := t.TempDir()
				return outsideDir, baseDir
			},
			expectError: true,
			errorText:   "escapes",
		},
		{
			name: "sibling directory attack",
			setupFunc: func(t *testing.T) (string, string) {
				parentDir := t.TempDir()
				baseDir := filepath.Join(parentDir, "base")
				require.NoError(t, os.Mkdir(baseDir, 0755))
				siblingDir := filepath.Join(parentDir, "sibling")
				require.NoError(t, os.Mkdir(siblingDir, 0755))
				targetPath := filepath.Join(siblingDir, "file.txt")
				return targetPath, baseDir
			},
			expectError: true,
			errorText:   "escapes",
		},
		{
			name: "file in subdirectory",
			setupFunc: func(t *testing.T) (string, string) {
				baseDir := t.TempDir()
				subdir := filepath.Join(baseDir, "waves", "wave-1")
				require.NoError(t, os.MkdirAll(subdir, 0755))
				targetPath := filepath.Join(subdir, "stdin.json")
				return targetPath, baseDir
			},
			expectError: false,
		},
		{
			name: "base-evil confusion (path prefix)",
			setupFunc: func(t *testing.T) (string, string) {
				parentDir := t.TempDir()
				baseDir := filepath.Join(parentDir, "base")
				require.NoError(t, os.Mkdir(baseDir, 0755))
				evilDir := filepath.Join(parentDir, "base-evil")
				require.NoError(t, os.Mkdir(evilDir, 0755))
				targetPath := filepath.Join(evilDir, "file.txt")
				return targetPath, baseDir
			},
			expectError: true,
			errorText:   "escapes",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			targetPath, baseDir := tc.setupFunc(t)
			err := validatePathWithinDir(targetPath, baseDir)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorText)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBuildEnvelope_WorkflowSpecificFields(t *testing.T) {
	teamDir := t.TempDir()

	// Create stdin with workflow-specific fields (e.g., problem_brief for braintrust)
	// Note: braintrust workflows use problem_brief.statement, not top-level task
	stdinPath := filepath.Join(teamDir, "stdin_einstein.json")
	stdinContent := `{
  "$schema": "braintrust-einstein-v1",
  "agent": "einstein",
  "workflow": "braintrust",
  "context": {
    "project_root": "/home/user/project",
    "team_dir": "/home/user/project/teams/test"
  },
  "task": "Perform theoretical analysis",
  "problem_brief": {
    "title": "Complex concurrency issue",
    "statement": "We need to solve X",
    "scope": {
      "in_scope": ["Module A"],
      "out_of_scope": ["Module B"]
    }
  }
}`
	require.NoError(t, os.WriteFile(stdinPath, []byte(stdinContent), 0644))

	member := &Member{
		Name:       "einstein-1",
		Agent:      "einstein",
		StdinFile:  "stdin_einstein.json",
		StdoutFile: "stdout_einstein.json",
	}

	envelope, err := buildPromptEnvelope(teamDir, member)
	require.NoError(t, err)

	// Verify workflow-specific fields are preserved in envelope
	assert.Contains(t, envelope, "problem_brief")
	assert.Contains(t, envelope, "Complex concurrency issue")
	assert.Contains(t, envelope, "in_scope")
	assert.Contains(t, envelope, "braintrust")
}

func TestValidatePathWithinDir_EdgeCases(t *testing.T) {
	t.Run("empty paths", func(t *testing.T) {
		err := validatePathWithinDir("", "")
		// Empty paths resolve to current directory, which is technically valid
		assert.NoError(t, err)
	})

	t.Run("base with trailing slash", func(t *testing.T) {
		baseDir := t.TempDir()
		targetPath := filepath.Join(baseDir, "file.txt")
		baseDirWithSlash := baseDir + string(filepath.Separator)
		err := validatePathWithinDir(targetPath, baseDirWithSlash)
		assert.NoError(t, err)
	})

	t.Run("nested subdirectories", func(t *testing.T) {
		baseDir := t.TempDir()
		deepPath := filepath.Join(baseDir, "a", "b", "c", "d", "file.txt")
		err := validatePathWithinDir(deepPath, baseDir)
		assert.NoError(t, err)
	})
}

func TestBuildEnvelope_MissingContext(t *testing.T) {
	teamDir := t.TempDir()

	// Create stdin with missing context field entirely
	stdinPath := filepath.Join(teamDir, "stdin_no_context.json")
	stdinContent := `{
  "$schema": "v1",
  "agent": "go-pro",
  "task": "Do something"
}`
	require.NoError(t, os.WriteFile(stdinPath, []byte(stdinContent), 0644))

	member := &Member{
		Name:       "worker-1",
		Agent:      "go-pro",
		StdinFile:  "stdin_no_context.json",
		StdoutFile: "stdout.json",
	}

	_, err := buildPromptEnvelope(teamDir, member)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context field is empty")
}

func TestBuildEnvelope_JSONSerializationPreservation(t *testing.T) {
	teamDir := t.TempDir()

	// Create stdin with nested structures to verify JSON round-trip
	stdinPath := filepath.Join(teamDir, "stdin_complex.json")
	stdinContent := `{
  "agent": "review-orchestrator",
  "context": {
    "project_root": "/test",
    "nested": {
      "deep": {
        "value": 42
      }
    }
  },
  "task": "Review changes",
  "constraints": ["Constraint 1", "Constraint 2"],
  "custom_field": {
    "array": [1, 2, 3],
    "object": {"key": "value"}
  }
}`
	require.NoError(t, os.WriteFile(stdinPath, []byte(stdinContent), 0644))

	member := &Member{
		Name:       "orchestrator-1",
		Agent:      "review-orchestrator",
		StdinFile:  "stdin_complex.json",
		StdoutFile: "stdout.json",
	}

	envelope, err := buildPromptEnvelope(teamDir, member)
	require.NoError(t, err)

	// Verify complex structures are preserved
	assert.Contains(t, envelope, `"deep"`)
	assert.Contains(t, envelope, `"value": 42`)
	assert.Contains(t, envelope, `"custom_field"`)
	assert.Contains(t, envelope, `"array"`)
	assert.Contains(t, envelope, `"Constraint 1"`)
}

func TestValidatePathWithinDir_RealWorldPaths(t *testing.T) {
	t.Run("typical team directory structure", func(t *testing.T) {
		baseDir := t.TempDir()
		teamStructure := []string{
			"waves/wave-1/stdin_worker1.json",
			"waves/wave-1/stdout_worker1.json",
			"waves/wave-2/stdin_worker2.json",
			"config.json",
			"heartbeat",
		}

		for _, path := range teamStructure {
			fullPath := filepath.Join(baseDir, path)
			require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
			require.NoError(t, os.WriteFile(fullPath, []byte("test"), 0644))

			err := validatePathWithinDir(fullPath, baseDir)
			assert.NoError(t, err, "path %s should be valid", path)
		}
	})

	t.Run("absolute path to system file", func(t *testing.T) {
		baseDir := t.TempDir()
		systemPath := "/etc/passwd"
		err := validatePathWithinDir(systemPath, baseDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "escapes")
	})
}
