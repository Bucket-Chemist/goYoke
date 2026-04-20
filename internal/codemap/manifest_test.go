package codemap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadManifest_NonExistent(t *testing.T) {
	m, err := LoadManifest(filepath.Join(t.TempDir(), "manifest.json"))
	require.NoError(t, err)
	require.NotNil(t, m)
	assert.Equal(t, "1.0", m.Version)
	assert.NotNil(t, m.Modules)
	assert.NotNil(t, m.Stats.Languages)
}

func TestLoadManifest_ValidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")

	raw := `{"version":"1.0","repo_name":"myrepo","modules":{},"stats":{"total_files":3,"total_modules":1,"total_symbols":10,"total_functions":5,"total_types":2,"languages":{"go":3}}}`
	require.NoError(t, os.WriteFile(path, []byte(raw), 0o644))

	m, err := LoadManifest(path)
	require.NoError(t, err)
	assert.Equal(t, "myrepo", m.RepoName)
	assert.Equal(t, 3, m.Stats.TotalFiles)
	assert.Equal(t, 1, m.Stats.TotalModules)
	assert.Equal(t, 3, m.Stats.Languages["go"])
}

func TestLoadManifest_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	require.NoError(t, os.WriteFile(path, []byte("{not json}"), 0o644))

	_, err := LoadManifest(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse manifest")
}

func TestWriteManifest_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")

	m := &Manifest{
		Version:  "1.0",
		RepoName: "testrepo",
		Modules:  map[string]*ManifestModule{},
		Stats:    ManifestStats{Languages: map[string]int{"go": 5}},
	}

	require.NoError(t, WriteManifest(path, m))
	assert.FileExists(t, path)

	// Verify .tmp file cleaned up
	_, err := os.Stat(path + ".tmp")
	assert.True(t, os.IsNotExist(err))

	// Verify round-trip
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var got Manifest
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, "testrepo", got.RepoName)
}

func TestWriteManifest_CreatesParentDir(t *testing.T) {
	base := t.TempDir()
	path := filepath.Join(base, "subdir", "nested", "manifest.json")

	m := &Manifest{Version: "1.0", Modules: map[string]*ManifestModule{}}
	require.NoError(t, WriteManifest(path, m))
	assert.FileExists(t, path)
}

func TestResolveRepoName_GitRepo(t *testing.T) {
	name := ResolveRepoName(repoRoot)
	assert.NotEmpty(t, name)
	// The repo name should be derived from the remote URL or directory basename.
	assert.Contains(t, name, "goYoke")
}

func TestResolveGitCommit_ValidRepo(t *testing.T) {
	commit := ResolveGitCommit(repoRoot)
	assert.NotEmpty(t, commit, "expected a git commit SHA from the repo")
	assert.Len(t, commit, 40, "expected full 40-char SHA")
}

func TestStatMtime_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "x.txt")
	require.NoError(t, os.WriteFile(f, []byte("hi"), 0o644))

	mtime := statMtime(f)
	assert.NotEmpty(t, mtime)
}

func TestStatMtime_Missing(t *testing.T) {
	mtime := statMtime(filepath.Join(t.TempDir(), "no-such-file"))
	assert.Empty(t, mtime)
}

func TestBuildManifest_Basic(t *testing.T) {
	extractions := []*ModuleExtraction{
		makeTestExtraction("cmd/app"),
		makeTestExtraction("internal/lib"),
	}

	m, err := BuildManifest(repoRoot, extractions, "v1.0.0")
	require.NoError(t, err)
	require.NotNil(t, m)
	assert.Equal(t, "1.0", m.Version)
	assert.Equal(t, "v1.0.0", m.ExtractorVersion)
	assert.Len(t, m.Modules, 2)
	assert.NotNil(t, m.LastFullMap)
}
