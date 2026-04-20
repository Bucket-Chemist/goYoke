package codemap

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tempNonGitDir creates a temporary directory that is NOT a git repo.
func tempNonGitDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "codemap-incr-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// writeFile writes content to path (creating parent dirs).
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestDetectChanges_NoManifest(t *testing.T) {
	dir := tempNonGitDir(t)
	cs, err := DetectChanges(nil, dir, map[ModuleKey][]string{})
	require.NoError(t, err)
	assert.True(t, cs.FullRemap)
}

func TestDetectChanges_EmptyGitCommit(t *testing.T) {
	dir := tempNonGitDir(t)
	m := &Manifest{
		Version:   "1.0",
		GitCommit: "", // no commit → full remap
		Modules:   map[string]*ManifestModule{"pkg": {Files: map[string]*ManifestFile{}}},
	}
	cs, err := DetectChanges(m, dir, map[ModuleKey][]string{})
	require.NoError(t, err)
	assert.True(t, cs.FullRemap)
}

func TestDetectChanges_EmptyModules(t *testing.T) {
	dir := tempNonGitDir(t)
	m := &Manifest{
		Version:   "1.0",
		GitCommit: "abc123",
		Modules:   map[string]*ManifestModule{},
	}
	cs, err := DetectChanges(m, dir, map[ModuleKey][]string{})
	require.NoError(t, err)
	assert.True(t, cs.FullRemap)
}

func TestDetectChanges_NoChanges(t *testing.T) {
	dir := tempNonGitDir(t)

	// Write a file, record its mtime, then set extracted_at in the future.
	goFile := filepath.Join(dir, "pkg", "foo.go")
	writeFile(t, goFile, "package pkg\n")

	future := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	m := &Manifest{
		Version:   "1.0",
		GitCommit: "abc123",
		Modules: map[string]*ManifestModule{
			"pkg": {
				Files: map[string]*ManifestFile{
					"pkg/foo.go": {ExtractedAt: future},
				},
			},
		},
	}
	discovered := map[ModuleKey][]string{
		{Path: "pkg", Language: "go"}: {"pkg/foo.go"},
	}
	cs, err := DetectChanges(m, dir, discovered)
	require.NoError(t, err)
	assert.False(t, cs.FullRemap)
	assert.Empty(t, cs.ChangedFiles)
	assert.Empty(t, cs.NewFiles)
	assert.Empty(t, cs.DeletedFiles)
	assert.Equal(t, "mtime", cs.Method)
}

func TestDetectChanges_ModifiedFile(t *testing.T) {
	dir := tempNonGitDir(t)

	goFile := filepath.Join(dir, "pkg", "foo.go")
	writeFile(t, goFile, "package pkg\n")

	// Set extracted_at to an hour ago — mtime is after it.
	past := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	m := &Manifest{
		Version:   "1.0",
		GitCommit: "abc123",
		Modules: map[string]*ManifestModule{
			"pkg": {
				Files: map[string]*ManifestFile{
					"pkg/foo.go": {ExtractedAt: past},
				},
			},
		},
	}
	discovered := map[ModuleKey][]string{
		{Path: "pkg", Language: "go"}: {"pkg/foo.go"},
	}
	cs, err := DetectChanges(m, dir, discovered)
	require.NoError(t, err)
	assert.False(t, cs.FullRemap)
	assert.Contains(t, cs.ChangedFiles, "pkg/foo.go")
	assert.Equal(t, "mtime", cs.Method)
}

func TestDetectChanges_NewFile(t *testing.T) {
	dir := tempNonGitDir(t)

	writeFile(t, filepath.Join(dir, "pkg", "foo.go"), "package pkg\n")
	writeFile(t, filepath.Join(dir, "pkg", "bar.go"), "package pkg\n")

	future := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	m := &Manifest{
		Version:   "1.0",
		GitCommit: "abc123",
		Modules: map[string]*ManifestModule{
			"pkg": {
				// Only foo.go in manifest — bar.go is new.
				Files: map[string]*ManifestFile{
					"pkg/foo.go": {ExtractedAt: future},
				},
			},
		},
	}
	discovered := map[ModuleKey][]string{
		{Path: "pkg", Language: "go"}: {"pkg/foo.go", "pkg/bar.go"},
	}
	cs, err := DetectChanges(m, dir, discovered)
	require.NoError(t, err)
	assert.False(t, cs.FullRemap)
	assert.Contains(t, cs.NewFiles, "pkg/bar.go")
	assert.NotContains(t, cs.ChangedFiles, "pkg/foo.go")
}

