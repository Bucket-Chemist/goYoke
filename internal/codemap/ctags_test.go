package codemap

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCtagsAvailable(t *testing.T) {
	// Just verify the function runs without panic. Whether ctags is installed
	// is environment-dependent.
	_ = CtagsAvailable()
}

func TestRunCtags_EmptyFiles(t *testing.T) {
	tags, err := RunCtags(nil, t.TempDir())
	require.NoError(t, err)
	assert.Empty(t, tags)
}

func TestRunCtags_PythonFile(t *testing.T) {
	if !CtagsAvailable() {
		t.Skip("ctags not installed")
	}

	dir := t.TempDir()
	src := `
def hello(name):
    pass

class Greeter:
    def greet(self):
        pass

MY_CONST = 42
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "greet.py"), []byte(src), 0o644))

	tags, err := RunCtags([]string{"greet.py"}, dir)
	require.NoError(t, err)
	assert.NotEmpty(t, tags)

	names := make(map[string]bool)
	for _, tag := range tags {
		names[tag.Name] = true
	}
	assert.True(t, names["hello"] || names["Greeter"], "expected function or class symbol")
}

func TestRunCtags_TypeScriptFile(t *testing.T) {
	if !CtagsAvailable() {
		t.Skip("ctags not installed")
	}

	dir := t.TempDir()
	src := `
export function greet(name: string): string {
    return "hello " + name;
}

export class Greeter {
    greet(): void {}
}

export interface Nameable {
    name: string;
}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "greet.ts"), []byte(src), 0o644))

	tags, err := RunCtags([]string{"greet.ts"}, dir)
	require.NoError(t, err)
	assert.NotEmpty(t, tags)

	names := make(map[string]bool)
	for _, tag := range tags {
		names[tag.Name] = true
	}
	assert.True(t, names["greet"] || names["Greeter"] || names["Nameable"],
		"expected at least one TS symbol")
}

