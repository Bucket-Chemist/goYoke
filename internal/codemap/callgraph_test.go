package codemap_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Bucket-Chemist/goYoke/internal/codemap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

// makeTestModule creates a temporary Go module with the given files and returns its path.
func makeTestModule(t *testing.T, modName string, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	goMod := fmt.Sprintf("module %s\n\ngo 1.21\n", modName)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644))
	for name, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	return dir
}

// loadPkg returns the first loaded package whose path matches pkgPath.
func loadPkg(t *testing.T, pkgs []*packages.Package, pkgPath string) *packages.Package {
	t.Helper()
	for _, pkg := range pkgs {
		if pkg.Types != nil && pkg.Types.Path() == pkgPath {
			return pkg
		}
	}
	t.Fatalf("package %q not found among %d loaded packages", pkgPath, len(pkgs))
	return nil
}

func TestExtractCallSites_IntraPackage(t *testing.T) {
	dir := makeTestModule(t, "testmod", map[string]string{
		"pkg/a.go": `package pkg

func Caller() {
	Callee()
}
`,
		"pkg/b.go": `package pkg

func Callee() {}
`,
	})

	pkgs, err := codemap.LoadTypeCheckedPackages(dir)
	require.NoError(t, err)

	pkg := loadPkg(t, pkgs, "testmod/pkg")
	sites := codemap.ExtractCallSites(pkg.Fset, pkg, "testmod")

	require.Len(t, sites, 1)
	assert.Equal(t, "pkg.Caller", sites[0].CallerFunc)
	assert.Equal(t, "pkg.Callee", sites[0].CalleeName)
	assert.False(t, sites[0].CrossModule)
	assert.False(t, sites[0].IsMethod)
}

func TestExtractCallSites_CrossPackage(t *testing.T) {
	dir := makeTestModule(t, "testmod", map[string]string{
		"cmd/main.go": `package cmd

import "testmod/pkg"

func RunCmd() {
	pkg.Hello()
}
`,
		"pkg/lib.go": `package pkg

func Hello() {}
`,
	})

	pkgs, err := codemap.LoadTypeCheckedPackages(dir)
	require.NoError(t, err)

	pkg := loadPkg(t, pkgs, "testmod/cmd")
	sites := codemap.ExtractCallSites(pkg.Fset, pkg, "testmod")

	require.Len(t, sites, 1)
	assert.Equal(t, "cmd.RunCmd", sites[0].CallerFunc)
	assert.Equal(t, "pkg.Hello", sites[0].CalleeName)
	assert.True(t, sites[0].CrossModule)
}

func TestExtractCallSites_StdlibExcluded(t *testing.T) {
	dir := makeTestModule(t, "testmod", map[string]string{
		"pkg/a.go": `package pkg

import "fmt"

func Printer() {
	fmt.Println("hello")
}
`,
	})

	pkgs, err := codemap.LoadTypeCheckedPackages(dir)
	require.NoError(t, err)

	pkg := loadPkg(t, pkgs, "testmod/pkg")
	sites := codemap.ExtractCallSites(pkg.Fset, pkg, "testmod")

	assert.Empty(t, sites, "fmt.Println should not produce a call site")
}

func TestExtractCallSites_BuiltinsExcluded(t *testing.T) {
	dir := makeTestModule(t, "testmod", map[string]string{
		"pkg/a.go": `package pkg

func Builder() []int {
	s := make([]int, 0)
	_ = len(s)
	s = append(s, 1)
	return s
}
`,
	})

	pkgs, err := codemap.LoadTypeCheckedPackages(dir)
	require.NoError(t, err)

	pkg := loadPkg(t, pkgs, "testmod/pkg")
	sites := codemap.ExtractCallSites(pkg.Fset, pkg, "testmod")

	assert.Empty(t, sites, "make/len/append builtins should not produce call sites")
}

func TestExtractCallSites_MethodCall(t *testing.T) {
	dir := makeTestModule(t, "testmod", map[string]string{
		"pkg/a.go": `package pkg

type Worker struct{}

func (w *Worker) Do() {}

func Run() {
	wk := &Worker{}
	wk.Do()
}
`,
	})

	pkgs, err := codemap.LoadTypeCheckedPackages(dir)
	require.NoError(t, err)

	pkg := loadPkg(t, pkgs, "testmod/pkg")
	sites := codemap.ExtractCallSites(pkg.Fset, pkg, "testmod")

	require.Len(t, sites, 1)
	assert.Equal(t, "pkg.Run", sites[0].CallerFunc)
	assert.Equal(t, "pkg.Do", sites[0].CalleeName)
	assert.True(t, sites[0].IsMethod)
}

