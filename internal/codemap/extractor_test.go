package codemap

import (
	"go/ast"
	"go/token"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseSource is a test helper that parses Go source and returns symbols + import graph.
func parseSource(t *testing.T, source string) ([]Symbol, ImportGraph) {
	t.Helper()
	fset := token.NewFileSet()
	f, errs, err := ParseGoFile(fset, "test.go", []byte(source))
	require.NoError(t, err, "ParseGoFile should not return fatal error")
	require.Empty(t, errs, "expected no parse errors for valid source")
	ext := &GoExtractor{}
	syms, graph := ext.ExtractFile(fset, f, []byte(source), "github.com/Bucket-Chemist/goYoke")
	return syms, graph
}

// findSymbol returns the first symbol with the given name or fails the test.
func findSymbol(t *testing.T, syms []Symbol, name string) Symbol {
	t.Helper()
	for _, s := range syms {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("symbol %q not found in %v", name, symbolNames(syms))
	return Symbol{}
}

func symbolNames(syms []Symbol) []string {
	names := make([]string, len(syms))
	for i, s := range syms {
		names[i] = s.Name
	}
	return names
}

// --- Function extraction ---

func TestExtractGoFunction_Simple(t *testing.T) {
	syms, _ := parseSource(t, `package p; func hello() {}`)
	s := findSymbol(t, syms, "hello")
	assert.Equal(t, "function", s.Kind)
	assert.False(t, s.Exported)
	assert.Nil(t, s.Receiver)
	assert.Nil(t, s.Params)
	assert.Nil(t, s.Returns)
}

func TestExtractGoFunction_Exported(t *testing.T) {
	syms, _ := parseSource(t, `package p; func Hello() {}`)
	s := findSymbol(t, syms, "Hello")
	assert.True(t, s.Exported)
}

func TestExtractGoFunction_WithParamsAndReturn(t *testing.T) {
	src := `package p; func add(a, b int) int { return a + b }`
	syms, _ := parseSource(t, src)
	s := findSymbol(t, syms, "add")
	assert.Equal(t, "function", s.Kind)
	require.Len(t, s.Params, 2)
	assert.Equal(t, "a", s.Params[0].Name)
	assert.Equal(t, "int", s.Params[0].Type)
	assert.Equal(t, "b", s.Params[1].Name)
	assert.Equal(t, "int", s.Params[1].Type)
	require.Len(t, s.Returns, 1)
	assert.Equal(t, "int", s.Returns[0].Type)
}

func TestExtractGoFunction_Variadic(t *testing.T) {
	src := `package p; func sum(nums ...int) int { return 0 }`
	syms, _ := parseSource(t, src)
	s := findSymbol(t, syms, "sum")
	require.Len(t, s.Params, 1)
	assert.Equal(t, "nums", s.Params[0].Name)
	assert.Equal(t, "...int", s.Params[0].Type)
}

func TestExtractGoFunction_MultipleReturns(t *testing.T) {
	src := `package p; func divide(a, b float64) (float64, error) { return 0, nil }`
	syms, _ := parseSource(t, src)
	s := findSymbol(t, syms, "divide")
	require.Len(t, s.Returns, 2)
	assert.Equal(t, "float64", s.Returns[0].Type)
	assert.Equal(t, "error", s.Returns[1].Type)
}

func TestExtractGoFunction_UnnamedParams(t *testing.T) {
	src := `package p; func f(int, string) {}`
	syms, _ := parseSource(t, src)
	s := findSymbol(t, syms, "f")
	require.Len(t, s.Params, 2)
	assert.Equal(t, "", s.Params[0].Name)
	assert.Equal(t, "int", s.Params[0].Type)
	assert.Equal(t, "", s.Params[1].Name)
	assert.Equal(t, "string", s.Params[1].Type)
}

func TestExtractGoFunction_Signature(t *testing.T) {
	src := `package p

func compute(x int, y int) (int, error) {
	return x + y, nil
}`
	syms, _ := parseSource(t, src)
	s := findSymbol(t, syms, "compute")
	assert.Contains(t, s.Signature, "func compute")
	assert.Contains(t, s.Signature, "x int")
	assert.Contains(t, s.Signature, "y int")
}

func TestExtractGoFunction_LineRange(t *testing.T) {
	src := "package p\n\nfunc foo() {\n}\n"
	syms, _ := parseSource(t, src)
	s := findSymbol(t, syms, "foo")
	assert.Equal(t, 3, s.LineStart)
	assert.Equal(t, 4, s.LineEnd)
}

// --- Method extraction ---

func TestExtractGoMethod_ValueReceiver(t *testing.T) {
	src := `package p
type T struct{}
func (t T) Do() {}`
	syms, _ := parseSource(t, src)
	s := findSymbol(t, syms, "Do")
	assert.Equal(t, "method", s.Kind)
	require.NotNil(t, s.Receiver)
	assert.Equal(t, "T", *s.Receiver)
}

func TestExtractGoMethod_PointerReceiver(t *testing.T) {
	src := `package p
type Client struct{}
func (c *Client) Send(msg string) error { return nil }`
	syms, _ := parseSource(t, src)
	s := findSymbol(t, syms, "Send")
	assert.Equal(t, "method", s.Kind)
	require.NotNil(t, s.Receiver)
	assert.Equal(t, "*Client", *s.Receiver)
	require.Len(t, s.Params, 1)
	assert.Equal(t, "msg", s.Params[0].Name)
	assert.Equal(t, "string", s.Params[0].Type)
	require.Len(t, s.Returns, 1)
	assert.Equal(t, "error", s.Returns[0].Type)
}

func TestExtractGoMethod_Unexported(t *testing.T) {
	src := `package p
type T struct{}
func (t T) internal() {}`
	syms, _ := parseSource(t, src)
	s := findSymbol(t, syms, "internal")
	assert.Equal(t, "method", s.Kind)
	assert.False(t, s.Exported)
}

// --- Type extraction ---

func TestExtractGoType_Struct(t *testing.T) {
	src := `package p; type Config struct{ Host string; Port int }`
	syms, _ := parseSource(t, src)
	s := findSymbol(t, syms, "Config")
	assert.Equal(t, "type", s.Kind)
	assert.True(t, s.Exported)
	assert.Contains(t, s.Signature, "type Config struct")
}

func TestExtractGoType_Interface(t *testing.T) {
	src := `package p; type Reader interface{ Read(p []byte) (int, error) }`
	syms, _ := parseSource(t, src)
	s := findSymbol(t, syms, "Reader")
	assert.Equal(t, "interface", s.Kind)
	assert.True(t, s.Exported)
	assert.Contains(t, s.Signature, "type Reader interface")
}

func TestExtractGoType_Alias(t *testing.T) {
	src := `package p; type ID = string`
	syms, _ := parseSource(t, src)
	s := findSymbol(t, syms, "ID")
	assert.Equal(t, "type", s.Kind)
}

func TestExtractGoType_Unexported(t *testing.T) {
	src := `package p; type config struct{}`
	syms, _ := parseSource(t, src)
	s := findSymbol(t, syms, "config")
	assert.Equal(t, "type", s.Kind)
	assert.False(t, s.Exported)
}

// --- Const extraction ---

func TestExtractGoConst(t *testing.T) {
	src := `package p; const MaxRetries = 3`
	syms, _ := parseSource(t, src)
	s := findSymbol(t, syms, "MaxRetries")
	assert.Equal(t, "const", s.Kind)
	assert.True(t, s.Exported)
	assert.Contains(t, s.Signature, "const MaxRetries")
}

func TestExtractGoConst_Block(t *testing.T) {
	src := `package p
const (
	A = 1
	b = 2
)`
	syms, _ := parseSource(t, src)
	a := findSymbol(t, syms, "A")
	assert.Equal(t, "const", a.Kind)
	assert.True(t, a.Exported)
	bSym := findSymbol(t, syms, "b")
	assert.Equal(t, "const", bSym.Kind)
	assert.False(t, bSym.Exported)
}

// --- Var extraction ---

func TestExtractGoVar(t *testing.T) {
	src := `package p; var ErrNotFound = errors.New("not found")`
	syms, _ := parseSource(t, src)
	s := findSymbol(t, syms, "ErrNotFound")
	assert.Equal(t, "var", s.Kind)
	assert.True(t, s.Exported)
	assert.Contains(t, s.Signature, "var ErrNotFound")
}

func TestExtractGoVar_Block(t *testing.T) {
	src := `package p
var (
	Host = "localhost"
	port = 8080
)`
	syms, _ := parseSource(t, src)
	h := findSymbol(t, syms, "Host")
	assert.Equal(t, "var", h.Kind)
	assert.True(t, h.Exported)
	p := findSymbol(t, syms, "port")
	assert.Equal(t, "var", p.Kind)
	assert.False(t, p.Exported)
}

// --- Import classification ---

func TestClassifyImport(t *testing.T) {
	ext := &GoExtractor{}
	module := "github.com/Bucket-Chemist/goYoke"

	tests := []struct {
		path string
		want string
	}{
		{"fmt", "stdlib"},
		{"os", "stdlib"},
		{"encoding/json", "stdlib"},
		{"go/ast", "stdlib"},
		{"github.com/Bucket-Chemist/goYoke/internal/codemap", "internal"},
		{"github.com/Bucket-Chemist/goYoke/pkg/routing", "internal"},
		{"github.com/Bucket-Chemist/goYoke/cmd/goyoke-validate", "internal"},
		{"github.com/sabhiram/go-gitignore", "external"},
		{"golang.org/x/tools/go/packages", "external"},
		{"github.com/stretchr/testify/require", "external"},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			got := ext.ClassifyImport(tc.path, module)
			assert.Equal(t, tc.want, got, "ClassifyImport(%q)", tc.path)
		})
	}
}