func TestRunCtags_RustFile(t *testing.T) {
	if !CtagsAvailable() {
		t.Skip("ctags not installed")
	}

	dir := t.TempDir()
	src := `
pub struct Point {
    pub x: f64,
    pub y: f64,
}

pub fn distance(a: &Point, b: &Point) -> f64 {
    0.0
}

pub trait Shape {
    fn area(&self) -> f64;
}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "geo.rs"), []byte(src), 0o644))

	tags, err := RunCtags([]string{"geo.rs"}, dir)
	require.NoError(t, err)
	assert.NotEmpty(t, tags)

	names := make(map[string]bool)
	for _, tag := range tags {
		names[tag.Name] = true
	}
	assert.True(t, names["Point"] || names["distance"] || names["Shape"],
		"expected Rust symbols")
}

func TestCtagsToSymbol_ExportedPython(t *testing.T) {
	tag := CtagsTag{Name: "my_func", Kind: "function", Line: 1, End: 5}
	sym := CtagsToSymbol(tag, "python")
	assert.Equal(t, "function", sym.Kind)
	assert.True(t, sym.Exported, "public Python function should be exported")
}

func TestCtagsToSymbol_PrivatePython(t *testing.T) {
	tag := CtagsTag{Name: "_private", Kind: "function", Line: 1}
	sym := CtagsToSymbol(tag, "python")
	assert.False(t, sym.Exported, "_private should not be exported")
}

func TestCtagsToSymbol_DunderPython(t *testing.T) {
	tag := CtagsTag{Name: "__init__", Kind: "function", Line: 1}
	sym := CtagsToSymbol(tag, "python")
	assert.True(t, sym.Exported, "__dunder__ should be exported")
}

func TestCtagsToSymbol_RustPublic(t *testing.T) {
	tag := CtagsTag{Name: "my_func", Kind: "function", Access: "public", Line: 1}
	sym := CtagsToSymbol(tag, "rust")
	assert.True(t, sym.Exported)
}

func TestCtagsToSymbol_RustPrivate(t *testing.T) {
	tag := CtagsTag{Name: "helper", Kind: "function", Access: "private", Line: 1}
	sym := CtagsToSymbol(tag, "rust")
	assert.False(t, sym.Exported)
}

func TestCtagsToSymbol_MethodScope(t *testing.T) {
	scope := "MyClass"
	tag := CtagsTag{
		Name: "do_thing", Kind: "function",
		ScopeKind: "class", Scope: scope, Line: 10,
	}
	sym := CtagsToSymbol(tag, "python")
	assert.Equal(t, "method", sym.Kind)
	require.NotNil(t, sym.Receiver)
	assert.Equal(t, "MyClass", *sym.Receiver)
}

func TestMapCtagsKind(t *testing.T) {
	tests := []struct{ in, want string }{
		{"function", "function"},
		{"func", "function"},
		{"method", "method"},
		{"class", "type"},
		{"struct", "type"},
		{"enum", "enum"},
		{"trait", "interface"},
		{"interface", "interface"},
		{"type", "type"},
		{"typedef", "type"},
		{"alias", "type"},
		{"constant", "const"},
		{"const", "const"},
		{"variable", "var"},
		{"var", "var"},
		{"field", "var"},
		{"member", "var"},
		{"property", "var"},
		{"use", "import"},
		{"import", "import"},
		{"module", "module"},
		{"namespace", "module"},
		{"package", "module"},
		{"unknown_kind", "unknown_kind"},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			assert.Equal(t, tc.want, mapCtagsKind(tc.in, "go"))
		})
	}
}

func TestCtagsToSymbol_TypeScriptExported(t *testing.T) {
	tag := CtagsTag{Name: "MyFunc", Kind: "function", Access: "export", Line: 1}
	sym := CtagsToSymbol(tag, "typescript")
	assert.True(t, sym.Exported, "access=export should be exported in TypeScript")
}

func TestCtagsToSymbol_TypeScriptPublic(t *testing.T) {
	tag := CtagsTag{Name: "MyClass", Kind: "class", Access: "public", Line: 1}
	sym := CtagsToSymbol(tag, "typescript")
	assert.True(t, sym.Exported, "access=public should be exported in TypeScript")
}

func TestCtagsToSymbol_RExported(t *testing.T) {
	tag := CtagsTag{Name: "my_func", Kind: "function", Line: 1}
	sym := CtagsToSymbol(tag, "r")
	assert.True(t, sym.Exported, "all R symbols should be exported")
}

func TestCtagsToSymbol_UnknownLangExported(t *testing.T) {
	tag := CtagsTag{Name: "Foo", Kind: "function", Line: 1}
	sym := CtagsToSymbol(tag, "cobol")
	assert.True(t, sym.Exported, "unknown language defaults to exported")
}

func TestExtractPythonDecorators(t *testing.T) {
	src := []byte(`
@app.route("/")
@login_required
def index():
    pass
`)
	symbols := []Symbol{
		{Name: "index", Kind: "function", LineStart: 4},
	}
	extractPythonDecorators(src, symbols)
	assert.Equal(t, []string{"@app.route(\"/\")", "@login_required"}, symbols[0].Decorators)
}

func TestExtractModuleCtags_Python(t *testing.T) {
	if !CtagsAvailable() {
		t.Skip("ctags not installed")
	}

	dir := t.TempDir()
	src := `
import os
import requests

def fetch(url):
    pass

class Client:
    def get(self):
        pass
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "client.py"), []byte(src), 0o644))

	extraction, err := ExtractModuleCtags("", []string{"client.py"}, "python", dir, "myapp", false)
	require.NoError(t, err)
	require.NotNil(t, extraction)

	assert.Equal(t, "python", extraction.Language)
	assert.Len(t, extraction.Files, 1)
	assert.NotEmpty(t, extraction.Files[0].Symbols)

	names := make(map[string]bool)
	for _, s := range extraction.Files[0].Symbols {
		names[s.Name] = true
	}
	assert.True(t, names["fetch"] || names["Client"], "expected Python symbols extracted")
	assert.Contains(t, extraction.Imports.External, "requests")
	assert.Contains(t, extraction.Imports.Stdlib, "os")
}

// --- Per-language fixture tests (use testdata/ files) ---

