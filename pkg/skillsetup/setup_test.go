package skillsetup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Bucket-Chemist/goYoke/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSkillGuardConfigFrom(t *testing.T) {
	fixture := writeFixtureIndex(t, map[string]*SkillGuardConfig{
		"braintrust": {
			RouterAllowedTools: []string{"Task", "Bash", "Read"},
			TeamDirSuffix:      "braintrust",
		},
		"review": {
			RouterAllowedTools: []string{"Task", "Bash"},
			TeamDirSuffix:      "code-review",
		},
	})

	tests := []struct {
		name      string
		skill     string
		wantNil   bool
		wantSuffix string
	}{
		{"team skill braintrust", "braintrust", false, "braintrust"},
		{"team skill review", "review", false, "code-review"},
		{"non-team skill", "dummies-guide", true, ""},
		{"unknown skill", "nonexistent", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := LoadSkillGuardConfigFrom(tt.skill, fixture)
			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, cfg)
			} else {
				require.NotNil(t, cfg)
				assert.Equal(t, tt.wantSuffix, cfg.TeamDirSuffix)
				assert.NotEmpty(t, cfg.RouterAllowedTools)
			}
		})
	}
}

func TestLoadSkillGuardConfigFrom_MissingFile(t *testing.T) {
	_, err := LoadSkillGuardConfigFrom("braintrust", "/nonexistent/path.json")
	assert.Error(t, err)
}

func TestLoadSkillGuardConfigFrom_InvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	require.NoError(t, os.WriteFile(path, []byte("{invalid"), 0644))

	_, err := LoadSkillGuardConfigFrom("braintrust", path)
	assert.Error(t, err)
}

func TestLoadSkillGuardConfigFrom_NoSkillGuardsSection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"agents": []}`), 0644))

	cfg, err := LoadSkillGuardConfigFrom("braintrust", path)
	require.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestCreateTeamDir(t *testing.T) {
	sessionDir := t.TempDir()

	teamDir, err := CreateTeamDir(sessionDir, "braintrust")
	require.NoError(t, err)

	assert.DirExists(t, teamDir)
	assert.Contains(t, teamDir, ".braintrust")
	assert.True(t, filepath.IsAbs(teamDir))

	parent := filepath.Dir(teamDir)
	assert.Equal(t, "teams", filepath.Base(parent))
}

func TestCreateTeamDir_DifferentSuffixes(t *testing.T) {
	sessionDir := t.TempDir()

	for _, suffix := range []string{"braintrust", "code-review", "implementation", "cleanup"} {
		teamDir, err := CreateTeamDir(sessionDir, suffix)
		require.NoError(t, err)
		assert.DirExists(t, teamDir)
		assert.Contains(t, filepath.Base(teamDir), suffix)
	}
}

func TestWriteGuardFile(t *testing.T) {
	origXDG := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", origXDG)
	os.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	sessionID := "test-session-write"
	guard := &config.ActiveSkill{
		FormatVersion:      2,
		Skill:              "braintrust",
		TeamDir:            "/tmp/fake-team",
		RouterAllowedTools: []string{"Task", "Bash"},
		CreatedAt:          "2026-04-18T00:00:00Z",
		SessionID:          sessionID,
		HolderPID:          12345,
	}

	err := WriteGuardFile(guard)
	require.NoError(t, err)

	guardPath := config.GetGuardFilePath(sessionID)
	data, err := os.ReadFile(guardPath)
	require.NoError(t, err)

	var loaded config.ActiveSkill
	require.NoError(t, json.Unmarshal(data, &loaded))
	assert.Equal(t, 2, loaded.FormatVersion)
	assert.Equal(t, "braintrust", loaded.Skill)
	assert.Equal(t, sessionID, loaded.SessionID)
	assert.Equal(t, 12345, loaded.HolderPID)
}

func TestWriteGuardFile_EmptySessionID(t *testing.T) {
	guard := &config.ActiveSkill{Skill: "test"}
	err := WriteGuardFile(guard)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SessionID is empty")
}

func TestRemoveGuardFiles(t *testing.T) {
	origXDG := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", origXDG)
	os.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	sessionID := "test-session-remove"
	guardPath := config.GetGuardFilePath(sessionID)
	lockPath := config.GetGuardLockPath(sessionID)

	require.NoError(t, os.WriteFile(guardPath, []byte("{}"), 0644))
	require.NoError(t, os.WriteFile(lockPath, []byte(""), 0644))

	err := RemoveGuardFiles(sessionID)
	assert.NoError(t, err)
	assert.NoFileExists(t, guardPath)
	assert.NoFileExists(t, lockPath)
}

func TestRemoveGuardFiles_Idempotent(t *testing.T) {
	origXDG := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", origXDG)
	os.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	err := RemoveGuardFiles("nonexistent-session")
	assert.NoError(t, err)
}

func TestResolveSessionDir(t *testing.T) {
	origDir := os.Getenv("GOYOKE_SESSION_DIR")
	defer os.Setenv("GOYOKE_SESSION_DIR", origDir)

	expected := "/tmp/test-session-dir"
	os.Setenv("GOYOKE_SESSION_DIR", expected)

	result := ResolveSessionDir()
	assert.Equal(t, expected, result)
}

func TestResolveSessionID(t *testing.T) {
	tests := []struct {
		name       string
		envID      string
		sessionDir string
		wantExact  string
	}{
		{"env var takes priority", "env-session-id", "/some/dir/fallback-id", "env-session-id"},
		{"basename of session dir", "", "/home/user/.goyoke/sessions/abc-123", "abc-123"},
		{"empty dir generates UUID", "", "", ""},
		{"unknown dir generates UUID", "", ".goyoke/sessions/unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := os.Getenv("GOYOKE_SESSION_ID")
			defer os.Setenv("GOYOKE_SESSION_ID", orig)

			os.Setenv("GOYOKE_SESSION_ID", tt.envID)
			result := ResolveSessionID(tt.sessionDir)

			if tt.wantExact != "" {
				assert.Equal(t, tt.wantExact, result)
			} else {
				assert.NotEmpty(t, result)
				assert.Len(t, result, 36, "should be a UUID")
			}
		})
	}
}

func writeFixtureIndex(t *testing.T, guards map[string]*SkillGuardConfig) string {
	t.Helper()
	index := struct {
		SkillGuards map[string]*SkillGuardConfig `json:"skill_guards"`
	}{SkillGuards: guards}

	data, err := json.Marshal(index)
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "agents-index.json")
	require.NoError(t, os.WriteFile(path, data, 0644))
	return path
}