func TestClassifyImport_EmptyModule(t *testing.T) {
	ext := &GoExtractor{}
	// With empty projectModule, nothing is "internal"
	assert.Equal(t, "stdlib", ext.ClassifyImport("fmt", ""))
	assert.Equal(t, "external", ext.ClassifyImport("github.com/foo/bar", ""))
}

// --- Import graph from file ---

func TestExtractFile_ImportGraph(t *testing.T) {
	src := `package p

import (
	"fmt"
	"os"
	"github.com/sabhiram/go-gitignore"
	"github.com/Bucket-Chemist/goYoke/internal/codemap"
)

func f() { _ = fmt.Println; _ = os.Exit; _ = gitignore.CompileIgnoreFile; _ = codemap.Version }
`
	fset := token.NewFileSet()
	f, _, err := ParseGoFile(fset, "test.go", []byte(src))
	require.NoError(t, err)
	ext := &GoExtractor{}
	_, graph := ext.ExtractFile(fset, f, []byte(src), "github.com/Bucket-Chemist/goYoke")

	assert.Contains(t, graph.Stdlib, "fmt")
	assert.Contains(t, graph.Stdlib, "os")
	assert.Contains(t, graph.External, "github.com/sabhiram/go-gitignore")
	assert.Contains(t, graph.Internal, "github.com/Bucket-Chemist/goYoke/internal/codemap")
}

