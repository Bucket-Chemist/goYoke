package codemap

import (
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
)

// ParseGoFile parses a Go source file and returns the AST plus any parse errors.
// It always returns a (possibly partial) AST — parse errors are non-fatal.
// Only returns a non-nil error for truly fatal failures (unreadable file when source is nil).
func ParseGoFile(fset *token.FileSet, path string, source []byte) (*ast.File, []ParseError, error) {
	f, err := parser.ParseFile(fset, path, source, parser.ParseComments)
	if err == nil {
		return f, nil, nil
	}

	errList, ok := err.(scanner.ErrorList)
	if !ok {
		// Fatal: not a scan error (e.g., file unreadable when source is nil)
		return nil, nil, err
	}

	parseErrors := make([]ParseError, 0, len(errList))
	for _, e := range errList {
		parseErrors = append(parseErrors, ParseError{
			Line:    e.Pos.Line,
			Column:  e.Pos.Column,
			Message: e.Msg,
		})
	}
	// f may be a partial AST — still usable for symbol extraction
	return f, parseErrors, nil
}