func TestDetectChanges_DeletedFile(t *testing.T) {
	dir := tempNonGitDir(t)

	// Only foo.go exists on disk; bar.go is in manifest but deleted.
	writeFile(t, filepath.Join(dir, "pkg", "foo.go"), "package pkg\n")

	future := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	m := &Manifest{
		Version:   "1.0",
		GitCommit: "abc123",
		Modules: map[string]*ManifestModule{
			"pkg": {
				Files: map[string]*ManifestFile{
					"pkg/foo.go": {ExtractedAt: future},
					"pkg/bar.go": {ExtractedAt: future},
				},
			},
		},
	}
	discovered := map[ModuleKey][]string{
		{Path: "pkg", Language: "go"}: {"pkg/foo.go"},
	}
	cs, err := DetectChanges(m, dir, discovered)
	require.NoError(t, err)
	assert.False(t, cs.FullRemap)
	assert.Contains(t, cs.DeletedFiles, "pkg/bar.go")
	assert.NotContains(t, cs.DeletedFiles, "pkg/foo.go")
}

func TestMergeExtraction_UpdateFile(t *testing.T) {
	existing := &ModuleExtraction{
		Module:   "pkg",
		Language: "go",
		Files: []FileExtract{
			{Path: "pkg/foo.go", LineCount: 10, Symbols: []Symbol{{Name: "OldFunc", Kind: "function"}}},
			{Path: "pkg/bar.go", LineCount: 5},
		},
		Imports: ImportGraph{Stdlib: []string{"fmt"}},
	}
	partial := &ModuleExtraction{
		Module:           "pkg",
		Language:         "go",
		ExtractedAt:      "2026-01-01T00:00:00Z",
		ExtractorVersion: "test",
		Files: []FileExtract{
			{Path: "pkg/foo.go", LineCount: 20, Symbols: []Symbol{{Name: "NewFunc", Kind: "function"}}},
		},
		Imports: ImportGraph{Stdlib: []string{"os"}},
	}

	merged := MergeExtraction(existing, partial)

	require.Len(t, merged.Files, 2)
	// Find foo.go in result — it must be from partial.
	var fooFile FileExtract
	for _, f := range merged.Files {
		if f.Path == "pkg/foo.go" {
			fooFile = f
		}
	}
	assert.Equal(t, 20, fooFile.LineCount, "updated file should have partial's data")
	assert.Equal(t, "NewFunc", fooFile.Symbols[0].Name)
	// bar.go retained from existing
	var barFound bool
	for _, f := range merged.Files {
		if f.Path == "pkg/bar.go" {
			barFound = true
		}
	}
	assert.True(t, barFound, "unchanged file should be retained")
	// Imports are unioned
	assert.Contains(t, merged.Imports.Stdlib, "fmt")
	assert.Contains(t, merged.Imports.Stdlib, "os")
}

func TestMergeExtraction_AddFile(t *testing.T) {
	existing := &ModuleExtraction{
		Module:   "pkg",
		Language: "go",
		Files:    []FileExtract{{Path: "pkg/foo.go", LineCount: 5}},
	}
	partial := &ModuleExtraction{
		Module:           "pkg",
		Language:         "go",
		ExtractedAt:      "2026-01-01T00:00:00Z",
		ExtractorVersion: "test",
		Files:            []FileExtract{{Path: "pkg/bar.go", LineCount: 8}},
	}

	merged := MergeExtraction(existing, partial)

	require.Len(t, merged.Files, 2)
	paths := []string{merged.Files[0].Path, merged.Files[1].Path}
	assert.Contains(t, paths, "pkg/foo.go")
	assert.Contains(t, paths, "pkg/bar.go")
	assert.Equal(t, "2026-01-01T00:00:00Z", merged.ExtractedAt)
}

func TestRemoveDeletedFiles(t *testing.T) {
	e := &ModuleExtraction{
		Module:   "pkg",
		Language: "go",
		Files: []FileExtract{
			{Path: "pkg/foo.go"},
			{Path: "pkg/bar.go"},
			{Path: "pkg/baz.go"},
		},
	}
	RemoveDeletedFiles(e, []string{"pkg/bar.go"})

	require.Len(t, e.Files, 2)
	for _, f := range e.Files {
		assert.NotEqual(t, "pkg/bar.go", f.Path, "deleted file should be removed")
	}
}

func TestRemoveDeletedFiles_AllGone(t *testing.T) {
	e := &ModuleExtraction{
		Module: "pkg",
		Files:  []FileExtract{{Path: "pkg/foo.go"}},
	}
	RemoveDeletedFiles(e, []string{"pkg/foo.go"})
	assert.Empty(t, e.Files)
}