// --- Parse errors ---

func TestParseErrors_ValidSource(t *testing.T) {
	fset := token.NewFileSet()
	_, errs, err := ParseGoFile(fset, "test.go", []byte(`package p; func f() {}`))
	require.NoError(t, err)
	assert.Empty(t, errs, "valid source should produce no parse errors")
}

func TestParseErrors_InvalidSyntax(t *testing.T) {
	fset := token.NewFileSet()
	// Unclosed function — guaranteed parse error
	src := "package p\nfunc bad(\n"
	f, errs, err := ParseGoFile(fset, "test.go", []byte(src))
	require.NoError(t, err, "parse errors should not be returned as fatal errors")
	assert.NotEmpty(t, errs, "invalid syntax should produce parse errors")
	// Partial AST may still be returned
	_ = f

	for _, pe := range errs {
		assert.Greater(t, pe.Line, 0, "error line should be positive")
		assert.NotEmpty(t, pe.Message, "error message should not be empty")
	}
}

func TestParseErrors_ExtractionContinues(t *testing.T) {
	// A file with a parse error should still extract whatever symbols are available
	// (partial AST). We just verify the call does not return a fatal error.
	src := "package p\ntype Good struct{}\nfunc bad(\n"
	fset := token.NewFileSet()
	f, errs, err := ParseGoFile(fset, "test.go", []byte(src))
	require.NoError(t, err)
	assert.NotEmpty(t, errs)
	if f != nil {
		ext := &GoExtractor{}
		syms, _ := ext.ExtractFile(fset, f, []byte(src), "")
		// Good struct may or may not appear depending on partial parse
		_ = syms
	}
}

// --- IsExported ---

func TestIsExported(t *testing.T) {
	ext := &GoExtractor{}
	assert.True(t, ext.IsExported("Exported"))
	assert.False(t, ext.IsExported("unexported"))
	assert.True(t, ext.IsExported("URL"))
	assert.False(t, ext.IsExported(""))
}

