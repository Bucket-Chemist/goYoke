package codemap

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

// CallSite represents a single project-internal function call relationship.
type CallSite struct {
	CallerFunc  string // fully qualified: "internal/codemap.ExtractModule"
	CalleeName  string // fully qualified: "internal/codemap.ParseGoFile"
	Line        int
	IsMethod    bool
	CrossModule bool
}

// LoadTypeCheckedPackages loads all packages under projectRoot with full type information.
func LoadTypeCheckedPackages(projectRoot string) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedSyntax |
			packages.NeedTypesInfo | packages.NeedImports,
		Dir: projectRoot,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("load packages: %w", err)
	}
	return pkgs, nil
}

// ExtractCallSites finds all project-internal call sites in a type-checked package.
// Only includes calls to functions within the same project module.
func ExtractCallSites(fset *token.FileSet, pkg *packages.Package, projectModule string) []CallSite {
	if pkg.TypesInfo == nil || pkg.Types == nil || len(pkg.Syntax) == 0 {
		return nil
	}

	pkgPath := pkg.Types.Path()
	relCallerPkg := stripModulePrefix(pkgPath, projectModule)

	var sites []CallSite

	for _, file := range pkg.Syntax {
		callers := buildFuncRanges(file, relCallerPkg)

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			ident := callFuncIdent(call.Fun)
			if ident == nil {
				return true
			}

			obj := pkg.TypesInfo.Uses[ident]
			if obj == nil {
				return true
			}

			// Only track function calls, not type conversions or built-ins.
			fn, isFn := obj.(*types.Func)
			if !isFn {
				return true
			}

			calleePkg := fn.Pkg()
			if calleePkg == nil {
				return true // built-in
			}

			calleePath := calleePkg.Path()
			if calleePath != projectModule && !strings.HasPrefix(calleePath, projectModule+"/") {
				return true // stdlib or external
			}

			relCalleePkg := stripModulePrefix(calleePath, projectModule)
			calleeFQN := relCalleePkg + "." + fn.Name()

			callerFQN := findEnclosingFunc(callers, call.Pos())
			if callerFQN == "" {
				return true
			}

			sig, _ := fn.Type().(*types.Signature)
			isMethod := sig != nil && sig.Recv() != nil

			pos := fset.Position(call.Pos())
			sites = append(sites, CallSite{
				CallerFunc:  callerFQN,
				CalleeName:  calleeFQN,
				Line:        pos.Line,
				IsMethod:    isMethod,
				CrossModule: fqnModule(callerFQN) != fqnModule(calleeFQN),
			})

			return true
		})
	}

	return sites
}

// ResolveCallGraph populates Calls and CalledBy on symbols in extractions using allSites.
func ResolveCallGraph(extractions []*ModuleExtraction, allSites []CallSite) {
	// Build lookup: "internal/codemap.ParseGoFile" → *Symbol
	lookup := make(map[string]*Symbol)
	for _, e := range extractions {
		for fi := range e.Files {
			for si := range e.Files[fi].Symbols {
				sym := &e.Files[fi].Symbols[si]
				if sym.Kind != "function" && sym.Kind != "method" {
					continue
				}
				key := e.Module + "." + sym.Name
				// Last writer wins on name collision (different receivers, same method name).
				lookup[key] = sym
			}
		}
	}

	// Aggregate into sets for deduplication.
	// Only include a call site when BOTH caller and callee are in the symbol table,
	// so calls and called_by counts remain equal (bidirectional consistency).
	callsMap := make(map[string]map[string]bool)
	calledByMap := make(map[string]map[string]bool)

	for _, site := range allSites {
		if _, callerOK := lookup[site.CallerFunc]; !callerOK {
			continue
		}
		if _, calleeOK := lookup[site.CalleeName]; !calleeOK {
			continue
		}

		if callsMap[site.CallerFunc] == nil {
			callsMap[site.CallerFunc] = make(map[string]bool)
		}
		callsMap[site.CallerFunc][site.CalleeName] = true

		if calledByMap[site.CalleeName] == nil {
			calledByMap[site.CalleeName] = make(map[string]bool)
		}
		calledByMap[site.CalleeName][site.CallerFunc] = true
	}

	for fqn, sym := range lookup {
		if calls := callsMap[fqn]; len(calls) > 0 {
			sym.Calls = sortedStringSet(calls)
		}
		if calledBy := calledByMap[fqn]; len(calledBy) > 0 {
			sym.CalledBy = sortedStringSet(calledBy)
		}
	}
}

// funcRange is a position span for a function or method declaration.
type funcRange struct {
	start token.Pos
	end   token.Pos
	name  string // relative FQN: "internal/codemap.ExtractModule"
}

// buildFuncRanges returns position spans for all function/method declarations in file.
func buildFuncRanges(file *ast.File, relPkg string) []funcRange {
	var ranges []funcRange
	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		ranges = append(ranges, funcRange{
			start: fd.Pos(),
			end:   fd.End(),
			name:  relPkg + "." + fd.Name.Name,
		})
	}
	return ranges
}

// findEnclosingFunc returns the FQN of the innermost function containing pos, or "".
func findEnclosingFunc(ranges []funcRange, pos token.Pos) string {
	for _, r := range ranges {
		if pos >= r.start && pos <= r.end {
			return r.name
		}
	}
	return ""
}

// callFuncIdent extracts the function identifier from a call expression's Fun field.
func callFuncIdent(expr ast.Expr) *ast.Ident {
	switch e := expr.(type) {
	case *ast.Ident:
		return e
	case *ast.SelectorExpr:
		return e.Sel
	default:
		return nil
	}
}

// stripModulePrefix removes the project module prefix from a full package path.
// "github.com/Bucket-Chemist/goYoke/internal/codemap" → "internal/codemap"
func stripModulePrefix(pkgPath, projectModule string) string {
	if pkgPath == projectModule {
		return ""
	}
	if after, ok := strings.CutPrefix(pkgPath, projectModule+"/"); ok {
		return after
	}
	return pkgPath
}

// fqnModule extracts the module portion of a fully qualified name.
// "internal/codemap.ParseGoFile" → "internal/codemap"
func fqnModule(fqn string) string {
	if idx := strings.LastIndex(fqn, "."); idx > 0 {
		return fqn[:idx]
	}
	return ""
}

// sortedStringSet converts a string set map to a sorted slice.
func sortedStringSet(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