func TestCtagsExtraction_RustFixture(t *testing.T) {
	if !CtagsAvailable() {
		t.Skip("ctags not installed")
	}
	tags, err := RunCtags([]string{"testdata/rust/simple.rs"}, ".")
	require.NoError(t, err)
	assert.NotEmpty(t, tags)

	byName := make(map[string]CtagsTag)
	for _, tag := range tags {
		byName[tag.Name] = tag
	}

	tests := []struct {
		symbol  string
		wantKnd string
	}{
		{"hello", "function"},
		{"Config", "type"},
		{"Status", "enum"},
	}
	for _, tc := range tests {
		t.Run(tc.symbol, func(t *testing.T) {
			tag, ok := byName[tc.symbol]
			assert.True(t, ok, "expected symbol %q in Rust extraction", tc.symbol)
			if ok {
				sym := CtagsToSymbol(tag, "rust")
				assert.Equal(t, tc.wantKnd, sym.Kind)
			}
		})
	}

	// Handler trait → kind="interface" after mapping
	if tag, ok := byName["Handler"]; ok {
		sym := CtagsToSymbol(tag, "rust")
		assert.Equal(t, "interface", sym.Kind, "trait should map to interface")
	}
}

func TestCtagsExtraction_TypeScriptFixture(t *testing.T) {
	if !CtagsAvailable() {
		t.Skip("ctags not installed")
	}
	tags, err := RunCtags([]string{"testdata/typescript/simple.ts"}, ".")
	require.NoError(t, err)
	assert.NotEmpty(t, tags)

	byName := make(map[string]CtagsTag)
	for _, tag := range tags {
		byName[tag.Name] = tag
	}

	tests := []struct {
		symbol  string
		wantKnd string
	}{
		{"greet", "function"},
		{"MyClass", "type"},
		{"Handler", "interface"},
		{"Status", "enum"},
	}
	for _, tc := range tests {
		t.Run(tc.symbol, func(t *testing.T) {
			tag, ok := byName[tc.symbol]
			assert.True(t, ok, "expected symbol %q in TypeScript extraction", tc.symbol)
			if ok {
				sym := CtagsToSymbol(tag, "typescript")
				assert.Equal(t, tc.wantKnd, sym.Kind)
			}
		})
	}
}

func TestCtagsExtraction_PythonFixture(t *testing.T) {
	if !CtagsAvailable() {
		t.Skip("ctags not installed")
	}
	tags, err := RunCtags([]string{"testdata/python/simple.py"}, ".")
	require.NoError(t, err)
	assert.NotEmpty(t, tags)

	byName := make(map[string]CtagsTag)
	for _, tag := range tags {
		byName[tag.Name] = tag
	}

	// hello is public
	if tag, ok := byName["hello"]; ok {
		sym := CtagsToSymbol(tag, "python")
		assert.True(t, sym.Exported, "hello() should be exported")
		assert.Equal(t, "function", sym.Kind)
	} else {
		t.Error("expected 'hello' function in Python extraction")
	}

	// _private_func is not public
	if tag, ok := byName["_private_func"]; ok {
		sym := CtagsToSymbol(tag, "python")
		assert.False(t, sym.Exported, "_private_func should not be exported")
	}

	// MyClass should be present
	_, hasMyClass := byName["MyClass"]
	assert.True(t, hasMyClass, "expected 'MyClass' class in Python extraction")
}

func TestCtagsExtraction_RFixture(t *testing.T) {
	if !CtagsAvailable() {
		t.Skip("ctags not installed")
	}
	tags, err := RunCtags([]string{"testdata/r/simple.R"}, ".")
	require.NoError(t, err)

	if len(tags) == 0 {
		t.Skip("ctags R support not available in this installation")
	}

	names := make(map[string]bool)
	for _, tag := range tags {
		names[tag.Name] = true
	}
	found := names["hello"] || names["greet"] || names["compute_sum"]
	assert.True(t, found, "expected at least one R function symbol")
}

func TestPythonDecoratorExtraction_Fixture(t *testing.T) {
	if !CtagsAvailable() {
		t.Skip("ctags not installed")
	}

	extraction, err := ExtractModuleCtags("", []string{"testdata/python/simple.py"}, "python", ".", "myapp", false)
	require.NoError(t, err)
	require.NotNil(t, extraction)
	require.NotEmpty(t, extraction.Files)

	// Find the 'value' property which is decorated with @property.
	var found bool
	for _, sym := range extraction.Files[0].Symbols {
		if sym.Name == "value" && len(sym.Decorators) > 0 {
			found = true
			assert.Contains(t, sym.Decorators, "@property")
			break
		}
	}
	if !found {
		// Verify @property is extracted somewhere — ctags may name it differently.
		for _, sym := range extraction.Files[0].Symbols {
			if len(sym.Decorators) > 0 {
				found = true
				break
			}
		}
	}
	assert.True(t, found, "expected at least one symbol with decorator in simple.py")
}
