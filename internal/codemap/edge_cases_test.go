package codemap

import (
	"fmt"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFixture_EmptyFile verifies that a file containing only a package declaration
// produces no symbols and no parse errors.
func TestFixture_EmptyFile(t *testing.T) {
	source, err := os.ReadFile("testdata/go/empty.go")
	require.NoError(t, err)

	fset := token.NewFileSet()
	f, parseErrors, err := ParseGoFile(fset, "empty.go", source)
	require.NoError(t, err)
	assert.Empty(t, parseErrors)
	require.NotNil(t, f)

	ext := &GoExtractor{}
	syms, graph := ext.ExtractFile(fset, f, source, "")
	assert.Empty(t, syms, "empty file should produce no symbols")
	assert.Empty(t, graph.Stdlib)
	assert.Empty(t, graph.External)
	assert.Empty(t, graph.Internal)
}

// TestFixture_BrokenSyntax verifies that a syntactically broken fixture file
// produces parse errors (error_count > 0) while still allowing partial extraction.
func TestFixture_BrokenSyntax(t *testing.T) {
	// ParseGoFile level: partial AST + non-empty parse errors.
	source, err := os.ReadFile("testdata/go/broken.go")
	require.NoError(t, err)

	fset := token.NewFileSet()
	f, parseErrors, err := ParseGoFile(fset, "broken.go", source)
	require.NoError(t, err, "broken syntax should return parse errors, not a fatal error")
	assert.NotEmpty(t, parseErrors, "broken.go must produce at least one parse error")

	// Partial AST may still allow symbol extraction — must not panic.
	if f != nil {
		ext := &GoExtractor{}
		syms, _ := ext.ExtractFile(fset, f, source, "")
		_ = syms
	}

	// ExtractModule level: ErrorCount on the FileExtract must be > 0.
	extraction, err := ExtractModule("testfixture", []string{"broken.go"}, "testdata/go", "", false)
	require.NoError(t, err)
	require.Len(t, extraction.Files, 1)
	assert.Greater(t, extraction.Files[0].ErrorCount, 0, "broken.go must produce error_count > 0 in module extraction")
}

// TestFixture_GeneratedDetection verifies that isGeneratedFile correctly identifies
// the generated.go fixture as a generated file.
func TestFixture_GeneratedDetection(t *testing.T) {
	// isGeneratedFile reads the file from disk.
	assert.True(t, isGeneratedFile("testdata/go/generated.go"), "generated.go should be detected as generated")
	assert.False(t, isGeneratedFile("testdata/go/simple.go"), "simple.go should not be detected as generated")
	assert.False(t, isGeneratedFile("testdata/go/empty.go"), "empty.go should not be detected as generated")
}

// TestEdgeCase_UnicodeIdentifiers verifies that functions with unicode names are
// extracted correctly (exported vs unexported based on first rune).
func TestEdgeCase_UnicodeIdentifiers(t *testing.T) {
	source := []byte("package p\n\nfunc café() {}\nfunc Über() string { return \"\" }\n")
	fset := token.NewFileSet()
	f, parseErrors, err := ParseGoFile(fset, "unicode.go", source)
	require.NoError(t, err)
	assert.Empty(t, parseErrors, "unicode identifiers are valid Go syntax")
	require.NotNil(t, f)

	ext := &GoExtractor{}
	syms, _ := ext.ExtractFile(fset, f, source, "")

	cafe := findSymbol(t, syms, "café")
	assert.Equal(t, "function", cafe.Kind)
	assert.False(t, cafe.Exported, "café starts with lowercase rune")

	uber := findSymbol(t, syms, "Über")
	assert.Equal(t, "function", uber.Kind)
	assert.True(t, uber.Exported, "Über starts with uppercase rune")
}

// TestEdgeCase_LargeFunction verifies that line_start and line_end span correctly
// for a function spread across many lines.
func TestEdgeCase_LargeFunction(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("package p\n\nfunc BigFunc() {\n")
	for i := range 50 {
		fmt.Fprintf(&sb, "\t_ = %d\n", i)
	}
	sb.WriteString("}\n")
	source := []byte(sb.String())

	fset := token.NewFileSet()
	f, _, err := ParseGoFile(fset, "large.go", source)
	require.NoError(t, err)

	ext := &GoExtractor{}
	syms, _ := ext.ExtractFile(fset, f, source, "")
	s := findSymbol(t, syms, "BigFunc")

	assert.Equal(t, 3, s.LineStart, "BigFunc starts at line 3")
	assert.Greater(t, s.LineEnd, s.LineStart, "line_end must be greater than line_start for a multi-line function")
	assert.GreaterOrEqual(t, s.LineEnd, 53, "function body spans at least 53 lines")
}

// TestEdgeCase_ConstIotaBlock verifies that a const block using iota produces
// one symbol per name with correct exported flags.
func TestEdgeCase_ConstIotaBlock(t *testing.T) {
	source := []byte("package p\n\nconst (\n\tA = iota\n\tB\n\tC\n)\n")
	fset := token.NewFileSet()
	f, _, err := ParseGoFile(fset, "iota.go", source)
	require.NoError(t, err)

	ext := &GoExtractor{}
	syms, _ := ext.ExtractFile(fset, f, source, "")

	names := symbolNames(syms)
	assert.ElementsMatch(t, []string{"A", "B", "C"}, names)

	for _, s := range syms {
		assert.Equal(t, "const", s.Kind)
		assert.True(t, s.Exported, "A/B/C are exported")
	}
}

// TestFixture_SimpleSymbols verifies the simple.go fixture yields the expected symbols.
func TestFixture_SimpleSymbols(t *testing.T) {
	source, err := os.ReadFile("testdata/go/simple.go")
	require.NoError(t, err)

	fset := token.NewFileSet()
	f, parseErrors, err := ParseGoFile(fset, "simple.go", source)
	require.NoError(t, err)
	assert.Empty(t, parseErrors)

	ext := &GoExtractor{}
	syms, _ := ext.ExtractFile(fset, f, source, "")

	hello := findSymbol(t, syms, "Hello")
	assert.Equal(t, "function", hello.Kind)
	assert.True(t, hello.Exported)

	add := findSymbol(t, syms, "add")
	assert.Equal(t, "function", add.Kind)
	assert.False(t, add.Exported)

	cfg := findSymbol(t, syms, "Config")
	assert.Equal(t, "type", cfg.Kind)
	assert.True(t, cfg.Exported)

	handler := findSymbol(t, syms, "Handler")
	assert.Equal(t, "interface", handler.Kind)
	assert.True(t, handler.Exported)
}

// TestFixture_MethodReceivers verifies the methods.go fixture yields correct receiver metadata.
func TestFixture_MethodReceivers(t *testing.T) {
	source, err := os.ReadFile("testdata/go/methods.go")
	require.NoError(t, err)

	fset := token.NewFileSet()
	f, parseErrors, err := ParseGoFile(fset, "methods.go", source)
	require.NoError(t, err)
	assert.Empty(t, parseErrors)

	ext := &GoExtractor{}
	syms, _ := ext.ExtractFile(fset, f, source, "")

	addr := findSymbol(t, syms, "Address")
	assert.Equal(t, "method", addr.Kind)
	require.NotNil(t, addr.Receiver)
	assert.Equal(t, "Server", *addr.Receiver)

	setPort := findSymbol(t, syms, "SetPort")
	assert.Equal(t, "method", setPort.Kind)
	require.NotNil(t, setPort.Receiver)
	assert.Equal(t, "*Server", *setPort.Receiver)
}

// TestFixture_Imports verifies the imports.go fixture classifies stdlib imports correctly.
func TestFixture_Imports(t *testing.T) {
	source, err := os.ReadFile("testdata/go/imports.go")
	require.NoError(t, err)

	fset := token.NewFileSet()
	f, parseErrors, err := ParseGoFile(fset, "imports.go", source)
	require.NoError(t, err)
	assert.Empty(t, parseErrors)

	ext := &GoExtractor{}
	_, graph := ext.ExtractFile(fset, f, source, "somemodule")
	assert.Contains(t, graph.Stdlib, "fmt")
	assert.Contains(t, graph.Stdlib, "os")
	assert.Empty(t, graph.External)
	assert.Empty(t, graph.Internal)
}
