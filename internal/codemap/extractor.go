package codemap

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Version is the extractor version, set at build time via -ldflags.
var Version = "dev"

// GoExtractor implements LanguageExtractor for Go source files using go/ast.
type GoExtractor struct{}

func (g *GoExtractor) Language() string        { return "go" }
func (g *GoExtractor) Extensions() []string    { return []string{".go"} }
func (g *GoExtractor) IsExported(name string) bool { return ast.IsExported(name) }

// ClassifyImport returns "internal", "external", or "stdlib".
// Internal: path starts with projectModule.
// Stdlib: first path segment contains no dot (e.g. "fmt", "encoding/json").
// External: everything else.
func (g *GoExtractor) ClassifyImport(importPath, projectModule string) string {
	if projectModule != "" && strings.HasPrefix(importPath, projectModule) {
		return "internal"
	}
	firstSeg, _, _ := strings.Cut(importPath, "/")
	if !strings.Contains(firstSeg, ".") {
		return "stdlib"
	}
	return "external"
}

// ExtractFile walks a parsed AST and extracts symbols and the import graph.
func (g *GoExtractor) ExtractFile(fset *token.FileSet, file *ast.File, source []byte, projectModule string) ([]Symbol, ImportGraph) {
	var symbols []Symbol
	var graph ImportGraph

	internalSet := make(map[string]bool)
	externalSet := make(map[string]bool)
	stdlibSet := make(map[string]bool)

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			symbols = append(symbols, g.extractFunc(fset, d, source))
		case *ast.GenDecl:
			syms, imports := g.extractGenDecl(fset, d, projectModule)
			symbols = append(symbols, syms...)
			for _, imp := range imports {
				switch imp.kind {
				case "internal":
					internalSet[imp.path] = true
				case "external":
					externalSet[imp.path] = true
				case "stdlib":
					stdlibSet[imp.path] = true
				}
			}
		}
	}

	for path := range internalSet {
		graph.Internal = append(graph.Internal, path)
	}
	for path := range externalSet {
		graph.External = append(graph.External, path)
	}
	for path := range stdlibSet {
		graph.Stdlib = append(graph.Stdlib, path)
	}

	return symbols, graph
}

type importEntry struct {
	path string
	kind string
}

func (g *GoExtractor) extractFunc(fset *token.FileSet, d *ast.FuncDecl, source []byte) Symbol {
	startPos := fset.Position(d.Pos())
	endPos := fset.Position(d.End())

	kind := "function"
	var receiver *string
	if d.Recv != nil && len(d.Recv.List) > 0 {
		kind = "method"
		recvStr := receiverTypeString(d.Recv.List[0].Type)
		receiver = &recvStr
	}

	return Symbol{
		Name:      d.Name.Name,
		Kind:      kind,
		Signature: buildFuncSignature(fset, d, source),
		Params:    extractParams(d.Type.Params),
		Returns:   extractReturns(d.Type.Results),
		Receiver:  receiver,
		LineStart: startPos.Line,
		LineEnd:   endPos.Line,
		Exported:  ast.IsExported(d.Name.Name),
	}
}

func (g *GoExtractor) extractGenDecl(fset *token.FileSet, d *ast.GenDecl, projectModule string) ([]Symbol, []importEntry) {
	var symbols []Symbol
	var imports []importEntry

	switch d.Tok {
	case token.IMPORT:
		for _, spec := range d.Specs {
			ispec, ok := spec.(*ast.ImportSpec)
			if !ok {
				continue
			}
			path := strings.Trim(ispec.Path.Value, `"`)
			imports = append(imports, importEntry{
				path: path,
				kind: g.ClassifyImport(path, projectModule),
			})
		}

	case token.TYPE:
		for _, spec := range d.Specs {
			tspec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			start := fset.Position(tspec.Pos())
			end := fset.Position(tspec.End())

			kind := "type"
			if _, isIface := tspec.Type.(*ast.InterfaceType); isIface {
				kind = "interface"
			}

			symbols = append(symbols, Symbol{
				Name:      tspec.Name.Name,
				Kind:      kind,
				Signature: typeSignatureString(tspec),
				LineStart: start.Line,
				LineEnd:   end.Line,
				Exported:  ast.IsExported(tspec.Name.Name),
			})
		}

	case token.CONST:
		for _, spec := range d.Specs {
			vspec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range vspec.Names {
				start := fset.Position(name.Pos())
				end := fset.Position(name.End())
				symbols = append(symbols, Symbol{
					Name:      name.Name,
					Kind:      "const",
					Signature: "const " + name.Name,
					LineStart: start.Line,
					LineEnd:   end.Line,
					Exported:  ast.IsExported(name.Name),
				})
			}
		}

	case token.VAR:
		for _, spec := range d.Specs {
			vspec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range vspec.Names {
				start := fset.Position(name.Pos())
				end := fset.Position(name.End())
				symbols = append(symbols, Symbol{
					Name:      name.Name,
					Kind:      "var",
					Signature: "var " + name.Name,
					LineStart: start.Line,
					LineEnd:   end.Line,
					Exported:  ast.IsExported(name.Name),
				})
			}
		}
	}

	return symbols, imports
}