func TestLoadExistingExtraction(t *testing.T) {
	dir := tempNonGitDir(t)

	e := &ModuleExtraction{
		Module:           "pkg/sub",
		Language:         "go",
		ExtractedAt:      "2026-01-01T00:00:00Z",
		ExtractorVersion: "test",
		Files:            []FileExtract{{Path: "pkg/sub/foo.go", LineCount: 7}},
	}
	require.NoError(t, WriteModuleExtraction(dir, e))

	loaded, err := LoadExistingExtraction(dir, "pkg/sub")
	require.NoError(t, err)
	assert.Equal(t, e.Module, loaded.Module)
	assert.Equal(t, e.ExtractedAt, loaded.ExtractedAt)
	require.Len(t, loaded.Files, 1)
	assert.Equal(t, 7, loaded.Files[0].LineCount)
}

func TestLoadExistingExtraction_NotFound(t *testing.T) {
	dir := tempNonGitDir(t)
	_, err := LoadExistingExtraction(dir, "does/not/exist")
	require.Error(t, err)
}

// TestDetectChanges_GitBased exercises the git code path in DetectChanges,
// covering filterToDiscovered and manifestOnlyFiles.
func TestDetectChanges_GitBased(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}

	dir, err := os.MkdirTemp("", "codemap-git-detect-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })

	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=t@t.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=t@t.com",
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v: %s", args, out)
	}

	git("init")
	git("config", "user.email", "t@t.com")
	git("config", "user.name", "test")

	// Commit 1: foo.go and manifest_only.go (will be "deleted" from discovery).
	writeFile(t, filepath.Join(dir, "pkg", "foo.go"), "package pkg\n")
	writeFile(t, filepath.Join(dir, "pkg", "manifest_only.go"), "package pkg\n")
	git("add", ".")
	git("commit", "-m", "initial")

	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	require.NoError(t, err)
	commit1 := strings.TrimSpace(string(out))

	// Commit 2: add bar.go.
	writeFile(t, filepath.Join(dir, "pkg", "bar.go"), "package pkg\n")
	git("add", ".")
	git("commit", "-m", "add bar")

	future := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	m := &Manifest{
		Version:   "1.0",
		GitCommit: commit1,
		Modules: map[string]*ManifestModule{
			"pkg": {
				Files: map[string]*ManifestFile{
					"pkg/foo.go":           {ExtractedAt: future},
					"pkg/manifest_only.go": {ExtractedAt: future},
				},
			},
		},
	}
	// manifest_only.go is NOT in discovered → should appear in DeletedFiles.
	discovered := map[ModuleKey][]string{
		{Path: "pkg", Language: "go"}: {"pkg/foo.go", "pkg/bar.go"},
	}

	cs, err := DetectChanges(m, dir, discovered)
	require.NoError(t, err)
	assert.False(t, cs.FullRemap)
	assert.Equal(t, "git", cs.Method)
	// bar.go was added since commit1 and is in discovered.
	assert.Contains(t, cs.NewFiles, "pkg/bar.go")
	// foo.go was unchanged — not in ChangedFiles or NewFiles.
	assert.NotContains(t, cs.ChangedFiles, "pkg/foo.go")
	// manifest_only.go was in manifest but not discovered.
	assert.Contains(t, cs.DeletedFiles, "pkg/manifest_only.go")
}

// TestGitChangedFiles creates a real temp git repo with two commits and verifies
// that gitChangedFiles correctly classifies added, modified, and deleted files.
func TestGitChangedFiles(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not in PATH")
	}

	dir, err := os.MkdirTemp("", "codemap-git-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })

	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=t@t.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=t@t.com",
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v: %s", args, out)
	}

	git("init")
	git("config", "user.email", "t@t.com")
	git("config", "user.name", "test")

	// Commit 1: add foo.go and keep.go.
	writeFile(t, filepath.Join(dir, "foo.go"), "package main\n")
	writeFile(t, filepath.Join(dir, "keep.go"), "package main\n")
	git("add", ".")
	git("commit", "-m", "initial")

	// Record commit 1 SHA.
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	require.NoError(t, err)
	commit1 := strings.TrimSpace(string(out))

	// Commit 2: modify foo.go, add bar.go, delete keep.go.
	writeFile(t, filepath.Join(dir, "foo.go"), "package main\nfunc A(){}\n")
	writeFile(t, filepath.Join(dir, "bar.go"), "package main\n")
	require.NoError(t, os.Remove(filepath.Join(dir, "keep.go")))
	git("add", ".")
	git("commit", "-m", "changes")

	cs, err := gitChangedFiles(dir, commit1, []string{".go"})
	require.NoError(t, err)
	assert.Equal(t, "git", cs.Method)
	assert.Contains(t, cs.ChangedFiles, "foo.go")
	assert.Contains(t, cs.NewFiles, "bar.go")
	assert.Contains(t, cs.DeletedFiles, "keep.go")
}
