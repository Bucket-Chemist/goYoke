package codemap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTestExtraction(module string) *ModuleExtraction {
	return &ModuleExtraction{
		Module:   module,
		Language: "go",
		Files: []FileExtract{
			{
				Path:      module + "/main.go",
				LineCount: 42,
				Symbols: []Symbol{
					{Name: "main", Kind: "function", Exported: false, LineStart: 1, LineEnd: 5},
				},
			},
		},
		Imports:          ImportGraph{Stdlib: []string{"fmt"}, Internal: nil, External: nil},
		ExtractedAt:      time.Now().UTC().Format(time.RFC3339),
		ExtractorVersion: "test",
	}
}

func TestWriteModuleExtraction(t *testing.T) {
	dir := t.TempDir()
	e := makeTestExtraction("cmd/test-module")

	err := WriteModuleExtraction(dir, e)
	require.NoError(t, err)

	expectedPath := filepath.Join(dir, "cmd--test-module.json")
	data, err := os.ReadFile(expectedPath)
	require.NoError(t, err, "output file must exist")

	var got ModuleExtraction
	require.NoError(t, json.Unmarshal(data, &got), "output must be valid JSON")
	assert.Equal(t, e.Module, got.Module)
	assert.Equal(t, e.Language, got.Language)
	assert.Len(t, got.Files, 1)
}

func TestWriteModuleExtraction_PrettyPrinted(t *testing.T) {
	dir := t.TempDir()
	e := makeTestExtraction("internal/foo")

	require.NoError(t, WriteModuleExtraction(dir, e))

	data, err := os.ReadFile(filepath.Join(dir, "internal--foo.json"))
	require.NoError(t, err)

	// Pretty-printed JSON starts with "{\n" not "{"
	assert.Contains(t, string(data), "\n  ", "expected indented JSON")
}

func TestWriteModuleExtraction_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	e := makeTestExtraction("cmd/atomic")

	require.NoError(t, WriteModuleExtraction(dir, e))

	// The .tmp file must not exist after a successful write.
	tmpPath := filepath.Join(dir, "cmd--atomic.json.tmp")
	_, err := os.Stat(tmpPath)
	assert.True(t, os.IsNotExist(err), ".tmp file must be removed after rename")
}

func TestWriteModuleExtraction_CreatesDir(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "subdir", "extract")
	e := makeTestExtraction("pkg/routing")

	require.NoError(t, WriteModuleExtraction(dir, e))
	assert.FileExists(t, filepath.Join(dir, "pkg--routing.json"))
}

func makeTestExtractionLang(module, language string) *ModuleExtraction {
	filePath := "file." + language
	if module != "" {
		filePath = module + "/" + filePath
	}
	return &ModuleExtraction{
		Module:           module,
		Language:         language,
		Files:            []FileExtract{{Path: filePath, LineCount: 10}},
		ExtractedAt:      time.Now().UTC().Format(time.RFC3339),
		ExtractorVersion: "test",
	}
}

func TestWriteModuleExtraction_NonGoLanguageSuffix(t *testing.T) {
	dir := t.TempDir()

	py := makeTestExtractionLang("pkg/utils", "python")
	require.NoError(t, WriteModuleExtraction(dir, py))
	assert.FileExists(t, filepath.Join(dir, "pkg--utils.python.json"))

	rs := makeTestExtractionLang("pkg/utils", "rust")
	require.NoError(t, WriteModuleExtraction(dir, rs))
	assert.FileExists(t, filepath.Join(dir, "pkg--utils.rust.json"))
}

func TestWriteModuleExtraction_RootDirMultiLang(t *testing.T) {
	dir := t.TempDir()

	py := makeTestExtractionLang("", "python")
	require.NoError(t, WriteModuleExtraction(dir, py))
	assert.FileExists(t, filepath.Join(dir, "_root.python.json"))

	rs := makeTestExtractionLang("", "rust")
	require.NoError(t, WriteModuleExtraction(dir, rs))
	assert.FileExists(t, filepath.Join(dir, "_root.rust.json"))

	ts := makeTestExtractionLang("", "typescript")
	require.NoError(t, WriteModuleExtraction(dir, ts))
	assert.FileExists(t, filepath.Join(dir, "_root.typescript.json"))

	// All three separate files must exist.
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 3)
}

func TestWriteModuleExtraction_GoRootDirNoSuffix(t *testing.T) {
	dir := t.TempDir()
	e := makeTestExtraction("")

	require.NoError(t, WriteModuleExtraction(dir, e))
	assert.FileExists(t, filepath.Join(dir, "_root.json"), "Go root module should produce _root.json without language suffix")
}

func TestWriteModuleExtraction_MultipleModules(t *testing.T) {
	dir := t.TempDir()
	modules := []string{"cmd/foo", "internal/bar", "pkg/baz"}
	for _, mod := range modules {
		require.NoError(t, WriteModuleExtraction(dir, makeTestExtraction(mod)))
	}

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, len(modules))
}