// receiverTypeString returns a human-readable receiver type string (e.g. "*Client", "T").
func receiverTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return "*" + receiverTypeString(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		// Generic receiver: T[A]
		return receiverTypeString(t.X)
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// extractParams converts an *ast.FieldList of function parameters into []Param.
func extractParams(fields *ast.FieldList) []Param {
	if fields == nil || len(fields.List) == 0 {
		return nil
	}
	var params []Param
	for _, field := range fields.List {
		typeStr := typeExprString(field.Type)
		if field.Names == nil {
			params = append(params, Param{Name: "", Type: typeStr})
			continue
		}
		for _, name := range field.Names {
			params = append(params, Param{Name: name.Name, Type: typeStr})
		}
	}
	return params
}

// extractReturns converts an *ast.FieldList of return values into []ReturnVal.
func extractReturns(fields *ast.FieldList) []ReturnVal {
	if fields == nil || len(fields.List) == 0 {
		return nil
	}
	var returns []ReturnVal
	for _, field := range fields.List {
		typeStr := typeExprString(field.Type)
		if field.Names == nil {
			returns = append(returns, ReturnVal{Type: typeStr})
			continue
		}
		// Named returns: one entry per name
		for range field.Names {
			returns = append(returns, ReturnVal{Type: typeStr})
		}
	}
	return returns
}

// typeExprString converts an AST type expression to a readable string.
func typeExprString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeExprString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + typeExprString(t.Elt)
		}
		return "[...]" + typeExprString(t.Elt)
	case *ast.MapType:
		return "map[" + typeExprString(t.Key) + "]" + typeExprString(t.Value)
	case *ast.SelectorExpr:
		return typeExprString(t.X) + "." + t.Sel.Name
	case *ast.Ellipsis:
		return "..." + typeExprString(t.Elt)
	case *ast.ChanType:
		switch t.Dir {
		case ast.SEND:
			return "chan<- " + typeExprString(t.Value)
		case ast.RECV:
			return "<-chan " + typeExprString(t.Value)
		default:
			return "chan " + typeExprString(t.Value)
		}
	case *ast.FuncType:
		return "func(...)"
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct{}"
	case *ast.IndexExpr:
		return typeExprString(t.X) + "[...]"
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// buildFuncSignature extracts the function signature text from source bytes,
// stopping at the opening brace. Falls back to the function name on any failure.
func buildFuncSignature(fset *token.FileSet, d *ast.FuncDecl, source []byte) string {
	if len(source) == 0 {
		return "func " + d.Name.Name
	}

	tf := fset.File(d.Pos())
	if tf == nil {
		return "func " + d.Name.Name
	}

	startOff := tf.Offset(d.Pos())
	var endOff int
	if d.Body != nil {
		endOff = tf.Offset(d.Body.Lbrace)
	} else {
		endOff = tf.Offset(d.End())
	}

	if startOff < 0 || endOff <= startOff || endOff > len(source) {
		return "func " + d.Name.Name
	}

	return string(bytes.TrimSpace(source[startOff:endOff]))
}

// typeSignatureString returns a concise signature for a type declaration.
func typeSignatureString(tspec *ast.TypeSpec) string {
	switch tspec.Type.(type) {
	case *ast.StructType:
		return "type " + tspec.Name.Name + " struct"
	case *ast.InterfaceType:
		return "type " + tspec.Name.Name + " interface"
	default:
		return "type " + tspec.Name.Name
	}
}

// ExtractModule runs Go symbol extraction across all files in a module.
func ExtractModule(modulePath string, filePaths []string, projectRoot string, projectModule string, verbose bool) (*ModuleExtraction, error) {
	ext := &GoExtractor{}
	fset := token.NewFileSet()

	result := &ModuleExtraction{
		Module:           modulePath,
		Language:         ext.Language(),
		ExtractedAt:      time.Now().UTC().Format(time.RFC3339),
		ExtractorVersion: Version,
	}

	internalSet := make(map[string]bool)
	externalSet := make(map[string]bool)
	stdlibSet := make(map[string]bool)

	for _, relPath := range filePaths {
		absPath := filepath.Join(projectRoot, relPath)
		source, err := os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", relPath, err)
		}

		astFile, parseErrs, err := ParseGoFile(fset, absPath, source)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", relPath, err)
		}

		lineCount := bytes.Count(source, []byte("\n"))
		if len(source) > 0 && source[len(source)-1] != '\n' {
			lineCount++
		}

		var symbols []Symbol
		var graph ImportGraph
		if astFile != nil {
			symbols, graph = ext.ExtractFile(fset, astFile, source, projectModule)
		}

		for _, p := range graph.Internal {
			internalSet[p] = true
		}
		for _, p := range graph.External {
			externalSet[p] = true
		}
		for _, p := range graph.Stdlib {
			stdlibSet[p] = true
		}

		result.Files = append(result.Files, FileExtract{
			Path:        relPath,
			LineCount:   lineCount,
			Symbols:     symbols,
			ErrorCount:  len(parseErrs),
			ParseErrors: parseErrs,
		})
	}

	// Merge deduplicated import graphs.
	for p := range internalSet {
		result.Imports.Internal = append(result.Imports.Internal, p)
	}
	for p := range externalSet {
		result.Imports.External = append(result.Imports.External, p)
	}
	for p := range stdlibSet {
		result.Imports.Stdlib = append(result.Imports.Stdlib, p)
	}
	sort.Strings(result.Imports.Internal)
	sort.Strings(result.Imports.External)
	sort.Strings(result.Imports.Stdlib)

	return result, nil
}