// --- ast.IsExported consistency ---

func TestAstIsExported_Consistency(t *testing.T) {
	ext := &GoExtractor{}
	src := `package p
func Exported() {}
func unexported() {}
type PublicType struct{}
type privateType struct{}
const PublicConst = 1
const privateConst = 2
`
	syms, _ := parseSource(t, src)
	for _, s := range syms {
		expected := ast.IsExported(s.Name)
		assert.Equal(t, expected, s.Exported, "symbol %q exported mismatch", s.Name)
		assert.Equal(t, expected, ext.IsExported(s.Name), "IsExported(%q) mismatch", s.Name)
	}
}

// --- Integration test ---

func TestExtractModule_Integration(t *testing.T) {
	modules, err := DiscoverFiles(repoRoot, DiscoveryOpts{})
	require.NoError(t, err)

	codemapFiles, ok := modules[ModuleKey{Path: "internal/codemap", Language: "go"}]
	require.True(t, ok, "expected internal/codemap module in discovery results")

	extraction, err := ExtractModule("internal/codemap", codemapFiles, repoRoot, "github.com/Bucket-Chemist/goYoke", false)
	require.NoError(t, err)
	require.NotNil(t, extraction)

	assert.Equal(t, "internal/codemap", extraction.Module)
	assert.Equal(t, "go", extraction.Language)
	assert.NotEmpty(t, extraction.ExtractedAt)
	assert.NotEmpty(t, extraction.Files)

	// types.go has 9 type/interface declarations
	typesFile := findFileExtract(t, extraction, "types.go")
	require.NotNil(t, typesFile, "types.go must be in extraction results")

	nameKind := make(map[string]string)
	for _, s := range typesFile.Symbols {
		nameKind[s.Name] = s.Kind
	}
	assert.Equal(t, "type", nameKind["ModuleExtraction"])
	assert.Equal(t, "type", nameKind["FileExtract"])
	assert.Equal(t, "type", nameKind["ParseError"])
	assert.Equal(t, "type", nameKind["Symbol"])
	assert.Equal(t, "type", nameKind["Param"])
	assert.Equal(t, "type", nameKind["ReturnVal"])
	assert.Equal(t, "type", nameKind["ImportGraph"])
	assert.Equal(t, "interface", nameKind["LanguageExtractor"])
	assert.Equal(t, "type", nameKind["DiscoveryOpts"])

	// Verify import graph classification on discovery.go
	discoveryFile := findFileExtract(t, extraction, "discovery.go")
	require.NotNil(t, discoveryFile, "discovery.go must be in extraction results")
	// discovery.go imports "bufio", "fmt", "os" (stdlib) and go-gitignore (external)
	assert.Contains(t, extraction.Imports.Stdlib, "fmt")
	assert.Contains(t, extraction.Imports.Stdlib, "os")
	assert.Contains(t, extraction.Imports.External, "github.com/sabhiram/go-gitignore")

	// Line counts must be positive
	for _, fe := range extraction.Files {
		assert.Greater(t, fe.LineCount, 0, "file %s should have positive line count", fe.Path)
	}

	// Total symbol count sanity check
	total := 0
	for _, fe := range extraction.Files {
		total += len(fe.Symbols)
	}
	assert.GreaterOrEqual(t, total, 15, "expected at least 15 symbols across codemap package")
}

func findFileExtract(t *testing.T, m *ModuleExtraction, base string) *FileExtract {
	t.Helper()
	for i := range m.Files {
		if filepath.Base(m.Files[i].Path) == base {
			return &m.Files[i]
		}
	}
	return nil
}

// --- ExtractModule error handling ---

func TestExtractModule_MissingFile(t *testing.T) {
	_, err := ExtractModule("mod", []string{"nonexistent.go"}, t.TempDir(), "", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "read")
}

// --- GoExtractor method coverage ---

func TestGoExtractor_Extensions(t *testing.T) {
	g := &GoExtractor{}
	exts := g.Extensions()
	require.Equal(t, []string{".go"}, exts)
}

func TestReceiverTypeString_StarExpr(t *testing.T) {
	syms, _ := parseSource(t, `package p
type Foo struct{}
func (f *Foo) Method() {}
`)
	s := findSymbol(t, syms, "Method")
	require.NotNil(t, s.Receiver)
	assert.Equal(t, "*Foo", *s.Receiver)
}