func TestResolveCallGraph_Bidirectional(t *testing.T) {
	dir := makeTestModule(t, "testmod", map[string]string{
		"pkg/a.go": `package pkg

func A() { B() }

func B() {}
`,
	})

	pkgs, err := codemap.LoadTypeCheckedPackages(dir)
	require.NoError(t, err)

	pkg := loadPkg(t, pkgs, "testmod/pkg")
	sites := codemap.ExtractCallSites(pkg.Fset, pkg, "testmod")
	require.NotEmpty(t, sites)

	extractions := []*codemap.ModuleExtraction{
		{
			Module:   "pkg",
			Language: "go",
			Files: []codemap.FileExtract{
				{
					Path: "pkg/a.go",
					Symbols: []codemap.Symbol{
						{Name: "A", Kind: "function"},
						{Name: "B", Kind: "function"},
					},
				},
			},
		},
	}

	codemap.ResolveCallGraph(extractions, sites)

	symA := &extractions[0].Files[0].Symbols[0]
	symB := &extractions[0].Files[0].Symbols[1]

	assert.Contains(t, symA.Calls, "pkg.B", "A.Calls should contain B")
	assert.Contains(t, symB.CalledBy, "pkg.A", "B.CalledBy should contain A")
	assert.Empty(t, symA.CalledBy)
	assert.Empty(t, symB.Calls)
}

func TestResolveCallGraph_Deduplication(t *testing.T) {
	// A calls B twice — should appear only once in Calls.
	dir := makeTestModule(t, "testmod", map[string]string{
		"pkg/a.go": `package pkg

func A() {
	B()
	B()
}

func B() {}
`,
	})

	pkgs, err := codemap.LoadTypeCheckedPackages(dir)
	require.NoError(t, err)

	pkg := loadPkg(t, pkgs, "testmod/pkg")
	sites := codemap.ExtractCallSites(pkg.Fset, pkg, "testmod")

	extractions := []*codemap.ModuleExtraction{
		{
			Module:   "pkg",
			Language: "go",
			Files: []codemap.FileExtract{
				{
					Path: "pkg/a.go",
					Symbols: []codemap.Symbol{
						{Name: "A", Kind: "function"},
						{Name: "B", Kind: "function"},
					},
				},
			},
		},
	}

	codemap.ResolveCallGraph(extractions, sites)

	symA := &extractions[0].Files[0].Symbols[0]
	assert.Len(t, symA.Calls, 1, "duplicate calls should be deduplicated")
	assert.Equal(t, "pkg.B", symA.Calls[0])
}

func TestCallGraph_GoYokeIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "could not determine source file path")
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))

	pkgs, err := codemap.LoadTypeCheckedPackages(repoRoot)
	require.NoError(t, err)
	require.NotEmpty(t, pkgs)

	const projectModule = "github.com/Bucket-Chemist/goYoke"
	var allSites []codemap.CallSite
	for _, pkg := range pkgs {
		sites := codemap.ExtractCallSites(pkg.Fset, pkg, projectModule)
		allSites = append(allSites, sites...)
	}

	require.NotEmpty(t, allSites, "goYoke repo should produce call sites")
	t.Logf("extracted %d call sites from goYoke repo", len(allSites))

	// ExtractModule in extractor.go calls ParseGoFile — verify this is found.
	foundExpected := false
	for _, s := range allSites {
		if strings.HasSuffix(s.CallerFunc, ".ExtractModule") && strings.HasSuffix(s.CalleeName, ".ParseGoFile") {
			foundExpected = true
			break
		}
	}
	assert.True(t, foundExpected, "expected ExtractModule → ParseGoFile call site")

	// Verify no fully-qualified external import paths leaked through.
	// Filtered callee FQNs are relative (e.g. "internal/codemap.ParseGoFile"),
	// not full import paths (e.g. "github.com/Bucket-Chemist/goYoke/internal/codemap.ParseGoFile").
	for _, s := range allSites {
		assert.NotContains(t, s.CalleeName, projectModule,
			"callee FQN should be relative, not a full import path: %s", s.CalleeName)
		assert.NotContains(t, s.CallerFunc, projectModule,
			"caller FQN should be relative, not a full import path: %s", s.CallerFunc)
	}
}
